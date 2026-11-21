package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/audit"
	"relayradar/internal/store"
)

// alertMatches applies the price thresholds (other kinds pass through).
func alertMatches(sub store.Subscription, c app.Change) bool {
	if c.Kind != "price_change" {
		return true
	}
	if sub.MinDeltaPct > 0 && math.Abs(c.DeltaPct) < sub.MinDeltaPct {
		return false
	}
	if sub.MaxPriceCNY > 0 && (c.PriceCNY <= 0 || c.PriceCNY > sub.MaxPriceCNY) {
		return false
	}
	return true
}

type alertHub struct {
	client *http.Client
	srv    *Server
	once   sync.Once
}

var providerKeys = func() map[string]bool {
	out := map[string]bool{}
	for _, p := range modelProviders {
		out[p.Key] = true
	}
	return out
}()

func validateProviders(list []string) ([]string, error) {
	clean := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, p := range list {
		if !providerKeys[p] {
			return nil, fmt.Errorf("unknown provider %q", p)
		}
		if !seen[p] {
			clean = append(clean, p)
			seen[p] = true
		}
	}
	return clean, nil
}

var alertKinds = map[string]bool{
	"price_change": true, "availability_drop": true, "purity_alert": true,
	"score_drop": true, "new_station": true, "provider_incident": true, "station_down": true, "station_recovered": true,
}

func newAlertHub(s *Server) *alertHub {
	dialer := &net.Dialer{
		Timeout: 6 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("blocked: unresolvable %q", address)
			}
			if !s.cfg.AllowLocalTargets && audit.IsBlockedIP(ip) {
				return fmt.Errorf("blocked: %s is not public", ip)
			}
			return nil
		},
	}
	return &alertHub{
		srv: s,
		client: &http.Client{
			Timeout: 8 * time.Second,
			Transport: &http.Transport{
				DialContext:       dialer.DialContext,
				Proxy:             nil,
				DisableKeepAlives: true,
				MaxIdleConns:      1,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func validateThresholds(minDeltaPct, maxPriceCNY float64) error {
	if minDeltaPct < 0 || minDeltaPct > 100 {
		return fmt.Errorf("minDeltaPct must be between 0 and 100")
	}
	if maxPriceCNY < 0 || maxPriceCNY > 1_000_000 {
		return fmt.Errorf("maxPriceCny must be between 0 and 1000000")
	}
	return nil
}

func (h *alertHub) add(ctx context.Context, channel, rawURL, email string, kinds, stations, providers []string, minDeltaPct, maxPriceCNY float64) (*store.Subscription, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = "webhook"
	}
	rawURL = strings.TrimSpace(rawURL)
	email = strings.TrimSpace(email)

	switch channel {
	case "webhook":
		if len(rawURL) > 500 {
			return nil, fmt.Errorf("webhook url is too long")
		}
		u, err := url.Parse(rawURL)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("invalid webhook url")
		}
		if u.Scheme != "https" && !(h.srv.cfg.AllowLocalTargets && u.Scheme == "http") {
			return nil, fmt.Errorf("webhook must use https")
		}
	case "email":
		if !h.srv.cfg.SMTPConfigured() {
			return nil, fmt.Errorf("email channel is not configured on this deployment")
		}
		if len(email) > 200 || !strings.Contains(email, "@") || strings.HasPrefix(email, "@") {
			return nil, fmt.Errorf("invalid email address")
		}
	default:
		return nil, fmt.Errorf("unknown channel")
	}

	if len(kinds) == 0 {
		kinds = []string{"price_change", "availability_drop", "purity_alert", "score_drop", "new_station"}
	}
	clean, err := validateKinds(kinds)
	if err != nil {
		return nil, err
	}
	cleanProviders, err := validateProviders(providers)
	if err != nil {
		return nil, err
	}
	if err := validateThresholds(minDeltaPct, maxPriceCNY); err != nil {
		return nil, err
	}

	all, err := h.srv.svc.Repo().ListSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	if len(all) >= 200 {
		return nil, fmt.Errorf("subscription limit reached")
	}

	now := time.Now().UTC()
	sub := &store.Subscription{
		ID: "sub-" + randToken(4), Channel: channel, URL: rawURL, Email: email,
		Secret: randToken(16), Kinds: clean, Stations: stations, Providers: cleanProviders,
		MinDeltaPct: minDeltaPct, MaxPriceCNY: maxPriceCNY,
		Enabled: true, CreatedAt: now, LastSentAt: now,
	}
	if err := h.srv.svc.Repo().CreateSubscription(ctx, *sub); err != nil {
		return nil, err
	}
	h.once.Do(func() { go h.loop() })
	return sub, nil
}

func (h *alertHub) loop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		h.dispatch()
	}
}

func (h *alertHub) dispatch() {
	ctx := context.Background()
	d, err := h.srv.svc.Dataset(ctx)
	if err != nil {
		return
	}
	subs, err := h.srv.svc.Repo().ListSubscriptions(ctx)
	if err != nil {
		return
	}

	for i := range subs {
		sub := &subs[i]
		if !sub.Enabled {
			continue
		}
		if sub.NextAttemptAt != nil && sub.NextAttemptAt.After(time.Now()) {
			continue
		}
		items := make([]json.RawMessage, 0, 10)
		latest := sub.LastSentAt
		for _, c := range d.Changes {
			if !c.At.After(sub.LastSentAt) {
				continue
			}
			if !contains(sub.Kinds, c.Kind) || (len(sub.Stations) > 0 && !contains(sub.Stations, c.StationSlug)) {
				continue
			}
			if !alertMatches(*sub, c) {
				continue
			}
			b, _ := json.Marshal(c)
			items = append(items, b)
			if c.At.After(latest) {
				latest = c.At
			}
		}
		if len(items) == 0 {
			continue
		}
		if len(items) > 20 {
			items = items[len(items)-20:]
		}
		payload := map[string]any{
			"event": "change", "subscriptionId": sub.ID, "sentAt": time.Now().UTC(),
			"items": json.RawMessage("[" + joinRaw(items) + "]"),
		}
		payloadJSON, _ := json.Marshal(payload)
		start := time.Now()
		status, resp, err := h.deliver(sub, "change", payload)
		h.recordDelivery(ctx, sub, "change", string(payloadJSON), status, resp, err, time.Since(start))
		if err != nil {
			attempts := sub.Attempts + 1
			next := time.Now().Add(backoff(attempts))
			if e := h.srv.svc.Repo().SetDeliveryState(ctx, sub.ID, attempts, err.Error(), &next, nil); e != nil {
				h.srv.log.Warn("record delivery failure", "sub", sub.ID, "err", e)
			}
			if attempts == 3 {
				h.notifyFailure(ctx, sub, err)
			}
			h.srv.log.Warn("webhook failed", "sub", sub.ID, "attempt", attempts, "err", err)
			continue
		}
		sent := latest
		if err := h.srv.svc.Repo().SetDeliveryState(ctx, sub.ID, 0, "", nil, &sent); err != nil {
			h.srv.log.Warn("record delivery success", "sub", sub.ID, "err", err)
		}
	}
}

// backoff returns 1m, 5m, 15m, then 60m between retries.
func backoff(attempts int) time.Duration {
	switch {
	case attempts <= 1:
		return time.Minute
	case attempts == 2:
		return 5 * time.Minute
	case attempts == 3:
		return 15 * time.Minute
	default:
		return time.Hour
	}
}

// truncate shortens a response snippet for storage.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func (h *alertHub) deliver(sub *store.Subscription, event string, payload map[string]any) (int, string, error) {
	if sub.Channel == "email" {
		return h.sendEmail(sub, event, payload)
	}
	return h.send(sub, event, payload)
}

func (h *alertHub) notifyStationDown(ctx context.Context, slug string, latency int) {
	subs, err := h.srv.svc.Repo().ListSubscriptions(ctx)
	if err != nil {
		return
	}
	payload := map[string]any{
		"event": "station_down", "stationSlug": slug, "latencyMs": latency, "sentAt": time.Now().UTC(),
	}
	for i := range subs {
		sub := &subs[i]
		if !sub.Enabled || !containsString(sub.Kinds, "station_down") {
			continue
		}
		if len(sub.Stations) > 0 && !containsString(sub.Stations, slug) {
			continue
		}
		start := time.Now()
		st, resp, derr := h.deliver(sub, "station_down", payload)
		payloadJSON, _ := json.Marshal(payload)
		h.recordDelivery(ctx, sub, "station_down", string(payloadJSON), st, resp, derr, time.Since(start))
	}
}

func (h *alertHub) notifyStationRecovered(ctx context.Context, slug string, latency int) {
	subs, err := h.srv.svc.Repo().ListSubscriptions(ctx)
	if err != nil {
		return
	}
	payload := map[string]any{
		"event": "station_recovered", "stationSlug": slug, "latencyMs": latency, "sentAt": time.Now().UTC(),
	}
	for i := range subs {
		sub := &subs[i]
		if !sub.Enabled || !containsString(sub.Kinds, "station_recovered") {
			continue
		}
		if len(sub.Stations) > 0 && !containsString(sub.Stations, slug) {
			continue
		}
		start := time.Now()
		st, resp, derr := h.deliver(sub, "station_recovered", payload)
		payloadJSON, _ := json.Marshal(payload)
		h.recordDelivery(ctx, sub, "station_recovered", string(payloadJSON), st, resp, derr, time.Since(start))
	}
}

// notifyProviderIncident fans a provider status change out to subscriptions
// that opted into the provider_incident kind.
func (h *alertHub) notifyProviderIncident(ctx context.Context, provider, status, detail string) {
	subs, err := h.srv.svc.Repo().ListSubscriptions(ctx)
	if err != nil {
		return
	}
	payload := map[string]any{
		"event": "provider_incident", "provider": provider, "status": status,
		"detail": detail, "sentAt": time.Now().UTC(),
	}
	for i := range subs {
		sub := &subs[i]
		if !sub.Enabled || !containsString(sub.Kinds, "provider_incident") {
			continue
		}
		if len(sub.Providers) > 0 && !containsString(sub.Providers, provider) {
			continue
		}
		start := time.Now()
		st, resp, derr := h.deliver(sub, "provider_incident", payload)
		payloadJSON, _ := json.Marshal(payload)
		h.recordDelivery(ctx, sub, "provider_incident", string(payloadJSON), st, resp, derr, time.Since(start))
	}
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// notifyFailure emails the subscriber (or the ops address for webhooks) once a
// subscription has failed three times in a row.
func (h *alertHub) notifyFailure(ctx context.Context, sub *store.Subscription, cause error) {
	cfg := h.srv.cfg
	if !cfg.SMTPConfigured() {
		return
	}
	to := sub.Email
	if sub.Channel != "email" {
		to = cfg.AlertFailureEmail
	}
	if to == "" {
		return
	}
	link := strings.TrimRight(cfg.PublicBaseURL, "/") + "/alerts?id=" + sub.ID + "&token=" + sub.Secret
	body := "RelayRadar 告警投递连续失败 3 次，已暂停自动重试前的最后阶段。\n\n" +
		"通道：" + sub.Channel + "\n" +
		"目标：" + sub.URL + "\n" +
		"最近错误：" + cause.Error() + "\n" +
		"管理订阅：" + link + "\n\n" +
		"请检查接收端是否可达、是否返回 2xx；修复后可在管理页手动重放最近一次投递。"
	if err := h.srv.sendAuthMail(to, "RelayRadar 告警投递连续失败", body); err != nil {
		h.srv.log.Warn("send failure notice failed", "sub", sub.ID, "err", err)
	}
}

func (h *alertHub) recordDelivery(ctx context.Context, sub *store.Subscription, event, payloadJSON string, status int, response string, err error, duration time.Duration) {
	_, _ = h.srv.svc.Repo().RecordDelivery(ctx, store.Delivery{
		SubscriptionID: sub.ID,
		Event:          event,
		Status:         status,
		OK:             err == nil,
		DurationMS:     int(duration.Milliseconds()),
		Response:       truncate(response, 500),
		Payload:        payloadJSON,
	})
}

func (h *alertHub) sendEmail(sub *store.Subscription, event string, payload map[string]any) (int, string, error) {
	cfg := h.srv.cfg
	if !cfg.SMTPConfigured() {
		return 0, "smtp is not configured", fmt.Errorf("smtp is not configured")
	}

	type item struct {
		Kind        string    `json:"kind"`
		Title       string    `json:"title"`
		Detail      string    `json:"detail"`
		StationName string    `json:"stationName"`
		At          time.Time `json:"at"`
	}
	var items []item
	if raw, err := json.Marshal(payload["items"]); err == nil {
		_ = json.Unmarshal(raw, &items)
	}

	subject := "RelayRadar 变更提醒"
	if event == "ping" {
		subject = "RelayRadar 测试事件"
	}
	var body strings.Builder
	body.WriteString("RelayRadar 告警\r\n\r\n")
	for _, it := range items {
		fmt.Fprintf(&body, "• [%s] %s\r\n  %s\r\n  %s  %s\r\n\r\n",
			it.Kind, it.Title, it.Detail, it.StationName, it.At.Format("2006-01-02 15:04"))
	}
	fmt.Fprintf(&body, "管理订阅：%s/alerts?id=%s&token=%s\r\n",
		strings.TrimRight(cfg.PublicBaseURL, "/"), sub.ID, sub.Secret)

	msg := "From: RelayRadar <" + cfg.SMTPFrom + ">\r\n" +
		"To: " + sub.Email + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" + body.String()

	host, _, err := net.SplitHostPort(cfg.SMTPAddr)
	if err != nil {
		host = cfg.SMTPAddr
	}
	var auth smtp.Auth
	if cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, host)
	}
	if err := smtp.SendMail(cfg.SMTPAddr, auth, cfg.SMTPFrom, []string{sub.Email}, []byte(msg)); err != nil {
		return 0, err.Error(), err
	}
	return 0, "smtp accepted", nil
}

func (h *alertHub) send(sub *store.Subscription, event string, payload map[string]any) (int, string, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err.Error(), err
	}
	mac := hmac.New(sha256.New, []byte(sub.Secret))
	mac.Write(body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RelayRadar-Webhook/1.0")
	req.Header.Set("X-RelayRadar-Event", event)
	req.Header.Set("X-RelayRadar-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := h.client.Do(req)
	if err != nil {
		return 0, err.Error(), err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(respBody), fmt.Errorf("receiver returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, string(respBody), nil
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Channel     string   `json:"channel"`
		URL         string   `json:"url"`
		Email       string   `json:"email"`
		Kinds       []string `json:"kinds"`
		Stations    []string `json:"stations"`
		Providers   []string `json:"providers"`
		MinDeltaPct float64  `json:"minDeltaPct"`
		MaxPriceCNY float64  `json:"maxPriceCny"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Stations) > 0 {
		d, err := s.svc.Dataset(r.Context())
		if err != nil {
			s.fail(w, http.StatusInternalServerError, err)
			return
		}
		for _, slug := range body.Stations {
			if _, ok := d.BySlug[slug]; !ok {
				writeError(w, http.StatusBadRequest, "unknown station "+slug)
				return
			}
		}
	}
	sub, err := s.alerts.add(r.Context(), body.Channel, body.URL, body.Email, body.Kinds, body.Stations, body.Providers, body.MinDeltaPct, body.MaxPriceCNY)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": sub.ID, "channel": sub.Channel, "url": sub.URL, "email": sub.Email,
		"secret": sub.Secret, "kinds": sub.Kinds, "stations": sub.Stations, "providers": sub.Providers,
		"minDeltaPct": sub.MinDeltaPct, "maxPriceCny": sub.MaxPriceCNY,
	})
}

func (s *Server) handleSubPing(w http.ResponseWriter, r *http.Request) {
	sub := s.alerts.get(r.Context(), r.PathValue("id"), r.Header.Get("X-RelayRadar-Token"))
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	payload := map[string]any{
		"event": "ping", "subscriptionId": sub.ID, "sentAt": time.Now().UTC(),
		"items": []any{map[string]any{
			"id": "ping", "kind": "new_station", "severity": "info",
			"title": "RelayRadar 测试事件", "detail": "如果你收到这条消息，说明告警通道配置正确。",
			"stationSlug": "test", "stationName": "RelayRadar", "at": time.Now().UTC(),
		}},
	}
	before := time.Now()
	payloadJSON, _ := json.Marshal(payload)
	status, resp, err := s.alerts.deliver(sub, "ping", payload)
	s.alerts.recordDelivery(r.Context(), sub, "ping", string(payloadJSON), status, resp, err, time.Since(before))
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "latencyMs": time.Since(before).Milliseconds(), "status": status})
}

func (s *Server) handleSubDeliveries(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-RelayRadar-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	sub := s.alerts.get(r.Context(), r.PathValue("id"), token)
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	items, err := s.svc.Repo().ListDeliveries(r.Context(), sub.ID, 50)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleSubReplay(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-RelayRadar-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	sub := s.alerts.get(r.Context(), r.PathValue("id"), token)
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	id, err := parseInt64(r.PathValue("logId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid delivery id")
		return
	}
	d, err := s.svc.Repo().GetDelivery(r.Context(), id)
	if err != nil || d.SubscriptionID != sub.ID {
		writeError(w, http.StatusNotFound, "delivery not found")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(d.Payload), &payload); err != nil || len(payload) == 0 {
		writeError(w, http.StatusBadRequest, "stored payload is not replayable")
		return
	}
	start := time.Now()
	status, resp, derr := s.alerts.deliver(sub, d.Event, payload)
	s.alerts.recordDelivery(r.Context(), sub, d.Event, d.Payload, status, resp, derr, time.Since(start))
	if derr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "status": status, "response": truncate(resp, 300), "error": derr.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": status, "latencyMs": time.Since(start).Milliseconds()})
}

func (s *Server) handleSubGet(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-RelayRadar-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	sub := s.alerts.get(r.Context(), r.PathValue("id"), token)
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	writeJSON(w, http.StatusOK, publicSub(*sub))
}

func (s *Server) handleSubUpdate(w http.ResponseWriter, r *http.Request) {
	sub := s.alerts.get(r.Context(), r.PathValue("id"), r.Header.Get("X-RelayRadar-Token"))
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	var body struct {
		Kinds       []string `json:"kinds"`
		Stations    []string `json:"stations"`
		Providers   []string `json:"providers"`
		Enabled     *bool    `json:"enabled"`
		MinDeltaPct *float64 `json:"minDeltaPct"`
		MaxPriceCNY *float64 `json:"maxPriceCny"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Kinds != nil {
		clean, err := validateKinds(body.Kinds)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sub.Kinds = clean
	}
	if body.Stations != nil {
		d, err := s.svc.Dataset(r.Context())
		if err != nil {
			s.fail(w, http.StatusInternalServerError, err)
			return
		}
		for _, slug := range body.Stations {
			if _, ok := d.BySlug[slug]; !ok {
				writeError(w, http.StatusBadRequest, "unknown station "+slug)
				return
			}
		}
		sub.Stations = body.Stations
	}
	if body.Enabled != nil {
		sub.Enabled = *body.Enabled
	}
	if body.MinDeltaPct != nil {
		if err := validateThresholds(*body.MinDeltaPct, sub.MaxPriceCNY); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sub.MinDeltaPct = *body.MinDeltaPct
	}
	if body.MaxPriceCNY != nil {
		if err := validateThresholds(sub.MinDeltaPct, *body.MaxPriceCNY); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sub.MaxPriceCNY = *body.MaxPriceCNY
	}
	if body.Providers != nil {
		clean, err := validateProviders(body.Providers)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sub.Providers = clean
	}
	if err := s.svc.Repo().UpdateSubscription(r.Context(), *sub); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, publicSub(*sub))
}

func (s *Server) handleSubDelete(w http.ResponseWriter, r *http.Request) {
	sub := s.alerts.get(r.Context(), r.PathValue("id"), r.Header.Get("X-RelayRadar-Token"))
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	if err := s.svc.Repo().DeleteSubscription(r.Context(), sub.ID); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *alertHub) get(ctx context.Context, id, token string) *store.Subscription {
	sub, err := h.srv.svc.Repo().GetSubscription(ctx, id)
	if err != nil {
		return nil
	}
	if !hmac.Equal([]byte(sub.Secret), []byte(token)) {
		return nil
	}
	return &sub
}

func publicSub(sub store.Subscription) map[string]any {
	return map[string]any{
		"id": sub.ID, "channel": sub.Channel, "url": sub.URL, "email": sub.Email,
		"kinds": sub.Kinds, "stations": sub.Stations, "providers": sub.Providers,
		"minDeltaPct": sub.MinDeltaPct, "maxPriceCny": sub.MaxPriceCNY,
		"enabled": sub.Enabled, "attempts": sub.Attempts, "lastError": sub.LastError,
		"nextAttemptAt": sub.NextAttemptAt, "createdAt": sub.CreatedAt, "lastSentAt": sub.LastSentAt,
	}
}

func validateKinds(kinds []string) ([]string, error) {
	if len(kinds) == 0 {
		return nil, fmt.Errorf("pick at least one event kind")
	}
	clean := make([]string, 0, len(kinds))
	for _, k := range kinds {
		if !alertKinds[k] {
			return nil, fmt.Errorf("unknown kind %q", k)
		}
		clean = append(clean, k)
	}
	return clean, nil
}

func randToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func joinRaw(items []json.RawMessage) string {
	parts := make([]string, 0, len(items))
	for _, b := range items {
		parts = append(parts, string(b))
	}
	return strings.Join(parts, ",")
}

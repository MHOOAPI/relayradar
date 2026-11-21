package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/audit"
	"relayradar/internal/config"
	"relayradar/internal/favicon"
	"relayradar/internal/scoring"
	"relayradar/internal/store"
)

type Server struct {
	svc            *app.Service
	prober         *audit.Prober
	favicons       *favicon.Fetcher
	cfg            config.Config
	log            *slog.Logger
	rl             *limiter
	authRL         *burstLimiter
	freeKeyRL      *burstLimiter
	claimRL        *burstLimiter
	verifyRL       *burstLimiter
	metrics        *httpMetrics
	captchaMu      sync.Mutex
	captcha        map[string]captchaItem
	captchaKey     []byte
	startedAt      time.Time
	modelStatusMu  sync.Mutex
	modelStatus    []modelStatusItem
	modelStatusAt  time.Time
	providerPrevMu sync.Mutex
	providerLast   map[string]string
	reachMu        sync.Mutex
	reachDown      map[string]int
	transMu        sync.Mutex
	transCache     map[string]string
	alerts         *alertHub
}

func New(svc *app.Service, cfg config.Config, log *slog.Logger) *Server {
	s := &Server{
		svc: svc,
		prober: audit.New(audit.Options{
			AllowLocal: cfg.AllowLocalTargets,
			Timeout:    30 * time.Second,
		}),
		favicons:     favicon.NewWithProxy(cfg.AllowLocalTargets, cfg.FaviconProxyURL),
		cfg:          cfg,
		log:          log,
		rl:           newLimiter(4 * time.Second),
		authRL:       newBurstLimiter(10, time.Minute),
		freeKeyRL:    newBurstLimiter(3, 10*time.Minute),
		claimRL:      newBurstLimiter(20, 10*time.Minute),
		verifyRL:     newBurstLimiter(10, 10*time.Minute),
		metrics:      newHTTPMetrics(),
		startedAt:    time.Now(),
		providerLast: map[string]string{},
		reachDown:    map[string]int{},
		transCache:   map[string]string{},
	}
	s.initCaptcha()
	s.alerts = newAlertHub(s)
	return s
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir(s.cfg.IconDir))))
	mux.HandleFunc("GET /healthz", s.handleHealth)
	// Pages that were folded into others keep their old URLs alive.
	mux.HandleFunc("GET /sponsor", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/faq", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /leaderboards/window", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/leaderboards", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/v1/model-status", s.handleModelStatus)
	mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	mux.HandleFunc("GET /api/v1/incidents", s.handleIncidents)
	mux.HandleFunc("GET /status.rss", s.handleStatusRSS)
	mux.HandleFunc("GET /api/v1/admin/incidents", s.handleAdminIncidents)
	mux.HandleFunc("POST /api/v1/admin/incidents", s.handleAdminIncidentCreate)
	mux.HandleFunc("PATCH /api/v1/admin/incidents/{id}", s.handleAdminIncidentUpdate)
	mux.HandleFunc("DELETE /api/v1/admin/incidents/{id}", s.handleAdminIncidentDelete)
	mux.HandleFunc("GET /api/v1/meta", s.handleMeta)
	mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	mux.HandleFunc("GET /api/v1/stations", s.handleStations)
	mux.HandleFunc("GET /api/v1/stations/{slug}", s.handleStation)
	mux.HandleFunc("GET /api/v1/reachability.csv", s.handleAllReachabilityCSV)
	mux.HandleFunc("GET /api/v1/stations/{slug}/reachability", s.handleStationReachability)
	mux.HandleFunc("GET /api/v1/stations/{slug}/reviews", s.handleReviews)
	mux.HandleFunc("GET /api/v1/stations/{slug}/history", s.handleStationHistory)
	mux.HandleFunc("GET /api/v1/leaderboards/window", s.handleWindowLeaderboard)
	mux.HandleFunc("POST /api/v1/outbound", s.handleOutbound)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/verify", s.handleVerifyEmail)
	mux.HandleFunc("POST /api/v1/auth/forgot", s.handleForgotPassword)
	mux.HandleFunc("POST /api/v1/auth/reset", s.handleResetPassword)
	mux.HandleFunc("GET /api/v1/auth/me", s.handleMe)
	mux.HandleFunc("GET /api/v1/account", s.handleAccount)
	mux.HandleFunc("PATCH /api/v1/account/subscriptions/{id}", s.handleAccountSubscription)
	mux.HandleFunc("DELETE /api/v1/account/subscriptions/{id}", s.handleAccountSubscription)
	mux.HandleFunc("GET /api/v1/admin/summary", s.handleAdminSummary)
	mux.HandleFunc("GET /api/v1/admin/submissions", s.handleAdminSubmissions)
	mux.HandleFunc("PATCH /api/v1/admin/submissions/{id}", s.handleAdminSubmissionStatus)
	mux.HandleFunc("POST /api/v1/admin/stations/{slug}/icon", s.handleAdminStationIcon)
	mux.HandleFunc("DELETE /api/v1/admin/stations/{slug}/icon", s.handleAdminStationIconReset)
	mux.HandleFunc("DELETE /api/v1/admin/reviews/{id}", s.handleAdminDeleteReview)
	mux.HandleFunc("GET /api/v1/admin/subscriptions", s.handleAdminSubscriptions)
	mux.HandleFunc("GET /api/v1/admin/users", s.handleAdminUsers)
	mux.HandleFunc("PATCH /api/v1/admin/users/{id}", s.handleAdminUserRole)
	mux.HandleFunc("GET /api/v1/admin/reviews", s.handleAdminReviews)
	mux.HandleFunc("GET /api/v1/admin/audit", s.handleAdminAudit)
	mux.HandleFunc("POST /api/v1/stations/{slug}/reviews", s.handleCreateReview)
	mux.HandleFunc("GET /api/v1/models", s.handleModels)
	mux.HandleFunc("GET /api/v1/models/{id}", s.handleModel)
	mux.HandleFunc("POST /api/v1/audit", s.handleAudit)
	mux.HandleFunc("POST /api/v1/audit/models", s.handleAuditModels)
	mux.HandleFunc("GET /api/v1/audit/checks", s.handleAuditChecks)
	mux.HandleFunc("GET /api/v1/captcha", s.handleCaptchaIssue)
	mux.HandleFunc("POST /api/v1/captcha/{id}/verify", s.handleCaptchaVerify)
	mux.HandleFunc("GET /api/v1/audit/checks/en", s.handleAuditChecksEN)
	mux.HandleFunc("GET /api/v1/audit/checks/en/{key}", s.handleAuditCheckEN)
	mux.HandleFunc("GET /api/v1/audit/checks/{key}", s.handleAuditCheck)
	mux.HandleFunc("POST /api/v1/subscriptions", s.handleSubscribe)
	mux.HandleFunc("POST /api/v1/probes", s.handleProbeIngest)
	mux.HandleFunc("GET /api/v1/admin/probe-config", s.handleProbeConfig)
	mux.HandleFunc("POST /api/v1/admin/probe-config", s.handleProbeConfig)
	mux.HandleFunc("POST /api/v1/subscriptions/{id}/ping", s.handleSubPing)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}/deliveries", s.handleSubDeliveries)
	mux.HandleFunc("POST /api/v1/subscriptions/{id}/deliveries/{logId}/replay", s.handleSubReplay)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}", s.handleSubGet)
	mux.HandleFunc("PATCH /api/v1/subscriptions/{id}", s.handleSubUpdate)
	mux.HandleFunc("DELETE /api/v1/subscriptions/{id}", s.handleSubDelete)
	mux.HandleFunc("GET /api/v1/audit/{id}", s.handleAuditReport)
	mux.HandleFunc("POST /api/v1/submissions", s.handleSubmission)
	mux.HandleFunc("POST /api/v1/submissions/prefill", s.handlePrefill)
	mux.HandleFunc("POST /api/v1/claims", s.handleClaim)
	mux.HandleFunc("POST /api/v1/claims/{token}/verify", s.handleVerifyClaim)
	mux.HandleFunc("GET /api/v1/claims/{token}/station", s.handleClaimedStation)
	mux.HandleFunc("GET /api/v1/claims/{token}/edits", s.handleClaimEditsMine)
	mux.HandleFunc("PATCH /api/v1/claims/{token}/station", s.handleUpdateClaimedStation)
	mux.HandleFunc("GET /api/v1/admin/claim-edits", s.handleAdminClaimEdits)
	mux.HandleFunc("POST /api/v1/admin/claim-edits/{id}/approve", s.handleAdminClaimEditDecision)
	mux.HandleFunc("POST /api/v1/admin/claim-edits/{id}/reject", s.handleAdminClaimEditDecision)
	mux.HandleFunc("GET /api/v1/changes", s.handleChanges)
	mux.HandleFunc("GET /api/v1/guides", s.handleGuidesAPI)
	mux.HandleFunc("GET /api/v1/guides/{slug}", s.handleGuideAPI)
	mux.HandleFunc("GET /api/v1/guides/en", s.handleGuidesEN)
	mux.HandleFunc("GET /api/v1/guides/en/{slug}", s.handleGuideEN)
	mux.HandleFunc("GET /api/v1/favicon/{slug}", s.handleFavicon)
	mux.HandleFunc("POST /api/v1/free-keys", s.handleFreeKeyCreate)
	mux.HandleFunc("GET /api/v1/free-keys", s.handleFreeKeyList)
	mux.HandleFunc("POST /api/v1/free-keys/{id}/claim", s.handleFreeKeyClaim)
	mux.HandleFunc("GET /api/v1/admin/free-keys", s.handleAdminFreeKeys)
	mux.HandleFunc("DELETE /api/v1/admin/free-keys/{id}", s.handleAdminFreeKeyDelete)
	mux.HandleFunc("GET /llms.txt", s.handleLLMs)
	mux.HandleFunc("GET /llms-full.txt", s.handleLLMsFull)
	mux.HandleFunc("GET /md/station/{slug}", s.handleStationMD)
	mux.HandleFunc("GET /md/model/{id}", s.handleModelMD)
	mux.HandleFunc("GET /changes.xml", s.handleChangesRSS)
	mux.HandleFunc("GET /sitemap.xml", s.handleSitemap)
	mux.HandleFunc("GET /robots.txt", s.handleRobots)

	if s.cfg.WebDir != "" {
		mux.Handle("/", s.spaHandler(s.cfg.WebDir))
	}

	return s.recoverMW(s.securityMW(s.csrfMW(s.corsMW(s.logMW(s.metricsMW(mux))))))
}

func (s *Server) csrfMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if origin != "" && !s.sameOrigin(r, origin) {
				writeError(w, http.StatusForbidden, "cross-origin request blocked")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) sameOrigin(r *http.Request, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := u.Hostname()
	want := r.Host
	if h, _, err := net.SplitHostPort(want); err == nil {
		want = h
	}
	if host == want {
		return true
	}
	if pub, err := url.Parse(s.cfg.PublicBaseURL); err == nil && pub.Hostname() == host {
		return true
	}
	return false
}

func (s *Server) securityMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

// spaHandler lives in seo.go: it injects per-page meta before serving index.html.

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// transparentPNG is a 1×1 transparent pixel used as the favicon miss response.
var transparentPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

// handleFavicon proxies and caches a station's icon so the browser never talks
// to third-party hosts directly.
func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	st, ok := d.BySlug[r.PathValue("slug")]
	if !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	data, ctype, ok := s.favicons.Get(r.Context(), st.Domain)
	if !ok {
		// The UI draws its own glyph behind the image, so answer with a
		// transparent pixel instead of a 404: the icon simply stays invisible
		// and the directory does not fill the console with failed requests.
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transparentPNG)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	ctx := r.Context()
	counts, lastProbe, err := s.svc.Repo().Counts(ctx)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	stations := counts["stations"]
	groups := counts["groups"]
	if d, derr := s.svc.Dataset(ctx); derr == nil {
		stations = len(d.Stations)
		groups = 0
		for _, st := range d.Stations {
			groups += st.GroupsCount
		}
	}
	writeJSONCached(w, r, etagOf("meta", lastProbe.String(), strconv.Itoa(stations), strconv.Itoa(groups)), map[string]any{
		"stations":           stations,
		"groups":             groups,
		"models":             counts["models"],
		"probes":             counts["probes"],
		"dataUpdatedAt":      lastProbe,
		"demo":               s.cfg.SeedDemo,
		"auditEnabled":       s.cfg.AuditEnabled,
		"methodologyVersion": "2.0",
	})
}

func (s *Server) handleStations(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	ctx := r.Context()
	d, err := s.svc.Dataset(ctx)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	q := r.URL.Query()
	items := app.Filter(d.Stations, q.Get("q"), q.Get("service"), q.Get("tier"), q.Get("sort"))
	total := len(items)
	limit := intParam(q.Get("limit"), 200)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	writeJSONCached(w, r, etagOf("stations", d.TakenAt.String(), q.Get("q"), q.Get("service"), q.Get("tier"), q.Get("sort"), strconv.Itoa(limit), strconv.Itoa(total)), map[string]any{
		"items":         items,
		"total":         total,
		"dataUpdatedAt": d.TakenAt,
	})
}

func (s *Server) handleStation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	d, err := s.svc.Dataset(ctx)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	st, ok := d.BySlug[slug]
	if !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	id, err := s.svc.Repo().StationIDBySlug(ctx, slug)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	reviews, err := s.svc.Repo().ListReviews(ctx, id)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	outbound, _ := s.svc.Repo().OutboundCount(ctx, slug)
	detail := app.StationDetail{
		Station:     *st,
		Groups:      d.GroupsByStation[slug],
		Models:      d.PerStationModels[slug],
		Reviews:     reviews,
		History:     d.History[slug],
		Outbound30d: outbound,
		Business: map[string]any{
			"minTopupCny":  st.MinTopup,
			"payments":     st.Payments,
			"invoice":      st.Invoice,
			"refundPolicy": st.RefundPolicy,
			"listedAt":     st.ListedAt,
			"claimed":      st.Claimed,
		},
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleOutbound(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug string `json:"slug"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	if _, ok := d.BySlug[body.Slug]; !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	if err := s.svc.Repo().RecordOutbound(r.Context(), body.Slug); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	n, _ := s.svc.Repo().OutboundCount(r.Context(), body.Slug)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "count30d": n})
}

func (s *Server) handleWindowLeaderboard(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	days := 7
	if r.URL.Query().Get("days") == "30" {
		days = 30
	}
	type row struct {
		Slug    string  `json:"slug"`
		Name    string  `json:"name"`
		Average float64 `json:"average"`
		Current float64 `json:"current"`
		Delta   float64 `json:"delta"`
		Uptime  float64 `json:"uptime"`
		Days    int     `json:"days"`
	}
	rows := make([]row, 0, len(d.Stations))
	for _, st := range d.Stations {
		hist := d.History[st.Slug]
		if len(hist) == 0 {
			continue
		}
		n := days
		if len(hist) < n {
			n = len(hist)
		}
		sum, up := 0.0, 0.0
		for _, p := range hist[len(hist)-n:] {
			sum += p.Score
			up += p.Uptime
		}
		avg := sum / float64(n)
		rows = append(rows, row{
			Slug: st.Slug, Name: st.Name, Average: scoring.Round1(avg),
			Current: st.RankScore, Delta: scoring.Round1(avg - st.RankScore),
			Uptime: scoring.Round1(up / float64(n)), Days: n,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Average > rows[j].Average })
	writeJSON(w, http.StatusOK, map[string]any{"items": rows, "windowDays": days})
}

func (s *Server) handleStationHistory(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	slug := r.PathValue("slug")
	if _, ok := d.BySlug[slug]; !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":         d.History[slug],
		"dataUpdatedAt": d.TakenAt,
	})
}

func (s *Server) handleReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	id, err := s.svc.Repo().StationIDBySlug(ctx, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	reviews, err := s.svc.Repo().ListReviews(ctx, id)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reviews})
}

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	var body struct {
		Author string `json:"author"`
		Rating int    `json:"rating"`
		Body   string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Author = strings.TrimSpace(body.Author)
	body.Body = strings.TrimSpace(body.Body)
	if body.Author == "" {
		body.Author = "匿名开发者"
	}
	if len([]rune(body.Author)) > 40 {
		writeError(w, http.StatusBadRequest, "author is too long")
		return
	}
	if n := len([]rune(body.Body)); n < 5 || n > 600 {
		writeError(w, http.StatusBadRequest, "review body must be 5-600 characters")
		return
	}
	if body.Rating < 1 || body.Rating > 5 {
		body.Rating = 5
	}
	id, err := s.svc.Repo().StationIDBySlug(ctx, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	rv, err := s.svc.Repo().InsertReview(ctx, id, body.Author, body.Rating, body.Body)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, rv)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	items := make([]app.ModelView, 0, len(d.Models))
	for _, m := range d.Models {
		if q != "" && !strings.Contains(strings.ToLower(m.Name), q) &&
			!strings.Contains(strings.ToLower(m.ID), q) &&
			!strings.Contains(strings.ToLower(m.Vendor), q) {
			continue
		}
		items = append(items, m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) handleModel(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	id := r.PathValue("id")
	m, ok := d.ModelByID[id]
	if !ok {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"model":        m,
		"offers":       d.Offers[id],
		"priceChanges": d.PriceChanges[id],
	})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.AuditEnabled {
		writeError(w, http.StatusServiceUnavailable, "audit is disabled on this deployment")
		return
	}
	var body struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
		Model   string `json:"model"`
		Lang    string `json:"lang"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.requireCaptcha(w, r) {
		return
	}
	if !s.rl.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many probes, retry in a few seconds")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()

	prober := s.prober
	if body.Lang == "en" {
		prober = audit.New(audit.Options{AllowLocal: s.cfg.AllowLocalTargets, Timeout: 30 * time.Second, Lang: "en"})
	}
	report, err := prober.Inspect(ctx, body.BaseURL, body.APIKey, body.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	payload, _ := json.Marshal(report)
	if err := s.svc.Repo().InsertAudit(ctx, store.AuditReport{
		ID: report.ID, TargetURL: report.TargetURL, Model: report.Model,
		Passed: report.Passed, Payload: payload, CreatedAt: report.CreatedAt,
	}); err != nil {
		s.log.Warn("persist audit failed", "err", err)
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleAuditModels(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.AuditEnabled {
		writeError(w, http.StatusServiceUnavailable, "audit is disabled on this deployment")
		return
	}
	var body struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Discovery is a cheap GET; it deliberately does not consume the audit
	// limiter so a user can list models and immediately run a probe.
	items, err := s.prober.ListModels(r.Context(), body.BaseURL, body.APIKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func (s *Server) handleAuditReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.svc.Repo().GetAudit(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(rep.Payload)
}

func (s *Server) fail(w http.ResponseWriter, code int, err error) {
	s.log.Error("request failed", "err", err)
	writeError(w, code, "internal error")
}

func (s *Server) logMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/healthz") {
			s.log.Info("http", "method", r.Method, "path", r.URL.Path, "ms", time.Since(start).Milliseconds())
		}
	})
}

func (s *Server) recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "recover", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMW(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.CORSOrigins {
		allowed[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// setPublicCache marks read-only, recomputed-every-30s responses as cacheable.
func setPublicCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=30, stale-while-revalidate=60")
}

func intParam(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type limiter struct {
	mu     sync.Mutex
	last   map[string]time.Time
	window time.Duration
}

type burstLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newBurstLimiter(limit int, window time.Duration) *burstLimiter {
	return &burstLimiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

func (l *burstLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	if len(l.hits) > 2048 {
		for k, v := range l.hits {
			if len(v) == 0 {
				delete(l.hits, k)
			}
		}
	}
	return true
}

func newLimiter(window time.Duration) *limiter {
	return &limiter{last: map[string]time.Time{}, window: window}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if t, ok := l.last[key]; ok && now.Sub(t) < l.window {
		return false
	}
	l.last[key] = now
	if len(l.last) > 2048 {
		for k, t := range l.last {
			if now.Sub(t) > l.window {
				delete(l.last, k)
			}
		}
	}
	return true
}

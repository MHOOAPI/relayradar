// Package audit implements a bounded, SSRF-guarded probe of an
// OpenAI-compatible relay endpoint.
package audit

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"syscall"
	"time"
)

type Check struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"` // pass | warn | fail
	Detail string `json:"detail"`
}

type Report struct {
	ID               string    `json:"id"`
	CreatedAt        time.Time `json:"createdAt"`
	TargetURL        string    `json:"targetUrl"`
	Endpoint         string    `json:"endpoint"`
	Model            string    `json:"model"`
	Passed           bool      `json:"passed"`
	TTFTms           int       `json:"ttftMs"`
	DurationMs       int       `json:"durationMs"`
	PromptTokens     int       `json:"promptTokens"`
	CompletionTokens int       `json:"completionTokens"`
	CachedTokens     int       `json:"cachedTokens"`
	EchoModel        string    `json:"echoModel"`
	Checks           []Check   `json:"checks"`
}

type Options struct {
	AllowLocal bool
	Timeout    time.Duration
	Lang       string
}

type Prober struct {
	opts Options
}

func New(opts Options) *Prober {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	return &Prober{opts: opts}
}

// Inspect runs two identical completion requests: one to measure TTFT and
// protocol health, one to observe whether prompt cache is passed through.
func (p *Prober) Inspect(ctx context.Context, rawURL, apiKey, model string) (*Report, error) {
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("model is required")
	}
	en := p.opts.Lang == "en"
	tr := func(zh, eng string) string {
		if en {
			return eng
		}
		return zh
	}
	u, err := ValidateTarget(rawURL, p.opts.AllowLocal)
	if err != nil {
		return nil, err
	}
	apiKey = strings.TrimSpace(apiKey)

	ctx, cancel := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancel()

	client := p.client()
	nonce := randHex(4)
	prompt := fmt.Sprintf("RelayRadar probe %s. Reply with the single word READY and nothing else.", nonce)

	start := time.Now()
	first, err := p.call(ctx, client, u, apiKey, model, prompt, true)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(start)

	// Byte-identical second call: a cache-aware endpoint should report cached
	// input tokens on this one.
	second, _ := p.call(ctx, client, u, apiKey, model, prompt, false)

	r := &Report{
		ID:               "RR-" + strings.ToUpper(randHex(4)),
		CreatedAt:        time.Now().UTC(),
		TargetURL:        rawURL,
		Endpoint:         chatEndpoint(u),
		Model:            model,
		TTFTms:           first.ttftMS,
		DurationMs:       int(elapsed.Milliseconds()),
		PromptTokens:     first.promptTokens,
		CompletionTokens: first.completionTokens,
		EchoModel:        first.model,
	}
	if second != nil {
		r.CachedTokens = second.cachedTokens
	}

	r.Checks = append(r.Checks, Check{
		Key: "reachable", Label: tr("端点可达", "Endpoint reachable"),
		Status: statusFor(first.statusCode >= 200 && first.statusCode < 300),
		Detail: fmt.Sprintf("HTTP %d · %s", first.statusCode, first.endpoint),
	})
	r.Checks = append(r.Checks, Check{
		Key: "auth", Label: tr("鉴权通过", "Authentication accepted"), Status: authStatus(first.statusCode),
		Detail: authDetail(first.statusCode, en),
	})
	if first.errText != "" {
		r.Checks = append(r.Checks, Check{
			Key: "upstream_error", Label: tr("上游响应", "Upstream response"), Status: "fail", Detail: truncate(first.errText, 240),
		})
	}
	r.Checks = append(r.Checks, Check{
		Key: "stream", Label: tr("流式完整性", "Stream integrity"), Status: streamStatus(first), Detail: streamDetail(first, en),
	})
	r.Checks = append(r.Checks, Check{
		Key: "model_echo", Label: tr("模型回显", "Model echo"), Status: echoStatus(first.model, model),
		Detail: echoDetail(first.model, model, en),
	})
	r.Checks = append(r.Checks, Check{
		Key: "usage", Label: tr("用量上报", "Usage reported"), Status: statusFor(first.promptTokens > 0),
		Detail: usageDetail(first, en),
	})
	if second != nil {
		consistent := second.promptTokens == first.promptTokens && first.promptTokens > 0
		r.Checks = append(r.Checks, Check{
			Key: "token_consistency", Label: tr("分词一致性", "Tokenizer consistency"), Status: statusFor(consistent),
			Detail: fmt.Sprintf(tr("两次相同请求 prompt_tokens：%d / %d", "prompt_tokens across two identical calls: %d / %d"), first.promptTokens, second.promptTokens),
		})
	}

	cacheStatus, cacheDetail := "warn", tr("第二次相同请求未返回缓存命中字段：缓存可能被中转站吞掉，或上游不支持显式缓存", "No cache hit on the second identical request: the relay may swallow the cache, or upstream has no explicit cache")
	if r.CachedTokens > 0 {
		cacheStatus = "pass"
		cacheDetail = fmt.Sprintf(tr("第二次请求 cached_tokens=%d，缓存透传正常", "cached_tokens=%d on the second request; cache passthrough looks fine"), r.CachedTokens)
	}
	r.Checks = append(r.Checks, Check{
		Key: "cache", Label: tr("缓存透传", "Cache passthrough"), Status: cacheStatus, Detail: cacheDetail,
	})

	ttft := "fail"
	switch {
	case first.ttftMS > 0 && first.ttftMS < 1500:
		ttft = "pass"
	case first.ttftMS >= 1500 && first.ttftMS < 4000:
		ttft = "warn"
	}
	r.Checks = append(r.Checks, Check{
		Key: "ttft", Label: tr("首字延迟", "First token latency"), Status: ttft,
		Detail: fmt.Sprintf(tr("TTFT %d ms（<1500 通过 / <4000 观察）", "TTFT %d ms (<1500 pass / <4000 watch)"), first.ttftMS),
	})

	r.Passed = true
	for _, c := range r.Checks {
		if c.Status == "fail" {
			r.Passed = false
			break
		}
	}
	return r, nil
}

type callResult struct {
	statusCode       int
	endpoint         string
	ttftMS           int
	promptTokens     int
	completionTokens int
	cachedTokens     int
	model            string
	sawDelta         bool
	sawDone          bool
	errText          string
}

func (p *Prober) call(ctx context.Context, client *http.Client, u *url.URL, key, model, prompt string, measureTTFT bool) (*callResult, error) {
	res := &callResult{endpoint: chatEndpoint(u)}
	payload := map[string]any{
		"model":          model,
		"stream":         true,
		"messages":       []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":     64,
		"stream_options": map[string]bool{"include_usage": true},
	}
	if !measureTTFT {
		payload["max_tokens"] = 8
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, res.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	defer resp.Body.Close()
	res.statusCode = resp.StatusCode

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		res.errText = strings.TrimSpace(string(b))
		return res, nil
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 4<<20))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			res.sawDone = true
			break
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				PromptDetails    *struct {
					CachedTokens int `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Model != "" {
			res.model = chunk.Model
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" && !res.sawDelta {
			res.sawDelta = true
			if measureTTFT {
				res.ttftMS = int(time.Since(start).Milliseconds())
			}
		}
		if chunk.Usage != nil {
			res.promptTokens = chunk.Usage.PromptTokens
			res.completionTokens = chunk.Usage.CompletionTokens
			if chunk.Usage.PromptDetails != nil {
				res.cachedTokens = chunk.Usage.PromptDetails.CachedTokens
			}
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		res.errText = err.Error()
	}
	return res, nil
}

// ModelOption is one id returned by an OpenAI-compatible /models endpoint.
type ModelOption struct {
	ID      string `json:"id"`
	OwnedBy string `json:"ownedBy,omitempty"`
}

// ListModels discovers the model ids a relay exposes, using the same SSRF
// guard as the probe. It never follows redirects and caps the body size.
func (p *Prober) ListModels(ctx context.Context, rawURL, apiKey string) ([]ModelOption, error) {
	u, err := ValidateTarget(rawURL, p.opts.AllowLocal)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	endpoint := strings.TrimSuffix(chatEndpoint(u), "/chat/completions") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("models endpoint returned HTTP %d: %s", resp.StatusCode, truncate(strings.TrimSpace(string(b)), 200))
	}

	var payload struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid /models response: %w", err)
	}

	seen := map[string]bool{}
	out := make([]ModelOption, 0, len(payload.Data))
	for _, m := range payload.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, ModelOption{ID: id, OwnedBy: strings.TrimSpace(m.OwnedBy)})
	}
	if len(out) == 0 {
		return nil, errors.New("the endpoint returned no model ids")
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (p *Prober) client() *http.Client {
	dialer := &net.Dialer{
		Timeout:   6 * time.Second,
		KeepAlive: 0,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("blocked: unresolvable address %q", address)
			}
			if !p.opts.AllowLocal && isBlockedIP(ip) {
				return fmt.Errorf("blocked: %s is not a public address", ip)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: p.opts.Timeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			Proxy:                 nil, // never inherit HTTP(S)_PROXY: it bypasses the guard
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
			DisableKeepAlives:     true,
			MaxIdleConns:          1,
			ForceAttemptHTTP2:     false,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ValidateTarget enforces scheme, shape and (unless allowed) public addressing.
func ValidateTarget(raw string, allowLocal bool) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("baseUrl is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid baseUrl: %w", err)
	}
	if u.Scheme != "https" && !(allowLocal && u.Scheme == "http") {
		return nil, errors.New("only https:// endpoints are accepted")
	}
	if u.User != nil {
		return nil, errors.New("credentials in the URL are not allowed")
	}
	if u.Hostname() == "" {
		return nil, errors.New("baseUrl must include a host")
	}
	if u.Port() != "" && u.Port() != "443" && !allowLocal {
		return nil, errors.New("only the default https port is allowed")
	}
	if allowLocal {
		return u, nil
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("cannot resolve %s: %w", u.Hostname(), err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("cannot resolve %s", u.Hostname())
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("%s resolves to a non-public address", u.Hostname())
		}
	}
	return u, nil
}

var blockedNets = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.168.0.0/16",
		"198.18.0.0/15", "224.0.0.0/4", "240.0.0.0/4",
		"::1/128", "fc00::/7", "fe80::/10", "ff00::/8",
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}()

// IsBlockedIP reports whether ip is outside the public internet. The favicon
// fetcher reuses it so both outbound paths share one policy.
func IsBlockedIP(ip net.IP) bool { return isBlockedIP(ip) }

func isBlockedIP(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func chatEndpoint(u *url.URL) string {
	p := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(p, "/chat/completions"):
	case strings.HasSuffix(p, "/v1"):
		p += "/chat/completions"
	case p == "":
		p = "/v1/chat/completions"
	default:
		p += "/v1/chat/completions"
	}
	out := *u
	out.Path = p
	out.RawQuery = ""
	out.Fragment = ""
	return out.String()
}

func statusFor(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

func authStatus(code int) string {
	switch {
	case code == 200 || code == 201:
		return "pass"
	case code == 401 || code == 403:
		return "fail"
	default:
		return "warn"
	}
}

func authDetail(code int, en bool) string {
	switch {
	case code == 401 || code == 403:
		if en {
			return "Authentication rejected (invalid or unauthorised key)"
		}
		return "鉴权被拒绝（Key 无效或未授权）"
	case code == 200 || code == 201:
		if en {
			return "Bearer key accepted"
		}
		return "Bearer Key 被接受"
	default:
		if en {
			return fmt.Sprintf("Upstream returned HTTP %d; auth state unknown", code)
		}
		return fmt.Sprintf("上游返回 HTTP %d，鉴权状态未知", code)
	}
}

func streamStatus(r *callResult) string {
	if r.sawDelta && r.sawDone {
		return "pass"
	}
	if r.sawDelta {
		return "warn"
	}
	return "fail"
}

func streamDetail(r *callResult, en bool) string {
	switch {
	case r.sawDelta && r.sawDone:
		if en {
			return "Content deltas and the [DONE] terminator were received"
		}
		return "收到增量内容与 [DONE] 结束标记"
	case r.sawDelta:
		if en {
			return "Content deltas received, but the [DONE] terminator is missing"
		}
		return "收到增量内容，但缺少 [DONE] 结束标记"
	default:
		if en {
			return "No streaming content delta was received"
		}
		return "未收到任何流式增量内容"
	}
}

func echoStatus(echo, want string) string {
	if echo == "" {
		return "warn"
	}
	if echo == want {
		return "pass"
	}
	return "warn"
}

func echoDetail(echo, want string, en bool) string {
	switch {
	case echo == "":
		if en {
			return "Response did not echo a model name"
		}
		return "响应未回显模型名"
	case echo == want:
		if en {
			return "Response model matches the request: " + echo
		}
		return "响应模型名与请求一致：" + echo
	default:
		if en {
			return fmt.Sprintf("Requested %s, response echoed %s (possible substitution)", want, echo)
		}
		return fmt.Sprintf("请求 %s，响应回显 %s（可能存在模型替换）", want, echo)
	}
}

func usageDetail(r *callResult, en bool) string {
	if r.promptTokens > 0 {
		if en {
			return fmt.Sprintf("prompt_tokens=%d, completion_tokens=%d", r.promptTokens, r.completionTokens)
		}
		return fmt.Sprintf("prompt_tokens=%d，completion_tokens=%d", r.promptTokens, r.completionTokens)
	}
	if en {
		return "No usage field in the response; billing cannot be reconciled"
	}
	return "响应未包含 usage 字段：无法核对计费口径"
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

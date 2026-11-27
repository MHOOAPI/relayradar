// Package favicon fetches and caches station favicons with the same SSRF
// policy as the audit prober.
package favicon

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"relayradar/internal/audit"
)

// iconService mirrors a site's declared icon; used only when the station's own
// host is unreachable or blocked (some CDNs drop non-browser TLS clients).
const iconService = "https://icons.duckduckgo.com/ip3/%s.ico"

const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

type entry struct {
	data  []byte
	ctype string
	ok    bool
	at    time.Time
}

type Fetcher struct {
	allowLocal bool
	client     *http.Client

	mu       sync.Mutex
	cache    map[string]entry
	inflight map[string]bool
}

func New(allowLocal bool) *Fetcher { return NewWithProxy(allowLocal, "") }

// NewWithProxy optionally routes favicon fetches through an operator-provided
// proxy. The proxy is trusted and only used for icons, never for audit probes.
func NewWithProxy(allowLocal bool, proxyURL string) *Fetcher {
	var proxy *url.URL
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil && u.Host != "" {
			proxy = u
		}
	}

	dialer := &net.Dialer{Timeout: 4 * time.Second}
	if proxy == nil {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || (!allowLocal && audit.IsBlockedIP(ip)) {
				return fmt.Errorf("blocked address %s", address)
			}
			return nil
		}
	}

	f := &Fetcher{allowLocal: allowLocal, cache: map[string]entry{}, inflight: map[string]bool{}}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   true,
	}
	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}
	f.client = &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			if req.URL.Scheme != "https" && !(allowLocal && req.URL.Scheme == "http") {
				return fmt.Errorf("redirect to non-https")
			}
			return nil
		},
	}
	return f
}

// Cached returns a fresh cached result without touching the network. The fourth
// value reports whether the entry is still fresh; when false the caller should
// serve a placeholder and call Warm.
func (f *Fetcher) Cached(domain string) ([]byte, string, bool, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !domainRe.MatchString(domain) {
		return nil, "", false, true
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.cache[domain]
	if !ok {
		return nil, "", false, false
	}
	ttl := 24 * time.Hour
	if !e.ok {
		ttl = time.Hour
	}
	if time.Since(e.at) >= ttl {
		return nil, "", false, false
	}
	return e.data, e.ctype, e.ok, true
}

// Ensure fetches one favicon in the background and stores it in the cache. It is
// safe to call repeatedly; concurrent calls for one domain collapse.
func (f *Fetcher) Ensure(domain string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !domainRe.MatchString(domain) {
		return
	}
	f.mu.Lock()
	if f.inflight[domain] {
		f.mu.Unlock()
		return
	}
	f.inflight[domain] = true
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.inflight, domain)
		f.mu.Unlock()
	}()

	data, ctype, ok := f.fetch(context.Background(), domain)

	f.mu.Lock()
	f.cache[domain] = entry{data: data, ctype: ctype, ok: ok, at: time.Now()}
	f.mu.Unlock()
}

// Get returns a cached or freshly fetched favicon for domain.
func (f *Fetcher) Get(ctx context.Context, domain string) ([]byte, string, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if !domainRe.MatchString(domain) {
		return nil, "", false
	}

	f.mu.Lock()
	if e, ok := f.cache[domain]; ok {
		ttl := 24 * time.Hour
		if !e.ok {
			ttl = time.Hour
		}
		if time.Since(e.at) < ttl {
			f.mu.Unlock()
			return e.data, e.ctype, e.ok
		}
	}
	f.mu.Unlock()

	data, ctype, ok := f.fetch(ctx, domain)

	f.mu.Lock()
	f.cache[domain] = entry{data: data, ctype: ctype, ok: ok, at: time.Now()}
	f.mu.Unlock()
	return data, ctype, ok
}

func (f *Fetcher) fetch(ctx context.Context, domain string) ([]byte, string, bool) {
	// The station's own /favicon.ico is tried first. Many sites only declare
	// their icon in the HTML head (or move it to a CDN path), so a miss falls
	// back to parsing <link rel="icon"> from the homepage.
	if strings.HasSuffix(domain, ".invalid") || strings.HasPrefix(domain, "localhost") {
		return nil, "", false
	}
	if data, ctype, ok := f.get(ctx, "https://"+domain+"/favicon.ico"); ok {
		return data, ctype, true
	}
	if page, ok := f.getPage(ctx, "https://"+domain+"/"); ok {
		if iconURL := parseIconHref(page, "https://"+domain+"/"); iconURL != "" {
			if data, ctype, ok := f.get(ctx, iconURL); ok {
				return data, ctype, true
			}
		}
	}
	// Prefer real mirrors before falling back to a generated glyph: DuckDuckGo
	// first, then Google s2 (one of them usually survives regional blocking).
	if data, ctype, ok := f.get(ctx, fmt.Sprintf(iconService, domain)); ok {
		return data, ctype, true
	}
	return f.get(ctx, fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=64", domain))
}

var (
	linkTagRe  = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	relAttrRe  = regexp.MustCompile(`(?is)\brel\s*=\s*["']([^"']*)["']`)
	hrefAttrRe = regexp.MustCompile(`(?is)\bhref\s*=\s*["']([^"']*)["']`)
)

// parseIconHref finds the best icon declared by the page: a touch icon first
// (usually the largest), then any rel containing "icon".
func parseIconHref(page []byte, pageURL string) string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	var fallback string
	for _, tag := range linkTagRe.FindAll(page, 200) {
		rel := relAttrRe.FindSubmatch(tag)
		if rel == nil {
			continue
		}
		relValue := strings.ToLower(string(rel[1]))
		if !strings.Contains(relValue, "icon") {
			continue
		}
		href := hrefAttrRe.FindSubmatch(tag)
		if href == nil {
			continue
		}
		raw := strings.TrimSpace(string(href[1]))
		if raw == "" || strings.HasPrefix(raw, "data:") {
			continue
		}
		ref, err := url.Parse(raw)
		if err != nil {
			continue
		}
		abs := base.ResolveReference(ref)
		if abs.Scheme != "https" && abs.Scheme != "http" {
			continue
		}
		if strings.Contains(relValue, "apple-touch-icon") {
			return abs.String()
		}
		if fallback == "" {
			fallback = abs.String()
		}
	}
	return fallback
}

// getPage fetches an HTML page for icon discovery. Unlike get it accepts
// non-image responses and caps the body at 512 KiB.
func (f *Fetcher) getPage(ctx context.Context, raw string) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil || len(body) < 32 {
		return nil, false
	}
	ctype := strings.ToLower(resp.Header.Get("Content-Type"))
	if ctype != "" && !strings.Contains(ctype, "html") && !strings.Contains(ctype, "xml") {
		return nil, false
	}
	return body, true
}

func (f *Fetcher) get(ctx context.Context, raw string) ([]byte, string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, "", false
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil || len(body) < 32 {
		return nil, "", false
	}
	ctype := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if !strings.HasPrefix(ctype, "image/") {
		ctype = sniff(body)
		if ctype == "" {
			return nil, "", false
		}
	}
	return body, ctype, true
}

func sniff(b []byte) string {
	switch {
	case len(b) > 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8:
		return "image/jpeg"
	case len(b) > 6 && string(b[:6]) == "GIF89a":
		return "image/gif"
	case len(b) > 4 && string(b[:4]) == "RIFF":
		return "image/webp"
	case len(b) > 4 && string(b[:4]) == "<svg":
		return "image/svg+xml"
	case len(b) > 4 && (b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0):
		return "image/x-icon"
	}
	return ""
}

// Warm fetches favicons for the given domains in the background so a large
// directory page does not have to wait on third-party hosts. Cached entries are
// skipped; failures are cached like normal misses.
func (f *Fetcher) Warm(domains []string) {
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" || !domainRe.MatchString(d) {
			continue
		}
		f.mu.Lock()
		e, ok := f.cache[d]
		f.mu.Unlock()
		if ok {
			ttl := 24 * time.Hour
			if !e.ok {
				ttl = time.Hour
			}
			if time.Since(e.at) < ttl {
				continue
			}
		}
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _, _ = f.Get(ctx, domain)
		}(d)
	}
	wg.Wait()
}

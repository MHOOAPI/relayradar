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

	mu    sync.Mutex
	cache map[string]entry
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

	dialer := &net.Dialer{Timeout: 5 * time.Second}
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

	f := &Fetcher{allowLocal: allowLocal, cache: map[string]entry{}}
	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   true,
	}
	if proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}
	f.client = &http.Client{
		Timeout:   8 * time.Second,
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
	candidates := []string{"https://" + domain + "/favicon.ico"}
	// The s2 endpoint is a fixed public host, so it is safe to use as a
	// fallback even when local targets are allowed. Skip obviously fake domains.
	if !strings.HasSuffix(domain, ".invalid") && !strings.HasPrefix(domain, "localhost") {
		candidates = append(candidates,
			"https://www.google.com/s2/favicons?sz=64&domain="+url.QueryEscape(domain))
	}
	for _, raw := range candidates {
		if data, ctype, ok := f.get(ctx, raw); ok {
			return data, ctype, true
		}
	}
	return nil, "", false
}

func (f *Fetcher) get(ctx context.Context, raw string) ([]byte, string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, "", false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RelayRadarBot/1.0; +https://relayradar.example/bot)")
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

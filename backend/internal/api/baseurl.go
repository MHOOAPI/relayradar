package api

import (
	"net"
	"net/http"
	"strings"
)

// isLocalHost reports whether host belongs to the loopback/dev family that must
// never end up in a public canonical URL.
func isLocalHost(host string) bool {
	h := strings.TrimSpace(strings.ToLower(host))
	if strings.HasPrefix(h, "[") {
		if i := strings.Index(h, "]"); i > -1 {
			h = h[1:i]
		}
	} else if strings.Count(h, ":") == 1 {
		h = h[:strings.LastIndex(h, ":")]
	}
	switch h {
	case "", "localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal":
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback() || ip.IsUnspecified()
	}
	return strings.HasSuffix(h, ".localhost")
}

// publicBase returns the origin for absolute, crawlable URLs. A configured
// non-local PUBLIC_BASE_URL wins; otherwise the request's forwarded host is
// used so a deployment that forgot the variable never publishes
// http://localhost/ as its canonical.
func (s *Server) publicBase(r *http.Request) string {
	configured := strings.TrimRight(strings.TrimSpace(s.cfg.PublicBaseURL), "/")
	if configured != "" && !isLocalHost(originHost(configured)) {
		return configured
	}
	if r != nil {
		host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
		if i := strings.IndexByte(host, ','); i > -1 {
			host = strings.TrimSpace(host[:i])
		}
		if host == "" {
			host = r.Host
		}
		if host != "" && !isLocalHost(host) {
			scheme := ""
			if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
				if i := strings.IndexByte(p, ','); i > -1 {
					p = p[:i]
				}
				scheme = strings.TrimSpace(strings.ToLower(p))
			}
			if scheme != "http" && scheme != "https" {
				if r.TLS != nil {
					scheme = "https"
				} else {
					scheme = "http"
				}
			}
			return scheme + "://" + host
		}
	}
	return configured
}

func originHost(origin string) string {
	s := origin
	if i := strings.Index(s, "://"); i > -1 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i > -1 {
		s = s[:i]
	}
	return s
}

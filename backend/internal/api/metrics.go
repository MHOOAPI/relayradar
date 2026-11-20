package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// httpMetrics is a tiny Prometheus-style collector. Keeping it dependency-free
// avoids pulling a client library into the image just for a few series.
type httpMetrics struct {
	mu     sync.Mutex
	counts map[metricKey]int64
	durs   map[string]*histogram
}

type metricKey struct {
	method string
	route  string
	status string
}

var metricBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

type histogram struct {
	buckets []int64
	sum     float64
	count   int64
}

func newHTTPMetrics() *httpMetrics {
	return &httpMetrics{counts: map[metricKey]int64{}, durs: map[string]*histogram{}}
}

func metricsRoute(path string) string {
	switch {
	case path == "/" || path == "":
		return "/"
	case strings.HasPrefix(path, "/api/v1/stations/"), strings.HasPrefix(path, "/stations/"):
		return "/stations"
	case strings.HasPrefix(path, "/api/v1/models/"), strings.HasPrefix(path, "/models/"):
		return "/models"
	case strings.HasPrefix(path, "/en/stations/"):
		return "/en/stations"
	case strings.HasPrefix(path, "/en/models/"):
		return "/en/models"
	case strings.HasPrefix(path, "/api/v1/"):
		return "/api/v1"
	case strings.HasPrefix(path, "/share/"):
		return "/share"
	case path == "/en" || strings.HasPrefix(path, "/en/"):
		return "/en"
	case path == "/metrics":
		return "/metrics"
	default:
		return "other"
	}
}

func (m *httpMetrics) observe(method, path string, status int, d time.Duration) {
	route := metricsRoute(path)
	st := fmt.Sprintf("%d", status)
	m.mu.Lock()
	m.counts[metricKey{method, route, st}]++
	h := m.durs[route]
	if h == nil {
		h = &histogram{buckets: make([]int64, len(metricBuckets))}
		m.durs[route] = h
	}
	sec := d.Seconds()
	h.sum += sec
	h.count++
	for i, b := range metricBuckets {
		if sec <= b {
			h.buckets[i]++
		}
	}
	m.mu.Unlock()
}

func (m *httpMetrics) render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP relayradar_http_requests_total HTTP requests by method, route group and status.\n")
	b.WriteString("# TYPE relayradar_http_requests_total counter\n")
	keys := make([]metricKey, 0, len(m.counts))
	for k := range m.counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	for _, k := range keys {
		fmt.Fprintf(&b, "relayradar_http_requests_total{method=%q,route=%q,status=%q} %d\n", k.method, k.route, k.status, m.counts[k])
	}

	b.WriteString("# HELP relayradar_http_request_duration_seconds Request duration by route group.\n")
	b.WriteString("# TYPE relayradar_http_request_duration_seconds histogram\n")
	routes := make([]string, 0, len(m.durs))
	for r := range m.durs {
		routes = append(routes, r)
	}
	sort.Strings(routes)
	for _, r := range routes {
		h := m.durs[r]
		for i, bound := range metricBuckets {
			fmt.Fprintf(&b, "relayradar_http_request_duration_seconds_bucket{route=%q,le=%q} %d\n", r, fmt.Sprintf("%g", bound), h.buckets[i])
		}
		fmt.Fprintf(&b, "relayradar_http_request_duration_seconds_bucket{route=%q,le=\"+Inf\"} %d\n", r, h.count)
		fmt.Fprintf(&b, "relayradar_http_request_duration_seconds_sum{route=%q} %g\n", r, h.sum)
		fmt.Fprintf(&b, "relayradar_http_request_duration_seconds_count{route=%q} %d\n", r, h.count)
	}
	b.WriteString("# HELP relayradar_up 1 when the API is serving traffic.\n")
	b.WriteString("# TYPE relayradar_up gauge\n")
	b.WriteString("relayradar_up 1\n")
	return b.String()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) metricsMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		s.metrics.observe(r.Method, r.URL.Path, status, time.Since(start))
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.metrics.render()))
}

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"relayradar/internal/config"
)

func TestNoIndexPageMeta(t *testing.T) {
	const base = "https://relayradar.example"
	cases := []struct {
		name  string
		url   string
		match bool
	}{
		{"archived audit report", "/audit?report=abc123", true},
		{"audit tool page stays indexable", "/audit", false},
		{"audit check doc stays indexable", "/audit/ttft", false},
		{"claim token", "/claim/tok-123", true},
		{"email verification", "/verify?token=abc", true},
		{"password reset", "/reset?token=abc", true},
		{"forgot password", "/forgot", true},
		{"alert management", "/alerts?id=sub-1&token=tok", true},
		{"home page", "/", false},
		{"models page", "/models", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, ok := noIndexPageMeta(base, httptest.NewRequest(http.MethodGet, c.url, nil))
			if ok != c.match {
				t.Fatalf("match = %v, want %v", ok, c.match)
			}
			if !ok {
				return
			}
			if !m.NoIndex {
				t.Error("matched page must be noindex")
			}
			if m.Title == "" || m.Canonical == "" {
				t.Errorf("noindex page needs a title and a canonical target, got %+v", m)
			}
			if strings.Contains(m.Canonical, "token=") || strings.Contains(m.Canonical, "/claim/") {
				t.Errorf("canonical must not carry a token: %q", m.Canonical)
			}
		})
	}
}

func TestRobotsTxt(t *testing.T) {
	s := &Server{cfg: config.Config{PublicBaseURL: "https://relayradar.example"}}
	rec := httptest.NewRecorder()
	s.handleRobots(rec, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Disallow: /api/",
		"Disallow: /claim/",
		"Sitemap: https://relayradar.example/sitemap.xml",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt missing %q:\n%s", want, body)
		}
	}
	// /audit/{key} pages are advertised in sitemap.xml, so the whole tree must
	// not be disallowed; individual report pages are noindexed instead.
	if strings.Contains(body, "Disallow: /audit\n") {
		t.Errorf("robots.txt disallows /audit while its check pages are in the sitemap:\n%s", body)
	}
}

func TestSitemapAuditPagesAreCrawlable(t *testing.T) {
	// Guard the contradiction fixed in robots.txt: every sitemap entry must be
	// allowed by the User-agent: * group.
	s := &Server{cfg: config.Config{PublicBaseURL: "https://relayradar.example"}}
	rec := httptest.NewRecorder()
	s.handleRobots(rec, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	disallowed := []string{}
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "Disallow: "); ok && v != "" {
			disallowed = append(disallowed, v)
		}
	}
	for _, c := range auditCheckDocs {
		loc := "/audit/" + c.Key
		for _, d := range disallowed {
			if strings.HasPrefix(loc, d) {
				t.Errorf("sitemap entry %s is blocked by robots rule %s", loc, d)
			}
		}
	}
}

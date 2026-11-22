package api

import (
	"net/http/httptest"
	"testing"

	"relayradar/internal/config"
)

func TestIsLocalHost(t *testing.T) {
	local := []string{"localhost", "localhost:8080", "127.0.0.1", "127.0.0.1:8088", "[::1]:8088", "::1", "0.0.0.0", "", "app.localhost:3000", "host.docker.internal"}
	for _, h := range local {
		if !isLocalHost(h) {
			t.Errorf("isLocalHost(%q) = false, want true", h)
		}
	}
	public := []string{"relayradar.example", "relayradar.example:443", "api.10.0.0.5.nip.io", "192.168.1.20"}
	for _, h := range public {
		if isLocalHost(h) {
			t.Errorf("isLocalHost(%q) = true, want false", h)
		}
	}
}

func TestPublicBasePrefersConfiguredPublicOrigin(t *testing.T) {
	s := &Server{cfg: config.Config{PublicBaseURL: "https://relayradar.example/"}}
	r := httptest.NewRequest("GET", "http://internal:8080/", nil)
	r.Host = "internal:8080"
	if got := s.publicBase(r); got != "https://relayradar.example" {
		t.Fatalf("publicBase = %q, want configured origin", got)
	}
}

func TestPublicBaseFallsBackToRequestHost(t *testing.T) {
	s := &Server{cfg: config.Config{PublicBaseURL: "http://localhost:8088"}}
	r := httptest.NewRequest("GET", "http://relayradar.example/stations", nil)
	r.Host = "relayradar.example"
	r.Header.Set("X-Forwarded-Proto", "https")

	if got := s.publicBase(r); got != "https://relayradar.example" {
		t.Fatalf("publicBase = %q, want request origin instead of localhost", got)
	}
}

func TestPublicBaseUsesForwardedHost(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	r.Host = "127.0.0.1:8080"
	r.Header.Set("X-Forwarded-Host", "relayradar.example, proxy.internal")
	r.Header.Set("X-Forwarded-Proto", "https, http")

	if got := s.publicBase(r); got != "https://relayradar.example" {
		t.Fatalf("publicBase = %q, want first forwarded host and proto", got)
	}
}

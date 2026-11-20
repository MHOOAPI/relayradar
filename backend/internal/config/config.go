package config

import (
	"os"
	"strconv"
	"strings"
)

// Config is loaded from the environment once at boot.
type Config struct {
	Addr         string
	DatabaseURL  string
	StoreMode    string // postgres | memory
	WebDir       string // optional: serve the built SPA from this directory
	PublicBaseURL string
	FaviconProxy bool
	// FaviconProxyURL optionally routes favicon fetches through an HTTP proxy
	// (icons only; audit probes still never use a proxy).
	FaviconProxyURL string
	SeedDemo     bool
	CORSOrigins  []string
	AuditEnabled bool
	// SMTP settings for email alert subscriptions. Leave SMTPAddr empty to
	// disable the email channel.
	SMTPAddr string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
	// AlertFailureEmail receives failure notices for webhook subscriptions.
	AlertFailureEmail string
	// ClaimDevBypass skips DNS/file ownership checks for local demos.
	ClaimDevBypass bool
	// ReachabilityEnabled toggles the credential-free homepage checker.
	ReachabilityEnabled bool
	// ReachabilityDownThreshold is consecutive down checks before alerting.
	ReachabilityDownThreshold int
	// ProbeIngestToken authenticates POST /api/v1/probes from the prober.
	ProbeIngestToken string
	// ProbeConfigPath is where the admin probe config is stored.
	ProbeConfigPath string
	// AdminEmails grants the admin role to these addresses on registration.
	AdminEmails []string
	// AllowLocalTargets relaxes the SSRF guard for local development only.
	// Never enable it on a public deployment.
	AllowLocalTargets bool
}

func (c Config) SMTPConfigured() bool { return c.SMTPAddr != "" && c.SMTPFrom != "" }

func Load() Config {
	c := Config{
		Addr:         env("ADDR", ":8080"),
		DatabaseURL:  env("DATABASE_URL", "postgres://relayradar:relayradar@localhost:5432/relayradar?sslmode=disable"),
		StoreMode:    env("STORE", "postgres"),
		WebDir:       env("WEB_DIR", ""),
		PublicBaseURL: env("PUBLIC_BASE_URL", "http://localhost:8080"),
		FaviconProxy: envBool("FAVICON_PROXY", true),
		FaviconProxyURL: env("FAVICON_PROXY_URL", ""),
		SeedDemo:     envBool("SEED_DEMO", true),
		AuditEnabled: envBool("AUDIT_ENABLED", true),
		CORSOrigins:  strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ","),
		SMTPAddr:     env("SMTP_ADDR", ""),
		SMTPUser:     env("SMTP_USER", ""),
		SMTPPass:     env("SMTP_PASS", ""),
		SMTPFrom:     env("SMTP_FROM", "alerts@relayradar.local"),
		AlertFailureEmail: env("ALERT_FAILURE_EMAIL", ""),
		ClaimDevBypass:    env("CLAIM_DEV_BYPASS", "") == "true",
		ReachabilityEnabled: env("REACHABILITY_ENABLED", "true") != "false",
		ReachabilityDownThreshold: envInt("REACHABILITY_DOWN_THRESHOLD", 2),
		ProbeIngestToken:          env("PROBE_INGEST_TOKEN", ""),
		ProbeConfigPath:           env("PROBES_CONFIG_PATH", "probes.json"),
		AdminEmails:  splitCSV(env("ADMIN_EMAILS", "")),
	}
	c.AllowLocalTargets = envBool("ALLOW_LOCAL_TARGETS", false)
	return c
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func splitCSV(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envBool(k string, def bool) bool {	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

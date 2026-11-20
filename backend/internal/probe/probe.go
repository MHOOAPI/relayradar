// Package probe implements the real, credential-carrying probe used by the
// standalone prober worker. The API itself never probes stations; it only
// serves measurements.
package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Target struct {
	Slug    string `json:"slug"`
	Service string `json:"service"`
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

type Config struct {
	Stations []Target `json:"stations"`
}

type Result struct {
	Status    string // ok | slow | down
	LatencyMS int
	Detail    string
}

// Run sends one short completion and classifies it. "slow" still counts as a
// success for availability; only transport or HTTP errors are "down".
func Run(ctx context.Context, t Target, client *http.Client) Result {
	base := strings.TrimRight(t.BaseURL, "/")
	if !strings.HasSuffix(base, "/chat/completions") {
		if strings.HasSuffix(base, "/v1") {
			base += "/chat/completions"
		} else {
			base += "/v1/chat/completions"
		}
	}
	body, _ := json.Marshal(map[string]any{
		"model":      t.Model,
		"messages":   []map[string]string{{"role": "user", "content": "Reply with OK."}},
		"max_tokens": 4,
		"stream":     false,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base, bytes.NewReader(body))
	if err != nil {
		return Result{Status: "down", Detail: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if t.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.APIKey)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Result{Status: "down", LatencyMS: int(time.Since(start).Milliseconds()), Detail: err.Error()}
	}
	defer resp.Body.Close()
	latency := int(time.Since(start).Milliseconds())
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{Status: "down", LatencyMS: latency, Detail: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	if latency > 3000 {
		return Result{Status: "slow", LatencyMS: latency, Detail: "latency above 3s"}
	}
	return Result{Status: "ok", LatencyMS: latency}
}

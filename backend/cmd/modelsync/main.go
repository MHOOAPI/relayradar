// Command modelsync pulls the live OpenRouter model catalog and writes the
// embedded catalog used by the demo seed. Run it to refresh model data:
//
//	go run ./cmd/modelsync
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type openRouterModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Created       int64  `json:"created"`
	Description   string `json:"description"`
	ContextLength int    `json:"context_length"`
	Pricing       struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
	Architecture struct {
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
}

type catalogModel struct {
	ID             string    `json:"id"`
	Vendor         string    `json:"vendor"`
	VendorName     string    `json:"vendorName"`
	Name           string    `json:"name"`
	Service        string    `json:"service"`
	ContextWindow  int       `json:"contextWindow"`
	InputUSD       float64   `json:"inputUsd"`
	OutputUSD      float64   `json:"outputUsd"`
	ReleasedAt     time.Time `json:"releasedAt"`
	CodingScore    float64   `json:"codingScore"`
	ReasoningScore float64   `json:"reasoningScore"`
	FreeTier       bool      `json:"freeTier"`
	FreeNote       string    `json:"freeNote"`
	Summary        string    `json:"summary"`
	Category       string    `json:"category"`
}

var vendorNames = map[string]string{
	"openai": "OpenAI", "anthropic": "Anthropic", "google": "Google", "deepseek": "DeepSeek",
	"x-ai": "xAI", "xai": "xAI", "qwen": "阿里通义", "moonshotai": "Moonshot", "meta-llama": "Meta",
	"mistralai": "Mistral", "bytedance": "字节跳动", "minimax": "MiniMax", "nvidia": "NVIDIA",
	"amazon": "Amazon", "microsoft": "Microsoft", "cohere": "Cohere", "perplexity": "Perplexity",
	"black-forest-labs": "Black Forest Labs", "stabilityai": "Stability AI", "kuaishou": "快手",
	"runwayml": "Runway", "pika": "Pika", "midjourney": "Midjourney", "baidu": "百度",
	"z-ai": "智谱", "thudm": "智谱", "01-ai": "零一万物", "tencent": "腾讯",
}

var serviceForVendor = map[string]string{
	"openai": "openai", "anthropic": "claude", "google": "gemini", "deepseek": "deepseek",
	"x-ai": "xai", "xai": "xai", "qwen": "qwen", "moonshotai": "moonshot",
}

func main() {
	out := "internal/store/models_catalog.json"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Get("https://openrouter.ai/api/v1/models")
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, "status:", resp.StatusCode)
		os.Exit(1)
	}
	var payload struct {
		Data []openRouterModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}

	seen := map[string]bool{}
	models := make([]catalogModel, 0, len(payload.Data))
	for _, m := range payload.Data {
		// Skip models without an input price: they would break price comparisons.
		if m.Pricing.Prompt == "0" || m.Pricing.Prompt == "" {
			continue
		}
		id := strings.NewReplacer("/", "--", ":", "-").Replace(m.ID)
		if seen[id] {
			continue
		}
		seen[id] = true
		vendor := m.ID
		if i := strings.Index(m.ID, "/"); i > 0 {
			vendor = m.ID[:i]
		}
		name := m.Name
		if i := strings.Index(name, ": "); i > 0 && strings.HasPrefix(name, vendorNames[vendor]) {
			name = name[i+2:]
		}
		category := "text"
		for _, o := range m.Architecture.OutputModalities {
			if o == "image" {
				category = "image"
			}
			if o == "video" {
				category = "video"
			}
		}
		summary := strings.TrimSpace(m.Description)
		if r := []rune(summary); len(r) > 160 {
			summary = string(r[:160]) + "..."
		}
		released := time.Unix(m.Created, 0).UTC()
		if m.Created == 0 {
			released = time.Now().UTC()
		}
		service := serviceForVendor[vendor]
		if category != "text" {
			service = ""
		}
		models = append(models, catalogModel{
			ID:             id,
			Vendor:         vendor,
			VendorName:     displayVendor(vendor),
			Name:           name,
			Service:        service,
			ContextWindow:  m.ContextLength,
			InputUSD:       perMillion(m.Pricing.Prompt),
			OutputUSD:      perMillion(m.Pricing.Completion),
			ReleasedAt:     released,
			CodingScore:    0,
			ReasoningScore: 0,
			Summary:        summary,
			Category:       category,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ReleasedAt.After(models[j].ReleasedAt) })

	b, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d models to %s\n", len(models), out)
}

func displayVendor(v string) string {
	if n, ok := vendorNames[v]; ok {
		return n
	}
	return v
}

func perMillion(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f * 1_000_000
}

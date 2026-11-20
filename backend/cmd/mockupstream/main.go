// Command mockupstream is a tiny OpenAI-compatible SSE server used to demo and
// test the audit probe without spending real tokens.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var mockModels = []string{"claude-opus-5-5", "claude-fable-5-2", "gpt-6.2-astra", "gemini-4-carbon"}

func main() {
	var calls int64
	addr := flag.String("addr", ":9099", "listen address")
	flag.Parse()

	http.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		type item struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		}
		out := struct {
			Object string `json:"object"`
			Data   []item `json:"data"`
		}{Object: "list"}
		for _, id := range mockModels {
			out.Data = append(out.Data, item{ID: id, Object: "model", OwnedBy: "mock"})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	http.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		log.Printf("webhook event=%s sig=%s body=%s",
			r.Header.Get("X-RelayRadar-Event"), r.Header.Get("X-RelayRadar-Signature"), string(b))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || strings.HasSuffix(auth, "bad-key") {
			http.Error(w, `{"error":{"message":"invalid api key"}}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		n := atomic.AddInt64(&calls, 1)
		cached := 0
		if n%2 == 0 {
			cached = 120
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		write := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}

		time.Sleep(320 * time.Millisecond) // simulated TTFT
		for _, part := range []string{"READ", "Y"} {
			write(map[string]any{
				"id": "chatcmpl-mock", "object": "chat.completion.chunk", "model": req.Model,
				"choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": part}}},
			})
			time.Sleep(40 * time.Millisecond)
		}
		write(map[string]any{
			"id": "chatcmpl-mock", "object": "chat.completion.chunk", "model": req.Model,
			"choices": []any{},
			"usage": map[string]any{
				"prompt_tokens": 137, "completion_tokens": 2,
				"prompt_tokens_details": map[string]int{"cached_tokens": cached},
			},
		})
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Println("mock upstream on :9099")
	log.Fatal(http.ListenAndServe(*addr, nil))
}

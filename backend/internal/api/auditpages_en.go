package api

import (
	"net/http"
)

// auditCheckDocsEN mirrors the Chinese check docs for the English edition.
var auditCheckDocsEN = []AuditCheckDoc{
	{
		Key: "reachable", Label: "Endpoint reachable",
		Short: "The base URL opens a TLS connection and returns 2xx.",
		Why:   "If the endpoint is unreachable, model echo, usage and cache checks cannot run; rule out network and certificate issues first.",
		How:   "Send a minimal request to {baseUrl}/chat/completions and record HTTP status, TLS handshake and total time.",
		Pass:  "HTTP 2xx, valid certificate, response within timeout.",
		Warn:  "2xx but close to the timeout, or a chain that needs extra intermediates.",
		Fail:  "Connection refused, DNS failure, certificate error, or 4xx/5xx.",
		Limit: "A single successful call says nothing about peak-hour or long-lived stability.",
	},
	{
		Key: "auth", Label: "Authentication accepted",
		Short: "Separate “bad key” from “provider broken”.",
		Why:   "401/403 is usually your key, not the provider being down; labelling it separately avoids false alarms.",
		How:   "Read the status and error body: 401/403 counts as auth failure, other errors as upstream errors.",
		Pass:  "Request accepted and content returned.",
		Warn:  "403 with a region, model or group restriction in the body.",
		Fail:  "401, or an explicit invalid api key error.",
		Limit: "Error codes are not standardised; some providers wrap failures in 200 + error JSON.",
	},
	{
		Key: "stream", Label: "Stream integrity",
		Short: "SSE deltas and the [DONE] terminator are present.",
		Why:   "A truncated stream hangs clients; without [DONE] many SDKs keep waiting.",
		How:   "Request stream=true and count data: lines, delta.content chunks and [DONE].",
		Pass:  "At least one content delta and a [DONE] terminator.",
		Warn:  "Content present but [DONE] missing, or an abnormal gap between deltas.",
		Fail:  "A full JSON body instead of SSE, or the stream stops before content.",
		Limit: "Does not judge content quality, only protocol completeness.",
	},
	{
		Key: "model_echo", Label: "Model echo",
		Short: "The response model matches the requested model.",
		Why:   "Substitution is most visible when the response echoes a different model name.",
		How:   "Request a model, then compare the model field in the body or stream chunks.",
		Pass:  "Exact match, or only case/alias differences.",
		Warn:  "An alias is echoed and the mapping cannot be confirmed publicly.",
		Fail:  "A different model is echoed, or the field is missing.",
		Limit: "The field can be forged; combine it with thinking signatures and behaviour fingerprints.",
	},
	{
		Key: "usage", Label: "Usage reported",
		Short: "usage exists and its fields are plausible.",
		Why:   "Without usage you cannot reconcile billing; some providers inflate tokens with their own tokenizer.",
		How:   "Read usage.prompt_tokens and completion_tokens and check they are positive.",
		Pass:  "usage exists and both token counts are positive.",
		Warn:  "usage exists but only totals, no details.",
		Fail:  "No usage field, or always zero.",
		Limit: "usage is computed by the provider; it cannot prove tokenizer inflation on its own.",
	},
	{
		Key: "token_consistency", Label: "Tokenizer consistency",
		Short: "Two identical requests report the same prompt_tokens.",
		Why:   "Different token counts for the same text suggest a different tokenizer or a rewritten request.",
		How:   "Send the exact same prompt twice and compare prompt_tokens.",
		Pass:  "Identical both times.",
		Warn:  "A 1–2 token difference (hidden system prompt effects).",
		Fail:  "A clear difference, or usage missing on the second call.",
		Limit: "Cannot separate a rewritten prompt from upstream instability.",
	},
	{
		Key: "cache", Label: "Cache passthrough",
		Short: "The second identical request reports cached_tokens.",
		Why:   "Official cache reads are often 0.1×; a provider that swallows the cache multiplies long-context cost by roughly 10×.",
		How:   "Send the same request twice and read prompt_tokens_details.cached_tokens on the second.",
		Pass:  "cached_tokens > 0 on the second call.",
		Warn:  "A cache field exists but is zero, or caching is declared unsupported.",
		Fail:  "No cache field and full price still charged.",
		Limit: "Cache hits depend on upstream policy; one miss does not prove the cache is swallowed.",
	},
	{
		Key: "ttft", Label: "First token latency",
		Short: "Time from request to the first content delta.",
		Why:   "TTFT drives perceived speed and reveals queueing or forwarding.",
		How:   "Measure request start to the first delta.content.",
		Pass:  "Under 1500 ms.",
		Warn:  "1500–4000 ms.",
		Fail:  "Over 4000 ms, or no content until the stream ends.",
		Limit: "One sample depends on network and upstream load; measure several times and take the median.",
	},
}

func auditCheckENByKey(key string) *AuditCheckDoc {
	for i := range auditCheckDocsEN {
		if auditCheckDocsEN[i].Key == key {
			return &auditCheckDocsEN[i]
		}
	}
	return nil
}

func (s *Server) handleAuditChecksEN(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": auditCheckDocsEN, "count": len(auditCheckDocsEN)})
}

func (s *Server) handleAuditCheckEN(w http.ResponseWriter, r *http.Request) {
	c := auditCheckENByKey(r.PathValue("key"))
	if c == nil {
		writeError(w, http.StatusNotFound, "check not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func auditCheckBodyEN(c *AuditCheckDoc) string {
	return `<article><h1>` + escHTML(c.Label) + `</h1><p>` + escHTML(c.Short) + `</p>` +
		`<h2>Why it matters</h2><p>` + escHTML(c.Why) + `</p>` +
		`<h2>How we measure</h2><p>` + escHTML(c.How) + `</p>` +
		`<h2>Verdicts</h2><ul><li><b>Pass</b>: ` + escHTML(c.Pass) + `</li>` +
		`<li><b>Warn</b>: ` + escHTML(c.Warn) + `</li>` +
		`<li><b>Fail</b>: ` + escHTML(c.Fail) + `</li></ul>` +
		`<h2>Limitations</h2><p>` + escHTML(c.Limit) + `</p>` +
		`<p><a href="/en/audit">Run an audit</a> · <a href="/audit/` + escHTML(c.Key) + `">中文说明</a></p></article>`
}

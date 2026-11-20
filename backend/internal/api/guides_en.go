package api

import (
	"html/template"
	"net/http"
)

// guideEN is the English edition of a guide. Chinese content lives in guides.go.
type guideEN struct {
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	Desc     string        `json:"desc"`
	Updated  string        `json:"updated"`
	Minutes  int           `json:"minutes"`
	Body     template.HTML `json:"-"`
	BodyHTML string        `json:"bodyHtml"`
}

func init() {
	for slug, g := range guidesEN {
		g.BodyHTML = string(g.Body)
		guidesEN[slug] = g
	}
}

var guidesEN = map[string]guideEN{
	"how-to-choose-relay": {
		Slug:    "how-to-choose-relay",
		Title:   "How to choose an LLM API relay: a 10-minute checklist",
		Desc:    "Authenticity, availability, billing and refunds — an ordered checklist plus the five most common cheap-relay traps.",
		Updated: "2026-11-19",
		Minutes: 7,
		Body: template.HTML(`<p>Choosing a relay is not about the lowest headline price. It is about picking a provider whose failures you can measure and whose money you can get back. Run this list in order; ten minutes is enough to filter most of the noise.</p>

<h2>Three questions first</h2>
<ol>
<li><strong>Is it actually forwarding the model you asked for?</strong> Check model echo, thinking signatures and behaviour, not the landing page.</li>
<li><strong>Is it stable?</strong> Look at 7-day availability and latency jitter, not a "99.99%" banner.</li>
<li><strong>Can you reconcile the bill?</strong> Exchange rate, group multiplier, tokenizer and cache rules must add up to what you are charged.</li>
</ol>

<h2>10-minute checklist</h2>
<table class="index-table">
<thead><tr><th>Check</th><th>How</th><th>Pass</th></tr></thead>
<tbody>
<tr><td>Domain age</td><td>WHOIS or the listing age on this site</td><td>≥ 6 months; top up small on new providers</td></tr>
<tr><td>Model echo</td><td>Send a request and read the response model field</td><td>Matches the request; otherwise it may be substituted</td></tr>
<tr><td>Usage reporting</td><td>Fixed prompt, inspect usage.prompt_tokens</td><td>Identical across two identical requests</td></tr>
<tr><td>Cache passthrough</td><td>Same request twice, inspect cached_tokens</td><td>Second call shows a cache hit and lower cost</td></tr>
<tr><td>Thinking signature</td><td>Claude with thinking enabled, check signature</td><td>Present and stable; missing suggests stripping</td></tr>
<tr><td>Real multiplier</td><td>Top up $1, run $1 of usage, compare the charge</td><td>Charge = official × rate × multiplier</td></tr>
<tr><td>Refund terms</td><td>Ask in writing whether unused balance is refundable</td><td>Written policy, not a chat promise</td></tr>
</tbody>
</table>
<p>Our <a href="/audit">audit</a> covers four of these automatically: echo, usage, tokenizer consistency and cache.</p>

<h2>Five common traps</h2>
<ul>
<li><strong>Cheap group, different upstream:</strong> the landing page shows a 0.1× group, but your key is routed to a cheaper, less stable upstream. Read the group, not the banner.</li>
<li><strong>Swallowed cache:</strong> official cache reads are often 0.1×; a relay that bills full price can multiply long-context costs.</li>
<li><strong>Tokenizer inflation:</strong> a different tokenizer can count 10–30% more tokens for the same text.</li>
<li><strong>Fake signatures:</strong> a correctly shaped but invalid signature, or no thinking block at all.</li>
<li><strong>Limited-time multipliers:</strong> the rate quietly changes after top-up. Our change feed records observed price adjustments.</li>
</ul>

<h2>Business terms matter too</h2>
<p>Refund policy, invoice type and minimum top-up define your exit cost. A provider that is 10% more expensive but refunds unused balance is usually a better production choice than a 30% cheaper one with "no refunds".</p>

<h2>Recommended order</h2>
<ol>
<li>Start with a provider at ≥ 99% availability and labelled official-direct.</li>
<li>Top up the minimum, run one audit with a temporary key.</li>
<li>Watch latency and billing for 3–7 days before moving production traffic.</li>
<li>Keep one cheap backup for failover.</li>
</ol>
<p>For a side-by-side view, use <a href="/compare">the comparison page</a>; it puts availability, speed, consistency, purity and business terms in one table.</p>`),
	},
	"multiplier-math": {
		Slug:    "multiplier-math",
		Title:   "Multipliers, exchange rates and caching: what a relay actually costs",
		Desc:    "Four factors multiply into your real CNY price per million tokens; a worked example plus the cache and tokenizer effects.",
		Updated: "2026-11-19",
		Minutes: 6,
		Body: template.HTML(`<p>Relay pricing pages usually show two numbers: an exchange rate and a multiplier. Real cost is four factors multiplied together, and hiding any one of them understates the bill.</p>

<h2>The formula</h2>
<pre><code>real price (CNY / 1M tokens)
  = official price (USD / 1M)
  × settlement rate (CNY per USD)
  × group multiplier
  × tokenizer inflation</code></pre>
<p>The "settlement rate" is not the bank rate; it is how many CNY the provider charges for $1 of credit. Typical range: 1.0–2.2.</p>

<h2>A worked example</h2>
<p>Claude Opus 5.5 is $4 / $20 per 1M (input / output). A provider with rate 1.5 and multiplier 0.4:</p>
<table class="index-table">
<thead><tr><th>Item</th><th>Math</th><th>Result</th></tr></thead>
<tbody>
<tr><td>Input</td><td>4 × 1.5 × 0.4</td><td>¥2.40 / 1M</td></tr>
<tr><td>Output</td><td>20 × 1.5 × 0.4</td><td>¥12.00 / 1M</td></tr>
<tr><td>With 1.2× inflation</td><td>multiply by 1.2</td><td>¥2.88 / ¥14.40</td></tr>
</tbody>
</table>
<p>So "0.4 multiplier" is really 0.48 when the tokenizer inflates by 20%. Always verify with a small top-up.</p>

<h2>Why caching matters</h2>
<p>Official cache reads are often billed at 0.1×. For a 500K-token repository context called 100 times a day:</p>
<ul>
<li>Cache works: 99 of 100 calls bill at 0.1×;</li>
<li>Cache swallowed: every call bills full price — roughly 9× the cost.</li>
</ul>
<p>Test it: send the same request twice and check usage for cached_tokens on the second call.</p>

<h2>Three wording traps</h2>
<ul>
<li><strong>"0.8 discount" vs "0.8 multiplier":</strong> the same thing; but "2 折" in Chinese means 20% of the price, which people misread as 2×.</li>
<li><strong>Per-request billing:</strong> some providers charge per call for long contexts; the headline unit price looks tiny.</li>
<li><strong>Bonus credit:</strong> usually non-refundable, non-invoiceable and restricted to certain groups.</li>
</ul>

<h2>Do the math yourself</h2>
<p>Use <a href="/compare">the comparison page</a> or the <a href="/en/models">model directory</a>. Prices are computed, not scored; the provider's actual charge is the final word.</p>`),
	},
	"signature-verification": {
		Slug:    "signature-verification",
		Title:   "Thinking signatures and “downgraded” relays: how to tell",
		Desc:    "What a thinking signature is, how relays strip or fake it, and a verification order that does not require an official key.",
		Updated: "2026-11-19",
		Minutes: 8,
		Body: template.HTML(`<p>"Downgraded" usually does not mean the model got worse. It means you are not talking to that model. Claude's thinking signature is the easiest signal to observe, but it is not a silver bullet.</p>

<h2>What the signature is</h2>
<p>With extended thinking enabled, Claude returns thinking blocks containing an opaque, server-generated signature. Clients cannot forge it; upstream can use it to verify that the thinking came from that model.</p>
<p>The common myth is "a 64-byte private-key signature". It has no fixed 64-byte structure and its length varies by implementation, so length checks are unreliable.</p>

<h2>What relays do</h2>
<ul>
<li><strong>Strip thinking:</strong> to save tokens or support old clients, drop the thinking block; the signature disappears.</li>
<li><strong>Substitute models:</strong> answer with a cheaper model; the signature is missing or does not match.</li>
<li><strong>Truncate context:</strong> silently cut long requests while billing as usual.</li>
<li><strong>Fake the field:</strong> return a correctly shaped but random signature that normal clients cannot distinguish.</li>
</ul>

<h2>An actionable order</h2>
<ol>
<li>Enable thinking explicitly and confirm a thinking block exists.</li>
<li>Inspect the signature field: present and stable across repeated calls.</li>
<li>Check model echo: the response model must match the request.</li>
<li>Run a behavioural fingerprint: a multi-step reasoning task that a distilled model usually fails.</li>
<li>Cross-check against an official endpoint when you have one.</li>
</ol>
<p>Our <a href="/audit">audit</a> covers steps 2–3 plus stream integrity, usage and caching. Behavioural fingerprinting and signature cross-verification remain on the roadmap; we do not market them as done.</p>

<h2>Why signature alone is not enough</h2>
<p>A signature proves the thinking came from the official model; it does not prove this particular answer was not downgraded. Some relays route simple requests to the official API and complex ones to a cheaper upstream, so a single sample can miss it.</p>
<ul>
<li>Test at different times, not once;</li>
<li>Watch availability, latency and price together — suspiciously cheap usually means a suspicious upstream;</li>
<li>Combine signature, echo, cache and tokenizer consistency instead of trusting one metric.</li>
</ul>

<h2>Bottom line</h2>
<p>A signature is necessary but not sufficient. Its value is that a missing signature is almost certainly a problem — not that a present one proves everything is fine.</p>
<p>Related: <a href="/en/methodology">methodology</a> explains why purity is not part of the score; <a href="/changes">the change feed</a> records purity alerts.</p>`),
	},
}

func guideENBySlug(slug string) *guideEN {
	if g, ok := guidesEN[slug]; ok {
		return &g
	}
	return nil
}

func guideENList() []guideEN {
	out := make([]guideEN, 0, len(guidesEN))
	for _, slug := range []string{"how-to-choose-relay", "multiplier-math", "signature-verification"} {
		if g, ok := guidesEN[slug]; ok {
			out = append(out, g)
		}
	}
	return out
}

func (s *Server) handleGuidesEN(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": guideENList()})
}

func (s *Server) handleGuideEN(w http.ResponseWriter, r *http.Request) {
	g := guideENBySlug(r.PathValue("slug"))
	if g == nil {
		writeError(w, http.StatusNotFound, "guide not found")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

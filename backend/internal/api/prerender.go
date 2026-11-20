package api

import (
	"encoding/xml"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"relayradar/internal/app"
)

// ---------------------------------------------------------------------------
// Prerendered, crawler-visible body. It lives inside #root and is replaced by
// React on first render, so users get the app while non-JS crawlers get real
// headings, tables and links.
// ---------------------------------------------------------------------------

func escHTML(s string) string { return html.EscapeString(s) }

func tableHTML(head []string, rows [][]string) template.HTML {
	var b strings.Builder
	b.WriteString(`<table class="index-table"><thead><tr>`)
	for _, h := range head {
		b.WriteString("<th>" + escHTML(h) + "</th>")
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, c := range row {
			b.WriteString("<td>" + c + "</td>") // cells are pre-escaped by callers
		}
		b.WriteString("</tr>")
	}
	b.WriteString(`</tbody></table>`)
	return template.HTML(b.String())
}

func stationLink(st app.StationView) string {
	return `<a href="/stations/` + escHTML(st.Slug) + `">` + escHTML(st.Name) + `</a>`
}

func (s *Server) bodyFor(r *http.Request, d *app.Dataset) template.HTML {
	path := r.URL.Path
	switch {
	case path == "/":
		return s.bodyHome(d)
	case path == "/leaderboards":
		return s.bodyLeaderboards(d)
	case path == "/models":
		return s.bodyModels(d)
	case path == "/stations":
		return bodyStationsDirectory(d)
	case strings.HasPrefix(path, "/stations/"):
		return s.bodyStation(d, strings.TrimPrefix(path, "/stations/"))
	case path == "/models/license":
		return s.bodyModelLicense(d, "")
	case strings.HasPrefix(path, "/models/license/"):
		return s.bodyModelLicense(d, strings.TrimPrefix(path, "/models/license/"))
	case strings.HasPrefix(path, "/models/"):
		return s.bodyModel(d, strings.TrimPrefix(path, "/models/"))
	case path == "/changes":
		return s.bodyChanges(d)
	case strings.HasPrefix(path, "/model-status/"):
		key := strings.TrimPrefix(path, "/model-status/")
		for _, p := range modelProviders {
			if p.Key == key {
				return template.HTML(`<article><h1>` + escHTML(p.Name) + ` 官方 API 状态</h1><p>当前状态与历史事件时间线来自 <a href="` + escHTML(p.StatusURL) + `" rel="nofollow noopener">官方状态页</a>。</p><p><a href="/model-status">所有厂商</a> · <a href="/status">本站服务状态</a></p></article>`)
			}
		}
		return ""
	case path == "/model-status":
		return template.HTML(`<article><h1>上游模型 API 状态</h1><p>聚合 OpenAI、Anthropic、Google Gemini、DeepSeek、Moonshot、阿里 Qwen 与 xAI 的官方 API 状态；优先读取官方状态页接口，缺失时用端点可达性探测兜底。</p><ul><li>OpenAI — status.openai.com</li><li>Anthropic — status.anthropic.com</li><li>Google Gemini — status.cloud.google.com</li><li>DeepSeek — status.deepseek.com</li><li>Moonshot — status.moonshot.cn</li><li>Alibaba Qwen — status.aliyun.com</li><li>xAI — status.x.ai</li></ul><p>这里展示的是上游厂商状态，不是 RelayRadar 自身；本站状态见 <a href="/status">/status</a>。</p></article>`)
	case path == "/status":
		return template.HTML(`<article><h1>服务状态</h1><p>公开 API、数据存储、探针采集、告警投递、接口检测与邮件通道的实时状态，每 30 秒刷新。</p><p>接口：<code>/api/v1/status</code>。当前演示部署暂无公开事故记录。</p></article>`)
	case path == "/free-api-keys":
		return template.HTML(`<article><h1>免费 API Key 分享</h1><p>用户提交的 OpenAI 兼容 Key，无需登录即可提交或领取。提交后会自动探测一次可用性；共享 Key 随时可能被吊销，请勿用于生产或绑定付费。</p><p>使用须知：不要提交绑定支付方式的主 Key；领取后请自行验证；分享 Key 可能违反部分服务商条款。</p><p><a href="/free">官方免费额度</a> · <a href="/">返回榜单</a></p></article>`)
	case path == "/free":
		return s.bodyFree(d)
	case path == "/faq":
		return faqBody()
	case path == "/guides":
		return guidesIndexHTML()
	case strings.HasPrefix(path, "/guides/"):
		if g := guideBySlug(strings.TrimPrefix(path, "/guides/")); g != nil {
			return guideBodyHTML(g)
		}
	case path == "/methodology":
		return template.HTML(`<article><h1>评分方法 v2</h1>
<p>综合分 = 0.62 × 可用性 + 0.26 × 速度 + 0.12 × 稳定性。可用性取 Wilson 95% 置信下界，速度由延迟中位数映射，稳定性衡量 p95 抖动；每个轴先做覆盖加权，再与最差分组混合（80% / 20%）。纯度与价格单独展示，不参与排名。</p>
<p>模型档案页另有「供给评分」：价格 0.35 · 覆盖 0.20 · 可用性 0.20 · 纯度 0.15 · 上下文 0.10，按可用维度归一；价格分 = 100 ×（1 − 最低价 ÷ 中位价 ÷ 2）。它只衡量购买成本与供给质量，不评价模型能力——能力需要公开基准，本站不做自测，因此不提供编程分或推理分。</p>
<p>完整公式、权重与 v1→v2 修订原因见 <a href="/methodology">评分方法页</a>。</p></article>`)
	case path == "/audit":
		return auditBody()
	case strings.HasPrefix(path, "/audit/"):
		if c := auditCheckByKey(strings.TrimPrefix(path, "/audit/")); c != nil {
			return auditCheckBody(c)
		}
	case path == "/en/changes":
		return enChangesBody(d)
	case path == "/en/free":
		return s.bodyEnFree(d)
	case path == "/en/audit":
		return enAuditBody()
	case strings.HasPrefix(path, "/en/audit/"):
		return template.HTML(auditCheckBodyEN(auditCheckENByKey(strings.TrimPrefix(path, "/en/audit/"))))
	case path == "/en/leaderboards":
		return enLeaderboardBody(d)
	case strings.HasPrefix(path, "/en/stations/"):
		return enStationBody(d, strings.TrimPrefix(path, "/en/stations/"))
	case path == "/en/guides":
		return enGuidesIndexBody()
	case strings.HasPrefix(path, "/en/guides/"):
		return enGuideBody(strings.TrimPrefix(path, "/en/guides/"))
	case path == "/en/models":
		return s.bodyEnModels(d)
	case strings.HasPrefix(path, "/en/models/"):
		return s.bodyEnModel(d, strings.TrimPrefix(path, "/en/models/"))
	case path == "/en":
		return s.bodyEnHome(d)
	case path == "/en/methodology":
		return enMethodologyBody()
	case path == "/en/faq":
		return enFaqBody()
	case path == "/sponsor":
		return sponsorBody()
	case path == "/api-docs":		return template.HTML(`<article><h1>公开 API</h1>
<p>只读 JSON 接口，按 IP 限流。榜单 <a href="/api/v1/stations">/api/v1/stations</a>，站点档案 <code>/api/v1/stations/{slug}</code>，模型 <code>/api/v1/models/{id}</code>，变更 <a href="/api/v1/changes">/api/v1/changes</a>，AI 摘要 <a href="/llms.txt">/llms.txt</a> 与 <a href="/llms-full.txt">/llms-full.txt</a>。</p></article>`)
	}
	return ""
}

func enGuidesIndexBody() template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Guides</h1><p>Long-form notes on choosing, billing and verifying LLM API relays.</p><ul>`)
	for _, g := range guideENList() {
		b.WriteString(`<li><a href="/en/guides/` + escHTML(g.Slug) + `">` + escHTML(g.Title) + `</a> — ` + escHTML(g.Desc) + `</li>`)
	}
	b.WriteString(`</ul><p><a href="/en">Rankings</a> · <a href="/guides">中文指南</a></p></article>`)
	return template.HTML(b.String())
}

func enGuideBody(slug string) template.HTML {
	g := guideENBySlug(slug)
	if g == nil {
		return ""
	}
	return template.HTML(`<article><h1>` + escHTML(g.Title) + `</h1><p>` + escHTML(g.Desc) + `</p>` + string(g.Body) + `</article>`)
}

func enAuditBody() template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Audit an OpenAI-compatible endpoint</h1>
<p>Send two real requests and measure first-token latency, stream integrity, model echo, usage consistency, tokenizer consistency and cache passthrough. The key is used only for this run and never stored.</p>
<h2>What it checks</h2><ul>`)
	for _, c := range auditCheckDocsEN {
		b.WriteString(`<li><a href="/en/audit/` + escHTML(c.Key) + `">` + escHTML(c.Label) + `</a> — ` + escHTML(c.Short) + `</li>`)
	}
	b.WriteString(`</ul><p>The audit does not judge whether a model is “dumbed down”; it only checks observable protocol and billing signals.</p>
<p><a href="/audit">Run the Chinese interface</a> · <a href="/en/methodology">Methodology</a></p></article>`)
	return template.HTML(b.String())
}

func enChangesBody(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Change feed</h1><p>New listings, purity alerts, availability drops, price moves and score regressions, derived from the same probes as the rankings.</p>`)
	rows := make([][]string, 0, 30)
	for i, c := range d.ChangesEN {
		if i >= 30 {
			break
		}
		rows = append(rows, []string{
			c.At.Format("2006-01-02 15:04"),
			`<a href="/en/stations/` + escHTML(c.StationSlug) + `">` + escHTML(c.StationName) + `</a>`,
			escHTML(c.Kind),
			escHTML(c.Title),
			escHTML(c.Detail),
		})
	}
	b.WriteString(string(tableHTML([]string{"Time", "Provider", "Kind", "Title", "Detail"}, rows)))
	b.WriteString(`<p><a href="/en">Rankings</a> · <a href="/changes">中文变更雷达</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyEnFree(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Free tiers and trial credits</h1><p>Official free tiers and trial credits only; we do not list shared or resold keys. Quotas change often — confirm on the official console.</p>`)
	rows := make([][]string, 0, 8)
	for _, m := range d.Models {
		if !m.FreeTier {
			continue
		}
		rows = append(rows, []string{
			`<a href="/en/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			fmt.Sprintf("$%.2f / $%.2f", m.InputUSD, m.OutputUSD),
			fmt.Sprintf("¥%.2f", m.MinInputCNY),
			escHTML(m.License),
		})
	}
	b.WriteString(string(tableHTML([]string{"Model", "Official /1M", "Cheapest input", "License"}, rows)))
	b.WriteString(`<h2>Official free programmes</h2><ul>
<li><a href="https://aistudio.google.com" rel="nofollow noopener">Google AI Studio</a> — Gemini free tier with RPM/RPD limits</li>
<li><a href="https://openrouter.ai/models?max_price=0" rel="nofollow noopener">OpenRouter Free</a> — models with a :free suffix</li>
<li><a href="https://console.groq.com" rel="nofollow noopener">Groq Console</a> — developer free tier</li>
<li><a href="https://platform.deepseek.com" rel="nofollow noopener">DeepSeek</a> — trial credits for new accounts</li>
<li><a href="https://bailian.console.aliyun.com" rel="nofollow noopener">Alibaba Model Studio</a> — daily Qwen quota</li>
<li><a href="https://platform.moonshot.cn" rel="nofollow noopener">Moonshot</a> — trial credits for new accounts</li>
</ul><p><a href="/free">中文免费额度</a></p></article>`)
	return template.HTML(b.String())
}

func enLeaderboardBody(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Relay leaderboard</h1><p>All providers ranked by the v2 score: 0.62 × availability + 0.26 × speed + 0.12 × consistency. Price is separate.</p>`)
	rows := make([][]string, 0, len(d.Stations))
	for _, st := range d.Stations {
		rows = append(rows, []string{
			fmt.Sprintf("%d", st.Rank),
			`<a href="/en/stations/` + escHTML(st.Slug) + `">` + escHTML(st.Name) + `</a>`,
			escHTML(st.Domain),
			fmt.Sprintf("%.1f ±%.1f", st.RankScore, st.ScoreMargin),
			fmt.Sprintf("%.1f%%", st.UptimePct),
			fmt.Sprintf("%.1f", st.Speed),
			fmt.Sprintf("%.1f", st.Consistency),
			fmt.Sprintf("¥%.2f", st.FromPriceCNY),
		})
	}
	b.WriteString(string(tableHTML([]string{"#", "Provider", "Domain", "Score", "Raw uptime", "Speed", "Consistency", "Min input"}, rows)))
	b.WriteString(`<p><a href="/en">Summary</a> · <a href="/en/methodology">Methodology</a> · <a href="/leaderboards">中文排行榜</a></p></article>`)
	return template.HTML(b.String())
}

func enStationBody(d *app.Dataset, slug string) template.HTML {
	st, ok := d.BySlug[slug]
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<article><h1>%s</h1><p>%s</p>`, escHTML(st.Name), escHTML(st.TaglineEN))
	fmt.Fprintf(&b, `<p><code>%s</code> · rank #%d of %d · score %.1f ±%.1f · cheapest ¥%.2f /1M</p>`,
		escHTML(st.Domain), st.Rank, len(d.Stations), st.RankScore, st.ScoreMargin, st.FromPriceCNY)
	b.WriteString(`<h2>Metrics (7-day window)</h2><table class="index-table"><tbody>`)
	fmt.Fprintf(&b, `<tr><td>Availability (raw)</td><td>%.1f%%</td></tr><tr><td>Speed score</td><td>%.1f</td></tr>`, st.UptimePct, st.Speed)
	fmt.Fprintf(&b, `<tr><td>Consistency</td><td>%.1f</td></tr><tr><td>Purity</td><td>%s</td></tr>`, st.Consistency, escHTML(tierLabel(st.PurityTier)))
	fmt.Fprintf(&b, `<tr><td>Last probe</td><td>%s</td></tr>`, st.LastProbeAt.Format("2006-01-02 15:04"))
	b.WriteString(`</tbody></table>`)

	b.WriteString(`<h2>Groups</h2>`)
	grows := make([][]string, 0, len(d.GroupsByStation[slug]))
	for _, g := range d.GroupsByStation[slug] {
		grows = append(grows, []string{
			escHTML(g.Name), fmt.Sprintf("%.2f×", g.Multiplier), fmt.Sprintf("%d ms", g.LatencyMS),
			fmt.Sprintf("%.1f%%", g.Availability), fmt.Sprintf("%d", g.Samples),
		})
	}
	b.WriteString(string(tableHTML([]string{"Group", "Multiplier", "Latency", "Availability", "Samples"}, grows)))

	b.WriteString(`<h2>Models and best prices</h2>`)
	mrows := make([][]string, 0, len(d.PerStationModels[slug]))
	for _, m := range d.PerStationModels[slug] {
		mrows = append(mrows, []string{
			`<a href="/en/models/` + escHTML(m.Model.ID) + `">` + escHTML(m.Model.Name) + `</a>`,
			fmt.Sprintf("¥%.2f", m.BestInputCNY), fmt.Sprintf("¥%.2f", m.BestOutputCNY), escHTML(m.CheapestGroup),
		})
	}
	b.WriteString(string(tableHTML([]string{"Model", "Input /1M", "Output /1M", "Cheapest group"}, mrows)))

	if pts := d.History[slug]; len(pts) > 1 {
		first, last := pts[0], pts[len(pts)-1]
		fmt.Fprintf(&b, `<h2>30-day trend</h2><p>%s → %s (%+.1f).</p>`, first.Date, last.Date, last.Score-first.Score)
	}
	b.WriteString(`<p><a href="/en/leaderboards">All providers</a> · <a href="/stations/` + escHTML(slug) + `">中文档案</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyEnModels(d *app.Dataset) template.HTML {
	rows := make([][]string, 0, len(d.Models))
	for _, m := range d.Models {
		rows = append(rows, []string{
			`<a href="/en/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			escHTML(m.VendorName),
			fmt.Sprintf("%d", m.ContextWindow),
			fmt.Sprintf("$%.2f / $%.2f", m.InputUSD, m.OutputUSD),
			fmt.Sprintf("¥%.2f", m.MinInputCNY),
			escHTML(licenseLabel(m.License)),
		})
	}
	return template.HTML(`<article><h1>Model directory and lowest relay prices</h1><p>Official USD prices next to the cheapest relay price in CNY.</p>` +
		string(tableHTML([]string{"Model", "Vendor", "Context", "Official /1M", "Cheapest input", "License"}, rows)) +
		`<p><a href="/en">Rankings</a> · <a href="/en/methodology">Methodology</a> · <a href="/models">中文模型目录</a></p></article>`)
}

func (s *Server) bodyEnModel(d *app.Dataset, id string) template.HTML {
	m, ok := d.ModelByID[id]
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<article><h1>%s</h1><p>%s</p>`, escHTML(m.Name), escHTML(m.SummaryEN))
	fmt.Fprintf(&b, `<p>Official $%.2f / $%.2f per 1M · context %d · license %s · coding %.1f · reasoning %.1f · cheapest ¥%.2f</p>`,
		m.InputUSD, m.OutputUSD, m.ContextWindow, escHTML(licenseLabel(m.License)), m.CodingScore, m.ReasoningScore, m.MinInputCNY)
	b.WriteString(`<h2>Relay offers</h2>`)
	rows := make([][]string, 0, 12)
	for i, o := range d.Offers[id] {
		if i >= 12 {
			break
		}
		rows = append(rows, []string{
			`<a href="/stations/` + escHTML(o.StationSlug) + `">` + escHTML(o.StationName) + `</a>`,
			escHTML(o.GroupName),
			fmt.Sprintf("%.2f×", o.Multiplier),
			fmt.Sprintf("¥%.2f", o.InputCNY),
			fmt.Sprintf("¥%.2f", o.OutputCNY),
			escHTML(tierLabel(o.PurityTier)),
		})
	}
	b.WriteString(string(tableHTML([]string{"Provider", "Group", "Multiplier", "Input/1M", "Output/1M", "Purity"}, rows)))
	b.WriteString(`<p><a href="/en/models">All models</a> · <a href="/models/` + escHTML(m.ID) + `">中文页</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyEnHome(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>Independent rankings for LLM API relay providers</h1>`)
	fmt.Fprintf(&b, `<p>%d providers and %d groups, measured by continuous probes. Score = 0.62 × availability + 0.26 × speed + 0.12 × consistency; price and purity are shown separately and never change the score.</p>`, len(d.Stations), countGroups(d))
	b.WriteString(`<h2>Current ranking</h2>`)
	rows := make([][]string, 0, 10)
	for i, st := range d.Stations {
		if i >= 10 {
			break
		}
		price := "—"
		if st.FromPriceCNY >= 0 {
			price = fmt.Sprintf("¥%.2f", st.FromPriceCNY)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", st.Rank),
			`<a href="/stations/` + escHTML(st.Slug) + `">` + escHTML(st.Name) + `</a>`,
			escHTML(st.Domain),
			fmt.Sprintf("%.1f ±%.1f", st.RankScore, st.ScoreMargin),
			fmt.Sprintf("%.1f%%", st.UptimePct),
			price,
			escHTML(tierLabel(st.PurityTier)),
		})
	}
	b.WriteString(string(tableHTML([]string{"#", "Provider", "Domain", "Score", "Raw uptime", "Min input/1M", "Purity"}, rows)))
	b.WriteString(`<p>Provider names are shown as published (often Chinese); domains are the stable identifier.</p>`)
	b.WriteString(`<p><a href="/en/methodology">Scoring methodology</a> · <a href="/en/faq">FAQ</a> · <a href="/methodology">中文方法论</a> · <a href="/">中文站</a></p></article>`)
	return template.HTML(b.String())
}

func enMethodologyBody() template.HTML {
	return template.HTML(`<article><h1>Scoring methodology v2</h1>
<p>Every score is reproducible from the probe data; no editorial weight is applied.</p>
<h2>Per group</h2>
<pre>A = 100 × Wilson95(k, n)          # availability, 95% lower bound
S = 100 × clamp((5000 − p50) / 4200)   # median latency: 800ms → 100, 5000ms → 0
C = 100 × clamp(1 − jitter / 2)        # jitter = (p95 − p50) / max(p50, 150)
Q = 0.62·A + 0.26·S + 0.12·C</pre>
<p>A group needs at least 24 probes in the 7-day window to be scored. The station axis uses 80% coverage-weighted mean plus 20% worst group, so one bad group cannot be hidden by many good ones.</p>
<h2>Uncertainty</h2>
<p>The ± value next to a station score is the half-width of the Wilson 95% interval for availability, multiplied by 0.62 (its weight). It only covers availability sampling error; speed and consistency are point estimates. When the gap between #1 and #2 is smaller than the combined margin, we say they are within error.</p>
<h2>Purity and price</h2>
<p>Purity checks only cover Claude groups (thinking signature and behaviour fingerprint) and are shown as a label, not part of the score. Price is official USD × settlement rate × group multiplier; it is shown separately.</p>
<h2>Limitations</h2>
<p>Probes are short and cheap: they measure reachability and latency, not long-output throughput. A single probe window cannot prove a provider never degrades. Signature checks prove presence, not absolute authenticity.</p>
<p><a href="/en">Back to rankings</a> · <a href="/en/faq">FAQ</a></p></article>`)
}

func enFaqBody() template.HTML {
	return template.HTML(`<article><h1>Frequently asked questions</h1>
<h2>What is an LLM API relay?</h2><p>A relay aggregates several official APIs behind one domain and one key, billing through its own exchange rate and group multiplier. It removes multi-account and payment friction, but routing and billing are controlled by the provider, so independent monitoring matters.</p>
<h2>How do you detect a watered-down relay?</h2><p>Two families of signals: protocol and billing (does the response echo the requested model, are usage tokens reported and stable across identical requests, is prompt caching passed through), and sampling (is the Claude thinking signature present and consistent). The audit covers the first; purity checks cover the second.</p>
<h2>Why is price not part of the score?</h2><p>Cheap can mean worse upstreams, lower availability or no refunds. Folding price into the score would systematically punish expensive-but-stable providers. Price is a separate column for readers to weigh.</p>
<h2>What does ± mean?</h2><p>It is the availability sampling error (Wilson 95% half-width × 0.62). Speed and consistency are point estimates; we do not invent an overall interval.</p>
<h2>How often is data updated?</h2><p>Probes run continuously; scoring uses a 7-day window. A group needs 24 probes to be scored, and a station needs at least half of its groups scored to be ranked.</p>
<h2>Can rankings be bought?</h2><p>No. Sponsorships are labelled and never change scores, ordering or audit results.</p>
<h2>Is the data real?</h2><p>This repository ships a demo dataset: provider names and domains are public, but multipliers, probes, scores and reviews are generated. It is not an assessment of the real providers.</p>
<h2>What does the audit not do?</h2><p>It does not judge whether a model is "dumbed down"; it only checks observable protocol and billing signals. Behavioural fingerprinting and signature cross-verification remain on the roadmap.</p>
<p><a href="/en">Back to rankings</a> · <a href="/en/methodology">Methodology</a> · <a href="/faq">中文 FAQ</a></p></article>`)
}

func (s *Server) bodyHome(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>大模型 API 中转站榜单</h1>`)
	fmt.Fprintf(&b, `<p>覆盖 %d 家站点、%d 个分组，数据时间 %s。综合分由可用性、速度与稳定性三轴加权，价格与纯度单独展示。</p>`,
		len(d.Stations), countGroups(d), d.TakenAt.Format("2006-01-02 15:04 MST"))
	b.WriteString(`<p>新手先看 <a href="/guides/how-to-choose-relay">10 分钟选站清单</a>，计费口径见 <a href="/guides/multiplier-math">倍率与缓存</a>，验真方法见 <a href="/guides/signature-verification">thinking 签名判断</a>；常见问题在 <a href="/faq">FAQ</a>。</p>`)
	b.WriteString(`<h2>当前排名</h2>`)

	rows := make([][]string, 0, 12)
	for i, st := range d.Stations {
		if i >= 12 {
			break
		}
		price := "—"
		if st.FromPriceCNY >= 0 {
			price = fmt.Sprintf("¥%.2f", st.FromPriceCNY)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", st.Rank),
			stationLink(st),
			escHTML(st.Domain),
			fmt.Sprintf("%.1f", st.RankScore),
			fmt.Sprintf("%.1f%%", st.UptimePct),
			price,
			escHTML(tierLabel(st.PurityTier)),
		})
	}
	b.WriteString(string(tableHTML([]string{"#", "站点", "域名", "综合分", "原始可用率", "最低输入价/1M", "纯度"}, rows)))
	b.WriteString(`<p><a href="/methodology">评分方法</a> · <a href="/changes">变更雷达</a> · <a href="/leaderboards">排行榜</a> · <a href="/faq">常见问题</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyLeaderboards(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>排行榜：模型能力与站点稳定性</h1><p>能力分来自官方公开基准，速度与稳定性来自连续探针。</p>`)

	if len(d.Stations) > 0 {
		best, up, fast := d.Stations[0], d.Stations[0], d.Stations[0]
		var cheap *app.StationView
		for i := range d.Stations {
			st := &d.Stations[i]
			if st.RankScore > best.RankScore {
				best = *st
			}
			if st.UptimePct > up.UptimePct {
				up = *st
			}
			if st.Speed > fast.Speed {
				fast = *st
			}
			if st.FromPriceCNY >= 0 && st.PurityTier != "counterfeit" && (cheap == nil || st.FromPriceCNY < cheap.FromPriceCNY) {
				cheap = st
			}
		}
		tieNote := ""
		if len(d.Stations) > 1 {
			second := d.Stations[1]
			if best.RankScore-second.RankScore <= best.ScoreMargin+second.ScoreMargin {
				tieNote = fmt.Sprintf("（与第 2 名 %s 的差距在误差范围内）", escHTML(second.Name))
			}
		}
		fmt.Fprintf(&b, `<p><b>当前答案：</b>%d 家站点里，<a href="/stations/%s">%s</a> 综合分 %.1f（±%.1f，95%% 置信，仅统计可用性抽样误差）排第一%s；原始可用率最高的是 <a href="/stations/%s">%s</a>（%.1f%%），速度最快的是 <a href="/stations/%s">%s</a>（%.0f 分）`,
			len(d.Stations), best.Slug, escHTML(best.Name), best.RankScore, best.ScoreMargin, tieNote,
			up.Slug, escHTML(up.Name), up.UptimePct, fast.Slug, escHTML(fast.Name), fast.Speed)
		if cheap != nil {
			fmt.Fprintf(&b, `，最低输入价 ¥%.2f/1M 来自 <a href="/stations/%s">%s</a>`, cheap.FromPriceCNY, cheap.Slug, escHTML(cheap.Name))
		}
		b.WriteString(`。价格不参与综合分，排名随探针更新变化。</p>`)
	}

	models := append([]app.ModelView(nil), d.Models...)
	sort.SliceStable(models, func(i, j int) bool { return models[i].CodingScore > models[j].CodingScore })
	b.WriteString(`<h2>编程能力</h2>`)
	rows := make([][]string, 0, 10)
	for i, m := range models {
		if i >= 10 {
			break
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			`<a href="/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			escHTML(m.VendorName),
			fmt.Sprintf("%.1f", m.CodingScore),
			fmt.Sprintf("%.1f", m.ReasoningScore),
			fmt.Sprintf("¥%.2f", m.MinInputCNY),
		})
	}
	b.WriteString(string(tableHTML([]string{"#", "模型", "厂商", "编程分", "推理分", "最低输入价"}, rows)))

	bySpeed := append([]app.StationView(nil), d.Stations...)
	sort.SliceStable(bySpeed, func(i, j int) bool { return bySpeed[i].Speed > bySpeed[j].Speed })
	b.WriteString(`<h2>站点速度</h2>`)
	rows = rows[:0]
	for i, st := range bySpeed {
		if i >= 10 {
			break
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1), stationLink(st),
			fmt.Sprintf("%.0f", st.Speed), fmt.Sprintf("%.1f%%", st.UptimePct),
		})
	}
	b.WriteString(string(tableHTML([]string{"#", "站点", "速度分", "原始可用率"}, rows)))
	b.WriteString(`</article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyModels(d *app.Dataset) template.HTML {
	rows := make([][]string, 0, len(d.Models))
	for _, m := range d.Models {
		ctx, official := "—", "—"
		if m.Category == "" || m.Category == "text" {
			ctx = fmtContextWindow(m.ContextWindow)
			if m.InputUSD > 0 {
				official = fmt.Sprintf("$%.2f / $%.2f", m.InputUSD, m.OutputUSD)
			}
		}
		released := "—"
		if !m.ReleasedAt.IsZero() {
			released = m.ReleasedAt.Format("2006-01")
		}
		price := "—"
		if m.MinInputCNY >= 0 {
			price = fmt.Sprintf("¥%.2f", m.MinInputCNY)
		}
		rows = append(rows, []string{
			`<a href="/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			escHTML(displayVendor(m.Vendor, m.VendorName)),
			escHTML(ctx),
			released,
			escHTML(official),
			price,
			fmt.Sprintf("%d", m.OffersCount),
		})
	}
	return template.HTML(`<article><h1>模型与报价</h1><p>按厂商、上下文与发布时间浏览模型；官方美元价与站点折算人民币价并列，并标注报价站点数。金额单位：官方价为美元 / 每百万 Token，最低价为人民币（文本按每百万 Token，图片按张，视频按秒）。</p>` +
		string(tableHTML([]string{"模型", "厂商", "上下文", "发布", "官方价", "最低价", "报价站点"}, rows)) + `</article>`)
}

// displayVendor turns the catalog's vendor slug into a readable name; the
// upstream catalog leaves a few vendors ("meta", "~z-ai") without a proper one.
func displayVendor(vendor, name string) string {
	base := strings.TrimPrefix(vendor, "~")
	if label, ok := vendorLabels[base]; ok {
		return label
	}
	if name != "" && name != vendor {
		return name
	}
	return base
}

var vendorLabels = map[string]string{
	"openai":     "OpenAI",
	"anthropic":  "Anthropic",
	"google":     "Google",
	"deepseek":   "DeepSeek",
	"qwen":       "阿里通义",
	"z-ai":       "智谱",
	"moonshotai": "Moonshot",
	"minimax":    "MiniMax",
	"x-ai":       "xAI",
	"meta":       "Meta",
	"meta-llama": "Meta",
	"mistralai":  "Mistral",
	"tencent":    "腾讯",
}

// fmtContextWindow renders a token window the way the models table does.
func fmtContextWindow(n int) string {
	switch {
	case n <= 0:
		return "—"
	case n >= 1000000:
		if n%1000000 == 0 {
			return fmt.Sprintf("%dM", n/1000000)
		}
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	default:
		return fmt.Sprintf("%dK", n/1000)
	}
}

func bodyStationsDirectory(d *app.Dataset) template.HTML {
	rows := make([][]string, 0, len(d.Stations))
	for _, st := range d.Stations {
		if st.OfficialVendor {
			continue
		}
		services := strings.Join(st.Services, " / ")
		payments := strings.Join(st.Payments, " / ")
		if payments == "" {
			payments = "—"
		}
		refund := st.RefundPolicy
		if refund == "" {
			refund = "—"
		}
		invoice := "不支持"
		if st.Invoice {
			invoice = "支持"
		}
		rating := "暂无评价"
		if st.ReviewsCount > 0 {
			rating = fmt.Sprintf("%.1f / 5 · %d 条", st.RatingAvg, st.ReviewsCount)
		}
		rows = append(rows, []string{
			`<a href="/stations/` + escHTML(st.Slug) + `">` + escHTML(st.Name) + `</a><br><span class="mono">` + escHTML(st.Domain) + `</span>`,
			escHTML(services) + fmt.Sprintf("<br>%d 个模型", st.ModelsCount),
			fmt.Sprintf("%.1f%%", st.UptimePct),
			escHTML(payments),
			escHTML(refund),
			invoice,
			rating,
		})
	}
	return template.HTML(`<article><h1>中转站目录</h1><p>按模型覆盖、健康状态、付款方式、退款、发票与用户评价浏览；实时评分与价格见<a href="/">首页榜单</a>。</p>` +
		string(tableHTML([]string{"站点", "模型覆盖", "健康状态", "付款方式", "退款", "发票", "用户评价"}, rows)) + `</article>`)
}

func (s *Server) bodyStation(d *app.Dataset, slug string) template.HTML {
	st, ok := d.BySlug[slug]
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<article><h1>%s</h1><p>%s</p>`, escHTML(st.Name), escHTML(st.Tagline))
	fmt.Fprintf(&b, `<p>综合分 %.1f（第 %d 名）· 原始可用率 %.1f%% · 速度分 %.1f · 稳定性分 %.1f · 纯度 %s</p>`,
		st.RankScore, st.Rank, st.UptimePct, st.Speed, st.Consistency, tierLabel(st.PurityTier))

	if hist := d.History[slug]; len(hist) > 0 {
		b.WriteString(`<h2>近 7 天综合分</h2>`)
		rows := make([][]string, 0, 7)
		start := len(hist) - 7
		if start < 0 {
			start = 0
		}
		for _, p := range hist[start:] {
			rows = append(rows, []string{
				escHTML(p.Date),
				fmt.Sprintf("%.1f", p.Score),
				fmt.Sprintf("%.1f", p.Availability),
				fmt.Sprintf("%.1f", p.Speed),
				fmt.Sprintf("%.1f", p.Consistency),
				fmt.Sprintf("%.1f%%", p.Uptime),
			})
		}
		b.WriteString(string(tableHTML([]string{"日期", "综合分", "可用性分", "速度分", "稳定性分", "当日可用率"}, rows)))
	}

	b.WriteString(`<h2>分组与线路</h2>`)
	rows := make([][]string, 0)
	for _, g := range d.GroupsByStation[slug] {
		rows = append(rows, []string{
			escHTML(g.Name), escHTML(g.Service),
			fmt.Sprintf("%.2f×", g.Multiplier),
			fmt.Sprintf("%d ms", g.LatencyMS),
			fmt.Sprintf("%.1f", g.Availability),
			fmt.Sprintf("%.1f", g.Speed),
			fmt.Sprintf("%.1f", g.Consistency),
			escHTML(tierLabel(g.PurityTier)),
		})
	}
	b.WriteString(string(tableHTML([]string{"分组", "服务", "倍率", "延迟", "可用性分", "速度分", "稳定性分", "纯度"}, rows)))

	b.WriteString(`<h2>支持模型与人民币实价</h2>`)
	rows = rows[:0]
	for _, sm := range d.PerStationModels[slug] {
		rows = append(rows, []string{
			`<a href="/models/` + escHTML(sm.Model.ID) + `">` + escHTML(sm.Model.Name) + `</a>`,
			escHTML(sm.Model.VendorName),
			fmt.Sprintf("¥%.2f", sm.BestInputCNY),
			fmt.Sprintf("¥%.2f", sm.BestOutputCNY),
			escHTML(sm.CheapestGroup),
		})
	}
	b.WriteString(string(tableHTML([]string{"模型", "厂商", "输入/1M", "输出/1M", "最低价分组"}, rows)))
	fmt.Fprintf(&b, `<p>最低充值 ¥%.0f · 支付方式 %s · 退款政策 %s</p>`,
		st.MinTopup, escHTML(strings.Join(st.Payments, "、")), escHTML(st.RefundPolicy))
	b.WriteString(`<p><a href="/stations/` + escHTML(st.Slug) + `">查看交互版档案</a> · <a href="/md/station/` + escHTML(st.Slug) + `">Markdown 版本</a> · <a href="/audit">检测该站接口</a></p></article>`)
	return template.HTML(b.String())
}

func licenseLabel(slug string) string {
	switch slug {
	case "mit":
		return "MIT"
	case "apache-2.0":
		return "Apache-2.0"
	case "modified-mit":
		return "Modified MIT"
	case "proprietary":
		return "闭源 / 专有"
	}
	return slug
}

func (s *Server) bodyModelLicense(d *app.Dataset, slug string) template.HTML {
	type entry struct {
		slug  string
		count int
	}
	counts := map[string]int{}
	for _, m := range d.Models {
		counts[m.License]++
	}
	var b strings.Builder
	if slug == "" {
		b.WriteString(`<article><h1>按许可证浏览模型</h1><p>许可证决定权重能否自托管、能否商用。</p><ul>`)
		for _, key := range []string{"proprietary", "mit", "apache-2.0", "modified-mit"} {
			if counts[key] == 0 {
				continue
			}
			fmt.Fprintf(&b, `<li><a href="/models/license/%s">%s</a>（%d 个模型）</li>`, escHTML(key), escHTML(licenseLabel(key)), counts[key])
		}
		b.WriteString(`</ul></article>`)
		return template.HTML(b.String())
	}

	fmt.Fprintf(&b, `<article><h1>%s 许可证模型</h1><p>以下模型使用 %s 许可证。</p>`, escHTML(licenseLabel(slug)), escHTML(licenseLabel(slug)))
	rows := make([][]string, 0)
	for _, m := range d.Models {
		if m.License != slug {
			continue
		}
		rows = append(rows, []string{
			`<a href="/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			escHTML(m.VendorName),
			fmt.Sprintf("$%.2f / $%.2f", m.InputUSD, m.OutputUSD),
			fmt.Sprintf("¥%.2f", m.MinInputCNY),
		})
	}
	b.WriteString(string(tableHTML([]string{"模型", "厂商", "官方价/1M", "全网最低输入"}, rows)))
	b.WriteString(`<p><a href="/models/license">全部许可证</a> · <a href="/models">模型目录</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyModel(d *app.Dataset, id string) template.HTML {
	m, ok := d.ModelByID[id]
	if !ok {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<article><h1>%s</h1><p>%s</p>`, escHTML(m.Name), escHTML(zhModelIntro(m)))

	b.WriteString(`<h2>关键数据</h2><ul>`)
	fmt.Fprintf(&b, `<li>厂商：%s · 分类：%s · 上下文：%s · 许可证：%s</li>`,
		escHTML(m.VendorName), escHTML(categoryLabel(m.Category)), escHTML(fmtContextWindow(m.ContextWindow)), escHTML(licenseLabel(m.License)))
	if m.Category == "" || m.Category == "text" {
		fmt.Fprintf(&b, `<li>官方价：%s / %s 每百万 Token；全网最低：%s（%s）；报价站点：%d 个</li>`,
			moneyUSD(m.InputUSD), moneyUSD(m.OutputUSD), moneyCNY(m.MinInputCNY), escHTML(m.BestStation), m.OffersCount)
	} else {
		fmt.Fprintf(&b, `<li>计费：按%s；最低价：%s（%s）；报价站点：%d 个</li>`,
			unitLabel(m.Category), moneyCNY(m.MinInputCNY), escHTML(m.BestStation), m.OffersCount)
	}
	fmt.Fprintf(&b, `<li>供给评分：%s / 100（价格 %s · 覆盖 %s · 可用性 %s · 纯度 %s · 上下文 %s）</li></ul>`,
		scoreText(m.RelayScore), scoreText(m.PriceScore), scoreText(m.CoverageScore),
		scoreText(m.AvailabilityScore), scoreText(m.PurityScore), scoreText(m.ContextScore))

	b.WriteString(`<h2>模型评分</h2><p>评分只衡量“现在买这个模型有多划算、供给有多稳”，不评价模型能力；能力需要公开基准，本站没有自测。</p>`)
	scoreRows := [][]string{
		{"价格", scoreText(m.PriceScore), priceScoreNote(m)},
		{"覆盖", scoreText(m.CoverageScore), fmt.Sprintf("%d 个站点有报价", m.OffersCount)},
		{"可用性", scoreText(m.AvailabilityScore), fmt.Sprintf("报价站点平均可用性 %.1f", m.AvgAvailability)},
		{"纯度", scoreText(m.PurityScore), fmt.Sprintf("官方直连 %d · 混合 %d · 未检测 %d · 疑似掺假 %d", m.PurityOfficial, m.PurityMixed, m.PurityUnknown, m.PurityCounterfeit)},
		{"上下文", scoreText(m.ContextScore), fmt.Sprintf("窗口 %s", fmtContextWindow(m.ContextWindow))},
	}
	b.WriteString(string(tableHTML([]string{"维度", "分数", "依据"}, scoreRows)))

	b.WriteString(`<h2>价格与供给</h2><ul>`)
	if m.PriceLowCNY > 0 {
		fmt.Fprintf(&b, `<li>实价区间：%s – %s（中位 %s），共 %d 个报价</li>`, moneyCNY(m.PriceLowCNY), moneyCNY(m.PriceHighCNY), moneyCNY(m.PriceMedianCNY), m.OffersCount)
	} else {
		b.WriteString(`<li>暂无中转站报价，以下为官方价参考。</li>`)
	}
	if m.PriceRatio > 0 {
		fmt.Fprintf(&b, `<li>最低价 ÷ 中位价：%.2f×（越低说明越值得比价）</li>`, m.PriceRatio)
	}
	if m.Category == "" || m.Category == "text" {
		fmt.Fprintf(&b, `<li>上下文窗口：%s；发布日期：%s</li>`, fmtContextWindow(m.ContextWindow), dateLabel(m.ReleasedAt))
	}
	b.WriteString(`</ul>`)

	b.WriteString(`<h2>全网报价</h2>`)
	rows := make([][]string, 0, 12)
	for i, o := range d.Offers[id] {
		if i >= 12 {
			break
		}
		rows = append(rows, []string{
			`<a href="/stations/` + escHTML(o.StationSlug) + `">` + escHTML(o.StationName) + `</a>`,
			escHTML(o.GroupName),
			fmt.Sprintf("%.2f×", o.Multiplier),
			fmt.Sprintf("¥%.2f", o.InputCNY),
			fmt.Sprintf("¥%.2f", o.OutputCNY),
			escHTML(tierLabel(o.PurityTier)),
		})
	}
	b.WriteString(string(tableHTML([]string{"站点", "分组", "倍率", "输入/1M", "输出/1M", "纯度"}, rows)))

	if changes := d.PriceChanges[m.ID]; len(changes) > 0 {
		b.WriteString(`<h2>价格变动记录</h2>`)
		rows := make([][]string, 0, len(changes))
		for _, c := range changes {
			rows = append(rows, []string{
				`<a href="/stations/` + escHTML(c.StationSlug) + `">` + escHTML(c.StationName) + `</a>`,
				escHTML(c.GroupName),
				fmt.Sprintf("¥%.2f → ¥%.2f", c.OldInputCNY, c.NewInputCNY),
				fmt.Sprintf("%+.1f%%", c.DeltaPct),
				escHTML(c.At.Format("2006-01-02")),
			})
		}
		b.WriteString(string(tableHTML([]string{"站点", "分组", "输入价", "变动", "记录时间"}, rows)))
	}

	// Same-vendor models give crawlers and readers a way to compare.
	related := make([]string, 0, 6)
	for _, other := range d.Models {
		if other.ID == m.ID || other.Vendor != m.Vendor {
			continue
		}
		related = append(related, `<li><a href="/models/`+escHTML(other.ID)+`">`+escHTML(other.Name)+`</a></li>`)
		if len(related) >= 6 {
			break
		}
	}
	if len(related) > 0 {
		fmt.Fprintf(&b, `<h2>同厂商模型</h2><ul>%s</ul>`, strings.Join(related, ""))
	}

	b.WriteString(`<h2>常见问题</h2>`)
	for _, q := range modelFAQ(m) {
		fmt.Fprintf(&b, `<h3>%s</h3><p>%s</p>`, escHTML(q.Q), escHTML(q.A))
	}

	b.WriteString(`<p><a href="/models/` + escHTML(m.ID) + `">查看交互版</a> · <a href="/md/model/` + escHTML(m.ID) + `">Markdown 版本</a> · <a href="/audit?url=">检测该模型所在接口</a></p></article>`)
	return template.HTML(b.String())
}

// categoryLabel renders a model category for humans.
func categoryLabel(c string) string {
	switch c {
	case "image":
		return "图片生成"
	case "video":
		return "视频生成"
	default:
		return "文本"
	}
}

func unitLabel(c string) string {
	if c == "image" {
		return "张"
	}
	if c == "video" {
		return "秒"
	}
	return "百万 Token"
}

func moneyUSD(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("$%.2f", v)
}

func moneyCNY(v float64) string {
	if v < 0 {
		return "—"
	}
	return fmt.Sprintf("¥%.2f", v)
}

func scoreText(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f", v)
}

func dateLabel(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02")
}

func priceScoreNote(m *app.ModelView) string {
	if m.PriceLowCNY <= 0 || m.PriceMedianCNY <= 0 {
		return "缺少报价，暂不计分"
	}
	return fmt.Sprintf("最低 %s，中位 %s（%.2f×）", moneyCNY(m.PriceLowCNY), moneyCNY(m.PriceMedianCNY), m.PriceRatio)
}

// modelFAQ is shared by the prerendered page and the FAQPage JSON-LD.
type faqEntry struct {
	Q string
	A string
}

func modelFAQ(m *app.ModelView) []faqEntry {
	out := make([]faqEntry, 0, 4)
	if m.Category == "" || m.Category == "text" {
		out = append(out, faqEntry{
			Q: fmt.Sprintf("%s 的官方价是多少？", m.Name),
			A: fmt.Sprintf("官方标价 %s 输入 / %s 输出 每百万 Token（%s）。中转站实价通常是官方美元价乘以站点自有汇率与分组倍率，所以同一模型在不同站点会差出数倍。", moneyUSD(m.InputUSD), moneyUSD(m.OutputUSD), m.VendorName),
		})
	} else {
		out = append(out, faqEntry{
			Q: fmt.Sprintf("%s 怎么计费？", m.Name),
			A: fmt.Sprintf("%s 属于%s，按%s计费；中转站会在此基础上叠加倍率，页面里的“最低价”已折算成人民币。", m.Name, categoryLabel(m.Category), unitLabel(m.Category)),
		})
	}
	if m.PriceLowCNY > 0 {
		out = append(out, faqEntry{
			Q: fmt.Sprintf("%s 全网最低价是多少，哪家站点？", m.Name),
			A: fmt.Sprintf("目前收录 %d 个报价，最低 %s（%s）；价格区间 %s – %s。最低价不代表最稳，需要同时看该站点的可用性与纯度。", m.OffersCount, moneyCNY(m.PriceLowCNY), m.BestStation, moneyCNY(m.PriceLowCNY), moneyCNY(m.PriceHighCNY)),
		})
	} else {
		out = append(out, faqEntry{
			Q: fmt.Sprintf("为什么 %s 没有中转站报价？", m.Name),
			A: "目前没有收录到该模型的中转站报价，页面上只展示官方价；一旦有站点上架并被同步，这里会自动出现实价。",
		})
	}
	out = append(out, faqEntry{
		Q: fmt.Sprintf("怎么判断中转站的 %s 有没有被替换或降智？", m.Name),
		A: "用本站的接口检测对该站点发两次相同请求，核对模型回显、usage、分词一致性、缓存透传与首字延迟；这些是可观测信号，检测不会直接给“降智”下结论。",
	})
	out = append(out, faqEntry{
		Q: fmt.Sprintf("%s 支持多长上下文？", m.Name),
		A: fmt.Sprintf("目录记录为 %s。实际可用长度取决于站点是否透传长上下文，建议用接口检测或长文档测试验证。", fmtContextWindow(m.ContextWindow)),
	})
	if m.Summary != "" {
		out = append(out, faqEntry{
			Q: fmt.Sprintf("官方怎么介绍 %s？", m.Name),
			A: m.Summary + "（英文原文）",
		})
	}
	return out
}

// zhModelIntro is the one-line Chinese description shown at the top of the
// model page; the catalog summary is usually English.
func zhModelIntro(m *app.ModelView) string {
	price := "—"
	if m.InputUSD > 0 {
		price = moneyUSD(m.InputUSD) + " / " + moneyUSD(m.OutputUSD) + " 每百万 Token"
	} else if m.Category != "" && m.Category != "text" {
		price = "按" + unitLabel(m.Category) + "计费"
	}
	ctx := ""
	if m.Category == "" || m.Category == "text" {
		ctx = "，上下文 " + fmtContextWindow(m.ContextWindow)
	}
	return fmt.Sprintf("%s 的%s模型%s；官方价 %s，%d 个站点有报价。", m.VendorName, categoryLabel(m.Category), ctx, price, m.OffersCount)
}

func (s *Server) bodyChanges(d *app.Dataset) template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>变更雷达</h1><p>自动检测价格调整、可用率骤降、纯度异常与新站上架。</p><ul>`)
	for i, c := range d.Changes {
		if i >= 20 {
			break
		}
		fmt.Fprintf(&b, `<li><time datetime="%s">%s</time> <a href="/stations/%s">%s</a> — %s：%s</li>`,
			c.At.Format(time.RFC3339), c.At.Format("2006-01-02"),
			escHTML(c.StationSlug), escHTML(c.StationName), escHTML(c.Title), escHTML(c.Detail))
	}
	b.WriteString(`</ul><p><a href="/changes.xml">RSS 订阅</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) bodyFree(d *app.Dataset) template.HTML {
	rows := make([][]string, 0)
	for _, m := range d.Models {
		if !m.FreeTier {
			continue
		}
		rows = append(rows, []string{
			`<a href="/models/` + escHTML(m.ID) + `">` + escHTML(m.Name) + `</a>`,
			escHTML(m.FreeNote),
			fmt.Sprintf("$%.2f / $%.2f", m.InputUSD, m.OutputUSD),
			fmt.Sprintf("¥%.2f", m.MinInputCNY),
		})
	}
	return template.HTML(`<article><h1>免费额度目录</h1><p>官方免费层与赠送额度，附限速与绑卡要求；只收录官方入口。</p>` +
		string(tableHTML([]string{"模型", "免费额度", "官方价/1M", "中转最低"}, rows)) +
		`<h2>官方免费计划</h2><ul>
<li><a href="https://aistudio.google.com" rel="nofollow noopener">Google AI Studio</a> — Gemini 免费层，按 RPM/RPD 限速</li>
<li><a href="https://openrouter.ai/models?max_price=0" rel="nofollow noopener">OpenRouter Free</a> — 带 :free 后缀的模型，有限速</li>
<li><a href="https://console.groq.com" rel="nofollow noopener">Groq Console</a> — 开发者免费层</li>
<li><a href="https://platform.deepseek.com" rel="nofollow noopener">DeepSeek 开放平台</a> — 新账号赠送额度</li>
<li><a href="https://bailian.console.aliyun.com" rel="nofollow noopener">阿里云百炼</a> — Qwen 每日免费额度</li>
<li><a href="https://platform.moonshot.cn" rel="nofollow noopener">Moonshot 开放平台</a> — 新账号试用额度</li>
</ul><p>额度与限速以官方页面为准；不提供共享 Key。</p></article>`)
}

func countGroups(d *app.Dataset) int {
	n := 0
	for _, st := range d.Stations {
		n += st.GroupsCount
	}
	return n
}

func tierLabel(t string) string {
	switch t {
	case "official":
		return "官方直连"
	case "mixed":
		return "混合线路"
	case "counterfeit":
		return "疑似掺假"
	}
	return "未检测"
}

// ---------------------------------------------------------------------------
// FAQ
// ---------------------------------------------------------------------------

var faqItems = []struct{ Q, A string }{
	{"大模型 API 中转站是什么？和官方 API 有什么区别？",
		"中转站把多家官方 API 聚合到一个域名下，用同一个 Key 调用不同模型，并按自己的汇率与分组倍率计费。它省去了多平台开户和支付的麻烦，但计费口径、路由与上游质量由站点控制，因此需要独立监测。"},
	{"怎么判断一家中转站有没有“掺水”？",
		"看两类信号：一是协议与计费，例如响应是否回显请求的模型、usage 是否上报、两次相同请求的 token 是否一致、缓存命中是否透传；二是抽检，例如 Claude 的 thinking 签名是否完整。我们的接口检测一次完成前一类，纯度抽检完成后一类。"},
	{"为什么综合分不包含价格？",
		"便宜可能意味着更差的上游、更低的可用性或更少的退款保障，把价格塞进分数会让“贵但稳”的站点被系统性压低。价格单独成列，由读者按自己的预算权衡。"},
	{"可用性分和页面上的原始可用率有什么区别？",
		"原始可用率是成功探针占比；可用性分取 Wilson 95% 置信下界，样本越少越拿不到满分（84/84 次成功得 95.6 分，100/100 得 96.3 分）。排名用后者，展示用前者。"},
	{"数据多久更新一次？样本量多少？",
		"探针按固定节奏写入，评分窗口为最近 7 天；每个分组至少 24 条探针才计分，站点至少一半分组计分才进入榜单。"},
	{"纯度抽检是怎么做的？",
		"只对 Claude 分组做思考签名与行为指纹抽检，约 3 天一次，取 7 天中位数；得分 ≥80 记为官方直连，30–79 为混合线路，低于 30 为疑似掺假。纯度不参与排名。"},
	{"榜单排名可以购买吗？",
		"不可以。排序只使用探针与抽检数据，赞助位、商务合作与广告不改变分数，也不会伪装成自然推荐。"},
	{"充值前应该注意什么？",
		"先用小额充值（¥5–¥20）和临时 Key 验证签名、倍率与缓存，再投入生产；优先选择支持按量退款、能开发票、域名运营时间较长的站点。"},
	{"接口检测会上传我的 Key 吗？",
		"Key 只用于本次探测，不写日志、不落库，报告只保存端点、模型与测量结果。生产环境请始终使用可随时吊销的临时 Key。"},
	{"为什么有些站点显示“未检测”？",
		"纯度抽检只覆盖 Claude 分组。没有 Claude 分组的站点，或尚未完成抽检的分组，会显示“未检测”，这不是负面评价。"},
	{"这个网站的数据是真实的吗？",
		"当前仓库使用演示数据集：站点名称与域名取自媒体公开信息，仅用于展示图标；倍率、探针、评分与评价为生成值，不代表真实商家表现。"},
}

func auditBody() template.HTML {
	return template.HTML(`<article><h1>API 中转站接口检测</h1>
<p>对任意 OpenAI 兼容端点发送两次完全相同的请求，从协议与计费层面判断它是否可信。从站点拉取模型列表后可以多选，逐个模型检测，每个模型生成一份可分享的报告。页面会逐项列出检查内容与通过标准，检测过程中显示每个模型的进度，结束后给出「检查项 × 模型」结果矩阵与逐项明细。检测不判断“模型是否降智”，只核对可观测信号。</p>
<h2>检查项</h2>
<ul>
<li>端点可达：HTTP 2xx、TLS 正常、不跟随重定向</li>
<li>鉴权通过：401/403 单独标记为失败</li>
<li>流式完整性：收到 delta.content 与 [DONE]</li>
<li>模型回显：响应 model 与请求一致，否则提示可能替换</li>
<li>用量上报：usage.prompt_tokens &gt; 0</li>
<li>分词一致性：两次相同请求的 prompt_tokens 一致</li>
<li>缓存透传：第二次 cached_tokens &gt; 0</li>
<li>首字延迟：&lt;1500ms 通过，&lt;4000ms 观察</li>
</ul>
<h2>安全</h2>
<p>Key 只用于本次探测，不写日志、不落库；服务端对目标地址做 SSRF 校验（仅 https、拒绝私网与链路本地地址、不跟随重定向）。</p>
<h2>检测项目</h2>
<ul><li><a href="/audit/reachable">端点可达</a></li><li><a href="/audit/auth">鉴权通过</a></li><li><a href="/audit/stream">流式完整性</a></li><li><a href="/audit/model_echo">模型回显</a></li><li><a href="/audit/usage">用量上报</a></li><li><a href="/audit/token_consistency">分词一致性</a></li><li><a href="/audit/cache">缓存透传</a></li><li><a href="/audit/ttft">首字延迟</a></li></ul>
<p><a href="/audit">打开检测工具</a> · <a href="/methodology">评分方法</a></p></article>`)
}

func sponsorBody() template.HTML {
	return template.HTML(`<article><h1>赞助与广告</h1>
<p>赞助位用于覆盖探针与带宽成本，不改变分数、排序或测评结论。</p>
<h2>规则</h2>
<ul><li>所有赞助内容标注 Sponsor，不与自然结果混排。</li><li>赞助不参与评分，也不能要求删除负面检测结果。</li><li>文案与数据分离：评分与探针由本站独立生成。</li><li>不接受无法核实主体的投放。</li></ul>
<h2>位置与示例报价</h2>
<table class="index-table"><thead><tr><th>位置</th><th>示例价格</th><th>说明</th></tr></thead><tbody>
<tr><td>首页赞助条</td><td>¥149 / 月</td><td>顶部一条文字赞助，标注 Sponsor</td></tr>
<tr><td>目录赞助卡</td><td>¥249 / 月</td><td>目录侧栏卡片，展示 logo 与简介</td></tr>
<tr><td>榜单页赞助位</td><td>¥399 / 月</td><td>固定位置，与自然结果分离</td></tr>
</tbody></table>
<p>示例报价，最终以合同为准；当前为演示部署，不构成真实要约。</p>
<p><a href="/methodology">评分方法</a> · <a href="/">返回榜单</a></p></article>`)
}

func faqBody() template.HTML {	var b strings.Builder
	b.WriteString(`<article><h1>常见问题</h1>`)
	for _, q := range faqItems {
		fmt.Fprintf(&b, `<h2>%s</h2><p>%s</p>`, escHTML(q.Q), escHTML(q.A))
	}
	b.WriteString(`<p><a href="/methodology">评分方法</a> · <a href="/">返回榜单</a></p></article>`)
	return template.HTML(b.String())
}

func (s *Server) ldFor(r *http.Request, m pageMeta, d *app.Dataset) []any {
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	org := map[string]any{
		"@context": "https://schema.org", "@type": "Organization",
		"name": "RelayRadar", "url": base + "/",
	}
	out := []any{org}

	crumb := []any{}
	addCrumb := func(name, url string) {
		crumb = append(crumb, map[string]any{"@type": "ListItem", "position": len(crumb) + 1, "name": name, "item": url})
	}
	addCrumb("首页", base+"/")

	path := r.URL.Path
	switch {
	case path == "/":
		out = append(out, map[string]any{
			"@context": "https://schema.org", "@type": "WebSite",
			"name": "RelayRadar 中转雷达", "url": base + "/",
			"potentialAction": map[string]any{
				"@type": "SearchAction", "target": base + "/?q={query}", "query-input": "required name=query",
			},
		})
		items := []any{}
		for i, st := range d.Stations {
			if i >= 10 {
				break
			}
			items = append(items, map[string]any{
				"@type": "ListItem", "position": i + 1,
				"name": st.Name, "url": base + "/stations/" + st.Slug,
			})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "ItemList", "itemListElement": items})
	case path == "/models":
		addCrumb("模型目录", base+"/models")
		items := make([]any, 0, 20)
		for i, m := range d.Models {
			if i >= 20 {
				break
			}
			items = append(items, map[string]any{
				"@type": "ListItem", "position": i + 1,
				"name": m.Name, "url": base + "/models/" + m.ID,
			})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "ItemList", "itemListElement": items})
	case path == "/stations":
		addCrumb("站点目录", base+"/stations")
		items := make([]any, 0, 20)
		for _, st := range d.Stations {
			if st.OfficialVendor || len(items) >= 20 {
				continue
			}
			items = append(items, map[string]any{
				"@type": "ListItem", "position": len(items) + 1,
				"name": st.Name, "url": base + "/stations/" + st.Slug,
			})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "ItemList", "itemListElement": items})
	case path == "/leaderboards":
		addCrumb("排行榜", base+"/leaderboards")
		items := make([]any, 0, 10)
		for _, st := range d.Stations {
			if st.OfficialVendor || len(items) >= 10 {
				continue
			}
			items = append(items, map[string]any{
				"@type": "ListItem", "position": len(items) + 1,
				"name": st.Name, "url": base + "/stations/" + st.Slug,
			})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "ItemList", "itemListElement": items})
	case strings.HasPrefix(path, "/stations/"):
		slug := strings.TrimPrefix(path, "/stations/")
		if st, ok := d.BySlug[slug]; ok {
			addCrumb(st.Name, base+"/stations/"+st.Slug)
			out = append(out, map[string]any{
				"@context": "https://schema.org", "@type": "WebPage",
				"name": st.Name, "url": base + "/stations/" + st.Slug,
				"description": st.Tagline, "isPartOf": map[string]any{"@type": "WebSite", "url": base + "/"},
			})
		}
	case strings.HasPrefix(path, "/models/"):
		id := strings.TrimPrefix(path, "/models/")
		if mm, ok := d.ModelByID[id]; ok {
			addCrumb("模型", base+"/models")
			addCrumb(mm.Name, base+"/models/"+mm.ID)
			out = append(out, map[string]any{
				"@context": "https://schema.org", "@type": "WebPage",
				"name": mm.Name, "url": base + "/models/" + mm.ID, "description": mm.Summary,
			})
			if mm.PriceLowCNY > 0 {
				out = append(out, map[string]any{
					"@context":    "https://schema.org",
					"@type":       "Product",
					"name":        mm.Name,
					"description": mm.Summary,
					"brand":       map[string]any{"@type": "Brand", "name": mm.VendorName},
					"category":    categoryLabel(mm.Category),
					"offers": map[string]any{
						"@type":         "AggregateOffer",
						"priceCurrency": "CNY",
						"lowPrice":      mm.PriceLowCNY,
						"highPrice":     mm.PriceHighCNY,
						"offerCount":    mm.OffersCount,
						"availability":  "https://schema.org/InStock",
					},
				})
			}
			faq := make([]any, 0, 4)
			for _, q := range modelFAQ(mm) {
				faq = append(faq, map[string]any{
					"@type": "Question", "name": q.Q,
					"acceptedAnswer": map[string]any{"@type": "Answer", "text": q.A},
				})
			}
			out = append(out, map[string]any{
				"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": faq,
			})
		}
	case path == "/faq":
		items := []any{}
		for _, q := range faqItems {
			items = append(items, map[string]any{
				"@type": "Question", "name": q.Q,
				"acceptedAnswer": map[string]any{"@type": "Answer", "text": q.A},
			})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": items})
		addCrumb("常见问题", base+"/faq")
	case strings.HasPrefix(path, "/guides/"):
		if g := guideBySlug(strings.TrimPrefix(path, "/guides/")); g != nil {
			addCrumb("指南", base+"/guides")
			addCrumb(g.Title, base+"/guides/"+g.Slug)
			out = append(out, map[string]any{
				"@context":     "https://schema.org",
				"@type":        "TechArticle",
				"headline":     g.Title,
				"description":  g.Desc,
				"dateModified": g.Updated,
				"datePublished": g.Updated,
				"inLanguage":   "zh-CN",
				"author":       map[string]any{"@type": "Organization", "name": "RelayRadar"},
				"publisher":    map[string]any{"@type": "Organization", "name": "RelayRadar"},
				"mainEntityOfPage": base + "/guides/" + g.Slug,
			})
		}
	case path == "/guides":
		addCrumb("指南", base+"/guides")
	}

	if len(crumb) > 1 {
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": crumb})
	}
	_ = m
	return out
}

// ---------------------------------------------------------------------------
// Markdown endpoints for AI agents: /md/station/{slug}, /md/model/{id}
// ---------------------------------------------------------------------------

func (s *Server) handleStationMD(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	st, ok := d.BySlug[r.PathValue("slug")]
	if !ok {
		writeError(w, http.StatusNotFound, "station not found")
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> %s\n\n", st.Name, st.Tagline)
	fmt.Fprintf(&b, "- 域名：%s\n- 综合分：%.1f（第 %d 名）\n- 原始可用率：%.1f%%\n- 速度分：%.1f\n- 稳定性分：%.1f\n- 纯度：%s（%.0f）\n- 最低输入价：¥%.2f / 1M\n- 最低充值：¥%.0f\n- 支付：%s\n- 退款：%s\n\n",
		st.Domain, st.RankScore, st.Rank, st.UptimePct, st.Speed, st.Consistency,
		tierLabel(st.PurityTier), st.Purity, st.FromPriceCNY, st.MinTopup,
		strings.Join(st.Payments, "、"), st.RefundPolicy)
	b.WriteString("## 分组\n\n| 分组 | 服务 | 倍率 | 延迟 | 可用性分 | 速度分 | 稳定性分 | 纯度 |\n|---|---|---|---|---|---|---|---|\n")
	for _, g := range d.GroupsByStation[st.Slug] {
		fmt.Fprintf(&b, "| %s | %s | %.2f× | %d ms | %.1f | %.1f | %.1f | %s |\n",
			g.Name, g.Service, g.Multiplier, g.LatencyMS, g.Availability, g.Speed, g.Consistency, tierLabel(g.PurityTier))
	}
	b.WriteString("\n## 模型与实价（¥ / 1M tokens）\n\n| 模型 | 输入 | 输出 | 最低价分组 |\n|---|---|---|---|\n")
	for _, sm := range d.PerStationModels[st.Slug] {
		fmt.Fprintf(&b, "| %s | %.2f | %.2f | %s |\n", sm.Model.ID, sm.BestInputCNY, sm.BestOutputCNY, sm.CheapestGroup)
	}
	b.WriteString("\n数据时间：" + d.TakenAt.UTC().Format(time.RFC3339) + "\n")
	b.WriteString("评分方法：" + s.cfg.PublicBaseURL + "/methodology\n")

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func (s *Server) handleModelMD(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	m, ok := d.ModelByID[r.PathValue("id")]
	if !ok {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%s\n\n", m.Name, m.Summary)
	fmt.Fprintf(&b, "- 厂商：%s\n- 官方价：$%.2f / $%.2f per 1M\n- 上下文：%d\n- 编程分：%.1f · 推理分：%.1f\n- 全网最低输入：¥%.2f / 1M\n\n",
		m.VendorName, m.InputUSD, m.OutputUSD, m.ContextWindow, m.CodingScore, m.ReasoningScore, m.MinInputCNY)
	b.WriteString("## 报价\n\n| 站点 | 分组 | 倍率 | 输入 | 输出 | 纯度 |\n|---|---|---|---|---|---|\n")
	for _, o := range d.Offers[m.ID] {
		fmt.Fprintf(&b, "| %s | %s | %.2f× | ¥%.2f | ¥%.2f | %s |\n",
			o.StationName, o.GroupName, o.Multiplier, o.InputCNY, o.OutputCNY, tierLabel(o.PurityTier))
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

// ---------------------------------------------------------------------------
// llms-full.txt + RSS
// ---------------------------------------------------------------------------

func (s *Server) handleLLMsFull(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	var b strings.Builder
	b.WriteString("# RelayRadar — full dataset\n\n")
	fmt.Fprintf(&b, "Data timestamp: %s\nBase URL: %s\n\n", d.TakenAt.UTC().Format(time.RFC3339), s.cfg.PublicBaseURL)

	b.WriteString("## Stations\n\n")
	for _, st := range d.Stations {
		fmt.Fprintf(&b, "### %s (`%s`)\n", st.Name, st.Slug)
		fmt.Fprintf(&b, "- rank %d · score %.1f · uptime %.1f%% · speed %.1f · consistency %.1f · purity %s\n",
			st.Rank, st.RankScore, st.UptimePct, st.Speed, st.Consistency, tierLabel(st.PurityTier))
		fmt.Fprintf(&b, "- domain %s · min topup ¥%.0f · invoice %v · refund %s\n", st.Domain, st.MinTopup, st.Invoice, st.RefundPolicy)
		fmt.Fprintf(&b, "- markdown: %s/md/station/%s\n", s.cfg.PublicBaseURL, st.Slug)
	}

	b.WriteString("\n## Models\n\n")
	for _, m := range d.Models {
		fmt.Fprintf(&b, "### %s (`%s`)\n", m.Name, m.ID)
		fmt.Fprintf(&b, "- vendor %s · official $%.2f/$%.2f · ctx %d · coding %.1f · reasoning %.1f\n",
			m.VendorName, m.InputUSD, m.OutputUSD, m.ContextWindow, m.CodingScore, m.ReasoningScore)
		fmt.Fprintf(&b, "- cheapest input ¥%.2f · offers %d · markdown %s/md/model/%s\n",
			m.MinInputCNY, m.OffersCount, s.cfg.PublicBaseURL, m.ID)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []rssItem `xml:"item"`
	} `xml:"channel"`
}

func (s *Server) handleChangesRSS(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	var doc rssDoc
	doc.Version = "2.0"
	doc.Channel.Title = "RelayRadar 变更雷达"
	doc.Channel.Link = base + "/changes"
	doc.Channel.Description = "中转站价格、可用率、纯度与新站变更"
	for i, c := range d.Changes {
		if i >= 30 {
			break
		}
		doc.Channel.Items = append(doc.Channel.Items, rssItem{
			Title:       c.StationName + " · " + c.Title,
			Link:        base + "/stations/" + c.StationSlug,
			Description: c.Detail,
			PubDate:     c.At.Format(time.RFC1123Z),
			GUID:        base + "/changes#" + c.ID,
		})
	}
	out, _ := xml.MarshalIndent(doc, "", "  ")
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

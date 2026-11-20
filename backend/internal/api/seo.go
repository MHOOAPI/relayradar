package api

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"relayradar/internal/app"
)

type Hreflang struct {
	Lang string
	URL  string
}

type pageMeta struct {
	Title     string
	Desc      string
	Canonical string
	Alternate string
	Image     string
	Hreflang  []Hreflang
	NoIndex   bool
	// Body is prerendered, crawler-visible content placed inside #root before
	// React mounts. React replaces it on first render, so users see the app.
	Body template.HTML
	LD   []any
}

var titleRe = regexp.MustCompile(`(?s)<title>.*?</title>`)
var descRe = regexp.MustCompile(`(?s)<meta name="description" content="[^"]*"\s*/?>`)
var ogImageRe = regexp.MustCompile(`(?s)<meta property="og:image" content="[^"]*"\s*/?>\s*`)
var twImageRe = regexp.MustCompile(`(?s)<meta name="twitter:image" content="[^"]*"\s*/?>\s*`)

// spaHandler serves the built React app, falling back to index.html with
// per-page meta tags so social crawlers and search engines see real titles.
func (s *Server) spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	indexPath := filepath.Join(dir, "index.html")
	var index []byte
	var indexMod time.Time

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Re-read index.html when the file changes so a rebuilt frontend does
		// not keep serving hashed asset names that no longer exist.
		if info, err := os.Stat(indexPath); err == nil && info.ModTime().After(indexMod) {
			if b, err := os.ReadFile(indexPath); err == nil {
				index = b
				indexMod = info.ModTime()
			}
		}
		clean := filepath.Clean("/" + r.URL.Path)
		candidate := filepath.Join(dir, clean)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		if len(index) == 0 {
			http.ServeFile(w, r, indexPath)
			return
		}
		// A missing asset should 404 rather than fall back to the SPA shell.
		if ext := filepath.Ext(clean); ext != "" && ext != ".html" {
			http.NotFound(w, r)
			return
		}
		page := index
		if m, ok := s.pageMeta(r); ok {
			if d, err := s.svc.Dataset(r.Context()); err == nil {
				m.Body = s.bodyFor(r, d)
				m.LD = append(m.LD, s.ldFor(r, m, d)...)
			}
			page = injectMeta(index, m, s.cfg.PublicBaseURL)
		} else if missingContent(r.URL.Path) {
			// Unknown station/model/guide: return a real 404 so crawlers do not
			// index a soft-404, while the SPA still renders its NotFound page.
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(index)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(page)
	})
}

// missingContent reports whether the path looks like a detail page that was
// not found, so the handler can answer 404 instead of a soft 200.
func missingContent(path string) bool {
	for _, prefix := range []string{"/stations/", "/models/", "/guides/"} {
		rest, ok := strings.CutPrefix(path, prefix)
		if ok && rest != "" && !strings.Contains(rest, "/") {
			return true
		}
	}
	return false
}

func (s *Server) pageMeta(r *http.Request) (pageMeta, bool) {	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		return pageMeta{}, false
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	path := r.URL.Path

	if key, ok := strings.CutPrefix(path, "/audit/"); ok && key != "" {
		if c := auditCheckByKey(key); c != nil {
			return pageMeta{
				Title:     c.Label + "：接口检测项目 | RelayRadar",
				Desc:      c.Short,
				Canonical: base + path,
			}, true
		}
	}

	if path == "/models/license" {
		return pageMeta{
			Title:     "按许可证浏览模型 | RelayRadar",
			Desc:      "按 MIT、Apache-2.0、Modified MIT 与闭源许可证筛选模型，查看许可证与实价。",
			Canonical: base + path,
		}, true
	}
	if slug, ok := strings.CutPrefix(path, "/models/license/"); ok && slug != "" {
		return pageMeta{
			Title:     licenseLabel(slug) + " 许可证模型 | RelayRadar",
			Desc:      "使用 " + licenseLabel(slug) + " 许可证的大模型，附官方价与全网最低中转实价。",
			Canonical: base + path,
		}, true
	}

	if path == "/en/audit" {
		return pageMeta{
			Title:     "Audit an OpenAI-compatible endpoint | RelayRadar",
			Desc:      "Real requests measure TTFT, stream integrity, model echo, usage, tokenizer consistency and cache passthrough. Keys are never stored.",
			Canonical: base + "/en/audit",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/audit"},
				{Lang: "zh-CN", URL: base + "/audit"},
			},
		}, true
	}
	if key, ok := strings.CutPrefix(path, "/en/audit/"); ok && key != "" {
		c := auditCheckENByKey(key)
		if c == nil {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     c.Label + " — audit check | RelayRadar",
			Desc:      c.Short,
			Canonical: base + "/en/audit/" + key,
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/audit/" + key},
				{Lang: "zh-CN", URL: base + "/audit/" + key},
			},
		}, true
	}

	if path == "/en/leaderboards" {
		return pageMeta{
			Title:     "Relay leaderboard: all providers ranked | RelayRadar",
			Desc:      "Full provider leaderboard with availability, speed, consistency, purity and lowest price in one table.",
			Canonical: base + "/en/leaderboards",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/leaderboards"},
				{Lang: "zh-CN", URL: base + "/leaderboards"},
			},
		}, true
	}
	if slug, ok := strings.CutPrefix(path, "/en/stations/"); ok && slug != "" {
		st, found := d.BySlug[slug]
		if !found {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     st.Name + " relay profile | RelayRadar",
			Desc:      st.TaglineEN,
			Canonical: base + "/en/stations/" + slug,
			Image:     "/share/station-" + slug + ".png",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/stations/" + slug},
				{Lang: "zh-CN", URL: base + "/stations/" + slug},
			},
		}, true
	}

	if path == "/en/guides" {
		return pageMeta{
			Title:     "Guides: choosing, billing and verifying relays | RelayRadar",
			Desc:      "Three long-form guides: a 10-minute provider checklist, multiplier and cache math, and how to judge thinking signatures.",
			Canonical: base + "/en/guides",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/guides"},
				{Lang: "zh-CN", URL: base + "/guides"},
			},
		}, true
	}
	if slug, ok := strings.CutPrefix(path, "/en/guides/"); ok && slug != "" {
		g := guideENBySlug(slug)
		if g == nil {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     g.Title + " | RelayRadar",
			Desc:      g.Desc,
			Canonical: base + "/en/guides/" + slug,
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/guides/" + slug},
				{Lang: "zh-CN", URL: base + "/guides/" + slug},
			},
		}, true
	}

	if path == "/en/changes" {
		return pageMeta{
			Title:     "Change feed: new listings, purity alerts and price moves | RelayRadar",
			Desc:      "New listings, purity alerts, availability drops, price moves and score regressions, derived from the same probes as the rankings.",
			Canonical: base + "/en/changes",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/changes"},
				{Lang: "zh-CN", URL: base + "/changes"},
			},
		}, true
	}
	if path == "/en/free" {
		return pageMeta{
			Title:     "Free tiers and trial credits for LLM APIs | RelayRadar",
			Desc:      "Official free tiers and trial credits only, with the cheapest relay price next to the official one.",
			Canonical: base + "/en/free",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/free"},
				{Lang: "zh-CN", URL: base + "/free"},
			},
		}, true
	}

	if path == "/en/models" {
		return pageMeta{
			Title:     "Model directory and lowest relay prices | RelayRadar",
			Desc:      "Official USD prices next to the cheapest relay price in CNY, with context window and license.",
			Canonical: base + "/en/models",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/models"},
				{Lang: "zh-CN", URL: base + "/models"},
			},
		}, true
	}
	if id, ok := strings.CutPrefix(path, "/en/models/"); ok && id != "" {
		m, found := d.ModelByID[id]
		if !found {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     m.Name + " pricing and relays | RelayRadar",
			Desc:      m.SummaryEN,
			Canonical: base + "/en/models/" + id,
			Image:     "/share/model-" + id + ".png",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/models/" + id},
				{Lang: "zh-CN", URL: base + "/models/" + id},
			},
		}, true
	}

	if path == "/en/status" {
		return pageMeta{
			Title:     "Service status | RelayRadar",
			Desc:      "Live status for the public API, database, probes, alert delivery, audit and email channel.",
			Canonical: base + "/en/status",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/status"},
				{Lang: "zh-CN", URL: base + "/status"},
			},
		}, true
	}
	if key, ok := strings.CutPrefix(path, "/en/model-status/"); ok && key != "" {
		for _, p := range modelProviders {
			if p.Key == key {
				return pageMeta{
					Title:     p.Name + " official API status | RelayRadar",
					Desc:      "Current status and incident timeline for " + p.Name + ", sourced from the official status page.",
					Canonical: base + "/en/model-status/" + key,
					Hreflang: []Hreflang{
						{Lang: "en", URL: base + "/en/model-status/" + key},
						{Lang: "zh-CN", URL: base + "/model-status/" + key},
					},
				}, true
			}
		}
		return pageMeta{}, false
	}
	if path == "/en/model-status" {
		return pageMeta{
			Title:     "Upstream model API status | RelayRadar",
			Desc:      "Official API status for OpenAI, Anthropic, Google Gemini, DeepSeek, Moonshot, Qwen and xAI.",
			Canonical: base + "/en/model-status",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/model-status"},
				{Lang: "zh-CN", URL: base + "/model-status"},
			},
		}, true
	}
	if key, ok := strings.CutPrefix(path, "/model-status/"); ok && key != "" {
		for _, p := range modelProviders {
			if p.Key == key {
				return pageMeta{
					Title:     p.Name + " 官方 API 状态历史 | RelayRadar",
					Desc:      p.Name + " 的官方 API 当前状态与历史事件时间线，数据来自官方状态页。",
					Canonical: base + "/model-status/" + key,
				}, true
			}
		}
		return pageMeta{}, false
	}
	if path == "/model-status" {
		return pageMeta{
			Title:     "上游模型 API 状态：OpenAI / Anthropic / Gemini 等 | RelayRadar",
			Desc:      "OpenAI、Anthropic、Google Gemini、DeepSeek、Moonshot、阿里 Qwen、xAI 的官方 API 状态聚合，优先读取官方状态页接口。",
			Canonical: base + "/model-status",
		}, true
	}
	if path == "/status" {
		return pageMeta{
			Title:     "服务状态 | RelayRadar",
			Desc:      "公开 API、数据存储、探针采集、告警投递、接口检测与邮件通道的实时状态。",
			Canonical: base + "/status",
		}, true
	}
	if path == "/free-api-keys" {
		return pageMeta{
			Title:     "免费 API Key 分享：无需登录提交与领取 | RelayRadar",
			Desc:      "用户提交的 OpenAI 兼容免费 Key，无需登录即可提交或领取；提交后自动探测一次可用性。共享 Key 随时可能失效。",
			Canonical: base + "/free-api-keys",
		}, true
	}

	switch path {
	case "/":
		return pageMeta{
			Title:     "RelayRadar 中转雷达 · 大模型 API 中转站独立监测与排序",
			Desc:      "用连续探针、Wilson 可用性下界、速度与稳定性三轴，给大模型 API 中转站排序；价格、纯度与变更单独展示。",
			Canonical: base + "/",
			Hreflang: []Hreflang{
				{Lang: "zh-CN", URL: base + "/"},
				{Lang: "en", URL: base + "/en"},
				{Lang: "x-default", URL: base + "/"},
			},
		}, true
	case "/en":
		return pageMeta{
			Title:     "RelayRadar — independent LLM API relay monitoring",
			Desc:      "Independent rankings for LLM API relay providers: Wilson availability lower bound, latency, consistency, purity checks and price changes.",
			Canonical: base + "/en",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en"},
				{Lang: "zh-CN", URL: base + "/"},
				{Lang: "x-default", URL: base + "/"},
			},
		}, true
	case "/en/methodology":
		return pageMeta{
			Title:     "Scoring methodology v2 | RelayRadar",
			Desc:      "Availability uses a Wilson 95% lower bound, speed maps median latency, consistency measures p95 jitter; weights are 0.62 / 0.26 / 0.12.",
			Canonical: base + "/en/methodology",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/methodology"},
				{Lang: "zh-CN", URL: base + "/methodology"},
			},
		}, true
	case "/en/faq":
		return pageMeta{
			Title:     "FAQ: relays, scoring and audits | RelayRadar",
			Desc:      "How to choose an LLM relay, how scoring works, what the audit checks, and what our data does and does not cover.",
			Canonical: base + "/en/faq",
			Hreflang: []Hreflang{
				{Lang: "en", URL: base + "/en/faq"},
				{Lang: "zh-CN", URL: base + "/faq"},
			},
		}, true
	case "/models":
		return pageMeta{
			Title:     "模型目录与全网最低价 | RelayRadar",
			Desc:      "对比官方美元价与中转站折算人民币实价，按模型反查全部报价站点。",
			Canonical: base + path,
		}, true
	case "/api-docs":
		return pageMeta{
			Title:     "公开 API 与数据口径 | RelayRadar",
			Desc:      "只读 JSON 接口、字段含义、限流与 SSRF 防护说明，附 llms.txt 与 llms-full.txt。",
			Canonical: base + path,
		}, true
	case "/audit":
		return pageMeta{
			Title:     "API 中转站接口检测：签名、计费与缓存 | RelayRadar",
			Desc:      "对任意 OpenAI 兼容端点发两次相同请求，检查模型回显、usage、分词一致性、缓存透传与首字延迟。",
			Canonical: base + path,
		}, true
	case "/submit":
		return pageMeta{
			Title:     "提交站点与站长认领 | RelayRadar",
			Desc:      "提交新站点进入观察队列，或通过 DNS TXT 令牌认领档案，补充倍率与商务条款。",
			Canonical: base + path,
		}, true
	case "/login":
		return pageMeta{Title: "登录 | RelayRadar", Desc: "登录后管理提交与告警订阅。", Canonical: base + path, NoIndex: true}, true
	case "/register":
		return pageMeta{Title: "注册 | RelayRadar", Desc: "创建账号，管理提交与告警订阅。", Canonical: base + path, NoIndex: true}, true
	case "/admin":
		return pageMeta{Title: "后台 | RelayRadar", Desc: "站点运营后台。", Canonical: base + path, NoIndex: true}, true
	case "/sponsor":
		return pageMeta{
			Title:     "赞助与广告：位置、规则与报价 | RelayRadar",
			Desc:      "赞助位明确标注、不参与评分与排序；首页赞助条、目录卡片与榜单页位置的示例报价。",
			Canonical: base + path,
		}, true
	case "/faq":
		return pageMeta{
			Title:     "常见问题：中转站怎么选、评分怎么算 | RelayRadar",
			Desc:      "中转站是什么、怎么判断是否掺水、倍率怎么算、评分为什么不含价格、数据多久更新一次。",
			Canonical: base + path,
			Hreflang: []Hreflang{
				{Lang: "zh-CN", URL: base + "/faq"},
				{Lang: "en", URL: base + "/en/faq"},
			},
		}, true
	case "/guides":
		return pageMeta{
			Title:     "指南：中转站选择、计费与验真 | RelayRadar",
			Desc:      "三篇长文：10 分钟验证清单、倍率与缓存计费公式、thinking 签名与降智的判断方法。",
			Canonical: base + path,
		}, true
	case "/changes":
		return pageMeta{
			Title:     "变更雷达 · 价格与可用性异动 | RelayRadar",
			Desc:      "从探针与价格历史自动检测：可用率骤降、纯度抽检异常、价格调整与新站上架。",
			Canonical: base + path,
		}, true
	case "/leaderboards":
		return pageMeta{
			Title:     "排行榜 · 模型能力 / 性价比 / 站点稳定性 | RelayRadar",
			Desc:      "编程、推理、性价比与站点速度、稳定性分榜，能力与价格分开看。",
			Canonical: base + path,
		}, true
	case "/compare":
		return pageMeta{
			Title:     "中转站横向对比 | RelayRadar",
			Desc:      "最多 4 家并排对比可用性、速度、稳定性、纯度与商务条款，链接可分享。",
			Canonical: base + path,
		}, true
	case "/free":
		return pageMeta{
			Title:     "免费额度目录 · 官方免费层与赠送额度 | RelayRadar",
			Desc:      "主流模型的官方免费层、赠送额度、限速与绑卡要求，附使用建议。",
			Canonical: base + path,
		}, true
	case "/methodology":
		return pageMeta{
			Title:     "评分方法 v2 · 公式、权重与局限 | RelayRadar",
			Desc:      "Wilson 下界、速度与稳定性轴、覆盖加权与最差分组混合，全部可手算复现。",
			Canonical: base + path,
			Hreflang: []Hreflang{
				{Lang: "zh-CN", URL: base + "/methodology"},
				{Lang: "en", URL: base + "/en/methodology"},
			},
		}, true
	}

	if slug, ok := strings.CutPrefix(path, "/guides/"); ok && slug != "" {
		g := guideBySlug(slug)
		if g == nil {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     g.Title + " | RelayRadar",
			Desc:      g.Desc,
			Canonical: base + path,
		}, true
	}

	if slug, ok := strings.CutPrefix(path, "/go/"); ok && slug != "" {
		st, found := d.BySlug[slug]
		if !found {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     "即将离开：前往 " + st.Name + " | RelayRadar",
			Desc:      st.Name + "（" + st.Domain + "）访问前提示：核对域名、先小额充值、验证计费；RelayRadar 不参与交易、不收款。",
			Canonical: base + "/stations/" + slug,
			NoIndex:   true,
		}, true
	}

	if path == "/stations" {
		return pageMeta{
			Title:     "中转站目录：模型覆盖、支付、退款、发票与评价 | RelayRadar",
			Desc:      "按模型覆盖、健康状态、付款方式、退款政策、发票与用户评价浏览中转站；实时评分与价格见首页榜单。",
			Canonical: base + "/stations",
		}, true
	}

	if slug, ok := strings.CutPrefix(path, "/stations/"); ok && slug != "" {
		st, found := d.BySlug[slug]
		if !found {
			return pageMeta{}, false
		}
		price := "暂无报价"
		if st.FromPriceCNY >= 0 {
			price = fmt.Sprintf("最低输入价 ¥%.2f/百万 Token", st.FromPriceCNY)
		}
		return pageMeta{
			Title:     fmt.Sprintf("%s 中转站档案 · 排名 #%d | RelayRadar", st.Name, st.Rank),
			Desc:      fmt.Sprintf("%s。综合分 %.1f，原始可用率 %.1f%%，%s。", st.Tagline, st.RankScore, st.UptimePct, price),
			Canonical: base + path,
			Alternate: base + "/md/station/" + st.Slug,
			Image:     "/share/station-" + st.Slug + ".png",
		}, true
	}

	if id, ok := strings.CutPrefix(path, "/models/"); ok && id != "" {		m, found := d.ModelByID[id]
		if !found {
			return pageMeta{}, false
		}
		return pageMeta{
			Title:     fmt.Sprintf("%s 价格与中转站报价 | RelayRadar", m.Name),
			Desc:      fmt.Sprintf("%s：官方 $%.2f/$%.2f per 1M，全网最低 ¥%.2f/百万输入 Token。", m.Summary, m.InputUSD, m.OutputUSD, m.MinInputCNY),
			Canonical: base + path,
			Alternate: base + "/md/model/" + m.ID,
			Image:     "/share/model-" + m.ID + ".png",
		}, true
	}

	return pageMeta{}, false
}

func injectMeta(index []byte, m pageMeta, base string) []byte {	doc := string(index)
	title := html.EscapeString(m.Title)
	desc := html.EscapeString(m.Desc)
	canonical := html.EscapeString(m.Canonical)

	s := titleRe.ReplaceAllString(doc, "<title>"+title+"</title>")
	s = descRe.ReplaceAllString(s, `<meta name="description" content="`+desc+`" />`)
	// Drop the static fallback image so each page exposes exactly one og:image.
	s = ogImageRe.ReplaceAllString(s, "")
	s = twImageRe.ReplaceAllString(s, "")

	ld, _ := json.Marshal(map[string]string{
		"@context": "https://schema.org", "@type": "WebPage",
		"name": m.Title, "description": m.Desc, "url": m.Canonical,
	})
	blocks := []string{string(ld)}
	for _, v := range m.LD {
		if b, err := json.Marshal(v); err == nil {
			blocks = append(blocks, string(b))
		}
	}
	og := html.EscapeString(base) + "/og.png"
	if m.Image != "" {
		og = html.EscapeString(base) + m.Image
	}
	extra := strings.Join([]string{
		`<link rel="canonical" href="` + canonical + `" />`,
		`<meta property="og:site_name" content="RelayRadar" />`,
		`<meta property="og:type" content="website" />`,
		`<meta property="og:title" content="` + title + `" />`,
		`<meta property="og:description" content="` + desc + `" />`,
		`<meta property="og:url" content="` + canonical + `" />`,
		`<meta property="og:image" content="` + og + `" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="twitter:title" content="` + title + `" />`,
		`<meta name="twitter:description" content="` + desc + `" />`,
		`<meta name="twitter:image" content="` + og + `" />`,
	}, "\n    ")
	for _, b := range blocks {
		extra += "\n    " + `<script type="application/ld+json">` + b + `</script>`
	}
	if m.NoIndex {
		extra += "\n    " + `<meta name="robots" content="noindex,follow" />`
	}
	if m.Alternate != "" {
		extra += "\n    " + `<link rel="alternate" type="text/markdown" href="` + html.EscapeString(m.Alternate) + `" />`
	}
	for _, alt := range m.Hreflang {
		extra += "\n    " + `<link rel="alternate" hreflang="` + html.EscapeString(alt.Lang) + `" href="` + html.EscapeString(alt.URL) + `" />`
	}

	out := strings.Replace(s, "</head>", "    "+extra+"\n  </head>", 1)
	if len(m.Body) > 0 {
		out = strings.Replace(out, `<div id="root"></div>`, `<div id="root">`+string(m.Body)+`</div>`, 1)
	}
	return []byte(out)
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	last := d.TakenAt.UTC().Format("2006-01-02")
	add := func(loc, freq, prio string) {
		fmt.Fprintf(&b, "  <url><loc>%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>\n",
			html.EscapeString(base+loc), last, freq, prio)
	}
	add("/", "hourly", "1.0")
	add("/en", "weekly", "0.6")
	add("/en/methodology", "monthly", "0.5")
	add("/en/faq", "monthly", "0.5")
	add("/en/guides", "weekly", "0.6")
	for _, slug := range []string{"how-to-choose-relay", "multiplier-math", "signature-verification"} {
		add("/en/guides/"+slug, "monthly", "0.6")
	}
	add("/en/models", "daily", "0.6")
	add("/en/leaderboards", "daily", "0.6")
	add("/en/audit", "monthly", "0.6")
	add("/en/changes", "hourly", "0.6")
	add("/en/free", "weekly", "0.5")
	for _, c := range auditCheckDocsEN {
		add("/en/audit/"+c.Key, "monthly", "0.5")
	}
	for _, m := range d.Models {
		add("/en/models/"+m.ID, "daily", "0.5")
	}
	for _, st := range d.Stations {
		add("/en/stations/"+st.Slug, "daily", "0.5")
	}
	add("/leaderboards", "daily", "0.9")
	add("/models", "daily", "0.8")
	add("/stations", "daily", "0.8")
	add("/changes", "hourly", "0.8")
	add("/free", "weekly", "0.7")
	add("/free-api-keys", "hourly", "0.7")
	add("/faq", "monthly", "0.7")
	add("/sponsor", "monthly", "0.4")
	add("/status", "hourly", "0.6")
	add("/model-status", "hourly", "0.7")
	for _, p := range modelProviders {
		add("/model-status/"+p.Key, "hourly", "0.6")
		add("/en/model-status/"+p.Key, "hourly", "0.5")
	}
	add("/en/model-status", "hourly", "0.6")
	add("/en/status", "hourly", "0.5")
	for _, c := range auditCheckDocs {
		add("/audit/"+c.Key, "monthly", "0.6")
	}
	add("/guides", "weekly", "0.8")
	for _, g := range guides {
		add("/guides/"+g.Slug, "monthly", "0.7")
	}
	add("/compare", "weekly", "0.6")
	add("/methodology", "monthly", "0.6")
	add("/api-docs", "monthly", "0.5")
	for _, st := range d.Stations {
		add("/stations/"+st.Slug, "daily", "0.8")
	}
	for _, m := range d.Models {
		add("/models/"+m.ID, "daily", "0.7")
	}
	add("/models/license", "monthly", "0.6")
	for _, lic := range []string{"proprietary", "mit", "apache-2.0", "modified-mit"} {
		add("/models/license/"+lic, "monthly", "0.6")
	}
	b.WriteString("</urlset>\n")

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(b.String()))
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	body := `# Search engines and AI answer engines are welcome.
User-agent: *
Allow: /
Disallow: /api/
Disallow: /audit
Disallow: /go/
Disallow: /admin
Disallow: /metrics

User-agent: GPTBot
Allow: /

User-agent: OAI-SearchBot
Allow: /

User-agent: ClaudeBot
Allow: /

User-agent: Claude-SearchBot
Allow: /

User-agent: PerplexityBot
Allow: /

User-agent: Google-Extended
Allow: /

User-agent: CCBot
Allow: /

User-agent: Applebot-Extended
Allow: /

Sitemap: ` + base + `/sitemap.xml
`
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(body))
}

func (s *Server) handleChanges(w http.ResponseWriter, r *http.Request) {
	setPublicCache(w)
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	q := r.URL.Query()
	slug := q.Get("station")
	kind := q.Get("kind")
	items := make([]app.Change, 0, len(d.Changes))
	source := d.Changes
	if q.Get("lang") == "en" {
		source = d.ChangesEN
	}
	for _, c := range source {
		if slug != "" && c.StationSlug != slug {
			continue
		}
		if kind != "" && kind != "all" && c.Kind != kind {
			continue
		}
		items = append(items, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":         items,
		"total":         len(items),
		"dataUpdatedAt": d.TakenAt,
	})
}

package api

import (
	"html/template"
	"net/http"
	"strings"
)

// Guide is a long-form article rendered both server-side (crawlers) and via the
// API (React). One source of truth keeps the two views identical.
type Guide struct {
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	Desc     string        `json:"desc"`
	Updated  string        `json:"updated"`
	Minutes  int           `json:"minutes"`
	Body     template.HTML `json:"-"`
	BodyHTML string        `json:"bodyHtml"`
}

var guides = []Guide{
	{
		Slug:    "how-to-choose-relay",
		Title:   "如何挑选大模型 API 中转站：一份 10 分钟验证清单",
		Desc:    "从真实性、可用性、计费口径到退款条款，给出可执行的检查顺序，以及五种最常见的低价套路。",
		Updated: "2026-11-19",
		Minutes: 7,
		Body: template.HTML(`<p>选中转站不是选“最便宜”，而是选“出了问题能算清楚、能退得掉”。下面这份清单按顺序做，10 分钟足够排除大部分坑。</p>

<h2>先问三个问题</h2>
<ol>
<li><strong>它是不是真的在转发官方模型？</strong> 看签名、模型回显、行为指纹，而不是看首页宣传。</li>
<li><strong>它稳不稳？</strong> 看 7 天可用率与延迟抖动，而不是看一张“99.99%”的海报。</li>
<li><strong>它算不算得清？</strong> 看汇率、分组倍率、分词口径与缓存规则是否能对上真实扣费。</li>
</ol>

<h2>10 分钟验证清单</h2>
<table class="index-table">
<thead><tr><th>检查项</th><th>怎么做</th><th>通过标准</th></tr></thead>
<tbody>
<tr><td>域名年龄</td><td>Whois 或本站的运营天数标签</td><td>≥ 6 个月；新站只充小额</td></tr>
<tr><td>模型回显</td><td>请求里指定模型，看响应 model 字段</td><td>与请求一致；不一致可能被替换</td></tr>
<tr><td>用量上报</td><td>发一条固定 prompt，看 usage.prompt_tokens</td><td>两次相同请求 token 数一致</td></tr>
<tr><td>缓存透传</td><td>同一请求连发两次，看 cached_tokens</td><td>第二次出现缓存命中且扣费下降</td></tr>
<tr><td>thinking 签名</td><td>Claude 模型开 thinking，看返回块里的 signature</td><td>字段存在且长度稳定；缺失可能被剥离</td></tr>
<tr><td>真实倍率</td><td>充值 1 元，跑 1 美元额度，看实际扣费</td><td>扣费 = 官方价 × 汇率 × 倍率</td></tr>
<tr><td>退款条款</td><td>问客服“未使用余额能否原路退”</td><td>有书面规则，不是口头承诺</td></tr>
</tbody>
</table>
<p>本站的 <a href="/audit">接口检测</a> 会自动完成其中四项（回显、usage、分词一致性、缓存），并把结果存成可分享的报告。</p>

<h2>五种最常见的低价套路</h2>
<ul>
<li><strong>低价分组混发：</strong> 首页展示 0.1x 分组，实际给你的是另一条更便宜、更不稳定的上游。看分组，不看首页价。</li>
<li><strong>吞掉缓存：</strong> 官方 cache read 只要 0.1x，中转站按全价扣，长上下文场景成本能翻几倍。</li>
<li><strong>分词膨胀：</strong> 换一个 tokenizer，同一段文字多算 10–30%，倍率看起来低，实际更贵。</li>
<li><strong>假签名：</strong> 返回一个格式对但内容不对的 signature，或者干脆不返回 thinking 块。</li>
<li><strong>限时倍率：</strong> 充值后悄悄调高分组倍率，等你发现时余额已经消耗大半。本站的<a href="/changes">变更雷达</a>会记录价格调整。</li>
</ul>

<h2>商务条款同样重要</h2>
<p>退款政策、发票类型、最低充值额决定了你的“退出成本”。一家价格贵 10% 但支持按量退款、能开专票的站，通常比便宜 30% 但“充值不退”的站更适合生产环境。</p>

<h2>推荐的下单顺序</h2>
<ol>
<li>先用 <a href="/">榜单</a> 里可用性 ≥ 99%、纯度“官方直连”的站点；</li>
<li>充最低额度（通常 ¥5–¥20），用临时 Key 跑一遍检测；</li>
<li>连续观察 3–7 天的延迟与扣费，再决定是否放生产流量；</li>
<li>同时保留一家低价备份，主站故障时切换。</li>
</ol>
<p>需要横向比较时用<a href="/compare">对比页</a>，它会把可用性、速度、稳定性、纯度与商务条款放在同一张表里。</p>`),
	},
	{
		Slug:    "multiplier-math",
		Title:   "倍率、汇率与缓存：中转站计费到底怎么算",
		Desc:    "用三个公式拆开“0.5 倍率”的宣传话术，算出每百万 token 的真实人民币成本，并解释缓存与分词膨胀的影响。",
		Updated: "2026-11-19",
		Minutes: 6,
		Body: template.HTML(`<p>中转站的报价页通常只写“汇率”和“倍率”，但真实成本由四个因子相乘决定。任何一个因子被藏起来，你的成本就会被低估。</p>

<h2>公式</h2>
<pre><code>真实单价（¥ / 1M tokens）
  = 官方单价（$ / 1M）
  × 结算汇率（¥ / $1）
  × 分组倍率
  × 分词膨胀系数</code></pre>
<p>其中“结算汇率”不是银行汇率，而是站点定义的“1 美元额度卖多少人民币”。常见范围是 1.0–2.2。</p>

<h2>一个具体例子</h2>
<p>Claude Opus 5.5 官方价是 $4 / $20（输入 / 输出，每百万 token）。某站汇率 1.5，分组倍率 0.4：</p>
<table class="index-table">
<thead><tr><th>项目</th><th>计算</th><th>结果</th></tr></thead>
<tbody>
<tr><td>输入</td><td>4 × 1.5 × 0.4</td><td>¥2.40 / 1M</td></tr>
<tr><td>输出</td><td>20 × 1.5 × 0.4</td><td>¥12.00 / 1M</td></tr>
<tr><td>若分词膨胀 1.2x</td><td>再乘 1.2</td><td>¥2.88 / ¥14.40</td></tr>
</tbody>
</table>
<p>所以“0.4 倍率”在膨胀 1.2x 时实际是 0.48 倍率。本站的价格列已经包含膨胀系数，但仍建议自己跑一次小额验证。</p>

<h2>缓存为什么重要</h2>
<p>官方对缓存命中通常只收 0.1x。一段 50 万 token 的代码仓上下文，如果每天重复调用 100 次：</p>
<ul>
<li>缓存正常：100 次里 99 次按 0.1x 计费；</li>
<li>缓存被吞：每次都按全价，成本约为前者的 9 倍。</li>
</ul>
<p>判断方法：同一请求连发两次，第二次的 usage 里应出现 cached_tokens，且扣费明显下降。本站的<a href="/audit">接口检测</a>会把“缓存透传”单独列为一项。</p>

<h2>三种常见话术</h2>
<ul>
<li><strong>“0.8 折” vs “0.8 倍”：</strong> 前者是原价的 80%，后者是官方价 × 0.8，二者一样；但有些站把“0.2 倍率”写成“2 折”，实际是 20%。</li>
<li><strong>“按次计费”：</strong> 部分站对长上下文按请求次数收费，单价看起来极低，实际随上下文长度线性上涨。</li>
<li><strong>“赠送余额”：</strong> 赠送部分往往不能退款、不能开发票，且可能限制分组。</li>
</ul>

<h2>自己算一遍</h2>
<p>用<a href="/compare">横向对比</a>看同一模型在不同站点的实价，或在<a href="/models">模型目录</a>里按模型反查全网最低价。价格是我们计算的，不参与评分，最终以站点实际扣费为准。</p>`),
	},
	{
		Slug:    "signature-verification",
		Title:   "thinking 签名与“降智”：怎么判断一个 Claude 中转是不是真的",
		Desc:    "解释 thinking signature 是什么、中转站会怎么剥离或伪造，以及一套不依赖官方密钥的验证思路与它的局限。",
		Updated: "2026-11-19",
		Minutes: 8,
		Body: template.HTML(`<p>“降智”通常不是模型变笨，而是你拿到的根本不是那个模型。Claude 的 thinking 签名是目前最容易观测的一个信号，但它不是万能钥匙。</p>

<h2>thinking 签名是什么</h2>
<p>开启 extended thinking 后，Claude 会在响应里返回 thinking 块，其中包含一段由服务端生成、客户端无法伪造的签名串。它是一段不透明的加密数据，用于让上游校验这段思考确实由该模型产生。</p>
<p>常见的误解是“64 字节私钥签名”。它并没有固定的 64 字节结构，长度会随实现变化；把它当作“长度校验”并不可靠。</p>

<h2>中转站会做什么</h2>
<ul>
<li><strong>剥离 thinking：</strong> 为了省 token 或兼容旧客户端，直接丢弃 thinking 块，签名自然消失。</li>
<li><strong>替换模型：</strong> 用更便宜的模型回答，签名要么没有，要么与请求的模型不匹配。</li>
<li><strong>截断上下文：</strong> 长上下文请求被悄悄裁剪，输出质量下降但计费照旧。</li>
<li><strong>伪造字段：</strong> 返回一个格式正确但内容随机的 signature，普通客户端无法分辨。</li>
</ul>

<h2>可操作的验证顺序</h2>
<ol>
<li><strong>请求里显式开启 thinking</strong>，确认响应包含 thinking 块；</li>
<li><strong>检查 signature 字段</strong>是否存在、是否稳定（同一 prompt 多次调用应持续存在）；</li>
<li><strong>核对模型回显</strong>，响应 model 必须与请求一致；</li>
<li><strong>做行为指纹</strong>：同一道需要多步推理的题，真模型与蒸馏模型的表现差异通常可复现；</li>
<li><strong>交叉验证</strong>：用官方端点跑同一 prompt，比较签名与输出结构。</li>
</ol>
<p>本站的<a href="/audit">接口检测</a>覆盖第 2、3 步以及流式完整性、usage 与缓存；行为指纹与签名交叉验证仍在路线图上，我们不会把它包装成“已完成”。</p>

<h2>为什么不能只看签名</h2>
<p>签名只能证明“这段 thinking 来自官方”，不能证明“这次回答没有被降级”。有些站会把简单请求转发给官方、复杂请求转发给廉价上游，单次抽检很容易漏掉。因此我们建议：</p>
<ul>
<li>在不同时间段多测几次，而不是只测一次；</li>
<li>同时看可用性、延迟与价格，异常便宜往往伴随异常上游；</li>
<li>把签名、回显、缓存、分词一致性放在一起判断，而不是迷信单一指标。</li>
</ul>

<h2>结论</h2>
<p>签名是必要不充分的证据。它的价值在于“缺失时几乎可以确定有问题”，而不是“存在时就一定没问题”。把这一点想清楚，就不会被“签名率 100%”的宣传带走。</p>
<p>相关阅读：<a href="/methodology">评分方法</a> 解释了纯度为什么不参与排名；<a href="/changes">变更雷达</a> 会记录纯度抽检异常。</p>`),
	},
}

func init() {
	for i := range guides {
		guides[i].BodyHTML = string(guides[i].Body)
	}
}

func guideBySlug(slug string) *Guide {
	for i := range guides {
		if guides[i].Slug == slug {
			return &guides[i]
		}
	}
	return nil
}

func guidesIndexHTML() template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>指南</h1><p>关于中转站选择、计费与验真的长文，全部可被搜索引擎与 AI 引擎直接读取。</p><ul>`)
	for _, g := range guides {
		b.WriteString(`<li><a href="/guides/` + escHTML(g.Slug) + `">` + escHTML(g.Title) + `</a> — ` + escHTML(g.Desc) + `</li>`)
	}
	b.WriteString(`</ul></article>`)
	return template.HTML(b.String())
}

func guideBodyHTML(g *Guide) template.HTML {
	return template.HTML(`<article><h1>` + escHTML(g.Title) + `</h1>` +
		`<p class="guide-meta">更新于 ` + escHTML(g.Updated) + ` · 约 ` + escHTML(itoa(g.Minutes)) + ` 分钟阅读</p>` +
		string(g.Body) + `</article>`)
}

func (s *Server) handleGuidesAPI(w http.ResponseWriter, r *http.Request) {
	items := make([]Guide, 0, len(guides))
	for _, g := range guides {
		g.BodyHTML = ""
		g.Body = ""
		items = append(items, g)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleGuideAPI(w http.ResponseWriter, r *http.Request) {
	g := guideBySlug(r.PathValue("slug"))
	if g == nil {
		writeError(w, http.StatusNotFound, "guide not found")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

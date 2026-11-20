package api

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// AuditCheckDoc is the long-form explanation behind one audit check.
type AuditCheckDoc struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Short  string `json:"short"`
	Why    string `json:"why"`
	How    string `json:"how"`
	Pass   string `json:"pass"`
	Warn   string `json:"warn"`
	Fail   string `json:"fail"`
	Limit  string `json:"limit"`
}

var auditCheckDocs = []AuditCheckDoc{
	{
		Key: "reachable", Label: "端点可达",
		Short: "确认 Base URL 能建立 TLS 连接并返回 2xx。",
		Why:   "连不上时，后面的模型回显、usage、缓存全部无从谈起；先把网络与证书问题排除。",
		How:   "向 {baseUrl}/chat/completions 发一条最小请求，记录 HTTP 状态、TLS 握手与总耗时。",
		Pass:  "HTTP 2xx，TLS 证书有效，响应在超时内返回。",
		Warn:  "HTTP 2xx 但耗时接近超时，或证书链需要额外中间证书。",
		Fail:  "连接被拒绝、DNS 解析失败、证书错误，或返回 4xx/5xx。",
		Limit: "只验证单次可达，不代表高峰期或长连接下的稳定性。",
	},
	{
		Key: "auth", Label: "鉴权通过",
		Short: "区分「Key 无效」和「站点故障」。",
		Why:   "401/403 通常是你自己的 Key 问题，而不是站点不可用；把它单独标记能避免误判。",
		How:   "看响应状态码与错误体：401/403 归为鉴权失败，其余错误归为上游错误。",
		Pass:  "请求被接受并返回内容。",
		Warn:  "403 且错误体提示地区、模型或分组限制。",
		Fail:  "401，或错误体明确为 invalid api key。",
		Limit: "不同站点的错误码并不统一，个别站点会用 200 + 错误 JSON 包装失败。",
	},
	{
		Key: "stream", Label: "流式完整性",
		Short: "检查 SSE 增量与 [DONE] 结束标记。",
		Why:   "流式被截断会让客户端长时间挂起；缺少 [DONE] 时，很多 SDK 会一直等待。",
		How:   "以 stream=true 请求，统计 data: 行、delta.content 与 [DONE]。",
		Pass:  "收到至少一个内容增量，并以 [DONE] 结束。",
		Warn:  "有内容但缺少 [DONE]，或增量之间间隔异常。",
		Fail:  "返回完整 JSON 而非 SSE，或流在内容前中断。",
		Limit: "不校验内容质量；只判断协议层是否完整。",
	},
	{
		Key: "model_echo", Label: "模型回显",
		Short: "响应里的 model 是否与请求一致。",
		Why:   "中转站替换模型时，最常见的迹象就是回显了另一个模型名。",
		How:   "请求指定模型，读取响应体或流式 chunk 中的 model 字段并对比。",
		Pass:  "完全一致，或仅大小写/别名差异。",
		Warn:  "回显为别名，但无法从公开信息确认映射关系。",
		Fail:  "回显为另一个模型，或完全不返回 model 字段。",
		Limit: "回显可以伪造；需要结合 thinking 签名与行为指纹一起判断。",
	},
	{
		Key: "usage", Label: "用量上报",
		Short: "usage 是否存在且字段合理。",
		Why:   "没有 usage，就无法核对计费；部分站点会用自己的分词口径虚报 token。",
		How:   "读取 usage.prompt_tokens / completion_tokens，检查是否为正数。",
		Pass:  "usage 存在，prompt_tokens 与 completion_tokens 均大于 0。",
		Warn:  "usage 存在但只有总量，缺少明细。",
		Fail:  "完全没有 usage 字段，或恒为 0。",
		Limit: "usage 由站点计算，无法仅凭它判断分词是否膨胀。",
	},
	{
		Key: "token_consistency", Label: "分词一致性",
		Short: "两次相同请求的 prompt_tokens 是否一致。",
		Why:   "同一段文本在两次调用里被算出不同 token 数，说明中间可能换了 tokenizer 或改写了请求。",
		How:   "用完全相同的 prompt 连发两次，比较 prompt_tokens。",
		Pass:  "两次完全一致。",
		Warn:  "差异在 1–2 个 token 内（可能受隐藏系统提示影响）。",
		Fail:  "差异明显，或第二次直接不返回 usage。",
		Limit: "不能区分「站点改写了提示词」与「上游本身不稳定」。",
	},
	{
		Key: "cache", Label: "缓存透传",
		Short: "第二次相同请求是否报告 cached_tokens。",
		Why:   "官方缓存通常只收 0.1x；中转站吞掉缓存，会把长上下文成本放大近 10 倍。",
		How:   "同一请求连发两次，读取第二次 usage 里的 prompt_tokens_details.cached_tokens。",
		Pass:  "第二次 cached_tokens > 0。",
		Warn:  "有缓存字段但为 0，或站点声明「不支持缓存」。",
		Fail:  "不存在缓存字段，且扣费仍按全价。",
		Limit: "缓存命中受上游策略影响；一次未命中不等于站点一定吞缓存。",
	},
	{
		Key: "ttft", Label: "首字延迟",
		Short: "从请求发出到第一个内容增量的时间。",
		Why:   "TTFT 决定交互体感，也是判断站点是否在排队或转发的直观指标。",
		How:   "记录请求开始到首个 delta.content 的时间。",
		Pass:  "小于 1500ms。",
		Warn:  "1500–4000ms。",
		Fail:  "大于 4000ms 或直到流结束都没有内容。",
		Limit: "单次测量受网络与上游负载影响；应多测几次取中位数。",
	},
}

func (s *Server) handleAuditChecks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": auditCheckDocs, "count": len(auditCheckDocs)})
}

func (s *Server) handleAuditCheck(w http.ResponseWriter, r *http.Request) {
	c := auditCheckByKey(r.PathValue("key"))
	if c == nil {
		writeError(w, http.StatusNotFound, "check not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func auditCheckByKey(key string) *AuditCheckDoc {
	for i := range auditCheckDocs {
		if auditCheckDocs[i].Key == key {
			return &auditCheckDocs[i]
		}
	}
	return nil
}

func auditChecksBody() template.HTML {
	var b strings.Builder
	b.WriteString(`<article><h1>接口检测项目</h1><p>每一项检测都在回答一个具体问题；下面是 8 项检查的判定标准与局限。</p><ul>`)
	for _, c := range auditCheckDocs {
		fmt.Fprintf(&b, `<li><a href="/audit/%s">%s</a> — %s</li>`, escHTML(c.Key), escHTML(c.Label), escHTML(c.Short))
	}
	b.WriteString(`</ul></article>`)
	return template.HTML(b.String())
}

func auditCheckBody(c *AuditCheckDoc) template.HTML {
	var b strings.Builder
	fmt.Fprintf(&b, `<article><h1>%s</h1><p>%s</p>`, escHTML(c.Label), escHTML(c.Short))
	fmt.Fprintf(&b, `<h2>为什么重要</h2><p>%s</p>`, escHTML(c.Why))
	fmt.Fprintf(&b, `<h2>怎么测</h2><p>%s</p>`, escHTML(c.How))
	b.WriteString(`<h2>判定标准</h2><table class="index-table"><thead><tr><th>结果</th><th>条件</th></tr></thead><tbody>`)
	fmt.Fprintf(&b, `<tr><td>通过</td><td>%s</td></tr>`, escHTML(c.Pass))
	fmt.Fprintf(&b, `<tr><td>观察</td><td>%s</td></tr>`, escHTML(c.Warn))
	fmt.Fprintf(&b, `<tr><td>失败</td><td>%s</td></tr>`, escHTML(c.Fail))
	b.WriteString(`</tbody></table>`)
	fmt.Fprintf(&b, `<h2>局限</h2><p>%s</p>`, escHTML(c.Limit))
	b.WriteString(`<p><a href="/audit">打开检测工具</a> · <a href="/methodology">评分方法</a></p></article>`)
	return template.HTML(b.String())
}

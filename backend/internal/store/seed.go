package store

import (
	"math"
	"math/rand"
	"strings"
	"time"
)

type seedStation struct {
	Slug      string
	Name      string
	Domain    string
	Tagline   string
	Rate      float64
	MinTopup  float64
	Payments  []string
	Invoice   bool
	Refund    string
	Claimed   bool
	DaysOld   int
	Quality   float64
	Purity    float64
	Services  string
	NewDomain bool
}

// seedLogos maps demo stations to bundled brand icons (fetched once from the
// public favicon service). Real deployments can instead leave this empty and
// let the API proxy fetch /api/v1/favicon/{slug} at runtime.
var seedLogos = map[string]string{
	"aihubmix":     "/logos/aihubmix.png",
	"302ai":        "/logos/302ai.png",
	"siliconflow":  "/logos/siliconflow.png",
	"openrouter":   "/logos/openrouter.png",
	"packyapi":     "/logos/packyapi.png",
	"qiniu":        "/logos/qiniu.png",
	"ppio":         "/logos/ppio.png",
	"closeai":      "/logos/closeai.ico",
	"dmxapi":       "/logos/dmxapi.png",
	"bailian":      "/logos/bailian.png",
	"ofoxai":       "/logos/ofoxai.png",
	"ohmygpt":      "/logos/ohmygpt.png",
	"weelinking":   "/logos/weelinking.png",
	"n1nai":        "/logos/n1nai.png",
	"apiyi":        "/logos/apiyi.png",
	"yunwu":        "/logos/yunwu.png",
	"laozhang":     "/logos/laozhang.png",
	"poixe":        "/logos/poixe.png",
	"apikl":        "/logos/apikl.png",
	"zenmux":       "/logos/zenmux.png",
	"gptgod":       "/logos/gptgod.png",
	"uiuiapi":      "/logos/uiuiapi.png",
	"ferryapi":     "/logos/ferryapi.png",
	"xuanshuapi":   "/logos/xuanshuapi.png",
}

var seedStations = []seedStation{
	{"aihubmix", "AIHubMix", "aihubmix.com", "官转聚合 · 企业合规主体", 1.55, 10, []string{"支付宝", "微信", "USDT"}, true, "余额 7 天原路退", false, 420, 0.96, 0.96, "claude,openai,gemini", false},
	{"302ai", "302.AI", "302.ai", "全球模型聚合 · 企业网关", 1.75, 10, []string{"支付宝", "微信", "USDT"}, true, "按量可退", false, 920, 0.94, 0.95, "openai,gemini,claude", false},
	{"siliconflow", "硅基流动", "siliconflow.cn", "国产自建推理集群", 1.00, 10, []string{"支付宝", "微信"}, true, "企业实名保障", false, 980, 0.92, 0.93, "deepseek,qwen,moonshot", false},
	{"openrouter", "OpenRouter", "openrouter.ai", "多云路由 · 官方转售", 2.10, 5, []string{"Stripe", "信用卡"}, false, "Unused 余额可退", false, 1260, 0.93, 0.94, "openai,gemini,claude", false},
	{"packyapi", "PackyAPI", "packyapi.com", "Claude Code 专线", 1.45, 10, []string{"支付宝", "微信"}, false, "7 天工单可退", false, 640, 0.90, 0.92, "claude", false},
	{"qiniu", "七牛云 AI", "qiniu.com", "企业高可用骨干网", 1.20, 10, []string{"支付宝", "微信", "对公"}, true, "企业 SLA", false, 4200, 0.91, 0.92, "claude,deepseek", false},
	{"ppio", "PPIO 派欧云", "ppinfra.com", "分布式 GPU 算力池", 1.00, 10, []string{"支付宝", "微信"}, true, "对公 SLA", false, 1580, 0.89, 0.90, "deepseek,qwen", false},
	{"closeai", "CloseAI", "closeai-asia.com", "Azure / AWS 企业官转", 1.70, 10, []string{"支付宝", "微信"}, true, "工单退余款", false, 1120, 0.88, 0.91, "openai,claude", false},
	{"dmxapi", "DMXAPI", "dmxapi.com", "三大原生协议企业直连", 1.60, 10, []string{"支付宝", "USDT"}, false, "余额原路退", false, 790, 0.86, 0.90, "claude,openai,gemini", false},
	{"bailian", "阿里云百炼", "aliyun.com", "Qwen / DeepSeek 企业专线", 1.00, 10, []string{"支付宝", "对公"}, true, "阿里云账单", false, 5100, 0.93, 0.94, "deepseek,qwen", false},
	{"ofoxai", "OfoxAI", "ofox.ai", "美西 CN2 加速原生官转", 1.90, 10, []string{"支付宝", "USDT"}, false, "7 日未用可退", false, 510, 0.87, 0.89, "claude,openai", false},
	{"ohmygpt", "OhMyGPT", "ohmygpt.com", "OpenAI / Claude 多区直连", 1.80, 10, []string{"支付宝", "微信"}, false, "工单退余款", false, 1150, 0.85, 0.88, "claude,openai", false},
	{"weelinking", "Weelinking", "weelinking.com", "高可用容灾 · 特惠组", 1.45, 10, []string{"支付宝", "微信"}, false, "随时退余款", false, 580, 0.83, 0.87, "claude,openai,gemini", false},
	{"n1nai", "n1n.ai", "n1n.ai", "低延迟全球路由", 1.65, 10, []string{"支付宝", "USDT"}, false, "1 元起充可退", false, 430, 0.80, 0.86, "claude", false},
	{"apiyi", "API易", "apiyi.com", "Claude / GPT 直连低价", 1.55, 10, []string{"支付宝", "微信"}, false, "7 天内可退", false, 690, 0.78, 0.86, "claude,openai", false},
	{"yunwu", "云雾 API", "yunwu.ai", "特惠组与官转组隔离", 1.35, 5, []string{"支付宝", "微信"}, false, "小额起充", false, 720, 0.72, 0.85, "claude,openai", false},
	{"laozhang", "老板 AI", "laozhang.ai", "海外企业卡直连", 1.85, 10, []string{"支付宝", "微信"}, false, "工单协商退", false, 660, 0.68, 0.84, "claude,openai", false},
	{"poixe", "Poixe AI", "poixe.com", "多协议统一网关 · 企业结算", 1.90, 10, []string{"支付宝", "微信"}, true, "余额可退", false, 300, 0.86, 0.90, "claude,openai,gemini", false},
	{"apikl", "API 快连", "apikl.ai", "Claude Code / Codex 直连", 1.50, 10, []string{"支付宝", "微信"}, false, "7 天未用可退", false, 380, 0.88, 0.91, "claude,openai", false},
	{"zenmux", "ZenMux", "zenmux.ai", "OpenAI / Anthropic 协议转换", 2.00, 5, []string{"Stripe", "信用卡"}, false, "按量可退", false, 700, 0.90, 0.93, "openai,claude,gemini", false},
	{"gptgod", "GPTGOD", "gptgod.online", "低价逆向渠道 · 稳定性一般", 0.90, 1, []string{"支付宝"}, false, "小额起充", false, 520, 0.70, 0.80, "openai,claude", false},
	{"uiuiapi", "UiUiAPI", "uiuiapi.com", "官方渠道聚合 · 311 模型", 1.10, 10, []string{"支付宝", "微信"}, false, "余额可退", false, 610, 0.85, 0.88, "openai,claude,gemini", false},
	{"ferryapi", "FerryAPI", "ferryapi.io", "失败请求不扣费", 1.20, 5, []string{"Stripe"}, false, "失败不扣费", false, 260, 0.80, 0.86, "openai,deepseek,claude", false},
	{"xuanshuapi", "玄枢 API", "xuanshuapi.com", "根路径直出 Anthropic / Gemini", 1.40, 10, []string{"支付宝", "微信"}, false, "工单退款", false, 210, 0.82, 0.87, "claude,gemini,openai", false},
	{"mkeai", "MKEAI", "mkeai.com", "社区论坛 + 中转混合", 1.30, 10, []string{"支付宝"}, false, "社区规则", false, 320, 0.75, 0.83, "openai,deepseek", false},
	{"wolf-cheap", "WolfCheap", "wolfcheap.invalid", "虚构黑榜样例 · 假签名 + 1.62x 暗扣", 0.40, 100, []string{"个人码高危"}, false, "个人码高危", false, 9, 0.18, 0.02, "claude,openai", true},
}

var seedModels = []Model{
	{ID: "claude-fable-5-2", Vendor: "anthropic", VendorName: "Anthropic", Name: "Claude Fable 5.2", Service: "claude", ContextWindow: 1000000, InputUSD: 8, OutputUSD: 40, ReleasedAt: date("2026-11-09"), CodingScore: 95.1, ReasoningScore: 95.0, Summary: "编程与长程代理旗舰，原生思考签名，1M 上下文。"},
	{ID: "claude-opus-5-5", Vendor: "anthropic", VendorName: "Anthropic", Name: "Claude Opus 5.5", Service: "claude", ContextWindow: 1000000, InputUSD: 4, OutputUSD: 20, ReleasedAt: date("2026-09-22"), CodingScore: 92.8, ReasoningScore: 92.4, Summary: "生产环境稳健主力，价格为 Fable 5.2 的一半。"},
	{ID: "claude-sonnet-5-5", Vendor: "anthropic", VendorName: "Anthropic", Name: "Claude Sonnet 5.5", Service: "claude", ContextWindow: 1000000, InputUSD: 2, OutputUSD: 10, ReleasedAt: date("2026-09-28"), CodingScore: 90.6, ReasoningScore: 89.2, Summary: "高吞吐补全首选，适合 IDE 实时场景。"},
	{ID: "claude-opus-4-8", Vendor: "anthropic", VendorName: "Anthropic", Name: "Claude Opus 4.8", Service: "claude", ContextWindow: 500000, InputUSD: 5, OutputUSD: 25, ReleasedAt: date("2026-06-18"), CodingScore: 90.2, ReasoningScore: 90.9, Summary: "上一代旗舰，仍是最常见的中转验真基准模型。"},
	{ID: "gpt-6.2-astra", Vendor: "openai", VendorName: "OpenAI", Name: "GPT-6.2 Astra", Service: "openai", ContextWindow: 1050000, InputUSD: 8, OutputUSD: 40, ReleasedAt: date("2026-11-17"), CodingScore: 94.6, ReasoningScore: 95.4, Summary: "多文件自治重构旗舰，原生 1.05M 窗口。"},
	{ID: "gpt-6.2-sol", Vendor: "openai", VendorName: "OpenAI", Name: "GPT-6.2 Sol", Service: "openai", ContextWindow: 1050000, InputUSD: 2, OutputUSD: 10, ReleasedAt: date("2026-11-17"), CodingScore: 91.8, ReasoningScore: 92.0, Summary: "与 Astra 同日发布的高速生产力型号。"},
	{ID: "gpt-6.1-terra", Vendor: "openai", VendorName: "OpenAI", Name: "GPT-6.1 Terra", Service: "openai", ContextWindow: 400000, InputUSD: 0.6, OutputUSD: 2.4, ReleasedAt: date("2026-08-14"), CodingScore: 88.7, ReasoningScore: 88.1, Summary: "低成本批处理型号，适合分类与抽取。"},
	{ID: "gemini-4-carbon", Vendor: "google", VendorName: "Google", Name: "Gemini 4 Carbon", Service: "gemini", ContextWindow: 1000000, InputUSD: 2, OutputUSD: 10, ReleasedAt: date("2026-10-28"), CodingScore: 93.2, ReasoningScore: 95.5, Summary: "原生多模态推理旗舰，隐式缓存折扣高。"},
	{ID: "gemini-4-hydrogen", Vendor: "google", VendorName: "Google", Name: "Gemini 4 Hydrogen", Service: "gemini", ContextWindow: 1000000, InputUSD: 1.25, OutputUSD: 5, ReleasedAt: date("2026-11-12"), CodingScore: 91.0, ReasoningScore: 91.4, FreeTier: true, FreeNote: "AI Studio 免费层 1,000 次/天（需绑定结算账号）", Summary: "低延迟前端生成型号，单价低。"},
	{ID: "grok-4-7", Vendor: "xai", VendorName: "xAI", Name: "Grok 4.7", Service: "xai", ContextWindow: 500000, InputUSD: 2, OutputUSD: 6, ReleasedAt: date("2026-09-21"), CodingScore: 89.5, ReasoningScore: 90.8, Summary: "支持多路子代理并行推理。"},
	{ID: "deepseek-v4-1-flash", Vendor: "deepseek", VendorName: "DeepSeek", Name: "DeepSeek V4.1 Flash", Service: "deepseek", ContextWindow: 1000000, InputUSD: 0.3, OutputUSD: 1.2, ReleasedAt: date("2026-10-17"), CodingScore: 88.5, ReasoningScore: 89.6, FreeTier: true, FreeNote: "新账号赠送 500 万 tokens，30 天有效", Summary: "开源权重高能效版，国产算力吞吐高。"},
	{ID: "kimi-k3-1-pro", Vendor: "moonshot", VendorName: "Moonshot", Name: "Kimi K3.1 Pro", Service: "moonshot", ContextWindow: 1000000, InputUSD: 2, OutputUSD: 10, ReleasedAt: date("2026-11-04"), CodingScore: 90.4, ReasoningScore: 90.2, Summary: "开放权重，超长代码仓上下文稳定。"},
	{ID: "qwen4-max", Vendor: "qwen", VendorName: "阿里通义", Name: "Qwen4-Max", Service: "qwen", ContextWindow: 1000000, InputUSD: 1.5, OutputUSD: 4.5, ReleasedAt: date("2026-11-01"), CodingScore: 89.4, ReasoningScore: 89.8, FreeTier: true, FreeNote: "阿里云百炼每日 100 万 tokens 免费额度", Summary: "万亿 MoE，中文结构化抽取标杆。"},
	{ID: "gpt-6.2-nano", Vendor: "openai", VendorName: "OpenAI", Name: "GPT-6.2 Nano", Service: "openai", ContextWindow: 400000, InputUSD: 0.15, OutputUSD: 0.6, ReleasedAt: date("2026-11-17"), CodingScore: 84.2, ReasoningScore: 85.0, Summary: "超低延迟小模型，适合高频分类与路由。"},
	{ID: "claude-haiku-5-5", Vendor: "anthropic", VendorName: "Anthropic", Name: "Claude Haiku 5.5", Service: "claude", ContextWindow: 500000, InputUSD: 0.8, OutputUSD: 4, ReleasedAt: date("2026-10-02"), CodingScore: 86.8, ReasoningScore: 86.1, Summary: "轻量 Claude，长上下文摘要与批处理便宜。"},
	{ID: "gemini-4-nitrogen", Vendor: "google", VendorName: "Google", Name: "Gemini 4 Nitrogen", Service: "gemini", ContextWindow: 1000000, InputUSD: 0.4, OutputUSD: 1.6, ReleasedAt: date("2026-11-12"), CodingScore: 87.9, ReasoningScore: 88.4, Summary: "低成本多模态型号，适合大规模文档处理。"},
	{ID: "deepseek-r2", Vendor: "deepseek", VendorName: "DeepSeek", Name: "DeepSeek R2", Service: "deepseek", ContextWindow: 256000, InputUSD: 0.5, OutputUSD: 2, ReleasedAt: date("2026-09-05"), CodingScore: 88.1, ReasoningScore: 91.2, Summary: "开源推理模型，数学与代码推理性价比高。"},
	{ID: "kimi-k3-turbo", Vendor: "moonshot", VendorName: "Moonshot", Name: "Kimi K3 Turbo", Service: "moonshot", ContextWindow: 2000000, InputUSD: 0.6, OutputUSD: 2.4, ReleasedAt: date("2026-10-20"), CodingScore: 87.6, ReasoningScore: 87.9, Summary: "2M 超长上下文，适合整仓代码与长文档。"},
	{ID: "qwen4-plus", Vendor: "qwen", VendorName: "阿里通义", Name: "Qwen4-Plus", Service: "qwen", ContextWindow: 1000000, InputUSD: 0.4, OutputUSD: 1.2, ReleasedAt: date("2026-10-08"), CodingScore: 86.4, ReasoningScore: 86.9, Summary: "中端主力，中文任务与函数调用稳定。"},
	{ID: "grok-4-7-mini", Vendor: "xai", VendorName: "xAI", Name: "Grok 4.7 Mini", Service: "xai", ContextWindow: 256000, InputUSD: 0.5, OutputUSD: 1.5, ReleasedAt: date("2026-10-01"), CodingScore: 84.9, ReasoningScore: 86.2, Summary: "轻量 Grok，适合实时对话与检索。"},
	{ID: "gpt-image-3", Vendor: "openai", VendorName: "OpenAI", Name: "GPT Image 3", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-10-30"), Category: "image", Summary: "文本渲染与多图一致性强的图像模型，约 $0.04/张。"},
	{ID: "dall-e-4", Vendor: "openai", VendorName: "OpenAI", Name: "DALL·E 4", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-08-21"), Category: "image", Summary: "创意构图与风格化插画，约 $0.03/张。"},
	{ID: "midjourney-v8", Vendor: "midjourney", VendorName: "Midjourney", Name: "Midjourney V8", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-09-12"), Category: "image", Summary: "审美与风格一致性领先，订阅制约 $10/月起。"},
	{ID: "flux-2-pro", Vendor: "blackforest", VendorName: "Black Forest Labs", Name: "FLUX 2 Pro", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-07-18"), Category: "image", Summary: "开放权重生态，写实人像与局部重绘，约 $0.05/张。"},
	{ID: "stable-diffusion-4", Vendor: "stability", VendorName: "Stability AI", Name: "Stable Diffusion 4", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-06-30"), Category: "image", Summary: "可自建部署的开源图像模型，边际成本最低。"},
	{ID: "imagen-5", Vendor: "google", VendorName: "Google", Name: "Imagen 5", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-10-28"), Category: "image", Summary: "写实摄影与文字排版强，约 $0.06/张。"},
	{ID: "seedream-5", Vendor: "bytedance", VendorName: "字节跳动", Name: "Seedream 5", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-11-02"), Category: "image", Summary: "中文海报与电商图，约 ¥0.2/张。"},
	{ID: "qwen-image-2", Vendor: "qwen", VendorName: "阿里通义", Name: "Qwen-Image 2", Service: "image", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-09-25"), Category: "image", Summary: "开源中文图像模型，支持长文本渲染。"},
	{ID: "sora-3", Vendor: "openai", VendorName: "OpenAI", Name: "Sora 3", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-11-05"), Category: "video", Summary: "长镜头叙事与音画同步，按秒计费约 $0.30/s。"},
	{ID: "veo-4", Vendor: "google", VendorName: "Google", Name: "Veo 4", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-10-28"), Category: "video", Summary: "电影感运镜与 4K 输出，约 $0.35/s。"},
	{ID: "kling-3", Vendor: "kuaishou", VendorName: "快手", Name: "Kling 3.0", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-09-18"), Category: "video", Summary: "人物动作与中文提示词强，约 ¥1.5/s。"},
	{ID: "runway-gen5", Vendor: "runway", VendorName: "Runway", Name: "Runway Gen-5", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-08-08"), Category: "video", Summary: "镜头控制与风格迁移，订阅制。"},
	{ID: "pika-3", Vendor: "pika", VendorName: "Pika", Name: "Pika 3", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-07-22"), Category: "video", Summary: "短视频特效与模板化生成，价格低。"},
	{ID: "hailuo-03", Vendor: "minimax", VendorName: "MiniMax", Name: "Hailuo 03", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-10-10"), Category: "video", Summary: "国产短视频模型，人物一致性较好，约 ¥0.8/s。"},
	{ID: "wan-3", Vendor: "qwen", VendorName: "阿里通义", Name: "Wan 3.0", Service: "video", ContextWindow: 0, InputUSD: 0, OutputUSD: 0, ReleasedAt: date("2026-11-08"), Category: "video", Summary: "开源视频模型，可自建推理，支持图生视频。"},
}

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

var seedReviews = []struct {
	Station string
	Author  string
	Rating  int
	Body    string
	Days    int
}{
	{"aihubmix", "RustKernel", 5, "Claude 与 GPT 两条线都跑过，thinking 签名完整返回，缓存命中按 0.1x 扣。", 4},
	{"aihubmix", "claude-coder", 5, "发票开得挺快，晚高峰延迟略涨但没断过。", 11},
	{"aihubmix", "bytecat", 4, "整体稳定，控制台账单粒度一般，希望补分组用量导出。", 19},
	{"openrouter", "hexo-dev", 5, "多模型路由方便，Stripe 付款没遇到风控，余额退款 3 天到账。", 7},
	{"openrouter", "kernelpanic", 4, "价格偏高，但长上下文不截断，模型回显一致。", 24},
	{"siliconflow", "数据搬运工", 5, "国产开源模型价格低，DeepSeek 吞吐不错，接口兼容 OpenAI。", 5},
	{"packyapi", "tsuki", 4, "Claude Code 直连稳定，连续一周没遇到 429。", 9},
	{"bailian", "ops-zhang", 4, "企业账号走阿里云账单，合规和开票都省事。", 13},
	{"wolf-cheap", "匿名用户", 1, "注册 9 天的站，测了 20 次有 7 次直接 502，建议只充小额。", 2},
}

// BuildSeedSnapshot produces the deterministic demo dataset shared by the
// PostgreSQL seeder and the in-memory store.
func BuildSeedSnapshot() (*Snapshot, []Review) {
	rng := rand.New(rand.NewSource(20261118))
	now := time.Now().UTC().Truncate(time.Hour)
	snap := &Snapshot{TakenAt: now}

	modelByService := map[string][]Model{}
	for _, m := range seedModels {
		modelByService[m.Service] = append(modelByService[m.Service], m)
	}
	snap.Models = append(snap.Models, seedModels...)
	for i := range snap.Models {
		if snap.Models[i].Category == "" {
			snap.Models[i].Category = "text"
		}
	}

	var reviews []Review

	for i, st := range seedStations {
		stationID := int64(i + 1)
		snap.Stations = append(snap.Stations, Station{
			ID: stationID, Slug: st.Slug, Name: st.Name, Domain: st.Domain,
			BaseURL: "https://" + st.Domain + "/v1", Tagline: st.Tagline,
			RateCNY: st.Rate, MinTopupCNY: st.MinTopup, Payments: st.Payments,
			Invoice: st.Invoice, RefundPolicy: st.Refund, Claimed: st.Claimed,
			IconURL:  seedLogos[st.Slug],
			ListedAt: now.AddDate(0, 0, -st.DaysOld),
		})

		for _, svc := range strings.Split(st.Services, ",") {
			svc = strings.TrimSpace(svc)
			groupID := int64(len(snap.Groups) + 1)
			mult := math.Round((0.35+rng.Float64()*0.30)*100) / 100
			snap.Groups = append(snap.Groups, Group{
				ID: groupID, StationID: stationID, Slug: svc, Name: serviceLabel(svc) + " 默认组",
				Service: svc, Multiplier: mult,
			})

			for _, m := range modelByService[svc] {
				in := round2(m.InputUSD * st.Rate * mult)
				out := round2(m.OutputUSD * st.Rate * mult)
				cache := round2(in * 0.1)
				snap.GroupModels = append(snap.GroupModels, GroupModel{
					GroupID: groupID, ModelID: m.ID,
					InputCNY: &in, OutputCNY: &out, CacheReadCNY: &cache,
				})
			}

			groupQuality := clamp01(st.Quality + (rng.Float64()-0.5)*0.08)
			for k := 35 * 12; k >= 0; k-- {
				at := now.Add(-time.Duration(k) * 2 * time.Hour)
				hour := at.Hour()
				// One demo station degrades over the last week so the change
				// radar has a real score-regression event to report.
				effQuality := groupQuality
				if st.Slug == "n1nai" {
					daysAgo := float64(k) / 12.0
					if daysAgo < 7 {
						effQuality = clamp01(effQuality - (7-daysAgo)/7*0.15)
					}
				}
				peak := 1.0
				if hour >= 18 && hour < 24 {
					peak = 1.0 - 0.03*(1-effQuality)
				}
				p := (0.62 + effQuality*0.375) * peak
				roll := rng.Float64()
				status := "ok"
				latency := 0
				switch {
				case roll < p:
					status = "ok"
					latency = int(650 + (1-effQuality)*4200 + rng.Float64()*380)
				case roll < p+0.015:
					status = "slow"
					latency = int(650 + (1-effQuality)*4200 + 2600 + rng.Float64()*1200)
				default:
					status = "down"
				}
				snap.Probes = append(snap.Probes, Probe{
					GroupID: groupID, Status: status, LatencyMS: latency, ProbedAt: at,
				})
			}

			// Purity is only measured on Claude groups (thinking-signature
			// cross-check); other services are reported as "not measured".
			if svc == "claude" {
				for d := 35; d >= 0; d -= 3 {
					score := 12 + clamp01(st.Purity)*86 + (rng.Float64()-0.5)*8
					score = math.Max(0, math.Min(99, math.Round(score)))
					tier := "mixed"
					if score >= 80 {
						tier = "official"
					} else if score < 30 {
						tier = "counterfeit"
					}
					detected := ""
					if score >= 60 {
						detected = seedModels[0].ID
					}
					snap.Purity = append(snap.Purity, Purity{
						GroupID: groupID, Score: int(score), Tier: tier, DetectedModel: detected,
						CheckedAt: now.AddDate(0, 0, -d),
					})
				}
			}
		}
	}

	stationIDBySlug := map[string]int64{}
	for _, st := range snap.Stations {
		stationIDBySlug[st.Slug] = st.ID
	}

	// Seed a few price movements so the change feed has real history.
	modelService := map[string]string{}
	for _, m := range seedModels {
		modelService[m.ID] = m.Service
	}
	moves := []struct {
		slug, model string
		factor      float64
		daysAgo     int
	}{
		{"yunwu", "claude-opus-5-5", 0.85, 6},
		{"dmxapi", "gemini-4-carbon", 1.08, 8},
		{"laozhang", "gpt-6.2-sol", 0.88, 4},
		{"aihubmix", "gpt-6.2-astra", 0.92, 5},
		{"packyapi", "claude-fable-5-2", 1.07, 7},
		{"qiniu", "deepseek-v4-1-flash", 0.94, 9},
		{"openrouter", "gemini-4-hydrogen", 0.90, 3},
	}
	for _, mv := range moves {
		var groupID int64
		for _, g := range snap.Groups {
			if g.StationID == stationIDBySlug[mv.slug] && g.Service == modelService[mv.model] {
				groupID = g.ID
				break
			}
		}
		if groupID == 0 {
			continue
		}
		for _, gm := range snap.GroupModels {
			if gm.GroupID != groupID || gm.ModelID != mv.model || gm.InputCNY == nil || gm.OutputCNY == nil {
				continue
			}
			// factor is the new/old ratio: 0.85 means the price fell 15%.
			oldIn := round2(*gm.InputCNY / mv.factor)
			oldOut := round2(*gm.OutputCNY / mv.factor)
			later := mv.daysAgo - 2
			if later < 0 {
				later = 0
			}
			snap.Prices = append(snap.Prices,
				PricePoint{GroupID: groupID, ModelID: gm.ModelID, InputCNY: &oldIn, OutputCNY: &oldOut, RecordedAt: now.AddDate(0, 0, -mv.daysAgo)},
				PricePoint{GroupID: groupID, ModelID: gm.ModelID, InputCNY: gm.InputCNY, OutputCNY: gm.OutputCNY, RecordedAt: now.AddDate(0, 0, -later)},
			)
		}
	}

	for i, r := range seedReviews {
		reviews = append(reviews, Review{
			ID: int64(i + 1), Author: r.Author, Rating: r.Rating, Body: r.Body,
			Verified: true, CreatedAt: now.AddDate(0, 0, -r.Days),
		})
		reviews[len(reviews)-1].StationID = stationIDBySlug[r.Station]
	}

	return snap, reviews
}

func serviceLabel(s string) string {
	switch s {
	case "claude":
		return "Claude"
	case "openai":
		return "OpenAI"
	case "gemini":
		return "Gemini"
	case "xai":
		return "Grok"
	case "deepseek":
		return "DeepSeek"
	case "moonshot":
		return "Kimi"
	case "qwen":
		return "Qwen"
	}
	return s
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

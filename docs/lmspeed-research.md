# LMSpeed 调研笔记

抓取时间：2026-11-19。对象：https://lmspeed.net/ 及其主要栏目页。

## 站点结构

| 路径 | 作用 |
|---|---|
| `/` | 首页：赞助位、最新/最强模型表、入口 |
| `/model`、`/model/{slug}` | 模型目录与详情：官方价（带 publisher 链接）、上下文、许可证、发布时间、覆盖 provider 数、benchmark 表 |
| `/provider` | Provider 目录：726 家在列、8 个分类、赞助卡片、密集对比表 |
| `/provider/category/{relay,free,official,inference,china,...}` | 分类筛选，带数量 |
| `/leaderboard/{agents,coding,reasoning,knowledge,math,...}` | 按能力分的排行榜 |
| `/free-api-keys` | 免费额度目录 |
| `/sponsor` | 赞助报价页（`From $149/mo`、`$249/mo` 等档位） |
| `/methodology` | 方法论、版本号、数据来源 |
| `Submit` | 提交入口 |

## 值得借鉴的点

1. **Answer-first（当前答案）**
   排行榜顶部先用一段自然语言给出结论：“Among the currently visible models, X ranks #1 with 63.9 ± 8.1…”，再放表格。
   对 GEO 极友好：AI 引擎可以直接摘这一段，不用解析几十列表格。

2. **不确定性直接写进分数**
   `63.9±8.1`、`80% uncertainty range`。这比只给一个分数更诚实，也降低“榜单被当成绝对真理”的风险。

3. **数据来源与方法论被显式标注**
   页头同时出现：`Updated November 18, 2026`、`Methodology 3.0`、`Benchmark data from Benchize`。
   每条外部价格还带 `Official · publisher` 链接。

4. **证据/状态列**
   排行榜有 `Status`（Rated）与 `Evidence`（`3/4 dimensions · 8 families`），不是只给排名。
   Provider 表有 `Health`、`Tests`、`Last check` 三列，把“最近一次验证”变成一等公民。

5. **分类即筛选，带计数**
   `Signup credits 5 / Verified 48 / API Relays 85 / Free 26 / Official 24 / China 61 …`
   用户不用先搜索，点分类就能缩小范围。

6. **密集但可读的对比表**
   表头分组（Planning / Tool use / Execution / Recovery），列名是具体 benchmark，数字只在本列内可比；页面明确写了“bar 长度不可跨列比较”。

7. **赞助披露**
   赞助卡片标 `Sponsor`，首页有独立的 “Our sponsors” 区块与 `Your product here` 位，和自然结果分开。

8. **Ctrl K 全局搜索**
   顶栏固定 “Search… Ctrl K”，键盘优先。

9. **Provider 表字段**
   `Model makers / Contact / Categories / Health / Payment / Invoices / Refunds / Tests / Last check` —— 和我们的商务条款字段高度重合，可以补 `Health` 与 `Last check`。

## 导航对比

| LMSpeed | 我们 | 结论 |
|---|---|---|
| Models | 模型 | 一致 |
| Providers | 榜单（首页即目录） | 结构一致，但 LMSpeed 有独立目录页与分类计数；我们给首页筛选补了计数 |
| Free（免费 API Key） | 免费额度（模型免费层，在“更多”里） | 语义不同，不照搬；保持“模型免费层”定位 |
| Leaderboard | 排行榜 | 一致，且我们多了「当前答案」与 ± 区间 |
| Tools（下拉：检测等） | 检测 / 对比 / 变更 | 一致，只是我们平铺 |
| Sponsor | 无 | 涉及商务与标注合规，暂不做 |
| Submit | 提交站点（按钮） | 一致 |
| Search… Ctrl K | Ctrl K 全站搜索 | 已实现 |
| 面包屑 | 详情页有，栏目页没有 | 可统一，优先级低 |

结论：导航层级不需要重构，LMSpeed 真正值得拿的是**分类计数**与**全局搜索**，两者都已落地；独立 Provider 目录页与赞助位暂不做。

## 全站路由地图（来自 sitemap.xml）

LMSpeed 的 sitemap 暴露了完整功能面，且每个页面都有 en / zh-CN / ru 三个语言版本 + hreflang：

| 路由族 | 内容 |
|---|---|
| `/model`、`/model/{slug}` | 模型目录与详情（质量/速度排名、官方价、上下文、许可证、分类性能、对比入口） |
| `/model/license`、`/model/license/{license}` | 按许可证聚合的模型目录（Apache-2.0、Gemma、Llama、MIT 等） |
| `/provider`、`/provider/{slug}` | Provider 目录与详情（健康度、支付、发票、退款、Tests、Last check） |
| `/provider/category/{relay,free,official,inference,china,...}`、`/provider/verified`、`/provider/signup-credit` | 分类与标签筛选 |
| `/leaderboard/best-model-for-{agents,coding,reasoning,knowledge,math,multilingual,multimodal,instruction-following}` | 能力榜 |
| `/leaderboard/best-{throughput,latency}-models-{weekly,monthly}`、`/leaderboard/best-health-providers-{weekly,monthly}` | 周榜 / 月榜 |
| `/audit/{prompt-extraction,instruction-conflict,jailbreak-identity,context-window,tool-substitution,error-leakage,stream-integrity,web3-prompt-injection,latency-variance}` | 每项安全/协议检测的独立说明页 |
| `/free-api-keys` | 免费 Key 目录 |
| `/sponsor` | 赞助报价 |
| `/methodology`、`/submit` | 方法论与提交 |

## 功能矩阵

| 能力 | LMSpeed | 我们 | 结论 |
|---|---|---|---|
| 站点/Provider 目录 | 726 家、8 分类、健康/测试/最近检查列 | 18 家、筛选计数、商务条款 | 结构对齐，规模不同 |
| 模型详情 | 质量/速度排名、许可证、分类性能、± 区间 | 价格、报价、编程/推理分 | 缺许可证与分类性能 |
| 模型按许可证聚合 | 有 | 无 | 需要 license 字段，暂缓 |
| 能力榜 | 8 个维度 | 编程/推理/性价比 + 速度/稳定性 | 维度少 |
| 周榜/月榜 | 有 | 只有 14 天趋势 | 数据够但未做榜单 |
| 检测项说明页 | 9 个 | **8 个（本轮新增）** | 已对齐 |
| 免费 Key 目录 | 有 | 只有模型免费层 | 定位不同 |
| 多语言 | en/zh/ru + hreflang | 仅中文 | 暂不做 |
| 赞助位 | 有 | 无 | 合规考虑，不做 |

## 本次已借鉴

- 排行榜新增「当前答案」区块：一句自然语言给出综合分第一、可用率最高、速度最快、最低价，并附数据时间与口径；服务端预渲染同段文字，爬虫可直接读。
- 检测项说明页：8 个 `/audit/{key}` 独立页面（为什么重要 / 怎么测 / 判定标准 / 局限）。
- 许可证维度：模型带 license，新增 `/models/license` 与 `/models/license/{license}` 聚合页。
- 模型对比：`/compare/models` 同表对比两个模型的官方价、实价、能力分与许可证。
- 周榜 / 月榜：`/leaderboards/window` 按 7 / 30 天窗口取均值，历史不足时自动缩短并标注实际天数。
- 支付方式改为官方品牌字形（Simple Icons 的支付宝 / 微信 / Tether / Stripe / Visa / Mastercard）。
- 厂商图标改为官方品牌字形。
- 站点图标补齐运行时抓取能力：`FAVICON_PROXY_URL` 可让图标抓取走代理（仅图标，审计探测仍不走代理），新站点无内置图标时自动回落到 `/api/v1/favicon/{slug}`。

## 暂缓

- `±` 不确定性区间：需要把 Wilson 区间与速度/稳定性样本合成一个误差范围，统计口径要单独设计，不能拍脑袋。
- 模型许可证、publisher 链接、benchmark 家族覆盖度：需要补数据源字段。
- 免费 API Key 目录、赞助位与报价页：涉及商务与合规，先不做。
- Ctrl K 命令面板：纯前端可做，优先级低于数据质量。

## 结论

LMSpeed 的核心优势不是“页面多”，而是**结论先行 + 不确定性 + 来源可追溯 + 证据列**。这四点比再堆十个表格更值得抄。我们的 v2 评分已经有 Wilson 下界和样本数，下一步应该把“置信/样本”真正展示到界面上，而不是只藏在方法论里。

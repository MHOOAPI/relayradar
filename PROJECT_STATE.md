# RelayRadar 项目状态（交接文档）

> 用途：跨会话交接。新对话只需读这份文件 + 仓库即可继续，不依赖聊天历史。
> 最后更新：2026-11-21

## 1. 当前线上状态

- 版本：**v0.2.39**（全站氛围层：环境光/网格/颗粒、首页雷达扫描、路由淡入、数字滚动、卡片浮起）
- 仓库：https://github.com/MHOOAPI/relayradar（public，main 受分支保护）
- 本地栈：`docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d`，镜像来自 GHCR
- 访问：https://localhost/（自签证书）、http://localhost:8088/
- 发布方式：打 `v*` tag → Deploy workflow 构建镜像 → GitHub Release 自动生成

## 2. 技术栈

- 后端：Go 1.25，net/http，pgx v5，内嵌 SQL 迁移（001–022）
- 前端：React 19 + TS + Vite 8，手写 CSS，无 UI 框架
- 部署：Docker Compose（db / api / nginx / nginx-http / prober）
- CI：`.github/workflows/ci.yml`（backend / web / compose / smoke），main 必须过 CI 才能合并

## 3. 数据现状（重要）

| 数据 | 来源 | 状态 |
|---|---|---|
| 模型目录 | OpenRouter `/api/v1/models` | 真实，491 个，每日 11:00 自动同步 |
| mhoo 文本价 | mhoo `/api/ratio_config` | 真实，25 个，¥/1M |
| mhoo 图片价 | mhoo `/api/pricing` | 真实，12 个，¥/张 |
| 其他站点报价 | 演示合成 | 标记为演示数据 |
| 站点目录（新增 120 家） | [daheiai/awesome-api-proxy](https://github.com/daheiai/awesome-api-proxy) 公开清单 | 名称/域名/简介为真实来源；倍率、探针、评分、纯度为演示值；支付/退款仅在原文提及时填写 |
| 探针 | prober（容器内，每 10 分钟） | mhoo 真实探针正常入库（约 5s，记为 slow）；mock 目标已从本地 probes.json 移除 |
| 评分 | 由探针计算 | 有真实数据后自动生效 |

同步命令：

```bash
cd backend
go run ./cmd/modelsync                  # 模型目录
STATION_API_KEY=... go run ./cmd/stationsync -base https://api.mhoo.cc -slug mhoo
```

密钥：`STATION_API_KEY` 存于 GitHub Secret；本地 `probes.json`（gitignored）也有明文。用户已决定暂不轮换。

## 4. 已完成能力

### 站点与模型
- 站点目录、详情、榜单、可达性历史、CSV 导出
- 站点目录：526 个（23 个精选 + 503 个公开目录导入，来自 596 行的公开列表；每家保留 6–14 个代表性模型）；部署启动时 SyncDemoStations 只增量补齐新站点，不清空既有探针与后台改动
- 模型档案页：供给评分（价格 / 覆盖 / 可用性 / 纯度 / 上下文）、规格与价格区间、纯度与可用性分布、同厂商模型、FAQ、报价表与价格变动记录
- 模型列表范围：默认只看有站点报价的模型（150 个，不含 batch / free 变体），按报价站点数排序；「展开全部目录」看完整 491 个
- 模型分类：文本 / 图片 / 视频
- 厂商图标：内置 simple-icons 官方 SVG path；站点图标：内置 logo → favicon 代理兜底 → 自定义上传（迁移 022 回填空 icon_url）

### 商业字段
- 提交表单：支付方式多选、开票、退款政策
- 后台审核：内联编辑（chips + 校验 + 已保存提示），通过时按 domain 覆盖站点
- 站点页展示：支付 / 开票 / 退款

### 检测与审计
- 首页内嵌 Key 检测（baseUrl + key → 模型数），是首页唯一的检测入口；成功后可直接跳转完整检测（8 项，自动带上 baseUrl）
- 免费 API Key **/free-api-keys** 已从「更多」提为一级导航：卡片式列表（状态/领取数/遮罩 Key/服务商/模型标签/备注/领取 CTA）、可用/待验证/服务商筛选与即时搜索、可用数与累计领取统计、领取弹窗一键复制、提交弹窗带滑块验证码、使用与风险说明
- 导航与信息架构：主导航精简为 榜单 / 站点 / 模型 / 排行 / 检测 / 免费 API / 更多（变更雷达与横向对比在「更多」里）；移除赞助页与周榜页（保留 301），移除英文镜像菜单与重复 404；页面不再使用顶部小标签（eyebrow），标题直接承载层级；首页 / 站点 / 模型 / 排行榜默认各显示 50–60 行并「显示更多」，避免 500+ 行一次性铺开
- 个人中心 **/account**（登录后）：账号资料与邮箱验证状态、告警订阅（暂停 / 恢复 / 删除，按登录邮箱归属）、提交记录与审核状态、localStorage 关注列表、账号安全入口；未登录时显示登录 / 注册引导。页面 noindex，数据由 `GET /api/v1/account` 按会话用户返回
- 敏感操作保护：接口检测、提交站点、注册、找回密码、分享免费 Key 均需先完成**滑块验证码**（服务端出图、HMAC 令牌 15 分钟；`CAPTCHA_ENABLED=false` 可关）
- 接口检测：页面顶部列出 **8 项检查**（内容 + 通过标准），检测中显示逐模型进度与耗时，结束后给出「检查项 × 模型」矩阵与逐项明细；从站点拉取模型后可**多选（不限数量）**，逐个检测并各自生成报告，可随时停止（按服务端 4 秒限流排队，失败逐条展示）
- 提交站点填 Key 自动填充（prefill：模型、按次价、文本价）
- 官方状态页事件中文翻译（缓存 + 失败回退英文）
- 后台操作日志覆盖：图标、商务字段、审核、探针配置等
- 归档检测报告（`/audit?report=`）、认领/验证/重置/告警令牌页均返回 `noindex,follow`；`/audit` 与检测说明页保持可索引

### 图标
- 上传 png/jpeg/webp ≤2MB
- 前端中心裁剪 256×256
- 恢复默认回到 favicon
- 替换/恢复时清理旧文件；Docker 卷 `icons`

## 5. 已知限制 / 待办

1. **真实站点只有 mhoo 一家**，全网最低价意义有限；
2. **mock 目标需要 `go run ./cmd/mockupstream -addr :9199`**；默认 probes.json 只保留 mhoo 真实站点；
3. **模型列表范围按「有无站点报价」筛选**，不是人工精选；
4. **官方价 `—`**：mhoo 只给人民币倍率，没有官方美元价；
5. mhoo key 曾暴露在聊天/本地文件，用户选择暂不轮换；
6. 英文页面已全部移除，后端仍保留 `/api/v1/guides/en` 等数据接口；
7. 本机 git 推送不通（TLS/连接被重置），改动通过 GitHub API 提交，见第 9 节。

## 6. 关键设计与决策

- 设计系统见 `DESIGN.md`：信号绿 `#2ee6a6`、炭黑、One Signal / Status Only / Tabular / No Gradient Text
- 产品定位见 `PRODUCT.md`：独立探针 + 可复算评分，广告不参与排名
- 数据边界：站点真实、价格部分真实、探针真实，评分与部分报价为演示
- SEO/GEO：每页独立 title / description / canonical；JSON-LD 覆盖 Organization、WebSite、ItemList（列表页）、Product + AggregateOffer（模型页）、FAQPage（模型页 / 常见问题）、BreadcrumbList、TechArticle（指南）；另有 /llms.txt、/md/model、/md/station 给 AI 抓取
- 分支保护：main 必须通过 CI；所有改动走 PR + auto-merge
- 模型 ID 采用 `vendor--model` 形式（斜杠替换），避免路由问题

## 7. 常用命令

```bash
# 本地开发
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
cd web && npm run build
cd backend && go test ./...

# 发版
git tag -a vX.Y.Z -m "..." && git push origin vX.Y.Z

# 本地 CI
pwsh scripts/ci.ps1 -SkipInstall -SkipDocker
```

## 8. 建议的下一步

1. 接入 2–3 个真实站点（提供 baseUrl + key 即可复用 stationsync）
2. 宿主机定时跑 prober，或用可达代理让容器出网
3. mhoo key 轮换（可选）
4. 模型列表：等真实站点变多后，用真实报价数据替代演示值

## 9. 本机推送问题

`git fetch/push` 到 GitHub 会报 `schannel: AcquireCredentialsHandle failed` 或 `Recv failure: Connection was reset`（只有 `gh` CLI 能通）。工作方式：

```powershell
# 1) 本地提交 2) 用 GitHub API 建 blob/tree/commit/ref 3) gh pr create + gh pr merge --auto
# 脚本样例：.cache/ghpush/publish.ps1（gitignored，可重写）
```

`git config --local http.sslBackend openssl` 可以救回部分 fetch，但 push 仍不稳定。

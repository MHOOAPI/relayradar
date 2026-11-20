# RelayRadar · 中转雷达

[![CI](https://github.com/MHOOAPI/relayradar/actions/workflows/ci.yml/badge.svg)](https://github.com/MHOOAPI/relayradar/actions/workflows/ci.yml)

大模型 API 中转站的独立监测与排序站：连续探针测可用性、速度基准测延迟、思考签名抽检验真伪，公开倍率折算真实人民币价格。

> 演示数据集：站点名称与域名取自公开站点，仅用于展示真实品牌图标；倍率、探针、评分与评价均为生成值，不代表真实商家表现。接入真实数据前请替换 `internal/store/seed.go`。

## 功能

- **中转站榜单**：22 家第三方中转 / 聚合站，按评分 v2 排序
- **评分 v2**：可用性 0.62、速度 0.26、稳定性 0.12，Wilson 95% 下界 + p50/p95
- **纯度抽检**：Claude thinking 签名抽检，识别疑似掺假
- **模型库与报价**：13 个模型，按分组倍率折算人民币价格；最低价跳过疑似掺假站
- **横向对比 / 排行榜 / 变更雷暴**：价格、可用性、分数变化
- **官方厂商排除**：阿里云百炼、硅基流动、七牛云、PPIO 不进入中转站榜单与报价
- **无凭据可达性探测**：每 15 分钟探测站点首页，区分可达 / 拒绝探测 / 不可达 / 未知，并记录 90 天历史；目录支持「只看可达」（`?reachable=1`）与按可达延迟排序；连续 2 次不可达触发 `station_down`，恢复后触发 `station_recovered`
- **上游模型状态**：OpenAI / Anthropic / Gemini / DeepSeek / Moonshot / Qwen / xAI 官方状态、事件时间线与 90 天快照
- **本站服务状态**：90 天可用率、事故、RSS、邮件 / webhook 告警
- **站长认领**：DNS TXT 或 `.well-known` 文件验证，资料修改后台审核，邮件通知
- **中英文双语**、SEO / GEO、sitemap、服务端预渲染
- **鉴权**：邮箱注册登录、PBKDF2、CSRF 校验、限流、审计日志

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25、net/http ServeMux、pgx v5、内嵌 SQL 迁移 |
| 前端 | React 19、TypeScript 5.9、Vite 7、React Router 7 |
| 数据库 | PostgreSQL 17（内存存储用于演示） |
| 部署 | Docker Compose、Caddy 反代 |

## 快速开始

一条命令启动并打印可用入口：

```bash
# Windows（或双击）
scripts\task.cmd dev
scripts\task.cmd ps / logs / stop / test / build

# macOS / Linux
make dev
make ps / logs / stop / test / build
```

没有 `make` 时也可直接 `scripts/dev.cmd` / `./scripts/dev.sh`。

首次运行会自动创建 `.env`（随机数据库密码）；如果已有 PostgreSQL 数据卷，脚本会提示先 `docker compose down -v`。脚本启动前会检查 80 / 443 / 8088 是否被其他程序占用，冲突时直接报出 PID 与进程名；确认要强行启动可加 `-Force`。

手动启动：

```bash
cp .env.example .env
(cd web && npm ci && npm run build)          # Dockerfile 复用本地构建产物
cp backend/probes.example.json probes.json   # 填 Key；不跑真实探针可留空文件
docker compose up --build
# HTTPS: https://localhost（本地自签证书）
# HTTP:  http://localhost:8088（无需证书）
```

`docker-compose.yml` 里的 `prober` 服务每 10 分钟跑一轮，通过 `-api` 上报到 API；需要设置 `PROBE_INGEST_TOKEN` 并挂载 `probes.json`。

本地 HTTPS 需要 `deploy/certs/fullchain.pem` 与 `privkey.pem`；没有正式证书时可用自签证书。`nginx-http` 服务同时提供 `http://localhost:8088`，不经过证书，适合本地开发。

## 生产部署（预构建镜像）

镜像发布在 GHCR，生产不必本地 build：

```bash
RELAYRADAR_TAG=v0.1.0 \
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

`docker-compose.prod.yml` 会把 `api` 与 `prober` 切换到 `ghcr.io/mhooapi/relayradar:<tag>`。

## 本地端口

| 端口 | 归属 | 说明 |
|---|---|---|
| 80 | `nginx` | 301 跳转到 HTTPS |
| 443 | `nginx` | HTTPS 主入口（自签证书） |
| 8088 | `nginx-http` | 纯 HTTP 入口，无需证书 |
| 8080 | 容器内 API | **不在宿主机暴露**；宿主机 8080 可能被其他项目占用 |
| 5432 | 容器内 PostgreSQL | 不映射到宿主机 |

如果 `http://127.0.0.1:8080` 打不开 RelayRadar，是正常的：那不是本站入口，请用 443 或 8088。

本地开发：

```bash
cd backend && go run ./cmd/api
cd web && npm install && npm run dev
```

演示环境变量：

```bash
STORE=memory ADDR=:8080 ALLOW_LOCAL_TARGETS=true CLAIM_DEV_BYPASS=true \
PUBLIC_BASE_URL=http://127.0.0.1:8080 \
WEB_DIR=../web/dist ADMIN_EMAILS=admin@example.com \
go run ./cmd/api
```

首次启动自动建表并写入演示数据：**26 家站点 / 52 个分组 / 25,681 条探针**，覆盖 35 天。

## 页面

| 路径 | 说明 |
|---|---|
| `/` | 首页榜单与实时变更 |
| `/models`、`/models/:id` | 模型库与报价 |
| `/leaderboards` | 模型 / 站点排行榜 |
| `/compare` | 多站对比 |
| `/stations`、`/stations/:slug` | 中转站目录与档案 |
| `/changes` | 变更雷暴与订阅 |
| `/audit` | 接口检测 |
| `/guides`、`/guides/en` | 选站指南 |
| `/submit`、`/claim/:token` | 提交与认领 |
| `/status`、`/en/status` | 本站服务状态 |
| `/model-status`、`/model-status/:key` | 上游模型 API 状态 |
| `/en/model-status`、`/en/model-status/:key` | 英文厂商状态 |
| `/free-api-keys` | 免费 Key 目录 |
| `/admin` | 后台 |

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/meta` | 站点 / 分组 / 模型 / 探针计数 |
| GET | `/api/v1/stations` | 中转站列表（含 `modelIds`、最近可达性） |
| GET | `/api/v1/stations/{slug}/reachability` | 90 天首页可达性历史；`?format=csv` 导出 CSV |
| GET | `/api/v1/reachability.csv` | 全站最新可达性快照 CSV |
| GET | `/api/v1/leaderboards` | 排行榜 |
| GET | `/api/v1/models`、`/models/{id}` | 模型与报价 |
| GET | `/api/v1/changes` | 变更 |
| GET | `/api/v1/status` | 本站组件状态 |
| GET | `/api/v1/model-status` | 上游厂商状态 + 历史 |
| GET | `/api/v1/incidents`、`/status.rss` | 事故与 RSS |
| POST | `/api/v1/subscriptions` | 告警订阅（`provider_incident` 可按 `providers` 过滤） |
| POST | `/api/v1/claims` | 生成认领令牌 |
| POST | `/api/v1/claims/{token}/verify` | DNS / 文件验证（限流 10 次/10 分钟） |
| GET/PATCH | `/api/v1/claims/{token}/station` | 读取 / 提交资料修改（进入审核） |
| GET | `/api/v1/claims/{token}/edits` | 站长提交历史 |
| GET/POST | `/api/v1/admin/claim-edits` | 后台审核列表 / 通过 / 拒绝 |

## 环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `ADDR` | `:8080` | 监听地址 |
| `STORE` | `postgres` | `postgres` 或 `memory` |
| `DATABASE_URL` | — | PostgreSQL 连接串 |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | 对外地址 |
| `WEB_DIR` | `../web/dist` | 前端产物 |
| `ADMIN_EMAILS` | — | 管理员邮箱，逗号分隔 |
| `ALLOW_LOCAL_TARGETS` | `false` | 允许探测内网地址（仅演示） |
| `CLAIM_DEV_BYPASS` | `false` | 跳过认领 DNS 校验（生产必须 false） |
| `REACHABILITY_ENABLED` | `true` | 是否启用站点首页可达性探测 |
| `REACHABILITY_DOWN_THRESHOLD` | `2` | 连续几次不可达触发 `station_down` |
| `PROBE_INGEST_TOKEN` | 空 | prober 通过 `POST /api/v1/probes` 上报所需的令牌 |
| `PROBES_CONFIG_PATH` | `probes.json` | 后台「真实探针」保存配置的文件路径 |
| `HTTPS_PROXY` / `HTTP_PROXY` | 空 | 可达性探测与 favicon 抓取使用的代理；Go 不读取 Windows 系统代理 |
| `SMTP_ADDR` / `SMTP_FROM` | — | 邮件发送 |
| `ALERT_FAILURE_EMAIL` | — | webhook 连续失败提醒 |
| `FAVICON_PROXY_URL` | — | favicon 代理 |

## 数据说明

- 站点：AIHubMix、302.AI、OpenRouter、PackyAPI、CloseAI、DMXAPI、OfoxAI、OhMyGPT、Weelinking、n1n.ai、API易、云雾 API、老板 AI、Poixe AI、API 快连、ZenMux、GPTGOD、UiUiAPI、FerryAPI、玄枢 API、MKEAI、WolfCheap（虚构黑榜样例）
- 官方厂商（阿里云百炼、硅基流动、七牛云、PPIO）保留在数据集中用于官方价对照，但不进入中转站榜单
- 探针数据为生成值，禁止用于真实购买决策

## 测试

```bash
cd backend && go vet ./... && go test ./...
cd web && npm run build
```

CI 在 `.github/workflows/ci.yml`：backend vet + test、web build、compose 校验与镜像构建。本地可先跑 `make test`（Windows 用 `scripts\task.cmd test`）。

真实探针 worker（需要站点 Key）：

```bash
cd backend
cp probes.example.json probes.json   # 填入各站真实 Key，或在后台「真实探针」编辑
go run ./cmd/prober -config probes.json -validate
go run ./cmd/prober -config probes.json -once -api http://127.0.0.1:8080 -token "$PROBE_INGEST_TOKEN"
```

## 已知限制

- 演示数据，不是真实监测结果
- 认领校验在演示环境以 `CLAIM_DEV_BYPASS=true` 跳过
- 浅色主题、部分移动端表格尚未完成
- Docker 镜像未在本机构建（本机 Docker daemon 不可用），仅验证 `docker compose config`

## 来源

- [awesome-claude-api](https://github.com/peter123023/awesome-claude-api)
- [awesome-ai-api-proxy providers](https://raw.githubusercontent.com/howardpen9/awesome-ai-api-proxy/main/data/providers.yaml)
- [awesome-ai-proxy](https://github.com/mn-api/awesome-ai-proxy)
- [LaoZhang API 文档](https://docs.laozhang.ai/about)
- [Poixe 文档](https://docs.poixe.com/cn/api-reference/introduction/base-url)
- [API 快连文档](https://docs.apikl.ai/claude-code-proxy)

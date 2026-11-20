const ENDPOINTS = [
  ['GET', '/api/v1/meta', '数据时间、站点/分组/探针计数、是否演示数据'],
  ['GET', '/api/v1/stations', '榜单。参数：service、tier、q、sort、limit'],
  ['GET', '/api/v1/stations/{slug}', '站点档案：分组、模型实价、评价、商务条款'],
  ['GET', '/api/v1/stations/{slug}/history', '近 14 天综合分 / 可用性 / 速度 / 稳定性序列'],
  ['GET', '/api/v1/stations/{slug}/reviews', '评价列表'],
  ['POST', '/api/v1/stations/{slug}/reviews', '发布评价 {author, rating, body}'],
  ['GET', '/api/v1/models', '模型目录与全网最低价'],
  ['GET', '/api/v1/models/{id}', '模型详情 + 全部站点报价'],
  ['POST', '/api/v1/audit', '接口检测 {baseUrl, apiKey, model}'],
  ['POST', '/api/v1/audit/models', '列出端点的模型 ID {baseUrl, apiKey}'],
  ['GET', '/api/v1/audit/{id}', '读取已存档的检测报告'],
  ['POST', '/api/v1/subscriptions', '创建 webhook 告警 {url, kinds, stations}'],
  ['GET', '/api/v1/subscriptions/{id}?token=', '读取订阅状态（失败次数 / 最近错误）'],
  ['PATCH', '/api/v1/subscriptions/{id}', '修改事件类型或暂停投递（X-RelayRadar-Token）'],
  ['POST', '/api/v1/subscriptions/{id}/ping', '发送测试事件（X-RelayRadar-Token）'],
  ['DELETE', '/api/v1/subscriptions/{id}', '删除订阅（X-RelayRadar-Token）'],
  ['POST', '/api/v1/submissions', '提交新站点'],
  ['POST', '/api/v1/claims', '生成认领令牌 {slug, email}'],
  ['POST', '/api/v1/claims/{token}/verify', '校验认领令牌'],
  ['GET', '/llms.txt', '给 AI Agent 的纯文本摘要（榜单 + 模型 + 接口）'],
];

export function ApiDocsPage() {
  return (
    <main className="wrap docs">
      <div className="page-head">
        <div>
          <h1>公开接口</h1>
          <p className="sub">
            只读接口无需鉴权，限流按 IP。所有时间均为 UTC，价格单位为人民币 / 百万 Token。
          </p>
        </div>
        <div className="head-actions">
          <a className="btn btn-ghost btn-sm" href="/llms.txt" target="_blank" rel="noreferrer">llms.txt</a>
        </div>
      </div>

      <section className="panel">
        <div className="table-wrap">
          <table className="index-table docs-table">
            <thead>
              <tr><th>方法</th><th>路径</th><th>说明</th></tr>
            </thead>
            <tbody>
              {ENDPOINTS.map(([method, path, desc]) => (
                <tr key={path + method}>
                  <td><span className={`http-method http-method-${method.toLowerCase()}`}>{method}</span></td>
                  <td className="mono path">{path}</td>
                  <td className="dim">{desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel pad">
        <h2>示例</h2>
        <pre className="formula-block mono">{`# 榜单前 5 家，按价格排序
curl -s 'https://your-host/api/v1/stations?sort=price&limit=5'

# 单站档案
curl -s 'https://your-host/api/v1/stations/lingxi'

# 探测任意 OpenAI 兼容端点（生产环境需 https）
curl -s -X POST 'https://your-host/api/v1/audit' \\
  -H 'Content-Type: application/json' \\
  -d '{"baseUrl":"https://api.example.com/v1","apiKey":"sk-...","model":"claude-opus-5-5"}'`}</pre>
        <p className="fine">
          检测接口对目标地址做了 SSRF 防护：仅 https、拒绝私网与链路本地地址、不跟随重定向、30 秒超时、按 IP 限流。
        </p>
      </section>

      <section className="panel pad">
        <h2>数据口径</h2>
        <ul className="limits">
          <li><b>rankScore</b>：可用性 × (0.6 + 0.4 × 速度)，分组按 0.8ⁱ 加权，范围 0–100。</li>
          <li><b>uptimePct</b>：原始可用率，成功探针 / 全部探针，用于展示；<b>availability</b> 是排名用的平滑指数。</li>
          <li><b>purity</b>：仅 Claude 分组，7 天中位数；<b>purityTier</b> 为 official / mixed / counterfeit / unknown。</li>
          <li><b>fromPriceCny</b>：全站所有模型中最低的输入价，-1 表示无报价。</li>
        </ul>
        <p className="fine">
          引用数据请保留 <code className="mono">dataUpdatedAt</code> 时间戳；演示数据集请勿当作真实评测传播。
        </p>
      </section>
    </main>
  );
}

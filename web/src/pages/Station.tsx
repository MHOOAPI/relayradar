import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api';
import type { StatusHistoryDay, StationDetail } from '../types';
import { External, fmtCny, fmtDate, Spinner, Stars, TierBadge, WatchStar } from '../components/ui';
import { StationMark, VendorMark } from '../components/marks';
import { PayMark } from '../components/pay';
import { ArrowUpRight, CalendarIcon, CoinIcon, CopyIcon, ProbeIcon, PulseIcon, ReceiptIcon, ShieldIcon, WalletIcon } from '../components/icons';
import { TrendChart } from '../components/TrendChart';
import type { Change } from '../types';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const SERVICE_LABEL: Record<string, string> = {
  claude: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  deepseek: 'DeepSeek',
  xai: 'Grok',
  moonshot: 'Kimi',
  qwen: 'Qwen',
};

const fmtCtx = (n: number) => (n >= 1000000 ? `${(n / 1000000).toFixed(n % 1000000 === 0 ? 0 : 2)}M` : `${Math.round(n / 1000)}K`);

export function StationPage() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const [data, setData] = useState<StationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { push } = useToast();
  const [changes, setChanges] = useState<Change[]>([]);
  const [reach, setReach] = useState<StatusHistoryDay[] | null>(null);
  const [metric, setMetric] = useState<'score' | 'availability' | 'speed' | 'consistency'>('score');

  const [author, setAuthor] = useState('');
  const [rating, setRating] = useState(5);
  const [body, setBody] = useState('');
  const [posting, setPosting] = useState(false);
  const [postError, setPostError] = useState<string | null>(null);

  useEffect(() => {
    if (!slug) return;
    setData(null);
    setError(null);
    api.station(slug).then(setData).catch((e: Error) => setError(e.message));
    api.changes(slug).then((r) => setChanges(r.items)).catch(() => setChanges([]));
    api.stationReachability(slug).then((r) => setReach(r.days)).catch(() => setReach([]));
  }, [slug]);

  const copy = (text: string) => {
    void copyText(text).then((ok) => push(ok ? 'Base URL 已复制' : '复制失败，请手动选择', ok ? 'ok' : 'err'));
  };

  const submitReview = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!slug || body.trim().length < 5) {
      setPostError('评价至少 5 个字');
      return;
    }
    setPosting(true);
    setPostError(null);
    try {
      const rv = await api.reviews(slug, { author: author.trim() || '匿名开发者', rating, body: body.trim() });
      setData((prev) => (prev ? { ...prev, reviews: [rv, ...prev.reviews] } : prev));
      setBody('');
      setAuthor('');
    } catch (err) {
      setPostError((err as Error).message);
    } finally {
      setPosting(false);
    }
  };

  if (error) {
    return (
      <main className="wrap">
        <div className="error-box">加载失败：{error} <button className="btn btn-ghost btn-sm" onClick={() => navigate('/')}>返回榜单</button></div>
      </main>
    );
  }

  if (!data) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 正在读取站点档案…</div></main>;
  }

  const { station: s, groups, models, reviews } = data;
  const avg = reviews.length ? reviews.reduce((a, r) => a + r.rating, 0) / reviews.length : 0;

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/">榜单</Link><span>/</span><span className="dim">{s.slug}</span>
      </div>

      {s.newDomain && (
        <div className="notice notice-warn">
          <b>新域名观察期</b>
          <span>该站上架仅 {Math.max(1, Math.round((Date.now() - new Date(s.listedAt).getTime()) / 86400000))} 天，建议单次充值不超过 ¥20，先用小额 Key 验证再投入生产。</span>
        </div>
      )}

      <section className="detail-head panel">
        <div className="detail-main">
          <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={54} />
          <div>
            <div className="detail-title">
              <h1>{s.name}</h1>
              {s.rank > 0 && <span className="rank-pill mono">#{s.rank}</span>}
              {s.claimed && <span className="badge-ok">✓ 已认领</span>}
              {s.purityTier === 'counterfeit' && <span className="badge-bad">抽检异常</span>}
              <WatchStar slug={s.slug} size={16} />
            </div>
            <p className="tagline">{s.tagline}</p>
            <div className="detail-links mono" style={{ marginTop: 6 }}>
              <span>支付 {(s as unknown as { payments?: string[] }).payments?.join(' / ') || '—'}</span>
              <span>·</span>
              <span>开票 {(s as unknown as { invoice?: boolean }).invoice ? '支持' : '不支持'}</span>
              <span>·</span>
              <span>退款 {(s as unknown as { refundPolicy?: string }).refundPolicy || '—'}</span>
            </div>
            <div className="detail-links mono">
              <Link to={`/go/${s.slug}`}>{s.domain} <External /></Link>
              <span className="dim">{s.baseUrl}</span>
              <button className="link-btn copy-btn" onClick={() => copy(s.baseUrl)}>
                <CopyIcon />
                复制 Base URL
              </button>
            </div>
          </div>
        </div>
        <div className="detail-actions">
          <Link className="btn btn-ghost" to={`/go/${s.slug}`}>
            访问官网 <ArrowUpRight />
          </Link>
          <Link className="btn btn-primary" to={`/audit?url=${encodeURIComponent(s.baseUrl)}`}>
            <ProbeIcon /> 检测该站接口
          </Link>
          {s.claimed ? (
            <span className="claimed-badge">已认领</span>
          ) : (
            <Link className="btn btn-ghost" to={`/submit?claim=${s.slug}`}>认领该站点</Link>
          )}
        </div>
      </section>

      <section className="scorecards five">
        {[
          { label: '综合分', value: s.rankScore, note: `全站第 ${s.rank || '—'} 名 · 0–100` },
          { label: '可用性分', value: s.availability, note: `原始 ${s.uptimePct.toFixed(1)}% · Wilson 下界` },
          { label: '速度分', value: s.speed, note: '延迟中位数映射' },
          { label: '稳定性分', value: s.consistency, note: 'p95 / p50 抖动' },
        ].map((c) => (
          <div className="scorecard" key={c.label}>
            <span className="label">{c.label}</span>
            <b className="big">
              {c.value.toFixed(1)}
              {c.label === '综合分' && s.scoreMargin > 0 && (
                <small className="ci" title="95% 置信区间，仅统计可用性抽样误差；速度与稳定性为点估计">±{s.scoreMargin.toFixed(1)}</small>
              )}
            </b>
            <small>{c.note}</small>
            <span className="mini-meter"><i style={{ width: `${Math.max(0, Math.min(100, c.value))}%` }} /></span>
          </div>
        ))}
        <div className="scorecard">
          <span className="label">纯度抽检</span>
          <b className="big">{s.purityMeasured ? s.purity.toFixed(0) : '—'}</b>
          <TierBadge tier={s.purityTier} />
          <span className="mini-meter"><i style={{ width: `${s.purityMeasured ? s.purity : 0}%` }} /></span>
        </div>
      </section>

      <section className="panel breakdown">
        <div className="breakdown-head">
          <div>
            <h2>综合分构成</h2>
            <p className="sub">三个轴各自 0–100，按 0.62 / 0.26 / 0.12 加权；每个轴先做覆盖加权，再与最差分组混合。± 仅统计可用性抽样误差（95% 置信），速度与稳定性为点估计。</p>
          </div>
          <span className="mono dim">
            {s.samples} 条探针 · {s.scoredGroups}/{s.groupsCount} 个分组计分
          </span>
        </div>
        <div className="breakdown-rows">
          {[
            ['可用性', s.availability, 0.62],
            ['速度', s.speed, 0.26],
            ['稳定性', s.consistency, 0.12],
          ].map(([label, value, weight]) => (
            <div className="bd-row" key={String(label)}>
              <span>{label}</span>
              <b className="mono">{(value as number).toFixed(1)}</b>
              <div className="bd-bar"><i style={{ width: `${Math.max(0, Math.min(100, value as number))}%` }} /></div>
              <small className="mono dim">× {weight}</small>
            </div>
          ))}
        </div>
      </section>

      {reach && reach.some((d) => d.status !== 'unknown') && (() => {
        const known = reach.filter((d) => d.status !== 'unknown');
        const up = known.length ? known.reduce((sum, d) => sum + d.uptime, 0) / known.length : 0;
        return (
          <section className="panel reach-panel">
            <div className="panel-head">
              <div>
                <h2>首页可达性</h2>
                <p className="sub">无凭据探测站点首页，仅代表可访问，不代表模型可用性。</p>
              </div>
              <span className="mono dim">近 90 天 · {known.length} 天数据 · {up.toFixed(1)}%</span>
              <a className="link-btn tiny" href={`/api/v1/stations/${encodeURIComponent(s.slug)}/reachability?format=csv`}>导出 CSV</a>
            </div>
            <div className="provider-history-bars">
              {reach.map((d) => <i key={d.date} className={`b-${d.status}`} title={`${d.date} · ${d.uptime.toFixed(0)}%`} />)}
            </div>
          </section>
        );
      })()}

      {data.history.length > 1 && (() => {
        const hist = data.history;
        const values = hist.map((p) => p[metric]);
        const latest = values[values.length - 1];
        const delta = latest - values[0];
        const labels = hist.map((p) => p.date.slice(5));
        const color =
          metric === 'score' ? '#2ee6a6' : metric === 'availability' ? '#6aa8ff' : metric === 'speed' ? '#f5b942' : '#b79bff';
        const METRICS: Array<[typeof metric, string]> = [
          ['score', '综合分'],
          ['availability', '可用性'],
          ['speed', '速度'],
          ['consistency', '稳定性'],
        ];
        return (
          <section className="panel trend-panel">
            <div className="panel-head">
              <div>
                <h2>近 30 天趋势</h2>
                <p className="sub">每天用截至当天的 7 天探针窗口重算，口径与实时分一致。± 仅统计可用性抽样误差，速度与稳定性为点估计。</p>
              </div>
              <div className="tabs trend-tabs">
                {METRICS.map(([k, label]) => (
                  <button key={k} className={`tab ${metric === k ? 'on' : ''}`} onClick={() => setMetric(k)}>{label}</button>
                ))}
              </div>
            </div>
            <div className="trend-body">
              <div className="trend-summary">
                <b className="mono">{latest.toFixed(1)}</b>
                <span className={delta >= 0 ? 'delta up' : 'delta down'}>
                  {delta >= 0 ? '+' : ''}{delta.toFixed(1)} <small>vs 30 天前</small>
                </span>
                <span className="mono dim">样本 {s.samples}</span>
              </div>
              <TrendChart values={values} labels={labels} color={color} />
            </div>
          </section>
        );
      })()}

      {changes.length > 0 && (
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>近期变更</h2>
              <p className="sub">由探针与价格历史自动检测，不含人工通告。</p>
            </div>
            <Link className="link-btn" to={`/changes?station=${s.slug}`}>全部变更 →</Link>
          </div>
          <ul className="changes-list">
            {changes.slice(0, 5).map((c) => (
              <li key={c.id} className={`change change-${c.severity}`}>
                <span className="change-dot" />
                <div className="change-body">
                  <b>{c.title}</b>
                  <p>{c.detail}</p>
                </div>
                <time className="mono dim">{fmtDate(c.at)}</time>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>分组与线路</h2>
            <p className="sub">同一站点不同分组的倍率、可用性与纯度可以完全不同，别只看首页宣传价。</p>
          </div>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <thead>
              <tr>
                <th>分组</th><th>服务</th><th className="num">倍率</th><th>当前状态</th>
                <th className="num">延迟</th><th className="num">可用性分</th><th className="num hide-sm">速度</th>
                <th className="num hide-sm">稳定性</th><th>纯度</th><th className="num">模型</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <tr key={g.slug}>
                  <td><b>{g.name}</b><div className="mono dim tiny">{g.slug}</div></td>
                  <td><span className="svc">{SERVICE_LABEL[g.service] ?? g.service}</span></td>
                  <td className="num mono">{g.multiplier.toFixed(2)}×</td>
                  <td>
                    <span className={`status status-${g.status}`}>
                      <i />{g.status === 'ok' ? '正常' : g.status === 'slow' ? '偏慢' : g.status === 'unknown' || !g.status ? '未探测' : '不可用'}
                    </span>
                  </td>
                  <td className="num mono dim">{g.latencyMs ? `${g.latencyMs} ms` : '—'}</td>
                  <td className="num mono">{g.availability.toFixed(1)}</td>
                  <td className="num mono dim hide-sm">{g.speed.toFixed(1)}</td>
                  <td className="num mono dim hide-sm">{g.scored ? g.consistency.toFixed(1) : '—'}</td>
                  <td><TierBadge tier={g.purityTier} score={g.purity} /></td>
                  <td className="num mono dim">{g.modelCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>支持模型与本站实价</h2>
            <p className="sub">按该站最便宜的分组折算，单位为每百万 Token 人民币。</p>
          </div>
          <span className="mono dim">{models.length} 个模型</span>
        </div>
        <div className="table-wrap">
          <p className="table-hint">左右滑动查看全部列</p>
          <table className="index-table">
            <thead>
              <tr>
                <th>模型</th><th className="hide-sm">厂商</th><th className="num hide-sm">上下文</th>
                <th className="num hide-sm">官方价 /1M</th><th className="num">本站输入</th><th className="num">本站输出</th>
                <th className="hide-sm">最低价分组</th>
              </tr>
            </thead>
            <tbody>
              {models.map((m) => (
                <tr key={m.model.id} onClick={() => navigate(`/models/${encodeURIComponent(m.model.id)}`)} className="clickable">
                  <td>
                    <div className="cell-station">
                      <VendorMark vendor={m.model.vendor} size={26} />
                      <div className="station-meta">
                        <Link className="station-name" to={`/models/${encodeURIComponent(m.model.id)}`} onClick={(e) => e.stopPropagation()}>{m.model.name}</Link>
                        <span className="mono dim tiny">{m.model.id}</span>
                      </div>
                    </div>
                  </td>
                  <td className="hide-sm dim">{m.model.vendorName}</td>
                  <td className="num mono dim hide-sm">{m.model.contextWindow > 0 ? fmtCtx(m.model.contextWindow) : '—'}</td>
                  <td className="num mono dim hide-sm">{m.model.inputUsd > 0 ? `$${m.model.inputUsd} / $${m.model.outputUsd}` : '—'}</td>
                  <td className="num mono">{fmtCny(m.bestInputCny)}</td>
                  <td className="num mono accent">{fmtCny(m.bestOutputCny)}</td>
                  <td className="hide-sm dim">{m.cheapestGroup}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <div className="two-col">
        <section className="panel">
          <div className="panel-head"><h2>商务条款</h2></div>
          <dl className="kv">
            <div><dt><CoinIcon /> 最低充值</dt><dd className="mono">¥{s.minTopupCny}</dd></div>
            <div>
              <dt><WalletIcon /> 支付方式</dt>
              <dd>
                <span className="pay-list">
                  {s.payments.map((p) => (
                    <span className="pay-chip" key={p}>
                      <PayMark method={p} size={16} />
                      {p}
                    </span>
                  ))}
                </span>
              </dd>
            </div>
            <div><dt><ReceiptIcon /> 发票</dt><dd>{s.invoice ? '支持开票' : '不支持开票'}</dd></div>
            <div><dt><ShieldIcon /> 退款政策</dt><dd>{s.refundPolicy}</dd></div>
            <div><dt><CalendarIcon /> 上架时间</dt><dd className="mono">{fmtDate(s.listedAt)}</dd></div>
            <div><dt><PulseIcon /> 最近探针</dt><dd className="mono">{new Date(s.lastProbeAt).toLocaleString('zh-CN', { hour12: false })}</dd></div>
            <div><dt><ArrowUpRight /> 近 30 天跳转</dt><dd className="mono">{data.outbound30d ?? 0} 次</dd></div>
          </dl>
          <p className="fine pad">
            以上信息来自站点公开页面与站长提交，可能过期；付款前请以站点当前条款为准。
          </p>
        </section>

        <section className="panel">
          <div className="panel-head">
            <h2>开发者评价</h2>
            <span className="mono dim">{reviews.length} 条 · 均分 {avg ? avg.toFixed(1) : '—'}</span>
          </div>
          <div className="reviews">
            {reviews.length === 0 && <p className="empty">还没有评价，欢迎补充第一条实测记录。</p>}
            {reviews.map((r) => (
              <article key={r.id} className="review">
                <div className="review-head">
                  <b>{r.author}</b>
                  <Stars value={r.rating} />
                  <span className="mono dim tiny">{new Date(r.createdAt).toLocaleDateString('zh-CN')}</span>
                  {r.verified && <span className="badge-ok tiny">已验证</span>}
                </div>
                <p>{r.body}</p>
              </article>
            ))}
          </div>
          <form className="review-form" onSubmit={submitReview}>
            <div className="row">
              <input value={author} onChange={(e) => setAuthor(e.target.value)} placeholder="昵称（可选）" maxLength={40} />
              <select value={rating} onChange={(e) => setRating(Number(e.target.value))} aria-label="评分">
                {[5, 4, 3, 2, 1].map((n) => <option key={n} value={n}>{n} 星</option>)}
              </select>
            </div>
            <textarea value={body} onChange={(e) => setBody(e.target.value)} placeholder="写下你的真实调用体验：延迟、倍率、缓存、退款……" maxLength={600} rows={3} />
            {postError && <p className="form-error">{postError}</p>}
            <div className="form-foot">
              <span className="fine">评价仅代表用户个人体验，不参与评分。</span>
              <button className="btn btn-primary btn-sm" disabled={posting}>{posting ? '提交中…' : '发布评价'}</button>
            </div>
          </form>
        </section>
      </div>
    </main>
  );
}

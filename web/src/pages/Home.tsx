import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { Change, Meta, ModelSummary, Station } from '../types';
import { Arrow, fmtCny, fmtNum, TierBadge, UptimeBars, WatchStar } from '../components/ui';
import { StationMark } from '../components/marks';
import { serviceLabel } from '../lib/services';
import { KeyCheck } from '../components/KeyCheck';
import { useWatchlist } from '../lib/watchlist';

const SERVICES: Array<[string, string]> = [
  ['all', '全部'],
  ['claude', 'Claude'],
  ['openai', 'OpenAI'],
  ['gemini', 'Gemini'],
  ['deepseek', 'DeepSeek'],
  ['xai', 'Grok'],
];

const TIERS: Array<[string, string]> = [
  ['all', '全部'],
  ['official', '官方直连'],
  ['mixed', '混合线路'],
  ['counterfeit', '疑似掺假'],
];

const SORTS: Array<[string, string]> = [
  ['rank', '综合分'],
  ['availability', '可用性'],
  ['speed', '速度'],
  ['price', '最低价'],
  ['models', '模型数'],
];


const CHANGE_LABEL: Record<string, string> = {
  price_change: '价格',
  availability_drop: '可用性',
  purity_alert: '纯度',
  new_station: '新站',
  score_drop: '分数',
};

export function Home() {
  const [stations, setStations] = useState<Station[] | null>(null);
  const [meta, setMeta] = useState<Meta | null>(null);
  const heroStations = useCountUp(meta?.stations);
  const heroGroups = useCountUp(meta?.groups);
  const heroProbes = useCountUp(meta?.probes);
  const [error, setError] = useState<string | null>(null);
  const [changes, setChanges] = useState<Change[]>([]);
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [params, setParams] = useSearchParams();
  const [service, setService] = useState(() => {
    const v = params.get('svc') ?? 'all';
    return v === 'all' || SERVICES.some(([k]) => k === v) ? v : 'all';
  });
  const [tier, setTier] = useState(() => {
    const v = params.get('tier') ?? 'all';
    return v === 'all' || TIERS.some(([k]) => k === v) ? v : 'all';
  });
  const [sort, setSort] = useState(() => {
    const v = params.get('sort') ?? 'rank';
    return SORTS.some(([k]) => k === v) ? v : 'rank';
  });
  const [q, setQ] = useState(params.get('q') ?? '');
  const [onlyWatch, setOnlyWatch] = useState(params.get('watch') === '1');
  const radarRef = useRef<HTMLDivElement>(null);
  // Hero parallax: the radar drifts slower than the page. Cheap rAF-throttled
  // scroll handler, skipped entirely when the user asks for reduced motion.
  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    let raf = 0;
    const onScroll = () => {
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        const y = Math.min(window.scrollY, 700);
        if (radarRef.current) radarRef.current.style.transform = `translate3d(0, ${(y * 0.16).toFixed(1)}px, 0)`;
      });
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      window.removeEventListener('scroll', onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  }, []);
  const { ids: watched } = useWatchlist();

  useEffect(() => {
    api.changes().then((r) => setChanges((r.items ?? []).slice(0, 6))).catch(() => setChanges([]));
    api.models().then((r) => setModels(r.items ?? [])).catch(() => setModels([]));
    const timer = window.setInterval(() => {
      if (document.visibilityState !== 'visible') return;
      api.stations(1000).then((r) => setStations(r.items)).catch(() => {});
      api.meta().then(setMeta).catch(() => {});
    }, 60000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (service !== 'all') p.set('svc', service);
    if (tier !== 'all') p.set('tier', tier);
    if (sort !== 'rank') p.set('sort', sort);
    if (onlyWatch) p.set('watch', '1');
    setParams(p, { replace: true });
  }, [q, service, tier, sort, onlyWatch, setParams]);

  const resetFilters = () => {
    setQ('');
    setService('all');
    setTier('all');
    setSort('rank');
    setOnlyWatch(false);
  };
  const navigate = useNavigate();

  useEffect(() => {
    Promise.all([api.stations(1000), api.meta()])
      .then(([s, m]) => {
        setStations(s.items);
        setMeta(m);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const serviceCounts = useMemo(() => {
    const m: Record<string, number> = { all: stations?.length ?? 0 };
    for (const s of stations ?? []) {
      for (const sv of s.services) m[sv] = (m[sv] ?? 0) + 1;
    }
    return m;
  }, [stations]);

  const tierCounts = useMemo(() => {
    const m: Record<string, number> = { all: stations?.length ?? 0 };
    for (const s of stations ?? []) m[s.purityTier] = (m[s.purityTier] ?? 0) + 1;
    return m;
  }, [stations]);

  const rows = useMemo(() => {
    if (!stations) return [];
    const needle = q.trim().toLowerCase();
    let list = stations.filter((s) => {
      if (needle && !`${s.name} ${s.domain} ${s.tagline}`.toLowerCase().includes(needle)) return false;
      if (service !== 'all' && !s.services.includes(service)) return false;
      if (tier !== 'all' && s.purityTier !== tier) return false;
      if (onlyWatch && !watched.includes(s.slug)) return false;
      return true;
    });
    list = [...list];
    if (sort === 'availability') list.sort((a, b) => b.availability - a.availability);
    else if (sort === 'speed') list.sort((a, b) => b.speed - a.speed);
    else if (sort === 'price') list.sort((a, b) => (a.fromPriceCny < 0 ? 1 : b.fromPriceCny < 0 ? -1 : a.fromPriceCny - b.fromPriceCny));
    else if (sort === 'models') list.sort((a, b) => b.modelsCount - a.modelsCount);
    else list.sort((a, b) => a.rank - b.rank);
    return list;
  }, [stations, service, tier, sort, q, onlyWatch, watched]);

  const [visible, setVisible] = useState(60);
  useEffect(() => setVisible(60), [q, service, tier, sort, onlyWatch]);
  const shown = rows.slice(0, visible);

  const exportCsv = () => {
    const esc = (v: string | number | boolean) => `"${String(v).replace(/"/g, '""')}"`;
    const header = ['rank', 'name', 'domain', 'services', 'rankScore', 'uptimePct', 'speed', 'purityTier', 'purity', 'models', 'fromPriceCny', 'minTopupCny', 'invoice'];
    const lines = [header.join(',')].concat(
      rows.map((s) => [
        s.rank, s.name, s.domain, s.services.join('|'), s.rankScore, s.uptimePct, s.speed,
        s.purityTier, s.purity, s.modelsCount, s.fromPriceCny, s.minTopupCny, s.invoice,
      ].map(esc).join(',')),
    );
    const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'relayradar-stations.csv';
    a.click();
    URL.revokeObjectURL(a.href);
  };

  return (
    <main className="wrap home">
      <section className="hero">
        <div className="hero-radar" aria-hidden="true" ref={radarRef}>
          <span className="hr-ring r1" />
          <span className="hr-ring r2" />
          <span className="hr-ring r3" />
          <span className="hr-sweep" />
          <span className="hr-blip b1" />
          <span className="hr-blip b2" />
          <span className="hr-blip b3" />
        </div>
        <div className="hero-copy">
          <h1>
            别猜哪家中转站
            <br />
            <em>掺了水</em>，看数据。
          </h1>
          <p>
            每 2 小时一次真实探针，按可用性、速度、思考签名抽检和真实倍率排序。
            赞助位与广告不参与排名。
          </p>
          <div className="hero-cta">
            <Link to="/methodology" className="btn btn-ghost">评分怎么算的</Link>
          </div>
          <div className="trust-strip">
            <span className="trust-item">
              <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m3 8.5 3 3 7-7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" /></svg>
              独立第三方，不收款
            </span>
            <span className="trust-item">
              <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m3 8.5 3 3 7-7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" /></svg>
              口径公开，可手算复现
            </span>
            <span className="trust-item">
              <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="m3 8.5 3 3 7-7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" /></svg>
              赞助位不参与排名
            </span>
          </div>
          <Link to="/guides/how-to-choose-relay" className="hero-link">
            新手？先看 10 分钟选站清单 →
          </Link>
          <a className="hero-scroll mono" href="#ranking">向下看完整榜单 <span aria-hidden="true">↓</span></a>
        </div>

        <div className="hero-card">
          <div className="hero-card-head">
            <span className="mono label">评分构成</span>
            <span className="mono tag-live">v2.0</span>
          </div>
          <p className="hero-formula">综合分 = 62% 可用性 + 26% 速度 + 12% 稳定性</p>
          <p className="hero-formula-note">
            可用性取 95% 置信下界，小样本不吹满分；纯度与价格单列，不参与排名。
          </p>
          <div className="hero-stats">
            <div><b>{meta ? fmtNum(heroStations) : '—'}</b><span>站点</span></div>
            <div><b>{meta ? fmtNum(heroGroups) : '—'}</b><span>分组</span></div>
            <div><b>{meta ? fmtNum(heroProbes) : '—'}</b><span>探针</span></div>
          </div>
          <Link className="hero-formula-link" to="/methodology">看完整公式与已知局限 →</Link>
        </div>
      </section>

      <KeyCheck />

      {changes.length > 0 && (
        <div className="live-strip" aria-label="最近变更">
          <span className="live-strip-label"><i className="live-dot" />LIVE</span>
          <div className="live-strip-viewport">
            <div className="live-strip-track">
              {[...changes, ...changes].map((c, i) => (
                <Link
                  key={`${c.id}-${i}`}
                  to={c.stationSlug ? `/stations/${c.stationSlug}` : '/changes'}
                  className="live-item"
                  aria-hidden={i >= changes.length}
                  tabIndex={i >= changes.length ? -1 : 0}
                >
                  <span className={`live-kind ${c.severity}`}>{CHANGE_LABEL[c.kind] ?? c.kind}</span>
                  <span className="live-title">{c.title}</span>
                  <span className="live-station mono">{c.stationName}</span>
                </Link>
              ))}
            </div>
          </div>
          <Link className="live-more" to="/changes">全部变更 →</Link>
        </div>
      )}

      <section className="home-steps">
        {[
          ['每 2 小时真实探针', '可用性、p50/p95 延迟、思考签名抽检，全部来自连续探测。'],
          ['评分只认数据', 'Wilson 下界 + 覆盖加权，赞助位与广告不参与排名。'],
          ['选站、比价、告警', '目录看覆盖与支付，模型页比价，变更可订阅。'],
        ].map(([title, desc]) => (
          <div key={title} className="home-step">
            <b>{title}</b>
            <p>{desc}</p>
          </div>
        ))}
      </section>

      <div className="toolbar">
        <div className="chips">
          {SERVICES.map(([k, label]) => (
            <button key={k} className={`chip ${service === k ? 'on' : ''}`} onClick={() => setService(k)}>
              {label}
              <span className="chip-count">{serviceCounts[k] ?? 0}</span>
            </button>
          ))}
          <span className="chip-sep" />
          <label className="chip-select">
            <span className="dim">纯度</span>
            <select value={tier} onChange={(e) => setTier(e.target.value)} aria-label="按纯度筛选">
              {TIERS.map(([k, label]) => (
                <option key={k} value={k}>{label}（{tierCounts[k] ?? 0}）</option>
              ))}
            </select>
          </label>
          <span className="chip-sep" />
          <button className={`chip ${onlyWatch ? 'on' : ''}`} onClick={() => setOnlyWatch((v) => !v)}>
            ★ 关注 {watched.length}
          </button>
        </div>
        <div className="toolbar-right">
          <div className="search">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <circle cx="7" cy="7" r="4.6" stroke="currentColor" strokeWidth="1.3" />
              <path d="m10.6 10.6 3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
            </svg>
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索站点 / 域名" aria-label="搜索站点" />
          </div>
          <select value={sort} onChange={(e) => setSort(e.target.value)} aria-label="排序方式">
            {SORTS.map(([k, label]) => (
              <option key={k} value={k}>按{label}</option>
            ))}
          </select>
          {(q || service !== 'all' || tier !== 'all' || sort !== 'rank' || onlyWatch) && (
            <button type="button" className="link-btn" onClick={resetFilters}>清除筛选</button>
          )}
        </div>
      </div>

      <section className="panel" id="ranking">
        <div className="panel-head">
          <div>
            <h2>中转站实时榜单</h2>
            <p className="sub">
              共 {stations?.length ?? 0} 家 · 当前 {rows.length} 家
              {rows.length > shown.length ? `（显示前 ${shown.length}）` : ''} · 点击任意行查看分组与完整支持模型
            </p>
          </div>
          <div className="panel-head-right">
            <span className="mono dim">
              数据时间 {meta ? new Date(meta.dataUpdatedAt).toLocaleString('zh-CN', { hour12: false }) : '—'}
            </span>
            <button className="btn btn-ghost btn-xs" onClick={exportCsv} disabled={rows.length === 0}>导出 CSV</button>
          </div>
        </div>

        {error && <div className="error-box">接口请求失败：{error}</div>}

        <div className="table-wrap home-table">
          <table className="index-table">
            <thead>
              <tr>
                <th className="col-rank">#</th>
                <th>站点</th>
                <th className="hide-md">线路</th>
                <th className="num">排名分</th>
                <th title="原始可用率；排名使用 95% 置信下界">可用率</th>
                <th className="num hide-sm">速度分</th>
                <th>纯度</th>
                <th className="num hide-sm">模型</th>
                <th className="num">最低输入价</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {!stations && !error && (
                Array.from({ length: 7 }).map((_, i) => (
                  <tr key={`sk-${i}`} className="sk-row">
                    <td><span className="sk" style={{ width: 20 }} /></td>
                    <td><span className="sk" style={{ width: 150 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                    <td><span className="sk" style={{ width: 90 }} /></td>
                    <td><span className="sk" style={{ width: 110 }} /></td>
                    <td><span className="sk" style={{ width: 50 }} /></td>
                  </tr>
                ))
              )}
              {stations && rows.length === 0 && (
                <tr><td colSpan={10} className="loading">没有符合条件的站点，试试放宽筛选。</td></tr>
              )}
              {shown.map((s) => (
                <tr key={s.slug} onClick={() => navigate(`/stations/${s.slug}`)}>
                  <td className={`col-rank rank r${Math.min(s.rank, 4)}`}>{s.rank || '—'}</td>
                  <td>
                    <div className="cell-station">
                      <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={34} />
                      <div className="station-meta">
                        <div className="station-line">
                          <Link to={`/stations/${s.slug}`} className="station-name" onClick={(e) => e.stopPropagation()}>
                            {s.name}
                          </Link>
                          {s.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
                          {s.newDomain && <span className="badge-warn">新站</span>}
                          <WatchStar slug={s.slug} size={13} />
                        </div>
                        <span className="station-domain mono">{s.domain}</span>
                      </div>
                    </div>
                  </td>
                  <td className="hide-md">
                    <div className="services compact" title={s.services.map(serviceLabel).join(' / ')}>
                      {s.services.slice(0, 2).map((sv) => (
                        <span key={sv} className="svc">{serviceLabel(sv)}</span>
                      ))}
                      {s.services.length > 2 && (
                        <span className="svc-more mono">+{s.services.length - 2}</span>
                      )}
                    </div>
                  </td>
                  <td
                    className="num"
                    title={`综合分 = 可用性 ${s.availability}×0.62 + 速度 ${s.speed}×0.26 + 稳定性 ${s.consistency}×0.12`}
                  >
                    <b className="score">{s.rankScore.toFixed(1)}</b>
                  </td>
                  <td>
                    <div className="avail" title={`可用性指数 ${s.availability.toFixed(1)}（排名用，含平滑）`}>
                      <span className="mono">{s.uptimePct.toFixed(1)}%</span>
                      <UptimeBars data={s.uptime} />
                    </div>
                  </td>
                  <td className="num hide-sm mono dim">{s.speed.toFixed(0)}</td>
                  <td><TierBadge tier={s.purityTier} score={s.purity} /></td>
                  <td className="num hide-sm mono dim">{s.modelsCount}</td>
                  <td className="num">
                    <span className="price">{fmtCny(s.fromPriceCny)}</span>
                    <small className="per">/M</small>
                  </td>
                  <td className="col-go"><Arrow /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="station-cards">
          {!stations && Array.from({ length: 5 }).map((_, i) => (
            <div key={`skc-${i}`} className="station-card">
              <div className="sc-row1"><span className="sk" style={{ width: 26 }} /><span className="sk" style={{ width: 30, height: 30 }} /><span className="sk" style={{ width: 130 }} /></div>
              <div className="sc-row2"><span className="sk" style={{ width: 90 }} /><span className="sk" style={{ width: 60 }} /></div>
            </div>
          ))}
          {stations && rows.length === 0 && <p className="fine" style={{ padding: 16 }}>没有符合条件的站点，试试放宽筛选。</p>}
          {shown.map((s) => (
            <Link key={s.slug} to={`/stations/${s.slug}`} className="station-card">
              <div className="sc-row1">
                <span className={`rank r${Math.min(s.rank, 4)}`}>{s.rank || '—'}</span>
                <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                <div className="sc-id">
                  <b>{s.name}</b>
                  <small className="mono">{s.domain}</small>
                </div>
                <span className="sc-score mono">{s.rankScore.toFixed(1)}<small>分</small></span>
              </div>
              <div className="sc-row2">
                <TierBadge tier={s.purityTier} score={s.purity} />
                <span className="sc-uptime mono">{s.uptimePct.toFixed(1)}%</span>
                <UptimeBars data={s.uptime} width={3} />
                <span className="sc-price mono">{fmtCny(s.fromPriceCny)}<small>/M</small></span>
              </div>
            </Link>
          ))}
        </div>

        {rows.length > shown.length && (
          <div className="list-more">
            <button className="btn btn-ghost btn-sm" onClick={() => setVisible((v) => v + 120)}>
              显示更多站点
            </button>
            <span className="mono">已显示 {shown.length} / {rows.length}</span>
          </div>
        )}

        <div className="panel-foot mono dim">
          价格 = 官方 USD 单价 × 站点结算汇率 × 分组倍率；取该站全部模型中的最低输入价，非单一模型报价。
        </div>
      </section>

      <section className="home-section">
        <div className="home-sec-head">
          <div>
            <h2>热门模型最低价</h2>
            <p className="sub">同一模型在不同中转分组差价可达数倍，这里只取当前最低输入价。</p>
          </div>
          <Link className="link-btn" to="/models">全部模型 →</Link>
        </div>
        <div className="home-model-grid">
          {models.filter((m) => m.minInputCny > 0).sort((a, b) => b.codingScore - a.codingScore).slice(0, 4).map((m) => (
            <Link key={m.id} className="home-model-card" to={`/models/${encodeURIComponent(m.id)}`}>
              <div className="hmc-head">
                <b>{m.name}</b>
                <span className={`tier-dot ${m.vendor}`} />
              </div>
              <div className="hmc-price">{fmtCny(m.minInputCny)}<small>/M 输入</small></div>
              <div className="hmc-meta mono">
                编程 {m.codingScore.toFixed(1)} · {m.offersCount} 条报价
              </div>
            </Link>
          ))}
          {models.length === 0 && <p className="fine">加载模型报价中…</p>}
        </div>
      </section>

      <section className="home-section">
        <div className="home-sec-head">
          <div>
            <h2>先看方法，再选站</h2>
          </div>
          <Link className="link-btn" to="/guides">全部指南 →</Link>
        </div>
        <div className="home-guide-grid">
          {[
            ['how-to-choose-relay', '10 分钟选站清单', '从可用性、倍率、退款与发票四个维度快速排除。'],
            ['multiplier-math', '倍率与计费怎么算', '别只看“几折”，把倍率、汇率和缓存折算成真实成本。'],
            ['signature-verification', '验证 Claude 是否掺水', 'thinking 签名、上下文与输出一致性怎么测。'],
          ].map(([slug, title, desc]) => (
            <Link key={slug} className="home-guide-card" to={`/guides/${slug}`}>
              <b>{title}</b>
              <span>{desc}</span>
              <em>阅读 →</em>
            </Link>
          ))}
        </div>
      </section>

      <section className="home-cta-band">
        <div>
          <h2>你的站点值得被看见</h2>
          <p className="sub">提交站点进入监测，或认领档案维护商务信息；评分与探针数据不受影响。</p>
        </div>
        <div className="home-cta-actions">
          <Link className="btn btn-primary" to="/submit">提交站点</Link>
          <Link className="btn btn-ghost" to="/submit?claim=1">认领档案</Link>
        </div>
      </section>
    </main>
  );
}

// Animate a number from 0 on mount, respecting reduced-motion.
function useCountUp(target: number | undefined, duration = 900) {
  const [value, setValue] = useState(0);
  useEffect(() => {
    if (!target) {
      setValue(0);
      return;
    }
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      setValue(target);
      return;
    }
    let raf = 0;
    const start = performance.now();
    const tick = (now: number) => {
      const p = Math.min(1, (now - start) / duration);
      setValue(Math.round(target * (1 - Math.pow(1 - p, 3))));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, duration]);
  return value;
}
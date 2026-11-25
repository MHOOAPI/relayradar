import '../styles/home.css';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { Change, Meta, ModelSummary, Station } from '../types';
import { Arrow, fmtCny, fmtNum, TierBadge, UptimeBars, WatchStar } from '../components/ui';
import { StationMark } from '../components/marks';
import { serviceLabel } from '../lib/services';
import { KeyCheck } from '../components/KeyCheck';
import { useWatchlist } from '../lib/watchlist';
import { usePointerGlow, useScrollParallax } from '../lib/parallax';
import { copyText } from '../lib/clipboard';
import { useToast } from '../lib/toast';

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
  ['watched', '已关注优先'],
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
  const { push } = useToast();
  const [copied, setCopied] = useState(false);
  const [menuCopied, setMenuCopied] = useState(false);
  const [copyFailed, setCopyFailed] = useState(false);
  const copyFilterLink = async () => {
    const ok = await copyText(window.location.href);
    setCopied(ok);
    setCopyFailed(!ok);
    push(ok ? '已复制筛选链接' : '复制失败，请手动选择地址栏', ok ? 'ok' : 'err');
    if (ok) window.setTimeout(() => setCopied(false), 1600);
    return ok;
  };
  const copyFromMenu = async () => {
    const ok = await copyFilterLink();
    setMenuCopied(ok);
    window.setTimeout(() => {
      setMenuCopied(false);
      if (headMoreRef.current) headMoreRef.current.open = false;
    }, 1200);
  };
  const radarRef = useRef<HTMLDivElement>(null);
  const radarInnerRef = useRef<HTMLSpanElement>(null);
  const heroRef = useRef<HTMLElement>(null);
  const heroCardRef = useRef<HTMLDivElement>(null);
  const rankingRef = useRef<HTMLElement>(null);
  const toolbarRef = useRef<HTMLDivElement>(null);
  const chipsRef = useRef<HTMLDivElement>(null);
  const headMoreRef = useRef<HTMLDetailsElement>(null);

  useEffect(() => {
    const onDown = (e: PointerEvent) => {
      const el = headMoreRef.current;
      if (el?.open && !el.contains(e.target as Node)) el.open = false;
    };
    const onKey = (e: KeyboardEvent) => {
      const el = headMoreRef.current;
      if (e.key === 'Escape' && el?.open) el.open = false;
    };
    document.addEventListener('pointerdown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('pointerdown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, []);

  useEffect(() => {
    const el = toolbarRef.current;
    if (!el) return;
    const set = () => {
      document.documentElement.style.setProperty('--rank-head-top', `${64 + el.offsetHeight}px`);
      document.documentElement.style.setProperty('--toolbar-h', `${el.offsetHeight}px`);
      const chips = chipsRef.current;
      if (chips) chips.classList.toggle('is-scrollable', chips.scrollWidth > chips.clientWidth + 1);
    };
    set();
    const ro = new ResizeObserver(set);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  useScrollParallax(radarRef, 0.16, 700);
  usePointerGlow(rankingRef, !!stations);

  // Pointer parallax inside the hero: the radar leans toward the cursor a few
  // pixels. Desktop pointers only, and never when reduced motion is requested.
  useEffect(() => {
    const hero = heroRef.current;
    if (!hero) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if (!window.matchMedia('(pointer: fine)').matches) return;
    let raf = 0;
    let tx = 0;
    let ty = 0;
    const onMove = (e: MouseEvent) => {
      const rect = hero.getBoundingClientRect();
      tx = ((e.clientX - rect.left) / rect.width - 0.5) * 14;
      ty = ((e.clientY - rect.top) / rect.height - 0.5) * 10;
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        const el = radarInnerRef.current;
        if (el) el.style.transform = `translate3d(${tx.toFixed(1)}px, ${ty.toFixed(1)}px, 0)`;
        const card = heroCardRef.current;
        if (card) card.style.transform = `translate3d(${(-tx * 0.35).toFixed(1)}px, ${(-ty * 0.35).toFixed(1)}px, 0)`;
      });
    };
    const onLeave = () => {
      const el = radarInnerRef.current;
      if (el) el.style.transform = 'translate3d(0, 0, 0)';
      const card = heroCardRef.current;
      if (card) card.style.transform = 'translate3d(0, 0, 0)';
    };
    hero.addEventListener('mousemove', onMove);
    hero.addEventListener('mouseleave', onLeave);
    return () => {
      hero.removeEventListener('mousemove', onMove);
      hero.removeEventListener('mouseleave', onLeave);
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
    const hash = window.location.hash;
    setParams(p, { replace: true });
    // React Router's replace drops the fragment; keep #ranking deep links working.
    if (hash) window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}${hash}`);
  }, [q, service, tier, sort, onlyWatch, setParams]);

  useEffect(() => {
    if (!stations || window.location.hash !== '#ranking') return;
    const t = window.setTimeout(() => document.getElementById('ranking')?.scrollIntoView({ block: 'start' }), 120);
    return () => window.clearTimeout(t);
  }, [stations]);

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
    if (sort === 'availability') list.sort((a, b) => b.uptimePct - a.uptimePct);
    else if (sort === 'speed') list.sort((a, b) => b.speed - a.speed);
    else if (sort === 'price') list.sort((a, b) => (a.fromPriceCny < 0 ? 1 : b.fromPriceCny < 0 ? -1 : a.fromPriceCny - b.fromPriceCny));
    else if (sort === 'models') list.sort((a, b) => b.modelsCount - a.modelsCount);
    else if (sort === 'watched') list.sort((a, b) => (watched.includes(a.slug) ? 0 : 1) - (watched.includes(b.slug) ? 0 : 1) || a.rank - b.rank);
    else list.sort((a, b) => a.rank - b.rank);
    return list;
  }, [stations, service, tier, sort, q, onlyWatch, watched]);

  const [visible, setVisible] = useState(() => (typeof window !== 'undefined' && window.matchMedia('(max-width: 760px)').matches ? 15 : 50));
  const pageSize = () => (typeof window !== 'undefined' && window.matchMedia('(max-width: 760px)').matches ? 30 : 50);
  useEffect(() => setVisible(pageSize()), [q, service, tier, sort, onlyWatch]);
  const shown = rows.slice(0, visible);
  const unscoredCount = rows.filter((s) => s.rankScore === 0).length;
  const [jump, setJump] = useState('');
  const jumpToRank = () => {
    const n = Math.max(1, Math.min(rows.length, Number(jump) || 0));
    if (!n) return;
    const targetIndex = sort === 'rank' ? rows.findIndex((s) => s.rank === n) : n - 1;
    if (targetIndex < 0) return;
    setVisible((v) => Math.max(v, targetIndex + 6));
    window.setTimeout(() => {
      const mobile = window.matchMedia('(max-width: 760px)').matches;
      const list = mobile
        ? document.querySelectorAll('.station-card')
        : document.querySelectorAll('.home-table tbody tr');
      (list[targetIndex] as HTMLElement | undefined)?.scrollIntoView({ block: 'center' });
    }, 180);
  };

  const heroTop = useMemo(
    () => [...(stations ?? [])].filter((s) => s.rank > 0 && s.rankScore > 0).sort((a, b) => a.rank - b.rank).slice(0, 5),
    [stations],
  );

  const radarPoints = useMemo(() => {
    const list = (stations ?? [])
      .filter((s) => s.reachability && s.reachability.latencyMs > 0)
      .slice(0, 28);
    if (!list.length) return [];
    const frac = (ms: number) => {
      const t = Math.min(Math.max(ms, 50), 4000);
      if (t <= 100) return 0.06 + ((Math.log(t) - Math.log(50)) / (Math.log(100) - Math.log(50))) * 0.1;
      if (t <= 500) return 0.16 + ((Math.log(t) - Math.log(100)) / (Math.log(500) - Math.log(100))) * 0.12;
      if (t <= 2000) return 0.28 + ((Math.log(t) - Math.log(500)) / (Math.log(2000) - Math.log(500))) * 0.13;
      return 0.41 + Math.min((t - 2000) / 2000, 1) * 0.06;
    };
    return list.map((s, i) => {
      const angle = (i / list.length) * Math.PI * 2 - Math.PI / 2;
      const radius = frac(s.reachability!.latencyMs);
      const tone = s.purityTier === 'official' ? 'official' : s.purityTier === 'mixed' ? 'mixed' : 'unknown';
      return {
        slug: s.slug,
        name: s.name,
        latency: s.reachability!.latencyMs,
        left: 50 + Math.cos(angle) * radius * 100,
        top: 50 + Math.sin(angle) * radius * 100,
        tone,
      };
    });
  }, [stations]);

  const hotModels = useMemo(
    () =>
      (models ?? [])
        .filter((m) => m.minInputCny > 0 && !/(?:^|[-:])batch|(?:^|[-:])free/i.test(m.id))
        .sort((a, b) => b.offersCount - a.offersCount)
        .slice(0, 4),
    [models],
  );

  const priceAnomalyCount = hotModels.filter((m) => {
    const official = m.inputUsd * 7.2;
    const pct = official > 0 ? (m.minInputCny / official) * 100 : 0;
    return pct > 0 && pct < 5;
  }).length;

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
      <section className="hero" ref={heroRef} data-anim-scope>
        <span className="hero-scan" aria-hidden="true"><i /></span>
        <div className="hero-radar" aria-hidden="true" ref={radarRef}>
          <span className="radar-inner" ref={radarInnerRef}>
            <span className="hr-ring r1" />
            <span className="hr-ring r2" />
            <span className="hr-ring r3" />
            {radarPoints.length > 0 && (
              <>
                <span className="hr-tick t1 mono">100ms</span>
                <span className="hr-tick t2 mono">500ms</span>
                <span className="hr-tick t3 mono">2s</span>
              </>
            )}
            <span className="hr-sweep" />
            {radarPoints.length > 0 ? (
              radarPoints.map((b) => (
                <span key={b.slug} className={`hr-blip ${b.tone}`} style={{ left: `${b.left}%`, top: `${b.top}%` }} />
              ))
            ) : (
              <>
                <span className="hr-blip b1" />
                <span className="hr-blip b2" />
                <span className="hr-blip b3" />
              </>
            )}
          </span>
        </div>
        <div className="hero-copy">
          <h1>
            别猜哪家中转站
            <br />
            <em>掺了水</em>，看数据。
          </h1>
          <p>
            {meta?.demo
              ? '独立探针 + 可复算评分，赞助位与广告不参与排名。'
              : '每 2 小时一次真实探针，按可用性、速度、思考签名抽检和真实倍率排序。赞助位与广告不参与排名。'}
          </p>
          <div className="hero-cta">
            <a className="btn btn-primary" href="#ranking">看实时榜单</a>
            <Link to="/methodology" className="btn btn-ghost">评分怎么算的</Link>
          </div>
          <div className="hero-formula-line mono">
            <span>综合分 = 62% 可用性 + 26% 速度 + 12% 稳定性</span>
            <Link to="/methodology">看完整公式与已知局限 →</Link>
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
        </div>

        <div className="hero-board" ref={heroCardRef}>
          <div className="hero-board-head">
            <span className="mono label">此刻前 5</span>
          </div>
          <ol className="hero-board-list">
            {heroTop.map((s) => (
              <li key={s.slug}>
                <Link to={`/stations/${s.slug}`} className="hb-row">
                  <span className="hb-rank mono">{s.rank}</span>
                  <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={22} />
                  <span className="hb-name">
                    {s.name}
                    <small className="mono">{s.domain}</small>
                  </span>
                  <span className="hb-score mono">{s.rankScore.toFixed(1)}</span>
                  <span className="hb-price mono">{fmtCny(s.fromPriceCny)}</span>
                </Link>
              </li>
            ))}
            {heroTop.length === 0 && <li className="hb-empty">{stations ? '暂无已评分站点' : '加载榜单…'}</li>}
          </ol>
          <p className="hb-foot mono">
            {meta ? fmtNum(heroStations) : '—'} 站点 · {meta ? fmtNum(heroGroups) : '—'} 分组 · {meta ? fmtNum(heroProbes) : '—'} 探针
          </p>
          {radarPoints.length > 0 && (
            <p className="hb-radar-note mono">
              雷达：已绘制前 {radarPoints.length}/{meta?.stations ?? '—'} 站 · 圈=延迟(对数) · 角度=排名 · 色=纯度
            </p>
          )}
        </div>
      </section>

      {changes.length > 0 && (
        <div className="live-strip" aria-label={meta?.demo ? '示例变更' : '最近变更'} data-anim-scope>
          <span className={`live-strip-label ${meta?.demo ? 'demo' : ''}`}>
            <i className="live-dot" />{meta?.demo ? '示例变更' : 'LIVE'}
          </span>
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
          meta?.demo
            ? ['演示数据集', '探针、评分与价格为示例值；评分方法与真实部署一致。']
            : ['每 2 小时真实探针', '可用性、p50/p95 延迟、思考签名抽检，全部来自连续探测。'],
          ['评分只认数据', 'Wilson 下界 + 覆盖加权，赞助位与广告不参与排名。'],
          ['选站、比价、告警', '目录看覆盖与支付，模型页比价，变更可订阅。'],
        ].map(([title, desc]) => (
          <div key={title} className="home-step">
            <b>{title}</b>
            <p>{desc}</p>
          </div>
        ))}
      </section>

      <div className="ranking-scope">
        <div className="toolbar" ref={toolbarRef}>
        <div className="chips" ref={chipsRef}>
          {SERVICES.map(([k, label]) => (
            <button key={k} className={`chip ${service === k ? 'on' : ''}`} onClick={() => setService(k)}>
              {label}
              <span className="chip-count">{serviceCounts[k] ?? 0}</span>
            </button>
          ))}
        </div>
        <div className="toolbar-right">
          <label className="chip-select">
            <span className="dim">纯度</span>
            <select value={tier} onChange={(e) => setTier(e.target.value)} aria-label="按纯度筛选">
              {TIERS.map(([k, label]) => (
                <option key={k} value={k}>{label}（{tierCounts[k] ?? 0}）</option>
              ))}
            </select>
          </label>
          <button className={`chip ${onlyWatch ? 'on' : ''}`} aria-pressed={onlyWatch} onClick={() => setOnlyWatch((v) => !v)}>
            ★ 关注 {watched.length}
          </button>
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

      <section className="panel glow-surface" id="ranking" ref={rankingRef}>
        <div className="panel-head">
          <div>
            <h2>中转站实时榜单</h2>
            <p className="sub">
              共 {stations?.length ?? 0} 家 · 当前 {rows.length} 家
              {rows.length > shown.length ? `（显示前 ${shown.length}）` : ''}
              {unscoredCount > 0 ? ` · ${unscoredCount} 家未评分` : ''}
              <span className="head-hint"> · 点击行查看分组与完整支持模型</span>
            </p>
          </div>
          <div className="panel-head-right">
            <span
              className={`prov-badge ${meta?.demo ? 'demo' : 'live'}`}
              title={meta?.demo ? '探针、评分与价格为生成值，仅用于演示' : '来自真实探针与站点公开价格'}
            >
              {meta?.demo ? '演示数据' : '实测数据'}
            </span>
            <span className="mono dim head-time desktop-only">
              {meta?.demo ? '生成时间' : '数据时间'} {meta ? new Date(meta.dataUpdatedAt).toLocaleString('zh-CN', { hour12: false }) : '—'}
            </span>
            <button className="btn btn-ghost btn-xs head-action" onClick={copyFilterLink}>{copied ? '已复制 ✓' : '复制筛选链接'}</button>
            <button className="btn btn-ghost btn-xs head-action" onClick={exportCsv} disabled={rows.length === 0}>导出 CSV</button>
            <details
              className="head-more"
              ref={headMoreRef}
              onToggle={(e) => { if (e.currentTarget.open) { setMenuCopied(false); setCopyFailed(false); } }}
            >
              <summary aria-label="更多操作">⋯</summary>
              <div className="head-more-menu">
                <button type="button" onClick={copyFromMenu}>{menuCopied ? '已复制 ✓' : copyFailed ? '复制失败' : '复制筛选链接'}</button>
                <button type="button" onClick={() => { if (headMoreRef.current) headMoreRef.current.open = false; exportCsv(); }} disabled={rows.length === 0}>导出 CSV</button>
              </div>
            </details>
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
                <th className="num">
                  <button type="button" className="th-sort" onClick={() => setSort('rank')} aria-pressed={sort === 'rank'}>
                    排名分 <small>0–100</small><span aria-hidden="true">{sort === 'rank' ? '↓' : '↕'}</span>
                  </button>
                </th>
                <th>
                  <button type="button" className="th-sort" onClick={() => setSort('availability')} aria-pressed={sort === 'availability'}>
                    可用率<span aria-hidden="true">{sort === 'availability' ? '↓' : '↕'}</span>
                  </button>
                </th>
                <th className="num hide-sm">
                  <button type="button" className="th-sort" onClick={() => setSort('speed')} aria-pressed={sort === 'speed'}>
                    速度分 <small>0–100</small><span aria-hidden="true">{sort === 'speed' ? '↓' : '↕'}</span>
                  </button>
                </th>
                <th>纯度 <small>0–100</small></th>
                <th className="num hide-sm">
                  <button type="button" className="th-sort" onClick={() => setSort('models')} aria-pressed={sort === 'models'}>
                    模型 <small>个</small><span aria-hidden="true">{sort === 'models' ? '↓' : '↕'}</span>
                  </button>
                </th>
                <th className="num">
                  <button type="button" className="th-sort" onClick={() => setSort('price')} aria-pressed={sort === 'price'}>
                    最低输入价 <small>¥/M</small><span aria-hidden="true">{sort === 'price' ? '↑' : '↕'}</span>
                  </button>
                </th>
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
                          <button
                            type="button"
                            className="row-compare"
                            title="把该站加入横向对比"
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate(`/compare?ids=${encodeURIComponent(s.slug)}`);
                            }}
                          >
                            对比
                          </button>
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
                    title={s.rankScore > 0 ? `综合分 = 可用性 ${s.availability}×0.62 + 速度 ${s.speed}×0.26 + 稳定性 ${s.consistency}×0.12` : '暂无评分：探针样本不足，先看可用率'}
                  >
                    <b className={`score ${s.rankScore > 0 ? '' : 'dim'}`}>{s.rankScore > 0 ? s.rankScore.toFixed(1) : '未评分'}</b>
                  </td>
                  <td>
                    <div className="avail" title={`可用性指数 ${s.availability.toFixed(1)}（排名用，含平滑）`}>
                      <span className="mono">{s.uptimePct.toFixed(1)}%</span>
                      <UptimeBars data={s.uptime} />
                    </div>
                  </td>
                  <td className="num hide-sm mono dim">{s.rankScore > 0 ? s.speed.toFixed(0) : '—'}</td>
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
            <div key={s.slug} className="station-card">
              <Link to={`/stations/${s.slug}`} className="sc-link">
                <div className="sc-row1">
                  <span className={`rank r${Math.min(s.rank, 4)}`}>{s.rank || '—'}</span>
                  <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                  <div className="sc-id">
                    <b>{s.name}</b>
                    <small className="mono">{s.domain}</small>
                  </div>
                  <span className={`sc-score mono ${s.rankScore > 0 ? '' : 'dim'}`}>{s.rankScore > 0 ? s.rankScore.toFixed(1) : '未评分'}<small>{s.rankScore > 0 ? '分' : ''}</small></span>
                </div>
                <div className="sc-row2">
                  <TierBadge tier={s.purityTier} score={s.purity} />
                  <span className="sc-uptime mono">{s.uptimePct.toFixed(1)}%</span>
                  <UptimeBars data={s.uptime} width={3} />
                  <span className="sc-price mono">{fmtCny(s.fromPriceCny)}<small>/M</small></span>
                </div>
                <div className="sc-services">
                  {s.services.slice(0, 3).map((sv) => (
                    <span key={sv}>{SERVICES.find(([k]) => k === sv)?.[1] ?? sv}</span>
                  ))}
                  {s.services.length > 3 && <span>+{s.services.length - 3}</span>}
                </div>
              </Link>
              <div className="sc-actions">
                <WatchStar slug={s.slug} size={14} />
                <button
                  type="button"
                  className="row-compare always"
                  onClick={() => navigate(`/compare?ids=${encodeURIComponent(s.slug)}`)}
                >
                  对比
                </button>
              </div>
            </div>
          ))}
        </div>

        {rows.length > shown.length && (
          <div className="list-more">
            <button className="btn btn-ghost btn-sm" onClick={() => setVisible((v) => v + 45)}>
              再看 45 家
            </button>
            <Link className="btn btn-ghost btn-sm" to="/stations">完整榜单 →</Link>
            <span className="mono">已显示 {shown.length} / {rows.length}</span>
          </div>
        )}

        <div className="panel-foot mono dim">
          <span>价格 = 官方 USD 单价 × 站点结算汇率 × 分组倍率；取该站全部模型中的最低输入价，非单一模型报价。</span>
          <label className="rank-jump">
            跳到
            <input
              type="number"
              min={1}
              max={rows.length}
              value={jump}
              onChange={(e) => setJump(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); jumpToRank(); } }}
              aria-label={sort === 'rank' ? '跳到第几名' : '跳到第几行'}
            />
            {sort === 'rank' ? '名' : '行'}
          </label>
          <span className="foot-time">{meta ? new Date(meta.dataUpdatedAt).toLocaleString('zh-CN', { hour12: false }) : '—'}</span>
        </div>
      </section>
      </div>

      <KeyCheck />
      <p className="home-alert-cta mono">
        <Link to="/changes?subscribe=1">关注站点后，降价与宕机会进入变更雷达</Link>
        <span className="dim"> · </span>
        <Link to="/changes?subscribe=1">订阅告警 →</Link>
      </p>

      <section className="home-section">
        <div className="home-sec-head">
          <div>
            <h2>热门模型最低价</h2>
            <p className="sub">按报价站点数取前 4，已排除 batch / free 变体；价格为该模型的全网最低输入价。</p>
            {priceAnomalyCount > 0 && (
              <p className="fine home-anomaly-note">
                {priceAnomalyCount} 个模型的官方价与最低价差超过 95%，先核对分组倍率与纯度。
              </p>
            )}
          </div>
          <Link className="link-btn" to="/models">全部模型 →</Link>
        </div>
        <div className="home-model-grid">
          {hotModels.map((m) => {
            const officialCny = m.inputUsd * 7.2;
            const pct = officialCny > 0 ? (m.minInputCny / officialCny) * 100 : 0;
            return (
              <Link key={m.id} className="home-model-card" to={`/models/${encodeURIComponent(m.id)}`}>
                <div className="hmc-head">
                  <b>{m.name}</b>
                  <span className={`tier-dot ${m.vendor}`} />
                </div>
                <div className="hmc-price">{fmtCny(m.minInputCny)}<small>/M 输入</small></div>
                <div className="hmc-meta mono" title={officialCny > 0 ? '官方价 × 7.2 与本站最低价的比值；差异大时请核对分组倍率' : undefined}>
                  {m.offersCount} 站报价
                  {officialCny > 0 ? ` · 官方 ¥${officialCny.toFixed(1)} → ${pct.toFixed(1)}%` : ''}
                </div>
              </Link>
            );
          })}
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
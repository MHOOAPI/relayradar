import '../styles/models.css';
import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny, fmtUsd } from '../components/ui';
import { VendorMark } from '../components/marks';

const fmtCtx = (n: number) => (n >= 1000000 ? `${(n / 1000000).toFixed(n % 1000000 === 0 ? 0 : 2)}M` : `${Math.round(n / 1000)}K`);

const CATEGORIES = [
  ['all', '全部'],
  ['text', '文本'],
  ['image', '图片'],
  ['video', '视频'],
] as const;

const CATEGORY_LABEL: Record<string, string> = { text: '文本', image: '图片', video: '视频' };

// Batch pricing and free tiers answer a different question than "what does this
// model cost per token", so they never appear in the default shortlist. They
// stay reachable from the full catalog.
const VARIANT_RE = /(?:^|[-:])batch|(?:^|[-:])free/i;

const hasOffer = (m: ModelSummary) => m.offersCount > 0 && !VARIANT_RE.test(m.id);

// Most-listed first: the number of stations offering a model is the only
// popularity signal we actually measure, and priced rows are what this page is
// about. Ties stay grouped by vendor so families read together.
const byOffers = (a: ModelSummary, b: ModelSummary) =>
  b.offersCount - a.offersCount ||
  a.vendorName.localeCompare(b.vendorName, 'zh-CN') ||
  a.name.localeCompare(b.name, 'zh-CN');

export function ModelsPage() {
  const [items, setItems] = useState<ModelSummary[] | null>(null);
  const [params, setParams] = useSearchParams();
  const [q, setQ] = useState(params.get('q') ?? '');
  const [cat, setCat] = useState(params.get('cat') ?? 'all');
  const [all, setAll] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (cat !== 'all') p.set('cat', cat);
    setParams(p, { replace: true });
  }, [q, cat, setParams]);
  const navigate = useNavigate();

  useEffect(() => {
    api.models().then((r) => setItems(r.items)).catch((e: Error) => setError(e.message));
  }, []);

  const priced = useMemo(() => (items ?? []).filter(hasOffer), [items]);
  // With no prices in the dataset the shortlist would be empty; fall back to the
  // full catalog instead of showing a blank table.
  const showAll = all || (items !== null && priced.length === 0);
  const scopeItems = useMemo(() => (items ? (showAll ? items : priced) : []), [items, showAll, priced]);

  const spotlight = useMemo(() => {
    if (!items) return [] as Array<{ m: ModelSummary; label: string }>;
    const pool = items.filter(hasOffer);
    const picks: Array<{ m: ModelSummary; label: string }> = [];
    const add = (m: ModelSummary | undefined, label: string) => {
      if (m && !picks.some((p) => p.m.id === m.id)) picks.push({ m, label });
    };
    add([...pool].sort((a, b) => b.relayScore - a.relayScore)[0], '供给评分最高');
    add([...pool].sort((a, b) => b.offersCount - a.offersCount)[0], '覆盖最广');
    add([...pool].sort((a, b) => Date.parse(b.releasedAt) - Date.parse(a.releasedAt))[0], '最新发布');
    add([...pool].filter((m) => m.minInputCny > 0).sort((a, b) => a.minInputCny - b.minInputCny)[0], '全网最低价');
    return picks.slice(0, 4);
  }, [items]);

  const spotMetric = (m: ModelSummary, label: string) => {
    if (label === '供给评分最高') return `供给 ${m.relayScore.toFixed(0)} · ${m.offersCount} 站报价`;
    if (label === '覆盖最广') return `${m.offersCount} 站报价 · ${fmtCny(m.minInputCny)}/M 起`;
    if (label === '最新发布') return `${new Date(m.releasedAt).toLocaleDateString('zh-CN')} 上架`;
    return `${fmtCny(m.minInputCny)}/M · ${m.offersCount} 站有货`;
  };

  const rows = useMemo(() => {
    const n = q.trim().toLowerCase();
    return scopeItems
      .filter((m) => (cat === 'all' || (m.category || 'text') === cat)
        && (!n || `${m.name} ${m.id} ${m.vendor}`.toLowerCase().includes(n)))
      .sort(byOffers);
  }, [scopeItems, q, cat]);

  const [visible, setVisible] = useState(60);
  useEffect(() => setVisible(60), [q, cat, showAll]);
  const shown = rows.slice(0, visible);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>模型与全网最低价</h1>
          <p className="sub">官方美元价 vs 中转站折算人民币实价，差价来自站点汇率与分组倍率。</p>
        </div>
        <div className="search">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <circle cx="7" cy="7" r="4.6" stroke="currentColor" strokeWidth="1.3" />
            <path d="m10.6 10.6 3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
          </svg>
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索模型 / 厂商" aria-label="搜索模型" />
        </div>
      </div>

      <div className="model-links mono">
        <Link to="/models/license">按许可证浏览 →</Link>
        <Link to="/compare/models">模型对比 →</Link>
      </div>

      {q === '' && cat === 'all' && spotlight.length > 0 && (
        <div className="model-spotlight">
          {spotlight.map(({ m, label }) => (
            <Link key={m.id} className="spot-card" to={`/models/${encodeURIComponent(m.id)}`}>
              <div className="spot-top">
                <VendorMark vendor={m.vendor} size={24} />
                <span className="spot-label mono">{label}</span>
              </div>
              <b className="spot-name">{m.name}</b>
              <span className="spot-metric mono">{spotMetric(m, label)}</span>
            </Link>
          ))}
        </div>
      )}

      <div className="cat-tabs">
        {CATEGORIES.map(([key, label]) => (
          <button key={key} className={`cat-tab ${cat === key ? 'on' : ''}`} onClick={() => setCat(key)}>
            {label}
            <span>{key === 'all' ? scopeItems.length : scopeItems.filter((m) => (m.category || 'text') === key).length}</span>
          </button>
        ))}
      </div>

      <p className="model-scope mono">
        {priced.length === 0 ? (
          <span className="dim">暂无站点报价 · 显示全部目录</span>
        ) : (
          <button className="link-btn" onClick={() => setAll(!showAll)}>
            {showAll
              ? `显示全部目录 ${items?.length ?? ''} · 回到有报价 ${priced.length}`
              : `只看有报价 ${priced.length} 个 · 展开全部目录 ${items?.length ?? ''}`}
          </button>
        )}
        <span className="dim"> · 当前 {rows.length} 个</span>
      </p>
      {!showAll && (
        <p className="fine model-scope-note">
          报价来自站点公开倍率；batch 与免费额度不计入，没有站点报价的模型仍可在全部目录中找到。
        </p>
      )}

      {error && <div className="error-box">{error}</div>}

      <section className="panel">
        <div className="table-wrap models-table">
          <table className="index-table">
            <thead>
              <tr>
                <th>模型</th><th className="hide-sm">厂商</th><th className="num hide-sm">上下文</th>
                <th className="num">官方价 /1M</th><th className="num">全网最低输入</th><th className="num">报价站点</th>
              </tr>
            </thead>
            <tbody>
              {!items && !error && (
                Array.from({ length: 6 }).map((_, i) => (
                  <tr key={`sk-${i}`} className="sk-row">
                    <td><span className="sk" style={{ width: 160 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                    <td><span className="sk" style={{ width: 60 }} /></td>
                    <td><span className="sk" style={{ width: 90 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                  </tr>
                ))
              )}
              {shown.map((m) => (
                <tr key={m.id} className="clickable" onClick={() => navigate(`/models/${encodeURIComponent(m.id)}`)}>
                  <td>
                    <div className="cell-station">
                      <VendorMark vendor={m.vendor} size={30} />
                      <div className="station-meta">
                        <Link className="station-name" to={`/models/${encodeURIComponent(m.id)}`} onClick={(e) => e.stopPropagation()}>{m.name}</Link>
                        <span className="mono dim tiny">{m.id}</span>
                      </div>
                      <span className="cat-chip">{CATEGORY_LABEL[m.category || 'text']}</span>
                    </div>
                  </td>
                  <td className="hide-sm dim">{m.vendorName}</td>
                  <td className="num mono dim hide-sm">{m.category === 'text' || !m.category ? fmtCtx(m.contextWindow) : '—'}</td>
                  <td className="num mono dim">{m.category && m.category !== 'text' ? '按张/秒计费' : m.inputUsd > 0 ? `${fmtUsd(m.inputUsd)} / ${fmtUsd(m.outputUsd)}` : '—'}</td>
                  <td className="num mono accent">{fmtCny(m.minInputCny)}</td>
                  <td className="num mono dim">{m.offersCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="model-cards">
          {!items && Array.from({ length: 5 }).map((_, i) => (
            <div key={`skc-${i}`} className="model-card">
              <span className="sk" style={{ width: 30, height: 30 }} />
              <div className="mc-id"><span className="sk" style={{ width: 140 }} /></div>
            </div>
          ))}
          {items && rows.length === 0 && <p className="fine" style={{ padding: 16 }}>没有匹配的模型。</p>}
          {shown.map((m) => (
            <Link key={m.id} className="model-card" to={`/models/${encodeURIComponent(m.id)}`}>
              <VendorMark vendor={m.vendor} size={30} />
              <div className="mc-id">
                <b>{m.name}</b>
                <small>{m.vendorName} · {CATEGORY_LABEL[m.category || 'text']} · {m.license}</small>
              </div>
              <div className="mc-price">
                <span>{fmtCny(m.minInputCny)}</span>
                <small>{m.offersCount} 站报价</small>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {rows.length > shown.length && (
        <div className="list-more">
          <button className="btn btn-ghost btn-sm" onClick={() => setVisible((v) => v + 120)}>
            显示更多模型
          </button>
          <span className="mono">已显示 {shown.length} / {rows.length}</span>
        </div>
      )}
    </main>
  );
}

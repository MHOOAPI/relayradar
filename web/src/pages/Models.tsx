import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny, fmtUsd } from '../components/ui';
import { VendorMark } from '../components/marks';

const CATEGORIES = [
  ['all', '全部'],
  ['text', '文本'],
  ['image', '图片'],
  ['video', '视频'],
] as const;

const CATEGORY_LABEL: Record<string, string> = { text: '文本', image: '图片', video: '视频' };

type SortKey = 'newest' | 'offers' | 'context' | 'price';

const SORTS: Array<[SortKey, string]> = [
  ['newest', '最新发布'],
  ['offers', '报价最多'],
  ['context', '上下文最大'],
  ['price', '价格最低'],
];

// 厂商 id 是拉丁 slug，有中文名的用中文名，其余保留官方写法。
const VENDOR_LABEL: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  google: 'Google',
  deepseek: 'DeepSeek',
  qwen: '阿里通义',
  'z-ai': '智谱',
  moonshotai: 'Moonshot',
  minimax: 'MiniMax',
  'x-ai': 'xAI',
  meta: 'Meta',
  'meta-llama': 'Meta',
  mistralai: 'Mistral',
  tencent: '腾讯',
};

const vendorName = (m: ModelSummary) =>
  VENDOR_LABEL[m.vendor.replace(/^~/, '')] ?? (m.vendorName !== m.vendor ? m.vendorName : m.vendor.replace(/^~/, ''));

const ctxLabel = (n: number) => {
  if (!n || n <= 0) return '—';
  if (n >= 1000000) return `${Number((n / 1000000).toFixed(2))}M`;
  return `${Math.round(n / 1000)}K`;
};

const releaseLabel = (iso: string) => {
  if (!iso || iso.startsWith('0001')) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
};

const releaseValue = (iso: string) => (iso && !iso.startsWith('0001') ? Date.parse(iso) || 0 : 0);

const priceLabel = (m: ModelSummary) => {
  if (m.minInputCny < 0) return '—';
  const unit = m.category === 'image' ? '/张' : m.category === 'video' ? '/秒' : '';
  return `${fmtCny(m.minInputCny)}${unit}`;
};

// 批量价与免费额度回答的是另一个问题，不进默认列表；完整目录里仍可看到。
const VARIANT_RE = /(?:^|[-:])batch|(?:^|[-:])free/i;
const hasOffer = (m: ModelSummary) => m.offersCount > 0 && !VARIANT_RE.test(m.id);

export function ModelsPage() {
  const [items, setItems] = useState<ModelSummary[] | null>(null);
  const [params, setParams] = useSearchParams();
  const [q, setQ] = useState(params.get('q') ?? '');
  const [cat, setCat] = useState(params.get('cat') ?? 'all');
  const [vendor, setVendor] = useState(params.get('vendor') ?? 'all');
  const [sort, setSort] = useState<SortKey>((params.get('sort') as SortKey) ?? 'newest');
  const [all, setAll] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const p = new URLSearchParams();
    if (q) p.set('q', q);
    if (cat !== 'all') p.set('cat', cat);
    if (vendor !== 'all') p.set('vendor', vendor);
    if (sort !== 'newest') p.set('sort', sort);
    setParams(p, { replace: true });
  }, [q, cat, vendor, sort, setParams]);

  useEffect(() => {
    api.models().then((r) => setItems(r.items)).catch((e: Error) => setError(e.message));
  }, []);

  const priced = useMemo(() => (items ?? []).filter(hasOffer), [items]);
  // 没有任何报价时默认列表会是空的，退回完整目录。
  const showAll = all || (items !== null && priced.length === 0);
  const scopeItems = useMemo(() => (items ? (showAll ? items : priced) : []), [items, showAll, priced]);

  const vendors = useMemo(() => {
    const map = new Map<string, { label: string; count: number }>();
    for (const m of items ?? []) {
      const cur = map.get(m.vendor);
      if (cur) cur.count += 1;
      else map.set(m.vendor, { label: vendorName(m), count: 1 });
    }
    return [...map.entries()]
      .map(([id, v]) => ({ id, ...v }))
      .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label, 'zh-CN'));
  }, [items]);

  const vendorScoped = useMemo(
    () => scopeItems.filter((m) => vendor === 'all' || m.vendor === vendor),
    [scopeItems, vendor],
  );

  const rows = useMemo(() => {
    const n = q.trim().toLowerCase();
    const list = vendorScoped.filter((m) => (cat === 'all' || (m.category || 'text') === cat)
      && (!n || `${m.name} ${m.id} ${vendorName(m)} ${m.vendor}`.toLowerCase().includes(n)));
    const byName = (a: ModelSummary, b: ModelSummary) => a.name.localeCompare(b.name, 'zh-CN');
    const cmp: Record<SortKey, (a: ModelSummary, b: ModelSummary) => number> = {
      newest: (a, b) => releaseValue(b.releasedAt) - releaseValue(a.releasedAt) || b.offersCount - a.offersCount || byName(a, b),
      offers: (a, b) => b.offersCount - a.offersCount || byName(a, b),
      context: (a, b) => b.contextWindow - a.contextWindow || byName(a, b),
      price: (a, b) =>
        (a.minInputCny < 0 ? Number.MAX_SAFE_INTEGER : a.minInputCny)
        - (b.minInputCny < 0 ? Number.MAX_SAFE_INTEGER : b.minInputCny) || byName(a, b),
    };
    return [...list].sort(cmp[sort]);
  }, [vendorScoped, q, cat, sort]);

  const stats = useMemo(() => {
    if (!items) return null;
    return { total: items.length, vendors: vendors.length, priced: priced.length };
  }, [items, vendors, priced]);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />模型目录</span>
          <h1>模型与报价</h1>
          <p className="sub">
            {stats
              ? `收录 ${stats.total} 个模型、${stats.vendors} 家厂商，其中 ${stats.priced} 个已有站点报价。`
              : '按厂商、上下文与发布时间浏览模型。'}
          </p>
        </div>
        <div className="model-links mono">
          <Link to="/models/license">按许可证浏览 →</Link>
          <Link to="/compare/models">模型对比 →</Link>
        </div>
      </div>

      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search search-wide">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <circle cx="7" cy="7" r="4.6" stroke="currentColor" strokeWidth="1.3" />
              <path d="m10.6 10.6 3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
            </svg>
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索模型、厂商或模型 ID" aria-label="搜索模型" />
          </div>
        </div>
        <div className="toolbar-right">
          <select value={vendor} onChange={(e) => setVendor(e.target.value)} aria-label="按厂商筛选">
            <option value="all">全部厂商（{items?.length ?? 0}）</option>
            {vendors.map((v) => (
              <option key={v.id} value={v.id}>{v.label}（{v.count}）</option>
            ))}
          </select>
          <select value={sort} onChange={(e) => setSort(e.target.value as SortKey)} aria-label="排序方式">
            {SORTS.map(([key, label]) => (
              <option key={key} value={key}>{label}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="cat-tabs">
        {CATEGORIES.map(([key, label]) => (
          <button key={key} className={`cat-tab ${cat === key ? 'on' : ''}`} onClick={() => setCat(key)}>
            {label}
            <span>{key === 'all' ? vendorScoped.length : vendorScoped.filter((m) => (m.category || 'text') === key).length}</span>
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
      <p className="fine model-scope-note">
        {showAll
          ? '显示的是完整模型目录，包含没有站点报价的模型。'
          : '报价来自站点公开倍率；batch 与免费额度不计入，没有站点报价的模型仍可在全部目录中找到。'}
      </p>
      <p className="fine model-scope-note">
        单位：最低价为人民币（文本每百万 Token，图片按张，视频按秒），官方价为美元每百万 Token。
      </p>

      {error && <div className="error-box">{error}</div>}

      <section className="panel">
        <div className="table-wrap models-table">
          <table className="index-table">
            <thead>
              <tr>
                <th>模型</th>
                <th className="hide-sm">厂商</th>
                <th className="num">上下文</th>
                <th className="num hide-md">发布</th>
                <th className="num hide-md">官方价</th>
                <th className="num">最低价</th>
                <th className="num">报价站点</th>
              </tr>
            </thead>
            <tbody>
              {!items && !error && (
                Array.from({ length: 6 }).map((_, i) => (
                  <tr key={`sk-${i}`} className="sk-row">
                    <td><span className="sk" style={{ width: 160 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                    <td><span className="sk" style={{ width: 60 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                    <td><span className="sk" style={{ width: 70 }} /></td>
                  </tr>
                ))
              )}
              {items && rows.length === 0 && (
                <tr><td colSpan={7}><p className="fine table-empty">没有匹配的模型，换个关键词或筛选条件试试。</p></td></tr>
              )}
              {rows.map((m) => (
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
                  <td className="hide-sm dim">{vendorName(m)}</td>
                  <td className="num mono dim">{m.category === 'image' || m.category === 'video' ? '—' : ctxLabel(m.contextWindow)}</td>
                  <td className="num mono dim hide-md">{releaseLabel(m.releasedAt)}</td>
                  <td className="num mono dim hide-md">
                    {m.category === 'image' || m.category === 'video' ? '—' : m.inputUsd > 0 ? `${fmtUsd(m.inputUsd)} / ${fmtUsd(m.outputUsd)}` : '—'}
                  </td>
                  <td className="num mono accent">{priceLabel(m)}</td>
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
          {items && rows.length === 0 && <p className="fine" style={{ padding: 16 }}>没有匹配的模型，换个关键词或筛选条件试试。</p>}
          {rows.map((m) => (
            <Link key={m.id} className="model-card" to={`/models/${encodeURIComponent(m.id)}`}>
              <VendorMark vendor={m.vendor} size={30} />
              <div className="mc-id">
                <b>{m.name}</b>
                <small>{vendorName(m)} · {CATEGORY_LABEL[m.category || 'text']}</small>
                <small className="mc-tags">
                  {m.category === 'image' || m.category === 'video' ? '' : `${ctxLabel(m.contextWindow)} 上下文`}
                  {m.category === 'image' || m.category === 'video' ? '' : ' · '}
                  {releaseLabel(m.releasedAt)} 发布
                </small>
              </div>
              <div className="mc-price">
                <span>{priceLabel(m)}</span>
                <small>{m.offersCount} 站报价</small>
              </div>
            </Link>
          ))}
        </div>
      </section>
    </main>
  );
}

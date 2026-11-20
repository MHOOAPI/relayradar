import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary } from '../types';
import { fmtCny } from '../components/ui';
import { VendorMark } from '../components/marks';

const fmtCtx = (n: number) => (n >= 1000000 ? `${(n / 1000000).toFixed(n % 1000000 === 0 ? 0 : 2)}M` : `${Math.round(n / 1000)}K`);

const CATEGORIES = [
  ['all', '全部'],
  ['text', '文本'],
  ['image', '图片'],
  ['video', '视频'],
] as const;

const CATEGORY_LABEL: Record<string, string> = { text: '文本', image: '图片', video: '视频' };

export function ModelsPage() {
  const [items, setItems] = useState<ModelSummary[] | null>(null);
  const [params, setParams] = useSearchParams();
  const [q, setQ] = useState(params.get('q') ?? '');
  const [cat, setCat] = useState(params.get('cat') ?? 'all');
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

  const rows = useMemo(() => {
    if (!items) return [];
    const n = q.trim().toLowerCase();
    return items.filter((m) => (cat === 'all' || (m.category || 'text') === cat) && (!n || `${m.name} ${m.id} ${m.vendor}`.toLowerCase().includes(n)));
  }, [items, q, cat]);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />MODELS</span>
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

      <div className="cat-tabs">
        {CATEGORIES.map(([key, label]) => (
          <button key={key} className={`cat-tab ${cat === key ? 'on' : ''}`} onClick={() => setCat(key)}>
            {label}
            <span>{key === 'all' ? items?.length ?? '' : items?.filter((m) => (m.category || 'text') === key).length ?? ''}</span>
          </button>
        ))}
      </div>

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
                  <td className="hide-sm dim">{m.vendorName}</td>
                  <td className="num mono dim hide-sm">{m.category === 'text' || !m.category ? fmtCtx(m.contextWindow) : '—'}</td>
                  <td className="num mono dim">{m.category === 'text' || !m.category ? `$${m.inputUsd} / $${m.outputUsd}` : '按张/秒计费'}</td>
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
          {rows.map((m) => (
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
    </main>
  );
}

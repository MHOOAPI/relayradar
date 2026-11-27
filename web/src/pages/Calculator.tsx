import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary, Offer } from '../types';
import { fmtCny, Spinner, TierBadge } from '../components/ui';
import { StationMark } from '../components/marks';
import { copyText } from '../lib/clipboard';
import { useToast } from '../lib/toast';
import '../styles/calculator.css';

export function CalculatorPage() {
  const [params, setParams] = useSearchParams();
  const { push } = useToast();
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [modelId, setModelId] = useState(params.get('model') ?? '');
  const [inM, setInM] = useState(() => Math.max(0, Number(params.get('in')) || 10));
  const [outM, setOutM] = useState(() => Math.max(0, Number(params.get('out')) || 2));
  const [cachePct, setCachePct] = useState(() => Math.min(90, Math.max(0, Number(params.get('cache')) || 0)));
  const [offers, setOffers] = useState<Offer[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [limit, setLimit] = useState(30);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api.models().then((r) => {
      setModels(r.items);
      if (!params.get('model')) {
        const first = r.items.find((m) => m.offersCount > 0) ?? r.items[0];
        if (first) setModelId(first.id);
      }
    }).catch(() => setModels([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!modelId) return;
    setLoading(true);
    setOffers(null);
    api.model(modelId)
      .then((r) => setOffers(r.offers))
      .catch(() => setOffers([]))
      .finally(() => setLoading(false));
  }, [modelId]);

  useEffect(() => {
    const next = new URLSearchParams();
    if (modelId) next.set('model', modelId);
    next.set('in', String(inM));
    next.set('out', String(outM));
    if (cachePct > 0) next.set('cache', String(cachePct));
    setParams(next, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [modelId, inM, outM, cachePct]);

  const model = useMemo(() => models.find((m) => m.id === modelId), [models, modelId]);

  const rows = useMemo(() => {
    if (!offers) return [];
    const cache = Math.min(90, Math.max(0, cachePct)) / 100;
    const best = new Map<string, Offer & { monthly: number }>();
    for (const o of offers) {
      const cacheRead = o.cacheReadCny > 0 ? o.cacheReadCny : o.inputCny;
      const monthly = o.inputCny * inM * (1 - cache) + cacheRead * inM * cache + o.outputCny * outM;
      const prev = best.get(o.stationSlug);
      if (!prev || monthly < prev.monthly) best.set(o.stationSlug, { ...o, monthly });
    }
    return [...best.values()].sort((a, b) => a.monthly - b.monthly);
  }, [offers, inM, outM, cachePct]);

  const shown = rows.slice(0, limit);
  const cheapest = rows[0];

  const copyLink = async () => {
    const ok = await copyText(window.location.href);
    setCopied(ok);
    push(ok ? '已复制估算链接' : '复制失败，请手动选择地址栏', ok ? 'ok' : 'err');
    if (ok) window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <main className="wrap calc-page">
      <div className="page-head">
        <div>
          <h1>成本估算器</h1>
          <p className="sub">按你每月的输入 / 输出 token 量，把同一模型在各中转站的报价折算成月成本；只比较有该模型报价的站点。</p>
        </div>
        <button type="button" className="link-btn" onClick={copyLink}>{copied ? '已复制 ✓' : '复制估算链接'}</button>
      </div>

      <section className="panel calc-form">
        <div className="calc-fields">
          <label className="calc-field calc-model">
            <span>模型</span>
            <select value={modelId} onChange={(e) => setModelId(e.target.value)} aria-label="选择模型">
              {models.map((m) => (
                <option key={m.id} value={m.id}>{m.name} · {m.vendorName}</option>
              ))}
            </select>
          </label>
          <label className="calc-field">
            <span>月输入（百万 token）</span>
            <input type="number" min={0} step={1} value={inM} onChange={(e) => setInM(Math.max(0, Number(e.target.value)))} />
          </label>
          <label className="calc-field">
            <span>月输出（百万 token）</span>
            <input type="number" min={0} step={1} value={outM} onChange={(e) => setOutM(Math.max(0, Number(e.target.value)))} />
          </label>
          <label className="calc-field">
            <span>缓存命中 {cachePct}%</span>
            <input type="range" min={0} max={90} step={5} value={cachePct} onChange={(e) => setCachePct(Number(e.target.value))} />
          </label>
        </div>
        {model && (
          <p className="calc-hint mono dim">
            官方价 ${model.inputUsd.toFixed(2)} 输入 / ${model.outputUsd.toFixed(2)} 输出 每百万 token · {model.offersCount} 站报价
          </p>
        )}
      </section>

      {cheapest && (
        <section className="calc-summary">
          <div>
            <span className="mono dim">当前最低月成本</span>
            <b className="calc-big">{fmtCny(cheapest.monthly)}</b>
            <span className="dim">/月</span>
          </div>
          <div className="calc-summary-station">
            <StationMark slug={cheapest.stationSlug} name={cheapest.stationName} iconUrl={cheapest.iconUrl} size={28} />
            <div>
              <Link to={`/stations/${cheapest.stationSlug}`}><b>{cheapest.stationName}</b></Link>
              <small className="mono dim">{cheapest.groupName} · {cheapest.multiplier.toFixed(2)}×</small>
            </div>
          </div>
        </section>
      )}

      <section className="panel calc-table-wrap">
        <div className="panel-head">
          <div>
            <h2>按模型报价折算</h2>
            <p className="sub">成本 = 输入价 × 输入量 × (1−缓存命中) + 缓存价 × 输入量 × 命中 + 输出价 × 输出量；取每站最便宜的可用分组。</p>
          </div>
          <span className="mono dim">{rows.length} 家可比</span>
        </div>
        {loading && <div className="loading"><Spinner /> 正在折算…</div>}
        {!loading && rows.length === 0 && <p className="fine calc-empty">该模型暂时没有可比报价，换一个模型试试。</p>}
        {!loading && rows.length > 0 && (
          <>
            <div className="calc-table-scroll">
              <table className="calc-table">
                <thead>
                  <tr>
                    <th>#</th>
                    <th>站点</th>
                    <th>分组</th>
                    <th className="num">倍率</th>
                    <th className="num">输入 ¥/M</th>
                    <th className="num">输出 ¥/M</th>
                    <th className="num">月成本</th>
                    <th>纯度</th>
                  </tr>
                </thead>
                <tbody>
                  {shown.map((o, i) => (
                    <tr key={o.stationSlug}>
                      <td className="mono dim">{i + 1}</td>
                      <td>
                        <Link className="calc-station" to={`/stations/${o.stationSlug}`}>
                          <StationMark slug={o.stationSlug} name={o.stationName} iconUrl={o.iconUrl} size={22} />
                          <span>{o.stationName}</span>
                        </Link>
                      </td>
                      <td className="mono dim">{o.groupName}</td>
                      <td className="num mono">{o.multiplier.toFixed(2)}×</td>
                      <td className="num mono">{fmtCny(o.inputCny)}</td>
                      <td className="num mono">{fmtCny(o.outputCny)}</td>
                      <td className="num mono calc-cost">{fmtCny(o.monthly)}</td>
                      <td><TierBadge tier={o.purityTier} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {rows.length > limit && (
              <div className="calc-more">
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => setLimit((v) => v + 30)}>再看 30 家</button>
              </div>
            )}
          </>
        )}
      </section>

      <p className="fine calc-note">
        估算基于当前快照报价，不含充值门槛、赠送额度与限时折扣；倍率以分组为准，最终价格以站点账单为准。
        {model && <> 返回 <Link to={`/models/${encodeURIComponent(model.id)}`}>模型档案</Link> 查看全部报价。</>}
      </p>
    </main>
  );
}

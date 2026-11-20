import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelProviderStatus, ModelSummary, Offer, PriceChange } from '../types';
import { fmtCny, fmtDate, Spinner, TierBadge } from '../components/ui';
import { StationMark, VendorMark } from '../components/marks';

const fmtCtx = (n: number) => (n >= 1000000 ? `${(n / 1000000).toFixed(n % 1000000 === 0 ? 0 : 2)}M` : `${Math.round(n / 1000)}K`);

export function ModelPage() {
  const { id } = useParams();
  const [model, setModel] = useState<ModelSummary | null>(null);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [priceChanges, setPriceChanges] = useState<PriceChange[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [upstream, setUpstream] = useState<ModelProviderStatus | null>(null);

  useEffect(() => {
    if (!model) return;
    const map: Record<string, string> = {
      anthropic: 'anthropic', openai: 'openai', google: 'google', deepseek: 'deepseek',
      moonshot: 'moonshot', kimi: 'moonshot', qwen: 'alibaba', xai: 'xai',
    };
    let alive = true;
    api.modelStatus()
      .then((r) => {
        if (alive) setUpstream(r.providers.find((p) => p.key === map[model.vendor]) ?? null);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [model?.vendor]);

  useEffect(() => {
    if (!id) return;
    setModel(null);
    api.model(id)
      .then((r) => {
        setModel(r.model);
        setOffers(r.offers ?? []);
        setPriceChanges(r.priceChanges ?? []);
      })
      .catch((e: Error) => setError(e.message));
  }, [id]);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!model) return <main className="wrap"><div className="loading-page"><Spinner /> 读取模型…</div></main>;

  const best = offers.find((o) => o.purityTier !== 'counterfeit' && o.inputCny > 0);
  const perUnit = model.category === 'image' ? '/张' : model.category === 'video' ? '/秒' : '';

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/models">模型</Link><span>/</span><span className="dim">{model.id}</span>
      </div>

      {upstream && upstream.status !== 'operational' && (
        <div className={`upstream-banner ${upstream.status}`}>
          <b>{upstream.name} 上游状态：{upstream.status === 'degraded' ? '降级' : upstream.status === 'outage' ? '不可用' : '未知'}</b>
          <span>{upstream.description || '官方状态页暂未提供说明'}</span>
          <Link to={`/model-status/${upstream.key}`}>查看官方状态历史 →</Link>
        </div>
      )}

      <section className="panel detail-head">
        <div className="detail-main">
          <VendorMark vendor={model.vendor} size={54} />
          <div>
            <div className="detail-title">
              <h1>{model.name}</h1>
              <span className="svc">{model.vendorName}</span>
            </div>
            <p className="tagline">{model.summary}</p>
            <div className="detail-links mono dim">
              <span>{model.id}</span>
              <span>·</span>
              <span>上下文 {fmtCtx(model.contextWindow)}</span>
              <span>·</span>
              <Link to={`/models/license/${model.license}`}>许可证 {model.license}</Link>
              <span>·</span>
              <Link to={`/compare/models?a=${model.id}`}>对比模型</Link>
            </div>
          </div>
        </div>
        <div className="model-scores">
          {model.category === 'text' || !model.category ? (
            <>
              <div><span>官方输入</span><b className="mono">${model.inputUsd}</b></div>
              <div><span>官方输出</span><b className="mono">${model.outputUsd}</b></div>
              <div><span>编程分</span><b className="mono">{model.codingScore}</b></div>
              <div><span>推理分</span><b className="mono">{model.reasoningScore}</b></div>
            </>
          ) : (
            <>
              <div><span>类型</span><b>{model.category === 'image' ? '图片生成' : '视频生成'}</b></div>
              <div><span>计费</span><b className="mono">按{model.category === 'image' ? '张' : '秒'}计费</b></div>
            </>
          )}
        </div>
      </section>

      <section className="panel model-offers">
        <div className="panel-head">
          <div>
            <h2>全网报价（按输入价升序）</h2>
            <p className="sub">同一模型可能在不同分组的倍率下差价数倍，注意同时看纯度与可用性。</p>
          </div>
          <span className="mono dim">
            {offers.length} 条报价
            {best && ` · 最低 ${fmtCny(best.inputCny)} · ${best.stationName}`}
          </span>
        </div>
        <div className="table-wrap offer-table">
          <table className="index-table">
            <thead>
              <tr>
                <th className="num">#</th><th>站点</th><th>分组</th><th className="num">倍率</th>
                <th className="num">{perUnit ? `价格 ${perUnit}` : '输入 /1M'}</th><th className="num">{perUnit ? '同价' : '输出 /1M'}</th><th>纯度</th><th className="num">可用性指数</th>
              </tr>
            </thead>
            <tbody>
              {offers.length === 0 && (
                <tr><td colSpan={8} className="loading">暂无中转站报价；以上为官方价参考。</td></tr>
              )}
              {offers.map((o, i) => (
                <tr key={`${o.stationSlug}-${o.groupSlug}`} className={o === best ? 'offer-best' : undefined}>
                  <td className="num rank dim">{i + 1}</td>
                  <td>
                    <div className="cell-station">
                      <StationMark slug={o.stationSlug} name={o.stationName} iconUrl={o.iconUrl} size={26} />
                      <Link className="station-name" to={`/stations/${o.stationSlug}`}>{o.stationName}</Link>
                      {o.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
                      {o === best && <span className="best-chip">最低价</span>}
                    </div>
                  </td>
                  <td><span className="mono dim">{o.groupName}</span></td>
                  <td className="num mono">{o.multiplier.toFixed(2)}×</td>
                  <td className="num mono">{fmtCny(o.inputCny)}{perUnit}</td>
                  <td className="num mono accent">{fmtCny(o.outputCny)}{perUnit}</td>
                  <td><TierBadge tier={o.purityTier} /></td>
                  <td className="num mono dim">{o.availability.toFixed(1)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="offer-cards">
          {offers.map((o, i) => (
            <Link key={`${o.stationSlug}-${o.groupSlug}`} to={`/stations/${o.stationSlug}`} className={`offer-card${o === best ? ' best' : ''}`}>
              <div className="oc-head">
                <span className="mono dim">{i + 1}</span>
                <StationMark slug={o.stationSlug} name={o.stationName} iconUrl={o.iconUrl} size={28} />
                <div className="oc-id">
                  <b>{o.stationName}</b>
                  <small className="mono">{o.groupName}</small>
                </div>
                {o.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
              </div>
              {o === best && <span className="best-chip">最低价</span>}
              <div className="oc-prices">
                <span>输入 <b className="mono">{fmtCny(o.inputCny)}</b></span>
                <span>输出 <b className="mono">{fmtCny(o.outputCny)}</b></span>
                <span>倍率 <b className="mono">{o.multiplier.toFixed(2)}×</b></span>
              </div>
              <div className="oc-foot">
                <TierBadge tier={o.purityTier} />
                <span className="mono dim">可用性 {o.availability.toFixed(1)}</span>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {priceChanges.length > 0 && (
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>价格变动记录</h2>
              <p className="sub">来自历史价格快照，只记录实际观测到的调整；未列出的站点代表价格未变。</p>
            </div>
            <span className="mono dim">{priceChanges.length} 次</span>
          </div>
          <div className="table-wrap">
            <table className="index-table">
              <thead>
                <tr>
                  <th>站点</th><th>分组</th><th className="num">输入价</th>
                  <th className="num">变动</th><th>记录时间</th>
                </tr>
              </thead>
              <tbody>
                {priceChanges.map((c) => (
                  <tr key={`${c.stationSlug}-${c.groupName}-${c.at}`}>
                    <td><Link className="station-name" to={`/stations/${c.stationSlug}`}>{c.stationName}</Link></td>
                    <td><span className="mono dim">{c.groupName}</span></td>
                    <td className="num mono">{fmtCny(c.oldInputCny)} → {fmtCny(c.newInputCny)}</td>
                    <td className={`num mono ${c.deltaPct > 0 ? 'delta-up' : 'delta-down'}`}>
                      {c.deltaPct > 0 ? '+' : ''}{c.deltaPct.toFixed(1)}%
                    </td>
                    <td><span className="mono dim">{fmtDate(c.at)}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </main>
  );
}

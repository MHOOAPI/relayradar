import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { Station } from '../types';
import { Spinner, UptimeBars } from '../components/ui';
import { StationMark, VendorMark } from '../components/marks';
import { PayMark } from '../components/pay';
import { SERVICE_LABEL, SERVICE_VENDOR } from '../lib/services';


const healthOf = (s: Station) => {
  if (s.uptimePct >= 97) return { cls: 'ok', label: '正常' };
  if (s.uptimePct >= 88) return { cls: 'warn', label: '波动' };
  return { cls: 'bad', label: '异常' };
};

const reachInfo = (r: Station['reachability']) => {
  if (!r) return null;
  if (r.status === 'ok') return { cls: 'ok', label: `可达 ${r.latencyMs}ms` };
  if (r.status === 'blocked') return { cls: 'warn', label: '拒绝探测' };
  if (r.status === 'unknown') return { cls: 'unk', label: '未知' };
  return { cls: 'down', label: '不可达' };
};

export function StationsPage() {
  const [params] = useSearchParams();
  const [stations, setStations] = useState<Station[] | null>(null);
  const [q, setQ] = useState(params.get('q') ?? '');
  const [sort, setSort] = useState<'rank' | 'uptime' | 'reviews' | 'reach'>('rank');
  const [onlyClaimed, setOnlyClaimed] = useState(false);
  const [minModels, setMinModels] = useState(0);
  const [onlyReachable, setOnlyReachable] = useState(false);
  const [totalModels, setTotalModels] = useState(0);
  const [modelList, setModelList] = useState<Array<{ id: string; name: string }>>([]);
  const [compareOpen, setCompareOpen] = useState(false);
  const [coverageStation, setCoverageStation] = useState<Station | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    document.title = '中转站目录：模型覆盖、支付、退款与评价 | RelayRadar';
    api.stations(1000).then((r) => setStations(r.items)).catch((e: Error) => setError(e.message));
    api.meta().then((m) => setTotalModels(m.models)).catch(() => setTotalModels(0));
    api.models().then((r) => setModelList(r.items.map((m) => ({ id: m.id, name: m.name })))).catch(() => setModelList([]));
  }, []);

  useEffect(() => {
    setQ(params.get('q') ?? '');
    if (params.get('reachable') === '1') setOnlyReachable(true);
  }, [params]);

  const rows = useMemo(() => {
    const needle = q.trim().toLowerCase();
    const list = (stations ?? []).filter(
      (s) => !s.officialVendor && (!onlyClaimed || s.claimed) && (!minModels || s.modelsCount >= minModels) && (!onlyReachable || s.reachability?.status === 'ok') && (!needle || `${s.name} ${s.domain} ${s.slug}`.toLowerCase().includes(needle)),
    );
    return [...list].sort((a, b) => {
      if (sort === 'uptime') return b.uptimePct - a.uptimePct;
      if (sort === 'reviews') return b.reviewsCount - a.reviewsCount || b.ratingAvg - a.ratingAvg;
      if (sort === 'reach') {
        const rank = (s: Station) => (s.reachability?.status === 'ok' ? 0 : s.reachability?.status === 'blocked' ? 1 : s.reachability?.status === 'unknown' ? 2 : 3);
        return rank(a) - rank(b) || (a.reachability?.latencyMs ?? 99999) - (b.reachability?.latencyMs ?? 99999);
      }
      return a.rank - b.rank;
    });
  }, [stations, q, sort, onlyClaimed, minModels, onlyReachable]);

  const [visible, setVisible] = useState(60);
  useEffect(() => setVisible(60), [q, sort, onlyClaimed, minModels, onlyReachable]);
  const shown = rows.slice(0, visible);

  const missingOf = (s: Station) =>
    modelList.filter((m) => !(s.modelIds ?? []).includes(m.id)).map((m) => m.name);

  const coveredOf = (s: Station) =>
    modelList.filter((m) => (s.modelIds ?? []).includes(m.id)).map((m) => m.name);

  const claimStats = useMemo(() => {
    // Stations without samples (e.g. a real-price-only station) would drag the
    // averages to 0%; they are excluded from the claimed/unclaimed comparison.
    const list = (stations ?? []).filter((s) => !s.officialVendor && s.samples > 0);
    const agg = (claimed: boolean) => {
      const group = list.filter((s) => s.claimed === claimed);
      if (group.length === 0) return null;
      const avg = (f: (s: (typeof group)[number]) => number) =>
        group.reduce((sum, s) => sum + (Number.isFinite(f(s)) ? f(s) : 0), 0) / group.length;
      const rated = group.filter((s) => s.ratingAvg > 0);
      const rating = rated.length ? rated.reduce((sum, s) => sum + s.ratingAvg, 0) / rated.length : 0;
      const withRefund = group.filter((s) => s.refundPolicy.trim() && !/不退|no refund/i.test(s.refundPolicy));
      return {
        count: group.length,
        uptime: avg((s) => s.uptimePct),
        rating,
        models: avg((s) => s.modelsCount),
        coverage: totalModels > 0 ? (avg((s) => s.modelsCount) / totalModels) * 100 : 0,
        invoicePct: (group.filter((s) => s.invoice).length / group.length) * 100,
        refundPct: (withRefund.length / group.length) * 100,
      };
    };
    return { claimed: agg(true), unclaimed: agg(false) };
  }, [stations, totalModels]);

  useEffect(() => {
    const slug = params.get('coverage');
    if (!slug || !stations) return;
    const found = stations.find((s) => s.slug === slug);
    if (found) setCoverageStation(found);
  }, [params, stations]);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>中转站目录</h1>
          <p className="sub">只收录第三方中转 / 聚合站；阿里云百炼、硅基流动等官方厂商不在此列。按模型覆盖、健康状态、付款方式、退款、发票与用户评价浏览。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/">看实时榜单 →</Link>
      </div>

      {claimStats.claimed && claimStats.unclaimed && claimStats.claimed.count >= 2 && claimStats.unclaimed.count >= 3 && (
        <section className={`claim-compare ${compareOpen ? 'open' : ''}`}>
          <div className="claim-compare-col">
            <span className="mono">✓ 已认领 · {claimStats.claimed.count} 家</span>
            <b>{claimStats.claimed.uptime.toFixed(1)}%<small> 平均可用率</small></b>
            <div className="claim-compare-rows">
              <span>平均覆盖 <b className="mono">{claimStats.claimed.coverage.toFixed(0)}%</b></span>
              <span>支持发票 <b className="mono">{claimStats.claimed.invoicePct.toFixed(0)}%</b></span>
              <span>有退款说明 <b className="mono">{claimStats.claimed.refundPct.toFixed(0)}%</b></span>
              <span>平均评分 <b className="mono">{claimStats.claimed.rating ? claimStats.claimed.rating.toFixed(1) : '—'}</b></span>
            </div>
          </div>
          <div className="claim-compare-col">
            <span className="mono">未认领 · {claimStats.unclaimed.count} 家</span>
            <b>{claimStats.unclaimed.uptime.toFixed(1)}%<small> 平均可用率</small></b>
            <div className="claim-compare-rows">
              <span>平均覆盖 <b className="mono">{claimStats.unclaimed.coverage.toFixed(0)}%</b></span>
              <span>支持发票 <b className="mono">{claimStats.unclaimed.invoicePct.toFixed(0)}%</b></span>
              <span>有退款说明 <b className="mono">{claimStats.unclaimed.refundPct.toFixed(0)}%</b></span>
              <span>平均评分 <b className="mono">{claimStats.unclaimed.rating ? claimStats.unclaimed.rating.toFixed(1) : '—'}</b></span>
            </div>
          </div>
          <p className="fine claim-compare-note">认领只证明站长身份，不改变探针与评分；样本较小时对比仅供参考。</p>
          <button className="compare-toggle" onClick={() => setCompareOpen((v) => !v)}>
            {compareOpen ? '收起明细' : '展开明细'}
          </button>
        </section>
      )}

      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search">
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索站点 / 域名" aria-label="搜索站点" />
          </div>
        </div>
        <div className="toolbar-right">
          <label className="check-inline">
            <input type="checkbox" checked={onlyClaimed} onChange={(e) => setOnlyClaimed(e.target.checked)} />
            只看已认领
          </label>
          <label className="check-inline">
            <input type="checkbox" checked={onlyReachable} onChange={(e) => setOnlyReachable(e.target.checked)} />
            只看可达
          </label>
          <select value={minModels} onChange={(e) => setMinModels(Number(e.target.value))} aria-label="模型覆盖筛选">
            <option value={0}>全部覆盖</option>
            <option value={8}>覆盖 ≥ 8</option>
            <option value={10}>覆盖 ≥ 10</option>
            <option value={12}>覆盖 ≥ 12</option>
          </select>
          <select value={sort} onChange={(e) => setSort(e.target.value as typeof sort)} aria-label="排序方式">
            <option value="rank">按排名</option>
            <option value="uptime">按可用率</option>
            <option value="reviews">按评价数</option>
            <option value="reach">按可达延迟</option>
          </select>
        </div>
      </div>

      {error && <div className="error-box">{error}</div>}

      <section className="panel">
        <div className="table-wrap dir-table">
          <table className="index-table">
            <thead>
              <tr>
                <th>站点</th>
                <th>模型覆盖</th>
                <th>健康状态</th>
                <th>付款方式</th>
                <th>退款</th>
                <th>发票</th>
                <th>用户评价</th>
              </tr>
            </thead>
            <tbody>
              {!stations && !error && (
                <tr><td colSpan={7} className="loading"><Spinner /> 加载站点目录…</td></tr>
              )}
              {shown.map((s) => {
                const h = healthOf(s);
                return (
                  <tr key={s.slug} className="clickable" onClick={() => navigate(`/stations/${s.slug}`)}>
                    <td>
                      <div className="cell-station">
                        <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                        <div className="station-meta">
                          <Link className="station-name" to={`/stations/${s.slug}`} onClick={(e) => e.stopPropagation()}>{s.name}</Link>
                          {s.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
                          <span className="station-domain mono">{s.domain}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div className="svc-icons">
                        {s.services.slice(0, 5).map((sv) => {
                          const vendor = SERVICE_VENDOR[sv];
                          return vendor ? (
                            <span key={sv} className="svc-icon" title={SERVICE_LABEL[sv] ?? sv}>
                              <VendorMark vendor={vendor} size={18} />
                            </span>
                          ) : (
                            <span key={sv} className="svc">{SERVICE_LABEL[sv] ?? sv}</span>
                          );
                        })}
                        {s.services.length > 5 && <span className="dim tiny">+{s.services.length - 5}</span>}
                      </div>
                      <button
                        type="button"
                        className="coverage-btn mono dim tiny"
                        onClick={(e) => { e.stopPropagation(); setCoverageStation(s); }}
                        title="查看覆盖详情"
                      >
                        {s.modelsCount}{totalModels > 0 ? `/${totalModels}` : ''} 个模型
                        {missingOf(s).length > 0 && <span className="dim"> · 缺{missingOf(s).length}</span>}
                      </button>
                    </td>
                    <td>
                      <div className="health">
                        <i className={`health-dot ${h.cls}`} />
                        <span className="mono">{s.uptimePct.toFixed(1)}%</span>
                        <span className="dim tiny">{h.label}</span>
                        {s.reachability && (() => {
                          const ri = reachInfo(s.reachability)!;
                          return <span className={`reach-tag ${ri.cls}`} title="无凭据首页连通性探测">{ri.label}</span>;
                        })()}
                      </div>
                      <UptimeBars data={s.uptime} width={3} />
                    </td>
                    <td>
                      <span className="pay-list">
                        {s.payments.slice(0, 3).map((p) => (
                          <span key={p} className="pay-item"><PayMark method={p} size={14} />{p}</span>
                        ))}
                        {s.payments.length === 0 && <span className="dim">—</span>}
                      </span>
                    </td>
                    <td className="dir-refund" title={s.refundPolicy}>{s.refundPolicy || '—'}</td>
                    <td>{s.invoice ? <span className="pill-ok">支持</span> : <span className="dim">不支持</span>}</td>
                    <td>
                      {s.reviewsCount > 0 ? (
                        <span className="rating"><b className="mono">{s.ratingAvg.toFixed(1)}</b><span className="dim"> / 5 · {s.reviewsCount} 条</span></span>
                      ) : (
                        <span className="dim">暂无评价</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        <div className="dir-cards">
          {!stations && <p className="fine" style={{ padding: 16 }}>加载中…</p>}
          {shown.map((s) => {
            const h = healthOf(s);
            return (
              <Link key={s.slug} className="dir-card" to={`/stations/${s.slug}`}>
                <div className="dir-card-head">
                  <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                  <div className="dir-id"><b>{s.name}{s.claimed && <span className="badge-ok" title="站长已认领">✓</span>}</b><small className="mono">{s.domain}</small></div>
                  <span className={`health-dot ${h.cls}`} title={h.label} />
                </div>
                <div className="dir-line"><span className="mono dim">覆盖</span>{s.services.map((sv) => {
                  const vendor = SERVICE_VENDOR[sv];
                  return vendor ? (
                    <span key={sv} className="svc-icon" title={SERVICE_LABEL[sv] ?? sv}><VendorMark vendor={vendor} size={16} /></span>
                  ) : (
                    <span key={sv} className="svc">{SERVICE_LABEL[sv] ?? sv}</span>
                  );
                })}<button
                  type="button"
                  className="coverage-btn dim tiny"
                  onClick={(e) => { e.preventDefault(); e.stopPropagation(); setCoverageStation(s); }}
                >{s.modelsCount}{totalModels > 0 ? `/${totalModels}` : ''} 模型{missingOf(s).length > 0 ? ` · 缺${missingOf(s).length}` : ''}</button></div>
                <div className="dir-line"><span className="mono dim">健康</span><b className="mono">{s.uptimePct.toFixed(1)}%</b><UptimeBars data={s.uptime} width={3} />
                  {s.reachability && (() => {
                    const ri = reachInfo(s.reachability)!;
                    return <span className={`reach-tag ${ri.cls}`} title="无凭据首页连通性探测">{ri.label}</span>;
                  })()}
                </div>
                <div className="dir-line"><span className="mono dim">支付</span>{s.payments.slice(0, 3).map((p) => <span key={p} className="pay-item"><PayMark method={p} size={14} />{p}</span>)}</div>
                <div className="dir-line"><span className="mono dim">退款</span><span className="dir-refund">{s.refundPolicy || '—'}</span><span className="dim">· {s.invoice ? '可开票' : '不开发票'}</span></div>
                <div className="dir-line"><span className="mono dim">评价</span>{s.reviewsCount > 0 ? <span className="rating"><b className="mono">{s.ratingAvg.toFixed(1)}</b><span className="dim"> / 5 · {s.reviewsCount} 条</span></span> : <span className="dim">暂无</span>}</div>
              </Link>
            );
          })}
        </div>
      </section>

      {rows.length > shown.length && (
        <div className="list-more">
          <button className="btn btn-ghost btn-sm" onClick={() => setVisible((v) => v + 120)}>
            显示更多站点
          </button>
          <span className="mono">已显示 {shown.length} / {rows.length}</span>
        </div>
      )}

      {coverageStation && (
        <div className="modal-backdrop" onClick={() => setCoverageStation(null)} role="presentation">
          <div className="coverage-modal" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
            <div className="coverage-modal-head">
              <div>
                <b>{coverageStation.name}</b>
                <span className="mono dim">{coverageStation.modelsCount}/{totalModels || '—'} 模型</span>
              </div>
              <button className="link-btn" onClick={() => setCoverageStation(null)}>关闭</button>
            </div>
            <div className="coverage-group">
              <span className="mono dim">已覆盖（{coveredOf(coverageStation).length}）</span>
              <div className="coverage-chips">
                {coveredOf(coverageStation).map((m) => <span key={m} className="cov-chip ok">{m}</span>)}
              </div>
            </div>
            <div className="coverage-group">
              <span className="mono dim">未覆盖（{missingOf(coverageStation).length}）</span>
              <div className="coverage-chips">
                {missingOf(coverageStation).length === 0 && <span className="fine">全部覆盖</span>}
                {missingOf(coverageStation).map((m) => <span key={m} className="cov-chip miss">{m}</span>)}
              </div>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}

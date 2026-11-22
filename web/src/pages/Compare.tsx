import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { Station } from '../types';
import { fmtCny, Spinner, TierBadge, UptimeBars } from '../components/ui';
import { StationMark } from '../components/marks';
import { PayMark } from '../components/pay';
import { useWatchlist } from '../lib/watchlist';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

type Row = {
  label: string;
  render: (s: Station) => React.ReactNode;
  score?: (s: Station) => number;
  better?: 'high' | 'low';
};

const rows: Row[] = [
  { label: '综合排名', render: (s) => (s.rank ? `#${s.rank}` : '未排名'), score: (s) => (s.rank ? -s.rank : -999), better: 'high' },
  { label: '排名分', render: (s) => <b className="mono">{s.rankScore.toFixed(1)}</b>, score: (s) => s.rankScore, better: 'high' },
  { label: '原始可用率', render: (s) => <span className="mono">{s.uptimePct.toFixed(1)}%</span>, score: (s) => s.uptimePct, better: 'high' },
  { label: '可用性分', render: (s) => <span className="mono">{s.availability.toFixed(1)}</span>, score: (s) => s.availability, better: 'high' },
  { label: '速度分', render: (s) => <span className="mono">{s.speed.toFixed(1)}</span>, score: (s) => s.speed, better: 'high' },
  { label: '稳定性分', render: (s) => <span className="mono">{s.consistency.toFixed(1)}</span>, score: (s) => s.consistency, better: 'high' },
  { label: '纯度抽检', render: (s) => <TierBadge tier={s.purityTier} score={s.purity} />, score: (s) => (s.purityMeasured ? s.purity : -1), better: 'high' },
  { label: '最低输入价', render: (s) => <span className="mono">{fmtCny(s.fromPriceCny)}</span>, score: (s) => (s.fromPriceCny < 0 ? 1e9 : s.fromPriceCny), better: 'low' },
  { label: '起充金额', render: (s) => <span className="mono">¥{s.minTopupCny}</span>, score: (s) => s.minTopupCny, better: 'low' },
  { label: '模型数', render: (s) => <span className="mono">{s.modelsCount}</span>, score: (s) => s.modelsCount, better: 'high' },
  { label: '探针样本', render: (s) => <span className="mono dim">{s.samples}</span> },
  { label: '发票', render: (s) => (s.invoice ? '支持' : '不支持') },
  { label: '退款政策', render: (s) => s.refundPolicy },
  { label: '支付方式', render: (s) => (
    <span className="pay-list">
      {s.payments.map((p) => (
        <span className="pay-chip" key={p}><PayMark method={p} size={15} />{p}</span>
      ))}
    </span>
  ) },
  { label: '近 14 天', render: (s) => <UptimeBars data={s.uptime} width={8} /> },
];

export function ComparePage() {
  const [params, setParams] = useSearchParams();
  const [stations, setStations] = useState<Station[] | null>(null);
  const [q, setQ] = useState('');
  const { push } = useToast();
  const { ids: watched } = useWatchlist();

  useEffect(() => {
    api.stations(1000).then((r) => setStations(r.items)).catch(() => setStations([]));
  }, []);

  const ids = useMemo(() => (params.get('ids') || '').split(',').filter(Boolean), [params]);
  const selected = useMemo(
    () => (stations ?? []).filter((s) => ids.includes(s.slug)),
    [stations, ids],
  );

  const toggle = (slug: string) => {
    const next = ids.includes(slug) ? ids.filter((x) => x !== slug) : ids.length >= 4 ? ids : [...ids, slug];
    setParams(next.length ? { ids: next.join(',') } : {}, { replace: true });
  };

  const swapFirstTwo = () => {
    if (ids.length < 2) return;
    const next = [ids[1], ids[0], ...ids.slice(2)];
    setParams({ ids: next.join(',') }, { replace: true });
  };

  const clearAll = () => setParams({}, { replace: true });

  const applyCombo = (kind: 'main' | 'official' | 'top3') => {
    if (!stations?.length) return;
    const pool = stations.filter((s) => s.rankScore > 0);
    const affordable = stations
      .filter((s) => s.fromPriceCny > 0 && s.purityTier !== 'counterfeit')
      .sort((a, b) => a.fromPriceCny - b.fromPriceCny);
    let slugs: string[] = [];
    if (kind === 'top3') {
      slugs = [...pool].sort((a, b) => b.rankScore - a.rankScore).slice(0, 3).map((s) => s.slug);
    } else if (kind === 'main') {
      const best = [...pool].sort((a, b) => b.rankScore - a.rankScore)[0];
      const cheap = affordable.find((s) => s.slug !== best?.slug);
      slugs = [best?.slug, cheap?.slug].filter((x): x is string => !!x);
    } else {
      const official = pool
        .filter((s) => s.purityTier === 'official')
        .sort((a, b) => b.rankScore - a.rankScore)[0];
      const cheap = affordable.find((s) => s.slug !== official?.slug && s.purityTier !== 'official')
        ?? affordable.find((s) => s.slug !== official?.slug);
      slugs = [official?.slug, cheap?.slug].filter((x): x is string => !!x);
    }
    const p = new URLSearchParams(params);
    if (slugs.length) p.set('ids', [...new Set(slugs)].join(',')); else p.delete('ids');
    setParams(p, { replace: true });
  };

  const options = useMemo(() => {
    if (!stations) return [];
    const n = q.trim().toLowerCase();
    const filtered = stations.filter((s) => !n || `${s.name} ${s.domain} ${s.slug}`.toLowerCase().includes(n));
    return [...filtered].sort((a, b) => {
      const wa = watched.includes(a.slug) ? 0 : 1;
      const wb = watched.includes(b.slug) ? 0 : 1;
      return wa - wb || a.rank - b.rank;
    });
  }, [stations, q, watched]);

  const [visible, setVisible] = useState(48);
  useEffect(() => setVisible(48), [q]);
  const optionsShown = options.slice(0, visible);

  const bestIndex = (row: Row): number => {
    if (!row.score || selected.length < 2) return -1;
    let idx = -1;
    let val = row.better === 'low' ? Number.POSITIVE_INFINITY : Number.NEGATIVE_INFINITY;
    selected.forEach((s, i) => {
      const v = row.score!(s);
      if (row.better === 'low' ? v < val : v > val) {
        val = v;
        idx = i;
      }
    });
    return idx;
  };

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>横向对比</h1>
          <p className="sub">最多同时对比 4 家。绿色为该行最优项，但“便宜”与“稳”经常不可兼得。</p>
        </div>
        <div className="head-actions">
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => {
              void copyText(window.location.href).then((ok) =>
                ok ? push('对比链接已复制') : push('复制失败，请手动复制地址栏', 'err'),
              );
            }}
            disabled={selected.length < 2}
          >
            复制对比链接
          </button>
        </div>
      </div>

      <section className="panel picker-panel">
        <div className="panel-head">
          <div>
            <h2>选择站点</h2>
            <p className="sub">已选 {selected.length}/4，结果会写进 URL，可直接分享。</p>
          </div>
          <div className="search">
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="筛选站点" aria-label="筛选站点" />
          </div>
        </div>
        <div className="compare-presets">
          <span className="fine">推荐组合</span>
          <button type="button" className="chip" onClick={() => applyCombo('main')}>高可用 + 低价备份</button>
          <button type="button" className="chip" onClick={() => applyCombo('official')}>官方直连 + 平价线路</button>
          <button type="button" className="chip" onClick={() => applyCombo('top3')}>评分前三</button>
        </div>
        {selected.length > 0 && (
          <div className="compare-bar">
            {selected.map((s, i) => (
              <span className="compare-chip" key={s.slug}>
                <b className="mono">{i + 1}</b>
                {s.name}
                <button type="button" aria-label={`移除 ${s.name}`} onClick={() => toggle(s.slug)}>×</button>
              </span>
            ))}
            {selected.length >= 2 && <button type="button" className="link-btn" onClick={swapFirstTwo}>⇄ 交换前两列</button>}
            <button type="button" className="link-btn" onClick={clearAll}>清空</button>
          </div>
        )}
        <div className="picker">
          {!stations && <div className="loading"><Spinner /> 加载中…</div>}
          {optionsShown.map((s) => (
            <button
              key={s.slug}
              className={`pick ${ids.includes(s.slug) ? 'on' : ''}`}
              onClick={() => toggle(s.slug)}
              disabled={!ids.includes(s.slug) && ids.length >= 4}
            >
              <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={24} />
              <span>{s.name}</span>
              {watched.includes(s.slug) && <i className="pick-star" aria-label="已关注">★</i>}
              <i className="pick-check">{ids.includes(s.slug) ? '✓' : '+'}</i>
            </button>
          ))}
        </div>
        {options.length > optionsShown.length && (
          <div className="list-more">
            <button className="btn btn-ghost btn-sm" onClick={() => setVisible((v) => v + 96)}>显示更多站点</button>
            <span className="mono">已显示 {optionsShown.length} / {options.length}</span>
          </div>
        )}
      </section>

      {selected.length < 2 ? (
        <div className="panel empty-panel">
          <b>至少选择 2 家站点</b>
          <p>推荐组合：一家高可用主力 + 一家低价备份，再对比纯度与退款政策。</p>
          {stations && (
            <div className="empty-actions">
              <button className="btn btn-primary btn-sm" onClick={() => applyCombo('main')}>高可用 + 低价备份</button>
              <button className="btn btn-ghost btn-sm" onClick={() => applyCombo('official')}>官方直连 + 平价线路</button>
            </div>
          )}
        </div>
      ) : (
        <section className="panel">
          <p className="fine compare-hint">左右滑动查看全部对比，首列固定。</p>
          <div className="table-wrap compare-wrap">
            <table className="compare-table">
              <thead>
                <tr>
                  <th>对比维度</th>
                  {selected.map((s) => (
                    <th key={s.slug}>
                      <Link to={`/stations/${s.slug}`} className="compare-head">
                        <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                        <span>
                          <b>{s.name}</b>
                          <small className="mono">{s.domain}</small>
                        </span>
                      </Link>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const best = bestIndex(row);
                  return (
                    <tr key={row.label}>
                      <td className="row-label">{row.label}</td>
                      {selected.map((s, i) => (
                        <td key={s.slug} className={i === best ? 'best' : ''}>
                          {row.render(s)}
                        </td>
                      ))}
                    </tr>
                  );
                })}
                <tr>
                  <td className="row-label">操作</td>
                  {selected.map((s) => (
                    <td key={s.slug}>
                      <Link className="btn btn-ghost btn-sm" to={`/stations/${s.slug}`}>查看档案</Link>
                    </td>
                  ))}
                </tr>
              </tbody>
            </table>
          </div>
          <div className="panel-foot mono dim">
            绿色高亮只代表该维度数值更优，不代表综合推荐；分组的倍率差异可能比站点差异更大。
          </div>
        </section>
      )}
    </main>
  );
}

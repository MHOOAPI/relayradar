import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { ModelSummary, Station } from '../types';
import { fmtCny, Spinner, TierBadge, UptimeBars } from '../components/ui';
import { StationMark, VendorMark } from '../components/marks';

type Tab = 'coding' | 'reasoning' | 'value' | 'speed' | 'consistency';

const TABS: Array<[Tab, string, string]> = [
  ['coding', '编程能力', '官方 Coding 跑分，越高越强'],
  ['reasoning', '推理能力', '官方 Reasoning 跑分，越高越强'],
  ['value', '性价比', '能力均分 ÷ 全网最低输入价，每 1 元买到的分数'],
  ['speed', '站点速度', '延迟中位数映射，100 为 1 秒内'],
  ['consistency', '站点稳定性', 'p95 相对 p50 的抖动，越小越稳'],
];

const avg = (m: ModelSummary) => (m.codingScore + m.reasoningScore) / 2;
const valueOf = (m: ModelSummary) => (m.minInputCny > 0 ? avg(m) / m.minInputCny : 0);

export function LeaderboardsPage() {
  const [tab, setTab] = useState<Tab>('coding');
  const [models, setModels] = useState<ModelSummary[] | null>(null);
  const [stations, setStations] = useState<Station[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    Promise.all([api.models(), api.stations()])
      .then(([m, s]) => {
        setModels(m.items);
        setStations(s.items);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  const modelRows = useMemo(() => {
    if (!models) return [];
    const list = [...models];
    if (tab === 'coding') list.sort((a, b) => b.codingScore - a.codingScore);
    else if (tab === 'reasoning') list.sort((a, b) => b.reasoningScore - a.reasoningScore);
    else list.sort((a, b) => valueOf(b) - valueOf(a));
    return list;
  }, [models, tab]);

  const isStationTab = tab === 'speed' || tab === 'consistency';

  const answer = useMemo(() => {
    if (!stations?.length) return null;
    const byScore = [...stations].sort((a, b) => b.rankScore - a.rankScore);
    const byUptime = [...stations].sort((a, b) => b.uptimePct - a.uptimePct);
    const bySpeed = [...stations].sort((a, b) => b.speed - a.speed);
    const byPrice = [...stations].filter((s) => s.fromPriceCny >= 0 && s.purityTier !== 'counterfeit').sort((a, b) => a.fromPriceCny - b.fromPriceCny);
    return {
      best: byScore[0],
      second: byScore[1],
      tie: byScore.length > 1 && Math.abs(byScore[0].rankScore - byScore[1].rankScore) <= byScore[0].scoreMargin + byScore[1].scoreMargin,
      uptime: byUptime[0],
      speed: bySpeed[0],
      cheap: byPrice[0],
      updated: byScore[0]?.lastProbeAt,
    };
  }, [stations]);

  const speedRows = useMemo(() => {
    if (!stations) return [];
    return [...stations].sort((a, b) =>
      tab === 'speed'
        ? b.speed - a.speed || b.uptimePct - a.uptimePct
        : b.consistency - a.consistency || b.uptimePct - a.uptimePct,
    );
  }, [stations, tab]);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />LEADERBOARDS</span>
          <h1>排行榜</h1>
          <p className="sub">能力分来自官方公开基准，价格取全网中转站最低实价；两个维度分开看，不合成一个玄学总分。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/leaderboards/window">周榜 / 月榜 →</Link>
      </div>

      {error && <div className="error-box">{error}</div>}

      {answer && (
        <section className="panel answer-card">
          <span className="eyebrow"><i />CURRENT ANSWER</span>
          <p className="answer-text">
            当前 {stations?.length} 家站点里，
            <Link to={`/stations/${answer.best.slug}`}>{answer.best.name}</Link> 综合分{' '}
            <b>{answer.best.rankScore.toFixed(1)}</b>{' '}
            {answer.best.scoreMargin > 0 && (
              <span className="ci" title="95% 置信区间，仅统计可用性抽样误差">±{answer.best.scoreMargin.toFixed(1)}</span>
            )}{' '}
            排第一{answer.tie && answer.second ? `（与第 2 名 ${answer.second.name} 的差距在误差范围内）` : ''}；原始可用率最高的是{' '}
            <Link to={`/stations/${answer.uptime.slug}`}>{answer.uptime.name}</Link>（{answer.uptime.uptimePct.toFixed(1)}%），
            速度最快的是 <Link to={`/stations/${answer.speed.slug}`}>{answer.speed.name}</Link>（{answer.speed.speed.toFixed(0)} 分），
            最低输入价 <b>{fmtCny(answer.cheap.fromPriceCny)}</b>/1M 来自{' '}
            <Link to={`/stations/${answer.cheap.slug}`}>{answer.cheap.name}</Link>。
          </p>
          <p className="fine">
            数据时间 {answer.updated ? new Date(answer.updated).toLocaleString('zh-CN', { hour12: false }) : '—'} · 口径 v2.0 ·
            排名随探针更新变化，价格不含在综合分内。
          </p>
        </section>
      )}

      <div className="tabs">
        {TABS.map(([key, label]) => (
          <button key={key} className={`tab ${tab === key ? 'on' : ''}`} onClick={() => setTab(key)}>{label}</button>
        ))}
        <span className="tabs-note">{TABS.find(([k]) => k === tab)?.[2]}</span>
      </div>

      {!isStationTab && (
        <section className="panel">
          <div className="table-wrap lb-table">
            <table className="index-table">
              <thead>
                <tr>
                  <th className="col-rank">#</th>
                  <th>模型</th>
                  <th className="hide-md">厂商</th>
                  <th>{tab === 'reasoning' ? '推理分' : '编程分'}</th>
                  {tab === 'value' && <th className="num">性价比</th>}
                  <th className="num">最低输入价</th>
                  <th className="num hide-sm">输出价</th>
                  <th className="num hide-sm">报价站点</th>
                </tr>
              </thead>
              <tbody>
                {!models && !error && <tr><td colSpan={8} className="loading"><Spinner /> 加载中…</td></tr>}
                {modelRows.map((m, i) => {
                  const shown = tab === 'reasoning' ? m.reasoningScore : m.codingScore;
                  const width = Math.max(4, Math.min(100, ((shown - 80) / 20) * 100));
                  return (
                    <tr key={m.id} className="clickable" onClick={() => navigate(`/models/${encodeURIComponent(m.id)}`)}>
                      <td className={`col-rank rank ${i < 3 ? `r${i + 1}` : ''}`}>{i + 1}</td>
                      <td>
                        <div className="cell-station">
                          <VendorMark vendor={m.vendor} size={28} />
                          <div className="station-meta">
                            <Link className="station-name" to={`/models/${encodeURIComponent(m.id)}`} onClick={(e) => e.stopPropagation()}>{m.name}</Link>
                            <span className="mono dim tiny">{m.id}</span>
                          </div>
                        </div>
                      </td>
                      <td className="hide-md dim">{m.vendorName}</td>
                      <td>
                        <div className="metric">
                          <b className="mono">{shown.toFixed(1)}</b>
                          <span className="meter"><i style={{ width: `${width}%` }} /></span>
                        </div>
                      </td>
                      {tab === 'value' && (
                        <td className="num mono accent">{valueOf(m).toFixed(1)}<small className="per"> 分/元</small></td>
                      )}
                      <td className="num mono">{fmtCny(m.minInputCny)}</td>
                      <td className="num mono dim hide-sm">{fmtCny(m.minOutputCny)}</td>
                      <td className="num mono dim hide-sm">{m.offersCount}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <div className="lb-cards">
            {!models && Array.from({ length: 5 }).map((_, i) => (
              <div key={`skm-${i}`} className="lb-card"><span className="sk" style={{ width: 120 }} /></div>
            ))}
            {modelRows.map((m, i) => {
              const shown = tab === 'reasoning' ? m.reasoningScore : m.codingScore;
              return (
                <Link key={m.id} className="lb-card" to={`/models/${encodeURIComponent(m.id)}`}>
                  <span className={`rank r${Math.min(i + 1, 4)}`}>{i + 1}</span>
                  <VendorMark vendor={m.vendor} size={26} />
                  <div className="lb-id">
                    <b>{m.name}</b>
                    <small>{m.vendorName}</small>
                  </div>
                  <div className="lb-metric">
                    <b className="mono">{shown.toFixed(1)}</b>
                    <small>{fmtCny(m.minInputCny)}{tab === 'value' ? ` · ${valueOf(m).toFixed(1)} 分/元` : ''}</small>
                  </div>
                </Link>
              );
            })}
          </div>
          <div className="panel-foot mono dim">
            {tab === 'value'
              ? '性价比 =（编程分 + 推理分）/ 2 ÷ 最低输入价，仅作横向参考；低价站可能伴随更低的可用性与纯度。'
              : '跑分为厂商公开基准，不构成对实际任务表现的保证；价格取全站最低分组。'}
          </div>
        </section>
      )}

      {isStationTab && (
        <section className="panel">
          <div className="table-wrap lb-table">
            <table className="index-table">
              <thead>
                <tr>
                  <th className="col-rank">#</th>
                  <th>站点</th>
                  <th className="num">{tab === 'speed' ? '速度分' : '稳定性分'}</th>
                  <th>原始可用率</th>
                  <th>纯度</th>
                  <th className="num">最低输入价</th>
                  <th className="num hide-sm">模型数</th>
                </tr>
              </thead>
              <tbody>
                {!stations && !error && <tr><td colSpan={7} className="loading"><Spinner /> 加载中…</td></tr>}
                {speedRows.map((s, i) => (
                  <tr key={s.slug}>
                    <td className={`col-rank rank ${i < 3 ? `r${i + 1}` : ''}`}>{i + 1}</td>
                    <td>
                      <div className="cell-station">
                        <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={30} />
                        <div className="station-meta">
                          <Link className="station-name" to={`/stations/${s.slug}`}>{s.name}</Link>
                          {s.claimed && <span className="badge-ok" title="站长已认领">✓</span>}
                          <span className="station-domain mono">{s.domain}</span>
                        </div>
                      </div>
                    </td>
                    <td className="num"><b className="score">{(tab === 'speed' ? s.speed : s.consistency).toFixed(0)}</b></td>
                    <td>
                      <div className="avail">
                        <span className="mono">{s.uptimePct.toFixed(1)}%</span>
                        <UptimeBars data={s.uptime} />
                      </div>
                    </td>
                    <td><TierBadge tier={s.purityTier} score={s.purity} /></td>
                    <td className="num mono">{fmtCny(s.fromPriceCny)}</td>
                    <td className="num mono dim hide-sm">{s.modelsCount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="lb-cards">
            {!stations && Array.from({ length: 5 }).map((_, i) => (
              <div key={`sks-${i}`} className="lb-card"><span className="sk" style={{ width: 120 }} /></div>
            ))}
            {speedRows.map((s, i) => (
              <Link key={s.slug} className="lb-card" to={`/stations/${s.slug}`}>
                <span className={`rank r${Math.min(i + 1, 4)}`}>{i + 1}</span>
                <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={26} />
                <div className="lb-id">
                  <b>{s.name}{s.claimed && <span className="badge-ok" title="站长已认领">✓</span>}</b>
                  <small>{s.uptimePct.toFixed(1)}% · {s.purityTier === 'official' ? '官方直达' : s.purityTier === 'mixed' ? '混合' : '疑似掺假'}</small>
                </div>
                <div className="lb-metric">
                  <b className="mono">{(tab === 'speed' ? s.speed : s.consistency).toFixed(0)}</b>
                  <small>{fmtCny(s.fromPriceCny)}</small>
                </div>
              </Link>
            ))}
          </div>
          <div className="panel-foot mono dim">
            {tab === 'speed'
              ? '速度为探针延迟中位数映射：1 秒内 100 分，8 秒 0 分；不反映长输出吞吐。'
              : '稳定性 = p95 相对 p50 的抖动，抖动越小分越高；样本不足 10 条按 60 分处理。'}
          </div>
        </section>
      )}
    </main>
  );
}

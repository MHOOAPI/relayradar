import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../../api';
import type { StationDetail } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';
import { StationMark } from '../../components/marks';

export function EnStationPage() {
  const { slug = '' } = useParams();
  const [detail, setDetail] = useState<StationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.station(slug)
      .then((d) => {
        setDetail(d);
        document.title = `${d.station.name} relay profile | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [slug]);

  if (error) return <main className="wrap"><div className="error-box">{error} <Link className="link-btn" to="/en/leaderboards">Back to leaderboard</Link></div></main>;
  if (!detail) return <main className="wrap"><div className="loading-page"><Spinner /> Loading provider…</div></main>;

  const s = detail.station;
  const pts = detail.history;
  const delta = pts.length > 1 ? pts[pts.length - 1].score - pts[0].score : 0;

  return (
    <main className="wrap">
      <div className="crumbs mono">
        <Link to="/en/leaderboards">Leaderboard</Link><span>/</span><span className="dim">{s.slug}</span>
      </div>

      <section className="panel detail-head">
        <div className="detail-main">
          <StationMark slug={s.slug} name={s.name} iconUrl={s.iconUrl} size={54} />
          <div>
            <div className="detail-title">
              <h1>{s.name}</h1>
              <span className="svc mono">{s.domain}</span>
            </div>
            <p className="tagline">{s.taglineEn || s.tagline}</p>
            <div className="detail-links mono dim">
              <span>rank #{s.rank}</span>
              <span>·</span>
              <span>{s.groupsCount} groups</span>
              <span>·</span>
              <span>{s.modelsCount} models</span>
              <span>·</span>
              <Link to={`/stations/${s.slug}`}>中文档案</Link>
            </div>
          </div>
        </div>
        <div className="model-scores">
          <div><span>Score</span><b className="mono">{s.rankScore.toFixed(1)}</b></div>
          <div><span>± margin</span><b className="mono">±{s.scoreMargin.toFixed(1)}</b></div>
          <div><span>Raw uptime</span><b className="mono">{s.uptimePct.toFixed(1)}%</b></div>
          <div><span>Min input</span><b className="mono accent">{fmtCny(s.fromPriceCny)}</b></div>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Metrics (7-day window)</h2>
            <p className="sub">Availability is a Wilson 95% lower bound; speed maps median latency; consistency measures p95 jitter.</p>
          </div>
          <span className="mono dim">{pts.length > 1 ? `${delta >= 0 ? '+' : ''}${delta.toFixed(1)} vs 30d` : '—'}</span>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <tbody>
              <tr><td className="dim">Availability score</td><td className="num mono">{s.availability.toFixed(1)}</td><td className="dim">Raw uptime</td><td className="num mono">{s.uptimePct.toFixed(1)}%</td></tr>
              <tr><td className="dim">Speed</td><td className="num mono">{s.speed.toFixed(1)}</td><td className="dim">Consistency</td><td className="num mono">{s.consistency.toFixed(1)}</td></tr>
              <tr><td className="dim">Purity</td><td className="num mono">{s.purityTier}</td><td className="dim">Samples</td><td className="num mono">{s.samples}</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head">
          <div><h2>Groups</h2><p className="sub">Multiplier applies to the official USD price; lower is not always better.</p></div>
          <span className="mono dim">{detail.groups.length}</span>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <thead><tr><th>Group</th><th className="num">Multiplier</th><th className="num">Latency</th><th className="num hide-sm">Availability</th><th className="num hide-sm">Samples</th><th className="hide-sm">Status</th></tr></thead>
            <tbody>
              {detail.groups.map((g) => (
                <tr key={g.slug}>
                  <td className="mono">{g.name.replace('默认组', 'default')}</td>
                  <td className="num mono">{g.multiplier.toFixed(2)}×</td>
                  <td className="num mono">{g.latencyMs} ms</td>
                  <td className="num mono hide-sm">{g.availability.toFixed(1)}%</td>
                  <td className="num mono dim hide-sm">{g.samples}</td>
                  <td className="hide-sm">{g.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head">
          <div><h2>Models and best prices</h2><p className="sub">Cheapest group for each model on this provider.</p></div>
          <span className="mono dim">{detail.models.length}</span>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <thead><tr><th>Model</th><th className="num">Input /1M</th><th className="num">Output /1M</th><th className="hide-sm">Cheapest group</th></tr></thead>
            <tbody>
              {detail.models.map((m) => (
                <tr key={m.model.id}>
                  <td><Link className="station-name" to={`/en/models/${m.model.id}`}>{m.model.name}</Link></td>
                  <td className="num mono">{fmtCny(m.bestInputCny)}</td>
                  <td className="num mono accent">{fmtCny(m.bestOutputCny)}</td>
                  <td className="mono dim hide-sm">{m.cheapestGroup.replace('默认组', 'default')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  );
}

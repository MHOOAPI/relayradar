import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { Station } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';

export function EnLeaderboardPage() {
  const [items, setItems] = useState<Station[] | null>(null);

  useEffect(() => {
    document.title = 'Relay leaderboard: all providers ranked | RelayRadar';
    api.stations().then((r) => setItems(r.items)).catch(() => setItems([]));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />LEADERBOARD</span>
          <h1>Relay leaderboard</h1>
          <p className="sub">All providers ranked by the v2 score: 0.62 × availability + 0.26 × speed + 0.12 × consistency. Price is separate. <Link to="/en/methodology">Methodology</Link> · <Link to="/leaderboards">中文排行榜</Link></p>
        </div>
      </div>

      {!items && <div className="loading"><Spinner /> Loading leaderboard…</div>}

      {items && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead>
                <tr>
                  <th className="num">#</th>
                  <th>Provider</th>
                  <th className="hide-sm">Domain</th>
                  <th className="num">Score</th>
                  <th className="num hide-sm">Raw uptime</th>
                  <th className="num hide-sm">Speed</th>
                  <th className="num hide-sm">Consistency</th>
                  <th className="num">Min input</th>
                </tr>
              </thead>
              <tbody>
                {items.map((s) => (
                  <tr key={s.slug}>
                    <td className="num rank">{s.rank}</td>
                    <td><Link className="station-name" to={`/en/stations/${s.slug}`}>{s.name}</Link></td>
                    <td className="mono dim hide-sm">{s.domain}</td>
                    <td className="num mono">{s.rankScore.toFixed(1)} {s.scoreMargin > 0 && <small className="dim">±{s.scoreMargin.toFixed(1)}</small>}</td>
                    <td className="num mono hide-sm">{s.uptimePct.toFixed(1)}%</td>
                    <td className="num mono hide-sm">{s.speed.toFixed(1)}</td>
                    <td className="num mono hide-sm">{s.consistency.toFixed(1)}</td>
                    <td className="num mono accent">{fmtCny(s.fromPriceCny)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="panel-foot mono dim">Demo dataset — scores and probes are generated; see the FAQ.</div>
        </section>
      )}
    </main>
  );
}

import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { Station } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';

export function EnHomePage() {
  const [stations, setStations] = useState<Station[] | null>(null);

  useEffect(() => {
    document.title = 'RelayRadar — independent LLM API relay monitoring';
    api.stations().then((r) => setStations(r.items)).catch(() => setStations([]));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />INDEPENDENT RELAY INDEX</span>
          <h1>Independent rankings for LLM API relay providers</h1>
          <p className="sub">
            Continuous probes, a Wilson 95% availability lower bound, latency and consistency. Price and purity are
            shown separately and never change the score. <Link to="/en/methodology">Methodology</Link> ·{' '}
            <Link to="/en/faq">FAQ</Link> · <Link to="/">中文站</Link>
          </p>
        </div>
      </div>

      {!stations && <div className="loading"><Spinner /> Loading rankings…</div>}

      {stations && (
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
                  <th className="num">Min input /1M</th>
                </tr>
              </thead>
              <tbody>
                {stations.slice(0, 10).map((s) => (
                  <tr key={s.slug}>
                    <td className="num rank">{s.rank}</td>
                    <td><Link className="station-name" to={`/en/stations/${s.slug}`}>{s.name}</Link></td>
                    <td className="mono dim hide-sm">{s.domain}</td>
                    <td className="num mono">{s.rankScore.toFixed(1)} {s.scoreMargin > 0 && <small className="dim">±{s.scoreMargin.toFixed(1)}</small>}</td>
                    <td className="num mono hide-sm">{s.uptimePct.toFixed(1)}%</td>
                    <td className="num mono accent">{fmtCny(s.fromPriceCny)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="panel-foot mono dim">
            Provider names are shown as published (often Chinese); domains are the stable identifier. Data is a demo
            dataset — see the FAQ.
          </div>
        </section>
      )}
    </main>
  );
}

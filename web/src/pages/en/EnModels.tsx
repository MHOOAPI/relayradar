import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { ModelSummary } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';

export function EnModelsPage() {
  const [items, setItems] = useState<ModelSummary[] | null>(null);

  useEffect(() => {
    document.title = 'Model directory and lowest relay prices | RelayRadar';
    api.models().then((r) => setItems(r.items)).catch(() => setItems([]));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />MODELS</span>
          <h1>Model directory and lowest relay prices</h1>
          <p className="sub">Official USD prices next to the cheapest relay price in CNY, with context window and license. <Link to="/en">Rankings</Link> · <Link to="/en/methodology">Methodology</Link></p>
        </div>
      </div>
      {!items && <div className="loading"><Spinner /> Loading models…</div>}
      {items && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead>
                <tr><th>Model</th><th className="hide-sm">Vendor</th><th className="num hide-sm">Context</th><th className="num">Official /1M</th><th className="num">Cheapest input</th><th className="hide-sm">License</th></tr>
              </thead>
              <tbody>
                {items.map((m) => (
                  <tr key={m.id}>
                    <td>
                      <div className="station-meta">
                        <Link className="station-name" to={`/en/models/${m.id}`}>{m.name}</Link>
                        <span className="mono dim tiny">{m.id}</span>
                      </div>
                    </td>
                    <td className="dim hide-sm">{m.vendorName}</td>
                    <td className="num mono hide-sm">{m.contextWindow >= 1000000 ? `${(m.contextWindow / 1000000).toFixed(1)}M` : `${Math.round(m.contextWindow / 1000)}K`}</td>
                    <td className="num mono">${m.inputUsd} / ${m.outputUsd}</td>
                    <td className="num mono accent">{fmtCny(m.minInputCny)}</td>
                    <td className="dim hide-sm">{m.license}</td>
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

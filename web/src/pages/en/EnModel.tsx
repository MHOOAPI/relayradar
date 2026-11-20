import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../../api';
import type { ModelSummary, Offer } from '../../types';
import { fmtCny, Spinner } from '../../components/ui';
import { VendorMark } from '../../components/marks';

export function EnModelPage() {
  const { id = '' } = useParams();
  const [model, setModel] = useState<ModelSummary | null>(null);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.model(id)
      .then((r) => {
        setModel(r.model);
        setOffers(r.offers ?? []);
        document.title = `${r.model.name} pricing and relays | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [id]);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!model) return <main className="wrap"><div className="loading-page"><Spinner /> Loading model…</div></main>;

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/en">Rankings</Link><span>/</span><Link to="/en/models">Models</Link><span>/</span><span className="dim">{model.id}</span></div>
      <section className="panel detail-head">
        <div className="detail-main">
          <VendorMark vendor={model.vendor} size={54} />
          <div>
            <div className="detail-title">
              <h1>{model.name}</h1>
              <span className="svc">{model.vendorName}</span>
            </div>
            <p className="tagline">{model.summaryEn || model.summary}</p>
            <div className="detail-links mono dim">
              <span>{model.id}</span>
              <span>·</span>
              <span>context {model.contextWindow >= 1000000 ? `${(model.contextWindow / 1000000).toFixed(1)}M` : `${Math.round(model.contextWindow / 1000)}K`}</span>
              <span>·</span>
              <Link to={`/en/models/${model.id}`}>license {model.license}</Link>
            </div>
          </div>
        </div>
        <div className="model-scores">
          <div><span>Official input</span><b className="mono">${model.inputUsd}</b></div>
          <div><span>Official output</span><b className="mono">${model.outputUsd}</b></div>
          <div><span>Coding</span><b className="mono">{model.codingScore}</b></div>
          <div><span>Reasoning</span><b className="mono">{model.reasoningScore}</b></div>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Relay offers (cheapest input first)</h2>
            <p className="sub">The same model can differ several times over across groups; check purity and availability too.</p>
          </div>
          <span className="mono dim">{offers.length} offers</span>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <thead>
              <tr><th className="num">#</th><th>Provider</th><th className="hide-sm">Group</th><th className="num">Multiplier</th><th className="num">Input /1M</th><th className="num">Output /1M</th><th className="hide-sm">Purity</th></tr>
            </thead>
            <tbody>
              {offers.map((o, i) => (
                <tr key={`${o.stationSlug}-${o.groupSlug}`}>
                  <td className="num dim">{i + 1}</td>
                  <td><Link className="station-name" to={`/en/stations/${o.stationSlug}`}>{o.stationName}</Link></td>
                  <td className="mono dim hide-sm">{o.groupName}</td>
                  <td className="num mono">{o.multiplier.toFixed(2)}×</td>
                  <td className="num mono">{fmtCny(o.inputCny)}</td>
                  <td className="num mono accent">{fmtCny(o.outputCny)}</td>
                  <td className="dim hide-sm">{o.purityTier}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  );
}

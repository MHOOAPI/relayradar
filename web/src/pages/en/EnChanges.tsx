import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { Change } from '../../types';
import { Spinner } from '../../components/ui';

const KIND_LABEL: Record<string, string> = {
  new_station: 'new listing',
  purity_alert: 'purity alert',
  availability_drop: 'availability drop',
  price_change: 'price move',
  score_drop: 'score regression',
};

export function EnChangesPage() {
  const [items, setItems] = useState<Change[] | null>(null);
  const [kind, setKind] = useState('all');

  useEffect(() => {
    document.title = 'Change feed: new listings, purity alerts and price moves | RelayRadar';
    api.changes(undefined, undefined, 'en').then((r) => setItems(r.items)).catch(() => setItems([]));
  }, []);

  const shown = items?.filter((c) => kind === 'all' || c.kind === kind) ?? [];

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />CHANGE FEED</span>
          <h1>Change feed</h1>
          <p className="sub">New listings, purity alerts, availability drops, price moves and score regressions — derived from the same probes as the rankings. <Link to="/changes">中文变更雷达</Link></p>
        </div>
      </div>

      <div className="chips">
        {['all', 'price_change', 'availability_drop', 'purity_alert', 'score_drop', 'new_station'].map((k) => (
          <button key={k} className={`chip ${kind === k ? 'on' : ''}`} onClick={() => setKind(k)}>
            {k === 'all' ? 'All' : KIND_LABEL[k]}
          </button>
        ))}
      </div>

      {!items && <div className="loading"><Spinner /> Loading changes…</div>}

      {items && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th>Time</th><th>Provider</th><th>Kind</th><th>Change</th><th className="hide-sm">Detail</th></tr></thead>
              <tbody>
                {shown.length === 0 && <tr><td colSpan={5}><span className="fine">No changes of this kind.</span></td></tr>}
                {shown.map((c) => (
                  <tr key={c.id}>
                    <td className="mono dim">{new Date(c.at).toLocaleString('en-US', { hour12: false })}</td>
                    <td><Link className="station-name" to={`/en/stations/${c.stationSlug}`}>{c.stationName}</Link></td>
                    <td><span className={`sev sev-${c.severity}`}>{KIND_LABEL[c.kind] ?? c.kind}</span></td>
                    <td>{c.title}</td>
                    <td className="fine hide-sm">{c.detail}</td>
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

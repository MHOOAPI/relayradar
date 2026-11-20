import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { ModelProviderStatus } from '../../types';
import { Spinner } from '../../components/ui';

const LABEL: Record<string, string> = {
  operational: 'Operational',
  degraded: 'Degraded',
  outage: 'Outage',
  unknown: 'Unknown',
};

const SOURCE: Record<string, string> = {
  statuspage: 'official status page',
  probe: 'reachability probe',
};

export function EnModelStatusPage() {
  const [items, setItems] = useState<ModelProviderStatus[] | null>(null);
  const [checkedAt, setCheckedAt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = 'Upstream model API status | RelayRadar';
    let alive = true;
    const load = () =>
      api.modelStatus()
        .then((r) => {
          if (!alive) return;
          setItems(r.providers);
          setCheckedAt(r.checkedAt);
          setError(null);
        })
        .catch((e: Error) => alive && setError(e.message));
    load();
    const timer = window.setInterval(load, 60000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />UPSTREAM STATUS</span>
          <h1>Upstream model API status</h1>
          <p className="sub">
            Official API status for OpenAI, Anthropic, Google Gemini, DeepSeek, Moonshot, Alibaba Qwen and xAI.
            We read official Statuspage APIs where available and fall back to endpoint reachability.
          </p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/en/status">RelayRadar status →</Link>
      </div>

      {error && <div className="error-box">{error}</div>}
      {!items && !error && <div className="loading"><Spinner /> Loading provider status…</div>}

      <div className="provider-grid">
        {items?.map((p) => (
          <article key={p.key} className={`provider-card status-${p.status}`}>
            <div className="provider-head">
              <b>{p.name}</b>
              <span className={`status-pill ${p.status}`}>{LABEL[p.status] ?? p.status}</span>
            </div>
            <Link className="provider-detail-link" to={`/en/model-status/${p.key}`}>View history →</Link>
            <p className="provider-desc">{p.description || '—'}</p>
            {p.components && p.components.length > 0 && (
              <div className="provider-components">
                {p.components.slice(0, 4).map((c) => <span key={c.name} className={`pc ${c.status}`}>{c.name}</span>)}
              </div>
            )}
            {p.incidents && p.incidents.length > 0 && (
              <a className="provider-incident" href={p.incidents[0].url || p.statusUrl} target="_blank" rel="noopener noreferrer">
                Latest: {p.incidents[0].name}
              </a>
            )}
            <div className="provider-foot">
              <span className={`source-badge ${p.checkedVia}`}>{SOURCE[p.checkedVia] ?? p.checkedVia}</span>
              <a href={p.statusUrl} target="_blank" rel="noopener noreferrer">official status ↗</a>
            </div>
          </article>
        ))}
      </div>

      <p className="fine" style={{ marginTop: 16 }}>
        {checkedAt && `Checked ${new Date(checkedAt).toLocaleString('en-US')} · refreshes every 60s. `}
        This is the upstream providers' status, not RelayRadar itself — see <Link to="/en/status">/en/status</Link>.
        Probe failures may also come from local network or regional restrictions.
      </p>
    </main>
  );
}

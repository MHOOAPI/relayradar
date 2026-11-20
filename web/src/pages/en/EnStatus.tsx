import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { StatusResponse } from '../../types';
import { Spinner } from '../../components/ui';

const LABEL: Record<string, string> = {
  operational: 'All systems operational',
  degraded: 'Partial degradation',
  outage: 'Service outage',
  unknown: 'Unknown',
};

const INCIDENT_LABEL: Record<string, string> = {
  investigating: 'investigating',
  identified: 'identified',
  monitoring: 'monitoring',
  resolved: 'resolved',
};

const COMPONENT_DESC: Record<string, string> = {
  api: 'Public JSON API and pages',
  database: 'Postgres / in-memory snapshot',
  probes: 'Station probing pipeline',
  alerts: 'Webhook and email delivery',
  audit: 'OpenAI-compatible endpoint audit',
  email: 'SMTP mail channel',
};

const COMPONENT_NAME: Record<string, string> = {
  api: 'Public API',
  database: 'Data store',
  probes: 'Probe pipeline',
  alerts: 'Alert delivery',
  audit: 'Endpoint audit',
  email: 'Email channel',
};

export function EnStatusPage() {
  const [data, setData] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = 'Service status | RelayRadar';
    let alive = true;
    const load = () => api.status().then((r) => alive && setData(r)).catch((e: Error) => alive && setError(e.message));
    load();
    const timer = window.setInterval(load, 30000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, []);

  const overall = data?.status ?? 'operational';
  return (
    <main className="wrap">
      <section className={`status-hero status-${overall}`}>
        <div className="status-hero-main">
          <span className="status-orb"><i /></span>
          <div>
            <span className="mono status-kicker">SYSTEM STATUS</span>
            <h1>{LABEL[overall] ?? 'Unknown'}</h1>
            <p>{data ? `Checked ${new Date(data.checkedAt).toLocaleTimeString('en-US')} · v${data.version}` : 'Loading…'}</p>
          </div>
        </div>
        <div className="status-hero-side mono"><span>API</span><b>/api/v1/status</b><span>Refresh</span><b>30s</b></div>
      </section>

      {error && <div className="error-box">Status API unavailable: {error}</div>}
      {!data && !error && <div className="loading"><Spinner /> Loading components…</div>}

      {data && (
        <section className="panel status-panel">
          <div className="panel-head"><div><h2>Components</h2><p className="sub">Checked against real dependencies.</p></div></div>
          <ul className="status-list">
            {data.components.map((c) => (
              <li key={c.key} className={`status-row status-${c.status}`}>
                <span className="status-dot" />
                <div className="status-info"><b>{COMPONENT_NAME[c.key] ?? c.name}</b><small>{COMPONENT_DESC[c.key] ?? ''}</small></div>
                <span className={`status-pill ${c.status}`}>{LABEL[c.status] ?? c.status}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="panel status-panel">
        <div className="panel-head"><div><h2>Recent incidents</h2><p className="sub">Public incidents and maintenance.</p></div><a className="mono dim tiny" href="/status.rss">RSS</a></div>
        {data && data.incidents.length > 0 ? (
          <ul className="incident-list">
            {data.incidents.map((i) => (
              <li key={i.id} className="incident">
                <span className="status-pill operational">{INCIDENT_LABEL[i.status] ?? i.status}</span>
                <div className="incident-main">
                  <b>{i.title}</b>
                  {i.body && <p>{i.body}</p>}
                  <small className="mono">{i.component} · {new Date(i.startedAt).toLocaleString('en-US')}</small>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <div className="status-empty"><span className="status-dot" /><p>No public incidents in the last 90 days.</p></div>
        )}
      </section>

      <p className="fine" style={{ marginTop: 14 }}><Link to="/status">中文状态页</Link> · <Link to="/en">Rankings</Link></p>
    </main>
  );
}

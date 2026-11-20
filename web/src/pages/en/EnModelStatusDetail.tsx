import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../../api';
import type { ModelProviderStatus } from '../../types';
import { Spinner } from '../../components/ui';

const STATUS_LABEL: Record<string, string> = {
  operational: 'Operational',
  degraded: 'Degraded',
  outage: 'Outage',
  unknown: 'Unknown',
};

const IMPACT_LABEL: Record<string, string> = {
  none: 'none',
  minor: 'minor',
  major: 'major',
  critical: 'critical',
  maintenance: 'maintenance',
};

const dayOf = (iso?: string) => (iso ? iso.slice(0, 10) : '');

export function EnModelStatusDetailPage() {
  const { key = '' } = useParams();
  const [providers, setProviders] = useState<ModelProviderStatus[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.modelStatus()
      .then((r) => {
        setProviders(r.providers);
        const p = r.providers.find((x) => x.key === key);
        if (p) document.title = `${p.name} official API status history | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [key]);

  const provider = providers?.find((p) => p.key === key) ?? null;

  const days = useMemo(() => {
    const map = new Map<string, { date: string; items: NonNullable<ModelProviderStatus['incidents']> }>();
    for (const inc of provider?.incidents ?? []) {
      const d = dayOf(inc.startedAt || inc.created);
      if (!d) continue;
      if (!map.has(d)) map.set(d, { date: d, items: [] });
      map.get(d)!.items.push(inc);
    }
    return [...map.values()].sort((a, b) => b.date.localeCompare(a.date));
  }, [provider]);

  if (error) return <main className="wrap"><div className="error-box">{error}</div></main>;
  if (!providers) return <main className="wrap"><div className="loading-page"><Spinner /> Loading status history…</div></main>;
  if (!provider) return <main className="wrap"><div className="error-box">Provider not found. <Link className="link-btn" to="/en/model-status">Back</Link></div></main>;

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/en/model-status">Model API status</Link><span>/</span><span className="dim">{provider.key}</span></div>

      <div className="provider-tabs">
        {providers.map((p) => (
          <Link key={p.key} className={`provider-tab ${p.key === provider.key ? 'on' : ''}`} to={`/en/model-status/${p.key}`}>
            <i className={`pc-dot ${p.status}`} />
            {p.name}
          </Link>
        ))}
      </div>

      <section className={`model-status-hero status-${provider.status}`}>
        <div>
          <span className="mono status-kicker">OFFICIAL API STATUS</span>
          <h1>{provider.name} official API status</h1>
          <p>Current: {STATUS_LABEL[provider.status] ?? provider.status} · {provider.description || '—'}</p>
          <small className="mono">
            {provider.checkedVia === 'statuspage' ? 'from official status page API' : 'from reachability probe'}
            {' · '}
            {new Date(provider.checkedAt).toLocaleString('en-US')}
          </small>
        </div>
        <a className="btn btn-ghost btn-sm" href={provider.statusUrl} target="_blank" rel="noopener noreferrer">Official status ↗</a>
      </section>

      {provider.history && provider.history.some((d) => d.status !== 'unknown') && (
        <div className="provider-history panel">
          <div className="provider-history-bars">
            {provider.history.map((d) => <i key={d.date} className={`b-${d.status}`} title={`${d.date} · ${d.uptime.toFixed(1)}%`} />)}
          </div>
          <span className="mono dim">
            {(() => {
              const known = provider.history!.filter((d) => d.status !== 'unknown');
              if (!known.length) return 'no history yet';
              const up = known.reduce((s, d) => s + d.uptime, 0) / known.length;
              return `last 90d · ${known.length} days · uptime ${up.toFixed(1)}%`;
            })()}
          </span>
        </div>
      )}

      <div className="status-history-head">
        <h2>Incident history</h2>
        <span className="mono dim">{provider.incidents?.length ?? 0} incidents</span>
      </div>

      {days.length === 0 && <div className="panel status-empty"><span className="status-dot" /><p>No public incidents from the official status page.</p></div>}

      <div className="status-timeline">
        {days.map((d) => {
          const worst = d.items.some((i) => i.impact === 'critical' || i.impact === 'major')
            ? 'bad'
            : d.items.some((i) => i.impact === 'minor')
              ? 'warn'
              : 'ok';
          return (
            <article key={d.date} className={`timeline-day timeline-${worst}`}>
              <div className="timeline-date mono"><b>{d.date.slice(8, 10)}</b><span>{d.date.slice(0, 7)}</span></div>
              <div className="timeline-body">
                <div className="timeline-day-head">
                  <span className={`status-pill ${worst === 'ok' ? 'operational' : worst === 'warn' ? 'degraded' : 'outage'}`}>
                    {worst === 'ok' ? 'normal' : worst === 'warn' ? 'degraded' : 'incident'}
                  </span>
                  <span className="fine">{d.items.length} incident(s)</span>
                </div>
                {d.items.map((inc) => (
                  <div className="timeline-incident" key={`${inc.name}-${inc.created}`}>
                    <b>{inc.name}</b>
                    <div className="timeline-meta mono">
                      <span>{inc.status}</span>
                      {inc.impact && <span>· {IMPACT_LABEL[inc.impact] ?? inc.impact}</span>}
                      {inc.components && inc.components.length > 0 && <span>· {inc.components.join(' / ')}</span>}
                    </div>
                    {inc.updates && inc.updates.length > 0 && (
                      <ul className="timeline-updates">
                        {inc.updates.slice(0, 4).map((u, idx) => (
                          <li key={idx}>
                            <span className="mono">{new Date(u.createdAt).toLocaleTimeString('en-US')}</span>
                            <span className="mono dim">{u.status}</span>
                            <p>{u.body}</p>
                          </li>
                        ))}
                      </ul>
                    )}
                    {inc.url && <a className="link-btn tiny" href={inc.url} target="_blank" rel="noopener noreferrer">Official incident ↗</a>}
                  </div>
                ))}
              </div>
            </article>
          );
        })}
      </div>

      <p className="fine" style={{ marginTop: 16 }}>
        Incident data comes from the provider's official Statuspage; providers without one show reachability only.
        RelayRadar's own status: <Link to="/en/status">/en/status</Link>.
      </p>
    </main>
  );
}

import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../../api';
import type { AuditReport, CheckStatus, ModelOption, ModelSummary } from '../../types';
import { Spinner } from '../../components/ui';
import { AlertMark, CheckMark, XMark } from '../../components/icons';

const STATUS_LABEL: Record<CheckStatus, string> = { pass: 'pass', warn: 'watch', fail: 'fail' };

export function EnAuditPage() {
  const [params] = useSearchParams();
  const [baseUrl, setBaseUrl] = useState(params.get('url') ?? '');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('claude-opus-5-5');
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [report, setReport] = useState<AuditReport | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [remote, setRemote] = useState<ModelOption[]>([]);
  const [discovering, setDiscovering] = useState(false);
  const [discoverError, setDiscoverError] = useState<string | null>(null);
  const [manual, setManual] = useState(false);
  const autoRan = useRef(false);

  useEffect(() => {
    document.title = 'Audit an OpenAI-compatible endpoint | RelayRadar';
    api.models().then((r) => setModels(r.items)).catch(() => setModels([]));
  }, []);

  useEffect(() => {
    const id = params.get('report');
    if (!id) return;
    api.auditReport(id).then(setReport).catch((e: Error) => setError(e.message));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (autoRan.current) return;
    if (baseUrl.trim() && !apiKey.trim()) {
      autoRan.current = true;
      void discover();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const discover = async () => {
    if (!baseUrl.trim()) {
      setDiscoverError('Enter a base URL first');
      return;
    }
    setDiscovering(true);
    setDiscoverError(null);
    try {
      const r = await api.auditModels(baseUrl.trim(), apiKey.trim());
      setRemote(r.items);
      setManual(false);
      if (r.items.length && !r.items.some((m) => m.id === model)) {
        setModel(r.items[0].id);
      }
    } catch (err) {
      setDiscoverError((err as Error).message);
      setManual(true);
    } finally {
      setDiscovering(false);
    }
  };

  const run = async (e: React.FormEvent) => {
    e.preventDefault();
    setRunning(true);
    setError(null);
    setReport(null);
    try {
      const r = await api.audit({ baseUrl: baseUrl.trim(), apiKey: apiKey.trim(), model, lang: 'en' });
      setReport(r);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setRunning(false);
    }
  };

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />ZERO-TRUST PROBE</span>
          <h1>Endpoint audit</h1>
          <p className="sub">
            Send two real requests to an OpenAI-compatible endpoint and measure first-token latency, stream integrity,
            model echo, usage consistency and cache passthrough.
          </p>
        </div>
      </div>

      <div className="audit-layout">
        <section className="panel audit-form-panel">
          <div className="panel-head"><h2>Probe settings</h2></div>
          <form className="audit-form" onSubmit={run}>
            <label>
              <span>Base URL</span>
              <input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://api.example.com/v1" required />
            </label>
            <label>
              <span>Temporary API key</span>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                onBlur={() => {
                  if (apiKey.trim().length > 8 && baseUrl.trim() && remote.length === 0 && !discovering) void discover();
                }}
                placeholder="sk-..."
                autoComplete="off"
              />
            </label>
            <label>
              <span className="field-label">
                Model ID
                <button type="button" className="link-btn mono field-action" onClick={discover} disabled={discovering || !baseUrl.trim()}>
                  {discovering ? 'Fetching…' : 'Fetch from endpoint'}
                </button>
              </span>
              {remote.length > 0 && !manual ? (
                <select value={model} onChange={(e) => setModel(e.target.value)} required>
                  {remote.map((m) => <option key={m.id} value={m.id}>{m.id}</option>)}
                </select>
              ) : (
                <input list="en-model-options" value={model} onChange={(e) => setModel(e.target.value)} placeholder="claude-opus-5-5" required />
              )}
              <datalist id="en-model-options">
                {models.map((m) => <option key={m.id} value={m.id} />)}
              </datalist>
              {remote.length > 0 && !manual && <small className="field-hint">Fetched {remote.length} models from /v1/models</small>}
              {discoverError && (
                <small className="field-hint bad">
                  {discoverError}
                  <button type="button" className="link-btn" onClick={() => setManual(true)}>enter manually</button>
                </small>
              )}
            </label>

            <div className="audit-note">
              <b>About the key</b>
              <p>The key is used for this run only, never logged or stored; reports keep the endpoint, model and measurements. Always use a temporary key you can revoke.</p>
            </div>

            <button className="btn btn-primary audit-submit" disabled={running || !model.trim()}>
              {running ? <><Spinner /> Probing…</> : 'Start audit →'}
            </button>
            {error && <p className="form-error">{error}</p>}
          </form>
        </section>

        <section className="panel audit-result-panel">
          <div className="panel-head">
            <h2>Result</h2>
            {report && (
              <div className="report-actions">
                <span className={`verdict ${report.passed ? 'ok' : 'bad'}`}>
                  {report.passed ? 'passed' : 'risk found'} · {report.id}
                </span>
                <button
                  className="link-btn mono"
                  onClick={() => {
                    navigator.clipboard?.writeText(`${window.location.origin}/en/audit?report=${report.id}`);
                    setCopied(true);
                    window.setTimeout(() => setCopied(false), 1600);
                  }}
                >
                  {copied ? 'copied' : 'copy report link'}
                </button>
              </div>
            )}
          </div>

          {!report && !running && (
            <div className="audit-empty">
              <p>The audit checks, in order:</p>
              <ul>
                <li>Endpoint reachability and TLS certificate</li>
                <li>Whether the key is accepted (401/403 flagged separately)</li>
                <li>SSE deltas and the [DONE] terminator</li>
                <li>Whether the response model matches the request</li>
                <li>usage presence and prompt_tokens consistency across two identical calls</li>
                <li>Cache passthrough (cached_tokens)</li>
                <li>First-token latency (TTFT)</li>
              </ul>
              <p className="fine">The audit does not judge whether a model is “dumbed down”; it only checks observable protocol and billing signals.</p>
            </div>
          )}

          {running && <div className="loading-page"><Spinner /> Connecting and reading the first delta…</div>}

          {report && (
            <div className="report">
              <div className="report-meta mono">
                <div><span>Endpoint</span><b>{report.endpoint}</b></div>
                <div><span>Model</span><b>{report.model}</b></div>
                <div><span>TTFT</span><b>{report.ttftMs} ms</b></div>
                <div><span>Duration</span><b>{report.durationMs} ms</b></div>
                <div><span>tokens</span><b>{report.promptTokens} in / {report.completionTokens} out</b></div>
                <div><span>Cache</span><b>{report.cachedTokens || '—'}</b></div>
              </div>
              <ul className="checks">
                {report.checks.map((c) => (
                  <li key={c.key} className={`check check-${c.status}`}>
                    <span className="check-icon">
                      {c.status === 'pass' ? <CheckMark /> : c.status === 'warn' ? <AlertMark /> : <XMark />}
                    </span>
                    <div>
                      <div className="check-head">
                        <b>{c.label}</b>
                        <span className="mono">{STATUS_LABEL[c.status]}</span>
                        <Link className="link-btn tiny" to={`/en/audit/${c.key}`}>details</Link>
                      </div>
                      <p>{c.detail}</p>
                    </div>
                  </li>
                ))}
              </ul>
              <p className="fine">Report {report.id} is archived. Results describe this request only, not every route of the provider.</p>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

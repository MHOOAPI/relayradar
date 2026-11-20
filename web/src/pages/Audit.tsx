import { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { AuditReport, CheckStatus, ModelOption, ModelSummary } from '../types';
import { Spinner } from '../components/ui';
import { AlertMark, CheckMark, XMark } from '../components/icons';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const STATUS_LABEL: Record<CheckStatus, string> = { pass: '通过', warn: '观察', fail: '失败' };

export function AuditPage() {
  const [params] = useSearchParams();
  const [baseUrl, setBaseUrl] = useState(params.get('url') ?? '');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('claude-opus-5-5');
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [report, setReport] = useState<AuditReport | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { push } = useToast();
  const [remote, setRemote] = useState<ModelOption[]>([]);
  const [discovering, setDiscovering] = useState(false);
  const [discoverError, setDiscoverError] = useState<string | null>(null);
  const [manual, setManual] = useState(false);
  const autoRan = useRef(false);

  useEffect(() => {
    api.models().then((r) => setModels(r.items)).catch(() => setModels([]));
  }, []);

  useEffect(() => {
    const id = params.get('report');
    if (!id) return;
    api.auditReport(id).then(setReport).catch((e: Error) => setError(e.message));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Coming from a station page (?url=...): try to list its models right away
  // so the user does not have to type an id.
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
      setDiscoverError('请先填写 Base URL');
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

  const run = async (e: React.FormEvent) => {    e.preventDefault();
    setRunning(true);
    setError(null);
    setReport(null);
    try {
      const r = await api.audit({ baseUrl: baseUrl.trim(), apiKey: apiKey.trim(), model });
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
          <h1>接口检测</h1>
          <p className="sub">
            对一个 OpenAI 兼容端点发两次真实请求，测量首字延迟、流式完整性、模型回显、用量口径与缓存透传。
          </p>
        </div>
      </div>

      <div className="audit-layout">
        <section className="panel audit-form-panel">
          <div className="panel-head"><h2>探测参数</h2></div>
          <form className="audit-form" onSubmit={run}>
            <label>
              <span>Base URL</span>
              <input
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder="https://api.example.com/v1"
                required
              />
            </label>
            <label>
              <span>临时 API Key</span>
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
                模型 ID
                <button
                  type="button"
                  className="link-btn mono field-action"
                  onClick={discover}
                  disabled={discovering || !baseUrl.trim()}
                >
                  {discovering ? '拉取中…' : '从站点拉取'}
                </button>
              </span>
              {remote.length > 0 && !manual ? (
                <select value={model} onChange={(e) => setModel(e.target.value)} required>
                  {remote.map((m) => (
                    <option key={m.id} value={m.id}>{m.id}</option>
                  ))}
                </select>
              ) : (
                <input
                  list="model-options"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="claude-opus-5-5"
                  required
                />
              )}
              <datalist id="model-options">
                {models.map((m) => <option key={m.id} value={m.id} />)}
              </datalist>
              {remote.length > 0 && !manual && (
                <small className="field-hint">已从 /v1/models 拉取 {remote.length} 个模型</small>
              )}
              {discoverError && (
                <small className="field-hint bad">
                  {discoverError}
                  <button type="button" className="link-btn" onClick={() => setManual(true)}>改成手动输入</button>
                </small>
              )}
            </label>

            <div className="audit-note">
              <b>关于 Key</b>
              <p>Key 只用于本次探测，不写日志、不落库；报告只保存端点、模型与测量结果。生产环境请始终使用可随时吊销的临时 Key。</p>
            </div>

            <button className="btn btn-primary audit-submit" disabled={running || !model.trim()}>
              {running ? <><Spinner /> 正在探测…</> : '开始检测 →'}
            </button>
            {error && <p className="form-error">{error}</p>}
          </form>
        </section>

        <section className="panel audit-result-panel">
          <div className="panel-head">
            <h2>检测结果</h2>
            {report && (
              <div className="report-actions">
                <span className={`verdict ${report.passed ? 'ok' : 'bad'}`}>
                  {report.passed ? '通过' : '发现风险'} · {report.id}
                </span>
                <button
                  className="link-btn mono"
                  onClick={() => {
                    void copyText(`${window.location.origin}/audit?report=${report.id}`).then((ok) =>
                      ok ? push('报告链接已复制') : push('复制失败，请手动复制地址栏', 'err'),
                    );
                  }}
                >
                  复制报告链接
                </button>
              </div>
            )}
          </div>

          {!report && !running && (
            <div className="audit-empty">
              <p>检测会依次核对：</p>
              <ul>
                <li>端点可达与 TLS 证书</li>
                <li>鉴权是否被接受（401/403 单独标记）</li>
                <li>SSE 流式增量与 [DONE] 结束标记</li>
                <li>响应回显的模型名是否与请求一致</li>
                <li>usage 是否上报、两次相同请求的 prompt_tokens 是否一致</li>
                <li>缓存命中字段是否透传（cached_tokens）</li>
                <li>首字延迟 TTFT</li>
              </ul>
              <p className="fine">检测不会判断模型是否“降智”，只核对可观测的协议与计费信号。</p>
            </div>
          )}

          {running && <div className="loading-page"><Spinner /> 正在建立连接并读取首个增量…</div>}

          {report && (
            <div className="report">
              <div className="report-meta mono">
                <div><span>端点</span><b>{report.endpoint}</b></div>
                <div><span>模型</span><b>{report.model}</b></div>
                <div><span>TTFT</span><b>{report.ttftMs} ms</b></div>
                <div><span>耗时</span><b>{report.durationMs} ms</b></div>
                <div><span>tokens</span><b>{report.promptTokens} in / {report.completionTokens} out</b></div>
                <div><span>缓存命中</span><b>{report.cachedTokens || '—'}</b></div>
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
                        <Link className="link-btn tiny" to={`/audit/${c.key}`}>说明</Link>
                      </div>
                      <p>{c.detail}</p>
                    </div>
                  </li>
                ))}
              </ul>
              <p className="fine">报告 ID {report.id} 已存档。检测结果只反映本次请求，不代表该站点的全部线路。</p>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

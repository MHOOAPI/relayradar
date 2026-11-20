import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api, ApiError } from '../api';
import type { AuditCheckDoc, AuditReport, CheckStatus, ModelOption, ModelSummary } from '../types';
import { Spinner } from '../components/ui';
import { AlertMark, CheckMark, XMark } from '../components/icons';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const STATUS_LABEL: Record<CheckStatus, string> = { pass: '通过', warn: '观察', fail: '失败' };

// 服务端对 /audit 按 IP 限流（4 秒一次），多模型顺序检测按这个节奏排队。
const AUDIT_GAP_MS = 4200;

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const reportLink = (id: string) => `${window.location.origin}/audit?report=${id}`;

type RunStatus = 'queued' | 'running' | 'done' | 'failed';

interface RunItem {
  model: string;
  status: RunStatus;
  startedAt?: number;
  durationMs?: number;
  error?: string;
}

const STATUS_TEXT: Record<RunStatus, string> = {
  queued: '排队中',
  running: '检测中',
  done: '完成',
  failed: '失败',
};

// 检测流程：真实执行顺序，用来解释「点下去之后发生了什么」。
const RUN_STEPS = [
  ['连接端点', '建立 TLS 连接，记录 HTTP 状态与握手耗时'],
  ['连发两次相同请求', '第一次带 stream=true，第二次用于比对分词与缓存'],
  ['读取流式响应', '记录首个增量时间、[DONE] 结束标记与回显的模型名'],
  ['逐项判定并存档', '8 项检查给出通过 / 观察 / 失败，报告可分享'],
];

function CheckIntro({ docs }: { docs: AuditCheckDoc[] }) {
  return (
    <section className="panel audit-checks">
      <div className="panel-head">
        <h2>检测什么</h2>
        <span className="fine">{docs.length > 0 ? `${docs.length} 项协议与计费检查，每项给出通过 / 观察 / 失败` : '加载检查项…'}</span>
      </div>
      <div className="audit-check-grid">
        {docs.map((c, i) => (
          <Link key={c.key} className="audit-check-card" to={`/audit/${c.key}`}>
            <span className="ac-head">
              <span className="mono dim ac-index">{String(i + 1).padStart(2, '0')}</span>
              <b>{c.label}</b>
            </span>
            <p>{c.short}</p>
            <span className="ac-pass">通过：{c.pass}</span>
          </Link>
        ))}
      </div>
    </section>
  );
}

function RunSteps() {
  return (
    <ol className="run-steps">
      {RUN_STEPS.map(([title, detail], i) => (
        <li key={title}>
          <span className="mono run-step-index">{i + 1}</span>
          <div>
            <b>{title}</b>
            <p>{detail}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}

function ResultMatrix({ reports }: { reports: AuditReport[] }) {
  const keys = reports[0].checks.map((c) => ({ key: c.key, label: c.label }));
  const mark = (status?: CheckStatus) => (status === 'pass' ? '✓' : status === 'warn' ? '!' : status === 'fail' ? '✕' : '·');
  return (
    <div className="result-matrix-wrap">
      <table className="index-table result-matrix">
        <thead>
          <tr>
            <th>检查项</th>
            {reports.map((r) => (
              <th key={r.id} className="mono matrix-model" title={r.model}>{r.model}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {keys.map((k) => (
            <tr key={k.key}>
              <td><Link to={`/audit/${k.key}`}>{k.label}</Link></td>
              {reports.map((r) => {
                const check = r.checks.find((c) => c.key === k.key);
                const status = check?.status;
                return (
                  <td key={r.id} className={`matrix-cell ${status ?? 'unknown'}`} title={check?.detail}>
                    {mark(status)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ReportCard({ report, defaultOpen }: { report: AuditReport; defaultOpen: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  const { push } = useToast();

  return (
    <div className="report-card">
      <button type="button" className="report-card-head" onClick={() => setOpen((v) => !v)} aria-expanded={open}>
        <span className="mono r-model">{report.model}</span>
        <span className={`verdict ${report.passed ? 'ok' : 'bad'}`}>{report.passed ? '通过' : '发现风险'}</span>
        <span className="mono dim r-metric">{report.ttftMs} ms</span>
        <span className="mono dim r-metric hide-md">{report.durationMs} ms</span>
        <span className="r-toggle mono dim">{open ? '收起明细' : '查看明细'}</span>
      </button>
      {open && (
        <div className="report-card-body">
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
          <p className="fine">
            报告 ID {report.id} 已存档。
            <button
              type="button"
              className="link-btn mono"
              onClick={() => {
                void copyText(reportLink(report.id)).then((ok) =>
                  ok ? push('报告链接已复制') : push('复制失败，请手动复制地址栏', 'err'),
                );
              }}
            >
              复制报告链接
            </button>
          </p>
        </div>
      )}
    </div>
  );
}

export function AuditPage() {
  const [params] = useSearchParams();
  const [baseUrl, setBaseUrl] = useState(params.get('url') ?? '');
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('claude-opus-5-5');
  const [models, setModels] = useState<ModelSummary[]>([]);
  const [checkDocs, setCheckDocs] = useState<AuditCheckDoc[]>([]);
  const [reports, setReports] = useState<AuditReport[]>([]);
  const [failures, setFailures] = useState<Array<{ model: string; error: string }>>([]);
  const [runItems, setRunItems] = useState<RunItem[]>([]);
  const [now, setNow] = useState(() => Date.now());
  const [running, setRunning] = useState(false);
  const [stopped, setStopped] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { push } = useToast();
  const [remote, setRemote] = useState<ModelOption[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [filter, setFilter] = useState('');
  const [discovering, setDiscovering] = useState(false);
  const [discoverError, setDiscoverError] = useState<string | null>(null);
  const [manual, setManual] = useState(false);
  const autoRan = useRef(false);
  const lastAuditAt = useRef(0);
  const stopRef = useRef(false);

  useEffect(() => {
    api.models().then((r) => setModels(r.items)).catch(() => setModels([]));
    api.auditChecks().then((r) => setCheckDocs(r.items)).catch(() => setCheckDocs([]));
  }, []);

  useEffect(() => {
    if (!running) return;
    const timer = window.setInterval(() => setNow(Date.now()), 500);
    return () => window.clearInterval(timer);
  }, [running]);

  useEffect(() => {
    const id = params.get('report');
    if (!id) return;
    api.auditReport(id).then((r) => setReports([r])).catch((e: Error) => setError(e.message));
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
      setFilter('');
      const first = r.items[0]?.id;
      setSelected(first ? [first] : []);
      if (first) setModel(first);
    } catch (err) {
      setDiscoverError((err as Error).message);
      setManual(true);
    } finally {
      setDiscovering(false);
    }
  };

  const usingPicker = remote.length > 0 && !manual;
  const targets = usingPicker ? selected : [model.trim()].filter(Boolean);

  const filteredRemote = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    return needle ? remote.filter((m) => m.id.toLowerCase().includes(needle)) : remote;
  }, [remote, filter]);

  const toggle = (id: string) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  };

  const allFilteredSelected = filteredRemote.length > 0 && filteredRemote.every((m) => selected.includes(m.id));

  const toggleAllFiltered = () => {
    if (allFilteredSelected) {
      setSelected(selected.filter((id) => !filteredRemote.some((m) => m.id === id)));
      return;
    }
    setSelected([...new Set([...selected, ...filteredRemote.map((m) => m.id)])]);
  };

  const run = async (e: React.FormEvent) => {
    e.preventDefault();
    if (targets.length === 0) {
      setError(usingPicker ? '请先从列表中选择至少一个模型' : '请填写模型 ID');
      return;
    }
    setRunning(true);
    setStopped(false);
    stopRef.current = false;
    setError(null);
    setReports([]);
    setFailures([]);
    setRunItems(targets.map((m) => ({ model: m, status: 'queued' })));
    const acc: AuditReport[] = [];
    const fails: Array<{ model: string; error: string }> = [];

    for (let i = 0; i < targets.length; i++) {
      if (stopRef.current) {
        setStopped(true);
        break;
      }
      const target = targets[i];
      const startedAt = Date.now();
      setRunItems((prev) => prev.map((it, idx) => (idx === i ? { ...it, status: 'running', startedAt } : it)));
      if (i > 0) {
        const wait = AUDIT_GAP_MS - (Date.now() - lastAuditAt.current);
        if (wait > 0) await sleep(wait);
      }
      const probe = () => {
        lastAuditAt.current = Date.now();
        return api.audit({ baseUrl: baseUrl.trim(), apiKey: apiKey.trim(), model: target });
      };
      try {
        let report: AuditReport;
        try {
          report = await probe();
        } catch (err) {
          if (err instanceof ApiError && err.status === 429) {
            await sleep(AUDIT_GAP_MS + 800);
            report = await probe();
          } else {
            throw err;
          }
        }
        acc.push(report);
        setReports([...acc]);
        setRunItems((prev) =>
          prev.map((it, idx) => (idx === i ? { ...it, status: 'done', durationMs: Date.now() - startedAt } : it)),
        );
      } catch (err) {
        const message = (err as Error).message;
        fails.push({ model: target, error: message });
        setFailures([...fails]);
        setRunItems((prev) =>
          prev.map((it, idx) => (idx === i ? { ...it, status: 'failed', durationMs: Date.now() - startedAt, error: message } : it)),
        );
      }
    }

    setRunning(false);
  };

  const passed = reports.filter((r) => r.passed).length;
  const risky = reports.length - passed;
  const finished = runItems.filter((it) => it.status === 'done' || it.status === 'failed').length;

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />ZERO-TRUST PROBE</span>
          <h1>接口检测</h1>
          <p className="sub">
            对任意 OpenAI 兼容端点发两次真实请求，核对 8 项协议与计费信号：端点可达、鉴权、流式完整性、模型回显、usage、分词一致性、缓存透传与首字延迟。
            从站点拉取模型后可以多选，逐个检测并各自生成报告。
          </p>
        </div>
      </div>

      <CheckIntro docs={checkDocs} />

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

            <div className="audit-field">
              <span className="field-label">
                模型{usingPicker ? `（可多选，已选 ${selected.length}/${remote.length}）` : ' ID'}
                <button
                  type="button"
                  className="link-btn mono field-action"
                  onClick={discover}
                  disabled={discovering || !baseUrl.trim()}
                >
                  {discovering ? '拉取中…' : remote.length > 0 ? '重新拉取' : '从站点拉取'}
                </button>
              </span>

              {usingPicker ? (
                <div className="model-picker">
                  <div className="model-picker-head">
                    <input
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      placeholder="筛选模型 ID"
                      aria-label="筛选模型"
                    />
                    <button type="button" className="link-btn mono" onClick={toggleAllFiltered}>
                      {allFilteredSelected ? '取消全选' : '全选'}
                    </button>
                  </div>
                  <div className="model-picker-list">
                    {filteredRemote.map((m) => (
                      <label key={m.id} className={`model-row ${selected.includes(m.id) ? 'on' : ''}`}>
                        <input
                          type="checkbox"
                          checked={selected.includes(m.id)}
                          onChange={() => toggle(m.id)}
                        />
                        <span className="mono">{m.id}</span>
                      </label>
                    ))}
                    {filteredRemote.length === 0 && <p className="fine model-picker-empty">没有匹配的模型</p>}
                  </div>
                  <div className="model-picker-foot mono">
                    <span>已选 {selected.length} / {remote.length} 个模型</span>
                    {selected.length > 0 && (
                      <button type="button" className="link-btn" onClick={() => setSelected([])}>清空</button>
                    )}
                  </div>
                </div>
              ) : (
                <>
                  <input
                    list="model-options"
                    value={model}
                    onChange={(e) => setModel(e.target.value)}
                    placeholder="claude-opus-5-5"
                  />
                  <datalist id="model-options">
                    {models.map((m) => <option key={m.id} value={m.id} />)}
                  </datalist>
                </>
              )}

              {usingPicker && (
                <small className="field-hint">
                  已从 /v1/models 拉取 {remote.length} 个模型，可全选；每个模型约 4–8 秒，按服务端限流排队。
                </small>
              )}
              {discoverError && (
                <small className="field-hint bad">
                  {discoverError}
                  <button type="button" className="link-btn" onClick={() => setManual(true)}>改成手动输入</button>
                </small>
              )}
              {remote.length > 0 && manual && (
                <small className="field-hint">
                  手动输入模式
                  <button type="button" className="link-btn" onClick={() => setManual(false)}>回到列表</button>
                </small>
              )}
            </div>

            <div className="audit-note">
              <b>关于 Key</b>
              <p>Key 只用于本次探测，不写日志、不落库；报告只保存端点、模型与测量结果。生产环境请始终使用可随时吊销的临时 Key。</p>
            </div>

            <button className="btn btn-primary audit-submit" disabled={running || targets.length === 0}>
              {running
                ? <><Spinner /> {finished}/{runItems.length || targets.length} 已完成</>
                : targets.length > 1 ? `开始检测 ${targets.length} 个模型 →` : '开始检测 →'}
            </button>
            {running && (
              <button
                type="button"
                className="btn btn-ghost audit-stop"
                onClick={() => { stopRef.current = true; }}
                disabled={stopped}
              >
                {stopped ? '正在停止…' : '停止检测'}
              </button>
            )}
            {!running && targets.length > 2 && (
              <p className="fine audit-estimate">
                {targets.length} 个模型将按顺序检测，预计约 {targets.length <= 10
                  ? `${targets.length * 6} 秒`
                  : `${Math.max(1, Math.round((targets.length * 6) / 60))} 分钟`}（每 4 秒一个，可随时停止）
              </p>
            )}
            {error && <p className="form-error">{error}</p>}
          </form>
        </section>

        <section className="panel audit-result-panel">
          <div className="panel-head">
            <h2>检测结果</h2>
            {(reports.length > 0 || failures.length > 0) && !running && (
              <div className="report-actions">
                <span className="mono dim result-count">
                  共 {reports.length + failures.length} 个 · 通过 {passed} · 风险 {risky}
                  {failures.length > 0 ? ` · 失败 ${failures.length}` : ''}
                </span>
                {reports.length > 1 && (
                  <button
                    type="button"
                    className="link-btn mono"
                    onClick={() => {
                      void copyText(reports.map((r) => reportLink(r.id)).join('\n')).then((ok) =>
                        ok ? push(`已复制 ${reports.length} 个报告链接`) : push('复制失败，请手动复制地址栏', 'err'),
                      );
                    }}
                  >
                    复制全部链接
                  </button>
                )}
              </div>
            )}
          </div>

          {running && (
            <div className="run-panel">
              <div className="run-summary">
                <Spinner />
                <span>正在检测 {Math.min(finished + 1, runItems.length || targets.length)}/{runItems.length || targets.length}</span>
                <div className="run-bar" aria-hidden="true">
                  <i style={{ width: `${runItems.length ? (finished / runItems.length) * 100 : 0}%` }} />
                </div>
              </div>
              <ul className="run-list">
                {runItems.map((it) => (
                  <li key={it.model} className={`run-item ${it.status}`}>
                    <span className="run-dot" />
                    <span className="mono run-model">{it.model}</span>
                    <span className="run-status">
                      {STATUS_TEXT[it.status]}
                      {it.status === 'running' && it.startedAt ? ` · ${((now - it.startedAt) / 1000).toFixed(1)}s` : ''}
                      {it.status === 'done' && it.durationMs ? ` · ${(it.durationMs / 1000).toFixed(1)}s` : ''}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {!running && reports.length === 0 && failures.length === 0 && (
            <div className="audit-empty">
              <h3>检测过程</h3>
              <RunSteps />
              <p className="fine">点「开始检测」后，这里会按模型显示进度，并在结束后给出逐项结果与对比矩阵。</p>
            </div>
          )}

          {(reports.length > 0 || failures.length > 0) && (
            <div className="report-list">
              {stopped && (
                <p className="fine audit-stopped">
                  已停止：完成 {finished} / {runItems.length} 个模型，剩下 {Math.max(0, runItems.length - finished)} 个未检测，重新开始会从头跑。
                </p>
              )}
              {reports.length > 1 && (
                <div className="matrix-block">
                  <h3>检查项 × 模型</h3>
                  <ResultMatrix reports={reports} />
                </div>
              )}
              {reports.map((r, i) => (
                <ReportCard key={r.id} report={r} defaultOpen={reports.length === 1 || i === 0} />
              ))}
              {failures.map((f) => (
                <div key={f.model} className="report-fail">
                  <b className="mono">{f.model}</b>
                  <span>{f.error}</span>
                </div>
              ))}
            </div>
          )}

          {reports.length > 0 && (
            <p className="fine report-foot">
              检测结果只反映本次请求，不代表该站点的全部线路；报告链接可以分享，但不含 Key。
            </p>
          )}
        </section>
      </div>
    </main>
  );
}

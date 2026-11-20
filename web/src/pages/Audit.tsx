import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api, ApiError } from '../api';
import type { AuditReport, CheckStatus, ModelOption, ModelSummary } from '../types';
import { Spinner } from '../components/ui';
import { AlertMark, CheckMark, XMark } from '../components/icons';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const STATUS_LABEL: Record<CheckStatus, string> = { pass: '通过', warn: '观察', fail: '失败' };

// 单次最多检测几个模型：每个模型要发两次真实请求，太多会拖很久。
const MAX_MODELS = 8;
// 服务端对 /audit 按 IP 限流（4 秒一次），多模型顺序检测按这个节奏排队。
const AUDIT_GAP_MS = 4200;

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const reportLink = (id: string) => `${window.location.origin}/audit?report=${id}`;

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
  const [reports, setReports] = useState<AuditReport[]>([]);
  const [failures, setFailures] = useState<Array<{ model: string; error: string }>>([]);
  const [progress, setProgress] = useState<{ done: number; total: number; current: string } | null>(null);
  const [running, setRunning] = useState(false);
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

  useEffect(() => {
    api.models().then((r) => setModels(r.items)).catch(() => setModels([]));
  }, []);

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
    if (selected.includes(id)) {
      setSelected(selected.filter((x) => x !== id));
      return;
    }
    if (selected.length >= MAX_MODELS) {
      push(`单次最多检测 ${MAX_MODELS} 个模型`, 'err');
      return;
    }
    setSelected([...selected, id]);
  };

  const allFilteredSelected = filteredRemote.length > 0 && filteredRemote.every((m) => selected.includes(m.id));

  const toggleAllFiltered = () => {
    if (allFilteredSelected) {
      setSelected(selected.filter((id) => !filteredRemote.some((m) => m.id === id)));
      return;
    }
    const merged = [...selected];
    for (const m of filteredRemote) {
      if (merged.length >= MAX_MODELS) break;
      if (!merged.includes(m.id)) merged.push(m.id);
    }
    if (merged.length >= MAX_MODELS && filteredRemote.some((m) => !merged.includes(m.id))) {
      push(`单次最多检测 ${MAX_MODELS} 个模型`, 'err');
    }
    setSelected(merged);
  };

  const run = async (e: React.FormEvent) => {
    e.preventDefault();
    if (targets.length === 0) {
      setError(usingPicker ? '请先从列表中选择至少一个模型' : '请填写模型 ID');
      return;
    }
    setRunning(true);
    setError(null);
    setReports([]);
    setFailures([]);
    const acc: AuditReport[] = [];
    const fails: Array<{ model: string; error: string }> = [];

    for (let i = 0; i < targets.length; i++) {
      const target = targets[i];
      setProgress({ done: i, total: targets.length, current: target });
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
      } catch (err) {
        fails.push({ model: target, error: (err as Error).message });
        setFailures([...fails]);
      }
    }

    setProgress(null);
    setRunning(false);
  };

  const passed = reports.filter((r) => r.passed).length;
  const risky = reports.length - passed;

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />ZERO-TRUST PROBE</span>
          <h1>接口检测</h1>
          <p className="sub">
            对一个 OpenAI 兼容端点发两次真实请求，测量首字延迟、流式完整性、模型回显、用量口径与缓存透传。从站点拉取模型后可以多选，逐个检测。
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
                  已从 /v1/models 拉取 {remote.length} 个模型，单次最多检测 {MAX_MODELS} 个。
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
                ? <><Spinner /> {progress ? `正在检测 ${progress.done + 1}/${progress.total}` : '正在探测…'}</>
                : targets.length > 1 ? `开始检测 ${targets.length} 个模型 →` : '开始检测 →'}
            </button>
            {error && <p className="form-error">{error}</p>}
          </form>
        </section>

        <section className="panel audit-result-panel">
          <div className="panel-head">
            <h2>检测结果</h2>
            {reports.length + failures.length > 0 && (
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

          {reports.length === 0 && failures.length === 0 && !running && (
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

          {running && (
            <div className="run-progress">
              <Spinner />
              <span>
                {progress ? `正在检测 ${progress.done + 1}/${progress.total}：` : '正在建立连接…'}
                <b className="mono">{progress?.current ?? ''}</b>
              </span>
              {progress && progress.done > 0 && <span className="fine">每个模型约 4 秒排队，避免触发限流。</span>}
            </div>
          )}

          {(reports.length > 0 || failures.length > 0) && (
            <div className="report-list">
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
            <p className="fine report-foot">检测结果只反映本次请求，不代表该站点的全部线路；报告链接可以分享，但不含 Key。</p>
          )}
        </section>
      </div>
    </main>
  );
}

import { Fragment, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { AdminSummary, Submission } from '../types';
import { Spinner } from '../components/ui';
import { useAuth } from '../lib/auth';

type Sub = { id: string; channel: string; url: string; email: string; enabled: boolean; attempts: number; lastError: string; createdAt: string; lastSentAt: string };
type Review = { id: number; stationSlug: string; stationName: string; author: string; rating: number; body: string; createdAt: string };
type AdminAction = { id: number; actorId: number; actorEmail: string; action: string; target: string; detail: string; ip: string; createdAt: string };
type Incident = { id: number; title: string; status: string; component: string; body: string; startedAt: string; resolvedAt?: string };
type ClaimEditRow = { id: number; stationSlug: string; stationName: string; tagline: string; refundPolicy: string; invoice: boolean; minTopupCny: number; status: string; createdAt: string };

const STATUS_LABEL: Record<string, string> = { pending: '待审核', approved: '已通过', rejected: '已拒绝' };
const PAY_OPTIONS = ['支付宝', '微信', 'USDT', 'Stripe', '信用卡'];

export function AdminPage() {
  const { user, loading } = useAuth();
  const [summary, setSummary] = useState<AdminSummary | null>(null);
  const [subs, setSubs] = useState<Submission[]>([]);
  const [alerts, setAlerts] = useState<Sub[]>([]);
  const [users, setUsers] = useState<import('../types').User[]>([]);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [audit, setAudit] = useState<AdminAction[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [edits, setEdits] = useState<ClaimEditRow[]>([]);
  const [showAllEdits, setShowAllEdits] = useState(false);
  const [auditQ, setAuditQ] = useState('');
  const [auditLimit, setAuditLimit] = useState(100);
  const [probeText, setProbeText] = useState('');
  const [probeSaved, setProbeSaved] = useState<string | null>(null);
  const [tab, setTab] = useState<'overview' | 'submissions' | 'alerts' | 'users' | 'reviews' | 'audit' | 'incidents' | 'claims' | 'probes'>('overview');
  const [error, setError] = useState<string | null>(null);
  const [snapMsg, setSnapMsg] = useState('');
  const [snapBusy, setSnapBusy] = useState(false);

  const load = () => {
    api.adminProbeConfig().then((r) => setProbeText(r.config)).catch(() => setProbeText(''));
    Promise.all([api.adminSummary(), api.adminSubmissions(), api.adminSubscriptions(), api.adminUsers(), api.adminReviews(), api.adminAudit(auditLimit), api.adminIncidents(), api.adminClaimEdits(showAllEdits)])
      .then(([s, sub, a, u, r, log, inc, ce]) => {
        setSummary(s);
        setSubs(sub.items);
        setAlerts(a.items);
        setUsers(u.items);
        setReviews(r.items);
        setAudit(log.items);
        setIncidents(inc.items);
        setEdits(ce.items);
      })
      .catch((e: Error) => setError(e.message));
  };

  useEffect(() => {
    if (user?.role === 'admin') load();
  }, [user]);

  useEffect(() => {
    if (user?.role === 'admin') load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showAllEdits]);

  if (loading) return <main className="wrap"><div className="loading-page"><Spinner /> 读取账号…</div></main>;
  if (!user) {
    return (
      <main className="wrap"><div className="empty-panel" style={{ padding: '60px 24px', marginTop: 40 }}>
        <b>需要登录</b><p>后台仅对管理员开放。</p><Link className="btn btn-primary btn-sm" to="/login">去登录</Link>
      </div></main>
    );
  }
  if (user.role !== 'admin') {
    return <main className="wrap"><div className="error-box">当前账号没有管理员权限。</div></main>;
  }

  const setStatus = async (id: number, status: string) => {
    try {
      await api.adminSetSubmission(id, status);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const [editId, setEditId] = useState<number | null>(null);
  const [savedId, setSavedId] = useState<number | null>(null);
  const [draft, setDraft] = useState({ payments: '', invoice: false, refundPolicy: '' });
  const [editError, setEditError] = useState<string | null>(null);

  const startEdit = (s: Submission) => {
    setEditId(s.id);
    setSavedId(null);
    setEditError(null);
    setDraft({ payments: (s.payments ?? []).join(','), invoice: s.invoice ?? false, refundPolicy: s.refundPolicy ?? '' });
  };

  const saveTerms = async (s: Submission) => {
    setEditError(null);
    const payments = draft.payments.split(',').map((x) => x.trim()).filter(Boolean);
    if (payments.length === 0) {
      setEditError('请至少选择一种支付方式');
      return;
    }
    if (draft.refundPolicy.length > 200) {
      setEditError('退款政策不能超过 200 字');
      return;
    }
    try {
      await api.adminSetSubmission(s.id, s.status, {
        payments,
        invoice: draft.invoice,
        refundPolicy: draft.refundPolicy,
      });
      setEditId(null);
      setSavedId(s.id);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const delReview = async (id: number) => {
    if (!window.confirm('删除这条评价？')) return;
    try {
      await api.adminDeleteReview(id);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const setRole = async (id: number, role: 'admin' | 'user') => {
    try {
      await api.adminSetUserRole(id, role);
      load();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const [incTitle, setIncTitle] = useState('');
  const [incBody, setIncBody] = useState('');
  const [incComp, setIncComp] = useState('api');

  const createIncident = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!incTitle.trim()) return;
    try {
      await api.createIncident({ title: incTitle.trim(), component: incComp, body: incBody.trim(), status: 'investigating' });
      setIncTitle('');
      setIncBody('');
      load();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const resolveIncident = async (inc: Incident) => {
    try {
      await api.updateIncident(inc.id, { status: 'resolved', body: inc.body });
      load();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const removeIncident = async (id: number) => {
    if (!window.confirm('删除这条事故记录？')) return;
    try {
      await api.deleteIncident(id);
      load();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const [iconMsg, setIconMsg] = useState<Record<string, string>>({});

  const squareIcon = async (file: File): Promise<Blob | null> => {
    const bitmap = await createImageBitmap(file);
    const side = Math.min(bitmap.width, bitmap.height);
    const canvas = document.createElement('canvas');
    canvas.width = 256;
    canvas.height = 256;
    const ctx = canvas.getContext('2d');
    if (!ctx) return null;
    ctx.drawImage(bitmap, (bitmap.width - side) / 2, (bitmap.height - side) / 2, side, side, 0, 0, 256, 256);
    return new Promise((resolve) => canvas.toBlob((b) => resolve(b), 'image/png'));
  };

  const uploadIcon = async (slug: string, file: File | undefined) => {
    if (!file) return;
    try {
      const square = await squareIcon(file);
      if (!square) throw new Error('无法处理该图片');
      await api.adminUploadIcon(slug, square);
      setIconMsg((m) => ({ ...m, [slug]: '已更新' }));
      load();
    } catch (e) {
      setIconMsg((m) => ({ ...m, [slug]: (e as Error).message }));
    }
  };

  const resetIcon = async (slug: string) => {
    try {
      await api.adminResetIcon(slug);
      setIconMsg((m) => ({ ...m, [slug]: '已恢复默认' }));
      load();
    } catch (e) {
      setIconMsg((m) => ({ ...m, [slug]: (e as Error).message }));
    }
  };

  const decideEdit = async (id: number, approve: boolean) => {
    const note = approve ? '' : (window.prompt('拒绝理由（会发给站长，可留空）') ?? '');
    try {
      if (approve) await api.approveClaimEdit(id, note);
      else await api.rejectClaimEdit(id, note);
      load();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>后台</h1>
          <p className="sub">审核提交、查看告警订阅与数据概览。</p>
        </div>
        <span className="mono dim">{user.email}</span>
      </div>

      {error && <div className="error-box">{error}</div>}

      <div className="tabs">
        <button className={`tab ${tab === 'overview' ? 'on' : ''}`} onClick={() => setTab('overview')}>概览</button>
        <button className={`tab ${tab === 'submissions' ? 'on' : ''}`} onClick={() => setTab('submissions')}>提交审核（{subs.filter((s) => s.status !== 'approved' && s.status !== 'rejected').length}）</button>
        <button className={`tab ${tab === 'alerts' ? 'on' : ''}`} onClick={() => setTab('alerts')}>
          告警订阅（{alerts.length}）
          {alerts.filter((a) => a.attempts >= 3).length > 0 && <span className="tab-bad"> · {alerts.filter((a) => a.attempts >= 3).length} 失败</span>}
        </button>
        <button className={`tab ${tab === 'users' ? 'on' : ''}`} onClick={() => setTab('users')}>用户（{users.length}）</button>
        <button className={`tab ${tab === 'reviews' ? 'on' : ''}`} onClick={() => setTab('reviews')}>评价（{reviews.length}）</button>
        <button className={`tab ${tab === 'audit' ? 'on' : ''}`} onClick={() => setTab('audit')}>操作日志（{audit.length}）</button>
        <button className={`tab ${tab === 'incidents' ? 'on' : ''}`} onClick={() => setTab('incidents')}>事故（{incidents.length}）</button>
        <button className={`tab ${tab === 'claims' ? 'on' : ''}`} onClick={() => setTab('claims')}>认领修改（{edits.length}）</button>
        <button className={`tab ${tab === 'probes' ? 'on' : ''}`} onClick={() => setTab('probes')}>真实探针</button>
      </div>

      {tab === 'overview' && summary && (
        <>
          <div className="admin-grid">
            {[
              ['站点', summary.counts.stations],
              ['分组', summary.counts.groups],
              ['模型', summary.counts.models],
              ['探针', summary.counts.probes],
              ['提交', summary.submissions],
              ['待审核', summary.pendingSubmissions],
              ['已认领', summary.claimedStations],
              ['待审修改', summary.pendingClaimEdits],
              ['探针过期', summary.staleProbes],
              ['订阅', summary.subscriptions],
              ['评价', summary.counts.reviews],
            ].map(([label, value]) => (
              <div className="panel admin-stat" key={String(label)}>
                <span>{label}</span><b className="mono">{value ?? '—'}</b>
              </div>
            ))}
          </div>
          <p className="fine">数据时间 {new Date(summary.updatedAt).toLocaleString('zh-CN', { hour12: false })}</p>
          <div className="panel admin-snapshot">
            <div>
              <h2>价格快照</h2>
              <p className="sub">立即把当前分组价格写入 price_daily，用于补跑或刚导入数据后刷新。</p>
            </div>
            <div className="admin-snapshot-actions">
              {snapMsg && <span className="mono dim">{snapMsg}</span>}
              <button
                className="btn btn-ghost btn-sm"
                disabled={snapBusy}
                onClick={async () => {
                  setSnapBusy(true);
                  setSnapMsg('');
                  try {
                    const r = await api.adminRunPriceSnapshot();
                    setSnapMsg(`写入 ${r.rows} 行，耗时 ${r.durationMs}ms`);
                  } catch (e) {
                    setSnapMsg((e as Error).message);
                  } finally {
                    setSnapBusy(false);
                  }
                }}
              >
                {snapBusy ? '运行中…' : '立即运行快照'}
              </button>
            </div>
          </div>
          {summary.failingSubscriptions > 0 && (
            <div className="error-box" style={{ marginTop: 12 }}>
              有 {summary.failingSubscriptions} 个告警订阅连续失败 ≥3 次，已发送失败提醒；去「告警订阅」查看并重放。
            </div>
          )}
        </>
      )}

      {tab === 'submissions' && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th className="num">#</th><th>名称</th><th>域名</th><th>状态</th><th>时间</th><th>操作</th></tr></thead>
              <tbody>
                {subs.length === 0 && <tr><td colSpan={6}><span className="fine">暂无提交</span></td></tr>}
                {subs.map((s) => (
                  <Fragment key={s.id}>
                  <tr>
                    <td className="num mono dim">{s.id}</td>
                    <td>{s.name}</td>
                    <td className="mono dim">{s.domain}</td>
                    <td><span className={`status-pill ${s.status}`}>{STATUS_LABEL[s.status] ?? s.status}</span></td>
                    <td className="mono dim">{new Date(s.createdAt).toLocaleDateString('zh-CN')}</td>
                    <td>
                      <button className="link-btn" onClick={() => startEdit(s)}>商务信息</button>
                      <button className="link-btn" onClick={() => setStatus(s.id, 'approved')}>通过</button>
                      <button className="link-btn" onClick={() => setStatus(s.id, 'rejected')}>拒绝</button>
                      {savedId === s.id && <span className="fine" style={{ marginLeft: 6 }}>已保存</span>}
                    </td>
                  </tr>
                  {editId === s.id && (
                    <tr key={`edit-${s.id}`} className="term-edit-row">
                      <td colSpan={6}>
                        <div className="term-edit">
                          <label className="term-payments"><span>支付方式</span>
                            <div className="term-chips">
                              {PAY_OPTIONS.map((p) => {
                                const on = draft.payments.split(',').map((x) => x.trim()).includes(p);
                                return (
                                  <button
                                    key={p}
                                    type="button"
                                    className={`term-chip ${on ? 'on' : ''}`}
                                    onClick={() => {
                                      const list = draft.payments.split(',').map((x) => x.trim()).filter(Boolean);
                                      const next = on ? list.filter((x) => x !== p) : [...list, p];
                                      setDraft({ ...draft, payments: next.join(',') });
                                    }}
                                  >
                                    {p}
                                  </button>
                                );
                              })}
                            </div>
                          </label>
                          <label className="term-check"><input type="checkbox" checked={draft.invoice} onChange={(e) => setDraft({ ...draft, invoice: e.target.checked })} /> 支持开票</label>
                          <label><span>退款政策</span><input value={draft.refundPolicy} onChange={(e) => setDraft({ ...draft, refundPolicy: e.target.value })} placeholder="例如 7 天未用可退" /></label>
                          <div className="term-actions">
                            <button className="btn btn-sm btn-primary" onClick={() => saveTerms(s)}>保存</button>
                            <button className="btn btn-sm" onClick={() => setEditId(null)}>取消</button>
                          </div>
                          {editError && <span className="term-error">{editError}</span>}
                        </div>
                      </td>
                    </tr>
                  )}
                  </Fragment>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {tab === 'alerts' && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th>ID</th><th>通道</th><th>目标</th><th>状态</th><th className="num">失败</th><th>最近错误</th></tr></thead>
              <tbody>
                {alerts.length === 0 && <tr><td colSpan={6}><span className="fine">暂无订阅</span></td></tr>}
                {alerts.map((a) => (
                  <tr key={a.id}>
                    <td className="mono dim">{a.id}</td>
                    <td>{a.channel === 'email' ? '邮件' : 'Webhook'}</td>
                    <td className="mono dim">{a.channel === 'email' ? a.email : a.url}</td>
                    <td>{a.enabled ? '投递中' : '已暂停'}</td>
                    <td className="num mono">{a.attempts}</td>
                    <td className="fine">{a.lastError || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
      {tab === 'users' && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th className="num">ID</th><th>邮箱</th><th>昵称</th><th>角色</th><th>注册时间</th><th>操作</th></tr></thead>
              <tbody>
                {users.length === 0 && <tr><td colSpan={6}><span className="fine">暂无用户</span></td></tr>}
                {users.map((u) => (
                  <tr key={u.id}>
                    <td className="num mono dim">{u.id}</td>
                    <td className="mono">{u.email}</td>
                    <td>{u.name}</td>
                    <td><span className={`status-pill ${u.role === 'admin' ? 'approved' : 'pending'}`}>{u.role}</span></td>
                    <td className="mono dim">{new Date(u.createdAt).toLocaleString('zh-CN', { hour12: false })}</td>
                    <td>
                      {u.role === 'admin' ? (
                        <button className="link-btn" onClick={() => setRole(u.id, 'user')} disabled={u.id === user.id}>取消管理员</button>
                      ) : (
                        <button className="link-btn" onClick={() => setRole(u.id, 'admin')}>设为管理员</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {tab === 'reviews' && (
        <section className="panel">
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th className="num">ID</th><th>站点</th><th>作者</th><th className="num">评分</th><th>内容</th><th>时间</th><th>操作</th></tr></thead>
              <tbody>
                {reviews.length === 0 && <tr><td colSpan={7}><span className="fine">暂无评价</span></td></tr>}
                {reviews.map((r) => (
                  <tr key={r.id}>
                    <td className="num mono dim">{r.id}</td>
                    <td><Link className="station-name" to={`/stations/${r.stationSlug}`}>{r.stationName}</Link></td>
                    <td>{r.author}</td>
                    <td className="num mono">{r.rating}</td>
                    <td className="review-body">{r.body}</td>
                    <td className="mono dim">{new Date(r.createdAt).toLocaleDateString('zh-CN')}</td>
                    <td><button className="link-btn" onClick={() => delReview(r.id)}>删除</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
      {tab === 'audit' && (
        <section className="panel">
          <div className="panel-head">
            <div><h2>操作日志</h2><p className="sub">按站点、操作者或动作过滤。</p></div>
            <input className="mp-search" style={{ width: 220 }} value={auditQ} onChange={(e) => setAuditQ(e.target.value)} placeholder="过滤：station:slug / 邮箱 / 动作" />
            <button className="btn btn-ghost btn-sm" onClick={() => setAuditLimit((n) => Math.min(500, n + 100))}>加载更多</button>
          </div>
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>对象</th><th>详情</th><th className="hide-sm">IP</th></tr></thead>
              <tbody>
                {audit.length === 0 && <tr><td colSpan={6}><span className="fine">暂无操作记录</span></td></tr>}
                {audit.filter((a) => !auditQ || `${a.actorEmail} ${a.action} ${a.target} ${a.detail}`.toLowerCase().includes(auditQ.toLowerCase())).map((a) => (
                  <tr key={a.id}>
                    <td className="mono dim">{new Date(a.createdAt).toLocaleString('zh-CN', { hour12: false })}</td>
                    <td className="mono">{a.actorEmail}</td>
                    <td><span className="sev sev-info">{a.action}</span></td>
                    <td className="mono dim">{a.target}</td>
                    <td className="fine">{a.detail}</td>
                    <td className="mono dim hide-sm">{a.ip}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
      {tab === 'probes' && (
        <section className="panel">
          <div className="panel-head">
            <div><h2>真实探针配置</h2><p className="sub">填写各站 Key 的 probes.json；保存后由 `cmd/prober` 读取执行。</p></div>
            <button className="btn btn-primary btn-sm" onClick={async () => {
              try {
                const r = await api.saveAdminProbeConfig(probeText);
                setProbeSaved(`已保存 ${r.targets} 个目标`);
              } catch (err) {
                setProbeSaved((err as Error).message);
              }
            }}>保存配置</button>
          </div>
          <div className="probe-editor-wrap">
            <textarea className="probe-editor" value={probeText} onChange={(e) => setProbeText(e.target.value)} spellCheck={false}
              placeholder='{"stations":[{"slug":"aihubmix","service":"claude","baseUrl":"https://aihubmix.com/v1","apiKey":"sk-...","model":"claude-opus-5-5"}]}' />
            {probeSaved && <span className="fine">{probeSaved}</span>}
            <p className="fine">Key 只保存在服务端文件，接口返回时会打码。请同时设置 `PROBE_INGEST_TOKEN` 并让 prober 以 `-api` 模式上报。</p>
          </div>
        </section>
      )}

      {tab === 'incidents' && (
        <section className="panel">
          <form className="incident-form" onSubmit={createIncident}>
            <input value={incTitle} onChange={(e) => setIncTitle(e.target.value)} placeholder="事故标题，如「告警投递延迟」" required />
            <select value={incComp} onChange={(e) => setIncComp(e.target.value)} aria-label="影响组件">
              <option value="api">公开 API</option>
              <option value="database">数据存储</option>
              <option value="probes">探针采集</option>
              <option value="alerts">告警投递</option>
              <option value="audit">接口检测</option>
              <option value="email">邮件通道</option>
            </select>
            <input value={incBody} onChange={(e) => setIncBody(e.target.value)} placeholder="说明（可选）" />
            <button className="btn btn-primary btn-sm" type="submit">发布事故</button>
          </form>
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th>标题</th><th>组件</th><th>状态</th><th>开始</th><th>操作</th></tr></thead>
              <tbody>
                {incidents.length === 0 && <tr><td colSpan={5}><span className="fine">暂无事故记录</span></td></tr>}
                {incidents.map((i) => (
                  <tr key={i.id}>
                    <td>{i.title}</td>
                    <td className="mono dim">{i.component}</td>
                    <td><span className={`status-pill ${i.status === 'resolved' ? 'operational' : 'degraded'}`}>{i.status}</span></td>
                    <td className="mono dim">{new Date(i.startedAt).toLocaleString('zh-CN', { hour12: false })}</td>
                    <td>
                      {i.status !== 'resolved' && <button className="link-btn" onClick={() => resolveIncident(i)}>标记恢复</button>}
                      <button className="link-btn" onClick={() => removeIncident(i.id)}>删除</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
      {tab === 'claims' && (
        <section className="panel">
          <div className="panel-head">
            <div><h2>认领修改</h2><p className="sub">站长提交的资料修改；通过后写入站点。</p></div>
            <label className="check-inline">
              <input type="checkbox" checked={showAllEdits} onChange={(e) => { setShowAllEdits(e.target.checked); }} />
              查看已处理
            </label>
          </div>
          <div className="table-wrap">
            <table className="index-table">
              <thead><tr><th>站点</th><th>简介</th><th>退款</th><th className="num">最低充值</th><th>发票</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                {edits.length === 0 && <tr><td colSpan={7}><span className="fine">暂无认领修改</span></td></tr>}
                {edits.map((e) => (
                  <tr key={e.id}>
                    <td><Link className="station-name" to={`/stations/${e.stationSlug}`}>{e.stationName}</Link></td>
                    <td className="fine">{e.tagline || '—'}</td>
                    <td className="fine">{e.refundPolicy || '—'}</td>
                    <td className="num mono">{e.minTopupCny}</td>
                    <td>{e.invoice ? '支持' : '不支持'}</td>
                    <td><span className={`status-pill ${e.status === 'pending' ? 'degraded' : e.status === 'approved' ? 'operational' : 'outage'}`}>{e.status}</span></td>
                    <td>
                      <label className="link-btn" style={{ cursor: 'pointer' }}>
                        图标
                        <input type="file" accept="image/png,image/jpeg,image/webp" hidden
                          onChange={(ev) => uploadIcon(e.stationSlug, ev.target.files?.[0])} />
                      </label>
                      <button className="link-btn" onClick={() => resetIcon(e.stationSlug)}>恢复</button>
                      {iconMsg[e.stationSlug] && <span className="fine" style={{ marginLeft: 6 }}>{iconMsg[e.stationSlug]}</span>}
                      {e.status === 'pending' && (
                        <>
                          <button className="link-btn" onClick={() => decideEdit(e.id, true)}>通过</button>
                          <button className="link-btn" onClick={() => decideEdit(e.id, false)}>拒绝</button>
                        </>
                      )}
                    </td>
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

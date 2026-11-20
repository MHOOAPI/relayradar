import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { AccountData, AlertSubscription, Station } from '../types';
import { Spinner } from '../components/ui';
import { useAuth } from '../lib/auth';
import { useWatchlist } from '../lib/watchlist';
import { useToast } from '../lib/toast';

const KIND_LABEL: Record<string, string> = {
  price_change: '价格变动',
  availability_drop: '可用率下降',
  purity_alert: '纯度异常',
  score_drop: '分数回退',
  new_station: '新站上架',
  provider_incident: '上游异常',
  station_down: '站点不可达',
  station_recovered: '站点恢复',
};

const SUB_STATUS: Record<string, { label: string; cls: string }> = {
  pending: { label: '待审核', cls: 'warn' },
  approved: { label: '已通过', cls: 'ok' },
  rejected: { label: '未通过', cls: 'bad' },
};

const fmtDate = (iso?: string) => {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const fmtDateTime = (iso?: string) => {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('zh-CN', { hour12: false, month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
};

export function AccountPage() {
  const { user, loading, logout } = useAuth();
  const { push } = useToast();
  const [data, setData] = useState<AccountData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [stations, setStations] = useState<Station[]>([]);
  const { ids, toggle } = useWatchlist();

  const load = async () => {
    try {
      const r = await api.account();
      setData(r);
      setError(null);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  useEffect(() => {
    if (user) void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id]);

  useEffect(() => {
    api.stations().then((r) => setStations(r.items)).catch(() => setStations([]));
  }, []);

  const watch = useMemo(
    () => ids.map((slug) => stations.find((s) => s.slug === slug)).filter((s): s is Station => Boolean(s)),
    [ids, stations],
  );

  const setSub = async (sub: AlertSubscription, enabled: boolean) => {
    setBusy(sub.id);
    try {
      await api.accountSetSub(sub.id, enabled);
      setData((prev) => prev && {
        ...prev,
        subscriptions: prev.subscriptions.map((x) => (x.id === sub.id ? { ...x, enabled } : x)),
        stats: { ...prev.stats, active: prev.subscriptions.filter((x) => (x.id === sub.id ? enabled : x.enabled)).length },
      });
      push(enabled ? '已恢复投递' : '已暂停投递', 'ok');
    } catch (e) {
      push((e as Error).message, 'err');
    } finally {
      setBusy(null);
    }
  };

  const removeSub = async (sub: AlertSubscription) => {
    if (!window.confirm('删除这条订阅？删除后不会再收到该订阅的通知。')) return;
    setBusy(sub.id);
    try {
      await api.accountDeleteSub(sub.id);
      setData((prev) => prev && {
        ...prev,
        subscriptions: prev.subscriptions.filter((x) => x.id !== sub.id),
        stats: { ...prev.stats, subscriptions: prev.stats.subscriptions - 1, active: prev.stats.active - (sub.enabled ? 1 : 0) },
      });
      push('订阅已删除', 'ok');
    } catch (e) {
      push((e as Error).message, 'err');
    } finally {
      setBusy(null);
    }
  };

  if (loading) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 读取账号…</div></main>;
  }

  if (!user) {
    return (
      <main className="wrap">
        <div className="page-head"><div>
          <span className="eyebrow"><i />ACCOUNT</span>
          <h1>个人中心</h1>
          <p className="sub">登录后可以在这里管理账号资料、告警订阅、提交记录与关注列表。</p>
        </div></div>
        <section className="panel acct-guest">
          <p>你还没有登录。个人中心只对账号本人可见，数据按登录邮箱归属。</p>
          <div className="acct-actions">
            <Link className="btn btn-primary btn-sm" to="/login?next=/account">登录</Link>
            <Link className="btn btn-ghost btn-sm" to="/register">注册账号</Link>
          </div>
        </section>
      </main>
    );
  }

  const stats = data?.stats ?? { subscriptions: 0, active: 0, submissions: 0, pending: 0 };
  const subs = data?.subscriptions ?? [];
  const subsWriteUps = data?.submissions ?? [];

  return (
    <main className="wrap">
      <div className="page-head"><div>
        <span className="eyebrow"><i />ACCOUNT</span>
        <h1>个人中心</h1>
        <p className="sub">账号资料、告警订阅、提交记录与关注列表；数据只按登录邮箱归属，其他账号看不到。</p>
      </div></div>

      {error && <div className="error-box">{error} <button type="button" className="link-btn" onClick={() => void load()}>重试</button></div>}

      <div className="account-grid">
        <div className="account-main">
          <section className="panel acct-card">
            <div className="acct-id">
              <span className="acct-avatar" aria-hidden="true">{(user.name || user.email).slice(0, 1).toUpperCase()}</span>
              <div>
                <div className="acct-name">
                  {user.name || '未命名'}
                  {user.role === 'admin' && <span className="chip accent">管理员</span>}
                </div>
                <div className="acct-mail mono">{user.email}</div>
              </div>
            </div>
            <dl className="acct-meta">
              <div><dt>邮箱状态</dt><dd>{user.emailVerified ? <span className="chip ok">已验证</span> : <span className="chip warn">待验证</span>}</dd></div>
              <div><dt>注册时间</dt><dd className="mono">{fmtDate(user.createdAt)}</dd></div>
              <div><dt>账号 ID</dt><dd className="mono">#{user.id}</dd></div>
            </dl>
          </section>

          <div className="stat-row">
            <div><span>告警订阅</span><b className="mono">{stats.subscriptions}</b></div>
            <div><span>生效中</span><b className="mono accent">{stats.active}</b></div>
            <div><span>提交记录</span><b className="mono">{stats.submissions}</b></div>
            <div><span>待审核</span><b className="mono">{stats.pending}</b></div>
          </div>

          <section className="panel">
            <div className="panel-head">
              <div>
                <h2>我的告警订阅</h2>
                <p className="sub">价格变动、可用率下降、纯度异常等事件会按订阅推送到邮件或 Webhook。</p>
              </div>
              <Link className="link-btn mono" to="/alerts">新建 / 完整管理 →</Link>
            </div>
            {subs.length === 0 ? (
              <div className="acct-empty">
                <b>还没有订阅</b>
                <p>订阅后，关注站点出问题时你会先收到通知，不用天天自己刷。</p>
                <Link className="btn btn-ghost btn-sm" to="/alerts">去订阅告警 →</Link>
              </div>
            ) : (
              <div className="table-wrap">
                <table className="index-table acct-subs-table">
                  <thead>
                    <tr><th>通道 / 目标</th><th>事件</th><th>站点范围</th><th>状态</th><th>创建时间</th><th className="num">操作</th></tr>
                  </thead>
                  <tbody>
                    {subs.map((sub) => (
                      <tr key={sub.id}>
                        <td>
                          <div className="sub-target">
                            <span className="chip">{sub.channel === 'email' ? '邮件' : 'Webhook'}</span>
                            <span className="mono dim">{sub.channel === 'email' ? sub.email : sub.url}</span>
                          </div>
                        </td>
                        <td><span className="mono dim">{sub.kinds.map((k) => KIND_LABEL[k] ?? k).join(' · ') || '全部'}</span></td>
                        <td><span className="mono dim">{sub.stations.length === 0 ? '全部站点' : sub.stations.length + ' 个站点'}</span></td>
                        <td>
                          {sub.enabled
                            ? <span className="chip ok">投递中</span>
                            : <span className="chip">已暂停</span>}
                          {sub.lastError && <div className="sub-error mono" title={sub.lastError}>{sub.lastError.slice(0, 40)}</div>}
                        </td>
                        <td className="mono dim">{fmtDate(sub.createdAt)}</td>
                        <td className="num">
                          <div className="row-actions">
                            <button type="button" className="link-btn mono" disabled={busy === sub.id} onClick={() => void setSub(sub, !sub.enabled)}>
                              {sub.enabled ? '暂停' : '恢复'}
                            </button>
                            <button type="button" className="link-btn mono danger" disabled={busy === sub.id} onClick={() => void removeSub(sub)}>删除</button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="panel">
            <div className="panel-head">
              <div>
                <h2>我的提交</h2>
                <p className="sub">用本账号邮箱提交的站点；审核通过后会自动出现在站点目录里。</p>
              </div>
              <Link className="link-btn mono" to="/submit">提交新站点 →</Link>
            </div>
            {subsWriteUps.length === 0 ? (
              <div className="acct-empty">
                <b>还没有提交记录</b>
                <p>发现好用的中转站？提交后我们会探测并纳入排名。</p>
                <Link className="btn btn-ghost btn-sm" to="/submit">提交站点 →</Link>
              </div>
            ) : (
              <div className="table-wrap">
                <table className="index-table">
                  <thead><tr><th>站点</th><th>域名</th><th>状态</th><th>提交时间</th><th className="acct-note-col">备注</th></tr></thead>
                  <tbody>
                    {subsWriteUps.map((s) => {
                      const st = SUB_STATUS[s.status] ?? { label: s.status, cls: '' };
                      return (
                        <tr key={s.id}>
                          <td><b>{s.name}</b></td>
                          <td><span className="mono dim">{s.domain}</span></td>
                          <td><span className={`chip ${st.cls}`}>{st.label}</span></td>
                          <td className="mono dim">{fmtDateTime(s.createdAt)}</td>
                          <td className="acct-note-col"><span className="dim">{s.note || '—'}</span></td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>

        <aside className="account-side">
          <section className="panel acct-card">
            <div className="panel-head"><h2>账号安全</h2></div>
            <div className="acct-security">
              <div>
                <span>邮箱验证</span>
                {user.emailVerified
                  ? <span className="chip ok">已验证</span>
                  : <><span className="chip warn">待验证</span><p className="fine">收不到验证邮件可重新注册或联系管理员。</p></>}
              </div>
              <div>
                <span>登录方式</span>
                <span className="chip">邮箱 + 密码</span>
              </div>
              <Link className="btn btn-ghost btn-sm" to="/forgot">修改密码（邮件重置）</Link>
              {user.role === 'admin' && <Link className="btn btn-ghost btn-sm" to="/admin">进入后台</Link>}
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => void logout()}>退出登录</button>
            </div>
          </section>

          <section className="panel acct-card">
            <div className="panel-head"><h2>我的关注</h2><span className="mono dim">{watch.length}</span></div>
            {watch.length === 0 ? (
              <div className="acct-empty small">
                <p>在站点页点「关注」后，这里会列出你盯着的站点。</p>
                <Link className="link-btn mono" to="/stations">去站点目录 →</Link>
              </div>
            ) : (
              <ul className="watch-list">
                {watch.map((s) => (
                  <li key={s.slug}>
                    <Link to={`/stations/${s.slug}`}>{s.name}</Link>
                    <button type="button" className="link-btn mono" onClick={() => toggle(s.slug)} aria-label={`取消关注 ${s.name}`}>✕</button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </aside>
      </div>
    </main>
  );
}

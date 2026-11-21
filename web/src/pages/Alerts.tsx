import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { AlertSubscription, DeliveryLog } from '../types';
import { Spinner } from '../components/ui';

const KIND_LABEL: Record<string, string> = {
  price_change: '价格变动',
  availability_drop: '可用率下降',
  purity_alert: '纯度异常',
  score_drop: '分数回退',
  new_station: '新站上架',
  provider_incident: '上游厂商异常',
  station_down: '站点不可达',
  station_recovered: '站点恢复',
};

const fmt = (iso?: string) => (iso ? new Date(iso).toLocaleString('zh-CN', { hour12: false }) : '—');

export function AlertsPage() {
  const [params] = useSearchParams();
  const id = params.get('id') ?? '';
  const token = params.get('token') ?? '';

  const [sub, setSub] = useState<AlertSubscription | null>(null);
  const [kinds, setKinds] = useState<string[]>([]);
  const [enabled, setEnabled] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState('');
  const [busy, setBusy] = useState(false);
  const [deleted, setDeleted] = useState(false);
  const [deliveries, setDeliveries] = useState<DeliveryLog[]>([]);

  const loadDeliveries = () => {
    if (!id || !token) return;
    api.subDeliveries(id, token).then((r) => setDeliveries(r.items)).catch(() => setDeliveries([]));
  };

  useEffect(() => {
    if (!id || !token) {
      setError('管理链接缺少 id 或 token，请使用创建订阅时保存的链接。');
      return;
    }
    api.getSub(id, token)
      .then((s) => {
        setSub(s);
        setKinds(s.kinds);
        setEnabled(s.enabled);
      })
      .catch((e: Error) => setError(e.message));
    loadDeliveries();
  }, [id, token]);

  const save = async () => {
    setBusy(true);
    setMsg('');
    try {
      const s = await api.updateSub(id, token, { kinds, enabled });
      setSub(s);
      setMsg('已保存');
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const ping = async () => {
    setMsg('');
    try {
      const r = await api.pingSub(id, token);
      setMsg(`测试事件已发送，延迟 ${r.latencyMs}ms`);
      loadDeliveries();
    } catch (e) {
      setMsg((e as Error).message);
    }
  };

  const replay = async (logId: number) => {
    setMsg('重放中…');
    try {
      const r = await api.replayDelivery(id, token, logId);
      setMsg(r.ok ? `重放成功，HTTP ${r.status || 'SMTP'}，${r.latencyMs}ms` : `重放失败：${r.error ?? r.response ?? r.status}`);
      loadDeliveries();
    } catch (e) {
      setMsg((e as Error).message);
    }
  };

  const remove = async () => {
    if (!window.confirm('确认删除这个订阅？删除后无法恢复。')) return;
    try {
      await api.deleteSub(id, token);
      setDeleted(true);
      setSub(null);
    } catch (e) {
      setMsg((e as Error).message);
    }
  };

  if (!id || !token) {
    return (
      <main className="wrap">
        <div className="page-head">
          <div>
            <h1>告警订阅管理</h1>
            <p className="sub">这个页面需要一个来自订阅邮件的私有链接，链接里带有 id 与 token。</p>
          </div>
        </div>
        <section className="panel alerts-card">
          <b>链接不完整</b>
          <p className="sub">请打开订阅确认邮件里的「管理订阅」链接；也可以登录后在个人中心查看自己的订阅。</p>
          <div className="acct-actions">
            <Link className="btn btn-primary btn-sm" to="/changes">去变更雷达创建订阅</Link>
            <Link className="btn btn-ghost btn-sm" to="/account">个人中心</Link>
          </div>
        </section>
      </main>
    );
  }

  if (error && !sub) {
    return (
      <main className="wrap">
        <div className="page-head">
          <div>
            <h1>告警订阅管理</h1>
            <p className="sub">管理事件类型、暂停投递或删除订阅。这个链接含密钥，请勿公开分享。</p>
          </div>
          <Link className="btn btn-ghost btn-sm" to="/changes">返回变更雷达</Link>
        </div>
        <div className="error-box">{error}</div>
      </main>
    );
  }

  if (deleted) {
    return (
      <main className="wrap">
        <div className="empty-panel" style={{ padding: '60px 24px', marginTop: 40 }}>
          <b>订阅已删除</b>
          <p>这个管理链接已失效。需要重新订阅请回到变更雷达。</p>
          <Link className="btn btn-sm btn-ghost" to="/changes">返回变更雷达</Link>
        </div>
      </main>
    );
  }

  if (!sub) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 读取订阅…</div></main>;
  }

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>告警订阅管理</h1>
          <p className="sub">管理事件类型、暂停投递或删除订阅。这个链接含密钥，请勿公开分享。</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/changes">返回变更雷达</Link>
      </div>

      <section className="panel alerts-card">
        <dl className="kv">
          <div><dt>订阅 ID</dt><dd className="mono">{sub.id}</dd></div>
          <div><dt>状态</dt><dd>{enabled ? '投递中' : '已暂停'}</dd></div>
          <div><dt>通道</dt><dd>{sub.channel === 'email' ? '邮件' : 'Webhook'}</dd></div>
          <div><dt>创建时间</dt><dd className="mono">{fmt(sub.createdAt)}</dd></div>
          <div className="wide">
            <dt>{sub.channel === 'email' ? '接收邮箱' : 'Webhook'}</dt>
            <dd className="mono break">{sub.channel === 'email' ? sub.email : sub.url}</dd>
          </div>
          <div><dt>最近投递</dt><dd className="mono">{fmt(sub.lastSentAt)}</dd></div>
          <div><dt>连续失败</dt><dd className="mono">{sub.attempts}</dd></div>
          <div><dt>下次重试</dt><dd className="mono">{fmt(sub.nextAttemptAt)}</dd></div>
        </dl>
        {sub.lastError && <p className="form-error" style={{ padding: '10px 20px' }}>最近错误：{sub.lastError}</p>}

        <div className="alerts-form">
          <p className="fine" style={{ margin: '0 0 4px' }}>
            价格变动来自每日快照（同一站点同一模型的价格与上次不同即记录）；可用率与分数事件来自探针。
          </p>
          <div className="sub-kinds">
            {Object.entries(KIND_LABEL).map(([k, label]) => (
              <label key={k} className="check-inline">
                <input
                  type="checkbox"
                  checked={kinds.includes(k)}
                  onChange={(e) => setKinds((prev) => (e.target.checked ? [...prev, k] : prev.filter((x) => x !== k)))}
                />
                {label}
              </label>
            ))}
          </div>
          <label className="check-inline">
            <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
            启用投递（取消勾选即暂停，保留订阅）
          </label>
          <div className="subscribe-actions">
            <button className="btn btn-primary btn-sm" onClick={save} disabled={busy || kinds.length === 0}>
              {busy ? '保存中…' : '保存修改'}
            </button>
            <button className="btn btn-ghost btn-sm" onClick={ping}>发送测试事件</button>
            <button className="btn btn-ghost btn-sm danger" onClick={remove}>删除订阅</button>
            {msg && <span className="mono dim tiny">{msg}</span>}
          </div>
        </div>
      </section>

      <section className="panel deliveries-card">
        <div className="panel-head">
          <div>
            <h2>投递记录</h2>
            <p className="sub">最近 50 次投递；失败可手动重放，重放会写入一条新记录。</p>
          </div>
          <button className="btn btn-ghost btn-sm" onClick={loadDeliveries}>刷新</button>
        </div>
        <div className="table-wrap">
          <table className="index-table">
            <thead><tr><th>时间</th><th>事件</th><th className="num">状态</th><th className="hide-sm">结果</th><th className="num">耗时</th><th>响应</th><th>操作</th></tr></thead>
            <tbody>
              {deliveries.length === 0 && <tr><td colSpan={7}><span className="fine">还没有投递记录，先发一次测试事件。</span></td></tr>}
              {deliveries.map((d) => (
                <tr key={d.id}>
                  <td className="mono dim">{fmt(d.createdAt)}</td>
                  <td className="mono">{d.event}</td>
                  <td className="num mono">{d.status || 'SMTP'}</td>
                  <td className="hide-sm"><span className={`status-pill ${d.ok ? 'approved' : 'rejected'}`}>{d.ok ? '成功' : '失败'}</span></td>
                  <td className="num mono">{d.durationMs}ms</td>
                  <td className="delivery-resp mono dim">{d.response || '—'}</td>
                  <td><button className="link-btn" onClick={() => replay(d.id)}>重放</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </main>
  );
}

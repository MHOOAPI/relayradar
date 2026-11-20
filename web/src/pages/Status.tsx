import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { StatusResponse } from '../types';import { Spinner } from '../components/ui';

const LABEL: Record<string, string> = {
  operational: '运行正常',
  degraded: '部分降级',
  outage: '服务中断',
  unknown: '状态未知',
};

const COMPONENT_DESC: Record<string, string> = {
  api: '公开 JSON API 与页面请求',
  database: 'Postgres / 内存快照读取',
  probes: '站点探针采集与聚合',
  alerts: 'Webhook / 邮件告警投递',
  audit: 'OpenAI 兼容端点检测',
  email: 'SMTP 邮件通道',
};

const fmtDuration = (sec: number) => {
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  return h > 0 ? `${h} 小时 ${m} 分` : `${m} 分钟`;
};

const daysFor = (data: StatusResponse | null, key: string) =>
  data?.history.find((h) => h.key === key)?.days ?? [];

const uptimeOf = (days: { status: string; uptime: number }[]) => {
  const known = days.filter((d) => d.status !== 'unknown');
  if (!known.length) return null;
  return known.reduce((sum, d) => sum + d.uptime, 0) / known.length;
};

const INCIDENT_LABEL: Record<string, string> = {
  investigating: '调查中',
  identified: '已定位',
  monitoring: '观察中',
  resolved: '已恢复',
};

export function StatusPage() {
  const [data, setData] = useState<StatusResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [last, setLast] = useState<Date | null>(null);

  useEffect(() => {
    document.title = '服务状态 | RelayRadar';
    let alive = true;
    const load = () => {
      api.status()
        .then((r) => {
          if (!alive) return;
          setData(r);
          setLast(new Date());
          setError(null);
        })
        .catch((e: Error) => alive && setError(e.message));
    };
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
            <h1>{LABEL[overall] ?? '状态未知'}</h1>
            <p>
              {data
                ? `最后检查 ${last?.toLocaleTimeString('zh-CN', { hour12: false })} · 已运行 ${fmtDuration(data.uptimeSeconds)} · 版本 v${data.version}`
                : '正在读取服务状态…'}
            </p>
          </div>
        </div>
        <div className="status-hero-side mono">
          <span>API</span><b>/api/v1/status</b>
          <span>刷新</span><b>30s</b>
        </div>
      </section>

      {error && <div className="error-box">状态接口暂时不可用：{error}</div>}
      {!data && !error && <div className="loading"><Spinner /> 加载组件状态…</div>}

      {data && (
        <section className="panel status-panel">
          <div className="panel-head">
            <div><h2>组件状态</h2><p className="sub">按真实依赖检查，不是静态文案。</p></div>
            <span className="mono dim">{data.components.length} 项</span>
          </div>
          <ul className="status-list">
            {data.components.map((c) => {
              const days = daysFor(data, c.key);
              const up = uptimeOf(days);
              return (
                <li key={c.key} className={`status-row status-${c.status}`}>
                  <span className="status-dot" />
                  <div className="status-info">
                    <b>{c.name}</b>
                    <small>{COMPONENT_DESC[c.key] ?? ''}</small>
                  </div>
                  <div className="status-bars" aria-hidden="true">
                    {days.map((d) => (
                      <i key={d.date} className={`b-${d.status}`} title={`${d.date} · ${d.uptime.toFixed(1)}%`} />
                    ))}
                  </div>
                  <span className="status-uptime mono">{up === null ? '—' : `${up.toFixed(1)}%`}</span>
                  <span className="status-detail mono">{c.detail}</span>
                  <span className={`status-pill ${c.status}`}>{LABEL[c.status] ?? c.status}</span>
                </li>
              );
            })}
          </ul>
        </section>
      )}

      <section className="panel status-panel">
        <div className="panel-head">
          <div><h2>最近事件</h2><p className="sub">公开事故与维护记录，按开始时间倒序。</p></div>
          <a className="mono dim tiny" href="/status.rss">RSS</a>
        </div>
        {data && (data.incidents?.length ?? 0) > 0 ? (
          <ul className="incident-list">
            {data.incidents.map((i) => (
              <li key={i.id} className={`incident incident-${i.status}`}>
                <span className={`status-pill ${i.status === 'resolved' ? 'operational' : i.status === 'monitoring' ? 'degraded' : 'outage'}`}>
                  {INCIDENT_LABEL[i.status] ?? i.status}
                </span>
                <div className="incident-main">
                  <b>{i.title}</b>
                  {i.body && <p>{i.body}</p>}
                  <small className="mono">
                    {i.component} · 开始 {new Date(i.startedAt).toLocaleString('zh-CN', { hour12: false })}
                    {i.resolvedAt && ` · 恢复 ${new Date(i.resolvedAt).toLocaleString('zh-CN', { hour12: false })}`}
                  </small>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <div className="status-empty">
            <span className="status-dot" />
            <p>过去 90 天没有公开事故记录。</p>
          </div>
        )}
      </section>

      <p className="fine" style={{ marginTop: 14 }}>
        需要变更通知？订阅 <Link to="/changes">变更雷达</Link> 的告警，或查看 <Link to="/methodology">评分方法</Link>。
      </p>
    </main>
  );
}

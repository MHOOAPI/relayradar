import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { ModelProviderStatus } from '../types';
import { Spinner } from '../components/ui';

const STATUS_LABEL: Record<string, string> = {
  operational: '正常运行',
  degraded: '部分降级',
  outage: '服务中断',
  unknown: '状态未知',
};

const IMPACT_LABEL: Record<string, string> = {
  none: '正常',
  minor: '轻微',
  major: '严重',
  critical: '严重',
  maintenance: '维护',
};

const dayOf = (iso?: string) => (iso ? iso.slice(0, 10) : '');

export function ModelStatusDetailPage() {
  const { key = '' } = useParams();
  const [providers, setProviders] = useState<ModelProviderStatus[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.modelStatus()
      .then((r) => {
        setProviders(r.providers);
        const p = r.providers.find((x) => x.key === key);
        if (p) document.title = `${p.name} 官方 API 状态历史 | RelayRadar`;
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
  if (!providers) return <main className="wrap"><div className="loading-page"><Spinner /> 读取状态历史…</div></main>;
  if (!provider) {
    return (
      <main className="wrap">
        <div className="error-box">没有找到这个厂商。<Link className="link-btn" to="/model-status">返回模型状态</Link></div>
      </main>
    );
  }

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/model-status">模型 API 状态</Link><span>/</span><span className="dim">{provider.key}</span></div>

      <div className="provider-tabs">
        {providers.map((p) => (
          <Link key={p.key} className={`provider-tab ${p.key === provider.key ? 'on' : ''}`} to={`/model-status/${p.key}`}>
            <i className={`pc-dot ${p.status}`} />
            {p.name}
          </Link>
        ))}
      </div>

      <section className={`model-status-hero status-${provider.status}`}>
        <div>
          <span className="mono status-kicker">OFFICIAL API STATUS</span>
          <h1>{provider.name} 官方 API 状态</h1>
          <p>当前：{STATUS_LABEL[provider.status] ?? provider.status} · {provider.description || '—'}</p>
          <small className="mono">
            {provider.checkedVia === 'statuspage' ? '来自官方状态页接口' : '来自端点可达性探测'}
            {' · '}
            {new Date(provider.checkedAt).toLocaleString('zh-CN', { hour12: false })}
          </small>
        </div>
        <a className="btn btn-ghost btn-sm" href={provider.statusUrl} target="_blank" rel="noopener noreferrer">官方状态页 ↗</a>
      </section>

      <div className="status-history-head">
        <h2>历史记录</h2>
        <span className="mono dim">{provider.incidents?.length ?? 0} 条事件</span>
      </div>

      {provider.history && provider.history.some((d) => d.status !== 'unknown') && (
        <div className="provider-history panel">
          <div className="provider-history-bars">
            {provider.history.map((d) => (
              <i key={d.date} className={`b-${d.status}`} title={`${d.date} · ${d.uptime.toFixed(1)}%`} />
            ))}
          </div>
          <span className="mono dim">
            {(() => {
              const known = provider.history!.filter((d) => d.status !== 'unknown');
              if (!known.length) return '暂无历史数据';
              const up = known.reduce((s, d) => s + d.uptime, 0) / known.length;
              return `近 90 天 · ${known.length} 天数据 · 可用率 ${up.toFixed(1)}%`;
            })()}
          </span>
        </div>
      )}

      {days.length === 0 && (
        <div className="panel status-empty"><span className="status-dot" /><p>官方状态页暂未提供公开事件记录。</p></div>
      )}

      <div className="status-timeline">
        {days.map((d) => {
          const worst = d.items.some((i) => i.impact === 'critical' || i.impact === 'major')
            ? 'bad'
            : d.items.some((i) => i.impact === 'minor')
              ? 'warn'
              : 'ok';
          return (
            <article key={d.date} className={`timeline-day timeline-${worst}`}>
              <div className="timeline-date mono">
                <b>{d.date.slice(8, 10)}</b>
                <span>{d.date.slice(5, 7)} 月</span>
              </div>
              <div className="timeline-body">
                <div className="timeline-day-head">
                  <span className={`status-pill ${worst === 'ok' ? 'operational' : worst === 'warn' ? 'degraded' : 'outage'}`}>
                    {worst === 'ok' ? '正常' : worst === 'warn' ? '降级' : '异常'}
                  </span>
                  <span className="fine">{d.items.length} 条事件</span>
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
                            <span className="mono">{new Date(u.createdAt).toLocaleTimeString('zh-CN', { hour12: false })}</span>
                            <span className="mono dim">{u.status}</span>
                            <p>{u.body}</p>
                          </li>
                        ))}
                      </ul>
                    )}
                    {inc.url && <a className="link-btn tiny" href={inc.url} target="_blank" rel="noopener noreferrer">查看官方事件 ↗</a>}
                  </div>
                ))}
              </div>
            </article>
          );
        })}
      </div>

      <p className="fine" style={{ marginTop: 16 }}>
        事件数据来自厂商官方 Statuspage；没有机器接口的厂商只显示当前可达性。本站服务状态见 <Link to="/status">/status</Link>。
      </p>
    </main>
  );
}

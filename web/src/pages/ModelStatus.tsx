import '../styles/status.css';
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { ModelProviderStatus } from '../types';
import { Spinner } from '../components/ui';

const LABEL: Record<string, string> = {
  operational: '运行正常',
  degraded: '降级 / 限流',
  outage: '不可用',
  unknown: '状态未知',
};

const SOURCE: Record<string, string> = {
  statuspage: '官方状态页',
  probe: '可达性探测',
};

export function ModelStatusPage() {
  const [items, setItems] = useState<ModelProviderStatus[] | null>(null);
  const [checkedAt, setCheckedAt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = '上游模型 API 状态：OpenAI / Anthropic / Gemini 等 | RelayRadar';
    let alive = true;
    const load = () =>
      api.modelStatus()
        .then((r) => {
          if (!alive) return;
          setItems(r.providers);
          setCheckedAt(r.checkedAt);
          setError(null);
        })
        .catch((e: Error) => alive && setError(e.message));
    load();
    const timer = window.setInterval(load, 60000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>上游模型 API 状态</h1>
          <p className="sub">
            OpenAI、Anthropic、Google Gemini、DeepSeek、Moonshot、阿里 Qwen、xAI 的官方 API 状态。
            优先读取官方状态页机器接口，缺失时用端点可达性探测兜底。
          </p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/status">本站服务状态 →</Link>
      </div>

      {error && <div className="error-box">{error}</div>}
      {!items && !error && <div className="loading"><Spinner /> 读取上游状态…</div>}

      <div className="provider-grid">
        {items?.map((p) => (
          <article key={p.key} className={`provider-card status-${p.status}`}>
            <div className="provider-head">
              <b>{p.name}</b>
              <span className={`status-pill ${p.status}`}>{LABEL[p.status] ?? p.status}</span>
            </div>
            <Link className="provider-detail-link" to={`/model-status/${p.key}`}>查看状态历史 →</Link>
            <p className="provider-desc">{p.description || '—'}</p>
            {p.components && p.components.length > 0 && (
              <div className="provider-components">
                {p.components.slice(0, 4).map((c) => (
                  <span key={c.name} className={`pc ${c.status}`} title={c.status}>{c.name}</span>
                ))}
              </div>
            )}
            {p.incidents && p.incidents.length > 0 && (
              <a className="provider-incident" href={p.incidents[0].url || p.statusUrl} target="_blank" rel="noopener noreferrer">
                最近事件：{p.incidents[0].name}
              </a>
            )}
            <div className="provider-foot">
              <span className={`source-badge ${p.checkedVia}`}>{SOURCE[p.checkedVia] ?? p.checkedVia}</span>
              <a href={p.statusUrl} target="_blank" rel="noopener noreferrer">官方状态页 ↗</a>
            </div>
          </article>
        ))}
      </div>

      <p className="fine" style={{ marginTop: 16 }}>
        {checkedAt && `检查时间 ${new Date(checkedAt).toLocaleString('zh-CN', { hour12: false })} · 每 60 秒刷新。`}
        这里展示的是上游模型厂商的官方 API 状态，不是 RelayRadar 自身；本站状态见 <Link to="/status">/status</Link>。
        「可达性探测」失败也可能由本机网络或地区限制导致，请以官方状态页为准。
      </p>
    </main>
  );
}

import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import type { Change } from '../types';
import { Spinner } from '../components/ui';
import { useWatchlist } from '../lib/watchlist';

const KIND_LABEL: Record<string, string> = {
  all: '全部',
  price_change: '价格变动',
  availability_drop: '可用率下降',
  purity_alert: '纯度异常',
  score_drop: '分数回退',
  new_station: '新站上架',
  provider_incident: '上游厂商异常',
  station_down: '站点不可达',
  station_recovered: '站点恢复',
};

const SEVERITY_LABEL: Record<string, string> = { info: '提示', warn: '注意', critical: '高危' };
const SUB_KINDS = ['price_change', 'availability_drop', 'purity_alert', 'score_drop', 'new_station', 'provider_incident', 'station_down', 'station_recovered'];

export function ChangesPage() {
  const [params, setParams] = useSearchParams();
  const station = params.get('station') ?? '';
  const [kind, setKind] = useState(params.get('kind') ?? 'all');
  const [onlyWatch, setOnlyWatch] = useState(params.get('watch') === '1');
  const [items, setItems] = useState<Change[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setParams((prev) => {
      const p = new URLSearchParams(prev);
      if (kind !== 'all') p.set('kind', kind); else p.delete('kind');
      if (onlyWatch) p.set('watch', '1'); else p.delete('watch');
      return p;
    }, { replace: true });
  }, [kind, onlyWatch, setParams]);
  const { ids } = useWatchlist();
  const [hookUrl, setHookUrl] = useState('');
  const [channel, setChannel] = useState<'webhook' | 'email'>('webhook');
  const [email, setEmail] = useState('');
  const [subKinds, setSubKinds] = useState<string[]>(SUB_KINDS);
  const [subProviders, setSubProviders] = useState<string[]>([]);
  const [subWatch, setSubWatch] = useState(false);
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<{ id: string; secret: string } | null>(null);
  const [subMsg, setSubMsg] = useState('');
  const [subErr, setSubErr] = useState<string | null>(null);

  const createSub = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setSubErr(null);
    setSubMsg('');
    try {
      const r = await api.subscribe({
        channel,
        url: channel === 'webhook' ? hookUrl.trim() : '',
        email: channel === 'email' ? email.trim() : '',
        kinds: subKinds,
        stations: subWatch ? ids : [],
        providers: subProviders,
      });
      setCreated({ id: r.id, secret: r.secret });
    } catch (err) {
      setSubErr((err as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const pingSub = async () => {
    if (!created) return;
    try {
      const r = await api.pingSub(created.id, created.secret);
      setSubMsg(`测试事件已发送，延迟 ${r.latencyMs}ms`);
    } catch (err) {
      setSubMsg((err as Error).message);
    }
  };

  const removeSub = async () => {
    if (!created) return;
    try {
      await api.deleteSub(created.id, created.secret);
      setCreated(null);
      setSubMsg('订阅已删除');
    } catch (err) {
      setSubMsg((err as Error).message);
    }
  };

  useEffect(() => {
    setItems(null);
    api.changes(station)
      .then((r) => setItems(r.items))
      .catch((e: Error) => setError(e.message));
  }, [station]);

  const rows = useMemo(() => {
    if (!items) return [];
    return items.filter((c) => {
      if (kind !== 'all' && c.kind !== kind) return false;
      if (onlyWatch && !ids.includes(c.stationSlug)) return false;
      return true;
    });
  }, [items, kind, onlyWatch, ids]);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>变更雷达</h1>
          <p className="sub">
            从探针与价格历史里自动检测：可用率骤降、纯度抽检异常、价格调整、新站上架。
            没有人工爆料，也没有付费置顶。
          </p>
        </div>
      </div>

      {error && <div className="error-box">{error}</div>}

      <div className="tabs">
        {Object.entries(KIND_LABEL).map(([k, label]) => (
          <button key={k} className={`tab ${kind === k ? 'on' : ''}`} onClick={() => setKind(k)}>{label}</button>
        ))}
        <button className={`tab ${onlyWatch ? 'on' : ''}`} onClick={() => setOnlyWatch((v) => !v)}>
          ★ 只看关注（{ids.length}）
        </button>
        {station && <span className="tabs-note">已过滤站点：<Link className="link-btn" to="/changes">清除</Link></span>}
      </div>

      <details className="panel subscribe-panel" open={params.get('subscribe') === '1'}>
        <summary>
          <span>
            <b>告警订阅（Webhook / 邮件）</b>
            <small>新变更推送到你的地址或邮箱，Webhook 带 HMAC-SHA256 签名。</small>
          </span>
          <span className="subscribe-plus mono">+</span>
        </summary>
        <div className="subscribe-body">
          <form className="subscribe-form" onSubmit={createSub}>
            <label>
              <span>投递方式</span>
              <select value={channel} onChange={(e) => setChannel(e.target.value as 'webhook' | 'email')}>
                <option value="webhook">Webhook（JSON + 签名）</option>
                <option value="email">邮件（需要服务端配置 SMTP）</option>
              </select>
            </label>
            {channel === 'webhook' ? (
              <label>
                <span>Webhook URL</span>
                <input
                  value={hookUrl}
                  onChange={(e) => setHookUrl(e.target.value)}
                  placeholder="https://example.com/hooks/relayradar"
                  required
                />
              </label>
            ) : (
              <label>
                <span>接收邮箱</span>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  required
                />
              </label>
            )}
            <div className="sub-kinds">
              {SUB_KINDS.map((k) => (
                <label key={k} className="check-inline">
                  <input
                    type="checkbox"
                    checked={subKinds.includes(k)}
                    onChange={(e) =>
                      setSubKinds((prev) => (e.target.checked ? [...prev, k] : prev.filter((x) => x !== k)))
                    }
                  />
                  {KIND_LABEL[k]}
                </label>
              ))}
            </div>
            <div className="sub-kinds">
              <span className="ref-label mono">上游厂商</span>
              {[
                ['openai', 'OpenAI'], ['anthropic', 'Anthropic'], ['google', 'Gemini'],
                ['deepseek', 'DeepSeek'], ['moonshot', 'Moonshot'], ['alibaba', 'Qwen'], ['xai', 'xAI'],
              ].map(([key, label]) => (
                <label key={key} className="check-inline">
                  <input
                    type="checkbox"
                    checked={subProviders.includes(key)}
                    onChange={(e) => setSubProviders((prev) => (e.target.checked ? [...prev, key] : prev.filter((x) => x !== key)))}
                  />
                  {label}
                </label>
              ))}
            </div>
            {ids.length > 0 && (
              <label className="check-inline">
                <input type="checkbox" checked={subWatch} onChange={(e) => setSubWatch(e.target.checked)} />
                只推送关注列表（{ids.length} 家）
              </label>
            )}
            <div className="subscribe-actions">
              <button className="btn btn-primary btn-sm" disabled={creating || subKinds.length === 0}>
                {creating ? '创建中…' : '创建订阅'}
              </button>
              <span className="fine">最近 30 秒轮询一次，只推送新事件。</span>
            </div>
            {subErr && <p className="form-error">{subErr}</p>}
          </form>

          {created && (
            <div className="sub-created">
              <p>已创建 <b className="mono">{created.id}</b>。签名密钥只显示一次，请立即保存：</p>
              <code className="sub-secret mono">{created.secret}</code>
              <div className="subscribe-actions">
                <button type="button" className="btn btn-sm btn-ghost" onClick={pingSub}>发送测试事件</button>
                <Link className="btn btn-sm btn-ghost" to={`/alerts?id=${created.id}&token=${created.secret}`}>管理订阅</Link>
                <button type="button" className="btn btn-sm btn-ghost" onClick={removeSub}>删除订阅</button>
                {subMsg && <span className="mono dim tiny">{subMsg}</span>}
              </div>
            </div>
          )}
        </div>
      </details>

      <section className="panel">
        {!items && !error && <div className="loading"><Spinner /> 计算变更中…</div>}
        {items && rows.length === 0 && (
          <div className="empty-panel" style={{ padding: '40px 24px' }}>
            <b>没有匹配的变更</b>
            <p>换个筛选条件，或去榜单关注几家站点后再回来。</p>
          </div>
        )}
        <ul className="changes-list big">
          {rows.map((c) => (
            <li key={c.id} className={`change change-${c.severity}`}>
              <span className="change-dot" />
              <div className="change-body">
                <div className="change-title">
                  <b>{c.title}</b>
                  <span className={`sev sev-${c.severity}`}>{SEVERITY_LABEL[c.severity]}</span>
                  <span className="mono dim tiny">{KIND_LABEL[c.kind]}</span>
                </div>
                <p>{c.detail}</p>
                <Link className="change-station" to={`/stations/${c.stationSlug}`}>{c.stationName} →</Link>
              </div>
              <time className="mono dim">{new Date(c.at).toLocaleString('zh-CN', { hour12: false })}</time>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}

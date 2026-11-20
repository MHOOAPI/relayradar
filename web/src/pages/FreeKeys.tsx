import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import { SliderCaptcha } from '../components/SliderCaptcha';
import type { FreeKeyView } from '../types';
import { Spinner } from '../components/ui';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const STATUS: Record<string, { label: string; cls: string }> = {
  available: { label: '可用', cls: 'ok' },
  invalid: { label: '已失效', cls: 'bad' },
  unknown: { label: '待验证', cls: 'warn' },
};

const fmtDate = (iso: string) => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' });
};

const modelTags = (models: string) =>
  models
    .split(/[,，\s]+/)
    .map((m) => m.trim())
    .filter(Boolean);

export function FreeKeysPage() {
  const [items, setItems] = useState<FreeKeyView[] | null>(null);
  const [q, setQ] = useState('');
  const [provider, setProvider] = useState('');
  const [status, setStatus] = useState<'all' | 'available' | 'unknown'>('all');
  const [showInvalid, setShowInvalid] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const { push } = useToast();

  const [claim, setClaim] = useState<{ apiKey: string; baseUrl: string; models: string; warning: string } | null>(null);
  const [claiming, setClaiming] = useState<number | null>(null);

  const load = () => {
    setItems(null);
    api
      .freeKeys('', showInvalid)
      .then((r) => setItems(r.items))
      .catch((e: Error) => setError(e.message));
  };

  useEffect(() => {
    document.title = '免费 API Key：社区共享、无需登录领取 | RelayRadar';
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showInvalid]);

  const doClaim = async (id: number) => {
    setClaiming(id);
    try {
      const r = await api.claimFreeKey(id);
      setClaim(r);
    } catch (e) {
      push((e as Error).message, 'err');
    } finally {
      setClaiming(null);
    }
  };

  const available = items?.filter((k) => k.status === 'available') ?? [];
  const claims = items?.reduce((sum, k) => sum + k.claims, 0) ?? 0;
  const providers = useMemo(() => {
    const set = new Map<string, number>();
    for (const k of items ?? []) set.set(k.provider, (set.get(k.provider) ?? 0) + 1);
    return [...set.entries()].sort((a, b) => b[1] - a[1]);
  }, [items]);

  const needle = q.trim().toLowerCase();
  const filtered = (items ?? []).filter((k) => {
    if (status !== 'all' && k.status !== status) return false;
    if (provider && k.provider !== provider) return false;
    if (needle) {
      const hay = `${k.provider} ${k.baseUrl} ${k.models} ${k.submitter} ${k.note}`.toLowerCase();
      if (!hay.includes(needle)) return false;
    }
    return true;
  });

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/free">免费额度</Link><span>/</span><span className="dim">免费 API Key</span></div>

      <section className="panel fk-hero">
        <div className="fk-hero-main">
          <span className="eyebrow"><i />FREE API KEYS</span>
          <h1>免费 API Key</h1>
          <p className="sub">
            社区共享的 OpenAI 兼容 Key，无需登录即可领取。每个 Key 提交时会自动探测一次，并显示社区判定；共享 Key 随时可能被吊销，
            <b>不要用于生产或绑定付费方式</b>。
          </p>
          <div className="fk-actions">
            <button className="btn btn-primary btn-sm" onClick={() => setShowForm(true)}>+ 分享我的 Key</button>
            <Link className="btn btn-ghost btn-sm" to="/free">官方免费额度 →</Link>
            <Link className="btn btn-ghost btn-sm" to="/audit">先检测再用 →</Link>
          </div>
        </div>
        <div className="fk-hero-stats">
          <div><b className="mono accent">{available.length}</b><span>可用 Key</span></div>
          <div><b className="mono">{claims}</b><span>累计领取</span></div>
          <div><b className="mono">{providers.length}</b><span>服务商</span></div>
        </div>
      </section>

      <div className="fk-toolbar">
        <div className="fk-search">
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="搜索服务商、站点或模型"
            aria-label="搜索免费 Key"
          />
        </div>
        <div className="fk-filters">
          <div className="seg">
            {([['all', '全部'], ['available', '可用'], ['unknown', '待验证']] as const).map(([key, label]) => (
              <button key={key} type="button" className={`seg-btn ${status === key ? 'on' : ''}`} onClick={() => setStatus(key)}>
                {label}
              </button>
            ))}
          </div>
          {providers.length > 1 && (
            <select className="fk-select" value={provider} onChange={(e) => setProvider(e.target.value)} aria-label="按服务商筛选">
              <option value="">全部服务商</option>
              {providers.map(([name, count]) => <option key={name} value={name}>{name}（{count}）</option>)}
            </select>
          )}
          <label className="check-inline">
            <input type="checkbox" checked={showInvalid} onChange={(e) => setShowInvalid(e.target.checked)} />
            显示已失效
          </label>
        </div>
      </div>

      {error && <div className="error-box">{error}</div>}

      {!items && (
        <div className="fk-grid">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="panel fk-card fk-skeleton"><Spinner /> 加载中…</div>
          ))}
        </div>
      )}

      {items && filtered.length === 0 && (
        <section className="panel fk-empty">
          <b>{q || provider || status !== 'all' || showInvalid ? '没有符合条件的 Key' : '还没有人分享 Key'}</b>
          <p>
            {items.length === 0
              ? '有闲置的临时 Key？分享出来，别人领取时你会看到领取次数。'
              : '换个关键词或筛选条件试试；失效 Key 默认隐藏。'}
          </p>
          <div className="fk-actions">
            <button className="btn btn-primary btn-sm" onClick={() => setShowForm(true)}>+ 分享我的 Key</button>
            {items.length > 0 && <button className="btn btn-ghost btn-sm" onClick={() => { setQ(''); setProvider(''); setStatus('all'); setShowInvalid(false); }}>重置筛选</button>}
          </div>
        </section>
      )}

      {filtered.length > 0 && (
        <div className="fk-grid">
          {filtered.map((k) => {
            const st = STATUS[k.status] ?? STATUS.unknown;
            const tags = modelTags(k.models);
            return (
              <article key={k.id} className={`panel fk-card ${k.status}`}>
                <header className="fk-card-head">
                  <span className={`chip ${st.cls}`}>{st.label}</span>
                  <span className="mono dim">{k.claims} 次领取</span>
                </header>
                <div className="fk-key mono">{k.keyMasked}</div>
                <div className="fk-provider">
                  <b>{k.provider}</b>
                  <span className="mono dim">{k.baseUrl}</span>
                </div>
                {tags.length > 0 && (
                  <div className="fk-models">
                    {tags.slice(0, 4).map((m) => <span key={m} className="chip mono">{m}</span>)}
                    {tags.length > 4 && <span className="chip mono">+{tags.length - 4}</span>}
                  </div>
                )}
                {k.note && <p className="fk-note">{k.note}</p>}
                <footer className="fk-card-foot">
                  <span className="dim">{k.submitter || '匿名'} · {fmtDate(k.createdAt)}</span>
                  <button
                    className="btn btn-primary btn-sm"
                    onClick={() => void doClaim(k.id)}
                    disabled={claiming === k.id || k.status === 'invalid'}
                  >
                    {claiming === k.id ? '领取中…' : k.status === 'invalid' ? '已失效' : '领取 →'}
                  </button>
                </footer>
              </article>
            );
          })}
        </div>
      )}

      <section className="panel fk-how">
        <div className="fk-how-main">
          <h2>这些 Key 从哪来，怎么用</h2>
          <ol className="fk-steps">
            <li><b>有人分享</b><span>提交 Base URL 与临时 Key，服务端立即探测一次并标注可用性。</span></li>
            <li><b>你领取</b><span>点「领取」拿到完整 Key 与 Base URL，直接填进 OpenAI 兼容客户端。</span></li>
            <li><b>先验证再依赖</b><span>共享 Key 可能随时失效或被他人用超额度；重要任务请用官方 Key，或先用接口检测确认。</span></li>
          </ol>
        </div>
        <div className="fk-how-side">
          <h3>风险提示</h3>
          <ul>
            <li>不要提交绑定支付方式、额度较高的主 Key。</li>
            <li>不要把共享 Key 用在生产环境或付费业务。</li>
            <li>分享 Key 可能违反部分服务商条款，请自行判断。</li>
          </ul>
          <Link className="link-btn mono" to="/methodology#audit">接口检测怎么判 →</Link>
        </div>
      </section>

      {showForm && (
        <SubmitModal
          onClose={() => setShowForm(false)}
          onDone={(st) => {
            setShowForm(false);
            push(st === 'available' ? '提交成功，Key 已验证可用' : st === 'invalid' ? '提交成功，但探测显示 Key 不可用' : '提交成功，等待验证', st === 'invalid' ? 'err' : 'ok');
            load();
          }}
        />
      )}

      {claim && (
        <div className="modal-mask" onClick={() => setClaim(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head"><h2>领取成功</h2><button className="mm-close" onClick={() => setClaim(null)}>✕</button></div>
            <p className="fine">请立即复制保存；共享 Key 可能随时被吊销。</p>
            <label className="fk-copy">
              <span>Base URL</span>
              <div>
                <code className="mono">{claim.baseUrl}</code>
                <button type="button" className="link-btn" onClick={() => void copyText(claim.baseUrl).then((ok) => push(ok ? 'Base URL 已复制' : '复制失败', ok ? 'ok' : 'err'))}>复制</button>
              </div>
            </label>
            <label className="fk-copy">
              <span>API Key</span>
              <div>
                <code className="mono">{claim.apiKey}</code>
                <button type="button" className="link-btn" onClick={() => void copyText(claim.apiKey).then((ok) => push(ok ? 'API Key 已复制' : '复制失败', ok ? 'ok' : 'err'))}>复制</button>
              </div>
            </label>
            {claim.models && (
              <div className="fk-claim-models">
                {modelTags(claim.models).map((m) => <span key={m} className="chip mono">{m}</span>)}
              </div>
            )}
            <p className="fk-warning">{claim.warning}</p>
          </div>
        </div>
      )}
    </main>
  );
}

function SubmitModal({ onClose, onDone }: { onClose: () => void; onDone: (status: string) => void }) {
  const [baseUrl, setBaseUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [models, setModels] = useState('');
  const [submitter, setSubmitter] = useState('');
  const [note, setNote] = useState('');
  const [website, setWebsite] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [captchaOK, setCaptchaOK] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    try {
      const r = await api.createFreeKey({ baseUrl: baseUrl.trim(), apiKey: apiKey.trim(), models: models.trim(), note: note.trim(), submitter: submitter.trim(), website });
      onDone(r.status);
    } catch (e2) {
      setErr((e2 as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h2>分享免费 API Key</h2>
          <button className="mm-close" onClick={onClose}>✕</button>
        </div>
        <p className="fine">提交后会立即探测一次可用性；无需登录。请勿提交绑定支付方式的主 Key。</p>
        <form className="auth-form" onSubmit={submit}>
          <label><span>Base URL</span><input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://api.example.com/v1" required /></label>
          <label><span>API Key</span><input value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="sk-..." required autoComplete="off" /></label>
          <label><span>可用模型（可选）</span><input value={models} onChange={(e) => setModels(e.target.value)} placeholder="gpt-4o-mini, claude-sonnet-4" /></label>
          <label><span>昵称（可选）</span><input value={submitter} onChange={(e) => setSubmitter(e.target.value)} placeholder="匿名" /></label>
          <label><span>备注（可选）</span><input value={note} onChange={(e) => setNote(e.target.value)} placeholder="限速、有效期等" /></label>
          <input className="hp-field" tabIndex={-1} autoComplete="off" value={website} onChange={(e) => setWebsite(e.target.value)} aria-hidden="true" />
          {err && <p className="form-error">{err}</p>}
          <SliderCaptcha onVerified={() => setCaptchaOK(true)} />
          <button className="btn btn-primary" disabled={busy || !captchaOK}>{busy ? '提交并探测中…' : '提交并探测'}</button>
        </form>
        <p className="fine" style={{ marginTop: 10 }}>分享 Key 可能违反部分服务商条款；请自行判断风险。</p>
      </div>
    </div>
  );
}

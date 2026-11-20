import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import { SliderCaptcha } from '../components/SliderCaptcha';
import type { FreeKeyView } from '../types';
import { Spinner } from '../components/ui';
import { useToast } from '../lib/toast';
import { copyText } from '../lib/clipboard';

const STATUS: Record<string, { label: string; cls: string }> = {
  available: { label: '可用', cls: 'ok' },
  invalid: { label: '不可用', cls: 'bad' },
  unknown: { label: '待验证', cls: 'warn' },
};

const fmtDate = (iso: string) => new Date(iso).toLocaleString('zh-CN', { hour12: false });

export function FreeKeysPage() {
  const [items, setItems] = useState<FreeKeyView[] | null>(null);
  const [q, setQ] = useState('');
  const [hideInvalid, setHideInvalid] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const { push } = useToast();

  const [claim, setClaim] = useState<{ apiKey: string; baseUrl: string; models: string; warning: string } | null>(null);
  const [claiming, setClaiming] = useState<number | null>(null);

  const load = () => {
    api.freeKeys(q, !hideInvalid)
      .then((r) => setItems(r.items))
      .catch((e: Error) => setError(e.message));
  };

  useEffect(() => {
    document.title = '免费 API Key 分享：无需登录提交与领取 | RelayRadar';
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hideInvalid]);

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

  return (
    <main className="wrap">
      <div className="crumbs mono"><Link to="/free">免费额度</Link><span>/</span><span className="dim">免费 API Key</span></div>
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />FREE API KEYS</span>
          <h1>免费 API Key 分享</h1>
          <p className="sub">用户提交的 OpenAI 兼容 Key，无需登录即可提交或领取。共享 Key 随时可能被吊销，请勿用于生产或绑定付费。</p>
        </div>
        <button className="btn btn-primary btn-sm" onClick={() => setShowForm(true)}>+ 分享 Key</button>
      </div>

      <div className="tabs">
        <Link className="tab" to="/free">免费模型</Link>
        <span className="tab on">免费 API Key</span>
      </div>

      <div className="toolbar">
        <div className="toolbar-left">
          <div className="search">
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="搜索服务商、站点或模型" aria-label="搜索免费 Key" />
          </div>
          <button className="btn btn-ghost btn-sm" onClick={load}>搜索</button>
        </div>
        <div className="toolbar-right">
          <label className="check-inline">
            <input type="checkbox" checked={hideInvalid} onChange={(e) => setHideInvalid(e.target.checked)} />
            隐藏已失效 Key
          </label>
        </div>
      </div>

      {error && <div className="error-box">{error}</div>}

      <section className="panel">
        <div className="table-wrap">
          <table className="index-table">
            <thead>
              <tr><th>社区判定</th><th>API Key / 状态</th><th>模型</th><th>服务商</th><th>发布人 / 日期</th><th>操作</th></tr>
            </thead>
            <tbody>
              {!items && <tr><td colSpan={6} className="loading"><Spinner /> 加载中…</td></tr>}
              {items && items.length === 0 && (
                <tr><td colSpan={6}><span className="fine">还没有 Key，点击右上角「分享 Key」贡献一个。</span></td></tr>
              )}
              {items?.map((k) => {
                const st = STATUS[k.status] ?? STATUS.unknown;
                return (
                  <tr key={k.id}>
                    <td className="mono dim">{k.claims} 次领取</td>
                    <td>
                      <span className="mono">{k.keyMasked}</span>
                      <span className={`key-status ${st.cls}`}><i />{st.label}</span>
                    </td>
                    <td className="mono dim">{k.models || '—'}</td>
                    <td>
                      <div>{k.provider}</div>
                      <span className="mono dim tiny">{k.baseUrl}</span>
                    </td>
                    <td>
                      <div>{k.submitter || '匿名'}</div>
                      <span className="dim tiny">{fmtDate(k.createdAt)}</span>
                    </td>
                    <td>
                      <button className="btn btn-ghost btn-sm" onClick={() => doClaim(k.id)} disabled={claiming === k.id || k.status === 'invalid'}>
                        {claiming === k.id ? '领取中…' : '领取'}
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel pad tips-panel">
        <h2>使用与提交须知</h2>
        <ul className="limits">
          <li>共享 Key 由用户提交，本站只做一次可用性探测，不保证持续可用，也不对扣费、封号负责。</li>
          <li>不要提交绑定了支付方式、额度较高的主 Key；建议使用临时 Key 或小额度 Key。</li>
          <li>领取后如果可用，请回到列表页确认状态；失效 Key 会被隐藏。</li>
          <li>分享 Key 可能违反部分服务商条款，请自行判断风险。</li>
        </ul>
      </section>

      {showForm && (
        <SubmitModal
          onClose={() => setShowForm(false)}
          onDone={(status) => {
            setShowForm(false);
            push(status === 'available' ? '提交成功，Key 已验证可用' : status === 'invalid' ? '提交成功，但探测显示 Key 不可用' : '提交成功，等待验证', status === 'invalid' ? 'err' : 'ok');
            load();
          }}
        />
      )}

      {claim && (
        <div className="modal-mask" onClick={() => setClaim(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head"><h2>领取成功</h2><button className="mm-close" onClick={() => setClaim(null)}>✕</button></div>
            <p className="fine">请立即复制保存；共享 Key 可能随时被吊销。</p>
            <div className="key-box">
              <span className="mono">{claim.apiKey}</span>
              <button type="button" className="link-btn" onClick={() => void copyText(claim.apiKey).then((ok) => push(ok ? 'API Key 已复制' : '复制失败', ok ? 'ok' : 'err'))}>复制</button>
            </div>
            <div className="key-box">
              <span className="mono">{claim.baseUrl}</span>
              <button type="button" className="link-btn" onClick={() => void copyText(claim.baseUrl).then((ok) => push(ok ? 'Base URL 已复制' : '复制失败', ok ? 'ok' : 'err'))}>复制</button>
            </div>
            {claim.models && <p className="fine mono">模型：{claim.models}</p>}
            <p className="form-error" style={{ background: 'transparent', padding: 0 }}>{claim.warning}</p>
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
          <button className="btn btn-primary" disabled={busy || !captchaOK}>{busy ? '提交并探测中…' : '提交'}</button>
        </form>
        <p className="fine" style={{ marginTop: 10 }}>分享 Key 可能违反部分服务商条款；请自行判断风险。</p>
      </div>
    </div>
  );
}

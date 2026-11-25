import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';

const hostOf = (raw: string) => {
  try {
    return new URL(raw.trim()).host || raw.trim();
  } catch {
    return '';
  }
};

const normalizeUrl = (raw: string) => {
  const v = raw.trim();
  if (!v) return v;
  return /^https?:\/\//i.test(v) ? v : `https://${v}`;
};

export function KeyCheck() {
  const [baseUrl, setBaseUrl] = useState('');
  const [key, setKey] = useState('');
  const [showKey, setShowKey] = useState(false);
  const [loading, setLoading] = useState(false);
  const [urlError, setUrlError] = useState('');
  const [keyError, setKeyError] = useState('');
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);

  const host = useMemo(() => hostOf(baseUrl), [baseUrl]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const url = normalizeUrl(baseUrl);
    const nextUrlError = /^https?:\/\/[^\s/]+/i.test(url) ? '' : '接口地址需要是完整网址，例如 https://api.example.com/v1';
    const nextKeyError = key.trim() ? '' : '请粘贴一个可随时吊销的临时 Key';
    setUrlError(nextUrlError);
    setKeyError(nextKeyError);
    if (nextUrlError || nextKeyError) {
      setResult(null);
      return;
    }
    setLoading(true);
    setResult(null);
    try {
      const r = await api.auditModels(url.replace(/\/$/, ''), key.trim());
      setResult({ ok: true, text: `Key 有效 · ${hostOf(url)} 返回 ${r.count} 个模型` });
    } catch (err) {
      const msg = (err as Error).message || '';
      const h = hostOf(url);
      let text = msg;
      if (msg.includes('401') || msg.includes('403')) text = `Key 被 ${h} 拒绝：检查是否复制完整或已过期`;
      else if (msg.includes('429')) text = `${h} 返回限流（429），稍后再试`;
      else if (/fetch|network|failed/i.test(msg)) text = `无法连接 ${h}：检查地址、网络或站点状态`;
      setResult({ ok: false, text });
    } finally {
      setLoading(false);
    }
  };

  return (
    <form className="keycheck" onSubmit={submit} noValidate>
      <div className="kc-head">
        <b>检测你的 Key</b>
        <span className="mono dim tiny">只存在当前页面内存，不写日志、不落库</span>
      </div>
      <div className="kc-row">
        <div className="kc-field">
          <span className="kc-label">接口地址</span>
          <input
            value={baseUrl}
            onChange={(e) => { setBaseUrl(e.target.value); setUrlError(''); setResult(null); }}
            placeholder="https://api.example.com/v1"
            aria-label="接口地址"
            aria-invalid={!!urlError}
            autoComplete="off"
            spellCheck={false}
          />
          {urlError && <span className="kc-field-err" role="alert">{urlError}</span>}
        </div>
        <div className="kc-field">
          <span className="kc-label">临时 Key</span>
          <div className="kc-key">
            <input
              value={key}
              onChange={(e) => { setKey(e.target.value); setKeyError(''); setResult(null); }}
              placeholder="sk-...（建议用临时 Key）"
              aria-label="API Key"
              aria-invalid={!!keyError}
              type={showKey ? 'text' : 'password'}
              autoComplete="off"
              spellCheck={false}
            />
            <button type="button" className="kc-toggle" onClick={() => setShowKey((v) => !v)} aria-label={showKey ? '隐藏 Key' : '显示 Key'}>
              {showKey ? '隐藏' : '显示'}
            </button>
            {key && (
              <button type="button" className="kc-toggle" onClick={() => setKey('')} aria-label="清空 Key">
                清空
              </button>
            )}
          </div>
          {keyError && <span className="kc-field-err" role="alert">{keyError}</span>}
        </div>
        <button className="btn btn-primary" disabled={loading}>{loading ? '检测中…' : '开始检测'}</button>
      </div>
      <p className="kc-dest mono" role="status">
        {host ? <>将用你的 Key 向 <b>{host}</b> 发起 1 次只读请求（拉取模型列表）</> : '填写接口地址后，这里会显示请求发往哪个站点'}
      </p>
      {result && (
        <p className={`kc-result ${result.ok ? 'ok' : 'err'}`} role="status">
          {result.text}
          {result.ok && (
            <Link className="kc-more" to={`/audit?url=${encodeURIComponent(normalizeUrl(baseUrl))}`}>
              继续做完整检测（8 项）→
            </Link>
          )}
        </p>
      )}
    </form>
  );
}

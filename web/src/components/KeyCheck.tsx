import { useState } from 'react';
import { api } from '../api';

export function KeyCheck() {
  const [baseUrl, setBaseUrl] = useState('https://api.mhoo.cc/v1');
  const [key, setKey] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!key.trim()) return;
    setLoading(true);
    setResult(null);
    try {
      const r = await api.auditModels(baseUrl.trim().replace(/\/$/, ''), key.trim());
      setResult({ ok: true, text: `Key 有效 · 拉取到 ${r.count} 个模型` });
    } catch (err) {
      setResult({ ok: false, text: (err as Error).message });
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="keycheck" onSubmit={submit}>
      <div className="kc-head">
        <b>检测你的 Key</b>
        <span className="mono dim tiny">不会保存，只做一次请求</span>
      </div>
      <div className="kc-row">
        <input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://api.example.com/v1" aria-label="接口地址" />
        <input value={key} onChange={(e) => setKey(e.target.value)} placeholder="sk-..." aria-label="API Key" type="password" />
        <button className="btn btn-primary" disabled={loading}>{loading ? '检测中…' : '开始检测'}</button>
      </div>
      {result && <p className={`kc-result ${result.ok ? 'ok' : 'err'}`}>{result.text}</p>}
    </form>
  );
}

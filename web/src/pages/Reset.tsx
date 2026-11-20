import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';

export function ResetPage() {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (password !== confirm) {
      setError('两次输入的密码不一致');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await api.resetPassword(token, password);
      setDone(true);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="wrap">
      <section className="panel auth-card">
        <span className="eyebrow"><i />ACCOUNT</span>
        <h1>重置密码</h1>
        {!token && <p className="form-error">链接缺少 token，请从邮件里重新打开。</p>}
        {done ? (
          <p className="sub" style={{ marginTop: 18 }}>密码已更新，所有旧会话已失效，请重新登录。</p>
        ) : (
          <form className="auth-form" onSubmit={submit}>
            <label><span>新密码</span><input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} required autoComplete="new-password" /></label>
            <label><span>确认新密码</span><input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} minLength={8} required autoComplete="new-password" /></label>
            {error && <p className="form-error">{error}</p>}
            <button className="btn btn-primary" disabled={busy || !token}>{busy ? '提交中…' : '更新密码'}</button>
          </form>
        )}
        <p className="fine"><Link to="/login">返回登录</Link></p>
      </section>
    </main>
  );
}

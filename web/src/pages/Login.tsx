import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../lib/auth';

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const u = await login(email.trim(), password);
      navigate(u.role === 'admin' ? '/admin' : '/', { replace: true });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="wrap">
      <section className="panel auth-card">
        <h1>登录</h1>
        <p className="sub">登录后可提交站点、管理告警订阅；管理员可进入后台。</p>
        <form className="auth-form" onSubmit={submit}>
          <label><span>邮箱</span><input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" /></label>
          <label><span>密码</span><input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" /></label>
          {error && <p className="form-error">{error}</p>}
          <button className="btn btn-primary" disabled={busy}>{busy ? '登录中…' : '登录'}</button>
        </form>
        <p className="fine">还没有账号？<Link to="/register">注册</Link> · <Link to="/forgot">忘记密码</Link></p>
      </section>
    </main>
  );
}

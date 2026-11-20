import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../lib/auth';

export function RegisterPage() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const u = await register(email.trim(), password, name.trim());
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
        <span className="eyebrow"><i />ACCOUNT</span>
        <h1>注册</h1>
        <p className="sub">账号只用于提交与订阅管理，不参与评分；密码至少 8 位。注册后会发送验证邮件（需服务端配置 SMTP）。</p>
        <form className="auth-form" onSubmit={submit}>
          <label><span>昵称（可选）</span><input value={name} onChange={(e) => setName(e.target.value)} maxLength={60} autoComplete="nickname" /></label>
          <label><span>邮箱</span><input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" /></label>
          <label><span>密码</span><input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} maxLength={128} required autoComplete="new-password" /></label>
          {error && <p className="form-error">{error}</p>}
          <button className="btn btn-primary" disabled={busy}>{busy ? '注册中…' : '创建账号'}</button>
        </form>
        <p className="fine">已有账号？<Link to="/login">登录</Link></p>
      </section>
    </main>
  );
}

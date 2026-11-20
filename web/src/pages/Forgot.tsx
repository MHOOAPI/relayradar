import { useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import { SliderCaptcha } from '../components/SliderCaptcha';

export function ForgotPage() {
  const [email, setEmail] = useState('');
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [captchaOK, setCaptchaOK] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await api.forgotPassword(email.trim());
      setSent(true);
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="wrap">
      <section className="panel auth-card">
        <h1>找回密码</h1>
        <p className="sub">输入注册邮箱，如果账号存在，会收到一封 1 小时内有效的重置链接。</p>
        {sent ? (
          <p className="sub" style={{ marginTop: 18 }}>如果该邮箱已注册，重置邮件已经发出，请查收（含垃圾箱）。</p>
        ) : (
          <form className="auth-form" onSubmit={submit}>
            <label><span>邮箱</span><input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" /></label>
            <SliderCaptcha onVerified={() => setCaptchaOK(true)} />
            <button className="btn btn-primary" disabled={busy || !captchaOK}>{busy ? '发送中…' : '发送重置链接'}</button>
          </form>
        )}
        <p className="fine"><Link to="/login">返回登录</Link></p>
      </section>
    </main>
  );
}

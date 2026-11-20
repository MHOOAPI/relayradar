import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api';
import { Spinner } from '../components/ui';

export function VerifyPage() {
  const [params] = useSearchParams();
  const token = params.get('token') ?? '';
  const [state, setState] = useState<'loading' | 'ok' | 'error'>('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setState('error');
      setMessage('验证链接缺少 token');
      return;
    }
    api.verifyEmail(token)
      .then(() => setState('ok'))
      .catch((e: Error) => {
        setState('error');
        setMessage(e.message);
      });
  }, [token]);

  return (
    <main className="wrap">
      <section className="panel auth-card">
        <h1>邮箱验证</h1>
        {state === 'loading' && <p className="sub"><Spinner /> 正在验证…</p>}
        {state === 'ok' && <p className="sub">邮箱已验证，可以正常使用账号。</p>}
        {state === 'error' && <p className="form-error">{message}</p>}
        <p className="fine" style={{ marginTop: 18 }}>
          <Link className="btn btn-primary btn-sm" to="/login">去登录</Link>
        </p>
      </section>
    </main>
  );
}

import { useEffect, useRef, useState } from 'react';
import { api, setCaptchaToken } from '../api';

interface Challenge {
  disabled?: boolean;
  id: string;
  background: string;
  piece: string;
  width: number;
  height: number;
  pieceSize: number;
  pieceY: number;
  maxX: number;
}

type Status = 'loading' | 'idle' | 'dragging' | 'verifying' | 'done' | 'error';

// SliderCaptcha solves the server-drawn puzzle and stores the token that
// sensitive requests send as X-Captcha-Token.
export function SliderCaptcha({ onVerified, hint = '按住滑块，把缺口拖到阴影处' }: {
  onVerified?: (token: string) => void;
  hint?: string;
}) {
  const [challenge, setChallenge] = useState<Challenge | null>(null);
  const [x, setX] = useState(0);
  const [status, setStatus] = useState<Status>('loading');
  const [error, setError] = useState('');
  const drag = useRef<{ startX: number; startLeft: number } | null>(null);

  const load = async () => {
    setStatus('loading');
    setError('');
    setX(0);
    try {
      const c = await api.captcha();
      if (c.disabled) {
        // Server-side captcha is off (CAPTCHA_ENABLED=false): treat as passed.
        setStatus('done');
        onVerified?.('disabled');
        return;
      }
      setChallenge(c);
      setStatus('idle');
    } catch (err) {
      setError((err as Error).message);
      setStatus('error');
    }
  };

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const move = (clientX: number) => {
    if (!drag.current || !challenge) return;
    const next = Math.max(0, Math.min(challenge.maxX, drag.current.startLeft + (clientX - drag.current.startX)));
    setX(next);
  };

  const start = (e: React.PointerEvent) => {
    if (!challenge || status === 'done' || status === 'verifying') return;
    drag.current = { startX: e.clientX, startLeft: x };
    setStatus('dragging');
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  };

  const finish = async () => {
    if (!drag.current || !challenge) return;
    drag.current = null;
    setStatus('verifying');
    try {
      const r = await api.verifyCaptcha(challenge.id, x);
      setCaptchaToken(r.token);
      setStatus('done');
      onVerified?.(r.token);
    } catch (err) {
      setError((err as Error).message);
      setStatus('error');
      await load();
    }
  };

  if (status === 'loading') {
    return <div className="slider-captcha"><span className="cap-hint">加载验证码…</span></div>;
  }
  if (!challenge || status === 'error' && !challenge) {
    return (
      <div className="slider-captcha">
        <span className="cap-hint err">{error || '验证码加载失败'}</span>
        <button type="button" className="link-btn mono" onClick={() => void load()}>重试</button>
      </div>
    );
  }

  return (
    <div className={`slider-captcha ${status}`}>
      <div className="cap-image" style={{ width: challenge.width, height: challenge.height }}>
        <img src={challenge.background} alt="" width={challenge.width} height={challenge.height} draggable={false} />
        <img
          className="cap-piece"
          src={challenge.piece}
          alt=""
          width={challenge.pieceSize}
          height={challenge.pieceSize}
          draggable={false}
          style={{ left: x, top: challenge.pieceY }}
        />
      </div>
      <div className="cap-track">
        <div className="cap-fill" style={{ width: x + challenge.pieceSize }} />
        <button
          type="button"
          className="cap-knob"
          style={{ left: x }}
          aria-label="拖动滑块完成验证"
          onPointerDown={start}
          onPointerMove={(e) => move(e.clientX)}
          onPointerUp={() => void finish()}
          onPointerCancel={() => void finish()}
        >
          ⟷
        </button>
        {status === 'done' ? (
          <span className="cap-hint ok">验证通过</span>
        ) : (
          <span className="cap-hint">{status === 'error' ? error : hint}</span>
        )}
      </div>
      {status === 'done' && (
        <button type="button" className="link-btn mono cap-reset" onClick={() => void load()}>重新验证</button>
      )}
    </div>
  );
}

import type { ReactNode } from 'react';
import { PAY_PATHS } from './payPaths';

const norm = (m: string) => m.trim().toLowerCase();

function Tile({ color, size, children }: { color: string; size: number; children: ReactNode }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <rect width="24" height="24" rx="6" fill={color} />
      {children}
    </svg>
  );
}

function Brand({ color, path, size }: { color: string; path: string; size: number }) {
  return (
    <Tile color={color} size={size}>
      <g transform="translate(3.6 3.6) scale(0.7)">
        <path d={path} fill="#fff" />
      </g>
    </Tile>
  );
}

/** Official payment brand marks (Simple Icons glyphs on brand-color tiles). */
export function PayMark({ method, size = 16 }: { method: string; size?: number }) {
  const m = norm(method);

  if (m.includes('支付宝') || m.includes('alipay')) {
    return <Brand color="#1677FF" path={PAY_PATHS.alipay} size={size} />;
  }
  if (m.includes('微信') || m.includes('wechat')) {
    return <Brand color="#07C160" path={PAY_PATHS.wechat} size={size} />;
  }
  if (m.includes('usdt') || m.includes('tether') || m.includes('trc20')) {
    return <Brand color="#26A17B" path={PAY_PATHS.tether} size={size} />;
  }
  if (m.includes('stripe')) {
    return <Brand color="#635BFF" path={PAY_PATHS.stripe} size={size} />;
  }
  if (m.includes('visa')) {
    return <Brand color="#1A1F71" path={PAY_PATHS.visa} size={size} />;
  }
  if (m.includes('mastercard') || m.includes('万事达')) {
    return <Brand color="#EB001B" path={PAY_PATHS.mastercard} size={size} />;
  }
  if (m.includes('信用卡') || m.includes('card')) {
    return (
      <Tile color="#2B3646" size={size}>
        <rect x="4.6" y="7.4" width="14.8" height="9.6" rx="2" fill="#fff" />
        <rect x="4.6" y="10.4" width="14.8" height="1.8" fill="#2B3646" />
        <rect x="7.2" y="13.6" width="4.2" height="1.3" rx=".65" fill="#2B3646" opacity=".5" />
      </Tile>
    );
  }
  if (m.includes('对公') || m.includes('银行') || m.includes('bank') || m.includes('transfer')) {
    return (
      <Tile color="#3C4B5D" size={size}>
        <path d="M12 4.6 19 9H5l7-4.4Z" fill="#fff" />
        <path d="M6.4 10.6v5.2M9.6 10.6v5.2M12.4 10.6v5.2M15.6 10.6v5.2M18 10.6v5.2" stroke="#fff" strokeWidth="1.3" strokeLinecap="round" />
        <path d="M4.6 18.2h14.8" stroke="#fff" strokeWidth="1.3" strokeLinecap="round" />
      </Tile>
    );
  }
  if (m.includes('高危') || m.includes('个人码') || m.includes('warn')) {
    return (
      <Tile color="#8A5A1F" size={size}>
        <path d="M12 5.2 19 17.5H5L12 5.2Z" fill="none" stroke="#fff" strokeWidth="1.6" strokeLinejoin="round" />
        <path d="M12 10v3.4" stroke="#fff" strokeWidth="1.6" strokeLinecap="round" />
        <circle cx="12" cy="15.4" r=".9" fill="#fff" />
      </Tile>
    );
  }
  return (
    <Tile color="#3A4656" size={size}>
      <circle cx="12" cy="12" r="6.2" fill="none" stroke="#fff" strokeWidth="1.5" />
      <path d="M9.4 12h5.2" stroke="#fff" strokeWidth="1.5" strokeLinecap="round" />
    </Tile>
  );
}

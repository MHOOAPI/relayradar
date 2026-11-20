import type { PurityTier } from '../types';
import { useWatchlist } from '../lib/watchlist';
import { StarMark } from './icons';

export const fmtCny = (n: number, digits = 2) => (n < 0 ? '—' : `¥${n.toFixed(digits)}`);
export const fmtNum = (n: number) => n.toLocaleString('zh-CN');
export const pct = (n: number) => `${n.toFixed(1)}%`;
export const fmtDate = (iso: string) =>
  new Date(iso).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' });
export const fmtDateTime = (iso: string) =>
  new Date(iso).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });

export const TIER_LABEL: Record<PurityTier, string> = {
  official: '官方直连',
  mixed: '混合线路',
  counterfeit: '疑似掺假',
  unknown: '未检测',
};

export function TierBadge({ tier, score }: { tier: PurityTier; score?: number }) {
  return (
    <span className={`tier tier-${tier}`}>
      <i className="tier-dot" />
      {TIER_LABEL[tier]}
      {typeof score === 'number' && score > 0 && <b>{score.toFixed(0)}</b>}
    </span>
  );
}

export function UptimeBars({ data, width = 4 }: { data: number[]; width?: number }) {
  if (!data || data.length === 0) return null;
  return (
    <span className="uptime" aria-label="近 14 天可用率">
      {data.map((v, i) => {
        const cls = v < 0 ? 'none' : v >= 97 ? 'ok' : v >= 88 ? 'warn' : 'bad';
        return <i key={i} className={`uptime-bar ${cls}`} style={{ width }} title={v < 0 ? '无数据' : `${v}%`} />;
      })}
    </span>
  );
}

const HUES = [
  ['#12362b', '#2ee6a6'],
  ['#1b2b45', '#6aa8ff'],
  ['#3a2418', '#f5b942'],
  ['#2c1f3d', '#b48cff'],
  ['#3b1f28', '#ff7a9c'],
  ['#1d3540', '#53d7e8'],
];

export function Avatar({ name, size = 34 }: { name: string; size?: number }) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  const [bg, fg] = HUES[h % HUES.length];
  const letter = name.replace(/[^\p{L}\p{N}]/gu, '').slice(0, 1).toUpperCase() || '?';
  return (
    <span
      className="avatar"
      style={{
        width: size,
        height: size,
        background: `linear-gradient(145deg, ${bg}, #0a0d12)`,
        color: fg,
        fontSize: size * 0.42,
        borderColor: `${fg}33`,
      }}
    >
      {letter}
    </span>
  );
}

export function Stars({ value }: { value: number }) {
  return (
    <span className="stars" title={`${value} / 5`}>
      {Array.from({ length: 5 }).map((_, i) => (
        <i key={i} className={i < Math.round(value) ? 'on' : ''}>
          ★
        </i>
      ))}
    </span>
  );
}

export function Logo({ size = 26 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <circle cx="16" cy="16" r="13.5" stroke="#2ee6a6" strokeOpacity="0.35" />
      <circle cx="16" cy="16" r="7.5" stroke="#2ee6a6" strokeOpacity="0.55" />
      <path d="M16 16 L25.5 8.5" stroke="#2ee6a6" strokeWidth="1.8" strokeLinecap="round" />
      <circle cx="16" cy="16" r="2.2" fill="#2ee6a6" />
      <circle cx="25.5" cy="8.5" r="2" fill="#2ee6a6" />
    </svg>
  );
}

export function Arrow() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M3 8h9M8.5 4.5 12 8l-3.5 3.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function External() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M6 3h7v7M13 3 5 11" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function Spinner() {
  return <span className="spinner" aria-label="加载中" />;
}

export function WatchStar({ slug, size = 15 }: { slug: string; size?: number }) {
  const { has, toggle } = useWatchlist();
  const on = has(slug);
  return (
    <button
      type="button"
      className={`watch${on ? ' on' : ''}`}
      style={{ fontSize: size }}
      title={on ? '取消关注' : '关注该站点'}
      aria-label={on ? '取消关注' : '关注该站点'}
      aria-pressed={on}
      onClick={(e) => {
        e.stopPropagation();
        toggle(slug);
      }}
    >
      <StarMark size={size} filled={on} />
    </button>
  );
}
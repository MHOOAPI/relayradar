import { useEffect, useState } from 'react';
import { VENDOR_PATHS } from './vendorPaths';

const VENDOR_SLUG: Record<string, string> = {
  anthropic: 'anthropic',
  openai: 'openai',
  google: 'googlegemini',
  deepseek: 'deepseek',
  moonshot: 'moonshotai',
  kimi: 'moonshotai',
  qwen: 'qwen',
  meta: 'meta',
  mistral: 'mistralai',
  xai: 'x',
};

const VENDOR_COLOR: Record<string, string> = {
  openai: '#e8edf5',
  anthropic: '#d97757',
  google: '#8ab4f8',
  deepseek: '#5b7cff',
  moonshot: '#a78bfa',
  kimi: '#a78bfa',
  qwen: '#8b7bff',
  meta: '#4d8dff',
  mistral: '#ff8a3d',
  xai: '#e8edf5',
};

export function VendorMark({ vendor, size = 26 }: { vendor: string; size?: number }) {
  const slug = VENDOR_SLUG[vendor];
  const color = VENDOR_COLOR[vendor] ?? '#9fb0c7';
  if (!slug || !VENDOR_PATHS[slug]) {
    return (
      <span className="vendor-mark vendor-fallback" style={{ width: size, height: size, color }} title={vendor}>
        <b>{vendor.slice(0, 1).toUpperCase()}</b>
      </span>
    );
  }
  return (
    <span className="vendor-mark" style={{ width: size, height: size }} title={vendor}>
      <svg viewBox="0 0 24 24" width={size * 0.68} height={size * 0.68} aria-hidden="true">
        <path d={VENDOR_PATHS[slug]} fill={color} />
      </svg>
    </span>
  );
}
function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i += 1) h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
  return Math.abs(h);
}

function initialsOf(name: string) {
  const cleaned = name.trim();
  if (!cleaned) return '?';
  const two = Array.from(cleaned).slice(0, 2).join('');
  return two.toUpperCase();
}

export function StationMark({
  slug,
  name,
  iconUrl,
  size = 34,
}: {
  slug: string;
  name: string;
  iconUrl?: string;
  size?: number;
}) {
  const [failed, setFailed] = useState(false);
  const key = slug || name;
  // Stations without a hand-picked icon fall back to the favicon proxy, which
  // serves the site's own icon; when that misses, show the station's initials
  // rather than a random glyph that could pass for a logo.
  const src = iconUrl || (slug ? `/api/v1/favicon/${slug}` : '');
  const h = hash(key);
  const hue = (h * 137) % 360;
  const fg = `hsl(${hue} 70% 74%)`;
  const bg = `hsl(${hue} 45% 11%)`;
  const border = `hsl(${hue} 45% 26%)`;

  useEffect(() => setFailed(false), [src]);

  return (
    <span
      className="station-mark"
      style={{
        width: size,
        height: size,
        background: `radial-gradient(120% 120% at 20% 0%, ${bg} 0%, #0a0d12 78%)`,
        borderColor: border,
      }}
      title={name}
      aria-label={name}
    >
      <b className="station-mark-initials" style={{ color: fg, fontSize: Math.round(size * 0.4) }} aria-hidden="true">
        {initialsOf(name)}
      </b>
      {src && !failed && (
        <img
          className="station-mark-img"
          src={src}
          alt=""
          loading="lazy"
          onError={() => setFailed(true)}
        />
      )}
    </span>
  );
}
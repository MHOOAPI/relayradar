import { useMemo } from 'react';

// Decorative, data-derived backdrops for detail pages. Always aria-hidden and
// pointer-events: none; the content above keeps its own stacking context.

export function WaveBackdrop({ data, onPick }: { data: number[]; onPick?: () => void }) {
  const points = useMemo(() => {
    if (!data || data.length < 2) return null;
    return data.map((v, i) => {
      const x = (i / (data.length - 1)) * 100;
      const clamped = v < 0 ? 50 : Math.max(0, Math.min(100, v));
      return { x, y: 96 - (clamped / 100) * 62, v, i };
    });
  }, [data]);

  if (!points) return null;
  const pts = points.map((p) => `${p.x.toFixed(2)},${p.y.toFixed(2)}`).join(' L');
  return (
    <svg className="wave-backdrop" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="wave-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="rgba(46,230,166,0.10)" />
          <stop offset="100%" stopColor="rgba(46,230,166,0)" />
        </linearGradient>
      </defs>
      <path d={`M0,100 L${pts} L100,100 Z`} fill="url(#wave-fill)" />
      <path
        className="wave-line"
        d={`M${pts}`}
        fill="none"
        stroke="rgba(46,230,166,0.32)"
        strokeWidth="0.35"
        vectorEffect="non-scaling-stroke"
      />
      {points.map((p) => (
        <circle
          key={p.i}
          className="wave-point"
          cx={p.x}
          cy={p.y}
          r={2.4}
          fill="transparent"
          pointerEvents="all"
          onClick={onPick}
          role={onPick ? 'button' : undefined}
          tabIndex={onPick ? 0 : undefined}
          onKeyDown={(e) => {
            if (!onPick) return;
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onPick();
            }
          }}
        >
          <title>
            {p.v < 0 ? `近 14 天第 ${p.i + 1} 天 · 无数据` : `近 14 天第 ${p.i + 1} 天 · ${p.v}%`}
            {onPick ? ' · 点击看 30 天趋势' : ''}
          </title>
        </circle>
      ))}
    </svg>
  );
}

export type PriceSample = { price: number; multiplier: number; station?: string; slug?: string; tier?: string; rank?: number };

const TIER_RGB: Record<string, string> = {
  official: '46,230,166',
  mixed: '106,168,255',
  unknown: '86,164,150',
  counterfeit: '214,120,96',
};

const TIER_LABEL: Record<string, string> = {
  official: '官方直连',
  mixed: '混合线路',
  unknown: '未检测',
  counterfeit: '疑似掺假',
};

export function PriceTexture({
  offers,
  onPick,
}: {
  offers: PriceSample[];
  onPick?: (slug: string) => void;
}) {
  const bars = useMemo(() => {
    const list = offers.filter((o) => o.price > 0).sort((a, b) => a.price - b.price);
    if (list.length === 0) return [];
    const step = Math.max(1, Math.floor(list.length / 72));
    const picked: PriceSample[] = [];
    for (let i = 0; i < list.length; i += step) picked.push(list[i]);
    const mults = picked.map((o) => o.multiplier).filter((m) => m > 0);
    const minMult = mults.length ? Math.min(...mults) : 1;
    const maxMult = mults.length ? Math.max(...mults) : 1;
    const cheap = picked[0]?.price ?? 0;
    return picked.map((o, i) => {
      const norm = maxMult > minMult ? (o.multiplier - minMult) / (maxMult - minMult) : 0.5;
      const rankBonus = o.rank && o.rank > 0 ? Math.max(0, 1 - Math.min(o.rank, 300) / 300) * 0.08 : 0;
      const alpha = 0.06 + (1 - norm) * 0.16 + rankBonus;
      return {
        x: i,
        h: 8 + (Math.log10(o.price + 1) / Math.log10(cheap * 4 + 1)) * 88,
        alpha: o.price === cheap ? 0.34 : alpha,
        slug: o.slug,
        rgb: TIER_RGB[o.tier ?? 'unknown'] ?? TIER_RGB.unknown,
        label: `¥${o.price.toFixed(2)}/M · ${o.multiplier.toFixed(2)}×${o.station ? ` · ${o.station}` : ''}${o.tier ? ` · ${TIER_LABEL[o.tier] ?? o.tier}` : ''}${o.rank ? ` · #${o.rank}` : ''}`,
      };
    });
  }, [offers]);

  if (bars.length === 0) return null;
  return (
    <svg className="price-texture" viewBox={`0 0 ${bars.length} 100`} preserveAspectRatio="none" aria-hidden="true">
      {bars.map((b) => (
        <g key={b.x}>
          <rect
            x={b.x + 0.2}
            y={100 - Math.min(100, b.h)}
            width={0.6}
            height={Math.min(100, b.h)}
            fill={`rgba(${b.rgb},${b.alpha.toFixed(2)})`}
          />
          <rect
            className="price-hit"
            x={b.x}
            y={0}
            width={1}
            height={100}
            fill="transparent"
            pointerEvents="all"
            onClick={() => b.slug && onPick?.(b.slug)}
            role={b.slug && onPick ? 'button' : undefined}
            tabIndex={b.slug && onPick ? 0 : undefined}
            onKeyDown={(e) => {
              if (!b.slug || !onPick) return;
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onPick(b.slug);
              }
            }}
          >
            <title>{b.label}{b.slug && onPick ? ' · 点击查看该站点' : ''}</title>
          </rect>
        </g>
      ))}
    </svg>
  );
}

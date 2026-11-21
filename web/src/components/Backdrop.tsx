import { useMemo } from 'react';

// Decorative, data-derived backdrops for detail pages. Always aria-hidden and
// pointer-events: none; the content above keeps its own stacking context.

export function WaveBackdrop({ data }: { data: number[] }) {
  const path = useMemo(() => {
    if (!data || data.length < 2) return null;
    const pts = data.map((v, i) => {
      const x = (i / (data.length - 1)) * 100;
      const clamped = v < 0 ? 50 : Math.max(0, Math.min(100, v));
      const y = 96 - (clamped / 100) * 62;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    });
    return {
      line: `M${pts.join(' L')}`,
      area: `M0,100 L${pts.join(' L')} L100,100 Z`,
    };
  }, [data]);

  if (!path) return null;
  return (
    <svg className="wave-backdrop" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="wave-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="rgba(46,230,166,0.10)" />
          <stop offset="100%" stopColor="rgba(46,230,166,0)" />
        </linearGradient>
      </defs>
      <path d={path.area} fill="url(#wave-fill)" />
      <path
        className="wave-line"
        d={path.line}
        fill="none"
        stroke="rgba(46,230,166,0.32)"
        strokeWidth="0.35"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

export type PriceSample = { price: number; multiplier: number };

export function PriceTexture({ offers }: { offers: PriceSample[] }) {
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
      const alpha = 0.06 + (1 - norm) * 0.16;
      return {
        x: i,
        h: 8 + (Math.log10(o.price + 1) / Math.log10(cheap * 4 + 1)) * 88,
        alpha: o.price === cheap ? 0.34 : alpha,
      };
    });
  }, [offers]);

  if (bars.length === 0) return null;
  return (
    <svg className="price-texture" viewBox={`0 0 ${bars.length} 100`} preserveAspectRatio="none" aria-hidden="true">
      {bars.map((b) => (
        <rect
          key={b.x}
          x={b.x + 0.2}
          y={100 - Math.min(100, b.h)}
          width={0.6}
          height={Math.min(100, b.h)}
          fill={`rgba(46,230,166,${b.alpha.toFixed(2)})`}
        />
      ))}
    </svg>
  );
}

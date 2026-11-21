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
      <path d={path.line} fill="none" stroke="rgba(46,230,166,0.32)" strokeWidth="0.35" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

export function PriceTexture({ prices }: { prices: number[] }) {
  const bars = useMemo(() => {
    const values = prices.filter((p) => p > 0).sort((a, b) => a - b);
    if (values.length === 0) return [];
    const max = values[values.length - 1];
    const step = Math.max(1, Math.floor(values.length / 72));
    const picked: number[] = [];
    for (let i = 0; i < values.length; i += step) picked.push(values[i]);
    return picked.map((v, i) => ({
      x: i,
      h: 8 + (Math.log10(v + 1) / Math.log10(max + 1)) * 88,
      strong: i % 7 === 0,
    }));
  }, [prices]);

  if (bars.length === 0) return null;
  return (
    <svg className="price-texture" viewBox={`0 0 ${bars.length} 100`} preserveAspectRatio="none" aria-hidden="true">
      {bars.map((b) => (
        <rect
          key={b.x}
          x={b.x + 0.2}
          y={100 - b.h}
          width={0.6}
          height={b.h}
          fill={b.strong ? 'rgba(46,230,166,0.20)' : 'rgba(46,230,166,0.08)'}
        />
      ))}
    </svg>
  );
}

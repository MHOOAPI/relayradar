import { useState } from 'react';

type Props = {
  values: number[];
  labels: string[];
  color?: string;
  height?: number;
  min?: number;
  max?: number;
  unit?: string;
  selected?: number | null;
  onSelect?: (index: number) => void;
};

/** Dependency-free SVG line chart with a soft area fill and hover readout. */
export function TrendChart({ values, labels, color = '#2ee6a6', height = 180, min, max, unit = '', selected = null, onSelect }: Props) {
  const [hover, setHover] = useState<number | null>(null);
  const W = 720;
  const H = height;
  const padX = 8;
  const padY = 14;

  if (values.length < 2) {
    return <div className="trend-empty">数据不足，至少需要 2 天记录</div>;
  }

  const lo = min ?? Math.min(...values);
  const hi = max ?? Math.max(...values);
  const span = hi - lo || 1;
  const x = (i: number) => padX + (i / (values.length - 1)) * (W - padX * 2);
  const y = (v: number) => padY + (1 - (v - lo) / span) * (H - padY * 2);
  const step = (W - padX * 2) / (values.length - 1);

  const line = values.map((v, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(v).toFixed(1)}`).join(' ');
  const area = `${line} L${x(values.length - 1).toFixed(1)},${H - padY} L${x(0).toFixed(1)},${H - padY} Z`;
  const gid = `grad-${color.replace('#', '')}`;

  const hoverX = hover === null ? 0 : x(hover);
  const hoverY = hover === null ? 0 : y(values[hover]);
  const tipX = Math.max(padX + 44, Math.min(W - padX - 44, hoverX));
  const tipY = Math.max(22, hoverY - 16);

  return (
    <div className="trend-chart">
      <svg viewBox={`0 0 ${W} ${H}`} width="100%" height={H} role="img" aria-label="趋势图">
        <defs>
          <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.28" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        {[0, 0.5, 1].map((t) => (
          <line
            key={t}
            x1={padX}
            x2={W - padX}
            y1={padY + t * (H - padY * 2)}
            y2={padY + t * (H - padY * 2)}
            stroke="#1a1f28"
            strokeDasharray={t === 1 ? '0' : '3 4'}
          />
        ))}
        <path d={area} fill={`url(#${gid})`} />
        <path d={line} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
        {hover !== null && (
          <g pointerEvents="none">
            <line x1={hoverX} x2={hoverX} y1={padY} y2={H - padY} stroke="#33404f" strokeDasharray="2 3" />
            <circle cx={hoverX} cy={hoverY} r="4.6" fill={color} stroke="#06070a" strokeWidth="2" />
            <rect x={tipX - 44} y={tipY - 15} width="88" height="26" rx="6" fill="#0b1117" stroke="#26313d" />
            <text x={tipX} y={tipY + 3} textAnchor="middle" fontSize="11.5" fill="#e9edf4">
              {values[hover].toFixed(1)}{unit}
            </text>
          </g>
        )}
        {values.map((v, i) => (
          <g key={i}>
            <circle cx={x(i)} cy={y(v)} r={hover === i || selected === i ? 4.2 : 2.4} fill={color} opacity={hover === null || hover === i || selected === i ? 1 : 0.45} />
            <rect
              x={x(i) - step / 2}
              y={0}
              width={step}
              height={H}
              fill="transparent"
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(null)}
              onClick={() => onSelect?.(i)}
              style={{ cursor: onSelect ? 'pointer' : 'default' }}
            />
          </g>
        ))}
      </svg>
      <div className="trend-axis mono">
        <span>{labels[0]}</span>
        <span className="trend-range">
          {hover !== null ? `${labels[hover]} · ${values[hover].toFixed(1)}${unit}` : `${lo.toFixed(1)}${unit} – ${hi.toFixed(1)}${unit}`}
        </span>
        <span>{labels[labels.length - 1]}</span>
      </div>
    </div>
  );
}

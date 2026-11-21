import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Station } from '../types';
import { fmtCny } from './ui';

const TIER_COLOR: Record<string, string> = {
  official: '#2ee6a6',
  mixed: '#6aa8ff',
  unknown: '#8b98ab',
  counterfeit: '#e07a5f',
};

const TIER_LABEL: Record<string, string> = {
  official: '官方直连',
  mixed: '混合线路',
  unknown: '未检测',
  counterfeit: '疑似掺假',
};

const W = 760;
const H = 380;
const PAD_L = 52;
const PAD_R = 20;
const PAD_T = 26;
const PAD_B = 42;

/** Score vs cheapest input price scatter, log-scaled on price. */
export function Quadrant({ stations }: { stations: Station[] }) {
  const navigate = useNavigate();
  const [hover, setHover] = useState<number | null>(null);

  const chart = useMemo(() => {
    const pts = stations.filter((s) => s.fromPriceCny > 0 && s.rankScore > 0);
    if (pts.length < 3) return null;
    const prices = pts.map((s) => s.fromPriceCny);
    const lo = Math.log10(Math.min(...prices));
    const hi = Math.log10(Math.max(...prices));
    const span = hi - lo || 1;
    const scores = pts.map((s) => s.rankScore);
    const smin = Math.max(0, Math.min(...scores) - 4);
    const smax = 100;
    const x = (v: number) => PAD_L + ((Math.log10(v) - lo) / span) * (W - PAD_L - PAD_R);
    const y = (v: number) => H - PAD_B - ((v - smin) / (smax - smin)) * (H - PAD_T - PAD_B);
    const sortedPrices = [...prices].sort((a, b) => a - b);
    const medianPrice = sortedPrices[Math.floor(sortedPrices.length / 2)];
    const sortedScores = [...scores].sort((a, b) => a - b);
    const medianScore = sortedScores[Math.floor(sortedScores.length / 2)];
    return { pts, x, y, lo, hi, smin, medianPrice, medianScore };
  }, [stations]);

  if (!chart) {
    return <div className="quadrant-empty">至少需要 3 家有报价的站点才能绘制象限图。</div>;
  }
  const { pts, x, y, medianPrice, medianScore } = chart;
  const active = hover !== null ? pts[hover] : null;
  const activeX = active ? x(active.fromPriceCny) : 0;
  const activeY = active ? y(active.rankScore) : 0;

  return (
    <div className="quadrant">
      <div className="quadrant-legend mono dim">
        {Object.entries(TIER_LABEL).map(([k, label]) => (
          <span key={k}><i style={{ background: TIER_COLOR[k] }} />{label}</span>
        ))}
        <span className="quadrant-hint">左上角 = 综合分高、价格低</span>
      </div>
      <div className="quadrant-canvas">
        <svg viewBox={`0 0 ${W} ${H}`} role="img" aria-label="站点综合分与最低价象限图">
          {[0.25, 0.5, 0.75].map((t) => (
            <line key={`gx${t}`} x1={PAD_L + t * (W - PAD_L - PAD_R)} x2={PAD_L + t * (W - PAD_L - PAD_R)} y1={PAD_T} y2={H - PAD_B} stroke="#1a2029" strokeWidth="1" />
          ))}
          {[0.25, 0.5, 0.75].map((t) => (
            <line key={`gy${t}`} x1={PAD_L} x2={W - PAD_R} y1={PAD_T + t * (H - PAD_T - PAD_B)} y2={PAD_T + t * (H - PAD_T - PAD_B)} stroke="#1a2029" strokeWidth="1" />
          ))}
          <line x1={x(medianPrice)} x2={x(medianPrice)} y1={PAD_T} y2={H - PAD_B} stroke="#31404e" strokeDasharray="3 4" />
          <line x1={PAD_L} x2={W - PAD_R} y1={y(medianScore)} y2={y(medianScore)} stroke="#31404e" strokeDasharray="3 4" />

          <text x={PAD_L + 10} y={PAD_T + 16} fill="#5f6b7d" fontSize="10">高分低价</text>
          <text x={W - PAD_R - 10} y={PAD_T + 16} fill="#5f6b7d" fontSize="10" textAnchor="end">高分高价</text>
          <text x={PAD_L + 10} y={H - PAD_B - 18} fill="#5f6b7d" fontSize="10">低分低价</text>
          <text x={W - PAD_R - 10} y={H - PAD_B - 18} fill="#5f6b7d" fontSize="10" textAnchor="end">低分高价</text>
          <text x={(PAD_L + W - PAD_R) / 2} y={H - 12} fill="#5f6b7d" fontSize="10" textAnchor="middle" letterSpacing="1">最低输入价（对数）→</text>
          <text x={14} y={PAD_T + 60} fill="#5f6b7d" fontSize="10" transform={`rotate(-90 14 ${PAD_T + 60})`} textAnchor="middle">综合分 ↑</text>

          <text x={PAD_L} y={H - 12} fill="#7a8599" fontSize="10" fontFamily="var(--mono)">{fmtCny(Math.min(...pts.map((s) => s.fromPriceCny)))}</text>
          <text x={W - PAD_R} y={H - 12} fill="#7a8599" fontSize="10" textAnchor="end" fontFamily="var(--mono)">{fmtCny(Math.max(...pts.map((s) => s.fromPriceCny)))}/M</text>
          <text x={PAD_L - 8} y={H - PAD_B} fill="#7a8599" fontSize="10" textAnchor="end" dominantBaseline="middle" fontFamily="var(--mono)">{chart.smin.toFixed(0)}</text>
          <text x={PAD_L - 8} y={PAD_T + 4} fill="#7a8599" fontSize="10" textAnchor="end" dominantBaseline="middle" fontFamily="var(--mono)">100</text>

          {pts.map((s, i) => (
            <circle
              key={s.slug}
              cx={x(s.fromPriceCny)}
              cy={y(s.rankScore)}
              r={hover === i ? 6 : 3.6}
              fill={TIER_COLOR[s.purityTier] ?? TIER_COLOR.unknown}
              fillOpacity={hover === null || hover === i ? 0.9 : 0.55}
              stroke={hover === i ? '#e9edf4' : 'none'}
              strokeWidth="1"
              style={{ cursor: 'pointer' }}
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(null)}
              onClick={() => navigate(`/stations/${s.slug}`)}
            >
              <title>{`${s.name} · 综合分 ${s.rankScore.toFixed(1)} · ${fmtCny(s.fromPriceCny)}/M`}</title>
            </circle>
          ))}
        </svg>
        {active && (
          <div
            className="quadrant-tip"
            style={{ left: `${(activeX / W) * 100}%`, top: `${(activeY / H) * 100}%` }}
          >
            <b>{active.name}</b>
            <span className="mono">综合分 {active.rankScore.toFixed(1)} · {fmtCny(active.fromPriceCny)}/M</span>
            <span className="dim tiny">{TIER_LABEL[active.purityTier] ?? active.purityTier}</span>
          </div>
        )}
      </div>
    </div>
  );
}

import { Link } from 'react-router-dom';

export function EmptyState({
  title,
  desc,
  action,
  actionTo = '/',
}: {
  title: string;
  desc: string;
  action?: string;
  actionTo?: string;
}) {
  return (
    <div className="empty-state">
      <svg width="132" height="132" viewBox="0 0 132 132" fill="none" aria-hidden="true">
        <defs>
          <radialGradient id="es-glow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#2ee6a6" stopOpacity=".18" />
            <stop offset="100%" stopColor="#2ee6a6" stopOpacity="0" />
          </radialGradient>
        </defs>
        <circle cx="66" cy="66" r="58" fill="url(#es-glow)" />
        <circle cx="66" cy="66" r="46" stroke="#25303c" strokeDasharray="3 5" />
        <circle cx="66" cy="66" r="30" stroke="#1d2833" />
        <circle cx="66" cy="66" r="3" fill="#2ee6a6" />
        <path d="M66 8v10M66 114v10M8 66h10M114 66h10" stroke="#2b3a48" strokeLinecap="round" />
        <path d="M29 29l7 7M103 103l-7-7M103 29l-7 7M29 103l7-7" stroke="#22303b" strokeLinecap="round" />
      </svg>
      <b>{title}</b>
      <p>{desc}</p>
      {action && <Link className="btn btn-primary btn-sm" to={actionTo}>{action}</Link>}
    </div>
  );
}

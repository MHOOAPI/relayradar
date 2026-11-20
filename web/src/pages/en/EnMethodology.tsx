import { Link } from 'react-router-dom';

export function EnMethodologyPage() {
  return (
    <main className="wrap method">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />METHODOLOGY V2</span>
          <h1>Scoring methodology v2</h1>
          <p className="sub">Every score is reproducible from the probe data; no editorial weight is applied.</p>
        </div>
        <Link className="btn btn-ghost btn-sm" to="/en">Back to rankings</Link>
      </div>

      <section className="panel pad">
        <h2>Per group</h2>
        <pre className="formula-block mono">{`A = 100 × Wilson95(k, n)              # availability, 95% lower bound
S = 100 × clamp((5000 − p50) / 4200)  # 800ms → 100, 5000ms → 0
C = 100 × clamp(1 − jitter / 2)       # jitter = (p95 − p50) / max(p50, 150)
Q = 0.62·A + 0.26·S + 0.12·C`}</pre>
        <p className="fine">
          A group needs at least 24 probes in the 7-day window to be scored. The station axis uses 80% coverage-weighted
          mean plus 20% worst group, so one bad group cannot be hidden by many good ones.
        </p>
      </section>

      <section className="panel pad">
        <h2>Uncertainty</h2>
        <p className="fine">
          The ± value is the half-width of the Wilson 95% interval for availability, multiplied by 0.62 (its weight). It
          only covers availability sampling error; speed and consistency are point estimates. When the gap between #1
          and #2 is smaller than the combined margin, we say they are within error.
        </p>
        <h2>Purity and price</h2>
        <p className="fine">
          Purity checks only cover Claude groups (thinking signature and behaviour fingerprint) and are shown as a
          label, not part of the score. Price is official USD × settlement rate × group multiplier, also separate.
        </p>
        <h2>Limitations</h2>
        <p className="fine">
          Probes are short and cheap: they measure reachability and latency, not long-output throughput. A single window
          cannot prove a provider never degrades. Signature checks prove presence, not absolute authenticity.
        </p>
      </section>
    </main>
  );
}

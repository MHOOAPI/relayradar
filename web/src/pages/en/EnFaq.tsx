import { Link } from 'react-router-dom';

const ITEMS = [
  ['What is an LLM API relay?', 'A relay aggregates several official APIs behind one domain and one key, billing through its own exchange rate and group multiplier. It removes multi-account and payment friction, but routing and billing are controlled by the provider, so independent monitoring matters.'],
  ['How do you detect a watered-down relay?', 'Protocol and billing signals (does the response echo the requested model, are usage tokens reported and stable across identical requests, is prompt caching passed through) plus sampling (is the Claude thinking signature present and consistent). The audit covers the first; purity checks cover the second.'],
  ['Why is price not part of the score?', 'Cheap can mean worse upstreams, lower availability or no refunds. Folding price into the score would systematically punish expensive-but-stable providers. Price is a separate column.'],
  ['What does ± mean?', 'Availability sampling error: Wilson 95% half-width × 0.62. Speed and consistency are point estimates; we do not invent an overall interval.'],
  ['How often is data updated?', 'Probes run continuously and scoring uses a 7-day window. A group needs 24 probes to be scored; a station needs at least half of its groups scored to be ranked.'],
  ['Can rankings be bought?', 'No. Sponsorships are labelled and never change scores, ordering or audit results.'],
  ['Is the data real?', 'This repository ships a demo dataset: provider names and domains are public, but multipliers, probes, scores and reviews are generated. It is not an assessment of the real providers.'],
  ['What does the audit not do?', 'It does not judge whether a model is "dumbed down"; it only checks observable protocol and billing signals. Behavioural fingerprinting and signature cross-verification remain on the roadmap.'],
];

export function EnFaqPage() {
  return (
    <main className="wrap method">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />FAQ</span>
          <h1>Frequently asked questions</h1>
          <p className="sub">Relays, scoring, audits and data coverage. <Link to="/en/methodology">Methodology</Link> · <Link to="/en">Rankings</Link> · <Link to="/faq">中文 FAQ</Link></p>
        </div>
      </div>
      <section className="panel pad faq-panel">
        {ITEMS.map(([q, a], i) => (
          <details key={q} className="faq-item" open={i === 0}>
            <summary>{q}</summary>
            <p>{a}</p>
          </details>
        ))}
      </section>
    </main>
  );
}

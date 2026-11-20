import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../../api';
import type { GuideEN } from '../../types/guide';
import { Spinner } from '../../components/ui';

export function EnGuidesPage() {
  const [items, setItems] = useState<GuideEN[] | null>(null);

  useEffect(() => {
    document.title = 'Guides: choosing, billing and verifying relays | RelayRadar';
    api.guidesEn().then((r) => setItems(r.items)).catch(() => setItems([]));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <span className="eyebrow"><i />GUIDES</span>
          <h1>Guides</h1>
          <p className="sub">Long-form notes on choosing, billing and verifying LLM API relays. <Link to="/en">Rankings</Link> · <Link to="/en/methodology">Methodology</Link></p>
        </div>
      </div>
      {!items && <div className="loading"><Spinner /> Loading guides…</div>}
      <div className="guide-grid">
        {items?.map((g, i) => (
          <article key={g.slug} className="panel guide-card">
            <span className="mono dim tiny">0{i + 1} · {g.minutes} min read</span>
            <h2><Link to={`/en/guides/${g.slug}`}>{g.title}</Link></h2>
            <p>{g.desc}</p>
            <div className="guide-card-foot">
              <span className="mono dim tiny">Updated {g.updated}</span>
              <Link className="link-btn" to={`/en/guides/${g.slug}`}>Read →</Link>
            </div>
          </article>
        ))}
      </div>
    </main>
  );
}

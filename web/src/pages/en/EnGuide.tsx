import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../../api';
import type { GuideEN } from '../../types/guide';
import { Spinner } from '../../components/ui';

export function EnGuidePage() {
  const { slug = '' } = useParams();
  const [guide, setGuide] = useState<GuideEN | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.guideEn(slug)
      .then((g) => {
        setGuide(g);
        document.title = `${g.title} | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [slug]);

  if (error) return <main className="wrap"><div className="error-box">{error} <Link className="link-btn" to="/en/guides">Back to guides</Link></div></main>;
  if (!guide) return <main className="wrap"><div className="loading-page"><Spinner /> Loading guide…</div></main>;

  return (
    <main className="wrap">
      <div className="guide-page">
        <div className="crumbs mono">
          <Link to="/en/guides">Guides</Link><span>/</span><span className="dim">{guide.slug}</span>
        </div>
        <article className="guide-article">
          <header className="guide-head">
            <h1>{guide.title}</h1>
            <p className="guide-meta mono">Updated {guide.updated} · {guide.minutes} min read</p>
            <p className="guide-lead">{guide.desc}</p>
          </header>
          <div className="guide-body" dangerouslySetInnerHTML={{ __html: guide.bodyHtml ?? '' }} />
          <footer className="guide-foot">
            <span>Related</span>
            <Link to="/en">Rankings</Link>
            <Link to="/en/methodology">Methodology</Link>
            <Link to="/guides">中文指南</Link>
          </footer>
        </article>
      </div>
    </main>
  );
}

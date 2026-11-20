import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../api';
import type { GuideDetail } from '../types/guide';
import { Spinner } from '../components/ui';

export function GuidePage() {
  const { slug } = useParams();
  const [guide, setGuide] = useState<GuideDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!slug) return;
    setGuide(null);
    api.guide(slug)
      .then((g) => {
        setGuide(g);
        document.title = `${g.title} | RelayRadar`;
      })
      .catch((e: Error) => setError(e.message));
  }, [slug]);

  if (error) {
    return (
      <main className="wrap">
        <div className="error-box">
          加载失败：{error} <Link className="btn btn-sm btn-ghost" to="/guides">返回指南</Link>
        </div>
      </main>
    );
  }

  if (!guide) {
    return <main className="wrap"><div className="loading-page"><Spinner /> 读取文章…</div></main>;
  }

  return (
    <main className="wrap">
      <div className="guide-page">
        <div className="crumbs mono">
          <Link to="/guides">指南</Link><span>/</span><span className="dim">{guide.slug}</span>
        </div>

        <article className="guide-article">
          <header className="guide-head">
            <h1>{guide.title}</h1>
            <p className="guide-meta mono">更新 {guide.updated} · 约 {guide.minutes} 分钟阅读</p>
            <p className="guide-lead">{guide.desc}</p>
          </header>

          <div className="guide-body" dangerouslySetInnerHTML={{ __html: guide.bodyHtml }} />

          <footer className="guide-foot">
            <span>觉得有用？</span>
            <Link to="/">看实时榜单</Link>
            <Link to="/audit">检测你的接口</Link>
            <Link to="/guides">更多指南</Link>
          </footer>
        </article>
      </div>
    </main>
  );
}

import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { GuideSummary } from '../types/guide';
import { Spinner } from '../components/ui';

export function GuidesPage() {
  const [items, setItems] = useState<GuideSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    document.title = '指南：中转站选择、计费与验真 | RelayRadar';
    api.guides()
      .then((r) => setItems(r.items))
      .catch((e: Error) => setError(e.message));
  }, []);

  return (
    <main className="wrap">
      <div className="page-head">
        <div>
          <h1>指南</h1>
          <p className="sub">关于中转站选择、计费与验真的长文，全部可被搜索引擎与 AI 引擎直接读取。</p>
        </div>
      </div>

      {error && <div className="error-box">{error}</div>}
      {!items && !error && <div className="loading"><Spinner /> 加载中…</div>}

      <div className="guide-grid">
        {items?.map((g, i) => (
          <article key={g.slug} className="panel guide-card">
            <span className="mono dim tiny">0{i + 1} · {g.minutes} 分钟</span>
            <h2><Link to={`/guides/${g.slug}`}>{g.title}</Link></h2>
            <p>{g.desc}</p>
            <div className="guide-card-foot">
              <span className="mono dim tiny">更新 {g.updated}</span>
              <Link className="link-btn" to={`/guides/${g.slug}`}>阅读 →</Link>
            </div>
          </article>
        ))}
      </div>
    </main>
  );
}

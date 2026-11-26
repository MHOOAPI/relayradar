import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

export function NotFoundPage() {
  const [term, setTerm] = useState('');
  const navigate = useNavigate();

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const q = term.trim();
    if (!q) return;
    navigate(`/stations?q=${encodeURIComponent(q)}`);
  };

  return (
    <main className="wrap">
      <div className="notfound">
        <img className="nf-art" src="/img/notfound.webp" alt="" width={180} height={180} />
        <span className="mono dim">404</span>
        <h1>页面不存在</h1>
        <p className="sub">链接可能已失效，或者这个地址还没有内容。直接搜站点，或回到榜单。</p>
        <form className="nf-search" onSubmit={submit} role="search">
          <input
            value={term}
            onChange={(e) => setTerm(e.target.value)}
            placeholder="搜索站点名或域名，例如 mhoo、wolf"
            aria-label="搜索站点"
          />
          <button className="btn btn-primary btn-sm">搜索</button>
        </form>
        <p className="nf-links">
          <Link to="/">回到榜单</Link>
          <span>·</span>
          <Link to="/stations">站点目录</Link>
          <span>·</span>
          <Link to="/models">模型价格</Link>
          <span>·</span>
          <Link to="/faq">常见问题</Link>
        </p>
      </div>
    </main>
  );
}

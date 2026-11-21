import { Suspense, useEffect, useState } from 'react';
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom';
import { api } from '../api';
import type { Meta } from '../types';
import { fmtDateTime, Logo, Spinner } from './ui';
import { CommandPalette } from './CommandPalette';
import { useAuth } from '../lib/auth';

const REFS = [
  { name: 'PriceAI', url: 'https://priceai.cc', desc: '卡网订阅与中转 API 比价' },
  { name: 'OkkMax', url: 'https://www.okkmax.com', desc: '可用性 / 速度 / 纯度榜单' },
  { name: 'VeriDrop', url: 'https://veridrop.app', desc: '中转站真伪检测（开源）' },
  { name: '禾维 AI', url: 'https://www.hvoyai.com', desc: 'Claude 中转站检测' },
  { name: 'LMSpeed', url: 'https://lmspeed.net', desc: '模型跑分与 API 测速' },
];

const CRUMB_LABELS: Record<string, string> = {
  '/models': '模型',
  '/leaderboards': '排行榜',
  '/compare': '横向对比',
  '/changes': '变更雷达',
  '/free': '免费额度',
  '/methodology': '评分方法',
  '/faq': '常见问题',
  '/api-docs': '公开 API',
  '/audit': '接口检测',
  '/submit': '提交站点',
  '/stations': '站点目录',
  '/status': '服务状态',
  '/model-status': '模型 API 状态',
  '/alerts': '告警订阅',
};

export function Layout() {
  const [meta, setMeta] = useState<Meta | null>(null);
  const { pathname } = useLocation();
  const { user, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  useEffect(() => {
    api.meta().then(setMeta).catch(() => setMeta(null));
  }, []);

  return (
    <div className="app">
      <div className="atmosphere" aria-hidden="true">
        <span className="atm-grain" />
      </div>
      <div className="topbar">
        <div className="wrap topbar-inner">
          <div className="topbar-left">
            <span className="live"><i />LIVE</span>
            <span className="mono">
              {meta ? fmtDateTime(meta.dataUpdatedAt) : '—'}
            </span>
            <span className="dot-sep" />
            <span className="mono dim">
              {meta
                ? `${meta.stations} 站点 · ${meta.groups} 分组 · ${meta.probes.toLocaleString('zh-CN')} 次探针`
                : '连接中…'}
            </span>
          </div>
          <div className="topbar-right">
            {meta?.demo && <span className="demo-tag">DEMO 数据集</span>}
            <span className="mono dim">方法 v{meta?.methodologyVersion ?? '1.0'}</span>
          </div>
        </div>
      </div>

      <header className="header">
        <div className="wrap header-inner">
          <Link to="/" className="brand">
            <Logo />
            <span className="brand-text">
              <b>RelayRadar</b>
              <small>中转雷达</small>
            </span>
          </Link>

          <nav className="nav">
            <NavLink to="/" end>榜单</NavLink>
            <NavLink to="/stations">站点</NavLink>
            <NavLink to="/models">模型</NavLink>
            <NavLink to="/leaderboards">排行</NavLink>
            <NavLink to="/audit">检测</NavLink>
            <NavLink to="/free-api-keys">免费 API</NavLink>
            <div className="ref">
              <button type="button" className="ref-btn">更多 <span>▾</span></button>
              <div className="ref-menu">
                <Link to="/changes"><b>变更雷达</b><small>价格、可用率与纯度异常</small></Link>
                <Link to="/compare"><b>横向对比</b><small>按可用性、价格与纯度并排</small></Link>
                <Link to="/free"><b>免费额度</b><small>官方免费层与赠送额度</small></Link>
                <Link to="/submit"><b>提交与认领</b><small>站点提交、DNS 认领</small></Link>
                <Link to="/guides"><b>指南</b><small>选择、计费与验真</small></Link>
                <Link to="/faq"><b>常见问题</b><small>中转站、评分与数据口径</small></Link>
                <Link to="/methodology"><b>评分方法 v2</b><small>公式、权重与局限</small></Link>
                <Link to="/api-docs"><b>公开 API</b><small>JSON 接口与 llms.txt</small></Link>
                <Link to="/status"><b>服务状态</b><small>本站 API / 探针 / 投递</small></Link>
                <Link to="/model-status"><b>模型 API 状态</b><small>OpenAI / Anthropic / Gemini 等</small></Link>
                    <div className="ref-sep" />
                    <span className="ref-label mono">参考站</span>
                    {REFS.map((r) => (
                      <a key={r.name} href={r.url} target="_blank" rel="noreferrer noopener">
                        <b>{r.name}</b>
                        <small>{r.desc}</small>
                      </a>
                    ))}
              </div>
            </div>
          </nav>

          <div className="header-actions">
            <CommandPalette />
            {user ? (
              <div className="header-user">
                {user.role === 'admin' && <Link className="link-btn" to="/admin">后台</Link>}
                <Link className="mono dim header-account" to="/account" title="个人中心">{user.name || '个人中心'}</Link>
                <button type="button" className="link-btn" onClick={() => void logout()}>退出</button>
              </div>
            ) : (
              <div className="header-user">
                <Link className="link-btn" to="/login">登录</Link>
                <Link className="link-btn" to="/register">注册</Link>
              </div>
            )}
            <Link className="btn btn-ghost btn-sm" to="/submit">提交站点</Link>
          </div>

          <button
            type="button"
            className="menu-btn"
            aria-label="打开菜单"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((v) => !v)}
          >
            <span /><span /><span />
          </button>
        </div>
      </header>

      {menuOpen && (
        <div className="mobile-menu" onClick={() => setMenuOpen(false)}>
          <div className="mobile-menu-panel" onClick={(e) => e.stopPropagation()}>
            <div className="mm-head">
              <b>RelayRadar</b>
              <button type="button" className="mm-close" aria-label="关闭菜单" onClick={() => setMenuOpen(false)}>✕</button>
            </div>
            <nav className="mm-links">
              <Link to="/">榜单</Link>
              <Link to="/stations">站点目录</Link>
              <Link to="/models">模型价格</Link>
              <Link to="/leaderboards">排行榜</Link>
              <Link to="/changes">变更雷达</Link>
              <Link to="/audit">接口检测</Link>
              <Link to="/free-api-keys">免费 API Key</Link>
              <Link to="/compare">横向对比</Link>
              <Link to="/free">免费额度</Link>
              <Link to="/guides">指南</Link>
              <Link to="/submit">提交站点</Link>
              <Link to="/faq">常见问题</Link>
              <Link to="/methodology">评分方法</Link>
              <Link to="/api-docs">公开 API</Link>
              <Link to="/status">服务状态</Link>
              <Link to="/model-status">模型 API 状态</Link>
              {user ? <Link to="/account">个人中心</Link> : <Link to="/login">登录 / 注册</Link>}
            </nav>
          </div>
        </div>
      )}

      {meta?.demo && (
        <div className="demo-bar">
          <div className="wrap">
            <b>演示数据</b>
            站点域名用于展示真实图标；评分、探针与评价均为生成值，不代表真实商家表现。
          </div>
        </div>
      )}

      {(() => {
        const label = CRUMB_LABELS[pathname];
        if (!label) return null;
        return (
          <div className="wrap">
            <div className="crumbs mono">
              <Link to="/">首页</Link><span>/</span><span className="dim">{label}</span>
            </div>
          </div>
        );
      })()}

      <Suspense fallback={<div className="wrap loading-page"><Spinner /> 正在加载…</div>}>
        <div className="route-fade" key={pathname}>
          <Outlet />
        </div>
      </Suspense>

      <footer className="footer">
        <div className="wrap footer-grid">
          <div>
            <div className="brand footer-brand">
              <Logo size={22} />
              <span className="brand-text"><b>RelayRadar</b><small>中转雷达</small></span>
            </div>
            <p className="fine">
              独立的第三方监测项目。不做交易闭环、不收款、不担保任何站点。
              分数只来自探针与抽检数据，赞助与广告不参与排序。
            </p>
          </div>
          <div>
            <h4>产品</h4>
            <Link to="/">中转站榜单</Link>
            <Link to="/models">模型价格</Link>
            <Link to="/leaderboards">排行榜</Link>
            <Link to="/compare">横向对比</Link>
            <Link to="/changes">变更雷达</Link>
            <Link to="/guides">指南</Link>
          </div>
          <div>
            <h4>参与</h4>
            <Link to="/free">免费额度</Link>
            <Link to="/submit">提交站点</Link>
            <Link to="/submit">站长认领</Link>
            <Link to="/faq">常见问题</Link>
            <Link to="/api-docs">公开 API</Link>
            <a href="/sitemap.xml" target="_blank" rel="noreferrer">sitemap</a>
          </div>
          <div>
            <h4>参考站</h4>
            {REFS.map((r) => (
              <a key={r.name} href={r.url} target="_blank" rel="noreferrer noopener">{r.name}</a>
            ))}
          </div>
          <div>
            <h4>声明</h4>
            <Link to="/methodology">评分方法</Link>
            <span>数据为演示数据集</span>
            <span>不构成购买建议</span>
          </div>
        </div>
        <div className="wrap footer-status mono">
          <span><i className="status-dot" /> <Link to="/status">服务状态</Link></span>
          <span>数据更新 {meta ? fmtDateTime(meta.dataUpdatedAt) : '—'}</span>
          <span>独立监测 · 不构成购买建议</span>
        </div>
        <div className="wrap footer-bottom mono">
          <span>© 2026 RelayRadar</span>
          <span>Go · React/TypeScript · PostgreSQL · 自定义 CSS</span>
        </div>
      </footer>
    </div>
  );
}

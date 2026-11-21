import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { GuideSummary } from '../types/guide';
import type { SearchModel, SearchStation, Station, ModelSummary } from '../types';

type Item = { key: string; label: string; sub: string; to: string; group: string };

export function CommandPalette() {
  const [open, setOpen] = useState(() => new URLSearchParams(window.location.search).has('cmd'));
  const [q, setQ] = useState('');
  const [active, setActive] = useState(0);
  const [results, setResults] = useState<{ stations: SearchStation[]; models: SearchModel[] }>({ stations: [], models: [] });
  const [tops, setTops] = useState<{ stations: Station[]; models: ModelSummary[] }>({ stations: [], models: [] });
  const [guides, setGuides] = useState<GuideSummary[]>([]);
  const [loaded, setLoaded] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((v) => !v);
      } else if (e.key === 'Escape') {
        setOpen(false);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  // Only the empty-state suggestions and the (small) guide list are prefetched;
  // everything typed goes through the server so the palette can reach all
  // stations and models, not just the first page.
  useEffect(() => {
    if (!open) return;
    setActive(0);
    window.setTimeout(() => inputRef.current?.focus(), 30);
    if (loaded) return;
    Promise.all([api.stations(6), api.models(), api.guides()])
      .then(([s, m, g]) => {
        setTops({ stations: s.items, models: m.items.slice(0, 3) });
        setGuides(g.items);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [open, loaded]);

  useEffect(() => {
    const query = q.trim();
    if (!query) {
      setResults({ stations: [], models: [] });
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      api.search(query, 8, controller.signal)
        .then((r) => setResults({ stations: r.stations, models: r.models }))
        .catch(() => {});
    }, 160);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [q]);

  const items = useMemo<Item[]>(() => {
    const query = q.trim().toLowerCase();
    const guideItems: Item[] = guides.map((g) => ({
      key: `g-${g.slug}`,
      label: g.title,
      sub: '指南',
      to: `/guides/${g.slug}`,
      group: '指南',
    }));

    if (!query) {
      return [
        ...tops.stations.map((s) => ({
          key: `s-${s.slug}`,
          label: s.name,
          sub: `${s.domain} · 综合分 ${s.rankScore.toFixed(1)}`,
          to: `/stations/${s.slug}`,
          group: '站点',
        })),
        ...tops.models.map((m) => ({
          key: `m-${m.id}`,
          label: m.name,
          sub: `${m.vendorName} · ¥${m.minInputCny.toFixed(2)}/1M`,
          to: `/models/${m.id}`,
          group: '模型',
        })),
        ...guideItems.slice(0, 3),
      ];
    }

    return [
      ...results.stations.map((s) => ({
        key: `s-${s.slug}`,
        label: s.name,
        sub: `${s.domain} · 综合分 ${s.rankScore.toFixed(1)}`,
        to: `/stations/${s.slug}`,
        group: '站点',
      })),
      ...results.models.map((m) => ({
        key: `m-${m.id}`,
        label: m.name,
        sub: `${m.vendorName} · ¥${m.minInputCny.toFixed(2)}/1M`,
        to: `/models/${m.id}`,
        group: '模型',
      })),
      ...guideItems.filter((g) => g.label.toLowerCase().includes(query)).slice(0, 4),
    ];
  }, [q, results, tops, guides]);

  const go = (it: Item) => {
    setOpen(false);
    setQ('');
    navigate(it.to);
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActive((i) => Math.min(items.length - 1, i + 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActive((i) => Math.max(0, i - 1));
    } else if (e.key === 'Enter' && items[active]) {
      e.preventDefault();
      go(items[active]);
    }
  };

  return (
    <>
      <button type="button" className="cmd-trigger" onClick={() => setOpen(true)} aria-label="搜索">
        <span>搜索</span>
        <kbd>Ctrl K</kbd>
      </button>

      {open && (
        <div className="cmd-overlay" onMouseDown={(e) => { if (e.target === e.currentTarget) setOpen(false); }}>
          <div className="cmd-panel" role="dialog" aria-modal="true" aria-label="全站搜索">
            <input
              ref={inputRef}
              className="cmd-input"
              value={q}
              onChange={(e) => { setQ(e.target.value); setActive(0); }}
              onKeyDown={onKeyDown}
              placeholder="搜索站点、模型、指南…"
            />
            <div className="cmd-list">
              {items.length === 0 && <div className="cmd-empty">没有匹配结果</div>}
              {items.map((it, i) => (
                <button
                  type="button"
                  key={it.key}
                  className={`cmd-item ${i === active ? 'on' : ''}`}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => go(it)}
                >
                  <span>
                    <b>{it.label}</b>
                    <small>{it.sub}</small>
                  </span>
                  <span className="mono dim tiny">{it.group}</span>
                </button>
              ))}
            </div>
            <div className="cmd-foot mono">
              <span>↑↓ 选择</span><span>↵ 打开</span><span>Esc 关闭</span>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

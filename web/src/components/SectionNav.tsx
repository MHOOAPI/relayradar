import { useEffect, useRef, useState } from 'react';

export interface SectionItem {
  id: string;
  label: string;
}

/** Sticky in-page navigation that tracks the section currently in view. */
export function SectionNav({ items }: { items: SectionItem[] }) {
  const [active, setActive] = useState(items[0]?.id ?? '');
  const navRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const sections = items
      .map((i) => document.getElementById(i.id))
      .filter((el): el is HTMLElement => el !== null);
    if (!sections.length) return;
    const io = new IntersectionObserver(
      (entries) => {
        const first = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
        if (first) setActive(first.target.id);
      },
      { rootMargin: '-130px 0px -60% 0px' },
    );
    sections.forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, [items]);

  useEffect(() => {
    const hash = window.location.hash.slice(1);
    if (hash && items.some((i) => i.id === hash)) setActive(hash);
  }, [items]);

  const jump = (id: string) => {
    const el = document.getElementById(id);
    if (!el) return;
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - 116, behavior: reduce ? 'auto' : 'smooth' });
    window.history.replaceState(null, '', `#${id}`);
    setActive(id);
  };

  return (
    <nav className="section-nav" aria-label="本页章节" ref={navRef}>
      {items.map((i) => (
        <button
          key={i.id}
          type="button"
          className={`section-nav-item ${active === i.id ? 'on' : ''}`}
          aria-current={active === i.id ? 'true' : undefined}
          onClick={() => jump(i.id)}
        >
          {i.label}
        </button>
      ))}
    </nav>
  );
}

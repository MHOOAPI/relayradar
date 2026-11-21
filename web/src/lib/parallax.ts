import { useEffect } from 'react';
import type { RefObject } from 'react';

// Scroll-linked parallax with one shared rAF loop per element. Disabled when
// the visitor asks for reduced motion, in which case no listener is attached.
export function useScrollParallax(ref: RefObject<HTMLElement | null>, factor: number, max = 800) {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    let raf = 0;
    const onScroll = () => {
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        const el = ref.current;
        if (!el) return;
        const y = Math.min(window.scrollY, max);
        el.style.transform = `translate3d(0, ${(y * factor).toFixed(1)}px, 0)`;
      });
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      window.removeEventListener('scroll', onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  }, [ref, factor, max]);
}


// Soft pointer-follow light on a large surface (table/panel). Document-level
// listener filters by the current ref so late mounts still work; --glow fades
// the light as the panel moves away from the viewport centre.
export function usePointerGlow(ref: RefObject<HTMLElement | null>, enabled = true) {
  useEffect(() => {
    if (!enabled) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if (!window.matchMedia('(pointer: fine)').matches) return;
    let raf = 0;
    const setGlow = () => {
      const el = ref.current;
      if (!el) return;
      const r = el.getBoundingClientRect();
      const vh = window.innerHeight;
      const visible = Math.max(0, Math.min(r.bottom, vh) - Math.max(r.top, 0));
      const fade = Math.max(0.1, Math.min(1, visible / Math.min(r.height, vh)));
      el.style.setProperty('--glow', fade.toFixed(2));
    };
    const onMove = (e: MouseEvent) => {
      const el = ref.current;
      if (!el) return;
      const target = e.target as Node | null;
      if (!target || !el.contains(target)) {
        el.style.setProperty('--active', '0');
        return;
      }
      el.style.setProperty('--active', '1');
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        const node = ref.current;
        if (!node) return;
        const r = node.getBoundingClientRect();
        node.style.setProperty('--mx', `${(((e.clientX - r.left) / r.width) * 100).toFixed(2)}%`);
        node.style.setProperty('--my', `${(((e.clientY - r.top) / r.height) * 100).toFixed(2)}%`);
      });
    };
    const onScroll = () => {
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        setGlow();
      });
    };
    setGlow();
    document.addEventListener('mousemove', onMove, { passive: true });
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      document.removeEventListener('mousemove', onMove);
      window.removeEventListener('scroll', onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  }, [ref, enabled]);
}
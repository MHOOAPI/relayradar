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


// Soft pointer-follow light on a large surface (table/panel). The listener lives
// on the document and filters by the current ref, so it survives late mounts.
export function usePointerGlow(ref: RefObject<HTMLElement | null>, enabled = true) {
  useEffect(() => {
    if (!enabled) return;
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    if (!window.matchMedia('(pointer: fine)').matches) return;
    let raf = 0;
    const onMove = (e: MouseEvent) => {
      const el = ref.current;
      if (!el) return;
      const target = e.target as Node | null;
      if (!target || !el.contains(target)) return;
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
    document.addEventListener('mousemove', onMove, { passive: true });
    return () => {
      document.removeEventListener('mousemove', onMove);
      if (raf) cancelAnimationFrame(raf);
    };
  }, [ref, enabled]);
}
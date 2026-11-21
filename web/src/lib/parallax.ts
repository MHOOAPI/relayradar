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

import { useEffect, useState } from 'react';

const KEY = 'rr.watch.v1';
const listeners = new Set<() => void>();

function read(): string[] {
  try {
    const raw = localStorage.getItem(KEY);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((x) => typeof x === 'string') : [];
  } catch {
    return [];
  }
}

function write(ids: string[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(ids));
  } catch {
    /* storage disabled */
  }
  listeners.forEach((fn) => fn());
}

/** Watchlist persisted in localStorage, shared across every component. */
export function useWatchlist() {
  const [ids, setIds] = useState<string[]>(read);

  useEffect(() => {
    const sync = () => setIds(read());
    listeners.add(sync);
    window.addEventListener('storage', sync);
    return () => {
      listeners.delete(sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  return {
    ids,
    has: (slug: string) => ids.includes(slug),
    toggle: (slug: string) => {
      const current = read();
      write(current.includes(slug) ? current.filter((x) => x !== slug) : [...current, slug]);
    },
  };
}

import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react';

type ToastKind = 'ok' | 'err' | 'info';
type Item = { id: number; msg: string; kind: ToastKind };

const ToastCtx = createContext<{ push: (msg: string, kind?: ToastKind) => void }>({ push: () => {} });

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Item[]>([]);
  const idRef = useRef(1);

  const push = useCallback((msg: string, kind: ToastKind = 'ok') => {
    const id = idRef.current++;
    setItems((prev) => [...prev.slice(-2), { id, msg, kind }]);
    window.setTimeout(() => setItems((prev) => prev.filter((t) => t.id !== id)), 2800);
  }, []);

  const value = useMemo(() => ({ push }), [push]);

  return (
    <ToastCtx.Provider value={value}>
      {children}
      <div className="toasts" role="status" aria-live="polite">
        {items.map((t) => (
          <div key={t.id} className={`toast toast-${t.kind}`}>{t.msg}</div>
        ))}
      </div>
    </ToastCtx.Provider>
  );
}

export function useToast() {
  return useContext(ToastCtx);
}

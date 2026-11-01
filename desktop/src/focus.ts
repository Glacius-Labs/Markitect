import { useEffect } from 'react';

// atMostEvery runs fn when called, but not again until ms have passed.
export function atMostEvery(ms: number, fn: () => void, now: () => number = Date.now): () => void {
  let last = -Infinity;
  return () => {
    const t = now();
    if (t - last >= ms) {
      last = t;
      fn();
    }
  };
}

// useRefreshOnFocus reloads when the window comes back, so edits made in an
// editor or a terminal meanwhile show up. It reloads at most every few seconds:
// a reload must not be able to feed itself, as it did when each reload started
// a program that opened a window, which took the focus and gave it back.
export function useRefreshOnFocus(load: () => void) {
  useEffect(() => {
    const onFocus = atMostEvery(5000, load);
    window.addEventListener('focus', onFocus);
    return () => window.removeEventListener('focus', onFocus);
  }, [load]);
}

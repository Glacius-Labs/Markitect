import { useEffect, useRef } from 'react';

// atMostEvery runs fn when called, or, when it ran less than ms ago or was only
// just made, as soon as that time is up. Calls in between count as one, so the
// last call is never lost.
export function atMostEvery(ms: number, fn: () => void) {
  let last = Date.now();
  let waiting: ReturnType<typeof setTimeout> | undefined;
  return {
    call() {
      waiting ??= setTimeout(
        () => {
          waiting = undefined;
          last = Date.now();
          fn();
        },
        Math.max(0, last + ms - Date.now()),
      );
    },
    stop() {
      clearTimeout(waiting);
    },
  };
}

// useRefreshOnFocus reloads when the window comes back, so edits made in an
// editor or a terminal meanwhile show up. It reloads at most every few seconds,
// over the whole life of the page, whatever load is on each render: a reload
// must not be able to feed itself, as it did when each reload started a program
// that opened a window, which took the focus and gave it back.
export function useRefreshOnFocus(load: () => void) {
  const current = useRef(load);
  current.current = load;
  useEffect(() => {
    const refresh = atMostEvery(5000, () => current.current());
    window.addEventListener('focus', refresh.call);
    return () => {
      window.removeEventListener('focus', refresh.call);
      refresh.stop();
    };
  }, []);
}

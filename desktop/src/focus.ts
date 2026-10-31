import { useEffect } from 'react';

// useRefreshOnFocus reloads when the window comes back, so edits made in an
// editor or a terminal meanwhile show up.
export function useRefreshOnFocus(load: () => void) {
  useEffect(() => {
    window.addEventListener('focus', load);
    return () => window.removeEventListener('focus', load);
  }, [load]);
}

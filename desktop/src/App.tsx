import { useCallback, useEffect, useState } from 'react';
import { api, ApiError, type AppInfo } from './api';
import { Hub, useEscapeTo, type InitRun } from './Hub';
import { Projects } from './Projects';

// The address after # is the place in the app, so back and forward work.
function place(): { project?: string } {
  const m = /^#\/p\/([0-9a-f]+)$/.exec(window.location.hash);
  return m ? { project: m[1] } : {};
}

export function App() {
  const [info, setInfo] = useState<AppInfo | null>(null);
  const [closed, setClosed] = useState(false);
  const [where, setWhere] = useState(place);
  // The init that created a project, shown on its hub until the owner leaves.
  const [created, setCreated] = useState<{ id: string; init: InitRun } | null>(null);

  const loadInfo = useCallback(() => {
    api.app().then(setInfo, (e: Error) => setClosed(e instanceof ApiError && e.status === 401));
  }, []);

  useEffect(() => {
    loadInfo();
    const onHash = () => {
      const next = place();
      setWhere(next);
      setCreated((c) => (c && c.id === next.project ? c : null));
    };
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, [loadInfo]);

  const home = useCallback(() => {
    window.location.hash = '#/';
  }, []);
  useEscapeTo(home);

  return (
    <div className="app">
      <header className="topbar">
        <a className="topbar__brand" href="#/">
          <img src="/markitect-mark.svg" alt="" className="topbar__mark" />
          <span>Markitect</span>
        </a>
        {info && <span className="topbar__version small">{info.version}</span>}
      </header>
      <main className="page">
        {closed ? (
          <div className="gl-empty">
            <h3 className="gl-empty__title">This window lost its connection</h3>
            <p className="gl-empty__body">Markitect was started again. Open it with markitect desktop.</p>
          </div>
        ) : where.project ? (
          <Hub key={where.project} id={where.project} init={created?.id === where.project ? created.init : undefined} onGone={home} />
        ) : (
          <Projects
            info={info}
            onAdded={(r) => {
              setCreated(r.output ? { id: r.project.id, init: { ok: r.ok, output: r.output } } : null);
              loadInfo();
              window.location.hash = `#/p/${r.project.id}`;
            }}
          />
        )}
      </main>
    </div>
  );
}

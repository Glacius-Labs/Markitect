import { useCallback, useEffect, useState } from 'react';
import { api, ApiError, type AppInfo } from './api';
import { Hub, useEscapeTo, type InitRun } from './Hub';
import { ModelArea } from './Model';
import { Projects } from './Projects';
import { hub, parse } from './route';

const place = () => parse(window.location.hash);

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
  // Escape goes up one level: from an area to its project, from a project to
  // the list.
  const up = useCallback(() => {
    window.location.hash = where.area && where.project ? hub(where.project) : '#/';
  }, [where]);
  useEscapeTo(up);

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
        ) : where.project && where.area === 'model' ? (
          <ModelArea key={where.project} id={where.project} view={where.view ?? 'reader'} item={where.item} />
        ) : where.project ? (
          <Hub key={where.project} id={where.project} init={created?.id === where.project ? created.init : undefined} onGone={home} />
        ) : (
          <Projects
            info={info}
            onAdded={(r) => {
              setCreated(r.output || !r.ok ? { id: r.project.id, init: { ok: r.ok, output: r.output } } : null);
              loadInfo();
              window.location.hash = `#/p/${r.project.id}`;
            }}
          />
        )}
      </main>
    </div>
  );
}

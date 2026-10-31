import { useEffect, useMemo, useState } from 'react';
import { api, type BaseKind, type ModelView, type Project } from './api';
import { MapView } from './MapView';
import { index } from './modeldata';
import { Organization } from './Organization';
import { Quality } from './Quality';
import { Reader } from './Reader';
import { hub, modelHref, views, type View } from './route';
import { plural } from './status';
import { Alert } from './ui';

const titles: Record<View, string> = { reader: 'Reader', map: 'Map', organization: 'Organization', quality: 'Quality' };

// How often a visible window asks whether the model changed.
const followEvery = 2000;

// useModel follows a project's model. A visible window asks again every two
// seconds; the service answers "nothing changed" unless a file, the last
// commit or the last landing did, so edits made in another editor show up.
export function useModel(id: string, base: BaseKind | undefined): { view?: ModelView; error?: string } {
  const [state, setState] = useState<{ view?: ModelView; error?: string }>({});
  useEffect(() => {
    let etag = '';
    let stopped = false;
    let busy = false;
    let timer = 0;
    const tick = async () => {
      window.clearTimeout(timer);
      if (busy || stopped) {
        return;
      }
      busy = true;
      try {
        if (document.visibilityState === 'visible') {
          const r = await api.model(id, base, etag);
          if (!stopped) {
            if (r) {
              etag = r.etag;
              setState({ view: r.view });
            } else {
              setState((s) => (s.error ? { view: s.view } : s));
            }
          }
        }
      } catch (e) {
        if (!stopped) {
          setState((s) => ({ view: s.view, error: e instanceof Error ? e.message : String(e) }));
        }
      } finally {
        busy = false;
        if (!stopped) {
          timer = window.setTimeout(tick, followEvery);
        }
      }
    };
    void tick();
    const wake = () => void tick();
    document.addEventListener('visibilitychange', wake);
    window.addEventListener('focus', wake);
    return () => {
      stopped = true;
      window.clearTimeout(timer);
      document.removeEventListener('visibilitychange', wake);
      window.removeEventListener('focus', wake);
    };
  }, [id, base]);
  return state;
}

// typing reports whether a key goes to a field rather than to the app.
export function typing(e: KeyboardEvent): boolean {
  const t = e.target;
  return t instanceof HTMLElement && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName));
}

// ModelArea is the Model area: the four views over the working tree's model,
// with the change from a base one switch away.
export function ModelArea(props: { id: string; view: View; item?: string }) {
  const { id, view, item } = props;
  const [project, setProject] = useState<Project | null>(null);
  const [diff, setDiff] = useState(false);
  const [base, setBase] = useState<BaseKind | undefined>();
  const { view: m, error } = useModel(id, base);
  const ix = useMemo(() => (m && m.initialized ? index(m) : null), [m]);

  useEffect(() => {
    api.project(id).then(setProject, () => setProject(null));
  }, [id]);

  // The digits 1 to 4 switch the views, as they enter the areas on the hub.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.altKey || e.ctrlKey || e.metaKey || typing(e) || document.querySelector('dialog[open]')) {
        return;
      }
      const n = Number(e.key);
      if (n >= 1 && n <= views.length) {
        window.location.hash = modelHref(id, views[n - 1]);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [id]);

  const name = project?.name ?? 'Project';
  const landing = m?.bases.landing;
  return (
    <>
      <nav aria-label="Breadcrumb">
        <ol className="gl-breadcrumb">
          <li>
            <a href="#/">Projects</a>
          </li>
          <li>
            <a href={hub(id)}>{name}</a>
          </li>
          <li>
            <span aria-current="page">Model</span>
          </li>
        </ol>
      </nav>
      <header className="gl-page-header">
        <div className="gl-page-header__text">
          <h1 className="gl-page-header__title">Model</h1>
          <p className="gl-page-header__description">
            {ix ? `${plural(ix.m.definitions.filter((d) => !d.library).length, 'Definition')} in ${plural(ix.m.packages.length, 'Package')}, as the working tree has it now` : 'The model as the working tree has it now'}
          </p>
        </div>
      </header>
      <div className="model-bar">
        <div className="gl-tabs" role="tablist" aria-label="Views">
          {views.map((v, i) => (
            <a key={v} className="gl-tab" role="tab" aria-selected={v === view} href={modelHref(id, v)}>
              <span className="key">{i + 1}</span>
              {titles[v]}
            </a>
          ))}
        </div>
        <div className="model-bar__diff">
          <label className="switch">
            <input type="checkbox" role="switch" className="gl-switch" checked={diff} onChange={(e) => setDiff(e.target.checked)} />
            Diff
          </label>
          {diff && m?.base && (
            <div className="gl-segmented" role="radiogroup" aria-label="Compare with">
              <button
                type="button"
                className="gl-segmented__item"
                role="radio"
                aria-checked={m.base.kind === 'landing'}
                disabled={!landing}
                title={landing ? `The last commit a Run landed, ${short(landing)}: the base apply takes` : 'No Run has landed yet'}
                onClick={() => setBase('landing')}
              >
                Last landing
              </button>
              <button
                type="button"
                className="gl-segmented__item"
                role="radio"
                aria-checked={m.base.kind === 'commit'}
                title={m.bases.commit ? `The last commit, ${short(m.bases.commit)}: what is not committed yet` : undefined}
                onClick={() => setBase('commit')}
              >
                Last commit
              </button>
            </div>
          )}
        </div>
      </div>
      {diff && m?.initialized && <p className="small muted">{compared(m)}</p>}
      {error && <Alert tone="danger" title="The model cannot be read" body={error} />}
      {m && !m.initialized && (
        <Alert tone="info" title="Markitect is not set up in this project" body="Set it up from the project's page; then the model shows here.">
          <a className="gl-button" data-size="sm" href={hub(id)}>
            Go to the project
          </a>
        </Alert>
      )}
      {ix && ix.m.errors.length > 0 && (
        <Alert tone="danger" title={`The model has ${plural(ix.m.errors.length, 'compile error')}`} body="The views show what compiles. The Quality view lists every error.">
          <a className="gl-button" data-size="sm" href={modelHref(id, 'quality')}>
            Show the errors
          </a>
        </Alert>
      )}
      {ix && view === 'reader' && <Reader ix={ix} pid={id} item={item} diff={diff} />}
      {ix && view === 'map' && <MapView ix={ix} pid={id} item={item} diff={diff} />}
      {ix && view === 'organization' && <Organization ix={ix} pid={id} item={item} diff={diff} />}
      {ix && view === 'quality' && <Quality ix={ix} pid={id} item={item} diff={diff} />}
    </>
  );
}

export function short(commit: string): string {
  return commit.slice(0, 8);
}

// compared says in words what the diff compares with.
function compared(m: ModelView): string {
  if (!m.base) {
    return 'Nothing is committed yet, so the whole model is new.';
  }
  if (m.base.kind === 'landing') {
    return `Compared with the last landing, ${short(m.base.commit)}: what apply would realize.`;
  }
  const none = m.bases.landing ? '' : ' No Run has landed yet.';
  return `Compared with the last commit, ${short(m.base.commit)}: what is not committed yet.${none}`;
}

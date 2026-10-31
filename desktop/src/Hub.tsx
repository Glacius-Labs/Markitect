import { useCallback, useEffect, useState } from 'react';
import { api, ApiError, type Project } from './api';
import { useRefreshOnFocus } from './focus';
import { nextStep, plural } from './status';
import { Alert, Badge, Dialog, Icon } from './ui';

const areas = [
  { key: '1', icon: 'layers', title: 'Model', body: 'Read the model, see its Packages and Managers, and judge its quality.' },
  { key: '2', icon: 'activity', title: 'Runs', body: 'Start apply and verify, and follow every agent while it works.' },
  { key: '3', icon: 'layout-dashboard', title: 'Statistics', body: 'How Runs go, where the work struggles, how the model develops.' },
  { key: '4', icon: 'settings', title: 'Settings', body: 'Required checks, agents and logins.' },
];

// Hub is a project's home: its state, the one next step, and the areas.
export function Hub(props: { id: string; output?: string; onGone: () => void }) {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [output, setOutput] = useState(props.output ?? '');
  const [removing, setRemoving] = useState(false);

  const { id, onGone } = props;
  const load = useCallback(() => {
    api.project(id).then(
      (p) => {
        setProject(p);
        setError('');
      },
      (e: Error) => {
        if (e instanceof ApiError && e.status === 404) {
          onGone();
          return;
        }
        setError(e.message);
      },
    );
  }, [id, onGone]);
  useEffect(load, [load]);
  useRefreshOnFocus(load);

  if (!project) {
    return error ? <Alert tone="danger" title="This project cannot be read" body={error} /> : null;
  }

  const init = async () => {
    setBusy(true);
    try {
      const r = await api.init(project.id);
      setProject(r.project);
      setOutput(r.output);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    await api.remove(project.id);
    props.onGone();
  };

  const step = nextStep(project.status);
  const check = project.status.check;
  return (
    <>
      <nav aria-label="Breadcrumb">
        <ol className="gl-breadcrumb">
          <li>
            <a href="#/">Projects</a>
          </li>
          <li>
            <span aria-current="page">{project.name}</span>
          </li>
        </ol>
      </nav>
      <header className="gl-page-header">
        <div className="gl-page-header__text">
          <h1 className="gl-page-header__title">{project.name}</h1>
          <p className="gl-page-header__description path">{project.path}</p>
        </div>
        <div className="gl-page-header__actions">
          <button className="gl-button" data-size="sm" onClick={load}>
            Check again
          </button>
          <button className="gl-button" data-variant="ghost" data-size="sm" onClick={() => setRemoving(true)}>
            Remove from list
          </button>
        </div>
      </header>
      {error && <Alert tone="danger" title="That did not work" body={error} />}
      <Alert tone={step.tone} title={step.title} body={step.body}>
        {step.action === 'init' && (
          <button className="gl-button" data-variant="primary" data-size="sm" aria-busy={busy || undefined} disabled={busy} onClick={init}>
            Set up Markitect
          </button>
        )}
      </Alert>
      {check && !check.ok && (
        <ul className="errors">
          {check.errors.map((e, i) => (
            <li key={i}>{e}</li>
          ))}
          {check.errorCount > check.errors.length && <li className="muted">and {plural(check.errorCount - check.errors.length, 'more error')}; markitect check lists them all</li>}
        </ul>
      )}
      {check && (
        <div className="gl-stats">
          <Stat label="Definitions" value={check.definitions} />
          <Stat label="Packages" value={check.packages} />
          <Stat label="Files accounted for" value={check.files} />
        </div>
      )}
      {output && (
        <details className="output">
          <summary>Output of markitect init</summary>
          <div className="gl-codeblock">
            <pre>{output}</pre>
          </div>
        </details>
      )}
      <section className="section" aria-labelledby="areas">
        <h2 className="h3" id="areas">
          Areas
        </h2>
        <div className="gl-cards">
          {areas.map((a) => (
            <div key={a.key} className="gl-card area" aria-disabled="true">
              <span className="gl-card__icon">
                <Icon name={a.icon} />
              </span>
              <h3 className="gl-card__title">
                <span className="key">{a.key}</span>
                {a.title}
              </h3>
              <p className="gl-card__body">{a.body}</p>
              <div className="gl-card__footer">
                <Badge tone="neutral">Not built yet</Badge>
              </div>
            </div>
          ))}
        </div>
      </section>
      {removing && (
        <Dialog
          title={`Remove ${project.name} from the list?`}
          onClose={() => setRemoving(false)}
          footer={
            <>
              <button className="gl-button" onClick={() => setRemoving(false)}>
                Cancel
              </button>
              <button className="gl-button" data-variant="primary" onClick={remove}>
                Remove from list
              </button>
            </>
          }
        >
          <p>The folder stays as it is. You can open it again at any time.</p>
        </Dialog>
      )}
    </>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="gl-stat">
      <span className="gl-stat__label">{label}</span>
      <span className="gl-stat__value">{value}</span>
    </div>
  );
}

// useEscapeTo goes one level up with Escape, unless a dialog is open, which
// Escape closes instead.
export function useEscapeTo(target: () => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !document.querySelector('dialog[open]')) {
        target();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [target]);
}

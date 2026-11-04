import { useCallback, useEffect, useRef, useState } from 'react';
import { api, ApiError, type Project } from './api';
import { openChat } from './Chat';
import { useRefreshOnFocus } from './focus';
import { typing } from './Model';
import { modelHref, settingsHref, statisticsHref } from './route';
import { showChat, useChatChanged } from './shown';
import { nextStep, plural } from './status';
import { Tiles } from './Tiles';
import { Alert, Dialog } from './ui';

// What the Draw the model button says to the chat that it starts.
const drawMessage = "Draw the model from this project's code.";

// InitRun is how a markitect init ended, with its output.
export interface InitRun {
  ok: boolean;
  output: string;
}

// Hub is a project's home: its state, the one next step, and the areas.
export function Hub(props: { id: string; init?: InitRun; onGone: () => void }) {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [ran, setRan] = useState<InitRun | null>(props.init ?? null);
  const [removing, setRemoving] = useState(false);

  const { id, onGone } = props;
  // Only the last reload counts, whichever answer comes first.
  const latest = useRef(0);
  const load = useCallback(() => {
    const asked = ++latest.current;
    return api.project(id).then(
      (p) => {
        if (asked !== latest.current) {
          return;
        }
        setProject(p);
        setError('');
      },
      (e: Error) => {
        if (asked !== latest.current) {
          return;
        }
        if (e instanceof ApiError && e.status === 404) {
          onGone();
          return;
        }
        setError(e.message);
      },
    );
  }, [id, onGone]);
  useEffect(() => {
    void load();
  }, [load]);
  useRefreshOnFocus(load);
  // A proposal that came or was decided changes the next step.
  useChatChanged(load);

  // While a chat draws the model the hub follows it, with the panel closed too.
  const drawing = project?.waiting?.extract?.working;
  useEffect(() => {
    if (!drawing) {
      return;
    }
    const timer = window.setInterval(() => void load(), 3000);
    return () => window.clearInterval(timer);
  }, [drawing, load]);

  // The digits enter the areas that are built.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.altKey || e.ctrlKey || e.metaKey || typing(e) || document.querySelector('dialog[open]')) {
        return;
      }
      if (e.key === '1' && project?.status.check) {
        window.location.hash = modelHref(id, 'reader');
      } else if (e.key === '3') {
        window.location.hash = statisticsHref(id);
      } else if (e.key === '4') {
        window.location.hash = settingsHref(id);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [id, project]);

  if (!project) {
    return error ? <Alert tone="danger" title="This project cannot be read" body={error} /> : null;
  }

  const init = async () => {
    setBusy(true);
    try {
      const r = await api.init(project.id);
      // A reload asked before this answer knows less than it does.
      latest.current++;
      setProject(r.project);
      setRan({ ok: r.ok, output: r.output });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const draw = async () => {
    setBusy(true);
    try {
      const c = await api.newChat(project.id, drawMessage, 'extract');
      openChat(project.id, c.id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    try {
      await api.remove(project.id);
      onGone();
    } catch (e) {
      setRemoving(false);
      setError(e instanceof Error ? e.message : String(e));
      setBusy(false);
    }
  };

  const step = nextStep(project.status, project.waiting);
  const waitingChat = step.proposal;
  const check = project.status.check;
  return (
    <div className="hub">
      <div className="hub__side">
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
      {ran && !ran.ok && <Alert tone="danger" title="markitect init did not finish" body="Its output below says why." />}
      <Alert tone={step.tone} title={step.title} body={step.body}>
        {step.action === 'init' && (
          <button className="gl-button" data-variant="primary" data-size="sm" aria-busy={busy || undefined} disabled={busy} onClick={init}>
            Set up Markitect
          </button>
        )}
        {step.chat && (
          <button className="gl-button" data-variant="primary" data-size="sm" onClick={() => openChat(project.id, step.chat!)}>
            Open the chat
          </button>
        )}
        {step.action === 'draw' && (
          <button className="gl-button" data-variant={step.chat ? undefined : 'primary'} data-size="sm" aria-busy={busy || undefined} disabled={busy} onClick={() => void draw()}>
            {step.chat ? 'Start again' : 'Draw the model'}
          </button>
        )}
        {waitingChat && (
          <>
            <button
              className="gl-button"
              data-variant="primary"
              data-size="sm"
              onClick={() => {
                showChat(project.id, waitingChat);
                window.location.hash = modelHref(project.id, 'reader');
              }}
            >
              Show in the views
            </button>
            <button className="gl-button" data-size="sm" onClick={() => openChat(project.id, waitingChat)}>
              Open the chat
            </button>
          </>
        )}
      </Alert>
      {check && !check.ok && !check.notDrawn && (
        <ul className="errors">
          {check.errors.map((e, i) => (
            <li key={i}>{e}</li>
          ))}
          {check.errorCount > check.errors.length && <li className="muted">and {plural(check.errorCount - check.errors.length, 'more error')}; markitect check lists them all</li>}
        </ul>
      )}
      {check?.notDrawn && (
        <details className="output">
          <summary>The {plural(check.errorCount, 'file')} with no place in a model yet</summary>
          <ul className="errors">
            {check.errors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
            {check.errorCount > check.errors.length && <li className="muted">and {plural(check.errorCount - check.errors.length, 'more file')}</li>}
          </ul>
        </details>
      )}
      {ran && ran.output && (
        <details className="output" open={!ran.ok}>
          <summary>Output of markitect init</summary>
          <div className="gl-codeblock">
            <pre>{ran.output}</pre>
          </div>
        </details>
      )}
      </div>
      <section className="section tiles-section" aria-labelledby="areas">
        <h2 className="gl-sr-only" id="areas">
          Areas
        </h2>
        <Tiles project={project} />
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
              <button className="gl-button" data-variant="primary" aria-busy={busy || undefined} disabled={busy} onClick={remove}>
                Remove from list
              </button>
            </>
          }
        >
          <p>The folder stays as it is. You can open it again at any time.</p>
        </Dialog>
      )}
    </div>
  );
}

// useEscapeTo goes one level up with Escape, unless a dialog is open, which
// Escape closes instead, or a field has the focus, whose Escape is its own.
export function useEscapeTo(target: () => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !typing(e) && !document.querySelector('dialog[open]')) {
        target();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [target]);
}

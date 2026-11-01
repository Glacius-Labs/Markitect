import { useEffect, useState } from 'react';
import { api, type AppSettings, type LoginState, type Project, type RequiredCheck } from './api';
import { useRefreshOnFocus } from './focus';
import { hub } from './route';
import { Alert, Badge } from './ui';

// Settings shows the project's required checks and how this Markitect runs
// its agents: the provider, model and Manager image, and the account logins.
// Until the Runs, Markitect reads them from the environment it was started
// in, as markitect apply does, and changes none of them.

export function Settings({ id }: { id: string }) {
  const [project, setProject] = useState<Project | null>(null);
  const [checks, setChecks] = useState<{ initialized: boolean; checks: RequiredCheck[]; problem?: string } | null>(null);
  const [app, setApp] = useState<AppSettings | null>(null);
  const [error, setError] = useState('');

  const load = () => {
    api.project(id).then(setProject, () => setProject(null));
    api.projectSettings(id).then(setChecks, (e: Error) => setError(e.message));
    api.settings().then(setApp, (e: Error) => setError(e.message));
  };
  useEffect(load, [id]);
  useRefreshOnFocus(load);

  return (
    <>
      <nav aria-label="Breadcrumb">
        <ol className="gl-breadcrumb">
          <li>
            <a href="#/">Projects</a>
          </li>
          <li>
            <a href={hub(id)}>{project?.name ?? 'Project'}</a>
          </li>
          <li>
            <span aria-current="page">Settings</span>
          </li>
        </ol>
      </nav>
      <header className="gl-page-header">
        <div className="gl-page-header__text">
          <h1 className="gl-page-header__title">Settings</h1>
          <p className="gl-page-header__description">The project's required checks, and how this Markitect runs its agents.</p>
        </div>
      </header>
      {error && <Alert tone="danger" title="Settings cannot be read" body={error} />}

      <section className="section">
        <h2 className="h3">This project</h2>
        <p className="small muted">
          The required checks are the one proof of a Run that does not come from an agent. They live in <span className="path">.markitect/execution.yaml</span>; edit it there for now.
        </p>
        {checks?.problem && <Alert tone="danger" title="The required checks cannot be read" body={checks.problem} />}
        {checks && !checks.initialized ? (
          <p className="muted">Markitect is not set up in this project yet.</p>
        ) : checks && checks.checks.length === 0 ? (
          !checks.problem && <p className="muted">No required checks yet.</p>
        ) : (
          checks && (
            <div className="gl-table-wrap">
              <table className="gl-table" data-density="compact">
                <thead>
                  <tr>
                    <th>Check</th>
                    <th>Command</th>
                    <th>Folder</th>
                    <th data-numeric>Time limit</th>
                    <th>Image</th>
                  </tr>
                </thead>
                <tbody>
                  {checks.checks.map((c) => (
                    <tr key={c.name}>
                      <td>{c.name}</td>
                      <td className="path">{c.command}</td>
                      <td className="path">{c.directory || '.'}</td>
                      <td data-numeric>{duration(c.timeout)}</td>
                      <td className="path">{c.environment}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
      </section>

      {app && (
        <section className="section">
          <h2 className="h3">This Markitect</h2>
          <p className="small muted">
            Chats and Runs start their agents with these settings, read from the environment Markitect was started in, the variables <span className="path">MARKITECT_*</span> that
            markitect apply reads too. Markitect sets them itself later.
          </p>
          {(app.agents.problems ?? []).map((p) => (
            <Alert key={p} tone="warning" title="No agent can start yet" body={p} />
          ))}
          <dl className="settings">
            <Setting name="Provider" value={providerName[app.agents.provider] ?? app.agents.provider} />
            <Setting name="Model" value={app.agents.model ? `${app.agents.model}, effort ${app.agents.effort}` : 'not set'} />
            <Setting name="Manager image" value={app.agents.image || 'not set'} mono />
            <Setting name="Provider network" value={app.agents.network} mono />
            <Setting name="Docker" value={app.agents.engine || 'not found'} mono />
          </dl>
          <h3 className="title">Logins</h3>
          <p className="small muted">Agents use your account login, never an API key. A login copy cannot refresh itself, so log in again on this computer when one runs out.</p>
          <div className="gl-cards logins">
            {app.logins.map((l) => (
              <Login key={l.provider} l={l} />
            ))}
          </div>
        </section>
      )}
    </>
  );
}

function Setting({ name, value, mono }: { name: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt>{name}</dt>
      <dd className={mono ? 'path' : undefined}>{value}</dd>
    </div>
  );
}

const providerName: Record<string, string> = { claude: 'Claude Code', codex: 'Codex' };

function Login({ l }: { l: LoginState }) {
  const tone = l.state === 'ok' ? 'success' : l.state === 'missing' ? 'neutral' : 'danger';
  const label = l.state === 'ok' ? (l.expires ? `Valid until ${new Date(l.expires).toLocaleString()}` : 'Logged in') : l.state === 'missing' ? 'Not logged in' : 'Not usable';
  return (
    <div className="gl-card login">
      <h4 className="gl-card__title">
        {providerName[l.provider] ?? l.provider} {l.used && <Badge tone="info">used by the agents</Badge>}
      </h4>
      <p>
        <Badge tone={tone}>{label}</Badge>
      </p>
      {l.detail && <p className="small muted">{l.detail}</p>}
      <p className="small muted">
        <span className="path">{l.file}</span>
      </p>
      <p className="small">
        Log in with the provider's own command in a terminal on this computer: <span className="path">{l.command}</span>. Using it also renews the login.
      </p>
    </div>
  );
}

// duration writes a Go duration, given in nanoseconds, the way execution.yaml
// does.
export function duration(ns: number): string {
  if (ns % 1e9 !== 0 || ns === 0) {
    return ns % 1e6 === 0 ? `${ns / 1e6}ms` : `${ns / 1e9}s`;
  }
  const s = ns / 1e9;
  if (s % 3600 === 0) {
    return `${s / 3600}h`;
  }
  if (s % 60 === 0) {
    return `${s / 60}m`;
  }
  return `${s}s`;
}

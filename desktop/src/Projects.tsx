import { useCallback, useEffect, useState } from 'react';
import { api, type AppInfo, type Project, type Result } from './api';
import { AddDialog, type Mode } from './AddDialog';
import { badge, plural } from './status';
import { Alert, Badge, Icon } from './ui';
import { useRefreshOnFocus } from './focus';

// Projects is the start screen: the user's projects and the three ways in.
export function Projects(props: { info: AppInfo | null; onAdded: (r: Result) => void }) {
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [error, setError] = useState('');
  const [dialog, setDialog] = useState<Mode | null>(null);

  const load = useCallback(() => {
    api.projects().then(
      (list) => {
        setProjects(list);
        setError('');
      },
      (e: Error) => setError(e.message),
    );
  }, []);
  useEffect(load, [load]);
  useRefreshOnFocus(load);

  const actions = (
    <>
      <button className="gl-button" data-size="sm" onClick={() => setDialog('open')}>
        <Icon name="box" /> Open folder
      </button>
      <button className="gl-button" data-size="sm" onClick={() => setDialog('clone')}>
        <Icon name="git-branch" /> Clone
      </button>
      <button className="gl-button" data-variant="primary" data-size="sm" onClick={() => setDialog('create')}>
        <Icon name="plus" /> New project
      </button>
    </>
  );

  return (
    <>
      <header className="gl-page-header">
        <div className="gl-page-header__text">
          <h1 className="gl-page-header__title">Projects</h1>
          {projects && projects.length > 0 && <p className="gl-page-header__description">{plural(projects.length, 'project')}.</p>}
        </div>
        {projects && projects.length > 0 && <div className="gl-page-header__actions">{actions}</div>}
      </header>
      {error && <Alert tone="danger" title="The project list cannot be read" body={error} />}
      {projects?.length === 0 && (
        <div className="gl-empty">
          <span className="gl-empty__icon">
            <Icon name="box" size="md" />
          </span>
          <h3 className="gl-empty__title">No projects yet</h3>
          <p className="gl-empty__body">Open a folder that is a Git repository, clone one, or start a new project.</p>
          <div className="gl-empty__actions">{actions}</div>
        </div>
      )}
      {projects && projects.length > 0 && (
        <div className="gl-cards">
          {projects.map((p) => {
            const b = badge(p.status);
            return (
              <a key={p.id} className="gl-card" href={`#/p/${p.id}`}>
                <span className="gl-card__icon">
                  <Icon name="box" />
                </span>
                <h3 className="gl-card__title">{p.name}</h3>
                <p className="gl-card__body path">{p.path}</p>
                <div className="gl-card__footer">
                  <Badge tone={b.tone}>{b.label}</Badge>
                </div>
              </a>
            );
          })}
        </div>
      )}
      {dialog && (
        <AddDialog
          mode={dialog}
          info={props.info}
          onClose={() => setDialog(null)}
          onAdded={(r) => {
            setDialog(null);
            props.onAdded(r);
          }}
        />
      )}
    </>
  );
}

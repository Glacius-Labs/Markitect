import { useState, type FormEvent } from 'react';
import { api, type AppInfo, type Result } from './api';
import { Alert, Dialog, Field } from './ui';

export type Mode = 'open' | 'create' | 'clone';

const titles: Record<Mode, string> = {
  open: 'Open a folder',
  create: 'Start a new project',
  clone: 'Clone a repository',
};

// AddDialog brings a project into the list in one of the three ways. The
// service suggests the parent folder: the one that received the last new
// project, else the home folder.
export function AddDialog(props: { mode: Mode; info: AppInfo | null; onClose: () => void; onAdded: (r: Result) => void }) {
  const { mode } = props;
  const [path, setPath] = useState('');
  const [parent, setParent] = useState(props.info?.parent ?? '');
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      const req =
        mode === 'open'
          ? { action: 'open' as const, path: path.trim() }
          : mode === 'create'
            ? { action: 'create' as const, parent: parent.trim(), name: name.trim() }
            : { action: 'clone' as const, url: url.trim(), parent: parent.trim(), name: name.trim() || undefined };
      props.onAdded(await api.add(req));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  const action = { open: 'Open', create: 'Create project', clone: 'Clone' }[mode];
  return (
    <Dialog
      title={titles[mode]}
      onClose={props.onClose}
      footer={
        <>
          <button type="button" className="gl-button" onClick={props.onClose}>
            Cancel
          </button>
          <button type="submit" form="add-project" className="gl-button" data-variant="primary" aria-busy={busy || undefined} disabled={busy}>
            {action}
          </button>
        </>
      }
    >
      <form id="add-project" className="form" onSubmit={submit}>
        {mode === 'open' && (
          <Field id="path" label="Folder" hint="A folder inside a Git repository, as a full path. Markitect opens the repository it belongs to." value={path} onChange={setPath} mono />
        )}
        {mode === 'clone' && <Field id="url" label="Repository URL" hint="Any URL git clone accepts." value={url} onChange={setUrl} mono />}
        {mode !== 'open' && (
          <>
            <Field id="parent" label="Parent folder" hint="The folder that receives the project, as a full path." value={parent} onChange={setParent} mono />
            <Field
              id="name"
              label="Name"
              optional={mode === 'clone'}
              hint={mode === 'create' ? 'The new folder. Markitect makes it a Git repository and runs markitect init in it.' : 'The new folder; by default the repository name.'}
              value={name}
              onChange={setName}
            />
          </>
        )}
        {error && <Alert tone="danger" title="That did not work" body={error} />}
      </form>
    </Dialog>
  );
}

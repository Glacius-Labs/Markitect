import { useEffect, useRef, useState, type FormEvent } from 'react';
import { api, type Conflict, type Definition } from './api';
import { packageName, type Index } from './modeldata';
import { modelHref } from './route';
import { Alert, Dialog, Field } from './ui';

// Hand edits: a Definition opened in an editor saves straight into the
// working tree, and the views show it, with what check says, once saved. A
// draft survives moving around the app in the same window, with the text it
// started from, so a save after the file changed elsewhere still asks.

const draftKey = (pid: string, id: string) => `markitect.draft.${pid}.${id}`;
const openKey = (pid: string, id: string) => `markitect.open.${pid}.${id}`;

// A new Definition opens in the editor if the Reader shows it soon after.
const openWithin = 60_000;

interface Draft {
  text: string;
  opened: string; // the text the editor started from
  base: string; // its digest
}

function read(key: string): string | null {
  try {
    return sessionStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string | null) {
  try {
    if (value === null) {
      sessionStorage.removeItem(key);
    } else {
      sessionStorage.setItem(key, value);
    }
  } catch {
    // No storage in this window: the draft lives only while the editor does.
  }
}

function readDraft(key: string): Draft | null {
  try {
    const d = JSON.parse(read(key) ?? 'null') as Draft | null;
    return d && typeof d.text === 'string' && typeof d.opened === 'string' && typeof d.base === 'string' ? d : null;
  } catch {
    return null;
  }
}

// startsEditing reports whether a Definition opens in the editor: it has a
// draft, or it was just created.
export function startsEditing(pid: string, id: string): boolean {
  const at = Number(read(openKey(pid, id)));
  write(openKey(pid, id), null);
  return Date.now() - at < openWithin || readDraft(draftKey(pid, id)) !== null;
}

export function Editor(props: { pid: string; def: Definition; onClose: () => void; refresh: () => void }) {
  const { pid, def, onClose, refresh } = props;
  const key = draftKey(pid, def.id);
  const [start] = useState(() => readDraft(key) ?? { text: def.text, opened: def.text, base: def.digest });
  const [opened, setOpened] = useState(start.opened);
  const [base, setBase] = useState(start.base);
  const [text, setText] = useState(start.text);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState<Conflict | null>(null);
  // The digest of the text just saved: the editor closes once the views
  // show it, so a second Edit starts from it.
  const [saved, setSaved] = useState('');
  const dirty = text !== opened;

  useEffect(() => {
    write(key, dirty ? JSON.stringify({ text, opened, base } satisfies Draft) : null);
  }, [key, text, opened, base, dirty]);

  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    if (!saved) {
      return;
    }
    if (def.digest === saved) {
      close.current();
      return;
    }
    const t = window.setTimeout(() => close.current(), 10_000);
    return () => window.clearTimeout(t);
  }, [saved, def.digest]);

  const save = async (against: string) => {
    if (busy || saved) {
      return;
    }
    setBusy(true);
    setError('');
    try {
      const r = await api.saveDefinition(pid, def.id, text, against);
      if (r.ok) {
        write(key, null);
        setOpened(text);
        setSaved(r.digest);
        refresh();
        return;
      }
      setConflict(r.conflict);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
    setBusy(false);
  };
  const cancel = () => {
    write(key, null);
    onClose();
  };

  return (
    <div className="editor">
      {conflict && (
        <Alert tone="warning" title={conflict.error} body="Another editor saved this file while you worked on it. Keep your text, which replaces theirs, or take theirs and drop yours.">
          {dirty && (
            <button className="gl-button" data-size="sm" disabled={busy} onClick={() => void save(conflict.digest)}>
              Keep mine
            </button>
          )}
          {conflict.digest && (
            <button
              className="gl-button"
              data-size="sm"
              disabled={busy}
              onClick={() => {
                setText(conflict.text);
                setOpened(conflict.text);
                setBase(conflict.digest);
                setConflict(null);
                setBusy(false);
              }}
            >
              Take theirs
            </button>
          )}
        </Alert>
      )}
      {!conflict && !saved && def.digest !== base && <Alert tone="info" title="The file changed in another editor" body="Saving asks which text to keep." />}
      {error && <Alert tone="danger" title="Not saved" body={error} />}
      <textarea
        className="gl-textarea editor__text"
        aria-label={`The text of ${def.title || def.id}`}
        value={text}
        spellCheck={false}
        autoFocus
        readOnly={!!saved}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
            e.preventDefault();
            if (dirty) {
              void save(base);
            }
          } else if (e.key === 'Escape' && !dirty) {
            e.preventDefault();
            cancel();
          }
        }}
      />
      <div className="editor__actions">
        <button className="gl-button" data-variant="primary" aria-busy={busy || undefined} disabled={busy || !dirty} onClick={() => void save(base)}>
          {saved ? 'Saved' : 'Save'}
        </button>
        <button className="gl-button" disabled={busy} onClick={cancel}>
          {dirty ? 'Discard' : 'Close'}
        </button>
        <span className="small muted">Ctrl+S saves into {def.file}; markitect check runs on it again.</span>
      </div>
    </div>
  );
}

const namePattern = /^[A-Za-z0-9][A-Za-z0-9_-]*$/;

// NewDefinition writes a new Definition into a Package and opens it in the
// editor.
export function NewDefinition(props: { ix: Index; pid: string; pkg: string; onClose: () => void; refresh: () => void }) {
  const { ix, pid, pkg, onClose, refresh } = props;
  // A Kind marked internal serves only its own Package and those below.
  const kinds = ix.m.definitions
    .filter((d) => d.kind === 'Kind' && (!d.internal || d.package === '' || pkg === d.package || pkg.startsWith(`${d.package}/`)))
    .map((d) => d.id)
    .sort();
  const [name, setName] = useState('');
  const [title, setTitle] = useState('');
  const [kind, setKind] = useState(kinds.includes('std/Concept') ? 'std/Concept' : (kinds[0] ?? ''));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const id = pkg === '' ? name.trim() : `${pkg}/${name.trim()}`;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!namePattern.test(name.trim())) {
      setError('A name starts with a letter or digit and holds only letters, digits, - and _.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const r = await api.saveDefinition(pid, id, `---\nkind: ${kind}\n---\n# ${title.trim() || name.trim()}\n\n`, '');
      if (!r.ok) {
        setError(r.conflict.error);
        setBusy(false);
        return;
      }
      write(openKey(pid, id), String(Date.now()));
      refresh();
      window.location.hash = modelHref(pid, 'reader', id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setBusy(false);
    }
  };

  return (
    <Dialog
      title={`New Definition in ${packageName(pkg)}`}
      onClose={onClose}
      footer={
        <>
          <button type="button" className="gl-button" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" form="new-definition" className="gl-button" data-variant="primary" aria-busy={busy || undefined} disabled={busy}>
            Create and edit
          </button>
        </>
      }
    >
      <form id="new-definition" className="form" onSubmit={submit}>
        <Field id="new-name" label="Name" hint={`The file name without .md; its ID becomes ${id || '…'}.`} value={name} onChange={setName} mono />
        <div className="gl-field">
          <label className="gl-field__label" htmlFor="new-kind">
            Kind
          </label>
          <select className="gl-select" id="new-kind" value={kind} onChange={(e) => setKind(e.target.value)}>
            {kinds.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </div>
        <Field id="new-title" label="Title" optional hint="The heading; the name when left empty." value={title} onChange={setTitle} />
        {error && <Alert tone="danger" title="Not created" body={error} />}
      </form>
    </Dialog>
  );
}

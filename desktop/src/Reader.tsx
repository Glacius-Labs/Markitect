import { Children, useEffect, useMemo, useState, type ReactNode } from 'react';
import type { Definition, Package } from './api';
import { diffLines } from './diff';
import { Editor, NewDefinition, startsEditing } from './Editor';
import { render } from './markdown';
import { changeLabel, changeTone, depth, firstSentence, packageName, since, type Index } from './modeldata';
import { modelHref } from './route';
import { Alert, Badge } from './ui';

interface Props {
  ix: Index;
  pid: string;
  item?: string;
  diff: boolean;
  refresh: () => void;
  // The views show a chat's proposal, which is not in the working tree to
  // edit.
  editable: boolean;
}

// Reader shows the model as documentation: every Definition with its links,
// who refers to it, which files realize it and which Managers see it.
export function Reader({ ix, pid, item, diff, refresh, editable }: Props) {
  const { m, defs, pkgs } = ix;
  const target = item ?? (defs.has('project') ? 'project' : '');
  const def = defs.get(target);
  const pkg = def ? undefined : pkgs.get(target);
  const removed = m.diff.definitions.find((d) => d.id === target && d.change === 'removed');
  // Another Definition opens at its top. The block keeps scrollTo's result,
  // a promise in newer browsers, from being taken for a cleanup.
  useEffect(() => {
    window.scrollTo({ top: 0 });
  }, [target]);
  return (
    <div className="reader">
      <ReaderNav ix={ix} pid={pid} selected={target} diff={diff} />
      <article className="reader__page" aria-live="polite">
        {def ? (
          <DefinitionPage key={def.id} ix={ix} pid={pid} def={def} diff={diff} refresh={refresh} editable={editable} />
        ) : pkg ? (
          <PackagePage ix={ix} pid={pid} pkg={pkg} diff={diff} refresh={refresh} editable={editable} />
        ) : removed ? (
          <>
            <Alert tone="warning" title={`${removed.title} was removed`} body={`${removed.id} is in the base but no longer in the working tree.`} />
            <TextDiff before={removed.baseText ?? ''} after="" />
          </>
        ) : (
          <Alert tone="info" title="Not in the model" body={`${target} is neither a Definition nor a Package of the working tree.`} />
        )}
      </article>
    </div>
  );
}

function ReaderNav({ ix, pid, selected, diff }: { ix: Index; pid: string; selected: string; diff: boolean }) {
  const [filter, setFilter] = useState('');
  const f = filter.trim().toLowerCase();
  const match = (d: Definition) => f === '' || d.title.toLowerCase().includes(f) || d.id.toLowerCase().includes(f);
  const libraries = ix.m.definitions.filter((d) => d.library && match(d));
  const removed = diff ? ix.m.diff.definitions.filter((d) => d.change === 'removed') : [];
  const item = (d: { id: string; title: string; change?: string }, extra?: ReactNode) => (
    <a key={d.id} className="gl-nav__item reader__item" href={modelHref(pid, 'reader', d.id)} aria-current={d.id === selected ? 'page' : undefined} title={d.id}>
      <span className="reader__title">{d.title || d.id}</span>
      {extra}
      {diff && d.change && <Badge tone={changeTone[d.change as keyof typeof changeTone]}>{changeLabel[d.change as keyof typeof changeLabel]}</Badge>}
    </a>
  );
  return (
    <nav className="reader__nav" aria-label="Definitions">
      <input className="gl-input" type="search" placeholder="Find a Definition" aria-label="Find a Definition" value={filter} onChange={(e) => setFilter(e.target.value)} spellCheck={false} />
      <div className="gl-nav reader__list">
        {ix.tree.map((p) => {
          const own = p.own.map((id) => ix.defs.get(id)!).filter((d) => d && match(d));
          if (f !== '' && own.length === 0 && !p.path.toLowerCase().includes(f)) {
            return null;
          }
          return (
            <div key={p.path} className="reader__group" data-depth={Math.min(depth(p.path), 4)}>
              <a className="gl-nav__group reader__package" href={modelHref(pid, 'reader', p.path)} aria-current={p.path === selected ? 'page' : undefined}>
                {packageName(p.path)}
                {diff && p.change && <Badge tone={changeTone[p.change]}>{changeLabel[p.change]}</Badge>}
              </a>
              {own.map((d) => item(d, d.internal ? <span className="reader__hint">internal</span> : undefined))}
            </div>
          );
        })}
        {removed.length > 0 && (
          <div className="reader__group">
            <span className="gl-nav__group">Removed {since(ix.m)}</span>
            {removed.map((d) => item(d))}
          </div>
        )}
        {libraries.length > 0 && (
          <details className="reader__group reader__libraries" open={f !== ''}>
            <summary className="gl-nav__group">Kinds from Libraries</summary>
            {libraries.map((d) => item(d))}
          </details>
        )}
      </div>
    </nav>
  );
}

function DefinitionPage({ ix, pid, def, diff, refresh, editable }: { ix: Index; pid: string; def: Definition; diff: boolean; refresh: () => void; editable: boolean }) {
  const [opened, setEditing] = useState(() => !def.library && startsEditing(pid, def.id));
  const editing = opened && editable;
  const resolved = useMemo(() => new Map(def.links.map((l) => [l.raw, { target: l.target, package: l.package }])), [def]);
  const html = useMemo(() => render(def.text, { resolve: (raw) => resolved.get(raw), href: (t) => modelHref(pid, 'reader', t) }), [def, resolved, pid]);
  const errors = ix.m.errors.filter((e) => e.definition === def.id);
  const warnings = ix.m.lint.filter((e) => e.definition === def.id);
  const change = ix.m.diff.definitions.find((c) => c.id === def.id);
  const kind = ix.defs.get(def.kind);
  const linked = [...new Map(def.links.filter((l) => !l.kind).map((l) => [l.target || `?${l.raw}`, l])).values()];
  const ref = (id: string) => <a href={modelHref(pid, 'reader', id)}>{ix.defs.get(id)?.title ?? id}</a>;
  const pkgLink = (path: string, view: 'reader' | 'organization' = 'reader') => (
    <a key={path} href={modelHref(pid, view, path)}>
      {packageName(path)}
    </a>
  );
  return (
    <>
      <div className="def__meta">
        {kind ? (
          <a className="gl-badge" href={modelHref(pid, 'reader', def.kind)} title="Its Kind">
            {def.kind}
          </a>
        ) : (
          <Badge tone="danger">{def.kind || 'no Kind'}</Badge>
        )}
        {def.library ? <span className="muted">Library {def.library}</span> : <span className="muted">in {pkgLink(def.package)}</span>}
        {def.internal && <Badge tone="info">internal</Badge>}
        {diff && change && (
          <Badge tone={changeTone[change.change]}>
            {changeLabel[change.change]} {since(ix.m)}
          </Badge>
        )}
        {def.file && <span className="path small def__file">{def.file}</span>}
        {!def.library && editable && !editing && (
          <button className="gl-button" data-size="sm" onClick={() => setEditing(true)}>
            Edit
          </button>
        )}
      </div>
      {errors.map((e, i) => (
        <Alert key={`e${i}`} tone="danger" title={e.code} body={`${e.line ? `Line ${e.line}: ` : ''}${e.message}`} />
      ))}
      {warnings.map((e, i) => (
        <Alert key={`w${i}`} tone="warning" title={e.code} body={e.message} />
      ))}
      {editing ? (
        <Editor pid={pid} def={def} onClose={() => setEditing(false)} refresh={refresh} />
      ) : diff && change ? (
        <TextDiff before={change.baseText ?? ''} after={def.text} />
      ) : (
        <div className="prose" dangerouslySetInnerHTML={{ __html: html }} />
      )}
      <div className="facts">
        {!def.library && (
          <Fact title="Links to" empty="No links">
            {linked.map((l) =>
              !l.target ? (
                <li key={l.raw}>
                  <span className="wikilink" data-broken>
                    {l.raw}
                  </span>{' '}
                  <span className="muted small">resolves to nothing</span>
                </li>
              ) : (
                <li key={l.target}>{l.package ? <>Package {pkgLink(l.target)}</> : ref(l.target)}</li>
              ),
            )}
          </Fact>
        )}
        <Fact title={def.library ? 'Used by' : 'Referenced by'} empty="Nothing refers to it">
          {def.referencedBy.map((r, i) => (
            <li key={i}>
              {ref(r.from)} {r.via === 'kind' && <span className="muted small">as its Kind</span>}
            </li>
          ))}
        </Fact>
        {!def.library && (
          <Fact title="Realized by" empty="No file realizes it">
            {def.realizedBy.map((r) => (
              <li key={r.file}>
                <span className="path">{r.file}</span>
                {r.reason && <span className="muted small"> {r.reason}</span>}
              </li>
            ))}
          </Fact>
        )}
        {def.seenBy && (
          <Fact title="Seen by" empty="No Manager">
            {def.seenBy.full.length > 0 && <li>In full: {join(def.seenBy.full.map((p) => pkgLink(p, 'organization')))}</li>}
            {def.seenBy.contents.length > 0 && <li>By title and first sentence: {join(def.seenBy.contents.map((p) => pkgLink(p, 'organization')))}</li>}
          </Fact>
        )}
      </div>
    </>
  );
}

function PackagePage({ ix, pid, pkg, diff, refresh, editable }: { ix: Index; pid: string; pkg: Package; diff: boolean; refresh: () => void; editable: boolean }) {
  const [creating, setCreating] = useState(false);
  const errors = ix.m.errors.filter((e) => e.package === pkg.path && !e.definition);
  const warnings = ix.m.lint.filter((e) => e.package === pkg.path && !e.definition);
  return (
    <>
      <div className="def__meta">
        <Badge tone="neutral">Package</Badge>
        {pkg.path !== '' && (
          <span className="muted">
            inside <a href={modelHref(pid, 'reader', pkg.parent)}>{packageName(pkg.parent)}</a>
          </span>
        )}
        {diff && pkg.change && (
          <Badge tone={changeTone[pkg.change]}>
            {changeLabel[pkg.change]} {since(ix.m)}
          </Badge>
        )}
        <a href={modelHref(pid, 'map', pkg.path)}>On the Map</a>
        <a href={modelHref(pid, 'organization', pkg.path)}>Its Manager</a>
        {editable && (
          <button className="gl-button def__file" data-size="sm" onClick={() => setCreating(true)}>
            New Definition
          </button>
        )}
      </div>
      {creating && <NewDefinition ix={ix} pid={pid} pkg={pkg.path} onClose={() => setCreating(false)} refresh={refresh} />}
      <h1 className="h2 reader__heading">{packageName(pkg.path)}</h1>
      {errors.map((e, i) => (
        <Alert key={`e${i}`} tone="danger" title={e.code} body={e.message} />
      ))}
      {warnings.map((e, i) => (
        <Alert key={`w${i}`} tone="warning" title={e.code} body={e.message} />
      ))}
      <div className="facts">
        <Fact title="Owns" empty={pkg.path === '' ? 'Every file no other Package claims' : 'No path pattern yet'}>
          {pkg.patterns.map((p) => (
            <li key={p} className="path">
              {p}
            </li>
          ))}
          {pkg.files > 0 && <li className="muted small">{pkg.files} files accounted for</li>}
        </Fact>
        <Fact title="Packages inside" empty="None">
          {pkg.children.map((c) => (
            <li key={c}>
              <a href={modelHref(pid, 'reader', c)}>{c}</a>
            </li>
          ))}
        </Fact>
      </div>
      <section className="section">
        <h2 className="h3">Definitions</h2>
        {pkg.own.length === 0 ? (
          <p className="muted">It has no Definitions of its own.</p>
        ) : (
          <ul className="def-list">
            {pkg.own.map((id) => {
              const d = ix.defs.get(id)!;
              return (
                <li key={id}>
                  <a href={modelHref(pid, 'reader', id)}>{d.title || id}</a> <span className="muted small">{d.kind}</span>
                  {diff && d.change && <Badge tone={changeTone[d.change]}>{changeLabel[d.change]}</Badge>}
                  <p className="small muted">{firstSentence(d.text)}</p>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </>
  );
}

function Fact({ title, empty, children }: { title: string; empty: string; children: ReactNode }) {
  const list = Children.toArray(children);
  return (
    <section className="gl-card fact">
      <h2 className="gl-card__title">{title}</h2>
      {list.length === 0 ? <p className="muted small">{empty}</p> : <ul className="fact__list">{list}</ul>}
    </section>
  );
}

function join(parts: ReactNode[]): ReactNode {
  return parts.flatMap((p, i) => (i === 0 ? [p] : [', ', p]));
}

// TextDiff shows how a Definition's text changed, line by line.
export function TextDiff({ before, after }: { before: string; after: string }) {
  const lines = useMemo(() => diffLines(before, after), [before, after]);
  const mark = { same: ' ', add: '+', del: '-' } as const;
  return (
    <div className="gl-codeblock textdiff" aria-label="Changes since the base">
      <pre>
        {lines.map((l, i) => (
          <span key={i} className="textdiff__line" data-op={l.op}>
            <span className="textdiff__mark" aria-hidden="true">
              {mark[l.op]}
            </span>
            {l.op === 'add' && <span className="gl-sr-only">added: </span>}
            {l.op === 'del' && <span className="gl-sr-only">removed: </span>}
            {l.text}
            {'\n'}
          </span>
        ))}
      </pre>
    </div>
  );
}

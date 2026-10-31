import type { Item } from './api';
import { short } from './Model';
import { changeLabel, changeTone, items, leaf, packageName, type Index } from './modeldata';
import { modelHref } from './route';
import { plural } from './status';
import { Alert, Badge } from './ui';

interface Props {
  ix: Index;
  pid: string;
  item?: string;
  diff: boolean;
}

// Organization draws the Manager tree as an organization chart: one Manager
// per Package, what it is responsible for, and its checklist for the current
// change as markitect impact derives it. With the diff on, only the Managers
// that work stand out.
export function Organization({ ix, pid, item, diff }: Props) {
  const { m, pkgs } = ix;
  const selected = item !== undefined && pkgs.has(item) ? item : undefined;
  const im = m.impact;
  return (
    <>
      <p className="small muted">
        {im
          ? im.full
            ? m.impactNote
            : `${plural(im.packages?.length ?? 0, 'Manager')} would work on the change${diff || !m.base || m.base.kind === 'tree' ? '' : ` since the ${m.base.kind === 'landing' ? 'last landing' : 'last commit'}, ${short(m.base.commit ?? '')}`}.`
          : m.impactNote}
      </p>
      <div className="org">
        <div className={`org__chart gl-card${diff ? ' org__chart--focus' : ''}`}>
          <ul className="org__tree">
            <Manager ix={ix} pid={pid} path="" selected={selected} />
          </ul>
        </div>
        <aside className="org__side">{selected !== undefined ? <ManagerPanel ix={ix} pid={pid} path={selected} diff={diff} /> : <p className="muted">Select a Manager to see what it is responsible for and its checklist.</p>}</aside>
      </div>
    </>
  );
}

function Manager({ ix, pid, path, selected }: { ix: Index; pid: string; path: string; selected?: string }) {
  const p = ix.pkgs.get(path);
  if (!p) {
    return null;
  }
  const works = ix.working.has(path);
  const n = items(ix.checklists.get(path));
  return (
    <li>
      <a className="org__card" href={modelHref(pid, 'organization', path)} aria-current={path === selected ? 'true' : undefined} data-works={works || undefined} title={packageName(path)}>
        <span className="org__name">{leaf(path)}</span>
        <span className="org__sub">{plural(p.own.length, 'Definition')}</span>
        {works ? <Badge tone="info">{n > 0 ? plural(n, 'item') : 'integrates'}</Badge> : <span className="org__idle">no work</span>}
      </a>
      {p.children.length > 0 && (
        <ul>
          {[...p.children].sort().map((c) => (
            <Manager key={c} ix={ix} pid={pid} path={c} selected={selected} />
          ))}
        </ul>
      )}
    </li>
  );
}

function ManagerPanel({ ix, pid, path, diff }: { ix: Index; pid: string; path: string; diff: boolean }) {
  const p = ix.pkgs.get(path)!;
  const c = ix.checklists.get(path);
  const works = ix.working.has(path);
  const files = (ix.m.impact?.reachedFiles ?? []).filter((f) => f.owner === path);
  const def = (id: string) => (
    <a href={modelHref(pid, 'reader', id)} title={id}>
      {ix.defs.get(id)?.title ?? id}
    </a>
  );
  const list = (title: string, xs: Item[] | null | undefined) =>
    xs && xs.length > 0 ? (
      <>
        <h4 className="label">{title}</h4>
        <ul className="checklist">
          {xs.map((x) => (
            <li key={x.definition}>
              {def(x.definition)}
              <span className="muted small"> {x.reasons.join('; ')}</span>
            </li>
          ))}
        </ul>
      </>
    ) : null;
  return (
    <>
      <div className="def__meta">
        <Badge tone="neutral">Manager</Badge>
        {works ? <Badge tone="info">works on the change</Badge> : <span className="muted">no work on the change</span>}
        <a href={modelHref(pid, 'reader', path)}>Read its Package</a>
        <a href={modelHref(pid, 'map', path)}>On the Map</a>
      </div>
      <h2 className="h3 pkg-name">{packageName(path)}</h2>
      <section className="section">
        <h3 className="title">Responsible for</h3>
        {p.own.length === 0 ? (
          <p className="muted small">No Definitions of its own.</p>
        ) : (
          <ul className="checklist">
            {p.own.map((id) => {
              const d = ix.defs.get(id);
              return (
                <li key={id}>
                  {def(id)} <span className="muted small">{d?.kind}</span>
                  {diff && d?.change && <Badge tone={changeTone[d.change]}>{changeLabel[d.change]}</Badge>}
                </li>
              );
            })}
          </ul>
        )}
        <p className="small">
          {p.patterns.length > 0 ? (
            <>
              Owns <span className="path">{p.patterns.join(', ')}</span>, {plural(p.files, 'file')} accounted for.
            </>
          ) : path === '' ? (
            `Owns every file no other Package claims, ${plural(p.files, 'file')} accounted for.`
          ) : (
            'Claims no path pattern yet, so it owns no file.'
          )}
        </p>
        {p.children.length > 0 && (
          <p className="small">
            Leads{' '}
            {p.children.map((ch, i) => (
              <span key={ch}>
                {i > 0 && ', '}
                <a href={modelHref(pid, 'organization', ch)}>{leaf(ch)}</a>
              </span>
            ))}
            , and integrates their work.
          </p>
        )}
      </section>
      <section className="section">
        <h3 className="title">Checklist for the change</h3>
        {!ix.m.impact ? (
          <p className="muted small">{ix.m.impactNote}</p>
        ) : !works ? (
          <p className="muted small">The change does not reach this Manager.</p>
        ) : items(c) === 0 && files.length === 0 ? (
          <p className="muted small">No items of its own; it reviews and integrates its children's work.</p>
        ) : (
          <>
            {list('Own', c?.own)}
            {list('Inherited', c?.inherited)}
            {c?.findings && c.findings.length > 0 && <p className="small">{plural(c.findings.length, 'Finding')}</p>}
            {c?.stale && c.stale.length > 0 && (
              <>
                <h4 className="label">Stale realization</h4>
                <ul className="checklist">
                  {c.stale.map((s, i) => (
                    <li key={i} className="small">
                      {s.code}: <span className="path">{s.file ?? s.target ?? s.pageFile}</span>
                    </li>
                  ))}
                </ul>
              </>
            )}
            {files.length > 0 && (
              <>
                <h4 className="label">Files the change reaches</h4>
                <ul className="checklist">
                  {files.map((f) => (
                    <li key={f.file} className="small">
                      <span className="path">{f.file}</span> {f.change && <span className="muted">{f.change}</span>}
                    </li>
                  ))}
                </ul>
              </>
            )}
          </>
        )}
      </section>
      {ix.m.impact && ix.m.impact.notes && ix.m.impact.notes.length > 0 && <Alert tone="info" title="Notes" body={ix.m.impact.notes.join(' ')} />}
    </>
  );
}

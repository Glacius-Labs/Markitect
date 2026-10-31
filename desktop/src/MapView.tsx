import { useMemo } from 'react';
import type { Edge } from './api';
import { layout } from './layout';
import { changeLabel, changeTone, packageName, since, type Index } from './modeldata';
import { modelHref } from './route';
import { Alert, Badge } from './ui';

interface Props {
  ix: Index;
  pid: string;
  item?: string;
  diff: boolean;
}

// MapView draws the Package graph: a Package above the Packages it depends
// on, cycles and Packages with compile errors marked, and with the diff on
// what is new or gone since the base.
export function MapView({ ix, pid, item, diff }: Props) {
  const { m, pkgs } = ix;
  const selected = item !== undefined && pkgs.has(item) ? item : undefined;
  const gone = useMemo(() => (diff ? m.diff.edges.filter((e) => e.change === 'removed' && pkgs.has(e.from) && pkgs.has(e.to)) : []), [diff, m, pkgs]);
  const drawing = useMemo(
    () =>
      layout(
        m.packages.map((p) => ({ id: p.path, label: packageName(p.path) })),
        [...m.edges, ...gone],
      ),
    [m, gone],
  );
  const added = new Set(diff ? m.diff.edges.filter((e) => e.change === 'added').map((e) => `${e.from}\u0000${e.to}`) : []);
  const removed = new Set(gone.map((e) => `${e.from}\u0000${e.to}`));
  const cycle = new Set(m.edges.filter((e) => e.cycle).map((e) => `${e.from}\u0000${e.to}`));
  const broken = new Set(m.errors.map((e) => e.package));
  const inCycle = new Set(m.cycles.flat());

  const edgeClass = (key: string, from: string, to: string) => {
    const c = ['map__edge'];
    if (removed.has(key)) c.push('map__edge--removed');
    else if (added.has(key)) c.push('map__edge--added');
    if (cycle.has(key)) c.push('map__edge--cycle');
    if (selected !== undefined && (from === selected || to === selected)) c.push('map__edge--active');
    else if (selected !== undefined) c.push('map__edge--quiet');
    return c.join(' ');
  };
  const marker = (cls: string) =>
    cls.includes('--removed') ? 'removed' : cls.includes('--cycle') ? 'cycle' : cls.includes('--added') ? 'added' : cls.includes('--active') ? 'active' : 'plain';

  return (
    <div className="map">
      <div
        className="map__canvas gl-card"
        onClick={(e) => {
          // A click beside the Packages, on the card or the drawing's ground,
          // lets go of the selection.
          const t = e.target as Element;
          if (t === e.currentTarget || t.tagName.toLowerCase() === 'svg') {
            window.location.hash = modelHref(pid, 'map');
          }
        }}
      >
        <svg width={drawing.width} height={drawing.height} viewBox={`0 0 ${drawing.width} ${drawing.height}`} role="group" aria-label="The Package graph">
          <defs>
            {['plain', 'active', 'cycle', 'added', 'removed'].map((k) => (
              <marker key={k} id={`arrow-${k}`} className={`map__arrow map__arrow--${k}`} viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                <path d="M0,0 L10,5 L0,10 z" />
              </marker>
            ))}
          </defs>
          {drawing.edges.map((e) => {
            const key = `${e.from}\u0000${e.to}`;
            const cls = edgeClass(key, e.from, e.to);
            return (
              <path key={key} d={e.d} className={cls} markerEnd={`url(#arrow-${marker(cls)})`}>
                <title>{`${packageName(e.from)} depends on ${packageName(e.to)}`}</title>
              </path>
            );
          })}
          {drawing.nodes.map((n) => {
            const p = pkgs.get(n.id)!;
            const c = ['map__node'];
            if (n.id === selected) c.push('map__node--selected');
            if (broken.has(n.id) || inCycle.has(n.id)) c.push('map__node--broken');
            if (diff && p.change === 'added') c.push('map__node--added');
            return (
              <a key={n.id} href={modelHref(pid, 'map', n.id)} className={c.join(' ')} aria-label={`Package ${packageName(n.id)}`}>
                <rect x={n.x} y={n.y} width={n.w} height={n.h} rx="6" />
                <text x={n.x + n.w / 2} y={n.y + n.h / 2} dominantBaseline="central" textAnchor="middle">
                  {n.label}
                </text>
              </a>
            );
          })}
        </svg>
      </div>
      <aside className="map__side">
        {selected !== undefined ? <PackagePanel ix={ix} pid={pid} path={selected} diff={diff} /> : <Overview ix={ix} pid={pid} diff={diff} />}
      </aside>
    </div>
  );
}

function Overview({ ix, pid, diff }: { ix: Index; pid: string; diff: boolean }) {
  const { m } = ix;
  const gonePackages = diff ? m.diff.packages.filter((p) => p.change === 'removed') : [];
  return (
    <>
      <ul className="gl-legend">
        <li className="legend--plain">depends on</li>
        <li className="legend--cycle">in a cycle</li>
        {diff && <li className="legend--added">new {since(m)}</li>}
        {diff && <li className="legend--removed">gone {since(m)}</li>}
      </ul>
      {m.cycles.map((c) => (
        <Alert key={c.join()} tone="danger" title="A cycle" body={`${c.map(packageName).join(', ')} depend on each other. Move the shared concept into a common parent, or invert one dependency.`} />
      ))}
      {gonePackages.length > 0 && <Alert tone="warning" title={`Packages removed ${since(m)}`} body={gonePackages.map((p) => p.path).join(', ')} />}
      <section className="section">
        <h2 className="h3">Dependencies</h2>
        {m.edges.length === 0 ? (
          <p className="muted">No Package depends on another.</p>
        ) : (
          <ul className="dep-list">
            {m.edges.map((e) => (
              <DepLine key={`${e.from}>${e.to}`} pid={pid} e={e} />
            ))}
          </ul>
        )}
        <p className="small muted">Select a Package to see its links. Links up the tree, to a parent's Definitions, are free and not drawn.</p>
      </section>
    </>
  );
}

function DepLine({ pid, e, show }: { pid: string; e: Edge; show?: 'from' | 'to' }) {
  const from = <a href={modelHref(pid, 'map', e.from)}>{packageName(e.from)}</a>;
  const to = <a href={modelHref(pid, 'map', e.to)}>{packageName(e.to)}</a>;
  return (
    <li>
      {show === 'to' ? to : show === 'from' ? from : (
        <>
          {from} → {to}
        </>
      )}{' '}
      <span className="muted small">
        {e.uses.length} {e.uses.length === 1 ? 'link' : 'links'}
      </span>
      {e.cycle && (
        <>
          {' '}
          <Badge tone="danger">cycle</Badge>
        </>
      )}
      <ul className="dep-list__uses">
        {e.uses.map((u, i) => (
          <li key={i} className="small">
            <a href={modelHref(pid, 'reader', u.from)}>{u.from}</a> {u.kind ? 'is of' : 'links'} <a href={modelHref(pid, 'reader', u.target)}>{u.target}</a>
          </li>
        ))}
      </ul>
    </li>
  );
}

function PackagePanel({ ix, pid, path, diff }: { ix: Index; pid: string; path: string; diff: boolean }) {
  const { m } = ix;
  const p = ix.pkgs.get(path)!;
  const out = m.edges.filter((e) => e.from === path);
  const into = m.edges.filter((e) => e.to === path);
  const errors = m.errors.filter((e) => e.package === path);
  const warnings = m.lint.filter((e) => e.package === path);
  return (
    <>
      <div className="def__meta">
        <Badge tone="neutral">Package</Badge>
        {diff && p.change && (
          <Badge tone={changeTone[p.change]}>
            {changeLabel[p.change]} {since(m)}
          </Badge>
        )}
        <a href={modelHref(pid, 'reader', path)}>Read it</a>
        <a href={modelHref(pid, 'organization', path)}>Its Manager</a>
      </div>
      <h2 className="h3 pkg-name">{packageName(path)}</h2>
      <dl className="measures">
        <div>
          <dt>Definitions</dt>
          <dd>{p.definitions}</dd>
        </div>
        <div>
          <dt>Depends on</dt>
          <dd>{p.efferent}</dd>
        </div>
        <div>
          <dt>Used by</dt>
          <dd>{p.afferent}</dd>
        </div>
        <div>
          <dt>Instability</dt>
          <dd>{p.instability.toFixed(2)}</dd>
        </div>
      </dl>
      {errors.length > 0 && <Alert tone="danger" title={`${errors.length} compile ${errors.length === 1 ? 'error' : 'errors'}`} body={errors.map((e) => e.message).join(' ')} />}
      {warnings.length > 0 && (
        <Alert tone="warning" title={`${warnings.length} lint ${warnings.length === 1 ? 'warning' : 'warnings'}`}>
          <a className="gl-button" data-size="sm" href={modelHref(pid, 'quality', path)}>
            Show them
          </a>
        </Alert>
      )}
      <section className="section">
        <h3 className="title">Depends on</h3>
        {out.length === 0 ? <p className="muted small">Nothing besides its parents.</p> : <ul className="dep-list">{out.map((e) => <DepLine key={e.to} pid={pid} e={e} show="to" />)}</ul>}
        <h3 className="title">Used by</h3>
        {into.length === 0 ? <p className="muted small">No other Package.</p> : <ul className="dep-list">{into.map((e) => <DepLine key={e.from} pid={pid} e={e} show="from" />)}</ul>}
      </section>
    </>
  );
}

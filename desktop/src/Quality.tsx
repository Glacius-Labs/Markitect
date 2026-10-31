import type { ReactNode } from 'react';
import type { Diagnostic } from './api';
import { byPackage, lintTitle, packageName, type Index } from './modeldata';
import { modelHref } from './route';
import { plural } from './status';
import { Badge } from './ui';

interface Props {
  ix: Index;
  pid: string;
  item?: string;
  diff: boolean;
}

// Quality shows how good the model is, per Package: what markitect lint warns
// about, the compile errors, the cycles, the coupling between Packages and
// the Definitions nothing uses. With the diff on, it shows which warnings are
// new and which are gone since the base.
export function Quality({ ix, pid, item, diff }: Props) {
  const { m, pkgs } = ix;
  const selected = item !== undefined && pkgs.has(item) ? item : undefined;
  const warnings = byPackage(m.lint);
  const errors = byPackage(m.errors);
  const unused = m.lint.filter((d) => d.code === 'lint.unused').length;
  const show = (ds: Diagnostic[]) => (selected === undefined ? ds : ds.filter((d) => d.package === selected));
  const added = m.diff.lint.filter((d) => d.change === 'added');
  const gone = m.diff.lint.filter((d) => d.change === 'removed');
  return (
    <>
      <div className="gl-stats">
        <Stat label="Lint warnings" value={m.lint.length} />
        <Stat label="Compile errors" value={m.errors.length} />
        <Stat label="Cycles" value={m.cycles.length} />
        <Stat label="Unused Definitions" value={unused} />
      </div>
      {diff && (
        <section className="section">
          <h2 className="h3">Since the base</h2>
          {added.length === 0 && gone.length === 0 ? (
            <p className="muted">No warning came or went.</p>
          ) : (
            <ul className="findings">
              {added.map((d, i) => (
                <Finding key={`a${i}`} pid={pid} d={d} badge={<Badge tone="warning">New</Badge>} />
              ))}
              {gone.map((d, i) => (
                <Finding key={`g${i}`} pid={pid} d={d} badge={<Badge tone="success">Gone</Badge>} />
              ))}
            </ul>
          )}
        </section>
      )}
      <section className="section">
        <h2 className="h3">Packages</h2>
        <div className="gl-table-wrap">
          <table className="gl-table" data-density="compact">
            <thead>
              <tr>
                <th>Package</th>
                <th data-numeric>Definitions</th>
                <th data-numeric title="Efferent coupling, Ce">Depends on</th>
                <th data-numeric title="Afferent coupling, Ca">Used by</th>
                <th data-numeric title="Ce / (Ca + Ce)">Instability</th>
                <th data-numeric>Warnings</th>
                <th data-numeric>Errors</th>
              </tr>
            </thead>
            <tbody>
              {ix.tree.map((p) => (
                <tr key={p.path} aria-selected={p.path === selected || undefined}>
                  <td>
                    <a href={p.path === selected ? modelHref(pid, 'quality') : modelHref(pid, 'quality', p.path)}>{packageName(p.path)}</a>
                  </td>
                  <td data-numeric>{p.definitions}</td>
                  <td data-numeric>{p.efferent}</td>
                  <td data-numeric>{p.afferent}</td>
                  <td data-numeric>{p.instability.toFixed(2)}</td>
                  <td data-numeric>{warnings.get(p.path)?.length ?? 0}</td>
                  <td data-numeric>{errors.get(p.path)?.length ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="small muted">Coupling counts the explicit edges, not the free link to a parent. A dependency should point towards the more stable Package, the one with the lower instability.</p>
      </section>
      {show(m.errors).length > 0 && (
        <section className="section">
          <h2 className="h3">Compile errors{selected !== undefined && ` in ${packageName(selected)}`}</h2>
          <ul className="findings">
            {show(m.errors).map((d, i) => (
              <Finding key={i} pid={pid} d={d} badge={<Badge tone="danger">{d.code}</Badge>} />
            ))}
          </ul>
        </section>
      )}
      <section className="section">
        <h2 className="h3">Lint warnings{selected !== undefined && ` in ${packageName(selected)}`}</h2>
        {show(m.lint).length === 0 ? (
          <p className="muted">No warning{selected !== undefined ? ' here' : ''}.</p>
        ) : (
          groups(show(m.lint)).map(([code, ds]) => (
            <div key={code} className="lint-group">
              <h3 className="title">
                {lintTitle[code] ?? code} <span className="muted small">{code}, {plural(ds.length, 'warning')}</span>
              </h3>
              <ul className="findings">
                {ds.map((d, i) => (
                  <Finding key={i} pid={pid} d={d} />
                ))}
              </ul>
            </div>
          ))
        )}
      </section>
      <p className="small muted">improve, which has each Manager look at its area and propose a better model, starts here once it is built.</p>
    </>
  );
}

function groups(ds: Diagnostic[]): [string, Diagnostic[]][] {
  const out = new Map<string, Diagnostic[]>();
  for (const d of ds) {
    out.set(d.code, [...(out.get(d.code) ?? []), d]);
  }
  return [...out.entries()].sort((a, b) => a[0].localeCompare(b[0]));
}

function Finding({ pid, d, badge }: { pid: string; d: Diagnostic; badge?: ReactNode }) {
  return (
    <li>
      {badge} {d.message}{' '}
      <span className="muted small">
        {d.definition ? <a href={modelHref(pid, 'reader', d.definition)}>{d.definition}</a> : <a href={modelHref(pid, 'reader', d.package)}>{packageName(d.package)}</a>}
        {d.file && (
          <>
            {' '}
            <span className="path">
              {d.file}
              {d.line ? `:${d.line}` : ''}
            </span>
          </>
        )}
      </span>
    </li>
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

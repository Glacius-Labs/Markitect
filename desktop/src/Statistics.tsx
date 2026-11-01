import { useEffect, useState } from 'react';
import { api, type Project, type StatPoint } from './api';
import { useRefreshOnFocus } from './focus';
import { hub } from './route';
import { Alert, Badge } from './ui';

// Statistics shows how the model develops: its size and what check and lint
// find in it, at each commit that changed Markitect's folder and in the
// working tree now. How Runs go and where the work struggles come with the
// Runs.

export function Statistics({ id }: { id: string }) {
  const [project, setProject] = useState<Project | null>(null);
  const [points, setPoints] = useState<StatPoint[] | null>(null);
  const [limit, setLimit] = useState(0);
  const [error, setError] = useState('');

  const load = () => {
    api.project(id).then(setProject, () => setProject(null));
    api.statistics(id).then(
      (s) => {
        setPoints(s.model);
        setLimit(s.limit);
        setError('');
      },
      (e: Error) => setError(e.message),
    );
  };
  useEffect(load, [id]);
  useRefreshOnFocus(load);

  const commits = points ? points.length - 1 : 0;
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
            <span aria-current="page">Statistics</span>
          </li>
        </ol>
      </nav>
      <header className="gl-page-header">
        <div className="gl-page-header__text">
          <h1 className="gl-page-header__title">Statistics</h1>
          <p className="gl-page-header__description">How the model develops, commit by commit. How Runs go comes with the Runs.</p>
        </div>
      </header>
      {error && <Alert tone="danger" title="The statistics cannot be read" body={error} />}
      {!points && !error && (
        <p className="muted" role="status">
          <span className="gl-spinner" aria-hidden="true" /> Reading the model at its commits; the first time takes a moment.
        </p>
      )}
      {points && (
        <>
          <p className="small muted">
            {commits === 0
              ? 'No commit changed the model yet; the numbers are the working tree’s.'
              : `The last ${commits === limit ? `${limit} commits` : commits === 1 ? 'commit' : `${commits} commits`} that changed .markitect/, and the working tree now. A larger dot is a commit a Run landed.`}
          </p>
          <div className="stats-charts">
            <Chart
              title="Size"
              points={points}
              series={[
                { key: 'definitions', label: 'Definitions' },
                { key: 'packages', label: 'Packages' },
              ]}
            />
            <Chart
              title="Findings"
              points={points}
              series={[
                { key: 'warnings', label: 'Lint warnings' },
                { key: 'errors', label: 'Compile errors' },
              ]}
            />
          </div>
          <section className="section">
            <h2 className="h3">Commits</h2>
            <div className="gl-table-wrap">
              <table className="gl-table" data-density="compact">
                <thead>
                  <tr>
                    <th>When</th>
                    <th>Commit</th>
                    <th data-numeric>Definitions</th>
                    <th data-numeric>Packages</th>
                    <th data-numeric>Warnings</th>
                    <th data-numeric>Errors</th>
                  </tr>
                </thead>
                <tbody>
                  {[...points].reverse().map((p) => (
                    <tr key={p.commit || 'now'}>
                      <td>{p.commit ? new Date(p.when).toLocaleDateString() : 'now'}</td>
                      <td>
                        {p.commit ? (
                          <>
                            <span className="path">{p.commit.slice(0, 8)}</span> {p.subject} {p.landed && <Badge tone="info">landed</Badge>}
                          </>
                        ) : (
                          <span className="muted">the working tree</span>
                        )}
                      </td>
                      <td data-numeric>{p.definitions}</td>
                      <td data-numeric>{p.packages}</td>
                      <td data-numeric>{p.warnings}</td>
                      <td data-numeric>{p.errors}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </>
      )}
    </>
  );
}

type Key = 'definitions' | 'packages' | 'warnings' | 'errors';

const width = 480;
const height = 180;
const pad = { left: 36, right: 12, top: 12, bottom: 24 };

// Chart draws a few series over the points, oldest left, the working tree
// right.
export function Chart({ title, points, series }: { title: string; points: StatPoint[]; series: { key: Key; label: string }[] }) {
  const max = Math.max(1, ...points.flatMap((p) => series.map((s) => p[s.key])));
  const x = (i: number) => pad.left + (points.length === 1 ? (width - pad.left - pad.right) / 2 : (i * (width - pad.left - pad.right)) / (points.length - 1));
  const y = (v: number) => pad.top + (height - pad.top - pad.bottom) * (1 - v / max);
  const ticks = [0, Math.round(max / 2), max];
  return (
    <figure className="gl-chart stats-chart">
      <figcaption className="title">{title}</figcaption>
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`${title}: ${series.map((s) => `${s.label} from ${points[0][s.key]} to ${points[points.length - 1][s.key]}`).join('; ')}`}>
        {ticks.map((t) => (
          <g key={t}>
            <line className="gl-chart__grid" x1={pad.left} x2={width - pad.right} y1={y(t)} y2={y(t)} />
            <text className="gl-chart__label" x={pad.left - 6} y={y(t)} textAnchor="end" dominantBaseline="central">
              {t}
            </text>
          </g>
        ))}
        <text className="gl-chart__label" x={pad.left} y={height - 6}>
          {points.length > 1 ? new Date(points[0].when).toLocaleDateString() : ''}
        </text>
        <text className="gl-chart__label" x={width - pad.right} y={height - 6} textAnchor="end">
          now
        </text>
        {series.map((s, n) => (
          <g key={s.key} data-series={n + 1}>
            <polyline data-series={n + 1} points={points.map((p, i) => `${x(i)},${y(p[s.key])}`).join(' ')} />
            {points.map((p, i) => (
              <circle key={i} data-series={n + 1} cx={x(i)} cy={y(p[s.key])} r={p.landed ? 4 : 2.5} className={p.landed ? undefined : 'stats-dot'}>
                <title>{`${p.commit ? `${p.commit.slice(0, 8)} ${p.subject}` : 'the working tree'}: ${p[s.key]} ${s.label.toLowerCase()}`}</title>
              </circle>
            ))}
          </g>
        ))}
      </svg>
      <ul className="gl-legend">
        {series.map((s, n) => (
          <li key={s.key} data-series={n + 1}>
            {s.label}: {points[points.length - 1][s.key]}
          </li>
        ))}
      </ul>
    </figure>
  );
}

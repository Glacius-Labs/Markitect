import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { api, type AppSettings, type ModelView, type Project, type RequiredCheck, type StatPoint } from './api';
import { layout } from './layout';
import { packageName } from './modeldata';
import { modelHref, settingsHref, statisticsHref } from './route';
import { plural, runsLine } from './status';
import { Badge, Icon } from './ui';

// What the tiles show beyond the hub's own answer. Each part comes on its own,
// and a tile without it still opens its area.
interface Live {
  model?: ModelView;
  points?: StatPoint[];
  settings?: AppSettings;
  checks?: RequiredCheck[];
}

function useLive(project: Project): Live {
  const [live, setLive] = useState<Live>({});
  // The hub reloads its project often, every few seconds while a chat draws the
  // model, and only a change in what it reports is worth asking again. Opening
  // the hub asks anew, which covers a change no number shows.
  const { id, status, waiting } = project;
  const c = status.check;
  const asked = [id, status.initialized, c?.notDrawn, c?.definitions, c?.packages, c?.files, c?.warnings, c?.errorCount, waiting?.uncommitted, waiting?.landed, waiting?.unlanded].join();
  useEffect(() => {
    let stopped = false;
    const put = (part: Live) => {
      if (!stopped) {
        setLive((l) => ({ ...l, ...part }));
      }
    };
    const none = () => {};
    api.settings().then((settings) => put({ settings }), none);
    if (status.initialized) {
      api.projectSettings(id).then((s) => put({ checks: s.checks }), none);
      api.statistics(id).then((s) => put({ points: s.model }), none);
    }
    if (c && !c.notDrawn) {
      api.model(id, undefined, '').then((r) => r && put({ model: r.view }), none);
    }
    return () => {
      stopped = true;
    };
  }, [asked]);
  return live;
}

// Tiles are the hub's way into the areas: large enough to fill a monitor, each
// showing what is in its area now.
export function Tiles({ project }: { project: Project }) {
  const live = useLive(project);
  const { id } = project;
  const check = project.status.check;
  const drawn = check && !check.notDrawn;
  // Only a model that compiles has its Runs measured.
  const runs = check?.ok ? runsLine(project.waiting) : '';
  return (
    <div className="tiles">
      <Tile
        n="1"
        icon="layers"
        title="Model"
        body="Read the model, see its Packages and Managers, and judge its quality."
        href={check ? modelHref(id, 'reader') : undefined}
        badge={!check ? <Badge tone="neutral">Set up Markitect first</Badge> : !drawn ? <Badge tone="neutral">No model yet</Badge> : <Badge tone={check.warnings > 0 ? 'warning' : 'neutral'}>{plural(check.warnings, 'warning')}</Badge>}
        facts={
          drawn && (
            <dl className="tile__numbers">
              <Number label="Definitions" value={check.definitions} />
              <Number label="Packages" value={check.packages} />
              <Number label="Files" value={check.files} />
            </dl>
          )
        }
      >
        {drawn && live.model && <MiniMap m={live.model} />}
      </Tile>
      <Tile n="2" icon="activity" title="Runs" body="Start apply and verify, and follow every agent while it works." badge={<Badge tone="neutral">Not built yet</Badge>} facts={runs && <p className="tile__fact">{runs}</p>} />
      <Tile n="3" icon="layout-dashboard" title="Statistics" body="How Runs go, where the work struggles, how the model develops." href={statisticsHref(id)} badge={<Badge tone="neutral">The model over time</Badge>}>
        {live.points && <Spark points={live.points} />}
      </Tile>
      <Tile n="4" icon="settings" title="Settings" body="Required checks, agents and logins." href={settingsHref(id)} badge={<Badge tone="neutral">Logins, agents and checks</Badge>}>
        <SettingsFacts settings={live.settings} checks={live.checks} />
      </Tile>
    </div>
  );
}

function Tile(props: { n: string; icon: string; title: string; body: string; href?: string; badge: ReactNode; facts?: ReactNode; children?: ReactNode }) {
  const inner = (
    <div className="tile__inner">
      <div className="tile__text">
        <span className="gl-card__icon">
          <Icon name={props.icon} />
        </span>
        <h3 className="gl-card__title">
          <span className="key">{props.n}</span>
          {props.title}
        </h3>
        <p className="gl-card__body">{props.body}</p>
        {props.facts}
        <div className="gl-card__footer">{props.badge}</div>
      </div>
      {props.children && <div className="tile__live">{props.children}</div>}
    </div>
  );
  return props.href ? (
    <a className="gl-card area area--live tile" href={props.href}>
      {inner}
    </a>
  ) : (
    <div className="gl-card area tile">{inner}</div>
  );
}

function Number({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

// MiniMap is the Map's drawing without its links, since the tile is one.
function MiniMap({ m }: { m: ModelView }) {
  const d = useMemo(() => layout(m.packages.map((p) => ({ id: p.path, label: packageName(p.path) })), m.edges), [m]);
  return (
    <svg className="tile__map" viewBox={`0 0 ${d.width} ${d.height}`} role="img" aria-label="The Package graph">
      <defs>
        <marker id="tile-arrow" className="map__arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0,0 L10,5 L0,10 z" />
        </marker>
      </defs>
      {d.edges.map((e) => (
        <path key={`${e.from}\u0000${e.to}`} d={e.d} className="map__edge" markerEnd="url(#tile-arrow)" />
      ))}
      {d.nodes.map((n) => (
        <g key={n.id} className="map__node">
          <rect x={n.x} y={n.y} width={n.w} height={n.h} rx="6" />
          <text x={n.x + n.w / 2} y={n.y + n.h / 2} dominantBaseline="central" textAnchor="middle">
            {n.label}
          </text>
        </g>
      ))}
    </svg>
  );
}

function Spark({ points }: { points: StatPoint[] }) {
  const set = points.filter((p) => p.initialized);
  const last = set.at(-1);
  if (!last) {
    return null;
  }
  const most = Math.max(...set.map((p) => p.definitions), 1);
  const line = set.map((p, i) => `${set.length === 1 ? 50 : (i / (set.length - 1)) * 100},${30 - (p.definitions / most) * 26}`).join(' ');
  const commits = set.filter((p) => p.commit).length;
  return (
    <figure className="tile__spark">
      {set.length > 1 && (
        <svg viewBox="0 0 100 32" preserveAspectRatio="none" role="img" aria-label="Definitions over the commits">
          <polyline points={line} />
        </svg>
      )}
      <figcaption className="small muted">
        {plural(last.definitions, 'Definition')} now, over {plural(commits, 'commit')} that changed the model.
      </figcaption>
    </figure>
  );
}

function SettingsFacts({ settings, checks }: { settings?: AppSettings; checks?: RequiredCheck[] }) {
  if (!settings) {
    return null;
  }
  const login = settings.logins.find((l) => l.used);
  return (
    <dl className="tile__facts">
      <div>
        <dt>Agents</dt>
        <dd>{settings.agents.model ? `${settings.agents.model}, effort ${settings.agents.effort}` : 'not set'}</dd>
      </div>
      {login && (
        <div>
          <dt>Login</dt>
          <dd>{login.state === 'ok' ? `${login.provider}, ready` : `${login.provider}, ${login.state}`}</dd>
        </div>
      )}
      {checks && (
        <div>
          <dt>Required checks</dt>
          <dd>{checks.length === 0 ? 'none yet' : checks.map((c) => c.name).join(', ')}</dd>
        </div>
      )}
    </dl>
  );
}

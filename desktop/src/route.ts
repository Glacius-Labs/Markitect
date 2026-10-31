// The address after # is the place in the app, so back and forward work:
//
//   #/                                  the projects
//   #/p/<project>                       a project's hub
//   #/p/<project>/model[/<view>[/<item>]]  the Model area
//   #/p/<project>/settings              Settings
//
// The item is a Definition ID or a Package path, Root written as '.'.

export const views = ['reader', 'map', 'organization', 'quality'] as const;
export type View = (typeof views)[number];

export interface Place {
  project?: string;
  area?: 'model' | 'settings';
  view?: View;
  item?: string; // a Definition ID or a Package path; '' is Root
}

export function parse(hash: string): Place {
  const settings = /^#\/p\/([0-9a-f]+)\/settings$/.exec(hash);
  if (settings) {
    return { project: settings[1], area: 'settings' };
  }
  const m = /^#\/p\/([0-9a-f]+)(?:\/(model)(?:\/([a-z]+)(?:\/(.+))?)?)?$/.exec(hash);
  if (!m) {
    return {};
  }
  const place: Place = { project: m[1] };
  if (m[2]) {
    place.area = 'model';
    place.view = 'reader';
    if (m[3] && (views as readonly string[]).includes(m[3])) {
      place.view = m[3] as View;
    }
    if (m[4] !== undefined) {
      // An address typed by hand may hold a broken escape; it names no item.
      try {
        const item = decodeURIComponent(m[4]);
        place.item = item === '.' ? '' : item;
      } catch {
        // no item
      }
    }
  }
  return place;
}

export function hub(project: string): string {
  return `#/p/${project}`;
}

export function settingsHref(project: string): string {
  return `#/p/${project}/settings`;
}

export function modelHref(project: string, view: View, item?: string): string {
  const base = `#/p/${project}/model/${view}`;
  if (item === undefined) {
    return base;
  }
  return `${base}/${item === '' ? '.' : item.split('/').map(encodeURIComponent).join('/')}`;
}

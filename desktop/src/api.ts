// The JSON routes of internal/adapters/desktop. The session cookie the token
// exchange set travels with every request.

export interface Check {
  ok: boolean;
  errorCount: number;
  errors: string[];
  definitions: number;
  packages: number;
  files: number;
  warnings: number;
}

export interface Status {
  initialized: boolean;
  check?: Check;
  problem?: string;
}

export interface Project {
  id: string;
  name: string;
  path: string;
  status: Status;
}

export interface AppInfo {
  version: string;
  // The folder that received the last new project, else the home folder.
  parent: string;
  // Whether the window has a folder dialog of its own: the native app has,
  // a browser window has not.
  folderDialog: boolean;
}

export interface Result {
  project: Project;
  ok: boolean;
  output: string;
}

export type AddRequest =
  | { action: 'open'; path: string }
  | { action: 'create'; parent: string; name: string }
  | { action: 'clone'; url: string; parent: string; name?: string };

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
  }
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  let data: unknown;
  try {
    data = text ? JSON.parse(text) : undefined;
  } catch {
    data = undefined;
  }
  if (!res.ok) {
    const message = (data as { error?: string } | undefined)?.error ?? text.trim();
    throw new ApiError(message || res.statusText, res.status);
  }
  return data as T;
}

// The Model area's document: the working tree's model with what refs, graph,
// impact and lint say about it, and the change from a base. Root's path is ''.

export type Change = 'added' | 'modified' | 'removed';
export type BaseKind = 'landing' | 'commit';

export interface Diagnostic {
  code: string;
  file?: string;
  line?: number;
  message: string;
  package: string;
  definition?: string;
}

export interface Ref {
  from: string;
  via: 'link' | 'kind' | 'registry';
  file: string;
  line?: number;
}

export interface Definition {
  id: string;
  title: string;
  kind: string;
  library?: string;
  package: string;
  file?: string;
  internal?: boolean;
  words: number;
  text: string;
  links: { raw: string; target: string; package?: boolean; kind?: boolean; line: number }[];
  referencedBy: Ref[];
  realizedBy: { file: string; reason: string; page: string; line: number }[];
  // The Packages whose Managers know it in full, and those that see it by
  // title in a child's table of contents.
  seenBy?: { full: string[]; contents: string[] };
  change?: Change;
}

export interface Package {
  path: string;
  parent: string;
  name: string;
  definitions: number;
  afferent: number;
  efferent: number;
  instability: number;
  children: string[];
  own: string[];
  patterns: string[];
  hasPage: boolean;
  files: number;
  change?: Change;
}

export interface Edge {
  from: string;
  to: string;
  uses: { from: string; target: string; kind?: boolean; file: string; line: number }[];
  cycle?: boolean;
}

export interface Item {
  definition: string;
  reasons: string[];
}

export interface Checklist {
  package: string;
  own: Item[] | null;
  inherited: Item[] | null;
  findings: unknown[] | null;
  stale: { code: string; package: string; pageFile: string; line?: number; file?: string; target?: string }[] | null;
}

export interface Impact {
  full: boolean;
  changedDefinitions: { path: string; change: Change }[] | null;
  changedFiles: { path: string; change: Change }[] | null;
  reached: Item[] | null;
  reachedFiles: { file: string; owner: string; change?: Change; for?: string[] }[] | null;
  packages: string[] | null;
  checklists: Checklist[] | null;
  notes?: string[];
}

export interface ModelDiff {
  definitions: { id: string; change: Change; title: string; package: string; baseText?: string }[];
  packages: { path: string; change: Change }[];
  edges: { from: string; to: string; change: Change }[];
  lint: (Diagnostic & { change: Change })[];
}

export interface ModelView {
  initialized: boolean;
  bases: { commit?: string; landing?: string };
  base: { kind: BaseKind; commit: string } | null;
  errors: Diagnostic[];
  lint: Diagnostic[];
  definitions: Definition[];
  packages: Package[];
  edges: Edge[];
  cycles: string[][];
  impact?: Impact;
  impactNote?: string;
  diff: ModelDiff;
}

const project = (id: string) => `/api/projects/${encodeURIComponent(id)}`;

// model asks for the Model area's document. With the ETag of the last answer
// it returns null when nothing changed since.
async function model(id: string, base: BaseKind | undefined, etag: string): Promise<{ view: ModelView; etag: string } | null> {
  const res = await fetch(`${project(id)}/model${base ? `?base=${base}` : ''}`, { headers: etag ? { 'If-None-Match': etag } : undefined });
  if (res.status === 304) {
    return null;
  }
  const text = await res.text();
  if (!res.ok) {
    let message = text.trim();
    try {
      message = (JSON.parse(text) as { error?: string }).error ?? message;
    } catch {
      // not JSON
    }
    throw new ApiError(message || res.statusText, res.status);
  }
  return { view: JSON.parse(text) as ModelView, etag: res.headers.get('ETag') ?? '' };
}

export const api = {
  app: () => call<AppInfo>('GET', '/api/app'),
  projects: () => call<{ projects: Project[] }>('GET', '/api/projects').then((r) => r.projects),
  project: (id: string) => call<{ project: Project }>('GET', project(id)).then((r) => r.project),
  add: (req: AddRequest) => call<Result>('POST', '/api/projects', req),
  init: (id: string) => call<Result>('POST', `${project(id)}/init`, {}),
  remove: (id: string) => call<void>('DELETE', project(id)),
  model,
  // The folder the user chose in the window's dialog, or '' when cancelled.
  chooseFolder: (start: string) => call<{ path: string }>('POST', '/api/folder', { start }).then((r) => r.path),
};

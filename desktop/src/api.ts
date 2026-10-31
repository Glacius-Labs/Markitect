// The JSON routes of internal/adapters/desktop. The session cookie the token
// exchange set travels with every request.

export interface Check {
  ok: boolean;
  errorCount: number;
  errors: string[];
  definitions: number;
  packages: number;
  files: number;
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

const project = (id: string) => `/api/projects/${encodeURIComponent(id)}`;

export const api = {
  app: () => call<AppInfo>('GET', '/api/app'),
  projects: () => call<{ projects: Project[] }>('GET', '/api/projects').then((r) => r.projects),
  project: (id: string) => call<{ project: Project }>('GET', project(id)).then((r) => r.project),
  add: (req: AddRequest) => call<Result>('POST', '/api/projects', req),
  init: (id: string) => call<Result>('POST', `${project(id)}/init`, {}),
  remove: (id: string) => call<void>('DELETE', project(id)),
  // The folder the user chose in the window's dialog, or '' when cancelled.
  chooseFolder: (start: string) => call<{ path: string }>('POST', '/api/folder', { start }).then((r) => r.path),
};

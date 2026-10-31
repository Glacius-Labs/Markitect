# Markitect Desktop

`internal/adapters/desktop` is Markitect Desktop, the app over the command line that users know as Markitect ([DEC-089](../../../docs/concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../../../docs/design/markitect-desktop.md)). This page is the canonical description of what is built: slice D1, projects and hub.

## Starting it

```text
markitect desktop [--listen 127.0.0.1:0] [--no-open]
```

- The first start runs the service and opens a window: Edge or Chrome in app mode, else the default browser ([`window.go`](window.go)). `--no-open` prints the address with its token instead.
- Every later start finds the running service through `desktop.json` in the user's config folder (`%AppData%\Markitect`, `~/Library/Application Support/Markitect`, `~/.config/Markitect`), opens another window on it and returns. A record whose service no longer answers is replaced.
- The service runs until the command ends (Ctrl+C). Closing a window stops nothing.

## Safety

- The service listens on a loopback address only, and answers only requests addressed to `127.0.0.1`, `localhost` or `[::1]` at its port, so a page of another site that resolves its own name to 127.0.0.1 gets nothing.
- Each start makes a random token. The window opens `/?t=TOKEN`, which sets it as an `HttpOnly`, `SameSite=Strict` cookie and redirects; every other request needs that cookie.
- A write must come from the service's own origin, and a `POST` must be JSON.
- Every response carries a content security policy that lets the page load only its own files; the page loads nothing from elsewhere.

## Routes

| Route | Does |
|---|---|
| `GET /api/ping` | Answers while the service runs; a second start uses it |
| `GET /api/app` | The version and the user's home folder |
| `GET /api/projects` | The project list, each with its status |
| `POST /api/projects` | Brings a project in: `open` a folder inside a repository, `create` a new folder with Git and `markitect init`, or `clone` a URL into a new folder |
| `GET /api/projects/{id}` | One project with its status |
| `POST /api/projects/{id}/init` | Runs `markitect init` in the project |
| `DELETE /api/projects/{id}` | Takes the project off the list; its folder stays |

A project's status is what `markitect check` finds in its working tree: whether `.markitect/` is set up, the compile errors, and the counts of Definitions, Packages and accounted files. The list lives in `projects.json` in the config folder.

## Ports

The adapter imports no other adapter. `internal/bootstrap` supplies Git (open, `git init`, `git clone`) through the repo adapter, and `markitect init` by running the command line's own `init`, so the app's init is the same command with the same output.

## The UI

The source is in [`desktop/`](../../../desktop/): TypeScript and React, built by Vite, with the [Glacius Labs design system](../../../desktop/vendor/glacius-design-system/README.md) vendored as released. `npm run build` type-checks it and writes the build into `ui/`, which is committed and embedded, so `go build` needs no Node. Never edit `ui/`; rebuild it. CI rebuilds it and fails when the committed build differs.

```text
cd desktop
npm ci
npm test
npm run build
```

For work on the UI, `npm run dev` serves it with hot reload and forwards `/api` to a running `markitect desktop --listen 127.0.0.1:7340` (another address through `MARKITECT_DESKTOP`); open the dev server's page through the service's `/?t=` link first, so the cookie is set.

## Tests

- `server_test.go`: the host check, the token exchange, the content security policy and that the built page loads nothing from elsewhere, the write rules, and opening, creating, cloning, initializing and removing projects against fake Git.
- `desktop_test.go`: a service serves the window it opened and removes its record; a second start opens a window on the running service; a stale record is replaced; only loopback addresses are served.
- `store_test.go` and `window_test.go`: the project list and the choice of window per platform.
- `internal/bootstrap/desktop_test.go`: creating, opening and cloning real repositories, and that `markitect check` passes after the app's init.
- `desktop/src/*.test.ts`: the hub's next step and the project badges.

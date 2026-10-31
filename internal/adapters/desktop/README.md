# Markitect Desktop

`internal/adapters/desktop` is Markitect Desktop, the app over the command line that users know as Markitect ([DEC-089](../../../docs/concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../../../docs/design/markitect-desktop.md)). This page is the canonical description of what is built: slice D1, projects and hub.

## Starting it

```text
markitect desktop [--listen 127.0.0.1:0] [--no-open]
```

- The first start runs the service and opens a window: Edge or Chrome in app mode, else the default browser ([`window.go`](window.go)). `--no-open` prints the window's link instead.
- Every later start finds the running service through `desktop.json` in the user's config folder (`%AppData%\Markitect`, `~/Library/Application Support/Markitect`, `~/.config/Markitect`), opens another window on it and returns. A record whose service does not prove itself is replaced. Publishing the record is exclusive, so two starts at once share one service.
- The service runs until the command ends (Ctrl+C). Closing a window stops nothing.

## Safety

- The service listens on a loopback address only, and answers only requests addressed to `127.0.0.1`, `localhost` or `[::1]` at its port, so a page of another site that resolves its own name to 127.0.0.1 gets nothing.
- Three secrets, each with one use:
  - **The launch code** in a window's link (`/?t=CODE`) works once and for two minutes. It may show up on the browser's command line, so it buys nothing more than one session.
  - **The session** is an `HttpOnly`, `SameSite=Strict` cookie, named after the service's port, that the launch code sets. Every request of the page needs it.
  - **The record's secret** stays in `desktop.json`, readable by the user only. A later start asks the service to prove it holds the secret (`/api/hello`, an HMAC over a fresh challenge), so a program that took over the port after a crash learns nothing; only then does the start send the secret to get a launch code for its window (`/api/launch`).
- A write that names an origin must name the service's own, and a `POST` must be JSON; a form or page of another site can send neither.
- Every response carries a content security policy that lets the page load only its own files; the page loads nothing from elsewhere.
- A new project's folder name must mean the same on Windows, Linux and macOS: no separator, reserved character, control character, leading dot or device name such as `CON`. A clone runs with no terminal prompt, SSH in batch mode unless the user configured an SSH command, the `ext` transport off and a 30-minute limit; a clone that fails leaves no folder behind.

## Routes

| Route | Does |
|---|---|
| `GET /api/hello?c=CHALLENGE` | Proves to a later start that this service holds the record's secret; needs no session |
| `POST /api/launch` | A launch code for another window, for a start that sends the record's secret; needs no session |
| `GET /api/app` | The version, and the folder to suggest for a new project: the last one used, else the home folder |
| `GET /api/projects` | The project list, each with its status |
| `POST /api/projects` | Brings a project in: `open` a folder inside a repository, `create` a new folder with Git and `markitect init`, or `clone` a URL into a new folder |
| `GET /api/projects/{id}` | One project with its status |
| `POST /api/projects/{id}/init` | Runs `markitect init` in the project |
| `DELETE /api/projects/{id}` | Takes the project off the list; its folder stays |

A project's status is what `markitect check` finds in its working tree: whether `.markitect/` is set up, the compile errors, and the counts of Definitions, Packages and accounted files; the list reads up to four projects at once. The list and the last parent folder live in `projects.json` in the config folder. On Windows and macOS, whose file systems ignore case, two spellings of one path are one project.

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

For work on the UI, start `markitect desktop --no-open --listen 127.0.0.1:7340` and open the link it prints once, which sets the session cookie for 127.0.0.1. Then `npm run dev` serves the UI with hot reload at `http://127.0.0.1:5173` and forwards `/api` to the service under the service's own host and origin (another service through `MARKITECT_DESKTOP`).

## Tests

- `server_test.go`: the host check; launch codes that work once and briefly; the session; the proof and the launch for a later start; the content security policy and that the built page loads nothing from elsewhere; the write rules; opening, creating, cloning, initializing and removing projects against fake Git, with the folder names refused; the suggested parent.
- `desktop_test.go`: a service serves the window it opened and removes its record, which never holds a launch code; a second start opens a window on the running service; a stale record is replaced; a program on the recorded port learns no secret; two starts at once share one service; only loopback addresses are served.
- `store_test.go` and `window_test.go`: the project list and the choice of window per platform.
- `internal/bootstrap/desktop_test.go`: creating, opening and cloning real repositories, and that `markitect check` passes after the app's init.
- `desktop/src/*.test.ts`: the hub's next step and the project badges.

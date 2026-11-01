# Markitect Desktop

`internal/adapters/desktop` is Markitect Desktop, the app over the command line that users know as Markitect ([DEC-089](../../../docs/concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli), [design](../../../docs/design/markitect-desktop.md)). This page is the canonical description of what is built: slice D1, projects and hub; slice D2, the native app ([`cmd/markitect-desktop`](../../../cmd/markitect-desktop/main.go)); slice D3, the [Model area](#the-model-area) with its four views, the diff and hand edits; of slice D4 [the chat](#the-chat) and [Settings](#settings); and the model's part of [Statistics](#statistics).

## Starting it

Markitect starts in one of two ways, over the same service:

- **The native app,** `markitect-desktop`, is what a user starts: no terminal, its own windows, a tray icon and the platform's folder dialog ([below](#the-native-app)).
- **The command line,** for work on the app and for a machine without the native app:

  ```text
  markitect desktop [--listen 127.0.0.1:0] [--no-open]
  ```

  Its windows are Edge or Chrome in app mode, else the default browser ([`window.go`](window.go)). `--no-open` prints the window's link instead. The service runs until the command ends (Ctrl+C).

Either way:

- There is one service per user. The first start runs it and opens a window. Every later start finds it through `desktop.json` in the user's config folder (`%AppData%\Markitect`, `~/Library/Application Support/Markitect`, `~/.config/Markitect`), asks it for another window and returns; with `--no-open` it prints a window's link instead. The service opens every window itself, so the windows are always the first start's kind: a later `markitect desktop` opens a native window when the native app runs. On Windows the later start lets the service bring that window to the front.
- A record whose service does not prove itself is replaced. Publishing the record is exclusive, so two starts at once share one service. The service answers from the moment its record exists, so a start that comes before its first window still finds it.
- Closing a window stops nothing.

## Safety

- The service listens on a loopback address only, and answers only requests addressed to `127.0.0.1`, `localhost` or `[::1]` at its port, so a page of another site that resolves its own name to 127.0.0.1 gets nothing.
- Three secrets, each with one use:
  - **The launch code** in a window's link (`/?t=CODE`) works once and for two minutes. It may show up on the browser's command line, so it buys nothing more than one session.
  - **The session** is an `HttpOnly`, `SameSite=Strict` cookie, named after the service's port, that the launch code sets. Every request of the page needs it.
  - **The record's secret** stays in `desktop.json`, readable by the user only. A later start asks the service to prove it holds the secret (`/api/hello`, an HMAC over a fresh challenge), so a program that took over the port after a crash learns nothing; only then does the start send the secret, to have the service open a window (`/api/window`) or to get a window's link (`/api/launch`).
- A write that names an origin must name the service's own, and a `POST` or `PUT` must be JSON; a form or page of another site can send neither.
- A hand edit writes only a Definition's file below `.markitect/model/`, named by the model language's names that also mean the same on every platform, and never through a symbolic link or a Windows junction, which could lead out of the repository. One edit checks and writes at a time, and a file is replaced in one step.
- Every response carries a content security policy that lets the page load only its own files; the page loads nothing from elsewhere.
- A new project's folder name must mean the same on Windows, Linux and macOS: no separator, reserved character, control character, leading dot or device name such as `CON`. A clone runs with no terminal prompt, SSH in batch mode unless the user configured an SSH command, the `ext` transport off and a 30-minute limit; a clone that fails leaves no folder behind.

## Routes

| Route | Does |
|---|---|
| `GET /api/hello?c=CHALLENGE` | Proves to a later start that this service holds the record's secret; needs no session |
| `POST /api/window` | Opens another window, for a start that sends the record's secret; needs no session. When no window opens, the error carries the window's link |
| `POST /api/launch` | A launch code for another window's link, for a start that sends the record's secret; needs no session |
| `GET /api/app` | The version, the folder to suggest for a new project (the last one used, else the home folder), and whether the window has a folder dialog |
| `POST /api/folder` | Asks the user for a folder in the window's own dialog, one dialog at a time, starting at a given full path; the answer is the folder, or empty when the user cancelled. A browser window has no dialog (501), and the user types the path |
| `GET /api/projects` | The project list, each with its status |
| `POST /api/projects` | Brings a project in: `open` a folder inside a repository, `create` a new folder with Git and `markitect init`, or `clone` a URL into a new folder |
| `GET /api/projects/{id}` | One project with its status |
| `POST /api/projects/{id}/init` | Runs `markitect init` in the project |
| `PUT /api/projects/{id}/definitions/{definition}` | Writes a Definition's text into the working tree, given the digest of the text the editor started from, or none for a new Definition; `409` with the file's text as it is now when it changed meanwhile ([hand edits](#hand-edits)) |
| `GET /api/projects/{id}/statistics` | The model's size and findings at each of its last 60 commits, oldest first, and in the working tree ([Statistics](#statistics)) |
| `GET /api/settings` | How this Markitect runs its agents and the state of each provider's account login ([Settings](#settings)) |
| `GET /api/projects/{id}/settings` | The project's required checks from `execution.yaml` |
| `GET /api/projects/{id}/chats` | The project's conversations, newest first, and whether this Markitect has chat agents |
| `POST /api/projects/{id}/chats` | Starts a conversation with its first message and its skill, `modeling` or `extract`; the turn runs in the background |
| `GET /api/projects/{id}/chats/{chat}` | One conversation: its messages, its proposals and whether a turn runs |
| `POST /api/projects/{id}/chats/{chat}/messages` | The owner's next message; `409` while a turn runs |
| `POST /api/projects/{id}/chats/{chat}/accept`, `/reject` | Accepts the open proposal into the working tree, or rejects it; an accept over a working tree that changed under the proposal writes nothing and answers `409` with the files |
| `DELETE /api/projects/{id}/chats/{chat}` | Deletes a conversation and ends its agent |
| `GET /api/projects/{id}/model?base=landing\|commit` | The Model area's document ([below](#the-model-area)); with the `ETag` of the last answer in `If-None-Match`, `304` when nothing changed. With `chat=ID`, the conversation's open proposal on top of the working tree, measured from the working tree |
| `DELETE /api/projects/{id}` | Takes the project off the list; its folder stays |

A project's status is what `markitect check` finds in its working tree: whether `.markitect/` is set up, the compile errors, and the counts of Definitions, Packages and accounted files; the list reads up to four projects at once. The list and the last parent folder live in `projects.json` in the config folder. On Windows and macOS, whose file systems ignore case, two spellings of one path are one project.

## Ports

The adapter imports no other adapter. `internal/bootstrap` supplies Git (open, `git init`, `git clone`, a commit and the last landing) through the repo adapter, with a stamp per file (size and modification time) that the Model area follows the working tree by, and `markitect init` by running the command line's own `init`, so the app's init is the same command with the same output. The window is the caller's: `Config.OpenWindow` opens one on a link, and `Config.ChooseFolder`, if set, asks for a folder. `Config.Chats`, if set, starts the chat agents; without it the chat says so. `Config.Settings`, if set, reads what Settings shows of the agents and logins. `Listen` claims the service and starts answering, `Serve` opens the first window and serves until its context ends, and `Window` opens another; `Start` does all three for the command line.

## The Model area

The hub's first area shows the project's model as the working tree has it, in four views over one document that `GET /api/projects/{id}/model` computes with the same functions as the model commands: the compile errors of `check`, `refs` for each Definition, the Package graph of `graph`, `impact` with every working Manager's checklist, and `lint`.

- **Reader:** the model as documentation, Package by Package. A Definition shows its Markdown, with its links leading on through the Reader and a link that resolves to nothing marked; its Kind and Package; the Definitions that link to it or are of it; the files whose registry entries realize it; and the Managers that see it, in full or by title in a child's table of contents, as a Manager's starting context holds it (concept 2.0 §5). A Package shows what it owns, its children and its Definitions. Raw HTML stays text, and links to other sites show their address but do not open, since a window of the app is no browser.
- **Map:** the Package graph, each Package above the Packages it depends on, with the links behind every edge, cycles and Packages with compile errors marked. Links up the tree are free and not drawn.
- **Organization:** the Manager tree as an organization chart: per Manager its Definitions, the paths it owns, and its checklist for the change, with the reasons and the files the change reaches.
- **Quality:** the lint warnings by code and Package, the compile errors, the cycles, the coupling table and the unused Definitions.

The digits 1 to 4 switch the views; Escape goes back to the hub.

**The change and the diff.** The change is measured from a base: the last landing, the commit `apply` starts from when no base is named, or, when no Run has landed or the owner asks for it, the last commit, so that what is not committed yet shows. With nothing committed, the whole model is the change, as with `--full`. The Organization's checklists are `impact` from that base. A switch turns the diff on in every view: the Reader marks new, changed and removed Definitions and shows a changed one's text line by line against the base, the Map marks new and gone edges and Packages, the Organization dims the Managers the change does not reach, and Quality lists the warnings that came and went.

**Following the folder.** A visible window asks for the document every two seconds. The service answers `304` unless a file Git sees, the last commit or the last landing changed, so an edit made in another editor shows within seconds without reading the model again in between. The service keeps the last document per project and base, and the projects at up to eight base commits in all, which are slow to read through Git and never change. A document read while its request ended is not kept, since a read cut short would stay wrong.

### Hand edits

Beside the chat, the owner changes the model by hand. A Definition's **Edit** opens its text, front matter included, in an editor that writes the file straight into the working tree: **Save** or Ctrl+S, and the views show it with what `check` and `lint` say, at once. **New Definition** on a Package's page writes a Definition with its Kind and title there and opens it. Deleting or moving a Definition stays with the owner's own editor or the chat.

- The editor names the digest of the text it started from, and a draft keeps it. When another editor saved the file meanwhile, nothing is written: the owner keeps their text, which then replaces the other, or takes the other and drops theirs.
- A file keeps its line ends; Markitect reads both as LF.
- A draft survives moving around the app in the same window, and is gone when the window closes.

## The chat

The chat is the main way to change the model ([design](../../../docs/design/markitect-desktop.md#the-chat)). A **Chat** button in the top bar opens it beside any area of a project, over the page in a narrow window. A conversation either changes the model, with the modeling skill, or extracts one from the code, with the Extract skill; it keeps its skill.

- **The agent** is the composition's (`Chats`), a [chat agent](../../agents/README.md#chat-agents) in its own container with the Manager image and the provider login that `apply` uses: one per conversation, started on a copy of the working tree as Git sees it, uncommitted changes included, when the conversation's first turn of this app's life begins. An agent that takes over a conversation, after a restart or after a turn that did not answer, gets the conversation so far and starts with the open proposal on top, so it keeps proposing it; a file of the proposal that the owner changed since comes in the owner's version, and the chat says so. The window asks for the conversation every second while a turn runs. One step at a time: a message, an accept, a reject or a delete during a turn is refused with `409`.
- **The proposal** is what the conversation changed below `.markitect/` since the agent's start, read after every turn; a later turn's proposal replaces the one before. Changes elsewhere are named and not proposed. **Show in the views** has the Model area show the working tree with the proposal on top, its diff measured from the working tree, with Accept and Reject at the top; editing waits until it shows the working tree again.
- **Only files below `.markitect/`** with plain names are proposed, whatever the agent reports; anything else is named and not written.
- **Accept** writes the proposal into the working tree, as one step against hand edits, unless a file it touches changed there since the agent's start; then nothing is written, the proposal names the files, and the next turn starts a fresh agent from the working tree as it is, with the rest of the proposal on top. A file that already holds what the proposal writes counts as written, so an accept cut off midway completes when repeated. After an accept the agent's start moves to the accepted state; after a reject the agent drops its changes. The owner commits as usual.
- **Conversations** live in `chats/<project>/` in the config folder, one JSON file each, and in memory while the app runs, so a window never reads a file a turn is writing; a conversation that cannot be saved stays in memory, and the panel says so. A closed proposal keeps its files but not their texts. Quitting stops the turns and ends every agent, each with up to ten seconds; an agent that finishes starting after that is ended at once.

## Statistics

The hub's third area, entered with the digit 3, shows how the model develops: its Definitions and Packages, and what `lint` and `check` find in it, at each of the last 60 commits that changed `.markitect/` and in the working tree now, as two small charts and a table. A commit a Run landed stands out. The service loads the project at each commit as `check` would, reading only `.markitect/` from Git, and keeps each commit's numbers, since a commit never changes. How Runs go and where the work struggles come with the Runs.

## Settings

The hub's fourth area, entered with the digit 4, shows two things and changes neither yet:

- **This project:** the required checks of `.markitect/execution.yaml`, or why they cannot be read. The form that edits them comes with the Runs.
- **This Markitect:** the provider, model, effort, Manager image, provider network and Docker command that chats and Runs start their agents with, read from the environment Markitect was started in (`MARKITECT_*`, as `markitect apply` reads them) without asking the container engine, and why no agent can start yet as far as that shows without the engine: a missing Manager image or Docker command, the engine's own network, or an unusable login of the provider in use. Each provider's account login shows whether it is usable and until when, as an agent would admit it; of a login's content, only its expiry leaves the composition. It names the provider's own command to log in again.

## The native app

[`cmd/markitect-desktop`](../../../cmd/markitect-desktop/main.go) is the same service in a native shell, [Wails v3](https://wails.io/) inside the Go process, chosen by the [D2 probe](../../../docs/design/markitect-desktop.md#the-native-shell). It shows the service's pages in the platform's own web view (WebView2 on Windows, WKWebView on macOS, WebKitGTK on Linux), loaded from the loopback address with a launch code like a browser window, so the safety above holds unchanged. The page talks only to the service: the shell binds no Go function to it, and the web view's own message channel moves and resizes the window and nothing else.

- **Windows:** titled Markitect, with the app's icon, the design system's dark background while the page loads, at least 390 pixels wide.
- **Tray:** a click brings a window to the front, or opens one when none is open; its menu opens Markitect, opens a new window or quits Markitect. On macOS the menu bar icon is a template, and a click on the Dock icon with no window open opens one.
- **Closing the last window** keeps Markitect running in the tray; quitting from the tray, or from the app menu on macOS, stops the service and removes its record. On a desktop without a tray, such as GNOME without an extension, starting Markitect again opens a window.
- **Folder dialog:** the platform's own, over the window in front, for the folder fields of the project dialogs.
- **Errors** at start or while serving show in a dialog, since the app has no terminal. A failure that leaves no window, such as a missing web view, is appended to `desktop-error.log` in the config folder.
- **Quitting** cancels the service and removes its record without waiting for requests in flight, which may wait for the window loop that quits.
- **Data:** WebView2 keeps its data in `WebView` in the config folder above.

It builds only with the `native` tag, so that `go build ./...`, `go vet ./...` and `go test ./...` stay free of C toolchains and GTK. On Windows it needs no C compiler; it is linked as a GUI program, so it opens no terminal. macOS needs the Xcode command line tools, and Linux needs cgo with GTK 4 and WebKitGTK 6 (`libgtk-4-dev libwebkitgtk-6.0-dev` on Ubuntu 24.04). WebKitGTK sandboxes its web process with bubblewrap, which needs unprivileged user namespaces; Ubuntu desktops allow them, GitHub's Ubuntu runners do not, so CI allows them before it starts the app.

```text
go build -tags native -ldflags=-H=windowsgui ./cmd/markitect-desktop   # Windows
go build -tags native ./cmd/markitect-desktop                          # macOS and Linux
```

The icons in `cmd/markitect-desktop/` are rendered from the sources in [`desktop/icons/`](../../../desktop/icons/): the Markitect mark on the design system's dark background for windows, taskbar, Dock and tray, and the mark alone as the macOS menu bar template. A Windows executable icon, a macOS `.app` bundle and a Linux `.desktop` entry come with packaging.

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

- `server_test.go`: the host check; launch codes that work once and briefly; the session; the proof, the window and the launch for a later start; the folder dialog, one at a time and only where the window has one; the content security policy and that the built page loads nothing from elsewhere; the write rules; opening, creating, cloning, initializing and removing projects against fake Git, with the folder names refused; the suggested parent.
- `chat_test.go`: conversations over scripted agents: a proposal shown on top of the working tree and accepted into it exactly, outside changes left out; an accept over a changed working tree that writes nothing, then a reject; a later turn's proposal replacing the earlier; one message at a time; a conversation reopened after a restart starting a fresh agent with the conversation so far; an agent that cannot start, and an app without agents.
- `statistics_test.go`: the numbers at each commit and in the working tree, the landed commit marked, a commit's numbers read once, and a project without commits.
- `settings_test.go`: Settings show the agents and logins the composition reads, and the project's required checks, a broken `execution.yaml` and a project without Markitect.
- `edit_test.go`: a hand edit writes the file, and the views show it; a new Definition is created only where none is; an edit of a file that changed or was removed meanwhile writes nothing and returns the file as it is; a file keeps its CRLF line ends; only Definition IDs of the model language are written, and only as JSON.
- `model_test.go`: the Model area's document against a project in memory: Definitions with their references, realizing files and who sees them; Packages, edges and cycles; impact from the last commit or the last landing, and the whole model with nothing committed; the diff with the warnings that came and went; compile errors placed by Package and Definition; `304` until a file or a commit changes.
- `desktop_test.go`: a service serves the window it opened and removes its record, which never holds a launch code; a second start has the running service open the window, or prints a link with `--no-open`; a window that fails hands its link to the start that asked; a later start names a service of an earlier version and stops waiting for one that does not answer; a service answers from `Listen` on and leaves no record when closed; a stale record is replaced; a program on the recorded port learns no secret; two starts at once share one service; only loopback addresses are served.
- `store_test.go` and `window_test.go`: the project list and the choice of window per platform.
- `internal/bootstrap/settings_test.go`: the agents' settings from the environment, a usable login with its expiry, a missing one and a refused API key, with no token in what Settings shows.
- `internal/bootstrap/desktop_test.go`: creating, opening and cloning real repositories; reading the last commit, the last landing and the file stamps; writing into the working tree but not through a symbolic link; that `markitect check` passes after the app's init; and that `ListenDesktop`, which the native app calls, shares one service per user.
- The native app has no unit tests; CI builds it on request on Windows, macOS and Linux and starts it ([CONTRIBUTING](../../../CONTRIBUTING.md#verify-a-change)).
- `desktop/src/*.test.ts`: the hub's next step and the project badges; the Reader's Markdown with its links; the line diff; the Map's layout; the places in the app.

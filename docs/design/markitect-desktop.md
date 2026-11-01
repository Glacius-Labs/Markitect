# Markitect Desktop

Status: design of 31 October 2026, following the owner's decision [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli). The owner answered each area's main question the same day: the ways into a project, Extract's stops, how modeling works, the four views, the Run view and its start, Statistics, Settings, the chat and the build order. The details around those answers, such as layout, conflict handling and option placement, are this session's proposal. It replaces the postponed [model explorer](https://github.com/Glacius-Labs/Markitect/blob/0f8cbb1baca0174dcf0472cf964b91f835d2cc4e/docs/design/model-explorer.md), whose views the Model area takes up. Open questions and the current slice are on the [Desktop state page](../workstreams/desktop.md).

## Name

Inside the project the app is called Markitect Desktop. Users see only Markitect: the window title, the app icon, the installer and the website say Markitect, the way Claude and ChatGPT name their desktop apps. The CLI keeps its name `markitect`, and `markitect desktop` starts the app from a terminal.

## Goal

One place to work on a Markitect project: choose or create it, model it, run commands, follow every agent while it works and see how the project is doing. The app wraps the CLI: every Command and model tool it offers is the CLI's, with the same result. It adds views, live state and a chat. The chat is the app's own, and its model changes reach the owner as proposals.

## What the person sees

The structure the owner asked for on 21 October 2026 still holds: a hub, a few areas, cards, one next step instead of a dashboard. The digits 1 to 4 enter the areas, and Escape goes up one level.

### Projects

The start screen.

- A per-user list of projects.
- Three ways in:
  - opening a folder that is a Git repository, with or without `.markitect/`;
  - creating a project: a new folder with Git and `markitect init`;
  - cloning one from a Git URL.

  Creating a repository on GitHub comes later. Modeling needs no remote, but landing does: without one, a Run keeps its work on its Run branch, because the host never pushes to a checked-out branch ([host Git](../../internal/run/README.md#host-git)). How a project created in the app gets a remote is a question for the Runs phase.
- `init` on an opened repository that has no `.markitect/` yet.
- Extract, which draws a model from existing code through the [chat](#chat) with the Extract skill. It asks in the chat where the code and its documentation leave something open, and it stops at two checkpoints:
  1. the Package cut, shown on the map with the folders each Package claims, which the owner confirms or changes before any Definition is written, because the cut is the project's architecture;
  2. the whole model as one proposal, reviewed in the Model views before it is accepted.

  The Extract skill gains the first stop, which today it makes only when the code allows two cuts.

Choosing a project opens its hub.

### Hub

The project's state: whether the model passes `check`, the last Result, any Run working now, and the one next step for the owner, such as a proposal waiting for acceptance or a Run that stopped on a model problem. There is one tile per area with a one-line summary.

### 1. Model

- **Modeling.** The chat is the main way to change the model. A hand edit is the second: each Definition in the reader opens in a Markdown editor that saves straight into the working tree, and `check` runs again on save. The views follow the folder live, so edits made in Obsidian or an editor show up too.
- **The model as it looks.** While the owner models, the views show the model as it is, or as it will be with the chat's proposal, never mainly as a change. The diff is one click away. Bringing the code in line with the model is the job of `apply`, so after an accepted change the hub's next step offers it.
- **Four views.**
  1. Reader: the model as documentation. Each Definition with its links, Referenced by, Realized by and Seen by. Hand edits start here.
  2. Map: the Package graph, with cycles and broken rules marked.
  3. Organization: the Manager tree as an organization chart, with what each Manager is responsible for and its checklist for the current change (`impact`).
  4. Quality: how good the model is. The `lint` signals, cycles, coupling between Packages and Definitions nothing uses, per Package. `improve` starts here once it is built ([concept 2.0 §7](../concepts/v2-concept.md#7-commands-that-start-agents)): each Manager looks at its area, and the Run ends with a model proposal, shown like a chat's proposal.
- **Diff.** A toggle in every view, not a view of its own.

### 2. Runs

- Starting `apply` and `verify`, later the other commands of §7, and stopping a Run. A start shows a preview first: the base it starts from, which Managers will work, shown on the Organization tree, and each one's checklist (`impact`), with the usual limits. Then Start runs it. `--full`, a Findings file and the limits sit under "More options".
- The project's Runs with their Results.
- A Run's live view opens on the Manager tree:
  - one card per Manager with its maker and checker and their state: working, waiting and for whom, approved, rejected or escalated;
  - the current activity of each in one line;
  - a click on a card opens that agent's live transcript: its messages, tool calls, pushes and verdicts;
  - a timeline under the tree shows the Run so far.

  A closed Run plays back the same way.
- Messages and work on the tree. Messages travel only along the tree: `ask` and `escalate` up to the parent, `answer` and `assign` down to a child. Work moves as commits: a child's approved work up when its parent merges it, a parent's approved commit down when its children start from it. Both are drawn on the tree's edges, in two styles, animated as they happen and in playback.
  - An open question stays as a marker on the asker's card and its edge until it is answered; a click opens the thread.
  - Shared Notes, which go to no one in particular, sit in a notes panel beside the tree.
  - Root's escalation is a model defect: the Run ends without landing, and the hub shows the owner's decision with its recommendation.

### 3. Statistics

Three parts, each over time and per command:

1. Runs: landed or not and why, duration, agent starts and tokens.
2. Where the work struggles: rejections, escalations and attempts per Manager. They point at Packages whose model may be unclear.
3. The model: Definitions, Packages and `lint` warnings, so its quality shows as a trend.

Usage is counted in tokens, not money, since providers run on account login. The Run numbers come from the Results, the model numbers from the model at each landed commit.

### 4. Settings

Today the host takes its agent settings from environment variables; the app sets them in two levels.

- **Project settings:**
  - the required checks of `execution.yaml` as a form, which writes into the repository;
  - the Manager image with the project's toolchain, with a button that builds it;
  - the provider, model and effort, when the project differs from the app's default;
  - the default limits of a Run.
- **App settings:**
  - the Claude and Codex logins, with their state and a button that runs the provider's own login;
  - Docker's state;
  - the default provider, model and effort;
  - notifications.

  The app has no light theme: the design system uses dark themes only on screen.

Everything except `execution.yaml` stays in the user's config folder, outside the repository. So that the app and the CLI start the same agents, the CLI reads the same settings, with its environment variables still overriding them; that comes with the Runs phase, since the agent settings serve the Runs.

### Chat

Chats with an agent on the open project, as in Claude Code Desktop or the ChatGPT app. A chat explains the model, proposes model changes through the modeling skill and runs Extract.

- **Several conversations per project,** each kept with its history, its proposals and whether they were accepted.
- **A panel beside any area,** so the model stays in view while the owner talks. The Reader follows the Definitions the chat is discussing.
- **Proposals.** A model change reaches the owner as the proposed model in the views, with its diff one click away, to accept or reject as [concept 2.0 §8](../concepts/v2-concept.md#8-authoring) requires.
- **Commands.** A chat can offer one, such as `apply` after an accepted change. The offer opens the same preview as the Runs area, and the owner starts it there.

## Architecture

```text
markitect desktop  (one Go process per user)
├─ loopback server on 127.0.0.1:<port>
│   ├─ the UI, embedded in the binary
│   ├─ JSON routes: projects, model views, settings, commands
│   └─ one event stream (server-sent events): Runs, agents, chat
├─ the same functions as the CLI: project, analysis, run, agents
└─ the window: the native shell (markitect-desktop), or Edge or Chrome in app mode (markitect desktop)
```

- **One service per user.** The first start runs the service and opens a window, and every later start opens another window on it. Closing the window stops nothing: Runs and chats go on, and a reopened window reconnects to the stream. A lock file in the user's config folder names the port.
- **Same functions as the CLI.** The desktop is an adapter beside `internal/adapters/cli`. Like the CLI, it reaches the services through `internal/bootstrap`, and it parses no command output.
- **Live events.** Today the engine writes the Result when a Run ends. The agent adapters read each agent's output stream (Claude Code's stream-json, Codex's events) and can write it as transcripts ([Manager agents](../../internal/agents/README.md)). The live view needs both as events while the Run works. That means a port in `internal/run` through which the engine reports its schedule, starts, pushes, verdicts and messages, plus each agent's output lines. The CLI can print the same events. This is `internal/run`'s scope, so it is built together with the [Build and evaluation session](../workstreams/build.md), once the way Runs execute is settled ([Delivery](#delivery)).
- **Chat agents.** A chat is an agent in a container, but no Run: it depends on the container client and the provider's CLI, not on the Run engine. It runs with the provider's account login, on a copy of the project as it is in the working tree, uncommitted changes included. No folder of the user's is mounted ([concept 2.0 §9](../concepts/v2-concept.md#9-platform)). Its changes are a proposal, which the views show as the proposed model, with its diff one click away, while the conversation goes on. When the owner accepts, the app writes the proposal into the working tree as a patch, and the owner commits it as usual. If a file the patch touches changed since the copy, the app writes nothing and shows the conflict.
- **Safety.** The server listens on loopback only and refuses requests addressed to any other host (against DNS rebinding). Every route requires the launch token, and every response carries a strict content security policy. The page loads no external resource.
- **Window.**
  - First, Edge or Chrome in app mode, using the launcher of the retired cockpit ([`window.go`](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/internal/adapters/cockpit/window.go) at the archive point). `markitect desktop` keeps it, for work on the app and for a machine without the native app.
  - Then a native shell, so that Markitect starts without a terminal and has its own title bar, taskbar icon, tray and notifications: Wails v3, chosen by the probe of slice D2 ([below](#the-native-shell)).
- **No terminal window.** On Windows the app's entry point is built as a GUI program (`-H windowsgui`) while `markitect` stays a console program, so `cmd/markitect-desktop` sits beside `cmd/markitect`. macOS gets an `.app` bundle and Linux a `.desktop` entry with packaging.
- **UI.** TypeScript with React, built by Vite, with its source in `desktop/`. The built files are committed under `internal/adapters/desktop/ui/` and embedded, so `go build` needs no Node. CI rebuilds them and fails on a difference, as it does for the code map. The source packager learns the UI folder.
- **Look.** The Glacius Labs design system, the private package `@glacius-labs/design-system`, which Markitect ships under the design system's own notice ([DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli)): its CSS, tokens, fonts and icons, with React components that follow its `spec.json`.

| Package | Role |
|---|---|
| `internal/adapters/desktop` (new) | The loopback server, the routes, the event stream, the embedded UI and the window launcher |
| `internal/bootstrap` | Builds the desktop over the same services as the CLI |
| `internal/adapters/cli` | Gains the `desktop` verb |
| `internal/tooling/architecture`, `internal/tooling/release`, CI | The new packages' rows and the code map, the packager's UI folder, the UI build check |
| `cmd/markitect-desktop` (new) | The native app: the service in Wails windows, with tray and folder dialog |
| `internal/run` | Later gains the event feed for the live view, built with Build once Run execution is settled |
| `desktop/` (new, not Go) | The UI source |

### The native shell

The D2 probe on 31 October 2026 chose **Wails v3**, pinned at v3.0.0.

- **One process, one language.** Wails runs the platform's own web view inside the Go process, so the app stays one Go program as DEC-089 says. Tauri 2 would put the window in a Rust process with the Go service as a second, sidecar process, and the build would need Rust and Node beside Go. Tauri was not built; its cost is in its structure, not in what a probe measures.
- **Stable now.** The design named Wails' beta as its risk; Wails v3.0.0 was released stable on 26 October 2026. It is new, so the version is pinned and the shell stays thin: windows, tray and folder dialog, about 200 lines, cheap to replace.
- **The probe on Windows** passed with a small program before the shell was built: a native window on the loopback service with its session cookie and the content security policy in force, a second window opened from an HTTP request, the platform's folder dialog asked for over HTTP, and all windows closed with the app running on in the tray. No C compiler is needed on Windows; the app is about 22 MB.
- **Linux and macOS** need cgo, with GTK 4 and WebKitGTK 6 on Linux. The app therefore builds only with the `native` tag, so that the rest of the repository keeps building, vetting and testing without them. CI type-checks the shell on every run by building it for Windows, and builds and starts it on all three platforms when asked ([Markitect Desktop](../../internal/adapters/desktop/README.md#the-native-app)).
- **The same safety.** A native window loads the service's page from the loopback address with a launch code, exactly like a browser window, and the shell binds no Go function to the page. The folder dialog is a route of the service that the native app answers.
- **The service opens every window.** A later start asks the running service for a window instead of opening one itself, so the native app's windows stay native, whichever way Markitect is started again.

### The Model area

Details settled while building D3 on 31 October 2026, under the owner's go-ahead to decide them; [Markitect Desktop](../../internal/adapters/desktop/README.md#the-model-area) describes what is built.

- **One document, four views.** One route computes the model with `check`, `refs`, `graph`, `impact` and `lint` and serves it to all four views, so the views never disagree about the model.
- **The base of the change** is the last landing, which is the base `apply` takes, so the Organization's checklists are what a Run would start from. Where no Run has landed, or when the owner switches to it, the base is the last commit, which shows what is not committed yet. With nothing committed, the whole model is the change.
- **Following the folder** is a poll every two seconds that the service answers with "nothing changed" unless a file, the last commit or the last landing did. The event stream comes with the Runs phase. While nothing changes, a poll costs one `git ls-files`, a stat of each file Git sees and one `git rev-parse`; on a very large repository a watcher may replace it.
- **Seen by** follows a Manager's starting context as the harness renders it: its own Package, its parents' Definitions, the Definitions they link to and their Kinds in full, and a child's public Definitions by title.
- **Markdown** is rendered with markdown-it with raw HTML off. A link of the model leads on in the Reader; links to other sites show their address and do not open, since the native window is no browser and binds nothing.
- **The Map's layout** is the app's own small layered drawing, a Package above what it depends on, instead of a graph library.
- **Hand edits** use a plain text editor over the whole file, front matter included, rather than a rich Markdown editor: the file is the model, and the Reader shows how it reads. A save names the text it started from, so a file another editor changed is never overwritten unseen. New Definitions start from a Package's page; deleting and moving stay with the owner's editor or the chat.

### The chat

Details settled on 31 October 2026, under the owner's go-ahead to decide them, before D4 is built. While the evaluations keep Docker, the chat is built and tested against the agent runtime's scripted stand-in and runs for real once Docker is free.

- **One agent per conversation, beside the Managers.** A chat agent is a new kind of agent in `internal/agents`, added beside the Manager agent without changing it: the same Manager image, limits, provider network, account login copy and provider adapters, but no Run. Its `/work` is filled from the project's working tree as Git sees it, uncommitted changes included, sent into the container as an archive; no folder of the user's is mounted. Inside, that state is committed as the conversation's start.
- **What it knows.** The chat's system text says what it is for and carries the modeling skill, or the Extract skill for an extraction. It names the one rule the app enforces: only files below `.markitect/` are proposed. The agent reads the model in `/work` with the `markitect` commands of the image. It gets no Manager tools.
- **Turns.** Each message of the owner is one turn of the same provider conversation. The container lives while the app runs and the conversation is open. A conversation reopened after the app restarted starts a fresh agent, whose first message carries the conversation so far.
- **The proposal** is what the turn changed below `.markitect/` against the conversation's start, read from the container after every turn. Changes elsewhere are named but not proposed. The Model views show the working tree with the proposal on top, and the diff against the working tree is one click away.
- **Accept and reject.** Accepting writes the proposal into the working tree, but only if no file it touches changed there since the conversation's start; otherwise nothing is written and the app names the files. After an accept, the container's start moves to the accepted state; after a reject, the container drops the proposal. The owner commits as usual.
- **Conversations** are kept per project in the user's config folder: messages, each proposal and whether it was accepted. The panel lists them, newest first, beside any area.
- **While a turn runs,** the window asks for the conversation every second and shows the agent's messages as they arrive, a message at a time; the event stream comes with the Runs.
- **Extract** is a conversation that starts with the Extract skill and its task. Its first stop, the Package cut, is a proposal of the registry pages with their path patterns and one anchor Definition per Package, shown on the Map before any other Definition is written. Its second is the whole model as one proposal. The skill gains the first stop as a change of its own, made one writer at a time with Build.
- **Logins** live in Settings, at app level: whether the Claude and Codex logins exist, whose account they are and when they expire, read from the provider's own login file, and how to log in with the provider's own CLI.

## Delivery

Each slice is one PR with its tests and is useful on its own. The app is built against what is stable first. How Runs execute may still change fundamentally with the evaluations that are running: Manager containers that live for the whole Run or for one start, Git handoffs or one shared checkout, or no containers at all. So the Runs area waits until that is settled, and its sections above, which describe today's engine, are revised then.

**The model first,** with the native shell early, as DEC-089 says, so that the first version used day to day already behaves like a desktop app:

1. **D1, Projects and hub.** `markitect desktop`, the service, the browser window, the project list, opening, creating and cloning projects, `init`, and the hub with the model check. Built.
2. **D2, Native shell.** The probe and the shell, no terminal window, the app icon and the title Markitect. Built: Wails v3 ([above](#the-native-shell)), with a tray and the folder dialog.
3. **D3, Model.** Reader, Map, Organization and Quality, the diff toggle and hand edits. Built ([below](#the-model-area)).
4. **D4, Chat and Extract.** Chat agents in containers, proposals in the views, Extract with its two stops, and the app settings for the logins ([above](#the-chat)). In three parts: the conversations, proposals and panel in the app over a port; the chat agent in its container; Extract and the logins.

**Surroundings, when the model work is blocked:** the model part of Statistics and a read-only Settings, both built; the rest of the app settings, notifications and packaging.

**Runs, once their execution is settled:** starting Runs with their preview and Results, with the project settings for agents; the live view with its event feed from the engine, built with Build; the Run parts of Statistics.

`improve` and the other commands join as they are built; installers and updates come with the first release.

## Tests

- **Go:** `httptest` for every route, the host check, the token, and that no external resource is loaded. One end-to-end test of `markitect desktop --no-open` against a fixture project in `testdata/`.
- **UI:** component tests with Vitest, and Playwright against the fixture project.
- **Screenshots:** each slice's PR carries them at 1440 and 390 pixels wide.

## Not in this design

A personal agent beyond the project chat ([IDEA-015](../concepts/register.md#idea-015-cockpit-with-a-conversational-personal-agent)), the loop of continuous change ([concept 2.0 §10](../concepts/v2-concept.md#10-continuous-change)), and Government.

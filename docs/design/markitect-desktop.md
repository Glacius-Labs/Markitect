# Markitect Desktop

Status: design proposal of 31 October 2026, following the owner's decision [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli). Nothing is built. It replaces the postponed [model explorer](https://github.com/Glacius-Labs/Markitect/blob/0f8cbb1baca0174dcf0472cf964b91f835d2cc4e/docs/design/model-explorer.md), whose four model views the Model area takes up. Open questions and the current slice are on the [Desktop state page](../workstreams/desktop.md).

## Name

Inside the project the app is called Markitect Desktop. Users see only Markitect: the window title, the app icon, the installer and the website say Markitect, the way Claude and ChatGPT name their desktop apps. The CLI keeps its name `markitect`, and `markitect desktop` starts the app from a terminal.

## Goal

One place to work on a Markitect project: choose or create it, model it, run commands, follow every agent while it works and see how the project is doing. The app wraps the CLI. Every action it offers is a CLI command or model tool with the same result, so nothing in the app is out of the CLI's reach. The app adds views, live state and a chat.

## What the person sees

The structure the owner asked for on 21 October 2026 still holds: a hub, a few areas, cards, one next step instead of a dashboard. The digits 1 to 4 enter the areas, and Escape goes up one level.

### Projects

The start screen.

- A per-user list of projects.
- Opening a folder, with or without `.markitect/`.
- Creating a project: a new folder with Git and `markitect init`.
- `init` on an existing repository.
- Extract, which draws a model from existing code through the [chat](#chat) with the Extract skill.

Choosing a project opens its hub.

### Hub

The project's state: whether the model passes `check`, the last Result, any Run working now, and the one next step for the owner, such as a proposal waiting for acceptance or a Run that stopped on a model problem. There is one tile per area with a one-line summary.

### 1. Model

- **Views.**
  - Read: each Definition with its links, Referenced by, Realized by and Seen by.
  - Map: the Package graph with cycles and layers.
  - Organization: the Manager tree, with each Manager's checklist for a change.
  - Changes: what a change touches between two model states (`impact`).
- **Quality.** The `lint` signals and metrics per Package, shown on the map and in the reader.
- **Improve.** Starts `improve` once it is built ([concept 2.0 §7](../concepts/v2-concept.md#7-commands-that-start-agents)). Each Manager looks at its area, and the Run ends with a model proposal that the owner reads as a diff and accepts or rejects.

### 2. Runs

- Starting `apply` and `verify` with their options, later the other commands of §7, and stopping a Run.
- The project's Runs with their Results.
- A Run's live view:
  - the Manager tree, who works now and who waits for whom;
  - each maker's and checker's activity as it happens: turns, tool calls, pushes, verdicts and messages;
  - the timeline.

  A closed Run plays back the same way.

### 3. Statistics

Over the project's Runs: duration, agent starts and tokens, landed and not landed, rejections and escalations, per command, per Manager and over time. The numbers come from the Results.

### 4. Settings

- The project's `execution.yaml` with its required checks.
- The providers and their login state.
- Docker and the Manager image.
- Default limits.

App settings are kept apart from project settings.

### Chat

A chat with an agent on the open project, reachable from every area, as in Claude Desktop or the ChatGPT app. It explains the model, proposes model changes through the modeling skill and runs Extract. A model change it proposes reaches the owner as a diff to accept, as [concept 2.0 §8](../concepts/v2-concept.md#8-authoring) requires. It can offer to start a command, and the owner confirms.

## Architecture

```text
markitect desktop  (one Go process per user)
├─ loopback server on 127.0.0.1:<port>
│   ├─ the UI, embedded in the binary
│   ├─ JSON routes: projects, model views, settings, commands
│   └─ one event stream (server-sent events): Runs, agents, chat
├─ the same functions as the CLI: project, analysis, run, agents
└─ the window: Edge or Chrome in app mode, later a native shell
```

- **One service per user.** The first start runs the service and opens a window, and every later start opens another window on it. Closing the window stops nothing: Runs and chats go on, and a reopened window reconnects to the stream. A lock file in the user's config folder names the port.
- **Same functions as the CLI.** The desktop is an adapter beside `internal/adapters/cli`. Like the CLI, it reaches the services through `internal/bootstrap`, and it parses no command output.
- **Live events.** Today the engine writes the Result when a Run ends. The agent adapters read each agent's output stream (Claude Code's stream-json, Codex's events) and can write it as transcripts ([Manager agents](../../internal/agents/README.md)). The live view needs both as events while the Run works. That means a port in `internal/run` through which the engine reports its schedule, starts, pushes, verdicts and messages, plus each agent's output lines. The CLI can print the same events. This is `internal/run`'s scope, so it is built together with the [Build and evaluation session](../workstreams/build.md).
- **Chat agents.** A chat is an agent like a Manager. It runs in a container with the provider's account login, works on a clone of the project at a commit, and has no folder of the user's mounted ([concept 2.0 §9](../concepts/v2-concept.md#9-platform)). It pushes its proposal to a branch. Accepting the proposal lands it on the target branch like a Run's result, and the user pulls it ([§6.5](../concepts/v2-concept.md#65-landing)).
- **Safety.** The server listens on loopback only and refuses requests addressed to any other host (against DNS rebinding). Every route requires the launch token, and every response carries a strict content security policy. The page loads no external resource.
- **Window.**
  - First, Edge or Chrome in app mode, using the launcher of the retired cockpit ([`window.go`](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/internal/adapters/cockpit/window.go) at the archive point).
  - Then a native shell, so that Markitect starts without a terminal and has its own title bar, taskbar icon, tray and notifications. Wails v3 runs the same UI inside the Go process but is still in beta. Tauri 2 is stable but adds Rust and a second process. The shell slice decides between them after a probe on Windows, Linux and macOS.
- **No terminal window.** On Windows the app's entry point is built as a GUI program (`-H windowsgui`) while `markitect` stays a console program, so `cmd/markitect-desktop` sits beside `cmd/markitect`. macOS gets an `.app` bundle and Linux a `.desktop` entry.
- **UI.** TypeScript with React, built by Vite, with its source in `desktop/`. The built files are committed under `internal/adapters/desktop/ui/` and embedded, so `go build` needs no Node. CI rebuilds them and fails on a difference, as it does for the code map. The source packager learns the UI folder.
- **Look.** The [Glacius Labs design system](https://github.com/Glacius-Labs/Design-System) (`@glacius-labs/design-system`): its CSS, tokens, fonts and icons, with React components that follow its `spec.json`.

| Package | Role |
|---|---|
| `internal/adapters/desktop` (new) | The loopback server, the routes, the event stream, the embedded UI and the window launcher |
| `internal/bootstrap` | Builds the desktop over the same services as the CLI |
| `cmd/markitect-desktop` (new) | The GUI entry point, later the native shell |
| `internal/run` | Gains the event port, built with Build |
| `desktop/` (new, not Go) | The UI source |

## Delivery

Each slice is one PR with its tests and is useful on its own.

1. **D1, Projects and hub.** `markitect desktop`, the service, the browser window, the project list, creating and opening projects, `init`, and the hub with the model check.
2. **D2, Native shell.** The probe and the shell, no terminal window, the app icon and the title Markitect.
3. **D3, Model.** Read, Map, Organization, Changes and the lint signals.
4. **D4, Runs.** Starting and stopping `apply` and `verify`, and the Runs with their Results.
5. **D5, Live view.** The engine's event port, built with Build, then the live view and playback.
6. **D6, Chat.** Chat agents in containers, proposals as diffs, and Extract.
7. **D7, Statistics and Settings.**

`improve` and the other commands join as they are built; installers and updates come with the first release.

## Tests

- **Go:** `httptest` for every route, the host check, the token, and that no external resource is loaded. One end-to-end test of `markitect desktop --no-open` against a fixture project in `testdata/`.
- **UI:** component tests with Vitest, and Playwright against the fixture project.
- **Screenshots:** each slice's PR carries them at 1440 and 390 pixels wide.

## Not in this design

A personal agent beyond the project chat ([IDEA-015](../concepts/register.md#idea-015-cockpit-with-a-conversational-personal-agent)), the loop of continuous change ([concept 2.0 §10](../concepts/v2-concept.md#10-continuous-change)), and Government.

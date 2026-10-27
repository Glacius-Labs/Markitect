# Cockpit (lane 11)

The cockpit lane is about a desktop UI, in the spirit of Claude Desktop or the ChatGPT app, in which people see and understand the canonical model and its runs. The owner's reason is that at some point people must also understand the canonical model. The Cockpit is outside V1 ([V1 concept, scope](../concepts/v1-concept.md#2-v1-scope)). This page holds the lane's scope, what exists, what understanding a model needs and the open owner questions.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. It is a topic lane: it takes stock and discusses with the owner, and builds nothing until the owner says so.

## Scope

- **Owns:** this page, and later the Cockpit's own code and design page once the owner starts the build.
- **Uses, does not own:**
  - The compiler's `markitect check` report, which is the Cockpit's main input (lane 8).
  - The model language and its Kinds (lane 6), the Libraries and the modeling skills (lane 7).
  - The Run and Apply records of each variant (lanes 1 to 3) and the evaluation suite's results (lane 4).
  - The [model analysis tool](../../tools/model-analysis/README.md) under `tools/`.
- **Elsewhere:** running Markitect on Windows or Linux is the platform lane (lane 5). Starting runs from outside is the commands lane (lane 9) and the continuous-change lane (lane 10).

## What existed

From 20 to 21 October 2026 Markitect had a cockpit ([DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration), [DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [design](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/cockpit.md)).

- **Shape:** `markitect cockpit` ran a Go HTTP server on a loopback address and opened a plain-JavaScript page in an app window of the installed Edge or Chrome. No build chain, no external resources.
- **Areas:** a hub and four areas, as the owner asked on 21 October: Model (browse Packages and Definitions, the organization, model sessions), Runs (start, stop, live view, replay), Decisions (proposals) and Settings. Codex built an organization chart of the Manager tree.
- **Size:** about 65 Go and JavaScript files with their tests, about 780 KB in the adapter alone, built in about two days.
- **End:** the V1 cutover on 23 October removed it with Extract, authoring and Setup. Its source is kept on the archive branch `archive/v1-pre-cutover-20261023` ([adapter README](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/internal/adapters/cockpit/README.md), [archive](../archive.md)). Owner use was not established.
- **Writer:** Codex was the single writer of that cockpit scope while it built the chart. That scope no longer exists on `main`; no code of this lane exists yet.

## What is reusable

From `main` today:

- **The JSON report of `markitect check`.** It carries every local Definition with its Kind, body, typed references and resolved body links, the imported type contracts, the registry records and the execution ownership. It is the natural data source for a model view.
- **`markitect context PACKAGE`.** It shows what one Manager sees and why each Definition was selected.
- **`internal/organization`.** It derives the Manager tree from the Packages and resolves execution settings.
- **`markitect docs`.** It renders a readable generated view of the model.
- **The model analysis tool.** It already computes most of the understanding: the Package graph with coupling and cycles, the references to a Definition, the impact between two model states, and lint signals. It is Python and reads the check report.
- **Closed Run records** (`internal/adapters/runstore`). Each holds per-Manager session observations (profile, image, harness digest), Findings that may name a Definition, and handoffs with their decision, diff, children and check.

From the archive, as reference only:

- The UI ideas: the hub and areas, the organization chart layout, the Definition browser with link navigation and backlinks, replay up to a chosen record, the loopback-only server and the app window.
- Not the code. It was written for the pre-V1 Apply: an `events.jsonl` stream, proposals, model sessions and setup, none of which exist today. Reviving it would be a port, not a restore.

Gaps today:

- **No live view of a run.** The Run record is written at the end. During a Run only Notes and provider transcripts appear, and there is no event stream. A live view needs one from the host.
- **Records sit in the engine volume.** On Windows, Apply runs in the Linux runner, and its records live in the Docker volume `markitect-state-<repositoryIdentity>`. A Windows UI cannot read them directly.
- **Each variant records runs differently.** The host variant (lane 1) has Run records, the Claude Code plugin variant (lane 2) has Claude Code sessions, and the Git variant (lane 3, [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) has a branch per Manager.

## What understanding a model needs

1. **Map.** The Package tree with its size, and the Package dependency graph with coupling, cycles and broken dependency rules.
2. **Definition reader.** The body with clickable links, the typed fields, the Kind, everything that refers to it, and the files that realize it.
3. **Kinds.** The project's own Kinds and the Library Kinds, with their instances.
4. **Impact.** Between two model states (two commits): what changed, who refers to it, which files and which Managers are affected.
5. **Manager tree.** Which Manager owns which Package, what each one sees with the reasons, and its execution profile.
6. **Quality signals.** The lint warnings, placed on the map.
7. **Model to code.** From a file to its Definition and back, through the registry.
8. **Run views.** For a closed run: the Manager tree with each Manager's outcome, handoffs, checks and the diff, and the Findings pinned to their Definitions, so one sees which parts of the model caused trouble. For Apply: the sequence of Implement, Verify and corrections. Later, a live view.

Points 1 to 7 need only the check report and the analysis logic, so no host change. Point 8 needs run records that the UI can read.

## Open owner questions

1. **When and what first.** The Cockpit is outside V1. When should building start, and with what? The recommendation is a read-only model explorer (points 1 to 7) first, then views of closed runs, and a live view last.
2. **Technology.** The recommendation is the old shape: a local server inside `markitect` and a plain page in an app window, with no build chain. The alternatives are a native shell (Tauri, Electron, Wails) or static HTML generated like `markitect docs`.
3. **Where the graph logic lives.** The analysis logic is in the Python tool under `tools/`, the product is Go. Should the Cockpit start as a tool next to it, or move that logic into Go in the product? The recommendation is to move it into the product once the explorer is wanted, so there is one implementation.
4. **Read-only or editing.** DEC-065 let an Agent propose model changes from the cockpit. Interactive authoring is outside V1 and touches the modeling-skills lane. The recommendation is read-only first; editing stays in the editor and the agent chat.
5. **One view for all variants.** Should run views read a variant-neutral record, perhaps the evaluation suite's result format, or one adapter per variant? This needs lanes 1 to 4.
6. **Who writes the code.** Codex's single-writer role ended with the old scope. The recommendation is that this lane's sessions write the new code.
7. **A portal at all.** One outside view says not to build a new portal and to reach people where they already work, such as GitHub or chat ([Zach Lloyd](https://x.com/zachlloydtweets/status/2110808772058960367), cited by Traycer's [software factory](https://traycer.ai/software-factory)). The same guide shows a result view that works: before a release, one page groups the changes into areas, names the risky ones and why, what was checked and what was not, and every human call as a decision with a recommendation. People read it in five minutes and decide. Should a run's result reach the owner like that, as a page or a PR comment, before any desktop app? The recommendation is yes for run results; understanding the model still needs an explorer.
8. **Personal agent.** A chat inside the Cockpit ([IDEA-015](../concepts/register.md#idea-015-cockpit-with-a-conversational-personal-agent)) stays parked unless the owner says otherwise.

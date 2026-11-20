# The plugin

The jobs and reading entries of the rebuilt tool `markitect-next` as a Claude Code plugin: skills that a session runs and agents it starts, one per role ([DEC-132](../decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners), [DEC-141](../decisions.md#dec-141-agents-get-tasks-not-process)). [Jobs and runners](../concept/jobs-and-runners.md#jobs) and [the factory](../concept/the-factory.md#the-jobs) in the concept say what the jobs are for; this page says what the plugin holds and how firm its checks are. The examples are a small lending library.

## What it holds

The plugin is the folder [`plugin-next/`](../../plugin-next/), named `markitect-next` in its `.claude-plugin/plugin.json`, beside the plugin of the 0.34.0 tool until it replaces it ([roadmap](../roadmap.md#current-steps)). It holds no hook, no launcher and no program: the session drives the work, and `markitect-next` only computes and checks. Until a launcher fetches the version a project pins, `markitect-next` is built from source and put on the PATH, as the `getting-started` skill says.

| Path | Holds |
|---|---|
| `agents/ROLE.md` | One agent per role of the tool: its task and its answer form, with `name`, `description`, `tools` and `model` |
| `skills/NAME/SKILL.md` | One skill per job and per reading entry, and `getting-started` |
| `jobs.md` | The frame every job skill runs in: the job's branch and folder, the steps, the reports and the landing |

Claude Code names them with the plugin's scope: the agent `markitect-next:builder`, the skill `markitect-next:markitect-build`.

## Skills and jobs

| Skill | Job | Starts | Lands |
|---|---|---|---|
| `markitect-architect` | Conversations with the owner: the first design, ideas, the owner's choice; and the architect job | `architect` | The spec change, through the build job; an idea's branch waits for the owner |
| `markitect-build` | Build | `counter-checker`, `builder`, `reviewer`, round after round | Counter-checks, artifacts, verdicts and reports |
| `markitect-review` | Review: a spot check, or all documents | `reviewer` | Verdicts and reports; a new `fail` goes to the build job |
| `markitect-control-sample` | Control sample | `sampler` and a fresh `reviewer` on a throwaway copy | Only `.markitect/samples/NAME.md` |
| `markitect-bughunt` | Bug hunt | `reviewer`, hunting | Reports |
| `markitect-cleanup` | Cleanup | `builder`, then the build rounds | Artifacts with no change of behaviour |
| `markitect-improve` | Improve | `builder`, then the build rounds | Artifacts changed only where the spec leaves them free, and reports |
| `markitect-read` | Read and question | `reader` | Reports |
| `markitect-triage` | Triage | `triage` | Rating, Actions and Closing of reports, then the job of each switch |
| `markitect-intake` | Intake | `intake`, which reads the signal isolated | One report with Source `intake` |
| `markitect-briefing` | The briefing, a reading entry | None | Nothing |
| `markitect-explain` | Explain, a reading entry | None | Nothing |

A job runs as [`jobs.md`](../../plugin-next/jobs.md) says. The session works on a branch of the job's own, in a worktree outside the project folder, since a worktree inside it would be a file the project does not explain. Before each step it commits and lays out the before-state, a detached worktree at that commit; it starts the role's agent with the model `markitect-next settings --json` names for it; after the step it runs `markitect-next status --step ROLE --before BEFORE --target TARGETDIR` and commits. At the end it lands the branch by a merge when `markitect-next status --target TARGETDIR` exits 0, the `tests` command of the settings exits 0 and every step check passed. The session holds every loop, the three rounds without a shorter list of the build job included; no agent starts another.

**The last look** of the briefing lies in the reader's profile, `~/.markitect/next/last-look/ID/`, ID the project folder's real path with every character but letters and digits replaced by `-`: `seen/`, an export of the commit last seen, which `compare` reads as a project, `seen-commit`, and `next/` with `next-commit` while a briefing is shown. The 0.34.0 tool uses no path below `~/.markitect/next/`.

## The answer form

Agents write no file in `.markitect/`. What they find they give in their answer, and the session writes it.

- **A finding** is a fenced `report` block: a title, then the sections Place, Finding, Occurred or risk and Violated property in the [report form](reports.md#reports). The session writes it as `.markitect/reports/NAME.md` with the section Source, `job JOB ROLE, DAY`, or `intake, DAY` for intake, and checks it with `markitect-next check`. Rating, Actions and Closing are triage's.
- **The planted deviations** of the sampler are a fenced `planted` block of lines `PATH: DEVIATION`, which the session writes into the sample's Planted section.
- **The triage** agent writes the Rating, Actions and Closing sections into the reports itself, as its role may, and answers with the switch of each report.

```report
# The fee for a late book
## Place
docs/spec/lending/return.md
## Finding
The spec says when a book comes back, and not what a late one costs.
## Occurred or risk
risk
## Violated property
complete
```

## How firm it is

In a session the runner is Claude itself ([where the check is a fact](../concept/jobs-and-runners.md#rights-come-from-the-steps)). So every check of the plugin makes shortcuts visible and proves nothing:

- **The step check and the landing check** compare folders the session lays out and run where the session runs them. A session that lays out another before-state, or skips a check, is not stopped. In CI, where a script no agent can change runs them, they are a fact.
- **The agents' limits** are Claude Code's: an agent's `tools` and the session's permissions. Agents run with the rights of whoever runs them.
- **A counter-checker reads no artifact** only as its task says: nothing checks what an agent reads.
- **A control sample is blind** only as far as the reviewer is told nothing: `markitect-next status` in the copy lists the documents whose artifacts the sampler changed, and the harmless changes it makes beside the deviations are what keeps that list from naming the planted ones.
- **Intake keeps outside text from the session** only as long as the session saves the signal without reading it.

## The check of the plugin

A test of the command line, `internal/cli/plugin_test.go`, reads every skill, every agent, `jobs.md` and this page. Every command line of `markitect-next` in their code must parse as the tool parses it, with a command the tool has and only that command's flags; every word after `markitect-next` must be a command; every skill or agent named with the plugin's scope must exist, and every relative link must lead to a file. It checks the front matter of each file, and that the plugin has an agent for each role of the tool and no other.

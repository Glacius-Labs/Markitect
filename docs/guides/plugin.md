# The plugin

The `markitect` command, and the jobs and reading entries of Markitect as a Claude Code plugin: skills that a session runs and agents it starts, one per role ([where Main is](../roadmap.md#where-main-is)). [Where Main is](../roadmap.md#where-main-is) says how the plugin relates to the concept; this page says what the plugin holds, how it brings the tool and how firm its checks are. The examples are a small lending library.

## What it holds

The plugin is the folder [`plugin/`](../../plugin/), named `markitect` in its `.claude-plugin/plugin.json`. The session drives the work, and `markitect` only computes and checks.

| Path | Holds |
|---|---|
| `agents/ROLE.md` | One agent per role of the tool: its task and its answer form, with `name`, `description`, `tools` and `model` |
| `skills/NAME/SKILL.md` | One skill per job and per reading entry, and `getting-started` |
| `jobs.md` | The frame every job skill runs in, which the skills name as `${CLAUDE_PLUGIN_ROOT}/jobs.md`: the job's branch and folder, the steps, the reports and the landing |
| `bin/markitect`, `bin/markitect.cmd` | The launcher, and its shim for PowerShell and cmd on Windows ([below](#the-launcher)) |
| `hooks/hooks.json`, `hooks/path.sh` | Two hooks at the start of a session, both for the launcher; the plugin has no hook before a tool |

Claude Code names them with the plugin's scope: the agent `markitect:builder`, the skill `markitect:markitect-build`.

## The launcher

While the plugin is on, Claude Code puts its `bin/` folder on the PATH of its shell, so a session runs `markitect` without anyone installing it. The first call downloads the binary of the release that has the plugin's own version, `markitect-vX.Y.Z-OS-ARCH` with `.exe` on Windows, and `markitect-vX.Y.Z-checksums.txt` from the [releases](https://github.com/Glacius-Labs/Markitect/releases), checks the binary against the checksums, keeps it in the plugin's data folder, `~/.claude/plugins/data/markitect-glacius-labs/vX.Y.Z/`, and starts it. Later calls start it at once, and a plugin update with a new version fetches the matching binary the same way. `MARKITECT_RELEASES_URL` names a mirror with the same layout. The launcher fetches the plugin's version, not the version the project pins; fetching the pinned version is step 5 of the [roadmap](../roadmap.md#current-steps), and `markitect version` says whether the two match.

A development version of the plugin, such as `0.35.0-dev`, has no release binary and fetches nothing: its launcher starts the next `markitect` on the PATH, passing over its own `bin/`, with every argument. So for development, build the tool from source, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@COMMIT` with the commit of the plugin, and keep its folder on the PATH; the plugin's `bin/` may come before it.

The two hooks at the start of a session serve the launcher alone. The first puts `bin/` on the PATH of the Bash tool, which on Windows does not get it from Claude Code: it appends one line to the file `CLAUDE_ENV_FILE` names, and Claude Code waits for it, so that the first command finds `markitect`. The second fetches the binary in the background, outside the shell's sandbox, in a project that has a `.markitect/` folder; in any other project the plugin downloads nothing until `markitect` is called.

| Observation | Response |
|---|---|
| The launcher says it cannot download a file of the release | The launcher could not reach the releases page; a sandboxed shell may block it. Start a new session in the project, whose start hook fetches outside the sandbox, or run `markitect` once in a session without the sandbox |
| The launcher says a development plugin needs the tool built from source | No `markitect` follows the plugin's `bin/` on the PATH. Build the tool from source, as above, and put its folder on the PATH |
| `markitect` exists in Claude Code but not in your terminal | The plugin puts it on the PATH of Claude Code's tools only. Download the binary of the release, check it against the checksums file and put it on your PATH as `markitect` |
| `markitect: command not found` in the Bash tool on Windows | The first start hook puts it on the Bash tool's PATH. A session that started before the plugin was installed has no such hook: start a new one |
| In PowerShell, the launcher says it needs the bash of Git for Windows | `markitect.cmd` runs the launcher with the bash that belongs to the `git.exe` on the PATH, and found none. Set `CLAUDE_CODE_GIT_BASH_PATH` to the `bash.exe` of Git for Windows, as Claude Code itself reads it |

## Skills and jobs

| Skill | Job | Starts | Lands |
|---|---|---|---|
| `markitect-architect` | Conversations with the owner: the first design, ideas, the owner's choice; and the architect job | `architect` | The spec change, through the build job; an idea's branch waits for the owner |
| `markitect-build` | Build | `counter-checker`, `builder`, `reviewer`, round after round | Counter-checks, artifacts, verdicts and reports |
| `markitect-review` | Review: a spot check, or all documents | `reviewer` | Verdicts and reports; a new `fail` goes to the build job |
| `markitect-control-sample` | Control sample | `sampler` and a fresh `reviewer` on a throwaway copy | Only `.markitect/samples/NAME.md` |
| `markitect-bughunt` | Bug hunt | `reviewer`, hunting | Reports, and what triage answers on its branch |
| `markitect-cleanup` | Cleanup | `builder`, then the build rounds | Artifacts with no change of behaviour |
| `markitect-improve` | Improve | `builder`, then the build rounds | Artifacts changed only where the spec leaves them free, and reports |
| `markitect-read` | Read and question | `reader` | Reports, and what triage answers on its branch |
| `markitect-triage` | Triage | `triage` | Rating, Actions and Closing of reports, and the answer of each switch on its branch |
| `markitect-intake` | Intake | `intake`, which reads the signal isolated | One report with Source `intake` |
| `markitect-briefing` | The briefing, a reading entry | None | Nothing |
| `markitect-explain` | Explain, a reading entry | None | Nothing |

A job runs as [`jobs.md`](../../plugin/jobs.md) says. The session works on a branch of the job's own, in a worktree outside the project folder, since a worktree inside it would be a file the project does not explain. In a session, Claude Code asks before it writes outside the project. Before each step it commits and lays out the before-state, a detached worktree at that commit; it starts the role's agent with the model `markitect settings --json` names for it in `settings.models`; after the step it runs `markitect status --step ROLE --before BEFORE` and commits. The step check reads the write table from the before-state: the job's branch starts at the target's commit, and a step that changes a control file is denied and keeps the branch from landing. At the end the session lays out a checkout of the target and lands the branch by a merge when `markitect status --target TARGETDIR` exits 0, the command in `settings.tests` exits 0 and every step check passed.

An open report on a document of the spec keeps the branch from landing unless it is escalated ([waiting on a report](reports.md#waiting-on-a-report)). So a job that writes one runs triage on its own branch before it builds on: an escalated report lands open and waits on the owner, and any other is answered on that branch and closed `done` there by triage, so that its closing lands with its answer. The owner's choice between proposals works the same way: it replaces `escalated` with the branch of the chosen proposal, and the report lands closed with the built change. The session holds every loop, the three rounds without a shorter list of the build job included; no agent starts another.

**The last look** of the briefing lies in the reader's profile, `~/.markitect/last-look/ID/`, ID the project folder's real path with every character but letters and digits replaced by `-`: `seen/`, an export of the commit last seen, which `compare` reads as a project, `seen-commit`, and `next/` with `next-commit` while a briefing is shown.

## The answer form

Agents write no file in `.markitect/`. What they find they give in their answer, and the session writes it.

- **A finding** is a fenced `report` block: a title, then the sections Place, Finding, Occurred or risk and Violated property in the [report form](reports.md#reports). The session writes it as `.markitect/reports/NAME.md` with the section Source, `job JOB ROLE, DAY`, or `intake, DAY` for intake, and checks it with `markitect check`. Rating, Actions and Closing are triage's.
- **The planted deviations** of the sampler are a fenced `planted` block of lines `PATH: DEVIATION`, which the session writes into the sample's Planted section.
- **The triage** agent writes the Rating, Actions and Closing sections into the reports itself, as its role may, and answers with the switch of each report. Jobs and the briefing read reports through `markitect reports`, not from the files.

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

In a session the runner is Claude itself ([where Main is](../roadmap.md#where-main-is)). So every check of the plugin makes shortcuts visible and proves nothing:

- **The step check** compares folders that Claude lays out, in a session as in CI, so with skills it makes shortcuts visible everywhere and proves nothing. What is a fact in CI is the landing check and the tests, run by the workflow as a step of its own, outside every agent.
- **The landing check** in a session runs where the session runs it; a session that skips it is not stopped.
- **The agents' limits** are Claude Code's: an agent's `tools` and the session's permissions. Agents run with the rights of whoever runs them.
- **A counter-checker reads no artifact** only as its task says: nothing checks what an agent reads.
- **A control sample is blind** only as far as the reviewer is told nothing: `markitect status` in the copy lists the documents whose artifacts the sampler changed, and the harmless changes it makes beside the deviations are what keeps that list from naming the planted ones.
- **Intake keeps outside text from the session** only as long as the session saves the signal without reading it.

## The check of the plugin

A test of the command line, `internal/cli/plugin_test.go`, reads every skill, every agent, `jobs.md` and this page. It covers what stands in code spans and fenced blocks: every command and flag of `markitect` there must parse as the tool parses it, with a command the tool has and only that command's flags, a role of `--step` must be one of the tool's, and a template of `apply` one of the library's. A value of a command's JSON is written `COMMAND.KEY`, such as `settings.tests` or `status.waiting.counts`, and must be a key of that command's JSON document. Every word after `markitect` must be a command, every skill or agent named with the plugin's scope must exist, and every relative link and every path below `${CLAUDE_PLUGIN_ROOT}` must lead to a file. It checks the front matter of each file, and that the plugin has an agent for each role of the tool and no other. What the texts ask of an agent, it does not check.

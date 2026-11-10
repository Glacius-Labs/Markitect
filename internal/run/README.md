# The record

`internal/run` holds the record of what was verified ([DEC-099](../../docs/decisions.md#dec-099-a-record-of-what-was-verified), [concept, §6](../../docs/concept.md#6-the-record)). Like Go's test cache, it keeps for each Definition the state at which a reviewer last verified it, and so says what is left to verify. The tool supports agent work and does not steer it: it computes, checks and records, and decides nothing about who works when. Whoever drives the agents (a skill, a workflow or a person) starts them and makes every Git step. Nothing here starts or drives an agent, and nothing uses Git ([DEC-096](../../docs/decisions.md#dec-096-the-tool-computes-and-checks-a-driver-acts), [DEC-097](../../docs/decisions.md#dec-097-markitect-uses-no-git)).

| Package | Role |
| --- | --- |
| `internal/run` | The record: the inputs of each Definition, `.markitect/review.sum`, `status`, `mark`, the hook, the migration from the certificates of 0.20.0 and the move of the spec folder. |
| `internal/adapters/folder` | The walk of the project folder and its `.gitignore` files, for the record and the command line ([below](#the-folder)). |
| `internal/bootstrap` | Builds the engine for the command line, with the lock of each project's record in the user's cache folder. |

## The folder

The project folder is the nearest folder with `.markitect/manifest.yaml`. Its files come from walking it, without what the project's `.gitignore` files exclude, read as plain text: nested ones, and those of the folders above it up to the top of the Git repository it lies in, each from its own folder, as Git applies them ([`folder.Disk`](../adapters/folder/folder.go)). A `.gitignore` above the project folder that excludes the project folder itself, or a folder on the way to it, is not applied, though Git would ignore everything there. A folder named with `--repo` must exist, so that a mistyped one is not taken for the project around it. Patterns follow Git: `!` brings a file back unless its folder is excluded, a trailing `/**` holds what is inside a folder and not the folder, and `[!x]` means "not x". Git's own folder is not walked. Another repository, a folder whose `.git` is a folder with a `HEAD` or a file naming one, is one entry, as in Git, so that `check` names it until it is registered or ignored; its files are not walked, and a change inside it is not seen. A symbolic link counts as where it points, as Git stores it. Global Git settings are out of scope, and so are files Git tracks although a `.gitignore` excludes them: the walk does not see those, and a registry row for one is an error. Every command reads the folder this way, the spec's commands included.

## The inputs

A Definition's inputs are everything its verification depended on, each by its SHA-256:

| Key | Input |
|---|---|
| `self` | The Definition's own text |
| `kind:ID` | Its Kind |
| `link:ID` | A Definition it links to |
| `file:PATH` | A file whose registry row names it |
| `evidence:PATH` | A file the reviewer gave as evidence |

A Definition of the spec counts by its text. A Library Definition counts by its Library's digest in the lock, so that a newer binary changes nothing until the lock is written again; a Library counts as one, as a Go module does, so a change of one Kind of a committed Library puts the Definitions of all its Kinds on the list. Core's `Kind` is no input, and neither is a link to a Package. Line endings do not count in a text file's hash, so a checkout on another system matches.

This reaches what the [concept](../../docs/concept.md#how-status-works) puts at distance 0 and 1: a Definition that changed, one that links to a changed Definition or is of a changed Kind, one whose files changed, and one whose registry rows gained or lost it. A changed Rule above a Package reaches only itself and what links to it. A change of a row's reason, or of a file the ignore file lists, reaches nothing. The one difference, the Library counted as a whole, lists more, never less.

The **fingerprint** of an item is the first six characters of the hash of every input but the evidence, which the reviewer chooses after it read the item. `mark` also takes a longer prefix of the hash.

## The record file

`.markitect/review.sum` is part of the project's fixed layout and committed with it, like `go.sum`. It holds one line per Definition, sorted, each a JSON object with the Definition, the verdict, the inputs with their full hashes, the evidence as `path` or `path:line` and the reason on one line:

```text
{"definition":"members/loans/loan","verdict":"ok","inputs":{"evidence:src/members/loans/loan.go":"sha256:…","file:src/members/loans/loan.go":"sha256:…","kind:std/Concept":"sha256:…","link:members/member":"sha256:…","self":"sha256:…"},"evidence":["src/members/loans/loan.go:3"]}
```

The verdicts are `ok` (implemented, as the evidence shows), `fail` (not, and why), `skip` (verified as far as the project can show, and why) and `spec-problem` (the spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it). The record holds hashes, not text: an agent that wants what a Definition said before reads it from Git.

Three optional fields, a trial of [DEC-102](../../docs/decisions.md#dec-102-what-vet-measures), are each left out when empty:

- `cases` and `covered`: the reviewer's count of the distinct cases, conditions and exceptions the Definition demands, and of those its evidence demonstrates, as `mark` took them. Nothing reads them yet; they are kept to calibrate a later risk measure.
- `revisions`: how often the Definition's own text changed between two verdicts on it. `mark` takes the count of the Definition's lines and adds one when none of them has the new line's `self` hash, so a first verdict has none, a change of a link, a file or the evidence adds none, and so does a verdict on a gone Definition, which has no own text. Of several lines, as after a merge, it counts on from the most. `vet` reads it for `vet.hotspot`, as change history without Git.

The record has no format version: the fields are optional, so a line without them reads as no count and no revision, and its counting starts with the next verdict.

**Merging.** `init` and `update` write `.markitect/.gitattributes` with `review.sum merge=union`, which also joins the layout, so Git keeps the lines of both sides of a merge. Of several lines for one Definition, `status` takes the one whose inputs match the folder; two that match with different verdicts are a conflict to verify again. A line markitect cannot read, such as a broken one written by hand, is an error until the owner fixes or removes it.

The record is a record, not a proof: a line written by hand shows in the diff, and [the hook](#the-hook) keeps agents from writing one.

## status

`markitect status [--all]` reads the folder and the record and changes nothing. It lists:

- every Definition with a changed input or no record, with which inputs changed: `changed: self, file:src/x.go (new), evidence:tests/x_test.go (gone)`;
- every Definition recorded `fail` or `spec-problem` at its inputs as they are, with why; a changed one also says what was recorded before;
- every Definition gone from the spec that still has a line, with the files that line names: are they removed, or do they have a new basis?
- every conflict;
- the errors in the realization: a file neither listed nor ignored, a stale registry row, an unused Definition;
- the lines of the record it cannot read, each as a diagnostic `.markitect/review.sum:LINE: record.unreadable: …`.

Each item is printed as `DEFINITION@FINGERPRINT`, which the reviewer copies into `mark`, and names where to look: the Definition's file, its row files and the evidence of its last line. A Definition recorded `skip` is a note. A spec with errors gives no list: there is none until it compiles. `--all` lists every Definition, to verify everything again.

It exits 0 when it lists nothing: then the project compiles and every Definition is verified at its current inputs. It exits 1 when it lists something, and so with `--all` whenever the spec has a Definition. Notes never change the exit code, a `skip` or the note on standard error that the project's harness comes from another markitect alike. It looks at no network: whether a newer binary exists is `update --check`'s question.

## mark

`markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]… [-m "WHY"] [--cases N [--covered M]]` records a verdict at the state the reviewer read:

```text
markitect mark members/limit@3f9a1c ok -e src/loans.go:42 -e tests/loans_test.go:88
markitect mark loans/renewal@81be07 fail -e src/renew.go:30 -m "allows a fourth renewal; the Definition says three"
```

- `ok` names at least one piece of evidence with `-e`; `fail`, `skip` and `spec-problem` say why with `-m`, kept on one line.
- `--cases N` and `--covered M` are counts the reviewer may give with any verdict: N cases, conditions and exceptions the Definition demands, M of them demonstrated by its evidence. M needs N and is at most N; both are 0 or more. They go on the line as `cases` and `covered`.
- Evidence is `path` or `path:line`, also `path:line-line` and `path:line:col`. A trailing note in parentheses, such as a test name, is dropped, not stored; a note without them is refused, since `-m` holds the reason. Each file must be a file of the folder, named from the project folder; its hash becomes an input when the verdict is recorded.
- It refuses, records nothing and exits 1 when the inputs differ from the fingerprint, as when a builder edited a file beside the reviewer or the reviewer's own test run changed one; when evidence is not a file of the folder; when the Definition is unknown; when the spec has errors; or when the record has a line it cannot read. A command it cannot read exits 2.
- A gone Definition is marked `ok` when its files are removed or have a new basis, which removes its line, or `fail` when not; `skip` is refused.

It writes under a lock in the user's cache folder (`markitect/locks/`, one per project folder), reading the folder and the record afresh, and replaces the record as one whole file in one step, so parallel reviewers, such as one per Package, lose no line. It writes the file clean: one line per Definition, the matching one of several, both of a conflict, the closest when none matches, and one line for a gone Definition with every file its lines named. Each line it keeps counts the most revisions any line of its Definition counted, so a merge loses none.

## The hook

One command, `markitect hook pretool`, which `init` and `update` add to the project's `.claude/settings.json` in three groups before a tool call: one before a shell command (`if: "Bash(markitect mark*)"`), one before every tool that writes a file (`Edit|Write|MultiEdit|NotebookEdit`), and one before every tool that starts an agent or a workflow (`Agent|Task|Workflow`). Hooks in project settings run inside subagents too, and their input names the agent. Each group runs `command -v markitect >/dev/null 2>&1 || exit 0; MARKITECT_HOOK=project markitect hook pretool` in bash, as the plugin's hooks do: it calls `markitect` only where its shell finds one, which is a binary on the `PATH`. A hook does not see the plugin's `bin/`, so with the plugin alone it does nothing, and the plugin's own hooks guard. Called with `MARKITECT_HOOK=project`, the binary leaves the guard to the plugin where the plugin guards too, so that it runs once: where the last of the user's, the project's and the local settings that names `markitect@…` enables it, and Claude Code's `plugins/installed_plugins.json` lists it with an install whose `hooks/hooks.json` runs `hook pretool` before the shell and before every tool that writes a file or starts an agent; an older plugin's hooks do not count. What it cannot read counts as no, so a doubt costs a second run of the guard, never the guard. The hook holds the rights of the four roles ([DEC-100](../../docs/decisions.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)) and guards the record and the spec, not the process:

- **Only the reviewer agent runs `markitect mark`.** So the agent that built does not verify its own work. A call from the owner's session through Claude Code is refused too.
- **A subagent writes only what its role may.** The builder, and any subagent that is none of the four roles, writes code (every file outside `.markitect/`) and the registry pages (`registry.md` and `registry/*.md`); the architect writes the spec (the rest of `.markitect/`) and the registry pages; the reviewer and the auditor write nothing; and nobody writes `review.sum`, which only `mark` writes, or the files that steer the agents: the project's `.claude/settings.json` and `settings.local.json`, Markitect's skills, agents and workflows (`.claude/workflows/markitect-*`), the harness configuration `.markitect/harness.yaml`, which the architect may not write either, and `AGENTS.md` and `CLAUDE.md`. So a builder cannot weaken a Definition during the work, write lines into the record, or drop the hook from work that comes back by itself. The owner's own session, where the owner architects, is no subagent and stays free.
- **The builder starts only the reviewer,** the other roles start no agent, and no subagent starts the builder or the architect. Claude Code ignores the agent types a subagent's own `tools` list, so the hook holds this.
- **No subagent runs a workflow,** since a workflow starts agents of every role ([DEC-103](../../docs/decisions.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)). The owner's session starts the parallel workflow, and its agents' calls come to the hook with their types like any subagent's.

The hook reads the call as text, so a command hidden in a script, or one that names `markitect` by a path or another spelling, gets through: the project's hook starts only for what Claude Code's rule `Bash(markitect mark*)` matches. It guards against mistakes, not against intent. The plugin's hooks filter in their own shell, without regard to case, before starting its launcher, and those before writing and agent tools act only in a project that has a `.markitect/` folder. `AGENTS.md` and `CLAUDE.md` are matched as written, so a `docs/agents.md` of the project stays the builder's.

## The agents and skills

The four agents are files `init` writes, each holding its tasks: the builder (build, clean up, hunt bugs), the reviewer (verify, hunt bugs), the architect (write large parts of the spec) and the auditor (read the spec and report). The builder may read, edit, run the shell and start agents, which the hook narrows to the reviewer; the reviewer and the auditor have no edit or agent tools, and the architect no agent tool. Each defaults to `sonnet` with effort `high`; `init`, `update` and `markitect config` write the model and effort the harness configuration gives each role into its file ([DEC-103](../../docs/decisions.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow), [strategies and configuration](../../docs/guides/project-workflow.md#strategies-and-configuration)). Everything else comes from the owner's session: its permission mode, settings, plugins and memory.

The skills are plain prompts with no injected command, so no exit code can abort them. `/markitect-build`, `/markitect-cleanup` and `/markitect-bughunt` start the builder in a worktree of its own on its task; it runs `status`, does its work, starts the reviewer, and repeats until `status` lists nothing it can settle or only what needs the owner, or three rounds left the list no shorter. The session then commits the work on the worktree's branch and merges it into the owner's branch: build when `status` is empty there, cleanup and bughunt when it lists nothing new. `init` sets `worktree.baseRef` to `"head"` so the worktree starts from the owner's branch, and the tool accounts for Claude Code's worktrees under `.claude/worktrees/` itself. `/markitect-review` starts the reviewer on what `status` lists, or on everything with `--all`; it changes no code, and the session commits the record. Each of these skills runs by its strategy: sequentially as described, or through the parallel workflow `.claude/workflows/markitect-parallel.js`, which works per Package in waves and reads `status --json`, whose `packages` give each Package's items and dependencies. `/markitect-audit` starts the auditor, and `/markitect-schedule` sets up routines through Claude Code's scheduling. The [project workflow](../../docs/guides/project-workflow.md#7-build-and-review) says how to use them.

## The migration

`markitect update` turns a project of 0.20.0 into one with a record. When `.markitect/results/` holds a certificate and the project has no record yet, it writes the record from the newest certificate: newest by base (one that another names as its base is older), then by the time it records. Every Definition of the certified spec is recorded `ok` at the certified state, computed from the spec text and file hashes the certificate holds, with the certificate's pointers as evidence where it has them; a `not shown` answer becomes `skip`. Then it removes the results folder, which no longer belongs to the layout: `check` names a file there as `project.results` until it is gone. A project that already has its record keeps it. `update --check` and `version` count the migration as out of date.

`checklist`, `answer` and `gate` print on standard output, with exit 0, that they are gone and to run `markitect update`, so that an old project's skill does not fail silently.

## The move of the spec folder

An older project keeps its spec in `.markitect/model/`. A project loads a spec there where it is, with one error, `project.model`, that stops every command until `markitect update` moves it ([`MoveSpec`](specmove.go)); `init` refuses until then. `update` does this first, before anything else, and renames the folder to `.markitect/spec/`. A certificate of 0.20.0 names the old folder too, and the migration above reads it as if it named the new one.

The record keeps every verdict. A line is keyed by its Definition's ID, the path below the folder, and its inputs are hashes of content, so the move changes none of them. Only evidence can name a file of the spec, as when a reviewer cites the Definition itself; a `file:` input never does, since a registry row never names a file of `.markitect/`. `update` renames that evidence in place: the key `evidence:.markitect/model/x.md` becomes `evidence:.markitect/spec/x.md` with the same hash, and so does the evidence it lists. Evidence is no part of the fingerprint, so no fingerprint changes. Every other line stays as it is, in its place, also one markitect cannot read. The record is renamed whenever it names the old folder, so a line that a merge brings in from a branch that has not moved yet is renamed by the next `update` too. Like the migration above, the rename takes no lock: `update` is the owner's step, not one beside a running reviewer.

With files in both folders the project has two specs, as when `init` of a newer markitect ran before `update`: `update` refuses, changes nothing and says to keep one. Empty folders count as none. `update --check` and `version` count the move as out of date, and `update` notes when `AGENTS.md`, which it never edits, still names `.markitect/model`.


## Tests

`record_test.go` plays the builder and the reviewer on plain folders without Git while `status` says what is to verify: what a change reaches, the Kind and the Library, the fingerprint and evidence, the verdicts, the counts of cases and the revisions, a gone Definition, merged lines and conflicts, an unreadable line, errors, and parallel reviewers. `migrate_test.go` covers the migration from certificates written as 0.20.0 wrote them, `specmove_test.go` the move of the spec folder with its record and certificates, `hook_test.go` the hook, and `lock_test.go` the lock.

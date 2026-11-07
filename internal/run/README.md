# The record

`internal/run` holds the record of what was verified ([DEC-099](../../docs/concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache), [tooling plan](../../docs/concepts/tooling-plan.md)). Like Go's test cache, it keeps for each Definition the state at which a reviewer last verified it, and so says what is left to verify. The tool supports agent work and does not steer it: it computes, checks and records, and decides nothing about who works when. Whoever drives the agents (a skill, a workflow or a person) starts them and makes every Git step. Nothing here starts or drives an agent, and nothing uses Git ([DEC-096](../../docs/concepts/register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](../../docs/concepts/register.md#dec-097-markitect-uses-no-git)).

| Package | Role |
| --- | --- |
| `internal/run` | The record: the inputs of each Definition, `.markitect/review.sum`, `status`, `mark`, the hook, and the migration from the certificates of 0.20.0. |
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

A model Definition counts by its text. A Library Definition counts by its Library's digest in the lock, so that a newer binary changes nothing until the lock is written again; a Library counts as one, as a Go module does, so a change of one Kind of a committed Library puts the Definitions of all its Kinds on the list. Core's `Kind` is no input, and neither is a link to a Package. Line endings do not count in a text file's hash, so a checkout on another system matches.

This reaches what concept 4.0's checklist reached at distance 0 and 1 ([concept 4.0 §3](../../docs/concepts/v4-concept.md#how-status-works)): a Definition that changed, one that links to a changed Definition or is of a changed Kind, one whose files changed, and one whose registry rows gained or lost it. A changed Rule above a Package reaches only itself and what links to it. A change of a row's reason, or of a file the ignore file lists, reaches nothing. The one difference, the Library counted as a whole, lists more, never less.

The **fingerprint** of an item is the first six characters of the hash of every input but the evidence, which the reviewer chooses after it read the item. `mark` also takes a longer prefix of the hash.

## The record file

`.markitect/review.sum` is part of the project's fixed layout and committed with it, like `go.sum`. It holds one line per Definition, sorted, each a JSON object with the Definition, the verdict, the inputs with their full hashes, the evidence as `path` or `path:line` and the reason on one line:

```text
{"definition":"members/loans/loan","verdict":"ok","inputs":{"evidence:src/members/loans/loan.go":"sha256:…","file:src/members/loans/loan.go":"sha256:…","kind:std/Concept":"sha256:…","link:members/member":"sha256:…","self":"sha256:…"},"evidence":["src/members/loans/loan.go:3"]}
```

The verdicts are `ok` (implemented, as the evidence shows), `fail` (not, and why), `skip` (verified as far as the project can show, and why) and `spec-problem` (the spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it). The record holds hashes, not text: an agent that wants what a Definition said before reads it from Git.

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

`markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]… [-m "WHY"]` records a verdict at the state the reviewer read:

```text
markitect mark members/limit@3f9a1c ok -e src/loans.go:42 -e tests/loans_test.go:88
markitect mark loans/renewal@81be07 fail -e src/renew.go:30 -m "allows a fourth renewal; the Definition says three"
```

- `ok` names at least one piece of evidence with `-e`; `fail`, `skip` and `spec-problem` say why with `-m`, kept on one line.
- Evidence is `path` or `path:line`, also `path:line-line` and `path:line:col`. A trailing note in parentheses, such as a test name, is dropped, not stored; a note without them is refused, since `-m` holds the reason. Each file must be a file of the folder, named from the project folder; its hash becomes an input when the verdict is recorded.
- It refuses, records nothing and exits 1 when the inputs differ from the fingerprint, as when a builder edited a file beside the reviewer or the reviewer's own test run changed one; when evidence is not a file of the folder; when the Definition is unknown; when the spec has errors; or when the record has a line it cannot read. A command it cannot read exits 2.
- A gone Definition is marked `ok` when its files are removed or have a new basis, which removes its line, or `fail` when not; `skip` is refused.

It writes under a lock in the user's cache folder (`markitect/locks/`, one per project folder), reading the folder and the record afresh, and replaces the record as one whole file in one step, so parallel reviewers, such as one per Package, lose no line. It writes the file clean: one line per Definition, the matching one of several, both of a conflict, the closest when none matches, and one line for a gone Definition with every file its lines named.

## The hook

One command, `markitect hook pretool`, which `init` and `update` add to the project's `.claude/settings.json` in two groups before a tool call: one before a shell command (`if: "Bash(markitect mark*)"`), one before every tool that writes a file (`Edit|Write|MultiEdit|NotebookEdit`). Hooks in project settings run inside subagents too, and their input names the agent. The hook guards the record and the spec, not the process:

- **Only the reviewer agent runs `markitect mark`.** So the agent that built does not verify its own work. A call from the owner's session through Claude Code is refused too.
- **No subagent writes under `.markitect/`, except the registry pages** (`registry.md` and `registry/*.md`), which the builder keeps. So a builder cannot weaken a Definition during the work or write lines into the record. The owner's own session, where the spec is modelled, is no subagent and stays free.

The hook reads the call as text, so a command hidden in a script, or one that names `markitect` by a path or another spelling, gets through: the project's hook starts only for what Claude Code's rule `Bash(markitect mark*)` matches. It guards against mistakes, not against intent. The plugin's hooks filter in their own shell, without regard to case, before starting its launcher.

## The agents and skills

Both agents are files `init` writes. The builder may read, edit, run the shell and start only the reviewer (`Agent(markitect-reviewer)`), so it cannot get past the hook through a general-purpose helper; the reviewer has no edit or agent tools. Each defaults to `sonnet` with effort `high`, and the manifest's `model`, if it names one, is written into both agent files by `init` and `update`. Everything else comes from the owner's session: its permission mode, settings, plugins and memory. The agents hold their task and the verdict words in their own files, which outlive compaction.

The skills are plain prompts with no injected command, so no exit code can abort them. `/markitect-build` starts the builder, which runs `status`, makes the project match the spec for what it lists, starts the reviewer, and repeats until `status` lists nothing or only what needs the owner; then the session commits the work with the record. `/markitect-review` starts the reviewer on what `status` lists, or on everything with `--all`; it changes no code, and the session commits the record. Nothing counts rounds: the owner stops the work when it should stop.

## The migration

`markitect update` turns a project of 0.20.0 into one with a record. When `.markitect/results/` holds a certificate and the project has no record yet, it writes the record from the newest certificate: newest by base (one that another names as its base is older), then by the time it records. Every Definition of the certified spec is recorded `ok` at the certified state, computed from the spec text and file hashes the certificate holds, with the certificate's pointers as evidence where it has them; a `not shown` answer becomes `skip`. Then it removes the results folder, which no longer belongs to the layout: `check` names a file there as `project.results` until it is gone. A project that already has its record keeps it. `update --check` and `version` count the migration as out of date.

For one release, `checklist`, `answer` and `gate` print on standard output, with exit 0, that they are gone and to run `markitect update`, so that an old project's skill does not fail silently.

## What went

The checklist, the stored answers, the work under way in the cache, `gate` with its rows, the certificates, the Result and Findings pages, repair rounds and the builder's report of a spec problem went with 0.21.0 ([DEC-099](../../docs/concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)). So did three refusals of `gate` during the work: of a changed file the ignore file lists, of a changed harness, `.gitignore` or `settings.json`, and of a symbolic link that leaves the folder. The [tooling plan](../../docs/concepts/tooling-plan.md#what-this-gives-up-and-what-it-keeps) says what that gives up.

## Tests

`record_test.go` plays the builder and the reviewer on plain folders without Git while `status` says what is to verify: what a change reaches, the Kind and the Library, the fingerprint and evidence, the verdicts, a gone Definition, merged lines and conflicts, an unreadable line, errors, and parallel reviewers. `migrate_test.go` covers the migration from certificates written as 0.20.0 wrote them, `hook_test.go` the hook, and `lock_test.go` the lock.

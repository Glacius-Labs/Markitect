# The record

`internal/run` holds the record of what was verified ([DEC-099](../../docs/decisions.md#dec-099-a-record-of-what-was-verified), [concept, §6](../../docs/concept.md#6-the-record)). Like Go's test cache, it keeps for each item the state at which a reviewer last verified it, where an item is a Definition that is no part of another, with its parts ([DEC-111](../../docs/decisions.md#dec-111-a-thing-with-parts-is-built-and-verified-as-one)), and so says what is left to verify. The tool supports agent work and does not steer it: it computes, checks and records, and decides nothing about who works when. Whoever drives the agents (a skill, a workflow or a person) starts them and makes every Git step. Nothing here starts or drives an agent, and nothing uses Git ([DEC-096](../../docs/decisions.md#dec-096-the-tool-computes-and-checks-a-driver-acts), [DEC-097](../../docs/decisions.md#dec-097-markitect-uses-no-git)).

| Package | Role |
| --- | --- |
| `internal/run` | The record: the inputs of each Definition, `.markitect/review.sum`, `status` and `mark`. [The hook](../guard/README.md#the-hook) that keeps agents from writing it is `internal/guard`'s. |
| `internal/adapters/folder` | The walk of the project folder and its `.gitignore` files, for the record and the command line ([below](#the-folder)). |
| `internal/bootstrap` | Builds the engine for the command line, with the lock of each project's record in the user's cache folder. |

## The folder

The project folder is the nearest folder with `.markitect/manifest.yaml`. Its files come from walking it, without what the project's `.gitignore` files exclude, read as plain text: nested ones, and those of the folders above it up to the top of the Git repository it lies in, each from its own folder, as Git applies them ([`folder.Disk`](../adapters/folder/folder.go)). A `.gitignore` above the project folder that excludes the project folder itself, or a folder on the way to it, is not applied, though Git would ignore everything there. A folder named with `--repo` must exist, so that a mistyped one is not taken for the project around it. Patterns follow Git: `!` brings a file back unless its folder is excluded, a trailing `/**` holds what is inside a folder and not the folder, and `[!x]` means "not x". Git's own folder is not walked. Another repository, a folder whose `.git` is a folder with a `HEAD` or a file naming one, is one entry, as in Git, so that `check` names it until it is registered or ignored; its files are not walked, and a change inside it is not seen. A symbolic link counts as where it points, as Git stores it. Global Git settings are out of scope, and so are files Git tracks although a `.gitignore` excludes them: the walk does not see those, and a registry row for one is an error. Every command reads the folder this way, the spec's commands included.

## The inputs

An item is what is verified as one: a Definition that is no part of another, with its parts. A part that has parts of its own is an item of its own, and the thing above it does not cover it ([`spec.Units`](../spec/spec.go)). An item's inputs are everything its verification depended on, each by its SHA-256:

| Key | Input |
|---|---|
| `self` | The Definition's own text |
| `part:ID` | The text of a part it covers |
| `kind:ID` | Its Kind, or the Kind of a part |
| `link:ID` | A Definition that it or a part links to, outside the item |
| `file:PATH` | A file whose registry row names it or a part |
| `bound:PACKAGE` | What binds it in its Package or in a Package above it, `bound:Root` for Root: the digest of the Package's directory |
| `evidence:PATH` | A file the reviewer gave as evidence |

A Definition of the spec counts by its text. A Library Definition counts by its Library's digest in the lock, so that a newer binary changes nothing until the lock is written again; a Library counts as one, as a Go module does, so a change of one Kind of a committed Library puts the Definitions of all its Kinds on the list. Core's `Kind` and `Package` change only with the tool and are no inputs, and neither is a link to a Package. Line endings do not count in a text file's hash, so a checkout on another system matches.

**What binds** ([DEC-113](../../docs/decisions.md#dec-113-what-binds-is-an-input-of-what-it-binds)). What binds a Package is each item that lies in the Package itself and whose Kind declares `binds: true`, such as a `std/Rule` ([`spec.Binding`](../spec/schema.go)). The Package's **directory** maps each of them, by its ID, to the hash of its text and the texts of the parts it covers, and its digest is the SHA-256 of the directory in the order of the IDs. An item a file realizes carries `bound:P` for its Package and for each Package above it that has a directory, the binding things themselves included. What no file realizes is bound by nothing: an item whose Kind declares `realized: false`, such as a Goal, and a Kind ([`spec.Bound`](../spec/schema.go)).

This reaches what the [concept](../../docs/concept.md#how-status-works) puts at distance 0 and 1: a Definition that changed, one that links to a changed Definition or is of a changed Kind, one whose files changed, one whose registry rows gained or lost it, and one that a changed Definition binds. A changed Rule reaches itself and every item it binds, which is everything built when the Rule is Root's. A change of a row's reason, or of a file the ignore file lists, reaches nothing. The one difference, the Library counted as a whole, lists more, never less.

The **fingerprint** of an item is the first six characters of the hash of every input but the evidence, which the reviewer chooses after it read the item. `mark` also takes a longer prefix of the hash.

## The record file

`.markitect/review.sum` is part of the project's fixed layout and committed with it, like `go.sum`. It holds one line per Definition, sorted, each a JSON object with the Definition, the verdict, the inputs with their full hashes, the evidence as `path` or `path:line` and the reason on one line:

```text
{"definition":"members/loans/loan","verdict":"ok","inputs":{"bound:Root":"sha256:…","evidence:src/members/loans/loan.go":"sha256:…","file:src/members/loans/loan.go":"sha256:…","kind:std/Concept":"sha256:…","link:members/member":"sha256:…","self":"sha256:…"},"evidence":["src/members/loans/loan.go:3"]}
```

After the lines, sorted, come the directories the lines name, one per Package and digest: the Package, the digest, and each binding thing with its hash. `status` reads the directory a line names to say which binding Definition changed since, also once that Definition is verified again:

```text
{"bound":"Root","digest":"sha256:…","binds":{"architecture/rules/persistence":"sha256:…","loan-limit":"sha256:…"}}
```

A directory whose digest is not that of what it binds is a line markitect cannot read, like any other broken line.

The verdicts are `ok` (implemented, as the evidence shows), `fail` (not, and why), `skip` (verified as far as the project can show, and why) and `spec-problem` (the spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it). The record holds hashes, not text: an agent that wants what a Definition said before reads it from Git.

Three more fields are each left out when empty:

- `cases` and `covered`: the reviewer's count of the distinct cases, conditions and exceptions the Definition demands, and of those its evidence demonstrates, as `mark` took them. An `ok` and a `skip` always carry them, and `vet` reads them for `vet.risk`, from the line at the Definition's text as it is; of several such lines, as a merge can leave them until the next `mark` settles them, it judges the one with the highest risk, and none when one of them is a `fail` or a `spec-problem` ([DEC-106](../../docs/decisions.md#dec-106-the-reviewer-counts-and-vet-notes-thin-evidence)).
- `revisions`: how often the Definition's own text changed between two verdicts on it. `mark` takes the count of the Definition's lines and adds one when none of them has the new line's `self` hash, so a first verdict has none, a change of a link, a file or the evidence adds none, and so does a verdict on a gone Definition, which has no own text. Of several lines, as after a merge, it counts on from the most. `vet` reads it for `vet.hotspot`, as change history without Git.

The record has no format version: the fields are optional, so a line without them reads as no count and no revision, and its counting starts with the next verdict. A line written before what binds was an input has no `bound:` input, so its item, where something binds it, is listed once, with each Package that binds it as new, such as `bound:Root (new)`.

**Merging.** `init` and `update` write `.markitect/.gitattributes` with `review.sum merge=union`, which also joins the layout, so Git keeps the lines of both sides of a merge. Of several lines for one Definition, `status` takes the one whose inputs match the folder; two that match with different verdicts are a conflict to verify again. The directories of both sides are kept, a directory twice is the same line, and the next `mark` drops what no line names. A line markitect cannot read, such as a broken one written by hand, is an error until the owner fixes or removes it.

The record is a record, not a proof: a line written by hand shows in the diff, and [the hook](../guard/README.md#the-hook) keeps agents from writing one.

## status

`markitect status [--all] [--package P] [--slices [--size N]] [--slice A,B]` reads the folder and the record and changes nothing. It lists:

- every item with a changed input or no record, with which inputs changed: `changed: self, part:members/operations/lend/conditions/in-stock, file:src/x.go (new), evidence:tests/x_test.go (gone)`; an item with parts says how many it has and that `markitect doc` shows it whole;
- for a changed `bound:P`, in place of the Package, each binding Definition that differs between the directory the line names and the directory now, the item itself left out: `changed: bound:architecture/rules/persistence`, with `(new)` for one the line's directory lacks and `(gone)` for one the directory now lacks. Where the line has no `bound:P`, or the record no longer holds the directory it names, it names the Package: `bound:Root (new)`, `bound:Root`;
- every Definition recorded `fail` or `spec-problem` at its inputs as they are, with why; a changed one also says what was recorded before;
- every Definition gone from the spec that still has a line, with the files that line names: are they removed, or do they have a new basis?
- every conflict;
- the errors in the realization: a file neither listed nor ignored, a stale registry row, an unused Definition;
- the lines of the record it cannot read, each as a diagnostic `.markitect/review.sum:LINE: record.unreadable: …`.

Each item is printed as `DEFINITION@FINGERPRINT`, which the reviewer copies into `mark`, and names where to look: the Definition's file, its row files and the evidence of its last line. When every item is a `spec-problem` recorded at the inputs as they are, and nothing else is left, one more line says that only spec-problems are left, and the JSON has `for_the_owner`: they wait for the owner, and no builder settles them. A Definition recorded `skip` is a note. So is one line that says that the architect's questions for the owner wait in `.markitect/questions.md`, while that file exists; a list limited to a Package leaves it out. A spec with errors gives no list: there is none until it compiles. `--all` lists every Definition, to verify everything again.

`--package P` limits the list to the items whose Definition belongs to the Package of that name, `Root` for the one at the top, and not to a Package below it. A Definition in a folder that is no Package belongs to the nearest Package above it, and a Definition that is gone to the nearest Package above its path that exists. A strand of a [parallel run](../../docs/concept.md#10-drivers) asks for its part this way. The errors in the realization and the lines of the record it cannot read belong to no Package, so it leaves them out and says how many there are. It then exits 0 when nothing is to verify in that Package, whatever the others hold. A Package the spec does not have is a usage error.

It exits 0 when it lists nothing: then the project compiles and every Definition is verified at its current inputs. It exits 1 when it lists something, and so with `--all` whenever the spec has a Definition. Notes never change the exit code, a `skip` or the note on standard error that the project's harness comes from another markitect alike. It looks at no network: whether a newer binary exists is `update --check`'s question.

## mark

`markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]… [-m "WHY"] [--cases N --covered M]` records a verdict at the state the reviewer read:

```text
markitect mark members/limit@3f9a1c ok -e src/loans.go:42 -e tests/loans_test.go:88 --cases 4 --covered 4
markitect mark loans/renewal@81be07 fail -e src/renew.go:30 -m "allows a fourth renewal; the Definition says three"
```

- `ok` names at least one piece of evidence with `-e`; `fail`, `skip` and `spec-problem` say why with `-m`, kept on one line.
- `--cases N` and `--covered M` are the reviewer's counts: N cases, conditions and exceptions the Definition demands, M of them demonstrated by its evidence. An `ok` and a `skip` need both, so that the record says how much of a Definition its evidence shows, and `mark` refuses either without them; a `fail` and a `spec-problem` may carry them; a Definition gone from the spec has no cases, so its `ok` needs none. M needs N and is at most N; both are 0 or more. They go on the line as `cases` and `covered`.
- Evidence is `path` or `path:line`, also `path:line-line` and `path:line:col`. A trailing note in parentheses, such as a test name, is dropped, not stored; a note without them is refused, since `-m` holds the reason. Each file must be a file of the folder, named from the project folder; its hash becomes an input when the verdict is recorded.
- It refuses, records nothing and exits 1 when the inputs differ from the fingerprint, as when a builder edited a file beside the reviewer or the reviewer's own test run changed one; when evidence is not a file of the folder; when the Definition is unknown; when the spec has errors; or when the record has a line it cannot read. A command it cannot read exits 2.
- A gone Definition is marked `ok` when its files are removed or have a new basis, which removes its line, or `fail` when not; `skip` is refused.

It writes under a lock in the user's cache folder (`markitect/locks/`, one per project folder), reading the folder and the record afresh, and replaces the record as one whole file in one step, so parallel reviewers, such as one per Package, lose no line. It writes the file clean: one line per Definition, the matching one of several, both of a conflict, the closest when none matches, and one line for a gone Definition with every file its lines named. Each line it keeps counts the most revisions any line of its Definition counted, so a merge loses none. With the new line it writes the directories its `bound:` inputs name, and it keeps only the directories some line names, each once.

## The agents and skills

The four agents are files `init` writes, each holding its tasks: the builder (build, clean up, hunt bugs), the reviewer (verify, hunt bugs), the architect (write large parts of the spec) and the auditor (read the spec and report). The builder may read, edit, run the shell and start agents, which [the hook](../guard/README.md#the-hook) narrows to the reviewer; the reviewer and the auditor have no edit or agent tools, and the architect no agent tool. The builder and the reviewer default to `sonnet` at effort `high`, the architect and the auditor to `opus` at effort `medium`; `init`, `update` and `markitect config` write the model and effort the harness configuration gives each role into its file ([DEC-103](../../docs/decisions.md#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow), [strategies and configuration](../../docs/guides/project-workflow.md#strategies-and-configuration)). Everything else comes from the owner's session: its permission mode, settings, plugins and memory.

The skills are plain prompts with no injected command, so no exit code can abort them. `/markitect-build`, `/markitect-cleanup` and `/markitect-bughunt` start the builder in a worktree of its own on its task; it runs `status`, does its work, starts the reviewer, and repeats until `status` lists nothing it can settle or only what needs the owner, or three rounds left the list no shorter. The session then commits the work on the worktree's branch, has the reviewer verify once more, after a sequential run, what `status` still lists as changed or without a record there, and merges the branch into the owner's branch: build when `status` is empty there or says that only `spec-problem`s are left, cleanup and bughunt when it lists nothing new. `init` sets `worktree.baseRef` to `"head"` so the worktree starts from the owner's branch, and the tool accounts for Claude Code's worktrees under `.claude/worktrees/` itself. A build of a list of more than 40 items goes in slices ([DEC-112](../../docs/decisions.md#dec-112-a-long-list-is-built-in-slices)): `slices.go` cuts what a builder can settle by Package in the order of the dependencies, neighbours joined within the size, with a last slice for what is verified last and whatever is left; `status --slices` prints the cut, `status --slice A,B` lists one slice, whose command names the Packages before its own too, and the session takes the slices one after the other, each built, verified and merged before the next. `/markitect-review` starts the reviewer on what `status` lists, or on everything with `--all`; it changes no code, and the session commits the record. Each of these skills runs by its strategy: sequentially as described, or through the parallel workflow `.claude/workflows/markitect-parallel.js`, which works per Package in waves and reads `status --json`, whose `packages` give each Package's items and dependencies. `/markitect-audit` starts the auditor, and `/markitect-schedule` sets up routines through Claude Code's scheduling. The [project workflow](../../docs/guides/project-workflow.md#7-build-and-review) says how to use them.

## Tests

`record_test.go` plays the builder and the reviewer on plain folders without Git while `status` says what is to verify: what a change reaches, the Kind and the Library, the fingerprint and evidence, the verdicts, the counts of cases and the revisions, a gone Definition, merged lines and conflicts, an unreadable line, errors, and parallel reviewers. `bound_test.go` covers what binds: what a changed Rule reaches and how `status` names it, a record without `bound:` inputs, and the directories in the record. `lock_test.go` covers the lock.

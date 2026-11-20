# Rights and landing

What each role of a job may change, how the rebuilt tool `markitect-next` checks a step against it, and how it checks whether a branch lands ([DEC-127](../decisions.md#dec-127-one-landing-rule-and-a-spec-change-lands-with-its-realization), [DEC-132](../decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners)). [Rights come from the steps](../concept/jobs-and-runners.md#rights-come-from-the-steps) and [the landing rule](../concept/the-factory.md#the-landing-rule) in the concept say what they are for; this page says precisely what the tool checks. The examples are a small lending library.

## The write table

The write table ships with the tool. A step runs as one role, and its role may change only what the table gives it, between the project before the step and the project after it.

| Role | May change |
|---|---|
| `builder` | Artifacts: any file outside the spec folder, `.markitect/`, the counter-check folder and the control files, and the commit of a nested repository |
| `counter-checker` | Only files in the counter-check folder the settings name; nothing when they name none |
| `reviewer` | Only `.markitect/state.jsonl` and `.markitect/history.jsonl`: every line it adds to the history is a `verdict` event, and every row it adds or changes has the source `mark` and a `verdict` event of the step on its document, with its verdict and its `inputs`. No other file |
| `architect` | The spec folder and `.markitect/ignore`; `move` events, and rows with the source `move` that a move of the step [carried](#rows-and-events) from a row there was; in reports, only a place that a `move` event of the step names in its `reports` |
| `job` | The job itself, which the runner writes from an agent's answer: new reports whose source is a job or `intake`, holding only Source, Place, Finding, Occurred or risk, Violated property and Prevention, since rating, actions and closing are triage's; the Prevention section of reports; and files in `.markitect/samples/` |
| `triage` | In reports there were, only the Rating, Actions and Closing sections, a closing that names a duplicate included |
| `owner` | A branch from a conversation: everything, the control files and new reports with the owner as source included, but never a row with the source `mark`, never a `verdict` event, and a row with the source `move` only as a move of the step [carried](#rows-and-events) it |
| `reader`, `sampler`, `intake` | Nothing. The job writes the sampler's list of planted deviations and the intake's report |

**Intake** is a job that reads a signal from outside isolated and writes only a report of fixed form ([intake](../concept/reports-and-actions.md#intake)). Its agent writes nothing; the job writes the report from its answer, with `intake` as the source, so the role `intake` has no rights and the step `job` writes its report.

**Control files** are `.markitect/version`, `.markitect/spec`, `.markitect/settings.md`, `.markitect/.gitattributes`, `.markitect/rules.md`, which holds the rules learned of the owner's decisions, and the CI definition: the files the [settings' CI section](reports.md#settings) names, by default `.github/workflows/`. Only the owner changes them. Any other file of `.markitect/`, outside `reports/` and `samples/`, is the owner's too.

**The history is only appended to.** No role, the owner included, removes a line of it.

**Case.** A path whose first part is `.markitect` in any case, a control file in any case and a file the CI patterns take in any case count as what they name, so that `.Markitect/settings.md`, another file on a system that tells case apart and the same on one that does not, is never an artifact.

### Rows and events

A row binds a verdict, so a row stands only with the event that explains it.

- **A row with the source `mark`** needs a `verdict` event added in the same step on its document, with its verdict and with `inputs` equal to the [hash of its inputs](project-reference.md#the-history). An `ok` row has evidence, and a `fail` or `skip` row a reason, as `mark` writes them.
- **A `move` event** stands only for a move made. The step check replays the step's moves in their order on the spec before the step, as `move` computes them, and as [`compare`](project-reference.md#compare) replays them. Each event's document is one before it and its new path none, the two differ, the document is gone after the step unless a later move brought it back, and every document the event names as rewritten, its new path included, has after the step exactly what the replay makes of it, line ends read as [hashing](project-reference.md#hashing) reads them. So a move onto itself, or a copy presented as a move, is denied.
- **A row with the source `move`** is exactly what `move` makes of the rows of the state before, through the replayed moves ([move and the state](project-reference.md#move-and-the-state)). So a `fail` cannot turn into an `ok`, a new document gets no row, and a document edited beside a move keeps its row: the edit is an ordinary change of the spec, and `status` lists the document as changed. When one event of the step is no move made, no row with the source `move` stands.

**Narrowing.** A project narrows the table with lines `ROLE: PATTERN` in the settings' [Write table](reports.md#settings), written like the lines of a `.gitignore` file at the top of the project. A role with such lines may change only a file both the shipped table and its patterns allow: a pattern takes a file as `.gitignore` would exclude it, by itself or by a folder above it, and as there, a negation brings back no file whose folder a pattern takes. So `builder: src/*` with `builder: !src/vendor/` lets the builder change `src/loans.go` and not `src/vendor/lib.go`. A role without lines keeps the shipped table.

## The step check

`markitect-next status --step ROLE --before DIR [--target DIR]` checks what a step of ROLE changed. The project after the step is the one the command works in, and DIR is the project as it was before the step, a folder with `.markitect/version` that the runner lays out, as for [`compare`](project-reference.md#compare); the tool uses no Git. It changes nothing.

- **What it compares.** The files Git keeps of both projects: the files of the [walk](project-reference.md#the-walk) with what `.markitect/ignore` excludes, since that file says only what the project does not explain, and every file of `.markitect/`. What `.gitignore` excludes does not land and is not compared. A nested repository, one entry of the walk, counts by the commit its HEAD names, as Git records it: the hash in `.git/HEAD`, or the ref it names, resolved through a loose ref first and then `packed-refs`, following a ref that names another ref up to five times as Git does, in the Git folder a `.git` file names too, and in the common folder of a worktree. What is not committed in it does not land and does not count. A runner lays out the states before and after a step with a Git folder of their own for each nested repository, so that the step cannot change the commit the state before reads. A file is changed when it is new, gone, or its [hash](project-reference.md#hashing) differs, so CRLF and LF do not count. The state is compared by row, as the tool writes rows, the history by line, and a report by section.
- **Where the table is read from.** The control files, the spec folder, the counter-check folder, the CI definition and the narrowing are read from `--target` when it is given, and otherwise from `--before`; never from the project after the step, so a step cannot widen its own rights.
- **What it prints.** One line per change the table denies, `PATH: write.denied: ROLE may not …`, naming the row's document or the event for the state and the history, then `step ROLE: N changes the write table denies`. It exits 1 when there is any, 2 for an unknown role or settings with errors.

```text
docs/spec/lending/return.md: write.denied: builder may not change the spec
.markitect/state.jsonl: write.denied: architect may not write a row with source mark: lending/borrow.md
step builder: 2 changes the write table denies
```

## The landing check

`markitect-next status --target DIR` checks whether the branch the command works in lands on the target, the project DIR of the target branch. It takes `status` on both, each with its [waiting entries](reports.md#waiting-on-a-report), and compares their entries by document, or file, and kind: changed, suspect, unexplained and waiting. Fingerprints do not count. Every entry of the branch that the target does not show keeps it from landing, apart from documents waiting on the owner with no change their reports do not excuse, which land unbuilt and visibly waiting. It prints one line per such entry, `adds KIND: DOCUMENT: CHANGES`, then `landing: N entries the target does not show`, and exits 1 when there is any. While `check` finds errors on the branch, it gives no answer and exits 1.

```text
adds changed: roles/member.md: self
adds suspect: lending/borrow.md: link:roles/member.md
landing: 2 entries the target does not show; the tests and the counter-checks are the runner's to run
```

The tests and the counter-checks are the runner's part of the landing rule: the tool runs nothing. `--step` and `--target` combine: the command then checks the step against the target's table, and the landing. With `--json`, `status` with either flag prints `step`, the role or empty, `errors`, `violations`, with `file`, `line`, 0, `code` and `message`, `landing`, whether it checked the landing, and `added`, with `kind`, `doc` and `changes`.

## How firm it is

In CI the runner is a script no agent can change, which lays out the states out of every agent's reach: there the checks are a fact, and a required check a job cannot skip. In a session or a scheduled task the runner is Claude itself, which could lay out a state that is not the one before the step: there the checks make shortcuts visible and prove nothing ([where the check is a fact](../concept/jobs-and-runners.md#rights-come-from-the-steps)).

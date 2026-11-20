# Reports, settings and control samples

The files of fixed form that the rebuilt tool `markitect-next` reads beside the spec, the state and the history: the reports in `.markitect/reports/`, the settings in `.markitect/settings.md` and the results of control samples in `.markitect/samples/` ([DEC-128](../decisions.md#dec-128-reports-and-actions-hold-everything-that-needs-a-change), [DEC-132](../decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners)). Each is a document of the language, of a type the tool owns ([library](library.md#the-types-the-tool-owns)), and holds values of fixed form that `check` checks and the tool reads: the waiting entries of `status`, the counter-check folder of `mark`, the figures of the factory and the reports of `compare`. [Reports and actions](../concept/reports-and-actions.md), [trust](../concept/trust.md) and [jobs and runners](../concept/jobs-and-runners.md#the-project) in the concept say what they are for; this page says precisely what the tool reads. The examples are a small lending library.

## Values

A value stands on the lines of its section as written, not as Markdown renders them, so that `src/__init__.py` or a pattern such as `src/**/*.go` keeps every character. The tool reads each line of a section without HTML comments, blank lines and the white space around it, drops a leading list marker, `- `, `* ` or `+ `, and the backticks of a line written wholly as code. Where a section holds one value, it is its first line, and what follows is free text. Keywords are written in lowercase, as below. A NAME, of a report, a sample, a job or a duplicate, is lowercase letters and digits in words joined by hyphens, such as `late-fee`, and a day is `YYYY-MM-DD`.

## Reports

A report is a file `.markitect/reports/NAME.md`, one per report. Its sections are those of the report type: Source, Place, Finding, Occurred or risk and Violated property are required, Rating, Actions, Prevention and Closing optional. Finding and Prevention are free text.

| Section | Value |
|---|---|
| Source | `owner`, `intake` or `job JOB ROLE`, then `, YYYY-MM-DD`, the day it was raised: `job build reviewer, 2026-11-20`. ROLE is one of the [roles](#settings) |
| Place | One path below the project folder, of a spec document, an artifact or a folder, such as `docs/spec/lending/return.md`, or a Markdown link whose target, relative to the report, leads there: `[Return](../../docs/spec/lending/return.md)`. It may not lie in `.markitect/` or a `.git` folder |
| Occurred or risk | `occurred` or `risk` |
| Violated property | `necessary`, `complete`, `consistent`, `unambiguous and verifiable` or `structured`, the five properties of a good spec; `correct`, when the spec says the wrong thing; or `none`, when the spec holds and an artifact deviates from it |
| Rating | `impact LEVEL, likelihood LEVEL`, each LEVEL `low`, `medium` or `high` |
| Actions | A line `branch NAME` for each branch that answers it, then optionally a line `escalated`, followed by the proposals for the owner as text, at least one line |
| Closing | `YYYY-MM-DD: done`, `YYYY-MM-DD: duplicate of NAME` or `YYYY-MM-DD: nothing to do`, with an optional period, then the reason on the lines below. The day is not before the day it was raised |

A report is **open** while it has no Closing section.

**A place that is gone.** A build may delete the artifact a report names, so a place need not exist. When the place of an open report is gone, `status` says so in a note, `note: report NAME: place gone: PLACE`, which changes neither its list nor its exit code. The place names its path in the case it is written, on every system: a place in another case is gone.

**A place that moves.** `move` rewrites the place of every report that names the moved document, closed reports included: a path takes the new path, a link the new path relative to the report, with its `#section`. The rewritten reports are listed with the files `move` changed and in the `reports` list of its event in the [history](project-reference.md#the-history). A path is rewritten where its line holds the clean path, so `./docs/spec/lending/return.md` becomes `./docs/spec/loans/return.md`. A path written otherwise, such as `docs/spec/lending/../lending/return.md`, a path whose new path holds white space, and a link whose new path would need `<` and `>`, stay as they are and then name a place that is gone.

```markdown
# The fee for a late book

## Source
job read reader, 2026-11-18

## Place
[Return](../../docs/spec/lending/return.md)

## Finding
The spec says when a book comes back, and not what a late one costs.

## Occurred or risk
risk

## Violated property
complete

## Actions
- escalated
- Charge a fee per day late.
- Charge one fee, whatever the delay.
```

## Waiting on a report

An open report whose violated property is not `none` and whose place is a document of the spec makes that document **waiting on report NAME** in [`status`](project-reference.md#status). It is then listed only as waiting, not as changed or suspect, and a `skip` note on it is left out; its neighbours are listed as before.

- Waiting excuses only the need to build the document: having no row, or a change of its own text, its type or its links. A change of an artifact, a counter-check, a file of the evidence or a linked file of the spec folder, a `fail` and a conflict are listed in the entry and counted as usual.
- When every open report it waits on says `escalated` in its actions, the document **waits on the owner**: it is shown, and does not make `status` exit 1 unless it has such a change.
- Otherwise the waiting entry counts like any entry.
- A report whose property is `none` blocks nothing: its action repairs the artifact, and the document is listed as the state says.

```text
waiting: lending/return.md@5d0e9b11: report late-fee (escalated: waits on the owner)
0 changed, 0 suspect, 0 unexplained, 1 waiting, 1 of them on the owner
```

A waiting entry reads `waiting: DOCUMENT@FINGERPRINT: report NAME`, with every report it waits on, ` (escalated: waits on the owner)` when it waits on the owner, and `: CHANGES` with every change, as a changed or suspect entry names them. The summary adds `, N waiting` and `, N of them on the owner` when there are any. With `--json`, `status` lists `waiting` with `doc`, `fingerprint`, `reports`, `escalated`, `changes`, what it would list for the document otherwise, empty when nothing moved, and `counts`, whether the entry makes `status` exit 1.

## Settings

The settings are `.markitect/settings.md`, of the settings type. Every section is optional, and a project without the file has no settings. A section the type does not declare is the project's own and is not read.

| Section | Value |
|---|---|
| Models | A line `ROLE: MODEL` per role, MODEL one word: the model the runner gives the role |
| Counter-checks | One folder below the project folder, outside the spec folder, `.markitect/` and any `.git` folder. It need not exist yet |
| Cadence | A line `JOB: TEXT` per job, the text read by the runners |
| Write table | Lines `ROLE: PATTERN`, ROLE one of the roles, `job` or `owner`, each pattern written like a line of a `.gitignore` file, which narrow what the role may write ([narrowing](rights.md#the-write-table)). The tool checks that each parses: it names something, every `[` is closed, every class such as `[:digit:]` is one Git knows, and it does not end in a lone backslash |
| CI | The files of the CI definition, a [control file](rights.md#the-write-table): a pattern per line, written like a line of a `.gitignore` file, checked as the write table's are. Without the section it is `.github/workflows/` |
| Tests | One command line that runs the project's tests and its counter-checks, such as `go test ./...`. A runner runs it in the project folder for the [landing rule](rights.md#the-landing-check), and the reviewer runs it as evidence; it passes when it exits 0. Without the section a runner has nothing to run, and lands nothing |

The roles are `architect`, `builder`, `reviewer`, `counter-checker`, `sampler`, `reader`, `triage` and `intake`. A role or a job named twice in one section is an error.

`init` writes these settings:

```markdown
# Settings

## Models

- architect: opus
- builder: sonnet
- reviewer: sonnet
- counter-checker: sonnet
- sampler: sonnet
- reader: sonnet
- triage: sonnet
- intake: sonnet

## Counter-checks

tests/counter-checks
```

Once the settings name a counter-check folder, [`mark`](project-reference.md#mark) takes a counter-check, `--check PATH`, only from inside it.

`markitect-next settings` prints the settings, so that a runner or an agent reads them without reading Markdown: the models, the counter-check folder, the tests command, the cadence, the write table and the CI definition, each `(none)` when the settings name nothing, after what `check` finds in the settings file. It exits 1 while `check` finds errors there, and says `no settings` without the file. With `--json` it prints `present`, whether the file is there, `errors`, with `file`, `line`, `code` and `message`, `models` and `cadence`, objects by role and by job, `counter_checks` and `tests`, empty when not named, `write_table`, with `role` and `pattern`, and `ci`. A value that is not of its form is left out.

```text
Models:
  builder: sonnet
  reviewer: sonnet
Counter-checks: tests/counter-checks
Tests: go test ./...
Cadence: (none)
Write table: (none)
CI: .github/workflows/
```

## Control samples

The result of a control sample is a file `.markitect/samples/NAME.md`, of the sample type, with four required sections.

| Section | Value |
|---|---|
| Source | `job control-sample sampler, YYYY-MM-DD` |
| Copy | The commit or branch of the throwaway copy, as text |
| Planted | A line `PATH: DEVIATION` per planted deviation, PATH a document below the spec folder, as `status` names it. A document may have several |
| Verdicts | A line `PATH: VERDICT` per document the reviewer judged on the copy, VERDICT `ok`, `fail` or `skip`, one line per document |

Every planted document has a line in Verdicts. The planted documents need not be in the spec now, since a sample records a copy of the past.

```markdown
# Sample of 19 November

## Source
job control-sample sampler, 2026-11-19

## Copy
branch sample-2026-11-19

## Planted
- lending/borrow.md: lends a sixth book
- lending/return.md: takes a book back after four weeks

## Verdicts
- lending/borrow.md: fail
- lending/return.md: ok
```

## What check reports

`check` checks every file of `.markitect/reports/` and `.markitect/samples/` and the settings beside the spec. A file of either folder whose name starts with a dot, such as `.gitkeep`, is passed over. Each document is checked against its type as a document of the spec is ([language reference](language-reference.md#what-check-reports)): `title` for its title, `section.missing` at its title for a required section it lacks, and `section.empty` for a section it has that holds no text. It needs no type line, and links in it are not checked. The diagnostics name the file below the project folder, such as `.markitect/reports/late-fee.md:13: report.form: …`.

| Code | Reported when |
|---|---|
| `report.form` | A file of `.markitect/reports/` is no `NAME.md` with a NAME as above, or a value of a report is not of its form |
| `report.duplicate` | A closing says `duplicate of NAME`, and no other report has that name |
| `settings.form` | A value of the settings is not of its form |
| `sample.form` | A file of `.markitect/samples/` is no `NAME.md`, or a value of a sample is not of its form |
| `sample.verdict` | A planted document has no line in Verdicts |

`check` prints its summary as `N errors in N documents`, and adds `, N reports and N samples` when there are any; with `--json` it prints `reports` and `samples`, their numbers, and `settings`, whether the file is there. While `check` finds errors here, `status` gives no list and `mark` writes nothing, as for errors of the spec, since both read the reports and the settings. `move` reads neither and refuses only for the spec and the state.

## The figures of the factory

`markitect-next figures` prints, after the [change signals](signals.md#the-change-signals), the figures of how the factory runs ([the tool](../concept/the-tool.md#the-figures)), each with the project's own reference beside it. They read no clock: the same files give the same figures anywhere. A value that is not of its form counts for nothing, and what `check` finds in the files is printed first as a note.

| Figure | Name | Lists |
|---|---|---|
| Hit rate | `hit-rate` | Per sample, by name, the planted documents that got `fail`, out of all planted, and the documents missed. The reference is the rate over all samples |
| Contradictions | `contradictions` | Every verdict that differs from the latest earlier verdict on the same document at the same `inputs` ([the history](project-reference.md#the-history)). The reference is the number of such re-verdicts, out of the verdicts that name their inputs; verdict events without `inputs` do not count |
| Open reports | `open-reports` | Each day on which reports were raised or closed, by the days in Source and Closing: how many were raised, closed and open at its end |
| Findings per area | `findings-per-area` | The reports and the open ones per area: the top folder of a place below the spec folder, `(spec)` for a document at its top or the spec folder itself, or `artifacts` for a place outside it |
| Recurring reports | `recurring-reports` | Each report with its duplicates, and each place named by several reports |
| Jobs stopped without progress | `stopped-jobs` | Not computed: no report says yet that a job stopped without progress |

```text
### hit-rate (1)

reference: 1 of 2 planted documents (50%) over 1 sample
first (2026-11-19): 1 of 2 found; missed lending/return.md
```

With `--json`, `figures` adds `samples` with `name`, `day`, `planted`, `hit` and `missed`, `planted` and `hit` over all, `reverdicts`, `contradictions` with `doc`, `was`, `now` and `time`, `reports` with `total`, `open`, `days` with `day`, `raised`, `closed` and `open`, `areas` with `area`, `reports` and `open`, and `recurring` with `kind`, `duplicates` or `place`, `place`, `reports` and `detail`, `stopped_jobs`, always null, and `notes`; the references join `reference`.

## Reports in a comparison

[`compare`](project-reference.md#compare) names, under **Reports**, the reports by name that are new, closed, changed or gone between the two projects. A report open in OLD with a closing in NEW is closed, with the first line of its closing; any other change of its text is changed. A closed or changed report shows the diff of its text.

```text
## Reports (2)

closed: late-fee: 2026-11-20: done
--- .markitect/reports/late-fee.md
+++ .markitect/reports/late-fee.md
@@ -16,3 +16,7 @@
…
new: new-wish
```

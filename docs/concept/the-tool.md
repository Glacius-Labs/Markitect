# The tool

**The `markitect` program computes and checks; a runner acts** ([DEC-096](../decisions.md#dec-096-the-tool-computes-and-checks-a-runner-acts)). It checks the spec, computes the state and what is to verify, binds verdicts and computes the factory's figures. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project. It uses no Git ([DEC-097](../decisions.md#dec-097-markitect-uses-no-git)): it reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, and the runner makes every Git step. So it does not know where it runs, and gives the same answer in a session and in CI.

## Commands

| Command | Does |
|---|---|
| `init` | Sets up a project: `.markitect/`, with the version and the path of the spec, and the spec folder, by default `docs/spec/`. It writes nothing else ([§12](jobs-and-runners.md#the-project)) |
| `apply TEMPLATE [TARGET]` | Copies a template from the library into the spec, with every package it needs; a package the spec has is never overwritten, and a path that exists stops it ([§3](the-language.md#packages-and-templates)) |
| `new TYPE PATH [TITLE]` | Creates a document of a type, with its title, the type line and the type's required sections |
| `move OLD NEW` | Moves or renames a document, rewrites every link to it and the links inside it, carries its row in the state and records the move ([§6](the-state.md#moving-a-document)) |
| `check` | Checks the spec: titles, types, sections and links ([§3](the-language.md#what-check-catches)). It checks the form of reports and of the results of control samples against their types, and the settings |
| `show DOCUMENT` | A document with its type, its sections, what it links to and what links to it with the section the link stands in, its artifacts, its counter-checks and its verdict; for a type, its documents |
| `table TYPE` | Every document of a type as a table, one column per section |
| `topics [TOPIC]` | A document, a type or a folder as one text with everything that belongs to it, and the signals that say where reading pays off ([§5](is-the-spec-good.md#the-spec-itself)) |
| `status` | What is changed, suspect, unexplained or waiting on a report ([§6](the-state.md#what-status-lists)). Given the state of the target branch, it applies the landing rule; given the states before and after a step and the step's role, it checks the write table ([§12](jobs-and-runners.md#rights-come-from-the-steps)) |
| `mark DOCUMENT@FINGERPRINT VERDICT` | Binds `ok`, `fail` or `skip`, with the artifacts and the evidence or the reason, to the state `status` showed ([§6](the-state.md#binding-a-verdict)) |
| `diff OLD NEW` | Compares two states, each a folder with a spec, a state and reports: what is new, gone, changed or suspect, with the old and the new text ([§10](overview.md#comparing-two-states)) |
| `figures` | The factory's figures ([below](#the-figures)) |
| `version` | The tool's version, and whether it is the one the project pins |
| `licenses` | The third-party notices |

**What the tool writes.** Of the state and the history it writes only what it binds: `mark` a verdict row and its history entry, `move` the moved row and its history entry. `init`, `apply`, `new` and `move` write documents on request. What a job reports, reports and the results of control samples, are files of fixed form that the job writes and `check` checks ([DEC-132](../decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners)). There is no command to report, to start or end a run, or to steer anything.

## The figures

What the briefing shows of how the factory runs is computed by the tool, so that it is no agent's claim ([§10](overview.md#briefing)):

- the hit rate of control samples, from the list of planted deviations and the reviewer's verdicts on the copy;
- the rate at which spot checks contradict earlier verdicts on unchanged documents, from the history;
- open reports over time, findings per area, recurring reports and jobs that stopped without progress, from the reports;
- the change signals of the spec, from the history ([§5](is-the-spec-good.md#its-changes)).

A failing counter-check is a `fail` and a report, and counts there; passed counter-checks are no figure of their own.

## Output and exit codes

A diagnostic is one line, `file:line: code: message`. With `--json`, a command prints one document that starts with a `format_version` and a `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run.

# The tool

**The `markitect` program computes and checks; whoever starts an agent acts** ([DEC-142](../decisions.md#dec-142-the-tool-computes-and-checks-a-runner-acts)). It checks the spec and the documents of the work, computes the state and what is to verify, and binds verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project. It uses no Git ([DEC-097](../decisions.md#dec-097-markitect-uses-no-git)): it reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, and agents and people make every Git step. So it does not know where it runs, and gives the same answer in a session and in CI.

The tool informs; it steers nothing ([DEC-148](../decisions.md#dec-148-markitect-is-a-tool-a-spec-with-templates-and-skills)). The way of working stands in the spec and the skills realize it ([§8](the-circle.md)); nothing of it is enforced through the tool, and `status` is never a gate.

## Commands

The commands are flat, as Git's are. The help groups them.

| Group | Command | Does |
|---|---|---|
| Look | `show DOCUMENT` | A document with its type, its sections, what it links to and what links to it with the section the link stands in, its artifacts and its verdict; for a type, its documents |
| | `table [TYPE]` | Every document of a type as a table, one column per section; without a type, every document with its verdict and artifacts |
| | `topic [TOPIC]` | The topic view: a document, a type or a folder as one text with everything that belongs to it, and the signals that say where reading pays off; without a topic, the signals of the whole spec ([§5](is-the-spec-good.md#the-spec-itself)) and the change signals of its history ([§5](is-the-spec-good.md#its-changes)) |
| | `questions` | The competency questions of the packages and templates the spec uses ([§5](is-the-spec-good.md#the-spec-itself)) |
| | `compare OLD [NEW]` | Compares two states, each a folder with a spec, a state and the work: what is new, gone, changed or suspect, with the old and the new text ([§10](overview.md#comparing-two-states)) |
| Check | `check` | Checks the spec, titles, types, sections and links ([§3](the-language.md#what-check-catches)), and the documents of the work in `.markitect/` against their types |
| | `status` | What is changed, suspect or unexplained ([§6](the-state.md#what-status-lists)). It only shows |
| Change the spec | `new TYPE PATH [TITLE]` | Creates a document of a type, with its title, the type line and the type's required sections |
| | `apply TEMPLATE [TARGET]` | Copies a template from the library into the spec, with every package it needs; a package the spec has is never overwritten, and a path that exists stops it ([§3](the-language.md#packages-and-templates)) |
| | `move OLD NEW` | Moves or renames a document, rewrites every link to it, in the spec and in the work, and the links inside it, carries its row in the state and records the move ([§6](the-state.md#moving-a-document)) |
| Confirm | `confirm DOCUMENT@FINGERPRINT VERDICT` | Binds `ok`, `fail` or `skip`, with the artifacts and the evidence or the reason, to the state `status` showed ([§6](the-state.md#binding-a-verdict)) |
| Set up | `init` | Sets up a project: `.markitect/` with its empty folders for the work, the spec with the four templates, and the skills of the standard ([§12](the-project.md)) |
| | `version` | The tool's version, and whether it is the one the project pins |
| | `licenses` | The third-party notices |

**What the tool writes.** Of the state and the history it writes only what it binds: `confirm` a verdict row and its history entry, `move` the moved row and its history entry. `init`, `apply`, `new` and `move` write documents on request. Plan items, findings, reports and briefings are files that agents write and `check` checks. There is no command to report, to start or end a run, to land a branch or to steer anything, and no settings file.

## Output and exit codes

A diagnostic is one line, `file:line: code: message`. With `--json`, a command prints one document that starts with a `format_version` and a `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run.

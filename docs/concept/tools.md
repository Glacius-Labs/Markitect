# The tools

## Commands on the spec

None of these starts an agent or uses Git. `--repo DIR` names a folder of the project, whose root is the nearest folder with `.markitect/manifest.yaml` (default: the current folder), and `--json` prints JSON for scripts.

| Command | Does |
|---|---|
| `install` | Writes the personal skill, once per machine ([§9](harness.md)). Optional: `init` says the same and gives a project its own skills |
| `init` | Sets up a project: `.markitect/`, the first page of the spec, the harness, `.markitect/.gitattributes`, a line in `.gitignore` for Claude Code's worktrees, and `AGENTS.md` and `CLAUDE.md` if they are missing. An empty folder is enough. It makes no repository and commits nothing, and sets `worktree.baseRef` in the project's settings when they name no base. Run again, it refreshes the harness and never touches spec pages |
| `tidy` | Brings the lock in line with the manifest, as `go mod tidy` does, and first names in the manifest the version of a standard Library this binary ships, where it names another. The lock holds a fingerprint of each Library, as `go.sum` does for modules; `check` only reads it, refuses a Library that no longer matches and says to run `tidy` |
| `check` | Compiles the project and fails on any error ([below](#what-check-refuses)). `--spec` looks only at the spec itself |
| `list [--deps]` | The Packages with their Definitions. With `--deps`, what each Package depends on and the links behind it, with its coupling, abstractness and distance |
| `doc TARGET` | A Definition with its text, what it links to and what links to it, the files that realize it and its record; with its parts and the parts of other things it links; or a Package with the text of its anchor and its public Definitions apart from its internal ones. `doc Root` shows Root |
| `vet [--strict]` | Measures the spec's design ([§4](is-the-spec-good.md)). It warns, and exits 1 only with `--strict` |
| `update` | Installs the newest release over the binary, then lets the new binary bring what an earlier build wrote in line: the personal skill if there is one, and in a project the harness and the lock ([keep it current](../guides/project-workflow.md#8-keep-it-current)). It commits nothing and says to read the lock before committing it. `--check` changes nothing and exits 1 if anything is out of date; `--files` leaves the binary alone; `--force` replaces a development build |
| `version` | The build, and whether the personal skill and the project match it |
| `config KEY`, `config [--user] KEY VALUE` | Reads the value in force, or sets one of the harness configuration ([§9](harness.md)): the agents' model and effort per role, the strategy per skill and custom workflows, in the project's `.markitect/harness.yaml` or with `--user` in the user's; it rewrites the agents' files. `--list` shows each value and where it comes from, `--unset KEY` removes one |
| `licenses` | The third-party notices |

**Output and exit codes** follow Go's tools. A diagnostic is one line, `file:line: code: message`; `check` prints only diagnostics and one summary line, and `vet` only diagnostics and one line of notes. With `--json`, `check`, `vet`, `list`, `doc`, `status`, `mark` and `config --list` print one document that starts with `format_version` and `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run; a note never changes the exit code. [The spec language guide](../guides/spec-language.md#diagnostics-json-and-exit-codes) owns the details.

**`update`** does what `claude update` does ([DEC-095](../decisions.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)). It reads the newest tag from the redirect of GitHub's releases page, downloads the binary for the system, checks it against the checksums file of that release and replaces the running executable: by renaming over it on Linux and macOS, and on Windows by moving the old file aside first. The new binary then finishes the update, since only it knows the files it ships. This is the only place where Markitect uses the network, and only when the owner runs it; without a network it says so and still refreshes the files. A development build older than the newest release is not replaced unless `--force` says so. A binary the plugin manages is left to the plugin. Commands that read a project print one note when its skills, agents and hooks are older than the binary.

## What `check` refuses

`check` passes or fails, with no warnings: a project either compiles or it doesn't. Its errors come in two groups, and it prints them apart. One case passes: when the registry lists no file and the only errors are Definitions nothing realizes, it says that the spec compiles and nothing is built yet. `status` makes no such exception.

| Group | Errors | Who settles them |
|---|---|---|
| **The spec** | A Kind or link that does not resolve, a Package cycle, a link into a child's Definitions or to an internal Definition from outside, a Library that does not match the lock | The owner, in the spec |
| **The realization** | A file neither listed nor ignored, a registry entry for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes | The builder, in the work |

So `check` goes red the moment the owner deletes a Definition or adds one nothing uses yet, and stays red until the work has realized the change: spec and code don't match yet. `--spec` answers the narrower question "is the spec itself valid?", which is what a session on the spec asks. `status` gives no list while the spec has errors; errors in the realization are on its list.

## How `status` works

Like `go test`, which does not run a test again when nothing it depends on changed, Markitect keeps for each item, a Definition with its parts, the state at which a reviewer last verified it ([DEC-099](../decisions.md#dec-099-a-record-of-what-was-verified), [the inputs](../../internal/run/README.md#the-inputs)). That state is the set of everything the verification depended on, its **inputs**, each by its hash: the Definition's own text and that of each of its parts; their Kinds and the Definitions they link to, a Library Definition by its Library's digest in the lock, so that a newer binary changes nothing; the files its registry rows name; what binds it without a link, Package by Package; and the files the reviewer gave as evidence. **What is to verify** is every Definition with a changed input or no record, computed from the folder and the record alone.

| Distance | Reached | On the list |
|---|---|---|
| 0 | Definitions that changed, Definitions whose files or evidence changed, and Definitions a registry row gained or lost | Yes |
| 1 | Definitions that link to a changed one, are of a changed Kind, or are bound by a changed one | Yes |
| 2 and more | Everything further along the links | No |

**What binds is an input of what it binds** ([DEC-113](../decisions.md#dec-113-what-binds-is-an-input-of-what-it-binds)). A changed Rule, Decision or Constraint reaches every Definition of its Package and of the Packages below it that a file realizes, and `status` names it on each of them: `changed: bound:architecture/rules/persistence`. The record keeps what bound each item when it was verified, so the name stays after the Rule itself is verified again, until each item it binds is verified against it. A Rule of Root binds everything that is built, so its change lists everything; a Rule that holds for one area lives in that area's Package. What no file realizes, such as a goal, and a Kind are bound by nothing. A Library counts as one, so a change of one Kind of a committed Library lists the Definitions of all its Kinds: more, never less. Three more kinds of item stay on the list until they are verified again: a Definition recorded `fail` or `spec-problem`, a Definition gone from the spec, with the files that realized it, and an error in the realization.

`status` doesn't follow the code's own imports ([roadmap](../roadmap.md#later)). A full review now and then, `/markitect-review --all`, is the safety net. On the change steps of Makerspace, a study project, one step of links reached 33 of the 36 existing files that truly changed, and deeper reach 35 ([measurement](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/impact-reach-20261103.md)).

## Commands of the record

These are for whoever drives the agents, and for CI. They start no agent, keep nothing running, steer nothing and use no Git. [The record](../../internal/run/README.md) owns how they behave.

| Command | Called by | Does |
|---|---|---|
| `markitect status` | The builder, the reviewer, the skills, the owner and CI | What is to verify, with which inputs changed, where to look and each item's fingerprint. `--all` lists every Definition, and `--package P` only the items of one Package, for a strand of a parallel run. `--slices` prints the slices a build of the list goes in, and `--slice` lists one. Exit 1 when it lists something; notes, such as a `skip` or an outdated harness, never change the exit code |
| `markitect mark` | The reviewer | Records `ok`, `fail`, `skip` or `spec-problem` on one Definition at the fingerprint `status` printed, as in `markitect mark members/limit@3f9a1c ok -e src/loans.go:42 --cases 4 --covered 4`, with its evidence and the reviewer's counts, or why ([§6](record.md)). An `ok` and a `skip` carry the reviewer's counts, `--cases N --covered M` |
| `markitect run` | The skills, the workflows and the routines | Tells the history that a run starts, with its kind, strategy and trigger, that a slice or a round starts, and that the run ends, with how; the tool adds the time, the list and each role's model and effort at the start, the items of the slice, and what the list says at the end ([§6](record.md)) |
| `markitect inbox` | The skills, the owner and agents | Lists what waits, opens an item with `add` and closes one with `close` ([§6](record.md)) |
| `markitect hook` | Hooks only | The check of [§9](harness.md); what it refuses goes into the history |

**The folder.** Every command reads the project folder by walking it, without what the project's `.gitignore` files exclude, read as plain text ([the folder](../../internal/run/README.md#the-folder)). So ignored build output does not count, and a folder needs no repository. Files that Markitect's own ignore file lists are walked like any other; they need no registry row and are no input.

**The ignore file** is `.markitect/ignore`, written like a `.gitignore`: one pattern per line, `#` for a comment, and no `!`. The tool accounts for what `init` installs by itself, so a new project has no ignore file.

**There are no required checks** ([DEC-099](../decisions.md#dec-099-a-record-of-what-was-verified)). A project that wants tests states them in the spec, with the command that runs them, and the reviewer runs them as evidence. So no proof in the work comes from outside an agent, apart from `check` and the record's inputs. If a Definition is verified that a test would have caught, the checks come back. The agents' `model` and `effort` are in their agent files, written from the harness configuration.

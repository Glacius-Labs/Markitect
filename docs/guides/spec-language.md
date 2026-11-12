# Spec language and commands

This guide is the canonical description of the spec language of the [concept](../concept.md#the-spec) ([DEC-080](../decisions.md#dec-080-the-spec-is-typed-documentation-in-markdown), [DEC-006](../decisions.md#dec-006-every-file-is-accounted-for)) as `markitect` implements it, and of the commands `markitect` has. None of them starts an agent or uses Git. Where the concept leaves a detail open, the [details](#details) at the end say what holds. [What makes a good spec](good-spec.md) says how to use the language well.

## The project's Markitect folder

```text
.markitect/
  manifest.yaml        the Libraries the project uses, with their versions
  lock.yaml            the digest of each Library, written by markitect tidy
  spec/                the spec; this folder is the Root Package
    project.md         the project anchor (std/Project)
    members/           a Package, because it holds its anchor
      members.md       the anchor of members (kind Package): ID members/members
      member.md        a Definition: ID members/member
      rules/           a folder: a namespace, no Package
        loan-limit.md  a Definition of members: ID members/rules/loan-limit
  registry.md          Root's registry page
  registry/
    members.md         the registry page of Package members
    members/loans.md   the registry page of Package members/loans
  ignore               files that deliberately belong to no Package, if any
  questions.md         the architect's questions that wait for the owner, if any
  libraries/<name>/    a Library that does not ship with markitect
  review.sum           the record of what was verified, written by markitect mark
  harness.yaml         the project's harness configuration, written by markitect config, if any
  .gitattributes       review.sum merge=union, written by init and update
```

Every file below `.markitect/` is accounted for by the compiler itself and needs no registry entry. A file there that has no place in this layout is an error (`project.file`). `questions.md` is plain Markdown that nothing compiles: the owner's session keeps there what the architect has to ask and the owner cannot answer at once, one numbered point per question, and removes the file with its last point ([DEC-100](../decisions.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). A file in the registry folder that is not a Markdown page is an error too (`registry.page`), and so is a file the walk finds but the compiler cannot read (`project.read`). `harness.yaml` sets the agents' model and effort, the skills' strategies and custom workflows; a file there that `markitect config` would not take is an error (`harness.syntax`) ([strategies and configuration](project-workflow.md#strategies-and-configuration)). Text files under `.markitect/` are read with LF line ends, so a Windows checkout with CRLF compiles, locks and compares like the commit it came from.

The project folder is the nearest folder, the one given or above, with `.markitect/manifest.yaml`. The compiler sees its files without what its `.gitignore` files exclude, read as text; it uses no Git ([the folder](../../internal/run/README.md#the-folder)).

## Definitions

Everything in a spec is a Definition: one Markdown file with YAML front matter.

```markdown
---
kind: std/Feature
---
# Create reservation

A [[members/member|member]] reserves a [[equipment/machine|machine]] for a slot.
```

- The front matter holds `kind` and, where needed, `internal: true`. Nothing else; another key is an error (`definition.front_matter`).
- The file name without `.md` is the Definition's name, and its path below `spec/` is its ID: `members/member.md` is `members/member`. Names start with a letter or digit and use only letters, digits, `-` and `_` (`spec.name`).
- The body starts with one level-1 heading, the title; no other level-1 heading follows (`definition.title`).
- The rest is documentation. Exact values such as wire names and error codes are written in backticks.
- The spec folder holds only `.md` files and folders (`spec.file`). Two entries of one folder whose names differ only in case, or a file and a folder of the same name (`member.md` and `member/`), collide (`spec.collision`): a link could not tell them apart.

## Packages and folders

A folder is a namespace, as in C#: it gives the Definitions in it their path and means nothing else. A project orders its Definitions in folders as it reads best, and Markitect prescribes no layout.

A Package is a module of the project, as a project is in C#: the unit of dependencies, of visibility and of file ownership. The spec folder itself is the Root Package. Below it, a folder is a Package when it holds its anchor, a Definition of kind `Package` that carries the folder's name: `members/members.md` makes `members` a Package ([DEC-108](../decisions.md#dec-108-a-folder-is-a-namespace-a-package-is-declared-by-its-anchor)).

- **The anchor says what the Package is:** what it is responsible for and what it keeps to itself. It is verified like every Definition, a file realizes it or a Definition links to it (`definition.unused`), and a link to the Package does not link it. A Definition of kind `Package` that does not carry its folder's name, or lies at the top of the spec folder, is an error (`spec.package`). Root needs none: what the project is, the project's anchor `project.md` says by convention, and the tool gives that file no meaning of its own.
- **What lies in a folder that is no Package belongs to the nearest Package above it,** and to Root when there is none. Scope, visibility and dependencies are that Package's, so the Definitions in two folders of one Package link each other freely.
- **A folder that is no Package is no part of the project.** A link to it resolves to nothing (`link.unresolved`), and a registry page for it belongs to no Package (`stale.page`).
- **A Package may lie in any folder.** `members/concepts/card/card.md` of kind `Package` makes `members/concepts/card` a Package whose parent is `members`.

**A thing with parts** is a folder that holds the Definition of its own name, the whole, and its parts in it or below it: `operations/lend/lend.md` with `operations/lend/conditions/in-stock.md` ([DEC-109](../decisions.md#dec-109-a-thing-with-parts-is-read-as-a-whole)). Each part is a Definition like any other: verified, realized and linked alone, and it links its whole only where it relies on what the whole states.

- **A link to the folder means the whole.** `[[members/operations/lend]]` is the Definition `members/operations/lend/lend`, in a Definition's text, as a `kind` and in a registry row.
- **`doc` shows the whole with its parts,** and a part with the whole it belongs to ([the commands](#the-commands)).
- **`vet` counts things.** Parts do not count for the size of a Package, and a part that names its whole without linking it is no unlinked mention.
- **A Package ends a thing.** An anchor is no whole, and what lies in a Package is no part of a thing above it.

## Kinds and Libraries

Core offers two Definitions: `Kind` and `Package`. A Kind is a Definition whose `kind` is `Kind`; its body says what it is and how it is used. Every other Definition's `kind` names a Kind by path (`kind.unresolved`, `kind.not_a_kind`). `Package` is the Kind of an anchor, the Definition that makes its folder a [Package](#packages-and-folders).

A Library is a folder of Kind Definitions, one `<Name>.md` file each at its top level (`library.file`, `library.not_a_kind`, `library.internal`); a Library's name follows the same rules as other names (`library.name`). A Library Definition may link only to Libraries the manifest names and to Core's `Kind` and `Package`. No Kind of a Library has a special meaning to Markitect.

The standard Libraries ship inside the binary at version 1.1.0:

| Library | Kinds |
|---|---|
| `std` | [Project](../../internal/libraries/std/Project.md), [Rule](../../internal/libraries/std/Rule.md), [Concept](../../internal/libraries/std/Concept.md), [Feature](../../internal/libraries/std/Feature.md), [Decision](../../internal/libraries/std/Decision.md) |
| `rest` | [Endpoint](../../internal/libraries/rest/Endpoint.md) |
| `cli` | [Command](../../internal/libraries/cli/Command.md) |

Every project uses `std`. A project writes its own Kinds for anything else, one small file each, anywhere in its spec; a Kind shared by several Packages sits at Root, because using a Kind is a dependency. Any other Library is committed under `.markitect/libraries/<name>/`.

### Manifest and lock

```yaml
# .markitect/manifest.yaml
libraries:
  std: 1.1.0
  rest: 1.1.0
```

The manifest names each Library with its version and nothing else (`manifest.syntax`, `manifest.missing`). A standard Library must have the version this binary ships (`library.version`), and no project folder may replace it (`library.reserved`). A version names one content: when a Kind of a standard Library changes, the version of the standard Libraries moves, and `markitect tidy` and `markitect update` name the new one in the manifest before they write the lock. Any other named Library must exist under `.markitect/libraries/` (`library.missing`), and every Library folder there must be named (`library.undeclared`).

`markitect tidy` brings `.markitect/lock.yaml` in line with the manifest, as `go mod tidy` does: it writes each Library's version and the SHA-256 digest of its files. `check` only reads the lock. It refuses a Library whose version or content does not match the lock, and a lock entry the manifest does not name (`lock.mismatch`, `lock.missing`, `lock.syntax`), and says to run `markitect tidy`. `init` and `update` keep the lock current. A new markitect binary whose standard Libraries changed therefore needs a new lock, which shows the change in review.

## Links, scope and visibility

A link names its target by path: `[[members/member]]` inside the spec, `[[std/Rule]]` for a Library Kind, `[[members]]` for a Package, `[[members/operations/lend]]` for a [thing with parts](#packages-and-folders). A label after `|` and a section after `#` are allowed and ignored for resolution: `[[members/member#Status|member]]`. In a Markdown table, write the label's bar as `\|`. Links inside code spans and code blocks are text, not links. There are no imports and no relative links: a link to a Definition in the same Package still names its full path, `[[members/member]]` from inside `members/`.

- A path whose first name is a Library the manifest names is a Library link. No top-level folder or Definition of the spec may have a Library's name, or the name `Kind` or `Package` (`spec.reserved`).
- A link that resolves to nothing, or that is not a path of valid names (`../x`, `/x`), is an error (`link.unresolved`, `link.syntax`). A folder that is neither a Package nor a thing with parts is nothing a link resolves to.

**Every link is a relation** ([DEC-104](../decisions.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them)). The spec behaves like code, and code has no "see also", so a link is never decorative. A link to a Definition says that the source relies on what the target states: a change to the target puts the source back on the list of `status`. A link to a Package says that the source's module depends on that module or, down the tree, says something about a part of the parent's organization. The Kind, the tree with its visibility, the registry and the evidence state the other relations. A link that would only help a reader find something does not belong in the spec; navigation belongs in documentation rendered from it. The [guide to a good spec](good-spec.md#every-link-is-a-relation) shows how identities, a module's surface and the modules it uses are written under this rule.

**Scope is location.** Everything a Package defines reaches its whole subtree: a Rule at Root binds the whole project.

**Visibility as in Go.** A Definition is public unless its front matter says `internal: true`. Only its own Package and the Packages below it may link to an internal Definition, or use it as a Kind (`link.internal`). Visibility binds links, not reading: Packages above and reviewers may read internal Definitions, and registry entries may link any Definition they realize. A Package's public Definitions are what it publishes, as a Go package's exported names are; `doc` shows them apart from its internal ones, and a spec does not list them again.

**Dependencies as in Go.** A link from a Definition in Package A to a Definition in another Package B, or to the Package B itself, is an edge A → B. Using a Kind is an edge too. Every Package also has an implicit edge to its parent, because its parent's Definitions reach it. So:

- a link up the tree, to an ancestor's Definition, is free: the implicit edge already exists;
- a parent may link to a child Package, which names part of the organization and is no edge (`authorization is realized in [[pipeline]]`), but not to a Definition inside a child, nor use a child's Kind (`link.downward`);
- links sideways are edges, and a cycle of edges is an error (`package.cycle`). The diagnostic shows one path through the cycle, including implicit edges ("a/b lies inside a").

A cycle is broken by moving the shared concept into a common parent, or by inverting the dependency: the needing side describes its need in its own Package, and the other side links to it. Library links make no edges; Libraries lie outside the Package graph.

## The realization record

### Registry pages

Every Package may have one registry page: `.markitect/registry.md` for Root, `.markitect/registry/<package path>.md` for the others. A folder that is no Package has none: the files that realize its Definitions are listed, like every file, on the page of the Package that owns them. The front matter names the path patterns the Package owns; a table lists every file it owns with the Definitions the file realizes and why.

```markdown
---
paths:
  - src/members/**
  - tests/members/**
  - docs/members*.md
---
# Registry: members

| File | Realizes | Why |
|---|---|---|
| `src/members/member.go` | [[members/member]] | Member type and validation |
| `tests/members/member_test.go` | [[members/member]] [[members/register-member]] | Tests |
```

- **Patterns** are repository-relative paths. `*` and `?` match within one name; a whole segment `**` matches any number of folders. A trailing `**` matches what lies inside the folder, not the folder itself. A pattern without wildcards names one file. `[ ]` and `{ }` are not allowed (`registry.pattern`).
- **Ownership:** the most specific matching pattern owns a file. A pattern without wildcards beats every pattern with one; otherwise the pattern with more literal characters (other than `*`, `?` and `/`) wins. Root owns every file no other Package claims and names no patterns. Two Packages claiming the same pattern is an error (`registry.duplicate_pattern`), and so are two equally specific patterns of different Packages that match the same file (`registry.ambiguous_owner`).
- **First patterns:** a Package other than Root with no patterns, or no page, compiles but owns no file (`vet.no_paths`). The owner writes a new Package's Definitions; the builder writes its registry page with its first patterns, and `check` and `status` still require every file to be accounted for. The builder maintains every registry page. Patterns say who owns a file; they do not limit what an agent may touch.
- **Accounting:** owning is not accounting. Every repository file outside `.markitect/` must be listed on its owner's page or be ignored (`file.unregistered`). A file listed on another Package's page (`registry.wrong_owner`), listed twice (`registry.duplicate_entry`), listed and ignored (`registry.ignored_file`), ignored by Git (`registry.git_ignored`) or below `.markitect/` (`registry.own_file`) is an error.
- **Rows** are the rows of every table whose first header is `File`, with three cells: the file (in backticks or plain), the links to the Definitions it realizes (none is allowed), and a short reason (`registry.table`). A link in the Realizes cell must name a Definition of the spec, not a Package or a Library Kind (`registry.link`). A link label inside a table needs its bar escaped, `[[members/member\|member]]`; an unescaped bar splits the row and is an error (`registry.table`). Other text on the page is free. The front matter holds only `paths` (`registry.front_matter`).
- **Shared files** such as the README, a CI pipeline or a compose file are owned by the lowest common parent of the Packages they serve, often Root, and their entry links every Definition they realize.

### Stale realization

When the owner deletes a Definition, a file or a Package, the registry still names it. That is stale realization, an error that says spec and code do not match yet, so the work does not stop on it, and `status` lists it until it is settled:

| Code | What |
|---|---|
| `stale.definition` | a registry link to a Definition that no longer exists |
| `stale.file` | an entry for a file that no longer exists |
| `stale.page` | the page of a Package that no longer exists, or of a folder that is no Package, and each file still listed only there |

`check` fails on each, in its group of errors in the realization, as it does on a file the registry does not list and on a Definition nothing links to and no file realizes (`definition.unused`). `check --spec` looks only at the spec itself.

A project with nothing built yet passes: when the registry lists no file and the only errors are `definition.unused`, `check` says that the spec compiles and nothing is built yet, and exits 0. A file in the project folder, or anything registered, ends that. `--json` then carries `"nothing_built": true`. This holds for `check` only: `status` still lists a Definition nothing realizes.

### Ignore file

`.markitect/ignore` lists files that deliberately belong to no Package. It reads like a `.gitignore`: one pattern per line and `#` for a comment. A project that needs none has no such file.

```text
# Legal text, realizes nothing
LICENSE
docs/assets/**
```

Its patterns are the registry's, not Git's: they start at the top of the project folder, `*` and `?` stay within a folder, `**` is a whole segment, and `!` does not exist (`ignore.syntax`). So `*.png` means the top folder only, and a folder's files are `docs/assets/**`, not `docs/assets/`.

It can't merge with `.gitignore`, because the two answer different questions: `.gitignore` says what Git does not track, and this file says what Git tracks and no Package owns. A file Git ignores needs no line here.

The tool accounts for what `init` installs by itself: `AGENTS.md`, `CLAUDE.md`, `.claude/settings.json`, the `markitect-*` skills, agents and workflows, and `.gitignore`, and for the worktrees Claude Code keeps under `.claude/worktrees/`. They need no line here and no registry entry, though a registry page may list them. The ignore file lies under `.markitect/`, so no subagent but the architect may change it ([the hook](../../internal/guard/README.md#the-hook)); a file it lists is no input of any Definition.

### The agents' model and effort

The model and effort of Markitect's agents are set in the harness configuration, `.markitect/harness.yaml` for the project and `~/.markitect/harness.yaml` for all of a user's projects, with `markitect config` ([strategies and configuration](project-workflow.md#strategies-and-configuration)); `init`, `update` and `config` write them into the agent files. Without it the builder and the reviewer use `sonnet` at effort `high`, and the architect and the auditor `opus` at effort `medium`, as their files say. The manifest holds only the Libraries.

```yaml
# .markitect/harness.yaml
agents:
  builder:
    model: opus
```

## Commands

All commands take `--repo DIR`, a folder of the project (default: the current folder), and are flat, with no groups ([DEC-096](../decisions.md#dec-096-the-tool-computes-and-checks-a-driver-acts)). `check`, `vet`, `list`, `doc`, `status`, `mark` and `config --list` take `--json`.

### Diagnostics, JSON and exit codes

As in Go's tools:

- **Diagnostics** are one line each, `file:line: code: message`. Codes that start with `spec.` concern the spec itself. `check` prints only diagnostics and one summary line, `vet` its warnings and one line that counts its notes (with `--strict`, the notes as diagnostics instead), and `status` prints the errors in the realization as diagnostics.
- **`--json`** prints one document that starts with `format_version` (1) and `kind`, the command's name. A diagnostic in it is an object with `severity` (`error`, `warning` or `note`), `code`, `message`, `file` and `line`.
- **Exit codes:** 0 when all is well; 1 when the command did its job and found something (errors of `check`, items of `status`, a refused `mark`, warnings of `vet` with `--strict`, a `doc` target that is neither a Definition nor a Package); 2 when it could not run (a usage error, no project). Notes, such as a `skip` or an outdated harness, never change the exit code.

The fields of each document:

| Command | Fields after `format_version` and `kind` |
|---|---|
| `check` | `ok`, `nothing_built`, `definitions`, `packages`, `files`, `spec_errors`, `realization_errors`, `diagnostics` |
| `vet` | `diagnostics`: the warnings, then every note with `severity` `note`, `--strict` or not; a `vet.unlinked_mention` adds `targets`, the Definitions it names, and `structure`, `against` or `new_dependency` for those a link to which would close a Package cycle or add a Package dependency |
| `list` | `packages`: `path`, `name`, `parent`, `definitions` (`id`, `title`, `kind`, `file`, `internal`) and, with `--deps`, each Definition's `links` and the Package's `deps` (`afferent`, `efferent`, `instability`, `abstractness`, `distance`, `depends_on` with the `uses` behind each edge: `from`, `target`, `via` (`link` or `kind`), `file`, `line`) |
| `doc` | `definition`: `id`, `title`, `kind`, `file`, `internal`, `links`, `package`, `library`, `text`, `whole` (the Definition it is a part of), `parts` (`id`, `title`, `kind`, `group`, the folder it lies in below the whole, and `text`), `linked_from`, `realized_by`, `record`; or `package`: `path`, `name`, `parent`, `children`, `anchor`, `text` (the anchor's text without front matter), `page`, `paths`, `definitions` (each with `internal`), `linked_from`, `files`; or, for a target that is neither, `diagnostics` (`doc.unknown`) |
| `status` | `clean`, `for_the_owner` (something is left, and all of it is `spec-problem`s recorded at the inputs as they are), `package` and `not_listed` (only with `--package`: every other field but `packages` is then that Package's part, and `not_listed` counts the errors in the realization and the unreadable lines, which belong to no Package), `definitions`, `spec_errors`, `realization_errors`, `packages` (each Package of the spec, by name: `package`, `definitions`, `items` and `spec_problems` on the list, and `depends_on`, the Packages it depends on as `list --deps` has them, where a link to an ancestor is implicit), `items` (`definition`, `title`, `state`, `fingerprint`, `changes`, `verdict`, `why`, `verdicts`, `look`), `diagnostics` (the spec's errors, the realization's, and `record.unreadable` for each line of the record markitect cannot read), `skipped`, `unreadable_lines`, `questions` (the path of the questions file, when it exists and the list is not limited to a Package) |
| `mark` | `recorded`, `item`, `verdict`, `message` |
| `config` | `values` (`key`, `value`, `origin`: `built-in`, `user` or `project`, `file`, `via`), `problems` |

`init`, `update`, `tidy`, `install` and `version` print text only.

### The commands

| Command | Does |
|---|---|
| `markitect install` | Writes the personal skill `~/.claude/skills/markitect/SKILL.md` from the binary, and says where. It tells Claude Code what Markitect is in any folder, how to begin and how to upgrade, and allows no command in advance. Optional: `init` gives a project its own skills. After an upgrade, `update` refreshes it. It takes no `--repo`. |
| `markitect update [--check] [--files] [--force]` | Installs the newest release over the binary (checked against the release's checksums; not a binary the plugin manages, and a development build only with `--force`; without a network it says so and goes on), then lets the new binary bring what an earlier build wrote in line with it; `--files` skips the binary. That is: the personal skill if it exists, and in a project the skills in `.claude/skills/`, the four agents with the model and effort of the harness configuration, the parallel workflow, the hooks and `worktree.baseRef` (unless the settings name a base), `.markitect/.gitattributes` and the lock, whose change the owner reads before committing it. It commits nothing. Compares without regard to line endings. Never changes a spec file, `AGENTS.md` or `CLAUDE.md`, and does not create the personal skill. `--check` changes nothing, lists what is out of date and exits 1 if anything is. Outside a project it refreshes the personal skill only. |
| `markitect config [--user] KEY VALUE` | Sets a value of the harness configuration, in the project's `.markitect/harness.yaml` or with `--user` in `~/.markitect/harness.yaml`, and rewrites the agents' files of the project; key by key the project's value beats the user's, which beats the built-in one. `--unset KEY` removes a value (exit 1 when the file sets none), `KEY` alone prints the value in force, `--list` every value with where it comes from (exit 1 when a strategy names a workflow no level defines). The keys and values: [strategies and configuration](project-workflow.md#strategies-and-configuration). A bad key or value is a usage error, exit 2. |
| `markitect version` | Prints the build, then whether the personal skill and the project in `--repo` are up to date, out of date or (the personal skill) not installed. Changes nothing. |
| `markitect init [--name TITLE]` | Writes what is missing: the manifest with `std`, the project anchor `spec/project.md` if the spec folder is empty, Root's registry page, the lock, `AGENTS.md` and a `CLAUDE.md` that imports it. Installs or updates the skills in `.claude/skills/`, the harness (the build, review, cleanup, bughunt, audit, schedule and config skills, the builder, reviewer, architect and auditor agents with the model and effort of the harness configuration, the parallel workflow `.claude/workflows/markitect-parallel.js`, and in `.claude/settings.json` the hooks and `worktree.baseRef: "head"` unless the settings name a base), `.markitect/.gitattributes` and, unless a line already ignores it, `.claude/worktrees/` in `.gitignore`. In a folder of a project it works on that project. It makes no repository and commits nothing, and prints what Markitect is, with a tip when the personal skill is missing or old. Never changes an existing spec file, `AGENTS.md` or `CLAUDE.md`, and never writes to the home folder. |
| `markitect check [--spec]` | Compiles the project and prints every error as a diagnostic, the spec's first and then the realization's, and one summary line: `ok: N Definitions in M Packages, K files accounted`; the sentence for a project where [nothing is built yet](#the-realization-record); or `N errors: X in the spec, for the owner to settle; Y in the realization, so spec and code do not match yet, which /markitect-build settles`. Exit 1 when there is an error. `--spec` looks only at the spec. |
| `markitect tidy` | Brings the lock in line with the manifest and the Libraries it names, as `go mod tidy` does. Where the manifest names another version of a standard Library than this binary ships, it names the shipped one there first, and says so. The lock's header reads "Written by markitect tidy from the manifest". |
| `markitect vet [--strict]` | Analyses the spec's design: coupling, size, hubs, and as notes unlinked mentions, statements, placement and hotspots ([the warnings and notes](#vet-warnings-and-notes)). Prints its warnings as diagnostic lines, then one line that counts its notes, such as `notes: 12 unlinked mentions in 5 Definitions, 3 more against or beyond the Package structure, 1 other note; markitect vet --strict lists them`, and nothing when there is neither. `--strict` prints the notes as diagnostic lines instead of that line. Exit 1 only with `--strict` and a warning; notes never change it. |
| `markitect list [--deps]` | Each Package by name (`Root` for the root) with its Definitions below it as `  ID: Title (Kind)`. With `--deps` the Package line adds `(Ca n, Ce n, instability x.xx, abstractness x.xx, distance x.xx)`, information only, and is followed by `  depends on P: A links B; C is of Kind D`, and a Definition by the links it makes, `    links X, Y`. |
| `markitect doc TARGET` | A Definition: `ID: Title (Kind)`, its Package and file (or its Library), the whole it is a part of (`Part of ID`), its text without front matter, its parts (each folder's under the folder's name, a part's title and ID before its text, in the order the whole links them and then by ID), what it links to, what links to it (`  FROM (link\|kind), FILE:LINE`), the files that realize it (`  FILE, PAGE:LINE`) and its record (`Record: ID@FP, STATE`, with the changes or the reason). A Package: its parent, registry page and paths, the Packages below it, the text of its anchor, its public Definitions and, apart, its internal ones, what links to it and its files. `doc Root` or `doc .` shows Root. Exit 1 for a target that is neither. |

`vet`, `list` and `doc` also run over a spec with errors and say so on standard error.

Two commands work on the record of what was verified, `.markitect/review.sum` ([DEC-099](../decisions.md#dec-099-a-record-of-what-was-verified)). The `/markitect-build` and `/markitect-review` skills have the agents run them ([project workflow](project-workflow.md#7-build-and-review)). [The record](../../internal/run/README.md) has the inputs and the rules.

| Command | Does |
|---|---|
| `markitect status [--all] [--package P]` | Lists what is to verify: each Definition with a changed input or no record, with what changed, where to look and its fingerprint; each Definition recorded `fail` or `spec-problem`; each Definition gone from the spec, with the files that realized it; conflicts after a merge; and the errors in the realization. When only `spec-problem`s are left, one line says so. A `skip` is a note, and so is the last line, which says that `.markitect/questions.md` exists while it does. Exit 0 when it lists nothing, 1 when it lists something; a spec with errors has no list. `--all` lists every Definition. `--package P` lists only the items whose Definition belongs to the Package of that name, `Root` for the one at the top, and not those of the Packages below it, and counts the errors in the realization and the unreadable lines of the record without listing them, since they belong to no Package; exit 0 then means that nothing is to verify in that Package. Changes nothing. |
| `markitect mark DEFINITION@FINGERPRINT VERDICT [-e EVIDENCE]... [-m WHY] [--cases N --covered M]` | Records `ok`, `fail`, `skip` or `spec-problem` at the state `status` printed, which prints each item as `DEFINITION@FINGERPRINT`. `ok` names its evidence with `-e`, files of the folder as `path` or `path:line`; the others say why with `-m`. `--cases` and `--covered` are the reviewer's counts of the Definition's cases, conditions and exceptions and of those its evidence demonstrates, kept on the line; `mark` refuses an `ok` or a `skip` on a Definition of the spec without both ([the record file](../../internal/run/README.md#the-record-file)), which also counts how often the Definition's own text changed between verdicts. Refuses, and records nothing, when the inputs changed since the fingerprint or evidence does not exist. In an agent session, only the reviewer agent may run it (the hook). |
| `markitect hook pretool` | The hook before a tool call: only the reviewer records, a subagent writes only what its role may, and none starts the builder, the architect or a workflow ([the hook](../../internal/guard/README.md#the-hook)). Hooks only. |

An unknown command is a usage error, exit 2.

### Vet warnings and notes

`vet` measures the spec like code, from its text and links alone, and from the record's count of revisions ([DEC-102](../decisions.md#dec-102-what-vet-measures)). The thresholds are provisional, set on [measurements of the Makerspace specs](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/spec-design-measures-20261107.md). A Definition nothing links to and no file realizes, and a stale registry row, are errors of `check`, not warnings. N is the number of the spec's own Definitions, Library Kinds left out; a Definition's fan-in counts the Definitions that link to it or are of it as Kind.

**Warnings** were precise on real specs; `vet --strict` exits 1 on them.

| Code | Warns when |
|---|---|
| `vet.coupling` | a Package depends on more than 5 Packages (efferent coupling Ce) |
| `vet.unstable_dependency` | a Package depends on a less stable one: instability Ce / (Ca + Ce) of the target is higher, Martin's Stable Dependencies Principle |
| `vet.definition_size` | a Definition's body has more than 1000 words |
| `vet.package_size` | a Package has more than 25 Definitions of its own, the parts of a thing not counted |
| `vet.hub` | a change to one Definition alone would put more than a fifth of the spec back on the list to verify, and more than 10: fan-in + 1 > max(10, N / 5). It counts Root's Definitions and leaves Kinds out, whose change concerns all their Definitions by design |
| `vet.thin_package` | a Package holds only its anchor and has only one child |
| `vet.no_paths` | a Package other than Root claims no path pattern, as above |

**Notes** have a known blind spot or no calibration yet, so they never change the exit code. Plain `vet` counts them in one line; `--strict` lists them; `--json` always carries them.

| Code | Notes when |
|---|---|
| `vet.unlinked_mention` | a Definition's title, or the last segment of its ID with hyphens read as spaces, appears in another Definition's prose with no link to it: matched as whole words without regard to case, with up to two trailing letters, outside the title, code, raw HTML and `[[…]]` links (a Markdown link's label counts as prose). Not reported: where the Definition links the named one or is of it as Kind, where it may not link it (into a child, or an internal Definition from outside), where a longer name covers the words (its own included), where a linked Definition's name covers them, where the named one is a Kind or a Package's anchor, whose name is the Package's own, and where it is the whole the Definition is a part of. One line per Definition, at its first mention. Mentions a link to which would close a Package cycle or add a Package dependency the spec does not have are counted apart, each group in a line of its own that says so. About half to three-quarters are real missing links; [`/markitect-audit`](project-workflow.md#7-build-and-review) has the auditor triage them |
| `vet.statements` | a Definition has more than 20 statements: list items and table rows, header rows included |
| `vet.placement` | every Definition that links to a Definition lies in one Package below it, so it could move down there, unless it binds its Package by scope |
| `vet.hotspot` | how often a Definition's own text changed between verdicts, from the record, times its fan-in, is more than a tenth of the spec and more than 20: revisions × fan-in > max(20, N / 10) |
| `vet.risk` | a Definition whose `ok` or `skip`, at its text as it is, has a risk above 30, from the reviewer's counts in the record: with *c* one more than its cases and *e* the share of them its evidence demonstrates, 0 for a `skip`, the risk is *c*² × (1 − *e*)³ + *c* ([what it is for](good-spec.md#evidence-for-complexity-a-crap-analogue)) |

Coupling counts explicit edges only, not the implicit edge to the parent. `list --deps` shows each Package's abstractness, the share of its own Definitions that are Kinds, and its distance from the main sequence, \|A + I − 1\|; `vet` warns about neither. [What makes a good spec](good-spec.md) says what each measure is for, and [DEC-102](../decisions.md#dec-102-what-vet-measures) records these measures and thresholds and the ones not built yet.

## Details

Details the concept leaves open:

- **Layout.** The fixed `.markitect/` layout above; registry pages mirror Package paths, Root's page is `registry.md`.
- **A gone Definition's Package.** A Definition that is gone from the spec counts for the nearest Package above its path that exists, and for Root when there is none.
- **Library links.** A path whose first name is a Library is a Library link, so no top-level name of the spec may equal a Library's name.
- **Local Library versions.** A local Library carries no version file; the manifest names its version and the lock pins its digest.
- **Specificity.** An exact path beats every pattern; otherwise more literal characters win; a tie between Packages is an error.
- **Stale realization** and unused Definitions are errors of the realization: they stop `check`, and `status` lists them, but they do not stop its list of Definitions ([concept](../concept.md#what-check-refuses), [DEC-006](../decisions.md#dec-006-every-file-is-accounted-for)). A file of the project folder without any entry is the same.
- **No inherited items.** A changed Rule above a Package is one item, with what links to it, not an item on every Package's list below it ([DEC-099](../decisions.md#dec-099-a-record-of-what-was-verified)).
- **How far a change reaches.** A Definition's inputs take one step of links, its Kind, and the registry rows in both directions, so a row that gains or loses it reaches it; Definitions further along the links are not inputs. The code's own imports are not followed yet; a later step that knows the realization's languages can widen reach.
- **The harness** is installed into `.claude/`, for Claude Code: the architect, build, review, cleanup, bughunt, audit, schedule and config skills, the builder, reviewer, architect and auditor agents, and the parallel workflow ([DEC-100](../decisions.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)).

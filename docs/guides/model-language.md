# Model language and model commands

This guide is the canonical description of the model language of [concept 2.0](../concepts/v2-concept.md) (§2–§4, [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation), [DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)) as `markitect` implements it, and of the model commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint`. None of them starts an agent. Where the concept leaves a detail open, the [build decisions](#build-decisions) at the end record what was chosen.

## The project's Markitect folder

```text
.markitect/
  manifest.yaml        the Libraries the project uses, with their versions
  lock.yaml            the digest of each Library, written by markitect lock
  execution.yaml       the required checks
  model/               the model; this folder is the Root Package
    project.md         the project anchor (std/Project)
    members/           a Package
      member.md        a Definition: ID members/member
  registry.md          Root's registry page
  registry/
    members.md         the registry page of Package members
    members/loans.md   the registry page of Package members/loans
  ignore.md            files that deliberately belong to no Package
  libraries/<name>/    a Library that does not ship with markitect
  results/             Run Results (written by Runs, not by these commands)
```

Every file below `.markitect/` is accounted for by the compiler itself and needs no registry entry. A file there that has no place in this layout is an error (`project.file`), and so is a file in the registry folder that is not a Markdown page (`registry.page`) or a file Git lists but the compiler cannot read (`project.read`). The installed tool, `.markitect/tool/` and `.markitect/bootstrap/`, is also Markitect's own. Text files under `.markitect/` are read with LF line ends, so a Windows checkout with CRLF compiles, locks and compares like the commit it came from.

The compiler sees exactly what Git sees: tracked files and untracked files Git does not ignore.

## Definitions

Everything in a model is a Definition: one Markdown file with YAML front matter.

```markdown
---
kind: std/Feature
---
# Create reservation

A [[members/member|member]] reserves a [[equipment/machine|machine]] for a slot.
```

- The front matter holds `kind` and, where needed, `internal: true`. Nothing else; another key is an error (`definition.front_matter`).
- The file name without `.md` is the Definition's name, and its path below `model/` is its ID: `members/member.md` is `members/member`. Names start with a letter or digit and use only letters, digits, `-` and `_` (`model.name`).
- The body starts with one level-1 heading, the title; no other level-1 heading follows (`definition.title`).
- The rest is documentation. Exact values such as wire names and error codes are written in backticks.
- The model folder holds only `.md` files and folders (`model.file`). Two entries of one folder whose names differ only in case, or a file and a folder of the same name (`member.md` and `member/`), collide (`model.collision`): a link could not tell them apart.

## Kinds and Libraries

Core offers one thing: the Definition `Kind`. A Kind is a Definition whose `kind` is `Kind`; its body says what it is and how it is used. Every other Definition's `kind` names a Kind by path (`kind.unresolved`, `kind.not_a_kind`).

A Library is a folder of Kind Definitions, one `<Name>.md` file each at its top level (`library.file`, `library.not_a_kind`, `library.internal`); a Library's name follows the same rules as other names (`library.name`). A Library Definition may link only to Libraries the manifest names and to `Kind`. No Kind has a special meaning to Markitect.

The standard Libraries ship inside the binary at version 1.0.0:

| Library | Kinds |
|---|---|
| `std` | [Project](../../internal/libraries/std/Project.md), [Rule](../../internal/libraries/std/Rule.md), [Concept](../../internal/libraries/std/Concept.md), [Feature](../../internal/libraries/std/Feature.md), [Decision](../../internal/libraries/std/Decision.md) |
| `rest` | [Endpoint](../../internal/libraries/rest/Endpoint.md) |
| `cli` | [Command](../../internal/libraries/cli/Command.md) |

Every project uses `std`. A project writes its own Kinds for anything else, one small file each, anywhere in its model; a Kind shared by several Packages sits at Root, because using a Kind is a dependency. Any other Library is committed under `.markitect/libraries/<name>/`.

### Manifest and lock

```yaml
# .markitect/manifest.yaml
libraries:
  std: 1.0.0
  rest: 1.0.0
```

The manifest names each Library with its version and nothing else (`manifest.syntax`, `manifest.missing`). A standard Library must have the version this binary ships (`library.version`), and no project folder may replace it (`library.reserved`). Any other named Library must exist under `.markitect/libraries/` (`library.missing`), and every Library folder there must be named (`library.undeclared`).

`markitect lock` writes `.markitect/lock.yaml` with each Library's version and the SHA-256 digest of its files. `check` refuses a Library whose version or content does not match the lock, and a lock entry the manifest does not name (`lock.mismatch`, `lock.missing`, `lock.syntax`). A new markitect binary whose standard Libraries changed therefore needs a new lock, which shows the change in review.

## Links, scope and visibility

A link names its target by path: `[[members/member]]` inside the model, `[[std/Rule]]` for a Library Kind, `[[members]]` for a Package. A label after `|` and a section after `#` are allowed and ignored for resolution: `[[members/member#Status|member]]`. In a Markdown table, write the label's bar as `\|`. Links inside code spans and code blocks are text, not links. There are no imports and no relative links: a link to a Definition in the same Package still names its full path, `[[members/member]]` from inside `members/`.

- A path whose first name is a Library the manifest names is a Library link. No top-level Package or Definition of the model may have a Library's name, or the name `Kind` (`model.reserved`).
- A link that resolves to nothing, or that is not a path of valid names (`../x`, `/x`), is an error (`link.unresolved`, `link.syntax`).

**Scope is location.** Everything a Package defines reaches its whole subtree: a Rule at Root binds the whole project.

**Visibility as in Go.** A Definition is public unless its front matter says `internal: true`. Only its own Package and the Packages below it may link to an internal Definition, or use it as a Kind (`link.internal`). Visibility binds links, not reading: Managers above and checkers may read internal Definitions, and registry entries may link any Definition they realize.

**Dependencies as in Go.** A link from a Definition in Package A to a Definition in another Package B, or to the Package B itself, is an edge A → B. Using a Kind is an edge too. Every Package also has an implicit edge to its parent, because its parent's Definitions reach it. So:

- a link up the tree, to an ancestor's Definition, is free: the implicit edge already exists;
- a parent may link to a child Package, which names part of the organization and is no edge (`authorization is realized in [[pipeline]]`), but not to a Definition inside a child, nor use a child's Kind (`link.downward`);
- links sideways are edges, and a cycle of edges is an error (`package.cycle`). The diagnostic shows one path through the cycle, including implicit edges ("a/b lies inside a").

A cycle is broken by moving the shared concept into a common parent, or by inverting the dependency: the needing side describes its need in its own Package, and the other side links to it. Library links make no edges; Libraries lie outside the Package graph.

## The realization record

### Registry pages

Every Package may have one registry page: `.markitect/registry.md` for Root, `.markitect/registry/<package path>.md` for the others. The front matter names the path patterns the Package owns; a table lists every file it owns with the Definitions the file realizes and why.

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
- **Accounting:** owning is not accounting. Every repository file outside `.markitect/` must be listed on its owner's page or be ignored (`file.unregistered`). A file listed on another Package's page (`registry.wrong_owner`), listed twice (`registry.duplicate_entry`), listed and ignored (`registry.ignored_file`), ignored by Git (`registry.git_ignored`) or below `.markitect/` (`registry.own_file`) is an error.
- **Rows** are the rows of every table whose first header is `File`, with three cells: the file (in backticks or plain), the links to the Definitions it realizes (none is allowed), and a short reason (`registry.table`). A link in the Realizes cell must name a model Definition, not a Package or a Library Kind (`registry.link`). A link label inside a table needs its bar escaped, `[[members/member\|member]]`; an unescaped bar splits the row and is an error (`registry.table`). Other text on the page is free. The front matter holds only `paths` (`registry.front_matter`).
- **Shared files** such as the README, a CI pipeline or a compose file are owned by the lowest common parent of the Packages they serve, often Root, and their entry links every Definition they realize.

### Stale realization

When the owner deletes a Definition, a file or a Package, the registry still names it. That is stale realization, not a compile error, so the initial gate does not block the Run that cleans it up:

| Code | What | Who cleans it up |
|---|---|---|
| `stale.definition` | a registry link to a Definition that no longer exists | the page's Package |
| `stale.file` | an entry for a file that no longer exists | the page's Package |
| `stale.page` | the page of a Package that no longer exists, and each file still listed only there | Root for the page; the file's current owner for each file |

`check` passes, `lint` warns (`lint.stale`), and `impact` puts each item on the checklist of the Manager that must clean it up.

### Ignore page

`.markitect/ignore.md` lists files that deliberately belong to no Package, in a table whose first header is `Path`: a pattern and a reason (`ignore.syntax`). `markitect init` adds the installed skill folders to it.

```markdown
| Path | Why |
|---|---|
| `.claude/skills/markitect-*/**` | Skills installed by markitect init |
| LICENSE | Legal text, realizes nothing |
```

### Required checks

`.markitect/execution.yaml` holds only the required checks, the one proof in a Run that does not come from an agent (`execution.syntax`, `execution.missing`):

```yaml
checks:
  - name: test
    command: go test ./...      # run with sh -c
    directory: .                # repository-relative; default .
    timeout: 10m                # a Go duration, required
    environment: golang:1.27    # the container image the check runs in
```

## Commands

All commands take `--repo DIR`, a folder inside the project's Git repository (default: the current folder). Every command except `init` and `lock` takes `--json` and then prints JSON instead of Markdown. Exit code 0 means success, 1 that the command ran and found errors, 2 that it could not run.

| Command | Does |
|---|---|
| `markitect init [--name TITLE]` | Writes what is missing: the manifest with `std`, `execution.yaml` with no checks, the project anchor `model/project.md` if the model folder is empty, Root's registry page, the ignore page and the lock. Installs or updates the skills in `.claude/skills/` and `.agents/skills/` and lists them on the ignore page. Never changes an existing model file. |
| `markitect check` | Compiles the project and prints every compile error, one per line as `file:line: code: message`. Errors only; exit 1 when there is one. |
| `markitect lock` | Writes the lock from the manifest and the Libraries it names. |
| `markitect refs TARGET` | Who links to a Definition or Package: Definitions by link or as their Kind, and the registry entries that realize it. |
| `markitect graph` | The Package dependency graph as a Mermaid diagram, with each Package's coupling and the links behind each edge. |
| `markitect impact --base COMMIT [--full] [--findings FILE]` | What the change from the base commit to the working tree reaches, and each working Manager's checklist. Refuses a project with compile errors. |
| `markitect lint` | Warnings only, and the coupling table. |

`refs`, `graph` and `lint` also run over a project with compile errors and say so on standard error.

### Impact and checklists

`impact` compares the model and the repository files at the base commit with the working tree:

- **Changed Definitions** are the model's and the Libraries' Definitions that were added, modified or removed.
- **Reached Definitions** are the model Definitions that changed, that link to or are of a Definition that changed, or that a changed file realizes (through the registry; a removed file through the base's registry). Reach follows one step of links.
- **Reached files** are the changed files and the files that realize a reached Definition, each with its owner.
- **Working Managers** are the Packages with a checklist item or a reached file, and their parents up to Root, which review and integrate.

Each working Manager's checklist (concept §6.2) has these parts:

| Part | Holds |
|---|---|
| Own | its own Definitions the change reaches, each with why |
| Inherited | the Definitions of the Packages above it that the change changed; a changed Rule at Root is an item for every Manager below it |
| Findings | every Finding whose Definition is in its Package |
| Stale | the stale realization it must clean up |

With `--full` every model Definition is reached, every Manager works, and the checklist holds all own and all inherited Definitions.

### Findings

A Finding names a Definition and says what was observed and what is expected. A Findings page holds one Finding per level-2 heading; the heading holds exactly one link, and the paragraphs below start with `Observed:` and `Expected:` (`findings.syntax`):

```markdown
# Findings

## [[members/member]]

Observed: a member with three open loans could borrow a fourth.

Expected: the loan limit of [[policy]] refuses it with `limit_reached`.
```

A Finding whose Definition no longer exists becomes a side note of the impact.

### Lint warnings

`lint` measures the model like code, from its links alone. The thresholds are provisional.

| Code | Warns when |
|---|---|
| `lint.coupling` | a Package depends on more than 5 Packages (efferent coupling Ce) |
| `lint.unstable_dependency` | a Package depends on a less stable one: instability Ce / (Ca + Ce) of the target is higher |
| `lint.definition_size` | a Definition's body has more than 1000 words |
| `lint.package_size` | a Package has more than 25 Definitions of its own |
| `lint.unused` | nothing links to a Definition (by link or as its Kind) and no file realizes it |
| `lint.reach` | a change to a Definition outside Root would give more than half of the Managers a checklist item or a reached file, in a model of at least 6 Packages |
| `lint.thin_package` | a Package has no Definitions of its own and only one child |
| `lint.stale` | stale realization, as above |

Change hotspots from Git history come later. Coupling counts explicit edges only, not the implicit edge to the parent.

## Build decisions

Details the concept leaves open, settled while building T1:

- **Layout.** The fixed `.markitect/` layout above; registry pages mirror Package paths, Root's page is `registry.md`.
- **Library links.** A path whose first name is a Library is a Library link, so no top-level model name may equal a Library's name.
- **Local Library versions.** A local Library carries no version file; the manifest names its version and the lock pins its digest.
- **Specificity.** An exact path beats every pattern; otherwise more literal characters win; a tie between Packages is an error.
- **Stale realization** is reported, not refused (above): a registry link to a deleted Definition or an entry for a missing file does not block the Run that cleans it up. A repository file without any entry stays a compile error. Confirmed by the owner on 28 October 2026 (concept §3).
- **Inherited items** are only Definitions above that the change itself changed, not those reached through a file or a link; otherwise a README edit at Root would make every Manager work. Confirmed by the owner on 28 October 2026 (concept §6.2).
- **How far impact follows.** One step of links, and the registry in both directions. The code's own imports are not followed yet; a later step that knows the realization's languages can widen reach.
- **The required check's environment** is the container image it runs in; T3 binds how the Run's remote runs it.
- **Skills** are installed into `.claude/skills/` (Claude Code) and `.agents/skills/` (the Agent Skills folder that Codex and other agents read).

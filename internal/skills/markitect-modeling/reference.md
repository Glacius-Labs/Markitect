# Reference: the model form

A compact summary. The canonical text is `docs/guides/model-language.md` in the Markitect repository; if they differ, the guide wins.

## Layout

```text
.markitect/
  manifest.yaml        Libraries and versions (std: 1.0.0, rest: 1.0.0, ...)
  lock.yaml            Library digests, written by markitect lock
  execution.yaml       the required checks (name, command, directory, timeout, environment)
  model/               the model; this folder is the Root Package
    project.md         the project anchor (std/Project)
    <package>/<name>.md
  registry.md          Root's registry page
  registry/<package path>.md
  ignore.md            files that belong to no Package
  libraries/<name>/    Libraries that do not ship with markitect
```

Everything under `.markitect/` is accounted for by the compiler and needs no registry entry. The compiler sees what Git sees: tracked files and untracked files that Git does not ignore.

## Definitions

One Markdown file with YAML front matter.

- Front matter: `kind` (required) and `internal: true` (optional). Another key is `definition.front_matter`.
- File name without `.md` is the name, the path below `model/` is the ID (`members/member`). Names: letters, digits, `-`, `_`, starting with a letter or digit (`model.name`).
- Body: one level-1 heading, the title, first; no second one (`definition.title`); then prose. Exact values in backticks.
- The model folder holds only `.md` files and folders (`model.file`). Two entries differing only in case, or `member.md` beside `member/`, collide (`model.collision`).
- A top-level name may not be a Library name or `Kind` (`model.reserved`).

## Kinds and Libraries

A Kind is a Definition with `kind: Kind`; its body says what a Definition of that Kind contains. Every other Definition names its Kind by path.

| Kind | Use for |
|---|---|
| `std/Project` | The anchor at Root: what the project is, for whom, the Packages, what is out of scope. |
| `std/Rule` | Something every artifact in its scope must satisfy: policies, conventions, layout, quality. |
| `std/Concept` | One domain thing: what it is, its attributes, relations, states, invariants. |
| `std/Feature` | One use case: who does what, the outcome and every refusal a caller sees. |
| `std/Decision` | A choice made and kept: context, options, decision, consequences. |
| `rest/Endpoint` | One HTTP operation: title `METHOD /path`, request, each response with status and error codes; links its Feature. |
| `cli/Command` | One command as typed: arguments, flags, output, exit codes, refusals; links its Feature. |

A project writes its own Kinds as small files (`kind: Kind`) anywhere in its model; a Kind several Packages use sits at Root. A Library is a folder of `<Name>.md` Kind files under `.markitect/libraries/<name>/`, named in the manifest. Standard Libraries ship in the binary at 1.0.0. After a manifest change run `markitect lock`.

## Links, scope, visibility, cycles

- A link is `[[path]]`: the full path from the model folder, even inside the same Package (`[[members/member]]`), `[[std/Rule]]` for a Library Kind, `[[members]]` for a Package. Optional `#section` and `|label` do not matter for resolution. In a Markdown table write the label's bar as `\|`. Links in code spans and code blocks are text. No imports, no relative links.
- **Scope is location.** What a Package defines reaches its whole subtree; a Rule at Root binds the project.
- **Visibility.** Public unless `internal: true`; then only its own Package and Packages below may link to it or use it as a Kind (`link.internal`). Reading is not restricted: Managers above and checkers read internal Definitions, and registry entries may link any Definition.
- **Dependencies.** A link from Package A to a Definition in, or to, another Package B is an edge A to B; using a Kind is an edge. Every Package also has an implicit edge to its parent. Linking up to an ancestor's Definition is free. A parent may link to a child Package but not to a Definition inside it (`link.downward`). Links to Library Kinds make no edge. A cycle of edges is an error (`package.cycle`).
- Break a cycle by moving the shared concept to the common parent, or by inverting the dependency: the needing side describes its need in its own Package and the other side links to it.

## Registry pages

One page per Package: `.markitect/registry.md` for Root, `.markitect/registry/<package path>.md` for the others (`registry/members/loans.md` for `members/loans`).

```markdown
---
paths:
  - internal/members/**
---
# Registry: members

| File | Realizes | Why |
|---|---|---|
| `internal/members/member.go` | [[members/member]] | Member type |
```

- Front matter: `paths`, a list of repository-relative patterns, and nothing else (`registry.front_matter`). Root's page has no patterns.
- Patterns: `*` and `?` inside one name; a whole segment `**` matches any depth; a trailing `**` matches what is inside the folder. No `[ ]` or `{ }` (`registry.pattern`).
- Ownership: the most specific pattern wins; a pattern with no wildcard beats any with one; otherwise more literal characters win. Root owns what no one claims. Two Packages with the same pattern: `registry.duplicate_pattern`; equal specificity matching the same file: `registry.ambiguous_owner`.
- Accounting: every file outside `.markitect/` is listed on its owner's page or on the ignore page (`file.unregistered`). Listed on a page of a different owner: `registry.wrong_owner`; listed twice: `registry.duplicate_entry`; listed and ignored: `registry.ignored_file`; Git-ignored: `registry.git_ignored`; below `.markitect/`: `registry.own_file`.
- Rows: tables whose first header is `File`, three cells: file, links to the Definitions it realizes (model Definitions only; none is allowed), a short reason (`registry.table`, `registry.link`). The registry folder holds only pages (`registry.page`).
- Shared files (README, CI pipeline, compose file) belong to the lowest common parent of the Packages they serve, often Root, and list every Definition they realize.

## Ignore page

`.markitect/ignore.md`: a table whose first header is `Path`, with a pattern and a reason (`ignore.syntax`). It holds files that deliberately belong to no Package, such as `LICENSE`. `markitect init` lists the installed skill folders there.

```markdown
| Path | Why |
|---|---|
| `LICENSE` | Legal text, realizes nothing |
```

## Stale realization

A registry entry that names a deleted Definition (`stale.definition`), a missing file (`stale.file`) or a missing Package's page (`stale.page`) is not an error: `check` passes, `lint` warns (`lint.stale`), `impact` assigns the clean-up.

## Commands

All take `--repo DIR`; all but `init` and `lock` take `--json`. Exit 0 ok, 1 errors found, 2 could not run.

| Command | Use while modeling |
|---|---|
| `init [--name TITLE]` | Once per project: manifest, lock, project anchor, registry and ignore pages, skills. Never changes an existing model file. |
| `check` | Compile: errors only. |
| `lock` | Rewrite `lock.yaml` after a manifest or Library change. |
| `refs TARGET` | Who links to a Definition or Package, and which files realize it. |
| `graph` | Package dependency graph with coupling and the links behind each edge. |
| `impact --base COMMIT [--full]` | What the change since COMMIT reaches and each Manager's checklist. Needs a project that compiles. |
| `lint` | Warnings and the coupling table. |

## Warnings (`lint`)

`lint.coupling` (more than 5 dependencies), `lint.unstable_dependency` (depends on a less stable Package), `lint.definition_size` (over 1000 words), `lint.package_size` (over 25 Definitions), `lint.unused` (nothing links it, no file realizes it), `lint.reach` (a change would make more than half the Managers work), `lint.thin_package` (no Definitions and one child), `lint.stale`.

## Error codes

| Group | Codes |
|---|---|
| Project | `project.file` (a file under `.markitect/` has no place in the layout), `project.read` |
| Definition and model | `definition.front_matter`, `definition.title`, `model.name`, `model.file`, `model.collision`, `model.reserved` |
| Kind and Library | `kind.unresolved`, `kind.not_a_kind`, `library.name`, `library.file`, `library.not_a_kind`, `library.internal`, `library.version`, `library.reserved`, `library.missing`, `library.undeclared` |
| Manifest and lock | `manifest.syntax`, `manifest.missing`, `lock.mismatch`, `lock.missing`, `lock.syntax` |
| Links and Packages | `link.unresolved`, `link.syntax`, `link.internal`, `link.downward`, `package.cycle` |
| Registry and ignore | `file.unregistered`, `registry.front_matter`, `registry.page`, `registry.pattern`, `registry.duplicate_pattern`, `registry.ambiguous_owner`, `registry.wrong_owner`, `registry.duplicate_entry`, `registry.ignored_file`, `registry.git_ignored`, `registry.own_file`, `registry.table`, `registry.link`, `ignore.syntax` |
| Other | `execution.syntax`, `execution.missing`, `findings.syntax` |

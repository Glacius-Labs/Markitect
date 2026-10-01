# Documentation routers and placement

This optional Project feature checks local navigation. It does not assign Markdown resource identities, infer semantic ownership, or add graph dependencies. An adopting project owns its documentation taxonomy and any code-to-documentation correspondence rules.

## Placement while authoring

Start at a configured documentation root and follow local `README.md` routers toward the narrowest plausible owner. Compare the Project's Areas, the destination router's scope, adjacent owner contracts, and existing documents before writing. Search for a canonical source that already owns the subject; update it instead of creating a second account. Create a new document only for a distinct responsibility, then update its router and the parent router if a directory was added. If several owners remain plausible, report the candidates and the reason for the proposed placement rather than treating a path as semantic proof.

Read the local documents first and follow cross-area references when the subject requires them. Local completeness is an authoring and review heuristic, not a compiler guarantee. Router links may point anywhere useful; the router's direct-child coverage is the checked minimum, not a ban on shortcuts. A navigation link never creates a `rules`, `uses`, or file-input dependency.

## Project contract

`spec.documentation.roots` is an optional nonempty list of normalized, repository-relative POSIX directory paths. Roots must exist in the selected snapshot and must not overlap or collide by case. There are no exclusions or alternate router names. Omitting `documentation` disables router checking; Areas do not activate it implicitly. Only the local Project snapshot is checked, not pinned package archives.

```yaml
spec:
  areas:
    - name: general
      path: docs/general
  documentation:
    roots: [docs]
```

## Participating directories and coverage

The configured root always participates. Beneath it, a directory participates if at least one Markdown file (`.md`) exists at or below it in the selected source snapshot. Empty directories and directories containing only non-Markdown assets do not participate. Every participating directory needs an exact `README.md`. Its router must link to each directly contained Markdown file other than its own `README.md` and to every directly contained participating child directory. A child link may target the child directory or its `README.md`. Additional links, including deeper and cross-area references, are allowed. Generated Markdown files still count because they are visible navigation targets. The check examines snapshot paths and bytes, never the host filesystem's current state.

## Local link targets

The router checker inspects Markdown link destinations in `README.md`; ordinary non-router pages are outside this check. A destination with a URI scheme or beginning `//` is external and ignored; Windows drive paths such as `C:/page.md` or `C:\page.md` are invalid. A fragment-only link is local to the current document and needs no file lookup. For a local destination, split the query and fragment before decoding the path; percent-decode the path once (without treating `+` as a space), normalize `.` and `..` relative to the router's directory using `/`, and reject paths that escape the repository. A leading `/`, backslash, invalid percent escape, or NUL is invalid. A resulting file must be in the snapshot; a resulting directory must have a snapshot file beneath it. A fragment's heading and a query's meaning are not checked. Link targets remain navigation data and are never added to the dependency graph.

The checker recognizes ordinary inline Markdown links and reference-style links in routers, including angle-bracket destinations and optional titles. It does not interpret HTML links or bare path text. Images are not navigation links. Links in fenced or inline code are examples and are ignored. Diagnostics are sorted by path, line, code and message so equal snapshots produce equal results.

## Diagnostics

`documentation.root.missing` reports an absent configured root. `documentation.router.missing` reports a participating directory without `README.md`. `documentation.router.unlisted-directory` and `documentation.router.unlisted-file` report uncovered direct children. `documentation.link.missing` reports an absent local target, and `documentation.link.invalid` reports an unsafe or undecodable local destination. Invalid root syntax, duplicate roots and overlapping roots are strict Project parse errors with a YAML line number. Structural diagnostics do not assess the prose or the chosen owner.

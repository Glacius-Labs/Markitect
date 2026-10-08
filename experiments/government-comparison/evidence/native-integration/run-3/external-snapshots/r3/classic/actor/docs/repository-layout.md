# Repository layout

The recommended layout separates typed Markitect knowledge, human-owned documentation, generated provider entrypoints, and ordinary project artifacts. It is an authoring convention, not a new parsing constraint. Explicit Project configuration remains authoritative; existing Areas under `docs/` or other supported paths remain valid.

## Ownership and discovery

Keep `markitect.yaml` at the repository root. The CLI loads that exact Project entrypoint; `--repo` selects the project root rather than searching parent directories. Prefer `.markitect/areas/<owner>/` for new canonical Areas and organize each Area by responsibility before resource kind:

```text
markitect.yaml
.markitect/
  areas/
    engineering/
      persistence/
        migrations.rule.yaml
        migrations.workflow.yaml
    shared/
      review/
        principles.text.yaml
  packages/                     # exact pinned content archives, when needed
  review/configs/               # optional authored review configuration
docs/
  README.md
  engineering/
    README.md
    persistence.md
.agents/skills/                 # declared generated provider entrypoints
.codex/agents/
.claude/skills/
.claude/agents/
.claude/rules/
AGENTS.md                       # project-owned root instructions
CLAUDE.md                       # project-owned root instructions
src/
api/
deployment/
```

Directories appear only when they have content. Do not scaffold global empty `rules/`, `skills/`, or `workflows/` directories. Prefer a focused file for a distinct responsibility; update the existing owner when the responsibility is already modeled.

| Zone | Ownership |
|---|---|
| `markitect.yaml` | Project topology, explicit imports, checks, outputs, and package pins |
| `.markitect/areas/` | Recommended location for canonical typed YAML resources, organized by responsibility |
| `.markitect/packages/` | Optional location for immutable, explicitly pinned content archives |
| `.markitect/review/configs/` | Optional authored review configuration, referenced explicitly by the review command |
| `docs/` | Human-owned documentation and local README navigation |
| Declared provider outputs | Generated projections in native provider paths; other provider files remain project-owned |
| `AGENTS.md`, `CLAUDE.md` | Project-owned instructions and navigation |
| Ordinary artifact paths | Files referenced explicitly when needed, without being absorbed into the model |

`.markitect/` is intended for declarative, reviewable, Git-tracked inputs, not caches, logs, reports, databases, or runtime state. Do not ignore the directory wholesale. These directory names do not activate features: package archives still need exact Project pins; review configuration still needs an explicit command input.

## Explicit Areas and file names

An Area name is a DNS label of at most 63 characters, and a local resource's namespace must match its owning Area name. Area paths must be normalized relative POSIX paths and must be unique; nested Area paths are allowed. The most-specific matching Area owns a resource, while `rules` declared on matching ancestor Areas still apply. Project parsing rejects exact duplicate path strings but does not diagnose case-only differences; avoid those for cross-platform portability. Directory ancestry and adjacency do not create imports, namespaces, or dependencies.

```yaml
spec:
  areas:
    - name: shared
      path: .markitect/areas/shared
    - name: engineering
      path: .markitect/areas/engineering
      imports: [shared, documentation]
    - name: documentation
      path: docs
  documentation:
    roots: [docs]
```

`<name>.<kind>.yaml` is an optional naming convention, using a lower-case kind suffix, for example `change-review.rule.yaml`. YAML `kind` and metadata determine type and identity. No filename lint or type inference is added. Both `.yaml` and `.yml` remain supported. An existing project's configured Area path takes precedence over the recommendation.

Human-owned Markdown under `docs/` remains ordinary documentation. Its README routers describe local responsibility and direct children. Links are navigation, never `uses`, `rules`, `needs`, `implements`, or `files` declarations. Declare exact ordinary inputs through `spec.files` when their bytes affect context or impact. Source code, API descriptions, deployment files, and other artifacts stay in their project-owned locations. Markitect checks explicit inputs, hashes and changes; syntax trees, symbols, call graphs, and domain-specific interpretation remain outside the core.

Area ownership applies to ordinary input paths as well as typed resource files; it does not turn those files into Markitect resources. A resource may declare an ordinary input only when both paths belong to Areas and, across Areas, the resource's Area explicitly imports the input's Area. The example uses a `documentation` Area for `docs/` and imports it from `engineering` to make that input boundary explicit.

See the executable [repository-layout example](../examples/repository-layout/README.md) for Areas, an explicit cross-area dependency, ordinary documentation input, and a native Claude rule projection. The [minimal example](../examples/minimal/README.md) uses canonical YAML under `.markitect/areas/sample/` and explicitly selected Markdown views under `docs/markitect/`.

## Generated projections

Provider outputs remain at their native `.agents/`, `.codex/`, and `.claude/` paths and require explicit targets and mappings. Existing generated files identify Markitect and route to their sources. Edit the canonical YAML or declared project-owned source, then regenerate; never edit managed output by hand. Root `AGENTS.md` and `CLAUDE.md` remain project-owned. The [provider adapter contract](provider-adapters.md) owns exact output and inventory behavior.

Generic Markdown views are opt-in through the `markdown` target. A resource under `.markitect/areas/engineering/persistence/change-review.rule.yaml` renders to `docs/markitect/engineering/persistence/change-review.rule.md`; an existing matching kind suffix appears only once. Markitect creates managed local README routers within `docs/markitect/`. Provider entrypoints link directly to canonical YAML whether or not Markdown views are enabled.

### Source-relative prose navigation

Author Markdown links and images in resource `spec.text` and Markdown descriptions relative to the canonical YAML file's directory. Rendering resolves that URL in the explicit local snapshot, then makes it relative to the concrete output. A generic Markdown view maps a link to a known local typed YAML resource to that resource's selected central view. Copied provider Agent text keeps typed links aimed at canonical YAML; ordinary document and image links are rebased to the actual source file in both projections. Query strings and fragments retain their destination semantics. Schemed URLs, site-root URLs, empty paths and local anchors remain unchanged.

An old sibling companion URL is recognized only from an exact local resource path by replacing its YAML extension with `.md`. An existing unmarked ordinary Markdown file at that path wins. A missing or Markitect-marked sibling can route to the selected view (or canonical YAML in copied provider text); multiple typed candidates for a used companion alias produce an error. Renaming a canonical file still requires updating source links; the renderer does not guess old names, resolve package archive paths against consumer files, add graph edges, or declare ordinary inputs from prose.

The destination scanner handles inline links and images, reference definitions and URI-escaped targets. It preserves surrounding Markdown, titles and code examples. Raw HTML attributes are not projected. This is navigation rewriting, not a complete Markdown parser or a check of every ordinary link target. Relative destinations that escape the repository or contain non-portable path components fail projection. The executable [Markdown navigation example](../examples/markdown-navigation/README.md) checks the source-relative contract and actual selected targets.

For GitHub review presentation, use `.gitattributes` entries scoped to outputs the project actually owns:

```gitattributes
.agents/skills/**/SKILL.md linguist-generated=true
.codex/agents/*.toml linguist-generated=true
.claude/skills/**/SKILL.md linguist-generated=true
.claude/agents/*.md linguist-generated=true
.claude/rules/*.md linguist-generated=true
docs/markitect/** linguist-generated=true
```

Use these globs only when every matching file is generated; otherwise list exact paths. Broad `.agents/**`, `.codex/**`, or `.claude/**` entries are appropriate only if the entire matching tree is generated. They can otherwise hide hand-authored configuration or policy. The dedicated `docs/markitect/**` tree is managed when the Markdown target is selected; ordinary `docs/**` and canonical YAML remain human-owned sources. Initialization does not edit `.gitattributes` or `.gitignore`.

## Compatibility and adoption

| Decision | Current implementation |
|---|---|
| Recommended Areas | `.markitect/areas/<owner>/`, organized by responsibility; no mandatory migration |
| Initialization | Omitting `--path` selects `.markitect/areas/<namespace>`; an explicit supported path still overrides it |
| Root discovery and graph | Unchanged; no implicit relationships or new Project fields |
| File naming | Optional convention; YAML remains authoritative |
| Provider paths and managed headers | Existing native paths and generation contract retained |
| Markdown views | No output by default; existing `spec.targets` now accepts `markdown` and writes under `docs/markitect/` |
| Tool distribution | Current pins use `.markitect/tool/` and `.markitect/bootstrap/`; immutable older releases keep their historical paths |
| Layout lint and migration helper | Deferred until a demonstrated invariant or adoption need justifies them |

The new default affects initialization plans that omit `--path`; earlier versions required that flag. Explicit invocations such as `--path docs/<namespace>` retain their paths. Preview the exact plan before writing. Existing Project configuration is loaded as before.

An adopting project may deliberately move its own canonical files: change its Area paths, update exact file/mapping/navigation references, regenerate Markdown views and provider outputs, remove obsolete generated files in the reviewed candidate, and run its checks. Moving files changes snapshot paths and context fingerprints. Compile fresh evidence; do not assume path-independent review reuse or automatic cleanup. Markitect does not perform this migration on the project's behalf.

When enabling Markdown views in an existing Project, add the target, render into a committed candidate, review the new files, and deliberately remove old generated files in the same migration. Current source pins use `.markitect/tool/` and `.markitect/bootstrap/`; an installed older release retains its own pin contract. Source changes do not update an installed release; release verification and publication follow [Operations](operations.md).

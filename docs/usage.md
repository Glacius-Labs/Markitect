# Using Markitect

This guide describes the current source model. Consult [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions and their source versions; the [roadmap](implementation-plan.md) owns current source status.

## Project and resource model

Markitect projects declare YAML resources in `markitect.yaml` and configured content areas. The namespaced kinds are `Text`, `Rule`, `Contract`, `Workflow`, `Skill`, and `Agent`; `Project` declares topology and execution/output configuration. Namespaces and names are DNS labels. Paths, area imports, and resource declarations govern ownership and direct access.

Use `rules` to attach requirements, `uses` for concrete resource dependencies, `needs` to require a Contract, `implements` to declare a Contract signature, and Project `bindings` to select implementations. Use `files` for exact ordinary UTF-8 project artifact inputs needed by a resource. Markitect does not infer dependencies from prose links or from the contents of project artifacts.

See [Project artifact inputs](documentation.md) for the file-input context and impact contract and its executable example, and [Architecture](architecture.md#project-artifact-boundary) for the boundary on domain-specific analysis.
See [Documentation routers](documentation-routers.md) for optional local navigation checks and the separate authoring placement procedure.
See [Repository layout](repository-layout.md) for the recommended `.markitect/areas/` convention and compatibility boundaries. Explicit configured paths remain valid.

A Project can declare named areas, direct imports, renderer targets, and rule adapters. References must resolve to authored resources. The following neutral shape shows those relationships; copy the [minimal example](../examples/minimal/README.md) for a runnable project:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: sample-project
spec:
  areas:
    - name: shared
      path: .markitect/areas/shared
    - name: sample
      path: .markitect/areas/sample
      imports: [shared]
      rules:
        - kind: Rule
          namespace: shared
          name: change-review
  checks:
    - name: unit-tests
      run: [go, test, ./...]
  documentation:
    roots: [docs]
  targets: [codex, claude]
  ruleAdapters:
    review-context:
      - kind: Rule
        namespace: shared
        name: change-review
```

`ruleAdapters` maps generated rule entrypoint names to source references; these mappings produce Claude rule outputs when the Claude target is selected. `targets` accepts `markdown`, `codex`, and `claude`. Generic Markdown resource views are opt-in through the `markdown` target; without it, canonical YAML is the only Markitect-owned documentation source. Markdown views live under `docs/markitect/<area>/` with each resource's source-relative subdirectory and a filename ending in its actual kind. Generated local README navigation stays within that generated root. Provider outputs link directly to canonical YAML, independent of Markdown views. See [Repository layout](repository-layout.md) and [Provider adapters](provider-adapters.md) for paths and ownership. A Project may also pin direct offline content packages in `spec.packages`. Each entry contains `name`, `version`, `source`, `archive`, and `sha256`; this list is the complete content lock. `markitect.lock.yaml` remains exclusively the CLI distribution lock. See [Content packages](content-packages.md) for manifests, exports, archive creation, query selection, and boundaries. The [Project schema](../schema/Project.yaml) and [Agent schema](../schema/Agent.yaml) list the supported fields. Provider metadata configures generated output; it is not a runtime dependency of the Markitect core.

`spec.documentation.roots` opts into snapshot-based README router checks. It does not turn Markdown into typed resources or infer dependencies. Omit it when the project has no router contract. A passing router check says that local navigation is structurally complete; it does not establish that a document was placed with the correct semantic owner.

## Initialize a project

For an existing repository that has no `markitect.yaml`, `init` previews a minimal Project and one area README:

```powershell
markitect init --repo . --name project-name --namespace owner
```

The preview shows the exact `markitect.yaml` content and both file paths. Initialization does not choose checks, rules, resources, output targets, packages, templates, or edits to root documentation. The selected area path must not already exist. Preview is read-only and can run outside Git.

Without `--path`, the area is `.markitect/areas/owner` (using the selected namespace). The plan creates only `markitect.yaml` and `.markitect/areas/owner/README.md`, plus their required directories. Existing layouts remain supported: use `--path docs/ai` or another supported repository-relative path to select a different area. Earlier versions required `--path`; existing explicit invocations continue to select the same area.

After reviewing the plan, `--write` recomputes it and validates the prospective Project with the normal parser, graph, and output checks before exclusive file creation:

```powershell
markitect init --repo . --name project-name --namespace owner --write
```

Writing requires a named non-protected Git branch and the shared write lock. In a Git checkout, `--repo` must name the Git worktree root; preview also works outside Git. The repository root must exist; names and the area path must be safe, and initialization rejects aliases, excluded or unsafe paths, existing configuration or area content, and deleted or staged tracked targets. Only `markitect.yaml` and the selected area's `README.md` are created; the selected area directory must be absent and is created exclusively. The result lists `written` files and `createdDirectories`, including on a partial failure. Inspect those paths and the recovery instructions before retrying; no cleanup is automatic, and a directory may have acquired other content. This is not a multi-file transaction. Structural validation can pass with no Project checks. `verify` remains incomplete until the project owner declares real checks and commits the candidate.

## Declare verification commands

`Project.spec.checks` is optional for structural authoring. `verify` requires at least one declared check; if none are configured, it returns `incomplete-evidence` rather than treating the project as verified.

Each check has a stable name and a `run` argv list. The first element is a bare executable name resolved through `PATH`; it cannot be a path. Use an interpreter as the executable for repository scripts, for example `[go, run, tools/check-docs.go]`. Arguments are passed directly; no shell parses variables, pipes, redirects, wildcards, or command chaining. Declare the gates needed for the project's verification question. `verify --revision COMMIT` materializes and runs the exact committed Project and commands within the execution limits in [Operations](operations.md).

Commands run with the caller's local authority. Snapshot isolation fixes the input tree but is not an operating-system sandbox. Checks should be read-only and limited to the intended project.

## Render

For project-owned Codex, Claude, and shared entrypoints, declare `spec.targets` and, when needed, `spec.providerAdapters`; see [Provider adapters](provider-adapters.md). For opt-in sourced functional assertions, declare `spec.consistency`; see [Factual consistency](consistency.md). Both are absent by default.

`render` checks or writes only the managed output families selected by Project `spec.targets` and `spec.ruleAdapters`. The `markdown` target produces opt-in resource views under `docs/markitect/`; Codex and Claude outputs use their native provider paths. No generic Markdown view is created by default. Provider outputs route directly to canonical YAML and do not depend on the Markdown target. An undeclared output is not covered.

## Commands

| Command | Purpose |
|---|---|
| `check` | Validate YAML, the resource graph, declared file inputs, managed outputs, and explicitly configured documentation routers. |
| `init` | Preview a minimal project plan; `--write` creates only its Project file and one area README. |
| `verify` | Check an immutable revision and run the Project's declared commands. |
| `inventory` | List Markdown candidates and typed resources; it does not infer dependencies. |
| `find` | Search valid resources by literal text and exact optional filters. |
| `explain` | Show a resource's canonical path, area, and direct relationships. |
| `authoring` | Compile core authoring guidance embedded in the executable. |
| `context` | Compile an entry closure and declared files, or add a fixed-run task snapshot and exact project artifact paths from a committed manifest. |
| `impact` | Compare two immutable revisions and report changed paths and affected resources. |
| `review` | Record an advisory report or determine whether its declared inputs permit reuse. |
| `render` | Check or write explicitly selected Markdown and provider outputs. |
| `format` | Check or write canonical YAML formatting. |
| `schema` | Check or write generated schema YAML. |
| `pack` | Build a deterministic offline content ZIP from a fixed Git revision and emit a suggested Project pin. |
| `package` | Build a deterministic source archive and YAML tool lock. |
| `bundle` | Build a complete distribution from an exact revision. |
| `install` | Validate a distribution and preview a complete pin change; `--write` applies the plan. |
| `version` | Print the CLI version and platform. |
| `licenses` | Print bundled third-party notices; works offline without a Project. |

Existing-content imports are implemented and reviewed as scripts owned by the project being migrated. Markitect core does not include a migration command or presume source documentation structure.

Content packages are loaded from exact committed archive bytes in the selected Project snapshot. Markitect does not fetch the `source` coordinate, resolve ranges, load nested packages, activate imported Rules, or write imported resources. The package contract was introduced in v0.3.0 and remains part of the current source model; see [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for distributions that include it.

## Upgrade from v0.1.0

The old Project `profile` field is removed from the current model. Replace profile-based verification with explicit checks:

```yaml
spec:
  checks:
    - name: validation
      run: [go, test, ./...]
```

Review each command and its arguments as project-owned policy. There is no automatic translation from a legacy profile setting to a check, no implicit bootstrap check, and no inferred Python or provider gate. Add import scripts to the project repository and review their output as ordinary source changes.

When a schema-changing upgrade makes an older revision invalid under the current CLI, current-version `impact` and review reuse cannot cross that schema boundary. Keep evidence from the old version as historical. Upgrade configuration and content in a complete candidate commit, then run the current checks, compile fresh context, and complete a new review as needed; that candidate starts a new evidence baseline. Do not relax current parsing or invent a cross-version impact result to bridge the boundary.

Rendering produces only explicitly selected outputs. Add the `markdown` target when the project wants generic resource views, and add provider targets and rule adapters for the provider outputs it owns. Compare generated files, run the declared checks against the exact candidate, and commit configuration, content, and generated outputs together. Never edit an immutable release to represent a new version.

## Fixed snapshots and advisory review

Omitting `--revision` uses the working tree and marks results provisional. A commit revision selects one immutable Git tree. `impact` needs both a base and candidate revision.

Context reports its snapshot and selected-input digests; these identify bytes, not semantic truth. Impact conservatively includes old and new dependencies, and unknown or unmodelled inputs can broaden its result. Review reuse requires matching executable, configuration, context, and eligible impact. The CLI does not call a model, judge a report, authenticate its author, or transfer human acceptance.

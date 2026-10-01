# Using Markitect

This guide describes the current source model. Consult [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions and their source versions; the [roadmap](implementation-plan.md) owns current source status.

## Project and resource model

Markitect projects declare YAML resources in `markitect.yaml` and configured content areas. The namespaced kinds are `Text`, `Rule`, `Contract`, `Workflow`, `Skill`, and `Agent`; `Project` declares topology and execution/output configuration. Namespaces and names are DNS labels. Paths, area imports, and resource declarations govern ownership and direct access.

Use `rules` to attach requirements, `uses` for concrete resource dependencies, `needs` to require a Contract, `implements` to declare a Contract signature, and Project `bindings` to select implementations. Use `files` for exact ordinary UTF-8 file inputs needed by a resource. Markitect does not infer dependencies from prose links.

See [Documentation inputs](documentation.md) for the file-input context and impact contract and its executable example.

A Project can declare named areas, direct imports, renderer targets, and rule adapters. References must resolve to authored resources. The following neutral shape shows those relationships; copy the [minimal example](../examples/minimal/README.md) for a runnable project:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: sample-project
spec:
  areas:
    - name: shared
      path: docs/shared
    - name: sample
      path: docs/sample
      imports: [shared]
      rules:
        - kind: Rule
          namespace: shared
          name: change-review
  checks:
    - name: unit-tests
      run: [go, test, ./...]
  targets: [codex, claude]
  ruleAdapters:
    review-context:
      - kind: Rule
        namespace: shared
        name: change-review
```

`ruleAdapters` maps generated rule entrypoint names to source references; these mappings currently produce Claude rule views when the Claude target is selected. `targets` currently accepts `codex` and `claude`. A Project may also pin direct offline content packages in `spec.packages`. Each entry contains `name`, `version`, `source`, `archive`, and `sha256`; this list is the complete content lock. `markitect.lock.yaml` remains exclusively the CLI distribution lock. See [Content packages](content-packages.md) for manifests, exports, archive creation, query selection, and boundaries. The [Project schema](../schema/Project.yaml) and [Agent schema](../schema/Agent.yaml) list the supported fields. Provider metadata configures generated output; it is not a runtime dependency of the Markitect core.

## Initialize a project

For an existing repository that has no `markitect.yaml`, `init` previews a minimal Project and one area README:

```powershell
markitect init --repo . --name project-name --namespace owner --path docs/ai
```

The preview shows the exact `markitect.yaml` content and both file paths. Initialization does not choose checks, rules, resources, output targets, packages, templates, or edits to root documentation. The selected area path must not already exist. Preview is read-only and can run outside Git.

After reviewing the plan, `--write` recomputes it and validates the prospective Project with the normal parser, graph, and output checks before exclusive file creation:

```powershell
markitect init --repo . --name project-name --namespace owner --path docs/ai --write
```

Writing requires a named non-protected Git branch and the shared write lock. In a Git checkout, `--repo` must name the Git worktree root; preview also works outside Git. The repository root must exist; names and the area path must be safe, and initialization rejects aliases, excluded or unsafe paths, existing configuration or area content, and deleted or staged tracked targets. Only `markitect.yaml` and the selected area's `README.md` are created; the selected area directory must be absent and is created exclusively. The result lists `written` files and `createdDirectories`, including on a partial failure. Inspect those paths and the recovery instructions before retrying; no cleanup is automatic, and a directory may have acquired other content. This is not a multi-file transaction. Structural validation can pass with no Project checks. `verify` remains incomplete until the project owner declares real checks and commits the candidate.

## Declare verification commands

`Project.spec.checks` is optional for structural authoring. `verify` requires at least one declared check; if none are configured, it returns `incomplete-evidence` rather than treating the project as verified.

Each check has a stable name and a `run` argv list. The first element is a bare executable name resolved through `PATH`; it cannot be a path. Use an interpreter as the executable for repository scripts, for example `[go, run, tools/check-docs.go]`. Arguments are passed directly; no shell parses variables, pipes, redirects, wildcards, or command chaining. Declare the gates needed for the project's verification question. `verify --revision COMMIT` materializes and runs the exact committed Project and commands within the execution limits in [Operations](operations.md).

Commands run with the caller's local authority. Snapshot isolation fixes the input tree but is not an operating-system sandbox. Checks should be read-only and limited to the intended project.

## Render

`render` checks or writes Markitect's generic managed views. Additional outputs are produced only for Project `spec.targets` and `spec.ruleAdapters`. Codex and Claude outputs are product-owned adapters, selected only when the matching target is declared. No additional compatibility output is selected implicitly. Inspect the generated schema and the current target list; an undeclared output is not covered.

## Commands

| Command | Purpose |
|---|---|
| `check` | Validate YAML, the resource graph, declared file inputs, and managed outputs. |
| `init` | Preview a minimal project plan; `--write` creates only its Project file and one area README. |
| `verify` | Check an immutable revision and run the Project's declared commands. |
| `inventory` | List Markdown candidates and typed resources; it does not infer dependencies. |
| `find` | Search valid resources by literal text and exact optional filters. |
| `explain` | Show a resource's canonical path, area, and direct relationships. |
| `authoring` | Compile core authoring guidance embedded in the executable. |
| `context` | Compile the dependency closure and declared file inputs for one entry. |
| `impact` | Compare two immutable revisions and report changed paths and affected resources. |
| `review` | Record an advisory report or determine whether its declared inputs permit reuse. |
| `render` | Check or write generic views and explicitly declared render outputs. |
| `format` | Check or write canonical YAML formatting. |
| `schema` | Check or write generated schema YAML. |
| `pack` | Build a deterministic offline content ZIP from a fixed Git revision and emit a suggested Project pin. |
| `package` | Build a deterministic source archive and YAML tool lock. |
| `bundle` | Build a complete distribution from an exact revision. |
| `install` | Validate a distribution and preview a complete pin change; `--write` applies the plan. |
| `version` | Print the CLI version and platform. |

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

Rendering now defaults to generic views. Add an explicit target and, where needed, a rule adapter for every additional output the project owns. Compare generated files, run the declared checks against the exact candidate, and commit configuration, content, and generated outputs together. Never edit an immutable release to represent a new version.

## Fixed snapshots and advisory review

Omitting `--revision` uses the working tree and marks results provisional. A commit revision selects one immutable Git tree. `impact` needs both a base and candidate revision.

Context reports its snapshot and selected-input digests; these identify bytes, not semantic truth. Impact conservatively includes old and new dependencies, and unknown or unmodelled inputs can broaden its result. Review reuse requires matching executable, configuration, context, and eligible impact. The CLI does not call a model, judge a report, authenticate its author, or transfer human acceptance.

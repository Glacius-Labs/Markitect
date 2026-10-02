# Using Markitect

This guide records the current source CLI and the v0.10.0 target language and adapter contracts as implementation progresses. The v0.9.1 release contains the earlier built-in resource model; the v0.10.0 target loads declared Domain definitions before typed resources. Consult [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions and their source versions; the [roadmap](implementation-plan.md) owns current source status and completion.

## Project and resource model

Markitect Projects declare YAML resources in configured content areas. The bundled AI-working Domain supplies `Text`, `Rule`, `Contract`, `Workflow`, `Skill`, and `Agent`. A Project may load additional versioned Domains from exact local snapshot-relative files or explicitly pinned package members through `spec.domains`; those definitions provide closed resource schemas, typed relations, and bounded constraints. `Project` declares topology and execution/output configuration. Each resource's API version and kind identify its Domain-qualified type. Namespaces and names are DNS labels. Paths, area imports, and resource declarations govern ownership and direct access.

### Define and load a Domain

A Domain declares its API version, resource kinds and properties, typed relationships, and constraints. The Project loads the exact definition file before validating resource instances:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata:
  name: software
spec:
  apiVersion: example.org/v1
  kinds:
    Module:
      inputsField: sourceFiles
      required: [intent, dependsOn]
      properties:
        intent:
          type: string
        sourceFiles:
          type: array
          items:
            type: string
        dependsOn:
          type: array
          items:
            type: ref
            refKind: Module
  relations:
    dependsOn:
      field: dependsOn
      sourceKinds: [Module]
      targetKinds: [Module]
      context: true
      invalidate: true
      acyclic: true
  constraints:
    - name: module-intent
      select:
        kind: Module
      assert:
        op: present
        field: intent
```

The Project selects the definition explicitly:

```yaml
spec:
  domains: [domains/software.yaml]
```

`spec.domains` accepts a sequence of unique `.yaml` or `.yml` file paths relative to the selected source snapshot. A Domain from a directly pinned package can be selected as `package:PIN/DOMAIN-MEMBER`; `DOMAIN-MEMBER` must be declared in that package's `spec.domains` (it need not be a resource export). Packages do not activate Domains implicitly. The Project loader registers selected definitions first, then validates resource documents and their references.

A resource uses the Domain's `apiVersion` and one of its declared kinds. Its relation field is a typed reference and is evaluated under the named relation's graph rules:

```yaml
apiVersion: example.org/v1
kind: Module
metadata:
  name: survey
  namespace: intake
spec:
  intent: Own survey intake and response behavior.
  sourceFiles: [src/Survey/Survey.csproj]
  dependsOn:
    - kind: Module
      namespace: platform
      name: core
```

References in custom Domain resources resolve within that same Domain API version; their typed references and named relations cannot cross into another Domain. The bundled AI-working Domain has explicit `uses` relationships that can target a registered custom Domain when the reference names its qualified `apiVersion`. Relations separately declare context traversal, invalidation, and cycle behavior. A Domain constraint selects resources by its own kind and exact metadata labels, then applies a closed assertion operator. Supported operators are `present`, `equal`, `allowed`, `allowed-targets`, `count`, and `unique`; `count` requires at least one nonnegative `min` or `max`. `allowed-targets` checks actual relation target kinds against the assertion's `values`, which must be within the relation descriptor's declared `targetKinds`. These checks apply only to declared model data and selected inputs.

An optional `inputsField` on a kind names one declared array-of-string property whose paths are exact opaque project artifact inputs. Markitect snapshots those files and uses their byte changes for context and impact; it does not parse their domain-specific structure.

Package content does not activate its Domain. Select a package-owned definition explicitly with `package:PIN/DOMAIN-MEMBER` after adding the exact package pin; the member must be declared by the package manifest's `spec.domains`. Local Domain files and activated package Domain bytes are fixed context inputs; edits to them affect impact. `format` canonicalizes local Domain definitions and resource YAML. See the [canonical engineering plan](canonical-engineering-plan.md) for the full language and adapter guarantees.

The [canonical engineering example](../examples/canonical-engineering/README.md) demonstrates independent Software and Delivery Domains, relation-specific context and impact, a policy value used in documentation, agent context, and deterministic checking, and a read-only adapter for concrete project inputs.

For the bundled AI-working Domain, use `rules` to attach requirements, `uses` for concrete resource dependencies, `needs` to require a Contract, `implements` to declare a Contract signature, and Project `bindings` to select implementations. Other Domains own their own typed fields and relations. `inputsField` explicitly maps a Domain property containing exact file paths into opaque artifact inputs. Markitect does not infer relations from prose links or analyze the contents of project artifacts.

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

### Model output and adapters

`model` emits the normalized semantic model as versioned YAML for adapters and inspection. It includes fixed snapshot and configuration identity, Domain definitions and their source digests, generic resources with qualified identity and provenance, and resolved relationships with their declared graph effects. It does not expose raw authoring YAML or the Go graph representation. `context`, `impact`, `explain`, and configured consumers use the same resolved meaning.

The v0.10.0 target declares adapter entries under `spec.adapters`, each with `name`, `type`, `version`, and adapter-specific `config`. Mappings are explicit canonical inputs. Use only adapter types and configuration fields documented for the adapter version. A command adapter uses exact snapshot `inputs`, argv for its supported lifecycle stages, plugin-owned `parameters`, and an explicit non-secret `target` identity whenever apply is enabled. Keep credentials out of Project mappings and saved plans; command adapters use their own secure external credential lookup. The built-in local `markitect-render` adapter needs no `spec.adapters` entry; it observes outputs selected by existing Project targets and mappings. A new observation can report drift even when source resources have not changed. The current working-tree CLI exposes both local and configured command adapters; the published v0.9.1 binary does not include the v0.10.0 contract.

`reconcile` separates observation and planning from writes:

```powershell
New-Item -ItemType Directory -Force .artifacts/markitect/reconcile | Out-Null
markitect reconcile --repo . --action observe --adapter markitect-render
markitect reconcile --repo . --action plan --adapter markitect-render > .artifacts/markitect/reconcile/plan.yaml
markitect reconcile --repo . --action apply --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml --write
markitect reconcile --repo . --action verify --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml
```

`observe` and `plan` write results only to standard output. Save the plan in `.artifacts/markitect/reconcile/` before applying. Apply is limited to the working tree and requires the unchanged plan plus explicit `--write`; it rejects stale inputs or changed plan content. Verification checks outputs after application. Plans do not automatically remove stale files. External command adapter inputs, protocol, output limits, and local-authority boundary are detailed in the [adapter contract](provider-adapters.md#command-adapter-contract).

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
| `check` | Validate Domain definitions, YAML resources, relations, constraints, declared file inputs, managed outputs, and explicitly configured documentation routers. |
| `init` | Preview a minimal project plan; `--write` creates only its Project file and one area README. |
| `verify` | Check an immutable revision and run the Project's declared commands. |
| `inventory` | List Markdown candidates and typed resources; it does not infer dependencies. |
| `find` | Search valid resources by literal text and exact optional filters. |
| `explain` | Show a resource's canonical path, area, and direct relationships. |
| `model` | Emit the normalized semantic model, its provenance, and resolved relation descriptors. |
| `authoring` | Compile core authoring guidance embedded in the executable. |
| `context` | Compile the selected resource closure, activated Domain definitions, and declared file inputs; a committed run manifest can also select a fixed task snapshot and exact artifact paths. |
| `impact` | Compare two immutable revisions and report changed inputs and affected resources under relation-specific rules. |
| `review` | Record an advisory report or determine whether its declared inputs permit reuse. |
| `reconcile` | Observe a configured target, plan exact changes, apply a selected plan, or verify its result. |
| `render` | Check or write explicitly selected Markdown and provider outputs. |
| `format` | Check or write canonical formatting for resources and loaded local Domain definitions. |
| `schema` | Check or write generated schema YAML. |
| `pack` | Build a deterministic offline content ZIP from a fixed Git revision and emit a suggested Project pin. |
| `package` | Build a deterministic source archive and YAML tool lock. |
| `bundle` | Build a complete distribution from an exact revision. |
| `install` | Validate a distribution and preview a complete pin change; `--write` applies the plan. |
| `version` | Print the CLI version and platform. |
| `licenses` | Print bundled third-party notices; works offline without a Project. |

Existing-content imports are implemented and reviewed as scripts owned by the project being migrated. Markitect core does not include a migration command or presume source documentation structure.

Content packages are loaded from exact committed archive bytes in the selected Project snapshot. Markitect does not fetch the `source` coordinate, resolve ranges, or load nested packages. An imported resource or Domain does not become active until explicitly selected; package resources are read-only. The package contract was introduced in v0.3.0 and remains part of the current source model; see [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for distributions that include it.

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

Context reports its snapshot and selected-input digests, including activated Domain definitions. The `model` command binds normalized resources and relations to snapshot and configuration digests. These identify the bytes and declarations evaluated, not semantic truth. Impact follows each declared relation's context and invalidation rules; set-based constraints include their selected members, and unknown or unmodelled inputs can broaden its result. Adapter observations have separate source and observation digests, so external drift can be detected under unchanged canonical inputs. Review reuse requires matching executable, configuration, context, and eligible impact. The CLI does not call a model, judge a report, authenticate its author, or transfer human acceptance.

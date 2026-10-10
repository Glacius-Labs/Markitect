# Using Markitect

This page describes the Project/Domain engineering-resource compiler and historical versioned CLI contracts. Explicit source additions and removals are labelled below; published releases retain their actual immutable bytes. It is not the recommended path for new model-first projects. Start with the [model-first project workflow](project-workflow.md); it documents the current `markitect project` source surface, its source-versus-release distinction, implemented commands and planned journey stages. The project model in a committed revision is the accepted repository specification Markitect uses for that revision; a draft is a proposal, and a commit or digest does not authenticate human approval.

The published [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.14.1) preserves Project/Domain behavior and bundles a separate canonical Projection alpha. New projects should not start with these legacy contracts. Current source does not promise executable backwards compatibility. Source changes do not update an installed binary. The [roadmap](implementation-plan.md) owns exact release and evidence status.

Current source needs Git 2.40 or later for the model-first project workflow; see [Project workflow](project-workflow.md#connect-and-inspect).

<a id="legacy-projectdomain-cli-compatibility"></a>
## Structural and historical Project/Domain CLI reference

The compiler, check, schema, render and artifact-accounting commands serve Markitect's own canonical engineering resources. Versioned histories preserve what older releases shipped; historical age alone is not a reason to retain a function. Do not combine `markitect.yaml` / `.markitect/areas/` resources with the current `.markitect/project.yaml` / `.markitect/model/` project model. No automatic conversion is implied; migration is an explicit project-owned activity with reviewed meaning and ownership.

<a id="canonical-brownfield-adoption-unreleased-source"></a>
The experimental canonical Projection alpha bundled with v0.14.1, including canonical brownfield adoption and `copy-me --action infer`, has been removed from current source. Its guide remains at the [v0.14.1 tag](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/canonical-projections.md).

<a id="selective-adoption-preparation-and-copy-me-unreleased-source"></a>
## Selective adoption preparation and Copy Me (v0.13.0)

These commands are included in v0.13.0 as bounded, versioned byte/reference infrastructure; immutable v0.12.0 has the earlier Copy Me guidance, not this durable handoff. Existing greenfield `init` is unchanged. No Project, Domain or ContextRun is required to prepare opaque engineering evidence. The [contract](design/selective-adoption-handoff.md) owns selection, identity, privacy and retention semantics.

Create an owner-supplied scope YAML, for example:

```yaml
apiVersion: markitect.example.org/adoption-scope/v1alpha1
id: architecture-review
purpose: Review ownership conventions in selected engineering guidance
review: owner-supplied-scope-review-2026-10-04
privacy:
  constraints: Public-safe evidence only; no source secrets or personal records
  allowExcerpts: false
retention: Owner removes the external workspace after this review
repositories:
  - id: product
    root: C:/projects/product
    commit: REPLACE_WITH_OWNER_SELECTED_FULL_LOWERCASE_COMMIT
    paths:
      - path: docs/architecture.md
        reason: Canonical boundary guidance
      - path: docs/decisions/legacy-service.md
        reason: Explicit counterevidence
    exclusions:
      - path: private
        reason: Withheld by owner
coverage:
  - id: ownership
    repository: product
    question: Is ownership consistently explicit in this selected sample?
    state: uninspected
    reason: Capture precedes interpretation
```

The commit placeholder must be replaced with an exact 40- or 64-digit commit. Paths are literal, case-sensitive Git tree names, not directories or globs. Preparation checks the entire selected list before opening its blobs. A working-tree edit is not evidence. A newer commit is a new capture even when only unselected files changed; the selected-only content digest may remain equal.

```powershell
markitect prepare --scope C:/review/scope.yaml --output C:/review/capture-1
# Review the emitted handoff and copy its handoff.digest:
markitect prepare --scope C:/review/scope.yaml --output C:/review/capture-1 --write --expect REVIEWED_HANDOFF_DIGEST
```

The destination is an absent absolute directory whose parent exists, outside every selected repository and its Git metadata. Preview writes nothing. Write recomputes the capture, checks the exact digest, and creates the directory/files exclusively. It persists `handoff.yaml` and selected bytes under `evidence/<repository-id>/<exact-path>`. Partial creation is reported without rollback; reruns refuse an existing workspace. Access and retention are owner responsibilities; Windows ACLs are not set by this command.

Selected Git objects must already be available locally. Preparation blocks remote transports and automatic lazy fetching, so a partial clone with missing evidence is refused instead of silently downloading objects into the source repository.

Prepare separate candidate files and one `copy-me-queue/v1alpha1` YAML with evidence entries, `candidates: [{stableID, path, digest}]`, preserved coverage IDs/repositories/questions and optional requests. Coverage state/reason may advance as an interpretive claim; the report retains original preparation coverage as well. Candidate paths are literal relative paths beneath the queue directory; hashes bind raw candidate-file bytes. The [executable fixture](../examples/selective-adoption/README.md) provides complete queue, candidate and decision examples and a runnable lifecycle.

```powershell
markitect copy-me --workspace C:/review/capture-1 --queue C:/review/dossier/queue.yaml
markitect copy-me --workspace C:/review/capture-1 --queue C:/review/dossier/queue.yaml --decision C:/review/dossier/decision.yaml
```

The report supplies raw handoff/queue/candidate hashes for an explicit human-supplied decision. A decision binds one candidate revision, queue bytes, handoff bytes and handoff identity, plus supplied reviewer/date/rationale/scope and `accept|reject|defer|split|revise`. Changed selected bytes fail handoff validation; changed candidate/queue/handoff bytes stale an old decision. Conflicts, uncertainty and coverage remain visible. A request for another repository path grants no read permission: the owner must supply a new preparation scope.

Both commands exit 0 for completed capture or byte/reference validation, 2 for refusal, invalid input or stale binding. Neither authenticates the owner/reviewer, infers policy, evaluates discovery quality, calls a model or automatically adopts an accepted candidate. Copy Me does not read Git or broaden the captured evidence. Only a later explicit reviewed project change can create canonical architecture.

## Project and resource model

The published v0.13.0 executable adds explicit read-only policy-failure analysis, described below. The immutable v0.12.0 executable does not support this option.

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

References in custom Domain resources resolve within that same Domain API version; their typed references and named relations cannot cross into another Domain. The bundled AI-working Domain has explicit `uses` relationships that can target a registered custom Domain when the reference names its qualified `apiVersion`. Relations separately declare context traversal, invalidation, and cycle behavior. A Domain constraint selects resources by its own kind and exact metadata labels, then applies a closed assertion operator. The published v0.11.0 operators are `present`, `equal`, `allowed`, `allowed-targets`, `count`, and `unique`. v0.12.0 adds `same-target`; `count` requires at least one nonnegative `min` or `max`. `allowed-targets` checks actual relation target kinds against the assertion's `values`, which must be within the relation descriptor's declared `targetKinds`. These checks apply only to declared model data and selected inputs.

An optional `inputsField` on a kind names one declared array-of-string property whose paths are exact opaque project artifact inputs. Markitect snapshots those files and uses their byte changes for context and impact; it does not parse their domain-specific structure.

Package content does not activate its Domain. Select a package-owned definition explicitly with `package:PIN/DOMAIN-MEMBER` after adding the exact package pin; the member must be declared by the package manifest's `spec.domains`. Local Domain files and activated package Domain bytes are fixed context inputs; edits to them affect impact. `format` canonicalizes local Domain definitions and resource YAML. See the [canonical engineering plan](canonical-engineering-plan.md) for the full language and adapter guarantees.

The [canonical engineering example](../examples/canonical-engineering/README.md) demonstrates independent Software and Delivery Domains, relation-specific context and impact, a policy value used in documentation, agent context, and deterministic checking, and a read-only adapter for concrete project inputs.

## Published v0.11.0 architecture-contract capabilities

v0.11.0 treats exact-pinned package Domains as reusable architecture contracts. Existing package declarations and Domain activation remain explicit; updating a contract means reviewing an exact package pin change, running `check` and fixed-snapshot `impact`, and inspecting PolicyResults. Markitect will not rewrite application source as part of a package update.

When a Project selects `markdown`, the Domain projection generates normalized schema, relation, and constraint views at `docs/markitect/_domains/*.domain.md`. Relevant per-resource PolicyResults, including a waived result's recorded decision, appear in generated resource views. These generated files remain derived views; canonical Domain definitions and policy stay in the selected package or Project source.

## v0.12.0 resolved-target equality

v0.12.0 adds `same-target` after the v0.11.0 baseline. This assertion compares two fixed paths of named relations from each selected subject; it does not execute a graph query. Each path has one or two steps, all in the selected Domain. `select.kind` is required. Undefined relations, incompatible kind transitions, wildcard target kinds and disjoint terminal kinds are rejected in the Domain definition. No field expressions, collection traversal or implicit applicability are supported.

```yaml
- name: deployment-products-agree
  select:
    kind: Deployment
  assert:
    op: same-target
    left: [deploysService, belongsToProduct]
    right: [deploysTo, belongsToProduct]
```

Every step must have exactly one declared reference and one resolved target. Missing, unresolved, wrong-kind or ambiguous steps are structural errors with no equality PolicyResult; an exception cannot waive them. Declared relation cycle rules still apply independently; bounded traversal introduces no mixed-relation cycle prohibition. Fully resolved paths produce one per-subject `passed` or `failed` result by comparing their final canonical GraphKeys. A failed result can be waived only for that exact subject and constraint under the existing exception rules.

Both passed and failed results expose `comparison.left` and `comparison.right`: ordered `relations`, edge `steps` with `from`, `to`, `relation`, Domain API and source path/resource line, and the final `target`. Join the result's API/constraint to the normalized Domain and exact `domainInputs` provenance to explain selection and rule origin. Constraint digests include both paths and their relation definitions. For this operator, the subject digest also binds the canonical contents of all traversed resources and the resolved identities; it conservatively becomes stale even when a changed input does not alter the final equality. Source location metadata does not determine equality.

Impact derives policy-read dependencies from both paths, including valid prefixes of incomplete paths and the old and new snapshots. These dependencies do not insert Context edges or change relation flags. Context includes comparison evidence for its included subjects, but resources along a policy-only path enter the context closure only through existing `context: true` relations. See the [design assessment](design/resolved-target-equality.md), [Delivery fixture](../examples/delivery-target-equality/README.md) and [versioned Software fixture](../examples/software-architecture/README.md). Software Feature ownership uses an explicit label cohort; an unlabeled Feature-bearing UseCase is outside that policy.

The `count.scope` values are `resource` and `selection`. If omitted, `scope` defaults to `selection`. A selection-scoped count without a `relation` counts selected resources; with a `relation`, it counts the total relation targets across the selected resources. A resource-scoped count requires a `relation` and evaluates its number of targets separately for each selected subject. Selection boundaries must include additions and removals. PolicyResults report `passed`, `failed`, or `waived` per constraint and subject, exposed by the normalized model, `check`/`verify`, and relevant context output.

This excerpt is the v2 fixture's per-UseCase Validator constraint. The `hasValidator` relation supplies the exact field and target kind counted by the assertion:

```yaml
apiVersion: "markitect.example.org/v1alpha1"
kind: "Domain"
metadata:
  name: "engineering-constitution-v2"
spec:
  apiVersion: "engineering.markitect.org/v1beta1"
  kinds:
    UseCase:
      required: ["validators"]
      properties:
        validators:
          type: "array"
          items: {type: "ref", refKind: "Validator"}
    Validator:
      required: ["summary"]
      properties:
        summary: {type: "string"}
  relations:
    hasValidator:
      field: "validators"
      sourceKinds: ["UseCase"]
      targetKinds: ["Validator"]
      context: true
      invalidate: true
  constraints:
    - name: "each-usecase-has-validator"
      select: {kind: "UseCase"}
      assert:
        op: "count"
        scope: "resource"
        relation: "hasValidator"
        min: 1
```

This is an illustrative excerpt from the v2 fixture; the complete Domain also declares its other kinds and relations. `constraintDigest` binds the constraint and the relation semantics it depends on, not just the relation's name. Changing the field, endpoint kinds, or graph behavior makes an old exception stale. Do not calculate or guess digests. After changing the package pin and resource API versions, run `model`; it emits the failing `PolicyResult` before returning a nonzero status. Copy the matching result's `constraintDigest` and `subjectDigest` exactly. The failed `check` report also includes PolicyResults. The values below are deliberately invalid placeholders until replaced with those output strings:

```yaml
spec:
  policyDate: "2026-10-02"
  policyExceptions:
    - name: "validator-transition"
      apiVersion: "engineering.markitect.org/v1beta1"
      constraint: "each-usecase-has-validator"
      subject: "engineering/engineering.markitect.org/v1beta1/UseCase/create-order"
      constraintDigest: "sha256:<COPY-EXACT-CONSTRAINTDIGEST-FROM-FAILED-RESULT>"
      subjectDigest: "sha256:<COPY-EXACT-SUBJECTDIGEST-FROM-FAILED-RESULT>"
      rationale: "Keep the existing UseCase unblocked while its Validator slice is implemented."
      owner: "architecture-owner"
      decision: "Approved as a temporary, reviewable architecture exception."
      expiresOn: "2026-10-30"
```

The example subject is the exact GraphKey used by the fixture; for another project copy the `subject` from its failed PolicyResult. `policyDate` is a frozen as-of value. An exception with this `expiresOn` is expired when `policyDate` reaches that date. After adding a Validator and removing the exception, rerun `check`; never keep an exception once its result passes.

`Project.spec.policyExceptions` is limited to 64 entries. Each entry binds `name`, `apiVersion`, `constraint`, exact subject GraphKey, `constraintDigest`, `subjectDigest`, `rationale`, `owner`, and `decision`. Optional `expiresOn` uses `YYYY-MM-DD` and requires an explicitly frozen `policyDate` carried as as-of snapshot evidence. An exception expires when `policyDate >= expiresOn`; no ambient wall-clock value is used. Waived PolicyResults retain the original violation and expose rationale, owner, decision, expiry, and policy date. Exceptions may waive only per-resource constraints; unknown, stale, unused, expired, or malformed entries fail. No exception can waive types/schema, references, cycles, selection-wide policies, relation bounds, or global structural checks. These fields record a declared rationale and decision; they do not authenticate a person or establish approval.

The Copy Me workflow guides an authoring agent to analyze only explicitly selected fixed-snapshot files with recorded hashes and present observations, hypotheses, counterexamples, and uncertainty in a separate candidate for human review. Only an explicit reviewed source/package-pin change adopts a candidate. This is authoring guidance, not a new discover CLI or a model call in Markitect's deterministic core. Code-level claims such as Query non-mutation and Handler/Test/Docs file presence require declared inputs plus project-owned checks or configured adapters; the core will not infer those facts. See the [engineering constitution](engineering-constitution.md) for complete scope and examples.

To copy the runnable v0.11.0 fixtures from the repository root into separate temporary work areas:

```powershell
$runRoot = Join-Path $env:TEMP ("markitect-v011-" + [guid]::NewGuid().ToString("N"))
$null = New-Item -ItemType Directory -Path $runRoot
$constitution = Join-Path $runRoot "engineering-constitution"
Copy-Item -Recurse .\examples\engineering-constitution $constitution
go run ./src/cmd/markitect check --repo $constitution

$discoveryProject = Join-Path $runRoot "engineering-discovery-project"
$discoveryDossier = Join-Path $runRoot "engineering-discovery-dossier"
Copy-Item -Recurse .\examples\engineering-discovery\project-template $discoveryProject
Copy-Item -Recurse .\examples\engineering-discovery\dossier-template $discoveryDossier
```

The constitution fixture starts at package v1; the isolated test exercises updating the exact pin to v2, the failing model result, a source-bound waiver, and removal of the waiver after implementation: `go test ./src/harness/examples -run '^TestEngineeringConstitutionV2PackageMigrationAndWaiverLifecycle$' -count=1`. The discovery fixture continues from the copied templates with the [engineering-discovery pilot instructions](../examples/engineering-discovery/README.md), which freeze the Project commit, run `context --run context-run.yaml`, and show how to fill the separate evidence dossier. Neither fixture makes the proposal or waiver self-approving.

The core language remains a finite set of deterministic assertions. CUE and OPA are evaluation candidates only if repeated real architecture policies exceed that language's readable expressive power; a specialist would be an explicit adapter. See the [engineering constitution](engineering-constitution.md#constraint-language-and-specialist-engines) for official references and the trigger. No performance or market claim is implied.

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

The v0.12.0 model also exposes `domainInputs[].apiVersion` and `name`. Join a PolicyResult's `apiVersion` and `constraint` to the normalized Domain's selector/assertion, and its API version to the exact Domain input path, package/version and digest. This explains both applicability and rule origin without activating another Domain implicitly. Context Domain inputs carry `domainApiVersion` and `domainName`; resource inputs expose the reachable incoming context relations in `via`. Impact `causes` explains changed resource/input seeds, old/new invalidation relationships and conservative project-wide causes. See the [software architecture experiment](design/software-architecture-experiment.md) for concrete output and replay steps. These explanation fields are included in v0.12.0 and were not part of the published v0.11.0 executable.

`model` emits the normalized semantic model as versioned YAML for adapters and inspection. It includes fixed snapshot and configuration identity, Domain definitions and their source digests, generic resources with qualified identity and provenance, and resolved relationships with their declared graph effects. It does not expose raw authoring YAML or the Go graph representation. `context`, `impact`, `explain`, and configured consumers use the same resolved meaning.

The v0.10.0 contract declares adapter entries under `spec.adapters`, each with `name`, `type`, `version`, and adapter-specific `config`. Mappings are explicit canonical inputs. Use only adapter types and configuration fields documented for the adapter version. A command adapter uses exact snapshot `inputs`, argv for its supported lifecycle stages, plugin-owned `parameters`, and an explicit non-secret `target` identity whenever apply is enabled. Keep credentials out of Project mappings and saved plans; command adapters use their own secure external credential lookup. The built-in local `markitect-render` adapter needs no `spec.adapters` entry; it observes outputs selected by existing Project targets and mappings. A new observation can report drift even when source resources have not changed. The command adapter executes with the local caller's authority and is not an operating-system sandbox; bounded evidence does not imply semantic truth or human acceptance.

`reconcile` separates observation and planning from writes:

```powershell
New-Item -ItemType Directory -Force .artifacts/markitect/reconcile | Out-Null
markitect reconcile --repo . --action observe --adapter markitect-render
markitect reconcile --repo . --action plan --adapter markitect-render > .artifacts/markitect/reconcile/plan.yaml
markitect reconcile --repo . --action apply --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml --write
markitect reconcile --repo . --action verify --adapter markitect-render --plan .artifacts/markitect/reconcile/plan.yaml
```

`observe` and `plan` write results only to standard output. Save the plan in `.artifacts/markitect/reconcile/` before applying. Apply is limited to the working tree and requires the unchanged plan plus explicit `--write`; it rejects stale inputs or changed plan content. Verification checks outputs after application. Plans do not automatically remove stale files. External command adapter inputs, protocol, output limits, and local-authority boundary are detailed in the [adapter contract at v0.14.1](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/provider-adapters.md#supported-command-adapter-contract).

<a id="project-local-projections-current-source-not-yet-released"></a>
### Project-local projections (experimental alpha bundled with v0.14.1)

A Project can register one `local-projection` adapter to bind exact contract and artifact-coverage configuration paths. The `projection` command observes and plans those declared representations, applies a reviewed candidate only with explicit write authorization, and verifies them at an immutable revision. `markitect check` can report structural and observed projection state, but it does not establish AI semantics or replace the configured verification checks. See the [projection guide](projections.md) for registration, complete AI candidates, digest review, verification, and partial-write recovery. This API is bundled in v0.14.1 as an experimental alpha capability; it is not part of stable support.

<a id="canonical-controller-actions-source-only-alpha"></a>
<a id="declared-scope-completion-audit-source-only-alpha"></a>
<a id="goal-led-modeling-cli-source-only-alpha"></a>
<a id="retained-evidence-refresh-source-only-alpha"></a>
The canonical controller, completion audit, goal-led modeling and evidence refresh actions belonged to the canonical Projection alpha bundled with v0.14.1 and have been removed from current source; the [v0.14.1 usage guide](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/usage.md#canonical-controller-actions-source-only-alpha) describes them.

## Initialize a project

Current source uses `markitect project init`; see the [model-first commands](project-operations.md#cli-setup-and-current-roots). The top-level initializer has been removed from current source. Versioned release histories retain the commands they actually shipped.

## Declare verification commands

`Project.spec.checks` is optional for structural authoring. `verify` requires at least one declared check; if none are configured, it returns `incomplete-evidence` rather than treating the project as verified.

The optional per-check budget below is an unreleased source follow-up. The published v0.14.1 CLI retains its fixed 600-second check limit and does not accept `timeoutSeconds` on Project checks. The source candidate accepts up to 5400 seconds per check.

Each check has a stable name and a `run` argv list. Its optional integer `timeoutSeconds` selects 1 through 5400 seconds, with 600 seconds when omitted. For example, `{name: go-tests, timeoutSeconds: 5400, run: [go, test, ./..., -count=1, -timeout=60m]}` explicitly bounds both Markitect's check process and Go's test binaries. Each YAML result reports the effective `timeoutMilliseconds`; timeout remains incomplete evidence. The first argv element is a bare executable name resolved through `PATH`; it cannot be a path. Use an interpreter as the executable for repository scripts, for example `[go, run, tools/check-docs.go]`. Arguments are passed directly; no shell parses variables, pipes, redirects, wildcards, or command chaining. Declare the gates needed for the project's verification question. `verify --revision COMMIT` materializes and runs the exact committed Project and commands within the execution limits in [Operations](operations.md).

Commands run with the caller's local authority. Snapshot isolation fixes the input tree but is not an operating-system sandbox. Checks should be read-only and limited to the intended project.

## Render

For project-owned Codex, Claude, and shared entrypoints, declare `spec.targets` and, when needed, `spec.providerAdapters`; see [Provider adapters](provider-adapters.md). For opt-in sourced functional assertions, declare `spec.consistency`; see [Factual consistency](consistency.md). Both are absent by default.

`render` checks or writes only the managed output families selected by Project `spec.targets` and `spec.ruleAdapters`. The `markdown` target produces opt-in resource views under `docs/markitect/`; Codex and Claude outputs use their native provider paths. No generic Markdown view is created by default. Provider outputs route directly to canonical YAML and do not depend on the Markdown target. An undeclared output is not covered.

## Commands

| Command | Purpose |
|---|---|
| `check` | Validate Domain definitions, YAML resources, relations, constraints, declared file inputs, managed outputs, and explicitly configured documentation routers. |
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

<a id="read-only-analysis-of-failed-policies-unreleased-source"></a>
### Read-only analysis of failed policies (v0.13.0)

Ordinary `context` and `impact` remain strict. To investigate a structurally valid candidate whose ordinary PolicyResults fail, use the explicit `--analyze-policy-failures` option:

```powershell
markitect context --repo . --revision <candidate-commit> --api-version <domain-api> --namespace <namespace> --kind UseCase --name <name> --analyze-policy-failures
markitect impact --repo . --base <base-commit> --revision <candidate-commit> --analyze-policy-failures
```

The output has an `analysis` marker with per-snapshot structural, policy and validation status, failed-result count, and configuration/model identity. A policy-failing candidate remains `validationStatus: failed`. **Retain stdout on exit 1:** completed analysis returns 1 if either analyzed snapshot has failed policies, including a failing base with a repaired candidate. It returns 0 only when analysis completed and neither side has unwaived failures. Invocation, acquisition or compilation errors use exit 2; structural diagnostic reports retain their failure exit and contain no Context/Impact analysis payload.

Context follows the normal declared closure and includes its applicable PolicyResults, exact Domain/package inputs, declared files and relation `via` evidence. The header reports overall candidate status without adding unrelated failing subjects or documentation to the selected closure. Policy dependencies do not become Context edges. The diagnostic context digest binds its analysis status and normalized model identity.

Impact retains `changed`, the conservative `affected` set and its `causes`. It additionally reports changed PolicyResults and their direct subjects separately, with both-side rule/source identities and the applicable selector/assertion. An absent result is represented as analysis-only `not-applicable`, with a reason distinguishing an undefined constraint, absent subject, unselected result or changed resource/collection result scope; it is not a passed PolicyResult. Direct subjects describe evaluation changes, not a guaranteed implementation-change cohort. Package/configuration changes can still broaden the conservative set far beyond those subjects.

Malformed schemas/resources, unresolved or wrong-kind references, relation bounds, prohibited cycles, invalid singleton traversal, input failures and invalid/stale exceptions block analysis. No exception is required to inspect an ordinary failure. Exceptions remain explicit governance decisions. `check`, `verify`, reconciliation and writes retain their strict behavior; this option is rejected for those commands and for `context --run`. Analysis is not acceptance or verification. See the [design contract](design/policy-failure-analysis.md) and the [preserved-bundle replay](../experiments/policy-failure-analysis/README.md).

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

<a id="markitect-first-release-candidate"></a>
## Markitect-first (published v0.13.0)

The provider-neutral [Change workflow](../src/internal/host/embedded/resources/workflow-markitect-first-change.yaml) is canonical. `markitect authoring` includes it without requiring a Project. An adopting owner can put a durable pointer to that version-bound command and the project's selected resource Context in its root agent guidance; selected provider Skills are derived through existing targets. Markitect's own [root Project](../markitect.yaml) uses `development/Skill/engineering-change`; its human-owned AGENTS.md is navigation, not a second workflow owner. Greenfield Init remains unchanged.

Begin with a full BASE commit: `markitect check --repo PATH --revision BASE`, then select context with `markitect context --repo PATH --revision BASE --namespace OWNER --kind Skill --name ENTRY`. Use find/explain/model to select a defensible owner. Classify implementation-only, engineering-intent change, policy migration or ambiguity. Intent changes update canonical desired state before technical implementation; commit/check the intended candidate, run fixed impact, then implement. Implementation-only work follows existing intent without artificial model edits. Failed ordinary policy is inspectable only through explicit read-only diagnostic options; strict acceptance and structural blockers remain unchanged.

### Managed artifact accounting

The standalone source-package helper is built with `go build -o BIN/markitect-check-artifacts ./src/cmd/markitect-check-artifacts`. It is included in the source distribution, not a separate release binary asset. Use an exact source-package pin and Go matching the package requirement; never silently execute an unrelated installed helper. Declare it as an explicit project check, for example `run: [markitect-check-artifacts, --repo, ., --config, markitect-artifacts.yaml]`, with the reviewed helper on PATH. Markitect dogfood uses the equivalent `go run` argv from its own fixed source snapshot.

`markitect-artifacts.yaml` is a separate closed `markitect.example.org/artifact-coverage/v1alpha1` / `ArtifactCoverage` document:

```yaml
apiVersion: markitect.example.org/artifact-coverage/v1alpha1
kind: ArtifactCoverage
spec:
  roots: [src, docs/engineering, .markitect]
  tooling:
    - path: .markitect/coverage.yaml
      owner: project-artifact-accounting
  exclusions:
    - path: src/vendor/retained.txt
      reason: retained third-party fixture outside canonical ownership
```

The example config must be saved at the named tooling path and selected with `--config .markitect/coverage.yaml`. Roots are literal file/directory paths; no overlapping roots, globs or implicit whole-repository scope. Exclusions are exact existing files with reasons; subtree exclusions are deliberately unsupported. Derived canonical sources, resolved resource inputs and renderer outputs need no second declaration. Other managed files require an exact tooling owner or exclusion. Unknown, stale, orphaned, colliding or aliased paths fail. Exit 0 means accounted configured paths, 1 means findings, 2 means configuration/acquisition failure. This is path accounting; it does not prove source semantics or generated byte freshness.

Run the helper directly before committing to detect untracked managed files. Run it through `markitect verify --revision CANDIDATE` to bind coverage to the materialized fixed snapshot. Existing `check` supplies generated-byte comparison; configured specialist checks and adapters retain their own proof limits. Reconciliation remains observe/plan, review an input-bound saved plan, explicit apply with `--write`, verify and an empty converged plan. Complete only with valid desired state, declared check evidence, accounted scope, converged configured outputs and no hidden owner decision. The [design](design/managed-artifact-coverage.md) states source-loader limits; [vision](vision.md) owns the product thesis, not a claim of measured benefit.

## Independent Module checks (current source)

The consolidated source adds the read-only helper `src/cmd/markitect-check-modules`; it is not a published v0.13.0 binary asset. Build it from a reviewed source package or run it from this checkout:

```powershell
go run ./src/cmd/markitect-check-modules --repo . --revision <full-commit> --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config
```

At least one exact configuration path is required; an omitted Module is reported as `not-configured`. Omitting `--revision` selects the provisional working-tree snapshot. Configuration and artifact paths must already be exact declared inputs of canonical resources. Host loads and strictly compiles one model, supplies selected bytes/ownership facts, and composes the two independent Modules. The report binds snapshot, model and configuration digests. Exit 0 means configured checks passed, 1 means findings, and 2 means invalid invocation/input or compilation failure. There is no write/install mode.

The [checked-in configurations](../.markitect/modules) demonstrate closed Module-owned formats. They use the `.config` suffix so the existing authoring loader does not mistake ordinary configuration for a canonical YAML resource. Hooks check declared entrypoint bytes and ownership; the tracked pre-commit file is not installed. Pipelines check exact artifact bytes, ownership, and configured YAML scalar locations against explicit check argv joined with spaces. They do not prove shell semantics, pipeline execution or complete provider coverage. The [Module guide](development/modules.md) owns implementation boundaries; the [consolidation report](validation/clean-architecture-consolidation.md) owns validation evidence.

# Repository layout

Current source uses canonical language v1, explicit input selection and exact artifact registrations. [Project workflow](project-workflow.md) describes preparation; [Architecture](architecture.md) explains responsibility boundaries.

## Recommended target-project layout

```text
.markitect/
  model.yaml                 canonical manifest
  model-lock.yaml            explicit dependency lock
  model/                     canonical Definition tree and local Packages
  execution.yaml             finite declarations and named profile selections
  ignore.yaml                optional exact explained file exceptions
  registry/
    registry.yaml            ordinary root artifact ownership
    package/registry.yaml    local Package artifact ownership
  state/                     run registration and writer lock; outside census
docs/markitect/project.md    official generated readable view
```

Model granularity and artifact locations remain the author's choice. Canonical dependency manifests are explicitly selected; they are not discovered under an assumed directory. The operator supplies an explicit Orchestrator profile document for requested metadata preparation. No automatic scaffold, configuration migration or execution is implied.

The future private SQLite journal lives outside the repository; `.markitect/state/` holds only its registration and writer lock. See the [journal contract](design/canonical-model/manager-apply-journal.md#physical-store-and-commit-contract). Current read-only preparation creates no journal.

## Ownership and boundaries

The canonical model owns intended behavior. Registry shards assign exact artifact paths to one responsibility and existing Definitions. Execution declarations own operational selections and ceilings. Embedded product defaults have their own versioned source. Orchestrator metadata owns literal executable/protocol observations. Captured canonical and control inputs are protected separately; an exact ignore cannot override input protection or satisfy missing realization.

## Source repository layout

Markitect has one root Go module:

```text
cmd/markitect/                 product entrypoint
cmd/markitect-release/         independent release maintenance entrypoint
cmd/markitect-check-*/         repository gate entrypoints
internal/model/               canonical v1 compiler
internal/snapshot/            pure fixed tree values
internal/organization/        pure topology/context/settings
internal/realization/         pure ownership/accounting/candidates
internal/apply/               preparation and consumer ports
internal/modeldocs/           generated view and write plan
internal/adapters/            explicit repository/profile/document adapters
internal/bootstrap/           actual product composition
internal/buildinfo/           build version
internal/tooling/             explicit independent maintenance services
internal/testkit/             hermetic test-only support
integration/                  standalone public bootstrap
tools/smoke/                  source distribution consumer checks
```

The [code map](development/code-map.md) lists every actual Go package and its owning guide. Nested product modules and unknown responsibility paths fail source gates. Tests live with their consumers. Independent canonical fixtures and the Library oracle retain their original bytes.

The self-model is [model.yaml](../.markitect/model.yaml), [model-lock.yaml](../.markitect/model-lock.yaml) and `.markitect/model/`; [project.md](markitect/project.md) is its officially generated view. [AGENTS.md](../AGENTS.md) routes work. `.github/` owns hosted workflows and `.githooks/` the staged-tree pre-commit check. Keep new build caches and logs outside governed source inputs; Gitignore does not filter the census.

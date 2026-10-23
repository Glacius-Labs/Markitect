# Repository layout

Current source uses canonical language v1, explicit input selection and exact artifact registrations. [Project workflow](project-workflow.md) describes preparation; [Architecture](architecture.md) explains responsibility boundaries.

## Recommended target-project layout

```text
.markitect/
  model.yaml                 canonical manifest
  model-lock.yaml            explicit dependency lock
  model/                     canonical Definition tree and local Packages
  execution.yaml             v2 Package instructions and scoped required checks
  ignore.yaml                optional exact explained file exceptions
  registry/
    registry.yaml            Root's artifact shard
    package/registry.yaml    local Package shard, maintained by its Manager
docs/markitect/project.md    official generated readable view
```

Model granularity and artifact locations remain the author's choice. Canonical dependency manifests are explicitly selected; they are not discovered under an assumed directory. The operator supplies an explicit Runtime profile document for requested metadata preparation. No automatic scaffold, configuration migration or execution is implied.

Standalone Run keeps durable candidates, terminal records and project Notes outside the repository in the explicitly selected state root. Temporary Manager and Git environments close before successful return; stored data can be reopened independently. See the [Run](../internal/run/README.md) and [workspace service](../internal/adapters/gitrepo/WORKSPACES.md). S3 source composes public Apply, independently reopened stored Verify approval and guarded publication. Native Apply qualification remains separate and Extract is retired. Read-only preparation creates no Run state.

## Ownership and boundaries

The canonical model owns intended behavior. Registry shards assign exact artifact paths to one responsibility and existing Definitions. Execution v2 declarations own Package instructions and scoped checks. Outer runtime configuration owns native/check profiles; invocation configuration owns finite limits and normal account-login references. Static metadata owns literal image/protocol observations. Captured canonical and control inputs are protected separately; an exact ignore cannot override input protection or satisfy missing realization.

## Source repository layout

Markitect has one root Go module:

```text
cmd/markitect/                 product entrypoint
cmd/markitect-release/         independent release maintenance entrypoint
cmd/markitect-check-*/         repository gate entrypoints
internal/model/               canonical v1 Model check
internal/snapshot/            pure fixed tree values
internal/organization/        pure topology/context/settings
internal/realization/         pure ownership/accounting/candidates
internal/project/             structural preparation and candidate validation
internal/run/                 closed complete Manager hierarchy and its ports
internal/runtimeconfig/       acquired static native/check profile envelopes
internal/harness/             pure model-derived native instructions
internal/apply/               command preflight, closed workflow and durable publication
internal/workflow/            direct complete Assessment, Implement and Verify sequence
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

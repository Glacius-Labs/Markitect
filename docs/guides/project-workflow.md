# Project workflow

`check`, `context` and guarded `docs` consume structural Project preparation and start no Agent. `apply` runs Assessment, Implement and Verify, each over the complete model-derived Manager hierarchy, and publishes only the independently reopened stored Verify approval after all Run resources close. Extract, Cockpit, interactive authoring and Setup are not part of V1. The [architecture](../architecture.md) explains the responsibilities and the [delivery plan](../implementation-plan.md) owns implementation status and evidence authority.

## Connect and inspect

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context PACKAGE --repo .
```

The default manifest is `.markitect/model.yaml`. `--model` selects another exact manifest; repeat `--dependency PATH` for every dependency. There is no folder or registry discovery. The manifest and lock satisfy the [canonical v1 contract](../design/language-contract.md).

An omitted revision observes the actual draft, including Gitignored files. `--revision REF` resolves once and observes that exact commit. Neither preview supplies a publication guard. Public fixed previews refuse unbound present-day external inputs. Closed Runs retain already captured external model/instruction bytes in their candidate storage and use those exact values for fixed candidate observation; they do not reread mutable external paths or vendor them into product files.

## Model and readiness

`check` defaults to the Model check, registry, complete file accounting, authored execution declarations and static contexts. `context <package>` reports selected Definitions, provenance and inclusion reasons for that Package, including ordinary Root `.`.

`markitect.execution/v2` declares ordered Manager instruction lists, ownership, scope and project checks. Runtime profiles, provider settings and invocation limits have separate owners. `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH` additionally prepares closed static native/check metadata from acquired bytes. The [Codex](../../internal/adapters/codexappserver/README.md) and [check](../../internal/adapters/checks/README.md) adapters own their exact profile versions and supported metadata. Image preparation starts no provider or project command. Host `process` check profiles are retired. Static success establishes no native runtime capability.

## Project layout

The layout below is what Markitect recommends for a project that uses it. [Architecture](../architecture.md) explains responsibility boundaries; the layout of Markitect's own repository is in [Repository layout](../development/repository-layout.md).

### Recommended layout

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

Run keeps durable candidates, terminal records and project Notes outside the repository in the explicitly selected state root. Temporary Manager and Git environments close before a Run returns; stored data can be reopened independently. See the [Run](../../internal/run/README.md) and [workspace service](../../internal/adapters/gitrepo/README.md#run-git-workspace-service). Native Apply qualification is separate evidence. Read-only preparation creates no Run state.

### Ownership and boundaries

The canonical model owns intended behavior. Registry shards assign exact artifact paths to one responsibility and the Definitions they realize. When the owner deletes a Definition, its links become stale: `check` still passes and lists them under `stages.structural.files.accounting.stale`, Apply's Implement has the shard owner remove or realign them, and no candidate or publication keeps one. Removing or merging a whole Package works the same way: `check` lists its shard under `stages.structural.registry.orphaned`, the Manager that now owns each artifact's declared area (Root where none does) registers it in its own shard, and Root deletes the old shard. Execution v2 declarations own Package instructions and scoped checks. Outer runtime configuration owns native/check profiles; invocation configuration owns finite limits and normal account-login references. Static metadata owns literal image/protocol observations. Captured canonical and control inputs are protected separately; an exact ignore cannot override input protection or satisfy missing realization.

## Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

Read `view` in the preview. `canonicalModelDigest` identifies model meaning; `contentDigest` hashes rendered Markdown; `previewDigest` binds observed sources, target bytes/existence/mode and generated content. Use only the returned `previewDigest` with `--expect`; changed sources or targets require a fresh preview. Fixed-revision views never write. Protected model/control/instruction inputs cannot be targets. The independent document guard records actual completed paths and read-back.

Previews may include explicit external sources. `docs --write` requires every acquired model, control and instruction source to be repository-bound because external target disjointness has not been established.

## Explicit Runtime and invocation files

Copy [runtime-profile.example.yaml](../../examples/runtime-profile.example.yaml) and [run.example.json](../../examples/run.example.json) outside the repository and replace every uppercase placeholder. Source configuration does not qualify a provider.

- `ENGINE_EXECUTABLE` is the explicit native absolute engine path. Escape backslashes in JSON and quote YAML paths correctly.
- `IMAGE_ID`, `AGENTD_BUILD` and `CODEX_SCHEMA_DIGEST` identify the pinned image and its actual observed build metadata. `ACCOUNT_MODEL` selects the normal account model; check profile names match declared Project checks.
- `ACCOUNT_LOGIN_PATH` is the dedicated normal-account login JSON outside the realization tree. The `example-account` reference matches both files. The provider owns login provisioning and expiry behavior.
- `STATE_DIRECTORY` is an absolute path outside the repository. `PRODUCT_BUILD` is the operator's build assertion; retain the frozen source SHA separately.
- `DEADLINE_UTC` is an absolute RFC3339 UTC deadline ending in `Z`. Manager count, messages, provider idle timeout, cleanup and persistence all have finite positive limits. `providerIdleTimeoutSeconds` bounds provider work without a served tool call; time Markitect spends serving the Manager's tool calls, such as waiting for its children, does not count, and each served call restarts it.

The [example tests](../../internal/bootstrap/configuration_examples_test.go) substitute both exact files, route the explicit Apply invocation through the CLI, and prepare the captured Runtime with controlled image metadata. They check native format 4 settings, the shared account reference, and declared Check requests, then stop at a deliberate boundary before runtime startup or credential reads. A separate negative case keeps invalid preflight free of metadata and state acquisition. These are static composition and CLI transport tests; actual account login and Apply execution require separate native evidence.

## Execute, verify, and apply

Use explicit committed selections and finite invocation limits:

The target must be on an isolated non-protected branch, for example `git switch -c dev/realization`, with its tracked baseline committed. Apply checks this existing write policy before opening any Manager resources and again during publication. Accounted ignored inputs remain part of the captured baseline.

```powershell
& "ABSOLUTE_MARKITECT_BINARY" apply --repo "REPOSITORY" --runtime-profile "ABSOLUTE_RUNTIME_YAML" --run-config "ABSOLUTE_RUN_JSON"
```

Use the same quoted binary without `&` in a POSIX shell. `--model` and repeated `--dependency` select committed inputs. `--revision` and `--metadata` remain preparation options. A successful preflight starts real Run environments, provider Sessions and required checks. Exactly one executable Codex account profile is selected by its presence in the captured Runtime document; every profile uses the invocation engine.

The [Run](../../internal/run/README.md) gives every Manager its own fresh environment and Git copy, direct delegation and reviewed upward integration. An untracked ELF file in a Manager's copy that no registry entry names by exact path, such as a `go build` executable, is a build output and operational state that never enters a candidate; every Implement check and the Run record list these files, and Verify receives those of the Implement Run. Every other file in the copy, a registered binary included, needs its registration, exact ignore entry or input role. A Manager asks only its direct parent or its direct children; siblings align through their common parent, and nobody outside the Run is asked. Each Implement child submission reaches its parent with its deterministic check result; an invalid one goes back to the child only through the parent's rejection with Findings. A material missing decision or contradiction produces a model blocker; correct the model outside the Run. There is no answer command or preloaded owner decision. Assessment judges model readiness and Approve always leads to Implement, even for no-change. Only Findings that name a Definition of the current checked model reach Implement; Assessment approvals and Verify judgments keep the others as Shared Notes ([DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model)). Implement submits a candidate; fresh Verify alone can approve publication. Every Verify sees the complete diff against the command's bound input and, when a stored closed Verify approval covers exactly that input under the same model, execution configuration and dependencies, the host fact that it was already verified. Under that fact, Verify approves a changed candidate only when Root's approval names the defect the change corrects as a Finding with a current Definition, which the Run record keeps; otherwise any change is a Reject. Verify Reject carries its Findings, each naming a current Definition, into fresh Implement and Verify Runs with the original shared allowance and deadline. Assessment Reject, model blockers and technical or closure failures stop publication.

### Invocation configuration

The closed UTF-8 JSON uses `formatVersion: 4`. Required fields are `repositoryIdentity`, `productBuild`, `stateDirectory`, `engineExecutable`, `deadline`, `maxManagers`, `maxMessages`, `providerIdleTimeoutSeconds`, `cleanupTimeoutSeconds`, `persistenceTimeoutSeconds` and `credentials`. Optional `workDeadline` reserves part of the whole deadline and must precede it. Unknown, duplicate, null, relative-path and invalid values are refused. Old strategy/role, ownerRequest, helper, harvest and ownerDecisions fields are not accepted. There are no ambient defaults. See the exact [example](../../examples/run.example.json).

Format 3 is refused with a message that names the change. Its `messageTimeoutSeconds` bounded all provider work of one response; format 4's `providerIdleTimeoutSeconds` bounds only provider work without a served tool call, so no limit silently changes its meaning. Convert a format 3 file by setting `formatVersion: 4` and choosing `providerIdleTimeoutSeconds` for the new meaning.

The example allows 600 seconds of provider work without a served tool call. Time the host spends serving the Manager's tool calls, such as awaiting children, does not count, and every served call restarts the limit, so a Manager that keeps delegating, awaiting and reviewing is not stopped by it; the overall deadline still bounds the whole Apply. In Phase C, Root's single Implement response hit the earlier whole-response limit six times while it kept delegating. One observed successful native Library message exchange alone exceeded the previous 120-second example limit; 600 seconds was the finite bound used in that native evidence. Configure this limit for the project within the original overall deadline and message allowance; the example does not guarantee that every stretch of provider work will fit.

The message allowance counts messages to Managers. A question costs nothing of its own: it is charged only as part of the message that delivers it, together with the receiver's other waiting events. Each Manager has at most 8 open questions. Before a correction pair after a Verify Reject, Apply checks that the remaining allowance can start Implement and Verify with every Manager once each; otherwise it ends incomplete with the cause "budget does not cover a correction pair".

### Dedicated credential references

Each exact credential reference selects a provider, mode and absolute native file. Codex accepts `dedicated-runtime-login`. There is no API-key route. Parsing and static metadata preparation do not read credentials. Only the native resolver reads the selected regular file with path/link checks, safe errors and a 1 MiB bound. The provider owns normal-login validation and the session copy; source files are not rotated by the resolver.

### Results and evidence

The CLI names the project state directory and an immutable `<state>/<project>/applies/<id>.json` report. Run records live under `<state>/<project>/runs/`, candidates under `candidates/`, Notes under `notes/` and native transcripts under `transcripts/<apply-id>/`. The retained Apply report and printed JSON use the same final status/outcome projection. Each Run persists the exact input reference, Project binding, technical and semantic outcomes, required findings, handoffs, actual check receipts and closure observations, plus concrete native identity observations when available. Candidates are independently verified self-contained bundles; external captured inputs remain in that same candidate directory. Shared Notes retain their Run/Manager/context and survive concurrent appends. New adapter instances reopen candidates/results after temporary environments close.

### Apply in the Linux runner

Add `--runner RUNNER_IMAGE_ID` to the same command to run Apply in the [Linux runner image](../design/linux-runner.md) through the run configuration's engine, for example Docker Desktop on Windows. The runner clones committed history from a read-only mount of the checkout's `.git` directory (the working tree is never mounted), keeps Apply state in the engine volume `markitect-state-<repositoryIdentity>` and returns a verified change as a new branch, `markitect/apply-<apply id>` or the name given with `--result-branch`; the checkout's working tree, index and current branch stay unchanged. `productBuild` names the runner image's build. The runner applies committed history only: it refuses uncommitted changes and untracked files that Git does not ignore rather than dropping them, and Git-ignored files, accounted ones included, do not reach it. Any named branch, `main` included, may be checked out; an existing result branch is refused. The runner takes repository-relative `--model`, `--dependency` and instruction boundaries only, and refuses a detached HEAD, linked worktrees and shared clones. Each launch keeps its translated inputs, hand-back and log under `<stateDirectory>/<repositoryIdentity>/runner/`. The runner receives the engine's socket, which is host-level control of the engine, and the engine must be local: a set `DOCKER_HOST` or a remote Docker context is refused. The [native qualification](../validation/v1-l1-runner-qualification-20261027.md) passed from Windows through Docker Desktop, with one Library Apply and an early stop. The WSL launch form and a native Linux engine are not yet observed.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, declare the `ownership` areas in `.markitect/execution.yaml` and register the realization files in each owner's `.markitect/registry/<package>/registry.yaml`, as exact files or as directories ending in `/`. A file may realize several Definitions but has exactly one effective registration and one owner, the most specific area. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate or wrong-owner registrations, files without an owner and overlaps with protected inputs fail before any Agent work; Apply repairs none of them. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime.

## Record preparation and interrupted work

Preparation and generated-document operations create no Run or candidate. Preserve actual diagnostics and completed document paths. A Run that exhausts its limits or has uncertain observation/closure remains incomplete; its mandatory result records what happened. Stored candidates and Notes survive closure, but no automatic resume of old Manager environments exists. Apply uses a state-directory invocation lock and guarded per-path publication, preserves intervening user changes, reports partial transfers and never commits or pushes the owner checkout.

## Acceptance and status

Exit 0 means the requested stage passed; 1 is known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. A committed model binds a specification revision, not an authenticated human decision. Local source checks, physical container proofs, native provider behavior, hosted CI, integration, releases and owner acceptance remain distinct evidence.

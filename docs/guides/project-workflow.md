# Project workflow

This is the one guide for using Markitect on a project. The commands are `check`, `context`, `docs`, `apply`, `version`, `licenses` and `help`; build them from `cmd/markitect`. `check`, `context` and guarded `docs` consume structural Project preparation and start no Agent. `apply` runs Assessment, Implement and Verify, each over the complete model-derived Manager hierarchy, and publishes only the independently reopened stored Verify approval after all Run resources close. Assessment, Implement and Verify are mandates of `apply`, not separate commands. Extract, Cockpit, interactive authoring and Setup are not part of V1, and the CLI has no MCP server. The [architecture](../architecture.md) explains the responsibilities, the [delivery plan](../implementation-plan.md) owns implementation status and evidence authority, and [CONTRIBUTING](../../CONTRIBUTING.md) owns Markitect's own source gates.

## Where the commands run

The model commands run wherever Go and Git run. The Apply host runs on Linux ([DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution)).

| Host | `check`, `context`, `docs` | `apply` |
|---|---|---|
| Linux | Native | Native, or in the [Linux runner](#apply-in-the-linux-runner) with `--runner` |
| Windows | Native | Only in the Linux runner, started through Docker Desktop with `--runner RUNNER_IMAGE_ID`. Without `--runner`, `apply` is refused before it reads any input |

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

### Ownership and boundaries

The canonical model owns intended behavior. Registry shards assign exact artifact paths to one responsibility and the Definitions they realize. When the owner deletes a Definition, its links become stale: `check` still passes and lists them under `stages.structural.files.accounting.stale`, Apply's Implement has the shard owner remove or realign them, and no candidate or publication keeps one. Removing or merging a whole Package works the same way: `check` lists its shard under `stages.structural.registry.orphaned`, the Manager that now owns each artifact's declared area (Root where none does) registers it in its own shard, and Root deletes the old shard. Execution v2 declarations own Package instructions and scoped checks. Outer runtime configuration owns native/check profiles; invocation configuration owns finite limits and normal account-login references. Static metadata owns literal image/protocol observations. Captured canonical and control inputs are protected separately; an exact ignore cannot override input protection or satisfy missing realization.

### Existing repositories

Author the intended canonical v1 model and explicit dependencies, declare the `ownership` areas in `.markitect/execution.yaml` and register the realization files in each owner's `.markitect/registry/<package>/registry.yaml`, as exact files or as directories ending in `/`. A file may realize several Definitions but has exactly one effective registration and one owner, the most specific area. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate or wrong-owner registrations, files without an owner and overlaps with protected inputs fail before any Agent work; Apply repairs none of them. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime.

## Model commands

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context PACKAGE --repo .
go run ./cmd/markitect docs --repo .
```

### Inputs and revisions

The default manifest is `.markitect/model.yaml`. `--model` selects another exact manifest; repeat `--dependency PATH` for every dependency. There is no folder or registry discovery. Repository-relative coordinates remain relative to the selected repository; absolute manifests establish explicit acquisition boundaries. The manifest and lock satisfy the [canonical v1 contract](../design/language-contract.md).

An omitted revision observes the actual draft, including Gitignored files. `--revision REF` resolves the ref once through the repository port and observes exactly that commit, keeping its full identity. Neither preview supplies a publication guard. Fixed acquisition never falls back to current external Runtime profile or dependency bytes: public fixed previews refuse unbound present-day external inputs. Closed Runs retain already captured external model/instruction bytes in their candidate storage and use those exact values for fixed candidate observation; they do not reread mutable external paths or vendor them into product files.

### Check

`check --repo PATH` runs the actual structural preparation over the explicit canonical v1 manifest and dependencies. By default it covers the Model check, registry, complete file accounting, authored execution declarations and static contexts. The report keeps each acquisition attempt and the actual Model check, execution, registry, file-accounting and context-selection statuses; a failed or incomplete acquisition cannot become an empty passing model.

`markitect.execution/v2` declares ordered Manager instruction lists, ownership, scope and project checks. Runtime profiles, provider settings and invocation limits have separate owners. Metadata is not requested unless `--metadata` is given: `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH` additionally prepares closed static native/check metadata from acquired bytes, and no environment or provider discovery substitutes for that explicit document. The Agent and Check adapters parse only the selected executable, schema and profile bytes; product defaults come from their embedded versioned source, and effective settings keep their provenance. The [Codex](../../internal/adapters/codexappserver/README.md) and [check](../../internal/adapters/checks/README.md) adapters own their exact profile versions and supported metadata. Image preparation starts no provider or project command. Host `process` check profiles are retired. Reports neither start a process nor establish execution readiness, and static success establishes no native runtime capability. [Provider adapters](provider-adapters.md) describes the metadata and runtime boundary.

### Context

`context PACKAGE --repo PATH` selects one local Package, including intermediate or empty Packages from the Model check inventory; `.` selects the ordinary Root. It reports the actual Definition content, resolved values, provenance and inclusion reasons. Imported Packages are not local Managers. It creates no task prompt, mandate, Run, budget or evidence record.

### Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

`docs` renders the actual checked v1 Definitions to `docs/markitect/project.md`. Read `view` in the preview: `canonicalModelDigest` identifies model meaning; `contentDigest` hashes the rendered Markdown; `previewDigest` binds the observed sources, the target bytes, existence and mode, and the generated content. After reviewing the preview, write with only its returned `view.previewDigest`; changed sources or targets require a fresh preview, and the expected digest is never reconstructed. Fixed-revision views never write. The independent document guard rechecks the source/target plan, refuses protected model, control and instruction inputs as targets, reads back the actual bytes and records the completed paths, which remain after a partial failure. The guarded document write is not Apply and not a filesystem transaction.

Previews may include explicitly selected external sources. `docs --write` requires every acquired model, control and instruction source to be repository-bound, because external target disjointness has not been established; otherwise it returns `not_completed` before any effect. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Apply

`apply` starts real Agents in containers, runs the required checks and publishes only a verified candidate; it never creates an owner commit. It is implemented and pending owner acceptance: run it only with explicit authority. It needs a committed repository, an explicit absolute Runtime profile (`--runtime-profile`) and an explicit absolute run configuration (`--run-config`); without them it reports `not_completed` before acquisition. It also needs a container engine (a local one for the runner), the pinned runtime image, a dedicated normal-account login and, on Windows, the Linux runner image.

A Run is closed: no question or answer crosses Root to the user or the invoking AI, there is no `answer` command and the run configuration accepts no owner decisions. A model that lacks a material decision or contradicts itself ends the command without publication ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)); amend the model and start a new Apply. The [architecture](../architecture.md#workflow-and-apply) gives the responsibilities and the [Run README](../../internal/run/README.md) the collaboration contract and limits.

### Explicit Runtime and invocation files

Copy [runtime-profile.example.yaml](../../examples/runtime-profile.example.yaml) and [run.example.json](../../examples/run.example.json) outside the repository and replace every uppercase placeholder. Source configuration does not qualify a provider.

- `ENGINE_EXECUTABLE` is the explicit native absolute engine path, on Windows that of Docker Desktop's `docker.exe`. Every profile uses this invocation engine. Escape backslashes in JSON and quote YAML paths correctly.
- `IMAGE_ID`, `AGENTD_BUILD` and `CODEX_SCHEMA_DIGEST` identify the pinned image and its actual observed build metadata. `ACCOUNT_MODEL` selects the normal account model; check profile names match declared Project checks.
- `ACCOUNT_LOGIN_PATH` is the dedicated normal-account login JSON outside the realization tree. The `example-account` reference matches both files. The provider owns login provisioning and expiry behavior.
- `STATE_DIRECTORY` is an absolute path outside the repository. In the runner it receives the launch directories, while Apply state lives in an engine volume ([state and evidence](#state-and-evidence)).
- `PRODUCT_BUILD` is the operator's build assertion: the build of the binary that runs Apply, which in the runner is the runner image's build. Retain the frozen source SHA separately.
- `DEADLINE_UTC` is an absolute RFC3339 UTC deadline ending in `Z`. Manager count, messages, provider idle timeout, cleanup and persistence all have finite positive limits. `providerIdleTimeoutSeconds` bounds provider work without a served tool call; time Markitect spends serving the Manager's tool calls, such as waiting for its children, does not count, and each served call restarts it.

The [example tests](../../internal/bootstrap/configuration_examples_test.go) substitute both exact files, route the explicit Apply invocation through the CLI, and prepare the captured Runtime with controlled image metadata. They check native format 4 settings, the shared account reference, and declared Check requests, then stop at a deliberate boundary before runtime startup or credential reads. A separate negative case keeps invalid preflight free of metadata and state acquisition. These are static composition and CLI transport tests; actual account login and Apply execution require separate native evidence.

### Invocation configuration

The closed UTF-8 JSON uses `formatVersion: 4`. Required fields are `repositoryIdentity`, `productBuild`, `stateDirectory`, `engineExecutable`, `deadline`, `maxManagers`, `maxMessages`, `providerIdleTimeoutSeconds`, `cleanupTimeoutSeconds`, `persistenceTimeoutSeconds` and `credentials`. Optional `workDeadline` reserves part of the whole deadline and must precede it. Unknown, duplicate, null, relative-path and invalid values are refused. Old strategy/role, ownerRequest, helper, harvest and ownerDecisions fields are not accepted. There are no ambient defaults. See the exact [example](../../examples/run.example.json).

Format 3 is refused with a message that names the change. Its `messageTimeoutSeconds` bounded all provider work of one response; format 4's `providerIdleTimeoutSeconds` bounds only provider work without a served tool call, so no limit silently changes its meaning. Convert a format 3 file by setting `formatVersion: 4` and choosing `providerIdleTimeoutSeconds` for the new meaning.

The example allows 600 seconds of provider work without a served tool call. Time the host spends serving the Manager's tool calls, such as awaiting children, does not count, and every served call restarts the limit, so a Manager that keeps delegating, awaiting and reviewing is not stopped by it; the overall deadline still bounds the whole Apply. In Phase C, Root's single Implement response hit the earlier whole-response limit six times while it kept delegating. One observed successful native Library message exchange alone exceeded the previous 120-second example limit; 600 seconds was the finite bound used in that native evidence. Configure this limit for the project within the original overall deadline and message allowance; the example does not guarantee that every stretch of provider work will fit.

The message allowance counts messages to Managers. A question costs nothing of its own: it is charged only as part of the message that delivers it, together with the receiver's other waiting events. Each Manager has at most 8 open questions. Before a correction pair after a Verify Reject, Apply checks that the remaining allowance can start Implement and Verify with every Manager once each; otherwise it ends incomplete with the cause "budget does not cover a correction pair".

### Dedicated credential references

Each exact credential reference selects a provider, mode and absolute native file outside the repository. Codex (`codex-app-server`) accepts `dedicated-runtime-login`, a normal account login of the provider CLI. Claude Code (`claude-code`), an addition beyond V1's Codex scope, accepts only `subscription-credentials`, a Claude Code `.credentials.json` account login ([Claude adapter](../../internal/adapters/claudecode/README.md#login)). There is no API-key route, and Apply with a Claude profile refuses an environment that sets an API-key variable such as `ANTHROPIC_API_KEY`. Parsing and static metadata preparation do not read credentials. Only the native resolver reads the selected regular file with path/link checks, safe errors and a 1 MiB bound. The provider owns normal-login validation and the session copy; source files are not rotated by the resolver. In the runner, the launcher first checks each login the same way and then mounts each file on its own, read-only; the runner's translated configuration names the mounted path, never the content.

### Execute, verify, and apply

The target's tracked baseline must be committed. Natively, the target must also be on an isolated non-protected branch, for example `git switch -c dev/realization`; Apply checks this existing write policy before opening any Manager resources and again during publication. The runner accepts any named branch, because it publishes into its own clone ([checkout form](#apply-in-the-linux-runner)). On Linux, run Apply natively:

```sh
"ABSOLUTE_MARKITECT_BINARY" apply --repo "REPOSITORY" --runtime-profile "ABSOLUTE_RUNTIME_YAML" --run-config "ABSOLUTE_RUN_JSON"
```

On Windows, run the same Apply in the [Linux runner](#apply-in-the-linux-runner), here in PowerShell:

```powershell
& "ABSOLUTE_MARKITECT_BINARY" apply --repo "REPOSITORY" --runtime-profile "ABSOLUTE_RUNTIME_YAML" --run-config "ABSOLUTE_RUN_JSON" --runner "RUNNER_IMAGE_ID"
```

`--model` and repeated `--dependency` select committed inputs. `--revision` and `--metadata` remain preparation options. A successful preflight starts real Run environments, provider Sessions and required checks. Exactly one executable Codex account profile is selected by its presence in the captured Runtime document. Natively, accounted ignored inputs remain part of the captured baseline, and publication writes the verified paths into the checkout's working tree; the runner sees committed files only and returns its change as a new branch.

The [Run](../../internal/run/README.md) gives every Manager its own fresh environment and Git copy, direct delegation and reviewed upward integration. An untracked ELF file in a Manager's copy that no registry entry names by exact path, such as a `go build` executable, is a build output and operational state that never enters a candidate; every Implement check and the Run record list these files, and Verify receives those of the Implement Run. Every other file in the copy, a registered binary included, needs its registration, exact ignore entry or input role. A Manager asks only its direct parent or its direct children; siblings align through their common parent, and nobody outside the Run is asked. Each Implement child submission reaches its parent with its deterministic check result; an invalid one goes back to the child only through the parent's rejection with Findings. A material missing decision or contradiction produces a model blocker; correct the model outside the Run. There is no answer command or preloaded owner decision. Assessment judges model readiness and Approve always leads to Implement, even for no-change. Only Findings that name a Definition of the current checked model reach Implement; Assessment approvals and Verify judgments keep the others as Shared Notes ([DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model)). Implement submits a candidate; fresh Verify alone can approve publication. Every Verify sees the complete diff against the command's bound input and, when a stored closed Verify approval covers exactly that input under the same model, execution configuration and dependencies, the host fact that it was already verified. Under that fact, Verify approves a changed candidate only when Root's approval names the defect the change corrects as a Finding with a current Definition, which the Run record keeps; otherwise any change is a Reject. Verify Reject carries its Findings, each naming a current Definition, into fresh Implement and Verify Runs with the original shared allowance and deadline. Assessment Reject, model blockers and technical or closure failures stop publication.

### Apply in the Linux runner

`--runner RUNNER_IMAGE_ID` runs the same Apply in the [Linux runner image](../design/linux-runner.md), started through the run configuration's engine. On Windows this is the only form of Apply, started through Docker Desktop; on Linux it is optional. Apply inside the runner is the unchanged product: the same preflight, Runs, Verify approval and guarded publication, here into the runner's own clone. The [runner design](../design/linux-runner.md) owns the rationale and the exact mechanics; this section is how to use it.

Build the image from the repository root at the source to run, then read its ID and its build:

```sh
docker build -f packaging/runner/Dockerfile --build-arg MARKITECT_REVISION="$(git rev-parse HEAD)" -t markitect-runner .
docker image inspect --format "{{.Id}}" markitect-runner
docker run --rm --entrypoint cat RUNNER_IMAGE_ID /opt/markitect/runner.env
```

`--runner` takes the image ID (`sha256:` and 64 hex digits), not a tag. The run configuration's `productBuild` names the runner's build, `<version>+<revision>` as `runner.env` records it; the runner checks it against its own binary, and the host binary's build is not checked.

**Inputs.** The files keep their native meaning; the launcher derives the runner's copies and keeps them in the launch directory. The run configuration's engine starts the runner, its state directory receives the launch directory, and its limits, deadlines, project identity and `productBuild` stay unchanged. Every Runtime profile `engineExecutable` must name the run configuration's engine. `--model`, `--dependency` and instruction boundaries must be repository-relative; YAML anchors and aliases are refused, and no external manifest or instruction directory is mounted.

**Mounts.** The runner receives the checkout's `.git` directory read-only, never its working tree, so untracked and ignored files such as `.env` stay invisible; the translated run configuration and Runtime profile, read-only; each login file on its own, read-only; the engine's socket, which is host-level control of the engine; the launch's writable `result/` hand-back area; and the state volume `markitect-state-<repositoryIdentity>`. The runner itself has no network. Agents run in the hardened, non-root sibling containers as they do natively. The [design](../design/linux-runner.md#mounts) lists the exact mount targets.

**Refusals.** The launcher fails closed before the launch:

- **Dirty checkout.** Uncommitted changes to tracked files and untracked files that Git does not ignore are refused with their count and an example: commit or remove them; runner mode applies committed history only. Nothing is dropped silently. Git-ignored files are not refused and do not reach the runner, accounted ones included, so an Apply that depends on them sees a different baseline than a native Apply.
- **Checkout form.** The checkout must be the repository root and main working tree, on a named branch, with its own objects: a detached HEAD, linked worktrees, submodules and shared clones (`objects/info/alternates`) are refused. Any named branch, `main` included, may be checked out, because the runner writes only its clone and a new branch.
- **Remote engine.** The engine must run on this machine, because bind sources and the socket resolve on the engine's host. A set `DOCKER_HOST` is refused, and so is a current Docker context, also one selected by `DOCKER_CONTEXT`, whose endpoint is not a local socket or named pipe (`unix://`, `npipe://`). Docker Desktop's `desktop-linux` context is local. Rootless engines with another socket path are not supported.
- **Existing branch.** A result branch name that already exists is refused, before the launch and again at the import.
- **Paths.** On Windows, drive paths such as `C:\Users\me\repo` are passed in Docker Desktop's `C:/Users/me/repo` form; UNC, device, WSL (`\\wsl$\...`), drive-relative and relative paths are refused, as is any path that would change the meaning of the `--mount` value. A checkout inside a WSL distribution is launched from WSL.

**Result branch.** Only when Apply passed with a read-back publication does the runner commit exactly the published paths, as `Markitect Runner <runner@markitect.invalid>` on top of the observed base, and hand them back as a Git bundle. The launcher verifies the bundle, checks that the commit has the single parent base and changes exactly the published paths, and creates the branch `markitect/apply-<apply id>`, or the name given with `--result-branch` (accepted only with `--runner`). The checkout's working tree, index and current branch stay unchanged. Review and merge the branch like any other: Apply still creates no owner commit, and the runner's commit exists only on the new branch. A no-change publication returns no branch. A failed import names the bundle, which stays in the launch directory.

**Stopping.** The launch is bounded by the run configuration's deadline plus ten minutes for clone, hand-back and stop. Ctrl+C asks the engine to stop the runner container with a four-minute grace period, so Apply closes its containers and keeps its evidence; a stop that arrives after a verified hand-back does not cancel the short import. A host process that dies without cancelling leaves the runner running until Apply's own deadline, with the state lock held: find it with `docker ps --filter label=markitect.runner` and stop it with `docker stop --time 240 markitect-runner-<launch>`. `docker rm --force markitect-runner-<launch>` is the last resort, after which Apply's sibling containers (label `markitect.store`) can remain.

**Qualification.** The [native qualification](../validation/v1-l1-runner-qualification-20261027.md) passed from Windows through Docker Desktop: the mounts, one Library Apply handed back as a branch, and an early stop. The WSL launch form was not observed, because Docker Desktop's WSL integration was off on the qualification host; it needs that integration enabled. A native Linux engine and a stop during publication or import are not yet observed.

## State and evidence

Preparation and generated-document operations create no Run, candidate or evidence. The CLI names the project state directory and the immutable Apply report, whose `evidencePath` is `<state>/<repositoryIdentity>/applies/<apply id>.json`. Beside it, Run records live under `runs/`, the index of completed Verify approvals under `approvals/`, candidates under `candidates/`, Notes under `notes/`, native transcripts under `transcripts/<apply id>/` and host-Git scratch under `temporary/`; one `apply.lock` per state directory serializes Applies. The retained Apply report and printed JSON use the same final status/outcome projection.

Each Run persists the exact input reference, Project binding, technical and semantic outcomes, required findings, handoffs, actual check receipts and closure observations, plus concrete native identity observations when available. Temporary Manager and Git environments close before a Run returns; candidates, terminal records and Notes stay in the state root and can be reopened independently by new adapter instances ([Run](../../internal/run/README.md), [workspace service](../../internal/adapters/gitrepo/README.md#run-git-workspace-service)). Candidates are independently verified self-contained bundles; external captured inputs remain in that same candidate directory. Shared Notes retain their Run/Manager/context and survive concurrent appends.

In the runner, that state lives in the engine volume `markitect-state-<repositoryIdentity>` instead of a host directory, because host binds through Docker Desktop do not preserve executable modes. The volume survives launches, so its lock serializes concurrent runners of one project and evidence accumulates; it has no garbage collection. Read it through any container that mounts it, for example in a POSIX shell: `docker run --rm --entrypoint tar --mount type=volume,source=markitect-state-PROJECT,target=/s,readonly RUNNER_IMAGE_ID -C /s -c . > state.tar`. On the host, each launch gets `<stateDirectory>/<repositoryIdentity>/runner/<launch>/`:

- `invocation/run.json` and `invocation/runtime-profile.yaml`: the translated inputs, which hold login paths inside the runner, not login contents;
- `result/apply-report.json`: the runner's Apply report, also embedded in the printed report;
- `result/result.json`: the runner's hand-back record;
- `result/result.bundle`: the verified change, when there is one;
- `runner.log`: the runner's diagnostics, also streamed to standard error;
- `launch.json`: the host report as printed.

Preserve actual diagnostics and completed paths. A Run that exhausts its limits or has uncertain observation or closure remains incomplete; its mandatory result records what happened. Stored candidates and Notes survive closure, but no automatic resume of old Manager environments exists. Native Apply uses the state-directory invocation lock and guarded per-path publication, preserves intervening user changes, reports partial transfers and never commits or pushes the owner checkout. Multi-file or interrupted effects are not transactional and not safe to replay blindly. Native Apply qualification is separate evidence; the [qualification record](../validation/v1-qualification-20261024.md) holds it.

## Operations

### Authoring and checking

Keep intended behavior in canonical Definitions, exact realization ownership in the registry and operational declarations outside the model. Check the actual structural preparation and the owning Package's context before edits. Implementation-only changes need no artificial intent change. The complete census includes Git-ignored files.

### Troubleshooting

| Observation | Response |
| --- | --- |
| Structural check reports an unaccounted file | Register its exact owner/Definition basis or make an exact explained ignore decision; protected inputs cannot be ignored. |
| Known model/control failure | Read the actual diagnostics and repair the authored input before a new preview. |
| Incomplete acquisition or metadata | Preserve observed facts and diagnose the unavailable bytes, budget or source binding. |
| Preview digest changed | Review a fresh source/target-bound preview; never reconstruct the expected token. |
| Document write reports completed paths and an error | Inspect those actual effects and current sources before deciding recovery. |
| Fixed preview refuses current external sources | Use historically bound inputs or an explicit working preview; do not silently substitute today's bytes. |
| `apply` on Windows reports that native Apply is unavailable | Add `--runner RUNNER_IMAGE_ID` to the same command. |
| The runner refuses uncommitted changes or untracked files | Commit or remove them; the runner applies committed history only. |
| The runner refuses `DOCKER_HOST` or the current context | Unset `DOCKER_HOST` and select the local engine's context, for example `desktop-linux` for Docker Desktop. |
| The result branch already exists | Merge or delete it, or name another with `--result-branch`. |
| An Apply ends incomplete or with a partial transfer | Read its report and retained evidence; inspect the completed paths and the current checkout before a new Apply. |

### Exit codes and evidence

Exit 0 means the requested stage passed; 1 is known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. A committed model binds a specification revision, not an authenticated human decision. Local source checks, physical container proofs, native provider behavior, hosted CI, integration, releases and owner acceptance remain distinct evidence.

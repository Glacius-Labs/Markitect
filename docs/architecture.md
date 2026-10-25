# Architecture

How the V1 source is composed and where each responsibility lives. [V1 concept 1.0](concepts/v1-concept.md) is the target and the [V1 execution architecture](design/v1-execution-architecture.md) owns the concrete package contracts. The [delivery plan](implementation-plan.md) lists the steps and known deviations, and the [qualification record](validation/v1-qualification-20261024.md) holds the evidence. Each package README owns its detail; the [code map](development/code-map.md) lists every package.

## Product flow

`cmd/markitect → internal/bootstrap → CLI adapter → Project / Apply`.

- `check`, `context` and `docs` call [Project](../internal/project/README.md) directly. Project captures the sources through its own port, checks the model, registry and file accounting, reads the execution declarations and selects the static Manager contexts. These commands start no Agent and create no Run state.
- `docs` renders the readable model view in `internal/modeldocs`; `internal/adapters/documents` writes it behind an independent guard that is separate from Apply publication.
- `apply` prepares from one complete committed capture, then runs [Apply](../internal/apply/README.md): preflight, Workflow, and publication of the stored approval.

Bootstrap alone builds the concrete adapters and the state directory. CLI code parses arguments and presents reports. It does not invoke Git itself or start a second lifecycle.

## Layers

All maintained Go packages use one root module. [Modules](development/modules.md#dependency-direction) states the allowed imports, and a mechanical gate checks them in production, test and platform files.

| Responsibility | Packages | Depends on |
| --- | --- | --- |
| Canonical language, Model check | `internal/model` | No other Markitect responsibility |
| Fixed path, file and tree values | `internal/snapshot` | No other Markitect responsibility |
| Manager topology, context selection and settings | `internal/organization` | Model, snapshot |
| Registry, ownership, accounting and candidates | `internal/realization` | Model, snapshot |
| Checked capture and preparation | `internal/project` | The four packages above and its own source port |
| Static native and check profile metadata | `internal/runtimeconfig` | Model, organization, Project, snapshot; no Run or provider session |
| Model-derived instructions for native Sessions, rendered by the native adapter | `internal/harness` | Project values; no I/O, no Run |
| Closed hierarchical Run | `internal/run` | Project, organization, snapshot; Session, workspace, check and record ports |
| Sequence of Runs | `internal/workflow` | Run values and its own Executor interface |
| Preflight, publication after closure | `internal/apply` | Model, Project, Workflow, Run result values |
| Generated model view | `internal/modeldocs` | Project, model, snapshot |
| Concrete adapters | `internal/adapters/*`, `internal/agentd` | Their inward consumers; no general sibling privilege |
| Composition | `internal/bootstrap` | Actual services and adapters |
| Entrypoints | `cmd/markitect`, `cmd/markitect-agentd` | Bootstrap, agentd |
| Maintenance | `internal/tooling/*`, `cmd/markitect-check-*`, `cmd/markitect-release` | Explicit maintenance edges, separate from product execution |

## Foundations and Project

`model`, `snapshot`, `organization` and `realization` are pure values and deterministic operations over supplied facts. They have no repository, filesystem, Git or provider responsibility. Model digests, snapshot tree identities and acquisition guards are different values.

There is one Root and one Manager per local modeled Package, including intermediate and empty Packages. The topology is derived from the model; it is not a canonical Manager Kind.

Project acquires through its `Source` and `PreviewSource` ports. A committed capture requires a clean HEAD; working and fixed previews never return a publication guard. `Prepared` is the immutable result: checked inputs, canonical result, topology and context, registry and layout, instructions and declared checks. A new input closure needs a fresh capture, never an extended old guard. Runtime profiles, provider settings and invocation limits are outside Project; `runtimeconfig` prepares the static metadata for them on request. Static metadata grants no execution capability.

## Run

`run.Executor.Execute` runs one Assessment, Implement or Verify over the complete checked hierarchy. Every Manager gets a fresh native Session and its own Git copy; the native adapter loads the model-derived harness into that Session. Parents delegate to direct children, review the fixed child handoff by commit, integrate accepted work upward and answer questions that go only to a direct parent or child. Run keeps one explicit assignment state per Manager and owns the fixed Manager tool set (`delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes`, `append_note`). A judgment is bound to the received task and the assessed state; the end of a native response is not a judgment.

Every Implement child submission reaches its parent together with its check result, and the parent cannot approve or integrate an invalid one. Assessment and Verify keep the fixed input candidate; review scratch never replaces product bytes. Notes are optional attributed hints stored separately from the mandatory result.

Run performs no process, provider or filesystem I/O itself. It consumes the Sessions, workspace, candidate-check and record ports, closes every Manager environment and the Git service on every path, saves and reads back the terminal result, and exposes one closed-result validation (`ValidateClosed`) that workflow and publication use without naming any Run resource. The [Run README](../internal/run/README.md) describes the collaboration contract, checks, Notes and limits.

## Workflow and Apply

Workflow runs Assessment first. Assessment approval always leads to Implement, also when the result changes nothing. Implement submits a candidate; a fresh Verify alone can approve it. A Verify rejection carries mandatory Findings into a fresh Implement and a fresh Verify under the original shared allowance and deadline. Assessment rejection, a model blocker, or technical or cleanup uncertainty stops the command without publication; Apply never repairs the model.

After every Run has closed, Apply reopens the stored final Verify approval, and only then does the publisher reconstruct the verified candidate with host Git in an isolated repository, repeat the declared checks in fresh Linux copies, compare the complete captured Census, transfer guarded paths and read them back. Apply creates no owner commit and does not push. A transfer that fails part-way reports the completed paths. The [Apply README](../internal/apply/README.md) and the [publication boundary](../internal/adapters/gitrepo/PUBLISHER.md) describe the details.

## Adapters

| Adapter | Responsibility |
| --- | --- |
| `gitrepo` | Complete repository capture, private Run Git copies (`LinuxGit` on private Linux volumes for native Run copies, `LocalGit` for controlled tests and as the host-side candidate store and publication Git), self-contained stored candidates and guarded publication ([workspaces](../internal/adapters/gitrepo/WORKSPACES.md)) |
| `runstore` | Immutable terminal Run results and concurrent project Notes |
| `codexappserver` | Codex App Server Sessions with the model-derived harness and the nine Run tools ([README](../internal/adapters/codexappserver/README.md)) |
| `container`, `internal/agentd` | Container engine client, the frame protocol and the session helper that runs the provider CLI and bridges its tools ([container runtime](../internal/adapters/container/README.md)) |
| `checks` | Declared project checks in pinned containers over disposable input and home copies ([README](../internal/adapters/checks/README.md)) |
| `documents` | Guarded write of the generated view |
| `cli` | Command parsing and JSON reports |

Codex with normal account login is the only provider path in V1 ([D18](concepts/v1-concept.md#23-provider-scope-d18)). The [provider guide](guides/provider-adapters.md) covers the profiles and the native boundary. The [base image](../packaging/runtime/README.md) pins Codex.

## Authority and evidence

The canonical model states intended behavior; exact registry entries state artifact ownership and Definition basis. Execution declarations, runtime profiles, invocation limits and run evidence have separate owners. A committed model binds a specification revision, not an authenticated human decision. Source checks, hosted CI, provider execution, releases and owner acceptance are separate evidence.

Every regular file in the actual census is a protected canonical or control input, an exact registered artifact or an exact explained ignore. Registry shards derive their owner from their Package coordinate; each artifact links existing Definitions and has one owner. Gitignore is not accounting.

## State and evidence storage

Preparation creates no Run, candidate or evidence. `apply` keeps its state outside the repository in the operator's state directory: one `apply.lock` per state directory and, per repository identity, `applies/` (the Apply report), `runs/` (closed Run results), `candidates/` (stored Git bundles), `notes/`, `transcripts/` and `temporary/` (scratch for host Git). Document writes retain the observed completed paths if a later validation fails.

The generated view distinguishes the canonical model digest, the rendered content digest and the source- and target-bound preview digest. Writes recheck sources and target, reject protected inputs and historical writes, and read back the actual bytes. Multi-file or interrupted effects are not assumed transactional.

## Distribution

Source packaging, release publication and guarded installation are independent maintenance services. Source archives use the root `cmd/` and `internal/` layout. Source-smoke checks neither publish a release nor demonstrate provider or Apply acceptance. See [distribution](../packaging/README.md) and [operations](guides/operations.md#release-operations).

Earlier product generations are listed in the [archive index](archive.md).

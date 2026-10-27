# Architecture

How the V1 source is composed, where each responsibility lives and which contracts and invariants hold between the packages. [V1 concept 1.0](concepts/v1-concept.md) is the target ([DEC-068](concepts/register.md#dec-068-v1-concept-10)) and this page owns the concrete package contracts. The [delivery plan](implementation-plan.md) lists the steps and known deviations, and the [qualification record](validation/v1-qualification-20261024.md) holds the evidence. Each package README owns its detail; the [code map](development/code-map.md) lists every package.

## Product flow

`cmd/markitect → internal/bootstrap → CLI adapter → Project / Apply`.

- `check`, `context` and `docs` call [Project](../internal/project/README.md) directly. Project captures the sources through its own port, checks the model, registry and file accounting, reads the execution declarations and selects the static Manager contexts. These commands start no Agent and create no Run state.
- `docs` renders the readable model view in `internal/modeldocs`; `internal/adapters/documents` writes it behind an independent guard that is separate from Apply publication.
- `apply` prepares from one complete committed capture, then runs [Apply](../internal/apply/README.md): preflight, Workflow, and publication of the stored approval. This Apply host runs on Linux. On Windows, bootstrap refuses `apply` without `--runner` before it reads any input; `--runner` starts the same Apply in the [Linux runner](design/linux-runner.md), which hands its verified change back as a new branch.

Bootstrap alone builds the concrete adapters and the state directory and injects them. CLI code parses arguments and presents reports. It does not invoke Git itself or start a second lifecycle. Run organizes the Managers, the workflow chooses the Runs and the command publishes: CLI → Apply → workflow → Run → Project and foundations.

## Layers

All maintained Go packages use one root module. Packages are organized by concrete responsibility and consumer-owned ports; there is no legacy product layer and no second Model check. One rule table in [`internal/tooling/architecture/rules.go`](../internal/tooling/architecture/rules.go) gives every package a layer, lists which layers each layer may import and names the few exceptions, some of them for test files only; the [import gate](#import-gate) enforces it. Foundations never import outward, adapters share only the container client and import no other adapter, only test files import `internal/testkit`, and the consumer bootstrap and source smoke import no Markitect package. No sibling adapter or common tooling parent grants a general import exemption. The generated [code map](development/code-map.md) shows the table, and the source packager accepts exactly its packages below `cmd/` and `internal/`. The table below describes the responsibilities.

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
| Preflight, the sequence of Runs, publication after closure | `internal/apply` | Model, Project, Run values and its own Executor interface; no adapter |
| Generated model view | `internal/modeldocs` | Project, model, snapshot |
| Concrete adapters | `internal/adapters/*`, `internal/agentd` | Their inward consumers; no general sibling privilege |
| Guarded working-tree writes, Git identity and repository-path rules | `internal/guardedwrite` | No other Markitect responsibility |
| Composition | `internal/bootstrap` | Actual services and adapters |
| Entrypoints | `cmd/markitect`, `cmd/markitect-runner`, `cmd/markitect-agentd` | Bootstrap, the runner adapter, agentd |
| Maintenance | `internal/tooling/*`, `cmd/markitect-release` | The guarded writer and build data; separate from product execution |

### Import gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown and removed packages, layer directions and named exceptions, table completeness, owning-document links, external-test self imports and test-only `testkit` use. The document adapter may use only the reviewed guarded-write selectors. Third-party imports remain permitted: this is a Markitect dependency check, not a general proof of I/O purity. `internal/testkit` itself imports no Markitect package. Go fixture directories and independent evaluation assets are not exemptions from product package ownership.

Every Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. The generated code map is byte-checked; it is an inventory of actual packages, not proof that a designed package exists. Regenerate it with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

### Adding, changing or removing a package

Coordinate shared contracts first. A new package root needs an actual consumer and a distinct responsibility. Land each contract with a caller and a concrete implementation, never as an interface-only wave; an absent package needs no stub or success gate, and no consumer can import it. Add, move or remove the package's row in the rule table, give it a package doc comment and regenerate the code map. Source packaging takes its roots from the same table; update its data selection and tests for a new embedded asset. Keep one root module and reject nested product modules. Delete obsolete source, commands and gates together. There is no shared-contracts package, repository framework, DSL, event bus or duplicate model ontology.

## Foundations and Project

`model`, `snapshot`, `organization` and `realization` are pure values and deterministic operations over supplied facts. They have no repository, filesystem, Git or provider responsibility. Model digests, snapshot tree identities, file content identities and acquisition guards are different values.

- **Snapshot** represents exact repository-relative paths, modes, immutable bytes, entries, trees and deterministic content identities. Construction rejects invalid spellings, collisions and unsupported entries explicitly, and comparisons need no Git or filesystem I/O.
- **Model** checks the complete canonical language v1 from supplied immutable sources. It keeps exact identities, values, links, provenance and deterministic model digests, and rejects invalid or incomplete constructs. The fixed Core Definitions hold the normative language contracts. Neither the Model check nor the deterministic values infer physical repository identity or authorize a write.

There is one Root and one Manager per local modeled Package, including intermediate and empty Packages. The topology is derived from the model; it is not a canonical Manager Kind.

Project acquires through its `Source` and `PreviewSource` ports. A committed capture requires a clean HEAD and returns the actual HEAD with an opaque comparison guard; working and fixed previews never return a publication guard, and a fixed preview refuses current external inputs without historical binding. The complete census includes Git-ignored untracked files and excludes only technical Git administration and `.markitect/state/**`. Project prepares everything from one complete capture of explicitly selected model and control inputs: it runs `model.Check`, registry parsing and accounting, `organization.BuildTopology` and `SelectContexts` on fixed bytes and reuses their values and diagnostics directly. It keeps original bytes, source provenance and authored absence, and distinguishes known invalid input from incomplete acquisition. `check` needs no credentials, images or container metadata.

`Prepared` is the immutable result: checked inputs, canonical result, topology and context, registry and layout, instructions and declared checks. It is an in-process value, not another serialized compiler AST; persistence keeps enough fixed source and input identities to recapture and verify it. The model content digest alone does not cover the complete organizational Package inventory, and Git commit, tree and model identity stay distinct. A new input closure needs a fresh capture, never an extended old guard or extra bytes fetched into it. Apply fixes the canonical, control and ignore inputs, while valid registry updates remain realization work; candidate validation reuses the foundation logic against the fixed intent and the candidate's registry and tree, so there is no second model validator.

Runtime profiles, provider settings and invocation limits are outside Project; `runtimeconfig` prepares the static metadata for them on request. Project does not decode runtime profiles, resolve strategies, roles or budgets, start Sessions, execute checks, allocate working copies or publish. `runtimeconfig` is the outer preparation consumer: Project and Run do not import it, it reads no credentials and it never resolves model intent. Its observations retain only the identities, instructions and diagnostics that its consumers use. Static metadata grants no execution capability. Agent profile metadata binds the immutable runtime image identity and the whole metadata file bytes. Preparation may inspect the image and copy build metadata from a created, never started container; it never reads a host provider executable, probes a version, starts a Session or invokes a model. A build-recorded schema digest is an assertion, not an observed schema inventory, and instructions come only from captured inputs.

Physical identity, exact source spellings, redirect refusal, bytes and finite acquisition budgets are adapter obligations; the pure Model check establishes none of them. A passed preparation report does not establish a real session, check execution, verified candidate, operating-system sandbox or Apply readiness.

## Run

`run.Executor.Execute` runs one Assessment, Implement or Verify over the complete checked hierarchy; one executor handles all three mandates, and a Root-only Run is valid only for an actually Root-only model. Every Manager gets a fresh native Session and its own Git copy; the native adapter loads the model-derived harness into that Session. Parents delegate to direct children, review the fixed child handoff by commit, integrate accepted work upward and answer questions that go only to a direct parent or child. Run keeps one explicit assignment state per Manager and owns the fixed Manager tool set (`delegate`, `await`, `review`, `ask`, `reply`, `judge`, `check`, `read_notes`, `append_note`). A judgment is bound to the received task and the assessed state; the end of a native response is not a judgment. The host binds known identities: the acting Manager, Run, mandate and own commits are never requested from the model, Findings carry no sender field, and `review` names the reviewed child commit so that no stale approval can apply.

The native adapters wait for a pending tool's response and cannot answer synchronously while a tool call is pending (no Answer-during-Exchange). Questions therefore arrive at supported tool and message boundaries, without nested inference or immediate preemption; direct delegation remains the Manager's responsibility. The [qualification record](validation/v1-qualification-20261024.md) keeps the native observations of crossed questions and later answers on the earlier contract, and each additional provider needs its own evidence.

Every Implement child submission reaches its parent together with its check result, and the parent cannot approve or integrate an invalid one. Assessment and Verify keep the fixed input candidate; review scratch never replaces product bytes. Notes are optional attributed hints stored separately from the mandatory result: persistent, project-scoped and noncanonical, with provenance, they grant no authority and schedule no work. Mandatory Findings stay in results and explicit next-Run inputs ([hand-over between Runs](../internal/run/README.md#hand-over-between-runs)). Verify sees the fixed model and candidate, the complete diff against the command's bound input across correction loops and the necessary technical observations, never Implement conversations or self-approval. Hidden outside-oracle expectations never enter Implement context.

Run performs no process, provider or filesystem I/O itself. It consumes the Sessions, workspace, candidate-check and record ports below and closes and stores everything as the [lifecycle](#run-lifecycle) describes; workflow and publication use its one closed-result validation (`ValidateClosed`) without naming any Run resource. The [Run README](../internal/run/README.md) describes the collaboration contract, checks, Notes and limits.

### Ports

| Contract | Producer → consumer | Contents |
| --- | --- | --- |
| Project `Prepared` and `CandidateRef` | Capture and Git store → Run, diagnostics, Apply | Checked input binding; durable project-scoped storage key, Git object format, commit and tree; no live object or volume |
| Run `Request` and `Result` | Workflow or standalone caller → `Execute` → workflow, CLI | ID, mandate, prepared binding, input candidate, the command's bound input with the optional host fact that it was already verified, required Findings and shared allowance; technical completion, judgment, output reference, observations and actual cleanup |
| Run closed-result validation | Run → workflow, publication | One check that a stored or returned result is bound, technically completed, fully closed and carries a valid mandate decision with its required Findings; callers know no resource names |
| Workflow `Executor` | Run → Apply composition | One fully closed Run per call; the workflow chooses the next call |
| Run `Sessions` | Native adapter → Run | Open with harness and workspace, ordinary messages and tools, cancellation, closure and observations; no domain Turn or Pass |
| Run workspace and candidates | Git adapter → Run | Open one Run, task bases, fixed handoffs, reviewed integration, retain and reopen candidates, close; handles stay inside execution |
| Run records and Notes | Run store → Run, Manager tools | Save and read the closed record; append Notes with origin and context and read them in bounded pages ([Shared Notes](../internal/run/README.md#shared-notes)) |
| Run checks | Check adapter → Run | Named check on the exact candidate; actual exit, timeout and cleanup observations |
| Apply publication | Git adapter → command | Durable accepted candidate, source and target guards, transfer and readback; no living Run |

Ports are consumer-local interfaces. One concrete adapter can implement several small ports; there is no combined Git, source and publication interface. No session handle appears in a persistent result. Technical completed, failed or incomplete is separate from the semantic approve, reject or model blocker: Implement submits a checked candidate, not publication approval, and lost transport is not Reject.

### Run lifecycle

`Execute` is the single closure owner, also in standalone use:

1. Validate the checked binding, full topology, mandate and finite allowance. Open one Git workspace service for this Run and fresh Manager environments only for its topology, with no reservation for future Runs. Each Manager keeps its copy and conversation within the Run.
2. Install the harness and coordinate the full hierarchy. Questions are delivered while other Managers wait; mutations are serialized per parent copy, not across all children; cancellation releases waiters. Parents review and integrate fixed child results in completion order.
3. End active writes before capturing a handoff or Root commit, through ordinary session synchronization and internal Git commits. Decisions and checks bind to that immutable object; a later change needs renewed affected checks and review. There is no per-write or freeze protocol.
4. Persist the required candidate objects and verify that they are independently readable before deleting temporary copies. Diagnostic partial candidates are kept where possible; failed retention never yields acceptance.
5. Close every Manager process, container and home, the Run's Git service and temporary volumes, also after a partial start. Cleanup is bounded and runs even after cancellation, without granting further agent work; actual close errors are recorded.
6. Save and read back the terminal result with its cleanup outcome, then return. A storage failure returns a technical error with the available diagnostic identities and never claims a saved result; a crash before terminal storage leaves the Run incomplete, never an inferred success or an automatic resume.

Cleanup uncertainty, missing decisions and persistence failures block a successful return and publication. A standalone caller supplies finite limits; Apply supplies one shared, concurrency-safe allowance and the original deadline to every Run, corrections included. Resource counts have a technical meaning, not domain Turn semantics, and do not reset after a Reject. The [Run README](../internal/run/README.md#limits-and-evidence) describes the provider idle limit and the budget causes.

Native Run copies live on private Linux volumes: a bounded probe on 23 October 2026 observed that host-directory binds through Windows and Docker Desktop do not preserve executable modes ([record](validation/v1-qualification-20261024.md#bounded-component-probes)). Git operations and the Manager work on the same copy, only that Manager's copy is mounted into its container, and no second writable host mirror is synchronized; local Git serves controlled tests. Native and check attachments identify the actual owned copy and never interpret a host directory as a volume. Candidate file modes are verified through Git objects in the independent empty repository that reads a bundle back, never through a Windows checkout, which is not a filesystem-mode oracle.

### Checks and review

The same deterministic core serves four checkpoints: command preflight before resources are acquired, every fixed child handoff, fixed Root completion and publication. Review may reject an invalid candidate, but must not repair it just to obtain a green gate. Required project tests stay meaningfully scoped and commit-bound; not every child runs the entire suite. The [Run README](../internal/run/README.md#checks) describes what each mandate checks.

Accounting proves registered ownership, not who changed a file. Review therefore also receives the Manager and task, the diff from task base to result and the exact integrated child commits with their integration bases. A parent checks unexplained out-of-area changes while recognizing legitimate child integration, and never subtracts accepted child paths or hunks from its own diff, since later parent edits could disappear. Parent conflict-resolution changes still need review; unresolved child-owned work returns to the child. This is small Git review context, not a global provenance ledger or access control.

## Workflow and Apply

Workflow runs Assessment first. Assessment approval always leads to Implement, also when the result changes nothing. Implement submits a candidate; a fresh Verify alone can approve it. A Verify rejection carries mandatory Findings into a fresh Implement and a fresh Verify under the original shared allowance and deadline. Assessment rejection, a model blocker, or technical or cleanup uncertainty stops the command without publication; Apply never repairs the model.

After every Run has closed, Apply reopens the stored final Verify approval, and only then does the publisher reconstruct the verified candidate with host Git in an isolated repository, repeat the declared checks in fresh Linux copies, compare the complete captured Census, transfer guarded paths and read them back. Approval binds the exact candidate and project inputs. Publication never reuses the closed Run service, and no Agent container remains. Any relevant drift from the captured clean target ends in a conflict: there is no semantic merge and no reuse of an approval for changed content. Guards are checked immediately before each mutation and read back afterwards; cooperating command locks and guarded per-path transfer limit races but cannot make an unrelated editor transactionally exclusive, so intervening user changes are preserved. Apply creates no owner commit and does not push. A transfer that fails part-way reports the completed paths. The [Apply README](../internal/apply/README.md) and the [publication boundary](../internal/adapters/gitrepo/README.md#publication-boundary) describe the details.

## Adapters

| Adapter | Responsibility |
| --- | --- |
| `gitrepo` | Complete repository capture, private Run Git copies (`LinuxGit` on private Linux volumes for native Run copies, `LocalGit` for controlled tests and as the host-side candidate store and publication Git), self-contained stored candidates and guarded publication, in separate collaborators ([README](../internal/adapters/gitrepo/README.md)) |
| `runstore` | Immutable terminal Run results, an index of completed Verify approvals and concurrent project Notes |
| `codexappserver` | Codex App Server Sessions with the model-derived harness and the nine Run tools ([README](../internal/adapters/codexappserver/README.md)) |
| `claudecode` | Claude Code Sessions with subscription login, the model-derived harness as system prompt and the nine Run tools; an addition beyond V1's Codex scope, native qualification open ([README](../internal/adapters/claudecode/README.md)) |
| `container`, `internal/agentd` | Container engine client, the frame protocol and the session helper that runs the provider CLI and bridges its tools ([container runtime](../internal/adapters/container/README.md)) |
| `checks` | Declared project checks in pinned containers over disposable input and home copies ([README](../internal/adapters/checks/README.md)) |
| `documents` | Guarded write of the generated view |
| `runner` | `markitect apply --runner`: starts the Linux runner image through the local container engine with the checkout's `.git` and the provider logins mounted read-only, and imports the verified result as a new branch ([design](design/linux-runner.md)) |
| `cli` | Command parsing and JSON reports |

Concrete adapters implement physical acquisition and protocols; the values they serve stay independent of execution and infrastructure.

- **Repository capture** records the complete repository census and the selected inputs with exact paths, modes and bytes. It resolves a symbolic revision once, keeps the full identity, rejects redirects and changed directory identity, and never replaces historical sources with current external bytes.
- **Checks** keep literal arguments, the fixed environment policy, finite resource and output limits, and exact image and executable identities. Each declared check runs against the fixed candidate on an independently verified disposable copy with an empty private home and keeps the actual image, tree, start, exit, output, timeout and closure observations. Unknown or uncertain execution or cleanup never counts as a passing check.
- **The CLI** reports actual statuses, sources and inclusion reasons; retained reports and printed JSON use the same final projection. Usage errors, known failures and incomplete work have distinct exit statuses.

Codex with normal account login is V1's provider path ([D18](concepts/v1-concept.md#23-provider-scope-d18)). Claude Code with subscription login is an addition beyond that scope (slice K1): its adapter is in source, and its native qualification K1b is open. The [provider guide](guides/provider-adapters.md) covers the profiles and the native boundary. The [base image](../packaging/runtime/README.md) pins Codex and Claude Code.

The harness and provider chain runs from the versioned Markitect rules through the canonical project, the Manager's ownership and context, the mandate and Findings to the native instructions, tools and start and the provider's actual loading; text or digest tests alone do not prove it. The V1 harness uses the provider's native base and developer instructions with the complete statically selected mandatory context ([D14](concepts/v1-concept.md#155-v1-form-of-the-harness-d14)); modular files, skills or loadable context are added only for a demonstrated size or usability need. A generic AGENTS.md representation for unknown providers stays planned as best effort, not as an adapter. Harness text holds role, rules, context, task and answer format, not host process states. Generated instructions and tools live outside product candidate content, and overflowing mandatory context is reported, never silently truncated. Each provider adapter inventories the repository's instruction files and the native load precedence, uses supported native overrides or controlled instruction mounts, so the generated instructions are authoritative without editing tracked files, and tests the actual loaded paths, tool availability and priority; a pinned provider that cannot prevent contradictory instruction loading fails that capability gate. Native subagents do not replace modeled Managers.

A Manager receives access-only material from a normal account login through one bounded private setup message before its provider starts, never a shared mutable host home or refresh authority, and no separate auth-helper container is involved. The setup uses allowlisted paths, rejects symlink traversal, restricts permissions and redacts secrets; credentials stay outside arguments, the container environment, logs, harness, Notes and bundles, and a partial setup keeps its cleanup responsibility. The one exception is Claude Code's long-lived subscription token, which the CLI reads only from its process environment: it travels in the provider environment of each start, never the container's, and Claude Code strips it from the processes it starts. There is no API-key route. Login expiry, refresh and every further provider need their own evidence ([Codex adapter](../internal/adapters/codexappserver/README.md#login-and-evidence-limits), [container runtime](../internal/adapters/container/README.md#integrating-a-session-adapter)). Provider-dependent contract decisions stay provisional until bounded native evidence exists: the [early native hierarchy evidence](validation/v1-qualification-20261024.md#early-native-hierarchy-evidence) met this gate for Codex on the earlier contract, and the [A1 contract proof](validation/a1-native-contract-proof-20261024.md) covered the changed collaboration contract. The [Claude adapter](../internal/adapters/claudecode/README.md) implements the stabilized session and tool contract and needs its own native gate (K1b). Each provider's qualification is independent, and native turn identifiers remain adapter facts.

## Authority and evidence

The canonical model states intended behavior; exact registry entries state artifact ownership and Definition basis. Execution declarations, runtime profiles, invocation limits and run evidence have separate owners. A committed model binds a specification revision, not an authenticated human decision. Source checks, hosted CI, provider execution, releases and owner acceptance are separate evidence.

Every regular file in the actual census is a protected canonical or control input, an exact registered artifact or an exact explained ignore. Registry shards derive their owner from their Package coordinate; each artifact links Definitions and has one owner. A link to a Definition the model no longer has is stale: the input of a command keeps it as work for Implement, while a candidate check refuses it within the responsible Manager's subtree and publication refuses any. Gitignore is not accounting.

## State and evidence storage

Preparation creates no Run, candidate or evidence. `apply` keeps its state outside the repository in the operator's state directory: one `apply.lock` per state directory and, per repository identity, `applies/` (the Apply report), `runs/` (closed Run results), `candidates/` (stored Git bundles), `notes/`, `transcripts/` and `temporary/` (scratch for host Git). In the Linux runner, this state directory is the engine volume `markitect-state-<repositoryIdentity>`. Document writes retain the observed completed paths if a later validation fails.

A candidate is a self-contained Git bundle with a named candidate ref and bound metadata. It holds the candidate objects and their required ancestry without depending on an old volume or an incremental bundle, and is verified in an empty Git repository with exact commit and tree readback. A bundle is storage, not approval ([Git documentation](https://git-scm.com/docs/git-bundle)). Git objects and Run records share the project state root as separate records, and candidate data is secured before its terminal reference is written. Terminal records use a temporary file, atomic publication, flush and readback; no multi-file transaction is claimed. Notes are unique append records with atomic visibility and bounded reads, so concurrent processes cannot overwrite them. V1 needs no Git server, database, event sourcing, recovery framework or automatic garbage collection of retained evidence.

The generated view is rendered deterministically from compiled Definitions and distinguishes the canonical model digest, the rendered content digest and the source- and target-bound preview digest. Writes touch only the reviewed generated target, recheck sources and target, reject protected inputs (model, control and instruction inputs) and historical writes, and read back the actual bytes. Multi-file or interrupted effects are not assumed transactional.

## Distribution

Source packaging, bundling and guarded installation are independent maintenance services; Main has no release publisher. Source archives use the root `cmd/` and `internal/` layout. The maintenance-only `release.Input` keeps fixed source files, modes and commit provenance for packaging and installation; its fingerprint is not a snapshot tree digest, a canonical model digest or a publication guard. Published archives keep their original bootstrap and identity. Source-smoke checks neither publish a release nor demonstrate provider or Apply acceptance. See [distribution](../packaging/README.md) and [operations](guides/operations.md#release-operations).

Earlier product generations are listed in the [archive index](archive.md). Earlier capability, module and product-layer contracts remain in the [historical module guide](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/history/development-modules-legacy-sections-20261010.md) and the [preceding source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

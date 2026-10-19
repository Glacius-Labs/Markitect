# Go architecture for reliable Apply

> **Historical, 19 October 2026:** [manager-apply-runtime.md](manager-apply-runtime.md) owns the implemented contract of Apply ([DEC-056](../../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace)). The text below describes the earlier design and is kept as history; do not implement it or read it as current behavior.

This document owns the target packages, their responsibilities and the dependency direction. The [port contracts](manager-apply-port-contracts.md) own the interfaces between them, and the [rebuild plan](manager-apply-rebuild-plan.md) owns what happens to the current source. The architecture gate described in [module rules](../../development/modules.md#apply-target-dependencies) enforces the dependency table below.

## Organize by responsibility

Ports and Adapters with ordinary, idiomatic Go packages. Deterministic model and realization rules stay independent of external systems. The workflow is coordinated through small capabilities that concrete adapters implement. Packages say what they do; not every function needs an interface.

| Package | Responsibility |
| --- | --- |
| `model` | Parse explicitly supplied bytes, compile complete v1, resolve contracts and links, compute canonical content and diagnostics. No repository reads, provider calls or prose interpretation. |
| `snapshot` | Small fixed file-tree values and comparisons. No Git commands or storage policy. |
| `organization` | Derive the root and local Package Managers, resolve defaults and exact overrides, render inspectable context and mandates from supplied facts. No Agent startup or run imports. |
| `realization` | Registry, artifact ownership, path claims, candidate differences, composition, Definition assessments and file accounting over supplied values. |
| `run` | Assignments and fixed-basis runs, tasks, questions and answers, delegation, role sessions, review, integration, budgets, lifecycle and history. It guarantees unchanged operation propagation and messages routed along tree edges ([M1](manager-apply-core.md#messages)), validates transitions and Agent requests, and enforces mandatory review/integration. Managers judge local realization and integration; they do not choose participation or child operation prompts. |
| `apply` | The artifact reconciliation use case: bind inputs, prepare the organization, invoke `run`, require aggregate verification, then request guarded publication and read-back. |

`run` is the common execution mechanism, first used only by Apply; later operations reuse it instead of bringing their own Manager engine. A Manager is a logical identity with tasks, configuration and session references, not a generated Go package or a permanently running goroutine. The first strategy runs one step at a time ([S5](manager-apply-core.md#steps)). A parent may end a step with `wait` while children work, but remains eligible for its own ready work, including prerequisites for those children. Only work with an unmet prerequisite waits; the [execution contract](manager-apply-execution.md) owns admission.

Manager responsibility, provider conversation, execution container and step workspace version have different lifetimes. `run` owns readiness, admission and state transitions against the immutable run inputs. `realization` validates the [D2](manager-apply-core.md#deliveries-and-composition) composition of deliveries and own changes over supplied values; it never operates a live workspace. Workspace/runtime adapters report actual materialization and termination evidence; provider adapters report conversation continuation or its limits. `run` decides whether those observations satisfy the transition, and `apply` owns final verification/publication orchestration. `organization` renders the unchanged operation, current step facts and separate parent hints. The [execution contract](manager-apply-execution.md) owns role/effect/budget separation, and the [continuity contract](manager-apply-concept.md#working-bases-and-conversation-continuity) owns the lifetime requirements. No adapter, report view or provider conversation becomes another scheduler or workflow ledger.

## Source layout

```text
cmd/markitect/
cmd/markitect-agentd/
internal/
  model/
  snapshot/
  organization/
  realization/
  run/
  apply/
  agentd/
  adapters/
    cli/
    mcp/
    gitrepo/
    journal/
    container/
    codexappserver/
    claudecode/
    checks/
  bootstrap/
```

`bootstrap` constructs the concrete adapters and passes them to their consumers through constructors; `cmd/markitect` only calls it. CLI and MCP call the same application operations and own no review, retry or publication policy. `codexappserver` and `claudecode` are the session adapters. `container` is the concrete client of the container engine for sessions, checks and workspaces, not another execution protocol. `agentd` is the session helper inside every container; `cmd/markitect-agentd` builds it for Linux, and the product's base runtime image contains it ([rebuild plan](manager-apply-rebuild-plan.md#container-runtime)). Repository maintenance and release tooling stay outside these product packages.

The first command surface keeps the verb-table principles of [DEC-017](../../concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table): top-level verbs from one table that drives CLI help, dispatch and the MCP tools, with the same arguments.

| Verb | Purpose |
| --- | --- |
| `check` | Compile the model and check coverage and the registry. |
| `context <package>` | Preview a Manager's context package with inclusion reasons. |
| `apply [--continue <assignment>]` | Start an Apply, or continue an assignment after a run that did not complete it. |
| `resume <run>` | Reopen an interrupted run: reconcile its effects, then go on with the same basis. |
| `status [run]` | Show the assignment and run overview, or one run. |
| `inspect <run> [task]` | Show the task tree, timeline and per-step input, report, difference and evidence. |
| `answer <question>` | Bind an owner answer to a question version. |
| `stop <run>` | Request a stop and report the observed result. |

Inside `run`, group coherent task, review, budget, lifecycle and history responsibilities; split only for a distinct responsibility rather than prescribing a file for each concept. Avoid shared Project DTOs, global interface catalogues and untyped property bags. Use the root `cmd` and `internal` layout ([Go module layout](https://go.dev/doc/modules/layout)); the [rebuild map](manager-apply-rebuild-plan.md#source-replacement-map) retains the historical `src` migration baseline.

## Dependency direction and ports

| Package | May import |
| --- | --- |
| `internal/model/`, `internal/snapshot/` | No other target package |
| `internal/organization/`, `internal/realization/` | Model, Snapshot |
| `internal/run/` | Model, Snapshot, Organization, Realization |
| `internal/apply/` | Model, Snapshot, Organization, Realization, Run |
| `internal/adapters/{cli,mcp,gitrepo,journal,codexappserver,claudecode,checks}/` | Model, Snapshot, Organization, Realization, Run, Apply; `gitrepo`, `codexappserver`, `claudecode` and `checks` also `container` |
| `internal/adapters/container/` | The frame codec of `agentd` only |
| `internal/agentd/` | No other target package |
| `cmd/markitect-agentd/` | `agentd` |
| `internal/bootstrap/` | Every target package except commands |
| `cmd/markitect/` | Bootstrap |

Organization and realization never import each other, and the compiler depends on neither. Adapters do not import each other except for the stated `container` access. The [current module rules](../../development/modules.md) enforce source dependencies; the historical replacement map is not a second import policy.

| Port | Consumer | Concrete work |
| --- | --- | --- |
| `apply.Source` | `apply` | Capture model bytes, explicit dependencies, configuration, registry, repository census and the source binding. |
| `apply.Control` | `apply` | Acquire the repository controller and store, supply the journal, check the lock and release it. |
| `apply.Publisher` | `apply` | Recheck the captured source, publish the verified artifacts and registry together, report actual writes and read-back. |
| `run.Sessions` | `run` | Start a configured role in its container, supply tasks and context, observe messages and tool requests, resume, cancel and establish termination. |
| `run.Workspaces` | `run` | Materialize an owned candidate on a runtime volume, observe actual changes, close or retain it. |
| `run.Checks` | `run`, `apply` | Execute explicit project checks on the exact candidate, for reviews, for the aggregate and for the early check before a submit. |
| `run.Journal` | `run`, `apply` | Durably append events and exact payloads under one writer. |

Interfaces live with their consumers, and adapters return concrete types ([Go code review comments](https://go.dev/wiki/CodeReviewComments#interfaces)); one adapter may satisfy several narrow ports. `adapters/journal` implements both `run.Journal` and `apply.Control`, and `adapters/gitrepo` implements `apply.Source`, `apply.Publisher` and `run.Workspaces`, the last on runtime volumes through `container`. `codexappserver` and `claudecode` each implement `run.Sessions`. Effectful calls take `context.Context`; pure computations need no interface to be testable. Avoid generic `common`, `types` or `interfaces` packages ([package names](https://go.dev/blog/package-names)).

The Orchestrator owns the task and message protocol, effective authority and history. An Agent adapter translates provider transport into those records; it cannot accept a delivery, grant a path or pass a review. Manager and reviewer use the same session port with different roles. Provider wire objects and session IDs never become model types. The registry belongs to `realization`, outside the Definition roots, and is published together with its artifact tree.

The [library walkthrough](manager-apply-library-flow.md) follows a root task and its lending child through these boundaries, from model input to publication.

# Go architecture for reliable Apply

This document owns the target packages, their responsibilities and the dependency direction. The [port contracts](manager-apply-port-contracts.md) own the interfaces between them, and the [rebuild plan](manager-apply-rebuild-plan.md) owns what happens to the current source. The architecture gate described in [module rules](../../development/modules.md#apply-target-dependencies) enforces the dependency table below.

## Organize by responsibility

Ports and Adapters with ordinary, idiomatic Go packages. Deterministic model and realization rules stay independent of external systems. The workflow is coordinated through small capabilities that concrete adapters implement. Packages say what they do; not every function needs an interface.

| Package | Responsibility |
| --- | --- |
| `model` | Parse explicitly supplied bytes, compile complete v1, resolve contracts and links, compute canonical content and diagnostics. No repository reads, provider calls or prose interpretation. |
| `snapshot` | Small fixed file-tree values and comparisons. No Git commands or storage policy. |
| `organization` | Derive the root and local Package Managers, resolve defaults and exact overrides, render inspectable context and mandates from supplied facts. No Agent startup or run imports. |
| `realization` | Registry, artifact ownership, path claims, candidate differences, composition, Definition assessments and file accounting over supplied values. |
| `run` | Assignments and fixed-basis runs, tasks, questions and answers, delegation, role sessions, review, integration, budgets, lifecycle and history. It validates transitions and Agent requests; Managers choose their outcomes and task prompts. |
| `apply` | The artifact reconciliation use case: bind inputs, prepare the organization, invoke `run`, require aggregate verification, then request guarded publication and read-back. |

`run` is the common execution mechanism, first used only by Apply; later operations reuse it instead of bringing their own Manager engine. A Manager is a logical identity with tasks, configuration and session references, not a generated Go package or a permanently running goroutine. One writer is active at a time. A parent yields with recorded child tasks pending; the Host runs them and resumes the parent with their results.

## Source layout

```text
cmd/markitect/
internal/
  model/
  snapshot/
  organization/
  realization/
  run/
  apply/
  adapters/
    cli/
    mcp/
    gitrepo/
    journal/
    codexappserver/
    checks/
    process/
  bootstrap/
```

`bootstrap` constructs the concrete adapters and passes them to their consumers through constructors; `cmd/markitect` only calls it. CLI and MCP call the same application operations and own no review, retry or publication policy. `codexappserver` is the first session adapter. `process` is the small shared process-lifetime helper for that adapter and for project checks, not another execution protocol. Repository maintenance and release tooling stay outside these product packages.

The first command surface keeps the verb-table principles of [DEC-017](../../concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table): top-level verbs from one table that drives CLI help, dispatch and the MCP tools, with the same arguments.

| Verb | Purpose |
| --- | --- |
| `check` | Compile the model and check coverage and the registry. |
| `context <package>` | Preview a Manager's context package with inclusion reasons. |
| `apply [--continue <assignment>]` | Start or continue an Apply. |
| `status [run]` | Show the assignment and run overview, or one run. |
| `inspect <run> [task]` | Show the task tree, timeline and per-step input, report, difference and evidence. |
| `answer <question>` | Bind an owner answer to a question version. |
| `stop <run>` | Request a stop and report the observed result. |

Inside `run`, start with coherent files such as `task.go`, `coordinator.go`, `review.go`, `budget.go`, `lifecycle.go`, `history.go` and `ports.go`, and split only for a distinct responsibility. Avoid shared Project DTOs, global interface catalogues and untyped property bags. The root `cmd` and `internal` layout replaces the current `src` prefix ([Go module layout](https://go.dev/doc/modules/layout)).

## Dependency direction and ports

| Package | May import |
| --- | --- |
| `internal/model/`, `internal/snapshot/` | No other target package |
| `internal/organization/`, `internal/realization/` | Model, Snapshot |
| `internal/run/` | Model, Snapshot, Organization, Realization |
| `internal/apply/` | Model, Snapshot, Organization, Realization, Run |
| `internal/adapters/{cli,mcp,gitrepo,journal,codexappserver,checks}/` | Model, Snapshot, Organization, Realization, Run, Apply; `codexappserver` and `checks` also `process` |
| `internal/adapters/process/` | No other target package |
| `internal/bootstrap/` | Every target package except commands |
| `cmd/markitect/` | Bootstrap |

Organization and realization never import each other, and the compiler depends on neither. Adapters do not import each other except for `process`. Target packages never import current packages and vice versa.

| Port | Consumer | Concrete work |
| --- | --- | --- |
| `apply.Source` | `apply` | Capture model bytes, explicit dependencies, configuration, registry, repository census and the source binding. |
| `apply.Control` | `apply` | Acquire the repository controller and store, supply the journal, check the lock and release it. |
| `apply.Publisher` | `apply` | Recheck the captured source, publish the verified artifacts and registry together, report actual writes and read-back. |
| `run.Sessions` | `run` | Start a configured role, supply tasks and context, observe messages and tool requests, resume, cancel and establish termination. |
| `run.Workspaces` | `run` | Materialize an owned candidate, observe actual changes, close or retain it. |
| `run.Checks` | `run`, `apply` | Execute explicit project checks on the exact candidate, for reviews and for the aggregate. |
| `run.Journal` | `run`, `apply` | Durably append events and exact payloads under one writer. |

Interfaces live with their consumers, and adapters return concrete types ([Go code review comments](https://go.dev/wiki/CodeReviewComments#interfaces)); one adapter may satisfy several narrow ports. `adapters/journal` implements both `run.Journal` and `apply.Control`, and `adapters/gitrepo` implements `apply.Source`, `apply.Publisher` and `run.Workspaces`. Effectful calls take `context.Context`; pure computations need no interface to be testable. Avoid generic `common`, `types` or `interfaces` packages ([package names](https://go.dev/blog/package-names)).

The Host owns the task and message protocol, effective authority and history. An Agent adapter translates provider transport into those records; it cannot accept a delivery, grant a path or pass a review. Manager and reviewer use the same session port with different roles. Provider wire objects and session IDs never become model types. The registry belongs to `realization`, outside the Definition roots, and is published together with its artifact tree.

The [library walkthrough](manager-apply-library-flow.md) follows a root task and its lending child through these boundaries, from model input to publication.

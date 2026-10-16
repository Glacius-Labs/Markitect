# Manager organization and reliable Apply

This is the product concept for realizing a canonical model through Managers. It owns organization, context, delegation, deliveries, review purposes, artifact ownership, the Apply lifecycle and reporting. The [design map](README.md#apply-design) lists the technical contracts that refine it and states the status of the whole set. The [north star](manager-realization-north-star.md) keeps the ideas and variants that this concept did not select.

The concept uses the [completed language v1](minimal-language/language-contract.md) without new Core Kinds or semantics. The owner decisions behind it are [DEC-025 to DEC-036](../../concepts/register.md#dec-025-package-managers-organize-realization).

The product is an organization that realizes an explicitly described project world. The canonical model supplies intended meaning and constraints. A deterministic Host organizes work, supplies context, validates deliveries and records the process. AI Managers make bounded judgments, delegate work and integrate results. The first operation is Apply: reconcile the existing artifacts with the current model, including when the model has not changed.

## Selected choices

| Question | Choice | Reason |
| --- | --- | --- |
| Organization | One ordinary root Manager and one Manager per local Package. | Responsibility follows the structure the author already maintains. |
| Code, tests and documentation | One hierarchy responsible for all artifact forms. | A feature's implementation, tests and documentation stay coordinated. |
| Default team | The Manager does its local work; a fresh independent reviewer assesses each delivery and integration. | The smallest team that keeps independent review. |
| Final check | The Host's deterministic evidence check after the root's integration review; no separate verifier Agent. | The root's integration review already judges the whole; the Host makes sure all evidence is current. |
| Parent knowledge | Summaries first; targeted, recorded source inspection when needed. | Local responsibility without depending entirely on child claims. |
| Context | Own Definitions and contracts, explicit neighbors, Definitions whose modeled scope covers the Package, and task facts. | Simple, inspectable compilation without semantic retrieval. |
| Order | The root plans the delivery order from the cross-Package references: producers before consumers, the skeleton first. | Consumers build on accepted producer deliveries. |
| Acceptance | Apply starts from the committed model and writes its verified result into the working tree; the owner's commit accepts it. | Acceptance stays with the owner, as for the model. |
| First reconciliation | Assess the full selected local model; reviewed no-change deliveries are allowed. | Reliability does not depend on an unproven incremental invalidation scheme. |
| Execution | One writing task at a time, in owned candidate workspaces. | Reliable integration before concurrency. |
| File ownership | One accountable Manager per artifact; many-to-many links between artifacts and Definitions. | Write responsibility is separate from what an output realizes. |
| Prompts and history | Record the actual inputs and observed events; derive readable reports from them. | Work is inspectable without sending every transcript to every Agent. |
| Other commands | Later operations over the same organization. | The first delivery concentrates on Apply. |

## Four kinds of input and record

| Concern | Owner and meaning |
| --- | --- |
| Canonical model | Authored Definitions, contracts, bindings and relationships: what should hold. |
| Organization configuration | Project-owned execution choices: teams, profiles, limits, instructions and context presentation. It cannot redefine canonical meaning ([configuration](manager-apply-configuration.md)). |
| Realization registry | Tool-maintained, versioned metadata on artifact ownership and justified links to Definitions. It describes the actual realization, not a second intended world. |
| Execution history | Immutable records of inputs, communication, candidates, reviews and outcomes ([journal](manager-apply-journal.md)). |

Binding intent lives only in the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)). The test is simple: if violating something makes an artifact wrong, it is model content. That includes realization constraints such as a public interface, the programming language or required build and test commands; the library's [callable interface](library-example/m1/model/library/callable-interface.yaml) is an example. Configuration and instructions hold only execution choices and operational guidance, such as profiles, limits, reporting style or tool usage. A reviewer who finds a project rule in configuration or custom instructions records a finding, and the rule moves into the model.

Configuration and registry stay outside the canonical Definition roots; v1 would otherwise require a Definition envelope. The compiler produces a bound structural model and diagnostics and does not interpret prose as rules. The Host applies deterministic workflow rules to explicit snapshots. Agents interpret meaning and produce reviewable work. No prompt or report can change the operation target or authorize a model edit during Apply.

## Authoring and inspectable preparation

A small optional software vocabulary and the complete [library example](library-example/README.md) are ordinary, versioned models. Derive the vocabulary from the example's needs instead of building a universal catalogue. It has no privileged compiler behavior; authors can use their own terms, replace it or extend it. Templates produce visible, editable Definitions in the uniform YAML form, not a second authoring language.

Before Agents start, the author can preview what each Manager will receive: which statements, contracts, responsibilities and explicit connections, and why each is included. Original model content, deterministic explanations and later Agent judgments stay distinguishable. The preview exposes mistaken context and missing connections; it cannot establish semantic completeness. Preparation is part of every Apply; there is no requirement to precompile contexts for every possible task.

Add structure only when it exposes a concrete failure possibility or saves recurring human work. Prefer clearer instructions and views over new roles or parallel records.

## Organization and configuration

The root Manager follows the same protocol as every other Manager. It has its own runtime identity for the whole selected local model; v1 forbids a root Package, so none is invented. Top-level Package Managers are its children, and parents follow folder ancestry. Intermediate Packages have ordinary Managers even without local Definitions. The one exception is a pass-through: an intermediate Package without local Definitions and with exactly one child starts no Agent. The Host hands the task down and the child's delivery up, because there is nothing to integrate. Every local Package has a Manager slot; an empty leaf has no reconciliation work and starts no Agent. The run binds the derived topology as execution metadata; empty folders stay out of the canonical model digest as v1 specifies.

Imported models supply read-only vocabulary and context and create no Managers. Editing a library is a task in that library's own project. One local model is the first supported organization.

Configuration has project defaults plus exact Manager overrides and separate hard ceilings. An override for lending does not configure lending/reminders; restrictions in a delegated task are a separate runtime narrowing. The [configuration contract](manager-apply-configuration.md#resolve-settings-deterministically) owns resolution. Configuration cannot disable required delivery or integration review, the final evidence check, coverage or source-drift guards.

| Configurable part | First design |
| --- | --- |
| Team | Manager plus independent reviewer. Implementer roles, reviewer per Definition and separate artifact hierarchies are later strategies. |
| Execution profiles | Provider, model, runtime arguments and instruction files per role, with exact Manager overrides. |
| Scope of shared rules | The model states a rule's scope through a Property whose values are local Package paths, for example `appliesTo: [lending]`, or `[.]` for the whole model. Configuration only names which Kind and Property carry scope; it never lists individual Definitions. |
| Owner approval | Optional list of Packages whose changes wait for the owner's approval before publication. |
| Review batching | Optional: one reviewer session may assess several no-change deliveries, each with its own bound verdict. |
| Artifact creation | Permitted creation areas and exclusive reservation of new paths. Artifact paths need not mirror Package paths. |
| Limits | One shared assignment ledger and finite rounds, owned by the [execution contract](manager-apply-execution.md). |
| Concurrency | One active writing task. Later parallelism must use the same reservation and integration rules. |

### Project-level artifacts and the skeleton

Ownership follows creation: the Manager that reserves a new path owns it. For a project-level artifact, the requiring Definition decides who creates it. In the library, `library/callable-interface` requires the Go module, the entry point and the build and test commands, so the `library` Manager creates and owns them. When Definitions in several Packages require the same artifact, their nearest common parent decides the owner and records the decision; that can be the root. A file that no Definition justifies is never owned: it is a Markitect input or explicitly ignored, so CI configuration, for example, needs a Definition before Apply manages it.

When a required skeleton is missing, its owner delivers it first: module, entry point and the conventions the model requires. The other areas then add their parts within their own paths and expose internal APIs. A Manager whose artifact integrates other areas, such as the entry point that dispatches every operation, receives a second task after those areas have delivered, to wire them in. The skeleton and the wiring are ordinary reviewed deliveries, not special phases.

## Tasks and delegation

Only the root receives the operation's user instruction. For Apply, the Host adds the invariant that artifacts must conform to the selected model and that model content is read-only. Each parent creates tasks for its child Packages. The Host validates the parent-child route and schedules invocations; Managers do not launch unrecorded tasks.

| Task field | Meaning |
| --- | --- |
| Identity and parent | Run, task, parent task, issuing and receiving Manager, attempt. |
| Outcome | What the receiving area is expected to accomplish. |
| Scope and basis | Responsible Package or subtree, bound model references, candidate base and operation target. |
| Constraints | Relevant restrictions and agreements from the model and the enclosing task. |
| Completion | Required result, reviews, checks and open questions to resolve. |
| Inputs | Context package, prior reports and explicitly supplied additional information. |

Delegation states outcomes and model obligations, not a file-edit plan. Existing write ownership comes from the registry; new paths are reserved through the Host. The root may state an interface constraint the model already requires, but it does not choose the child's files. A reviewer judges whether a prompt takes over the child's implementation decisions; no syntax check can prove that.

The root plans the delivery order. The Host supplies the cross-Package reference graph with its directions as planning input, and the root records an order with reasons: the owner of the skeleton first, then producers before their consumers. A Manager may receive several tasks in one run. In the library the order is library for the skeleton, then members and catalog, then lending with reminders, and finally library again to wire the operations into the entry point. References often point both ways, so the order is a recorded judgment, not a computed sort. The Host enforces only that a consumer task starts from accepted producer deliveries ([execution](manager-apply-execution.md#accepted-deliveries-and-dependent-work)); the root may revise the order with a recorded reason.

Each area owns its internal API. A consumer uses only the producer's API and never reaches into its files or data; it requests a missing or changed API through the common parent. The integration review checks that consumers use only producer APIs and that those APIs satisfy the connected Definitions.

Sibling requests go through the nearest common parent. A consumer can request a contract or service change but cannot edit the producer's files. Requests, conflicts and agreements are recorded. If an agreed change invalidates another area's completed assessment, that Manager gets a new task against the updated candidate. Unknown consequences broaden review. A conflict or dependency cycle that cannot be resolved within the run is returned as an open decision, not an endless exchange.

## Context packages

Each context package is an immutable rendered input plus source references. Its bytes and the actual outgoing invocation are recorded. Preparation is deterministic for its inputs; later semantic conclusions are attributed Agent judgments.

The initial Manager package contains:

1. Its task and role, the mandate (responsibility, binding basis, freedom, artifact authority, reserved choices and limits, as the [configuration](manager-apply-configuration.md#the-mandate-section) renders it), effective configuration, operation target and review requirements.
2. Every local Definition in its Package, in stable path order: the filename as heading, then identity, Kind, complete body and structured `spec`, plus the bound Kind and Property contracts.
3. A See also section: direct outgoing references and body links, and direct incoming references to the Package, each with reason and direction. It includes the directly connected Definitions and their type contracts; further references stay available to inspect.
4. Every Definition whose modeled scope covers this Package, because its scope lists the Package, one of its ancestors or `.`, as a read-only view of the bound Definition with its reason for inclusion. It does not extend import resolution.
5. A compact local Package directory for routing questions, and a child index with identities, purposes, task states and latest validated reports. It does not load foreign Definitions, artifacts or transcripts.
6. The Manager's artifact inventory and relevant candidate files for local work. A Manager that integrates children first receives their delivery summaries, agreements and evidence references.

A rendered package larger than the configured limit stops preparation with a finding to split the Package; a package is never truncated silently. The role prompts for root, Manager and reviewer are product-owned, versioned templates. Their identity is recorded with every invocation, changes to them go through review, and custom instructions are only appended.

An implementer receives the local task and candidate inputs. A reviewer works in fresh context without the implementer's conversation: its first step receives the task and the relevant model contracts only and records the expected behavior and discriminating cases; the second step supplies the actual candidate and evidence. The Host checks that a reviewer's rendered package contains what its role requires; the reviewer judges whether that context suffices.

Managers can request further Definitions, reports, transcripts or artifact files by identity; the request and the returned content are recorded. A coordinating parent is not loaded with descendant source files initially but can inspect them to resolve a conflict or judge integration. Need to know keeps Agents focused; it is not an operating-system isolation boundary.

Normal context discovery follows explicit references. A body link marks a connection, not proof that every obligation at its target applies. A missing necessary connection is a modeling defect to surface, not permission to ignore a known obligation. Apply never invents canonical links. The optional Optimizer belongs to a separate model operation ([north star](manager-realization-north-star.md#missing-links-and-the-optimizer)).

## Checking effects across Package boundaries

During local assessment, the Manager and its reviewer both ask: which statements in this area require behavior from another area, and is that connection visible in the model? For example, members states that suspended members cannot borrow, while borrowing belongs to lending. Reading the local statement is enough to raise the question; no semantic search is required.

Raise an ordinary open question naming the source Definition and passage, the possibly affected Package and the suspected effect, and send it through the nearest common parent. If no target is known, report it to the parent without inventing one. Apply can report the gap but cannot add the link; a model correction is a separate model operation followed by a new Apply. An unresolved question that prevents a sound assessment blocks completion of the affected work and therefore of the aggregate.

This is a review duty over the statements actually assessed, not a guarantee that every missing connection is found.

## Deliveries and evidence

Every Manager, including the root, returns the same delivery shape:

| Part | Required account |
| --- | --- |
| Binding | Task, attempt, source base and exact candidate. |
| Assessment | Each local Definition and how it relates to the realization. |
| Artifact changes | Actual harvested delta, registry changes and ownership claims. |
| Explanation | What changed, why, and which Definitions justify it. |
| Delegated results | Child deliveries and the aggregate candidate they were integrated into. |
| Evidence | Independent review verdicts, checks and exact candidate bindings. |
| Open work | Unresolved questions, conflicts, missing evidence and requests to other areas. |

The Host validates shape, scope, candidate bytes and lifecycle. A report cannot substitute claimed changes for the observed workspace delta.

Definition assessments are a small table, not a second semantic model. Each record holds the Definition, a disposition (`realized`, `constrains`, `vocabulary` or `unresolved`), the artifacts, the evidence and a reason. `realized` names the artifacts that realize the Definition; `constrains` names the other realization it constrains; `vocabulary` applies to declarations that need no standalone artifact. A positive disposition explains the observable realization or concrete use and why the evidence suffices; "done" or a file list is not enough. For behavior, the evidence names the tests or checks that would fail if the Definition were violated, and the reviewer judges whether they really discriminate. The Host checks the census, statuses, bindings and evidence references; the reviewer judges their substance. Every local Definition needs a disposition other than `unresolved`; an unresolved one prevents completion even when every file is registered.

The `vocabulary` disposition means a declaration needs no standalone artifact. It does not remove obligations its body or contracts impose on concrete uses; cite those uses and their assessments. The table shows what was examined; it does not prove that every natural-language obligation was understood. All artifact forms, including tests, are reviewed against their claimed model basis.

An unchanged Package can deliver a reviewed no-change result that still explains its realization and evidence. An old successful report is evidence for its old inputs, not current approval.

## Distinct review responsibilities

| Review | Question it must answer |
| --- | --- |
| Local delivery | Do this area's code, tests and documentation fulfill its Definitions and relevant connected obligations? Are effects on other areas accounted for? |
| Integration | Does the composed candidate satisfy the connected behavior and contracts across areas, such as a suspension preventing a new loan? At the root it also asks whether all areas follow the model's project-wide conventions uniformly. |
| Final evidence check | Deterministic, by the Host: does the exact aggregate have every required local and integration review bound to its current subject, passing required checks, conforming coverage and a complete Definition census, with no open blocking finding? |

The root's integration review is the semantic review of the whole system; no separate verifier Agent repeats it. Every review works in these two steps, so the model is read before the solution, and the recorded expectations show it. Project-wide conventions, such as how errors are reported, are model Rules with project scope, so the root's integration review can check them across areas. Where the configuration allows it, one reviewer session assesses several no-change deliveries, each with its own verdict bound to its subject.

Each result is bound to its purpose and candidate. A local pass never replaces an integration review. Later stages can cite an earlier finding and its recorded resolution instead of repeating the discussion; a changed candidate, scope or assumption requires renewed assessment.

For the library demonstration, independent M1 and M2 expectations are frozen outside Agent write scope, so generated code and tests cannot redefine success together ([evaluation contract](library-example/evaluation-contract.md)). This evaluates the experiment; adopting projects do not maintain a second specification.

## Artifact registry and ownership

The registry lives in `.markitect/registry/`, one file per Manager that mirrors the Package folders: `.markitect/registry/lending/registry.yaml` for lending, `.markitect/registry/registry.yaml` for the root. The file determines the accountable Manager. Each record names its path, artifact form (`code`, `test`, `doc`, `build` or `config`), the Definitions it realizes and a short explanation:

```yaml
apiVersion: markitect.registry/v1
artifacts:
  - path: realization/internal/lending/borrow.go
    form: code
    realizes: [lending/borrow, lending/loan-policy]
    why: Admits a new loan only within the open-loan limit.
```

Only guarded publication writes registry files. An edit made outside Apply is drift: `check` validates it, and the next Apply's intake reassesses it. A registry revision is bound to the candidate tree. Review and check observations refer to that revision instead of being copied into mutable verdict fields.

Registry paths are independent of Package paths; lending can own source, test and documentation files in three directories. Other Managers can read or cite those files but cannot claim them. Managers propose registration through Host tools; the Host validates them and builds the registry delta. Editing the registry file directly is not a way around ownership.

Before writing a new path, a Manager reserves it through the Host, which checks the operation target, permitted creation areas, exact spelling and exclusive ownership. Model and configuration paths cannot be obtained through an artifact reservation. A shared file has one owner; others request changes through their common parent. A rename needs a new reservation and keeps provenance. A deletion removes the current registration only with a justified candidate deletion; history keeps the old registration. Moving responsibility between Packages is an explicit, recorded transfer reviewed with the affected delivery.

Coverage classifies every repository file as a canonical or Markitect input, a registered artifact or an explicitly ignored file ([DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). Registrations need a valid owner and a resolvable model basis. Overlapping ownership, missing files, stale registrations and unexplained files are findings. A blanket ignore cannot turn missing realization into success. Ignore configuration is project-owned, fixed for the run and read-only during Apply; Managers cannot hide a file by adding an ignore. Coverage ignores never change the language loader's complete scan of Definitions.

### Markitect state

Markitect's own state stays out of the realization: the journal lives outside the repository, and the small control metadata in `.markitect/state/` is ignored by Git and excluded from source staleness ([journal](manager-apply-journal.md#physical-store-and-commit-contract)). Repository classification and ownership belong to the Host's coverage stage, which keeps the language compiler free of I/O.

### Removed and moved intent

Apply also removes realization whose intent is gone ([DEC-034](../../concepts/register.md#dec-034-removed-intent-removes-its-realization)). When a Definition disappears, every artifact whose only basis it was is orphaned. Its owner deletes it in the same Apply, or records a new justified basis that review accepts; an artifact that keeps other bases only loses the removed link. If the owning Package itself disappeared, its records pass to the nearest existing ancestor Manager, or to the root. Language v1 does not guess renames (section 9), so a moved or renamed Definition is a removal plus an addition: the link to the old identity becomes a blocking finding, and the owner re-registers the artifact against the new identity with review. When the move crosses Packages, the record moves to the new owner's registry file through a recorded transfer. Stale code from removed intent is not a matter of style; leaving it fails the reconciliation.

### Intake of unexplained files

Initially unexplained files go through read-only intake coordinated by the root: candidate registrations are justified against Definitions and independently reviewed before write tasks use them. The root may delegate that investigation. Ambiguous responsibility remains a decision, not an automatic claim. Final publication requires conforming coverage of the aggregate candidate.

## Apply lifecycle

Logical Apply is the whole reconciliation below; guarded publication is its last step.

| Stage | Work and exit condition |
| --- | --- |
| Prepare | Read the model, configuration, registry and ignore policy from the committed state. Uncommitted changes to tracked files, or untracked files outside Git-ignored paths, stop the run before preparation. Acquire the repository snapshot and tool identities, compile v1 and bind the organization. Invalid model inputs stop the run. |
| Start root | Give the instruction and a bounded project overview to the root. Fix the artifact-only target and settle unresolved ownership through intake before write tasks begin. |
| Assess and delegate | Every local Package with Definitions is considered. Managers assess local realization and issue outcome-based child tasks. |
| Work and review | Local changes happen in an owned candidate workspace. The Host harvests the actual delta; a fresh reviewer assesses it. A failed review returns grounded rework; an unresolved model question pauses that task without editing the model. |
| Integrate upward | The parent composes deliveries on an exact base and checks interface and cross-area consistency. A fresh integration reviewer examines the aggregate and relevant child evidence. |
| Verify aggregate | At the root candidate: required project checks, then the Host's final evidence check over the whole Manager tree. The root's fresh integration review has already assessed the aggregate semantically. |
| Preflight | Recheck the captured repository, model, configuration, registry and ignore inputs. Drift makes the run stale; earlier checks cannot authorize changed bytes. |
| Publish | If the configuration requires the owner's approval for an affected Package, the Host raises an approval question with the result summary, and the run waits for the answer. The guarded writer then applies the accepted candidate and registry revision to the working tree. Apply never commits, merges, releases or deploys. |
| Read back | Confirm actual paths, bytes, modes and registry against the verified candidate. A partial write or failed read-back is not a successful Apply. |

Full reconciliation of the selected local model is the default; it does not claim coverage of intent outside the model. Do not implement incremental reuse first; [computed scope](#toward-computed-scope) follows later. The graph and registry can explain possible impact; unknown semantic impact broadens examination.

Later tasks build on the current accepted integration base, and every delivery carries its exact base. Because every file has one owner and one task writes at a time, composing deliveries is a disjoint union of file sets; an overlap is a Host error that stops the run, not a merge. Parents compose validated child deltas but cannot author child-owned files; a correction to child content returns to that child as a task. Parent-owned integration artifacts can be changed by the parent. When composition changes a reviewed subject or its assumptions, the affected reviews and checks are renewed. No last-writer-wins merge or silent reassignment resolves a conflict.

The owner's commit of the published result is the acceptance, exactly as a committed model revision is the accepted specification ([DEC-035](../../concepts/register.md#dec-035-committed-model-in-owner-commit-accepts-the-result)). The run report proposes a commit message with the trailers `Markitect-Run` and `Markitect-Model`. After a partial publication or a result the owner does not accept, the owner commits or reverts the working tree before the next run.

Review rounds are finite, and after repeated rejections the parent assesses the plan; the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction) owns counting, budgets and exhaustion. A parent cannot turn an unresolved child delivery into a complete one by summarizing it.

The [execution contract](manager-apply-execution.md#one-run-state-machine) owns the single run state machine and its [terminal outcomes](manager-apply-execution.md#terminal-outcomes-and-displayed-status). History survives interruption: reopening a run first reconciles recorded invocations, child lifecycle and publication outcomes. Completed provider work is not replayed automatically, and unknown side effects stay visible.

## Reports and execution history

One run history serves two views: concise upward reports for coordination and a drill-down record for investigation. Reports reference evidence; they do not replace it. A parent can request underlying sources, candidate differences and conversations.

At minimum the history records run, task, attempt and Agent identities, causal links, effective role configuration, model and candidate bindings, prepared context, actual rendered prompts and custom instructions, sent requests, returned messages and reports, context and tool reads, delegation, reservations, candidate deltas, review rounds, checks, integration, publication and read-back. Original events are kept; corrections are new events. Large payloads are content-addressed blobs. Prepared, sent and confirmed or unknown delivery states stay distinct, so a built template never counts as a received request. Opaque provider behavior is a visibility limit, not a transcript Markitect may invent.

Every candidate transition identifies its prior and resulting tree and a retrievable difference. Records distinguish a proposed change, an Agent claim, a Host-observed change, a reviewed aggregate and an applied result. Rejected candidates stay inspectable. Parallel causal branches are not flattened into a false order by timestamps.

The normal user report presents the result, model basis, evidence, limitations and decisions needing attention. A small coordination account from the same records shows local work and no-change outcomes, cross-area requests, invocations, known usage and elapsed time. It separates a Manager's own effort from descendant work and waiting, and unknown usage stays unknown. It always reports the same measures: Agent steps, sessions and tokens per role, review rounds per purpose, findings by kind and severity, the share of no-change deliveries, owner interventions and active execution time. These feed the evaluation of [DEC-009](../../concepts/register.md#dec-009-the-central-evaluation-question) and [PRM-005](../../concepts/register.md#promises). The account helps judge whether one Manager per Package works in practice; it adds no scores, thresholds or automatic reorganization.

A failure to record required work or publication events durably stops further dispatch or publication. Recorded inputs and outputs support inspection; they do not guarantee replay, semantic correctness or human acceptance. The first inspection surface is a run summary, the Manager task tree, an event timeline and per-step access to input, report, candidate difference and evidence. No cockpit is needed for the first Apply.

## Library walkthrough

The [library delegation walkthrough](manager-apply-library-flow.md) traces the root, lending, a rejected six-loan candidate, fresh review, rework, reminders and integration against M1. The [library M1](library-example/README.md) has the Packages library (shared scope and rules), catalog, members, lending and lending/reminders, with explicit connections from membership and availability to lending. The same lifecycle covers drift repair: if the existing lending code, its test and the usage guide allow a sixth loan while the model is unchanged, Apply realigns all three against the five-loan rule without a model edit.

## Design checks and failure handling

These counterexamples check the contracts; each becomes a test in the package that owns it.

| Situation | Required behavior |
| --- | --- |
| Artifacts drift while the model is unchanged | Apply assesses and reconciles them; no artificial model change is needed. |
| A binding rule has no needed context link | If noticed, report the modeling gap. Neither edit the model nor treat the missing link as cancelling the rule. |
| A local statement seems to require another area's behavior | Manager and reviewer raise a source-grounded question through the common parent. An unresolved required assessment blocks completion. |
| Two Managers claim the same file | Reject the overlapping claim and coordinate one owner. Output order never picks a winner. |
| A worker changes a canonical Definition during Apply | Reject the candidate as outside the operation target. |
| A `vocabulary` disposition hides a required feature | Independent review rejects the unsupported assessment. |
| A Kind needs no artifact but constrains its uses | Assess the concrete uses and their realization. |
| Every file is registered, but a sixth loan is still allowed | Coverage can pass; semantic review and domain checks must catch the behavior. |
| Every file is owned, but a Definition has no justified disposition | Refuse completion. |
| Child review passed, but composition breaks the shared contract | Integration review fails; the owning area repairs and the new candidate is revalidated. |
| A local pass is offered as integration or final evidence | Reject the substituted review purpose. |
| A parent needs a child's file to judge integration | Supply a recorded targeted read. There is no file-blindness rule. |
| An external edit changes the checkout or model during work | Preflight reports stale and preserves the candidates. |
| Review cannot converge or a model contradiction remains | Finish Incomplete within the bounded rounds, with the open question and retained evidence. |
| A Definition was removed, but its artifacts remain | Their owner deletes them or records a new basis that review accepts; otherwise the run cannot complete. |
| A Definition moved to another Package | The link to the old identity is a blocking finding until the owner re-registers the artifact; a move across Packages transfers the record. |
| The model or another file has uncommitted changes when Apply starts | Refuse before preparation. |
| Configuration or custom instructions state a project rule | The reviewer records a finding; the rule moves into the model. |
| Publication writes some files and then fails | The run reconciles until the actual writes are known. A known partial result finishes Failed with its exact paths. |
| An Agent reports success without a recorded delta or evidence | The Host refuses completion. |
| A required event cannot be stored durably | Stop before further dispatch or publication. |
| A rendered context exceeds the size limit | Stop preparation with a finding to split the Package. |
| Composed deliveries touch the same file | Host error; ownership is broken and the run stops. |
| A consumer reads another area's files or data directly | The integration review records a finding; the consumer must use the producer's API. |
| A provider call fails before anything was sent | Retry automatically within the configured limit, each attempt charged; otherwise reconcile. |

## Toward computed scope

Computed scope, the vision's first mechanism, follows once the two-Apply cycle works. It derives the affected Managers from the model difference per Definition, through the direct references and the registry links. Only those Managers receive tasks, and unknown impact still broadens the scope. Managers outside it get no task, and their last accepted realization stands. Because declared references cannot show every code dependency ([ENH-001](../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact)), a full reconciliation stays available and advisable from time to time. The data this needs is recorded from the first run on: Definition digests per run, registry links and the reference graph.

## Relation to current source and delivery

The [rebuild plan](manager-apply-rebuild-plan.md#source-replacement-map) decides which current mechanisms are reused, rewritten or deleted; the existing Reconcile operation already asks for complete responsibility against unchanged intent and reasoned no-change results. The [delivery plan](../../implementation-plan.md) owns packages and order, and the [acceptance gates](manager-apply-implementation-plan.md) own what each step must prove. Further commands, team strategies, concurrency and incremental reuse wait until the two-Apply cycle is demonstrated. Government remains a later, separate authority over model changes ([DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)).

# Manager organization and reliable Apply

Date: 16 October 2026. **Developed design proposal for owner review.** This concept resolves the working alternatives in the [north star](manager-realization-north-star.md) into a first design for reliable Apply. It uses the [completed language v1](minimal-language/language-contract.md) without adding Core Kinds or changing its semantics. Proposed choices here are not additional owner decisions or implemented features.

The owner's later [implementation clarification](sources/manager-realization-20261016.md#two-apply-cycle-and-freedom-to-redesign) sets the demonstration and rebuild policy: fully author M1, generate and inspect its realization, change to M2, Apply to the existing result and inspect again. Compatibility is not required; obsolete source and workflows may be deleted. The [implementation plan](manager-apply-implementation-plan.md) puts the Go architecture, Playground and actual Agent communication ahead of substantial execution work. Existing implementation suggestions below are reuse candidates, not requirements to preserve old code or APIs.

The [supplied independent review and its resolution](review-resolution-20261016.md) refine authoring, missing connections, Definition assessments, review responsibilities and observable coordination effort. These refinements use the existing roles and records; they do not add language constructs or a second organization.

The product is an organization that realizes an explicitly described project world. The canonical model supplies that world's intended meaning and constraints. A deterministic Host organizes work, supplies context, validates deliveries and records the process. AI Managers make bounded judgments, delegate work and integrate results. The initial operation is Apply: reconcile the existing artifacts with the current model, including when the model has not changed.

## Recommended choices

| Question | Proposal | Reason |
| --- | --- | --- |
| Organization | One ordinary root Manager and one Manager per local Package. | Derive responsibility from the structure the author already maintains. |
| Code tests and documentation | One hierarchy responsible for all artifact forms. | Keep a feature's implementation, tests and documentation coordinated without multiplying ownership trees. |
| Default team | Manager performs local work; a fresh independent reviewer assesses its delivery and integration. | Start with the smallest team that preserves independent review. Separate implementers remain configurable. |
| Parent knowledge | Summaries first; targeted source inspection available and recorded. | Preserve local responsibility without making integration depend entirely on child claims. |
| Context | Own Definitions and contracts, explicit neighboring references, shared declared context and task-specific facts. | Start with simple, inspectable compilation and no semantic retrieval prerequisite. |
| First reconciliation | Assess the full selected project model, with no-change deliveries allowed. | Avoid making reliability depend on an unproven incremental semantic invalidation scheme. |
| Execution | Sequential child delivery by default, in owned candidate workspaces. | Establish reliable integration before optimizing concurrency. |
| File ownership | One accountable Manager per artifact; many-to-many links between artifacts and Definitions. | Separate write responsibility from what an output realizes. |
| Prompts and history | Preserve the actual supplied inputs and observed events, with readable reports derived from them. | Make work inspectable without sending all transcripts to all Agents. |
| Other commands | Later operation definitions over the same organization. | Keep the first delivery focused on Apply. |

## Four separate kinds of input and record

| Concern | Owner and meaning |
| --- | --- |
| Canonical model | Authored Definitions, their contracts, bindings and relationships describe what should hold. |
| Organization configuration | Project-owned execution choices select teams, providers, budgets, instructions and context presentation. They do not redefine canonical meaning. |
| Realization registry | Tool-maintained, versioned metadata describes artifact ownership and justified links to Definitions. It describes the actual realization, not a second intended-world model. |
| Execution history | Immutable records describe the inputs, communication, candidates, reviews and outcomes of particular runs. |

Configuration and registry data remain outside the canonical Definition roots. They must not be disguised as extra YAML files under those roots, where v1 would require a Definition envelope. Exact serialized schemas should be designed with the implementation; they are ordinary runtime contracts, not new language constructs.

The compiler produces a bound structural model and diagnostics. It does not interpret arbitrary prose as a formal rule system. The Host reads explicit snapshots and applies deterministic workflow rules. Agents interpret meaning against those sources and produce reviewable work. Neither a prompt nor a Manager report can change the Host's declared operation target or authorize a model edit during Apply.

## Authoring and inspectable preparation

Provide a small optional software vocabulary and a complete library example as an ordinary, versioned model. Derive the initial Kinds and Properties from the example's actual needs rather than creating a universal catalogue in advance. The vocabulary has no privileged compiler behavior. Authors can use their own domain terms, replace the starter vocabulary or extend it through the same language rules.

Templates produce visible, editable Definitions in the uniform YAML form; they are not a second authoring language. Before starting Agents, preparation should let an author inspect which original statements, contracts, responsibilities and explicit connections each Manager will receive, with the reason for each inclusion. Keep original model content, deterministic context explanations and later Agent judgments distinguishable. This preview helps expose mistaken context and missing connections; it cannot establish semantic completeness.

Add structure only when it exposes a concrete failure possibility or saves recurring human work. Prefer clearer work instructions and views over new roles or parallel records.

## Organization and configuration

The root Manager is the same role and follows the same protocol as every other Manager. It has a distinct runtime identity for the whole selected local model; it does not invent a root Package, which v1 forbids. Top-level Package Managers are its children. A Package's parent Manager follows the folder ancestry. Intermediate Packages have ordinary Managers even when they have no local Definitions.

Every local Package has a Manager slot. An empty leaf has no reconciliation work and needs no Agent invocation. Package identities and the acquired local folder inventory bind the derived organization for the run. This topology binding is execution metadata; empty folders remain excluded from the canonical model digest as specified by v1.

Imported models supply read-only vocabulary and context. They do not automatically create executable Managers. Editing a library is a task in that library's own project. One local model is the first supported organization; combining several independently authored local organizations is a later extension.

Configuration has one project default and exact Manager overrides. The root has an explicit override entry like any Manager. Runtime default resolution is deliberate configuration behavior; it is not Package inheritance in the canonical language.

| Configurable part | First design |
| --- | --- |
| Team | Manager plus independent reviewer by default; optional local implementer roles. |
| Execution profiles | Provider, model, runtime arguments and instruction files per role, with per-Manager overrides. |
| Shared context | Explicit Definition references to include for every Manager, plus per-Manager additions. Inclusion provides context; it does not itself grant a statement global applicability. |
| Kind guidance | Optional task and review recipes keyed by resolved Kind identity. They use the bound Kind's contract, do not change it and grant no special Core execution rights. |
| Artifact creation | Permitted creation areas and exclusive reservation of new paths, without assuming that artifact paths mirror Package paths. |
| Limits | Finite run and invocation budgets; three candidate review rounds per work or integration task by default. |
| Concurrency | One active writing task by default. Later bounded parallelism must use the same reservation and integration rules. |

The first implementation supports the default team and explicit profiles. Reviewer-per-Definition and multiple artifact-specific hierarchies remain later team strategies. Configuration cannot disable required independent delivery review, aggregate integration review, final verification, coverage or source-drift guards.

## Tasks and delegation

Only the root receives the operation's user instruction. For Apply, the Host also fixes the invariant that artifact realization must conform to the selected model and that model content is read-only. The root creates tasks for child Packages; each child may do the same for its own children. The Host validates this parent-child structure and schedules invocations. Managers do not launch arbitrary unrecorded tasks outside it.

A task is a small runtime record with these conceptual fields:

| Field | Meaning |
| --- | --- |
| Identity and parent | Run, task, parent task, issuing Manager, receiving Manager and attempt. |
| Outcome | What the receiving area is expected to accomplish. |
| Scope and basis | Responsible Package or subtree, bound model references, candidate base and operation target. |
| Constraints | Relevant restrictions and agreements from the model and enclosing task. |
| Completion | Required result, review, checks and open questions to resolve. |
| Inputs | Context package, prior reports and explicitly supplied additional information. |

Delegation specifies outcomes and model obligations, not a descendant file-edit plan. Runtime tools derive existing write ownership from the registry. They reserve new paths following configuration and the receiving Manager's claim. There is no required `filesToEdit` field in the parent's task. A reviewer assesses whether a free-text prompt improperly takes over the child's implementation decisions; no syntax check claims to prove that a prompt is suitably abstract.

For example, the root tells lending to reconcile borrowing with the registered-member and five-loan requirements. Lending decides whether its implementation, tests or documentation need changes. The root can state a concrete interface constraint already required by the model, without choosing the child's internal files.

Sibling requests go through the nearest common parent. The parent coordinates a task or an agreed delivery order for the other Manager. A consumer can request a contract or a service change but cannot edit the producer's files. Pending requests, conflicts and the resulting agreement are recorded. If the agreed change invalidates another area's completed assessment, that Manager receives a new task against the updated candidate. Unknown consequences broaden review rather than preserving an unsupported earlier pass. Scheduling is finite: a conflict or dependency cycle that cannot be resolved within the run is returned as an open decision rather than an endless exchange.

## Context packages

Each context package is an immutable rendered input plus source references. The exact rendered bytes and actual outgoing invocation are recorded. Preparing the context is deterministic for its bound inputs; semantic conclusions produced later remain attributed Agent judgments.

The initial Manager package contains:

1. Its task, role, effective configuration, operation target and review requirements.
2. Every local Definition in its own Package, in stable path order. Use the source filename as a heading, followed by identity, Kind, complete body and structured `spec`. Supply the bound Kind and Property contracts needed to understand the values.
3. A See also section for direct outgoing references and body links, and direct incoming references whose targets are in the Package. Label each reason and direction. Include the direct source or target Definition and its needed type contracts; further references remain available to inspect rather than being expanded without limit.
4. The explicitly configured shared context and per-Manager additions. These are read-only views of already bound Definitions and do not extend file-local import resolution.
5. A compact local Package directory for routing questions, plus a child index with identities, purposes from available model content, current task states and latest validated reports. The directory identifies possible other areas without loading all foreign Definitions, artifacts or transcripts. It does not extend import visibility.
6. The Manager's current artifact inventory and relevant existing candidate files for local work. A Manager that is integrating children initially receives their delivery summaries, public agreements and review/check references.

An implementer receives the local task and candidate inputs. A reviewer receives the actual candidate, full relevant model contracts and evidence, with fresh context rather than the implementer's conversation. Role-specific packages can share source blobs without repeatedly storing duplicate text.

Full source files are available for targeted reading. Managers can request additional definitions, reports, transcripts or artifact files by identity. Record the request and returned content. A coordinating parent is not initially loaded with descendant source files, but can inspect them to resolve a conflict or evaluate integration. This is a context policy, not an assertion of operating-system isolation.

Use explicit references for normal context discovery. A body link marks an explicit reference, not proof that every obligation at its target applies to the source. The surrounding statement and contracts determine applicability. Configured shared context can deliberately expose a project-wide rule; adding that reference is not a hidden new Core rule operator. Missing necessary author connections are modeling defects to surface, not permission to disregard a known obligation. Apply does not invent canonical links to compensate.

The optional Optimizer can later investigate beyond these connections and propose model edits in a separate model-targeted operation. The first Apply does not require a semantic index, embeddings, context ranking or precompilation of every possible task. A separate preparation command can be added later; each Apply prepares its own current inputs.

## Checking effects across Package boundaries

During local assessment, both the Manager and its fresh reviewer ask: which statements in this area require behavior from another area, and is that connection visible in the model? For example, members may state that suspended members cannot borrow, while the borrowing behavior belongs to lending. Reading that local statement is enough to raise a possible missing connection; no automatic semantic search is required.

Use an ordinary open question, identifying the source Definition and passage, the possibly affected Package and the suspected effect. Send it through the nearest common parent for assessment with the affected area. The Package directory supports routing; it does not prove which obligations apply. If no target is known, report the question to the parent without inventing one.

Apply can inspect and report the issue, but cannot add the canonical link. A needed model correction belongs to a separate model operation, followed by a new Apply against that model. An unresolved question that prevents a sound assessment blocks completion of the affected work and therefore of the aggregate; other bounded investigation can still finish. Do not treat absence from compiled context as permission to ignore a binding statement.

This is a review duty over the statements actually assessed, not a guarantee that every missing connection will be discovered. The strict authored-link baseline remains; the optional Optimizer provides wider investigation later.

## Deliveries and evidence

Every Manager returns the same delivery shape, including the root:

| Part | Required account |
| --- | --- |
| Binding | Task, attempt, source base and exact candidate. |
| Assessment | Each local Definition considered and how it relates to the realization. |
| Artifact changes | Actual harvested delta, registry changes and ownership claims. |
| Explanation | What changed, why, and which model Definitions justify it. |
| Delegated results | Child deliveries and the aggregate candidate they were integrated into. |
| Evidence | Independent review verdicts, checks and precise candidate bindings. |
| Open work | Unresolved questions, conflicts, missing evidence and requests to other areas. |

The Host validates the shape, scope, candidate bytes and lifecycle. An Agent's report cannot substitute invented changes for the observed workspace delta.

Definition assessments use a small report table, not a second semantic model. Each Definition is recorded as realized by named artifacts, constraining other identified realization, vocabulary-only, or unresolved. A reason and supporting references accompany each entry. Conditional non-applicability needs its model basis. A reviewer assesses these judgments; a vocabulary-only label is not an automatic escape from implementing behavior.

Every positive disposition explains the observable realization or concrete use and why the cited evidence is sufficient. The Host checks the census, status, reason, model/candidate binding and evidence references; the reviewer assesses their substance. An assertion such as "done" or a list of files is insufficient. A required Definition with unresolved realization prevents completion even if every existing file is registered.

Vocabulary-only means that a declaration needs no standalone artifact. It does not remove obligations that its body or contracts impose on concrete uses. Cite those uses and their realization assessments; explain an unused declaration when appropriate. Broader behavioral statements still need assessment. This avoids both a vocabulary escape hatch and a separate database of atomized requirements.

The table provides a checkable census of what was examined, not a proof that every natural-language obligation was understood. It also avoids demanding one artifact per Definition. Kind and Property declarations can supply vocabulary, while their instance obligations are assessed in the concrete realization. All artifact forms, including tests, are reviewed against their claimed model basis.

An unchanged Package can deliver a reviewed no-change result. The report still explains its realization and evidence. An old successful report remains evidence for its old inputs; it is not silently treated as current approval.

## Distinct review responsibilities

| Review | Question it must answer |
| --- | --- |
| Local delivery | Does this area's code, tests and documentation fulfill its own Definitions and relevant connected obligations? Are possible effects on other areas accounted for? |
| Integration | Does the composed candidate satisfy the connected behavior and contracts across areas, such as a membership suspension preventing a new loan? |
| Final verification | Does the exact aggregate have complete current evidence, required checks and cross-area verification, with no unresolved required findings? |

Bind each result to its review purpose and candidate. A local pass does not replace integration or final verification. Later stages can cite an earlier finding and its recorded resolution instead of repeating the same discussion. Reassess when the candidate, scope or assumptions change; a reference does not waive required independent review of the current aggregate.

For the library demonstration, freeze independent M1 and M2 expectations outside Agent write scope so that generated code and tests cannot silently redefine success together. This is a bounded evaluation arrangement for the experiment, not a requirement for every adopting project to maintain a permanent second specification. Ordinary project tests and their model basis still need independent review.

## Artifact registry and ownership

One registry record per artifact identifies its path, accountable Manager, artifact form, model Definition references and a short realization explanation. A revision of the registry is bound to the candidate tree; the associated source and result digests let readers identify the exact file content. Runtime review and check observations refer to that registry revision rather than being copied into mutable global verdict fields.

Registry paths are independent of model Package paths. For example, lending can own source, test and documentation files in three physical directories. Another Manager can read those files or cite them as context but cannot concurrently claim their write ownership. Managers propose registration operations through Host tools; the Host validates them and builds the registry delta. An Agent cannot bypass ownership checks by editing the registry file directly.

Before writing a new path, a Manager reserves it through the Host. The Host checks operation target, permitted artifact creation areas, exact path spelling and exclusive ownership. New model/configuration paths are not obtained merely by calling an artifact registration tool. Shared files have one owner; other Managers request changes from that owner. Renames preserve explicit provenance and require a new path reservation. Deletions remove current registration only with a justified candidate deletion; historical registrations and evidence remain in the history. Moving responsibility between Packages requires an explicit, recorded transfer reviewed with the affected delivery.

The coverage pass classifies every repository file as an acquired canonical/Markitect input, a registered artifact, or an explicitly ignored file, following the accepted [coverage principle](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored). Artifact registrations need a valid owner and resolvable model basis. Overlapping ownership, missing files, stale registrations and unexplained files are findings. Operational history and scratch storage receive explicit narrow tool classifications or ignore rules with reasons; a blanket repository ignore cannot turn missing realization into success. Coverage ignores do not change the language loader's complete Definition scan.

Ignore configuration is project-owned, fixed for the run and read-only during artifact Apply. Managers cannot hide an unexplained file by creating a new ignore. Unresolved initial ownership becomes read-only intake coordinated by the root: candidate registrations are justified against model Definitions and independently reviewed before write tasks use them. The root can delegate this investigation; it need not inspect every unknown file itself. Ambiguous responsibility remains a decision to resolve, not an automatic claim. Final publication requires conforming coverage of the aggregate candidate. This preserves conservative handling of unexplained files without pretending that a registry row proves correctness.

Operational journaling uses an explicitly recognized tool namespace outside the candidate artifact tree. The run binds the namespace policy, not a frozen copy of its own growing journal, so recording a message does not make its product inputs stale. Other repository files remain subject to the captured source and coverage rules; unknown helper folders do not become excluded merely because a process calls them logs.

The language compiler validates the model. Repository classification and artifact ownership belong to the Host's coverage stage over an acquired census; this distinction preserves the provider-free, I/O-free Core boundary.

## Apply lifecycle

Logical Apply is the whole reconciliation operation below. Guarded publication is its final step; existing low-level guarded Apply remains a useful implementation boundary.

| Stage | Work and exit condition |
| --- | --- |
| Prepare | Acquire an explicit model closure, repository snapshot, configuration, registry, ignore policy and tool identities. Compile v1 and bind the organization. Record unresolved ownership; invalid model inputs stop the run. |
| Start root | Supply the operation's instruction and bounded project overview to the ordinary root Manager. Fix the artifact-only target and coordinate read-only intake where needed. Settle the assigned files' ownership before their write tasks begin. |
| Assess and delegate | Every local Package with Definitions is considered. Managers assess local realization and issue outcome-based child tasks. Empty coordination nodes organize children. |
| Work and review | Perform local changes in an owned candidate workspace. Harvest the actual delta and run a fresh independent review. A failed review returns grounded rework; an unresolved model question pauses that task without editing the model. |
| Integrate upward | The parent composes deliveries on an exact base, records the aggregate candidate and checks interface and cross-area consistency. Its fresh integration reviewer examines the aggregate and relevant child evidence. |
| Verify aggregate | At the root candidate, run required project checks, complete file coverage and Definition-assessment census. Independently verify the result against the model and required Manager results. |
| Preflight | Recheck the captured repository, model closure, configuration, registry and ignore inputs. Source drift makes the run stale. Prior checks cannot authorize changed bytes. |
| Publish | Use the guarded writer to apply the accepted candidate and registry revision under the operation's existing authority. Do not merge, release or deploy implicitly. |
| Read back | Confirm actual resulting paths, bytes, modes and registry against the verified candidate. Record the result and readable summary. A partial write or read-back failure is not a successful Apply. |

The first default is full reconciliation of the selected local model, not a claim that all project intent outside it is covered. Local no-change results are allowed. Do not implement speculative incremental reuse first. The existing graph and registry can explain possible impact, but unknown semantic impact broadens examination or leaves a finding.

Default child writing is sequential. Later tasks use the current accepted integration base, and delivered candidates carry their exact bases. A composition conflict goes back to the accountable owner. Parents may compose validated child deltas, but their own authoring scope does not expand to all descendant files. A corrective edit to child-owned content returns as a task to that child; parent-owned integration artifacts can be changed by the parent. Whenever composition changes a previously reviewed subject or its relevant assumptions, renew the affected review and checks. The parent still reviews its complete aggregate. No last-writer-wins merge or reassignment of ownership resolves a conflict implicitly.

The [distinct review responsibilities](#distinct-review-responsibilities) apply at every level. Work review also checks delegation readiness; the root has the same integration obligation as other Managers. The final verifier is a fresh role, independent of the Managers that authored the result; it receives the complete required evidence set and must report omissions.

Three candidate review attempts is the proposed default per work or integration task. Configuration can select another finite limit, always within the global time, cost and invocation ceilings. Exhaustion returns incomplete work. A parent cannot turn an unresolved child delivery into a complete one merely by summarizing it.

## Outcomes and interrupted runs

| Outcome | Meaning |
| --- | --- |
| Applied | Required reconciliation, review, verification, guards and read-back completed for the declared scope. |
| No change | The assessed realization required no publication; its current scope and successful evidence are recorded. |
| Needs decision | A model ambiguity, incompatible requirement or unresolved ownership/coordination question needs action outside the current task. |
| Failed | An execution, validation, review or check failure prevents success. |
| Stale | Bound inputs changed; a new current run must prepare and reassess. |
| Interrupted or uncertain | The history cannot yet establish a completed lifecycle or publication outcome. |

These are semantic outcome names for the proposed protocol, not current CLI exit codes. Task-level rework and partial reports remain intermediate states. No-change is successful only after its required assessments and checks, not because no Agent happened to edit a file.

History survives interruption. Reopening a run first reconciles recorded invocations, child lifecycle and publication outcomes. Completed provider work is not automatically replayed. A new attempt binds its current inputs and is recorded as such. Unknown side effects or incomplete publication remain explicit. Recovery must retain the existing Runtime guards rather than introducing a second automatic retry loop.

## Reports and execution history

Use two views over one underlying run history: concise upward reports for coordination, and a drill-down execution record for investigation. Reports reference evidence; they do not replace it. The parent can request underlying source, candidate differences and conversations as needed.

At minimum, record run/task/attempt/Agent identities, parent and causal event links, effective role/provider configuration, model and candidate bindings, prepared context, actual rendered prompts and custom instructions, observable sent requests, returned messages and reports, context/tool reads, delegation, registry reservations, candidate deltas, review rounds, check invocations/results, integration, publication and read-back. Preserve original event content and add later corrections as new events.

The normal user report presents the result, model basis, evidence, limitations and decisions needing attention. Detailed Agent coordination remains available through the history; users should not have to manage every exchange to obtain a result.

Derive a small coordination account from these same records: local work or no-change outcomes, requests across areas, observed invocations, available usage and elapsed time. Distinguish a Manager's own effort from descendant work and waiting; do not add overlapping durations as though they were independent. Unknown usage stays unknown. This helps assess whether one Manager per Package is useful in practice without introducing scores, automatic reorganization, thresholds or another telemetry subsystem. Suggest a model or organization change only from a concrete observed problem, for the author to consider.

Every candidate transition identifies its prior and resulting tree and a retrievable difference. Records distinguish a proposed change, an Agent claim, a Host-observed change, a reviewed aggregate and an applied result. A failed or rejected candidate stays inspectable. The log supports parallel causal branches without fabricating a total execution order from timestamps alone.

Store large exact payloads as content-addressed blobs referenced by the events; identical context need not be duplicated. Recording prepared, actually sent and confirmed/unknown delivery states avoids claiming that a provider received a request solely because a template was built. Inherited history available to a continuing Agent session is part of the input record or referenced session history. Additional opaque provider behavior remains a visibility limit, not a complete transcript Markitect can invent.

Report only observed usage and known runtime identities. Exact historical inputs and outputs support inspection; they do not guarantee deterministic replay, semantic correctness or human acceptance. If observation is incomplete, the run shows that limitation. A failure to durably record required work or publication events blocks further dispatch/publication rather than returning an untraceable success.

The first inspection surface should offer a run summary, Manager task tree, event timeline and per-step access to input, report, candidate difference and evidence. A later UI can render the same records; no cockpit is required for the first Apply.

## Library walkthrough

This is a design walkthrough, not an executed provider run or a runtime verification claim. The domain vocabulary is illustrative and can be declared with ordinary Kinds and Properties.

The local organization has catalog, members, lending and lending/reminders. The model states that only registered, non-suspended members may borrow, at most five books simultaneously; the referenced book must be available. The relevant membership and catalog Definitions are explicitly connected to lending. A late-return reminder requirement belongs to lending/reminders.

Assume the existing lending implementation allows a sixth book, its test repeats that error and the usage guide says six. Their registry entries already identify lending as owner. The user requests Apply without editing the model.

| Step | Manager action and recorded result |
| --- | --- |
| Preparation | Host compiles the model, acquires the actual artifact tree and registry, and binds the profiles and operation. Model and ignore files are read-only for this run. |
| Root task | Root receives the instruction to reconcile the library. It receives a child overview and declared project context, not every source file. |
| Delegation | Root asks catalog, members and lending to assess and reconcile their areas. It describes outcomes and model references; it does not name the lending implementation file to edit. |
| Local assessment | Members confirms its registration/suspension contract; catalog confirms availability. Both return current assessed results. Lending's context contains its own Definitions and the connected member/book contracts. |
| Local realization | Lending examines its owned files and changes implementation, tests and guide to the five-book requirement. It registers any new output before writing it and records the model basis of each change. |
| Child work | Lending delegates the reminder requirement to reminders. That Manager returns a reviewed no-change result if its current realization is adequate. |
| Review and rework | A fresh reviewer checks the implementation and tests against the model. If a candidate still allows a sixth book through a second path, it returns a grounded correction. Lending revises and resubmits within its review budget. |
| Lending integration | Lending composes the accepted local and reminder results. Its integration review checks the aggregate, including interactions between lending state and reminders. |
| Root integration | Root composes the three areas and checks cross-area behavior. It can request exact membership or lending source if reports leave an interface question unresolved. |
| Final verification | Independent verification examines the aggregate and required evidence. Structural coverage checks every file and its owner or explicit classification; the Definition table shows what was assessed. |
| Publication | Guarded preflight confirms unchanged inputs; the candidate and registry are published, read back and recorded. The model remains identical. |

Illustrative expected cases for the domain assessment are a registered active member with four loans requesting an available book, a member already at five loans, an unregistered member, a suspended member and an unavailable book. These expectations come from the example model; an Agent-written test suite is independently assessed against them. A package-wide change can require additional cases, such as simultaneous borrowing, if the bound model and implementation make them relevant.

The final report says which artifacts changed, which Definitions they realize, the exact tested candidate, reviews and check outcomes, and any limitations. It can answer who supplied the erroneous test, which prompt they received, what the reviewer found and which change finally reached the checkout.

## Design checks and failure handling

The following are conceptual counterexamples used to check the proposed contracts. They are not executed tests.

| Situation | Required behavior |
| --- | --- |
| Existing artifacts drift while the model is unchanged | Apply still assesses and reconciles them. No artificial model change is needed. |
| A binding suspension rule has no needed context link | If noticed, report the modeling gap; do not silently edit the model or pretend absence of a link cancels the rule. Undiscovered gaps remain a known semantic limit, addressed later by model review and the optional Optimizer. |
| A local statement appears to require another area's behavior | Manager and reviewer raise a source-grounded boundary question through the common parent. A required unresolved assessment blocks completion. |
| Two Managers claim the same file | Reject the overlapping claim and coordinate one owner. Do not let output order select a winner. |
| A worker changes a canonical Definition during Apply | Reject that candidate as outside the artifact operation target. |
| A Manager assigns a vocabulary-only label to avoid a required feature | Independent review must reject the unsupported assessment. |
| A Kind needs no standalone artifact but imposes behavior on its uses | Assess the concrete uses and their realization; vocabulary-only does not discharge those obligations. |
| Every file is registered but the sixth loan is still allowed | Structural coverage can pass; semantic review and domain checks must examine the behavior. Registration is not correctness evidence. |
| Every file is owned but a required Definition has no justified realization | Refuse completion even though structural file coverage passes. |
| Child review passed, but parent composition breaks the shared contract | Aggregate integration review fails; repair the owning area and revalidate the new candidate. |
| A local pass is offered as integration or final evidence | Reject the substitution of review purpose; assess the exact aggregate with the required stage. |
| A parent needs a child's file to evaluate integration | Supply a recorded targeted read. There is no absolute file-blindness rule. |
| An external edit changes the checkout or model during work | Preflight reports stale and preserves candidates. Do not apply old verification to new bytes. |
| Review cannot converge or a model contradiction remains | Return incomplete/needs decision with bounded attempts and retained evidence. |
| Publication writes some files and then fails | Record actual writes and uncertainty; return partial failure, inspect and recover before further publication. |
| An Agent reports success without a recorded delta or required evidence | Host validation refuses completion. |
| A required event cannot be stored durably | Stop before further unrecorded execution or publication. |

## Relation to current source

The fixed source baseline for this assessment is Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. The north-star documentation branch starts from it. [Architecture](../../architecture.md), [project operations](../../project-operations.md) and [provider adapters](../../provider-adapters.md) describe the existing method and execution boundaries.

| Existing foundation | Reuse direction and remaining work |
| --- | --- |
| Pure structural Core and fixed snapshots | Preserve deterministic compilation and explicit inputs. The completed v1 authoring language still needs a production compiler and conformance corpus; the current fixed project Schema is not that implementation. |
| Recursive work, independent review and integration | Reuse the method contracts. Derive the organization from v1 Packages instead of requiring canonical Manager Definitions; adapt context and task transport. |
| [Existing Reconcile operation](../../../src/internal/runtime/projectrun/operation.go) | Already asks for the complete responsibility against unchanged model intent, missing realization and reasoned no-change results. The new Apply makes that reconciliation the main artifact operation; it does not invent the capability. |
| Role profiles and finite budgets | Extend the existing runtime composition for the selected team and per-Manager configuration rather than create a second execution launcher. |
| Candidate workspaces and delta validation | Reuse guarded ownership and actual-delta validation. New registry/path reservation and v1 target classification need concrete implementation. |
| Coverage, check and verification stages | Preserve whole-repository accounting and conservative unknown handling. Replace fixed-vocabulary realization inputs with the proposed registry and Definition assessments. |
| Domain records, receipts and record storage | Reuse the storage boundary while specifying the richer task/event/blob records. Existing partial telemetry is not full recording of every proposed detail. |
| Guarded publication | Preserve exact-source checks and lifecycle outcomes; add explicit final read-back of the accepted aggregate and registry in this design. |

This is a clean realization of the new language and organizational concept, not a claim that Package-based Apply already works or only needs a CLI rename. Reuse an existing component when it fits; rewrite or delete it when that yields a cleaner target. Preserve original evidence and verify the required source-binding, ownership, publication and recovery properties in the new implementation without retaining old workflow compatibility.

## First delivery sequence

The [implementation plan](manager-apply-implementation-plan.md) is the current delivery owner for this concept. Its small slices replace the earlier compatibility-oriented adapter/cutover sequence.

1. Fully describe and encode library M1, select the M2 change and review independent expectations for both.
2. Design the Go product architecture and the Playground, including what to keep, rewrite or delete. The Playground uses the real product and does not implement a second Manager system.
3. Implement complete v1 compilation, organization/context preparation and actual role-session startup with durable communication.
4. Implement local realization/review, recursive integration, registry/coverage, aggregate verification and guarded publication/read-back.
5. Use real Agents to generate A1 from M1 and inspect the code, tests, documentation and trace. Change only the model to M2, Apply to A1 and inspect A2 and its differences. Keep model, product build, profile and artifact evidence distinct.
6. Consolidate concrete negative cases and remove remaining obsolete implementation and documentation. Add further commands only after this core cycle is demonstrated.

Each step needs its own concrete tests and evidence. This sequence is an implementation proposal, not a backlog assignment, provider-run authority, release plan or acceptance claim. Only after the Apply path is demonstrated should additional commands, richer team strategies, concurrency optimizations and incremental context/impact reuse be selected.

Government remains a later, separate model-change authority layer. The same organization and history could support it, but Apply's artifact-only contract and the distinction between reviewed suggestions and authorized model changes remain intact.

Return to the [north star](manager-realization-north-star.md) or [canonical-model overview](README.md).

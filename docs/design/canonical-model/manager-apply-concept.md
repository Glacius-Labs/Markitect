# Manager organization and reliable Apply

## Current agreed concept — 19 October 2026

**Status:** owner-accepted direction, implemented on Main (PR #463 and #472) with real library runs in the validation records; owner acceptance is pending. [DEC-056](../../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace), amended by [DEC-058](../../concepts/register.md#dec-058-apply-always-executes-before-independent-verify), replaces conflicting process choices in the earlier design below. DEC-058 removes initial Verify for every Apply and starts with Execute and integration at Root. This section is the current behavioral owner; the [runtime contract](manager-apply-runtime.md) records the implemented behavior. The [library walkthrough](manager-apply-library-flow.md#current-library-walkthrough) applies the accepted direction to the existing example.

The purpose is model fidelity with less routine human supervision: permanent responsibilities, parallel independent work, different review perspectives and recursive integration. Manager count, message count and a green test suite are not success criteria by themselves. Apply reconciles artifacts with the canonical model, including drift under unchanged intent; it never changes the model to make a realization pass.

### Permanent organization and ownership

One ordinary Root Manager and one Manager per local canonical Package form the responsibility tree. Canonical Package identity, not incidental code/test subdirectories, defines a Manager. Parents follow the model hierarchy. Every Manager assesses its area; Root cannot select a smaller team and silently omit responsibilities. Managers are accountable roles, not individual provider conversations.

The default layout mirrors the model hierarchy beneath declared artifact roots such as `src`, `tests` and `docs`. The owner explicitly confirmed this default on 19 October 2026 ([source](../../concepts/sources/recursive-apply-simplification-20261019.md#default-layout-and-model-only-start)). Roots and explicit exceptions are persistent project choices, not per-Apply negotiations. Support existing framework layouts and colocated tests. Resolve the mapping before writing even when the corresponding artifact directories do not yet exist; the absence of files does not prevent deriving Managers or their areas. The deepest Manager mapped to an artifact path owns it; ancestor areas exclude deeper owners. Shared files at an artifact root belong to Root unless a persistent explicit mapping says otherwise. Reject overlapping or ambiguous ownership before work starts, including path aliases under the actual platform's path rules.

This deliberately replaces ownership-by-creation/reservation and common-ancestor negotiation for ordinary new paths. A Manager creates files in its own area and registers them itself without per-file approval. Transfers and layout changes are explicit changes to the persistent organization, not a way to evade a finding during Apply. The registry preserves exact paths, one owner, links to existing Definitions and reasons. One artifact can realize several Definitions without gaining several owners.

Keep separate registry shards per responsibility, structurally corresponding to the model, and aggregate them for complete accounting. Do not make a global writable registry the synchronization point. Registry/control files remain outside the canonical Definition roots. Physical shard names and any format migration are implementation details to settle coherently, not new language v1 semantics.

Every observed regular file is a recognized protected input, an exactly registered artifact or an explicitly explained ignore. Preflight blocks on ambiguous ownership and on control faults no Manager can repair. Repairable accounting drift of the start stand, such as an unregistered file in a Manager's area, is that Manager's first work ([DEC-062](../../concepts/register.md#dec-062-start-stand-drift-is-the-owners-first-work)). Repeat the census for every candidate, including new files. Existing `.markitect/ignore.yaml` is the current ignore input: the discussion's word `.markitectignore` does not introduce another supported format. Gitignore is not artifact accounting. Root ownership of otherwise shared paths does not silently register unexplained files.

**Trust boundary:** with a declared layout, [DEC-059](../../concepts/register.md#dec-059-enforced-responsibilities-build-start-and-an-outside-oracle) enforces writing ownership in the shared working tree: an Execute turn can write only its own areas and registry shard. Without a layout, Managers are trusted to respect writing ownership, and ordinary session/tool evidence is inspected for violations. A shared Git diff cannot reliably attribute every write to an Agent; do not claim complete violation detection or security isolation. An observed violation becomes a concrete correction/escalation, not an automatic ownership transfer. Protected model intent cannot be redefined by a Manager.

### Context and communication

Each Manager gets the same reconciliation mandate plus its identity, owned areas, direct parent and children with their roles, relevant model obligations and required cross-area contracts. A per-area base prompt or `AGENTS.md` may explain work and responsibility; it must not become a second owner of business rules. Compile applicable obligations and their provenance; do not infer missing canonical links or replace supplied obligations with Agent-selected relevance.

Communication is exclusively between direct parent and child. Give the Manager only those session contacts. Necessary knowledge of another area's interface is not permission to contact its Manager. A question outside the local area goes to the parent, which resolves it, delegates to a direct child or escalates further. Only Root asks the human. Root is not a central planner for every file or task; each parent coordinates its own children.

Prefer native session messaging as a transport where its observed capabilities fit. Do not assume either provider enforces tree routing. Keep ordinary readable questions, answers and findings; no special answer-only steps, per-hop AI relay transactions or bespoke message taxonomy. Do not add a direct sibling channel to avoid a parent bottleneck. Useful project coordination belongs in a parent's next normal work, not a mandatory separate planning ceremony.

The first strategy's parent/child contact list is an operational tool/context boundary, not a claim that an Agent cannot discover other identities. Observe and report routing violations. If provider tooling exposes unrestricted cross-session calls, the implementation must state that limitation or restrict the exposed calls; it must not silently claim enforced routing from a prompt alone.

For a prerequisite, the consumer explains the missing interface or file to its parent. The parent gets the owner to supply it in the shared tree and tells the consumer what changed. No `limited` delivery or separate prerequisite acceptance is required before development can proceed. Missing scaffolding and mutually waiting children are coordination work for their parent, not grounds to invent duplicate shared types. A genuinely unresolved policy goes upward rather than being bypassed locally.

### Parent foundations before child realization

Before a child's realization work begins, its parent first provides the concrete foundation that child needs and communicates what is available through the normal parent/child exchange. The foundation must actually exist and be usable for the child's work, not merely be promised. Root starts with the shared root-owned basis; each parent applies the same rule to its own children. The owner explicitly accepted this [start rule](../../concepts/sources/apply-execute-first-20261019.md#owner-confirms-parent-foundations-before-child-work).

Parents create or preserve the needed basis within their own responsibility and coordinate prerequisites owned elsewhere through the tree. Already suitable foundations can remain unchanged; this rule also applies to an existing realization and adds no case-1 route or separate bootstrap phase. It requires neither a new readiness schema nor per-file approval. The parent identifies the available files, contracts or other concrete prerequisites in ordinary communication.

After their parent-supplied foundations are available, independent siblings may progress in parallel. Additional dependencies between their areas still require actual producer results coordinated by the parent; the parent foundation does not make every sibling interface available. A parent may continue other own work while children work. Its final completion includes integration of its children's results and therefore follows that work: children wait for the needed foundation, not for their parent's final integrated completion.

Results are integrated upward through the same tree. Once Execute and integration are complete, capture one immutable candidate and run the independent whole-tree Verify round, with children assessed before parents. These are dependencies within the accepted Execute-first Apply, not new runtime implementation or a choice to omit required Managers.

### Existing foundations during Execute

With existing artifacts, the parent first assesses the current foundation within Execute, before its children begin realization. Compare the applicable current model, the actual files/interfaces and the known model-based needs of the direct children. Read the relevant implementation and use suitable development checks where useful; an old availability message or a green build alone does not establish that the foundation still meets the required contract. This is local reconciliation work, not an initial independent Verify round or a separate case-2 process ([owner acceptance](../../concepts/sources/apply-execute-first-20261019.md#owner-confirms-existing-foundation-assessment)).

Preserve a foundation that still fits. Correct or supplement a deficient one within the parent's ownership, or coordinate its actual owner through the tree, before dependent child work starts. State the concrete usable basis and relevant changes in ordinary parent/child communication. Unsettled intent goes upward to Root and the owner; it is not silently filled in or rewritten to match the artifacts. Existing files do not authorize skipping a required Manager, and an unchanged result needs no artificial edit.

This assessment can miss a dependency. A child that discovers an overlooked gap reports it to its parent; the responsible owner supplies the correction. If a foundation changes after dependent work has begun, the parent informs the affected children and coordinates reassessment of that work against the changed basis before integration completes. The earlier availability statement is no longer sufficient for the changed contract. No new approval record or fine-grained verification-reuse mechanism is introduced: integrate upward, capture the complete candidate and independently Verify the whole tree as usual.

### Start from the model without prepared implementation

The next library demonstration must start with the accepted model and necessary execution controls, but no application artifacts or prepared internal APIs. Runtime profiles, declared artifact roots/exceptions and bound owner decisions are setup inputs; they must not hide an application skeleton, internal signatures, shared domain types or stub implementations. Missing realization is work for Execute, not a structural accounting failure for nonexistent files.

Each Manager creates its own directories, artifacts and registry entries when needed. Root creates shared root-owned prerequisites such as the Go module. Other Managers create their own interfaces and implementations and coordinate dependencies through their parent/children. Root must not bootstrap child-owned implementations or signatures across the tree. Existing explicit callable/model contracts remain binding; technical choices left open by the model are made during ordinary realization. No additional planning phase or prerequisite approval is required.

The [library walkthrough](manager-apply-library-flow.md#first-apply-m1-to-a1) applies this boundary. Preserve evidence of who supplied prerequisites and how interface dependencies were resolved, including waits, concrete conflicts and external interventions. The earlier experiment's bootstrap supplied cross-area skeletons before parallel work, so it does not establish this start-from-model behavior.

### Shared working tree and reconciliation cycle

All Execute sessions use one shared mutable working tree. Separate Agent containers may mount that same directory; a private container does not imply a private code copy. No per-Manager branches, continual rebases, delta replay or parent workspace updates are needed. Container lifetime and provider transport remain technical details; the existing Linux/container boundary is not a requirement to recreate a workspace for every turn.

1. **Prepare.** Bind the committed model, project controls, starting repository, owner decisions and finite budget. Perform complete structural preparation and accounting before dispatch. Derive every required Manager and its mandate. Structural preparation is not an initial semantic Verify round.
2. **Execute and integrate, starting at Root.** Root begins the reconciliation work and it proceeds down the fixed parent/child hierarchy. Each Manager examines the available realization against its applicable model as part of Execute: preserve what fits, create what is missing, correct deviations and follow the existing removal rules. The mandate itself authorizes this bounded work; it needs no initial Verify finding to trigger it. Before child realization begins, the parent supplies the concrete required foundation and communicates its availability as described [above](#parent-foundations-before-child-realization). Parents coordinate their direct children and integrate results upward; they do not write child-owned files. After their needed foundations exist, independent branches may work in parallel in the shared tree. A parent's remaining own work and final child integration need not be complete before child work starts. All required Managers participate; Root cannot choose to omit a branch. In-progress checks may fail because collaborators are unfinished; report that honestly instead of weakening checks or declaring completion.
3. **Quiesce and capture.** After Execute and integration, stop further writing work and finish or terminate existing write commands, background processes and tools. A chat's 'ready' message alone is not proof of quiescence. Capture one immutable, complete candidate with the bound model/control basis and registry. If stop or capture is uncertain, retain incomplete work and do not publish. A read-only snapshot alone does not excuse a racy capture.
4. **Independent Verify.** All fresh Verify sessions and mandatory checks use that same immutable candidate. Writers stay paused for this round. Build/test outputs use separate scratch space. Independent branches verify in parallel, children before their parents; parents inspect their own obligations and their children's actual interaction and reports. Root completes the whole-project assessment in a fresh conversation too. Execute's own assessment cannot approve its result.
5. **Repair or finish.** Blocking findings go through the tree to the owners. Resume Execute and integration for corrections, capture a new candidate, and repeat the whole independent Verify round. Do not build dependency-based review reuse first. Green with complete participation, closed blocking findings and passing required checks permits guarded publication and exact read-back of the reviewed bytes. A changed model or control basis requires a new bound run, not a continuation under old intent. Exhausted budget or unresolved decisions leave a visible incomplete outcome; the loop is finite.

Every Apply follows this order, whether artifacts are absent, partly present or already conforming. Execute is required reconciliation work, not a requirement to edit files: an owner can establish that no change is needed. The subsequent independent Verify still assesses every responsibility and the required checks. If the bytes remain unchanged, Verify assesses that unchanged candidate; there is no separate no-change route, initial Verify shortcut or bootstrap phase for case 1.

The owner's [execute-first decision](../../concepts/sources/apply-execute-first-20261019.md) addresses a concrete workflow observation: before the first realization, initial Verify mostly reports absence already known to Execute. The challenged assumption is that every Apply needs a separate independent starting assessment. The design change uses Execute's local assessment for all starting states and keeps independent Verify after integration. The expected improvement is one common entry with no artificial missing-implementation findings or case-specific scheduler, while the same complete candidate still receives independent assessment. This is a design expectation, not a measured runtime result.

The deliberate tradeoff is a global pause for closing verification and a new whole-tree review round after repair. It is simpler than continuously merging private workspaces or tracking precise review invalidation. It can waste waiting/review time; measure that before adding finer scheduling or reuse.

### Fresh verification and recursive acceptance

Every Verify starts a fresh provider conversation without the author's dialogue or justification. It receives the relevant complete model, fixed owner decisions, exact candidate, check results, ordinary open findings and direct-child assessments. Fresh does not mean missing obligations or forgetting unresolved defects. Execute conversations may retain working context.

The same Manager responsibility covers realization, coordination, integration and review, with independent review sessions. A parent reviews its own applicable obligations and cross-child behavior using actual artifacts, not just green summaries. It need not duplicate every leaf test. Its Execute conversation cannot approve its own work: use a fresh Verify session, including at Root, without introducing another permanent Manager hierarchy. Child reports are evidence to inspect, not authority to waive a parent obligation.

An unchanged area still gets assessed; no separate no-change delivery transaction or expectations-case ledger is required. A Manager's result is a short statement of checked scope and stand, findings or satisfaction, required-check status and unresolved dependencies/questions. Every required Manager must be represented before Root can close.

Blocking findings require a concrete model violation, absent evidence explicitly required by the model, or a failed mandatory closing check. Under [DEC-060](../../concepts/register.md#dec-060-owners-realize-others-use-parents-settle-cross-area-matters), a Definition realized outside its owner's artifacts (rebuilt in another area instead of using the owner's realization) and a change outside a Manager's own area are such violations; the parent settles cross-area matters between its children and its Verify checks their actual use. Cite the obligation and observed behavior or source evidence. An arbitrary preferred test depth is advisory. A reproduction is valuable but an unambiguous source-level contradiction can also be concrete evidence. Do not require every conceivable case to have a separately recorded execution receipt.

Keep earlier unresolved findings until independently closed or shown inapplicable on the current candidate with a reason. A fresh reviewer overlooking one does not close it. A small finding record is enough; do not recreate the former expectations/case matrix. Reviewers may derive technical consequences from existing obligations, but cannot create new business policy.

The independent library oracle complements model review. The prior experiment's oracle was green before review found two runtime defects and missing model-required suspension coverage. Neither the oracle nor Agent agreement proves complete model fidelity. Green means the specified checks and independent assessments passed for the identified stand, within their stated limits.

### Small Orchestrator boundary

The deterministic component prepares and supplies contexts, starts sessions, exposes only allowed contacts, keeps all responsibilities visible, bounds resource use, establishes quiescence/candidates and checks completion/publication. Retain enough ordinary execution evidence to identify the inputs, participant results, findings, checks and published stand, including uncertainty. Exact transport, record/schema and stop mechanics are implementation work. Do not substitute a second AI workflow planner or retain the old task machine merely under different names.

Managers decide how to investigate and repair within their responsibility. The runtime does not prescribe per-case test plans, per-file reservations, separate reply turns, delivery purposes/scopes, rejection-counter resets or progress exceptions. A blocked parent-child relationship remains visible; budget and completion constraints cannot be waived by an Agent.

### First-demonstration scope

The owner confirmed after independent concept review on 19 October 2026: **no additional Fork, team or Skill functionality for the first end-to-end demonstration.** Assess their usefulness later against concrete observed bottlenecks, rather than adding them as anticipated optimizations. The already agreed Manager responsibilities, parallel Execute and fresh Verify remain the baseline; this adds no new product roles, helper lifecycle or acceptance requirement.

This confirmation leaves Claude's running implementation assignment unchanged. It does not prohibit existing development tools already allowed by that assignment, authorize extra experiments or require implementing the later departmental/Fork/Skill proposal. The [verbatim confirmation](../../concepts/sources/recursive-apply-simplification-20261019.md#first-demonstration-scope-after-independent-review) records the boundary.

### Implementation handoff and supersession

The later Claude session should start with this section, the [current library walkthrough](manager-apply-library-flow.md#current-library-walkthrough), [DEC-056](../../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace), the canonical self-model and the repository's normal contribution instructions. This documentation session does not authorize or perform runtime implementation, further provider runs, releases, merges of implementation work or human acceptance of generated artifacts. A separate owner dispatch will authorize that work.

The owner dispatched the DEC-056 work ([dispatch](../../concepts/sources/recursive-apply-implementation-20261019.md)); it is on Main with DEC-058's execute-first order (PR #463 and #472). [manager-apply-runtime.md](manager-apply-runtime.md) is the technical owner of the implemented behavior. This section owns the accepted target and supersession map.

| Earlier mechanism | Current direction |
| --- | --- |
| Fixed responsibility hierarchy and artifact-only Apply | Retained; no per-run ownership discovery or AI-selected participation. |
| Initial Verify before Execute, including a Verify-only route for an already conforming start | DEC-058: every Apply starts with Execute and integration at Root, proceeds through the fixed tree and then receives independent Verify. Execute can preserve all bytes; no special route for absent or unchanged artifacts. |
| Ownership by `reserve_path`, creation and nearest-common-parent decisions | Replace with persistent artifact roots, deepest-owner mapping, explicit exceptions and owner-maintained registry shards. |
| `library` owns all project-level module files because its Definition requires them | Deliberately reassign shared root files to Root under the new layout rule; keep their links to `library/callable-interface`. |
| Sequential per-turn private workspaces and accepted-delta composition | Parallel Execute on one shared working tree; one immutable closing candidate per round. |
| `local`/`integration`, `limited`/`complete`, no-change acceptance transactions | Ordinary work/results and recursive fresh Verify; keep every responsibility assessed. |
| Expectations before work, recorded case matrices and per-case execution receipts | One fresh model-based Verify with concrete findings and actual required checks. |
| Separate mandatory reviewers for each delivery plus parent integration reviewers | Fresh Verify sessions for each responsibility plus upward integration review; no second permanent hierarchy. |
| Mandatory named reviewer roster and dispute-only arbiter | Not required for this simplified first strategy; extra perspectives are optional, not an additional acceptance bureaucracy. |
| Rejection limit, restart/progress exceptions, completion-reserve admissions by delivery purpose | Finite overall execution budget and visible incomplete outcomes; no special rejection counter for this strategy. |
| Mandatory passing whole-project checks before each partial submission | Required checks on the closing candidate; useful development checks remain available without a submission ceremony. |
| Hop-by-hop protocol and special reply steps | Direct parent/child contacts using ordinary session communication; only Root escalates to the human. |

This amends the conflicting choices in DEC-036, DEC-039 to DEC-041, DEC-043 and DEC-045 to DEC-055; it retains their useful intent where compatible, including independent risk derivation and run-wide owner decisions. The owner-approved concept discussion is an exception to the old freeze. The older detailed [core](../../history/apply-design-before-dec-056/manager-apply-core.md), [execution](../../history/apply-design-before-dec-056/manager-apply-execution.md), [configuration](../../history/apply-design-before-dec-056/manager-apply-configuration.md), [ports](../../history/apply-design-before-dec-056/manager-apply-port-contracts.md), [journal](../../history/apply-design-before-dec-056/manager-apply-journal.md), architecture and acceptance documents must be revised consistently during implementation. Their old tool/state schemas are not constraints to preserve for compatibility.

Unsettled engineering details are deliberately limited: concrete evidence of writer quiescence and safe snapshot capture; persistent mapping/shard schema including path aliases; transport/address restriction using observed provider capabilities; and minimal result/finding/check records. Choose the smallest mechanism satisfying this behavior, then demonstrate it in complete workflows. Do not turn these details into more permanent roles, compulsory planning phases or case-specific decisions.

### Evidence and independent review

The preceding isolated experiment used Luna for 15 Agent jobs, completed M1→A1 and loan-limit-five-to-three M2→A2, and reported about 13 minutes plus 5–6 minutes respectively, about 20 minutes from experiment creation. It used prepared ownership/APIs, separate writer copies and external orchestration. Its final model reviews, Go checks and independent oracle passed after three review findings were fixed. Actual billed money is unknown. These timings exclude the full surrounding design/harness work and do not prove the shared-tree strategy, native communication, write attribution, live migration or scalability.

The experiment's export is `luna-parallel-experiment/results.json` alongside the associated chat artifact, not a portable repository fixture. Preserve that evidence at its original scope; do not label this documentation or the new target as an executed Apply. Independent design review findings and their disposition are recorded below after review of this exact documentation change.

Two independent Luna reviewers assessed the written concept: one for unnecessary bureaucracy, one for concurrency and evidence risks. Their first-pass concerns led to explicit preservation of no-change assessment, an explicit Root/module ownership change, a correct library topology, observable quiescence rather than a chat acknowledgement, retained open findings and an honest trust/attribution boundary. Their document review also requested a source for the Luna runtime preference, unambiguous supersession in the delivery plan/vision and separation of publication from human acceptance. Those corrections were incorporated. No reviewer proposed another role, case ledger or state machine as necessary. Exact stop/capture implementation, path aliases, actual message-tool limits and shared-tree scalability remain unproved engineering boundaries; design review is not runtime acceptance.

## Earlier design reference

The following text preserves the previous detailed design and its link anchors for existing sources. **It predates DEC-056. It is not an alternative implementation contract.** Where it conflicts with the current agreed concept above, implement the current concept and update the affected technical owners coherently. Historical source quotations and completed evidence retain their original scope.

This is the product concept for realizing a canonical model through Managers. It owns organization, context, delegation, deliveries, review purposes, artifact ownership, the Apply lifecycle and reporting. The [design map](README.md#apply-design) lists the technical contracts that refine it and states the status of the whole set. The [north star](manager-realization-north-star.md) keeps the ideas and variants that this concept did not select.

The concept uses the [completed language v1](minimal-language/language-contract.md) without new Core Kinds or semantics. The owner decisions include [DEC-025 to DEC-036](../../concepts/register.md#dec-025-package-managers-organize-realization) and the superseding [DEC-039 process clarification](../../concepts/register.md#dec-039-deterministic-manager-process).

The product is an organization that realizes an explicitly described project world. The canonical model supplies intended meaning and constraints. A deterministic Orchestrator organizes work, supplies context, validates deliveries and records the process. AI Managers make bounded realization judgments and integrate results. The Orchestrator propagates the operation and enforces the process; an Agent does not choose who participates or rewrite another Manager's operation. The first operation is Apply: reconcile the existing artifacts with the current model, including when the model has not changed.

The owner accepted [named review teams, conflict arbitration and product defaults](../../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) on 17 October 2026. This bounded extension replaces the single-reviewer-only target; source implementation remains separately versioned. The [configuration owner](../../history/apply-design-before-dec-056/manager-apply-configuration.md) holds the team syntax/defaults; the [execution owner](../../history/apply-design-before-dec-056/manager-apply-execution.md#review-conflicts-and-arbitration) holds dispute and rejection semantics.

## Selected choices

| Question | Choice | Reason |
| --- | --- | --- |
| Organization | One ordinary root Manager and one Manager per local Package. | Responsibility follows the structure the author already maintains. |
| Code, tests and documentation | One hierarchy responsible for all artifact forms. | A feature's implementation, tests and documentation stay coordinated. |
| Default team | The Manager does its local work; the configured named independent reviewers assess each delivery and integration. A fresh reviewArbiter settles explicit disputes. | Project defaults and exact Manager exceptions preserve independent review without another Manager hierarchy. |
| Final check | The Orchestrator's deterministic evidence check after the root's integration review; no separate verifier Agent. | The root's integration review already judges the whole; the Orchestrator makes sure all evidence is current. |
| Parent knowledge | Summaries first; targeted, recorded source inspection when needed. | Local responsibility without depending entirely on child claims. |
| Context | Own Definitions and contracts, explicit neighbors, Definitions whose modeled scope covers the Package, and task facts. | Simple, inspectable compilation without semantic retrieval. |
| Order | The Orchestrator propagates the fixed operation down every branch, then runs ready work, dependency responses and upward integration under a fixed protocol. | No Agent can omit a responsibility or bypass its review. |
| Acceptance | Apply starts from the committed model and writes its verified result into the working tree; the owner's commit accepts it. | Acceptance stays with the owner, as for the model. |
| First reconciliation | Every local Manager receives the operation and assesses its area; no-change results receive review too. | Participation never depends on root relevance judgment or incremental invalidation. |
| Execution | One step at a time ([S5](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)), in owned candidate workspaces. | Reliable integration before concurrency. |
| File ownership | One accountable Manager per artifact; many-to-many links between artifacts and Definitions. | Write responsibility is separate from what an output realizes. |
| Prompts and history | Record the actual inputs and observed events; derive readable reports from them. | Work is inspectable without sending every transcript to every Agent. |
| Other commands | Later operations over the same organization. | The first delivery concentrates on Apply. |

## Four kinds of input and record

| Concern | Owner and meaning |
| --- | --- |
| Canonical model | Authored Definitions, contracts, bindings and relationships: what should hold. |
| Organization configuration | Project-owned execution choices: teams, profiles, limits, instructions and context presentation. It cannot redefine canonical meaning ([configuration](../../history/apply-design-before-dec-056/manager-apply-configuration.md)). |
| Realization registry | Tool-maintained, versioned metadata on artifact ownership and justified links to Definitions. It describes the actual realization, not a second intended world. |
| Execution history | Immutable records of inputs, communication, candidates, reviews and outcomes ([journal](../../history/apply-design-before-dec-056/manager-apply-journal.md)). |

Binding intent lives only in the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)). The test is simple: if violating something makes an artifact wrong, it is model content. That includes realization constraints such as a public interface, the programming language or required build and test commands; the library's [callable interface](library-example/m1/model/library/callable-interface.yaml) is an example. Configuration and instructions hold only execution choices and operational guidance, such as profiles, limits, reporting style or tool usage. A reviewer who finds a project rule in configuration or custom instructions records a finding, and the rule moves into the model.

Configuration and registry stay outside the canonical Definition roots; v1 would otherwise require a Definition envelope. The compiler produces a bound structural model and diagnostics and does not interpret prose as rules. The Orchestrator applies deterministic workflow rules to explicit snapshots. Agents interpret meaning and produce reviewable work. No prompt or report can change the operation target or authorize a model edit during Apply.

## Authoring and inspectable preparation

A small optional software vocabulary and the complete [library example](library-example/README.md) are ordinary, versioned models. Derive the vocabulary from the example's needs instead of building a universal catalogue. It has no privileged compiler behavior; authors can use their own terms, replace it or extend it. Templates produce visible, editable Definitions in the uniform YAML form, not a second authoring language.

Before Agents start, the author can preview what each Manager will receive: which statements, contracts, responsibilities and explicit connections, and why each is included. Original model content, deterministic explanations and later Agent judgments stay distinguishable. The preview exposes mistaken context and missing connections; it cannot establish semantic completeness. Preparation is part of every Apply; there is no requirement to precompile contexts for every possible task.

Add structure only when it exposes a concrete failure possibility or saves recurring human work. Prefer clearer instructions and views over new roles or parallel records.

## Organization and configuration

The root Manager follows the same protocol as every other Manager. It has its own runtime identity for the whole selected local model; v1 forbids a root Package, so none is invented. Top-level Package Managers are its children, and parents follow folder ancestry. Every Manager receives the operation, including an empty leaf or an intermediate Package with one child; there is no pass-through startup exception in this strategy. An empty local Definition set is an explicit assessment result, not permission to skip that Manager or its existing ownership and child responsibilities. Insufficient capacity leaves the run incomplete rather than silently omitting a Manager. The run binds the derived topology as execution metadata; empty folders stay out of the canonical model digest as v1 specifies.

Imported models supply read-only vocabulary and context and create no Managers. Editing a library is a task in that library's own project. One local model is the first supported organization.

Configuration has project defaults plus exact Manager overrides and separate hard ceilings. An override for lending does not configure lending/reminders. The Orchestrator enforces model obligations, resolved configuration and fixed process constraints; parent hints cannot add task restrictions or narrow a child's responsibility. The [configuration contract](../../history/apply-design-before-dec-056/manager-apply-configuration.md#resolve-settings-deterministically) owns resolution. Configuration cannot disable required delivery or integration review, the final evidence check, coverage or source-drift guards.

| Configurable part | First design |
| --- | --- |
| Team | Manager plus at least one named independent reviewer, and a configured dispute-only reviewArbiter. The earlier single-reviewer source is not multi-reviewer support. Implementer roles, reviewer per Definition and separate artifact hierarchies remain later strategies. |
| Execution profiles | Project role model/effort and instructions with exact Manager overrides; explicit compatible Orchestrator inputs supply provider/runtime/resources and credential references. |
| Scope of shared rules | The model states a rule's scope through a Property whose values are local Package paths, for example `appliesTo: [lending]`, or `[.]` for the whole model. Configuration only names which Kind and Property carry scope; it never lists individual Definitions. |
| Owner approval | Optional list of Packages whose changes wait for the owner's approval before publication. |
| Review batching | Optional: one reviewer session may assess several no-change deliveries, each with its own bound verdict. |
| Artifact creation | Permitted creation areas and exclusive reservation of new paths. Artifact paths need not mirror Package paths. |
| Limits | One shared assignment ledger and a finite rejection allowance, owned by the [execution contract](../../history/apply-design-before-dec-056/manager-apply-execution.md). |
| Concurrency | One step at a time ([S5](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)). Later parallelism must use the same reservation and integration rules. |

### Project-level artifacts and the skeleton

Ownership follows creation: the Manager that reserves a new path owns it. For a project-level artifact, the requiring Definition decides who creates it. In the library, `library/callable-interface` requires the Go module, the entry point and the build and test commands, so the `library` Manager creates and owns them. When Definitions in several Packages require the same artifact, their nearest common parent decides the owner and records the decision; that can be the root. A file that no Definition justifies is never owned: it is a Markitect input or explicitly ignored, so CI configuration, for example, needs a Definition before Apply manages it.

When a required skeleton is missing, its owner can supply a limited normal delivery while dependent work waits: module, entry point and the conventions the model requires. Its exact usable scope and outstanding work remain explicit; Orchestrator acceptance of this prerequisite does not complete that Manager or subtree. The consumer waits until the delivery is reviewed and accepted; its next step then sees it in the aggregate ([D3](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). The same owner later integrates the completed operations into its entry point. These use ordinary owned work and review; the root does not invent a global delivery plan, and the Orchestrator does not infer an execution order from arbitrary references.

## Tasks and delegation

Apply is one static, versioned operation instruction: reconcile the artifacts of your responsibility with the bound model, assess existing realization, correct deviations and explain the result. It is combined with each Manager's separately compiled context, resolved configuration, role template and current Orchestrator facts. The instruction enters through the root and reaches every local Manager unchanged. Questions, findings and delivery messages carry current work facts; they cannot replace, narrow or rewrite that instruction. The model remains read-only.

The Orchestrator records every parent-child handoff and guarantees recursive propagation to all direct children. It does not depend on an Agent calling a delegation tool, returning a participant list or remembering a child. Logical receipt and physical session dispatch stay distinct: a stored prompt is not proof the Agent received it. Each Manager acts only in steps the Orchestrator admits ([T4](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)), and none of its containers runs between steps ([S1](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)). Admission order cannot skip a Manager.

The organization metaphor has a precise boundary: Managers own local realization and the integration of their direct children's results; the Orchestrator owns mandatory participation, permitted routes, required reviews and completion. The root is an ordinary integration Manager, not an AI workflow planner. A parent retains its own artifacts and can resolve local prerequisites before its children finish.

| Fixed process responsibility | Agent judgment |
| --- | --- |
| Compiler pipeline derives hierarchy, explicit context and inclusion reasons. | Interpret the supplied obligations in the local realization. |
| Orchestrator propagates the same operation to every local Manager. | Return a justified change or no-change result. |
| Orchestrator admits ready work one step at a time and enforces finite budgets. | Report concrete prerequisites, questions and corrective work. |
| Orchestrator routes every message along tree edges and records each hop. | Explain a question or proposed agreement with its model basis. |
| Orchestrator requires local and integration review and current evidence. | Independent reviewer judges whether realization and evidence suffice. |

### Fixed Manager cycle

1. **Receive and propagate.** The Orchestrator binds the operation and per-Manager context, records receipt and propagates the unchanged instruction to every direct child. Each Manager remains in the completion census. A failed or uncertain delivery cannot count as completed participation.
2. **Work and review.** Every task starts ready: there is no Explore phase, no write barrier and no global Plan barrier or root-authored delivery plan ([DEC-052](../../concepts/register.md#dec-052-implement-then-review-no-explore-phase)). Each Manager decides its own realization work within the unchanged mandate, and unresolved dependencies hold only the work that needs them. A reviewer records the cases for a task's work before its step ([DEC-049](../../concepts/register.md#dec-049-review-cases-are-recorded-before-the-work-they-assess)). Each work step ends with `submit` or `wait` ([T5](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)); a Manager that finds nothing to change submits a no-change result. The active owner writes only its allowed artifacts. The Orchestrator observes and seals the candidate. Independent local review judges changes or a no-change proposal against the recorded cases with the required checks. A normal delivery may have a limited usable scope while its Manager remains open: independent review and Orchestrator acceptance must precede dependent implementation, whose next step then sees the delivery in the aggregate ([D3](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). Review is required during handoffs, not only at the end.
3. **Integrate upward.** Once every direct child is done and local work is accepted, the parent's integration step receives the composed aggregate of its subtree with summaries and evidence and judges their interaction ([T9](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)). Parent-owned integration changes receive ordinary review. A fresh integration review judges the aggregate; one-child parents still participate.
4. **Correct through the tree.** A finding is an issue addressed to its responsible owner ([I1](../../history/apply-design-before-dec-056/manager-apply-core.md#issues), [D5](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). Dependent work waits for the required reviewed result. Changed content or assumptions require renewed affected assessment; old acceptance does not carry to a different subject. Unknown affected scope broadens inquiry and review, never narrows participation.
5. **Complete and publish.** A Manager's 'done' is a proposal. The Orchestrator accepts a subtree only after its local and descendant obligations, current required reviews/checks and blocking questions are complete. At the root, final evidence, source preflight, any required owner approval, guarded publication and read-back remain mandatory. All Agents agreeing does not establish success by itself.

These are steps within the existing run lifecycle, tracked by the [task machine](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks), not another run state machine. In the first Apply, steps run one at a time ([S5](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)). Later discoveries reopen affected planning or assessment without stopping independent branches.

### Ready work and parent guidance

The Orchestrator records when work becomes ready and admits it in the order [T4](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks) fixes. Eligibility includes current inputs, resolved prerequisites and budget. A blocked task does not prevent other ready tasks from running, and becoming ready again records a new queue entry without erasing its earlier wait. This is deterministic for the same recorded event sequence; different provider response times may produce different sequences.

A dependency wait is a `wait` outcome that names the missing accepted result or decision ([T5](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)). Because no container runs between steps, a waiting task holds no write access; artifact ownership remains unchanged, and uncertain writer effects are reconciled before the workspace admits another step ([S2](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)). Mutual waits go to the common parent for bounded clarification ([T12](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)), not repeated polling or guessed ordering. A necessary missing model relationship still requires a separate model correction.

Parent hints are separate, attributed messages alongside the unchanged operation, compiled context and current Orchestrator facts. They may point out an integration concern or suggest an implementation approach within the model's freedom. They cannot create binding project rules, narrow another Manager's responsibility or waive a required check. Parents work from summaries of results, interfaces, prerequisites, findings, uncertainty and evidence; they request targeted detail when needed rather than reading every descendant transcript.

The owner confirmed a bounded, versioned direct-child report ([confirmation](../../concepts/sources/manager-process-clarification-20261017.md#bounded-coordination-and-idle-execution)). Its [message contract](../../history/apply-design-before-dec-056/manager-apply-port-contracts.md#bounded-child-reports) distinguishes limited delivery and completed responsibility, binds the exact step or delivery, and preserves prerequisites, findings, uncertainty and evidence references. A per-child limit alone is insufficient: the assembled parent input also stays within its context limit. Visible partitioning and targeted retrieval retain every required child's status and outstanding obligations; omitted detail is never agreement or completion. Summaries supply Agent judgments, not Orchestrator acceptance.

### Communication and dependencies

Messages travel only along tree edges, and the Orchestrator records every hop ([M1](../../history/apply-design-before-dec-056/manager-apply-core.md#messages)). A question to a known Manager travels up to the common parent and down the appropriate branch; the Orchestrator forwards it without dispatching the Managers in between ([M2](../../history/apply-design-before-dec-056/manager-apply-core.md#messages)). A model link can add compiled context but never grants a communication shortcut or write ownership.

Name the addressee from explicit responsibility and model facts when possible. When the target is unknown, ask the parent, which replies or names a better addressee in its subtree ([M3](../../history/apply-design-before-dec-056/manager-apply-core.md#messages)). The Orchestrator records the question, its hops and outstanding replies, so silence is not agreement. Inquiry seeks responsibility or an implementation fact, not an inferred missing model relationship. Apply cannot add a missing link or compensate by inventing context; a necessary missing relationship requires a separate model correction and blocks the affected assessment.

A reference is not automatically an execution dependency. A concrete dependency states the needed reviewed result and its basis. Its owner may work while dependents wait, including a parent supplying a skeleton. Managers retain local API ownership; consumers request changes through the tree and never edit producer files. Unresolvable ownership, circular waits or conflicting requirements remain visible decisions or incomplete work, not an endless exchange. Inquiry and correction consume the existing finite budget and cannot reset review counters.

### Remaining process detail and technical boundary

The owner confirmed local planning, recorded readiness, separate parent hints and limited normal deliveries on 17 October 2026 ([source](../../concepts/sources/manager-process-clarification-20261017.md#explore-planning-and-limited-deliveries)); the Explore part of that confirmation was replaced on 18 October 2026 ([DEC-052](../../concepts/register.md#dec-052-implement-then-review-no-explore-phase)). These choices address late discovery of shared prerequisites without returning participant selection or workflow design to AI. The library trace must show an accepted skeleton unblocking a consumer while its producer stays open, and independent work progressing around waits.

Review reuse requires an unchanged complete bound subject, including its relevant inputs and assumptions. An Agent's claim that a changed file is unrelated is insufficient. Changed bindings require renewed affected assessment; unknown effects widen it. Whether a local review survives composition onto a newer aggregate is decided by [D4](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition). An altered aggregate needs current integration evidence even if a local subject is unchanged. Fine-grained semantic reuse is not required for the first Apply.

The owner also selected a review rejection limit: successful deliveries consume shared execution capacity and restart the rejection count for their purpose ([DEC-050](../../concepts/register.md#dec-050-an-accepted-delivery-restarts-the-rejection-count)). Rejections accumulate per run, Manager and review purpose until the next accepted delivery, and a rejection that finds fewer gaps than the last one does not count ([DEC-054](../../concepts/register.md#dec-054-a-rejection-that-finds-fewer-gaps-does-not-count)); after two, the parent assesses the repair plan, and the default third counted rejection leaves the task incomplete, which finishes the run Incomplete ([T14](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)). This replaces the earlier submission-count design, which could exhaust its limit on successful handoffs alone. The [execution contract](../../history/apply-design-before-dec-056/manager-apply-execution.md#admission-amendments-and-finite-correction) owns exact counting and exhaustion. Before recursive implementation, define and validate the technical message, readiness, delivery and counter bindings against this process. No second planner, reviewer hierarchy, semantic index or parallel workflow ledger is needed.

The compiler pipeline here means deterministic preparation over compiled model facts. The existing `model`/`organization` separation can provide it; no new Core Kind or model-language change is implied. Managers and reviewers still interpret prose. The compiler can reject unresolved references but cannot prove that every necessary semantic relationship was authored.

### Working bases and conversation continuity

The owner selected continuity across limited deliveries rather than a fresh conversation for each changed working basis ([confirmation](../../concepts/sources/manager-process-clarification-20261017.md#working-bases-and-session-continuity)). The earlier contract fixed a candidate to the whole execution session; it would unnecessarily discard context when a Manager needs a new accepted prerequisite. The first Apply realizes this choice in a step-bound form: each step runs in its own container ([S1](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)) and continues the Manager's conversation or receives an Orchestrator-built handoff ([S3](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)), and nothing changes a workspace inside a running container ([S6](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)). The target separates four lifetimes:

| Subject | Lifetime and authority |
| --- | --- |
| Run inputs | Immutable captured model, configuration and original repository inputs. A model or configuration change still requires the existing stop/continuation process. |
| Manager working basis | The accepted aggregate current at the step's admission ([D3](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). Every later step sees each newly accepted delivery. |
| Own working changes | The Manager's unaccepted changes, recorded after each writing step ([S4](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)) and replayed onto the current aggregate by the D2 rule. They are neither an accepted dependency nor an approved result. |
| Review candidate | An immutable snapshot and its exact review purpose, inputs and evidence. Later working changes cannot alter that subject or inherit its approval. |

A stable Manager identity and conversation span many steps. Each step records the aggregate version it saw, and its file edits are observed outputs. Changed assumptions reopen affected assessment; unknown influence broadens it.

The next message identifies the current aggregate version, accepted input changes, preserved own changes, open findings and relevant reassessment. The Manager checks changed interfaces and assumptions from current sources. Old conversational claims are context, not current evidence. New dialogue, a container replacement or a context handoff cannot reset responsibility, findings, budgets or rejection counters.

Continuation stays within the existing session-step and context limits. Use bounded summaries and retrievable evidence rather than copying every transcript into each step; never silently omit mandatory current obligations or findings to fit. An oversized required context remains a visible refusal. Provider-internal compaction or unavailable history is an observation limit, not a claim that all earlier context was preserved.

Demonstrate the library case: lending keeps its own unfinished work when a catalog delivery is accepted, sees that delivery in its next step's aggregate and reassesses the affected interface; a crash during materialization cannot cause an unrecorded continuation or a repeated effect ([S2](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)).

The owner also selected bounded idle execution, not one permanently running container per Manager. A logical Manager remains in the census while its conversation and task evidence are retained without a running environment. In the first Apply the warm limit is zero, so every step is a charged session start ([S6](../../history/apply-design-before-dec-056/manager-apply-core.md#steps)); the [idle policy](../../history/apply-design-before-dec-056/manager-apply-execution.md#idle-execution-and-resumption) keeps its accounting.

These refinements address the owner's scaling concern without replacing the first Apply's complete participation. An observable library case delivers a limited catalog report without declaring catalog complete, exposes all pending lending prerequisites even when reports are partitioned, and resumes lending in a new container with the same responsibility and counters. Reuse of earlier Manager assessments and coordination-based modeling suggestions remain [later extensions](manager-realization-north-star.md#later-operations-and-extensions), not exceptions to complete participation or required evidence.

### Fixed-source comparison and improvement criterion

Comparison basis: Main [`0392cd2f`](https://github.com/Glacius-Labs/Markitect/tree/0392cd2fd724d98b3fc715cb5fd3db22e2d5d224), 17 October 2026. This is an assessment snapshot, not a live backlog. The owner observed that AI-created child prompts and root planning reintroduced discretion over already explicit structure ([source](../../concepts/sources/manager-process-clarification-20261017.md)).

| Surface at that source | Assessment and required follow-through |
| --- | --- |
| `internal/organization/select_context.go`, `topology.go` | Explicit selection and direct parent/child inventory fit. The `PassThrough` startup assumption and empty-leaf omission must not become scheduling policy. |
| `internal/organization/render_templates.go`, `render_context.go` | Bounded deterministic rendering fits; prompts to 'delegate outcomes' and arbitrary supplied task outcomes must become the static operation plus separately identified current messages. |
| `internal/run/binding.go` review-floor calculation | It excludes empty local work and pass-through integration. Reassess the reserve against mandatory participation and the corrected required reviews. |
| `internal/run/session*.go`, `state.go`, `restore*.go` | Durable sessions, sends, budgets and uncertainty are useful foundations. They do not yet implement the Manager message protocol or recursive completion. Preserve their effect observations. |
| PR329, despite its branch name `dev/run-local-review-ledger` | Its integrated change records complete versioned basis inventories. It is not a local-review or delegation engine. |
| `internal/apply/control*.go`, snapshot/realization and container/agentd foundations | Control, exact candidates, ownership and physical isolation remain applicable. This decision requests no runtime rewrite. |
| Earlier design `plan` / `delegate` tools and parent-created prompts | Superseded by this process. The [core protocol](../../history/apply-design-before-dec-056/manager-apply-core.md#host-tools) defines the Orchestrator tools; old illustrative APIs are not implementation authority. |

Observable improvement: a controlled trace delivers the identical operation instruction to every Manager with separately compiled context even when the root omits a child in its output; forwards cross-area questions only along tree edges, recording every hop; records reviewed no-change results; services a parent prerequisite without deadlock; and refuses subtree success while a child, review, dependency response or changed-subject assessment remains open. Real provider execution and the two real Applies remain separate evidence.

## Context packages

Each context package is an immutable rendered input plus source references. Its bytes and the actual outgoing invocation are recorded. Context compilation is deterministic for its explicit inputs; later semantic conclusions are attributed Agent judgments. No Agent selects initial relevance or repairs missing model relationships by adding inferred context.

The initial Manager package contains:

1. Its task and role, the mandate (responsibility, binding basis, freedom, artifact authority, reserved choices and limits, as the [configuration](../../history/apply-design-before-dec-056/manager-apply-configuration.md#the-mandate-section) renders it), effective configuration, operation target and review requirements.
2. Every local Definition in its Package, in stable path order: the filename as heading, then identity, Kind, complete body and structured `spec`, plus the bound Kind and Property contracts.
3. A See also section: direct outgoing references and body links, and direct incoming references to the Package, each with reason and direction. It includes the directly connected Definitions and their type contracts; further references stay available to inspect.
4. Every Definition whose modeled scope covers this Package, because its scope lists the Package, one of its ancestors or `.`, as a read-only view of the bound Definition with its reason for inclusion. It does not extend import resolution.
5. A compact local Package directory for routing questions, and a child index with identities, purposes, task states and latest validated reports. It does not load foreign Definitions, artifacts or transcripts.
6. The Manager's artifact inventory and relevant candidate files for local work. A Manager that integrates children first receives their delivery summaries, agreements and evidence references.

A rendered package larger than the configured limit stops preparation with a finding to split the Package; a package is never truncated silently. The role prompts for root, Manager and reviewer are product-owned, versioned templates. Their identity is recorded with every invocation, changes to them go through review, and custom instructions are only appended.

An implementer receives the local task and candidate inputs. A reviewer works in fresh context without the implementer's conversation: its first step receives the task and the relevant model contracts only and records the expected behavior and discriminating cases; the second step supplies the actual candidate and evidence. The Orchestrator checks that a reviewer's rendered package contains what its role requires; the reviewer judges whether that context suffices.

Managers can request further Definitions, reports, transcripts or artifact files by identity; the request and the returned content are recorded. A coordinating parent is not loaded with descendant source files initially but can inspect them to resolve a conflict or judge integration. Need to know keeps Agents focused; it is not an operating-system isolation boundary.

Normal context discovery follows explicit references. A body link marks a connection, not proof that every obligation at its target applies. A missing necessary connection is a modeling defect to surface, not permission to ignore a known obligation. Apply never invents canonical links. A necessary missing link blocks the affected assessment rather than being compensated with inferred Apply context. The optional Optimizer belongs to a separate model operation ([north star](manager-realization-north-star.md#missing-links-and-the-optimizer)).

## Checking effects across Package boundaries

During local assessment, the Manager and its reviewer both ask: which statements in this area require behavior from another area, and is that connection visible in the model? For example, members states that suspended members cannot borrow, while borrowing belongs to lending. Reading the local statement is enough to raise the question; no semantic search is required.

Raise an ordinary open question naming the source Definition and passage, the possibly affected Package and the suspected effect, and address it to that Package's Manager; the Orchestrator forwards it through the nearest common parent ([M2](../../history/apply-design-before-dec-056/manager-apply-core.md#messages)). If no target is known, address it to the parent without inventing one ([M3](../../history/apply-design-before-dec-056/manager-apply-core.md#messages)). Apply can report the gap but cannot add the link; a model correction is a separate model operation followed by a new Apply. An unresolved question that prevents a sound assessment blocks completion of the affected work and therefore of the aggregate.

This is a review duty over the statements actually assessed, not a guarantee that every missing connection is found.

## Technical risk derivation

Managers and independent reviewers must derive relevant technical failure modes from the applicable model obligations, available operating conditions and actual realization. The owner need not enumerate every implementation risk or author a separate Definition for every technical consequence of an existing obligation. Managers perform this investigation in every work step, including implementation, rework and integration; each reviewer independently examines the local or composed subject. A no-change result has the same duty.

Ask how the realization could violate an obligation under the conditions in which it is intended to operate. Relevant concerns can include concurrent updates, the interval between checking a precondition and committing, partial failure, retries, lifecycle transitions and external effects. These are examples to select against the actual responsibility, not an exhaustive checklist or an instruction to invent operating conditions. For example, when concurrent use is in scope, an obligation that publication freezes content requires examining a write that read editable state before another operation published it. A green sequential test alone does not settle that concern.

Deriving a necessary technical consequence of existing intent is realization judgment within the mandate. Managers investigate, choose a conforming implementation and repair deviations; reviewers judge the reasoning and evidence. A different business policy, an exceptional tradeoff outside the mandate or a necessary missing model relationship still follows the existing question and separate model-change route. Technical inference does not authorize invented canonical links, new business restrictions, unsupported runtime assumptions or additional execution rights.

The two review stages remain unchanged. Before candidate access, reviewers derive expectations and distinguishing cases from the task and model. After candidate access, they must inspect the actual implementation and integration for additional risks and reassess whether the original cases suffice. The first-stage expectations remain recorded; additional concerns and needed cases use the existing findings and evidence path in the [execution contract](../../history/apply-design-before-dec-056/manager-apply-execution.md#evidence-need-survives-budget-pressure). Managers retain the grounds, investigation and unresolved concerns in their existing reports and Definition assessments. No separate risk register, role, workflow stage or schema is introduced.

The [owner clarification of 17 October 2026](../../concepts/register.md#dec-045-managers-and-reviewers-derive-technical-risks) addresses the gap between handling a recognized concern and actively discovering it. The observable improvement is a Manager or reviewer deriving a consequential implementation risk and obtaining a distinguishing check without owner coaching or an enumerated defect in the model. This remains a capability to demonstrate; the sequential, process-local library example does not establish concurrency or database-risk discovery.

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

The Orchestrator validates shape, scope, candidate bytes and lifecycle. A report cannot substitute claimed changes for the observed workspace delta.

Definition assessments are a small table, not a second semantic model. Each record holds the Definition, a disposition (`realized`, `constrains`, `vocabulary` or `unresolved`), the artifacts, the evidence and a reason. `realized` names the artifacts that realize the Definition; `constrains` names the other realization it constrains; `vocabulary` applies to declarations that need no standalone artifact. A positive disposition explains the observable realization or concrete use and why the evidence suffices; "done" or a file list is not enough. For behavior, the evidence names the tests or checks that would fail if the Definition were violated, and the reviewer judges whether they really discriminate. The Orchestrator checks the census, statuses, bindings and evidence references; the reviewer judges their substance. Every local Definition needs a disposition other than `unresolved`; an unresolved one prevents completion even when every file is registered.

The `vocabulary` disposition means a declaration needs no standalone artifact. It does not remove obligations its body or contracts impose on concrete uses; cite those uses and their assessments. The table shows what was examined; it does not prove that every natural-language obligation was understood. All artifact forms, including tests, are reviewed against their claimed model basis.

An unchanged Package can deliver a reviewed no-change result that still explains its realization and evidence. An old successful report is evidence for its old inputs, not current approval.

## Distinct review responsibilities

Each active named reviewer performs the required review responsibilities independently against the same frozen subject, including expectations before candidate evidence. Names and operational instructions describe perspectives, not permission to omit applicable obligations. Collect the complete roster; a generic individual pass cannot close another reviewer's finding. The [group and arbiter rules](../../history/apply-design-before-dec-056/manager-apply-execution.md#review-conflicts-and-arbitration) determine acceptance. All review conversations and arbitration start fresh; neither a repeated provider conversation nor the implementer's transcript supplies independence.

| Review | Question it must answer |
| --- | --- |
| Local delivery | Do this area's code, tests and documentation fulfill its Definitions and relevant connected obligations? Are effects on other areas accounted for? |
| Integration | Does the composed candidate satisfy the connected behavior and contracts across areas, such as a suspension preventing a new loan? At the root it also asks whether all areas follow the model's project-wide conventions uniformly. |
| Final evidence check | Deterministic, by the Orchestrator: does the exact aggregate have every required local and integration review bound to its current subject, passing required checks, conforming coverage and a complete Definition census, with no open blocking finding? |

The root's integration review is the semantic review of the whole system; no separate verifier Agent repeats it. Every local or integration review assesses its candidate against expectations recorded from the bound model without seeing any candidate, normally before the task's writing step for that purpose ([T7](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)). These are two stages of the same review purpose, not another reviewer hierarchy. Project-wide conventions, such as how errors are reported, are model Rules with project scope, so the root's integration review can check them across areas. Where the configuration allows it, one reviewer session assesses several no-change deliveries, each with its own verdict bound to its subject. Batching never removes a required subject or reduces the [protected preparation floor](../../history/apply-design-before-dec-056/manager-apply-execution.md#one-ledger-with-protected-completion-capacity).

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

Only guarded publication writes registry files. An unmanaged edit is drift: `check` validates it, and its owner registers or ignores the file before the next Apply. A registry revision is bound to the candidate tree. Review and check observations refer to that revision instead of being copied into mutable verdict fields.

The owner-accepted later [model-migration workflow](commands.md#model-restructuring-as-an-explicit-migration) also carries the registry/configuration adjustments necessary for an explicit identity and responsibility transfer. It reviews model and metadata together and retains realization artifacts. This is future controlled model-operation publication, not permission for current Apply workers to edit protected inputs or for unreviewed registry edits to bypass review; its producer support remains unimplemented.

Registry paths are independent of Package paths; lending can own source, test and documentation files in three directories. Other Managers can read or cite those files but cannot claim them. Managers propose registration through Orchestrator tools; the Orchestrator validates them and builds the registry delta. Editing the registry file directly is not a way around ownership.

Before writing a new path, a Manager reserves it through the Orchestrator, which checks the operation target, permitted creation areas, exact spelling and exclusive ownership. Model and configuration paths cannot be obtained through an artifact reservation. A shared file has one owner; others request changes through their common parent. A rename needs a new reservation and keeps provenance. A deletion removes the current registration only with a justified candidate deletion; history keeps the old registration. Moving responsibility between Packages is an explicit, recorded transfer reviewed with the affected delivery.

Coverage classifies every repository file as a canonical or Markitect input, a registered artifact or an explicitly ignored file ([DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). Registrations need a valid owner and a resolvable model basis. Overlapping ownership, missing files, stale registrations and unexplained files are findings. A blanket ignore cannot turn missing realization into success. Ignore configuration is project-owned, fixed for the run and read-only during Apply; Managers cannot hide a file by adding an ignore. Coverage ignores never change the language loader's complete scan of Definitions.

### Markitect state

Markitect's own state stays out of the realization: the journal lives outside the repository, and the small control metadata in `.markitect/state/` is ignored by Git and excluded from source staleness ([journal](../../history/apply-design-before-dec-056/manager-apply-journal.md#physical-store-and-commit-contract)). Repository classification and ownership belong to the Orchestrator's coverage stage, which keeps the language compiler free of I/O.

### Removed and moved intent

Apply also removes realization whose intent is gone ([DEC-034](../../concepts/register.md#dec-034-removed-intent-removes-its-realization)). When a Definition disappears, every artifact whose only basis it was is orphaned. Its owner deletes it in the same Apply, or records a new justified basis that review accepts; an artifact that keeps other bases only loses the removed link. If the owning Package itself disappeared, its records pass to the nearest existing ancestor Manager, or to the root. Language v1 does not guess renames (section 9), so a moved or renamed Definition is a removal plus an addition: the link to the old identity becomes a blocking finding, and the owner re-registers the artifact against the new identity with review. When the move crosses Packages, the record moves to the new owner's registry file through a recorded transfer. Stale code from removed intent is not a matter of style; leaving it fails the reconciliation.

### Unexplained files

A file of the captured source that no registration or ignore entry explains is settled by its owner before Apply: the owner registers it against Definitions or ignores it with a reason. No run decides it, and the file check fails the run on an unexplained file ([DEC-052](../../concepts/register.md#dec-052-implement-then-review-no-explore-phase)). Final publication requires conforming coverage of the aggregate candidate.

## Apply lifecycle

Logical Apply is the whole reconciliation below; guarded publication is its last step.

| Stage | Work and exit condition |
| --- | --- |
| Prepare | Read the model, configuration, registry and ignore policy from the committed state. Uncommitted changes to tracked files, or untracked files outside Git-ignored paths, stop the run before preparation. Acquire the repository snapshot and tool identities, compile v1 and bind the organization. Invalid model inputs stop the run. |
| Start root | Deliver the static instruction at the root and guarantee unchanged propagation to every local Manager with separately compiled context. Fix the artifact-only target. Ownership of every source file is settled before the run. |
| Work and review | Every local Manager, including empty and intermediate Packages, works from the start; a Manager that needs no change submits a no-change result ([T5](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)). Local changes happen in an owned candidate workspace. The Orchestrator harvests the actual delta; a fresh reviewer assesses it. A failed review returns grounded rework; an unresolved model question pauses that task without editing the model. |
| Integrate upward | The parent's integration step receives the composed aggregate of its subtree ([T9](../../history/apply-design-before-dec-056/manager-apply-core.md#manager-tasks)) and checks interface and cross-area consistency. A fresh integration reviewer examines the aggregate and relevant child evidence. |
| Verify aggregate | At the root candidate: required project checks, then the Orchestrator's final evidence check over the whole Manager tree. The root's fresh integration review has already assessed the aggregate semantically. |
| Preflight | Recheck the captured repository, model, configuration, registry and ignore inputs. Drift makes the run stale; earlier checks cannot authorize changed bytes. |
| Publish | If the configuration requires the owner's approval for an affected Package, the Orchestrator raises an approval question with the result summary, and the run waits for the answer. The guarded writer then applies the accepted candidate and registry revision to the working tree. Apply never commits, merges, releases or deploys. |
| Read back | Confirm actual paths, bytes, modes and registry against the verified candidate. A partial write or failed read-back is not a successful Apply. |

Full reconciliation of the selected local model is the default; it does not claim coverage of intent outside the model. Do not implement incremental reuse first; [computed scope](#toward-computed-scope) follows later. The graph and registry can explain possible impact; unknown semantic impact broadens examination.

The Orchestrator keeps one accepted aggregate per run and composes each accepted delivery onto it by replaying its owned delta ([D2](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)); every step starts from the current aggregate ([D3](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). A failed composition condition is an Orchestrator error that stops the run, not a merge. A parent never writes child-owned paths; a correction to child content is an issue addressed to that child ([D5](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition)). Parent-owned integration artifacts can be changed by the parent. [D4](../../history/apply-design-before-dec-056/manager-apply-core.md#deliveries-and-composition) decides which reviews composition renews. No last-writer-wins merge or silent reassignment resolves a conflict.

The owner's commit of the published result is the acceptance, exactly as a committed model revision is the accepted specification ([DEC-035](../../concepts/register.md#dec-035-committed-model-in-owner-commit-accepts-the-result)). The run report proposes a commit message with the trailers `Markitect-Run` and `Markitect-Model`. After a partial publication or a result the owner does not accept, the owner commits or reverts the working tree before the next run.

The rejection allowance is finite, and after repeated rejections the parent assesses the plan; the [execution contract](../../history/apply-design-before-dec-056/manager-apply-execution.md#admission-amendments-and-finite-correction) owns counting, budgets and exhaustion. A parent cannot turn an unresolved child delivery into a complete one by summarizing it.

The [execution contract](../../history/apply-design-before-dec-056/manager-apply-execution.md#one-run-state-machine) owns the single run state machine and its [terminal outcomes](../../history/apply-design-before-dec-056/manager-apply-execution.md#terminal-outcomes-and-displayed-status). History survives interruption: reopening a run first reconciles recorded invocations, child lifecycle and publication outcomes. Completed provider work is not replayed automatically, and unknown side effects stay visible.

## Reports and execution history

One run history serves two views: concise upward reports for coordination and a drill-down record for investigation. Reports reference evidence; they do not replace it. A parent can request underlying sources, candidate differences and conversations.

At minimum the history records run, task, attempt and Agent identities, causal links, effective role configuration, model and candidate bindings, prepared context, actual rendered prompts and custom instructions, sent requests, returned messages and reports, context and tool reads, delegation, reservations, candidate deltas, review submissions and rejections, checks, integration, publication and read-back. Original events are kept; corrections are new events. Large payloads are content-addressed blobs. Prepared, sent and confirmed or unknown delivery states stay distinct, so a built template never counts as a received request. Opaque provider behavior is a visibility limit, not a transcript Markitect may invent.

Every candidate transition identifies its prior and resulting tree and a retrievable difference. Records distinguish a proposed change, an Agent claim, an Orchestrator-observed change, a reviewed aggregate and an applied result. Rejected candidates stay inspectable. Parallel causal branches are not flattened into a false order by timestamps.

The normal user report presents the result, model basis, evidence, limitations and decisions needing attention. A small coordination account from the same records shows local work and no-change outcomes, cross-area requests, invocations, known usage and elapsed time. It separates a Manager's own effort from descendant work and waiting, and unknown usage stays unknown. It always reports the same measures: Agent steps, sessions and tokens per role, review submissions and rejections per purpose, findings by kind and severity, the share of no-change deliveries, owner interventions and active execution time. These feed the evaluation of [DEC-009](../../concepts/register.md#dec-009-the-central-evaluation-question) and [PRM-005](../../concepts/register.md#promises). The account helps judge whether one Manager per Package works in practice; it adds no scores, thresholds or automatic reorganization.

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
| Review cannot converge or a model contradiction remains | Finish Incomplete within the rejection and assignment limits, with the open question and retained evidence. |
| A Definition was removed, but its artifacts remain | Their owner deletes them or records a new basis that review accepts; otherwise the run cannot complete. |
| A Definition moved to another Package | The link to the old identity is a blocking finding until the owner re-registers the artifact; a move across Packages transfers the record. |
| The model or another file has uncommitted changes when Apply starts | Refuse before preparation. |
| Configuration or custom instructions state a project rule | The reviewer records a finding; the rule moves into the model. |
| Publication writes some files and then fails | The run reconciles until the actual writes are known. A known partial result finishes Failed with its exact paths. |
| An Agent reports success without a recorded delta or evidence | The Orchestrator refuses completion. |
| A required event cannot be stored durably | Stop before further dispatch or publication. |
| A rendered context exceeds the size limit | Stop preparation with a finding to split the Package. |
| Composed deliveries touch the same file | Orchestrator error; ownership is broken and the run stops. |
| A consumer reads another area's files or data directly | The integration review records a finding; the consumer must use the producer's API. |
| A provider call fails before anything was sent | Retry automatically within the configured limit, each attempt charged; otherwise reconcile. |

## Toward computed scope

Computed scope, the vision's first mechanism, follows once the two-Apply cycle works. It derives the affected Managers from the model difference per Definition, through the direct references and the registry links. Only those Managers receive tasks, and unknown impact still broadens the scope. Managers outside it get no task, and their last accepted realization stands. Because declared references cannot show every code dependency ([ENH-001](../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact)), a full reconciliation stays available and advisable from time to time. The data this needs is recorded from the first run on: Definition digests per run, registry links and the reference graph.

## Relation to current source and delivery

The [rebuild plan](../../history/apply-design-before-dec-056/manager-apply-rebuild-plan.md#source-replacement-map) decides which current mechanisms are reused, rewritten or deleted; the existing Reconcile operation already asks for complete responsibility against unchanged intent and reasoned no-change results. The [delivery plan](../../implementation-plan.md) owns packages and order, and the [acceptance gates](manager-apply-implementation-plan.md) own what each step must prove. Further commands, team strategies, concurrency and incremental reuse wait until the two-Apply cycle is demonstrated. Government remains a later, separate authority over model changes ([DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)).

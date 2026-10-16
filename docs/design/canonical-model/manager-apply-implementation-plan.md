# Implementation plan for the first reliable Apply

Date: 16 October 2026. Revised after the [owner's two-Apply and clean-redesign clarification](sources/manager-realization-20261016.md#two-apply-cycle-and-freedom-to-redesign). This is the proposed delivery plan for the [Manager Apply concept](manager-apply-concept.md). P00 to P12 are local plan identifiers, not active backlog assignments. Acceptance criteria below describe future evidence; writing the plan does not establish that they passed.

The [supplied concept review and its resolution](review-resolution-20261016.md) add concrete criteria to these slices: boundary questions, grounded Definition dispositions, distinct review purposes, inspectable preparation and observed coordination effort. No new role, language layer or separate optimization subsystem is needed for these refinements.

## The actual target

The principal demonstration is one continuous development sequence:

1. Describe the small library system fully and write its intended world as canonical model M1.
2. Run real Managers and their Agents through Apply R1 to produce implementation, tests and documentation A1.
3. Examine A1's actual code, behavior, tests, documentation and execution history against M1.
4. Change the model to M2 while keeping A1 as the starting realization.
5. Run Apply R2 to reconcile that existing realization with M2, producing A2.
6. Examine A2 again, including behavior, changed and unchanged areas, integration and traceability.

This demonstrates initial realization and subsequent model-driven development. Repairing artificially damaged artifacts against unchanged intent remains a useful additional test, not the primary scenario. A third, no-change Apply can check idempotent reconciliation, but does not replace the required model-change cycle.

Use one local model, one ordinary root Manager, one Manager per Package and a shared hierarchy for code, tests and documentation. The initial strategy uses a Manager and fresh independent reviewer per active area, configurable profiles and sequential writing. Other team strategies and commands remain extensions. The completed canonical language v1 remains the language contract.

## Clean redesign policy

The owner explicitly authorizes a direct, potentially radical rebuild around the new concept. There is no requirement to keep current commands, schemas, record codecs, package layout, tests or workflows compatible. Delete obsolete source and replace inappropriate abstractions instead of building adapters merely to preserve them. The old model is evidence of prior work, not the design template.

There will be one target product architecture and one target canonical model path. No compatibility frontend, mandatory legacy adapter, dual-format selection or permanent parallel product route is planned. Existing primitives are candidates for reuse only when they fit the new responsibilities and make the result simpler. Their interfaces and implementations may change.

Historical source and evidence remain in Git or their original artifacts; they do not require a live compatibility interpreter in the rebuilt product. Old verified records cannot authorize new work. Published assets remain historical immutable files. This policy concerns product restructuring, not deletion of unrelated local work or external project data.

Small steps mean bounded, reviewable changes with tests for the target contract. They do not mean keeping every obsolete command working between intermediate commits. Remove obsolete tests together with the behavior they specify; preserve or rewrite tests for still-required invariants such as target isolation, stale-input refusal, bounded review, actual-delta validation and honest partial failure. Update checks, canonical sources and guidance as part of each deliberate architectural replacement rather than adding exemptions to old gates.

## What the current source tells us

Source inspected: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. These are code-reading findings and reuse candidates, not a fresh runtime or CI verdict.

| Current area | Finding and proposed treatment |
| --- | --- |
| [Structural Core](../../../src/internal/core/types.go) and [compiler](../../../src/internal/core/compile.go) | The Schema/Definition envelope and namespace identities differ from completed v1. Implement the v1 contract directly; retain individual useful algorithms only where they fit. |
| [Fixed project Schema](../../../src/internal/core/projectmodel/projectmodel.go) | Manager, Statement, Artifact, Check and Decision are currently predefined authoring types. Remove this as the architectural center; derive organization from Packages and keep execution/realization records outside canonical language semantics. |
| [Project acquisition](../../../src/internal/application/projectwork/project.go) and [Runtime Host](../../../src/internal/runtime/projectrun/types.go) | Current planning and execution are tightly coupled to old Project/Report DTOs. Replace that coupling using the new architecture; a temporary old-model adapter is not a milestone. Rewrite a coupled unit if that is clearer than adapting it. |
| [Executor protocol](../../../src/internal/runtime/agentexec/README.md), native/process transports and profiles | Useful implementation material for starting role sessions and receiving results. Select the first real transport during architecture design; no obligation to preserve every current adapter or response schema. |
| Snapshots, [guarded publication](../../../src/internal/runtime/projectrun/apply.go), candidates and [record storage](../../../src/internal/infrastructure/recordstore/store.go) | Evaluate the primitives against the new contracts. Preserve required properties through tests, not necessarily existing type names or call graphs. Add explicit read-back after final writing. |
| [Current playground](../../../experiments/case-playground/README.md) | Its existing purpose is a comparative multi-wave experiment. Design a simple library scenario runner first. Reuse capture/container tools if useful; do not carry its old onboarding or command sequence into the new product as requirements. |

Before coding the replacement, make a keep/rewrite/delete map at responsibility level. It should identify concrete useful components, obsolete consumers and the tests that move with the target responsibilities. It is not a commitment to retain every accepted historical interface. The owner's instruction to simplify and delete supersedes the earlier compatibility-oriented delivery sequence retained in Git history.

## Library scenario and independent expectations

First model the library in readable prose, then encode all selected statements with ordinary v1 Kinds, Properties and concrete Definitions. The [small library M1](library-example/README.md) encodes the [confirmed business rules](library-example/business-rules.md) and has bounded structural evidence; the evaluation interface and independent executable expectations remain P00 work. Its Packages are library for shared scope/rules, catalog, members, lending and lending/reminders. Cross-Package relationships, returns, a loan limit and simple overdue reminders are explicit. Review the model for meaning and internal consistency before using it to generate implementation.

Keep the first domain example small: a local callable service or CLI with no UI, deployment or production database. Select the precise fixture interface in P00, rather than freeze an existing product example's design. A small Go service is a possible implementation choice. Explicit assessment dates avoid reliance on the machine clock. The first scenario processes domain operations sequentially; concurrent borrowing correctness is a separate future case.

M1 permits a new loan only when the resulting current count for a registered, active member is at most five. M2 changes that admission limit to three. Existing loans remain returnable; lowering the limit does not delete them or force a return. In the authored M1, the policy body refers to its structured value without duplicating the numeral, so M2 changes only `spec.maxOpenLoans` in `lending/loan-policy`. Prose and spec must still agree; no precedence rule hides a contradiction. Apply is still artifact-only: the scenario author makes the M1-to-M2 model edit between runs, and each Apply treats its selected model as read-only.

| Case | M1 expectation | M2 expectation |
| --- | --- | --- |
| Active registered member with two loans requests an available copy | Accept; count becomes three. | Accept; count becomes three. |
| Active registered member with three loans requests an available copy | Accept; count becomes four. | Reject; count and copy state remain unchanged. |
| Active registered member with four loans requests an available copy | Accept; count becomes five. | Reject further borrowing. |
| Active registered member already at five requests a copy | Reject; state remains unchanged. | Reject further borrowing. |
| Unregistered or suspended member requests a copy | Reject. | Reject. |
| An unavailable copy is requested | Reject. | Reject. |
| A currently borrowed copy is returned | Reduce loan count and make the copy available. | Same behavior. |
| Reminder assessment at an explicit date | Include open overdue loans; exclude returned and not-yet-overdue loans. | Same behavior. |

The three-loan request is an especially useful witness: exactly the same input changes from allowed under M1 to rejected under M2. This demonstrates that the second Apply followed changed intent.

Author and review the expectation sets independently of the generated library implementation. Keep their bytes outside Agent write scope and recheck them before/after evaluation. Agents may write normal tests as artifacts, but those tests cannot replace the independent assessment. Read-only scope is validated by Markitect; it is not a claim of OS isolation from same-user processes.

This independent expectation set evaluates the library experiment. It does not introduce a permanent second specification that every adopting project must maintain. Generated project tests remain ordinary artifacts reviewed against the canonical model.

R1 starts with the model, project configuration and only the minimal fixture/build contract needed to evaluate the result. It does not start with a handwritten complete library implementation. R2 starts from the actual reviewed A1 tree and registry; do not regenerate from a fresh empty directory or manually patch its code first. Save all failed attempts, rather than replacing them with an unexplained successful snapshot.

## Two design packages before implementation

The Go source architecture and Playground need deliberate designs. Their target responsibilities follow the concept, not the old package tree. The following boundaries are starting questions, not prescribed package names:

- Model acquisition and pure language compilation.
- Organization, context preparation and explicit operation targets.
- Agent sessions, task delegation, communication and finite execution.
- Candidate ownership, realization registry, review and integration.
- Project checks, aggregate verification and guarded publication/read-back.
- Durable records and report/inspection projections.
- Application entry points and transport adapters.
- External scenario preparation and independent assessment in the Playground.

The Go design must choose concrete packages, dependency direction, data ownership and lifecycle contracts, then map current source to keep/rewrite/delete. It may replace the current layer/package structure deliberately. Keep model providers and repository I/O out of deterministic language semantics.

The Playground design must define how it creates the library repository, supplies M1/M2, launches the real product, captures outputs and evaluates them. It owns scenario inputs and expected results. It must not implement its own Manager hierarchy, prompt delegation, review loop or alternate Apply. Comparative arms, cost studies and a broad benchmark program are outside this first demonstration.

## Starting Agents and carrying communication

The product needs an actual execution subsystem, not only compiled prompts. Choose one real executor first and exercise it through a narrow session interface: start, submit task/context, receive events/results, stop/cancel and establish the observed terminal state. Existing native App Server code is a reuse candidate; a scripted executor remains useful for tests only.

The Host starts the root's session. When a Manager delegates, it emits a structured child task through the Host. The Host validates the receiving Package and operation scope, builds its context, starts the configured role session and records its identity. Each parent supplies the child's outcome-oriented task prompt. The initial user instruction is not independently broadcast to all Managers.

Route delegation, clarification requests, detail reads, reports, review findings and rework through durable Host-mediated messages. Messages identify the run, task, sender, recipient, attempt and causal predecessor. Child results return to the parent; sibling coordination uses the nearest common parent. A waiting task has an explicit state and finite budget. Reviewer invocations start with fresh context and the actual candidate, not the implementer's full conversation.

Keep one clear relationship between logical Manager identity and concrete session/attempt identities. A continued work session and a newly started reviewer session must be distinguishable. Record actual sent prompts, supplied history, tool/context results, returned content and observation limits. Session launch, communication and cleanup are acceptance criteria of the first real scenario, not a later adapter project.

## Small dependency-ordered steps

P00 through P12 are planning slices. Split the larger design or compiler slices into the listed substeps, each with its own focused review and gate. Implementation starts after the architecture and Playground contracts are concrete enough to avoid coding an accidental second design.

### P00 Model the library and choose the two-run change

Write the readable project description, complete M1 canonical sources, the exact M2 edit, a minimal evaluation interface and independently reviewed expectations. The model can initially be checked by the bounded design tools where supported, with their actual limitations recorded; production-v1 validity is established in P03. Do not copy the intentionally invalid Command/command Exchange fixture as a positive model.

Derive a small optional starter vocabulary from this example and keep it an ordinary, visible model. Use domain terms where useful. During model review, list statements whose behavior crosses Package boundaries and check the explicit connections. Include obligations carried by Kind and Property bodies and their concrete uses. This is a readable review of the example, not a new requirements database or semantic index.

**Gate:** every scenario expectation has a clear basis in M1 or M2. Cross-Package connections are explicit, including the membership suspension that lending must enforce. Vocabulary declarations cannot hide required behavior behind a vocabulary-only label. The model change is an intentional five-to-three change with no contradictory body/spec values. No implementation solution or file-edit plan is embedded in the expected-result set.

### P01 Design the Go product architecture and removal map

Specify target responsibilities, packages, allowed imports, runtime data contracts and lifecycle ownership. Identify what will be copied, adapted, rewritten or deleted. Include a decision on the first executor transport, configuration ownership, source binding, registry writes, communication and record formats. Design from the completed model language and Manager concept; preserving the old Project/Report DTO is not a goal.

**Gate:** walk a root task and one child through the proposed boundaries from model input to candidate, review, publication and reporting. Every state/output has one owner. No architectural dependency exists solely for backward compatibility. The removal map covers obsolete commands, tests, model sources and generated views, not just Go files.

### P02 Design the library Playground scenario

Specify an external runner that prepares the repository, selects a fixed Markitect build/profile, executes the actual product, inspects A1, makes the model-only M2 change and executes R2 against A1. Define source snapshots, result locations, independent checks and readable comparison reports. Reuse existing Playground infrastructure selectively.

**Gate:** a dry scripted run proves scenario preparation, artifact capture, expectation invocation and M1/M2 bookkeeping. It does not count as real Apply. The runner never fabricates Agent reports, contains no duplicate Manager/review orchestration and never repairs generated code between R1 and R2.

### P03 Implement complete production v1 compilation

Deliver bounded slices: **P03a** restricted YAML values/envelopes with exact numbers and provenance; **P03b** model roots, Packages, manifests, dependency acquisition and strict Unicode/path rules; **P03c** fixed Core, Kind/Property contracts, references, objects and finite recursion; **P03d** body links and their Markdown profile; **P03e** canonical bytes, locks and full conformance results. Acquisition supplies explicit inputs to the pure compiler.

**Gate:** M1 and M2 compile with different correct content digests. Independent conformance cases cover more than the library's constructs. Invalid and incomplete input gets no successful canonical bytes/digest; resource exhaustion is distinct from invalid language. Core byte fixtures, exact large numbers, body content, names/imports and dependency-lock counterexamples have independent expected results. No partial compiler is advertised as full v1.

### P04 Establish the new product skeleton and simple preparation

Implement the agreed architecture's application composition, model input, Manager topology, profiles and own-Package/See-also context. Add the registry/classification and target contracts needed by subsequent slices. Preview the actual statements, contracts, responsibilities and explicit connections with inclusion reasons before starting Agents. Add a compact Package directory for routing questions without loading all foreign content or extending imports. Select one new product path; delete obsolete loader/entry-point dependencies as they are replaced.

Plan the repository's own bootstrap at this checkpoint: the new compiler/check/context can validate a freshly authored Markitect model and source ownership before the full Apply exists. Update active CI/contributor checks and generated-source owners to the implemented target in the same reviewed source cut. Do not keep a legacy compiler merely to validate an obsolete self-model. If a larger cut is needed, prepare it in an isolated feature branch and integrate when its new required checks are meaningful and passing.

**Gate:** root and local Package identities/contexts are correct, imports do not spawn library Managers, and complete values/bodies survive preparation. Inclusion reasons and the Package directory are inspectable and distinguish source facts from explanations. A targeted detail read is available. The rebuilt tree has one active source model/check route, even though Apply is not yet complete. Relevant new architecture and coverage checks pass; obsolete behavior tests are removed rather than silently skipped.

### P05 Start real role sessions and persist communication

Implement the selected executor interface, effective role profiles, Host-mediated task/message transport, session identity, cancellation and the minimum durable prompt/event history. Start with a root and one child role; add fresh reviewers using the same mechanism. Test ordinary completion, process/transport failure and uncertain termination.

**Gate:** a controlled adapter test verifies exact prompts, context, messages, causal links and lifecycle states. A separately identified bounded real-adapter smoke proves actual Agent start, response and termination. Scripted success is not substituted for that smoke. The Host refuses further unrecorded dispatch if required event storage fails.

### P06 Produce and review one Package candidate

Add local artifact workspaces, exclusive path reservations, Host-owned registry changes and fresh independent review. Permit implementation, normal tests and documentation for one Package. Enforce artifact-only changes and finite rework.

Manager and reviewer explicitly assess which local statements need another area's behavior and whether the needed model connection is present. Use ordinary open questions containing the source passage, suspected effect and affected area when known. Each positive Definition disposition needs a reason and concrete realization or use with supporting evidence; the Host validates report structure and bindings, the reviewer validates substance.

**Gate:** controlled tests reject attempted model/config/ignore edits, direct registry modification and overlapping ownership. A bad candidate is rejected, revised and reviewed as a new exact candidate. A vocabulary-only claim that ignores obligations on concrete uses fails review. Missing evidence, an unresolved required assessment or review exhaustion cannot return completion. No target checkout is published in this slice.

### P07 Delegate recursively and integrate upward

Add the library's ordinary root Manager, the library, catalog, members and lending Package Managers, lending's reminders child and parent coordination. The library Package owns shared scope and rules; it is not a special root role. Parents issue their own task prompts, compose child results and request corrections from the accountable child. Route suspected missing connections through the common parent without adding links during Apply. Add read-only intake for initially unexplained artifact ownership. Integration review focuses on connected behavior across the composed areas, beyond their local deliveries.

**Gate:** only root receives the user instruction; the trace shows actual child prompts and responses. A parent can integrate deliveries but cannot casually author arbitrary child-owned files. A sibling contract request or boundary question reaches the affected area through the parent. A missing connection needed for sound assessment yields an open model issue and prevents aggregate completion. A broken aggregate fails integration review even if individual deliveries passed; blocked children prevent a complete result.

### P08 Verify candidates and expose the full report

Run project-owned checks, full file accounting, Definition assessments and fresh aggregate verification on the assembled candidate. Check both directions: artifacts have justified model links, and required Definitions have justified realization. Add independent Playground expectation evaluation without putting those assessment sources inside worker write scope. Expose the run tree/timeline with prompts, additional reads, reports, reviews and deltas. Derive a concise user result and coordination account from the same events, separating local work, cross-area requests, observed usage and waiting from descendant effort.

**Gate:** actual candidate bytes, registry and evidence bindings agree. Missing child reviews, stale evidence, unsupported Definition dispositions and tampered assessment inputs invalidate success. All files being owned cannot compensate for missing required realization. A local pass cannot stand in for integration or final verification. Resolved findings can be referenced, but changed subjects or assumptions require renewed assessment. Reports distinguish Agent claims from observed changes and unknown usage from zero; they add no automatic restructuring, scoring or thresholds. The normal tests generated with the code do not replace independent behavior or source inspection.

### P09 Guard publication read back and handle interruption

Implement or reuse the agreed source guards, candidate/registry publication, post-write read-back and recovery records. Test these mechanics on owned temporary repositories. Reuse a suitable primitive directly or rewrite it if old workflow dependencies dominate it; no old command/API compatibility is required.

**Gate:** resulting paths, bytes, modes, deletions and registry match the verified candidate. Changed model/source/configuration inputs cause stale; incomplete writing or read-back failure never reports successful Apply. Recovery inspects recorded outcomes and does not blindly replay a completed Agent invocation. No-change still requires a justified assessment.

### P10 Run and inspect the real two-Apply library cycle

Freeze one Markitect source/build S and effective role configuration for the experiment. Use the real selected Agents to realize M1 from the minimal initial project. Save model, configuration, messages, A1 and reports. Independently inspect the actual code structure, error paths, tests and documentation, and run M1 expectations. Record findings, including excessive scaffolding, unused code or test-specific shortcuts.

Then make only the agreed model edit to M2. Keep A1's implementation and registry as the starting point and use the same product build/profile. Execute Apply R2, save A2 and inspect again. Run M2 expectations and unchanged behavior cases; examine the code/documentation difference and the model basis of each change. No hand-written code patch is allowed between the two Applies in the acceptance scenario.

Inspect the existing reports for local work versus coordination, observed Agent effort and required human intervention. Use concrete problems to suggest later Package or team changes; do not infer a generally superior organization or a cost result from this one case.

**Gate:** both real runs have recorded Agent lifecycle/communication, independent review, aggregate checks and exact applied read-back. The identical three-loan borrowing input is allowed under M1 and rejected under M2. Unchanged membership/catalog/return/reminder behavior stays correct. Source inspection supports the claimed realization; passing tests alone is insufficient. A2 clearly derives from A1 plus M2, not a hidden reset or reference implementation.

If a product defect requires changing S, retain the failed evidence and restart the complete cycle against the new build for a comparable final result. A further no-change run is useful supporting evidence. One successful cycle is a bounded feasibility result, not a global proof, comparative study or general cost claim.

### P11 Consolidate reliability against concrete failure cases

Negative tests belong to P03-P09 as each contract is implemented. This checkpoint checks their composition: stale target, missing link surfaced as a model issue, vocabulary-only used to evade obligations on concrete uses, complete file coverage with missing required realization, a local pass offered for another review purpose, ownership conflict, invalid child report, exceeded review limit, communication interruption, event-storage failure, rejected aggregate and partial publication. A controlled missing-link case proves the question/coordination/failure contract when a gap is reported; it cannot prove that Agents discover every semantic gap.

**Gate:** each case has a reproducible input, expected honest outcome and inspectable trace. The real two-Apply cycle still works on the current candidate. A failure cannot be hidden by an ignore, a rewritten model, an invented review or a final summary claiming completion.

### P12 Finish the clean product and evaluate the next direction

Remove remaining unused source, adapters, old-format examples and obsolete command/documentation claims. Review the architecture against the implemented ownership/dependency map. Refine Markitect's own canonical model from the new architecture and intended product obligations; no obligation to mechanically migrate the old taxonomy or expose historical runtimes remains.

**Gate:** one target execution path, current source model and maintained Playground case remain. Source, architecture, coverage, generated-view, CLI/MCP and packaging checks match the new product and pass before integration. Historical releases/evidence retain their original identities. Record what the two-Apply case actually establishes and use its findings to select further work, rather than immediately adding all proposed commands or Government.

## Evidence and implementation discipline

Each implementation slice binds its source SHA, task scope, retained/removed contracts, checks, observed result and evidence class. Keep source tests, scripted executor tests, real Agent starts, R1/R2 behavior/source inspection, post-write read-back, hosted CI and human acceptance distinct. A fake executor is a diagnostic tool, not a replacement for the requested demonstration.

Small source changes can be developed and reviewed independently, and obsolete functionality may disappear as part of a deliberate cut. Before integration, required checks must describe and validate the new target. Remove or replace superseded checks with their owners; do not keep compatibility machinery just to make old assumptions pass, and do not weaken still-required target invariants. Follow the currently applicable repository contribution process until it is deliberately revised with the architecture work.

This plan updates design and scope. It does not execute providers, alter the production Go architecture or delete existing product source by itself. The next concrete work is P00's complete library model and P01/P02's Go and Playground designs, with their boundaries agreed before substantial runtime implementation.

Return to the [concept](manager-apply-concept.md), [north star](manager-realization-north-star.md) or [canonical-model overview](README.md).

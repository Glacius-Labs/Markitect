# Acceptance gates for the first reliable Apply

This document owns what each step of the rebuild must prove. The [delivery plan](../../implementation-plan.md) owns the packages, their order and dispatch; the [rebuild plan](manager-apply-rebuild-plan.md) owns what each package replaces. The gate identifiers P00 to P12 are stable names for acceptance criteria, not work assignments.

## The target

The principal demonstration is one development sequence on the small library:

1. Write the intended world as the canonical model [M1](library-example/README.md).
2. Real Managers and their Agents run Apply R1 and produce implementation, tests and documentation A1.
3. Examine A1's code, behavior, tests, documentation and execution history against M1.
4. Change only the model: M2 lowers the loan limit from five to three (`spec.maxOpenLoans` in `lending/loan-policy`). A1 stays the starting realization.
5. Run Apply R2 to reconcile A1 with M2, producing A2.
6. Examine A2 again, including changed and unchanged areas, integration and traceability.

The decisive witness is the fourth-loan request: after three successful loans, M1 accepts a fourth and M2 refuses it without changing state. Existing loans stay returnable. The [evaluation contract](library-example/evaluation-contract.md) owns the full case families. Repairing artificially damaged artifacts and a third no-change Apply are useful additional cases, not substitutes.

## Evidence discipline

Each implementation slice records its source SHA, scope, retained and removed contracts, checks, observed result and evidence class. Source tests, scripted executor tests, real Agent starts, R1 and R2 behavior and source inspection, post-write read-back, hosted CI and human acceptance stay distinct. A fake executor is a diagnostic tool, never the demonstration.

Obsolete behavior disappears together with its tests; tests for still-required invariants (target isolation, stale-input refusal, bounded review, actual-delta validation, honest partial failure) are kept or rewritten for the new target. Superseded checks are removed with their owners. No skip, allowlist or compatibility machinery makes an unimplemented replacement look complete.

## Gates and packages

| Gate | Subject | Delivery packages |
| --- | --- | --- |
| P00 | Library model and the two-run change | Design (done as documents) |
| P01 | Go architecture and removal map | Design; APPLY-BOOT enforces the target graph |
| P02 | Library Playground | APPLY-PLAY |
| P03 | Complete v1 compilation | APPLY-MODEL; APPLY-SNAPSHOT supplies tree values |
| P04 | Product skeleton and preparation | APPLY-ORG, APPLY-REALIZATION, APPLY-CUTOVER |
| P05 | Real role sessions and durable communication | APPLY-RUN, APPLY-JOURNAL, APPLY-RUNTIME, APPLY-AGENT |
| P06 | One Package candidate | APPLY-RUN, APPLY-REALIZATION, APPLY-RUNTIME, APPLY-REPOSITORY, APPLY-INTEGRATE |
| P07 | Recursive delegation and integration | APPLY-RUN, APPLY-INTEGRATE |
| P08 | Aggregate verification and reports | APPLY-INTEGRATE |
| P09 | Guarded publication, read-back and interruption | APPLY-REPOSITORY, APPLY-INTEGRATE |
| P10 | Real two-Apply cycle | APPLY-INTEGRATE |
| P11 | Failure cases in composition | Every package for its own cases; APPLY-INTEGRATE composes them |
| P12 | Clean product | APPLY-INTEGRATE, with the coordinator |

### P00 Library model and the two-run change

A readable description, complete M1 sources including the [callable interface](library-example/m1/model/library/callable-interface.yaml), the exact M2 edit and independently reviewed expectations. A small optional starter vocabulary derived from the example stays an ordinary, visible model. The model review lists statements whose behavior crosses Package boundaries, including obligations carried by Kind and Property bodies, and checks their explicit connections.

**Gate:** every expectation has a clear basis in M1 or M2. Cross-Package connections are explicit, including the membership suspension that lending must enforce. Vocabulary declarations cannot hide required behavior. The change is an intentional five-to-three edit with consistent body and `spec`. The expectations contain no implementation solution or file-edit plan.

### P01 Go architecture and removal map

The [Go architecture](manager-apply-go-architecture.md), [port contracts](manager-apply-port-contracts.md), [configuration](manager-apply-configuration.md), [execution](manager-apply-execution.md), [journal](manager-apply-journal.md) and [rebuild plan](manager-apply-rebuild-plan.md) specify packages, imports, runtime records, lifecycle ownership, the first transport and the keep, rewrite and delete map.

**Gate:** a root task and one child walk through the boundaries from model input to candidate, review, publication and report, as in the [library walkthrough](manager-apply-library-flow.md). Every state and output has one owner. No dependency exists only for backward compatibility. The removal map covers obsolete commands, tests, model sources and generated views, not only Go files.

### P02 Library Playground

The [Playground contract](manager-apply-playground.md), M1's [callable interface](library-example/m1/model/library/callable-interface.yaml) and the [evaluation contract](library-example/evaluation-contract.md) define the external runner, the two-run sequence, the ten operations, the case families, the source bindings and the comparison report. The independent oracle runs without the product and can be built in parallel with the compiler. The runner's scripted dry run needs the new product entry point.

**Gate:** the oracle discriminates wrong results, including state mutation on refusal and the opposite-profile mismatch on the fourth-loan witness. A scripted dry run proves scenario preparation, artifact capture, expectation invocation, M1 and M2 bookkeeping and preservation of failures; it is labeled synthetic and never counts as real Apply. The runner fabricates no Agent reports, contains no Manager or review orchestration and never repairs generated code between R1 and R2.

### P03 Complete v1 compilation

Bounded slices: restricted YAML values and envelopes with exact numbers and provenance; model roots, Packages, manifests, dependency acquisition and strict Unicode and path rules; fixed Core, Kind and Property contracts, references, objects and finite recursion; body links and their Markdown profile; canonical bytes, locks and full conformance results. Acquisition supplies explicit inputs to the pure compiler.

**Gate:** M1 and M2 compile with different, correct content digests. Independent conformance cases cover more than the library's constructs. Invalid or incomplete input gets no canonical bytes or digest, and resource exhaustion is distinct from invalid language. Core byte fixtures, large numbers, body content, names and imports and dependency-lock counterexamples have independent expected results. No partial compiler is advertised as full v1.

### P04 Product skeleton and preparation

The new application composition, model input, Manager topology, profiles and context packages, plus the registry in its `.markitect/registry/` layout, coverage classification, scope selection from the model and the target contracts the later gates need. The author can preview the statements, contracts, responsibilities and explicit connections each Manager receives, with inclusion reasons, before any Agent starts. The repository's own model, its initial registry, source ownership and gates move to the new compiler and check in the same reviewed cutover.

**Gate:** root and Package identities and contexts are correct, imports spawn no library Managers, and complete values and bodies survive preparation. Inclusion reasons and the Package directory are inspectable and distinguish source facts from explanations; a targeted detail read works. A Definition reaches another Package's context only through an explicit link or its modeled scope, never through a configured list. `check` validates Markitect's own registry, so every repository file is covered before Apply exists. The rebuilt tree has one active model and check route even though Apply is not available yet. Obsolete behavior tests are removed, not skipped.

### P05 Real role sessions and durable communication

The session port, effective role profiles, Host-mediated task and message transport, session identity, cancellation, the assignment budget and the journal. Start with a root and one child role, then add fresh reviewers through the same mechanism.

**Gate:** a controlled adapter test verifies exact prompts, context, messages, causal links and lifecycle states, including ordinary completion, process or transport failure and uncertain termination. Reservations and settlement, the completion-reserve admission rule, no charge while waiting, immutable outcomes and reconciliation follow the [execution contract](manager-apply-execution.md). A separately labeled bounded smoke with each real adapter, Codex App Server and Claude Code, in its runtime container proves actual Agent start, one Host tool call over MCP, a file change the Host observes, response and termination; scripted success never substitutes for it ([rebuild plan](manager-apply-rebuild-plan.md#bind-the-runtime-image-before-real-work)). The Host refuses further dispatch when required event storage fails.

### P06 One Package candidate

Local workspaces, exclusive path reservations, Host-owned registry changes and fresh independent review for one Package's implementation, tests and documentation, with finite rework. Manager and reviewer ask which local statements need another area's behavior and whether the model connects them. Every positive Definition disposition has a reason and concrete realization or use.

**Gate:** controlled tests reject model, configuration and ignore edits, direct registry changes and overlapping ownership. A bad candidate is rejected, revised and reviewed as a new exact candidate. A `vocabulary` disposition that ignores obligations on concrete uses fails review. An artifact whose only Definition was removed must be deleted or rejustified, and a project rule found in configuration or custom instructions becomes a finding. Missing evidence, an unresolved required assessment or exhausted rounds cannot return completion. Nothing is published in this gate.

### P07 Recursive delegation and integration

Follow the [fixed Manager cycle](manager-apply-concept.md#fixed-manager-cycle) and [library walkthrough](manager-apply-library-flow.md). The same static Apply operation reaches the root and every local Manager, including empty and intermediate Packages. Compiler-derived context and configuration are assembled separately for each. No Agent chooses participants, writes another Manager's operation or supplies a global delivery order. Adjacent messages carry questions, reviewed results and corrections. Parents retain their own prerequisites and integration work. A reference is not automatically an execution dependency.

**Gate:** a deliberately unhelpful root output cannot suppress any child receipt. The observed trace retains the same operation identity and complete local scope at every Manager, refuses sibling/non-adjacent calls, and includes independently reviewed no-change results. No artifact writing precedes the initial Explore assessment of every Manager; subsequent local planning has no global barrier. The Host records oldest-eligible readiness with stable ties, and parent hints cannot replace the operation or add binding rules. A waiting child holds no active write grant and does not prevent owned prerequisite work. A limited normal delivery needs independent review, Host acceptance and composition into the consumer's concrete base; it never completes the producer Manager. Unknown routing triggers bounded branch inquiry with explicit outstanding replies, not silent exclusion. Missing required model links stay open for separate model correction. Parent completion fails while any descendant, required review, dependency response or current-subject evidence is missing. Changed complete review bindings require renewed assessment, and unknown impact broadens it. A broken aggregate fails integration review even if every local delivery passed. Successful limited deliveries consume shared capacity but neither use nor reset rejection allowance. Rejections accumulate under a stable run/Manager/review-purpose identity, count once on replay, and cannot be split across new tasks or deliveries. At the default limit, two rejections require parent assessment before further correction and the third makes the work incomplete. Define the exact technical bindings before implementing this gate; controlled mechanics are not real-provider proof.

### P08 Aggregate verification and reports

Project checks and the Host's final evidence check (file accounting, the Definition census and review bindings) on the assembled candidate, in both directions: artifacts have justified model links, and required Definitions have justified realization. Independent Playground expectations run outside worker write scope. The run tree and timeline expose prompts, reads, reports, reviews and deltas; the concise user result and coordination account derive from the same events.

**Gate:** candidate bytes, registry and evidence bindings agree. Missing child reviews, stale evidence, unsupported dispositions and tampered assessment inputs invalidate success. Complete file ownership cannot compensate for missing required realization. A local pass cannot stand in for an integration review or the final evidence check. Reports distinguish Agent claims from observed changes and unknown usage from zero, and add no automatic restructuring, scores or thresholds.

### P09 Guarded publication, read-back and interruption

Source guards, candidate and registry publication, post-write read-back and recovery records, tested on owned temporary repositories.

**Gate:** uncommitted model or working-tree changes refuse the run before preparation. Publication writes only the working tree and never commits. Resulting paths, bytes, modes, deletions and registry match the verified candidate. Changed model, source or configuration inputs make the run stale. Incomplete writing or failed read-back never reports success. Recovery inspects recorded outcomes and never replays a completed Agent invocation. No-change still requires a justified assessment.

### P10 Real two-Apply cycle

Freeze one Markitect build and role configuration. Real Agents realize M1 from the minimal starting project; the model, configuration, messages, A1 and reports are saved, and A1's code structure, error paths, tests and documentation are inspected independently, including excess scaffolding, unused code or test-specific shortcuts. Then only the M2 edit is made, and R2 runs against the actual A1 and registry with the same build and profile. No hand-written code change is allowed between the Applies.

**Gate:** both runs have recorded Agent lifecycle and communication, independent review, aggregate checks and exact read-back. The fourth-loan witness changes from accepted to refused, and unchanged membership, catalog, return and reminder behavior stays correct. Source inspection supports the claimed realization; passing tests alone are insufficient. A2 visibly derives from A1 plus M2, not from a reset or a reference implementation. If a product defect forces a new build, the failed evidence is kept and the whole cycle restarts. One successful cycle is a bounded feasibility result, not a general proof or a cost claim.

### P11 Failure cases in composition

Negative tests belong to the gate that implements each contract. This gate checks their composition: stale target, missing link surfaced as a model issue, a `vocabulary` disposition used to evade obligations, complete file coverage with missing realization, a local pass offered for another purpose, ownership conflict, invalid child report, exhausted review rejection allowance, communication interruption, event-storage failure, rejected aggregate, partial publication, an orphaned artifact after a removed Definition, a dangling registry link after a moved Definition, uncommitted model changes and a project rule hidden in custom instructions.

**Gate:** each case has a reproducible input, an honest expected outcome and an inspectable trace, and the real two-Apply cycle still works on the current candidate. No failure can hide behind an ignore, a rewritten model, an invented review or a summary claiming completion.

### P12 Clean product

Remove the remaining unused source, adapters, old-format examples and obsolete command and documentation claims. Review the architecture against the implemented ownership and dependency map, and refine Markitect's own canonical model from the new architecture.

**Gate:** one execution path, one current source model and one maintained Playground case remain. Source, architecture, coverage, generated-view, CLI and MCP and packaging checks match the new product and pass. Historical releases and evidence keep their identities. What the two-Apply case established is recorded and used to choose further work, instead of adding every proposed command or Government at once.

# Standard operating model

This design owns the reusable operating contract for turning accepted canonical intent into independently verified representations. The operating methodology owns the wider method; the roadmap owns implemented versus planned status. This is a source-only development direction, not a release or an assurance claim.

## Canonical intent and ordinary discretion

The canonical model contains what applies. Exploration, alternatives and conversation history do not become canonical merely because an agent considered them. Users need not enumerate ordinary implementation freedoms: agents choose implementation details consistent with the applicable model. If requirements conflict or cannot be fulfilled together, return the specific decision to the owner rather than silently changing the standard.

The government analogy describes durable responsibilities: establish intent, coordinate affected work, execute bounded representations, independently examine them, and examine their composition. It does not require a government vocabulary in Core, an idea lifecycle, a permission-list authoring burden, or autonomous legislation.

## One reusable operating cycle

1. Bind the accepted source revision, runtime, target ownership and current evidence. Compile structural validity before implementation.
2. Derive affected representation work from model changes and observed drift. Keep the canonical reference graph distinct from the execution and assurance graph.
3. Account for every canonical Projection in the selected model. Relate its selected Definitions and Policies to representation work, owned artifacts, configured checks and current verification. Report missing selection instead of silently omitting it.
4. Execute work through the existing Module tools and bounded Executor. Independent targets may run concurrently when ownership and dependencies permit. Parents receive actual child candidates or current observed representations.
5. Independently verify local obligations and each parent's composition obligations. Acknowledging an observation list is bookkeeping, not proof of semantic understanding. Passing children never replace the parent's own evidence.
6. Feed fresh attributable failures back through the normal bounded repair path. Preserve failed attempts. Stale evidence, technical failure, unsatisfiable intent and missing inputs have distinct next actions.
7. Finish with a read-only audit of the entire declared model/target scope. No work, no refresh and no unknown target artifact are necessary but not sufficient: all required local and parent evidence must also be current and passing.
8. A later model or target change starts another cycle. A completed cycle remains tied to its fixed inputs; it is not a permanent assertion about a changing repository.

Review and Apply remain explicit trust boundaries. A review need not be performed by a human for every routine change, but an operating wrapper must not silently drop the existing exact-candidate review, stale-plan or write-intent contracts. No new command gains implicit mutation authority.

## Saved Apply-to-Verify handoff

A successful materializing Apply reports its exact canonical `sourceRevision` and immutable `evidenceRevision`. The optional `controller-verify --apply-result FILE` input reads that saved JSON report instead of asking an operator to copy the two revisions. The existing explicit `--base`/`--revision` form remains supported; the forms cannot be combined, including explicitly empty revision flags.

The saved report must be closed, bounded JSON for a `materialized-unverified` Apply, with full immutable source/evidence commits and nonempty, valid materialization records that all bind the same source. Refused, partial, no-materialization, inconsistent or malformed reports do not start verification. A retained projection needing fresh assurance is allowed: the normal Verify path decides whether current verification or separate evidence refresh is necessary.

The file is an explicit navigation input, not authenticated evidence of review, authorization, freshness or current ledger selection. Verify reconstructs the selected source, runtime, active records, artifacts, checks and child bindings exactly as in the explicit-revision path. A saved post-Apply report lets a fresh process continue with verification after an interruption; it does not reapply targets or supply an automatic retry policy. Reports predating the explicit source field can still be used through the existing revision form.

## Completion audit

The first implementation checkpoint adds a read-only controller audit. It uses the same proposal, record, verification freshness and assurance semantics as the existing controller. It invokes neither an Executor nor a Verifier and does not append records or alter artifacts.

The audit requires read-only content observation of all declared targets through an already configured auditAll runtime. Observation invokes no Executor, Verifier or fixed check and writes no operational state. It must not silently change runtime configuration to manufacture fresh evidence. Each canonical Projection is represented in the report, including its canonical subject and policy references, assurance scope, owned artifacts and available current record and verification identities.

Technical completion requires:
- at least one declared canonical Projection; an empty scope provides no completion evidence;
- every canonical Projection in the selected source to be accounted for;
- explicitly configured assurance scopes to be valid and reachable, including each required parent;
- no pending materialization, escalation, evidence refresh, ownership conflict or unknown artifact within the declared target inventory;
- current passing verification for every required local and parent scope, with exact target, source, checks, runtime and child bindings;
- stable target inventory, selected input bytes and ledger/active selection across final readback during the audit.

Intentional exact target-path exclusions remain visible in the report with their reasons, outside the managed claim. They do not remove canonical Projections or their obligations from accounting. An exclusion is not proof of conformity and need not block completion of the explicitly declared managed scope. Facts outside the declared model and target roots are not covered. A complete audit does not establish the sufficiency of arbitrary prose, unmodeled obligations, universal correctness or human acceptance.

The report binds the selected source and proposal, runtime configuration, ledger state and exact evidence used. It presents uncovered or stale items and next actions; it never turns a no-op into PASS. Reading a previously passing result without reconstructing its current binding is insufficient.

## Implementation boundaries

Core remains the structural Schema/Kind/Property/Definition compiler. Host owns execution, operational state and closure. Schema Modules supply vocabulary; Projection Modules supply target expertise. The default process is a first-class product behavior above Core. Its invariant mechanics are implemented once; project-specific checks and decisions remain project-owned.

Existing projection records provide source-to-artifact provenance. The audit adds the reverse accounting view: which declared representation obligations have corresponding current work and evidence. This is bounded structural accounting; semantic coverage still depends on meaningful independent checks.

## Finite delivery sequence

The audit and executable protocol examples are implemented over the current controller; their bounded coverage is recorded in [the validation guide](../validation/standard-operating-model.md). They exercise a model change across two representations, a failing local judgment, repair under unchanged intent, and final declared-scope closure. Completed live-agent pilots and their limits are recorded there as well; [the roadmap](../implementation-plan.md#standard-operating-model-checkpoint) owns the next evidence step. Preserve unaffected valid bytes and expose missing coverage.

A continuous local runner follows only after the finite cycle is usable. It must reuse the same commands and contracts, persist enough state to resume safely, bound retries, reject stale work and become idle when no work remains. Kubernetes is a possible later execution environment, not a prerequisite. Long-running scheduling and general automatic repair are not claimed by the audit checkpoint.

Do not expand the proof harness merely to add more experiments. Each checkpoint has a finite exit: supported behavior, meaningful negative cases, independent review and exact-source contribution evidence.


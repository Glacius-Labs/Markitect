# Prompt — Start the Parallel Markitect Development Wave

Continue from the repository state produced by the immediately preceding **parallelization / coordinator preparation** task.

Do not assume this prompt's suggested workstream graph overrides the repository-owned workstream contracts you just established.

Your first responsibility is to consume your own final coordinator output and the canonical repository documents created or updated during that preparation.

The objective of this iteration is:

> **START THE FIRST SAFE PARALLEL DEVELOPMENT WAVE AND COORDINATE IT THROUGH INTEGRATION.**

You are no longer expected to implement every workstream yourself.

Act primarily as:

- Integration Coordinator
- Architecture Guardian
- Shared-Contract Owner
- Workstream Dispatcher
- Integration Reviewer

Use bounded parallel implementers for implementation work wherever the finalized architecture says that parallel work is safe.

---

# 1. Preconditions

Before launching any workstream, confirm the coordinator-preparation task produced:

- an exact parallel-development baseline commit,
- green required repository gates,
- current architecture/invariant documentation,
- workstream specifications,
- explicit dependency information,
- an adapter parallelizability assessment,
- clear ownership boundaries for Init, Copy Me, adapters and Konfyra adoption.

If any of these are missing in a way that makes delegation unsafe, repair only that preparation gap first.

Do not invent a new product feature to unblock delegation.

---

# 2. Use the repository as authority

Future implementers must work from repository-owned context, not previous conversation history.

Use the canonical documents created during the previous iteration, including as applicable:

- architecture
- engineering constitution
- Core invariants
- parallel-work guide
- parallelizability report
- workstream specifications
- adapter contract
- risk register / triage guidance
- existing-project adoption design
- Init/adoption design

When dispatching an implementer, point it to the relevant repository documents.

Do not rely on hidden context that only you know.

---

# 3. Launch the maximum SAFE parallel set

Use the finalized dependency graph from the previous task.

Do not serialize work merely for convenience.

Especially:

> **Unrelated adapters should be developed in parallel if the adapter contract is stable enough.**

If the previous parallelizability audit concluded that adapter fan-out is safe, launch the individual adapter workstreams concurrently.

Expected possible workstreams include:

- Existing-project Init / Adoption Bootstrap
- Copy Me / Engineering Discovery
- Risk / Invariant Triage
- .NET Adapter
- Documentation Adapter
- Codex Adapter
- Claude Adapter
- GitHub Adapter
- Azure DevOps Adapter
- Konfyra Adoption / Inventory

Only launch workstreams that the repository-owned dependency graph marks as safe.

If a workstream has a hard dependency, do not fake parallelism.

If a dependency is only soft, allow work that does not depend on the unfinished contract to proceed.

---

# 4. Parallel adapter development is an architecture test

If the adapter contract is marked stable, adapters should fan out independently.

Each adapter implementer should:

- consume the same stable semantic/adapter contract,
- work in its own branch/worktree,
- not modify another adapter,
- not use another adapter's generated output as truth,
- not add provider-specific fields to Core,
- declare capabilities explicitly,
- declare target ownership,
- preserve Observe / Plan / Apply / Verify boundaries where applicable,
- document evidence limitations,
- provide focused tests and negative controls.

If an adapter unexpectedly requires a Core or shared-contract change:

1. stop that specific adapter at the boundary,
2. record the concrete reason,
3. classify the request as:
   - generic missing contract,
   - provider-specific leakage,
   - test/infrastructure issue,
   - target-ownership conflict,
   - accidental central coupling,
4. do not let the adapter implementer patch Core independently,
5. review the request centrally,
6. make the smallest generic repair only if justified,
7. rebase/restart affected adapters from the corrected shared contract.

Do not allow one adapter's convenience to reshape Core.

---

# 5. Existing-project Init workstream

The Init implementer should own only the bootstrap/control-plane concern.

Expected responsibilities:

- safe existing-project initialization,
- preview-first behavior,
- discovery workspace,
- explicit evidence-scope configuration,
- snapshot preparation,
- privacy/scope review,
- shadow-mode preparation.

It must not:

- infer canonical engineering policy,
- auto-adopt discovered rules,
- rewrite existing project guidance,
- implement Copy Me semantics,
- introduce project-specific concepts into Core.

---

# 6. Copy Me workstream

The Copy Me implementer owns evidence interpretation and candidate-generation workflow.

Expected responsibilities:

- observations,
- hypotheses,
- supporting evidence,
- counterexamples,
- deduplication,
- conflicts,
- uncertainty,
- candidate model,
- human decision queue,
- explicit review/adoption workflow.

Preserve:

```text
Observed != Canonical
AI proposal != Authority
```

Copy Me should remain an authoring/discovery capability built on existing semantic foundations, not a second hidden policy engine.

If implementation pressures Core semantics, report the pressure instead of expanding Core automatically.

---

# 7. Risk / invariant triage workstream

This agent should first classify existing documented risks.

For each risk:

- ALREADY PROTECTED
- TEST HARDENING
- DOCUMENTATION ONLY
- REAL DEFECT
- DESIGN PROPOSAL
- NEEDS REAL-WORLD EVIDENCE

The triage agent should not treat each risk as a mandatory feature.

It may implement focused tests or fixes only when the workstream contract explicitly permits this and the change does not redefine shared semantics.

Any substantial Core change returns to coordinator review.

---

# 8. Konfyra adoption workstream

Treat Konfyra primarily as a real adopter / consumer.

It should begin as early as the dependency graph safely permits.

Even before full Init/Copy Me integration, it may usually perform bounded inventory/evidence work if that does not bind itself to unfinished APIs.

Expected responsibilities:

- inventory existing engineering knowledge,
- classify evidence sources,
- identify duplicate representations,
- identify conflicts,
- map existing docs / AGENTS / Claude / Codex / architecture tests / code evidence,
- prepare adoption coverage,
- exercise Init and Copy Me when those contracts become available,
- build a candidate engineering constitution,
- identify required adapters,
- report authoring/adoption friction.

It must not silently patch Core when Konfyra-specific behavior is awkward.

Instead produce:

```text
Gap
Evidence
Classification
Why current mechanisms are insufficient
Possible alternatives
Whether the need appears generic or adopter-specific
```

The coordinator decides whether platform work is warranted.

---

# 9. One branch/worktree per workstream

Use isolated branches/worktrees.

Suggested naming should follow repository conventions.

For example:

```text
codex/init-adoption
codex/copy-me
codex/risk-triage
codex/adapter-dotnet
codex/adapter-docs
codex/adapter-codex
codex/adapter-claude
codex/adapter-github
codex/adapter-azure-devops
codex/konfyra-adoption
```

No workstream should directly modify `main`.

---

# 10. Every implementer gets a bounded contract

Each dispatched agent must receive:

- objective,
- repository baseline,
- required canonical docs to inspect,
- owned subsystem,
- allowed changes,
- forbidden changes,
- dependencies,
- required tests,
- required evidence,
- exit criteria,
- completion questions.

Do not send a vague "implement adapter X" prompt.

Use the repository workstream specification as the primary task contract.

---

# 11. Do not allow agents to compete for shared semantics

If two agents both believe they own:

- normalized model types,
- adapter SPI,
- Core policy semantics,
- shared schema semantics,
- the same generated target,
- the same canonical documentation owner,

stop and resolve ownership centrally.

Parallel work is for independent execution, not duplicated authority.

---

# 12. Coordinator monitoring loop

While workstreams execute:

1. monitor progress,
2. answer only shared-contract questions centrally,
3. preserve workstream boundaries,
4. detect unexpected coupling early,
5. reject opportunistic scope growth,
6. allow agents to continue independently where possible,
7. keep a live integration/dependency status.

Maintain a concise coordination record such as:

```text
Workstream
Branch
Baseline
Status
Blocked by
Shared-contract request
PR
CI
Integration readiness
```

Use repository-owned tracking if an existing mechanism fits.

---

# 13. PR and integration policy

For each completed workstream:

```text
local validation
    ↓
focused PR
    ↓
required CI
    ↓
independent review
    ↓
coordinator architecture review
    ↓
integration
```

The coordinator should explicitly check:

- workstream stayed in scope,
- shared invariants remain intact,
- no provider leakage entered Core,
- no duplicate canonical owner was created,
- no adapter-to-adapter truth dependency appeared,
- tests prove the claimed behavior,
- docs describe actual behavior,
- evidence limits are honest.

---

# 14. Integration order

Integrate by actual dependency order, not completion time alone.

Typical rules:

- independent adapters may merge in any safe order if the shared contract is unchanged,
- contract changes merge before workstreams that depend on them,
- Init may merge independently from Copy Me if their interface remains stable,
- Copy Me should consume the agreed discovery/adoption boundary,
- Konfyra adoption should rebase onto the integrated capabilities it is validating,
- shared documentation consolidation may happen after an integration wave.

Do not create unnecessary dependencies between otherwise independent PRs.

---

# 15. Shared-contract change protocol

If a workstream discovers that the stable contract is insufficient:

1. do not let the workstream change it unilaterally,
2. produce a small design/change request,
3. show at least one concrete failing use case,
4. explain alternatives,
5. explain impact on other parallel workstreams,
6. decide centrally whether the change is generic,
7. if accepted, integrate the shared-contract change,
8. rebase affected workstreams.

This is especially important for:

- Semantic Model
- Domain semantics
- Adapter Contract
- policy evaluation
- impact semantics
- reconciliation semantics
- project/adoption state boundaries

---

# 16. Preserve the Core feature freeze unless evidence changes it

Do not opportunistically add:

- Pattern
- Trait
- inheritance
- general graph query language
- fan-out quantification
- arbitrary selectors
- provider-specific Core fields
- source-code semantic inference in Core
- background reconciliation operator

A parallel implementer may produce evidence for a future need.

Evidence is not automatic authority to implement it.

---

# 17. Keep real adoption feedback separate from platform authority

Konfyra and other adopter workstreams are valuable precisely because they pressure the platform.

Do not "make the adopter pass" by immediately teaching Core every project-specific concept.

When friction appears, first ask:

- bad modeling?
- missing authoring UX?
- missing adapter capability?
- missing discovery behavior?
- real generic semantic gap?
- adopter-specific policy?

Only the final generic cases should pressure Core semantics.

---

# 18. Test the modularity hypothesis

This first wave should itself answer:

### Adapters

- Were unrelated adapters actually implementable independently?
- Did any require another adapter's output?
- Did any need provider-specific Core changes?
- Was central registration a bottleneck?
- Did target ownership remain isolated?

### Init / Copy Me

- Could they progress independently behind their contract?
- Did either start implementing the other's responsibilities?

### Konfyra

- Could it act as a consumer without becoming a Core-development branch?

### Risks

- Could most risks be handled as tests/evidence rather than Core rewrites?

Record the results.

Parallelization is itself a product architecture experiment.

---

# 19. Do not publish releases during the active fan-out by default

Prefer:

```text
parallel wave
    ↓
integration
    ↓
stabilization
    ↓
cross-workstream validation
    ↓
release decision
```

Do not create a release merely because one adapter merged.

If a critical independent patch requires publication, handle it as a separate explicit decision.

---

# 20. First-wave completion condition

The first parallel wave is complete when:

- all launched workstreams are either integrated or explicitly closed/deferred,
- shared-contract requests are resolved,
- integration main is green,
- no known adapter coupling regression remains,
- cross-workstream tests pass,
- workstream docs/status are current,
- Konfyra/adoption evidence is preserved,
- remaining work is re-prioritized based on evidence.

---

# 21. Final coordinator report

At the end of the first wave, answer:

1. Which workstreams were launched concurrently?
2. Which completed independently?
3. Which became blocked by shared contracts?
4. Did adapter fan-out work as intended?
5. Which adapter, if any, required Core modification?
6. Did any adapter depend on another adapter output?
7. Which coupling findings were real architecture defects?
8. Did Init and Copy Me remain cleanly separated?
9. Could Konfyra behave as a consumer?
10. Which risks became concrete defects?
11. Which risks required only tests/docs?
12. What was the main merge/integration hotspot?
13. Did the repository-owned context prove sufficient for fresh implementers?
14. What should be changed before the second parallel wave?
15. Which workstreams are ready for the next wave?
16. Is the integrated result stable enough for a release candidate?

Do not publish a release automatically.

---

# Operating principle

The development process should now test Markitect's own architecture.

> **If independent subsystems cannot be developed independently, treat that as evidence.**

> **Centralize semantic authority. Decentralize bounded execution.**

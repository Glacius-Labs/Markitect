# Prompt — Prepare Markitect for Safe Parallel AI Development

Continue Markitect from the current main branch **after the currently assigned work has been completed and integrated**.

This iteration is NOT a feature-development iteration.

Its purpose is to prepare Markitect for:

> **SAFE PARALLEL DEVELOPMENT BY MULTIPLE INDEPENDENT AI IMPLEMENTERS**

Your role for this iteration is:

- Integration Coordinator
- Architecture Guardian
- Shared-Contract Owner
- Parallelization Auditor

The goal is to establish a stable baseline and enough durable repository context that future implementers do not need access to previous ChatGPT/Codex conversations in order to understand:

- what Markitect is becoming,
- what is already implemented,
- what is canonical,
- which architectural invariants must be preserved,
- which ideas are deliberate non-goals,
- which subsystem owns which responsibility,
- how new Domains, adapters and adoption features should integrate,
- which parts should be independently parallelizable,
- and what evidence is required before Core is expanded.

Do not implement the planned product workstreams yet.

Prepare the repository so those workstreams can be delegated safely.

---

## 1. Establish a clean baseline first

Before preparing parallel work:

1. Inspect current `main` and all recently integrated work.
2. Confirm there are no unfinished migrations, stale generated schemas, or half-adopted contracts.
3. Run the normal repository gates:
   - Go tests
   - vet
   - build
   - module verification
   - schema checks
   - executable examples
   - package/bootstrap checks
   - Windows/Linux CI where applicable
4. Check repository status and generated artifacts.
5. Ensure documentation describes actual source behavior rather than future assumptions.
6. Do not publish a new release merely for this preparation work.

At the end, identify one exact commit as:

```text
PARALLEL DEVELOPMENT BASELINE
```

Future workstreams should branch/worktree from this commit unless an explicit dependency requires a later integrated workstream.

---

## 2. Review the product direction

Inspect at least:

- README.md
- docs/architecture.md
- docs/engineering-constitution.md
- docs/implementation-plan.md
- docs/strategy/
- docs/design/
- docs/validation/
- CONTRIBUTING.md
- current Domain/schema implementation
- normalized model
- context
- impact
- PolicyResults
- policy exceptions
- reconciliation
- adapter contracts
- packages
- authoring resources
- Copy Me material
- existing-project adoption material
- any existing plugin/adapter documentation

The intended direction is:

> Markitect is a typed, deterministic engineering-knowledge and governance system.

It lets adopters define canonical engineering concepts, relationships, constraints, policies and processes.

Those semantics can be consumed by:

- humans,
- AI agents,
- documentation projections,
- deterministic checks,
- specialized adapters,
- reconciliation workflows.

Core should remain small and domain-neutral.

Guiding product statements include:

> AI should implement your architecture, not reinvent it.

> Markitect defines the grammar of valid engineering changes.

> One engineering truth, many technological projections.

Do not turn slogans into unsupported behavior.

---

## 3. Consolidate the Core invariants

Use the attached `01-core-invariants.md` as design input.

Create or update one concise, canonical repository document that future implementers can rely on.

Do not create duplicate competing constitutions if equivalent canonical documents already exist.

At minimum preserve:

- deterministic Core,
- syntax != semantic model,
- explicit semantic relationships,
- canonical/observed/generated/inferred/proposed state separation,
- generated outputs are not canonical,
- AI proposals require explicit adoption,
- domain meaning separate from technology adapters,
- adapters reconcile against canonical semantics, never one another,
- structural invalidity distinct from ordinary policy failure,
- read-only analysis must not weaken acceptance,
- Check/Verify remain strict,
- Apply remains explicit,
- evidence must say what it proves,
- exceptions remain narrow and source-bound,
- impact broadens conservatively under uncertainty,
- no new Core primitive without repeated concrete evidence,
- no accidental general query/policy language,
- no provider semantics in Core,
- no generic source-code semantics in Core,
- versioned architecture evolution remains explicit,
- benefit claims require evidence.

---

## 4. Treat parallelizability as an architecture quality property

This is important.

Do not assume sequential implementation is acceptable merely because coordination is convenient.

For each subsystem, ask:

> Can two independent implementers work from the same stable contract without editing each other's implementation or shared semantic code?

Classify every workstream into one of:

### Class A — should be strongly independently parallelizable

Examples:

- .NET adapter
- GitHub adapter
- Azure DevOps adapter
- Docs adapter
- Codex adapter
- Claude adapter

Expectation:

> Two unrelated adapters should be implementable concurrently from the same stable Semantic Model / Adapter Contract.

If this is not currently true, investigate why.

Possible causes:

- missing stable SPI,
- central registration hotspot,
- provider semantics leaking into Core,
- shared mutable output ownership,
- one adapter depending on another adapter's generated representation,
- test infrastructure coupling,
- hidden global configuration assumptions.

Treat such causes as architecture findings.

### Class B — conceptually coupled, but largely parallelizable behind explicit contracts

Examples:

- existing-project init
- Copy Me
- Konfyra adoption
- authoring UX

They may have dependency order, but their internal implementation should not require uncontrolled cross-editing.

### Class C — shared compiler/kernel semantics

Examples:

- normalized semantic IR
- policy semantics
- constraint language
- impact soundness
- Core state model
- reconciliation semantics

Here stronger coordination is normal.

Do not use Class C as an excuse for avoidable coupling in Class A/B.

---

## 5. Perform an explicit Parallelizability Audit

Create a durable report.

Suggested location:

```text
docs/development/parallelizability.md
```

For every major subsystem record:

- public/internal contract,
- implementation owner,
- shared dependencies,
- files/packages normally touched,
- whether another implementation can proceed independently,
- expected merge hotspots,
- whether Core changes are normally required,
- whether another adapter/module output is consumed,
- classification A/B/C,
- blocking modularity findings.

For adapters explicitly answer:

1. Can Adapter A and Adapter B be developed in parallel from the same baseline?
2. Do they consume the same normalized semantic model contract?
3. Does either require another adapter's output?
4. Does either require provider-specific fields in Core?
5. Does adding a new adapter require editing a central switch/registry?
6. Can the adapter be tested independently?
7. Can it declare capabilities independently?
8. Can it own its targets without overlap?

If the answers expose unnecessary coupling, propose the **smallest** modularity repair.

Do not build a large plugin framework merely because it is aesthetically appealing.

---

## 6. Adapter modularity is a strong invariant

Use the attached `04-adapter-modularity-contract.md`.

Target invariant:

> **Adding a new adapter must not require modification of another adapter or provider-specific modification of the semantic Core.**

And:

> **Two adapters targeting unrelated systems should be implementable concurrently from one stable semantic-model/adapter contract.**

Preferred dependency direction:

```text
                    Semantic Model
                         │
                  Adapter Contract
                         │
       ┌─────────────────┼──────────────────┐
       ▼                 ▼                  ▼
   .NET Adapter      GitHub Adapter    Azure Adapter
```

Forbidden/undesired patterns include:

```text
Claude Adapter
    ↓
Docs Adapter
    ↓
generated Markdown
```

or:

```text
Azure Adapter requires Core.Resource.AzureAreaPath
GitHub Adapter requires Core.Resource.GitHubRepositoryId
```

Provider mappings belong in adapter/configuration boundaries.

If a stable adapter contract already exists, freeze/document it and fan out.

If not, identify and implement only the minimum contract repair needed to make fan-out safe.

Do not build the adapters themselves in this iteration.

---

## 7. Create a parallel development guide

Create a repository-owned guide for future AI implementers.

Suggested location:

```text
docs/development/parallel-work.md
```

It should explain:

- coordinator role,
- implementer role,
- branch/worktree expectations,
- integration flow,
- shared-contract ownership,
- Core-change escalation,
- testing requirements,
- evidence requirements,
- release boundaries,
- parallelizability expectations.

Recommended flow:

```text
Implementer
    ↓
local validation
    ↓
focused PR
    ↓
CI
    ↓
independent review
    ↓
Coordinator checks shared-contract compatibility
    ↓
integration
```

Agents must not independently redefine shared semantic contracts.

---

## 8. Define workstream ownership

Prepare implementer-ready work packages.

Do NOT implement them in this iteration.

At minimum:

### A. Existing-project init / adoption bootstrap

Owns:

- safe init flow for mature repositories,
- discovery workspace,
- evidence-scope configuration,
- explicit source selection,
- snapshot preparation,
- privacy/scope review,
- shadow-mode bootstrap.

Must NOT:

- infer active rules during init,
- rewrite existing AGENTS/Claude/Codex/docs,
- auto-adopt policy,
- add project-specific architecture semantics to Core.

### B. Copy Me / engineering discovery

Owns:

- observations,
- hypotheses,
- supporting evidence,
- counterexamples,
- deduplication,
- conflict detection,
- legacy/current distinction,
- uncertainty,
- candidate model,
- human decision queue,
- explicit adoption workflow.

Must preserve:

```text
Observed != Canonical
AI proposal != Authority
```

### C. Risk / invariant triage

For each documented risk classify:

- ALREADY PROTECTED
- TEST HARDENING
- DOCUMENTATION ONLY
- REAL DEFECT
- DESIGN PROPOSAL
- NEEDS REAL-WORLD EVIDENCE

Do not assume each risk implies implementation.

### D. Adapter contract / modularity

Purpose:

- confirm/freeze the shared adapter contract,
- remove only blocking coupling if needed,
- make independent adapter fan-out safe.

### E. Individual adapters

Prepare separate work packages for likely adapters:

- .NET
- Documentation
- Codex
- Claude
- GitHub
- Azure DevOps

Each should define:

- exact purpose,
- semantic concepts consumed,
- capabilities,
- target ownership,
- evidence boundaries,
- tests,
- explicit non-goals.

### F. Konfyra adoption

Treat Konfyra as a real consumer.

Owns:

- inventory existing engineering knowledge,
- existing-project initialization,
- evidence selection,
- Copy Me discovery,
- candidate engineering constitution,
- conflict/decision queue,
- migration coverage,
- shadow mode,
- adapter requirements,
- authoring friction,
- adoption report.

When it finds a gap:

```text
record
classify
provide evidence
```

It must not immediately change Core.

---

## 9. Define the Init / Copy Me / Konfyra boundary

Use attached adoption documents.

### Existing-project init owns

```text
control-plane setup
discovery workspace
evidence-scope configuration
snapshot preparation
safe preview/write behavior
```

### Copy Me owns

```text
evidence interpretation
candidate engineering facts
deduplication
conflicts
uncertainty
decision queue
candidate/adoption workflow
```

### Konfyra adoption owns

```text
real-world consumption
migration coverage
feedback/evidence
```

Do not allow three different implementations of discovery semantics.

---

## 10. Define the dependency graph

Produce explicit:

- hard dependencies,
- soft dependencies,
- independent workstreams.

Expected shape:

```text
Parallel-safe baseline
        │
        ├── Risk triage
        │
        ├── Existing-project init
        │       │
        │       └── Copy Me
        │               │
        │               └── Konfyra adoption
        │
        └── Adapter contract/modularity audit
                │
                ├── .NET
                ├── Docs
                ├── Codex
                ├── Claude
                ├── GitHub
                └── Azure DevOps
```

But do not force this graph if the actual repository supports more parallelism.

If Copy Me can begin independently from a stable candidate/evidence contract, say so.

If Konfyra inventory can begin immediately, say so.

If all adapters can start immediately from the current contract, say so.

Parallelize as aggressively as architecture safely allows.

---

## 11. Identify shared hot spots

Identify likely merge-conflict areas:

- README.md
- docs/architecture.md
- docs/implementation-plan.md
- CLI registration
- schema generation
- normalized model types
- adapter interfaces
- shared test helpers

For each, determine whether it is:

- a legitimate shared contract,
- or avoidable central coupling.

Avoidable registration/switch hotspots should be flagged.

Do not automatically refactor them unless they actually block safe fan-out.

---

## 12. Prepare implementer-ready work packages

Each future workstream specification must contain:

- Context
- Objective
- Scope
- Current relevant implementation
- Owned subsystem
- Allowed changes
- Forbidden changes
- Dependencies
- Required design questions
- Required tests
- Required evidence
- Exit criteria
- Completion questions

Suggested directory:

```text
docs/workstreams/
```

Possible files:

```text
README.md
init-existing-project.md
copy-me.md
risk-triage.md
adapter-contract.md
adapter-dotnet.md
adapter-docs.md
adapter-codex.md
adapter-claude.md
adapter-github.md
adapter-azure-devops.md
konfyra-adoption.md
```

Use repository conventions if a better canonical location exists.

Fresh implementers should not require previous chat history.

---

## 13. Do not implement the workstreams yet

This preparation iteration is successful if it produces a safe parallelization baseline.

Do not opportunistically implement:

- Copy Me expansion,
- existing-project init,
- Konfyra migration,
- new adapters,
- new Domain operators,
- Pattern/Trait,
- inheritance,
- graph query DSL,
- source-code analysis,
- background operator.

Exception:

If the Parallelizability Audit exposes a **small, concrete, blocking modularity defect** that prevents obvious Class-A fan-out, you may propose and implement the minimal repair needed to establish a stable contract.

If you do that:

- document the defect first,
- keep the repair minimal,
- do not expand product capability,
- prove the adapters can then fan out independently.

---

## 14. Validation

Before completion:

- run full normal repository gates,
- verify generated files are current,
- ensure workstream docs do not contradict one another,
- ensure references resolve,
- verify two agents cannot both believe they own the same semantic contract,
- verify current release/source status remains accurate,
- inspect repository for accidental local/private paths,
- verify adapters do not depend on each other's projections,
- verify no workstream spec silently assumes a future Core primitive.

---

## 15. Final output

At completion report:

1. Exact parallel-development baseline commit.
2. Current release versus current source status.
3. Which shared contracts are frozen enough for parallel work.
4. Which contracts still require coordinator ownership.
5. Workstream list.
6. Dependency graph.
7. Which workstreams can start immediately.
8. Which must wait.
9. Main merge-conflict hotspots.
10. Any unresolved architecture decision that makes parallel work unsafe.
11. Whether adapter fan-out is safe **right now**.
12. If not, the concrete modularity defect preventing it.
13. Whether adapter development requires touching Core.
14. Whether any adapter currently depends on another adapter output.
15. Whether Init and Copy Me are sufficiently separated.
16. Whether Konfyra inventory can begin immediately.
17. Whether any documented risk currently requires a blocking Core fix.
18. Recommended first parallel wave.
19. Recommended integration order.
20. Whether the repository itself now contains enough durable context for a fresh implementer.

Do not publish a release automatically.

The goal is:

> **MAKE THE REPOSITORY ITSELF THE SOURCE OF CONTEXT REQUIRED FOR SAFE PARALLEL AI DEVELOPMENT.**

And use parallelizability as an architectural test:

> **If unrelated adapters cannot evolve independently, explain and fix the coupling rather than normalizing sequential development.**

Implementation organization should mirror the product philosophy:

> **Centralize semantic authority. Decentralize bounded execution.**

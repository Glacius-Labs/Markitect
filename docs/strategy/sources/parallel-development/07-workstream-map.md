# Markitect — Suggested Parallel Workstream Map

## Coordinator

Owns:

```text
shared semantic contracts
Core invariants
integration
cross-workstream decisions
release readiness
```

Does not implement every workstream.

---

# Workstream A — Existing-Project Init

Goal:

Prepare mature repositories for adoption.

Owns:

```text
control-plane setup
discovery workspace
evidence scope
snapshot preparation
safe preview/write flow
```

Should be largely independent from Copy Me internals.

---

# Workstream B — Copy Me

Goal:

Turn selected evidence into reviewable candidate engineering knowledge.

Owns:

```text
observations
hypotheses
support/counterevidence
deduplication
conflicts
uncertainty
candidate model
decision queue
adoption proposal
```

Must not auto-adopt.

---

# Workstream C — Risk Triage

Goal:

Review known risk register.

Output classification:

```text
ALREADY PROTECTED
TEST HARDENING
DOCUMENTATION ONLY
REAL DEFECT
DESIGN PROPOSAL
NEEDS REAL-WORLD EVIDENCE
```

Should mostly be analysis/tests, not uncontrolled Core refactors.

---

# Workstream D — Adapter Contract / Parallelizability

Goal:

Prove adapters can fan out independently.

Owns:

```text
adapter SPI/contract
capabilities
target ownership
Observe/Plan/Apply/Verify semantics
provenance/evidence contract
parallelizability audit
```

If current contract is already adequate, this workstream should be small and mostly confirm/freeze it.

---

# Workstream E — .NET Adapter

Goal:

Technology-specific .NET/MSBuild evidence/projection according to stable adapter contract.

Must not add .NET semantics to Core.

---

# Workstream F — Documentation Adapter

Goal:

Human documentation/projection behavior.

Must consume canonical semantic model directly.

Must not become the source for other adapters.

---

# Workstream G — Codex Adapter

Goal:

Codex-specific projections/instructions.

Must not consume Docs output as canonical truth.

---

# Workstream H — Claude Adapter

Goal:

Claude-specific projections/instructions.

Must consume canonical semantics independently.

---

# Workstream I — GitHub Adapter

Goal:

GitHub-specific observed/projected state as justified.

Provider mapping remains adapter-owned.

---

# Workstream J — Azure DevOps Adapter

Goal:

Azure DevOps Areas/work-items/process integration where justified.

No Azure-specific fields in semantic Core merely for convenience.

---

# Workstream K — Konfyra Adoption

Goal:

Use Markitect against a real mature project.

Owns:

```text
inventory
evidence mapping
candidate constitution
decision queue
coverage
shadow mode
adoption friction
adapter gaps
report
```

Must not self-authorize Core changes.

---

# Dependency guidance

Possible graph:

```text
Parallel baseline
   │
   ├── Risk Triage
   │
   ├── Init ───────┐
   │               ├── Copy Me ─────┐
   │               │                │
   │               └────────────────┼── Konfyra Adoption
   │                                │
   └── Adapter Contract ────────────┼── .NET
                                    ├── Docs
                                    ├── Codex
                                    ├── Claude
                                    ├── GitHub
                                    └── Azure DevOps
```

But the coordinator should maximize safe concurrency.

Examples:

- Konfyra inventory can start before Init/Copy Me implementation is complete.
- Copy Me data-model work may begin before CLI integration if contracts are clear.
- Adapter agents should start immediately if the adapter contract is already stable.
- Do not invent artificial sequencing.

---

# Completion rule for each agent

Every agent returns:

```text
what changed
why
owned scope
what was not changed
tests
evidence
remaining gaps
integration dependencies
Core changes requested, if any
```

Any requested Core change must be justified as generic evidence, not local convenience.

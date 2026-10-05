# Coordinator Supplement — Desired-State-First Changes + Mandatory Completion Gate

Use this as an additional directive during the current autonomous A/B gauntlet.

## Critical experiment boundary

Do **not** contaminate the currently frozen development cohort.

The product candidate currently under scored development evaluation must remain unchanged until the protocol reaches its explicit **global fix / no-fix decision**.

Therefore:

1. preserve all current scored development runs and exclusions exactly;
2. finish the feasible development assessment;
3. classify the observed failures;
4. treat the mechanism below as a **candidate generic product fix / operating-model improvement**;
5. implement it only after the frozen protocol permits a candidate change;
6. bind the changed product to a new exact candidate identity;
7. test it first with focused regressions;
8. then use the reserved holdout according to the already frozen protocol.

Do not retroactively rescore the original development runs as if the new mechanism had existed.

---

## 1. Product hypothesis

The recent DDD observations suggest that agent instructions alone are insufficient.

An agent may correctly implement code while failing to update the canonical Markitect model.

Use two complementary enforcement directions:

```text
FORWARD CONTROL
Markitect desired state
    ↓
Impact / expected affected surface
    ↓
Implementation

BACKWARD VERIFICATION
Actual candidate change
    ↓
Completion Gate
    ↓
Does implementation correspond to declared intent?
```

Principle:

> **Markitect should not rely on an agent remembering every governance step. The system should make an incomplete or unexplained change unable to become green.**

---

## 2. Every task starts through Markitect

Do not interpret this as "every task must mutate Markitect."

Every task begins with Markitect preflight and is classified into:

```text
IMPLEMENTATION_ONLY
ENGINEERING_INTENT_CHANGE
POLICY / ARCHITECTURE_MIGRATION
REQUIRES_OWNER_DECISION
```

### Implementation-only

Examples:
- bug fix inside an existing UseCase;
- algorithm change;
- test expansion;
- implementation refactor inside existing modeled boundaries.

Flow:

```text
fixed BASE
→ Markitect context
→ identify governing Resources / policies / managed artifacts
→ create bounded Change Plan
→ implement
→ completion gate
```

No fake semantic edit.

### Engineering-intent change

Examples:
- new UseCase;
- new Handler/Validator relationship;
- new Module;
- changed ownership/dependency/process;
- new required artifact;
- changed agent rule;
- changed pipeline/hook contract.

Flow:

```text
fixed BASE
→ update canonical Markitect desired state FIRST
→ validate
→ policy-failure analysis if migration is intentionally intermediate
→ Impact
→ create bounded Change Plan
→ implementation
→ completion gate
```

### Requires-owner-decision

Stop before choosing architecture on the owner's behalf.

The agent may prepare alternatives, affected Resources, consequences and required evidence, but may not silently change desired state.

---

## 3. Introduce an explicit Change Plan

Evaluate and implement the cleanest form of an **input-bound Change Plan**.

This should normally be a Host / Module / workflow concept, **not a new Domain Kind or Core primitive**.

It is a transient execution contract between accepted desired engineering state and candidate implementation.

Bind at least:

```text
BASE source revision
Project/config identity
normalized model digest
task/change identity
change classification
changed/selected canonical Resources
Impact result identity
expected affected Resources
expected / allowed artifact surface
required checks
required projections / reconciliation targets
declared exceptions if already owner-approved
```

Keep arbitrary LLM reasoning out of the trust identity.

---

## 4. Desired state first for semantic changes

For `ENGINEERING_INTENT_CHANGE` and migrations:

Do not begin implementation from architecture described only in prose.

Desired semantic state must first exist canonically.

Then Markitect should produce something like:

```text
Changed intent:
  UseCase/CreateOrder
  Validator/CreateOrderValidator

Affected:
  Orders module
  CreateOrder Handler
  architecture policy
  selected artifacts
  provider projections
```

Only then implement.

---

## 5. Mandatory Completion Gate

Create one authoritative candidate verification operation/workflow.

Exact command naming is a design decision, conceptually:

```text
markitect verify-change BASE..CANDIDATE
```

or equivalent.

Question answered:

> **Is the actual candidate change a valid and sufficiently accounted implementation of the declared engineering change?**

Compose existing capabilities rather than duplicating them.

### A. Semantic validity

- canonical model structurally valid;
- applicable policies pass or have explicit valid treatment;
- no hidden auto-waiver;
- migration state explicit.

### B. Desired-state / implementation alignment

Where implementation evidence exists:

- required technical evidence corresponds to declared owners/relations;
- new architecture-relevant implementation is reflected in accepted desired state;
- removed/renamed implementation leaves no stale canonical ownership;
- contradictory observed evidence blocks completion.

Target example:

```text
Code:
CreateOrderValidator exists

Canonical model:
CreateOrder.validators = []

Result:
FAIL — implementation evidence and desired engineering state are inconsistent.
```

Do not move source-language semantics into Core. Use Modules/project-owned checks.

### C. Managed artifact accounting

Every changed file in managed roots must be:

```text
canonical Markitect source
declared project artifact input
generated output with known owner
tool-owned artifact
explicitly excluded
```

otherwise fail.

Also detect stale declared inputs, deleted inputs, ownerless generated output, collisions and stale provider output.

### D. Change-plan surface

Compare actual diff with plan:

```text
EXPECTED
ALLOWED_SUPPORTING_CHANGE
GENERATED
EXPLICITLY_EXCLUDED
UNEXPLAINED
```

`UNEXPLAINED` should fail or require explicit replanning / owner decision.

Do not pretend Impact is perfectly precise; distinguish conservative supporting scope from truly unexplained change.

### E. Required checks

Require relevant evidence such as:

```text
project tests
architecture tests
Artifact Coverage
Agent Rules
Git Hooks
Pipelines
.NET / implementation evidence
projection consistency
reconciliation verify
```

A green semantic model alone is not completion.

---

## 6. Gate levels

Use the same semantics at multiple points.

### Agent/local fast gate
Fast, actionable, suitable for self-repair.

### Pre-push gate
Broader:
- desired state;
- impact/change-plan consistency;
- artifact coverage;
- module checks;
- projection drift;
- architecture checks.

### CI gate
Authoritative fixed BASE/CANDIDATE verification:
- full required checks;
- change-plan verification;
- reconciliation verify;
- configured Module evidence.

CI must not trust a mutable plan rewritten after implementation without proper rebinding.

---

## 7. Prevent retroactive architecture justification

Critical failure mode:

```text
agent implements arbitrary code
→ gate fails
→ agent edits Markitect afterward merely to bless it
```

For engineering-intent changes, bind the plan to the **pre-implementation desired-state revision/digest**.

If canonical engineering state changes after implementation began, report:

```text
STALE_PLAN
```

and require a new explicit planning cycle.

History should show:

```text
desired state
→ implementation
```

not silently:

```text
implementation
→ retroactive architecture justification
```

Implementation-only work binds to the unchanged accepted model at BASE.

---

## 8. Preserve PR #77 architecture

Keep:

```text
CLI → Host → independent Modules → Core
```

and:

```text
Core→Module = 0
Core→Host = 0
Module→Module = 0
Module→Host = 0
```

The Completion Gate should be Host composition over independent evidence providers.

If a cohesive `ChangeControl` / `ChangeGate` Module is useful, it may consume Core IR plus explicit Host-supplied diff/evidence facts.

It must not import sibling Modules.

Host composes Artifact Coverage, Agent Rules, Pipelines, etc.

---

## 9. Reuse existing Modules

Preferred composition:

```text
Host
├── Core semantic/model validation
├── Change Plan / gate orchestration
├── Artifact Coverage
├── Agent Rules
├── Git Hooks
├── Pipelines
├── .NET / implementation evidence
├── Markdown
└── configured project checks/adapters
```

Each Module returns bounded evidence.

Host decides overall gate state.

---

## 10. Explicit evidence states

Do not collapse everything to pass/fail.

Support distinctions such as:

```text
PASS
FAIL
INCOMPLETE
UNKNOWN
WAIVED
REQUIRES_OWNER_DECISION
STALE_PLAN
```

Examples:
- analyzer unavailable → `INCOMPLETE`;
- unmanaged file → `FAIL`;
- owner-decision task → `REQUIRES_OWNER_DECISION`;
- desired model changed after plan → `STALE_PLAN`.

---

## 11. Instructions become secondary to enforcement

Keep the Markitect-first Workflow as guidance.

But safety must not be:

> "The agent was told to update Markitect first."

It must be:

> "If the agent fails to keep intent and implementation aligned, the task cannot pass the Completion Gate."

This directly addresses the observed DDD drift.

---

## 12. Add regressions from current gauntlet findings

After the global development fix/no-fix decision permits product changes, add deterministic cases for:

1. Validator exists in code but no canonical Validator ownership → fail.
2. New Query + Handler exist without canonical owners → fail.
3. Handler renamed in code but model retains old identity → fail.
4. Implementation-only bug fix inside already-accounted artifact → pass without fake model edit.
5. Unmanaged file → fail.
6. Stale generated projection → fail.
7. Cross-module shortcut requiring owner decision → `REQUIRES_OWNER_DECISION`.
8. Model changes after implementation began under Plan P1 → `STALE_PLAN`.

---

## 13. Keep Impact honest

Impact provides:

```text
expected semantic change surface
declared dependent resources
conservative affected resources
causes
```

Artifact Coverage and implementation Modules provide observed technical evidence.

Keep them separate.

Do not silently narrow unknowns to make the gate look precise.

---

## 14. Holdout strategy

When development runs complete:

1. make global evidence-based `FIX` or `NO-FIX` decision;
2. if `NO-FIX`, run holdout with original candidate;
3. if `FIX`, implement this only to the extent justified by development evidence;
4. freeze a new candidate binary/digest;
5. do not alter holdout tasks/oracles;
6. run reserved holdout;
7. compare original development behavior, changed-candidate holdout and A baseline holdout.

Do not claim improvement if only the benchmark harness changed.

---

## 15. Success criterion

The previously observed failure class:

```text
Code correct but Markitect stale
```

should become:

```text
Code correct
Markitect stale
→ Gate red
→ agent self-repairs
→ only then complete
```

while ordinary implementation-only work remains cheap:

```text
Bug fix
→ no pointless YAML churn
→ Gate green
```

---

## 16. Record failure honestly

Record if:

- legitimate implementation changes are frequently unexplained;
- the plan becomes manual bookkeeping;
- Impact is too broad;
- agents edit Markitect merely to satisfy gates;
- gates duplicate architecture tests without removing coordination;
- plan staleness creates excessive friction;
- Modules would need sibling dependencies.

Do not weaken the experiment to make the mechanism win.

---

## 17. Deliverables

After eligible implementation, produce:

1. design/decision document;
2. trust/lifecycle model;
3. clean Host/Module placement;
4. CLI/workflow surface;
5. deterministic plan schema if persisted;
6. fast local gate;
7. fixed-snapshot CI gate;
8. regression fixtures from gauntlet failure classes;
9. Markitect self-dogfood;
10. human/agent UX docs;
11. new candidate binary/digest;
12. holdout evidence under unchanged tasks/oracles.

---

## Product statement

> **Markitect defines desired engineering state before implementation when engineering intent changes.**

> **Agents implement inside that state.**

> **A mandatory completion gate compares the resulting repository against declared intent and refuses unexplained drift.**

> **Humans are required for architecture decisions, not for reminding agents to keep model, code, rules and generated surfaces synchronized.**

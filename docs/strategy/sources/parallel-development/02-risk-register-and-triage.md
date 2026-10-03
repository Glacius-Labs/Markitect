# Markitect — Risk Register and Triage Guidance

This document summarizes the main known architectural/product risks and how parallel agents should treat them.

The purpose is **not** to create one feature branch per risk.

Each risk should first be classified:

```text
ALREADY PROTECTED
TEST HARDENING
DOCUMENTATION ONLY
REAL DEFECT
DESIGN PROPOSAL
NEEDS REAL-WORLD EVIDENCE
```

Only concrete defects/hardening gaps should automatically become implementation tasks.

---

## 1. Kernel too weak vs too powerful

Risk:

- too weak → endless special cases and new Kinds/operators,
- too powerful → Markitect becomes a general programming/policy language.

Potential treatment:

- keep a small finite semantic kernel,
- require repeated pressure before new primitives,
- use specialist engines/adapters where richer semantics are justified.

---

## 2. Hidden semantic relationships

Risk:

If dependencies are implicit, then:

```text
impact
context
reconciliation
explainability
```

become incomplete.

Potential treatment:

- explicit typed relations,
- provenance,
- relation-specific context/invalidation semantics,
- conservative fallback when dependencies are unknown.

---

## 3. Policy composition ambiguity

Risk:

Multiple scopes/packages may disagree:

```text
Company requires A
Architecture package requires B
Project forbids B
Exception waives A
```

Potential treatment:

- deterministic composition,
- explicit conflicts,
- no "last YAML wins",
- explicit waiver/override semantics,
- explainable effective policy.

Do not invent composition semantics without evidence.

---

## 4. State conflation

Risk:

Mixing:

```text
canonical
observed
generated
inferred
proposed
```

creates silent authority changes.

Potential treatment:

- explicit state separation,
- adoption boundary,
- projections never canonical,
- observed state never writes desired state automatically.

---

## 5. Reconciliation non-convergence

Risk:

Adapters oscillate or repeatedly produce operations.

Potential treatment:

- deterministic Observe/Plan,
- explicit Apply,
- Verify,
- plan binding,
- idempotency/convergence tests,
- second-plan-empty style evidence where appropriate.

---

## 6. Impact false negatives

Risk:

Missing an affected resource is more dangerous than over-inclusion.

Potential treatment:

- explicit dependency semantics,
- conservative broadening under uncertainty,
- structured causes,
- separate direct subjects from conservative review set,
- real adopter measurement of noise.

---

## 7. Plugin/adapter leakage into Core

Risk:

Technology-specific semantics enter canonical resources.

Bad examples:

```text
Core.Resource.AzureAreaPath
Core.Resource.GitHubRepositoryId
```

Potential treatment:

- adapter mappings,
- anti-corruption boundary,
- domain meaning remains provider-neutral.

---

## 8. Adapter coupling

Risk:

One adapter consumes another adapter's generated output or requires shared provider-specific changes.

Bad pattern:

```text
Claude adapter
    ↓
Docs adapter
    ↓
generated Markdown
```

Potential treatment:

- stable semantic model contract,
- stable adapter protocol,
- independent target ownership,
- capability declaration,
- direct canonical consumption.

Parallelizability should expose this risk.

---

## 9. Model/package evolution

Risk:

A new version silently reinterprets old meaning.

Potential treatment:

- exact package pins,
- immutable historical versions,
- explicit adoption,
- impact,
- PolicyResult changes,
- stale exceptions,
- migration evidence.

---

## 10. Explainability/provenance

Risk:

A result cannot answer "why?".

Potential treatment:

Every important result should identify:

```text
subject
rule
Domain
package/source
constraint
relation/path
evidence
cause
```

without exposing irrelevant implementation internals.

---

## 11. Constraint-language creep

Risk:

Small operators evolve into JSONPath/CEL/Rego-like arbitrary logic.

Potential treatment:

- finite operators,
- bounded semantics,
- no arbitrary expressions,
- evaluate CUE/OPA or specialist checks when repeated richer cases appear.

---

## 12. Model vs reality

Risk:

A structurally correct canonical model is mistaken for implementation truth.

Potential treatment:

Distinguish evidence types.

Example:

```text
canonical dependency
observed ProjectReference
runtime dependency
business behavior
```

are separate claims.

---

## 13. Architecture fossilization

Risk:

Strong architecture rules prevent legitimate evolution.

Potential treatment:

- versioned architecture packages,
- explicit architecture changes,
- impact,
- exceptions,
- migration workflow,
- human decision remains authoritative.

---

## 14. Plugin trust/security

Risk:

Apply-capable adapters have excessive authority.

Potential treatment:

```text
Observe = read-only
Plan = pure/read-only
Apply = explicit mutation
Verify = read-only
```

plus least privilege, exact plan identity, audit evidence.

---

## 15. Authoring burden

Risk:

Maintaining Domains/resources/packages costs more than the duplication they replace.

Potential treatment:

- real adopter studies,
- authoring UX,
- discovery/bootstrap,
- generated views,
- avoid manual duplicate facts,
- compare against simpler alternatives such as AGENTS.md + architecture tests.

---

## 16. Context quality

Risk:

Compiled context is:

```text
too broad
missing important narrative
larger than simple alternatives
```

Potential treatment:

- explicit context relations,
- source-selection review,
- inclusion provenance,
- measure required/missing/irrelevant content,
- do not assume smaller == better.

---

## 17. Adoption completeness

Risk:

Existing projects forget rules during migration.

Potential treatment:

- repository inventory,
- evidence-backed Copy Me,
- conflict detection,
- migration coverage report,
- shadow mode,
- explicit cutover.

---

## 18. Parallel implementation pressure

Risk:

Markitect itself becomes too coupled to develop in parallel.

Interpretation:

This is especially concerning for adapters.

Potential treatment:

- parallelizability audit,
- stable semantic model,
- stable adapter contract,
- remove provider-specific central hotspots,
- coordinator only owns genuinely shared semantics.

A subsystem that cannot be parallelized should explain whether the dependency is:

```text
legitimate shared kernel semantics
or
avoidable modularity failure
```

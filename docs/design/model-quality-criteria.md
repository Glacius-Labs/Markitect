# Canonical model quality criteria v0.2

**Status:** working version v0.2, 26 October 2026, of workstream W3 of the [canonical model track](model-track-charter.md), after an independent review. It is the checkable side of the [modeling guide](modeling-guide.md). Thresholds are provisional until the pilots (W6).

**Basis:**
- the code analogy of the [founding intent](../concepts/sources/founding-intent-20261026.md);
- [ENH-002](../concepts/register.md#enh-002-model-quality-linter-and-metrics): findings and recommendations, not hard gates by default;
- [DEC-005](../concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility): granularity is the author's choice;
- [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) and [DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate): what the model leaves open is a realization choice that parents settle.

## How the criteria apply

| Where | Hard rules (H) | Review criteria (R) | Signals (S) |
|---|---|---|---|
| Our reference and evaluation models (the self-model, pilots, study portfolios) | Must pass | Must pass | Reviewed; each one kept is justified in one line |
| Other projects | Reported by the W5 tool as findings | Reported | Reported |

- The product does not block on these criteria (ENH-002, DEC-005). For now they live in the W5 tool. Warnings in `markitect check` would change the product CLI and need an owner decision first.
- They are not the material-gap criteria of the mandatory Model assessment, which stays as accepted on 21 October 2026.

## Hard rules: mechanical

| # | Rule | Check | Code analogy | Anti-pattern |
|---|---|---|---|---|
| H1 | **Every owner decision is stated once and reaches each consumer.** A decision that several Packages follow sits in their common parent or a shared Package, and reaches every Manager that needs it by a typed reference, a body link or modeled scope | Tool: a Definition whose text uses a term owned by another Package without an edge to it; the same normative sentence in two Definitions. Review: for each cross-cutting decision, how it reaches each consumer | Single source of truth | **Unreached rule**, **Copied rule** |
| H2 | **No Package cycles.** The Package graph is acyclic; an edge is any typed reference or body link (D3) | Tool: strongly connected components | Go forbids import cycles | **Big ball of mud** |
| H3 | **Every Package is a module.** Before realization, each Package with Definitions has an ownership area or has children; after realization, its area holds registered files. Packages that only define Kinds and Properties are exempt | Tool: `execution.yaml` ownership plus the registry shards | No empty namespaces | **Topic folder** |
| H4 | **Shapes are typed.** Arguments, codes, fields, results and scopes are typed fields of a Kind, not Markdown lists in the body | Tool hint: body patterns such as "Arguments:", "Refusals:", "Fields:"; confirmed in review | Use structs, not comments | **Prose compensation** |

## Review criteria

| # | Criterion | Check | Anti-pattern |
|---|---|---|---|
| R1 | **Coverage.** Project, values, domain, behavior, policies, architecture and way of working are each modeled. Architecture is the owner's decision, encoded as the model's structure (R5); anything left open is stated as free by the owner's explicit choice | Review against the guide's concern table | **Requirements-only model** |
| R2 | **Only owner decisions.** Nothing in the model is a decision the owner did not make or approve. Cross-Package choices the owner left open are stated as free, not invented | Review against the owner's sources or approved model diffs | **Invented decision** |
| R3 | **No process, evidence or restatement.** No Manager process, no run or test results, no consequence restated from another rule | Review | **Process prose**, **Restated rule** |
| R5 | **One architecture.** The module structure is the Package tree with its ownership areas; architecture Definitions only add dependency rules, interfaces and placement. No Definition describes a component structure parallel to the Package tree | Review; tool hint: Definitions of a Component-like Kind, or names that mirror Packages | **Shadow architecture** |
| R4 | **Agrees with the owner's decisions.** Where the realization differs from the model, that is a realization defect or an unrecorded decision to add to the model (DEC-063), never a silent model edit | Review of model against code and decisions | **Stale model** |

## Signals

| # | Signal | Check | Default (provisional) | Code analogy | Anti-pattern |
|---|---|---|---|---|---|
| S1 | **Size** | Words of prose per content Definition; tables exempt | Warning above 150, strong above 300 | Long functions | **God Definition** |
| S2 | **Missing types** | Backticked names used in three or more Packages that are neither a Definition ID, a field name, nor an enum or wire value | Any occurrence | Primitive obsession | **Stringly typed** |
| S3 | **Dependency direction** | Edges from a more stable Package to a less stable one (instability = outgoing / (incoming + outgoing)); violations of declared DependencyRules | Any declared-rule violation; direction warnings above an instability difference of 0.3 | Stable dependencies principle, Clean Architecture's dependency rule | **Inverted policy** |
| S4 | **Change locality** | Between two `check` reports: Packages touched by a change that touches no Policy Definition (a change that does touch one counts as cross-cutting) | Above 2 | Shotgun surgery | **Scattered change** |
| S5 | **Peer coupling** | Share of edges between peer feature Packages; edges to the kernel and shared policies are excluded | Report the value; compare in W6 | Low coupling, high cohesion | **Leaky modules** |
| S6 | **Redundancy** | Near-duplicate sentences across Definitions | Threshold set in W6 | DRY | **Restated rule** |
| S7 | **Unused structure** | Required or project-declared typed fields that no Definition sets (optional fields of Library Kinds excluded); unused imports | Any occurrence | Dead fields, unused imports | **Speculative type** |
| S8 | **Copied Library Kind** | A project Kind whose fields include a Library Kind's without `extends`, unless it tightens a cardinality or changes an object shape | Any occurrence | Copy-paste inheritance | **Forked type** |
| S9 | **Isolated Definition** | A content Definition with no edge other than its `kind`; the project Package is exempt | Any occurrence | Dead code | **Orphan** |
| S10 | **Open cross-Package choice** | A choice two or more Packages must agree on is neither decided nor stated as free | Review hint | — | **Silent gap** |

## Later research, not v0.2 rules

- **CRAP-style risk:** branching in a Definition multiplied by missing verification coverage. It needs tests bound to statements ([ENH-003](../concepts/register.md#enh-003-test-to-statement-binding-and-test-review)).
- **Model mutation:** change one statement and check whether any registered test or oracle case can tell the old realization from the new. It needs Apply runs or a statement-to-test binding.
- **Verifiability:** every Rule states how a violation would be noticed. A review hint now; mechanical once ENH-003 exists.

## Measured today (to be re-run with the W5 tool)

| | Makerspace c9 | Self-model |
|---|---|---|
| H1 | No unreached owner decision found yet; tool pending | Coverage gap rather than H1: Project, Workflow and Harness lack Definitions (R1) |
| H2 | Fails: all 15 Packages in one link cycle | Passes trivially: no edges |
| H4 | Fails: all 38 operations list their arguments and refusals in prose | Partly: Checks are typed, Rules are prose |
| R1 | Fails: no architecture (left to the implementer, with no owner decision) and almost no way of working | Partly |
| S1 | About 5 over 300 words of prose (computed-values 886, audit 453, utilization-report 383, invoice 371, makerspace 354) | 5 over 300; the largest has 793 |
| S2 | To be re-measured with the stated method; a plain count of backticked names finds 57, mostly roles, refusal codes and states | Not measured |
| S4 | C1–C8: 2, 7, 1, 4, 6, 3, 8, 3 Packages (no Policy Kind yet, so no step counts as cross-cutting) | 13 of 18 commits touched one Package |
| S9 | 0 | All 26 content Definitions |

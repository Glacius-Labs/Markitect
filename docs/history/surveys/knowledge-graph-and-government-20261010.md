# Survey: knowledge graph and Government

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for KG-01 to KG-03, GOV-01, GOV-02 and IDEA-01. Decisions: register DEC-018 (knowledge graph) and DEC-019 (Government).

## Knowledge graph

**Idea:** The YAML model is already a typed graph:
- a Manager owns;
- a Statement `uses` and `requires`;
- an Artifact `realizes`;
- a Check `checks`.

The knowledge graph makes this queryable and explainable: who depends on this and why, which decision supports a rule, which run led to an Apply. YAML stays canonical. RDF, SHACL and OWL were assessed and rejected as the canonical format.

**Earlier work:** `codex/knowledge-graph` (`ab405644`).
- Based on 9 October, in the old `internal/` layout.
- About 4.4k production lines of Go, 3.8k test lines and 43k lines of committed evidence logs.
- New packages: `projectknowledge`, `projectgraph`, `knowledgeevidence`, `knowledgeprotocol`.
- Changes to `projectmodel`: first-class Decisions and `IdentityChange`.
- CLI: `project knowledge --knowledge-action graph|relations|explain|trace|history|coverage`, plus a separate `project knowledge-mcp`.
- Example: `examples/project-knowledge`.
- Results:
  - Focused tests and process journeys passed with no provider calls.
  - Full Verify never passed.
  - A dry-run merge onto current main gave 24 conflicts, 9 of them from the layout move.

**Independent assessment:** In `archive/wip/government-assessment-untracked-docs-20261010` it concludes "not merge-ready":
- The Manager view drops list edges (`projectgraph/service.go`).
- There are two privacy leaks around `IdentityChange`.
- Decision cycle findings are non-deterministic.
- Results are stale by default.
- It creates a second visibility owner next to Context.
- The branch carries clutter.

The pure core, `projectknowledge` (about 670 lines), is sound. The accompanying refinement plan turns the knowledge graph into an explanation of impact. That matches DEC-018.

**Inputs for KG-01:**
- The twelve knowledge questions become table tests.
- Check impact for order dependence and lost deltas. The suspected cause is in `projectmodel/impact.go`: queue construction iterates a Go map, and statements reached through `uses` do not get coverage.
- Results must be identical under permutation and with `-count=200`.

## Government

**Branch:** `codex/government-implementation-20261010` (`99769916`).
- Based on `f12ffb00`. The committed part is documentation only: 32 German files under `docs/design/government-concept/*`, `docs/research/government-evaluation/*` and `docs/work-items/government-implementation/handoff.md`.
- The work in progress is in `archive/wip/government-implementation-20261010`. It touches `projectmodel/{analyze,projectmodel,types}.go`.
  - It does not compile: `ResolveGovernment` is undefined, and one assignment is invalid.
  - It would change every project's report digest.

**v1 contract scope:**
- An optional `Manager.spec.government` on the root: owner principal, reviewer Manager, attempt limit, reserved statements.
- `spec.modelChangeScope` per Manager.
- The only delegated change: editing the description of a delegated `concept` or `use-case` statement.
- An independent reviewer runs as a separate execution.
- Policy is compiled from the previously accepted model.
- Sequence: decision, reserved write, commit, receipt, cursor.
- A cooperative owner channel and revocation by epoch.
- No courts in v1.

**Planned packages:** changes to `projectmodel` and `projectwork`, new `governmentstate` and `government` packages, and gates on every entry point (`projectwork.ApplyEdit`, `projectapp.Readiness`, and Plan, Run, Resume, Repair, Verify, Preflight, Apply and Deliver in `projectrun`).

**Older code:** `codex/government-p1` holds the G1–G5 implementation (about 23k lines, old layout). Reference only.

**Designs found elsewhere:**

| Area | Status | Location |
|---|---|---|
| Courts and appeal | concept only | `legislation-and-courts.md` |
| Precedents | concept only; advisory, binding only after canonical adoption | Government concept GD13; register IDEA-005 |
| Briefings | concept: cursor briefings and immediate notices | `operations.md`. Main already has `brief`, `briefings` and `dismiss` for Manager model-change briefings. |
| Monitoring and audit | concept only | `operations.md` |
| Cockpit, visualization, gamification, Kubernetes/etcd, shared events | no design | owner notes in `docs/concepts/sources/concepts-chat-20261009/` |

## Collision zones

- **model zone:** `projectmodel` (analyze, types, report digest), touched by both KG and Government. DEC-019 sequences them.
- **runtime zone:** the Government gate points in `projectrun`.
- **interface zone:** the knowledge-graph CLI verbs and the Government briefings.
- **Tests:** both lines need deterministic invariant tests and a Full Verify that finishes within its budget.

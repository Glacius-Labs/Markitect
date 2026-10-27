# Lane 8: the model compiler

**Status:** state page, 27 October 2026. Taking stock; nothing is built until the owner says so.

**Goal:** `markitect check` works like a compiler for a typed language: it catches every structural error early and says exactly where, and it never judges meaning ([DEC-003](../concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept), [founding intent](../vision.md#founding-intent)). Around it, people and agents get the everyday tools of a language: find references, dependency graph, impact of a change and lint.

**Scope:** the read-only model commands `check`, `context`, `docs` and the proposed `tree`; their report contract; [diagnostics](../design/diagnostic-contract.md); per-Manager [context selection](../design/manager-context.md); dependency locks; and the [analysis tool](../../tools/model-analysis/README.md) under `tools/model-analysis/`. The compiler is shared by every variant and lives on `main`.

**Lanes** (the owner's split of 27 October 2026): 1 host as coordinator, 2 Claude Code plugin, 3 agents collaborate through Git, 4 evaluation suite, 5 Windows/Linux execution, 6 model language, 7 modeling skills and model design, 8 this page, 9 agentic commands, 10 continuous change, 11 Cockpit/IDE.

**Not in scope:**
- What the language means: the [language contract](../design/language-contract.md), Core and the Libraries belong to lane 6. This lane enforces what lane 6 decides.
- How models are written: the modeling skill belongs to lane 7. It is this lane's main user.
- Commands that start Managers, such as an agentic impact check, belong to lane 9. A model view in the UI belongs to lane 11. Both would read this lane's outputs.

Language semantics, the compiler, snapshot values, organization derivation and accounting are a protected foundation (concept §5.5). Changing them needs an explicit owner decision.

## What exists

| Area | State |
|---|---|
| `check` | Structural check of the whole input in stages: model, execution declarations, registry, file accounting (every file covered or ignored, stale registry links listed) and context selection. Deterministic JSON report; exit 0 passed, 1 known failure, 2 usage, 3 incomplete observation. About 15,700 lines of Go in `internal/model` |
| Diagnostics | About 130 stable codes in families (`name.*`, `contract.*`, `lock.*`, `yaml.*`, `binding.*`, `dependency.*`, `value.*`, `limit.*` and others), each with source file and line/column of the token start. The report carries all of them; its `error` field names only the first |
| Diagnostic coordinates | Exact byte and YAML ranges for body links are a [separate tool profile](../design/diagnostic-contract.md). The Go compiler does not claim it: positions point at a token start, not at an exact link span |
| `context <package>` | What one Manager receives and why: own Definitions, See also, type contracts with inclusion reasons, and a compact index of its child areas ([PR #754](https://github.com/Glacius-Labs/Markitect/pull/754)) |
| `docs` | Preview or guarded write of a generated model view |
| `tree` | The model-derived Manager tree with parent, children and ownership areas. Exists only on the plugin variant branch (`variant/plugin`, commit `b1320b31`), not on `main` |
| Dependencies and lock | Every dependency is passed by path with a repeated `--dependency`; there is no discovery. The entry model needs a sibling `model-lock.yaml`; `check` fails with `lock.missing` or a digest mismatch otherwise. Nothing in the product writes the lock |
| Analysis tool | `tools/model-analysis/model_analysis.py`, Python standard library only, reads the `check` JSON: `diagnostics` (all findings, root causes first), `summary`, `graph` (Package edges, cycles, coupling), `refs`, `impact` (two reports compared, optionally transitive), `lint` (warnings, never failures) and `lock` (writes the lock). Built for W5 of the [model-track charter](../design/model-track-charter.md); tested in CI; not part of the product or its release bundle |

## Gaps

1. **A project cannot finish a model with the product alone.** Writing the lock and reading all diagnostics need the Python tool, which ships only in the Markitect source. The modeling skill points agents at `tools/model-analysis/`.
2. **Failures are hard to read.** `check` prints JSON only and its `error` names the first problem; the skill has to tell agents to run a second tool.
3. **Dependencies are typed out every time.** Each call repeats every Library manifest by path, although the lock already names the models.
4. **The graph is computed twice.** Go context selection and the Python tool each derive edges from the model (D3: any typed reference or body link). The skill, the Cockpit (lane 11) and an impact command (lane 9) all need the same answers, and two implementations can drift.
5. **Impact is offline only.** It needs two saved reports; there is no "compare with this commit". How [DEC-023](../concepts/register.md#dec-023-impact-marks-change-and-context-without-dropping-either) maps to language v1 is still open ([OQ-009](../concepts/register.md#open-questions)).
6. **`tree` is not on `main`**, although it is read-only and every variant can use it.
7. **No exact spans.** Without the coordinates profile, an editor or the Cockpit cannot underline a broken link precisely.
8. **What counts as an error lives in prose.** The skill tells agents to treat `unknown-package`, `dependency-rule`, `interface`, `import-cycle` and every strong lint finding as errors; no tool contract says so. Lint thresholds are provisional ([OQ-002](../concepts/register.md#open-questions)).

## Candidate features

In a suggested order: small ones that remove the Python dependency first, then one shared analysis core.

| # | Feature | What it gives | Size | Touches the protected compiler? |
|---|---|---|---|---|
| 1 | `markitect lock` | Writes or updates `model-lock.yaml` from the checked dependencies | Small | No: CLI over existing results |
| 2 | Readable `check` output | A text mode that lists every diagnostic, root causes first, beside the JSON | Small | No |
| 3 | `tree` on `main` | Moves the plugin branch's command to the shared product | Small | No |
| 4 | Analysis core in Go | One package over the checked model for edges, `graph`, `refs` and `impact`, with a stable JSON contract; `impact --base REV` reuses `--revision` capture. Replaces the Python logic so the skill, Cockpit and lane 9 read one answer | Medium | No: reads checked values; new package and architecture-table row |
| 5 | `markitect lint` | ENH-002 signals as warnings, exit 0 by default; a strict switch for our own reference and evaluation models, as the charter asks | Medium | No |
| 6 | Dependency discovery | Find dependencies from the lock or a vendored `libraries/` folder instead of repeated `--dependency` | Small to medium | Needs lane 6 agreement on manifest and lock meaning |
| 7 | Exact spans | Implement the coordinates profile for body links | Medium | No new semantics; the profile is a tool contract |
| Later | Observed code dependencies widen impact ([ENH-001](../concepts/register.md#enh-001-observed-code-dependencies-widen-impact)); test-to-statement signals ([ENH-003](../concepts/register.md#enh-003-test-to-statement-binding-and-test-review)) | | | |

Impact stays information. It never narrows which Managers take part ([DEC-039](../concepts/register.md#dec-039-deterministic-manager-process)).

## Open questions for the owner

1. **Promote the analysis into the product?** Recommendation: yes, in the order above. Features 1–3 first; then the Go analysis core, after which the Python tool is retired.
2. **Who owns `tools/model-analysis/`** (lane 6's question 6)? Recommendation: lane 8, because it checks models; lane 7 uses it.
3. **May product lint know the standard Libraries?** `dependency-rule` and `interface` read the `markitect/architecture` Kinds by name, but the compiler gives Libraries no semantics. Recommendation: lint may know standard Libraries and stays warnings; the compiler stays Library-blind.
4. **Should any lint signal become a compiler error?** `unknown-package` looks structural (a name that does not resolve). Recommendation: only through a language change in lane 6, for example a value type for Package references; then it is an ordinary type error and needs no special case.
5. **What needs your explicit decision?** Recommendation: changes to `internal/model` semantics and diagnostics do; new read-only commands and the analysis package over checked values follow the normal PR rules.

## Boundaries with other lanes

- **Lane 6:** owns what is valid; this lane implements and reports it. Any new rule, value type or lock meaning is agreed there first.
- **Lane 7:** the skill is the first consumer of features 1, 2 and 5; once they exist, the skill drops its Python steps.
- **Lane 9:** an agentic impact check would start from feature 4's impact result.
- **Lane 11:** the Cockpit's map, Definition reader, impact, Manager tree and lint views (lane 11's state page, "What understanding a model needs") need features 3–5 and 7 as stable JSON.
- **Lane 2:** `tree` moves from its branch to `main` (feature 3); the variant then merges `main`.

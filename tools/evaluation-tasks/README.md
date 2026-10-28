# Evaluation tasks

The study models of the Library and the Makerspace in the [concept 2.0](../../docs/concepts/v2-concept.md) form, one complete snapshot per step of each change series. They are the model input of the [evaluation suite](../../docs/guides/evaluation-suite.md)'s change-series tasks (step T6 of the [concept 2.0 build](../../docs/implementation-plan.md#concept-20-build)). The concept 1.0 originals stay where they are, frozen: [testdata/library](../../testdata/library/README.md), [comparison/phase-b](../comparison/README.md) and [comparison/makerspace-v2](../comparison/makerspace-v2/README.md). Hidden judges and requirements are unchanged.

## Layout

Each step folder is the content of a target project's `.markitect/` folder, as the [model language guide](../../docs/guides/model-language.md) describes it:

```text
<project>/<step>/
  manifest.yaml     std 1.0.0, nothing else
  lock.yaml         written by markitect lock
  execution.yaml    the required checks: go build and go test in realization/
  model/            the Root Package
```

No step ships realization, so no step has registry pages: each Package's Manager writes its own page when it creates files.

| Project | Steps | Change |
|---|---|---|
| `library` | `m1` | The small library: members, copies, loans, returns and overdue notices |
| | `m2` | Loan limit 5 → 3 |
| | `m3` | Short-loan copies with a 7-day period |
| | `m4` | One renewal per loan |
| | `m5` | No member suspension; `setMemberStatus` removed |
| `makerspace` | `w0` | The fresh build: ten modules, 37 operations |
| | `c1` to `c9` | The Phase C backlog: `forbidden` refusals audited, lockouts, bulk discount, the steward role, account blocks, certification changes, the billing rules restructured, opening-hours change; `c9` changes no model content and equals `c8` |

The steps follow their sources one to one: Library `m1` is [testdata/library/m1](../../testdata/library/m1/model.yaml), `m2` its loan-limit edit and `m3` to `m5` the [Phase B overlays](../comparison/phase-b); Makerspace `w0` to `c9` are [makerspace-v2](../comparison/makerspace-v2/README.md) v2.1, `model/` and `c1/model/` to `c9/model/`.

## How the conversion reads the old models

Both conversions keep every statement and rewrite only form. The equivalence records hold the independent reviews: [Library](../../docs/validation/t2-library-equivalence-20261028.md) and [Makerspace](../../docs/validation/t2-makerspace-equivalence-20261028.md).

- **Kinds.** Concepts, Rules, Features and the project anchor use `std`. Neither project has a REST or CLI interface: the Library's JSON Lines interface stays one Root Rule with an operations table, and each Feature names its operation. The Makerspace defines two Kinds of its own at Root, `Operation` (one protocol operation with wire name, arguments, refusals, result, permissions and audit target) and `Interface` (what other modules may use of one module).
- **No typed fields.** Typed values became prose: the Library's loan policy holds its limit and periods in one table; the Makerspace's records list their fields, its operations their arguments and results. Value types and enumerations used in one place are written where they are used; the shared formats are one Root Concept, [values](makerspace/w0/model/values.md), and the shared refusal codes another.
- **Root is the project.** Everything that used to apply to the whole project lives in Root: the Library's `library` Package; the Makerspace's project Package, its `policies` Package (shared formats, roles, refusal codes, the audit log and the project-wide rules) and the project-wide protocol rules `order-of-checks` and `argument-values`. Root may not link down into a Package, so it names Packages instead (`[[lending]]`). The Makerspace's `policies` Go package stays a module of the [architecture](makerspace/w0/model/architecture.md), owned by the project itself.
- **Scope is location.** A rule that used to reach other modules through `appliesTo` now lives in the module it belongs to and is linked from each module it binds. Where that link would close a cycle (bookings' reservation and lockout rules for equipment and safety), the consuming module states its need itself, as before.
- **No cycles.** Links point up or sideways. Library `members` and `catalog` no longer link `lending`: eligibility and availability state the rule for lending in prose, and lending links them.
- **Visibility.** A Makerspace Definition is internal unless its module's interface lists it, a rule of it binds another module, or another module links it.

## Check

Copy a step into a Git repository's `.markitect/` folder and run `markitect lock` and `markitect check` there. Every step compiles with no error.

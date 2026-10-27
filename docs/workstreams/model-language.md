# Lane 6: the canonical model as a language

**Status:** state page, 27 October 2026. Taking stock; nothing is built until the owner says so.

**Goal:** the canonical model is typed programming in prose ([founding intent](../vision.md#founding-intent)). Projects define their own Kinds, Libraries supply common ones, the compiler checks structure only, and a model is judged like code.

**Scope:** the language ([language contract](../design/language-contract.md), [Kind interfaces](../design/kind-interfaces.md)), the [standard Libraries](../design/standard-libraries.md) in `libraries/`, the [quality criteria](../design/model-quality-criteria.md) as a definition of a good model, and the [model-track charter](../design/model-track-charter.md).

**Lanes** (the owner's split of 27 October 2026): 1 host as coordinator, 2 Claude Code plugin, 3 agents collaborate through Git, 4 evaluation suite, 5 Windows/Linux execution, 6 this page, 7 modeling skills and model design, 8 compiler, 9 agentic commands, 10 continuous change, 11 Cockpit/IDE.

**Not in scope:**
- How people and agents write and change models: the modeling skill and the [modeling guide](../design/modeling-guide.md) belong to lane 7.
- What the compiler checks belongs to lane 8. A change in the language contract that the compiler must enforce is agreed with lane 8.
- Comparison suites belong to the evaluation lane (lane 4).

Changes to the language contract, Core, compiler and snapshot values touch the protected foundation (concept §5.5) and need an explicit owner decision.

## State

| Area | State |
|---|---|
| Language | Language contract v1 with Core 1.2.0: Kind interfaces (`implements`) instead of inheritance, and `valueType` for a field's type ([DEC-075](../concepts/register.md#dec-075-kind-extension-and-one-core-120), [PR #757](https://github.com/Glacius-Labs/Markitect/pull/757)) |
| Libraries | `values`, `practice`, `data`, `api` and `architecture`, all 1.0.0, in `libraries/`; outline v0.3.1 accepted. `software-basics` 0.3.0 stays only for older fixtures |
| Quality criteria | v0.2: hard rules for our own reference and evaluation models, warnings in the product. Thresholds are provisional |
| Pilot model | `makerspace-v2` (W0 and change steps C1–C9) in `tools/comparison/makerspace-v2/`, with v2.1 fixing a missing type in the equipment Interface ([PR #779](https://github.com/Glacius-Labs/Markitect/pull/779)) |
| Old-versus-new gate | Done on Codex Sol High, one run per attempt ([record](../validation/v1-makerspace-gate-20261027.md)). The old model published W0, C1 and C2. The new model published W0 on its second attempt (the first hit a since-fixed publication defect), C1 at once, and C2 only on the v2.1 repeat after a model blocker. Every published step passed the hidden oracle. The new model used 12 Managers instead of 21; the cost comparison is in the record. A pilot, not a study |
| Working notes | The previous session's drafts are on branch `handover/model-quality-log` (folder `model-quality/`). They are unreviewed and do not belong on `main` |

## Decisions in force

- **Dependency edge (D3):** for analysis and linting, any typed reference or body link is a dependency; imports only resolve names. For semantics, a link alone still declares no dependency.
- **Package visibility** is a Library convention checked by a linter first; a core rule follows only on evidence.
- **Architecture lives in the model:** the Package tree, dependency rules, Interfaces and placement, with no second description beside it.
- **Evaluation models:** the makerspace old-versus-new gate stays on Codex Sol High; later comparison suites run on Codex Luna ([source](../concepts/sources/owner-morning-decisions-20261027.md)).
- **Impact check** waits until the evaluation shows the full run is better (the owner's lane assignment of 27 October; not yet recorded in the register).
- **Self-model pilot** is deferred until Markitect is stable.

## Decisions taken under delegation

On 27 October the owner let the previous session accept its own recommendations ("auto accept your recommendations") and asked to review them later. The [decision log](https://github.com/Glacius-Labs/Markitect/blob/handover/model-quality-log/model-quality/decision-log-20261027.md) has the reasons and how to undo each one. Grouped by the lane that now owns it:

| # | Decision | Owner lane |
|---|---|---|
| 8 | When requirements leave the order of refusals open, the listed order is an owner decision | 6 |
| 9 | makerspace-v2.1: the equipment Interface exposes `rate-cents-per-minute` instead of a separate rate type for bookings | 6 |
| 4, 10 | The analysis tool reads the model's own architecture; a new `interface` signal checks typed uses only, not body links | 7 or 8 (open, question 2 below) |
| 1, 3, 5, 6, 7 | The modeling skill: built in parallel with the gate, lives in `packaging/skills/markitect-modeling/`, trialled twice on neutral library requirements | 7 |
| 2 | Claude adapter after the Linux runner; V1 acceptance stays Codex-only | 1 and 5 |

## Owner decision: the five Library questions (27 October 2026)

The owner, in this lane's session: "Accept the recommendations on the five Library questions: unordered-refusal flag, refusals in Interface, invariant placement via an interface, no envelope Kind yet, no alias-only Core change yet. Prepare the plan, build nothing yet."

### Plan (not started)

One PR, owned by this lane. No Core or compiler change, so lane 8 is not involved.

| Change | Library | Version |
|---|---|---|
| New optional boolean Property `unorderedRefusals`. When true, any refusal that applies may be returned, because the requirements fix no precedence. The bodies of `refusals` and `Operation` say so. A project Operation Kind adds the Property only when it needs it | `api` | 1.1.0 |
| New optional Property `refusalCodes` on `Interface`, a list of references to `Api.RefusalCode`; the body names refusal codes as part of the public surface | `architecture` | 1.1.0, on `api` 1.1.0 |
| `Invariant` implements `Practice.Rule` and declares the optional `appliesTo`. A Placement's `concern` stays `Practice.Rule` and now accepts an Invariant. An invariant is a binding statement, so it is a Rule | `data` | 1.1.0, now depends on `practice` 1.0.0 |

All three are minor versions: existing models stay valid. Putting the flag into the `Operation` interface itself would make `api` 2.0.0 and force every project Operation Kind to change, for a field few operations need.

**In the same PR:** the lock files, the version pins in `internal/bootstrap/libraries_test.go`, the [standard Libraries](../design/standard-libraries.md) page (Library sections and the answered-questions table), and a verbatim source record of the decision under `docs/concepts/sources/`.

**Not changed:** `makerspace-v2` keeps its vendored 1.0.0 copies; it is frozen gate material. A v2.2 that uses the new fields would come only when the evaluation lane asks for it.

**Follow-ups for other lanes, after the merge:**
- Lane 7: update the modeling skill's Library reference and its rule that the listed refusal order is an owner decision (decision 8 above).
- The analysis tool's owner (question 2 below): count refusal codes in an Interface as typed uses in the `interface` signal.
- Coordinator: a register line under DEC-074.

**Unchanged by decision:** no wire-envelope Kind, and imports stay visible without their alias. Both wait until a real model needs them.

## Open questions for the owner

1. **Small expressiveness gaps** (maps, unions, defaults, uniqueness, formats, ranges, constraints across fields). The unreviewed W1 draft recommends no Core change: Library patterns, and a revisit only if a real model is harmed. The owner's "fix all of that" after W1 may have meant these too; this needs a clear answer.
2. **Who owns the analysis tool** (`tools/model-analysis/`): lane 7 (it supports writing models) or lane 8 (it checks models).

## Candidate ideas

- **Review intent per Package** (from a session's reading of Anthropic's [code-review article](https://claude.dev/blog/solving-code-review-at-anthropic/)): their reviewers are steered by short hand-kept files per code owner (what failure looks like here, what to skip) and by risk tiers per path. A Package in the model could carry its risk level and a "what failure looks like" statement, so review focus and approval tiers come from the model instead of a second source. Open: a Library Kind or nothing; not a Core change.

## Housekeeping for the Coordinator

- The [DEC-075](../concepts/register.md#dec-075-kind-extension-and-one-core-120) status still says "Not yet in source"; it merged in PR #757.
- The delivery plan's W1 row still waits for a scratch `markitect check` after the K-S/K-R probe; that is likely stale.

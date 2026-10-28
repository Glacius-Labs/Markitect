# T2: Makerspace models, old and new equivalent

**Verdict:** the ten Makerspace steps in the [concept 2.0](../concepts/v2-concept.md) form say the same as their concept 1.0 originals, after three small fixes and four wording fixes that were applied to every step. Step T2 of the [concept 2.0 build](../implementation-plan.md#concept-20-build).

| | |
|---|---|
| Date | 28 October 2026 |
| Old | [makerspace-v2](../../tools/comparison/makerspace-v2/README.md) v2.1: `model/` (W0, 181 Definitions) and `c1/model/` to `c9/model/` |
| New | [tools/evaluation-tasks/makerspace](../../tools/evaluation-tasks/README.md) `w0` (108 Definitions) to `c9` |
| Reviewers | Two independent Claude Opus 5.5 subagents that did not write the conversion: one walked every old W0 file against the new model, the other compared each old change step's diff with the new one and checked the 2.0 structure of every step by script |
| Hidden material | The Makerspace oracle, judges and requirements are unchanged |

## Result

| Step | Change | Verdict |
|---|---|---|
| `w0` | fresh build | equivalent after fixes 1 to 3 |
| `c1` | `forbidden` refusals audited | equivalent |
| `c2` | lockouts | equivalent after fix 4 |
| `c3` | bulk discount | equivalent |
| `c4` | the steward role | equivalent |
| `c5` | account blocks | equivalent |
| `c6` | certification changes | equivalent |
| `c7` | billing rules restructured | equivalent |
| `c8` | opening hours and granularity | equivalent |
| `c9` | no model change | equivalent; identical to `c8`, as in the old series |

All 37 operations of W0, its 20 records, 25 refusal codes, every enumeration value, every range and every rule text are present with the same meaning; so are the project's language, module root, build and test commands and the old ownership and checks. In every step, every link resolves, Root links no Definition inside a Package, no internal Definition is linked from outside its Package, there is no Package cycle, and the links between Packages follow the architecture's dependency table.

## Fixes

All fixes are made by the conversion, so they hold in every step.

1. **`listAuditEntries` reachable** (meaning): protocol's [requests](../../tools/evaluation-tasks/makerspace/w0/model/protocol/requests.md) listed the module interfaces as the operations; it now also names [list-audit-entries](../../tools/evaluation-tasks/makerspace/w0/model/list-audit-entries.md), which sits in Root.
2. **Order of checks placed in protocol** (meaning): `order-of-checks` and `argument-values` moved from protocol to Root, which the architecture makes the `policies` module. The [architecture](../../tools/evaluation-tasks/makerspace/w0/model/architecture.md) now says protocol realizes the whole order of checks, including checks 1 and 5, and Root holds these rules only so every module reads them.
3. **Interfaces reach their users** (meaning): every module used to see every interface. Each interface now links the interfaces of the modules its module may use, and billing's invoice building and bookings' check-out link [tier](../../tools/evaluation-tasks/makerspace/w0/model/members/tier.md), which holds the fee and the discount.
4. **Lockouts outlive incidents** (wording, C2 on): safety cannot link bookings' lockout rule without a cycle, so [incidents](../../tools/evaluation-tasks/makerspace/c2/model/safety/incidents.md) now states the part it needs: resolving an incident does not end its lockouts.
5. **Wording:** the `Interface` Kind admits public rules that bind other modules; [schedule maintenance](../../tools/evaluation-tasks/makerspace/w0/model/equipment/schedule-maintenance.md) states what a missed reservation is and that overlap is judged on the reserved interval; the machine rate reads "cents per minute"; the reservation's `status` links the reservation rules again.

## How the form changed, with the same information

- **Root is the project.** The project Package `makerspace` and the `policies` Package became Root: project anchor, [architecture](../../tools/evaluation-tasks/makerspace/w0/model/architecture.md) (dependency rules, placements and the old ownership), test and documentation rules, the project-wide rules, [values](../../tools/evaluation-tasks/makerspace/w0/model/values.md), [refusal codes](../../tools/evaluation-tasks/makerspace/w0/model/refusal-codes.md), roles in [authorization](../../tools/evaluation-tasks/makerspace/w0/model/authorization.md), and the audit log. The Go package `realization/policies` stays a module, owned by the project itself.
- **Two project Kinds** in Root replace the typed operation and interface Kinds: `Operation` lists wire name, arguments, refusals, result, permissions and audit target; `Interface` lists what other modules may use.
- **No typed fields.** Enumerations and value types used in one place are written where they are used; refusal codes sit in Root or in their module's interface. That is why W0 has 108 Definitions instead of 181.
- **Scope is location.** Rules that reached other modules through `appliesTo` are linked from those modules. Two links would close a cycle; there the consuming module states its need (fix 4, and equipment's existing statement of what an active reservation is). Three reaches changed without changing behavior: `argument-values` now reaches every module; opening hours and booking granularity live in bookings, which realizes them; equipment, safety, reporting and protocol no longer receive the membership rule, which constrains none of their operations.
- **Visibility** follows the old interfaces: a Definition is internal unless its interface lists it, it is a rule that binds another module, or another module links it.
- **Extra edits in the change steps** come only from the conversion: inline enumerations gain new values where the old enumeration did (C1, C4, C6), new refusal codes become rows of their interface's table (C2, C5), and C7's renamed billing rules change the links of their consumers in bookings, training, materials and reporting. `tests` does not change in C7, because Root names the billing Package rather than a rule inside it.

## Check on T1

Every step compiles with no error on T1's branch `dev/awesome-chaum-554c4d` at `e3c922ee`, with the lock that branch's `markitect lock` writes. To be confirmed on `main` once T1 merges.

# Proposal: what `vet` measures

**Status:** a proposal for the owner, written on 7 November 2026. Nothing on this page is decided or built. It answers part of [OQ-002](register.md#open-questions) and the question [concept 4.0 §4](v4-concept.md#4-is-the-spec-good) leaves open: the details of `vet`, its further measures and thresholds.

**Basis:** the [measurements of 7 November 2026](../validation/spec-design-measures-20261107.md) on the two archived Makerspace specs, 115 and 419 Definitions, and the [guide to a good spec](../guides/good-spec.md), which says what each measure is for. [The model language guide](../guides/model-language.md#vet-warnings) owns what `vet` does today, and changes with it once the owner decides.

## In brief

- **Replace `vet.reach` with `vet.hub`,** which counts Root and leaves Kinds out. On both specs it warns about `values` and nothing else; `vet.reach` warns about three Interfaces, which are linked widely by design, and misses `values`.
- **Keep the other warnings,** and give `vet.unstable_dependency` a margin of 0.1, so that one Package every module links does not shift instabilities into new warnings.
- **Add a stricter tier,** printed only with `--strict`: statements per Definition, Package cohesion, placement, and a CRAP analogue that reads `review.sum`.
- **No layer in Kinds.** The tree and the stability measure already carry the Dependency Rule, and on both specs a declared layer would have found nothing more.

## The measures

N is the number of the spec's model Definitions. Fan-in counts the Definitions that link to a Definition or are of it as Kind. Hits are on Makerspace c8 and the 5× project's c8, as the archive holds them.

### Warnings

`vet` prints these always and exits 1 on them only with `--strict`, as today.

| Code | Warns when | Threshold | Hits: Makerspace, 5× | Change |
|---|---|---|---|---|
| `vet.hub` | A Definition other than a Kind, Root included, whose change alone puts more than a quarter of the spec back on the list: fan-in + 1 > max(10, 0.25 N) | 25 %, at least 10 | `values` (61 of 115); `values` (270 of 419) | New, replaces `vet.reach` |
| `vet.coupling` | A Package depends on more than 5 Packages (Ce) | 5 | `protocol` (8); 6 Packages, `bookings` 12 at most | Keep |
| `vet.unstable_dependency` | A Package depends on one whose instability Ce / (Ca + Ce) is higher by more than the margin | margin 0.1 | none; 2 (differences 0.12 and 0.18) | Add the margin; today 0 and 3 |
| `vet.definition_size` | A Definition's body has more than 1000 words | 1000 | none (largest 430); none (largest 995) | Keep |
| `vet.package_size` | A Package has more than 25 Definitions of its own, Root included | 25 | none (Root 21); none | Keep |
| `vet.thin_package` | A Package has no Definitions of its own and one child | — | none; none | Keep |
| `vet.no_paths` | A Package other than Root claims no path pattern | — | as today | Keep |

**Why `vet.hub` and not `vet.reach`.** `vet.reach` counts the Packages a change reaches and skips Root. On Makerspace it warns about the Interfaces of `billing`, `equipment` and `members`, whose purpose is to be used by other modules, and stays silent about `values`, which 60 Definitions link while using a median of 2 of its 13 formats. `vet.hub` counts Definitions, which is what `status` lists, and counts Root, where the worst hub sits. Kinds are left out because a change to a Kind truly concerns all its Definitions. At 20 % instead of 25 % it would also warn about `members/member` (25 of 115, 91 of 419), whose linkers mostly name a member by identifier only; 25 % leaves that to the author. After `values` is split it warns about `values/identifier` alone, a true hub, which the author keeps with one line of reason, as the modeling skill asks of any warning.

**Why the margin.** Splitting `values` into a Package gave every module one more efferent edge and shifted instabilities, which raised three new `vet.unstable_dependency` warnings with differences of 0.04, 0.06 and 0.10. The warnings that stood before differ by 0.10 to 0.18. A margin of 0.1 keeps the larger ones and drops the shifts. It is provisional, like every threshold here.

### Only with `--strict`

With `--strict`, `vet` also runs these, prints them, and exits 1 on any warning of either tier. They are stricter because each has a known blind spot, or no calibration yet.

| Code | Warns when | Threshold | Hits: Makerspace, 5× | Why only strict |
|---|---|---|---|---|
| `vet.statements` | A Definition has more than 20 statements: list items and table rows in its body | 20 | 2 (`reporting/revenue-report` 25, `bookings/create-reservation` 22); 8 (`list-order` 67 at most) | The anchor and the architecture are long by nature (5× `architecture` 60, `project` 26) |
| `vet.cohesion` | A Package falls into two or more groups of two or more Definitions each, with no link between the groups | — | none; none | No hit on either spec, so untested against a real split |
| `vet.placement` | Every Definition that links to a Definition lies below it in one child Package, so it could move down to that child | — | none; none | A Rule that binds by scope may be linked from one child only and still belong higher. Both specs are one level deep, so untested |
| `vet.risk` | A Definition recorded `ok` at its current inputs scores above 30 on c² × (1 − e)³ + c, with c = 1 + statements and e = evidence pointers / c, at most 1 | 30 | no record exists | Uncalibrated, and it can be gamed by naming more pointers. It also makes `vet` read `.markitect/review.sum` beside the spec |

### Measured and not proposed

| Measure | Why not |
|---|---|
| Abstractness and distance from the main sequence, per Package | Needs to know which Kinds are abstract, and on both specs it told good Packages from bad ones no better than instability: every area Package holds its vocabulary beside its operations ([guide](../guides/good-spec.md#stable-dependencies-and-stable-abstractions)) |
| Every-link reach | The record follows one step; the every-link reach is for the author to read, not to warn about. `list --deps` and `doc` already show the links |
| Fan-out | Its largest values are Interfaces, which list their module's operations by design |
| Relational cohesion H, links leaving a Definition's Package | Low values were the outer layer (`protocol`, `reporting`) and operations that use another area's Concepts, both legitimate |
| Definition cycles | All lie inside one Package; about half are a Concept and its Rule naming each other |
| A Kind that links Definitions other than Kinds | One hit per spec, the `Operation` Kind, whose links reflect what its Definitions truly rely on; whether it mixes levels needs judgment |

## Should Kinds declare a layer?

So that `vet` could check the Dependency Rule by name, a Kind could declare its layer in its front matter, as in `layer: domain`, and `vet` would warn about a link from an inner layer to an outer one.

| Way | What it gives | What it costs |
|---|---|---|
| **1. No declaration (recommended).** The tree carries the inner direction: a parent may not link into a child, so a parent is inner to its children. `vet.unstable_dependency` checks that dependencies point towards stability. The architecture Decision at Root names the layers, and the auditor checks the edges against it | Works today, with no change to the language | A layer violation between two siblings whose instabilities happen to fit goes unseen by `vet` |
| **2. `layer:` in a Kind's front matter** | `vet` checks every edge against named layers | Changes the language, whose front matter holds only `kind` and `internal`, and gives Kinds a meaning to Markitect, which [DEC-080](register.md#dec-080-the-model-is-typed-documentation) removed in favour of code-like metrics and the cycle rule. Reopening that needs new evidence |
| **3. Layers as nested Packages,** the inner layer the parent of the outer | `link.downward` enforces the rule for free | One more level per module, and a tree that reads inside out to people used to `api/` at the top |

**Recommendation: way 1, for now.** On both specs the Dependency Rule already holds: no Package depends on the outer `protocol`, the area modules reach each other's operations only through Interfaces, and the stability measure finds no violation in Makerspace and three small ones in the 5× project. A declared layer would have found nothing that these do not. The same reason holds for abstractness: `vet` cannot compute it without knowing what Kinds mean, and Makerspace's operations use a project Kind, so the standard Kinds alone would not tell. Revisit when a real spec shows an inner Package depending on an outer one that the stability measure misses.

## Decisions for the owner

1. **`vet.hub` replaces `vet.reach`:** Definitions, Root included, Kinds left out, above a quarter of the spec and at least 10. Recommended.
2. **A margin of 0.1 on `vet.unstable_dependency`.** Recommended.
3. **A stricter tier under `--strict`** with `vet.statements`, `vet.cohesion` and `vet.placement`. Recommended, with the thresholds above as provisional.
4. **`vet.risk`, the CRAP analogue,** which makes `vet` read `review.sum`. Recommended only under `--strict`, and only once the record of a real build can calibrate it.
5. **No layer in Kinds**, as above. Recommended.

Once decided, the build changes `internal/analysis`, its tests, and the warning table of [the model language guide](../guides/model-language.md#vet-warnings), and this page becomes the plan's record, or goes.

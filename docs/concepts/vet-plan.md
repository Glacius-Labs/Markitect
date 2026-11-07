# Proposal: what `vet` measures

**Status:** a proposal for the owner, written on 7 November 2026. Nothing on this page is decided or built. It answers part of [OQ-002](register.md#open-questions) and the question [concept 4.0 §4](v4-concept.md#4-is-the-spec-good) leaves open: the details of `vet`, its further measures and thresholds. It follows the rules the owner set for it the same day:

- no change to the language and no new field;
- no explicit layers: the Package tree is the Dependency Rule, as in Go;
- a Package's abstractness comes from its Kinds, as Martin counts interfaces;
- the tool never treats a Kind specially by its name: what depends on a Kind's meaning is for the agents, who read its text.

**Basis:** the [measurements of 7 November 2026](../validation/spec-design-measures-20261107.md) on the two archived Makerspace specs, 115 and 419 Definitions, and the [guide to a good spec](../guides/good-spec.md), which says what each measure is for. [The model language guide](../guides/model-language.md#vet-warnings) owns what `vet` does today, and changes with it once the owner decides.

## In brief

- **Replace `vet.reach` with `vet.hub`,** which counts Definitions, counts Root, and leaves Kinds out. On both specs it warns about `values` and `members/member`, the hub that cost most re-verification in the 5× change steps; `vet.reach` warns about Interfaces, which are linked widely by design, and misses `values`.
- **Add Martin's abstractness and distance:** A and D in `list --deps`, a warning for the zone of uselessness, and the zone of pain under `--strict`, since without a measure of change it names every stable core.
- **Keep the other warnings,** with a margin of 0.1 on `vet.unstable_dependency`.
- **Add a stricter tier** under `--strict`: statements per Definition, Package cohesion, placement, and a CRAP analogue that reads `review.sum`.
- **Add one Kind to `std`, `std/Value`,** for the value objects of domain-driven design, where the measurements show a gap.

## What `vet` knows of Kinds

`vet` knows one Kind: Core's `Kind`, which every Kind Definition has as its own Kind. A Definition of Kind `Kind` counts towards its Package's abstractness, and `vet.hub` leaves it out, since a change to a Kind concerns all its Definitions by design. Nothing else in `vet` depends on which Kind a Definition has. A project's own Kinds and `std`'s are treated alike, and a measure that would need to know what a Kind means, such as which Kinds are concrete, is not proposed.

## The measures

N is the number of the spec's model Definitions. Fan-in counts the Definitions that link to a Definition or are of it as Kind. A is the share of a Package's own Definitions that are of Kind `Kind`, I is Ce / (Ca + Ce), and D is |A + I − 1|. Hits are on Makerspace c8 and the 5× project's c8, as the archive holds them.

### Warnings

`vet` prints these always and exits 1 on them only with `--strict`, as today.

| Code | Warns when | Threshold | Hits: Makerspace, 5× | Change |
|---|---|---|---|---|
| `vet.hub` | A Definition not of Kind `Kind`, Root included, whose change alone puts more than a fifth of the spec back on the list: fan-in + 1 > max(10, 0.2 N) | 20 %, at least 10 | `values` (61 of 115), `members/member` (25); `values` (270 of 419), `members/member` (91) | New, replaces `vet.reach` |
| `vet.zone_of_uselessness` | A Package is abstract and unstable: D > 0.7 with A + I > 1, so Kinds that little depends on | 0.7 | none; none | New |
| `vet.coupling` | A Package depends on more than 5 Packages (Ce) | 5 | `protocol` (8); 6 Packages, `bookings` 12 at most | Keep |
| `vet.unstable_dependency` | A sideways edge goes from a Package to one whose instability is higher by more than the margin | margin 0.1 | none; 2 (differences 0.12 and 0.18) | Add the margin; today 0 and 3 |
| `vet.definition_size` | A Definition's body has more than 1000 words | 1000 | none (largest 430); none (largest 995) | Keep |
| `vet.package_size` | A Package has more than 25 Definitions of its own, Root included | 25 | none (Root 21); none | Keep |
| `vet.thin_package` | A Package has no Definitions of its own and one child | — | none; none | Keep |
| `vet.no_paths` | A Package other than Root claims no path pattern | — | as today | Keep |

**`list --deps`** adds each Package's A and D beside its instability, in its text and as `abstractness` and `distance` in its JSON.

**Why `vet.hub` and not `vet.reach`.** `vet.reach` counts the Packages a change reaches and skips Root. On Makerspace it warns about the Interfaces of `billing`, `equipment` and `members`, whose purpose is to be used by other modules, and stays silent about `values`, which 60 Definitions link while using a median of 2 of its 13 formats. `vet.hub` counts Definitions, which is what `status` lists, and counts Root, where the worst hub sits. At 20 % it also names `members/member`: in the 5× project's eight change steps, `members/member` changed once and put 88 Definitions back on the list, and the changes of nine widely linked Definitions together put 247 of the steps' 468 items there. At 25 % it would name `values` alone, which never changed. After `values` is split, it names `values/identifier`, a true hub, which the author keeps with one line of reason, as the modeling skill asks of any warning, and in the 5× project still `members/member`.

**Why the zones split.** Both specs keep their Kinds at Root, so A is 0 in every other Package and D is 1 − I. The zone of pain then holds Root and the stable core: `billing`, `equipment` and `members` in Makerspace, eight Packages in the 5× project. That is where change hurts, but Martin counts it harmful only for what changes often, and in Makerspace nothing widely linked there changed. A snapshot cannot tell, so the zone of pain waits under `--strict` until `vet` can see change. The zone of uselessness needs no such knowledge: Kinds in an unstable Package are Kinds little uses.

**Why the margin.** Splitting `values` into a Package gave every module one more efferent edge and shifted instabilities, which raised three new `vet.unstable_dependency` warnings with differences of 0.04, 0.06 and 0.10. The warnings that stood before differ by 0.10 to 0.18. A margin of 0.1 keeps the larger ones and drops the shifts. Without it, the warning is Martin's Stable Dependencies Principle exactly; with it, it is the same rule with a tolerance, provisional like every threshold here.

### Only with `--strict`

With `--strict`, `vet` also runs these, prints them, and exits 1 on any warning of either tier. They are stricter because each has a known blind spot, or no calibration yet.

| Code | Warns when | Threshold | Hits: Makerspace, 5× | Why only strict |
|---|---|---|---|---|
| `vet.zone_of_pain` | A Package is stable and concrete: D > 0.7 with A + I < 1 | 0.7 | Root, `billing`, `equipment`, `members`; Root and 8 Packages | Every domain has a stable core; it hurts only where it changes, which a snapshot cannot see |
| `vet.statements` | A Definition has more than 20 statements: list items and table rows in its body | 20 | 2 (`reporting/revenue-report` 25, `bookings/create-reservation` 22); 8 (`list-order` 67 at most) | The anchor and the architecture are long by nature (5× `architecture` 60, `project` 26) |
| `vet.cohesion` | A Package falls into two or more groups of two or more Definitions each, with no link between the groups | — | none; none | No hit on either spec, so untested against a real split |
| `vet.placement` | Every Definition that links to a Definition lies below it in one child Package, so it could move down to that child | — | none; none | A Rule that binds by scope may be linked from one child only and still belong higher. Both specs are one level deep, so untested |
| `vet.risk` | A Definition recorded `ok` at its current inputs scores above 30 on c² × (1 − e)³ + c, with c = 1 + statements and e = evidence pointers / c, at most 1 | 30 | no record exists | Uncalibrated, and it can be gamed by naming more pointers. It also makes `vet` read `.markitect/review.sum` beside the spec |

### Measured and not proposed

| Measure | Why not |
|---|---|
| Change hotspots: a Definition's fan-in times how often it changes | The strongest sign of real cost in the 5× series, but it needs history, and the tool reads no Git ([DEC-097](register.md#dec-097-markitect-uses-no-git)). Later, as concept 4.0 §4 says, perhaps from one record to the next; then the zone of pain can become a warning for volatile Packages |
| Abstractness by what Kinds mean (Concepts and Rules abstract, Operations concrete) | Needs `vet` to know Kinds by name, and told good Packages from bad ones no better than instability |
| Every-link reach | The record follows one step; the every-link reach is for the author to read, not to warn about. `list --deps` and `doc` already show the links |
| Fan-out | Its largest values are Interfaces, which list their module's operations by design |
| Relational cohesion H, links leaving a Definition's Package | Low values were the outer `protocol` and `reporting` and operations that use another area's Concepts, all legitimate |
| Definition cycles | All lie inside one Package; about half are a Concept and its Rule naming each other |
| A Kind that links Definitions other than Kinds | One hit per spec, the `Operation` Kind, whose links reflect what its Definitions truly rely on; whether it mixes levels needs judgment |

## The Dependency Rule without layers

The owner settled this on 7 November 2026: there are no explicit layers and no mechanism by which one Kind refines another. Markitect has Packages and links like Go, and Go has no layers either.

- **The tree carries it.** Links up are free and links down are forbidden (`link.downward`), so dependencies already point from the specific to the general.
- **Sideways edges** are checked by the cycle ban (`package.cycle`) and by `vet.unstable_dependency`, Martin's Stable Dependencies Principle from the graph alone.
- **A project's own limits** use what exists: `internal: true` for what a Package keeps to itself, which `check` enforces (`link.internal`), and an ordinary Rule for the rest, which the reviewer checks. Makerspace marks 19 Rules `internal: true`.

On both specs the rule holds without more: no Package depends on the outer `protocol`, the area modules reach each other's operations only through Interfaces, and the stability measure finds no violation in Makerspace and three small ones in the 5× project.

## A new `std` Kind: Value

New `std` Kinds come only where the measurements show a gap and an established concept fills it. One gap shows.

**The concept.** A value object in domain-driven design (Eric Evans, *Domain-Driven Design*, 2003): an element of the domain defined by its content alone, with no identity of its own and no lifecycle; two values with the same content are equal, and a value is replaced, not changed. Its counterpart is the entity, which has an identity that persists while its attributes change.

**The gap.** `std/Concept` explains "one thing of the project's domain: what it is, what it consists of, how it relates to other Concepts, and which states it can be in": an entity in all but name, and Makerspace's Concepts read so ("A record is identified by `memberId`"). `std` has no Kind for a value. Makerspace wrote its thirteen value formats, identifiers, amounts, timestamps and the like, as one Concept page, `values`, which became the largest hub of both specs: 60 and 269 linkers, using a median of 2 and 1 of its formats. Split, the formats are thirteen Definitions with no identity, relations or states, written as Concepts for want of a Kind.

**The proposed Kind,** as a draft of its Definition:

```markdown
---
kind: Kind
---
# Value

A Value Definition describes one kind of value that the project's records, requests or results carry, such as an identifier, an amount of money or a date. A value is defined by its content alone: it has no identity and no states, and two values with the same content are equal.

- State its form and range exactly, in backticks: the wire type, the pattern or the bounds, and an example.
- State when two values are equal, and how a value is normalized, if it is.
- State its unit, if it has one.
- A Value links only the Values it is made of. The Definitions that use it link it, one link per Value.
```

`std/Concept` would gain one sentence: a value with no identity is a `[[std/Value]]`. The tool would do nothing special with it. A new Kind in `std` is a new version of the Library: every project's lock changes with `markitect update`, and since a Library Definition's input is its Library's digest, every Definition of a `std` Kind goes back on the list once.

**Considered and not proposed:**

| Candidate | Grounded in | Why not |
|---|---|---|
| Entity | DDD | `std/Concept` already is one |
| Interface, a module's published surface | Parnas's information hiding, a C4 component | Both specs define it as a project Kind (8 and 46 Definitions), but visibility and Package links already carry a module's boundary, and the measurements show no problem it would fix |
| Aggregate | DDD | No measured gap; a Package groups what belongs together |
| Architecture decision record | ADR | `std/Decision` already is one |
| Views of the architecture | C4, arc42 | The Project, the architecture Decision and the Packages cover them |

## Decisions for the owner

1. **`vet.hub` replaces `vet.reach`:** Definitions, Root included, Kinds left out, above a fifth of the spec and at least 10. Recommended.
2. **Martin's abstractness and distance:** A and D in `list --deps`, `vet.zone_of_uselessness` as a warning, `vet.zone_of_pain` under `--strict`. Recommended.
3. **A margin of 0.1 on `vet.unstable_dependency`.** Recommended.
4. **A stricter tier under `--strict`** with `vet.statements`, `vet.cohesion` and `vet.placement`. Recommended, with the thresholds above as provisional.
5. **`vet.risk`, the CRAP analogue,** which makes `vet` read `review.sum`. Recommended only under `--strict`, and only once the record of a real build can calibrate it.
6. **`std/Value`** in a new version of `std`, with one sentence in `std/Concept`. Recommended.

Once decided, the build changes `internal/analysis`, its tests, `list --deps`, and the warning table of [the model language guide](../guides/model-language.md#vet-warnings); a new `std` Kind also changes `internal/libraries/std` and the Library's version. This page then becomes the plan's record, or goes.

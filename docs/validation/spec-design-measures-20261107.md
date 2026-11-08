# How well the Makerspace specs are designed: measurements — 7 November 2026

This session measured the design of the two archived Makerspace specs the way code is measured: coupling, stability, abstractness, cohesion, fan-in and fan-out, how far a change reaches, what changed over the change steps, and where prose names a Definition without linking it. It then split the `values` page that [concept 4.0 §4](../concepts/v4-concept.md#4-is-the-spec-good) names as a hub, and measured again. It started no agent and changed no tool. The reading is this session's and decides nothing; the [guide to a good spec](../guides/good-spec.md) draws the rules from it, and the [`vet` plan](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/concepts/vet-plan.md) proposes what `vet` measures.

**What the numbers show:**

- **The Package structure is sound.** No Package cycle, nothing depends on the outer `protocol` Package, dependencies point towards the stable `members` and `equipment`, and no Makerspace edge points to a less stable Package; the 5× project has three such edges.
- **Behaviour is never a hub.** An Operation is linked by at most two Definitions in Makerspace and three in the 5× project, mostly only by its own module's Interface. The links go to Concepts, Kinds and formats, which is what the stable-abstractions principle asks, per Definition.
- **Abstractness, counted as Martin counts it, adds little here.** A Package's abstractness is its share of Definitions of Kind `Kind`, as Martin counts interfaces and abstract classes. Both specs keep their Kinds at Root, so every other Package has abstractness 0 and its distance from the main sequence is 1 − I. The zone of pain holds Root and the stable core, `members`, `equipment` and `billing` (eight Packages in the 5× project), and the zone of uselessness is empty.
- **Where change hurts is a widely linked Definition that changes.** No Makerspace Definition linked by 10 or more changed in eight steps. In the 5× project 9 of 29 did, `members/member`, `billing/charge` and `members/membership` first, and 248 of the 468 items the eight steps put back on the list came from their changes alone. `values` never changed.
- **Root is the weak spot.** `values` alone is linked by 60 of 115 Definitions (269 of 419 in the 5× project), and Root also holds a module: formats, refusal codes, audit and authorization, with the only Operation at Root. Links up the tree are free, so none of this shows in Package coupling, and `vet.reach` skips Root.
- **Splitting `values` narrows what a format change lists, and nothing else.** With the formats at Root, a change to one format lists between 2 and 52 Definitions instead of 61, a median of 5. The real change steps never changed a format, so what they put back on the list stays the same. Following every link, the four widest Makerspace steps fall from 85–88 % of the spec to 45–60 %, and the 5× project's from 94–96 % to 70–82 %, but not because of the split: the split dropped the one link `values` made, back to its user `argument-values`, and removing that link alone gives the same numbers.
- **Prose names much that it does not link.** A text check finds a Definition's name in another Definition's prose with no link to it in 233 pairs of Makerspace Definitions and 863 of the 5× project, after filters that use the structure only. Two judges of the same samples found half to three-quarters of them real missing links; the rest were ordinary words and other meanings of a name.

## Source and method

**Fixtures.** The archive point [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks): `tools/evaluation-tasks/makerspace` (steps `w0` and `c1` to `c8`, each with its finished registry from `references/<step>`) and `tools/evaluation-tasks/makerspace-large` (the 5× project, steps `w0` to `c8`, each with its own registry). Each step was copied into a scratch project folder with its manifest, model and registry, and `markitect tidy` wrote the lock. `markitect check --spec` passes on every step. Only public fixtures were read.

**Tool.** `markitect` built from Main `836ff17c` (0.22.0-dev). The links come from `markitect list --deps --json`; Package coupling and instability are the tool's own (explicit edges only, links up the tree free). `markitect vet` ran on each step as it is today.

**Throwaway scripts,** outside the repository; the method below repeats them:

| Script | Reads | Computes |
|---|---|---|
| measure | `list --deps --json` and the Definition files | The measures below, per Definition and per Package |
| split | A step's model | The split of `values`, in two variants ([below](#the-split)) |
| steps | Two consecutive steps | The Definitions a step changed, and how far that change reaches at one step, two steps and every link |
| mentions | `list --deps --json` and the Definition files | Unlinked mentions ([below](#unlinked-mentions)) |

**Measures:**

| Measure | How it is computed |
|---|---|
| Definition link | A link from a Definition to a model Definition, and a Definition's project Kind. Links to Packages and Library Kinds are no Definition links: a Package link is no input of the record either |
| Fan-in, fan-out | Definitions that link to a Definition or are of it as their Kind; Definitions it links to |
| One-step reach | The Definition and its fan-in: what a change to it alone puts back on the list of `status` |
| Every-link reach | The Definition and everything that reaches it by following links backwards, as in the [3 November measurement](impact-reach-20261103.md) |
| Ca, Ce, I | Afferent and efferent coupling and instability Ce / (Ca + Ce), as `list --deps` reports them |
| A | The share of a Package's own Definitions whose Kind is `Kind`, as Martin's abstractness counts interfaces and abstract classes among all classes. It reads no Kind's meaning |
| D, zones | \|A + I − 1\|, the distance from the main sequence. Above 0.7 a Package lies in the zone of pain (stable and concrete, A + I below 0.3) or in the zone of uselessness (abstract and unstable, A + I above 1.7) |
| Volatility | Definitions a step added, changed or removed, per Package and per Definition, over c1 to c8; for each change of an existing Definition, the one-step reach it caused at that step. A hub is a Definition with fan-in 10 or more at c8. An item comes from hub changes alone when no other change of its step reached it; the changed hubs themselves are not counted |
| Cohesion | Connected groups of a Package's own Definitions, joined by links inside it; relational cohesion H = (links inside + 1) / Definitions; the share of the Package's links that stay inside, not counting links up the tree |
| Statements | List items and table rows in a Definition's body, header rows included, a count of what a reviewer verifies |
| Definition cycles | Strongly connected groups of more than one Definition |
| Percentiles | Nearest rank; reach and fan-in over all model Definitions |

**The method repeats the 3 November measurement:** one step, two steps and every link give its numbers for c1 to c8, and its per-Definition distribution for the 5× project (2 / 8 / 269 at one step, 7 / 70 / 395 for every link). The one difference is c7's count of changed Definitions, 12 here against 16 there, since a Definition the step removed is not counted here.

## The specs as a whole

| Spec | Definitions | Packages | Definition links | Fan-in median / p90 / max | Fan-out median / p90 / max | One-step reach median / p90 / max | Every-link reach median / p90 / max | Over half the spec by every link (at Root) | Definition cycles |
|---|---|---|---|---|---|---|---|---|---|
| Makerspace w0 | 108 | 10 | 353 | 1 / 7 / 58 | 3 / 8 / 13 | 2 / 8 / 59 | 8 / 51 / 96 | 9 (9) | 3 |
| Makerspace c4 | 112 | 10 | 376 | 1 / 7 / 60 | 3 / 7 / 16 | 2 / 8 / 61 | 8 / 52 / 100 | 9 (9) | 3 |
| Makerspace c8 | 115 | 10 | 382 | 1 / 7 / 60 | 2 / 8 / 16 | 2 / 8 / 61 | 8 / 52 / 103 | 9 (9) | 3 |
| 5× w0 | 413 | 48 | 1798 | 1 / 7 / 268 | 4 / 9 / 23 | 2 / 8 / 269 | 7 / 70 / 395 | 17 (15) | 27 |
| 5× c8 | 419 | 48 | 1860 | 1 / 7 / 269 | 4 / 9 / 24 | 2 / 8 / 270 | 8 / 85 / 402 | 18 (15) | 29 |

The shape holds across the steps: a typical Definition is linked once and reaches 2 at one step and about 8 by every link, and a handful at Root reach most of the spec.

`vet` today warns on every Makerspace step about the same things: `protocol` depends on 8 Packages (`vet.coupling`), and the Interfaces of `billing`, `equipment` and `members` reach 6 of 10 Packages (`vet.reach`). On the 5× project it warns about six Packages with more than 5 dependencies (`bookings` 12), three less stable dependencies, and `members/member` and `members/members-interface`. It never warns about `values`, since `vet.reach` skips Root.

## Packages

Makerspace c8:

| Package | Definitions | Ca | Ce | I | A | D | Zone | Groups | H | Links inside | Up | Out | Share inside | Changes c1–c8 (steps) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Root | 21 | 0 | 0 | 0 | 0.10 | 0.90 | pain | 4 | 2.00 | 41 | 0 | 0 | 1.00 | 23 (6) |
| billing | 16 | 5 | 1 | 0.17 | 0 | 0.83 | pain | 2 | 1.38 | 21 | 18 | 10 | 0.68 | 18 (3) |
| bookings | 17 | 2 | 5 | 0.71 | 0 | 0.29 | | 2 | 1.65 | 27 | 18 | 28 | 0.49 | 20 (6) |
| equipment | 12 | 5 | 0 | 0.00 | 0 | 1.00 | pain | 1 | 1.92 | 22 | 15 | 0 | 1.00 | 3 (1) |
| materials | 9 | 1 | 2 | 0.67 | 0 | 0.33 | | 1 | 1.67 | 14 | 12 | 10 | 0.58 | 5 (4) |
| members | 11 | 5 | 0 | 0.00 | 0 | 1.00 | pain | 1 | 1.82 | 19 | 14 | 0 | 1.00 | 3 (1) |
| protocol | 6 | 0 | 8 | 1.00 | 0 | 0.00 | | 5 | 0.33 | 1 | 4 | 9 | 0.10 | 1 (1) |
| reporting | 6 | 1 | 3 | 0.75 | 0 | 0.25 | | 2 | 1.17 | 6 | 7 | 16 | 0.27 | 5 (2) |
| safety | 6 | 2 | 1 | 0.33 | 0 | 0.67 | | 1 | 1.50 | 8 | 7 | 3 | 0.73 | 4 (2) |
| training | 11 | 2 | 3 | 0.60 | 0 | 0.40 | | 1 | 2.00 | 21 | 14 | 17 | 0.55 | 6 (2) |

Root's coupling is 0 by the tool's count only: links up are free, and 9 of the 9 other Packages link Root's Definitions. Its instability is 0 either way, since it links nothing outside itself. Step w0 gives the same A, D and zones.

The 5× project, 47 Packages besides Root: Ce median 3 and at most 12 (`bookings`); I median 0.75; A 0 in every Package but Root (0.10); D median 0.25; in the zone of pain Root, `billing`, `equipment`, `guardians`, `ledger`, `lockouts`, `members`, `notices` and `rooms`; nothing in the zone of uselessness; H median 1.88, lowest 0.33 (`protocol`); share inside median 0.56, lowest 0.21 (`compliance`).

**Coupling and stability.** No Package depends on `protocol` in either project: it is the outer layer that wires the modules, and its Ce of 8 is what such a layer has. The stable Packages, `members` and `equipment` (I 0), are what the domain rests on. The three less stable dependencies of the 5× project differ by 0.10, 0.12 and 0.18 in instability.

**Abstractness and the main sequence.** A Kind is to Definitions what an interface is to classes: the form others follow. Both specs define their two project Kinds, `Operation` and `Interface`, at Root, so Root's abstractness is 2 of 21 and every other Package's is 0. D is then 1 − I outside Root and adds nothing to instability. The zone of pain, stable and concrete, holds the Packages whose Definitions the others' links rest on, such as `members` (`members/member`), `equipment` (`equipment/machine`), `billing` (`billing/charge`) and `ledger` (`ledger/postings`), and in the 5× project also smaller stable ones such as `guardians`, `lockouts`, `notices` and `rooms`. Martin calls that zone harmful only for what changes often; the [volatility](#volatility) below shows which of them do. A reading of what each Kind means, Rules and Interfaces as abstract and Operations as concrete, was also tried. Counting Concepts as abstract, it puts every area Package at 0.5 to 0.7; counting them as concrete, `members` and `equipment` move to D 0.73–0.75. Either way it tells little beyond instability, and the tool cannot use it, since it treats every Kind alike. In code the Package is the unit that is released and depended on; here the unit of verification is the Definition, and a change to `members/register-member` puts back only itself and its Interface, however stable `members` is. Per Definition, what is linked widely is the form and the vocabulary, not the behaviour:

| Kind, Makerspace c8 | Definitions | Fan-in median | Fan-in max |
|---|---|---|---|
| `Operation` | 38 | 1 | 2 |
| `std/Rule` | 41 | 1 | 7 |
| `Interface` | 8 | 3 | 9 |
| `std/Concept` | 24 | 6 | 60 |
| `Kind` | 2 | 23.5 | 39 |

The one Operation linked from outside its module's Interface is Root's `list-audit-entries`, by `audit` and `protocol/requests`. In the 5× project an Operation has at most 3 linkers, and 14 links reach an Operation from another Package other than through an Interface. Ten come from Rules that add a gate or an effect to another module's operation, such as `hazmat/hazard-gate` on `materials/purchase-material`: the inverted dependency the [model language guide](../guides/model-language.md#links-scope-and-visibility) describes, where the needing side states its need. One is `protocol/requests` naming Root's `list-audit-entries`. Three come from Operations that build on another module's operation: `access/enter-door` on `billing/get-account`, `finance/finance-summary` on `billing/issue-credit` and `waitlists/promote-waitlist` on `training/enroll-in-course`.

**Cohesion.** At c8 every Package is one connected group, apart from single Definitions that link nothing in it: Rules that bind their Package by scope (`billing/due-date-and-block`, `bookings/metering`, `reporting/reports`, and at Root `documentation` and `list-order`) and Root's Kind `Interface`. At c8 no Package of either project falls into two groups of two or more Definitions each. At w0 and c1, `bookings` does: three Rules, `booking-granularity`, `opening-hours` and `reservation-rules`, link each other but nothing else in the Package. They bind it by scope, so this is no second area. `protocol` is the exception in kind: six Rules with one link between them, each tying other modules' Interfaces to the wire.

**Placement.** A Definition sits too high when every Definition that links to it lies below it in one child Package. None does in either project: both are one level deep, and each Root Definition that is linked at all is linked from Root itself or from two or more child Packages.

## Definitions

Hubs, by fan-in:

| Definition | Kind | Makerspace c8: fan-in, Packages | 5× c8: fan-in, Packages |
|---|---|---|---|
| `values` | Concept, at Root | 60, 9 | 269, 47 |
| `Operation` | Kind, at Root | 39, 8 | 193, 46 |
| `members/member` | Concept | 24, 4 | 90, 27 |
| `members/members-interface` | Interface | 9, 5 | 48, 31 |
| `Interface` | Kind, at Root | 8, 8 | 46, 46 |
| `equipment/machine`, `equipment/family` | Concept | 14 and 9 | 31 and 32 |
| `ledger/postings`, `billing/charge` | Rule, Concept | — | 23 and 20 |

- **`values`** holds thirteen unrelated formats. Its linkers use a median of 2 of them in Makerspace and 1 in the 5× project, at most 5.
- **`members/member`** is linked by 17 of its 24 linkers, and by 83 of 90 in the 5× project, only to say that an identifier names an existing member. They rely on the member's identity, not its fields, so a new field puts them all back on the list. It is the same pattern as `values`, milder.
- **`Operation`** is a true hub: every Operation is of it. It also links six Root Definitions (`argument-values`, `audit`, `audit-entry`, `authorization`, `order-of-checks`, `refusal-codes`), so following every link, a change to any of them reaches all Operations. The Operations do rely on them: 21 of the 38 say "Refusals at check 7". But none links `order-of-checks` itself, so a change to the order of checks puts the Kind back on the list, not those Operations.
- **The largest fan-out** belongs to the Interfaces, which list their module's operations (`bookings/bookings-interface` 16, 24 in the 5× project), and to two Operations (`materials/purchase-material` 13, `reporting/utilization-report` 11).
- **Size.** Words per Definition: median 88, p90 253, largest 430 (`architecture`); in the 5× project 99, 237 and 995 (`ledger/postings`). Statements: median 6, p90 12, largest 25 (`reporting/revenue-report`); in the 5× project 7, 13 and 67 (`list-order`, a table of the order of every list).
- **Definition cycles** all lie inside one Package, as they must, since a cycle across Packages is a compile error. Many are a Concept and its Rule naming each other (`bookings/reservation` and `bookings/reservations`; 10 of 29 in the 5× project, and 13 consist of Concepts and Rules only); most others include an Operation. Two are at Root: `values`, `argument-values`, `money`, `computed-times`, `identifier-limits` and `integer-limits`, and `Operation`, `audit`, `list-audit-entries` and `order-of-checks`. A cycle at Root makes its members act as one hub when every link is followed.

## Volatility

Over the eight change steps, counting Definitions a step added, changed or removed (the Package table above has Makerspace's counts per Package):

- **Root changes most** in both projects: in 6 of 8 Makerspace steps and in all 8 of the 5× project's. What changes there is the anchor, which lists the modules (4 and 8 times), `computed-times`, `time`, `architecture` and, in the 5× project, `list-order`. All have fan-in 0 to 2, so their changes put little back on the list. Root's hubs, `values`, `Operation` and `Interface`, never changed.
- **In Makerspace no widely linked Definition changed.** None of the 5 Definitions with fan-in 10 or more changed, against 49 of the other 110; no item of any step came only from a changed hub.
- **In the 5× project widely linked Definitions did change,** 9 of 29 against 41 of the other 390, and 248 of the steps' 468 items came from their changes alone:

| Definition, 5× project | Fan-in at c8 | Changes | Items its changes put on the list |
|---|---|---|---|
| `members/member` | 90 | 1 | 88 |
| `billing/charge` | 20 | 3 | 60 |
| `members/membership` | 20 | 3 | 60 |
| `ledger/postings` | 23 | 2 | 48 |
| `members/members-interface` | 48 | 1 | 47 |
| `waivers/waiver-gate` | 10 | 2 | 21 |

In Makerspace the largest such sums are `billing/billing-interface` (14), `time` (12) and `bookings/create-reservation` (8). So the zone of pain did hurt in the 5× series, through `members`, `billing` and `ledger`, and Root, though volatile, hurt little, because what changed there was what nothing links.

## Root

Makerspace c8, Root's 21 Definitions:

| Definition | Kind | Fan-in | Other Packages linking it | Reading |
|---|---|---|---|---|
| `project` | Project | 0 | 0 | The anchor |
| `architecture` | Decision | 1 | 0 | The architecture |
| `Operation`, `Interface` | Kind | 39, 8 | 8, 8 | Shared Kinds |
| `time`, `money`, `list-order`, `order-of-checks`, `request-limits`, `identifier-limits`, `integer-limits`, `computed-times`, `argument-values`, `tests`, `documentation` | Rule | 0 to 7 | 0 to 2 | Project-wide Rules; most bind by scope |
| `values` | Concept | 60 | 9 | Formats of every module |
| `refusal-codes` | Concept | 7 | 1 | Codes the protocol shares |
| `authorization`, `audit` | Rule | 5, 3 | 1, 0 | Policies of a module |
| `audit-entry` | Concept | 3 | 0 | A record of that module |
| `list-audit-entries` | Operation | 2 | 1 | The only Operation at Root |

The project page says it outright: the area modules, the outer layer `protocol`, "and the shared policies module, which this folder describes". So Root is both the project's anchor and a module. Two things keep that module at Root:

- **A parent may not link into a child** (`link.downward`). The Root Kind `Operation` links `audit`, `audit-entry`, `authorization` and `refusal-codes`, so they cannot move into a Package below Root, and `audit-entry` in turn uses the formats.
- **Links up are free.** Coupling to Root shows in no Package's Ce, and `vet.reach` leaves Root out, so the hub costs nothing in today's measures.

## The split

`values` was split into one Definition per format, by script, on every step of both projects, in two variants:

- **A `values/` Package** below Root, with the thirteen formats as Concepts (`values/identifier`, `values/cent-amount`, …), each holding its row of the table, the units folded into `cent-amount` and `minute-count`. Every field line that names a format links it, as in ``- `blocked`: [[values/flag|flag]]``; a field "integer from `1` to `100000000` cents" links `cent-amount` at its unit, "whether …" links `flag`, and "minutes of …" links `minute-count`. The header ``**Fields**, in the formats of [[values]]`` stays and now names the Package, which is no input. Root's own Definitions may not link into the Package, so `audit-entry`, `identifier-limits`, `money`, `time` and `project` keep a link to the Package. The formats link nothing, so the old back link from `values` to `argument-values`, and with it Root's six-member cycle, is gone. Root's registry page lists the five formats its file realizes instead of `values`.
- **The same thirteen formats at Root,** where every Definition, Root's included, can link the format it uses.

Both variants compile on every step. All 60 linkers (269 in the 5× project) map to their formats by these rules, apart from `project`, which names the formats as part of a module.

**Before and after, Makerspace c8 and 5× c8:**

| | Makerspace before | Makerspace, `values/` | Makerspace, at Root | 5× before | 5×, `values/` |
|---|---|---|---|---|---|
| Definitions, Packages | 115, 10 | 127, 11 | 127, 10 | 419, 48 | 431, 49 |
| Definition links | 382 | 439 | 449 | 1860 | 2063 |
| Largest fan-in | 60 (`values`) | 49 (`values/identifier`) | 51 (`identifier`) | 269 (`values`) | 242 (`values/identifier`) |
| One-step reach max | 61 | 50 | 52 | 270 | 243 |
| Every-link reach median / p90 / max | 8 / 52 / 103 | 9 / 53 / 81 | 9 / 55 / 91 | 8 / 85 / 402 | 8 / 95 / 378 |
| Over half the spec by every link | 9 | 3 | 6 | 18 | 22 |
| Root's own Definitions | 21 | 20 | 33 | 21 | 20 |
| Definition cycles | 3 | 2 | 2 | 29 | 28 |

**What a change to one format lists** (one-step reach, the format included): before the split, 61 Definitions for any format in Makerspace and 270 in the 5× project. At Root, Root's own Definitions link their formats too, so each format lists one to two more than in the `values/` Package. These are lower bounds: of the field lines, 78 of 276 in Makerspace and 345 of 1079 in the 5× project name no format and link none. Most of them are records and enumerations, but a few rely on a format without naming it, such as a sum of amounts, or the 5× project's 54 quantities written "integer from …" without a unit.

| Format | Makerspace, `values/` | Makerspace, at Root | 5×, `values/` | 5×, at Root |
|---|---|---|---|---|
| identifier | 50 | 52 | 243 | 245 |
| text | 19 | 19 | 65 | 65 |
| cent amount | 17 | 18 | 63 | 64 |
| timestamp | 15 | 17 | 35 | 37 |
| count | 5 | 6 | 16 | 17 |
| minute count | 5 | 5 | 7 | 7 |
| date | 4 | 5 | 21 | 22 |
| flag | 4 | 4 | 17 | 17 |
| month | 4 | 5 | 11 | 12 |
| percent | 3 | 3 | 3 | 3 |
| clock time | 2 | 2 | 2 | 2 |
| operation name | 2 | 3 | 2 | 3 |
| audit target | 1 | 2 | 1 | 2 |
| **Median** | **4** | **5** | **16** | **17** |

**The real change steps,** before and after the split into `values/` (Definitions reached, and their share of the step's spec):

| Step | Changed | One step before / after | Two steps before / after | Every link before / after |
|---|---|---|---|---|
| Makerspace c1 | 5 | 9 (8 %) / 9 (7 %) | 47 (43 %) / 47 (39 %) | 93 (86 %) / 55 (45 %) |
| Makerspace c2 | 23 | 35 (31 %) / 35 (28 %) | 42 (37 %) / 41 (33 %) | 98 (88 %) / 74 (60 %) |
| Makerspace c3 | 2 | 3 / 3 | 4 / 4 | 4 (3 %) / 4 (3 %) |
| Makerspace c4 | 14 | 24 (21 %) / 24 (19 %) | 58 (51 %) / 58 (46 %) | 58 (51 %) / 58 (46 %) |
| Makerspace c5 | 11 | 20 (17 %) / 20 (16 %) | 24 (21 %) / 23 (18 %) | 99 (87 %) / 60 (48 %) |
| Makerspace c6 | 12 | 17 (15 %) / 17 (13 %) | 21 (18 %) / 20 (16 %) | 97 (85 %) / 59 (47 %) |
| Makerspace c7 | 12 | 17 / 17 | 20 / 20 | 20 (17 %) / 20 (15 %) |
| Makerspace c8 | 5 | 8 / 8 | 10 / 10 | 10 (8 %) / 10 (7 %) |
| 5× c1 | 3 | 5 / 5 | 9 / 9 | 9 (2 %) / 9 (2 %) |
| 5× c2 | 14 | 29 / 29 | 57 / 56 | 392 (94 %) / 298 (70 %) |
| 5× c3 | 13 | 63 / 63 | 107 / 107 | 147 (35 %) / 147 (34 %) |
| 5× c4 | 9 | 138 / 138 | 211 / 211 | 230 (55 %) / 230 (53 %) |
| 5× c5 | 26 | 86 / 86 | 153 / 152 | 408 (95 %) / 359 (81 %) |
| 5× c6 | 9 | 47 / 47 | 263 / 262 | 408 (95 %) / 363 (82 %) |
| 5× c7 | 23 | 75 / 75 | 130 / 129 | 411 (96 %) / 334 (76 %) |
| 5× c8 | 14 | 25 / 25 | 45 / 45 | 121 (28 %) / 121 (28 %) |

No step changed `values`, so one step, which is what `status` lists, reaches the same Definitions before and after. Every link narrows on the steps where it passed through `values`, but the split is not what narrows it. `values` entered every-link reach only through the one link it made, back to `argument-values`, which uses the formats; removing that link alone from the unsplit specs gives the same every-link numbers on every row. A link from the general back to what uses it closed Root's six-member cycle, and through it every change that reached `argument-values` reached all 60 users of `values`.

**What is left of every-link reach** runs largely through the `Operation` Kind on Makerspace: stopping at it alone takes 44 of the 55 Definitions out of c1 after the split, 37 of 60 out of c5 and 39 of 59 out of c6. In the 5× project it takes 221 of 298 out of c2, but only 83 of 359 out of c5, 70 of 363 out of c6 and 144 of 334 out of c7, where `members/member` comes next (22 to 32 on c5 and c6) and the rest spreads over many Definitions.

**What the split costs:**

- **Files:** 13 Definitions for 1. Their bodies hold 302 words against the page's 350: the table's header and the sentence that linked back to `argument-values` went. A new Package needs a registry page with its first paths, or `vet.no_paths` warns.
- **Links and lines:** Makerspace gains 57 Definition links and 199 written links, on 198 lines in 55 files; the 5× project gains 203 Definition links and 735 written links, on 734 lines in 264 files. Root's registry page changes two rows.
- **Reading:** every field line carries a link, and its rendered label still reads "identifier" or "cent amount". Field lists read as before; the text of a field changed only for "whether …", "the wire name …" and "minutes of …", which name their format now.
- **Root cannot follow** in the `values/` variant. Five Root Definitions keep only the Package link, so a change to a format no longer lists them, and `values/audit-target`, whose only user was Root's `audit-entry`, is linked by nothing: `check` reports it unused until a registry row realizes it. With the formats at Root instead, every link is exact, but Root grows to 33 Definitions and `vet.package_size` warns.
- **New warnings in the `values/` variant:** `vet.reach` on four formats (`values/identifier` reaches 10 of 11 Packages), `bookings` reaches Ce 6 (`vet.coupling`), and in the 5× project every Package gains one efferent edge, which shifts instabilities: three new `vet.unstable_dependency` warnings appear, with differences of 0.04, 0.06 and 0.10.

## Unlinked mentions

In code, dependencies are complete by construction: a missing import does not compile. In a spec they are complete only by discipline, since prose can name a concept without linking it. The edge is then missing, and a change to the concept does not put the Definition that relies on it back on the list. This check looks for such names, from the text and the structure alone, with no word list of any language:

- **Names.** A Definition's names are its title and the last segment of its ID, hyphens read as spaces. A name matches as whole words, without regard to case, with up to two trailing letters for inflection ("members", "invoices").
- **Where.** In every other Definition's body, without its front matter, its own title, code blocks, code spans and links, label included.
- **Candidate.** A match in a Definition that neither links the named one nor is of it as its Kind.
- **Filters,** each from the structure: the link must be allowed (not into a child, not to an internal Definition from outside, not closing a Package cycle); where names overlap, the longest wins, the Definition's own name included ("maintenance window" names the window, not `equipment/maintenance`); words that a linked Definition's name already covers do not count ("invoices" beside a linked `billing/invoice`); Kinds are no targets, since their names are mostly ordinary words ("operation"); and the link must add no new Package dependency. A mention the Definition's Kind links is still reported: the Operations' "Audit target" line relies on `audit`, which only the `Operation` Kind links, and a change to `audit` puts the Kind back on the list, not the Operations.

The cycle filter and the new-dependency filter remove mentions that go against the structure or beyond it, not noise alone. In the 5× project the new-dependency filter removes 363 pairs, 157 of them the word "order" against `purchasing/order`, but also real dependencies nobody declared; the cycle filter takes 44 and 189 pairs, counted after the other filters, where a Definition relies on something against the direction of the dependencies. The [`vet` plan](https://github.com/Glacius-Labs/Markitect/blob/d85e0fa1302ec95d1c82c6d2c088bf2ff34b6557/docs/concepts/vet-plan.md#unlinked-mentions) counts them apart.

Pairs left after each filter, on c8:

| Stage | Makerspace | 5× |
|---|---|---|
| A name found, no link | 455 | 1754 |
| The link is allowed: not into a child, not internal | 330 | 1557 |
| No cycle; not covered by a linked name | 274 | 1335 |
| The longest name wins; Kinds are no targets | 239 | 1226 |
| No new Package dependency | 233 | 863 |

In Makerspace the 233 pairs come from 86 Definitions and name 42; in the 5× project 863 pairs come from 337 and name 86. The most named without a link: `time` (37; 155 in the 5× project), `audit` (33; 145), `members/member` (31; 112), `equipment/machine` (14; 47), `refusal-codes` (8; 45), `billing/credit` and `billing/invoice` (9 each in Makerspace).

**How many are real.** This session and the independent reviewer each judged the same random samples, drawn before the Kind's links stopped covering mentions: 40 of Makerspace's 205 pairs then and 30 of the 5× project's 727. A pair is real when the Definition relies on what the named one states, so that a change to it should put the Definition back on the list.

| Sample | Judge | Real | Doubtful | Noise |
|---|---|---|---|---|
| Makerspace, 40 pairs | this session | 26 | 4 | 10 |
| | the reviewer | 21 | 8 | 11 |
| 5× project, 30 pairs | this session | 21 | 1 | 8 |
| | the reviewer | 17 | 6 | 7 |

So 52 to 65 % were real in Makerspace and 57 to 70 % in the 5× project, and up to three-quarters with the doubtful ones; with samples this small, each figure is uncertain by about 15 points. The 28 and 136 pairs added since are, all but one each, Operations whose "Audit target" line relies on `audit`, real by the rule of linking what one relies on; the other two name `audit-entry` and `refusal-codes`.

- **Real,** for example: `billing/issue-credit` issues a credit without linking `billing/credit`; `bookings/metering` applies "the member's tier discount" without linking `members/tier`; `bookings/reservation-rules` refuses a member "blocked by a long-overdue invoice" without linking `billing/invoice`; `authorization` names "check 6 of the order of checks" without linking `order-of-checks`. Four of Makerspace's 26 are "the request time", which `time` defines among other rules.
- **Noise:** an ordinary word that is also a name ("closing time", "date and time", "the project itself", "the listed values", "the refusal codes of this module"); another meaning of a name (the role `member` in a permissions line, the JSON members of a response, a member who "holds" a ticket against `rooms/hold`, a job's payment against `billing/payment`); a mention in passing or in a negation ("no invoice and no credit is involved"); a record and the report that holds it naming each other.
- **A frequency filter was tried and dropped:** treating a name that more than a quarter of the Definitions mention without a link as an ordinary word removed `time` and `members/member` in Makerspace, whose mentions were mostly real.

Two of the most named targets are broad. `time` holds "request time" among several rules, so a Definition of its own for the request time would be matched exactly by the longest name. "Member" names both a Concept and a role, which is a finding about the spec's language in its own right.

## Reading

- **The split was worth it on its own terms, and only there.** The links had claimed thirteen formats where two were used; after the split, a change to a rarely used format lists two Definitions, not 61. It does not shorten any real step's list, because the formats never changed; its gain is the change that has not happened yet.
- **Under-linking is common.** Half to three-quarters of the unlinked mentions are links the record misses: roughly 130 to 180 in Makerspace's 115 Definitions. The spec's dependencies are as complete as its author's discipline, and a text check catches much of what discipline misses, at a quarter to a half noise.
- **Link direction mattered more than the split.** One link from the general `values` back to its user `argument-values` doubled every-link reach on four Makerspace steps. Linking from the specific to the general, never back, holds inside a Package too.
- **A true hub stays a hub.** `values/identifier` is used almost everywhere, and the `Operation` Kind is the form of half the spec. Their reach is real; what helps there is keeping them small and stable, and linking them directly where they are relied on.
- **Root needs the most care.** Root may not depend on anything below it, so a module at Root pulls its own dependencies up after it, and links up hide coupling from today's measures.
- **The zone of pain names the right Packages, not yet the right moment.** Abstractness counted from Kinds is 0 outside Root, so the zone of pain is the stable core, which every domain has. It hurt where widely linked Definitions changed, which the 5× series shows and a single snapshot cannot: the cost is fan-in times change.
- **`std` has no Kind for a value.** The thirteen formats are values in the sense of domain-driven design: defined by their form alone, with no identity and no states. `std/Concept` describes one thing with attributes, relations and states, so the author put thirteen values on one Concept page, the hub this record measures. A Kind for a value might have invited one Definition each; the measurements cannot tell.

## Limits

- Two specs by one author, of one domain, one level deep. A spec with deeper Packages may show placement and Dependency Rule problems these cannot.
- Abstractness counts Definitions of Kind `Kind`. Both specs keep their Kinds at Root, so the measure has only been seen at 0 outside Root; a spec that defines Kinds in its Packages may use it more.
- Volatility comes from eight designed change steps per project, written to exercise the specs, not from a project's real history.
- The split was made by script, by rules matched against how these specs write fields. A person would likely also move Root's module into a Package of its own, which was not tried.
- Reach counts Definitions put back on the list, not what an agent does with them; whether a narrower list helps an agent build or review is untested. File inputs, which also put Definitions back on the list, were not measured here; the [3 November measurement](impact-reach-20261103.md) did that.
- No record of a real build exists for these specs, so nothing here measures evidence in `review.sum`.
- Whether an unlinked mention is real was judged by this session alone, on samples of 40 and 30 pairs, from the text around each match.

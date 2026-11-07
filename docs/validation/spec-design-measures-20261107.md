# How well the Makerspace specs are designed: measurements — 7 November 2026

This session measured the design of the two archived Makerspace specs the way code is measured: coupling, stability, abstractness, cohesion, fan-in and fan-out, and how far a change reaches. It then split the `values` page that [concept 4.0 §4](../concepts/v4-concept.md#4-is-the-spec-good) names as a hub, and measured again. It started no agent and changed no tool. The reading is this session's and decides nothing; the [guide to a good spec](../guides/good-spec.md) draws the rules from it, and the [`vet` plan](../concepts/vet-plan.md) proposes what `vet` measures.

**What the numbers show:**

- **The Package structure is sound.** No Package cycle, nothing depends on the outer `protocol` Package, dependencies point towards the stable `members` and `equipment`, and no Makerspace edge points to a less stable Package; the 5× project has three such edges.
- **Behaviour is never a hub.** An Operation is linked by at most two Definitions in Makerspace and three in the 5× project, mostly only by its own module's Interface. The links go to Concepts, Kinds and formats, which is what the stable-abstractions principle asks, per Definition.
- **Package abstractness and the distance from the main sequence say little here.** Every area Package holds its vocabulary and its operations, so its abstractness sits between 0.5 and 0.7 whatever its stability.
- **Root is the weak spot.** `values` alone is linked by 60 of 115 Definitions (269 of 419 in the 5× project), and Root also holds a module: formats, refusal codes, audit and authorization, with the only Operation at Root. Links up the tree are free, so none of this shows in Package coupling, and `vet.reach` skips Root.
- **Splitting `values` narrows reach where links were false, not elsewhere.** A change to one format then lists between 1 and 50 Definitions instead of 61. The real change steps never changed a format, so what they put back on the list stays the same; following every link, the four widest Makerspace steps fall from 85–88 % of the spec to 45–60 %, and the 5× project's from 94–96 % to 70–82 %. Most of what remains runs through the `Operation` Kind.

## Source and method

**Fixtures.** The archive point [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks): `tools/evaluation-tasks/makerspace` (steps `w0` and `c1` to `c8`, each with its finished registry from `references/<step>`) and `tools/evaluation-tasks/makerspace-large` (the 5× project, steps `w0` to `c8`, each with its own registry). Each step was copied into a scratch project folder with its manifest, model and registry, and `markitect tidy` wrote the lock. `markitect check --spec` passes on every step. Only public fixtures were read.

**Tool.** `markitect` built from Main `836ff17c` (0.22.0-dev). The links come from `markitect list --deps --json`; Package coupling and instability are the tool's own (explicit edges only, links up the tree free). `markitect vet` ran on each step as it is today.

**Throwaway scripts,** outside the repository; the method below repeats them:

| Script | Reads | Computes |
|---|---|---|
| measure | `list --deps --json` and the Definition files | The measures below, per Definition and per Package |
| split | A step's model | The split of `values`, in two variants ([below](#the-split)) |
| steps | Two consecutive steps | The Definitions a step changed, and how far that change reaches at one step, two steps and every link |

**Measures:**

| Measure | How it is computed |
|---|---|
| Definition link | A link from a Definition to a model Definition, and a Definition's project Kind. Links to Packages and Library Kinds are no Definition links: a Package link is no input of the record either |
| Fan-in, fan-out | Definitions that link to a Definition or are of it as their Kind; Definitions it links to |
| One-step reach | The Definition and its fan-in: what a change to it alone puts back on the list of `status` |
| Every-link reach | The Definition and everything that reaches it by following links backwards, as in the [3 November measurement](impact-reach-20261103.md) |
| Ca, Ce, I | Afferent and efferent coupling and instability Ce / (Ca + Ce), as `list --deps` reports them |
| A | Abstract Definitions / (abstract + concrete), by a provisional rule: Kinds, Concepts, Rules and the project Kind `Interface` are abstract, since they state what something is or must satisfy; the project Kind `Operation` and Features, Endpoints and Commands are concrete, since they state what happens. The Project and Decisions count as neither |
| D | \|A + I − 1\|, the distance from the main sequence |
| Cohesion | Connected groups of a Package's own Definitions, joined by links inside it; relational cohesion H = (links inside + 1) / Definitions; the share of the Package's links that stay inside, not counting links up the tree |
| Statements | List items and table rows in a Definition's body, a count of what a reviewer verifies |
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

| Package | Definitions | Ca | Ce | I | A | D | Groups | H | Links inside | Up | Out | Share inside |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Root | 21 | 0 | 0 | 0 | 0.95 | 0.05 | 4 | 2.00 | 41 | 0 | 0 | 1.00 |
| billing | 16 | 5 | 1 | 0.17 | 0.69 | 0.15 | 2 | 1.38 | 21 | 18 | 10 | 0.68 |
| bookings | 17 | 2 | 5 | 0.71 | 0.59 | 0.30 | 2 | 1.65 | 27 | 18 | 28 | 0.49 |
| equipment | 12 | 5 | 0 | 0.00 | 0.50 | 0.50 | 1 | 1.92 | 22 | 15 | 0 | 1.00 |
| materials | 9 | 1 | 2 | 0.67 | 0.56 | 0.22 | 1 | 1.67 | 14 | 12 | 10 | 0.58 |
| members | 11 | 5 | 0 | 0.00 | 0.55 | 0.45 | 1 | 1.82 | 19 | 14 | 0 | 1.00 |
| protocol | 6 | 0 | 8 | 1.00 | 1.00 | 1.00 | 5 | 0.33 | 1 | 4 | 9 | 0.10 |
| reporting | 6 | 1 | 3 | 0.75 | 0.67 | 0.42 | 2 | 1.17 | 6 | 7 | 16 | 0.27 |
| safety | 6 | 2 | 1 | 0.33 | 0.50 | 0.17 | 1 | 1.50 | 8 | 7 | 3 | 0.73 |
| training | 11 | 2 | 3 | 0.60 | 0.55 | 0.15 | 1 | 2.00 | 21 | 14 | 17 | 0.55 |

Root's coupling is 0 by the tool's count only: links up are free, and 9 of the 9 other Packages link Root's Definitions. Steps w0 and c4 give the same picture, with D above 0.5 only for `protocol`.

The 5× project, 47 Packages besides Root: Ce median 3 and at most 12 (`bookings`); I median 0.75; A between 0.33 and 0.67, apart from `protocol` at 1.0; D median 0.31, above 0.5 for `protocol` and six leaf Packages heavy in Rules (`assessments`, `compliance`, `dunning`, `escalations`, `reporting`, `waitlists`); H median 1.88, lowest 0.33 (`protocol`); share inside median 0.56, lowest 0.21 (`compliance`).

**Coupling and stability.** No Package depends on `protocol` in either project: it is the outer layer that wires the modules, and its Ce of 8 is what such a layer has. The stable Packages, `members` and `equipment` (I 0), are what the domain rests on. The three less stable dependencies of the 5× project differ by 0.10, 0.12 and 0.18 in instability.

**Abstractness and the main sequence.** By the provisional rule, every area Package is half to two-thirds abstract, because each holds its Concepts and Rules beside its Operations. So D mostly mirrors I: the stable `members` and `equipment` sit at D 0.45–0.5 for having Operations, and leaf Packages at about 0.5 for having Rules. In code the Package is the unit that is released and depended on; here the unit of verification is the Definition, and a change to `members/register-member` puts back only itself and its Interface, however stable `members` is. The Rules of `protocol` describe the wire protocol, which is concrete, yet count as abstract, so the rule also misjudges a Package of Rules. Per Definition the principle holds:

| Kind, Makerspace c8 | Definitions | Fan-in median | Fan-in max |
|---|---|---|---|
| `Operation` | 38 | 1 | 2 |
| `std/Rule` | 41 | 1 | 7 |
| `Interface` | 8 | 3 | 9 |
| `std/Concept` | 24 | 6 | 60 |
| `Kind` | 2 | 23.5 | 39 |

The one Operation linked from outside its module's Interface is Root's `list-audit-entries`, by `audit` and `protocol/requests`. In the 5× project an Operation has at most 3 linkers, and 14 links reach an Operation from another Package other than through an Interface. All come from Rules that add a gate or an effect to another module's operation, such as `hazmat/hazard-gate` on `materials/purchase-material` and `waitlists/promote-waitlist` on `training/enroll-in-course`: the inverted dependency the [model language guide](../guides/model-language.md#links-scope-and-visibility) describes, where the needing side states its need.

**Cohesion.** Every Package is one connected group, apart from single Rules that bind their Package by scope and link nothing in it (`billing/due-date-and-block`, `bookings/metering`, `reporting/reports`, and at Root `documentation`, `list-order` and the Kind `Interface`). No Package of either project falls into two groups of two or more Definitions each. `protocol` is the exception in kind: six Rules with one link between them, each tying other modules' Interfaces to the wire.

**Placement.** A Definition sits too high when every Definition that links to it lies below it in one child Package. None does in either project: both are one level deep, and Root's Definitions are linked from Root itself.

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
- **Definition cycles** all lie inside one Package, as they must, since a cycle across Packages is a compile error. About half are a Concept and its Rule naming each other (`bookings/reservation` and `bookings/reservations`; 13 of 29 in the 5× project); most others add an Operation. Two are at Root: `values`, `argument-values`, `money`, `computed-times`, `identifier-limits` and `integer-limits`, and `Operation`, `audit`, `list-audit-entries` and `order-of-checks`. A cycle at Root makes its members act as one hub when every link is followed.

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

- **A `values/` Package** below Root, with the thirteen formats as Concepts (`values/identifier`, `values/cent-amount`, …), each holding its row of the table, the units folded into `cent-amount` and `minute-count`. Every field line links the format it names, as in ``- `blocked`: [[values/flag|flag]]``; a field "integer from `1` to `100000000` cents" links `cent-amount` at its unit, "whether …" links `flag`, and "minutes of …" links `minute-count`. The header ``**Fields**, in the formats of [[values]]`` stays and now names the Package, which is no input. Root's own Definitions may not link into the Package, so `audit-entry`, `identifier-limits`, `money`, `time` and `project` keep a link to the Package. The formats link nothing, so the old back link from `values` to `argument-values`, and with it Root's six-member cycle, is gone. Root's registry page lists the five formats its file realizes instead of `values`.
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

**What a change to one format lists** (one-step reach, the format included): before the split, 61 Definitions for any format in Makerspace and 270 in the 5× project.

| Format | Makerspace | 5× |
|---|---|---|
| identifier | 50 | 243 |
| text | 19 | 65 |
| cent amount | 17 | 63 |
| timestamp | 15 | 35 |
| count | 5 | 16 |
| minute count | 5 | 7 |
| date | 4 | 21 |
| flag | 4 | 17 |
| month | 4 | 11 |
| percent | 3 | 3 |
| clock time | 2 | 2 |
| operation name | 2 | 2 |
| audit target | 1 | 1 |
| **Median** | **4** | **16** |

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

No step changed `values`, so one step, which is what `status` lists, reaches the same Definitions before and after. Every link narrows wherever it passed through `values`.

**What is left of every-link reach** runs through the `Operation` Kind. Stopping at it alone takes 44 of the 55 Definitions out of Makerspace c1 after the split, 37 of 60 out of c5 and 39 of 59 out of c6; in the 5× project 221 of 298 on c2 and 144 of 334 on c7, with `members/member` next (22 to 32 on c5 and c6).

**What the split costs:**

- **Files:** 13 Definitions for 1, with the same 354 words. A new Package needs a registry page with its first paths, or `vet.no_paths` warns.
- **Links and lines:** Makerspace gains 57 Definition links and 199 written links, on 198 lines in 55 files; the 5× project gains 203 Definition links and 735 written links, on 734 lines in 264 files. Root's registry page changes two rows.
- **Reading:** every field line carries a link, and its rendered label still reads "identifier" or "cent amount". Field lists read as before; the text of a field changed only for "whether …", "the wire name …" and "minutes of …", which name their format now.
- **Root cannot follow** in the `values/` variant. Five Root Definitions keep only the Package link, so a change to a format no longer lists them, and `values/audit-target`, whose only user was Root's `audit-entry`, is linked by nothing: `check` reports it unused until a registry row realizes it. With the formats at Root instead, every link is exact, but Root grows to 33 Definitions and `vet.package_size` warns.
- **New warnings in the `values/` variant:** `vet.reach` on four formats (`values/identifier` reaches 10 of 11 Packages), `bookings` reaches Ce 6 (`vet.coupling`), and in the 5× project every Package gains one efferent edge, which shifts instabilities: three new `vet.unstable_dependency` warnings appear, with differences of 0.04, 0.06 and 0.10.

## Reading

- **The split was worth it on its own terms.** The links had claimed thirteen formats where two were used; after the split, a change to a rarely used format lists two Definitions, not 61, and the widest every-link reaches halve on Makerspace. It does not shorten any real step's list, because the formats never changed; its gain is the change that has not happened yet.
- **A true hub stays a hub.** `values/identifier` is used almost everywhere, and the `Operation` Kind is the form of half the spec. Their reach is real; what helps there is keeping them small and stable, and linking them directly where they are relied on.
- **Root needs the most care.** Root may not depend on anything below it, so a module at Root pulls its own dependencies up after it, and links up hide coupling from today's measures.
- **Package-level abstractness is the wrong grain.** The Definition is the unit of verification, so the stable-abstractions principle is best read per Definition: what has fan-in should state what something is, not what happens.

## Limits

- Two specs by one author, of one domain, one level deep. A spec with deeper Packages may show placement and Dependency Rule problems these cannot.
- The abstractness rule is provisional and relies on what the Kinds mean; `vet` knows no Kind's meaning today ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)).
- The split was made by script, by rules matched against how these specs write fields. A person would likely also move Root's module into a Package of its own, which was not tried.
- Reach counts Definitions put back on the list, not what an agent does with them; whether a narrower list helps an agent build or review is untested. File inputs, which also put Definitions back on the list, were not measured here; the [3 November measurement](impact-reach-20261103.md) did that.
- No record of a real build exists for these specs, so nothing here measures evidence in `review.sum`.

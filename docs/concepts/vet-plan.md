# What `vet` measures: the plan

**Status:** the plan's record. Proposed on 7 November 2026; on 8 November 2026 the owner accepted its seven decisions with changes ([DEC-102](register.md#dec-102-what-vet-measures)), and they were built and released as 0.23.0. [Decisions](#decisions) says what was decided and how it was built; the rest of the page keeps the analysis as it was proposed, with a note where the decision differs. It answers part of [OQ-002](register.md#open-questions) and the question [concept 4.0 §4](v4-concept.md#4-is-the-spec-good) left open: the details of `vet`, its further measures and thresholds. It follows the rules the owner set for it on 7 November 2026:

- no change to the language and no new field;
- no explicit layers: the Package tree is the Dependency Rule, as in Go;
- a Package's abstractness comes from its Kinds, as Martin counts interfaces;
- the tool never treats a Kind specially by its name: what depends on a Kind's meaning is for the agents, who read its text.

**Basis:** the [measurements of 7 November 2026](../validation/spec-design-measures-20261107.md) on the two archived Makerspace specs, 115 and 419 Definitions, and the [guide to a good spec](../guides/good-spec.md), which says what each measure is for. [The model language guide](../guides/model-language.md#vet-warnings-and-notes) owns what `vet` does.

## In brief

As decided and built ([DEC-102](register.md#dec-102-what-vet-measures)):

- **Unlinked mentions,** a priority: a Definition's name in another Definition's prose with no link to it. Half to three-quarters of what it finds on the Makerspace specs are real missing links, roughly 130 to 180 in Makerspace alone, and the rest are ordinary words and other meanings of a name. So it is a note, with a one-line count in every run, and the auditor triages the candidates for the owner.
- **`vet.hub` replaces `vet.reach`.** It counts Definitions, counts Root, and leaves Kinds out. On both specs it warns about `values` and `members/member`, the hub that cost most re-verification in the 5× change steps. `vet.reach` warned about Interfaces on Makerspace, which are linked widely by design, and missed `values`.
- **Martin's abstractness and distance,** A and D, are in `list --deps`. The notes for the zones wait until `vet` can see change.
- **The other warnings stay** as they were, `vet.unstable_dependency` included, Martin's rule exactly.
- **Notes,** which `--strict` shows and which never change the exit code: unlinked mentions, statements per Definition, placement, and hotspots from the record's count of revisions. The CRAP analogue waits for real data.
- **Agents judge what a count cannot,** and only ever warn ([AI-assisted measures](#ai-assisted-measures)); the reviewer counts cases while it marks.
- **No `std/Value`:** the owner decided against a Kind for values and against a central page of values.

## What `vet` knows of Kinds

`vet` knows one Kind: Core's `Kind`, which every Kind Definition has as its own Kind. A Definition of Kind `Kind` counts towards its Package's abstractness, as the owner set. Two further uses go beyond that, and the owner accepted both with decisions 1 and 2: `vet.hub` leaves Kinds out, since a change to a Kind concerns all its Definitions by design, and `vet.unlinked_mention` never names a Kind, since Kinds' names are mostly ordinary words. Nothing else in `vet` depends on which Kind a Definition has. A project's own Kinds and `std`'s are treated alike, and a measure that would need to know what a Kind means, such as which Kinds are concrete, is not proposed.

## The measures

N is the number of the spec's model Definitions. Fan-in counts the Definitions that link to a Definition or are of it as Kind. A is the share of a Package's own Definitions that are of Kind `Kind`, I is Ce / (Ca + Ce), and D is |A + I − 1|. Hits are on Makerspace c8 and the 5× project's c8, as the archive holds them.

### Warnings

`vet` prints these always and exits 1 on them only with `--strict`, as today. They are the measures that were precise on both specs.

| Code | Warns when | Threshold | Hits: Makerspace, 5× | Change |
|---|---|---|---|---|
| `vet.hub` | A Definition not of Kind `Kind`, Root included, whose change alone puts more than a fifth of the spec back on the list: fan-in + 1 > max(10, 0.2 N) | 20 %, at least 10 | `values` (61 of 115), `members/member` (25); `values` (270 of 419), `members/member` (91) | New, replaces `vet.reach` |
| `vet.coupling` | A Package depends on more than 5 Packages (Ce) | 5 | `protocol` (8); 6 Packages, `bookings` 12 at most | Keep |
| `vet.unstable_dependency` | A sideways edge goes from a Package to a less stable one: Martin's Stable Dependencies Principle | — | none; 3 (differences 0.10, 0.12 and 0.18) | Keep |
| `vet.definition_size` | A Definition's body has more than 1000 words | 1000 | none (largest 430); none (largest 995) | Keep |
| `vet.package_size` | A Package has more than 25 Definitions of its own, Root included | 25 | none (Root 21); none | Keep |
| `vet.thin_package` | A Package has no Definitions of its own and one child | — | none; none | Keep |
| `vet.no_paths` | A Package other than Root claims no path pattern | — | as today | Keep |

**`list --deps`** adds each Package's A and D beside its instability, in its text and as `abstractness` and `distance` in its JSON. Built as proposed.

**Why `vet.hub` and not `vet.reach`.** `vet.reach` counts the Packages a change reaches and skips Root. On Makerspace it warns about the Interfaces of `billing`, `equipment` and `members`, whose purpose is to be used by other modules, and stays silent about `values`, which 60 Definitions link while using a median of 2 of its 13 formats. `vet.hub` counts Definitions, which is what `status` lists, and counts Root, where the worst hub sits. At 20 % it also names `members/member`: in the 5× project's eight change steps, `members/member` changed once and put 88 Definitions back on the list, and the changes of nine widely linked Definitions caused 248 of the steps' 468 items, items no other change reached. At 25 % it would name `values` alone, which never changed. After `values` is split, it names `values/identifier`, a true hub, which the author keeps with one line of reason, as `/markitect-design` asks of any warning, and in the 5× project still `members/member`.

**Why the zones split.** Both specs keep their Kinds at Root, so A is 0 in every other Package and D is 1 − I. The zone of pain then holds Root and the stable core: `billing`, `equipment` and `members` in Makerspace, eight Packages in the 5× project. That is where change hurts, but Martin counts it harmful only for what changes often, and in Makerspace nothing widely linked there changed. A snapshot cannot tell, so the zone of pain is a note until `vet` can see change. The zone of uselessness, Kinds in an unstable Package that little uses, needs no such knowledge, but neither spec has a hit, so its precision is untested and it starts as a note too. **Decided:** neither zone note is built yet; both wait until change frequency exists, which the record's revisions now begin to give.

**Why no margin on stability.** Splitting `values` into a Package gave every module one more efferent edge and shifted instabilities, which raised three new `vet.unstable_dependency` warnings, with differences of 0.035, 0.062 and exactly 0.100. The three that stood before differ by exactly 0.100, 0.125 and 0.183. No margin separates the two sets, so the rule stays Martin's, exactly as `vet` has it today.

### Notes, shown with `--strict`

These have a known blind spot or no calibration yet, so they never change the exit code, as notes already do in Markitect's commands ([exit codes](../guides/model-language.md#diagnostics-json-and-exit-codes)). `vet --strict` prints them after the warnings and still exits 1 on warnings only; `--json` always carries them, with `severity` `note`. Plain `vet` prints one line that counts them and says to run `vet --strict` to see them, so it stays silent only when there are no notes either. **Decided:** `vet.unlinked_mention`, `vet.statements` and `vet.placement` are built as below; the zones and `vet.risk` wait; and `vet.hotspot` was added, from the record's revisions ([Decisions](#decisions), 7).

| Code | Notes when | Threshold | Hits: Makerspace, 5× | Why a note |
|---|---|---|---|---|
| `vet.unlinked_mention` | A Definition's title, or the last segment of its ID, appears in another Definition's prose with no link to it, after the filters [below](#unlinked-mentions) | — | 233 pairs from 86 Definitions; 863 from 337 | A quarter to a half is noise, which no filter without a word list removed |
| `vet.zone_of_uselessness` | A Package is abstract and unstable: D > 0.7 with A + I > 1, so Kinds that little depends on | 0.7 | none; none | No hit on either spec, so its precision is untested |
| `vet.zone_of_pain` | A Package is stable and concrete: D > 0.7 with A + I < 1 | 0.7 | Root, `billing`, `equipment`, `members`; Root and 8 Packages | Every domain has a stable core; it hurts only where it changes, which a snapshot cannot see |
| `vet.statements` | A Definition has more than 20 statements: list items and table rows in its body | 20 | 2 (`reporting/revenue-report` 25, `bookings/create-reservation` 22); 8 (`list-order` 67 at most) | The anchor and the architecture are long by nature (5× `architecture` 60, `project` 26) |
| `vet.placement` | Every Definition that links to a Definition lies below it in one child Package, so it could move down to that child | — | none; none | A Rule that binds by scope may be linked from one child only and still belong higher. Both specs are one level deep, so untested |
| `vet.risk` | A Definition recorded `ok` or `skip` at its current inputs scores above 30 on c² × (1 − e)³ + c, as defined under [AI-assisted measures](#ai-assisted-measures) | 30 | no record exists | Uncalibrated, and it rests on the reviewer's own counts, which a reviewer could set too high or too low. It also makes `vet` read `.markitect/review.sum` beside the spec |

### Unlinked mentions

In code a missing import does not compile; in a spec, prose can name a concept without linking it, and the record then misses a re-verification. `vet.unlinked_mention` reads the text and the structure only, so it works for any language a spec is written in:

- **Names:** a Definition's title and the last segment of its ID, hyphens read as spaces, matched as whole words without regard to case, with up to two trailing letters for inflection.
- **Where:** the prose of every other Definition, without its front matter, its own title, code blocks, code spans and links.
- **Not reported:** where the Definition links the named one or is of it as Kind; where it may not link it at all (into a child, or to an internal Definition from outside); where a longer name covers the words, the Definition's own name included; where a linked Definition's name covers them ("invoices" beside a linked `billing/invoice`); and where the named one is a Kind, whose name is mostly an ordinary word. A name the Definition's Kind links is reported: a change to it puts the Kind back on the list, not the Definition.
- **Counted apart:** a mention whose link would close a Package cycle, or add a Package dependency the spec does not have. These go against the structure or beyond it. Many are ordinary words (157 of the 5× project's 363 new-dependency pairs are "order" against `purchasing/order`), but some are real dependencies the author never declared, which the author should see. `--json` lists them with `against` or `new_dependency`, and the summary line counts them.
- **One line per Definition** that names others, at the line of the first mention: `billing/issue-credit.md:12: vet.unlinked_mention: names billing/credit and time without linking them`.

On Makerspace c8 it reports 233 pairs and counts 50 apart, 44 of them against the structure; on the 5× project 863, and 552 apart, 189 against it. Two judges of a sample of 40 and of 30 found 52 to 65 % and 57 to 70 % real, up to three-quarters with the doubtful ones ([record](../validation/spec-design-measures-20261107.md#unlinked-mentions)). The noise is ordinary words that are also names ("closing time" against `time`), other meanings of a name (the role "member" against `members/member`), and mentions in passing. A spec reduces it by naming its concepts precisely, for instance a Definition of its own for the request time. A frequency filter was tried and dropped, since it removed mostly real mentions. Once a spec's own measurements show the noise low, the owner may make it a warning.

### Measured and not proposed

| Measure | Why not |
|---|---|
| Change hotspots: a Definition's fan-in times how often it changes | The strongest sign of real cost in the 5× series, but it needs history, and the tool reads no Git ([DEC-097](register.md#dec-097-markitect-uses-no-git)). **Decided** on 8 November 2026: the record counts revisions, and `vet.hotspot` is built from them as a note ([Decisions](#decisions), 7); the zone of pain may become a warning for volatile Packages once real records show it |
| Abstractness by what Kinds mean (Concepts and Rules abstract, Operations concrete) | Needs `vet` to know Kinds by name, and told good Packages from bad ones no better than instability |
| Every-link reach | The record follows one step; the every-link reach is for the author to read, not to warn about. `list --deps` and `doc` already show the links |
| Fan-out | Its largest values are Interfaces, which list their module's operations by design |
| Relational cohesion H, links leaving a Definition's Package | Low values were the outer `protocol` and `reporting` and operations that use another area's Concepts, all legitimate |
| Package cohesion: two or more groups of two or more Definitions with no link between them | Its one hit, `bookings` at steps w0 and c1, was three Rules that bind the Package by scope, not a second area |
| A margin on `vet.unstable_dependency` | The shifts a shared Package causes and the warnings that stood before overlap ([above](#warnings)) |
| Definition cycles | All lie inside one Package; many are a Concept and its Rule naming each other |
| A Kind that links Definitions other than Kinds | One hit per spec, the `Operation` Kind, whose links reflect what its Definitions truly rely on; whether it mixes levels needs judgment |

## The Dependency Rule without layers

The owner settled this on 7 November 2026: there are no explicit layers and no mechanism by which one Kind refines another. Markitect has Packages and links like Go, and Go has no layers either.

- **The tree carries it.** Links up are free and links down are forbidden (`link.downward`), so dependencies already point from the specific to the general.
- **Sideways edges** are checked by the cycle ban (`package.cycle`) and by `vet.unstable_dependency`, Martin's Stable Dependencies Principle from the graph alone.
- **A project's own limits** use what exists: `internal: true` for what a Package keeps to itself, which `check` enforces (`link.internal`), and an ordinary Rule for the rest, which the reviewer checks. Makerspace marks 19 Rules `internal: true`.

On Makerspace the rule holds without more: no Package depends on the outer `protocol`, the area modules reach each other's operations only through Interfaces, and the stability measure finds no violation. The 5× project has three small stability violations, and three Operations that build on another module's operation rather than its Interface, such as `access/enter-door` on `billing/get-account`; a project that wants to forbid that writes it as a Rule, or marks the operations `internal: true` and lets other modules use the Interface.

## A new `std` Kind: Value

**Not adopted.** The owner decided against `std/Value` on 8 November 2026, and against a central page of values: general knowledge is not specified, and a value rule of the project's own, its format, range, unit or equality, is stated once, as a small Definition of its own, in the only Package that uses it or at the lowest common parent of those that share it ([Decisions](#decisions), 6; [good spec](../guides/good-spec.md#say-it-once-link-exactly-what-you-use)). The proposal follows as it was made.

New `std` Kinds come only where the measurements show a gap and an established concept fills it. No candidate meets that rule in full. One comes close, and is put to the owner as a possible exception, not recommended on the measurements alone.

**The concept.** A value object in domain-driven design (Eric Evans, *Domain-Driven Design*, 2003): an element of the domain defined by its content alone, with no identity of its own and no lifecycle; two values with the same content are equal, and a value is replaced, not changed. Its counterpart is the entity, which has an identity that persists while its attributes change.

**The gap.** `std/Concept` explains "one thing of the project's domain: what it is, what it consists of, how it relates to other Concepts, and which states it can be in": an entity in all but name, and Makerspace's Concepts read so ("A record is identified by `memberId`"). `std` has no Kind for a value. Makerspace wrote its thirteen value formats, identifiers, amounts, timestamps and the like, as one Concept page, `values`, which became the largest hub of both specs: 60 and 269 linkers, using a median of 2 and 1 of its formats. The gap is argued from the Kinds' texts, not measured as harm: the split works as thirteen `std/Concept` files, and a Kind for values might, but need not, have led the author to one Definition each. What the measurements do support is the draft's last rule: `values` linked back to `argument-values`, a Rule that uses it, and that one link doubled every-link reach on four Makerspace steps. A Value links only the Values it is made of.

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

## AI-assisted measures

**Decided as a trial** ([Decisions](#decisions), 7): the counts on `mark` and the record's revisions are built, and `vet.risk` waits for the record of real builds. The aim is that the principles hold in practice, "good enough" and with no guarantee, through deterministic checks and through agents reviewing exactly these properties.

**Two levels.**

| Level | Judges | What | Effect |
|---|---|---|---|
| `vet` | The tool, deterministically, in CI too | Everything structural on this page: coupling, stability, abstractness, hubs, size, unlinked mentions | Warnings and notes as above |
| Agents | The reviewer, while it verifies, and the auditor, the read-only role `/markitect-audit` starts ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role)) | The semantic properties a count cannot see: how complex a Definition is, contradictions, vague statements, mixed levels of abstraction, terms used inconsistently | The reviewer records the counts below, and a contradiction or a gap it finds as a `spec-problem` on the nearest Definition, the record being the only store ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), point 4). The auditor judges vagueness, mixed levels and inconsistent terms, and reports them to the owner. Neither adds a block |

**Complexity by counting, not rating.** The reviewer reads every Definition it verifies anyway. While it marks, it may record how many distinct cases, conditions and exceptions the Definition demands, and how many of them its evidence demonstrates, as optional fields of `mark`: `markitect mark fees/waiver@3f9a1c ok -e src/fees.go:40 -e tests/fees_test.go:12 --cases 7 --covered 5`. A count is checkable, where a rating of "complex" is not. The record keeps it on the Definition's line in `review.sum`, so it is computed only when the Definition's inputs change, exactly when the verification is. This changes `mark` and the record's format, not the spec language. Until a count exists, `vet` falls back to the statements it counts itself.

**A CRAP analogue on the record.** `vet.risk` scores each Definition recorded `ok` or `skip` at its current inputs: *risk = c² × (1 − e)³ + c*. *c* is 1 + the cases, conditions and exceptions the reviewer counted for the Definition, or 1 + its statements until a count exists. *e* is the share of those cases the reviewer found its evidence demonstrates, or, until it records that, the evidence pointers divided by *c*, at most 1. A `skip` has *e* 0. A `fail`, or a Definition with no verdict at its current inputs, is on the list of `status` already and is not scored. Tests stay the project's ([DEC-100](register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), point 6), so the tool marks no evidence as a test; the reviewer, which may run the project's tests as evidence, says how many cases its evidence demonstrates. A Definition of 7 cases (*c* 8) with 2 of them demonstrated scores 64 × 0.36 + 8 ≈ 31, with 4 about 13; with none, any Definition of more than four cases scores above 30. `vet --strict` lists those above 30 as notes.

**Validation.** Keep it only if the data bear it out, and measure it so that the verdict cannot decide its own score. Score each review from the case count the reviewer records in it and from the covered cases of the verdict before it, none for a first review, so that a `fail` does not raise its own score. That needs the count on every verdict, `fail` included. Once real builds have recorded at least 30 `fail` verdicts, and at least 10 in each group, compare how often a review ends in `fail` when its score was above 30 with how often it does otherwise. If the first is not about twice the second, the count and `vet.risk` go again. One bias remains: the count comes from the same review as the verdict, so a reviewer that fails a Definition may count more cases; comparing the counts one Definition gets across reviews shows how large that is.

**What it costs.** No agent run of its own: the count comes from the review that happens anyway. Two optional flags on `mark`, two optional fields in the record, and a few sentences in the reviewer's agent file. The auditor's findings go to the owner in its report, as DEC-100 has it, and into no file of the tool.

## Decisions

The owner decided on 8 November 2026 ([DEC-102](register.md#dec-102-what-vet-measures)). Each proposal, then what was decided and built:

1. **`vet.unlinked_mention`,** proposed as a note with a one-line summary in every run, to become a warning once its noise is low; mentions against or beyond the structure counted apart. **Accepted, and more:** `/markitect-audit` and the auditor's agent file have the auditor triage the candidates into real missing links and ordinary words and report the real ones to the owner; the architect then adds the links. Built so; on the archived specs it gives exactly the counts of the [record](../validation/spec-design-measures-20261107.md#unlinked-mentions).
2. **`vet.hub` replaces `vet.reach`:** Definitions, Root included, Kinds left out, above a fifth of the spec and at least 10. **Accepted** and built.
3. **Martin's abstractness and distance:** A and D in `list --deps`, both zones as notes. **Accepted in part:** A and D in `list --deps`, text and JSON, as information only; the zone notes wait until change frequency exists. `vet.hub` and `vet.unlinked_mention` leave Kinds out, as proposed.
4. **Notes for `vet.statements` and `vet.placement`,** shown with `--strict` and never changing the exit code. **Accepted** and built, with the thresholds above as provisional.
5. **`vet.risk`, the CRAP analogue.** **Waits** for real data; not built.
6. **`std/Value`.** **Not adopted,** and no central page of values: general knowledge is not specified, and a project's own value rules are stated once, at the right place. [What makes a good spec](../guides/good-spec.md#say-it-once-link-exactly-what-you-use) and the `markitect-design` skill state the rule.
7. **AI-assisted measures** as a trial. **Accepted, and more:** `mark` takes `--cases N` and `--covered M`, stored on the record's line, and the reviewer's agent file says to count cases while marking. The record also counts revisions per Definition, how often its own text changed between verdicts, as a basis for hotspots without Git, and `vet.hotspot` notes, with `--strict`, a Definition whose fan-in times revisions is above max(20, N / 10). That threshold was calibrated on the eight change steps of both archived projects: on the 5× project it names the five Definitions whose changes cost most, from `members/member` (90) to `ledger/postings` (46), with nothing between 46 and 20; on Makerspace, where no widely linked Definition changed, the largest is 14 and it names nothing ([DEC-102](register.md#dec-102-what-vet-measures) has the figures).

Built in 0.23.0: `internal/analysis` and its tests, `internal/run` (the counts and revisions on the record's line), `list --deps`, `vet` and `mark` on the command line, the auditor and reviewer agent files, `/markitect-audit` and the design skill, and the guides. No `std` Kind changed.

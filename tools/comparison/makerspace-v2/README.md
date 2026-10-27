# Makerspace v2

The Makerspace W0 program remodeled with the standard Libraries and an owner-decided architecture: workstream W6 of the [canonical model track](../../../docs/design/model-track-charter.md).

**Role.** Pilot-only material: Stage 1 (pilot) of the evaluation and the development and calibration of its instruments. It is not portfolio material; confirmatory portfolio models come from an independent model author. The frozen [phase-b/](../phase-b) and [phase-c/](../phase-c) material stays untouched.

**Source.** The neutral requirements [phase-c/requirements/w0.md](../phase-c/requirements/w0.md) as of the Phase C materials freeze `4840fb20f52c1e0e0367e909d833919ca8a215c6`, plus the architecture the owner approved on 27 October 2026, and the change steps [phase-c/requirements/c1.md](../phase-c/requirements/c1.md) to c9.md applied in order, as the Evaluation session chose, so that the old-versus-new Apply comparison runs on the same neutral deltas as Phase C. That comparison is pilot or uncounted material and runs under its own bounded dispatch.

**Versions.** v2.1, 27 October 2026, adds the machine rate type `rate-cents-per-minute` to the equipment Interface's types, in W0 and every change step, with the rendered documents and section 16 updated. In v2.0, `bookings/reservation` recorded the rate at check-in in a field of that type although the Interface did not expose it. The Assessments of W0 and C1 approved that; C2's Assessment caught it as a model blocker. Exposing the type follows the approved architecture: the Interface already said that other modules ask for a machine's current rate. The old-versus-new comparison gate ran on v2.0; the new arm's C2 reruns on v2.1 after the Codex quota resets on 2 November 2026. `model_analysis.py lint` now reports such a use as `interface`.

## Contents

| Path | Holds |
| --- | --- |
| `model.yaml`, `model-lock.yaml`, `execution.yaml`, `model/` | The project model as it goes into a repository's `.markitect/`. `execution.yaml` maps each Package to its folder under `realization/`, selects `Practice.Rule` by `appliesTo` as modeled scope, and binds the build and test checks |
| `libraries/` | Vendored copies of `markitect-core` 1.2.0 ([testdata/core](../../../testdata/core)) and `markitect-values`, `-data`, `-api`, `-practice`, `-architecture` 1.0.0 ([libraries/](../../../libraries)); the lock pins their digests |
| `check.py` | Compiles the model with the real `markitect check` and verifies the lock and that each vendored Library equals its source |
| `spec/render.py`, `spec/requirements-w0.md` | The single-agent arm's document, rendered from the model: every body (body links reduced to their text, two phrases about "the model" reworded), every typed field and each rule's scope as fixed sentences and tables. `check` re-renders and compares |
| `requirements/w0.md` | The equivalence partner: the neutral requirements with section 16 "Architecture", rendered from the model, and §2.1 pointing to it |
| `c1/model/` … `c9/model/`, `spec/requirements-c1.md` … `-c9.md`, `requirements/c1.md` … `c9.md` | The change steps: one complete model snapshot per step (each includes all earlier changes; they share the manifest, lock and execution configuration), its rendered single-agent document, and the neutral delta, which is phase-c's text plus the section 16 lines the step changes. C9 is byte-identical to C8 |

```text
python -I -B tools/comparison/makerspace-v2/check.py check [STEP]
python -I -B tools/comparison/makerspace-v2/spec/render.py check [--step STEP]
```

## Architecture

The language does not allow Definitions directly in the model root, so the owner's root-level decisions live in the project Package `makerspace`: the `Practice.Project`, the test and documentation obligations, one `DependencyRule` per Package and three `Placement`s. Ten flat Packages sit beneath it; each is one Manager and one Go package.

| Package | Holds | May depend on |
| --- | --- | --- |
| `policies` | Value types, units, roles, shared refusal codes, the audit log, and the authorization, audit, money, time, ordering and limit rules | nothing |
| `members` | Users, members, tiers, standing | policies |
| `equipment` | Families, machines, maintenance | policies |
| `billing` | Charges, invoices, payments, credit, accounts | members, policies |
| `training` | Courses, enrollments, certifications | members, equipment, billing, policies |
| `bookings` | Reservations, check-in, metering, schedules; lockouts from C2 | members, training, equipment, billing, policies; safety from C2 |
| `materials` | Stock and purchases | members, billing, policies |
| `safety` | Incidents | equipment, policies |
| `reporting` | Utilization and revenue reports | equipment, bookings, billing, policies |
| `protocol` | Request and response lines, invalid requests, argument checks, initial state, order of checks | all of the above |

Architecture rules for structural checks (oracle authors derive their checks from these Definitions):

- **Dependency rules** (`model/makerspace/*-dependencies.yaml`): a Package may depend only on itself and the Packages listed; Libraries, Core and the Go standard library are not counted.
- **Placements:** authorization (checks 2, 4 and 6), audit entries and the request-time order are realized once in `protocol`.
- **Interfaces** (`model/makerspace/<area>/<area>-interface.yaml`): each area's operations and record types that others may use; other Packages use nothing else of it.
- **Dependency inversion:** equipment needs to know whether an active reservation overlaps a maintenance window but may not depend on bookings; it declares the need and `protocol` supplies it from bookings.

## Measured on this model

| Criterion | Result |
| --- | --- |
| Compiles | Yes, 181 Definitions in 11 Packages, 12 Managers with Root |
| H1 rules reach consumers | Every Manager receives 24 to 31 Rules by modeled scope |
| H2 no Package cycles | Holds; all model edges follow the dependency rules |
| H4 shapes typed | All 37 operations, 20 records and 25 refusal codes are typed |
| S5 peer coupling | Of the 22 Package edges leaving the eight area Packages, 14 (64 %) go to another area and 8 to policies. The cross-area rules of the requirements (reservations need members, training and equipment; usage, courses and purchases charge billing) make this high |
| S3 stability | policies 0.00, members 0.28, equipment 0.45, billing 0.79, the other areas 0.87 to 1.00; edges point toward stabler Packages |
| S1 size | Prose words per body, table and code lines excluded: none over 300; 11 above 150, each one topic carried over whole from the requirements (overview, time, computed times, integer limits, argument values, order of checks, invoice generation, invalid requests, utilization report, money, request limits) |
| R3 restatement | The requirements' "cross-area rules at a glance" were dropped from the overview: they restate the reservation and invoice rules |
| G5 | Root still receives no context; the project Package `makerspace` sees its own rules |
| S4 change locality | Modules whose Definitions a step changes: C1 4, C2 7, C3 1, C4 6, C5 5, C6 4, C7 3, C8 3, C9 0. C1, C2, C4, C5, C6 and C8 touch policies and count as cross-cutting. C7 is description only, yet touches 3 modules because renamed billing rules change the links that point to them. Phase C's old model: 2, 7, 1, 4, 6, 3, 8, 3 |
| Change steps | Every step compiles, keeps its dependency rules and passes the renderer's checks |

## Equivalence record

- **Section 16 is derived from the model.** `requirements/w0.md` §16 is rendered from the model's architecture, so the equivalence check on the architecture compares the model with itself. That is acceptable for pilot material only; for portfolio material, v0.3 §4.1a requires the reverse order: the requirements author writes the architecture section first, the neutrality review approves the module cut, and the model author derives the model from it.
- **Who chose the module cut.** The model-track session proposed the cut (ten flat modules, `policies` as the shared kernel, `protocol` as the outer layer) and the owner approved it. For the portfolio, whether a good engineer would choose this cut independent of Markitect is the neutrality question of §4.1a and must be answered by someone outside the Markitect side.
- **Accepted difference.** The model drops w0 §1's "cross-area rules at a glance" (R3: it restates the reservation and invoice rules); the neutral copy keeps it. Neither arm loses information.
- **Arm-equal wording.** `spec/render.py check` refuses Markitect vocabulary (Package, Definition, Kind, Library aliases, Manager, "model", body-link brackets) in the single-agent document and in §16.
- **Change deltas carry section 16 lines.** `requirements/c2.md`, `c5.md` and `c6.md` add the architecture lines those steps change (C2: the bookings dependency on safety and the bookings and billing interfaces; C5: the billing interface; C6: the training interface), rendered from the model like §16 itself: pilot only. The owner decided on 27 October 2026 that C2's lockouts live in bookings.
- **Comparable data.** The single-agent documents here carry section 16 and its changes; Phase C's arm B never had them, so old arm B results are no baseline for either side of this material. The old-versus-new comparison runs Markitect on the Phase C model and on this model on one current source; a single-agent reference would have to be re-run on these documents. Never mix in old arm B data.
- **Task typing.** C1, C2, C5 and C6 change interface or dependency lines of section 16, so in the pilot they count as partly architecture changes besides their behavior change.
- **To watch in the pilot.** The `policies` Manager's context of 187 items may drive cost or failures on the Markitect side.

## Findings for the Libraries

- **Internal capabilities have no Kind.** Areas use each other through capabilities that are not wire operations ("is this certification valid on date D?"). `Architecture.Interface` lists only `Api.Operation`s and types, so these live in Interface bodies as prose.
- **The shared kernel draws a large context.** The `policies` Manager receives 187 items, most of them incoming references from every operation.
- **`Api.results` says result fields come "in order".** Read alone, that could clash with JSON member order being free (the model's records-in-responses rule says so); the next `markitect-api` version should say the order is the documentation order only.

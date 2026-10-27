# Makerspace v2

The Makerspace W0 program remodeled with the standard Libraries and an owner-decided architecture: workstream W6 of the [canonical model track](../../../docs/design/model-track-charter.md).

**Role.** Pilot-only material: Stage 1 (pilot) of the evaluation and the development and calibration of its instruments. It is not portfolio material; confirmatory portfolio models come from an independent model author. The frozen [phase-b/](../phase-b) and [phase-c/](../phase-c) material stays untouched.

**Source.** The neutral requirements [phase-c/requirements/w0.md](../phase-c/requirements/w0.md) as of the Phase C materials freeze `4840fb20f52c1e0e0367e909d833919ca8a215c6`, plus the architecture the owner approved on 27 October 2026. Change steps are not remodeled yet; the pilot needs C1 (cross-cutting), C7 (architecture change) and C9 (no change) in a later slice.

## Contents

| Path | Holds |
| --- | --- |
| `model.yaml`, `model-lock.yaml`, `execution.yaml`, `model/` | The project model as it goes into a repository's `.markitect/`. `execution.yaml` maps each Package to its folder under `realization/`, selects `Practice.Rule` by `appliesTo` as modeled scope, and binds the build and test checks |
| `libraries/` | Vendored copies of `markitect-core` 1.2.0 ([testdata/core](../../../testdata/core)) and `markitect-values`, `-data`, `-api`, `-practice`, `-architecture` 1.0.0 ([libraries/](../../../libraries)); the lock pins their digests |
| `check.py` | Compiles the model with the real `markitect check` and verifies the lock and that each vendored Library equals its source |
| `spec/render.py`, `spec/requirements-w0.md` | The single-agent arm's document, rendered from the model: every body (body links reduced to their text, two phrases about "the model" reworded), every typed field and each rule's scope as fixed sentences and tables. `check` re-renders and compares |
| `requirements/w0.md` | The equivalence partner: the neutral requirements with section 16 "Architecture", rendered from the model, and §2.1 pointing to it |

```text
python -I -B tools/comparison/makerspace-v2/check.py check
python -I -B tools/comparison/makerspace-v2/spec/render.py check
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
| `bookings` | Reservations, check-in, metering, schedules | members, training, equipment, billing, policies |
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

## Equivalence record

- **Section 16 is derived from the model.** `requirements/w0.md` §16 is rendered from the model's architecture, so the equivalence check on the architecture compares the model with itself. That is acceptable for pilot material only; for portfolio material, v0.3 §4.1a requires the reverse order: the requirements author writes the architecture section first, the neutrality review approves the module cut, and the model author derives the model from it.
- **Who chose the module cut.** The model-track session proposed the cut (ten flat modules, `policies` as the shared kernel, `protocol` as the outer layer) and the owner approved it. For the portfolio, whether a good engineer would choose this cut independent of Markitect is the neutrality question of §4.1a and must be answered by someone outside the Markitect side.
- **Accepted difference.** The model drops w0 §1's "cross-area rules at a glance" (R3: it restates the reservation and invoice rules); the neutral copy keeps it. Neither arm loses information.
- **Arm-equal wording.** `spec/render.py check` refuses Markitect vocabulary (Package, Definition, Kind, Library aliases, Manager, "model", body-link brackets) in the single-agent document and in §16.
- **To watch in the pilot.** The `policies` Manager's context of 187 items may drive cost or failures on the Markitect side.

## Findings for the Libraries

- **Internal capabilities have no Kind.** Areas use each other through capabilities that are not wire operations ("is this certification valid on date D?"). `Architecture.Interface` lists only `Api.Operation`s and types, so these live in Interface bodies as prose.
- **The shared kernel draws a large context.** The `policies` Manager receives 187 items, most of them incoming references from every operation.
- **`Api.results` says result fields come "in order".** Read alone, that could clash with JSON member order being free (the model's records-in-responses rule says so); the next `markitect-api` version should say the order is the documentation order only.

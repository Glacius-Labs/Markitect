# Execute-first Apply — 19 October 2026

Recorder's note: direct owner discussion in the Codex chat **Workflow**, thread `01a1542d-641c-7643-86e9-ffcf003e1420`. Owner quotations preserve their original wording. Assistant proposals and interpretation are labeled separately. [DEC-058](../register.md#dec-058-apply-always-executes-before-independent-verify) records acceptance; the [Apply concept](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-concept.md#shared-working-tree-and-reconciliation-cycle) owns the target. This record supplies no implementation or runtime evidence.

## Owner challenges initial Verify

> Brauchen wir in Fall 1 wirklich eine initiale Verify-Runde? Wenn noch keine Artefakte existieren, kann Root doch direkt die gemeinsamen Grundlagen schaffen und dann seine Kinder loslegen lassen.

Recorder's summary of the assistant response: it initially proposed a direct Execute entry for absent artifacts while keeping initial Verify for existing artifacts. The owner rejected introducing a separate route:

> Wir wollen aber keinen Sonderweg haben. Der Grundprozess soll gleich bleiben. Wie bekommen wir das einheitlich hin?

## Common process proposed

Recorder's summary of the subsequent assistant proposal, not verbatim owner text:

- Every Apply starts with Execute and integration after normal structural preparation, whether artifacts are absent or present.
- Execute includes each Manager's local assessment against the model: preserve valid realization, create missing artifacts, repair deviations and remove obsolete realization under the existing rules. No change is a legitimate result; the process does not require an edit or an artificial initial finding.
- Root handles shared prerequisites in its own area, and each parent coordinates its direct children. Needed prerequisites determine work readiness; there is no global bootstrap phase or requirement to complete every parent task before independent child work can progress.
- After integration, quiesce writers, capture one complete immutable candidate and assess it in fresh independent Verify conversations plus required checks. Findings lead to owner corrections and another candidate/Verify round. Independence and publication requirements remain.
- Removing initial Verify is common to all Apply starts. The tradeoff is that an already conforming stand also receives Execute's local assessment before independent Verify.

## Owner acceptance and Root question

> Ja, halten wir das so fest. Aktualisiere die Docs: Apply startet immer mit Execute und Integration, danach folgt die unabhängige Verify-Runde. Keine initiale Verify-Runde und kein Sonderweg für Fall 1. Noch keine Implementierung.
>
> Das heißt dann vermutlich auch, dass generell der Apply immer im Root beginnt und nach unten wandert korrekt?

Recorder's interpretation and assistant clarification: the phase order is explicitly accepted. Starting at Root and proceeding through direct parent/child relationships describes the entry and coordination of the existing permanent organization, with integration and child assessment leading back upward. It does not grant Root a choice to omit required Managers, require all parent work to finish before a child begins, or select a new detailed scheduler. Only Root asks the human; owners retain their artifacts and the canonical model stays protected.

The accepted change amends DEC-056's initial-Verify sequence. Runtime, tests, execution settings, fixtures and the canonical self-model still describe that earlier implementation and remain unchanged by this documentation task. This instruction authorizes no implementation, provider run, demonstration, release or migration.

## Owner confirms parent foundations before child work

Recorder's note: subsequent discussion in the same thread on 19 October 2026. The owner challenged the earlier assistant wording that independent child work need not wait for all parent work:

> Warte, ich denke Kinder sollten immer auf ihre Eltern warten, sonst bauen sie auf keiner Grundlage auf oder?

Recorder's summary of the assistant clarification and case-1 walkthrough, not verbatim owner text:

- The parent first provides the actual foundation its direct children need and communicates what is available; only then does their realization begin. A promised future foundation is insufficient.
- Root supplies the common root-owned basis, such as the Library's Go module. Siblings may then work in parallel where their further dependencies permit it. Missing sibling interfaces are still supplied by their actual owners through parent coordination.
- Lending similarly provides its own loan-query foundation before Reminders begins. Managers keep their own artifact ownership; parents do not prebuild child-owned stubs or implementations.
- Parent completion includes integration of child results. Children therefore wait for the needed foundation, not the parent's final integrated completion. Results lead upward; subsequent independent Verify examines the complete immutable candidate, children before parents.

The owner accepted the clarification and requested documentation:

> Ja, halte das so in den Docs fest: Eltern schaffen zuerst die benötigte Grundlage, dann beginnen ihre Kinder. Ergebnisse werden nach oben integriert und anschließend unabhängig verifiziert. Weiterhin keine Implementierung.

Recorder's interpretation: this makes the parent-before-child start dependency explicit within DEC-058's single Apply process. An existing suitable foundation can be preserved; no artificial edit, initial Verify, special bootstrap process, per-file approval or new readiness schema is introduced. The documentation preserves the distinction between accepted target and unchanged implementation and grants no runtime, test, settings, canonical-model or fixture changes.

## Owner confirms existing foundation assessment

Recorder's note: subsequent case-2 discussion in the same thread on 19 October 2026. The owner asked how Apply works with existing artifacts and how parents determine whether their foundations still fit.

Recorder's summary of the assistant walkthrough, not verbatim owner text:

- Parents compare the current model, actual files and interfaces, and known requirements of their direct children within Execute. They preserve a matching foundation or coordinate its owner's correction or supplementation before dependent child realization begins, then identify the concrete available basis.
- In the illustrative five-to-three loan-limit change, Root can preserve its still-suitable Go module. Lending assesses whether its existing loan-query contract still supports Reminders, preserves it when suitable and communicates its availability before Reminders begins. A policy limit change does not automatically require an interface change.
- All required Managers reconcile their responsibilities, including unchanged areas. Children's later observations can reveal a missed dependency; parents coordinate owner corrections and inform affected children when a basis already in use changes. Upward integration and subsequent independent Verify of one complete immutable candidate remain required.
- Existing artifacts change the work performed inside Execute, not the common Apply process or its order. Local parent assessment is not a prior independent Verify and cannot alone establish the correctness of the final integrated candidate.

The owner accepted the case-2 walkthrough and requested documentation:

> Ja, halte auch den Ablauf für Fall 2 so in den Docs fest: Eltern prüfen bestehende Grundlagen im Execute und erhalten oder korrigieren sie, bevor ihre Kinder beginnen. Weiterhin keine Implementierung.

Recorder's interpretation: this clarifies how DEC-058 applies to an existing realization, preserving the previously accepted parent-before-child start, ownership, integration, independent Verify and documentation-only boundary. No runtime, test, settings, canonical-model or fixture change is authorized here.

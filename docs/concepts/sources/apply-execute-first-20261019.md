# Execute-first Apply — 19 October 2026

Recorder's note: direct owner discussion in the Codex chat **Workflow**, thread `01a1542d-641c-7643-86e9-ffcf003e1420`. Owner quotations preserve their original wording. Assistant proposals and interpretation are labeled separately. [DEC-058](../register.md#dec-058-apply-always-executes-before-independent-verify) records acceptance; the [Apply concept](../../design/canonical-model/manager-apply-concept.md#shared-working-tree-and-reconciliation-cycle) owns the target. This record supplies no implementation or runtime evidence.

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

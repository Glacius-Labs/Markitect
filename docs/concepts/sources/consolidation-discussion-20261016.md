# Owner instructions for the concept consolidation

Date: 16 October 2026, evening. Verbatim owner messages from a Claude Code session, in their original order.

Recorder's note: The session first reviewed the documents, the implementation state and current work, then consolidated the Apply concept on branch `codex/apply-concept-consolidation`. The owner also asked to include the concept branch `docs/manager-realization-north-star`, which was merged at `3ebe3c6e6d283b998020350334e24a7303b089b7`.

## Overview and assessment

> Schaue dir alle Dokumente an. Ließ dir das neue Konzept durch, die Vision und alles mögliche was du an Kontext finden kannst. Dann schaue dir an, was der aktuelle Implementierungsstand ist und woran aktuell gearbeitet wird. Anschließend gib mir einen Überblick und ein Assessment.

> Betrachte auch den docs branch der die ganze Zeit geupdated wird und beziehe die Dokumente dort mit ein

## Straighten and consolidate the concept

> Kannst du das Konzept geradeziehen und enstprechend ausdünnen und aufräumen und konsolidieren?

## Simplify and bring the result to main

> Du darfst das Konzept auch gerne vereinfachen, verbessern oder vereinheitlichen falls du Dinge findest, die man besser machen könnte. Bringe den sauberen Stand dann auf main

Recorder's note: Under this authorization the consolidation simplified the first-Apply contracts without changing the owner decisions:

- Managers no longer allocate budgets. A configured completion reserve and one admission rule protect reviews and checks.
- The optional delivery deadline was deferred.
- Each Apply invocation is its own assignment. Successor runs arise only when the basis changes mid-run, so the two-Apply demonstration is two ordinary invocations.
- A finding has three states: open, resolved and not applicable.
- Dependent work starts from an accepted, composed delivery, without exposure gating.
- One gate scheme, P00 to P12, is keyed to the delivery packages.

## Quality over effort and compatibility

> Beachte auch dass wir hohen Wert auf Sauberkeit, architektonische Qualität, Qualität generell, Einfachheit und Stabilität legen. Aufwand sollte keine Rolle spielen. Ebenso nicht Kompatibilität mit alten Versionen.

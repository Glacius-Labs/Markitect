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

## Adopt the recommendations

> Analysiere nochmal das Konzept mit Bezug auf die Vision und allen Infos die du hast und erstelle eine große Liste an Vorschlägen für Überarbeitungen, Refinements, Cleanups und Erweiterungen des Konzepts. Definiere auch gerne aus, was noch underdefined ist

> Übernimm deine Empfehlungen

Recorder's note: The assistant's list ended with a recommendation to decide twelve items now, before the organization, realization and run packages start. The owner adopted them:

- A1: put the library's callable interface into M1 as `library/callable-interface`, so no binding constraint lives outside the model.
- A2: binding rules belong in the model; configuration and instructions carry only execution choices and operational guidance.
- A3: a shared rule's scope comes from the model; configuration only names the Kind and Property that carry scope.
- A4: Apply deletes or rejustifies artifacts whose Definition was removed, and migrates registry links of moved Definitions.
- B1: the Manager whose Package requires a project-level artifact owns it; the root owns only artifacts no Definition requires; a missing skeleton is delivered first.
- B2: the root records the delivery order with reasons, based on the reference graph the Host supplies.
- B4: the registry lives in `.markitect/registry/`, one file per Manager.
- B5: Markitect's own initial registry is created at the cutover and validated by `check` until Apply exists.
- B8: Apply starts from the committed model with a clean working tree, writes only the working tree and never commits; the owner's commit accepts the result.
- B10: a closed catalogue of Host tools per role.
- C1: no separate final verifier Agent; a deterministic Host evidence check plus the root's integration review.
- C2: no Superseded handoff; a run that needs a changed basis finishes Incomplete, and a continuation follows.

The other items of the list remain proposals.

## Adopt all recommendations

> Stelle sicher, dass du deine Empfehlungen alle in das Konzept einarbeitest und nichts vergisst

Recorder's note: The owner extended the adoption to every item of the assistant's list. The items beyond the first twelve were incorporated as follows:

- A5: a path toward computed scope, with the data recorded from the first run on (concept, "Toward computed scope").
- A6: DEC-018 and DEC-021 to DEC-023 are marked as written for the earlier Project model; their mapping is OQ-009.
- A7: optional owner approval before publication for selected Packages (configuration, execution, DEC-035).
- A8: project-wide conventions are model Rules with project scope, checked by the root's integration review.
- A9: Definition assessments name the tests that would fail on violation; reviewers judge their discrimination.
- B3: each area owns its internal API; consumers use only it.
- B6: a fixed Definition-assessment schema; every Definition needs a disposition other than unresolved.
- B7: a context size limit that stops preparation instead of truncating.
- B9: blocking and advisory findings; only blocking ones count.
- B11: product-owned, versioned role prompts whose identity is recorded.
- B12: the observed model edit is compared with the approved patch; deviations are shown, not blocked.
- B13: automatic retries only for calls that provably sent nothing, each charged.
- B14: questions in the baseline demonstration count as interventions; questions about rules M1 resolves count as defects.
- B15: a session step limit with a replacement session from the journal summary.
- B16: the first command surface keeps DEC-017's verb-table principles.
- C3: composition is a disjoint union; overlap is a Host error.
- C4: one journal stream per repository.
- C5: the mandate is a section of the context package.
- C6: four question states.
- C7: pass-through intermediate Packages start no Agent.
- C8: no Kind guidance in the first version.
- D1: two-phase, model-first reviews.
- D2: optional batching of no-change reviews.
- D3: reviewers from a different model family once a second adapter exists.
- D4: a fixed set of run measures.
- D5: the Playground report lists every cross-Package obligation of M1 with its evidence.
- D6: a glossary in the design map.
- D7: an implementation that deviates from a contract changes it in the same pull request.
- D8: register notes on CPT-004, OQ-006 and ENH-003.
- E1 to E10: recorded as later extensions in the north star and linked from the register entries they serve.

## Polish and integrate

> Wenn du fertig bist, poliere alles nochmal auf und sorge dafür dass das Konzept sauber ist. Dann bringe es auf main

Recorder's note: The independent reviews before integration led to two alignments. B1's clause on the root was aligned with DEC-006 and DEC-025: every owned artifact needs a Definition, and the common parent of the requiring Packages, possibly the root, owns an artifact several of them require. C2 replaces the successor runs mentioned in the first recorder's note; runs after a basis change are continuations.

# Technical risk derivation by Managers and reviewers

Date: 17 October 2026. Direct owner messages in the project-analysis conversation. Owner wording is preserved verbatim; the context below is a recorder's note.

## Owner question

> Muss ich solche technischen Risiken wirklich einzeln modellieren, oder sollte Markitect aus allgemeinen Qualitätsregeln selbst ableiten, dass Nebenläufigkeit und Veröffentlichung korrekt abgesichert werden müssen?

## Owner instruction

> Arbeite diese Präzisierungen bitte in das bestehende Konzept und die Rollen-Templates ein, ohne neue Mechanismen einzuführen. Die eigenständige Ableitung technischer Risiken soll klar zur Pflicht von Managern und Reviewern gehören.

## Recorded context

Recorder's note: the preceding assessment distinguished the existing handling of recognized risks from the less explicit duty to discover technical failure modes. Its examples concerned stale snapshot writes and an edit racing with publication. These examples illustrate possible violations of existing obligations; they are not new project policies or a requirement that owners enumerate implementation defects.

The clarification makes active investigation part of the existing Manager and independent-review duties, preserves model-first expectations and adds implementation-specific inquiry after candidate access. Existing reports, Definition assessments, findings, checks and rework carry the reasoning and evidence. No separate risk mechanism or new execution authority is proposed. A distinguishing future observation would be discovery and correction without owner coaching; this instruction and template change do not establish that result.

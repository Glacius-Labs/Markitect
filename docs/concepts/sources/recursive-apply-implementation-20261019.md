# Recursive Apply implementation dispatch — owner, 19 October 2026

Recorder's note: verbatim excerpts from the owner's dispatch of the Claude implementation session for [DEC-056](../register.md#dec-056-recursive-reconciliation-with-a-shared-workspace). The [Apply concept](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026) owns the behavior; this source records the implementation authority, the runtime model choice and the demonstration bounds.

## Authority

> Implementiere das auf Main beschlossene vereinfachte rekursive Apply-Konzept in Markitect und demonstriere es anschließend am Bibliotheksbeispiel.

> Der aktuelle Konzeptabschnitt und DEC-056 haben Vorrang vor ausdrücklich als überholt gekennzeichneten Detailverträgen. Implementiere die neue Richtung direkt; erhalte den alten Apparat nicht aus Kompatibilitätsgründen.

> Baue keine neuen Fallmatrizen, Reservierungsprotokolle, Delivery-Scopes, Rejection-Counter-Ausnahmen oder zusätzliche dauerhafte Reviewer-Hierarchie.

## Runtime model choice

> Verwende Sonnet für selbst gestartete Implementierungs-, Manager-, Verify- und Review-Agenten.

> Keine Codex-/OpenAI-Inferenzaufrufe und kein stiller Modell-Fallback.

> Diese Anweisung ersetzt ausdrücklich die frühere Luna-Vorgabe im Bibliotheks-Walkthrough.

Recorder's note: this replaces the Luna runtime preference recorded in the [library walkthrough](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-library-flow.md#current-library-walkthrough) for the demonstration. It does not remove the Codex adapter from the product.

## Demonstration bounds

> M1 → A1; danach ausschließlich Modellgrenze fünf auf drei ändern; M2 gegen tatsächliches A1 → A2.

> Ein vollständiger realer Testlauf einschließlich beider Applies, Vorbereitung, Agentstarts, Reviews, Checks und Korrekturen hat ein gemeinsames hartes Wall-Clock-Limit von 30 Minuten.

> Erzwinge die Grenze außerhalb der Agentprompts. Plane Abbruch/Cleanup innerhalb der Grenze ein. Unteragenten und Neustarts erhalten kein neues Gesamtbudget.

> Bei Überschreitung: Prozesse beenden, vorhandene Evidenz sichern und den Lauf als Incomplete berichten. Keine automatische Kette weiterer 30-Minuten-Läufe.

> Behaupte keinen vollständigen Apply, wenn nur ein externes Skript die wesentliche Orchestrierung übernommen hat.

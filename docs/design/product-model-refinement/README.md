# Markitect: Grundmodell und aktueller Konzeptentwurf

Aktualisiert am 13. Oktober 2026. Diese Map trennt vereinbarte Produktgrundlagen, bestätigte Entwurfsrichtung, offene Detailvorschläge und datierte Vorarbeiten. Sie ist kein Implementierungs- oder Testnachweis.

## Grundmodell

| Hauptblock | Funktion |
| --- | --- |
| Kanonisches Modell | Maßgebliche, explizite Beschreibung des geltenden Projekts, seiner Konzepte, Regeln und Realisierungsverpflichtungen. |
| Repository | Dateien, welche die geltende Beschreibung realisieren. |
| Apply | Abstrakter Mechanismus, der Typisierung und Struktur für systematisch koordinierte Agentenrealisierung nutzt. |

Die Arbeitsidee bleibt, das Modell zu pflegen und seine Realisierung durchführen zu lassen. Die interne Zuordnung dieses abstrakten Apply-Begriffs zu konkreten CLI-Kommandos ist damit nicht festgelegt.

Der gewählte Verantwortungsdateiname ist `mandate.yaml`. `Concept`, `Rule`, `UseCase` sowie die Architektur-Kinds `System`, `Component`, `Module`, `Interface` sind gewählt; die bisherige Category-Unterscheidung entfällt dafür. Die Umbenennung des vorhandenen Kinds `Manager` ist nicht beschlossen.

## Aktueller Arbeitsstand und Lesereihenfolge

1. [Konsolidierter Konzept- und Umsetzungsentwurf](canonical-model-proposal.md): kleine Struktur, nominale Regeln, verbindliche Prosa, explizite Beziehungen und methodische Reviews.
2. [Vollständiges aktuelles Bibliotheksmodell](library-model-example.md): Manifest, Mandat, Fachdefinitionen und getrennte Frontend-/Backend-/Datenbank-/Prüf-/Dokumentationsbeiträge.
3. [Architektur als YAML](library-architecture-example.md): System, Components, Modules, Interfaces, ihre Beziehungen und Architekturregel; zusammen mit dem Fachmodell 34 Definitionen.
4. [Konkreter Änderungsdurchlauf](library-change-walkthrough.md): Überfälligkeitsregel, fehlende Abhängigkeit, Modellreview, konservative Wirkung und neue Prüfaufträge.
5. [Fester Main-Abgleich vom 11. Oktober](canonical-model-main-comparison.md): Ausgangspunkt für einen beim Implementierungsstart zu erneuernden Vergleich.

Der methodische Ausgangspunkt und die nominalen Fach-/Architektur-Kinds sind miteinander vereinbar. Zusammensetzung, Interface-Zusammenarbeit, Fachzuordnung, Artefaktrealisierung und Mandate bleiben eigene Beziehungen. Wie tief Fachproperties und Bedingungen formalisiert werden, bleibt eine separate Frage. Umfangreiche Facetten sind eine mögliche spätere Erweiterung. Prosa bleibt verbindlich; ihre Bedeutung und die Vollständigkeit der Beziehungen werden nicht allein vom Strukturcompiler bewiesen.

Offen sind insbesondere Zielmengen und Umsetzung gemischter Referenzen, Versions-/Migrationsvertrag, Bedarf für weitere Beschreibungstypen sowie konkrete Fachdetails des Beispiels. `vNext` ist ein Platzhalter. Das neue Beispiel wurde nicht angelegt, kompiliert oder ausgeführt.

## Übergabe an den Implementer

Der Proposal enthält [Designbegründungen](canonical-model-proposal.md#81-begründung-der-designentscheidungen), [Umsetzungspakete](canonical-model-proposal.md#82-umsetzungsvorbereitung-für-den-implementer) und [geplante Nachweise](canonical-model-proposal.md#83-prüfplan-für-die-spätere-implementierung).

Die Produktbasis dieses Worktrees ist älter als der letzte Main-Abgleich. Die Umsetzung startet auf einem neuen festen aktuellen Main-Stand und berücksichtigt dessen laufende Arbeit. Die Dokumentationsänderungen werden gezielt übernommen. Dieser Auftrag beginnt keine Implementierung, Studien, Veröffentlichung oder Backlogaktivierung.

## Leitfrage und datierte Vorarbeiten

Die [Arbeitsnotiz](system-description-notes.md) hält die [Leitfrage](system-description-notes.md#leitfrage), ursprüngliche Nutzerformulierungen und abgeleiteten Arbeitsstand fest: Wie beschreibt man ein Repository für ein Softwaresystem?

| Vorarbeit | Heutige Einordnung |
| --- | --- |
| [Ausführlicher Sprachentwurf](canonical-model-draft.md) | Frühere Ausdrucks- und Vollständigkeitsüberlegungen; die aktuelle Empfehlung steht im konsolidierten Proposal. |
| [C#-Modellierung](csharp-model-design.md) | Beschreibendes Metamodell und Gegenprobe; keine vorgeschriebene Kind-/Klassenliste oder Produktimplementierung. |
| [Facetten-/Dateisystembeispiel](library-filesystem-example.md) | Datiertes umfangreicheres Syntaxbeispiel; optionale Formalisierung, keine aktuelle Mindeststruktur. |
| [Methodische v1alpha1-Vorstufe](library-methodical-alternative.md) | Frühere Statement-/Category-Form; durch das aktuelle nominale YAML-Beispiel als Arbeitsdarstellung abgelöst. |

Die illustrative C#-Quelle bleibt unverändert. Frühere Quellen werden nicht nachträglich als implementiert oder als akzeptierter aktueller Formatvertrag dargestellt.

## Herkunft

Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`, Grundlagen vom 10. Oktober, Konsolidierung ausdrücklich am 12. Oktober 2026 beauftragt.

- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Ursprüngliche Branchbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Eingang der vollständigen Konsolidierung: `07e03215c968302361f5acedc8e69bd573ac95be`; beim Beginn sauber.
- Eingang der Kind-Aktualisierung: `5d18e8f4438cc1d24a10882f04b10501d65db20e`; beim Beginn sauber.
- Eingang der Architekturergänzung: `ec7ba72d25c7f17cedcfb61e6b77f11d5cddb5ed`; beim Beginn sauber, am 13. Oktober ausdrücklich beauftragt.
- Letzter gesonderter Implementierungsvergleich: `32afd1a57e715130d056e98e433721c6e7a10278` vom 11. Oktober; heute nicht neu als aktueller Stand verifiziert.

Kanonische Produktdokumente bleiben [Vision](../../vision.md), [Konzeptregister](../../concepts/register.md) und [Architektur](../../architecture.md). Dieser Konzeptordner setzt ihre aktuellen Verträge und den koordinierten Modell-Owner nicht automatisch außer Kraft.

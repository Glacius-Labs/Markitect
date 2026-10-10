# Markitect: vereinbartes Grundmodell

Dieses Dokument enthält ausschließlich die gemeinsam vereinbarten Grundlagen der Produktdiskussion. Weitere Inhalte werden erst aufgenommen, wenn wir uns darauf geeinigt haben.

## Grundmodell

Markitect besteht auf der grundlegenden Produktebene aus drei Hauptblöcken:

| Hauptblock | Funktion |
| --- | --- |
| Kanonisches Modell | Beschreibt typisiert und explizit, wie das Projekt aussehen und funktionieren soll und welche Konzepte es gibt. Es ist die maßgebliche Grundlage für die Realisierung. |
| Repository | Enthält die Dateien, die das kanonische Modell realisieren. |
| Apply | Nutzt die Typisierung und Struktur des Modells, um AI-Agenten systematisch zu koordinieren und das Modell im Repository zu realisieren. |

Die Arbeitsidee ist, das kanonische Modell zu pflegen und seine Realisierung durch Apply durchführen zu lassen.

`Apply` bezeichnet hier den abstrakten Realisierungsmechanismus. Seine interne Architektur und die Zuordnung zu bestehenden Komponenten oder CLI-Kommandos sind damit noch nicht festgelegt.

## Herkunft

Vereinbart im Markitect-Produktgespräch am 10. Oktober 2026, Chat `01a121f1-b948-7050-ae5d-9921b99db9c0`.

- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Ausgangspunkt: `origin/main`, Commit `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`; Worktree beim Anlegen sauber.

Dies ist eine vereinbarte Produktgrundlage aus dem Gespräch, kein Nachweis des implementierten Stands. Die bestehenden kanonischen Produktdokumente bleiben in [Vision](../../vision.md), [Konzeptregister](../../concepts/register.md) und [Architektur](../../architecture.md).

## Zugehörige Arbeitsnotiz

Auf ausdrücklichen Wunsch des Nutzers ist der Arbeitsansatz mit allgemeinen Beschreibungsfragen, einer Bibliotheksverwaltung als Beispiel und seinen Gedanken zu Programmiersprachen sowie Hierarchie und Graph in einer [separaten Notiz](system-description-notes.md) festgehalten. Deren Begriffskandidaten und offene Analogien sind keine zusätzlichen vereinbarten Grundlagen.

Die Notiz hält zuerst die [zentrale Leitfrage](system-description-notes.md#leitfrage) fest: Wie beschreibt man ein Repository für ein Softwaresystem? Anschließend folgt der [abgeleitete Arbeitsstand](system-description-notes.md#abgeleiteter-arbeitsstand) mit Beschreibungsfragen, Artefaktbeiträgen, einem vorgeschlagenen Maßstab für ausreichende Beschreibung und dem weiteren Vorgehen.

## Beauftragter Entwurf

Der [Proposal zum Aufbau des kanonischen Modells](canonical-model-proposal.md) vom 11. Oktober 2026 plant die Beschreibungssprache anhand der Leitfrage und des festgehaltenen Arbeitsstands. Sein [Abgleich mit frisch abgerufenem Main](canonical-model-main-comparison.md) trennt dokumentierte Verträge, implementierten Code, gelesene Testquellen und offene Arbeit am festen Commit `32afd1a57e715130d056e98e433721c6e7a10278`. Beide Dokumente sind Vorschlag beziehungsweise Quellenvergleich; keine angenommenen Modelländerungen oder Implementierungsaufträge.

Für die gemeinsame Umsetzung mit dem Implementer enthält der Proposal die [Begründung der Designentscheidungen](canonical-model-proposal.md#81-begründung-der-designentscheidungen), eine [geordnete Umsetzungsvorbereitung](canonical-model-proposal.md#82-umsetzungsvorbereitung-für-den-implementer) und einen [Prüfplan](canonical-model-proposal.md#83-prüfplan-für-die-spätere-implementierung). Offene Formatentscheidungen bleiben ausdrücklich sichtbar; die Produktbasis wird beim Implementierungsstart erneut aus Main bestimmt.

Der [Entwurf der kanonischen Beschreibungssprache](canonical-model-draft.md) beschreibt Inhalte, Ausdrucksmittel, Hierarchie und Beziehungen, Realisierung ohne manuelle Aufzählung jeder Datei sowie Verify als mögliche Rückkopplung. Abschnitt 2 hält das vom Nutzer gewünschte Vorgehen fest. Status: Entwurf zur Diskussion; seine Einzelvorschläge sind noch nicht angenommen.

Der [konkrete C#-Modellierungsentwurf](csharp-model-design.md) arbeitet `SoftwareSystem` und `Repository` am Bibliotheksbeispiel aus. Er enthält ihre Unterobjekte, Beziehungen, Invarianten, vorgeschlagene Dienstgrenzen und Patterns sowie die Ableitung eines Modellvertrags und einer Marketingbotschaft. Die zugehörigen C#-Dateien sind illustrative Dokumentationsartefakte, kein Produktcode. Auch diese Vorschläge sind noch nicht angenommen.

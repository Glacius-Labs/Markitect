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

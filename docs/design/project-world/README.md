# Markitect als gemeinsames Projektmodell

Stand: 8. Oktober 2026, Europe/Berlin.

## Auftrag und Status

Dieses Paket hält die direkte Produktdiskussion fest und beginnt die Planung in einem eigenen Worktree. Der Nutzer hat das Festhalten und Planen beauftragt. Die unten genannten Richtungsentscheidungen stammen aus dieser Diskussion; die konkreten Modellverträge und Umsetzungsschritte bleiben ein überprüfbarer Entwurf. Es wird keine neue API, Laufzeitfähigkeit oder Veröffentlichung behauptet.

Markitect soll ein gemeinsames, kanonisches Verständnis des Projekts ausdrücken: Ziele, Fachbegriffe, Konzepte, Use Cases, Regeln, Architektur, Organisation und Arbeitsabläufe. Verantwortliche lassen diese Vorgaben innerhalb ihrer Befugnisse umsetzen und prüfen. Das Repository enthält ihre konkreten Realisierungen und die kanonischen Modelleingaben selbst.

## Festgehaltene Richtungsentscheidungen

1. **Gemeinsame Fachsprache:** Bedeutende Begriffe werden definiert und in Konzepten, Regeln und Use Cases ausdrücklich verwendet. Begriffsdefinition, strukturelle Validierung und Prüfung der Umsetzung sind getrennte Fähigkeiten. Nicht jedes Detail muss sofort maschinell durchgesetzt werden.
2. **Management als öffentliches Modell:** Arbeit wird nach Verantwortungsbereichen organisiert. Bereiche können sich beliebig tief in weitere Bereiche gliedern. Jeder Bereich behält die Verantwortung für seinen Gesamtauftrag und die Integration seiner Teilaufträge.
3. **Eindeutige Verantwortung:** Jedes deklarierte Modul, jeder Namespace und jeder Verantwortungsbereich hat genau einen auflösbaren Verantwortlichen. Mehrere Mitwirkende und Prüfer sind möglich. Eine konkrete Agenteninstanz ist nicht automatisch der dauerhafte Verantwortliche.
4. **Getrennte, verbundene Modellbereiche:** Fachliche Konzepte und Use Cases erhalten einen klaren Platz; Organisation, Prozesse und Workflows ebenfalls. Code, Tests, Dokumentation, Pipelines und Konfiguration sind miteinander verknüpfte Realisierungen.
5. **Begriffswechsel:** „Projektion“ wird als öffentlicher Zielbegriff aufgegeben. Er wird nicht pauschal durch „Bereich“ ersetzt. Verantwortung, Zielvorgabe, Arbeitsauftrag, Fähigkeit und Artefakt bekommen jeweils ihren eigenen Begriff. Bestehende technische Namen werden ausschließlich in der Übergangsplanung bezeichnet.
6. **Vom Entwurf zur Verbindlichkeit:** Brainstorming bleibt frei und kann unvollständig oder widersprüchlich sein. Akzeptierte Modellinhalte sind davon erkennbar getrennt. Nicht jede Notiz benötigt einen formalen Vorgang.
7. **Gewünschter Arbeitsablauf:** Projekt aufbauen, Modell entwickeln oder ändern, betroffene Arbeit ableiten, delegiert umsetzen, unabhängig prüfen und das Repository erneut mit dem Soll abgleichen. Reparaturen bei unverändertem Soll bleiben möglich.

## Dokumente und Eigentümerschaft

| Dokument | Verantworteter Inhalt |
|---|---|
| [Zielmodell](model.md) | Begriffe, vorgeschlagene Verträge, Verantwortungsregeln und durchgehendes Order-Beispiel |
| [Umsetzungsplan](implementation-plan.md) | Quellabgleich, endliche Schritte, Abnahmeszenarien und offene Entscheidungen |

Dieses Paket ist der Besitzer dieser neuen Planung. Es ändert keine veröffentlichten Verträge. Die bestehende [Architektur](../../architecture.md) beschreibt ihren jeweiligen Quellstand; die [Roadmap](../../implementation-plan.md) verweist auf die neue Planung. Der [Government-Entwurf](../government/README.md) und der [Architect-Abgleich](../architect-government-checkpoint.md) liefern Vorarbeit. Deren zusätzliche Institutionen werden hier nicht automatisch übernommen.

## Arbeitsgrundlage

- Worktree: `C:/Users/Consiliari/.codex/worktrees/project-world-planning/Markitect`.
- Feature-Branch: `codex/project-world-planning`.
- Exakte Planungsbasis: `53be4b98ece0cd9e797530f18c29ec8ae683f3fd` aus `codex/government-assessment`; dadurch bleibt die bisherige Diskussion verfügbar.
- Separat betrachtete technische Hauptlinie: lokaler Remote-Ref `origin/main` bei `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Der Ref wurde für diese Planung lokal gelesen, nicht neu vom Server abgefragt.

Die Planungsbasis enthält ältere Produktquellen neben neueren Entwurfs- und Koordinationsdokumenten. Sie ist kein aktueller Implementierungsstand der Hauptlinie. Vor Produktcodeänderungen müssen die Dokumente auf eine ausdrücklich gewählte technische Basis übertragen und die tatsächlichen Verträge dort abgeglichen werden. Andere aktive Worktrees werden durch diese Planung nicht verändert.

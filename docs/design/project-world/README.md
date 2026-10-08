# Markitect als gemeinsames Projektmodell

Stand: 8. Oktober 2026, Europe/Berlin.

## Auftrag und Status

Dieses Paket hält die direkte Produktdiskussion fest und beginnt die Planung in einem eigenen Worktree. Der Nutzer hat das Festhalten und Planen beauftragt. Die unten genannten Richtungsentscheidungen stammen aus dieser Diskussion; die konkreten Modellverträge und Umsetzungsschritte bleiben ein überprüfbarer Entwurf. Es wird keine neue API, Laufzeitfähigkeit oder Veröffentlichung behauptet.

Markitect soll ein gemeinsames, kanonisches Verständnis des Projekts ausdrücken: Ziele, Fachbegriffe, Konzepte, Use Cases, Regeln, Architektur, Organisation und Arbeitsabläufe. Verantwortliche lassen diese Vorgaben innerhalb ihrer Befugnisse umsetzen und prüfen. Das Repository enthält ihre konkreten Realisierungen und die kanonischen Modelleingaben selbst.

## Festgehaltene Richtungsentscheidungen

1. **Gemeinsame Fachsprache:** Bedeutende Begriffe werden definiert und in Konzepten, Regeln und Use Cases ausdrücklich verwendet. Begriffsdefinition, strukturelle Validierung und Prüfung der Umsetzung sind getrennte Fähigkeiten. Nicht jedes Detail muss sofort maschinell durchgesetzt werden.
2. **Management als öffentliches Modell:** Arbeit wird nach rekursiven verantworteten Modulen organisiert. „Modul“ ist der aktuelle Namensvorschlag für die gemeinsame Einheit aus fachlicher Gliederung, Namensraum und Verantwortung. Jeder Manager behält die Verantwortung für seinen Gesamtauftrag und die Integration seiner Teilaufträge.
3. **Eindeutige Verantwortung:** Jedes deklarierte Modul hat genau einen auflösbaren Manager. Interne Namensräume und Modellinhalte sind über diese ausdrücklich gewählte Gliederung eindeutig verantwortet; es entstehen keine parallelen lokalen Ownershipregister. Mehrere Mitwirkende und Prüfer sind möglich. Eine konkrete Agenteninstanz ist nicht automatisch die dauerhafte Rolle.
4. **Verbundene Sichten auf fachliche Slices:** Konzepte, Use Cases, Regeln und Erwartungen bleiben bei ihrer Fachlichkeit. Architektur, Organisation, Prozesse und Workflows sind ebenfalls verantwortete Modellinhalte. Code, Tests, Dokumentation, Pipelines und Konfiguration sind miteinander verknüpfte Realisierungen.
5. **Begriffswechsel:** „Projektion“ wird als öffentlicher Zielbegriff aufgegeben. Er wird nicht pauschal durch „Bereich“ ersetzt. Verantwortung, Zielvorgabe, Arbeitsauftrag, Fähigkeit und Artefakt bekommen jeweils ihren eigenen Begriff. Bestehende technische Namen werden ausschließlich in der Übergangsplanung bezeichnet.
6. **Vom Entwurf zur Verbindlichkeit:** Brainstorming bleibt frei und kann unvollständig oder widersprüchlich sein. Akzeptierte Modellinhalte sind davon erkennbar getrennt. Nicht jede Notiz benötigt einen formalen Vorgang.
7. **Gewünschter Arbeitsablauf:** Projekt aufbauen, Modell entwickeln oder ändern, betroffene Arbeit ableiten, delegiert umsetzen, unabhängig prüfen und das Repository erneut mit dem Soll abgleichen. Reparaturen bei unverändertem Soll bleiben möglich.
8. **Manager als Abstraktions- und Entscheidungsebene:** Jeder Bereichsmanager bündelt ausschließlich den relevanten Kontext, delegiert Ziele und Entscheidungsspielraum, erhält Berichte und verantwortet die Integration. Die Hierarchie ist der Ablauf für Änderungen und Entscheidungen, nicht nur eine Beschreibung von Zuständigkeiten.
9. **Geschäftsführung und subsidiäre Eskalation:** Der Geschäftsführer an der Wurzel besitzt umfassende Entscheidungsbefugnis im vom Nutzer übertragenen Projektauftrag. Bereichsmanager entscheiden Routinefragen innerhalb ihrer Zuständigkeit selbst. Ein ungelöster Konflikt steigt nur bis zum nächsten Vorfahren mit ausreichender Zuständigkeit und Befugnis. Der Nutzer bleibt oberste Instanz für wichtige, ausdrücklich vorbehaltene oder durch keine delegierte Ebene entscheidbare Fragen.
10. **Ein eigener AI-Agent pro Manager:** Jede Managerrolle einschließlich Geschäftsführung wird durch einen eigenen, auf ihre Funktion spezialisierten AI-Agenten ausgeübt. Jeder hat seinen eigenen Arbeitskontext. Eltern benötigen weder die vollständigen Kontexte ihrer Kinder noch Kenntnis darüber, welche internen Details dort geladen sind. Zusammenarbeit erfolgt über Aufträge, Berichte, Schnittstellen und konkrete Rückfragen.
11. **Fachliche Gliederung und verborgene Werkzeugablage:** Zusammengehörige Konzepte, Verhalten, Regeln und Prüferwartungen liegen gemeinsam in rekursiven Slices statt in globalen Typordnern. Alle Markitect-eigenen Dateien liegen unter `.markitect/`. Die kombinierte Einheit aus Modul, Namensraum und Verantwortung sowie `manager.yaml` als deren Deklaration sind die aktuellen Benennungsvorschläge.
12. **Modellpflege als Hauptarbeit:** Der normale Nutzer entwickelt die Spezifikation im Gespräch mit einem AI-Agenten und liest erzeugte Dokumentation und verständliche fachliche Änderungen. YAML ist das interne, versionierbare Speicherformat. Das gewünschte Verhalten zu modellieren wird zur primären Form des Programmierens; Umsetzung, Integration und belastbare Prüfung bleiben ausdrückliche Aufgaben.
13. **Modell und Dateien verbinden:** Der Zusammenhang mit Dokumentation, Backend, Frontend, Tests und Konfiguration bleibt ausdrücklich modelliert. Dateiverantwortung und fachlicher Bezug erfüllen unterschiedliche Aufgaben. Der konkrete Zuordnungsentwurf verbindet Modellinhalte und Realisierungen; Syntax und technische Validierung sind noch umzusetzen. Keine Eins-zu-eins-Abbildung zwischen YAML-Datei und Ergebnisdatei wird vorausgesetzt.
14. **Dauerhafte Zuordnung und vollständiger Projektauftrag:** Akzeptierte Dateiverantwortung und Realisierungsbezüge sind versionierte Modelleingaben. Der Compiler bzw. Host erzeugt daraus einen wiederherstellbaren, an den tatsächlichen Dateistand gebundenen Index. Architektur, Arbeitsweise und erwartete Artefakte sind ebenfalls verantwortete Modellinhalte; vorhandene Dateien allein definieren nicht, welche Ergebnisse erforderlich sind.

## Dokumente und Eigentümerschaft

| Dokument | Verantworteter Inhalt |
|---|---|
| [Zielmodell](model.md) | Begriffe, vorgeschlagene Verträge, Verantwortungsregeln und durchgehendes Order-Beispiel |
| [Fachliche Module und Vertical Slices](conceptual-modules.md) | Neuester Zielstand: `.markitect/`, Managerdeklaration, fachliche Slices, Dateiverantwortung und Dateibezüge, Verträge über Grenzen und modellzentrierte Bedienung; Shop-Beleg noch im ersten Zuschnitt |
| [Projektvertrag](project-contract.md) | Architektur, Arbeitsweise, erforderliche Artefakte, dauerhafte Zuordnungsquellen, abgeleiteter Dateiindex und Änderung ihres gebundenen Stands |
| [Adoption und Agentenarbeitsweg](adoption-and-agent-boundary.md) | Durchsetzungsgrenzen, Brownfield-Distillation, Installation und tägliche Agentenarbeit |
| [Umsetzungsvertrag](delivery-contract.md) | Gewählte technische Basis, Schemaabbildung, neue Befehle, isolierte Teamaufteilung und Integrationsgates |
| [Umsetzungsplan](implementation-plan.md) | Quellabgleich, endliche Schritte, Abnahmeszenarien und offene Entscheidungen |
| [Shop-Walkthrough](shop-walkthrough.md) | Konkrete Antworten zu Dateistruktur, Modell, Änderungen und Start der Umsetzung |
| [Shop-Modellbeleg](shop-example/README.md) | Vollständige vorgeschlagene YAML-Dateien, Vorher-/Nachher-Stand und lokaler Belegprüfer |
| [Laufzeitprotokoll](operating-protocol.md) | Kontextpakete, Nachrichten, Kandidaten, Delegation, Integration, Eskalation und Wiederaufnahme |

Dieses Paket ist der Besitzer dieser neuen Planung. Es ändert keine veröffentlichten Verträge. Die bestehende [Architektur](../../architecture.md) beschreibt ihren jeweiligen Quellstand; die [Roadmap](../../implementation-plan.md) verweist auf die neue Planung. Frühere Government-/Architect-Planung ist im ursprünglichen Planungscommit `d3f3b43` erhalten und wird nicht als benötigte Produktabhängigkeit kopiert. Deren zusätzliche Institutionen werden hier nicht automatisch übernommen.

Lesereihenfolge für den aktuellen Zielstand: diese Richtungsentscheidungen, fachliche Module, Projektvertrag und Laufzeitprotokoll, anschließend Umsetzungsplan. Das allgemeine Zielmodell beschreibt die Prinzipien und verwendet zur Illustration teils die älteren Shop-Identitäten; Walkthrough und YAML-Beleg bewahren das erste ausführliche Beispiel. Bei Abweichungen gilt für Gliederung und Bedienung der neueste Modul-/Projektvertrag. Die konkrete Migration des ersten Belegs gehört zu P0 und ist vor Produktimplementierung erforderlich. Historische Beispiele begründen keine abweichenden Zielanforderungen.

## Arbeitsgrundlage

- Worktree: `C:/Users/Consiliari/.codex/worktrees/project-world-planning/Markitect`.
- Feature-Branch: `codex/project-world-planning`.
- Exakte Planungsbasis: `53be4b98ece0cd9e797530f18c29ec8ae683f3fd` aus `codex/government-assessment`; dadurch bleibt die bisherige Diskussion verfügbar.
- Separat betrachtete technische Hauptlinie: lokaler Remote-Ref `origin/main` bei `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Der Ref wurde für diese Planung lokal gelesen, nicht neu vom Server abgefragt.

Für den jetzigen Implementierungsauftrag wurde `origin/main` erneut abgefragt und bestätigt: `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Der neue isolierte Worktree liegt unter `C:/Users/Consiliari/.codex/worktrees/model-driven-delivery/Markitect`, Branch `codex/model-driven-delivery`. Dieses Paket wurde aus dem unveränderten Planungsworktree übernommen. Der ursprüngliche Planungsstand und andere aktive Arbeitskopien bleiben erhalten.

Die Planungsbasis enthält ältere Produktquellen neben neueren Entwurfs- und Koordinationsdokumenten. Sie ist kein aktueller Implementierungsstand der Hauptlinie. Vor Produktcodeänderungen müssen die Dokumente auf eine ausdrücklich gewählte technische Basis übertragen und die tatsächlichen Verträge dort abgeglichen werden. Andere aktive Worktrees werden durch diese Planung nicht verändert.

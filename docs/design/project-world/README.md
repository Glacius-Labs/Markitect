# Markitect als gemeinsames Projektmodell

Stand: 9. Oktober 2026, Europe/Berlin.

## Auftrag und Status

Dieses Paket hält die direkte Produktdiskussion und den anschließend erteilten Umsetzungsauftrag fest. Die neue experimentelle Sourceoberfläche heißt `markitect project ...`; ihr verbindlicher technischer Stand steht im [Umsetzungsvertrag](delivery-contract.md). Der [Arbeitsablauf](../../project-workflow.md) und das [ausführbare Shop-Projekt](../../../examples/project-world/README.md) zeigen die konkrete Bedienung und Dateistruktur. Umsetzung und Prüfung erfolgen in einem eigenen Worktree auf `codex/model-driven-delivery`; eine Veröffentlichung ist damit nicht verbunden.

Die [Nutzerpräzisierung vom 9. Oktober](model-first-user-workflow.md) definiert den nächsten Produktstand: kanonisches Systemmodell mit verschiedenen Realisierungen, Provider-Onboarding für Beitragende, Explore bis zum ersten erfolgreichen Apply, konfigurierbare Manager, standardmäßige Dokumentation unter `docs/`, vollständige Dateiklassifikation vor Apply und iterative Brownfield-Rückführung vor einem gesonderten Aufräumauftrag. Sie hat bei abweichenden älteren Zielvorschlägen Vorrang; ihre Anforderungen sind nicht automatisch implementiert.

Markitect soll ein gemeinsames, kanonisches Verständnis des Projekts ausdrücken: Ziele, Fachbegriffe, Konzepte, Use Cases, Regeln, Architektur, Organisation und Arbeitsabläufe. Verantwortliche lassen diese Vorgaben innerhalb ihrer Befugnisse umsetzen und prüfen. Das Repository enthält ihre konkreten Realisierungen und die kanonischen Modelleingaben selbst.

## Festgehaltene Richtungsentscheidungen

1. **Gemeinsame Fachsprache:** Bedeutende Begriffe werden definiert und in Konzepten, Regeln und Use Cases ausdrücklich verwendet. Begriffsdefinition, strukturelle Validierung und Prüfung der Umsetzung sind getrennte Fähigkeiten. Nicht jedes Detail muss sofort maschinell durchgesetzt werden.
2. **Management als öffentliches Modell:** Arbeit wird nach rekursiven verantworteten Modulen organisiert. „Modul“ ist der aktuelle Namensvorschlag für die gemeinsame Einheit aus fachlicher Gliederung, Namensraum und Verantwortung. Jeder Manager behält die Verantwortung für seinen Gesamtauftrag und die Integration seiner Teilaufträge.
3. **Eindeutige Verantwortung:** Jedes deklarierte Modul hat genau einen auflösbaren Manager. Interne Namensräume und Modellinhalte sind über diese ausdrücklich gewählte Gliederung eindeutig verantwortet; es entstehen keine parallelen lokalen Ownershipregister. Mehrere Mitwirkende und Prüfer sind möglich. Eine konkrete Agenteninstanz ist nicht automatisch die dauerhafte Rolle.
4. **Verbundene Sichten auf fachliche Slices:** Konzepte, Use Cases, Regeln und Erwartungen bleiben bei ihrer Fachlichkeit. Architektur, Organisation, Prozesse und Workflows sind ebenfalls verantwortete Modellinhalte. Code, Tests, Dokumentation, Pipelines und Konfiguration sind miteinander verknüpfte Realisierungen.
5. **Repräsentationen des Modells:** Code, Tests, Dokumentation und Infrastruktur realisieren die Konzepte und Regeln des kanonischen Systemmodells. Nach der Nutzerpräzisierung vom 9. Oktober ist „Projektion“ dafür wieder ein zulässiger fachlicher Begriff; die frühere Absicht, ihn öffentlich aufzugeben, ist ersetzt. Verantwortung, Arbeitsauftrag, Fähigkeit und Artefakt behalten ihre eigenen Bedeutungen. Bestehende technische Projection-Verträge werden dadurch nicht stillschweigend geändert.
6. **Vom Entwurf zur Verbindlichkeit:** Brainstorming bleibt frei und kann unvollständig oder widersprüchlich sein. Akzeptierte Modellinhalte sind davon erkennbar getrennt. Nicht jede Notiz benötigt einen formalen Vorgang.
7. **Gewünschter Arbeitsablauf:** Projekt aufbauen, Modell entwickeln oder ändern, betroffene Arbeit ableiten, delegiert umsetzen, unabhängig prüfen und das Repository erneut mit dem Soll abgleichen. Reparaturen bei unverändertem Soll bleiben möglich.
8. **Manager als Abstraktions- und Entscheidungsebene:** Jeder Bereichsmanager bündelt ausschließlich den relevanten Kontext, delegiert Ziele und Entscheidungsspielraum, erhält Berichte und verantwortet die Integration. Die Hierarchie ist der Ablauf für Änderungen und Entscheidungen, nicht nur eine Beschreibung von Zuständigkeiten.
9. **Geschäftsführung und subsidiäre Eskalation:** Der Geschäftsführer an der Wurzel besitzt umfassende Entscheidungsbefugnis im vom Nutzer übertragenen Projektauftrag. Bereichsmanager entscheiden Routinefragen innerhalb ihrer Zuständigkeit selbst. Ein ungelöster Konflikt steigt nur bis zum nächsten Vorfahren mit ausreichender Zuständigkeit und Befugnis. Der Nutzer bleibt oberste Instanz für wichtige, ausdrücklich vorbehaltene oder durch keine delegierte Ebene entscheidbare Fragen.
10. **Ein eigener AI-Agent pro Manager:** Jede Managerrolle einschließlich Geschäftsführung wird durch einen eigenen, auf ihre Funktion spezialisierten AI-Agenten ausgeübt. Jeder hat seinen eigenen Arbeitskontext. Eltern benötigen weder die vollständigen Kontexte ihrer Kinder noch Kenntnis darüber, welche internen Details dort geladen sind. Zusammenarbeit erfolgt über Aufträge, Berichte, Schnittstellen und konkrete Rückfragen.
11. **Fachliche Gliederung und verborgene Werkzeugablage:** Zusammengehörige Konzepte, Verhalten, Regeln und Prüferwartungen liegen gemeinsam in rekursiven Slices statt in globalen Typordnern. Alle Markitect-eigenen Dateien liegen unter `.markitect/`. Die kombinierte Einheit aus Modul, Namensraum und Verantwortung sowie `manager.yaml` als deren Deklaration sind die aktuellen Benennungsvorschläge.
12. **Modellpflege als Hauptarbeit:** Der normale Nutzer entwickelt die Spezifikation im Gespräch mit einem AI-Agenten und liest erzeugte Dokumentation und verständliche fachliche Änderungen. YAML ist das interne, versionierbare Speicherformat. Das gewünschte Verhalten zu modellieren wird zur primären Form des Programmierens; Umsetzung, Integration und belastbare Prüfung bleiben ausdrückliche Aufgaben.
13. **Modell und Dateien verbinden:** Der Zusammenhang mit Dokumentation, Backend, Frontend, Tests und Konfiguration bleibt ausdrücklich modelliert. Dateiverantwortung und fachlicher Bezug erfüllen unterschiedliche Aufgaben. Der technische Vertrag verbindet Manager, Statements, Artefakte und Checks mit den erfassten Dateien; es wird keine Eins-zu-eins-Abbildung zwischen YAML-Datei und Ergebnisdatei vorausgesetzt.
14. **Dauerhafte Zuordnung und vollständiger Projektauftrag:** Akzeptierte Dateiverantwortung und Realisierungsbezüge sind versionierte Modelleingaben. Der Compiler bzw. Host erzeugt daraus einen wiederherstellbaren, an den tatsächlichen Dateistand gebundenen Index. Architektur, Arbeitsweise und erwartete Artefakte sind ebenfalls verantwortete Modellinhalte; vorhandene Dateien allein definieren nicht, welche Ergebnisse erforderlich sind.

## Dokumente und Eigentümerschaft

| Dokument | Verantworteter Inhalt |
|---|---|
| [Kanonisches Systemmodell und Benutzerablauf](model-first-user-workflow.md) | Neueste Nutzerpräzisierung, Greenfield-Explore, Provider-Onboarding, vollständige Dateiabdeckung, Brownfield-Rückführung, Quellabgleich und nächste Abnahmekriterien |
| [Praktischer Arbeitsablauf](../../project-workflow.md) | Installation des Sourcekandidaten, Init, Migration, Modelländerung und delegierte Umsetzung |
| [Ausführbares Shop-Projekt](../../../examples/project-world/README.md) | Aktuelles Schema, vertikale Slices, konkrete Dateizuordnung und prüfbarer Python-/SQLite-Code |
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

Lesereihenfolge: diese Richtungsentscheidungen, Umsetzungsvertrag, praktischer Arbeitsablauf und ausführbares Shop-Projekt. Fachliche Module, Projektvertrag und Laufzeitprotokoll erläutern die ausführlichere Zielrichtung. Das allgemeine Zielmodell, der ältere Walkthrough und `shop-example/` bewahren den ersten Entwurf mit abweichenden Identitäten und Syntax; dessen Dateien enden bewusst auf `.yaml.example`. Für die implementierte Syntax gilt der Umsetzungsvertrag mit `project.markitect.example.org/v1alpha1`, nicht dieser historische Beleg. Die umfangreichen Abnahmeszenarien im Plan sind Anforderungen; nur tatsächlich ausgeführte Prüfungen sind Evidenz.

## Arbeitsgrundlage

- Worktree: `C:/Users/Consiliari/.codex/worktrees/project-world-planning/Markitect`.
- Feature-Branch: `codex/project-world-planning`.
- Exakte Planungsbasis: `53be4b98ece0cd9e797530f18c29ec8ae683f3fd` aus `codex/government-assessment`; dadurch bleibt die bisherige Diskussion verfügbar.
- Separat betrachtete technische Hauptlinie: lokaler Remote-Ref `origin/main` bei `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Der Ref wurde für diese Planung lokal gelesen, nicht neu vom Server abgefragt.

Für den jetzigen Implementierungsauftrag wurde `origin/main` erneut abgefragt und bestätigt: `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`. Der neue isolierte Worktree liegt unter `C:/Users/Consiliari/.codex/worktrees/model-driven-delivery/Markitect`, Branch `codex/model-driven-delivery`. Dieses Paket wurde aus dem unveränderten Planungsworktree übernommen. Der ursprüngliche Planungsstand und andere aktive Arbeitskopien bleiben erhalten.

Die ursprüngliche Planungsbasis enthält ältere Produktquellen neben neueren Entwurfs- und Koordinationsdokumenten. Der Quellabgleich für die Umsetzung erfolgte auf der oben genannten Hauptlinienbasis und ist im Umsetzungsvertrag festgehalten. Andere aktive Worktrees werden durch diese Arbeit nicht verändert.

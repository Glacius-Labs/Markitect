# Operationsumfang und Briefings bei Modelländerungen

Stand: 9. Oktober 2026, Europe/Berlin. Direkte Nutzerpräzisierung und daraus abgeleitete Implementierungsvorschläge für `codex/model-driven-delivery`. Quellabgleich: `11f080815722089d84869fef536b94cae9321e74`. Dieser Handoff definiert den nächsten Produktvertrag; neue Befehle, Profile und Benachrichtigungen werden damit nicht als implementiert ausgegeben.

Impact begrenzt die Implementierungsarbeit, aber nicht den Anspruch einer vollständigen Konformitätsprüfung. Betroffene Manager bearbeiten einen Änderungsauftrag; bei vollständigem Verify beurteilen alle Manager ihre fortbestehenden Verpflichtungen am selben finalen Repositorykandidaten. Cleanup verbessert gültige Realisierungen innerhalb des angenommenen Weltbilds. Reconcile stellt das Weltbild über alle Verantwortungsbereiche her, auch ohne bekannte Modelländerung.

Diese Vorgabe ergänzt den [kanonischen Benutzerablauf](model-first-user-workflow.md) und besitzt die hier präzisierten Operationsverträge, Strictness und Änderungsbriefings. Frühere abweichende Vorschläge wie ein allein impactbegrenzter Gesamtabschluss oder ein erst nach Adoption zu untersuchender allgemeiner Clean-Befehl werden entsprechend präzisiert. Tatsächliche aktuelle CLI-Syntax und Implementierungsgrenzen stehen weiterhin im [Arbeitsablauf](../../project-workflow.md) und [Umsetzungsvertrag](delivery-contract.md).

## Zwei Entwicklungsachsen

Zuerst ist **Model → Repo** praktisch nachzuweisen: kanonisches Modell, Apply, Verify, Cleanup und Reconcile; einheitliche Realisierungen über aufeinanderfolgende Änderungen; verwendbare Git-Zusammenarbeit; belastbare Case Studies. Cleanup kann später bei PRs oder regelmäßig, beispielsweise nachts, über eine Pipeline laufen. Dieser Dokumentationsauftrag richtet keine Pipeline oder Automation ein.

Danach folgt **Idea / Work Item / Trigger / Support Ticket → Model**. Die Government-Idee soll vermeiden, dass der Nutzer jede Modellentscheidung selbst initiieren und begleiten muss. Bestehende Managementebenen können innerhalb ausdrücklicher Mandate selbst entscheiden; wichtige oder vorbehaltene Entscheidungen werden eskaliert. Ressorts für übergreifende Anforderungen oder eine Instanz für ungeklärte Regelauslegung sind spätere Gestaltungsmöglichkeiten, keine Voraussetzung für die erste Achse.

Ein zukünftiger Sprintablauf kann Tasks parallel analysieren, Modellvorschläge gegen eine gemeinsame Basis erstellen, Impact berechnen, Konflikte entscheiden, Waves bilden, Modellstände integrieren und anschließend realisieren. Alternativ kann ein zusammenhängendes Feature zunächst gemeinsam modelliert und danach als Paket angewandt werden. Vorläufige Betroffenheit vor einem Vorschlag und konkreter Impact nach einer Modelländerung sind unterschiedliche Schritte; spätere Erkenntnisse korrigieren den Arbeitsplan. Ein textuell konfliktfreier Git-Merge beweist keine fachliche Vereinbarkeit. Modellierung wird nicht allein wegen kleinerer Dateimengen als immer billig oder konfliktfrei angenommen.

## Gemeinsamer Ausführungsvertrag

Die folgenden Begriffe beschreiben Benutzeroperationen, nicht zwingend die Namen neuer CLI-Subcommands. Der bisherige Sourcekandidat trennt insbesondere `plan`, `run`, `verify` und das abschließende materialisierende `apply`. Eine ergonomische Zusammenfassung darf diese Kandidaten-, Befugnis-, Budget- und Prüfbindungen nicht verlieren.

| Operation | Ausgewählte Verantwortung | Erlaubte Arbeit | Erfolgreicher Abschluss |
|---|---|---|---|
| Plan | Betroffene Modellgegenstände, Artefakte, Dateien, Checks und Manager samt notwendiger Vorfahren | Lesen, Beziehungen auflösen und gebundenen Arbeitsplan erzeugen | Auswahl und Unsicherheit sind nachvollziehbar; noch keine Implementierung |
| Apply | Durch Auftrag und Impact betroffene Manager sowie ihre Integratoren | Gegen das angenommene Modell implementieren, lokal reviewen, integrieren und bekannte Fehler begrenzt reparieren | Geprüfter Kandidat erfüllt den beauftragten Umfang; ein behaupteter sauberer Gesamtprojektstand verlangt zusätzlich vollständiges Verify |
| Verify | Alle deklarierten Manager, jeweils für ihren Bereich; Eltern zusätzlich für gemeinsame Verträge und Integration | Bestehende Realisierung beurteilen und deklarierte Prüfungen ausführen; keine Reparatur | Alle anwendbaren Verpflichtungen besitzen aktuelle ausreichende Nachweise am selben finalen Stand |
| Cleanup | Alle Manager, jeweils für ihre Realisierungen; Eltern koordinieren gemeinsame Verbesserungen | Qualität innerhalb unveränderter fachlicher Vorgaben verbessern, Änderungen reviewen und integrieren | Verbesserungen erhalten die geltenden Regeln und bestehen finales vollständiges Verify; begründetes No-op ist zulässig |
| Reconcile | Alle Manager einschließlich Elternintegration | Wie Apply, aber gegen das gesamte angenommene Modell statt ausschließlich den bekannten Änderungsimpact | Fehlende oder abweichende Realisierungen sind bearbeitet und vollständiges Verify besteht |

Reconcile ist damit nicht nur eine größere Dateiauswahl. Es sucht auch nach bisher nicht eingeplanten Abweichungen vom unveränderten Soll. Konforme Bereiche dürfen ohne Änderungen schließen. Ein erfolgreicher Lauf verlangt keine künstlichen Schreibaufträge an jeden Manager.

### Implementierungsumfang und Prüfungsumfang

Die Plan-Dateimenge ist Ausgangspunkt für Umsetzung und Delegation. Sie darf keine künstliche Grenze für benötigte neue Dateien oder bisher fehlende Pflichtartefakte werden. Impact muss neben existierenden Dateien die erforderlichen Realisierungen und Integrationspfade einbeziehen. Unbekannte Beziehungen werden als Unsicherheit weitergegeben und gegebenenfalls breiter untersucht, nicht als Unbetroffenheit gewertet.

Nicht betroffene Manager erhalten bei Apply keinen Implementierungsauftrag. Das entbindet sie nicht von einem anschließenden vollständigen Verify. Ergibt diese Prüfung eine bisher übersehene Auswirkung, wird sie als Befund an den zuständigen Manager gegeben. Ein neuer oder ausdrücklich aktualisierter Implementierungsplan darf die nötige Arbeit aufnehmen; die alte Auswahl ist kein Grund, den Befund zu ignorieren. Es folgt eine neue Kandidatenprüfung.

Das gesamte Repository bleibt nach dem [Dateiabdeckungsvertrag](model-first-user-workflow.md#änderungen-und-vollständige-dateiabdeckung) klassifiziert. Eine kleine Implementierungsmenge verkleinert weder das Gesamtinventar noch die Pflicht, unbekannte Dateien sichtbar zu halten.

### Vollständiges Verify

Jeder Manager beurteilt seinen eigenen verantworteten Umfang gegen das angenommene Modell. Gemeinsame Regeln und benötigte öffentliche Nachbarverträge gehören zum Prüfungskontext. Eltern beurteilen ausdrücklich das Zusammenspiel der Kinder, bereichsübergreifende Invarianten und ihre eigenen Verpflichtungen. Ausschließlich lokale Zustimmung würde gerade die Lücke erhalten, die der Nutzer schließen möchte.

Alle Bewertungen beziehen sich auf denselben unveränderlichen finalen Kandidaten. Erzeugt eine Integration oder Korrektur neue Bytes, müssen betroffene Nachweise erneuert werden. Vor der Übernahme werden Modell, Dateiindex, Tool-/Prüfkonfiguration und Zielstand erneut gebunden. Ergebnisberichte nennen pro Manager Umfang, relevante Verträge, Prüfungen, Befunde und verbleibende Grenzen. Fehlende oder abgebrochene Bewertungen zählen nicht als PASS.

Verify ist konzeptionell Apply, bei dem alle Manager den bestehenden Stand ohne Realisierungsänderung akzeptieren können. Es erzeugt keine korrigierten Artefakte und ändert keine fachlichen Modellvorgaben; operative Reports und Receipts dürfen ausdrücklich geschrieben werden. Ein Befund löst in dieser Operation keine versteckte Reparatur aus. Vorschläge für eine spätere Reparatur sind zulässig. Der Prüfmodus muss auch auf einem bereits vorhandenen Repositorysnapshot ohne vorausgehenden Änderungslauf verwendbar werden.

„Alle Manager prüfen“ bedeutet eine Prüfung aller dauerhaften Verantwortungen, nicht sämtliche Rollen bekommen den gesamten Repositoryinhalt oder jeder historische Agent wird neu gestartet. Jeder Manager erhält seinen begrenzten relevanten Kontext; unabhängige Reviewer und Hostchecks bleiben zusätzliche Nachweise. Ein rein routender Manager prüft seine tatsächlich geltenden Delegations-/Integrationspflichten, statt eine lokale Implementierung zu erfinden. Begründete Nichtanwendbarkeit darf keine fehlenden oder gescheiterten Pflichten verdecken.

Der Standard dieser Vorgabe ist eine aktuelle Beurteilung jedes Managers. Spätere Optimierungen durch Ergebniswiederverwendung benötigen einen eigenen, sichtbaren Vertrag für vollständig identische relevante Eingaben und Prüfbindungen. Das alte Impactresultat allein berechtigt nicht dazu, unbetroffene Manager aus einem vollständigen Verify auszuschließen. Ein ausdrücklich begrenzter Vorabcheck darf existieren, wird aber nicht als vollständiges Verify ausgegeben.

### Cleanup verbessert innerhalb des Weltbilds

Cleanup ist mehr als Formatierung und mehr als Reparatur. Mögliche Verbesserungen sind verständlicher Code, weniger doppelte Logik, klarere Dateistruktur und Dokumentation mit weniger Wiederholung. Fachliches Verhalten, öffentliche Zusagen, geltende Architektur und Prozessvorgaben bleiben erhalten. Qualitätsziele und zulässige Spielräume müssen genügend klar sein, damit nicht jeder Manager beliebige Stilpräferenzen durchsetzt.

Code, Dokumentation, Tests und weitere Realisierungen dürfen sich ändern. Bezüge und Dateiverantwortung werden aktualisiert; generierte Dokumentation wird über ihren Besitzer regeneriert. Eine Änderung von Geschäftsregeln oder Entscheidungskompetenzen ist kein Cleanup. Technische Zuordnungsänderungen werden validiert und am finalen Stand mitgeprüft; eine echte semantische Modelländerung braucht ihren eigenen Entscheidungsweg.

Manager untersuchen alle eigenen Dateien, schlagen sinnvolle begrenzte Verbesserungen vor und dürfen mit begründetem No-op schließen. Gemeinsame Dateien behalten einen eindeutigen Writer. Eltern integrieren Verbesserungen und prüfen grenzübergreifende Auswirkungen. Nach Zusammenführung läuft vollständiges Verify; frühere Einzel-PASS gelten nicht automatisch für den integrierten Stand. Ein nur kosmetischer Verbesserungswunsch lässt ein ansonsten konformes Verify nicht scheitern. Eine verbindliche Qualitätsregel dagegen schon.

Cleanup hat explizite Zeit-, Start-, Kosten-, Änderungs- und Wiederholungsgrenzen. Es soll keine endlosen gegenseitigen Umformatierungen oder Änderungen ohne begründeten Nutzen erzeugen. Für einen regelmäßigen Lauf ist ein geprüfter Kandidat das Ergebnis; dessen automatische Übernahme folgt den bestehenden Projektbefugnissen und Branchbedingungen. Wird der Zielstand inzwischen verändert, darf ein alter Kandidat nicht blind angewandt werden.

## Strictness pro Manager

Der Nutzer wünscht konfigurierbare Strictness. Empfohlen ist ein aufgelöstes Prüfprofil pro Manager mit Projektdefault und zulässigen lokalen Ergänzungen. Namen wie `normal`, `gründlich` oder `vertieft` sind Vorschläge; konkrete Schemafelder sind noch festzulegen.

Strictness kann Prüftiefe, zusätzliche Gegenbeispiele, Reviewrunden und verlangte Evidenz innerhalb des Mandats steuern. Sie verändert weder den fachlichen Sollzustand noch die Zuständigkeit. Eine niedrige Einstellung darf keine Pflichtregel abschalten, eine vorgeschriebene Prüfung überspringen, einen Manager aus vollständigem Verify entfernen oder einen unbekannten Bereich als bestanden ausgeben. Geerbte Mindestanforderungen und verbindliche Projektchecks bleiben wirksam. Ein Budgetende ergibt fehlende Evidenz beziehungsweise einen unvollständigen Lauf, keinen stillen Rückfall auf geringere Qualität.

Vier Einstellungen bleiben unterscheidbar: **Strictness** bestimmt die Prüfung, **Befundseverity** beschreibt die Bedeutung eines Problems, **Benachrichtigungsschwelle** bestimmt die Darstellung und **Befugnis** bestimmt, wer entscheiden darf. Eine kritische Regelverletzung wird nicht durch Wegklicken oder ein niedrigeres lokales Profil zulässig. Die Severity einer Regelverletzung ist nicht automatisch die Entscheidungsklasse einer Änderung dieser Regel. Die automatische Weiterentwicklung des Modells bleibt als zweite Achse getrennt.

## Briefing nach jeder kanonischen Modelländerung

Jede angenommene Änderung am kanonischen Modell erzeugt ein dauerhaft nachvollziehbares Änderungsereignis und eine verständliche Zusammenfassung. Unverbindliche Exploration bleibt als Entwurf erkennbar; ein Entwurf darf nicht als bereits geltende Änderung gebrieft werden. Mehrere Ereignisse dürfen zur Darstellung gebündelt werden, müssen aber einzeln rekonstruierbar bleiben.

Ein Briefing enthält mindestens alte und neue Modellbasis, geänderte Gegenstände, die tatsächliche Änderung ihrer Bedeutung, bekannte betroffene Manager und Realisierungen, Herkunft beziehungsweise Entscheidung, offene Fragen und erwartete Folgearbeit. Ein struktureller Dateiumbau wird als solcher bezeichnet und nicht als neue Geschäftsregel ausgegeben. Ein globaler Kurzüberblick ergänzt bereichsspezifische Briefings mit den relevanten öffentlichen Verträgen. Agenten erhalten keine beliebigen privaten Kontextprotokolle anderer Manager.

Jeder Manager muss bei der nächsten Arbeit gegen die neue Modellbasis den für ihn relevanten Änderungsstand im Kontext erhalten; dies gilt auch, wenn er keinen Implementierungsauftrag bekommt. Ein gespeichertes Briefing muss nicht sofort alle Agentenprozesse starten. Der Host bindet das verwendete Briefing an den aktuellen Auftrag, sodass ein Manager nicht unbemerkt auf einem alten Weltbild arbeitet. Ein bereits laufender Auftrag bleibt an seiner benannten Basis; relevante Änderungen führen zu sichtbarer Neuplanung oder Neubindung statt heimlichem Kontextwechsel.

### Meldungen und Dismiss

Der Nutzer erhält eine nach Severity sortierte Übersicht mit Zusammenfassungen. Vorgeschlagen sind Einträge mit stabiler Kennung, Ereignis-/Modellbezug, Kategorie, Severity, betroffenen Bereichen, Begründung, nächster möglicher Aktion und Verweis auf den Detailnachweis. Denkbare Kategorien sind Information, Verbesserungsmöglichkeit, Konformitätsbefund, Blockade und Entscheidungsbedarf. Das konkrete Severityvokabular und Sortierverfahren bleiben Schemaarbeit; bestehende Compilerdiagnosen werden nicht still zu einem fertigen Notificationmodell umgedeutet.

Gelesen, bestätigt, dismissed und fachlich gelöst sind verschiedene Zustände. Dismiss entfernt beziehungsweise reduziert eine Meldung in der persönlichen Übersicht, schließt aber weder eine Regelverletzung noch eine Pflicht, erteilt keine Genehmigung und löst keine Ausnahme aus. Pflichtbriefings bleiben für Agenten verfügbar. Ein tatsächlich gelöster Befund verweist auf Entscheidung oder frische Prüfevidenz. Unveränderte bereits dismissierte Meldungen sollen nicht bei jedem Poll erneut erscheinen; eine neue erhebliche Änderung oder neue betroffene Basis kann eine nachvollziehbar verknüpfte Aktualisierung erzeugen.

Der Briefing-/Meldungsvertrag ist produktneutral. Ein CLI-/Agentenreport und persistente Records können zuerst genügen; eine eigene Inbox, externe Benachrichtigungskanäle und deren Authentifizierung sind spätere Oberflächenentscheidungen. Der Nutzer soll Meldungen verstehen können, ohne rohe Laufprotokolle zu lesen. Keine Severity- oder Sichtbarkeitseinstellung ersetzt delegierte Entscheidungsbefugnis.

## Git Zusammenarbeit und Pipelineabschluss

Mehrere Entwickler und Sessions können Modellvorschläge auf eigenen Branches vorbereiten. Günstige Strukturprüfungen laufen früh; angenommene zusammenhängende Änderungen können für die teure Realisierung gebündelt werden. Vorläufige Merges sind erneut strukturell und fachlich zu prüfen. Ein konfliktfreier Textmerge oder ein kleiner YAML-Diff beweist keine unabhängige Realisierung. Waves berücksichtigen Modellabhängigkeiten, gemeinsame Artefakte, Writer und Integrationspflichten.

Ein möglicher geschützter Abschluss ist: zusammengeführten Kandidaten erstellen, gezieltes Apply beziehungsweise erforderliches Reconcile durchführen, optional Cleanup, vollständiges Verify und erst dann den exakten Gesamtstand übernehmen. Das umfasst projektspezifische Startanleitungen, Smoke Tests, Migrationen und externe Voraussetzungen, sofern sie zum angenommenen Abnahmevertrag gehören. Nicht verfügbare Drittanbieterumgebungen begrenzen den Nachweis sichtbar; sie werden nicht durch ein lokales PASS ersetzt.

Eine Pipeline kann einen PR vor endgültiger Übernahme ablehnen. Erst nach einem endgültigen Merge zu prüfen erzeugt dagegen zunächst einen fehlerhaften Branch und verlangt Reparatur oder Rücknahme. Branches mit dem Anspruch eines sauberen Gesamtstands brauchen den ersten Vertrag. Ein separat benannter Sollbranch darf vorübergehend ausstehende Realisierungen enthalten, muss diesen Status aber zeigen.

Ein regelmäßiger Cleanup darf die Commitbasis laufender Arbeit nicht still überschreiben. Agentenarbeit kann in unabhängigen Bereichen parallel laufen; finale Übernahmen werden gegen den aktuellen Zielstand koordiniert. Für denselben Modellstand wird die Anwendung nicht unabhängig für Development, Staging und Produktion neu generiert. Geprüfte Builds können weitergereicht werden; umgebungsspezifischer Infrastrukturabgleich ist eine eigene deklarierte Aufgabe. Die konkrete verteilte Queue und Git-/CI-Anbindung sind Folgearbeit.

## Vorhandene Bausteine und Umsetzungslücken

An der oben genannten Quellbasis liefert `internal/modules/projectmodel/` Manager, Statements, Artefakte, Dateiindex und Impact. `internal/host/projectrun/plan.go` wählt Implementierungsmanager und Checks; `run.go`, `review.go`, `rework.go` und `obligations.go` besitzen Delegation, lokale Reviews, Integration und explizite Auftragsabschlüsse. `verify.go` prüft aktuell einen integrierten Run, führt die geplanten Checks aus und kann einen optionalen AI-Verifier verwenden. Das ist noch kein frei aufrufbares vollständiges Verify mit einer Bewertung jedes Managers.

`internal/host/projectcli/run.go` besitzt derzeit keine neuen Project-Operationen `cleanup` oder `reconcile`. Historische Top-Level-Reconcileverträge behalten ihre Bedeutung. In `projectrun` existiert noch kein Manager-Strictnessprofil oder persistenter Briefing-/Dismissvertrag. Die vorhandene `Finding.Severity` in `projectmodel` ist eine Diagnoseklassifikation, nicht bereits die hier beschriebene Benachrichtigungs- oder Befugnispolicy. Die [Sourcevalidierung](../../validation/project-world-delivery.md) dokumentiert die tatsächlichen endlichen Versuche; dieser Quellabgleich fügt keinen Produktlauf hinzu.

Empfohlene nächste Schritte: Operationsumfang und Ergebnisvertrag festlegen; vollständiges read-only Manager-Verify gegen einen festen Snapshot ermöglichen; Briefingereignisse und ihre Kontextbindung ergänzen; danach Reconcile und Cleanup auf derselben Ausführung und Abschlussprüfung aufbauen. Strictness ergänzt diese Prüfwege, ohne die vorhandenen Pflichtgrenzen zu schwächen. Provider-Onboarding und Gesamtinventur aus dem vorherigen Handoff bleiben notwendige Teile des Gesamtprodukts. Government-Entscheidungsautonomie wird anschließend getrennt geplant.

## Abnahmeszenarien und Messung

| Szenario | Erwartung |
|---|---|
| Plan betrifft nur Orders | Apply delegiert Umsetzung nur an nötige Manager und Vorfahren; vollständiges Verify bewertet zusätzlich alle übrigen Manager |
| Unverändertes Inventory verletzt bereits eine Regel | Vollständiges Verify entdeckt den Befund unabhängig vom vorherigen Impact; Apply hat ihn nicht als unbetroffen abgehakt |
| Zwei Kinder bestehen lokal, gemeinsame Invariante scheitert | Elternprüfung verhindert Gesamt-PASS und liefert eine konkrete Integrationspflicht |
| Verify findet einen reparierbaren Fehler | Keine Änderung an Modell oder Realisierungen; Befund kann in einen neuen gebundenen Implementierungsauftrag eingehen |
| Ein Manager oder Pflichtcheck fehlt beziehungsweise läuft ins Budgetende | Vollständiges Verify bleibt unvollständig oder gescheitert; keine implizite Zustimmung |
| Alle Bereiche erfüllen das Modell unverändert | Verify und Reconcile dürfen mit begründetem No-op und aktuellen Nachweisen schließen |
| Cleanup entfernt doppelte Doku und vereinfacht Code | Regeln und Zusagen bleiben erhalten; Bezüge stimmen; endgültiger Kandidat besteht vollständiges Verify |
| Cleanup möchte eine Geschäftsregel lockern | Als separate Modellentscheidung ausweisen; nicht unter Qualitätsverbesserung übernehmen |
| Niedrige Strictness bei verbindlicher Regel | Regel, Mindestprofil und Pflichtchecks bleiben wirksam; kosmetische Hinweise allein blockieren Verify nicht |
| Jede angenommene Modelländerung | Ereignis, verständliches Briefing und relevante Managerkontexte sind an die neue Basis gebunden |
| Nutzer dismissed einen kritischen Befund | Darstellung ändert sich; Pflicht und gescheiterter Prüfstatus bleiben offen |
| Neues Modell während eines laufenden Auftrags | Alte Basis bleibt sichtbar; relevante Neubindung ist explizit; veraltete Evidenz wird nicht übernommen |
| Parallele Tickets und nächtlicher Cleanup | Unabhängige Kandidaten und eindeutige Writer; kein Überschreiben neuer Arbeit durch alten Applystand |
| Projekt verlangt Startweg und Smoke Test | Tatsächlicher Start und vereinbarte Abläufe werden geprüft; fehlende externe Voraussetzungen begrenzen den Abschluss |

Case Studies sollen ganze Entwicklungsfolgen statt nur Erstimplementierungen untersuchen: Modelländerung, parallele Arbeit, Cleanup, unveränderte Drift, Refactoring und abschließende Abnahme. Die Stützenvermietung ist der vorgeschlagene durchgehende Fall. Kontrollierte Fehler in unbetroffenen Bereichen und bereichsübergreifende Widersprüche prüfen ausdrücklich, ob vollständiges Verify mehr erkennt als impactbegrenzte Prüfung.

Zu erheben sind übersehene Anforderungen, widersprüchliche Realisierungen, Fehlalarme, Reparaturrunden, menschliche Eingriffe, Wartezeiten und Gesamtkosten einschließlich Einrichtung, Modellpflege und Reviews. Luna High und konventionelle Entwicklung erhalten vergleichbare Aufgaben, Werkzeuge und Entscheidungsspielräume; fehlgeschlagene Versuche bleiben Teil der Bilanz. Erst wiederholte Ergebnisse begründen eine Reduktion menschlicher PR-Reviews. Automatisierte Reviews, Checks und nachvollziehbare Entscheidungen bleiben erhalten. Simulierte Studien sind kontrollierte Evidenz und noch kein Nachweis allgemeiner Produktionsautonomie.

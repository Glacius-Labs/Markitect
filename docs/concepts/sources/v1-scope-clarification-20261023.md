# Auftrag: unabhängige Gegenprüfung des Markitect-V1-Plans v0.17

Du bist weiterhin der Lead Architect und Coordinator von Markitect.

Repository: C:\Users\Consiliari\Glacius Labs\Markitect

Lies die beigefügten Dokumente:

1. markitect-v1-plan-v0.17.md
   Konsolidierter Scope, Zielarchitektur und vorgeschlagener Umsetzungsplan.

2. code-review-20261023.md
   Unabhängige gezielte Codeprüfung an Main
   447ac3b152de79b52de80513e14dc94349a060bd,
   mit konkreten Quellen und ausdrücklich genannten Grenzen.

Der aktuelle Auftrag ist Gegenprüfung und Planung, nicht Implementierung.

Keine Produktcodeänderungen, keine schreibenden Implementierungs-Chats,
keine PRs oder Merges, keine Provider-/Apply-/Containerläufe.

Eigene isolierte Planungsdokumente und lesende Analyse/Subagenten sind
zulässig. Begrenzte lokale provider- und containerfreie Source-Tests dürfen
zur Überprüfung ausgeführt werden. Berichte exakt, was tatsächlich lief.
Verändere weder fremde Worktrees noch uncommittete Arbeiten.

## 1. Verbindliche Owner-Präzisierung

Wir bauen einen hochwertigen, architektonisch sauberen
Model→Artifacts-Ausführungskern.

Erhalten bleiben soll die geleistete Arbeit am kanonischen Modell,
seiner Sprache und Funktionsweise sowie an Compiler, Registry,
Ownership, Accounting und geeigneten deterministischen Grundlagen.

Die heutige agentische Runtime besitzt keinen Bestandsschutz.
Ihre Typen, APIs, Package-Grenzen und Implementierungen dürfen
vollständig ersetzt werden.

Eine neue Kernarchitektur darf nicht dadurch schlechter werden,
dass man vorhandenen Code weiterverwenden möchte.

Extract, Cockpit/UI, interaktives Authoring und die Migration sonstiger
Altverbraucher sind ausdrücklich außerhalb des Scopes. Sie dürfen keine
Anforderungen an die neue Foundation oder deren Abnahme stellen.

Diese Klarstellung ersetzt widersprechende Erhaltungspflichten aus
v0.16 und deinen bisherigen Entwürfen.

## 2. Prüfe den Plan unabhängig

Übernimm den beigefügten Plan nicht unkritisch.

Prüfe insbesondere:

- Übereinstimmung mit den akzeptierten Produktentscheidungen.
- Richtigkeit der Codebefunde am tatsächlichen Quellstand.
- Klare Verantwortlichkeiten und Abhängigkeitsrichtungen.
- Vermeidung von unnötigen Abstraktionen und doppelten Modellen.
- Einen einheitlichen, build- und testfähigen engen V1-Pfad.
- Vollständige Verbindung von Run-Ende, Persistenz, Harness und Publication.
- Frühe Integrationsfähigkeit statt vieler ungenutzter Interfaces.

Begründe Abweichungen mit konkreten Dateien, Funktionen, Abhängigkeiten
oder nachvollziehbaren Gegenbeispielen.

„Vorhanden“, „weniger Aufwand“ oder „Extract benötigt das“ sind keine
ausreichenden Gründe für eine unpassende Kernarchitektur.

## 3. Quellstand und Evidenz

Erfasse den aktuellen Main-SHA. Falls er von 447ac3b… abweicht, trenne
neue Änderungen ausdrücklich von der beigefügten Codeprüfung.

Die beigefügte Prüfung ist statisch und behauptet keine selbst
ausgeführten Go-, Docker- oder Provider-Tests.

Verwende dein bestehendes Audit, statt alles neu zu erfassen.
Prüfe besonders:

- bootstrap/execution.go:
  öffentlicher Assignment-Pfad und Apply-weite Git-Lebensdauer.
- apply/assignment_realization.go:
  separate Reviews, fortgesetzte Implement-Gespräche und
  Live-Objekte im Ergebnis.
- run/assignment_review.go, assignment_schedule.go, agents.go, turn.go:
  Einzelreview sowie Assignment-/Turn-Kopplung.
- apply/assignment_publication.go:
  Abhängigkeit auf result.Work.
- gitrepo/git_session.go und assignment_workspace_ids.go:
  Reservierung künftiger Reviews und instanzgebundene Workspaces.
- Codex agents.go und agents_credential.go:
  alte Run-Typen, native Providerprotokolle, API-Key-Pfad und Auth-Helper.
- bootstrap/main.go und ausgeschlossene Verbraucher:
  tatsächliche Build-Kopplungen.
- Modell-/Snapshot-/Topologie-/Accounting-Kern:
  zu bewahrende Semantik und geeignete Wiederverwendung.

Historische SHAs sind nach dem Source-History-Reset nicht automatisch
Vorfahren des heutigen Main. Prüfe ihre Verfügbarkeit, bevor du sie
als spätere Rückgriffsmöglichkeit voraussetzt.

## 4. Unverhandelbare Konzeptgrenzen

Assessment, Implement und Verify verwenden dieselbe vollständige
erforderliche modellierte Managerhierarchie.

Keine Root-only-Abkürzung bei mehrstufiger Organisation.
Keine separate Review-Engine hinter einem gemeinsamen Wrapper.

Ein Workflow orchestriert Runs.
Ein Run organisiert Manager.
Ein Run veröffentlicht nicht ins Nutzerrepository.

Frische Gespräche, Managercontainer und ein Git Workspace Service
je Run.

Keine fachliche Turn-/Pass-API oder kleinteilige Prozessmaschine.
Native Protokollnamen wie turn/start dürfen im Provideradapter bleiben.

Normale technische Nachrichten, Synchronisation, Abbruch und endliche
Ressourcenlimits bleiben erforderlich.

Die vier deterministischen Kontrollpunkte bleiben bestehen.
Während eines Runs liefern Fehler zunächst reparierbares Feedback.
Fachliches Reject, Modellblocker und technische Fehler bleiben
unterscheidbar.

Shared Notes gehören zu V1, sind persistent und nichtkanonisch.

Dauerhafte Kandidaten und verbindliche Ergebnisse dürfen nicht von
weiterlaufenden Containern oder alten Assignment-Objekten abhängen.

## 5. Erwartete Plananpassung

Entferne die bisherigen Extract-/Authoring- und UI-Migrationen aus
dem Auftrag. Entferne Extract-spezifische Anforderungen aus neuen
Run-Verträgen.

Schlage einen begrenzten Stilllegungsschnitt für ausgeschlossene
aktive Commands, Quellen und Tests vor.

Kein kaputter aktiver Build, keine pauschalen Test-Skips, keine
versteckte zweite Runtime. Die tatsächliche Stilllegung erfolgt erst
nach Freigabe; jetzt wird sie nur präzise geplant.

Leite Packages und Schnittstellen aus Verantwortung ab.

Keine Obergrenze von drei neuen Packages, aber auch kein Interface
oder Package ohne begründeten Bedarf. Die Bestandspfade sind
Untersuchungsorte, keine Architekturvorgabe.

Benenne insbesondere:

- Den Owner des vollständigen Run-Abschlusses einschließlich
  Persistenz und Cleanup, auch beim Standalone-Aufruf.
- Den Owner der vollständigen Harness-Kette bis zur nativen Nutzung.
- Den Übergabevertrag ohne Live-Workspace-/Tree-Objekte.
- Die Unterscheidung zwischen registrierter Ownership und tatsächlichen
  Manageränderungen einschließlich korrekter Kinderintegration.
- Den sicheren Ersatz der Auth-Kleinstcontainer.
- Die Entfernung des Codex-API-Key-Produktpfads.
- Publication nach Ende aller vorherigen Run-Container.

## 6. Ergebnis dieser Gegenprüfung

Liefere:

1. Bestätigte Befunde und konkrete Korrekturen am beigefügten Plan.
2. Eine begründete Ziel-Paketstruktur und Abhängigkeitsrichtung.
3. Die minimale Anbindung an das bewahrte Modell-/Compilerfundament.
4. Den Stilllegungsumfang ausgeschlossener Verbraucher.
5. Wenige notwendige gemeinsame Verträge mit tatsächlichen Konsumenten.
6. Den ersten vollständigen vertikalen Nachweis und anschließende
   Implementierungsschnitte.
7. Echte offene Owner-Entscheidungen, keine Rückfragen zu gewöhnlichen
   technischen Details.

Lass mindestens einen frischen unabhängigen lesenden Reviewer die
resultierenden Grenzen und Widersprüche prüfen.

Der Reviewer soll auch Überengineering und implizite Altcodebindungen
suchen, nicht nur die Existenz aller Tabellenfelder bestätigen.

Konsolidiere deine bestehenden Planungsdokumente. Erzeuge keine weitere
widersprechende Serie. Historische Arbeitsaufträge gehören nicht in den
normalen Einstieg neuer Implementierer.

Beende diesen Auftrag mit dem Gegenprüfungsbericht und dem korrigierten Plan.

Beginne danach nicht automatisch mit der Implementierung.
Wir geben den überprüften Plan ausdrücklich frei.

Reale Container- und Providerprüfungen benötigen später zusätzlich
einen klar begrenzten Ausführungsauftrag.

Die Zielgröße ist ein einfacher, kohärenter, hochwertiger Kern —
weder maximal übernommener Altcode noch maximal neu geschriebener Code.
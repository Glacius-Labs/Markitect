# Liefer- und Koordinationsplan

Stand: 2026-10-07. Vorbereitung für die zwei vom Nutzer angekündigten Chats. Keine Implementierung oder Chat-Beauftragung ist durch dieses Dokument bereits erfolgt. Die Autorisierung zur Ausführung folgt auf das angekündigte Go; technische Einzelentscheidungen innerhalb des dann übertragenen Auftrags brauchen keine wiederholten pauschalen Rückfragen.

## Verantwortung und Arbeitskontexte

Der Koordinator besitzt das Gesamtziel, die [Architektur](architecture.md), Priorisierung, Schnittstellenentscheidungen, Ergebnisabnahme und die Zusammenführung von Befunden. Er prüft Beiträge selbst und nutzt unabhängige Subagenten für konkrete Reviews. Er ersetzt die unabhängige Studienbewertung nicht durch seine Erwartung an Government.

Implementer baut die Government-Variante und die dafür erforderlichen Tests. Case Study besitzt den vergleichbaren Fallaufbau, die Ausführungshülle, Datenaufzeichnung und unabhängige Bewertung gemäß [Studienprotokoll](evaluation.md). Beide Chats dürfen geeignete Subagenten und vorhandene Workflows verwenden. Für jeden delegierten Auftrag werden Verantwortung, Schreibbereich, Eingaben und Erledigungskriterien benannt. Kleine eng gekoppelte Änderungen bleiben bei einem Agenten; voneinander unabhängige Untersuchungen und Reviews dürfen parallel erfolgen.

Beide angekündigten Chats sind nach Nutzerwunsch Sol 6.1 High. Diese Einstellung ist kein automatisch festgelegtes Modell für alle Versuchsakteure. Deren vollständige Modell-/Runnerkonfiguration wird im Studienprotokoll gesondert festgeschrieben.

Architect bleibt Eigentümer seiner Classic-Linie. Seine Release-, Test- und Härtungsarbeit wird nicht in die Government-Implementierung eingemischt. Bereits direkt erteilte Nutzeraufträge in jedem Chat haben Vorrang. Für Vergleiche zählt eine unveränderliche konkrete Classic-Version mit Nachweisen, nicht der ständig wechselnde Branchname oder eine Selbsteinschätzung ohne Abschlussbelege.

### Isolierung

- Government erhält einen eigenen benannten Feature-Branch und Worktree auf dem geprüften Quellstand `1ea5c76f55526fc4d721e865885436153f48b497`. Notwendige spätere Classic-Fixes werden einzeln geprüft und als explizite neue Baseline nachgezogen; ein laufender Versuch ändert seine Versionen nicht.
- Die aktuelle Dokumentationskopie enthält die Entwürfe, aber ältere Produktquellen. Historische Vorbereitungscommits und nachfolgende Designänderungen sind kein Ersatz für die Quellbasis. Der Koordinator überträgt gezielt das aktuelle Designpaket in die Government-Arbeitskopie und dokumentiert dessen Commit.
- Case Study erhält einen eigenen Arbeitsbereich mit versionierten Fixtures, Protokollen und öffentlichen Entwicklungsprüfungen. Jede Kombination aus Variante, Ausgangszustand und Wiederholung erhält eine frische unabhängige Versuchskopie sowie getrennte Laufdaten.
- Der Implementer schreibt keine privaten Bewertungsfälle oder Ergebnisbewertungen. Der Case-Study-Chat repariert keine Kandidaten stillschweigend. Ein Reviewbefund wird an den Eigentümer gegeben, eine Reparatur erzeugt einen neuen festgehaltenen Stand.
- Private Bewertungsdaten gehören nicht in die an Implementierungsagenten übergebenen Repositorys oder Kontexte. Welche tatsächlichen Zugriffsgrenzen der Runner bietet, wird dokumentiert; eine Ordnertrennung allein wird nicht als OS-Isolation bezeichnet.

v0.1 setzt kooperative Agenten voraus, die den Host-Ablauf und ihre zugewiesenen Arbeitsbereiche einhalten. Die verweigerte Übernahme bei fehlenden Stimmen ist eine Garantie dieses Ablaufs. Mit geerbten vollständigen OS-Rechten ist sie kein Schutz vor direkter Manipulation der aktiven Ref, fremder Arbeitskopien oder des Laufjournals. G2 prüft den gesteuerten Übernahmeweg; S1 dokumentiert und prüft die tatsächlich angebotenen Tool-/Dateigrenzen. Vertrauliche Studienbewertungen werden erst nach dem Einfrieren eines Kandidaten außerhalb seines Actor-Kontexts ausgewertet; bleibt ein technischer Zugriff möglich, wird das als begrenzter Leakage-Schutz ausgewiesen und nicht als nachgewiesene Abschottung bezeichnet.

## Auftragsformat

Jeder Auftrag nennt: Problem und gewünschtes Verhalten, Designversion und Quellcommit, konkrete Zuständigkeit und Dateigrenzen, bestehende Autorisierung, zu erhaltende Invarianten, überprüfbare Endkriterien, erforderliche Prüfungen und Berichtsumfang. Ein Plan ist nachprüfbar, ohne jede private Implementierungsentscheidung vorwegzunehmen.

Jeder Abschlussbericht enthält den tatsächlich geprüften Commit, konkrete Änderungen, geeignete Tests samt Ergebnissen, verbleibende Grenzen, Fehlversuche und vorgeschlagenen nächsten Schritt. Nicht verfügbare Daten bleiben unbekannt. Ein Agentenbericht, eine Testanzahl oder ein grüner Einzelcheck ersetzt keine fachliche Abnahme des Arbeitspakets.

Wird ein unabhängiges Folgepaket gestartet, darf es keine unbestätigte Schnittstelle eines offenen Vorgängers als feststehend voraussetzen. Schnittstellenänderungen werden zuerst in ihrem Designvertrag entschieden und anschließend an beide betroffenen Chats weitergegeben. Der Koordinator prüft vor jeder Fortsetzungsnachricht neue direkte Nutzeranweisungen und den tatsächlichen Chatstand.

## Endliche Meilensteine

Die technischen Details gehören in Architektur und Quellübergang. Diese Tabelle legt die Reihenfolge und die zu liefernden Nachweise fest.

| Stufe | Implementer | Case Study | Abschlussbedingung |
|---|---|---|---|
| D0: Designübergabe | Design lesen, Quellbasis binden, konkrete Widersprüche melden | Design und Bewertungsgrenzen lesen | Koordinator hält die gleiche Designversion, klare Arbeitsbereiche und offene Risiken fest; kein bloßes erneutes Planen ohne Befund |
| G1: Modell und Repository | Kanonische Organisationsbegriffe, deklarierte Verantwortungen, bidirektionale Artefaktzuordnung und lesender Arbeitsplan | Gemeinsame Anforderungen, Greenfield-Start, kontrollierte Brownfield-Herkunft und Backlog vorbereiten | Ein Auftrag ist auf echte Modellidentitäten und tatsächliche Artefakte zurückführbar; unbekannte, widersprüchliche oder mehrfach beanspruchte Verantwortung bleibt sichtbar |
| G2: Ein vollständiger Auftrag | Einen Bereich mit tatsächlicher Kandidatenerstellung, eigenem Review und geschützter Übernahme ausführen | Öffentliche Verhaltensprüfungen und Aufnahme echter Laufdaten bereitstellen | Tatsächliche Dateien werden geändert; fehlende Zustimmung, veralteter Kandidat und fehlender Nachweis verhindern die Übernahme; Erfolg und Abbruch liefern Berichte |
| G3: Rekursion und Zusammensetzung | Elternbereich mit mindestens zwei fachlichen Kindern, getrennten Kontexten und eigener Integrationsprüfung | Fall mit lokal bestandenen, aber zusammen fehlerhaften Ergebnissen bereitstellen | Übergeordneter Fehler wird trotz grüner Kinder erkannt; Reparatur erreicht einen frisch geprüften Gesamtstand; unabhängige Zweige dürfen parallel arbeiten |
| G4: Modellpflege und Konflikt | Delegierte substanzielle Modelländerung sowie nicht delegierten Zielkonflikt korrekt behandeln | Sichtbare Entwicklungsfälle für Modelllücke und Eskalation vorbereiten | Alte Ordnung legitimiert die Änderung; Selbstermächtigung scheitert; alle erforderlichen finalen Stimmen sind aktuell; ungelöster Konflikt blockiert nur abhängige Arbeit |
| G5: Begrenzter Dauerbetrieb | Persistente Queue, Ressourcengrenzen, Wiederaufnahme und sichere Promotion nach Unterbrechung | Abbruch- und Wiederaufnahmeablauf sowie vollständige Metrikerfassung prüfen | Keine doppelte Übernahme, keine Wiederverwendung veralteter Stimmen, keine unbekannte externe Wirkung als Erfolg; leere oder blockierte Queue wird ruhig |
| S1: Versuchsreife | Genau geprüften Government-Kandidaten mit Bedienpfad liefern | Drei ausführbare Varianten, sechs frische Starts, gefrorenes Protokoll und Bewertungszugriff nachweisen | Alle Varianten bestehen denselben öffentlichen Einrichtungs-/Ausführungssmoke; fehlende Fähigkeit bleibt als Readiness-Befund offen statt simuliert |
| S2: Vergleich | Nur dokumentierte Fehlerkorrekturen außerhalb bereits eingefrorener Läufe | Echte vergleichbare Läufe durchführen, Ergebnisse unabhängig prüfen | Rohdaten und Revisionen stimmen mit Auswertung überein; Abbrüche und Eingriffe zählen; mehrere Wiederholungen werden nach Kosten und Streuung geplant |

G1 bis G5 sind Funktionsschritte; sie müssen nicht jeweils eine neue öffentliche Version erzeugen. G2 liefert den ersten tatsächlichen Vertikalschnitt. Der vollständige Vergleich der beanspruchten autonomen Arbeitsweise setzt G5 voraus. Ein früherer Teilversuch wird mit seiner engeren Frage beschriftet und nicht als abgeschlossene Government-Fallstudie gewertet.

Stufen dürfen bei unveränderten Abhängigkeiten gemeinsam geliefert werden; ihre Nachweise dürfen nicht entfallen. Ein konkreter Fehler verlangt einen gezielten Test. Bestehende verbindliche Projektgates bleiben maßgeblich. Nach bestandenen erforderlichen Prüfungen werden keine zusätzlichen Tests allein zur Erhöhung der Testzahl angehängt. Verhalten mit relevanter Unsicherheit wird ehrlich als unbewiesen geführt.

## Koordinatorentscheidungen und Eskalation

Der Koordinator entscheidet innerhalb des erteilten Gestaltungsauftrags über Architekturzuschnitt, Begriffe, konkrete Refaktorierungen, interne Schnittstellen, Arbeitszerlegung und Tests. Technische Blockaden werden zuerst untersucht und mit einem konkreten Lösungsvorschlag bearbeitet. Subagenten dürfen Alternativen bewerten; ihr Mehrheitsvotum ersetzt keine begründete Entscheidung.

Zum Nutzer gehen Änderungen am Produktziel, neue externe Verpflichtungen oder Ausgaben außerhalb einer vereinbarten Grenze sowie ein unauflösbarer Konflikt zwischen seinen Anforderungen. Eine ausgewählte neue technische Lösung oder notwendige Refaktorierung ist für sich kein Anlass für eine neue pauschale Freigabe. Bis zum angekündigten Go bleibt der Stand jedoch Designarbeit.

Bei einer tatsächlichen Produktblockade beschreibt die Rückfrage den konkreten Kandidaten, die Entscheidung und ihre Folgen. Wartezeiten werden erfasst. Unabhängige autorisierte Arbeit kann fortfahren. Nicht autorisierte Veröffentlichungen, Änderungen an anderen Projekten oder externe Nachrichten werden nicht aus allgemeiner Architekturverantwortung abgeleitet.

## Vorbereiteter Erstauftrag: Implementer

Der folgende Text ist ein noch nicht versendeter Auftrag. Der Koordinator ergänzt beim Go die verifizierten absoluten Pfade und Commits; die Platzhalter sind kein lauffähiger Arbeitsauftrag.

> Du implementierst Markitect Government auf dem isolierten Feature-Branch und Worktree `<Government-Arbeitskopie>`, Ausgangscommit `<Quellcommit>`, Designpaket `<Designcommit und absoluter Pfad>`. Lies dessen README, Architektur, Quellübergang und diesen Lieferplan sowie AGENTS.md und die aktuellen Contribution-Anweisungen deiner Arbeitskopie. Der Nutzer hat dem Koordinator die Architekturverantwortung übertragen; dieses Design ist dein Arbeitsvertrag. Kanonisches Modell, purpose, Graph, Datei-Realisierung und rekursive unabhängige Prüfung bleiben verbindlich. Beginne mit G1 und dem für G2 erforderlichen kleinsten durchgängigen Vertragszuschnitt. Führe G1 bis zum prüfbaren Abschluss, statt nur einen weiteren Plan abzugeben. Melde echte Designwidersprüche mit einer konkreten Alternative. Verwende Subagenten für unabhängige Teile und Reviews, mit getrennten Schreibbereichen. Ändere weder Architect-Worktree noch private Studienfälle. Erfinde keine vorhandene Runner-, Sicherheits- oder Annahmefähigkeit durch Mock-Ergebnisse. Belege insbesondere unbekannte Zuständigkeiten, Modell-/Dateizuordnung und den konservativen Plan. Liefere einen Commit, relevante Testergebnisse, Grenzen und den konkreten nächsten Vertikalschnitt. Source- und Releasegates richten sich nach den Repo-Anweisungen; Government wird mit diesem Auftrag noch nicht öffentlich veröffentlicht. Direkte Nutzeranweisungen haben Vorrang. Berichte in deinem eigenen Chat; der Koordinator liest Fortschritt und relevante Abweichungen dort nach.

## Vorbereiteter Erstauftrag: Case Study

> Du besitzt den unabhängigen Vergleich aus klassischem agentic Coding, Markitect Classic und Markitect Government. Arbeitsbereich `<Studien-Arbeitskopie>`, Designpaket `<Designcommit und absoluter Pfad>`. Lies README, Studienprotokoll und Lieferplan. Bereite den gemeinsamen Fall, den leeren Greenfield-Start und den kontrollierten Brownfield-Start, denselben Backlog sowie eine nachvollziehbare Metrikerfassung vor. Implementiere die Ausführungshülle und öffentliche Entwicklungsprüfungen; halte private Bewertungsfälle getrennt von den an Implementer und Versuchsakteure übergebenen Daten. Kriterien kommen aus den gemeinsamen Anforderungen, keine nachträglichen Government-Vorteile. Eine gute konventionelle Agentenlösung darf dieselben Werkzeuge und Subagenten verwenden. Erfasse Einrichtung, Modellpflege, alle Versuche, Wartezeit, Tokens/Kosten soweit tatsächlich verfügbar und jeden menschlichen Eingriff. Trenne Protokollfixtures von echten Agentenläufen. Nutze Subagenten für unabhängig vorbereitete Prüfung, Fallaufbau und Auswertungsreview mit klarer Dateiverantwortung. Beginne noch keinen vollständigen kostenpflichtigen oder ungebundenen Vergleich: zuerst lieferst du festgehaltene Fixtures, Protokoll, öffentliche Smokes und die verbleibenden Runner-/Budgetvoraussetzungen. Ein echter Lauf braucht die vom Koordinator festgehaltenen Versionen und Grenzen. Repariere keine Versuchskandidaten stillschweigend. Liefere Commit, ausgeführte Prüfungen und einen ehrlichen Readiness-Bericht. Direkte Nutzeraufträge haben Vorrang; Rückkommunikation an andere Chats erfolgt nur im ausdrücklich autorisierten Rahmen.

## Vor jeder tatsächlichen Übergabe

1. Das Go und die tatsächlichen Chats mit Titel, ID, Modellkonfiguration und Arbeitskontext feststellen; keine neu angelegten fremden Chats anhand ähnlicher Namen erraten.
2. Ihre letzten direkten Nutzeraufträge lesen. Bei bestehenden Aufgaben die Übergabe anschließen, statt Arbeit zurückzusetzen.
3. Isolierte Schreibkontexte und genaue Ausgangsstände herstellen. Keine Tests oder Implementierung aus der alten Dokumentationsbasis starten.
4. Designpaket und Erstauftrag vollständig mit konkreten Pfaden und Commits übergeben. Nachrichten an Implementer und Case Study sind vom Nutzer für diesen Zweck vorgesehen; weitere Empfänger folgen daraus nicht automatisch.
5. Den ersten Fortschritt abwarten, Ergebnisse gegen die Endkriterien prüfen und nur konkrete nächste Schritte anstoßen. Überwachung, falls gewünscht, als ausdrückliche Thread-Automation einrichten; nicht von einem gespeicherten Plan allein auf einen laufenden Hintergrundprozess schließen.

Nach dem Designabschluss bleibt dieses Paket bereit zur Übergabe. Es entstehen in dieser Phase keine Produktquellen, neuen Agenten-Chats, Implementierungs-Worktrees oder gemessenen Fallstudien.

# Markitect V1 – Zielarchitektur und Umsetzungsplan

**Dokumentversion:** 0.16  
**Status:** Vom Owner akzeptierte Zielarchitektur und Implementierungsstrategie  
**Datum:** 23. Oktober 2026  
**Repository:** https://github.com/Glacius-Labs/Markitect  
**Referenzstand der bisherigen Analyse:** Main `4cbfcb005d9a3d0636139314955f070d3abdb219`

Dieses Dokument ist die fachliche und architektonische Grundlage für die Stabilisierung von Markitect V1. Vor der Implementierung muss der aktuelle Main-Stand erneut geprüft werden.

Historische Konzepte, Implementierungen und Experimente dienen als Erkenntnisquellen. Sie sind keine Verpflichtung zur Wiederverwendung bestehender Architektur oder bestehenden Codes.

---

# 1. Ziel

Markitect soll ein kanonisches Architekturmodell zuverlässig in konkrete Artefakte überführen und die bestehende Realisierung über Änderungen hinweg konsistent halten.

Die aktuelle V1 konzentriert sich ausschließlich auf:

**Model → Artifacts**

Dazu gehören:

1. Deterministische Modellkompilierung
2. Strukturelle Validierung
3. Semantisches Model Assessment
4. Implementierung durch eine Managerhierarchie
5. Unabhängige Verifikation
6. Korrekturen bei Findings
7. Kontrollierte Veröffentlichung

Artifacts → Model, insbesondere Extract, gehört nicht zur aktuellen Weiterentwicklung.

Bestehende Extract-Funktionalität soll nicht unbeabsichtigt beschädigt werden. Falls ihre Erhaltung erhebliche Architekturkompromisse erfordern würde, muss dies ausdrücklich als Entscheidung sichtbar gemacht werden.

## 1.1 Produktthese

Zuverlässigkeit entsteht nicht dadurch, dass ein einzelner Agent alle Anforderungen überblickt.

Sie entsteht durch:

- Ein autoritatives kanonisches Modell
- Explizite Verantwortlichkeiten
- Vollständige Artefaktzuordnung
- Hierarchische Arbeitsteilung
- Unabhängige Reviews
- Kontrollierte Git-Integration
- Unabhängige Verifikation
- Deterministische Kontrollen

## 1.2 Qualitätsprioritäten

In dieser Reihenfolge:

1. Fachliche Korrektheit
2. Saubere Architektur
3. Verständlicher, hochwertiger Code
4. Zuverlässigkeit und Wartbarkeit
5. Einfache Bedienbarkeit
6. Laufzeit- und Ressourceneffizienz
7. Wiederverwendung historischen Codes

Wiederverwendung ist willkommen, aber niemals auf Kosten einer besseren Architektur.

---

# 2. Verbindliche Architekturentscheidungen

## D1 – Universeller Run-Executor

Jede agentische Arbeit verwendet denselben grundlegenden Ausführungsmechanismus.

Ein **Run** führt einen bestimmten Auftrag durch die erforderliche Managerhierarchie aus.

Beispiele:

- Model Assessment
- Implement
- Verify

Der Run-Executor organisiert:

- Manager-Assignments
- Hierarchische Delegation
- Rückfragen und Antworten
- Abstimmung zwischen Verantwortungsbereichen
- Parallele Kinderarbeit
- Reviews und Korrekturen
- Git-basierte Übergaben und Integration
- Root-Abschluss

Er ist unabhängig von Apply verwendbar.

Ein Run entscheidet nicht, welcher Run danach ausgeführt wird.

## D2 – Workflow als Run-Komposition

Ein **Workflow** koordiniert mehrere Runs.

Er bestimmt:

- Reihenfolge
- Eingaben
- Ergebnisübergaben
- Bedingungen
- Wiederholungen
- Abbruch
- Gesamterfolg

Er enthält keine eigene Managerkoordination.

**Apply:**

```text
Compile & Check
      |
      v
Model Assessment
      |
      +-- Reject --> Stop
      |
      v
   Implement <---------+
      |                |
      v                |
    Verify             |
      |                |
      +-- Reject ------+
      |   Findings
      v
    Approve
      |
      v
Final Check
      |
      v
Publication
```

Failed und Incomplete sind von fachlichen Rejects zu unterscheiden.

Alle Runs eines Commands teilen dessen Gesamtbudget.

Workflows sollen später einfach konfigurierbar sein und Custom Commands ermöglichen. V1 benötigt keine allgemeine Workflow-DSL.

## D3 – Container und Git Workspace Service pro Run

Jeder Run besitzt eine neue Ausführungsumgebung.

Für jeden Manager:

- Eigener Container
- Eigener Git-Arbeitsstand
- Eigenes Gespräch
- Eigener Harness

Innerhalb eines Runs bleiben Container und Gespräche während Delegation, Abstimmung, Review und Korrektur bestehen.

Nach dem Run werden die Container geschlossen.

Ein **Git Workspace Service** besitzt ebenfalls genau die Lebensdauer eines Runs.

Er ist ausschließlich für Git-Arbeitsstände und zugehörige Operationen verantwortlich:

- Vorbereitung
- Klonen
- Ausgangs-Commits
- Git-Übergaben
- Integration
- Arbeitsstandaktualisierung
- Konfliktmeldung
- Lesen fixierter Commits

Er ist kein Agent und trifft keine fachlichen Entscheidungen.

Fixierte Git-Kandidaten müssen über das Ende eines Runs hinweg erhalten bleiben.

Für zukünftige Runs werden keine Workspace-Volumes im Voraus reserviert.

Unnötige kurzlebige Hilfscontainer sollen entfernt werden. Isolierte Check-Container bleiben zulässig, wenn technisch erforderlich.

## D4 – Deterministische Prüfungen

Vier verbindliche Kontrollpunkte:

**1. Vor Workflowbeginn**

Vollständige Modellkompilierung und anwendbare Strukturprüfung.

Bei Fehler sofort abbrechen, bevor ein Agent startet.

**2. Nach Kinderübergaben**

Der Host führt geeignete deterministische Prüfungen aus.

Das Ergebnis wird dem Vater als Feedback geliefert.

Fehler werden innerhalb des Runs durch die verantwortliche Organisation korrigiert.

**3. Vor erfolgreichem Root-Abschluss**

Ein Implement-Run darf keinen deterministisch ungültigen Gesamtstand als erfolgreich abschließen.

Root erhält Fehler als Feedback und lässt sie korrigieren.

**4. Vor Veröffentlichung**

Abschließende Prüfung des akzeptierten Kandidaten und seiner Publikationsvoraussetzungen.

Diese Prüfungen sollen vorhandene Kernmechanismen verwenden. Keine neue Check-Protokollarchitektur.

Verify darf einen fehlerhaften Kandidaten regulär mit Reject und Findings zurückweisen.

## D5 – Deterministischer Provider-Harness

Der Harness eines Managers setzt sich aus fünf Ebenen zusammen:

| Ebene | Verantwortung |
|---|---|
| Markitect Core | Allgemeine Prozess- und Zusammenarbeitsregeln |
| Projekt | Kanonische Fach- und Architekturregeln |
| Manager | Rolle, Organigramm, Ownership und Registry |
| Run | Mandat und auftragsspezifische Eingaben |
| Provider | Native Anweisungen, Skills, Hooks und Werkzeuge |

Allgemeine Markitect-Regeln gehören zum versionierten Produkt.

Projektspezifische Regeln stammen aus dem kanonischen Modell.

Gleiche relevante Eingaben sollen denselben Harness erzeugen.

Codex und Claude sollen ihre jeweiligen nativen Mechanismen verwenden.

Für unbekannte kompatible Provider wird ein generischer `AGENTS.md`-Fallback vorgesehen.

Der eigentliche Auftragsprompt soll kurz, verständlich und möglichst frei von allgemeinen Prozessanleitungen sein.

## D6 – Run-Ergebnisse und Shared Notes

Zwei unterschiedliche Datenarten:

**Verbindliche Run-Ergebnisse**

- Run-Identität
- Mandat
- Modellbindung
- Eingabestand
- Status
- Git-Kandidat
- Approve/Reject
- Findings
- Erforderliche technische Beobachtungen

Sie steuern den weiteren Workflow.

**Shared Notes**

- Erkenntnisse
- Verbesserungsideen
- Technische Schulden
- Auffälligkeiten
- Testlücken
- Optionale Nebenbefunde

Shared Notes gehören direkt zu V1.

Sie bleiben über Runs und Commands hinweg erhalten, sind aber nichtkanonisch und lösen keine automatische Arbeit aus.

Verpflichtende Findings dürfen nicht als optionale Notes versteckt werden.

Keine zusätzliche komplexe Memory- oder Evidence-Plattform.

## D7 – Architekturqualität vor Wiederverwendung

Die Zielarchitektur bestimmt die Implementierung.

Historischer Code ist nicht verbindlich.

Jede betroffene Komponente wird eingeordnet:

- **Keep:** Passt bereits.
- **Refactor:** Verantwortung passt, Struktur nicht.
- **Replace:** Konzeptionell ungeeignet.
- **Remove:** Nicht mehr benötigt.

Alte Pass-/Turn-Konzepte und unnötige Kompatibilitätsschichten sollen nicht erhalten bleiben.

Eine vollständige Neuentwicklung einzelner Komponenten ist ausdrücklich zulässig.

Ein vollständiger Repository-Rewrite ist dagegen kein Selbstzweck.

## D8 – Selektiver Clean-Slate-Rebuild

Die deterministischen Kernkomponenten sind bevorzugte Kandidaten für die Beibehaltung, sofern das Architektur-Audit ihre Qualität bestätigt.

Der agentische Ausführungskern darf dagegen unabhängig von bestehenden internen Schnittstellen neu aufgebaut werden.

Es ist ausdrücklich zulässig:

- Neue kohärente Komponenten einzuführen
- Vorhandene Package-Grenzen neu zu schneiden
- Veraltete Interfaces zu entfernen
- Alte Laufzeitimplementierungen zu verwerfen
- Tests auf die neue Semantik auszurichten

Am Ende darf nur eine kohärente produktive Apply-Ausführungsarchitektur bestehen.

## D9 – Koordinierte Multi-Chat-Implementierung

Die Umsetzung wird durch einen übergeordneten Codex-Koordinator organisiert.

Der Koordinator verantwortet:

- Architektur-Audit
- Zielstruktur
- Schnittstellenentscheidungen
- Abhängigkeiten
- Arbeitsaufträge
- Fortschritt und Koordination
- Architekturkonformität
- Reviews
- Integration
- Abschließenden Cleanup

Unabhängige Implementierungs-Chats erhalten:

- Eindeutigen Auftrag
- Eigenen Worktree
- Eigenen Branch
- Klare Schreibverantwortung
- Konkrete Abnahmekriterien
- Relevante Schnittstellen
- Lokale Tests

Sie dürfen Subagenten zur Umsetzung, Analyse und Review verwenden.

Der Koordinator ist keine zusätzliche konkurrierende Schreibinstanz für dieselben Komponenten.

**Keine unkoordinierten parallelen Änderungen an gemeinsam verantworteten Dateien.**

---

# 3. Logische Zielarchitektur

## 3.1 Model & Project State

Verantwortlich für:

- Kanonischen Compiler
- Modellidentitäten und Referenzen
- Registry
- Ownership
- Vollständige Dateizuordnung
- Projekt-Snapshots
- Ableitung der Managerorganisation
- Deterministische Kontextselektion

Der Kern bleibt unabhängig von Provider, Container, Git-Kommandos und CLI.

## 3.2 Command Application

Verantwortlich für:

- Einstieg in eine benannte Produktoperation
- Ausgangseingaben
- Vorbereitung
- Gesamtbudget
- Auswahl und Ausführung eines Workflows
- Gesamtergebnis
- Freigabe zur Veröffentlichung

Keine eigene Managerkoordination.

## 3.3 Workflow Orchestrator

Verantwortlich für:

- Run-Komposition
- Ergebnisübergaben
- Verzweigungen
- Korrekturschleifen
- Abbruchbedingungen

Keine Git-, Container- oder Providerlogik.

## 3.4 Universal Run Executor

Verantwortlich für:

- Managerhierarchie
- Assignments
- Delegation
- Kommunikation
- Fragen und Eskalation
- Parallelität
- Reviews
- Integration
- Root-Abschluss

Ein Executor für alle agentischen Mandate.

## 3.5 Harness Builder

Verantwortlich für:

- Zusammensetzung der Harness-Informationen
- Managerkontext
- Markitect-Core-Regeln
- Projektregeln
- Mandatsspezifische Konfiguration
- Providerneutrale Harness-Spezifikation
- Reproduzierbare Harness-Identität

Die Darstellung über Codex- oder Claude-Mechanismen gehört zu den Providerintegrationen.

## 3.6 Workspace Management

Zwei getrennte Verantwortlichkeiten:

**Git Workspace Service:** Veränderliche Arbeitsstände während eines Runs.

**Durable Candidate Storage:** Dauerhafte Aufbewahrung fixierter Git-Kandidaten zwischen Runs.

Beides muss nicht durch unterschiedliche Prozesse oder Container umgesetzt werden.

## 3.7 Run Information

Verantwortlich für:

- Verbindliche Run-Ergebnisse
- Findings
- Technische Beobachtungen
- Shared Notes
- Persistenz und Abruf

Keine fachliche Entscheidung über den nächsten Run.

## 3.8 Publication

Verantwortlich für die sichere Übertragung akzeptierter Kandidaten ins Nutzerrepository.

Prüft die notwendigen Zustandsbindungen und behandelt zwischenzeitliche Änderungen.

Keine fachlichen Korrekturen, keine Managerkoordination.

---

# 4. Unterstützende Infrastruktur

## Provider Adapter

Codex, Claude und gegebenenfalls weitere Anbieter.

Aufgaben:

- Provider starten
- Gespräche verwalten
- Nachrichten austauschen
- Werkzeugaufrufe vermitteln
- Abschlüsse und Fehler beobachten
- Ressourcen schließen

## Container Runtime

Container, Mounts, Ressourcenlimits und technische Lebenszyklen.

Keine fachliche Run-/Workflowlogik.

## Check Runner

Projektchecks, Builds, Tests und sonstige erforderliche technische Prüfungen.

Keine zweite Implementierung des Markitect-Compilers.

## Tool Bridge

Kontrollierte Agentenfähigkeiten für Kommunikation, Delegation, Reviews und Shared Notes.

MCP ist eine mögliche Bereitstellungsform, keine fachliche Architekturvorgabe.

## Presentation

CLI, Cockpit und Berichte.

Alle nutzen die tatsächlichen Anwendungsergebnisse und erfinden keine eigenen Ausführungszustände.

---

# 5. Abhängigkeitsregeln

Die Architektur folgt klaren Verantwortungsgrenzen.

**Nicht zulässig:**

- Compiler abhängig von Codex, Claude oder Docker
- Domain abhängig von Git- oder Containerimplementierungen
- Run Executor abhängig vom Apply-Workflow
- Workflow mit eigener Managerkoordination
- Harness Builder mit Git-Integrationslogik
- Provideradapter mit fachlichen Reviewentscheidungen
- Publisher mit Agentenkoordination
- Shared Notes als zweite kanonische Modellquelle

Die Infrastruktur implementiert die tatsächlich benötigten Verträge der Anwendungs- und Kernkomponenten.

Keine Interfaces, DTOs oder Frameworks ohne konkreten Bedarf.

---

# 6. Zustands- und Lebenszyklusregeln

## 6.1 Kanonischer Zustand

Modell, Registry, Ownership und kontrollierte Projektkonfiguration.

## 6.2 Compiled Project Context

Geprüfte und gebundene Sicht auf die Projektwelt.

Enthält Modellidentität, Organisation, Registry, Ownership und relevante Kontextinformationen.

## 6.3 Run State

Temporärer Zustand von Assignments, Kommunikation, Delegationen und Integration.

Endet mit dem Run.

## 6.4 Candidate

Fixierter Git-Stand.

Überlebt die Container und kann Ausgangspunkt weiterer Runs sein.

## 6.5 Workflow State

Aktueller Kandidat, Run-Ergebnisse, offene Findings und gemeinsames Budget.

Keine persistenten Agentengespräche.

## 6.6 Shared Notes

Persistente operative Beobachtungen ohne kanonische Autorität.

---

# 7. Implementierungsorganisation

## 7.1 Koordinator

Ein zentraler Codex-Chat übernimmt Architektur und Koordination.

Er arbeitet zunächst überwiegend lesend.

Er startet nicht sofort eine große Zahl paralleler Implementierungen.

Zunächst werden die Zielkomponenten und ihre Verträge definiert.

## 7.2 Architektur-Audit

Der Koordinator prüft den aktuellen Main-Stand.

Für jede relevante Komponente:

- Aktuelle Verantwortung
- Kopplung und Abhängigkeiten
- Bestehende Tests
- Vorhandene Garantien
- Redundanzen
- Zuordnung zur Zielarchitektur
- Keep/Refactor/Replace/Remove
- Konkrete Migrationsentscheidung

Die Bewertung muss begründet werden.

Nicht der geringste Änderungsaufwand, sondern die beste langfristige Architektur entscheidet.

## 7.3 Minimale Verträge

Vor paralleler Implementierung werden insbesondere die notwendigen Grenzen stabilisiert zwischen:

- Compiled Project Context
- Run Request und Result
- Mandat
- Candidate und Git-Persistenz
- Findings
- Harness Specification
- Provider Session
- Shared Notes
- Publication

Keine vollständige große Interface-Hierarchie vorab entwerfen.

Nur Verträge definieren, die mehrere Komponenten tatsächlich gemeinsam benötigen.

## 7.4 Parallele Implementierung

Unabhängige Komponenten dürfen parallel entwickelt werden.

Beispielhafte Arbeitsschwerpunkte:

- Run Core
- Harness
- Workspace/Candidate Storage
- Run Results und Shared Notes
- Providerintegration

Die Aufteilung ist anhand des tatsächlichen Codes und der Abhängigkeiten festzulegen.

Bei gemeinsamer Codeverantwortung gilt jeweils genau ein schreibender Owner.

## 7.5 Subagenten

Implementierungs-Chats dürfen Subagenten einsetzen für:

- Codeanalyse
- Gezielte Implementierung
- Tests
- Refactoring
- Unabhängige Reviews

Subagenten dürfen keine Schnittstellenentscheidungen anderer Arbeitspakete unkoordiniert verändern.

## 7.6 Abstimmung

Architektur- oder Schnittstellenprobleme werden an den Koordinator eskaliert.

Der Koordinator entscheidet über Änderungen am gemeinsamen Vertrag und informiert betroffene Implementierer.

Keine lokalen Workarounds, die unnötige globale Komplexität erzeugen.

## 7.7 Integration

Ein fokussierter PR je kohärenter Änderung.

Vor der Integration:

- Relevante lokale Tests
- Architektur- und Dependency-Checks
- Unabhängiger vollständiger Diff-Review
- Behandlung aller relevanten Findings
- Fixierter und überprüfter Commitstand

Integration in sinnvoller Abhängigkeitsreihenfolge.

Der Koordinator überprüft den tatsächlich integrierten Main-Stand, nicht lediglich einzelne erfolgreiche PRs.

---

# 8. Phasenplan

## P0 – Zielarchitektur übernehmen

- Aktuellen Main-Stand erfassen
- D1–D9 und logische Zielarchitektur als Ausgangspunkt verwenden
- DEC-067 und betroffene Konzeptdokumentation konsolidieren
- Historische Entscheidungen nicht stillschweigend als aktuelle Anforderungen übernehmen

## P1 – Architektur- und Code-Audit

- Bestehende Komponenten analysieren
- Keep/Refactor/Replace/Remove-Matrix erstellen
- Alte Pass-/Turn-Mechanik identifizieren
- Neue Verantwortungsgrenzen bestimmen
- Migrationsabhängigkeiten erfassen

## P2 – Schnittstellen und Fundament

- Minimale gemeinsame Verträge festlegen
- Compiler-/Registry-Basis sichern
- Git-Kandidatenidentitäten und Übergaben klären
- Abhängigkeitsrichtung und Paketverantwortung definieren

## P3 – Core-Implementierung

- Universeller Run-Executor
- Managerkommunikation und Delegation
- Workspace Service pro Run
- Candidate Storage
- Providerunabhängige Harness-Zusammensetzung

Hier kann parallel gearbeitet werden, soweit die gemeinsamen Verträge stabil sind.

## P4 – Anwendung und Provider

- Codex-Harness
- Claude-Harness beziehungsweise Providerintegration
- Apply-Workflow
- Assessment, Implement und Verify
- Findings-Korrekturschleife
- Run Results und Shared Notes

## P5 – Produktintegration

- CLI
- Setup
- Cockpit
- Modell und generierte Dokumentation
- Publisher
- Publikationsguards
- Gesamtergebnisberichte

## P6 – Cleanup

- Veraltete Pass-/Turn-Implementierungen entfernen
- Überholte Strategien beseitigen
- Tote Abstraktionen und Sonderfälle löschen
- Doppelte Datenmodelle entfernen
- Veraltete Tests ersetzen
- Dokumentation auf die aktuelle Implementierung ausrichten

## P7 – Fachliche Qualifizierung

- Fresh Library Realization
- M1→M2, nur Ausleihlimit fünf auf drei
- Preservation aller übrigen Verpflichtungen
- Seeded Defect
- Review- und Verify-Korrekturen
- Fehlgeschlagene Checks
- Cross-Run-Git-Ergebnisse
- Publication Conflicts

## P8 – Abschließende Qualitätsprüfung

- Vollständige lokale Tests
- Architektur- und Dependency-Gates
- Race-Tests
- Linux
- Hosted CI
- Reale Provider-End-to-End-Läufe
- Vollständiger Architekturreview
- Feststellung, ob obsolete Implementierungen zurückgeblieben sind

---

# 9. Release-Abnahmekriterien

V1 darf erst als stabil betrachtet werden, wenn:

1. Compile-Fehler jeden Agentenstart verhindern.
2. Assessment, Implement und Verify denselben Executor verwenden.
3. Runs unabhängig von Apply ausführbar sind.
4. Der Workflow seine Runs korrekt orchestriert.
5. Die Managerhierarchie zuverlässig kommuniziert.
6. Kindergebnisse fachlich überprüft und integriert werden.
7. Container und Workspace Service pro Run korrekt verwaltet werden.
8. Kandidaten zwischen Runs nicht verloren gehen.
9. Harnesses reproduzierbar erzeugt werden.
10. Registry und Ownership vollständig geprüft werden.
11. Die vier deterministischen Kontrollpunkte funktionieren.
12. Verify unabhängig über den tatsächlichen Kandidaten entscheidet.
13. Findings zuverlässig an Implement gelangen.
14. Shared Notes persistent bleiben und keine automatischen Aufträge auslösen.
15. Publication nur auf akzeptierte Kandidaten zugreift.
16. Ungültige oder unvollständige Ergebnisse nicht als Erfolg gemeldet werden.
17. Alte produktive Pass-/Turn-Pfade entfernt wurden.
18. Die Implementierung architektonisch kohärent und wartbar ist.
19. Die vorgesehenen Library- und Negativtests bestanden wurden.
20. Die finale bereinigte Runtime erneut vollständig qualifiziert wurde.

---

# 10. Bewusst ausgeschlossen

V1 benötigt nicht:

- Neue allgemeine Workflow-DSL
- Grafischen Workflow-Designer
- Dynamische Managerorganisation
- Selbstoptimierende Agenten
- Langfristig laufende Managercontainer
- Persistente Provider-Gespräche
- Komplexes Agentengedächtnis
- Universelle Code-Mode-Runtime
- Neues formales Evidence-Protokoll
- Per-write-Dateischutz
- Allgemeines Recovery-Framework
- Automatische Commits oder Pushes
- Kostenoptimierung vor Qualitätsnachweis

---

# 11. Leitlinie für die Umsetzung

Die Implementierer sollen nicht versuchen, die neue Architektur möglichst schnell in den vorhandenen Code einzubauen.

Sie sollen die akzeptierte Zielarchitektur sauber implementieren.

Das bedeutet:

**Keep, wenn eine bestehende Komponente gut passt.**

**Refactor, wenn die Verantwortung richtig ist, aber die Struktur verbessert werden muss.**

**Replace, wenn die bestehende Komponente konzeptionell ungeeignet ist.**

**Remove, wenn ein Mechanismus nicht mehr benötigt wird.**

Historische Probleme können anhand der Git-Historie nachvollzogen werden. Ihre alten Lösungen sind keine Pflicht zur Übernahme.

Die Architektur soll am Ende möglichst einfach, kohärent, verständlich und hochwertig sein.

---

# 12. Änderungsprotokoll

| Version | Änderung |
|---|---|
| 0.1–0.13 | Produktkonzept und Architekturentscheidungen D1–D6 |
| 0.14 | Architekturqualität vor Wiederverwendung |
| 0.15 | Logische Zielarchitektur |
| **0.16** | **D8 selektiver Clean-Slate-Rebuild und D9 koordinierte Multi-Chat-Implementierung akzeptiert; verbindlicher Architektur- und Umsetzungsplan für Codex-Koordination erstellt** |

**Nächster Schritt:** Architektur-Audit gegen den aktuellen Main-Stand und anschließende kontrollierte Implementierung entlang der hier festgelegten Verantwortlichkeiten.
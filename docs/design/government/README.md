# Markitect Government: Designpaket

**Aktiviert am 7. Oktober 2026:** Das Nutzer-Go liegt vor. [Aktive Koordination](coordination.md) bindet die tatsächlichen Chats Worker, Scientist und Architect, Arbeitsbereiche und Erstaufträge. Hinweise auf ein noch ausstehendes Go im ursprünglichen Entwurf sind damit historisch.

Stand: 2026-10-07. Design v0.1 und aktiver Forschungsauftrag des Koordinators. Die angenommenen Implementierungsstände und tatsächlichen Versuche stehen in der Koordinationsakte; Architekturentwürfe sind keine Ausführungsnachweise. Der präzisierte [Evaluationsauftrag](evaluation.md#nutzerziel-gedankliches-modell-und-dauerhafte-änderbarkeit) umfasst Modellverständnis, Repository-Qualität, Drift, Autonomie und einen konkreten Bedienvorschlag.

## Ziel und Auftrag

Ein akzeptiertes kanonisches Modell beschreibt die gewünschte Projektwelt. Das gesamte Projekt-Repository ist ihr konkretes Abbild. Eine rekursive Organisation verantwortet Änderungen, deren Umsetzung in Dateien, unabhängige Prüfung und das Zusammenspiel. Sie kann innerhalb expliziter Delegation auch das Modell weiterentwickeln. Der angestrebte Nutzen ist verlässliche, nachvollziehbare Entwicklungsarbeit mit wenig menschlicher Routinekoordination.

Der Nutzer hat dem Koordinator Verantwortung und freien Gestaltungsspielraum für Architektur, Umsetzung, gezielte Folgeversuche und die abschließende Produktempfehlung übertragen. Worker und Scientist arbeiten in getrennten Kontexten; Architect hält seine Classic-Linie für den Vergleich und konkrete Produktbefunde bereit. Das Ziel ist ein brauchbarer Weg vom gedanklichen Modell zur weitgehend autonomen, gut überprüften Projektentwicklung. Veröffentlichungen und die dauerhafte Ablösung von Classic folgen daraus nicht automatisch.

Die Government-Linie ist zunächst eine experimentelle Variante desselben Produkts. Bestehende technische Mechanismen werden nach ihrem Nutzen übernommen. Ihre Namen, vorhandene Paketzuschnitte oder Kompatibilitätsformen schreiben die neue öffentliche Architektur nicht vor. Die Entscheidung über eine dauerhafte Produktaufteilung folgt aus der Implementierung und dem Vergleich, nicht aus dem Namen des Experiments.

## Ein Auftrag im Zielmodell

Beispiel: „Eine stornierte Bestellung muss ihren reservierten Bestand freigeben.“ Der fachliche Graph verbindet Bestellung, Reservierung, Übergangsregeln und ihre Dateien. Der verantwortliche Elternbereich verteilt Teilaufträge an die Bereiche Bestellung und Bestand; beide können HTTP-, Datenbank-, Test- und Dokumentationsdateien verantworten. Technologie bestimmt hier die benötigte Fähigkeit. Sie erzeugt keine eigene Leitungsebene.

```mermaid
flowchart TD
    A[Auftrag und aktive Verfassung] --> B[Betroffene Verantwortung und Pflichten bestimmen]
    B --> C[Rekursiv delegieren und Dateien bearbeiten]
    C --> D[Lokal unabhängig prüfen]
    D --> E[Integrieren und auf Elternebenen prüfen]
    E --> F[Alle ausgewählten Ressorts stimmen zum Gesamtkandidaten ab]
    F --> G[Host übernimmt und berichtet]
    C --> H[Modelllücke oder Konflikt melden]
    H --> I[Zuständige Ebene entscheidet innerhalb ihres Mandats]
    I --> B
```

Die Elternprüfung untersucht, ob Stornierung und Bestandsfreigabe zusammen funktionieren. Ressorts prüfen anschließend ihre übergreifenden Pflichten am selben fertigen Kandidaten. Ergibt sich eine Modelllücke, wird eine begründete Änderung nach der bisher geltenden Ordnung entschieden und erneut umgesetzt und geprüft. Eine ungeklärte Entscheidung hält die davon abhängige Arbeit an; andere autorisierte Aufträge können weiterlaufen.

## Zuständigkeit der Dokumente

| Dokument | Verbindlicher Inhalt innerhalb dieses Designpakets |
|---|---|
| [Architektur](architecture.md) | Begriffe, Modell- und Artefaktverträge, rekursive Verantwortung, Autorität, Kandidaten, Review, Übernahme und Wiederaufnahme |
| [Quellübergang](source-transition.md) | An festen Classic-Quellen geprüfte Wiederverwendung, technische Lücken und Implementierungsreihenfolge |
| [Studienprotokoll](evaluation.md) | Gemeinsame Fälle, drei Versuchsarme, Messung, faire Ausgangslage und unabhängige Bewertung |
| [Liefer- und Koordinationsplan](delivery-plan.md) | Arbeitspakete, Endkriterien, Zuständigkeiten und vorbereitete Aufträge für die zukünftigen Chats |
| [Vorgemerkte Untersuchungsthemen](deferred-research.md) | Nachrangige Ideen zu SAT-inspirierten Modellprinzipien und Jev; keine aktiven Implementierungsaufträge |

Änderungen am fachlichen Design gehören in die Architektur, Änderungen an Bewertungskriterien ins Studienprotokoll. Der Lieferplan verweist darauf und führt keine zweite Version dieser Verträge. Bei Widersprüchen sind Implementierung und Versuch für den betroffenen Vertrag anzuhalten, bis der Koordinator die Dokumente konsistent berichtigt; Regeln werden nicht durch Auswahl des bequemeren Dokuments umgangen.

Die [ursprüngliche Diskussion](../government-of-coding-agents-assessment.md), das [frühere Betriebsmodell](../delegated-engineering-operating-model.md), die [Änderungslandkarte](../delegated-engineering-change-map.md) und der [erste Validierungsplan](../delegated-engineering-validation-plan.md) bleiben Herkunft und historischer Kontext. Neu getroffene Entscheidungen dieses Pakets haben für die geplante Government-Variante Vorrang. Sie ändern keine veröffentlichten Classic-Verträge.

## Leitplanken

- Kanonische Definitionen besitzen einen expliziten `purpose`; Struktur und Referenzen bleiben prüfbar. Ein Zwecktext allein ist kein Qualitätsnachweis.
- Akzeptiertes Soll, beobachtetes Repository, Aufträge und Ausführungsnachweise bleiben unterscheidbar. Beobachtungen schaffen keine eigene normative Autorität.
- Modellgegenstände und tatsächliche Dateien sind in beide Richtungen zuordenbar. Mehrere Verpflichtungen pro Datei sind möglich; unkoordinierte konkurrierende Writer sind es nicht.
- Verantwortungsbereiche strukturieren die Arbeit. Technologien und Renderer sind Fähigkeiten innerhalb dieser Organisation.
- Ein übergeordneter Bereich behält seinen Gesamtauftrag und seine Integrationspflicht. Jede ausführende Ebene wird unabhängig geprüft.
- Dauerhafte Zuständigkeit und vorübergehende Agenteninstanz sind getrennt. Feine Prüfläufe bis zur einzelnen Invariante sind möglich, aber kein Pflichtstandard.
- Delegation erweitert keine eigene Befugnis. Echte Modell- und Architekturänderungen können vorher delegiert werden; Selbstermächtigung und nachträgliches Schönschreiben eines Fehlversuchs sind ausgeschlossen.
- Jedes ausgewählte Ministerium stimmt dem festen finalen Kandidaten ausdrücklich zu. Auslassung, Timeout, veraltete Stimme und ein Gerichtsurteil ersetzen diese Zustimmung nicht.
- Jeder Lauf berichtet auch bei Fehler, Nichtbetroffenheit und Abbruch. Sichtbarkeitseinstellungen ändern weder Aufzeichnung noch Entscheidungsregeln.
- Wiederholungen, Parallelität und Dauerbetrieb haben Grenzen, stabile Zustände und nachvollziehbare Wiederaufnahme.

## Aktueller Nachweisstand

Der technische Ausgangspunkt der Quellanalyse ist der abgeschlossene Classic-Checkpoint `1ea5c76f55526fc4d721e865885436153f48b497`. Der [Abschlussabgleich](../architect-government-checkpoint.md) hält dessen begrenzte Piloten und Quellprüfungen fest. Die Dokumentations-Arbeitskopie auf `codex/government-assessment` enthält ältere Produktquellen und ist nicht als Government-Implementierungsbasis zu verwenden.

G1 bis G4 wurden inzwischen auf den in der Koordinationsakte genannten Quellen als lokale technische Mechanik angenommen; G5 ist beauftragt. Diese Nachweise verwenden deterministische Testakteure und belegen keine reale autonome Engineering-Qualität. Scientist prüft den tatsächlichen Ausführungspfad; Versuchsreife und der Vergleich bleiben gesonderte Nachweise. Aus dem Design oder einer bestandenen Mechanikprüfung folgt keine Überlegenheit gegenüber gewöhnlichen Agenten.

## Übergang nach dem Go

Die Chats und Arbeitsbereiche sind in der aktiven Koordination gebunden. Nach G5 und S1 folgt der Sechs-Zellen-Vergleich mit festgeschriebenem Protokoll. Der erweiterte Nutzerauftrag erlaubt anschließend begrenzte, durch Befunde begründete Komponentenversuche und eine konkrete Empfehlung für Aufbau und Benutzung von Markitect. Zwischenstände werden berichtet; gewöhnliche technische Folgeentscheidungen benötigen keine wiederholte pauschale Freigabe. Echte Zielkonflikte und neue externe Verpflichtungen werden konkret vorgelegt.

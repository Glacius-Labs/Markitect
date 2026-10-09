# Wie im Markitect-Projekt gearbeitet wird

**Quellenbasis:** jüngste Rollen-Chats, geschlossene Arbeitsverläufe, Branch-/Worktree-Inventar und Projektregeln bis 9. Oktober 2026. Die Rollen sind verwendete Arbeitsmuster, keine Behauptung, dass Markitect sie schon als Produktlaufzeit bereitstellt.

## Typischer Arbeitsfluss

1. **Auftrag und Remit klären.** Der Nutzer trennt häufig Produktdiskussion, Architekturvorschlag, Umsetzung, Versuch und Veröffentlichung. Ein angehängtes Konzept ist Input, nicht automatisch eine Anweisung oder eine Produktentscheidung.
2. **Ist-Zustand pinnen.** Pfad, Branch, SHA, Dirty-Files, Abhängigkeiten und der tatsächliche Ziel-Branch werden festgestellt. Besonders relevant: lokales `main` kann hinter dem Live-`origin/main` liegen.
3. **Arbeit isolieren und begrenzen.** Größere Stränge erhalten eigene Branches/Worktrees, klaren Besitzer, Eingaben, Ausgabe und Abbruchbedingungen. Konzepte, Government, Knowledge Graph, Marketing und Integration arbeiten getrennt; parallele Arbeit an gemeinsamen Dateien wird vermieden.
4. **Änderung und Nachweis auseinanderhalten.** Implementierer liefern Kandidaten. Tests, CLI-/Runtime-Beobachtungen, unabhängiges Review, Integrations-CI und menschliche Annahme sind getrennte Stufen.
5. **Zusammenführen, falls freigegeben.** Ein Kandidat wird nicht durch bloßes Pushen, PR-Öffnen, Branchname oder grüne Teilchecks zu Main. Release, Veröffentlichung, Adoption und Akzeptanz brauchen ihre jeweils eigene Evidenz.

## Rollen, wie sie in den Chats verwendet wurden

| Rolle | Gewöhnliche Aufgabe im Projekt | Typischer Nachweis |
|---|---|---|
| **Architect** | Welt-/Datenmodell, Grenzen, Verträge und Arbeitspakete entwerfen | klarer Vorschlag mit betroffenen Quellen und offenen Entscheidungen |
| **Overseer** | Remit, Abhängigkeiten, Reihenfolge, Ressourcengrenzen und Freigabe für nachgelagerte Arbeit koordinieren | pinbarer Auftrag, Zuständigkeit, explizite Gates und sauberer Übergabestatus |
| **Worker / Implementer** | begrenzte Änderung auf eigenem Branch/Worktree umsetzen | Commit/SHA, geänderte Pfade, ausgeführte Checks und nicht erledigte Punkte |
| **Reviewer / Astra** | Kandidaten möglichst unabhängig lesen und zu starke Schlussfolgerungen korrigieren | konkrete Befunde, exakter Quellenstand und Reviewgrenze |
| **Scientist** | fair kontrollierte Versuche und Qualitäts-/Aufwandsbeobachtung führen | gleiche Aufgaben/Ressourcen, frische Sitzungen, unveränderte Fehler- und NOT RUN-Einträge |
| **Coordinator / Integrator** | bestätigte Teilbeiträge komponieren und integrierte Gates organisieren | Liste eingebundener SHAs, Kompositionsdiff, vollständige Gate-Ergebnisse |
| **Monitor** | Chats, Branches, Eigentümer und zeitabhängige Blocker überblicken | datierte Statusaufnahme; keine automatische Quelle der Wahrheit |
| **Marketing / Concepts / Ideas** | Positionierung, Begriffe, Nutzungsszenarien und offene Ideen sammeln | Herkunft, Aussageart und explizite Grenze zwischen Idee und Beschluss |

Rollenberichte können Lücken oder veraltete Pins enthalten. Daher werden konkrete Aussagen gegen den jeweiligen Checkout oder Rohbeleg geprüft. Der Overseer hat beispielsweise einen zu starken C3-FAIL-Schluss eingegrenzt; Reviewberichte schlossen technische Befunde, aber daraus entstand nicht automatisch eine erfolgreiche Produktstudie. Das Knowledge-Graph-Team bewahrte fehlgeschlagene Vollverifikationen und fügte gezielte Korrekturen hinzu, statt den früheren Lauf als bestanden umzubenennen.

## Arbeitsregeln, die sich wiederholen

### 1. Quelle und Snapshot vor Statusaussagen

Der Nutzer fordert wiederholt genaue Branch-/SHA-/Dirty-Angaben und korrigiert veraltete „Main“-Bezüge. Der 9.-Oktober-Snapshot belegt konkret, dass der primäre Checkout auf `codex/government-assessment` bei `d26b494b` stand, während live `origin/main` bei `5be48ce1` lag und das lokale `main` weit zurückstand. Aussagen wie „integriert“, „aktuell“ oder „freigegeben“ brauchen daher Branch und Datum.

### 2. Erhalt von Fehlversuchen

Versuche werden als PASS, FAIL, INTERRUPTED, INCOMPLETE, QUALIFIED oder NOT RUN getrennt. Die zwei Vollsuite-Fehler des Integrationstrangs und zwei Vollverifikationsfehler der KG-Variante blieben erhalten, obwohl gezielte Nachtests bestanden. Fehlstarts und verschwendete Wiederholungen zählen zum tatsächlichen Arbeitsaufwand. Nach langen oder sachlich identischen Volltests wurden Wiederholungen teils bewusst gestoppt und ein frischer kombinierter Test für den akzeptierten Quellstand reserviert.

### 3. Unabhängigkeit und Humanentscheidung

Ein Implementierer kann einen fokussierten Fix melden; ein unabhängiger Reviewer prüft ihn separat. Selbst bestandene technische Checks belegen nur ihren deklarierten Umfang. Sie sagen nicht, dass das Modell die gesamte relevante Absicht enthält, die Agenten sicher arbeiten, das Ergebnis fachlich gut ist, die Adoption gelingt oder der Nutzer es angenommen hat.

### 4. Vergleichsstudien sind eigene Projekte

Conventional/Oldschool, Classic, Design und Government wurden nicht mit wechselnden Anforderungen zu einem Ranking vermischt. Eine belastbare Aussage braucht identische Aufgaben, echten Zugriff und gleiche Ressourcen; Einrichtung, Reparaturen, Abbrüche, Wiederholungen und menschliche Eingriffe gehören in den Aufwand. Sitzungen und Quellstände müssen frisch beziehungsweise eingefroren und nachprüfbar sein. Das aktuelle Material enthält Teilbelege und Berichte, aber keinen gültigen vollständig abgeschlossenen Sechs-Aufgaben-Vergleich. Einzelne Oldschool-Runs oder erfolgreiche Fixture-Prüfungen schließen diese Lücke nicht.

### 5. Korrekturen verändern nicht rückwirkend alte Aussagen

Quellenbindung und additive Korrektur sind wiederkehrend: Rohlogs bleiben erhalten; neue Anmerkungen erklären die Korrektur. Die Government-Evidenzkorrektur erhielt 122 frühere Dateien hashgleich und ergänzte Errata. Eine Worker-Aufarbeitung korrigierte die Trial-Zählung: vier Helper-Aufrufe statt zwei erwarteter, davon drei fehlgeschlagen. Bei vollen Suites wurden nachträgliche Fixture-Korrekturen mit neuen SHAs geführt. Ein Reviewer grenzte einen zu weit gefassten C3-FAIL-Befund ein und fand zugleich einen fehlenden semantischen Reparaturpfad. Bei Produktpositionierung wurde die Markdown-Frontmatter-Option nach der Nutzerkorrektur verworfen, während frühere Notizen nachvollziehbar blieben.

### 6. Produktideen sind nicht Umsetzungsaufträge

Concepts- und Marketing-Notizen halten Nutzerwortlaut, Hypothesen und eigene Deutungen getrennt. Ideen zu Briefings, Cockpit, Monitoring, Delegationsspielraum und Gamification sind ausdrücklich Zukunftsszenarien; der Nutzer vertagte Feinplanung bis zu abgeschlossenen Arbeitspaketen, Case-Study-Zahlen und stabilem Main. Die tägliche Historian-Automation soll eigenständig laufen; sie ist keine Aufforderung für manuelle ACK- oder Weckschleifen.

### 7. Release, Veröffentlichung und Adoption sind getrennte Übergänge

Schon die frühe Versionierung bindet Releases an einen exakten Quellstand, Windows-/Linux-Gates, unveränderliche Assets und Verifikationsmaterial. Ein grüner lokaler Build ist keine Veröffentlichung; ein veröffentlichter Markitect-Tag aktualisiert weder die Tool-Pins eines Consumers noch beweist er dessen Adoption. Konfyra- oder andere project-owned Policy und Releaseentscheidungen bleiben beim jeweiligen Projekt.

Die frühe Distributionsarbeit zeigt, warum der Status präzise benannt wird: Das v0.5-Release-Workflow-Benchmark scheiterte zunächst am abgeschnittenen Action-Pin; ein separat reparierter Lauf lieferte danach Messartefakte, aber keine allgemeine Speedup-Aussage. Die v0.7-WinGet-Manifeste waren lokal geprüft und eingereicht, doch die Microsoft-Katalogfreigabe blieb ein eigener, manueller Gate. Siehe [Produktionsbewertung](../../production-assessment.md), [WinGet-Status](../../winget.md) und [Messmethodik](../../measurement.md).

## Wiederkehrende Reibungspunkte

- **Parallelität erhöht Integrationslast.** Viele spezialisierte Worktrees ermöglichen getrennte Eigentümer, doch ein gemeinsamer Vollgate muss die tatsächliche Zusammensetzung prüfen.
- **Lange Laufzeiten machen kleine Fixture-Fehler teuer.** Zeitbudgets und Test-Fixtures beeinflussen Versuche; punktuelle Testreparaturen sind keine vollständige Wiederholung.
- **Status kann schneller veralten als Doku.** Main, laufender Checkout, Feature-Branch, installierter Release und Benutzerakzeptanz unterscheiden sich.
- **Ein Agentenbaum ist nicht automatisch unabhängig.** Gemeinsame falsche Annahmen, fehlender Kontext oder eine fehlerhafte Übergabe können alle Ebenen beeinflussen. Deshalb sind harte Remits, Quellenpins und unabhängige Prüfung selbst Teil des Verfahrens.
- **Nutzenmessung bleibt offen.** Eine Demo oder ein grüner Build belegt nicht, dass Agenten zuverlässiger arbeiten oder weniger menschliche Kontrolle brauchen.
- **Fehlstarts sind Teil des Befunds.** Frühere Scientist-Verläufe enthalten unterbrochene oder falsch gestartete Fälle (u. a. Readinglog und Roombook); fehlende Abschlussbelege wurden nicht als Qualitätsresultat ausgegeben. Der spätere Vergleichsauftrag blieb vor einer gültigen Vollauswertung stehen.

Diese Beschreibung fasst beobachtete Arbeitspraktiken zusammen. Sie bescheinigt weder, dass jeder Chat alle Regeln erfüllt hat, noch dass der zukünftige Markitect-Controller diese Arbeitsweise bereits technisch erzwingt.

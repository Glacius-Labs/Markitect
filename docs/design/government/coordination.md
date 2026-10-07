# Aktive Koordination: Worker, Scientist und Architect

Stand: 2026-10-07. Der Nutzer hat das Go zur Umsetzung und Koordination erteilt. Dieses Dokument aktiviert das Designpaket `09850535de0c2af5f38b2cfd4f20a62702b676eb` und ersetzt dessen historische Hinweise auf das noch ausstehende Go. Architektur, Nachweisgrenzen und endliche Lieferfolge bleiben bestehen. „Implementer“ im Lieferplan bezeichnet den Chat **Worker**, „Case Study“ den Chat **Scientist**.

## Auftrag und Eigentümer

| Chat | Lokale Chat-ID | Verantwortung |
|---|---|---|
| Overseer | `01a11367-a781-7683-a20f-46e12614dcb4` | Architekturentscheidungen, Auftragsteilung, unabhängige Abnahme, Fortschrittskoordination |
| Worker | `01a1169b-9357-7df3-a8f6-f913557c782a` | Government implementieren und produktbezogene Nachweise liefern; erster Auftrag G1 mit G2-Verträgen |
| Scientist | `01a1169c-3df6-7571-997d-48ab57365875` | Referenzprojekt, drei Versuchsarme, Greenfield/Brownfield, öffentliche Checks, getrennte Bewertung, Metriken und Vergleich |
| Architect | `01a111ec-a913-7830-91b0-925d0a1dd1b2` | Classic stabilisieren, erforderliche Quell-/Releaseprüfungen schließen, aufräumen und reproduzierbares Readiness-Paket für Scientist liefern |

Der Nutzer hat die gezielte Beauftragung und fortlaufende Koordination aller drei Chats ausdrücklich autorisiert. Direkte neuere Nutzeraufträge gehen vor. Ergebnisse werden in den jeweiligen Chats berichtet; Overseer liest sie zurück und übermittelt konkrete Verträge/Befunde. Nachrichten zwischen anderen Chats sind keine Voraussetzung des Ablaufs.

## Arbeitsbereiche und feste Ausgangsstände

- Overseer: `C:/Users/Consiliari/Glacius Labs/Markitect`, Branch `codex/government-assessment`; schreibt hier ausschließlich Design und Koordination.
- Worker: `C:/Users/Consiliari/.codex/worktrees/government-worker/Markitect`, Branch `codex/government-worker`.
- Scientist: `C:/Users/Consiliari/.codex/worktrees/government-scientist/Markitect`, Branch `codex/government-scientist`. Studienquellen liegen unter `experiments/government-comparison/`; private Bewertung und Laufdaten werden gesondert gehalten und nicht in Actor-Kopien übernommen.
- Beide neuen Arbeitskopien beginnen beim geprüften Classic-Commit `1ea5c76f55526fc4d721e865885436153f48b497`. Das Design wird als eigener Dokumentationscommit übertragen; die alte Produktquellbasis des Overseer-Branches wird nicht übernommen.
- Architect verwendet seinen bestehenden isolierten Classic-Kontext. Die zuletzt gelesene Abschlussmeldung nennt v0.14.1 und README-PR #85; die darin noch laufende Main-CI und endgültige Studienreife sind durch seinen aktuellen Auftrag zu bestätigen. Diese Meldung ersetzt keinen StudyClassicVersion-Freeze.

Jeder Chat verwendet bei Datei-/Shell-Zugriffen seinen absoluten zugewiesenen Arbeitsbereich, auch wenn seine Chat-Metadaten weiterhin das gemeinsame Ursprungsverzeichnis zeigen. Keine parallelen Branchwechsel in der Ursprungsarbeitskopie. Keine fremden WIP, Worktrees oder Belege löschen. Bei kollidierenden Schreibgrenzen entscheidet Overseer vor der Änderung.

## Endliche Aufträge

**Worker:** G1 vollständig liefern: versionierte Bereichs-/Ressort-/Mandatsbegriffe, explizite Modellidentitäten, bidirektionale Datei-Realisierung, deklariertes Inventar und lesender konservativer Plan. G2-Verträge für Kandidat, Evidenz, Votum und Übernahme mitdenken; keine simulierte Umsetzung als fertigen Vertikalschnitt ausgeben. Abschluss mit Commit, fokussierten Tests, erforderlichen Gates, unabhängigen Befunden und verbleibenden Grenzen. Overseer bewertet und beauftragt anschließend den konkreten nächsten Schritt G2 bis G5.

**Scientist:** Zunächst gemeinsamen Bestell-/Lagerreservierungsfall, Taskkarten, leeren Greenfield-Start, kontrollierten Brownfield-Stand, öffentliche Checks, Ergebnisformat und ausführbare Versuchshülle erstellen. Öffentliche Adapterverträge und tatsächliche Runner-/Modell-/Budgetvoraussetzungen liefern. Grundgerüst ist C#/.NET 10, HTTP und lokale SQLite; konkrete Pins werden geprüft und fixiert. Keine private Bewertung an Worker/Architect geben und keine Produktfehler selbst in deren Quellen reparieren. Vollständige Live-Vergleiche beginnen erst mit dokumentierten Readiness-Gates, eingefrorenen Produktständen und begrenztem Ressourcenprofil. Vorhandene autorisierte Mittel können nach Coordinator-Freeze genutzt werden; zusätzliche Käufe oder neue kostenpflichtige externe Verpflichtungen folgen nicht aus diesem Auftrag.

**Architect:** Begonnene Release-/Main-Prüfungen und nachgewiesene Classic-Blocker abschließen. Readiness-Paket enthält Quell-/Release-SHA, installierbare Artefakte und Integrität, Voraussetzungen, kopierbare Bedienbefehle, vollständigen Classic-Ablauf, Ergebnis-/Receiptpfade, notwendige manuelle Eingriffe und Alpha-Grenzen. Er besitzt passende Produktregressionstests und einen kleinen Bedien-Smoke. Gemeinsame Fixtures, Fallstudienmethodik und vergleichende Ergebnisse gehören Scientist. Nach Readiness Kandidaten halten; nur konkrete neue Produktbefunde bearbeiten.

## Laufende Steuerung

Overseer prüft kompakte Chatstände mit `wait_threads` und gespeicherten Cursors. Laufende unveränderte Arbeit wird nicht erneut angestoßen. Bei einem neuen Abschluss prüft er tatsächliche Dateien/SHAs und Nachweise, bevor er das nächste begrenzte Paket beauftragt. Ein idle-Status ist kein Abschlussnachweis. Ein fertiger Classic-Kandidat bleibt verfügbar; er wird nicht mit zusätzlichen Experimenten beschäftigt, während Government noch entsteht.

Neue Interface-Befunde werden als öffentlicher Vertrag abgestimmt und an betroffene Eigentümer gegeben. Private Holdout-Inhalte werden dabei nicht weitergereicht. Scientist darf fehlende Fähigkeiten als Readiness-Lücke ausweisen, statt einen Markitect-Arm stillschweigend zu simulieren. Eine Methodik darf im Vergleich verlieren.

Die gemeinsame Studienadapter-Hülle gehört Scientist: Request-/Result-Format, exakter Request-Hash, Versuchsansteuerung, Messung und Ressourcenverwaltung. Worker und Architect liefern die tatsächlichen Produktaufrufe, Fähigkeiten und Lücken, Versionsbindungen und Receipts. Architect muss dafür keinen neuen autonomen Runner oder Studienaufbau implementieren. Der öffentliche Vertrag v1.1 wurde am Scientist-Commit `3f1dcf47aff80b29b8f4f3cc39711adcf2e72b88` gelesen und an beide Produkteigentümer weitergegeben; private Bewertungsdaten waren nicht Teil der Übergabe.

Subagenten sind für klar getrennte Implementierung, Untersuchung und unabhängige Reviews erwünscht. Vermeide gleichzeitige schwere vollständige Prüfwellen, wenn sie auf derselben Maschine um Ressourcen konkurrieren; vorhandene genaue Nachweise nicht ohne Anlass wiederholen. Modellkosten, Wartezeiten und fehlgeschlagene Versuche werden für Studienläufe tatsächlich erfasst, nicht aus Aktivität geschätzt.

Die Beobachtung ist für diese drei Rollen aktiv. Auf ausdrücklichen Nutzerwunsch folgt nach jeder Überprüfung ein kurzer Report, auch bei unverändertem Verlauf: Prüfzeitpunkt in Europe/Berlin, je eine Status-/Fortschrittszeile zu Worker, Scientist und Architect sowie bei Bedarf Blockaden und nächste Koordinationsaktion. Gemeldeter Fortschritt und selbst geprüfter Abschluss bleiben unterscheidbar. Das Intervall bleibt zehn Minuten; unveränderte Arbeit wird nicht erneut beauftragt. Nach dem begrenzten vereinbarten Vergleichsergebnis oder einer echten ausstehenden Nutzerentscheidung pausiert die Beobachtung. Jev bleibt als nachrangige Untersuchung vorgemerkt und wird durch diese Koordination nicht zusätzlich gestartet. Government-Veröffentlichung oder dauerhafte Ablösung von Classic sind keine automatischen Folgen des Go.

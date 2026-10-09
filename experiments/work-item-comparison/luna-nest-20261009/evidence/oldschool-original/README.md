# Conventional Nest: Übergabevertrag

Paket `oldschool-luna-nest-contract-20261009`, 9. Oktober 2026.
Fallunabhängiger Beitrag zur gemeinsamen Vorgabe
`Markitect/docs/design/work-item-comparison-20261009.md`.
Vorbereitung abgeschlossen; kein Trialgrant, keine ausgeführte Nest-Validierung.

## Inhalt und Zuständigkeit

- `AGENTS.md`: gewöhnliche Projektarbeit, Fortschritt, Tests, Review und Integration.
- `config.toml`: angeforderte native Luna-High-Konfiguration einschließlich Subagenten.
- `Invoke-NestTurn.ps1`: ein nativer Actor-Aufruf pro mechanischem Dispatch.
- `final.schema.json`: gemeinsames lesbares Abschluss-/Fortsetzungsformat.
- `cell.example.json`: bewusst nicht ausführbar freigegebene Zellkonfiguration.
- `native-metadata.json`: in diesem Paket verifizierte Metadaten und Prüfgrenzen.

Scientist besitzt Seeds, fachliche Anforderungen, Backlog, gemeinsame Gates,
Budget und Bewertung. Dieser Beitrag enthält keine Falllösung, Architekturvorgabe,
fachliche Evaluatorhilfe oder Bewertungsrubrik. Er implementiert keine eigene
Produktversion. Der Actor entscheidet über Planung, Lösung, Aufteilung, Review
und Reparatur. Der Dispatcher liest keinen Quellcode und verbessert keine Prompts.

## Einmalige Vorbereitung durch Scientist

Im später konkret freigegebenen isolierten Repository existieren `main`,
`BACKLOG.md` und die fachlichen Projektregeln. Das Backlog ist endlich und erklärt
Reihenfolge/Abhängigkeiten und Abnahmekriterien. Originalregeln bleiben erhalten:
die beiliegenden allgemeinen Hinweise als Abschnitt in die lokale `AGENTS.md`
übernehmen. Keine Regeln des Markitect-Produktcheckouts übernehmen.

`PROGRESS.md` ist der vom Actor gepflegte, versionierte Anker. `.work/logs/` wird
im gemeinsamen Seed ignoriert, aber vollständig in privater Evidenz gesichert.
Im Nest liegen nur bekannte Anforderungen und Arbeitsartefakte; Holdouts,
fremde Lösungen und Bewertungsdaten bleiben außerhalb des Actor-Kontexts.

Pro Zelle einen eigenen nativen `CODEX_HOME` mit bestehender zulässiger
Authentifizierung vorbereiten. Die Konfiguration aus `config.toml` übernehmen;
zusätzliche erforderliche öffentliche Provider-/Toolsettings vorab einfrieren.
Keine Authentifizierungsdateien oder Umgebungsgeheimnisse in Evidenz kopieren.
Keine persönlichen Memories, fremden Sessions, globalen Skills oder fachlichen
Hooks aus anderen Armen einschleusen. Auch Vorfahren-`AGENTS.md`, Projekt-
`.codex`-Layer, Rollenoverrides und verwaltete Konfiguration gehören zur Prüfung
der tatsächlichen Eingaben. Der Launcher übernimmt nur die genannte Home-Auswahl
und Modell-/Reasoning-/Approval-/Sandbox-Overrides; er beweist keine vollständige
Isolation aller Konfigurationsquellen. Geteilte Hostrechte sind keine OS-Isolation.

Die Beispielzelle außerhalb des Actor-Repositories vervollständigen: exakte
Repository-/Evidenz-/Home-/Exe-Pfade, eingefrorene Exe- und Konfigurationshashes,
initialen `main`-SHA, UTC-Endzeit und positive endliche äußere Turn-/Zeitgrenzen.
Erst der konkrete Overseer-Ausführungsauftrag erlaubt `execution_authorized=true`
und die zugehörige `grant_id`. Dieses Feld dokumentiert Autorisierung; es erzeugt
sie nicht. Scientist friert auch Launcher, AGENTS, Schema, Backlog, Tools,
Konfigurationslayer und Seed-SHA vor Beginn ein.

## Ausführbarer Einstieg und Fortsetzung

Erst bei späterem Actualgrant, mit PowerShell 7:

```powershell
pwsh -NoProfile -File .\Invoke-NestTurn.ps1 -Cell C:\isolated-cell\cell.json -Mode Start
pwsh -NoProfile -File .\Invoke-NestTurn.ps1 -Cell C:\isolated-cell\cell.json -Mode Continue
```

Der erste Aufruf übergibt genau den gewöhnlichen Nutzerauftrag aus der gemeinsamen
Vorgabe. Jeder Folgeaufruf verwendet denselben festen Satz zur Fortsetzung anhand
von Backlog, Fortschrittsanker und Repository. Die einzige variable Ergänzung
nennt das verbleibende Zeitfenster. Keine Analyse von Actorfehlern, Quellcode oder
Reviewbefunden im äußeren Prompt. Beide Arme erhalten denselben Top-Level-Auftrag;
native projektlokale Hinweise unterscheiden sich entsprechend der Methode.

Jeder Dispatch startet einen frischen nativen `codex exec` im selben isolierten
Arbeitsrepository. Fortsetzung erfolgt aus expliziten Dateien und Git; es gibt
keinen Zugriff auf die Vorgeschichte dieses Koordinationschats und kein `--last`,
das eine fremde Session auswählen könnte. Native Session-IDs und Ereignisse werden
als Evidenz erhalten. Frische Kontexte je Zelle und deren Identität sind Scientists
Verantwortung; frische Fortsetzungs-Turns innerhalb der Zelle sind bewusst erlaubt.

Mechanische Zustandsübergänge für Scientists äußeren Dispatcher:

| Ergebnis | Feste nächste Aktion |
| --- | --- |
| Exit 0, Schema gültig, `continue` | Einmal `Continue`, solange gemeinsame Quote und Deadline reichen. |
| Exit 0, `completed` | Keine weitere Actor-Ausführung; Stand einfrieren und unabhängig prüfen. |
| `blocked` | Einfrieren, konkrete Blockade melden; keine fachlich ergänzte Fortsetzung. |
| Exit 124, kein terminaler Abschluss | Höchstens eine gewöhnliche `Continue`-Wiederaufnahme nach Timeout pro Zelle, wenn Restbudget besteht. |
| Sonstiger Exit/ungültiger oder fehlender Abschluss | Infrastruktur-/Protokollfehler melden, keine adaptive Reparatursitzung durch Scientist. |
| Deadline oder gemeinsame Quote erschöpft | Stoppen, vollständigen vorhandenen Stand und unvollständige Arbeit sichern. |

Die Einmalregel nach Timeout und das gemeinsame Budget setzt der Dispatcher um;
der dünne Launcher ist kein vollständiger Scheduler. Er sperrt parallele äußere
Dispatches derselben Evidenzwurzel, begrenzt native Prozesslaufzeit/Turnanzahl und
verweigert automatische Fortsetzung nach einem terminalen Actorbericht. Vorherige
Protokollfehler werden nach der Tabelle behandelt, nicht durch Dateiedits umgangen.
Ein erneuter Start mit leerem Evidenzverzeichnis ist kein zulässiges Quotenreset.

## Abschluss- und Evidenzvertrag

Der Actor darf während eines Turns das ganze verbleibende Backlog abarbeiten.
`continue` meint gewöhnliche offene Arbeit, `blocked` eine konkret benannte
unauflösbare Blockade, `completed` den beanspruchten vollständigen Abschluss.
`final.json` enthält Status, erledigte/offene Items, `main`-SHA, Feature-/Merge-SHAs,
Checkbefehle, Exitcodes, geprüfte SHAs und Logpfade. `completed` erfordert leere
Restliste/Blockade und erfolgreiche Pflichtchecks am endgültigen integrierten
Stand. Derselbe finale SHA wird außerhalb eines Commits berichtet, um keinen
selbstreferenziellen SHA im versionierten Fortschrittsanker zu verlangen.

Der Launcher speichert pro Turn exakten Prompt, Argumentliste, Zellmanifest,
angeforderte Einstellungen, Hashes, rohe JSONL-Ereignisse, stderr, finalen Actortext,
Exitcode, UTC-Zeitpunkte, gemessene native Wandzeit sowie `main` vorher/nachher und
Git-Status. Schemagültigkeit ist nur Protokollvalidität. Der Launcher liest keine
Anwendungsquellen, wählt keine Testbefehle und bescheinigt weder korrekte Checks
noch fachlichen Erfolg. JSON-Abschluss, Exit 0 und realer Merge sind unterschiedliche
Nachweise. Scientist sichert auch fehlgeschlagene und unterbrochene Turnverzeichnisse
sowie alle Actorlogs und uncommitteten/untracked Änderungen.

Nach Einfrieren prüft Scientist mindestens die tatsächliche `main`-Historie:
Mergecommit mit zwei Eltern, gemeldeter Featurecommit als Vorfahr des Mergecommits,
Mergecommit als Vorfahr des finalen `main`, Fortschritt gegenüber Seed und passende
Logs/Check-SHAs. Die gemeinsame unabhängige Bewertung prüft danach Funktionen,
Regeln, Regressionen und Qualität anhand ihrer vorher festgelegten Kriterien.
Ein Actorbericht ist keine unabhängige Bestätigung und keine menschliche Abnahme.

## Native Schnittstelle, Messung und Grenzen

Lokal tatsächlich gelesen bzw. nichtmodellgestützt abgefragt: npm-Paket
`@openai/codex` 0.130.0, native Windows-x64-Paketversion 0.130.0-win32-x64,
`codex-cli 0.130.0`, `codex exec --help`, PowerShell 7.6.5 und native Exe-SHA256.
Die lokale Help bestätigt `exec`, Modell-/Konfigurations-/Sandboxoptionen,
stdin-Prompt, `--json`, `--output-schema` und `--output-last-message`.
Die [offizielle Non-interactive-Dokumentation](https://learn.chatgpt.com/docs/non-interactive-mode)
beschreibt JSONL-Events einschließlich Usage sowie strukturierte Abschlussausgabe.
Die [offizielle Konfigurationsreferenz](https://learn.chatgpt.com/docs/config-file/config-reference)
benennt Reasoning und Standardmodell/-Reasoning für Subagenten.

Angefordert sind überall `gpt-6-luna` / `high`; die tatsächliche Modellauflösung,
Annahme der Subagent-Konfigurationsschlüssel in 0.130.0 und Profile aller Kinder
sind noch nicht praktisch nachgewiesen. Rollenoverrides dürfen diese Vorgabe nicht
ändern. Scientist muss diesen einen offenen nativen Profilnachweis beim später
autorisierten Einstieg aus realen Receipts/Sessionmetadaten festhalten. Bei
Abweichung stoppen und melden; kein automatischer Modell- oder Transportwechsel.
Es wurde kein fehlendes Werkzeug nachgewiesen und kein Probe-Actor gestartet.

Usage wird aus vorhandenen Rohreceipts erfasst, nicht geschätzt. Eltern- und
Kindreceipts separat zuordnen; ohne belegte Aggregationssemantik weder addieren
noch Vollständigkeit behaupten. Alle Reviews, Reparaturen, Kindstarts und Fehler
zählen zum gemeinsamen Gesamtbudget. Fehlende Receipts bedeuten unknown, nicht 0.
Kosten bleiben ohne gültige Preisbasis unknown. Wandzeit, Agentenzeit und
Menschenzeit bleiben getrennt. Der Launcher erzwingt weder Tokenlimits noch die
gesamte transitive Rollenquote; deren endliche Durchsetzung gehört zu Scientists
gemeinsamer Mess-/Prozessumgebung. Prozessbaum-Kill ist keine belegte Garantie
gegen entkoppelte Kinder; dies bleibt eine offen auszuweisende Containmentgrenze.

Hier wurden ausschließlich Vorgabe/Dokumentation/Metadaten gelesen und diese
Beitragsdateien geschrieben. Kein Actor, Provider-/Appserver-Probe, Test, Build,
Installation, Kauf, Reviewerkind oder Studienlauf. Launcher und Protokoll wurden
von ihrem Autor gelesen, aber weder ausgeführt noch separat syntax-/laufzeitgeprüft.
Die gemeinsame Vorgabe nennt zusätzlich einen späteren Readonly-Review des
öffentlichen Scientist-Nestvertrags. Dessen konkreter Dateipfad wurde noch nicht
übergeben; es wurden dafür keine privaten Scientist-Dateien gesucht oder gelesen.

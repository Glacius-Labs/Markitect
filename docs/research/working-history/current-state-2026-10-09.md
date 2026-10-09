# Quellen- und Worktree-Stand am 9. Oktober 2026

**Zeitpunkt:** 2026-10-09 20:29:01 UTC. Live- und Arbeitsbaumabfragen liefen dicht nacheinander; parallele Worktrees können sich während einer solchen Aufnahme weiterbewegen.

## Welche Quelle gilt wofür?

- **Öffentlicher Main:** `origin/main` und `git ls-remote origin refs/heads/main` stimmten auf `5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e` überein. Das ist der kanonische Stand für die öffentliche Produktlinie. Kein Fetch wurde ausgeführt.
- **Veralteter lokaler Branch:** Das Branchlabel `main` zeigte in dieser Historian-Clone auf `d4a07704`, 521 Commits hinter `origin/main`. „Lokal auf main“ wäre deshalb eine falsche Kurzform.
- **Primärer Benutzercheckout:** `codex/government-assessment`, `d26b494b19f345aea3917679b6a1cc792ed8492e`, mit vier untracked Dateien unter `docs/design/concepts/`. Diese Entwürfe sind nicht Teil von Main.
- **Historian:** eigener Branch/Worktree `codex/markitect-historian-working-history` am dokumentierten Stand; alle Historikeränderungen bleiben hier.

## Aktive getrennte Kandidaten

Die vollständige zeitpunktbezogene Tabelle enthält Pfad, Branch, SHA und Dirty-Pfade: [Worktree-Snapshot](git-worktrees-current-2026-10-09T20-29-01Z.tsv).

Wichtigste Zuordnungen in diesem Snapshot:

| Spur | Stand | Was der Stand belegt |
|---|---|---|
| Product Integration | `fc6d09a2`, sauberer separater Worktree | Großer Integrationskandidat mit gezielten Folgefixes; finaler Voll-/Plattformgate noch offen |
| Readiness candidate fix | `1756a669`, sauber | Test-only App-Server-Timeout-Korrektur und gezielter Paketnachweis; kein vollständiger integrierter Suite-PASS |
| Knowledge Graph | `ebe51470`, dirty | Gesamtverifikation fehlgeschlagen; letzter gezielter ProjectRun-Test bestanden; vollständiger Lauf danach noch offen |
| Government assessment | `d26b494b`, primärer Checkout | Datiertes Forschungsurteil, nicht automatisch Main-Entscheidung |
| Product App Server / MCP / Workspaces | getrennte Branches `8b3f69ec`, `6e4b620f`, `4840bac0` | Teilfähigkeiten mit eigenen Verträgen und Prüfungen; nicht gleichbedeutend mit dem fertigen Gesamtprodukt |
| Concepts / Marketing | eigene Worktrees auf `5be48ce1`, jeweils dirty | Quellgebundene Notizen und Hypothesen, noch keine Integration |
| Ideas | separater Branch, im Snapshot zeitabhängig | Neue Nutzerideen dokumentiert, keine Featurefreigabe |

Am Integrationsstand waren A01/A02/A03 nicht gestartet; der vorbereitete A01-Plan war noch keine Ausführung. Der vollständige finale Go-Lauf und Linux-/Windows-CI waren offen. Die unabhängige KG-Spur hatte eine echte providerfreie CLI/MCP-Reise mit statischen Fixtures nachgewiesen. Ihre erste Vollverifikation auf `d360ec01` scheiterte an zwei projectcli-Fixtures. Nach deren Korrektur scheiterte der Lauf auf `ebe51470` am ProjectRun-Test `TestDeliverResumesIntegratedRunAndCompletesAcknowledgedScope`: das Fixture lief 546 Sekunden, länger als sein vierminütiges Limit unter Suite-Last. Der gezielte Test bestand danach mit einem auf 15 Minuten angehobenen Fixture-Limit; Produktionsgrenzen blieben unverändert. Eine vollständige Verifikation nach diesem letzten Fixture-Fix stand aus, der Gesamtstatus blieb FAIL. Dies sind Branchberichte und keine Aussagen über Main.

## Chats und Abdeckung

Das Archivverzeichnis lieferte 243 Einträge auf fünf vollständig gelesenen Indexseiten. 25 Projektkandidaten wurden als Verlauf ausgewählt; ein weiterer davon – der Agent-Designer-Chat vom 1. Oktober – wurde in dieser Erweiterung ausgelesen. Zusammen umfasst der Historian nun **39 ausgewählte Verläufe**, jeweils bis zum letzten verfügbaren Seiten-Cursor. Die aktuelle Liste aktiver Chats lieferte **50 Einträge ohne Cursor**; der [aktive Snapshot](active-project-threads-2026-10-09T20-29-01Z.json) hält die 12 als Markitect oder unmittelbar zugehörig erkannten Verläufe fest. Weitere aktive Chats können außerhalb dieses Fensters liegen.

Der Agent-Designer-Chat enthält einen historischen Downloads-Anhang, der am damaligen Pfad nicht mehr vorhanden ist. Bei einigen Live-Verläufen sind Nachrichtentexte in der API leer; dort werden IDs und Agentenstatus nicht als wörtliche Nutzeräußerung ausgegeben. Der Integrator-Verlauf ist lückenhaft.

## Änderungs- und Prüfgrenze dieser Aufnahme

Historian hat hier lokale Markdown-/JSON-/TSV-Dokumentation geschrieben und JSON-/Git-Formatprüfungen ausgeführt. **Keine Produkt-Tests, Build, A01-Rolle, Agentenlauf oder Studie wurden in dieser Aufnahme gestartet.** Keine fremde Worktree-Datei wurde geändert. Es gab keinen Push, Main-Merge, Release oder Versand neuer Befunde an andere Chats.

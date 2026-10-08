# Runner-Metadaten: 0.162.0-alpha.2, Schema geändert

Die zwei erlaubten CLI-Metadatenaufrufe waren positiv. Der Kandidat meldet **`codex-cli 0.162.0-alpha.2`** und erzeugte **447 JSON-Schemadateien, 4.318.845 Bytes**. Von zwölf benötigten Schemaformen sind **acht bytegleich, vier geändert**. Das ist Metadaten- und statische Schemaevidenz; RPC-/Tool-/OS-/S1-Fähigkeit bleibt unbewiesen.

Grant `s1-runner-rebinding-metadata-20261008-r1`, issued `2026-10-08T15:29:49Z`; Basis `2e1d41bd120eda4b83336c3b5ad9fdaf4dda72f2`. Quelle S `e67bcbe55dc266b02ce669106e44459580d19c9f`, Bindungscommit `6c22203a6635fc7cfff7e9639d7d0cbdf2507176`. Der unabhängige Vorabreview prüfte exakte Quellen, Grant/Slot, argv, Fristen und Anfangszustand. Der Ausgaberoot fehlte vor Vorbereitung und wurde exklusiv angelegt; der öffentliche Zwei-Dateien-cwd blieb bytegleich.

## Tatsächliche Ausführung

Binary: `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe`, SHA-256 **`3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`**. Dieser Hash wurde vor und nach jedem Aufruf bestätigt. Die neue Version wird nicht als alte 0.160.1 ausgegeben.

| Exakte Argumente nach dem Binarypfad | Exit | stdout / stderr | gemessene Aufrufzeit / Grenze | Cleanup |
|---|---:|---:|---:|---:|
| `--version` | 0 | 26 / 0 Bytes | 0,2253 / 5 s | 0,1920 s |
| `app-server generate-json-schema --experimental --out C:/Users/Consiliari/Documents/Scientist-Probes/s1-runner-rebinding-metadata-next/schemas` | 0 | 0 / 0 Bytes | 0,5513 / 20 s | 0,2339 s |

Controller 1,3056 s; gebundene Ausführung samt Receipt-Readback **1,3196 / 45 s**. Der äußere Toolaufruf einschließlich Shell/Quellvorprüfung dauerte 2,0323 s. Kein Retry, kein beobachteter Mengen-/Fristenovershoot. Die Ausgabe lag unter 32 MiB/2.048 Dateien; Polling war keine harte Diskquota. Nur die eng validierte Versionszeile wurde als Text behalten, freie stdout-/stderr-Rohlogs wurden nicht persistiert. Windows Job vor Worker-GO, Job bei Abschluss geschlossen; das begrenzt Prozesslebensdauer, nicht Dateisystemzugriff, globale Nebenwirkungen oder Providerarbeit.

## Konkreter Vergleich mit dem gepinnten 0.160.1-Archiv

| Status | Schema-Mitglieder |
|---|---|
| Bytegleich | `v1/InitializeParams`, `v1/InitializeResponse`, `v2/ConfigReadParams`, `v2/ConfigReadResponse`, `v2/ConfigRequirementsReadResponse`, `v2/ThreadStartParams`, `v2/ThreadStatusChangedNotification`, `v2/ThreadClosedNotification` (jeweils `.json`) |
| Geändert | `ServerNotification.json`, `v2/ErrorNotification.json`, `v2/ThreadStartResponse.json`, `v2/ThreadStartedNotification.json` |

Die Top-Level-Properties und Required-Mengen der vier geänderten Formen sind gleich. In den referenzierten Typen wechselt **`CodexErrorInfo` von `oneOf` zu `anyOf`**, ergänzt um einen allgemeinen `string`-/`object`-Fallback. Der ServerNotification-Verbund ergänzt `ThreadPredictionResult` und `ThreadPredictionUpdatedNotification`; `McpServerOauthLoginCompletedNotification` erhält das optionale `loginId` vom Typ string/null.

Die syntaktischen Methoden-/Notification-Mengen ändern sich konkret: ClientRequest **167 → 170**, neu `account/bedrock/checkGovCloudRequirements`, `thread/attachmentOwner/list`, `thread/prediction/request`; ServerNotification **83 → 84**, neu `thread/prediction/updated`. ClientNotification bleibt 1, ServerRequest 11; keine entfernten Methodenliterale. Eine neue ungelistete Notification bleibt unter der unveränderten konservativen alten Allowlist terminal.

Die erreichbaren lokalen Definitionen wurden verglichen; keine unaufgelösten lokalen Referenzen oder dem vorhandenen Parser unbekannten Schema-Schlüsselwörter in den zwölf neuen Formen gefunden. Der Parser wurde weder ausgeführt noch angepasst. Das beweist keine vollständige Parser- oder RPC-Kompatibilität. Die 17 Runtime-Konfigurationspaare, gemeldete Policies und stderr-Strukturallowlist wurden durch diese Metadatenbefehle nicht praktisch geprüft. Bestehende Binary-/Schema-/Clientpins sind nicht automatisch neu gebunden.

Die [offizielle OpenAI-Dokumentation](https://learn.chatgpt.com/docs/app-server) begründet versionsspezifische Schemaerzeugung; [Developer Commands](https://learn.chatgpt.com/docs/developer-commands) beschreibt `--experimental`. Konkrete Versions-/Schemabelege stammen hier aus den zwei gebundenen Ausgaben, nicht aus allgemeiner Dokumentation.

## Belege und geschlossene Grenze

[Terminale Zähler und Zeiten](../evidence/s1-runner-rebinding-metadata-20261008-r1/terminal-summary.json), [447-Dateien-Manifest](../evidence/s1-runner-rebinding-metadata-20261008-r1/schema-manifest.json), [Schema-Diff samt Typen](../evidence/s1-runner-rebinding-metadata-20261008-r1/schema-diff.json), [Schemaarchiv](../evidence/s1-runner-rebinding-metadata-20261008-r1/generated-schemas.zip), [zehn externe Belegpaare](../evidence/s1-runner-rebinding-metadata-20261008-r1/external-archive.json), [unabhängiger Nachreview](../evidence/s1-runner-rebinding-metadata-20261008-r1/postreview.md).

Schemaarchiv SHA-256 `3157e8a55329cf4e5346676c3ef0c308402d2420f8f916d6a5b7e0e9c84356ad`; altes Archiv `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. Alle 13 Quellpins nach Ausführung unverändert. Keine bestehenden Clients/Parser/Produkte geändert, keine Tests oder breite Historienneuverifikation.

Genau zwei neue CLI-Metadatenaufrufe, kumulativ **acht**. Null neue App-Server-Sitzungen, RPC, Actorreservationen, Threads, Turns, Tools, Produkte/Produktwrapper/Fixturedelegates/Studienzellen. App-Server-Bäume/Actorreservationen bleiben sechs/sechs; native Produkthistorie 15/16/13/2250. 53.331 bekannte historische Tokens sind keine Gesamtnutzung. Native interne Aktivität, Usage, Serving und Billing bleiben unbekannt.

[Explizite lokale Slotfreigabe](../evidence/s1-runner-rebinding-metadata-20261008-r1/slot-release.json) am `2026-10-08T15:39:33.690152+00:00`; kanonische Aktualisierung bleibt Overseer-owned. Grant und Restallokation geschlossen, keine automatische Folgediagnose oder Runtimefreigabe. S1 offen, sechs Studienzellen **NOT RUN**. Keine Aussage zur Qualität von Conventional, Classic oder Government; kein Push, PR oder Release.

# Codex-Runner: Offlinebefund und nächster Bindungsschritt

Auftrag `s1-runner-rebinding-offline-20261008`, autorisiert `2026-10-08T15:16:04Z`; Basis `73b263c37b471be39fc8dad37311889a0be26393`. **Ein vorhandener Kandidat, andere Bytes als der alte Pin; Version und Schema-/RPC-Kompatibilität offen.** Codex wurde nicht ausgeführt.

## Beobachtung am 8. Oktober 2026 gegen 15:17 UTC

| Merkmal | Befund und Herkunft |
|---|---|
| Exakter Pfad | `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe` |
| SHA-256 | `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68` — `Get-FileHash -Algorithm SHA256` |
| Größe / Änderungszeit | 333.357.008 Bytes / `2026-10-07T20:15:47.4215883Z` — Dateimetadaten; kein Versions-/Installationsbeweis |
| PE-Version | `FileVersion`, `ProductVersion`, `FileDescription`, `ProductName`, `OriginalFilename` leer/null — `FileInfo.VersionInfo`; kein lesbarer Versionsstring |
| Desktop-Zuordnung | Ausschließlich `Win32_Process.ExecutablePath` für bestehende Prozesse namens `Codex.exe` abgefragt: eindeutiger Imagepfad entspricht diesem Kandidaten. Keine Argumente, Umgebungs-, Account- oder Credentialdaten gelesen. |
| Suchgrenze | `rg --files .../OpenAI/Codex -g codex.exe`: ein Treffer. Kein zweiter Kandidat untersucht; keine globale Maschineninventur. Momentaufnahme, kein dauerhafter Pin. |

Der alte Pfad `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe` fehlt weiterhin. Sein Hash `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916` stimmt mit dem gefundenen Kandidaten nicht überein. Das beweist keine globale Abwesenheit des Hashs und keine Versionsnummer oder Verschwindensursache. Der Kandidat wird nicht als die alte gepinnte 0.160.1-Binary bezeichnet.

## Bekannte und offene Kompatibilität

Das vorhandene 0.160.1-Schemaarchiv hat SHA-256 `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. Der vorbereitete [Diagnosevertrag](../evidence/s1-thread-start-diagnostic-20261008-r1/schema-contract.json) bindet zwölf Mitglieder für initialize/config/read/configRequirements/read/thread/start und enge Lifecycle-/Stopformen. Die alte Threadstartform enthält `permissions`, `ephemeral`, `modelProvider`; ihre Antwort verlangt unter anderem `sandbox`, `approvalPolicy`, `cwd`, `model`, `modelProvider`, `thread`. Das ist alte Schemaevidenz, keine Kandidatenbeobachtung.

Die [offizielle OpenAI-Dokumentation](https://learn.chatgpt.com/docs/app-server) beschreibt versionsspezifisch erzeugte JSON-Schemas und `experimentalApi` für experimentelle Felder; [Developer Commands](https://learn.chatgpt.com/docs/developer-commands) nennt dafür `--experimental` bei der Schemaerzeugung. Allgemeine Dokumentation belegt keine konkrete Kandidatenkompatibilität.

Die engen Clients pinnen alte Binary, Schema und genaue Profile: Ein Pfadtausch erfüllt diese Bindungen nicht. Offen bleiben die zwölf Formen samt referenzierten Typen, Methoden-/Notification-Enums, Parser-Schlüsselwörter, 17 Konfigurationspaare, Policyfelder und stderr-Strukturallowlist. Auch gleiche Schemas würden weder tatsächliche RPC-/Toolfähigkeit noch OS-Isolation beweisen.

## Genau ein vorgeschlagener nächster Schritt — jetzt nicht ausgeführt

**Separat autorisierte Metadatenbindung dieses Kandidaten**, höchstens zwei sequenzielle CLI-Aufrufe, keine App-Server-Sitzung oder RPC. Exakte vorgeschlagene argv; Ausgabepfad jetzt nicht angelegt:

```json
["C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe", "--version"]
["C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe", "app-server", "generate-json-schema", "--experimental", "--out", "C:/Users/Consiliari/Documents/Scientist-Probes/s1-runner-rebinding-metadata-next/schemas"]
```

Vorgeschlagene Grenzen: 5 s für Version, 20 s für Schema, insgesamt einschließlich Auswertung 45 s, Cleanup höchstens 5 s, Parallelität 1, Retries 0. Binaryhash vor/nach jedem Aufruf identisch prüfen. Höchstens 32 MiB/2.048 Schemadateien sowie 64 KiB stdout und 16 KiB stderr transient; begrenzte redigierte Metadaten statt allgemeiner Rawlogs. Ein neuer Auftrag muss Ausgabebereich und Erfassung ausdrücklich binden. CLI-Flagunterstützung dieses Kandidaten bleibt ungeprüft.

Erste fehlende/geänderte Binary, unerwarteter Ausgabebestand, ungültige Version, nichtleeres stderr, Nonzero-Exit, Frist-/Mengengrenze, fehlendes Schema oder unaufgelöste Vertragsabweichung beendet den Schritt ohne Retry, Help-Fallback oder Runtime-Start. Neue Pins: Binarypfad/-hash, gemeldete Version, argv/Fristen, Schema-Manifest/Mitgliedshashes und konkret verglichene bestehende Client-/Profil-/Parserbytes. Erkenntnisgewinn: Versionsidentität und byte-/strukturbezogener Diff der zwölf Formen samt Typen und Methoden-/Notification-Enums — gleich, konkrete Abweichung oder unaufgelöst. Keine vorsorgliche Parser-/Clientplattformänderung. Kein Binarysnapshot wird jetzt kopiert; Drift beendet die vorgeschlagene Prüfung.

Das wäre ein kurzer Schritt zur gleichen tatsächlichen Ausführungsumgebung für Conventional, Classic und Government, ohne Methodenqualitätsaussage oder vorweggenommene Startfreigabe.

## Abschluss

Dieser Auftrag: **null Codex-/CLI-/Metadaten-/App-Server-/Actor-/Modell-/Thread-/Turn-/Produkt-/Controller-/Wrapper-/Delegate-/Studienstarts, Reservationen, Freezes, Ledgeränderungen oder Tests.** Keine Binarykopie, Installation, Wiederherstellung, Config-/Trust-/Policy-/Produktsourcenänderung. Genau zwei neue Dokumentationsdateien einschließlich [unabhängigem Plausibilitätsreview](../evidence/s1-runner-rebinding-offline-20261008/review.md); kein breiter Historienaudit.

Übernommener akzeptierter Stand von `73b263c`: sechs App-Server-Bäume, sechs Actorreservationen, sechs CLI-Metadatenaufrufe; native Geschichte 15/16/13/2250. 53.331 bekannte historische Tokens, Gesamtnutzung/native Inferenz/Serving/Billing unbekannt. S1 offen; sechs Zellen **NOT RUN**. Alter Grant/Restquote geschlossen. Sauberer Dokumentationscommit, genau eine Abschluss-/Bedarfsmeldung, kein automatischer Folgeaufruf.

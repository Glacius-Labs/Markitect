# Öffentlicher Conventional-App-Handoff

Paket `prospective-app-context-contract-correction-20261009`. Offline-Beitrag,
9. Oktober 2026, innerhalb Rootende `2026-10-09T07:31:47Z` / maximal 900 s.
Rootgrant-SHA256:
`549bd9fa83de6a702cde51daadbdacd880e6ecb489a0ac16df750f4c8de189e7`.
Null Actor-/Provider-/Helper-/Assessor-/Trial-/Test-/Build-/Installations-/CLIprobeaufrufe.
Die bisherige fallunabhängige Arbeit bleibt erhalten. Der frühere CLI-Launcher
ist kein App-Einstieg und wird hier weder aktiviert noch umgebaut.

## Zuständigkeit und benötigte Bindings

Oldschool übernimmt nach einem separaten konkreten Rootgrant ausschließlich die
Conventional-Ausführung. Scientist besitzt Aufgaben, Stationsfreigaben, Messung
und neutrale Bewertung; Root kann diesen öffentlichen Handoff weiterreichen.
Keine semantischen Folgehinweise oder Evaluatorbefunde an den Implementierer.

Der spätere öffentliche Ausführungsumschlag muss enthalten:

- Eigenes absolutes Projektrepository, eigener Stateordner und Ledgerpfad;
  `repo`, `preparedCommit` und aktueller `stationIndex` aus dem eigenen State.
- Eigenes aktuelles öffentliches Backlog, Projektregeln und freigegebene Station;
  die exakten unveränderten Texte `nest.PROMPT` / `nest.CONTINUE` samt Sourcepin.
- Neuer exakter Ownergrant mit `executionAuthorized`, `model=gpt-6-luna`,
  `reasoning=high`, `preparedCommit`, `repo`, `notAfterUtc` und festem Grant-SHA.
  Der heutige Nullquote-Vorbereitungsgrant ist dafür unzulässig.
- Eingefrorene öffentliche App-/Ledger-/Nest-Quellen und vorhandener Pythonpfad
  für die Ledgerfunktionen; Scientist benennt die vorhandenen konkreten Aufrufe
  für `nest.snapshot`, `nest.advance`, `nest.freeze` und den Abschlussmarker.
  Deren Implementierungen oder Signaturen wurden hier nicht zusätzlich gelesen.

Diese eigenen Case-/State-/Prompt-/Interpreterbindungen wurden noch nicht
übergeben. Das ist eine offene Übergabe, kein nachgewiesener Werkzeugfehler und
kein Anlass für Probe-Actors oder eine zusätzliche Capabilityphase.

## Vorhandene Toolschritte, ausschließlich bei neuem Actualgrant

1. Mit `functions.exec` / `tools.clock__curr_time` Frist prüfen. Über vorhandene
   Python-/Shellwerkzeuge `app_ledger.reserve(own_state_dir, fresh_grant_path,
   'implementation')` aufrufen. Der Ledger bindet Repo/Seed/Grant und liefert
   `reservation` / `deadlineUtc`. Keine Reservation als echten Modellstart zählen.
2. Danach direkt **außerhalb** `functions.exec` den vorhandenen Toolcall
   `collaboration.spawn_agent` verwenden: eigener Taskname, `fork_turns="none"`,
   `model="gpt-6-luna"`, `reasoning_effort="high"`; Message ausschließlich der
   feste Initialtext plus eigene Projekt-/Regel-/State-/Ledger-/Ressourcenbindungen.
   Es gibt keinen CWD-Parameter. Jede Datei-/Shelloperation des Actors richtet
   sich explizit auf das eigene Case-Repository.
3. Den tatsächlich zurückgegebenen eigenen Handle, kanonischen Tasknamen,
   Callresult und Zeitpunkt mit `app_ledger.record(..., 'started', handle=...,
   receipt=...)` erfassen. Fehlcalls als `call-failed` dokumentieren; Reservation,
   Toolrequest und tatsächlich beobachteter Modellturn bleiben getrennt.
4. Eigene Lifecycleereignisse beobachten, nach Wartephasen erneut Uhr prüfen;
   `collaboration.wait_agent` höchstens 60000 ms, ausschließlich in einem eigenen
   gemessenen Agentbaum. Keine globale Actorinventur und keine fremden Handles.
   Bei tatsächlichem Turnende `returned` samt Receipt erfassen. Scientist gibt
   ausschließlich nach festem Completionmarker Snapshot/NextStation frei.
5. Eine erlaubte feste Fortsetzung zunächst über `reserve(...,
   continuation_handle=own_returned_handle)` buchen, dann
   `collaboration.followup_task(target=own_returned_handle, message=nest.CONTINUE
   plus eigene aktuelle Stations-/Ressourcenbindungen)`. Kein neuer hilfreicher
   Prompt. `send_message` nur an bereits aktive eigene Handles für Stop/Ressourcen;
   nicht als ungebuchte Aktivierung eines idle Actors.
6. Vor Deadline nur eigene erfasste Handles mit `collaboration.interrupt_agent`
   unterbrechen; eigene bekannte Shell-/Prozessjobs abgleichen. `interrupted`
   beweist keine vollständige Prozessbeendigung. Bei unbekanntem Stop `unknown-stop`
   festhalten: keine weitere Aktivierung. Der gelesene Ledger lässt solche
   terminalen Einträge nicht nachträglich per `record` umetikettieren.

Normale Planung, Tests, Reparaturen, Subagenten und Review bleiben Sache des
Actors. Jeder Helper wird vor dem tatsächlichen fresh Spawn im selben Ledger
reserviert, mit aktiver eigener Parentreservation und Luna High gebunden sowie
nachher erfasst. Keine zusätzlichen ungemessenen Helfer. Ausführung ohne
Markitect/Government; keine Veröffentlichung außerhalb des isolierten Projekts.

## Endlichkeit, Kontext und Abschluss

Der vorhandene Ledger verlangt beim ersten Start volle 7200 s vor Ownerende:
5400 s Implementation, Stationenden bei 1350/2700/4050/5400 s; maximal zwei äußere
Entries pro Station, acht insgesamt; 72 gemeinsame Appreservations, vier aktive
Entries, Tiefe zwei. Bewertung maximal 1200 s bis Offset 6600, dann 600 s Sicherung.
Keine Nachfüllung. Genau ein frischer finaler Assessor gehört Scientist, nach
Freeze und geklärten eigenen Stops. Reservierungen sind kooperativ, keine harte
globale Ressourcen- oder OS-Sperre. Toolpolicyblocker beendet Implementation ohne
Fortsetzung/Transportwechsel; ungeklärte Cleanupzustände verhindern neue Actors.

Prospektiv zulässig: gemeinsame System-/Developer-/Tool-/Workspaceinstruktionen,
allgemeine App-Memory-/Einrichtungshinweise ohne fremde Falllösung/private
Bewertung und eigene legitime Stationshistory. Verboten: konkrete fremde
Kandidaten/Fallcode, private Holdouts/Bewertungsvektoren/Urteile oder semantische
Evaluatorhilfe sowie gezieltes Lesen fremder Case-/Chat-/Memoryquellen.
Erfasst werden Ursprungskategorien, CWD, fork-Parameter und tatsächliche eigene
Receipts, kein privater Text. Ein pauschaler Memory-/Historybegriff ist kein
Inhaltsnachweis. Bekannte verbotene Exposition ist terminal; ungeklärte
fallspezifische Exposition bleibt offen. Keine Sanitization-/OS-Isolationsbehauptung.

Der gelesene `app-entry.md` enthält noch unscharfe Formulierungen zu „memory“ /
„private inherited exposure“ und nennt Scientist als Stationsdispatcher. Für den
späteren Sourcefreeze müssen Scientists parallele Quellenpflege und dieser
Oldschool-Ausführungsowner mit der neueren Root-Kontextfassung übereinstimmen.
Hier wurde ausschließlich der eigene Handoff geschrieben.

Nach geklärtem Stop sichern die vorhandenen Snapshot-/Freeze-Schritte `main`,
unfertige Bytes und Stationsgrenzen. Merges/Checks brauchen reale SHA-/Logbelege;
Transporterfolg ist keine Bewertung. Usage, Servingmodell und Kosten bleiben
ohne echte Receipts unknown. Danach eigener Ledgerabschluss und private Sicherung
durch Scientist, ohne Credentials. Menschliche Abnahme bleibt getrennt.

Gelesene öffentliche Sourcepins (momentaner Stand, kein finales Freeze):

- `public/app-entry.md`: `168c974a9ece21834117dabc53dcff7a2c9ef868cf58b5b555a0ccda0a584859`
- `public/app_ledger.py`: `3b3633202954804e983e53e939d852f58ab45a9b3a1d1e232954c6e7e3c75d20`
- Root `docs/design/work-item-comparison-20261009.md`, verwendet wurde nur der
  öffentliche Rahmen/letzte Kontextabschnitt:
  `181337e3cedb7c7c68c3d2b8c542a319b199450939d2517ec35bbfba26214838`

Keine privaten Nachbarordner, Casekandidaten, Holdouts, Chat-/Memorydateien gelesen;
keine Ledgerfunktion oder Kollaborationsaktion ausgeführt. Praktische
Laufzeitfähigkeit und tatsächlicher Actor-Kontext bleiben in diesem Offlinepaket
ungeprüft. Diese Datei erteilt keine Ausführungsautorität.

# Assessment: Knowledge-Graph-Branch

Stand: 10. Oktober 2026. Gegenstand: Branch `codex/knowledge-graph` @ `ab405644` (Worktree `~/.codex/worktrees/product-knowledge-graph`). Dieses Dokument ist eine Einschätzung, keine Produktentscheidung und keine Abnahme.

## Kurzurteil

Die Idee passt zu Markitect, der Umfang ist aber deutlich größer als jeder bisher gezeigte Bedarf, und die Implementierung ist **nicht merge-reif**. Der reine Graph-Kern ist sauber. Die Host-Schicht, die den Graphen pro Manager filtert, hat dagegen einen funktionalen Kernfehler (die eigentlichen Modellbeziehungen fehlen in der Manager-Sicht) und zwei Privacy-Lecks. Empfehlung: als Experiment behalten und später nur einen kleinen Kern neu auf die aktuelle Produktlinie bringen, statt den Branch zu mergen.

## 1. Die Idee

Herkunft: Owner-Diskussion vom 8.10., festgehalten in `docs/implementation-plan.md` (PR #88) als offene Bewertungsfrage: Markitect als versionierter, typisierter Knowledge Graph für Intent, Anforderungen, Entscheidungen und ihre expliziten Beziehungen, mit Vergleich zu RDF/SHACL/OWL.

Kernidee: Das YAML-Modell ist schon ein typisierter Graph (Manager besitzt, Statement `uses`/`requires` Statement, Artifact `realizes` Statement, Check `checks` Artifact). Der KG macht ihn **abfragbar und erklärbar**: „Was hängt woran und warum?“, „Welche Entscheidung stützt diese Regel?“, „Welche Arbeit führte zu diesem Apply?“. YAML bleibt kanonisch, der Graph ist eine abgeleitete, read-only, scope-gefilterte Sicht. Keine Inferenz aus Prosa, kein Graph-DB, kein RDF.

Das adressierte Problem ist real und deckt sich mit der Nutzerbeschreibung vom 9.10. (Agents vergessen relevante Dateien, Konsistenz bricht mit wachsender Größe).

Wichtiger Kontext: Die vorgelagerte Recherche R02 (`docs/design/research/results-20261009.md`) empfahl ausdrücklich nur die **kleinste** Erweiterung: ein read-only `Explain` für ein Statement auf einem festen Snapshot. Query-Sprache, RDF und Rename-Identität sollten warten. Danach hat eine direkte Nutzeranweisung den vollen Umfang KG04–KG07 freigegeben.

## 2. Der Plan

Quelle: `docs/design/research/knowledge-graph-implementation-20261009.md`, Backlog auf dem Branch.

| Item | Inhalt |
|---|---|
| KG01 | Read-only Inventur und genau zwölf Referenzfragen mit erwarteten Antworten und Grenzen |
| KG02 | Reines, deterministisches Graph-Projektionsmodul mit Index |
| KG03 | Begrenzte Navigation mit Witness-Pfaden |
| KG04 | Decisions als First-Class-Einträge, neue Art `IdentityChange` (renamed/replaced/retired) |
| KG05 | Host: Scope/Privacy-Auswahl (S) und Adapter auf bestehende Run/Exploration/Brownfield-Records (E) |
| KG06 | CLI `markitect project knowledge` und eigener MCP-Server mit sechs Aktionen |
| KG07 | Zwölf-Fragen-Regression, Prozess-Journey, Main-Integration |
| RF01–RF06 | Separate RDF/SPARQL-Fit-Bewertung, nur Bewertung, keine Umsetzung |

## 3. Was umgesetzt ist

Der Branch liegt 146 Commits vor `origin/main`. Davon sind nur **11 Commits echte KG-Arbeit** (ab `1495e1be`); die übrigen 135 stammen aus der Management-Linie `codex/model-first-operations`.

KG-eigener Diff: ca. 4.400 Zeilen Produktions-Go, 3.750 Zeilen Tests, 700 Zeilen Doku und **ca. 43.400 Zeilen committete Evidence-Logs**.

| Item | Behaupteter Status | Befund |
|---|---|---|
| KG01–KG03 | fertig, reviewed | Stimmt. `internal/modules/projectknowledge` ist schlank, korrekt, deterministisch und gut getestet. |
| KG04 | fertig, reviewed | Grundsätzlich vernünftig, aber mit nicht-deterministischen Findings, einer fehlenden Owner-Prüfung und undokumentierten Verhaltensänderungen (siehe 4). |
| KG05-S | fertig, reviewed | **Kernfehler**: In der Manager-Sicht fehlen alle Listen-Beziehungen. Außerdem zwei Privacy-Lecks. |
| KG05-E | fertig, reviewed | Read-only hält, nichts Veraltetes wird als „current“ gemeldet. Ohne `--revision` ist aber jeder Run „stale“. |
| KG06 | fertig, reviewed | Funktioniert. `explain` und `relations` sind identischer Code, MCP-Fehler sind nichtssagend. |
| KG07 | „targeted validated“ | Kein voller Verify jemals grün (d360ec01 und ebe51470 rot, danach nur gezielte Reruns). |
| KG07-M | wartet auf Main | Offen. |
| RF | Bewertung fertig | Schlüssig: YAML bleibt, RDF höchstens später als optionaler Export. |

Selbst geprüft: `go vet` und alle KG-Pakettests laufen auf dem Branch-Snapshot grün. Die Tests decken die unten genannten Fehler aber nicht ab.

## 4. Stimmigkeit der Implementierung

Gefunden von vier unabhängigen Reviews. Die mit ✔ markierten habe ich selbst am unveränderten Branch-Code nachvollzogen.

### Hoch

1. ✔ **Manager-Sicht verliert die eigentlichen Modellbeziehungen.**
   - Ort: `internal/host/projectgraph/service.go:921-944`.
   - Ursache: Core benennt Listen-Kanten `uses[0]`, `requires[0]`, `realizes[0]` und `checks[0]`. `edgeAllowed` vergleicht aber mit den nackten Namen, sodass diese Kanten im `default`-Zweig verworfen werden.
   - Folge: Im Orders-View fehlt `cancel-order → requires → release-reservation`; Artifacts haben keine `realizes`-Kanten, Checks keine `checks`-Kanten. Die Kernfragen Q02, Q11 und Q12 sind in der empfohlenen Sicht nicht beantwortbar.
   - Die bestehenden Tests bleiben grün, weil ein Trace über falsch beschriftete Datei-Kanten trotzdem ankommt (siehe Mittel 4).
   - `KG07-question-outcomes.md` behauptet für Q11 das Gegenteil.
2. ✔ **Privacy-Leck: fremde frühere Identitäten.**
   - Ort: `service.go:405-419`. `historicalIdentityFacts` lässt jedes `public` IdentityChange durch, ohne zu prüfen, ob dessen Subjekt für diesen Manager sichtbar ist.
   - Folge: Private frühere Namen eines anderen Managers erscheinen im Graphen, im Context aber nicht.
3. **Privacy-Lücke im Modell.**
   - Ort: `internal/modules/projectmodel/analyze.go:352-366`. Das Feld `previous` eines IdentityChange hat keine Owner- oder Privacy-Prüfung.
   - Folge: Ein Manager kann ein privates Statement eines anderen Managers als seine frühere Identität deklarieren. Dessen eigener Eintrag scheitert danach als „previous-ambiguous“.

### Mittel

1. ✔ **Nicht-deterministische Diagnosen.**
   - Ort: `analyze.go:344` und `:422`. Die Zyklenerkennung für Decisions und IdentityChanges iteriert über eine Go-Map.
   - Folge: Das Finding hängt mal an A, mal an B, und der Report-Digest ändert sich zwischen Läufen. Das verletzt das Prinzip „deterministic diagnostics“.
2. ✔ **Default-Abfrage meldet jeden Run als „stale“.**
   - Ort: `internal/host/knowledgeevidence/read.go:274`. Die Working-Tree-Revision ist leer, deshalb gilt ohne `--revision <commit>` alles als veraltet. Die Doku erwähnt das nicht; ehrlich wäre „unknown/provisional“.
   - Zusätzlich tragen Verification-, Check- und Apply-Knoten `verified:true`/`passed:true` ohne eigenen Freshness-Marker.
3. **Coverage ist kaum aussagekräftig.**
   - Records sind immer `partial`, und das Ergebnis besteht nur aus fünf Wörtern, ohne zu sagen, *was* fehlt.
   - Die Manager-Coverage ignoriert Findings von Decision und IdentityChange (`service.go:860`, `impact.go:116`).
4. **Falsche Herleitungs-Labels.** Pfad-abgeleitete Verknüpfungen werden als „exact source-file match“ (`documents-statement`, `checks-file`) ausgegeben (`service.go:443-454`).
5. **Verhaltensänderungen für bestehende Projekte.**
   - Bisher gültige Decision-Modelle können jetzt scheitern, z. B. mit `decision.actor-out-of-scope`.
   - Jeder Report-Digest ändert sich (neue Felder), und das generierte Projektdokument ändert sich für jedes Projekt.
   - Impact einer reinen Decision-Änderung wurde von „ganzes Projekt, unknown“ auf gezieltes Routing **verengt**. Das ist vertretbar, aber undokumentiert und nicht bewusst abgenommen.
6. **Windows-abhängiges Verhalten.**
   - Ort: `read.go:877`. `isMissing` prüft Fehlertexte, und die Windows-Meldung „cannot find the file specified“ fehlt in der Liste.
   - Folge: Dieselbe Abfrage liefert unter Linux „unknown“, unter Windows einen Fehler.
7. **Read-only-Abfrage nimmt den Schreib-Lock.** `--session` greift auf den exklusiven Brownfield-Lock zu und legt `session.lock` an, falls er fehlt. Ein pollender MCP-Agent kann dadurch echte Brownfield-Schreibvorgänge mit „being updated“ scheitern lassen.
8. **MCP-Fehler verschleiern die Ursache.** Jeder Fehler wird zu „knowledge query failed …“. Für den Manager-Scope ist das gewollt (forbidden = absent), für den Project-Scope nur hinderlich.

### Niedrig (Auswahl)

- `explain` und `relations` sind derselbe Codepfad (`query.go:77`).
- `history` mischt Nicht-History-Kanten hinein.
- Der dokumentierte Zustand „historical“ wird nie erzeugt.
- MCP liefert -32700 statt -32600.
- Limits sind dupliziert.
- `projectapp` baut den Graphen zweimal.
- Ein Backtick wird in Code-Spans als `&#96;` ausgegeben.
- Projekte mit mehr als 5.000 Definitionen sind komplett ausgeschlossen.
- Etwas toter Code.

### Was gut ist

- Der reine Kern `projectknowledge` (ca. 670 Zeilen) ist korrekt: Limits werden nicht umgangen, Witness-Richtungen stimmen, er ist deterministisch und bei Fan-out und Zyklen getestet.
- Die Read-only-Garantie hält.
- Path Traversal ist blockiert.
- Veraltetes wird nie als „current“ ausgegeben.
- Klare Grenzen: keine Inferenz aus Prosa, kein Ersatz für Context, Impact oder Apply.

### Muster dahinter

Der Branch trägt sehr viel Prozess-Evidenz: SHA-Receipts, Reader-Reviews, Rohlogs fehlgeschlagener Verify-Läufe. Trotzdem fehlt der Kernfunktion in der Hauptsicht das Wesentliche. Die Tests sichern Abläufe, nicht die eigentlichen Antworten. Ein einziger Test „Graph-Sicht ⊆ Context, und die erwarteten Kanten der zwölf Fragen sind da“ hätte H1 und H2 gefunden.

## 5. Produkt-Fit

**Passt:**
- Abgeleitet, read-only, YAML bleibt kanonisch.
- Keine neue Abhängigkeit, deterministische Digests, scope-bewusst.

**Echter Mehrwert:**
- „Wer nutzt das und warum?“ mit Erklärungspfad, ohne dass eine Kandidaten-Revision nötig ist (Impact braucht eine).
- Decisions werden in Report, Context und Impact sichtbar; das schließt eine echte Lücke.
- Ein verbundener Blick über die getrennten Run-Ledger.

**Doppelt oder spekulativ:**
- Etwa 7 der 12 Fragen beantworten Analyze, Context, Impact und Coverage schon heute.
- `graph` überschneidet sich mit `project index`/`context`, `coverage` mit `project coverage`.
- Die Sichtbarkeitsregeln sind in `service.go` ein zweites Mal implementiert, sodass die Privacy-Regel zwei Owner hat. H1 und H2 sind genau die Folge dieser Divergenz.
- `IdentityChange` ist handgepflegte History im kanonischen Modell. Sie dupliziert Git und ändert am Impact nichts; reine Anzeige.
- Die Evidence-Adapter (925 Zeilen) hängen an Ledgern, die die laufende Vereinfachung gerade umbaut.
- Der eigene `knowledge-mcp`-Server existiert neben dem vorhandenen MCP-Adapter.

**Timing:** Die aktuelle Richtung (Native-Runtime-Vereinfachung vom 10.10.) sagt selbst: KG bleibt separat, bis der Kern-Loop funktioniert. Auf Markitects eigenem Repo lässt sich das Feature nicht einmal ausprobieren, weil `.markitect/project.yaml` fehlt; nur die Beispiele funktionieren.

**Bedienbarkeit:**
- Node-IDs sind JSON-Tupel, die man in der Shell quoten muss.
- Die Doku beginnt mit Scope-Regeln statt mit „wofür und wann“, zeigt keine Beispielausgabe und besteht zu einem Drittel aus Disclaimern.
- Das Beispiel muss erst in ein frisches Git-Repo kopiert werden.

**RDF-Entscheidung:** schlüssig. YAML bleibt; RDF kommt höchstens als optionaler abgeleiteter Export, wenn ein konkreter Interop-Bedarf entsteht. Der ausführliche RF01–RF06-Plan für ein „jetzt nicht“ ist mehr als nötig.

## 6. Branch-Hygiene und Integration

- **Fremde Änderungen im Branch:**
  - CI-Timeout von 60 auf 90 Minuten, `go test` von 30 auf 60 Minuten.
  - **Produktschema** `timeoutSeconds` maximal von 1800 auf 3600 s. Das ist eine nutzerseitige Vertragsänderung, nur damit langsame Tests passen; `verify.go:66` sagt weiterhin 1800.
  - In `docs/usage.md` ist bereits kaputte Zeichenkodierung doppelt kodiert (`0Ã¢â‚¬â€œ2`).
- **Rohlogs im Repo:** ca. 43k Zeilen, als owned artifacts in `markitect-artifacts.yaml` eingetragen. Die Work-Item-Dokumente lesen sich wie Koordinationsprotokolle (Overseer, Root, „Luna High Reader“, SHA-Ketten). Das gehört in CI-Artefakte oder die PR-Beschreibung, nicht ins Produkt-Repo.
- **Integration:**
  - Die Produktlinie (`codex/product-integration-20261009`) ist seit der KG-Basis 112 Commits und 786 Dateien weiter, und der gesamte Go-Baum liegt jetzt unter `src/`.
  - Ein Probe-Merge ergibt 24 Konflikte: 13 in Go-Dateien (davon 4 Umzüge), 5 in der Doku und 4 in generierten Dateien/Konfiguration.
  - Machbar, aber ein Merge würde den ganzen Ballast mitnehmen.

## 7. Empfehlung

**Nicht mergen. Als Experiment behalten. Wenn der Kern-Loop stabil ist, einen reduzierten Kern neu auf die `src/`-Struktur portieren:**

1. **Übernehmen:**
   - das reine Modul `projectknowledge`;
   - Decisions in Report, Context und Impact, mit deterministischer Zyklenerkennung, Tests für die neuen Fehlerfälle und bewusst dokumentierter Impact-Änderung;
   - **eine** read-only Abfrage (Nachbarn/Erklärung plus begrenzter Trace) über die bestehende CLI und den bestehenden MCP-Adapter, mit lesbaren IDs.
2. **Ein Owner für Sichtbarkeit:** Die Graph-Sicht wird aus der gefilterten Ausgabe von `projectmodel.Context` gebaut, nicht über eine parallele Filterschicht. Pflichttest: Graph-Sicht ⊆ Context, und die erwarteten Kanten der zwölf Fragen sind vorhanden.
3. **Zurückstellen:** `IdentityChange`, die Evidence-Adapter und die Coverage-Aktion, bis eine konkrete Nutzer- oder Agentenfrage sie braucht und die Ledger nach der Vereinfachung stabil sind.
4. **Aufräumen:**
   - keine Rohlogs und Prozessprotokolle im Repo;
   - Timeout- und Schemaänderungen zurücknehmen bzw. separat entscheiden;
   - den Kodierungsfehler beheben;
   - eine kurze Nutzerdoku, die mit einer echten Frage und ihrer Antwort beginnt.

## Methode

- Plan- und Work-Item-Dokumente gelesen.
- Fünf parallele Reviews:
  - Graph-Kern und Modell;
  - Host-Scope/Privacy;
  - Evidence plus CLI/MCP plus Bedienbarkeit am Beispiel;
  - Produkt-Fit;
  - Branch- und Merge-Lage.
- `go vet` und alle KG-Paket-Tests auf einem Export des Branches ausgeführt (grün).
- Die wichtigsten Befunde per Scratch-Test bzw. Codestelle selbst nachvollzogen.
- Kein voller Verify, keine Änderungen am Branch oder Worktree.

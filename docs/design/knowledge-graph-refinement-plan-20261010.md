# Knowledge Graph: Refinement-Plan

Stand: 10. Oktober 2026, Version 2 (nach drei unabhängigen Kritiken). Status: **Entwurf**, nicht freigegeben, kein Implementierungsauftrag.

Grundlagen:
- das [KG-Assessment](knowledge-graph-assessment-20261010.md);
- das [unabhängige Assessment](independent-assessment-20261010.md), Abschnitte 3, 7 und 11;
- die Produktlinie `origin/codex/product-integration-20261009` @ `669cecd2`;
- das Government-Designpaket unter `docs/design/government/`.

## Kurzfassung

- **Neue Rolle:** Der Knowledge Graph wird **die Erklärung und Klassifikation des Impact**, kein eigenes Abfrage-Produkt.
  - Heute meldet Impact im Shop-Beispiel bei einer einzigen Statement-Änderung 9 von 10 Statements und alle 6 Manager als betroffen. Es sagt aber nicht, *warum*, und nicht, was davon *geändert* und was nur *gelesen* werden muss.
  - Genau das liefert der Plan: jedes betroffene Element mit Grund, Begründungspfad und Klasse `change` oder `context`.
  - Die Gesamtmenge bleibt dabei unverändert oder wird breiter. Die konservative Invalidierung wird also nicht geschwächt; einzige Ausnahme ist D2, falls freigegeben.
- **Dieselbe Logik auf einem Snapshot:** Sie liefert `project affected`, die Betroffenheit *vor* einem Kandidaten. Das ist Government-Schritt B, ohne zweite Closure.
- **Messen zuerst:** Stufe 0 aus dem unabhängigen Assessment (`impact` gegen `rg`, eingefrorene Pflichtstellen, mittelgroßes Repo) entscheidet, ob und wie weit gebaut wird.
- **Zuerst zwei bestehende Impact-Fehler beheben.** Beide habe ich auf der Produktlinie reproduziert:
  - Die Coverage hängt von der Map-Reihenfolge ab; eine Datei fehlt manchmal.
  - Unprojizierte Änderungen (Decisions, `purpose`) gehen still verloren, sobald gleichzeitig etwas anderes geändert wird.
- **Decisions ohne Schemaänderung** in Report, Context und Impact.
- **Generischer Graph-Index mit `project explain`** nur, wenn der Fragenkatalog eine echte Lücke zeigt.
- **Ein schlanker Prozess:** Tests, die nicht leer bestehen können, und ein adversarialer Reviewer pro PR.
- **Umfang:** ca. 950 Zeilen Produktionscode, mit dem bedingten Graph-Teil ca. 1.550. Der alte Branch hatte 4.400.

## Was die Kritik am Entwurf v1 geändert hat

- **Impact-Kern:** Impact ist kein reiner Rückwärts-Kern. Es läuft vorwärts über `uses`/`requires` und dann zu allen Konsumenten, praktisch also über die ganze Zusammenhangskomponente.
  - Eine schmale Vorschau wäre deshalb nur durch Weglassen möglich.
  - Stattdessen wird klassifiziert, nicht verengt.
- **Messung:** Sie kam in v1 zuletzt und maß nur Selbstkonsistenz. Jetzt kommt sie zuerst und misst gegen eine unabhängige Pflichtstellen-Liste.
- **Decisions:**
  - Schemafelder (`public`, `supersedes`) hätten den Model-Digest jedes Projekts geändert und akzeptierte Historien als veraltet markiert.
  - `supersedes` dupliziert Git; es entfällt.
  - Sichtbarkeit für Fremde wird zurückgestellt.
- **Struktur:**
  - Der Katalog liegt in v2 auf Host-Ebene, weil ein Modul keine Projekte laden darf.
  - Witness-Pfade entstehen im Impact-Kern selbst; das vermeidet einen Importzyklus.
- **Government:** Die Sichtbarkeits- und Mandatshaken aus v1 waren teils überflüssig (Mandate prüft Government bereits gegen die Vorversion) oder verfrüht (Ressorts vor dem Kernthesen-Test).
- **Kurze IDs:** Zwei ID-Formen parallel widersprechen der Entscheidung vom 9.10. gegen Kompatibilitätspfade. Das ist eine eigene, produktweite Entscheidung.
- **Prozess:** Spec-Autor, Implementer, drei Reviewer und Verifier pro Item waren unverhältnismäßig. Es bleiben Tests-first und ein adversarialer Reviewer.

## 1. Ausgangslage

- **Alter Branch:** `codex/knowledge-graph` @ `ab405644` wird nicht gemerged (siehe Assessment). Als Vorlage taugen der Index/Walk aus `projectknowledge` und Teile der Decision-Validierung.
- **Impact auf der Produktlinie:**
  - Es ist bewusst konservativ und sehr breit: eine Änderung an `cancel-order` ergibt 9/10 Statements, 6/6 Manager und fast alle Dateien.
  - Inhaltlich wird nur `ChangeImpact` ausgegeben, ohne Begründung. Das ist Risiko 2 im unabhängigen Assessment: „stille Lücken oder zu breite Mengen“.
- **Bestehende Fehler:**
  - Bei mehreren Ausgangspunkten ist die Coverage reihenfolgeabhängig (`impact.go:280-312`, zwei verschiedene Digests in 200 Läufen).
  - Der Unknown-Gesamt-Review greift nur, wenn *keine* projizierte Definition geändert wurde (`impact.go:259`). Eine unprojizierte Änderung (Decision, `purpose` eines Statements, Artifacts oder Checks) zusammen mit einer anderen Änderung bleibt deshalb ungeroutet.
  - Reproduziert: Die `purpose`-Änderung am öffentlichen Vertrag `release-reservation` plus eine unrelated Änderung ergibt kein Unknown, und der Vertrag wird nicht geroutet.
- **Startbedingung:** Die Koordination hält den KG nach dem Main-Abschluss an, bis eine neue direkte Anweisung kommt (`469f7d9b`). Der Backlog der Produktlinie führt noch `KG01-KG03 … independently_authorized_post_main_integration` (`docs/work-items/product-readiness/follow-up-backlog.yaml`). Dieser Eintrag wird durch diesen Plan ersetzt.

## 2. Zielbild und Einordnung

**Rolle im Gesamtziel:** Im langfristigen Ziel („Man pflegt eine maßgebliche Stelle, das kanonische Modell, und lässt die Änderung anwenden“) ist die explizite Zuordnung das Mittel gegen vergessene Stellen (Konzepte C04/C05). Sie hilft nur, wenn die berechnete Menge vollständig **und** handhabbar ist.
- Für die Vollständigkeit sorgt die konservative Invalidierung.
- Für die Handhabbarkeit fehlt heute die Trennung von „ändern“ und „lesen“ samt Begründung. Diese Trennung ist der eigentliche Beitrag des Graphen.

Die Trennung setzt die Empfehlung 7 des unabhängigen Assessments um: „Betroffene Menge und lesende Kontextmenge getrennt ausweisen“, ohne die AGENTS.md-Regel zu brechen.

Merkmalsfilter aus der Vision:

| | Erklärter, klassifizierter Impact / `project affected` |
|---|---|
| Entfallende menschliche Tätigkeit | Aus einer breiten Impact-Liste von Hand herausfinden, was wirklich geändert werden muss und warum ein Element überhaupt dabei ist; vor einem Auftrag die Betroffenheit abschätzen |
| Kanonischer Owner | `projectmodel`: ein Closure-Kern für Impact und Vorschau |
| Freiheit des Agents | Liest nur. Die Klasse ist eine Priorisierung, keine Erlaubnis, `context`-Elemente zu ignorieren |
| Nachweisgrenze | Deklarierte Beziehungen. Was nicht modelliert ist, bleibt unsichtbar (Stufe 0 misst genau das). Impact bleibt das Gate |
| Neuer Pflegeaufwand | Keiner im Modell. Code ca. 400 Zeilen im Kern |

**Government:**
- Schritt B („Betroffene Verantwortung und Pflichten bestimmen“) ist `affected` plus die Vorversions-Mandats- und Writer-Prüfung, die Government schon hat (`BuildPlan.authorize`). Es gibt keine eigene Closure.
- Der Kern übernimmt deshalb Governments Regel, über gemeinsam realisierte Dateien zu verbreitern (Shared-File).
- Graph-Ergebnisse erzeugen nie Autorität, Mandat oder Zustimmung.

## 3. Leitplanken

1. YAML bleibt die einzige kanonische Quelle. Alles Weitere ist abgeleitet, wird nicht persistiert und ist read-only.
2. **Ein Owner pro Regel:**
   - Sichtbarkeit gehört `projectmodel.Context`.
   - Betroffenheits-Semantik gehört einem Closure-Kern in `projectmodel`, für Impact und Vorschau gemeinsam.
   - Identität und Kanten gehören Core.
3. **Keine Verengung:** Die Vereinigung aller als betroffen gemeldeten Elemente ist nach jeder Änderung gleich oder größer als vorher. Klassifikation ist keine Verengung; Impact bleibt das Gate. Einzige Ausnahme ist D2, falls freigegeben.
4. **Determinismus:** Ausgaben und Digests sind von Eingabe- und Map-Reihenfolge unabhängig, geprüft per Permutations-Property-Test.
5. Unbekanntes und fehlgeschlagene Snapshots verbreitern vollständig. Leere Ergebnisse sind kein Nachweis.
6. Keine Inferenz aus Prosa oder Code.
7. **CLI zuerst:** MCP nur über `mcp.Register` im bestehenden Adapter.
8. Abnahme über Antworten (Katalog, Property-Tests, Messung), nicht über Prozessbelege. Evidence gehört in PR-Beschreibung und CI-Artefakte, nie als Roh-Log ins Repo.
9. Keine fremden Änderungen in einem Item (Timeouts, Schema-Limits usw.).
10. Das Modul-Import-Gate gilt auch für Tests.

## 4. Umfang

| Entscheidung | Inhalt | Begründung |
|---|---|---|
| **USE** | Stufe-0-Messung und ausführbarer Fragenkatalog | Bedarf und Abnahme werden messbar, bevor gebaut wird |
| **USE** | Korrektur der zwei Impact-Fehler (Reihenfolge, unprojizierte Änderungen) | Verbreitert nur; Voraussetzung für einen gemeinsamen Kern |
| **USE** | Decisions in Report, Context (nur eigene) und Impact, ohne Schemaänderung | Echte Lücke; kein Digest-Bruch für Projekte ohne Decisions |
| **USE** | Erklärter und klassifizierter Impact plus `project affected` aus einem Kern | Größter Nutzen, adressiert Risiko 2, Government-Schritt B |
| **USE** | Einbindung in Planung und Briefing | Agents sollen die Information bekommen, ohne an einen Befehl denken zu müssen |
| **BEDINGT** | Graph-Index und `project explain` | Nur wenn der Katalog Fragen zeigt, die `index`, `context` und erklärter Impact nicht in höchstens zwei Aufrufen beantworten |
| **DEFER** | Decision-Felder `public` und `supersedes`, fremde Sichtbarkeit von Decisions | Brauchen eine Schema-Migration; Git hält die Historie |
| **DEFER** | Manager-Scope für `affected` und Grenzmarker | Ancestor-Routing macht den Marker trivial wahr; gehört erst zur Context-Erweiterung |
| **DEFER** | IdentityChange, Record-Joins, Coverage- und History-Aktion, RDF | Kein demonstrierter Bedarf; Ledger im Umbau |
| **DEFER** | Kurze IDs | Produktweite UX-Entscheidung, nur als einheitliche Umstellung, nie zwei Formen parallel |
| **DEFER** | Sichten für Ressorts oder Integratoren | Erst nach dem Kernthesen-Test (Empfehlung 1 des unabhängigen Assessments) |
| **REJECT** | Zweiter MCP-Server, Host-Filterschicht, Roh-Logs und Receipts im Repo | Doppelte Owner, Ursache der alten Lecks, Evidence-Regel |
| **REJECT** | Regel `decision.actor-out-of-scope`, `authority`-Feld an Decision | `actor` ist Provenienz. Mandate prüft Government gegen die Vorversion; Briefings tragen bereits `Authority` |

## 5. Arbeitspakete

| Item | Ergebnis | Abhängig von | Produktion / Tests |
|---|---|---|---|
| K0 | Fragenkatalog und Stufe-0-Messung mit Go/No-Go | Freigabe (D1), Testrepo (D4) | – / ~400 + Messskript |
| K1 | Impact deterministisch, unprojizierte Änderungen nicht mehr still verloren, Modellgenerator | D1 | ~120 / ~350 |
| K2 | Decisions in Report, Context und Impact | K1, D2 | ~220 / ~250 |
| K3 | Gemeinsamer Closure-Kern: erklärter, klassifizierter Impact plus `project affected`, Wiederholungsmessung | K1, K2, D3, D5 | ~450 / ~600 |
| K4 | Planung und Briefing zeigen die `change`-Menge mit Gründen | K3 | ~150 / ~200 |
| K5 | *Bedingt:* Graph-Index und `project explain` | Lücke nach K3 | ~600 / ~550 |
| K6 | Doku, Beispiel, Backlog, Abschluss | K3/K4 (K5) | Doku |

**Stoppregeln:**
- **Nach K0:** Liegt der Recall von `impact` unter `rg`, ist nicht die Erklärung der Engpass, sondern Modell, Zuordnung oder Granularität. K3 bis K5 pausieren dann, und das Ergebnis geht an den Owner.
- **Nach K3:** Die Wiederholungsmessung gehört zur K3-Abnahme. Bringt die Klassifikation die `change`-Breite nicht unter das Dreifache der Pflichtstellen, entfällt K4/K5, und es wird berichtet.
- **K5** startet nur, wenn nach K3 Katalogzeilen offen sind, die `index`, `context` und der erklärte Impact nicht in höchstens zwei Aufrufen beantworten.

**Definition of Done des Plans:**
- Alle nicht zurückgestellten Katalogzeilen bestehen.
- Die Wiederholung der Stufe-0-Messung zeigt einen `change`-Recall mindestens auf `rg`-Niveau (Vorschlag ≥ 95 %) bei einer `change`-Breite von höchstens dem Dreifachen der Pflichtstellen. Die Schwellen sind Vorschläge des unabhängigen Assessments und werden vorab eingefroren.
- Impact besteht den Permutations-Test.
- Die Vereinigung der gemeldeten Mengen ist nie enger als vorher; einzige Ausnahme ist D2, falls freigegeben.

### K0: Fragenkatalog und Stufe-0-Messung

**Katalog:**
- **Ort:** `src/internal/host/projectapp/knowledge_questions_test.go`. Er nutzt eine temporäre Kopie von `examples/project-world` mit `git init` (Muster: `projectbriefing/briefing_test.go`), weil ein Modul keine Projekte laden darf.
- **Zeilen:** die zwölf KG01-Fragen. Jede Zeile enthält Scope, Startpunkt, mindestens eine positive Erwartung (exakte Kanten- oder Elementmengen mit Relationsname und Herleitung), verbotene Strings und die heutige Antwortquelle.
- **Lücken-Zeilen** laufen als *erwarteter Fehlschlag*:
  - Eine Lücke, die unerwartet besteht, lässt die Suite fehlschlagen.
  - Die Zahl der Lücken ist festgeschrieben und darf nur sinken.
- **Unabhängige Kantenzählung:** Erwartete Kantenzahlen pro Relation zählt ein simpler, unabhängiger Zähler über die YAML-Listen, nicht der Projektionscode.
- Die zurückgestellten Fragen (Q05, Q07, Q09, Q10) halten die heutige ehrliche Antwort fest.

**Stufe 0** (aus dem unabhängigen Assessment, Abschnitt 11):
- **Basis:** gemessen auf dem fixen Main-SHA vor K1, weil K0 und K1 parallel laufen.
- **Testrepo:**
  - ein mittelgroßes Repo (D4) mit zehn vorab definierten Änderungen (Umbenennung, Regeländerung, Vertragsänderung);
  - das Modell darf kein `Report.Unknown` haben (alle Dateien zugeordnet). Sonst verbreitert jedes Impact vollständig, und die Messung sagt nichts.
- **Pflichtstellen:** Ein **unabhängiger** Autor legt Pflicht- und Nicht-ändern-Stellen fest und friert sie per Hash ein.
- **Vergleich:** `markitect project impact` gegen eine Identifier-Suche mit `rg`.
- **Metriken:** Recall, Breite relativ zu den Pflichtstellen und Modellierungsaufwand.
- **Ergebnis:** eine kurze Notiz unter `docs/validation/`; Rohdaten als CI- oder Workflow-Artefakt.

**Abnahme:**
- Katalog grün mit der festgeschriebenen Lückenzahl.
- Ein Selbsttest verfälscht den eigenen Report-zu-Kanten-Adapter des Katalogs testweise (z. B. Listen-Kanten weglassen); dann muss der Katalog fehlschlagen.
- Die Messnotiz enthält das Go/No-Go nach den Stoppregeln.

### K1: Impact korrekt und deterministisch

**Wo:** `src/internal/modules/projectmodel/impact.go`, neu der Testgenerator `src/internal/modules/projectmodel/internal/modelgen/`.

**Was:**
- **Coverage reihenfolgeunabhängig:** Coverage wird für jedes Closure-Mitglied ergänzt, nicht nur beim ersten Fund als Konsument. Die Queue arbeitet sortiert.
- **Unprojizierte Deltas** (Decision-Definitionen, `purpose` von Statements, Artifacts und Checks) werden auch neben projizierten Änderungen erkannt:
  - Analyze berechnet einen Residual-Digest über genau diese Teile. Das Feld ist `json:"-"` und nicht im Report-Digest; alle Impact-Aufrufer analysieren in-process.
  - Weicht der Residual-Digest ab, setzt Impact Unknown, unabhängig von `len(changed)`. Das ist breiter als heute.
- **Modellgenerator** (nur für Tests, importiert nur `core` und liefert Definitionen und Inventar, kein Importzyklus):
  - fester Seed, `math/rand/v2`;
  - höchstens 4 Manager, Tiefe 3, ca. 25 Definitionen;
  - bewusst private Querverweise und Decisions.
- **Testlauf:** über `testing.F` mit einem festen Korpus von ca. 200 Seeds (deterministisch, Sekunden). Lokales `-fuzz` verkleinert Fehlschläge.

**Abnahme:**
- **Permutations-Property:** Gleiche Eingabe in beliebiger Reihenfolge ergibt den gleichen `ChangeImpact.Digest`.
- **Keine Verengung:** Jede Ergebnismenge ist ⊇ der alten Implementierung. Die alte Implementierung bleibt dafür im PR als Referenzkopie in einer `_test.go`-Datei und wird danach gelöscht.
- Beide reproduzierten Fehler sind als Regressionen enthalten:
  - zwei Seeds mit fehlender Datei;
  - `purpose` bzw. Decision plus eine unrelated Änderung.
- `ChangeImpact.Digest` ändert sich, wenn sich Mengen ändern. Die Konsumenten werden im PR benannt:
  - `projectrun/plan.go` prüft den gespeicherten `ChangeImpactDigest`, laufende `--since`-Pläne werden veraltet;
  - `projectwork/mutation.go` (EditPlan-Digest);
  - `projectbriefing`.

Dieses Item lohnt sich unabhängig vom KG und kann als Erstes nach Main.

### K2: Decisions in Report, Context und Impact

**Wo:** `projectmodel/{types.go, analyze.go, impact.go}`, `src/internal/host/projectwork/document.go`.

**Was:**
- **Typen:** `Decision{ID, Name, Namespace, Owner, Subject, Actor, Decision, Reason, Source}`. Owner ist der nächste Namespace-Manager, `Source` nur der Pfad.
  - `Report.Decisions` ist im Digest-Struct `omitempty`, damit sich Reports ohne Decisions nicht ändern.
  - `ManagerContext.Decisions` enthält nur eigene Decisions.
  - Für den Projekt-Scope gilt die volle Liste.
- **Schema:** keine Änderung. Die bestehende Decision-Kind (subject, decision, reason, actor) reicht.
- **Validierung:**
  - Subjekt nicht fremd-privat;
  - Owner vorhanden;
  - keine Actor-Scope-Regel;
  - mehrere Decisions je Subjekt sind erlaubt.
- `findingTouchesManager` berücksichtigt Decisions.
- **Impact:** Ein projiziertes Decision-Delta ist eine deklarierte Änderung seines Subjekts und wird mit Subjekt und Decision-Owner als Ausgangspunkten geroutet.
  - Ob das den bisherigen Unknown-Gesamt-Review für *projizierte* Decision-Deltas ersetzt, entscheidet D2.
  - Bei D2 = Ja fallen Decisions aus dem K1-Residual-Digest.
  - Alle übrigen unprojizierten Deltas behalten Unknown.
- **Dokument:** Der Abschnitt erscheint nur, wenn Decisions existieren.
- **Geteilte Verträge:** `ManagerContext` fließt in Run, Review und Full Verify (`projectrun/run.go`, `review.go`, `full_verify.go`). Wegen `omitempty` ändert sich nichts für Projekte ohne Decisions. Bei Projekten mit Decisions ändern sich InputDigests; das wird im PR benannt.

**Abnahme:**
- Ein Test pro Finding-Code.
- Eigene Decision-Fehler erscheinen im eigenen Context.
- Permutations-Property mit Decisions im Generator.
- Ein Digest-Test: Reports ohne Decisions bleiben byte-gleich.
- Die Katalogzeile Q04 wird grün.

### K3: Gemeinsamer Closure-Kern, erklärter und klassifizierter Impact, `project affected`

**Wo:**
- `projectmodel/impact.go` (Kern extrahieren), neu `projectmodel/affected.go`;
- `projectapp`, `projectcli`, `projectcli/mcp.go`;
- `projectonboarding/render.go` (Tool-Liste).

**Geteilte Verträge:** Neue Mengen (Shared-File) und `explanations` ändern `ChangeImpact.Digest`. Die Konsumenten werden wie in K1 im PR benannt: `projectrun/plan.go`, `projectwork/mutation.go`, `projectbriefing`.

**Was:**

1. **Kern:**
   - Signatur: `closure(graph(reports...), seeds{statements, managers, files, checks})`.
   - Er bildet die heutige Impact-Logik vollständig ab: Vereinigung beider Reports, Seeds aus Datei-Diffs, Manager-Seeds, Ancestor-Routing.
   - Eine BFS-Elternkarte liefert für jedes Element einen Witness-Pfad und einen Grund-Code: `changed`, `realizes`, `checks`, `uses-consumer`, `requires-consumer`, `uses-dependency`, `requires-dependency`, `shared-file`, `ancestor-routing`, `widened`.
2. **Shared-File-Regel:** Elemente, die dieselbe Datei realisieren wie ein betroffenes Element, werden mit betroffen. Das verbreitert Impact und wird im PR als bewusste Verbreiterung ausgewiesen (D5).
3. **Klassifikation (D3):**
   - **`change`:**
     - alle Seeds: geänderte Definitionen; bei einer Manager-Änderung dessen Elemente; Datei-, `--path`- und `--id`-Seeds samt den Statements, Artifacts und Checks ihrer FileEntries;
     - deren Realisierungen, Dateien und Checks;
     - direkte `uses`- und `requires`-Konsumenten geänderter Statements (öffentlich oder privat) samt Realisierungen;
     - Shared-File-Mitrealisierungen.
     - `widened` gehört immer zu `change`.
   - **`context`:** alles Übrige der Hülle.
   - Die Vereinigung ist exakt die bisherige Impact-Menge (plus Shared-File).
4. **`project impact`:** bekommt additiv `explanations[]` mit `{element, class, reason, witness}` und Breitenzahlen pro Klasse.
5. **`project affected --repo . --project (--id ID … | --path P …)`:**
   - Derselbe Kern auf **einem** Snapshot, nur im Projekt-Scope (wie Impact).
   - Ist ein Snapshot nicht erfolgreich analysiert oder die Inventur unbekannt, wird vollständig verbreitert.
   - Pfade werden normalisiert: Backslashes werden umgewandelt, `..` und absolute Pfade abgelehnt.
   - **Pfade, die noch nicht in `Report.Files` stehen** (neue Dateien):
     - Owner per Ownership-Selektor, Artifacts mit passendem Pfad-Selektor;
     - ohne Owner wird vollständig verbreitert;
     - ein `/`-Suffix seedet alle Einträge darunter.
6. **MCP:**
   - `project_affected` über `mcp.Register`, mit `PublicErrorMapper` für „not found“.
   - Die Erklärungen erscheinen im bestehenden Tool `project_impact`.
7. **Grenzen:**
   - Ergebnislimit mit `complete:false`. Es gilt nur für `explanations[]` und die `affected`-Ausgabe, nie für Impact-Mengen.
   - Ein Benchmark auf einem generierten Modell mit 5.000 Definitionen.

**Abnahme:**
- **Differenz:** Die Vereinigung der neuen Impact-Mengen ist ⊇ der K2-Implementierung über generierte Modelle mit permutierten Seeds.
- **Mutationsoperatoren:**
  - Text ändern;
  - Referenz je Relation hinzufügen oder entfernen;
  - `public`/`required` umschalten;
  - Namespace oder Quelle verschieben;
  - Löschen;
  - Decision ändern;
  - Datei ändern oder löschen;
  - Inventur unbekannt.
  - Eine Mindestzahl von Mutationen ohne Unknown ist festgeschrieben.
- **Vorschau-Property:**
  - S ist die Menge der geänderten Definitionen plus der geänderten Dateipfade.
  - Ist `Impact.Unknown` leer, gilt `routed ⊆ Affected(base,S) ∪ Affected(C,S)`.
  - Sonst muss `affected` für den betroffenen Snapshot vollständig verbreitern.
  - Der Fall `modelUnprojected` ist ausgenommen und dokumentiert.
- **Jeder Witness-Pfad** besteht nur aus echten Modellkanten oder Dateizuordnungen mit korrekter Herleitung.
- **CI:** Für jedes Beispiel werden die Breiten `change`/`context` berichtet (Empfehlung 7 des unabhängigen Assessments).
- **Weitere Tests:**
  - Windows-Pfadtests;
  - Onboarding-Tool-Listen-Test (`newnative_workflow_test.go`);
  - Test des geschlossenen Eingabeschemas für das neue MCP-Tool;
  - Katalogzeilen Q02, Q03, Q06 und Q12 grün.
- **Wiederholungsmessung:** Die Stufe-0-Messung wird mit der `change`-Klasse wiederholt, gleiche eingefrorene Pflichtstellen. Das Ergebnis entscheidet die Stoppregel nach K3 und die Definition of Done.

### K4: Einbindung in Planung und Briefing

**Wo:** `projectrun/plan.go` (Plan-Auswahl), `projectbriefing` (Modell-Briefing). Das sind geteilte Verträge; die Abstimmung mit dem Integration-Owner ist vorab nötig.

**Was:**
- Ein Plan bzw. Briefing für einen Auftrag enthält für jeden beteiligten Manager dessen sichtbare `change`-Elemente mit Grund und Witness.
- Die Manager-Auswahl (`changeImpact.Managers`) bleibt unverändert; ergänzt wird nur bei `--since`. Damit wird keine Auswahl über die Klassifikation verengt.
- Witness-Pfade enden an unsichtbaren fremden Knoten; es werden keine fremden IDs gezeigt.
- Die `context`-Elemente bleiben wie bisher im Context.

**Abnahme:**
- Der delegierte Manager-Input enthält die `change`-Elemente mit Gründen.
- Ein Canary-Test zeigt, dass private Tokens anderer Manager nie in Input, Ausgabe oder Fehlermeldungen auftauchen.
- Die InputDigest-Änderung ist im PR benannt.
- Der native Smoke-Lauf der Produktlinie bleibt grün.

### K5 (bedingt): Graph-Index und `project explain`

**Nur wenn nach K3** Katalogzeilen offen sind (Kandidaten: Q01, Q08, Q11), die `index`, `context` und der erklärte Impact nicht in höchstens zwei Aufrufen beantworten.

**Wo:** `src/internal/modules/projectmodel/knowledge/` (darf `projectmodel` importieren), `projectapp`, `projectcli`, MCP-Registrierung.

**Was:**
- **Index und Walk** werden aus `ab405644` portiert, mit diesen Fixes:
  - `New(nodes, edges)`;
  - keine 5.000-Definitionen-Grenze;
  - `omitzero`;
  - Limits an einer Stelle.
- **Manager-Sicht:** Sie wird **ausschließlich aus `ManagerContext`** gebaut, mit vollen Knoten und Referenzknoten.
  - Je Listeneintrag entsteht eine Kante mit normalisiertem Relationsnamen (`uses[0]` → `uses`).
  - Datei-Zuordnung kommt über `FileEntry.Owner`, gefiltert auf im Context sichtbare IDs.
  - Der Digest ist ein Digest der Sicht, nicht des Reports.
- **Projekt-Sicht:** aus dem Report.
- **`project explain`:** `--id ID [--depth 1..6] [--direction out|in|both]`, mit JSON-Tupel-IDs wie alle anderen Befehle (bis D6).

**Abnahme:**
- **Canary-Tokens:** Eindeutige Tokens in privaten Feldern tauchen nie in der Sicht eines anderen Managers, in `explain` oder in Fehlern auf.
- **Kantenbijektion:** Jede Context-Referenz ergibt eine Kante, und jede Kante entspricht einer Context-Referenz.
- **Selbsttests:** Wird der Normalisierer zur Identität oder der Sichtfilter abgeschaltet, muss die Suite fehlschlagen.
- **CLI gegen MCP:** Gleichheit, verglichen mit der sanitisierten CLI-Ausgabe.
- **„Verboten = fehlt“:** Im Manager-Scope sind beide Fälle identisch.
- Die Katalog-Lückenzeilen werden grün.

### K6: Abschluss

- **Doku:** eine kurze Seite. Sie beginnt mit einer echten Frage: „Was muss ich ändern, wenn sich release-reservation ändert, und warum?“ Danach kommen Befehl, gekürzte Antwort und fünf Zeilen Grenzen.
- **Weitere Doku:** Ergänzungen in `docs/project-operations.md` und `docs/project-workflow.md`. Die Onboarding-Hinweise kommen aus ihrer Go-Quelle (`projectonboarding/render.go`) und werden neu generiert.
- **Beispiel:** eine echte Decision in `examples/project-world`.
- **Backlog:**
  - `KG01-KG03` in `follow-up-backlog.yaml` wird durch dieses Paket ersetzt, ohne zweiten Roadmap-Eintrag.
  - Der alte Branch wird als `archive/knowledge-graph-ab405644` getaggt.
  - RDF wird als einzeiliges DEFER festgehalten.
- **Messung:** Das Ergebnis der Wiederholungsmessung aus K3 wird in `docs/validation/` dokumentiert. Es ist die Definition of Done.
- **Dogfooding:** Markitect selbst hat kein `.markitect/project.yaml`. Die Validierung erfolgt deshalb am Stufe-0-Repo und an den Beispielen; das wird ausdrücklich festgehalten.

## 6. Government-Anschluss (später)

- **Schritt B** ist `project affected` plus die vorhandene Mandats- und Writer-Prüfung gegen die Vorversion. Es gibt keine zweite Closure; die Shared-File-Regel liegt schon im Kern.
- **Weitere Sichten** (quer über Bereiche prüfende Rollen, Teilbaum-Integratoren) werden nur in einer Sichtfunktion von `projectmodel` ergänzt, nie im Graphen. Das geschieht erst nach dem Kernthesen-Test.
- **Offene Owner-Frage:** Darf eine quer prüfende Rolle private Statements in ihrem Auftrag lesen?
- **Begriffe:** Die Modell-`Decision` ist weder die `DecisionReference` des Briefings noch ein Abstimmungs- bzw. Acceptance-Record; es sind getrennte Typen. Ergebnisse erzeugen nie Zustimmung.
- **IdentityChange** nur, falls eingefrorene Identitäten Kontinuität brauchen. Die korrekte Semantik ist im Assessment festgehalten: `previous` gehört dem Owner, beide Identitäten gelten als betroffen, nichts wird automatisch übertragen.
- **Record-Joins** (Decision → Acceptance, offene Eskalationen blockieren Aufträge) erst, wenn die Ledger stabil sind. Dann gelten:
  - drei Binding-Zustände (Working Tree = unknown);
  - Freshness an jedem Knoten;
  - typisierte Not-found-Fehler;
  - keine Writer-Locks beim Lesen.

## 7. Wie ich es konkret umsetzen würde

1. **Basis:**
   - Pro Item ein kurzlebiger Branch vom akzeptierten Main; gestapelte PRs sind erlaubt.
   - **Baseline-Regel:** Fehlschläge, die schon auf Main bestehen und im PR aufgelistet sind, blockieren ein Item nicht.
2. **Tests zuerst:**
   - Jedes Item beginnt mit einem Commit, dessen neue Tests auf der Basis fehlschlagen bzw. als erwartete Lücke laufen.
   - Für K0 schreibt ein **unabhängiger** Subagent Katalogerwartungen und Pflichtstellen, ohne Implementierung. Ich prüfe sie gegen den Modellquelltext.
3. **Umsetzung:** ein Implementer-Subagent (Opus) im eigenen Worktree, beschränkt auf die im Item genannten Pfade.
4. **Ein adversarialer Reviewer pro PR** mit festem Auftrag:
   - K1, K2, K3: keine Verengung, Determinismus.
   - K3, K4, K5: Sichtbarkeit und Canary-Tokens.

   Befunde werden nur bei Streit separat verifiziert.
5. **Integration durch mich:**
   - fokussierte Tests, `go vet`, Architektur-Gate, Artifact-Check;
   - Neugenerierung, wo kanonische Quellen berührt sind;
   - Katalog.
   - Dann der PR mit kompakter Evidence (SHA, Befehle, Ergebnisse, Grenzen) und CI auf Linux und Windows.

   Merge nur durch Owner-Entscheidung, kein Release.
6. **Was ich nicht tue:**
   - Roh-Logs oder Receipts committen;
   - unveränderte Full-Suites wiederholen;
   - Timeouts oder Schema-Limits anheben;
   - Government-Begriffe vorwegnehmen.

**Reihenfolge:** K0 und K1 parallel → K2 → K3 → K4 → K5 (bedingt) → K6. Nach jedem Item ist Main in einem brauchbaren Zustand, und an jeder Stoppregel kann sauber abgebrochen werden.

## 8. Entscheidungen, die von dir gebraucht werden

| Nr | Frage | Mein Vorschlag |
|---|---|---|
| D1 | Start direkt nach Main-Abschluss mit K0 und K1 (K1 ist ein Bugfix, der sich unabhängig vom KG lohnt)? | Ja |
| D2 | Ersetzt ein *projiziertes* Decision-Delta (geroutet über sein Subjekt) den bisherigen Unknown-Gesamt-Review? | Ja. Nach der Projektion ist die Änderung deklariert; unprojizierte Deltas behalten Unknown. Das ist eine bewusste Owner-Entscheidung gegen die Invalidierungsregel. |
| D3 | Klassifikationsregel `change` gegen `context` wie in K3 (Vereinigung unverändert, Impact bleibt Gate)? | Ja; nach der K6-Messung nachschärfen |
| D4 | Testrepo für Stufe 0: eingefrorenes MyMeetings-Subset (646 Dateien), ein eigenes Projekt oder Markitect selbst (bräuchte erst ein Modell)? | MyMeetings-Subset |
| D5 | Shared-File-Verbreiterung auch in Impact (breiter als heute)? | Ja, wegen Government-Schritt B und weil es konservativ ist |
| D6 | Kurze IDs `Kind:namespace/name` produktweit statt JSON-Tupel, als einheitliche Umstellung ohne Parallelform? | Später separat entscheiden, nicht Teil dieses Pakets |

# Refinement-Plan: Vom geprüften Kern zur belegten delegierten Entwicklung

- **Stand:** 10.10.2026, revidiert nach Gegenprüfung durch drei Prüfer (Anhang).
- **Autor:** Claude. Workflow: vier Faktenleser, drei unabhängige Entwürfe, zwei Jurys, Synthese, drei Prüfer, Revision.
- **Status:** Vorschlag. Der Nutzer entscheidet. Der Plan ist kein Roadmap-Eintrag, keine Abnahme und kein Auftrag an die gestoppten Codex-Chats. Als KI-Evidenz ersetzt er keine menschliche Annahme.
- **Grundlage:** [Unabhängiges Assessment](independent-assessment-20261010.md), die Nutzerziele in `docs/design/concepts/`, der untracked [KG-Refinement-Plan](knowledge-graph-refinement-plan-20261010.md).

**Untersuchte Stände** (lokale Refs, kein `fetch`; CI per `gh`, nur lesend):

| Kürzel | Branch | SHA | Rolle |
|---|---|---|---|
| MAIN | `origin/main` = `main` = `codex/product-integration-20261009` | `f12ffb00` | Merge von PR89 am 10.10. 16:57 (Eltern `5be48ce1`, `669cecd2`; Baum `e32118af` = Baum von `669cecd2`). Push-CI `38061655290` beim Prüfen `in_progress` |
| CO | `codex/government-assessment` | `6623cd0b` | Koordination; „collective stop effective“; untracked: `concepts/` (4 Dateien), Assessment, KG-Assessment, KG-Plan |
| GW / GP1 | `codex/government-worker` / `codex/government-p1` | `04e225d5` / `839dc4e3` | Government G1–G5 (+P1), Alt-Layout |

**Lesehilfe:**
- Aussagen mit Beleg (`Datei:Zeile`, `REF:Pfad`) sind Tatsachen. Pfade ohne Präfix gelten auf MAIN. „Vorschlag“ und „Urteil“ kennzeichnen Wertungen. „(neu)“ heißt: existiert heute nicht.
- **Kennungen:** Assessment-Befunde (K, A, E, U, P, KX, EX, UX, AX, PX) wie in dessen Anhang A. Plan-eigen: Nutzerentscheidungen **ND1–ND11**, Gates **Gate Start, Gate 0.5, Gate 1–3, Gate 4.x** (nicht zu verwechseln mit den Government-Stufen G1–G5), Pakete F0.x, S0.x, S1.x, B3.x, R4.x, N1–N6, Fehlstellen-Ursachen **Z1–Z5**. Pakete des KG-Plans immer mit Präfix „KG-“.
- **Messschritte** MS0, MS1a, MS1b entsprechen den Assessment-Stufen 0, 1a, 1b. Sie sind nicht die „Stages 1–5“ aus `docs/measurement.md:11-17`: MS0 ist ein Komponentennachweis zu Stage 1 („impact and fanout“) in realistischer Größe, MS1a die „check-only variant“ der Controlled model comparison (`measurement.md:77`).
- CT = ein Arbeitstag einer Claude-Sitzung mit Subagenten, keine Personenzeit.

## 0. Kurzfassung

- **Leitidee:** Erst die Kernthese billig, blind und fair messen, dann ausbauen. Die Kernthese: Modell, explizite Zuordnung und Impact lassen weniger relevante Stellen vergessen, decken Regelverletzungen auf, und die Modellpflege bleibt tragbar.
- **Ausgangslage:** PR89 ist gemergt. Der kollektive Stopp gilt („Resume only on new direct human work“, CO `product-queue-closure-20261010.yaml`, `stop_boundary`). Der Plan startet nur mit ND2.
- **Phase 0 (Doku, ein PR):** Nutzerziele und Grundlagendokumente sichern, den Main-Status nach dem Merge nachziehen, Government-Disposition, Entscheidungsnotiz.
- **Phase 1 = MS0 (ca. 2 Wochen, schlank):**
  - Zuerst den vermuteten Determinismus-Defekt in `impact.go` deterministisch prüfen und gegebenenfalls beheben (einziger möglicher Produkt-PR in Phase 1).
  - Dann Recall und Breite von `project impact` auf MyMeetings, gegen eine gleich ausgestattete `rg`-Suche und gegen M∪rg, ohne LLM in der Auswertung, dazu ein kleiner Drift-Arm.
  - 16 Fälle (2 dev, 14 holdout), Entscheidungstabelle vorab eingefroren.
- **Phase 2 = MS1a (Pilot):** Codex mit und ohne nachweislich auf Lesewerkzeuge beschränktes Markitect, 6 Änderungen × 2 Arme × 3 Läufe.
- **Phase 3 nach Evidenz:** Kernklasse ins Produkt (erst additiv, Routing nur bei `CoverageMode: full`), Kosten, MS1b, zweiter Anbieter.
- **Phase 4:** Government-Mechanismen einzeln, je mit Ein-Faktor-Gate.
- **Wichtigste Gates:** Gate 1: Recall ≥ 0,90 und ≥ `rg`, Zusatznutzen ΔR ≥ 0,10, Breite ≤ 3× Pflichtstellen, Drift-Treffer ≥ 0,90, Pflege ≤ 20 %. Gate 2: ≥ 30 % weniger verpasste Pflichtstellen bei ≤ 1,5-fachen Tokens, nur als Pilot.

## 1. Zielbild in 6–12 Monaten

Drei Schichten; eine Schicht wird erst gebaut, wenn die darunterliegende ihr Gate bestanden hat.

| Schicht | Inhalt | Gate | Stand heute |
|---|---|---|---|
| 1 Modellkern (deterministisch) | Compiler, Datei-Zuordnung, Coverage, Impact mit Klassen **Kern** und **indirekt betroffen** samt Begründung, Granularitäts-Leitfaden | Gate 1 | Mechanik vorhanden; Recall und Breite nie gemessen (E11, U2); Reihenfolge-Defekt vermutet |
| 2 Lieferablauf | Work Item → Plan → Run → Full Verify → Apply | Gate 2, Gate 3 | A01 bestanden (Actual28), Wirkung unbelegt (E1) |
| 3 Organisation (optional) | Government als zweite Achse „Idea/Work Item → Model“ innerhalb von Mandaten (`docs/design/project-world/operation-scopes-and-model-briefings.md:9-13`) | Gate 4.x je Mechanismus | nur deterministisch belegt (E8), seit 9.10. keine eigene Linie |

**Begriff:** „Kontext“ bleibt der Context-Semantik von Markitect vorbehalten (`engineering-constitution.md:16`). Beide Impact-Klassen sind betroffen und invalidiert. Die Klasse steuert später höchstens die Arbeitszuweisung, nie Verify, Unknown oder Coverage. Die „Kontextmenge“ des Assessments entspricht der Klasse „indirekt“ in ihrer Rolle für die Arbeitszuweisung.

**Nach etwa 3 Monaten, unabhängig vom Ausgang:** Die Kernthese ist auf einem mittelgroßen echten Repository gemessen (Recall, Breite, Drift-Treffer, Aktualität, Pflege; Agentenwirkung als Pilot). Das Ergebnis steht in einer datierten Entscheidung. Die Positionierung folgt dem Befund („So zumindest die Hoffnung“, `concepts/user-description-20261009.md`).

**Go-Pfad (12 Monate):** belegter selektiver Kern mit Granularitäts-Leitfaden (C18) und Klassen (C05, C19); Plan-Arbeit nach Kern, Full Verify, Unknown und Coverage voll; Hierarchie so tief, wie MS1b sie trägt; Selbstmodell (UX3); Government nur, soweit Gate 4.x bestanden. Ohne Konfiguration verhält sich das Produkt wie heute.

**No-Go-Pfad:** Markitect bleibt Struktur- und Prüfwerkzeug (Compiler, Ownership, Coverage, Impact als Prüfhilfe neben der Suche). `projectrun` geht in Wartung. Government bleibt eingefrorene Referenz; über eine Archivierung entscheidet der Nutzer.

**Nicht Ziel:** autonome Dauer-Entwicklungsmaschine; Kabinett, Pflicht-Einstimmigkeit, Gericht; Modell als Code-Spiegel (`architecture.md:91`); tiefere Rekursion ohne Messung; zweiter Ausführungsstapel; KG-Ausbau vor Gate 1; SAT oder Jev; ein semantischer Gesamtcheck (C12) vor Gate 2, weil er das Skalierungsproblem wiederholt (K8); A01-Wiederholung ohne Defekt; ein Release ohne Owner; das Aufräumen fremder WIP.

## 2. Leitprinzipien

1. **Messen vor Bauen.** Das kritischste Risiko ist Weiterbauen auf ungetesteter Prämisse (Assessment §10 Nr. 1). Neue Core-Semantik verlangt „repeated concrete failures“, Alternativen und ein generisches Design (`engineering-constitution.md:26`), bei Core-Erweiterung Fälle aus zwei Vokabularen (`docs/development/parallel-work.md:19`).
2. **Vorab einfrieren, die einfachere Baseline muss gewinnen können** (`measurement.md:49`, `:21`). Protokoll, Kernregel, Schwellen, Entscheidungstabelle und Split-Seed werden vor jedem Lauf per Hash fixiert, durch einen Akt des Menschen (§6). Die Baseline bekommt dieselbe Modellstufe und dasselbe Budget wie der Markitect-Arm.
3. **Blindheit durch Reihenfolge und Rollen.** Modell vor den Fällen eingefroren; Replay-Wahrheit mechanisch; gesäte Fälle und Drift-Mutationen nicht von Claude (K4, KX1); Isolation technisch, wo möglich, sonst als Grenze im Bericht (E5).
4. **Konservativ bleibt konservativ.** Die Klassen sind zuerst nur Erklärung. Eine spätere Eingrenzung betrifft nur die Arbeitszuweisung, nur bei `CoverageMode: full` und nie Verify, Unknown oder Coverage (AGENTS.md; `engineering-constitution.md:23`). Sicherheitstests sind Untergrenzen (⊇ erwartet); Breite wird berichtet, nie nach oben gedeckelt.
5. **Ein Owner je Regel und Fakt.** Die Kernregel steht bis B3.1 nur im eingefrorenen Protokoll; das Messwerkzeug rechnet sie experimentell, das Produkt übernimmt sie erst in B3.1 und muss die Werkzeug-Ergebnisse reproduzieren. Status steht nur im bestehenden Readiness-Backlog (`docs/implementation-plan.md:3`).
6. **Menschliche Zeit ist Messgröße und Budget** (Assessment §8 Nr. 7). Jeder Kontaktpunkt hat eine Minutenangabe.
7. **Schlank statt Belegmaschine** (P1, P3, P4, E3). Timebox je Paket, keine Chronik, keine Grants, Berichte mit Urteil zuerst. Neue Dateien ehrlich gezählt (§6).
8. **Kleine PRs nacheinander, eingefrorener Kopf, Mensch merged.** Ein Kandidat zur Zeit (`parallel-work.md:34`). Ein Timeout ist ein Befund (`CONTRIBUTING.md:72`).

## 3. Phasen und Arbeitspakete

| Phase | Zeitraum (Vorschlag) | Pakete | CT | Mensch | Endet mit |
|---|---|---|---|---|---|
| 0 Fundament | ab ND2, frühestens 12.10. | F0.1–F0.4 | 2,5 | ca. 1,5 h | Doku-PR auf main |
| 1 MS0 | 12.–23.10. (+1 Woche bei S0.7) | S0.0–S0.6 (S0.7 bedingt) | 10,5 inkl. KG-K1-Fix (+4 bei S0.7) | ca. 6 h | Gate 1 |
| 2 MS1a | ca. 26.10.–20.11. | S1.1–S1.5 (+1 CT bei Read-only-Flag) | 9 (+1) | ca. 8 h | Gate 2 |
| 3 Ausbau nach Evidenz | Dez. 2026–März 2027 | B3.1–B3.6, N1–N6 | 15–30 | 10–15 h | Gate 3 |
| 4 Government | frühestens Q2 2027 | R4.1–R4.6 | je 2–5 | je ca. 1 h | Gate 4.x |

Kumuliert für den Menschen: ca. 7,5 h bis Gate 1, ca. 15,5 h bis Gate 2. Die CT-Zahlen setzen eine Claude-Sitzung zur Zeit voraus; parallele Subagenten sind darin enthalten, parallele Sitzungen nicht.

### Phase 0 – Fundament (nur Doku, ein PR)

**F0.1 Nutzerziele und Grundlagen sichern.** Claude, Freigabe durch den Menschen. Branch `codex/refine-docs` direkt von `f12ffb00`. 0,25 CT, Mensch 10 min.
- **Inhalt:** die vier Dateien aus CO `docs/design/concepts/`; nach ND8 zusätzlich Assessment, KG-Assessment und KG-Plan (der KG-Plan verlinkt das KG-Assessment, Zeile 6; ohne es entstünde ein toter Link).
- **Zeilenenden:** Alle sieben Quellen haben CRLF; das Repo normalisiert auf LF (`.gitattributes:1`). Gesichert wird **inhaltsgleich bis auf Zeilenende**. Die SHA-256 der Originalbytes stehen in der Commit-Nachricht (Muster: `docs/research/README.md:5`). Keine `.gitattributes`-Änderung, weil P12 („explicit LF attributes“) zurückgestellt ist.
- **Ort:** `docs/design/concepts/` bleibt, weil Assessment und KG-Dokumente relativ dorthin verlinken. Ein Eintrag in der Dokumentationskarte `docs/README.md` kommt hinzu.
- **Einzige Textänderung:** Der Link `../delegated-engineering-operating-model.md` (`canonical-model-and-delegated-development.md:280`) existiert auf MAIN nicht; er wird als „nur auf `codex/government-assessment`“ gekennzeichnet.
- **Abnahme:** `git diff --no-index --ignore-cr-at-eol` gegen die Quelle zeigt nur die Link-Zeile. Ein Wegwerf-Skript prüft lokale Links und Anker aller geänderten Markdown-Dateien; das Ergebnis steht im PR (`CONTRIBUTING.md:76`). Eine Repo-Linkprüfung gibt es nicht: `src/harness/examples/markdown_navigation_test.go` testet nur das Fixture `examples/markdown-navigation`, und `CheckDocumentationRouters` ist ohne `spec.documentation` in `markitect.yaml` inaktiv (`src/internal/host/documentation_routers.go:10-15`).
- **Vorher:** Der Concepts-Autor wird gefragt (die Sammlung läuft fort, `concepts/README.md:13-15`); ND10 klärt, welche Kopie danach kanonisch ist.

**F0.2 Status nach dem Merge und Regelhygiene.** Claude. 0,75 CT, Mensch 20 min.
- **W8 (wichtigster Punkt):** Main führt P03–P10 noch als `in_progress` (`docs/work-items/product-readiness/backlog.yaml:1738-1811`) und die Main-Integration als ausstehend (`backlog.yaml:18`, `:673`; `docs/README.md:5`; `docs/implementation-plan.md:9`). Der Abschluss steht nur auf CO (`product-queue-closure-20261010.yaml`: P03–P10 done, P06 „done_with_known_nonblocking_limit“, P11–P13 „deferred_paused“). Vorschlag: diesen Stand samt Merge-SHA und Ergebnis der Main-CI in Backlog, README und Roadmap übernehmen. Weil der Backlog-Owner (Root) gestoppt ist, gibt der Mensch die Übernahme ausdrücklich frei.
- **W1:** `docs/project-workflow.md:37` und `docs/project-operations.md:43` („ATTEMPTED, NOT PASSED“) gegen `docs/README.md:5` (Actual28 passed); beide verweisen künftig auf das Ledger.
- **W4:** `CONTRIBUTING.md:92-94` („Legacy top-level init … retains“) gegen `docs/usage.md:438` („has been removed“).
- **W5:** `AGENTS.md:5` verweist auf die Astra-Disposition, die die Roadmap als historisch führt.
- **W7:** `docs/development/risk-register.md` hat 16 Zeilen mit Pfaden vor der Layout-Umstellung; nur ein Statuskopf „historisch“.
- **W3 nur markieren:** `CONTRIBUTING.md:3` gegen `:106` (Classic-Kern „compatibility-only“ oder eigener Betrieb) ist eine Produktentscheidung (ND9), keine Hygiene.
- **Neue CONTRIBUTING-Regel:** Während laufender CI keine Commits auf den PR-Kopf. Evidenz danach in einem quellgebundenen Bericht oder Commit; im PR-Kommentar nur der Verweis (Festschnappschuss-Evidenz, AGENTS.md).
- **Abnahme:** Grep auf die W-Fundstellen ist leer; Prüfungen nach `CONTRIBUTING.md:76`; CI auf beiden Betriebssystemen grün.

**F0.3 Government-Disposition.** Claude, Entscheidung durch den Menschen (ND1). 1 CT, Mensch 20 min.
- `docs/design/government-disposition-YYYYMMDD.md` (neu, höchstens 120 Zeilen) auf main: übernommene, zurückgestellte und verworfene Teile; eingefrorene SHAs `04e225d5`, `839dc4e3`; Begriffsabbildung (§5.5); Erweiterungspunkte V1–V6, ausdrücklich nicht implementiert.
- **Gründe für den Wechsel am 9.10.** (A1) nur als Nutzerzitat mit Quelle oder gekennzeichnet „rekonstruiert, vom Nutzer zu bestätigen“.
- **Leitplanke** (kein neuer Kerncode setzt einen Reviewer je Manager-ID voraus) nur als nicht bindender Hinweis. Bindend würde sie erst durch eine Produktentscheidung in `architecture.md`, denn Government fügt der Classic-Methode keine Anforderungen hinzu (`operating-methodology.md:82`).
- **CO:** Claude committet dort nichts. Statusblöcke für `docs/design/government/*.md` und die Korrektur von CO `docs/README.md:12` (AX3) gehen als Patchvorschlag an den Nutzer, Datei für Datei freizugeben.
- **Abnahme:** Ein Faktenprüfer findet keine falsche Fundstelle.

**F0.4 Entscheidungsnotiz und Status.** Claude, Annahme durch den Menschen. 0,5 CT, Mensch 30 min.
- **Neu:** `docs/design/impact-measurement-decision-YYYYMMDD.md` (höchstens 120 Zeilen): angenommene ND, Gates, Schwellen, read-only Inventur der Branches und Worktrees (Löschen nur nach Einzelfreigabe, P11, U9). Kein Name mit „refinement“, um die Kollision mit `docs/refinement.md` zu vermeiden.
- **Status:** neue Einträge (F0.x, S0.x, S1.x) in `docs/work-items/product-readiness/follow-up-backlog.yaml`, wo Nach-Main-Items schon stehen (R03-F01, KG01–KG03). Kein zweiter Statusort, kein Status-README.
- **Geändert:** Roadmap höchstens 5 Zeilen (nächste Grenze nach P10, Verweis auf die Notiz, Aufhebung der Case-Study-Sperre für MS0/MS1a nach ND7, `backlog.yaml:7`); `docs/refinement.md` eine Zeile Verweis, sobald B3.1 eine Produktentscheidung wird.
- **KG01–KG03** (eigener Branch `codex/knowledge-graph`, eigener Owner-Thread) werden nicht ersetzt, sondern per Statusfortschreibung mit ND8-Beleg behandelt.

### Phase 1 – MS0: Kernthese ohne LLM in der Auswertung

**S0.0 Determinismus zuerst (KG-K1).** Claude, Codex-Review. 0,5 CT Prüfung, bei Bestätigung +1,5 CT Bugfix.
- **Tatsache:** `impact.go:276-283` baut die Queue aus `for id := range seed` (Go-Map, zufällige Reihenfolge). Über `uses` erreichte Statements kommen ohne `addCoverage` in die Hülle (`:288-296`); als Konsument werden sie später übersprungen (`!closure[consumer.ID]`, `:312`). Beispiel: A und B geändert, A uses X, X uses B. Wird A zuerst verarbeitet, fehlen X' Dateien und Checks.
- **Prüfung:** ein konstruierter Test auf einem Export von `f12ffb00` mit genau dieser Kette, ausgeführt mit `-count=200`. Der Doppellauf des früheren Entwurfs hätte den Defekt nur mit etwa 2 % Wahrscheinlichkeit gefunden (KG-Plan: 2 Digests in 200 Läufen).
- **Bei Bestätigung:** KG-K1 als erster Code-PR. Invariante: Ergebnis unabhängig von der Reihenfolge und ⊇ jedem Ergebnis der alten Implementierung; Konsumenten bekommen ihre Coverage auch dann, wenn sie schon über `uses` in der Hülle sind. Der PR ändert den Digest und benennt die Konsumenten (`projectrun/plan.go:357-358`, `:760-763`; `projectbriefing/briefing.go:73`).
- **Regel:** holdout läuft nur mit einem deterministischen Binary, gebaut aus einem exakten, reviewten SHA mit protokolliertem Hash (notfalls vor dem Merge aus dem eingefrorenen PR-Kopf).

**S0.1 Protokoll, Testrepository, Snapshot.** Claude, Freeze durch den Menschen. 1 CT, Mensch 45 min.
- **Ort:** im Evidenzordner, nach Gate 1 als `docs/validation/impact-measurement/ms0-protocol.yaml` (neu, YAML nach `measurement.md:3`).
- **Inhalt:** Repository, Snapshot S, Modellumfang, mechanische Fallauswahl, Arme, Metriken, Schwellen, vollständige Entscheidungstabelle (§4), Split-Seed, Rollen, Ursachen Z1–Z5, Granularitäts-Leitfaden als Hypothese (C18).
- **Kernregel (eigenständig, vollständig):**
  - *Kern:* alle Seeds, also geänderte Definitionen, Datei-Seeds aus Diffs samt Statements, Artefakten und Checks ihrer Dateien, Manager- und Check-Änderungen; deren Realisierungen, Dateien und Checks; `requires`-Ziele geänderter Statements samt Coverage (A24, `docs/design/project-world/implementation-plan.md:153`); direkte `uses`- und `requires`-Konsumenten geänderter Statements (öffentlich und privat) samt Coverage; alles aus Unknown-Verbreiterung (`impact.go:352-379`).
  - *Indirekt:* der Rest der heutigen Hülle (transitive Konsumenten, `uses`-Abhängigkeiten, Ahnen-Routing `impact.go:324-335`).
  - *Vereinigung* = heutige Hülle. Shared-File-Mitrealisierungen sind nicht enthalten, weil die heutige Hülle sie nicht kennt.
  - *Abweichung vom KG-Plan K3* (`knowledge-graph-refinement-plan-20261010.md:227-243`): dort Shared-File-Verbreiterung und Digest-Änderung, hier keine; A24-Ziele hier im Kern. Jede Präzisierung nach dem Freeze gilt als neue Runde auf frischen Fällen.
- **Testrepository (ND3):** MyMeetings upstream (`kgrzybek/modular-monolith-with-ddd`, MIT, C#), Klon im Evidenzordner. S liegt vor dem Pin `91c8ef24`, denn der Pin war `master` zur Vorbereitungszeit (`experiments/real-project-adoption/provenance.md:7`); er dient nur als Referenz. Modellumfang: die 646-Dateien-Auswahl (`provenance.md:11`), auf S projiziert; Pfade außerhalb zählen als Z4 und werden getrennt berichtet.
- **`rg`:** ist auf dem Rechner nicht installiert (PowerShell `Get-Command rg` leer; nur gebündelte Kopien in VS Code und im Codex-npm-Paket). ripgrep 14.1.1 wird als Release geladen, SHA-256 protokolliert, per absolutem Pfad aufgerufen.
- **Abnahme:** Der mechanische Filter liefert mindestens 30 funktionale Kandidaten-Commits nach S (MS0 10, Reserve für S0.7 6, MS1a 6, MS1b 8); sonst greift an T2 der Fallback (nur gesäte Fälle oder ein Fenster aus Markitects Historie). Protokoll-Hash per Akt des Menschen eingefroren.

**S0.2 Blindes Modell mit Frühwarnung Gate 0.5.** Modellierer, Abnahme durch den Menschen. Höchstens 2 CT, Mensch 60–90 min (gemessen).
- Der Modellierer bekommt nur `git archive S` (Modellumfang, ohne Historie und Fälle), `docs/project-workflow.md` und den Leitfaden. Er arbeitet als eigener Prozess mit Arbeitsverzeichnis im Export und Leseverboten außerhalb; der Mechanismus wird vorher getestet, das Restrisiko (gleiches OS-Konto) steht im Bericht.
- Werkzeug: `markitect project init` und `project edit` mit dem Binary aus `f12ffb00` (Hash protokolliert). Erfasst werden Modellzeilen, Manager, Statements, Artefakte, Agenten- und Menschminuten. Vergleichswert: Shop-Beispiel 378 YAML- zu 257 Python-Zeilen.
- **Gate 0.5:** mehr als 2 CT, mehr als 1.500 YAML-Zeilen oder `check`/`coverage` nicht konform → einseitige Notiz vor dem Fallbau (C16).
- **Abnahme:** `project check` succeeded, `project coverage` conforming, Modell-Hash vor S0.3 eingefroren.

**S0.3 Fälle und Pflichtstellen.** Getrennte Rollen. 2 CT plus Autor, Mensch 30 min.
- **Umfang (Default):** 10 Replay-Fälle (2 dev, 8 holdout) und 6 gesäte holdout-Fälle (je 2 fachliche Umbenennungen, Regeländerungen, Vertragsänderungen). Dazu der Drift-Arm.
- **Replay:**
  - Wahrheit T mechanisch: geänderte Dateien des Commits minus protokollierte Rauschklassen.
  - Intent = wörtliche Commit- bzw. PR-Nachricht. Reicht sie nicht, fällt der Fall nach Protokollregel heraus; niemand formuliert ihn um.
  - Chronologisch je Commit: Ist er ein Fall, schreibt der Modellierer zuerst das Modell-Delta aus Intent und Modell am Elterncommit, ohne den Diff. Erst danach führt eine **getrennte** Pflege-Instanz das Modell mit dem echten Commit nach. So sieht kein Delta-Autor die Lösung.
  - Messung: base = Elterncommit samt nachgeführtem Modell; revision = base + nur das Modell-Delta, mechanisch erzeugt. Das Werkzeug prüft, dass außerhalb `.markitect/` nichts geändert ist.
  - **Aktualität** vor jeder Nachführung: Anteil berührter Dateien mit Klasse `realization`, `tool-owned` oder `ignored` und ohne `coverage.*`-Befund (`projectcoverage/types.go:18-22`). **Pflege:** Modellzeilen der Nachführung je geänderter Codezeile. Veraltung wird als Aktualität ausgewiesen, nicht als Recall-Verlust.
- **Gesäte Fälle:** Intent, Pflicht- und Nicht-Ändern-Stellen schreibt ein **Nicht-Claude-Autor** (ND4), der Code sieht, nicht das Modell. Er startet schon an T2, weil er das Modell nicht braucht; getrennte Ordner halten den Modellierer blind.
- **Drift-Arm (C17, Assessment-Kriterium 3):** derselbe Autor liefert 6 Code-Mutationen, die je eine modellierte Verpflichtung verletzen, und 6 harmlose Änderungen. Gemessen: Liegt die verletzte Verpflichtung samt Check in der Impact-Menge? Schlägt der Check an? Den zweiten Teil gibt es nur bei vorhandener .NET-Toolchain, sonst `unavailable`. Ob deklarierte Checks ohne LLM-Rolle ausführbar sind, ist nicht geprüft; S0.1 klärt das.
- **`rg`-Arm:** Ein rg-Autor mit **derselben Modellstufe und demselben Budget** wie der Modellierer sieht Intent und Code am Elterncommit, nicht T und nicht das Modell. Er sucht iterativ bis zu einer im Protokoll festgelegten Zahl von Abfragen.
- „Keine Modelländerung nötig“ zählt als Ergebnis (P leer).
- **Abnahme:** Manifest-Hash; der Mensch hat 3 gesäte Fälle und 2 Drift-Mutationen stichprobenartig geprüft.

**S0.4 entfällt.** Die additive Kernklasse im Produkt (`--explain`) wandert nach B3.1 Teil 1. In MS0 rechnet nur das Messwerkzeug die Kernklasse experimentell; so entsteht keine CLI-Fläche für eine ungeprüfte Regel.

**S0.5 Messwerkzeug.** Claude. 2 CT, Mensch 15 min.
- Go-Programm (`measurement.md:3`) unter `experiments/impact-measurement/` (neu; Ort mit `docs/development/modules.md` abgleichen), Ergebnisse als YAML mit Hash-Manifest im Evidenzordner.
- Es ruft das Binary, `git` und das gepinnte `rg` als Prozesse auf, expandiert Verzeichniseinträge in `files` am Basis-Snapshot und schreibt je Fall und Arm P, T, Recall, Breite (Dateien), Fehlstellen.
- **Arme:** M (volle Menge), M-Kern (experimentell aus der eingefrorenen Kernregel und `project index` beider Revisionen), rg, dir (Projektverzeichnis der Seed-Datei), M∪rg.
- Für MS1a übernimmt S1.3 aus `experiments/agents-md-comparison/run.ps1` den Aufruf `codex exec --ignore-user-config --ephemeral --json` (`:1100`), das Usage-Parsing (`:636-656`) und die Sequenzsperre (`:898-957`) oder begründet einen Neubau.
- **Abnahme:** Smoke auf `examples/project-world` reproduziert 9 Statements und 6 Manager je Codedatei; mit dem deterministischen Binary ergeben 200 Läufe je Prüfstruktur denselben Ergebnis-Hash.

**S0.6 Lauf und Gate-1-Vorlage.** Werkzeug, Klassifikation durch Claude, Bestätigung durch den Menschen. 1,5 CT, Mensch 1,5 h.
- dev-Lauf nur zur Plausibilität; holdout genau einmal mit allen Armen.
- **Ursachen:** Z1 Datei unzugeordnet, Z2 Relation fehlt, Z3 Artefaktgruppe zu grob, Z4 außerhalb des Modellumfangs, Z5 Pflichtstelle strittig (Ausschluss nur mit Zustimmung des Menschen, Fall bleibt sichtbar).
- **Abnahme:** Vorlage auf einer Seite, alle Quoten mit Zähler und Nenner (`measurement.md:81`); Bericht `docs/validation/impact-measurement/ms0-report-YYYYMMDD.md` (neu) mit Codex-Review.

**S0.7 Nachbesserung (nur nach Entscheidungstabelle).** Höchstens 4 CT. Genau eine Runde je nach Ursache (Modellvariante bei Z2/Z3, Zuordnung oder Umfang bei Z1/Z4, sonst Kernregel-Variante). Konservative Menge unverändert. Gemessen nur auf den 6 Reserve-Fällen, die erst nach der Entscheidung gezogen und eingefroren werden.

### Phase 2 – MS1a (nur bei Gate 1 Go, nach ND7)

**S1.1 Protokoll.** Claude, Freeze durch den Menschen. 2 CT, Mensch 1 h.
- `docs/validation/impact-measurement/ms1a-protocol.yaml` (neu): 6 Änderungen aus der Replay-Reserve; das Modell ist am Elterncommit jeder Änderung gepinnt.
- **Arm C:** Codex mit gutem AGENTS.md, Tests und Review. **Arm M:** identisch, dazu das angenommene Modell und nur `project_check`, `project_index`, `project_context`, `project_impact`, `project_coverage`. Gleiches Modell, gleicher Effort, gleiches Budget, randomisierte Reihenfolge.
- **Wiederholung:** jede Änderung 3× je Arm, 36 Läufe (`measurement.md:79`). Unsicherheit vorab eingefroren: Vorzeichentest auf Änderungsebene über den Median der drei Läufe, plus Streuung je Arm.
- **Abnahme:** Trockenlauf des Evaluators gegen eine gute und eine schlechte Musterlösung.

**S1.2 Werkzeugbeschränkung und Isolation.** Claude, Konto durch den Menschen. 2 CT (+1 CT bei Read-only-Flag), Mensch 1 h.
- **Tatsache:** Der Markitect-MCP-Server hat keinen Read-only-Modus (`projectcli/options.go:41`: einziges Flag `--repo`). Derselbe Server registriert `project_edit`, `project_init`, `project_run`, `project_apply`, `project_deliver` u. a. (`projectcli/mcp.go:47-96`, `src/internal/host/mcp/tools.go:69-88`).
- **Vorgehen:** zuerst per Doku und Konfigurationstest klären, ob Codex eine Tool-Allowlist je MCP-Server kann (nicht geprüft). Sonst ein kleines Read-only-Flag als eigener Code-PR mit Codex-Review. Ein Negativtest prüft `tools/list` bzw. die Tool-Events: nur die fünf Lesewerkzeuge.
- **Isolation:** eigenes Windows-Konto, je Lauf frischer Klon ohne Eltern-`AGENTS.md`, leere Codex-Memory, ephemere Ports, Orakel per ACL gesperrt (E5, UX1). Im neuen Konto wird `storageRoot` explizit gesetzt, weil der Default unter umgeleitetem AppData offen ist (P13, zurückgestellt).
- Binary aus exaktem SHA, Hash protokolliert. **Abnahme:** negativer Isolationstest als Skript; je Arm ein Trockenlauf bis zur Thread-Aufnahme.

**S1.3 Dispatcher.** Claude. Ort `experiments/impact-measurement/` (nur Code). 2 CT. Zustandsdatei, fortsetzbar (E4, P5); Usage aus Codex-JSONL, fehlend = `unavailable`, nicht 0. **Abnahme:** Neustart nach Abbruch ohne Doppelstart.

**S1.4 Läufe und Blindprüfung.** 2 CT Betreuung, Mensch ca. 4 h. Höchstens 36 Hauptläufe plus 2 Trockenläufe plus je INVALID höchstens eine protokollierte Wiederholung. Der Mensch prüft 12 anonymisierte Diffs (je Änderung und Arm ein zufällig gezogener Lauf) blind und stoppt die Minuten.

**S1.5 Auswertung und Gate-2-Vorlage.** 1 CT, Mensch 1 h. Jede verpasste Pflichtstelle wird markiert: „Querschnitt ja/nein“, „Kontext fehlte ja/nein“ (Messhaken für Gate 4.2 und C19). Bericht `docs/validation/impact-measurement/ms1a-report-YYYYMMDD.md` (neu).

**Verhältnis zu `docs/design/work-item-comparison-20261009.md`:** Das CO-Dokument weist den Vergleich dem gestoppten Scientist zu. MS1a ersetzt diesen Vergleichsteil für die Dauer des Programms, damit er nicht zwei Owner hat (ND7).

### Phase 3 – Ausbau nach Evidenz (bedingt)

**B3.1 Kernklasse ins Produkt.** Bedingung: Gate 1 Go oder Zweig „Recall ok, Breite zu groß“, dazu R(M-Kern) ≥ 0,90 und Median-Breite(Kern) ≤ 3× Pflichtstellen (Dateien). Vorbedingung nach `engineering-constitution.md:26` und `parallel-work.md:19`: Alternativen (Normalisierung, Check, Adapter), generisches Design, Belege aus zwei Vokabularen (`examples/project-world` und MyMeetings), Entscheidung in `docs/refinement.md`. Codex-Review Pflicht.
- **Teil 1 (additiv, digestneutral, 2 CT):** `project impact --explain` (neu) mit Klasse `core|indirect`, Grund-Code und auslösender Kante je Statement, Datei und Check. `ChangeImpact` und Digest bleiben byte-gleich (`impact.go:395`, `analyze.go:577-583`). Property-Tests: Vereinigung = bisherige Mengen, Kern ⊆ voll, Kern = Werkzeug-Ergebnis auf allen MS0-Fällen. Doku in `docs/project-operations.md`.
- **Teil 2 (Vertrag und Routing, eigene Nutzerentscheidung, 3 CT):** Klassen in `ChangeImpact` und MCP (Digest ändert sich, Pläne werden `ErrStale`, steht im PR und in `project-operations.md`). Routing nur bei `CoverageMode: full`, denn nur dann läuft Full Verify (`projectrun/verify.go:259`); in anderen Modi bleiben die Tasks wie heute (`plan.go:503-563`). Indirekte Owner bleiben mit Grund und Kante in Plan und Briefing sichtbar, ihre Checks laufen (A25), ein fehlendes Urteil ist kein PASS. Property-Tests für beide Modi. Die Abweichung von KG-K4 („Manager-Auswahl bleibt unverändert“, KG-Plan Zeile 293) wird benannt.
- **Granularität:** nicht normative Authoring-Anleitung in `docs/project-workflow.md`; MyMeetings-spezifische Schnitte bleiben im MyMeetings-Modell (AGENTS.md, projektlokale Policy).
- Nachmessung nur auf unverbrauchten Fällen.

**B3.2 Kostenerfassung (bei Gate 2 Go).** 3 CT. Helper- und Kind-Usage aggregieren (`projectrun/cost.go:24-33` zählt sie heute als unknown, U10); Abgleich mit Usage-Exporten.

**B3.3 MS1b (bei Gate 2 Go).** 10 CT, Mensch 6 h. M-Deliver (`project deliver` mit Managern und Reviewern) gegen M-Info, 8 Änderungen aus der Reserve; Elternintegration und Kompositionsfehler getrennt berichtet (Stage 3, `measurement.md:15`); Readiness getrennt; kein A01-Rerun.

**B3.4 Zweiter Anbieter (bei Gate 2 Go, U3, Risiko 9).** ca. 5 CT. Claude-Adapter mit Tools (heute ohne Tools und mit einem Turn, Assessment Z. 193), dann MS1a-Wiederholung mit zweitem Anbieter; bewertet von der jeweils anderen Modellfamilie. R03-F01 (Anbieter-Leitfaden-Projektion, `follow-up-backlog.yaml:34-44`) wird dabei mit entschieden. Bis dahin gilt jede Wirkungsaussage nur für Codex.

**B3.5 Modell-Tiering (bei Gate 3 Go).** 8 CT. Gekreuzter Vergleich stark gegen gestaffelt (C10, K5).

**B3.6 No-Go-Pfad.** 2–3 CT. `vision.md` und Roadmap halten fest: Hypothese nicht bestätigt, Fokus auf den selektiven Kern; `projectrun` in Wartung; optional reale Vergessens-Vorfälle aus Nutzerprojekten sammeln (K12).

**Nach-Gate-2-Backlog** (je eigener Nutzerentscheid):
- **N1 Selbstmodell:** erst als Overlay unter `src/harness/…/testdata/` (`docs/usage.md:10` verbietet das Mischen der Formate), dann Umstellung des Eigenbetriebs (UX3).
- **N2 Ein Ausführungsstapel** (Assessment U1): Canonical-Controller und Projection-Alpha entfernen, erst nach N1 und ND9.
- **N3 CLI/MCP-Asymmetrie** (Assessment UX-Befunde) nach eigener Inventur; `project_preflight` ist kein Beispiel, die CLI hat es als `project apply` ohne `--write` (`projectcli/run.go:509-520`).
- **N4 `runOrResume`** (`projectrun/run.go:46`; Assessment U7) zerlegen, nur wenn ein Paket die Funktion ohnehin berührt.
- **N5** KG-K2, KG-K4, KG-K5 und `project affected` nach den Stoppregeln des KG-Plans.
- **N6** Binärevidenz in ein Artefaktlager auslagern, Branches nach Freigabe löschen.

### Arbeitspakete im Überblick

| ID | Ziel | Abnahme (Kern) | Aufwand | abhängig von |
|---|---|---|---|---|
| F0.1 | Nutzerziele, Grundlagen gesichert | inhaltsgleich bis auf Zeilenende und Link; Linkskript | 0,25 CT | ND2, ND8, ND10 |
| F0.2 | W8, W1, W4, W5, W7; Evidenz-Regel | Grep leer, CI grün | 0,75 CT | ND2, Main-CI grün |
| F0.3 | Government-Disposition | ≤ 120 Z., Faktencheck | 1 CT | ND1 |
| F0.4 | Entscheidungsnotiz, Status im Follow-up-Backlog | keine weiteren Dateien | 0,5 CT | ND1–ND11 |
| S0.0 | Determinismus (KG-K1) | konstruierter Test, ggf. Bugfix-PR | 0,5 (+1,5) CT | ND2 |
| S0.1 | Protokoll, Kernregel, S, `rg`-Pin | Hash per Akt des Menschen | 1 CT | ND3, ND5 |
| S0.2 | Blindes Modell | check/coverage grün; Gate 0.5 | ≤ 2 CT | S0.1 |
| S0.3 | 16 Fälle, Drift-Arm, Replay-Folge | Manifest-Hash, Stichprobe | 2 CT + Autor | S0.2, ND4 |
| S0.5 | Messwerkzeug | Smoke 9/6, 200 Läufe gleich | 2 CT | S0.0, S0.1 |
| S0.6 | holdout, Gate-1-Vorlage | Zähler/Nenner, Codex-Review | 1,5 CT | S0.3, S0.5 |
| S0.7 | eine Nachbesserung | nur Reserve-Fälle | ≤ 4 CT | Entscheidungstabelle |
| S1.1–S1.5 | MS1a | Gate-2-Vorlage, Beschränkung und Isolation bewiesen | 9 (+1) CT | Gate 1 Go, ND7, ND11 |
| B3.1 | Kernklasse (Teil 1, Teil 2) | Property-Tests, Digest-Folgen benannt | 2 + 3 CT | Gate 1, Kernbedingung |
| B3.2–B3.6 | Kosten, MS1b, 2. Anbieter, Tiering, No-Go | je Gate-Vorlage | siehe oben | Gate 2/3 |
| R4.1–R4.6 | Government-Mechanismen | je Gate 4.x | je 2–5 CT | §5 |

## 4. Entscheidungs-Gates

Die Schwellen sind Pilotschwellen aus Assessment §11, nicht aus Varianzdaten (EX3). Jedes Protokoll enthält eine **vollständige Entscheidungstabelle**: Die Regeln werden in fester Reihenfolge geprüft, die erste zutreffende gilt, die letzte fängt alles Übrige. Der Mensch entscheidet; Claude empfiehlt.

| Gate | Bedingung und Folge |
|---|---|
| **Gate Start** | Merge PR89 (erfüllt, `f12ffb00`) und Handoff (erfüllt, CO `6623cd0b`); Main-CI `38061655290` grün (beim Prüfen offen); ND2. Der Stopp hebt sich nicht von selbst auf. |
| **Gate 0.5** (in S0.2) | ≤ 2 CT, ≤ 1.500 Zeilen, konform → S0.3. Sonst einseitige Notiz; der Mensch wählt anderen Leitfaden oder kleineren Umfang; dominiert die Pflege schon beim Aufbau → B3.6 erwägen. |
| **Gate 1** (holdout) | 1. **Null-Befund:** R(rg) ≥ 0,90 und ΔR < 0,05 → keine MS1a für diese Änderungsart; Mensch wählt andere Änderungsklasse oder größeres Repo. 2. **Go:** R(M) ≥ 0,90 (gesät ≥ 0,95) und ≥ R(rg); ΔR = R(M∪rg) − R(rg) ≥ 0,10; Median-Breite(M) ≤ 3× Pflichtstellen; Drift-Treffer ≥ 0,90 (bei ausgeführten Checks Fehlalarm ≤ 10 %); Aktualität ≥ 0,95; Pflege ≤ 20 % der geänderten Codezeilen → Phase 2. 3. **Recall ok, Breite zu groß** (alles außer Breite erfüllt) → B3.1 Teil 1, dann Phase 2 mit Kernklasse, sofern die Kernbedingung erfüllt ist; sonst wie 5. 4. **No-Go:** R(M) < R(rg), Pflege > 50 % oder Schwellen nach S0.7 weiter verfehlt → B3.6. 5. **Übrige Fälle** (z. B. ΔR zwischen 0,05 und 0,10) → S0.7 einmal. 6. Restfall → der Mensch entscheidet mit schriftlicher Begründung. |
| **Gate 2** (MS1a, Pilot) | 1. **Null-Befund:** Σmiss(C) ≤ 5 % der Pflichtstellen → Problem in dieser Größe nicht reproduziert, kein Ausbau. 2. **Readiness-Befund:** Produktausfälle in ≥ 3 von 18 M-Läufen → beheben, MS1a einmal mit neuen Änderungen wiederholen. 3. **Go (nur nächster Messschritt, keine Produktaussage):** Σmiss(M) ≤ 0,7·Σmiss(C); Vorzeichentest zugunsten M (deskriptiv, Wert berichtet); Verstöße gegen Nicht-Ändern M ≤ C; Tokens M ≤ 1,5·C; Prüfminuten M ≤ C; Kosten unter der Grenze aus ND11 → B3.2, B3.3, B3.4. 4. Sonst → B3.6. 5. Restfall → Mensch, schriftlich. |
| **Gate 3** (MS1b) | miss(Deliver) ≤ miss(Info); Prüfminuten D ≤ 0,7·I; Kosten D ≤ Referenz (Assessment Kriterium 6) → Hierarchie als Standardweg, B3.5, R4.x möglich. Sonst bleibt die Hierarchie optional, keine weitere Rekursion. |
| **Gate 4.x** | je Mechanismus (§5.3) → optionale Konfiguration; sonst bleibt er Idee, kein automatisches Archiv. |

Jede Gate-Vorlage hat eine Seite, Urteil zuerst, Quoten mit Zähler und Nenner.

## 5. Government-Erweiterungspfad

### 5.1 Einordnung

- **Tatsachen:** MAIN enthält weder Government-Code noch -Doku. Das Design lehnt Pflicht-Ministerien und Einstimmigkeit ab (`docs/design/project-world/model.md:134`, `:183`) und ordnet Government der zweiten Achse zu: Managementebenen entscheiden innerhalb ausdrücklicher Mandate, Ressorts sind „spätere Gestaltungsmöglichkeiten“ (`operation-scopes-and-model-briefings.md:11-13`). Die Nutzerbeschreibung vom 9.10. nennt weder Kabinett noch Einstimmigkeit noch Gericht.
- **Vorschlag:** Government ist Schicht 3, kein zweiter Stapel. Ob Ressorts und Einstimmigkeit noch Ziel sind, ist ND1 (Assessment offene Frage 1, K13).

### 5.2 Jetzt vorbereiten (Phase 0–2, kein Government-Code)

- **Disposition (F0.3)** mit Begriffsabbildung und Erweiterungspunkten:
  - **V1 Begründung je Impact-Eintrag** und **V2 Kern/indirekt:** real mit B3.1. Grundlage einer späteren Kontext-Nachforderung (C19).
  - **V3 Urteils-Record:** Full Verify verlangt heute schon, dass alle konfigurierten Manager bestehen; fehlende oder unvollständige Urteile sind kein PASS (`projectrun/full_verify.go:449-498`). Es fehlen ein Record mit Rolle, Runde, Kandidat- und Evidenz-Digest und ein benannter Regelwert.
  - **V4 Prüferzuordnung (Manager, Belang) → Reviewer:** heute nur je Manager-ID (`projectrun/types.go:69-76`).
  - **V5 Entscheidungsklasse:** Befugnis kommt immer aus der Vorversion (`projectwork/mutation.go:503-575`).
  - **V6 Authentifizierte menschliche Annahme / Owner-Kanal** (AX2): Heute binden Commits und Digests Bytes, authentifizieren aber keine Zustimmung (`CONTRIBUTING.md:3`). Voraussetzung für R4.3.
  - V3–V6 kommen nicht als normativer Text in `delivery-contract.md` (sonst wiederholt sich A7).
- **Messhaken:** „Querschnitt“ und „Kontext fehlte“ aus S1.5.

### 5.3 Angliederung (Phase 4, je Mechanismus ein Ein-Faktor-Gate)

| ID | Mechanismus | Andockstelle (MAIN) | Voraussetzung | Gate 4.x (Go) |
|---|---|---|---|---|
| R4.1 | Urteils-Record, Regel `all-responsible` als einziger Wert | `FullManagerAssessment` (`full_verify.go:100-108`), Regel `:449-498`; Apply bleibt an den Verify-Digest gebunden (`apply.go:260`) | Gate 3 Go | 4.1: Verhalten byte-gleich, Recovery-Tests grün |
| R4.2 | Querschnittsprüfer (Ressort) ohne Schreibrecht | `ReviewConfig` (`types.go:69-76`), zunächst Runtime-Konfiguration | ≥ 30 % der Fehlstellen aus S1.5/B3.3 sind Querschnitt. **Bei ND1 = „Produktidentität“:** als Ein-Faktor-Test auf dem MS1a-Harness nach Gate 2, unabhängig von Gate 3 und `deliver` | 4.2: gleiches Token-Budget, ≥ +20 Prozentpunkte Entdeckung gesäter Querschnittsdefekte gegenüber einem allgemeinen Reviewer, ≤ 10 % Fehlblockaden |
| R4.3 | Entscheidungsklassen, geschützte Aussagen, mandatierte Modellpflege | `mutation.go:503-575` | ND1 (Agenten dürfen Modell ändern), V6 | 4.3: Modell-Review-Minuten ≤ Code-Review-Minuten je Item, 0 erfolgreiche Selbstermächtigungen |
| R4.4 | Owner-Rückkanal | Eskalation an `user` offen (`projectrun/run.go:2595-2600`); Muster `actor/authorityClaim/decisionReference` nur für Brownfield (`projectcli/resolution.go:20-24`) | R4.3 | 4.4: Lauf setzt nach Owner-Antwort ohne Replay fort |
| R4.5 | GP1-Robustheit (Fencing-Lease, Promotion-Intent, Hash-Journal) als einzelne Ports | `projectadoption/process_lock_windows.go`; guarded Apply ist „deliberately not a multi-file transaction“ (`project_guarded_write.go:53-56`) | konkreter Defekt oder vor R4.6 | 4.5: Crash-Tests je Schritt |
| R4.6 | Queue und Dauerbetrieb | `deliver` je Scope | Nutzerwunsch, R4.5, Gate 3 Go | 4.6: ≥ 5 abhängige Items, 0 veraltete Applies |

### 5.4 Zurückgestellt und verworfen

- **Zurückgestellt:** Pflicht-Einstimmigkeit als Default; Kabinett- und Verfassungsvokabular als Produktbegriffe; Organisationsänderung durch Agenten; Kontext-Nachforderung als eigener Host-Kanal (erst, wenn S1.5 Kontextlücken zeigt); `project affected` (N5); SAT und Jev nur auf direkten Nutzerauftrag.
- **Verworfen bis zu neuer Nutzerentscheidung:** Gericht und Gewaltenteilungsrollen (A6); Merge von GP1 (Alt-Layout); Rechte-Konfiguration je Rolle durch den Nutzer (`native-runtime-simplification-20261010.md:9-13`).

### 5.5 Begriffsabbildung (für die Disposition)

| Government | Design / MAIN | Heute im Code | Andockpunkt |
|---|---|---|---|
| Verfassung | committed Modell `.markitect/`, Plan bindet `ModelDigest` | vorhanden, ohne geschützten Kern | V5 → R4.3 |
| Bereich (rekursiv) | Manager (`parent`, `owns`) | vorhanden | – |
| Mandat | Delegation mit Zielen, Grenzen, Vorbehalten (`model.md:102`) | Namespace-Befugnis; Vorbehalte nur Freitext | V5 → R4.3 |
| Ressort / Kabinett | Querschnittsprüfer, „spätere Gestaltungsmöglichkeit“ | fehlt; Reviewer nur je Manager-ID | V4 → R4.2 |
| Einstimmigkeit | Regelwert über Urteile am selben Kandidaten | Full Verify verlangt alle Manager | V3 → R4.1; als Regelwert erst nach Gate 4.2 |
| Executor / Verifier | Manager plus unabhängiger Reviewer | vorhanden | – |
| Elternintegration, Kompositionsfehler | Parent-Pflichten (Stage 3, `measurement.md:15`) | hierarchisches Full Verify | B3.3 misst |
| Schritt B (vorläufige Betroffenheit) | `operation-scopes…:15` | nur `impact` nach Kandidat | V1/V2; N5 |
| Kontext-Nachforderung | Rückfrage an Kindmanager (`model.md:106`), C19 | fehlt | V2 |
| Annahme durch Owner | Owner-Entscheidung | nicht authentifiziert | V6 |
| PromotionIntent, CAS, Lease | guarded Apply, Lockdatei plus PID | schwächer als GP1 | R4.5 |
| Eskalation an Owner | subsidiäre Eskalation (`model.md:128-136`) | kein Rückweg | R4.4 |
| Queue / Dauerbetrieb | `deliver` je Scope | fehlt | R4.6 |
| Inventar mit `unknown` | `project coverage` | vorhanden | – |
| Gericht | – (`model.md:183`) | nein | verworfen |

## 6. Arbeitsweise und Betrieb

**Rollen:**
- **Der Mensch** ist der einzige Entscheider: ND1–ND11, jedes Gate, Freezes, Modellannahme, Blindprüfung, Merge, Konto, jede Löschung. Seine Akte werden als PR-Review-Approval oder als eigener Commit mit dem Protokoll- bzw. Modell-Hash festgehalten. Claude zitiert diese Akte, schreibt sie aber nie selbst („treat AI evidence as human acceptance“ ist verboten, AGENTS.md).
- **Claude** ist der einzige ausführende Agent (eine Sitzung mit Subagenten).
- **Codex** (andere Modellfamilie) nur gezielt: Autor der gesäten Fälle und Drift-Mutationen, Reviewer von S0.0, B3.1 und jeder Gate-Vorlage, Testsubjekt in MS1a, über frische `codex exec`-Läufe mit Read-only-Sandbox (Flags vorher per `--help` prüfen) oder einen vom Menschen beauftragten Chat.
- Die gestoppten Codex-Chats (Root, Integrator, KG, Scientist) bleiben gestoppt. Kein Koordinationschat, kein Heartbeat, keine ACK-Ketten.

**Status und Zustand:**
- Paketstatus nur in `follow-up-backlog.yaml`, gebündelt je Gate per Doku-PR aktualisiert.
- Arbeitsstatus und Workflow-Zustand liegen im Evidenzordner außerhalb von Git, mit festem Pfad (Vorschlag `%USERPROFILE%\markitect-evidence\refinement\`) und Hash-Manifest, nicht im sitzungsgebundenen Scratchpad.
- An jedem Gate wird der Evidenzordner an einen zweiten, vom Nutzer gewählten Ort gesichert (ND11; EX2, P10). Keine Markitect-Release-Assets, denn Releases laufen nur über den dokumentierten Prozess.

**CI, PRs, Branches:**
- Ein PR je Code-Paket; F0.1–F0.4 gebündelt in einen Doku-PR (Windows-Lauf bis 90 min, U11).
- PRs streng nacheinander: Doku-PR, dann KG-K1-Bugfix, dann ein eventuelles Read-only-Flag, später B3.1. Branches `codex/refine-<paket>` vom aktuellen Main-SHA; spätere Kandidaten werden per Merge von main nachgezogen, kein Force-Push.
- Kopf während der CI eingefroren; Merge nur durch den Menschen nach grüner CI auf beiden Betriebssystemen am exakten Kopf; Rerun nur mit benannter Hypothese, höchstens einmal; `ci.yaml` bleibt in Phase 0–2 unverändert.
- Jedes Paket beginnt mit dem Markitect-first-Einstieg (`AGENTS.md:3`) und den Gates aus `CONTRIBUTING.md:44-78`.
- Claude-Worktrees liegen außerhalb von `.codex/worktrees/*`. Fremde Branches, WIP und CO fasst Claude nicht an. Keine ZIP- oder EXE-Dateien in Git.

**Neue Dateien bis Gate 2** (ehrlich gezählt):
- 2 Planungsdateien: Entscheidungsnotiz und Government-Disposition; keine Status-Datei.
- 4 Mess-Dateien unter `docs/validation/impact-measurement/`: zwei Protokolle, zwei Berichte.
- 7 gesicherte Dokumente: 4 Concepts, Assessment, KG-Assessment, KG-Plan (nach ND8).
- Code: Messwerkzeug unter `experiments/impact-measurement/`, Tests der Code-PRs.

## 7. Konkrete Umsetzung durch Claude

**Standard-Workflow je Code-Paket** (S0.0-Bugfix, Read-only-Flag, B3.x, R4.x), als Claude-Code-Workflow mit Zustandsdatei im Evidenzordner:
1. **Spec (Hauptsitzung, Opus):** voller BASE-SHA, eigene und verbotene Pfade, Abnahmetests zuerst; `markitect check --repo . --revision BASE` und `context`.
2. **Blinder Testautor (Sonnet):** Property-, Negativ- und Golden-Tests aus der Spec, ohne `impact.go` und `plan.go` zu lesen. Sicherheitstests als Untergrenzen.
3. **Implementierer (Opus):** eigener Worktree, auf Paketpfade beschränkt.
4. **Deterministische Gates:** gezielt (`go test ./src/internal/modules/projectmodel/... -count=1`), dann volle Suite; `go vet`, Beispiel-Checks, `git diff --check`, `go run ./src/cmd/markitect-check-artifacts --repo . --config markitect-artifacts.yaml`.
5. **Review:** zwei Read-only-Reviewer (Opus, frischer Kontext; Linse 1 Korrektheit und Determinismus, Linse 2 konservative Invariante, Digest-Folgen, ein Owner). Ein Verifier (Sonnet) versucht jeden Befund zu widerlegen; nur bestätigte werden behoben.
6. **Codex-Review** bei Kernsemantik.
7. **Abschluss:** Claude pusht, öffnet den PR, friert den Kopf ein. Der Mensch merged.

**Messworkflows:**

| Schritt | Rollen (Modell) | Isolation | Prüfung |
|---|---|---|---|
| S0.1 | Kurator (Sonnet), mechanischer Filter per Werkzeug | Evidenzordner | Mensch friert ein |
| S0.2 | Modellierer (Opus) als eigener Prozess | Export-Ordner, Leseverbote, kein Web; Restrisiko im Bericht | Gegenprüfer (Opus, frisch); Mensch nimmt an |
| S0.3 | Delta-Autor (Opus, nur Intent + Modell); getrennte Pflege-Instanz (Opus); Fall- und Drift-Autor: Codex oder Mensch; rg-Autor (Opus, gleiches Budget) | getrennte Ordner; Reihenfolge Modell → Fälle; chronologisch Delta vor Pflege | Stichprobe Mensch |
| S0.6 | Werkzeug ohne LLM; Klassifizierer (Opus) und Gegenprüfer (Sonnet) je Fehlstelle | – | Mensch bestätigt; Codex-Review |
| MS1a | Protokoll (Opus), Dispatcher (Opus plus Review); Arme nur in Codex | eigenes OS-Konto, Werkzeug-Allowlist | Bewertung deterministisch; Claude höchstens sekundärer Blindprüfer |

**Was Claude nie tut:** sich selbst abnehmen, menschliche Akte protokollieren, als wären sie erfolgt, mergen, Tags oder Releases setzen, force-pushen, auf CO oder fremde Branches committen, fremde Worktrees bereinigen, als Arm in MS1a laufen, gesäte Pflichtstellen oder Drift-Mutationen schreiben.

**Berichtsform:** je Paket höchstens 10 Zeilen (Urteil zuerst, Zahlen mit Zähler und Nenner, nächste Entscheidung); je Gate eine Seite; Details auf Nachfrage.

## 8. Benötigte Nutzerentscheidungen

| # | Frage | Empfohlener Default |
|---|---|---|
| ND1 | Government: Produktidentität, optionale Schicht oder verworfen? Dürfen Agenten das Modell ändern? | Optionale Schicht 3, nur über Gate 4.x. Kabinett, Einstimmigkeit, Gericht zurückgestellt. Agenten machen bis Gate 4.3 nur Vorschläge, außer im heutigen eigenen Namespace |
| ND2 | Start trotz kollektivem Stopp? | Ja, ab Mo 12.10., sobald Main-CI `38061655290` grün ist. Gilt nur für diesen Plan, nicht für andere gestoppte Linien |
| ND3 | Testrepository und Modellumfang | MyMeetings upstream, S vor `91c8ef24`, 646-Dateien-Auswahl. Fallback an T2: nur gesäte Fälle oder Markitect-Historie |
| ND4 | Autor gesäter Fälle und Drift-Mutationen | Codex per `codex exec`; Mensch prüft 5 Stichproben. Alternative: Mensch selbst, ca. 5 h |
| ND5 | Schwellen, Kernregel, Umfang | Werte aus §4, Kernregel aus S0.1, 16 Fälle plus Drift-Arm. Ausweiten nur bei uneindeutigem Ergebnis |
| ND6 | Granularität und Pflege der Zuordnungen | Leitfaden als Hypothese, nie normativ für fremde Projekte. Bis Gate 4.3 schlägt der Agent Zuordnungen vor, der Mensch nimmt an; Ableitung aus Code nicht im Umfang |
| ND7 | MS1a: Case-Study-Sperre (`backlog.yaml:7`) für MS0/MS1a aufheben? Verhältnis zu `work-item-comparison-20261009.md`? | Aufheben nur für MS0 und MS1a, festgehalten in Backlog und Roadmap; MS1a ersetzt den dortigen Vergleichsteil; eigenes Windows-Konto, VM nur bei Fehlschlag |
| ND8 | KG-Plan und untracked Dokumente | Assessment, KG-Assessment, KG-Plan mit F0.1 sichern. KG-K0 geht in MS0 auf, KG-K1 = S0.0, KG-K3 abweichend in B3.1, Rest nach Gate 2. KG01–KG03 per Statusfortschreibung |
| ND9 | Classic-Kern (W3): „compatibility-only“ oder eigener Betrieb? | Widerspruch markieren; Entscheidung mit N2 |
| ND10 | Welche Concepts-Kopie ist nach dem Commit kanonisch? | Die auf main; Ergänzungen per kleinem PR; der CO-Ordner wird Referenz |
| ND11 | Budget und Sicherung | Bis Gate 2 höchstens 25 CT. MS1a-Läufe erst, wenn der Nutzer einen Geldbetrag als Obergrenze nennt. Absolute Prüfzeit je Work Item ≤ 20 min als Pilotgrenze. Evidenz-Sicherung an einen vom Nutzer gewählten Ort |

**Später, nicht jetzt:** B3.1 Teil 2 (Routing, Abweichung von KG-K4); B3.4 (zweiter Anbieter); verbindliche Leitplanke für Reviewer-Zuordnung.

## 9. Risiken und Gegenmaßnahmen

| Risiko | Gegenmaßnahme |
|---|---|
| Reihenfolge-Defekt verfälscht die Impact-Messung | S0.0 zuerst; holdout nur mit deterministischem Binary |
| Überanpassung von Modell oder Kernregel | Modell-Hash vor den Fällen; Kernregel und Entscheidungstabelle eingefroren; holdout genau einmal; Nachbesserung nur auf frischen Reserve-Fällen |
| Lösung sickert in Modell-Deltas | Intent nur aus Commit-Text; Delta-Autor getrennt von der Pflege-Instanz; Kandidat = nur Modell-Delta, mechanisch geprüft |
| Baseline benachteiligt | rg-Autor mit gleicher Modellstufe und gleichem Budget, iterativ |
| Breite Hülle besteht Gate 1 trotzdem | Breite ist Gate-1-Bedingung; eigener Zweig „Breite zu groß“ |
| Nur „nichts vergessen“ gemessen | Drift-Arm; C12/K8 bewusst zurückgestellt |
| Das Problem tritt in dieser Größe nicht auf | Null-Befund-Zweige an Gate 1 und 2 |
| KI-verzerrte Pflichtstellen | Replay-T mechanisch; gesäte Fälle nicht von Claude; Z5 bleibt sichtbar |
| Modellierer nicht blind | S-Export ohne Historie, eigener Prozess mit Leseverboten, Fälle nach dem Modell-Freeze; Restrisiko berichtet |
| Arm M nutzt Schreibwerkzeuge | Allowlist oder Read-only-Flag; Negativtest auf `tools/list` |
| Messapparat wird neuer Überbau (E3) | Timeboxen, Gate 0.5, gezählte Dateien, keine Grants; MS0 endet spätestens nach 10 Arbeitstagen mit Vorlage |
| Digest-Kopplung macht Pläne stale | Digest-ändernde Teile nur in S0.0-Bugfix und B3.1 Teil 2, Konsumenten benannt |
| Aussage gilt nur für einen Anbieter | B3.4; bis dahin ausdrücklich „nur Codex“ |
| Windows-CI-Schleifen | Doku gebündelt, Kopf eingefroren, `ci.yaml` unverändert, `rg` nicht in `go test` |
| Mensch hat wenig Zeit | Kontaktpunkte mit Minuten: ca. 7,5 h bis Gate 1, ca. 15,5 h bis Gate 2 |
| Rohdaten nur auf einem Rechner | Sicherung je Gate an zweiten Ort (ND11) |

## 10. Die ersten 10 Arbeitstage

Voraussetzung: ND2. Frühester Start Mo 12.10.

| Tag | Paket | Ergebnis |
|---|---|---|
| Mo 12.10. | Entscheidungsvorlage ND1–ND11 (eine Seite); Main-CI prüfen; S0.0 konstruierter Test auf Export; Upstream-Klon; `rg`-Download mit Hash; F0.1–F0.4 Entwurf | ND oder Defaults; Defekt bestätigt oder verworfen |
| Di 13.10. | Doku-PR öffnen; S0.1 Historienstatistik, Filter, Wahl von S, Eignungsprüfung; Briefing an Fall- und Drift-Autor (Start); bei Defekt KG-K1-Spec und Testautor | S fixiert, Kandidatenliste gehasht |
| Mi 14.10. | S0.2 Modellierer; KG-K1-Implementierung und Reviews; S0.5-Prototyp gegen `examples/project-world` | Modellentwurf; Smoke 9/6 |
| Do 15.10. | S0.2 konform, Gate 0.5, Modellabnahme; Freeze von Protokoll, Kernregel, Tabelle und Seed (Akt des Menschen); KG-K1-PR nach Doku-Merge öffnen | eingefrorenes Modell und Protokoll-Hash |
| Fr 16.10. | S0.3 Replay chronologisch (Deltas, Pflege, Aktualität); rg-Autor | Replay-Fälle eingefroren |
| Mo 19.10. | S0.3 gesäte Fälle und Drift-Arm übernehmen, Split, Stichprobe | Manifest-Hash |
| Di 20.10. | Puffer; deterministisches Binary (Merge oder exakter PR-Kopf); dev-Lauf | Plausibilitätsnotiz |
| Mi 21.10. | holdout genau einmal, alle Arme | Ergebnisdatei mit Hashes |
| Do 22.10. | Fehlstellen Z1–Z5, Bestätigung durch den Menschen; Gate-1-Vorlage; Codex-Review | Vorlage auf einer Seite |
| Fr 23.10. | Gate-1-Entscheidung (ca. 30 min); Status-Update im Backlog | dokumentierte Entscheidung, nächster Schritt |

## 11. Einordnung in die kanonischen Dokumente

Dieser Plan wird nicht committet. Bei Annahme ändern sich nur diese Orte, jeweils mit Freigabe des Menschen:

| Dokument | Änderung bei Annahme | Wann |
|---|---|---|
| `docs/implementation-plan.md` | höchstens 5 Zeilen: nächste Grenze nach P10, Verweis auf die Entscheidungsnotiz; W8-Nachzug von `:9` | Doku-PR |
| `docs/work-items/product-readiness/backlog.yaml` | W8: P03–P10 nach CO-Abschluss, Merge-SHA, Main-CI; Case-Study-Sperre nach ND7 | Doku-PR |
| `docs/work-items/product-readiness/follow-up-backlog.yaml` | neue Messpakete; KG01–KG03 per Statusfortschreibung | Doku-PR, dann je Gate |
| `docs/README.md` | W8-Nachzug von `:5`; Karteneintrag für Concepts und Assessment | Doku-PR |
| `docs/refinement.md` | eine Zeile Verweis, wenn B3.1 Produktentscheidung wird | B3.1 |
| `docs/design/project-world/*` | Phase 0–2 keine Änderung; B3.1: Klassen in `delivery-contract.md` | mit Umsetzung |
| `docs/project-operations.md`, `docs/project-workflow.md` | Statuszeilen → Ledger (F0.2); `--explain` und Granularitäts-Anleitung (B3.1) | je Paket |
| `CONTRIBUTING.md`, `AGENTS.md`, `risk-register.md` | W1, W4, W5, W7, Evidenz-Regel (F0.2) | Doku-PR |
| `docs/measurement.md` | unverändert; Berichte verweisen auf `:21`, `:77`, `:79`, `:81` | – |
| Government-Paket (CO) | nur Patchvorschlag an den Nutzer | nach ND1 |
| `docs/design/concepts/` und KG-Dokumente | inhaltsgleich gesichert, keine Kopfnotizen | F0.1 |

## 12. Bezug zu den Assessment-Befunden

| Befunde | adressiert durch |
|---|---|
| K1, E1, E11 (Kernthese, Recall ungemessen) | MS0, Gate 1; MS1a, Gate 2 |
| K2, KX3 (geschlossene Welt) | Z1/Z4, Aktualität per `coverage`, Arm M∪rg |
| K3, U6, A8 (Granularität, Pflegehölle) | Leitfaden, Gate 0.5, Pflege-Messung, B3.1 |
| U2, Empfehlung 7 (Breite, betroffen/Kontext) | Breite in Gate 1, M-Kern experimentell, B3.1 |
| K4, KX1, EX1 (korrelierte Prüfer, Fairness) | Nicht-Claude-Orakel, Codex-Review, gleich ausgestattete rg-Baseline |
| Kriterium 3, C17 (Drift, Regeln gelten weiter) | Drift-Arm in MS0 und Gate 1 |
| K8, C12 (Gesamtcheck) | zurückgestellt bis nach Gate 2 |
| E2, U5, K7 (Problemschwelle, Brownfield) | MyMeetings mit Replay |
| E3–E7 (Infrastruktur, Isolation, bewegliches Ziel, Asymmetrie) | Dispatcher mit Zustand, OS-Konto, Allowlist, Hash-Freeze, MS1a/MS1b-Trennung |
| EX3 (Schwellen ohne Varianz) | Pilotschwellen, 3 Wiederholungen, Vorzeichentest |
| EX2, P10 (Rohbelege privat, ein Rechner) | Sicherung je Gate (ND11) |
| U3, Risiko 9 (ein Anbieter) | B3.4 |
| K5, A5, E9, U10 (Kosten) | B3.2, B3.5, ND11 |
| K6, K13, A1–A4, A6, A7, A10, AX1, AX3 (Government) | ND1, F0.3, §5, R4.x |
| AX2 (nicht authentifizierte Annahme) | V6 vor R4.3 |
| K9 (Prüflast zum Modell) | Menschminuten der Modellabnahme, Gate 4.3 |
| K12 (Problemdiagnose) | Null-Befund-Zweige, B3.6 |
| A9, P9, U12, P1, P3, P13 (Status, Doku) | F0.1, F0.2 (W8), F0.4 |
| P2, U11 (CI-Abbrüche) | Evidenz-Regel, Kopf-Freeze, gebündelte Doku |
| P4, P5, P6 (Grants, Kontextverlust, Mensch als Regelkreis) | Timebox, Zustandsdatei, Menschminuten, Akte des Menschen |
| U1, U7, UX3, P11, U9 (Stapel, Monolith, Selbstmodell, Branches) | N1–N6 nach Gate 2; Inventur in F0.4 |
| UX1 (OS-Isolation) | S1.2 |
| PX1 (Stopp nach Main) | Gate Start, ND2 |

**Selbst geprüft** (bei der Revision, auf `f12ffb00` bzw. CO `6623cd0b`): Merge und Baumgleichheit; Push-CI `38061655290` noch `in_progress`; `.gitattributes:1`; CRLF in allen sieben Quellen; Backlog-Zeilen 7, 18, 673, 1738–1811; `impact.go:270-320`; `verify.go:259-281`; `options.go:41`; MCP-Registrierungen; `run.go:509-520`; `full_verify.go:98-110`, `:445-500`; `projectcoverage/types.go:14-22`; `rg` nicht im PATH; CO-Abschlussdatei (P06, P11–P13, `stop_boundary`); KG-Plan Zeilen 6, 227–243, 293; `experiments/agents-md-comparison/run.ps1` (1186 Zeilen); `risk-register.md` 16 Altpfad-Zeilen. **Nicht geprüft:** Eignung der MyMeetings-Historie; Codex-Tool-Allowlist je MCP-Server; Flags von `codex exec`; ob deklarierte Checks ohne LLM-Rolle ausführbar sind.

## Anhang: Gegenprüfung

| Prüfer | Punkt | Schwere | übernommen/verworfen + Grund |
|---|---|---|---|
| alle drei | Stand veraltet: PR89 gemergt (`f12ffb00`), Stopp aktiv, P06-Grenze, P11–P13 | hoch | übernommen; belegt per `git log`, CO-Abschlussdatei, `gh run view` |
| fakten, regeln | KG-K1: Doppellauf findet Defekt nicht; S0.4-Abnahmen widersprüchlich | hoch | übernommen; Code geprüft, neues S0.0 mit konstruiertem Test vor holdout |
| regeln | Routing in P3.1 nur bei `CoverageMode: full` sicher; Abweichung KG-K4 | hoch | übernommen (B3.1 Teil 2); `verify.go:259` geprüft |
| vollst., regeln | Replay-Methodik: Intent, Modellstand, Revision, Fallpool | hoch | übernommen; chronologisch Delta vor Pflege, Kandidat = Modell-Delta, ≥ 30 Kandidaten |
| vollst. | Gate 1 ohne Breiten-Bedingung | hoch | übernommen; Breite ≤ 3× und eigener Zweig |
| vollst. | rg-Baseline benachteiligt (Sonnet, 5 Begriffe) | hoch | übernommen; gleiche Modellstufe und gleiches Budget |
| vollst. | Nur halbes Nutzerziel (C17, Kriterium 3, C12/K8) | hoch | übernommen; Drift-Arm, C12/K8 begründet zurückgestellt; Check-Ausführung nur bei Toolchain |
| vollst. | Plan erzeugt Überbau, ID-Kollisionen | hoch | übernommen; 16 statt 24 Fälle, S0.4 und Golden-Test nach B3.1, ND/Gate n/B3.x/Z1–Z5 |
| fakten, regeln | Main-Status P03–P10 veraltet; zweiter Statusort; `docs/refinement.md`; KG01–KG03 | mittel | übernommen; W8, Status im Follow-up-Backlog, Statusfortschreibung |
| fakten, regeln, vollst. | Linkprüfung existiert nicht; KG-Assessment fehlt | mittel | übernommen; Wegwerf-Skript, KG-Assessment gesichert |
| fakten | Byte-Gleichheit scheitert an LF-Normalisierung | mittel | übernommen („inhaltsgleich bis auf Zeilenende“); `-text`-Eintrag verworfen, weil P12 zurückgestellt ist |
| fakten, regeln, vollst. | Kein Read-only-MCP-Modus für Arm M | mittel | übernommen; Allowlist prüfen, sonst Flag-PR, Negativtest |
| fakten | `rg` nicht installiert | mittel | übernommen; Release-Download mit SHA-256 |
| fakten, regeln | Kernregel ≠ KG-K3, unvollständig, gleicher Tag wie holdout | mittel | übernommen; Regel vollständig, Abweichungen benannt, Produktumsetzung erst B3.1 |
| regeln | Begriff „Kontext“ kollidiert mit Context-Semantik; Case-Study-Sperre und Stage-Begriffe | mittel | übernommen; Klassen `core`/`indirect`, ND7, „Messschritt“ abgegrenzt |
| regeln | Commits auf CO; Force-Push/Rebase; mehrere offene PRs | mittel | übernommen; nur Patchvorschlag, Merge statt Rebase, PRs nacheinander |
| regeln | Gate-Tabellen nicht erschöpfend; Wiederholungsregel `measurement.md:79` | mittel | übernommen; geordnete Entscheidungstabelle, 6 × 2 × 3 Läufe, Vorzeichentest |
| regeln | Evidenz in PR-Kommentaren; W3 als Hygiene; P3.1 ohne Alternativen und zweites Vokabular | mittel | übernommen; Bericht/Commit, W3 → ND9, B3.1-Vorbedingung, Granularität nicht normativ |
| regeln | Concepts nach Forschungskonvention ablegen | niedrig | teilweise: SHA-256 und Karteneintrag übernommen; Umzug nach `docs/research/` verworfen, weil Links aus Assessment und KG-Dokumenten brechen |
| regeln | PS1 statt Go, Breiten-Obergrenze, Binary-Pin, Leitplanke, rekonstruierte Gründe | niedrig | übernommen |
| vollst. | U3/Anbieterbindung, AX2/V6, Government-Kette und Kabinett-Abbildung | mittel | übernommen; B3.4, V6, R4.2-Zweig bei Produktidentität, Abbildung korrigiert |
| vollst. | Status und menschliche Akte; Isolation und Modellumfang; Rohdaten; Zeitplan; fehlende Defaults | mittel–niedrig | übernommen; Rohdaten als Release-Asset verworfen (Release-Prozess), stattdessen Sicherung nach ND11 |
| fakten | Einzelbelege: W6, N3, V3-Fundstelle, „Orphan“, Summen, 16 statt 14, Messinfrastruktur `run.ps1` | niedrig | übernommen; W6 gestrichen, Belege und Zahlen korrigiert |

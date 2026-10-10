# Unabhängiges Assessment: Markitect Government und kanonisches Modell

- **Stand:** 10.10.2026
- **Autor:** Claude (unabhängiges Assessment, Workflow mit fünf Linsen und Gegenprüfung)
- **Status:** Bewertung und Empfehlung, keine Entscheidung und keine Abnahme. Dieses Dokument ist KI-Evidenz und ersetzt keine menschliche Annahme.

**Untersuchte Stände** (lokale Refs, kein `fetch`):

| Linie | Branch | SHA |
|---|---|---|
| Koordination/Dokumentation (Checkout) | `codex/government-assessment` | `0d7f2e6e` bei Start, `469f7d9b` bei Abschluss; dazu untracked `docs/design/concepts/` |
| Classic-Hauptlinie | `origin/main` | `5be48ce1` (8.10., 22:00) |
| Government G1–G5 | `codex/government-worker` | `04e225d5` |
| Government G1–G5 + P1 | `codex/government-p1` | `839dc4e3` |
| Studien/Scientist | `codex/government-scientist` | `5ef17d73` |
| Design-Linie (linear) | `codex/model-driven-delivery` → `codex/model-first-operations` | `11f08081` → `1495e1be` |
| Produktkandidat (PR89) | `codex/product-integration-20261009` | `0deb37ca` bei Analyse, `669cecd2` aktueller Tip |
| Arbeitshistorie | `codex/markitect-historian-working-history` | `07a86d30` |

Lesehilfe: Aussagen mit Beleg (Datei:Zeile, `REF:Pfad`, Commit) sind beobachtete Tatsachen. Bewertungen sind als Urteil formuliert („Urteil:“, „spricht dafür“, „vermutlich“). Befund-IDs (K, A, E, U, P) stammen aus den fünf Linsen und stehen in Anhang A.

## 1. Kurzurteil

Das Vorhaben adressiert ein echtes Problem und hat einen tragfähigen Kern. Ein kanonisches Modell mit Typprüfung, expliziter Zuordnung zu Dateien und konservativer Impact-Analyse ist im Code real vorhanden und deterministisch prüfbar. Für die eigentlichen Wirkungsversprechen gibt es bis heute keinen gültigen Beleg: weniger vergessene Stellen, ein eingegrenzter Blast Radius, ein Vieraugenprinzip durch Subagenten, günstigere Modelle und weniger menschliche Aufsicht. Kein Markitect-Arm hat in der Vergleichsstudie einen angenommenen Kandidaten über die erste Aufgabe hinaus geliefert, und die Testfälle lagen unter der Projektgröße, ab der das Problem laut Nutzer auftritt.

Government ist als Architektur sorgfältig und als Mechanik solide gebaut, aber nur mit deterministischen Testakteuren belegt. Einstimmigkeit, Gericht und Kabinett sind Überbau ohne Nutzennachweis. Seit dem 9.10. ist Government per Nutzerauftrag kein eigenständiger Strang mehr, was das Designpaket nicht widerspiegelt. Der reale Agentenbetrieb zeigt viel Disziplin und Ehrlichkeit. Er zeigt aber auch, dass die auf den Prozess angewandte Belegführung die sichtbare Arbeit dominiert und dass der Mensch weiterhin der eigentliche Regelkreis ist.

Urteil: Als selektiver Produktkern ist das Vorhaben tragfähig. Als Versprechen einer weitgehend autonomen, vollständigen und günstigen Agentenorganisation ist es derzeit unbelegt. Vor jeder weiteren Organisationsschicht sollte ein billiger, gezielter Test der Kernthese stehen.

## 2. Was das Vorhaben will

Die eigentliche Zielbasis ist schmal. Es gibt genau eine wörtliche Nutzerbeschreibung (`docs/design/concepts/user-description-20261009.md`), und diese endet ihren Kernabsatz selbst mit „So zumindest die Hoffnung“. Die Konzepte C01–C19 (`canonical-model-and-delegated-development.md`) sind eine KI-Aufbereitung dieser Beschreibung. Der Diskussionsimpuls zur Granularität ist laut eigener Angabe „noch keine Nutzerentscheidung“ (`discussion-impulse-20261009-01.md:8`). Die Government-Ziele stammen aus einer früheren Phase am 7.10. (`government-of-coding-agents-assessment.md`). Die Spalte „Herkunft“ unterscheidet daher: **N** = Nutzerworte, **K** = Konzeptaufbereitung, **G** = Government-Paket, **I** = implizit, von diesem Assessment abgeleitet.

| # | Aussage | Art | Herkunft | Einschätzung |
|---|---|---|---|---|
| 1 | Agenten vergessen relevante Stellen, je größer das Projekt, desto mehr; Konsistenzläufe skalieren schlecht | Problemdiagnose | N P1 (`user-description:14`) | Plausibel und wichtig, aber nicht vermessen: Welche Änderungsarten wie oft zu Vergessenem führen, ist nicht erhoben (K12) |
| 2 | Eine maßgebliche Stelle (kanonisches Modell); das Repository spiegelt es; der Mensch bestimmt das Soll | Ziel/Konzept | N (Klarstellung, P2); C01, C02 | Stark für Querschnittsregeln und Verträge, riskant als Anspruch auf das ganze Repository (K3) |
| 3 | Typsystem und Compiler prüfen das Modell vor der Arbeit | Konzept | N P3; C03 | Umgesetzt (`internal/core/compile.go`). Die Compiler-Analogie überzeichnet: Ob Code das Modell realisiert, lässt sich nicht kompilieren |
| 4 | Explizite Zuordnung Modell↔Datei plus Diff grenzt den Blast Radius ein | Konzept | N P3; C04, C05 | Logischer Kern; gilt nur für deklarierte Beziehungen (K2); im Referenzbeispiel sehr breit (U2) |
| 5 | Verantwortungsbereiche und eine Managementhierarchie lassen sich aus Ordnern bzw. Namespaces ableiten | Idee | N P4; C06–C08 | In der Produktlinie umgesetzt (`nearestManager`); Querschnittspflichten passen schlecht dazu (K6) |
| 6 | Viele begrenzte Agenten erlauben deutlich günstigere Modelle | Hypothese | N P4; C10 | Nie getestet; alle Läufe nutzen ein Hochleistungsprofil (K5) |
| 7 | Viele Subagenten ergeben „quasi“ ein Vieraugenprinzip | Hypothese | N P4; C11 | Setzt unkorrelierte Fehler voraus; unbelegt (K4) |
| 8 | Ein Gesamtcheck gleicht alle Bereiche mit der kanonischen Wahrheit ab | Idee | N P4; C12 | Deterministisch nur für Struktur machbar; semantisch reproduziert er das Skalierungsproblem aus #1 (K8) |
| 9 | Cleanups und Refactorings werden zuverlässiger | Idee | N P4; C13 | Plausibel für modellierte Verpflichtungen; nicht getestet |
| 10 | Saubere, einheitliche Repositories, mehr Vertrauen in Agenten; Struktur soll belohnend sein, keine „Aufräumhölle“ | Ziel | N P5; C15, C16 | Gesamtwerturteil; nicht messbar formuliert (K10) |
| 11 | Kontinuierliche, weitgehend autonome Entwicklungsmaschine mit wenig Routineaufsicht | Ziel | G (`government-of-coding-agents-assessment.md:73ff`) | Geht über die Nutzerbeschreibung hinaus; die Umsetzung trägt es nicht (A3) |
| 12 | Kabinett wählbarer Ressorts; jede Änderung braucht die ausdrückliche Zustimmung aller ausgewählten Ressorts | Ziel (Nutzerforderung 7.10.) | G (`…assessment.md:16`) | In der neuesten Nutzerbeschreibung nicht erwähnt; ob es noch Ziel ist, ist offen (K13) |
| 13 | Agenten dürfen innerhalb expliziter Delegation das Modell weiterentwickeln | Konzept | G (`government/README.md:9`; `operating-model:155`) | Spannung zu #2 „Mensch bestimmt das Soll“, gemildert durch Pflicht zur Rückführung auf ein akzeptiertes Ziel (K9) |
| 14 | Das Modell deckt das Änderungsrelevante ab, ohne Spiegel des Codes zu werden | implizit | K (C18; Impuls, keine Nutzerentscheidung) | Zentrale offene Spannung (K3) |
| 15 | Zuordnungen bleiben über viele Änderungen aktuell | implizit | I (C04-Annahmen) | Ungemessen; die Traceability-Literatur nennt Linkpflege als Kostenfaktor |
| 16 | Prüfende Agenten sind hinreichend unabhängig von Ausführenden | implizit | I; `architecture.md:91` räumt Grenze ein | Bei gleicher Modellfamilie schwach begründet (K4) |
| 17 | Das Hauptproblem ist Vergessen (Abdeckung), nicht Fehlverständnis | implizit | I | Nicht erhoben. Liegt das Problem woanders, hilft bessere Abdeckung wenig |
| 18 | Ein Modell ist für Menschen leichter zu prüfen als Code | implizit | I (P1 „verständlich“, P2) | Gilt nur für ein kompaktes Modell, sonst verschiebt sich die Prüflast (K9) |
| 19 | Mechanische Absicherung (Digests, Leases, Journale, Recovery) ist der Engpass für vertrauenswürdige Agentenarbeit | implizit | I (Codeverteilung) | Die 27 gescheiterten A01-Versuche deuten eher auf Integration und Agentenverhalten als Engpass (U4) |
| 20 | Kleine Referenzfälle reichen, um die Wirkung zu zeigen | implizit | I (`evaluation.md:62-66`) | Widerspricht der größenabhängigen Nutzerhypothese (E2) |

## 3. Bewertung der Kernannahmen

| Annahme | Plausibilität | Belegstand | Risiko bei Irrtum | Billiger Test |
|---|---|---|---|---|
| Impact findet die betroffenen Stellen (Recall) | hoch innerhalb der deklarierten Welt | Nie gegen eine unabhängige Pflichtstellen-Liste gemessen. In MyMeetings Task 2 lieferte Impact beide direkten Subjekte (2/2), aber 69 konservativ betroffene Ressourcen (`origin/main:docs/validation/agents-md-vs-markitect.md`) | Stille Lücken bei fehlenden Zuordnungen | LLM-freier Vergleich mit vorab fixierter Liste, Baseline `rg` (Abschnitt 11) |
| Impact grenzt ein (Präzision) | mittel | In `examples/project-world` betrifft eine Beschreibungsänderung an einer öffentlichen Aussage 9 von 10 Aussagen und 6 von 6 Managern (U2, reproduziert) | Kein Fokus- und Kostenvorteil gegenüber „ganzes Modul prüfen“ | Breite des Impact je Beispiel in CI berichten |
| Zuordnungen bleiben aktuell | mittel | Ungemessen. Die Pfad-Abrechnung lässt unbekannte, verwaiste und kollidierende Pfade innerhalb konfigurierter Roots scheitern (`origin/main:docs/usage.md:552-562`) | Pflegehölle oder stille Lücke | Aktualität der Zuordnungen nach N echten Änderungen messen |
| Granularität ohne Code-Spiegel ist erreichbar | offen | Government-Fixture: 731 Modellzeilen für rund 73 Zeilen realisierte Dateien. `project-world`: 381 Modellzeilen zu 443 übrigen Zeilen (U6) | MDA-Muster: Modell wird gepflegter Code-Spiegel | Modelländerungsaufwand je Codeänderung im Pilot |
| Mehr Prüfer bedeuten mehr Abdeckung | niedrig bis mittel bei gleichem Modell | Keine Entdeckungsrate gemessen. Unabhängige Reviews wirkten auch im Conventional-Arm (E10) | Falsche Sicherheit, gemeinsame Fehlannahmen passieren alle Stufen | Gesäte Defekte, Fehlerüberlappung zweier Reviewer messen |
| Günstigere Modelle reichen | offen | Nie getestet; alle Rollen Luna High (`work-item-comparison-20261009.md:47`) | Koordinationsaufrufe fressen die Ersparnis | Gekreuzter Vergleich: durchgehend stark gegen Tiering |
| Hierarchie aus Namespaces trägt | mittel für Ownership, niedrig für Querschnitt | Real nur Tiefe 2 (A01). Im Design-Fall R12 gab es keinen erfolgreichen Manager-Rework (`assessment-2026-10-09.md:19`) | Querbezüge gehen verloren | Flach gegen rekursiv (`evaluation.md:140` sieht das vor) |
| Weniger menschliche Aufsicht | offen | Menschliche Zeit nirgends gemessen (`assessment-2026-10-09.md:45`) | Prüflast wandert vom Code zum Modell | Aktive Prüfminuten je akzeptiertem Work Item |
| Brownfield-Modell lässt sich rekonstruieren | offen | Drei v3-Brownfield-Zellen invalidiert. MyMeetings endete mit „D: inconclusive“ | Ein falsches Soll wird konsistent falsch umgesetzt | Einen Teilbereich eines echten Repositorys adoptieren, Zeit bis zum ersten Nutzen messen |

**Spannungen zwischen Konzepten** (Tatsachen mit Beleg, Gewichtung als Urteil):

- **C17 gegen C18:** C17 will vollständige Erfassung betroffener Verpflichtungen, C18 will keinen Code-Spiegel. Das Konzept führt dies selbst als offene Frage (`canonical-model…md:224-225, :239`). Urteil: Das ist die eigentliche Kernentscheidung, denn von ihr hängen Pflegekosten und Impact-Qualität ab. Erfassung (jede Datei abgerechnet) und Modelltiefe sind dabei verschiedene Achsen, und die Pfad-Abrechnung trennt sie bereits teilweise.
- **Konservative Invalidierung gegen Eingrenzung:** AGENTS.md verbietet, konservative Invalidierung zu schwächen. `docs/architecture.md:62` sagt: „Unknown or unmodelled inputs conservatively broaden results“. Gleichzeitig soll sich der Blast Radius eingrenzen. Beides zugleich ist nur möglich, wenn betroffene Menge und Kontextmenge getrennt werden (U2).
- **„Der Mensch bestimmt das Soll“ gegen delegierte Modell-Evolution** (`government/README.md:9`): Die erhoffte Entlastung entfällt, wenn Modell-Diffs so aufwendig zu prüfen sind wie Code-Diffs.
- **Hierarchie nach Bereichen gegen Querschnittspflichten:** C06 warnt selbst, dass ein technischer Namespace keine fachliche Hierarchie sein muss (`:88`). Vergessene Stellen entstehen typischerweise bereichsübergreifend.
- **Gesamtcheck gegen das Skalierungsargument aus P1** (K8).
- **Government-Rollenkonfiguration gegen die Nutzerentscheidung vom 10.10.:** Nutzer sollen Arbeit geben, statt Rechte je Rolle zu konfigurieren (`native-runtime-simplification-20261010.md:9-13`).
- **Messung mit demselben Modell wie die Gemessenen:** Ausführende und bewertende Akteure laufen mit Luna High (`work-item-comparison:47`). Die Studie teilt damit Fehlerquellen mit den Armen, die sie bewertet. Abgemildert wird das nur durch deterministische Holdouts.

## 4. Einordnung in den Stand der Technik

Die folgenden Quellen hat die Konzept-Linse abgerufen oder über Suchergebnisse gesehen. Die Gegenprüfung hat sie **nicht** nachgeprüft. Quellen aus 2026 sind nicht unabhängig verifiziert, „Nine Judges“ ist nur über eine Sekundärzusammenfassung belegt.

- **Modellgetriebene Entwicklung:** Erfolgreiche MDE-Praxis modelliert selten ganze Systeme. Sie modelliert Kernteile, oft mit DSLs, und hängt stark an organisatorischen Faktoren (Hutchinson/Whittle, https://eprints.lancs.ac.uk/id/eprint/69765). Urteil: Das stützt den selektiven Kern und warnt vor dem Vollständigkeitsanspruch.
- **Spec-driven Development:** Laut Böckeler/Fowler könnten „spec-as-source“-Ansätze „the downsides of both MDD and LLMs“ erben; Spec-Dateien seien „very verbose and tedious to review“ (https://www.martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html). Ähnlich äußert sich das Thoughtworks Radar (https://www.thoughtworks.com/radar/techniques/spec-driven-development). Relevant für K3 (Granularität) und K9 (Prüflastverschiebung).
- **Traceability:** In einer Mapping-Studie nennen etwa 30 % von 63 Primärstudien Erstellung und Pflege von Trace-Links als Kostenfaktor (https://arxiv.org/pdf/2108.02133). Relevant für die Zuordnungspflege (C04).
- **Deterministische Architekturprüfung:** Fitness Functions bzw. ArchUnit-artige Tests sind eine etablierte, billige Alternative für Teile des Gesamtchecks (https://concepts.dsebastien.net/concept/architectural-fitness-functions/). Ein Vergleich dieser Mittel mit Markitect-Komponenten fehlt (K12).
- **Multi-Agent-Systeme:**
  - Mehrfacher Tokenverbrauch (≈ 15× gegenüber Chat); geeignet vor allem für parallelisierbare Breitenaufgaben (https://www.anthropic.com/engineering/multi-agent-research-system).
  - Parallele Subagenten treffen widersprüchliche implizite Entscheidungen (https://cognition.ai/blog/dont-build-multi-agents).
  - Ein Reviewer mit sauberem Kontext hilft, und Routing über Modellfamilien hinweg brachte Gewinne. Ein schwächeres Primärmodell setzt die Qualitätsobergrenze (https://cognition.com/blog/multi-agents-working, 2026).
  - Fehler entstehen vor allem durch Systemdesign, Fehlabstimmung zwischen Agenten und schwache Verifikation (MAST, https://arxiv.org/abs/2503.13657v2).
- **LLM als Prüfer:** Die Fehlerähnlichkeit steigt mit der Fähigkeit, und LLM-Judges bevorzugen ähnliche Modelle (Goel et al. 2025, https://arxiv.org/pdf/2502.04313). Eine Folgearbeit (nur sekundär gesehen) schätzt neun Frontier-Judges auf etwa zwei effektive Stimmen (https://huggingface.co/papers/2605.29800).

Einordnung (Urteil): Die Literatur spricht gegen die Nebenhypothesen „viele gleiche Agenten = Vieraugenprinzip“ und „Zerlegung macht billig“, belegt aber kein Gegenteil für Markitect. Den deterministischen Kern (explizites Modell, Ownership, konservativer Impact als Eingang für Agenten) trifft sie nicht direkt. Eine Recherche zu direkt vergleichbaren Werkzeugen (z. B. OpenSpec, Kiro) wurde nicht durchgeführt.

## 5. Government-Architektur

**Was trägt** (Tatsachen aus `docs/design/government/architecture.md` und `codex/government-worker:docs/government.md`):

- Getrennte Identitäten und Wahrheitsebenen: Verantwortung, Agenteninstanz, Artefakt, Bericht und Urteil sind getrennt. Laufzustand ändert kein Gesetz (`architecture.md:15, :49-55`).
- Ein vollständiges Inventar mit sichtbarem „unknown“: „Unbekannt ist kein Pass“ (`:79-83`). Das G1-Beispiel endet bewusst mit Exit 1 bei unzugeordneten Dateien.
- n:m-Realisierung, genau ein Writer je Pfad, ein konservativer Plan über explizite Referenzen in beide Richtungen (`government.md:30`).
- Rekursive Bereiche mit Elternintegration. Das G3-Beispiel erkennt einen Kompositionsfehler trotz grüner Kinder (`government.md:69-75`).
- Eine zyklenfreie Bindungskette Kandidat → Evidenz → Entscheidung, CAS-Promotion, Fencing und Recovery ohne Replay unbekannter Effekte.
- Schutz gegen Selbstermächtigung: Nur die Vorversion bestimmt Befugnis (`architecture.md:109-115`). Einschränkung: Die Verfassung wird als „bereits akzeptierte“ Eingabe gebunden, die G1 nicht authentifizieren kann (`government.md:18`), und einen Owner-Kanal gibt es nicht. Der Schutz wirkt gegenüber Agenten im Lauf, nicht gegenüber dem Aufrufer.

**Was Überbau ist:**

- **Einstimmigkeit (A2):** Jedes eingefrorene Ressort muss ausdrücklich zustimmen, auch mit „assent-unaffected“. Jede neue Evidenz erzwingt eine neue Runde (`architecture.md:119-127`). Der Host prüft nur Form, Bindung und Entscheidungscode, und ein nicht leerer `reason` genügt (`government.md:48`). Urteil: Das verhindert stille Zustimmung, nicht falsche. Billiges Durchwinken und Blockade durch Rauschen sind plausible, aber ungemessene Risiken, denn alle Nachweise stammen von deterministischen Akteuren.
- **Gericht und Gewaltenteilungsrollen (A6):** §7 beschreibt ein Gerichtsverfahren, die Herkunft sah als Erstform keines vor (`operating-model:169-177`). Nicht implementiert (`government.md:85, :98`). Abnahmekriterium 7 aus §15 ist damit nicht belegt. Exekutive und Legislative werden nur einmal genannt (`architecture.md:35`).
- **Queue ohne Dauerbetrieb (A3):** Jeder Job bindet ein festes `expectedBase`. Ist die Active-Ref weitergelaufen, gilt „issue a fresh order and runtime binding“ (`codex/government-worker:internal/host/government/execution/queue.go:322-329`). Orders laufen seriell, Amendments dürfen nichts hinzufügen, Eskalationen haben keinen Kanal. Die versprochene kontinuierliche Entwicklungsmaschine trägt die Umsetzung damit nicht.
- **Konfigurationslast (A4):** eigene Runtime-Slots je Rolle (command, model, providerVersion, Timeouts), viele Zahlenlimits, rund 9.200 Zeilen Go ohne Tests (Paket plus CLI). Die Architektur fordert selbst einen „kleinen“ Begriffsbestand (`:19`).
- **Kosten nicht adressiert (A5):** Modellkosten und Modellwahl je Rolle kommen im Paket nicht vor. Die Mechanik lässt eher mehr Aufrufe erwarten (Executor und Verifier je Knoten, Root-Review, Voten aller Ressorts, Wiederholung je Reparatur). Gemessen ist das nicht.
- **Architektur nie nachgeführt (A7):** Einziger Commit `09850535` (7.10., 16:03). Nicht umgesetzt sind die Kontext-Nachforderung (§8, wichtig für C19), das Gericht, `accepted-complete` (es gibt nur `accepted-scoped`) und delegierte Organisationsänderungen.

**Vorgeschlagener Minimalkern** als Anforderung an die Design-Linie (Urteil, abgeleitet aus A4):

1. ein kanonisches Modell mit `purpose`,
2. n:m-Realisierung und ein Writer je Pfad,
3. ein Inventar mit sichtbarem `unknown`,
4. ein konservativer Plan mit dem Status jeder betroffenen Pflicht,
5. rekursive Bereiche mit Elternintegration,
6. ein unabhängiger Reviewer je Bereich plus vollständiges Verify,
7. ein gebundener Kandidat, die Ablehnung veralteter Stände, eine geschützte Übernahme und Wiederaufnahme ohne Replay.

Nur bei gemessenem Bedarf kommen hinzu: Querschnitts-Ressorts als zusätzliche Reviewer, mandatierte Modellpflege und eine persistente Queue. Nach `deferred-research.md` verschoben werden: Gericht, Gewaltenteilungsrollen, Pflicht-Vollvotum je Runde und das Verfassungs- und Kabinettsvokabular.

**Aktuelle Produktrichtung** (Tatsachen):

- Seit dem direkten Nutzerauftrag vom 9.10., 06:07 ist Design die Hauptlinie (`coordination.md:1250`; `work-item-comparison-20261009.md:3, :9`). Ein eigenständiger Government-Vergleich oder -Ausbau entfällt (`:87`).
- „Stillgelegt“ wäre trotzdem zu stark. Design ist beauftragt, „neue Government-Verfeinerungen … auf sinnvolle Angliederung“ zu prüfen (`work-item-comparison:9`), und `operation-scopes-and-model-briefings.md:13` auf dem Produktkandidaten nimmt die Government-Idee ausdrücklich auf.
- Eine inhaltliche Begründung der Entscheidung ist nicht dokumentiert.
- Das Paket ist nicht nachgeführt. Nur `evaluation.md:3` trägt den Hinweis. `architecture.md:5, :290` und `delivery-plan.md` beschreiben den Vor-Go-Stand. Der von AGENTS.md vorgegebene Einstieg `docs/README.md:12` nennt das Paket weiter „Current experimental Government design“.
- Im Produktkandidaten verweist keine Go-Datei auf Government.
- Die Begriffe unterscheiden sich: Government spricht von Verfassung, Ressort, Kabinett und Gericht, Design von Manager, Modul, Geschäftsführer und subsidiärer Eskalation. Eine Begriffsabbildung für die Angliederung fehlt (A10).

## 6. Evidenzlage

| Hypothese | Status | Beleg |
|---|---|---|
| Markitect (Modell + Agenten) führt zu weniger vergessenen Stellen als ein starker konventioneller Ablauf | unbelegt | Kein gültiger Vergleich (`assessment-2026-10-09.md:9`). Alle Markitect-Arme endeten vor oder an der ersten Aufgabe bzw. Station (E1) |
| Der Blast Radius grenzt sich ein | teilweise | Die Mechanik existiert deterministisch, die Präzision ist ungemessen; im Referenzbeispiel sehr breit (U2) |
| Das Modell ist vor der Arbeit strukturell prüfbar | belegt | `core.Compile`; `markitect project check` auf `project-world` erfolgreich (U, selbst ausgeführt) |
| Subagenten bilden ein wirksames Vieraugenprinzip | unbelegt für Markitect | Keine Entdeckungsrate gemessen. Im Conventional-Arm fand ein Review eine Regression (E10) |
| Günstigere Modelle reichen | unbelegt | Nie getestet; Tokens, Kosten und menschliche Zeit in Studien unbekannt (E9, U10) |
| Weniger menschliche Aufsicht | unbelegt | Nirgends gemessen; im eigenen Betrieb gab es etwa 24 direkte Nutzereingriffe (P6) |
| Die Government-Mechanik (Mandate, Rekursion, Voten, Promotion, Queue/Resume) funktioniert | belegt, nur mit deterministischen Akteuren | `codex/government-worker:docs/government.md:54, :75, :89, :108`; Build, Vet und Tests grün (U7) |
| Einstimmigkeit der Ressorts verbessert die Qualität | unbelegt | Keine vollständige Votenrunde mit realen Modellen (E8) |
| Wiederaufnahme ohne Replay | teilweise | G5 deterministisch; A01-Original-Turn-Recovery in Actual28 (`coordination.md:2319-2333`) |
| Die rekursive Managerhierarchie trägt | teilweise | A01: Root, Docs und Source mit Helper und Integrationsreview. Tiefe über 2 nicht belegt; R12 ohne erfolgreichen Manager-Rework |
| Die Wirkung tritt bei Bestandsprojekten (Brownfield) auf | unbelegt | v3-Brownfield-Zellen invalidiert (`coordination.md:1029-1033`); MyMeetings „D: inconclusive“ |
| Modellpflege lohnt sich (C16) | unbelegt | Indikatoren eher ungünstig (U6); keine Messung an einem realen Repository |
| Das Produkt läuft Ende-zu-Ende | teilweise | A01 im 28. Versuch an einem Projekt mit fünf Dateien (Ledger :2170-2290). PR89: Linux grün; Windows scheiterte auf `0deb37ca` am Onboarding-Replay, Neulauf auf `669cecd2` laut Akte laufend |
| Ein fairer Vergleich gelingt in wenigen Stunden auf einem gemeinsamen Windows-Host mit LLM-Studienleitung | widerlegt im gegebenen Setup | Kooperative Isolation, Kontextverlust der Studienleitung, Runner-Probleme (E3–E5) |

Zur Fairness gegenüber Conventional (Prüfernachtrag EX1/EX2): Die stärkste Conventional-Evidenz (46 öffentliche Checks PASS) stammt aus einer nachträglichen Prüfung eines nach Fristablauf reparierten Kandidaten. Classic und Government erhielten diese Chance nicht. Rohbelege und Oracles sind privat. Belastbar ist daher nur der Readiness-Befund, dass die Markitect-Arme ihre Aufgaben nicht erreichten. Dass Markitect die Lieferfähigkeit gesenkt hätte, ist kausal nicht belegt.

Ebenfalls zu korrigieren ist eine verbreitete Lesart: Das Protokoll kannte durchaus eine Metrik für Vergessenes. Es berichtet „erkannte und verpasste Pflichten“ primär (`evaluation.md:163`). In den öffentlichen Unterlagen fehlt aber eine vorab fixierte Pflichtstellen-Liste, und es gab weder eine Größenvariation noch einen Arm mit günstigerem Modell.

## 7. Umsetzung im Code

**Abdeckung C01–C19** (Matrix der Umsetzungs-Linie; Stichproben in der Gegenprüfung bestätigt, nicht jede Zeile einzeln):

| Status | Konzepte |
|---|---|
| strukturell und deterministisch implementiert | C01, C02 (strukturell), C03, C04, C06, C07, C09, C11 (mechanisch), C12 (für den modellierten Umfang), C14 |
| implementiert, Präzision ungemessen, im Beispiel sehr breit | C05 |
| teilweise | C08 (real Tiefe 2), C13 (Cleanup/Reconcile nur als Prompt-Mandat), C15, C17, C18, C19 (Kontext-Nachforderung aus Architektur §8 fehlt) |
| nur Konfiguration oder Hypothese | C10 (`Agent.Model`/`Pricing` ohne Messung), C16 |

Die Kernmechanik liegt **nicht** in der Government-Linie. Sie liegt im Classic-Kern (`internal/core/compile.go`) und im kompakten Modul `src/internal/modules/projectmodel` (1.297 Zeilen) des Produktkandidaten. Dort wird Ownership wie vom Nutzer gewünscht aus dem Namespace abgeleitet (`analyze.go:331`).

**Komplexität** (Zeilen gezählt, Linse U und Prüfer):

- Government `internal/host/government` auf `codex/government-p1`: 9.000 Produktions- und 4.856 Testzeilen, entstanden am 7.10. zwischen etwa 16:36 und 22:59.
- Produktkandidat:
  - `projectrun`: 16.576 Produktions- und 12.787 Testzeilen,
  - `projectadoption`: 5.642,
  - `codexappserver`: 3.005,
  - `projectwork`: 2.221,
  - `projectcli`: 2.029,
  - `projectbriefing`: 1.630,
  - `projectcoverage`: 1.609,
  - `projectmodel`: 1.297.
- `git diff origin/main codex/product-integration-20261009 -- '*.go'`: 665 Dateien, +73.450/−2.045.
- Urteil: Der Großteil fließt in Ausführung, Bindungen, Journale und Recovery. Die genaue Quote hängt an der Klassifikation, etwa ob `projectbriefing` und `projectadoption` als Hebel zählen. Brownfield hat viel Code, aber keine Validierung an einem realen Bestand.

**Lücken:**

- **Impact-Hülle (U2):** `impact.go:308-319` zieht für jedes Mitglied der Hülle alle direkten Konsumenten nach, auch für unveränderte, nur über `uses`/`requires` erreichte Aussagen. Das ist gewollt und getestet (`TestImpactFollowsTransitiveConsumers`). In einem dicht verbundenen Modell ergibt es aber nahezu Vollläufe. Zehn Aussagen widerlegen C05 nicht, zeigen aber, dass Eingrenzung ohne getrennte Kontextmenge kaum entsteht.
- **Echte LLM-Ausführung (U3):** belegt nur über einen Anbieter (Transport `process` und `codex-app-server`, `projectrun/types.go:96-99`). Der Claude-Adapter läuft ohne Tools und mit einem Turn. Alle Läufe betrafen kleine Spielfälle: A01, R12, die adaptiven Shop-Versuche.
- **Größe und Brownfield (U5):** Kein gemessener Lauf an einem mittelgroßen echten Repository. Markitect selbst nutzt für sich das Classic-Format und nicht `.markitect/project.yaml`, obwohl es der naheliegende Realtest wäre (UX3).
- **Keine OS-Isolation (UX1):** Leases und Digests schützen Konsistenz, nicht den Zugriff auf Zugangsdaten im selben Benutzerkonto (`clauderunner/README.md`; `government.md:113`).
- **Kostenerfassung partiell (U10):** Helper- und Kindstarts werden nicht aggregiert (`projectrun/cost.go:24-26`). `Pricing` ist eine Schätzung, keine Rechnung.
- **Lesbarkeit (U7):** `runOrResume` hat etwa 1.103 Zeilen (`run.go:46`).
- **Widersprüchlicher Status (U12):** Die Nutzerdokus melden „ATTEMPTED, NOT PASSED“ mit widersprüchlichen Details (`docs/project-workflow.md:37`, `docs/project-operations.md:43`). Das Ledger führt für Versuch 28 `a01_acceptance: passed`.

**Divergenz der Linien** (U1, U9, P11):

- Es gibt drei getrennte Ausführungsstapel: den Classic-Controller (auch im Produktkandidaten), Government G2–G5 (`codex/government-worker`, `codex/government-p1`) und `projectrun`.
- `codex/government-p1` liegt 20 Commits vor und 277 hinter dem Produktkandidaten und nutzt das alte Layout `internal/` statt `src/internal/`.
- Der Produktkandidat liegt 248 Commits vor `origin/main` und ist nicht gemergt.
- `model-driven-delivery` ⊂ `model-first-operations` ⊂ `product-integration` sind linear. Die Teilbranches app-server, mcp, workspaces und zwei Fix-Branches sind patch-äquivalent konsolidiert, was eine Stärke ist.
- Es gibt 327 Branch-Refs (223 lokale `codex/*`) und 17 Worktrees. Der Scientist-Branch bringt rund 889k Zeilen mit, überwiegend Evidenz, darunter 65 ZIP-Dateien (113 MB) und 4 EXE-Dateien (68 MB) in Git.

## 8. Arbeitsprozess und Steuerbarkeit

| Kennzahl | Wert | Beleg |
|---|---|---|
| Commits des Koordinationsbranches vor `origin/main` | 408 (7.10.: 62, 8.10.: 84, 9.10.: 147, 10.10.: >112) | `git rev-list --count origin/main..HEAD` |
| Davon seit 30.9. reine Änderungen an `coordination.md`/`-state.json` | 333 von 429 Nicht-Merge-Commits (gewollt reine Doku-Branch) | Linse P, bestätigt |
| `coordination.md` | 2.381 Zeilen, ≈ 535 KB; Kopf weiter „Stand: 2026-10-07“ | Datei, Z. 3 |
| Lesbarkeitsverfall | Anteil Tokens über 40 Zeichen: früh ≈ 1 %, spät ≈ 12–15 %; ab 9.10. 05:13Z überwiegend Englisch | Messung P und Prüfer |
| `coordination-state.json` | 1,69 MB; Scientist-Eintrag ≈ 830 KB (≈ 49 %) | Skriptauswertung |
| Grant-/Receipt-JSON im Government-Ordner | 46 Dateien | `ls` |
| Dokumentierte Kontextverluste von Agenten | ≥ 5 (Z. 167, 271, 309, 477, 1196), einer kostete die v4-Conventional-Zelle | `coordination.md` |
| Direkte Nutzereingriffe | ≈ 24 Abschnitte (Heuristik 33 inkl. Folgeeinträge), davon ≈ 7 Lockerungen; der Rest setzt Richtung oder erweitert den Umfang | Linse P, Prüfer |
| A01-Main-Smoke | 28 Versuche, 9.10. 20:26Z bis 10.10. 12:06Z (inkl. Hold und Unterbrechung); 184 Starts, davon 180 Provider; Kosten-Untergrenze 25.055.209 Mikroeinheiten (Einheit nicht angegeben, keine Rechnung) | Ledger :22-31, :2170-2290 |
| Doku-artige Commits auf dem Produktbranch am 10.10. | ≈ 46 von 83 | Prüfer P3 |
| Topologie | Stern: Root plus 9 Thread-Einträge; Rekursion nur im A01-Smoke | `coordination-state.json` |

**Was der reale Agentenbetrieb über die Hypothese lehrt** (Urteil):

1. **Kein Test des Government-Produkts.** Betrieben wurde Chat plus Markdown in Sternform, nicht der deterministische Host. Der Betrieb stützt die Government-Hypothese weder, noch widerlegt er sie.
2. **Explizite Artefakte retten Arbeit, implizite Chat-Kontinuität nicht.** Handoff, Cursor und Neu-Dispatch reparierten die meisten Kontextverluste. Erkannt wurden sie aber nur per Polling, und der Resume-Zustand wächst selbst zum Kontextproblem (E4, P5).
3. **Der Mensch ist der Regelkreis.** Not-Aus (9.10., 13:11Z), Reflexions-Hold, Budgetfreigabe („Budget darfst du solange machen bis es klappt“) und Vereinfachungen brachten messbaren Fortschritt. Urteil: Das Ausgangsdesign war überparametrisiert.
4. **Reviews fangen echte Fehler vor der Ausführung,** etwa den Zählfehler 7 statt 6 Starts (Z. 331) und den Fehler der Versuchshülle (Z. 449). Teils waren das Fehler des Verfahrens selbst.
5. **Rigor auf den Prozess angewandt wird teuer und erzeugt eigene Fehlerklassen.** Ein Beispiel: Auf PR-Köpfe committete Evidenz löste mit `cancel-in-progress` CI-Abbrüche aus. Root hatte die Windows-Abbrüche vermerkt (Z. 2345, 2349). Die Verbindung zu den eigenen Evidenz-Commits kam mit einem externen Hinweis des Nutzers (Z. 2363-2367). Nach dem Head-Freeze lief Windows durch und fand einen echten Defekt.
6. **Manuelle Synchronisation, die Markitect abschaffen will, findet im eigenen Betrieb statt.** Meilensteine stehen in mehreren Dateien (Akte, State, Fortschrittsdoku, Ledger, Backlog), und die Statusdoku widerspricht dem Ledger (U12). Darin liegt eine gute Gelegenheit zum Dogfooding.
7. **Die wichtigste Messgröße des Vorhabens fehlt im eigenen Datensatz:** die menschliche Aufsichtszeit. „Engineering-, Koordinations- und Reviewaufwand wurde nicht instrumentiert“ (Z. 453).

Neueste Lage (Tatsache, `coordination.md:2377-2380`, Commit `469f7d9b`): Der Nutzer hat am 10.10. entschieden, dass Budgets für die notwendige Stabilisierung frei sind und A01 ohne konkreten Defekt nicht wiederholt wird. Der Integrator schließt PR89/Main ab, danach folgt ein kollektiver Stopp ohne automatische Folgearbeiten.

## 9. Stärken, die erhalten werden sollten

- **Ehrlichkeit der Dokumentation:** Absicht, Hypothese, Interpretation und Gegenbeleg sind getrennt (C01–C19 mit „Grenzen und Gegenbelege“). Unbekanntes wird nicht als null gezählt (`assessment-2026-10-08.md:26-31`), und Fehlschläge bleiben ohne Schönfärberei erhalten.
- **Ein faires, falsifizierbares Studiendesign:** Conventional „muss gewinnen können“ (`evaluation.md:11`), Modellpflege zählt als Kosten (`:19`), und die Referenz ist stark ausgestattet statt ein Strohmann. Die Work-Item-Szenarien mit abschließender Umbenennung zielen auf die richtige Frage.
- **Konservative Impact-Semantik und sichtbares `unknown`** in allen Linien. Dazu gehört die Pfad-Abrechnung als deterministischer Mechanismus gegen vergessene Zuordnungen (`usage.md:552-562`).
- **Ein echter deterministischer Modell-Compiler** als gemeinsame Basis und ein Produktmodell (`projectmodel`), das das Nutzerkonzept sehr direkt abbildet (Manager, Statement, Artifact, Check, Decision; Ownership per Namespace).
- **Saubere Government-Grundbegriffe:** Trennung von Soll, Ist, Laufzustand und Befugnis, Elternintegration, gebundene Kandidaten und robuste Übernahme. Als Anforderungen an Design sind sie wertvoll.
- **Technisch solider Code:** Build, Vet und Government-Tests sind grün, von Linse und Prüfer selbst ausgeführt. Ein echter Ende-zu-Ende-Lauf existiert (A01, Actual28).
- **Funktionierende Prozessbausteine:** Not-Aus und Hold, kurze Entscheidungsdokumente (2–3 KB), Abschlussberichte mit der Entscheidung zuerst, begrenzte Pakete mit schneller Abnahme (G1–G5 an einem Abend) und die Schreibtrennung zwischen Koordination und Produktquelle.

## 10. Zentrale Risiken (priorisiert)

1. **Die Kernthese bleibt ungetestet, während weiter gebaut wird** (K1, E1; kritisch). Alle Investitionen in Hierarchie, Ressorts und Runtime ruhen auf einer unbestätigten Prämisse.
2. **Impact liefert entweder stille Lücken oder zu breite Mengen** (K2, U2, E11; hoch). In der geschlossenen Welt deklarierter Zuordnungen bleibt Fehlendes unsichtbar. Mit konservativer Verbreiterung schwindet der Eingrenzungsnutzen.
3. **Die Modellpflege wird zur Pflegehölle** (K3, A8, U6; hoch). Ohne festgelegte Granularität ist der Pflegeaufwand nicht planbar. Die Government-Form ist hier besonders ungünstig.
4. **Testfälle unter der Problemschwelle** (E2, U5, K7; hoch). Größenabhängigkeit und Brownfield sind ungeprüft, also genau das Einsatzgebiet.
5. **Kosten- und Aufsichtsversprechen kehren sich um** (K5, A5, E9, U10, K9; hoch). Mehr Aufrufe, Prüflast am Modell statt am Code, nichts davon gemessen.
6. **Formalisierung verdrängt Erkenntnis, der Prozess ist schwer steuerbar** (P1, P3, P4, E3, E6; hoch). Die Akte ist nicht mehr lesbar, das Vergleichsziel wechselte, und Belegführung dominiert die sichtbare Aktivität.
7. **Korrelierte Prüfer erzeugen falsche Sicherheit** (K4, KX1; mittel). Das gilt auch für die Studienbewertung selbst.
8. **Veraltete Doku führt zu falschen Annahmen über die Produktrichtung** (A1, AX3, A7, A9; mittel). Dazu kommen ungesicherte Nutzerziele: `docs/design/concepts/` ist untracked und auf keinem Branch.
9. **Anbieterbindung und fehlende OS-Isolation** (U3, UX1; mittel).

## 11. Empfehlungen

**Sofort** (im Rahmen der beschlossenen Main-Stabilisierung, ohne neue Features):

1. PR89 wie entschieden stabil nach Main bringen und den kollektiven Stopp halten. Bis zum Kernthesen-Test keine weiteren Organisationsschichten (Ressorts, tiefere Rekursion).
2. Das Government-Paket als historisch kennzeichnen: Statusblock in `README.md`, `architecture.md` und `delivery-plan.md`, Korrektur von `docs/README.md:12`. Dazu eine kurze Entscheidungsnotiz mit Gründen, in Design übernommenen Mechanismen, zurückgestellten Teilen und einem einheitlichen Statusbegriff.
3. `docs/design/concepts/` unter Concepts-Eigentümerschaft committen, damit die Nutzerziele gesichert und für die Produktlinie sichtbar sind.
4. Die Regel „keine Evidenz-Commits auf PR-Köpfen“ in CONTRIBUTING aufnehmen. Als Einstieg für Menschen eine `STATUS.md` mit höchstens 40 Zeilen; die Chronik wird nur noch Archiv.
5. Den Abnahmestatus nur im Ledger führen und die Nutzerdokus darauf verweisen lassen (U12).

**Nächste zwei Wochen:**

6. **Stufe 0 des Experiments** (unten) ohne LLM durchführen.
7. **Die Impact-Semantik als Designentscheidung schärfen.** Betroffene Menge (geänderte Definitionen, ihre Realisierungen, direkte Konsumenten geänderter öffentlicher Verträge) und lesende Kontextmenge getrennt ausweisen. Das muss ausdrücklich gegen die AGENTS.md-Regel zur konservativen Invalidierung abgewogen werden. Die Breite des Impact je Beispiel in CI berichten.
8. **Eine Granularitätsregel festlegen.** Das Modell besitzt Begriffe, bereichsübergreifende Invarianten, Schnittstellenverträge, Ownership und Artefaktgruppen. Alles darunter wird aus dem Code abgeleitet. Ein Diff auf eine unzugeordnete Datei bleibt ein harter Befund.
9. **Die Kostenerfassung vervollständigen:** Helper- und Kind-Usage aggregieren, Abgleich mit Usage-Exporten des Anbieters.
10. **Stufe 1 vorbereiten:** Harness als Skript, einen getaggten Produktstand einfrieren, OS-Isolation je Lauf (eigener Benutzer oder VM), ein einziger festgeschriebener Budgetrahmen statt Grant-Zyklen.

**Später** (erst nach Stufe 1):

11. Bei positivem Ergebnis: Modell-Tiering (C10) als gekreuzten Vergleich, Reviewer einer anderen Modellfamilie, Ressortreview gegen allgemeinen Review bei gleichem Budget als Ein-Faktor-Frage, Rekursion über zwei Ebenen nur bei gemessener Überlastung.
12. Nur noch einen Ausführungsstapel pflegen. `codex/government-p1` als Referenz einfrieren und überlegene Mechanismen (Fencing-Lease, Promotion-Recovery) als einzelne Issues gegen `projectrun` formulieren. `runOrResume` in benannte Stufen zerlegen.
13. Aufräumen: gemergte Branches und Worktrees entfernen, Binärevidenz in ein Artefaktlager mit Hash-Manifest auslagern, Koordination aus dem Government-Ordner an einen neutralen Ort verlegen.

**Kleinstes aussagekräftiges Experiment** (Entwurf):

- **Stufe 0, ohne LLM (1–2 Tage):**
  - Gegenstand: das bereits eingefrorene MyMeetings-Subset (646 Dateien), ein eigenes Repository des Nutzers oder Markitect selbst.
  - Zehn vorab definierte Änderungen: fachliche Umbenennung, Regeländerung, Vertragsänderung. Für jede Änderung legt ein unabhängiger Autor vorab die Pflichtstellen und die Nicht-Ändern-Stellen fest, eingefroren per Hash.
  - Verglichen wird `markitect project impact` mit einer Identifier-Suche per `rg`.
  - Metriken: Recall, Breite relativ zu den Pflichtstellen, Modellierungsaufwand (Stunden, Modellzeilen).
  - Weiter nur, wenn der Recall mindestens die Baseline erreicht und die Breite höchstens das Dreifache der Pflichtstellen beträgt. Sonst zuerst Modell, Granularität oder Impact-Semantik verbessern.
- **Stufe 1, gepaart mit Agenten:**
  - Acht Änderungen. Conventional (gutes AGENTS.md, Tests, Subagents, Reviewer) gegen Markitect mit angenommenem Modell, beide mit gleichem Modell. Die Reihenfolge wird randomisiert, jede Änderung zählt als eigene Einheit.
  - Primärmetrik: verpasste Pflichtstellen je Änderung gegen die eingefrorene Liste.
  - Sekundärmetriken: unzulässige Änderungen, Holdout-Tests, Tokens aus CLI-Usage-Receipts, Minuten eines blinden menschlichen Reviews, Anteil abgeschlossener Änderungen (Readiness getrennt berichtet).
  - Kostenregel: Initiale Modellvorbereitung wird separat ausgewiesen, zählt in der Gesamtbilanz aber mit. Modellpflege während der Arbeit zählt zum laufenden Aufwand (`work-item-comparison-20261009.md`).
  - Die Bewertung erfolgt deterministisch oder durch einen Bewerter einer anderen Modellfamilie. Die Studienleitung ist ein Skript, kein LLM.
  - Abbruch:
    - Schließt der Markitect-Arm mehr als 2 von 8 Änderungen wegen Produktfehlern nicht ab, ist das ein Readiness-Befund und die Stufe wird gestoppt.
    - Zeigt Markitect nach acht Paaren keine Verbesserung bei den verpassten Stellen und zugleich höhere Kosten oder Prüfzeit, bleibt Markitect ein selektives Hilfswerkzeug ohne Ausbau der Agentenorganisation.
  - Hinweis: Die Schwellen sind Vorschläge dieses Assessments und nicht aus Varianzdaten abgeleitet. `evaluation.md` verlangt, Wiederholungszahlen nicht ohne Grundlage zu erfinden. Stufe 1 ist daher zugleich der Pilot für die Varianzschätzung einer späteren Replikation.

**Messbare Erfolgskriterien** (vorab einfrieren, gegen die Conventional-Referenz; Schwellen als Vorschlag):

1. Recall betroffener Stellen bei gesäten Umbenennungen und Regeländerungen ≥ 95 % und nicht schlechter als die Referenz.
2. Breite des berechneten Wirkungsbereichs ≤ 3× die tatsächlich nötigen Stellen.
3. Erkennung gesäter Drift ≥ 90 % bei ≤ 10 % Fehlalarmen.
4. Aktive menschliche Prüfminuten je akzeptiertem Work Item mindestens 30 % unter der Referenz, bei gleicher Fehlerrate.
5. Modellpflege ≤ 20 % des Änderungsaufwands, Aktualität der Zuordnungen ≥ 95 % nach N Änderungen.
6. Kosten je akzeptiertem Work Item ≤ Referenz.

## 12. Offene Fragen an den Nutzer

1. Sind Ressorts und Einstimmigkeit (Nutzerforderung vom 7.10.) unter der Design-Linie noch Ziel, eine optionale Schicht oder verworfen?
2. Welche Granularität soll das Modell verbindlich haben (Begriffe, Invarianten, Verträge, Artefaktgruppen gegenüber Datei- oder Symbolebene), und wer pflegt die Zuordnungen: Mensch, Agent oder Ableitung aus dem Code?
3. Dürfen Agenten das Modell innerhalb einer Delegation ändern, oder nur Vorschläge machen, bis gemessen ist, dass Modell-Review billiger ist als Code-Review?
4. Welches mittelgroße Repository soll als Testfall dienen: das MyMeetings-Subset, ein eigenes Projekt oder Markitect selbst?
5. Welche Kosten und welche Prüfzeit je akzeptiertem Work Item sind akzeptabel, und gilt das vorgeschlagene Abbruchkriterium?
6. Soll nach dem Main-Merge zuerst der Kernthesen-Test laufen, bevor neue Produktfeatures oder Organisationsschichten entstehen?

## Anhang A: Befundliste

Schwere nach Gegenprüfung (korrigierte Schwere, wo vorhanden). Kein Befund wurde als Ganzes widerlegt. Einzelne Belege wurden verworfen, siehe Anhang B. X-IDs sind Nachträge der Prüfer.

| ID | Titel | Schwere | Prüfstatus |
|---|---|---|---|
| K1 | Kernthese ungetestet, kein gültiger Vergleichsbefund | kritisch | bestätigt |
| K2 | „Nichts vergessen“ gilt nur in geschlossener Welt deklarierter Zuordnungen | hoch | eingeschränkt |
| K3 | Granularitätsdilemma C17/C18 ungelöst | hoch | eingeschränkt |
| K4 | Vieraugenprinzip setzt Unabhängigkeit voraus, die bei gleichem Modell fehlt | mittel | eingeschränkt |
| K5 | Hypothese „günstigere Modelle“ ungetestet | hoch | eingeschränkt |
| K6 | Hierarchie aus Namespaces trifft Querschnittspflichten schlecht | mittel | eingeschränkt |
| K7 | Brownfield-Datenlücke; Modellierungsaufwand nicht als Konzeptkosten geführt | mittel | eingeschränkt |
| K8 | Gesamtcheck reproduziert das Skalierungsproblem, wenn semantisch ausgeführt | mittel | bestätigt |
| K9 | Prüflast wird zum Modell verschoben | mittel | bestätigt |
| K10 | Keine Zielwerte auf Produktebene | mittel | eingeschränkt |
| K11 | Formalisierungsaufwand wuchert (ein Richtungswechsel, per Nutzerauftrag) | mittel | eingeschränkt |
| K12 | Problemdiagnose und billigere Alternativen nicht vermessen | mittel | bestätigt |
| K13 | Status von Ressorts und Einstimmigkeit teils offen | niedrig | eingeschränkt |
| KX1 | Studie nutzt dasselbe Modell für Ausführung und Bewertung | mittel | Nachtrag Prüfer |
| KX2 | Konzeptmaterial vermischt Nutzerworte und KI-Ableitungen | mittel | Nachtrag Prüfer |
| KX3 | Pfad-Abrechnung als Gegengewicht zur geschlossenen Welt | niedrig | Nachtrag Prüfer |
| A1 | Government-Paket nicht nachgeführt, Entscheidung ohne Begründung | mittel | eingeschränkt |
| A2 | Einstimmigkeit formal streng, inhaltlich nur Formprüfung | hoch | eingeschränkt |
| A3 | Queue bindet festes `expectedBase`; kein Dauerbetrieb | hoch | bestätigt |
| A4 | Institutioneller Überbau übersteigt belegten Nutzen | mittel | eingeschränkt |
| A5 | Kostenhypothese C10 im Design nicht adressiert | mittel | eingeschränkt |
| A6 | Gericht beschrieben, nicht implementiert; Kriterium 7 offen | niedrig | eingeschränkt |
| A7 | Architektur nie nachgeführt; Zusagen nicht umgesetzt | mittel | bestätigt |
| A8 | Exakte Dateizuordnung und explizite Organisation statt Ableitung | mittel | bestätigt |
| A9 | Nutzerziele unversioniert, Koordination im Government-Ordner | niedrig | bestätigt |
| A10 | Zwei Vokabulare, keine Begriffsabbildung | niedrig | eingeschränkt |
| AX1 | Angliederung ist an Design beauftragt | mittel | Nachtrag Prüfer |
| AX2 | Schutz gegen Selbstermächtigung hängt an nicht authentifizierter Annahme | mittel | Nachtrag Prüfer |
| AX3 | `docs/README.md:12` führt Government als „Current“ | mittel | Nachtrag Prüfer |
| E1 | Kein gültiger Vergleich, kein Markitect-Arm über die erste Station | kritisch | bestätigt |
| E2 | Studiendesign unter der Problemschwelle, ohne Größen- und Kostenarm | hoch | eingeschränkt |
| E3 | Messinfrastruktur sehr umfangreich (Aufwandsrelation nicht gemessen) | mittel | eingeschränkt |
| E4 | Kontextverlust der LLM-Studienleitung entwertete zwei Zellen | hoch | bestätigt |
| E5 | Nur kooperative Isolation; v3-Brownfield-Zellen invalidiert | hoch | eingeschränkt |
| E6 | Bewegliches Vergleichsziel (Protokolle, Arme, Versionen) | hoch | eingeschränkt |
| E7 | Asymmetrische Ergebnisse; Readiness-, kein Kausalbefund | hoch | eingeschränkt |
| E8 | Government G1–G5 nur mit deterministischen Akteuren | mittel | bestätigt |
| E9 | Kostenhypothese nicht messbar | mittel | eingeschränkt |
| E10 | Vieraugenprinzip nicht Markitect-spezifisch belegt | niedrig | eingeschränkt |
| E11 | Impact-Recall nie gegen Pflichtstellen-Liste gemessen; Vorevidenz ungenutzt | mittel | eingeschränkt |
| E12 | Seit 9.10. Produktintegration statt Studie | mittel | bestätigt |
| EX1 | Asymmetrische Nachprüfung zugunsten Conventional | hoch | Nachtrag Prüfer |
| EX2 | Conventional-Befunde auf privaten Rohdaten | mittel | Nachtrag Prüfer |
| EX3 | Experimentschwellen ohne Varianzbasis | mittel | Nachtrag Prüfer |
| U1 | Drei parallele Ausführungsstapel | mittel | eingeschränkt |
| U2 | Impact-Hülle sehr breit im Referenzbeispiel | mittel | eingeschränkt |
| U3 | Echte LLM-Ausführung nur kleiner Fälle über einen Anbieter | hoch | eingeschränkt |
| U4 | Infrastruktur dominiert (Quote klassifikationsabhängig) | mittel | eingeschränkt |
| U5 | Keine Evidenz für Größe und Brownfield | hoch | bestätigt |
| U6 | Modellierungsaufwand der Government-Form | mittel | bestätigt |
| U7 | Monolithische Kernfunktionen | mittel | bestätigt |
| U8 | Abdeckungsmatrix C01–C19 | mittel | eingeschränkt |
| U9 | Starke Verzweigung, Main hinkt nach, Evidenz bläht Git | mittel | bestätigt |
| U10 | Token- und Kostenmessung partiell | mittel | bestätigt |
| U11 | CI- und Windows-Gate langsam, Abbruchschleifen | mittel | bestätigt |
| U12 | Statusdoku widerspricht Ledger | niedrig | bestätigt |
| UX1 | Keine OS-Isolation der Agentenausführung | mittel | Nachtrag Prüfer |
| UX2 | A01-Pass nach offenem Retry-Budget ist kein Zuverlässigkeitsbeleg | mittel | Nachtrag Prüfer |
| UX3 | Markitect modelliert sich selbst nicht mit dem Produktworkflow | mittel | Nachtrag Prüfer |
| P1 | Zentrale Akte für Menschen nicht mehr lesbar | hoch | eingeschränkt |
| P2 | Evidenz-Commits plus `cancel-in-progress` brechen CI ab | mittel | eingeschränkt |
| P3 | Aufwand in Steuerung und Belegführung, selbst ungemessen | mittel | eingeschränkt |
| P4 | Grant-/Freeze-Maschinerie erzeugt eigene Fehlerklasse | mittel | eingeschränkt |
| P5 | Kontextverlust als dominierender Ausfallmodus | hoch | bestätigt |
| P6 | Mensch als Regelkreis; Initialdesign zu restriktiv | mittel | eingeschränkt |
| P7 | 28 Versuche für einen Main-Smoke | mittel | eingeschränkt |
| P8 | Stern-Topologie statt rekursiver Organisation | mittel | bestätigt |
| P9 | Meilensteine in mehreren Dateien manuell synchron | niedrig | eingeschränkt |
| P10 | Provenienz dünn, Rohbelege privat | mittel | eingeschränkt |
| P11 | Branch- und Worktree-Wildwuchs | mittel | bestätigt |
| P12 | Zielbild: Rigor behalten, Beleg-Last verlagern | mittel | eingeschränkt (Empfehlung) |
| P13 | Konventionsdrift (Sprache, Überschriften, Präfixe) | niedrig | bestätigt |
| PX1 | Neueste Nutzerentscheidung: stabiler Main, dann Stopp | niedrig | Nachtrag Prüfer |

## Anhang B: Methode und Grenzen dieses Assessments

**Methode:**

- Fünf parallele Analyse-Linsen: Konzepte, Architektur, Evidenz, Umsetzung, Prozess. Jede wurde von einem eigenen skeptischen Prüfer gegengeprüft, Fundstellen wurden nachgelesen und Zahlen teils nachgezählt.
- Der Autor dieses Dokuments hat die Ergebnisse zusammengeführt, Doppelungen vereinigt und eingeschränkte Befunde in ihrer korrigierten Fassung übernommen.
- Selbst stichprobenartig geprüft: Branch-SHAs, `docs/README.md:12`, `evaluation.md:163`, die Zeilenzahl von `assessment-2026-10-09.md` (88), der Schluss von `coordination.md` (Z. 2368-2381), `work-item-comparison:9, :87` und die Nutzerbeschreibung.

**Gelesen bzw. ausgewertet:**

- Concepts-Ordner und Government-Designpaket.
- `coordination.md` per Überschriftenindex, gezielten Abschnitten und Greps; `coordination-state.json` per `python -I`.
- Herkunfts- und Richtungsdokumente vom 9./10.10.
- Classic-Dokumentation und -Code auf `origin/main`.
- Government-Code und -Doku auf `codex/government-worker`/`-p1`.
- Produktkandidat: `projectmodel`, `projectrun`, CI, Ledger.
- Scientist-Berichte: Terminal-Synthese, Roombook-Zellen 5 und 6.
- Historian-Branch.
- Ausgeführt im Scratchpad (Linse U und Prüfer, auf `git archive`-Exporten): Build, Vet und Tests von Government; Build des Produkt-Binarys; `project check` und `project impact` auf einer Kopie von `examples/project-world`.

**Verworfene oder korrigierte Teilaussagen der Linsen:**

- Zitate `assessment-2026-10-09.md:135/139/162/165` existieren nicht.
- Der Greenfield-`readiness_gap` ist kein Beleg gegen Einstimmigkeit, er traf Classic genauso.
- `coordination.md:2311` (behobener Host-Fehler) und der Kontextverlust der Studienleitung sind keine Belege gegen Hierarchien.
- „Mehrfacher Richtungswechsel“: Es gab einen, und er geht auf einen direkten Nutzerauftrag zurück.
- „Keine Metrik für Vergessenes“: Das Protokoll nennt sie (`evaluation.md:163`).
- „Conventional lieferte in jedem Lauf“, „alle Brownfield-Daten verloren“ und „Aufwand um Größenordnungen“ sind nicht belegt.
- „CI-Schleife erst durch externen Hinweis erkannt“ und „Eingriffe überwiegend Lockerungen“ sind überzeichnet.
- Eigene Zählungen (z. B. 123 Bezeichner, 27 Limits) sind nicht reproduzierbar.

**Nicht geprüft:**

- Inhalte der Codex-Chats (Overseer, Worker u. a.), GitHub-Läufe und PR-Kommentare. Der Ausgang von CI-Lauf `38057729908` und ob PR89 inzwischen gemergt ist, sind unbekannt.
- Private Rohbelege und Oracles der Studien.
- Die Inhalte der Binärdateien.
- `coordination.md` und `coordination-state.json` vollständig.
- `projectrun`/Resume im Detail, insbesondere ob aufeinander aufbauende Work Items ohne Neubindung laufen.
- Das lange `execution`-Testpaket von Government (vom Prüfer nicht erneut ausgeführt).
- Externe Literatur: von der Konzept-Linse gelesen, nicht gegengeprüft; Quellen aus 2026 nicht unabhängig verifiziert.
- Die Kosteneinheit der `*_cost_micros` im Ledger.

**Grenzen:**

- Dieses Assessment beruht selbst auf Agenten derselben Modellfamilie. Das in K4 und KX1 beschriebene Risiko korrelierter Urteile betrifft es ebenfalls.
- Die Gegenprüfung mildert dieses Risiko, ersetzt aber keine menschliche Prüfung.
- Zahlen zu Commits, Zeilen und Dateigrößen beziehen sich auf die oben genannten lokalen Stände. Einige Branches sind während der Analyse weitergewandert.

# Quellenbasis und Befundgrenzen

## Isolation und Snapshot

Alle `P`-Verweise dieses Pakets bezeichnen **ausschließlich** `fc6d09a234572c344279a342416475e788435f1f`. Dateizeilen sind 1-basiert. Root prüfte zentrale Schema-, Mutation-, Strictness-, Review-, Delivery- und Recoverybefunde zusätzlich zu den Subagenten. Ein Codepfad belegt vorhandene Mechanik; er belegt hier keinen erfolgreichen Lauf.

| Verwendeter Stand | Absoluter Pfad | Branch / vollständiger HEAD | Lokaler Zustand bei Prüfung | Verwendung |
|---|---|---|---|---|
| Eigene Evaluation | `C:/Users/Consiliari/.codex/worktrees/d08b/Markitect` | initial detached; danach `codex/government-evaluation-20261009`; `fc6d09a234572c344279a342416475e788435f1f` | initial sauber | Einzige aktuelle Produktbasis |
| Laufende Integration | `C:/Users/Consiliari/.codex/worktrees/product-integration/Markitect` | `codex/product-integration-20261009`; `fc6d09a234572c344279a342416475e788435f1f` | Auftrag nennt eine lokale Ledgeränderung; unsere Prüfung sah zusätzlich `integration-progress-20261009.md` geändert | Nur Isolation/Identität geprüft; keine WIP-Inhalte übernommen |
| Alte Government-Variante | `C:/Users/Consiliari/.codex/worktrees/government-worker/Markitect` | `codex/government-worker`; `04e225d5caee78c2a198607143863fca1e829750` | sauber | Getrennte experimentelle Source-/Reuseanalyse |
| Primärer Assessment-Checkout | `C:/Users/Consiliari/Glacius Labs/Markitect` | `codex/government-assessment`; `702e3f2321ea15fe203ec78cd66cf1ca7563623f` | Koordinationsdateien geändert und zusätzliche unversionierte Notizen/Grantdatei | Nur `git show` des festen Assessmentdokuments; keine aktuelle Produktcodebasis |
| Concepts | `C:/Users/Consiliari/.codex/worktrees/concepts-historian-notes/Markitect` | `codex/concepts-historian-notes`; `5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e` | `docs/README.md` geändert; `docs/design/concepts/` unversioniert | Ergänzende Ideen, ausdrücklich uncommittet und per Dateihash gebunden |
| Ideas | `C:/Users/Consiliari/.codex/worktrees/ideas-notes/Markitect` | `codex/ideas-notes-20261009`; `a57c64b40ca426c8e3a7211798e381c381ecc1ab` | sauber | Ergänzende Ideen, keine neuen Verträge |

Der beauftragende Chat reservierte A01 in fremder WIP. Am festen Produktcommit steht A01 auf `not_started`, `source: null` [P12]. Eine Reservierung, eine spätere lokale Änderung und die von uns vorausgesetzte Zuverlässigkeit sind drei verschiedene Dinge. Die Dokumentation am festen SHA meldet native Abnahme und finale Gates als NOT RUN [P1, P4]. Diese Untersuchung liest keine laufenden Acceptance-Ergebnisse nach und erweitert keine Grants.

## Aktueller Produktstand: vollständiger SHA und Zeilen

Für jede Zeile dieser Tabelle lautet der vollständige Source-SHA `fc6d09a234572c344279a342416475e788435f1f`.

| ID | Datei : Zeilen | Beleg und Grenze |
|---|---|---|
| P1 | `docs/README.md:3-7`; `docs/vision.md:7-41,87-112` | Integration candidate; Qualität/Intent-Treue als Ziel; Commit bindet Bytes, keine menschliche Freigabe |
| P2 | `docs/operating-methodology.md:15-45,47-57` | Intent, Execution, Independent Verification; Reparatur ohne künstlichen Intentwechsel; eigene Elternprüfung |
| P3 | `docs/architecture.md:3-22,85-91,145-151` | Host, aktuelle `src/`-Struktur, opaque artifacts, lokale Rechte und Evidenzgrenzen |
| P4 | `docs/project-workflow.md:9-25,33-37`; `docs/project-operations.md:19-39` | Explore/Edit/Readiness/Deliver und Recovery; technische Materialisierung ist keine Annahme |
| P5 | `src/internal/modules/projectmodel/projectmodel.go:5-67`; `src/internal/modules/projectmodel/analyze.go:20-46,82-131,155-220` | Manager/Statement/Artifact/Check/Decision; Ownership/Referenz-/Coverageprüfung; Decision hat subject/decision/reason/actor, keine Gerichtskompetenz |
| P6 | `src/internal/modules/projectmodel/impact.go:10-71`; `src/internal/host/projectrun/plan.go:393-449` | Expliziter Kontext und konservative Auswahl; keine vollständige fachliche Folgeninferierung |
| P7 | `src/internal/host/projectwork/mutation.go:139-175,255-307,341-376,482-595` | Alte Managerbasis, bounded Edit, Pflichtartefaktschutz, Frische; Actor-/Pfadprüfung ist kein materieller Legitimitätsnachweis |
| P8 | `src/internal/host/projectrun/operation.go:120-136`; `docs/design/project-world/operation-scopes-and-model-briefings.md:51-84,109-123` | Strictness addiert Evidenz/Gegenbeispiele; Mindestchecks bleiben; Dismiss ist keine Konfliktlösung |
| P9 | `src/internal/host/projectrun/review.go:103-170,542-560,657-695`; `src/internal/host/projectrun/full_verify.go:132-221` | Separates gebundenes Review, finaler Kandidat und gesamte Manager-/Censusprüfung; keine epistemische Unfehlbarkeit |
| P10 | `src/internal/host/projectrun/deliver.go:37-40,110-182`; `src/internal/host/projectrun/apply.go:41-158,187-319` | Autorisierte Plan/Run/Verify/Preflight/Apply-Kette, Zielbindung; keine Merge-/Releaseautorität |
| P11 | `src/internal/host/codexappserver/recovery.go:13-25,86-139`; `src/internal/host/projectrun/native_journal.go:72-103,141-179`; `src/internal/host/projectrun/start_accounting.go:10-23,89-103` | Recovery inspiziert ursprünglichen Turn ohne Replay; persistierte Starts und partielle Child-Beobachtung |
| P12 | `docs/work-items/product-readiness/evidence/native-acceptance-ledger.yaml:23-50` | Historischer geschlossener Pre-provider-Versuch; A01 am festen SHA nicht gestartet |
| P13 | `docs/design/project-world/README.md:5-34`; `docs/design/project-world/project-contract.md:17-19,69-85`; `docs/design/project-world/model.md:118-136` | Manager-/Geltungsmodell und subsidiäre Konfliktentscheidung als Entwurf; Government zusätzliche, spätere Achse |
| P14 | `docs/measurement.md:23-49`; `CONTRIBUTING.md:76-78` | Faire Messung inklusive Pflege/Recovery; prose-only Checks und keine Behauptung eines frischen Vollsuite-Ergebnisses |

Die vollständigen ergänzenden Codequellen und genaueren Bezugspunkte stehen in den jeweiligen [Analysen](README.md). Historische Abschnitte der Architektur und ältere Projektweltentwürfe sind als solche zu lesen; sie überschreiben die aktuelle Sourceoberfläche nicht.

## Alte Variante und Assessmentkontext

Alle `A`-Verweise bezeichnen **`04e225d5caee78c2a198607143863fca1e829750`**, unabhängig von `P`.

| ID | Datei : Zeilen | Beleg und Grenze |
|---|---|---|
| A1 | `internal/host/government/schema.go:30-65`; `internal/host/government/model.go:98-236` | Separate Constitution/Area/Ressort/Mandate/Responsibility/Realization-Welt |
| A2 | `internal/host/government/bindings.go:15-52,66-91,167-228`; `internal/host/government/promotion.go:20-38,65-166` | Material/Evidence/Vote/Decision und Ref-CAS; Caller muss Zustimmung prüfen |
| A3 | `internal/host/government/amendment.go:60-95,145-154`; `docs/government.md:77-89` | Amendments unter alter Befugnis, eingefrorene Organisationsidentitäten; kein Ownerkanal oder Court-Override |
| A4 | `internal/host/government/execution/queue.go:18-86,119-180`; `docs/government.md:91-104` | Endliche Queue, Budget/Journals/Resume; kein selbstinitiierender Dauerbetrieb |
| A5 | `docs/government.md:1-3,65-104`; `docs/validation/government-g1.md:1-36` | G1–G5-Source vorhanden; datierter G1-Bericht ist ein engerer Zwischenstand; Fixtures sind keine Urteilsgüte |
| A6 | `docs/design/government/architecture.md:97-139` | Alte konkrete Einstimmigkeits-/Gerichtsplanung; keine neue allgemeine Government-Policy |

Separater Entscheidungs-/Designkontext `C1`: `702e3f2321ea15fe203ec78cd66cf1ca7563623f:docs/design/government/assessment-2026-10-09.md:5-25,52-66`, gelesen per `git show`. Er hält den endlichen Untersuchungsblock, nicht abgeschlossene Vergleiche und Government als optionale Schicht fest. Seine damaligen Classic/Design-Quellstände sind keine aktuellen Produktstände. Hier werden weder dortige Ergebnisse neu reproduziert noch Counts als heutige Erfolgsraten verwendet. Die neue Nutzerprämisse steuert die strategische Bewertung.

## Gespräch und zusätzliche Notizen

Der Benutzer-/Assistentendialog des Produktchats `01a121f1-b948-7050-ae5d-9921b99db9c0` wurde vollständig bis raw-line 573 der zum Lesen verfügbaren lokalen Session erfasst. Die lokale Abschrift hat SHA-256 `45eaf3ec98c09571ab7a4b8ab3df9a16fc0c1b7b5a0936c551940c1d1fd98b51`; sie liegt unter `.artifacts/government-evaluation/product-conversation.md`. Das ist ein unversioniertes Erfassungsartefakt, kein Blob von `P`. [Nutzerideen](user-ideas.md) bewahrt relevante direkte Beiträge portabel. raw-line 188/200 präzisieren einmalige kanonische Pflege und den Produktanspruch; 389/411/434/457/524 liefern Freiheitsgrade, Government, Abwägung, Betrieb und diesen Auftrag. Historische Assistentenvorschläge bleiben Interpretation.

Die folgenden Concepts-Dateien wurden als uncommittete Notizen gelesen. Pfadwurzel: `C:/Users/Consiliari/.codex/worktrees/concepts-historian-notes/Markitect/docs/design/concepts/`. Ihre HEAD-Basis bindet **nicht** die neuen Notizen; dafür gelten die Dateihashes:

| Datei | SHA-256 der gelesenen Notiz | Einordnung |
|---|---|---|
| `government-as-model-governance-20261009.md` | `41d581e7dbbeccfd1a0b2d6cb3180a3a62c59b44ad23ba7066ebf5ab3dad4102` | Direkte Government-Idee und offene Rollengrenzen |
| `decision-framework-ideas-20261009.md` | `caf521bff32a73a31c0c088e115b37cb4ad88f3f804b4fbb5f3265b7ca159428` | Präzedenz bedeutet begründete Fälle, nicht Modelltraining oder automatische Normänderung |
| `operational-overview-ideas-20261009.md` | `ae293b4d9de0dd5b176f8bb58b53ff3e9d62ed7f7e40a4874dc7adefc96e7e0e` | Briefing/Beobachtbarkeit, offene Technik, ausgeschlossene Oberflächenideen |

Ideas: `a57c64b40ca426c8e3a7211798e381c381ecc1ab:docs/research/ideas/README.md`, insbesondere IDEA-20261009-008 bis -015; Dateihash `1a1347aca1e8c8846d0a12d6bba97369169d915706d319a2a5cf80136132cf37`. Die Einträge bestätigen offene Fragen und deferred planning. Sie sind ergänzende Ideenherkunft, keine Implementierungsevidenz. Es wurden keine Nachrichten an die anderen Chats gesendet und der Historian nicht aufgeweckt.

## Externe Primärquellen als begrenzte Forschungslinsen

Diese Quellen begründen Untersuchungsfragen, keine Produktgarantien oder Vorhersage für aktuelle Modelle:

- **Qualitätsziele als konkrete Szenarien:** ATAM behandelt konkurrierende Qualitätsattribute und deren Interaktionen. Unsere Folgerung ist, Projektprioritäten durch fallbezogene Szenarien und Folgen zu konkretisieren, statt Zahlengewichte als automatische Rechtfertigung zu behandeln. [SEI, Architecture Tradeoff Analysis Method, 1998](https://www.sei.cmu.edu/library/the-architecture-tradeoff-analysis-method/).
- **Richter sind fehlbare Evaluatoren:** Die MT-Bench-Arbeit untersucht unter anderem Positions-, Ausführlichkeits- und Selbstbevorzugungsbias von LLM-Judges. Daraus folgt hier nur die Prüfpflicht, dass sprachliche Zustimmung und Formatkonformität keine belastbare fachliche Annahme darstellen. Die damaligen Modelle und Aufgaben sind kein Test der Government-Entscheidungsgüte. [Zheng et al., Version 4](https://arxiv.org/abs/2306.05685v4).
- **Mehr Rollen schaffen neue Fehlerstellen:** MAST untersucht Systemdesign-, Inter-agent- und Verifikationsfehler in Multiagentensystemen. Für Government motiviert das begrenzte Zuständigkeiten, überprüfbare Übergaben und endliche Verfahren. Es liefert weder eine Erfolgswahrscheinlichkeit noch einen Kostenfaktor für Markitect. [Cemri et al., Version 3](https://arxiv.org/abs/2503.13657v3).

Weitere Primärquellen des unabhängigen Kritikers stehen in [seinem Bericht](analyses/critical-review.md). Ihre Übertragbarkeit wird dort ausdrücklich begrenzt. Kein hier gelesener Artikel evaluiert dieses Produkt oder beweist selbständige gute Modellpflege.

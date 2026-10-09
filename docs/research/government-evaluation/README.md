# Funktionale Government-Evaluation

Stand: 9. Oktober 2026. **Urteil dieser theoretischen Evaluation: Eine begrenzte Delegation der Modellpflege erscheint auf dem vorausgesetzten funktionierenden Markitect-Unterbau technisch implementierbar. Dafür müssen zusätzliche Entscheidungs- und Modellannahmeverträge geklärt und umgesetzt werden. Zuverlässige autonome Zielauslegung, dauerhafte Modellqualität und tatsächliche menschliche Entlastung sind damit nicht bewiesen.** Die kleinste sinnvolle Schicht verbindet einen Modellfall, explizite Entscheidungsbefugnis, unabhängige Prüfung und den vorhandenen Managerablauf. Eine vollständige Institutionenlandschaft wäre eine spätere Option.

## Scope und Prämisse

Der Nutzer setzt für diese strategische Untersuchung voraus, dass Markitect im kleinen Scope zuverlässig funktioniert: akzeptiertes kanonisches Modell, Managerzuständigkeiten, Context/Impact, bounded work, unabhängige Prüfung, kontrolliertes Apply und Recovery. Wir verwenden diese **Annahme**, ohne daraus bestandene Case Studies, Gesamtgates oder native Abnahme abzuleiten. Tatsächliche Sourcegrenzen werden separat erfasst. Die Untersuchung ist kein Produktbeweis und kein Implementierungsauftrag.

Government soll die Arbeit **vor** dem akzeptierten Modell delegieren: gewöhnliche Work Items und technische Anliegen interpretieren, angemessene Modelländerungen vorschlagen, Fachperspektiven einbeziehen, Konflikte innerhalb übertragener Autorität behandeln und relevante Entscheidungen an den Menschen zurückgeben. Das Modell bleibt die einzige kanonische Soll-Welt. Cockpit, UI, Visualisierung und Gamification sind ausgeschlossen. Ereignisse, Protokolle und Briefings werden nur für Autorität, Nachvollziehbarkeit und funktionalen Betrieb betrachtet. Kubernetes und etcd sind nicht ausgewählt.

## Feste Arbeitsbasis

- Eigener Worktree: `C:/Users/Consiliari/.codex/worktrees/d08b/Markitect`.
- Initialer HEAD: `fc6d09a234572c344279a342416475e788435f1f`; initial detached und sauber.
- Eigener Dokumentationsbranch: `codex/government-evaluation-20261009`, vom genannten Commit angelegt.
- Aktuelles Produkt ausschließlich an diesem festen SHA analysiert; weder Main noch eine Government-Variante wurde zur Produktbasis gemacht.
- Fremde Integrations-WIP wurde weder übernommen noch verändert. Spätere Integrationsarbeit verändert diese Evaluation nicht.
- Alte Government-Variante separat: `04e225d5caee78c2a198607143863fca1e829750`.

[Quellenbasis](source-basis.md) enthält Pfade, Branches, lokale Zustände, vollständige SHAs, Zeilenbezüge, Gesprächsherkunft und ergänzende Primärquellen. Die Evaluation verändert keine Produktverträge oder Roadmap. Der Research-Router enthält lediglich den Verweis auf dieses Paket.

## Lesen und Einordnung

| Dokument | Inhalt |
|---|---|
| [Umsetzbarkeitsurteil](feasibility-assessment.md) | Gesamturteil, Voraussetzungen, schwierige Teile, Aufwand und Alternativen |
| [Funktionaler Entwurf](functional-design.md) | Teilprobleme, Abhängigkeiten, Autorität, Work-Item-Weg, Rückkopplung, Betrieb und Modellqualität |
| [Spätere Validierungsfragen](validation-questions.md) | Was Experimente beweisen oder widerlegen müssten; keine gestarteten Versuche |
| [Nutzerideen](user-ideas.md) | Ausgewählte direkte Beiträge aus dem vollständig gelesenen Produktdialog, wortgetreu und von Vorschlägen getrennt |
| [Quellenbasis](source-basis.md) | Feste Sourcebefunde, datierter Entscheidungs-/Ideenkontext und externe Forschung |
| [Prüfprotokoll](verification-record.md) | Dokumentationschecks, Grenzen und unabhängige Gegenprüfung |

Sieben unabhängige Analyseberichte ergänzen die Synthese:

| Subagent | Bounded Untersuchung | Ergebnisdatei |
|---|---|---|
| fit | Produktproblem, menschliche Arbeit und Alternativen | [product-fit.md](analyses/product-fit.md) |
| authority | Vertikale Manager, horizontale Perspektiven und Lifecycle | [authority-and-lifecycle.md](analyses/authority-and-lifecycle.md) |
| adjudication | Konflikte, NFRs, Severity, Instanzen und Präzedenz | [adjudication.md](analyses/adjudication.md) |
| quality | Dauerhafte Qualität, Vereinfachung und Schutz der Erfolgskriterien | [model-quality.md](analyses/model-quality.md) |
| runtime | Frische, Konkurrenz, Recovery, Budget und Briefings | [operational-safety.md](analyses/operational-safety.md) |
| reuse | Aktuelles Produkt und alte Government-Mechanik | [source-reuse.md](analyses/source-reuse.md) |
| critic | Unabhängige stärkste Gegenargumente und spätere Syntheseprüfung | [critical-review.md](analyses/critical-review.md) |

Alle sieben wurden explizit mit `gpt-6-sol`, `reasoning_effort: high` und `fork_turns: none` gestartet; sie starteten keine weiteren Subagenten. Jeder besaß eine eigene Ergebnisdatei. Nur Root bearbeitet Synthese und Router. Der Kritiker formulierte seine Erstanalyse ohne die anderen Berichte und prüfte anschließend die Synthese gesondert in [synthesis-review.md](analyses/synthesis-review.md). Die dortigen vier Findings und ihre Bearbeitung stehen im Prüfprotokoll. Die Zahl der Agenten ist kein Qualitätsbeweis: gemeinsame Modellfamilie und Quellen können gemeinsame Fehler erzeugen.

## Evidenzsprache

**Nutzeridee** bezeichnet direkte Gedanken mit offenem Status. **Befund** bezeichnet gelesenen Code oder einen dokumentierten Vertrag am festen SHA; beides wird unterschieden. **Vorschlag/Bewertung** ist unsere Schlussfolgerung. **Hypothese** bezeichnet erwartete Wirkung, insbesondere Entscheidungsqualität, weniger Aufsicht und Kosten. Es wurden keine Produktimplementierung, Tests, Case Studies, realen Government-/Markitect-Produktläufe, Veröffentlichung oder Eingriffe in die laufende Integration gestartet.

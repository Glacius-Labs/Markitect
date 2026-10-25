# Owner directive: V1 acceptance and honest evaluation

Date: 25 October 2026. Source: the owner's direct message in a Claude Code session after an independent review, while the coordinator drove Phase B and the Phase C preparation. The owner statement is verbatim; the English translation is the recorder's and condensed where marked.

## Owner statement (verbatim)

> Markitect V1 – Nächste Schritte: Stabilisierung, Konzepttreue und belastbare Evaluation
> Nach der erneuten unabhängigen Prüfung möchte ich die nächsten Arbeiten klar priorisieren.
> Wir haben die grundlegende V1-Architektur umgesetzt und wesentliche technische Eigenschaften mehrfach nativ bestätigt. Ein neuer Architekturumbau ist derzeit nicht begründet.
> Allerdings haben Phase A und B keinen Qualitätsvorteil gegenüber einem einzelnen Codex-Agenten gezeigt, während Markitect erheblich mehr Zeit und Tokens benötigt.
> Wir müssen deshalb zwei Dinge unabhängig voneinander abschließen:
>
> 1. Technische V1-Abnahmefähigkeit: Funktioniert Markitect zuverlässig entsprechend unserem akzeptierten Konzept?
> 2. Produktnutzen: Unter welchen Bedingungen liefert die zusätzliche Managerorganisation nachweisbar bessere Ergebnisse als ein einzelner Agent?
>
> Diese beiden Fragen dürfen wir nicht vermischen.
> P0 – Phase-C-Eingaben wieder konsistent machen
> PR #713 hat nach einem echten Model Blocker neue Makerspace-Regeln für berechnete Werte, Datumsgrenzen und Integerlimits ergänzt.
> Auf Main wurden bislang die neutralen Requirements geändert. Die dazugehörigen kanonischen Modellsnapshots und gerenderten Einzelagent-Dokumente enthalten diese Ergänzungen noch nicht.
> Es dürfen keine gezählten Phase-C-Läufe starten, bevor diese Inkonsistenz geschlossen ist.
> Bitte:
>
> * Aktualisiere die kanonischen Modelle für alle betroffenen Zustände W0–C9.
> * Generiere die Einzelagent-Dokumentation erneut aus diesen Modellen.
> * Überprüfe die vollständige beidseitige semantische Äquivalenz zu den neutralen Anforderungen.
> * Prüfe den unabhängigen Oracle, Referenzimplementierungen, Mutanten und die dazugehörigen Kalibrierungsfälle auf Auswirkungen.
> * Ergänze fehlende Grenzfälle für die neuen Regeln.
> * Führe die betroffene Kalibrierung erneut durch und dokumentiere die neue Eingabeversion.
> * Aktualisiere gebundene Digests, Materialisierungsnachweise und den Vergleichs-Freeze.
>
> Die neutralen Requirements bleiben die gemeinsame Produktquelle für beide Versuchsarme.
> Ändere keine Produktanforderung lediglich, damit ein bestehender Oracle oder eine Referenzimplementierung grün wird.
> Bereits entstandene Fehlläufe und frühere Kalibrierungen bleiben historisch unverändert.
> Abschlusskriterium: Beide Versuchsarme erhalten nachweislich dieselbe beabsichtigte Fachlichkeit, und der unabhängige Bewertungsmaßstab wurde gegen diese Fachlichkeit geprüft.
> Danach ein einzelner begrenzter ungezählter Smoke zur Bestätigung des korrigierten Testaufbaus.
> P1 – ELF-Snapshot-Regel unabhängig überprüfen
> PR #706 klassifiziert alle ungetrackten ELF-Dateien anhand ihrer Dateisignatur als operative Build-Outputs.
> Das löst ein reales Problem, wirft aber eine Architekturfrage auf:
> Ein Dateiformat beweist nicht die fachliche Bedeutung einer Datei.
> Bitte prüfe anhand des tatsächlichen Codes und eines gezielten realen Git-Tests:
>
> * Das kanonische Modell verlangt ausdrücklich ein neues binäres ELF-Artefakt.
> * Der Manager erzeugt und registriert das Artefakt.
> * Er führt kein manuelles `git add` aus.
> * Der Host fixiert den Kandidaten und führt die vorgesehenen Checks aus.
>
> Welche Diagnose erhält der Manager? Kann er die Abgabe im vorgesehenen Ablauf korrigieren? Kann ein benötigtes Artefakt stillschweigend aus dem Kandidaten verschwinden?
> Prüfe außerdem den Fall, in dem ein solches Artefakt ausdrücklich gestagt wird, und die normalen Fälle temporärer Build-Outputs.
> Bewerte den Mechanismus gegen unser Prinzip vollständiger und nachvollziehbarer Artefaktzuordnung.
> Falls der bestehende Mechanismus sämtliche Garantien erfüllt, dokumentiere die Begründung. Falls er legitime Produktartefakte verdecken kann, implementiere die kleinste konzepttreue Korrektur.
> Keine allgemeine Build- oder Artefaktklassifizierungsplattform.
> P2 – Verbleibenden öffentlichen Verify-Reject-Nachweis erbringen
> Die bisherigen nativen Läufe sind wertvoll, aber der öffentliche Korrekturzweig wurde noch nicht vollständig durchlaufen.
> Ich möchte dafür einen gezielten, begrenzten End-to-End-Nachweis auf dem korrigierten Produktstand:
> Assessment Approve
> → Implement
> → unabhängiges Verify Reject mit konkreten Findings
> → neuer Implement Run
> → neues unabhängiges Verify Approve
> → Publication des exakt akzeptierten Kandidaten.
> Das Szenario muss fachlich legitim sein. Der Host darf nicht manipuliert werden, um künstlich einen Reject zu erzeugen.
> Ein geeignetes kontrolliertes Testprojekt oder ein nachvollziehbar konstruierter Fehlerfall ist zulässig.
> Prüfe besonders die Frische der Sessions, die unveränderte Gesamtdeadline, Kandidatenbindung, Ressourcenabschluss und den abschließenden Readback.
> Falls der Native Provider den vorgesehenen Fehler unerwartet bereits im Assessment erkennt, darf das nicht als P5-Erfolg umgedeutet werden.
> Begrenze Versuche und Kosten im Voraus. Bewahre jeden Versuch einschließlich eines möglichen Scheiterns.
> P3 – Aktuellen Code als zusammenhängenden Stand stabilisieren
> Untersuche die verbleibenden zeitabhängigen `gitrepo`-Testfehler.
> Unterscheide:
>
> * tatsächliche Produktfehler,
> * fehlerhafte Test-Synchronisation,
> * Last- oder Umgebungsprobleme,
> * und ungeeignete Zeitlimits.
>
> Keine pauschalen Timeout-Erhöhungen ohne Ursachenanalyse.
> Nach den notwendigen Korrekturen soll ein konkreter Endstand eingefroren und vollständig geprüft werden:
>
> * Build, Vet und sämtliche Go-Tests;
> * Architektur- und Modellchecks;
> * Linux-, Windows- und Race-CI;
> * Kandidatenpersistenz und Publication-Guards;
> * unabhängiger vollständiger Diff- und Architekturreview.
>
> Frühere erfolgreiche CI qualifiziert keinen später veränderten Code automatisch.
> P4 – Phase C durchführen und anschließend ehrlich auswerten
> Nach dem erfolgreichen Konsistenz-Gate und einem begrenzten Smoke soll die Makerspace-Vergleichsstudie entsprechend ihrem genehmigten Protokoll durchgeführt werden.
> Die Studieninfrastruktur soll nicht während der gezählten Läufe inhaltlich verändert werden.
> Behalte die Trennung zwischen neutralen Requirements, kanonischem Modell, gerenderter Dokumentation, unabhängigen Referenzen und verborgenem Oracle bei.
> Bewerte mindestens:
>
> * vollständige fachliche Korrektheit;
> * übergreifende Invarianten;
> * Regressionen nach Änderungen;
> * unbeabsichtigte Änderungen;
> * No-change und Preservation;
> * Laufzeit und Tokenverbrauch;
> * notwendige menschliche Eingriffe;
> * technische Fehlerraten und unvollständige Runs.
>
> Halte dich an die vorab festgelegten Entscheidungskriterien.
> Eine nachträgliche Änderung des Bewertungsmaßstabs darf nicht als ursprüngliches Studienergebnis ausgegeben werden.
> Ich will nicht, dass wir immer schwierigere Tests erfinden, bis Markitect irgendwann gewinnt. Ich will wissen, ob und unter welchen Bedingungen das Konzept tatsächlich einen Vorteil liefert.
> Wenn der Einzelagent auch Phase C gleich gut bewältigt, akzeptieren wir dieses Ergebnis und untersuchen anschließend die Ursachen für den hohen Aufwand von Markitect.
> Ein negatives Ergebnis bedeutet nicht automatisch, dass das kanonische Modell oder die Architektur verworfen werden muss. Es bedeutet aber, dass die zusätzliche Managerorganisation ihren wirtschaftlichen Vorteil bislang nicht gezeigt hat.
> Reihenfolge und Arbeitsweise
> P0 hat Vorrang vor weiteren gezählten Phase-C-Ausführungen.
> P1, P2 und P3 können vorbereitet werden, soweit Dateiverantwortung, Ressourcen und Git-Worktrees sauber getrennt bleiben.
> Die endgültige technische V1-Abnahme bezieht sich auf den gemeinsam geprüften tatsächlichen Endstand, nicht auf die Summe alter Nachweise verschiedener Commits.
> Phase C liefert eine separate Entscheidung über den nachgewiesenen Produktnutzen.
> Keine neue Produktfeatures, keine generische Workflow-DSL, keine zusätzliche Agentenorganisation und kein umfassendes Refactoring ohne konkreten Befund.
> Routineentscheidungen innerhalb des freigegebenen Scopes kannst du selbst treffen. Wesentliche Änderungen an der Produktsemantik legst du mir vorher begründet vor.
> Erwartetes Ergebnis
> Liefere zunächst einen kompakten Maßnahmenplan mit Verantwortlichkeiten, Abhängigkeiten, begrenzten Ausführungen und eindeutigen Abschlusskriterien.
> Führe danach die notwendigen Arbeiten kontrolliert aus.
> Am Ende möchte ich zwei getrennte Berichte:
> Technische V1-Abnahme: Welche Garantien sind auf dem finalen Stand tatsächlich bewiesen? Welche Grenzen bleiben offen?
> Vergleich und Produktthese: Welche Qualitäts-, Kosten- und Zuverlässigkeitsunterschiede wurden zwischen Markitect und dem Einzelagenten beobachtet? Welche Schlussfolgerungen sind durch die Daten gedeckt?
> Mein Maßstab bleibt derselbe:
> Das kanonische Modell bestimmt die Architektur. Die Manager tragen die fachliche Verantwortung. Der Host garantiert technische Korrektheit, ohne die Agenten unnötig zu mikromanagen.
> Wir wollen eine zuverlässige, einfache und nachvollziehbare V1 – und ein objektives Ergebnis darüber, ob ihre zusätzliche Komplexität einen echten Nutzen bringt.
> Release und Deployment bleiben separat freizugeben.

## English translation (recorder's, condensed)

The basic V1 architecture is implemented and essential technical properties were confirmed natively several times; a new architecture rework is not justified. Phases A and B showed no quality advantage over a single Codex agent, while Markitect needs considerably more time and tokens. Two things must therefore be closed independently:

1. Technical V1 acceptance: does Markitect work reliably according to the accepted concept?
2. Product value: under which conditions does the added Manager organization deliver demonstrably better results than one agent?

These two questions must not be mixed.

- **P0 – make the Phase C inputs consistent again.** PR #713 added Makerspace rules for computed values, date limits and integer limits after a genuine model blocker. So far only the neutral requirements changed on Main; the canonical model snapshots and the rendered single-agent documents do not contain the additions yet. No counted Phase C run may start before this inconsistency is closed. Update the canonical models for all affected states W0–C9; regenerate the single-agent documentation from them; check complete two-way semantic equivalence with the neutral requirements; check the independent oracle, reference implementations, mutants and calibration cases for impact; add missing boundary cases for the new rules; rerun the affected calibration and record the new input version; update bound digests, materialization evidence and the comparison freeze. The neutral requirements stay the common product source for both arms. Do not change a product requirement just to make an existing oracle or reference implementation pass. Failed runs and earlier calibrations stay historically unchanged. Completion: both arms demonstrably receive the same intended domain behavior, and the independent assessment was checked against it. Then one bounded, uncounted smoke confirms the corrected setup.
- **P1 – review the ELF snapshot rule independently.** PR #706 classifies every untracked ELF file by its signature as an operational build output. That solves a real problem but raises an architecture question: a file format does not prove what a file means. Check with the actual code and a targeted real-Git test the case where the canonical model explicitly requires a new binary ELF artifact, the Manager creates and registers it without a manual `git add`, and the host fixes the candidate and runs the declared checks: which diagnostic does the Manager get, can it correct the handoff in the normal flow, can a required artifact silently disappear from the candidate? Also check an explicitly staged artifact and the normal cases of temporary build outputs. Judge the mechanism against the principle of complete and traceable artifact accounting. If it meets every guarantee, document why; if it can hide legitimate product artifacts, implement the smallest concept-faithful correction. No general build or artifact classification platform.
- **P2 – produce the remaining public Verify-reject evidence.** A targeted, bounded end-to-end proof on the corrected product state: Assessment approves → Implement → independent Verify rejects with concrete Findings → a new Implement Run → a new independent Verify approves → publication of exactly the accepted candidate. The scenario must be legitimate in domain terms; the host must not be manipulated to produce an artificial reject. A suitable controlled test project or a traceable constructed fault case is allowed. Check session freshness, the unchanged total deadline, candidate binding, resource closure and the final readback. If the native provider unexpectedly catches the intended fault in Assessment, that must not be reinterpreted as a P5 success. Bound attempts and cost in advance and keep every attempt, including failure.
- **P3 – stabilize the current code as one coherent state.** Investigate the remaining time-dependent `gitrepo` test failures and distinguish actual product defects, faulty test synchronization, load or environment problems and unsuitable time limits. No blanket timeout increases without cause analysis. After the necessary fixes, freeze a concrete final state and check it completely: build, vet and all Go tests; architecture and model checks; Linux, Windows and race CI; candidate persistence and publication guards; an independent complete diff and architecture review. Earlier successful CI does not qualify later-changed code.
- **P4 – run Phase C and evaluate it honestly.** After the consistency gate and a bounded smoke, run the Makerspace comparison according to its approved protocol. The study infrastructure must not change in content during counted runs. Keep neutral requirements, canonical model, rendered documentation, independent references and hidden oracle separate. Evaluate at least complete domain correctness, cross-cutting invariants, regressions after changes, unintended changes, no-change and preservation, runtime and tokens, necessary human interventions, and technical failure rates and incomplete runs. Keep to the decision criteria fixed in advance; a later change of the evaluation rule must not be presented as the original result. "I do not want us to invent ever harder tests until Markitect eventually wins. I want to know whether and under which conditions the concept really delivers an advantage." If the single agent handles Phase C equally well, we accept that result and then investigate the causes of Markitect's high effort. A negative result does not automatically mean that the canonical model or the architecture must be discarded, but it means that the added Manager organization has not yet shown its economic advantage.

Order and way of working: P0 has priority over further counted Phase C executions. P1, P2 and P3 can be prepared as long as file responsibility, resources and Git worktrees stay cleanly separated. The final technical V1 acceptance refers to the jointly tested actual final state, not to the sum of old evidence from different commits. Phase C provides a separate decision on demonstrated product value. No new product features, no generic workflow DSL, no additional agent organization and no broad refactoring without a concrete finding. Routine decisions within the released scope are the coordinator's; material changes of product semantics go to the owner first, with reasons.

Expected result: first a compact action plan with responsibilities, dependencies, bounded executions and clear completion criteria; then the controlled execution. At the end, two separate reports: technical V1 acceptance (which guarantees are actually proven on the final state, which limits remain open) and comparison and product thesis (which quality, cost and reliability differences were observed, which conclusions the data support). The standard stays the same: the canonical model determines the architecture, the Managers carry the domain responsibility, the host guarantees technical correctness without unnecessarily micromanaging the agents. Release and deployment remain subject to separate approval.

## Recorded meaning

Recorder's interpretation: the directive separates technical acceptance from product value. The coordinator's plan with owners, dependencies, bounds and completion criteria is in the [delivery plan](../../implementation-plan.md#acceptance-and-evaluation-owner-directive-25-october). It does not change [DEC-072](../register.md#dec-072-stable-v1-justified-by-data); it sets the order and standards for meeting it.

# Unabhängiges Gesamtassessment: Markitect

**Reviewdatum:** 6. Oktober 2026. **Quelle:** ausschließlich der eingefrorene primäre Snapshot `4bdd7a2dda6e1e7b37b04f47046ee65adc081c6d`, Branch `codex/astra-assessment-snapshot`, unter `C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect`. Er besteht aus Coordinator-Basis `7c7abb1d26db7f117b54f2bd4a4ea02fb6865a9c` plus eingefrorenem Arbeitsstand; der zugelieferte Patch-Fingerprint lautet `C93DAC405ABC5E622AE814A969DFAF6F63DAA16C23E3F236165B54E2511C764F`. Die Dokumentintegration entspricht dem zwischenzeitlichen Coordinator-Commit `5543b811c628f114a16b84dcf64b10b0279913a4`; übrige zusätzliche Codeänderungen waren beim Capture WIP. Das ist eine synthetische Reviewrevision, kein veröffentlichter Release.

## Urteil

**Die Vision ist kohärent, und die grundlegende Architektur passt dazu. Ich empfehle keinen erneuten Core-Reset und kein zusätzliches Framework.** Die Trennung von akzeptierter Intention, strukturellem Compiler, installierten Fähigkeiten, konkreter Bindung, beobachteten Artefakten und operativer Evidenz ist sinnvoll und in wesentlichen Teilen tatsächlich implementiert.

**Der gegenwärtige Stand ist jedoch noch kein geschlossener autonomer Arbeitsprozess.** Besonders relevant: Nach einer fehlgeschlagenen semantischen Verifikation fehlt im normalen Controller der Übergang zu einem Reparaturauftrag bei unverändertem Intent und unveränderten, aber falschen Zielbytes. Erkennen und korrektes Rotbleiben funktionieren; das Weiterarbeiten aus diesem Befund ist nicht geschlossen. Daneben sind Driftbeobachtung, konservativer Evidenzrefresh und das abschließende Prozessgate noch so getrennt, dass ein erfahrener Coordinator sie aktiv zusammenhalten muss.

Die Sorge, sich in immer kleineren Proofdetails zu verlieren, ist daher berechtigt **als Priorisierungsproblem**, nicht als Argument gegen SHA-Bindung, unabhängige Prüfung oder sichere Writes. Diese Schutzmechanismen sind für autonome Änderungen notwendig. Ihr weiterer Ausbau darf aber nicht den Nachweis ersetzen, dass ein echter Auftrag mit Fehler, Reparatur und Elternprüfung zuverlässig zum Abschluss kommt.

Die neuen Dokumente ziehen die richtige Grenze: erst Methode und benötigte Fähigkeiten, dann Komponenten, vollständiger Ablauf, rekursive Komposition, kleine wiederholte Nutzung und anschließend fairer Vergleich. Diese Reihenfolge sollte jetzt die Arbeit steuern. Die ursprünglichen negativen Versuche bleiben historische Tatsachen; ihre wissenschaftliche Aussage darf dennoch kritisch beurteilt werden.

## 1. Was die Vision leisten kann – und was sie falsifizieren würde

Das Projektweltbild als ausgewählte Ontologie ist ein brauchbarer Autoritätsansatz: akzeptierte Konzepte, Zwecke, Beziehungen, Regeln und Verantwortlichkeiten werden explizit; ihre Repräsentationen sollen daran ausgerichtet werden. Entscheidend ist die inzwischen ausdrücklich formulierte Menge zulässiger Repräsentationen statt einer allgemeinen exakten Bytefunktion. Auch die Unterscheidung zwischen Intentänderung und Reparatur ist richtig. Das steht klar in [docs/operating-methodology.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/operating-methodology.md:7>) und [docs/operating-methodology.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/operating-methodology.md:29>).

Der Compiler beweist Struktur. Markitect kann außerdem nachvollziehbar ermitteln, welche **deklarierten** Zusammenhänge betroffen sind, passende Informationen übergeben, Arbeit begrenzen und fehlende Evidenz offenhalten. Projektchecks und Verifier müssen die eigentliche Übereinstimmung mit Bedeutung und Verhalten beurteilen. Das ist kein konzeptioneller Mangel des minimalen Core; genau dort gehört diese Verantwortung hin.

Die zentrale Hypothese lautet für mich: **Explizite Intention plus verbindlicher Arbeitsablauf und hinreichend unabhängige Prüfungen führen über mehrere Änderungen zu weniger übersehenen Verpflichtungen und geringerem routinemäßigem menschlichem Korrekturbedarf.** Sie lautet weder „jede Ontologie ist vollständig“ noch „ein Agent versteht jede Prosa richtig“.

Ein ernsthafter Gegenbefund wäre beispielsweise:

- Relevante Verpflichtungen lassen sich nur durch eine zweite, ebenso aufwendige Spezifikation neben dem eigentlichen Modell zuverlässig transportieren.
- Korrekte lokale Änderungen zerstören wiederholt unmodellierte, praktisch wesentliche Elternverträge, obwohl ein realistischer Modellierungsaufwand investiert wurde.
- Modell- und Prüfpflege benötigen dauerhaft mehr menschliche Aufmerksamkeit, ohne entsprechend bessere Qualität zu liefern.
- Verifier übersehen dieselben relevanten Fehler wie Executor, oder ihre Beurteilungen hängen mehr von bevorzugtem Codeaufbau als von akzeptierten Verpflichtungen ab.
- Ein im deklarierten Einsatzbereich vollständiger Ablauf lässt sich trotz implementierter Reparatur- und Refreshwege nicht ohne dauerndes Coordinator-Nachsteuern betreiben.

Ein Git-Timeout, eine fehlende Controllerfunktion oder ein ungültiger Harnessaufbau falsifizieren diese These dagegen nicht. Sie können die aktuelle Implementierung oder Einsatzreife klar begrenzen.

Auch „weniger menschliche PR-Kontrolle“ ist sinnvoll, wenn Routineentscheidungen vorher delegiert sind. Ownerentscheidungen und explizit geforderte Akzeptanz bleiben eigene Schritte; zusätzliche Menschen an jedem technischen Gate wären keine notwendige Konsequenz des Modells. Die Methodik unterscheidet das inzwischen angemessen: [docs/operating-methodology.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/operating-methodology.md:19>).

## 2. Was an Architektur und Umsetzung trägt

| Teil | Reviewbefund und praktische Bedeutung |
|---|---|
| Minimaler Core | `Schema/Kind/Property/Definition`, vollständige nominale Identität, geschlossene Typen, begrenzte Eingaben und aufgelöste Referenzen sind unabhängig von Projekt-, Provider- und Ausführungswissen. Modell-Digest und Quellenrevision bleiben getrennt. Das trägt die gewünschte Extensibilität ohne universelle Engineering-Ontologie. Belege: [internal/core/types.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/core/types.go:30>), [internal/core/types.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/core/types.go:108>), [internal/core/compile.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/core/compile.go:38>). |
| Module und Bindung | Schema/Projection werden tatsächlich getrennt validiert; Schema-zu-Projection-Abhängigkeit wird abgelehnt. Eine gewünschte Projection ist nicht ihre installierte konkrete Implementierung. Fehlende, doppelte oder mehrdeutige Bindungen scheitern explizit. Belege: [internal/host/canonical/catalog.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical/catalog.go:330>), [internal/host/canonical/catalog.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical/catalog.go:678>), [internal/host/canonical/binding.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical/binding.go:175>). |
| Impact und Kontext | Host gibt expliziten Referenzen eine nachvollziehbare Bedeutung für Rückwärtsinvalidation; Context folgt begrenzten ausgehenden Referenzen. Das ist konservativ und erklärbar. Es erkennt keine undeclared Sourceabhängigkeiten. Belege: [internal/host/canonical_impact.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_impact.go:94>), [internal/host/canonical_agent_context.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_agent_context.go:40>). |
| Ausführung und Writes | Kandidaten werden vor Apply gestaged; Apply rekonstruiert die geplanten Bytes, prüft Runtime/Scope/Preimages und dokumentiert Teilzustände. Es setzt Materialisierung nicht mit Verifikation gleich. Das ist eine tragfähige technische Grenze. Belege: [internal/host/canonical_controller_execution.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_execution.go:134>), [internal/host/canonical_controller_execution.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_execution.go:338>), [internal/host/canonical_controller_execution.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_execution.go:377>). |
| Rekursive Komposition | Eltern erhalten tatsächliche Kindkandidaten oder genau gebundene aktive Kindbytes. Fehlende geplante Kinder können nicht still durch alte Bytes ersetzt werden. Jeder Elternknoten braucht eigene Evidenz; ein grüner Parent kann rote Kinder nicht löschen. Belege: [internal/host/canonical_controller_dependencies.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_dependencies.go:127>), [internal/host/canonical_controller_verification.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_verification.go:678>), [internal/host/assurance/assurance.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/assurance/assurance.go:287>). |
| Operative Records | Unveränderliche Versuche, Verifikation und aktive Besitzerauswahl bleiben getrennt. Das vermeidet die gefährliche Gleichsetzung „Datei geschrieben = akzeptiert“. Auch fehlgeschlagene Verifikation löscht nicht die Ownership. Belege: [internal/host/recordstore/store.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/recordstore/store.go:336>), [internal/host/records/README.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/records/README.md:10>). |
| Selektive Akquisition | Der aktuelle Controller selektiert Git-Inhalte und beobachtete Zielbytes, statt den historischen Vollakquisitionsfehler einfach umzubenennen. Metadateninventar und tatsächliche Inhaltsbeobachtung werden unterschieden. Belege: [internal/host/canonical_scoped_reconciliation.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_scoped_reconciliation.go:150>), [internal/infrastructure/source/selective.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/infrastructure/source/selective.go:221>). |

Diese Architektur benötigt weiterhin einen Host mit beträchtlicher Verantwortung. Das ist hier nicht automatisch ein Clean-Architecture-Verstoß: Sourceakquisition, Laufzeitbindung, Scheduling, Dateischreiben und Recordpersistenz gehören tatsächlich zur Anwendung. Die statische, technologiespezifische Verdrahtung und etwa `*dotnet.ExecutorTask` in [internal/host/canonical_scoped_reconciliation.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_scoped_reconciliation.go:22>) begrenzen jedoch die Erweiterung ohne Hoständerung. **Module unabhängig vom Core bedeutet aktuell nicht „beliebiges Projection-Paket installieren und ohne Hostintegration ausführen“.** Das ist mit der dokumentierten statischen Komposition vereinbar. Erst wiederholte konkrete Erweiterungsreibung rechtfertigt eine Vereinheitlichung dieser kleinen Adaptergrenze; ein dynamischer Pluginloader ist dafür nicht erforderlich.

Die Sicherheitsarbeit an Pfaden, Bytes, Ledger und Prozesslebensdauer ist sachlich begründet. Die im Snapshot hinzugekommenen UTF-8-, Log-, Verzeichnisbegrenzungs- und Akquisitionskorrekturen adressieren echte technische Grenzen. Ich habe sie als Diff geprüft, aber ihre vollständige plattformübergreifende Wirksamkeit nicht erneut getestet. Insbesondere sind Prozessgruppen bzw. Windows Job Objects keine allgemeine Isolation des aufrufenden Benutzers; die Runnerdokumentation grenzt das korrekt ab: [internal/host/agentexec/README.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/agentexec/README.md:5>).

## 3. Priorisierte konkrete Findings

### F1 — P1: Fehlgeschlagene Verifikation wird nicht zu Reparaturarbeit

**Kategorie:** fehlender Prozessübergang in einer ansonsten implementierten Kette. **Confidence: hoch, aus vollständigem Sourcepfad; kein neuer realer Agentenreplay.**

Ein frisch angewendeter .NET-Kandidat kann semantisch falsch sein, obwohl seine Dateien exakt dem aktiven Record entsprechen. Der separate Verifier erkennt den Fehler und speichert ein rotes Resultat. Bei erneutem Propose bleibt Intent unverändert, die Materialisierung vollständig und der Bytevergleich unverändert.

Der Sourcepfad ist eindeutig:

1. [internal/host/canonical_controller_verification.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_verification.go:268>) hängt VerificationResults an; er verändert die aktiven Materialisierungsrecords nicht.
2. Ohne `auditAll` wird ein solcher Record anhand der Schedulingkriterien in [internal/host/canonical_scoped_reconciliation.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_scoped_reconciliation.go:194>) nicht zwingend überhaupt beobachtet.
3. Mit `auditAll` werden Bytes beobachtet; [internal/host/canonical_controller_reuse.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_reuse.go:105>) akzeptiert nur `passed` für Reuse. Der failed-Befund wird nicht in einen Repairauftrag übersetzt.
4. [internal/modules/dotnet/proposal.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/modules/dotnet/proposal.go:130>) erzeugt Work nur für initiale/unvollständige Materialisierung, Byte-Drift oder kanonische Betroffenheit. Andernfalls folgen in Zeilen 152–159 `no-op` und `evidence-refresh-required`.
5. [internal/host/canonical_controller_execution.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_execution.go:61>) führt ausschließlich `work` aus. Sein Agentenkontext in Zeilen 78–93 enthält keine früheren Verifierfindings.

Im operationalen VerificationResult verbleiben Ergebnis/Checks und Verweise auf den separaten Lauf; die detaillierten Beobachtungen sind im Verifierreport, werden aber hier nicht als Reparatureingabe weitergereicht ([internal/host/canonical_controller_verification.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_verification.go:805>), [internal/host/canonical_controller_verification.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_verification.go:859>)). Weder Refresh noch erneutes Verify reparieren falsche Bytes. Manuell vorbereitete Kandidaten über niedrigere Tools bleiben möglich; das schließt den normalen Prozess nicht.

**Folge:** Der Controller bewahrt das Scheitern korrekt, benötigt aber externe Intervention, um wieder implementierend tätig zu werden. Das ist keine Falsifikation der Ontologieidee.

**Kleinster fachlicher Abschluss:** einen expliziten, an Record/Intent/Artefakte gebundenen Reparaturgrund aus frischen fehlgeschlagenen Befunden an die zuständige Projection/Executoraufgabe übergeben. Missing/stale evidence, technischer Invocationfehler, echte semantische Verletzung und Ownerentscheidung müssen unterscheidbar bleiben. Reparaturversuche erhalten eine endliche Grenze; Erfolg bleibt beim unabhängigen Verifier. Keine Fake-Modelledits und keine neue Core-Semantik.

**Exit:** Erstkandidat verletzt unveränderten Intent; Verifier lehnt ab; erneute Planung liefert begrenzte Reparaturarbeit mit dem relevanten Befund; zweiter Kandidat passiert eigene und Elternprüfung; abschließender Audit meldet weder Arbeit noch erforderlichen Refresh. Der ursprüngliche Fehler bleibt in der Historie.

### F2 — P1: Der Betriebsmodus schließt die Beobachtung von unverändertem Intent noch nicht

**Kategorie:** bewusst begrenzte Mechanismen, fehlende verbindliche Ablaufentscheidung und Ende-zu-Ende-Evidenz. **Confidence: hoch.**

`auditAll=false` plant nach kanonischer Betroffenheit, Bindingänderung sowie fehlenden Dateien/Modusänderung. Ein Inhaltsfehler in einer bereits besessenen Datei wird ohne Inhaltsbeobachtung nicht erkannt. Die Metadaten enthalten Größe, das Scheduling nutzt hier jedoch nur Existenz/Modus; auch ein Größenwechsel löst in diesem Pfad keine Inhaltsprüfung aus. Beleg: [internal/host/canonical_scoped_reconciliation.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_scoped_reconciliation.go:197>). Deshalb ist das erhaltene C8-Negativresultat eine reale Grenze der normalen gezielten Planung.

`auditAll=true` beobachtet Ziele und ermöglicht auch Verifikationsreuse; das wird in [internal/host/canonical_controller_reuse.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_controller_reuse.go:37>) ausdrücklich vorausgesetzt. Gleichzeitig stalen globale Modell-/Revisionsbindungen alte Evidenz konservativ ([internal/host/canonical_impact.go](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/canonical_impact.go:198>)). Das ist sicher, aber ohne klaren Abschluss leicht eine Schleife aus „Work erledigt, neue Revision, Refresh offen“.

**Gegenargument:** Ein selektiver Plan muss nicht unbemerkt das ganze Repository lesen. `unobserved` wird korrekt ausgewiesen; hier liegt kein falsches PASS vor. Gewollte Selektivität und Driftkontrolle müssen aber im Produktablauf zusammenpassen.

**Kleinster Abschluss:** einen expliziten Beobachtungs-/Abschlussmodus für den vereinbarten verwalteten Scope festlegen: gezielte Arbeit plus abschließender Scopeaudit, der unbekannte, ausgeschlossene, unbeobachtete und veraltete Teile sichtbar behandelt. Für den ersten nutzbaren Flow genügt der vorhandene umfassende Audit dieses kleinen Scopes. Eine ausgefeilte lokale Freshnessoptimierung kann warten; globale Bindungen dürfen vorerst konservativ bleiben.

**Exit:** realer lokaler Intentwechsel und eine getrennte Inhaltsdrift bei identischem Intent werden vollständig repariert/verifiziert; unbetroffene gültige Bytes bleiben unverändert. Der letzte Plan ist innerhalb des deklarierten Scopes tatsächlich stabil, nicht nur leer, weil er nichts beobachtet hat.

### F3 — P1 für die Autonomiebehauptung: Forward-Plan und Backward-Gate sind noch überwiegend ein Arbeitsprotokoll

**Kategorie:** teilweise implementiert, semantische und chronologische Prozessschließung fehlt. **Confidence: hoch für die dokumentierte Grenze; keine vollständige Prüfung jedes Legacy-Checks.**

Die Methodik verlangt zurecht vorab Intention, Scope und Plan sowie abschließend den Vergleich der tatsächlichen Änderung mit diesen Verpflichtungen ([docs/operating-methodology.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/operating-methodology.md:29>)). Der ausführbare eingebettete Workflow weist Agents entsprechend an ([internal/host/embedded/resources/workflow-markitect-first-change.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/internal/host/embedded/resources/workflow-markitect-first-change.yaml:12>)). Die Records binden exakte Inputs und Apply schützt die eigene Kandidatenoberfläche. Sie beweisen aber nicht, dass die akzeptierte Intention zeitlich vor einer frei vorgenommenen Repositoryänderung feststand oder dass ein kompletter Projektänderungssatz alle relevanten breiten Regeln durchlaufen hat.

Ein final korrektes Artefakt, ein `context`-Aufruf und ein nachträglich passendes Modell sind keine austauschbaren Nachweise. Die aktuelle Methodik räumt selbst ein, dass kein CLI-Kommando ihre gesamte Abschlussregel durchsetzt (Zeile 34).

**Kleinster Abschluss:** für den ersten realen Arbeitsablauf ein operationales Startartefakt mit Base, Klassifikation, akzeptiertem Intentstand, Scope, delegierter Freiheit und vorgesehenen Prüfungen; am Ende tatsächlichen Diff, Ownership und lokale/Elternevidenz dagegen prüfen. Der unabhängige Verifier darf auch einen unerklärten Extraedit oder eine nachträgliche Modellsegnung ablehnen. Die bereits vorhandenen Plan-/Recordwerte wiederverwenden; eine universelle Authority- oder Obligation-DSL ist dafür nicht nötig.

**Exit:** eine normale Reparatur besteht ohne Ontologieänderung; ein kontrollierter Extraedit bzw. post-hoc geänderter Maßstab besteht nicht als abgeschlossener Change. Beobachtbare Prozessschritte werden von nicht beobachtbarer „mentaler Nutzung“ des Kontexts getrennt.

### F4 — P1 für die Proofsteuerung: C3-insufficient ist kein sauber isolierter Test fehlender entscheidbarer Intention

**Kategorie:** Problem der Versuchskonstruktion, nicht bewiesener allgemeiner Agentendefekt. **Confidence: hoch für die verbleibenden Inputs, mittel für die Zulässigkeit konkreter nicht vollständig eingesehener Agentenkandidaten.**

Der Kontrollfall ersetzt ausschließlich Mission-`guidance` durch „Represent the selected Mission as a clear .NET type.“ und verlangt zwingend Eskalation, angeblich weil Beziehungen, Einheit, Bereich und Capabilitymaximum fehlen: [examples/unknown-ontology/negative-specifications.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/examples/unknown-ontology/negative-specifications.md:9>).

Diese Dinge verschwinden aber nicht vollständig. Die unveränderte Axis-Policy nennt `MissionEvaluator`, Einheit und inklusive Grenzen ([examples/unknown-ontology/definitions/axis-dotnet.policy.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/examples/unknown-ontology/definitions/axis-dotnet.policy.yaml:12>)); die Capability-Policy fordert Namens-/Achsenabgleich und Maximum ([examples/unknown-ontology/definitions/capability-dotnet.policy.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/examples/unknown-ontology/definitions/capability-dotnet.policy.yaml:12>)). Schema und Definitions enthalten weiter Beziehungen, Werte und Zwecke ([examples/unknown-ontology/modules/mission/schema.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/examples/unknown-ontology/modules/mission/schema.yaml:5>), [examples/unknown-ontology/definitions/mission.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/examples/unknown-ontology/definitions/mission.yaml:6>)). Die beiden verbleibenden Policies habe ich zusätzlich direkt am historischen Fixturecommit `c07ca2dce6c0eb2fce565d52eb065a2b3b502826` geprüft; es handelt sich nicht nur um nachträgliche aktuelle Guidance.

Es fehlen insbesondere Teile des exakten Mission-API-Vertrags und der exakten Entscheidungsformulierung. Daraus folgt nicht automatisch, dass **keine zulässige Umsetzung** innerhalb der verbleibenden Freiheit möglich ist. Ein unbekannter Checker-Methodenname ist nur dann zwingende fehlende Intention, wenn API-Kompatibilität ausdrücklich akzeptierte Verpflichtung ist; der Checker darf den entfernten Vertrag nicht heimlich ersetzen.

**Konsequenz:** Die historischen FAILs bleiben unverändert richtige Aussagen über das eingefrorene erwartete Eskalationsverhalten. Sie beweisen wesentlich weniger über „Agent ignoriert unentscheidbare fehlende Bedeutung“, als die aktuelle Interpretation nahelegt.

**Kleinster Abschluss:** neuer, vorab geprüfter Holdout mit zwei fachlich unterschiedlichen zulässigen Entscheidungen, zwischen denen nur der Owner wählen darf, und einer expliziten Eskalationspflicht an dieser Stelle. Dazu ein Freiheitskontrollfall, in dem zwei unterschiedliche korrekte Implementierungen beide akzeptiert werden. Keine Umbewertung alter Records, keine nachträgliche Anpassung des Erfolgsmaßstabs an den beobachteten Kandidaten.

### F5 — P2: Die kanonischen Owner sind benannt, ihre Statusaussagen driften noch

**Kategorie:** Dokumentationsdefekt mit operativem Risiko. **Confidence: hoch.**

Die neue Trennung Vision/Methodik/Measurement/Roadmap ist gut. Die drei Grundlagen werden tatsächlich als Dateiinhalte geroutet: [.markitect/areas/development/repository-boundaries.rule.yaml](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/.markitect/areas/development/repository-boundaries.rule.yaml:12>). Das ist stärker als bloße README-Navigation.

Dagegen erklärt die weiterhin kanonische Constitution Module-driven work discovery pauschal zur Zukunft ([docs/engineering-constitution.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/engineering-constitution.md:118>)), während die Roadmap Module-owned proposals und den Controller bereits beschreibt ([docs/implementation-plan.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/implementation-plan.md:13>)). Die Alpha-Anleitung verweist in Zeile 76 auf „the controller below“, endet aber nach der bisherigen zielbezogenen API und Assurancebeschreibung ([docs/canonical-projections.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/canonical-projections.md:70>), [docs/canonical-projections.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/canonical-projections.md:112>)). Ein Leser findet dort keinen vollständigen Controllerdurchlauf. Auch historische Assurance-Orientierung und aktueller Status sind nicht überall deutlich genug getrennt.

**Kleinster Abschluss:** zeitgebundene Implementierungsbehauptungen im Roadmap-/Usage-Owner halten, veraltete Statussätze in Constitution/Design durch klare Verweise bzw. historische Kennzeichnung ersetzen. Ein einziger ausführbarer Guide muss den aktuellen Weg einschließlich Refresh, Failure/Repair und Abschluss zeigen. Keine neue Dokumentationsschicht.

### F6 — P2, Releasegrenze: Plattformgates und Runtimebetrieb sind noch offen

**Kategorie:** erhaltene technische Fehler/Incomplete-Evidenz, kein semantisches Gegenbeispiel. **Confidence: hoch aus gebundenen Berichten; keine neue vollständige Gatekampagne.**

Die dokumentierten Linux-Pakete bestanden; auf Windows bestand die direkte Go-Suite, während eingebettetes fixed-snapshot Verify mehrfach an der unveränderten Zehn-Minuten-Grenze unvollständig endete. Der Checkpoint hält Quelle und Outcomes getrennt ([docs/validation/proof-capability-checkpoint-2026-10-06.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/validation/proof-capability-checkpoint-2026-10-06.md:65>), [docs/validation/proof-capability-checkpoint-2026-10-06.md](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/docs/validation/proof-capability-checkpoint-2026-10-06.md:83>)). `-count=1` sichert frische Tests, beweist jedoch allein keine Timeoutursache und keinen neuen erfolgreichen Gateabschluss.

Das bleibt vor unterstütztem Windowsbetrieb bzw. Release zu lösen. Es rechtfertigt keine beliebige neue semantische Proofrunde. Diagnose auf eine benannte Hypothese und wenige Versuche begrenzen; nach einer konkreten Korrektur einmal das erforderliche Paket an einer festgehaltenen Revision laufen lassen. Identische rote Läufe ohne neue Hypothese vergrößern die Erkenntnis nicht.

## 4. Wie belastbar das bisherige Proofprogramm ist

Die Programme sind als technische Entwicklungs- und Fehlerfindungsinstrumente wertvoll. Sie haben Vollakquisition, schwache Checks, Kind-/Elternfehler, fehlende Observationsreferenzen und reale Betriebslücken sichtbar gemacht. Das ist Fortschritt. Der Checkpoint ist bemerkenswert zurückhaltend bei Erfolgsbehauptungen.

Die vorliegenden Resultate beantworten aber verschiedene Fragen:

| Evidenzklasse | Was aktuell getragen wird | Was offen bleibt |
|---|---|---|
| Technische Gates | Viele gebundene Boundarychecks und mindestens ältere vollständige Linuxpakete; zwei von mir ausgeführte bestehende .NET-Proposaltests bestanden am Review-Snapshot. | Kein von mir bestätigtes vollständiges aktuelles Windows-/Linux-/Releasepaket. |
| Agentenfeasibility | C3-positive-05 dokumentiert echte fünf Dateien, guarded Apply, festen Check und frischen Verifier. C9 zeigt einen echten rekursiven positiven Ablauf und kontrollierte Elternfehler. | Kein Beleg, dass beliebige Ontologien, Projektformen oder breite Qualitätsregeln zuverlässig abgedeckt werden. |
| Zusätzlicher Verifiernutzen | C10 dokumentiert quantity=1: fixer Checker grün, separater Verifier rot. | Keine Erkennungsrate und keine Unabhängigkeit der Fehlerverteilungen. |
| Zuverlässiger Betrieb | Source enthält Child-first-Ausführung, separate Verifier und konservativen Reuse. | Kleine wiederholte Nutzung ohne Coordinator-Reparatur des Harness, vollständiger C5/C8-Abschluss und stabile erneute Planung fehlen. |
| Vergleichender Nutzen | Die Hypothese und faire Messmethode sind klarer geworden. | Weder Qualitätsvorsprung noch weniger menschlicher Aufwand sind durch die aktuellen Fälle nachgewiesen. |
| Menschliche Akzeptanz | Technische Evidenz und Ownerautorität werden getrennt. | Kein PASS, Record, Agentenreview oder Merge ersetzt eine ausdrücklich erforderliche Akzeptanz. |

Direkt eingesehene Resultate: [experiments/operating-model-proof/runs/c3-positive-05/events/assessment-c3-result.json](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/experiments/operating-model-proof/runs/c3-positive-05/events/assessment-c3-result.json:1>), [experiments/operating-model-proof/runs/c9-positive-02/result.json](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/experiments/operating-model-proof/runs/c9-positive-02/result.json:1>), [experiments/operating-model-proof/runs/c10-orders-quantity1-v5/result.json](<C:/Users/Consiliari/.codex/worktrees/astra-method-assessment/Markitect/experiments/operating-model-proof/runs/c10-orders-quantity1-v5/result.json:1>). Die alten Resultate gelten jeweils an ihrer eigenen Quelle, nicht automatisch am Review-Snapshot.

**Die wichtigsten Repräsentativitätsprobleme sind nicht eine zu kleine Anzahl Mikrotests.** Die positiven Semantikfälle sind stark vorbereitete, kleine, explizit API-geführte .NET-Aufgaben. Das zeigt nützliche Feasibility, aber noch wenig über alltägliche Projektänderungen mit breiten Regeln, Dokumentation, bestehendem Code und Prozesspflichten. Die wiederholten Protokollkorrekturen und manuelle Vorbereitung sind zudem reale Kosten des aktuellen Verfahrens. Nicht jede dieser Kosten wird bleiben; heute dürfen sie aber nicht aus einer Autonomiebehauptung herausgerechnet werden.

Die neue vollständige Observationsliste schützt vor formal unvollständigen Antworten. Sie beweist nicht Verständnis jeder Datei und ist kein eigenständiger Qualitätsmaßstab. Ihre aktuelle Stärke darf nicht zur Forderung führen, jede freie valide Umsetzung in eine exakt vorgegebene Form zu pressen. Negative Tests sollen Verstöße gegen akzeptierte Verpflichtungen treffen, nicht Abweichungen vom Geschmack des Fixtureautors.

## 5. Endlicher Weg zu einem tatsächlich nutzbaren Flow

Ich würde die nächste Arbeitsphase bewusst auf einen kleinen, echten Projektablauf begrenzen. **Die Architektur dabei einfrieren, die beiden fehlenden Rückwege schließen und anschließend verwenden.**

1. **Einmalige Betriebsvereinbarung.** Ein repräsentatives kleines Projekt, zwei fachliche Bereiche, eine gemeinsame Regel, Dokumentation/Guidance und ein Elternvertrag. Festhalten: welche Teile verwaltet sind, was Ownerentscheidung ist, welche Freiheit bleibt, welche präzise und breite Regel geprüft werden und welche finale Beobachtung Pflicht ist. Erfolgs- und Abbruchkriterien vorab festlegen. Keine Erweiterung des Core.
2. **Prozesslücken schließen.** F1-Reparatureingabe mit alter Intention und bounded retry; F2-Abschluss mit Beobachtung und konservativem Refresh; F3-Start-/Abschlussnachweis über die tatsächliche Änderung. Den vorhandenen Controller und Recordpfad nutzen. Für diese Übergänge konkrete bestehende bzw. gezielte neue Regressionstests in der späteren Implementierungsarbeit, keine neue universelle Beweisplattform.
3. **Ein realer vollständiger Intentwechsel.** Agent formuliert innerhalb seiner Delegation die gewünschte Änderung im bestehenden Modell, leitet Kontext/Impact/Fanout ab, ändert echte Repräsentationen, erhält unabhängige lokale und Elternprüfung und repariert einen tatsächlich bzw. kontrolliert entstandenen Fehler. Der letzte aktuelle Scopeaudit muss stabil sein.
4. **Eine echte Reparatur bei identischem Intent.** Inhaltsdrift oder fehlgeschlagene Verifikation ohne Modelländerung. Kein vorbereiteter fertiger Zielkandidat als Ersatz für den Executor. Gültige andere Repräsentationen bleiben unverändert.
5. **Eine kleine vorab festgelegte Nutzungssequenz, beispielsweise sechs Aufträge.** Lokale Änderung; geteilte Regel; Dokumentationsfolge; vergessener Zielbereich; ungeänderte Intention mit Drift; Elternfehler trotz lokal grüner Kinder. Für einen Freiheitsfall zwei zulässige Lösungen, für einen Ambiguitätsfall echte notwendige Ownerentscheidung. Jeder Schritt startet aus dem vorherigen echten Zustand. Protokolliert werden Fehler, Recovery und tatsächliche menschliche Eingriffe, nicht nur Abschlussstatus.
6. **Danach Vergleichsstudie.** Gut ausgestattete konventionelle Agents gegen Markitect auf einem gleichwertigen Projektbacklog. Gleiche akzeptierte Anforderungen, Modelleinstellungen, Werkzeuge und erlaubte Entscheidungen; unabhängige Bewertung von Verhalten, breiten Regeln, Fanout und Integration. Setup, Modellpflege, Harness-/Recoveryarbeit und Review zählen mit. Tokens/Zeit sind sekundär. Nach Tuning frische Aufgaben, keine Wiederverwendung derselben bekannten Kontrollfehler als unabhängige Nutzenmessung.

**Weiterkriterium:** Der vereinbarte Scope wird abgeschlossen, negative Kontrollen werden korrekt behandelt, valide Alternativen werden erhalten, und eine erneute Planung bleibt innerhalb dieses Scopes stabil. Danach zur nächsten Stufe wechseln; nicht wegen weiterer denkbarer Mikrotests bei Stufe 1 bleiben.

**Reparaturkriterium:** Eine konkrete nicht funktionierende implementierte Grenze oder ein fehlender notwendiger Prozessübergang wird behoben und gezielt erneut geprüft. Frühere Ergebnisse bleiben erhalten.

**Eskalations-/Stopkriterium:** ungeklärte akzeptierte Intention oder Autorität; nicht beherrschte Writes/Ownership; wiederholte semantische Misses trotz passender unabhängiger Evidenz; oder ein realer Architekturwiderspruch, der den vereinbarten Einsatzfall verhindert. Technische Incomplete-Ergebnisse stoppen die jeweilige Erfolgsaussage, nicht automatisch die gesamte Produktidee.

## 6. Coverage und Beweisgrenzen dieses Reviews

Ich habe die neue Vision, Methodik und Measurement vollständig gelesen und mit Resetdesign, Roadmap, Constitution, Architecture, Alpha-Usage, Markitect-first, Checkpoint sowie ausgewählten Ownerinputs verglichen. Ownerinputs wurden als Provenienz gelesen, nicht als zusätzliche Ausführungsanweisung. Die vorangegangene Entwicklung wurde anhand Git-Historie und des Snapshotdeltas eingeordnet.

Im Code lag die Tiefe auf minimalem Core/Modulaktivierung, Binding, Context/Impact, selektiver Akquisition, Module-owned Proposals, Controller Execute/Apply/Verify/Reuse, rekursiven Dependencies und Assurance sowie operativen Records. Die neuen Agentexec-/Codexrunner-/Recordstore-/Akquisitionsänderungen wurden als Diff mit ihren Verträgen geprüft. CLI-Dispatch und ausgewählte vorhandene Tests wurden gegen die dokumentierten Operationen gelesen. Legacy-Consumers, alle Plattformdetails, alle hunderttausenden möglichen Eingabekombinationen und sämtliche privaten Rohtranskripte wurden nicht vollständig geprüft. Es gab keine weitere Providerinvokation und keine vollständige neue Gate- oder Benchmarkkampagne.

Gezielter vorhandener Check am Snapshot, Go 1.27.1, `GOENV=off`, `GOTOOLCHAIN=local`, Cache außerhalb des Snapshots:

```text
go test ./internal/modules/dotnet -run 'TestProposeNoopRequiresCurrentRequestBytesAndVerification|TestProposeDriftUnobservedUnknownAndRetiredArtifacts' -count=1 -v
PASS — beide Tests; Paket 0.199s.
```

Diese Tests bestätigen vorhandene No-op/Refresh- und Driftgrenzen; sie sind **kein neu geschriebener Ende-zu-Ende-Nachweis von F1**. F1 beruht auf dem oben aufgeführten vollständigen Übergangspfad und fehlendem Repairinput. Der Snapshot blieb unverändert; dieser Bericht ist die einzige Reviewausgabe.

**Gesamtempfehlung:** Den erreichten Architekturstand behalten. Als nächstes nicht noch eine abstrakte Fähigkeit oder eine strengere allgemeine Beweisregel bauen, sondern den fehlenden Übergang von roter Evidenz zu begrenzter Reparatur und den verbindlichen Abschluss des vorhandenen Flows herstellen. Dann eine endliche reale Sequenz durchführen und das Ergebnis entscheiden lassen.

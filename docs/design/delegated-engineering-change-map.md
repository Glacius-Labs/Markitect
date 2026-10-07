# Architektur-Abgleich: delegierte Engineering-Arbeit

**Untersuchungsbasis:** unveränderlicher Architekt-Kandidat `dc10f5454af381887a2f41004d0987294bd1fe62`. Diese Landkarte ist eine Produkt- und
   Architekturprüfung, keine Änderung am Code, CLI-Vertrag oder veröffentlichten Release. Der finale Gate-Stand dieses Kandidaten ist hier nicht
   bestätigt.

## Zweck und Lesart

Diese Notiz ordnet die bestehenden Markitect-Mechaniken einer möglichen, stärker delegierten Engineering-Organisation zu. Der Abgleich verwendet die
   vorgeschlagenen Government-Namen als öffentliche Begriffe; `Projection` bleibt die interne Bezeichnung des analysierten Quellvertrags. Ob Markitect
   als Produkt vollständig der Regierungsmetapher folgt, bleibt offen. Sie hält stabile Modellverträge, einzelne Aufträge, Laufzeitfähigkeiten,
   Artefakte und Nachweise auseinander. Die Begriffe „Projektordnung“, „Mandat“, „Arbeitsauftrag“, „Zielvorgabe“, „Ausführungsfähigkeit“ und „Bericht“
   sind Vorschläge für die Produktdiskussion, keine beschlossenen Umbenennungen.

Der Kandidat beschreibt Markitect als kanonisches Modell akzeptierter Engineering-Absicht, das ausgewählte Repräsentationen abgleicht. Sein Host
   bietet bereits einen begrenzten Controller-Zyklus. Das ist eine Grundlage für delegierte Arbeit, aber weder ein dauerhaft laufender
   Organisations-Runner noch eine automatische Annahme von Regeln oder Ergebnissen. Alle Quellverweise beziehen sich auf den oben genannten
   unveränderlichen Commit und sind mit `git show dc10f5454af381887a2f41004d0987294bd1fe62:<Pfad>` prüfbar: `docs/vision.md`, Abschnitt „Product
   definition“; `docs/architecture.md`, Abschnitt „Current implementation boundary“; `docs/operating-methodology.md`, Abschnitt „The change and
   reconciliation cycle“.

## Bestehende Verträge auf neue Produktbegriffe abbilden

| Vorgeschlagener Begriff | Vorhandener Vertrag und passender Anteil | Grenze und nötige Erweiterung |
|---|---|---|
| **Projektordnung** | Akzeptiertes Modell aus `Schema`, `Kind`, `Property` und `Definition`. `core.Compile` validiert Struktur, löst explizit deklarierte Referenzen auf und erzeugt ein deterministisches Modell. ( `internal/core/compile.go`, `Compile`) | Core bleibt strukturell und deterministisch. Er entscheidet weder, ob eine Regel sinnvoll ist, noch darf er Agenten orchestrieren oder Modelländerungen annehmen. Zweck, Gesetze und Änderungsbefugnisse liegen oberhalb davon. |
| **Mandat** | Noch kein direkt entsprechendes Objekt. `ProjectionPolicy` steuert Entscheidungen über eine Darstellung; ein `Schema Module` definiert Vokabular; ein `Projection Module` bietet Zielkompetenz. Keines davon ist eine dauerhafte, querliegende Zuständigkeit für Architektur, Sicherheit oder Betrieb. (`docs/development/modules.md`, Abschnitte „Installable Modules“ und „Responsibility map“) | Neues versioniertes Mandats- und Zuständigkeitsmodell, falls Ressorts in jeder Änderung auftreten sollen. Ein Mandat braucht seinen Geltungsbereich, Regeln, erforderliche Inputs, Review-Verfahren und Umgang mit Unsicherheit. Nicht in `ProjectionPolicy` umdeuten. |
| **Arbeitsauftrag** | `CanonicalControllerProposal` plant betroffene Darstellungsarbeit; `CanonicalReviewedRun` enthält die ausgeführten Kandidaten und ist an den Plan gebunden. Der Dotnet-Proposal-Typ `ExecutorTask` beschreibt begrenzte konkrete Arbeit. (`internal/host/canonical_controller.go`, `internal/modules/dotnet/proposal.go`) | Ein Run oder Plan ist ein einzelner Versuch mit festen Revisionen, kein langlebiger Backlog-Eintrag. Story, Idee, Priorität, Abhängigkeit, Delegation und Wiederaufnahme bräuchten einen separaten Auftrag-/Queue-Vertrag. |
| **Zielvorgabe** | Der kanonische Quellvertrag benennt eine gewünschte Darstellung, ihren exakten semantischen Umfang, Ziel und Policies; die Bindung an installierte Umsetzung bleibt separat. Im Kandidaten heißt dieser interne Typ `Projection`. (`internal/host/canonical/binding.go`, `ProjectionRequest`, `ProjectionBinding`) | „Zielvorgabe“ ist der vorgeschlagene konsistente öffentliche Begriff. Der dauerhafte Sollvertrag bleibt von Mandat und Einzelauftrag getrennt und nützlich, wenn Ministerien oder Executors wechseln. Ob Markitect insgesamt zur Regierungsmetapher wird, bleibt offen. |
| **Zielartefakt** | Exakte Pfade, Modi und Bytes; `ProjectionRecord` bindet erstellte, geänderte und beibehaltene Artefakte an Modell, Plan, Modul, Snapshot und Zielzustand. Ownership-Index zeigt Drift, Unbekanntes, Ausschlüsse und Konflikte. (`internal/host/records/records.go`, `ProjectionRecord`, `OwnershipIndex`) | Zielinventar und Ownership sind überprüfbare Tatsachen zum erfassten Scope. Sie beweisen keine vollständige Repo-Abdeckung. Der Ledger ist ein technischer Append-/Auswahlmechanismus und verleiht keinem Agenten oder Review automatisch Autorität. |
| **Ausführungsfähigkeit** | Exaktes installiertes Projection Module, Pin und eindeutiger interner Entry-Point; Host statisch komponiert die Fähigkeiten. Installation allein erzeugt keine Zielpflicht oder Artefakte. (`internal/host/canonical/binding.go`; `docs/design/canonical-projection-reset.md`, „Strict Module types“) | Module nicht automatisch in Ministerien umbenennen. Ein .NET- oder Markdown-Modul kann eine Werkzeugfähigkeit sein. Ein Ministerium braucht zusätzlich Mandat, Zuständigkeit und Review-/Entscheidungsleistung; es kann mehrere Fähigkeiten nutzen. |
| **Bericht / Nachweise** | `CanonicalReviewedRun`, Materialisierungsbericht, `ProjectionRecord`, `VerificationResult` und Run-Receipts binden konkrete Ausführung, Kandidaten, Checks und feste Eingaben. (`internal/host/canonical_controller.go`; `internal/host/records/records.go`) | Neues separates Regierungsprotokoll wäre nötig für Zuständigkeitsentscheidung, Einwand, Zustimmung, Unsicherheit, Begründung und Mandatsversion. Ein `passed` Verifier-Ergebnis ist keine Annahme und kein authentifizierter Ministerentscheid. `controller-apply` materialisiert Bytes; es ist keine Regierungsannahme. Vor finaler Zustimmung dürfen aktive Zielbytes nicht geändert werden. |
| **Unabhängige Prüfung** | Host ruft konfigurierten Verifier frisch und getrennt vom Executor auf; Assurance komponiert lokale und Elternnachweise in einem expliziten DAG. (`internal/host/canonical_controller_verification.go`; `internal/host/assurance/assurance.go`) | Frische Invocation und gebundene Eingaben belegen den geprüften Vorgang, nicht Unabhängigkeit im Sinne verschiedener Personen/Organisationen, semantische Suffizienz oder menschliche Zustimmung. Ministerielle Zustimmungen wären zusätzliche Ergebnisse. |

## Beibehalten, erweitern und trennen

**Beibehalten:** Pure Core-Compilation, explizite Quellrevisionen, Module-Pins, getrennte Konfigurationsbindung, begrenzte Input-Selektion,
   Plan-Digests, Candidate-Bytes, geschützte Pfade, guarded Apply, getrennte Materialisierungs- und Verifikationsergebnisse, konservative Behandlung
   von Unbekanntem sowie explizite Eskalation. Diese Mechaniken schaffen Nachvollziehbarkeit und verhindern, dass ein erfolgreiches Tool
   stillschweigend zur neuen Wahrheit wird. Architekturgrenzen sind in `docs/development/modules.md`, „Dependency direction“ und „Responsibility map“
   am Untersuchungs-Commit festgehalten.

**Erweitern:** Host braucht, falls delegierte Government-Funktionen gewünscht sind, Verträge für Mandate und Zuständigkeit, laufende Aufgaben/Backlog,
   Priorisierung und zulässige Delegation, ressortübergreifende Review-Aufträge sowie ein separates Entscheidungsprotokoll. Diese Ebene entscheidet,
   wer für eine Änderung Stellung nimmt und wann die Änderung angenommen werden darf. Sie ist Orchestrierung und Autorität, keine neue Schema- oder
   Projektionssemantik im Core.

**Aufteilen:** Dauerhafte Zielpflicht (heute kanonische Projection), ressortweites Mandat (neu), konkreter Arbeitsauftrag (heute Plan/Run plus künftig
   Backlog) und Ausführungsfähigkeit (heute Module/Entry-Point) haben unterschiedliche Identität, Laufzeit und Änderungsfrequenz. Ein Modulwechsel
   darf die Zielpflicht nicht ändern; eine neue Aufgabe darf kein Mandat erfinden; eine Zustimmung darf keine Artefaktmaterialisierung behaupten.

## Grenzen des aktuellen Quellkontexts und der Assurance

Der Executor-Kontext wird aus der ausdrücklich gewählten Projection-Request gebaut. Er enthält ausgewählte Definitions, die über ausgewählte
   Referenzen im begrenzten Kontext liegen; `BuildCanonicalAgentContext` begrenzt den Umfang und meldet ausgelassene oder externe Kanten. Das ist kein
   Versprechen, beliebige transitive Abhängigkeiten oder den ganzen Quellcode zu sehen. Ministerien bräuchten explizite, begründete Scope-Regeln für
   zusätzliche Artefaktbytes; der Host muss sie erfassen und an konkrete Eingaben binden, statt Kontextvollständigkeit zu unterstellen.
   (`internal/host/canonical_agent_context.go`; `internal/host/canonical/binding.go`, `ProjectionRequest`)

Das Assurance-DAG verbindet deklarierte Zielverträge, Scopes, Checks und Elternprüfung. Der Audit zählt kanonische Zielverträge und Nachweise in den
   konfigurierten Roots/Targets. Das liefert brauchbare Buchführung für den deklarierten Scope, aber keine Garantie, dass die Projektordnung
   vollständig ist oder jedes fachlich zuständige Ressort entdeckt wurde. (`internal/host/canonical_controller.go`, `CanonicalAssuranceScope`;
   `internal/host/canonical_controller_audit.go`, `AuditCanonicalController`)

Keine dieser Digest-, Ledger- oder Freshness-Regeln ersetzt Autorität: der Host guardet Schreibvorgänge, prüft erwartete Plan-/Ledger-Stände und
   trennt Materialisierung von Verifikation. Der Kandidat bindet **technisch**, welche Bytes und Runs zusammengehören; er authentifiziert nicht, wer
   den Plan freigegeben hat, und trifft keine automatische Annahmeentscheidung. Bestehende `--write`/Review-Grenzen bleiben erhalten, bis ein explizit
   delegiertes Autoritätsmodell samt prüfbarer Zuständigkeit und Delegation vorliegt. (`docs/usage.md`, Abschnitt „Canonical controller actions
   (source-only alpha)“ am Untersuchungs-Commit; `docs/architecture.md`, Abschnitte zu Controller und Records am Untersuchungs-Commit)

## Laufzeit, Adoption und Rückkopplung

Der heutige Controller hat explizite Aktionen `controller-propose`, `controller-execute`, `controller-apply`, `controller-verify` und
   `controller-audit`; volle immutable Revisions, extern gespeicherte Runtime/Ledgerdaten und aktuelle Bindungen sind Teil des Vertrags. Proposal
   bleibt lesend; Execute produziert Kandidaten; Apply benötigt Reviewed Run, exakten Digest und `--write`; Verify schreibt separat Nachweise. Das ist
   ein endlicher kontrollierter Zyklus, kein autonomer Backlog-Operator. (`docs/usage.md`, Abschnitt „Canonical controller actions (source-only
   alpha)“ am Untersuchungs-Commit; `internal/host/canonical_controller_execution.go`, `ExecuteCanonicalController` und `ApplyCanonicalController`)

Assurance kann unabhängige Zielaufgaben entlang deklarierter Abhängigkeiten child-first komponieren; der Operating-Method-Text erlaubt parallele
   unabhängige Ziele, wenn Ownership und Abhängigkeiten es zulassen. Diese Mechanik plant aber keine Prioritäten aus Stories, läuft nicht
   kontinuierlich und stellt keine sichere Wiederaufnahme eines dauerhaft arbeitenden Agenten her. Automatische Lease-Recovery ist im Controller-Code
   ausdrücklich nicht vorhanden. Ein Runner/Scheduler mit persistenter Queue, Abbruch-/Retry-Politik und überprüfbarer Wiederaufnahme bleibt
   zusätzliches Runtime-Design. (`internal/host/assurance/run.go`, `Execute`; `internal/host/canonical_controller_execution.go`;
   `docs/design/standard-operating-model.md`, „One reusable operating cycle“)

Brownfield-Vorschlag bleibt strikt von Kanonisierung getrennt. Die Adoption-Hand-off erfasst owner-selektierte Git-Blobs; Copy Me und
   Brownfield-Inference arbeiten nur mit den gebundenen Belegen. Sie entdecken keine Pfade und nehmen keinen Kandidaten an. `RunBrownfieldInference`
   liefert eine reviewbare Hypothese, aber kein angenommenes kanonisches Modell. So kann eine spätere Regierungsinstanz die Ordnung mit Agenten
   entwerfen und Verbesserungen vorschlagen, aber jede Modellpflege braucht eine explizite Befugnis: entweder Owner-Entscheidung oder ein vorab
   abgegrenztes Delegat mit Bedingungen, Versionierung, Evidenz und Eskalation. Nie darf beobachteter Ist-Zustand allein Soll werden.
   (`docs/design/selective-adoption-handoff.md` am Untersuchungs-Commit; `internal/host/canonical_inference.go`, `RunBrownfieldInference`;
   `docs/vision.md`, Abschnitte „Product definition“ und „Human, Markitect and agent responsibilities“ am Untersuchungs-Commit)

## Kleinster sinnvoller Vertikalschnitt nach Abschluss des Architects

Diese Reihenfolge ist eine Bewertungsgrundlage, keine bereits genehmigte Umsetzung. Alle Angaben zu internem Code und Dokumentation beziehen sich auf
   den unveränderlichen Untersuchungs-Commit `dc10f5454af381887a2f41004d0987294bd1fe62`.

1. **Kandidat einfrieren und Grenzen prüfen.** Architect-Arbeit abschließen, finalen Candidate-SHA und Gates ermitteln, Unterschiede zu
   `dc10f5454af381887a2f41004d0987294bd1fe62` dokumentieren. Die vorliegende Landkarte auf diesen finalen Snapshot neu binden.

2. **Begriffe und
   Authority-Modell entscheiden.** Ein kleines reales Projektmodell aufteilen in Projektordnung, ein versioniertes Ressortmandat, eine dauerhafte
   Zielvorgabe, konkrete Aufgaben und Ausführungsfähigkeiten. Festhalten, welche Routineänderungen delegierbar sind und was weiterhin eskaliert.

3. **Isolierter Vertragsversuch.** Einen Backlog-Eintrag aus einem begrenzten Brownfield-Scope in einem eigens dafür gewählten Testkabinett
   bearbeiten. Bei einer Änderung im bestehenden Projekt bleibt dessen Kabinett eingefroren; Ressorts werden nicht nach Einwänden entfernt. Aus einem
   Auftrag eine bestehende Zielvorgabe ausführen und gültige Artefakte erhalten. Kandidat nur in einem isolierten, entbehrlichen Candidate-Worktree
   bereitstellen. Hier darf guarded Apply vor den Ressortstimmen zur Kandidatenmaterialisierung dienen; der produktive Projektstand bleibt
   unverändert

4. **Verify und Mandatsprüfung.** Nach isolierter Materialisierung wird der feste Kandidat mit `controller-verify` frisch geprüft.
   Jedes Ressort des vor dem Auftrag gepinnten Kabinetts beurteilt denselben Kandidaten und dessen Nachweise anhand seines Mandats. Es dokumentiert
   Zustimmung, begründeten Einwand oder Unsicherheit. Schweigen, Fehler und Auslassung sind keine Zustimmung; kein Ressort wird nach einem Einwand
   entfernt. Jede Kandidatenänderung macht alte Verifier- und Ressortnachweise ungültig.

5. **Lernrücklauf testen.** Eine Umsetzungserkenntnis erzeugt
   zuerst eine Beobachtung und dann optional einen Modelländerungsentwurf mit Gegenbelegen, Unsicherheit und Impact. Reine Drift repariert die
   Umsetzung; nur eine ausdrücklich delegierte Modelländerung ändert den Sollvertrag.

6. **Finale Zustimmung vor Übernahme prüfen.** Erst nach
   Zustimmung aller Ressorts darf der exakt gebilligte Kandidat in den aktiven Projektstand übernommen werden. Diese Übernahme muss Kandidatendigest,
   Kabinett, Mandatsversionen und aktuellen Zielzustand erneut prüfen. Der bestehende `controller-apply` materialisiert im angegebenen Repository und
   ist kein separates Promote-/Übernahmeverfahren; er prüft kein Regierungsprotokoll. Candidate-Bindung, Writer und Übernahmeguard sind damit eine
   offene Fähigkeit. Bis sie vorhanden und verifiziert sind, darf eine manuelle Kopie nicht als geschützte Government-Übernahme gelten. Aktive
   Zielbytes vor Einstimmigkeit zu ändern, wäre außerhalb des Trials.

7. **Erst danach skalieren.** Bei Nutzen mit mehreren Ressorts und echtem Interessenkonflikt wiederholen. Anschließend entscheiden, welche internen
   Typnamen geändert werden und ob Queue, Runner und Wiederaufnahme eine getrennte Ausbaustufe sind.

Die vorhandenen Live-Pilots belegen einen begrenzten Change-/Repair-/Verify-/Audit-Ablauf in synthetischen oder kontrollierten Fällen. Sie sind kein
   Nachweis für Backlog-Autonomie, vollständige Brownfield-Erkundung, automatische Gesetzgebung, zuverlässige Ministeriumsentscheidungen oder
   reduzierte menschliche Aufsicht. Release- und Nutzenaussagen bleiben an finale Source-Gates und wiederholte reale Aufgaben gebunden.
   (`docs/implementation-plan.md`, „Standard operating model checkpoint“ und „Canonical reset and projection execution“; `docs/vision.md`, „Current
   basis and unproven benefit“)

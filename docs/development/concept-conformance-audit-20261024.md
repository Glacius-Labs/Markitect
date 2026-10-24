# Markitect V1 – Konzepttreue und dominante Probleme

**Prüfdatum:** 24. Oktober 2026  
**Fixierter Quellstand:** `6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998`  
**Vergleichsbasis:** Gemeinsame Konzeptentscheidungen D1–D9, konsolidierter Plan v0.17 und danach ausdrücklich akzeptierte enge S0–S5-Abgrenzung.  
**Art:** Unabhängige statische Code- und Vertragsprüfung. Keine Implementierungsfreigabe und keine Änderung am Repository.

## 1. Ergebnis

Die tragende Architektur ist tatsächlich umgesetzt: erhaltenes Modellfundament, eigenständige Projektvorbereitung, gemeinsamer hierarchischer Run-Executor, direkte Apply-Komposition, frische Sessions und Arbeitsumgebungen, dauerhafte Kandidaten und Ergebnisse, Shared Notes sowie Publication nach Run-Ende.

Das ist **nicht gleichbedeutend mit vollständiger Umsetzung aller vereinbarten Arbeitsweisen**. Konkrete Unterschiede bestehen bei den Kommunikationswegen, der Behandlung ungültiger Kinderübergaben und der Bindung fachlicher Abschlüsse an einzelne technische Nachrichten. Der Harness erfüllt Reproduzierbarkeit und native Instruktionsverwendung, aber nicht vollständig die ursprünglich angestrebte modulare, datei-/skillbasierte und gezielt nachladbare Arbeitsumgebung.

Zudem bleiben der bestätigte Context-Verlust im Publication-Guard und ein statisch nachvollziehbarer Größenkonflikt zwischen Notes-Speicherung und Tooltransport. Diese Befunde sind von fehlender nativer Qualifikation und subjektiven Refactoring-Wünschen zu unterscheiden.

**Empfehlung:** Kein erneuter Totalneubau. Den bestehenden Kern bewahren, konkrete Konzeptabweichungen explizit entscheiden beziehungsweise korrigieren und die betroffenen Übergänge mit gezielten Negativtests absichern. Ein grüner Library-Lauf ersetzt keine Prüfung der Organisations- und Verantwortungsregeln.

## 2. Quellen und Grenzen

Der ursprüngliche Plan `markitect-v1-plan-v0.17.md` wurde erneut gelesen. Relevant sind besonders §4 Verantwortlichkeiten, §5.2 Zusammenarbeit, §6.2 Kontrollpunkte, §8 Harness/Provider und §9 Notes/Verify. Frühere Vorschläge zur Extract-/UI-Erhaltung sind durch spätere Owner-Entscheidungen überholt und werden hier nicht als unerfüllte Anforderungen gezählt.

Der aktuelle Main wurde zu Beginn und am Ende der Prüfung auf denselben SHA aufgelöst. Zentrale Produktionsdateien wurden direkt über GitHub gelesen, insbesondere Run, Workflow, Apply, Harness, Codex-/Claude-Adapter, Project-Validierung, Runstore, Publication und Git-Guards. Zusätzlich wurden Git-Unterbäume der bewahrten Grundlagen verglichen und die dokumentierten Qualifikationsgrenzen gelesen.

In dieser Prüfung wurden **keine Go-, Container- oder Provider-Tests ausgeführt**. Der lokale Go-Stand ist älter als die Repository-Anforderung; ein unabhängiger Checkout stand nicht bereit. Dokumentierte native Erfolge werden als Repository-Evidenz wiedergegeben, nicht als selbst wiederholte Versuche. Die vollständigen lokalen Transkripte und Evidence-Pakete auf dem Windows-Host wurden nicht eingesehen. Dies ist keine vollständige Race-, Sicherheits- oder Zeile-für-Zeile-Prüfung des gesamten Repositorys.

Die Kategorien bedeuten:

- **Umgesetzt:** Im geprüften Produktpfad direkt erkennbar.
- **Abweichung:** Konkreter Unterschied zu einer vereinbarten Arbeitsweise.
- **Teilweise/anders:** Ein Teilziel ist erreicht, die ursprünglich vorgesehene Form oder Erweiterbarkeit nicht vollständig.
- **Risiko:** Statisch begründete Fehlerbedingung; keine Behauptung eines neu ausgeführten Laufzeitfehlers.
- **Nachweisgrenze:** Implementierung oder Komponentenbeweis vorhanden, breiterer Produktnachweis noch nicht erbracht.

## 3. Anforderungsmatrix

| Konzeptanforderung | Befund im fixierten Code | Einordnung |
|---|---|---|
| Modell, Compiler, Snapshot, Organisation und Realization bewahren | Vier Git-Unterbäume sind gegenüber `447ac3b1` identisch | Umgesetzt |
| Model→Artifacts; keine Modelländerung als Reparaturausweg | `Prepared.ValidateCandidate` bindet Modell-/Kontrollquellen; Registry darf sich ändern | Umgesetzt |
| Deterministische Prüfung vor Agentenstart | Gültiges `Prepared` ist Voraussetzung für Command und Executor | Umgesetzt |
| Ein Executor für Assessment, Implement, Verify | Alle drei verwenden `run.Executor.Execute` | Umgesetzt |
| Gesamte modellierte Hierarchie in jedem Mandat | Environments und Sessions werden für alle Topologie-Manager angelegt; Kinderarbeit/-review wird erwartet | Umgesetzt |
| Run unabhängig von Apply | Eigener Execute-Vertrag, Persistenz und Closure im Run | Umgesetzt |
| Workflow koordiniert nur Runs | Sequenz ist klein; `ValidateClosed` kennt zusätzlich konkrete Ressourcenbezeichnungen | Weitgehend; begrenztes Verantwortungsleck |
| Frische Gespräche und Workspaces je Run | Neue Open-Aufrufe, native Threads und private Homes je Execute | Umgesetzt |
| Ein Git Workspace Service je Run | Run öffnet und schließt seine WorkspaceService-Instanz | Umgesetzt |
| Parent delegiert/reviewt nur direkte Kinder | Entsprechende Parent-Prüfungen in `Delegate` und `Review` | Umgesetzt |
| Fragen und Abstimmung entlang der Hierarchie | `Ask` erlaubt jeden anderen Manager desselben Runs | Abweichung A1 |
| Kindercheck als Diagnose an den Vater | Bei ungültigem Submitted-Stand korrigiert der Host im Kind weiter, bevor dessen Handoff ausgeliefert wird | Abweichung A2 |
| Erfolgreicher Root benötigt gültigen Kandidaten | Implement-Check, finale Aggregation, feste Kandidatenbindung | Umgesetzt |
| Neue Implement-/Verify-Runs nach Reject | Workflow erzeugt neue Run-IDs und gibt tatsächliche Findings/Kandidaten weiter | Umgesetzt; öffentlicher nativer Zweignachweis offen |
| Technischer Fehler, Reject und Modellblocker getrennt | Technischer Status und semantische Entscheidung sind getrennte Werte | Umgesetzt |
| Kurzer Auftrag plus vorbereiteter Harness | Task/Feedback kompakt; Hintergrundwissen wird als umfangreicher Base-/Developer-Kontext injiziert | Teilweise/anders A4 |
| Provider-native Dateien/Skills und generischer Fallback | Codex verwendet native Instruktionsparameter und MCP; automatische Projektdateien sind deaktiviert; kein produktiver generischer Fallback nachgewiesen | Teilweise/anders A4 |
| Autonomie ohne fachliche Turn-Prozessmaschine | Alte Turn-Engine entfernt; aktive Nachrichten benötigen trotzdem jeweils ein neues `judge` | Spannung A3 |
| Ownership und Registry | Statische Zuordnung und Schutz kontrollierter Eingaben; tatsächliche Zuständigkeitsbefolgung bleibt Parent-/Verify-Review | Entspricht der ausdrücklich erlaubten Grenze ohne per-write ACL |
| Dauerhafte Kandidaten/Resultate nach Ressourcenende | Retain vor Cleanup, Terminal Save/Read danach; keine Live-Session im Ergebnis | Umgesetzt |
| Shared Notes über Runs/Commands | Unabhängige persistente Records, Append und Read | Umgesetzt; Abrufgrenze R2 |
| Normale CLI-Authentifizierung ohne API-Key-Pfad | Codex akzeptiert Account-Login und provisioniert über den Managercontainer | Umgesetzt |
| Claude und weitere Provider | Claude.Open gibt stets unavailable zurück; generische Instruktion ersetzt keinen Runtimeadapter | Explizite Fähigkeitsgrenze, nicht lediglich fehlender Test |
| Publication aus gespeicherter Zustimmung | Gespeicherter Verify-Result wird verglichen; Kandidat erneut geöffnet und geprüft | Umgesetzt |
| Sicherer Umgang mit Nutzeränderungen | Drift wird abgelehnt; Teilübertragung wird berichtet; keine automatische semantische Zusammenführung | Akzeptierte V1-Vereinfachung |
| Endliche Fristen bis Publication-Ende | Context geht in älteren Guard-/Git-Hilfen verloren | Bestätigter Mangel R1 |
| Kleine, kohärente Komponenten | Gute Paketgrenzen; zentrale Run-Datei bündelt viele Lebenszyklus-/Kommunikationszustände | Qualitätsrisiko, kein Nachweis eines generellen Architekturversagens |

### Unabhängiger Nachweis des bewahrten Fundaments

Die folgenden Git-Tree-Identitäten stimmen zwischen Ausgangsstand `447ac3b152de79b52de80513e14dc94349a060bd` und geprüftem Main überein. Damit ist nicht nur eine behauptete semantische Äquivalenz, sondern derselbe jeweilige Dateibaum belegt.

| Verzeichnis | Identischer Git-Tree |
|---|---|
| `internal/model` | `ded9b19b4e7572b2fe04adebf4e9ed0094943fa8` |
| `internal/snapshot` | `c2d6a7249fc4c6244dc105f1a04a7232ba593e9a` |
| `internal/organization` | `06731ff5babb4dd33620780392919e82305665f0` |
| `internal/realization` | `6f65fddf872ec3c1d35fb910ba782990c18d9204` |

Dies behauptet keine unveränderte Runtime oder unverändertes kanonisches Selbstmodell. Deren Anpassung an die neue Architektur war ausdrücklich beabsichtigt. Quellen: [S01], [S02].

## 4. Konkrete Abweichungen

### A1 – Geschwister und entfernte Manager können direkt miteinander sprechen

**Soll:** Die ursprünglichen Entscheidungen verlangen Kommunikation entlang der Hierarchie. Der v0.17-Plan nennt Geschwisterkoordination über gemeinsame Eltern und hält erforderliche Kommunikation auch im Notes-Abschnitt hierarchisch.

**Ist:** `run/inbox.go: ask` kontrolliert Existenz des Empfängers, ungleichen Absender und nichtleeren Text. Eine Parent-/Child-Beziehung wird nicht geprüft. `codexappserver/tools.go` beschreibt das Tool entsprechend als Frage an einen anderen Run-Manager. Dagegen erzwingen Delegation und Review tatsächlich die direkte Eltern-Kind-Beziehung. [S03], [S04], [S05]

**Konsequenz:** Der Baum ist für Aufgaben und Integration verbindlich, für Fragen nur eine Auswahl möglicher Adressen. Ein Geschwister kann das andere direkt fragen; ein Enkel kann Ebenen überspringen. Das ist keine externe Kommunikation mit dem Owner, aber es ist eine andere interne Organisationsregel.

**Warum:** Die Umsetzung erlaubt dies ausdrücklich. Eine dokumentierte Owner-Freigabe für die Lockerung wurde in den geprüften Vorgaben nicht gefunden. Das asynchrone Ask/Await/Reply-Protokoll ist mit den nachgewiesenen Providergrenzen begründet. Daraus folgt jedoch nicht, dass sämtliche Manager direkt erreichbar sein müssen: Auch hierarchische Nachbarn können asynchron miteinander sprechen.

**Empfehlung:** Die erlaubten Nachbarschaftsbeziehungen aus der vorhandenen Topologie ableiten und testen oder die gelockerte Regel ausdrücklich vom Owner akzeptieren lassen. Keine neue allgemeine ACL-/Routingplattform. Das Gegenprüfungsbeispiel muss zwei Geschwister sowie Root/Enkel umfassen, nicht nur Root und direktes Kind.

### A2 – Ungültige Kinderstände werden vor der Elternübergabe zurückgehalten

**Soll:** §6.2 des v0.17-Plans verlangt bei jeder Kinderübergabe die Diagnose an den Vater, mit Korrektur und Abstimmung innerhalb des Runs. Der endgültige Root-Stand bleibt ein hartes Gate.

**Ist:** `execution.work` erstellt nach `submitted` einen Snapshot und führt `x.check` aus. Bei `ErrCandidateInvalid` sendet der Host Findings direkt in eine neue Nachricht desselben Kindes und wiederholt die Schleife. Ein normales Handoff erreicht den Vater erst nach erfolgreicher Prüfung. Die Handoff-Struktur enthält keinen eigenständigen Bericht dieser vorherigen fehlgeschlagenen Prüfung. [S05], [S06]

**Konsequenz:** Lokale Selbstkorrektur ist möglich, aber der vorgesehene automatische Informationsfluss zum Vater fehlt. Das Kind kann zwar selbst fragen oder ein Reject liefern; es ist nicht von jeder Kommunikation abgeschnitten. Der Host entscheidet trotzdem, dass ein invalides Submitted-Ergebnis zunächst beim Kind bleibt. Bereichsübergreifende Ursachen können dadurch später beim Verantwortlichen ankommen.

**Warum:** Dieses Verfahren schützt die Eigenschaft „normal abgegebenes Implement-Handoff ist technisch gültig“. Das ist eine plausible technische Motivation, aber nicht dieselbe Verantwortungsverteilung wie „Übergabe plus Diagnose an den Vater“. Der dokumentierte Fehler, bei dem ein vollständiger Projekttest in einer unvollständigen Kinderkopie lief, zeigt, dass diese Grenze praktisch relevant ist. [S07]

**Empfehlung:** Vor einer pauschalen Konzeptabnahme explizit entscheiden. Eine Angleichung muss nicht ungültiges Mergen erlauben: Der Vater kann Diagnose und Stand sehen, während erfolgreiche Integration und Root-Abschluss weiterhin gesperrt bleiben. Kein neues mehrstufiges Freigabeprotokoll einführen.

### A3 – Fachlicher Abschluss hängt noch stark am Ende technischer Nachrichten

**Soll:** Runs und Manageraufträge, keine fachliche Turn-Maschine und kein report-to-continue. Eine kleine explizite semantische Entscheidung ist erlaubt; sie soll nicht zu einem Ritual für gewöhnliche Kommunikation werden.

**Ist:** Jede aktive `Exchange`-Fortsetzung benötigt eine frische Entscheidung. Im Codex-Adapter wird das vorherige `judgment` pro Nachricht zurückgesetzt. Fehlt nach der Nachricht ein semantisches Resultat, beendet `work` den Auftrag mit Fehler. Die allgemeine Harness-Regel verlangt die neue Entscheidung ausdrücklich auch dann, wenn eine aktive Fortsetzung nur ausstehende Fragen/Antworten behandelt. Erst Nachrichten nach bereits fixierter Übergabe haben eine Antwort-ohne-neues-Urteil-Sonderbehandlung. [S05], [S08], [S09]

Die kurze Beschreibung des `judge`-Tools sagt zugleich allgemein, gewöhnliche Frage-/Antwortnachrichten benötigten kein neues Urteil. Der Unterschied zwischen „aktiver Fortsetzung“ und „später Nachricht nach fixiertem Handoff“ wird dort nicht ebenso präzise gemacht wie in den allgemeinen Regeln. Das ist zumindest eine vermeidbare Mehrdeutigkeit des gelieferten Protokolls. [S04]

**Konsequenz:** Der Agent muss die technischen Phasen des Hosts korrekt unterscheiden. Er kann fachlich korrekt antworten und trotzdem als technisch gescheitert gelten. Mehrere historische Fehlläufe betreffen genau alte Aufgaben, unerwartetes Warten, bereits registrierte Reviewentscheidungen und fehlende erneute Urteile. [S07]

**Warum:** Die Regeln sollen veraltete Freigaben verhindern und Provider-Nachrichten mit fixierten Git-Handoffs synchronisieren. Der Hintergrund ist legitim. Die bisherige Stabilisierung hat aber zusätzliche Instruktionen und Zustandsunterscheidungen aufgebaut. Das ist keine Rückkehr der alten produktiven Turn-Engine, wohl aber eine konzeptionelle Spannung zur gewünschten einfachen Zusammenarbeit.

**Empfehlung:** Auftrag, Nachricht und Gültigkeit des letzten fachlichen Abschlusses getrennt beurteilen. Eine tatsächlich geänderte Aufgabe, ein geänderter Kandidat oder neue verpflichtende Findings dürfen keine alte Zustimmung erben. Eine rein kommunikative Antwort sollte andererseits nicht unnötig erneut dieselbe Zustimmung formulieren müssen. Erst Gegenbeispiele und aktuelle Zustandsübergänge prüfen, dann gezielt vereinfachen; `judge` nicht blind entfernen und natürliche Prosa nicht als Erfolgsbeweis benutzen.

Ein kleiner zusätzlicher Hinweis auf vermeidbare Protokollarbeit: Native Findings müssen ein `manager`-Feld enthalten, obwohl der Host im normalen Arbeits-/Reviewpfad den tatsächlichen Manager anschließend selbst einträgt. Das ist kein Hauptblocker, aber eine gute Probe für den Grundsatz „technisch Bekanntes nicht erneut vom Modell abfragen“. [S04], [S05]

### A4 – Harness als native Instruktionsinjektion statt vollständig modularer Arbeitsumgebung

**Soll:** Deterministische fünfteilige Zusammensetzung, kurze Aufgabenprompts, kontrollierte native Nutzung, kompakte Karte und gezielter Kontextzugriff. Ursprünglich wurden AGENTS.md/CLAUDE.md, Skills, Rules, Scripts und ein generischer Fallback konkret besprochen. v0.17 verlangt nicht jede native Funktion, aber eine tatsächlich nutzbare vorbereitete Umgebung.

**Ist:** `harness.Document` enthält zwei Strings, `Base` und `Developer`. Der Builder fügt vollständige Topologie, Ownership, Manager, Mandat, Findings, sämtliche ausgewählten Definitionsbodies/Felder/Provenienz und ausgewählte Instruktionsdateien zusammen. Der Codex-Adapter übergibt diese beim Threadstart über native Base-/Developer-Instructions. Die eigentliche Arbeitsnachricht besteht dagegen korrekt nur aus Task und Feedback. MCP-Werkzeuge und private Umgebung sind real integriert. [S08], [S09], [S10]

Automatisches Laden von Projektinstruktionen wird ausdrücklich deaktiviert (`project_doc_max_bytes=0`, leere Fallback-Dateinamen). Der private Codex-Home enthält die explizit provisionierten Login-Daten; im überprüften Ausführungspfad entsteht keine entsprechende generierte AGENTS-/Skill-Struktur als primäre Wissensquelle. Ein ausführbarer generischer Providerfallback ist nicht vorhanden. [S11]

**Bewertung:** Das ist kein „fehlender Harness“ und keine bloße Attrappe. Native Instruktionsparameter sind reale Providermechanismen, die zentralen Regeln bleiben von Task-Nachrichten getrennt. Die Abschaltung unkontrollierter Repository-Instruktionen ist als Prioritäts-/Determinismusmaßnahme nachvollziehbar.

Trotzdem ist die ursprüngliche Idee „Karte und abrufbare Arbeitsmittel statt große dauerhafte Kontextanleitung“ nur teilweise umgesetzt. Der notwendige statisch ausgewählte Kontext wird vollständig am Anfang injiziert; für Modellkontext gibt es im Run-Toolset keine nachladbare Leseoperation. Bei größerem Modell ist dies eine Skalierungs- und Klarheitsfrage, kein bereits gemessener allgemeiner Qualitätsverlust.

**Empfehlung:** Den tatsächlichen Ersatz ausdrücklich dokumentieren und abnehmen. Modularität und gezieltes Nachladen nur dort ergänzen, wo ein konkreter Umfangs-/Bediennachweis ihren Nutzen zeigt. Keine Pflicht, jedes Providerfeature einzubauen; aber auch keine Behauptung, die ursprünglich diskutierte Datei-/Skill-/Fallback-Variante sei schon vollständig umgesetzt.

### A5 – Workflow kontrolliert interne Ressourcenbezeichnungen des Runs

Die eigentliche Apply-Sequenz ist klein und fachlich korrekt. `workflow.ValidateClosed` prüft zusätzlich `session:<package>`, `copy:<package>`, `manager-workers` und `git-service`. Der Workflow muss dadurch technische Closure-Bezeichnungen kennen, obwohl der Executor seinen geschlossenen Rückgabevertrag bereits verantwortet. Apply verwendet diese Funktion wiederum zur Prüfung gespeicherter Zustimmung. [S12], [S13]

Die unabhängige Prüfung persistierter Ergebnisse ist sinnvoll und muss bleiben. Ihre Kenntnis der technischen Ressourcen sollte jedoch bei einer gemeinsam verwendeten Run-Resultatvalidierung liegen, statt die Workflow-Schicht an diese Namen zu koppeln. Dies ist ein begrenztes Architekturleck, keine zweite Orchestrierung und kein Anlass für ein neues Framework.

### A6 – Universell für heutige Mandate, nicht bereits für beliebige neue Aufträge

`Request` und `harness.Build` akzeptieren ausschließlich Assessment, Implement und Verify. Der Root-Auftrag wird im Executor fest vorgegeben. Ein weiterer Workflow kann diese Runs kombinieren; ein neuer Mandatstyp verlangt Änderungen an Core-/Harness-Prüfungen. [S06], [S08]

Für die enge V1 ist das vertretbar: Eine allgemeine Workflow-DSL wurde ausdrücklich vertagt. Es ist aber enger als das ursprüngliche langfristige Bild, in dem Mandate weitgehend vorbereitete Aufträge für denselben Mechanismus sind. Die vorhandene direkte Komposition nicht vorschnell durch eine universelle Engine ersetzen. Den Ausbaupunkt lediglich ehrlich benennen.

## 5. Bestätigte Mängel und konkrete Zuverlässigkeitsrisiken

### R1 – Publication-Guards verlieren die Aufrufdeadline

Der Publisher besitzt einen Context. `CaptureV1Files` und `ApplyOrderedChecked` nehmen ihn jedoch nicht entgegen. Ihre Git-Guards verwenden unter anderem `IdentifyGit`, `BranchName` und `GitOutput`; der gemeinsame Git-Start verwendet `exec.Command` statt einer an den Aufruf gebundenen Ausführung. Checks in den äußeren Callbacks können einen bereits blockierten inneren Git-Prozess nicht abbrechen. [S14], [S15], [S16]

Dies bestätigt den P2 des neuen Coordinators. Die Korrektur muss den gesamten tatsächlich aufgerufenen Guard-Pfad abdecken, nicht nur eine Funktion umbenennen. Partial-Write-Bericht, Branch-/Identitäts-/Dateiguards und Readback bleiben erhalten. Ein isolierter Abbruchtest muss die blockierende Git-Abfrage tatsächlich erreichen; der vorherige Sammeltest-Timeout beweist ihre konkrete Ursache nicht.

Es wurde in diesem Audit kein eigener Hänger provoziert. Eine garantierte sofortige Terminierung bei beliebigem Kernel-/Dateisystemstillstand wäre eine andere, hier nicht behauptete Eigenschaft.

### R2 – Der vollständige Notes-Abruf kann den Tooltransport übersteigen

Der Store lässt Records bis 16 MiB zu und liest projektweit bis zu 10.000 Einträge beziehungsweise insgesamt 16 MiB Text/Kontext. `read_notes` besitzt keinen Filter und keine Pagination und serialisiert alle Ergebnisse in eine einzelne Toolantwort. Der agentd-Toolkanal transportiert eine ToolReply pro Frame; `MaxPayload` beträgt 1 MiB. [S17], [S04], [S18]

Damit können gültig persistierte Notes bereits deutlich unter dem Speicherlimit nicht mehr als Ganzes zugestellt werden. JSON-/MCP-Einbettung verkleinert den verfügbaren Nutzraum zusätzlich. Das ist eine statisch nachvollziehbare Größeninkompatibilität; ein entsprechender neuer nativer Ausfall wurde hier nicht ausgeführt.

Ähnliche Ganzpaketgrenzen existieren bei umfangreichen Definitionskontexten und Textdiffs. Der bereits behobene Binärpatchfehler zeigt die praktische Relevanz der Transportgrenze, qualifiziert aber nicht beliebig große Text-/Notes-Antworten. [S19]

**Empfehlung:** Einen kleinen kontrollierten Grenztest schreiben; abrufbare Datenmenge an die tatsächliche Transportgrenze binden. Beispielsweise begrenzte Seiten mit Kontextfilter oder explizite Ergebnisreferenzen. Keine stumme Kürzung, keine Löschung historischer Notes, keine neue Vektordatenbank. Optionalität des Inhalts darf nicht bedeuten, dass beliebig wachsende optionale Historie spätere Arbeit regelmäßig blockiert.

### R3 – Prozesskomplexität sitzt an wenigen stark belasteten Stellen

`run/execute.go` enthält Umgebungsaufbau, Ressourcenende, Toolcallbacks, Delegationszustände, Review/Integration, Korrekturschleife und Checksammlung. `inbox.go` ergänzt Wakeups, Fragekorrelation, Nachlaufurteile und Abbruch. Mehrere Boolesche Zustände müssen zusammen mit Mutexen, WaitGroups und Kanälen kohärent bleiben. [S05], [S06], [S03]

Diese Synchronisation ist nicht automatisch überflüssig. Der Host muss echte parallele Provider und Schreibkopien beherrschen. Ein pauschaler Race-/Deadlockvorwurf wäre ohne Reproduktion unzulässig. Das dominante Wartbarkeitsrisiko ist dennoch die Kombination fachlicher Auftragszustände mit technischen Antwort-/Drainzuständen.

Das erste Qualitätsziel sollte eine klarere Semantik dieser Zustände sein, nicht eine kosmetische Verteilung identischer Komplexität auf mehr Dateien.

## 6. Bewusste Einschränkungen und fehlende Nachweise

### Akzeptierte beziehungsweise offengelegte V1-Grenzen

Extract, Cockpit/UI, Authoring und Setup wurden aus dem aktiven Umfang entfernt. Das war später ausdrücklich freigegeben und ist kein unbemerkter Funktionsverlust dieses Audits. Ebenso sind Konfliktabbruch statt semantischem Merge bei Nutzeränderungen, persistente Bundles statt langlebiger Services und MCP statt neuer Code-Mode-Runtime mit dem engen Konzept vereinbar.

Kein per-write-Schutz ist ausdrücklich gewollt. Aus korrektem Registry-Accounting folgt trotzdem nicht, dass jeder handelnde Manager nur eigene Dateien geändert hat. Der vollständige Diff, Aufgabenbasis und akzeptierte Kinderintegration liefern dem fachlichen Review den entsprechenden Kontext. Hier ist keine fehlende deterministische Einzelwrite-ACL als Defekt zu zählen.

### Provider

Claude ist nicht bloß ungetestet: `claudecode.Agents.Open` gibt im aktuellen Produktcode immer `ErrUnavailable` zurück. Codex kopiert absichtlich keine Refresh-Autorität in die Managerumgebung. Codex-native Erfolge bedeuten daher weder Claude-Funktionalität noch qualifizierte Login-Erneuerung. [S20], [S21]

Ein generischer AGENTS.md-Fallback wäre ohnehin nur Instruktionsdarstellung, kein ausführbarer unbekannter Provider. Das muss von einer behaupteten providerunabhängigen Produktbedienbarkeit getrennt bleiben.

### Qualifikation

Die Repository-Aufzeichnungen enthalten tatsächliche native M1- und M2-Realisierung, unabhängige Orakel, Preservation, Notes-Nutzung, Reparatur eines eingebauten Fehlers und striktes No-change. Das sind substanzielle Erfolge. Die Komponentenfolge Verify-Reject → frisches Implement → frisches Verify-Approve wurde ebenfalls nativ beobachtet; der äußere Controllerbericht bleibt wegen eines separat diagnostizierten Datumsvergleichsfehlers fehlgeschlagen. [S22]

Der native Beweis, dass der **öffentliche Apply-Workflow selbst** diesen Reject-Zweig auswählt und den daraus reparierten Kandidaten anschließend veröffentlicht, ist davon getrennt und noch offen. Eine normale Reparatur im initialen Implement ist kein Beweis dieses Zweigs. Ebenso fehlen breite Wiederholbarkeit und Aussagen für beliebige Projektgrößen. Das ist eine Nachweisgrenze, nicht ein daraus bewiesener Funktionsfehler.

## 7. Welche Probleme dominieren?

### 7.1 Beobachtet: Verständigung zwischen Agent und Host

Die dokumentierten Fehlläufe häufen sich an Abschluss, Fragen, bereits konsumierten Aufträgen, Handoff-/Reviewzuständen und der Platzierung von Checks. Das ist eine qualitative Einordnung der vorliegenden Berichte, keine statistische Fehlerrate. Die Verknüpfung von A2 und A3 ist wichtiger als zusätzliche Produktfeatures: Ein simples fachliches Ziel wird teilweise durch mehrere technische Kommunikationsregeln überlagert. [S07]

### 7.2 Beobachtet: Git-, Windows-, Container- und Transportgrenzen

Lange Pfade, Git-Konfigurationswirkung, Dateimodi, große Binärpatches und Cleanup unter Abbruch haben reale Korrekturen ausgelöst. Das sind überwiegend Infrastruktur-/Lebenszyklusprobleme, kein Beleg, dass die modellbasierte Organisation fachlich untauglich ist. Der aktuelle R1 und der skalierende R2 zeigen, dass zusammengesetzte technische Grenzen weiter wichtig bleiben. [S19], [S14], [S15], [S16], [S17], [S18]

### 7.3 Konzeptionell: Dokumentierte Zielregel und implementierter Vertrag driften

Ein Code kann zum aktualisierten Repository-Architekturtext passen, obwohl dieser inzwischen einen ursprünglich strengeren Owner-Beschluss weniger deutlich ausdrückt. „Kommunikation innerhalb des Runs“ ist nicht gleich „Kommunikation entlang der Hierarchie“. „Fehler im Kind reparierbar“ ist nicht gleich „Diagnose an den Vater“. Deshalb ist ein Vergleich gegen die ursprünglichen Entscheidungen notwendig, nicht nur gegen aktuelle Package-Tabellen.

### 7.4 Langfristig: Umfang und Nutzbarkeit

Große Vorabkontexte, vollständige Notes-Abrufe und umfangreiche prozessuale Hinweise skalieren nicht automatisch so gut wie die sauber getrennten Go-Packages. Aktuelle Library-Erfolge beantworten diese Frage nicht. Das ist ein begründeter nächster Messpunkt, keine Aufforderung zur großen Optimierungswelle.

## 8. Priorisierte nächste Schritte

1. **R1 vollständig schließen:** Context bis zu allen tatsächlich genutzten Publication-Guard-Git-Aufrufen; beobachtete Abbruchregression; keine verlorenen Teilpfade.
2. **A1/A2 gegen die Owner-Absicht gegenprüfen:** Kommunikationsgraph und Informationsfluss bei ungültiger Kinderabgabe konkret demonstrieren. Danach angleichen oder ausdrückliche bewusste Abweichung festhalten. Nicht allein aktuelle Docs als Freigabe interpretieren.
3. **A3 mit wenigen Gegenbeispielen vereinfachen:** Antwort ohne neue fachliche Arbeit, aktiver Stand mit neuer verpflichtender Finding, bereits fixiertes Handoff und verspätete Frage. Keine Wiederverwendung veralteter Zustimmung; keine unnötige Neuentscheidung für bloße Kommunikation.
4. **R2 kontrolliert reproduzieren und begrenzen:** Stored Notes unter Speicherlimit, aber über Toolframe; passende begrenzte Read-Schnittstelle statt größere pauschale Limits.
5. **A4/A5/A6 als Umfangs- und Qualitätsthemen entscheiden:** Native Instruktionsinjektion klar benennen, Workflow-Resultatprüfung näher zum Run-Vertrag halten, nur belegten Kontextbedarf modularisieren. Keine allgemeine Workflow-/Skillplattform auf Vorrat.
6. **Öffentlichen nativen Korrekturzweig gezielt nachweisen:** Auf fixiertem korrigiertem Stand, ohne erfolgreiche alte Nachweise umzudeuten. Danach begrenzte Codex-Abnahme mit klarer Provider- und Laufzeitmatrix.

Ein neuer Gesamtneubau ist durch diese Prüfung nicht begründet. Ebenso wäre die Aussage „nur noch der Git-Timeout, sonst alles wie konzipiert“ zu pauschal. Die richtige nächste Arbeit sind wenige überprüfbare Vertragsschnitte.

## 9. Quellen – auf den geprüften Commit gebunden

Die folgenden Links enthalten den fixierten Commit, keine bewegliche Main-Referenz. Funktionsnamen im Text sind die gezielt geprüften Stellen.

[S01]: https://api.github.com/repos/Glacius-Labs/Markitect/git/trees/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998:internal
[S02]: https://api.github.com/repos/Glacius-Labs/Markitect/git/trees/447ac3b152de79b52de80513e14dc94349a060bd:internal
[S03]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/inbox.go
[S04]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/tools.go
[S05]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/execute.go
[S06]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/run/contracts.go
[S07]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/docs/implementation-plan.md#v1-implementation-organization
[S08]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/harness/harness.go
[S09]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/agents.go
[S10]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/project/prepared.go
[S11]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/profile.go
[S12]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/workflow/apply.go
[S13]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/apply/apply.go
[S14]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/gitrepo/publication.go
[S15]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/tooling/guardedwrite/guardedwrite.go
[S16]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/tooling/source/git_snapshot.go
[S17]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/runstore/store.go
[S18]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/agentd/frame/frame.go
[S19]: https://github.com/Glacius-Labs/Markitect/pull/667
[S20]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/claudecode/agents.go
[S21]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/internal/adapters/codexappserver/agents_credential.go
[S22]: https://github.com/Glacius-Labs/Markitect/blob/6c7d758cbe55e079a7d0b6bcfdedb46e9ba58998/docs/implementation-plan.md#current-qualification-evidence

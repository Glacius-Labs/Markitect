# Abgleich des Modell-Proposals mit Main

Datum: 11. Oktober 2026. Gegenstand: [Proposal zum Aufbau des kanonischen Modells](canonical-model-proposal.md). Status: read-only Quellenvergleich; keine Produktänderung und kein Laufzeit-, Test- oder Studienergebnis.

## 1. Feste Vergleichsstände

| Stand | Zugriffspfad | Branch / Ref | Vollständige SHA | Lokale Änderungen |
| --- | --- | --- | --- | --- |
| Gesprächsnotizen und Vorarbeiten | `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect` | `docs/product-model-concepts` | `357a726347676ca8589a173b88aa8cb34d241c3d` | Beim Beginn sauber; danach nur Proposal, Vergleich und Navigations-/Statushinweise. |
| Frisch abgerufenes Main | Git-Objekte über denselben Repositoryzugang; kein Main-Checkout | `origin/main` | `32afd1a57e715130d056e98e433721c6e7a10278` | Ausschließlich committierte Dateien dieses SHA gelesen; Working-Tree-Änderungen gehören nicht zu diesem Stand. |

`git fetch origin main` hat den Vergleichsstand aktualisiert. Der feste Main-Commit ist „Merge pull request #173 from Glacius-Labs/docs/contribution-rules“, Commitdatum `2026-10-11T01:26:15+02:00`. Alle Main-Dateien wurden mit `git show SHA:Pfad` oder `git grep SHA` gelesen. Das ältere ausgecheckte Produkt unter dem Dokumentationsbranch wurde nicht als neuer Main-Code verwendet. Andere Worktrees oder offene Branches wurden nicht zu diesem Stand hinzugedacht.

Die erste Untersuchung galt `95251f1b76b9d7cb1c9c7321a7751146147a0b78` (Main-Commit vom selben Tag, `01:13:21+02:00`). Während des Abgleichs wurden unter anderem Decision-, Narrow-Routing-, Explain- und KnowledgeGraph-Arbeit integriert. Der hier berichtete Endstand berücksichtigt dieses Delta bis zur oben fixierten SHA. Unveränderte Core-, Schema-, Coverage- und Runtime-Dateien wurden anhand der ursprünglichen Lektüre und des Dateidiffs übernommen; geänderte relevante Quellen erneut gelesen. Der erste Dokumentationsstand ist als lokaler Commit `1b5cc7b2ba011e2b339aeed474d4195ff62cdc2f` erhalten. Weitere parallele Main-Änderungen nach dem letzten Abruf sind nicht Teil dieses Berichts.

Die Main-Links dieses Berichts zeigen auf GitHub-Dateien mit exakt diesem Commit. Sie dienen der Navigation; die Untersuchung erfolgte anhand der lokal abgerufenen Git-Objekte. Die Dokumentationsbasis unseres Branches bleibt ursprünglich `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.

## 2. Ergebnis

**Main enthält bereits ein brauchbares strukturelles und operatives Fundament. Der wichtigste zusätzliche Bedarf ist eine präzisere Sprache für die Beschreibung von Softwaresystemen und deren Realisierungsverpflichtungen.**

Der strukturelle Core kennt Schemas, Kinds, Properties und Definitions. Das konkrete Projectmodel kennt fünf Kinds: `Manager`, `Statement`, `Artifact`, `Check` und `Decision`. Statements halten viel Projektbedeutung als lesbaren Text und verbinden sie über `uses` und `requires`. Artefakte verbinden Statements mit Pfadbereichen und Prüfungen. Das erlaubt bereits Zuordnung ohne eine Definition pro Datei.

Main besitzt inzwischen auch abgeleitete KnowledgeGraph- und Impact-Erklärungsfunktionen: Sie machen bestehende Beziehungen, Decision-Subjects und Gründe für Änderungsfolgen sichtbar. Sie ergänzen Navigation und Erklärung, ohne die fachliche Beschreibungssprache automatisch zu erweitern.

Eine freie Systembeschreibung, die allein einen vollständigen Modell-, Zuständigkeits-, Artefakt- und Repositoryaufbau hervorbringt, ist damit nicht als abgeschlossener Ablauf belegt. Der fehlende semantische Vertrag ist auch nicht durch eine vollständige Pfadklassifikation bewiesen.

## 3. Dokumentierter Vertrag und Code

| Anforderung des Proposals | Dokumentierter Main-Vertrag | Tatsächlicher Code am festen SHA | Einordnung |
| --- | --- | --- | --- |
| Ein maßgebliches Modell und getrennte Evidenz | Vision: Modell als akzeptierte Projektdefinition, Code/Tests/Konfiguration/Dokumentation als Realisierungen. Architektur trennt Spezifikation, Kandidaten und Evidenz. [D1], [D2] | Projektmanifest, Modellquellen, Runtime und Run-Records sind getrennte Eingaben beziehungsweise Datensätze. [S5], [S7] | Fundament vorhanden; keine allgemeine semantische Korrektheitsgarantie. |
| Benannte, typisierte Definitionen | Core-Vertrag: source-unabhängige Strukturkompilierung. [D2] | Exakte Identität `(apiVersion, kind, namespace, name)`; skalare Typen, Enum, Objekt, Referenz/Kind-Referenz, Min-/MaxCount, geschlossene Properties, normalisiertes IR und Digest. [S1] | Vorhanden; zusätzliche Begriffsfamilien verlangen zunächst Vokabular, nicht automatisch neue Core-Primitiven. |
| Systemzweck, Begriffe, Datenstruktur, Verhalten, Qualität und Betrieb | Vision nennt Konzepte, Regeln, Architektur und Arbeitsweise. [D1] | Statement-Kategorien sind `concept`, `rule`, `use-case`, `architecture`, `workflow`; `description` bleibt Prosa. Keine entsprechenden strukturierten Fach-/Verhaltensfacetten im Schema. [S2] | In Text beschreibbar, aber semantisch nur grob typisiert. Hier liegt der Hauptbedarf des Proposals. |
| Eigenschaften und Kardinalitäten von Fachinstanzen | Keine allgemeine Bibliotheks-/Fachinstanzsprache im aktuellen Core-Vertrag. | Core-Property-Kardinalität zählt Werte einer Modelldefinition. Es gibt keinen generischen Laufzeitbestand von Exemplaren/Ausleihen und keinen Ausdrucksevaluator für deren gefilterte Kardinalitäten. [S1], [S2] | Beschreibung solcher Regeln ist möglich; ihre formale Semantik und Nachweise sind zusätzliche Arbeit. |
| Verhaltensverträge und erhaltene Invarianten | Checks und unabhängige Bewertung sind begrenzte Evidenz. [D2] | Keine allgemeinen Vor-/Nachbedingungs-, Zustandsübergangs-, Zeit-, Konkurrenz- oder Autorisierungsfelder; Check enthält literal argv, Statement-Bezüge und Begrenzung. [S2] | Prüfanschluss vorhanden; strukturierte Vertragssemantik fehlt im jetzigen Vokabular. |
| Hierarchie neben Querbeziehungen | Rekursive Manager, strukturierter Kontext und konservativer Impact. [D1], [D2] | Manager-Parent, Namespace-Hierarchie, nächster Manager als Definitionsowner; davon getrennt explizite Pfadverantwortung. Statements referenzieren andere Statements mit `uses`/`requires`. [S2], [S3] | Vorhandenes Beispiel für Baum plus Graph; noch kein allgemeiner Komponenten-/Systemkompositionsbaum. |
| Realisierung ohne jede Datei vorzugeben | Jede Datei ist zugeordnet oder ausdrücklich ignoriert; Mehrfachrealisierung ist erlaubt. [D3] | `Manager.owns` und `Artifact.paths` erlauben exakte Pfade oder Verzeichnispräfixe; eine Datei kann mehrere Artefakte/Statements betreffen. Beispiel nutzt `src/shop/orders/`. [S2], [S3], [E1] | Bereits teilweise erfüllt. Eine neue Funktion nur für „nicht jede Datei nennen“ ist nicht erforderlich. |
| Unterschiedliche Realisierungsbeiträge | Code, Tests, Konfiguration und Dokumentation sind Realisierungen. [D1] | Artifact-Rollen: implementation/documentation/verification/configuration. `realizes` ist ein gemeinsamer Bezug auf Statements; kein eigener Beitragstyp je referenzierter Aussage. [S2] | Gute Grundlage; feinere Semantik für implementiert/prüft/erklärt/konfiguriert/bereitstellt ist ein Erweiterungsvorschlag. |
| Abgeleiteter Realisierungsindex und Coverage | Full Coverage erfasst jede Datei im erklärten Umfang; Ignorieren bleibt ausdrücklich. [D2], [D3] | Census, Coverage-Entries und `projectmodel.Report.Files` halten konkrete Pfade, Digests, Zuordnung und Befunde. Ein Typ namens `RealizationIndex` ist nicht vorhanden. [S3], [S5], [S6] | Funktionale Vorläufer vorhanden. Neuer Name allein rechtfertigt keinen Neubau. Pfadzuordnung ersetzt keine Inhaltsprüfung. |
| Automatische semantische Zuordnung | Architektur: Dateien bleiben opaque bytes; keine Ableitung aus Programmsprachen, Symbolen, Prosalinks oder Dateinamen. [D2] | Zuordnung erfolgt aus den deklarierten Selektoren und dem Inventar. [S3], [S6] | Nicht vorhanden als allgemeine Sourceanalyse; kann später über bewusst begrenzte Werkzeuge ergänzt werden. |
| Nachvollziehbare Beziehungen und Impact-Gründe | DEC-018 und DEC-021 bis DEC-023 beschreiben den reduzierten Knowledge-Graph-Kern und Routing. [D4] | `ProjectGraph`, `ManagerGraph`, begrenztes `Trace`, `Explain` und `ExplainForManager` sind implementiert. Manageransicht entsteht aus dem vorhandenen Context; Change/Context klassifiziert bestehende Impact-Elemente. [S8] | Abgeleitete Sichten vorhanden; kein Ersatz für ein reichhaltigeres Fach-/Verhaltensvokabular. Keine neue kanonische Quelle. |
| Modelländerung → Kontext/Impact → realisierte Kandidaten → Apply | Dokumentierter Produktfluss mit festen Ständen, Review, Verify und Guarded Apply. [D2] | Model/View-Digests, Pläne, Kandidaten und Verify-/Apply-Bindungen sind in Projectwork/Projectrun vorhanden. [S5], [S7] | Mechanismen vorhanden. Nicht als Beweis beliebiger Repository-Erzeugung, Modellvollständigkeit oder Autonomieeffizienz zu verstehen. |

### 3.1 Drei verschiedene Arten von Lücken

1. **Vokabularlücke:** Beispielsweise strukturierte Fachproperties oder ein Verhaltensvertrag. Solche Daten lassen sich grundsätzlich mit den vorhandenen Objekt-, Listen- und Referenzmitteln modellieren. Das aktuelle Projektvokabular nutzt diese Struktur dafür noch nicht.
2. **Semantische Prüflücke:** Beispielsweise „die referenzierte Aussage muss eine Vorbedingung sein“, eine Komponentenbaumregel oder Übereinstimmung inverser Fachbeziehungen. Der Core prüft nicht generell diese Rollenbedeutung; eine reine, endliche Vokabularauswertung kann sie ergänzen.
3. **Realisierungs-/Evidenzlücke:** Beispielsweise tatsächlich atomare Vergabe oder korrekte Wiederherstellung. Dafür genügt selbst ein präziseres Schema nicht; passende Analyse, Tests oder Review müssen die realisierte Anwendung untersuchen.

Eine mögliche echte Core-Sprachlücke wäre eine benötigte nominale Referenzunion: Eine heutige Reference-Property nennt genau eine Ziel-Kind-Identität. Der Proposal löst das nicht durch eine ungeprüfte Core-Erweiterung. Strukturierte Facetten auf einem gemeinsamen Statement-Kind plus Rollenvalidierung sind zunächst ein Kandidat; mehrere eigenständige Kinds benötigen einen ausdrücklich entworfenen Referenzvertrag.

## 4. Akzeptierte Entscheidungen und ihr aktueller Codebefund

Der Main-Konzeptregister enthält die akzeptierten Entscheidungen DEC-021 bis DEC-023. Ihre Umsetzung wird hier anhand des Codes geprüft. Im ersten untersuchten Stand waren entsprechende Sichten und Routingregeln noch nicht enthalten; im festen Endstand sind sie integriert.

| Entscheidung / Zielvertrag | Code dieses Main-Stands | Konsequenz für den Proposal |
| --- | --- | --- |
| DEC-021: Statement-/Artifact-/Check-Purpose und bedeutungsfreie Schreibvarianten sollen enger routen. [D4] | `purposeChanges` routet Zweckänderungen nach dem betroffenen Definitionstyp. `writingChanges` erkennt bedeutungsfreie Schreibänderungen und routet über Modelldatei, Eigentümer und Definitionen daraus. Nicht zuordenbare Änderung bleibt konservativ. [S4] | Narrow Routing ist Codebefund. Neue Facetten brauchen dieselbe ausdrückliche Behandlung von Bedeutung, Quellschreibweise und unbekanntem Umfang. |
| DEC-022: Decisions erscheinen im Report und Managercontext; Änderung routet über Subject; Actor ist Provenienz. [D4] | `Analyze` projiziert Decisions; `Context` enthält eigene Decisions und benötigte öffentliche Subject-Statements. Änderungen berücksichtigen Owner und Subject, einschließlich alter/neuer Subjects; Actor besitzt keine allein daraus folgende Weisungsbefugnis. [S3], [S4] | Decision-Lesesicht und Subject-Routing vorhanden. Aufgezeichnete Provenienz bleibt von authentifizierter menschlicher Zustimmung getrennt. |
| DEC-023: Impact unterscheidet Change und Context, ohne Umfang zu verkleinern. [D4] | `ChangeImpact` behält seine Listen. Die separate `ImpactExplanation` klassifiziert genau deren Elemente als `change` oder `context`, ergänzt Grund/Witness und bindet den Impact-Digest. Unknown Scope stuft breit zu Change hoch. [S4], [S8] | Klassifikation ist eine abgeleitete Explain-Sicht und keine Erlaubnis, Kontext aus der Prüfung zu streichen. |

`ProjectGraph` zeigt die vorhandenen Definitionen, Inventardateien und deklarierten Kanten. `ManagerGraph` verwendet ausschließlich `ManagerContext`; fremde Inhalte werden nicht durch eine zweite Sichtbarkeitsregel geladen. Nicht inhaltlich sichtbare Kantenenden erscheinen gegebenenfalls als Identitätsreferenzen. `Trace` liefert deterministische Zeugenpfade und meldet abgeschnittene Ergebnisse über `Complete`. `ExplainForManager` begrenzt Erklärung und Witness auf sichtbaren Kontext und markiert abgeschnittene Witnesses als `Partial`. [S8]

Dieser Befund betrifft implementierte Funktionen im Projectmodel-Paket. Eine neue öffentliche CLI-/MCP-Oberfläche wird daraus nicht behauptet: Die gezielte Referenzsuche fand Aufrufe der neuen Graph-/Explain-Funktionen im Host bislang in Testquellen. Der aktuelle Graph beschreibt weiterhin das vorhandene Fünf-Kind-Vokabular; strukturierte Fachproperties, Verhaltensverträge oder Laufzeitinvarianten kommen dadurch nicht hinzu.

Die Erklärung klassifiziert die Impact-Listen für Statements, Manager, Dateien und Checks. Einzelne Artifact-/Decision-Einträge aus `ChangedDefinitions` erhalten dort keine eigene Klasse; sie vermitteln die Wirkung auf jene Listen. Die Actor-Behandlung als Provenienz ist im Routingcode erkennbar, aber ein isolierter Test für den Ausschluss eines ausschließlich als Actor genannten Managers wurde in den gelesenen Quellen nicht gefunden. Beides ist enger als eine pauschale Zusage, jede Änderung sei auf jeder Ebene vollständig erklärt.

## 5. Historische Annahme im älteren Entwurf

Der ältere Sprachentwurf vergleicht mit der Engineering Constitution seiner damaligen Basis `ecc69b0...` und nennt installierbare Schema- und Projection-Module. **Das ist kein aktueller Main-Vertrag.**

Main markiert die Canonical-Projection-Alpha und deren Schema-/Projection-Module-Split als historisch beziehungsweise entfernt. Die heutigen Pakete unter `internal/modules` sind Go-Implementierungseinheiten; `projectmodel` wird dem Core-Layer zugeordnet. Es gibt dadurch keinen belegten aktuellen dynamischen oder installierbaren Vokabularloader. [D5]

Der neue Proposal verwendet „Vokabular“ als semantischen Vertrag. Seine erste Umsetzung kann auf explizit komponierten Schema-/Validatorfunktionen beruhen; ein Installations- oder Pluginmodell wird nicht vorausgesetzt. Die alte Entwurfspassage bleibt als datierte Vorarbeit erhalten und erhält einen Hinweis auf diesen neueren Vergleich.

## 6. Gelesene Testquellen und ihre Aussagegrenze

| Quelltext | Was er als Prüfauftrag beschreibt | Befund dieser Arbeit |
| --- | --- | --- |
| `core/compile_test.go` | Typen, geschlossene Objekte, Kardinalitäten, Referenzen, deterministische Digests und Diagnostik. [T1] | Gelesen; nicht ausgeführt. |
| `projectmodel/projectmodel_test.go` | Mehrfachzuordnung, uses-Kontext gegenüber requires-Abdeckung, private Referenzen und konservativer Impact. [T2] | Gelesen; nicht ausgeführt. |
| Neue Decision-, Narrow-Routing-, Graph- und Explain-Testquellen | Subject-Routing, Zweck-/Schreibänderungen, Sichtbarkeit, deterministische Graph-/Zeugenpfade und unveränderte Impact-Mengen. [T2], [T4] | Gelesen; nicht ausgeführt. Die frühere breite Decision-only-Erwartung ist keine Aussage über den neuen Endstand. |
| Coverage- und Projectrun-Testquellen | Unklassifizierte Pfade, Ignorieren/Übergang, Full Verify und Apply-Bindungen. [T3] | Gelesen; keine Wiederholung der Tests oder von Providerläufen. |

Die Untersuchung trennt erwartetes Verhalten in Testquellen von tatsächlich ausgeführten Ergebnissen. Keine neue Aussage über Laufzeitzuverlässigkeit, Case Studies, Produktivität, Kosten oder menschliche Akzeptanz.

Konkrete gelesene neue Prüfaufträge sind unter anderem `TestImpactRoutesPurposeChangeLikeDescriptionChange`, `TestImpactRoutesMeaningFreeEditThroughItsFile`, `TestContextShowsOwnDecisionsAndTheirSubjects`, `TestImpactRoutesDecisionChangeThroughItsSubject`, `TestImpactNeverShrinksWhenChangesCombine`, `TestExplainClassifiesChangeAndContext` und `TestExplainCoversTheImpactWithRealWitnesses`. [T2], [T4]

## 7. Empfehlung nach dem Abgleich

1. Den vorhandenen strukturellen Core und die aktuelle Projektlinie als Ausgangspunkt verwenden.
2. Zuerst die fehlenden semantischen Beschreibungen am Bibliotheksbeispiel präzisieren und daraus versionierte Facetten beziehungsweise Kind-Verträge ableiten.
3. Bestehende Manager-, Artefakt-, Check-, Coverage- und Impact-Bezüge gezielt erweitern; nicht allein wegen anderer Namen duplizieren.
4. Für jedes neue Feld die Behandlung in Lesesichten, Digest, Kontext, Impact und Evidenz festlegen. Ein bloß erlaubtes, später unprojiziertes Feld reicht nicht.
5. Formalisierung nur für konkrete wiederkehrende Fälle ergänzen; Fachinstanzregeln bleiben von Modellstruktur und Runtime-Evidenz unterscheidbar.
6. Die Detailplanung mit dem bestehenden Modell-Owner und den laufenden Main-Arbeitspaketen abstimmen. Dieser Proposal aktiviert keine neue Implementierung und setzt keine Main-Gates außer Kraft.

## 8. Quellen am festen Main-SHA

Alle nachstehenden Referenzen sind auf `32afd1a57e715130d056e98e433721c6e7a10278` festgelegt.

- **D1:** [Vision: Produktauftrag und Arbeitsteilung](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/vision.md).
- **D2:** [Architektur: aktuelle Schichten, Artefakt- und Evidenzgrenzen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/architecture.md).
- **D3:** [DEC-006: vollständige Dateiabdeckung](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored).
- **D4:** [DEC-021 bis DEC-023: beschlossene Impact-/Decision-Richtung](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/concepts/register.md#dec-021-purpose-and-meaning-free-edits-route-narrowly).
- **D5:** [Aktuelle Pakete und historische Module](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/development/modules.md#go-capability-packages); [Engineering Constitution: historische Reset-Grenze](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/engineering-constitution.md#canonical-reset-source-boundary).
- [S1]: [Core-Typen und Identität, Zeilen 7–132](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/core/types.go#L7); [Compiler](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/core/compile.go#L38).
- [S2]: [Projectmodel-Schema, Zeilen 13–66](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/projectmodel.go#L13).
- [S3]: [Analyse: Zuordnung, Decisions und unprojizierte Bedeutung](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/analyze.go).
- [S4]: [Report, Decision, Impact, Erklärung und ManagerContext](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/types.go#L64); [Context und konservatives Routing](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/impact.go#L12).
- [S5]: [Projectwork: feste Eingaben und Coverage-Laden](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectwork/project.go#L26); [Konfigurationsvertrag](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectwork/config.go).
- [S6]: [Coverage-Census](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/census.go#L65); [Klassifikation](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/coverage.go#L54); [Ignore-Vertrag](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/ignore.go).
- [S7]: [Projectrun-Plan](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectrun/plan.go#L24); [Verify](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectrun/verify.go#L34); [Apply](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectrun/apply.go#L46).
- [E1]: [Shop-Artefakte: Präfix und mehrere Statements](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/examples/project-world/.markitect/model/commerce/sales/orders/artifacts.yaml#L1).
- [T1]: [Core-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/core/compile_test.go).
- [S8]: [KnowledgeGraph und begrenzte Traversierung](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/graph.go#L21); [Impact-Erklärung und sichtbarer Manager-Witness](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/explain.go#L42).
- [T2]: [Projectmodel-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/projectmodel_test.go); [Impact-Korrektheitsfälle](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/impact_correctness_test.go).
- [T3]: [Coverage-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/coverage_test.go); [Full-Verify-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectrun/full_verify_test.go).
- [T4]: [Graph-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/graph_test.go); [Explain-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/explain_test.go); [Produktfragen als Testquellen](https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectwork/knowledge_questions_test.go).

[D1]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/vision.md
[D2]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/architecture.md
[D3]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored
[D4]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/concepts/register.md#dec-021-purpose-and-meaning-free-edits-route-narrowly
[D5]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/docs/development/modules.md#go-capability-packages
[S1]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/core/types.go#L7
[S2]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/projectmodel.go#L13
[S3]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/analyze.go
[S4]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/impact.go#L12
[S5]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectwork/project.go#L26
[S6]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/coverage.go#L54
[S7]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectrun/plan.go#L24
[S8]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/graph.go#L21
[E1]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/examples/project-world/.markitect/model/commerce/sales/orders/artifacts.yaml#L1
[T1]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/core/compile_test.go
[T2]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/projectmodel_test.go
[T3]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/host/projectcoverage/coverage_test.go
[T4]: https://github.com/Glacius-Labs/Markitect/blob/32afd1a57e715130d056e98e433721c6e7a10278/src/internal/modules/projectmodel/explain_test.go

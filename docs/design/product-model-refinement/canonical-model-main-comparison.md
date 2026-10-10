# Abgleich des Modell-Proposals mit Main

Datum: 11. Oktober 2026. Gegenstand: [Proposal zum Aufbau des kanonischen Modells](canonical-model-proposal.md). Status: read-only Quellenvergleich; keine Produktänderung und kein Laufzeit-, Test- oder Studienergebnis.

## 1. Feste Vergleichsstände

| Stand | Zugriffspfad | Branch / Ref | Vollständige SHA | Lokale Änderungen |
| --- | --- | --- | --- | --- |
| Gesprächsnotizen und Vorarbeiten | `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect` | `docs/product-model-concepts` | `357a726347676ca8589a173b88aa8cb34d241c3d` | Beim Beginn sauber; danach nur Proposal, Vergleich und Navigations-/Statushinweise. |
| Frisch abgerufenes Main | Git-Objekte über denselben Repositoryzugang; kein Main-Checkout | `origin/main` | `95251f1b76b9d7cb1c9c7321a7751146147a0b78` | Ausschließlich committierte Dateien dieses SHA gelesen; Working-Tree-Änderungen gehören nicht zu diesem Stand. |

`git fetch origin main` hat den Vergleichsstand aktualisiert. Der feste Main-Commit ist „Merge pull request #169 from Glacius-Labs/fix/integration-review-required-for-child-deliveries“, Commitdatum `2026-10-11T01:13:21+02:00`. Alle Main-Dateien wurden mit `git show SHA:Pfad` oder `git grep SHA` gelesen. Das ältere ausgecheckte Produkt unter dem Dokumentationsbranch wurde nicht als neuer Main-Code verwendet. Andere Worktrees oder offene Branches wurden nicht zu diesem Stand hinzugedacht.

Die Main-Links dieses Berichts zeigen auf GitHub-Dateien mit exakt diesem Commit. Sie dienen der Navigation; die Untersuchung erfolgte anhand der lokal abgerufenen Git-Objekte. Die Dokumentationsbasis unseres Branches bleibt ursprünglich `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.

## 2. Ergebnis

**Main enthält bereits ein brauchbares strukturelles und operatives Fundament. Der wichtigste zusätzliche Bedarf ist eine präzisere Sprache für die Beschreibung von Softwaresystemen und deren Realisierungsverpflichtungen.**

Der strukturelle Core kennt Schemas, Kinds, Properties und Definitions. Das konkrete Projectmodel kennt fünf Kinds: `Manager`, `Statement`, `Artifact`, `Check` und `Decision`. Statements halten viel Projektbedeutung als lesbaren Text und verbinden sie über `uses` und `requires`. Artefakte verbinden Statements mit Pfadbereichen und Prüfungen. Das erlaubt bereits Zuordnung ohne eine Definition pro Datei.

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
| Modelländerung → Kontext/Impact → realisierte Kandidaten → Apply | Dokumentierter Produktfluss mit festen Ständen, Review, Verify und Guarded Apply. [D2] | Model/View-Digests, Pläne, Kandidaten und Verify-/Apply-Bindungen sind in Projectwork/Projectrun vorhanden. [S5], [S7] | Mechanismen vorhanden. Nicht als Beweis beliebiger Repository-Erzeugung, Modellvollständigkeit oder Autonomieeffizienz zu verstehen. |

### 3.1 Drei verschiedene Arten von Lücken

1. **Vokabularlücke:** Beispielsweise strukturierte Fachproperties oder ein Verhaltensvertrag. Solche Daten lassen sich grundsätzlich mit den vorhandenen Objekt-, Listen- und Referenzmitteln modellieren. Das aktuelle Projektvokabular nutzt diese Struktur dafür noch nicht.
2. **Semantische Prüflücke:** Beispielsweise „die referenzierte Aussage muss eine Vorbedingung sein“, eine Komponentenbaumregel oder Übereinstimmung inverser Fachbeziehungen. Der Core prüft nicht generell diese Rollenbedeutung; eine reine, endliche Vokabularauswertung kann sie ergänzen.
3. **Realisierungs-/Evidenzlücke:** Beispielsweise tatsächlich atomare Vergabe oder korrekte Wiederherstellung. Dafür genügt selbst ein präziseres Schema nicht; passende Analyse, Tests oder Review müssen die realisierte Anwendung untersuchen.

Eine mögliche echte Core-Sprachlücke wäre eine benötigte nominale Referenzunion: Eine heutige Reference-Property nennt genau eine Ziel-Kind-Identität. Der Proposal löst das nicht durch eine ungeprüfte Core-Erweiterung. Strukturierte Facetten auf einem gemeinsamen Statement-Kind plus Rollenvalidierung sind zunächst ein Kandidat; mehrere eigenständige Kinds benötigen einen ausdrücklich entworfenen Referenzvertrag.

## 4. Akzeptierte Entscheidungen sind nicht automatisch implementiert

Der Main-Konzeptregister enthält die akzeptierten Entscheidungen DEC-021 bis DEC-023. Ihre Umsetzung darf für diesen SHA nicht aus dem Register allein abgeleitet werden.

| Entscheidung / Zielvertrag | Code dieses Main-Stands | Konsequenz für den Proposal |
| --- | --- | --- |
| DEC-021: Statement-/Artifact-/Check-Purpose und bedeutungsfreie Schreibvarianten sollen enger routen. [D4] | `unprojectedDigest` erfasst deren Purpose; ein reiner Purpose-Wechsel sowie `unprojectedChange` oder `writingChange` können konservative breite Prüfung auslösen. Schreibvarianten werden hier noch nicht auf ihren Modellfile-Owner beschränkt. [S3], [S4] | Geänderte Bedeutung und relevante Quellschreibweise müssen in den Sichten sichtbar sein. Engere Regeln sind dokumentierter Zielvertrag, hier kein vollständiger Codebefund. |
| DEC-022: Decisions erscheinen im Report und Managercontext; Änderung routet über Subject; Actor ist Provenienz. [D4] | Schema prüft Decision-Referenzen, aber Report/ManagerContext besitzen kein Decision-Feld. Ganze Decisions gehen in den nicht projizierten Digest ein. [S2], [S3], [S4] | Eine kanonische Definition kann typisiert vorhanden sein und trotzdem als strukturierte Lesesicht fehlen. Genau diese Trennung ist im neuen Vokabular zu vermeiden. |
| DEC-023: Impact unterscheidet Change und Context, ohne Umfang zu verkleinern. [D4] | `ChangeImpact` besitzt die bisherigen Listen und keine entsprechenden Klassifikationsfelder. [S4] | Ein akzeptierter Priorisierungsvorschlag ist auf dieser SHA noch nicht die Datenform des Impact-Reports. |

Diese Abweichungen können laufende, noch nicht integrierte Arbeit betreffen. Es wurden keine anderen Branches untersucht oder als implementierter Main hinzugenommen. Es wird keine Ursache oder Verantwortlichkeit behauptet.

## 5. Historische Annahme im älteren Entwurf

Der ältere Sprachentwurf vergleicht mit der Engineering Constitution seiner damaligen Basis `ecc69b0...` und nennt installierbare Schema- und Projection-Module. **Das ist kein aktueller Main-Vertrag.**

Main markiert die Canonical-Projection-Alpha und deren Schema-/Projection-Module-Split als historisch beziehungsweise entfernt. Die heutigen Pakete unter `internal/modules` sind Go-Implementierungseinheiten; `projectmodel` wird dem Core-Layer zugeordnet. Es gibt dadurch keinen belegten aktuellen dynamischen oder installierbaren Vokabularloader. [D5]

Der neue Proposal verwendet „Vokabular“ als semantischen Vertrag. Seine erste Umsetzung kann auf explizit komponierten Schema-/Validatorfunktionen beruhen; ein Installations- oder Pluginmodell wird nicht vorausgesetzt. Die alte Entwurfspassage bleibt als datierte Vorarbeit erhalten und erhält einen Hinweis auf diesen neueren Vergleich.

## 6. Gelesene Testquellen und ihre Aussagegrenze

| Quelltext | Was er als Prüfauftrag beschreibt | Befund dieser Arbeit |
| --- | --- | --- |
| `core/compile_test.go` | Typen, geschlossene Objekte, Kardinalitäten, Referenzen, deterministische Digests und Diagnostik. [T1] | Gelesen; nicht ausgeführt. |
| `projectmodel/projectmodel_test.go` | Mehrfachzuordnung, uses-Kontext gegenüber requires-Abdeckung, private Referenzen und konservativer Impact. [T2] | Gelesen; nicht ausgeführt. |
| `TestImpactDecisionOnlyDeltaRoutesFullDeclaredReview` | Breite Prüfung bei nicht projizierter Modelländerung; passend zum Codebefund dieser SHA. [T2] | Gelesen; kein grüner Testbefund und keine Aussage über neuere offene KG-Arbeit. |
| Coverage- und Projectrun-Testquellen | Unklassifizierte Pfade, Ignorieren/Übergang, Full Verify und Apply-Bindungen. [T3] | Gelesen; keine Wiederholung der Tests oder von Providerläufen. |

Die Untersuchung trennt erwartetes Verhalten in Testquellen von tatsächlich ausgeführten Ergebnissen. Keine neue Aussage über Laufzeitzuverlässigkeit, Case Studies, Produktivität, Kosten oder menschliche Akzeptanz.

## 7. Empfehlung nach dem Abgleich

1. Den vorhandenen strukturellen Core und die aktuelle Projektlinie als Ausgangspunkt verwenden.
2. Zuerst die fehlenden semantischen Beschreibungen am Bibliotheksbeispiel präzisieren und daraus versionierte Facetten beziehungsweise Kind-Verträge ableiten.
3. Bestehende Manager-, Artefakt-, Check-, Coverage- und Impact-Bezüge gezielt erweitern; nicht allein wegen anderer Namen duplizieren.
4. Für jedes neue Feld die Behandlung in Lesesichten, Digest, Kontext, Impact und Evidenz festlegen. Ein bloß erlaubtes, später unprojiziertes Feld reicht nicht.
5. Formalisierung nur für konkrete wiederkehrende Fälle ergänzen; Fachinstanzregeln bleiben von Modellstruktur und Runtime-Evidenz unterscheidbar.
6. Die Detailplanung mit dem bestehenden Modell-Owner und den laufenden Main-Arbeitspaketen abstimmen. Dieser Proposal aktiviert keine neue Implementierung und setzt keine Main-Gates außer Kraft.

## 8. Quellen am festen Main-SHA

Alle nachstehenden Referenzen sind auf `95251f1b76b9d7cb1c9c7321a7751146147a0b78` festgelegt.

- **D1:** [Vision: Produktauftrag und Arbeitsteilung](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/vision.md).
- **D2:** [Architektur: aktuelle Schichten, Artefakt- und Evidenzgrenzen](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/architecture.md).
- **D3:** [DEC-006: vollständige Dateiabdeckung](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored).
- **D4:** [DEC-021 bis DEC-023: beschlossene Impact-/Decision-Richtung](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/concepts/register.md#dec-021-purpose-and-meaning-free-edits-route-narrowly).
- **D5:** [Aktuelle Pakete und historische Module](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/development/modules.md#go-capability-packages); [Engineering Constitution: historische Reset-Grenze](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/engineering-constitution.md#canonical-reset-source-boundary).
- [S1]: [Core-Typen und Identität, Zeilen 7–132](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/core/types.go#L7); [Compiler](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/core/compile.go#L38).
- [S2]: [Projectmodel-Schema, Zeilen 13–66](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/projectmodel.go#L13).
- [S3]: [Analyse: Zuordnung und unprojizierte Bedeutung](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/analyze.go#L136); [Namespace-/Managerregeln, ab Zeile 395](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/analyze.go#L395).
- [S4]: [Report, Impact und ManagerContext, Zeilen 76–129](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/types.go#L76); [konservativer Impact](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/impact.go#L510).
- [S5]: [Projectwork: feste Eingaben und Coverage-Laden](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectwork/project.go#L26); [Konfigurationsvertrag](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectwork/config.go).
- [S6]: [Coverage-Census](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/census.go#L65); [Klassifikation](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/coverage.go#L54); [Ignore-Vertrag](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/ignore.go).
- [S7]: [Projectrun-Plan](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectrun/plan.go#L24); [Verify](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectrun/verify.go#L34); [Apply](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectrun/apply.go#L46).
- [E1]: [Shop-Artefakte: Präfix und mehrere Statements](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/examples/project-world/.markitect/model/commerce/sales/orders/artifacts.yaml#L1).
- [T1]: [Core-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/core/compile_test.go).
- [T2]: [Projectmodel-Testquellen, Decision-Fall ab Zeile 99](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/projectmodel_test.go#L99).
- [T3]: [Coverage-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/coverage_test.go); [Full-Verify-Testquellen](https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectrun/full_verify_test.go).

[D1]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/vision.md
[D2]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/architecture.md
[D3]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored
[D4]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/concepts/register.md#dec-021-purpose-and-meaning-free-edits-route-narrowly
[D5]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/docs/development/modules.md#go-capability-packages
[S1]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/core/types.go#L7
[S2]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/projectmodel.go#L13
[S3]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/analyze.go#L136
[S4]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/types.go#L76
[S5]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectwork/project.go#L26
[S6]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/coverage.go#L54
[S7]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectrun/plan.go#L24
[E1]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/examples/project-world/.markitect/model/commerce/sales/orders/artifacts.yaml#L1
[T1]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/core/compile_test.go
[T2]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/modules/projectmodel/projectmodel_test.go#L99
[T3]: https://github.com/Glacius-Labs/Markitect/blob/95251f1b76b9d7cb1c9c7321a7751146147a0b78/src/internal/host/projectcoverage/coverage_test.go

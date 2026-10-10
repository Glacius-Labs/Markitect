# SoftwareSystem und Repository: konkreter Modellierungsentwurf

Stand: 10. Oktober 2026. Status: beauftragter Entwurf zur Diskussion, keine angenommene Produktentscheidung und kein implementierter Markitect-Vertrag.

Dieser Entwurf konkretisiert die [Beschreibungssprache](canonical-model-draft.md) am Bibliotheksbeispiel. Er enthält zwei illustrative C#-Quellen: die [Klassen und Dienstgrenzen](SoftwareProjectModel.cs) und eine [Belegung mit Bibliotheksdefinitionen](LibraryModelExample.cs). Beide liegen ausschließlich in diesem Dokumentationsordner. Sie wurden durch Lesen geprüft, weder kompiliert noch getestet. Es gibt kein neues ausführbares Projekt, keine Studie und keine Änderung am Produktcode.

## 1. Was diese Klassen repräsentieren

`SoftwareSystem` beschreibt ein System; `Repository` beschreibt dessen erwartete Realisierung in einem Quellrepository; `SoftwareProject` verbindet Systeme, Repositories und die Organisation der Arbeit. Ein Projekt kann mehrere Systeme und Repositories umfassen. Ein Repository kann mehrere Systeme realisieren.

Die Klassen bilden ein **beschreibendes Metamodell**. Eine `DomainTypeDefinition` namens `Loan` beschreibt, was eine Ausleihe ist. Sie enthält keine laufenden Ausleihen. Ein `BehaviorContract` namens `Borrow` beschreibt das Ausleihen. Seine Instanziierung verleiht kein Buch. Eine `ArtifactExpectation` fordert passende Anwendungsteile oder Prüfungen; sie erzeugt keine Dateien.

Die geltende Beschreibung und die beobachtete Realisierung bleiben unterscheidbar. Das Modell ist der Maßstab; ein beobachteter Fehler wird dadurch nicht zu einer neuen gültigen Regel. Das folgt der noch zu bestätigenden Arbeitsannahme zum Begriff IST in [Abschnitt 1.1 des übergeordneten Entwurfs](canonical-model-draft.md#11-der-geltende-systemstand).

Alle wichtigen Definitionen werden benannt, identifiziert und einmal im `DefinitionCatalog` geführt. Wurzelobjekte verweisen darauf. Der Katalog ist eine zusammengesetzte Lesesicht auf verteilte Modellquellen; er verlangt keine große manuell gepflegte Datei. Die C#-Records illustrieren diese Lesesicht, nicht das endgültige Autorenformat. Records und unveränderliche Collections dienen hier der Darstellung. Die Sprache garantiert damit weder semantische Modellgültigkeit noch einen stabilen Inhaltsdigest; beides braucht explizite Regeln. [C#-Records](https://learn.microsoft.com/en-us/dotnet/csharp/language-reference/builtin-types/record).

## 2. SoftwareSystem am Bibliotheksbeispiel

### 2.1 Inhalt und Unterobjekte

| Inhalt | Allgemeiner Modellbegriff | Bibliotheksbeispiel |
| --- | --- | --- |
| Problem und Nutzen | `ProblemDefinition`, `GoalDefinition` | Bücher nutzen können, ohne jedes Buch zu kaufen; Verfügbarkeit und Verbleib nachvollziehen. |
| Umfang | `ModelScope` | Software zur Ausleihe und Rückgabe; physische Reinigung und Buchbeschaffung außerhalb dieses kleinen Beispiels. |
| Beteiligte | `ActorDefinition` | Mitglied als Nutzer; in einer erweiterten Beschreibung auch Mitarbeitende und externe Katalogdienste. |
| Fachbegriffe | `DomainTypeDefinition` | Titel, Exemplar, Mitglied, Ausleihe. |
| Eigenschaften | `PropertyDefinition`, `PropertyType`, `Cardinality` | Exemplar-ID, Titelbezug, Zustand; Beginn, Frist und optionales Rückgabedatum der Ausleihe. |
| Fachbeziehungen | `AssociationDefinition` | Ein Titel hat viele Exemplare; jedes Exemplar hat genau einen Titel; historische Ausleihen verbinden Exemplare mit Mitgliedern. |
| Fähigkeiten und Verhalten | `CapabilityDefinition`, `BehaviorContract` | Ausleihen und Rückgabe erfassen; Eingaben, Erfolg, Ablehnung, Zustandswirkungen. |
| Fachregeln | `RuleDefinition` | Höchstens eine aktive Ausleihe pro Exemplar; Rückgabe beendet eine aktive Ausleihe und erhält die Historie. |
| Technische Gliederung | `ComponentDefinition`, `PortDefinition` | Bestand, Mitgliedschaft und Ausleihe; bereitgestellte und benötigte Verträge. |
| Abhängigkeiten | `DependencyDefinition` | Ausleihe benötigt Bestands- und Berechtigungsinformationen. |
| Qualität und Nachweis | `QualityRequirement`, `CheckProcedure` | Auch konkurrierende Ausleihanfragen erhalten die Eindeutigkeit; konkretes Prüfszenario. |
| Betrieb | `OperatingModel`, `ConfigurationParameter` | Anwendung und persistenter Speicher; konfigurierbare positive Ausleihdauer. |

Der Nutzen der Bibliothek ist größer als der Umfang ihrer Software. Auswahl, Anschaffungs- und Lagerkosten können ein Zielsystem für Bestandsentscheidungen bilden. Erst wenn die Software solche Entscheidungen unterstützen soll, werden diese als Fähigkeiten, Daten, Kennzahlen und Regeln aufgenommen. Die Zielbeschreibung einer Organisation bedeutet nicht automatisch, dass ihre gesamte Tätigkeit automatisiert werden soll.

### 2.2 Fachtypen und Beziehungen

| Fachtyp | Identität und Eigenschaften | Beziehungen / wichtige Regeln |
| --- | --- | --- |
| `Title` | `titleId`, `name`; später etwa Autoren oder ISBN | `Title → Copy`: 0..*; jedes Exemplar verweist auf genau einen Titel. |
| `Copy` | `copyId`, `title`, `condition` | `Copy → Loan`: 0..* über die gesamte Historie; höchstens eine aktive Ausleihe. |
| `Member` | `memberId`, `isEligible` | `Member → Loan`: 0..*; genaue Regeln zur Berechtigung sind im kleinen Beispiel außerhalb des Umfangs. |
| `Loan` | `loanId`, `copy`, `member`, `startedAt`, `dueAt`, optional `returnedAt` | Genau ein Exemplar und ein Mitglied; Frist nach Beginn; Rückgabe nicht vor Beginn. |

Eine Beziehungskardinalität beschreibt hier **Fachinstanzen**, nicht die Anzahl der Definitionen im Markitect-Graphen. Es gibt eine Definition `Copy`, aber viele Exemplare in der Anwendung. Die inverse Beziehung wird über die typisierte Eigenschaft beschrieben, etwa `Copy.title`; der Compiler muss die Übereinstimmung der beiden Beschreibungen prüfen.

Eine aktive Ausleihe ist im Beispiel eine Ausleihe ohne `returnedAt`. Die Bedingung „höchstens eine aktive“ ist eine gefilterte, zustandsabhängige Regel. Sie ist nicht gleichbedeutend mit „ein Exemplar darf insgesamt nur eine Ausleihe besitzen“.

### 2.3 Verhaltensverträge

| Vertrag | Vorbedingungen | Erfolg | Fehler / Grenzen |
| --- | --- | --- | --- |
| `Borrow` | Mitglied existiert und ist berechtigt; Exemplar ist intakt und frei. | Eine aktive Ausleihe wird dauerhaft angelegt; Exemplar, Mitglied und Frist sind gesetzt; Eindeutigkeit bleibt erhalten. | Ablehnung bei fehlender Berechtigung oder Verfügbarkeit; keine zusätzliche Ausleihe. Auch bei Konkurrenz muss die Vergabe atomar wirken. |
| `Return` | Ausleihe existiert und ist aktiv. | `returnedAt` wird dauerhaft gesetzt; Historie bleibt erhalten; diese Ausleihe ist nicht mehr aktiv. | Unbekannte oder bereits beendete Ausleihe wird abgelehnt und nicht verändert. Die physische Rückgabe wird nicht durch diesen Softwarevertrag bewiesen. |

Eine wiederholte Rückgabe könnte stattdessen idempotent erfolgreich sein. Das wäre eine andere erlaubte Produktentscheidung, keine Eigenschaft, die ein Agent stillschweigend aus dem Begriff „Rückgabe“ ableiten sollte. Wer Rückgaben erfassen darf, fehlt im kleinen Beispiel noch als verbindliche Autorisierungsregel. Der Vertrag enthält bislang die fachliche Zustandsänderung, keine vollständige Sicherheitsbeschreibung.

`BehaviorContract.Maintains` nennt separat die zu erhaltenden Invarianten. Sie gelten über den konkreten Erfolg hinaus und werden deshalb nicht als bloße Nachbedingungen umklassifiziert. Die Fristberechnung aus `loanDuration` ist bislang lesbarer Regeltext; eine maschinell verfolgbare Referenz auf den Konfigurationsparameter und die genaue Zeitsemantik sind noch auszuarbeiten.

Das beschriebene Konkurrenzszenario ist ein Prüfauftrag: zwei berechtigte Anfragen gleichzeitig auf dasselbe freie Exemplar; genau ein Erfolg, eine Ablehnung, eine gespeicherte aktive Ausleihe. Es wurde nicht ausgeführt und ersetzt keine umfassende Prüfung beliebiger Interleavings oder Ausfallsituationen.

### 2.4 Die Klasse in C#

Die positionalen Record-Parameter erzeugen die gleichnamigen Properties. `Ref<T>` enthält eine stabile ID und beschränkt den erwarteten Referenztyp. Die Methoden lösen Definitionen zur Betrachtung auf.

```csharp
public sealed record SoftwareSystem(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ModelScope Boundary,
    ImmutableArray<Ref<ProblemDefinition>> Problems,
    ImmutableArray<Ref<GoalDefinition>> Goals,
    ImmutableArray<Ref<ActorDefinition>> Actors,
    ImmutableArray<Ref<DomainTypeDefinition>> DomainTypes,
    ImmutableArray<Ref<AssociationDefinition>> Associations,
    ImmutableArray<Ref<CapabilityDefinition>> Capabilities,
    ImmutableArray<Ref<ComponentDefinition>> RootComponents,
    ImmutableArray<Ref<DependencyDefinition>> Dependencies,
    ImmutableArray<Ref<RuleDefinition>> Rules,
    ImmutableArray<Ref<QualityRequirement>> QualityRequirements,
    ImmutableArray<Ref<OperatingModel>> OperatingModels)
    : Definition(Id, Name, Meaning)
{
    public IEnumerable<CapabilityDefinition> GetCapabilities(
        DefinitionCatalog catalog) =>
        Capabilities.Select(reference => catalog.Resolve(reference));

    public IEnumerable<DomainTypeDefinition> GetDomainTypes(
        DefinitionCatalog catalog) =>
        DomainTypes.Select(reference => catalog.Resolve(reference));
}
```

`Borrow()` liegt als beschriebener Vertrag unter den Fähigkeiten. Eine Methode auf `SoftwareSystem`, die selbst Ausleihen ausführt, würde beschreibendes Modell und realisierte Anwendung vermischen. Welche Laufzeitklassen die Bibliotheksanwendung später dafür verwendet, bleibt eine ausdrücklich offene Umsetzungsentscheidung.

## 3. Repository am selben Beispiel

### 3.1 Was es zusätzlich beschreibt

Das Repository beantwortet: Welche Artefakte müssen die System- und Projektdefinition realisieren? Welche Regeln gelten für deren Ablage, Pflege und Zuordnung?

| Erwartete Artefaktfamilie | Beitrag | Beispielhafte Konvention |
| --- | --- | --- |
| Anwendungscode | Implementiert `Borrow` und `Return`. | `src/`; kohäsive Bereiche und passende Namespaces. |
| Fachcode / Datenhaltung | Realisiert Fachtypen und deren konsistente Speicherung. | Im Beispiel persistenter Zustand; gegebenenfalls Schemaänderungen unter `migrations/`. |
| Tests | Realisieren Prüfverfahren und prüfen Fachregeln. | `tests/`; Konkurrenzszenario für die aktive Ausleihe. |
| Nutzerdokumentation | Erklärt Fähigkeiten und ihre Grenzen. | `docs/`; Ausleihe, Rückgabe, Ablehnungen. |
| Konfiguration | Legt zulässige Betriebsparameter fest. | Beispielsweise Ausleihdauer; Werte und Geheimnisse brauchen getrennte Regeln. |
| Infrastruktur / Pipeline | Realisiert Bereitstellung und Projektverfahren. | Nur erforderlich, wenn das Projekt diese Betriebs- und Arbeitsform wählt. |
| Skills / Arbeitsanweisungen | Realisieren die vereinbarte Arbeitsweise. | Beschriebene Prüf-, Review- und Eskalationsverfahren. |

Die C#-Factory belegt zunächst vier Erwartungen: Anwendung, Konkurrenzprüfung, Nutzerdokumentation und Migration. Die Tabelle zeigt weitere Ausdrucksbedarfe. Die Factory behauptet keine vollständige Modellierung einer produktiven Bibliotheksplattform.

Eine `ArtifactExpectation` ist eine semantische Verpflichtung. „Es muss eine Prüfung der konkurrierenden Ausleihe geben“ ist präziser als „es muss eine Datei im Testordner geben“. Ein Ordner oder Name allein belegt keine Implementierung dieser Prüfung.

### 3.2 Die Klasse in C#

```csharp
public sealed record Repository(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<SoftwareSystem>> Systems,
    ImmutableArray<Ref<ArtifactExpectation>> ExpectedArtifacts,
    ImmutableArray<Ref<LayoutConvention>> Layout,
    ImmutableArray<Ref<RuleDefinition>> Policies,
    CoveragePolicy Coverage)
    : Definition(Id, Name, Meaning)
{
    public IEnumerable<ArtifactExpectation> GetExpectations(
        DefinitionCatalog catalog) =>
        ExpectedArtifacts.Select(reference => catalog.Resolve(reference));

    public IEnumerable<ArtifactExpectation> GetExpectedFor(
        DefinitionId subject, DefinitionCatalog catalog) =>
        GetExpectations(catalog).Where(expectation =>
            expectation.Contributions.Any(item => item.Subject.Id == subject));
}
```

`GetExpectedFor()` liefert deklarierte Erwartungen für eine Definition. Es liefert noch keine Dateien und keine vollständige Auswirkungsanalyse. Diese benötigt den konkreten Realisierungsindex und die relevanten Beziehungstypen.

### 3.3 Erwartung, Index und Verantwortung

```csharp
public sealed record ArtifactExpectation(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ArtifactRole Role, Presence Presence,
    Ref<LayoutConvention>? Location,
    ImmutableArray<ModelContribution> Contributions,
    Ref<ResponsibilityDefinition> WriteOwner,
    GeneratedOutputContract? GeneratedOutput = null)
    : Definition(Id, Name, Meaning);

public sealed record RealizationIndex(
    RepositorySnapshot Snapshot, CoverageExtent Extent,
    ImmutableArray<ObservedArtifact> Artifacts);
```

Die Zuordnung ist in beide Richtungen vielfach: Eine Definition kann mehrere Dateien benötigen; eine Datei kann mehrere Definitionen realisieren. Eine Datei kann eine Regel implementieren, eine zweite sie prüfen, eine dritte sie erklären. Diese Beiträge unterscheiden sich ausdrücklich.

Die konkrete Datei soll trotzdem einen eindeutigen Schreibverantwortlichen haben. `ObservedArtifact.ResolvedWriteOwner` ist deshalb eine abgeleitete Zuordnung mit Evidenz, keine zweite gesetzte Wahrheit. Wenn die zugeordneten Erwartungen widersprüchliche Eigentümer verlangen, entsteht ein Konflikt; die Zuordnung darf nicht still einen Gewinner wählen. Gemeinsam verwendete Dateien brauchen eine explizite Zuständigkeit oder feinere Aufteilung.

Für generierte Ausgaben werden maßgebliche Eingaben und ein Erzeugungsvertrag benannt. Erzeugung ist orthogonal zur Rolle: Auch Tests oder Konfigurationsdateien können generiert sein. Eine deklarierte `GeneratedView` braucht einen Erzeugungsvertrag; andere Rollen dürfen ebenfalls einen besitzen. Pro konkreter Ausgabe gilt eine maßgebliche Erzeugungszuständigkeit.

Die Coverage-Policy verlangt im Beispiel die Einordnung jeder Datei im betrachteten Umfang oder eine ausdrückliche Ausnahme mit Begründung. `Full`, `Partial` und `Unknown` beschreiben die Reichweite der Beobachtung. Eine Teilbeobachtung darf kein Urteil über vollständige Abdeckung tragen.

Ein ableitbarer Index ist kein zusätzliches Modell, das Menschen synchron halten sollen. Er ist eine erneuerbare, an Modell- und Repositorystand gebundene Sicht. Wenn seine Ermittlung unsicher ist, muss Markitect diese Unsicherheit zeigen. Dass alle Dateien klassifiziert wurden, beweist noch nicht, dass ihr Verhalten den Verträgen entspricht.

Der Name `Repository` bezeichnet hier ein Quellrepository. Er hat eine andere Bedeutung als das DDD-Repository-Pattern für den Zugriff auf gespeicherte Aggregate.

## 4. Weitere Klassen, Methoden und Patterns

### 4.1 Begriffs- und Klassenfamilien

| Klassen | Aufgabe / wichtige Properties |
| --- | --- |
| `DefinitionId`, `QualifiedName`, `Ref<T>`, `DefinitionCatalog` | Identität, Namespace, typisierter Verweis; `Resolve<T>()` liefert das passende Objekt oder meldet einen unauflösbaren Verweis. |
| `PropertyDefinition`, `PropertyType`, `Cardinality` | Eigenschaften, skalare Typen und Einheiten, erlaubte Werte, Referenzen, Häufigkeit, Identitätsfunktion. |
| `DomainTypeDefinition`, `AssociationDefinition` | Daten- und Begriffsstruktur, beschriebene Operationen, Invarianten; gerichtete fachliche Beziehungen mit Kardinalität. |
| `CapabilityDefinition`, `BehaviorContract`, `FailureOutcome` | Nutzerfähigkeiten, Eingaben, Ausgaben, Vor-/Nachbedingungen, Fehler, Zustandswirkungen und freie Entscheidungen. |
| `RuleDefinition`, `RuleScope`, `QualityRequirement`, `CheckProcedure` | Regel, expliziter Geltungsbereich, Begründung, Qualitätsmaßstab und verlangter Nachweis. |
| `ComponentDefinition`, `PortDefinition`, `DependencyDefinition` | Bausteinbaum, bereitgestellte/benötigte Verträge, getrennte Arten der Abhängigkeit. |
| `OperatingModel`, `ConfigurationParameter` | Betriebsstruktur, Umgebungen, Konfiguration, Verfahren, Randbedingungen. |
| `ResponsibilityDefinition`, `WorkflowDefinition`, `WorkflowStep` | Hierarchische Zuständigkeit, Befugnisse, Reviewpflichten und Abläufe. |
| `ArtifactExpectation`, `ModelContribution`, `LayoutConvention`, `CoveragePolicy` | Rolle, Pflicht/Optionalität/Verbot, semantischer Beitrag, Ablage, Schreibverantwortung und Abdeckung. |
| `SoftwareProject`, `CanonicalModel` | Verbindet fachliche, technische und organisatorische Beschreibung mit dem Definitionenkatalog. |
| `RepositorySnapshot`, `ObservedArtifact`, `RealizationIndex` | Konkreter Stand, beobachtete Dateien, Zuordnung und deren Evidenz. |
| `CompiledModel`, `Diagnostic`, `VerificationReport`, `ImpactReport` | Strukturdiagnostik, feste Inhaltsbezüge, Prüfbefunde und bekannte/unbekannte Auswirkungen. |

In der vollständigen C#-Quelle sind sämtliche verwendeten Klassen, Enums und Interfaces definiert. Die Klassenliste ist eine erklärende Projektion für dieses Gespräch. Sie ist kein Auftrag, jeden Namen als universellen Markitect-Core-Typ einzuführen.

Wenn eine Beziehung mehrere Definitionsarten als Ziel erlaubt, verwendet die Skizze `Ref<Definition>`. Die Sprache muss zusätzlich pro Beziehung festlegen und prüfen, welche Zielarten zulässig sind. Ein auflösbarer Verweis auf irgendeine Definition wäre allein noch kein sinnvoller Vertrag. `ComponentDefinition.Realizes` beschreibt fachlich-technische Beiträge; `ResponsibilityDefinition` beschreibt davon getrennt organisatorische Befugnisse.

### 4.2 Dienstgrenzen und Methoden

| Vorgeschlagene Grenze | Methode | Zuständigkeit |
| --- | --- | --- |
| `IModelCompiler` | `Compile(model)` | Quellen zusammensetzen, Namen/Typen/Struktur prüfen, festen Modellstand mit Diagnostik bilden. |
| `IRepositoryInspector` | `Inspect(repository, snapshot, model)` | Konkrete Artefakte des festen Stands erfassen und zuordnen; Unklarheiten erhalten. |
| `IImpactAnalyzer` | `Analyze(model, changedIds, index)` | Auswirkungen entlang ausdrücklich definierter Beziehungen und Zuordnungen ermitteln; unbekannte Bereiche melden. |
| `IVerificationService` | `Verify(model, repository, index)` | Benannte Prüfverfahren anwenden und Ergebnisse mit Modell-, Repository- und Werkzeugbezug berichten. |

Diese Interfaces sind vorgeschlagene Verantwortungsgrenzen ohne Implementierung. Ein `CompiledModel` kann Strukturfehler enthalten; eine Realisierung darf damit erst fortfahren, wenn die erforderliche Strukturprüfung erfolgreich ist. Ein Inhaltsdigest benötigt eine festgelegte Normalisierung und darf nicht aus C#-Objekthashcodes abgeleitet werden.

Der abstrakte Apply-Mechanismus würde diese Sichten verwenden: Modell prüfen → Stand erfassen → Auswirkungen und Arbeitsaufträge bestimmen → begrenzt realisieren → prüfen/reviewen → integrieren. Zuständige Agenten erhalten Definitionen, relevante Beziehungen, Regeln, Artefakterwartungen und offene Entscheidungsspielräume. Führt eine Umsetzung zu einem nötigen Modellwechsel, muss dieser explizit entschieden werden. Die Dienste beschreiben die Vorbereitung und Prüfung; sie sind kein implementierter Apply-Orchestrator.

### 4.3 Sinnvolle Patterns

| Pattern / Prinzip | Konkrete Verwendung | Begrenzung |
| --- | --- | --- |
| Value Objects | IDs, qualifizierte Namen und Kardinalitäten als kleine Datentypen. | Definitionen behalten Identität; nicht jedes Record ist fachlich ein Value Object. |
| Composite | Rekursive Bausteinstruktur und getrennte Verantwortungshierarchie. | Querabhängigkeiten sind eigene Kanten; sie werden nicht als Elternschaft missverstanden. |
| Specification / Design by Contract | Regeln, Geltungsbereiche, Vor-/Nachbedingungen und Nachweispflichten als adressierbare Daten. | Prosa wird nicht durch einen Klassennamen zu einer ausführbaren logischen Formel. |
| Ports and Adapters | Modelllogik getrennt von Dateisystem, Prüfwerkzeugen und Agentenausführung. | Die C#-Interfaces illustrieren Grenzen; konkrete Adapter sind nicht entworfen oder gebaut. |
| Factory / Builder | Lesbare Zusammensetzung des Beispiels und spätere sichere Erstellung validierter Objekte. | Die vorhandene Factory komponiert Daten; sie ersetzt keinen Validator. |

DDD hilft bei gemeinsamem Vokabular, abgegrenzten Kontexten und wirklichen Konsistenzgrenzen. `SoftwareSystem` ist deshalb kein gigantisches DDD-Aggregat, das das ganze Projekt atomar ändern müsste. Welche Aggregate die Bibliotheksanwendung braucht, richtet sich nach ihren Fachregeln und Persistenzgrenzen. [DDD Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf).

Aus Informationsbedarf folgt außerdem keine bestimmte Quellcodeabhängigkeit. In Clean Architecture kann der Ausleihfall eine innere Schnittstelle verlangen, die ein äußerer Speicheradapter implementiert. Laufzeitfluss und Quellcodeabhängigkeit werden darum als verschiedene Aussagen modelliert. [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html).

## 5. Welche Invarianten gelten wo?

| Ebene | Beispiele | Art des erforderlichen Nachweises |
| --- | --- | --- |
| Modellstruktur | Eindeutige IDs und qualifizierte Namen; passende auflösbare Referenzen; gültige Kardinalitäten; Eigenschaften und inverse Beziehungen stimmen überein. | Deterministische Strukturprüfung einer festgelegten Sprache. |
| Gliederung / Zuständigkeit | Elternbeziehungen sind azyklisch; im gewählten Baum höchstens ein Elternobjekt; gültige Wurzeln; Namespace vorhanden. | Struktur- und Zuständigkeitsprüfung. Nicht jeder andere Beziehungstyp muss azyklisch sein. |
| Regelgeltung | Ziele einer Regel existieren; lokale und übergreifende Geltung sind explizit; Pflicht und Ausnahme widersprechen sich nicht. | Strukturprüfung; semantische Widersprüche nur mit passendem Formalismus oder Review. |
| Realisierung | Datei ist zugeordnet oder begründet ausgenommen; Schreibverantwortung eindeutig; verlangtes Artefakt vorhanden; erzeugte Ausgabe hat maßgebliche Inputs und Erzeuger. | Vollständiger Index plus geeignete Inhalts-/Generierungsprüfung. Ein Pfad allein genügt nicht. |
| Anwendungsverhalten | Höchstens eine aktive Ausleihe; atomare Konkurrenzbehandlung; erfolgreiche Rückgabe erhält Historie; Daten nach Fehler konsistent. | Verhaltensprüfung, Analyse oder geeigneter formaler Nachweis am realisierten Stand. |
| Nutzerwert | Die Beschreibung vereinfacht Koordination; Aufwand und Fehler sinken; Ergebnisse bleiben verständlich. | Eigenständige Produkt- und Nutzerbewertung. Keine Folge eines grünen Strukturchecks. |

Die C#-Konstruktoren erzwingen diese Invarianten bislang nicht. Sie zeigen, wo Daten liegen sollen. Regeln sind in dieser Skizze typisiert adressierbar, ihre fachliche Bedeutung bleibt Prosa mit expliziten Prüfaufträgen. Ein vollständiger Autorenvertrag muss noch festlegen, welche Aussagen formal geprüft werden können und welche andere Evidenz benötigen.

Eine sinnvolle spätere Ergänzung wären begrenzte, typisierte Bedingungsausdrücke: Wertebereich, Pflichtwert, Mengeneindeutigkeit, gefilterte Kardinalität, Vergleich oder Zustandsübergang. Beispielsweise könnte die aktive Ausleihe durch eine typgeprüfte `CountWhere(returnedAt fehlt) <= 1`-Bedingung beschrieben werden. Dafür braucht es definierte Mengen-, Zeit-, Null- und Auswertungssemantik. Hier ist kein solcher Evaluator implementiert oder endgültig ausgewählt. Beliebige Turing-vollständige Modellprogramme würden die allgemeine Entscheidbarkeit von Prüfungen begrenzen und die Beschreibung schwerer kontrollierbar machen; Turing-Vollständigkeit ist deshalb kein Entwurfsziel des Modells.

## 6. Vorschlag für den kanonischen Modellvertrag

### 6.1 Eine knappe Beschreibung

> Das kanonische Modell beschreibt die geltende Definition eines Softwareprojekts: seinen Zweck und Umfang, seine Systeme und Fachbegriffe, ihr Verhalten und ihre Regeln, ihre technische und organisatorische Struktur sowie die geforderten Realisierungen und Nachweise. Es legt fest, was gelten muss und welche Entscheidungen bei der Umsetzung frei bleiben. Konkrete Dateien werden ihrer Bedeutung zugeordnet; sie müssen nicht einzeln vorgegeben werden.

„Vollständig“ wird auf einen erklärten Umfang und Detaillierungsgrad bezogen. Relevant offene Entscheidungen werden explizit gemacht. Eine solche Erklärung beweist ihre eigene Vollständigkeit noch nicht.

### 6.2 Verbindliche Vertragsinhalte als Entwurf

1. **Identität und Bedeutung:** Definitionen besitzen stabile Identität, Namen, Typ und verständliche Bedeutung. Begründung und Annahme sind von geltenden Verpflichtungen unterscheidbar.
2. **Gegenstand und Umfang:** Projekt, Systeme, Repositories, Grenzen und Ausschlüsse sind benannt. Fachtypen beschreiben Anwendungskonzepte, keine beobachteten Laufzeitinstanzen.
3. **Struktur und Beziehungen:** Eigenschaften, Kardinalitäten, Zusammensetzung, Schnittstellen und unterschiedliche Abhängigkeiten haben definierte Semantik. Hierarchien und Querbeziehungen bleiben getrennt.
4. **Verhalten:** Fähigkeiten und Operationen beschreiben Beteiligte, Eingaben, Erfolg, Zustandswirkungen und Fehler. Relevante zeitliche, konkurrierende oder externe Bedingungen müssen ausdrückbar sein.
5. **Regeln und Spielräume:** Invarianten, Qualitätsanforderungen, Geltungsbereiche, Ausnahmen, Umsetzungsvorgaben und delegierte Entscheidungen werden ausdrücklich beschrieben.
6. **Realisierung:** Das Modell erklärt, welche Artefaktrollen welche Definitionen implementieren, prüfen, erklären, konfigurieren oder bereitstellen sollen. Verbindliche Ablagekonventionen werden von optionalen technischen Entscheidungen unterschieden.
7. **Verantwortung und Arbeitsweise:** Zuständigkeiten, Befugnisse, Reviews und Verfahren sind beschreibbar. Eine bestimmte Agentenorganisation oder Regierungsform ist keine Voraussetzung jeder Projektspezifikation.
8. **Prüfung:** Erforderliche Prüfungen besitzen benannte Gegenstände, Verfahren, erwartete Evidenz und Umfang. Ergebnisse unterscheiden bestanden, verletzt, nicht geprüft und unbestimmbar.
9. **Fester Bezug:** Kompiliertes Modell, Realisierungsindex und Prüfergebnis referenzieren feste Stände. Veraltete oder unvollständige Evidenz wird nicht still weiterverwendet.
10. **Änderung:** Änderungen werden über ausdrücklich definierte Beziehungen und konkrete Zuordnungen verfolgt. Unklare Auswirkungen erzeugen Befunde; ein Agent darf daraus keine behauptete Vollständigkeit ableiten.

### 6.3 Wie ich es aufbauen würde

Ich würde **eine kleine strukturelle Sprache, erweiterbare Fachvokabulare und mehrere Lesesichten** entwerfen. Die kleine Sprache kennt Definitionen, Eigenschaften, typisierte Referenzen, Zusammensetzung, Regeln mit Geltungsbereich und ausdrücklich benannte Realisierungs-/Prüfbeziehungen. Bibliothek, Webdienst oder CLI liefern ihre eigenen Begriffe. `SoftwareSystem` und `Repository` sind verständliche, typisierte Projektionen dieses Gesamtmodells.

Autoren sollen überwiegend normale Beschreibung schreiben können. Maschinenrelevante Identitäten, Typen, Beziehungen, Eigentümer und Pflichten werden explizit strukturiert. Die konkrete Aufteilung auf YAML und Markdown muss gegen Markitects bestehende Schema/Kind/Property/Definition-Grundlage geprüft werden. Diese C#-Skizze nimmt keine neue YAML-Syntax oder zwingende Core-Kinds vorweg.

Eine vereinfachte **inhaltliche Notation, kein Parserformat**, wäre:

```text
Begriff Exemplar:
  Identität: Exemplar-ID
  Eigenschaften: Titelbezug, Zustand
  Beziehung: gehört zu genau einem Titel

Regel Eindeutige aktive Ausleihe:
  Gilt für: Exemplar und Ausleihe
  Aussage: höchstens eine Ausleihe ohne Rückgabedatum pro Exemplar
  Nachweis: definierte Konkurrenzprüfung

Fähigkeit Ausleihen:
  Nutzer: berechtigtes Mitglied
  Eingabe: Mitglied und Exemplar
  Erfolg: dauerhaft angelegte aktive Ausleihe mit Frist
  Fehler: Ablehnung ohne zusätzliche Ausleihe
  Muss erhalten: Eindeutige aktive Ausleihe

Realisierung:
  Anwendung implementiert Ausleihen
  Prüfung überprüft Eindeutige aktive Ausleihe
  Nutzerdokumentation erklärt Ausleihen und Ablehnungen
```

Die physische Gliederung unterstützt Orientierung und Kohäsion. Sie kann für Namespace und Zuständigkeitsvorschläge genutzt werden; konkrete Autorität muss ausdrücklich festgelegt sein. Der Abhängigkeitsgraph sagt zusätzlich, wer Informationen oder Verträge benötigt. Er bestimmt nicht automatisch einen Managerbaum. Bei mehreren fachlichen Interessen an einer Datei werden Beiträge und Schreibzuständigkeit getrennt beschrieben.

Die Strukturprüfung findet formale Unstimmigkeiten. Der Index zeigt, welche konkreten Artefakte die Definitionen realisieren. Verify prüft benannte Aussagen am festen Stand. Apply verwendet diese Grundlagen zur koordinierten Realisierung. Keine dieser Sichten wird zu einer weiteren manuell gepflegten Produktwahrheit.

## 7. Ableitung für Marketing

Die stärkste Produktbotschaft ist die Beziehung zwischen geltender Beschreibung und ihren vielen Realisierungen. Klassennamen, Graphen und Compiler erklären den Mechanismus im zweiten Schritt.

**Leitidee:** „Beschreibe dein System. Lass Agenten das Projekt danach gestalten.“

**Erklärung als Positionierung des angestrebten Produkts:** „Markitect macht deine Projektspezifikation zur gemeinsamen Grundlage für Code, Tests, Konfiguration und Dokumentation. Explizite Begriffe, Regeln und Beziehungen strukturieren die Arbeit der Agenten und machen Abweichungen nachvollziehbar.“ Diese Formulierung ist hier eine Marketinghypothese; sie setzt keinen neuen Nachweis der aktuellen Fähigkeiten voraus.

**Versprochener Ansatz:** Eine maßgebliche Beschreibung pflegen; ihre Auswirkungen auf unterschiedliche Artefakte systematisch verfolgen; Umsetzungen gegen definierte Regeln und Prüfungen bewerten. Daraus erhoffen wir weniger manuelle Synchronisation, verständlichere Repositories und mehr begründetes Vertrauen bei höherer Agentenautonomie. Diese Wirkungen müssen gesondert belegt werden.

Eine Produktdemonstration könnte zeigen: Ausleihregel ändern → betroffene Verhaltensverträge, Umsetzung und Prüfungen sichtbar machen → Agenten arbeiten mit klaren Zuständigkeiten → Modell und Repository am neuen Stand abgleichen. Konkrete Pfade kommen aus dem Realisierungsindex. Eine nur prosebezogene oder unklar zugeordnete Folge wird als Unsicherheit sichtbar. Die Demonstration ist hier vorgeschlagen, nicht ausgeführt oder als aktuelle Fähigkeit nachgewiesen.

Ein passendes Dreischrittmotiv ist **Beschreiben → koordiniert umsetzen → gezielt prüfen**. „Nie wieder vergessene Dateien“ oder „jede Änderung ist automatisch korrekt“ wären durch diesen Entwurf nicht gedeckt. Ein vertretbarer Anspruch ist, die Verantwortung für Zusammenhänge aus spontaner Agentenerinnerung in explizite Struktur und überprüfbare Verfahren zu verlagern.

Für die Chats sind drei verschiedene Beiträge geeignet: Marketing erhält Problem, Mechanismus und Nutzenhypothesen; Concepts erhält Begriffe, Geltung und Entwurfsannahmen; Ideas erhält die noch offene formale Regelsprache und eine spätere Bibliotheksdemonstration. Das hält Produktidentität, Definitionsentwurf und mögliche Erweiterungen unterscheidbar.

## 8. Einordnung und Herkunft

Diese Skizze entwickelt das vereinbarte Grundmodell weiter. Ihr Vergleich mit dem bestehenden Produkt bleibt auf die im [übergeordneten Entwurf](canonical-model-draft.md#10-verhältnis-zu-markitects-bestehendem-stand) gelesenen Dokumentverträge begrenzt. Die Entwurfstypen, Dienstgrenzen und Mechanismen sind keine Behauptung über Markitects Implementierung.

- Arbeitsort: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Produktbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15` von `origin/main`; kein erneutes Zusammenlegen anderer Branches.
- Dokumentationsstand vor dieser Ergänzung: `34ac689f36161e48b01df4e6d043ee4e51c5be50`; Worktree beim Beginn sauber.
- Lokale Ergänzungen: dieses Dokument, zwei illustrative C#-Dateien und Navigationslinks im vorhandenen Entwurfsordner.
- Beauftragt im Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`; keine zusätzlichen angenommenen Entscheidungen.

Die Klassenbelegung wurde im Lesen auf konsistente Referenzen und die Bibliotheksbedeutung geprüft. Das ist Entwurfsreview, kein Compilerlauf und kein Nachweis vollständiger Fach-, Sicherheits- oder Betriebsregeln. Die lokale Commit-SHA der Ergänzung wird bei der Übergabe berichtet.

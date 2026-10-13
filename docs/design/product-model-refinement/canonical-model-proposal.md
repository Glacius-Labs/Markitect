# Proposal: Kanonisches Modell mit nominalen Regeln und methodischem Review

Erstellt am 11. Oktober 2026; konsolidiert am 12. und architektonisch erweitert am 13. Oktober 2026. Status: beauftragter Konzept- und Umsetzungsentwurf auf Basis des Bibliotheksgesprächs. Dateiname `mandate.yaml`, der Auftragstyp `Mandate`, die fachlichen Kinds `Concept`, `Rule`, `UseCase` sowie `System`, `Component`, `Module`, `Interface` sind gewählt. Weitere Syntax- und Referenzdetails sind Vorschläge, keine implementierten Produktverträge.

## 1. Auftrag und Maßstab

Die [Leitfrage](system-description-notes.md#leitfrage) lautet: **Wie beschreibt man ein Repository für ein Softwaresystem?** Das kanonische Modell soll das geltende Projekt so beschreiben, dass ein Umsetzer daraus ein passendes Repository realisieren kann, ohne jede Datei vorzugeben oder wesentliche Entscheidungen unbemerkt zu erfinden.

Das Modell enthält Zweck, Umfang, Begriffe, Verhalten, Regeln, technische und organisatorische Festlegungen sowie Realisierungs- und Prüfpflichten. Vollständigkeit ist auf einen erklärten Umfang und Detaillierungsgrad bezogen. Nicht beschriebene Entscheidungen werden geklärt oder ausdrücklich delegiert; ein grüner Strukturcheck beweist keine vollständige Spezifikation.

Die drei Produktblöcke bleiben: kanonisches Modell, realisierendes Repository und abstrakter Apply-Mechanismus. Git-Historie und Work Items sind weiterhin außerhalb der geltenden Systembeschreibung. Ein beobachteter Implementierungsfehler verändert die kanonische Regel nicht automatisch.

Die [aktuelle Bibliotheksdarstellung](library-model-example.md) und ihr [Architektur-YAML](library-architecture-example.md) konkretisieren diese Fassung. Der [Änderungsdurchlauf](library-change-walkthrough.md) zeigt Koordination und Review. Der [Main-Abgleich vom 11. Oktober](canonical-model-main-comparison.md) ist historische, SHA-gebundene Implementierungsevidenz; er wurde heute nicht erneuert.

## 2. Empfohlener Aufbau

**Arbeitsrichtung:** kleiner struktureller Core, begrenztes nominales Projektvokabular, verbindliche Fachprosa, explizite Beziehungen und methodische Reviews. Zusätzliche Fachformalismen werden durch konkreten Bedarf begründet.

| Bestandteil | Zuständigkeit |
| --- | --- |
| Struktureller Core | Definitionen, Identitäten, Schemas, Werttypen und typisierte Referenzen; deterministische Normalisierung und Diagnosen. |
| Projektvokabular | Bedeutung der Definitionen und ihrer Beziehungen; insbesondere Regeln, Verantwortungen, Realisierung und Prüfanschlüsse. |
| Kanonische Projektdefinitionen | Was gilt, welche Entscheidungen frei bleiben und welche Beiträge erwartet werden. |
| Abgeleitete Sichten | Dokumentation, Kontext für Mandate und ihre Ausführer, Graph, Impact-Erklärung und konkreter Dateibestand. |
| Ausführung und Evidenz | Agentenbindung, Kandidaten, unabhängige Bewertung, Checks und Apply an festen Ständen. |

### 2.1 Eine maßgebliche Quelle pro Aussage

Eine geltende Aussage wird einmal definiert und andernorts referenziert. YAML trägt die maschinenrelevante Struktur und zunächst mehrzeilige Prosa. Lesbare Dokumentation wird daraus abgeleitet oder erfüllt einen anderen ausdrücklich beschriebenen Informationsauftrag. Dieselbe Spezifikation wird nicht unabhängig in YAML, Markdown und Agentenanweisungen kopiert.

Falls Markdown später kanonischer Inhalt wird, braucht es eindeutige Verantwortung, explizites Laden und Digest-Bindung. Externe Vorgaben und Beobachtungen behalten Quelle, Stand und Aktualitätsverantwortung; die gewählte Bindung an sie ist von ihrer externen Herkunft unterscheidbar.

## 3. Was beschrieben werden können muss

Die Perspektiven sind ein Fragenkatalog zur Abdeckung, keine Pflichtordner und kein universeller Klassenkatalog.

| Perspektive | Zu beschreibende Inhalte | Bibliothek |
| --- | --- | --- |
| Zweck und Kontext | Problem, Nutzen, Akteure, Grenzen, Annahmen, Ausschlüsse. | Bücher ausleihen und zurückgeben; Beschaffung außerhalb. |
| Fachstruktur | Gegenstände, Eigenschaften, Beziehungen, Zustände und Kardinalitäten. | Titel, Exemplar, Mitglied, Ausleihe. |
| Verhalten und Regeln | Voraussetzungen, Erfolg, Fehler, Zustandswirkungen, Invarianten, Konkurrenz und Wiederholung. | Ausleihen, Rückgabe, höchstens eine aktive Ausleihe. |
| Architektur und Betrieb | Bausteine, Schnittstellen, Abhängigkeiten, Umgebungen, Ressourcen, Konfiguration und Betriebsverfahren. | Datenhaltung und atomare Vergabe; Technik ausdrücklich festlegen oder delegieren. |
| Verantwortung und Arbeitsweise | Auftrag, zuständiger Bereich, Delegation, Befugnisse, Grenzen und Reviews. | Mandat für Ausleihregeln und deren Realisierung. |
| Realisierung und Nachweise | Benötigte Beiträge, Pfadregeln, Checks, Prüfung und Aussagegrenzen. | Implementierung, Tests und Bedienungsdokumentation. |

Quellcode/Toolchain, externe Abhängigkeiten, Daten/Migrationen, Tests/Analysen, Dokumentation, Build/Paketierung/CI, Infrastruktur/Konfiguration, Monitoring/Dashboards und Skills/Workflows sind auf Anwendbarkeit zu prüfen. Ihre Erwähnung macht sie nicht für jedes Projekt verpflichtend.

### 3.1 Fachtypen und Instanzen

Eine Modelldefinition `copy` beschreibt den Fachtyp Exemplar. Sie ist kein konkretes Buch. Laufende Ausleihen gehören zur Anwendung.

Drei Angaben sind zu unterscheiden: Kardinalität einer Modellproperty, beschriebene Kardinalität einer Fachbeziehung und eine gefilterte Invariante über Laufzeitinstanzen. „Höchstens eine aktive Ausleihe“ wird nicht allein durch Zählen der Modellreferenzen bewiesen.

### 3.2 Entscheidungen und Freiräume

Verbindliche Technik und Algorithmen dürfen Teil der Definition sein. Andere Entscheidungen bleiben ausdrücklich beim zuständigen Bereich. Atomare Vergabe kann beispielsweise Pflicht sein, während die Datenbanktechnik frei bleibt.

Eine dokumentierte Entscheidung referenziert ihren Gegenstand und bewahrt Begründung und Provenienz. Ein Commit oder Actor-Feld authentifiziert keine menschliche Freigabe. Offene Punkte sind nicht still delegiert.

## 4. Beziehungen, Gliederung und Verantwortung

| Struktur | Bedeutung | Grenze |
| --- | --- | --- |
| Namespace-/Dateibaum | Kohäsive Ablage und adressierbarer Bereich. | Die bekannte Loader-Konvention bindet Namespace und Verzeichnis; sie ist keine Laufzeitarchitektur. |
| Verantwortungsbaum | Parent, Zuständigkeit und ausdrückliche Delegation. | Ein Graphpfad allein verleiht keine Befugnis. |
| Fach-/Architekturgraph | Kontext, Verpflichtungen, Bausteine und Zusammenarbeit zwischen Definitionen. | Querbeziehungen verändern keine strukturelle oder organisatorische Elternschaft. |
| Ausführungsablauf | Abgeleitete Aufgaben, Reviews und Integration. | Eine fachliche Dependency ist nicht automatisch ein Ausführungsplan. |

Ein kleiner Bibliotheksausschnitt kann ein Wurzelmandat besitzen. Untergeordnete Mandate entstehen, wenn begrenzte Aufträge und Bereiche sie rechtfertigen. Übergreifende Interessen brauchen ausdrückliche Mandate oder Beziehungen. Zyklen werden je Relation beurteilt; Parent-Zyklen sind anders zu behandeln als mögliche fachliche Querreferenzen. Das Mandat beschreibt den Auftrag; seine Bindung an einen ausführenden Agenten wird separat festgelegt.

### 4.0 Architekturbegriffe und ihre Abgrenzung

Am 13. Oktober 2026 beauftragt: System, Component, Module und Interface samt Beziehungen werden als eigenständige Projektvokabular-Kinds festgehalten. Sie sind keine neuen universellen Core-Primitiven.

| Kind | Bedeutung | Bibliothek |
| --- | --- | --- |
| `System` | Zweck, Nutzer und erklärte Grenze einer zusammengehörigen Lösung zur Umgebung. | Bibliotheksanwendung. |
| `Component` | Technischer Baustein mit Laufzeit-/Bereitstellungsrolle. | Webfrontend, Backenddienst, Datenbankdienst. |
| `Module` | Kohäsive logische Gliederung, ohne notwendige eigene Laufzeit. | Bestand, Mitgliedschaft, Ausleihe, Adapter. |
| `Interface` | Benannter Vertrag der Zusammenarbeit, getrennt von seinem Anbieter. | Bibliotheks-API und Speicherverträge. |

Ein System kann im Projektumfang realisiert oder extern beschrieben sein. Für externe Systeme werden relevante Grenzen/Verträge deklariert, ohne ihre Interna als eigene Realisierungspflicht zu erfinden. Die Component-Definition ist eine bewusste Laufzeitbedeutung dieses Entwurfs.

Nutzerentscheidung vom 13. Oktober 2026: `ownership` entfällt vorerst. Der beschriebene Realisierungsumfang bleibt Prosa und Artefaktzuordnung: Der Datenbankserver kann wiederverwendete Fremdsoftware sein, während Konfiguration, Schema und Migrationen Projektbeiträge sind. Ein strukturiertes Feld zur Unterscheidung eigener und externer Systeme oder feinerer Zuständigkeiten ist nicht festgelegt.

Nutzerentscheidung vom 13. Oktober 2026: `contains` ist die einheitliche Beziehung für direkte strukturelle Zusammensetzung. Ihre Referenzen nennen immer ausdrücklich den Zielkind; die bisherigen Felder `components` und `modules` entfallen im aktuellen Entwurf. Für System sind zunächst Component-Ziele, für Component und Module jeweils Module-Ziele vorgesehen. Weitere Kinds können dieselbe Beziehung verwenden, sobald ihr Schema zulässige Kinder und Zusammensetzungsregeln definiert; diese Wiederverwendbarkeit gibt nicht automatisch beliebige Zieltypen frei.

Im ersten vollständigen Bibliotheksvertrag hat ein Component genau ein System als Parent; ein Module genau einen Component oder Parent-Module. Parents werden aus den `contains`-Listen abgeleitet, nicht als zweite manuelle Wahrheit gepflegt. Selbstbezüge und Zusammensetzungszyklen sind ungültig. Components dürfen bewusst nicht modellierte Interna haben, etwa der bereitgestellte Datenbankdienst. Zusammensetzung erklärt weder Agentenbefugnis noch Interface-Zusammenarbeit, Implementierungsabhängigkeit oder Realisierungsbeitrag; dafür bestehen eigene Beziehungen.

`provides` und `consumes` stehen am konkreten Anbieter/Verbraucher. Die veröffentlichte Component-Sicht kann aus seinen Modules abgeleitet werden; dieselbe Providerbeziehung wird nicht nochmals auf dem Parent kopiert. Direkt deklarierte Component-Verträge eignen sich für undurchsichtige Laufzeitbausteine. Konsumierte Interfaces benötigen im ersten Beispielvertrag einen eindeutigen Provider im erklärten Umfang; Mehrfachprovider-Auswahl ist noch nicht entworfen. Externe Systems können Verträge direkt bereitstellen.

`Module.dependsOn` beschreibt explizite Implementierungs-/Buildabhängigkeit zu Modules. Interface-Konsum ist Vertrags-/Zusammenarbeit und importiert nicht automatisch die Anbieterimplementierung. Keine doppelte Pflege derselben abgeleiteten Tatsache. Kommunikations- und Dependencyzyklen werden je Relation und geltenden Rules beurteilt, nicht pauschal verboten.

`Module.realizes` beziehungsweise ein direkter Component-Beitrag ordnet gewünschte Fachverantwortung der Architektur zu. Mehrere Bausteine können einen UseCase tragen; Prosa erläutert Bedienung, Orchestrierung oder Speicherung. Die Zuordnung ist weder Codebeleg noch Schreibowner. `Artifact.realizes` erklärt dagegen erwartete Repositorybeiträge; konkrete Dateien und Nachweise werden getrennt erhoben.

### Automatischer Umfang von realizes

Nutzerentscheidung vom 13. Oktober 2026: `realizes` schließt sein ausdrücklich referenziertes Ziel und alle unmittelbar oder mittelbar über `contains` enthaltenen Definitionen ein. Dafür gibt es kein `scope`-Feld und keinen zusätzlichen Schalter. Ein Artefakt für das Backend umfasst entsprechend seiner Rolle auch die enthaltenen Backend-Modules. Für einen kleineren Bereich wird dessen Zieldefinition direkt referenziert.

Der effektive Umfang ist die Vereinigung aus jedem direkten realizes-Ziel und dessen transitiver contains-Unterstruktur. Mehrfach erreichte Definitionen werden nur einmal als Ziel geführt; die ausdrücklichen und abgeleiteten Pfade bleiben erklärbar. Die Regel gilt überall, wo realizes zugelassen ist. Zulässige direkte Referenzziele und zulässige contains-Kinder bleiben durch ihre jeweiligen Schemas begrenzt.

Die Zielmenge für direkte realizes-Referenzen filtert enthaltene Nachfahren nicht nachträglich heraus. Falls contains künftig weitere Kinds zulässt, muss deren Einschluss mit der beschriebenen Beitragsrolle vereinbar sein; diese Schemaerweiterungen sind gesondert zu prüfen. Die gewählte Umfangsregel umfasst alle enthaltenen Definitionen.

Diese Umfangsableitung folgt ausschließlich `contains`. Sie erweitert sich nicht automatisch über `uses`, `requires`, `dependsOn`, `provides`, `consumes`, `appliesTo` oder eine weitere realizes-Beziehung. Solche Beziehungen liefern eigenen Kontext beziehungsweise Verpflichtungen und Beiträge; sie machen deren Ziele nicht zu enthaltenen Bestandteilen. Die Artefaktrolle bestimmt den Beitrag: Implementation implementiert, Dokumentation erklärt, Verifikation stellt Prüfungen bereit und Konfiguration konfiguriert. Der abgeleitete Auftrag beweist keine tatsächliche Realisierung oder bestandene Prüfung und erzeugt keine zusätzliche Schreibbefugnis.

Im derzeit dokumentierten Bibliotheksmodell enthält das Backend technische Modules. Concepts, Rules, UseCases und Interfaces sind dort über andere Beziehungen verbunden; eine zukünftige Einordnung dieser Definitionen unter contains ist noch zu besprechen. Ihre unabhängigen Artefaktzuordnungen bleiben deshalb ausdrücklich bestehen. Eine zusätzliche allgemeine Realisierungssemantik über alle Beziehungstypen ist mit dieser Entscheidung nicht eingeführt.

### 4.1 Minimaler Beziehungsvertrag

| Feld | Bedeutung | Kandidat für zulässige Ziel-Kinds |
| --- | --- | --- |
| `uses` | Benötigter Kontext. | `Concept`, `Rule`, `UseCase`, `System`, `Component`, `Module`, `Interface` |
| `UseCase.requires`, `Interface.requires` | Normative Verpflichtung. | `Rule` |
| `System.contains` | Direkte strukturelle Bestandteile; Zielkind immer ausdrücklich. | `Component` |
| `Component.contains`, `Module.contains` | Direkte strukturelle Bestandteile; Zielkind immer ausdrücklich. | `Module` |
| `provides`, `consumes` auf Component/Module, bei externem System an dessen Grenze | Vertrag anbieten beziehungsweise benötigen. | `Interface` |
| `Module.dependsOn` | Implementierungs-/Buildabhängigkeit. | `Module` |
| `Module.realizes`, direkter `Component.realizes` | Gewünschter architektonischer Fachbeitrag einschließlich contains-Unterstruktur. | `Concept`, `Rule`, `UseCase` |
| `Interface.useCases` | Über den Vertrag angebotene fachliche Fähigkeiten. | `UseCase` |
| `Rule.appliesTo` | Ausdrücklicher Geltungsbereich. | `Concept`, `Rule`, `UseCase`, `System`, `Component`, `Module`, `Interface` |
| `Artifact.realizes` | Erwarteter Repositorybeitrag entsprechend der Rolle, einschließlich contains-Unterstruktur. | `Concept`, `Rule`, `UseCase`, `System`, `Component`, `Module`, `Interface` |
| `Artifact.checks` | Ausführbares Prüfverfahren. | `Check` |
| `Check.uses` | Aussagen im Prüfgegenstand. | `Concept`, `Rule`, `UseCase`, `System`, `Component`, `Module`, `Interface` |
| `Mandate.parent` | Direkt übergeordnetes Mandat. | `Mandate` |
| `Decision.subject` | Gegenstand einer begründeten Entscheidung. | `Concept`, `Rule`, `UseCase`, `System`, `Component`, `Module`, `Interface` |
| `Decision.actor` | Bezug auf das verwendete Mandat; Feldname und Provenienzbedeutung noch einzeln zu prüfen. | `Mandate` |

Bei `requires` erklärt Fachprosa zunächst Voraussetzung oder erhaltene Invariante. `uses` sagt weder „gilt für“ noch etwas über die Zahl konkreter Bücher. `Rule.appliesTo` wirkt im ersten Vertrag direkt auf die benannten Ziele; dort muss die Regel im Kontext und Prüfumfang sichtbar sein. Automatische Subtree-Geltung wird nicht vorausgesetzt. Interfaceoperationen, Daten und Fehler werden zunächst in Prosa erklärt; eine UseCase-Referenz allein ist kein vollständiger Protokollvertrag.

Die hier eingeschränkte `requires`-Zielmenge ist ein neuer Vorschlag für diesen Bibliotheksvertrag. Der bisherige Statement-Vertrag konnte auch andere Statement-Verpflichtungen ausdrücken. Bestehende Projekte dürfen deshalb nicht automatisch oder verlustbehaftet umgedeutet werden.

### 4.2 Identität und Referenzsyntax

Identität umfasst API-Version, Kind, Namespace und Name. Umbenennung oder Kind-Wechsel benötigt Referenz- und Evidenzbehandlung. Ein UUID-Zwang wird nicht vorgeschlagen.

Nutzerentscheidung vom 13. Oktober 2026, anschließend um `contains` präzisiert: Eine Referenz nennt `kind` bei mehreren zulässigen Ziel-Kinds sowie immer bei `contains`. Bei anderen Feldern mit genau einem zulässigen Ziel-Kind bestimmen Schema und Beziehungsfeld den Typ; die Referenz enthält Namespace und Name. Die Zielauflösung bleibt typisiert, nicht allein namensbasiert. Der Kind einer Definitionsdatei bleibt ausdrücklich angegeben.

`requires` ist auf Rule festgelegt:

```yaml
requires:
  - namespace: library
    name: one-active-loan
```

`uses` lässt mehrere Ziel-Kinds zu; auch eine Liste mit derzeit nur einem Concept benötigt deshalb den Kind:

```yaml
uses:
  - kind: Concept
    namespace: library
    name: copy
```

Ohne `kind` werden die einheitlich typisierten Felder `provides`, `consumes`, `dependsOn`, `requires`, `useCases`, `checks`, `parent` und `actor` geschrieben. Bei `contains`, `uses`, `realizes`, `appliesTo` und `subject` ist `kind` erforderlich. Bei gemischten Referenzen ist die zulässige Zielmenge aus Abschnitt 4.1 maßgeblich, nicht die zufällige Zusammensetzung einer konkreten Liste. Vor Digest, Kontext, Graph und Impact wird ein aus dem Schema abgeleiteter fester Ziel-Kind ergänzt; Kürzung der Schreibweise entfernt keine Identität oder Kante.

**Gewählte einheitliche Form:** `contains` benennt die Beziehung und `kind` den enthaltenen Baustein. Auch bei momentan nur einem erlaubten Zieltyp bleibt der Kind sichtbar. Das ist eine bewusste Ausnahme von der Kurzschreibweise, damit dieselbe Form über verschiedene Quell-Kinds und später gegebenenfalls gemischte Kinder wiederverwendbar ist. Der Validator prüft trotzdem den angegebenen Kind gegen das Schema des Quell-Kinds.

```yaml
contains:
  - kind: Component
    namespace: library
    name: backend
```

Die API-Version des Ziels wird im Beispiel aus dem versionierten Referenzvertrag bestimmt: Alle Ziel-Kinds dieser Fassung gehören ausdrücklich zur gleichen `vNext`-Version. Das ist eine vorgeschlagene Versionierungsregel, keine freie Auflösung auf eine zufällig vorhandene Version. Cross-Version-Referenzen sind nicht implizit freigegeben. Jede Property hat eine endliche Zielmenge; „beliebiger Kind“ ist kein Ersatz für Typprüfung. Auch `uses`, Realisierung, Checks und Entscheidungsbezüge brauchen konsistente Auflösung.

**Implementierungsentscheidung offen:** Der bisher geprüfte Core-Vertrag benennt pro Reference-Property genau einen Zielkind. Gemischte Zielmengen verlangen entweder eine eng begrenzte, explizite Referenzunion oder getrennte typisierte Properties. Der gemeinsame YAML-Kandidat verwendet die Kind-angereicherte Referenzform bei gemischten Zielmengen und stets bei `contains`; bei anderen festen Zieltypen folgt der Kind aus dem Schema. Ob Core und Validator dies ohne neue Primitive tragen können, ist vor der Implementierung an aktuellem Main zu prüfen. Die Kurzform darf die bisherigen Diagnosen und Graphkanten nicht umgehen.

## 5. Realisierung ohne Aufzählung jeder Datei

Das Modell beschreibt benötigte Beiträge und ausgewählte technische Vorgaben. Artefakte ordnen diese Aussagen über exakte Pfade oder Verzeichnispräfixe zu. Der konkrete Dateibestand wird an einem festen Stand erfasst und daraus klassifiziert.

Modellquellen im Manifest explizit zu nennen ist etwas anderes als jede Anwendungsdatei einzeln zu spezifizieren. Private Hilfsdateien benötigen keine eigene Fachdefinition. Viele Aussagen können eine Datei betreffen; eine Aussage kann viele Dateien benötigen. Jede konkrete Schreibausgabe braucht trotzdem einen eindeutigen Owner.

| Urteil | Was es aussagt |
| --- | --- |
| Struktur gültig | Definierte Typen, Identitäten und Beziehungen passen zum gewählten Sprachvertrag. |
| Coverage vollständig | Jede Datei im erklärten Umfang ist zugeordnet oder ausdrücklich ausgenommen. |
| Realisierung konform | Die untersuchten Inhalte erfüllen die untersuchten Aussagen. |
| Beschreibung ausreichend | Der erklärte Umfang ist fachlich hinreichend beschrieben und freie Entscheidungen sind klar. |

Keines dieser Urteile ersetzt die anderen. Eine deklarierte Zuordnung beweist nicht, dass der Dateiinhalt die Aussage erfüllt. Ignorieren und Übergangsausnahmen sind explizit und behalten ihre Begrenzung.

## 6. Autorenformat und Sprachform

### 6.1 Verständliche Quellen

Eine Definition pro Datei übernimmt die Grenze des bisher geprüften Loaders. Der Verzeichnisbereich entspricht im Beispiel dem Namespace. Ein anderer Loadervertrag könnte später mehrere Definitionen pro Datei unterstützen; das gehört nicht zum ersten Konzeptschritt.

```text
.markitect/
  project.yaml
  model/
    mandate.yaml
    library/
      copy.yaml
      loan.yaml
      active-loan.yaml
      eligible-member.yaml
      available-copy.yaml
      one-active-loan.yaml
      borrow.yaml
      return.yaml
      library-system.yaml
      frontend.yaml
      backend.yaml
      database.yaml
      circulation.yaml
      library-api.yaml
      frontend-implementation.yaml
      backend-implementation.yaml
      database-configuration.yaml
      library-verification.yaml
      library-documentation.yaml
      library-contract-check.yaml
```

Der vollständige Baum und das Manifest stehen im [Bibliotheksbeispiel](library-model-example.md). Eine feinere fachliche Gliederung kann später Bestand, Mitgliedschaft und Ausleihe in eigene Namespacebereiche teilen.

### 6.2 Empfohlene erste Sprachform: Concept, Rule und UseCase

Der Nutzer hat `Concept`, `Rule` und `UseCase` als eigene Kinds gewählt. Concepts beschreiben fachliche Gegenstände, Begriffe und Zustände; Rules normative Bedingungen und Invarianten; UseCases Fähigkeiten und Abläufe. Fachlicher Inhalt bleibt zunächst verbindliche Prosa.

Für diese Unterscheidung entfällt `category`. Gemeinsame Header und Beschreibungsfelder folgen derselben Konvention; jeder Kind besitzt seinen geschlossenen Feld- und Referenzvertrag. Der nominale Unterschied ermöglicht gezielte Zieltypprüfung, ohne eine vollständig formalisierte Fachsprache zu verlangen.

Die Systemübersicht ist nach der Architekturergänzung ein `System`. Components, Modules und Interfaces erklären technische Aufteilung und Zusammenarbeit. Ein allgemeiner Statement-Kind wird nicht vorausgesetzt. Nutzerentscheidung vom 13. Oktober 2026: Der Definitionstyp `Manager` wird durch `Mandate` ersetzt. Er beschreibt einen Auftrag und dessen Grenzen; ein ausführender Agent ist separat gebunden. Weitere Beschreibungstypen benötigen einen konkreten Ausdrucksbedarf.

#### 6.2.1 Feldkandidat für den ersten Vertrag

Diese Arbeitstabelle konkretisiert den Entwurf für P0; sie behauptet kein vorhandenes Schema. Alle Definitionen besitzen `apiVersion`, `kind`, `metadata.name`, `metadata.namespace` und einen lesbaren `purpose`. Der Namespace kann für die Wurzel leer sein. Header und Spec sind geschlossen; unbenannte Zusatzfelder werden nicht still akzeptiert.

| Kind | Spec-Felder des Kandidaten | Pflicht und Begrenzung |
| --- | --- | --- |
| `Concept` | `description`, `public`, `uses` | Beschreibung erforderlich; Kontextbezüge optional. Keine Kategorie concept. |
| `Rule` | `description`, `public`, `uses`, `appliesTo` | Beschreibung erforderlich; Kontext und Geltungsziele explizit. Keine formale Bedingung vorausgesetzt. |
| `UseCase` | `description`, `public`, `uses`, `requires` | Beschreibung erforderlich; Kontextbezüge optional, Verpflichtungen als Rule-Referenzen. Keine Kategorie use-case. |
| `System` | `description`, `public`, `contains`, `uses`, bei externer Grenze `provides`/`consumes` | Zweck, Grenze und Realisierungsumfang in Prosa; contains mit ausdrücklichem Component-Kind, externe Interna optional. Kein ownership-Feld. |
| `Component` | `description`, `public`, `contains`, `provides`, `consumes`, `realizes`, `uses` | Laufzeitrolle erklären; contains mit ausdrücklichem Module-Kind, Verträge passend zuordnen. |
| `Module` | `description`, `public`, `contains`, `provides`, `consumes`, `dependsOn`, `realizes`, `uses` | Logische Aufgabe/Beitrag; optionale untergeordnete Modules mit ausdrücklichem Kind, genau ein struktureller Parent im vollständigen Beispiel. |
| `Interface` | `description`, `public`, `useCases`, `uses`, `requires` | Vertrag erklären; Verpflichtungen als Rules; Provider/Consumer stehen an Bausteinen. |
| `Mandate` | `parent`, `owns`, `instructions` | Auftrag mit höchstens einem übergeordneten Mandat, optionaler Pfadliste und lesbaren Anweisungen. Bisherige Felder sind Prüfkandidaten; Benennung eines Mandats ist keine Agentenbindung oder authentifizierte Autorisierung. |
| `Artifact` | `role`, `realizes`, `paths`, `checks`, `required`, `reason` | Rolle implementation/documentation/verification/configuration; optionale Ziel-/Pfad-/Checklisten, Erforderlichkeit und Begründung. |
| `Check` | `command`, `uses`, `limitation` | Nicht leere Liste literal argv; deklarierter Gegenstand und Evidenzgrenze. Ein Befehl ist keine Shell-Prosa. |
| `Decision` | `subject`, `decision`, `reason`, `actor` | Genau ein Subject und Actor sowie lesbare Entscheidung und Begründung. Actor authentifiziert keine Befugnis. |

Für fachliche und architektonische Definitions-Kinds ist `public` als Boolean vorgesehen; ohne ausdrückliches Public bleibt eine fremde Definition im Kandidaten privat. Ihre `uses`-Listen zeigen nur auf die endliche Zielmenge aus Abschnitt 4. Referenzen werden gegen die Zielmengen aus Abschnitt 4 geprüft. Die Mengenfelder bestimmen Modellwerte; sie zählen keine Laufzeitinstanzen. Ausführliche Defaults, Kardinalitäten und Normalisierung werden im versionierten P0-Vertrag festgeschrieben. Weitere Felder brauchen einen erklärten Bedarf und ihre Behandlung in allen Sichten.

### 6.3 Begrenzte Formalisierung

Strukturierte Fachproperties, Vor-/Nachbedingungen und Zustandsmodelle bleiben mögliche Erweiterungen. Sie werden zuerst an wiederkehrenden Fehlern und Autorenaufwand begründet. Ein eigener Ausdrucksevaluator oder Turing-Vollständigkeit ist keine Voraussetzung.

Eine spätere formale Regel braucht definierte Instanz-, Mengen-, Zeit- und Zustandssemantik. Der Compiler ihrer Definition beweist dadurch noch nicht, dass die Anwendung sie erhält. Die früheren [Facettenausschnitte](library-filesystem-example.md) und der [C#-Entwurf](csharp-model-design.md) bleiben datierte Vorarbeiten.

### 6.4 Offene Frage: Strenge und Methodik

Nutzerimpuls: „Eventuell reicht ein methodischer Ansatz bereits für das was Markitect erreichen will.“ Das ist eine Produktthese, kein gemessener Befund.

| Strukturell erzwingen | Methodisch beurteilen |
| --- | --- |
| Identitäten, geschlossene Felder, Werttypen und erlaubte Ziel-Kinds. | Verständlichkeit und fachliche Vollständigkeit der Beschreibung. |
| Referenzauflösung, Parent-/Ownerregeln und Dateizuordnung. | Vollständigkeit und Bedeutung der deklarierten Beziehungen. |
| Feste Stände, Änderungsvergleich und konservativer Umgang mit unbekanntem Umfang. | Erhalt der Regeln in Umsetzung, Dokumentation und Prüfungen. |
| Deklarierte Check-Bindung und tatsächlich aufgezeichnete Ergebnisse. | Ob die Verfahren die neue Aussage geeignet untersuchen. |

Fehler, Warnung und Reviewpflicht sind unterscheidbare Reaktionen. Ein allgemeiner Strictness-Schalter oder konfigurierbare Profile sind noch nicht entschieden. Fehlerhafte Referenzen dürfen nicht als bekannte korrekte Zuordnung behandelt werden; unsichere Wirkung darf nicht still zu engem Impact werden. Jede Zusage bleibt an tatsächlich durchgeführte Prüfung und deren Grenzen gebunden.

### 6.5 Alternative B: Minimale Struktur und methodisches Review

Die zuvor als Alternative B dokumentierte methodische Variante ist die Arbeitsbasis dieser konsolidierten Empfehlung. Sie wird mit den gewählten nominalen Kinds Concept, Rule und UseCase verbunden. Daraus folgt keine Annahme, dass umfangreiche Fachfacetten grundsätzlich unnötig sind; die weitergehende Formalisierung bleibt eine alternative Tiefe des gleichen Modells.

Das [bisherige v1alpha1-Beispiel](library-methodical-alternative.md) bleibt als datierte Vorstufe erhalten. Das [aktuelle vNext-Beispiel](library-model-example.md) zeigt erstmals alle Regeln und Referenzen konsistent in der neuen Entwurfsrichtung. `vNext` und seine Referenzform sind Platzhalter, keine unterstützte Version.

### 6.6 Gewählter Dateiname und bevorzugte Kind-Richtung

Nutzerentscheidung: „Wir nehmen mandate.“ Die Verantwortungsdatei heißt `mandate.yaml`. Ergänzende Nutzerentscheidung vom 13. Oktober 2026: „Okay, dann nehmen wir Mandate statt Manager.“ Der Modell-Kind heißt jetzt `Mandate`; dies benennt den Auftrag und keine ausführende Runtime-Rolle um. Identität und Referenzziele werden im vNext-Entwurf entsprechend angepasst. Die vorhandenen Felder werden beim schrittweisen Typreview gesondert geprüft; ihre bisherige Form ist durch die Umbenennung nicht abschließend bestätigt.

Nutzerentscheidung vom 12. Oktober 2026: „Okay, dann nehmen wir Concept, Rule und UseCase als eigene Kinds. Aktualisiere das Konzept und das Bibliotheksbeispiel entsprechend auf deinem Branch.“ Diese drei Kinds ersetzen die bisherige Statement-/Category-Einordnung im aktuellen Entwurf. Feld-, Versions- und Referenzdetails bleiben zu konkretisieren. Der semantische Begriff Mandat beschreibt Auftrag, Zuständigkeit, Befugnisse und Grenzen, ohne selbst eine authentifizierte Autorisierung zu beweisen.

## 7. Bibliotheksbeispiel und methodischer Ablauf

Die elf Ausgangsaussagen sind Systemzweck, Titel, Exemplar, Mitglied, Ausleihe, aktive Ausleihe, berechtigtes Mitglied, verfügbares Exemplar, höchstens eine aktive Ausleihe, Ausleihen und Rückgabe. Die Systemübersicht wird zum System; die übrigen fünf Concepts, drei Fach-Rules und zwei UseCases bleiben. Hinzu kommen drei Components, sechs Modules, fünf Interfaces und eine Architekturregel. Identität und Referenzen der bisherigen Concept-Systemübersicht werden ausdrücklich aktualisiert.

Ein Wurzelmandat ist ausreichend. Frontendcode, Backendcode, Datenbankkonfiguration, Prüfungen und Bedienungsdokumentation sind fünf getrennte Realisierungsbeiträge. Die beiden Fach-/Architektur-Checks sind klar markierte, nicht ausgeführte Platzhalter.

Die Wiederholungsrückgabe ohne Änderung des ursprünglichen Rückgabezeitpunkts und Berechtigung über aktiven Mitgliedsstatus sind vorgeschlagene Fachentscheidungen. Fristberechnung und Behandlung unbekannter Kennungen bleiben offen. Das Beispiel ist kein Vollständigkeitsnachweis und keine fertige Anwendung.

### 7.1 Modellreview

Vor Realisierung geht der Review jede betroffene Aussage systematisch durch: Zweck und Akteure, Begriffe, Voraussetzungen, Erfolg, Ablehnung, Zustandswirkungen, Wiederholung/Konkurrenz, Entscheidungsspielräume und benötigte Realisierungs-/Prüf-/Dokumentationsbeiträge. Er vergleicht Prosa mit expliziten Beziehungen und meldet fehlende oder fragliche Kanten.

Offene fachliche Entscheidungen gehen an den zuständigen menschlichen oder ausdrücklich autorisierten Entscheider. Eine zusätzliche Abhängigkeit wird in das Modell aufgenommen, bevor ein neuer gebundener Impact als Umsetzungsgrundlage dient.

### 7.2 Realisierungsreview

Ein unabhängiger Review untersucht die betroffenen Kandidatendateien gegen die geltenden Aussagen. Er prüft insbesondere, ob alle Einstiegspunkte Regeln erhalten, Fehler keine unerlaubten Teilwirkungen erzeugen, Konkurrenz/Wiederholung geeignet behandelt und behauptete Prüffälle tatsächlich untersucht werden. Er nennt Befunde, nicht untersuchte Aussagen und Grenzen.

Ein Pflichttext im Mandat beweist die Einhaltung des Reviews nicht. Ausführungsprotokoll und Prüfergebnis müssen nachvollziehbar sein. Automatische Befunde, menschliche Zustimmung und nachgewiesene Inhaltskonformität bleiben getrennt.

### 7.3 Änderungsfolgen und ihre Grenzen

Modelländerung → feste Basis/Kandidat → Strukturprüfung → Modellreview → aktualisiertes Modell/Impact → begrenzte Realisierung → unabhängiger Review/Checks → Integration/Verify → Guarded Apply.

Der Impact muss hinzugefügte, geänderte und entfernte Definitionen/Beziehungen berücksichtigen. Der konservative Ausgangspunkt untersucht Erreichbarkeit in beiden Richtungen auf der Vereinigung der deklarierten Kanten aus alter und neuer Sicht. Das umfasst mindestens die in jedem Einzelstand betroffenen Erreichbarkeiten und kann zusätzliche Pfade über kombinierte alte/neue Kanten enthalten. Veränderte Owners, Pfade und Check-Bindungen bleiben sichtbar. Unbekannter Umfang führt zu Erweiterung oder ausdrücklichem Unknown, nicht zu unbegründetem Weglassen.

Alle neuen Beziehungen brauchen definierte Wirkung: Vertragsänderung erreicht Provider/Consumer; geänderte Fachzuordnung zugeordnete Bausteine/Artefakte; geänderte Zusammensetzung ihre Integration; Geltungsänderung alte/neue Ziele. Das sind Entwurfsanforderungen, keine implementierten Routingregeln. Der bisher geprüfte Routingstand bildet eine konservative Hülle über `uses`/`requires` in beide Richtungen. Im eng verbundenen Bibliotheksmodell kann diese nahezu alle fachlichen Aussagen erreichen. Change/Context dient der Erklärung und Priorisierung; es verkürzt die relevante Menge nicht automatisch. Ein anderer engerer Algorithmus wäre eine eigene begründete Produktentscheidung.

Breite Artefaktpfade wie `src/Library/` liefern einen Untersuchungsbereich. Sie wählen keine einzelnen Funktionen aus und garantieren keinen kleinen Blast Radius. Welche Dateien verändert werden müssen, ist innerhalb dieses Bereichs zu untersuchen.

Der [Überfälligkeitsdurchlauf](library-change-walkthrough.md) zeigt, wie eine Prosaänderung zunächst eine fehlende Beziehung verbergen kann, wie Review sie ergänzt und welche Prüfungen danach erforderlich werden.

## 8. Umsetzungsplan für den Implementer

### 8.1 Begründung der Designentscheidungen

| Entscheidung | Begründung und Abwägung |
| --- | --- |
| Kleine Struktur, fachliche Prosa | Explizite Zuordnung und Methodik zuerst nutzen; zusätzlichen Pflegeaufwand für Formalisierung an konkretem Nutzen prüfen. |
| Nominale Fach- und Architektur-Kinds | Fachlichkeit, Grenzen, Bausteine und Verträge als Zieltypen unterscheiden; ohne allgemeinen Fachevaluator. |
| Keine Kategorie für die gewählten Rollen | Der Kind trägt die strukturelle Unterscheidung direkt; gemeinsame Beschreibungskonventionen werden geteilt. Weitere Kinds benötigen einen konkreten Ausdrucksbedarf. |
| Mandat von Agentenbindung unterscheiden | Ein stabiler Auftrag kann von unterschiedlichen Agenten umgesetzt werden; Datei, Modell-Kind und Runtime-Identität sind eigene Benennungsentscheidungen. |
| Explizite Beziehungen und konservativer Impact | Koordination nachvollziehbar aus deklarierten Daten ableiten; Prosalinks allein liefern keine geprüften Kanten. |
| Zielkind aus dem Feld, wo eindeutig | Bei speziellen Feldern Wiederholung vermeiden; gemischte Zielmengen benötigen einen Kind-Discriminator. Auflösung und Prüfung behalten die vollständige Zielidentität. |
| Einheitliches contains mit ausdrücklichem Kind | Zusammensetzung durch einen Beziehungsnamen beschreiben; sichtbare Zieltypen erhalten eine wiederverwendbare Form. Erlaubte Kinder und Parentregeln bleiben im jeweiligen Schema definiert. |
| Realizes schließt contains-Unterstruktur ein | Umfang aus der einmal gepflegten Zusammensetzung ableiten; Unterdefinitionen nicht erneut aufzählen. Ein kleinerer Zielknoten begrenzt den Auftrag. Kein scope-Feld; keine automatische Erweiterung über Querbeziehungen. |
| Erneuerbarer Dateibestand | Pfadregeln pflegen und konkrete Dateien erfassen; eine zweite manuelle Zuordnungsliste vermeiden. |
| Unabhängige Reviews und begrenzte Checks | Fachbedeutung und tatsächliche Realisierung prüfen; Struktur und Prüferfolg nicht als universelle Korrektheit ausgeben. |

Die erhoffte geringere Aufsicht, zuverlässigere Agentenkoordination und niedrigere Kosten sind weiterhin Hypothesen. Dieser Dokumentationsauftrag misst sie nicht.

### 8.2 Umsetzungsvorbereitung für den Implementer

Lesereihenfolge: dieser Proposal → [aktuelles YAML-Beispiel](library-model-example.md) → [Änderungsdurchlauf](library-change-walkthrough.md) → [historischer Main-Abgleich](canonical-model-main-comparison.md) als Ausgangspunkt für einen neuen Vergleich.

Der Dokumentationsbranch stammt aus einer älteren Main-Basis. Den Implementierungsbranch vom beim Start frisch abgerufenen `origin/main` anlegen; Pfad, Branch, vollständigen SHA und lokale Änderungen erfassen. Aktuelle Beitragshinweise und Modellzonen-Owner prüfen. Die hier dokumentierten Entwürfe gezielt übernehmen, nicht den alten ausgecheckten Produktcode als aktuellen Main verwenden.

| Paket | Ergebnis | Voraussichtlicher Bereich | Abnahmekriterium |
| --- | --- | --- | --- |
| P0 — Ausgangspunkt und Verträge | Main-Delta, Versionierung, Fach-/Architekturschemas, contains-/Provider-/Parentregeln, Referenzziele und bedingte Kind-Angabe fixieren. | Produkt-/Modellvertrag; Core-Ausdrucksbedarf prüfen. | Keine stille alpha-Umdeutung; contains immer mit geprüftem Kind, sonst feste Ziel-Kinds aus Schema oder gemischte Zielmengen explizit aufgelöst. |
| P1 — Vollständiger kleiner Durchstich | Bibliotheksmodell mit Mandat, Fach-/Architekturdefinitionen, Artefakten und Check-Anschlüssen laden/validieren. | Schema, Loader und Beispiel; Core nur bei belegter Lücke. | Gültige Ziele/Parents/Provider auflösbar; falscher Kind, Namespace, Zyklus oder mehrdeutiger Provider mit Diagnose. |
| P2 — Sichten und Bedeutungserhalt | Alle Fach-/Architektur-Kinds und neuen Kanten in Report, Kontext, Graph und Digest führen; realizes-Umfang über contains ableiten, AppliesTo bringt Regeln an Ziele. | Projectmodel-Analyse/Typen und vorhandene Ausgabeanschlüsse. | Direkte und abgeleitete realizes-Ziele sind erklärbar; Rollen sehen benötigte Regeln, private Inhalte werden nicht über eine zweite Sichtbarkeitslogik geladen. |
| P3 — Änderungsfolgen | Fach-, Vertrags-, Provider-, Zuordnungs- und Zusammensetzungsänderung unter Alt/Neu in Impact/Explain behandeln. | Projectmodel-Impact/Explain und feste Host-Bindungen. | Alt/Neu-Wirkung erhalten; unbekannter Umfang bleibt konservativ; kombinierte Änderungen verlieren keine Pflicht. |
| P4 — Realisierung und Review | Fach-/Architekturzuordnung, Coverage, Check-Bindungen und dokumentierten Reviewablauf durch vorhandene Ausführung führen. | Artefakte, Projectcoverage/Projectwork und bei Vertragsänderung Projectrun. | Keine neue Regel verschwindet aus Realisierung/Prüfung; Reviewergebnisse nennen Befunde und Grenzen. |
| P5 — Kontrast und Integration | Kleines CLI-Gegenbeispiel, aktuelle Nutzer-/Formatdokumentation und normale Kandidatengates. | Beispiele, Tests und zuständige Dokumente. | Kein Bibliotheks-, Datenbank- oder Zwang zu mehreren Mandaten; Checks gelten für festen Kandidaten. |

P1 bis P4 bilden gemeinsam einen vollständigen Durchstich. Ein nur zugelassener neuer Kind ist kein fertiges Feature. Die Pakete sind Vorschläge, keine vergebenen Backlog-IDs und keine gestarteten Implementierungsaufträge.

Der breitere Facetten-/Ausdrucksentwurf folgt nur bei konkret begründetem Bedarf. Government, Cockpit, Providerstudien, universelle Sourceanalyse und Releasepublikation sind nicht Teil dieses ersten Durchstichs.

### 8.3 Prüfplan für die spätere Implementierung

Heute sind diese Prüfungen ausschließlich geplant. Bei Umsetzung gelten zusätzlich die dann aktuellen Projektgates.

| Risiko | Gezielter Nachweis |
| --- | --- |
| Nominaler Kind ist nur ein Label | Verweis auf Concept oder UseCase an einem Rule-only-Ziel wird abgewiesen; gültiger Rule-Verweis wird aufgelöst. |
| Kurzreferenz verliert Typ oder Identität | Rule-only-Referenz ohne kind löst gegen Rule auf, nicht gegen gleichnamigen Concept; fehlender/unerlaubter kind bei gemischten Zielmengen ist ein Befund. Normalisierte Zielidentität bleibt in Digest, Kontext und Graph erhalten. |
| contains wird zum untypisierten Sammelfeld | Fehlender kind und nach Quell-Kind unzulässiger Zielkind erzeugen Befunde, auch bei nur einem erlaubten Zieltyp. Gültige direkte Kinder erzeugen Zusammensetzungskanten und abgeleitete Parents. |
| Realisierungsumfang driftet oder wächst über Querbeziehungen | Referenz auf Backend umfasst direkte und tiefere contains-Kinder; referenziertes Mitglied über uses wird nur Kontext, nicht zusätzliches Realisierungsziel. Neuer/entfernter/verschobener contains-Nachfahre verändert den abgeleiteten Umfang an Alt/Neu-Ständen nachvollziehbar; mehrfach erreichte Ziele bleiben eindeutig. |
| Gemischte Referenz verliert Bedeutung | Alle sieben Fach-/Architekturtypen bleiben in Kontext, Realisierung, Check und Entscheidung eindeutig und digest-relevant. |
| Struktur oder Anbieter ist mehrdeutig | Doppelte Parentzuordnung, Zusammensetzungszyklus sowie fehlender/mehrdeutiger Provider eines konsumierten Interface erzeugen Befunde. |
| Architekturzuordnung verschwindet | Geänderte Fachzuordnung, Interfacevertrag und AppliesTo-Ziel erreichen die relevanten alten/neuen Bausteine, Artefakte und Prüfungen; Prosa allein gilt nicht als vollständiger Codebeleg. |
| Neue Regel fehlt im Impact | Geänderte Rule und hinzugefügte/entfernte Kante erreichen erwartete Pflichten, Owners, Artefakte und Checks unter Alt/Neu-Betrachtung. |
| Scope wird still verkleinert | Kombinierte Änderungen erhalten Pflichten; Unknown und Context werden nicht unbegründet aus Prüfung gestrichen. |
| Pfadregeln sind irreführend | Neue Datei im Präfix wird klassifiziert; unzugeordnete Datei und widersprüchliche Owner bleiben Befunde. |
| Prüfauswahl wird als Beweis ausgegeben | Auswahl eines Checks, seine tatsächliche Ausführung und Eignung für den neuen Fall bleiben unterscheidbar. |
| Neue Version bricht alte Bedeutung | Explizite Versions-/Migrationsentscheidung; unveränderte alte Vertragsbedeutung, keine stillen nominalen Umdeutungen. |

Zur Übergabe gehören feste SHAs, lokale Änderungen, tatsächlich ausgeführte Ergebnisse, nicht ausgeführte Prüfungen und offene Entscheidungen. Eine spätere Repo-Extraktion mit begrenzten Turns wäre separat zu beauftragen und müsste Sprache, Extraktionsfähigkeit, Prüfumfang und Nutzerwert getrennt beurteilen.

## 9. Zusagen und offene Entscheidungen

Die strukturelle Zusage lautet: erkläre Definitionen und Beziehungen im gewählten Umfang eindeutig und leite nachvollziehbar Zuständigkeit, Kontext, Dateizuordnung und Prüfauswahl daraus ab. Sie garantiert weder vollständige Fachbeschreibung noch beliebige Anwendungskorrektheit oder fehlerfreie Agenten.

Offen bleiben besonders:
- Exakte API-Version und Migrationsweg.
- Referenzunion versus getrennte Properties sowie zulässige Zielmengen.
- Regeln für externen Scope, nicht modellierte Interna und mögliche Mehrfachprovider.
- Weitere Quell-Kinds für `contains`, zusätzliche erlaubte Kinder und deren Parentregeln; die gemeinsame Beziehung und die ausdrückliche Kind-Angabe sind bereits gewählt.
- Beitragsteilung bei gemeinsamer UseCase-Realisierung und Rolle von Architekturchecks.
- Bedarf weiterer Beschreibungstypen, ohne universelle Sammeltypen zu schaffen.
- Regelbedingungen versus Invarianten: zunächst Prosa oder eigener nominaler/Feldvertrag?
- Verbindliche Protokollierung der Reviewpflicht ohne Verwechslung mit Autorisierung.
- Angemessene Zuordnungsgranularität und Breite des Impact.
- Kriterien, die spätere Fachformalismen oder Strictness-Profile rechtfertigen.
- Konkrete noch offene Fachentscheidungen des Bibliotheksbeispiels.
- Felder und Beziehungen aller Typen im schrittweisen Bibliotheksreview; insbesondere Mandate.parent/owns/instructions und der Mandatsbezug von Decision.actor.

## 10. Herkunft und Status

- Produktgespräch: `01a121f1-b948-7050-ae5d-9921b99db9c0`; Konsolidierung ausdrücklich am 12. Oktober 2026 beauftragt.
- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang der vollständigen Konsolidierung: `07e03215c968302361f5acedc8e69bd573ac95be`; beim Beginn sauber.
- Eingang der Kind-Aktualisierung: `5d18e8f4438cc1d24a10882f04b10501d65db20e`; beim Beginn sauber.
- Eingang der Architekturergänzung vom 13. Oktober: `ec7ba72d25c7f17cedcfb61e6b77f11d5cddb5ed`; beim Beginn sauber.
- Eingang der Referenzvereinfachung und Entfernung von ownership vom 13. Oktober: `a188878bd5c8283acf02fd6f754cf404b1d3503e`; beim Beginn sauber.
- Eingang der contains-Entscheidung vom 13. Oktober: `def47b05ed553918ed80d7b74728d2a77a8c7572`; beim Beginn sauber.
- Eingang der realizes-Umfangsentscheidung vom 13. Oktober: `25828d773cf2d7ea0bfc7c9bbb69c04d8e09c7fe`; beim Beginn sauber.
- Eingang der Mandate-Umbenennung und des vereinbarten Typreviews vom 13. Oktober: `d50fa8c39bd11eadbd24c4c55357cb890f08a871`; beim Beginn sauber.
- Ursprüngliche Branchbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Letzter gesonderter Main-Abgleich: `32afd1a57e715130d056e98e433721c6e7a10278` vom 11. Oktober; heute nicht als aktuell neu verifiziert.
- Zusätzlicher reiner Quellenrefresh im Gespräch am 13. Oktober: `c19d9d1259fcd493e2cc7ad93d10226778b2b7f4`. Das Projektvokabular besitzt dort weiterhin die fünf bisherigen Kinds und einzelne Reference-Zielkinds; die neuen Fach-/Architektur-Kinds und Referenzmengen dieses Entwurfs sind nicht implementiert. Der historische Vergleichsbericht bleibt unverändert datiert.
- Lokale Änderungen dieses Auftrags: ausschließlich Konzept-, Beispiel-, Ablauf- und Navigationsdokumentation. Keine Produktimplementierung, Modellkompilierung, Tests, Studien oder Agentenausführung am Bibliotheksprojekt.

Die frühere Facettenempfehlung wird durch diesen zusammenhängenden Arbeitsentwurf abgelöst. Datierte Vorarbeiten bleiben mit Statushinweis erhalten. Kanonische Produktdokumente, Konzeptregister und Backlog werden durch diesen Entwurf nicht automatisch geändert.

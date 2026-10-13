# Bibliothek: Architektur als System, Components, Modules und Interfaces

Datum: 13. Oktober 2026. Status: auf Nutzerwunsch ergänzter Architekturentwurf zum [Konzept](canonical-model-proposal.md). Die vier Kinds und ihre Beziehungsrollen werden festgehalten; konkrete Feld-, Referenz-, Provider- und Versionsregeln sind Vertragskandidaten für die Implementierung. Alle YAML-Blöcke sind Dokumentation mit `vNext` als Platzhalter, keine kompilierte oder ausgeführte Projektfixture.

Das [Bibliotheksmodell](library-model-example.md) zeigt den Project-Kandidaten, den Modellordner, die Fachdefinitionen, das Mandat und die Artefakte/Checks. Die Quellen werden im Modellordner gefunden; eine Manifestliste wird nicht gepflegt. Dieses Dokument ist die einzige aktuelle YAML-Darstellung der hier aufgeführten Architekturdefinitionen; die Modelldateien sind als spätere Eingaben benannt, heute nicht angelegt.

YAML-Schreibweise aktualisiert am 14. Oktober 2026: Jede Definition besitzt einen eigenständigen Körper, hier mit dem Syntaxkandidaten definition. Metadata enthält Name und Namespace; purpose entfällt als separates Feld. Die Architekturaufteilung und ihre noch zu prüfenden Beziehungsfelder bleiben dabei unverändert. Explizite Körperlinks werden zuerst an Loan, Member und Exemplar im Fachmodell durchgespielt.

## Unterscheidung und Struktur

| Kind | Bedeutung | Beispiel |
| --- | --- | --- |
| System | Zweck und Grenze einer zusammengehörigen Lösung. | Bibliotheksanwendung. |
| Component | Laufzeit-/Bereitstellungsbaustein mit eigener technischer Rolle. | Frontend, Backend, Datenbankdienst. |
| Module | Kohäsive logische Gliederung, ohne notwendige eigene Laufzeit. | Bestand, Mitgliedschaft, Ausleihe, Adapter. |
| Interface | Benannter Vertrag der Zusammenarbeit, getrennt von seinem Anbieter. | Bibliotheks-API und Speicherverträge. |

```text
System library-system
├── Component frontend
│   └── Module lending-ui
├── Component backend
│   ├── Module inventory
│   ├── Module membership
│   ├── Module circulation
│   ├── Module http-adapter
│   └── Module persistence-adapter
└── Component database
```

`contains` erklärt direkte strukturelle Bestandteile mit ausdrücklich genanntem Zielkind. Im ersten Beispielvertrag enthält ein System Components; ein Component oder Module kann Modules enthalten. Ein Component besitzt genau ein System als Parent; ein Module genau einen Component oder ein übergeordnetes Module. Parents werden aus den contains-Kanten abgeleitet. Die Zusammensetzung ist zyklenfrei. Der Datenbankdienst hat keine hier modellierten Interna; seine bereitgestellte Implementierung und die projektverantwortete Konfiguration sind ausdrücklich getrennt.

Das ownership-Feld entfällt vorerst. Der Datenbankserver kann Fremdsoftware sein; Konfiguration, Schema und Migrationen bleiben erklärte Projektbeiträge. Feinere Zuständigkeiten oder ein strukturiertes Feld zur Unterscheidung eigener und externer Systeme sind noch nicht festgelegt.

`contains` ersetzt die bisherigen Felder `components`/`modules` und nennt immer den Zielkind. Das Schema des jeweiligen Quell-Kinds prüft, welche Kinder zulässig sind. Die gemeinsame Form ist für weitere Modelltypen wiederverwendbar, ohne deren Kinder bereits freizugeben. Bei anderen festen Zieltypen entfällt `kind`: `dependsOn` verweist auf Module, `provides`/`consumes` auf Interface, `useCases` auf UseCase und `requires` auf Rule. Bei `realizes`, `uses` und `appliesTo` bleiben mehrere Ziel-Kinds zulässig und `kind` erforderlich. Die Definitionsdateien behalten ihren eigenen Kind.

Parent, Namespace und Mandat sind verschiedene Beziehungen. Der Namespace bleibt hier für alle Definitionsdateien `library`; Zusammensetzung wird ausdrücklich deklariert. Die Modellablage muss weder Runtime- noch Codegliederung nachahmen. Die eindeutige Parentbeziehung begründet keine Agentenbefugnis.

## Verträge und Zusammenarbeit

| Interface | Anbieter im Beispiel | Verbraucher |
| --- | --- | --- |
| library-api | Module http-adapter | Module lending-ui |
| inventory-access | Module inventory | Module circulation |
| member-access | Module membership | Module circulation |
| loan-storage | Module persistence-adapter | Module circulation |
| database-access | Component database | Module persistence-adapter |

`provides` und `consumes` werden am konkreten Anbieter/Verbraucher gepflegt. Die öffentliche Frontend-/Backendansicht lässt sich aus der Modulzugehörigkeit ableiten; sie benötigt keine zweite Kopie derselben Anbieterbeziehung auf dem Component.

Der erste Beispielvertrag bindet jedes konsumierte Interface an genau einen deklarierten Anbieter innerhalb des erklärten Umfangs. Fehlender oder mehrdeutiger Anbieter ist ein Befund; Provider-Auswahl/mehrere Varianten wäre ausdrücklich zu entwerfen. Für ein externes System kann dessen veröffentlichter Vertrag deklariert werden, ohne interne Module zu erfinden.

`dependsOn` beschreibt eine ausdrücklich festgelegte Implementierungs-/Buildabhängigkeit zwischen Modules, nicht nochmals die über ein Interface erklärte Zusammenarbeit. Beispielsweise verweisen die Adapter nach innen auf circulation. Der Verbraucher eines Interface importiert dadurch nicht automatisch die konkrete Anbieterimplementierung. Die tatsächlichen Codeabhängigkeiten bleiben Gegenstand des Reviews beziehungsweise geeigneter Werkzeuge.

## Verbindung zur Fachlichkeit und Evidenz

`Module.realizes` ist eine Zuordnung fachlicher Beiträge zur Architektur: Das Modul soll an der Realisierung der genannten Concepts, Rules und UseCases mitwirken. Es ist kein Befund über vorhandenen Code und begründet keine Schreibzuständigkeit. Prosa erklärt, welchen Beitrag etwa Bedienung, Orchestrierung oder Speicherung übernehmen. Mehrere Module können gemeinsam einen UseCase tragen.

`Artifact.realizes` verbindet erwartete Repositorybeiträge mit Fach- oder Architekturdefinitionen. Erst Inventar, Kandidatenprüfung und Checks liefern Evidenz über tatsächliche Dateien. Das Wurzelmandat beschreibt die eindeutige Schreibzuständigkeit; die Architektur erzeugt keine zusätzliche automatische Verantwortungsstruktur. Der ausführende Agent wird separat an das Mandat gebunden.

Ein realizes-Ziel schließt automatisch seine gesamte transitive contains-Unterstruktur ein; ein scope-Feld ist nicht vorgesehen. Das Backendartefakt muss deshalb die fünf enthaltenen Modules nicht zusätzlich aufzählen. Der Beitrag bleibt durch seine Artefaktrolle bestimmt und ist kein Nachweis. Andere Beziehungen werden nicht als Zusammensetzung verfolgt. Im aktuellen Beispiel sind Fachdefinitionen und Interfaces nicht in contains einsortiert; ihre eigenständigen Zuordnungen bleiben ausdrücklich erhalten. Ob Modules künftig auch Concepts, Rules oder UseCases enthalten sollen und wie realizes insgesamt verstanden wird, ist Gegenstand der weiteren schrittweisen Diskussion.

`Rule.appliesTo` erklärt den Geltungsbereich einer Regel und muss sie in den Kontext der ausdrücklich genannten Ziele bringen. Für dieses Beispiel gilt die Relation direkt, ohne automatische Subtree- oder beliebige Graphausbreitung. Falls weitere Ziele nötig sind, werden sie deklariert. Die neue Architekturregel ist zunächst Prosa und kein vom Compiler ausgeführter Importtest.

Die deklarierte `dependsOn`-Liste ist keine ausführbare Verbotsmatrix. Eine automatische Durchsetzung verbotener Abhängigkeiten benötigt später einen definierten Regelmechanismus oder einen gebundenen Check; hier beurteilt der methodische Review die Einhaltung.

## YAML-Definitionen

Alle Definitionen liegen direkt unter `model/library/`, passend zum Namespace. Die Systemübersicht wechselt in der neuen Entwurfsfassung von Concept zu System; ihre vollständige Identität und Referenzen ändern sich entsprechend. Das ist keine stille Migration eines bestehenden Versionsvertrags.

### `model/library/library-system.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: System
metadata:
  namespace: library
  name: library-system
definition: |
  Die Anwendung ermöglicht Ausleihe und Rückgabe vorhandener Exemplare.
  Sie umfasst Webfrontend, Backend und einen bereitgestellten Datenbankdienst.
  Beschaffung, Gebühren und Mahnungen liegen außerhalb des Ausschnitts.
  Frameworks, Datenbankprodukt und Deploymentumgebung sind vor Umsetzung
  festzulegen oder ausdrücklich zu delegieren.
spec:
  public: true
  contains:
    - kind: Component
      namespace: library
      name: frontend
    - kind: Component
      namespace: library
      name: backend
    - kind: Component
      namespace: library
      name: database
```

### `model/library/frontend.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Component
metadata:
  namespace: library
  name: frontend
definition: |
  Der Webclient stellt die Bedienung von Ausleihe und Rückgabe bereit.
  Er verwendet den Bibliotheks-API-Vertrag über sein Bedienmodul.
  API-Adresse, Build und Auslieferung werden als technische Bedingungen festgelegt.
spec:
  public: true
  contains:
    - kind: Module
      namespace: library
      name: lending-ui
```

### `model/library/backend.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Component
metadata:
  namespace: library
  name: backend
definition: |
  Der Dienst stellt die Bibliotheks-API bereit und erhält die Fachregeln.
  Er verbindet Fachmodule und technische Adapter bei seiner Zusammenstellung.
  Er bringt einen erklärten Start-, Konfigurations- und Betriebsvertrag mit.
spec:
  public: true
  contains:
    - kind: Module
      namespace: library
      name: inventory
    - kind: Module
      namespace: library
      name: membership
    - kind: Module
      namespace: library
      name: circulation
    - kind: Module
      namespace: library
      name: http-adapter
    - kind: Module
      namespace: library
      name: persistence-adapter
```

### `model/library/database.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Component
metadata:
  namespace: library
  name: database
definition: |
  Der Datenbankdienst speichert Bibliotheksdaten dauerhaft.
  Seine Serverimplementierung wird nicht in diesem Projekt entwickelt;
  das Projekt verantwortet Konfiguration, Schema und nötige Migrationen.
  Das konkrete Produkt und dessen Betriebsvertrag bleiben festzulegen.
spec:
  public: true
  provides:
    - namespace: library
      name: database-access
```

### `model/library/lending-ui.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: lending-ui
definition: |
  Dieses Modul trägt zur Nutzung von Ausleihe und Rückgabe bei.
  Es erfasst Eingaben und stellt Ergebnisse sowie Ablehnungen dar.
  Die serverseitige Erhaltung der Fachregeln liegt beim Backend.
spec:
  public: true
  consumes:
    - namespace: library
      name: library-api
  realizes:
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
```

### `model/library/inventory.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: inventory
definition: |
  Dieses Modul beschreibt und verwaltet Titel und physische Exemplare.
  Sein Bestandsvertrag liefert benötigte Informationen;
  Ausleihverfügbarkeit ergibt sich zusätzlich aus aktiven Ausleihen.
spec:
  public: true
  provides:
    - namespace: library
      name: inventory-access
  realizes:
    - kind: Concept
      namespace: library
      name: title
    - kind: Concept
      namespace: library
      name: copy
```

### `model/library/membership.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: membership
definition: |
  Dieses Modul verwaltet Mitgliedsidentität und Berechtigungsstatus.
  Die ursprüngliche Statusregel wird hier fachlich zugeordnet.
  Eine spätere Überfälligkeitsbedingung benötigt zusätzlich Ausleihkontext
  und ist erneut architektonisch zuzuordnen.
spec:
  public: true
  provides:
    - namespace: library
      name: member-access
  realizes:
    - kind: Concept
      namespace: library
      name: member
    - kind: Rule
      namespace: library
      name: eligible-member
```

### `model/library/circulation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: circulation
definition: |
  Dieses Modul koordiniert Ausleihe und Rückgabe und erhält ihre Regeln.
  Bestands-, Mitglieds- und Speicherinformationen kommen über benannte Verträge.
  Die konkrete Persistenztechnik bleibt außerhalb der Fachlogik.
spec:
  public: true
  consumes:
    - namespace: library
      name: inventory-access
    - namespace: library
      name: member-access
    - namespace: library
      name: loan-storage
  realizes:
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
    - kind: Rule
      namespace: library
      name: available-copy
    - kind: Rule
      namespace: library
      name: one-active-loan
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
```

### `model/library/http-adapter.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: http-adapter
definition: |
  Dieses Modul übersetzt API-Eingaben, Ergebnisse und Fehler in die vereinbarten
  Ausleihabläufe. Es ruft die Anwendungsschnittstellen des Ausleihmoduls auf.
  Authentisierung, Fehlerabbildung und Versionsvertrag sind vor Umsetzung zu klären.
spec:
  public: true
  provides:
    - namespace: library
      name: library-api
  dependsOn:
    - namespace: library
      name: circulation
  realizes:
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
```

### `model/library/persistence-adapter.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Module
metadata:
  namespace: library
  name: persistence-adapter
definition: |
  Dieses Modul stellt den fachlichen Ausleihspeicher auf dem Datenbankdienst bereit.
  Es hängt für Fachwerte und Speichervertrag nach innen vom Ausleihmodul ab.
  Es übernimmt keine eigenständige Neudefinition der Fachregeln.
spec:
  public: true
  provides:
    - namespace: library
      name: loan-storage
  consumes:
    - namespace: library
      name: database-access
  dependsOn:
    - namespace: library
      name: circulation
  realizes:
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
    - kind: Rule
      namespace: library
      name: one-active-loan
```

### `model/library/library-api.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Interface
metadata:
  namespace: library
  name: library-api
definition: |
  Der API-Vertrag bietet Ausleihe und Rückgabe mit den jeweils geltenden
  Voraussetzungen, Ergebnissen und Fehlern an. Er verwendet benannte Identitäten.
  Transport, Versionierung und konkrete Datenformen sind verbindlich zu präzisieren.
  Der Adapterentwurf geht von einem HTTP-Vertrag aus.
spec:
  public: true
  useCases:
    - namespace: library
      name: borrow
    - namespace: library
      name: return
  uses:
    - kind: Concept
      namespace: library
      name: member
    - kind: Concept
      namespace: library
      name: copy
    - kind: Concept
      namespace: library
      name: loan
  requires:
    - namespace: library
      name: one-active-loan
```

### `model/library/inventory-access.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Interface
metadata:
  namespace: library
  name: inventory-access
definition: |
  Dieser Vertrag liefert die Existenz und relevanten Informationen eines
  Exemplars anhand seiner Identität sowie dessen Titelbezug.
  Er entscheidet nicht allein über aktive Ausleihen.
spec:
  public: true
  uses:
    - kind: Concept
      namespace: library
      name: copy
    - kind: Concept
      namespace: library
      name: title
```

### `model/library/member-access.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Interface
metadata:
  namespace: library
  name: member-access
definition: |
  Dieser Vertrag liefert Existenz, Identität und Berechtigungsstatus eines Mitglieds.
  Er liefert damit Kontext zur Beurteilung der geltenden Berechtigungsregel.
spec:
  public: true
  uses:
    - kind: Concept
      namespace: library
      name: member
```

### `model/library/loan-storage.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Interface
metadata:
  namespace: library
  name: loan-storage
definition: |
  Dieser Vertrag speichert und liest Ausleihen und erfasst Rückgaben dauerhaft.
  Eine konkurrierende Vergabe darf keine zweite aktive Ausleihe erzeugen.
  Verfügbarkeit und Erzeugung müssen durch das gewählte Verfahren konsistent sein.
  Die konkrete Datenbanktechnik wird durch diesen Vertrag nicht festgelegt.
spec:
  public: true
  uses:
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
    - kind: Concept
      namespace: library
      name: copy
  requires:
    - namespace: library
      name: one-active-loan
```

### `model/library/database-access.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Interface
metadata:
  namespace: library
  name: database-access
definition: |
  Dieser Vertrag beschreibt die benötigte dauerhafte Datenhaltung und ihre
  Zugriffs-, Konsistenz- und Konfigurationsbedingungen. Der Persistenzadapter
  verwendet ihn. Produkt, konkrete Operationen und Datenschema werden vor Umsetzung
  festgelegt; dieser Text ist kein vollständiger Datenbankprotokollvertrag.
spec:
  public: true
  uses:
    - kind: Concept
      namespace: library
      name: loan
```

### `model/library/domain-independent-of-storage.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Rule
metadata:
  namespace: library
  name: domain-independent-of-storage
definition: |
  Die Fachmodule dürfen nicht direkt von Datenbankdienst oder konkretem
  Persistenzadapter abhängen. Nötige Speicherzugriffe erfolgen über den fachlichen
  loan-storage-Vertrag. Technische Adapter dürfen von Fachwerten nach innen abhängen.
  Die tatsächlichen Codeabhängigkeiten sind durch Review und geeignete Checks zu prüfen.
spec:
  public: true
  uses:
    - kind: Component
      namespace: library
      name: database
    - kind: Module
      namespace: library
      name: persistence-adapter
    - kind: Interface
      namespace: library
      name: loan-storage
  appliesTo:
    - kind: Module
      namespace: library
      name: inventory
    - kind: Module
      namespace: library
      name: membership
    - kind: Module
      namespace: library
      name: circulation
```

## Noch zu klären und gezielt zu prüfen

Die Frontend-/Backendaufteilung ist als konkrete Architektur dieses Gesprächsbeispiels gewählt. Frameworks, Datenbankprodukt, Deployment, Authentisierung, konkrete Interfaceoperationen und Fehlerformen bleiben festzulegen oder ausdrücklich zu delegieren. Der Aufbau benötigt zusätzlich passende Build-, Start- und Bereitstellungsbeiträge; nicht jede technische Rootdatei ist durch die gezeigten Präfixe schon erfasst.

Strukturprüfung untersucht Identitäten, endliche Zielmengen, eindeutige Parentbeziehungen, Zusammensetzungszyklen und Providerzuordnung. Fach- und Realisierungsreview beurteilen Verträge, Beitragsteilung, tatsächliche Abhängigkeiten und geeignete Evidenz. Eine im Modell deklarierte dependency beweist keine Codekonformität. Interfacezyklen sind nicht pauschal Zusammensetzungszyklen.

Alle neuen Beziehungen brauchen Behandlung in Digest, Kontext, Graph, Alt/Neu-Impact, Artefaktzuordnung und Prüfauswahl. Die vormals besprochene Überfälligkeitsänderung muss auch die Architekturzuordnung prüfen, weil ihre Bedingung Mitglieds- und Ausleihkontext verbindet.

## Herkunft

Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`; Erweiterung am 13. Oktober 2026 ausdrücklich beauftragt. Worktree `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`, Branch `docs/product-model-concepts`, Eingang `ec7ba72d25c7f17cedcfb61e6b77f11d5cddb5ed`, beim Beginn sauber. Keine Produktimplementierung, Modelldateierzeugung, Modellkompilierung, Tests oder Studien.

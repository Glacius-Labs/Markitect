# Bibliothek: vollständiger Fach- und Architekturentwurf

Aktualisiert am 13. Oktober 2026. Status: zusammenhängendes Gesprächsbeispiel zum [Konzept](canonical-model-proposal.md). Gewählt sind `mandate.yaml`, Concept, Rule, UseCase und die ergänzten Architektur-Kinds System, Component, Module und Interface. `vNext`, Referenz- und Feldregeln sind Vertragskandidaten, keine unterstützte Version oder kompilierte Fixture.

Die Fachdefinitionen, Mandat, Artefakte, Checks und das vollständige Manifest stehen hier. Die aktuellen Architekturdefinitionen werden ausschließlich im [Architektur-YAML](library-architecture-example.md) ausgeschrieben. Zusammen zeigen beide Dokumente 34 Modelldefinitionen: 1 Manager, 1 System, 3 Components, 6 Modules, 5 Interfaces, 5 Concepts, 4 Rules, 2 UseCases, 5 Artifacts und 2 Checks.

## Umfang und Fachentscheidungen

Ausleihe und Rückgabe bilden den fachlichen Ausschnitt; Beschaffung, Gebühren und Mahnungen liegen außerhalb. Berechtigung über aktiven Mitgliedsstatus und unveränderte Wiederholungsrückgabe sind Fachvorschläge. Fristberechnung, unbekannte Kennungen und Datenbereitstellung müssen vor Umsetzung geklärt werden.

Die Architektur ergänzt Webfrontend, Backend, einen Datenbankdienst und die benannten Verträge. Frameworks, konkrete Datenbank, Betriebsumgebung und Authentisierung bleiben festzulegen oder ausdrücklich zu delegieren. Build-, Start- und Bereitstellungsbeiträge folgen diesen Entscheidungen.

| Fachdefinition | Kind | Rolle |
| --- | --- | --- |
| `title` | `Concept` | Fach- oder Zustandsbegriff als Prosa. |
| `copy` | `Concept` | Fach- oder Zustandsbegriff als Prosa. |
| `member` | `Concept` | Fach- oder Zustandsbegriff als Prosa. |
| `loan` | `Concept` | Fach- oder Zustandsbegriff als Prosa. |
| `active-loan` | `Concept` | Fach- oder Zustandsbegriff als Prosa. |
| `eligible-member` | `Rule` | Normative Bedingung oder Invariante. |
| `available-copy` | `Rule` | Normative Bedingung oder Invariante. |
| `one-active-loan` | `Rule` | Normative Bedingung oder Invariante. |
| `borrow` | `UseCase` | Verhaltensvertrag als Prosa. |
| `return` | `UseCase` | Verhaltensvertrag als Prosa. |

Die bisher als Concept gezeigte Übersicht `library-system` ist jetzt ein System; ihre Identität und alle aktuellen Referenzen ändern sich entsprechend. Concept, Rule und UseCase behalten ihre Bedeutung. Architekturzugehörigkeit verleiht keine Agentenbefugnis; der Auftrag steht weiterhin in `mandate.yaml` mit Kind Manager.

## Dateisystem und explizite Eingaben

```text
.markitect/
├── project.yaml
└── model/
    ├── mandate.yaml
    └── library/
        ├── library-system.yaml
        ├── title.yaml
        ├── copy.yaml
        ├── member.yaml
        ├── loan.yaml
        ├── active-loan.yaml
        ├── eligible-member.yaml
        ├── available-copy.yaml
        ├── one-active-loan.yaml
        ├── borrow.yaml
        ├── return.yaml
        ├── frontend.yaml
        ├── backend.yaml
        ├── database.yaml
        ├── lending-ui.yaml
        ├── inventory.yaml
        ├── membership.yaml
        ├── circulation.yaml
        ├── http-adapter.yaml
        ├── persistence-adapter.yaml
        ├── library-api.yaml
        ├── inventory-access.yaml
        ├── member-access.yaml
        ├── loan-storage.yaml
        ├── database-access.yaml
        ├── domain-independent-of-storage.yaml
        ├── frontend-implementation.yaml
        ├── backend-implementation.yaml
        ├── database-configuration.yaml
        ├── library-verification.yaml
        ├── library-documentation.yaml
        ├── library-contract-check.yaml
        └── architecture-contract-check.yaml
```

Eine Definition pro Datei und Namespace aus dem Verzeichnis übernehmen die bisher untersuchte Ladeform. Alle Definitionen außer dem Root-Mandat liegen im Namespace library. Die Anwendung muss diese Ablage nicht spiegeln. Zusammensetzung ist über `contains` mit ausdrücklichem Zielkind erklärt, nicht durch diesen flachen Modellordner.

Die Manifestliste nennt Modellquellen, nicht jede Anwendungsdatei. Artefaktpfade erfassen konkrete Realisierungsbereiche. Weitere tatsächliche Dateien benötigen Zuordnung oder ausdrückliche Ausnahme. Die gezeigten Pfade sind kein Beweis vollständiger späterer Build-/CI-/Konfigurationsabdeckung.

### `project.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
name: library-architecture-example
coverageMode: full
documentPath: docs/markitect/project.md
modelFiles:
  - .markitect/model/mandate.yaml
  - .markitect/model/library/library-system.yaml
  - .markitect/model/library/title.yaml
  - .markitect/model/library/copy.yaml
  - .markitect/model/library/member.yaml
  - .markitect/model/library/loan.yaml
  - .markitect/model/library/active-loan.yaml
  - .markitect/model/library/eligible-member.yaml
  - .markitect/model/library/available-copy.yaml
  - .markitect/model/library/one-active-loan.yaml
  - .markitect/model/library/borrow.yaml
  - .markitect/model/library/return.yaml
  - .markitect/model/library/frontend.yaml
  - .markitect/model/library/backend.yaml
  - .markitect/model/library/database.yaml
  - .markitect/model/library/lending-ui.yaml
  - .markitect/model/library/inventory.yaml
  - .markitect/model/library/membership.yaml
  - .markitect/model/library/circulation.yaml
  - .markitect/model/library/http-adapter.yaml
  - .markitect/model/library/persistence-adapter.yaml
  - .markitect/model/library/library-api.yaml
  - .markitect/model/library/inventory-access.yaml
  - .markitect/model/library/member-access.yaml
  - .markitect/model/library/loan-storage.yaml
  - .markitect/model/library/database-access.yaml
  - .markitect/model/library/domain-independent-of-storage.yaml
  - .markitect/model/library/frontend-implementation.yaml
  - .markitect/model/library/backend-implementation.yaml
  - .markitect/model/library/database-configuration.yaml
  - .markitect/model/library/library-verification.yaml
  - .markitect/model/library/library-documentation.yaml
  - .markitect/model/library/library-contract-check.yaml
  - .markitect/model/library/architecture-contract-check.yaml
```

## Auftrag: `model/mandate.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Manager
metadata:
  namespace: ""
  name: library-project
purpose: Verantwortet den Bibliotheksausschnitt und seine Realisierung.
spec:
  owns:
    - .
  instructions: |
    Prüfe fachliche Vollständigkeit und explizite Beziehungen bei Modelländerungen.
    Prüfe Umsetzung und Nachweise unabhängig anhand der betroffenen Aussagen.
    Melde offene Frist-, Fehler- und Autorisierungsentscheidungen vor ihrer Umsetzung.
    Entscheide technische Details innerhalb der ausdrücklich beschriebenen Freiräume.
```

Die Anweisungen formulieren Pflichten und Entscheidungsspielräume, beweisen aber weder authentifizierte Autorisierung noch ihre Einhaltung. Runtime-/Agentenbindung bleibt zusätzlich zu konfigurieren.

## Fachliche Concepts, Rules und UseCases

`uses` liefert Kontext; `UseCase.requires` benennt normative Verpflichtungen. Referenzen nennen ihren Zielkind bei mehreren zulässigen Ziel-Kinds und immer bei der einheitlichen Zusammensetzungsbeziehung `contains`: `uses` braucht `kind`, `requires` und `checks` erhalten den festen Zieltyp aus dem Schema. Auch eine aktuell rein fachliche `uses`-Liste bleibt typgemischt zulässig und nennt daher den Kind. Die API-Version folgt dem gemeinsamen versionierten Vertrag. Laufzeitinstanzen und deren Kardinalitäten bleiben von Modellreferenzen unterscheidbar.

### `model/library/title.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: title
purpose: Beschreibt einen Buchtitel.
spec:
  public: true
  description: |
    Ein Titel beschreibt ein Werk beziehungsweise eine Ausgabe.
    Mehrere physische Exemplare können demselben Titel zugeordnet sein.
```

### `model/library/copy.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: copy
purpose: Beschreibt ein einzelnes physisches Exemplar.
spec:
  public: true
  description: |
    Ein Exemplar besitzt eine eindeutige Identität und gehört genau einem Titel.
    Die Modelldefinition beschreibt den Fachtyp; konkrete Exemplare sind Anwendungsdaten.
  uses:
    - kind: Concept
      namespace: library
      name: title
```

### `model/library/member.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: member
purpose: Beschreibt ein Mitglied der Bibliothek.
spec:
  public: true
  description: |
    Ein Mitglied besitzt eine eindeutige Identität und einen Status seiner Ausleihberechtigung.
    Für diesen Entwurf unterscheiden wir aktive und nicht aktive Ausleihberechtigung.
```

### `model/library/loan.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: loan
purpose: Beschreibt eine Ausleihe und ihre Daten.
spec:
  public: true
  description: |
    Eine Ausleihe verbindet genau ein Mitglied mit genau einem Exemplar.
    Sie besitzt eine Identität, einen Beginn, eine Fälligkeit und einen optionalen Rückgabezeitpunkt.
    Die konkrete Fristberechnung ist vor der Umsetzung festzulegen.
  uses:
    - kind: Concept
      namespace: library
      name: member
    - kind: Concept
      namespace: library
      name: copy
```

### `model/library/active-loan.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: active-loan
purpose: Definiert den Zustand einer aktiven Ausleihe.
spec:
  public: true
  description: |
    Eine gespeicherte Ausleihe ist aktiv, solange kein Rückgabezeitpunkt erfasst ist.
  uses:
    - kind: Concept
      namespace: library
      name: loan
```

### `model/library/eligible-member.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Rule
metadata:
  namespace: library
  name: eligible-member
purpose: Definiert die Berechtigungsbedingung für das Ausleihen.
spec:
  public: true
  description: |
    Ein Mitglied darf ausleihen, wenn es existiert und seine Ausleihberechtigung aktiv ist.
    Die Bedingung wird bei der Vergabe geprüft.
  uses:
    - kind: Concept
      namespace: library
      name: member
```

### `model/library/available-copy.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Rule
metadata:
  namespace: library
  name: available-copy
purpose: Definiert die Verfügbarkeitsbedingung für das Ausleihen.
spec:
  public: true
  description: |
    Ein Exemplar ist für die Vergabe verfügbar, wenn es existiert und keine aktive Ausleihe besitzt.
    Eine frühere Verfügbarkeitsabfrage allein garantiert keine erfolgreiche Vergabe.
  uses:
    - kind: Concept
      namespace: library
      name: copy
    - kind: Concept
      namespace: library
      name: active-loan
```

### `model/library/one-active-loan.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Rule
metadata:
  namespace: library
  name: one-active-loan
purpose: Definiert die Eindeutigkeitsinvariante der Vergabe.
spec:
  public: true
  description: |
    Für jedes Exemplar besteht höchstens eine aktive Ausleihe.
    Diese Regel gilt auch bei konkurrierenden Anfragen und muss nach erfolgreichen
    sowie abgelehnten Operationen erhalten sein.
  uses:
    - kind: Concept
      namespace: library
      name: copy
    - kind: Concept
      namespace: library
      name: active-loan
```

### `model/library/borrow.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: UseCase
metadata:
  namespace: library
  name: borrow
purpose: Beschreibt die Vergabe eines Exemplars an ein Mitglied.
spec:
  public: true
  description: |
    Ein berechtigtes Mitglied darf ein verfügbares Exemplar ausleihen.
    Erfolg erzeugt eine dauerhaft gespeicherte aktive Ausleihe.
    Ablehnung erzeugt keine neue Ausleihe.
    Bei konkurrierenden Anfragen muss one-active-loan erhalten bleiben.
    Die technische Umsetzung der atomaren Vergabe darf der verantwortliche Bereich wählen.
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
      name: eligible-member
    - namespace: library
      name: available-copy
    - namespace: library
      name: one-active-loan
```

### `model/library/return.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: UseCase
metadata:
  namespace: library
  name: return
purpose: Beschreibt die Beendigung einer Ausleihe durch Rückgabe.
spec:
  public: true
  description: |
    Die Rückgabe einer aktiven Ausleihe wird dauerhaft durch einen Rückgabezeitpunkt erfasst.
    Danach ist diese Ausleihe nicht mehr aktiv.
    Eine wiederholte Rückgabe verändert den bereits erfassten Rückgabezeitpunkt nicht.
    Die Behandlung einer unbekannten Ausleihkennung ist vor der Umsetzung festzulegen.
  uses:
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
  requires:
    - namespace: library
      name: one-active-loan
```

## System, Components, Modules und Interfaces

Die [Architekturdarstellung](library-architecture-example.md) definiert die hier im Manifest geladenen Ressourcen samt Provider-/Consumerverträgen, Zuordnung fachlicher Beiträge und Architekturregel. Jede aktuelle YAML-Definition wird dort genau einmal gezeigt.

Das Ausleihmodul konsumiert Bestands-, Mitglieds- und Speicherverträge. Der HTTP-Adapter stellt die Bibliotheks-API bereit; der Persistenzadapter den fachlichen Speichervertrag. Der Datenbankdienst stellt einen davon getrennten technischen Vertrag bereit. Fachmodule müssen die konkrete Datenbanktechnik nicht importieren.

## Repositorybeiträge und Nachweise

Module.realizes ist eine architektonische Zuordnung gewünschter Beiträge. Artifact.realizes erklärt erwartete Repositoryrealisierung; erst konkrete Dateien, Reviews und ausgeführte Checks liefern Evidenz. Bedienung, Fachlogik, Adapter und Konfiguration können gemeinsam einen UseCase oder eine Rule tragen. Die Beschreibungen unterscheiden diese Beiträge.

Realizes umfasst das direkte Ziel einschließlich aller über contains enthaltenen Definitionen, ohne scope-Feld. Frontend- und Backendartefakte nennen deshalb ihre Component statt deren Modules zusätzlich aufzuzählen. Dokumentations- und Verifikationsartefakte nennen für den technischen Gesamtbereich das System. Der abgeleitete Beitrag gilt entsprechend ihrer Rolle und beweist keine geprüfte Umsetzung. Fachdefinitionen und Interfaces liegen im aktuellen Beispiel außerhalb der contains-Struktur; deren ausdrücklich gewünschte Beiträge bleiben separat referenziert. Andere Beziehungen erweitern den enthaltenen Bereich nicht automatisch.

Die Artefaktrolle konkretisiert den Beitrag: `implementation` implementiert Verhalten, `configuration` konfiguriert dessen technische Umgebung, `verification` stellt Prüfungen bereit und `documentation` erklärt das System. Ein Verifikations- oder Dokumentationsartefakt behauptet damit keine produktive Implementierung des referenzierten UseCase. Alle gezeigten Zuordnungen beschreiben erwartete Beiträge, keine bereits erbrachten Nachweise.

### `model/library/frontend-implementation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: frontend-implementation
purpose: Realisiert Bedienung und API-Verwendung des Frontends.
spec:
  role: implementation
  realizes:
    - kind: Component
      namespace: library
      name: frontend
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
  paths:
    - frontend/
  checks:
    - namespace: library
      name: library-contract-check
    - namespace: library
      name: architecture-contract-check
  required: true
  reason: Realisiert Bedienung und API-Verwendung des Frontends.
```

### `model/library/backend-implementation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: backend-implementation
purpose: Realisiert Backend, fachliche Abläufe und technische Adapter.
spec:
  role: implementation
  realizes:
    - kind: Component
      namespace: library
      name: backend
    - kind: Interface
      namespace: library
      name: library-api
    - kind: Interface
      namespace: library
      name: inventory-access
    - kind: Interface
      namespace: library
      name: member-access
    - kind: Interface
      namespace: library
      name: loan-storage
    - kind: Concept
      namespace: library
      name: title
    - kind: Concept
      namespace: library
      name: copy
    - kind: Concept
      namespace: library
      name: member
    - kind: Concept
      namespace: library
      name: loan
    - kind: Concept
      namespace: library
      name: active-loan
    - kind: Rule
      namespace: library
      name: eligible-member
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
    - kind: Rule
      namespace: library
      name: domain-independent-of-storage
  paths:
    - backend/
  checks:
    - namespace: library
      name: library-contract-check
    - namespace: library
      name: architecture-contract-check
  required: true
  reason: Realisiert Backend, fachliche Abläufe und technische Adapter.
```

### `model/library/database-configuration.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: database-configuration
purpose: Stellt Datenbankkonfiguration, Schema und nötige Migrationen bereit.
spec:
  role: configuration
  realizes:
    - kind: Component
      namespace: library
      name: database
    - kind: Interface
      namespace: library
      name: database-access
    - kind: Rule
      namespace: library
      name: one-active-loan
  paths:
    - deployment/database/
  checks:
    - namespace: library
      name: library-contract-check
    - namespace: library
      name: architecture-contract-check
  required: true
  reason: Stellt Datenbankkonfiguration, Schema und nötige Migrationen bereit.
```

### `model/library/library-verification.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: library-verification
purpose: Liefert Prüfbeiträge zu Fachverträgen, Interfaces und Architektur.
spec:
  role: verification
  realizes:
    - kind: Rule
      namespace: library
      name: eligible-member
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
    - kind: System
      namespace: library
      name: library-system
    - kind: Interface
      namespace: library
      name: library-api
    - kind: Interface
      namespace: library
      name: inventory-access
    - kind: Interface
      namespace: library
      name: member-access
    - kind: Interface
      namespace: library
      name: loan-storage
    - kind: Interface
      namespace: library
      name: database-access
    - kind: Rule
      namespace: library
      name: domain-independent-of-storage
  paths:
    - tests/
  checks:
    - namespace: library
      name: library-contract-check
    - namespace: library
      name: architecture-contract-check
  required: true
  reason: Liefert Prüfbeiträge zu Fachverträgen, Interfaces und Architektur.
```

### `model/library/library-documentation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: library-documentation
purpose: Erklärt System, Architektur, Bedienung und erklärten Prüfumfang.
spec:
  role: documentation
  realizes:
    - kind: System
      namespace: library
      name: library-system
    - kind: Interface
      namespace: library
      name: library-api
    - kind: Interface
      namespace: library
      name: inventory-access
    - kind: Interface
      namespace: library
      name: member-access
    - kind: Interface
      namespace: library
      name: loan-storage
    - kind: Interface
      namespace: library
      name: database-access
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
  paths:
    - docs/library/
    - docs/markitect/project.md
    - README.md
  required: true
  reason: Erklärt System, Architektur, Bedienung und erklärten Prüfumfang.
```

Die Bereiche frontend/, backend/, deployment/database/, tests/ und docs/library/ sind bewusst deklarierte Zuordnung. Sie garantieren keinen engen Blast Radius und wählen keine einzelnen Funktionen aus. Der Root-Manager bleibt der einzige Schreibowner dieses kleinen Beispiels. Keine dieser tatsächlichen Dateien wurde erzeugt.

## Prüfanschlüsse

### `model/library/library-contract-check.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Check
metadata:
  namespace: library
  name: library-contract-check
purpose: Prüft die Ausleih- und Rückgabeverträge anhand konkreter Fälle.
spec:
  command:
    - "<library-contract-check>"
  uses:
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
    - kind: Rule
      namespace: library
      name: eligible-member
    - kind: Rule
      namespace: library
      name: available-copy
    - kind: Rule
      namespace: library
      name: one-active-loan
  limitation: |
    Die ausführbare Datei ist hier ein Platzhalter und muss vor Nutzung ersetzt werden.
    Ein späterer Check belegt nur seine tatsächlich untersuchten Fälle.
    Er beweist weder Vollständigkeit der Beschreibung noch beliebige Produktionskonkurrenz.
```

### `model/library/architecture-contract-check.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Check
metadata:
  namespace: library
  name: architecture-contract-check
purpose: Untersucht Architektur- und Interfacepflichten am realisierten Stand.
spec:
  command:
    - "<architecture-contract-check>"
  uses:
    - kind: System
      namespace: library
      name: library-system
    - kind: Component
      namespace: library
      name: frontend
    - kind: Component
      namespace: library
      name: backend
    - kind: Component
      namespace: library
      name: database
    - kind: Module
      namespace: library
      name: lending-ui
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
    - kind: Interface
      namespace: library
      name: library-api
    - kind: Interface
      namespace: library
      name: inventory-access
    - kind: Interface
      namespace: library
      name: member-access
    - kind: Interface
      namespace: library
      name: loan-storage
    - kind: Interface
      namespace: library
      name: database-access
    - kind: Rule
      namespace: library
      name: domain-independent-of-storage
  limitation: |
    Der Befehl ist ein nicht ausführbarer Platzhalter.
    Vor Nutzung sind konkrete Vertrags-, Abhängigkeits- und Startprüfungen festzulegen.
    Geprüfte Modellbeziehungen beweisen keine tatsächlich eingehaltenen Codeabhängigkeiten.
```

Beide Befehle sind nicht ausführbare Platzhalter. Vor Nutzung sind tatsächliche Prüfprogramme, deren Bindungen und geprüfte Fälle festzulegen. Check-Auswahl, Ausführung und Eignung sind getrennte Befunde.

## Methodischer Prüfauftrag und Grenzen

Modellreview untersucht Fachbeschreibung, offene Entscheidungen, vollständige Beziehungen, Architekturgrenzen, Verträge und die gewünschte Beitragsteilung. Ein unabhängiger Realisierungsreview prüft die tatsächlichen Kandidatendateien, Interfaceverträge, Fachregeln, Abhängigkeitsrichtung und belastbare Nachweise.

Ein deklarierter Anbieter beweist noch keine korrekte Interfaceimplementierung. Eine deklarierte Abhängigkeit beweist keine tatsächliche Codeabhängigkeit. Eine Rule-Prosa ist kein ausgeführter Importvalidator. Strukturelle Gültigkeit, Coverage, Fachkonformität und menschliche Zustimmung bleiben verschieden.

Der [Überfälligkeitsdurchlauf](library-change-walkthrough.md) muss bei seiner Erweiterung auch die Modul-/Interfacezuordnung berücksichtigen: Eine fachliche Berechtigungsregel kann Mitglieds- und Ausleihkontext verbinden. Der Durchlauf ist keine aktuelle vollständige Impactliste dieser erweiterten Architektur.

## Herkunft und Evidenzgrenze

- Produktgespräch: `01a121f1-b948-7050-ae5d-9921b99db9c0`.
- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang der Architekturergänzung: `ec7ba72d25c7f17cedcfb61e6b77f11d5cddb5ed`, beim Beginn sauber; ausdrücklich am 13. Oktober 2026 beauftragt.
- [Methodische Vorstufe](library-methodical-alternative.md) und frühere Kind-Darstellung bleiben datierte Vorarbeit.
- Keine Projektfixture, Modellkompilierung, Tests, Provider-, Agenten- oder Studienausführung.

Alle YAML-Blöcke stehen ausschließlich in dieser Dokumentation. Vor Nutzung muss der neue Sprach- und Referenzvertrag versioniert spezifiziert und gegen die dann aktuelle Produktbasis implementiert und geprüft werden.

# Alternative B: Bibliothek mit strukturierten Beziehungen und fachlicher Prosa

Datum: 12. Oktober 2026. Status: auf Wunsch des Nutzers festgehaltene Alternative zum Facettenentwurf; kein gewählter Produktformatwechsel, keine Implementierung oder ausgeführte Prüfung.

Grundlage: [Proposal](canonical-model-proposal.md#65-alternative-b-minimale-struktur-und-methodisches-review), Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`. Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`, Branch `docs/product-model-concepts`, Eingang `83ff0931d07c01e34c3e1d61bc84cfe5fbfd1ebe`, beim Beginn sauber. Die Feldform orientiert sich am dort gelesenen Projectmodel-Schema und vorhandenen Beispielen. Dies ist kein erneuter Abgleich mit aktuellem Main; der vorherige [Main-Abgleich](canonical-model-main-comparison.md) bleibt SHA-gebunden.

Die YAML-Blöcke sind vollständig ausgeschriebene Gesprächsbeispiele in diesem Dokument. Sie wurden nicht als Projektfixture angelegt, kompiliert, angewendet oder getestet. Der Check enthält einen ausdrücklich nicht ausführbaren Platzhalter. Artefaktpfade benennen erwartete Dateien, die hier nicht erzeugt werden.

## Umfang und Annahmen

Betrachtet werden Ausleihe und Rückgabe. Beschaffung, Gebühren und Mahnungen bleiben außerhalb. Die alternative Modellierung verwendet die bestehenden groben Rollen `concept`, `rule` und `use-case`; sie benötigt keine neue Rolle `condition` und keine Fachfacetten.

Die Berechtigung über einen aktiven Mitgliedsstatus und die unveränderte Wiederholungsrückgabe sind vorgeschlagene Fachentscheidungen des Beispiels. Sie gelten nicht als unabhängig bestätigte Nutzeranforderungen. Fristberechnung und Behandlung unbekannter Ausleihkennungen bleiben ausdrücklich offen. Solche Lücken sind vor Realisierung zu entscheiden oder mit erklärtem Umfang auszunehmen.

## Dateisystem

```text
.markitect/
├── project.yaml
└── model/
    ├── manager.yaml
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
        ├── library-implementation.yaml
        ├── library-verification.yaml
        ├── library-documentation.yaml
        └── library-contract-check.yaml
```

Ein Root-Manager verantwortet diesen kleinen Ausschnitt. Seine Namespacewurzel ist leer; die Statements liegen in `library`. Der bisherige Loader verlangt genau ein Definitionsdokument pro Datei; das Beispiel übernimmt diese Grenze und benötigt keine YAML-Stream-Erweiterung. Die Definition ist die referenzierbare Einheit. Die Dateiaufteilung folgt hier der vorhandenen Ladeform. Alle Statements, Artefakte und der Check liegen direkt unter `model/library/`, passend zu ihrem Namespace `library`; der Root-Manager liegt direkt unter `model/`, passend zum leeren Namespace.

## Projektmanifest: `project.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
name: library-methodical-alternative
coverageMode: full
documentPath: docs/markitect/project.md
modelFiles:
  - .markitect/model/manager.yaml
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
  - .markitect/model/library/library-implementation.yaml
  - .markitect/model/library/library-verification.yaml
  - .markitect/model/library/library-documentation.yaml
  - .markitect/model/library/library-contract-check.yaml
```

## Verantwortung: `model/manager.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
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

Die Manager-Anweisungen formulieren methodische Pflichten. Ihre Einhaltung ist dadurch noch nicht deterministisch bewiesen. Agenten- und Ausführungsbindung sind zusätzlich nach dem gewählten Runtime-Vertrag zu konfigurieren.

## Aussagen und Beziehungen: `model/library/`

`uses` deklariert benötigten Kontext. `requires` deklariert die benötigte Verpflichtung. Ob sie eine Voraussetzung oder eine erhaltene Invariante ist, erläutert zunächst die Prosa. Eine `uses`-Kante beschreibt insbesondere nicht automatisch die Kardinalität einer Beziehung zwischen Fachinstanzen.

### `model/library/library-system.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: library-system
purpose: Beschreibt Zweck und Umfang der Bibliotheksanwendung.
spec:
  category: concept
  public: true
  description: |
    Mitglieder können vorhandene Bücher zeitweise ausleihen und zurückgeben.
    Der betrachtete Ausschnitt umfasst Ausleihe und Rückgabe.
    Beschaffung, Gebühren und Mahnungen liegen außerhalb dieses Ausschnitts.
```

### `model/library/title.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: title
purpose: Beschreibt einen Buchtitel.
spec:
  category: concept
  public: true
  description: |
    Ein Titel beschreibt ein Werk beziehungsweise eine Ausgabe.
    Mehrere physische Exemplare können demselben Titel zugeordnet sein.
```

### `model/library/copy.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: copy
purpose: Beschreibt ein einzelnes physisches Exemplar.
spec:
  category: concept
  public: true
  description: |
    Ein Exemplar besitzt eine eindeutige Identität und gehört genau einem Titel.
    Die Modelldefinition beschreibt den Fachtyp; konkrete Exemplare sind Anwendungsdaten.
  uses:
    - namespace: library
      name: title
```

### `model/library/member.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: member
purpose: Beschreibt ein Mitglied der Bibliothek.
spec:
  category: concept
  public: true
  description: |
    Ein Mitglied besitzt eine eindeutige Identität und einen Status seiner Ausleihberechtigung.
    Für diesen Entwurf unterscheiden wir aktive und nicht aktive Ausleihberechtigung.
```

### `model/library/loan.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: loan
purpose: Beschreibt eine Ausleihe und ihre Daten.
spec:
  category: concept
  public: true
  description: |
    Eine Ausleihe verbindet genau ein Mitglied mit genau einem Exemplar.
    Sie besitzt eine Identität, einen Beginn, eine Fälligkeit und einen optionalen Rückgabezeitpunkt.
    Die konkrete Fristberechnung ist vor der Umsetzung festzulegen.
  uses:
    - namespace: library
      name: member
    - namespace: library
      name: copy
```

### `model/library/active-loan.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: active-loan
purpose: Definiert den Zustand einer aktiven Ausleihe.
spec:
  category: concept
  public: true
  description: |
    Eine gespeicherte Ausleihe ist aktiv, solange kein Rückgabezeitpunkt erfasst ist.
  uses:
    - namespace: library
      name: loan
```

### `model/library/eligible-member.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: eligible-member
purpose: Definiert die Berechtigungsbedingung für das Ausleihen.
spec:
  category: rule
  public: true
  description: |
    Ein Mitglied darf ausleihen, wenn es existiert und seine Ausleihberechtigung aktiv ist.
    Die Bedingung wird bei der Vergabe geprüft.
  uses:
    - namespace: library
      name: member
```

### `model/library/available-copy.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: available-copy
purpose: Definiert die Verfügbarkeitsbedingung für das Ausleihen.
spec:
  category: rule
  public: true
  description: |
    Ein Exemplar ist für die Vergabe verfügbar, wenn es existiert und keine aktive Ausleihe besitzt.
    Eine frühere Verfügbarkeitsabfrage allein garantiert keine erfolgreiche Vergabe.
  uses:
    - namespace: library
      name: copy
    - namespace: library
      name: active-loan
```

### `model/library/one-active-loan.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: one-active-loan
purpose: Definiert die Eindeutigkeitsinvariante der Vergabe.
spec:
  category: rule
  public: true
  description: |
    Für jedes Exemplar besteht höchstens eine aktive Ausleihe.
    Diese Regel gilt auch bei konkurrierenden Anfragen und muss nach erfolgreichen
    sowie abgelehnten Operationen erhalten sein.
  uses:
    - namespace: library
      name: copy
    - namespace: library
      name: active-loan
```

### `model/library/borrow.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: borrow
purpose: Beschreibt die Vergabe eines Exemplars an ein Mitglied.
spec:
  category: use-case
  public: true
  description: |
    Ein berechtigtes Mitglied darf ein verfügbares Exemplar ausleihen.
    Erfolg erzeugt eine dauerhaft gespeicherte aktive Ausleihe.
    Ablehnung erzeugt keine neue Ausleihe.
    Bei konkurrierenden Anfragen muss one-active-loan erhalten bleiben.
    Die technische Umsetzung der atomaren Vergabe darf der verantwortliche Bereich wählen.
  uses:
    - namespace: library
      name: member
    - namespace: library
      name: copy
    - namespace: library
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
apiVersion: project.markitect.example.org/v1alpha1
kind: Statement
metadata:
  namespace: library
  name: return
purpose: Beschreibt die Beendigung einer Ausleihe durch Rückgabe.
spec:
  category: use-case
  public: true
  description: |
    Die Rückgabe einer aktiven Ausleihe wird dauerhaft durch einen Rückgabezeitpunkt erfasst.
    Danach ist diese Ausleihe nicht mehr aktiv.
    Eine wiederholte Rückgabe verändert den bereits erfassten Rückgabezeitpunkt nicht.
    Die Behandlung einer unbekannten Ausleihkennung ist vor der Umsetzung festzulegen.
  uses:
    - namespace: library
      name: loan
    - namespace: library
      name: active-loan
  requires:
    - namespace: library
      name: one-active-loan
```

`library-system` ist eine eigene Übersichtsdefinition. Es wurde keine pauschale Abhängigkeit aller Aussagen davon erfunden. Falls bestimmte Systemziele bindend auf Verträge wirken sollen, sind diese Ziele und ihre Beziehungen ausdrücklich zu ergänzen.

## Realisierung: `model/library/`

### `model/library/library-implementation.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Artifact
metadata:
  namespace: library
  name: library-implementation
purpose: Der Anwendungscode muss Fachbegriffe, Ausleihe, Rückgabe und ihre Bedingungen realisieren.
spec:
  role: implementation
  realizes:
    - namespace: library
      name: title
    - namespace: library
      name: copy
    - namespace: library
      name: member
    - namespace: library
      name: loan
    - namespace: library
      name: active-loan
    - namespace: library
      name: eligible-member
    - namespace: library
      name: available-copy
    - namespace: library
      name: one-active-loan
    - namespace: library
      name: borrow
    - namespace: library
      name: return
  paths:
    - src/Library/
  checks:
    - namespace: library
      name: library-contract-check
  required: true
  reason: Der Anwendungscode muss Fachbegriffe, Ausleihe, Rückgabe und ihre Bedingungen realisieren.
```

### `model/library/library-verification.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Artifact
metadata:
  namespace: library
  name: library-verification
purpose: Verträge und Regeln benötigen geeignete Nachweise einschließlich Konkurrenz.
spec:
  role: verification
  realizes:
    - namespace: library
      name: borrow
    - namespace: library
      name: return
    - namespace: library
      name: eligible-member
    - namespace: library
      name: available-copy
    - namespace: library
      name: one-active-loan
  paths:
    - tests/Library/
  checks:
    - namespace: library
      name: library-contract-check
  required: true
  reason: Verträge und Regeln benötigen geeignete Nachweise einschließlich Konkurrenz.
```

### `model/library/library-documentation.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Artifact
metadata:
  namespace: library
  name: library-documentation
purpose: Zweck, Bedienung und Ablehnungen müssen verständlich beschrieben werden.
spec:
  role: documentation
  realizes:
    - namespace: library
      name: library-system
    - namespace: library
      name: borrow
    - namespace: library
      name: return
  paths:
    - docs/library/
  required: true
  reason: Zweck, Bedienung und Ablehnungen müssen verständlich beschrieben werden.
```

Die konkrete Dateiabdeckung wird aus Inventar und deklarierten Pfadbereichen abgeleitet. Diese Zuordnung beweist keine Inhaltskonformität. Die erforderlichen Artefakte werden für ein tatsächliches Repository erst dann erfüllt, wenn passende Dateien und Nachweise vorhanden sind; weitere Dateien benötigen ebenfalls Zuordnung oder ausdrückliche Ausnahme.

## Prüfanschluss: `model/library/library-contract-check.yaml`

```yaml
apiVersion: project.markitect.example.org/v1alpha1
kind: Check
metadata:
  namespace: library
  name: library-contract-check
purpose: Prüft die Ausleih- und Rückgabeverträge anhand konkreter Fälle.
spec:
  command:
    - "<library-contract-check>"
  uses:
    - namespace: library
      name: borrow
    - namespace: library
      name: return
    - namespace: library
      name: eligible-member
    - namespace: library
      name: available-copy
    - namespace: library
      name: one-active-loan
  limitation: |
    Die ausführbare Datei ist hier ein Platzhalter und muss vor Nutzung ersetzt werden.
    Ein späterer Check belegt nur seine tatsächlich untersuchten Fälle.
    Er beweist weder Vollständigkeit der Beschreibung noch beliebige Produktionskonkurrenz.
```

## Methodischer Prüfauftrag

Der Modellreview prüft:

- Sind Zweck, Begriffe, Voraussetzungen, Erfolg, Ablehnung und Entscheidungsspielräume ausreichend erklärt?
- Sind alle für Kontext und Verpflichtungen relevanten Beziehungen ausdrücklich deklariert?
- Sind vorgeschlagene Fachentscheidungen angenommen und offene Punkte vor der Umsetzung geklärt?
- Haben notwendige Beiträge eine Realisierungs- und Prüfzuordnung?

Der unabhängige Realisierungsreview prüft:

- Bleibt höchstens eine aktive Ausleihe pro Exemplar auch bei konkurrierenden Vergaben erhalten?
- Erzeugt eine abgelehnte Vergabe keine neue Ausleihe?
- Beendet Rückgabe die aktive Ausleihe dauerhaft, und bleibt der ursprüngliche Rückgabezeitpunkt bei Wiederholung erhalten?
- Untersuchen die konkreten Checks diese Aussagen tatsächlich? Welche Behauptungen bleiben unbelegt?

Die Fachprüfung und Vollständigkeit der deklarierten Beziehungen bleiben methodische Aufgaben. Grobe Kategorie, Referenzauflösung und Dateizuordnung sind deterministisch prüfbare Struktur. Nicht erklärte oder unsichere Auswirkungen dürfen nicht als nachweislich vollständiger enger Impact behandelt werden. Die Alternative übernimmt keine allgemeine Vollständigkeits- oder Korrektheitsgarantie.

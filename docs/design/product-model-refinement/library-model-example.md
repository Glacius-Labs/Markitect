# Bibliothek: aktueller Modell- und Referenzentwurf

Datum: 12. Oktober 2026. Status: zusammenhängendes YAML-Gesprächsbeispiel zum [konsolidierten Konzept](canonical-model-proposal.md). Der Dateiname `mandate.yaml` sowie `Concept`, `Rule` und `UseCase` als eigene Kinds sind gewählt. `vNext`, Zielmengen und Kind-angereicherte Referenzen sind neue Vertragskandidaten, keine unterstützte Version oder kompilierte Fixture.

## Umfang und Fachentscheidungen

Ausleihe und Rückgabe bilden den betrachteten Ausschnitt; Beschaffung, Gebühren und Mahnungen liegen außerhalb. Die Mitgliedsberechtigung über einen aktiven Status und die unveränderte Wiederholungsrückgabe sind fachliche Vorschläge des Beispiels. Fristberechnung und unbekannte Kennungen bleiben ausdrücklich offen. Ein grüner Strukturcheck würde diese Lücken nicht automatisch schließen.

| Definition | Kind im Entwurf | Rolle |
| --- | --- | --- |
| `library-system` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `title` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `copy` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `member` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `loan` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `active-loan` | `Concept` | System-, Fach- oder Zustandsbeschreibung als Prosa. |
| `eligible-member` | `Rule` | Normative Bedingung beziehungsweise Invariante. |
| `available-copy` | `Rule` | Normative Bedingung beziehungsweise Invariante. |
| `one-active-loan` | `Rule` | Normative Bedingung beziehungsweise Invariante. |
| `borrow` | `UseCase` | Verhaltensvertrag als Prosa. |
| `return` | `UseCase` | Verhaltensvertrag als Prosa. |

`Concept` beschreibt fachliche Gegenstände, Begriffe und Zustände; `Rule` normative Bedingungen und Invarianten; `UseCase` Fähigkeiten und Abläufe. Für diese Unterscheidung entfällt `category`. Die Systemübersicht `library-system` ist hier ebenfalls ein Concept: Sie beschreibt, was die Bibliothek ist und welchen Zweck sie erfüllt. Ein eigener System-Kind ist nicht vorausgesetzt. Der Auftrag steht in `mandate.yaml`; sein Kind bleibt vorerst `Manager`.

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
        ├── library-implementation.yaml
        ├── library-verification.yaml
        ├── library-documentation.yaml
        └── library-contract-check.yaml
```

Eine Definition pro Datei und Namespace aus dem Verzeichnis übernehmen die Grenzen des bisher untersuchten Loaders. Alle fachlichen Definitionen, Artefakte und der Check liegen im Namespace `library`; das Root-Mandat liegt mit leerem Namespace direkt unter `model/`. Die Anwendung muss diese Dateiaufteilung nicht spiegeln.

Die Manifestliste nennt Modellquellen. Anwendungsdateien werden über Artefaktpfade erfasst; sie brauchen keine Einzelauflistung im Modellmanifest. Weitere unklassifizierte Projektdateien benötigen Zuordnung oder ausdrückliche Ausnahme.

### `project.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
name: library-methodical-alternative
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
  - .markitect/model/library/library-implementation.yaml
  - .markitect/model/library/library-verification.yaml
  - .markitect/model/library/library-documentation.yaml
  - .markitect/model/library/library-contract-check.yaml
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

Diese Anweisungen formulieren Pflichten und Entscheidungsspielräume. Die Datei beweist keine authentifizierte Befugnis oder tatsächliche Einhaltung. Runtime-/Agentenbindung ist zusätzlich nach dem gewählten Ausführungsvertrag zu konfigurieren.

## Aussagen und Regeln

`uses` liefert Kontext; `requires` benennt Verpflichtungen auf Rules. Der Kind jeder Referenz ist explizit. Die Ziel-API-Version bestimmt der einheitliche Versionsvertrag des Beispiels; Cross-Version-Verweise sind nicht implizit vorgesehen. Die konkrete Unterstützung gemischter Zielmengen ist noch zu implementieren beziehungsweise gegen aktuelles Main zu prüfen.

Eine Kante auf den Modelldatentyp `copy` beschreibt nicht automatisch eine konkrete Fachinstanz oder deren Kardinalität. Die Prosa bleibt der verbindliche fachliche Vertrag.

### `model/library/library-system.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Concept
metadata:
  namespace: library
  name: library-system
purpose: Beschreibt Zweck und Umfang der Bibliotheksanwendung.
spec:
  public: true
  description: |
    Mitglieder können vorhandene Bücher zeitweise ausleihen und zurückgeben.
    Der betrachtete Ausschnitt umfasst Ausleihe und Rückgabe.
    Beschaffung, Gebühren und Mahnungen liegen außerhalb dieses Ausschnitts.
```

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
    - kind: Rule
      namespace: library
      name: eligible-member
    - kind: Rule
      namespace: library
      name: available-copy
    - kind: Rule
      namespace: library
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
    - kind: Rule
      namespace: library
      name: one-active-loan
```

`library-system` ist eine adressierbare Übersichtsdefinition. Eine Änderung darin wird nicht durch eine frei erfundene Globalabhängigkeit automatisch auf alles verteilt. Soll ein Ziel konkrete Verträge verpflichten, muss sein Bezug ausdrücklich modelliert oder im Modellreview ergänzt werden.

## Realisierungsbeiträge

`realizes` erlaubt im Entwurf Concept-, Rule- und UseCase-Ziele; `checks` nur Check-Ziele. Die deklarierte Rolle unterscheidet Implementierung, Prüfung und Dokumentation. Eine Datei kann mehrere Aussagen betreffen; ihre konkrete Schreibverantwortung bleibt eindeutig.

### `model/library/library-implementation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: library-implementation
purpose: Der Anwendungscode muss Fachbegriffe, Ausleihe, Rückgabe und ihre Bedingungen realisieren.
spec:
  role: implementation
  realizes:
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
  paths:
    - src/Library/
  checks:
    - kind: Check
      namespace: library
      name: library-contract-check
  required: true
  reason: Der Anwendungscode muss Fachbegriffe, Ausleihe, Rückgabe und ihre Bedingungen realisieren.
```

### `model/library/library-verification.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: library-verification
purpose: Verträge und Regeln benötigen geeignete Nachweise einschließlich Konkurrenz.
spec:
  role: verification
  realizes:
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
  paths:
    - tests/Library/
  checks:
    - kind: Check
      namespace: library
      name: library-contract-check
  required: true
  reason: Verträge und Regeln benötigen geeignete Nachweise einschließlich Konkurrenz.
```

### `model/library/library-documentation.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library
  name: library-documentation
purpose: Zweck, Bedienung und Ablehnungen müssen verständlich beschrieben werden.
spec:
  role: documentation
  realizes:
    - kind: Concept
      namespace: library
      name: library-system
    - kind: UseCase
      namespace: library
      name: borrow
    - kind: UseCase
      namespace: library
      name: return
  paths:
    - docs/library/
  required: true
  reason: Zweck, Bedienung und Ablehnungen müssen verständlich beschrieben werden.
```

Die Präfixe `src/Library/`, `tests/Library/` und `docs/library/` sind bewusst breite Untersuchungsbereiche. Sie liefern keinen Nachweis einzelner Funktionen und keinen garantierten engen Blast Radius. Passende Dateien wurden durch diesen Dokumentationsauftrag nicht erzeugt.

## Prüfanschluss

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

Der Befehl bleibt ein nicht ausführbarer Platzhalter. Vor Nutzung muss ein tatsächliches Prüfprogramm samt passendem Umfang deklariert werden. Eine Check-Auswahl ist von Ausführung und Eignung für neue Prüffälle zu unterscheiden.

## Methodischer Prüfauftrag

Der Modellreview prüft Beschreibung, ausdrücklich freie/offene Entscheidungen und Vollständigkeit von `uses`/`requires`. Er geht Voraussetzungen, Erfolg, Fehler, Zustandswirkungen, Wiederholung/Konkurrenz und Realisierungs-/Prüf-/Dokumentationsbeiträge systematisch durch.

Der unabhängige Realisierungsreview untersucht die tatsächlichen Kandidatendateien gegen den geltenden Vertrag. Er beurteilt insbesondere Eindeutigkeit bei Konkurrenz, Ablehnung ohne neue Ausleihe, dauerhaft erfasste Rückgabe und Wiederholungsbehandlung. Er meldet Befunde und nicht untersuchte Aussagen; deklarierte Pflichten und feste Digests beweisen nicht die tatsächliche Einhaltung.

Der [Überfälligkeitsdurchlauf](library-change-walkthrough.md) zeigt eine neue Bedingung und die nötige Modellergänzung. Zusätzliche technische oder organisatorische Details außerhalb dieses Ausschnitts sind vor einer vollständigen Projektumsetzung zu beschreiben oder ausdrücklich aus dem Umfang auszunehmen.

## Herkunft und Evidenzgrenze

- Produktgespräch: `01a121f1-b948-7050-ae5d-9921b99db9c0`.
- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang der Konsolidierung: `07e03215c968302361f5acedc8e69bd573ac95be`; beim Beginn sauber.
- Kind-Aktualisierung am 12. Oktober 2026 ausdrücklich beauftragt; Eingang `5d18e8f4438cc1d24a10882f04b10501d65db20e`, beim Beginn sauber.
- Vorstufe: [methodische v1alpha1-Darstellung](library-methodical-alternative.md); deren Feldform und Referenzen sind nicht dieselben wie dieser vNext-Kandidat.
- Kein neuer Main-Abgleich, keine angelegte Projektfixture, Modellkompilierung, Tests, Provider- oder Agentenausführung.

Die YAML-Blöcke liegen ausschließlich in dieser Markdown-Dokumentation. Der neue Rule-/Referenzvertrag muss vor produktiver Nutzung versioniert spezifiziert und geprüft werden.

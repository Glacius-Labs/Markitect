# Bibliothek: Dateisystem und Modellsyntax als Gesprächsbeispiel

Datum: 11. Oktober 2026. Status: auf Wunsch des Nutzers festgehaltener Syntaxentwurf für die gemeinsame Arbeit mit dem Implementer. Kein vollständiges, kompilierbares Modell und keine angenommene Formatdefinition.

Grundlage: [Modell-Proposal](canonical-model-proposal.md), Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`. Ablage: Worktree `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`, Branch `docs/product-model-concepts`; Eingang dieser Ergänzung `8f5fa92aed4e8f13d1abbaa2f458a0542b18db93`, beim Beginn sauber. Das Beispiel wird nur in diesem Dokument gezeigt; die beschriebenen Projektdateien wurden nicht als ausführbare Fixtures angelegt.

## Mögliche Ablage

Die Gliederung folgt fachlicher Kohäsion. Innerhalb eines Bereichs stehen Begriffe, Regeln, Verhalten und Realisierungsverpflichtungen zusammen. Die Ordner bündeln Inhalte; explizite Modellreferenzen bilden den Graphen. Die konkrete Manager-/Namespacezuordnung bleibt ausdrücklich zu definieren.

```text
library/
├── .markitect/
│   ├── project.yaml                 # Explizite Modelleingaben und Projektumfang
│   ├── runtime.yaml                 # Agenten- und Ausführungsbindung
│   └── model/                       # Kanonische Projektbeschreibung
│       ├── manager.yaml             # Oberste Verantwortung
│       ├── library/
│       │   ├── manager.yaml
│       │   ├── overview.yaml        # Zweck, Nutzer, Systemgrenze
│       │   ├── goals.yaml           # Ziele und Qualitätsanforderungen
│       │   ├── architecture.yaml    # Bausteine und Schnittstellen
│       │   ├── inventory/
│       │   │   ├── manager.yaml
│       │   │   ├── title.yaml       # Was ein Buchtitel ist
│       │   │   ├── copy.yaml        # Was ein Exemplar ist
│       │   │   └── artifacts.yaml
│       │   ├── membership/
│       │   │   ├── manager.yaml
│       │   │   ├── member.yaml
│       │   │   ├── eligible-member.yaml
│       │   │   └── artifacts.yaml
│       │   └── circulation/
│       │       ├── manager.yaml
│       │       ├── loan.yaml
│       │       ├── borrow.yaml
│       │       ├── return.yaml
│       │       ├── one-active-loan.yaml
│       │       └── artifacts.yaml
│       ├── engineering/
│       │   ├── manager.yaml
│       │   ├── checks.yaml
│       │   └── workflow.yaml
│       └── operations/
│           ├── manager.yaml
│           ├── environment.yaml
│           ├── configuration.yaml
│           └── backup.yaml
├── src/                             # Realisierung durch Anwendungscode
├── tests/                           # Realisierung der Prüfpflichten
├── docs/                            # Lesbare Dokumentation und Modellsichten
└── deployment/                      # Realisierung der Betriebsanforderungen
```

Nicht jedes Projekt benötigt diese Bereiche oder diese Anzahl Dateien. Mehrere Definitionen können in einer Datei liegen. Die adressierbare Einheit ist die Definition; die Datei ist ihre Ablage. `project.yaml` muss die tatsächlichen Modelleingaben ausdrücklich auswählen; der Baum ist keine automatische Lade- oder Zuständigkeitsregel. In `docs/` können abgeleitete Sichten und weitere realisierte Informationspflichten liegen; kanonische Aussagen sollen nicht nochmals unabhängig gepflegt werden.

## Fachbegriff: `inventory/copy.yaml`

Alle folgenden YAML-Ausschnitte sind illustrative Syntax. `vNext` ist ein Platzhalter. Insbesondere `facets`, ihre Feldnamen und ihre Prüfregeln sind noch kein unterstützter Formatvertrag.

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Statement
metadata:
  namespace: library.inventory
  name: copy
purpose: Beschreibt ein einzelnes physisches Buchexemplar.
spec:
  category: concept
  public: true
  description: |
    Ein Exemplar gehört zu einem Buchtitel.
    Mehrere Exemplare können denselben Titel besitzen.
  facets:
    domain:
      identity: copyId
      properties:
        - name: copyId
          type: string
          required: true
        - name: title
          type: reference
          target:
            namespace: library.inventory
            name: title
          required: true
```

Diese Definition beschreibt den Fachtyp Exemplar. Konkrete Bücher und Kennungen gehören zur Anwendung. Der Modellverweis `target` benennt den Typ Titel; die Facheigenschaft `title` würde zur Laufzeit einen konkreten Titel referenzieren. Wie diese beiden Referenzebenen in der Syntax unterschieden und validiert werden, ist eine offene Formatentscheidung.

## Verhalten: `circulation/borrow.yaml`

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Statement
metadata:
  namespace: library.circulation
  name: borrow
purpose: Beschreibt das Ausleihen eines Exemplars.
spec:
  category: use-case
  public: true
  description: |
    Ein berechtigtes Mitglied kann ein verfügbares Exemplar ausleihen.
  uses:
    - namespace: library.inventory
      name: copy
    - namespace: library.membership
      name: member
  requires:
    - namespace: library.membership
      name: eligible-member
    - namespace: library.circulation
      name: one-active-loan
  facets:
    behavior:
      inputs:
        - name: memberId
          type: string
        - name: copyId
          type: string
      success: Eine aktive Ausleihe wird dauerhaft gespeichert.
      failure: Es wird keine zusätzliche Ausleihe erzeugt.
      concurrency: |
        Bei konkurrierenden Anfragen für dasselbe Exemplar
        darf höchstens eine Anfrage eine aktive Ausleihe erzeugen.
      delegatedDecisions:
        - Die technische Umsetzung der atomaren Vergabe.
```

`one-active-loan.yaml` soll eine unabhängig referenzierbare Regel beschreiben: Für jedes Exemplar darf höchstens eine aktive Ausleihe bestehen. Ausleihen und Rückgaben können sie gemeinsam referenzieren. Eine formalisierte Fassung benötigt eine definierte Instanz-, Zustands- und gegebenenfalls Zeitsemantik. Die Texte in `success`, `failure` und `concurrency` sind hier weiterhin Prosa; ihre Aufnahme in eine typisierte Facette beweist ihr Verhalten nicht.

## Realisierung: `circulation/artifacts.yaml`

Dieser Ausschnitt orientiert sich an den vorhandenen Artifact-Strukturen. Die zukünftige Version sowie eventuell feinere Beitragstypen bleiben Teil der Formatentscheidung.

```yaml
apiVersion: project.markitect.example.org/vNext
kind: Artifact
metadata:
  namespace: library.circulation
  name: lending-implementation
purpose: Ordnet die Ausleihimplementierung ihren Verpflichtungen zu.
spec:
  role: implementation
  realizes:
    - namespace: library.circulation
      name: borrow
    - namespace: library.circulation
      name: one-active-loan
  paths:
    - src/Library/Circulation/
  checks:
    - namespace: engineering
      name: concurrent-borrowing
  required: true
  reason: Ausleihen muss die Eindeutigkeit auch bei Konkurrenz erhalten.
```

Weitere Artifact-Definitionen sollen Tests und Bedienungsdokumentation zuordnen. Die konkreten Quelldateien im Pfadbereich erfasst der abgeleitete Dateibestand. Damit werden Erwartungen und Zuordnungsregeln gepflegt, ohne jede private Hilfsdatei einzeln als Fachdefinition aufzuführen. Vollständige Coverage und geeignete Inhaltsprüfung bleiben getrennte Aufgaben.

## Gemeinsam festzulegende Syntax

Für die Besprechung mit dem Implementer:

- Welche Definitionen brauchen eigene Identität, welche Inhalte bleiben innerhalb einer Definition?
- Bleiben die Rollen Facetten auf Statements oder erhalten einzelne Rollen eigene Kinds?
- Wie werden Werttypen, Modellreferenzen, Fachinstanzbeziehungen und Kardinalitäten dargestellt?
- Wie werden benannte Regeln, Prosa, formale Bedingungen und ausdrückliche Entscheidungsspielräume unterschieden?
- Welche Bindungen gelten zwischen Namespace, Manager, Quelldatei und realisiertem Pfadbereich?
- Wie werden alle Definitionen explizit geladen, versioniert und in Kontext, Impact, Graph und Evidenz berücksichtigt?
- Welche Angaben muss der Compiler erzwingen, welche bewertet der methodische Ablauf? Siehe die [offene Strenge-Frage](canonical-model-proposal.md#64-offene-frage-strenge-und-methodik).

Das nächste Ergebnis ist ein gemeinsam gewählter Formatvertrag anhand dieses Beispiels und eines CLI-Kontrastfalls. Dieses Dokument selbst enthält weder ein vollständiges Modell noch Ausführungs- oder Testevidenz.

# Markitect Architektur

**Status: Architektur mit implementiertem Pilot.** Überarbeitet am 30.09.2026. Das Grundmodell, die Go-CLI, erzeugte Schemas, Snapshot-Prüfung, kontrollierte Wiederverwendung lokaler AI-Berichte und Konfyra-Migration sind implementiert. Der [Modul-Leitfaden](../../../tools/markitect/README.md) beschreibt das tatsächlich verfügbare Verhalten; der [Umsetzungsplan](markitect-implementation-plan.md) trennt geprüfte Ergebnisse und offene Abnahme. Authentifizierte CI-Nachweise und Kubernetes bleiben weitere Ausbauschritte.

AI-Artefakte bilden eine Softwarearchitektur: Regeln definieren Anforderungen, Workflows organisieren Arbeit, Skills eröffnen einen Zugang, Agents übernehmen Verantwortlichkeiten und Texte liefern Wissen. Ihre Abhängigkeiten und Grenzen sollen ebenso ausdrücklich und überprüfbar werden wie die von Programmcode. Natürliche Sprache bleibt für Bedeutung und Begründung erhalten.

**Markitect** steht für Markdown + Architect und bezeichnet das gesamte System. Die CLI heißt `markitect`. Markitect wird in Go mit YAML als einheitlichem eigenen Dateiformat und einem kleinen Vokabular entwickelt. Es prüft feste Quellstände, erzeugt Provider-Dateien und begrenzt erneute Reviews auf die betroffenen Inhalte. Das Ressourcenmodell bereitet eine spätere Kubernetes-Erweiterung vor.

Markitect entsteht zunächst als eigenständiges Go-Modul unter `tools/markitect/` im aktuellen Workspace. Dieser Entwicklungsort bedeutet noch keine Migration des Cockpits. Das Modul muss aus diesem Ordner allein gebaut und getestet werden können und wird perspektivisch als eigenes Repository **Markitect** herausgelöst. Konfyra und das Cockpit verwenden versionierte Releases derselben Quelle mit ihren jeweiligen Profilen.

| Gegenstand | Name |
|---|---|
| System und künftiges Repository | `Markitect` |
| CLI und Binary | `markitect` beziehungsweise `markitect.exe` |
| Projektkonfiguration | `markitect.yaml` |
| Fixierte Werkzeugversion und SHA-256 des Quellpakets | `markitect.lock.yaml` |
| Lokale Prüfausgaben | `.artifacts/markitect/` |
| Künftiger Controller | `markitect-controller`, sobald er benötigt wird |

Die endgültige Go-Moduladresse und API-Domain werden beim Festlegen des Repository-Hosts beziehungsweise der kontrollierten Domain bestimmt. Der Name Markitect beansprucht keine bereits vorhandene Domain oder Paketregistrierung.

## Leitgedanken

Die Ausgangsfrage ist „Clean Code und Clean Architecture für AI-Systeme“: Wie lassen sich Verantwortung, Abstraktion, Abhängigkeitsrichtung und Änderbarkeit auch dann ordnen, wenn ein Teil des Systems aus Sprache besteht?

| Prinzip | Konkrete Konsequenz |
|---|---|
| Eine Verantwortung | Jede Regel beantwortet eine Frage; jeder Agent hat eine erkennbare Aufgabe. Ein neuer Typ braucht eine eigene prüfbare Bedeutung. |
| Eine maßgebliche Quelle | Inhalte werden einmal gepflegt. Provider-Dateien und lesbare Ansichten sind erzeugte Ausgaben. |
| Ausdrückliche Abhängigkeiten | Benötigte Regeln, Texte und Mechanismen stehen in typisierten Referenzen. |
| Abhängigkeit von Abstraktionen | Ein Ablauf kann einen Vertrag verlangen. Das Projekt wählt eine passende Implementierung. |
| Komposition | Vorhandene Inhalte und Mechanismen werden referenziert. Verdeckte Textvererbung und automatische Überschreibungen entfallen. |
| Einfachheit | Kurze Namen, wenige Typen und eine normale CLI. Neue Möglichkeiten kommen hinzu, wenn ein konkreter Anwendungsfall sie benötigt. |

Der Nutzen einer Abstraktion ist ihre Änderungsgrenze: Ein Review-Agent kann ausgetauscht werden, während der Workflow weiterhin denselben Review-Vertrag benötigt. Eine rein dekorative zusätzliche Ebene wird nicht eingeführt. Fachliche Widerspruchsfreiheit lässt sich aus beliebiger Prosa weiterhin nicht vollständig beweisen.

## Grundlage in beiden Repositories

Die ursprüngliche Untersuchung verwendet Konfyras lokalen Commit `9dc1cd558ae37fbb26e1f0fbc33b824b95144929`. Bei der Überarbeitung stand der lokale `master` auf `f3529fa32e4dd9b408b73cf7f29159849de03466`; der Vergleich zeigte keine Änderungen an `AGENTS.md`, `docs/general/` oder `scripts/`. Beide Angaben beziehen sich auf lokale Git-Stände. Das Cockpit wurde aus dem Arbeitsverzeichnis untersucht und hatte dabei kein Git-Repository. Die vom Nutzer berichteten Stundenläufe sind noch nicht vermessen.

| Gegenstand | Konfyra | Cockpit |
|---|---|---|
| Zuständigkeit | General, Core, Modules und Products einschließlich konkreter Bereiche | General, Consiliari, Kunde und Projekt |
| Quellen | Definitionen unter dem zuständigen `docs/`-Bereich | Dasselbe Eigentumsprinzip |
| Navigation | README-Router mit direkten Kindern | Dasselbe Router-Prinzip |
| Heute vorhandene Formate | Markdown, YAML für Skills, JSON-Frontmatter für Agents und Adapter-Mapping | Ähnliche Mechanismusformate |
| Heute vorhandene Werkzeuge | Python-Renderer, gemeinsamer Konsistenz-Runner und Regressionstests | Python-Renderer, Dokumentationschecker und Regressionstests |
| Bestehende Prüfungen | Adapter, Router, Links/Anker, Workflow-Einstiege und Code-Dokumentationszuordnung | Adapter, Router, Dateilinks und Workflow-Einstiege |
| Nachweisansatz | Work-Item-Hashes sowie Kandidaten- und Ziel-Commits | Externe Quellen mit beobachteten Git-Blobs |

Die Konfyra-Quellen sind `scripts/render-governance-adapters.py`, `scripts/check-repository-consistency.py`, `docs/general/rules/documentation.md`, `docs/general/workflows/ai-mechanism-authoring.md` und `docs/general/workflows/work-item-records.md` in den genannten Ständen. Sie bleiben Eigentümer ihrer Projektvorgaben.

Im Cockpit gelten die [Dokumentationsregel](../rules/documentation.md), die [Mechanismusregel](../rules/mechanisms.md) und das [Tooling](../tooling.md). Vorhandener Code und Tests liefern Migrationsanforderungen. Die ausdrückliche Werkzeugpräferenz des Nutzers ist Go; die heutige Implementierungssprache verpflichtet den neuen Kern nicht zu Python.

## Ein Format und eine Ressourcenform

Eigene Artefakte, Projektkonfiguration, Importsperren, Kontextmanifeste und Prüfnachweise verwenden YAML. Im neuen Modell gibt es keine zusätzliche TOML-Konfiguration und keine separat zu pflegenden JSON-Register. Markdown bleibt als Inhalt eines Textfeldes und als lesbare Dokumentationsansicht möglich. Die bestehenden README-Router bleiben Navigationsdokumente.

Provider können andere Zielformate verlangen. Diese werden ausschließlich erzeugt. Ebenso kann eine Kubernetes-API intern anders serialisieren. Das ändert nichts daran, dass Autoren die neuen Quellen einheitlich in YAML pflegen.

Jede Ressource hat dieselbe Form:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata:
  name: introduction
  namespace: cockpit-general
spec:
  text: |
    Das Cockpit bündelt Arbeitskontexte und gemeinsame Konventionen.
```

`markitect.example.org` ist ein ausdrücklich unverbindlicher Beispielname. Vor Veröffentlichung wird eine API-Gruppe unter einer tatsächlich kontrollierten Domain festgelegt. `apiVersion` versioniert das Format. `kind` bezeichnet den Typ, `metadata` die Identität und `spec` den gewünschten Inhalt. Die Form orientiert sich an [Kubernetes-Objekten](https://kubernetes.io/docs/concepts/overview/working-with-objects/).

Eine Datei enthält eine Ressource. Ein YAML-Dokument mit `spec.text: |` kann normalen Text oder Markdown enthalten. Es gibt keine Frontmatter-Syntax, Include-Makros, Textinterpolation oder zweite Sidecar-Datei für denselben Inhalt. Die Datei bleibt auch ohne Spezialeditor nutzbar.

Der YAML-Parser lehnt doppelte Schlüssel, unbekannte Felder, benutzerdefinierte Tags, Aliase und Merge-Keys ab. Namen verwenden Kleinbuchstaben, Ziffern und Bindestriche und bleiben im vorgeschlagenen Modell höchstens 63 Zeichen lang. Zeichenfolgen, Wahrheitswerte und Zahlen werden durch das Ressourcenschema unterschieden. Fehlermeldungen nennen Pfad und Zeile.

## Die sechs Inhaltstypen

| Typ | Bedeutung | Beispiel |
|---|---|---|
| `Text` | Gespeicherter Inhalt ohne eigene ausführbare oder normative Bedeutung | Begriffserklärung, Hintergrund, Notiz |
| `Rule` | Geltende Anforderung | Dokumentation beim zuständigen Eigentümer ändern |
| `Contract` | Erwartete Fähigkeit oder Schnittstelle | Änderung prüfen und Befunde liefern |
| `Workflow` | Beschriebener Ablauf | Änderung vorbereiten, prüfen und integrieren |
| `Skill` | Einstieg in einen Ablauf | AI-Mechanismus anlegen oder ändern |
| `Agent` | Konkrete Verantwortung und Arbeitsanweisung | Dokumentation auf Konsistenz prüfen |

`Project` dient zusätzlich als Konfigurationsressource für Bereiche, Bindungen und gewählte Provider. Erzeugte Nachweise sind Ausgaben des Werkzeugs. Eigene Typen für Decisions, Router, Registries, Hooks oder Tools entstehen im ersten Schritt nicht. Bestehende ADRs behalten ihre Konventionen; bei Bedarf kann eine neue Hintergrundnotiz als `Text` geführt werden. Ausführbare Hooks und Tools benötigen später eine eigene geprüfte Laufzeit- und Berechtigungssemantik.

`Text` ist die kleinste Einheit. Er wird erst geladen, wenn ein Auftrag ihn benötigt. Sein Inhalt wird nicht durch die Ablage zum globalen Prompt oder zur Regel. Markdown-Links innerhalb eines Textes sind zunächst Navigation. Verbindlich benötigte Quellen werden zusätzlich strukturiert angegeben; ein Textlabel darf tatsächliche Regelabhängigkeiten nicht verbergen.

## Kleine und eindeutige Namen

Eine Ressource wird über API-Gruppe, Typ, Namespace und Name identifiziert. Der Dateipfad ist ihr Speicherort. Dateiumbenennungen ändern die Identität nicht. API-Versionen bezeichnen Versionen derselben Ressource, keine zusätzlichen Identitäten. Ein Wechsel von Name oder Namespace ist eine ausdrückliche Referenzmigration.

Referenzen sind gewöhnliche YAML-Objekte statt einer eigenen Zeichenketten-Sprache:

```yaml
kind: Workflow
name: author-mechanism
namespace: cockpit-general
```

Im selben Namespace wird `namespace` weggelassen. In einem typisierten Feld wie `rules` ist `kind: Rule` bereits bekannt und wird weggelassen. Die Suche springt nicht automatisch in Elternbereiche oder andere Repositories. Provider-Namen werden zunächst aus `metadata.name` abgeleitet; Kollisionen werden gemeldet. Ein zusätzlicher Exportname wird erst eingeführt, wenn ein tatsächlicher Provider-Konflikt ihn erfordert.

Namespaces bilden fachliche Zuständigkeiten ab. Ihre erlaubte Zuordnung zu Repository-Pfaden steht einmal in `Project`. Ein Autor kann sich durch das Ändern von `metadata.namespace` keine allgemeinere Zuständigkeit geben: Der Validator vergleicht Namespace und Speicherort.

| Repository-Pfad | Vorgeschlagener Namespace |
|---|---|
| Cockpit `docs/general/` | `cockpit-general` |
| Cockpit `docs/consiliari/` | `cockpit-consiliari` |
| Cockpit `docs/customers/septeo/` | `cockpit-septeo` |
| Cockpit `docs/customers/septeo/projects/wz-assist/` | `cockpit-septeo-wz-assist` |
| Konfyra `docs/general/` | `konfyra-general` |
| Konfyra `docs/core/` | `konfyra-core` |
| Konfyra `docs/modules/analysis/appraisal/` | `konfyra-appraisal` |
| Konfyra `docs/products/LSJV/` | `konfyra-lsjv` |

Diese Namen sind Vorschläge. Bei verschachtelten Pfaden gewinnt der längste passende Bereichspfad; gleichwertige Mehrdeutigkeit ist ein Fehler. Die übrigen tatsächlichen Bereiche werden analog aufgenommen. Kubernetes-Namespaces sind flach: Die fachliche Hierarchie und zulässige Nutzung werden ausdrücklich im Projektprofil beschrieben.

## Wenige Beziehungsfelder

| Feld in `spec` | Bedeutung | Zulässige Verwendung |
|---|---|---|
| `text` | Inhalt oder Arbeitsanweisung | Alle Inhaltstypen |
| `rules` | Verbindliche Regeln | Agent, Skill, Workflow und Contract → Rule |
| `uses` | Direkt verwendete Inhalte oder konkrete Mechanismen | Skill → Workflow/Text; Workflow → Workflow/Skill/Agent/Text; Agent → Skill/Workflow/Text; Contract → Text |
| `needs` | Benötigte abstrakte Fähigkeit | Agent, Skill oder Workflow → Contract |
| `implements` | Erfüllter Vertrag | Agent, Skill oder Workflow → Contract |
| `input`, `output` | Benannte Ein- und Ausgaben | Contract und seine Implementierungen |

Nicht jeder Typ darf jedes Feld verwenden. Ein `Text` braucht nur `text`; eine `Rule` braucht ihren Text und kann optional eine implementierte Prüfung über `check` benennen. Der konkrete `kind` einer `uses`-Referenz macht deren Zieltyp prüfbar. Die Feldnamen sind bewusst kurz; separate Begriffe wie `routesTo`, `governedBy` oder `generatedFrom` sind für Autoren nicht erforderlich.

`uses` bindet eine fachliche Abhängigkeit in den Kontext ein. Es startet keinen Prozess. Reihenfolge und Vorgehen stehen zunächst im Workflow-Text; ein späterer ausführbarer Runner müsste dafür eine ausdrückliche Schrittsemantik ergänzen. Der aufgelöste Graph aus `uses` und den Implementierungen von `needs` darf in V1 keine Rekursion enthalten. Bei Contracts sind Zyklen zwischen „benötigt“ und „implementiert“ nicht schon für sich Ausführungszyklen: Entscheidend ist der gebundene Verbrauchergraph.

Die Herkunft erzeugter Dateien ermittelt der Renderer selbst. Router-Links bilden einen separaten Navigationsgraphen, in dem Rückverweise erlaubt sind. Allgemeine Regelgeltung ergibt sich zusätzlich aus dem ausgewählten Bereich und dessen Profil; sie kann nicht durch das Weglassen eines `rules`-Eintrags aufgehoben werden.

## Ein Skill als vollständiges Beispiel

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Skill
metadata:
  name: author-mechanism
  namespace: cockpit-general
spec:
  description: Einen AI-Mechanismus anlegen oder ändern.
  rules:
    - name: documentation
    - name: mechanisms
  uses:
    - kind: Workflow
      name: author-mechanism
  text: |
    Verwende diesen Skill für Änderungen an Regeln, Workflows,
    Skills oder Agents. Bestimme zuerst den zuständigen Bereich
    und folge dem referenzierten Workflow.
```

Konfyras Skill kann mit derselben Struktur `author-ai-mechanism` im Namespace `konfyra-general` heißen und den dortigen Workflow `ai-mechanism-authoring` verwenden. Die Projektinhalte bleiben getrennt, während derselbe Parser und dieselben Prüfungen arbeiten.

## Abstraktion durch Verträge

Ein Vertrag beschreibt eine Erwartung, die mehrere Implementierungen erfüllen können. Zunächst reicht ein einfacher Vertrag mit Implementierungstyp, benannten Eingaben, benannten Ergebnissen und erklärendem Text:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Contract
metadata:
  name: review
  namespace: cockpit-general
spec:
  kind: Agent
  input: [change]
  output: [findings]
  text: |
    Prüfe eine abgegrenzte Änderung. Liefere konkrete Befunde
    mit Quelle und Begründung. Benenne unvollständige Prüfungen.
```

Eine konkrete Implementierung verweist darauf:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Agent
metadata:
  name: documentation-reviewer
  namespace: cockpit-general
spec:
  implements:
    - name: review
  input: [change]
  output: [findings]
  text: |
    Prüfe Eigentum, Referenzen und Konsistenz der betroffenen
    Dokumentation. Verwende die Ergebnisse der Strukturchecks.
```

Ein Workflow kann die abstrakte Fähigkeit verlangen:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Workflow
metadata:
  name: review-change
  namespace: cockpit-general
spec:
  needs:
    - name: review
  text: |
    Stelle den Änderungskandidaten bereit, lasse ihn prüfen
    und bearbeite die gemeldeten Befunde.
```

Das Projekt bindet den Vertrag ausdrücklich an einen Agenten. Es gibt keine automatische Auswahl aus allen passenden Implementierungen. Fehlende oder mehrdeutige Bindungen verhindern die Auflösung des konkreten Arbeitskontexts. Ein strukturell gültiger Vertrag darf bereits ohne Bindung existieren; eine bindungsabhängige Prüfung ist dann sichtbar unvollständig.

In V1 müssen Typ und Ein-/Ausgaben der Implementierung exakt zu jedem angegebenen Vertrag passen. Eingaben und Ausgaben sind zunächst eindeutige symbolische Namen, noch keine Laufzeit-Datenschemas. Das ermöglicht einfache Strukturchecks und verhindert eine voreilige eigene Typsprache. Verhaltensevaluation und fachliches Review prüfen, ob der Agent die versprochene Leistung tatsächlich erbringt. Ein Eintrag unter `implements` allein beweist das nicht.

Verträge werden eingeführt, wenn mehrere Implementierungen oder unabhängige Änderungsrichtungen vorhanden sind. Ein einfacher Workflow darf direkt einen bekannten Agenten verwenden. Textwiederverwendung erfolgt über `uses` mit `kind: Text`; sie erzeugt weder Vererbung noch Textersetzung.

## Projektkonfiguration ebenfalls als YAML

Die Datei `markitect.yaml` enthält eine `Project`-Ressource. Das folgende Beispiel zeigt nur den General-Bereich; die produktive Konfiguration müsste alle tatsächlich erfassten Bereiche nennen:

```yaml
apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: cockpit
spec:
  profile: cockpit
  targets: [codex, claude]
  areas:
    - name: cockpit-general
      path: docs/general
  bindings:
    - contract:
        name: review
        namespace: cockpit-general
      implementation:
        kind: Agent
        name: documentation-reviewer
        namespace: cockpit-general
```

Konfyra verwendet `profile: konfyra` und seine eigenen Bereiche. Das Profil enthält geprüfte Regeln zu Zuordnung und Geltung, keine Kopie aller Artefakte. `Project` ist eine lokale Konfiguration ohne Namespace; eine spätere CRD dafür wäre ausdrücklich clusterweit. Inhaltstypen sind für namespaced CRDs vorgesehen.

Ein Bereich kann mit `imports` ausdrücklich nutzbare andere Bereiche nennen und mit `rules` allgemein anzuwendende Regeln referenzieren. Bereichsimporte sind nicht transitiv und laden keine Inhalte auf Vorrat. Eine konkrete Referenz muss zusätzlich den Typ- und Profilregeln entsprechen. Fachliche übergeordnete Anforderungen werden über das Profil aufgelöst und bleiben Teil jedes zutreffenden Kontextes.

Im Cockpit werden Kundenregeln nur im passenden Auftrag geladen. In Konfyra folgen Bereichsnutzung und Codebezüge den bestehenden Architekturregeln. Ein gemeinsamer Validator darf diese beiden Politiken nicht gleichsetzen. Namespace-Zugriff ist außerdem keine menschliche Befugnis und ersetzt weder Betriebssystemrechte noch Kubernetes-RBAC.

## Go und Clean Architecture im Werkzeug

Der Kern besteht aus gewöhnlichen Go-Typen für Artefakte, Referenzen, Contracts, Diagnosen und Nachweise. Seine Prüfungen arbeiten auf bereits geladenen Werten und kennen weder Git-Kommandos noch Kubernetes-Clients, Provider-Dateipfade oder Netzwerke.

```text
CLI oder später Kubernetes-Controller
                  |
        Anwendungsfälle und Prüfablauf
                  |
       Artefakte, Verträge und Regeln
                  ^
                  |
    Adapter für YAML, Git und Provider
```

Die Darstellung zeigt Aufrufwege; Codeabhängigkeiten zeigen zum Kern. Schnittstellen werden an tatsächlich austauschbaren Grenzen definiert, etwa Quellzugriff und Ausgabespeicher. Kleine reine Prüffunktionen benötigen kein eigenes Interface. Die CLI ist zunächst ein einzelnes Go-Binary und funktioniert ohne Cluster.

Versionierte Go-API-Typen und Validierungsdeklarationen bilden die Quelle für erzeugte YAML-Schemas. Für die spätere CRD-Erzeugung ist der von [Kubebuilder beschriebene Generatoransatz](https://book.kubebuilder.io/reference/generating-crd.html) geeignet. Schema und Go-Validierung dürfen nicht unabhängig voneinander dieselben Felder definieren; generierte Ausgaben und gemeinsame Fixtures prüfen ihre Übereinstimmung. Repository-Abhängigkeiten und inhaltliche Reviews bleiben eigene Prüfungen des Kerns.

Die vorhandenen Python-Checks können während der Migration als ausdrücklich registrierte Adapter weiterlaufen. Sie werden nur ersetzt, wenn Go-Prüfungen ihre belegten Aufgaben übernehmen. Die öffentliche Oberfläche und das Zielformat des neuen Werkzeugs bleiben Go und YAML. Beide Repositories verwenden dieselbe festgelegte Werkzeugversion und passende Profile, keine kopierten Implementierungen.

Der vorgeschlagene CLI-Einstieg bleibt klein. Die folgenden Befehle sind noch nicht implementiert; Commit-Platzhalter werden bei Ausführung durch feste IDs ersetzt:

```text
markitect check --revision COMMIT
markitect impact --base BASE_COMMIT --revision COMMIT
markitect context --revision COMMIT --kind Skill --name author-mechanism --namespace cockpit-general
markitect render --check
```

`check` prüft, `impact` erklärt betroffene Inhalte, `context` liefert den benötigten Arbeitskontext und `render --check` vergleicht erzeugte Ausgaben. Ohne ausdrücklichen Snapshot liefern Arbeitsverzeichnisprüfungen nur vorläufige Hinweise. Maschinenlesbare Ergebnisse werden als YAML ausgegeben. Exit-Code 0 steht für bestandene angeforderte Prüfungen, 1 für Befunde und 2 für eine unvollständige oder nicht ausführbare Prüfung. Ein späterer Schreibmodus folgt den Schreibgrenzen des jeweiligen Repositories.

## Der Weg zu Kubernetes

YAML mit passender Objektform bereitet die Integration vor. Tatsächliche Kubernetes-Unterstützung verlangt registrierte CustomResourceDefinitions mit strukturellen Schemas und einen Controller für die gewünschte Verarbeitung. CRDs verwenden dabei einen begrenzten OpenAPI-Schemaansatz; ein beliebiges JSON-Schema-2020-12-Dokument ist kein Ersatz. [Kubernetes CRDs](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)

Der Ausbau erfolgt in drei Schritten:

1. Lokale Go-CLI mit dem dargestellten Ressourcenmodell und reproduzierbaren Checks.
2. Generierte CRDs für ausgewählte Typen, getestet mit API-Server-Validierung und Roundtrips. Lokale und Serverprüfung müssen ihre bewusst unterschiedlichen Grenzen ausweisen.
3. Controller, der veröffentlichte Quellstände prüft und erzeugte Ergebnisse beziehungsweise Status aktualisiert.

Ein Controller kann später gewünschte Artefakte und tatsächlich erzeugte Provider-Ausgaben abgleichen. Er schreibt beobachtete Ergebnisse in `status`, während Git den gewünschten Inhalt in `spec` besitzt. Für Status gibt es einen eigenen Kubernetes-Mechanismus. [Status-Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)

Die Prüfung mehrerer Ressourcen braucht auch im Cluster einen gemeinsamen Snapshot: Mehrere aufeinanderfolgende `kubectl apply`-Operationen bilden keine Transaktion über den gesamten Abhängigkeitsgraphen. Ein späterer Controller muss einen vollständig veröffentlichten Commit beziehungsweise ein unveränderliches Paket auflösen und dessen Digest im Ergebnis nennen. Ein geändertes `resourceVersion` eines einzelnen Objekts ersetzt diesen Nachweis nicht.

Kubernetes-Schemas prüfen Objektformen. Referenzauflösung über mehrere Objekte und Repository-Grenzen übernimmt der gemeinsame Kern; ein Controller meldet unvollständige Zustände, bevor er Ergebnisse als verwendbar markiert. Unbekannte Felder müssen im vorgesehenen Einlieferungsweg abgewiesen werden; Kubernetes kann sie sonst beschneiden. Das erfordert eine ausdrücklich getestete Validierungsstrategie.

Umfangreiche Dokumentationsbestände und Nachweise bleiben in Git beziehungsweise einem Artefaktspeicher. Der Cluster braucht nur die für den Betrieb ausgewählten Ressourcen und Paketverweise. Referenzen sind fachliche Abhängigkeiten und werden nicht automatisch zu Kubernetes-`ownerReferences` mit deren Löschsemantik. Eine Agent-Ausführung erfordert später eigene Laufzeit-, Wiederholungs- und Berechtigungsregeln.

## Deterministische und semantische Prüfung

Der Validator prüft ohne Modellaufruf: YAML-Form, Namen, Identitäten, referenzierte Ziele, zulässige Typen, Bereichsgrenzen, Bindungen, Vertragssignaturen, Rekursion, Router-Verknüpfung und erzeugte Ausgaben. Eine `Rule` kann über `check` eine bekannte Go-Prüfung referenzieren. Unbekannte Check-Namen sind Fehler; Text in einer YAML-Datei wird nicht als Programm ausgeführt.

Regeltext und zugehöriger Check werden gemeinsam reviewed. Ein erfolgreicher Check beweist seine kodierte Bedingung; der Text kann weitergehende Anforderungen enthalten. Das Ergebnis unterscheidet daher strukturelle Gültigkeit, Abdeckung und semantische Bewertung.

Ein Review-Kontext enthält die geltenden Regeln, tatsächlich benötigten Texte, transitive konkrete Abhängigkeiten, gebundene Implementierungen und die betroffenen Änderungen. Jede aufgenommene Quelle erhält einen Aufnahmegrund. Die KI beurteilt damit gezielte Fragen wie „Passt die neue Abschlussbedingung noch zum Review-Vertrag?“.

Nicht deklarierte Bedeutungsbezüge bleiben eine Grenze. Bei untypisierten Inhalten und unbekannten Abhängigkeiten wird konservativ ein größerer Bereich geprüft. Eine neue Textdatei wird nicht global geladen; eine neue oder veränderte allgemein geltende Regel kann dagegen viele Ergebnisse ungültig machen. Das Modell muss erforderliche Quellen vollständig erhalten; ein Tokenlimit darf sie nicht still abschneiden.

## Snapshot und parallele Arbeit

Verbindliche Prüfungen lesen einen unveränderlichen Git-Commit mit Konfiguration und allen relevanten Quellen. Der Branch-Name wird einmal aufgelöst. Dateisystembasierte bestehende Checks laufen auf einer isolierten Materialisierung dieses Stands. Der Nachweis nennt den Commit, die Werkzeugversion und die verwendeten Profile.

Uncommittete Änderungen können schnelle vorläufige Hinweise liefern. Für eine verbindliche längere Prüfung veröffentlicht der Autor nach Abschluss zusammengehörender Änderungen einen konsistenten privaten Commit. Während dessen Prüfung kann er in einem getrennten Arbeitsstand weiterarbeiten. Zweimaliges Hashen eines bewegten Ordners garantiert keine atomare Aufnahme einer zusammengehörenden Änderung. Das Cockpit benötigt für diesen Ablauf noch Git.

Bei Integration wird der tatsächlich zusammengeführte Kandidat gegen den aktuellen Zielstand geprüft. Alter und neuer Graph bestimmen betroffene Ergebnisse; auch gelöschte Abhängigkeiten werden berücksichtigt. Ein verschobener Zielbranch erzwingt eine erneute Gültigkeitsentscheidung. Konfliktfrei zusammenführbare Dateien sind nicht automatisch fachlich konsistent. Menschliche Freigaben bleiben an die jeweiligen Repository-Regeln gebunden.

Nachweise werden in YAML außerhalb ihres eigenen Eingabesnapshots gespeichert. Ihre Eingaben umfassen Dateiinhalte, Pfade, relevante Dateimodi, Verzeichnisabfragen, Glob-Treffermengen, fehlende erwartete Dateien, Bindungen, Profil-/Adapter-Digests und externe Quellstände. Deshalb kann auch das Hinzufügen einer zuvor unbekannten Datei einen Nachweis ungültig machen.

Ein Fingerabdruck basiert auf Check-Version und normalisierten Eingabewerten in deterministischer Reihenfolge. Die genaue Bytekodierung ist ein versionierter interner Hashvertrag, kein zusätzliches Autorenformat. Rohes YAML mit beliebiger Schlüsselfolge oder Kommentaren ist keine verlässliche kanonische Serialisierung. Inhaltszeichenfolgen werden nicht semantisch normalisiert; Textänderungen bleiben relevant.

Schnelle Strukturchecks dürfen zunächst vollständig laufen. Der erste große Gewinn ist die Wiederverwendung unveränderter Reviews und die Begrenzung neuer Review-Aufträge. Semantische Nachweise binden außerdem Fragestellung, Promptversion, Modellkennung und tatsächlichen Kontext. Sie gelten nur für diese Prüfung und beweisen keine universelle Widerspruchsfreiheit. Ein beliebiges selbst geschriebenes YAML mit Erfolgsmeldung ist kein vertrauenswürdiger CI-Nachweis.

## Provider und Zusammenarbeit der Repositories

Provider-Adapter übersetzen geprüfte Quellen in die jeweils benötigten Skill-, Agent- und Regeldateien. Einschränkungen eines Providers bleiben sichtbar. Eine verlangte Fähigkeit ohne unterstützte Abbildung führt zu einer Diagnose; sie wird nicht stillschweigend entfernt. Ein deklarierter Agent erhält dadurch keine zusätzlichen realen Rechte.

Das Cockpit verweist zunächst auf Konfyras eigenen Arbeitskontext. Ein späterer Import kann ausdrücklich exportierte Ressourcen aus einem festen Konfyra-Stand verwenden. Ein erzeugtes `markitect.lock.yaml` bindet Repository, Commit, exportierte Ressourcen und Digest. Prüfungen aktualisieren diese Grundlage nicht stillschweigend über „latest“. Inhalte anderer Kunden werden dabei nicht als allgemeines Wissen übernommen.

Dasselbe Go-Werkzeug versteht beide Repository-Profile. Namespace-Namen werden repositoryübergreifend auf Kollision geprüft. Die aktuelle Arbeitsauswahl bestimmt, welche Importe verwendet werden. Eine technische Referenz ist keine Freigabe zur Weitergabe kundengebundener Inhalte.

## Einführung ohne zweite Wahrheitsquelle

Die bestehenden kanonischen Markdown-Dateien werden schrittweise migriert. Für ein migriertes Artefakt wird YAML die Quelle. Eine generierte Markdown-Ansicht kann denselben bisherigen Linkpfad erhalten; sie trägt Quellverweis und Generatorhinweis. Freier Text wird dabei aus `spec.text` erzeugt. Das Tool verhindert, dass alte Markdown-Quelle und neue YAML-Datei gleichzeitig als unabhängig kanonisch gelten.

Die heutigen Checker und Renderer akzeptieren dieses Modell noch nicht. Die Aufnahme von YAML, Router-Abdeckung und die Unterscheidung erzeugter Markdown-Ansichten müssen deshalb in derselben Migration umgesetzt werden. Es gibt pro Ausgabe genau einen zuständigen Renderer. Ein universeller sofortiger Umbau aller Dokumentation ist nicht erforderlich.

Zuerst wird eine für den Pilot verwendbare Markitect-Version gebaut und anhand von Fixtures sowie lesenden Prüfungen fester Repository-Stände verifiziert. Sobald diese Version steht, beginnt die Konfyra-Migration auf einem eigenen Branch: zuerst `author-ai-mechanism` einschließlich Dokumentationsprüfung, anschließend der übrige vereinbarte Umfang. Nach Konfyra-Abnahme folgt die Migration des Cockpits. Der [Markitect-Umsetzungsplan](markitect-implementation-plan.md) besitzt Arbeitspakete, Artefaktliste und die Kriterien für diese Übergänge; eine parallele Fortschrittsliste wird hier nicht gepflegt.

Ein Review-Vertrag zeigt im Konfyra-Pilot, ob die Abstraktion tatsächlich Austauschbarkeit schafft. Zwei Testkonfigurationen verwenden denselben Vertrag mit unterschiedlichen kompatiblen Implementierungen. Reiner Text muss unabhängig davon ohne weitere Konstrukte speicherbar sein. Kubernetes folgt als optionale Erweiterung nach den lokal bewährten Einsätzen.

## Erfolg und offene Entscheidungen

Gemessen werden Strukturprüfzeit, semantische Prüfzeit, Eingabetokens, wiederverwendbare Nachweise und Fehlerabdeckung. Strukturfeedback im Sekundenbereich und deutlich kleinere Review-Kontexte bei lokalen Änderungen sind Ziele; sie sind noch keine gemessenen Zusagen. Änderungen an übergeordneten Anforderungen dürfen große Prüfbereiche auslösen.

Entscheidende Prüffälle sind Referenzfehler, falscher Typ, Kundenüberschreitung, fehlende oder mehrdeutige Bindung, unpassende Vertragssignatur, neu hinzugefügte Regel, entfernter Verweis, während des Reviews veränderte Dateien, manipulierte Provider-Ausgabe und verschobener Integrationsstand. Geänderte Checker oder Profile müssen ebenfalls die passenden Nachweise ungültig machen.

Festgelegt sind Markitect als Name, Go, YAML, einfache Ressourcennamen, Text als eigener Typ und Contracts für gezielte Abstraktion. Offen bleiben der Host des künftigen Markitect-Repositories, die kontrollierte API-Domain, die Release-Verteilung und die Einrichtung von Git/CI im Cockpit. Eine tatsächliche Kubernetes-Erweiterung folgt erst auf das lokal erprobte Modell.

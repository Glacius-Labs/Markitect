# Proposal: Kanonisches Modell mit nominalen Regeln und methodischem Review

Erstellt am 11. Oktober 2026; vollständig konsolidiert am 12. Oktober 2026. Status: beauftragter Konzept- und Umsetzungsentwurf auf Basis des Bibliotheksgesprächs. Dateiname `mandate.yaml` gewählt; `kind: Rule` ist die bestätigte Entwurfsrichtung. Weitere Syntax- und Referenzdetails sind Vorschläge, keine implementierten Produktverträge.

## 1. Auftrag und Maßstab

Die [Leitfrage](system-description-notes.md#leitfrage) lautet: **Wie beschreibt man ein Repository für ein Softwaresystem?** Das kanonische Modell soll das geltende Projekt so beschreiben, dass ein Umsetzer daraus ein passendes Repository realisieren kann, ohne jede Datei vorzugeben oder wesentliche Entscheidungen unbemerkt zu erfinden.

Das Modell enthält Zweck, Umfang, Begriffe, Verhalten, Regeln, technische und organisatorische Festlegungen sowie Realisierungs- und Prüfpflichten. Vollständigkeit ist auf einen erklärten Umfang und Detaillierungsgrad bezogen. Nicht beschriebene Entscheidungen werden geklärt oder ausdrücklich delegiert; ein grüner Strukturcheck beweist keine vollständige Spezifikation.

Die drei Produktblöcke bleiben: kanonisches Modell, realisierendes Repository und abstrakter Apply-Mechanismus. Git-Historie und Work Items sind weiterhin außerhalb der geltenden Systembeschreibung. Ein beobachteter Implementierungsfehler verändert die kanonische Regel nicht automatisch.

Die [aktuelle Bibliotheksdarstellung](library-model-example.md) konkretisiert diese Fassung. Der [Änderungsdurchlauf](library-change-walkthrough.md) zeigt Koordination und Review. Der [Main-Abgleich vom 11. Oktober](canonical-model-main-comparison.md) ist historische, SHA-gebundene Implementierungsevidenz; er wurde heute nicht erneuert.

## 2. Empfohlener Aufbau

**Arbeitsrichtung:** kleiner struktureller Core, begrenztes nominales Projektvokabular, verbindliche Fachprosa, explizite Beziehungen und methodische Reviews. Zusätzliche Fachformalismen werden durch konkreten Bedarf begründet.

| Bestandteil | Zuständigkeit |
| --- | --- |
| Struktureller Core | Definitionen, Identitäten, Schemas, Werttypen und typisierte Referenzen; deterministische Normalisierung und Diagnosen. |
| Projektvokabular | Bedeutung der Definitionen und ihrer Beziehungen; insbesondere Regeln, Verantwortungen, Realisierung und Prüfanschlüsse. |
| Kanonische Projektdefinitionen | Was gilt, welche Entscheidungen frei bleiben und welche Beiträge erwartet werden. |
| Abgeleitete Sichten | Dokumentation, Managerkontext, Graph, Impact-Erklärung und konkreter Dateibestand. |
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
| Fachlicher Beziehungsgraph | Kontextbedarf und Verpflichtungen zwischen Definitionen. | Querbeziehungen verändern keine Elternschaft. |
| Ausführungsablauf | Abgeleitete Aufgaben, Reviews und Integration. | Eine fachliche Dependency ist nicht automatisch ein Ausführungsplan. |

Ein kleiner Bibliotheksausschnitt kann einen Root-Manager besitzen. Untergeordnete Verantwortungen entstehen, wenn begrenzte Aufträge und Bereiche sie rechtfertigen. Übergreifende Interessen brauchen ausdrückliche Mandate oder Beziehungen. Zyklen werden je Relation beurteilt; Parent-Zyklen sind anders zu behandeln als mögliche fachliche Querreferenzen.

### 4.1 Minimaler Beziehungsvertrag

| Feld | Bedeutung | Kandidat für zulässige Ziel-Kinds |
| --- | --- | --- |
| `uses` | Benötigter Kontext. | `Statement`, `Rule` |
| `requires` | Normative Verpflichtung, die ein Vertrag erfüllen oder erhalten muss. | Im ersten Bibliotheksentwurf `Rule` |
| `Artifact.realizes` | Erwarteter Realisierungsbeitrag. | `Statement`, `Rule` |
| `Artifact.checks` | Benanntes ausführbares Prüfverfahren. | `Check` |
| `Check.uses` | Aussagen im Prüfgegenstand. | `Statement`, `Rule` |
| `Manager.parent` | Direkt übergeordnete Verantwortung. | `Manager` |
| `Decision.subject` | Gegenstand einer begründeten Entscheidung. | Kandidat: `Statement`, `Rule` |
| `Decision.actor` | Aufgezeichnete Provenienz. | `Manager` |

Bei `requires` erklärt die Fachprosa zunächst, ob eine Regel Voraussetzung oder erhaltene Invariante ist. `uses` sagt weder „gilt für“ noch etwas über die Zahl konkreter Bücher. Zusätzliche Kanten wie `appliesTo`, `contains` oder `provides` sind Erweiterungskandidaten, keine still vorausgesetzten Pflichtfelder.

Die hier eingeschränkte `requires`-Zielmenge ist ein neuer Vorschlag für diesen Bibliotheksvertrag. Der bisherige Statement-Vertrag konnte auch andere Statement-Verpflichtungen ausdrücken. Bestehende Projekte dürfen deshalb nicht automatisch oder verlustbehaftet umgedeutet werden.

### 4.2 Identität und Referenzsyntax

Identität umfasst API-Version, Kind, Namespace und Name. Umbenennung oder Kind-Wechsel benötigt Referenz- und Evidenzbehandlung. Ein UUID-Zwang wird nicht vorgeschlagen.

Der neue YAML-Kandidat nennt den Zielkind ausdrücklich:

```yaml
requires:
  - kind: Rule
    namespace: library
    name: one-active-loan
```

Die API-Version des Ziels wird im Beispiel aus dem versionierten Referenzvertrag bestimmt: Alle Ziel-Kinds dieser Fassung gehören ausdrücklich zur gleichen `vNext`-Version. Das ist eine vorgeschlagene Versionierungsregel, keine freie Auflösung auf eine zufällig vorhandene Version. Cross-Version-Referenzen sind nicht implizit freigegeben. Jede Property hat eine endliche Zielmenge; „beliebiger Kind“ ist kein Ersatz für Typprüfung. Auch `uses`, Realisierung, Checks und Entscheidungsbezüge brauchen konsistente Auflösung.

**Implementierungsentscheidung offen:** Der bisher geprüfte Core-Vertrag benennt pro Reference-Property genau einen Zielkind. Gemischte Zielmengen verlangen entweder eine eng begrenzte, explizite Referenzunion oder getrennte typisierte Properties. Der gemeinsame YAML-Kandidat bevorzugt die Kind-angereicherte Referenzform. Ob Core und Validator diese ohne neue Primitive tragen können, ist vor der Implementierung an aktuellem Main zu prüfen. Kein untypisiertes Referenzobjekt darf still die bisherigen Diagnosen und Graphkanten umgehen.

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
      library-implementation.yaml
      library-verification.yaml
      library-documentation.yaml
      library-contract-check.yaml
```

Der vollständige Baum und das Manifest stehen im [Bibliotheksbeispiel](library-model-example.md). Eine feinere fachliche Gliederung kann später Bestand, Mitgliedschaft und Ausleihe in eigene Namespacebereiche teilen.

### 6.2 Empfohlene erste Sprachform: benannte Aussagen und nominale Regeln

Der kleinste neue Sprachkandidat trennt `Statement` und `Rule`. Ein Statement beschreibt einen Begriff, Verhalten, Architektur oder Arbeitsweise. Eine Rule beschreibt eine normative Bedingung oder Invariante. Fachlicher Inhalt bleibt zunächst verbindliche Prosa.

Für die übrigen Statement-Rollen bleiben die bisherigen Kategorien im Beispiel vorläufig erhalten. `Rule` benötigt keine zusätzliche Kategorie `rule`. Eigene Kinds `Concept`, `UseCase`, `System` oder `Mandate` sind noch nicht beschlossen. Ihre Notwendigkeit muss aus einem konkreten Zieltyp- oder Feldvertrag folgen.

Nominale Typisierung und Tiefe der Formalisierung sind unabhängig: Ein Rule-Kind kann gezielte Referenzprüfung ermöglichen, während der Regeltext methodisch beurteilt wird.

#### 6.2.1 Feldkandidat für den ersten Vertrag

Diese Arbeitstabelle konkretisiert den Entwurf für P0; sie behauptet kein vorhandenes Schema. Alle Definitionen besitzen `apiVersion`, `kind`, `metadata.name`, `metadata.namespace` und einen lesbaren `purpose`. Der Namespace kann für die Wurzel leer sein. Header und Spec sind geschlossen; unbenannte Zusatzfelder werden nicht still akzeptiert.

| Kind | Spec-Felder des Kandidaten | Pflicht und Begrenzung |
| --- | --- | --- |
| `Statement` | `category`, `description`, `public`, `uses`, `requires` | Kategorie zunächst concept/use-case/architecture/workflow; Beschreibung erforderlich. Uses/Requires sind optionale Referenzlisten. |
| `Rule` | `description`, `public`, `uses` | Beschreibung erforderlich; keine Kategorie rule. Kontextbezüge optional; keine formale Bedingung vorausgesetzt. |
| `Manager` | `parent`, `owns`, `instructions` | Höchstens ein Parent; optionale Pfadliste und lesbare Anweisungen. Roots/Ownerregeln werden getrennt validiert. |
| `Artifact` | `role`, `realizes`, `paths`, `checks`, `required`, `reason` | Rolle implementation/documentation/verification/configuration; optionale Ziel-/Pfad-/Checklisten, Erforderlichkeit und Begründung. |
| `Check` | `command`, `uses`, `limitation` | Nicht leere Liste literal argv; deklarierter Gegenstand und Evidenzgrenze. Ein Befehl ist keine Shell-Prosa. |
| `Decision` | `subject`, `decision`, `reason`, `actor` | Genau ein Subject und Actor sowie lesbare Entscheidung und Begründung. Actor authentifiziert keine Befugnis. |

Für Statement/Rule ist `public` als Boolean vorgesehen; ohne ausdrückliches Public bleibt eine fremde Definition im Kandidaten privat. Referenzen werden gegen die Zielmengen aus Abschnitt 4 geprüft. Die Mengenfelder bestimmen Modellwerte; sie zählen keine Laufzeitinstanzen. Ausführliche Defaults, Kardinalitäten und Normalisierung werden im versionierten P0-Vertrag festgeschrieben. Weitere Felder brauchen einen erklärten Bedarf und ihre Behandlung in allen Sichten.

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

Die zuvor als Alternative B dokumentierte methodische Variante ist die Arbeitsbasis dieser konsolidierten Empfehlung. Sie wird mit der bevorzugten nominalen Rule-Richtung verbunden. Daraus folgt keine Annahme, dass umfangreiche Fachfacetten grundsätzlich unnötig sind; die weitergehende Formalisierung bleibt eine alternative Tiefe des gleichen Modells.

Das [bisherige v1alpha1-Beispiel](library-methodical-alternative.md) bleibt als datierte Vorstufe erhalten. Das [aktuelle vNext-Beispiel](library-model-example.md) zeigt erstmals alle Regeln und Referenzen konsistent in der neuen Entwurfsrichtung. `vNext` und seine Referenzform sind Platzhalter, keine unterstützte Version.

### 6.6 Gewählter Dateiname und bevorzugte Kind-Richtung

Nutzerentscheidung: „Wir nehmen mandate.“ Die Verantwortungsdatei heißt `mandate.yaml`. Der bisherige Modell-Kind `Manager` bleibt im Beispiel erhalten; weder dessen Umbenennung noch die Benennung eines Runtime-Agenten ist dadurch entschieden.

Nutzerpräferenz: „Was category angeht fände ich kind rule interessanter“, anschließend Zustimmung zum Beispiel und Auftrag zur kompletten Konzeptaktualisierung. `kind: Rule` ist deshalb die bestätigte Entwurfsrichtung. Feld- und Referenzdetails bleiben zu konkretisieren. Der semantische Begriff Mandat beschreibt Auftrag, Zuständigkeit, Befugnisse und Grenzen, ohne selbst eine authentifizierte Autorisierung zu beweisen.

## 7. Bibliotheksbeispiel und methodischer Ablauf

Die elf Ausgangsaussagen sind Systemzweck, Titel, Exemplar, Mitglied, Ausleihe, aktive Ausleihe, berechtigtes Mitglied, verfügbares Exemplar, höchstens eine aktive Ausleihe, Ausleihen und Rückgabe. Die drei Bedingungen/Invarianten werden Rules; die übrigen Inhalte bleiben Statements.

Ein Root-Manager ist für den kleinen Ausschnitt ausreichend. Code, Prüfungen und Bedienungsdokumentation sind getrennte Realisierungsbeiträge. Der beschriebene Check ist ein klar markierter, nicht ausgeführter Platzhalter.

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

Der bisher geprüfte Routingstand bildet eine konservative Hülle über `uses`/`requires` in beide Richtungen. Im eng verbundenen Bibliotheksmodell kann diese nahezu alle fachlichen Aussagen erreichen. Change/Context dient der Erklärung und Priorisierung; es verkürzt die relevante Menge nicht automatisch. Ein anderer engerer Algorithmus wäre eine eigene begründete Produktentscheidung.

Breite Artefaktpfade wie `src/Library/` liefern einen Untersuchungsbereich. Sie wählen keine einzelnen Funktionen aus und garantieren keinen kleinen Blast Radius. Welche Dateien verändert werden müssen, ist innerhalb dieses Bereichs zu untersuchen.

Der [Überfälligkeitsdurchlauf](library-change-walkthrough.md) zeigt, wie eine Prosaänderung zunächst eine fehlende Beziehung verbergen kann, wie Review sie ergänzt und welche Prüfungen danach erforderlich werden.

## 8. Umsetzungsplan für den Implementer

### 8.1 Begründung der Designentscheidungen

| Entscheidung | Begründung und Abwägung |
| --- | --- |
| Kleine Struktur, fachliche Prosa | Explizite Zuordnung und Methodik zuerst nutzen; zusätzlichen Pflegeaufwand für Formalisierung an konkretem Nutzen prüfen. |
| Nominaler Rule-Kind | Regeln als referenzierbaren Zieltyp unterscheiden; dies benötigt noch keinen allgemeinen Regelauswerter. |
| Statement vorerst beibehalten | Weitere Rollen nicht ohne konkreten Bedarf zu neuen Kinds machen; ihre Kategorien bleiben zunächst lesbare Einordnung. |
| Mandat von Agentenbindung unterscheiden | Ein stabiler Auftrag kann von unterschiedlichen Agenten umgesetzt werden; Datei, Modell-Kind und Runtime-Identität sind eigene Benennungsentscheidungen. |
| Explizite Beziehungen und konservativer Impact | Koordination nachvollziehbar aus deklarierten Daten ableiten; Prosalinks allein liefern keine geprüften Kanten. |
| Erneuerbarer Dateibestand | Pfadregeln pflegen und konkrete Dateien erfassen; eine zweite manuelle Zuordnungsliste vermeiden. |
| Unabhängige Reviews und begrenzte Checks | Fachbedeutung und tatsächliche Realisierung prüfen; Struktur und Prüferfolg nicht als universelle Korrektheit ausgeben. |

Die erhoffte geringere Aufsicht, zuverlässigere Agentenkoordination und niedrigere Kosten sind weiterhin Hypothesen. Dieser Dokumentationsauftrag misst sie nicht.

### 8.2 Umsetzungsvorbereitung für den Implementer

Lesereihenfolge: dieser Proposal → [aktuelles YAML-Beispiel](library-model-example.md) → [Änderungsdurchlauf](library-change-walkthrough.md) → [historischer Main-Abgleich](canonical-model-main-comparison.md) als Ausgangspunkt für einen neuen Vergleich.

Der Dokumentationsbranch stammt aus einer älteren Main-Basis. Den Implementierungsbranch vom beim Start frisch abgerufenen `origin/main` anlegen; Pfad, Branch, vollständigen SHA und lokale Änderungen erfassen. Aktuelle Beitragshinweise und Modellzonen-Owner prüfen. Die hier dokumentierten Entwürfe gezielt übernehmen, nicht den alten ausgecheckten Produktcode als aktuellen Main verwenden.

| Paket | Ergebnis | Voraussichtlicher Bereich | Abnahmekriterium |
| --- | --- | --- | --- |
| P0 — Ausgangspunkt und Verträge | Main-Delta, Versionierung, Rule-Schema und zulässige Referenzziele gemeinsam fixieren. | Produkt-/Modellvertrag; Core-Ausdrucksbedarf prüfen. | Keine stille alpha-Umdeutung; gemischte Zielmengen sind explizit gelöst. |
| P1 — Vollständiger kleiner Durchstich | Bibliotheksmodell mit Mandat, Statements, Rules, Artefakten und Check-Anschluss laden/validieren. | Schema, Loader und Beispiel; Core nur bei belegter Lücke. | Gültige Definitionen auflösbar; falscher Zielkind, unbekanntes Feld und falscher Namespace mit deterministischer Diagnose. |
| P2 — Sichten und Bedeutungserhalt | Rules und alle neuen Kanten in Report, Kontext, Graph und Digest führen. | Projectmodel-Analyse/Typen und vorhandene Ausgabeanschlüsse. | Zuständige Rollen sehen benötigte Regeln; private Inhalte werden nicht über eine zweite Sichtbarkeitslogik geladen. |
| P3 — Änderungsfolgen | Rule-/Statement-Änderung, entfernte Kanten und gemischte Beziehungen in Impact/Explain behandeln. | Projectmodel-Impact/Explain und feste Host-Bindungen. | Alt/Neu-Wirkung erhalten; unbekannter Umfang bleibt konservativ; kombinierte Änderungen verlieren keine Pflicht. |
| P4 — Realisierung und Review | Rule-Zuordnung, Coverage, Check-Bindungen und dokumentierten Reviewablauf durch vorhandene Ausführung führen. | Artefakte, Projectcoverage/Projectwork und bei Vertragsänderung Projectrun. | Keine neue Regel verschwindet aus Realisierung/Prüfung; Reviewergebnisse nennen Befunde und Grenzen. |
| P5 — Kontrast und Integration | Kleines CLI-Gegenbeispiel, aktuelle Nutzer-/Formatdokumentation und normale Kandidatengates. | Beispiele, Tests und zuständige Dokumente. | Kein Bibliotheks-, Datenbank- oder Mehrmanager-Zwang; Checks gelten für festen Kandidaten. |

P1 bis P4 bilden gemeinsam einen vollständigen Durchstich. Ein nur zugelassener Rule-Kind ist kein fertiges Feature. Die Pakete sind Vorschläge, keine vergebenen Backlog-IDs und keine gestarteten Implementierungsaufträge.

Der breitere Facetten-/Ausdrucksentwurf folgt nur bei konkret begründetem Bedarf. Government, Cockpit, Providerstudien, universelle Sourceanalyse und Releasepublikation sind nicht Teil dieses ersten Durchstichs.

### 8.3 Prüfplan für die spätere Implementierung

Heute sind diese Prüfungen ausschließlich geplant. Bei Umsetzung gelten zusätzlich die dann aktuellen Projektgates.

| Risiko | Gezielter Nachweis |
| --- | --- |
| Nominaler Kind ist nur ein Label | Verweis auf Statement an einem Rule-only-Ziel wird abgewiesen; gültiger Rule-Verweis wird aufgelöst. |
| Gemischte Referenz verliert Bedeutung | Statement/Rule in Kontext, Realisierung, Check und Entscheidung bleiben eindeutig, digest-relevant und im richtigen Scope. |
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
- Beibehaltung oder spätere Ablösung der Kategorien für Nicht-Regel-Statements.
- Regelbedingungen versus Invarianten: zunächst Prosa oder eigener nominaler/Feldvertrag?
- Verbindliche Protokollierung der Reviewpflicht ohne Verwechslung mit Autorisierung.
- Angemessene Zuordnungsgranularität und Breite des Impact.
- Kriterien, die spätere Fachformalismen oder Strictness-Profile rechtfertigen.
- Konkrete noch offene Fachentscheidungen des Bibliotheksbeispiels.

## 10. Herkunft und Status

- Produktgespräch: `01a121f1-b948-7050-ae5d-9921b99db9c0`; Konsolidierung ausdrücklich am 12. Oktober 2026 beauftragt.
- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Eingang dieser Aktualisierung: `07e03215c968302361f5acedc8e69bd573ac95be`; beim Beginn sauber.
- Ursprüngliche Branchbasis: `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Letzter gesonderter Main-Abgleich: `32afd1a57e715130d056e98e433721c6e7a10278` vom 11. Oktober; heute nicht als aktuell neu verifiziert.
- Lokale Änderungen dieses Auftrags: ausschließlich Konzept-, Beispiel-, Ablauf- und Navigationsdokumentation. Keine Produktimplementierung, Modellkompilierung, Tests, Studien oder Agentenausführung am Bibliotheksprojekt.

Die frühere Facettenempfehlung wird durch diesen zusammenhängenden Arbeitsentwurf abgelöst. Datierte Vorarbeiten bleiben mit Statushinweis erhalten. Kanonische Produktdokumente, Konzeptregister und Backlog werden durch diesen Entwurf nicht automatisch geändert.

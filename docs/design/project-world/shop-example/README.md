# Shop: konkreter Modell- und Änderungsentwurf

Dieses Verzeichnis enthält einen vollständigen **Designbeleg**, keine lauffähige Shop-Anwendung und kein vom heutigen Markitect akzeptiertes Projektformat. `designVersion: project-world/0.1` kennzeichnet die vorgeschlagene Syntax. Die Dateien können als YAML gelesen und ihre expliziten Beziehungen überprüft werden; daraus entsteht kein Nachweis einer bereits implementierten Agentenlaufzeit.

Dieser Beleg bewahrt die erste Entwurfsiteration. Seine privaten YAML-Dateien tragen beim Übertragen auf die technische Hauptlinie die Endung `.yaml.example`, damit sie unter der bestehenden Dokumentations-Area nicht als aktuelle Produktressourcen geparst werden. Der lokale Belegprüfer liest sie unverändert als YAML. Die [neueste Gliederungs- und Bedienplanung](../conceptual-modules.md) verortet alle Markitect-eigenen Dateien unter `.markitect/`, verbindet Namensraum und Verantwortung in fachlichen Slices und empfiehlt `manager.yaml`. Die Belegdateien und ihr Prüfer sind noch nicht auf dieses Zielformat migriert.

Der [Walkthrough](../shop-walkthrough.md) erklärt Dateistruktur, Modell, Alltag und den Start der Umsetzung. Das [Laufzeitprotokoll](../operating-protocol.md) beschreibt Nachrichten, Kandidaten und Wiederaufnahme.

## Dateien

- [Projektmanifest](project/markitect.yaml.example): explizite Modellauswahl und Projektverantwortung.
- [Organisation](project/model/organization.yaml.example): fünf Bereiche, jeweils mit eigenem Manager-Agenten.
- [Module und Namespaces](project/model/modules.yaml.example): jeweils verantwortete Einheiten mit getrennten Bedeutungen.
- [Bestellungen](project/model/sales.yaml.example), [Bestand](project/model/inventory.yaml.example) und [Zusammenspiel](project/model/commerce.yaml.example): gemeinsame Fachsprache, Regeln und Use Cases.
- [Engineering](project/model/engineering.yaml.example): Architektur, Entscheidungsspielraum, Prozess und Workflow.
- [Realisierungen](project/model/realization.yaml.example): explizite Zuordnung zu geplanten Dateien und Prüfungen.
- [Runtimekonfiguration](project/markitect.runtime.yaml.example): Agentenrollen, Kontextauswahl und Ausführungsgrenzen außerhalb der fachlichen Bedeutung.
- [Vorher-Zustand](before/model/sales.yaml.example), [Vorher-Zustand des Zusammenspiels](before/model/commerce.yaml.example) und [vorherige Realisierungszuordnung](before/model/realization.yaml.example): ersetzen für den Baselinevergleich die gleichnamigen Zielmodelldateien.
- [Änderungsauftrag](changes/add-cancellation.md): von vorhandener Bestellerstellung zur zusätzlichen Stornierung.

`project/` beschreibt den vollständigen vorgeschlagenen Zielstand einschließlich Stornierung. Die Baseline benutzt dasselbe Manifest und dieselben übrigen Dateien, ersetzt aber `model/sales.yaml.example`, `model/commerce.yaml.example` und `model/realization.yaml.example` durch die drei Dateien unter `before/`. Diese Vergleichskonvention gehört ausschließlich zu diesem Designbeleg; sie ist keine geplante Overlayfunktion des Produkts.

Die in `realization.yaml.example` benannten C#-, Test-, Dokumentations- und Betriebsdateien sind gewünschte bzw. im Beispiel vorausgesetzte Realisierungen. Sie liegen nicht als implementierte Anwendung in diesem Verzeichnis. Die tatsächlichen Repositorypfade zeigt der Walkthrough. Unveränderte vorhandene Artefakte werden im beschriebenen Ablauf erhalten.

## Modellkonventionen dieses Entwurfs

Jeder Gegenstand hat `kind`, stabile `id`, `owner`, `area` und `purpose`. `owner` referenziert einen Manager; `area` bezeichnet den organisatorischen Ort. Die eine Zuständigkeit ist dadurch zweifach navigierbar, aber nicht doppelt besetzt. Ein Manager kann zugleich mehrere Einheiten verantworten. Die vorgeschlagenen Kurz-IDs müssen bei Umsetzung ausdrücklich auf die vollständige versionierte Core-Identität abgebildet werden; sie ersetzen diese nicht stillschweigend.

Die Dateien enthalten `resources`-Listen. Diese Bündelung hält das kleine Modell übersichtlich; Dateiname und Verzeichnis erzeugen keine fachliche Abhängigkeit. `uses`, `rules`, `module`, `namespace`, `parent`, `manager`, `manages`, `realizes`, `checks`, `policy`, `process` und `workflow` sind ausdrücklich deklarierte Referenzfelder. Ihre Wirkung ist unterschiedlich: Fachliche Verwendung beeinflusst Kontext/Impact, Elternbeziehungen steuern Managementpfade, Checks liefern Prüfpflichten. Kein allgemeiner Graphpfad wird automatisch zum Ausführungsauftrag.

Die vier Modellmodule sind lokale Quellen mit `revision: workspace`: Ihre tatsächliche Version ergibt sich aus dem gebundenen Quellstand und Inhaltsdigest. Wiederverwendbare veröffentlichte Pakete müssten separat unveränderlich versioniert und gepinnt werden. Die zwei Modellstände ändern somit keine angeblich immutable Paketversion. Namespace-IDs und Namenspräfixe sind über das Feld `namespace` ausdrücklich verbunden; gleiche Schreibweise wird nicht vorausgesetzt.

Die natürliche Sprache in `meaning`, `condition`, `outcome` und `expectations` erklärt gewünschtes Verhalten. Sie ist kein ausführbares Prädikat. Die Beispiel-Checks sind separate geplante Nachweise. Technische Quellcode-Symbole werden nicht aus Prosa inferiert.

`requiresRealization` benennt ausdrücklich Pflichten, deren Umsetzung bzw. Prüfung für den Use Case benötigt wird. Über `realizes` findet der Host die zuständigen Artefaktgruppen und Checks. Der verantwortliche Manager entscheidet, ob vorhandene Dateien samt zulässiger Evidenz genügen, neue Prüfung erforderlich ist oder konkrete Umsetzung fehlt. `uses` allein startet keinen Writer.

`managedRoots` bezeichnet zulässige neue Pfadbereiche und `knownPaths` ausdrücklich zugeordnete Einzelpfade außerhalb oder innerhalb dieser Bereiche. Der Host muss beide Grenzen bei konkreter Writer-Zuteilung beachten; die YAML-Deklaration allein gewährt keinen Prozesszugriff. Eine Definition kann in einem Modul gepflegt werden und einen anderen kanonischen Verantwortlichen haben. Ihre expliziten `owner`-/`area`-Beziehungen routen die Arbeit; im Beispiel integriert Handel die Use Cases im Bestellnamespace. Keine Realisierung in einem fremden Bereich erfolgt ohne ausdrückliche beteiligte Zuständigkeit und koordinierenden Vorfahren.

## Lokale Belegprüfung

[verify_design.py](verify_design.py) prüft ausschließlich diesen privaten Designbeleg: YAML-Eindeutigkeit, Referenzen, einzelne Manager je Bereich, verantwortete Module/Namespaces, Bereichsbaum, Kontexttrennung, eindeutige deklarierte Artefaktpfade und den Vorher-/Nachher-Unterschied. PyYAML wird dafür benötigt; es wird keine neue Produktabhängigkeit eingeführt.

```powershell
python docs/design/project-world/shop-example/verify_design.py
```

Die Prüfung startet keine Agenten und führt keine Shop-Tests aus. Sie beweist weder fachliche Richtigkeit noch die Realisierung der vorgeschlagenen Syntax durch Markitect.

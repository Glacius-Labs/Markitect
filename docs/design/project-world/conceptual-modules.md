# Fachliche Module und Vertical Slices

Status: nächste Entwurfsiteration aus der direkten Nutzerrückmeldung vom 8. Oktober 2026. Die Nutzervorgaben sind festgehalten; „Modul“ und die konkrete Syntax sind Arbeitsvorschläge. Die bestehende Shop-YAML bleibt ein Beleg des ersten Entwurfs und ist noch nicht auf diesen Zuschnitt umgebaut. Keine implementierte Produktfunktion wird behauptet.

## Präzisierung des Ziels

Der Nutzer verwirft `area` als öffentlichen Namen. Das Modell soll nach fachlichen Konzepten und zusammengehörigem Verhalten gegliedert werden, entsprechend Vertical Slices. Eine horizontale Ablage nach `rules/`, `concepts/` und `use-cases/` als hauptsächliche Struktur verteilt zusammengehöriges Wissen auf mehrere Stellen und ist nicht die gewünschte Zielgliederung.

Die Ordnerstruktur darf eine erklärte semantische Bedeutung erhalten: Sie trägt Namespacing, Zugehörigkeit und Verantwortung. Bereichsübergreifende Funktionen sollen durch ausdrückliche Abhängigkeiten, öffentliche Verträge und verantwortete Komposition gelöst werden, vergleichbar mit gut strukturiertem Code. Es entsteht keine zusätzliche Ablage für jede technisch übergreifende Beziehung.

## Arbeitsvorschlag: Modul

Ein Modul ist eine zusammenhängende, benannte und verantwortete Einheit des Projektmodells. Es besitzt Zweck, Namensraum, öffentliche Verträge, Inhalte und gegebenenfalls Untermodule. Die eigene Managerrolle wird durch einen spezialisierten AI-Agenten ausgeübt. Der Ordner bildet das Modul ab, sein Pfad trägt den sichtbaren Namensraum.

Modul, Namensraum und Managementverantwortung sind damit Facetten derselben primären Einheit. Für das normale lokale Projekt werden keine drei parallelen Register derselben Gliederung gepflegt. Ein wiederverwendbares versioniertes Paket ist eine Lieferform für solche Inhalte und benötigt weiterhin einen expliziten Maintainer und eine verantwortete Adoption. Der heutige technische Module-Vertrag wird damit nicht stillschweigend umgedeutet; die Übergangsplanung muss ihn ausdrücklich zuordnen.

Der Name „Modul“ ist eine Empfehlung zur Diskussion, kein bereits bestätigter endgültiger Produktbegriff. Er beschreibt sowohl fachliche Zusammengehörigkeit als auch rekursive Komposition, ohne eine bestimmte Organisationstiefe vorauszusetzen.

## Beispiel einer fachlichen Gliederung

```text
model/
  module.yaml                         Shop / Geschäftsführung
  commerce/
    module.yaml                       Handel / Integrationsmanager
    sales/
      module.yaml                     Sales-Manager
      orders/
        module.yaml                   Orders-Manager
        order.yaml                    Begriff und Lebenszyklus
        create-order.yaml             Erstellung mit Regeln und Prüffällen
        cancel-order.yaml             Stornierung mit Regeln und Prüffällen
    inventory/
      module.yaml                     Inventory-Manager
      reservations/
        reservation.yaml              Bedeutung der Reservierung
        reserve.yaml                  Reservierungsverhalten
        release.yaml                  Freigabe und Wiederholungssicherheit
```

Die Dateien eines kleinen Slice können sein Konzept, Verhalten, Regeln und prüfbare Erwartungen gemeinsam ausdrücken. Die Typisierung bleibt im YAML vorhanden; sie bestimmt nicht die oberste Ablagestruktur. Größere Slices können mehrere Dateien zusammenhalten, ohne ihre Zusammengehörigkeit zu verlieren.

Ein ausdrücklich deklariertes Untermodul erhält einen eigenen Manager-Agenten. Ein bloßer interner Sortierordner erbt eindeutig die Verantwortung seines nächstgelegenen Moduls und erzeugt keine zusätzliche Managerrolle. So lässt sich die Managementtiefe nach tatsächlicher Verantwortung wählen. Jeder Modellgegenstand bleibt eindeutig verantwortet.

Die genaue Namensauflösung ist zu spezifizieren: Welche Pfadsegmente bilden den Namespace, welche sind rein interne Ablage und wie wird eine Verschiebung nachvollziehbar migriert? Deklarierte Modulgrenzen und eindeutige Konventionen ersetzen heuristische Deutung beliebiger Ordnernamen.

## Funktionen über Modulgrenzen

`cancel-order` bleibt beim verantwortlichen Order-Modul und deklariert, dass es den öffentlichen Vertrag `commerce/inventory/reservations/release` benötigt. Inventory besitzt die Bedeutung und zugesicherten Eigenschaften der Freigabe. Orders beschreibt, warum und wann es diese benötigt. Das sind zwei unterschiedliche Aussagen mit je einem kanonischen Besitzer.

Der gemeinsame Handelsmanager koordiniert betroffene Kinder, entscheidet gemeinsame Schnittstellenkonflikte und verantwortet die Integration. Eine fachliche Funktion muss dafür nicht aus ihrem Modul in einen globalen Cross-Cutting-Ordner verschoben werden. Der fachliche Besitzer des Ablaufs und der übergeordnete Integrator behalten ihre jeweiligen Aufgaben.

Öffentliche Verträge liefern zugesicherte Eingaben, Ergebnisse, Fehlerverhalten und relevante Regeln. Ein Manager bekommt den benötigten Vertrag, nicht den gesamten internen Modell- oder Gesprächskontext eines Nachbarmoduls. Direkte Nutzung fremder interner Gegenstände muss ausdrücklich erlaubt oder als Verletzung der gewählten Grenze diagnostiziert werden.

Wirklich gemeinsame Bedeutung wird einmal an einem bewusst gewählten Besitzer definiert und importiert. Querschnittsregeln wie Zugriffsschutz können einen expliziten Geltungsbereich über mehrere Module besitzen. Das rechtfertigt weder eine Kopie pro Modul noch eine unspezifische Sammlung beliebiger Shared-Inhalte.

## Folgen für den nächsten Planungsstand

1. Den öffentlichen Namen wählen; Arbeitsvorschlag ist Modul.
2. Shop-Modell nach Orders, CreateOrder, CancelOrder und Reservations gliedern; zugehörige Regeln und Prüferwartungen zusammenhalten.
3. Parallele Area-/Namespace-/lokale Modulregister durch eine eindeutige Moduldeklaration und nachvollziehbar abgeleitete Werte ersetzen.
4. Öffentliche Verträge, zulässige Imports, interne Sichtbarkeit und Integrationszuständigkeit konkret darstellen.
5. Pfad-/Identitätsänderungen und daraus folgende Kontext-, Impact- und Evidenzänderungen ausdrücklich spezifizieren.

Der Compiler prüft deklarierte Struktur, Referenzen und gewählte Grenzen. Ob Implementierung, Kommentare und Dokumentation tatsächlich zur akzeptierten Bedeutung passen, bleibt eine zusätzliche Prüfpflicht. Die fachliche Gliederung verbessert Kontext und Änderungsnachverfolgung; sie beweist diese Konformität nicht allein.

# Fachliche Module und Vertical Slices

Status: nächste Entwurfsiteration aus der direkten Nutzerrückmeldung vom 8. Oktober 2026. Die Nutzervorgaben zu Vertical Slices, `.markitect/` und modellzentrierter Arbeit sind festgehalten; „Modul“, `manager.yaml` und die konkrete Syntax sind Arbeitsvorschläge. Die bestehende Shop-YAML bleibt ein Beleg des ersten Entwurfs und ist noch nicht auf diesen Zuschnitt umgebaut. Keine implementierte Produktfunktion wird behauptet.

## Präzisierung des Ziels

Der Nutzer verwirft `area` als öffentlichen Namen. Das Modell soll nach fachlichen Konzepten und zusammengehörigem Verhalten gegliedert werden, entsprechend Vertical Slices. Eine horizontale Ablage nach `rules/`, `concepts/` und `use-cases/` als hauptsächliche Struktur verteilt zusammengehöriges Wissen auf mehrere Stellen und ist nicht die gewünschte Zielgliederung.

Die Ordnerstruktur darf eine erklärte semantische Bedeutung erhalten: Sie trägt Namespacing, Zugehörigkeit und Verantwortung. Bereichsübergreifende Funktionen sollen durch ausdrückliche Abhängigkeiten, öffentliche Verträge und verantwortete Komposition gelöst werden, vergleichbar mit gut strukturiertem Code. Es entsteht keine zusätzliche Ablage für jede technisch übergreifende Beziehung.

## Arbeitsvorschlag: Modul

Ein Modul ist eine zusammenhängende, benannte und verantwortete Einheit des Projektmodells. Es besitzt Zweck, Namensraum, öffentliche Verträge, Inhalte und gegebenenfalls Untermodule. Die eigene Managerrolle wird durch einen spezialisierten AI-Agenten ausgeübt. Der Ordner bildet das Modul ab, sein Pfad trägt den sichtbaren Namensraum.

Modul, Namensraum und Managementverantwortung sind damit Facetten derselben primären Einheit. Für das normale lokale Projekt werden keine drei parallelen Register derselben Gliederung gepflegt. Ein wiederverwendbares versioniertes Paket ist eine Lieferform für solche Inhalte und benötigt weiterhin einen expliziten Maintainer und eine verantwortete Adoption. Der heutige technische Module-Vertrag wird damit nicht stillschweigend umgedeutet; die Übergangsplanung muss ihn ausdrücklich zuordnen.

Der Name „Modul“ ist eine Empfehlung zur Diskussion, kein bereits bestätigter endgültiger Produktbegriff. Er beschreibt sowohl fachliche Zusammengehörigkeit als auch rekursive Komposition, ohne eine bestimmte Organisationstiefe vorauszusetzen.

## Dateiname und Zuständigkeit

Der Nutzer schlägt `manager.yaml` oder `agent.yaml` vor. Empfohlen wird `manager.yaml` als Deklaration einer verantworteten Einheit: Auftrag, öffentliche Verträge, delegierte Befugnisse, relevante Vorgaben und untergeordnete Verantwortungen. Die eigene Managerrolle wird weiterhin durch einen spezialisierten AI-Agenten ausgeübt. Der Dateiname bezeichnet ihre Managementfunktion; konkrete Modellanbieter, Invocation und Laufzeitgrenzen gehören in die Laufzeitkonfiguration. Es wird kein zweites Register derselben Zuständigkeit daneben gepflegt.

`agent.yaml` bleibt eine mögliche Benennung, stellt aber die ausführende Technik stärker in den Vordergrund. Die endgültige Wahl ist offen. Nicht jede YAML-Datei ist eine Agentendefinition: Begriffe, Verhalten und Erwartungen bleiben eigenständige Modellinhalte ihres Slice.

## Beispiel einer fachlichen Gliederung

```text
.markitect/
  project.yaml                        Projekteinstieg und Modellauswahl
  runtime.yaml                        Ausführung der Managerrollen
  model/
    manager.yaml                      Shop / Geschäftsführung
    commerce/
      manager.yaml                    Handel / Integrationsmanager
      sales/
        manager.yaml                  Sales-Manager
        orders/
          manager.yaml                Orders-Manager
          order.yaml                  Begriff und Lebenszyklus
          create-order.yaml           Erstellung mit Regeln und Prüffällen
          cancel-order.yaml           Stornierung mit Regeln und Prüffällen
      inventory/
        manager.yaml                  Inventory-Manager
        reservations/                 interner Slice unter Inventory
          reservation.yaml            Bedeutung der Reservierung
          reserve.yaml                Reservierungsverhalten
          release.yaml                Freigabe und Wiederholungssicherheit
    engineering/
      manager.yaml                    technische Verantwortung
      delivery/
        workflow.yaml                 Build, Prüfung und Auslieferung
  drafts/                             noch unverbindliche Modellideen
  views/                              erzeugte lesbare Modellsichten
  runs/                               Laufaufträge, Berichte und Nachweise
  cache/                              wiederherstellbare lokale Daten
src/                                  Implementierung der Anwendung
tests/                                ausführbare Prüfungen
docs/                                 Dokumentation des Produkts
ops/                                  tatsächliche Betriebsdateien
```

Die Dateien eines kleinen Slice können sein Konzept, Verhalten, Regeln und prüfbare Erwartungen gemeinsam ausdrücken. Die Typisierung bleibt im YAML vorhanden; sie bestimmt nicht die oberste Ablagestruktur. Größere Slices können mehrere Dateien zusammenhalten, ohne ihre Zusammengehörigkeit zu verlieren.

Ein ausdrücklich deklariertes Untermodul erhält einen eigenen Manager-Agenten. Ein bloßer interner Sortierordner erbt eindeutig die Verantwortung seines nächstgelegenen Moduls und erzeugt keine zusätzliche Managerrolle. So lässt sich die Managementtiefe nach tatsächlicher Verantwortung wählen. Jeder Modellgegenstand bleibt eindeutig verantwortet.

Die genaue Namensauflösung ist zu spezifizieren: Welche Pfadsegmente bilden den Namespace, welche sind rein interne Ablage und wie wird eine Verschiebung nachvollziehbar migriert? Deklarierte Modulgrenzen und eindeutige Konventionen ersetzen heuristische Deutung beliebiger Ordnernamen.

## Markitect im Hintergrund, Spezifikation im Vordergrund

Alle Markitect-eigenen Modelle, Konfigurationen, Entwürfe, erzeugten Sichten und Arbeitsdaten liegen unter `.markitect/`. Die konkrete Anwendung behält ihre normalen Code-, Test-, Dokumentations- und Betriebsdateien. Diese sind Ergebnisse bzw. Gegenstände der Umsetzung. Ihre fachliche Zuordnung wird im Modell beschrieben.

Die versteckte Ablage bleibt auditierbar und versionierbar. Akzeptiertes Modell und erforderliche Projektkonfiguration gehören in Git; `drafts/` kann bewusst versionierte Exploration enthalten. Cache und gewöhnliche lokale Laufdaten werden standardmäßig ignoriert. Nachweise für Übergaben oder Abnahmen werden gezielt und mit gebundenen Eingabeständen aufbewahrt. Zugangsdaten werden nicht Bestandteil versionierter Laufzeitkonfiguration. Für erzeugte Sichten ist eine explizite Export-/Versionierungspolitik nötig; sie bleiben abgeleitete Ausgaben.

Der normale Nutzer liest und bearbeitet kein YAML. Er spricht mit einem AI-Agenten und liest eine daraus erzeugte Spezifikation: Fachbegriffe, Verhalten, Regeln, Beispiele, offene Entscheidungen und Zuständigkeiten. Die Dateien sind das interne, strukturierte Speicherformat. Die Bedienung und Navigation zeigen Begriffe und Zusammenhänge statt vorausgesetztem Wissen über versteckte Pfade.

Beispielauftrag: „Eine bestätigte Order darf bis zum Versand storniert werden. Dabei wird ihr reservierter Bestand genau einmal freigegeben. Passe das Modell an und setze es um.“

1. Der Agent klärt tatsächlich fehlende fachliche Entscheidungen und verwendet vorhandene Definitionen von Order, Versand und Reservierung.
2. Er verändert die betroffenen Modellinhalte und zeigt eine verständliche fachliche Differenz einschließlich neuer Regeln, Beispielen und Auswirkungen. Der zugrunde liegende YAML-Diff bleibt bei Bedarf einsehbar.
3. Der Compiler prüft deklarierte Struktur und Referenzen, bestimmt den expliziten Änderungsbezug und macht fehlende oder ungeklärte Abdeckung sichtbar.
4. Innerhalb des bereits erteilten Auftrags wird der gültige Modellstand angenommen und die Umsetzung daran gebunden. Ein Auftrag nur zur Planung startet keine Umsetzung. Gewöhnliche interne Zustandswechsel erzeugen keine zusätzliche pauschale Freigabeschleife.
5. Manager delegieren betroffene Arbeit, integrieren die Ergebnisse und liefern Prüfungen und Berichte. Eine notwendige Änderung der fachlichen Vorgaben fließt als nachvollziehbare Modelländerung zurück.
6. Die lesbare Spezifikation und der Umsetzungsstatus werden aus dem tatsächlich finalen Modell-/Artefaktstand aktualisiert. Modellversion und Prüfstand bleiben erkennbar; eine veraltete Sicht gilt nicht als aktueller Nachweis.

Damit fühlt sich Markitect nach Spec Driven Development an: Hauptarbeit ist, die gewünschte Welt präzise zu beschreiben und weiterzuentwickeln. Das ist das vom Nutzer gemeinte „neue Programmieren“. Es umfasst auch gezielte Architektur- und Prozessentscheidungen, wenn sie für das Projekt verbindlich sein sollen. Innerhalb offener Umsetzungsspielräume entscheiden die verantwortlichen Agenten selbst. Das Modell muss deshalb weder jeden Quellcodeausdruck vorwegnehmen noch sämtliche Implementierungsdetails doppelt speichern.

Lesbare Dokumentation erhält ihre Aussagen aus dem kanonischen Modell. Eine Änderung über diese Sicht oder das Gespräch muss auf das Modell zurückgeführt und erneut geprüft werden; direktes Überschreiben einer erzeugten Datei schafft keine zweite Wahrheit. Bewusst ausgenommene handgeschriebene Produktdokumentation hat einen erklärten Besitzer und wird bei relevanten Änderungen in den Abgleich einbezogen. Vollständigkeit wird als offene oder belegte Abdeckung sichtbar, nicht durch die bloße Existenz eines YAML-Eintrags behauptet.

Der Compiler beweist deklarierte strukturelle Eigenschaften. Ob das System die Stornierung tatsächlich korrekt ausführt, braucht weiterhin passende ausführbare Prüfungen und zusätzliche semantische Beurteilung. Der Nutzer sieht fachliche Erwartungen und ihren nachgewiesenen Umsetzungsstand gemeinsam.

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
6. Modell, Laufzeitkonfiguration und interne Ausgaben vollständig unter `.markitect/` verorten; versionierte Quellen und lokale Arbeitsdaten ausdrücklich unterscheiden.
7. Gespräch, lesbare Spezifikation und fachliche Änderungsansicht als normalen Produkteinstieg entwerfen; YAML bleibt einsehbares Speicherformat.

Der Compiler prüft deklarierte Struktur, Referenzen und gewählte Grenzen. Ob Implementierung, Kommentare und Dokumentation tatsächlich zur akzeptierten Bedeutung passen, bleibt eine zusätzliche Prüfpflicht. Die fachliche Gliederung verbessert Kontext und Änderungsnachverfolgung; sie beweist diese Konformität nicht allein.

# Fachliche Module und Vertical Slices

Die [Nutzerpräzisierung vom 9. Oktober 2026](model-first-user-workflow.md) ergänzt diesen Entwurf um den durchgängigen Benutzerablauf und vollständige Dateiabdeckung. Dort wird „Projektion“ wieder als fachlicher Begriff für die Realisierungen des kanonischen Modells verwendet; ältere abweichende Benennungsabsichten sind ersetzt. Für tatsächlich implementierte Syntax gilt weiterhin der [Umsetzungsvertrag](delivery-contract.md).

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

## Modell und Repositorydateien zuordnen

Die bisher unter „Projektionen“ gebündelte Verbindung zu konkreten Dateien bleibt erforderlich. Im Zielmodell wird sie als **Dateizuordnung** ausgedrückt: Welcher Manager verantwortet eine Datei, welche Modellinhalte werden darin umgesetzt, beschrieben oder geprüft und welche Fähigkeiten werden für ihre Bearbeitung gebraucht? Dateizuordnung ist zunächst ein Vertragsvorschlag, kein vorhandenes neues Produktfeature.

Der [Projektvertrag](project-contract.md) präzisiert die dauerhafte Speicherung dieser Beziehungen, den daraus erzeugten Dateiindex und die Anforderungen an Architektur, Arbeitsweise und erwartete Artefakte. Kanonische Zuordnung und abgeleiteter Index erhalten dort getrennte Verträge.

Die fachliche Gliederung bleibt maßgeblich. Orders kann Backend, Frontend, Tests und Dokumentation gemeinsam verantworten. Diese technischen Rollen erzwingen keine neuen Manager oder getrennten Modellhierarchien. Benötigt ein Projekt tatsächlich eine eigene Frontendverantwortung, kann es diese ausdrücklich delegieren und die Integration beim gemeinsamen Manager belassen.

### Verantwortung und Bezug getrennt ausdrücken

Zwei Angaben beantworten unterschiedliche Fragen und dürfen nicht verwechselt werden:

- **Dateiverantwortung:** Die gemeinsame Managerdeklaration benennt die verantworteten Repositorypfade bzw. Pfadmuster. Das ergibt eine prüfbare Zuordnung zu einem Modul. Ein Elternmodul kann Teilbäume ausdrücklich an Kinder delegieren; für eine konkrete Datei bleibt genau ein zuständiger Besitzer auflösbar. Mehrdeutige überlappende Zuordnungen sind ein Fehler, keine implizite Vorrangregel nach dem längsten Pfad.
- **Fachlicher Dateibezug:** Ein Modellinhalt referenziert die Dateien, die ihn implementieren, dokumentieren oder prüfen. Mehrere Modellinhalte dürfen dieselbe Datei benötigen; ein Modellinhalt darf mehrere Dateien über mehrere Technologien benötigen. Diese Beziehung überträgt weder Ownership noch Schreibbefugnis.

Eine Zuordnung verbindet deshalb die **Identität eines Modellgegenstands** mit einem Repositorypfad und einer Rolle. Die YAML-Datei ist sein Speicherort. Eine spätere Aufteilung mehrerer Definitionen auf verschiedene YAML-Dateien soll die fachliche Zuordnung nicht unnötig ändern. Die Rückwärtsansicht „Welche Dateien gehören zu dieser Modelldatei?“ wird aus den darin enthaltenen Definitionen abgeleitet.

Vorgeschlagene Rollen sind `implementation`, `documentation`, `verification` und `configuration`. Backend und Frontend können zusätzliche erklärende Kategorien von Implementierungen sein. Rolle und Technologie sind keine Verantwortungshierarchie. Eine Datei kann mehrere Rollen erfüllen; Dokumentationsbezug allein beweist keine implementierte Regel.

Die Zuordnung wird beim fachlichen Besitzer des Modellinhalts gepflegt, beispielsweise im `cancel-order.yaml` oder in einer lokalen, ausdrücklich angebundenen Begleitdatei desselben Slice. Ein globales manuell gepflegtes Zweitregister wird nicht benötigt. Der Compiler kann daraus einen Repositoryindex und beide Navigationsrichtungen erzeugen.

### Beispiel: Order stornieren

Die folgenden Repositorypfade sind illustrative zukünftige Artefakte, keine vorhandene Shop-Implementierung:

| Modellinhalt | Repositorydatei | Rolle | Dateiverantwortung |
|---|---|---|---|
| `commerce/sales/orders/cancel-order` | `src/backend/Orders/CancelOrder.cs` | Implementierung des Ablaufs | Orders |
| `commerce/sales/orders/cancel-order` | `src/frontend/orders/CancelOrderButton.tsx` | Implementierung der Bedienung | Orders |
| `commerce/sales/orders/cancel-order` | `tests/orders/CancelOrderTests.cs` | Ausführbare Verifikation | Orders |
| `commerce/sales/orders/cancel-order` | `docs/orders/cancellation.md` | Produktdokumentation | Orders |
| `commerce/inventory/reservations/release` | `src/backend/Inventory/ReleaseReservation.cs` | Implementierung der Freigabe | Inventory |
| `commerce/sales/orders/cancel-order` | `src/backend/Inventory/ReleaseReservation.cs` | Über benötigten Freigabevertrag abgeleiteter Implementierungsbezug | Inventory |

Die letzte Zeile ist eine abgeleitete Ansicht aus dem benötigten Freigabevertrag und dessen eigener Dateizuordnung, kein zweites manuell gepflegtes Inventar in Orders. Sie verbindet Orders mit einer fremdverantworteten Datei, erlaubt aber keinen unkoordinierten Schreibzugriff. Der fachliche Vertrag bleibt die öffentliche Freigabefunktion. Orders erhält zunächst diesen Vertrag und den relevanten Prüfstand; die Dateiverknüpfung zwingt fremden internen Quellcode nicht in jeden Managerkontext. Inventory bearbeitet seinen Kandidaten, der gemeinsame Manager integriert den Gesamtfall.

### Pflege, Ausführung und Prüfung

Am Anfang kann ein neues Konzept gültig sein, obwohl noch keine Implementierungsdatei existiert. Bekannte Zuordnungen, geplante Realisierungen und am Kandidaten tatsächlich beobachtete Dateien bleiben unterscheidbar. Ein unbekannter zukünftiger Dateiname muss nicht vor der Delegation erfunden werden. Der Manager erhält Ziel, verantworteten Umfang und benötigte Rollen; der Arbeiter meldet die konkreten entstandenen Dateien zurück.

Der Lauf führt den tatsächlichen Änderungsbestand. Er gleicht neue, verschobene, entfernte und veränderte Dateien mit Verantwortung und fachlichen Zuordnungen ab. Neue Zuordnungen werden innerhalb des erteilten Auftrags als nachvollziehbare Modelländerung validiert und angenommen. Finale Prüfungen binden den dadurch aktualisierten Modell- und Artefaktstand. Eine beobachtete Datei schafft für sich genommen keine akzeptierte Fachabsicht; bei unverändertem Soll bleibt eine Reparatur möglich.

Für die Ausführung wird aus den bestehenden Verantwortungen und dem konkreten Auftrag genau ein Writer pro veränderbarer Datei bestimmt. Pfadmuster beschreiben Zuständigkeit; sie ersetzen keine abgegrenzte Laufberechtigung. Gemeinsame Dateien wie OpenAPI-Dokumente oder zentrale Navigation haben einen benannten Besitzer bzw. Integrator, auch wenn mehrere Slices dazu beitragen. Direkt erzeugte Dateien besitzen zusätzlich eine erklärte Erzeugungsquelle und einen einzigen verantwortlichen Erzeugungsweg.

Der Compiler bzw. Host kann an einem festen Repositorysnapshot prüfen, ob Zuordnungen auf gültige Modellinhalte verweisen, Pfade auflösbar sind und Ownership eindeutig ist. Bei Pfadmustern werden Muster und tatsächlich aufgelöste Dateimenge gebunden; neu passende oder verschwundene Dateien dürfen nicht durch einen veralteten Index übersehen werden. Er kann fehlende erwartete Dateien, ungeklärte Dateien und mehrere Ansprüche melden. Explizite Fremdverwaltung und Ausschlüsse bleiben möglich.

Eine Zuordnung ist zunächst eine Behauptung über Zweck und Zusammenhang. Sie beweist nicht, dass `CancelOrder.cs` die Regel korrekt implementiert oder ein referenzierter Test sie tatsächlich prüft. Checkauswahl, reale Ergebnisse und semantische Beurteilung liefern dafür zusätzliche Nachweise. Unbekannte Abdeckung wird sichtbar; sie darf Impact nicht auf eine angeblich vollständige Dateiliste verengen. Auch neue Pflichten für unveränderte Dateien können frische Prüfung oder Anpassung verlangen.

Bei einer Modelländerung ergeben sich so betroffene fachliche Verträge und Dateibezüge, zuständige Manager und benötigte Prüfungen. Bei einer direkten Codeänderung zeigt derselbe Index umgekehrt, welche Vorgaben erneut abzugleichen sind. Abhängigkeiten, Dateizuordnung, Ausführungsrechte und Nachweise bleiben in ihrer Wirkung ausdrücklich unterschieden.

## Folgen für den nächsten Planungsstand

1. Den öffentlichen Namen wählen; Arbeitsvorschlag ist Modul.
2. Shop-Modell nach Orders, CreateOrder, CancelOrder und Reservations gliedern; zugehörige Regeln und Prüferwartungen zusammenhalten.
3. Parallele Area-/Namespace-/lokale Modulregister durch eine eindeutige Moduldeklaration und nachvollziehbar abgeleitete Werte ersetzen.
4. Öffentliche Verträge, zulässige Imports, interne Sichtbarkeit und Integrationszuständigkeit konkret darstellen.
5. Pfad-/Identitätsänderungen und daraus folgende Kontext-, Impact- und Evidenzänderungen ausdrücklich spezifizieren.
6. Modell, Laufzeitkonfiguration und interne Ausgaben vollständig unter `.markitect/` verorten; versionierte Quellen und lokale Arbeitsdaten ausdrücklich unterscheiden.
7. Gespräch, lesbare Spezifikation und fachliche Änderungsansicht als normalen Produkteinstieg entwerfen; YAML bleibt einsehbares Speicherformat.
8. Dateiverantwortung in der Managerdeklaration und fachliche Dateibezüge im jeweiligen Slice konkretisieren; einen ableitbaren Index statt mehrfach gepflegter Zuordnungen schaffen.

Der Compiler prüft deklarierte Struktur, Referenzen und gewählte Grenzen. Ob Implementierung, Kommentare und Dokumentation tatsächlich zur akzeptierten Bedeutung passen, bleibt eine zusätzliche Prüfpflicht. Die fachliche Gliederung verbessert Kontext und Änderungsnachverfolgung; sie beweist diese Konformität nicht allein.

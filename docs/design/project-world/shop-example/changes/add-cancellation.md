# Änderungsauftrag: Order vor Versand stornieren

Status: Beispiel eines Vorschlags; kein tatsächlicher Produktauftrag.

## Ausgangspunkt

Der Shop kann Orders erstellen und dafür Bestand reservieren. Eine allgemeine Reservierungsfreigabe ist im Bestandsvokabular bereits definiert. Ein Stornierungsablauf ist noch nicht als aktiver Use Case beschrieben. Das Vorher-Modell verwendet die drei Ersatzdateien unter `before/model/`; alle anderen Modelldateien entsprechen dem Beispielprojekt.

## Gewünschte Änderung

Eine bestätigte, noch nicht versandte Order soll storniert werden können. Alle zugehörigen aktiven Reservierungen werden zusammen mit dem Zustandswechsel in derselben Datenbanktransaktion freigegeben. Wiederholte Stornierung bleibt ohne zusätzliche Wirkung. Bereits versandte Orders werden abgelehnt. Die tatsächlich vorhandene Artikelmenge wird nicht erhöht; die verfügbare Menge wird wieder frei.

## Modelländerungen

- `model/sales.yaml.example`: Zustand `sales/cancelled`, Konzept `sales/cancellation`, Regel `sales/cancel-before-shipped` und Use Case `sales/cancel-order` ergänzen; Order verweist zusätzlich auf ihren neuen Zustand.
- `model/commerce.yaml.example`: die übergreifende Regel `commerce/cancellation-releases-reservation` ergänzen.
- `model/realization.yaml.example`: neuen Use Case und seine Regeln ausdrücklich bestehenden Artefaktgruppen und Prüfungen zuordnen. Neue konkrete Dateinamen werden damit noch nicht vorgegeben.
- `model/inventory.yaml.example`: bestehende Definitionen von Reservierung und Freigabe wiederverwenden. Die neue Verwendung führt trotzdem zu einer konkreten Umsetzungs-/Prüfpflicht für den Bestandsbereich.

Die Namen dienen der Verständigung; die ausdrücklich deklarierten Referenzen schaffen die maschinellen Beziehungen. Der Inventarmanager muss keine neue fachliche Definition erfinden, nur weil seine Umsetzung angepasst oder geprüft werden muss.

## Erwartete Arbeit nach dem Start

Der Handelsmanager delegiert den Order-Zustandswechsel und die lokalen Tests an Bestellungen. Lagerbestand stellt Freigabe und Wiederholungssicherheit bereit oder weist deren vorhandene korrekte Umsetzung nach. Handel ergänzt den vollständigen Use Case und seine Transaktions-/Fehlertests. Der Elternmanager koordiniert insbesondere Fehlerverhalten und Schnittstellen.

Mögliche neue Dateien sind `src/Shop/Commerce/CancelOrder.cs` und `tests/Shop.Tests/Commerce/CancelOrderTests.cs`. Diese konkreten neuen Namen werden im Arbeitsplan ausgewählt und erst nach tatsächlicher Umsetzung in die Realisierungszuordnung übernommen. Vorab ist nur ihr zulässiger verantworteter Bereich festgelegt. Bestehende `Orders.cs`, `Reservations.cs` und Dokumentation können angepasst werden.

Der Plattformmanager erhält keinen Implementierungsauftrag, solange Architektur, Build, Datenbankbereitstellung und Deploymentkonfiguration unverändert bleiben. Der Host darf bestehende Gesamtprüfungen trotzdem ausführen.

## Fachliche Erfolgskriterien

1. Zulässige Stornierung ändert Order und Reservierung zusammen.
2. Wiederholung erhöht den verfügbaren Bestand nicht erneut.
3. Stornierung nach Versand wird ohne Teiländerung abgelehnt.
4. Ein Fehler zwischen den beiden Änderungen lässt die gesamte Transaktion unverändert.
5. Andere Orders und deren Reservierungen bleiben unverändert.
6. Bestehende Bestellerstellung funktioniert weiterhin.

Die Kriterien werden vor Umsetzung gebunden und danach auf dem integrierten Kandidaten geprüft. Die Formulierung allein ist noch keine ausgeführte Prüfung.

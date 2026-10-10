# Ein kleines Shop-Projekt mit Markitect

Historischer Entwurf. Den aktuellen Dateibaum, das implementierte Modell und die Bedienung zeigen das [ausführbare Shop-Projekt](../../../examples/project-world/README.md) und der [Arbeitsablauf](../../project-workflow.md).

Status des folgenden Belegs: ausgearbeiteter Zielentwurf. Die folgenden Dateien, Bedienaktionen und Agentenabläufe sind ein Vorschlag für das künftige Produkt. Das [konkrete YAML-Beispiel](shop-example/README.md) ist ein lokal prüfbarer Designbeleg, keine implementierte Shop-Anwendung und kein aktueller CLI-Vertrag. Dieser Walkthrough beantwortet die fünf Benutzerfragen; das [Laufzeitprotokoll](operating-protocol.md) besitzt die detaillierten Ausführungsverträge.

Fortschreibung: Die nachfolgende Dateistruktur und die getrennten Area-/Namespace-/Moduldefinitionen zeigen die erste Iteration. Der [neueste Zielstand](conceptual-modules.md) führt Markitect-eigene Dateien unter `.markitect/` zusammen, gliedert nach fachlichen Slices und empfiehlt `manager.yaml` als gemeinsame Deklaration. Der Nutzer arbeitet über Gespräch und lesbare Spezifikation. Dieser Walkthrough ist noch nicht vollständig darauf migriert.

## 1. Welches Softwaresystem bauen wir?

`Shop` ist eine kleine Anwendung mit HTTP-API und einer relationalen Datenbank. Kunden können eine Order anlegen; dafür wird vorhandener Bestand reserviert. Das System wird um Stornierung vor Versand erweitert. Zahlung, Retoure, mehrere Lager und verteilte Dienste gehören nicht zum Beispiel.

Die fachlichen Module Bestellungen und Bestand leben in derselben Anwendung. Ein gemeinsamer Use Case kann beide in derselben Datenbanktransaktion ändern. Diese Architektur ist eine explizite Beispielentscheidung. Managementbereiche müssen keine Microservices sein.

Der Nutzer gibt Ziele und wichtige Vorbehalte vor. Darunter arbeitet diese Organisation:

```text
Nutzer — oberste Instanz
└─ Geschäftsführung / ceo                         eigener AI-Agent
   ├─ Handel / commerce                          eigener AI-Agent
   │  ├─ Bestellungen / orders                    eigener AI-Agent
   │  └─ Lagerbestand / inventory                 eigener AI-Agent
   └─ Plattform / platform                       eigener AI-Agent
```

Jeder Manager hat seinen eigenen funktionsspezifischen Kontext. Ausführende Arbeiter und unabhängige Prüfer werden für konkrete Aufträge zugeordnet. Die Managerhierarchie kann mit demselben Vertrag weiter verfeinert werden; für dieses kleine System genügt diese Tiefe.

## 2. Wie sieht die Dateistruktur aus?

So könnte das reale Zielrepository aussehen. Die Anwendungsdateien sind hier ein konkreter Layoutvorschlag; in diesem Planungsworktree liegen nur die Modelldateien und Belege dazu:

```text
shop/
├─ markitect.yaml                       Projekt, Besitzer, explizite Modelleingaben
├─ markitect.runtime.yaml               Agentenbetrieb und Laufgrenzen
├─ model/
│  ├─ organization.yaml                 Bereiche und Managerrollen
│  ├─ modules.yaml                      verantwortete Module und Namespaces
│  ├─ sales.yaml                        Order, Zustände, Regeln, Use Cases
│  ├─ inventory.yaml                    Bestand, Reservierung, Freigabe
│  ├─ commerce.yaml                     Regeln über beide Fachbereiche
│  ├─ engineering.yaml                  Architektur, Befugnisse, Prozess, Workflow
│  └─ realization.yaml                  Zuordnung zu Dateien und Prüfungen
├─ notes/
│  └─ cancellation-ideas.md             freie Ideen; nicht automatisch verbindlich
├─ src/Shop/
│  ├─ Shop.csproj
│  ├─ Program.cs
│  ├─ ShopDbContext.cs
│  ├─ Orders/
│  │  ├─ Orders.cs
│  │  └─ Endpoints.cs
│  ├─ Inventory/
│  │  └─ Reservations.cs
│  └─ Commerce/
│     ├─ CreateOrder.cs
│     └─ CancelOrder.cs                 entsteht durch die Beispieländerung
├─ tests/Shop.Tests/
│  ├─ Shop.Tests.csproj
│  ├─ Orders/OrderTests.cs
│  ├─ Inventory/ReservationTests.cs
│  └─ Commerce/
│     ├─ CreateOrderTests.cs
│     └─ CancelOrderTests.cs            entsteht durch die Beispieländerung
├─ docs/
│  ├─ orders.md
│  ├─ inventory.md
│  └─ commerce.md
├─ ops/
│  ├─ Dockerfile
│  └─ compose.yaml
├─ .github/workflows/ci.yaml
└─ .markitect/                          lokaler Arbeitsstand; keine Fachdefinitionen
   ├─ cache/                            wiederherstellbare technische Daten
   └─ runs/<run-id>/                     gebundene Aufträge, Berichte und Nachweise
```

Die drei wesentlichen Dinge sind damit räumlich erkennbar: `model/` enthält das akzeptierte Projektverständnis, `src/`, `tests/`, `docs/` und `ops/` seine konkreten Realisierungen, `.markitect/runs/` die Vorgänge und Nachweise. Freie Notizen bleiben unter `notes/` möglich. Modell und Umsetzung werden gemeinsam in Git versioniert. Runtimeartefakte können separat aufbewahrt oder als ausgewählte Evidenz exportiert werden; sie aktivieren keine neue Sollregel.

`model/` ist ein verständlicher Default, keine Pflicht für jedes Repository. Das Manifest wählt genau die relevanten YAML-Dateien. Ein größeres Projekt kann `sales.yaml` in mehrere Dateien aufteilen, ohne dadurch neue fachliche Identitäten oder Managementebenen zu erzeugen. Ein Ordnername oder Markdown-Link erzeugt keine maschinelle Beziehung.

`markitect.runtime.yaml` enthält Rollenbindungen, Kontextregeln, Werkzeuge und Laufgrenzen. Konkrete Anbieter-/Modellbindungen bleiben außerhalb der fachlichen Definitionen und werden pro Lauf festgehalten; der Designbeleg verlangt keine konkrete Anbieterwahl. Secrets gehören nicht in die versionierte Konfiguration.

## 3. Wie sieht das Modell aus?

### Organisation, Module und Namespaces

Der [Organisationsbeleg](shop-example/project/model/organization.yaml.example) enthält fünf Bereiche mit `parent` und genau einem `manager`. Jede Managerrolle ist auf einen eigenen AI-Agenten gebunden. Zwei Rollen können dasselbe AI-Modell verwenden, teilen aber keinen Gesprächskontext.

[Module und Namespaces](shop-example/project/model/modules.yaml.example) sind eigene verantwortete Gegenstände. Im Beispiel besitzt Bestellungen das Modul `sales-domain` und den Namespace `sales`. Handel verantwortet die übergreifenden Regeln. Der Use Case darf sich fachlich im Namespace `sales` befinden und trotzdem von Handel integriert werden. Namespacepflege, kanonische Ownership einer Definition und Ausführung sind ausdrückliche Beziehungen; sie werden nicht aus gleicher Schreibweise abgeleitet.

Das kleine Projekt verwendet lokale Modellmodule. Ihre Revision wird durch Quelle und Inhaltsdigest gebunden. Ein später veröffentlichtes wiederverwendbares Modul bekäme eine unveränderliche Paketversion, einen Maintainer und eine ausdrücklich verantwortete lokale Adoption. Die Verwaltung solcher Pakete ist nicht nötig, um mit dem kleinen Projekt zu beginnen.

### Gemeinsame Fachsprache

| Identität | Bedeutung | Verantwortlicher |
|---|---|---|
| `sales/order` | Kaufauftrag mit stabiler Identität, Positionen und Zustand | Orders-Manager |
| `sales/confirmed` | Angenommen und noch nicht versandt | Orders-Manager |
| `sales/shipped` | An den Versand übergeben | Orders-Manager |
| `sales/cancelled` | Wird nicht mehr ausgeführt | Orders-Manager |
| `sales/cancellation` | Zulässiger Wechsel von bestätigt zu storniert | Orders-Manager |
| `inventory/stock` | Tatsächlicher Bestand und daraus verfügbare Menge | Inventory-Manager |
| `inventory/reservation` | Aktive exklusive Zuordnung einer Menge zu einer Order | Inventory-Manager |
| `inventory/release` | Reservierung inaktiv machen, ohne tatsächlichen Bestand zu erhöhen | Inventory-Manager |

Der Compiler kann Identitäten, Typen und ausdrückliche Verwendungen prüfen. Die natürliche Bedeutung wird durch die Definition und die zuständigen Agenten verstanden. Prüfen lässt sie sich nur im Umfang geeigneter zusätzlicher Checks und unabhängiger Urteile.

### Regeln und Use Cases

Der [Zielstand der Bestellungen](shop-example/project/model/sales.yaml.example) enthält lokale Regeln. Die [übergreifende Regel](shop-example/project/model/commerce.yaml.example) beschreibt das Zusammenspiel. Ein Auszug aus der vorgeschlagenen Syntax lautet:

```yaml
kind: Rule
id: commerce/cancellation-releases-reservation
owner: commerce-manager
area: commerce
purpose: Nach Stornierung keinen Bestand für die aufgehobene Order blockieren.
uses:
  - sales/cancellation
  - sales/cancel-before-shipped
  - inventory/reservation
  - inventory/release
condition: Eine zulässige Stornierung hebt alle aktiven Reservierungen der Order auf.
outcome: Zustandswechsel und Freigabe erfolgen in derselben Datenbanktransaktion.
architecture: engineering/shop-monolith
```

Die Regel referenziert definierte Bedeutungen statt nur Wörter zu wiederholen. Der Use Case `sales/cancel-order` referenziert diese Regel und die lokale Stornierungsregel. Die lokale Regel verbietet Stornierung nach Versand und definiert die Wiederholung einer bereits erfolgten Stornierung.

Die Texte sind lesbare kanonische Vorgaben, kein eingebautes ausführbares Bedingungssystem. Konkrete Tests untersuchen erfolgreiche Stornierung, Wiederholung, Ablehnung nach Versand und Rollback bei Fehler zwischen den Teiländerungen. Zusätzlich benennt der Use Case mit `requiresRealization` ausdrücklich seine benötigten Pflichten. `realizes` ordnet diese vorhandenen Artefaktgruppen und Checks zu; Kontextverwendung allein startet keine Schreibarbeit.

### Architektur, Arbeitsweise und Realisierung

[Engineering](shop-example/project/model/engineering.yaml.example) beschreibt die gemeinsame Transaktion, lokale Entscheidungsfreiheit, den nächsten befugten Vorfahren bei Konflikten und wenige dem Nutzer vorbehaltene Grundsatzentscheidungen. Der Prozess legt den Lebenszyklus einer Änderung fest; der Workflow benennt seine konkreten Schritte. Beide besitzen einen Verantwortlichen.

[Realisierungen](shop-example/project/model/realization.yaml.example) ordnen Modellpflichten existierenden Dateien, verantworteten Schreibbereichen und Prüfungen zu. Bekannte Pfade werden ausdrücklich benannt. Neue Detailpfade können innerhalb des delegierten Bereichs erst im Arbeitsplan ausgewählt und nach tatsächlicher Umsetzung eingetragen werden. Nicht jede Methode und jeder Dateiname muss im Sollmodell vorgegeben sein.

## 4. Wie arbeitet man damit bei einer Änderung?

Der Benutzer kann das Modell direkt bearbeiten oder einem Agenten die gewünschte Änderung beschreiben. Ein möglicher Auftrag ist:

> Eine bestätigte Order soll vor Versand stornierbar sein. Gib dabei den reservierten Bestand frei. Passe das Modell entsprechend an und setze die Änderung anschließend um.

Dieser eine Auftrag kann Modellpflege und anschließende Umsetzung autorisieren. Die folgenden internen Schritte benötigen keine zweite pauschale Nutzerfreigabe, solange sie innerhalb der vorhandenen Befugnisse liegen. Wer zunächst nur diskutieren will, kann ausdrücklich einen Entwurf oder eine Folgenabschätzung verlangen.

Für das Beispiel ist der [Änderungsvorschlag](shop-example/changes/add-cancellation.md) konkret gespeichert. Der Vorher-Stand kann Orders anlegen und reservieren; er enthält noch keinen Stornierungs-Use-Case. Seine Bestandsbegriffe sind bereits definiert. Die Änderung ergänzt fünf Definitionen und erweitert den Order-Zustandsumfang:

- `sales/cancelled`, `sales/cancellation`, `sales/cancel-before-shipped`, `sales/cancel-order` werden ergänzt.
- `commerce/cancellation-releases-reservation` wird ergänzt.
- `sales/order` verweist zusätzlich auf den neuen Zustand.
- Die Realisierungs- und Prüfzuordnung wird ausdrücklich um den neuen Use Case und seine Regeln erweitert; neue Dateinamen bleiben Implementierungsentscheidungen im zulässigen Bereich.

Die Vorschläge werden zunächst strukturell und anhand der relevanten Regeln geprüft. Eine Folgenansicht zeigt bekannte betroffene Begriffe, Pflichten, Bereiche, Dateien, Prüfungen und vorhandene Lücken. Der zuständige Manager entscheidet fachliche Details innerhalb seiner Befugnis. Danach wird genau dieser Modellstand angenommen und versioniert.

Eine Notiz ist dadurch noch kein akzeptierter Modellstand. Auch das Speichern gültigen YAMLs ist nicht automatisch ein Ausführungsstart. Annahme bezeichnet den Sollzustand, nicht die Behauptung, dass das Repository ihn bereits erfüllt. Das System soll den Zwischenstand klar zeigen: **„Modell angenommen, Umsetzung offen.“**

Für kleine Änderungen braucht man weder neue Module noch zusätzliche Manager. Eine Definition wird an ihrem kanonischen Ort geändert. Der Compiler berechnet bekannte Folgen; die Hierarchie bearbeitet die daraus entstandene Arbeit. Routineentscheidungen bleiben bei der zuständigen Ebene.

## 5. Was passiert beim Start der Umsetzung?

### Schritt 1: Einen gebundenen Durchlauf eröffnen

Die gewünschte Bedienaktion heißt sinngemäß **„Umsetzung starten“**. Das ist ein Designbegriff, kein heute vorhandener CLI-Befehl. Ein zuvor erteilter Umsetzungsauftrag kann diesen Schritt bereits autorisieren.

Der Host bindet das angenommene Modell und seinen Digest, den genauen Repository-Basiscommit, Konfiguration und Tools, Entscheidungsrahmen, Prüfungen und Laufgrenzen. Alle Manageraufträge beziehen sich auf diese Eingaben. Eine spätere freie Dateiänderung verändert den laufenden Auftrag nicht rückwirkend.

### Schritt 2: Folgen und Managementpfade bestimmen

Die Auswertung betrachtet alte und neue deklarierte Beziehungen. Neue Pflichten brauchen auch ihre unveränderten Voraussetzungen: Die Definition `inventory/release` wurde nicht geändert, muss für den neuen Use Case aber realisiert oder als bereits korrekt nachgewiesen werden. „Nicht editiert“ bedeutet daher nicht „keine Arbeit möglich“.

Die neue `requiresRealization`-Beziehung verlangt insbesondere `inventory/release`. Dessen `realizes`-Zuordnung benennt Bestandsartefakte und lokale Checks. Der Host stellt vorhandene Realisierung und passende Evidenz gegenüber; der Inventarmanager entscheidet daraus konkrete Anpassung oder Nachweis. Es gibt keinen pauschalen Schreibauftrag allein wegen einer `uses`-Kante.

Die Pfade verlaufen in diesem Fall über `shop → commerce → orders` und `shop → commerce → inventory`. Die bekannten Modelländerungen betreffen vor allem Bestellungen und Handel; der neue Gesamtvertrag benötigt außerdem Bestand. Bestehende Bestellerstellung wird als mögliche Regression mitgeprüft.

Ein Kontextbezug zu einer unveränderten Architektur schafft keinen automatischen Plattformauftrag. Ownership-, Namespace- und Kontextbeziehungen haben nicht dieselbe Wirkung wie eine fachliche Umsetzungs- oder Prüfpflicht. Fehlende oder unbekannte Eingaben werden konservativ als Lücke behandelt.

### Schritt 3: Manager-Agenten mit getrennten Kontexten starten

| Agent | Kennt für diesen Auftrag | Delegiert bzw. entscheidet |
|---|---|---|
| Geschäftsführung | Ziel, relevante Projektregeln, Handelsauftrag und Gesamtstatus | Gibt Handel das Ergebnisziel; verantwortet das Projektergebnis |
| Handel | Stornierungsvertrag, direkte Schnittstellen beider Kinder, Integrationskriterien | Verteilt Teilaufträge, entscheidet gemeinsame Verträge und integriert |
| Bestellungen | Order-Zustände, lokale Stornierungsregel, benötigter Freigabevertrag | Lässt Zustandswechsel, API-Verhalten und lokale Tests anpassen |
| Lagerbestand | Reservierung, Freigabe, benötigte Order-Identität und Transaktionszusage | Lässt Wiederholungssicherheit und lokale Tests umsetzen bzw. nachweisen |
| Plattform | Kein neuer Arbeitskontext erforderlich | Bleibt inaktiv, solange Build-/Betriebsverträge unverändert sind |

Jeder Manager ist ein eigener AI-Agent. Ein Manager erfährt nicht, welche internen Dokumente oder Unteragenten seine Kinder nutzen. Die Manager tauschen Aufträge, Ergebniszusagen, konkrete Fragen und Berichte aus. Bestehende Gesamtprüfungen können ausgeführt werden, ohne deshalb einen Plattformmanager zu aktivieren.

### Schritt 4: Konkrete Arbeit unten erledigen

Die Manager vergeben konkrete Arbeitsaufträge an ausführende Agenten. Im Beispiel dürfen Bestellungen und Lagerbestand parallel ihre lokalen Dateien ändern. Handel koordiniert den bereichsübergreifenden Use Case und seine Integrationstests.

Jeder ausführende Auftrag arbeitet in einem isolierten Kandidaten auf der benannten Basis. Jeder geänderte Pfad hat genau einen Writer. Ein gemeinsamer Pfad wird einem Writer zugewiesen oder nacheinander koordiniert. Ein Git-Merge allein ist kein fachlicher Kompatibilitätsnachweis.

Die Arbeiter liefern tatsächliche Dateien und gebundene Berichte zurück. Ein unabhängiger Prüfer untersucht den jeweiligen Kandidaten. Die C#-Detailgestaltung bleibt innerhalb der vorgegebenen Regeln frei.

### Schritt 5: Ergebnisse auf jeder Ebene integrieren

Bestellungen und Lagerbestand berichten an Handel. Der Handelsmanager führt ihre konkreten Kandidaten zusammen und lässt den vollständigen Ablauf prüfen:

- Zustandswechsel und Freigabe werden gemeinsam committed oder gemeinsam zurückgerollt.
- Wiederholung verändert verfügbare Mengen nicht ein zweites Mal.
- Versandte Orders bleiben unverändert; die Stornierung wird abgelehnt.
- Andere Reservierungen und die bestehende Bestellerstellung bleiben korrekt.

Wenn Bestellungen eine sofortige Freigabe erwartet und Lagerbestand nur eine spätere Freigabe liefert, entscheidet Handel innerhalb seiner Befugnis die notwendige Anpassung und delegiert sie erneut. Handel benötigt dazu Ergebnis- und Schnittstellenzusagen, nicht die kompletten Kindkontexte. Reicht seine Befugnis nicht, eskaliert die konkrete Frage zur Geschäftsführung. Nur wichtige vorbehaltene oder dort nicht entscheidbare Fragen erreichen den Nutzer.

Handel berichtet eine integrierte Kandidatenrevision nach oben. Die Geschäftsführung verantwortet ihre eigenen Gesamtpflichten und die finale Prüfung auf genau diesem Kandidaten. Kind-PASSes werden nicht einfach als Gesamt-PASS addiert.

### Schritt 6: Übernehmen und verständlich berichten

Der vorgesehene Übernahmeprozess prüft die aktuelle Kandidatenbindung und übernimmt den erfolgreichen Stand in den vereinbarten lokalen Arbeitsbranch. Eine veränderte Zielbasis erzwingt erneuten Abgleich und erforderliche Prüfungen. Push, Merge in einen geschützten Hauptbranch, Release oder Deployment sind getrennte Vorgänge mit eigener Autorisierung.

Neue tatsächlich entstandene Pfade stehen zunächst im beobachteten Laufbericht. Soll ein neuer Pfad dauerhaft in `knownPaths` aufgenommen werden, ist das eine ausdrücklich autorisierte Metadatenänderung am Modell: Der zuständige Manager schlägt sie vor, der Host validiert und berechnet Impact erneut, der befugte Manager nimmt sie an, und die betroffenen Prüfaufträge werden an den finalen Modell-/Artefaktstand gebunden und erneut ausgeführt. Erst danach erfolgt die Ergebnisübernahme. Unveränderte Geschäftspflichten werden nicht stillschweigend neu formuliert; alte Check-PASSes werden nicht auf die neue Bindung übertragen.

Ein hypothetischer Benutzerbericht könnte lauten:

```text
Umgesetzt: Order vor Versand stornieren und reservierten Bestand freigeben.
Integriert: Bestellungen + Lagerbestand, gemeinsame Transaktionsprüfung.
Erhalten: Bestehende Bestellerstellung, Build und Betriebskonfiguration.
Prüfung: vereinbarte lokale und Gesamtfälle am finalen Kandidaten bestanden.
Offen: keine Entscheidung im beauftragten Umfang.
Deployment: nicht beauftragt.
```

Dieser Bericht ist illustrativ, kein Ergebnis einer hier ausgeführten Shop-Implementierung. Bei Fehler, fehlendem Check oder ungelöstem Konflikt würde der Bericht stattdessen den offenen Umfang benennen. Kein erfolgreicher Abschluss wird allein aus einer Agentenantwort abgeleitet.

## 6. Was geschieht in den anderen Fällen?

- **Nur Ideen sammeln:** Notizen bleiben frei; es startet kein Umsetzungsauftrag.
- **Modell ist angenommen, aber noch nicht umgesetzt:** Das neue Soll bleibt sichtbar offen, bis ein Auftrag startet.
- **Umsetzung war bereits korrekt:** Vorhandene Dateien werden erhalten; benötigte frische Nachweise werden beschafft. Kein künstlicher Rewrite.
- **Bug bei unverändertem Modell:** Reparatur auf demselben Soll, ohne erfundene neue Geschäftsregel.
- **Neue Erkenntnis erfordert eine Modelländerung:** Zuständiger Manager entscheidet oder eskaliert; betroffene Aufgaben werden auf die neue akzeptierte Basis neu geplant, unabhängige Arbeit bleibt möglich.
- **Abbruch oder Unterbrechung:** Aufgaben, Kandidaten, Entscheidungen und Berichte bleiben gebunden gespeichert; jeder Manager kann mit seinem eigenen Rollenstand fortgesetzt werden.

## 7. Was davon liegt jetzt vor?

Vorhanden sind der ausgearbeitete Plan, vollständige vorgeschlagene YAML-Modelle, zwei nachvollziehbare Modellstände, ein konkreter Änderungsvorschlag und ein lokal ausführbarer Belegprüfer. Die Umsetzung benötigt weiterhin die in der [Implementierungsplanung](implementation-plan.md) beschriebenen Produktverträge und Runtimefähigkeiten. Hier wurden keine Shop-Anwendung, Managerläufe oder Deploymentnachweise erstellt.

# Sprachgeschichte vor der normativen v1

**Historisch und nicht normativ.** Diese Ablage bewahrt die frühere Gesprächsherleitung, Varianten, Beispiele und damaligen Prüfstände (16. Oktober 2026). Aussagen wie „offen“, „gewählt“ oder „als Nächstes“ beziehen sich auf den jeweiligen Gesprächsstand. Sie sind keine zusätzlichen v1-Regeln.

Die alleinige normative Quelle ist die [Sprachspezifikation v1](language-contract.md). Die noch nicht gewählten [Verfahrensvorschläge](procedure-proposals.md) sind getrennt abgelegt. Aktuelle ausführbare Evidenz steht beim [Exchange-Beispiel](exchange-example/README.md).

Die nachfolgenden Abschnitte 1–10 und 12 stammen aus dem Sprachvertrag unmittelbar vor der redaktionellen Konsolidierung; Abschnitt 11 wurde separat als Verfahrensvorschlag erhalten. Die inhaltlichen Gesprächsabschnitte bleiben erhalten, Navigationsverweise werden auf ihre heutigen Eigentümer gerichtet. SHA-256 der früheren vollständigen UTF-8-Textfassung: `2f62267a7d359e5f4e395c13e614447f7b960effd2cf822aa4e2c864bc6c2370`.

---

## 1. Zweck und Umfang

Das kanonische Modell beschreibt die beabsichtigte Projektwelt: was gelten und entstehen soll. Verbindliche Prosa und typisierte Angaben tragen diese Beschreibung gemeinsam. Die tatsächlich entstandenen Dateien, laufenden Vorgänge und ausgeführten Nachweise sind davon zu unterscheiden.

Der Core stellt die allgemeinen Sprachmittel vollständig bereit. Projekte können ihr Vokabular selbst definieren. Bibliotheken liefern wiederverwendbare Definitionen; sie verleihen importierten Kinds keine zusätzlichen Sprachfähigkeiten oder Ausführungsrechte.

System, Container, Component, Command, Exchange, Rule oder andere Fachbegriffe sind mögliche Inhalte eines Vokabulars. Sie sind keine privilegierten Core-Kinds. Das Modell benötigt weder ein eingebautes Project noch eine vorgegebene Softwarearchitektur-Taxonomie.

## 2. Eine Grundform: Definition

Typverträge, Propertyverträge und konkrete Projektgegenstände sind Definitionen mit derselben Grundform:

| Feld | Inhalt |
| --- | --- |
| imports, optional | In dieser Definitionsdatei sichtbare fremde Pakete; ohne Angabe keine fremden Namen. |
| id | Stabile lokale Identität innerhalb des Pakets. |
| kind | Referenz auf die Typdefinition, deren Vertrag diese Definition erfüllt. |
| body | Verbindliche Beschreibung ihrer Bedeutung und Festlegungen. |
| spec | Objekt mit den laut Kind erlaubten strukturierten Werten und Beziehungen. |
| title, optional | Lesbare Bezeichnung; kein Referenzziel und kein Ersatz für body. |

id, kind und body sind nichtleere Angaben. spec kann leer sein, muss aber dem Typvertrag entsprechen. Unbekannte Felder sind Fehler; für zusätzliche Projektangaben werden Properties am jeweiligen Kind deklariert. Die Beispiele verwenden YAML als Autorenform.

Jede Definition steht vollständig in genau einer eigenen YAML-Datei. Das gilt gleichermaßen für Kinds, Properties und konkrete Gegenstände sowie für die Definitionen des festen Core-Pakets. Jede Definitionsdatei enthält dieselbe Hülle mit id, kind, body, spec und optional imports sowie title. Mehrere Definitionen in einer Datei, eingebettete Definitionen und kompakte Propertydeklarationen sind nicht zulässig. Die Felder innerhalb von spec richten sich nach dem jeweiligen Kind; strukturierte object-Werte bleiben gewöhnliche Werte und sind keine eingebetteten Definitionen.

Der Dateiname ist keine Identität. Der enthaltende Ordner bestimmt dagegen das Package und damit einen Teil der Identität. Eine ausdrücklich bestimmte Modellwurzel begrenzt die Ordnerstruktur. Die Bindung externer Modelle und ihrer genauen Inhalte wird getrennt von den Packages festgelegt; Modell ist die gemeinsame Versionierungs- und Verteilungseinheit. Jede Definitionsdatei wählt daraus mit imports selbst ihre sichtbaren fremden Pakete; eigene Paketdefinitionen sind ohne Import sichtbar. Versionen und Digests werden nicht in jeder Definitionsdatei wiederholt. Abschnitt 8 trennt diese beiden Aufgaben.

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: Die API fordert Dokumentmetadaten beim Connector an.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

Dieses Beispiel verwendet das unten erklärte importierte Vokabular. Laufzeit-Requests erzeugen keine neuen kanonischen Exchange-Definitionen. list-documents beschreibt den gewünschten Vertrag dieser Operation.

## 3. Der feste Anfang: Kind und Property

**Nutzerentscheidung vom 15. Oktober 2026:** Kind.properties und Property.properties werden zu einer gemeinsamen Property properties zusammengefasst. Der gewählte Core besteht aus **neun Definitionen: zwei Kinds und sieben Properties**. Für die vollständige vereinfachte Darstellung verwenden die Meta-Properties kurze IDs; dadurch benötigen sie kein ausdrücklich gesetztes spec.name. Das sind die aktuellen Designidentitäten, keine automatische Umdeutung alter Paketfassungen.

Der Core versteht zwei feste Meta-Kinds:

| Kind | Bedeutung seiner Definitionen |
| --- | --- |
| Kind | Deklarieren Typverträge für weitere Definitionen. |
| Property | Deklarieren benannte Eigenschaften oder Beziehungen eines solchen Vertrags. |

Im festen Core-Paket ist Kind selbst eine Definition von Kind; Property ist eine Definition von Kind. Eine Projektdefinition wie Exchange hat den Kind Kind. Eine konkrete Exchange-Definition hat den Kind Exchange. Ein kind-Ziel muss selbst eine gültige Definition des festen Meta-Kinds Kind sein.

| Meta-Property-ID | Vertrag |
| --- | --- |
| name | Optionaler ausdrücklich abweichender Feldname; ohne Angabe gilt die vollständige lokale Property-ID. |
| type | Erforderliche Auswahl eines bekannten Werttyps. |
| minCount | Erforderliche nichtnegative Mindestanzahl. |
| maxCount | Erforderliche Höchstanzahl beziehungsweise -1. |
| values | Bei enum erforderliche nichtleere Liste eindeutiger erlaubter Stringwerte; sonst nicht zulässig. |
| target | Bei reference erforderliche Referenz auf den Ziel-Kind; sonst nicht zulässig. |
| properties | Optionale Liste von Referenzen auf Propertydefinitionen: Feldvertrag einer Kind-spec oder eines object-Werts gemäß Abschnitt 3.2. |

Der Compiler kennt die Bedeutung dieser Felder und kontrolliert die bereitgestellte Selbstbeschreibung gegen den festen Anfang. Die Selbstbeschreibung erzeugt ihre eigene Gültigkeit nicht aus Prosa. Die Core-Identitäten und der gebundene Core-Inhalt können durch Projektdefinitionen nicht ersetzt werden. Ein lokaler Kind namens Kind oder ein beliebiger Importalias erzeugen keine Sonderrechte. Beliebige benutzerdefinierte Metaklassen sind für diesen Vertrag nicht vorgesehen.

### 3.1 Vollständiger Core in neun Definitionsdateien

Alle neun Dateien gehören zum Paket markitect/core. Sie benötigen keine Imports, weil ihre Referenzziele im eigenen Paket liegen. Jede Datei hat dieselbe vollständige Hülle. Der effektive Feldname jeder Meta-Property entspricht ihrer kurzen lokalen ID; spec.name entfällt deshalb durchgängig.

**kind.yaml**

```yaml
id: Kind
kind: Kind
body: >-
  Deklariert einen Typ. properties benennt die erlaubten Felder
  der spec seiner Definitionen. Der body einer Typdefinition beschreibt
  deren gemeinsame Bedeutung und geltende Festlegungen.
spec:
  properties:
    - properties
```

**property.yaml**

```yaml
id: Property
kind: Kind
body: >-
  Deklariert einen Feldvertrag mit Werttyp und Kardinalität.
  Der body beschreibt die Bedeutung seiner Werte oder Beziehungen.
  Bei type object benennt properties die erlaubten Felder des Objektwerts.
spec:
  properties:
    - name
    - type
    - minCount
    - maxCount
    - values
    - target
    - properties
```

**name.yaml**

```yaml
id: name
kind: Property
body: >-
  Optionaler ausdrücklich abweichender nichtleerer Feldname.
  Ohne Angabe gilt die vollständige lokale ID der Property als Feldname.
  Der effektive Feldname muss innerhalb jedes verwendenden Vertrags
  eindeutig sein.
spec:
  type: string
  minCount: 0
  maxCount: 1
```

**type.yaml**

```yaml
id: type
kind: Property
body: Der vom Core verstandene Werttyp dieser Property.
spec:
  type: enum
  minCount: 1
  maxCount: 1
  values:
    - string
    - boolean
    - integer
    - number
    - enum
    - object
    - reference
```

**min-count.yaml**

```yaml
id: minCount
kind: Property
body: Nichtnegative Mindestanzahl vorhandener Werte.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-count.yaml**

```yaml
id: maxCount
kind: Property
body: >-
  Höchstanzahl vorhandener Werte. Der Wert -1 erlaubt beliebig viele Werte
  innerhalb der Werkzeuggrenzen. Andere Werte müssen mindestens minCount
  entsprechen. Genau 1 bezeichnet einen Einzelwert; andere Höchstwerte
  eine Liste.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**values.yaml**

```yaml
id: values
kind: Property
body: >-
  Erlaubte Stringwerte einer enum-Property.
  Bei type enum ist eine nichtleere Liste eindeutiger Werte erforderlich.
  Bei anderen Werttypen ist diese Angabe nicht zulässig.
spec:
  type: string
  minCount: 0
  maxCount: -1
```

**target.yaml**

```yaml
id: target
kind: Property
body: >-
  Der verlangte Kind der referenzierten Definitionen.
  Bei type reference ist diese Angabe erforderlich.
  Bei anderen Werttypen ist sie nicht zulässig.
spec:
  type: reference
  target: Kind
  minCount: 0
  maxCount: 1
```

**properties.yaml**

```yaml
id: properties
kind: Property
body: >-
  Referenziert die erlaubten Feldverträge des hier beschriebenen Objekts.
  Die effektiven Feldnamen müssen innerhalb dieses Vertrags eindeutig sein.
spec:
  type: reference
  target: Property
  minCount: 0
  maxCount: -1
```

### 3.2 Gemeinsame Property am Kind und am Objekt

Die beiden früheren Properties stimmen strukturell überein: effektiver Name properties, type reference, target Property, minCount 0, maxCount -1. Die gewählte gemeinsame Definition benennt die erlaubten Feldverträge eines beschriebenen Objekts. Der Unterschied ist, welches Objekt beschrieben wird: die spec der Definitionen eines Kinds oder der Objektwert einer object-Property. Diese Zuordnung folgt aus dem jeweiligen festen Meta-Vertrag und nicht aus Namensähnlichkeit.

kind.yaml und property.yaml aus Abschnitt 3.1 referenzieren dieselbe Property properties. Insbesondere ist property.yaml selbst eine Definition mit kind Kind. Ihre Liste properties ist deshalb die erlaubte Feldmenge aller Propertydefinitionen; sie benötigt selbst kein spec.type. Referenzzyklen dieses endlichen Anfangs werden nicht unendlich textuell entfaltet.

**Anwendung am Kind und an einem geschachtelten Objekt.** Die vier folgenden Dateien liegen gedanklich in einem Beispielpaket mit ausdrücklich bereitgestelltem Core. Als Core-Vertrag gilt der nun gewählte Anfang aus Abschnitt 3.1 mit neun Definitionen; der Durchgang behauptet keine vorhandene Paketveröffentlichung oder echte Digestbindung. Ein Kind Envelope erlaubt ein erforderliches Objektfeld payload; darin ist ausschließlich messageId erlaubt.

**envelope.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: Envelope
kind: Kind
body: Beschreibt eine Nachrichtenhülle mit einem strukturierten Inhalt.
spec:
  properties:
    - payload
```

**payload.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: payload
kind: Property
body: Strukturierter Inhalt der Nachrichtenhülle.
spec:
  type: object
  minCount: 1
  maxCount: 1
  properties:
    - messageId
```

**message-id.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: messageId
kind: Property
body: Identifikator der Nachricht.
spec:
  type: string
  minCount: 1
  maxCount: 1
```

**request.yaml**

```yaml
id: request
kind: Envelope
body: Eine beabsichtigte Nachrichtenform für das Beispiel.
spec:
  payload:
    messageId: document-request
```

envelope.yaml verwendet properties aus dem Kind-Vertrag und bestimmt damit die erlaubten Felder von request.spec. payload.yaml verwendet dieselbe Meta-Property aus dem Property-Vertrag und bestimmt die Felder von request.spec.payload. messageId ist in beiden Fällen eine normale vollständige Propertydefinition. request.yaml benötigt keinen fremden Import, weil Envelope im eigenen Paket liegt; payload und messageId im konkreten Wert sind Feldschlüssel, keine Definitionsreferenzen.

**Die genaue Compilergrenze bleibt fest:**

| Betrachtete Definition | Bedeutung der gemeinsamen Property |
| --- | --- |
| kind ist der feste Meta-Kind Kind | properties deklariert die erlaubten Felder der spec der Definitionen dieses Kinds. Das gilt auch für kind.yaml und property.yaml selbst. |
| kind ist der feste Meta-Kind Property und spec.type ist object | properties deklariert die erlaubten Felder des jeweiligen Objektwerts. |
| kind ist der feste Meta-Kind Property mit anderem spec.type | properties ist als Vertragsangabe nicht zulässig, auch nicht als leere Liste. Die bereits bestehende Werttypbedingung bleibt erhalten. |
| kind ist ein gewöhnlicher Projekt-Kind | Dessen eigener Feldvertrag gilt. Ein dort erlaubtes Feld namens properties, auch bei Wiederverwendung der gemeinsamen Property, ist gewöhnlicher Dateninhalt und erzeugt keinen zusätzlichen Typ- oder Objektvertragsoperator. |

Die letzte Zeile verhindert eine unzulässige globale Reservierung des Feldnamens. Ein Projekt-Kind SchemaCatalog könnte properties als Liste von Propertyreferenzen deklarieren und konkrete Katalogeinträge damit beschreiben. Ein solcher Katalogeintrag wird dadurch nicht zum Kind; nur eine Definition des festen Meta-Kinds Kind deklariert einen neuen Typvertrag. Die feste Sprachbedeutung hängt an genau gebundenen Meta-Identitäten, nicht an geschriebenen Namen oder einem beliebigen Importalias.

**Gegenproben des Durchgangs:**

| Fall | Erwarteter Befund und Grund |
| --- | --- |
| request.spec.payload.messageId ist gesetzt | Strukturell passend zum zweistufigen geschlossenen Feldvertrag; das beweist keine fachlich geeignete MessageId. |
| request.spec.messageId steht statt payload.messageId | Nicht erlaubt: messageId gehört zum Objektvertrag von payload, nicht zur äußeren Envelope-spec. |
| payload enthält ein zusätzliches unbekanntes Feld | Strukturfehler im geschlossenen Objektvertrag. |
| messageId mit type string setzt zusätzlich properties | Strukturfehler: properties ist an einer Propertydefinition nur für type object erlaubt. |
| Ein properties-Eintrag referenziert einen konkreten Gegenstand statt einer Propertydefinition | Strukturfehler am Ziel-Kind der Referenz. |
| Zwei verschiedene Property-IDs ergeben im selben Vertrag denselben effektiven Feldnamen | Strukturfehler unabhängig davon, ob der Name ausdrücklich gesetzt oder aus der ID übernommen ist. |
| Derselbe Feldname tritt in verschiedenen unabhängigen Kind-/Objektverträgen auf | Kein Namenskonflikt zwischen diesen Verträgen; Eindeutigkeit wird pro Vertrag geprüft. Die gemeinsame Meta-Property darf in Kind und Property wiederverwendet werden. |
| Eine gewöhnliche Projektdefinition führt ein laut ihrem Kind erlaubtes Datenfeld properties | Normale Wert-/Referenzprüfung; keine automatische Schemaausführung. |
| Der Core-Anfang referenziert Kind, Property und properties zyklisch | Endlicher Definitionsgraph; die Referenzen werden nicht unendlich textuell entfaltet. |
| Eine object-Property verwendet sich selbst als eigenes Objektfeld | Bei minCount=0 sind endliche Werte möglich und die Rekursion ist erlaubt; bei minCount>0 ist der verpflichtende Selbstzyklus unzulässig. Abschnitt 5.3 bestimmt die genaue Grenze. |

**Gewähltes Ergebnis und Migration:** Die gemeinsame Property properties ersetzt im angestrebten Core die bisherigen getrennten Definitionen Kind.properties und Property.properties. Die übrigen Meta-Properties erhalten die kurzen IDs name, type, minCount, maxCount, values und target. Damit sind alle Core-Feldnamen ohne spec.name unmittelbar durch die jeweilige lokale ID bestimmt. Bestehende Referenzen, Körperlinks, Seedkontrolle und betroffene Modellbindungen müssen auf diese ausdrücklich geänderten Identitäten migriert werden. Es gibt keine stille Weiterleitung und keinen rückwirkend geänderten historischen Core-Inhalt. Die Bedingung für object-Properties bleibt eine feste Compilerregel.

Die Zusammenfassung ist eine Nutzerentscheidung; die Beispiele und Gegenproben erläutern das erwartete Verhalten des gewählten Vertrags. Damit wurde kein neuer produktiver Core-Anfang implementiert oder unabhängig ausgeführter Compiler-/Laufzeitnachweis erzeugt. Die historischen Berichte über zehn Core-Definitionen und deren Modellbindungen bleiben Evidenz ihres damaligen Stands.

## 4. Verbindliche Prosa und Geltungsbereich

body ist Teil des geltenden Modellvertrags und wird im gebundenen Inhalt erhalten.

- Der body eines Kinds beschreibt die gemeinsame Bedeutung und geltenden Festlegungen seiner Definitionen.
- Der body einer Property beschreibt die Bedeutung ihrer Werte und Beziehungen im verwendenden Vertrag.
- Der individuelle body ergänzt die Festlegungen des konkreten Gegenstands.
- Individuelle Angaben heben allgemeine Pflichten nicht still auf. Widersprüche werden als zu klärende Befunde behandelt.

Der Text kann Modellbedingungen, gewünschtes Laufzeitverhalten, Grenzen und Bedingungen seiner Anwendbarkeit beschreiben. Eine nicht formal prüfbare Aussage bleibt verbindlich. Der Core besitzt vorerst keine besonderen requirements-, roles- oder subject-Abschnitte und keine allgemeine formale Regelsprache. Ein Projekt kann solche Wörter als eigene Begriffe verwenden, ohne dadurch besondere Compilersemantik zu erhalten.

Beziehungen erhalten ihre fachliche Bedeutung aus dem Typ- und Propertyvertrag. Eine Aussage über requester betrifft die dort referenzierte Anwendung im beschriebenen Austausch. Aus einem Relationsnamen folgen weder globale Regelgeltung noch transitive Zusammensetzung, Agentenaufteilung oder Schreibrechte.

Ein bloßes Wort im body erzeugt keine Referenzkante. Explizite Körperlinks verwenden denselben Zieladressmechanismus. Der folgende Abschnitt konkretisiert die bereits im [früheren Prosaentwurf](../../product-model-refinement/canonical-model-proposal.md#25-definitionskörper-explizite-links-und-linkvorschläge) vereinbarte Richtung. Der Nutzer hat diesen Körperlinkvertrag im anschließenden Designgespräch gewählt. Die gewählten lexikalischen Regeln stehen in Abschnitt 4.5; eine Produktimplementierung wird nicht behauptet.

### 4.1 Ein Linkformat für alle Definitionen

Die Autorenform lautet `[[Ziel]]` oder `[[Ziel|Anzeigetext]]`. Das Ziel ist eine im eigenen Package oder durch direkte Imports verfügbare id, eine vollständige Referenz package/id oder eine ausdrücklich aliasqualifizierte Referenz Alias.id. Die Auflösung erfolgt im eigenen Paket und den dateilokalen Imports der Definition, deren body den Link enthält. Sie benutzt dieselben festen Paketbindungen wie kind und die Referenzwerte in spec. Der Anzeigetext beeinflusst die Auflösung nicht. Ohne ausdrücklichen Anzeigetext wird die geschriebene Zieladresse angezeigt.

| Körperlink im Projektpaket | Ziel im Beispiel |
| --- | --- |
| `[[Exchange|Austausch]]` | Die Kind-Definition Exchange im importierten Vokabular. |
| `[[list-documents|Dokumentabfrage]]` | Der konkrete beschriebene Austausch im eigenen Paket. |
| `[[api|Cloud-API]]` | Die konkrete API-Definition im eigenen Paket. |
| `[[requester]]` | Die Propertydefinition mit stabiler id requester und aktuellem Feldnamen requester. |

Jede Property ist über ihre ausdrücklich deklarierte lokale id adressierbar. Es gibt weder eine eigene Propertylink-Syntax noch automatische Suche nach einem Feldnamen. Eine Feldbezeichnung wird nicht automatisch zu einer zusätzlichen Property-ID.

### 4.2 Exchange: Vertrag, Feld und konkreter Wert

Im Vokabularpaket kann der Kind-Körper seine eigene Property adressieren:

```yaml
imports:
  - package: markitect/core
    as: Core

id: Exchange
kind: Kind
body: >-
  Beschreibt einen Request-/Reply-Vertrag.
  Die durch [[requester]] bestimmte Anwendung muss
  Antworten anhand der MessageId dem ursprünglichen Request zuordnen.
spec:
  properties:
    - requester
    - receiver
    - command
    - requestChannel
    - replyChannel
    - timeoutSeconds
    - maxAttempts
```

Diese Ansicht zeigt dieselbe Exchange-Definition mit dem für die Körperlinks betrachteten body; sie ist keine zweite Autorenquelle neben Abschnitt 6. Die referenzierten Properties stehen jeweils in eigenen Dateien. Der Link bezeichnet die Propertydeklaration. Der Satz beschreibt die Pflicht der Anwendung, die in jeder Exchange-Definition als requester eingetragen ist. Diese Aussage kommt aus der Prosa, nicht aus einem vom Core ausgeführten Zugriffsausdruck.

Im Projektpaket sieht der konkrete Ausschnitt so aus:

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: >-
  Dieser [[Exchange|Austausch]] dient der Dokumentabfrage.
  [[api|Die Cloud-API]] fordert die Daten bei [[onprem|dem Connector]] an.
  Die durch [[requester]] bestimmte Anwendung
  muss verspätete Antworten nach Ablauf der Wartefrist verwerfen.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

Hier entstehen Körperbezüge auf den Kind, zwei konkrete Definitionen und eine Propertydefinition. spec.requester setzt dagegen ausdrücklich die typisierte Auftraggeberbeziehung zu api. Der Körperlink setzt diesen Wert nicht. Stimmen Satz und spec nicht überein, ist das ein semantischer Widerspruch; eine erfolgreiche Zielauflösung beweist ihre Übereinstimmung nicht.

Ein Link auf requester bezeichnet keinen einzelnen Propertywert von list-documents. Der erste Umfang benötigt keine zusätzliche Ausdruckssprache für Dereferenzierung, Filter, Feldpfade oder berechnete Links. Ein Satz kann den Zusammenhang über den benannten Propertyvertrag ausdrücken; muss ein konkretes Ziel explizit im Graph erscheinen, wird auch dieses Ziel ausdrücklich referenziert.

### 4.3 Die feste Bedeutung des Links

Der Core kennt genau eine zusätzliche Beziehung aus diesem Markup: **Die Quelldefinition nimmt an dieser Körperstelle ausdrücklich auf die Zieldefinition Bezug.** Er hält je Vorkommen Quelle, kanonisches Paket-/ID-Ziel, optionalen Anzeigetext und Fundstelle fest. Wiederholte Links bleiben nachvollziehbare Vorkommen; eine zusammengefasste Graphsicht darf sie als einen Bezug mit mehreren Fundstellen darstellen. Dafür gibt es keine zweite manuelle uses-/references-Liste und keinen neuen eingebauten fachlichen Kind.

Die fachliche Aussage entsteht aus Satz, Kind- und Propertyvertrag. Daraus folgen diese Grenzen:

- Ein Kind-Link typisiert die Quelle nicht. Dafür gilt das Feld kind.
- Ein Property-Link erklärt oder verwendet den Feldvertrag; er trägt keinen Wert in spec ein und zählt nicht als Erfüllung einer Property-Kardinalität.
- Ein Definitionslink erklärt allein weder Enthaltenheit, Laufzeitaufruf, positive Abhängigkeit noch globale Regelgeltung. Auch „verwendet [[dms]] nicht“ enthält einen ausdrücklichen Bezug auf dms.
- Das Referenzieren einer Regel aktiviert sie nicht automatisch für den ganzen verlinkten Gegenstand. Der geltende Umfang muss aus ihrem Vertrag und der Aussage hervorgehen.
- Verbindliche unmarkierte Prosa bleibt Teil des Modells. Links helfen beim eindeutigen Bezeichnen, ersetzen aber weder die Aussage noch eine Vollständigkeitsbeurteilung.

Der Bezug ist damit für Navigation, Kontext und Änderungsprüfung maschinell nutzbar. Eine Änderung am Ziel macht die Quellen zu Kandidaten für eine erneute Beurteilung. Sie beweist keine nötige Codeänderung und erzeugt keinen Ausführungsauftrag. Beim semantischen Lesen gehört die gebundene Bedeutung des Ziels zum Verständnis des Satzes; bloße Erreichbarkeit im Graph importiert nicht transitiv sämtliche Pflichten aller erreichbaren Definitionen.

### 4.4 Prüfung, Umbenennung und Modellstand

Der Compiler kann aktive Links erkennen, ihre Ziele nach vollständigem Laden der Definitionen eindeutig auflösen und Herkunft erhalten. Fehlendes Ziel oder unbekannter Import ist ein Fehler; Titel, Anzeigetext und erwartete Bedeutung dienen nicht als Ausweichsuche. Anders als bei einer typisierten reference-Property ist im Körperlink zunächst jede Definition als Ziel zulässig. Ob der Satz den richtigen Kind, die richtige Property oder die richtige konkrete Definition meint, bleibt Gegenstand des semantischen Reviews.

Beim Wechsel eines Importaliases Connector zu Wire müssen nur ausdrücklich damit qualifizierte Autorenreferenzen angepasst werden, etwa Connector.Exchange zu Wire.Exchange. Eine eindeutige direkt verfügbare Referenz Exchange bleibt unverändert; das kanonische Ziel bleibt gleich. Im historischen Umbenennungsfall kann eine ausdrücklich erhaltene ID wie exchange.sender mit dem Feldnamen requester weiterhin verlinkt werden. Im aktuellen vereinfachten Beispiel heißt bereits die ID requester und der Link [[requester]] bleibt eindeutig. Ein veralteter Anzeigetext sender kann weiterhin strukturell gültig sein und dennoch sprachlich irreführen. Eine echte Änderung der Property-ID erfordert die Migration ihrer Links; ohne Migration sind alte Ziele nicht mehr auflösbar. Ein Versionsupdate bindet dieselbe Zielidentität an deren ausdrücklich gewählte neue Fassung.

Der feste Modellstand enthält den vollständigen Körper und die aufgelösten Zielidentitäten. Änderungen an Prosa oder Anzeigetext werden erhalten, auch wenn die Zielmenge gleich bleibt. Quellpositionen gehören zur Diagnoseherkunft, nicht zur logischen Zielidentität. Aliaswechsel ändern das kanonische Linkziel nicht; eine Änderung des gespeicherten Körpers kann trotzdem im Inhaltsvergleich sichtbar sein. Die genaue Digestkodierung steht inzwischen in Abschnitt 8.3.

Der Compiler prüft die Existenz der Bezüge, nicht die Wahrheit, Vollständigkeit oder korrekte Auslegung der Aussagen. Agenten können fehlende oder falsche Links vorschlagen; erst eine in der Quelle übernommene Änderung verändert den kanonischen Bezug. Linkvorschläge aus Wortähnlichkeit sind keine Modellkanten.

### 4.5 Verbindliches Körperlink- und Markdownprofil

**Nutzerentscheidung vom 15. Oktober 2026:** body verwendet [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/) mit der ausdrücklichen Erweiterung um Körperlinks. Diese Regeln gelten ausschließlich für body. title und andere Stringwerte werden nicht auf Modelllinks untersucht.

Die Verarbeitung beginnt mit der YAML-Dekodierung gemäß Abschnitt 5.1. Danach wird der dekodierte body als Markdown mit Körperlink-Erweiterung gelesen. YAML-Escapes können daher bereits vor der Markdown-Verarbeitung Zeichen der Linknotation ergeben. Aus späteren Markdown-Zeichenreferenzen wie &#91; oder gerendertem Text werden dagegen keine neuen Modelllinks erzeugt. Der ursprüngliche body bleibt erhalten; Normalisierung für die Markdown-Verarbeitung ist keine automatische Änderung der Quelle.

| Kontext | Verhalten |
| --- | --- |
| Gewöhnlicher Text, Überschrift, Liste, Zitat oder Formatierung außerhalb eines Links | Die ungeschützte Doppelklammernotation ist aktiv. |
| Inline-Code und eingerückter oder eingezäunter Codeblock | Wörtlicher Inhalt, keine Modelllinks oder Linksyntaxfehler daraus. |
| Gewöhnlicher Markdownlink oder Bild, einschließlich Text, Ziel und Referenzdefinition | Keine Körperlink-Auswertung innerhalb dieses Konstrukts. |
| Autolink, HTML-Tag, HTML-Kommentar oder HTML-Rohblock | Keine Körperlink-Auswertung darin. Normaler Markdowntext zwischen Inline-HTML-Tags folgt weiter seinen normalen Regeln; Linkelemente bilden keinen verschachtelten Modelllink. |

Die aktive Doppelklammernotation hat im gewöhnlichen Text Vorrang vor der gewöhnlichen Markdown-Linkerkennung: `[[api]](Text)` enthält den Modelllink api mit folgendem normalem Text. Ein bereits mit gewöhnlicher Markdownsyntax begonnener Link oder ein Bild schützt dagegen seinen Inhalt. Es gibt keine zusätzliche Bild-/Einbettungsform für Modelldefinitionen.

**Grammatik:** Ein aktiver Link hat genau die Form `[[Ziel]]` oder `[[Ziel|Anzeigetext]]`. Das Ziel ist eine vollständige Autorenreferenz nach Abschnitt 7 und 8. Es darf nicht leer sein und enthält keine umgebenden Leerzeichen; es wird weder getrimmt noch umgeschrieben. NFC, deklarierte Schreibweise, Importgrenzen, Kollisions- und Mehrdeutigkeitsregeln gelten unverändert. Der Link kann jede Definitionskategorie adressieren. Feldpfade, URL-Ziele, Titel- oder Ähnlichkeitssuche werden nicht ergänzt.

Der optionale Anzeigetext ist reiner Unicode-Text. Er wird nach den nachstehenden Escapes erhalten und nicht nochmals als Markdown, Zeichenreferenz, Modelllink oder Ausdruck ausgewertet. Er darf nicht leer sein oder ausschließlich aus Unicode-White_Space-Zeichen der gebundenen Unicode-Version bestehen. Formatierung steht außerhalb, beispielsweise `**[[api|Cloud-API]]**`. Innerhalb eines aktiven Links sind LF, CR, NEL, LINE SEPARATOR und PARAGRAPH SEPARATOR sowie verschachtelte ungeschützte Klammern ausgeschlossen.

**Escapes:** Außerhalb des Links gelten die CommonMark-Regeln. `\[[api]]` macht die Notation wörtlich; ein durch einen weiteren Backslash geschützter Backslash verhindert einen danach normal beginnenden Link nicht. Im Anzeigetext sind genau `\|`, `\[`, `\]` und `\\` für die entsprechenden literalen Zeichen erlaubt. Andere Backslashsequenzen im Linkinneren sind Fehler. Im Ziel sind keine Escapes nötig oder zulässig; seine Zeichen folgen der Namensgrammatik. Ein zusätzliches ungeschütztes |, eine ungeschützte einzelne Klammer oder ein aktives [[ ohne passenden Abschluss ist ein Markupfehler. Ein isoliertes ]] außerhalb eines aktiven Links bleibt gewöhnlicher Text.

Ein Link muss im dekodierten body einzeilig sein. YAML kann den Autorenquelltext über mehrere Dateizeilen falten, solange dabei im dekodierten Link kein verbotenes Zeilenumbruchzeichen entsteht. Ungültige aktive Notation ist ein Strukturfehler und wird nicht still zu Text zurückgestuft. Gleichartige Zeichenfolgen in den geschützten Kontexten bleiben dagegen wörtlich.

**Fundstellen:** Jedes Vorkommen erhält einen nullbasierten, am Ende exklusiven UTF-8-Bytebereich [startByte, endByte) im dekodierten ursprünglichen body. Er umfasst die vollständige Linknotation. Die Zuordnung zur ursprünglichen YAML-Datei hält zusätzlich Start und exklusives Ende als einbasierte Zeile und Spalte fest; Spalten zählen Unicode-Codepoints und keine UTF-8-Bytes oder Bildschirmbreiten. Dabei müssen Zitat-Escapes und YAML-Faltung berücksichtigt werden. Wiederholte Links bleiben einzelne Vorkommen. Fundstellen sind Herkunft und keine zusätzliche Zielidentität.

Der begrenzte Beispielprüfer verwendet eine isolierte, über requirements.txt gebundene CommonMark-Parserabhängigkeit und prüft Körperlink-Syntax, geschützte Kontexte, Referenzauflösung und Quellenzuordnung mit Gegenbeispielen. Die Go-Produktabhängigkeiten werden dadurch nicht verändert. Seine technische Grenze beträgt 2000000 UTF-8-Bytes je body und 64 Verschachtelungsebenen; eine nicht belegbare Quellenzuordnung oder ein Ressourcenabbruch meldet die Prüfung als nicht abgeschlossen. Wegen einer bekannten Leerraumabweichung der gebundenen Parserfassung meldet die Probe dies auch für ansonsten gelesene bodies mit U+001C bis U+001F oder U+0085. Das ist eine ausdrücklich ausgewiesene Werkzeuggrenze und kein Verbot dieser Zeichen im Sprachvertrag. Ein separat bestandener Körperlink-Teil beweist weder die Gültigkeit des gesamten Modells noch die Wahrheit der verknüpften Prosa. Die vollständige kanonische Bytekodierung steht inzwischen in Abschnitt 8.3; generische semantische Verfahren bleiben eine gesonderte Aufgabe.

## 5. Werte und Propertyverträge

| Werttyp | Bedeutung |
| --- | --- |
| string | Textwert. |
| boolean | Wahrheitswert. |
| integer | Beliebig genaue Ganzzahl. |
| number | Endlicher, exakt dargestellter Dezimalzahlenwert; Ganzzahlen sind ebenfalls zulässig. |
| enum | Einer der ausdrücklich deklarierten Stringwerte. |
| object | Objekt mit deklarierten, geschlossenen Feldverträgen. |
| reference | Referenz auf eine vorhandene Definition des ausdrücklich verlangten Kinds. |

Für Properties gelten folgende feste Sprachregeln:

- spec.name ist optional. Ohne Angabe ist der effektive Feldname exakt die vollständige lokale id der Propertydefinition. Paketname, Importalias und Referenzschreibweise gehören nicht zu diesem Feldnamen; Punkte oder Präfixe der lokalen ID werden nicht abgeschnitten.
- Ein ausdrücklich gesetztes spec.name muss ein nichtleerer String sein; null oder eine leere Angabe lösen keinen Fallback aus. Die effektiven Feldnamen müssen innerhalb jedes Kind- oder object-Vertrags eindeutig sein.
- minCount ist mindestens 0. maxCount ist -1 oder mindestens minCount.
- maxCount=1 bezeichnet einen Einzelwert; andere Höchstwerte bezeichnen eine Liste. -1 erlaubt beliebig viele Werte innerhalb endlicher Werkzeuggrenzen.
- Eine ausgelassene Angabe zählt als null vorhandene Werte. Der Datenwert null ist nicht zulässig.
- reference benötigt target. Das Ziel von target ist eine gültige Typdefinition.
- enum benötigt eine nichtleere Liste eindeutiger erlaubter Werte.
- object verwendet die unter properties referenzierten eigenständigen Propertydefinitionen. Nicht deklarierte Objektfelder sind Fehler. Ein konkreter Objektwert enthält Werte dieser Felder, keine neuen Propertydefinitionen.
- values, target und properties sind nur bei den dafür vorgesehenen Werttypen zulässig. Ein neuer Typname installiert keinen Parser oder Operator.

Ein separater Werttyp kindReference ist im neuen Vertrag nicht nötig. Ein Typbezug kann durch reference mit Ziel-Kind Kind beschrieben werden. Die Typbezüge im gemeinsamen Feld kind bleiben ausdrücklich als solche prüfbar.

Die Kardinalität begrenzt die Zahl der Werte, nicht deren numerischen Wertebereich. Eine fachliche Aussage wie „maxAttempts muss positiv sein“ benötigt weiterhin semantische Beurteilung, solange keine zusätzliche formale Wertebedingung eingeführt ist.

### 5.1 YAML- und Werteprofil

**Nutzerentscheidung vom 15. Oktober 2026:** Die Autorenform verwendet ein begrenztes Profil von [YAML 1.2.2](https://yaml.org/spec/1.2.2/). Der folgende Vertrag bestimmt die Leseform unabhängig von Bibliothek und Implementierungssprache. Er ist keine Wahl von PyYAML für das Produkt. Markitect wird in Go gebaut; der Python-Prüfer ist eine getrennte Entwurfsprobe. Seine Parser-Voreinstellungen sind nicht normativ.

| Bereich | Verbindliche Regel |
| --- | --- |
| Datei und Dokument | UTF-8, genau ein YAML-Dokument und ein Mapping als Wurzel. Ein einzelner Dokumentmarker und eine ausdrückliche YAML-1.2-Direktive sind zulässig. Andere Versionsdirektiven und Dokumentfolgen sind Fehler. |
| Schlüssel | Ausschließlich Strings, exakt und in NFC geschrieben. Doppelte sowie nach Abschnitt 7.2 kollidierende Schlüssel sind Fehler. Nichttextuelle Schlüssel werden nicht in Strings umgewandelt. |
| Explizite Inhalte | Anchors, YAML-Aliase, Merge-Keys und eigene Tags sind ausgeschlossen. Zur einheitlichen Autorenform sind auch explizite Standardtags wie !!str ausgeschlossen; gewöhnliche und zitierte Werte reichen aus. |
| Darstellung | Kommentare, Einrückung, Block-/Flowdarstellung und die Reihenfolge von Mapping-Schlüsseln erzeugen für sich keinen anderen Modellinhalt. Die YAML-Dekodierung muss dabei denselben Wert ergeben. |
| Strings | Einfache, zitierte und mehrzeilige Strings sind erlaubt. Zitierte und Blockskalare bleiben Strings, einschließlich leerer Strings. Escapezeichen, Zeilenfaltung und Abschlusszeilen werden nach YAML dekodiert; dieser dekodierte Text bleibt erhalten. Pflichtfelder können zusätzlich nichtleeren Inhalt verlangen. Ungepaarte Unicode-Surrogate sind keine zulässigen Textwerte. |
| Booleans | Ausschließlich die unzitierten Tokens true und false werden als Wahrheitswerte gelesen. yes, no, on, off, True oder FALSE bleiben Strings. |
| Datum und sonstiger Text | Datumsähnliche Tokens wie 2026-10-15 bleiben Strings. Es gibt keinen automatisch erzeugten Datumstyp. Andere unzitierte Tokens, die keine der unten definierten Formen treffen, bleiben Text. |
| Fehlende Angaben und null | Optionale Felder können fehlen. Der Datenwert null ist ausgeschlossen; auch eine leere YAML-Wertangabe sowie die unzitierten Nullformen Null, NULL und ~ sind Fehler. Ein literaler Text wie "null" muss zitiert sein. |
| Typprüfung | Keine nachträgliche Typumwandlung anhand der Property: "42" erfüllt keine integer-Property, 42 erfüllt keine string-Property und "true" erfüllt keine boolean-Property. |

Die dateilokalen Importaliase mit as bleiben zulässig. Das Verbot betrifft YAML-Anker und deren Aliasverweise, nicht unsere Importnamen.

**Zahlenlexik und Genauigkeit:** Zahlen verwenden ASCII-Dezimalziffern, optional ein vorangestelltes Minus und bei Dezimalzahlen einen ASCII-Punkt beziehungsweise einen Exponenten. Unicode in Namen und Texten führt keine weiteren Zahlenschreibweisen ein.

- Ein Integerliteral hat die Form `-?(0|[1-9][0-9]*)`, etwa 0, -42 oder eine beliebig lange Ziffernfolge ohne führende Nullen.
- Ein Dezimalliteral besitzt denselben Ganzzahlteil und entweder einen Dezimalpunkt mit mindestens einer folgenden Ziffer, einen Exponenten oder beides. Ein Exponent beginnt mit e oder E, erlaubt optional + oder - und enthält mindestens eine ASCII-Ziffer. Beispiele: 0.1, -42.0, 1e300, 1.25e-3. Ein führendes +, .5 und 1. sind keine Zahlenliterale dieses Profils.
- Integerwerte sind beliebig genau, Dezimalwerte werden ohne binäre Gleitkommarundung erhalten. Eine number-Property akzeptiert beide Literalarten. Eine integer-Property akzeptiert ausschließlich Integerwerte; ein Dezimalliteral wie 42.0 wird nicht nachträglich in einen Integer verwandelt.
- Hexadezimale, oktale und sonstige zusätzliche Zahlenschreibweisen werden nicht numerisch gelesen. Beispielsweise bleiben 0x10 und 012 Text und erfüllen keine number-Property. Zitierte Zahlen bleiben ebenfalls Text.
- Nichtendliche numerische Tokens sind ausgeschlossen. Dazu zählen .nan, .inf, +.inf, -.inf, NaN, Inf und Infinity mit optionalem Vorzeichen bei Inf/Infinity und unabhängig von Groß-/Kleinschreibung. Als ausdrücklich zitierter Text bleiben diese Zeichenfolgen zulässig.

Es gibt keinen sprachlichen 64-Bit-Bereich oder festgelegten Rundungskontext. Endliche Werkzeuglimits sind ausdrücklich zu melden; eine Limitüberschreitung ist keine still gerundete Zahl und kein Beleg für die fachliche Ungültigkeit des Modells. Die aktuelle Probe begrenzt Zahlen auf 100000 Koeffizientenziffern und einen absoluten Exponenten von 1000000000. Der Bericht nennt diese Grenzen; ein abgebrochener Quelllauf meldet die Prüfung als nicht abgeschlossen.

Mapping-Reihenfolge ist damit ohne semantische Bedeutung. Die gewählten Listenregeln stehen in Abschnitt 5.2. Die kanonische Serialisierung äquivalenter Zahlenschreibweisen bleibt Teil der gesonderten Kodierungsentscheidung.

Die begrenzte Probe benutzt PyYAML ausschließlich für die Syntaxzerlegung, liest Skalare selbst nach diesem Profil und prüft die Definitionen anschließend. Sie erweitert weder die Produktabhängigkeiten noch implementiert sie den neuen Go-Compiler. Die spätere Go-Verarbeitung muss denselben Sprachvertrag erfüllen; die Bibliothek darf ihn nicht durch andere implizite Lese- oder Konvertierungsregeln verändern.

### 5.2 Listenreihenfolge und fehlende gegenüber leeren Listen

**Nutzerentscheidung vom 15. Oktober 2026:** Gewöhnliche Wertlisten sind geordnet. Deklarative Aufzählungen des festen Cores und die Importliste sind ungeordnet. Eine fehlende optionale Liste und eine ausdrücklich leere Liste haben denselben kanonischen Wert. Die originale Autorenform bleibt als Quelleninformation erhalten.

| Verwendung | Bedeutung der Reihenfolge |
| --- | --- |
| Gewöhnliche mehrwertige Property, einschließlich reference- und object-Werten | Die Reihenfolge zählt: [a, b] ist verschieden von [b, a]. Wiederholungen bleiben erhalten, soweit kein zusätzlicher Vertrag sie ausschließt. |
| properties einer Definition des festen Meta-Kinds Kind | Ungeordnete Deklaration der erlaubten Felder. Referenzen werden anhand der kanonischen Zielidentität verglichen. |
| properties einer object-Property des festen Meta-Kinds Property | Ungeordnete Deklaration des Objektfeldvertrags; dieselbe Zielauflösung gilt. |
| values einer enum-Property des festen Meta-Kinds Property | Ungeordnete Menge erlaubter Stringwerte. Die Werte bleiben exakt und werden nicht normalisiert oder ohne Beachtung der Groß-/Kleinschreibung verglichen. |
| Dateilokale imports | Ungeordnete Auswahl sichtbarer Pakete und deklarierter Aliase. Die Reihenfolge erzeugt keinen Vorrang bei der Auflösung. |

Die besondere Bedeutung folgt ausschließlich aus dem festen Meta-Vertrag, nicht aus einem geschriebenen Feldnamen. Ein gewöhnlicher Kind mit einem Datenfeld properties oder values besitzt dort eine geordnete Wertliste. Das gilt auch bei Wiederverwendung der gemeinsamen Core-Property properties als gewöhnlichem Datenfeld. Es werden keine zusätzlichen Collection-Typen, Sortierfelder oder kompakten Autorenformen eingeführt.

Ungeordnet bedeutet nicht, dass Fehler entfernt werden: Doppelte Felddeklarationen, doppelte Enumwerte und doppelte Paketimports bleiben gemäß ihren bestehenden Verträgen unzulässig. Der Prüfer validiert diese Regeln vor dem Vergleich und darf ungültige Eingaben nicht durch Sortieren oder Deduplizieren gültig machen. Zugefügte oder entfernte Mitglieder bleiben inhaltliche Änderungen. Nur das Umordnen einer zulässigen deklarativen Aufzählung ändert ihren Vertrag nicht. Dies gilt auch für die deklarativen Listen der festen Core-Selbstbeschreibung.

**Fehlend und leer:** Bei einer zulässigen optionalen Listen-Property, also minCount 0 und maxCount ungleich 1, bedeuten ein fehlendes Feld und [] jeweils dieselbe leere Folge. Das gilt rekursiv in object-Werten, bei einem leeren Kind-/Objektfeldvertrag und für fehlende gegenüber leeren imports. Bei minCount größer 0 sind beide Formen ungültig. Zusätzliche Meta-Regeln bleiben wirksam: Die values einer enum-Property müssen beispielsweise weiterhin nichtleer sein. An einem Werttyp verbotene Felder werden nicht durch eine angenommene leere Liste ergänzt.

Die Regel betrifft weder optionale Einzelwerte noch leere Strings oder leere Objekte. Ein fehlender object-Einzelwert bleibt von einem ausdrücklich vorhandenen {} verschieden; optionale Listen innerhalb des vorhandenen Objekts werden trotzdem auf dieselbe leere Folge verglichen. null bleibt ausgeschlossen. Eine fehlende Liste bedeutet damit keine zusätzliche Information wie unbekannt, automatisch bestimmen oder übernehmen; ein benötigter Unterschied muss ausdrücklich im Projektvokabular ausgedrückt werden.

Für den Vergleich gewöhnlicher Listen bleiben Reihenfolge und Wiederholungen erhalten. Deklarative Referenzen werden nach eindeutiger Auflösung ohne Reihenfolge verglichen, Enumwerte als exakte Strings und Imports anhand ihrer aufgelösten Bindungen. Quelltext, Prosa, ursprüngliche Reihenfolge und Feldanwesenheit bleiben für Darstellung und Herkunft verfügbar; die Quelldateien werden nicht automatisch ergänzt oder sortiert.

Der Beispielprüfer besitzt dafür eine begrenzte Vergleichsprobe nach erfolgreicher Strukturprüfung. Sie prüft auch geschachtelte Listen, abweichende gewöhnliche Datenfelder und die feste Core-Selbstbeschreibung. Diese frühere Listenprobe ist keine vollständige kanonische Bytekodierung und erzeugt selbst keinen kanonischen Inhaltsdigest. Die vollständige Serialisierung, Defaults und Referenzprojektion sind inzwischen in Abschnitt 8.3 definiert und besitzen eigene Gegenproben.

### 5.3 Objektvertragszyklen und Ressourcengrenzen

**Nutzerentscheidung vom 16. Oktober 2026:** Gewöhnliche Definitionsreferenzen dürfen Zyklen bilden, einschließlich des festen Core-Anfangs. Referenzziele werden als Identitäten aufgelöst und nicht durch textuelle oder wertseitige Entfaltung ersetzt. Für Objektverträge gilt die folgende eigene Bedingung.

Ein Objektvertragsgraph hat die object-Propertydefinitionen als Knoten. Eine Kante führt vom Objektvertrag zu jeder darin als Feld deklarierten object-Property. Eine Kante ist verpflichtend, wenn die Ziel-Property minCount>0 hat; das gilt für Einzelwerte und nichtleere Listen gleichermaßen. Der Teilgraph ausschließlich verpflichtender Kanten muss azyklisch sein. Damit enthält jeder erlaubte Objektzyklus mindestens eine optionale Kante mit minCount=0. Auch nicht verwendete Objektverträge werden auf verpflichtende Zyklen geprüft; ein unmöglich zu erfüllender Vertrag wird nicht durch fehlende konkrete Instanzen gültig.

| Fall | Sprachregel |
| --- | --- |
| Optionale object-Property verwendet sich selbst | Erlaubt; der Wert kann ohne dieses Feld beziehungsweise mit leerer optionaler Liste enden. |
| Wechselseitige Objektverträge mit mindestens einer optionalen Kante je Zyklus | Erlaubt; endliche konkrete Werte sind möglich. |
| Objektzyklus ausschließlich über verpflichtende Felder | Fehler; ein endlicher Wert kann diese Pflicht nicht erfüllen. |
| Gewöhnliche reference-Property führt auf einen bereits referenzierten Gegenstand | Erlaubter Definitionsbezug; keine Objektentfaltung. |
| Leere Liste bei minCount=0 und maxCount=0 | Zulässig; keine Objektwerte werden erzeugt oder besucht. |

Objektwerte bleiben endliche, geschlossene YAML-Werte gemäß Abschnitt 5.1. Die Werteprüfung und kanonische Projektion folgen ausschließlich tatsächlich vorhandenen Werten; sie entfalten den Vertragsgraphen nicht vorab und erzeugen keine fehlenden optionalen Objekte. Die bisherigen Regeln für fehlende optionale Listen gelten innerhalb vorhandener Objekte unverändert. Das Verbot von YAML-Anchors und -Aliasen bleibt bestehen.

**Ressourcenpolitik:** Die Sprache setzt keine zusätzlichen festen Größen-, Mengen- oder Tiefenmaxima. Werkzeuge dürfen endliche Verarbeitungsbudgets verwenden, müssen deren Umfang und Werte aber ausdrücklich ausweisen und im Prüfbericht nennen. Eine reine Limitüberschreitung ist kein Nachweis einer Modellverletzung: Die betroffene Prüfung meldet not_completed und erzeugt für diesen Modellumfang weder kanonische Bytes noch Digest oder Vollständigkeitsnachweis. Ein bereits nachgewiesener Sprachfehler bleibt ein Fehler, auch wenn eine weitere Prüfung abbricht; der Abbruch wird zusätzlich ausgewiesen. Ein unabhängiger vollständig geprüfter Modellumfang kann weiterhin ein Ergebnis erhalten.

Vollständig abgeschlossene Läufe über dieselben gebundenen Eingaben müssen bei ausreichend großen Budgets dieselben kanonischen Bytes und Digests liefern. Budgets und Laufzeitkapazitäten gehören zu Werkzeugkonfiguration und Prüfprovenienz, nicht zur kanonischen Modellprojektion. Es gibt kein stilles Abschneiden, Überspringen oder Runden als erfolgreiche Auswertung. Laufzeitbedingte Rekursions- oder Speicherabbrüche werden ebenfalls als unvollständige Auswertung ausgewiesen.

Der begrenzte Python-Prüfer verwendet zusätzlich zu den bereits benannten Körperlink-/Zahlenbudgets ein ausdrücklich konfigurierbares Budget für tatsächlich verschachtelte Objektwerte: --max-object-depth, Standard 128. Seine Python-Rekursionsgrenze wird im Bericht genannt; es ist kein eigener fester Speicherbytebetrag konfiguriert. Ein Speicherabbruch wird als solcher gemeldet. Die Objektvertragsprüfung verwendet für ihren verpflichtenden Graphen einen iterativen Durchlauf; reine Vertragstiefe verbraucht kein Objektwert-Tiefenbudget. Diese Probegrenzen sind keine Sprachmaxima. Die Gegenproben kontrollieren abbrechbare Selbst-/Mehrknotenzyklen, verpflichtende Zyklen, geschlossene Werte, unveränderte Quellen und Bytes bei unterschiedlichen erfolgreichen Budgets sowie ausbleibende Digests bei Abbruch.

Die Regeln dieses Abschnitts betreffen Definitionsreferenzen und Objektverträge. Für den gesonderten model.yaml-Abhängigkeitsgraphen gilt die Azyklizitätsregel aus Abschnitt 8.1; eine zyklische Definitionsreferenz ist keine Selbstabhängigkeit des Modells.

## 6. Eigenständige Properties und eine Datei pro Definition

**Nutzerentscheidung vom 15. Oktober 2026:** Jede Definition erhält eine eigene Datei und dieselbe Grundform. Eine Property wird ausschließlich als vollständige Definition mit kind Property geschrieben. Unter Kind.spec.properties steht eine Liste von Referenzen auf solche Definitionen. Eingebettete Propertyverträge und kompakte Referenz-Sonderformen sind nicht zulässig.

Das [vollständige vereinfachte Exchange-Beispiel](exchange-example/README.md) enthält die tatsächlichen Definitionsdateien samt Core, Vokabular, konkreten Projektgegenständen und begrenzter Strukturprobe. Diese YAML-Dateien besitzen den ausführbaren Beispielinhalt; die folgenden Ansichten erläutern ihn. Der Sprachvertrag besitzt die allgemeinen Regeln. Historische Probedateien werden dadurch nicht umgeschrieben.

Exchange und seine sieben Properties liegen im Vokabularpaket. Container, Command, Channel und MqttService sind dort vollständig definiert, ebenso die von ihnen benötigten Properties. Kurze IDs bilden den jeweiligen Feldnamen; spec.name ist deshalb nicht nötig. Die Dateinamen dienen nur der Ablage. Insbesondere unterscheiden command-kind.yaml und command-property.yaml den Kind Command von der Property command auch auf Windows eindeutig.

**exchange.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: Exchange
kind: Kind
body: >-
  Beschreibt einen Request-/Reply-Vertrag zwischen requester und receiver. Antworten behalten
  die MessageId des Requests. timeoutSeconds gilt pro Versuch; maxAttempts zählt den ersten
  Versuch mit.
spec:
  properties:
  - requester
  - receiver
  - command
  - requestChannel
  - replyChannel
  - timeoutSeconds
  - maxAttempts
```

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**receiver.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: receiver
kind: Property
body: >-
  Anwendung in der Bearbeiterrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

**command-property.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core

id: command
kind: Property
body: Vertrag der angeforderten Operation.
spec:
  type: reference
  target: Command
  minCount: 1
  maxCount: 1
```

**request-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requestChannel
kind: Property
body: >-
  Kommunikationsweg für Aufträge.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**reply-channel.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: replyChannel
kind: Property
body: >-
  Kommunikationsweg für Antworten an die ausstellende Instanz.
spec:
  type: reference
  target: Channel
  minCount: 1
  maxCount: 1
```

**timeout-seconds.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: timeoutSeconds
kind: Property
body: >-
  Positive maximale Wartefrist pro Versuch in Sekunden.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

**max-attempts.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: maxAttempts
kind: Property
body: >-
  Positives Versuchsbudget einschließlich des ersten Versuchs.
spec:
  type: integer
  minCount: 1
  maxCount: 1
```

Der konkrete Gegenstand steht in [list-documents.yaml](exchange-example/model/example/connector/project/list-documents.yaml), seine Teilnehmer, der Command, die Channels und messaging jeweils ebenfalls in eigenen Dateien. Alle Referenzziele sind im Beispiel vorhanden.

**Weitere Nutzerentscheidung vom 15. Oktober 2026:** Property-IDs benötigen keine vorgeschriebenen Präfixe. spec.name ist optional und verwendet bei Abwesenheit die vollständige lokale Property-ID als Feldnamen. Ein Punkt in einer ausdrücklich gegliederten ID erzeugt weder einen Besitzer noch einen weiteren Namespace oder Dateipfad. Ein explizites name kann weiterhin einen abweichenden Feldnamen festlegen; es ändert die Identität nicht.

Properties können ohne Kopie von mehreren Kinds oder object-Verträgen referenziert werden. Auch jedes Feld eines geschachtelten object besitzt eine eigene Propertydefinition; die object-Property verweist unter spec.properties auf deren IDs. Die effektiven Feldnamen müssen innerhalb jedes verwendenden Vertrags eindeutig sein. Doppelte Definitionen, doppelte Autoren-Schlüssel und kollidierende Feldnamen sind Fehler. Die Diagnoseherkunft verweist auf die jeweilige Definitionsdatei.

### 6.1 Kurze Property-ID mit Standardfeldname

Eine kurze ID benötigt kein zusätzliches name-Feld. Die folgende Ansicht entspricht der Property requester aus dem vollständigen Beispiel. Sie ersetzt nicht automatisch eine alte Identität wie exchange.sender; eine solche Umstellung verlangt ausdrücklich angepasste Referenzen.

**requester.yaml**

```yaml
imports:
  - package: markitect/core
    as: Core
id: requester
kind: Property
body: >-
  Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  type: reference
  target: Container
  minCount: 1
  maxCount: 1
```

Ein Kind, der diese Property verwendet, referenziert requester statt exchange.sender. Sein konkretes spec-Feld heißt ohne weitere Angabe requester. Eine ausdrückliche Angabe name: requester ergäbe denselben effektiven Feldvertrag, ohne dass damit bereits gleiche Quell- oder Modelldigests behauptet werden. Die Autorenquelle und Herkunft bleiben erhalten; die genaue Digestkodierung ist weiterhin offen.

Der Standard wird aus der lokalen ID der Zieldefinition gebildet, nicht aus dem geschriebenen Bezug: Auch ein Importalias oder eine Referenz wie example/connector/vocabulary/requester ergibt den Feldnamen requester. Bei einer lokalen ID exchange.sender und fehlendem name wäre der Feldname dagegen vollständig exchange.sender; es wird weder sender noch requester geraten. Ein ausdrücklich gesetztes name: requester hält die bisherige Feldbenennung aufrecht.

Ändert sich eine ID bei fehlendem name, ändert sich auch ihr Standardfeldname. Sollen bestehende Feldschlüssel erhalten bleiben, muss bei der ID-Migration der bisherige Name ausdrücklich gesetzt werden. Umgekehrt kann bei stabiler ID ein ausdrücklich gesetztes name den Feldschlüssel ändern; betroffene konkrete specs müssen migriert werden. Das Löschen eines ausdrücklich abweichenden name aktiviert den ID-Standard und kann daher ebenfalls eine Feldmigration verlangen. IDs werden durch diese Regeln nicht automatisch umbenannt.

## 7. Identitäten und Referenzen

**Nutzerentscheidung vom 16. Oktober 2026:** Package ist ausschließlich Namensraum. Jeder Ordner unter einer ausdrücklich bestimmten Modellwurzel bildet genau ein Package; sein relativer Ordnerpfad bestimmt den Package-Namen. Zusätzliche Namespaces und ein namespace-Feld gibt es nicht. Die frühere Kopplung von Package und Versionierungs-/Verteilungseinheit ist aufgehoben. Imports machen weiterhin die vollständigen IDs der Definitionen eines ausdrücklich importierten Packages direkt verfügbar; kind kann beispielsweise Kind oder Exchange statt Core.Kind oder Connector.Exchange verwenden. Package-Namen verwenden / entsprechend dem relativen Ordnerpfad. Vollständige Package-Qualifizierung lautet package/id, kurze Aliasqualifizierung weiterhin Alias.id. Gegliederte lokale IDs behalten Punkte. Die frühere Punktschreibweise für Package-Namen ist abgelöst und keine gültige alternative Schreibweise. Modell ist die gemeinsame Versionierungs- und Verteilungseinheit; Abschnitt 8.1 bestimmt Manifest, Identität, Wurzel und Abhängigkeitsform. Modellabhängigkeiten müssen gemäß Abschnitt 8.1 azyklisch sein; Selbstabhängigkeiten sind damit ebenfalls verboten.

Die logische Identität bleibt **Paket-ID plus stabile lokale id**. Sie enthält weder Kind noch Anzeigename, Dateiname oder Versionsnummer. Der Ordnerpfad bestimmt jedoch die Paket-ID; ein anderer Packageordner ändert deshalb die Identität. Eine bestimmte Fassung wird zusätzlich durch ihren Modellstand und dessen Inhaltsbindungen bestimmt. Die Autorenreferenz package/id beziehungsweise Alias.id wird auf dieses kanonische Paar aufgelöst.

| Gegenstand | Paket | Lokale id |
| --- | --- | --- |
| Exchange-Kind | example/connector/vocabulary | Exchange |
| Auftraggeber-Property | example/connector/vocabulary | requester |
| Konkreter Austausch | example/connector/project | list-documents |
| Konkrete API | example/connector/project | api |

Die Package-Namen sind synthetische Beispiele mit / zwischen den Ordnerkomponenten. Eine ID wie exchange.sender ist ein vollständiger lokaler Name innerhalb eines Pakets; der Punkt definiert keine zusätzliche Namensraumhierarchie, Enthaltenheit oder Property-Dereferenzierung. Ein Import macht genau diese vollständige ID verfügbar, nicht automatisch den letzten Abschnitt sender. Ein / ist in einer lokalen id nicht erlaubt: Er trennt Package-Komponenten und bei vollständiger Qualifizierung das Package von der vollständigen lokalen ID. Die gewählte Umstellung wird ausdrücklich in Imports und Referenzen vorgenommen; alte Punktschreibweisen für Packages werden weder akzeptiert noch automatisch weitergeleitet.

Für Identität und Kollisionsprüfung macht Groß-/Kleinschreibung keinen Unterschied. Innerhalb eines Pakets dürfen daher auch IDs unterschiedlicher Kinds nicht allein durch Groß-/Kleinschreibung unterscheidbar sein. Die erlaubte Autorenform ist trotzdem streng und muss exakt eingehalten werden: Eine abweichende Schreibweise ist ein Fehler, keine akzeptierte alternative Referenz. Der Prüfer korrigiert oder normalisiert keine Quelldatei automatisch. Lesbare IDs werden nicht bei jedem Wechsel der bevorzugten Bezeichnung geändert oder absichtlich für andere Gegenstände wiederverwendet. Ein optionales title ist keine alternative Adresse.

Die Autorenauflösung verwendet direkt verfügbare Namen und bei Bedarf ausdrücklich qualifizierte Namen. Sichtbar sind ausschließlich das eigene Paket und die in der jeweiligen Definitionsdatei deklarierten Imports; die Bereitstellung eines Modells allein öffnet keine fremden Namen:

- Eine vollständige id wird gegen die Definitionen des eigenen Pakets und aller in dieser Definitionsdatei direkt importierten Pakete aufgelöst. Genau ein kanonisches Ziel muss passen. Beispiel: Kind stammt aus dem importierten Core-Paket, Exchange aus dem importierten Vokabular, api aus dem eigenen Projektpaket.
- Mehrere unterschiedliche Ziele mit derselben sichtbaren id sind auch bei bloßem Unterschied der Groß-/Kleinschreibung mehrdeutig. Es gibt weder einen stillen Vorrang lokaler Definitionen noch eine Auswahl nach dem erwarteten Kind oder der Reihenfolge der Imports. Die Referenz muss ausdrücklich qualifiziert oder die Namenskollision aufgelöst werden.
- Vollständige Qualifizierung verwendet importierterPackagepfad/id, etwa markitect/core/Kind oder example/connector/vocabulary/exchange.sender. Das letzte / trennt Package und vollständige lokale ID. Ein ausdrücklich deklarierter Kurzalias wird dagegen mit Punkt verwendet: Core.Kind oder Connector.exchange.sender. Beide Formen bezeichnen genau ein in dieser Definitionsdatei importiertes Package. Punkte in einer lokalen ID bleiben Teil dieser ID. Aliasqualifizierung kann weiterhin mit einer direkt verfügbaren gegliederten ID kollidieren; unterschiedliche Ziele sind ein Fehler. Es gibt keine schrittweise Namespace- oder Dateipfadsuche.
- Importaliases dürfen direkt verfügbare vollständige IDs nicht mit einer anderen Bedeutung überdecken. Wäre dieselbe Schreibweise sowohl eine ID als auch eine Aliasqualifizierung mit anderem Ziel, ist das ein Fehler; Alias oder ID müssen geändert werden.
- Nur direkte Imports machen Namen verfügbar. Weitere bereitgestellte Packages und die Inhalte transitiver Modellabhängigkeiten werden nicht automatisch in den sichtbaren Namensraum einer Datei aufgenommen. Ein Zugriff darauf verlangt einen eigenen ausdrücklichen Import in der verwendenden Definitionsdatei.

Alle Referenzstellen verwenden diese Zielauflösung: kind, Property.target, Referenzwerte in spec und Körperlinks. kind und Property.target müssen Typdefinitionen bezeichnen; ein konkreter requester-Wert muss beispielsweise eine Container-Definition bezeichnen. Der erwartete Zieltyp wird erst nach der eindeutigen Auflösung geprüft und nicht als Suchheuristik verwendet.

Fehlende oder mehrdeutige Namen, unbekannte Importaliases und widersprüchliche Importangaben sind Fehler. Es gibt keine Suche außerhalb des eigenen Pakets und der direkten Imports dieser Datei, automatische Versionswahl, Dateipfadinterpretation oder Auflösung über ähnliche Namen. Nach der Auflösung werden kanonische Paket-/ID-Ziele verwendet. Kurze und qualifizierte Referenzen dürfen dasselbe Ziel bezeichnen, erzeugen aber keine verschiedenen Identitäten.

### 7.1 Verbindliche Schreibkonventionen und Kollisionsprüfung

**Nutzerentscheidung vom 15. Oktober 2026:** Groß-/Kleinschreibung erzeugt keine verschiedene Identität. Die Schreibkonventionen werden als Sprachregeln erzwungen. Verstöße werden abgewiesen; ein Hinweis auf die erwartete Form ist erlaubt, eine stille Korrektur oder Weiterverarbeitung der abweichenden Form nicht.

| Sprachbezeichner | Vorgeschriebene Form | Beispiele |
| --- | --- | --- |
| Paketname | Aus Ordnerpfad abgeleitete kleingeschriebene Komponenten, mit / als Trenner | markitect/core, example/connector/vocabulary |
| ID einer Kind-Definition | PascalCase | Kind, Property, Exchange, MqttService |
| ID einer Propertydefinition | camelCase | requester, maxAttempts, messageId |
| Effektiver Feldname, einschließlich ausdrücklichem spec.name | camelCase; gegliederte Namen komponentenweise | requester, timeoutSeconds, exchange.sender |
| ID eines konkreten Gegenstands | kleingeschriebenes kebab-case | api, list-documents |
| Importalias | PascalCase | Core, Connector |
| YAML-Dateiname | kleingeschriebenes kebab-case mit .yaml | exchange.yaml, max-attempts.yaml |
| Modell-ID | Kleingeschriebene Unicode-Abschnitte, durch . oder - getrennt; Ziffernstart erlaubt, siehe Abschnitt 8.1 | markitect-core, markitect.core, 2026-modell |
| Versionslabel | Dieselbe Zeichengrammatik wie Modell-ID; zusätzlich keine Bereichsschreibweisen oder latest | 1.0, 1.0.0-rc.1, 2026.10 |

Die Kategorie einer Definition folgt aus ihrer aufgelösten Typisierung, nicht aus der Groß-/Kleinschreibung eines geschriebenen kind-Werts. Eine Kind-Definition ist eine Definition des festen Meta-Kinds Kind; eine Propertydefinition ist eine Definition des festen Meta-Kinds Property. Akronyme folgen derselben Wortschreibweise: MqttService und messageId, nicht MQTTService oder messageID. Eine Referenz muss exakt die deklarierte Schreibweise des eindeutigen Ziels verwenden. kind: exchange statt kind: Exchange ist somit ein Fehler, auch wenn die Diagnose das gemeinte Ziel finden kann. Ebenso sind unpassend geschriebene Paketnamen, Aliase, Feldschlüssel und Dateinamen Fehler. Sprachschlüssel wie id, imports, package, as und minCount müssen exakt geschrieben werden.

Gegliederte lokale IDs bleiben einzelne Namen: Für jede durch einen Punkt getrennte Komponente gilt die Konvention der jeweiligen Definitionskategorie. Eine Property-ID exchange.sender bleibt damit möglich; ohne spec.name ist exchange.sender dann auch der vollständige literale Feldschlüssel, kein Zugriffsausdruck. Auch für solche gegliederten Feldnamen gilt camelCase je Komponente. Ein Kind darf zum Beispiel Messaging.Exchange heißen. Dadurch entsteht kein zusätzlicher Namespace.

Paket-/Aliasqualifier und lokale ID werden anhand der deklarierten Imports und tatsächlichen Ziele aufgelöst, nicht anhand ihrer Groß-/Kleinschreibung oder des erwarteten Zieltyps. Es gibt keine Auswahl durch längsten Prefix oder zufällige Reihenfolge.

Kollisionen werden unabhängig von Groß-/Kleinschreibung geprüft: Paket-IDs, Definitions-IDs innerhalb eines Pakets, Aliase innerhalb einer Definitionsdatei, effektive Feldnamen innerhalb eines Kind-/Objektvertrags sowie Dateinamen im jeweiligen Paket. Kind und Property bekommen dabei keine getrennten ID-Namensräume. Insbesondere sind Command und command im selben Paket ein Konflikt, selbst wenn beide einzeln die Konvention ihrer Kategorie erfüllen. Unterschiedliche Dateinamen lösen diesen Definitionskonflikt nicht.

Der Vergleich dient ausschließlich Identität, Konflikterkennung und Diagnose. Quelltext, gespeicherte kanonische Schreibweise und Referenzen werden nicht automatisch umgeschrieben. Eine falsche Referenz ist bei eindeutigem Ziel ein Schreibfehler; mehrere normalisiert passende Zielidentitäten bleiben mehrdeutig und werden nicht durch die erwartete Kategorie oder eine exakt geschriebene Teilmenge ausgewählt. Datenwerte, Enumwerte, body, title, Topicmuster und Operationskennungen werden nicht allgemein ohne Beachtung der Groß-/Kleinschreibung verglichen; dafür gilt der jeweilige Datenvertrag.

Die Unicode-Grundlage und der genaue Namensvergleich stehen in Abschnitt 7.2. Das zuvor begrenzte ASCII-Profil ist abgelöst. Die zulässigen Zeichen folgen dem dort gebundenen Unicode-Profil; Körperlink-Escapes und Quellenpositionen sind in Abschnitt 4.5 festgelegt. Das YAML-/Werteprofil steht in Abschnitt 5.1.

Die bisherige Exchange-Dateisammlung enthält noch den Definitionskonflikt Command/command. Der Nutzer hat eine Modellierungsdiskussion vor einer Umbenennung oder Zusammenfassung verlangt. Deshalb bleibt dieser Konflikt ausdrücklich sichtbar und muss vom aktuellen Prüfer abgewiesen werden. Die früheren bestandenen Berichte beziehen sich auf die vorherige, noch groß-/kleinschreibungssensitive Regel; sie sind keine Bestätigung des aktuellen Stands.

### 7.2 Unicode, Normalform und Namensvergleich

**Nutzerentscheidung vom 15. Oktober 2026:** Unicode ist die durchgängige Textgrundlage. Alle Dateien verwenden UTF-8. Unicode ist in body, title und String-/Enumwerten ebenso zulässig wie in Paketnamen, lokalen IDs, Aliasen, effektiven Feldnamen und YAML-Dateinamen. Die festen Sprachschlüssel und Strukturzeichen wie /, Punkt, Bindestrich, .yaml und Körperlinkbegrenzer behalten ihre genaue Schreibweise und Bedeutung.

**Versionsbindung des aktuellen Entwurfsprofils:** Die Zeichentabellen sind auf Unicode 15.1.0 festgelegt, entsprechend [UAX #31, Revision 39](https://www.unicode.org/reports/tr31/tr31-39.html). Diese Bindung verwendet die Tabellen des vorhandenen Python-3.13-Prüfers und ist keine Behauptung der neuesten Unicode-Version. Ein Wechsel der Tabellen ist eine ausdrückliche Änderung des Sprachprofils. Eine Implementierung darf nicht still die jeweilige Betriebssystem-, Laufzeit- oder Ländereinstellung verwenden. Der Beispielprüfer kontrolliert die Tabellenversion und weist eine Abweichung ab.

Für Bezeichner gilt folgendes Profil:

- Ein Namenswort beginnt mit XID_Start und wird mit XID_Continue fortgesetzt. Connector_Punctuation, einschließlich Unterstrich, ist ausgeschlossen; dies erhält die bereits gewählten Schreibkonventionen. Leerzeichen, Emoji und beliebige Symbole werden dadurch nicht zu Namenszeichen. In normalen Textwerten bleiben sie gemäß dem YAML-Profil aus Abschnitt 5.1 verwendbar.
- / gliedert Package-Namen; Punkte gliedern lokale IDs. Jede Komponente muss nichtleer sein. Ein einzelner Ordnername enthält keine Punkte. Package-Namen sind relative logische Pfade ohne führendes oder abschließendes /, leere Komponenten, . oder .., Backslash, Laufwerkspräfix oder URL-Dekodierung. Es findet keine Dateisystem-Pfadnormalisierung statt. Aliase sind einzelne PascalCase-Namen ohne Punkte. Bei konkreten IDs und Dateinamen trennt der ASCII-Bindestrich nichtleere kleingeschriebene Wörter; nach einem Bindestrich darf ein Teil auch mit einer Unicode-Dezimalziffer beginnen, etwa größe-2. Die erste Komponente beginnt weiterhin mit XID_Start.
- PascalCase und camelCase werden anhand des ersten Buchstabens mit Groß-/Kleinschreibung einer Komponente geprüft. Titlecase-Zeichen gelten dabei als große Initiale. Ohne solche Buchstaben entfällt diese Initialenvorgabe, etwa bei 文档. Für kleingeschriebene Namen müssen alle Buchstaben mit Groß-/Kleinschreibung klein sein. In PascalCase-/camelCase-Wörtern sind aufeinanderfolgende Großbuchstaben weiterhin ausgeschlossen; kombinierende Marken werden bei dieser Prüfung übersprungen. Somit bleiben Überweisung, gültigBis und MqttService passende Beispiele. Diese Kategorien folgen weiter aus der Typisierung.

**Strenge Autorenform:** Namen, Dateinamen und geschriebene Referenzen müssen bereits in [Unicode-Normalform NFC](https://www.unicode.org/reports/tr15/) vorliegen. Eine zerlegte Schreibweise, etwa U+0075 U+0308 anstelle von U+00FC für ü, wird mit Diagnose abgewiesen. Weder eine Quelldatei noch ihr dekodierter Name wird still korrigiert. Eine Referenz muss außerdem exakt die deklarierte Schreibweise des eindeutigen Ziels verwenden. Gleichwertigkeit im Kollisionsvergleich macht eine abweichende Autorenreferenz nicht gültig.

**Kollisionsschlüssel:** Für Namen verwenden wir Unicode Canonical Caseless Matching mit vollständigem Default Case Folding, kulturunabhängig und ohne türkische Sonderbehandlung. Präzise gilt K(s) = NFD(toCasefold(NFD(s))) gemäß Definition D145 des Unicode-Standards. Zwei Namen kollidieren genau dann, wenn ihre Schlüssel gleich sind, jeweils im bereits festgelegten Geltungsbereich: Paket, Definition, dateilokaler Alias, Feldvertrag oder Dateiauswahl. Der Vergleichsschlüssel ist ein internes Prüfmittel, keine neue gespeicherte Autorenform. Insbesondere kollidieren straße und strasse sowie Straße und Strasse. Auch eine entsprechende Referenzabweichung wird abgewiesen; der Prüfer wählt nicht eine exakt geschriebene Teilmenge aus mehreren kollidierenden Zielen.

NFC und dieser Vergleich bewirken keine allgemeine Umschrift: ä und ae bleiben verschieden. Eine zusätzliche NFKC-Kompatibilitätsfaltung ist nicht gewählt. body, title, String- und Enumwerte werden weder normalisiert noch ohne Beachtung der Groß-/Kleinschreibung verglichen; dort bleibt der dekodierte Text erhalten und es gilt der jeweilige Datenvertrag. Die NFC-Anforderung betrifft auch Feldschlüssel, aber nicht deren gewöhnliche Textwerte. Herkunft und ursprüngliche Dateibytes bleiben für Diagnosen verfügbar.

Der begrenzte Beispielprüfer wendet diese Regeln auf seine geladenen Pakete, Namen, Importe, Feldverträge und strukturierten Referenzen an. Seine Gegenproben umfassen Unicode-Namen, Schriften ohne Groß-/Kleinschreibung, kombinierende Marken, unveränderte Textdaten, NFC-Verstöße, die straße/strasse-Kollision in allen Namensbereichen und strenge Referenzschreibweise. Die Körperlink-Regeln aus Abschnitt 4.5 und die Leseform aus Abschnitt 5.1 besitzen eigene Gegenproben; ein allgemeiner Parser-Konformitätsnachweis wird dadurch nicht behauptet. Die Frage nach einer getrennten Command-Definition ist eine Vokabularfrage und erweitert diesen Sprachvertrag nicht.

Die gemeinsame Grammatik für Modell-IDs und Versionslabels steht in Abschnitt 8.1. Sie verwendet dieselben Unicode-Tabellen, NFC-Autorenform und Kollisionsschlüssel, erlaubt jedoch am Beginn jedes Abschnitts zusätzlich Unicode-Dezimalziffern. Diese Erweiterung gilt nicht automatisch für Packageordner, Definitions-IDs, Importaliase oder Dateinamen.

### 7.3 Ordner und Packages

**Nutzerentscheidung vom 16. Oktober 2026:** Das kanonische Modell wird wie Dokumentation in Ordnern strukturiert. Die ausdrücklich bestimmte Modellwurzel ist selbst kein Package und darf keine Definitionsdateien unmittelbar enthalten. Unterhalb dieser Wurzel ist jeder Ordner ein Package, auch ein Zwischenordner ohne eigene Definitionen. Jede Definition liegt somit in einem benannten Unterordner und gehört genau zum Package ihres unmittelbar enthaltenden Ordners. Unterordner gehören zu eigenen Packages; eine rekursive Erfassung macht deren Definitionen nicht zu Mitgliedern des Elternpackages. Das Package des Elternordners wird nicht geerbt.

**Gewählte Autorenkonvention:** Packages werden nach fachlichem Zusammenhang beziehungsweise Feature geschnitten (package by feature / vertical slices). Zusammengehörige Kinds, Properties und konkrete Gegenstände stehen gemeinsam beim jeweiligen Feature. Ihre Definitionsart ist kein Grund für getrennte Ordner wie kinds/, properties/ oder instances/. Ein weiterer Unterordner entspricht einem fachlichen Teilbereich; jede Definition behält ihre eigene Datei. Diese Konvention ist eine Entscheidung zur Modellorganisation, kein zusätzliches Core-Konzept und kein syntaktisches Verbot bestimmter Ordnernamen. Der Strukturprüfer leitet aus einem Ordnernamen keine fachliche Angemessenheit ab.

Der Package-Name entspricht dem nicht leeren Pfad des unmittelbar enthaltenden Ordners relativ zur Modellwurzel und wird nicht zusätzlich in einer package.yaml oder in jeder Definition gepflegt. Es gibt weder einen leeren Package-Namen noch ein Package namens . oder einen Ersatznamen aus der Modell-ID. Die gewählte Schreibweise verbindet die Ordnerkomponenten mit / und entspricht damit unmittelbar dem relativen Ordnerpfad:

```text
model/                                  # Modellwurzel; kein Package, keine Definitionen hier
  markitect/                            # markitect
    core/                               # markitect/core
      kind.yaml                         # (markitect/core, Kind)
      property.yaml                     # (markitect/core, Property)
  example/                              # example
    connector/                          # example/connector
      exchange.yaml                     # (example/connector, Exchange): Kind
      requester.yaml                    # (example/connector, requester): Property
      list-documents.yaml               # (example/connector, list-documents): konkreter Gegenstand
```

Dieser Ausschnitt veranschaulicht die fachliche Gruppierung. Der ältere ausführbare Exchange-Durchstich trennt noch vocabulary/ und project/, um Imports und Referenzformen durchzuspielen; diese Aufteilung ist keine Empfehlung für die Autorenorganisation und kein Bestandteil der Sprachsyntax.

Ein Ordnername entspricht genau einer bisherigen kleingeschriebenen Package-Komponente aus Abschnitt 7.2; ein Punkt im einzelnen Ordnernamen erzeugt keine zusätzlichen Komponenten und ist in dieser Form unzulässig. Groß-/Kleinschreibung und Unicode-Kollisionen werden unabhängig vom Verhalten des Dateisystems geprüft, einschließlich straße/strasse. Namen müssen bereits in NFC und in der vorgeschriebenen Form vorliegen; keine automatische Korrektur.

Die Definitionen werden aus ihren jeweiligen Ordnern erfasst. Es gibt keine gesonderte definitionFiles-Liste pro Package und keinen zusätzlichen definitions/-Pflichtordner. Die lokale id steht weiterhin ausdrücklich in jeder Definitionsdatei; sie wird nicht aus dem Dateinamen abgeleitet. Umbenennen einer Datei im selben Ordner erhält ihre Identität. Verschieben einer Datei in einen anderen Packageordner oder Umbenennen/Verschieben eines Packageordners ändert die betroffenen qualifizierten Identitäten und verlangt angepasste Referenzen. Ein anderer physischer Speicherort der gesamten Modellwurzel erhält die relativen Package-Namen.

Die Ordnerhierarchie erzeugt keine impliziten Imports, Vererbung, Enthaltenheitsbeziehungen zwischen Definitionen oder fachlichen Beziehungen. Eltern-, Kind- und Geschwisterpackages müssen in jeder verwendenden Datei ausdrücklich importiert werden. Ein Package hat keine eigene Version, Abhängigkeitsliste oder Veröffentlichungsdatei. Modell bezeichnet die gemeinsame Versionierungs- und Verteilungseinheit und kann mehrere Packages enthalten. Abschnitt 8.1 bestimmt Modellidentität, Wurzel und Manifestform. Die Modellidentität erzeugt keinen automatischen Package-Präfix und keinen zusätzlichen Teil der Definitionsidentität. Abschnitt 8.2 bestimmt das Lockformat; Abschnitt 8.3 bestimmt die kanonischen Digestbytes.

Die Modellwurzel ist ausdrücklich in model.yaml bestimmt. Der aktuelle Durchstich verwendet für beide Modelle jeweils root: model und legt alle Definitionen in benannte Unterordner. Definitionen unmittelbar in dieser Wurzel sind Sprachfehler; der Prüfer meldet sie als fehlgeschlagen und verschiebt oder korrigiert die Dateien nicht automatisch. Das ausdrücklich gebundene model.yaml und die danebenliegende model-lock.yaml sind Modellmetadaten und keine Definitionsdateien; auch bei root: . erzeugen sie kein Wurzel-Package. Abschnitt 8.1 bestimmt die physische Pfadschreibweise, das Verbot von Umleitungen und überlappenden Modellwurzeln.

## 8. Dateilokale Imports und gebundene Modelle

**Gewählt:** Imports stehen in der jeweiligen Definitionsdatei und gelten ausschließlich dort. Package bezeichnet das aus dem Ordnerpfad abgeleitete Namensgebiet aus Abschnitt 7.3. Es ist keine Versionierungs- oder Verteilungseinheit. Die frühere package.yaml mit eigener Package-Version und dependencies ist abgelöst. Die gemeinsame Manifestform model.yaml steht in Abschnitt 8.1.

**list-documents.yaml im Ordner model/example/connector/project/:**

```yaml
imports:
  - package: example/connector/vocabulary
    as: Connector

id: list-documents
kind: Exchange
body: Die API fordert Dokumentmetadaten beim Connector an.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

Die Datei importiert das bereitgestellte Package example/connector/vocabulary und kann deshalb Exchange unmittelbar verwenden. api, onprem, list-documents-command und die konkreten Channels sind eigene Projektdefinitionen. Ein zusätzlicher Core-Import ist für diese Datei nicht erforderlich, weil sie keinen Core-Namen unmittelbar referenziert. exchange.yaml und jede seiner Propertydateien importieren markitect/core dagegen selbst, um Kind beziehungsweise Property verwenden zu können.

Das optionale Hüllenfeld imports enthält die ausdrücklich in dieser Datei sichtbaren fremden Packages. Ohne imports oder bei imports: [] sind ausschließlich Definitionen des eigenen Packages sichtbar. Ein unbekanntes oder im gebundenen Eingabesatz nicht bereitgestelltes Package ist ein Fehler. Ein fremdes Package muss aus dem eigenen Modell oder einer ausdrücklich deklarierten direkten Modellabhängigkeit stammen. Transitive Modelle werden zur Vertragsprüfung geladen; ein direkter Zugriff auf deren Packages verlangt eine eigene direkte Abhängigkeit im verwendenden Modellmanifest. Imports wählen weder eine neue Version noch einen physischen Dateisystempfad. Die Bereitstellung eines externen Modells und seine Version öffnen allein keine Namen; umgekehrt ist ein Import keine Installationsanweisung.

**Gewählte Aliasform:** Jeder Import ist ein Objekt mit erforderlichem package und optionalem as. Eine String-Kurzform wird nicht parallel geführt. as deklariert einen dateilokalen Namen in PascalCase und verändert weder Package-Identität noch verfügbare Definitionenmenge.

```yaml
imports:
  - package: markitect/core
    as: Core
  - package: example/connector/vocabulary
    as: Connector
```

Ohne as ist ein Package nur unter seinem vollständigen Package-Namen qualifizierbar. Mit as sind sowohl vollständiger Name als auch kurzer Alias zulässig. Die vollständigen lokalen IDs des importierten Packages werden in beiden Fällen unmittelbar verfügbar: Kind, Core.Kind und markitect/core/Kind können dasselbe Ziel bezeichnen. Ein Alias macht den Import nicht zu einem nur qualifiziert sichtbaren Import.

Aliase gelten nur in der deklarierenden Definitionsdatei. Imports anderer Dateien, die Imports einer referenzierten Kind-/Propertydefinition, Elternpackages und transitive Modellabhängigkeiten erweitern diese Sicht nicht. Referenzwerte und Körperlinks werden immer im Dateikontext aufgelöst, in dem sie geschrieben sind. Auch mehrere Packages derselben Verteilungseinheit brauchen ihre eigenen dateilokalen Imports.

Doppelte Aliase, doppelte Imports desselben Packages und verdeckte verschiedene Ziele werden unabhängig von Groß-/Kleinschreibung abgewiesen. Core.Kind und core.Kind sind keine verschiedenen Adressen; bei deklariertem Core ist core.Kind ein Fehler. Importauswahl, volle Package-Qualifizierung und Kurzalias dürfen zusammen keine unterschiedlichen Zielidentitäten für dieselbe normalisierte Referenzschreibweise erzeugen. Es gibt keinen Vorrang nach exakter Schreibweise, Ziel-Kind oder Prefixlänge. Abschnitt 7.2 besitzt das Unicode-Profil.

**Inhaltsbindung bleibt erforderlich:** Externe wiederverwendbare Modelle werden mit genauer Version und konkretem Inhaltsdigest gebunden. Versionsbereiche, latest und stiller Inhaltsaustausch unter einer gebundenen Version sind nicht vorgesehen. Der gebundene Eingabesatz muss jede kanonische Package-/ID-Identität eindeutig bereitstellen; mehrere Imports oder Aliase umgehen keinen Konflikt. Das lokale Modell wird durch seinen festen Quell-/Modellstand gebunden; jede lokale Änderung verlangt keine neue Veröffentlichungsnummer.

Das [vollständige Exchange-Beispiel](exchange-example/README.md) verwendet jetzt zwei model.yaml-Manifeste, explizite lokale Bereitstellung und aus den jeweiligen Modellwurzeln abgeleitete Packages. Die synthetischen Versionen behaupten keine veröffentlichten Bibliotheken. Sein Prüfer kontrolliert Manifeste, genaue lokale Versionsbindungen und Package-Zugehörigkeit; er prüft auch die Lockstruktur und deren Manifestbindungen, und die echten kanonischen Inhaltsdigests gültiger externer Modelle. Die [historische Beispieldatei](identities/project-c.yaml) und frühere Prüfberichte bleiben Belege ihres damaligen Formats und werden nicht umgeschrieben.

### 8.1 model.yaml: Identität, Wurzel und Abhängigkeiten

**Nutzerentscheidung vom 16. Oktober 2026:** Jedes zusammengehörige Modell besitzt ein Manifest mit dem festen Dateinamen model.yaml. Manifest bezeichnet die Aufgabe dieser Datei; Modell ist die gemeinsame Versionierungs- und Verteilungseinheit für mehrere Packages. model.yaml ist Bereitstellungsmetadatum, keine zusätzliche Definition und kein neuer privilegierter Core-Kind. Jede eigentliche Definition behält ihre einheitliche Hülle.

| Feld | Gewählter Vertrag |
| --- | --- |
| formatVersion | Erforderliche Ganzzahl; bindet Manifestformat und Sprachvertrag. Der aktuelle Entwurf verwendet 1. Unbekannte Fassungen werden nicht still als 1 interpretiert. |
| id | Erforderliche stabile, nichtleere Modellidentität; unabhängig von Checkout-/Installationsordner, Speicherort, Modellwurzel, Package-Namen und Version. |
| version | Genaue textuelle Veröffentlichungsfassung aller enthaltenen Packages. Bei lokaler unveröffentlichter Arbeit optional; für Veröffentlichung und als bereitgestellte Abhängigkeit erforderlich. |
| root | Erforderlicher nichtleerer relativer Verzeichnispfad, bezogen auf den Ordner der jeweiligen model.yaml. Es gibt keinen geratenen Standardordner. |
| dependencies | Erforderliches Mapping von direkt benötigter Modell-ID auf genaue textuelle Version; ohne Abhängigkeiten {}. Keine Packages, Importaliase, Versionsbereiche oder latest. |

Die Hülle ist geschlossen: unbekannte oder falsch geschriebene Felder sind Fehler. Eine Abhängigkeit muss exakt die deklarierte Modell-ID verwenden. Modell-IDs und Versionslabels sind YAML-Strings; numerische Werte werden nicht konvertiert. Die gewählte physische root-Pfadschreibweise steht weiter unten in diesem Abschnitt.

**Gewählte gemeinsame Zeichengrammatik für Modell-IDs und Versionslabels:**

```text
Label               = Abschnitt (("." | "-") Abschnitt)*
Abschnitt           = Startzeichen Fortsetzungszeichen*
Startzeichen        = XID_Start | Unicode-Dezimalziffer (General_Category=Nd)
Fortsetzungszeichen = XID_Continue
```

Das Unicode-Profil aus Abschnitt 7.2 gilt mit folgenden Präzisierungen:

- Connector_Punctuation, einschließlich _, ist überall ausgeschlossen. Zeichen mit Groß-/Kleinschreibung müssen klein sein; Schriften ohne Groß-/Kleinschreibung bleiben zulässig. Alle Labels müssen bereits NFC sein. Verstöße werden abgewiesen, nicht korrigiert.
- ASCII-Punkt und ASCII-Bindestrich trennen nichtleere Abschnitte. Es gibt keine führenden, abschließenden oder unmittelbar aufeinanderfolgenden Trennzeichen, auch nicht gemischt wie in 1.-0. Leerraum, /, Backslash, Emoji und sonstige nicht zum Profil gehörende Zeichen sind unzulässig.
- Dieselbe Grammatik gilt für Modellidentität, Abhängigkeitsschlüssel, veröffentlichte Version, Versionsanforderung und Lockbindung. Auch Modell-IDs dürfen mit Ziffern beginnen oder Punkte enthalten. Punkte erzeugen in diesen Labels weder Packages noch Namespaces; Bindestriche tragen ebenfalls keine zusätzliche Semantik.
- Gültig sind beispielsweise markitect-core, überweisungen, 文档, markitect.core, 2026-modell, 1.0, 1.0.0-rc.1 und ٢٠٢٦.١. Ungültig sind beispielsweise Markitect-Core, über_weisungen, modell/name, 1..0 und 1.0.0-.
- Für Versionslabels bleiben latest und Bereichsschreibweisen ausgeschlossen. Die bisher abgewiesenen punktgetrennten Bereichsmarker x, etwa x oder 1.x, bleiben reserviert; *, ^, ~, Vergleichs- und Bereichsoperatoren sind ohnehin keine Labelzeichen. Diese Reservierung betrifft nicht Modell-IDs.

Der vollständige Unicode-Kollisionsvergleich aus Abschnitt 7.2 gilt für Modell-IDs innerhalb der bereitgestellten Modellmenge und für Versionslabels innerhalb desselben Modells. straße und strasse dürfen dort keine verschiedenen Identitäten bezeichnen. Unterschiedliche Modelle dürfen dasselbe Versionslabel verwenden. Anforderungen und Lockeinträge müssen trotzdem exakt die deklarierte Autorenform verwenden; eine kollidierende Schreibweise ist keine gültige alternative Referenz. Der erste Sprachumfang bindet weiterhin nur eine Fassung je Modell-ID; die Kollisionsprüfung eröffnet keine Mehrversionsauswahl.

**Gewählte Versionssemantik:** Eine Version ist ein exaktes Textlabel. Manifest, direkte Abhängigkeitsanforderung und Lock müssen denselben dekodierten Text verwenden. Die Sprache interpretiert seine Bestandteile nicht als Zahlen, ergänzt keine fehlenden Komponenten und leitet weder Versionsreihenfolge noch Kompatibilität daraus ab. "1.0" und "1.0.0" bezeichnen unterschiedliche Fassungen; auch "1.0.1" erfüllt eine Anforderung auf "1.0.0" nicht. Versionsbereiche und latest bleiben verboten. Der konkrete Inhaltsstand wird zusätzlich durch den kanonischen Digest gebunden. Die gemeinsame Zeichengrammatik oben ist gewählt; sie definiert keine SemVer-Grammatik oder Kompatibilitätsauswertung.

**model.yaml**

```yaml
formatVersion: 1
id: connector-example
root: model
dependencies:
  markitect-core: "1.0.0"
```

Dieses lokale Modell enthält die Packages example/connector/vocabulary und example/connector/project samt ihren Zwischenordnern. Seine eigene version ist absichtlich ausgelassen: Lokale Änderungen werden durch den festen Quell-/Modellstand gebunden und verlangen keine neue Veröffentlichungsnummer. Eine version: "1.0.0" könnte für eine veröffentlichte Fassung zusätzlich gesetzt werden.

**dependencies/core/model.yaml**

```yaml
formatVersion: 1
id: markitect-core
version: "1.0.0"
root: model
dependencies: {}
```

Das zweite Manifest gehört zum separat bereitgestellten Core-Modell. Sein Ordner model/markitect/core/ stellt das Package markitect/core mit neun Definitionen bereit. dependencies/core/ ist ausschließlich der gewählte lokale Ablageort dieses Durchstichs, kein zusätzliches Autorenfeld, Registryname oder bereits gewähltes Lockformat. Die Bereitstellung beider Manifestdateien ist expliziter Prüfeingang. Paketnamen werden jeweils unter der eigenen root bestimmt und weder mit dem Installationspfad noch automatisch mit der Modell-ID präfigiert.

Die Abhängigkeit connector-example → markitect-core verlangt die exakte Identität und Version der bereitgestellten Fassung. Ein fehlendes Modell, abweichende Schreibweise, widersprüchliche Versionen direkter/transitiver Anforderungen oder ein Katalog mit mehreren Kandidaten derselben Modell-ID dürfen nicht still aufgelöst werden. Der erste Sprachumfang bindet pro Modell-ID eine Fassung; Mehrversionsimporte sind nicht vorgesehen. Wiederverwendbare externe Inhalte brauchen zusätzlich eine eindeutige Inhaltsbindung; das gewählte Lockformat steht in Abschnitt 8.2; die kanonischen Digestbytes stehen in Abschnitt 8.3. Der Manifestprüfer belegt nur die vorhandene lokale Zuordnung und Versionsgleichheit, keine Veröffentlichung oder bereits verifizierten Inhaltsdigest.

Ein mit Definitionen belegtes Package gehört im gebundenen Eingabesatz genau einem Modell. Zwei Modelle dürfen dasselbe Package auch bei unterschiedlichen lokalen IDs nicht aufteilen. Gemeinsame Zwischenordner bleiben möglich: getrennte Modelle mit markitect/core und markitect/http können beide den definitionslosen Zwischenordner markitect enthalten. Sobald ein Package Definitionen enthält, stammen diese aus genau einem Modell; ein anderswo lediglich als leerer Zwischenordner vorkommender gleicher Name liefert keine weiteren Definitionen. Unterschiedlich geschriebene, im Unicode-Vergleich kollidierende Package-Namen bleiben Fehler. Keine Lade- oder Importreihenfolge entscheidet über die Zugehörigkeit.

Imports bleiben dateilokal. Sie machen ausschließlich die tatsächlich importierten Packages sichtbar. Ein Package aus demselben Modell benötigt einen Import, sobald es nicht das eigene Package der Datei ist. Die Verfügbarkeit eines transitiven Modells eröffnet keinen direkten Import ohne eigene direkte Modellabhängigkeit. model.yaml wird weder aus Imports noch aus Prosa ergänzt; es ist keine Installations- oder Ausführungsanweisung.

Ein anderer physischer Speicherort des gesamten Modells, ein anderer Installationsordner oder eine identitätserhaltende Verlagerung seiner Wurzel ändern seine Modell-ID und die relativen Package-Namen nicht. Eine ausdrückliche Änderung der Modell-ID ändert die Adresse der Modellabhängigkeit, aber erzeugt nicht automatisch neue Package-/Definitionsidentitäten. root bestimmt, welche Dateien unter welchen relativen Package-Namen geladen werden; eine Änderung ist anhand dieses neuen Bestands zu prüfen.

**Gewählte Azyklizitätsregel für Modellabhängigkeiten:** Der gerichtete Graph der model.yaml-dependencies muss azyklisch sein. Knoten sind die eindeutig gebundenen Modell-IDs, Kanten die ausdrücklich deklarierten direkten Anforderungen. Im vollständigen erreichbaren Modellumfang darf kein Abhängigkeitsweg zum Ausgangsmodell zurückführen. Das verbietet Selbstabhängigkeiten als Zyklen der Länge eins ebenso wie wechselseitige oder längere Zyklen. Auch exakt passende Versionen oder Digests machen einen solchen Zyklus nicht gültig. Mehrere Modelle dürfen dieselbe Abhängigkeit verwenden, sofern die gemeinsame genaue Versions- und Inhaltsbindung passt.

Ein Modellzyklus ist ein Sprachfehler und liefert failed, keine bloß unvollständige Prüfung. Für den betroffenen ungültigen Modellumfang gibt es weder kanonische Bytes noch Digest. Die Regel benötigt keine Sonderbehandlung für Selbstabhängigkeiten. Fachlich untrennbar wechselseitig abhängige Teile können zu einem gemeinsamen Modell mit mehreren fachlichen Packages gehören. Gewöhnliche Definitionsreferenzen und Package-Imports innerhalb desselben Modells dürfen weiterhin Zyklen bilden; die Objektvertragsregeln aus Abschnitt 5.3 gelten zusätzlich. Aus Prosa oder bloßen Dateilinks werden keine dependencies ergänzt.

Die Azyklizitätsregel ist eine bewusst gewählte Grenze zwischen gemeinsam versionierten Verteilungseinheiten. Sie folgt nicht zwingend aus der Digestform, die fremde Inhalte gerade nicht rekursiv einbettet. Die Prüfung kontrolliert den erreichbaren Modellgraphen deterministisch; gemeinsame transitive Bibliotheken sind keine Zyklen.

**Gewählte physische Wurzelpfadregeln:**

- root ist entweder . oder ein nichtleerer relativer Ordnerpfad, bezogen auf den Ordner der jeweiligen model.yaml. Das Ziel muss bereits als Verzeichnis existieren; fehlende Ziele und Dateien statt Verzeichnissen sind Fehler. . bezeichnet ausschließlich den Manifestordner selbst.
- ASCII-/ ist auch unter Windows der einzige Trenner. Jede relative Komponente folgt der bestehenden kleingeschriebenen Unicode-Packageordnergrammatik aus Abschnitt 7.2: XID_Start am Beginn, danach XID_Continue, keine Connector_Punctuation. Die neue Modell-ID-/Versionslabelgrammatik erweitert diese Ordnergrammatik nicht. Insbesondere sind Punkte, Bindestriche und Ziffernstarts in einzelnen root-Ordnerkomponenten nicht zusätzlich erlaubt.
- Es gibt keine führenden oder abschließenden /, leeren Komponenten, eingebetteten . oder .., Backslashes, Laufwerkspräfixe oder absoluten Pfade. Die Autorenform wird vor einer Pfadinterpretation geprüft; ./model, docs//model und model/ werden nicht still normalisiert. Es gibt keine Variablen-, Home-, URL- oder sonstige Ersetzung.
- Die geschriebenen Komponenten müssen bereits NFC sein und exakt den tatsächlichen Verzeichnisnamen entsprechen, einschließlich Groß-/Kleinschreibung und Unicode-Schreibweise. Ein auf einem toleranten Dateisystem auffindbares Model ersetzt ein geschriebenes model nicht. Mehrere im Unicode-Kollisionsvergleich passende Verzeichnisse bleiben ein Fehler; die exakt geschriebene Teilmenge erhält keinen Vorrang.
- Symlinks, Junctions und vergleichbare Reparse-Umleitungen im relativen Wurzelpfad oder im erfassten Modellbaum sind verboten und werden nicht verfolgt. Auch die ausdrücklich geladenen Manifest- und Lockdateien werden nicht über solche Umleitungen gelesen. Der physische Manifestordner bildet den Bezugspunkt; der absolute Ablagepfad darüber erhält keine Package-Namenskonvention.
- Geladene Modellwurzeln dürfen weder dasselbe physische Verzeichnis bezeichnen noch ineinander liegen. Die Prüfung erfolgt für die gebundene Modellmenge vor der Definitionserfassung. Bei root: . müssen weitere gebundene Modelle deshalb außerhalb dieses Modellbaums liegen. Gemeinsame logische, leere Packagepräfixe aus getrennten physischen Wurzeln bleiben möglich.

Gültige Formen sind ., model, docs/model und überweisung/文档, sofern die exakt geschriebenen Verzeichnisse vorhanden sind. Ungültig sind beispielsweise ./model, model/, docs//model, ../shared/model, Model und C:\\models\\connector. Die absoluten Speicherorte der Modelle bleiben frei; eine identitätserhaltende physische Verlagerung verändert weder Package-Namen noch kanonische Digestbytes. Die Wurzel selbst bildet weiterhin kein Package und darf keine Definitionsdateien unmittelbar enthalten (Abschnitt 7.3).

Der begrenzte Prüfer kontrolliert diese Regeln einschließlich tatsächlicher Verzeichnisnamen, Unicode-Kollisionen, Überlappungen und einer echten Junction unter Windows beziehungsweise eines Symlinks auf anderen unterstützten Hosts. Unzulässige Wurzelpfade und Umleitungen sind fehlgeschlagene Sprachprüfungen, keine als nicht abgeschlossen behandelten Bereiche. Ein vollständiger Modellkonformitäts- oder Veröffentlichungsnachweis wird damit nicht behauptet.

### 8.2 model-lock.yaml: vollständige externe Inhaltsbindung

**Nutzerentscheidung vom 16. Oktober 2026:** Das generierte Lock wird als model-lock.yaml neben dem Einstiegsmanifest gespeichert und zusammen mit dem Modell versioniert. Es bindet sämtliche direkten und transitiven externen Modellabhängigkeiten genau einmal. Die eigene lokale Modellfassung wird durch den aktuellen Quell-/Modellstand gebunden und nicht als eigener externer Eintrag im Lock wiederholt.

Die geschlossene Form besteht aus genau formatVersion und models. Die aktuelle formatVersion ist die Ganzzahl 1. models ist ein Mapping von Modell-ID auf ein Objekt mit genau version und digest. Ohne externe Abhängigkeiten ist models: {} die vollständige Tabelle. Jeder Schlüssel ist die exakt geschriebene NFC-Modell-ID; der Unicode-Kollisionsvergleich aus Abschnitt 7.2 gilt auch hier. version ist die genaue textuelle Fassung. digest hat die Form sha256: mit genau 64 nachfolgenden kleingeschriebenen Hexadezimalziffern. Unbekannte Felder, Schreibabweichungen, fehlende Angaben und Kollisionen sind Fehler.

Das gespeicherte Beispiel enthält jetzt den echten, gegen Abschnitt 8.3 berechneten Core-Digest:

**model-lock.yaml**

```yaml
formatVersion: 1
models:
  markitect-core:
    version: "1.0.0"
    digest: "sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1"
```

Die [kanonische Core-Datei](exchange-example/canonical/core.json) enthält genau 3769 Bytes ohne BOM und ohne abschließenden Zeilenumbruch. Der eigene Core-Inhalt ist einschließlich seiner festen Selbstbeschreibung und Abhängigkeiten gültig und wurde vor der Kodierung geprüft. Der offene Command/command-Konflikt des anderen Modells macht diese unabhängige Core-Bindung nicht ungültig; für das fehlerhafte Connector-Modell wird dagegen kein Digest ausgegeben. Frühere Berichte mit einem Platzhalter bleiben historische Befunde ihres damaligen Stands.

**Vollständigkeit:** Aus den direkten Anforderungen des Einstiegsmanifests und den gebundenen model.yaml-Manifesten der Abhängigkeiten muss genau die externe Locktabelle entstehen. Fehlende, überzählige oder widersprüchliche Einträge sind Fehler. Es gibt keine zusätzliche manuelle Liste der direkten Abhängigkeiten und keine zweite Beschreibung der Graphkanten im Lock. Pro Modell-ID gilt eine Fassung; weder Importaliase noch Lade- oder Tabellenreihenfolge lösen Versions- oder Inhaltskonflikte auf.

**Prüfung:** Modell-ID, genaue Version und berechneter Inhaltsdigest müssen gleichzeitig passen. Der Prüfer kontrolliert alle erreichbaren Manifestanforderungen und anschließend jede Inhaltsbindung. Andere Inhalte unter derselben gebundenen Version werden abgewiesen. Ein Prüflauf schreibt, ergänzt oder aktualisiert das Lock nicht automatisch. Ein fehlendes Lock wird nicht durch eine still erzeugte neue Bindung ersetzt. Änderungen der Abhängigkeiten verlangen eine erneute Prüfung der vollständigen Bindung; lokale Definitionsänderungen allein verlangen kein neues Lock. Die Root-Locktabelle bestimmt die gemeinsame externe Bindung des ausgewerteten Modellstands; eine etwa mitgelieferte weitere Lockdatei eines Abhängigkeitsmodells ersetzt oder erweitert diese Tabelle nicht still.

**Digestumfang:** Ein Modelldigest bindet ausschließlich den eigenen Modellinhalt einschließlich eigener Modellmetadaten, deklarierter direkter Abhängigkeitsanforderungen, Packages und Definitionen. Er enthält keine rekursiv eingebetteten Digests fremder Modellinhalte. Die vollständige konkrete Bindung der transitiven Inhalte trägt die Locktabelle. Das vermeidet rekursive Digestdefinitionen; gewöhnliche Definitionsreferenzen bleiben Referenzen auf kanonische Identitäten und werden nicht unendlich textuell entfaltet. Die genaue Projektion, Defaults, Reihenfolgen und Bytekodierung stehen in Abschnitt 8.3.

**Bereitstellung:** Installationspfade, Ablageorte und Downloadadressen sind Bereitstellungs-/Provenienzeingaben und keine Lockfelder. Die bereits gewählte Modell-ID wird nicht um einen Speicherort ergänzt. Die Bereitstellung liefert Inhalte, deren Identität, Version und Digest geprüft werden müssen; der Lockeintrag allein ist keine Installationsanweisung.

Der Prüfer kontrolliert Hülle, Versionen, genaue Modellmenge und tatsächlich berechnete externe Modelldigests. Künstliche Vergleichsdigests bleiben auf isolierte Gegenproben beschränkt. Der Bericht trennt Manifestprüfung, Definitionsprüfung, kanonische Modellprojektion und Lockprüfung: Der Core und seine Inhaltsbindung bestehen, während der gesamte Definitionsgraph wegen Command/command weiterhin abgewiesen wird. Dateihashes bleiben zusätzlich Byteprovenienz ihrer jeweiligen Eingaben und sind nicht mit kanonischen Modelldigests gleichzusetzen.

### 8.3 Kanonische Bytes und SHA-256: model-digest/1

**Nutzerentscheidung vom 16. Oktober 2026:** Jeder Modelldigest ist SHA-256 über die nachfolgenden kanonischen Bytes. Die Kodierung beschreibt den ausgewerteten eigenen Modellinhalt. Vor jeder Projektion werden das ausgewählte Modell und seine vollständige Abhängigkeitshülle validiert: Manifeste, Schreibweisen/Kollisionen, feste Core-Selbstbeschreibung, Typverträge, Werte, Kardinalitäten und Körperlinks. Auch beim Laden festgestellte Fehler dürfen nicht durch eine spätere Map-Darstellung verloren gehen. Eine ungültige Fassung erhält keinen Digest; eine nicht abgeschlossene Prüfung ist kein gültiger Digestnachweis. Ein gültiges Abhängigkeitsmodell kann unabhängig von einem fehlerhaften, es verwendenden lokalen Modell geprüft und gehasht werden.

**Projektion.** Der kodierte Baum hat genau drei Felder:

| Feld | Inhalt |
| --- | --- |
| format | Der feste String markitect-model-digest/1 als Kennung dieser Projektion/Kodierung. |
| model | Eigenes formatVersion, id, optional version und die vollständigen direkten dependencies aus dem eigenen Manifest; root ist ausgeschlossen. |
| packages | Mapping eigener belegter Package-Namen auf ein Mapping ihrer lokalen Definitions-IDs auf die nachfolgenden Definitionsdatensätze. |

Ein Definitionsdatensatz enthält genau kind, body, bodyLinks, spec und bei gesetzter Autorenangabe title. Package und id werden durch die beiden Mappingschlüssel getragen und nicht nochmals im Datensatz wiederholt. kind ist die aufgelöste Referenzform aus der nachfolgenden Tabelle. body und title sind die dekodierten unveränderten Unicode-Strings. bodyLinks ist immer eine Liste kanonischer Zielreferenzen in der Reihenfolge der aktiven Vorkommen im body; Wiederholungen bleiben erhalten. Ein Ziel wird als ["reference", package, id] kodiert. Anzeigetexte und die geschriebene Linknotation bleiben bereits im unveränderten body enthalten. Byte-/Zeilenfundstellen werden nicht aufgenommen. spec ist ein Mapping effektiver Feldnamen auf die typisierten kanonischen Werte.

| Werttyp | Exakte Baumform |
| --- | --- |
| string | ["string", unveränderterString] |
| enum | ["enum", unveränderterErlaubterString] |
| boolean | ["boolean", true] beziehungsweise ["boolean", false] |
| integer | ["integer", dezimaleGanzzahlAlsString] |
| number | ["number", dezimalerKoeffizientAlsString, dezimalerExponentAlsString] |
| reference | ["reference", exaktGeschriebenesPackage, vollständigeLokaleID] |
| object | ["object", MappingDerEffektivenFelderAufTypisierteWerte] |
| Mehrfachwert | ["list", ListeDerEinzelwertformen] |

Die Typform folgt aus dem aufgelösten Feldvertrag, nicht aus YAML-Schreibweise oder Feldnamensähnlichkeit. Ganzzahlstrings verwenden 0 oder eine optionale negative Vorzeichenangabe gefolgt von Ziffern ohne führende Nullen; -0 wird zu 0. number bedeutet Koeffizient × 10 hoch Exponent. Der Koeffizient hat weder führende noch abschließende Nullen; entfernte abschließende Nullen werden dem Exponenten zugeschlagen. Das Vorzeichen steht nur bei einem negativen, von null verschiedenen Koeffizienten. Null ist stets ["number", "0", "0"]. Der Exponent ist ein Ganzzahlstring derselben Form. Dadurch ergeben 42, 42.0 und 4.2e1 bei einer number-Property dieselbe Form ["number", "42", "0"]. Bei integer bleibt ein Dezimalwert unzulässig. Es gibt keine binäre Gleitkommazahl oder kontextabhängige Dezimalrundung; große Exponenten werden nicht textuell als riesige Ziffernfolgen entfaltet.

**Defaults und Listen.** Fehlende optionale Listen werden gemäß Abschnitt 5.2 als leere Listen dargestellt, auch innerhalb von object-Werten. Die für einen konkreten Meta-Typ verbotenen Felder werden nicht durch Defaults eingefügt. Fehlendes spec.name einer Propertydefinition wird als vollständige lokale Property-ID eingesetzt; ein ausdrücklich gleicher Wert erzeugt dieselben Bytes. Andere fehlende optionale Einzelwerte werden nicht erfunden. Fehlendes title oder version bleibt von einem vorhandenen Wert unterscheidbar. Gewöhnliche Wertlisten behalten ihre Reihenfolge und Wiederholungen. Die festen Core-Deklarationslisten aus Abschnitt 5.2 werden nach den kanonischen JSON-Bytes ihrer einzelnen Elemente sortiert. Ein gewöhnliches Datenfeld mit gleichem Namen erhält diese Sortierung nicht. Die feste Core-Selbstbeschreibung vergleicht ausdrücklich gesetzte ID-Standardnamen ebenfalls anhand ihres wirksamen Werts.

**Referenzen und Text.** Strukturelle Referenzen werden auf das kanonische Package-/ID-Paar aufgelöst. Ihre kurze, voll qualifizierte oder aliasqualifizierte Autorenform verändert die Bytes nicht. imports und Aliase werden nach erfolgreicher Auflösung nicht als eigene Inhaltsfelder gespeichert; ein unbenutzter, weiterhin gültiger Import beeinflusst den Digest nicht. Entfernte Imports, Mehrdeutigkeiten oder Schreibfehler werden trotzdem vor der Projektion abgewiesen. Im body bleibt die geschriebene Linknotation dagegen Teil des Texts: Eine Aliasänderung im body ändert die Bytes, auch bei gleichem Ziel. Umgekehrt binden die zusätzlichen bodyLinks-Ziele eine durch andere Imports veränderte Auflösung bei unverändertem body. Körpertexte werden nicht gerendert, neu formatiert, getrimmt, casegefaltet oder nachträglich Unicode-normalisiert. Dasselbe gilt für title und gewöhnliche String-/Enumwerte. YAML-Kommentare, Einrückung, Quote-Auswahl und die Quellreihenfolge von Mappings beeinflussen nur die Autorenbytes, nicht den dekodierten Baum.

**JSON-/UTF-8-Kodierung.** Die folgenden Regeln gelten rekursiv für den gesamten Baum:

- JSON-Objektschlüssel werden aufsteigend lexikographisch nach ihren UTF-8-Bytes als vorzeichenlosen Bytes sortiert, unabhängig von Spracheinstellung oder Dateisystem. Schreibweisen bleiben exakt erhalten; der Kollisionsschlüssel ist kein gespeicherter Name.
- Zwischen Tokens steht kein Formatierungsleerraum. Objekt-/Arraytrenner sind exakt die ASCII-Zeichen {}, [], : und Komma. true und false sind kleingeschrieben. Die Manifestzahl formatVersion wird als normale JSON-Ganzzahl kodiert; typisierte Nutzwerte folgen ausschließlich der obigen Tabelle.
- Strings stehen in doppelten ASCII-Anführungszeichen. Anführungszeichen und Backslash werden mit Backslash geschützt. Backspace, Tab, LF, Form Feed und CR verwenden die kurzen JSON-Escapes b, t, n, f und r jeweils nach einem Backslash. Alle übrigen Zeichen U+0000 bis U+001F werden als Backslash-u und genau vier kleingeschriebene Hexadezimalziffern kodiert. Alle übrigen zulässigen Unicode-Skalarwerte werden direkt als UTF-8 kodiert, einschließlich /, <, >, &, U+2028 und U+2029. Es gibt keine zusätzliche HTML-Escapierung, ASCII-Umschrift oder Escape-Auswahl nach Laufzeitbibliothek.
- Die Ausgabe enthält weder BOM noch abschließenden Zeilenumbruch. Ungepaarte Surrogate und nicht endliche Zahlen sind keine gültigen Eingaben. Die Projektion verwendet kein null und keine untypisierten Nutzwerte oder binären Gleitkommazahlen.

**Ausgeschlossene Daten.** Dateinamen, root, Installations-/Downloadpfade, Quellbytes, Kommentare, Fundstellen, Prüfberichte, Lockdateien und fremde Modellinhalte sind keine Felder dieser Projektion. Ein eigener Digest enthält insbesondere keine rekursiven fremden Inhaltsdigests; die konkrete externe Bindung bleibt im Lock. Leere Packages werden nicht aufgenommen; Zwischenpackages ergeben sich aus den vollständigen Packagepfaden der enthaltenen Definitionen. Das verändert nicht die Autorenregel, dass jeder Ordner ein Package ist, und es erlaubt keine Umgehung von Schreib- oder Importfehlern bei der vorherigen Validierung. Git muss keine leeren Verzeichnisse erhalten, um dieselben kanonischen Inhaltsbytes zu erzeugen.

**Gegenproben und Grenze.** [canonical/core.json](exchange-example/canonical/core.json) ist ein festes Byteartefakt mit 3769 Bytes und Digest sha256:562a20613c64fb9cd3425ff39b5f9fb826b75deb153cdfac48b00fce6f5c19f1. Die Probe kontrolliert genaue Escape-/Sortiervektoren, exakte große Zahlen, wirksame Defaults, Listenregeln, Referenzformen, Textänderungen und vorangestellte Validierung. Ein unabhängiger Standard-JSON-Encoder wird zusätzlich gegen den projizierten Baum verglichen. Das ist ein ausführbarer Entwurfsnachweis des definierten Profils, keine allgemeine Parser-Konformität oder bereits produktive Go-Implementierung. Ressourcenlimits der Probe bleiben offengelegt und begründen keine heimliche alternative Kodierung.

## 9. Umbenennungen und Migrationen

| Änderung | Vertrag |
| --- | --- |
| title ändern | Identität bleibt; geänderte Darstellung kann im neuen Stand erfasst werden. |
| Datei innerhalb desselben Packageordners umbenennen | Identität bleibt, wenn id bestehen bleibt und die Datei weiter geladen wird. |
| Datei in einen anderen Ordner verschieben oder Packageordner umbenennen/verschieben | Package-Name und damit Identität ändern sich; Referenzen ausdrücklich migrieren. |
| Importalias einer Datei ändern | Nur damit qualifizierte Autorenreferenzen dieser Datei anpassen; andere Dateien, eindeutige direkt verfügbare Namen und kanonische Zielidentitäten bleiben. |
| Propertyfeld sender zu requester ändern | Neuer Feldschlüssel gilt im Typvertrag; konkrete specs müssen migriert werden. |
| Dabei Propertyidentität erhalten | id in der Propertydatei beibehalten und spec.name auf den gewünschten neuen Feldnamen setzen; etwa requester bei stabiler ID exchange.sender. |
| spec.name auslassen | Die vollständige lokale Property-ID ist der effektive Feldname. |
| Abweichendes spec.name entfernen | Der Feldname fällt auf die vollständige lokale ID zurück; betroffene konkrete specs müssen gegebenenfalls migriert werden. |
| ID bei fehlendem spec.name ändern | Identitätsmigration und Änderung des Standardfeldnamens; ein ausdrücklich gesetzter bisheriger Name kann den Feldschlüssel erhalten. |
| Property unter anderem Kind verwenden | Dieselbe eigenständige Propertydefinition ausdrücklich referenzieren, wenn derselbe Feldvertrag gewünscht ist. |
| Lokale id oder Paketzugehörigkeit ändern | Ausdrückliche Identitätsmigration und Anpassung bekannter Referenzen. |

Eine stabile Property-ID bewahrt Verweise auf den Feldvertrag, macht den alten Feldnamen aber nicht weiter gültig. id ist immer ausdrücklich erforderlich. Ein ausdrücklich gesetztes spec.name trennt den Feldnamen von dieser ID; ohne name gilt die ID als Standardfeldname. Ändert ein Autor zusätzlich die ID, ist das eine eigene Identitätsmigration; der Compiler errät keine Umbenennungsabsicht und erzeugt bei einer Feldumbenennung keine neue Property-ID.

Eine neue gemeinsam versionierte Modellfassung kann dieselben logischen Identitäten mit geänderten Verträgen enthalten. Die neue Bindung wird ausdrücklich gewählt und betroffene Modelle beziehungsweise Realisierungen werden gegen sie betrachtet. Weder eine stabile ID noch eine Versionsnummer beweist semantische Kompatibilität.

Veröffentlichte alte Fassungen und darauf bezogene Nachweise werden nicht rückwirkend geändert. Automatische Weiterleitungen, versteckte Namenshistorien und Zuordnung über ähnliche Namen sind nicht Bestandteil des minimalen Cores. Bei echter Identitätsänderung müssen bekannte kind-, target-, Wertreferenzen und explizite Körperlinks migriert werden. Freie Prosa benötigt zusätzlich Review.

## 10. Verarbeitung und feste Core-Regeln

Die Verarbeitung erfolgt unabhängig von der Dokumentreihenfolge:

1. Den bekannten Core-Vertrag und die ausdrücklich gebundenen Pakete bereitstellen.
2. Aus jeder Definitionsdatei genau eine vollständige Definition erfassen; die einheitliche Hülle prüfen, Herkunft erhalten und Identitätskollisionen abweisen. Eingebettete Propertydeklarationen und Definitionssammlungen sind Fehler.
3. Referenzen im eigenen Paket und den Imports der jeweiligen Definitionsdatei eindeutig auflösen.
4. Meta-Kinds und Propertyverträge gegen den festen Anfang und seine Sprachinvarianten prüfen.
5. Aus den gültigen Kind-/Propertydefinitionen geschlossene Typverträge bilden; den effektiven Feldnamen aus spec.name oder der vollständigen lokalen Property-ID bestimmen und Namenskollisionen abweisen.
6. Alle Definitionen gegen ihren Kind, Werttypen, Kardinalitäten und Referenzziele prüfen.
7. Einen festen Modellstand mit vollständigen bodies, Typverträgen, Paketbindungen und aufgelösten Bezügen bilden.

Der Compiler muss dafür die Dokumenthülle, die Ein-Datei-Regel, die beiden Meta-Kinds, deren Feldinterpretation, Werttypen, Kardinalitäten und Referenzregeln fest verstehen. Diese Bedeutung entsteht nicht dadurch, dass sie im body des festen Anfangs nochmals beschrieben ist.

Der selbsttypisierte Kind und gewöhnliche Referenzzyklen sind endliche Graphbeziehungen. Sie verlangen keine unendliche Typentfaltung. Objektvertragszyklen und Werkzeugbudgets folgen Abschnitt 5.3: Nur verpflichtende Objektzyklen sind unzulässig; tatsächlich vorhandene endliche Werte werden ohne Vorabentfaltung geprüft. Ressourcenabbrüche sind keine Sprachmaxima und dürfen keine erfolgreiche Prüfung ergeben.

Das Ergebnis macht Kind-, Property- und konkrete Definitionen gemeinsam adressierbar. Ein Inhaltsdigest bindet den Modellinhalt und die gewählten Verträge; Quellherkunft bleibt nachvollziehbar. Die genaue kanonische Datenkodierung ist in Abschnitt 8.3 gewählt. Produktive Fehlercodes und Berichtsformate bleiben getrennte Werkzeugausarbeitung.

## 12. Evidenz und noch offene Ausarbeitungen

Die [Probenübersicht in der Herleitung](README.md) und die verlinkten Berichte dokumentieren getrennte Umfänge:

- Das [vollständige vereinfachte Exchange-Beispiel](exchange-example/README.md) speichert den gewählten Core mit neun, das Vokabular mit siebzehn und das Projekt mit sieben Definitionen. Sein lokaler Prüfer und Bericht binden die gelesenen Dateien und prüfen den expliziten Beispielgraphen. Dies ist weder der neue Produktcompiler noch ein vollständiger Alias-/Markup-/Kanonisierungsnachweis.
- Der bestehende Go-Core wurde mit ausdrücklich von definition auf purpose adaptierten älteren Beispielen geprüft. Das belegt keine neue Autorenform.
- Die [Bootstrap-Probe](bootstrap/probe.json) prüft den endlichen Anfang und das gemeinsame Definitionenmodell in einem separaten Entwurfsprototyp.
- Die [Kompaktprobe](bootstrap/compact-probe.json) bestätigt für den früheren Stand die Gleichheit von ausgeschriebenen und eingebetteten Properties. Die kompakte Autorenform ist durch die spätere Nutzerentscheidung abgelöst; der historische Bericht bleibt unverändert.
- Der [Identitätsdurchgang](identities/probe.json) prüft separat importierte Versionen, Paketkontexte und die damals ergänzte stabile inline id. Seine drei gültigen Stände besitzen dieselben 51 logischen Identitäten mit unterschiedlichen Modellbindungen. Er prüft nicht die nun gewählte Ablage jeder Definition in einer eigenen Datei; der Identitätsvertrag bleibt mit ausdrücklich gesetzten Property-IDs bestehen.

Der nun gewählte Core hat neun Definitionen mit kurzen Meta-Property-IDs und einer gemeinsamen Property properties. Die historischen Bootstrap-/Identitätsproben betreffen ihren damaligen Anfang aus zehn Core-Definitionen; ihre Knoten-, Kanten- und Digestangaben werden nicht auf diesen neuen Stand übertragen.

Keiner dieser Berichte ist ein Nachweis bereits produktiver Unterstützung des gesamten neuen Vertrags. Insbesondere wurde die neue Laufzeitpflicht des Versionsbeispiels nicht durch einen ausgeführten Connector bestätigt.

Der abgeschlossene Sprachentscheidungsstand und die getrennten Implementierungs-/Verfahrensarbeiten stehen im nachfolgenden Abschlussabgleich. Bereits gewählte Regeln werden nicht wegen ihrer fehlenden Go-Implementierung wieder als offene Sprachentscheidungen geführt.

### 12.1 Stand der Definition des kanonischen Modells

**Abschlussabgleich vom 16. Oktober 2026:** Die kanonische Form des bislang gewählten minimalen Sprachumfangs ist definiert. Mit der gewählten Azyklizitätsregel für Modellabhängigkeiten ist auch die zuletzt offene Manifestfrage entschieden. Der erneute Abgleich hat keine weitere offene Grundsatzentscheidung in diesem Umfang identifiziert. Die folgende Zuordnung unterscheidet Sprachentscheidungen von ausführbarer Evidenz und Produktimplementierung.

| Bereich | Eigentümer und Stand |
| --- | --- |
| Einheitliche Definitionen, eine Datei je Kind/Property/Gegenstand, geschlossene Hülle | Abschnitt 2; gewählt. Keine kompakte oder eingebettete Definitionsform. |
| Fester Core und Feldverträge | Abschnitt 3; neun Definitionen, gemeinsame properties, kurze Property-IDs und optionaler ID-Standard für spec.name gewählt. |
| Verbindlicher body und explizite Körperlinks | Abschnitt 4, insbesondere 4.5; gewähltes CommonMark-/Linkprofil einschließlich Quellenpositionen. |
| YAML, Werttypen, Kardinalitäten und Defaults | Abschnitt 5.1 und 5.2; gewählt. |
| Objektvertragszyklen und Ressourcengrenzen | Abschnitt 5.3; optionale Abbruchkanten, Verbot verpflichtender Zyklen und reine Werkzeugbudgets gewählt. |
| Unicode, Schreibweise, Identitäten und Ordnerpackages | Abschnitt 7; gewählt, einschließlich Wurzel ohne Definitionen und fachlicher Package-Organisation. |
| Dateilokale Imports und Aliase | Abschnitt 8; gewählt, eindeutige Auflösung ohne Suchprioritäten. |
| Modelle, Versionstextlabels, gemeinsame Zeichengrammatik, physische Wurzelpfade und Modellabhängigkeiten | Abschnitt 8.1; gewählt. Modellabhängigkeiten sind azyklisch, einschließlich des Verbots von Selbstabhängigkeiten. |
| Lock und kanonische Bytes | Abschnitt 8.2 und 8.3; gewählt. Der tatsächliche Core-Digest und seine Bindung sind erprobt. |
| Änderungen und Verarbeitung | Abschnitt 9 und 10; Identitätsmigrationen sind ausdrücklich, Referenzen werden nicht unendlich entfaltet. |

Die ausführbare Probe ist ein gebundener Entwurfsnachweis, kein allgemeiner YAML-/CommonMark-Konformitätsnachweis und keine produktive Go-Implementierung. Der aktuelle Exchange-Eingabesatz bleibt wegen Command/command ungültig; sein Vokabularproblem ist keine fehlende Sprachfähigkeit. Seine ältere Aufteilung vocabulary/project ist keine empfohlene fachliche Package-Grenze.

Für die Produktion bleiben ein Go-Compiler, ein implementationsunabhängiger gültiger/ungültiger Referenztestkorpus, Diagnostik-/Berichtsformate und vollständige Quellenabbildung sowie Bereitstellung und Migration auszuarbeiten. Sie dürfen die gewählten Regeln nicht durch Parser-Voreinstellungen, Laufzeitgrenzen oder automatische Korrekturen verändern. Fehlercodes und genaue Ausgabeformate sind Werkzeugverträge, keine neuen Pflicht-Kinds.

### 12.2 Ergebnis des Vollständigkeitsabgleichs

**Keine weitere offene Sprachentscheidung im gewählten Umfang identifiziert.** Abschnitt 8.1 legt nun ausdrücklich fest, dass model.yaml-Abhängigkeiten einen azyklischen Graphen bilden müssen. A → A, A → B → A und längere Zyklen sind Fehler. Gemeinsame transitive Abhängigkeiten und gewöhnliche Referenzzyklen innerhalb eines Modells bleiben erlaubt; Objektverträge folgen ihrer eigenen Regel aus Abschnitt 5.3.

Der begrenzte Manifestprüfer weist Modellzyklen jetzt als failed ab und liefert dafür keinen Digest; die frühere vorläufige Einstufung als not_completed ist abgelöst. Gegenproben unterscheiden Selbst-, wechselseitige, längere und transitive Zyklen von gültigen Ketten und gemeinsam verwendeten Bibliotheken. Die Prüfung greift sowohl bei der physischen Manifestbindung als auch vor der kanonischen Projektion. Die Ressourcengrenzen des Werkzeugs bleiben davon getrennt.

Im übrigen bislang gewählten Sprachumfang hat dieser Abgleich keine weitere offene Grundsatzentscheidung identifiziert. Das ist kein Beweis der Eignung für jedes denkbare Projekt oder vollständiger Parserkonformität. Weitere formale Wertebedingungen, Mehrversionsimporte und zusätzliche Sprachmittel bleiben ausdrücklich optionale Erweiterungen.

**Getrennte weitere Arbeit:** Prosa und spec sind gemeinsam verbindlich. Ein Widerspruch bleibt ein semantischer Befund, auch wenn die Strukturprüfung besteht; es gibt keine automatische Vorrangregel. Kontext, semantischer Index, Aktualisierung, Realisierungszuordnung, Prüfplanung und Apply aus Abschnitt 11 sind ausgearbeitete Verfahrensvorschläge und nicht als Ganzes gewählte Teile der kanonischen Form. Ihre Ausführungsdaten verlangen keine zusätzlichen Core-Kinds. Produktimplementierung, Prüfevidenz und fachliche Akzeptanz bleiben getrennt.

Eine redaktionell gestraffte normative v1-Fassung mit Referenzfällen ist als Aufbereitung sinnvoll; sie soll die gewählten Sprachregeln und bewusst ausgeschlossenen Erweiterungen klar vom historischen Gespräch und den Vorschlägen aus Abschnitt 11 trennen. Das ist keine weitere offene Sprachentscheidung und keine Voraussetzung, den hier festgelegten Umfang zu implementieren.

Zurück zum [gemeinsamen Designstand](../consolidation.md) oder zur [Begriffsübersicht](../README.md).

## Frühere Navigationszusammenfassung der Core-Herleitung

Die nachfolgenden Statusangaben stammen aus der Navigation vor der v1-Konsolidierung.

Stand: 15. Oktober 2026. **Status: Gesprächsherleitung, frühere Varianten und begrenzte Prüfberichte.** Der aktuelle, zusammenhängende Stand steht im [konsolidierten Sprachvertrag](language-contract.md). Der Nutzer hat inzwischen auch den Identitätsvertrag gewählt und die Konsolidierung beauftragt. Frühere Vorschlags- und Statusangaben auf dieser Seite dokumentieren ihren jeweiligen Gesprächszeitpunkt; sie setzen die späteren Entscheidungen nicht außer Kraft. Produktunterstützung des neuen Gesamtvertrags wird nicht behauptet.

**Aktuell gewählte Richtung:** Der Nutzer entscheidet sich für **eine einheitliche Definitionenform mit den fest verstandenen Kinds Kind und Property**. In der Gesprächsfortsetzung legt er **eine Datei pro Definition** fest, einschließlich jeder Property und jedes konkreten Gegenstands, und schließt kompakte oder eingebettete Property-Sonderformen aus. Maßgeblich ist [Abschnitt 6 des Sprachvertrags](language-contract.md#property-names). body bleibt der verbindliche Körper; requirements und roles entfallen vorerst als besondere Sprachmittel. [Abschnitt 14](README.md#14-gewaehlte-grundform-und-kompakte-properties) dokumentiert die inzwischen abgelöste kompakte Autorenform und ihre damaligen Proben; eine neue Produktimplementierung ist damit nicht behauptet.

**Weitere gewählte Richtung:** Package ist der einzige Namensraum; jeder Ordner unter der Modellwurzel bildet ein Package, dessen Name aus dem relativen Pfad folgt. Ein zusätzliches namespace-Feld entfällt. Seit der Nutzerentscheidung vom 16. Oktober ist Package keine Versionierungs-/Verteilungseinheit mehr; diese Einheit heißt Modell. Direkte Imports in der jeweiligen Definitionsdatei machen vollständige Definitions-IDs ohne Präfix verfügbar. Die Bereitstellung und Versionsbindung zusammengehöriger Modelle wird getrennt festgelegt und öffnet keine Namen. Package-Namen und vollständige Package-Qualifizierung verwenden /; gegliederte lokale IDs und nötige Aliasqualifizierung weiterhin Punkte. Mehrdeutigkeiten werden abgewiesen. [Abschnitt 7 und 8 des Sprachvertrags](language-contract.md#names) besitzen diese spätere Entscheidung; die folgenden historischen Beispiele und Proben behalten ihre frühere Notation.

**Weitere Auswahl zu Properties:** Keine verpflichtenden ID-Präfixe; spec.name ist optional und verwendet bei Abwesenheit die vollständige lokale ID als Feldnamen. [Abschnitt 6.1 des Sprachvertrags](language-contract.md#property-names) besitzt diese Regel. Der Nutzer hat anschließend die gemeinsame Meta-Property properties übernommen. Der [vollständige Core mit neun Definitionsdateien](language-contract.md#core-definitions) verwendet kurze Meta-Property-IDs ohne ausdrückliches spec.name; [Abschnitt 3.2](language-contract.md#property-contracts) hält die Verwendung und Gegenproben fest. Historische Proben des früheren zehnteiligen Anfangs bleiben unverändert.

**Gewähltes YAML-/Werteprofil:** [Abschnitt 5.1](language-contract.md#yaml-values) besitzt die begrenzte YAML-1.2.2-Autorenform, eindeutige Skalarregeln ohne implizite Datums-/yes/on-Konvertierung, exakte endliche Zahlen und strenge Strukturprüfung. Python/PyYAML bleiben Hilfsmittel der Entwurfsprobe; das Go-Produkt wird dadurch nicht umgestellt. [Abschnitt 5.2](language-contract.md#lists) hält inzwischen geordnete Wertlisten, ungeordnete Core-Deklarationen/Imports und die kanonische Gleichheit fehlender/leerer optionaler Listen fest. Quellen bleiben unverändert; Abschnitt 8.3 bestimmt inzwischen die vollständige kanonische Bytekodierung.

**Gewählte Körperlink-Regeln:** [Abschnitt 4.5](language-contract.md#body-links) bindet CommonMark 0.31.2, geschützte Kontexte, exakte Ziele, einfache Anzeigetexte, Escapes und UTF-8-/YAML-Fundstellen. Der begrenzte Prüfer besitzt dazu einen Parser und Gegenproben; zwei aktive Links des Beispiels werden separat vom weiterhin ungültigen gesamten Definitionsgraphen geprüft. Ordnerpackages stehen inzwischen in Abschnitt 7.3. Abschnitt 8.1 wählt inzwischen model.yaml für stabile Modellidentität, relative Wurzel, optionale lokale Version und genaue direkte Modellabhängigkeiten. Belegte Packages gehören genau einem Modell; leere Zwischenordner können gemeinsam auftreten. Das Beispiel lädt zwei Manifeste. Abschnitt 8.2 wählt inzwischen model-lock.yaml für die vollständige externe Modellmenge mit Version und eigenem Inhaltsdigest, ohne zweite Graphbeschreibung oder Speicherorte. Abschnitt 8.3 bestimmt inzwischen model-digest/1 als genaue UTF-8-JSON-Projektion mit Defaults, aufgelösten Referenzen und unveränderten Texten. Der Platzhalter ist durch den echten Core-Digest ersetzt und gegen 3769 gespeicherte kanonische Bytes geprüft. Für das ungültige Connector-Modell wird kein Digest ausgegeben. Die Modellwurzel ist selbst kein Package und darf keine Definitionen unmittelbar enthalten; jede Definition gehört zu einem benannten Unterordner-Package. Modell-IDs und Versionslabels verwenden inzwischen dieselbe Grammatik: kleingeschriebene NFC-Unicode-Abschnitte, mit Ziffernstart und . oder - als Trennern. Physische Wurzelpfade sind ebenfalls gewählt: . oder exakt geschriebene relative /-Pfade mit vorhandenen Ordnern nach Packageordnergrammatik, ohne Normalisierung oder Umleitungen und ohne überlappende Modellwurzeln. Objektzyklus- und Ressourcenregeln stehen inzwischen in Abschnitt 5.3: nur verpflichtende Objektzyklen sind Fehler; Budgets begrenzen das Werkzeug und liefern bei Abbruch keinen Digest. Modellabhängigkeiten sind inzwischen als azyklisch festgelegt, einschließlich des Verbots von Selbstabhängigkeiten. Gemeinsame transitive Bibliotheken bleiben erlaubt. Der Abschlussabgleich identifiziert keine weitere offene Sprachentscheidung im gewählten minimalen Umfang; Produktimplementierung und Belege bleiben getrennt.

Das [vollständige vereinfachte Exchange-Beispiel](exchange-example/README.md) enthält die aktuellen Dateien für Core, Vokabular und Projekt sowie eine begrenzte reproduzierbare Strukturprobe. Inzwischen sind strenge Schreibkonventionen ohne automatische Korrektur, Identitäts-/Kollisionsprüfung unabhängig von Groß-/Kleinschreibung und dateilokale Importaliase gewählt. Jeder Import ist ein Objekt mit package und optionalem as. Anschließend sind Unicode als durchgängige Textgrundlage, NFC für Namen und Referenzen sowie vollständiger kulturunabhängiger Namensvergleich einschließlich straße/strasse gewählt; Abschnitt 7.2 besitzt das auf Unicode 15.1.0 gebundene Namensprofil. Der ungelöste Konflikt Command/command wird unter diesen Regeln abgewiesen; seine Modellierung bleibt eine Vokabularfrage. Die früheren Berichte und Beispiele weiter unten bleiben historische Herleitung. [Abschnitt 12.2 des Sprachvertrags](language-contract.md#conformance) trennt diesen Beispielstand von den verbleibenden Sprachentscheidungen.

Der [Identitaets- und Referenzvertrag](README.md#15-identitaeten-imports-versionen-und-umbenennungen) unterscheidet stabile Identitaet, optionale Anzeigenamen und die Bindung importierter Versionen. Nach dem Durchgang in Abschnitt 16 hat der Nutzer diesen Vertrag gewaehlt; die verbindlichen Designregeln und die noch offenen Formatdetails sind im konsolidierten Sprachvertrag getrennt festgehalten.

Der [vollstaendige Identitaetsdurchgang](README.md#16-identitaetsdurchgang-mit-import-update-und-feldumbenennung) prueft diesen Vorschlag inzwischen begrenzt mit importiertem Vokabular, drei gebundenen Staenden und sender/requester. Er implementiert keinen Produktloader.

Die [Untersuchung einer gemeinsamen Grundform](README.md#11-gemeinsame-grundform-und-body) ist die Grundlage der inzwischen gewaehlten Richtung. Die bisherigen YAML-Dateien und ihre hashgebundenen Berichte bleiben unveraenderte Nachweise des vorherigen Durchstichs mit definition. Der aktuelle Implementierungsname purpose bleibt davon unberuehrt.

Der anschliessend beauftragte [feste Anfang und die vollstaendige Typaufloesung](README.md#12-fester-anfang-und-typaufloesung-am-exchange) sind als eigene, begrenzte Entwurfsprobe mit body ausgearbeitet. Die Probe verwendet eine gemeinsame Definitionenform und ersetzt den Produktcompiler nicht.

Der [Vergleich beider Core-Varianten](README.md#13-vergleich-und-empfehlung-fuer-markitect) dokumentiert auch die damalige Empfehlung des Assistenten fuer die explizite Variante. Diese Empfehlung ist durch die anschliessende Nutzerwahl zugunsten der einheitlichen Form abgeloest. Fruehere Abschnitte bleiben als Herleitung und Vergleich erhalten.

# Markitect: normative Sprachspezifikation v1

**Status:** Abgeschlossene v1-Designbaseline, vom Nutzer am 16. Oktober 2026 bestätigt. Diese Datei ist die alleinige normative Quelle für die kanonische Form. Der gewählte v1-Sprachumfang ist fertig designt; es bestehen keine offenen Sprachentscheidungen innerhalb dieses Umfangs.

Der Abschluss betrifft das Sprachdesign. Produktive Go-Implementierung und ein breiter Konformitätskorpus stehen noch aus. Die ausführbaren Entwurfsproben belegen ihren jeweiligen Prüfungsumfang, keine vollständige Unterstützung durch den Produktcompiler. Projektvokabulare und die gesonderten Verfahrensvorschläge gehören nicht zum Abschluss dieser Sprachbaseline.

**Verbindlichkeit:** MUSS und DARF NICHT bezeichnen verpflichtende Regeln beziehungsweise Verbote. DARF bezeichnet eine erlaubte Möglichkeit. Auch die deklarativen Regeln, Tabellen, Grammatiken und die neun Core-Definitionen dieser Spezifikation sind normativ. Es gibt keine implizite Fehlerkorrektur. Beispiele und Prüfberichte außerhalb dieser Datei sind informativ und können diese Regeln nicht ergänzen oder überschreiben.

[Sprachgeschichte](language-history.md), [noch nicht gewählte Verfahrensvorschläge](procedure-proposals.md) und [ausführbares Exchange-Beispiel mit begrenzter Prüfevidenz](exchange-example/README.md) sind getrennte Dokumente. Sie sind keine alternativen Sprachverträge.

| Gegenstand | Normativer Abschnitt |
| --- | --- |
| Einheitliche Datei und fester Core | [Definition](#definition), [Core](#core-definitions), [Feldverträge](#property-contracts) |
| Verbindliche Prosa und Links | [body](#body), [Markdown-/Linkprofil](#body-links) |
| Werte, Listen und endliche Verarbeitung | [Werte](#values), [YAML](#yaml-values), [Listen](#lists), [Zyklen und Ressourcen](#cycles-resources) |
| Namen und Packages | [Auflösung](#names), [Schreibweise](#spelling), [Unicode](#unicode), [Ordner](#packages) |
| Gebundene Modelle | [Imports](#imports), [Manifest und Wurzel](#models), [Lock](#lock) |
| Kanonischer Inhalt und Gültigkeit | [Bytes](#canonical-bytes), [Änderungen](#changes), [Verarbeitung](#processing), [Konformität](#conformance) |


<a id="scope"></a>

## 1. Zweck und Umfang

Das kanonische Modell beschreibt die beabsichtigte Projektwelt: was gelten und entstehen soll. Verbindliche Prosa und typisierte Angaben tragen diese Beschreibung gemeinsam. Die tatsächlich entstandenen Dateien, laufenden Vorgänge und ausgeführten Nachweise sind davon zu unterscheiden.

Der Core stellt die allgemeinen Sprachmittel vollständig bereit. Projekte können ihr Vokabular selbst definieren. Bibliotheken liefern wiederverwendbare Definitionen; sie verleihen importierten Kinds keine zusätzlichen Sprachfähigkeiten oder Ausführungsrechte.

System, Container, Component, Command, Exchange, Rule oder andere Fachbegriffe sind mögliche Inhalte eines Vokabulars. Sie sind keine privilegierten Core-Kinds. Das Modell benötigt weder ein eingebautes Project noch eine vorgegebene Softwarearchitektur-Taxonomie.

<a id="definition"></a>

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

`id`, `kind` und `body` MÜSSEN nichtleere Strings sein. `body` MUSS außerdem mindestens ein Zeichen außerhalb der Unicode-Eigenschaft White_Space in Version 15.1.0 enthalten (Abschnitt 4). `spec` MUSS ein Mapping sein; es darf leer sein, wenn der Kind-Vertrag dies erlaubt. Ein vorhandenes `title` MUSS ein String sein. Unbekannte Hüllenfelder sind Fehler; zusätzliche Projektangaben werden als Properties am jeweiligen Kind deklariert.

Jede Definition steht vollständig in genau einer eigenen YAML-Datei. Das gilt gleichermaßen für Kinds, Properties und konkrete Gegenstände sowie für die Definitionen des festen Core-Pakets. Jede Definitionsdatei enthält dieselbe Hülle mit id, kind, body, spec und optional imports sowie title. Mehrere Definitionen in einer Datei, eingebettete Definitionen und kompakte Propertydeklarationen sind nicht zulässig. Die Felder innerhalb von spec richten sich nach dem jeweiligen Kind; strukturierte object-Werte bleiben gewöhnliche Werte und sind keine eingebetteten Definitionen.

Der Dateiname ist keine Identität. Der enthaltende Ordner bestimmt dagegen das Package und damit einen Teil der Identität. Eine ausdrücklich bestimmte Modellwurzel begrenzt die Ordnerstruktur. Die Bindung externer Modelle und ihrer genauen Inhalte wird getrennt von den Packages festgelegt; Modell ist die gemeinsame Versionierungs- und Verteilungseinheit. Jede Definitionsdatei wählt daraus mit imports selbst ihre sichtbaren fremden Pakete; eigene Paketdefinitionen sind ohne Import sichtbar. Versionen und Digests werden nicht in jeder Definitionsdatei wiederholt. Abschnitt 8 trennt diese beiden Aufgaben.

<a id="core"></a>

## 3. Der feste Anfang: Kind und Property

Der feste Core im Package `markitect/core` besteht aus genau neun Definitionen: den Kinds `Kind` und `Property` sowie den sieben Properties `name`, `type`, `minCount`, `maxCount`, `values`, `target` und `properties`. Abschnitt 3.1 definiert ihre vollständigen Inhalte.

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

<a id="core-definitions"></a>

### 3.1 Vollständiger Core in neun Definitionsdateien

Alle neun Dateien gehören zum Paket markitect/core. Sie benötigen keine Imports, weil ihre Referenzziele im eigenen Paket liegen. Jede Datei hat dieselbe vollständige Hülle. Der effektive Feldname jeder Meta-Property entspricht ihrer kurzen lokalen ID; spec.name entfällt deshalb durchgängig.

Auch beim Vergleich mit dieser festen Selbstbeschreibung sind fehlende imports und ausdrücklich gesetzte `imports: []` gemäß Abschnitt 5.2 gleichwertig. Die Autorenform bleibt erhalten; eine fehlerhafte Angabe oder eine nichtleere Importliste wird nicht als leere Liste behandelt.

Der Vergleich prüft den wirksamen Inhalt nach den allgemeinen Sprachregeln: Referenzen in kind, target und properties werden anhand ihrer aufgelösten Package-/ID-Ziele verglichen; wirksame Standardwerte und ungeordnete Deklarationen folgen Abschnitt 5.2 und 8.3. Kurze, vollständig qualifizierte und mit gültigem Importalias qualifizierte Referenzen sind auch im Core gleichwertig. Imports werden vollständig geprüft, sind nach erfolgreicher Auflösung aber kein zusätzliches Inhaltsfeld. Ein weiterhin gültiger unbenutzter Import verändert die Selbstbeschreibung nicht. Schreibfehler, Mehrdeutigkeiten, ungültige Imports, Duplikate und verbotene Felder bleiben Fehler. Die vollständigen bodies und alle sonstigen Inhaltswerte müssen weiterhin dem festen Core entsprechen; die Quelldateien werden nicht umgeschrieben.

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

<a id="property-contracts"></a>

### 3.2 Gemeinsame Property am Kind und am Objekt

Die feste Core-Property `properties` deklariert eine Liste eigenständiger Propertydefinitionen. Am Kind bestimmt sie die erlaubten Felder der `spec` seiner Definitionen; an einer object-Property bestimmt sie die erlaubten Felder ihres Objektwerts. Diese Bedeutung folgt ausschließlich aus den gebundenen Meta-Identitäten.

| Betrachtete Definition | Bedeutung der gemeinsamen Property |
| --- | --- |
| kind ist der feste Meta-Kind Kind | properties deklariert die erlaubten Felder der spec der Definitionen dieses Kinds. Das gilt auch für kind.yaml und property.yaml selbst. |
| kind ist der feste Meta-Kind Property und spec.type ist object | properties deklariert die erlaubten Felder des jeweiligen Objektwerts. |
| kind ist der feste Meta-Kind Property mit anderem spec.type | properties ist als Vertragsangabe nicht zulässig, auch nicht als leere Liste. Die bereits bestehende Werttypbedingung bleibt erhalten. |
| kind ist ein gewöhnlicher Projekt-Kind | Dessen eigener Feldvertrag gilt. Ein dort erlaubtes Feld namens properties, auch bei Wiederverwendung der gemeinsamen Property, ist gewöhnlicher Dateninhalt und erzeugt keinen zusätzlichen Typ- oder Objektvertragsoperator. |

Die letzte Zeile verhindert eine unzulässige globale Reservierung des Feldnamens. Ein Projekt-Kind SchemaCatalog könnte properties als Liste von Propertyreferenzen deklarieren und konkrete Katalogeinträge damit beschreiben. Ein solcher Katalogeintrag wird dadurch nicht zum Kind; nur eine Definition des festen Meta-Kinds Kind deklariert einen neuen Typvertrag. Die feste Sprachbedeutung hängt an genau gebundenen Meta-Identitäten, nicht an geschriebenen Namen oder einem beliebigen Importalias.

<a id="body"></a>

## 4. Verbindliche Prosa und Geltungsbereich

body ist Teil des geltenden Modellvertrags und wird im gebundenen Inhalt erhalten.

**Nichtleerer Inhalt:** Der nach YAML dekodierte body MUSS mindestens ein Zeichen enthalten, das nicht zur Unicode-Eigenschaft White_Space in der gebundenen Version 15.1.0 gehört. Ein leerer String und ein ausschließlich aus solchen Leerraumzeichen bestehender String sind Fehler. Es gilt dieselbe Zeichenmenge wie für Link-Anzeigetexte in Abschnitt 4.5, keine sprach- oder laufzeitspezifische strip-/isspace-Definition. Die Prüfung trimmt, ersetzt oder normalisiert den Text nicht. Führender, innerer und abschließender Leerraum bleiben vollständig erhalten und gehen unverändert in den kanonischen body-Wert ein. Diese Inhaltsbedingung erweitert keine Markdown- oder Körperlinkgrammatik und gilt nicht allgemein für gewöhnliche Stringwerte oder das optionale title.

- Der body eines Kinds beschreibt die gemeinsame Bedeutung und geltenden Festlegungen seiner Definitionen.
- Der body einer Property beschreibt die Bedeutung ihrer Werte und Beziehungen im verwendenden Vertrag.
- Der individuelle body ergänzt die Festlegungen des konkreten Gegenstands.
- Individuelle Angaben heben allgemeine Pflichten nicht still auf. Widersprüche werden als zu klärende Befunde behandelt.

Der Text kann Modellbedingungen, gewünschtes Laufzeitverhalten, Grenzen und Bedingungen seiner Anwendbarkeit beschreiben. Eine nicht formal prüfbare Aussage bleibt verbindlich. Der Core besitzt keine besonderen requirements-, roles- oder subject-Abschnitte und keine allgemeine formale Regelsprache. Ein Projekt kann solche Wörter als eigene Begriffe verwenden, ohne dadurch besondere Compilersemantik zu erhalten.

Beziehungen erhalten ihre fachliche Bedeutung aus dem Typ- und Propertyvertrag. Eine Aussage über requester betrifft die dort referenzierte Anwendung im beschriebenen Austausch. Aus einem Relationsnamen folgen weder globale Regelgeltung noch transitive Zusammensetzung, Agentenaufteilung oder Schreibrechte.

### 4.1 Ein Linkformat für alle Definitionen

Die Autorenform lautet `[[Ziel]]` oder `[[Ziel|Anzeigetext]]`. Das Ziel ist eine im eigenen Package oder durch direkte Imports verfügbare id, eine vollständige Referenz package/id oder eine ausdrücklich aliasqualifizierte Referenz Alias.id. Die Auflösung erfolgt im eigenen Paket und den dateilokalen Imports der Definition, deren body den Link enthält. Sie benutzt dieselben festen Paketbindungen wie kind und die Referenzwerte in spec. Der Anzeigetext beeinflusst die Auflösung nicht. Ohne ausdrücklichen Anzeigetext wird die geschriebene Zieladresse angezeigt.

Jede Property ist über ihre ausdrücklich deklarierte lokale id adressierbar. Es gibt weder eine eigene Propertylink-Syntax noch automatische Suche nach einem Feldnamen. Eine Feldbezeichnung wird nicht automatisch zu einer zusätzlichen Property-ID.

### 4.2 Deklaration und konkreter Wert

Ein Körperlink auf eine Property bezeichnet deren eigenständige Definition, keinen konkreten Feldwert und keinen Zugriffsausdruck. Ein konkreter Referenzwert wird in `spec` gesetzt. Fachliche Zusammenhänge trägt der umgebende Satz; stimmen Prosa und `spec` nicht überein, bleibt dies ein semantischer Widerspruch. Es gibt keine automatische Vorrangregel.


### 4.3 Die feste Bedeutung des Links

Der Core kennt genau eine zusätzliche Beziehung aus diesem Markup: **Die Quelldefinition nimmt an dieser Körperstelle ausdrücklich auf die Zieldefinition Bezug.** Er erhält die geordneten Vorkommen mit Quelldefinition, kanonischem Paket-/ID-Ziel und optionalem Anzeigetext. Wiederholte Links bleiben einzelne Vorkommen; eine zusammengefasste Graphsicht darf sie als einen Bezug mit mehreren Vorkommen darstellen. Dafür gibt es keine zweite manuelle uses-/references-Liste und keinen neuen eingebauten fachlichen Kind. Exakte Diagnosekoordinaten sind keine Voraussetzung dieser Beziehung.

Die fachliche Aussage entsteht aus Satz, Kind- und Propertyvertrag. Daraus folgen diese Grenzen:

- Ein Kind-Link typisiert die Quelle nicht. Dafür gilt das Feld kind.
- Ein Property-Link erklärt oder verwendet den Feldvertrag; er trägt keinen Wert in spec ein und zählt nicht als Erfüllung einer Property-Kardinalität.
- Ein Definitionslink erklärt allein weder Enthaltenheit, Laufzeitaufruf, positive Abhängigkeit noch globale Regelgeltung. Auch „verwendet [[dms]] nicht“ enthält einen ausdrücklichen Bezug auf dms.
- Das Referenzieren einer Regel aktiviert sie nicht automatisch für den ganzen verlinkten Gegenstand. Der geltende Umfang muss aus ihrem Vertrag und der Aussage hervorgehen.
- Verbindliche unmarkierte Prosa bleibt Teil des Modells. Links helfen beim eindeutigen Bezeichnen, ersetzen aber weder die Aussage noch eine Vollständigkeitsbeurteilung.

Die gebundene Bedeutung des Linkziels gehört zum Verständnis des Satzes. Bloße Erreichbarkeit im Graph importiert nicht transitiv sämtliche Pflichten aller erreichbaren Definitionen.

### 4.4 Prüfung, Umbenennung und Modellstand

Der Compiler kann aktive Links erkennen, ihre Ziele nach vollständigem Laden der Definitionen eindeutig auflösen und Herkunft erhalten. Fehlendes Ziel oder unbekannter Import ist ein Fehler; Titel, Anzeigetext und erwartete Bedeutung dienen nicht als Ausweichsuche. Anders als bei einer typisierten reference-Property ist im Körperlink zunächst jede Definition als Ziel zulässig. Ob der Satz den richtigen Kind, die richtige Property oder die richtige konkrete Definition meint, bleibt Gegenstand des semantischen Reviews.

Der feste Modellstand enthält den vollständigen Körper und die aufgelösten Zielidentitäten. Änderungen an Prosa oder Anzeigetext werden erhalten, auch wenn die Zielmenge gleich bleibt. Quellpositionen gehören zur Diagnoseherkunft, nicht zur logischen Zielidentität. Aliaswechsel ändern das kanonische Linkziel nicht; eine Änderung des gespeicherten Körpers kann trotzdem im Inhaltsvergleich sichtbar sein. Die Digestkodierung steht in Abschnitt 8.3.

Der Compiler prüft die Existenz der Bezüge, nicht die Wahrheit, Vollständigkeit oder korrekte Auslegung der Aussagen. Agenten können fehlende oder falsche Links vorschlagen; erst eine in der Quelle übernommene Änderung verändert den kanonischen Bezug. Linkvorschläge aus Wortähnlichkeit sind keine Modellkanten.

<a id="body-links"></a>

### 4.5 Verbindliches Körperlink- und Markdownprofil

body verwendet [CommonMark 0.31.2](https://spec.commonmark.org/0.31.2/) mit der ausdrücklichen Erweiterung um Körperlinks. Diese Regeln gelten ausschließlich für body. title und andere Stringwerte werden nicht auf Modelllinks untersucht.

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

Das Erhaltungsgebot umfasst ausdrücklich U+0000 (NUL), wenn es nach der YAML-Dekodierung im Anzeigetext vorliegt. Dort DARF es NICHT durch U+FFFD oder ein anderes Zeichen ersetzt werden. Die CommonMark-Normalisierung zur Erkennung des umgebenden Markdownkontexts ändert diesen literalen Wert der Körperlink-Erweiterung nicht. Ein tatsächlich geschriebener U+FFFD bleibt von NUL verschieden. Beispielsweise dekodiert `body: "[[api|A\0B]]"` zu einem Link mit den Anzeigetextzeichen U+0041, U+0000 und U+0042. Dies führt keinen zusätzlichen Körperlink-Escape ein; `\0` innerhalb eines bereits YAML-dekodierten Links bleibt gemäß der folgenden Escaperegel unzulässig.

**Escapes:** Außerhalb des Links gelten die CommonMark-Regeln. `\[[api]]` macht die Notation wörtlich; ein durch einen weiteren Backslash geschützter Backslash verhindert einen danach normal beginnenden Link nicht. Im Anzeigetext sind genau `\|`, `\[`, `\]` und `\\` für die entsprechenden literalen Zeichen erlaubt. Andere Backslashsequenzen im Linkinneren sind Fehler. Im Ziel sind keine Escapes nötig oder zulässig; seine Zeichen folgen der Namensgrammatik. Ein zusätzliches ungeschütztes |, eine ungeschützte einzelne Klammer oder ein aktives [[ ohne passenden Abschluss ist ein Markupfehler. Ein isoliertes ]] außerhalb eines aktiven Links bleibt gewöhnlicher Text.

Ein Link muss im dekodierten body einzeilig sein. YAML kann den Autorenquelltext über mehrere Dateizeilen falten, solange dabei im dekodierten Link kein verbotenes Zeilenumbruchzeichen entsteht. Ungültige aktive Notation ist ein Strukturfehler und wird nicht still zu Text zurückgestuft. Gleichartige Zeichenfolgen in den geschützten Kontexten bleiben dagegen wörtlich.

**Vorkommen und Diagnose:** Für die Sprache sind die vollständige Linkerkennung, eindeutige Zielauflösung sowie Reihenfolge und Wiederholung der Vorkommen verbindlich. Die Sprache schreibt weder UTF-8-Bytebereiche noch Zeilen-/Spaltenkoordinaten vor. Der separate [Diagnosevertrag für Werkzeuge](diagnostic-contract.md) bestimmt die exakte Koordinatenform für Werkzeuge, die dieses Profil anbieten. Diese Koordinaten sind Herkunft, keine Zielidentität und kein kanonischer Modellinhalt. Fehlende oder nicht belegbare Diagnosekoordinaten allein DÜRFEN eine erfolgreich abgeschlossene Inhaltsprüfung und ihren Digest NICHT verhindern. Eine unvollständige Linkerkennung, Zielauflösung oder sonstige Inhaltsprüfung verhindert weiterhin die kanonische Ausgabe für den betroffenen Modellumfang.

<a id="values"></a>

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

- spec.name ist optional. Ohne Angabe ist der effektive Feldname exakt die vollständige lokale id der Propertydefinition. Paketname, Importalias und Referenzschreibweise gehören nicht zu diesem Feldnamen; Präfixe der lokalen ID werden nicht abgeschnitten. Punkte sind in lokalen IDs und Feldnamen ausgeschlossen.
- Ein ausdrücklich gesetztes spec.name muss ein nichtleerer String sein; null oder eine leere Angabe lösen keinen Fallback aus. Die effektiven Feldnamen müssen innerhalb jedes Kind- oder object-Vertrags eindeutig sein.
- minCount ist mindestens 0. maxCount ist -1 oder mindestens minCount.
- maxCount=1 bezeichnet einen Einzelwert; andere Höchstwerte bezeichnen eine Liste. -1 setzt keine sprachliche Höchstanzahl; Ressourcenregeln stehen in Abschnitt 5.3.
- Eine ausgelassene Angabe zählt als null vorhandene Werte. Der Datenwert null ist nicht zulässig.
- reference benötigt target. Das Ziel von target ist eine gültige Typdefinition.
- enum benötigt eine nichtleere Liste eindeutiger erlaubter Werte.
- object verwendet die unter properties referenzierten eigenständigen Propertydefinitionen. Nicht deklarierte Objektfelder sind Fehler. Ein konkreter Objektwert enthält Werte dieser Felder, keine neuen Propertydefinitionen.
- values, target und properties sind nur bei den dafür vorgesehenen Werttypen zulässig. Ein neuer Typname installiert keinen Parser oder Operator.

Ein Typbezug wird durch reference mit Ziel-Kind Kind beschrieben; kindReference gehört nicht zu den Werttypen. Die Typbezüge im gemeinsamen Feld kind bleiben ausdrücklich als solche prüfbar.

Die Kardinalität begrenzt die Zahl der Werte, nicht deren numerischen Wertebereich. Eine fachliche Aussage wie „maxAttempts muss positiv sein“ benötigt weiterhin semantische Beurteilung, solange keine zusätzliche formale Wertebedingung eingeführt ist.

<a id="yaml-values"></a>

### 5.1 YAML- und Werteprofil

Die Autorenform verwendet das nachfolgende eingeschränkte Profil von [YAML 1.2.2](https://yaml.org/spec/1.2.2/). Dieses Profil bestimmt die Leseform unabhängig von Bibliothek und Implementierungssprache.

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

Es gibt keinen sprachlichen 64-Bit-Bereich oder festgelegten Rundungskontext. Endliche Werkzeuglimits folgen Abschnitt 5.3; sie dürfen keine gerundete oder abgeschnittene Zahl als erfolgreiche Auswertung liefern.


<a id="lists"></a>

### 5.2 Listenreihenfolge und fehlende gegenüber leeren Listen

Gewöhnliche Wertlisten sind geordnet. Deklarative Aufzählungen des festen Cores und die Importliste sind ungeordnet. Eine fehlende optionale Liste und eine ausdrücklich leere Liste haben denselben kanonischen Wert. Die originale Autorenform bleibt als Quelleninformation erhalten.

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

<a id="cycles-resources"></a>

### 5.3 Objektvertragszyklen und Ressourcengrenzen

Gewöhnliche Definitionsreferenzen dürfen Zyklen bilden, einschließlich des festen Core-Anfangs. Referenzziele werden als Identitäten aufgelöst und nicht durch textuelle oder wertseitige Entfaltung ersetzt. Für Objektverträge gilt die folgende eigene Bedingung.

Ein Objektvertragsgraph hat die object-Propertydefinitionen als Knoten. Eine Kante führt vom Objektvertrag zu jeder darin als Feld deklarierten object-Property. Eine Kante ist verpflichtend, wenn die Ziel-Property minCount>0 hat; das gilt für Einzelwerte und nichtleere Listen gleichermaßen. Der Teilgraph ausschließlich verpflichtender Kanten muss azyklisch sein. Damit enthält jeder erlaubte Objektzyklus mindestens eine optionale Kante mit minCount=0. Auch nicht verwendete Objektverträge werden auf verpflichtende Zyklen geprüft; ein unmöglich zu erfüllender Vertrag wird nicht durch fehlende konkrete Instanzen gültig.

| Fall | Sprachregel |
| --- | --- |
| Optionale object-Property verwendet sich selbst | Erlaubt; der Wert kann ohne dieses Feld beziehungsweise mit leerer optionaler Liste enden. |
| Wechselseitige Objektverträge mit mindestens einer optionalen Kante je Zyklus | Erlaubt; endliche konkrete Werte sind möglich. |
| Objektzyklus ausschließlich über verpflichtende Felder | Fehler; ein endlicher Wert kann diese Pflicht nicht erfüllen. |
| Gewöhnliche reference-Property führt auf einen bereits referenzierten Gegenstand | Erlaubter Definitionsbezug; keine Objektentfaltung. |
| Leere Liste bei minCount=0 und maxCount=0 | Zulässig; keine Objektwerte werden erzeugt oder besucht. |

Objektwerte bleiben endliche, geschlossene YAML-Werte gemäß Abschnitt 5.1. Die Werteprüfung und kanonische Projektion folgen ausschließlich tatsächlich vorhandenen Werten; sie entfalten den Vertragsgraphen nicht vorab und erzeugen keine fehlenden optionalen Objekte. Die bisherigen Regeln für fehlende optionale Listen gelten innerhalb vorhandener Objekte unverändert. Das Verbot von YAML-Anchors und -Aliasen bleibt bestehen.

**Ressourcenpolitik:** Die Sprache setzt keine zusätzlichen festen Größen-, Mengen- oder Tiefenmaxima. Werkzeuge dürfen endliche Verarbeitungsbudgets verwenden, müssen deren Umfang und Werte aber ausdrücklich ausweisen und im Prüfbericht nennen. Eine reine Limitüberschreitung ist kein Nachweis einer Modellverletzung: Die betroffene Inhaltsprüfung meldet not_completed und erzeugt für diesen Modellumfang weder kanonische Bytes noch Digest oder Vollständigkeitsnachweis. Ein bereits nachgewiesener Sprachfehler bleibt ein Fehler, auch wenn eine weitere Prüfung abbricht; der Abbruch wird zusätzlich ausgewiesen. Ein unabhängiger vollständig geprüfter Modellumfang kann weiterhin ein Ergebnis erhalten.

Vollständig abgeschlossene Läufe über dieselben gebundenen Eingaben müssen bei ausreichend großen Budgets dieselben kanonischen Bytes und Digests liefern. Budgets und Laufzeitkapazitäten gehören zu Werkzeugkonfiguration und Prüfprovenienz, nicht zur kanonischen Modellprojektion. Es gibt kein stilles Abschneiden, Überspringen oder Runden als erfolgreiche Auswertung. Laufzeitbedingte Rekursions- oder Speicherabbrüche bei der Inhaltsprüfung werden ebenfalls als unvollständige Auswertung ausgewiesen. Reine Koordinatenabbrüche betreffen ausschließlich das separate Diagnoseergebnis.

Die Regeln dieses Abschnitts betreffen Definitionsreferenzen und Objektverträge. Für den gesonderten model.yaml-Abhängigkeitsgraphen gilt die Azyklizitätsregel aus Abschnitt 8.1; eine zyklische Definitionsreferenz ist keine Selbstabhängigkeit des Modells.

<a id="property-names"></a>

## 6. Eigenständige Properties und eine Datei pro Definition

Jede Property ist eine vollständige Definition nach Abschnitt 2. Pflichtpräfixe für ihre ID gibt es nicht. Propertydefinitionen werden aus dem jeweiligen Kind- oder Objektvertrag ausdrücklich referenziert; Feldschlüssel konkreter Werte sind keine zusätzlichen Definitionen.

### 6.1 Kurze Property-ID mit Standardfeldname

`spec.name` ist optional. Ohne Angabe gilt exakt die lokale Property-ID als Feldname; `requester` ergibt `requester`. Präfixe werden nicht abgeschnitten. Ein gesetztes `spec.name` bestimmt einen abweichenden Feldnamen und muss dessen Schreibkonvention erfüllen. Weder die ID noch der Feldname darf Punkte enthalten. Beide Formen unterliegen derselben Eindeutigkeitsprüfung je Feldvertrag. Identität und Feldname bleiben verschiedene Größen; Migrationen folgen Abschnitt 9.


<a id="names"></a>

## 7. Identitäten und Referenzen

Package ist ausschließlich Namensraum. Jeder Ordner unter der ausdrücklichen Modellwurzel bildet ein Package; sein relativer Pfad bestimmt den Namen. Es gibt keine zusätzlichen Namespaces. Packages werden mit `/` gegliedert. Lokale IDs sind ungegliederte Namen; Punktnotation dient in Definitionsreferenzen ausschließlich der Aliasqualifizierung. Vollständige lokale IDs importierter Packages sind unmittelbar verfügbar.

Die logische Identität bleibt **Paket-ID plus stabile lokale id**. Sie enthält weder Kind noch Anzeigename, Dateiname oder Versionsnummer. Der Ordnerpfad bestimmt jedoch die Paket-ID; ein anderer Packageordner ändert deshalb die Identität. Eine bestimmte Fassung wird zusätzlich durch ihren Modellstand und dessen Inhaltsbindungen bestimmt. Die Autorenreferenz package/id beziehungsweise Alias.id wird auf dieses kanonische Paar aufgelöst.

| Gegenstand | Paket | Lokale id |
| --- | --- | --- |
| Exchange-Kind | example/connector/vocabulary | Exchange |
| Auftraggeber-Property | example/connector/vocabulary | requester |
| Konkreter Austausch | example/connector/project | list-documents |
| Konkrete API | example/connector/project | api |

Eine lokale ID ist ein einzelner Name nach der Konvention ihrer Definitionskategorie. Sie enthält weder Punkt noch `/`. Die fachliche Gliederung übernimmt das Package. Ein Import macht lokale IDs direkt verfügbar. In Definitionsreferenzen trennt `/` das Package von der lokalen ID; ein Punkt trennt einen ausdrücklich deklarierten Importalias von der lokalen ID. Beispielsweise bezeichnet `Core.Kind` die ID `Kind` aus dem unter `Core` importierten Package, keine lokale ID namens `Core.Kind`.

Für Identität und Kollisionsprüfung macht Groß-/Kleinschreibung keinen Unterschied. Innerhalb eines Pakets dürfen daher auch IDs unterschiedlicher Kinds nicht allein durch Groß-/Kleinschreibung unterscheidbar sein. Die erlaubte Autorenform ist trotzdem streng und muss exakt eingehalten werden: Eine abweichende Schreibweise ist ein Fehler, keine akzeptierte alternative Referenz. Der Prüfer korrigiert oder normalisiert keine Quelldatei automatisch. Lesbare IDs werden nicht bei jedem Wechsel der bevorzugten Bezeichnung geändert oder absichtlich für andere Gegenstände wiederverwendet. Ein optionales title ist keine alternative Adresse.

Die Autorenauflösung verwendet direkt verfügbare Namen und bei Bedarf ausdrücklich qualifizierte Namen. Sichtbar sind ausschließlich das eigene Paket und die in der jeweiligen Definitionsdatei deklarierten Imports; die Bereitstellung eines Modells allein öffnet keine fremden Namen:

- Eine vollständige id wird gegen die Definitionen des eigenen Pakets und aller in dieser Definitionsdatei direkt importierten Pakete aufgelöst. Genau ein kanonisches Ziel muss passen. Beispiel: Kind stammt aus dem importierten Core-Paket, Exchange aus dem importierten Vokabular, api aus dem eigenen Projektpaket.
- Mehrere unterschiedliche Ziele mit derselben sichtbaren id sind auch bei bloßem Unterschied der Groß-/Kleinschreibung mehrdeutig. Es gibt weder einen stillen Vorrang lokaler Definitionen noch eine Auswahl nach dem erwarteten Kind oder der Reihenfolge der Imports. Die Referenz muss ausdrücklich qualifiziert oder die Namenskollision aufgelöst werden.
- Vollständige Qualifizierung verwendet `package/id`, etwa `markitect/core/Kind` oder `example/connector/vocabulary/requester`. Das letzte `/` trennt Package und lokale ID. Das eigene Package ist immer auch vollständig qualifiziert verfügbar, ohne eigenen Import. Ein fremdes Package ist nur nach ausdrücklichem Import vollständig qualifiziert verfügbar. Aliasqualifizierung hat genau die Form `Alias.id`, etwa `Core.Kind` oder `Connector.requester`, mit genau einem Punkt. Weder Alias noch lokale ID enthält selbst Punkte. Es gibt keine schrittweise Namespace- oder Dateipfadsuche und keine Feld-Dereferenzierung.
- Gleiche sichtbare Kurz-IDs machen die Importauswahl für sich genommen nicht ungültig. Erst die Verwendung einer Referenzschreibweise mit mehreren unterschiedlichen Zielidentitäten ist ein Fehler. Eine eindeutige Alias- oder vollständige Qualifizierung bezeichnet das gewünschte eigene oder importierte Ziel. Eine lokale ID kann syntaktisch nicht mit einer Aliasqualifizierung zusammenfallen.
- Nur direkte Imports machen Namen verfügbar. Weitere bereitgestellte Packages und die Inhalte transitiver Modellabhängigkeiten werden nicht automatisch in den sichtbaren Namensraum einer Datei aufgenommen. Ein Zugriff darauf verlangt einen eigenen ausdrücklichen Import in der verwendenden Definitionsdatei.

Alle Referenzstellen verwenden diese Zielauflösung: kind, Property.target, Referenzwerte in spec und Körperlinks. kind und Property.target müssen Typdefinitionen bezeichnen; ein konkreter requester-Wert muss beispielsweise eine Container-Definition bezeichnen. Der erwartete Zieltyp wird erst nach der eindeutigen Auflösung geprüft und nicht als Suchheuristik verwendet.

Fehlende oder mehrdeutige Namen, unbekannte Importaliases und widersprüchliche Importangaben sind Fehler. Es gibt keine Suche außerhalb des eigenen Pakets und der direkten Imports dieser Datei, automatische Versionswahl, Dateipfadinterpretation oder Auflösung über ähnliche Namen. Nach der Auflösung werden kanonische Paket-/ID-Ziele verwendet. Kurze und qualifizierte Referenzen dürfen dasselbe Ziel bezeichnen, erzeugen aber keine verschiedenen Identitäten.

<a id="spelling"></a>

### 7.1 Verbindliche Schreibkonventionen und Kollisionsprüfung

Groß-/Kleinschreibung erzeugt keine verschiedene Identität. Die Schreibkonventionen werden als Sprachregeln erzwungen. Verstöße werden abgewiesen; ein Hinweis auf die erwartete Form ist erlaubt, eine stille Korrektur oder Weiterverarbeitung der abweichenden Form nicht.

| Sprachbezeichner | Vorgeschriebene Form | Beispiele |
| --- | --- | --- |
| Paketname | Aus Ordnerpfad abgeleitete kleingeschriebene Komponenten, mit / als Trenner | markitect/core, example/connector/vocabulary |
| ID einer Kind-Definition | PascalCase | Kind, Property, Exchange, MqttService |
| ID einer Propertydefinition | camelCase | requester, maxAttempts, messageId |
| Effektiver Feldname, einschließlich ausdrücklichem spec.name | Einzelner camelCase-Name ohne Punkte | requester, timeoutSeconds |
| ID eines konkreten Gegenstands | kleingeschriebenes kebab-case | api, list-documents |
| Importalias | PascalCase | Core, Connector |
| YAML-Dateiname | kleingeschriebenes kebab-case mit .yaml | exchange.yaml, max-attempts.yaml |
| Modell-ID | Kleingeschriebene Unicode-Abschnitte, durch . oder - getrennt; Ziffernstart erlaubt, siehe Abschnitt 8.1 | markitect-core, markitect.core, 2026-modell |
| Versionslabel | Dieselbe Zeichengrammatik wie Modell-ID; zusätzlich keine Bereichsschreibweisen oder latest | 1.0, 1.0.0-rc.1, 2026.10 |

Die Kategorie einer Definition folgt aus ihrer aufgelösten Typisierung, nicht aus der Groß-/Kleinschreibung eines geschriebenen kind-Werts. Eine Kind-Definition ist eine Definition des festen Meta-Kinds Kind; eine Propertydefinition ist eine Definition des festen Meta-Kinds Property. Akronyme folgen derselben Wortschreibweise: MqttService und messageId, nicht MQTTService oder messageID. Eine Referenz muss exakt die deklarierte Schreibweise des eindeutigen Ziels verwenden. kind: exchange statt kind: Exchange ist somit ein Fehler, auch wenn die Diagnose das gemeinte Ziel finden kann. Ebenso sind unpassend geschriebene Paketnamen, Aliase, Feldschlüssel und Dateinamen Fehler. Sprachschlüssel wie id, imports, package, as und minCount müssen exakt geschrieben werden.

Lokale IDs aller Definitionskategorien und effektive Feldnamen DÜRFEN KEINE Punkte enthalten. Das gilt für Kinds, Properties und konkrete Gegenstände ebenso wie für ausdrücklich gesetztes `spec.name` und Objektfeldverträge. Eine abweichende Namensangabe kann eine unzulässige ID nicht gültig machen. Die bestehenden PascalCase-, camelCase- und kebab-case-Konventionen gelten weiterhin. Eine fachliche Unterteilung wird im Package ausgedrückt, nicht durch gegliederte IDs. Verstöße werden abgewiesen; Punkte werden weder entfernt noch durch andere Zeichen ersetzt.

Paket-/Aliasqualifier und lokale ID werden anhand der deklarierten Imports und tatsächlichen Ziele aufgelöst, nicht anhand ihrer Groß-/Kleinschreibung oder des erwarteten Zieltyps. Es gibt keine Auswahl durch längsten Prefix oder zufällige Reihenfolge.

Kollisionen werden unabhängig von Groß-/Kleinschreibung geprüft: Paket-IDs, Definitions-IDs innerhalb eines Pakets, Aliase innerhalb einer Definitionsdatei, effektive Feldnamen innerhalb eines Kind-/Objektvertrags sowie Dateinamen im jeweiligen Paket. Kind und Property bekommen dabei keine getrennten ID-Namensräume. Insbesondere sind Command und command im selben Paket ein Konflikt, selbst wenn beide einzeln die Konvention ihrer Kategorie erfüllen. Unterschiedliche Dateinamen lösen diesen Definitionskonflikt nicht.

Der Vergleich dient ausschließlich Identität, Konflikterkennung und Diagnose. Quelltext, gespeicherte kanonische Schreibweise und Referenzen werden nicht automatisch umgeschrieben. Eine falsche Referenz ist bei eindeutigem Ziel ein Schreibfehler; mehrere normalisiert passende Zielidentitäten bleiben mehrdeutig und werden nicht durch die erwartete Kategorie oder eine exakt geschriebene Teilmenge ausgewählt. Datenwerte, Enumwerte, body, title, Topicmuster und Operationskennungen werden nicht allgemein ohne Beachtung der Groß-/Kleinschreibung verglichen; dafür gilt der jeweilige Datenvertrag.

Die Unicode-Grundlage und der genaue Namensvergleich stehen in Abschnitt 7.2. Das zuvor begrenzte ASCII-Profil ist abgelöst. Die zulässigen Zeichen folgen dem dort gebundenen Unicode-Profil; Körperlink-Escapes sind in Abschnitt 4.5 festgelegt; exakte Quellenpositionen folgen dem separaten [Diagnosevertrag](diagnostic-contract.md). Das YAML-/Werteprofil steht in Abschnitt 5.1.

<a id="unicode"></a>

### 7.2 Unicode, Normalform und Namensvergleich

Unicode ist die durchgängige Textgrundlage. Alle Dateien verwenden UTF-8. Unicode ist in body, title und String-/Enumwerten ebenso zulässig wie in Paketnamen, lokalen IDs, Aliasen, effektiven Feldnamen und YAML-Dateinamen. Die festen Sprachschlüssel und Strukturzeichen wie /, Punkt, Bindestrich, .yaml und Körperlinkbegrenzer behalten ihre genaue Schreibweise und Bedeutung.

Die Zeichentabellen sind auf Unicode 15.1.0 festgelegt, entsprechend [UAX #31, Revision 39](https://www.unicode.org/reports/tr31/tr31-39.html). Ein Tabellenwechsel ist eine ausdrückliche Änderung des Sprachprofils. Betriebssystem-, Laufzeit- oder Ländereinstellungen dürfen diese Bindung nicht ersetzen.

Für Bezeichner gilt folgendes Profil:

- Ein Namenswort beginnt mit XID_Start und wird mit XID_Continue fortgesetzt. Connector_Punctuation, einschließlich Unterstrich, ist ausgeschlossen; dies erhält die bereits gewählten Schreibkonventionen. Leerzeichen, Emoji und beliebige Symbole werden dadurch nicht zu Namenszeichen. In normalen Textwerten bleiben sie gemäß dem YAML-Profil aus Abschnitt 5.1 verwendbar.
- `/` gliedert Package-Namen; jede Packagekomponente ist nichtleer. Ein einzelner Ordnername, eine lokale ID, ein Alias und ein effektiver Feldname enthält keine Punkte. Package-Namen sind relative logische Pfade ohne führendes oder abschließendes `/`, leere Komponenten, `.` oder `..`, Backslash, Laufwerkspräfix oder URL-Dekodierung. Es findet keine Dateisystem-Pfadnormalisierung statt. Aliase sind einzelne PascalCase-Namen. Bei konkreten IDs und Dateinamen trennt der ASCII-Bindestrich nichtleere kleingeschriebene Wörter; nach einem Bindestrich darf ein Teil auch mit einer Unicode-Dezimalziffer beginnen, etwa `größe-2`. Das erste Wort beginnt weiterhin mit XID_Start. Die Punktnotation `Alias.id` ist eine Referenzform; Punkte in Modell-IDs, Versionslabels, Dezimalzahlen und gewöhnlichen Textwerten folgen ihren eigenen Regeln.
- PascalCase und camelCase werden anhand des ersten Buchstabens mit Groß-/Kleinschreibung des Namens geprüft. Titlecase-Zeichen gelten dabei als große Initiale. Ohne solche Buchstaben entfällt diese Initialenvorgabe, etwa bei 文档. Für kleingeschriebene Namen müssen alle Buchstaben mit Groß-/Kleinschreibung klein sein. In PascalCase-/camelCase-Wörtern sind aufeinanderfolgende Großbuchstaben weiterhin ausgeschlossen; kombinierende Marken werden bei dieser Prüfung übersprungen. Somit bleiben Überweisung, gültigBis und MqttService passende Beispiele. Diese Kategorien folgen weiter aus der Typisierung.

**Strenge Autorenform:** Namen, Dateinamen und geschriebene Referenzen müssen bereits in [Unicode-Normalform NFC](https://www.unicode.org/reports/tr15/) vorliegen. Eine zerlegte Schreibweise, etwa U+0075 U+0308 anstelle von U+00FC für ü, wird mit Diagnose abgewiesen. Weder eine Quelldatei noch ihr dekodierter Name wird still korrigiert. Eine Referenz muss außerdem exakt die deklarierte Schreibweise des eindeutigen Ziels verwenden. Gleichwertigkeit im Kollisionsvergleich macht eine abweichende Autorenreferenz nicht gültig.

**Kollisionsschlüssel:** Für Namen verwenden wir Unicode Canonical Caseless Matching mit vollständigem Default Case Folding, kulturunabhängig und ohne türkische Sonderbehandlung. Präzise gilt K(s) = NFD(toCasefold(NFD(s))) gemäß Definition D145 des Unicode-Standards. Zwei Namen kollidieren genau dann, wenn ihre Schlüssel gleich sind, jeweils im bereits festgelegten Geltungsbereich: Paket, Definition, dateilokaler Alias, Feldvertrag oder Dateiauswahl. Der Vergleichsschlüssel ist ein internes Prüfmittel, keine neue gespeicherte Autorenform. Insbesondere kollidieren straße und strasse sowie Straße und Strasse. Auch eine entsprechende Referenzabweichung wird abgewiesen; der Prüfer wählt nicht eine exakt geschriebene Teilmenge aus mehreren kollidierenden Zielen.

NFC und dieser Vergleich bewirken keine allgemeine Umschrift: ä und ae bleiben verschieden. Eine zusätzliche NFKC-Kompatibilitätsfaltung ist nicht gewählt. body, title, String- und Enumwerte werden weder normalisiert noch ohne Beachtung der Groß-/Kleinschreibung verglichen; dort bleibt der dekodierte Text erhalten und es gilt der jeweilige Datenvertrag. Die NFC-Anforderung betrifft auch Feldschlüssel, aber nicht deren gewöhnliche Textwerte. Herkunft und ursprüngliche Dateibytes bleiben für Diagnosen verfügbar.

Die gemeinsame Grammatik für Modell-IDs und Versionslabels steht in Abschnitt 8.1. Sie verwendet dieselben Unicode-Tabellen, NFC-Autorenform und Kollisionsschlüssel, erlaubt jedoch am Beginn jedes Abschnitts zusätzlich Unicode-Dezimalziffern. Diese Erweiterung gilt nicht automatisch für Packageordner, Definitions-IDs, Importaliase oder Dateinamen.

<a id="packages"></a>

### 7.3 Ordner und Packages

Das kanonische Modell wird wie Dokumentation in Ordnern strukturiert. Die ausdrücklich bestimmte Modellwurzel ist selbst kein Package und darf keine Definitionsdateien unmittelbar enthalten. Unterhalb dieser Wurzel ist jeder Ordner ein Package, auch ein Zwischenordner ohne eigene Definitionen. Jede Definition liegt somit in einem benannten Unterordner und gehört genau zum Package ihres unmittelbar enthaltenden Ordners. Unterordner gehören zu eigenen Packages; eine rekursive Erfassung macht deren Definitionen nicht zu Mitgliedern des Elternpackages. Das Package des Elternordners wird nicht geerbt.

**Autorenkonvention:** Packages werden nach fachlichem Zusammenhang beziehungsweise Feature geschnitten (package by feature / vertical slices). Zusammengehörige Kinds, Properties und konkrete Gegenstände stehen gemeinsam beim jeweiligen Feature. Ihre Definitionsart ist kein Grund für getrennte Ordner wie kinds/, properties/ oder instances/. Ein weiterer Unterordner entspricht einem fachlichen Teilbereich; jede Definition behält ihre eigene Datei. Diese Konvention ist eine Entscheidung zur Modellorganisation, kein zusätzliches Core-Konzept und kein syntaktisches Verbot bestimmter Ordnernamen. Der Strukturprüfer leitet aus einem Ordnernamen keine fachliche Angemessenheit ab.

Der Package-Name entspricht dem nicht leeren Pfad des unmittelbar enthaltenden Ordners relativ zur Modellwurzel und wird nicht zusätzlich in einer package.yaml oder in jeder Definition gepflegt. Es gibt weder einen leeren Package-Namen noch ein Package namens . oder einen Ersatznamen aus der Modell-ID. Die Ordnerkomponenten werden mit `/` verbunden; dies ist unmittelbar der relative Ordnerpfad.

Ein Ordnername entspricht genau einer bisherigen kleingeschriebenen Package-Komponente aus Abschnitt 7.2; ein Punkt im einzelnen Ordnernamen erzeugt keine zusätzlichen Komponenten und ist in dieser Form unzulässig. Groß-/Kleinschreibung und Unicode-Kollisionen werden unabhängig vom Verhalten des Dateisystems geprüft, einschließlich straße/strasse. Namen müssen bereits in NFC und in der vorgeschriebenen Form vorliegen; keine automatische Korrektur.

**Dateierfassung:** Die Erfassung MUSS den vollständigen Modellbaum einschließlich aller Unterordner durchlaufen; es gibt keine impliziten Ignore-Regeln. Jeder Unterordner bleibt ein Package gemäß diesem Abschnitt. Für reguläre Dateien gelten abschließend folgende Regeln:

- Jede Datei mit der exakten Endung `.yaml` ist eine Definitionsdatei und MUSS die vollständige Definitionshülle erfüllen. Eine zusätzliche YAML-Datendatei ohne Definitionshülle ist ein Fehler und wird nicht still übersprungen.
- Ausgenommen sind ausschließlich das ausdrücklich gebundene `model.yaml`-Manifest und seine unmittelbar danebenliegende `model-lock.yaml`, sofern sie im Modellbaum liegen. Die Ausnahme gilt für genau diese Pfade mit exakt geschriebenen Dateinamen, nicht pauschal für gleichnamige Dateien in Packages.
- `.yml` sowie Groß-/Kleinschreibungsvarianten der Endungen `.yaml` und `.yml` sind Fehler. Sie werden weder ignoriert noch umbenannt oder als alternative Autorenform akzeptiert. Für gültige Definitionsdateien gelten zusätzlich die strengen Dateinamensregeln aus Abschnitt 7.1.
- Andere reguläre Dateien, etwa Markdown, Bilder oder JSON, werden nicht als Definitionen gelesen und gehen nicht in den kanonischen Modellinhalt ein. Das Umleitungsverbot aus Abschnitt 8.1 gilt weiterhin für den gesamten erfassten Modellbaum.

Die Definitionen werden aus ihren jeweiligen Ordnern erfasst. Es gibt keine gesonderte definitionFiles-Liste pro Package und keinen zusätzlichen definitions/-Pflichtordner. Die lokale id steht weiterhin ausdrücklich in jeder Definitionsdatei; sie wird nicht aus dem Dateinamen abgeleitet. Umbenennen einer Datei im selben Ordner erhält ihre Identität. Verschieben einer Datei in einen anderen Packageordner oder Umbenennen/Verschieben eines Packageordners ändert die betroffenen qualifizierten Identitäten und verlangt angepasste Referenzen. Ein anderer physischer Speicherort der gesamten Modellwurzel erhält die relativen Package-Namen.

Die Ordnerhierarchie erzeugt keine impliziten Imports, Vererbung, Enthaltenheitsbeziehungen zwischen Definitionen oder fachlichen Beziehungen. Eltern-, Kind- und Geschwisterpackages müssen in jeder verwendenden Datei ausdrücklich importiert werden. Ein Package hat keine eigene Version, Abhängigkeitsliste oder Veröffentlichungsdatei. Modell bezeichnet die gemeinsame Versionierungs- und Verteilungseinheit und kann mehrere Packages enthalten. Abschnitt 8.1 bestimmt Modellidentität, Wurzel und Manifestform. Die Modellidentität erzeugt keinen automatischen Package-Präfix und keinen zusätzlichen Teil der Definitionsidentität. Abschnitt 8.2 bestimmt das Lockformat; Abschnitt 8.3 bestimmt die kanonischen Digestbytes.

Das ausdrücklich gebundene `model.yaml` und die danebenliegende `model-lock.yaml` sind Modellmetadaten und keine Definitionsdateien; auch bei `root: .` erzeugen sie kein Wurzel-Package. Definitionen unmittelbar in der Modellwurzel sind Fehler.


<a id="imports"></a>

## 8. Dateilokale Imports und gebundene Modelle

Importierbar sind ausschließlich Packages mit mindestens einer eigenen Definition. Ein leerer Ordner oder ein Package mit ausschließlich belegten Unterpackages ist nicht importierbar; seine Unterpackages müssen jeweils selbst importiert werden. Die physische Anwesenheit leerer Ordner öffnet daher keine Namen und begründet keine Importbindung.

Das optionale Hüllenfeld imports enthält die ausdrücklich in dieser Datei sichtbaren fremden Packages. Ohne imports oder bei imports: [] sind ausschließlich Definitionen des eigenen Packages sichtbar. Ein unbekanntes oder im gebundenen Eingabesatz nicht bereitgestelltes Package ist ein Fehler. Ein fremdes Package muss aus dem eigenen Modell oder einer ausdrücklich deklarierten direkten Modellabhängigkeit stammen. Transitive Modelle werden zur Vertragsprüfung geladen; ein direkter Zugriff auf deren Packages verlangt eine eigene direkte Abhängigkeit im verwendenden Modellmanifest. Imports wählen weder eine neue Version noch einen physischen Dateisystempfad. Die Bereitstellung eines externen Modells und seine Version öffnen allein keine Namen; umgekehrt ist ein Import keine Installationsanweisung.

**Importform:** Jeder Import ist ein geschlossenes Objekt mit erforderlichem `package` und optionalem `as`; weitere Felder sind Fehler. Beide Angaben sind Strings. Falls `as` vorhanden ist, muss es einen nichtleeren gültigen Alias enthalten. Eine String-Kurzform wird nicht parallel geführt. as deklariert einen dateilokalen Namen in PascalCase und verändert weder Package-Identität noch verfügbare Definitionenmenge.

Ohne as ist ein Package nur unter seinem vollständigen Package-Namen qualifizierbar. Mit as sind sowohl vollständiger Name als auch kurzer Alias zulässig. Die vollständigen lokalen IDs des importierten Packages werden in beiden Fällen unmittelbar verfügbar: Kind, Core.Kind und markitect/core/Kind können dasselbe Ziel bezeichnen. Ein Alias macht den Import nicht zu einem nur qualifiziert sichtbaren Import.

Aliase gelten nur in der deklarierenden Definitionsdatei. Imports anderer Dateien, die Imports einer referenzierten Kind-/Propertydefinition, Elternpackages und transitive Modellabhängigkeiten erweitern diese Sicht nicht. Referenzwerte und Körperlinks werden immer im Dateikontext aufgelöst, in dem sie geschrieben sind. Auch mehrere Packages derselben Verteilungseinheit brauchen ihre eigenen dateilokalen Imports.

Doppelte Aliase und doppelte Imports desselben Packages werden unabhängig von Groß-/Kleinschreibung bereits als Deklarationsfehler abgewiesen. Gleichnamige Definitionen aus unterschiedlichen sichtbaren Packages sind dagegen zulässig. Jede tatsächlich verwendete Referenz muss genau ein Ziel ergeben; konkurrierende Kurz-, Alias- oder vollständige Schreibweisen erhalten keinen Vorrang nach exakter Schreibweise, Ziel-Kind oder Prefixlänge. `Core.Kind` und `core.Kind` sind keine verschiedenen Adressen; bei deklariertem `Core` ist `core.Kind` ein Schreibfehler. Abschnitt 7.2 besitzt das Unicode-Profil.

**Inhaltsbindung bleibt erforderlich:** Externe wiederverwendbare Modelle werden mit genauer Version und konkretem Inhaltsdigest gebunden. Versionsbereiche, latest und stiller Inhaltsaustausch unter einer gebundenen Version sind nicht vorgesehen. Der gebundene Eingabesatz muss jede kanonische Package-/ID-Identität eindeutig bereitstellen; mehrere Imports oder Aliase umgehen keinen Konflikt. Das lokale Modell wird durch seinen festen Quell-/Modellstand gebunden; jede lokale Änderung verlangt keine neue Veröffentlichungsnummer.

<a id="models"></a>

### 8.1 model.yaml: Identität, Wurzel und Abhängigkeiten

Jedes zusammengehörige Modell besitzt ein Manifest mit dem festen Dateinamen model.yaml. Manifest bezeichnet die Aufgabe dieser Datei; Modell ist die gemeinsame Versionierungs- und Verteilungseinheit für mehrere Packages. model.yaml ist Bereitstellungsmetadatum, keine zusätzliche Definition und kein neuer privilegierter Core-Kind. Jede eigentliche Definition behält ihre einheitliche Hülle.

| Feld | Gewählter Vertrag |
| --- | --- |
| formatVersion | Erforderliche Ganzzahl; bindet Manifestformat und Sprachvertrag. v1 verwendet 1. Unbekannte Fassungen werden nicht still als 1 interpretiert. |
| id | Erforderliche stabile, nichtleere Modellidentität; unabhängig von Checkout-/Installationsordner, Speicherort, Modellwurzel, Package-Namen und Version. |
| version | Genaue textuelle Veröffentlichungsfassung aller enthaltenen Packages. Bei lokaler unveröffentlichter Arbeit optional; für Veröffentlichung und als bereitgestellte Abhängigkeit erforderlich. |
| root | Erforderlicher nichtleerer relativer Verzeichnispfad, bezogen auf den Ordner der jeweiligen model.yaml. Es gibt keinen geratenen Standardordner. |
| dependencies | Erforderliches Mapping von direkt benötigter Modell-ID auf genaue textuelle Version; ohne Abhängigkeiten {}. Keine Packages, Importaliase, Versionsbereiche oder latest. |

Die Hülle ist geschlossen: unbekannte oder falsch geschriebene Felder sind Fehler. Eine Abhängigkeit muss exakt die deklarierte Modell-ID verwenden. Modell-IDs und Versionslabels sind YAML-Strings; numerische Werte werden nicht konvertiert. Die physische root-Pfadschreibweise steht weiter unten in diesem Abschnitt.

**Gemeinsame Zeichengrammatik für Modell-IDs und Versionslabels:**

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
- `x` ist in Versionslabels nicht reserviert und hat keine Platzhalterbedeutung. `"x"`, `"1.x"` und `"release.x"` sind gewöhnliche exakte Textlabels. Eine Anforderung auf `"1.x"` verlangt genau diese Fassung und erfüllt sich nicht durch `"1.0"` oder `"1.7"`. `latest` bleibt als Versionslabel verboten; `*`, `^`, `~`, Vergleichs- und Bereichsoperatoren gehören weiterhin nicht zur Labelgrammatik. Die gemeinsame Grammatik für Modell-IDs bleibt unverändert.

Der vollständige Unicode-Kollisionsvergleich aus Abschnitt 7.2 gilt für Modell-IDs innerhalb der bereitgestellten Modellmenge und für Versionslabels innerhalb desselben Modells. straße und strasse dürfen dort keine verschiedenen Identitäten bezeichnen. Unterschiedliche Modelle dürfen dasselbe Versionslabel verwenden. Anforderungen und Lockeinträge müssen trotzdem exakt die deklarierte Autorenform verwenden; eine kollidierende Schreibweise ist keine gültige alternative Referenz. Der erste Sprachumfang bindet weiterhin nur eine Fassung je Modell-ID; die Kollisionsprüfung eröffnet keine Mehrversionsauswahl.

Eine Version ist ein exaktes Textlabel. Manifest, direkte Abhängigkeitsanforderung und Lock verwenden denselben dekodierten Text. Komponenten werden nicht als Zahlen verglichen, nicht ergänzt und begründen weder Reihenfolge noch Kompatibilität. `"1.0"` und `"1.0.0"` sind unterschiedliche Fassungen. Versionsbereiche und `latest` sind verboten; der genaue Inhalt ist zusätzlich durch den Digest gebunden.

Eine Abhängigkeit verlangt die exakt deklarierte Identität und Version der bereitgestellten Fassung. Fehlende Modelle, abweichende Schreibweisen, widersprüchliche direkte oder transitive Versionsanforderungen und mehrere Kandidaten derselben Modell-ID sind Fehler. v1 bindet genau eine Fassung pro Modell-ID. Externe Inhalte werden zusätzlich durch das Lock gemäß Abschnitt 8.2 gebunden.

Ein mit Definitionen belegtes Package gehört im gebundenen Eingabesatz genau einem Modell. Zwei Modelle dürfen dasselbe Package auch bei unterschiedlichen lokalen IDs nicht aufteilen. Gemeinsame Zwischenordner bleiben möglich: getrennte Modelle mit markitect/core und markitect/http können beide den definitionslosen Zwischenordner markitect enthalten. Sobald ein Package Definitionen enthält, stammen diese aus genau einem Modell; ein anderswo lediglich als leerer Zwischenordner vorkommender gleicher Name liefert keine weiteren Definitionen. Keine Lade- oder Importreihenfolge entscheidet über die Zugehörigkeit.

**Modellübergreifende Namensbindung:** In den gemeinsamen Package-Namensraum gehen ausschließlich belegte Packagepfade und alle daraus ableitbaren nichtleeren Zwischenpfade ein. Unterschiedlich geschriebene Namen dieser Menge, die im Unicode-Vergleich kollidieren, sind Fehler. Rein leere Teilbäume binden modellübergreifend keine Namen. Beispielsweise kollidiert ein belegtes straße/feature in einem Modell nicht mit einem leeren strasse/empty in einem anderen Modell. Enthält strasse/empty dagegen eine Definition, kollidieren die ableitbaren Zwischenpfade straße und strasse. Innerhalb jedes einzelnen Modellbaums werden weiterhin sämtliche Ordnernamen einschließlich leerer Teilbäume auf gültige Schreibweise, NFC und Unicode-Kollisionen geprüft. Jeder Ordner bleibt lokal ein Package; leere Packages bleiben nicht importierbar.

Imports bleiben dateilokal. Sie machen ausschließlich die tatsächlich importierten Packages sichtbar. Ein Package aus demselben Modell benötigt einen Import, sobald es nicht das eigene Package der Datei ist. Die Verfügbarkeit eines transitiven Modells eröffnet keinen direkten Import ohne eigene direkte Modellabhängigkeit. model.yaml wird weder aus Imports noch aus Prosa ergänzt; es ist keine Installations- oder Ausführungsanweisung.

Ein anderer physischer Speicherort des gesamten Modells, ein anderer Installationsordner oder eine identitätserhaltende Verlagerung seiner Wurzel ändern seine Modell-ID und die relativen Package-Namen nicht. Eine ausdrückliche Änderung der Modell-ID ändert die Adresse der Modellabhängigkeit, aber erzeugt nicht automatisch neue Package-/Definitionsidentitäten. root bestimmt, welche Dateien unter welchen relativen Package-Namen geladen werden; eine Änderung ist anhand dieses neuen Bestands zu prüfen.

**Modellabhängigkeiten:** Der gerichtete Graph der model.yaml-dependencies muss azyklisch sein. Knoten sind die eindeutig gebundenen Modell-IDs, Kanten die ausdrücklich deklarierten direkten Anforderungen. Im vollständigen erreichbaren Modellumfang darf kein Abhängigkeitsweg zum Ausgangsmodell zurückführen. Das verbietet Selbstabhängigkeiten als Zyklen der Länge eins ebenso wie wechselseitige oder längere Zyklen. Auch exakt passende Versionen oder Digests machen einen solchen Zyklus nicht gültig. Mehrere Modelle dürfen dieselbe Abhängigkeit verwenden, sofern die gemeinsame genaue Versions- und Inhaltsbindung passt.

Ein Modellzyklus ist ein Sprachfehler und liefert failed, keine bloß unvollständige Prüfung. Für den betroffenen ungültigen Modellumfang gibt es weder kanonische Bytes noch Digest. Die Regel benötigt keine Sonderbehandlung für Selbstabhängigkeiten. Fachlich untrennbar wechselseitig abhängige Teile können zu einem gemeinsamen Modell mit mehreren fachlichen Packages gehören. Gewöhnliche Definitionsreferenzen und Package-Imports innerhalb desselben Modells dürfen weiterhin Zyklen bilden; die Objektvertragsregeln aus Abschnitt 5.3 gelten zusätzlich. Aus Prosa oder bloßen Dateilinks werden keine dependencies ergänzt.

**Physische Wurzelpfadregeln:**

- root ist entweder . oder ein nichtleerer relativer Ordnerpfad, bezogen auf den Ordner der jeweiligen model.yaml. Das Ziel muss bereits als Verzeichnis existieren; fehlende Ziele und Dateien statt Verzeichnissen sind Fehler. . bezeichnet ausschließlich den Manifestordner selbst.
- ASCII-/ ist auch unter Windows der einzige Trenner. Jede relative Komponente folgt der bestehenden kleingeschriebenen Unicode-Packageordnergrammatik aus Abschnitt 7.2: XID_Start am Beginn, danach XID_Continue, keine Connector_Punctuation. Die neue Modell-ID-/Versionslabelgrammatik erweitert diese Ordnergrammatik nicht. Insbesondere sind Punkte, Bindestriche und Ziffernstarts in einzelnen root-Ordnerkomponenten nicht zusätzlich erlaubt.
- Es gibt keine führenden oder abschließenden /, leeren Komponenten, eingebetteten . oder .., Backslashes, Laufwerkspräfixe oder absoluten Pfade. Die Autorenform wird vor einer Pfadinterpretation geprüft; ./model, docs//model und model/ werden nicht still normalisiert. Es gibt keine Variablen-, Home-, URL- oder sonstige Ersetzung.
- Die geschriebenen Komponenten müssen bereits NFC sein und exakt den tatsächlichen Verzeichnisnamen entsprechen, einschließlich Groß-/Kleinschreibung und Unicode-Schreibweise. Ein auf einem toleranten Dateisystem auffindbares Model ersetzt ein geschriebenes model nicht. Mehrere im Unicode-Kollisionsvergleich passende Verzeichnisse bleiben ein Fehler; die exakt geschriebene Teilmenge erhält keinen Vorrang.
- Symlinks, Junctions und vergleichbare Reparse-Umleitungen im relativen Wurzelpfad oder im erfassten Modellbaum sind verboten und werden nicht verfolgt. Auch die ausdrücklich geladenen Manifest- und Lockdateien werden nicht über solche Umleitungen gelesen. Der physische Manifestordner bildet den Bezugspunkt; der absolute Ablagepfad darüber erhält keine Package-Namenskonvention.
- Geladene Modellwurzeln dürfen weder dasselbe physische Verzeichnis bezeichnen noch ineinander liegen. Die Prüfung erfolgt für die gebundene Modellmenge vor der Definitionserfassung. Bei root: . müssen weitere gebundene Modelle deshalb außerhalb dieses Modellbaums liegen. Gemeinsame logische, leere Packagepräfixe aus getrennten physischen Wurzeln bleiben möglich.

Gültige Formen sind ., model, docs/model und überweisung/文档, sofern die exakt geschriebenen Verzeichnisse vorhanden sind. Ungültig sind beispielsweise ./model, model/, docs//model, ../shared/model, Model und C:\\models\\connector. Die absoluten Speicherorte der Modelle bleiben frei; eine identitätserhaltende physische Verlagerung verändert weder Package-Namen noch kanonische Digestbytes. Die Wurzel selbst bildet weiterhin kein Package und darf keine Definitionsdateien unmittelbar enthalten (Abschnitt 7.3).

<a id="lock"></a>

### 8.2 model-lock.yaml: vollständige externe Inhaltsbindung

Das generierte Lock wird als model-lock.yaml neben dem Einstiegsmanifest gespeichert und zusammen mit dem Modell versioniert. Es bindet sämtliche direkten und transitiven externen Modellabhängigkeiten genau einmal. Die eigene lokale Modellfassung wird durch den aktuellen Quell-/Modellstand gebunden und nicht als eigener externer Eintrag im Lock wiederholt.

Die geschlossene Form besteht aus genau formatVersion und models. Die aktuelle formatVersion ist die Ganzzahl 1. models ist ein Mapping von Modell-ID auf ein Objekt mit genau version und digest. Ohne externe Abhängigkeiten ist models: {} die vollständige Tabelle. Jeder Schlüssel ist die exakt geschriebene NFC-Modell-ID; der Unicode-Kollisionsvergleich aus Abschnitt 7.2 gilt auch hier. version ist die genaue textuelle Fassung. digest hat die Form sha256: mit genau 64 nachfolgenden kleingeschriebenen Hexadezimalziffern. Unbekannte Felder, Schreibabweichungen, fehlende Angaben und Kollisionen sind Fehler.

**Vollständigkeit:** Aus den direkten Anforderungen des Einstiegsmanifests und den gebundenen model.yaml-Manifesten der Abhängigkeiten muss genau die externe Locktabelle entstehen. Fehlende, überzählige oder widersprüchliche Einträge sind Fehler. Es gibt keine zusätzliche manuelle Liste der direkten Abhängigkeiten und keine zweite Beschreibung der Graphkanten im Lock. Pro Modell-ID gilt eine Fassung; weder Importaliase noch Lade- oder Tabellenreihenfolge lösen Versions- oder Inhaltskonflikte auf.

**Prüfung:** Modell-ID, genaue Version und berechneter Inhaltsdigest müssen gleichzeitig passen. Der Prüfer kontrolliert alle erreichbaren Manifestanforderungen und anschließend jede Inhaltsbindung. Andere Inhalte unter derselben gebundenen Version werden abgewiesen. Ein Prüflauf schreibt, ergänzt oder aktualisiert das Lock nicht automatisch. Ein fehlendes Lock wird nicht durch eine still erzeugte neue Bindung ersetzt. Änderungen der Abhängigkeiten verlangen eine erneute Prüfung der vollständigen Bindung; lokale Definitionsänderungen allein verlangen kein neues Lock. Die Root-Locktabelle bestimmt die gemeinsame externe Bindung des ausgewerteten Modellstands; eine etwa mitgelieferte weitere Lockdatei eines Abhängigkeitsmodells ersetzt oder erweitert diese Tabelle nicht still.

**Digestumfang:** Ein Modelldigest bindet ausschließlich den eigenen Modellinhalt einschließlich eigener Modellmetadaten, deklarierter direkter Abhängigkeitsanforderungen, Packages und Definitionen. Er enthält keine rekursiv eingebetteten Digests fremder Modellinhalte. Die vollständige konkrete Bindung der transitiven Inhalte trägt die Locktabelle. Das vermeidet rekursive Digestdefinitionen; gewöhnliche Definitionsreferenzen bleiben Referenzen auf kanonische Identitäten und werden nicht unendlich textuell entfaltet. Die genaue Projektion, Defaults, Reihenfolgen und Bytekodierung stehen in Abschnitt 8.3.

**Bereitstellung:** Installationspfade, Ablageorte und Downloadadressen sind Bereitstellungs-/Provenienzeingaben und keine Lockfelder. Die bereits gewählte Modell-ID wird nicht um einen Speicherort ergänzt. Die Bereitstellung liefert Inhalte, deren Identität, Version und Digest geprüft werden müssen; der Lockeintrag allein ist keine Installationsanweisung.

<a id="canonical-bytes"></a>

### 8.3 Kanonische Bytes und SHA-256: model-digest/1

Jeder Modelldigest ist SHA-256 über die nachfolgenden kanonischen Bytes. Die Kodierung beschreibt den ausgewerteten eigenen Modellinhalt. Vor jeder Projektion werden das ausgewählte Modell und seine vollständige Abhängigkeitshülle validiert: Manifeste, Schreibweisen/Kollisionen, feste Core-Selbstbeschreibung, Typverträge, Werte, Kardinalitäten und Körperlinks. Auch beim Laden festgestellte Fehler dürfen nicht durch eine spätere Map-Darstellung verloren gehen. Eine ungültige Fassung erhält keinen Digest; eine nicht abgeschlossene Inhaltsprüfung ist kein gültiger Digestnachweis. Ein gültiges Abhängigkeitsmodell kann unabhängig von einem fehlerhaften, es verwendenden lokalen Modell geprüft und gehasht werden.

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

**Ausgeschlossene Daten.** Dateinamen, root, Installations-/Downloadpfade, Quellbytes, Kommentare, Fundstellen, Prüfberichte, Lockdateien und fremde Modellinhalte sind keine Felder dieser Projektion. Ein eigener Digest enthält insbesondere keine rekursiven fremden Inhaltsdigests; die konkrete externe Bindung bleibt im Lock. Leere Packages werden nicht aufgenommen; Zwischenpackages ergeben sich aus den vollständigen Packagepfaden der enthaltenen Definitionen. Dieselben belegten Pfade und ihre ableitbaren Zwischenpfade bestimmen auch die modellübergreifende Namensbindung nach Abschnitt 8.1. Leere Packages sind nicht importierbar; das Hinzufügen oder Entfernen lokal gültiger leerer Teilbäume verändert daher weder eine zulässige Importbindung noch die modellübergreifende Kollisionsprüfung. Das verändert nicht die Autorenregel, dass jeder Ordner ein Package ist, und es erlaubt keine Umgehung lokaler Schreib- oder Kollisionsfehler bei der vorherigen Validierung. Git muss keine leeren Verzeichnisse erhalten, um dieselben kanonischen Inhaltsbytes zu erzeugen.

<a id="changes"></a>

## 9. Umbenennungen und Migrationen

| Änderung | Vertrag |
| --- | --- |
| title ändern | Identität bleibt; geänderte Darstellung kann im neuen Stand erfasst werden. |
| Datei innerhalb desselben Packageordners umbenennen | Identität bleibt, wenn id bestehen bleibt und die Datei weiter geladen wird. |
| Datei in einen anderen Ordner verschieben oder Packageordner umbenennen/verschieben | Package-Name und damit Identität ändern sich; Referenzen ausdrücklich migrieren. |
| Importalias einer Datei ändern | Nur damit qualifizierte Autorenreferenzen dieser Datei anpassen; andere Dateien, eindeutige direkt verfügbare Namen und kanonische Zielidentitäten bleiben. |
| Propertyfeld sender zu requester ändern | Neuer Feldschlüssel gilt im Typvertrag; konkrete specs müssen migriert werden. |
| Dabei Propertyidentität erhalten | id in der Propertydatei beibehalten und spec.name auf den gewünschten neuen Feldnamen setzen; etwa requester bei stabiler ID sender. |
| spec.name auslassen | Die vollständige lokale Property-ID ist der effektive Feldname. |
| Abweichendes spec.name entfernen | Der Feldname fällt auf die vollständige lokale ID zurück; betroffene konkrete specs müssen gegebenenfalls migriert werden. |
| ID bei fehlendem spec.name ändern | Identitätsmigration und Änderung des Standardfeldnamens; ein ausdrücklich gesetzter bisheriger Name kann den Feldschlüssel erhalten. |
| Property unter anderem Kind verwenden | Dieselbe eigenständige Propertydefinition ausdrücklich referenzieren, wenn derselbe Feldvertrag gewünscht ist. |
| Lokale id oder Paketzugehörigkeit ändern | Ausdrückliche Identitätsmigration und Anpassung bekannter Referenzen. |

Eine stabile Property-ID bewahrt Verweise auf den Feldvertrag, macht den alten Feldnamen aber nicht weiter gültig. id ist immer ausdrücklich erforderlich. Ein ausdrücklich gesetztes spec.name trennt den Feldnamen von dieser ID; ohne name gilt die ID als Standardfeldname. Ändert ein Autor zusätzlich die ID, ist das eine eigene Identitätsmigration; der Compiler errät keine Umbenennungsabsicht und erzeugt bei einer Feldumbenennung keine neue Property-ID.

Eine neue gemeinsam versionierte Modellfassung kann dieselben logischen Identitäten mit geänderten Verträgen enthalten. Die neue Bindung wird ausdrücklich gewählt und betroffene Modelle beziehungsweise Realisierungen werden gegen sie betrachtet. Weder eine stabile ID noch eine Versionsnummer beweist semantische Kompatibilität.

Veröffentlichte alte Fassungen und darauf bezogene Nachweise werden nicht rückwirkend geändert. Automatische Weiterleitungen, versteckte Namenshistorien und Zuordnung über ähnliche Namen sind nicht Bestandteil des minimalen Cores. Bei echter Identitätsänderung müssen bekannte kind-, target-, Wertreferenzen und explizite Körperlinks migriert werden. Freie Prosa benötigt zusätzlich Review.

<a id="processing"></a>

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

Das Ergebnis macht Kind-, Property- und konkrete Definitionen gemeinsam adressierbar. Ein Inhaltsdigest bindet den Modellinhalt und die gewählten Verträge; Quellherkunft bleibt nachvollziehbar. Die kanonische Datenkodierung steht in Abschnitt 8.3. Produktive Fehlercodes und Berichtsformate bleiben getrennte Werkzeugausarbeitung.

<a id="conformance"></a>

## 11. Konformität und Abgrenzung

Ein vollständiges Ergebnis MUSS Hülle, Ein-Datei-Regel, feste Core-Selbstbeschreibung, Namen und Kollisionen, Manifeste und Wurzeln, Modellabhängigkeitsgraph, Imports, Typ-/Objektverträge, Werte, Kardinalitäten, Referenzen, Körperlinks sowie die externe Lockbindung prüfen. Ungültige Eingaben sind `failed`; reine Ressourcenabbrüche sind `not_completed` gemäß Abschnitt 5.3. Eine Teilprüfung DARF NICHT als vollständig gültiges Modell ausgegeben werden. Für ungültige oder inhaltlich unvollständig geprüfte Modellumfänge gibt es weder kanonische Bytes noch Digest. Das gesonderte Diagnosekoordinatenprofil ist keine zusätzliche Inhaltsprüfung; sein unvollständiger Status ändert weder das Ergebnis vollständig geprüfter Inhalte noch deren Digest. Ein Digest beweist nur den gemäß Abschnitt 8.3 geprüften Inhalt seines Modellumfangs.

Strukturelle Gültigkeit beweist weder die Wahrheit oder Vollständigkeit verbindlicher Prosa noch eine geeignete Modellierung, korrekte Realisierung, ausgeführte Laufzeitnachweise oder menschliche Akzeptanz. Fachliche Widersprüche bleiben semantische Befunde. Abgeleitete Prüflisten sind keine zweite kanonische Wahrheit.

Nicht Teil von v1 sind kompakte oder eingebettete Definitionen, zusätzliche Namespaces, benutzerdefinierte Metaklassen, automatische Namenskorrekturen oder Weiterleitungen, Versionsbereiche, Mehrversionsimporte und eine allgemeine formale Wertebedingungs- oder Ausführungssprache. Zusätzliche Fach-Kinds aus Bibliotheken erhalten keine privilegierte Compilersemantik.

Kontextbildung, semantischer Index, Änderungsbewertung, Realisierungszuordnung, Prüfplanung, Delegation, Integration und Apply sind keine durch diese Spezifikation festgelegten Verfahren. Die [separaten Verfahrensvorschläge](procedure-proposals.md) sind nicht gewählt. Produktive Go-Implementierung, Konformitätskorpus, Fehlercodes, Ausgabeformate, Bereitstellung und Migration sind gesonderte Implementierungsarbeit; daraus folgen keine weiteren Pflicht-Kinds und keine Änderung der v1-Regeln.

<a id="121-stand-der-definition-des-kanonischen-modells"></a>

Der gewählte minimale Sprachumfang ist als v1-Designbaseline abgeschlossen. Grundform, fester Core, Werte und Listen, Unicode und Schreibkonventionen, Identitäten und Imports, Packages, Modelle und Wurzeln, Abhängigkeiten und Lockbindung sowie kanonische Bytes sind festgelegt. Die bekannte Command/command-Kollision im informativen Exchange-Beispiel ist ein Vokabularkonflikt und keine offene Sprachentscheidung. Spätere Implementierungsbefunde können ausdrücklich dokumentierte Präzisierungen oder Korrekturen erfordern; weitere Sprachmittel verlangen eine ausdrückliche Erweiterungsentscheidung. Der frühere Abschlussabgleich bleibt in der [Sprachgeschichte](language-history.md#121-stand-der-definition-des-kanonischen-modells) erhalten.

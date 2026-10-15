# Herleitung des minimalen Cores und Connector-Gegenproben

Stand: 15. Oktober 2026. **Status: Gesprächsherleitung, frühere Varianten und begrenzte Prüfberichte.** Der aktuelle, zusammenhängende Stand steht im [konsolidierten Sprachvertrag](language-contract.md). Der Nutzer hat inzwischen auch den Identitätsvertrag gewählt und die Konsolidierung beauftragt. Frühere Vorschlags- und Statusangaben auf dieser Seite dokumentieren ihren jeweiligen Gesprächszeitpunkt; sie setzen die späteren Entscheidungen nicht außer Kraft. Produktunterstützung des neuen Gesamtvertrags wird nicht behauptet.

**Aktuell gewaehlte Richtung:** Der Nutzer entscheidet sich fuer **eine einheitliche Definitionenform mit den fest verstandenen Kinds Kind und Property**. body bleibt der verbindliche Koerper; requirements und roles entfallen vorerst als besondere Sprachmittel. [Abschnitt 14](#14-gewaehlte-grundform-und-kompakte-properties) haelt die Entscheidung und die kompakte Autorenform fest. Deren genaue Syntax und Normalisierungsdetails sind der ausgearbeitete Vorschlag zum Auftrag; eine neue Produktimplementierung ist damit nicht behauptet.

Der [Identitaets- und Referenzvertrag](#15-identitaeten-imports-versionen-und-umbenennungen) unterscheidet stabile Identitaet, optionale Anzeigenamen und die Bindung importierter Versionen. Nach dem Durchgang in Abschnitt 16 hat der Nutzer diesen Vertrag gewaehlt; die verbindlichen Designregeln und die noch offenen Formatdetails sind im konsolidierten Sprachvertrag getrennt festgehalten.

Der [vollstaendige Identitaetsdurchgang](#16-identitaetsdurchgang-mit-import-update-und-feldumbenennung) prueft diesen Vorschlag inzwischen begrenzt mit importiertem Vokabular, drei gebundenen Staenden und sender/requester. Er implementiert keinen Produktloader.

Die [Untersuchung einer gemeinsamen Grundform](#11-gemeinsame-grundform-und-body) ist die Grundlage der inzwischen gewaehlten Richtung. Die bisherigen YAML-Dateien und ihre hashgebundenen Berichte bleiben unveraenderte Nachweise des vorherigen Durchstichs mit definition. Der aktuelle Implementierungsname purpose bleibt davon unberuehrt.

Der anschliessend beauftragte [feste Anfang und die vollstaendige Typaufloesung](#12-fester-anfang-und-typaufloesung-am-exchange) sind als eigene, begrenzte Entwurfsprobe mit body ausgearbeitet. Die Probe verwendet eine gemeinsame Definitionenform und ersetzt den Produktcompiler nicht.

Der [Vergleich beider Core-Varianten](#13-vergleich-und-empfehlung-fuer-markitect) dokumentiert auch die damalige Empfehlung des Assistenten fuer die explizite Variante. Diese Empfehlung ist durch die anschliessende Nutzerwahl zugunsten der einheitlichen Form abgeloest. Fruehere Abschnitte bleiben als Herleitung und Vergleich erhalten.

## 1. Kind war bereits der Ansatzpunkt

Die Erinnerung des Nutzers trifft zu. Zwei vorhandene Grundlagen weisen in diese Richtung:

- Der [strukturelle Core](../../../../src/internal/core/README.md) kompiliert explizit gelieferte Schemas und Definitionen. Ein Schema deklariert frei benannte Kinds und ihre geschlossenen, typisierten Properties. Referenzen binden gegen genaue Ziel-Kinds; daraus entstehen Graphkanten. `kindReference` kann auf einen Kind verweisen, definiert ihn aber nicht und aktiviert keine Ausfuehrungslogik. Der Compiler ist nicht auf die fuenf Kinds des gegenwaertigen Projektvokabulars beschraenkt.
- Im [Proposal zu typisierter Prosa vom 14. Oktober](../../product-model-refinement/canonical-model-proposal.md#23-programmiernahe-arbeitsrichtung-typisierte-prosa) wird der Kind ausdruecklich als Sprachkonstrukt beschrieben, das die Interpretation seines individuellen Definitionskoerpers bestimmt. Bedeutung, Referenzen und erwartete Realisierung sollten bereits dort einmal am Kind erklaert werden.

Der jetzige [Projektworkflow](../../../../src/internal/core/projectmodel/projectmodel.go) verwendet trotzdem einen fest komponierten Vertrag mit Manager, Statement, Artifact, Check und Decision. Freie Schemadaten im Core sind nicht gleichbedeutend mit einem fertigen Projektloader, beliebigen Kontext-/Impactregeln oder Apply fuer benutzerdefinierte Kinds. Der [feste Main-Abgleich](../current-main-comparison.md) bleibt in seiner damaligen Rolle erhalten.

Dieser Entwurf verbindet den vorhandenen strukturellen Ansatz mit der jetzt beauftragten Erweiterbarkeit der typisierten Projektdokumentation. Er beginnt keine neue Sprachimplementierung und holt kein historisches Plugin-/Installationssystem zurueck.

## 2. Die kleine Grundlage

Dieser Abschnitt beschreibt die Ausgangsbasis aus dem vorhandenen Core. Die inzwischen gewaehlte einheitliche Definitionenform mit Kind und Property ist in Abschnitt 14 zusammengefasst.

| Sprachbegriff | Funktion |
| --- | --- |
| **Schema / Vokabular** | Versionierter Vertrag der in diesem Modell verfuegbaren Typen und ihrer Bedeutung. |
| **Kind** | Definiert eine Aussageart und deren erlaubte Properties. Seine Bedeutung erklaert, was eine konkrete Definition deklariert. |
| **Property** | Benannter, typisierter Wert mit Kardinalitaet. Eine Property vom Typ reference definiert eine gerichtete, benannte Relation zu einem Ziel-Kind. |
| **Definition** | Konkrete benannte Sollbeschreibung: Identitaet, Kind, individueller verbindlicher Koerper und strukturierte Werte beziehungsweise Beziehungen. |

Identitaet, Namensraum, Referenzaufloesung und kanonischer Inhaltsvergleich sind gemeinsame Sprachmechanismen. Sie sind keine weiteren fachlichen Projekt-Kinds. Der Kern muss weder System noch Command noch Rule fest einbauen. Auch eine spaeter gewuenschte eigenstaendige Rule-Definition wird durch ein Vokabular deklariert; die grundsaetzliche Verbindlichkeit kanonischer Aussagen benoetigt diesen speziellen Typ nicht.

**Minimaler Vorschlag:** Eigene Relationen werden zunaechst am Quell-Kind als Reference-Properties definiert. Deshalb ist `Component.provides` eine andere Deklaration als `System.provides`, auch wenn beide dasselbe Wort verwenden. Der aufgeloeste Bezug behaelt Quellidentitaet, Propertypfad und Zielidentitaet. Gleiche Namen fuehren nicht automatisch zu global gleicher Semantik.

Ein eigenstaendiger Relation-Kind wird erst benoetigt, wenn Beziehungen selbst eine unabhaengige Identitaet, eigene Aussagen oder Beziehungen benoetigen. Attribute einer Beziehung koennen teilweise schon durch strukturierte Objekte mit Referenz-Properties ausgedrueckt werden. Es wird hier weder ein neues Primitive fuer beliebige Kanten noch eine allgemeine Graphregelmaschine eingefuehrt.

### 2.1 Auftrag an den Core und Rolle der Bibliotheken

Praezisierung des Nutzers im weiteren Gespraech: Der Core muss die fuer Markitect erforderlichen allgemeinen Ausdrucksmittel vollstaendig bereitstellen. Bibliotheken dienen ausschliesslich dazu, fertiges Vokabular zu importieren. Sie schalten keine zusaetzlichen Sprachfaehigkeiten frei. Der Nutzer beauftragt nun die Definition dieses Cores und seiner Bestandteile; die nachfolgende Zusammensetzung bleibt ein Diskussionsvorschlag.

**Massstab:** Ein Projekt kann ohne importierte Vokabularbibliothek sein benoetigtes Vokabular selbst definieren und mit denselben Markitect-Faehigkeiten bearbeiten lassen. Ein lokaler Kind besitzt dieselbe sprachliche Stellung wie ein importierter Kind. Der Core kennt keinen privilegierten Component-, Rule- oder Mandate-Typ.

Allgemeine Ausdrucksfaehigkeit bedeutet dabei nicht, jede fachliche Wahrheit formal entscheiden zu koennen. Strukturierte Deklarationen und verbindliche Prosa tragen gemeinsam die Projektbeschreibung. Nicht formal pruefbare Aussagen bleiben verpflichtend und brauchen geeignete Beurteilung.

### 2.2 Vorschlag: vier Grundbausteine und gemeinsame Sprachmechanismen

Die vier Grundbausteine aus Abschnitt 2 werden so konkretisiert:

| Baustein | Mindestinhalt | Connector-Beispiel |
| --- | --- | --- |
| Schema | Identitaet und genaue Version eines Typvertrags, seine Bedeutung und enthaltenen Kinds. | Ein Schema deklariert das projektspezifische Kommunikationsvokabular. |
| Kind | Identitaet im Schema, allgemeine Bedeutung, erlaubte Properties und geltende Anforderungen. | Exchange beschreibt einen vereinbarten Nachrichtenaustausch und seine Rollen. |
| Property | Name, Bedeutung, Werttyp, Kardinalitaet und gegebenenfalls Referenzziel beziehungsweise Objektstruktur. | Exchange.sender referenziert genau einen Container. |
| Definition | Identitaet, genauer Kind, individuelle Sollbeschreibung und Werte der deklarierten Properties. | list-documents bestimmt API, Connector, Kanaele und Wartefrist. |

Schema ist hier der Sprachvertrag. Bibliothek bezeichnet dessen Bereitstellung beziehungsweise Zusammenstellung zur Wiederverwendung; sie ist keine zweite Art von Typdefinition. Ein Schema kann lokal geschrieben oder aus einer Bibliothek bezogen sein. Importauswahl und Dateiladen muessen einen eindeutigen Eingabesatz herstellen; Transport und Paketinstallation sind keine Bedeutung eines Kinds.

Dazu braucht der Core gemeinsame Mechanismen. Sie sind Bestandteile der Sprache, aber keine weiteren fachlichen Projekt-Kinds:

| Mechanismus | Vorgeschlagener Mindestvertrag | Stand gegenueber dem bisherigen Core |
| --- | --- | --- |
| Verbindlicher Koerper | Schema, Kind, Property und Definition tragen interpretierbare Prosa. Kind- und Propertyvertraege gelten fuer ihre Verwendungen; individuelle Aussagen ergaenzen sie und duerfen Widersprueche nicht still ueberschreiben. | Purpose-Felder vorhanden; vollstaendige Autoren- und Interpretationsregeln weiter zu praezisieren. |
| Identitaet und Referenzen | Eindeutige Namen in einem definierten Namensraum und Schema-/Kindvertrag; Referenzen werden exakt aufgeloest. | Vorhanden. Die genaue Autorenform fuer Koerperlinks bleibt offen. |
| Werte und Kardinalitaeten | Einfache Werte, strukturierte Objekte, Einzelwerte und Listen sowie Referenzen auf Definitionen und Kinds. | Vorhandene acht Werttypen sind der Ausgangspunkt; weitere Typkonstruktoren nur bei konkretem Bedarf. |
| Anforderungen | Verbindliche Anforderungen stehen in den Koerpern. Ihr Gegenstand und Geltungsbereich werden dort erklaert. | Strukturpruefung vorhanden; requirements und roles sind vorerst zurueckgestellt. Zusaetzliche formale Constraintoperatoren bleiben eine eigene offene Erweiterungsfrage. |
| Gebundener Modellstand | Alle verwendeten Typvertraege und Definitionen werden zusammen festgehalten; auch eine Bedeutungsabweichung bei unveraenderten Instanzen ist eine Aenderung. | Deterministische Normalisierung, Graph, Digest und Herkunft vorhanden; Bibliotheksauswahl und Produktanschluss bleiben zu entwerfen. |

Eine Beziehung benoetigt damit vorerst kein fuenftes universelles Deklarationsformat: Ihre Rolle wird als Reference-Property deklariert, der konkrete Bezug steht an einer Definition. Wenn eine Beziehung selbst Identitaet, mehrere Beteiligte oder eigene Aussagen braucht, kann ein eigener Kind sie beschreiben. Exchange ist bereits ein solches Beispiel mit mehreren referenzierten Rollen. Ob das fuer alle relevanten Faelle ausreicht, bleibt eine ausdrueckliche Gegenprobe.

### 2.3 Was daraus fuer Markitect folgt

Die Sprache muss den vollstaendigen geltenden Vertrag einer Definition zugaenglich machen: ihre eigene Beschreibung, ihren Kind, dessen Properties und Anforderungen sowie aufgeloeste Bezuege. Das Auslassen einer importierten oder lokal definierten Kind-Bedeutung darf nicht zur stillen Entpflichtung fuehren.

Darauf bauen generische Kontextbildung, Aenderungsbewertung, Realisierung und unabhaengige Pruefung auf. Deren Verfahren gehoeren weiterhin zum Markitect-Produkt, auch wenn ihre Ausfuehrungsdaten nicht Teil der kanonischen Projektbeschreibung sind. Der bestehende Strukturcompiler allein erfuellt diesen Produktauftrag noch nicht. Die neue Richtung hebt die bestehenden Pflichten zur Delegation, unabhaengigen Pruefung oder Repositoryzuordnung nicht nebenbei auf; sie legt deren neue sprachliche Anbindung noch nicht fest.

Insbesondere ist **eine aufgeloeste Beziehung noch keine berechenbare Wirkungsvorschrift**. Der Core kann ihre Endpunkte, Deklaration und Bedeutung bereitstellen. Welche Kontexte und Pruefungen sie betrifft, muss entweder durch einen ausdruecklichen generischen Sprachmechanismus beschrieben oder als begruendete semantische Bewertung behandelt werden. Der Relationsname allein darf weder Baumsemantik noch Regelgeltung oder Schreibbefugnis ausloesen.

Die naechsten Gegenproben betreffen deshalb: individuelle und allgemeine Anforderungen samt Geltungsbereich; Beziehungen mit eigener Beschreibung; explizite Bezuege im Koerper; sowie die Bedeutung einer Aenderung am Kind fuer dessen Definitionen. Vererbung, Zieltyp-Vereinigungen, ein eigenstaendiges Relation-Primitive und eine allgemeine formale Regelsprache sind weiterhin offene Erweiterungskandidaten. Diese Bestandsaufnahme entscheidet sie nicht vorab.

## 3. Bedeutung gehoert zum Sprachvertrag

Der Kind-Koerper definiert seine allgemeine Bedeutung; der Definitionskoerper enthaelt die individuellen Sollaussagen. Die Beschreibung von `Exchange` erklaert beispielsweise einmal die Rollen sender und receiver und die Antwortzuordnung. Die konkrete Definition `list-documents` legt ihre Beteiligten, Kanaele, Operation und Besonderheiten fest.

Der Compiler prueft, ob ein sender eine existierende Definition des erlaubten Kinds Container referenziert. Er beweist dadurch nicht, dass der betreffende Prozess Nachrichten richtig versendet. Das gehoert zur Interpretation, Realisierung und semantischen Pruefung des beschriebenen Projekts. Die Sprachbedeutung ist keine Agentenrolle und kein Mandate.

Entsprechend bedeutet die Property `Channel.topicPattern: string` zunaechst nur, dass ein String vorhanden ist. Dass `{tenantId}` aus einem autorisierten Kontext einzusetzen ist und keine MQTT-Wildcards enthalten darf, ist eine erklaerte Verpflichtung des Projektvokabulars. Eine formale Vorlagenpruefung waere eine gesondert definierte Validierung. Ein neuer Typname installiert keinen Parser, Brokeradapter oder Codegenerator.

Alle zur Interpretation benoetigten Schema-, Kind- und Propertykoerper gehoeren zum gebundenen Modellstand. Aendert sich die Bedeutung eines Kinds bei unveraenderten Instanzdateien, ist das weiterhin eine semantisch relevante Modellaenderung. Der existierende Core beruecksichtigt Schema-/Kind-/Property-Purpose bereits im Modelldigest; die neue Autorenform muss diesen Inhalt erhalten.

### 3.1 Zwei Pruefgegenstaende und die Qualitaet des Vertrags

Diskussionsvorschlag vom 15. Oktober auf die Frage des Nutzers, wie Bedeutung und Regeln eigener Kinds beschrieben und geprueft werden sollen. Die nachfolgenden Regeln und Felder sind Entwurf, keine zusaetzlich implementierten Core-Funktionen.

Zuerst ist der **Pruefgegenstand** zu bestimmen. Eine Regel kann die gueltige Modellbeschreibung betreffen oder eine Pflicht der daraus entstehenden Software ausdruecken. Daneben braucht der Kind-Vertrag selbst eine fachliche Beurteilung seiner Verstaendlichkeit, Widerspruchsfreiheit und Eignung.

| Gegenstand | Leitfrage | Geeignete Pruefung |
| --- | --- | --- |
| Modellkonformitaet | Erfuellt eine konkrete Definition den deklarierten Kind-Vertrag? | Compiler fuer ausdruecklich formal beschriebene Bedingungen; Agent fuer fachliche Prosa, Angemessenheit und Vollstaendigkeit. |
| Vertragsqualitaet | Sind Kind, Regeln und Relationen sinnvoll, eindeutig und miteinander vereinbar? | Fachliches Modellreview; formale Gegenproben und Werkzeuge koennen Teilbefunde beitragen. |
| Realisierungskonformitaet | Erfuellt der feste Umsetzungskandidat die geltenden Pflichten? | Deterministische Tests und Analysen, soweit geeignet, plus unabhaengige Agentenbewertung ihrer Aussagekraft und des Kandidaten. |

Die Wahl der Pruefmethode folgt danach. Auch eine Realisierungspflicht kann maschinell getestet werden; diese Ausfuehrung gehoert jedoch nicht zum reinen Strukturcompiler. Umgekehrt benoetigt die Beurteilung einer Modelldefinition teilweise einen Agenten, wenn ihre Bedeutung in Prosa vorliegt. Die Einteilung in Modell und Realisierung ist deshalb keine Einteilung in automatisch und menschlich beziehungsweise agentisch.

### 3.2 Inhalt eines selbst definierten Kind-Vertrags

Ich wuerde am Kind vier Inhalte beschreiben: Bedeutung und Gegenstand seiner Definitionen; strukturierte Properties und Relationsrollen; Bedingungen fuer gueltige Modelldefinitionen; Pflichten, welche die Realisierung erfuellen muss. Fuer die letzten beiden Gruppen werden unterscheidbare Regelbezeichnungen vorgeschlagen, wenn sie unabhaengig betroffen oder beurteilt werden sollen.

Eine solche Regel erklaert ihren Gegenstand, ihre geltende Aussage und gegebenenfalls die Bedingung ihrer Anwendbarkeit. Eine formal beschriebene Bedingung kann ein zusaetzlicher, exakt pruefbarer Bestandteil dieser einen Regel sein. Die benannten Regeln werden einmal am Kind gepflegt und gelten fuer seine Definitionen gemaess ihrem erklaerten Geltungsbereich. Eine konkrete Definition ergaenzt ihre individuellen Pflichten; allgemeine Regeln werden dort nicht erneut kopiert. Der Kind-Koerper soll die Regelbedeutung nicht nochmals als unabhaengige zweite Wahrheit duplizieren.

Ein eingebauter Projekt-Kind namens Rule ist dafuer nicht erforderlich. Ein optionaler Regelabschnitt waere Teil des Meta-Vertrags fuer Kinds. Ob er eigene strukturierte Felder bekommt oder vorerst als benannte Absaetze im Koerper ausgedrueckt wird, ist eine Autorenformatentscheidung. Vorhandene verbindliche Prosa bleibt verpflichtend, auch wenn sie keinen formalen Constraint besitzt.

### 3.3 Beispiel am Kind Exchange

| Aussage des Kind-Vertrags | Gegenstand | Was daraus folgt |
| --- | --- | --- |
| sender referenziert genau einen Container. | Modell | Bereits durch Reference-Ziel und Kardinalitaet im Core pruefbar. |
| timeoutSeconds und maxAttempts sind positive Ganzzahlen. | Modell | Ganzzahltyp ist vorhanden; die positive Wertebereichsbedingung benoetigt eine ausdrueckliche neue Constraintfunktion oder ein Modellreview. minCount beschraenkt die Zahl der Werte, nicht ihren Zahlenwert. |
| Sender und Empfaenger sind fuer die Operation passend gewaehlt. | Modell / Vertragsqualitaet | Der Compiler prueft die Zieltypen. Die fachliche Rollenverteilung benoetigt Review; eine spaetere vollstaendige formale Abbildung koennte Teilbedingungen pruefen. |
| Eine Antwort erhaelt die MessageId des Requests. | Realisierung | Agent bewertet Umsetzung und geeignete Zuordnungspruefungen an einem festen Kandidaten; der Compiler kann dies nicht aus einem Referenzgraphen beweisen. |
| Der Mandant stammt aus einem autorisierten Kontext. | Realisierung | Zugriffspfade, Eingabeherkunft und geeignete negative Tests untersuchen; ein vorhandenes tenantId-Feld reicht nicht als Nachweis. |
| Jede Empfaengerreplika kann die an ihren Pool verteilte Operation bearbeiten. | Modell und Realisierung mit getrennten Erwartungen | Deklarierte Faehigkeiten koennten im Modell verglichen werden. Tatsaechliche Registrierung, Konfiguration und Betrieb brauchen eigene Nachweise. Die Deklaration beweist nicht ihre Umsetzung. |

#### 3.3.1 Bedeutung und Properties des Exchange-Vertrags

Konkretisierung auf Auftrag des Nutzers: Die folgenden Felder sind ein zusammenhaengender Vertragsentwurf, keine bereits beschlossene oder implementierte Syntax. Die vorhandenen Beispieldateien und ihr begrenzter Strukturpruefbericht bleiben unveraendert.

Die anschliessende [Pruefung des Zusatznutzens](#36-brauchen-requirements-und-roles-eigene-core-semantik) stellt requirements und roles als eigene Sprachmittel infrage. Der Nutzer entscheidet danach ausdruecklich, sie vorerst wegzulassen. Die nachfolgende strukturierte Variante ist damit zurueckgestellt und bleibt nur zum Vergleich erhalten.

Die allgemeine Bedeutung lautet: Eine Exchange-Definition beschreibt den gewuenschten Request-/Reply-Vertrag fuer eine Operation. Sie beschreibt weder einen einzelnen laufenden Austausch noch eine Nachricht. sender stellt die Operation aus; receiver bearbeitet sie. requestChannel und replyChannel beschreiben die Kommunikationswege. Der Kind bindet diese Rollen aneinander; seine Anforderungen praezisieren das erwartete Verhalten.

| Property | Typ / Anzahl | Bedeutung |
| --- | --- | --- |
| sender | Container / genau 1 | Anwendung in der Auftraggeberrolle dieses Austauschs. |
| receiver | Container / genau 1 | Anwendung in der Bearbeiterrolle dieses Austauschs. |
| command | Command / genau 1 | Vertrag der angeforderten Operation. |
| requestChannel | Channel / genau 1 | Kanal fuer die Auftragsrichtung. |
| replyChannel | Channel / genau 1 | Kanal fuer Antworten an die ausstellende Instanz. |
| timeoutSeconds | integer / genau 1 | Wartefrist pro Versuch in Sekunden. |
| maxAttempts | integer / genau 1 | Maximale Versuchszahl einschliesslich des ersten Versuchs. |

Diese Rollen sind Teil dieses Kinds. Der Core kennt weder sender noch die MQTT-Bedeutung von Channel vorab. Jede Property erhaelt ihre eigene Bedeutung und ihren Strukturvertrag. Dass eine Rolle auf Container zeigt, beschreibt keine Agentenzustaendigkeit und erteilt keine Schreibbefugnis.

#### 3.3.2 Geltungsbereich einer Anforderung

Der vorgeschlagene Meta-Abschnitt heisst hier `requirements`; er konkretisiert die fruehere illustrative `rules`-Skizze. Ein Requirement ist eine Unterstruktur des Kind- oder Definitionsvertrags, kein zusaetzlicher fest eingebauter fachlicher Rule-Kind.

| Angabe | Vertrag |
| --- | --- |
| Deklarationsort | Am Kind gilt die Anforderung fuer jede Definition genau dieses Kinds. An einer Definition ergaenzt sie nur deren individuelle Pflichten. Es wird keine Typvererbung vorausgesetzt. |
| id | Stabile lokale Bezeichnung innerhalb des deklarierenden Vertrags. Ihre qualifizierte Adresse schliesst Schema-/Kind- beziehungsweise Definitionsidentitaet ein. Ein gleicher lokaler Name an einer Definition ueberschreibt keine Kind-Anforderung. |
| subject | `model` bezeichnet eine Anforderung an die Modellbeschreibung; `realization` bezeichnet eine Pflicht ihrer Umsetzung. Beide sind kanonische Sollaussagen. Die Auswahl legt nicht die Pruefmethode fest. |
| roles | Optional: explizite Reference-Properties des deklarierenden Kinds, deren Beteiligte fuer diese Anforderung relevant sind. Jede Rolle wird relativ zur jeweils betrachteten Definition gebunden. Ohne roles bleibt der Gegenstand die Definition beziehungsweise ihre gesamte Realisierung im beschriebenen Umfang. |
| definition | Verbindliche Aussage einschliesslich fachlicher Bedingungen ihrer Anwendbarkeit. Eine nicht formal ausdrueckbare Bedingung wird semantisch beurteilt und bei Unsicherheit als offen behandelt. |
| constraint | Optional: formaler Anteil derselben Anforderung. Nur bekannte Operatoren und gueltige Bindungen sind zulaessig; die Skizze unten verwendet Property und numerische Untergrenze. |

`roles` bezeichnet die betroffenen Beteiligten; die konkrete Pflichtenteilung steht in der Aussage. Es bedeutet weder, dass jeder Beteiligte jede Teilpflicht uebernimmt, noch bestimmt es die Agentenaufteilung. Im ersten Vorschlag sind ausschliesslich direkte Reference-Properties als Rollen erlaubt, keine frei formulierten Selektorausdruecke oder transitiven Graphpfade. Ein unbekannter Rollenname oder eine Wertproperty an dieser Stelle ist ein Strukturfehler dieses vorgeschlagenen Vertrags.

Insbesondere bleibt der Kontext **die einzelne Exchange-Definition**: Eine Anforderung fuer deren receiver betrifft den referenzierten Container in dieser Rolle und in diesem Austausch. Sie veraendert nicht dessen allgemeine Definition, gilt nicht automatisch fuer andere Exchanges und wandert nicht pauschal zu dessen Components. Bei der Realisierungspruefung muessen dennoch die relevanten implementierenden Teile betrachtet werden. Fachlicher Geltungsbereich und benoetigter Pruefkontext sind verschieden.

#### 3.3.3 Vertragsausschnitt mit Anforderungen

Der Ausschnitt steht innerhalb der Kind-Deklaration von Exchange. Die oben vollstaendig aufgefuehrten Propertyvertraege stehen daneben unter properties; sie werden hier nicht wiederholt.

```yaml
requirements:
  - id: positive-timeout
    subject: model
    definition: Die Wartefrist pro Versuch muss mindestens eine Sekunde betragen.
    constraint:
      property: timeoutSeconds
      minimum: 1

  - id: positive-attempts
    subject: model
    definition: Das Versuchsbudget muss mindestens den ersten Versuch erlauben.
    constraint:
      property: maxAttempts
      minimum: 1

  - id: register-before-publish
    subject: realization
    roles: [sender]
    definition: >-
      Der Sender muss die Zuordnung eines wartenden Austauschs vor der
      Publikation des Requests einrichten, sodass eine sofortige Antwort
      dem wartenden Austausch zugeordnet werden kann.

  - id: preserve-message-id
    subject: realization
    roles: [sender, receiver]
    definition: >-
      Der Empfaenger muss eine Antwort mit der MessageId des Requests senden.
      Der Sender muss die Antwort innerhalb seiner Instanz anhand dieser
      MessageId dem wartenden Austausch zuordnen. Die CorrelationId dient
      der uebergreifenden Nachverfolgung und ersetzt diese Zuordnung nicht.

  - id: reply-destination
    subject: realization
    roles: [sender, receiver]
    definition: >-
      Der Sender muss dem Request eine fuer die ausstellende Instanz
      autorisierte Antwortadresse gemaess replyChannel mitgeben. Der
      Empfaenger muss die Antwort an diese Adresse senden. Die Zustellung
      darf nicht an eine beliebige andere Senderinstanz verteilt werden.

  - id: bounded-retries
    subject: realization
    roles: [sender]
    definition: >-
      Der Sender darf hoechstens maxAttempts Versuche ausloesen und muss
      pro Versuch nach timeoutSeconds ohne Antwort die Wartefrist beenden.
      Weitere Versuche verwenden dieselbe MessageId. Eine abgelaufene
      Wartefrist beweist nicht, dass die Operation nicht ausgefuehrt wurde.
```

Diese Skizze ist nicht Teil der gegen den heutigen Core geprueften YAML-Dateien. Sie legt keine allgemeine Ausdruckssprache fest. Ein vorhandener allgemeiner Integer-Typ kann daraus nicht selbststaendig einen minimum-Operator lernen. Der Compiler muss den erlaubten Operator und seine Wert-/Typsemantik kennen und validieren. Unbekannte oder fehlerhaft gebundene formale Constraints sind ein Befund, keine still ignorierte Prosa. Ebenso ist ein Widerspruch zwischen formaler Bedingung und Regeltext ein zu klaerender Modellfehler; keine Darstellung gewinnt stillschweigend.

Die gemeinsamen Pflichten werden in einer fertigen Autorenform einmal unter requirements gepflegt. Der vorhandene breite Exchange-Koerper in vocabulary.yaml beschreibt sie zum Teil noch in einem Absatz; dieser nachweisgebundene erste Durchstich wird hier absichtlich nicht umgeschrieben. Bei einer spaeteren Umstellung bleiben im allgemeinen Koerper die Bedeutung und die Rollenbeschreibung, waehrend die benannten Anforderungen ihre jeweilige Aussage tragen. Diese zwei Entwicklungsstaende sind keine Aufforderung, zwei unabhaengige Wahrheiten manuell synchron zu halten.

#### 3.3.4 Bindung an list-documents und Pruefung

Fuer die vorhandene Definition list-documents werden sender an api, receiver an onprem, command an list-documents-command, requestChannel an requests und replyChannel an replies gebunden. Die Wartefrist betraegt 30 Sekunden pro Versuch, das Budget hoechstens zwei Versuche. Daraus folgt ohne zusaetzliche Festlegung weder eine feste Gesamtfrist von 60 Sekunden noch eine Garantie einmaliger fachlicher Ausfuehrung.

Im vorgeschlagenen Format kann diese eine Definition zusaetzlich erklaeren:

```yaml
requirements:
  - id: late-replies
    subject: realization
    roles: [sender]
    definition: >-
      Antworten ohne passenden offenen Wartezustand werden verworfen und
      duerfen keinen anderen wartenden Austausch abschliessen.
```

Diese Anforderung ergaenzt nur list-documents. Sie wird fuer dieses Beispiel aus dem bereits vorhandenen Definitionskoerper herausgezogen, nicht als neue allgemeine Exchange-Pflicht eingefuehrt. Kind-Anforderungen und konkrete Anforderungen gelten gemeinsam. Auch die geltenden Vertraege des Commands und der Channels bleiben zu beruecksichtigen. Weder die Propertywerte noch die individuelle Prosa koennen die allgemeine Pflicht zur Antwortzuordnung still abschalten. Ein Widerspruch wird als zu klaerender Befund behandelt; ein kuenftiger ausdruecklicher Ausnahmemechanismus ist hier nicht entschieden.

Der heutige Compiler kann die Propertytypen, Kardinalitaeten und Referenzziele pruefen. Ein um diesen Vertrag erweiterter Compiler koennte zusaetzlich Rollenbindung, Requirementadressen und die beiden minimum-Constraints pruefen. Er kann aus diesen Deklarationen nicht feststellen, ob die laufende Implementierung die Wartezuordnung rechtzeitig registriert oder Antworten korrekt verarbeitet. Das braucht Implementierungs- und Integrationsnachweise sowie unabhaengige Bewertung gegen den festen Modell- und Kandidatenstand. MessageId ist hier eine im Protokoll beschriebene Laufzeitgroesse, keine bei jeder Exchange-Definition auszufuellende Property.

### 3.4 Arbeitsteilung und Evidenz

Der Compiler kann exakt formulierte Bedingungen ueber den vollstaendig bereitgestellten Modellstand pruefen. Mit geeigneten ausdruecklichen Constraintfunktionen waeren spaeter auch Feldvergleiche, deklarierte Eindeutigkeit oder Zyklenfreiheit einer bestimmten Relation moeglich. Welche Relationen das verlangen, bestimmt der erklaerte Vokabularvertrag. Er darf weder `contains` allein anhand seines Namens mit Baumsemantik versehen noch aus einer fachlichen Verwendung eine Schreib- oder Delegationsbefugnis ableiten.

Ein Agent beurteilt die Verstaendlichkeit und ausreichende Festlegung der Prosa, fachliche Widersprueche, fehlende Beziehungen und die Eignung der gewaehlten Regeln. Ein unabhaengiger Verifier beurteilt den festen Realisierungskandidaten gegen diese Pflichten und benoetigt dafuer Kind-/Propertybedeutung, konkrete Definitionen, relevante Nachbarvertraege, Befunde und geeignete Ausfuehrungsnachweise. Implementierung und erfolgreiche eigene Tests sind fuer sich keine unabhaengige semantische Abnahme. Wenn der Vertrag unklar ist, wird ein Befund beziehungsweise eine vorgeschlagene Klaerung gemeldet; die Erfolgsbedingung wird nicht passend zur Umsetzung umgedeutet.

Ergebnisse werden auf ihre Aussage begrenzt: strukturell geprueft, durch ein bestimmtes Verfahren am festen Kandidaten belegt, verletzt oder noch offen. Nicht untersuchte oder unentscheidbare erforderliche Pflichten bleiben offen und werden nicht als erfuellt zusammengefasst. Die Eignung des Tests fuer die Verpflichtung muss ebenfalls beurteilt werden. Ergebnis- und Ausfuehrungsdaten gehoeren zum Nachweis; sie sind keine neuen kanonischen Sollregeln.

### 3.5 Vorschlag fuer den ersten Sprachumfang

Zunaechst die vorhandene Strukturpruefung verwenden, alle Bedeutungs- und Realisierungspflichten verbindlich beschreiben und Modellreview sowie unabhaengige Realisierungspruefung als getrennte Nachweise behandeln. Zusaetzliche formale Constraints werden nur fuer konkrete, wiederkehrende Beschreibungsbedarfe eingefuehrt; positive Zahlenbereiche sind hier eine naheliegende erste Gegenprobe. Der Ansatz benoetigt zum Start weder beliebigen ausfuehrbaren Code im Modell noch einen allgemeinen Regelinterpreter.

Wird ein Kind-Vertrag geaendert, sind seine Definitionen und die daraus entstandenen betroffenen Realisierungen erneut zu betrachten. Ein unveraendertes Instanzdokument belegt keine unveraenderte Pflicht. Der Digest erfasst die Aenderung bereits strukturell; die generische Kontext-/Impact- und Pruefsteuerung dafuer bleibt ein noch zu entwerfender Produktanschluss.

### 3.6 Brauchen requirements und roles eigene Core-Semantik?

Weitere Nutzerfrage: Lassen sich Bedeutung, Anforderungen und ihr Geltungsbereich bereits durch verbindliche Prosa und Properties ausdruecken? Diese Gegenprobe prueft den Nutzen der vorgeschlagenen neuen Sprachmittel. Sie ist eine Empfehlung, keine bereits vom Nutzer beschlossene Streichung.

**Nachfolgende Nutzerentscheidung:** requirements und roles werden vorerst weggelassen. Die hier begruendete schlanke Richtung ist fuer den naechsten Durchstich gewaehlt; daraus folgt noch keine Annahme einer bestimmten Dateisyntax oder Implementierung.

**Ergebnis am Exchange:** Die bisher beschriebenen Pflichten brauchen weder einen privilegierten requirements-Abschnitt noch ein rollenbindendes roles-Feld, um ausgedrueckt und gegen die Umsetzung beurteilt werden zu koennen. Die verbindliche Beschreibung des Kinds gilt fuer jede seiner Definitionen. Die Propertyvertraege benennen die Beteiligten; die konkrete Definition bindet sie ueber typisierte Referenzen. Eine Aussage kann ihren Geltungsbereich ausdruecklich erklaeren.

Ein Ausschnitt des Kind-Koerpers kann lauten:

> Jede Definition dieses Kinds beschreibt einen Request-/Reply-Vertrag fuer ihre angegebene Operation. Die folgenden Pflichten gelten fuer die unter sender und receiver referenzierten Anwendungen bei diesem Austausch.
>
> Der sender muss die Wartezuordnung vor der Publikation des Requests einrichten. Der receiver muss die MessageId des Requests in der Antwort erhalten. Der sender muss die Antwort anhand dieser MessageId dem wartenden Austausch zuordnen; die CorrelationId ersetzt diese Zuordnung nicht.
>
> Pro Versuch gilt timeoutSeconds. Insgesamt sind hoechstens maxAttempts Versuche zulaessig, einschliesslich des ersten. Weitere Versuche verwenden dieselbe MessageId. Ein Timeout beweist nicht, dass die Operation nicht ausgefuehrt wurde.

Die Properties sender und receiver bleiben Reference-Properties auf Container mit genau einem Wert. timeoutSeconds und maxAttempts bleiben Integer-Properties, deren Bedeutung und Werteanforderungen an den Properties beschrieben werden. Die gemeinsam geltende Antwortzuordnung steht einmal im Kind-Koerper. Ein eigener Abschnitt muss diese Aussagen nicht parallel wiederholen.

| Gewuenschter Nutzen | Prosa und Properties | Was requirements beziehungsweise roles zusaetzlich liefern wuerden |
| --- | --- | --- |
| Verbindliche Pflichten | Im Sprachvertrag bereits fuer saemtliche kanonische Prosa festgelegt. | Keine zusaetzliche Verbindlichkeit. |
| Allgemeine gegenueber individuellen Pflichten | Kind-Koerper gilt fuer dessen Definitionen; individuelle Beschreibung ergaenzt die einzelne Definition. | Maschinenlesbare Untergliederung, ohne allein neue fachliche Bedeutung zu liefern. |
| Geltungsbereich bei Sender und Empfaenger | Referenz-Properties binden die Beteiligten. Der Text beschreibt ihre Pflichten im Kontext dieser Exchange-Definition. | Ein roles-Feld kann Beteiligtenzuordnung und erlaubte Rollennamen unabhaengig vom Text strukturell pruefbar machen. Es beweist weder die inhaltliche Zuordnung des Textes noch die Vollstaendigkeit der Rollenliste. |
| Pruefung des Laufzeitverhaltens | Umsetzung, Tests und unabhaengiges Review gegen den vollstaendigen Vertrag. | Ein Requirementdatensatz mit Prosa macht das Verhalten nicht automatisch entscheidbar. |
| Exakte Adressierung einer einzelnen Pflicht | Zunaechst Verweis auf Kind-/Property-/Definitionsidentitaet und gebundenen Koerper mit Textstelle. | Eine eigene stabile Requirementidentitaet erleichtert pflichtgenaue Befunde, Referenzen und Nachweiszuordnung. Ein allgemeiner Koerperanker waere eine andere, gesondert zu definierende Loesung. |
| Gezielte Kontexte und Aenderungsfolgen | Vollstaendiger Kind-Vertrag und relevante Property-/Definitionsvertraege als sichere, gegebenenfalls breite Grundlage. Semantische Zusammenhaenge brauchen Beurteilung. | Explizite Zuordnung kann deklarierte Beteiligte pro Pflicht berechenbar machen. Die konkrete Verarbeitungssemantik und der Umgang mit fehlenden Deklarationen muessten zusaetzlich festgelegt werden. |
| Formale Wertebedingungen | Bekannte Constraints koennen grundsaetzlich direkt am Propertyvertrag ansetzen. | Ein universeller Requirementabschnitt ist fuer einen minimum-Operator nicht notwendig. Propertyuebergreifende oder graphweite Praedikate brauchen gegebenenfalls einen anderen Anknuepfungspunkt. |

Die letzte Zeile ist kein Nachweis bereits implementierter Wertebereiche. Der jetzige Core kennt den Operator minimum nicht. **Formal pruefbare Bedingungen und strukturierte Anforderungskataloge sind verschiedene Designentscheidungen.** Wir koennen Erstere bei konkretem Bedarf ergaenzen, ohne dafuer saemtliche Prosa in Requirementobjekte umzuformen.

Der Preis der schlankeren Form ist ausdruecklich: Der Compiler kann aus einem Prosaabsatz nicht verlaesslich einzelne Pflichten, Rollenbindung, Vollstaendigkeit oder fachliche Anwendungsbedingungen berechnen. Ohne Requirements ist ein Nachweis pro einzeln identifizierter Pflicht nicht allein durch das Autorenformat gewaehrleistet. Agenten koennen fuer einen festen Modellstand eine Pruefliste mit Quellverweisen ableiten; sie ist ein pruefbares Arbeitsergebnis mit Vollstaendigkeitsunsicherheit und kein zweiter kanonischer Vertrag. Bei Zweifeln wird der vollstaendige geltende Vertrag beruecksichtigt und fehlende Pruefung bleibt offen. Ein engerer Kontext allein aus einer ungesicherten Extraktion waere kein zuverlaessiger Ersatz.

Ebenso sind eine in Prosa genannte Property und ein formal gebundener Koerperlink zu unterscheiden. Ein nacktes Wort sender erzeugt keine vom Compiler gepruefte Referenz zur Propertydeklaration. Ein spaeterer allgemeiner Symbol-/Linkmechanismus koennte solche Bindungen pruefen, ohne ein besonderes roles-Feld zu verlangen. Sein Vertrag ist weiter offen.

**Empfehlung fuer den ersten Core-Vertrag:** Verbindliche Koerper an Schema, Kind, Property und Definition; typisierte Properties und Referenzen; Kind-Pflichten gelten fuer jede eigene Definition; individuelle Aussagen ergaenzen und heben allgemeine Pflichten nicht still auf. requirements, subject und roles erhalten vorerst keine besondere Core-Verarbeitung. Ihre bisher diskutierten Bedeutungen lassen sich fachlich im Koerper ausdruecken. Die strukturierte Variante aus Abschnitt 3.3 bleibt Kandidat, falls wir stabile Einzelpflichten und berechenbare Zuordnungen als notwendige Produktfaehigkeit belegen.

Zusaetzliche Struktur ist weiterhin durch selbst definierte Kinds moeglich, etwa ein projektdefinierter Requirement-Kind mit Beziehungen und Prosa. Dessen Inhalt erhaelt dadurch nicht automatisch die Bedeutung eines speziellen Compileroperators oder eine neue Kontextsteuerung. Eine gewoehnliche Property an Exchange betrifft zudem ihre konkreten Definitionen; sie ist kein Ersatz fuer einen universellen Meta-Abschnitt am Kind. Das Ziel ist, einen solchen Spezialmechanismus erst bei belegtem Bedarf festzulegen.

## 4. Konkrete Autorenform des Kandidaten

Die zwei Beispieldateien zeigen die Ebenen getrennt:

- [vocabulary.yaml](vocabulary.yaml) deklariert die acht projektspezifischen Kinds und ihre Properties, einschliesslich ihrer eigenen Relationen.
- [connector.yaml](connector.yaml) enthaelt dreizehn konkrete Definitionen, die diese Typen verwenden. Es sind Projektbeschreibungen; einzelne Commands, Mandanten und Prozessinstanzen sind Laufzeitdaten.

Im Vokabular steht beispielsweise:

```yaml
apiVersion: connector.example.org/v1alpha1
definition: Vokabular des Connector-Projekts.
kinds:
  Exchange:
    definition: Beschreibt einen Request-/Reply-Vertrag zwischen zwei Anwendungen.
    properties:
      sender:
        definition: Die Anwendung in der ausstellenden Rolle.
        type: reference
        target: {apiVersion: connector.example.org/v1alpha1, kind: Container}
        minCount: 1
        maxCount: 1
```

Dieser Ausschnitt zeigt nur eine Relation. Das vollstaendige Vokabular enthaelt den Ziel-Kind und die weiteren Felder. Ein Definitionsausschnitt verwendet den Vertrag so:

```yaml
apiVersion: connector.example.org/v1alpha1
kind: Exchange
metadata: {namespace: connector, name: list-documents}
definition: Die API fordert die Dokumentmetadaten der autorisierten Kundenumgebung an.
spec:
  sender: {namespace: connector, name: api}
```

Die zwei Ausschnitte sind keine eigenstaendig vollstaendigen Modelle. Die beiden verlinkten Dateien bilden zusammen den ausgearbeiteten Durchstich. Die Kurzreferenz erhaelt ihren Ziel-Kind und ihre genaue API-Version aus dem Propertyvertrag. Keine Aufloesung auf die zufaellig neueste Version und keine Auswahl des ersten gleichnamigen Symbols.

`definition` bezeichnet in den bisherigen Beispieldateien den Koerper fuer Schema, Kind, Property und konkrete Definition. Im weiteren Gespraech waehlt der Nutzer dafuer body; die alten Beispieldateien bleiben als gebundener Pruefstand erhalten. Ein Schema ist in diesem ersten Durchstich ein eigenes Dokumentformat mit fest bekannter Metastruktur. Ob Kinds und Properties selbst Definitionen mit gemeinsamer Grundform werden sollen, wird nun in Abschnitt 11 untersucht. Der erste Durchstich beweist diese neue Lade- und Bootstraprichtung nicht.

Die Beispiele verwenden die vorhandene Core-Kardinalitaet: `maxCount: 1` bedeutet Einzelwert, andere Maximalwerte eine Liste; `-1` bedeutet strukturell unbegrenzt innerhalb der allgemeinen Compilergrenzen. Der erste Durchstich erlaubt pro Referenzfeld genau einen Ziel-Kind. Eine Relation fuer mehrere Ziel-Kinds ist noch zu entwerfen; die fruehere Referenzunion-Frage wird nicht als geloest ausgegeben.

## 5. Eigene Typen und ihre Relationen im Connector-Beispiel

| Projektdefinierter Kind | Was er in diesem Vokabular beschreibt | Eigene Relationsfelder |
| --- | --- | --- |
| System | Zusammengehoerige Loesung mit erklaerter Grenze. | containers -> Container; exchanges -> Exchange; provides -> Interface |
| Container | Laufzeitanwendung und ihr vorgesehener Ort. | components -> Component |
| Component | Funktionale Verantwortung innerhalb einer Anwendung. | provides/consumes -> Interface; emits/handles -> Command |
| Interface | Vertrag eines externen oder internen Zugangs. | In diesem kleinen Ausschnitt keine strukturierten Relationsfelder. |
| MqttService | Benoetigte MQTT-Laufzeitinfrastruktur. | In diesem kleinen Ausschnitt keine strukturierten Relationsfelder. |
| Channel | Parametrisierte Adressfamilie und Zustellabsicht. | broker -> MqttService |
| Command | Versionierter Operationsvertrag mit Eingaben und Resultat. | In diesem kleinen Ausschnitt keine strukturierten Relationsfelder. |
| Exchange | Verbindet Teilnehmer, Operation, Requestkanal und Replykanal. | sender/receiver -> Container; command -> Command; requestChannel/replyChannel -> Channel |

Das sind acht Typen **dieses Projekts**, keine acht neuen Markitect-Primitiven. C4-orientierte Architekturbegriffe und spezielle Kommunikationsbegriffe werden durch dieselben Sprachmechanismen deklariert. Die Wahl von MqttService loest die zuvor sichtbare Brokerfrage ohne eine neue universelle C4-Abstraktion: Ein Projekt darf den relevanten Infrastrukturgegenstand ausdruecklich typisieren. Wie eine C4- oder Bereitstellungssicht daraus abgeleitet wird, ist ein weiterer Vertrag.

Der gerichtete Zusammenhang lautet auszugsweise:

```text
platform:System --containers--> api:Container
api:Container --components--> http-edge:Component
http-edge:Component --emits--> list-documents-command:Command
platform:System --exchanges--> list-documents:Exchange
list-documents:Exchange --sender--> api:Container
list-documents:Exchange --receiver--> onprem:Container
list-documents:Exchange --command--> list-documents-command:Command
list-documents:Exchange --requestChannel--> requests:Channel
list-documents:Exchange --replyChannel--> replies:Channel
requests:Channel --broker--> messaging:MqttService
replies:Channel --broker--> messaging:MqttService
onprem:Container --components--> dms-handler:Component
dms-handler:Component --handles--> list-documents-command:Command
dms-handler:Component --consumes--> dms-access:Interface
dms:System --provides--> dms-access:Interface
```

Der Graph wird aus den einmal deklarierten Referenzwerten abgeleitet. Die umgekehrte Sicht, welche Definitionen einen Vertrag benutzen, benoetigt keine zweite gepflegte Liste. Der Compiler erkennt aus einem Namen wie containers oder emits allein weder transitive Zusammensetzung noch Impactausbreitung oder eine Reihenfolge von Agentenarbeit.

## 6. Topicwahl und Antworten als typisierte Projektaussagen

Die Beispieldateien verwenden bewusst synthetische Adressen. Sie behaupten keine neue Konfiguration des untersuchten DMS-Projekts. Die lokale Quellenanalyse und ihre tatsaechlichen Topicformen stehen im [ausfuehrlichen Connector-Beispiel](../connector-example.md).

`requests:Channel` deklariert `bridge/{tenantId}/requests`. Der Koerper beschreibt die autorisierte Herkunft von tenantId und die gemeinsame Abonnementgruppe. `replies:Channel` deklariert `bridge/replies/{senderInstanceId}` mit der Rolle einer ausstellenden API-Instanz. Beide referenzieren denselben beschriebenen MQTT-Dienst.

`list-documents:Exchange` verbindet diese Kanaele mit sender api, receiver onprem und dem Operationsvertrag `dms.documents.list.v1`. Die individuelle Definition setzt 30 Sekunden pro Versuch und zwei Versuche. Der Kind erklaert einmal die Erhaltung der MessageId und die unterschiedliche Aufgabe der CorrelationId. Der individuelle Koerper beschreibt Wiederholungs- und Fehlergrenzen.

So lassen sich die Rollen, parametrisierte Topicwahl, typisierte Operationsanforderung und Antwortzuordnung als Sollbeschreibung ausdruecken. Die Sprache muss selbst weder MQTT noch HTTP verstehen. Die technische Eignung, Sicherheit und korrekte Umsetzung dieser Aussagen muessen dennoch untersucht werden.

Der Durchstich ist auf eine lesende Dokumentlistenoperation begrenzt. Gesamtfrist, Retryverzoegerung, verteilte Idempotenz, Dateitransfer, vollstaendige Authentifizierung und Deployment sowie die Erweiterungen UserManagement/Winsolvenz bleiben ausserhalb dieser kleinen Instanz. Sie bleiben notwendige Beschreibungsfragen des groesseren Beispiels; sie wurden nicht durch ein gruennes Struktururteil erledigt.

## 7. Welche Pruefung ist bereits generisch moeglich?

Mit dem vorhandenen Strukturprinzip pruefbar sind unter anderem: bekannte und eindeutige Kinds; bekannte und eindeutige Definitionen; geschlossene Felder; Werttypen und Kardinalitaeten; existierende exakt typisierte Referenzziele; eine deterministische Graphableitung und ein Inhaltsdigest.

Gegenproben des Entwurfs sind ein nicht vorhandener Empfaenger, ein Component an einem Container-only-Feld, ein fehlendes erforderliches Reply-Channel-Feld sowie ein unbekanntes Spec-Feld. Ein eigener Relationsname benoetigt keinen neuen Fall im Strukturcompiler.

Nicht allein dadurch geprueft sind die positive Bereichsregel fuer Fristen, die gueltigen Platzhalter einer Topicvorlage, die eindeutige Verwendung einer operationId im ganzen Projekt, passende Implementierungen aller angebotenen Interfaces, die erlaubte Zahl struktureller Parents und das tatsaechliche Laufzeitverhalten. Dafuer ist ein expliziter semantischer Pruefvertrag erforderlich. Der erste Entwurf verpflichtet sich weder auf einen Ausdrucksevaluator noch auf ausfuehrbaren Code in Vokabulardateien.

**Durchgefuehrte Gegenprobe:** Die beiden YAML-Dateien wurden eingelesen und ihre Koerperfelder explizit von `definition` auf die bestehenden Core-`purpose`-Felder abgebildet. Die restlichen Werte blieben erhalten; Ganzzahlwerte wurden als solche weitergegeben. Der unveraenderte aktuelle Core hat die acht Kinds, dreizehn Definitionen und siebzehn Referenzkanten erfolgreich kompiliert. Sechs ungueltige Varianten wurden wie erwartet abgewiesen: fehlende Pflichtrelation, falscher Ziel-Kind, unbekanntes Spec-Feld, nicht aufloesbares Ziel, unbekannter Definitions-Kind und doppelte Identitaet. Aenderungen am Definitionskoerper, an der Kind-Bedeutung und an der Relationsbedeutung veraenderten jeweils den Digest.

Die Gegenprobe hat auch die Grenze bestaetigt: `maxAttempts: 0` wird als Integer strukturell akzeptiert, obwohl der erklaerte Exchange-Vertrag einen positiven Wert fordert. Der [begrenzte Pruefbericht](structural-probe.json) bindet die YAML-Eingaben ueber ihre SHA-256-Werte und nennt den unveraenderten Core-Quellstand. Die temporaere Probe veraenderte keine Implementierungsdateien im Repository. Sie ist keine neue Produktfunktion und kein generischer Validator fuer die vorgeschlagene Autorenform.

Die Beispieldateien sind damit weiterhin keine im aktuellen Projektworkflow direkt ladbaren `.markitect`-Modelle und kein Nachweis einer ausgefuehrten Connector-Loesung. Kontext, Impact und Apply fuer diese eigenen Kinds wurden dadurch nicht validiert.

## 8. Bestehende Grundlage und noch zu entwerfende Anschluesse

| Bereich | Vorhanden beziehungsweise hier vorgeschlagen |
| --- | --- |
| Frei benannte nominale Kinds und geschlossene Properties | Im strukturellen Core vorhanden. |
| Typisierte Referenzen, Kardinalitaeten, Definitionengraph und Digest | Im Core vorhanden; Reference bindet genau einen Ziel-Kind. |
| `kindReference` | Vorhanden fuer den Bezug auf einen Typ. Es ist weder eine Typdeklaration noch eine Ausfuehrungsanweisung. |
| Koerper unter body | Vom Nutzer gewaehlter Feldname. Die aelteren Beispieldateien verwenden noch definition, der aktuelle Core verwendet purpose an Schema/Kind/Property/Definition. Die ausdrueckliche Abbildung kann den Inhalt erhalten; eine neue Produkt-Autorenform ist noch nicht implementiert. |
| Projektdateien waehlen ihr Vokabular | Noch zu entwerfender Loader-/Auswahlvertrag. Ein vorhandener apiVersion-Wert installiert kein Vokabular. |
| Eigene Referenznamen | Im Core als Properties strukturell ausdrueckbar. Ihre Aussagebedeutung stammt aus dem Vokabular; generischer Impact oder Apply folgt daraus noch nicht. |
| Generische Kontexte, semantische Pruefung und Apply | Muessen an Definitionen und ihren Kind-/Propertyvertraegen ausgerichtet werden. Der bisherige Manager-/Statement-Projektworkflow leistet das nicht automatisch fuer beliebige Kinds. |

Ein Modellstand muss die verwendeten Vokabulare und Definitionen zusammen fest binden. Erst werden alle ausgewaehlten Schemas erfasst, danach Definitionen zugeordnet und Referenzen aufgeloest. Fuer den Durchstich werden die beiden Dateien ausdruecklich zusammen betrachtet. Die Auswahl-/Ladekonfiguration ist Werkzeugkonfiguration; ein Projektinhalt muss dafuer kein eingebauter Project-Kind sein.

Fuer die erste Version empfehle ich exakte API-Versionen, vollstaendig bekannte Eingaben, keine implizite Vererbung, keine stillen Schemaerweiterungen und keine automatische Plugin-Ausfuehrung. Bei einer spaeteren Aufteilung in mehrere Vokabulare muessen ihre Referenzziele explizit im gebundenen Eingabesatz vorhanden sein. Der Core kann explizite Cross-Schema-Referenzen pruefen; die bequeme paketierte Bereitstellung und Versionierungs-/Migrationspolitik sind damit noch nicht entschieden.

Die bereits diskutierten expliziten Koerperlinks bleiben eine moegliche Autorenform fuer Verwendungsbezuege. Dieser Durchstich zeigt absichtlich strukturierte, typisierte Relationsfelder. Eine spaetere Linkform muss in dieselbe nachvollziehbare Referenzdarstellung normalisieren; doppelte manuelle uses-Listen und aus blossen Worttreffern erfundene Kanten werden dadurch nicht erforderlich. Der genaue Link-/Qualifizierungsvertrag bleibt offen.

## 9. Naechste Entscheidungen

Der kleinste zusammenhaengende Vorschlag lautet: **Schema, Kind, Property und Definition mit verbindlicher Prosa; eine gerichtete Relation ist zunaechst eine typisierte Reference-Property.** System, Container, Component, Rule, Command oder andere Begriffe sind Inhalte eines gewaehlten Vokabulars.

Am Beispiel sind jetzt besonders zu entscheiden: Reicht die Relation am Quell-Kind oder benoetigen wir eigenstaendige Relationsdefinitionen? Welche wenigen strukturellen Bedingungen brauchen deklarative Pruefung ueber die vorhandenen Werttypen hinaus? Wie erhalten Kontext, Aenderungsvergleich und Realisierung die Bedeutung eines benutzerdefinierten Kinds und seiner Beziehungen? Erst danach sollte die genaue Autorenform als Sprachvertrag fixiert werden.

## 10. Vollstaendiger Exchange-Durchgang mit dem minimalen Core

Auftrag des Nutzers nach der Entscheidung, requirements und roles vorerst wegzulassen. Grundlage sind die vollstaendigen Dateien [vocabulary.yaml](vocabulary.yaml) und [connector.yaml](connector.yaml). Beide verwenden bereits ausschliesslich Schema, Kind, Property, Definition und verbindliche Koerper. Sie benoetigen keine Umstellung auf die zurueckgestellte strukturierte Variante. Das Beispiel ist eine Sollbeschreibung einer lesenden Dokumentlistenoperation, keine vollstaendige Produktionsspezifikation der Connector-Plattform.

### 10.1 Den Eingabesatz und die Sprache bestimmen

Fuer diesen Durchgang werden die beiden Dateien ausdruecklich gemeinsam geladen. vocabulary.yaml liefert das Schema connector.example.org/v1alpha1 mit acht projektspezifischen Kinds. connector.yaml liefert dreizehn konkrete Definitionen. Es wird keine Bibliothek importiert; dieselben Typvertraege koennten spaeter aus einer Bibliothek kommen. Die Sprache erhaelt dadurch keine anderen Faehigkeiten.

Der gemeinsame Vertrag lautet: Kanonische Prosa ist verbindlich. Eine Kind-Beschreibung gilt fuer jede Definition genau dieses Kinds. Die Propertybeschreibung erklaert die Bedeutung ihrer Werte und Referenzen. Individuelle Beschreibungen ergaenzen die konkrete Definition; sie koennen allgemeine Pflichten nicht still aufheben. Fachliche Bedingungen und Grenzen werden in dieser Prosa beschrieben. Das Beispiel benoetigt weder requirements noch roles, subject oder eine allgemeine formale Constraint-Sprache.

Dieser gepruefte Durchstich benutzt noch definition fuer den Koerper von Schema, Kind, Property und Definition. Anschliessend waehlt der Nutzer body fuer die angestrebte Autorenform. Der heutige Core benutzt an diesen Stellen purpose; die begrenzte Probe bildet die damaligen Felder ausdruecklich ab. Der Produktloader wird durch diese Demonstration nicht geaendert.

### 10.2 Exchange als Kind beschreiben

Der allgemeine Koerper von Exchange in vocabulary.yaml beschreibt eine Request-/Reply-Vereinbarung: sender publiziert die unter command beschriebene Operation auf requestChannel und empfaengt ihre Antworten ueber eine instanzbezogene Adresse von replyChannel. receiver bearbeitet den Auftrag und antwortet an die mitgeteilte Adresse. Antworten erhalten die MessageId des Requests, die den wartenden Austausch identifiziert. CorrelationId dient der uebergreifenden Nachverfolgung. timeoutSeconds gilt pro Versuch; maxAttempts zaehlt den ersten Versuch mit. Beide Werte muessen positiv sein. Die Definition beschreibt den Vertrag, keinen einzelnen Laufzeitvorgang.

Diese Aussagen gelten fuer alle Definitionen dieses Kinds im gebundenen Schema. Die vollstaendige Kind-Deklaration steht in vocabulary.yaml. Ihre Properties sind:

| Property | Strukturvertrag | Bedeutung |
| --- | --- | --- |
| sender | reference auf Container, genau 1 | Anwendung in der ausstellenden Rolle dieses Austauschs. |
| receiver | reference auf Container, genau 1 | Anwendung in der bearbeitenden Rolle dieses Austauschs. |
| command | reference auf Command, genau 1 | Vertrag der angeforderten Operation. |
| requestChannel | reference auf Channel, genau 1 | Kanal der Auftragsrichtung. |
| replyChannel | reference auf Channel, genau 1 | Kanal fuer Antworten an die ausstellende Instanz. |
| timeoutSeconds | integer, genau 1 | Positive maximale Wartefrist eines Versuchs in Sekunden. |
| maxAttempts | integer, genau 1 | Positives Versuchsbudget einschliesslich des ersten Versuchs. |

Der folgende reale Ausschnitt aus der Beispieldatei zeigt eine Propertydeklaration innerhalb von Exchange.properties:

```yaml
sender:
  definition: Anwendung in der Rolle des ausstellenden Auftraggebers.
  type: reference
  target: {apiVersion: connector.example.org/v1alpha1, kind: Container}
  minCount: 1
  maxCount: 1
```

Der Name sender ist projektspezifisch. Der Core prueft das Referenzziel und die Anzahl. Die Bedeutung als Auftraggeber kommt aus dem beschriebenen Vertrag. Die positive Werteanforderung der Integer-Properties bleibt verbindlich, obwohl der heutige Compiler sie nicht formal prueft.

### 10.3 Die referenzierten Gegenstaende definieren

Die folgenden Definitionen stehen ebenfalls in connector.yaml; es gibt keine unaufgeloesten Platzhalterdefinitionen:

| Definition | Kind | Relevanter individueller Vertrag |
| --- | --- | --- |
| api | Container | Cloud-Anwendung; autorisierter Mandantenkontext, HTTP-Zugang und instanzbezogene Antworten. Ein Neustart uebernimmt keinen frueheren Wartezustand. |
| onprem | Container | Anwendung in der Kundenumgebung; ausgehende MQTT-Verbindung und Bearbeitung nur fuer die autorisierte Umgebung. |
| list-documents-command | Command | Operation dms.documents.list.v1; FolderId sowie optionale PageSize und ContinuationToken als Eingaben; Dokumentmetadaten, gegebenenfalls NextContinuationToken oder fachlicher Fehler als Ergebnis. Der Auftrag veraendert keine Dokumente. |
| requests | Channel | Topicfamilie bridge/{tenantId}/requests; tenantId aus der autorisierten Zielumgebung; Shared Subscription fuer den passenden Connectorpool. Die Operation wird ueber ihre Kennung ausgewaehlt. |
| replies | Channel | Topicfamilie bridge/replies/{senderInstanceId}; Antworten an die ausstellende API-Instanz, keine konkurrierende Shared Subscription fuer diese Adresse. |
| messaging | MqttService | Getrennte Brokerlaufzeit, erreichbar aus Cloud und Kundenumgebung, mit autorisierten Publikations- und Abonnementadressen. |

Die beiden Channels referenzieren messaging und verlangen at-least-once als Transportabsicht. Das ist keine Garantie einmaliger fachlicher Ausfuehrung. Ihre Topicschablonen sind Stringwerte mit beschriebener Bedeutung; Markitect fuehrt sie nicht aus. Laufzeitwerte wie tenantId, senderInstanceId und MessageId sind hier keine eigenen kanonischen Definitionen.

System-, Component- und Interface-Definitionen der vollstaendigen Datei geben den weiteren Architektur- und Zugangsrahmen. Sie werden fuer diesen Durchstich weder entfernt noch als neue Core-Begriffe behandelt.

### 10.4 Die konkrete Exchange-Definition binden

Die nachfolgende Definition stammt unveraendert aus connector.yaml:

```yaml
apiVersion: connector.example.org/v1alpha1
kind: Exchange
metadata: {namespace: connector, name: list-documents}
definition: >-
  Die API bestimmt den autorisierten Mandanten, bildet den Command und
  registriert den wartenden Austausch vor der Publikation. Der Connector waehlt
  anhand der Operationskennung die Auftragsbearbeitung. Der Transporterfolg und
  der fachliche Resultstatus werden unterschieden. Wiederholungen verwenden
  dieselbe MessageId; spaete Antworten ohne Wartezustand werden verworfen.
  Dieser lesende Durchstich verlangt keine globale Exactly-once-Garantie.
  Bei schreibenden Operationen ist die erlaubte Wiederholungswirkung gesondert
  zu entscheiden. Die Versuchszahl ist ein endliches Budget; ein Timeout
  beweist keine Nichtausfuehrung. Eine Gesamtfrist und Retry-Verzoegerung sind
  in dieser minimalen Instanz noch offen.
spec:
  sender: {namespace: connector, name: api}
  receiver: {namespace: connector, name: onprem}
  command: {namespace: connector, name: list-documents-command}
  requestChannel: {namespace: connector, name: requests}
  replyChannel: {namespace: connector, name: replies}
  timeoutSeconds: 30
  maxAttempts: 2
```

Ihre Identitaet ist das Tupel (connector.example.org/v1alpha1, Exchange, connector, list-documents). Referenzwerte lassen Kind und API-Version aus, weil ihre Properties diese genau festlegen. Beispielsweise wird sender exakt gegen (connector.example.org/v1alpha1, Container, connector, api) aufgeloest. Ein anderes Schema mit demselben Namen oder die zufaellig neueste Version ist kein Ersatz.

Der Compiler erzeugt aus den einmal gepflegten Referenzwerten fuenf ausgehende Kanten der Exchange-Definition. Die beiden Channel-Definitionen erzeugen ihre eigenen Kanten zum Broker. Eine zweite manuell gepflegte Rollen-, uses- oder Kantenliste ist nicht erforderlich.

### 10.5 Den geltenden Vertrag zusammenstellen

Zur Beurteilung von list-documents gehoeren mindestens der Schema-Koerper, der Exchange-Koerper und seine Propertyvertraege, die konkreten Werte und der individuelle Koerper sowie die einschlaegigen Vertraege der referenzierten Teilnehmer, Operation und Kanaele. Der vollstaendige Eingabesatz bleibt gebunden verfuegbar. Die genaue generische Kontextauswahl fuer das Produkt ist damit nicht bereits implementiert.

Der Kind beschreibt die allgemeine Antwortzuordnung. Die konkrete Definition ergaenzt fuer diesen Austausch unter anderem Registrierung vor Publikation, Wiederverwendung der MessageId und Verwerfen spaeter Antworten ohne Wartezustand. Die Channelvertraege erklaeren, wie Zielmandant und Senderinstanz die Adressen bestimmen. Diese Aussagen werden gemeinsam gelesen, ohne sie in eine zweite verbindliche Requirementliste zu kopieren.

Fuer die Rollenbindung reicht der beschriebene Zusammenhang: sender ist hier api; receiver ist onprem. Eine Aussage ueber sender betrifft api bei diesem Austausch. Sie wird weder zur globalen Pflicht aller API-Operationen noch zu einer automatischen Regel aller Components. Der Pruefkontext muss gleichwohl die relevanten implementierenden Teile erfassen.

### 10.6 Einen gewuenschten Laufzeitfall nachvollziehen

1. Die API nimmt einen autorisierten HTTP-Auftrag entgegen und bestimmt den Zielmandanten. Eingaben werden auf den Commandvertrag abgebildet.
2. Die ausstellende Prozessinstanz richtet ihren Antwortempfang und die Wartezuordnung ein. Sie erzeugt beziehungsweise uebernimmt die MessageId und gibt ihre autorisierte Antwortadresse gemaess replies mit.
3. Der Request wird ueber bridge/{tenantId}/requests publiziert. Die Operationskennung steht im Command beziehungsweise in der Nachrichtenhuelle, nicht im hier gewaehlten Topic.
4. Eine passende Connectorreplika empfaengt den Request, waehlt anhand dms.documents.list.v1 die Auftragsbearbeitung und liest die erlaubten Dokumentmetadaten.
5. Sie antwortet mit derselben MessageId an bridge/replies/{senderInstanceId}. Die API ordnet die Antwort ihrem wartenden Austausch zu und bildet das fachliche Ergebnis auf den HTTP-Vertrag ab.
6. Ohne Antwort beendet sie nach 30 Sekunden die Wartefrist dieses Versuchs. Ein weiterer Versuch ist innerhalb des Budgets von hoechstens zwei Versuchen moeglich und verwendet dieselbe MessageId. Ein Timeout belegt keine Nichtausfuehrung. Gesamtfrist und Verzoegerung sind noch nicht festgelegt.

Die Beschreibung legt dieses gewuenschte Verhalten fest; sie ist kein Ablaufprogramm des Core-Compilers. Ein konkreter Runtimeaufruf benoetigt keinen neuen kanonischen Exchange-Datensatz. Agenten, Implementierung, Testfaelle und ausgefuehrte Pruefnachweise entstehen in der Realisierungsarbeit.

### 10.7 Gueltigkeit und Umsetzung getrennt pruefen

| Gegenprobe | Urteil des bestehenden Strukturcompilers | Weitere Beurteilung |
| --- | --- | --- |
| replyChannel fehlt | Abgewiesen: Pflichtproperty fehlt. | Keine fachliche Ersatzannahme. |
| sender nennt einen anderen Kind | Abgewiesen: falscher Ziel-Kind. | Keine Deutung anhand aehnlicher Namen. |
| Ein Ziel existiert nicht | Abgewiesen: nicht aufloesbare Referenz. | Fehlende Definition beziehungsweise Referenz korrigieren. |
| maxAttempts ist 0 | Strukturell akzeptierter Integer. | Verletzt den beschriebenen positiven Wertebereich; Modellreview muss dies feststellen. |
| replyChannel zeigt auf requests | Strukturell akzeptierte Channel-Referenz. | Die Eignung fuer den instanzbezogenen Antwortvertrag ist fachlich zu pruefen; ein passender Zieltyp allein genuegt nicht. |
| Implementierung registriert erst nach Publikation | Aus der Modellstruktur nicht feststellbar. | Implementierungsreview und ein geeigneter Fall mit sofortiger Antwort koennen den Verstoss nachweisen. |
| Antwort enthaelt eine andere MessageId | Aus der Modellstruktur nicht feststellbar. | Zuordnungs- und Integrationspruefung gegen den festen Umsetzungskandidaten. |

**Erneut ausgefuehrte Probe:** Frisch aus den beiden YAML-Dateien adaptierte Eingaben wurden gegen den unveraenderten Core kompiliert. Ergebnis: acht Kinds, dreizehn Definitionen, siebzehn Referenzkanten; fuenf ausgehende Kanten an list-documents. Die sechs negativen Strukturvarianten wurden erneut abgewiesen. Die beiden semantischen Grenzfaelle maxAttempts=0 und replyChannel=requests wurden strukturell akzeptiert. Koerper-, Kind- und Propertybedeutung blieben im Digest enthalten. Der [gebundene Durchgangsbericht](exchange-walkthrough-probe.json) nennt Eingabehashes, Quellstand, Adaptation und normalisierte Exchange-Bezuege. Die Abwesenheit der besonderen Requirementfelder wurde an den strukturierten Eingaben geprueft.

Das ist keine ausgefuehrte Connector-Integration und kein Beleg fuer bereits generischen Kontext, Impact oder Apply. Ein Agent kann gegen den vollstaendigen gebundenen Vertrag eine Pruefliste und passende Nachweise erarbeiten. Eine solche Liste ist ein abgeleitetes Arbeitsergebnis; ihre Vollstaendigkeit ist zu beurteilen, nicht durch ihr Format garantiert. Ergebnisse muessen auf den festen Modell- und Kandidatenstand bezogen sein. Nicht gepruefte Pflichten bleiben offen.

### 10.8 Eine Aenderung weiterverfolgen

Wird timeoutSeconds von 30 auf 15 geaendert, bleiben die Kanten gleich, aber die konkrete Wartepflicht und der Modelldigest aendern sich. Wird die allgemeine Antwortzuordnung im Exchange-Kind geaendert, aendert sich der geltende Vertrag seiner Definitionen auch bei unveraenderten Instanzdateien. Eine Aenderung an replies.topicPattern betrifft dagegen einen referenzierten Vertrag; eingehende Kanten machen dessen direkte Verwendungen auffindbar.

Der Core liefert fuer diese Faelle Identitaeten, bekannte Bezuege und den gebundenen Inhalt. Er bestimmt daraus nicht automatisch die vollstaendige fachliche Aenderungswirkung oder eine Agentenaufteilung. Solange es keine engere zuverlaessige Regel gibt, darf eine semantische Bewertung relevante Pflichten nicht durch einen zu schmalen Kontext abschneiden.

### 10.9 Was dieser Durchgang klaert und was offen bleibt

Das Beispiel braucht als Sprachgrundlage vier Deklarationsbausteine: Schema, Kind, Property und Definition. Verbindliche Prosa, Identitaet, typisierte Werte, Referenzen und gemeinsame Modellbindung tragen den beschriebenen Vertrag. Exchange, Container, Command und Channel bleiben Inhalte dieses Projektvokabulars.

Offen bleiben die genaue allgemeine Koerperlink-Syntax, weitere formale Wertebedingungen, Loader-/Importkomfort sowie generische Kontext-, Impact- und Realisierungsverfahren. Fachlich ist dieser kleine Durchstich bewusst auf einen lesenden Austausch beschraenkt. Die noch offenen Fragen zu Gesamtfrist, Retryverzoegerung, Wiederholungswirkung schreibender Operationen und weiteren Betriebsbedingungen koennen durch zusaetzliche Prosa, Properties oder eigene Kinds beschrieben werden; ihre Loesung wird nicht durch ein erfolgreiches Struktururteil behauptet.

## 11. Gemeinsame Grundform und body

**Nutzerentscheidung:** Der verbindliche Koerper soll body heissen. Die Wahl gilt fuer die angestrebte Autorenform und vermeidet die doppelte Verwendung von Definition als Gegenstand und Textfeld. Eine Umbenennung des konkreten Modellgegenstands in Element war zuvor eine Empfehlung des Assistenten, keine vom Nutzer angenommene Sprachentscheidung.

**Untersuchungsauftrag:** Koennen Kind, Property und konkrete Elemente dieselbe Grundform haben, sodass Kinds selbst Definitionen sind? Die folgende Variante ist ein Diskussionsvorschlag. Sie ersetzt weder die heutigen Core-Datenstrukturen noch die nachgewiesene Compilerfunktion durch eine Behauptung bereits vorhandener Selbstbeschreibung.

### 11.1 Zwei Arten von Vereinheitlichung unterscheiden

Eine gemeinsame Dokumenthuelle kann Identitaet, Typbezug, body und strukturierte Werte vereinheitlichen. Weitergehend koennen Kinds und Properties selbst referenzierbare Definitionen werden: Exchange ist eine Definition des eingebauten Kinds Kind, sender ist eine Definition des eingebauten Kinds Property, und list-documents ist eine Definition von Exchange.

Beide Varianten teilen eine Darstellung. Erst die zweite macht die Typvertraege zu gewoehnlich adressierbaren Knoten des Modells. Sie kann Sonderformen fuer den Bezug auf Kinds und fuer die Darstellung ihrer Aenderungen vermindern. Die Bedeutung von Typdeklaration, Propertyvertrag und konkreter Verwendung bleibt dabei verschieden.

### 11.2 Eine gemeinsame Huelle am Exchange

Die vorgeschlagene Grundform lautet: Identitaet, kind als Referenz auf die Typdefinition, verbindlicher body und spec als Werte gemaess diesem Typvertrag. Die folgenden IDs wie core.Kind und connector.Exchange sind **symbolische Kurzschreibweisen fuer die Diskussion**, keine bereits festgelegte Identitaetsgrammatik. Die endgueltige Form muss Namensraum, Version und eindeutige Aufloesung bewahren. Insbesondere wird hier die heutige Identitaet aus apiVersion, kind, namespace und name nicht still weiterverwendet: Im Kandidaten ist die Identitaet einer Typdefinition von ihrem eigenen Kind zu unterscheiden und der Typbezug ausdruecklich auf sie zu richten.

Ein bewusst verkleinerter Ausschnitt mit sender und timeoutSeconds:

```yaml
id: connector.Exchange
kind: core.Kind
body: >-
  Beschreibt einen Request-/Reply-Vertrag. sender bezeichnet die
  auftraggebende Anwendung. Antworten behalten die MessageId des Requests.
  timeoutSeconds gilt als Wartefrist pro Versuch.
spec:
  properties:
    - connector.Exchange.sender
    - connector.Exchange.timeoutSeconds
---
id: connector.Exchange.sender
kind: core.Property
body: Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  name: sender
  type: reference
  target: connector.Container
  minCount: 1
  maxCount: 1
---
id: connector.Exchange.timeoutSeconds
kind: core.Property
body: Positive maximale Wartefrist pro Versuch in Sekunden.
spec:
  name: timeoutSeconds
  type: integer
  minCount: 1
  maxCount: 1
---
id: connector.list-documents
kind: connector.Exchange
body: >-
  Die API fordert Dokumentmetadaten beim Connector an. Sie registriert
  den wartenden Austausch vor der Publikation.
spec:
  sender: connector.api
  timeoutSeconds: 30
```

Der Ausschnitt ist kein vollstaendiger neuer Modelldatensatz. core.Kind, core.Property, connector.Container und connector.api muessen mit ihren uebrigen Vertraegen bereitgestellt werden; receiver, command, Kanaele und Versuchszahl des vollen Beispiels sind hier nur zur Verdeutlichung der Huelle ausgelassen. Kein gezeigtes spec wird ausgefuehrt. Die MessageId bleibt eine beschriebene Laufzeitgroesse.

| Definition | Ihr Kind | Interpretation |
| --- | --- | --- |
| connector.Exchange | core.Kind | Deklariert einen Typ fuer weitere Definitionen. |
| connector.Exchange.sender | core.Property | Deklariert eine benannte, typisierte Angabe dieses Typs. |
| connector.list-documents | connector.Exchange | Beschreibt einen konkreten gewuenschten Austausch. |

Eine Property erhaelt ihren Platz in Exchange durch dessen properties-Bezug. Ihr spec.name legt den Wertschluessel in konkreten Exchanges fest; der Compiler soll ihn nicht aus dem letzten Segment ihrer ID erraten. Doppelte Namen in einem Kind und falsche Propertyziele muessen abgewiesen werden. Ein gleichnamiges sender in einem anderen Kind besitzt dadurch keine automatisch gleiche Bedeutung.

### 11.3 Der feste Anfang verschwindet nicht

Der Core muss eine bekannte, versionierte Grunddefinition von core.Kind und core.Property mitbringen. Im selbstbeschreibenden Kandidaten hat core.Property den Kind core.Kind; core.Kind kann selbst den Kind core.Kind besitzen. Diese eine bekannte Selbstbindung ist ein festgelegter Anfang, keine Aufforderung, eine unendliche Folge von Metakinds zu laden. Die Grunddefinitionen werden gemeinsam bereitgestellt und vom Compiler mit fester Bedeutung erkannt. Projekte koennen sie weder durch gleichnamige lokale Definitionen ersetzen noch durch Prosa neue Compileroperatoren installieren.

Die Selbstbeschreibung dokumentiert den Meta-Vertrag, beweist ihn aber nicht aus sich selbst. Der Compiler muss insbesondere weiter wissen: wie eine Typdeklaration die erlaubten Properties bestimmt, welche Werttypen existieren, wie Kardinalitaeten gelten, wie eine Reference an ein Ziel gebunden wird und wie body in Modellbindung und nachfolgende Beurteilung eingeht.

Ein moeglicher enger Ladevertrag lautet:

1. Den festen, versionierten Core-Vertrag und den vollstaendigen Eingabesatz erfassen; Identitaeten auf Eindeutigkeit pruefen.
2. Die als core.Kind deklarierten Typen und ihre core.Property-Vertraege mit den bekannten Metaregeln pruefen und aufloesen.
3. Uebrige Definitionen gegen ihre nun bekannten Kinds pruefen; alle Referenzen aufloesen und den gemeinsamen Modellstand binden.

Ein kind-Bezug muss auf eine gueltige Typdefinition zeigen, nicht auf eine beliebige vorhandene Definition. Freie benutzerdefinierte Metaklassen oder eine beliebige Kette wechselnder Typinterpretationen werden fuer diesen Kandidaten nicht benoetigt. Alle benoetigten Typvertraege muessen im gebundenen Eingabesatz stehen; Dateireihenfolge darf ihre Verfuegbarkeit nicht bestimmen.

### 11.4 Gewonnene Einfachheit und neue Kosten

| Frage | Gemeinsame Definitionen | Verbleibender Bedarf / Preis |
| --- | --- | --- |
| Dokumentform | Eine Huelle fuer Typ, Property und konkreten Gegenstand. | spec hat je nach Kind weiterhin unterschiedliche Bedeutung und Struktur. |
| Referenzen | Kinds und Properties koennen mit demselben Grundmechanismus referenziert werden wie andere Definitionen. | Ein Typbezug und eine gewoehnliche fachliche Referenz haben weiterhin verschiedene Rollen. Ein Ersatz fuer das bisherige kindReference muss ausdruecklich festgelegt werden. |
| Aenderungsgraph | Kind- und Propertydefinitionen sind adressierbare Knoten; Verwendungen und Typbezuege koennen gemeinsam sichtbar werden. | Die vollstaendige fachliche Aenderungswirkung ist damit noch nicht bestimmt. Der heutige Core bildet Kindbezuege nicht als Definitionenkanten ab. |
| Zahl der Grundformen | Eine allgemeine Definition statt getrennter Dokumentformen. | Kind und Property bleiben zwei vom Compiler verstandene Meta-Kinds; Werttypen, Bindung und Kardinalitaet bleiben feste Semantik. |
| Properties als eigene Definitionen | Eigene Identitaet, Referenzierbarkeit, body und Herkunft pro Property. | Mehr Deklarationen und Verweise fuer kleine Vertraege; Versionierung und Zuordnung muessen eindeutig sein. |
| Schema / Bibliothek | Vokabulare koennen als versionierte Zusammenstellungen von Definitionen bereitgestellt werden. | Auswahl, Namensraum, exakte Versionsbindung und geschlossene Typvertraege bleiben noetig. Ob ein Schema selbst ein Kind wird, ist noch offen. |

Eine spaetere eingebettete Kurzform fuer Properties waere moeglich, wenn sie exakt auf dieselben adressierbaren Deklarationen normalisiert und keine zweite Bedeutung erfindet. Fuer die gegenwaertige Diskussion wird sie nicht zusaetzlich festgelegt. Eine einheitliche Huelle allein zwingt nicht dazu, jede kleine Property in eine eigene Datei auszulagern; mehrere Definitionen koennen in einem Dokument stehen.

### 11.5 Empfehlung und erforderliche Gegenproben

**Empfehlung zur weiteren Pruefung:** Eine Definition als allgemeine Grundform untersuchen; Kind und Property als die vom Core verstandenen Typdefinitionen darin darstellen. Die konkrete Projektbeschreibung ist dann eine Definition eines selbst definierten Kinds. body ist in allen Faellen der verbindliche Koerper. Die bisherige Bezeichnung Element kann zur Erklaerung konkreter Gegenstaende dienen, ohne einen zusaetzlichen eingebauten Kind zu schaffen.

Das verringert die Zahl der Darstellungsformen, nicht die Zahl aller erforderlichen Sprachregeln. Die naechsten Gegenproben sind ein vollstaendiger fester Core-Anfang, doppelte oder widerspruechliche Propertydeklarationen, falsche kind-Ziele, exakte Cross-Vokabular-Versionen sowie Aenderungen an Typ- und Propertybedeutung bei unveraenderten konkreten Definitionen. Es muss geprueft werden, ob die einheitlichen Bezuege Kontext und Aenderungsbewertung vereinfachen, statt nur YAML umzuschichten.

Die bisherigen Strukturproben pruefen getrennte Schemas und Definitionen mit ausdruecklicher Koerperabbildung. Sie belegen weder die neue id-/kind-Huelle noch den selbstbeschreibenden Anfang oder einen generischen Meta-Compiler. In diesem Schritt wurden nur der Entwurf und der gewaehlte Feldname dokumentiert.

## 12. Fester Anfang und Typaufloesung am Exchange

Auftrag des Nutzers: Den festen Anfang mit Kind und Property sowie die vollstaendige Typaufloesung am Exchange durchspielen und die wirklich fest benoetigten Coreregeln sichtbar machen. Die folgenden Dateien bilden zusammen einen geschlossenen strukturellen Eingabesatz fuer diesen Kandidaten:

- [bootstrap/core.yaml](bootstrap/core.yaml): zehn feste Core-Definitionen mit body.
- [bootstrap/connector.yaml](bootstrap/connector.yaml): acht Projekt-Kinds, zwanzig Projekt-Properties und dreizehn konkrete Gegenstaende.
- [bootstrap/probe.json](bootstrap/probe.json): Ergebnisse der begrenzten eigenstaendigen Entwurfsprobe, Eingabehashes und ausgewaehlte aufgeloeste Kanten.

Die neue Form ist ein Vorschlag. Die Probe ist kein Aufruf des bestehenden Go-Core-Compilers, keine neue Produktfunktion und keine ausgefuehrte Connector-Loesung. Sie untersucht ausschliesslich den endlichen Anfang, die Typ-/Propertyaufloesung, Strukturpruefung und gemeinsame Inhaltsbindung. body ist die bereits gewaehlte Autorenbezeichnung. requirements und roles bleiben weggelassen.

### 12.1 Die feste Dokumenthuelle

Jede Definition besitzt genau id, kind, body und spec. id ist eine eindeutige, nichtleere Identitaet, kind ein exakter Bezug auf eine Typdefinition, body nichtleere verbindliche Prosa und spec ein Objekt mit den typgemaess erlaubten Werten. Ein mehrfach vorhandenes id ist ungueltig.

Die Probe benutzt IDs wie core/v1/Kind, connector/v1/Exchange und project/v1/list-documents als exakte Strings. Die Versionsanteile sind ausdruecklich angegeben; es gibt weder Kurznamenssuche noch automatische Wahl der neuesten Version. Diese lesbare Notation ist weiterhin ein Kandidat fuer die Identitaetsform. Die Werte werden nicht aus Dateinamen oder aus den letzten ID-Segmenten erraten. core/ bezeichnet in der Probe den reservierten Grundvertrag; ein Projekt darf dort keine Definition ersetzen oder hinzufuegen.

### 12.2 Zwei Meta-Kinds und ihre acht eigenen Properties

Der feste Anfang umfasst nicht nur zwei Namen. Er muss auch deren Vertragsstruktur bereitstellen. Kompakt dargestellt:

```yaml
id: core/v1/Kind
kind: core/v1/Kind
body: >-
  Deklariert einen Typ. properties benennt seine erlaubten Angaben.
  Sein body beschreibt die gemeinsame Bedeutung seiner Definitionen.
spec:
  properties:
    - core/v1/Kind/properties
---
id: core/v1/Property
kind: core/v1/Kind
body: Deklariert eine benannte Eigenschaft oder Beziehung eines Typvertrags.
spec:
  properties:
    - core/v1/Property/name
    - core/v1/Property/type
    - core/v1/Property/minCount
    - core/v1/Property/maxCount
    - core/v1/Property/values
    - core/v1/Property/target
    - core/v1/Property/properties
```

Alle acht referenzierten Properties sind selbst Definitionen von core/v1/Property. Ihre Vertraege sind:

| Property | Werttyp | Anzahl | Festgelegte Aufgabe |
| --- | --- | --- | --- |
| Kind.properties | reference auf Property | 0 bis beliebig viele | Verknuepft einen Typ mit seinen erlaubten Propertyvertraegen. |
| Property.name | string | genau 1 | Benennt den Wertschluessel in einer konkreten spec. |
| Property.type | enum | genau 1 | Waehlt einen bekannten Werttyp. |
| Property.minCount | integer | genau 1 | Bestimmt die Mindestanzahl von Werten. |
| Property.maxCount | integer | genau 1 | Bestimmt Hoechstanzahl und Einzelwert-/Listenform. |
| Property.values | string | 0 bis beliebig viele | Legt die erlaubten Werte eines enum fest. |
| Property.target | reference auf Kind | 0 oder 1 | Legt den Ziel-Kind einer reference fest. |
| Property.properties | reference auf Property | 0 bis beliebig viele | Beschreibt die Felder eines object. |

Damit bestehen die zehn festen Definitionen aus zwei Typdefinitionen und acht Propertydefinitionen. Das ist ein vollstaendiger Anfang fuer den hier untersuchten Werteumfang, kein Beweis der kleinstmoeglichen Zahl aller notwendigen Deklarationen.

### 12.3 Die entscheidende zyklische Definition

Der vollstaendige feste Datensatz enthaelt insbesondere:

```yaml
id: core/v1/Kind/properties
kind: core/v1/Property
body: Die Propertyvertraege dieses Kinds mit innerhalb des Kinds eindeutigen Namen.
spec:
  name: properties
  type: reference
  minCount: 0
  maxCount: -1
  target: core/v1/Property
---
id: core/v1/Property/target
kind: core/v1/Property
body: Nur bei reference erforderlich; bestimmt den exakten Ziel-Kind.
spec:
  name: target
  type: reference
  minCount: 0
  maxCount: 1
  target: core/v1/Kind
```

Kind.properties zeigt auf Properties; Property.target zeigt auf Kinds; Kind hat selbst den Kind Kind. Das ist ein endlicher Referenzgraph. Referenzpruefung bedeutet hier, Identitaet und deklarierten Ziel-Kind zu vergleichen. Sie bedeutet nicht, den gesamten Zielvertrag bei jedem Bezug rekursiv neu zu entfalten. Deshalb benoetigt dieser Anfang keinen Kind des Kind des Kind in unendlicher Folge.

Der Compiler beginnt trotzdem nicht mit unbekanntem YAML und glaubt dessen Selbstbeschreibung. Er startet mit den von der gewaehlten Core-Version fest mitgelieferten Meta-Vertraegen, prueft damit die Core-Definitionen und kontrolliert anschliessend, dass ihre daraus abgeleiteten Vertraege genau diesem Anfang entsprechen. Die Probe benutzt dazu zwei feste Metavertragstabellen. Der Core-Seed ist eine ausdruecklich vertrauenswuerdig bereitgestellte Eingabe der Probe; die Probe implementiert keine Paketinstallation, Signaturpruefung oder Produktions-Laderabsicherung.

### 12.4 Das Exchange-Vokabular anschliessen

Im vollstaendigen Connector-Datensatz ist Exchange eine Definition von core/v1/Kind. Seine properties-Liste referenziert die sieben Definitionen sender, receiver, command, requestChannel, replyChannel, timeoutSeconds und maxAttempts. Deren bodies beschreiben ihre Bedeutung; ihre specs deklarieren den Strukturvertrag.

Beispielsweise:

```yaml
id: connector/v1/Exchange/sender
kind: core/v1/Property
body: Anwendung in der Rolle des ausstellenden Auftraggebers.
spec:
  name: sender
  type: reference
  target: connector/v1/Container
  minCount: 1
  maxCount: 1
```

Die Typdefinition connector/v1/Container besitzt den Kind core/v1/Kind. Die konkrete API besitzt dagegen den Kind connector/v1/Container. Dieser Unterschied ist ausschlaggebend: sender.target darf Container referenzieren, nicht die konkrete API.

Die konkrete Definition project/v1/list-documents hat den Kind connector/v1/Exchange. Sie setzt sender auf project/v1/api, receiver auf project/v1/onprem, command auf project/v1/list-documents-command, requestChannel auf project/v1/requests und replyChannel auf project/v1/replies. Ihre Werte timeoutSeconds=30 und maxAttempts=2 bleiben unveraendert. Der individuelle body beschreibt unter anderem Registrierung vor Publikation, Wiederholungen mit derselben MessageId und das Verwerfen spaeter Antworten ohne Wartezustand.

Die neue Datei uebernimmt die acht Typkoerper, zwanzig Propertykoerper und dreizehn individuellen Koerper aus dem vorherigen Beispiel und stellt ihre Bezuege explizit dar. Sie ist kein Nachweis einer verlustfreien allgemeinen Migration des alten Schemaformats; insbesondere bleibt die allgemeine Paket-/Schemahuelle eine separate Designfrage.

### 12.5 Typaufloesung ohne Abhaengigkeit von der Dateireihenfolge

1. **Grundvertrag waehlen.** Den fest mitgelieferten Core-Vertrag fuer core/v1 bereitstellen. Ein Projektimport kann ihn nicht ueberschreiben.
2. **Identitaeten erfassen.** Alle Core- und Projektdefinitionen gemeinsam indizieren. Huelle, Eindeutigkeit und reservierte IDs pruefen. Die Reihenfolge der Dokumente ist unerheblich.
3. **Typbezug pruefen.** Jeder kind-Verweis muss auf eine vorhandene Definition zeigen, deren eigener kind exakt core/v1/Kind ist. Die feste Selbstbindung von core/v1/Kind ist gueltig. Beliebige selbsttypisierte Projektdefinitionen sind dadurch nicht zugelassen.
4. **Metavertraege pruefen.** Definitionen von Kind und Property gegen den festen Anfang pruefen. Dann ihre zusaetzlichen Sprachinvarianten pruefen, etwa gueltige Kardinalitaeten und die Pflicht von target bei reference.
5. **Typvertraege aufbauen.** Fuer jeden Kind die referenzierten Propertydefinitionen anhand spec.name sammeln. Doppelte Feldnamen sind Fehler. Objektvertraege werden begrenzt aufgebaut; zyklische unendlich zu entfaltende Objektfeldvertraege sind in diesem Kandidaten ungueltig. Gewoehnliche Referenzzyklen sind davon verschieden.
6. **Alle Werte pruefen.** Jede Definition, einschliesslich der Core-Definitionen, gegen ihren so bekannten Kind pruefen. Unbekannte Felder, falsche Wertformen, fehlende Pflichtwerte oder falsche Referenzziele abweisen.
7. **Modell binden.** Alle Definitionen samt bodies und strukturierter Vertraege normalisieren, Typ- und Wertreferenzen als unterscheidbare Kanten festhalten und einen gemeinsamen Inhaltsdigest bilden.

Fuer Exchange.sender ergibt sich die konkrete Pruefkette:

```text
list-documents.kind -> Exchange
Exchange.kind -> core.Kind
Exchange.spec.properties -> Exchange.sender
Exchange.sender.kind -> core.Property
Exchange.sender.spec.target -> Container
Container.kind -> core.Kind
list-documents.spec.sender -> api
api.kind -> Container
```

Damit ist formal geprueft, dass sender eine existierende Container-Definition bezeichnet. Die Aussage, dass diese Anwendung fachlich als Sender geeignet ist und Nachrichten richtig behandelt, folgt nicht aus dieser Kette.

### 12.6 Welche Regeln wirklich fest im Core stehen

| Feste Regel | Warum sie nicht allein aus dem body folgt |
| --- | --- |
| Huelle und Identitaet | Der Compiler muss wissen, welche Felder Identitaet, Typbezug, Koerper und strukturierte Werte tragen. |
| Bedeutung der zwei Meta-Kinds | Nur die fest gebundenen Kind-/Property-Identitaeten erzeugen Typ- und Feldvertraege. Ein Projekt-Kind mit dem Namen Kind hat keine Sonderrechte. |
| Verbindung von Typ und Properties | spec.properties eines Kinds und spec.name einer Property bestimmen seine erlaubten Wertschluessel; ID-Segmente tun das nicht. |
| Werttypsemantik | string, boolean, integer, number, enum, object und reference brauchen implementierte Pruefregeln. Ein neuer Enumtext installiert keinen neuen Operator. |
| Kardinalitaetssemantik | minCount ist nichtnegativ; maxCount ist -1 oder mindestens minCount. maxCount=1 bedeutet Einzelwert, sonst Liste. Null und unbekannte Felder sind im Kandidaten ungueltig. |
| Bedingte Metaregeln | reference erfordert target; enum erfordert eindeutige erlaubte Werte; object verwendet seine Feldvertraege. Unpassende Zusatzfelder sind Fehler. Diese Zusammenhaenge sind im Seed nicht durch blosses integer oder string ausgedrueckt. |
| Exakte Referenzpruefung | Ein Ziel muss existieren und sein deklarierter Kind muss zum Propertyvertrag passen. Keine implizite Vererbung, Versionsergaenzung oder Suchheuristik. |
| Bindung und allgemeine Koerpergeltung | Typ- und Propertykoerper gehoeren zum Vertrag ihrer Verwendungen; individuelle Aussagen ergaenzen sie. Widersprueche werden nicht durch stilles Ueberschreiben geloest. Der Digest muss die gesamten bodies erhalten. |
| Endlichkeit und Grenzen | Die Verarbeitung muss mit festen Groessen-/Tiefengrenzen enden. Der Prototyp ist keine gehaertete Produktionsimplementierung aller Eingabegrenzen. |

Insbesondere ist die Bereichsregel fuer Property.minCount eine fest verstandene Sprachinvariante. Daraus folgt keine automatische Kenntnis des positiven Wertebereichs von Exchange.maxAttempts. Letzterer steht weiterhin in fachlicher Prosa und benoetigt semantische Beurteilung, solange kein allgemeiner Werteconstraint eingefuehrt wird.

Ein eigener kindReference-Werttyp ist im Kandidaten nicht noetig: Ein Bezug auf einen Typ ist eine normale reference mit target core/v1/Kind. Der heutige Produkt-Core hat weiterhin seinen bisherigen kindReference-Typ; dessen Migration wird hier nicht implementiert.

### 12.7 Tatsaechlich durchgefuehrte Entwurfsprobe

Ein temporaerer eigenstaendiger Python-Prototyp hat den fest bereitgestellten Core-Vertrag und die neue Connector-Datei geladen. Die Basis wurde strukturell akzeptiert: zehn feste Core-Definitionen plus einundvierzig Projektdefinitionen ergeben einundfuenfzig Knoten, darunter zehn Kinds und achtundzwanzig Properties. Der abgeleitete Graph besitzt einhundertdreizehn Kanten, einschliesslich kind-Verweisen und den neuen Meta-Beziehungen. Er hat deshalb einen anderen Umfang als die siebzehn rein fachlichen Referenzkanten des frueheren Strukturbeispiels.

Sechzehn ungueltige Varianten wurden gezielt abgewiesen, darunter: Core-Ueberschreibung, doppelte Identitaet, unbekannter Kind, konkretes Element als kind-Ziel, konkretes Element in einer Propertyliste, doppelte Propertynamen, konkretes Element als Property.target, fehlendes target, ungueltige Kardinalitaet, unbekannter Werttyp, unbekannter formaler Constraint, fehlende Pflichtproperty, falscher Referenz-Kind, fehlende genaue Version, unbekanntes Instanzfeld und beliebige Selbsttypisierung. Die Fehlercodes stehen im verlinkten Bericht.

Die Probe bestaetigt ausserdem fuer diesen Datensatz die Selbstkonsistenz des festen Anfangs, Reihenfolgeunabhaengigkeit und die Beruecksichtigung geaenderter Kind-, Property- und individueller bodies im Digest. Ein zusaetzlicher gewoehnlicher Projekt-Kind namens Kind erhaelt keine besondere Sprachbedeutung. maxAttempts=0 und replyChannel=requests bleiben dagegen strukturell akzeptiert und verlangen dieselbe fachliche Beurteilung wie zuvor.

Der Bericht bindet die beiden Eingabedateien und den temporaeren Prototyp ueber SHA-256 und nennt den Repository-Quellstand als Kontext. Er ist Evidenz fuer diese begrenzte Entwurfsprobe, kein Ergebnis des heutigen Go-Core-Compilers oder ein universeller Sprachbeweis. Die neue Probe prueft keine selbststaendige semantische Interpretation von Prosa, keinen produktiven Loader, keine generische Delegation, keinen vollstaendigen Impact und keine Connector-Laufzeit.

### 12.8 Bewertung dieses Kandidaten

Die gemeinsame Grundform kann den festen Anfang und das vollstaendige strukturierte Exchange-Beispiel tragen. Die Typ- und Propertybezuege sind nun im selben adressierbaren Graphen wie die konkreten Gegenstaende. Dazu muss der Core weiterhin zwei Meta-Kinds, die Interpretation ihrer Felder, Werttypen, Bindung und Kardinalitaeten fest verstehen.

Der Gewinn ist eine gemeinsame Definitionenform und ein gemeinsamer Bezugsmechanismus. Die festen Regeln sind durch die Selbstbeschreibung sichtbar und nachvollziehbar, aber nicht aus dem Compiler verschwunden. Ob das fuer Autoren und die nachfolgende Markitect-Verarbeitung insgesamt einfacher ist als getrennte Schema-/Definitionsformen, bleibt die naechste Bewertung. Es wird hier keine generelle Ausreichendheit allein aus erfolgreicher Strukturpruefung abgeleitet.

## 13. Vergleich und Empfehlung fuer Markitect

Auftrag des Nutzers: Die gemeinsame Grundform mit dem bisherigen Core vergleichen, echte Einsparungen und neue Komplexitaet benennen und eine Variante empfehlen. Bewertet werden der vorhandene strukturelle Core und die begrenzte selbstbeschreibende Entwurfsprobe. Die aktuell fest akzeptierten Entscheidungen body sowie das vorlaeufige Weglassen von requirements und roles gelten unabhaengig von der Variantenwahl.

**Nachfolgende Nutzerentscheidung:** Die einheitliche Grundform mit Kind und Property wird gewaehlt. Die Empfehlung in 13.4 ist damit ueberholt; die sachlichen Abwaegungen bleiben die Herleitung. Die aktuelle Richtung und die kompakte Propertyform stehen in Abschnitt 14.

### 13.1 Was nicht eingespart wird

Kind bleibt ein Typvertrag, Property bleibt eine Feld-/Relationsdeklaration und ein konkreter Gegenstand bleibt eine Verwendung dieses Vertrags. Die gemeinsame Huelle kann diese Rollen ausdruecken, hebt sie aber nicht auf. Beide Varianten benoetigen Werttypen, Kardinalitaeten, exakte Referenzbindung, Namens-/Versionsregeln sowie verbindliche Prosa und geeignete semantische Beurteilung.

Frei definierbares Projektvokabular ist bereits im bisherigen Core moeglich. Weder Bibliotheksimporte noch ein verbindlicher Koerper noch die Beruecksichtigung einer Kind-Aenderung setzen einen selbsttypisierten Kind voraus. Der vorhandene Digest bindet bereits die Kind- und Propertybedeutung. Der bisherige Produktworkflow mit festem Manager-/Statement-Vokabular ist eine weitere Schicht und darf nicht mit der Ausdrucksfaehigkeit des strukturellen Cores verwechselt werden.

### 13.2 Was die gemeinsame Grundform wirklich gewinnt

| Bereich | Bisheriger expliziter Core | Selbstbeschreibender Kandidat | Tatsaechlicher Unterschied |
| --- | --- | --- | --- |
| Dokumentstruktur | Schema mit Kind-/Propertyvertraegen; konkrete Definitionen daneben. | Eine Huelle aus id, kind, body und spec. | Eine einheitliche Datensatzform. |
| Kind und konkrete Gegenstaende | Unterschiedliche adressierbare Vertragstypen; kindReference als eigener Werttyp. | Kinds sind Definitionen des festen Meta-Kinds Kind. | Typziele koennen mit dem allgemeinen Referenzmechanismus adressiert werden. Die besondere Typpruefung bleibt. |
| Properties | Im Typvertrag eingebettet und ueber ihren Namen beziehungsweise Pfad bestimmt. | Properties besitzen explizite Definitionenidentitaeten und werden vom Kind referenziert. | Erstklassige Propertyknoten fuer Anzeige, Herkunft und Bezuege. |
| Gemeinsamer Propertyvertrag | Gleichartige Angaben stehen im bisherigen Core als eingebettete Deklarationen an ihren jeweiligen Kinds; eine Wiederverwendung braucht Autoren-/Loaderhilfe oder eine Erweiterung. | Mehrere Kinds koennen ausdruecklich dieselbe Propertydefinition referenzieren. | Ein gemeinsam gepflegter Feldvertrag wird strukturell unmittelbar moeglich. Gleichnamige Felder sind dadurch nicht automatisch gleich; der bisherige Exchange-Datensatz benoetigt diese Wiederverwendung nicht. |
| Modellgraph | Der heutige Core liefert Kanten zwischen konkreten Definitionen; Typvertraege liegen daneben. | Der Prototyp stellt auch kind-Bezuege und Meta-Beziehungen als Kanten dar. | Gemeinsamer Graph wird unmittelbar sichtbar. Eine solche Sicht kann auch aus dem expliziten Core abgeleitet werden. |
| Schema | Deklariert einen genauen versionierten Typvertrag. | Kann als Paket-/Importzusammenstellung ausserhalb der einheitlichen Definitionenform behandelt werden. | Das besondere Schemadokument kann entfallen; Gruppierung, Versionierung und Auswahl bleiben erforderlich. |
| Identitaet | Konkrete Identitaet enthaelt den Kind. | Der Kandidat trennt id vom kind-Verweis. | Eine eigenstaendige Identitaetsentscheidung; sie ist nicht logisch an Selbstbeschreibung gebunden. |

Eine gemeinsame Huelle und die Ablage im selben Graphen sind echte Vereinheitlichungen. Die Behauptung, vier notwendige semantische Konzepte auf eines reduziert zu haben, waere dagegen zu stark. Ein Grundformat enthaelt weiterhin zwei fest interpretierte Meta-Kinds und konkrete Verwendungen. Ebenso wird ein entfallendes kindReference durch die weiterhin erforderliche Unterscheidung zwischen Typziel und konkretem Ziel semantisch nicht voellig ersatzlos gestrichen.

### 13.3 Welche Komplexitaet hinzukommt

Der explizite Core beginnt mit bekannten Datenstrukturen und prueft gelieferte Typvertraege. Der selbstbeschreibende Kandidat benoetigt zusaetzlich einen festen versionierten Anfang, seine Grunddefinitionen und den Abgleich zwischen ihrer deklarativen Darstellung und den implementierten Metaregeln. Kind.kind=Kind ist begrenzt handhabbar, muss aber eine ausdrueckliche Sprachregel und fuer Werkzeuge verstaendlich sein.

Bei einer vollstaendigen Vereinheitlichung kommen ausserdem folgende Aufgaben hinzu:

- Core-Identitaeten und der gelieferte Grundvertrag muessen von Projektdefinitionen unterscheidbar und gegen Ueberschreiben geschuetzt sein.
- kind darf nur gueltige Typdefinitionen referenzieren. Beliebige vorhandene Knoten oder frei selbsttypisierte Projektknoten reichen nicht.
- Properties erhalten eigene Identitaeten und einen zusaetzlichen Eintrag in der Propertyliste ihres Kinds. name und id besitzen unterschiedliche Aufgaben.
- Ein kleiner Typvertrag wird in mehrere Datensaetze und Bezuege aufgeteilt. Eine eingebettete Autorenkurzform wuerde wiederum einen ausdruecklichen Normalisierungsvertrag brauchen.
- Paketierung und Versionierung muessen fuer die neue Identitaetsform geklaert werden. Die bisherigen vierteiligen Definitionenidentitaeten gehen nicht automatisch darin auf.

Die projektfachlichen Inhalte werden dadurch nicht weniger: Der alte Durchstich beschreibt acht Kinds, zwanzig Properties und dreizehn konkrete Gegenstaende. Die neue Datei beschreibt dieselben einundvierzig Projektbestandteile nun als explizite Definitionen und fuegt zehn feste Core-Definitionen hinzu. Dreizehn alte Definitionen mit einundfuenfzig neuen Knoten direkt zu vergleichen, waere irrefuehrend; die alten eingebetteten Vertraege duerfen nicht verschwinden. Genauso wenig belegt die neue Knoten- oder Kantenzahl fuer sich eine Laufzeit- oder Kontextkostensteigerung. Dafuer wurde keine Messung durchgefuehrt.

### 13.4 Empfehlung

**Damals abgegebene Assistentenempfehlung, inzwischen durch die Nutzerwahl abgeloest:** Fuer Markitect zunaechst den expliziten strukturellen Core weiterentwickeln. Kind- und Propertyvertraege bleiben unmittelbar deklarierte Typstruktur, konkrete Gegenstaende verwenden diese Typen. Alle Koerper heissen in der angestrebten Autorenform body. requirements und roles erhalten wie beschlossen keine besondere Verarbeitung.

Der relevante Gewinn der neuen Untersuchung soll erhalten bleiben: **Kind-, Property- und konkrete Definitionen sollen gemeinsam adressierbar und im Modellgraphen sichtbar sein.** Dafuer koennen bestehende Schema-/Kindidentitaeten und eindeutige Propertypfade verwendet beziehungsweise geeignet praezisiert werden. Typbezuege und Zuordnung der Properties werden daraus abgeleitet. Autoren muessen nicht zusaetzlich jede Property als eigenstaendige Instanz von core.Property pflegen. Eine solche Graphsicht und ihre genaue Adressierung sind eine geplante Erweiterung, keine bereits vorhandene Funktion des heutigen Definitionengraphen.

Das trennt eine einheitliche Sicht fuer Werkzeuge und Kontextbildung von einer vollstaendigen Selbstbeschreibung der Sprache. Eine eigene Referenz auf einen Kind kann im expliziten Sprachvertrag vorerst als kindReference bestehen bleiben. Referenzen auf Propertyvertraege und Koerperlinks brauchen gegebenenfalls einen allgemeinen Symboladressvertrag; dessen genaue Syntax ist weiterhin offen. Weder diese Adressierung noch eine neue unabhaengige Elementidentitaet werden hier nebenbei als akzeptiert vorausgesetzt.

Entscheidend fuer die Empfehlung ist, dass die bisher gefundenen Beschreibungsbedarfe eine vollstaendige Behandlung der Typ- und Propertydeklarationen als gleichartige Autorendatensaetze noch nicht zwingend verlangen. Der feste selbstbeschreibende Anfang schafft vor allem Einheitlichkeit der Darstellung und unmittelbar referenzierbare, gegebenenfalls gemeinsam nutzbare Propertyvertraege. Die Arbeit an vollstaendigen geltenden Kontexten, semantischer Aenderungsbewertung und unabhaengiger Realisierungspruefung bleibt in beiden Varianten bestehen. Eine geringere Zahl sichtbarer Dokumentformen allein rechtfertigt den zusaetzlichen Meta-Vertrag deshalb vorerst nicht. Das ist eine begruendete Entwurfsentscheidung, keine Messung des spaeteren Implementierungs- oder Wartungsaufwands.

Die gemeinsame Definitionenform bleibt eine ernsthafte Alternative, wenn sich die unmittelbare Bearbeitung und Verknuepfung von Typ- und Propertydefinitionen als zentraler Arbeitsfall erweist und eine abgeleitete gemeinsame Sicht dafuer nicht ausreicht. Der bisherige Prototyp beweist weder diesen Zusatznutzen noch die generelle Ueberlegenheit einer Variante. Es wurde keine Implementierung auf den empfohlenen Weg umgestellt.

## 14. Gewaehlte Grundform und kompakte Properties

**Nutzerentscheidung:** Markitect verwendet als neue Designrichtung die einheitliche Grundform mit Kind und Property. Eine Typdefinition, eine Propertydefinition und die Beschreibung eines konkreten Projektgegenstands sind Definitionen mit Identitaet, kind-Bezug, verbindlichem body und strukturierten Werten. Kind und Property sind die fest verstandenen Grundtypen. Projektbegriffe werden als weitere Kinds definiert. Die Wahl von body und das vorlaeufige Weglassen von requirements und roles bleiben bestehen.

**Zusaetzlicher Auftrag:** Properties kompakt direkt beim Kind darstellen. Die folgende Autorenform und ihre Normalisierung sind ein konkreter Vorschlag zu diesem Auftrag. Der Produktcompiler und sein heutiges Autorenformat wurden nicht geaendert.

### 14.1 Eine eingebettete Property ist eine Kurzform ihrer Definition

Unter spec.properties eines Kinds kann statt einer Liste ausgeschriebener Propertyreferenzen eine Zuordnung vom Feldnamen zu eingebetteten Propertyvertraegen stehen. Eine eingebettete Property enthaelt ihren body und ihre Vertragsfelder direkt. id, kind und name werden eindeutig aus dem Ort ergaenzt; sie werden nicht noch einmal gepflegt.

Ein gekuerzter Exchange-Ausschnitt:

```yaml
id: connector/v1/Exchange
kind: core/v1/Kind
body: >-
  Beschreibt einen Request-/Reply-Vertrag zwischen sender und receiver.
  Antworten behalten die MessageId des Requests.
  timeoutSeconds gilt als Wartefrist pro Versuch.
spec:
  properties:
    sender:
      body: Anwendung in der Auftraggeberrolle dieses Austauschs.
      type: reference
      target: connector/v1/Container
      minCount: 1
      maxCount: 1
    receiver:
      body: Anwendung in der Bearbeiterrolle dieses Austauschs.
      type: reference
      target: connector/v1/Container
      minCount: 1
      maxCount: 1
    timeoutSeconds:
      body: Positive maximale Wartefrist pro Versuch in Sekunden.
      type: integer
      minCount: 1
      maxCount: 1
```

Der Ausschnitt reduziert den Vertrag nur zur Erklaerung. Die [vollstaendige kompakte Connector-Datei](bootstrap/connector-compact.yaml) enthaelt alle acht Kinds und dreizehn konkreten Gegenstaende mit saemtlichen sieben Exchange-Properties und insgesamt zwanzig eingebetteten Propertydefinitionen.

Der Ausschnitt sender wird vor der Typpruefung genau zu folgender ausgeschriebener Definition:

```yaml
id: connector/v1/Exchange/sender
kind: core/v1/Property
body: Anwendung in der Auftraggeberrolle dieses Austauschs.
spec:
  name: sender
  type: reference
  target: connector/v1/Container
  minCount: 1
  maxCount: 1
```

Am ausgeschriebenen Exchange steht danach unter spec.properties die Referenz connector/v1/Exchange/sender, entsprechend fuer alle weiteren Properties. Das Modell enthaelt weiterhin dieselbe Propertydefinition mit eigener Identitaet und Bedeutung. Es gibt keine zwei parallel zu pflegenden kanonischen Beschreibungen; die ausgeschriebene Form ist das abgeleitete Ergebnis der kompakten Eingabe.

### 14.2 Eindeutige Normalisierungsregeln

1. Die Kurzform gilt in den bekannten Metakontexten Kind.spec.properties und object-Property.spec.properties. Ein beliebiges fachliches Feld namens properties aktiviert sie nicht.
2. Der Zuordnungsschluessel ist der Propertyname. Eine eingebettete Definition bekommt die ID aus der exakten Besitzer-ID, einem Trenner / und diesem Namen; kind ist core/v1/Property. spec.name wird aus dem Schluessel gesetzt.
3. body wird unveraendert in die Propertydefinition uebernommen. type, target, minCount, maxCount sowie gegebenenfalls values und properties werden in ihre spec uebernommen. Es werden keine zusaetzlichen Kardinalitaetsdefaults eingefuehrt.
4. Die eingebettete Form darf id, kind und name nicht widerspruechlich nochmals angeben. Unbekannte Felder sind Fehler. Der Prototyp verlangt einen nichtleeren Namen ohne /; damit ist der Trenner fuer die Ableitung eindeutig. Die endgueltige Identitaetsgrammatik bleibt zu spezifizieren.
5. Die erzeugten Definitionen werden zusammen mit dem uebrigen Eingabesatz indiziert und typgeprueft. Eine Kollision mit einer anderen erzeugten oder ausgeschriebenen ID ist ein Fehler, auch bei gleichem Inhalt. Doppelte YAML-Schluessel werden bereits beim Einlesen abgewiesen.
6. Die bisherige ausgeschriebene Referenzliste bleibt moeglich. Als kompakte Wiederverwendung kann ein Zuordnungswert direkt eine Property-ID sein; deren name muss mit dem Zuordnungsschluessel uebereinstimmen. Ein Alias, der die Property still umbenennt, ist nicht vorgesehen.
7. Bei einem object koennen dessen Felder wiederum kompakt deklariert werden. Ihre IDs werden relativ zur umgebenden Property abgeleitet, beispielsweise Kind/envelope/messageId. Die bekannten Tiefengrenzen gelten weiter.

Fuer eine spaetere produktive Umsetzung muss Herkunft erhalten bleiben: Ein Fehler an der erzeugten sender-Property soll auf die entsprechende eingebettete Angabe zeigen. Der Normalisierungsbericht dokumentiert dafuer Besitzer, Feldschluessel und erzeugte ID; er ist keine fertige Produktdiagnostik mit Dateizeilen.

### 14.3 Referenzierbarkeit und Umbenennung

Der nachfolgende Identitaetsentwurf in Abschnitt 15 untersucht zusaetzlich eine optionale stabile lokale id fuer eingebettete Properties. Das ist eine ausdrueckliche Fortentwicklung der hier geprueften Kurzsyntax, die inline id noch abweist. Die bisherigen Proben bleiben unveraendert und belegen diese neue Ausnahme nicht; die neue begrenzte Probe in Abschnitt 16 untersucht sie separat.

Eine eingebettete Property ist weiterhin ueber ihre abgeleitete ID referenzierbar. Sie kann von einem weiteren Kind ohne Kopie wiederverwendet werden, wenn dessen Feldname derselbe bleibt. In der kompakten Form lautet ein solcher Ausschnitt:

```yaml
spec:
  properties:
    sender: connector/v1/Exchange/sender
```

Das Wiederverwenden eines gemeinsamen Vertrags ist eine ausdrueckliche Autorenentscheidung, keine automatische Gleichsetzung aller gleichnamigen Felder. Fuer eine dauerhaft gemeinsam gepflegte Property kann ihre ausgeschriebene Definition an einer eigenen stabilen ID stehen. Ein Verschieben oder Umbenennen einer eingebetteten Property aendert dagegen ihre abgeleitete ID und muss bei bestehenden Referenzen als Modellaenderung behandelt werden. Es gibt kein stilles Auffinden ueber alte Namen.

Die konkrete Definition list-documents aendert sich durch die Kurzform nicht. Sie referenziert weiterhin den Kind Exchange und setzt sender, receiver, command, Kanaele und Wartewerte. Sie muss weder die Propertydefinitionen noch deren bodies wiederholen.

### 14.4 Ausgefuehrte begrenzte Probe

Ein temporaerer Normalisierungsprototyp hat die kompakte Datei eingelesen, ihre zwanzig eingebetteten Properties ausgeschrieben und das Ergebnis mit der bisherigen ausgeschriebenen Connector-Datei verglichen. Die einundzwanzig direkt geschriebenen Definitionen ergeben nach Normalisierung dieselben einundvierzig Projektdefinitionen. Zusammen mit dem unveraenderten festen Anfang bleiben es einundfuenfzig Definitionen und einhundertdreizehn Kanten.

**Ergebnis:** Definitioneninhalte, Modelldigest und Kanten sind zwischen ausgeschriebener und kompakter Eingabe identisch. Neun ungueltige Autorenvarianten wurden abgewiesen: zusaetzliche inline id, kind oder name, fehlender body, ungueltiger Name, ungueltige Form, Kollision einer erzeugten ID, widerspruechlicher Name bei Propertywiederverwendung und doppelte YAML-Schluessel. Wiederverwendung ohne Propertykopie, geschachtelte Objektfelder und ein unveraendertes fachliches Datenfeld namens properties wurden ebenfalls gegengeprueft.

Der [kompakte Pruefbericht](bootstrap/compact-probe.json) bindet die drei Eingabedateien und beide temporaeren Prototypen ueber SHA-256. Die alte [Bootstrap-Probe](bootstrap/probe.json) und ihre Eingangsdaten bleiben unveraendert. Diese neue Probe belegt die begrenzte Normalisierung gegen denselben separaten Entwurfscompiler. Sie ist keine bereits implementierte Produktunterstuetzung und prueft weder semantische Prosa noch die Connector-Laufzeit.

## 15. Identitaeten, Imports, Versionen und Umbenennungen

**Nachfolgende Nutzerentscheidung:** Der hier entworfene und in Abschnitt 16 durchgespielte Identitätsvertrag wird gewählt. Seine aktuelle konsolidierte Fassung steht im [Sprachvertrag](language-contract.md#7-identitäten-und-referenzen). Dieser Abschnitt bleibt die frühere Ausarbeitung; noch offene lexikalische und produktive Details werden durch die Auswahl nicht als implementiert behandelt.

Auftrag des Nutzers: Kinds, konkrete Definitionen und eingebettete Properties auch mit Imports, Versionen und Umbenennungen eindeutig adressieren. Der folgende Identitaetsvertrag ist ein zusammenhaengender Vorschlag. Gewaehlt bleiben die einheitliche Grundform, Kind und Property, body sowie die kompakte Darstellung; die neue Identitaetsnotation, ein optionales title und die nachfolgende Importpolitik werden durch diesen Entwurf nicht als bereits angenommen behandelt.

### 15.1 Drei verschiedene Fragen

| Frage | Vorgeschlagene Antwort |
| --- | --- |
| Welcher Gegenstand ist gemeint? | Stabile logische Identitaet aus Paket-ID und definitionslokaler id. Der Kind ist kein Teil dieser Identitaet. |
| Wie heisst er fuer Leser? | Optionaler Anzeigename title; sonst kann die id angezeigt werden. Der Anzeigename ist kein Referenzziel. |
| Welche Fassung gilt gerade? | Der gebundene Modellstand legt die exakten Paketversionen und Inhalte fest. |

Beispiel einer logischen Identitaet: (example/connector-project, Exchange). Sie kann ueber mehrere Modellstaende bestehen bleiben. Die Adresse einer bestimmten Fassung benoetigt zusaetzlich deren Modell- beziehungsweise Paketbindung. Eine Referenz auf eine dauerhafte Identitaet allein verspricht keine unveraenderte Bedeutung ueber Versionswechsel hinweg.

IDs duerfen lesbar sein. Eine einmal vergebene id wird jedoch nicht bei jeder sprachlichen Umbenennung angepasst und nicht absichtlich fuer einen anderen Gegenstand wiederverwendet. Universell vorgeschriebene UUIDs sind fuer diesen Vorschlag nicht erforderlich. Dass eine inhaltliche Aenderung dieselbe Sache fortentwickelt oder einen neuen Gegenstand schafft, ist teilweise eine fachliche Entscheidung, keine rein syntaktische Pruefung.

### 15.2 Paket und Importkontext

Ein Paket liefert einen zusammengehoerigen Satz von Definitionen mit einer stabilen Paket-ID. Es kann Projektdefinitionen oder Bibliotheksvokabular enthalten. Paket- und Importangaben sind die Auswahl-/Ladehuelle um die einheitlichen Definitionen, keine neuen eingebauten fachlichen Kinds.

Ein beispielhafter Projektkopf koennte lauten:

```yaml
package: example/connector-project
imports:
  core:
    package: markitect/core
    version: "1.0.0"
  architecture:
    package: markitect/architecture
    version: "1.2.0"
```

Diese Paketnamen und Versionswerte sind Designbeispiele, keine Behauptung verfuegbarer veroeffentlichter Bibliotheken. core und architecture sind nur lokale Importkuerzel. Sie gehoeren nicht zur dauerhaften Identitaet der importierten Definitionen. Die im Prototyp benutzten IDs wie connector/v1/Exchange bleiben der nachgewiesene vorherige Stand; die Trennung von Paket, lokaler Identitaet und Versionsbindung ist eine vorgeschlagene Weiterentwicklung.

Das Projekt deklariert seine eigenen Definitionen in example/connector-project. Innerhalb einer Bibliothek gelten deren eigener Paket- und Importkontext. Importkuerzel des Projekts werden nicht in Bibliotheken hineingetragen. Transitive Abhaengigkeiten koennen fuer die Aufloesung eines importierten Vertrags geladen werden; Autoren erhalten dadurch keine implizite globale Namenssuche ueber alle Abhaengigkeiten. Fuer einen eigenen qualifizierten Autorenbezug wird der betreffende Import ausdruecklich benannt.

Der Compiler muss den fest verstandenen Core an dessen tatsaechlich gebundener Paket-/Definitionsidentitaet erkennen. Ein Alias core oder ein lokaler Kind mit dem Namen Kind verleihen keine Sonderbedeutung. Der feste Core-Vertrag darf weiterhin nicht durch einen Projektimport ersetzt werden.

### 15.3 Eine Referenzschreibweise fuer alle Ziele

| Autorenbezug | Aufloesung im Projektbeispiel |
| --- | --- |
| Exchange | Lokale Definition (example/connector-project, Exchange). |
| list-documents | Lokale Definition (example/connector-project, list-documents). |
| Exchange/sender | Lokale Propertydefinition (example/connector-project, Exchange/sender). |
| architecture:Container | Definition (markitect/architecture, Container) aus dem angegebenen Import. |
| core:Kind | Fester Meta-Kind (markitect/core, Kind) aus dem gebundenen Core. |

Der vorgeschlagene Kurzvertrag ist: ein unqualifizierter Bezug sucht ausschliesslich im eigenen Paket; alias:id sucht ausschliesslich im benannten Import. Fehlende Ziele, unbekannte Aliase, doppelte lokale IDs oder doppelte Importkuerzel sind Fehler. Es gibt keine Suche nach Anzeigenamen, keine Auswahl anhand des erwarteten Zieltyps, keine Dateipfadauflosung und keinen Ersatz durch einen aehnlichen Namen.

Lokale IDs sind gross-/kleinschreibungssensitiv und werden exakt verglichen. Der Trenner : ist fuer die Importqualifizierung reserviert; leere Bestandteile und Pfadsegmente wie . oder .. sind in dieser Kurznotation nicht vorgesehen. / kann als expliziter Trenner von Unteridentitaeten verwendet werden, ist aber kein Dateisystempfad. Eine vollstaendige Grammatik und Escape-/Unicodepolitik muss vor einer Implementierung festgelegt werden; die Beispiele erzwingen hier keine stillen Normalisierungen.

Nach der Autorenaufloesung steht im Modell die kanonische Paket-/Definitionsidentitaet. Sie kann fuer Werkzeuge beispielsweise strukturiert als {package: markitect/architecture, id: Container} dargestellt werden. Ein weiteres langes Autorenformat wird dadurch noch nicht zwingend eingefuehrt. Alle nachfolgenden Typpruefungen verwenden die gebundenen Zielidentitaeten, nicht die lokalen Aliasstrings.

kind, Property.target und konkrete reference-Werte benutzen denselben Adressmechanismus. Ihre Typanforderungen unterscheiden sich weiterhin: kind und ein Property.target muessen auf gueltige Kinds zeigen; Exchange.sender verlangt eine Container-Definition. Ein expliziter Koerperlink sollte spaeter dieselbe Zielaufloesung verwenden. Seine genaue Markupsyntax ist weiterhin offen, und ein normaler Worttreffer im body wird nicht zur kanonischen Referenz.

### 15.4 Versionen an den Modellstand binden

**Vorschlag fuer den ersten Umfang:** Pro Paket-ID ist in einem gebundenen Modell genau eine Version zulaessig. Jeder Bibliotheksimport benennt eine genaue Version. Keine Versionbereiche, kein latest und keine Auswahl des zuerst gefundenen Pakets. Falls zwei direkte oder transitive Imports fuer dieselbe Paket-ID verschiedene Versionen benoetigen, wird ein Konflikt gemeldet.

Die gewaehlte Version wird zusammen mit einem Digest des konkreten Paketinhalts gebunden. Eine Versionsbezeichnung allein reicht nicht als Inhaltsnachweis. Gleiche Paket-ID und Version mit abweichenden Bytes duerfen einen bereits gebundenen Inhalt nicht still ersetzen. Die Auswahl- beziehungsweise Lockinformation muss auch die transitive Abhaengigkeitsschliessung eindeutig machen. Die genaue Datei-/Installationsform dieser Information bleibt offen; der Strukturcompiler soll dafuer kein Netzwerk oder Paketinstallationsprogramm ausfuehren.

Das lokale Projekt benoetigt fuer jede uncommittete Aenderung keine neue Veroeffentlichungsnummer. Seine festen Eingaben werden durch den jeweiligen Quell-/Modellstand gebunden. Der gemeinsame Modellstand erfasst Projektinhalt, verwendete Core-/Bibliotheksvertraege und die ausgewaehlten Paketbindungen. Referenzen muessen dafuer nicht rekursiv den noch zu berechnenden Modelldigest in ihre eigenen Inhalte einsetzen.

Ein Paketupdate laesst beispielsweise die logische Identitaet Container bestehen, waehlt aber einen neuen Vertrag. Danach werden betroffene Definitionen und ihre Realisierungen gegen den neuen gebundenen Stand betrachtet. Die Versionsnummer allein beweist keine fachliche Kompatibilitaet. Fuer die Bewertung eines Nachweises wird die konkrete Modellbindung benoetigt, nicht bloss die dauerhafte Definitionen-ID.

Die Beschraenkung auf eine Version pro Paket ist eine bewusste Vereinfachung dieses Vorschlags, keine grundsaetzliche Unmoeglichkeit paralleler Versionen. Eine spaetere Mehrversionsloesung braeuchte ausdrueckliche Version-/Instanzqualifizierung und Typvergleiche innerhalb genau dieser Bindungen. Zwei Aliase auf verschiedene Versionen duerfen im ersten Umfang diese Regel nicht umgehen. Verschiedene fachliche Softwareversionen als Projektgegenstaende zu beschreiben, bleibt davon unberuehrt.

### 15.5 Exchange mit stabiler Identitaet und lesbarer Bezeichnung

Innerhalb des obigen Pakets kann ein Kind wie folgt deklariert werden:

```yaml
id: Exchange
title: Request-/Reply-Vertrag
kind: core:Kind
body: >-
  Beschreibt den Austausch eines Auftrags und seiner Antwort.
  Der Empfaenger erhaelt die MessageId in der Antwort.
spec:
  properties:
    sender:
      body: Anwendung in der Auftraggeberrolle.
      type: reference
      target: architecture:Container
      minCount: 1
      maxCount: 1
```

Das ist ein Ausschnitt der vorgeschlagenen neuen Identitaetsform, kein gegen den bestehenden Produktcompiler gepruefter Eingabesatz. title ist ein optionaler Vorschlag fuer die gemeinsame Huelle; es ersetzt keinen body und keine Feldbedeutung. Ein title-Wechsel aendert nicht die Identitaet. Der gespeicherte Modellstand darf die geaenderte Darstellung trotzdem erfassen.

Die Property hat im Normalfall die logische Identitaet (example/connector-project, Exchange/sender). Eine konkrete Definition list-documents im selben Paket setzt kind auf Exchange und sender auf api. Ob die Dateien Exchange.yaml oder routing-model.yaml heissen, aendert diese Identitaeten nicht.

### 15.6 Anzeigename, Feldname und Identitaet umbenennen

**Anzeigename:** Aus dem Titel Request-/Reply-Vertrag darf Auftragsaustausch werden. id Exchange, Typbezuege und Propertyidentitaeten bleiben gleich. Die Autorenbezeichnung muss nicht dauerhaft den historischen ID-Text spiegeln.

**Propertyfeld:** Aus sender wird requester. Das ist eine Aenderung des Typvertrags, weil konkrete specs nun den neuen Schluessel verwenden muessen. Soll dieselbe Propertyidentitaet erhalten bleiben, schlage ich eine optionale stabile lokale id in der eingebetteten Form vor:

```yaml
spec:
  properties:
    requester:
      id: sender
      body: Anwendung in der Auftraggeberrolle.
      type: reference
      target: architecture:Container
      minCount: 1
      maxCount: 1
```

Hier ist sender ein stabiler lokaler Identitaetsbaustein, keine Paketadresse und kein alternativer Feldname. Die Normalisierung erzeugt weiter id Exchange/sender und kind core:Property, setzt aber spec.name auf requester. body und Vertragsfelder werden wie bisher uebernommen. Ohne ausdrueckliche id ist der Baustein weiterhin der Zuordnungsschluessel. id darf keine fremde Paketadresse oder Besitzerhierarchie einschleusen; die genaue Segmentgrammatik ist noch festzulegen.

**Das erweitert den vorherigen Kurzsyntaxvorschlag bewusst.** Die Probe aus Abschnitt 14 weist inline id noch ab. Sie bleibt ein unveraenderter Nachweis der damaligen Syntax und belegt diese Erweiterung nicht. Eine zukuenftige Normalisierung muss die optionale id pruefen und Kollisionen abweisen. kind und ein separates name werden weiterhin nicht frei in die Kurzform hineingeschrieben.

Die stabile Propertyidentitaet bewahrt Bezuege auf den Feldvertrag. Sie migriert nicht automatisch bestehende konkrete specs: sender muss dort zu requester werden. Auch relevante Prosa und Integrationserwartungen sind auf die Vertragsaenderung zu pruefen. Reine Textvorkommen sind keine vollstaendig vom Compiler erfassten Referenzen.

**Echte ID-Aenderung:** Soll auch Exchange selbst eine neue id erhalten, ist das eine ausdrueckliche Identitaetsmigration. Bekannte kind-, target-, Wertreferenzen und explizite Koerperlinks muessen angepasst werden. Der Compiler meldet stehengebliebene alte Ziele als Fehler. Eine dauerhafte automatische Weiterleitung oder versteckte Namenshistorie wird fuer den minimalen Core nicht vorgeschlagen.

### 15.7 Verschieben, Imports umbenennen und Bibliotheken weiterentwickeln

| Aenderung | Erwartetes Verhalten |
| --- | --- |
| Definition in eine andere Datei desselben Pakets verschieben | Identitaet und Referenzen bleiben bei gleichem Paket und id bestehen. Die Ladeauswahl muss die Datei weiterhin erfassen. |
| Importkuerzel architecture zu arch aendern | Autorenbezuege dieses Pakets werden angepasst; die kanonischen Zielidentitaeten bleiben gleich. Andere Pakete behalten ihre eigenen Importkuerzel. |
| Eingebettete Property ausschreiben | Im selben Paket mit derselben vollstaendigen lokalen id weiterfuehren, etwa Exchange/sender. Sie bleibt derselbe Gegenstand. |
| Property unter einen anderen Kind verschieben | Standardableitung wuerde die ID aendern. Fuer Identitaetserhalt die bisherige Property ausschreiben und ausdruecklich wiederverwenden; keine heimliche neue Besitzeradresse erfinden. |
| Definition in ein anderes Paket verschieben | Die Paket-ID ist Teil der Identitaet. Das ist eine ausdrueckliche Migration oder eine neue Definition, kein blosser Dateiumzug. |
| Zwei Pakete liefern jeweils eine id Container | Kein Konflikt: Die Paket-IDs unterscheiden die Ziele. Lokale Aliasqualifizierung macht den Autorenbezug eindeutig. |
| Bibliotheksversion aktualisieren | Geaenderte Inhalte neu binden und gegen den neuen Vertrag pruefen. Alte veroeffentlichte Staende und darauf bezogene Nachweise bleiben gebunden. |
| Definition loeschen und unter aehnlichem Namen neu anlegen | Bestehende Referenzen brechen; keine automatische Zuordnung anhand des Namens. |

Ein veroeffentlichter Bibliotheksstand wird nicht nachtraeglich fuer eine Umbenennung umgeschrieben. Eine neue Fassung kann IDs beibehalten, ihren Anzeigenamen aendern oder einen ausdruecklichen Migrationsschritt benoetigen. Ob alte Leser weiterhin funktionieren, haengt vom erhaltenen Vertrag ab. Der Core leitet daraus weder Entwicklungsprozesse noch Freigaberechte ab.

### 15.8 Was der Core beziehungsweise der Loader pruefen kann

Loader und Importauswahl stellen einen eindeutigen Eingabesatz samt Paketbindungen her. Der strukturelle Core prueft auf dieser Grundlage eindeutige Identitaeten, exakte Referenzziele, passende Kinds, gueltige abgeleitete Property-IDs und widerspruchsfreie Feldnamen. Alle benoetigten Vertragstexte bleiben in der Modellbindung.

Dass zwei frei formulierte Beschreibungen denselben Gegenstand meinen, eine Umbenennung fachlich harmlos ist oder eine neue Version die Absicht bewahrt, bleibt Gegenstand von Review und geeigneten Nachweisen. Keine stabile ID ist ein semantischer Kompatibilitaetsbeweis.

**Empfohlener erster Vertrag:** Paket-ID plus stabile lokale ID; lokale und aliasqualifizierte Referenzen; ein exakt gebundener Inhalt pro Paket im Modell; optionaler Anzeigename; standardmaessig abgeleitete und bei Bedarf ausdruecklich stabilisierte Propertyidentitaeten. Keine automatische Namenssuche, Versionswahl oder Weiterleitung. Der Vorschlag bleibt zur Besprechung; der folgende Abschnitt 16 prueft seine Aufloesung und Bindung inzwischen in einem eigenstaendigen begrenzten Prototyp. Ein produktiver Importloader ist damit nicht implementiert.

## 16. Identitaetsdurchgang mit Import, Update und Feldumbenennung

**Nachfolgende Nutzerentscheidung:** Der Identitätsvertrag dieses Durchgangs wird festgehalten und mit der gewählten Grundform als [zusammenhängender Sprachvertrag](language-contract.md) konsolidiert. Die folgenden Abschnitte dokumentieren den geprüften Entwurfsstand und dessen Grenzen.

Auftrag des Nutzers: Den Identitaetsvertrag am Exchange mit importiertem Vokabular, Versionsupdate und sender/requester vollstaendig durchspielen. Die Daten liegen unter identities/ und sind ein geschlossenes synthetisches Designbeispiel. Die Paketnamen behaupten keine existierenden veroeffentlichten Bibliotheken. Die gewaehlte Grundrichtung bleibt erhalten; die konkrete Paket-/Versionspolitik und die optionale stabile inline id bleiben untersuchte Vorschlaege.

### 16.1 Getrennte Paketverantwortung

| Paket | Inhalt |
| --- | --- |
| markitect/core, Version 1.0.0 | Fester Anfang aus Kind, Property und ihren acht Propertydefinitionen. |
| example/connector-vocabulary | Importierbares Vokabular mit den acht Kinds und zwanzig Properties des Connector-Beispiels, darunter Exchange. |
| example/connector-project | Die dreizehn konkreten Projektdefinitionen, darunter api, onprem, Kanaele, Command und list-documents. |

Der [Core-Artefakt](identities/core-1.0.0.yaml) besitzt einen bekannten Inhalt. Das Vokabular importiert core fuer seine Typdefinitionen. Das Projekt importiert core und das Vokabular unter connector. Die wiederholte Abhaengigkeit auf dieselbe Core-Version wird einmal gebunden; der Anfang wird nicht mehrfach instanziiert.

### 16.2 Stand A: Vokabular 1.0.0

Der Projektkopf in [project-a.yaml](identities/project-a.yaml) lautet:

```yaml
package: example/connector-project
imports:
  core:
    package: markitect/core
    version: "1.0.0"
  connector:
    package: example/connector-vocabulary
    version: "1.0.0"
```

Die [Vokabulardatei 1.0.0](identities/vocabulary-1.0.0.yaml) importiert ihrerseits core und beschreibt Exchange, hier auf die Sender-Property gekuerzt:

```yaml
id: Exchange
kind: core:Kind
body: >-
  Beschreibt den Request-/Reply-Vertrag zwischen sender und receiver.
  Antworten behalten die MessageId des Requests.
spec:
  properties:
    sender:
      body: Anwendung in der Auftraggeberrolle.
      type: reference
      target: Container
      minCount: 1
      maxCount: 1
```

Container wird in dieser Vokabulardatei lokal aufgeloest. Das Projekt kuerzt nicht rueckwirkend die Importnamen innerhalb der Bibliothek um. Die konkrete Definition im Projekt ist dagegen von Kind connector:Exchange; sie setzt sender auf api, receiver auf onprem, command auf list-documents-command, requestChannel auf requests, replyChannel auf replies und die Werte auf 30 Sekunden beziehungsweise zwei Versuche.

Die kanonischen Zielidentitaeten lauten:

| Autorenbezug und Kontext | Kanonische Identitaet |
| --- | --- |
| core:Kind im Vokabular | (markitect/core, Kind) |
| connector:Exchange im Projekt | (example/connector-vocabulary, Exchange) |
| connector:Exchange/sender im Projekt | (example/connector-vocabulary, Exchange/sender) |
| Container als Propertyziel im Vokabular | (example/connector-vocabulary, Container) |
| api als sender-Wert im Projekt | (example/connector-project, api) |
| list-documents im Projekt | (example/connector-project, list-documents) |

Die ersten beiden Paketfassungen sind mit ihren tatsaechlichen SHA-256-Digests in [locks.yaml, Bindung a](identities/locks.yaml) festgehalten. Der Projektinhalt und diese gebundenen Bibliotheksinhalte bilden den festen Stand A. Die konkrete Aufloesungskette ist:

```text
Projekt.list-documents.kind -> Vokabular.Exchange
Vokabular.Exchange.kind -> Core.Kind
Vokabular.Exchange.properties -> Vokabular.Exchange/sender
Vokabular.Exchange/sender.target -> Vokabular.Container
Projekt.list-documents.sender -> Projekt.api
Projekt.api.kind -> Vokabular.Container
```

Alle sieben Exchange-Properties und saemtliche referenzierten Definitionen sind in den Dateien vorhanden. Die Darstellung oben ist nur der Ausschnitt fuer die Identitaetspruefung. Stand A wird strukturell akzeptiert.

### 16.3 Stand B: Update auf 1.1.0 bei unveraenderten Projektdefinitionen

Die [Vokabulardatei 1.1.0](identities/vocabulary-1.1.0.yaml) behaelt alle IDs und Propertyvertraege. Sie ergaenzt im Exchange-body eine neue allgemeine Pflicht:

> Der Sender muss die Wartefrist jedes Versuchs anhand monotoner Zeitmessung bestimmen.

[project-b.yaml](identities/project-b.yaml) aendert nur den Versionsimport auf 1.1.0. Die zugehoerige Inhaltsbindung steht unter b in locks.yaml. Die konkreten Projektdefinitionen bleiben unveraendert und werden strukturell weiterhin akzeptiert. Auch die Referenzkanten bleiben gleich.

Der gebundene Modellstand aendert sich trotzdem: Der Kind-Vertrag hat einen anderen body, das Paket besitzt einen anderen Inhalt und das Projekt waehlt diesen neuen Inhalt ausdruecklich. Eine alte Inhaltsbindung fuer 1.0.0 darf nicht als Bindung fuer 1.1.0 weiterverwendet werden.

**Die Strukturpruefung bestaetigt die neue Zeitmessung nicht.** Umsetzung und geeignete Nachweise muessen gegen die neue Pflicht betrachtet werden. Die Versionsnummer ist hier eine genaue Beispielbezeichnung, kein automatisch bewiesenes Kompatibilitaetsversprechen. Die Probe aendert oder prueft keinen laufenden Connector.

### 16.4 Stand C: Version 2.0.0 benennt sender zu requester um

Die [Vokabulardatei 2.0.0](identities/vocabulary-2.0.0.yaml) uebernimmt die Pflicht aus 1.1.0 und benennt den Exchange-Feldschluessel um. Sie behaelt die Propertyidentitaet ausdruecklich bei:

```yaml
spec:
  properties:
    requester:
      id: sender
      body: Anwendung in der Auftraggeberrolle.
      type: reference
      target: Container
      minCount: 1
      maxCount: 1
```

Bei der Normalisierung wird daraus weiter die Definition (example/connector-vocabulary, Exchange/sender). Deren spec.name ist jetzt requester. Es entstehen weder eine zweite Property namens requester noch eine automatische Weiterleitung von alten Datenfeldern.

Wird ausschliesslich der Import aktualisiert, scheitert der [veraltete Projektstand](identities/project-c-stale.yaml): sender ist in der neuen Exchange-spec nicht mehr erlaubt und requester ist erforderlich. Der Prototyp meldet zuerst spec.unknown-property fuer den alten Schluessel. Die stabile Property-ID macht den alten Feldschluessel nicht weiter gueltig.

Der [migrierte Projektstand C](identities/project-c.yaml) setzt den neuen Schluessel; auf die relevanten Werte gekuerzt:

```yaml
id: list-documents
kind: connector:Exchange
body: >-
  Die API fordert Dokumentmetadaten an und registriert
  den wartenden Austausch vor der Publikation.
spec:
  requester: api
  receiver: onprem
  command: list-documents-command
  requestChannel: requests
  replyChannel: replies
  timeoutSeconds: 30
  maxAttempts: 2
```

Mit Bindung c wird dieser Stand strukturell akzeptiert. Ein ausdruecklicher Bezug connector:Exchange/sender findet in A, B und C dieselbe logische Propertyidentitaet. Die Wertebeziehung api bleibt dieselbe; ihr Feldpfad im konkreten Exchange aendert sich von spec.sender zu spec.requester. Die allgemeine Beschreibung wird in 2.0.0 auf requester angepasst. Ob jede weitere fachliche Prosa sinnvoll angepasst ist, kann reine Referenzaufloesung nicht entscheiden.

### 16.5 Was gleich bleibt und was sich aendert

| Merkmal | A | B | C, migriert |
| --- | --- | --- | --- |
| Gebundenes Vokabular | 1.0.0 | 1.1.0 | 2.0.0 |
| Logische Exchange-Identitaet | example/connector-vocabulary + Exchange | Gleich | Gleich |
| Logische Propertyidentitaet | example/connector-vocabulary + Exchange/sender | Gleich | Gleich |
| Konkrete Identitaet | example/connector-project + list-documents | Gleich | Gleich |
| Feldschluessel | sender | sender | requester |
| Zeitmessungspflicht | Noch nicht allgemein ergaenzt | Im Kind-body ergaenzt | Weiter geltend |
| Strukturelles Ergebnis | Akzeptiert | Akzeptiert | Nach Feldmigration akzeptiert |
| Gebundener Modellstand | A | Neu | Erneut neu |

Alle drei gueltigen Staende enthalten dieselben einundfuenfzig logischen Identitaeten und jeweils einhundertdreizehn Kanten. Die gleiche Anzahl bedeutet weder gleiche Inhalte noch gleiche Pflichten. Die konkreten Paketdigests und die drei unterschiedlichen Modelldigests stehen im [Pruefbericht](identities/probe.json).

### 16.6 Gegenprobe: stabile ID bei der Umbenennung vergessen

Wird requester ohne id: sender deklariert, entsteht nach der Standardableitung die neue Propertyidentitaet Exchange/requester. Exchange/sender ist dann nicht mehr vorhanden. Der bereits migrierte konkrete Exchange kann trotzdem strukturell gueltig sein, wenn er keine alte Propertyadresse referenziert.

Das ist wichtig: Der Compiler kann aus dem Feldnamenwechsel allein nicht wissen, dass eine identitaetserhaltende Umbenennung beabsichtigt war. Der Vergleich zeigt dann geloeschte und hinzugefuegte Identitaeten; tatsaechliche alte Propertyreferenzen waeren unaufloesbar. Die ausdrueckliche stabile id unterscheidet die gewuenschte Umbenennung von der Anlage eines neuen Feldvertrags.

### 16.7 Ausgefuehrte Aufloesungs- und Bindungsprobe

Ein temporaerer eigenstaendiger Paket-/Referenzprototyp hat alle drei gueltigen Staende mit ihren jeweiligen Inhaltsbindungen aufgeloest. Er hat die aufgeloesten Paket-/ID-Paare als eindeutige JSON-Paare an den frueheren unabhaengigen Strukturprototyp weitergegeben. Der feste Anfang wurde auf die bekannten Core-Identitaeten gebunden; die Fachvokabulare erhielten dadurch keine besonderen Operatoren.

Die Probe hat bestaetigt:

- Alle drei gueltigen Staende besitzen dieselben logischen Identitaeten, aber unterschiedliche gebundene Modelldigests.
- Stand B aendert den Kind-body ohne Aenderung der Felder oder Referenzkanten.
- Die Property-ID bleibt bei sender/requester erhalten, wenn die ausdrueckliche stabile id gesetzt ist.
- Ein Wechsel des lokalen Importkuerzels connector zu wire mit Anpassung der Autorenbezuege behaelt fuer dieses Beispiel die aufgeloesten Inhalte und die Bindung bei.
- Ohne stabile inline id ist der alte Propertygegenstand weg und ein neuer vorhanden; das kann ohne alte Referenzen trotzdem strukturell gueltig sein.

Neun negative Varianten wurden abgewiesen: altes Projektfeld unter Version 2.0.0, neuer Import mit alter Lockbindung, nicht bereitgestellte genaue Version, zwei Versionen derselben Paket-ID, geaenderte Bytes unter derselben gebundenen Version, unbekannter Alias, unqualifizierte Suche nach einem nur importiert vorhandenen Kind, Verwendung eines Projektaliases innerhalb der Bibliothek sowie eine Typdefinition anstelle einer Containerinstanz als konkreter requester-Wert.

Der Bericht bindet neun YAML-Dateien, den temporaeren Resolver und den verwendeten separaten Strukturprototyp ueber SHA-256. Die bisherigen Compiler-/Normalisierungsproben und ihre Eingaben wurden nicht veraendert. Diese neue Probe ist kein Produktloader, keine Paketregistry, kein Signatur-/Vertrauensnachweis, kein allgemeiner Koerperlinkparser und keine ausgefuehrte Realisierungspruefung. Der optionale Anzeigename title wird fuer diesen Durchgang nicht benoetigt und bleibt eine separate Formatfrage.

### 16.8 Ergebnis fuer die weitere Sprachentscheidung

Der vorgeschlagene Identitaetsvertrag traegt dieses importierte, aktualisierte und umbenannte Exchange-Beispiel. Dauerhafte Identitaet, gebundener Vertragsstand und aktueller Feldname lassen sich unterscheiden. Die notwendige Feldmigration wird sichtbar, ohne alle Bezuege auf die Property zu zerbrechen.

Die Versions-/Importpolitik und die optionale stabile inline id sind damit konkret und begrenzt untersucht. Ihre Uebernahme als allgemeiner Sprachvertrag, eine endgueltige Identitaetsgrammatik sowie produktive Lade-, Diagnose- und Migrationsverfahren bleiben zu entscheiden. Keine der erfolgreichen Strukturproben belegt die semantische Kompatibilitaet eines beliebigen Updates.

Zurueck zum [gemeinsamen Designstand](../consolidation.md) und zur [Begriffsuebersicht](../README.md).

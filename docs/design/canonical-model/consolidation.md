# Gemeinsamer Designstand für Markitect

Stand: 15. Oktober 2026. Auftrag: aktuellen Main und die beiden Modellkonzepte konsolidieren und die offenen Widersprüche gemeinsam entscheiden. Diese Seite ist der Einstieg in das weitere Designgespräch. Empfehlungen sind Vorschläge; eine Übernahme von Dokumenten ist keine Entscheidung über ihre offenen Inhalte.

**Aktuelle Sprachgrundlage:** Der Nutzer hat die einheitliche Grundform mit Kind und Property, body, kompakte Properties, den durchgespielten Identitätsvertrag und anschließend die Körperlinks gewählt. Der [zusammenhängende Core-Sprachvertrag](minimal-language/language-contract.md) besitzt diesen aktuellen Designstand. Diese Seite hält dessen Herkunft und die übrigen offenen Produktfragen fest. Die nachfolgenden Gesprächsnotizen enthalten auch inzwischen überholte Vorschläge; die gewählten Regeln bedeuten keine bereits produktiv implementierte neue Sprache.

**Gewählte Körperlinks:** [Abschnitt 4 des Sprachvertrags](minimal-language/language-contract.md#41-ein-linkformat-für-alle-definitionen) hält `[[Ziel]]` und `[[Ziel|Anzeigetext]]` fest. Kinds, konkrete Definitionen und normalisierte Properties verwenden dieselbe Paket-/ID-Auflösung. Ein Link erzeugt einen ausdrücklichen Bezug; der umgebende Satz trägt die fachliche Aussage. Propertydeklaration und konkreter Propertywert bleiben unterschieden. Der Nutzer hat diesen Vorschlag gewählt; vollständige lexikalische Formatdetails und die Produktimplementierung bleiben offen.

**Nächste Ausarbeitung: Kontext und Auswirkungen.** [Abschnitt 11 des Sprachvertrags](minimal-language/language-contract.md#111-vorschlag-kontext-als-begründete-zusammenstellung) schlägt eine begründete Kontextbildung, strukturell berechnete Änderungskandidaten und semantische Beurteilung der Geltung vor. Am Exchange werden unter anderem eine neue Pflicht im Kind-body, eine geänderte Wartefrist und die Topicbildung geprüft. Unverlinkte globale Prosa, entfernte Bezüge und fehlende Realisierungszuordnungen bleiben ausdrücklich zu behandeln. Das ist ein Verfahrensvorschlag zum neuen Auftrag, keine neue Nutzerentscheidung oder bereits implementierte generische Impactberechnung.

**Konkretisierung des semantischen Index:** [Abschnitt 11.7 bis 11.9](minimal-language/language-contract.md#117-semantischer-index-quelle-auslegung-und-konkrete-geltung) trennt auf weiteren Nutzerauftrag Originalaussage, allgemeine Auslegung, konkrete Geltung und untersuchte Abdeckung. Die Zeitmessungspflicht wird über den Exchange und dessen stabile Auftraggeber-Property auf api bezogen. Strukturprüfung, unabhängiges semantisches Review und Abdeckungsprüfung bleiben unterscheidbar. Änderungen an Quellen, Belegungen, Kandidatenmengen und möglichen Gegenregeln verlangen jeweils eine begründete Aktualitätsprüfung; die Tabellen beschreiben den Vorschlag und keinen ausgeführten semantischen Prüflauf.

**Durchgang zur Indexaktualisierung:** [Abschnitt 11.10](minimal-language/language-contract.md#1110-durchgang-zweiter-exchange-und-wechsel-seines-requester) fügt auf Nutzerauftrag einen zweiten Exchange hinzu und wechselt danach dessen requester von api zu onprem. Er unterscheidet die Wiederverwendung einer allgemeinen Auslegung, die Übernahme unveränderter konkreter Befunde, neue Rollenbelegungen und die erneuerte Mengen-/Rollenabdeckung. Ein alter positiver Prüfzustand wird nur mit begründeter Aktualitätsprüfung an den neuen Stand übernommen. Eigene rollenunabhängige Kanalverträge sind ausdrücklich vorausgesetzt; der Durchgang ist ein Designbeispiel, kein Implementierungs- oder Laufzeitnachweis.

**Abhängigkeiten der Wiederverwendung:** [Abschnitt 11.11](minimal-language/language-contract.md#1111-wiederverwendbarkeit-gelesene-inhalte-auswahlräume-und-neue-einflüsse) konkretisiert auf Nutzerauftrag direkte Inhalts-/Strukturabhängigkeiten, Auswahlräume einschließlich bisheriger Nichttreffer, negative Voraussetzungen, abgeleitete Vorbefunde und Verarbeitungsbindungen. Eine vollständige Prüfung der Änderungen im gebundenen Modellumfang ergänzt diese Eintragsabhängigkeiten, damit neue globale Aussagen ohne alte Links nicht aus der Aktualitätsprüfung fallen. Der Vorschlag trennt einen weiterhin richtigen alten Befund von der Vollständigkeit des aktuellen Kontexts und beansprucht keine deterministische Garantie der semantischen Interpretation.

**Globale Zusatzpflicht und Ausnahme:** [Abschnitt 11.12](minimal-language/language-contract.md#1112-durchgang-globale-zusatzpflicht-und-ausdrückliche-ausnahme) führt den Indexdurchgang mit einer neuen Pflicht zur Uhrquellenprotokollierung und ihrer später ausdrücklich begrenzten Geltung für check-status fort. Er unterscheidet neue Anwendungen, überholte Quellauslegungen, weiterhin gültige Zeitmessungspflichten und erneuerte Abdeckung einschließlich ausgeschlossener Fälle. Eine nur lokal gegen die unveränderte globale Pflicht behauptete Ausnahme bleibt als Konflikt zu klären. Auch dieser Durchgang ist eine Designgegenprobe mit angenommenen erfolgreichen Reviews, kein ausgeführter semantischer Nachweis.

**Datenvertrag des semantischen Index:** [Abschnitt 11.13](minimal-language/language-contract.md#1113-vorschlag-eines-datenvertrags-für-den-semantischen-index) konkretisiert auf Nutzerauftrag unveränderliche Auslegungs-, Anwendungs-, Abdeckungs- und Einflussbefunde, geschlossene Abhängigkeitsvarianten, genaue Quellen-/Modellbindungen sowie eigenständige Reviews und Aktualitätsbindungen. Fachliche Prüfung und Aktualität werden getrennt abgeleitet. Ein Exchange-Datensatz zeigt die Zeitmessungspflicht von onprem bei check-status und ihre Übernahme an einen Folgestand. Die Formen sind ein Datenmodellentwurf mit symbolischen Beispielreferenzen, noch kein produktives Transportformat oder ausgeführter Indexvalidator.

**Ausgefüllter Übergang M3 → M4 und Sprachstand:** [Abschnitt 11.14](minimal-language/language-contract.md#1114-vollständiges-datensatzprotokoll-für-m3--m4) ordnet auf Nutzerauftrag die betroffenen Recordfassungen, angenommenen Reviews und neuen Bindings vollständig im betrachteten Indexausschnitt zu. Drei Zeitmessungsrecords werden bei ausdrücklich unveränderten direkten Grundlagen übernommen; neue Logauslegung, Anwendungen, Abdeckung und Einflussbefund erzeugen acht Vordergrundrecords und elf Bindings. Hintergrunderschließung und erfolgreiche Reviews sind benannte Voraussetzungen, kein ausgeführter Nachweis. [Abschnitt 12.1](minimal-language/language-contract.md#121-stand-der-definition-des-kanonischen-modells) trennt die weitgehend gewählte Fachsprache von offenen Formatdetails, abgeleiteten Indexverfahren und der noch ausstehenden Produktimplementierung.

## Quellen und Zusammenführung

| Quelle | Fester Eingang | Beitrag |
| --- | --- | --- |
| `main` | `6472d42a93c1be7e542d1fa06ad10a308dd15b8a` | Aktuelle Produktgrundlage, Implementierung, Ursprungserklärung und bestehende Verträge. |
| `docs/product-model-concepts` | `3e5583cb67eb931e830211ce63378d4526191a3c` | [Typisierte Prosa, Sprachentwurf und Bibliotheksmodell](../product-model-refinement/README.md); darin dokumentierte Nutzerentscheidungen und offene Syntaxfragen. |
| `docs/canonical-model-vocabulary` | `3e5ccdbe217b66a5332aec8b913116f90cb2091a` | [Vierzehn Arbeitsbegriffe](README.md), [Beziehungen](relationships.md) und [Tasks als Begriffsbeispiel](task-management-example.md). |
| `docs/task-management-model` | `45c99e82a61aa5f1774b093970f2709a632424e9` | Ergänzende [eigenständige Tasks-Projektdokumentation](../task-management/README.md) und ausdrücklich [historischer Modellierungsversuch](../task-management-model/README.md). |

Die beiden eigentlichen Modellkonzepte sind `product-model-concepts` und `canonical-model-vocabulary`. Tasks ist zusätzliches Vergleichsmaterial. Seine eigenständige Dokumentation bleibt als Projektbeschreibung ohne vorausgesetzte Markitect-Taxonomie lesbar.

Die Arbeitsbranch `docs/design-consolidation` beginnt auf dem oben genannten Main. Aus den drei Dokumentationsbranches werden ausschließlich ihre neuen Designordner übernommen. Ihre Änderungen außerhalb dieser Ordner betreffen nur die gemeinsam neu aufgebaute Designnavigation. Der alte Produktcode des Bibliotheksbranches wird nicht übernommen: Sein gemeinsamer Vorfahr liegt 505 Main-Commits zurück. Das neue Vokabular und der Tasks-Branch haben diesen Main bereits als Vorfahren.

Die Quellen bleiben mit ihrer Herkunft erhalten. Diese Seite führt ihre Aussagen zusammen; neue Nutzerentscheidungen werden ausdrücklich datiert in Definitionen und betroffenen Entwürfen nachgeführt. [Der Main-Abgleich](current-main-comparison.md) nennt die heute tatsächlich untersuchten Implementierungsgrenzen. Der [Vergleich vom 11. Oktober](../product-model-refinement/canonical-model-main-comparison.md) bleibt historische Evidenz für seinen damaligen Stand.

## Gemeinsame Grundlage

Markitect beschreibt die beabsichtigte Projektwelt und lässt deren Repositoryrealisierung koordiniert ausführen. Menschen entscheiden Bedeutung und Freiräume; Strukturprüfung, nachvollziehbarer Impact, begrenzte Zuständigkeiten, unabhängige Reviews und konkrete Checks unterstützen die Realisierung. Das ist die Grundlage der [Vision](../../vision.md) und des [Konzeptregisters](../../concepts/register.md).

Beide Designlinien lassen sich darin zusammenführen: Die Begriffe erklären, **welche Art von Aussage** beschrieben wird. Ein Definitionskörper enthält die individuellen Aussagen. Explizite Beziehungen verbinden Definitionen. Daraus sollen Kontext, Realisierungsaufträge und Prüfaufträge entstehen. Wie genau diese Ableitung funktioniert, ist pro Beziehung zu bestimmen. Ein Wort, ein Link, eine Zusammensetzung und eine Realisierungspflicht sind verschiedene Dinge.

## Arbeitsziel und Wackelkandidaten

Präzisierung des Nutzers vom 15. Oktober 2026: Ausgangspunkt des jüngeren Branches ist eine ausführliche, typisierte Projektdokumentation. Das kanonische Modell soll alles Relevante über das beabsichtigte Projekt ausdrücken können: was sein soll und welche Festlegungen gelten. Das Vokabular wird aus diesem Beschreibungsbedarf entwickelt. Die vorhandene Implementierung und ihr bisheriger Koordinationsablauf bestimmen nicht vorab, welche Begriffe dieses neue Modell enthalten muss.

Das Ziel bleibt, die beabsichtigte Projektwelt zu beschreiben und durch Apply realisieren zu lassen. Zu unterscheiden sind dabei:

| Ebene | Inhalt |
| --- | --- |
| Kanonische Projektbeschreibung | Zweck, Umfang, Fachlichkeit, Verhalten, Regeln, gewünschte Architektur sowie relevante technische und betriebliche Festlegungen. |
| Apply-Verfahren | Interpretation der Beschreibung, Aufteilung der Agentenarbeit, Umsetzung, Prüfung und Integration. Die genaue neue Arbeitsweise ist noch zu entwerfen. |
| Realisierung und Nachweise | Durch die Umsetzung entstandene Artefakte, ihre ermittelte Zuordnung zum Modell sowie ausgeführte Prüfungen und beobachtete Ergebnisse. |

Konkrete entstandene Dateien sind Realisierung. Eine beabsichtigte Verpflichtung wie „eine Bedienungsanleitung muss vorhanden sein“ kann dennoch für die Projektbeschreibung relevant sein. Ob solche Verpflichtungen einen eigenen Artifact-Typ benötigen, bleibt eine eigenständige Designfrage. Entsprechend werden Anforderungen an die Softwarearchitektur von der tatsächlich erzeugten Datei- oder Paketstruktur unterschieden.

Der Nutzer benennt die folgenden Begriffe ausdrücklich als **Wackelkandidaten**. Das bedeutet: Bedarf, Platz im Modell und Abgrenzung sind zu prüfen; keiner dieser Begriffe ist durch diese Einordnung bereits gestrichen. Die nachfolgenden Begründungen halten den Gesprächsstand fest, nicht neue abschließende Typentscheidungen.

| Wackelkandidat | Begründung und offene Frage |
| --- | --- |
| **System** | Project beschreibt bereits den Gesamtzusammenhang, Component kann ein allgemeiner Baustein sein. Zu klären ist, welchen zusätzlichen Ausdruck System ermöglicht: etwa eine eigenständige Lösung mit Außengrenze, mehrere Lösungen in einem Projekt oder relevante externe Systeme. Diese möglichen Aufgaben sind Prüfbeispiele, noch keine bestätigte Rechtfertigung für einen eigenen Typ. |
| **Component gegenüber Module** | Nach der Erweiterung von Component auf logische und verschachtelbare Bausteine überschneiden sich die bisherigen Bedeutungen. Gewünschte Architektur kann zum Soll des Projekts gehören; offen ist, ob dafür zwei Typen nötig sind und welche unterschiedliche Aussage beziehungsweise Relation sie ermöglichen. Die Wahl der allgemeinen Component-Bedeutung entscheidet noch nicht über den endgültigen Bedarf beider Typen. |
| **Artifact** | Das Modell beschreibt, was sein soll; erst die Umsetzung führt zu konkreten Artefakten. Ein Modelltyp für entstandene Dateien würde diese Ebenen vermischen. Zu prüfen bleibt, wie notwendige Realisierungsbeiträge oder Ausgabevorgaben beschrieben werden und ob dafür ein eigener Typ erforderlich ist. |
| **Mandate** | Die bisherige Bedeutung beschreibt Auftrag, Zuständigkeit und Grenzen der ausführenden Agents und liegt damit zunächst bei der Arbeitsweise für Apply. Ein Platz im kanonischen Projektvokabular ist dadurch nicht begründet. Für das Projekt selbst relevante fachliche oder betriebliche Verantwortlichkeiten müssten davon unterschieden werden. |
| **Decision** | Die geltende Festlegung und gegebenenfalls ihre Begründung können zur Projektbeschreibung gehören. Der Entscheidungsverlauf und seine historische Herkunft erfüllen einen anderen Informationszweck. Offen ist, welchen notwendigen Ausdruck eine eigene Decision-Definition zusätzlich zur Beschreibung am Gegenstand der Festlegung ermöglicht. |

Zusätzlich ist **der Bedarf an Prozessbeschreibungen und damit die Rolle von Workflow und Procedure offen**. Fachliche Abläufe oder vorgeschriebene Betriebsverfahren können zum beabsichtigten Projekt gehören. Klassische Entwicklungsabläufe mit Pull Requests, Reviews und ähnlichen Stationen sind kein vorausgesetzter Bestandteil des neuen kanonischen Modells. Die neue Apply-Variante soll solche bisherigen Abläufe mehr oder weniger umgehen beziehungsweise entblockieren können; ihr konkreter Ersatz und ihre Prüf- und Integrationsmechanismen sind hier noch nicht festgelegt. Die Entwicklungsprozesse im Tasks-Begriffsbeispiel bleiben Anschauungsmaterial, keine Anforderung an dieses neue Modell.

**Aktuelle Priorität des Nutzers:** System, Component und Module gemeinsam klären. Ausgangspunkt sind konkrete Aussagen einer ausreichenden Projektbeschreibung. Wir prüfen, welchen eigenständigen Ausdruck und welche Beziehungen jeder Begriff ermöglicht, wo Überschneidungen liegen und was eine Zusammenlegung verlieren würde. Die Klärung von Artifact, Mandate und Decision folgt danach. Es ist noch keine Auswahl der zu behaltenden Typen getroffen.

### Beauftragte Sprachgrundlage statt wachsendem festem Typenkatalog

Die folgenden Absätze dokumentieren den Gesprächsweg und die Status zum jeweiligen Zeitpunkt. Maßgeblich für die aktuelle Auswahl ist der oben verlinkte konsolidierte Sprachvertrag.

Im weiteren Gespraech am 15. Oktober nennt der Nutzer zwei tragfaehige Wege: wenige starke Begriffe fuer beliebige Projekte oder die Moeglichkeit, Begriffe selbst zu definieren. Anschliessend beauftragt er ausdruecklich den [Entwurf einer minimalen Sprachgrundlage mit eigenen Typen und Relationen am Connector-Beispiel](minimal-language/README.md) und erinnert an die fruehere `kind`-Richtung.

Der neue Entwurf untersucht Schema/Vokabular, Kind, Property und Definition als kleine Grundlage. Eine Relation wird zunaechst als benannte Reference-Property am Quell-Kind beschrieben. System, Container, Component und spezielle Kommunikationsbegriffe sind darin selbst deklarierte Vokabulartypen. Die konkrete Syntax, Relationsform, semantische Validierung sowie generischer Kontext und Apply bleiben Entwurfsfragen; der Auftrag ist keine Annahme aller vorgeschlagenen Details. Die vorherigen Begriffsdiskussionen bleiben Anforderungen und Gegenproben fuer ein moegliches Vokabular.

**Weitere Nutzerpraezisierung:** Der Core muss die allgemeinen Ausdrucksmittel vollstaendig bereitstellen, die Markitect braucht. Vokabularbibliotheken sind ausschliesslich eine Importhilfe fuer fertige Begriffe; ein Projekt muss dieselben Kinds und Vertraege auch selbst definieren koennen. Anschliessend beauftragt der Nutzer die Definition des Cores und seiner Bestandteile. Die [Core-Bestandsaufnahme](minimal-language/README.md#21-auftrag-an-den-core-und-rolle-der-bibliotheken) konkretisiert den Vorschlag aus Schema, Kind, Property und Definition mit gemeinsamen Mechanismen fuer Bedeutung, Referenzen, Anforderungen und gebundene Modellstaende. Die konkrete Zusammensetzung und zusaetzlichen Regeln sind weiterhin zu besprechen.

**Nutzerentscheidung im weiteren Gespraech:** requirements und roles werden vorerst als besondere Core-Sprachmittel weggelassen. Der Nutzer beauftragt den [vollstaendigen Exchange-Durchgang](minimal-language/README.md#10-vollstaendiger-exchange-durchgang-mit-dem-minimalen-core) mit verbindlicher Prosa, Properties und Referenzen. Die vorherige strukturierte Requirementvariante bleibt ein zurueckgestellter Vergleich. Die Entscheidung akzeptiert weder automatisch eine neue Dateisyntax noch eine bereits generische Kontext-/Impact-/Apply-Implementierung.

**Weitere Nutzerentscheidung und Untersuchungsauftrag:** Der verbindliche Textkoerper heisst in der angestrebten Autorenform body. Der Nutzer beauftragt ausserdem die Pruefung, ob Kind, Property und konkrete Elemente dieselbe Grundform haben und Kinds selbst Definitionen sein koennen. Die [gemeinsame Grundform](minimal-language/README.md#11-gemeinsame-grundform-und-body) untersucht eine allgemeine Definition mit Kind und Property als festen Meta-Kinds. Diese Vereinheitlichung, eine neue Identitaetsform und ein selbstbeschreibender Core sind Vorschlaege; gewaehlt ist bislang der Feldname body. Die bisherigen Beispieldateien und Pruefberichte mit definition bleiben als nachvollziehbarer vorheriger Stand erhalten.

**Anschliessender Untersuchungsauftrag:** Den festen Anfang mit Kind und Property und die Typaufloesung am Exchange vollstaendig durchspielen. Der [ausgearbeitete Kandidat](minimal-language/README.md#12-fester-anfang-und-typaufloesung-am-exchange) umfasst zwei Meta-Kinds mit acht eigenen Propertydefinitionen und eine eigenstaendige begrenzte Entwurfsprobe. Er zeigt, welche Sprachregeln trotz gemeinsamer Definitionenform fest im Core bleiben. Das ist weiterhin ein Designkandidat, keine Umstellung des Produktcompilers oder Annahme aller vorgeschlagenen Regeln.

**Beauftragter Variantenvergleich:** Der Nutzer laesst echte Einsparungen, neue Komplexitaet und die Eignung beider Core-Varianten vergleichen. Die [Empfehlung](minimal-language/README.md#13-vergleich-und-empfehlung-fuer-markitect) lautet, zunaechst beim expliziten strukturellen Core zu bleiben, body zu verwenden und Typ-/Propertyvertraege in einer gemeinsamen adressierbaren Graphsicht sichtbar zu machen. Die vollstaendige Selbstbeschreibung wird dafuer vorerst nicht empfohlen. Das ist eine Assistentenempfehlung und keine Auswahlentscheidung des Nutzers; beide untersuchten Varianten bleiben dokumentiert.

**Gewaehlte Core-Richtung:** Der Nutzer entscheidet sich ausdruecklich fuer die **einheitliche Grundform mit Kind und Property** und beauftragt eine kompakte Propertydarstellung direkt beim Kind. Diese Entscheidung loest die vorherige Assistentenempfehlung fuer den expliziten Core ab. Kinds, Properties und konkrete Gegenstaende sind in der gewaehlten Richtung Definitionen mit verbindlichem body; Kind und Property sind die festen Grundtypen. requirements und roles bleiben vorerst weg. Die [kompakte Propertyform](minimal-language/README.md#14-gewaehlte-grundform-und-kompakte-properties) normalisiert eingebettete Angaben auf dieselben referenzierbaren Propertydefinitionen. Die konkrete Kurzsyntax, abgeleitete Identitaetsnotation und produktive Implementierung sind dadurch nicht pauschal als abgeschlossen behandelt.

**Naechster Untersuchungsauftrag:** Identitaeten und Referenzen fuer Kinds, konkrete Definitionen und eingebettete Properties auch bei Imports, Versionen und Umbenennungen klaeren. Der [Identitaetsentwurf](minimal-language/README.md#15-identitaeten-imports-versionen-und-umbenennungen) schlaegt Paket-ID plus stabile lokale ID, lokale Importkuerzel, genaue Inhalts-/Versionsbindung und eine optionale stabile lokale Property-ID fuer Feldumbenennungen vor. title als Anzeigename, eine Version pro Paket und die neue Kurznotation sind Vorschlaege zur Besprechung. Die vorherigen Proben und ihre abgeleiteten IDs bleiben unveraenderte Nachweise ihres damaligen Vertrags; sie pruefen diesen neuen Import-/Identitaetsentwurf nicht.

**Beauftragter Identitaetsdurchgang:** Importiertes Vokabular, ein Versionsupdate und sender/requester werden [am vollstaendigen Connector-Datensatz durchgespielt](minimal-language/README.md#16-identitaetsdurchgang-mit-import-update-und-feldumbenennung). Drei gueltige Staende und ein veralteter Zwischenstand zeigen die unveraenderten logischen Identitaeten, neue Inhaltsbindungen und die notwendige Feldmigration. Ein eigenstaendiger begrenzter Prototyp prueft Paketkontexte, Inhaltsbindungen und die optionale stabile inline id. Das erweitert die Evidenz des Entwurfs; es ist keine Produktimplementierung oder pauschale Annahme aller vorgeschlagenen Identitaets-/Importregeln.

**Nutzerentscheidung zur Konsolidierung:** „Dann halten wir den Identitätsvertrag so fest.“ Der Nutzer beauftragt anschließend, Grundform, kompakte Properties, Identitäten und Referenzen als zusammenhängenden Sprachvertrag zu konsolidieren. Die [aktuelle Fassung](minimal-language/language-contract.md) hält damit auch stabile Paket-/lokale Identitäten, genaue Inhalts-/Versionsbindung, lokale Importkürzel und die identitätserhaltende Feldumbenennung fest. Lexikalische Detailgrammatik, Körperlinks, Paketdatei-/Loaderausarbeitung und generische Produktanbindung bleiben als offene Arbeit ausgewiesen. Die bisherigen Proben und historischen Eingaben werden nicht nachträglich verändert.

Wir führen drei Status getrennt:

- **Main-Vertrag:** Verhalten und Format, das der feste aktuelle Produktstand tatsächlich unterstützt.
- **Im Bibliotheksgespräch vereinbarte Designrichtung:** unter anderem `Mandate`, nominale Fach-Kinds, eigenständige Definitionskörper, explizite Körperlinks und die dort beschriebene `contains`-/`realizes`-Richtung. Die Quelle dokumentiert diese Entscheidungen; diese Konsolidierung implementiert sie nicht.
- **Arbeitsdefinition oder Vorschlag:** die vierzehn Begriffe und ihre Verbindungen, konkrete Felder, zusätzliche Kinds, Syntax und die nachfolgenden Empfehlungen. Neuere Commitzeit allein macht daraus keinen Ersatz für eine frühere Nutzerentscheidung.

## Widersprüche und Entscheidungen

### 1. Component: allgemeiner Systemteil oder Laufzeitbaustein?

**Bisherige Bibliotheksfassung:** Ein `Component` war auf Laufzeit-/Bereitstellungsbausteine begrenzt. Ein `System` enthielt Components; ein Component enthielt logische Modules. Der erste Beispielvertrag erlaubte keine direkte Component-zu-Component-Zusammensetzung. Siehe [Architekturbegriffe im Proposal](../product-model-refinement/canonical-model-proposal.md#4-beziehungen-gliederung-und-verantwortung).

**Neues Vokabular:** [Component](component.md) bedeutet allgemein einen Systemteil. Die [Beziehungen](relationships.md#structure-and-files) erlauben verschachtelte Components. Task Logic, Application, Command Line und Local Storage sind dabei logische Verantwortungsbereiche, keine zugesagten getrennten Laufzeiteinheiten.

**Nutzerentscheidung vom 15. Oktober 2026:** „Allgemeiner Systembaustein, auch logisch und verschachtelbar.“ Die [Component-Definition](component.md) gilt damit auch für Task Logic und für ineinander gegliederte logische Teile. Ein eigener Prozess oder eine eigene Bereitstellung ist keine Voraussetzung. Die frühere enge Bedeutung ist abgelöst.

**Folgen:** Die Bibliotheksbausteine Frontend, Backend und Datenbank bleiben gültige Beispiele für Components. Sie begrenzen den Begriff nicht mehr auf Laufzeitarchitektur. Component-zu-Component-Zusammensetzung wird zulässig; die bereits vereinbarte Ableitung des `realizes`-Umfangs folgt auch diesen `contains`-Nachfahren. Eigenständiger Nutzen und Abgrenzung von System und Module sowie vollständige Parent- und Zielmengenregeln bleiben zu entscheiden. Die beiden Kinds werden nicht allein durch diese Component-Entscheidung entfernt.

### 2. Interface: Actor-Zugang oder technischer Vertrag?

**Bibliothek:** Interfaces verbinden auch interne Modules, etwa über Service- und Repositoryverträge; `provides` und `consumes` unterscheiden Anbieter und Verbraucher. **Neues Vokabular:** Ein [Interface](interface.md) ist der Zugang eines Actors zu UseCases.

**Entscheidung:** Gilt Interface ausschließlich für Außeninteraktion, oder auch für technische Zusammenarbeit innerhalb der Software? Ein interner Aufrufer darf nicht allein deshalb zu einem neuen Actor werden, damit ein bestehender Vertrag in die Definition passt.

**Empfehlung:** Interface als vereinbarten Interaktionsvertrag fassen; Actor-Zugang ist ein konkreter Einsatz. Anbieter, Verbraucher, angebotene Operationen und Fehler müssen im jeweiligen Beispiel nachvollziehbar bleiben. Das ist eine vorgeschlagene Erweiterung der kurzen Arbeitsdefinition.

### 3. Vierzehn Begriffe sind noch keine vollständige Kind-Liste

Das neue Vokabular ergänzt Actor, Scenario, Workflow, Procedure und Condition. Es definiert Mandate, System und Module nicht. Ihr Fehlen widerruft die älteren Entscheidungen nicht. Umgekehrt begründet eine verständliche Begriffsdefinition noch keinen eigenen maschinenlesbaren Kind.

Der gemeinsam zu prüfende Bestand umfasst damit siebzehn benannte Begriffe. Die neue [Einordnung der Wackelkandidaten](#arbeitsziel-und-wackelkandidaten) stellt Bedarf und Abgrenzung von System, Component/Module, Artifact, Mandate und Decision ausdrücklich zur Diskussion. Ihre frühere Aufnahme in einen Entwurf nimmt das Ergebnis dieser Prüfung nicht vorweg.

**Entscheidung:** Für jeden Begriff prüfen wir getrennt, ob er eigene Identität und Referenzierbarkeit benötigt, welche individuellen Aussagen er trägt und was sein Vorhandensein für Realisierung oder Prüfung bewirkt. Eine feste Zahl von Kinds ist kein Ziel.

**Besonderer Namenskonflikt:** `Decision.actor` referenziert im Bibliotheksentwurf ein Mandate, während Actor im neuen Vokabular eine beteiligte Rolle beschreibt. Rolle, Auftrag, ausführender Agent und dokumentierte Entscheidungsherkunft sind einzeln zu klären. Keines dieser Felder authentifiziert eine Freigabe.

### 4. Allgemeine Verbindung oder berechenbare Relation?

Das Vokabular erklärt Verbindungen wie „verarbeitet“, „dokumentiert“ und „prüft“ in Alltagssprache. Der Bibliotheksentwurf unterscheidet strukturierte Beziehungen und wählt für `realizes` einen automatischen Umfang über die `contains`-Unterstruktur.

**Entscheidung:** Die alltagssprachlichen Beziehungen werden erst nach ihrer Einzelprüfung auf einen Sprachvertrag abgebildet. Ein Körperlink erklärt eine explizite Verwendung, aber nicht automatisch Enthaltenheit, Regelgeltung, Realisierung oder Schreibzuständigkeit. Es gibt keine pauschale Umsetzung aller Verbindungen als `uses`.

**Erhaltene Designrichtung:** Wo `realizes` verwendet wird, umfasst es nach der dokumentierten Entscheidung Ziel und transitive `contains`-Nachfahren entsprechend der Beitragsrolle. Andere Beziehungen vergrößern diese Zielmenge nicht automatisch. Ändern wir die erlaubten `contains`-Kinder, müssen wir deshalb auch diese Folgewirkung prüfen.

### 5. Check: Prüfauftrag oder ausschließlich ausführbarer Befehl?

Die neue [Check-Definition](check.md) beschreibt Methode und Erfolgskriterium und kann auch einen Dokumentationsvergleich oder ein Review tragen. Main und der konkrete Bibliotheks-Feldkandidat verlangen dagegen einen nicht leeren Befehl als literale Argumentliste.

**Entscheidung:** Soll Check der allgemeine Prüfauftrag sein, dessen Methode ein Befehl oder ein begründetes Review sein kann? Oder bleiben manuelle beziehungsweise agentische Bewertungen ein gesonderter Mechanismus?

**Empfehlung:** Zuerst Erwartung, Methode und Erfolgskriterium fachlich beschreiben. Danach entscheiden, wie die Durchführung gebunden wird. Deklaration, Ausführung und aufgezeichnetes Ergebnis bleiben verschieden. Ein Reviewauftrag darf nicht als bereits ausgeführter Command-Check erscheinen.

### 6. Coverage entfernen: Konfiguration oder Produktpflicht?

Der Bibliotheksentwurf entfernt Coverage und seine Moduskonfiguration aus diesem Konzeptschritt und spezifiziert keine zusätzliche vollständige Repositoryabdeckung. Main und [DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored) verlangen weiterhin, dass jede Datei zugeordnet oder ausdrücklich ausgenommen ist.

**Entscheidung:** Das Entfernen eines Begriffs oder Schalters beantwortet nicht, ob unbekannte Dateien die Realisierung weiterhin blockieren. Soll diese Pflicht erhalten bleiben, braucht auch die neue Sprache eine nachvollziehbare Dateierfassung und Zuständigkeitsprüfung. Soll sie entfallen, wäre das eine ausdrückliche Änderung der bestehenden Produktentscheidung.

**Empfehlung:** Vollständige Dateirechenschaft erhalten; Modellvereinfachung und Konfigurationsort separat entscheiden. Hier wird weder ein alter Schalter wieder eingeführt noch DEC-006 geändert.

### 7. Rule, Condition, Procedure und Workflow

Hier liegt zunächst eine Präzisierung statt eines belegten Widerspruchs vor: Rule verpflichtet; Condition beschreibt einen Wahrheitswert; Check untersucht eine Erwartung. Procedure erklärt eine Tätigkeit; Workflow koordiniert Tätigkeiten und Beteiligte. Scenario beschreibt einen konkreten Fall. Die [Beziehungsseite](relationships.md) liefert die Arbeitsabgrenzung.

**Offen:** Ob Bedingungen eigene Definitionen brauchen, wie sie ausgewertet werden und welche Workflow-Übergänge maschinell erfolgen. Die Beschreibungen begründen noch keinen Ausdrucksevaluator und keine ausführbare Workflowengine. Auch ein imperativ formulierter Procedure-Text erteilt einem Agenten nicht selbst die Befugnis, die Schritte auszuführen.

Nach der jüngsten [Zielpräzisierung](#arbeitsziel-und-wackelkandidaten) ist bereits der Bedarf eigener Prozessbeschreibungen offen. Wir prüfen zunächst den Gegenstand des Ablaufs und seinen Beitrag zur Sollbeschreibung, bevor wir Workflow und Procedure als benötigte Typen behandeln.

### 8. Zwei Tasks-Darstellungen mit unterschiedlicher Genauigkeit

Das [Begriffsbeispiel](task-management-example.md) verwendet Title/Identifier und beispielhafte `src/...`-Pfade. Die [eigenständige Projektdokumentation](../task-management/README.md) verwendet Text/Reference, Python 3.13, `python -m tasks` und einen genaueren Speichervertrag. Ein fehlendes Bestätigungssignal nach dem Commit beweist dort ausdrücklich keinen fehlgeschlagenen Speichervorgang.

**Konkrete Abweichung:** Das Begriffsbeispiel sagt allgemein, ein fehlgeschlagenes Speichern bewahre die bisherigen Daten. Der [präzisere Speichervertrag](../task-management/docs/storage.md#save-procedure-and-commit-point) trennt Fehler vor dem Ersetzen von Fehlern bei der anschließenden Bestätigung. Diese Grenze muss eine spätere gemeinsame Modellierung erhalten.

**Vorgeschlagene Verwendung:** Die eigenständigen Dokumente sind die inhaltliche Gegenprobe für das Tasks-Projekt; das Begriffsbeispiel bleibt ein vereinfachter Entwurf für Begriffe. Wir gleichen ausgewählte Aussagen explizit ab und geben die verschiedenen Pfade und Namen nicht als denselben fertigen Modellstand aus. Der historische sieben-Kind-Versuch bleibt historisch.

## Nächster Designschritt

Die aktuelle Aufgabe ist die [minimale Sprachgrundlage](minimal-language/README.md) mit einem Connector-Durchstich fuer eigene Kinds und Relationen. Daran pruefen wir Ausdrucksfaehigkeit, Bedeutung und Pruefbarkeit, bevor ein neuer Sprachvertrag fixiert wird. System, Container, Component und die weiteren Wackelkandidaten werden als moegliche Vokabulartypen untersucht. Die konkrete Apply-Arbeitsweise wird getrennt betrachtet; der Umgang mit Dateirechenschaft ist vor einer ausfuehrbaren neuen Variante zu klaeren.

Für jede Entscheidung halten wir die gewählte Aussage, ein kleines Tasks-Beispiel, eine Gegenprobe an der Bibliothek, Auswirkungen auf bestehende Entscheidungen und verbleibende Fragen fest. Erst nach der Einigung ändern wir die betroffenen Begriffsdefinitionen und Beispiele. Syntax und Implementierung folgen den geklärten Bedeutungen.

Diese Konsolidierung enthält keine neue Sprachimplementierung und keinen Anwendungs- oder Wirksamkeitsnachweis. Aktuelle Produktverträge bleiben bei [Vision](../../vision.md), [Architektur](../../architecture.md), [Konzeptregister](../../concepts/register.md) und den bestehenden Operationsguides; Umsetzungsstatus bleibt bei [Roadmap](../../implementation-plan.md) und Backlog.

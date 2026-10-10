# Projektvertrag: Architektur, Arbeitsweise und Artefakte

Status: konsolidierter Zielentwurf vom 8. Oktober 2026. Die Produktvorgaben stammen aus der Nutzerdiskussion; Dateinamen, Felder und Speicherlayout sind konkrete Planungsvorschläge. Kein aktuelles CLI-Format oder implementierter Compiler wird behauptet. [Fachliche Module](conceptual-modules.md) besitzen die Modulgliederung und Dateibezüge; dieses Dokument präzisiert Projektvorgaben, erwartete Artefakte und die dauerhafte Nachverfolgung dieser Bezüge.

## Ein Projekt beschreibt auch seine eigene Arbeitsweise

Das Modell enthält neben Fachlichkeit drei verbundene Arten von Vorgaben. Sie werden beim jeweiligen Besitzer gepflegt und über ausdrückliche Referenzen und Geltungsbereiche verbunden, nicht über ein zweites globales Register.

| Vorgabe | Inhalt | Beispielablage unter `.markitect/model/` |
|---|---|---|
| Architektur | Systemgrenzen, Komponenten, technische Entscheidungen, Schnittstellen, Datenhaltung und Qualitätsanforderungen | `engineering/architecture.yaml`, lokale Ergänzungen in einem Slice |
| Arbeitsweise | Entscheidungsrechte, Entwicklungs- und Prüfregeln, Prozess, ausführbarer Workflow, Übernahme- und Auslieferungsbedingungen | `engineering/delivery/workflow.yaml`, relevante Zuständigkeiten in `manager.yaml` |
| Erwartete Artefakte | Welche Ergebnisse für einen Zweck erforderlich sind, wer sie verantwortet und wie sie überprüft werden | beim Use Case oder in einer lokalen Begleitdatei wie `orders/artifacts.yaml` |

Diese Ablagen sind Beispiele für zusammengehörige Konzepte. Sie begründen keine verpflichtenden horizontalen Ordner `architectures/`, `processes/` oder `artifacts/`. Ein kleiner Slice darf Vorgaben und Artefakterwartungen in einer Datei bündeln. Die Managerdeklaration referenziert geltende Verträge statt deren Inhalt zu kopieren.

Projektweite Vorgaben besitzen einen erklärten Geltungsbereich, zum Beispiel alle Anwendungsmodule oder alle öffentlich erreichbaren Endpunkte. Lokale Ergänzungen dürfen innerhalb delegierter Befugnisse präzisieren. Eine Ausnahme benötigt einen dazu befugten Entscheider, Begründung und einen ausdrücklich bezeichneten Scope; ein Kind überschreibt eine Elternvorgabe nicht durch ein gleichnamiges Feld. Deterministisch prüfbare Konflikte werden diagnostiziert, übrige Konflikte bleiben als Entscheidungsbedarf sichtbar. Auch nichtfunktionale Anforderungen wie Zugriffsschutz, Datenaufbewahrung, Leistung oder Verfügbarkeit können solche Vorgaben sein, sofern sie im jeweiligen Projekt relevant sind.

Die **aktive Modellrevision** bestimmt, wer einen Managervertrag ändern darf. Ein Manager kann Änderungen seiner `manager.yaml` vorschlagen und gewöhnliche Fachdetails im bestehenden Mandat entscheiden. Erweiterung eigener Befugnisse, Übernahme fremder Dateien oder Änderung der delegierten Grenze erfordert den nach bisherigem Vertrag befugten Vorfahren bzw. Nutzer. Der vorgeschlagene neue Vertrag kann sich nicht selbst autorisieren. Der erste Projektauftrag und Wurzelvertrag werden durch die ausdrücklichen Vorgaben des Nutzers begründet.

## Konkret am kleinen Shop

Der erste ausführbare Produktnachweis soll bewusst klein bleiben: ein Backend mit relationaler Datenbank, den fachlichen Modulen Orders und Inventory sowie einem integrierten Stornierungsablauf. Die Freigabe des reservierten Bestands und der Zustandswechsel erfolgen atomar in derselben Transaktion. Die Managementhierarchie erzeugt keine zusätzlichen Dienste. Ein Frontend ist für diesen ersten Nachweis nicht erforderlich; das Frontendbeispiel im Zuordnungsentwurf illustriert ein späteres, gleichartig zu behandelndes Artefakt.

Die Arbeitsweise lautet: fachliche Änderung entwickeln, Struktur und Folgen prüfen, innerhalb der vorhandenen Befugnis Modellstand annehmen, ausdrücklich beauftragte Umsetzung delegieren, Kindresultate prüfen und integrieren, den finalen Kandidaten gegen die Erwartungen abgleichen. Ein Auftrag darf Modellpflege und Umsetzung gemeinsam umfassen. Der Nutzer entscheidet nur wichtige vorbehaltene oder durch keine delegierte Ebene entscheidbare Fragen. Veröffentlichung und Betrieb haben ihre eigenen Projektbedingungen; ein erfolgreicher Build aktiviert sie nicht selbst.

Für „Order stornieren“ sind im gewählten Shop-Scope mindestens erforderlich:

| Erwartung | Zuständigkeit | Abschlusskriterium |
|---|---|---|
| Stornierungsablauf in der Anwendung | Orders, mit öffentlichem Freigabevertrag von Inventory | Stornierung vor Versand; wiederholte Ausführung gibt Bestand nicht mehrfach frei |
| Realisierung der Bestandsfreigabe | Inventory | Reservierung wird korrekt und wiederholungssicher freigegeben |
| Lokale Tests und Prüfung des gemeinsamen Transaktionsablaufs | zuständige Fachmanager und gemeinsamer Integrator | ausgeführte Prüfungen am finalen Kandidaten; gemeinsamer Fehlerfall rollt beide Änderungen zurück |
| Lesbare fachliche Spezifikation | zuständiger Modellbesitzer, über den verantworteten Erzeugungsweg | aktuelle Begriffe, Regeln, Beispiele und erkennbare Modellversion |
| Produkt-/API-Dokumentation | ausdrücklich zugeteilter Dateibesitzer | betroffener Nutzungsablauf und Fehlerverhalten entsprechen dem finalen Stand |

Bestehende Pipeline- und Betriebsartefakte werden weiter verwendet, soweit sie genügen. Eine neue Funktion verlangt nicht automatisch neue Docker- oder Pipeline-Dateien. Ihre unveränderte Gültigkeit und benötigte Prüfungen bleiben nachvollziehbar. Zusätzliche Ergebnisse wie UI, Migration, Betriebsanleitung oder Monitoring werden erforderlich, wenn eine ausdrücklich geltende Projektvorgabe oder eine konkrete Änderung sie verlangt. Jede Anwendbarkeitsentscheidung ist begründet; eine bloße fehlende Zuordnung macht ein Pflichtartefakt nicht optional.

## Erwartung, Zuordnung und Nachweis sind eigene Aussagen

Eine **Artefakterwartung** benennt Identität, fachlichen Zweck bzw. erfüllte Verpflichtungen, Rolle, verantwortetes Modul, Geltung/Anwendbarkeit und Abschlusskriterien. Ein konkreter Pfad darf zunächst offen sein. Pflicht und offene Umsetzung bleiben trotzdem sichtbar. Die Syntax und das minimale Schema sind vor der Produktimplementierung zu wählen.

Eine **Dateizuordnung** verbindet eine solche Erwartung bzw. einen Modellinhalt mit konkreten Repositorypfaden und deren Rollen. Ein überprüfbarer Bezug verlangt mindestens Modellidentität, Pfad oder ausdrücklich definierten Selektor und Beziehungsart. Die Dateiverantwortung ergibt sich aus der zuständigen Managerdeklaration; sie wird nicht als zweiter unabhängig editierbarer Owner in jeder Beziehung kopiert. Abgeleitete Ansichten können den aufgelösten Owner anzeigen.

Ein **Nachweis** bindet das ausgeführte Verfahren, seine Eingaben, den tatsächlichen Kandidaten und das Ergebnis. Eine vorhandene Testdatei ist noch kein ausgeführter Test; eine grüne Prüfung ist nur für ihren ausgewiesenen Umfang gültig. Der lesbare Status zeigt deshalb mindestens: strukturell gültig oder ungültig, fehlende Realisierung, vorhandene Dateizuordnung, ausgeführte Prüfung und verbleibende semantische Lücken. Diese Zustände werden nicht auf ein pauschales „fertig“ reduziert.

## Dauerhafte Quellen und wiederherstellbares Compilerresultat

Die Zuordnungen dürfen nicht ausschließlich im Gedächtnis eines Agenten oder in einem flüchtigen Laufbericht stehen. Akzeptierte Managerdeklarationen, Artefakterwartungen, fachliche Dateibezüge und ausdrücklich angenommene Umbenennungen bilden versionierte kanonische Quellen unter `.markitect/model/`. Git erhält die Entwicklung der jeweils geltenden Zuständigkeit und Absicht; eine alte Revision behält ihre damalige Verantwortung.

Aus diesen Quellen und einem ausdrücklich ausgewählten Repositorysnapshot erzeugt der Compiler bzw. Host einen **Dateiindex**. Er ist eine wiederherstellbare Ausgabe, keine weitere handgepflegte Wahrheit. Ein möglicher lokaler Speicherort ist `.markitect/cache/compiled/`; langfristig benötigte gebundene Ergebnisse werden gezielt zusammen mit den Laufnachweisen aufbewahrt. Ein verlorener Cache darf weder Zuordnungen noch akzeptierte Entscheidungen verlieren.

Die Gültigkeitsbindung umfasst zusätzlich eindeutige Compiler-/Host-Buildidentitäten bzw. Inhaltsdigests, die tatsächlich verwendeten Schema-/Paket-/Policyinhalte und relevante Prüf-/Werkzeugkonfiguration. Projektauswahl, Ausschlüsse, Inventarisierungs- und Pfadnormalisierungsregeln einschließlich Groß-/Kleinschreibung und Linkbehandlung sind Teil dieser Bindung. Ein wiederverwendeter Versionsname genügt nicht. Ein gespeicherter Index oder Impactbericht darf nur bei übereinstimmenden Eingabebindungen wiederverwendet werden. Nach Abbruch vor Delegation werden sie bei Bedarf deterministisch neu erzeugt und mit dem gespeicherten Auftragsstand abgeglichen; Wiederaufnahme erzeugt keine doppelt erteilten Aufträge. Ein Cache ist löschbar, die für einen beauftragten Lauf erforderlichen Entscheidungshistorien und Kandidatenbezüge sind es nicht.

Der Index enthält oder referenziert für jede erfasste Datei:

- normalisierten Repositorypfad, Klassifikation und auflösbaren verantwortlichen Manager;
- fachliche Zugehörigkeit, direkt verknüpfte Modellidentitäten und über explizite Verträge abgeleitete Bezüge samt Herkunft;
- Rollen und erfüllte Artefakterwartungen sowie zuständige Erzeugungsquelle bei generierten Dateien;
- die getrennte Beobachtung, ob die Datei im Snapshot vorhanden ist, und Bezüge zu passenden Prüfungen;
- Herkunft jeder Aussage und die Bindung an Modellrevision, Repositoryinhalt, Compiler-/Schemaversion und relevante Konfiguration.

Erwartete, noch nicht vorhandene Dateien bzw. Ergebnisse bleiben im selben Abgleich sichtbar. Die Rückwärtsansichten nach Manager, Modellinhalt und ursprünglicher YAML-Quelldatei werden daraus abgeleitet. Ein reiner Zeitstempel oder HEAD-Name genügt nicht zur Gültigkeitsprüfung; auch lokale Änderungen und die tatsächlich aufgelöste Dateimenge sind Eingaben. Der erste Implementierungsschritt darf den Index vollständig neu aufbauen. Inkrementelle Berechnung ist eine spätere Optimierung mit demselben Ergebnisvertrag.

Pfade werden repositoryrelativ und eindeutig normalisiert. Absolute Pfade, `..`-Ausbrüche und mehrdeutige Schreibweisen werden nicht als gewöhnliche Zuordnung akzeptiert. Regeln für Groß-/Kleinschreibung und verfolgte Links müssen auf der technischen Basis ausdrücklich festgelegt und mit Git-/Plattformverhalten abgeglichen werden. Fremdverwaltete, generierte oder ausgeschlossene Dateien besitzen eine erklärte Klassifikation und einen zuständigen Abgleichbesitzer; nicht inventarisierte Dateien erscheinen als Lücke. Exploration darf solche Lücken enthalten. Ein erfolgreicher Abschluss beansprucht nur den beauftragten und tatsächlich abgeglichenen Umfang, nicht pauschal vollständige Repositoriumabdeckung.

Kanonische Dateien unter `.markitect/model/` bleiben Modelleingaben. Cache, Laufdaten und abgeleitete Sichten sind gemäß erklärter Auswahl Ausgabe bzw. Arbeitsdaten; der Compiler darf seine eigenen wechselnden Ausgaben nicht versehentlich erneut als fachliche Eingaben inventarisieren. Eine bewusst einbezogene erzeugte Produktdatei wird mit ihrem Erzeugungsweg und explizitem Abgleich behandelt. Die Klassifikation gehört zur gebundenen Konfiguration.

## Änderungen an Modell und Dateien gemeinsam abgleichen

Bei jedem relevanten Abgleich werden alte und neue Modellrevision sowie alte und neue Dateimenge betrachtet. Dadurch verschwinden entfernte Referenzen, gelöschte Dateien oder nicht mehr passende Pfadmuster nicht vor der Impactberechnung. Modellidentität und Speicherort sind getrennt: Alle deklariert bedeutungstragenden Modellpfadsegmente wirken auf den Namespace; eine solche Verschiebung verlangt eine ausdrückliche Identitätsmigration. Ein interner Ordner ohne eigenen Manager bleibt unter der Verantwortung des nächsten deklarierten Managers. Die endgültige Namensauflösung wird vor P1 festgelegt.

Dateiumbenennungen können von Git oder einem Agenten vorgeschlagen werden; Ähnlichkeit allein bewahrt keine akzeptierte Zuordnung. Die befugte Aktualisierung benennt alten und neuen Pfad und den fachlichen Zusammenhang. Wenn ein Modellinhalt entfällt, werden bisherige Realisierungen und Verbraucher erneut abgeglichen. Daraus folgt keine automatische Löschung: gemeinsame Verwendung, notwendige Datenmigration und technische Restaufgaben sind zu entscheiden und zu prüfen.

Versehentliches Löschen lässt die bisherige Pflicht und erwartete Zuordnung bestehen und erzeugt einen Befund. Ein weiter aktives Artefakterfordernis bleibt nach dem Löschen seiner einzigen Realisierung unerfüllt, selbst wenn zusätzlich der Dateibezug entfernt wurde. Absichtliches Entfernen benötigt eine befugte Entscheidung über Ersatz, begründete Nichtanwendbarkeit oder tatsächliches Entfallen der Pflicht; das Entfernen der Pflicht selbst erfordert eine versionierte Vertragsänderung. Ersatzdateien und Umbenennungen erhalten frische Artefaktbindungen, und betroffene Nachweise werden erneut abgeglichen. Die Entscheidung bindet Pflicht, alte/neue Zuordnung und Evidenzbehandlung im gemeinsamen Kandidaten. Gleichartige neue Bytes machen alte Evidenz nicht automatisch gültig.

Für neue Dateien verläuft die Pflege: tatsächlicher Änderungsbericht → befugter Zuordnungsvorschlag → Struktur-/Ownershipprüfung → angenommene Modellrevision → erneuter Impact und finale Prüfung am gebundenen Stand. Ein zusammengehöriger Kandidat wird nur mit seinem dazugehörigen Modell- und Zuordnungsstand übernommen. Abbruch oder verworfener Kandidat aktiviert dessen vorgeschlagene Zuordnung nicht im Zielrepository. Die Wiederaufnahme rekonstruiert den Stand aus versionierten Quellen und gebundenen Laufdaten.

Änderungen an Architektur, Prüfpflichten oder Workflow sind selbst Modelländerungen mit betroffenen Verbrauchern. Ein laufender Auftrag darf seine eigenen Abschlusskriterien nicht stillschweigend lockern. Er bleibt an die geltenden Vorgaben gebunden, bis eine befugte Änderung angenommen, ihre Auswirkung auf laufende Arbeit bestimmt und der Auftrag ausdrücklich neu gebunden wurde. Alte Nachweise werden nur bei nachgewiesener Gültigkeit wiederverwendet.

## Compiler, Manager und Grenze des Nachweises

Der Compiler prüft deterministisch Identitäten, Referenzen, deklarierte Zuständigkeiten, erlaubte Beziehungen, gewählte formale Regeln und die strukturelle Artefaktabdeckung. Die Hostschicht bindet Dateisnapshots, Index und Ausführungsnachweise. Manager entscheiden fachliche Bedeutung, offene Anwendbarkeit, sinnvolle Umsetzung und Integration innerhalb ihrer Befugnisse. Ein unbekannter oder widersprüchlicher Bezug wird als Befund weitergereicht und nicht als Nichtbetroffenheit ausgegeben.

Der Sollvertrag ist damit dauerhaft verfügbar: **Wer verantwortet die Datei, wozu gehört sie, was soll sie realisieren und wie wurde das geprüft?** Die Verknüpfung erlaubt diese Fragen reproduzierbar zu beantworten. Fachliche Konformität von Code, Kommentaren und Dokumentation benötigt weiterhin passende Checks und unabhängige Beurteilung; der reine Index beweist sie nicht.

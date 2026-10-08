# Plan: vom Projektmodell zur verantworteten Umsetzung

Stand: 8. Oktober 2026. Dies ist der Umsetzungsplan des [Planungspakets](README.md). Er beauftragt zunächst Planung und behauptet keine Produktimplementierung. Die [Root-Roadmap](../../implementation-plan.md) bleibt Besitzer des allgemeinen Quell- und Release-Status.

## 1. Aktuelle Arbeitsgrenze

Erledigt in diesem Planungsschritt: isolierter Worktree und Feature-Branch, Festhalten der Nutzerrichtung, Modellvorschlag, Terminologieabgrenzung, erste Umsetzungsschritte und Abnahmeszenarien. Produktcode, Schemas, Runtime, bestehende Releaseverträge und andere Worktrees bleiben in diesem Schritt unverändert.

Die Planungsbasis `53be4b98ece0cd9e797530f18c29ec8ae683f3fd` enthält die bisherige Government-Diskussion. Die lokale Hauptlinienreferenz `a97cbd5ef3e0b22b9e6397501047a4e01dc90204` ist separat zu untersuchen. Der [Architect-Abgleich](../architect-government-checkpoint.md) bindet historische technische Belege an `1ea5c76f55526fc4d721e865885436153f48b497`; sie gelten nicht automatisch für die neue Planung oder andere SHAs.

Vor der ersten Produktcodeänderung: technische Basis wählen, Änderungen und aktive Besitzer inventarisieren, dieses Planungspaket übertragen, tatsächliche Verträge an exakten Quellen nachlesen. Die bloße Existenz eines aktuellen Remote-Refs ist keine erneute CI- oder Runtime-Verifikation.

## 2. Wiederverwendung und Lücken

Die bestehende [Änderungslandkarte](../delegated-engineering-change-map.md) und der [Quellübergang](../government/source-transition.md) sind Herkunft für den ersten Abgleich, keine Belege neu ausgeführter Tests. Der folgende Hauptlinienabgleich wurde durch Lesen der angegebenen Dateien mit `git show a97cbd5ef3e0b22b9e6397501047a4e01dc90204:<Pfad>` unabhängig nachgeprüft.

| Quelle an `a97cbd5ef3e0b22b9e6397501047a4e01dc90204` | Beobachtung und Grenze |
|---|---|
| `internal/core/types.go`, `internal/core/compile.go` | Schema, Kind, Property, Definition, vollständige Identität und explizite Referenzkanten existieren. Compile führt keine Policy oder I/O aus. Die Kanten besitzen keine automatische fachliche Traversierungssemantik. Verantwortungs-/Workflowregeln gehören in ausdrücklich dafür zuständige Prüfungen oberhalb des strukturellen Core. |
| `examples/capability-ontologies/workflow-responsibility/schema.yaml` | Role und WorkItem mit genau einer accountable-Referenz sowie optionalen Reviewern sind ein struktureller Ausgangspunkt. Das Beispiel enthält noch keine rekursive Verantwortungsorganisation. |
| `examples/canonical-projection/definitions/create-order.use-case.yaml` | Ein UseCase referenziert einen Handler. Definitionen von Order, Stornierung und Reservierung sowie ihre fachliche Prüfung entstehen dadurch nicht. |
| `internal/host/authoring/model.go` | Der historische Area-Typ enthält Name, Path, Imports und Rules. Er ist nicht der neue rekursive Bereich mit einem Verantwortlichen; Namenskollision explizit auflösen. |
| `docs/development/modules.md`, `docs/design/canonical-projection-reset.md` | Installierbare Module sind im heutigen Vertrag strikt schema- oder zielbezogene Fähigkeitspakete. Namespace ist Teil der DefinitionIdentity; beides schafft keine Managementhierarchie. Neue Modul-/Ownershipsyntax und Altverträge müssen bewusst zugeordnet werden. |

Das ist ein begrenzter Quellabgleich, keine Vollständigkeitsprüfung aller Implementierungen oder erneute Gate-Ausführung. Der Produktcode der Planungsbasis wird damit nicht aufgewertet.

| Mechanismus | Geplante Behandlung |
|---|---|
| Struktureller deterministischer Compiler, typisierte Referenzen und Provenienz | Behalten; neue Vokabulare über vorhandene Mechanismen prüfen, keinen NLP-Compiler voraussetzen |
| Feste Quellen und konservative Folgenabschätzung | Behalten; Auswirkungen unterschiedlicher Beziehungstypen ausdrücklich definieren |
| Akzeptierte Definitionen | Um klaren Begriffs-/Kontextvertrag und Verwendungen in Regeln/Use Cases ergänzen |
| Bestehende Ziel- und Artefaktverträge | Von Verantwortungsbereichen und einmaligen Arbeitsaufträgen unterscheiden; Auftragsorganisation entkoppeln |
| Lokale und Elternprüfung | Für Verantwortungsbereiche nutzbar machen, ohne Prüfpflichten durch Aggregation zu verlieren |
| Pakete und technische Fähigkeiten | Versionierung und explizite Adoption beibehalten; Maintainer und lokale Verantwortlichkeit ergänzen |
| Freie Notizen und Vorschläge | Leichten Weg zur bewussten Modellannahme entwickeln; unverbindliche Inhalte nicht aktivieren |
| Dateiinventar und Nachweise | Viele-zu-viele Zuordnung, eindeutige Writer und sichtbare unbekannte Artefakte erhalten |

## 3. Begriffsübergang

„Projektion“ ist für das Zielprodukt zurückgezogen. Technische Altverträge benötigen eine semantische Zuordnung statt globaler Textersetzung:

| Historischer Begriff / Vertrag | Zielbegriff nach tatsächlicher Bedeutung |
|---|---|
| `Projection` als dauerhafter Vertrag einer Darstellung | Zielvorgabe bzw. Realisierungsvertrag; endgültigen öffentlichen Namen am Beispiel entscheiden |
| `Projection Module` als installierte Zielkompetenz | Umsetzungsfähigkeit bzw. Werkzeugmodul |
| `ProjectionPolicy` | Regeln für eine bestimmte Realisierung; keine allgemeine Bereichsbefugnis |
| Controller-Proposal / Run | Arbeitsplan / konkreter Durchlauf |
| Projection-gebundener Assurance-Scope | Prüfung eines expliziten Verantwortungs- und Integrationsumfangs |
| Materialisierte Dateien | Artefakte bzw. Realisierungen |

Ein **Bereich** ersetzt keinen der Altverträge pauschal. Der Bereich verantwortet Ziele und Arbeit; der Realisierungsvertrag beschreibt, was entstehen bzw. erhalten bleiben soll. Die spätere Umstellung betrifft Dokumentation, Authoring, Schemas, CLI, Berichte, gespeicherte Records und Beispiele gemeinsam.

Kompatibilität bestimmt nicht die Zielarchitektur. Trotzdem muss jede Änderung alter Konfigurationen oder gespeicherter Evidenz ausdrücklich migriert oder als neue inkompatible Version ausgewiesen werden. Unveränderliche Releases und historische Berichte werden nicht umgeschrieben. Unbekannte Pflichtabdeckung oder nicht migrierbare Nachweise bleiben sichtbar.

## 4. Endliche Umsetzungsschritte

Die [konkrete Bedien- und Modellbeschreibung](shop-walkthrough.md), der [vollständige Modellbeleg](shop-example/README.md) und das [Laufzeitprotokoll](operating-protocol.md) liefern jetzt den ersten durchgehenden Entwurf für P0. Die Formate sind ausdrücklich privat vorgeschlagene Designsyntax. P1–P4 sind damit weder implementiert noch technisch abgenommen.

### P0 — Modell am Beispiel präzisieren

Lieferung: akzeptierter Minimalwortschatz, vollständiger Order-/Reservierungsfall, Entscheidungsprotokoll für die unten offenen Fragen und exakter technischer Basis-SHA. Den vorhandenen Shop-Beleg und seinen Prüfer auf die aktuelle Slice-Struktur unter `.markitect/` migrieren; bis dahin bleibt er ausdrücklich erste Iteration.

Endkriterium: Eine Person kann von einer freien Notiz zu akzeptierten Begriffen, Use Case, Verantwortung, konkretem Auftrag und überprüfter Realisierung navigieren. Ungeklärte Fachentscheidungen sind ausdrücklich offen. Öffentliche Begriffe haben jeweils eine Bedeutung.

### P1 — Fachsprache und Ownership ausdrücken

Lieferung: kleinstes kanonisches YAML-Beispiel für Konzepte, Kontext, Regeln, Use Cases, rekursive verantwortete Module und Rollenauflösung; Namensraum und Verantwortung folgen der gemeinsamen Deklaration gemäß neuestem Zielstand. Dazu gehören Schemata und gezielte Compiler-/Hostdiagnostik.

Endkriterium: Mehrdeutige Referenzen, fehlende Verantwortliche, zyklische Bereichseltern und widersprüchliche Eigentümerschaft werden deterministisch gemeldet. Ein tieferer Bereich benötigt keinen neuen Typ. Prosa erzeugt keine versteckten Referenzen. Vorhandene Kernverträge zuerst nutzen; erst belegte Lücken erweitern.

### P2 — Organisation und Prozesse verbinden

Lieferung: ausdrückliche Zuständigkeit, ein eigener spezialisierter AI-Agent je Geschäftsführer-/Managerrolle, getrennte Arbeitskontexte, Entscheidungsbefugnisse, Kontextzuschnitt pro Ebene, Delegations- und Reportingpfade, Prozesse, deklarierte Workflow-Schritte und konkrete Aufträge; getrennte kanonische Definitionen und Laufzustände.

Endkriterium: Eine Modelländerung wird entlang der betroffenen Managementpfade von oben nach unten konkretisiert; Berichte und Entscheidungen laufen nachvollziehbar zurück. Jeder Manager arbeitet als eigener AI-Agent mit funktionsspezifischem Kontext. Er muss die Kindkontexte nicht kennen; offene Integrationsfragen klärt er über gezielte Aufträge und Rückfragen. Gewöhnliche Entscheidungen werden innerhalb delegierter Befugnisse selbst getroffen. Konflikte steigen nur bis zur nächsten entscheidungsfähigen Ebene; der Nutzer erhält ausschließlich wichtige vorbehaltene oder durch keine Managementebene lösbare Fragen. Modellierungsrekursion wird von begrenzter Laufzeit getrennt.

### P3 — Umsetzung und Prüfung nach Bereichen organisieren

Lieferung: Artefaktzuordnung, Fähigkeitenauswahl, Kandidatenbildung, Writerkoordination, aussagekräftige Kindberichte, unabhängige lokale Prüfung und aktive Kompatibilitätsprüfung und Integration durch jeden Elternmanager.

Endkriterium: Der Stornierungsfall verändert tatsächlich erforderliche Dateien; fehlende Bestandsfreigabe wird trotz erfolgreicher lokaler Order-Prüfung sichtbar. Modelländerung und Reparatur bei unverändertem Modell sind getrennt durchlaufbar. Zwei zuständige Bereiche schreiben nicht unkoordiniert denselben Pfad.

### P4 — Produkteinstieg und Übergang abschließen

Lieferung: README und Navigation entlang des Projekts, Gespräch als Einstieg, erzeugte lesbare Spezifikation mit fachlicher Änderungsansicht, ausführbares Beispiel, umgestellte Authoring-Ressourcen, öffentliche Terminologie und expliziter Umgang mit alten Records/Konfigurationen. Der [neueste Zielstand](conceptual-modules.md) besitzt die Ablage unter `.markitect/`, die Slice-Struktur und die vorgeschlagene `manager.yaml`-Deklaration. Modellquellen und erforderliche Konfiguration werden versioniert; lokale Arbeitsdaten und abgeleitete Sichten erhalten eine explizite Ablage-/Exportpolitik.

Endkriterium: Ein neues Projekt kann mit Ziel, Fachbegriffen und Use Case beginnen, ohne zuerst Skills, Agents oder eine Anbieterintegration zu konfigurieren. Der geplante Begriff ist aus aktiven öffentlichen Verträgen entfernt; historische Namen erscheinen nur in klaren Übergangs- bzw. Herkunftsabschnitten. Erforderliche Gates laufen am finalen Kandidaten; Release bleibt eigener Auftrag.

Ein Nutzer kann eine fachliche Änderung im Gespräch entwickeln, ihre Bedeutung und Auswirkungen prüfen und die bereits beauftragte Umsetzung starten lassen, ohne YAML zu lesen oder manuell zu bearbeiten. Lesbare Sichten zeigen ihren Modellstand und führen Änderungen auf die kanonischen Quellen zurück. Das Modell ist die primäre Arbeitsoberfläche; nachvollziehbare Realisierung und Prüfung bleiben sichtbar.

## 5. Abnahmeszenarien

### Technische Abbildung des ausgearbeiteten Beispiels

Arbeitsentscheidungen: lokales Modellverzeichnis mit expliziter Dateiauswahl; fünf Managerrollen; verantwortete Namespace-/Moduldeklarationen; ein Monolith mit gemeinsamer Datenbanktransaktion; getrennte Fachdefinitionen und Runtimekonfiguration; getrennte Agentenaufträge und pro Ebene integrierte Kandidaten. Der private Belegprüfer validiert ausschließlich die innere Konsistenz dieses Beispiels.

| Beispielvertrag | Konkrete Umsetzungslücke |
|---|---|
| `designVersion`, `resources`, kurze IDs | Neue Authoring-/Schemaentscheidung oberhalb des Core. Kein heutiges CLI-Format und keine zweite kanonische IR. Quelle eindeutig auf vollständige versionierte Core-Identitäten abbilden. |
| Concept/State und gemischte Referenzlisten | Der heutige Core-Referenzvertrag benennt je Property genau einen Ziel-Kind. Vor P1 gemeinsamen Term-Kind mit Kategorie oder getrennte typisierte Felder für Konzepte, Zustände und Regeln wählen. Kein vorhandenes Union-Referenzfeature behaupten. |
| Owner/Area/Module/Namespace | Auflösbare Rollen und konkrete Kardinalitäten; spezifische Hostprüfungen für Parent-Zyklen und delegierte Befugnisse oberhalb struktureller Core-Referenzen. |
| Beziehungseffekte | Kontext, Impact, Pflichtabdeckung und Routing ausdrücklich unterscheiden. `uses` liefert Kontext/Änderungsbezug; `requiresRealization` verlangt Umsetzung bzw. Nachweis, `realizes` verbindet mit Artefaktgruppen. Ownership-/Namespacekanten aktivieren nicht alle Gegenstände derselben Organisation. |
| Neue Pflicht nutzt unveränderte Definitionen | Vorhandene Realisierung und Checks auf die neue Verpflichtung beziehen; beim Shop kann Inventararbeit entstehen, obwohl seine Fachdefinitionen unverändert bleiben. |
| Artefaktzuordnung | Kanonischen Bereich und gewünschte Zuordnung von konkretem Writer und beobachteten Bytes trennen. Neue Pfade zunächst im Laufbericht erfassen, autorisierte Zuordnungsänderung validieren/annehmen, Impact neu bestimmen und betroffene Checks am final gebundenen Stand ausführen. |
| Eigener Manager-Agent je Rolle | Hostinvocation, rollenbezogener gespeicherter Stand, getrennte Kontextauswahl und typisierte Nachrichten; kein geteilter Gesprächsverlauf unter Rollenetiketten. |
| Delegation und Integration | Auftragsbaum unabhängig vom bisherigen Ziel-/Prüfscope; isolierte Kindkandidaten, eindeutige Writer und eigener Integrationskandidat pro Elternauftrag. |
| Modellannahme und Start | Zwei gebundene Zustandsübergänge, die in einem bereits autorisierten Nutzerauftrag zusammen ausgeführt werden dürfen; keine zusätzliche manuelle Routinefreigabe ableiten. |
| Modellrevision während des Laufs | Betroffene Aufträge neu binden; unabhängige Arbeit nur nach nachvollziehbarer Gültigkeitsprüfung wiederverwenden. Finale Evidenz bindet tatsächlich finalen Modell-/Artefaktstand. |

Diese Tabelle bestätigt keinen bereits vorhandenen Decoder oder Runner. Vor Kernänderungen oder Recordmigrationen müssen Ausdrucksbedarf und betroffene Verbraucher gegen die exakt gewählte technische Basis geprüft werden.

### Geplante Produktfälle

Die folgenden Szenarien sind Anforderungen für spätere Implementierungsprüfungen. Sie wurden in diesem Planungsschritt nicht ausgeführt.

| ID | Fall | Erwartung |
|---|---|---|
| A1 | Brainstorming enthält widersprüchliche Order-Definitionen | Exploration bleibt möglich; beide Vorschläge schaffen noch keine aktive Pflicht |
| A2 | Order existiert in Verkauf und Einkauf | Kontextgebundene Referenzen lösen eindeutig auf; unklare Referenzen werden gemeldet |
| A3 | Bedeutung von Reservierung wird geändert | Ausdrückliche Verbraucher, betroffene Realisierungen und ungültige Nachweise werden bestimmt; unbekannte Eingaben engen Impact nicht heimlich ein |
| A4 | Ein definierter Begriff hat keinen ausführbaren Check | Struktur kann gültig sein; semantische Durchsetzung wird nicht als bestanden ausgegeben |
| A5 | Bereich, Namespace oder Modul hat keinen Verantwortlichen | Akzeptierter operativer Vertrag ist ungültig bzw. unvollständig; keine stille Git-/Elterninferenz |
| A6 | Ein Bereich wird mehrfach rekursiv unterteilt | Derselbe Vertrag gilt auf jeder Tiefe; Zyklen und zwei Eltern werden abgelehnt |
| A7 | Ein Kind versucht eigene Befugnisse auszuweiten | Ausführung bzw. Übernahme wird verweigert; der geltende Elternrahmen bleibt maßgeblich |
| A8 | Orders und Lagerbestand liefern lokal erfolgreiche Ergebnisse | Elternprüfung untersucht zusätzlich den vollständigen Ablauf und kann dennoch fehlschlagen |
| A9 | Zwei Bereiche benötigen dieselbe Datei | Ein konkreter Writer oder ein expliziter Integrator koordiniert den Kandidaten |
| A10 | Projekt übernimmt ein versioniertes Modul | Maintainer und lokale Adoption sind eindeutig verantwortet; Installation aktiviert keine neue Pflicht |
| A11 | Prozess und konkreter Workflow werden geändert | Zuständigkeit, betroffene Abläufe und Nachweise sind getrennt nachvollziehbar |
| A12 | Docker-Datei ist noch keinem Modellzweck zugeordnet | Als ungeklärt sichtbar; keine automatisch erfundene Absicht und kein automatisches Löschen |
| A13 | Artefakt driftet bei unverändertem Soll | Reparaturauftrag entsteht ohne fingierte Modelländerung; neue Prüfung bindet tatsächliche Bytes |
| A14 | Öffentlicher Altbegriff wird ersetzt | Alte Evidenz wird nicht stillschweigend neu gebunden; Versions-/Migrationsentscheidung ist prüfbar |
| A15 | Eine Änderung betrifft zwei tiefe Bereiche und einen gemeinsamen Vorfahren | Information und Aufträge folgen den betroffenen Pfaden von oben nach unten; jeder Manager übersetzt den Auftrag für seine Kinder |
| A16 | Ein Manager erhält seinen Arbeitskontext | Relevante Regeln, Schnittstellen und Befugnisse sind enthalten; Kindkontexte und deren interne Zusammenstellung muss er nicht kennen |
| A17 | Zwei Kindberichte bestehen lokal, verlangen aber inkompatible Schnittstellen | Elternintegration erkennt den Konflikt; der Manager entscheidet innerhalb seiner Befugnis und delegiert die nötige Anpassung |
| A18 | Ein Konflikt kann vom gemeinsamen Elternmanager entschieden werden | Er entscheidet und berichtet; keine unnötige Eskalation zur Geschäftsführung oder zum Nutzer |
| A19 | Erst ein höherer Manager besitzt die erforderliche Befugnis | Eskalation folgt dem Elternpfad mit konkreter Frage, Fakten, Alternativen und Empfehlung; Entscheidung fließt nach unten zurück |
| A20 | Eine gewöhnliche Frage erreicht die umfassend befugte Geschäftsführung | Sie entscheidet selbst; ein dem Nutzer vorbehaltener Grundsatzkonflikt wird dagegen gezielt vorgelegt |
| A21 | Ein Kind liefert veraltete Evidenz oder einen unvollständigen Bericht | Der Elternmanager kann den Gesamtauftrag nicht als erfolgreich abschließen; erneuter Abgleich oder fehlende Arbeit wird konkret benannt |
| A22 | Geschäftsführung, Elternmanager und Kindmanager bearbeiten eine Änderung | Jede Rolle wird durch einen eigenen AI-Agenten mit eigenem Kontext ausgeübt; ein gemeinsamer Gesprächskontext mit Rollenetiketten genügt nicht |
| A23 | Ein Kindbereich erweitert seine interne Zerlegung oder lädt mehr Fachdetails | Der Elternmanager kann über unveränderte Ergebnis-/Schnittstellenverträge weiterarbeiten; nur relevante Folgen, Berichte und konkrete Antworten gelangen nach oben |
| A24 | Neuer Use Case verlangt unveränderte Bestandsdefinitionen | `requiresRealization` und `realizes` machen benötigte Umsetzung bzw. Prüfung sichtbar; unveränderte Quelle wird nicht mit Nichtbetroffenheit verwechselt |
| A25 | Eine neue Regel verwendet unveränderte Architektur als Kontext | Bestehende Checks laufen; ein Plattformmanager erhält erst bei tatsächlich betroffenen Plattformpflichten Arbeit |
| A26 | Nutzerauftrag umfasst Modellpflege und Umsetzung | Interne Annahme-/Startschritte bleiben gebunden, erfordern aber keine zusätzliche pauschale Rückfrage |
| A27 | Umsetzung erzeugt neuen Pfad im delegierten Bereich | Writer, autorisierte Zuordnungsänderung und erneute finale Modell-/Artefaktprüfung bleiben nachvollziehbar |
| A28 | Nutzer beschreibt eine neue Regel im Gespräch | Strukturierte Modelländerung, verständliche fachliche Differenz und passende Prüferwartungen entstehen ohne manuelles YAML-Editing |
| A29 | Neues Projekt wird angelegt | Alle Markitect-eigenen Dateien liegen unter `.markitect/`; kanonische Quellen sind versionierbar, lokale Arbeitsdaten werden ausdrücklich behandelt |
| A30 | Nutzer ändert die lesbare Spezifikation | Änderung fließt über die kanonischen Modelleingaben und deren Prüfung; kein konkurrierender Besitzer derselben Aussage entsteht |
| A31 | Modell wurde geändert, lesbare Sicht oder Prüfung ist älter | Gebundener Stand und fehlende Aktualität sind sichtbar; ein gültiges Modell wird nicht mit nachgewiesener Umsetzung gleichgesetzt |

## 6. Noch zu entscheidende Details

- Minimale Typen: Der Designbeleg unterscheidet Konzept und Zustand. Im tatsächlichen Schema gemeinsamen Term-Kind oder getrennte typisierte Referenzfelder wählen; das Beispiel beweist keine polymorphe Core-Referenz.
- Kontext: Nach aktueller Nutzerpräzisierung tragen fachliche Ordner und eine gemeinsame Deklaration Namensraum und Verantwortung, statt paralleler lokaler Register. `manager.yaml` ist die empfohlene, noch offene Benennung. Die endgültige Syntax, Pfad-/Identitätsmigration und die Abbildung heutiger Namespace-Strings sind offen, nicht die Ownershippflicht.
- Rollenauflösung: Referenz auf Person/Team/Rolle und ihr Entscheidungsverfahren; ein eindeutiger Verantwortungsbezug darf keine unklare Mehrheitsentscheidung verbergen.
- Managervertrag: Minimalformat für Kontextpaket, delegierten Ergebnisauftrag, Kindbericht, Integrationsurteil und entscheidungsfähige Eskalation; auf jeder Ebene gleichartig, fachlich unterschiedlich gefüllt.
- Agentenlebenszyklus: Eigener Agent je Manager ist festgelegt; Laufzeit, Fortsetzung und gespeicherter Rollenstand sind noch zu konkretisieren. Eltern dürfen daraus keine Abhängigkeit vom internen Kontext ihrer Kinder erhalten.
- Entscheidungsrahmen: Welche Entscheidungen sind pro Bereich delegiert und welche wichtigen Fragen ausdrücklich dem Nutzer vorbehalten? Breite Routineautonomie ist der Default des Zielmodells; konkrete Projektgrenzen bleiben explizit.
- Modulgrenze: Modellvokabular und Werkzeugpakete benötigen unterscheidbare Verträge. Ein gemeinsames Lieferformat ist eine spätere technische Entscheidung.
- Prozessumfang: Für den ersten Durchlauf genügt ein bewusst begrenzter Workflow. Dauerqueue und unbeaufsichtigter Betrieb sind eigene Nachweise.
- Detailprüfung: Welche Verpflichtungen werden deterministisch geprüft, welche semantisch beurteilt, welche zunächst nur beschrieben? Sichtbare Abdeckung statt pauschalem Enforcement.
- Eigentümerwechsel und Bereichsumbau: Identität, historische Verantwortung, Delegation und Impact bei Verschiebung präzisieren.
- Übergang: Neue öffentliche Syntax und eventuell inkompatible Version ausdrücklich wählen; vorhandene Namen bestimmen nicht das Zielmodell.

Diese Punkte sind konkrete nächste Planungsarbeit. Sie blockieren weder den vorliegenden Entwurf noch verlangen sie eine pauschale neue Freigabe für gewöhnliche Planung.

## 7. Validierung dieses Planungsschritts

Für die Dokumentationsänderung: unabhängige Konzept- und Übergangsreviews, Synthese ihrer Befunde, lokale Prüfung der Markdown-Ziele und `git diff --check`. Keine Produkt-, Runtime-, CI- oder Produktivitätsprüfung wird daraus abgeleitet. Vor Produktcodeänderungen gelten die zum ausgewählten Quellstand passenden Contribution-Gates und zusätzliche fokussierte Prüfungen der Szenarien; keine schwere Vollsuite allein für diesen Plan.

Durchgeführt am 8. Oktober 2026 für den ersten Planungscommit `e325bcb`: zwei unabhängige Planungsreviews und eigener Quellabgleich. Die Rückmeldungen zur Trennung von Rollenverantwortung/Bereichszuordnung sowie zur expliziten Ownership deklarierter Namespaces wurden eingearbeitet. Fünf geänderte Markdown-Dateien enthalten 43 geprüfte lokale Linkziele, davon keines fehlend. Whitespaceprüfung bestanden. Keine Produkt- oder Laufzeittests ausgeführt; A1–A14 waren geplante Abnahmeszenarien.

Die anschließende direkte Nutzerpräzisierung ergänzt den Manager als Abstraktions-, Delegations-, Reporting-, Integrations- und Entscheidungspunkt sowie den Top-down-/Bottom-up-Ablauf und Eskalation zur nächsten befugten Ebene. A15–A21 sind zusätzliche geplante Szenarien, keine ausgeführten Tests. Für diese reine Dokumentationspräzisierung bestanden erneut die Whitespaceprüfung und die Prüfung aller zwölf lokalen Linkziele in den drei geänderten Dateien. Es wurden keine Produkt- oder Laufzeittests ausgeführt.

Die weitere direkte Nutzerpräzisierung legt einen eigenen spezialisierten AI-Agenten pro Manager und unabhängige Arbeitskontexte fest. A16 wird präzisiert, A22–A23 ergänzen die geplante Abnahme. Eine Prüfung dieser Dokumentationsänderung belegt noch keinen tatsächlichen Multi-Agentenbetrieb.

Der ausgearbeitete Shop-Beleg enthält beide Modellstände, fünf Managerrollen und konkrete Beispielartefaktpfade. Der lokale Belegprüfer untersucht Referenzen, Ownership, Bereichsbaum, Kontextvorgaben und den Deltafall. A24–A27 ergänzen die geplante Produktabnahme. Ein erfolgreicher Beleglauf ist keine Ausführung von P1–P4 und keine Prüfung der Beispielanwendung.

Durchgeführt für diese Ausarbeitung: unabhängiger Beispielreview mit anschließendem Nachreview; die Befunde zu expliziter Pflichtabdeckung, neuem Realisierungsbedarf bei unveränderten Begriffen, Writergrenzen, Core-Referenzabbildung und späteren Zuordnungsänderungen wurden eingearbeitet. Der Belegprüfer besteht für 39 Baseline-/44 Zieldefinitionen, fünf Managerrollen und 17 deklarierte Artefaktpfade; er bestätigt fünf neue und fünf geänderte Definitionen einschließlich Realisierungs-/Prüfzuordnung. Sieben Markdown-Dateien mit 51 lokalen Linkzielen bestehen die Zielprüfung. Whitespaceprüfung wird einschließlich aller neuen Dateien vor dem lokalen Commit ausgeführt. Null Shop-Prüfungen, null Beispiel-Managerstarts und kein Deployment; die zur Planung eingesetzten Reviewagenten sind davon getrennt.

Die weitere Nutzerpräzisierung legt `.markitect/` als Ablage aller Markitect-eigenen Dateien sowie Gespräch und lesbare Spezifikation als normalen Zugang fest. `manager.yaml` ist die empfohlene Benennung aus den beiden Nutzervorschlägen. A28–A31 beschreiben die geplante Bedienabnahme. Für diese Dokumentationsfortschreibung wurden 47 lokale Linkziele in fünf Dateien und Whitespace geprüft. Der ältere YAML-Beleg wurde weder migriert noch erneut ausgeführt; keine Produkt- oder Laufzeitprüfung wird daraus abgeleitet.

# Markitect Umsetzungsplan

**Status: Markitect implementiert, Konfyra-Pilot in Prüfung.** Aktualisiert am 30.09.2026. Die Zielreihenfolge bleibt: Markitect als eigenständiges Werkzeug entwickeln, Konfyra auf einem eigenen Branch erproben und abnehmen, anschließend das Cockpit migrieren. Der folgende Arbeitsstand dokumentiert die bereits erfolgte Umsetzung; die Arbeitspakete darunter enthalten weiterhin das vollständige Ziel und seine Abnahmekriterien.

Das [Markitect-Architekturkonzept](markitect-architecture.md) besitzt Ressourcentypen und Formatentscheidungen. Dieser Plan besitzt Arbeitspakete, benötigte Artefakte, Reihenfolge und Abnahme. Go, YAML, einfache Begriffe und eine eigenständig nutzbare lokale CLI sind die Vorgaben. Kubernetes ist ein späterer Adapter desselben Kerns.

## Ausführungsstand vom 30.09.2026

- [Markitect](../../../tools/markitect/README.md) ist ein eigenständig versioniertes Go-Modul unter `tools/markitect`, mit striktem YAML, sechs Inhaltstypen, Projektkonfiguration, Contracts/Bindings, Kontexten, Änderungsfolgen, Renderern und erzeugten YAML-Schemas. Die Implementierung bündelt kleine Anwendungsfunktionen in `internal/app`.
- Die CLI bietet `inventory`, `check`, `verify`, `context`, `impact`, `review`, `format`, `render`, `migrate`, `schema`, `package` und `version`. `verify` führt Konfyras vorhandene Governance-Prüfungen auf einer isolierten Materialisierung desselben Git-Commits aus. `review` speichert einen tatsächlichen AI-Bericht mit Frage, Promptversion, Modell, Aufwand und festen Eingaben oder prüft seine Wiederverwendbarkeit. Es ruft kein Modell auf und überträgt keine menschlichen Freigaben.
- Der isolierte Konfyra-Branch `feature/markitect` basiert auf dem erneut gelesenen lokalen Master `74d18c56e20dbb272dc085a389bfdaf291ee4562`. Seine Testumgebung liegt unter `.worktrees/konfyra`. Der erste feste Pilot-Kandidat ist `d62f84f78f3b92f936de8a4758b1af0af668891c`; weitere Korrekturen folgen als eigene Commits.
- Beim Import wurden die 64 am Ausgangsstand aktiven Mechanismen in YAML überführt. Der Abgleich ergab keine Änderung ihrer Provider-Metadaten. Drei Textkörper wurden bewusst für den neuen Authoring-Ablauf geändert. Weitere Workflow-Abhängigkeiten wurden anhand ihrer Quellformulierungen bewertet und ausdrücklich eingetragen; der Importer liefert solche Links künftig nur als Prüfvorschläge.
- Markdown bleibt die erzeugte Ansicht am bisherigen Linkziel. In Konfyra besitzt Markitect diese Ansichten, der vorhandene Python-Renderer weiterhin die Provider-Dateien. Der reguläre Konsistenz-Einstieg prüft beide Stufen. Konfyra bindet ein durch SHA-256 fixiertes Quellarchiv und denselben Bootstrap-Ablauf für Windows und Linux ein.
- Go-Tests und `go vet` sowie die Bootstrap-Tests wurden unter Windows und in einem temporären Linux-Container mit Go 1.27.1 ausgeführt. Der feste erste Konfyra-Kandidat bestand `verify`. Die neue reine Strukturprüfung lag in drei lokalen Läufen bei rund 1–2 Sekunden; der gesamte Governance-Lauf bei rund sechs Sekunden. Das sind lokale Stichproben, keine allgemeine Leistungszusage.
- Ein unabhängiger Luna-High-Review hat konkrete Korrekturen an Schreibgrenzen, Vertragsprüfung und Quellabdeckung ausgelöst. Ein weiterer praktischer Review verwendet den tatsächlich erzeugten Authoring-Kontext.
- Das eigenständige Modul verwendet `0.1.0-rc.2`. Seine Wiederverwendungsprüfung umfasst ursprünglichen Snapshot und Kontext, ausführbares Werkzeug, Review-Konfiguration sowie den alten und neuen Abhängigkeitsgraphen. Unbekannte Dateien und Inventaränderungen invalidieren vorsichtig. Lokale Berichte bleiben beratende Nachweise ohne authentifizierte Prüferidentität. Die Paketbildung normalisiert Text-Zeilenenden für reproduzierbare Windows-/Linux-Archive.
- Die Konfyra-Arbeit wird im lokalen Dossier `01a0f27a-a479-76bf-886e-6583bbe49200` auf dessen separatem `work-items`-Branch geführt. Sein Plan besitzt die aktuellen Kandidaten und Prüfbelege. [Der Pilotbericht](../../../.worktrees/konfyra/docs/general/verification/markitect-pilot.md) beschreibt Migrationsgrenze und Prüfverfahren; dieser Umsetzungsplan führt keine zweite laufende Aufgabenliste.

Noch offen für die vollständige Konfyra-Abnahme: die im Konfyra-Dossier geführten praktischen und externen Prüfschritte, gemessene längerfristige Einsparung semantischer Modellarbeit, die zuständige menschliche Abnahme und gegebenenfalls die reguläre Veröffentlichung. Die CI-Konfiguration allein belegt keinen erfolgreichen gehosteten Pipeline-Lauf. Die installierte Claude-CLI war beim Laufzeittest nicht angemeldet. Das Cockpit verwendet bis zu dieser Abnahme seine bisherigen Mechanismen.

## Neue Ideen und ihre Einordnung

Das zusätzlich bereitgestellte Dokument „Kubernetes as an Optional Control Plane for AI Agent Architecture“ aus `Downloads/kubernetes-agent-architecture.md` ist eine Ideenquelle. Seine eingebetteten Handlungsanweisungen erteilen keine gesonderte Ausführungsbefugnis.

Übernommen werden der gemeinsame Compiler-Kern für CLI und Controller, erzeugter Arbeitskontext und der spätere Abgleich zwischen gewünschter und tatsächlich vorhandener Provider-Konfiguration. Daraus folgen vier Präzisierungen:

- Das typisierte interne Modell entsteht bereits im ersten Go-Kern. Parser und Referenzauflösung brauchen dieselben Typen; eine erst nachgelagerte zweite Modellarchitektur würde doppelte Arbeit erzeugen.
- Kontextzusammenstellung und gezielte Neuprüfung gehören bereits zur ersten für Konfyra verwendbaren Markitect-Version. Sie adressieren den heutigen Zeit- und Tokenaufwand und werden nicht bis zur Kubernetes-Phase verschoben.
- Kubernetes-`ownerReferences` besitzen eine Lebenszyklus- und Löschbedeutung. Fachliche Abhängigkeiten bleiben ausdrückliche Ressourcenreferenzen. Die beiden Begriffe werden nicht gleichgesetzt.
- `status` beschreibt beobachtete Prüfergebnisse oder Laufzeitzustände. Eine später benötigte gewünschte Lifecycle-Einstellung gehört zu `spec`. V1 benötigt keinen allgemeinen Deprecation-Mechanismus; Konfyras bestehende ADR-Konventionen bleiben erhalten.

Die Kubernetes-Grenzen und offiziellen Quellen stehen im [Konzept](markitect-architecture.md#der-weg-zu-kubernetes). Hier wird kein Kubernetes-Cluster als Voraussetzung für den Pilot eingeführt.

## Ausgangspunkt und Arbeitsgrenze

Der am 30.09.2026 gelesene lokale Konfyra-`master` ist `f3529fa32e4dd9b408b73cf7f29159849de03466`. Im vorhandenen Checkout waren `.azure/work-item-sync.md`, `.azure/work-item-sync.yml` und `tools/work-items/README.md` uncommittet geändert. Diese Dateien sind fremde laufende Arbeit; sie werden nicht zurückgesetzt, gestasht oder automatisch in den Pilot übernommen. Der Stand wird vor der tatsächlichen Umsetzung erneut ermittelt.

Im genannten Commit liefert die direkte Dateisuche unter den Mechanismusordnern 19 Agent-, 25 Skill-, 12 Rule- und 8 Workflow-Kandidaten. Das sind datierte Suchergebnisse, keine vollständige fachliche Klassifikation: Adapter-Mappings können weitere normative Quellen außerhalb dieser Ordner benennen. Ein Inventar muss diese ebenfalls aufnehmen. Erzeugte Provider-Dateien zählen nicht als zusätzliche kanonische Mechanismen.

Konfyras `AGENTS.md`, `docs/general/rules/worktree-workflow.md`, `docs/general/workflows/delivery.md` und `docs/general/workflows/ai-mechanism-authoring.md` bestimmen Isolation und Lieferung. Die spätere Konfyra-Migration verwendet einen eigenen, nicht geschützten Branch in einem dedizierten Worktree. Vorgeschlagener Branchname: `feature/markitect`. Ein bereits existierender geeigneter Worktree wird nach Prüfung bevorzugt; ein belegter Branch wird nicht überschrieben.

Ausgangspunkt ist der bei Beginn festgehaltene lokale `master`, wie vom Nutzer gewünscht. Eine vorherige Aktualisierung oder ein späteres Einbeziehen neuer Master-Commits wird als bewusste Änderung der Grundlage dokumentiert. Zur Integrationsvorbereitung werden Remote- und Zielstand nach Konfyras Verfahren geprüft; der lokale Master wird nicht stillschweigend ersetzt.

Vor der ersten Konfyra-Mutation werden die aktive Person, dokumentierte Zuständigkeit und passende Arbeitsroute dort festgestellt. Die Konfyra-Anwendung wird nach dessen Delivery-Verfahren im zuständigen Git-Work-Item-Dossier geführt. Dieser Plan verweist dann auf dieses Dossier, statt einen zweiten Fortschrittsstand für die Migration zu pflegen. Markitect selbst bleibt ein eigenständiges Vorhaben. Work-Item-IDs werden erst nach tatsächlicher Anlage genannt.

## Markitect als eigenständiges Vorhaben

**Markitect** bedeutet Markdown + Architect. System und künftiges Repository heißen Markitect, CLI und Binary `markitect`. Projektkonfiguration ist `markitect.yaml`, externe Quellbindungen stehen in `markitect.lock.yaml`, lokale Ergebnisse unter `.artifacts/markitect/`.

Markitect entsteht zunächst als eigenständiges Go-Modul unter `tools/markitect/` im aktuellen Cockpit-Workspace. Es muss aus diesem Ordner allein gebaut, getestet und versioniert werden können. Weder Konfyras Produktcode noch Cockpit-Skripte dürfen dafür als versteckte Build-Abhängigkeiten nötig sein. Quellzugriff auf ein Zielrepository geschieht nur über eine ausdrücklich übergebene Wurzel und einen festgehaltenen Stand. Der Entwicklungsort bedeutet keine frühe Migration des Cockpits.

Perspektivisch wird dieses Modul als eigenes Repository **Markitect** herausgezogen. Quellcode, Tests, Schemas, Tool-Dokumentation und Release-Anweisungen bilden dabei eine geschlossene Einheit. Architektur- und Umsetzungsdokumentation ziehen mit; im Cockpit bleiben dann Verweise. Build und Tests müssen auch in einer isolierten Kopie des Moduls funktionieren. Repository-Host, Go-Moduladresse und endgültige API-Domain werden festgelegt, bevor externe Releases oder Kubernetes-Ressourcen veröffentlicht werden.

Konfyra und das Cockpit sind Nutzer desselben Werkzeugs. Sie halten ihre kanonischen Inhalte, Bereichsregeln und projektspezifischen Integrationen selbst. Wiederverwendbare Format-, Provider- und Prüflogik gehört zu Markitect. Konfyra-spezifische Check-Implementierungen bleiben vorerst registrierte Repository-Adapter. Ein neues Profil darf keine zweite Kopie des Markitect-Kerns erzeugen.

## Zu bauende Artefakte

Entwicklungsdateien unter `tools/markitect/` beziehen sich auf den aktuellen Workspace; nach der Extraktion ist dieser Ordner die Wurzel des Markitect-Repositories. Einträge mit `Zielrepo/` gehören erst bei Anwendung nach Konfyra beziehungsweise ins Cockpit. Ressourcen- und Dateiformate werden aus dem Konzept übernommen, nicht erneut hier definiert.

| Artefakt | Vorgeschlagener Ort | Zweck und Lieferzeitpunkt |
|---|---|---|
| Go-Modul und Einstieg | `tools/markitect/go.mod`, `tools/markitect/cmd/markitect/` | Baubares Werkzeug; erstes Arbeitspaket |
| Domänenmodell und Resolver | `tools/markitect/internal/core/` | Typen, Referenzen, Geltung, Bindungen und Diagnosen ohne Kubernetes-/Provider-Abhängigkeit |
| Quellzugriff | `tools/markitect/internal/source/` | Unveränderliche Git-Snapshots und klar als vorläufig markiertes Arbeitsverzeichnis |
| YAML-Leser und API-Typen | `tools/markitect/internal/format/` | Striktes Lesen, Quellpositionen, Ressourcenversion und Normalisierung |
| Prüflogik und Abhängigkeitsauswertung | `tools/markitect/internal/check/` | Strukturchecks und Änderungen im alten/neuen Graphen; nutzt das gemeinsame Modell |
| Kontext- und Nachweiserstellung | `tools/markitect/internal/context/`, `tools/markitect/internal/evidence/` | Vollständiger benötigter Kontext, Eingabemanifeste, Fingerabdrücke und Wiederverwendung |
| Provider-Ausgaben | `tools/markitect/internal/render/` | Genau ein Besitzer pro erzeugter Datei; Driftvergleich und kontrolliertes Schreiben |
| Profile und Beispielprojekte | `tools/markitect/testdata/`, später `Zielrepo/markitect.yaml` | Testbare Bereichszuordnung; die tatsächliche Projektpolitik liegt beim Zielrepository |
| Projektkonfiguration | `Zielrepo/markitect.yaml` | Gewählte Bereiche, Provider und ausdrückliche Bindungen; keine zweite Artefaktliste |
| Erzeugte Ressourcenschemas | `tools/markitect/schema/` | YAML-Ausgaben aus versionierten API-Typen/Validierungsdeklarationen, nicht von Hand synchronisiert |
| Fixtures und erwartete Ausgaben | `tools/markitect/testdata/` und Go-Tests | Positive Beispiele, Fehlerfälle, vorhandene Provider-Semantik und Migrationsparität |
| Pilotquellen | Bei den bestehenden Eigentümern unter `Zielrepo/docs/` | YAML-Fassungen des zusammenhängenden Authoring-Ablaufs |
| Lesbare Ansichten | Bisherige `.md`-Linkziele migrierter Quellen im Zielrepo | Aus YAML erzeugte Dokumentation mit Herkunftsmarker |
| Provider-Projektionen | Bestehende `.agents/`, `.claude/`, `.codex/`-Ziele im Zielrepo | Weiterhin nutzbare Einstiege; keine neuen parallelen Regelkopien |
| Migrations- und Prüfbericht | `.artifacts/markitect/` oder CI-Artefaktspeicher | Generiertes YAML für Inventar, Ausgabevergleich, Abdeckung, Laufzeiten und Nachweise |
| Tool-Dokumentation | `tools/markitect/README.md` und Modul-Dokumentation | Markitect bauen, testen, verwenden und später eigenständig veröffentlichen |
| Repository-Anleitung | `Zielrepo/docs/general/tools/markitect.md` | Version, lokale Aufrufe, Eigentum und Herkunft der erzeugten Dateien |
| Authoring-Anpassung | Bestehender Skill/Workflow samt Dokumentationsregel | Agenten pflegen YAML-Quellen und verwenden den neuen Prüfeinstieg |
| Markitect-Release | Eigenständiger Build-/Test-/Release-Ablauf im Modul | Versioniertes Binary mit Digest für beide Nutzer; unabhängig von Konfyra-Produktbuilds |
| Repository-CI | Bestehender Konsistenz-Runner und Konfyras `.azure/ci.yml`, später Cockpit-CI | Fixierte Markitect-Version auf den jeweiligen Integrationskandidaten anwenden |

Die Paketaufteilung bezeichnet Verantwortlichkeiten. Kleine Funktionen dürfen zunächst gemeinsam liegen; es werden keine leeren Framework-Pakete oder Interfaces ohne Austauschbedarf erzeugt. Go-Module und Go-Quellcode sind Build-Artefakte. Alle vom neuen Werkzeug eingeführten eigenen Daten- und Konfigurationsdateien verwenden YAML.

## Arbeitspaket 1 Markitect aufbauen

1. Das eigenständige Go-Modul `tools/markitect/` mit README, CLI-Einstieg und Fixtures anlegen. Modulversionierung und ein vom restlichen Workspace unabhängiger Build gehören zum Grundgerüst.
2. Feste Konfyra-Stände lesend untersuchen und repräsentative Struktur-/Provider-Eigenschaften als abgegrenzte Fixtures erfassen. Tests brauchen keinen Zugriff auf einen bestimmten lokalen Konfyra-Pfad und übernehmen keine kundengebundenen Produktinhalte als allgemeine Beispiele.
3. Einen Snapshot-Reader und einen ersten `markitect check --revision COMMIT` bauen. Er kann vorhandene Checks über ausdrücklich konfigurierte Adapter auf einer isolierten Materialisierung aufrufen und meldet die erreichte Abdeckung.
4. Canonical-/Provider-Inventar, Diagnosen und Prüfergebnisse als YAML ausgeben. Laufzeit, Quellcommit und aufgerufene Checks bilden die erste Vergleichsgrundlage.
5. Reproduzierbare Fälle für Textänderung, gemeinsame Regel, neue Datei, Umbenennung, Adapter-Manipulation und bewegten Zielstand festlegen.

**Abnahme:** Markitect lässt sich unabhängig bauen und prüft denselben Commit reproduzierbar. Änderungen in einem fremden Arbeitsverzeichnis verändern keinen laufenden Check. Ein fehlender Runner oder eine nicht erfasste Quelle erzeugt keinen unbegründeten grünen Gesamtstatus. Konfyra ist in diesem Arbeitspaket eine lesende Referenzquelle und wird noch nicht migriert.

## Arbeitspaket 2 Markitect Ressourcen und Referenzen

Der Kern erhält die im Konzept beschriebenen Typen. Die erste für Konfyra verwendbare Version unterstützt `Text`, `Rule`, `Workflow`, `Skill`, `Agent`, `Contract` und `Project` für die Konfiguration. Implementiert wird in kleinen Schritten; Contracts und Bindungen werden an isolierten Review-Beispielen geprüft, bevor die Konfyra-Migration beginnt. Schemafehler, doppelte Namen, unzulässige Felder, falsche Zieltypen und Bereichsverletzungen liefern stabile Diagnosen mit Quelle.

Legacy-Leser bilden vorhandene Markdown-/Frontmatter-Quellen auf dasselbe Modell ab. Sie sind Migrationsadapter. Sie erfinden aus jedem Markdown-Link keine normative Abhängigkeit. Unklare Beziehungen werden als Migrationslücke ausgegeben und fachlich eingeordnet. Migrierte YAML-Quellen und ihre erzeugten Markdown-Ansichten werden im Inventar nur einmal kanonisch gezählt.

**Abnahme:** Die Fehlerfixtures werden ohne Modellaufruf erkannt. Namens- und Typauflösung funktionieren vor der ersten echten Quellenkonvertierung. Der Bericht zeigt, welche Bereiche vollständig typisiert und welche nur über Legacy-Adapter erfasst sind.

## Arbeitspaket 3 Markitect Kontext Ausgabe und Nachweise

`markitect context` erstellt einen auf den Snapshot gebundenen Arbeitskontext mit Symbol, Pfad, Inhalt, Fingerabdruck und Aufnahmegrund. Es löst Bereichsregeln, konkrete Abhängigkeiten, Contracts und die ausgewählten Implementierungen vollständig auf. Ein Größenlimit darf Pflichtquellen nicht abschneiden; größere Aufgaben werden vollständig aufgeteilt oder als unvollständig gemeldet.

`markitect impact` vergleicht alten und neuen Graphen. Nachweise erfassen neben gelesenen Dateiinhalten die abgefragten Verzeichnisse, Trefferlisten, fehlenden erwarteten Dateien, Werkzeugversion, Profile, Bindungen und externe Stände. Eine neue Regel und eine entfernte Abhängigkeit gehören zu den verpflichtenden Tests. Ein beobachtetes Agent-Leseset ist keine vollständige Abhängigkeitsdefinition.

Zunächst werden schnelle Strukturchecks weiterhin vollständig ausgeführt. Wiederverwendung fokussiert die teuren semantischen Reviews. Jedes wiederverwendete Ergebnis muss seine Frage, Eingaben und Review-Konfiguration nennen. Nicht belegte Unabhängigkeit führt zu einer breiteren Prüfung.

**Abnahme:** Eine parallele Änderung verfälscht keinen laufenden Review. Eine lokale Änderung erzeugt einen nachvollziehbar begrenzten neuen Auftrag; eine allgemeine Regeländerung invalidiert alle davon abhängigen Nachweise. Wiederholte Prüfung unveränderter semantischer Eingaben benötigt keinen erneuten Modellaufruf, sofern das betreffende Review-Profil Wiederverwendung erlaubt.

Provider-Renderer werden hier an Fixtures und ausdrücklich ausgewählten festen Quellständen geprüft. Bestehende Einstellungen zu Modell, Aufwand, Werkzeugen und Berechtigungsgrenzen müssen erhalten bleiben. Markitect besitzt jeweils nur die ihm zugeordneten Ausgaben und kann Drift ohne Schreibzugriff melden. Schreibtests verwenden isolierte Testverzeichnisse.

## Übergang Markitect ist für Konfyra verwendbar

„Markitect steht“ bedeutet für den ersten Einsatz:

- Das Go-Modul lässt sich unabhängig bauen und testen.
- Die sechs Inhaltstypen und Projektkonfiguration sind in YAML lesbar und geprüft.
- `check`, `context`, `impact` und `render --check` arbeiten auf festen Quellständen; der kontrollierte Render-Schreibweg ist in Fixtures getestet.
- Referenzen, Bereichsgrenzen, Contracts und Bindungen besitzen positive und negative Tests.
- Provider-Ausgaben und lesbare Markdown-Ansichten sind erzeugbar und gegen Drift prüfbar.
- Neue und entfernte Abhängigkeiten invalidieren die zutreffenden Nachweise.
- Legacy-Migrationsadapter und unbekannte Fälle werden sichtbar ausgewiesen.
- Eine festgelegte Binary-Version mit Herkunft und Digest ist für den Konfyra-Pilot verfügbar.

Diese Version ist eine nutzbare erste Lieferung, keine Behauptung einer fertigen Kubernetes-Plattform. Die Konfyra-Anwendung beginnt erst nach dieser Werkzeugprüfung. Das spätere eigene Repository ist vorbereitet, seine externe Veröffentlichung ist kein künstliches Hindernis für den lokalen Pilot.

## Arbeitspaket 4 Markitect auf Konfyra anwenden

Jetzt werden Konfyras Arbeitskontext und Zielstand erneut geprüft, der dedizierte Branch/Worktree angelegt und die vorhandenen Governance-Checks dort vermessen. Die ausgewählte Markitect-Version wird angebunden. Erst hier beginnt die echte Quellenmigration. Der erste vertikale Ablauf besteht aus:

- `docs/general/skills/author-ai-mechanism.md`,
- `docs/general/workflows/ai-mechanism-authoring.md`,
- den tatsächlich benötigten Regeln, zunächst Dokumentations- und Arbeitsgrenzen aus dem ermittelten Kontext,
- `docs/general/agents/documentation-specialist.md`,
- einem gezielten Review-Vertrag, wenn die bestehende Verantwortung dafür fachlich passt, und einem einfachen Text-Beispiel.

Die verbindliche Quellmenge wird aus dem Inventar festgelegt; sämtliche transitiv erforderlichen Quellen bleiben im Snapshot enthalten. Noch nicht migrierte Ziele dürfen über den gekennzeichneten Legacy-Leser aufgelöst werden. Der Pilot darf keine Regeln auslassen, nur um klein zu wirken.

Jede Migrationseinheit umfasst YAML-Quelle, erzeugte Markdown-Ansicht am bisherigen Linkziel, Router, Provider-Ausgaben und erforderliche Checker-Anpassung. Der alte Inhalt wird nicht unabhängig weitergepflegt. Die geltende Konfyra-Regel „direkte Markdown-Datei als kanonischer Mechanismus“ muss beim ersten YAML-Einsatz ausdrücklich an ihrem Eigentümer angepasst werden; ein Tool darf sie nicht still umgehen.

Das Review-Beispiel verbindet `needs`, `implements` und eine ausdrückliche Projektbindung. Strukturchecks prüfen Typ und Signatur. Zwei Fixtures zeigen austauschbare Implementierungen; produktiv wird nur die tatsächlich gewählte Bindung geladen. Ein fehlender Vertrag oder eine nicht auflösbare Bindung erscheint als Fehler beziehungsweise unvollständiger Arbeitskontext.

Provider-Ausgaben werden mit dem Ausgangsstand verglichen. Bestehende Angaben zu Modell, Aufwand, Werkzeugen, Schreibgrenzen und etwaigen Laufzeitlimits dürfen nicht verlorengehen. Ein beabsichtigter Unterschied wird einzeln erklärt. Der neue Renderer erzeugt nur seine zugeordneten Ausgaben; der alte Renderer bleibt für den übrigen Bestand zuständig.

**Abnahme:** Der bestehende Authoring-Auftrag funktioniert für die im Repository unterstützten Provider mit den migrierten Quellen. Das Werkzeug findet den Einstieg, lädt alle benötigten Regeln und erzeugt passende Ausgaben. Ein begleiteter realer Agentlauf prüft die Nutzbarkeit; ungetestete Provider-Laufzeit wird nicht als bestätigt dargestellt. Strukturelle Parität allein ist keine Verhaltensabnahme.

## Arbeitspaket 5 Konfyra vollständig im vereinbarten Umfang migrieren

Nach dem Pilot werden die übrigen ermittelten Agents, Skills, Regeln und Workflows in kohärenten Bereichen migriert: gemeinsame General-Quellen zuerst, anschließend Core, Modules und Products nach ihren tatsächlichen Abhängigkeiten. Ein Bereich kann auf bereits typisierte oder offen gekennzeichnete Legacy-Quellen verweisen; diese Restliste muss vor Konfyra-Abnahme aufgelöst sein.

„Konfyra migriert“ bedeutet hier: Alle im Inventar als aktive AI-Mechanismen und zugehörige normative Konfiguration klassifizierten Quellen verwenden das neue Modell oder eine ausdrücklich abgegrenzte, belegte externe Quelle. Es bedeutet nicht, jede ADR, jeden Codevertrag, jedes Produkt-Handbuch und alle Work-Item-Dossiers pauschal in YAML umzuschreiben. Deren Links und relevante Inhalte bleiben geprüft und werden bei Bedarf in den Arbeitskontext aufgenommen.

Insbesondere sind die vorhandenen Work-Item-Formate und ihre Synchronisierung kein ungefragter Teil dieser Formatmigration. Der neue Validator kann deren Nachweisprinzipien nutzen und Quellen referenzieren. Er ersetzt nicht die fachliche Zuständigkeit des Work-Item-Tools.

Pro Bereich werden die owning Dokumentation, lokale Router, Authoring-Hinweise, Provider-Mappings und Regressionstests gemeinsam angepasst. Der erste globale Quellenindex ist generiert; es entsteht kein zweites manuell gepflegtes Register. Ausschluss- und Legacy-Einträge müssen begründet und sichtbar sein. Ein unklassifiziertes aktives Artefakt verhindert die Behauptung vollständiger Migration.

**Abnahme:** Das Inventar weist für den festgelegten Umfang keine ungeklärten oder doppelt kanonischen Quellen mehr aus. Alle verwalteten Ausgaben stammen von eindeutig zugeordneten Quellen. Vorhandene Code-/Dokumentationschecks bleiben wirksam, selbst wenn sie vorläufig über den bestehenden Python-Runner laufen.

## Arbeitspaket 6 Konfyra bewähren und liefern

Der bestehende Konsistenz-Runner und `.azure/ci.yml` binden die neuen Checks ein. Go-Tests, Formatprüfung, reproduzierbare Schema-/Adapter-Erzeugung und Regressionen laufen zusätzlich zu den weiterhin zuständigen Prüfungen. Breitere Produkt-, Work-Item- oder Umgebungsgates folgen dem tatsächlichen Änderungsumfang und Konfyras Delivery-Regeln; sie werden weder pauschal abgeschaltet noch ohne Anlass gestartet.

Die Anwendung wird mit den festgelegten Änderungsszenarien am migrierten Branch geprüft. Verglichen werden Dauer, Umfang des Review-Kontexts, Eingabetokens und erkannte Fehler mit der Baseline. Ein gemischter Integrationskandidat wird am aktuellen Zielstand geprüft. Die unabhängige Review- und menschliche Abnahmeroute bleibt die des Repositories.

Vor einer Übernahme ins Cockpit liegt ein verwendbares, eindeutig versioniertes Binary für die benötigten Plattformen vor. Windows ist Pflicht für den lokalen Workflow; Linux wird entsprechend dem tatsächlich verwendeten CI-Runner geprüft. Binary-Herkunft, Version und Digest werden dokumentiert. Als erster Verteilungskanal reicht ein versioniertes CI-Artefakt mit einem stabilen Bezug auf den Quellcommit. Ein frei driftendes „latest“ reicht nicht.

**Konfyra-Abnahme:** Der vereinbarte Quellenumfang ist migriert, die erforderlichen Gates bestehen, reale Authoring-/Review-Aufträge funktionieren, Änderungsfolgen und Parallelarbeit sind getestet, die Einsparungen sind gemessen und die zuständige Abnahme ist erfolgt. Ein positives Syntaxergebnis oder ein unbewerteter Prototyp allein erfüllt diesen Übergang nicht. Der erste Cockpit-Einsatz verwendet die in Konfyra abgenommene Markitect-Version aus der eigenständigen Toolquelle. Die Konfyra-Abnahme kann auf einem unveränderlichen Branch-Kandidaten erfolgen; eine Integration nach `master` folgt separat der geltenden PR- und Freigaberoute. Danach wird der tatsächlich gelieferte Commit verifiziert.

## Arbeitspaket 7 Das Cockpit ausstatten

Erst nach Konfyra-Abnahme wird das Cockpit auf dieselbe geprüfte Markitect-Version migriert. Dass Markitect zunächst in diesem Workspace entwickelt wurde, ersetzt diese gesonderte Anwendung nicht. Der gemeinsame Go-Kern bleibt unverändert; abweichende Geltung wird im Cockpit-Profil modelliert. Wenn dafür eine Kernänderung nötig wird, erhält sie gemeinsame Regressionen für beide Profile und eine neue Werkzeugversion.

Benötigt werden das Cockpit-Profil, `markitect.yaml`, die Git-/Prüfumgebung des Cockpits und anschließend die Migration seiner General-, Consiliari-, Kunden- und Projektmechanismen. Die Einrichtung eines Remotes oder CI-Zielsystems ist eine eigene konkrete Ablageentscheidung; sie wird nicht erfunden. Lokale Git-Snapshots ermöglichen zuvor bereits reproduzierbare Prüfungen.

Die Abnahme ergänzt zu den Konfyra-Szenarien: Arbeit ohne Projektbezug, Kundentrennung, projektspezifische Regeln, Konflikte mit allgemeinen Anforderungen und externe Konfyra-Verweise auf festgehaltene Versionen. Ein Import darf nicht sämtliche Konfyra- oder Kundenregeln global laden.

**Abnahme:** Das Cockpit benutzt denselben Kern, bietet geprüfte Kontexte auf jeder Ebene und erzeugt passende Provider-Einstiege. Fremde Kundeninhalte und Projektregeln erscheinen nicht im falschen Arbeitskontext. Für den zweiten Einsatz wird keine zweite Toolimplementierung aufgebaut.

## Spätere Kubernetes Artefakte

Nach den beiden lokalen Einsätzen können versionierte CRDs, ein Controller-Einstieg, RBAC-/Installationsmanifeste und API-Server-Tests entstehen. Alle eigenen Manifeste verwenden YAML. Der Controller ruft den bereits bewährten Kern auf und berichtet beobachteten Zustand; er erfindet keine zweite Validierungslogik.

Zusätzlich benötigt der Controller eine Veröffentlichung vollständiger unveränderlicher Pakete oder Git-Stände. Einzelne nacheinander aktualisierte Custom Resources bilden keinen gemeinsamen atomaren Quellstand. Status und Prüfnachweis müssen den tatsächlich aufgelösten Paketstand ausweisen. Erst ein konkreter Betriebsbedarf rechtfertigt automatische Reconciliation von Laufzeitkonfiguration.

## Messung und Stoppkriterien

| Prüffrage | Erwarteter Nachweis |
|---|---|
| Sind alle Quellen erfasst? | Generiertes Inventar mit Herkunft, Klassifikation und sichtbaren Lücken |
| Bleibt bestehendes Verhalten erhalten? | Ausgabevergleich plus begleitete Provider-/Authoring-Aufträge |
| Ist Strukturprüfung schnell genug? | Baseline und neue Zeiten auf derselben Umgebung; Ziel für den reinen neuen Strukturcheck zunächst höchstens fünf Sekunden am Pilotbestand, noch keine Zusage |
| Sparen wir Modellarbeit? | Vergleich identischer Aufgaben, Kontextgrößen und Eingabetokens; unveränderte Reviews ohne neue Modellanfrage |
| Können wir parallel arbeiten? | Snapshot während laufender Bearbeitung unverändert; gezielte Neuprüfung des Integrationskandidaten |
| Werden neue und entfernte Abhängigkeiten erkannt? | Negative Fixtures für neue Regeln, gelöschte Referenzen und geänderte Bindungen |
| Werden Fehler vollständig gemeldet? | Fehlender Runner, unbekannte Felder und unvollständiger Kontext erzeugen keinen grünen Gesamtstatus |

Bei verlorenen Provider-Einstellungen, nicht erfassten Abhängigkeiten, verdeckter Kundenüberschreitung oder widersprüchlichen kanonischen Quellen stoppt die betreffende Migrationseinheit. Sie wird vor einer weiteren Ausweitung repariert. Rückkehr zu einem vorherigen Stand erfolgt über kontrollierte Branch-Commits, nicht durch Löschen fremder Arbeit. Das Binärformat, die Quellen und die erzeugten Ausgaben werden als zusammengehöriger Kandidat geprüft.

## Womit wir anfangen

Der erste konkrete Auftrag lautet: **Markitect als eigenständiges Go-Modul aufbauen, die YAML-Ressourcen und Snapshot-Prüfung implementieren und seine Eignung mit isolierten Fixtures belegen.**

Die erste kleine Lieferung besteht aus `tools/markitect/`, einem baubaren `markitect`-Binary, YAML-Leser, Quellstandbindung, Diagnosen und Tests. Darauf folgen Resolver, Provider-Ausgaben, Kontext und Änderungsfolgen bis zum beschriebenen Übergang. Danach beginnen wir die geplante Konfyra-Migration auf eigenem Branch. Anschließend wird das Cockpit Nutzer derselben Markitect-Version.

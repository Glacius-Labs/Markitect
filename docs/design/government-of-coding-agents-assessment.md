# Markitect-Produktzweck und Government: Prüfung vor Designentscheidung

Stand: 2026-10-07, Europe/Berlin. **Diskussions- und Evaluationsentwurf; keine implementierte Funktion und kein Auftrag zum sofortigen Umbau.**

**Checkpoint erreicht:** Der Architect hat den begrenzten Grundmodell-Schritt auf `1ea5c76f55526fc4d721e865885436153f48b497` abgeschlossen. Der [Abschlussabgleich](architect-government-checkpoint.md) dokumentiert 34 bestandene Windows-Gates, die endliche Pilotfolge und verbleibende Grenzen. Die drei vorbereiteten Entwürfe sind auf den finalen Stand abgeglichen; weitere Umsetzung wartet auf die Government-Entscheidung. Die folgenden früheren Arbeitsanweisungen und die Bestandsaufnahme auf `2c5299e` bleiben als Entwicklungskontext erhalten.

## Verbindlicher Arbeitsauftrag und Reihenfolge

Der Nutzer hat folgenden nächsten Entscheidungspunkt festgelegt:

1. Der Architect schließt seinen derzeitigen Grundmodell-Schritt mit einem soliden, nachvollziehbar validierten Stand ab.
2. An diesem Stand halten wir mit weiterem Ausbau an. Zuerst bestimmen wir Markitects Problem, Produktzweck und Zielbild; daran prüfen wir die Government-Richtung und entscheiden gemeinsam über das weitere Vorgehen. Spätere ausdrückliche Nutzeraufträge im Architect-Chat werden dabei berücksichtigt; laufende autorisierte Arbeit wird nicht durch eine veraltete Wiedervorlage unterbrochen.
3. **Government als Produktidentität ist eine offene Hypothese, keine bevorzugt beschlossene Zukunft.** Der Nutzer hat die Reihenfolge ausdrücklich korrigiert: zuerst verstehen, wofür Markitect gebaut wird, dann prüfen, ob Government dazu passt oder ein anderes Produkt daraus macht. Falls die Metapher gewählt wird, soll sie durchgängig stimmig sein; diese Gestaltungspräferenz ersetzt keine Nutzenprüfung.
4. Der Nutzer stellt sein Kabinett aus angebotenen, wiederverwendbaren Ministerien zusammen. **Eine Änderung wird nur akzeptiert, wenn jedes ausgewählte Ministerium der endgültigen Fassung ausdrücklich zugestimmt hat.**
5. Jev/TypeSafe AI bleibt eine nachrangige Untersuchung. Zuerst erfolgt die Government-Entscheidung; Jev wird nicht in den laufenden Architect-Schritt aufgenommen.

Jetzt autorisiert sind Bestandsaufnahme und Designvorbereitung. Der komplette Entwurf unten ist noch keine angenommene neue Architektur. Insbesondere werden aktuelle Vision, Core-Verträge, veröffentlichte Versionen und der aktive Architect-Kandidat durch diese Notiz nicht umgeschrieben.

**Neueste Arbeitsanweisung:** Der Nutzer hat ausdrücklich freigegeben, die Architekturvorbereitung bereits parallel zu den laufenden Architect-Prüfungen auszuführen. Dazu entstehen auf diesem separaten Dokumentationsbranch das [Betriebsmodell](delegated-engineering-operating-model.md), die [Änderungslandkarte](delegated-engineering-change-map.md) und der [Validierungsplan](delegated-engineering-validation-plan.md). Ihre vorläufige technische Vergleichsbasis ist der inzwischen festgeschriebene Architect-Kandidat `dc10f5454af381887a2f41004d0987294bd1fe62`. Nach seinem Abschluss wird gezielt auf den tatsächlichen finalen Kandidaten abgeglichen. Damit ist die Vorbereitung jetzt beauftragt; ein Produktionsumbau oder eine Veröffentlichung folgt daraus nicht.

Ein grüner Einzeltest, ein inaktiver Chat oder ein erfolgreicher Teilversuch sind kein automatischer Nachweis eines soliden Gesamtstands. Für den Checkpoint brauchen wir einen benannten Kandidaten, den Abschluss des aktuellen begrenzten Ablaufs, zugehörige Quellprüfungen und eine klare Liste verbleibender Grenzen. Weitere Experimente werden nicht beliebig an diesen Abschluss angehängt.

## Vergleichsbasis

Untersucht wurde der separate Architect-Worktree `C:/Users/Consiliari/.codex/worktrees/standard-operating-model/Markitect`, Kandidat `2c5299e0a876ba491e6d9fc93b8965fe21d62917` (`codex/standard-operating-model`). Quellverträge wurden zusätzlich mit `git show` aus dieser Revision gelesen. Diese Notiz liegt auf einem eigenen Dokumentationsbranch des ursprünglichen Checkouts; dessen ältere `main` ist **nicht** die technische Vergleichsbasis.

Die ursprüngliche Analyse ist keine erneute Testsuite oder Releasefreigabe. Der spätere finale Stand und seine abgeschlossenen Prüfungen sind gesondert im [Abschlussabgleich](architect-government-checkpoint.md) dokumentiert.

Wichtige Quellanker relativ zum untersuchten Kandidaten:

| Quelle | Beobachteter Vertrag |
|---|---|
| `docs/vision.md` | Kanonische Projektwelt, autonome Umsetzung und menschliche Verantwortung für wesentliche Entscheidungen |
| `docs/design/standard-operating-model.md` | Government bisher als Analogie; ausführbarer endlicher Projektions-, Prüf-, Reparatur- und Auditablauf |
| `docs/development/modules.md` | Installierbare Pakete strikt Schema oder Projection; Host komponiert technische Fähigkeiten |
| `internal/core/types.go`, `internal/core/compile.go` | Struktureller, providerunabhängiger Schema-/Definition-Compiler |
| `internal/host/canonical_controller.go` | Ein konfigurierter Executor und Verifier; Assurance-Scopes sind jeweils einer Projection zugeordnet |
| `internal/host/canonical_controller_execution.go` | Zusammengestellte, gebundene Änderungskandidaten |
| `internal/host/canonical_controller_verification.go` | Frische technische Verifikation mit Nachweis- und Eingabebindung |
| `internal/host/agentexec/types.go` | Agentenauftrag trägt ModulePin und ProjectionID; Antwort enthält Beobachtungen und Evidenzreferenzen |
| `internal/host/assurance/assurance.go` | Endlicher Graph für lokale und übergeordnete Prüfungen |
| `internal/host/records/records.go`, `internal/host/recordstore/` | Herkunft, Materialisierung, Verifikation und betriebliche Historie |
| `internal/host/canonical_controller_audit.go` | Abschluss für deklarierte Projektionen, aktuelle Nachweise und offene Arbeit |
| `docs/canonical-projections.md`, `docs/usage.md` | Explizite Alpha-Aktionen; Materialisierung als Evidence-Commit bewegt HEAD und Index nicht |

## Produktzweck vor Metapher

Die ältere Produktbeschreibung beginnt bei wartbarem AI-facing Engineering-Wissen: Regeln, Workflows, Skills, Agentenanweisungen und Verträge erhalten kanonische Eigentümer, explizite Beziehungen, passenden Kontext und nachvollziehbare Änderungsfolgen. Das Problem ist nicht nur fehlendes Review, sondern verteiltes, widersprüchliches oder vergessenes Wissen und unklarer Nachweisstand.

Der aktuelle Architect-Entwurf entwickelt diesen Ansatz weiter. Seine kanonische Vision lautet: **Die gewünschte Projektwelt einmal definieren und ihre Darstellungen mit dieser Absicht in Einklang halten.** Menschen verantworten Ziele, Architektur, Grenzen, Regeln und echte Entscheidungen. Agenten setzen innerhalb dieser Vorgaben um, prüfen und reparieren. Markitect verbindet akzeptierte Absicht, relevante Folgen, betroffene Darstellungen und aktuelle unabhängige Nachweise.

Das ist eine Entwicklung vom strukturierten Engineering-Wissen zum ausführbaren Sollmodell. Sie ist bereits im aktuellen Entwurf enthalten und wurde nicht erst durch die Ministeriumsidee eingeführt. Ob sie im Alltag genügend Qualität und Entlastung bringt, bleibt zu untersuchen. Der untersuchte Kandidat enthält begrenzte Ausführungs-, Prüf-, Reparatur- und Auditfunktionen sowie synthetische Durchläufe; er belegt weder allgemeine autonome Nutzbarkeit noch eine vollständig überwachte reale Codebasis. Die aktuellen Alpha-Verträge sind getrennt vom veröffentlichten Produkt zu behandeln.

| Problem | Markitects vorgesehener Mechanismus | Beitrag beziehungsweise Grenze von Ministerien |
|---|---|---|
| Dieselbe Regel widerspricht sich in Dokumentation, Agentenanweisungen und Umsetzung. | Ein akzeptierter semantischer Eigentümer; abhängige Darstellungen statt konkurrierender Wahrheit. | Ressorts können Abweichungen erkennen. Eigene unabhängige Regelkopien würden das Problem wieder einführen. |
| Eine Änderung vergisst betroffene Nachbarbereiche. | Explizite Beziehungen, konservativer Impact und Umsetzung aller betroffenen Darstellungen. | Mehrere fachliche Blickwinkel können Lücken finden; Zustimmung allein leitet fehlende Arbeit nicht zuverlässig ab. |
| Eine notwendige Darstellung existiert noch gar nicht. | Gewünschte Darstellungen mit beobachtetem Bestand vergleichen und fehlende Arbeit planen. | Ein reines Diff-Review kann das Nichtvorhandene übersehen. |
| Bestehende Dateien driften ohne neuen Änderungsauftrag. | Beobachten und gegen unveränderte akzeptierte Absicht reparieren. | Ein Abstimmungsverfahren braucht zusätzlich einen Auslöser für Beobachtung und Reparatur. |
| Eine Implementierung wird nur anhand eigener Tests freigegeben. | Unabhängige Prüfung mit passender Evidenz und eigener Integrationsverantwortung. | Fachlich getrennte Ressorts können Prüfbreite erhöhen. Mehr Stimmen oder Modellaufrufe beweisen keine unabhängige Wahrheit. |
| Menschen koordinieren jede Routineänderung und synchronisieren Regeln von Hand. | Wiederverwendbare Vorgaben, begrenzte Ausführung, Reparatur und gezielte Eskalation. | Klare Mandate können entlasten; obligatorische Einstimmigkeit kann auch zusätzliche Runden und vermeidbare Eskalationen erzeugen. |

**Drei Ebenen sind getrennt zu entscheiden:** Die Regierungsmetapher ist eine Sprache für das Produkt. Ressorts sind eine mögliche Organisation fachlicher Verantwortlichkeiten. Einstimmigkeit ist eine konkrete Annahmepolitik. Keine dieser Entscheidungen folgt automatisch aus den anderen. Die Möglichkeit einer bestehenden gemeinsamen Ursache oder eines ausgelassenen Ressorts bleibt auch bei einstimmiger Zustimmung bestehen.

### Konkretes Referenzszenario: ein ungeordnetes bestehendes Repository

Der Nutzer hat den gewünschten Einsatz präzisiert: ein aktiv entwickeltes, wenig dokumentiertes Repository mit zunächst unbekanntem Stack, Frontend, Backend, CI/CD, verschiedenen Bereichen sowie Stories, Tasks und Ideen. Ein Teil funktioniert, ein Teil nicht. Im eigenen Branch soll zuerst verstanden und geordnet werden, was existiert und gewollt ist. Danach sollen Agenten das Repository schrittweise konsolidieren und offene Arbeit unter einem belastbaren Verfahren umsetzen. Ziel sind ein nachvollziehbar strukturiertes Repository und überwiegend autonome Entwicklung mit wenig Routineaufsicht. Dies ist ein gewünschtes Anwendungsszenario, kein aktuelles Liefer- oder Zuverlässigkeitsversprechen.

Hier wäre Government umfassender als eine Sammlung von Prüfern. Es würde Bestandsaufnahme, Zielbildung, Priorisierung, Umsetzung, Ressortprüfung, Reparatur und Betrieb organisieren. Markitect könnte das verbindliche Projektmodell und den nachvollziehbaren Abgleich tragen. Diese Verbindung ist plausibel zu untersuchen, entscheidet aber noch nicht über die Produktidentität.

#### Haupterwartung: kontinuierliche autonome Entwicklungsarbeit

Der Nutzer präzisiert den zentralen Produktnutzen: Nach Überführung und Modellierung soll eine große Menge offener Arbeit in eine kontinuierlich arbeitende Maschine eingegeben werden können. Expliziter beobachteter Bestand, akzeptiertes kanonisches Modell, Engineering-Methodik, AI, Koordination und Parallelisierung sollen gemeinsam ein sauberes, einheitliches und gut verifiziertes Repository hervorbringen, bei möglichst wenig laufender Aufsicht. Einrichtung und Modellierung sind der Aufbau; über längere Aufgabenfolgen zuverlässig erledigte Arbeit ist der erwartete Nutzen. Dies ist ein Zielbild, keine Behauptung, dass der aktuelle Kandidat diese Maschine bereits liefert.

Der erforderliche Betriebsablauf umfasst Arbeitsaufnahme und Klärung innerhalb der delegierten Ziele; Zerlegung und Reihenfolge nach Abhängigkeiten; parallele Umsetzung tatsächlich unabhängiger Arbeit; getrennte Prüfung; Zusammenführung und eigene Integrationsprüfung; begrenzte Reparatur; Wiederaufnahme nach Unterbrechungen; sowie Aktualisierung von Arbeitsstand, betroffenen Darstellungen und Nachweisen. Ergebnisse einzelner Agenten müssen gegen den aktuellen gemeinsamen Stand zusammenpassen. Gleichzeitige Agentenaufrufe allein belegen diese Fähigkeit nicht.

Routinearbeit innerhalb der vereinbarten Befugnisse soll ohne einzelne menschliche Freigaben fortgesetzt werden können. Konkrete Entscheidungen außerhalb dieser Befugnisse werden gebündelt und verständlich vorgelegt; unabhängige Arbeit kann währenddessen weitergehen. Ein bloß wiederholtes Nachfragen nach gewöhnlichen Implementierungsdetails oder ein durch notwendige manuelle Integration verdeckter Stillstand verfehlt diese Erwartung.

Am Architect-Checkpoint ist daher neben dem Grundmodell ausdrücklich die Distanz zu diesem Betriebsziel zu bestimmen. Maßgeblich sind eine vorab benannte realistische Aufgabenfolge, funktional und architektonisch bewertete Ergebnisse, vollständige betroffene Arbeit, Integration, verbleibende Fehler, Wiederaufnahme und tatsächlich benötigte menschliche Koordination einschließlich Modellpflege. Weder grüne Einzelaufgaben noch größere Agentenzahl oder ein formal vollständiges Kabinett reichen als Nachweis. Government bleibt eine mögliche Organisation dieser Arbeit; die Produktidentität ist weiterhin offen.

Vier Dinge müssen unterscheidbar bleiben; sie erfordern nicht automatisch vier neue Core-Ressourcentypen:

| Gegenstand | Bedeutung |
|---|---|
| Beobachteter Bestand | Stack, Komponenten, Abhängigkeiten, Pipeline- und Laufzeitbefunde, dokumentierte Entscheidungen, unbekannte Bereiche und Unsicherheit; jeweils mit Herkunft. Eine vorhandene Konvention ist zunächst eine Beobachtung. |
| Akzeptiertes Zielbild / Projektordnung | Was das Projekt leisten soll, gewünschte Grenzen, Verantwortlichkeiten, Qualitätsmaßstäbe und Entscheidungsbefugnisse. Geltendes Soll und noch nicht beschlossene Vorschläge bleiben unterscheidbar. |
| Backlog und Übergangsplan | Produktarbeit, Defekte, Konsolidierung, fehlende Nachweise und Abweichungen vom Soll; mit Abhängigkeiten, Priorität und überprüfbaren Erledigungskriterien. Nicht jede Idee ist ein angenommener Auftrag. |
| Betriebs- und Prüfevidenz | Was an welchem Stand untersucht, geändert und geprüft wurde, welche Einwände offen sind und welche Grenzen verbleiben. Nachweise legitimieren keine neue Absicht. |

Eine Verfassung kann Ziele und tragende Regeln ausdrücken; sie sollte nicht automatisch jede Datei oder jedes gewöhnliche Implementierungsdetail festschreiben. Ein zugehöriges Projektmodell kann die benötigten konkreten Beziehungen führen. Die genaue sprachliche Aufteilung ist offen. Das Zielbild ist nicht einfach eine Abschrift des chaotischen Ist-Zustands.

Der Übergang wäre schrittweise: Bestand und funktionierendes Verhalten erfassen; ein erstes begrenztes Zielbild bestätigen; Build-/Test-/Pipeline-Grundlagen und besonders hinderliche Lücken stabilisieren; dann kleine überprüfbare Konsolidierungs- und Produktänderungen bearbeiten. Bestehende gültige Implementierungen erhalten. Kein vorausgesetzter Komplettumbau, kein stilles Löschen ungeklärter Artefakte und kein Zwang, das ganze Repository vor dem ersten Nutzen vollständig zu modellieren. Die konkrete Reihenfolge folgt Produktzielen, Risiken und Abhängigkeiten.

#### Verfassung gemeinsam entwerfen und delegiert weiterentwickeln

Agenten beziehungsweise Ressorts können Bestandsbefunde zusammentragen, Konflikte erkennen und Entwürfe für Regeln und Zielbild vorlegen. Anfangs fehlen häufig Informationen über Produktabsicht, erwünschtes Verhalten und Prioritäten; diese sind gezielt einzuholen und dürfen nicht aus Code oder grünen Tests erfunden werden.

Automatische Weiterentwicklung ist als separate Designfrage denkbar. Eine vorher festgelegte Delegation könnte gewöhnliche Modellpflege und bestimmte fachliche Entscheidungen ohne einzelne menschliche Freigabe erlauben. Welche Änderungen darunter fallen, muss inhaltlich beschrieben werden; auch eine vermeintliche Präzisierung kann eine neue Verpflichtung schaffen. Jede wirksame Änderung braucht eine bereits akzeptierte Grundlage oder passende delegierte Befugnis, eine getrennte Prüfung sowie nachvollziehbare Folgen. Außerhalb dieses Rahmens bleibt sie ein Vorschlag für den zuständigen Entscheider.

Agenten dürfen ihre eigenen Befugnisse nicht durch ein Modell-Update erweitern. Eine fehlgeschlagene Umsetzung darf nicht durch nachträgliches Abschwächen ihrer Anforderungen als erfolgreich erscheinen. Eine echte Änderung des Solls braucht einen eigenen begründeten Änderungsweg und erneute Planung/Prüfung; eine bekannte historische Abweichung wird dadurch nicht rückwirkend konform. Das Verfahren für Verfassungsänderungen und das für gewöhnliche Umsetzung sind getrennt zu gestalten. Einstimmigkeit der ausgewählten Ressorts ersetzt die nötige Änderungsbefugnis nicht.

Mandate benennen fachliche Belange; sie bestimmen nicht allein die politischen Produktprioritäten. Welche Ressorts sinnvoll sind, folgt aus Ziel und Bestand. Ein DDD-Ministerium ist kein vorausgesetzter Bedarf jedes unbekannten Repositories. Die bisher vorgeschlagene Zustimmung jedes ausgewählten Ressorts bleibt eine zu evaluierende Annahmepolitik.

#### Rückkopplung: Modell-Evolution aus Umsetzungserfahrung

Der Nutzer hält die Projektionsarchitektur ausdrücklich als mögliche Grundlage offen: ein belastbarer Weg vom kanonischen Modell zum Repository und ein Weg von beobachteten Artefakten zu einem Modellvorschlag können zusammenpassen. Zusätzlich braucht die kontinuierliche Entwicklungsmaschine einen Weg, auf dem Ausführungsprobleme, Modelllücken, widersprüchliche Anforderungen und bessere technische Lösungen zur Weiterentwicklung des Modells führen. Die Modellpflege darf nicht ausschließlich von neuen Nutzerprompts abhängen. Dieser Rückkopplungsprozess ist eine offene Designanforderung, keine bereits implementierte Fähigkeit.

Eine mögliche Aufteilung unterscheidet dauerhafte Produktziele, verbindliche Grenzen und Änderungsbefugnisse von weiterentwickelbaren konkreten Architekturentscheidungen und der laufenden Bestandsaufnahme. Diese Unterscheidung setzt keine neuen Core-Kinds voraus. Nicht jedes technische Detail braucht Verfassungsrang oder eine individuelle menschliche Freigabe. Auch echte Architekturentscheidungen können innerhalb zielorientierter Delegation autonom getroffen werden; der delegierte Bereich darf nicht auf redaktionelle Pflege beschränkt bleiben, wenn das Autonomieziel erreicht werden soll.

Der vorgeschlagene Rückweg ist: konkreter Befund → Klassifikation → begründeter Modelländerungsvorschlag mit Folgen und Alternativen → unabhängige Bewertung anhand übergeordneter Ziele und bestehender Änderungsbefugnis → delegierter Beschluss oder konkrete Besitzerentscheidung → neue Modellversion → erneute Planung, Umsetzung und Prüfung betroffener Darstellungen. Eine Idee oder Beobachtung allein wird dadurch noch nicht kanonisch; ein universelles Ideenregister folgt daraus nicht.

| Befund | Vorgesehene Reaktion |
|---|---|
| Die Umsetzung verletzt klare erfüllbare Regeln. | Umsetzung reparieren. |
| Eine bessere Lösung liegt innerhalb vorhandener Implementierungsfreiheit. | Gewöhnliche Umsetzung verbessern; keine künstliche Modelländerung. |
| Konkrete Modelllücke, widersprüchliche technische Vorgaben oder begründete Architekturverbesserung. | Getrennten Modellvorschlag bewerten; innerhalb bestehender ausreichender Delegation auch automatisch entscheiden. |
| Entscheidung verschiebt nicht delegierte Ziele, Grenzen, Befugnisse oder ungelöste Produktprioritäten. | Gezielte Entscheidung mit Empfehlung und Folgen vorlegen; unabhängige Arbeit fortsetzen. |

Beispielsweise kann die eigenständige Konsolidierung einer mehrfach implementierten Geschäftsregel samt Wahl eines fachlichen Eigentümers eine echte autonome Architekturentscheidung sein, wenn die geltenden Ziele, Abhängigkeitsregeln und der delegierte Auftrag sie tragen. Ob unabhängige Auslieferbarkeit zweier Produktbereiche zugunsten gemeinsamer Komponenten aufgegeben werden soll, kann dagegen eine offene Produktentscheidung sein. Eine fehlende Befugnis lässt sich nicht allein durch technische Erfolgsmessung ersetzen.

Modelländerungen werden nach dem bestehenden Änderungsverfahren beurteilt; Kandidatenimplementierungen nach dem danach angenommenen Modell. Der ausführende Agent ist nicht alleiniger Entscheider über eine Änderung seines eigenen Maßstabs. Beschlüsse binden Ausgangsmodell, Modelländerung und Begründung; sie legitimieren keine historischen Fehlversuche nachträglich. Betroffene laufende Pläne und Nachweise müssen auf Aktualität geprüft und nötigenfalls neu erstellt werden. Bewusste Übergangsarbeit bleibt offen sichtbar. Hypothesen können in begrenzten Kandidaten geprüft werden, ohne vor ihrer Annahme die geltende Ordnung zu überschreiben.

Government könnte dafür Gesetzgebung, Ausführung und unabhängige Prüfung organisieren, während die Projektionsarchitektur die Umsetzung des angenommenen Modells trägt. Diese Kombination ist eine konkrete zu evaluierende Hypothese und keine vorweggenommene Produktentscheidung. Im untersuchten Quellentwurf existieren Intent-Änderungen und begrenzte Reverse-Inferenz, aber kein durchgehender autonomer Modell-Evolutionsbetrieb. Die heutige Vision legt echte Absichts- und Architekturänderungen beim menschlichen Eigentümer ab; weitergehende Delegation wäre bewusst zu entscheiden, nicht als bereits vorhandener Vertrag darzustellen.

#### Liefergrenze

Der untersuchte Architect-Kandidat enthält Bausteine für explizite Brownfield-Inferenz, eng begrenzte Adoption, Reconciliation, Verifikation und Audit. Die Inferenz verarbeitet ausdrücklich ausgewählte Eingaben; sie entdeckt nicht selbstständig das unbekannte Repository und beweist keine Bestandsvollständigkeit. Ein verlässlicher Ablauf für beliebige unbekannte Repositories, ausreichende stackbezogene Analyse, Backlog-Konsolidierung und Priorisierung sowie ein dauerhafter autonomer Arbeitsbetrieb mit delegierter Verfassungsentwicklung sind damit nicht nachgewiesen. Diese Lücken sind beim späteren Checkpoint anhand des finalen Kandidaten getrennt von einem funktionierenden ersten Grundmodell zu bewerten. Ein grundlegender stabiler Ablauf kann einen ersten begrenzten Bereich tragen, ohne bereits das gesamte Anwendungsszenario abzudecken.

### Vorläufige Einordnung

Government kann zu Markitect passen, wenn Projektordnung und menschlich akzeptierte Absicht die Autorität behalten, Ressorts daraus ihre Mandate ableiten und das gesamte Verfahren auch fehlende Darstellungen, Änderungsfolgen und Drift behandelt. Dann dient die Organisation dem bestehenden Sollmodell.

Wenn das Produkt hauptsächlich beliebige Codeänderungen durch auswählbare DDD-, Architektur- und Sicherheitsagenten abstimmen lässt, entsteht dagegen primär eine Review- und Freigabeplattform. Sie kann nützlich sein, erfüllt aber Markitects Sollmodell- und Abgleichversprechen nicht schon durch das Kabinett. Ob eine solche Plattform eigenständig sein oder Markitect nutzen sollte, wäre eine separate Produktentscheidung.

Auch eine vollständig ausgeschmückte Regierung ist nicht automatisch die bessere Bedienung. Institutionen, Stimmen und Verfahren müssen konkrete Verantwortung verständlicher machen oder wiederkehrende Arbeit entfernen. Die Metapher darf weder neue semantische Autorität erzeugen noch zusätzliche Bürokratie zum Selbstzweck machen. Gegenwärtig ist kein vollständiger Government-Umbau begründet.

### Entscheidungskriterien für den Checkpoint

Zuerst einen konkreten Nutzerablauf benennen, den Markitect verbessern soll. Danach gegenüber dem vorhandenen Executor-/Verifier-Verfahren prüfen:

1. Welche übersehene Pflicht, widersprüchliche Darstellung oder fehlende Folge wird mit Ressorts besser behandelt?
2. Bleiben kanonische Absicht, vollständige betroffene Arbeit, Drift-Reparatur und gültige Implementierungsfreiheit erhalten?
3. Welche menschliche Koordination entfällt, und welche neue Pflege, Wartezeit und Konfliktklärung entsteht?
4. Liegt der Nutzen an fachlicher Spezialisierung, an Einstimmigkeit oder lediglich an einer verständlicheren Sprache?
5. Rechtfertigt der belegte Nutzen Government als Markitect selbst, eine darauf aufbauende Anwendung, ein unabhängiges Projekt oder das Verwerfen der Idee?

Erst aus dieser Prüfung folgt ein Architekturauftrag. Der nachfolgende Regierungsentwurf beschreibt ausschließlich, was bei Auswahl dieser Richtung zu gestalten wäre; er begründet die Auswahl nicht.

## Das Umdenken bei Auswahl der Government-Richtung

Heute organisiert Markitect Arbeit wesentlich entlang gewünschter Darstellungen: Eine Projection benennt Bedeutung, Scope und Ziel; ein gebundenes Modul liefert Zielwissen. Verifikation folgt dieser Darstellung und ihren Eltern-/Kindbeziehungen.

Die Government organisiert Verantwortung entlang fachlicher Belange. Ein DDD-Ministerium kann Code, Tests, Dokumentation und Konfiguration gemeinsam beurteilen. Mehrere Ministerien prüfen dieselben Artefakte, ohne dadurch mehrere Schreib-Eigentümer dieser Dateien zu werden.

Damit entstehen zwei unterschiedliche Beziehungen:

- **Umsetzungsverantwortung:** Wer darf welche Artefakte verändern, und wodurch wurden sie abgeleitet?
- **Prüfzuständigkeit:** Welche Ministerien müssen welche Belange vertreten und ausdrücklich zustimmen?

Beide Beziehungen müssen zusammenpassen. Sie dürfen nicht zu einer einzigen Datei- oder Projektionszuordnung zusammengezogen werden. Impact kann Umsetzung und Prüftiefe vorbereiten, aber kein ausgewähltes Ministerium von der Abstimmung ausschließen.

Die einheitliche Produktvorstellung wäre: Der Nutzer gestaltet die Projektordnung, stellt die Regierung zusammen und erteilt Änderungsaufträge. Markitect organisiert Ausführung, Ressortprüfung, Einwände, Überarbeitung und Annahme. Compiler, Projektionswerkzeuge und technische Adapter sind die darunterliegenden Werkzeuge dieser Regierung.

## Was bei dieser Richtung bleiben kann und geändert werden müsste

| Baustein | Bewertung | Government-Entsprechung und notwendige Änderung |
|---|---|---|
| Kanonisches Projektmodell | Behalten | Die verbindliche Projektordnung umfasst Konzepte, Beziehungen, Ziele und Regeln; sie wird nicht auf Verbote reduziert. |
| Minimaler struktureller Core | Behalten | Prüft ausdrückliche Sprache und Referenzen. Regierungsbegriffe können zunächst in einem gelieferten Schema ausgedrückt werden; universelle Ministeriumslogik im Compiler ist nicht begründet. |
| Kontext, Impact und Herkunft | Behalten und erweitern | Erschließen geltendes Recht und betroffene Arbeit. Hinzu kommt die Frage, welches Ressort welchen Belang tatsächlich geprüft hat. |
| Exakte Snapshots, Digests und feste Eingaben | Behalten | Identifizieren Gesetzesstand, Kabinett, Kandidat und geprüfte Belege. Ein Hash ist weiterhin keine Authentifizierung oder menschliche Annahme. |
| Ein Schreib-Eigentümer je Artefakt | Behalten | Mehrere Ressorts können ein Dokument prüfen; das macht sie nicht zu konkurrierenden Schreibern. |
| Executor, getrennte Verifikation und Reparatur | Behalten und anpassen | Ausführende Arbeit und Ressorturteil werden getrennt besetzt. Ein Ministerium kann Werkzeuge und Spezialisten verwenden, ist aber nicht alleiniger Richter eigener Umsetzung. |
| Geschützte Materialisierung und Verweigerung veralteter Pläne | Behalten | Kandidaten dürfen nicht unter geänderten Eingaben unbemerkt übernommen werden. Government-Annahme kommt als eigener Vertrag hinzu. |
| Operational Ledger und unveränderte Fehlversuche | Behalten und erweitern | Speichern zusätzlich Abstimmungsrunden, Ministeriumsurteile und Einwandsauflösung. |
| Lokale und übergeordnete Assurance | Behalten als Nachweiswerkzeug | Integration bleibt eigenständig zu prüfen. Ein grüner Elternknoten ersetzt keine Ressortzustimmung. |
| Brownfield-Adoption | Behalten | Bestehende gültige Dateien erhalten; unbekannte oder ausgeschlossene Bereiche sichtbar machen. |
| Projection als primäre Nutzerorganisation | Neu ausrichten | Ministerien und Änderungsverfahren bestimmen die Produktbedienung. Zielrepräsentationen bleiben technische Arbeitsgegenstände. |
| Installierbares Modul als Schema oder Projection | Produktvertrag erweitern | Ein Ministerium bündelt Mandat, Verfahren, Regelbezüge, Sprache und Werkzeuganforderungen. Es passt nicht unmittelbar in einen der heutigen beiden Pakettypen. |
| Ein Verifier, Projection als Prüfidentität | Neu gestalten | Mehrere getrennt gebundene Ressorts dürfen dasselbe Änderungspaket aus verschiedenen Blickwinkeln prüfen. |
| Abschlussaudit für Projektionen | Erweitern | Zusätzlich vollständiges Kabinett, aktuelle ausdrückliche Zustimmungen und erledigte Einwände prüfen. |
| Kontinuierlicher Runner | Später ergänzen | Bestehende Zustands- und Wiederaufnahmegrundlagen nutzen; kein Voraussetzungsausbau vor dem Government-Checkpoint. |

Eine zentrale konkrete Grenze: `CanonicalAssuranceScope` enthält eine `ProjectionID`; die aktuelle Konfigurationsvalidierung verlangt eindeutige Projektionen je Scope. Drei unabhängige Ressorts für denselben Änderungskandidaten sind deshalb keine reine Konfigurationsvariation dieses Vertrags. Auch `VerificationResult` ist an einen Materialisierungsrecord gebunden und stellt kein Kabinettsvotum dar.

## Ein kohärentes erstes Regierungsmodell

Die folgenden Begriffe sind ein Designvorschlag, keine bereits unterstützten Ressourcentypen oder Befehle.

| Begriff | Verantwortung |
|---|---|
| Projektordnung | Verbindliches fachliches und technisches Weltbild einschließlich Gesetzen und Zielen |
| Verfassung | Grundverfahren: Zuständigkeiten, Einstimmigkeit, menschliche Modellhoheit und Umgang mit unlösbaren Konflikten |
| Regierung / Kabinett | Vollständige ausgewählte Menge der Ministerien mit festen Paketversionen und projektbezogenen Mandaten |
| Ministerium | Dauerhafte Verantwortung für einen Belang; kein dauerhaft laufender Agentenprozess erforderlich |
| Änderungsauftrag | Gewünschtes Ergebnis, Einordnung als Intentänderung, Umsetzung oder Reparatur und benannte Ausgangsbasis |
| Änderungsakte | Gemeinsame gebundene Grundlage mit Kandidat, Projektordnung, Kabinett, tatsächlicher Änderung und Nachweisen |
| Ausführende Verwaltung | Koordiniert und bearbeitet Aufgaben mit den passenden technischen Werkzeugen und Spezialisten |
| Ressortstellungnahme | Ausdrückliche Zustimmung oder konkrete Einwände beziehungsweise fehlende Beurteilbarkeit |
| Regierungsbeschluss | Annahme des endgültigen Kandidaten bei vollständiger aktueller Einstimmigkeit und erfüllten technischen Voraussetzungen |

Die Verfassung darf kein neuer Verwaltungsaufwand werden, in dem jede erlaubte Implementierungsentscheidung einzeln freigegeben werden muss. Das kanonische Modell enthält, was gilt; gewöhnliche Detailentscheidungen bleiben bei den Agenten. Explorative Gespräche brauchen weiterhin kein verpflichtendes Ideen- oder Entscheidungsstatusregister. Eine betriebliche Änderungsakte verfolgt tatsächliche Arbeit, nicht jeden Gedanken.

## Ministerien als angebotene Pakete

Ein Ministeriumspaket sollte mindestens enthalten:

- eindeutige Identität, feste Version und Inhaltsdigest;
- verständliches Ressort und projektbezogen konkretisierbares Mandat;
- Regelbezüge und transparente angebotene Standardregeln;
- Anforderungen an Kontext, Nachweise und technische Werkzeuge;
- Prüfverfahren einschließlich Betroffenheitsprüfung und gemeinsamer Änderungen;
- Vertrag für Stellungnahmen, Einwände und unzureichende Evidenz;
- Anforderungen an getrennte Ausführung und Prüfung sowie negative Prüffälle.

Das Paket bringt Fachwissen mit. Verbindliche Projektregeln haben weiterhin einen kanonischen Eigentümer. Bei der Auswahl können angebotene Standardregeln bewusst übernommen werden; danach darf keine konkurrierende Regelkopie im Ministeriumsprompt entstehen. Ein Paketupdate ist keine stillschweigende Gesetzesänderung.

Als erste technische Option kann ein Government-Paket auf exakt gepinnte bestehende Schema- und Zielwerkzeugpakete verweisen. Die heutige strikte Trennung dieser unteren Pakete muss dafür nicht beiläufig aufgeweicht werden. Ob ein eigenständiges Ministeriumsmanifest oder ein umfassender neuer Paketvertrag sinnvoller ist, gehört zum Checkpoint. Installation unbekannter Programme oder ein allgemeiner dynamischer Pluginloader folgt aus der Produktidee nicht automatisch.

DDD, Clean Architecture und Sicherheit wären sinnvolle erste Ressorts. Ein DDD-Ministerium darf nicht überall ein reiches Domänenmodell erzwingen: Es beurteilt die geltende Projektordnung und die dort tatsächlich erforderlichen Invarianten. Ressortwissen ist keine Befugnis, eigene Vorlieben zu Projektgesetzen zu machen.

## Endlicher Ablauf von Auftrag bis Annahme

1. **Auftrag und Ordnung klären.** Vorhandene Projektordnung und Kabinett binden. Eine tatsächliche Intentänderung wird am kanonischen Eigentümer beschlossen; ein Bugfix erfordert keine erfundene Modelländerung.
2. **Arbeit vorbereiten.** Kontext, Auswirkungen, Zielzustände und Eigentümerschaft ableiten. Jedes Ministerium kann schon zum Plan Anforderungen oder erkennbare Konflikte benennen. Das ist noch keine Zustimmung zu späteren Dateien.
3. **Kandidat erstellen.** Ausführende Agenten erzeugen eine zusammenhängende Änderung in einem gesonderten Kandidaten-Arbeitsbereich. Technische Checks und Integrationsprüfungen laufen dort. Die aktiven verwalteten Projektdateien werden vor der einstimmigen Annahme nicht durch diese Kandidatenarbeit verändert; ein separater Arbeitsbereich ist dabei noch keine Betriebssystem-Sandbox.
4. **Akte einfrieren.** Alle Ressorts erhalten dieselbe Identität des vollständigen Kandidaten und der geltenden Ordnung. Ressorts erhalten die relevante Evidenz und können weitere begründete Einsicht anfordern; ein reiner Dateidiff genügt nicht immer.
5. **Alle Ministerien beteiligen.** Das vollständige Kabinett wird befragt. Jedes Ressort prüft seine Betroffenheit und sein Mandat. Zulässige Zustimmung ist auch: "Mein Ressort ist nicht berührt; ich stimme dieser Fassung zu." Das ist ein ausdrückliches Urteil, keine Enthaltung.
6. **Einwände bearbeiten.** Einwände nennen Regel, betroffene Stelle, Befund und eine überprüfbare Bedingung für die Erledigung. Technische Schwierigkeiten gehen zurück zur ausführenden Verwaltung. Echte widersprüchliche Gesetze oder fehlende wesentliche Entscheidungen gehen an die nächsthöhere ausreichend befugte Instanz; nur außerhalb der delegierten Befugnisse an den Nutzer.
7. **Neue Fassung erneut abstimmen.** Jede Änderung am Kandidaten eröffnet eine neue Abstimmungsrunde für das gesamte Kabinett. Bisherige Stimmen bleiben historisch erhalten und gelten nicht für die neue Fassung. Für den ersten Entwurf werden keine stillschweigenden Zustimmungsübertragungen optimiert.
8. **Beschluss und geschützte Annahme.** Nur vollständige aktuelle Einstimmigkeit kann zur Annahme führen. Die Übernahme prüft erneut Kandidat, Ordnung, Kabinett, Werkzeuge, Nachweise und tatsächliche Zielbytes. Geänderte Eingaben lassen das Verfahren offen.
9. **Abschluss und spätere Beobachtung.** Der Abschluss nennt Stimmen, erledigte Einwände, Grenzen und technischen Stand. Spätere Abweichung startet einen Reparaturauftrag unter derselben Ordnung. Dauerhafte Ereignisbeobachtung und Wiederaufnahme sind ein anschließender Runtime-Schritt.

Die heutige Aktion `controller-apply` schreibt Artefaktbytes an konfigurierte Zielpfade, erzeugt einen technischen Evidence-Commit und bewegt HEAD/Index nicht. Das erzeugt **für sich genommen weder Isolation noch Government-Annahme**. Im vorgeschlagenen Verfahren darf eine Materialisierung vor Kabinettszustimmung nur in einen ausdrücklich vorbereiteten separaten Kandidaten-Arbeitsbereich erfolgen. Dasselbe Kommando unverändert auf aktive Projektziele anzuwenden, erfüllt diese Grenze nicht. Die anschließende Übernahme in den aktiven Projektstand benötigt einen eigenen, nach der Abstimmung geschützten Annahmevertrag. Ob die heutigen Snapshot-/Writer-Fähigkeiten diesen getrennten Arbeitsbereich vollständig tragen, ist vor Implementierung zu prüfen. Ein vorhandenes Apply darf nicht unbemerkt zum politischen Beschluss umgedeutet werden.

## Zustimmungs- und Nachweisvertrag

Ein Ressorturteil muss mindestens binden: Änderungsakte und Runde, endgültigen Kandidaten, Ausgangsbasis, akzeptierte Projektordnung, Kabinettszusammensetzung, Ministeriumspaket und Mandat, Prüfer-/Werkzeugkonfiguration, verwendete Evidenz, geprüfte Belange und seine ausdrückliche Entscheidung. Einwände und Unsicherheit bleiben Bestandteil desselben Urteils. Anwendungsfehler und ausbleibende Antwort sind betriebliche Zustände, keine Stimmen.

Die Annahmebedingung lautet für ein bewusst ausgewähltes, nicht leeres Kabinett:

> Technische Voraussetzungen erfüllt UND alle gewählten Ministerien haben aktuell genau diesem Kandidaten unter genau dieser Ordnung und diesem Kabinett ausdrücklich zugestimmt UND kein verbindlicher Einwand ist offen UND die tatsächliche Übernahme entspricht dem geprüften Kandidaten.

"Prüfung bestanden", "keine Arbeit", "nicht gefragt", "Zeitlimit erreicht", "unbekannt" und eine Zustimmung zu einer älteren Fassung erfüllen diese Bedingung nicht. Unterschiedliche Modelle allein beweisen keine unabhängige Prüfung. Herkunftsdigests allein authentifizieren keine Person oder Institution.

Die Akte muss sowohl **Artefaktvollständigkeit** als auch **Beteiligungsvollständigkeit** prüfen. Vollzählige Ministerien beweisen nicht, dass das Kabinett ausreichend gewählt wurde oder dass jedes Ressort alle relevanten Folgen erkannt hat. Die Abdeckung bleibt nachvollziehbar begrenzt; strukturelle Vollzähligkeit darf keine universelle Korrektheitsbehauptung werden.

## Praktische Nutzung als Zielbild

Für ein vorhandenes .NET-Projekt würde der Nutzer zunächst sein bestehendes Weltbild bestätigen und das Kabinett aus DDD, Clean Architecture und Sicherheit wählen. Die Übernahme bestehenden Codes inventarisiert, prüft und erhält gültige Dateien. Unmodellierte Bereiche werden sichtbar geführt.

Danach lautet ein Auftrag etwa: "Verlagere diese Fähigkeit aus dem Produkt in das Modul; das Produkt soll sie nur konfigurieren." Die Verwaltung leitet notwendige Arbeit ab. Die Ressorts prüfen dieselbe endgültige Änderung aus ihren jeweiligen Blickwinkeln. Der Nutzer sieht eine knappe Akte mit offenen Einwänden, erforderlichen Entscheidungen und dem abschließenden Beschluss. Er muss weder Projektionsdetails noch einzelne Prüferprompts koordinieren.

Dasselbe Grundverfahren gilt für eine gewöhnliche Reparatur. Der Nutzer muss keine neue Regierung entwerfen und keine Regelkopien in Agentenanweisungen nachziehen. Benutzeroberfläche und öffentliche Begriffe sollen dieses eine Modell durchziehen. Zusätzliche Institutionen wie Kammern oder Gerichte werden erst eingeführt, wenn eine tatsächlich benötigte Verantwortung sie rechtfertigt.

## Offene Entscheidungen am Checkpoint

| Frage | Entscheidungsvorschlag beziehungsweise offene Grenze |
|---|---|
| Ist Government die Produktidentität? | Offen. Zuerst Zweck und Nutzen prüfen; Markitect selbst, darauf aufbauende Anwendung, unabhängiges Projekt und Verwerfen bleiben mögliche Ergebnisse. |
| Was bedeutet Zustimmung fachlich? | Ressortgebundenes Urteil gegen geltende Regeln, keine freie Geschmacks- oder Mehrheitsentscheidung. |
| Darf ein Ressort zugleich ausführen? | Getrennte konkrete Prüfinstanz und unabhängige Nachweise erforderlich; eigenes Ergebnis nicht allein selbst freigeben. |
| Wer löst Ressortkonflikte? | Verwaltung sucht eine Lösung innerhalb der Ordnung. Unvereinbare Anforderungen gehen an die ausreichend befugte Instanz, nötigenfalls bis zum Nutzer. Das Verhältnis eines Gerichtsurteils zum absoluten Ressortveto bleibt offen. |
| Wie werden Kabinett und Verfassung geändert? | Kein Entfernen eines widersprechenden Ressorts während einer Runde. Das Verfahren für Änderungen an der Abstimmungsordnung selbst muss ausdrücklich festgelegt werden: bisheriges Kabinett, Nutzerhoheit, möglicher Übergang. Noch kein implizites Override. |
| Können frühere Zustimmungen übernommen werden? | Für den ersten Entwurf nein: neue Kandidatenfassung, neue Stimmen aller Ministerien. Spätere Optimierung muss die Einstimmigkeitsregel erhalten. |
| Wann wird aus Materialisierung Annahme? | Heutiges Evidence-Apply und endgültige Government-Annahme ausdrücklich trennen und den Übergang gegen geänderte Eingaben schützen. |
| Wie erkennen wir fehlende Ressorts? | Auswahlhilfe und sichtbare Abdeckung untersuchen; Einstimmigkeit im gewählten Kabinett deckt ausgelassene Belange nicht automatisch ab. |
| Welche Anpassungen sind kompatibel? | Alte Releases bleiben unverändert. Neue Ressourcen, Paketverträge, Zustimmungen und Bedienung brauchen eine bewusste Versionierungs-/Migrationsentscheidung. |

## Vorschlag für das weitere Vorgehen nach dem Grundmodell

Der Nutzer hält eine Anpassung von Markitect samt konsistenter Umbenennung für denkbar und vermutet, dass die wesentlichen Grundmechanismen bereits passen. Betriebsmodell, Begriffskarte und Validierungsplan werden auf seine neueste Anweisung jetzt parallel vorbereitet. Die Implementierungsetappen bleiben Vorschläge zur Entscheidung. Die gegenwärtige Arbeit des Architect wird zum benannten Abschluss geführt; ihr tatsächlicher finaler Kandidat wird anschließend als Umsetzungsausgangsbasis gesichert und gegen die Vorbereitung abgeglichen.

1. **Betriebsmodell als Entscheidungsentwurf konkretisieren.** Den bereits beschriebenen Zweck festhalten: kontinuierlich gut verifizierte Entwicklungsarbeit bei wenig notwendiger menschlicher Koordination. Hierarchie und fachübergreifende Prüfung unterscheiden. Jede Ebene besitzt ein Mandat, begrenzten relevanten Kontext und pro Durchlauf einen eigenen Agentenauftrag. Ziele, Befugnisse und Mittel werden nach unten übertragen; Berichte und Vorschläge gehen nach oben. Ausführungs- und unabhängige Prüfaufträge bleiben getrennt. Routine-Implementierungsfreiheit braucht keine vollständige Erlaubnisliste.
2. **Entscheidung, Bericht und Sichtbarkeit festlegen.** Jeder Durchlauf berichtet; übergeordnete Ebenen prüfen auch ihre eigene Zusammensetzungspflicht anhand aktueller Belege. Ein Bericht ist noch keine bindende Ressortzustimmung. Festlegen, welcher konkrete Akteur diese abgibt und wie fehlende oder widersprüchliche Berichte behandelt werden. Anzeigeoptionen alles/wichtiges/nichts gelten je Ebene oder insgesamt; sie ändern keine Befugnis und erzeugen keine automatische Zustimmung. Modelländerungen innerhalb ausreichender Delegation dürfen autonom beschlossen werden. Das Verhältnis von Gerichtsurteil und absolutem Ressortveto sowie das Verfahren für Regel- und Mandatsänderungen bleiben vor Implementierung zu entscheiden; die bisherige ausdrückliche Einstimmigkeitsanforderung wird nicht stillschweigend entfernt.
3. **Quell- und Begriffsabbildung am finalen Kandidaten erstellen.** Pro relevantem Vertrag ausdrücklich behalten, erweitern, aufteilen oder ersetzen entscheiden. Gewünschte Darstellung, dauerhafte Verantwortung, konkrete Arbeit und technische Fähigkeit bleiben unterscheidbar. Eine Projection ist heute ein dauerhafter Sollvertrag und weder ein einmaliger Auftrag noch eine handelnde Instanz. Öffentliche und interne Begriffe können konsistent angepasst werden, sobald ihre Bedeutung beschlossen ist; ein bloßer Austausch von Klassennamen liefert die neue Befugnis- und Betriebslogik nicht.
4. **Einen kleinsten durchgängigen Ablauf liefern und untersuchen.** Nach separater Beauftragung nur die dazu fehlenden Betriebsverträge auf vorhandenen Mechanismen ergänzen. Ein klar begrenzter Bereich eines realen Repositories oder eine vorhandene belastbare Ausgangsbasis genügt; unbekannte Artefakte bleiben sichtbar. Zuerst eine normale Aufgabe mit Delegation, Ausführung, unabhängiger Prüfung, Zusammensetzung und Berichten durchlaufen. Danach eine klar delegierte Modelländerung samt erneuter Umsetzung und eine konkrete Entscheidung außerhalb des Mandats untersuchen. Unterbrechung/Wiederaufnahme und tatsächliche Parallelität sind eigene nächste Fälle, sobald der endliche Ablauf trägt; kein sofortiges allgemeines Organisations- oder Pluginframework.
5. **Auswerten und Migration entscheiden.** Funktion und Architektur, übersehene Pflichten, falsche Einwände, Integration, menschliche Eingriffe und Modell-/Betriebspflege gegen vorher benannte Erwartungen und einen gut ausgestatteten klassischen Ablauf beurteilen. Erst daraus breitere Umstellung und konsistente Produktterminologie ableiten. Bei begrifflichen oder strukturellen Änderungen Beispiele, Konfiguration, Schemaquellen, generierte Ausgaben und dokumentierte öffentliche Verträge gemeinsam migrieren. Bestehende unveränderliche Releases bleiben historische Verträge; Veröffentlichung ist ein separater Schritt.

Arbeitsbegriffe für die Abbildung, noch nicht festgelegt: **Projektordnung** für das gesamte akzeptierte kanonische Modell; **Verfassung** für dessen tragende Ziele und Befugnisse; **Mandat** für dauerhafte Verantwortung; **Arbeitsauftrag** für konkrete Arbeit; **Zielvorgabe** für die gewünschte Darstellung; **Ausführungsfähigkeit** für Zielwissen und Werkzeuge; **Prüfauftrag** für unabhängige Beurteilung; **Bericht/Akte** für Ergebnisse, Nachweise und offene Entscheidungen. Ein Ressort ist eine fachliche Organisation und kann mehrere solcher Fähigkeiten und Aufträge verwenden. Kein einzelner bisheriger Pakettyp wird automatisch zu einem vollständigen Ministerium.

## Begrenzte Evaluation nach Architect-Abschluss

Zuerst den dokumentierten finalen Architect-Kandidaten sichern und gegen diese Bestandsaufnahme abgleichen. Dann Produktzweck und Government-Fit gemeinsam entscheiden. Die folgenden Fälle stammen aus dem ursprünglichen Kabinettsentwurf und sind mögliche vertiefende Prüfungen, keine Verpflichtung, vor dem ersten Nutzen alle Fälle oder eine vollständige Regierung zu implementieren. Sie ergänzen den oben beschriebenen schrittweisen Betriebsversuch. Nur bei positivem Ergebnis kann ein separat beauftragter begrenzter Prototyp folgen; eine Möglichkeit wären drei Ministerien und dieselben Aufgaben wie der vorhandene Executor-/Verifier-Ablauf:

1. **Gewöhnliche lokale Änderung:** alle Ressorts antworten, auch nicht betroffene; gültige Nachbarartefakte bleiben erhalten.
2. **Übergreifende Verletzung:** ein relevantes Architektur- oder Sicherheitsproblem wird aus einer anderen Ressortperspektive entdeckt.
3. **Einwand, Reparatur, neue Runde:** die Reparatur macht alle früheren Stimmen historisch; Zustimmung und finaler Kandidat stimmen exakt überein.
4. **Ausfall oder unklare Zuständigkeit:** fehlende Beurteilbarkeit erzeugt offene Arbeit und begrenzte Wiederaufnahme, keine Annahme durch Zeitablauf.
5. **Echter Regelkonflikt und Kabinettsänderung:** die notwendige Nutzerentscheidung wird präzise vorgelegt; kein stilles Umschreiben der Gesetze oder Entfernen des widersprechenden Ressorts.

Erfolg bedeutet bessere erkennbare Abdeckung und weniger notwendige menschliche Koordination bei erhaltenem Qualitätsmaßstab. Vollzählige Ja-Stimmen allein reichen nicht. Festzuhalten sind übersehene Verstöße, unberechtigte Einwände, Erledigung, Wiederholungsrunden, echte versus vermeidbare Nutzerentscheidungen und verbleibende Unsicherheit. Laufzeit und Kosten ergänzen diese Beobachtungen.

Das Ergebnis dieses Checkpoints ist eine bewusste Produkt- und Designentscheidung mit einem begrenzten nächsten Auftrag. Ein weiterer autonomer Ausbau, ein neues Release, ein allgemeines Pluginframework und Jev-Integration beginnen dadurch nicht automatisch.

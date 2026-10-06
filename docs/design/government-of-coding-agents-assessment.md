# Markitect-Produktzweck und Government: Prüfung vor Designentscheidung

Stand: 2026-10-07, Europe/Berlin. **Diskussions- und Evaluationsentwurf; keine implementierte Funktion und kein Auftrag zum sofortigen Umbau.**

## Verbindlicher Arbeitsauftrag und Reihenfolge

Der Nutzer hat folgenden nächsten Entscheidungspunkt festgelegt:

1. Der Architect schließt seinen derzeitigen Grundmodell-Schritt mit einem soliden, nachvollziehbar validierten Stand ab.
2. An diesem Stand halten wir mit weiterem Ausbau an. Zuerst bestimmen wir Markitects Problem, Produktzweck und Zielbild; daran prüfen wir die Government-Richtung und entscheiden gemeinsam über das weitere Vorgehen. Spätere ausdrückliche Nutzeraufträge im Architect-Chat werden dabei berücksichtigt; laufende autorisierte Arbeit wird nicht durch eine veraltete Wiedervorlage unterbrochen.
3. **Government als Produktidentität ist eine offene Hypothese, keine bevorzugt beschlossene Zukunft.** Der Nutzer hat die Reihenfolge ausdrücklich korrigiert: zuerst verstehen, wofür Markitect gebaut wird, dann prüfen, ob Government dazu passt oder ein anderes Produkt daraus macht. Falls die Metapher gewählt wird, soll sie durchgängig stimmig sein; diese Gestaltungspräferenz ersetzt keine Nutzenprüfung.
4. Der Nutzer stellt sein Kabinett aus angebotenen, wiederverwendbaren Ministerien zusammen. **Eine Änderung wird nur akzeptiert, wenn jedes ausgewählte Ministerium der endgültigen Fassung ausdrücklich zugestimmt hat.**
5. Jev/TypeSafe AI bleibt eine nachrangige Untersuchung. Zuerst erfolgt die Government-Entscheidung; Jev wird nicht in den laufenden Architect-Schritt aufgenommen.

Jetzt autorisiert sind Bestandsaufnahme und Designvorbereitung. Der komplette Entwurf unten ist noch keine angenommene neue Architektur. Insbesondere werden aktuelle Vision, Core-Verträge, veröffentlichte Versionen und der aktive Architect-Kandidat durch diese Notiz nicht umgeschrieben.

Ein grüner Einzeltest, ein inaktiver Chat oder ein erfolgreicher Teilversuch sind kein automatischer Nachweis eines soliden Gesamtstands. Für den Checkpoint brauchen wir einen benannten Kandidaten, den Abschluss des aktuellen begrenzten Ablaufs, zugehörige Quellprüfungen und eine klare Liste verbleibender Grenzen. Weitere Experimente werden nicht beliebig an diesen Abschluss angehängt.

## Vergleichsbasis

Untersucht wurde der separate Architect-Worktree `C:/Users/Consiliari/.codex/worktrees/standard-operating-model/Markitect`, Kandidat `2c5299e0a876ba491e6d9fc93b8965fe21d62917` (`codex/standard-operating-model`). Quellverträge wurden zusätzlich mit `git show` aus dieser Revision gelesen. Diese Notiz liegt auf einem eigenen Dokumentationsbranch des ursprünglichen Checkouts; dessen ältere `main` ist **nicht** die technische Vergleichsbasis.

Die Analyse ist keine erneute Testsuite oder Releasefreigabe. Der Architect führt seine abschließenden Kandidatenprüfungen selbst durch. Beim späteren Checkpoint wird dessen tatsächlicher finaler Stand erneut abgeglichen.

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
6. **Einwände bearbeiten.** Einwände nennen Regel, betroffene Stelle, Befund und eine überprüfbare Bedingung für die Erledigung. Technische Schwierigkeiten gehen zurück zur ausführenden Verwaltung. Echte widersprüchliche Gesetze oder fehlende wesentliche Entscheidungen gehen zum Nutzer.
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
| Wer löst Ressortkonflikte? | Verwaltung sucht eine Lösung innerhalb der Ordnung. Unvereinbare Anforderungen erfordern eine konkrete Nutzerentscheidung. |
| Wie werden Kabinett und Verfassung geändert? | Kein Entfernen eines widersprechenden Ressorts während einer Runde. Das Verfahren für Änderungen an der Abstimmungsordnung selbst muss ausdrücklich festgelegt werden: bisheriges Kabinett, Nutzerhoheit, möglicher Übergang. Noch kein implizites Override. |
| Können frühere Zustimmungen übernommen werden? | Für den ersten Entwurf nein: neue Kandidatenfassung, neue Stimmen aller Ministerien. Spätere Optimierung muss die Einstimmigkeitsregel erhalten. |
| Wann wird aus Materialisierung Annahme? | Heutiges Evidence-Apply und endgültige Government-Annahme ausdrücklich trennen und den Übergang gegen geänderte Eingaben schützen. |
| Wie erkennen wir fehlende Ressorts? | Auswahlhilfe und sichtbare Abdeckung untersuchen; Einstimmigkeit im gewählten Kabinett deckt ausgelassene Belange nicht automatisch ab. |
| Welche Anpassungen sind kompatibel? | Alte Releases bleiben unverändert. Neue Ressourcen, Paketverträge, Zustimmungen und Bedienung brauchen eine bewusste Versionierungs-/Migrationsentscheidung. |

## Begrenzte Evaluation nach Architect-Abschluss

Zuerst den dokumentierten finalen Architect-Kandidaten sichern und gegen diese Bestandsaufnahme abgleichen. Dann Produktzweck und Government-Fit gemeinsam entscheiden. Nur bei positivem Ergebnis kann ein separat beauftragter begrenzter Prototyp folgen; eine Möglichkeit wären drei Ministerien und dieselben Aufgaben wie der vorhandene Executor-/Verifier-Ablauf:

1. **Gewöhnliche lokale Änderung:** alle Ressorts antworten, auch nicht betroffene; gültige Nachbarartefakte bleiben erhalten.
2. **Übergreifende Verletzung:** ein relevantes Architektur- oder Sicherheitsproblem wird aus einer anderen Ressortperspektive entdeckt.
3. **Einwand, Reparatur, neue Runde:** die Reparatur macht alle früheren Stimmen historisch; Zustimmung und finaler Kandidat stimmen exakt überein.
4. **Ausfall oder unklare Zuständigkeit:** fehlende Beurteilbarkeit erzeugt offene Arbeit und begrenzte Wiederaufnahme, keine Annahme durch Zeitablauf.
5. **Echter Regelkonflikt und Kabinettsänderung:** die notwendige Nutzerentscheidung wird präzise vorgelegt; kein stilles Umschreiben der Gesetze oder Entfernen des widersprechenden Ressorts.

Erfolg bedeutet bessere erkennbare Abdeckung und weniger notwendige menschliche Koordination bei erhaltenem Qualitätsmaßstab. Vollzählige Ja-Stimmen allein reichen nicht. Festzuhalten sind übersehene Verstöße, unberechtigte Einwände, Erledigung, Wiederholungsrunden, echte versus vermeidbare Nutzerentscheidungen und verbleibende Unsicherheit. Laufzeit und Kosten ergänzen diese Beobachtungen.

Das Ergebnis dieses Checkpoints ist eine bewusste Produkt- und Designentscheidung mit einem begrenzten nächsten Auftrag. Ein weiterer autonomer Ausbau, ein neues Release, ein allgemeines Pluginframework und Jev-Integration beginnen dadurch nicht automatisch.

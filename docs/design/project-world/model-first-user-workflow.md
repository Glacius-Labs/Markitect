# Kanonisches Systemmodell und durchgängiger Arbeitsablauf

Stand: 9. Oktober 2026, Europe/Berlin. Direkte Produktvorgabe des Nutzers für `codex/model-driven-delivery`; Dokumentations- und Implementerhandoff, keine Behauptung bereits implementierter Funktionen. Quellabgleich: `c1f24b1c55dc3d5c63658b7af2b0fb9f7e4e5604`.

Das Ziel ist ein typisiertes, kanonisches Weltbild, das das gewünschte System, Projekt und Repository definiert. Code, Tests, Dokumentation und Infrastruktur sind verschiedene Realisierungen oder Projektionen der darin beschriebenen Konzepte und Regeln. Eine organisierte Agentenausführung soll diese Vorgaben über alle Repräsentationen hinweg besser einhalten, gezielter validieren und bei Änderungen erhalten als gewöhnliches Agentic Coding.

Diese Präzisierung besitzt den neuen Benutzerablauf, die vollständige Dateiabdeckung und die Brownfield-Rückführung. Bei abweichenden früheren Produktvorschlägen im [Planungspaket](README.md) gilt diese Richtung. Der [Umsetzungsvertrag](delivery-contract.md) und der [CLI-Arbeitsablauf](../../project-workflow.md) bleiben die Beschreibung der jeweils tatsächlich vorhandenen Technik; eine neue Anforderung ändert deren Implementierungsstatus nicht.

## Problem und gewünschter Nutzen

Das Ausgangsbeispiel ist Software für die Vermietung von Baustützen: Bestand, zeitbezogene Verfügbarkeit, Mietaufträge, Ausgabe, Rückgabe, Lieferung, Ersatzbeschaffung und Abrechnung werden zunächst ungeordnet im Gespräch beschrieben. Das Modell wird fortlaufend verfeinert, später kommen Architektur, Modularisierung und Entwicklungsprozesse hinzu. Mehr Mitarbeiter meint hier zusätzliche **Softwareentwickler**, die das Repository gemeinsam weiterentwickeln, nicht zusätzliche Beschäftigte des Vermietungsbetriebs.

Mit wachsender Teamgröße entstehen unterschiedliche Wissensstände und konkurrierende Aussagen in Gesprächen, Code, Kommentaren, Tests und Dokumentation. Globale Konsolidierung findet wiederholt Widersprüche und kostet Zeit, Modellaufrufe sowie menschliche Koordination. Teamzuschnitt und Softwaregrenzen sollen deshalb bewusst zusammenpassen; Verantwortungsbereiche sind sowohl fachliche Abstraktionsgrenzen als auch Träger der Änderungsbearbeitung. Daraus folgt keine Pflicht zu separaten Microservices oder einer Managerrolle pro Entwickler.

Die Produktthese lautet: Ein strukturell geprüftes gemeinsames Modell, explizite Realisierungsbezüge, begrenzte Kontexte, zuständige Manager, getrennte Reviews und rekursive Integration verbessern Kohärenz und Anforderungstreue. Die zusätzliche Organisation soll mit Agenten auf dem Niveau **Luna High** wirtschaftlich tragbar sein und den wiederholten Einsatz sehr teurer Spitzenmodelle zur globalen Konsolidierung reduzieren. Qualität, Kosten und menschlicher Aufwand sind gemeinsam zu messen; weder Kostenneutralität noch Überlegenheit gegenüber Astra Extra High, Fable max oder einem konventionellen Prozess werden vorausgesetzt. Die genannten Modellniveaus beschreiben den Nutzervergleich, keine fest kodierten Produktabhängigkeiten oder verifizierten Providerangebote.

## Modell und Repräsentationen

Das kanonische Modell enthält Fachbegriffe, Konzepte, Regeln, Verhalten, Architektur, Verantwortung und Entwicklungsarbeitsweise. Der Compiler prüft die formal ausdrückbaren Eigenschaften, insbesondere Typen, Identitäten, Referenzen und strukturelle Konsistenz. Eine fehlende deklarierte Begriffsreferenz muss vor der Umsetzung diagnostiziert werden. Unreferenzierte Wörter in beliebiger Prosa werden dadurch nicht automatisch zu maschinell erkannten Modellfehlern; der Modellierungsagent muss semantische Lücken erkennen und explizit machen.

Ein Konzept kann viele Realisierungen besitzen; eine Datei kann mehrere Konzepte oder Regeln realisieren. Jeder relevante Bezug muss vom Modell zur Datei und von der Datei zum Modell auflösbar sein. Dateiverantwortung allein genügt nicht als Realisierungsnachweis. Ein Regelwechsel soll dadurch Code, Tests, Dokumentation, Kommentare innerhalb betroffener Dateien und weitere deklarierte Verbraucher in den Änderungs- und Prüfauftrag bringen.

„Projektion“ ist für diesen fachlichen Zusammenhang wieder ein zulässiger Begriff. Das ersetzt die frühere Absicht, ihn öffentlich aufzugeben. Daraus folgt weder eine Rückkehr zu alten technischen Projection-Kinds noch die Forderung, alle Ergebnisse deterministisch zu rendern. Ein Build beweist strukturelle Eigenschaften von Code; entsprechend beweist Modellkompilation strukturelle Eigenschaften des Modells. Semantische Übereinstimmung der unterschiedlichen Repräsentationen benötigt zusätzlich geeignete Checks und unabhängige Beurteilung.

## Greenfield von Init über Explore zum ersten Apply

1. **Projekt initialisieren.** Die CLI erzeugt Standarddateien. Der Gesprächsagent fragt, welche Coding Agents die Beitragenden verwenden, beispielsweise Claude und Codex. Für die Auswahl werden projektlokale Einstiegspunkte, passende `.claude/`- und `.codex/`-Inhalte, `AGENTS.md` sowie benötigte Skills, Workflows, Agenten und Rules eingerichtet. Konkrete Providerformate müssen zu den unterstützten Versionen passen. Bestehende Dateien werden erhalten oder über eine sichtbare Zusammenführung aktualisiert; fremde Inhalte werden nicht blind überschrieben.
2. **Beitragende automatisch einführen.** Wer mit einem ausgewählten Coding Agent im Repository arbeitet, soll den Markitect-Arbeitsweg über dessen native Einstiegspunkte erhalten, ohne Markitect-Kommandos oder das Speicherformat kennen zu müssen. Alle Einstiege führen zum selben Modell und Ablauf. Projekthinweise allein verhindern keinen technischen Bypass durch einen Agenten mit gewöhnlichen Schreibrechten; Abschlussprüfungen und gegebenenfalls CI müssen die tatsächlichen Ergebnisse abgleichen.
3. **Explore beginnen.** Der Agent fragt nach Zweck und Ziel des Projekts. Der Nutzer erzählt frei; der Agent modelliert während des Gesprächs statt ausschließlich eine Sammlung unabhängiger Markdown-Notizen anzulegen. Unverbindliche Ideen, Annahmen, Entscheidungen und akzeptierte Vorgaben bleiben unterscheidbar. Der Compiler wird fortlaufend für den strukturierten Entwurf genutzt. Exploration darf zeitweilig unvollständig sein und führt noch keine Implementierung aus.
4. **Gezielt nachfragen.** Neue Begriffe, Beziehungen und Compilerdiagnosen führen zu konkreten Rückfragen oder Vorschlägen. Der Agent sammelt offene Punkte und unterbricht nicht jeden Gedanken mit einem vollständigen Fragebogen. Fachlich blockierende Entscheidungen werden spätestens vor der betroffenen Umsetzung geklärt. Externe Fakten und deren Aktualität werden nicht aus der Modellstruktur erfunden.
5. **Technik und Organisation entwickeln.** Der Agent schlägt Technologien, Verantwortungsbereiche, Modularisierung, Architektur und Arbeitsweise vor und begründet sie anhand des beschriebenen Modells. Der Nutzer erfährt, welche Standardmanager vorgesehen sind und wie Mandat, Kontext, Delegation, Reviews, Laufzeit und Modellwahl konfiguriert werden können. Eine andere technische Entscheidung ist eine nachvollziehbare Modelländerung.
6. **Lesbare Dokumentation laufend erzeugen.** Standardziel ist ein konfigurierbarer `docs/`-Ordner im Repositoryroot. Die fachliche Dokumentation wird aus dem aktuellen Modell abgeleitet und zeigt ihren Stand. Sie besitzt einen klaren Erzeugungsweg; Änderungen über Gespräch oder lesbare Sicht fließen auf das Modell zurück. Andere bewusst handgeschriebene Dokumente behalten ausdrückliche Besitzer und Beziehungen. Das heutige `.markitect/views/project.md` allein erfüllt diesen Benutzerstandard noch nicht.
7. **Bereitschaft für einen ersten Umfang feststellen.** Der Agent zeigt Modellstand, offene Entscheidungen, erforderliche Artefakte, Prüferwartungen und die vorgesehene Dateistruktur. „Vollständig genug“ bezieht sich auf einen benannten ersten Umfang, nicht auf jede zukünftige Funktion. Der Nutzer soll mindestens die vorgeschlagene Dateistruktur ansehen und bestätigen. Der bereits erteilte Auftrag kann diese Entscheidung und anschließende Umsetzung umfassen; interne Zustandswechsel brauchen keine zusätzlichen pauschalen Freigaben.
8. **Erstes Apply abschließen.** Der Benutzerablauf umfasst Planung, delegierte Implementierung, lokale Reviews, Integration, frische Checks und Übernahme. Erst ein erfolgreiches erstes Apply beendet Explore und liefert für den vereinbarten Umfang ein tatsächlich ausführbares Repository. Die vorhandenen getrennten `plan/run/verify/apply`-Schritte dürfen dafür hinter einem verständlichen Ablauf liegen; `apply` darf nicht unbemerkt seine Prüfbindungen verlieren oder unmittelbar ungeprüften Code übernehmen. Ein fehlgeschlagener Versuch beendet Explore nicht erfolgreich. Ein Apply veröffentlicht oder deployt nicht automatisch.

Die genaue Speicherung des Explore-Zustands, das Bereitschaftsformat und die ergonomische Befehlsform sind Implementierungsentscheidungen. Die Bedeutung der Phase und die überprüfbaren Übergänge sind Produktvorgaben. Spätere Exploration zu neuen Funktionen bleibt möglich, ohne das gesamte Projekt in den initialen Zustand zurückzusetzen.

## Änderungen und vollständige Dateiabdeckung

Nach dem ersten Apply beginnt eine Änderung beim Modell beziehungsweise bei einer Reparatur gegen unverändertes Soll. `plan` löst Beziehungen auf und zeigt betroffene Modellgegenstände, Manager, Artefakte, Dateien und Prüfungen. Neue, geänderte, verschobene und entfernte Realisierungen werden bei jeder Umsetzung dauerhaft mitgeführt. Übernahme bindet den finalen Modell-, Zuordnungs-, Datei- und Prüfstand gemeinsam.

Der gewünschte Applyvertrag lautet:

```text
Alle Repositorydateien
  = modellreferenzierte Realisierungen
  ∪ ausdrücklich durch .markitectignore ignorierte Dateien
  ∪ ausdrücklich klassifizierte Markitect-Dateien
```

Jede Datei muss eine eindeutige Abschlussklassifikation erhalten. Mehrere fachliche Referenzen auf dieselbe Realisierung sind zulässig. Fehlende oder widersprüchliche Klassifikation blockiert Apply. Eine Ownerzuordnung ohne fachlichen oder ausdrücklich technischen Modellbezug reicht nicht. Aktive Artefaktpflichten dürfen nicht durch Ignorieren ihrer Realisierungen beseitigt werden.

Die konkrete Grundmenge muss der Implementer explizit definieren: Git-Commit beziehungsweise Index, tatsächlich anzuwendender Kandidat und beobachteter Arbeitsbaum, einschließlich neuer und ungetrackter Dateien. Nur bekannte `inventoryRoots` zu prüfen reicht nicht, wenn dadurch andere Dateien unsichtbar bleiben. Ignorierte Buildausgaben müssen nicht als Fachmodell gelesen werden, ihre Ausschlussentscheidung muss aber nachvollziehbar sein. Die finale Prüfung bindet die angewendete Ignoreregeldatei und den Dateistand; ein neuer unbekannter Pfad nach der Vorschau macht den Abschluss veraltet.

Für `.markitectignore` sind Syntax, Musterpriorität, gegebenenfalls Negation, Pfadnormalisierung, Links, Submodule und eingebettete Repositories konkret festzulegen. `.gitignore` ist kein stiller Ersatz für diese Entscheidung. Git-Metadaten wie `.git/` beziehungsweise eine Worktree-`.git`-Datei benötigen eine dokumentierte Infrastrukturklassifikation; automatische Erkennung ähnlich bekannten Repositorywerkzeugen ist zu prüfen, nicht bereits festgelegt. Binärdateien können strukturell klassifiziert werden, ohne ihnen eine unbelegte semantische Analyse zuzuschreiben.

„Markitect-Datei“ ist eine überprüfbare, begrenzte Klasse für Modellquellen, Konfiguration, generierte Tooldateien und lokale Arbeitsdaten. Der Name `.markitect/` darf keine beliebigen fremden Dateien unsichtbar machen. Providerdateien und erzeugte Root-Dokumentation werden über ihren konkreten Erzeugungsweg und Besitzer klassifiziert. Zu breite Ignoremuster müssen bei der Einrichtung sichtbar sein; eine neue Ignoreentscheidung ist keine automatische Reparatur.

Vollständige Dateiabdeckung beweist, dass nichts unerklärt bleibt. Sie beweist noch nicht, dass jede implementierte Regel richtig ist oder kein Kommentar widerspricht. Dafür werden strukturelle Abdeckung, semantische Prüfung, ausgeführte Tests und verbleibende Lücken getrennt ausgewiesen.

## Brownfield vom Repository zum Modell

Sowohl ein chaotisches als auch ein gut strukturiertes Repository beginnt mit einer festen Aufnahme und Orientierung. Verzeichnisse, Namespaces, Module, Architekturhinweise und vorhandene Zuständigkeiten liefern Vorschläge für die fachliche und Managementstruktur. Die Dateistruktur ist ein Ausgangspunkt, keine automatisch gültige Ownershipentscheidung. Ein guter vorhandener Zuschnitt soll übernommen werden können; ein schlechter wird zunächst als beobachteter Zustand erfasst.

Zuerst wird das bestehende Repository in ein Modell überführt. Der Anwendungscode bleibt dabei unverändert. Sobald Verantwortungsbereiche geklärt sind, erhalten die Manager die jeweils relevanten Dateien und öffentlichen Nachbarverträge. Sie verfeinern das Modell iterativ anhand der vorhandenen Realisierungen und melden fehlende Begriffe, Beziehungen, Regeln, Abdeckung und Widersprüche. Der Host integriert die Modellvorschläge, prüft Struktur und Zuordnungen und führt die verfügbaren, passend gebundenen Prüfungen aus.

Beobachtetes Verhalten, dokumentierte Absicht und künftig gewünschtes Verhalten bleiben unterscheidbar. Beispiel: Der Code stellt eine zurückgegebene Stütze sofort wieder bereit, die Anleitung verlangt vorher eine Prüfung. Dieser Widerspruch wird zum Nutzer eskaliert; kein Manager darf ihn durch stilles Umschreiben des Solls lösen. Manager können fachliche Übereinstimmung beurteilen, doch ihr Einvernehmen ist kein Vollständigkeitsbeweis. Der Abschluss benennt geprüfte Zusammenhänge, verbleibende Lücken und bewusst vertagte Bereiche.

Der bestehende selektive Discovery-/Distillationsweg bleibt dafür ein Baustein. Die neue Zielrichtung ergänzt die iterative Rückführung durch dieselbe Managerorganisation. Teilübernahmen sind während der Erkundung möglich. Vor einem Apply mit dem neuen Vollabdeckungsanspruch muss der gesamte Repositorybestand klassifiziert sein; noch nicht fachlich übernommene Bereiche sind als begründete, sichtbare Übergangsausschlüsse auszuweisen, nicht als bereits konform.

Erst anschließend folgt ein gesonderter Aufräumauftrag. Ein möglicher `clean`-Befehl ist ein **Vorschlag**, noch kein festgelegter CLI-Vertrag. Er soll dieselbe Manager-, Review-, Integrations- und Prüforganisation verwenden. Sein Plan unterscheidet reine Reorganisation, Reparatur gegen akzeptiertes Soll und echte fachliche oder architektonische Änderungen. Dateiverschiebungen erhalten aktualisierte Referenzen und Ownership; vorhandenes Verhalten wird durch geeignete Regressionstests geschützt. Konflikte über das gewünschte Verhalten gehen an den Nutzer. Import des Iststands und Aufräumen werden nicht zu einer unsichtbaren gemeinsamen Mutation.

## Abstand zum vorhandenen Sourcekandidaten

Der folgende Abgleich beschreibt gelesene Quellen an der oben genannten Basis, keine neu ausgeführten Produktprüfungen. Bestehende Laufnachweise stehen in der [Validierung](../../validation/project-world-delivery.md).

| Baustein | Vorhandene Grundlage | Fehlender Teil dieser Vorgabe |
|---|---|---|
| Typisiertes Modell | `internal/core/` und `internal/modules/projectmodel/`: Compiler, Manager, Statements, Artefakte, Checks und explizite Referenzen | Geführte fortlaufende Begriffsarbeit und sichtbare Explore-Bereitschaft; stärkere Fachtypisierung bedarf gegebenenfalls eigener Schemaerweiterungen |
| Dateibezüge | `Artifact.realizes`, Pfade, FileEntry, Index und Impact | Vollständiger Repositoryvertrag statt bloß ausgewählter Inventarwurzeln; eindeutige Ignore-/Toolklassifikation |
| Initialisierung | `internal/host/projectwork/init.go` erzeugt fünf Markitect-eigene Dateien mit leerem Runtimeplaceholder und leerem Inventar | Providerwahl, native Einstiegspunkte, Skills/Workflows/Rules und zusammenhängender Gesprächsstart |
| Providerbetrieb | `internal/host/projectsetup/` und Runner konfigurieren ausgewählten lokalen Betrieb | Gemeinsames Onboarding für mehrere Contributor-Provider; keine stillen globalen Änderungen |
| Lesbare Sicht | `internal/host/projectwork/document.go` erzeugt `.markitect/views/project.md` | Konfigurierbares `docs/`-Standardziel und dessen zuverlässige Aktualisierung im Benutzerablauf |
| Delegierte Umsetzung | `internal/host/projectrun/`: Manageraufrufe, Reviews, Integration, Nacharbeit, Checks und gebundenes Apply | Zusammengesetzter Explore-bis-Apply-Ablauf und erprobte Bedienung für neue Beitragende |
| Brownfield | `internal/host/projectadoption/`: feste ausgewählte Belege, Distillation, Resolution und partielle Adoption | Iteratives Reverse-Modellieren durch die Managerhierarchie, Abschlusskriterien und gesonderter Clean-Ablauf |
| Wirtschaftlichkeit und Konsistenz | Endliche Shop- und Brownfieldversuche mit realen Provideraufrufen | Fairer wiederholter Vergleich mit konventioneller Entwicklung, einschließlich Einrichtung, Modellpflege, Reviews, Fehlversuchen und menschlicher Zeit |

Die benötigten Ausführungsbausteine sind damit weitgehend vorhanden. Die beschriebene Benutzerführung und der strenge Gesamtvertrag sind noch kein fertiges Produkt. Ein belastbarer Fertigstellungsprozentsatz oder eine Zeitschätzung folgt aus diesem Quellabgleich nicht.

## Implementierungsreihenfolge und Abnahme

Der nächste Implementer soll zuerst den Gesamtvertrag konkretisieren: Dateigrundmenge, Klassifikation, Ignoresemantik, Modellbezüge und Explore-/Applyzustände. Darauf folgen Provider-Onboarding und geführte Modellierung mit Dokumentation, dann der vollständige Greenfieldablauf, schließlich Reverse-Adoption und der getrennte Aufräumweg. Die minimalen Coregrenzen und bestehenden Kandidaten-/Prüfbindungen bleiben erhalten; keine universelle Quellcodeinterpretation im Compiler.

| Fall | Erwarteter Nachweis |
|---|---|
| Neues Projekt mit Claude und Codex | Beide Einstiegspunkte führen auf dieselben kanonischen Vorgaben; vorhandene Konfiguration bleibt erhalten; neue Beitragende brauchen keine Markitect-Befehlskenntnis |
| Ungeordnetes Gespräch mit neuen Begriffen | Modell entsteht während Explore, strukturelle Fehler werden diagnostiziert, offene Entscheidungen bleiben sichtbar und kein Umsetzungslauf startet versehentlich |
| Technologie- und Modulvorschlag | Begründete Alternativen, sichtbare konfigurierbare Managerdefaults und durch Nutzer betrachtete/bestätigte Dateistruktur |
| Erstes erfolgreiches Apply | Ausführbarer benannter Erstumfang, frische lokale und Integrationsprüfungen, aktuelle Dokumentation unter konfiguriertem `docs/`-Ziel und nachvollziehbares Ende von Explore |
| Eine Regel betrifft mehrere Repräsentationen | Impact nennt die verknüpften Code-, Test-, Dokumentations- und weiteren Artefakte; Änderungen und Prüfungen gelten für denselben finalen Stand |
| Neue, ungetrackte oder außerhalb bisheriger Wurzeln liegende Datei | Gesamtinventur erkennt sie; Apply scheitert ohne gültigen Modellbezug oder begründete Klassifikation |
| Ignoreänderung, umbenannte Datei oder gelöschtes Pflichtartefakt | Planbindung wird frisch geprüft; Bezüge bleiben nachvollziehbar; aktive Pflicht verschwindet nicht durch Ausschluss |
| Gut strukturiertes Brownfieldprojekt | Vorhandene Grenzen werden als begründeter Vorschlag genutzt; Modellübernahme ändert keine Anwendungsbytes |
| Chaotisches Brownfieldprojekt | Istzustand wird zuerst modelliert; Konflikte bleiben sichtbar und werden vom Nutzer entschieden; Aufräumen erfolgt als gesonderter geprüfter Kandidat |
| Manager melden Übereinstimmung bei unzureichenden Tests | Bericht zeigt fehlende semantische Evidenz statt uneingeschränktem Gesamt-PASS |
| Zusätzlicher Entwickler beginnt in einem frischen Chat | Geltende Regeln, Arbeitsweise und Kontext werden aus dem Repository rekonstruiert; der Workflow wird über den ausgewählten Coding Agent gefunden |
| Luna-High-Vergleich | Gleiche Aufgaben, Anforderungen und Entscheidungsspielräume; gemessene Fehler, übersehene Folgearbeit, menschliche Eingriffe, gesamte Tokenkosten und Dauer über mehrere Änderungen |

Als durchgehender Produktfall eignet sich die Stützenvermietung mit Reservierung, Ausgabe, Teilrückgabe, Beschädigungsprüfung und erneuter Verfügbarkeit. Eine spätere Regeländerung und eine zusätzliche Entwicklerperson prüfen, ob das gemeinsame Weltbild und seine Realisierungen über Zeit erhalten bleiben. Dieser Fall ergänzt die kleinen Shopbelege; er ist noch nicht ausgeführt.

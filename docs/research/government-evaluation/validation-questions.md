# Spätere Validierung und mögliche Widerlegung

**Untersuchungsentwurf, keine gestarteten Tests oder Studien.** Dieses Paket evaluiert theoretisch unter der Nutzerprämisse einer zuverlässigen kleinen Markitect-Basis. Implementierung und Experimente warten auf die vom Nutzer genannten laufenden Arbeitspakete, Case-Study-Zahlen und eine saubere stabile Version. Eine spätere Ausführung braucht einen eigenen konkreten Auftrag; dieses Dokument gibt keine Startfreigabe.

## Getrennte Nachweise

Ein Government-Controller kann alle Akten korrekt binden und trotzdem schlechte Entscheidungen treffen. Daher sind vier Nachweise getrennt nötig:

| Ebene | Was sie zeigen soll | Was sie nicht zeigt |
|---|---|---|
| Vertrags-/Mechanikprüfung | Alte Befugnis, Pflichtprüfung, Kandidatenbindung, Zustandsübergänge, Konkurrenz, Recovery und Budget funktionieren | Richtiger fachlicher Modellinhalt |
| Echte bounded Entscheidungsfähigkeit | Reale Agenten interpretieren einen Auftrag, behandeln einen Konflikt und entscheiden oder eskalieren innerhalb des Mandats | Zuverlässigkeit über viele Aufgaben oder dauerhafte Entlastung |
| Longitudinale Modellpflege | Ziele und fortgeltende Regeln bleiben über mehrere Änderungen, Ausnahmen und Revisionen verständlich und gültig | Überlegenheit gegenüber einer starken einfacheren Methode |
| Fairer Produktvergleich | Qualität, Gesamtaufmerksamkeit und Kosten im gleichen Aufgabenscope | Universelle Autonomie oder Annahme in anderen Projekten |

Fixtures sind sinnvoll für Zustandsmaschinen und Gegenfälle. Echte Actoraufrufe, gutes Fallurteil, unabhängige Produktprüfung und menschliche Annahme bleiben eigene Evidenz. Alte G1-Berichte oder synthetische Zustimmung dürfen keinen dieser späteren Nachweise ersetzen. Die aktuelle Produkt-/Releasebasis bleibt für jedes Ergebnis explizit [P1, P12, P14 und A5 im Quellenregister](source-basis.md).

## Fragen und Gegenfälle

| Frage | Späterer Fall oder Veränderungsfolge | Mögliches Scheitern |
|---|---|---|
| Wird unveränderte Absicht erkannt? | Defekten Zustandsübergang reparieren, anschließend echtes neues Verhalten anfragen | Jeder Bug wird eine Regeländerung, oder neue Absicht wird verdeckt als Reparatur umgesetzt. |
| Bleibt der Originalzweck erhalten? | Mehrdeutiges Export-Feature mit unterschiedlichen Nutzern, Daten und Berechtigungen | Ein sauber geprüftes Modell realisiert eine unbeabsichtigte Lesart. |
| Entscheidet die richtige Instanz? | Lokale Änderung, gemeinsamer Vertrag und reservierte Grundregel mit ähnlichem Wortlaut | Scope/Severity wird bequem kleingeredet oder jede Kleinigkeit geht nach oben. |
| Darf der Vorschlag sich selbst autorisieren? | Modelldelta erweitert ein Mandat oder entfernt einen Pflichtcheck | Neue Befugnis wird benutzt, bevor sie unter alter Autorität angenommen wurde. |
| Liefert ein Fachreview zusätzlichen Wert? | Sicherheits-/Architekturfall mit zunächst überzeugendem Managerentwurf | Neue Perspektiven finden nur Dubletten oder schaffen unbegründete Blockaden. |
| Können Ziele sinnvoll abgewogen werden? | Alle Varianten erfüllen harte Regeln, unterscheiden sich aber in Einfachheit, Latenz und Betriebsaufwand | Globale Gewichte verdecken lokale Fakten; Richter erfindet eine Nutzerpriorität. |
| Sind unbekannte Fakten sichtbar? | Ein entscheidender Last-/Sicherheitsbefund fehlt oder widerspricht einem Gutachten | Mehrheit oder überzeugender Text wird als Evidenzersatz akzeptiert. |
| Lernen Gründe statt bloßer Resultate? | Ähnliche Fälle mit geänderten Voraussetzungen; danach geändertes Projektziel | Alte Ausnahme wird zur allgemeinen Norm oder richtiger neuer Fall wird durch obsoleten Präzedenz blockiert. |
| Können echte Norm-/Prüfänderungen erfolgen? | Erst fehlgeschlagene Implementierung bei unverändertem Ziel, dann ausdrücklich autorisierte Zieländerung | Checks werden für Erfolgsdruck gelöscht oder eine zulässige neue Absicht bleibt unmöglich. |
| Bleibt das Modell einfach? | Mehrere lokale Features, wiederholte Ausnahme, gezielte Zusammenfassung und Streichvorschlag | Regelzahl/Redundanz wachsen; Vereinfachung verliert Schutzwirkung. |
| Bleiben konkurrierende Fälle frisch? | Zwei disjunkte Deltas mit derselben Priorität/Norm; Befugniswiderruf während Review | Dateidisjunktheit wird als semantische Kompatibilität behandelt; veralteter Beschluss wird ausgeführt. |
| Werden unbekannte Wirkungen sicher behandelt? | Unterbrechung vor/nach Entscheidung oder Apply; verlorene Antwort | Neue Run-/Fall-ID startet dieselbe Wirkung nochmals oder unbekannt wird zu PASS. |
| Sind Briefings entscheidungsfähig? | Abweichende Stellungnahme, offene Evidenz und nicht eskalierter Fehler | Zusammenfassung versteckt Unsicherheit oder zeigt Aktivität als bewiesene Qualität. |

Diese Fälle sind keine versteckten neuen Produktanforderungen. Sie zeigen, welche Behauptung eine spätere Ausbaustufe überhaupt tragen müsste. Entscheidungen außerhalb ihres vorgesehenen Scopes können regulär beim Menschen enden; ein sinnvoller Abstention-Pfad zählt nicht als technischer Fehler.

## Starke Vergleichsmodelle

Für den isolierten Zusatznutzen von Government sollte die gleiche funktionierende Markitect-Basis verwendet werden:

1. Agent erstellt Modellvorschlag, Mensch entscheidet, normale Manager/Verifier realisieren.
2. Manager dürfen dieselben eng festgelegten Routineklassen entscheiden, mit unabhängiger Prüfung und direkter Eskalation.
3. Die vorgeschlagene Government-Schicht ergänzt fallbezogene Fachperspektiven und begründete Konfliktinstanz.

Ein Präzedenzdienst und Fachreview lassen sich anschließend einzeln hinzunehmen, um ihren Grenznutzen zu untersuchen. Eine größere konventionelle Vergleichsmethode kann separat sinnvoll sein; sie ersetzt nicht den isolierten Vergleich der Zusatzinstitutionen. Alle Varianten brauchen gleiche Originalanliegen, Projektziele, zugängliche Ressourcen, tatsächliche Werkzeuge und Prüfpflichten. Vorbereitete Regeln oder kostenlose Reviews dürfen Government keinen unbezahlten Vorsprung geben.

Zwei Produktentscheidungen werden getrennt bewertet: **Variante 1 gegen 2** prüft den Nutzen delegierter Modellannahme. **Variante 2 gegen 3** prüft den zusätzlichen Nutzen der Fachinstitutionen und Konfliktinstanz. Für beide werden vor Ausführung eigene Qualitäts-/Aufmerksamkeitsschwellen festgelegt und Einrichtung, Mandate, Fallpflege, Reviews und spätere Rücknahme je Variante vollständig zugerechnet. Gewinnt der Managerarm gegen Human Approval und schlagen zusätzliche Institutionen ihn nicht, spricht das für schmale Government-Delegation und gegen ihren institutionellen Ausbau.

Aufgaben sollen echte Zielwechsel, unklare Anfragen, Reparatur, Bereichsfolgen und längere Pflegefolgen enthalten. Zufällige oder verblindete Zuordnung, wiederholte frische Akteure und neue zurückgehaltene Gegenfälle vermindern nachträgliche Anpassung. Anforderungen und Bewertung werden möglichst vor den Agentenvorschlägen festgelegt; notwendige spätere Ownerentscheidungen bleiben als solche dokumentiert. Wo es mehrere legitime Varianten gibt, darf ein Oracle nicht nur die bevorzugte Implementierung zählen. Ein einzelnes AI-Judge-Urteil ist kein ausreichender Qualitätsmaßstab.

## Was zu messen ist

Primär sind **richtige, befugte und dauerhaft zwecktreue Modellentscheidungen** sowie Qualität ihrer Realisierung. Dazu kommen übersehene Verpflichtungen, falsche Präzedenzanwendung, unerlaubte Norm-/Prüfabsenkung und Integrationsfehler. Getrennt zählen Klassifikations- und Eskalationsfehler: Weniger Fragen können echte Entlastung oder verschwiegene Risiken sein. Notwendige menschliche Entscheidung ist keine schlechte Autonomie.

Die menschliche Bilanz umfasst Aufnahme/Modellierung, Mandats- und Zielpflege, Lesen von Briefings, Fachklärung, Audit, Recovery und spätere Korrektur. Agentenwandzeit, Actorstarts, Tool-/Providerrequests und Token-/Kostenwerte werden separat erfasst; unbekannte Werte bleiben unbekannt. Einrichtung und Fehlversuche zählen mit. Ein langsamerer oder teurerer Ablauf kann bei besserer Qualität sinnvoll sein, muss diese Wirkung aber zeigen. Preis-/Modellannahmen werden an den tatsächlichen späteren Versuch gebunden.

Modellqualität braucht neben Counts unabhängige Aufgaben: Kann ein unbeteiligter Prüfer die aktuell geltende Regel und ihren Grund finden, eine spätere Änderung korrekt ableiten und obsoleten Präzedenz erkennen? Regelzahl, Textmenge oder wenige Einwände allein messen das nicht. Ausnahmen und neue Institutionen müssen an nachweislich verhindertem Fehler beziehungsweise entfallener menschlicher Tätigkeit bewertet werden.

## Entscheidung nach dem Versuch

Vor einem tatsächlichen Vergleich sollen projektbezogene Erfolgsschwellen und nicht kompensierbare Fehlerarten feststehen. Hier werden keine erfundenen Prozent- oder Zeitgrenzen gesetzt. Eine unbefugte Änderung einer reservierten Pflicht darf nicht mit schnellerem Durchsatz aufgerechnet werden. Null beobachtete Fehler in einer endlichen Stichprobe wäre trotzdem keine allgemeine Sicherheitsgarantie.

Die funktionale Empfehlung würde gestärkt durch wiederholt richtige autonome Routineentscheidungen, relevante neue Fachbefunde, zuverlässige Vorbehaltseskalation, zwecktreue Modellpflege und geringere gesamte menschliche Aufmerksamkeit bei mindestens gleicher Qualität. Sie würde geschwächt oder widerlegt durch mehr stille Fehlentscheidungen, keine zusätzliche Wirkung der Fachinstitutionen, steigende Fall-/Mandatspflege oder abgesenkte Erfolgskriterien. Dann wäre eine Reduktion auf delegierte Manager, selektives Fachreview oder menschliches Approval ein gutes Ergebnis der Untersuchung.

Das Ergebnis der Delegationsprüfung und das Ergebnis der Institutionenprüfung bleiben getrennt. Eine gute autonome Managerentscheidung ist kein Nachweis, dass ein Ministerium oder Gericht zusätzlichen Nutzen geliefert hat.

Die entscheidende offene Frage bleibt: **Kann eine begrenzte Entscheidungsschicht zusätzliche menschliche Modellarbeit übernehmen, ohne die vom Menschen gesetzten Ziele und Qualitätsmaßstäbe langsam zu verändern?** Ein fertiges Protokoll beantwortet diese Frage noch nicht.

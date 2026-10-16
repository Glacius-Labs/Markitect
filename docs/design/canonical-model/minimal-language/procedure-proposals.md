# Verfahrensvorschläge zu Kontext, Auswirkungen und semantischem Index

**Nicht gewählt und nicht normativ.** Diese Herleitung aus dem Designgespräch ist kein Teil der [Sprachspezifikation v1](language-contract.md). Kontextbildung, Änderungsbewertung, semantischer Index, Aktualisierung, Realisierungszuordnung, Prüfplanung und Apply sind Verfahrensvorschläge; ihre Records und Abläufe sind weder verpflichtende Sprachkonstrukte noch zusätzliche Core-Kinds.

Die ursprünglichen Abschnittsnummern 11.1–11.14 und historischen Notationen bleiben zur Nachvollziehbarkeit erhalten. Konkrete Beispiele, angenommene Reviews und berechnete Durchgänge belegen keine ausgeführte Produktimplementierung oder fachliche Akzeptanz. Eine spätere Auswahl muss ausdrücklich erfolgen und ihren Geltungsbereich benennen.

Herkunft und ältere Sprachvarianten stehen in der [Sprachgeschichte](language-history.md); aktuelle ausführbare Evidenz steht beim [Exchange-Beispiel](exchange-example/README.md).

---

## 11. Prüfung und Realisierung

| Aufgabe | Aussagegrenze |
| --- | --- |
| Strukturprüfung | Identitäten, geschlossene Felder, Werttypen, Kardinalitäten, Meta-Verträge und Referenzziele. |
| Semantisches Modellreview | Verständlichkeit, fachliche Eignung, Widersprüche, Geltungsbereich und fehlende Festlegungen. |
| Realisierungsprüfung | Fester Umsetzungskandidat gegen den geltenden Modellstand; geeignete Tests, Analysen und unabhängige Bewertung. |

Ein gültiger Graph beweist keine korrekte MessageId-Zuordnung, monotone Zeitmessung oder Mandantentrennung. Eine neue Pflicht im Kind-body gilt auch bei unveränderten konkreten Definitionsdateien. Nicht untersuchte Pflichten bleiben offen. Abgeleitete Prüflisten sind Arbeitsergebnisse und keine zweite kanonische Wahrheit.

Kontextbildung, fachliche Änderungsbewertung, Delegation, Integration und Apply bleiben notwendige Markitect-Aufgaben. Dieser Sprachvertrag legt deren neue generische Anbindung nicht bereits vollständig fest und führt ihre Ausführungsdaten nicht als vorgeschriebenes Projektvokabular ein. Bestehende Produktpflichten werden durch diese Sprachkonsolidierung nicht nebenbei aufgehoben.

**Notation der nachfolgenden Verfahrensherleitung:** Die Indexdurchgänge aus der vorherigen Session behalten ihre damaligen Paket-IDs, Schrägstrich-IDs und alias:id-Beispiele, insbesondere wo sie konkrete historische Probedateien zitieren. Sie illustrieren die Indexverfahren, nicht die nun gewählte Autorennotation. Für neue Definitionsdateien und Referenzen gelten die Package-/Importregeln mit / und Alias.id aus Abschnitt 7 und 8; eine Übertragung dieser Beispiele verlangt ausdrücklich angepasste Namen und Referenzen. Die historischen Probeberichte werden nicht umgeschrieben.

### 11.1 Vorschlag: Kontext als begründete Zusammenstellung

**Status von Abschnitt 11 ab hier: beauftragte Ausarbeitung, noch keine gewählte Verfahrensentscheidung oder implementierte generische Kontext-/Impactberechnung.** Der Vorschlag nutzt die gewählte Sprache, ohne neue Pflichtbegriffe wie Requirement, Rule oder Manager in ihr einzubauen. Er konkretisiert für das neue Modell die Aufgaben, die das bestehende Produkt mit seinem festen Projektvokabular bereits bearbeitet. Er ersetzt dessen akzeptierte Routing- und Prüfpflichten nicht durch eine Dokumentänderung.

Kontext beantwortet: Welche Festlegungen und Zusammenhänge muss man für die betrachtete Aufgabe verstehen? Auswirkung beantwortet: Welche Gegenstände und Realisierungen müssen wegen dieser Änderung erneut beurteilt werden? Kontextrelevanz ist weder eine Änderungsanweisung noch der Nachweis, dass etwas unverändert bleiben kann.

Für eine betrachtete Definition stellt Markitect zunächst bereit:

1. Die Definition mit vollständigem body und spec sowie ihrer festen Identität und Bindung.
2. Ihren Kind und dessen vollständige verwendete Propertyverträge, einschließlich geschachtelter object-Verträge und der bodies. Diese Festlegungen gelten auch ohne Wiederholung in der konkreten Definition.
3. Die referenzierten Gegenstände aus spec und die expliziten Körperlinkziele mit ihren benötigten Verträgen. Dabei bleibt erkennbar, ob etwa api ein konkreter Teilnehmer oder Container dessen Kind ist.
4. Eingehende Bezüge und weitere Aussagen als zu beurteilende Kontextkandidaten. Eine anderswo beschriebene Pflicht kann den betrachteten Gegenstand betreffen.

Die Zusammenstellung nennt für jeden Eintrag den Grund und die Quelle: beispielsweise „Exchange ist der Kind von list-documents“, „Exchange/sender ist dessen Auftraggeber-Property“ oder „list-documents referenziert api als requester“. Gemeinsame Verträge werden einmal bereitgestellt und aus allen betroffenen Stellen referenziert. Der feste Meta-Vertrag kann gemeinsam vorliegen, statt bei jedem Gegenstand rekursiv nochmals ausgeschrieben zu werden.

Das Modell bleibt vollständig adressierbar. Weiterführende Verträge können bei Bedarf nachgeladen werden. Eine begrenzte erste Kontextlieferung ist keine abgeschlossene Vollständigkeitsprüfung: noch zu beurteilende Kandidaten und fehlender Kontext müssen sichtbar bleiben. Zusammenfassungen dienen der Orientierung; sie ersetzen die verbindlichen Originalaussagen nicht.

### 11.2 Strukturell ableitbare Kandidaten einer Änderung

Die Auswirkungsanalyse vergleicht zwei feste Modellstände. Sie erfasst hinzugefügte, entfernte und geänderte Definitionen, bodies, Propertyverträge, Werte, Referenzvorkommen und Paketbindungen. Unveränderte konkrete Quelldateien bedeuten keine unveränderten geltenden Verträge. Ein Paketupdate wird anhand seiner gebundenen Inhalte betrachtet, nicht nur anhand seiner Versionszeichenfolge.

Der strukturelle Teil kann folgende Gründe berechnen:

| Ausgangspunkt | Sicher auffindbare Kandidaten beziehungsweise Prüfaufgaben |
| --- | --- |
| Geänderte Definition | Die Definition selbst und ihre ein- und ausgehenden Bezüge. Die Bezüge allein beweisen noch keine fachliche Auswirkung auf jedes Nachbarziel. |
| Geänderter Kind-Vertrag, auch nur im body | Seine Definitionen und deren nun geltende Verträge. |
| Geänderte Property, auch wiederverwendet | Alle Kinds beziehungsweise object-Verträge, die sie verwenden, und die davon betroffenen Definitionen. Auch ein fehlender optionaler Wert kann bei verändertem Vertrag erneut prüfbedürftig sein. |
| Geänderter reference-Wert | Quellfeld, Feldvertrag sowie bisheriges und neues Ziel als Kandidaten. |
| Geändertes Ziel eines Körperlinks oder gelöschtes Ziel | Die bisherigen und neuen Linkvorkommen beziehungsweise Quellen; fehlende Ziele werden als Strukturfehler diagnostiziert. |

Für gelöschte oder umgehängte Bezüge wird auch der alte Graph verwendet. Sonst könnte etwa die bisherige API aus dem betrachteten Bereich verschwinden, gerade weil requester auf eine andere Anwendung umgestellt wurde. Ein neuer strukturell ungültiger Stand liefert zunächst Diagnosen, keinen gültigen Realisierungsauftrag.

Die Kantenarten bleiben unterscheidbar: Typisierung, Vertragsverwendung, typisierte Wertreferenz und Körperbezug. Eine vollständige ungerichtete Erreichbarkeit würde bei gemeinsamen Kinds oder dem Core fast das ganze Modell erfassen. Deshalb ist sie kein Beweis fachlicher Auswirkung. Das Verfahren erweitert und beurteilt Kandidaten anhand ihrer Gründe; es behandelt nicht jeden Graphpfad als fachliche Abhängigkeit.

### 11.3 Semantische Geltung aus Prosa beurteilen

Ein body kann schreiben: „Die durch requester bestimmte Anwendung muss verspätete Antworten verwerfen.“ Der Core kennt requester als deklarierte Property und kann die zugehörigen Anwendungen finden. Er kennt die Bedeutung dieser Pflicht aber nicht allein aus dem Namen oder Link. Die Anwendung des Satzes auf api erfordert eine semantische Beurteilung des gesamten Vertrags und der konkreten Belegung.

Eine solche Beurteilung liefert abgeleitete Ergebnisse mit Herkunft: welche Aussage welchen Gegenstand unter welchen Bedingungen betrifft, welche weitere Definition dafür benötigt wird und welche Frage offen bleibt. Das sind Arbeits- und Prüfergebnisse gegen einen festen Modellstand, keine zweite manuell gepflegte Spezifikation und keine zusätzlich verlangten Core-Definitionen. Sie dürfen die akzeptierte Prosa nicht umdeuten oder neue Architekturentscheidungen still einführen. Widersprüche oder Entscheidungen außerhalb der eingeräumten Freiheit gehen an den Nutzer.

Es gibt eine unvermeidliche Grenze: In freier Prosa kann auch ohne Link stehen, dass alle externen Zugriffe eine bestimmte Authentifizierung benötigen. Eine Suche nur entlang expliziter Kanten kann diesen Geltungsbereich nicht vollständig ermitteln. Für einen belastbaren begrenzten Kontext muss Markitect daher die Aussagen des gesamten relevanten Modellumfangs auf solche Geltungsbereiche untersuchen oder bereits geprüfte, passend gebundene Ableitungen verwenden. Wortsuche kann dabei helfen; ein fehlender Suchtreffer beweist keine Nichtbetroffenheit.

Ein daraus erzeugter Index kann Kontext wiederholt bereitstellen, ohne jedem Agenten das gesamte Modell zu senden. Er muss Aussagen und Fundstellen, betrachtete Reichweite, Begründungen und Unsicherheiten erhalten. Neue oder geänderte bodies, Verträge, Belegungen und Definitionen können die Zuordnung verändern. Insbesondere darf eine neue passende Definition nicht aus einer zuvor bewerteten Aussage über „alle“ herausfallen. Wiederverwendung braucht daher nachgewiesen passende Eingaben und Reichweite; die vollständige Invalidierungsregel ist noch auszuarbeiten.

Diese semantische Erschließung ist fehlbar und benötigt unabhängiges Review. Kann sie einen Bereich nicht begründet eingrenzen, bleibt der größere Bereich zur Prüfung offen; erforderlichenfalls ist das der gesamte modellierte Umfang. Eine Begrenzung durch ein Kontextbudget darf nicht als fachlicher Ausschluss erscheinen. Die minimale Sprache ermöglicht die Beschreibung; sie macht deren vollständiges Verständnis nicht deterministisch.

### 11.4 Exchange-Gegenproben

Für list-documents aus Abschnitt 4 gehören dessen sieben Propertyverträge, der Exchange-body und die konkret referenzierten Teilnehmer, Command- und Channelverträge zum Ausgangskontext. Eingehende Aussagen und die Erschließung aus Abschnitt 11.3 ergänzen ihn. Am Beispiel ergeben sich diese unterschiedlichen Fälle:

| Änderung | Struktureller Grund | Inhaltlich zu klärende Folge |
| --- | --- | --- |
| timeoutSeconds von 30 auf 60 | Geänderter Wert samt Propertyvertrag und Exchange. | Die API verwendet als requester die Wartefrist; ihre Realisierung und passende Prüfungen sind zu untersuchen. Eine Codeänderung am Broker folgt daraus nicht automatisch. |
| Exchange.body verlangt nun monotone Zeitmessung | Geänderter Kind-Vertrag betrifft alle Exchange-Definitionen. | Für jede Exchange-Belegung die Auftraggeberpflicht bestimmen; bei list-documents betrifft sie api, obwohl dessen Definitionsdatei unverändert ist. |
| sender wird bei gleicher Property-ID zu requester | Geänderter Feldvertrag betrifft seine Verwender; konkrete alte Schlüssel sind Strukturfehler. | Modellangaben migrieren. Ob Softwarefeldnamen mitgeändert werden müssen, hängt von den beabsichtigten Schnittstellen und der bisherigen Realisierung ab. |
| Topicbildung im body von requests ändert sich | Geänderter Channel und seine Verbraucherbezüge, darunter list-documents. | Auftragspublikation und passende Subscriptions auf API- und Connectorseite gemeinsam beurteilen; Brokerkonfiguration nur soweit sie davon abhängt. |
| Linkanzeige oder title ändert sich | Änderung wird erhalten, Identitäten und referenzierte Werte bleiben gleich. | Darstellungsänderung oder Bedeutungsänderung prüfen; daraus folgt keine pauschale Softwareänderung. |

Der nachvollziehbare Grundpfad der Zeitmessung lautet: geänderte Pflicht im Exchange-body → list-documents hat diesen Kind → requester ist durch Exchange/sender deklariert → spec.requester verweist auf api. Die ersten und letzten Zuordnungen sind strukturell prüfbar; dass die konkrete Zeitmessung eine Pflicht dieser Anwendung ist, stammt aus der semantischen Auslegung des Satzes.

### 11.5 Von Modellbetroffenheit zu Arbeit und Nachweisen

Die Ergebnisse unterscheiden mindestens einen festgestellten Strukturfehler, eine noch zu beurteilende mögliche Folge, einen benötigten Kontext und eine begründete Realisierungs-/Prüfaufgabe. Ein Ergebnis „keine Umsetzung nötig“ benötigt ebenfalls einen zum Fall passenden Grund; es darf nicht aus einer fehlenden direkten Kante abgeleitet werden. Diese Unterscheidung ist kein neuer Pflichtenkatalog im kanonischen Modell und keine Erlaubnis, bisher verlangte Kontextprüfungen wegzulassen.

Aus dem Modell allein folgen noch keine konkreten Dateipfade oder vollständigen ausgeführten Tests. Dafür braucht Markitect die gebundene Zuordnung der vorhandenen Realisierung zu den Modellgegenständen, den tatsächlichen Umsetzungskandidaten und geeignete Nachweise. Diese Daten gehören zur Realisierungs-/Apply-Ebene. Manuell oder maschinell ermittelte Zuordnungen benötigen eine ausgewiesene Grundlage und Abdeckung. Für eine neue Pflicht können Realisierung und Nachweise bisher fehlen.

Unbekannte Zuordnungen bleiben Untersuchungsaufgaben; ein Pfad darf nicht nur aus einem ähnlichen Namen erraten werden. Beobachtete Codeabhängigkeiten können den betrachteten Bereich erweitern und Modelllücken aufdecken, ändern aber nicht still die beabsichtigte Welt. Der [akzeptierte Problemstand zu Codeabhängigkeiten](../../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact) bleibt relevant.

### 11.6 Bindung, Nachvollziehbarkeit und noch nötige Gegenproben

Ein abgeleiteter Kontext oder Auswirkungsbericht nennt seine Modellstände, Paketbindungen, betrachteten Umfang, Gründe mit Fundstellen, semantischen Beurteilungen, offenen Bereiche und gegebenenfalls die verwendete Realisierung. Bei geänderten Eingaben wird seine Gültigkeit erneut geprüft. Zyklen dürfen keine endlose Verarbeitung erzeugen; eine Begrenzung meldet offene Arbeit. Eine spätere Zusammenstellung muss zusätzliche Erkenntnisse zur Prüfung aufnehmen können, ohne bestehende ungeklärte Gründe still zu verlieren.

Das sind Verfahrensanforderungen für einen nächsten Entwurf. Noch festzulegen und zu erproben sind insbesondere Umfang und unabhängige Prüfung der semantischen Erschließung, sichere Wiederverwendung, Erweiterung des Kontexts und die Bindung zur Realisierung. Die Tabellen sind gedankliche Gegenproben, kein Testbericht eines neuen generischen Impactverfahrens. Ein Implementierungsversuch müsste unter anderem neue globale Pflichten ohne Links, neue Instanzen unter alten Pflichten, entfernte Referenzen, wiederverwendete Properties, widersprüchliche Aussagen und fehlende Realisierungszuordnungen abdecken.

### 11.7 Semantischer Index: Quelle, Auslegung und konkrete Geltung

Der nächste Nutzerauftrag konkretisiert, was im Index steht, wie seine Geltungsableitungen geprüft werden und wann sie erneut zu beurteilen sind. Auch diese Ausarbeitung ist ein Verfahrensvorschlag. Der Index ist ein wiederherstellbares Arbeitsergebnis zu gebundenen Modellständen, kein zusätzlicher kanonischer Modellteil und keine im Core eingebaute Ontologie von Pflichten.

Am Beispiel der monotonen Zeitmessung sind drei getrennte Ergebnisse nötig:

| Ergebnis | Konkreter Inhalt |
| --- | --- |
| Quellenbefund | Der Satz im body von `(example/connector-vocabulary, Exchange)`, seine genaue Fundstelle und der unveränderte umgebende Vertrag. |
| Geprüfte Auslegung | Die Zeitmessungspflicht betrifft die durch die Property `(example/connector-vocabulary, Exchange/sender)` bestimmte Anwendung bei jedem Exchange und pro Versuch. |
| Konkrete Geltung | Bei `(example/connector-project, list-documents)` ist diese Anwendung `(example/connector-project, api)`. Die Pflicht betrifft api in diesem Austausch; sie ist damit nicht pauschal eine Pflicht sämtlicher Vorgänge dieser Anwendung. |

Im [gebundenen Vokabularbeispiel](identities/vocabulary-2.0.0.yaml) lautet der Quellsatz tatsächlich: „Der Sender muss die Wartefrist jedes Versuchs anhand monotoner Zeitmessung bestimmen.“ Dass Sender hier der heute requester genannten Property entspricht, ist eine zu prüfende Auslegung anhand des Vertrags. Die stabile Property-ID Exchange/sender oder Wortähnlichkeit beweist diese Bedeutung nicht. Die historische Beispieldatei wird für diese Untersuchung nicht umgeschrieben.

Ein Auslegungseintrag hält mindestens Folgendes fest:

- **Quelle:** kanonische Definitionsidentität, gebundener Inhalt, body-Fundstelle und Originalaussage. Die Fundstelle ist eine Herkunftsangabe gegen diesen Inhalt, keine neue dauerhafte Sprach-ID für jeden Satz. Bei nicht eindeutig wiederauffindbarer Passage wird die Zuordnung neu geprüft.
- **Auslegung und Grenzen:** die Bedeutung in eigenen Worten, die gemeinten Begriffe und Rollen, Bedingungen, Ausnahmen und der betrachtete Geltungsbereich. Bei dieser Aussage gehören „pro Versuch“ und „Auftraggeber dieses Austauschs“ ausdrücklich dazu.
- **Grundlage:** verwendete Kind-/Propertyverträge, individuelle Festlegungen, Referenzwerte und sonstiger gelesener Kontext; strukturelle Tatsachen bleiben von semantischen Schlussfolgerungen unterscheidbar.
- **Prüfung:** vorgeschlagen, unabhängig bestätigt, beanstandet oder offen; mit Begründung und gebundenem Prüfgegenstand. Die Feststellung, dass ein Satz eine verbindliche Pflicht und keine bloße Erläuterung ist, gehört ebenfalls zur Auslegung.

Für jede konkrete Anwendung werden zusätzlich der betrachtete Austausch, das Rollen-Ziel, die Bedingungsbelegung, der strukturelle Grundpfad und das Ergebnis festgehalten. Geltung und Prüfzustand sind verschiedene Angaben: „gilt“ ist zunächst eine Behauptung; sie kann noch ungeprüft sein. Ein negatives Ergebnis lautet beispielsweise „aus dieser Aussage keine Zeitmessungspflicht allein aufgrund der receiver-Rolle“, nicht „onprem hat keine Pflichten“.

Für den Beispielstand lautet ein lesbarer Anwendungseintrag:

| Angabe | Wert |
| --- | --- |
| Aussage | Monotone Zeitmessung der Wartefrist jedes Versuchs. |
| Herkunft | Exchange.body im gebundenen Vokabularpaket. |
| Betrachteter Austausch | list-documents im Projektpaket. |
| Rolle | Property Exchange/sender, Feldname requester. |
| Betroffener Gegenstand | api im Projektpaket. |
| Grundpfad | list-documents.kind → Exchange; Exchange verwendet Exchange/sender; list-documents.spec.requester → api. |
| Geltungsbehauptung | api muss für diesen Austausch die Wartefrist jedes Versuchs monoton bestimmen. |
| Prüfzustand dieses Beispiels | Vorschlag im Designgespräch; keine bereits ausgeführte unabhängige semantische Prüfung. |
| Realisierungsnachweis | Nicht Bestandteil dieses Geltungsbefunds; noch nicht geführt. |

Eine allgemeine Auslegung darf eine wiederverwendbare Auswahl wie „alle Definitionen dieses Kinds, jeweils ihre durch diese Property bestimmte Anwendung“ beschreiben. Nur wenn diese Auswahl ausdrücklich gegen die Quelle geprüft und mit festgelegten Operationen ausführbar ist, darf Markitect ihre konkreten Treffer anschließend deterministisch bestimmen. Ein vom Agenten frei erzeugter Ausdruck wird nicht allein durch seine Existenz zu vertrauenswürdiger Compilersemantik. Der Core interpretiert weder das Wort requester noch diesen Satz als fest eingebaute Operation. Für komplexere Prosa bleibt die Auswahl selbst semantische Arbeit.

### 11.8 Die abgeleitete Geltung unabhängig prüfen

Die Prüfung hat drei Aufgaben mit unterschiedlichen Aussagegrenzen:

1. **Strukturelle Prüfung:** Die Quellfassung ist gebunden und die Fundstelle passt; Exchange ist ein Kind; list-documents hat diesen Kind; Exchange/sender ist dessen gültiger Propertyvertrag; der aktuelle Feldname ist requester; der Referenzwert bezeichnet api. Das lässt sich gegen den kompilierten Stand prüfen.
2. **Semantische Prüfung:** Ein unabhängiger Reviewer erhält Originalaussage, vollständige benötigte Verträge, konkrete Belegung und relevante weitere Aussagen. Er prüft selbst, ob Sender hier requester meint, die Pflicht je Versuch gilt, Bedingungen oder Ausnahmen fehlen und die Anwendung auf api korrekt ist. Er erhält nicht nur die Zusammenfassung des ersten Agenten. Nicht auflösbare Mehrdeutigkeit bleibt offen; ein echter Widerspruch wird nicht durch eine Indexentscheidung überstimmt.
3. **Abdeckungsprüfung:** Betrachtet wurden auch mögliche ausgelassene Aussagen und Fälle. Nur vorgeschlagene positive Treffer zu bestätigen, reicht nicht. Gegenproben müssen eine Vertauschung von requester und receiver, einen weiteren Exchange, dieselbe Anwendung in anderer Rolle und zusätzliche Einschränkungen im individuellen body unterscheiden. Das prüft die Auslegung; es führt die Zeitmessung der Software noch nicht aus.

Zur Abdeckung gehört ein eigener Nachweis des untersuchten Umfangs: Welche Quell-bodies wurden erschlossen, welche Aussagen wurden noch nicht geklärt, welche Definitionen beziehungsweise Kandidatengruppen wurden betrachtet, und welche Ausschlüsse sind mit welchem Grund geprüft? Gleich begründete Fälle dürfen gruppiert werden; die Gruppe muss am festen Stand eindeutig bestimmt sein. Im Beispiel kann die Auswahl aller Exchange-Definitionen gebunden werden, statt zu behaupten, die heutige Trefferliste sei für immer vollständig. Bei der anfänglichen Erschließung muss auch Prosa außerhalb dieser ausgewählten Gruppe auf weiter reichende Geltung untersucht werden.

Ein offener Kandidat ist keine negative Feststellung. Auch ein Review des falschen oder unvollständigen Kandidatenraums beweist keine Vollständigkeit. Die Abdeckungsangabe bezeichnet daher den tatsächlich untersuchten Raum und verbleibende Unsicherheit. Die unabhängige Prüfung verringert Fehlerrisiken; sie beweist nicht die vollständige Erfassung beliebiger Prosa.

### 11.9 Änderungen: Auslegung, Anwendung und Abdeckung getrennt erneuern

Für einen neuen Modellstand gelten bisherige Ergebnisse zunächst nicht ungeprüft als aktuell. Markitect vergleicht die Eingaben und unterscheidet, welche Auslegung, konkrete Anwendung oder Abdeckung neu zu beurteilen ist. Alte Ergebnisse bleiben als Ergebnisse des alten Stands erhalten. Eine Neuauswertung kann dasselbe Resultat liefern; ihre aktuelle Gültigkeit benötigt trotzdem einen nachvollziehbaren Grund.

| Änderung | Erforderliche Neubetrachtung im Beispiel |
| --- | --- |
| Der Quellsatz oder sein Bedeutungsumfeld ändert sich | Auslegung semantisch erneut prüfen; ihre Anwendungen und betroffene Abdeckung anschließend erneuern. Ein unveränderter Satz kann durch eine neue Ausnahme im selben Vertrag eine andere Bedeutung erhalten. |
| Property.body, Typ oder Rollenbedeutung ändern sich | Die Zuordnung der Aussage zu Exchange/sender erneut beurteilen; gegebenenfalls alle davon abhängigen Anwendungen. Stabile Identität beweist keine stabile Bedeutung. |
| requester wechselt von api zu einer anderen Anwendung | Anwendung für diesen Exchange erneuern; bisheriges und neues Ziel berücksichtigen. Die geprüfte allgemeine Auslegung kann bestehen bleiben, wenn ihre vollständigen Grundlagen weiterhin passen. |
| Ein neuer Exchange wird hinzugefügt oder eine Definition erhält diesen Kind | Auswahl und Abdeckung erneut auswerten, die neue Rollenbelegung prüfen und eine neue Anwendung bilden. Eine unveränderte Liste bisheriger Treffer genügt nicht. |
| Ein Exchange oder dessen Rollenbezug entfällt | Bisherige Anwendung am neuen Stand entfernen beziehungsweise als nicht mehr geltend kennzeichnen. Andere Anwendungen derselben Pflicht auf api bleiben separat bestehen. |
| timeoutSeconds ändert sich von 30 auf 60 | Die Aussage „monotone Zeitmessung pro Versuch“ kann unverändert gelten, sofern ihre Auslegung wertunabhängig ist. Die neue konkrete Festlegung und die Realisierung sind trotzdem zu prüfen; ein unveränderter Geltungsbefund ist kein unveränderter Prüfnachweis. |
| sender wird bei gleicher Property-ID zu requester | Strukturelle Zuordnung unter dem neuen Feldnamen erneut prüfen; terminologische Auslegung und Fundstellen bei Bedarf semantisch erneuern. Keine bloße Zeichenersetzung als Bedeutungsbeweis verwenden. |
| Eine andere Aussage führt eine Ausnahme oder globale Pflicht ein | Betroffene Auslegungen, Anwendungen und die Abdeckung auch ohne direkte Links neu beurteilen. Ist die Reichweite unklar, den weiteren modellierten Umfang einbeziehen. |
| Ein Import wird aktualisiert | Gebundene Inhalte und die betrachteten Definitionsmengen vergleichen. Gleiche Paket-/Definitionsidentitäten reichen für Wiederverwendung nicht aus. |
| Nur ein Dateiname im selben Package oder Importkürzel ändert sich | Kanonische Identitäten bleiben bestehen; Herkunft und gegebenenfalls body-Quelltext neu abbilden. Eine belegte reine Darstellungsänderung benötigt keine neue fachliche Schlussfolgerung. |
| Nur die Software ändert sich | Modellgeltung kann bestehen bleiben; der Realisierungsnachweis muss zum neuen Kandidaten passen. |

Zur sicheren Wiederverwendung gehören damit nicht nur tatsächlich gelesene Definitionen. Festzuhalten sind auch die Auswahlräume und negative Voraussetzungen, auf denen ein Ergebnis beruht: etwa „alle Exchange-Definitionen dieses Modellumfangs“ oder das geprüfte Fehlen einer relevanten Ausnahme. Sonst bleibt ein alter Index scheinbar gültig, wenn eine neue passende Definition oder entgegenstehende Aussage hinzukommt. Ein Agentenbericht mit einer Liste gelesener IDs ist dafür allein unzureichend.

Als erste konservative Verfahrensregel wird vorgeschlagen: Jede Modelländerung löst eine Aktualitätsprüfung des Index aus; Wiederverwendung erfolgt nur mit belegbar unveränderter Grundlage und Abdeckung. Ist diese Abhängigkeit noch nicht belastbar abgegrenzt, wird der größere semantische Prüfbereich erneut beurteilt. Das bedeutet nicht, bei jedem geänderten Wert jeden Satz neu auszulegen. Es verhindert aber, dass eine unbewiesene lokale Begrenzung als sicherer Cachetreffer behandelt wird. Die genaue Datenkodierung, Auswahlausführung und der Nachweis vollständiger Wiederverwendungsgrundlagen bleiben Implementierungsfragen.

### 11.10 Durchgang: zweiter Exchange und Wechsel seines requester

Auf Nutzerauftrag wird der vorgeschlagene Index nun über drei feste Modellstände aktualisiert. Dies ist ein vollständiger gedanklicher Durchgang für die betrachtete Zeitmessungspflicht, kein ausgeführter semantischer Prüflauf und keine Behauptung über die gesamte Software. Für den Ablauf wird eine zuvor unabhängig bestätigte allgemeine Auslegung angenommen; die bisherigen Berichte enthalten eine solche Bestätigung nicht.

**Abgegrenztes Beispiel.** E1 bezeichnet im Folgenden list-documents mit requester api und receiver onprem. E2 ist ein neuer Exchange check-status. Benötigte Unterstützungsdefinitionen werden für den Ausgangsstand vorausgesetzt: status-service als zulässiger Container, check-status-command als lesender Statusvertrag sowie status-requests und status-replies als zulässige Channel-Definitionen. Deren Verträge erlauben autorisierte Auftraggeber beider Standorte, instanzbezogene Antworten an den jeweiligen requester und die passende MessageId-Zuordnung. Sie schreiben weder api als einzigen Auftraggeber noch eine Ausnahme von der Zeitmessungspflicht vor. Wenn diese Definitionen erst neu angelegt werden müssten, gehörten auch sie mit ihren eigenen Aussagen zur zu prüfenden Änderung.

Diese eigenen Kanäle vermeiden eine versteckte Annahme: Die bisherigen Kanäle requests und replies enthalten unter anderem Aussagen über die API als Sender. Der spätere Rollenwechsel ist deshalb nicht einfach als bereits kompatibel mit diesen älteren Kanalverträgen zu behandeln.

Die vollständige E2-Definition im Stand M1 lautet unter diesen Voraussetzungen:

```yaml
id: check-status
kind: connector:Exchange
body: >-
  Der requester fragt den Betriebsstatus beim receiver ab.
  Die Operation ist lesend. Antworten behalten die MessageId des Requests.
  Verspaetete Antworten ohne Wartezustand werden verworfen.
spec:
  requester: api
  receiver: status-service
  command: check-status-command
  requestChannel: status-requests
  replyChannel: status-replies
  timeoutSeconds: 10
  maxAttempts: 2
```

M2 ändert ausschließlich E2.spec.requester von api zu onprem. Der rollenbezogene body bleibt stehen. Die Propertyidentität bleibt `(example/connector-vocabulary, Exchange/sender)`; api, onprem und beide Exchanges liegen im Projektpaket. E1, die Paketbindungen und die allgemeinen Kind-/Propertyverträge sind in allen drei Ständen unverändert. M0, M1 und M2 sind Bezeichnungen dieses Durchgangs, keine neuen Core-Gegenstände.

**M0: gebundener Ausgangspunkt.** Der Index enthält die angenommene geprüfte Auslegung „requester jedes Exchange bestimmt die Wartefrist pro Versuch monoton“. Für den ausgewählten Exchange-Bereich wurde die Menge `{E1}` betrachtet. Die konkrete Anwendung lautet „api für E1“. Quellen, Auslegung, Auswahlumfang, individuelle Aussagen und geprüfte Einschränkungen sind an M0 gebunden. Es gibt keine negative Feststellung, dass onprem niemals Auftraggeber sein könnte.

**M0 → M1: E2 hinzufügen.** Die Aktualisierung durchläuft diese Schritte:

1. **Änderung und Struktur prüfen.** E2 ist hinzugekommen. Seine Identität ist frei, sein Kind ist Exchange; alle sieben Felder, Kardinalitäten und Zieltypen sind gültig zu prüfen. Die Angabe requester wird über den aktuellen Propertyvertrag auf api aufgelöst. Aus einer lediglich gültigen Container-Referenz folgt noch keine fachliche Zulässigkeit als Auftraggeber.
2. **Neue Aussagen erschließen.** E2.body wird erstmals vollständig beurteilt. Der Prüfer untersucht zusammen mit den benötigten Verträgen, ob der neue Text eine Ausnahme, einen Widerspruch oder eine Aussage mit Wirkung über E2 hinaus enthält. Erst nach dieser Untersuchung kann die Änderung semantisch lokal eingegrenzt werden. In diesem gewählten Beispiel soll sie keine weitere globale Festlegung einführen; dies ist eine zu prüfende Voraussetzung, kein Schluss aus dem kleinen Textdiff.
3. **Auslegung wiederverwenden, Auswahl erneuern.** Die allgemeine Zeitmessungsauslegung kann erhalten bleiben, wenn ihre Quellen und Bedeutungsgrundlagen unverändert sind und die neue Aussage sie nicht verändert. Die Auswahl ist dagegen jetzt `{E1, E2}`. Ihr alter Abdeckungsbefund passt nicht unverändert auf M1. Eine geprüfte ausführbare Auswahl darf die Menge neu ermitteln; andernfalls ist diese Auswahl erneut semantisch zu beurteilen.
4. **Neue Anwendung prüfen.** Für E2 entsteht der Vorschlag „api für E2“. Der strukturelle Grundpfad ist neu zu prüfen; die semantische Prüfung betrachtet die individuelle Beschreibung, Teilnehmer-, Command- und Channelverträge sowie weitere einschlägige Aussagen. Bei bestätigter Anwendbarkeit erhält dieser Fall seinen eigenen aktuellen Prüfzustand. Eine bloße Kopie des alten Eintrags „api für E1“ genügt nicht.
5. **Bestehende Ergebnisse übernehmen.** Die Anwendung „api für E1“ kann nach der Aktualitätsprüfung an M1 übernommen werden: unveränderte Grundlagen und keine neu ermittelte entgegenstehende Aussage. Ihre frühere Prüfung bleibt Herkunft; ein neuer Übernahmebefund erklärt ihre Gültigkeit am neuen Stand. Unveränderte Bytes allein wären bei einer neuen globalen Ausnahme unzureichend.
6. **Kontext und Arbeit bilden.** Der API-Kontext enthält nun zwei an verschiedene Exchanges gebundene Anwendungen. Für E2 entstehen Realisierungs-/Prüfaufgaben; bisherige Nachweise für E1 decken E2 nicht automatisch ab. Ob gemeinsame Implementierung oder Integration weitere Prüfungen für E1 verlangt, ist anhand der tatsächlichen Realisierung zu beurteilen.

**M1 → M2: E2.requester von api auf onprem ändern.** Wieder sind mehrere Teile getrennt zu behandeln:

1. **Alten und neuen Bezug vergleichen.** Es gibt keine neue Exchange-Definition und keine Änderung der Auswahlmenge `{E1, E2}`. Geändert ist die Rollenbelegung von E2. Markitect behält das alte Ziel api für die Änderungsbewertung und prüft das neue Ziel onprem gegen den Container-Vertrag. Ein ungültiger neuer Bezug blockiert einen gültigen neuen Modellstand.
2. **Allgemeine Auslegung auf Aktualität prüfen.** Die Aussage betrifft weiterhin den jeweiligen requester, unabhängig von dessen Identität oder Standort. Unter den genannten unveränderten Grundlagen kann die allgemeine Auslegung wiederverwendet werden. Das ist keine neue Deutung des Wortes requester aus seinem Namen.
3. **Konkrete Anwendung neu bestimmen und prüfen.** „api für E2“ gilt in M2 nicht mehr. Dafür entsteht „onprem für E2“. Die Auftraggeber-Eignung, Autorisierung, Antwortadresse, Wartezustandsverwaltung und etwaige Einschränkungen durch andere Verträge werden für die neue Belegung beurteilt. Bei Widersprüchen bleibt der Fall beanstandet oder offen; der Index erteilt keine Ausnahme von der Architektur. Unter den gewählten kompatiblen Verträgen wird die Zeitmessungspflicht für onprem in E2 bestätigt angenommen.
4. **Abdeckung aktualisieren.** Der Mengenbefund für die Exchange-Mitglieder kann nach Gleichheitsprüfung erhalten bleiben. Der Befund zu ihren Rollenbelegungen muss erneuert werden. Abdeckung bedeutet hier sowohl „beide Exchanges betrachtet“ als auch „ihre aktuellen Belegungen beurteilt“. Nur die erste Aussage ist unverändert.
5. **Andere Anwendung erhalten und Kontexte erneuern.** „api für E1“ bleibt nach Aktualitätsprüfung gültig. Der API-Kontext verliert ausschließlich die Anwendung für E2; der onprem-Kontext erhält sie. Historische M1-Ergebnisse werden nicht überschrieben. Referenzen auf api als Teilnehmer anderer Vorgänge bleiben erhalten.
6. **Realisierungsfolgen getrennt untersuchen.** Die bisherige Auftraggeberrealisierung für E2 auf API-Seite muss gegebenenfalls entfallen oder umgestellt werden; auf onprem-Seite sind Auftragserzeugung, monotone Zeitmessung und Antwortzuordnung zu realisieren und zu prüfen. Auch receiver, Kanalberechtigungen und Integration können betroffen sein. Aus der Änderung der Modellgeltung folgt noch keine vollständige Dateiliste und kein bestandener Test.

Der erwartete Index nach jeweils erfolgreicher Beurteilung lautet für diese eine Pflicht:

| Stand | Exchange-Auswahl | Anwendung auf api | Anwendung auf onprem |
| --- | --- | --- | --- |
| M0 | E1 | Für E1. | Keine Anwendung aus dieser Pflicht im betrachteten Stand. |
| M1 | E1, E2 | Für E1 und separat für E2. | Keine Anwendung aus dieser Pflicht im betrachteten Stand. |
| M2 | E1, E2 | Weiter für E1. | Neu für E2. |

Die Wiederverwendung ergibt sich damit nicht nur aus der Gleichheit von Ziel-IDs:

| Teil des Index | M0 → M1 | M1 → M2 |
| --- | --- | --- |
| Originalaussage und allgemeine Auslegung | Nach Aktualitäts- und Gegenprüfungen wiederverwendbar. | Unter denselben Voraussetzungen wiederverwendbar. |
| Anwendung auf api für E1 | Nach Prüfung möglicher neuer Einflüsse übernehmbar. | Nach Prüfung möglicher Auswirkungen des Rollenwechsels übernehmbar. |
| Anwendung für E2 | Neu erschließen und prüfen. | Neue Rollenbelegung prüfen; bisherige Anwendung nur historisch behalten. |
| Vollständigkeit der Exchange-Auswahl | Neue Menge ermitteln und Abdeckung erneuern. | Gleiche Menge strukturell bestätigen; aktuelle Rollenabdeckung erneuern. |
| Konkrete Teilnehmer-/Channelprüfung für E2 | Neu. | Nicht pauschal wiederverwendbar; neuer Auftraggeber verändert relevante Voraussetzungen. |
| Nachweis der Softwarekonformität | Für E2 noch zu führen. | Zum neuen Umsetzungskandidaten erneut zu führen; Modellreview ersetzt ihn nicht. |

Dieser Durchgang schlägt außerdem eine wichtige Berichtseigenschaft vor: Auch die Übernahme eines alten Befunds ist ein Ergebnis mit Grund und neuer Standbindung. Ein vorheriger Status „bestätigt“ wird nicht ohne Aktualitätsprüfung auf M1 oder M2 kopiert. Weder ein offener neuer Fall noch seine Priorisierung als Kontext darf als erledigte Prüfung erscheinen. Die konkrete Automatisierung dieser Schritte ist noch zu entwerfen und zu erproben.

### 11.11 Wiederverwendbarkeit: gelesene Inhalte, Auswahlräume und neue Einflüsse

Der weitere Nutzerauftrag konkretisiert die Abhängigkeiten, mit denen ein bestehender Indexeintrag auf Wiederverwendbarkeit geprüft wird. Der folgende Vorschlag ergänzt den Durchgang, ohne eine neue Core-Sprache für Regeln oder Geltungsprädikate festzulegen. Die Erfassung von Abhängigkeiten ist Teil der Indexverarbeitung. Eine Liste der beim ersten Lesen besuchten IDs reicht dafür nicht: Ein bisher unbekannter Satz kann eine globale Festlegung beanspruchen, eine andere Aussage einschränken oder eine neue Pflicht hinzufügen.

**Zwei getrennte Grundlagen.** Jeder Eintrag hängt einerseits von seinen konkreten Quellen und Auslegungsschritten ab. Andererseits hängt seine Verwendbarkeit im aktuellen Kontext davon ab, dass Änderungen im gesamten gebundenen Modellumfang auf mögliche zusätzliche oder entgegenstehende Aussagen untersucht wurden. Der Inhalt eines alten Befunds kann weiterhin stimmen, während der daraus erzeugte Kontext noch unvollständig ist. „Unveränderte Grundlage“ und „keine ungeklärten neuen Einflüsse“ sind deshalb getrennte Prüfungen.

Zu erfassen sind mindestens diese Abhängigkeiten:

| Abhängigkeit | Zu bindender Inhalt | Welche Änderung damit sichtbar wird |
| --- | --- | --- |
| Quellaussage und Bedeutungsumfeld | Kanonische Identität, vollständiger maßgeblicher body, Fundstelle, verwendete Kind-/Propertyverträge und weiterer interpretierter Kontext. Für den Einstieg werden eher ganze Definitionen als nur einzelne Satzfragmente gebunden. | Neue Ausnahme neben einem unveränderten Satz; geänderte Bedeutung der Auftraggeber-Property; weggefallene Begründung. |
| Konkrete Struktur und Werte | Verwendete Typisierung, stabile Property-IDs mit aktuellem Feldnamen, Referenzziele, Bedingungen und relevante Werte beziehungsweise deren Fehlen. Nicht ausreichend abgegrenzte Ausschnitte werden konservativ als vollständiger Definitionsinhalt gebunden. | requester-Wechsel, veränderte Bedingungswerte, neu gesetzte optionale Angaben oder gelöschte Beziehungen. |
| Auswahlraum und Auswahlverfahren | Fester betrachteter Modellumfang, geprüfte Bedeutung der Auswahl, verwendete Operationen, Mitglieder des Ausgangsraums und alle für die Auswahl nötigen Eingaben. | Neue Exchange-Definition oder ein bisher ausgeschlossener Gegenstand, der durch eine Änderung nun passt. |
| Negative Voraussetzungen und Ausschlüsse | Welcher Bereich auf welche Einschränkung geprüft wurde, zugehörige Quellen, geprüfte Ausschlussgründe und offene Fälle. Auch ein leeres Ergebnis benötigt eine Grundlage. | Eine neue mögliche Ausnahme oder die Löschung eines bisherigen Ausschlussgrunds. |
| Andere abgeleitete Befunde | Die genauen Fassungen verwendeter Auslegungen, ihrer Prüfungen und Abdeckung sowie Voraussetzungen für deren Wiederverwendung. | Eine beanstandete Rolleninterpretation oder ein ungültig gewordener Abdeckungsbefund wirkt auf seine abhängigen Anwendungen. |
| Modellbindung und Verarbeitungsvertrag | Paket-/Inhaltsbindungen, vollständiges Eingabeverzeichnis, Auflösungs-/Normalisierungsregeln und verwendete Index-/Prüfverfahren. | Neu eingebundener Bibliotheksinhalt, geänderte Auflösung oder eine Änderung des Verfahrens, auf dessen Ergebnis man sich stützt. |

IDs und Digests belegen Identität und Inhaltsgleichheit; sie belegen weder die ausreichende Auswahl der Grundlagen noch deren richtige Auslegung. Eine Inhaltsbindung muss den tatsächlich geprüften Umfang treffen. Ein geänderter Gesamtmodell-Digest allein sagt noch nicht, welcher Befund fachlich neu auszulegen ist, stellt aber sicher, dass der alte Stand nicht einfach als aktueller ausgegeben wird.

**Auswahlabhängigkeiten gehen über bisherige Treffer hinaus.** Für „alle Exchanges“ wird die Mitgliedschaft im vollständigen ausgewählten Definitionsraum überwacht, nicht nur die Liste bereits bewerteter Anwendungen. Bei einer bedingten Auswahl wie „alle Exchanges mit requester am Standort cloud“ gehören auch die requester-Bezüge und Standortwerte der bisher nicht passenden Kandidaten zur Auswahlgrundlage. Sonst könnte ein Standortwechsel unbemerkt einen neuen Treffer erzeugen. Für ein geprüft ausführbares Auswahlverfahren können diese Abhängigkeiten technisch vorgegeben werden; für freie semantische Auswahlen bleibt die vollständige Reichweite selbst prüfbedürftig. Ein Reverse-Index auf gelesene Treffer allein löst dieses Problem nicht.

**Neue globale Aussagen benötigen eine vollständige Änderungsprüfung.** Für einen Übergang zwischen zwei festen Ständen wird unabhängig von vorhandenen Eintragsabhängigkeiten eine vollständige Änderungsmenge aus dem gesamten gebundenen Eingabeumfang ermittelt. Dazu gehören Hinzufügungen, Änderungen und Löschungen von Definitionen und importierten Inhalten sowie geänderte Typen, Werte und aufgelöste Beziehungen. Eine unveränderte Prosa kann durch eine andere Belegung oder Bindung eine andere Reichweite erhalten. Daher werden nicht ausschließlich bearbeitete body-Zeichenfolgen betrachtet.

Diese Änderungsprüfung hält für jeden Änderungsbereich fest: Untersuchung noch offen, mögliche Wirkung auf benannte Bereiche oder begründet keine zusätzliche Wirkung auf den betrachteten Befund. Ihre Grundlage enthält den vorherigen und neuen Inhalt und den zum Verständnis nötigen Kontext. Bei semantischen Reichweitenfeststellungen ist unabhängige Prüfung erforderlich. Die Änderungsmenge vollständig abzuarbeiten ist strukturell kontrollierbar; ihre Bedeutung zuverlässig einzugrenzen bleibt eine semantische Aufgabe.

Insbesondere gilt:

- Eine neue oder geänderte Aussage wird nicht allein wegen fehlender Links, eines lokalen Dateipfads, ihres Kinds oder fehlender Schlüsselwörter als lokal behandelt. Im gewählten Core darf ihr body einen weiteren Geltungsbereich beschreiben.
- Bei Löschungen wird die frühere Aussage berücksichtigt: Eine weggefallene Einschränkung kann bestehende Pflichten erweitern. Bei einer geänderten Aussage zählen frühere und neue Reichweite.
- Eine Aussage wie „keine relevante Ausnahme gefunden“ ist ein geprüfter Befund gegen einen bestimmten Umfang. Sie wird nicht als strukturell bewiesene Abwesenheit von beliebiger semantischer Ausnahme gespeichert. Ein neu hinzugekommener Text ist zunächst ungeprüft und kein weiterer negativer Treffer.
- Eine bereits ausgeschlossene Quelle wird bei maßgeblicher Änderung erneut betrachtet. Neue Definitionen und neue Importinhalte dürfen nicht dadurch ausgeschlossen werden, dass sie im alten Auswahlraum fehlen.
- Ein unklarer möglicher Einfluss hält die betroffenen Befunde für den neuen Stand offen. Kann sein Bereich nicht begründet eingegrenzt werden, ist der größere gebundene Modellumfang zu beurteilen. Es gibt keine automatische Entlastung durch niedrige Priorität oder ein Kontextbudget.

Das vollständige Eingabeverzeichnis und seine Bindung bilden dafür eine konservative äußere Abhängigkeit. Jede Modelländerung verlangt einen aktuellen Befund zur möglichen Reichweite ihrer Änderungen. Im Einstieg wird dieser Bereich nicht aufgrund von automatisch vergebenen Themenetiketten partitioniert. Ein Wortfilter oder eine Vektorsuche kann Kandidaten liefern, aber keine neue Aussage von der Änderungsprüfung ausnehmen. Ein vom gleichen alten Index behauptetes „hier kann nichts Relevantes stehen“ darf nicht selbst die einzige Grundlage dafür sein, neue Texte ungelesen auszuschließen.

**Ablauf für einen Wiederverwendungsversuch.** Markitect geht für den betrachteten Eintrag in dieser Reihenfolge vor:

1. Neuen Modellstand kompilieren und gegenüber dem gebundenen Ausgangsstand die vollständigen Änderungen feststellen. Bei ungültiger Struktur entsteht kein gültiger neuer Indexstand.
2. Direkte Grundlagen, vorausgesetzte Auslegungen und Auswahlräume vergleichen beziehungsweise neu auswerten. Ist etwa requester geändert, wird die konkrete Anwendung neu gebildet. Eine unveränderte Auswahlmenge beweist noch keine unveränderten Rollenbelegungen.
3. Die Änderungen im gebundenen Modellumfang auf neue, geänderte oder weggefallene Einflüsse untersuchen. Ein alter globaler Prüfstand allein genügt dafür nicht. Bleiben hierbei relevante Fragen offen, darf der Eintrag noch nicht als vollständig aktueller Kontext verwendet werden.
4. Mögliche neue Einflüsse zuordnen und betroffene Auslegungen, Anwendungen oder Abdeckung erneut beurteilen. Eine zusätzliche unabhängige Pflicht kann einen neuen Eintrag erfordern, ohne die alte Zeitmessungsauslegung zu ändern. Eine beanspruchte Ausnahme kann dagegen einen Konflikt erzeugen; sie setzt bestehende Pflichten nicht still außer Kraft.
5. Nur ausreichend belegte Wiederverwendung an den neuen Stand binden. Der Übernahmebefund nennt unveränderte beziehungsweise neu bestätigte Grundlagen, aktuelle Auswahl-/Abdeckungsbefunde und die abgeschlossene Prüfung neuer Einflüsse. Historische Ergebnisse bleiben erhalten.

Das Ergebnis lautet entweder „für diesen Stand wiederverwendbar“, „aufgrund benannter Änderungen neu abzuleiten“ oder „Aktualität noch offen“. Die ursprünglichen Prüfbelege bleiben von diesem aktuellen Übernahmebefund unterscheidbar. Auch ein wiederverwendeter Eintrag kann zusätzliche neue Kontextnachbarn bekommen. Ein zusammengesetzter Kontext gilt erst dann als aktuell, wenn seine benötigten Einträge und die Prüfung weiterer Einflüsse zum selben Zielstand passen. Noch offene Arbeit wird nicht durch eine Mischung alter und neuer positiver Zustände verdeckt.

Für die Zeitmessungspflicht aus dem bisherigen M2 ergeben sich diese Gegenproben:

| Neue Änderung | Auslösende Abhängigkeit | Folge |
| --- | --- | --- |
| In einer bisher unbeteiligten Definition steht nun „Alle Auftraggeber müssen zusätzlich die verwendete Uhrquelle protokollieren.“ | Vollständige Änderungsprüfung, auch ohne Verweis auf Exchange oder api. | Geltungsbereich prüfen und gegebenenfalls neue Anwendungen ergänzen. Der alte Zeitmessungsbefund kann bestehen bleiben; der frühere Kontext ist nicht allein deshalb vollständig. |
| Eine andere Definition beansprucht „Für Statusabfragen genügt die Wanduhr.“ | Prüfung neuer Einflüsse sowie bisher geprüfte negative Voraussetzung zu Ausnahmen. | Möglichen Konflikt mit dem Exchange-Vertrag für check-status beurteilen. Keine automatische Ausnahme nach dem Prinzip „spezifischer gewinnt“. |
| Ein dritter Exchange entsteht | Auswahlraum aller Exchanges und neue Aussagen im Modellumfang. | Neue Anwendung und Abdeckung prüfen; vorhandene Auslegung kann bei unveränderten Grundlagen erhalten bleiben. |
| check-status.requester wechselt wieder von onprem zu api | Konkrete Rollenbelegung und davon abhängige Auswahl-/Anwendungsbefunde. | Beide Ziele berücksichtigen und die Anwendung erneuern; list-documents bleibt ein separater Fall. |
| Eine bereits erfasste Einschränkung wird gelöscht | Alte Quellfassung, Voraussetzungen und vollständiger Löschungsbefund. | Bisher ausgeschlossene Fälle erneut beurteilen. Ein nur aus dem neuen Graph abgeleiteter Blick wäre unzureichend. |

Der Vorschlag stellt damit sicher, dass neue Eingaben nicht allein wegen einer zu engen Cache-Abhängigkeit ungeprüft bleiben. Er verspricht nicht, dass ein Agent jede semantische Verbindung fehlerfrei erkennt. Anfangs sind die größeren Inhaltsbindungen und die vollständige Änderungsprüfung vorzuziehen; feinere Wiederverwendung darf erst auf geprüft ausreichender Abhängigkeits- und Abdeckungserfassung beruhen. Weder ein genauer Datenvertrag noch ein ausführbarer Wiederverwendungsprüfer sind mit dieser Designgegenprobe bereits implementiert.

### 11.12 Durchgang: globale Zusatzpflicht und ausdrückliche Ausnahme

Der nächste Nutzerauftrag spielt eine neue globale Pflicht und anschließend eine Ausnahme für check-status durch. Ausgangspunkt ist M2 aus Abschnitt 11.10: list-documents hat requester api, check-status hat requester onprem. Die betrachteten Definitionen, Rollenbelegungen und Paketbindungen bleiben danach gleich. Alle folgenden Bestätigungen beschreiben erwartete Ergebnisse nach erfolgreicher unabhängiger Prüfung im Gedankenbeispiel; eine solche semantische Prüfung wurde hier nicht ausgeführt.

Für die Fallabgrenzung betrifft die neue Pflicht die zusätzliche Protokollierung der Uhrquelle. Die spätere Ausnahme begrenzt ausschließlich diese neue Pflicht. Sie hebt die bisherige Pflicht zur monotonen Zeitmessung nicht auf. Die Quellaussage wird im Beispiel im body der vorhandenen Projektdefinition platform ergänzt. Ihre globale Reichweite entsteht aus ihrem Satz, nicht aus dem Kind System, ihrer Ablage oder einer automatischen Enthaltenheitsregel.

Als lesbare Kürzel für Indexbefunde verwenden wir T für die allgemeine Auslegung der Zeitmessungspflicht, T1 für ihre Anwendung auf api bei list-documents und T2 für ihre Anwendung auf onprem bei check-status. Diese Kürzel sind keine zusätzlichen kanonischen Definitionen oder zugesagten dauerhaften Satz-IDs.

**M2 → M3: eine globale Pflicht ergänzen.** Die neue Aussage lautet:

> Zusätzlich muss für jeden Exchange die durch requester bestimmte Anwendung bei jedem Versuch die verwendete Uhrquelle protokollieren.

Der Compiler kann den geänderten body und seinen gebundenen neuen Inhalt erfassen. Der Satz benötigt keine Links, um verbindlich zu sein. Dass er beide Exchanges und deren Auftraggeber meint, ist semantisch zu erschließen und unabhängig zu prüfen.

1. **Aktualität öffnen:** Der alte Befund zur vollständigen Erschließung möglicher neuer Einflüsse passt nicht auf M3. Bisherige Kontexte dürfen bis zur Untersuchung der Ergänzung nicht als vollständig aktuell gelten. T, T1 und T2 sind deshalb nicht automatisch fachlich falsch; ihre Verwendung am neuen Stand benötigt eine aktuelle Prüfung. Falls platform selbst zu ihren konkret gebundenen Bedeutungsgrundlagen gehörte, ist außerdem diese Inhaltsabhängigkeit geändert.
2. **Neue Auslegung bilden:** Es entsteht L3: Die Auftraggeber aller Exchanges protokollieren pro Versuch die verwendete Uhrquelle. Zu prüfen sind insbesondere Rollenbezug, Pflichtcharakter, Auswahlumfang, Verhältnis zur monotonen Zeitmessung und mögliche andere Einschränkungen. Die Pflicht gilt je Austausch und Versuch, nicht beliebig für jeden Vorgang der referenzierten Anwendung.
3. **Neue Anwendungen prüfen:** L3-1 betrifft api für list-documents; L3-2 betrifft onprem für check-status. Die Typ-/Propertyzuordnungen und Referenzwerte können aus aktuellen strukturellen Befunden übernommen werden. Die Anwendung dieser neuen Pflicht und ihre Vereinbarkeit mit den konkreten Verträgen sind neu zu beurteilen.
4. **Bisherige Pflicht bestätigen:** Nach Prüfung des Verhältnisses beider Aussagen können T, T1 und T2 für M3 übernommen beziehungsweise erneut bestätigt werden. Die neue Pflicht ergänzt die Zeitmessung; sie setzt sie nicht außer Kraft. Ein ganzer alter Definitionshash darf nicht als unverändert ausgegeben werden, wenn der betrachtete Bedeutungsinhalt tatsächlich geändert ist.
5. **Abdeckung und Kontexte erneuern:** Die Exchange-Grundmenge bleibt {list-documents, check-status}; ihre unveränderte Mitgliedschaft kann strukturell bestätigt werden. Die neue Protokollierungspflicht, beide Anwendungen und ihre möglichen Einschränkungen waren bisher nicht abgedeckt. Erst ihre Prüfung begründet einen neuen vollständigen Befund für diesen betrachteten Bereich. API- und onprem-Kontext erhalten jeweils eine zusätzliche Pflicht.

Nach erfolgreicher Beurteilung enthält der betrachtete Ausschnitt des M3-Index T, T1, T2 und zusätzlich L3, L3-1, L3-2 samt neuer Abdeckung. Historische M2-Ergebnisse bleiben unverändert. Die Softwarekonformität hinsichtlich der zusätzlichen Protokollierung ist damit noch nicht nachgewiesen.

**M3 → M4: die globale Aussage ausdrücklich begrenzen.** Für den widerspruchsfreien Hauptfall wird die zuvor ergänzte Aussage im selben Vertrag ersetzt durch:

> Zusätzlich muss für jeden Exchange außer [[check-status]] die durch requester bestimmte Anwendung bei jedem Versuch die verwendete Uhrquelle protokollieren.

Das ausdrückliche Ziel des Körperlinks wird im Projektpaket aufgelöst. Die Ausnahme entsteht aus dem Satz; der Link allein kennt keine Ausnahmeoperation. Die neue Quellfassung bleibt unter derselben Definitionsidentität platform, besitzt aber einen anderen body und damit eine andere gebundene Inhaltsgrundlage.

1. **Geänderte Quelle und abhängige Befunde markieren:** L3 ist keine aktuelle Auslegung der neuen Quellfassung. Seine Anwendungen L3-1 und L3-2 sowie der Befund „diese Pflicht gilt für beide Exchanges“ sind deshalb nicht unverändert nach M4 übernehmbar. Ein früherer Befund „keine einschlägige Ausnahme gefunden“ für die Protokollierung ist ebenfalls nicht mehr aktuell. Sie bleiben gültige historische Ergebnisse zu M3.
2. **Neue Auslegung prüfen:** L4 lautet nun „Protokollierung für alle Exchanges außer dem ausdrücklich bezeichneten check-status“. Der Reviewer prüft Quellvergleich, eindeutiges Ausnahmeziel, begrenzten Umfang der Ausnahme und Widersprüche im weiteren Kontext. Insbesondere muss klar bleiben, dass die Zeitmessungspflicht davon unberührt ist. Eine passende Ausnahme darf weder aus einem ähnlich benannten Status-Command erraten noch auf sämtliche Statusabfragen ausgedehnt werden.
3. **Beide bisherigen Anwendungen erneut einordnen:** Für list-documents entsteht beziehungsweise bestätigt sich der aktuelle positive Befund L4-1 auf api. Die unveränderte Rollenbelegung und strukturelle Zuordnung können wiederverwendet werden; die Anwendung unter dem geänderten Vertrag benötigt einen neuen Geltungs-/Übernahmebefund. Für check-status gilt L3-2 in M4 nicht mehr. Die Abdeckung hält stattdessen ausdrücklich fest: aus L4 keine Protokollierungspflicht für onprem in diesem Exchange, aufgrund der geprüften Ausnahme.
4. **Mengen und negative Ergebnisse unterscheiden:** Die betrachtete Exchange-Grundmenge hat weiter zwei Mitglieder. Die positive Auswahl der Protokollierungspflicht schrumpft von zwei auf eines. check-status verschwindet nicht aus der Abdeckung, sondern bleibt als geprüfter ausgeschlossener Fall enthalten. Das verhindert, dass ein fehlender positiver Eintrag mit einer übersehenen Prüfung verwechselt wird.
5. **Zeitmessung und weitere Pflichten auf Wechselwirkungen prüfen:** T1 und T2 bleiben nach passender Aktualitäts-/Verträglichkeitsprüfung bestehen. Eine Ausnahme von L4 ist kein Beleg für eine Ausnahme von T oder einer anderen Protokollierungsvorgabe. Reicht die Formulierung über die betrachtete Pflicht hinaus oder ist sie mehrdeutig, bleibt der weitere Bereich offen.
6. **Aktuelle Kontexte und Realisierungsfolgen bilden:** Die API bleibt für list-documents zur Zeitmessung und zur Protokollierung verpflichtet. onprem bleibt für check-status zur monotonen Zeitmessung verpflichtet; die zusätzliche Protokollierung ist durch diese Aussage nicht mehr verlangt. Ihr Wegfall ist kein Verbot, weiterhin zu protokollieren. Umsetzung und Tests müssen gegen die neue Pflichtmenge beurteilt werden, statt vorhandene Logs oder Tests automatisch zu entfernen.

Die erwarteten aktuellen Geltungsbefunde nach jeweils erfolgreicher Prüfung lauten:

| Pflicht und Anwendung | M2 | M3 | M4 |
| --- | --- | --- | --- |
| Zeitmessung: api für list-documents | Gilt. | Nach Aktualitätsprüfung weiter gültig. | Nach Aktualitätsprüfung weiter gültig. |
| Zeitmessung: onprem für check-status | Gilt. | Nach Aktualitätsprüfung weiter gültig. | Nach Prüfung der begrenzten Ausnahme weiter gültig. |
| Zusätzliche Protokollierung: api für list-documents | Noch nicht aus dieser Quelle verlangt. | Neuer positiver Befund L3-1. | Neuer beziehungsweise erneut bestätigter positiver Befund L4-1. |
| Zusätzliche Protokollierung: onprem für check-status | Noch nicht aus dieser Quelle verlangt. | Neuer positiver Befund L3-2. | Alter positiver Befund nicht mehr aktuell; ausdrücklich geprüfter Ausschluss aus L4. |

Für „ungültig“ muss der Bericht daher unterscheiden: Quelle oder Herleitung sind für den neuen Stand überholt; ein konkreter Geltungsanspruch trifft nicht mehr zu; oder lediglich die Aktualitäts-/Vollständigkeitsprüfung ist noch offen. Diese verschiedenen Befunde verlangen unterschiedliche Arbeit. Eine globale Ergänzung macht nicht jede bestehende Einzelpflicht falsch; eine geänderte Quellauslegung lässt sich aber auch nicht ohne erneuten Übernahmegrund an ihre alten Anwendungen hängen.

**Alternativfall: eine lokale Ausnahme behaupten, ohne die globale Pflicht anzupassen.** Stünde in check-status.body lediglich „Diese Protokollierung ist hier nicht erforderlich“, während die unbeschränkte globale Pflicht bestehen bleibt, wäre ihre Verträglichkeit offen beziehungsweise widersprüchlich. Es gibt weiterhin keine Core-Regel „lokal oder spezifisch gewinnt“. Der Index muss die Gegenbehauptung und den betroffenen Geltungsbefund für check-status zur Klärung markieren, statt L3-2 still zu entfernen oder eine bestätigte Befreiung zu behaupten. list-documents kann nur nach begründeter Eingrenzung des Konflikts als unberührt bestätigt werden. Ob die beabsichtigte Ausnahme durch Präzisierung der globalen Aussage oder eine andere ausdrücklich vereinbarte Regel aufgelöst wird, ist eine Modellentscheidung; der Index darf sie nicht selbst erfinden.

### 11.13 Vorschlag eines Datenvertrags für den semantischen Index

Der Nutzer beauftragt nun einen konkreten Datenvertrag aus den Durchgängen. Der folgende Vertrag beschreibt abgeleitete Indexdaten, nicht weitere Definitionen oder Meta-Kinds der kanonischen Sprache. Er ist ein Entwurf des Datenmodells; seine Serialisierung, Digestkodierung und produktive Implementierung sind nicht bereits festgelegt. Fachliche Geltung, unabhängige Prüfung und Aktualität bleiben getrennte Angaben.

#### Hülle, Identitäten und Modellbindung

Ein Indexpaket enthält formatVersion, targetSnapshot, procedures, records, reviews, bindings und requiredCoverage. formatVersion ist im Vorschlag semantic-index/v1. Die drei Sammlungen records, reviews und bindings sind Listen mit eindeutigen Inhalts-IDs; sie enthalten oder referenzieren unveränderliche Datensätze. requiredCoverage ist eine Liste genauer BindingRefs. Alle genannten Felder sind erforderlich; Sammlungen können leer sein, ergeben dann aber gegebenenfalls nur einen unvollständigen Index. Jeder fremde Datensatz und jedes Inhaltsmanifest muss mit seiner genauen Fassung verfügbar sein; ein Digest ohne auflösbaren Inhalt genügt nicht zur Nachprüfung. procedures bezeichnet die genaue Procedures-Struktur oder eine auflösbare Inhaltsreferenz auf sie; dieselbe Regel gilt in Einträgen und Prüfdatensätzen.

| Typ | Erforderliche Felder und Bedeutung |
| --- | --- |
| ModelRef | package und id, beide nichtleere Strings. Bezeichnet eine kanonische Definition ohne Alias oder eingebaute Versionsnummer. |
| SnapshotRef | modelDigest, packageBindingsDigest, inventoryDigest und compilerContractDigest. Die referenzierten Manifeste binden alle geladenen Definitionen, Paketfassungen/Inhalte und die verwendete Auflösung/Normalisierung. |
| Procedures | indexContractDigest, derivationProcedureDigest und reviewPolicyDigest. Binden Daten-/Ableitungsregeln und Anforderungen an die Prüfung; ein Modellimport installiert diese Verfahren nicht. |
| RecordRef, ReviewRef, BindingRef | Jeweils die exakte unveränderliche Datensatzfassung über ihre Inhalts-ID. Eine lesbare Bezeichnung kann angezeigt werden, ersetzt diese Identität aber nicht. |
| InventoryRef | scope, snapshot und digest. Der auflösbare Inhalt enthält die vollständigen eindeutig identifizierten Elemente dieses Umfangs, einschließlich Definitionen, Kandidaten oder Änderungen und ihrer nötigen Werte. |

targetSnapshot bezeichnet den Stand, für den die Hülle zusammengestellt wird. Das gesamte gebundene Modell ist der feste äußere Umfang bound-model; kleinere Auswahlräume müssen auf ihn zurückführen. Bei einer Importänderung gehören auch die dadurch geänderten geladenen Inhalte zur Bestandsdifferenz. Snapshot- und Verfahrensreferenzen sind exakt zu vergleichen; abweichende Fassungen verlangen eine Aktualitätsprüfung und gegebenenfalls ausdrücklich nachgewiesene Kompatibilität, keine Annahme anhand ähnlicher Versionsnamen.

Datensatz-IDs sind technische Inhaltsidentitäten des Index. Sie sind keine stabilen fachlichen IDs des kanonischen Modells. Bei ihrer Inhaltsberechnung wird das eigene ID-Feld ausgelassen, damit kein Selbsthash verlangt wird. Eine Änderung an Behauptung, Abhängigkeiten oder Herkunft erzeugt eine neue Datensatzfassung. Optionales replaces verweist ausschließlich auf die historische Vorgängerfassung; es ist keine fortgeltende fachliche Voraussetzung und keine Berechtigung zur Übernahme ihrer Reviews.

#### Gemeinsame Eintragsform und vier Eintragsarten

Jeder Eintrag besitzt die erforderlichen Felder recordId, recordType, originSnapshot, procedures, producer, sources, dependencies und claim. producer enthält runId und die genaue erzeugende Verfahrensfassung. sources und dependencies sind Listen; für Auslegungen ist mindestens eine Quellstelle erforderlich, für Anwendungen mindestens die geprüfte Auslegung und die konkrete Fallgrundlage. In anderen Fällen muss die erforderliche Grundlage durch Quell-, Inventar- oder Vorbefundreferenzen vorhanden sein. Unbekannte Felder oder recordType-Werte sind Fehler; optionale Felder werden ausgelassen, statt mit null belegt.

sources enthalten jeweils definition als ModelRef, definitionDigest, pointer, startByte, endByte und quote. Ein Pointer verwendet die JSON-Pointer-Schreibweise; für Prosa ist pointer zunächst /body. Der Bereich [startByte, endByte) zählt UTF-8-Bytes im dekodierten Feldwert, ohne Unicode-Normalisierung, nicht Positionen in der YAML-Datei. Die Grenzen müssen auf Zeichenränder fallen und der Ausschnitt muss exakt quote ergeben. Die Quelldatei-/Zeilenabbildung ist zusätzliche Herkunft. Der vollständige gebundene Vertrag bleibt verfügbar; eine korrekte Quellstelle beweist keine ausreichende Kontextauswahl.

| recordType | Erforderlicher claim-Inhalt |
| --- | --- |
| interpretation | meaning als verbindlichkeitsgetreue Auslegung in Prosa, force als obligation, permission, prohibition, information oder unresolved, sowie scope mit description und inventory. Die Auswahl muss zur konkreten Fassung eines betrachteten Umfangs passen. |
| application | interpretation als RecordRef, subject als ModelRef, context als nichtleere Liste von ModelRefs, result als applies, excluded, uncertain oder conflict, statement und reason als nichtleere Texte. Optional property als ModelRef bezeichnet die betrachtete Rolle. |
| coverage | purpose als sources, cases oder changes, inventory als InventoryRef und results als Liste der unten definierten Einzelergebnisse. Ein optionales baseSnapshot ist bei changes erforderlich. |
| influence | baseSnapshot, targetSnapshot, changes als InventoryRef, considered als Liste genauer RecordRefs oder als bound-model, result als affects, no-additional-influence oder uncertain, affected als Liste von ModelRefs/RecordRefs oder bound-model, sowie reason. Das Ergebnis bezieht sich auf den angegebenen Vergleich und betrachteten Bereich. |

scope.inventory einer Auslegung bindet den bei ihrer Erstellung betrachteten Ausgangsraum, nicht eine für immer abgeschlossene Trefferliste. Die allgemeine Bedeutung „alle Exchanges“ kann unverändert bleiben, während ihre aktuelle Fallmenge neu ausgewertet wird. Nur diese mengenbezogene Beobachtung darf eine erfolgreiche Neuauswertung ersetzen; Aussagen und sonstige direkte Bedeutungsgrundlagen bleiben unverändert zu prüfen.

Ein application-Eintrag mit subject onprem, context [check-status], property Exchange/sender und result excluded besagt ausschließlich: Die referenzierte Auslegung erzeugt für diesen Teilnehmer in diesem Austausch keine solche Pflicht. Er enthält weder eine globale Befreiung von onprem noch ein Verbot der beschriebenen Handlung. uncertain und conflict sind tatsächliche Ergebnisse mit Klärungsbedarf und keine negativen Geltungsnachweise. force bezeichnet eine abgeleitete Interpretation, kein neues Sprachmittel für Anforderungen im Modell.

#### Abhängigkeiten als geschlossene Varianten

Jede Abhängigkeit hat eine im Eintrag eindeutige key und einen type. Die übrigen erforderlichen Felder hängen vom type ab. Für den Einstieg gelten vollständige normalisierte Definitionen als Inhaltsgranularität; beliebige agentisch ausgesuchte Feldfragmente sind keine belastbare automatische Optimierung.

| type | Erforderliche weitere Felder | Gültigkeitsbedingung bei Wiederverwendung |
| --- | --- | --- |
| content | definition als ModelRef, expectedDigest | Die vollständige Definition hat dieselbe normalisierte Inhaltsfassung. Das erfasst auch Werte und deren Fehlen. |
| record | record als RecordRef | Der genaue vorausgesetzte Befund ist selbst für den Zielstand verwendbar. Eine neue Auslegung mit gleicher Formulierung ist nicht automatisch dieselbe Voraussetzung. |
| selection | basis als RecordRef der geprüften Auswahlbedeutung, input als InventoryRef, result als InventoryRef, evaluation als geprüfte Verfahrensreferenz | Die Auswahl wird gegen den vollständigen aktuellen Eingaberaum neu ausgewertet. Die relevante beobachtete Ergebnismenge bleibt gleich. Veränderte Kandidaten-/Belegungsabdeckung ist trotzdem neu zu prüfen. |
| absence | statement, scope als InventoryRef, coverage als RecordRef | Die genau bezeichnete negative Voraussetzung wird für den aktuellen Umfang erneut bestätigt; ein vormals leeres Ergebnis wird nicht einfach übernommen. |
| watch | scope mit Wert bound-model | Eine aktuelle vollständige Prüfung der neuen Einflüsse ist für den Eintrag vorhanden. Ungleiche Modellinventare verlangen diese Prüfung, nicht zwingend eine neue Auslegung des Eintrags. |

evaluation einer selection darf nur ein festgelegtes geprüftes Verfahren bezeichnen. Kann die Auswahl nicht strukturell ausgeführt werden, muss die Neuauswertung ein entsprechend geprüftes semantisches Ergebnis liefern. Ein frei erzeugtes Programm oder eine Beschreibung wird nicht als vertrauenswürdige Abfrage ausgeführt. input enthält auch Nichttreffer und relevante Auswahlwerte; result enthält nicht nur eine Trefferzahl, sondern die vollständigen beobachteten Zuordnungen. Bei Anwendungen gehören requester-Zuordnungen zur Beobachtung, nicht nur Exchange-IDs.

Jeder fachlich für den aktuellen Kontext zu verwendende Eintrag wird durch watch oder die gleichwertige verpflichtende Prüfung in seiner Binding erfasst. Diese Prüfung ist niemals optional, nur weil dependencies keine neue Quelle enthält. Wenn sich eine content-Grundlage oder eine genaue vorausgesetzte record-Fassung ändert, kann der alte Eintrag nicht als unverändert übernommen werden. Auch bei gleicher Schlussfolgerung entsteht dann ein neuer Befund mit aktuellen Grundlagen. Eine erfolgreiche erneute Auswahl- oder Abwesenheitsprüfung darf dagegen die dafür ausdrücklich vorgesehene Beobachtung bestätigen; das ist ein protokollierter Prüfschritt, kein Vergleich allein der alten Digests.

#### Abdeckung und vollständige Änderungserfassung

Ein coverage.results-Element enthält itemId aus seinem Inventar, outcome, reason und records als Liste präziser Befundreferenzen. Für purpose sources sind die outcomes interpreted, no-semantic-claim oder unresolved; für cases assessed, excluded oder unresolved; für changes assessed oder unresolved. reason ist auch bei Ausschlüssen erforderlich. assessed oder interpreted benötigen passende zugehörige Befunde. Ein negativer no-semantic-claim-Befund muss semantisch begründet und geprüft sein. Es gibt genau ein Ergebnis je Inventarelement; fehlende Ergebnisse werden als offen behandelt, fremde oder doppelte itemIds sind Fehler.

Für sources muss das verpflichtende äußere Inventar sämtliche geladenen Definitions-bodies und ihre Bedeutungsgrundlagen erfassen. Für cases muss die Eingabemenge samt relevanten Belegungen eindeutig feststehen. Für changes wird das vollständige Inventar aus dem Vergleich zweier fester Modellstände erzeugt. Entfernte Einträge erhalten ebenfalls eine itemId und Vorherherkunft. Weder Suchtreffer noch eine alte Auswahl bestimmen, welche Änderungen in dieses Inventar aufgenommen werden.

requiredCoverage der Hülle nennt genaue bindings für die Quellenerschließung des äußeren Modellumfangs, die benötigten Fallbereiche und bei einer Übernahme die vollständige Änderungsprüfung. Ein neuer Stand darf alternativ vollständig neu erschlossen werden; ein ungebundener alter Abdeckungsbefund ersetzt diese Arbeit nicht. Für coverage und influence werden im Einstieg aktuelle Fassungen mit den Zielinventaren beziehungsweise dem genauen Standvergleich erstellt. Ihre Inhalte und unabhängigen Prüfungen bilden eine vorgelagerte Grundlage: Sie werden direkt gegen den festen Eingabestand validiert, ohne wiederum ihre eigene vollständige Einflussfreigabe vorauszusetzen. Die nachgelagerten interpretation-/application-Bindings konsumieren diese Grundlage. Reine records-Verweise aus Abdeckung und Einfluss auf betrachtete Behauptungen dienen der Nachvollziehbarkeit und sind keine wechselseitige Verwendbarkeitsvoraussetzung. So wird verhindert, dass Eintrag und Änderungsfreigabe ihre Gültigkeit gegenseitig ohne unabhängige Grundlage bestätigen.

#### Prüfberichte, Aktualitätsbindungen und abgeleiteter Status

Ein Review ist ein unveränderlicher Datensatz mit reviewId, subject als genauer RecordRef oder BindingRef, snapshot, procedures, check als structural, semantic oder coverage, verdict als pass, fail oder open, reviewer mit runId und Rolle, contextManifest als Inhaltsreferenz, evidence als Liste gebundener Belege und reason. Die Begründung ist immer erforderlich. Die Anforderungen an unabhängige semantische Prüfung stehen in der gebundenen reviewPolicy. Verschiedene runIds oder ein vom Erzeuger behauptetes independent: true genügen als Nachweis nicht; die Prüfumgebung und der gelieferte Originalkontext müssen die geforderte Trennung nachvollziehbar machen.

Die Binding ist die eigentliche Verwendbarkeitsaussage für einen bestimmten Stand. Erforderlich sind bindingId, record als RecordRef, targetSnapshot, procedures, mode als original oder reuse, dependencyChecks, coverageBindings und reason. Bei original muss targetSnapshot zum originSnapshot des Eintrags passen. Bei reuse ist previousBinding für genau denselben recordId erforderlich. dependencyChecks ist eine Liste von Objekten mit key, result als equal, reassessed, changed oder unresolved, evidence als Liste genauer Belegreferenzen und reason. Jede dependency.key erhält genau einen Befund; unbekannte oder doppelte Schlüssel sind Fehler. reassessed benötigt aktuelle konkrete Belege. coverageBindings referenziert die für den Eintrag nötige Quellen-, Fall- und Einflussprüfung; für deren vorgelagerte Validierung gilt die soeben beschriebene Reihenfolge. Die Binding erhält eigene strukturelle Prüfberichte und bei semantischen Übernahmeentscheidungen die nach Policy nötigen Reviews. Ein vollständig mechanischer Übernahmeschritt kann sich auf vorhandene passende semantische Prüfbelege stützen, statt die gleiche Auslegung nochmals beurteilen zu lassen.

Auch bei mode original müssen Abhängigkeiten und Abdeckung geprüft werden. previousBinding und replaces dokumentieren nur Herkunft; sie erfüllen keine einzige aktuelle Prüfpflicht automatisch. Exakte Referenzen und Validierungsabhängigkeiten sind endlich aufzulösen; zirkuläre Rechtfertigung ist ungültig. Historische und diagnostische Verweise dürfen Zyklen zeigen, erzeugen aber keinen Gültigkeitsbeweis.

Der Status wird aus Befunden und Bindungen berechnet und nicht als frei setzbares approved-Feld gespeichert:

| Achse | Werte und Ableitung |
| --- | --- |
| reviewState | unreviewed: nötige Belege fehlen; confirmed: alle erforderlichen Prüfungen erfolgreich; disputed: ein maßgeblicher fail-Befund liegt vor; open: eine erforderliche Prüfung bleibt ausdrücklich ungeklärt. disputed hat Vorrang vor open, open vor unreviewed. |
| freshness | current: Abhängigkeiten und erforderliche Abdeckung passen zum Zielstand; stale: eine maßgebliche Grundlage hat sich nachweislich geändert; pending: die Aktualitätsprüfung ist unvollständig. stale hat Vorrang vor pending. |
| usability | usable nur bei reviewState confirmed und freshness current sowie ohne ungeklärte Behauptung oder benötigten Kontext. uncertain, unresolved oder eine ungeklärte force erlauben keine verwendbare Geltung. Sonst blocked bei fail, conflict oder stale; andernfalls pending. |

Ein Reviewer darf eine Behauptung mit result conflict zutreffend bestätigen: Der Befund ist dann geprüft, der Konflikt aber weiterhin ungelöst. Er wird als Diagnose bereitgestellt, nicht als verwendbare positive oder negative Pflicht. Ebenso können einzelne Einträge usable sein, während die gesamte Hülle wegen unvollständiger requiredCoverage nur einen partiellen Kontext liefert. Eine spätere positive Review-Zeile überschreibt einen alten fail-Befund derselben Fassung nicht nach dem Prinzip „letzter gewinnt“. Der Entwurf verlangt in diesem Fall eine neue nachvollziehbare Befund-/Bindingfassung mit eigener Prüfung; alte Fassungen bleiben unverändert.

#### Exchange: lesbare Datensatzansicht und Übernahme

Die folgenden Referenznamen sind ausdrücklich symbolische Kurzformen exakter Inhalts-/Snapshotreferenzen, keine behaupteten realen Digests oder bereits gültigen Produktdateien. M4 bezeichnet den Stand mit Protokollierungsausnahme aus Abschnitt 11.12; T die allgemeine Zeitmessungsauslegung. D-CHECK, D-ROLE und D-ONPREM bezeichnen die jeweils vollständigen Definitionsdigests. Diese Ansicht zeigt einen application-Datensatz vollständig in seiner vorgeschlagenen Form; referenzierte Datensätze und Manifeste sind in der Ansicht ausgelagert.

```yaml
recordId: A-TIME-CHECK
recordType: application
originSnapshot: M4
procedures: INDEX-PROCEDURES-1
producer:
  runId: derive-time-check
  procedure: DERIVATION-1
sources: []
dependencies:
  - key: meaning
    type: record
    record: T
  - key: exchange
    type: content
    definition: {package: example/connector-project, id: check-status}
    expectedDigest: D-CHECK
  - key: role
    type: content
    definition: {package: example/connector-vocabulary, id: Exchange/sender}
    expectedDigest: D-ROLE
  - key: participant
    type: content
    definition: {package: example/connector-project, id: onprem}
    expectedDigest: D-ONPREM
  - key: new-influences
    type: watch
    scope: bound-model
claim:
  interpretation: T
  subject: {package: example/connector-project, id: onprem}
  context:
    - {package: example/connector-project, id: check-status}
  property: {package: example/connector-vocabulary, id: Exchange/sender}
  result: applies
  statement: onprem bestimmt die Wartefrist jedes Versuchs von check-status monoton.
  reason: check-status ist ein Exchange und sein requester ist onprem.
```

Die Quellen der Pflicht sind hier über T gebunden, statt als manuelle zweite Kopie gepflegt zu werden. Die konkret aufgeführten Grundlagen sind die wesentlichen direkten Quellen dieses Beispiels, keine pauschale Behauptung einer vollständigen Kontextliste. Weitere vom Reviewer benötigte Grundlagen müssen ebenfalls als dependencies erfasst werden; für die vollständige Validierung sind die aufgelösten Quellen von T, Prüfberichte und Abdeckungsbindungen erforderlich.

Für einen hypothetischen Folgestand M5 kann derselbe recordId übernommen werden, wenn seine direkten Grundlagen unverändert bleiben und die vollständige Änderungsprüfung keine ungeklärten Einflüsse auf diese Geltung lässt. Die Binding enthält dann targetSnapshot M5, mode reuse, previousBinding als genaue M4-Binding, equal-Befunde für unveränderte Grundlagen, einen reassessed-Befund mit aktueller Einflussprüfung für new-influences und die M5-Abdeckungsbindungen. Die ursprüngliche semantische Prüfung bleibt eine Prüfung ihres ursprünglichen Gegenstands; die neue Binding erklärt ihre zulässige Wiederverwendung.

Wechselt dagegen check-status.requester, ändert sich mindestens D-CHECK. Ein Übernahmeversuch für A-TIME-CHECK liefert für diesen Stand freshness stale; die historische M4-Binding wird nicht nachträglich geändert. Eine neue Anwendung mit geändertem subject und aktuellen Inhaltsabhängigkeiten braucht einen neuen recordId; sie kann T als Bedeutungsvoraussetzung weiterverwenden, falls dessen eigene Binding current bleibt. Ändert sich die globale Protokollierungsauslegung L3 zu L4, ist dasselbe Prinzip auf die davon abhängigen Anwendungen anzuwenden. Eine übernommene Zeitmessungsanwendung ersetzt keine neu erforderliche Protokollierungsanwendung oder deren geprüften Ausschluss.

#### Mindestvalidierung und verbleibende Arbeit

Ein späterer Validator muss geschlossene Formen, Enums, eindeutige Schlüssel, referenzierte Inhalte, Quellenbereiche, Inventarvollständigkeit, genaue Prüfgegenstände, Abhängigkeits-/Bindingbezüge und Statusableitung prüfen. Es gibt keine stillen Namenssuchen oder Fallbacks auf ältere bestätigte Fassungen. Eine alte oder unvollständige Binding kann als Diagnose gelesen werden, aber nicht als Bestätigung für den aktuellen Stand.

Der Vertrag legt noch keine JSON-Schema-Datei, ausführbare Abfragesprache oder unabhängige Review-Implementierung vor. Insbesondere bleiben Digestkodierung, die vollständig maschinenlesbaren Manifestformen, Kriterien geprüfter Auswahlausführung und die konkrete Durchsetzung der Reviewtrennung auszuarbeiten. Die strukturierte Prüfung eines späteren Transportformats wird keine semantische Garantie der gewählten Abhängigkeiten oder Geltungsurteile schaffen. Das YAML-Beispiel wurde auf Lesbarkeit geprüft, nicht gegen einen produktiven Indexvalidator oder einen ausgeführten semantischen Review.

### 11.14 Vollständiges Datensatzprotokoll für M3 → M4

Der Nutzer beauftragt die vollständige Instanziierung des Datenvertrags für die Ausnahme von check-status und eine Einordnung des Sprachstands. Der folgende Katalog füllt alle im betrachteten Übergang benötigten Befunde, Prüfungen und Bindings in einer symbolischen Lesedarstellung aus. Die Referenznamen vertreten genaue Inhaltsreferenzen; sie sind keine erfundenen echten SHA-Werte. Das Protokoll nimmt einen bereits geprüften M3-Ausgangsindex und die unten beschriebenen erfolgreichen neuen Reviews an. Diese Reviews wurden nicht ausgeführt. Das ist kein selbständig ladbares Transportpaket des ganzen Projektindex; unveränderte Hintergrunddaten des angenommenen Ausgangsindex und die vollständigen Inventarmanifeste werden als verfügbare Voraussetzungen benannt, statt sie als neue Erkenntnisse auszugeben.

#### Fest gebundene Eingaben und Ausgangszustand

| Referenz | Ausgefüllter Inhalt dieses Beispiels |
| --- | --- |
| M3 | Snapshot mit beiden Exchanges, requester api für list-documents und onprem für check-status; uneingeschränkte zusätzliche Protokollierungspflicht. |
| M4 | Derselbe Definitionsbestand, dieselben Paketbindungen und Rollenbelegungen; ausschließlich die globale Protokollierungsaussage im platform.body nimmt nun check-status aus. |
| P1 | Unveränderte indexContract-, derivationProcedure- und reviewPolicy-Fassungen. Die Beispielpolicy verlangt strukturelle Prüfung jeder neuen Record-/Bindingfassung und unabhängige semantische beziehungsweise Abdeckungsprüfung der entsprechenden neuen Befunde. |
| D-EXCHANGE, D-ROLE, D-LIST, D-CHECK, D-API, D-ONPREM | In beiden Ständen gleiche vollständige Inhaltsfassungen der jeweils benannten Definitionen; D-ROLE bezeichnet Exchange/sender im Vokabularpaket. Weitere gelesene Teilnehmer-/Command-/Channelverträge sind im unveränderten Kontextmanifest gebunden. |
| D-PLATFORM-3, D-PLATFORM-4 | Unterschiedliche vollständige Inhaltsfassungen derselben Projektdefinition platform. Keine Inhaltsgleichheit behaupten. |
| SOURCES-3, SOURCES-4 | Vollständige Inventare der Quell-bodies und Bedeutungsgrundlagen am jeweiligen Stand; genau ein geänderter Quellbereich. Die Eingabe-/Paketmanifeste und Originaltexte sind verfügbar. |
| CASES-3, CASES-4 | Dieselben zwei betrachteten Fälle mit den aktuellen Rollenbelegungen: list-documents → api, check-status → onprem. Die positive Auswahl der Logpflicht ist davon zu unterscheiden. |
| DELTA-34 | Vollständiges Vergleichsinventar M3 → M4 mit dem einen body-Änderungsbereich und dessen beiden Quellfassungen. |

Die sechs fachlichen Ausgangseinträge sind T, T1, T2, L3, L3-1 und L3-2 aus Abschnitt 11.12. T interpretiert die monotone Zeitmessung, T1 und T2 wenden sie auf die zwei Fälle an. L3 interpretiert die Protokollierung ohne Ausnahme, L3-1 und L3-2 wenden sie an. Alle haben genaue M3-Bindings B3-T, B3-T1, B3-T2, B3-L3, B3-L3-1 und B3-L3-2 sowie vorhandene bestandene Record- und Bindingprüfungen. Die vollständige Hintergrunderschließung und ihre M3-Abdeckung sind weitere angenommene Voraussetzungen.

Für die Zeitmessungsbefunde sind die direkten Bedeutungsgrundlagen im Beispiel Exchange, dessen Auftraggeber-Property und die benötigten konkreten Fallverträge; die globale Protokollierungsquelle ist keine unverändert zu vergleichende direkte content-Abhängigkeit von T. Ihre möglichen Einflüsse werden über watch und die aktuelle Einflussprüfung beurteilt. Hätte ein tatsächlicher Befund D-PLATFORM-3 direkt als Grundlage gebunden, könnte genau diese alte Datensatzfassung nicht unverändert übernommen werden: Dann wäre ein neu begründeter Befund erforderlich, auch bei gleicher Schlussfolgerung. Die folgende Wiederverwendung gilt ausdrücklich für die hier festgelegten Abhängigkeiten.

#### Acht neue Records und drei wiederverwendete fachliche Records

Für jeden neuen Record sind originSnapshot M4, procedures P1 und producer {runId: scenario-derive-M4, procedure: P1.derivationProcedure} gesetzt. sources nennen die genaue neue Fundstelle in D-PLATFORM-4, soweit die Auslegung betroffen ist; die Anwendungen binden diese Herkunft über L4. dependencies enthalten die nachstehend genannten vollständigen Grundlagen und gegebenenfalls watch. Die Kürzel bezeichnen echte ModelRefs beziehungsweise RecordRefs nach Abschnitt 11.13; die Eintragsart bestimmt den dort vollständig definierten claim-Aufbau.

| recordId / recordType | claim und konkrete Grundlage |
| --- | --- |
| L4 / interpretation | meaning: requester protokolliert pro Versuch die Uhrquelle für alle Exchanges außer check-status; force: obligation; scope.description: alle Exchange-Fälle außer dem ausdrücklich bezeichneten Austausch; scope.inventory: CASES-4. Quelle: neuer globaler Satz in D-PLATFORM-4. dependencies: content platform=D-PLATFORM-4, content Exchange=D-EXCHANGE, content Exchange/sender=D-ROLE, content check-status=D-CHECK für das aufgelöste Ausnahmeziel, watch bound-model. replaces: L3. |
| L4-1 / application | interpretation L4; subject api; context [list-documents]; property Exchange/sender; result applies; statement: api protokolliert je Versuch von list-documents die Uhrquelle; reason: dieser Exchange fällt nicht unter die ausdrücklich benannte Ausnahme. dependencies: record L4, content list-documents=D-LIST, api=D-API, Auftraggeber-Property und weitere benötigte unveränderte Fallverträge, watch. replaces: L3-1. |
| L4-2 / application | interpretation L4; subject onprem; context [check-status]; property Exchange/sender; result excluded; statement: Aus L4 folgt für onprem bei check-status keine Protokollierungspflicht; reason: das exakt aufgelöste Ausnahmeziel ist dieser Exchange. dependencies: record L4, content check-status=D-CHECK, onprem=D-ONPREM, Auftraggeber-Property und benötigte Fallverträge, watch. replaces: L3-2. |
| I34 / influence | baseSnapshot M3; targetSnapshot M4; changes DELTA-34; considered [T,T1,T2,L3,L3-1,L3-2]; result affects; affected [L3,L3-1,L3-2]; reason: die neue Einschränkung ändert die Logauslegung und ihre Anwendungen, betrifft aber nicht die Zeitmessung. Quelle: beide globalen Quellfassungen. dependencies: beide festen Vergleichsmanifeste und die geprüften Bedeutungsgrundlagen. Keine Voraussetzung einer bereits aktuellen T-/L4-Binding. |
| CS4 / coverage | purpose sources; inventory SOURCES-4. Jede Quelle besitzt einen Befund: der geänderte globale Satz verweist auf L4; unveränderte Quellbereiche verweisen nach Aktualitätsprüfung auf vorhandene oder nötigenfalls neu gebundene Hintergrundbefunde. Keine Quelle wird wegen fehlender Links übersprungen. Diese vorausgesetzte vollständige Quellabdeckung ist mitzuliefern, nicht durch die acht Vordergrundrecords allein bewiesen. |
| CT4 / coverage | purpose cases; inventory CASES-4; results: list-documents → assessed mit [T1], check-status → assessed mit [T2]. reason jeweils: die Protokollierungsausnahme ändert die monotone Zeitmessung nicht. |
| CL4 / coverage | purpose cases; inventory CASES-4; results: list-documents → assessed mit [L4-1], check-status → excluded mit [L4-2]. reason: explizite Ausnahme betrifft genau den zweiten Fall und ausschließlich L4. Beide Fälle bleiben in der Abdeckung. |
| CD34 / coverage | purpose changes; baseSnapshot M3; inventory DELTA-34; results: der eine Änderungsbereich → assessed mit [I34]. reason: frühere und neue Reichweite samt Wechselwirkungen beurteilt; keine weiteren Änderungsbereiche im vollständigen Inventar. |

Bei allen vier Coverage-Records bilden die vollständigen jeweiligen Inventare die direkte Eingabegrundlage. Ihre related RecordRefs beschreiben betrachtete Behauptungen und sind keine gegenseitigen Aktualitätsvoraussetzungen. Die genau gebundenen Quell-/Inventarbelege stehen jeweils in sources beziehungsweise dependencies. watch wird für diese vorgelagerten Zertifikate nicht durch eine Forderung nach ihrer eigenen bereits fertigen Freigabe ersetzt: Es gilt die direkte unabhängige Prüfung am festen Stand aus Abschnitt 11.13.

T, T1 und T2 erhalten keine neuen recordIds und keine geänderten originSnapshots. Ihre Inhaltsobjekte und ursprünglichen Reviews bleiben unverändert erhalten. L3, L3-1 und L3-2 bleiben historische M3-Befunde. Sie erhalten keine erfolgreiche M4-Übernahmebindung. Insbesondere wird L3-2 nicht in place von applies zu excluded geändert.

#### Die neuen Reviews vollständig zuordnen

Alle folgenden Sollberichte haben snapshot M4 und procedures P1. reviewer.runId unterscheidet den strukturellen Validatorlauf scenario-validate-M4 vom unabhängigen semantischen Prüflauf scenario-review-M4. Die Namen sind ausdrücklich Szenariobezeichnungen und kein Beleg tatsächlicher Unabhängigkeit. contextManifest enthält die jeweils genannten Originalquellen, beide festen Modellstände soweit nötig und die konkret betrachteten Befunde. evidence bindet die Quellvergleiche, Strukturbelege oder unabhängig beurteilten Geltungsgründe. verdict pass ist hier jeweils die angenommene erfolgreiche Beurteilung, kein ausgeführtes Ergebnis.

| Review-IDs und genauer subject | check | Ausgefüllter Prüfgrund und Kontext |
| --- | --- | --- |
| RS-L4, RS-L4-1, RS-L4-2, RS-I34, RS-CS4, RS-CT4, RS-CL4, RS-CD34; jeweils gleichnamiger Record | structural | Geschlossene Datensatzform, exakte Quell-/Zielreferenzen, Inventare, Abhängigkeitsbezüge und Herkunft sind für die genaue neue Fassung gültig. |
| RM-L4 → L4 | semantic | Der vollständige Quellvergleich begrenzt nur die zusätzliche Protokollierung. check-status ist eindeutig identifiziert; die Zeitmessungspflicht wird nicht verändert. |
| RM-L4-1 → L4-1 | semantic | list-documents liegt außerhalb der Ausnahme; api ist dort weiterhin requester. Die positive Anwendung passt zu den konkreten Verträgen. |
| RM-L4-2 → L4-2 | semantic | Der Ausschluss gilt für onprem nur bei check-status und nur aus L4; er behauptet weder ein Protokollierungsverbot noch eine Zeitmessungsausnahme. |
| RM-I34 → I34 | semantic | Die Änderung beansprucht keine weitere Ausnahme. Alte und neue Reichweite wurden anhand der Quellen und relevanten Gegenregeln geprüft. Die Wiederverwendbarkeit von T/T1/T2 bleibt auf deren unveränderte direkte Grundlagen beschränkt. |
| RC-CS4 → CS4 | coverage | Sämtliche Elemente von SOURCES-4 haben passende Befunde; ungeklärte Hintergrundbereiche würden diesen pass-Bericht verhindern. Die vorhandene M3-Abdeckung allein genügt nicht. |
| RC-CT4 → CT4 | coverage | Beide aktuellen Exchange-/requester-Fälle sind für T betrachtet; keiner fällt durch einen Rollen- oder Mengenwechsel aus der Auswahl. |
| RC-CL4 → CL4 | coverage | Die Grundmenge enthält beide Fälle; positive Anwendung und expliziter Ausschluss sind vollständig und korrekt unterschieden. |
| RC-CD34 → CD34 | coverage | DELTA-34 stammt aus dem vollständigen Standvergleich und ist vollständig beurteilt. |

Die ursprünglichen T-/T1-/T2-Reviews bleiben Belege ihrer ursprünglichen Recordfassungen. Es entstehen keine fingierten neuen semantischen T-Reviews allein durch Änderung des Snapshotnamens. Für die Übernahme werden die neuen Einfluss-/Abdeckungsberichte und die mechanisch geprüften unveränderten Grundlagen hinzugezogen. Jeder fail oder open an einer benötigten Stelle würde den beschriebenen erfolgreichen Endzustand verhindern.

#### Elf neue Bindings und ihre Abhängigkeitsprüfungen

Jede neue Binding hat targetSnapshot M4, procedures P1 und einen nichtleeren reason aus der folgenden Tabelle. Für jede dependency.key des gebundenen Records wird genau ein dependencyChecks-Ergebnis eingetragen, mit Beleg auf den vollständigen Inhaltsvergleich, den aktuellen Vorgängerbefund oder die benannte neue Beurteilung. Die aktuelle Gleichheit der direkten Grundlagen wird nicht aus einer unveränderten Dateibezeichnung gefolgert.

| bindingId → record | mode / previousBinding | dependencyChecks und coverageBindings |
| --- | --- | --- |
| B4-I34 → I34 | original | Vergleichsmanifeste und genaue Quellfassungen geprüft; direkte Belege aus M3/M4. coverageBindings leer: vorgelagerte Einflussprüfung benötigt nicht ihre eigene Freigabe. |
| B4-CS4 → CS4 | original | Quelleninventar M4 vollständig geprüft. coverageBindings leer; direkte eigenständige Abdeckungsprüfung. |
| B4-CT4 → CT4 | original | Beide Fallzuordnungen geprüft. coverageBindings leer; die zugehörigen semantischen Belege stehen in den Quellen-/Einflussprüfungen und Recordprüfungen. |
| B4-CL4 → CL4 | original | Beide Fälle einschließlich des Ausschlusses geprüft. coverageBindings leer; Originalquelle L4 und konkrete Belegung stehen im Reviewkontext. |
| B4-CD34 → CD34 | original | Vollständiges Delta und I34 geprüft. coverageBindings [B4-I34]; keine Rückkante auf B4-CD34. |
| B4-T → T | reuse / B3-T | Direkte Inhalte und genaue Vorbefunde equal; new-influences reassessed mit I34 und CD34. coverageBindings [B4-CS4,B4-CD34]. Grund: dieselbe Zeitmessungsauslegung, begründet unberührt von der Änderung. |
| B4-T1 → T1 | reuse / B3-T1 | Genaue Auslegung T über B4-T verwendbar; direkte Fall-/Property-/Teilnehmergrundlagen equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CT4]. Grund: api bleibt für list-documents verpflichtet. |
| B4-T2 → T2 | reuse / B3-T2 | Genaue Auslegung T über B4-T verwendbar; direkte Fall-/Property-/Teilnehmergrundlagen equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CT4]. Grund: onprem bleibt für check-status zur Zeitmessung verpflichtet. |
| B4-L4 → L4 | original | Neue Quellfassung und alle direkten Inhaltsgrundlagen gegen ihre originSnapshot-M4-Werte equal; watch reassessed mit I34. coverageBindings [B4-CS4,B4-CD34]. Grund: geprüfte neue Auslegung mit begrenztem Geltungsbereich. |
| B4-L4-1 → L4-1 | original | Auslegung L4 über B4-L4 verwendbar; konkrete neue Recordgrundlagen equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CL4]. Grund: positive Protokollierungspflicht für api bei list-documents. |
| B4-L4-2 → L4-2 | original | Auslegung L4 über B4-L4 verwendbar; konkrete neue Recordgrundlagen equal; watch reassessed. coverageBindings [B4-CS4,B4-CD34,B4-CL4]. Grund: geprüfter Ausschluss von onprem bei check-status aus dieser Pflicht. |

Für jede dieser elf Bindings entsteht ein struktureller Sollbericht RB-<bindingId> mit dem exakten BindingRef als subject, snapshot M4, procedures P1, check structural und angenommenem verdict pass. Sein contextManifest enthält Record, genaue alte Binding bei reuse, dependencyChecks, aktuelle Abdeckungs-/Einflussbindungen und die benötigten unabhängigen semantischen Belege. reason lautet jeweils: vollständige Abhängigkeiten und zulässige Belegübernahme am Zielstand geprüft. Ein equal-Befund auf einen vorausgesetzten Record setzt dessen aktuelle zulässige Binding voraus; es wird nicht nur seine Zeichenfolge verglichen.

Damit enthält der betrachtete Vordergrund **acht neue Records, elf neue Bindings und 27 neue Soll-Reviews**: acht Record-Strukturprüfungen, vier semantische Reviews, vier Abdeckungsreviews und elf Binding-Strukturprüfungen. Hinzu kommen die als vorhanden vorausgesetzten unveränderten beziehungsweise eigenständig zu aktualisierenden Hintergrundbefunde. Das ist eine konkrete einfache Aufteilung der Beispielpolicy, keine Pflicht zu 27 getrennten Agentenläufen oder eine allgemeine Kostenangabe. Berichte können aus denselben passenden unabhängigen Läufen stammen, ihre exakten Prüfgegenstände bleiben getrennt.

Die M4-Hülle referenziert als requiredCoverage [B4-CS4,B4-CT4,B4-CL4,B4-CD34]. Die drei aktuellen Zeitmessungsbindungen und drei aktuellen Logbindungen bilden den betrachteten fachlichen Ausschnitt. Erst bei tatsächlich bestätigten Prüfungen und vollständiger Hintergrundabdeckung wäre daraus confirmed/current/usable abzuleiten. Hier sind die erfolgreichen Reviews angenommen, daher ist dieses Protokoll kein aktueller Produktnachweis. Die M3-Bindings bleiben historische gültige Bindings für M3. Ein Übernahmeversuch von L3/L3-1/L3-2 nach M4 wäre wegen geänderter Quell-/Auslegungsvoraussetzungen stale; diese Diagnose verändert die alten Bindings nicht.

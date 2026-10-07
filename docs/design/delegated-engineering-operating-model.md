# Delegiertes Entwicklungsbetriebsmodell

**Status:** Architekturvorbereitung und prüfbarer Diskussionsentwurf. Kein angenommener Vertrag, keine implementierte Funktion und kein Auftrag zum Umbau.

**Vergleichsbasis:** finaler Architect-Kandidat `1ea5c76f55526fc4d721e865885436153f48b497`. Der [Abschlussabgleich](architect-government-checkpoint.md) bestätigt den begrenzten Grundmodell-Checkpoint und unveränderte technische Verträge gegenüber der ursprünglichen Vorbereitung auf `dc10f54`. Der Entwurf selbst ist weiterhin nicht angenommen oder implementiert.

## Zweck und Ziel

Das Ziel ist eine kontinuierliche, weitgehend autonome Entwicklungsmaschine für ein bestehendes, möglicherweise unübersichtliches Repository. Nach einer initialen Bestandsaufnahme und Zielklärung soll sie umfangreiche Arbeit ordnen, priorisieren, parallel ausführen, integrieren, prüfen, reparieren und nach Unterbrechungen fortsetzen.

Der Nutzer soll Ziele, tragende Regeln, Grenzen und Delegationen festlegen. Gewöhnliche technische Entscheidungen und Routinearbeit sollen innerhalb dieses Rahmens ohne Mikro-Erlaubnislisten und ohne einzelne Freigabe jedes Schritts erfolgen.

Das Modell soll den Ist-Zustand nicht mit dem Soll verwechseln. Beobachteter Code beweist nicht, dass eine Architektur beabsichtigt ist. Ein grüner Test beweist nicht, dass das Produktziel erfüllt ist. Eine erfolgreiche Umsetzung schafft keine neue Befugnis.

Dieser Entwurf verwendet „Verfassung“, „Regierung“ und „Ministerium“ als mögliche Sprache für ein Betriebsmodell. Die Wahl dieser Produktidentität bleibt offen. Die technischen und organisatorischen Verantwortungen sind unabhängig davon zu prüfen.

Der Nutzer hat kontinuierliche, weitgehend autonome Entwicklungsarbeit bei wenig Routineaufsicht als zentrale Erwartung benannt. Offen ist, wie weit der finale Architect-Kandidat dieses Ziel trägt und welche Government-Produktform dafür angemessen ist. Der [Architektur-Abgleich](delegated-engineering-change-map.md) ordnet bestehende Mechaniken zu; der [gestufte Validierungsplan](delegated-engineering-validation-plan.md) beschreibt mögliche Nachweise nach dem Architect-Checkpoint.

## Festgehaltene Leitplanken

Die folgenden Leitplanken wurden am 2026-10-07 in der Diskussion ausdrücklich festgehalten. Sie beschreiben das zu bewahrende Zielbild, keine bereits vollständig implementierte Fähigkeit. Die konkrete Government-Architektur und ein paralleler Implementierungsversuch bleiben zur Diskussion gestellt; der separat beauftragte Classic-Release wird dadurch nicht angehalten.

### Repository als materialisiertes Abbild

Das gesamte Projekt-Repository soll als Abbild des akzeptierten kanonischen Modells verstanden werden. Der Compiler-/Kubernetes-/Terraform-Gedanke bleibt zentral: akzeptiertes Soll, beobachteter Dateistand, abgeleitete Arbeit, kontrollierte Materialisierung und erneute Prüfung bilden einen Abgleichkreislauf. Ziel ist die vollständige Erfassung des Repository-Umfangs; die heutige deklarierte Teilabdeckung darf nicht als bereits vollständige Erfüllung dieses Ziels dargestellt werden.

Eine Umsetzung im Repository muss sich in konkreten Dateien beziehungsweise Dateiänderungen nachweisen lassen. Dazu zählen Code, Dokumentation, Tests, Konfiguration, Automatisierung und andere Repository-Artefakte. Ein Plan, eine Agentenantwort oder eine Zustimmung allein ist keine implementierte Änderung. Vorhandene Dateien beweisen ihrerseits noch keine korrekte Umsetzung oder einen erfolgreichen externen Betrieb. Verpflichtungen über unerlaubte Abhängigkeiten oder andere Abwesenheiten brauchen geeignete Prüfevidenz, keine erfundene zusätzliche Implementierungsdatei.

Die Beziehungen müssen in beide Richtungen nachvollziehbar sein: Welche Artefakte implementieren einen Modellgegenstand, und welche Modellgegenstände beziehungsweise Verpflichtungen begründen ein Artefakt? Ein Gegenstand kann mehrere Dateien betreffen; eine Datei kann mehrere Modellgegenstände realisieren. Diese Zuordnung ist von der eindeutigen Verantwortung für einen konkreten Schreibvorgang zu unterscheiden. Zielzustände dürfen als Vorgaben mit Implementierungsfreiheit beschrieben sein; nicht jeder Dateiname oder jede Methode muss vorab im Sollmodell stehen.

Kanonische Quelldateien im selben Repository sind als Modelleingaben zu klassifizieren. Ihre Ablage macht sie nicht zu selbstgenerierenden Ausgaben. Fremdverwaltete und noch ungeklärte Dateien brauchen eine ausdrückliche Einordnung, damit der Anspruch auf vollständige Abdeckung überprüfbar bleibt. Die genaue Repräsentation dieser Zuordnungen und Kategorien ist eine offene Designfrage.

Pflichtangabe `purpose`, explizite Modellbeziehungen und daraus abgeleitete Impact-Betrachtungen bleiben gewünschte Grundlagen. Ein Zwecktext erklärt die Existenz eines Gegenstands; er beweist weder dessen Sinnhaftigkeit noch die Einhaltung seiner Verpflichtungen.

### Rekursive Managementhierarchie

Das untersuchte Organisationsbild ist eine Managementhierarchie über Verantwortungsbereiche: Auftrag oder Änderung → betroffene Bereiche und Verantwortliche bestimmen → Ziele und Befugnisse nach unten delegieren → denselben Prozess bei Bedarf rekursiv durchlaufen → ausführen → unabhängig prüfen und Ergebnisse rekursiv nach oben integrieren. Jede Ebene behält die Verantwortung für ihren gesamten Auftrag und ihre eigenen Integrationspflichten.

Das ersetzt den Abgleich zwischen Modell und Dateien nicht. Die Hierarchie organisiert, wer diesen Abgleich für welchen Teil verantwortet, wer entscheidet und wer unabhängig prüft. Fachlicher Modellgraph, Artefaktzuordnung, Codeabhängigkeiten, Delegationshierarchie und Prüfbeziehungen sind miteinander verbunden, haben aber unterschiedliche Bedeutungen. Ob und wie dauerhafte fachliche Ministerien zusätzlich in diese Hierarchie eingebunden werden, bleibt offen.

### Veränderbare Granularität der Agentenarbeit

Agenten sollen bei Bedarf für eng begrenzte Verantwortungen und Prüfaufträge erzeugt werden können, bis hin zu einem eigenen Prüflauf für eine einzelne Invariante. Verantwortlichkeit besteht dauerhaft im Modell; die konkrete Agenteninstanz kann für einen Vorgang entstehen. Jede zusätzliche Instanz erhält den für ihre Pflicht notwendigen Kontext und die gebundene Evidenz.

Die geringe personelle Beschaffungs- und Einarbeitungshürde gegenüber menschlichen Teams ist eine Gestaltungsmöglichkeit. Kontextaufbereitung, Modellaufrufe, Tokenverbrauch, Laufzeit, Abstimmung und Ergebnisintegration bleiben reale Kosten. Ein gesonderter Agent pro Invariante ist deshalb eine mögliche Granularität, kein pauschaler Standard und kein automatischer Qualitätsnachweis. Deterministische Prüfungen bleiben dort sinnvoll, wo sie eine Verpflichtung verlässlich prüfen können.

Die Granularität soll konfigurierbar beziehungsweise begründet wählbar sein. Bündeln oder Aufteilen darf keine Prüfpflicht entfernen: Verantwortlicher, betrachteter Stand, Ergebnis und offene Unsicherheit bleiben nachvollziehbar. Nach welchen Kriterien das System selbst weiter zerlegen darf, welche Grenzen gelten und wie Kosten gegen Qualität abgewogen werden, ist noch zu entscheiden.

## Vier getrennte Arbeitsgrundlagen

| Grundlage | Inhalt und Zweck |
|---|---|
| Beobachteter Ist-Zustand | Tatsächliche Dateien, Komponenten, Schnittstellen, Abhängigkeiten, Build- und Laufzeitbefunde, vorhandene Aufgaben sowie erkennbare Unbekannte. Jeder Befund braucht Herkunft und Stand; Unsicherheit bleibt sichtbar. |
| Akzeptiertes Soll | Ziele, tragende Projektregeln, gewünschte Verantwortungsgrenzen, Qualitätsmaßstäbe, Prioritäten und delegierte Entscheidungsfreiheit. Vorschläge werden erst nach einem gültigen Beschluss zum geltenden Soll. |
| Backlog | Konkrete Produktarbeit, Defekte, Konsolidierung, fehlende Nachweise und Übergangsschritte. Jeder Eintrag verweist auf Ziel oder Befund, Abhängigkeiten und überprüfbares Erledigungskriterium. |
| Evidenz | Gebundene Quellstände, Änderungsartefakte, Prüfungen, Einwände, Urteile und Beschlüsse. Evidenz beschreibt, was beobachtet oder beurteilt wurde; sie legitimiert keine neue Absicht. |

Diese Trennung verlangt nicht automatisch vier neue Ressourcentypen. Entscheidend sind unterschiedliche Bedeutung, Herkunft und Änderungswirkung.

Der Bestand wird schrittweise erfasst. Unbekannte Bereiche dürfen als solche dokumentiert werden; sie sind weder automatisch fehlerhaft noch automatisch akzeptiert. Gültige bestehende Arbeit bleibt erhalten, bis ein begründeter Auftrag sie ändert.

Das akzeptierte Soll benennt die gewünschte Welt und ihre tragenden Regeln. Es muss nicht jede Datei, Klasse oder gewöhnliche Implementierungsentscheidung vorab festlegen.

Der Backlog ist die ausführbare Differenz zwischen akzeptiertem Soll, beobachtetem Bestand und ausdrücklich angenommenen Produktaufträgen. Eine Idee oder Agentenempfehlung wird nicht allein durch Ablage zum angenommenen Arbeitsauftrag.

## Hierarchie, Befugnis und Berichte

Ziele und Entscheidungsfreiheit werden von der übergeordneten Ebene nach unten übertragen. Arbeitsergebnisse, Risiken, Einwände und Vorschläge werden nach oben berichtet. Ein Vorgesetzter koordiniert und entscheidet nur innerhalb seiner delegierten Zuständigkeit.

Eine Delegation soll Ziel, Verantwortungsbereich, tragende Grenzen und wesentliche Eskalationsfälle in verständlicher Form nennen. Sie soll keine vollständige Liste zulässiger Implementierungsschritte werden.

Eine untergeordnete Ebene darf technische Mittel, Zerlegung und Reihenfolge innerhalb des übertragenen Ziels wählen. Sie darf dadurch weder eigene Befugnisse erweitern noch höherliegende Ziele, geschützte Grenzen oder die Annahmeregel ändern.

Jede Ebene liefert für jeden Durchlauf einen Bericht. Mindestens gehören dazu erledigte Arbeit, betroffene Bereiche und Schnittstellen, verwendete Belege, Prüfergebnisse, offene Einwände, Unsicherheiten, Folgearbeit und erforderliche Entscheidungen.

Eine Ebene berichtet auch, wenn sie keine Arbeit findet oder nicht zuständig ist. „Nicht betroffen“ ist eine explizite Feststellung mit Begründung, kein stilles Auslassen.

Die Anzeige kann pro Ebene und insgesamt zwischen „alles“, „Wichtiges“ und „nichts“ unterscheiden. Das ändert nur Sichtbarkeit und Benachrichtigung. Es ändert keine Zuständigkeit, Delegation, Zustimmung oder Annahmebedingung. Unterdrückte Anzeige bedeutet nicht unterdrückte Erfassung.

## Ein Durchlauf von Auftrag bis Abschluss

### 1. Auftrag und feste Arbeitsgrundlage

Ein Durchlauf bindet einen unveränderlichen Quellstand, das akzeptierte Soll, den Backlog-Ausschnitt, die relevanten Bereichsmandate, Werkzeuge, Ressourcenlimits und verfügbare aktuelle Evidenz.

Der Auftrag kann eine Solländerung, eine Produktaufgabe, eine Reparatur, Bestandsaufnahme oder Wiederaufnahme sein. Der Typ des Auftrags bestimmt, welche Beschlüsse und Nachweise erforderlich sind.

Eine Veränderung von Modell, Mandat, Ressortauswahl, Werkzeugen oder relevanten Quelldaten macht davon abhängige Pläne, Zustimmungen und Evidenz auf dem neuen Stand überprüfungsbedürftig.

### 2. Planung und Delegation

Die koordinierende Ebene zerlegt den Auftrag nach fachlicher Zuständigkeit, Abhängigkeiten und Integrationsgrenzen. Sie plant Änderungen an Modell, Code, Tests, Dokumentation und Betrieb als zusammenhängende Arbeit.

Jeder Verantwortungsbereich erhält pro Durchlauf einen eigenen Agentendurchlauf beziehungsweise eine eigene Agenteninstanz mit passendem Ziel, geltenden Regeln, betroffenen eigenen Artefakten, benötigtem Kontext sowie relevanten Schnittstellen und Nachbarverträgen.

Kontext wird ausreichend für die Aufgabe bereitgestellt, aber auf den Verantwortungsbereich und seine tatsächlichen Abhängigkeiten begrenzt. Fehlen erforderliche Nachbarbelege, kann der Agent sie gezielt anfordern; jeder neue Input wird an Quelle und Stand gebunden. Unbekannte oder nicht beschaffbare Belege bleiben sichtbar und werden nicht durch Annahmen ersetzt. Der Agent soll nicht jede andere Ressorthistorie als Eingabe erhalten müssen.

Unabhängige Aufgaben können parallel bearbeitet werden, wenn gemeinsame Dateien, Schnittstellen, Daten und Integrationsreihenfolge dies zulassen. Erkannte Abhängigkeiten werden geplant, statt Parallelität allein als Fortschritt zu zählen.

### 3. Ausführung und Kandidatenbildung

Ausführende Agenten erstellen begrenzte Änderungskandidaten. Sie ändern innerhalb ihrer übertragenen Verantwortung; sie dürfen Anforderungen nicht stillschweigend abschwächen, um ihre eigene Arbeit zu legitimieren.

Ein Kandidat enthält die resultierenden Bytes und Modi, die betroffenen Modell- und Quellstände, Zuordnung zu Aufträgen, Ausführungsnachweise und offene Punkte. Eine textuelle Behauptung „fertig“ ersetzt diese Identitäten nicht.

Nach unabhängigen Teilaufgaben integriert eine verantwortliche Ebene deren Kandidaten in einen gemeinsamen Stand und prüft die Zusammensetzung. Einzelerfolg beweist nicht, dass die Änderungen zusammenpassen.

### 4. Unabhängige Prüfung und Einwände

Prüfung ist ein eigener Agentendurchlauf mit eigener Aufgabe, passendem Kontext und eigener Evidenz. Der prüfende Agent darf nicht allein die Annahme seiner eigenen Änderung feststellen.

Die Prüfung betrachtet lokale Regeln, die Auswirkungen auf direkte Nachbarn und die Komposition an der verantwortlichen Elternebene. Prüfkontext enthält die relevanten Schnittstellen und veränderten Nachbarzustände, nicht ungebundene Gesprächsbehauptungen.

Technische Prüfungen belegen nur ihren erklärten Umfang. Semantische Urteile nennen Regel, Befund, Referenzen, Unsicherheit und überprüfbare Erledigungsbedingung. Fehlende oder fehlerhafte Prüfungen bleiben offen und werden nicht zu Erfolg umgedeutet.

Wenn ein Kandidat abgelehnt oder repariert wird, entsteht ein neuer Kandidat. Frühere Ergebnisse bleiben als Historie erhalten, gelten jedoch nicht ohne erneute Bindungsprüfung für die neue Fassung.

### 5. Annahme und Übernahme

Planung, Ausführung, Verifikation, fachliches Urteil und Übernahme sind getrennte Zustände. Ein Plan autorisiert keine Ausführung. Ein Executor-Vorschlag ist kein Beschluss. Ein Prüfergebnis allein überträgt keinen Kandidaten in den akzeptierten Projektstand.

Falls das Government-Modell beschlossen wird, bleibt die bisher ausdrücklich genannte Annahmeregel erhalten: Jedes ausgewählte Ministerium muss der endgültigen exakten Fassung ausdrücklich zustimmen. Eine begründete Zustimmung „nicht betroffen“ zählt als Stimme. Schweigen, Timeout, unvollständige Prüfung oder Zustimmung zu einer älteren Fassung zählen nicht.

Ein Ressorturteil, ein gerichtliches Urteil und ein ausführbares Artefakt sind verschiedene Gegenstände. Artefaktidentität beweist nicht Zustimmung; Zustimmung ist kein Buildnachweis; ein Gerichtsurteil ist keine Gesetzesänderung.

Die technische `Apply`-Grenze der Vergleichsbasis darf nicht als Isolation oder endgültige Annahme beschrieben werden. Der Architect-Vertrag und die vorhandene Bedienung sind vor einem Entwurf für sichere Kandidatenbereiche und Übernahme erneut am finalen Stand abzugleichen. Vor einer einstimmigen Annahme darf ein Kandidat nicht durch bloße Benennung zu einer isolierten Änderung werden.

Ein möglicher Zielablauf ist: Kandidat herstellen → unabhängige Prüfungen und Ressorturteile an genau diesem Kandidaten → offene Einwände reparieren → aktualisierte Fassung erneut prüfen → geschützte Übernahme mit erneuter Bindungsprüfung → Verifikation des übernommenen Standes. Wie das zur heutigen Apply-Reihenfolge passt und welche zusätzlichen Staging-/Writer-Garantien nötig sind, bleibt eine technische Designfrage.

### 6. Audit und nächste Arbeit

Ein Abschlussbericht nennt den gebundenen Stand, akzeptiertes Soll, geänderte Artefakte, Ressorturteile, Verifikation, offenen Backlog und Grenzen der Evidenz.

Eine übergeordnete Ebene prüft, ob alle deklarierten betroffenen Bereiche, Integrationspflichten und erforderlichen Berichte enthalten sind. Sie nimmt ihre Zusammensetzungspflicht selbst wahr.

Ein offener Konflikt blockiert die davon abhängige Arbeit. Unabhängige Backlog-Arbeit kann fortfahren. Das Ergebnis eines Durchlaufs ist an seine unveränderliche Arbeitsgrundlage gebunden; spätere Repository-Änderungen starten einen neuen Abgleich.

## Weiterentwicklung des Sollmodells

Das Modell braucht Rückkopplung aus Umsetzung und Betrieb, ohne dass jeder Fehler als neue Regel oder jeder Regelvorschlag manuelle Zustimmung verlangt.

Auslöser kann eine konkrete unerfüllbare Vorgabe, ein wiederkehrendes Defektmuster, ein nachgewiesener Widerspruch zwischen Bereichen, eine fehlende Zuständigkeit oder eine belegte Architekturverbesserung sein. Einzelne Beobachtungen werden zunächst als Evidenz behandelt.

Jeder Vorschlag ordnet den Anlass ein:

| Klasse | Reaktion |
|---|---|
| Implementierungsfehler | Klare erfüllbare Vorgabe verletzt; Umsetzung reparieren, Soll beibehalten. |
| Modelllücke | Eine relevante Anforderung ist nicht aus dem geltenden Soll ableitbar oder die Regeln widersprechen sich; Änderungsvorschlag mit Alternativen und Folgen erstellen. |
| Optimierung | Vorgaben sind erfüllbar, eine Lösung ist aber effizienter oder einfacher; innerhalb technischer Freiheit umsetzen oder als priorisierbaren Backlog-Eintrag melden. |
| Nicht delegierter Zielkonflikt | Die Auswahl verschiebt Ziele, Grenzen, Datenverhalten, Kompatibilität oder Prioritäten außerhalb der bestehenden Delegation; konkrete Entscheidung nach oben geben. |

Delegierte Modell-Evolution darf echte Architekturentscheidungen umfassen. Sie ist nicht auf Textpflege oder Umbenennung beschränkt. Der Vorschlag muss auf ein bereits akzeptiertes Ziel und eine bestehende ausreichende Delegation zurückführbar sein.

Das Verfahren darf neue Implementierungsbelege nicht als alleinige Quelle einer neuen Absicht verwenden. Agenten können weder ihre eigene Änderungsbefugnis erweitern noch eine misslungene Umsetzung durch Abschwächung des Maßstabs als konform erklären.

Modelländerung und Umsetzung werden gemeinsam nachverfolgt, aber getrennt beurteilt: Ein unabhängiger Prüfer bewertet den Regelvorschlag gegen übergeordnete Ziele; die Ausführung setzt erst ein gültig angenommenes neues Soll um. Die bisherige Nutzerforderung, dass jedes ausgewählte Ressort Änderungen ausdrücklich billigt, bleibt als Gestaltungsanforderung erhalten. Sie legt jedoch noch nicht fest, welches Kabinett über Verfassungs-, Mandats- oder Ressortänderungen entscheidet. Dieses Gesetzgebungsverfahren muss ausdrücklich beschlossen werden; bis dahin ist kein autonomes Modell-Update als freigegeben anzunehmen. Ein Teilversuch mit Modell-Evolution setzt eine explizite, ausreichende Delegation und ein zuvor festgelegtes Annahmeverfahren voraus.

### Beispiel einer delegierten Architekturentscheidung

Die Verfassung verlangt, dass jede Geschäftsregel einen fachlichen Eigentümer hat, und delegiert Modulgrenzen an die Architekturverantwortung. Bei der Arbeit zeigt sich mit Tests und Änderungsverlauf, dass zwei Module dieselbe Regel unabhängig ändern und auseinanderlaufen.

Wenn eine ausreichende Delegation und ein gültiges Annahmeverfahren diese Entscheidungsklasse umfassen, können die Agenten innerhalb dieses Rahmens entscheiden, welches Modul der fachliche Eigentümer wird, die gemeinsame Regel dort bündeln und das andere Modul über eine vorhandene Schnittstelle anbinden. Das ist eine echte Architekturentscheidung, keine bloße Dokumentationspflege. Der Prüfer muss belegen, dass die Änderung das akzeptierte Ziel wahrt. Im Kabinettsmodell müssen die ausgewählten Ministerien der finalen Kandidatenfassung ausdrücklich zustimmen; wie diese Regel auf Verfassungs- und Mandatsänderungen anzuwenden ist, bleibt gesondert zu beschließen.

Wäre die Entscheidung zugleich, unabhängige Bereitstellung zugunsten einer gemeinsamen Laufzeit aufzugeben, und gäbe es dafür keine geltende Priorität oder Delegation, müsste diese konkrete Produktabwägung nach oben. Bis dahin können andere unabhängige Arbeiten weitergehen.

## Ressortkonflikt und Gericht: kleinstes Erstmodell

Die Managementhierarchie und Gewaltenteilung können unterschiedliche Fragen abdecken: Hierarchie verteilt Ziele und Entscheidungsfreiheit und transportiert Berichte; eine Legislative setzt geltende Regeln, Exekutive führt sie aus und Judikative legt sie im Streitfall aus.

Die kleinste kohärente erste Form braucht noch kein eigenes Gericht. Ein Ressortkonflikt wird anhand geltender Regeln und delegierter Prioritäten gelöst. Bleibt eine echte Auslegungsfrage, Eskalation an eine dafür bestimmte unabhängige Entscheidungsinstanz erwägen. Ist eine neue Priorität, ein neues Ziel oder eine Regeländerung nötig, entscheidet die hierfür autorisierte übergeordnete Instanz.

Ein späteres Gericht dürfte nur geltende Regeln auf den konkreten Fall anwenden und den zulässigen Handlungsspielraum bestimmen. Es dürfte keine neue Regel oder politische Priorität setzen. Bei fehlender Rechtsgrundlage würde es die Frage an die gesetzgebende Instanz zurückgeben.

**Offene Spannung:** Ein bindendes Gerichtsurteil könnte einen Einwand als nicht durch geltendes Recht gestützt beurteilen; die zuvor erklärte Einstimmigkeitsregel verlangt dennoch Zustimmung jedes ausgewählten Ministeriums. Ein Urteil darf diese Zustimmung nicht stillschweigend ersetzen. Bis ein ausdrücklicher Entscheidungsweg vereinbart ist, bleibt der Kandidat offen, wenn ein Ressort nicht zustimmt. Für den Erstentwurf wird deshalb kein gerichtlicher Überstimmungsmechanismus angenommen.

## Begrenzter Betrieb und Wiederaufnahme

Der dauerhafte Laufzeitprozess nimmt Arbeit nur innerhalb gesetzter Ressourcen- und Delegationsgrenzen auf. Arbeit kann nach Priorität und Abhängigkeiten in begrenzte Durchläufe zerlegt werden.

Jeder Durchlauf begrenzt Parallelität, Agentenaufrufe, Zeit, Wiederholungen, Kontextgröße und externe Kosten. Überschreitungen erzeugen Bericht und Wiederaufnahmezustand, keine unbegrenzte Schleife.

Ein Checkpoint speichert festen Quellstand, Modellstand, Kandidaten, Auftrag, Abhängigkeiten, verbrauchte Ressourcen, Einwände, Stimmen und offene nächste Schritte. Wiederaufnahme prüft Quell-, Modell-, Mandats-, Werkzeug- und Evidenzfrische, bevor Arbeit fortgesetzt wird.

Veraltete Kandidaten oder Stimmen werden nicht stillschweigend wiederverwendet. Ein Ressort kann Arbeit pausieren oder ein Ergebnis eskalieren; dies blockiert nur abhängige Arbeiten. Der Runner wird idle, wenn kein ausführbarer Auftrag vorliegt oder ein benötigter Entscheid aussteht.

Kontinuierliche Beobachtung ist eine spätere Laufzeitfähigkeit, nicht Voraussetzung, um den endlichen Arbeitszyklus zu definieren. Kubernetes, ein allgemeiner Pluginloader und automatische Installation zusätzlicher Werkzeuge folgen nicht aus diesem Betriebsmodell.

## Prüffälle und Erfolgskriterien

Die Erprobung erfolgt stufenweise gemäß dem [Validierungsplan](delegated-engineering-validation-plan.md), beginnend mit dem finalen Architect-Kandidaten und einem begrenzten vertikalen Lauf. Die vollständige Szenarioliste ist kein Vorbedingungs-Harness für den ersten Nutzen. Je nach Ergebnissen werden Modell-Evolution, Ressortkonflikt, Parallelität und Wiederaufnahme in späteren Stufen geprüft.

Bewertet werden übersehene Pflichten, falsche Einwände, Integrationsfehler, Modelländerungen ohne gültige Grundlage, unnötige Eskalationen, menschliche Koordination, Wiederaufnahmequalität und verbleibende Unsicherheit. Grünmeldungen einzelner Tests oder formale Einstimmigkeit allein belegen keine allgemeine autonome Zuverlässigkeit.

## Baseline und belegbare Grenzen

Die technische Vergleichsbasis ist der unveränderliche Commit `1ea5c76f55526fc4d721e865885436153f48b497`. Der finale Abschlusslauf und die einzige Dokumentationsänderung gegenüber der Vorbereitung sind im [Abschlussabgleich](architect-government-checkpoint.md) festgehalten.

Für die zugrunde liegende Projektions- und Controller-Architektur sind insbesondere die quellgebundenen Beschreibungen und Verträge aus dieser Revision zu prüfen:

- `docs/design/standard-operating-model.md` beschreibt den endlichen projektionsorientierten Ablauf, Abschlussaudit und kontinuierlichen Runner als späteren Schritt.
- `internal/host/canonical_controller.go` enthält Controllerkonfiguration, Projektion-gebundene Assurance-Scope-Struktur und Laufdatentypen.
- `internal/host/canonical_controller_execution.go` beschreibt gebundene Kandidatenerstellung und Executor-Aufrufe.
- `internal/host/canonical_controller_verification.go` beschreibt getrennte aktuelle Verifikation und Evidenzbindung.
- `internal/host/canonical_controller_audit.go` beschreibt read-only Abschluss für deklarierte Projektionen und aktuelle Evidenz.
- `internal/host/agentexec/types.go` beschreibt Rollen, Aufträge, Kandidatendateien, Beobachtungen und Laufbelege.
- `internal/host/records/records.go` und `internal/host/recordstore/` beschreiben Materialisierungs-, Verifikations- und Betriebsdatensätze.
- `docs/usage.md` und `docs/canonical-projections.md` beschreiben die Bediengrenzen der Quellalpha einschließlich Apply.

Diese Quellen belegen technische Verträge nur für den jeweils untersuchten Commit und den erklärten Umfang. Sie belegen weder Government-Verhalten noch eine allgemeine autonome Entwicklungsmaschine, Prozessisolation, menschliche Akzeptanz oder fachliche Richtigkeit. Der Abgleich mit dem finalen Architect-Stand ist vor weiteren Designentscheidungen erforderlich.

## Offene Entscheidungen nach dem Architect-Checkpoint

- Wie weit trägt der finale Architect-Kandidat das zentrale Ziel kontinuierlicher, weitgehend autonomer Entwicklungsarbeit, und welche Government-Produktform ist dafür angemessen?
- Welche Teile der Verfassung dürfen innerhalb welcher allgemein formulierten Delegation automatisch weiterentwickelt werden?
- Wie bindet der Betriebsprozess Ministeriumsberichte und ausdrückliche Einstimmigkeit an den exakten finalen Kandidaten?
- Wie werden Kandidaten außerhalb der aktiven Projektdateien vorbereitet und anschließend geschützt übernommen, ohne `Apply` als Isolation zu missverstehen?
- Wie werden ein bindendes Gerichtsurteil und die Einstimmigkeit jedes ausgewählten Ministeriums miteinander vereinbart?
- Welche Berichtssicht ist standardmäßig sinnvoll und wie bleiben unterdrückte Meldungen dennoch zuverlässig erfasst?
- Welche Grenzen für Kosten, Laufzeit, Parallelität, Wiederholungen und menschliche Eskalation machen den Betrieb benutzbar?

Diese Fragen sind Gegenstand der späteren Evaluation und Entscheidung. Dieser Entwurf beantwortet sie nicht stillschweigend und autorisiert keine Implementierung.

# Markitect Konzepte für kanonisches Modell und delegierte Entwicklung

Markitect soll Menschen ermöglichen, das gewünschte Projekt an einer maßgeblichen Quelle zu beschreiben und seine Realisierungen durch delegierte Arbeit anpassen und prüfen zu lassen. Explizite Struktur, Beziehungen und Verantwortlichkeiten sollen verständliche Software und verlässlichere Agentenarbeit ermöglichen. Spezifikation und Ordnung sollen einen dauerhaften praktischen Nutzen haben.

**Status:** Konzeptaufnahme vom 9. Oktober 2026 aus der [Nutzerbeschreibung](user-description-20261009.md). Die Einträge unterscheiden Nutzerabsicht, berichtete Erfahrung, Wirkungshypothese und eigene Interpretation. Sie begründen keine Priorisierung oder Umsetzung. In dieser Aufnahme werden keine implementierten Fähigkeiten festgestellt.

## Herkunft und Aussagearten

Die Herkunftsangaben beziehen sich auf die fünf Absätze der wörtlichen Beschreibung: P1 schildert die Erfahrung mit vergessenen Dateien und die Frage nach verständlicher Software; P2 beschreibt kanonisches Soll und den Aufwand manueller Kontrolle; P3 beschreibt Typsystem, Compiler, Zuordnung und Änderungsfolgen; P4 beschreibt Organisation, Agenten, Modelle, Prüfung und Cleanup; P5 beschreibt Grenzen und erhofften Produktnutzen. K bezeichnet die separat übergebene vorangegangene Klarstellung. A bezeichnet den Einrichtungsauftrag für den Chat, der insbesondere eine rekursive Managerhierarchie festzuhalten verlangt.

| Aussageart | Bedeutung für diese Aufnahme |
|---|---|
| Nutzerabsicht | Gewünschtes Produktziel oder gewünschter Mechanismus aus P1 bis P5 beziehungsweise K |
| Erfahrung | Vom Nutzer berichtete Beobachtung; keine verallgemeinerte Messung |
| Hypothese | Erwarteter Zusammenhang, dessen Wirkung offen bleibt |
| Eigene Interpretation | Aufbereitung oder Ergänzung des Concepts-Chats; keine zugeschriebene Nutzerentscheidung |
| Entscheidung | Eine ausdrücklich getroffene Festlegung mit benannter Herkunft; die Beschreibung liefert keine konkrete Architekturwahl oder Roadmap-Priorisierung |
| Implementierte Fähigkeit | Verhalten eines geprüften Quellstands; aus dieser Beschreibung allein nicht feststellbar |

Die kanonische Quelle bedeutet eine maßgebliche Verantwortung pro Modellfakt. **Eigene Interpretation:** Das verlangt keine einzige physische Datei. Die Quelle kann geordnet aus mehreren Dateien bestehen, ohne dass derselbe Fakt mehrere gleichberechtigte Eigentümer bekommt.

Die Felder zu Annahmen, Grenzen, offenen Fragen und Beziehungen enthalten analytische Ergänzungen des Concepts-Chats, soweit nicht ausdrücklich auf eine Aussage des Nutzers verwiesen wird. „Mechanismus“ beschreibt das vorgesehene Konzept; „erhoffte Wirkung“ bleibt eine Erwartung. Beide Felder sind keine Fähigkeitsnachweise.

## Konzeptnotizen

### C01 Eine kanonische Quelle

- **Idee und Problem:** Der Mensch pflegt das maßgebliche Modell. Damit soll das Vergessen von Änderungen bei manueller Pflege vieler Stellen vermieden werden.
- **Mechanismus:** Eine Änderung am Soll wird Ausgangspunkt für Anpassungen abgeleiteter Realisierungen und ihre Prüfung. Diese Anpassungen können delegierte Arbeit erfordern; reine Textgenerierung ist nicht vorausgesetzt.
- **Annahmen:** Es ist erkennbar, welche Inhalte verbindliches Soll und welche Realisierungen sind. Die zuständige Person kann Änderungen am Soll bestimmen.
- **Erhoffte Wirkung:** Weniger konkurrierende Aussagen und weniger manuelle Synchronisierung.
- **Herkunft und Status:** Nutzerabsicht aus K und P2; Wirkungshypothese aus der Erfahrung in P1.
- **Grenzen und Gegenbelege:** Ein falsches oder unvollständiges Soll kann konsistent umgesetzt werden und dennoch das falsche Produkt ergeben. Mehrere versteckte Quellen derselben Vorgabe würden den Mechanismus unterlaufen.
- **Offene Frage:** Wie werden widersprüchliche bestehende Aussagen auf eine maßgebliche Quelle zurückgeführt?
- **Beziehungen:** Grundlage für C02, C04, C05 und C16.

### C02 Soll Welt und Realisierung

- **Idee und Problem:** Verständlich sein soll sowohl, was Software tut, als auch, ob sie das Gewollte tut. Dateien allein beantworten die zweite Frage nicht.
- **Mechanismus:** Das kanonische Modell beschreibt das Soll. Das Repository liefert den beobachteten Ist-Zustand. Prüfung beurteilt ihre Übereinstimmung anhand benannter Verpflichtungen.
- **Annahmen:** Vorgaben sind ausreichend klar, um ihre Erfüllung untersuchen zu können; Beobachtungen sind dem betrachteten Stand zugeordnet.
- **Erhoffte Wirkung:** Abweichungen werden erklärbar, ohne eine vorhandene Implementierung automatisch zum Maßstab zu machen.
- **Herkunft und Status:** Nutzerabsicht aus P1, P2 und K. Die ausdrückliche Dreiteilung ist eine eigene Interpretation im Sinne des Einrichtungsauftrags.
- **Grenzen und Gegenbelege:** Übereinstimmung mit dem Modell beweist weder Vollständigkeit des Modells noch tatsächliche Nutzerzufriedenheit. **Eigene Interpretation:** Mehrere unterschiedliche Realisierungen können dasselbe Soll erfüllen; Entsprechung muss keine eindeutige Ausgabe bedeuten.
- **Offene Frage:** Welche Freiheiten bleiben der Realisierung, und welche Anforderungen erfordern Laufzeitbeobachtung?
- **Beziehungen:** C03 prüft Struktur; C11 und C12 untersuchen Realisierung; C14 betrifft Vertrauen.

### C03 Typisiertes Modell und frühe strukturelle Prüfung

- **Idee und Problem:** Modellfehler sollen vor der Ausführung auffallen, statt erst während delegierter Arbeit entdeckt zu werden.
- **Mechanismus:** Modellgegenstände und Beziehungen erhalten definierte Typen. Ein Compiler prüft, ob das Modell strukturell zusammenpasst. **Eigene Interpretation:** Denkbare Prüfgegenstände sind fehlende Referenzen, unzulässige Beziehungen und widersprüchliche strukturelle Verträge; die genaue Sprache bleibt offen.
- **Annahmen:** Relevante Strukturregeln sind ausdrückbar und deterministisch prüfbar.
- **Erhoffte Wirkung:** Früheres Feedback und weniger Arbeit auf einer ungültigen Grundlage.
- **Herkunft und Status:** Vorgesehener Mechanismus aus P3, kein Nachweis eines vorhandenen Compilers.
- **Grenzen und Gegenbelege:** „Kompilierbar“ beweist keine gute Spezifikation oder semantische Richtigkeit. Ein Fehler außerhalb des Typsystems kann unentdeckt bleiben. Die Compiler-Analogie legt keine allgemeine Lösbarkeit fachlicher Widersprüche fest.
- **Offene Frage:** Welche Fehlerklassen gehören in die Strukturprüfung, welche in fachliche Prüfung?
- **Beziehungen:** Stützt C04 und C05; ersetzt C11, C12 oder menschliche Beurteilung nicht.

### C04 Was gehört wozu

- **Idee und Problem:** Sichtbar werden soll, welche Dateien welche Teile des Modells realisieren. Relevante Dateien sollen nicht allein vom Erinnerungsvermögen eines Agenten abhängen.
- **Mechanismus:** Explizite Zuordnungen verbinden Modellgegenstände und Projektartefakte. **Eigene Interpretation:** Die Zuordnung sollte in beiden Richtungen untersuchbar sein; eine Datei kann mehrere Gegenstände realisieren und ein Gegenstand mehrere Dateien betreffen.
- **Annahmen:** Zuordnungen werden gepflegt und die notwendige Granularität lässt sich praktisch handhaben.
- **Erhoffte Wirkung:** Verantwortliche und betroffene Realisierungen lassen sich gezielter finden.
- **Herkunft und Status:** Nutzerabsicht und Mechanismus aus P3; die bidirektionale Mehrfachzuordnung ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Eine Deklaration beweist ihre inhaltliche Richtigkeit nicht. Fehlende, veraltete oder zu grobe Zuordnungen können gerade die gesuchten Dateien auslassen.
- **Offene Frage:** Wie werden noch nicht zugeordnete Dateien und Verpflichtungen ohne konkrete Implementierungsdatei sichtbar?
- **Beziehungen:** Liefert die Grundlage für C05; unterstützt C07, C12 und C13.

### C05 Explizite Beziehungen und Änderungsfolgen

- **Idee und Problem:** Nach einer Modelländerung sollen relevante Folgen sichtbar werden, damit Regeln und betroffene Dateien nicht vergessen werden.
- **Mechanismus:** Eine Diff-Analyse verbindet die Änderung mit expliziten Beziehungen und daraus folgenden Arbeits- und Prüfpflichten. **Eigene Interpretation:** Alte und neue Beziehungen sind relevant, damit entfernte Verbindungen nicht ihre bisherigen Verbraucher aus der Untersuchung verschwinden lassen.
- **Annahmen:** Beziehungen haben klare Bedeutung und erfassen die für eine Änderung wesentlichen Abhängigkeiten.
- **Erhoffte Wirkung:** Der „Blast Radius“, also der mögliche Wirkungsbereich einer Änderung, wird nachvollziehbar und in geeigneten Fällen kleiner.
- **Herkunft und Status:** Mechanismus und Wirkungshypothese aus P3 und P4; Betrachtung beider Stände ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Indirekte oder dynamische Abhängigkeiten können fehlen. Ein kleiner berechneter Bereich kann unvollständig sein. **Eigene Interpretation:** Unsicherheit sollte die Untersuchung erweitern können. Nicht jede Beziehung bedeutet dieselbe Änderungswirkung.
- **Offene Frage:** Wie unterscheidet man fachliche Beziehung, Prüfpflicht, Kontextbedarf und Ausführungsabhängigkeit?
- **Beziehungen:** C04 liefert Zuordnungen; C08 und C09 nutzen den Umfang; C12 ergänzt lokale Prüfung.

### C06 Namespaces und Abstraktionsebenen

- **Idee und Problem:** Das Modell soll wie geordnete Dokumentation oder Code gruppiert werden, damit seine Teile verständlich und handhabbar bleiben.
- **Mechanismus:** Ordner beziehungsweise Namespaces ordnen Gegenstände nach Bereichen und Abstraktionsebenen.
- **Annahmen:** Die gewählte Gruppierung bildet relevante fachliche Grenzen ausreichend gut ab.
- **Erhoffte Wirkung:** Orientierung und Grundlage für passende Verantwortungsbereiche.
- **Herkunft und Status:** Nutzerabsicht und organisatorische Analogie aus P4.
- **Grenzen und Gegenbelege:** Eine Ordnerstruktur allein erklärt weder Verantwortung noch Beziehungen. Querschnittspflichten können mehrere Bereiche betreffen. Ein technischer Namespace muss keine fachliche Hierarchie sein.
- **Offene Frage:** Welche Organisationsinformationen sind ausdrücklich modelliert, welche werden nach einer noch zu bestimmenden Regel abgeleitet?
- **Beziehungen:** Hilft C07 und C08; ist von den Abhängigkeiten in C05 zu unterscheiden.

### C07 Verantwortungsbereiche

- **Idee und Problem:** Jeder Bereich soll einen erkennbaren Auftrag erhalten, damit Prüf- und Änderungsarbeit nicht zwischen Zuständigkeiten verloren geht.
- **Mechanismus:** Modellbereiche werden Verantwortlichen und begrenzten Zielen zugeordnet. **Eigene Interpretation:** Fachliche Zuständigkeit, Befugnis zur Entscheidung und Verantwortung für einen konkreten Schreibvorgang sind getrennte Fragen.
- **Annahmen:** Grenzen, Schnittstellen und übergreifende Pflichten sind verständlich; eine Lücke kann sichtbar gemacht werden.
- **Erhoffte Wirkung:** Nachvollziehbare Abdeckung und weniger doppelte oder ausbleibende Arbeit.
- **Herkunft und Status:** Nutzerabsicht aus P4; Trennung der Verantwortungsarten ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Exklusive lokale Zuständigkeit kann Querschnittsprobleme verdecken. Mehrere verantwortliche Bereiche können widersprüchliche Vorschläge erzeugen.
- **Offene Frage:** Wer klärt Zuständigkeitslücken und Konflikte zwischen Bereichen?
- **Beziehungen:** Nutzt C04 und C06; trägt C08, C09 und C11.

### C08 Rekursive Managerhierarchie

- **Idee und Problem:** Umfangreiche Arbeit soll in einer Hierarchie begrenzter Verantwortungsbereiche koordiniert werden.
- **Mechanismus:** **Eigene Interpretation der gewünschten Rekursion:** Ein Manager zerlegt seinen Auftrag, delegiert Teilziele und integriert geprüfte Ergebnisse. Untergeordnete Manager können denselben Ablauf für ihren Bereich wiederholen. Jede Ebene prüft auch ihr Zusammenspiel.
- **Annahmen:** Ziele lassen sich sinnvoll zerlegen; Schnittstellen und Integrationspflichten bleiben erhalten; Rekursion endet bei handhabbaren Aufträgen.
- **Erhoffte Wirkung:** Große Vorhaben werden mit begrenztem Kontext bearbeitbar, ohne ihre Gesamtverantwortung aufzugeben.
- **Herkunft und Status:** Managementhierarchie aus P4; Rekursion ausdrücklich im Auftrag A genannt. Der beschriebene Ablauf ist eigene Interpretation, keine aus P4 zitierte Detailentscheidung.
- **Grenzen und Gegenbelege:** Tiefe Hierarchien können Kontextverlust, Koordinationskosten und falsche Zusammenfassungen verstärken. Ein erfolgreicher Teilauftrag beweist noch keine korrekte Integration.
- **Offene Frage:** Wann lohnt weitere Zerlegung, und wer entscheidet bei Konflikten oder fehlender Entscheidungsbefugnis?
- **Beziehungen:** Organisiert C07, C09 und C11; C12 prüft die Gesamtkomposition.

### C09 Begrenzte Agentenziele

- **Idee und Problem:** Ein Agent soll nicht gleichzeitig alle Interessen und Dateien eines komplexen Projekts berücksichtigen müssen.
- **Mechanismus:** Agenten erhalten klare Ziele, Interessen und Verantwortungsbereiche. **Eigene Interpretation:** Begrenzung braucht trotzdem ausreichenden Kontext über angrenzende Verträge, übergeordnete Regeln und die betrachtete Änderung.
- **Annahmen:** Teilziele sind verständlich und gemeinsam mit dem Gesamtziel vereinbar.
- **Erhoffte Wirkung:** Weniger Überforderung und gezieltere Untersuchung relevanter Stellen.
- **Herkunft und Status:** Nutzerabsicht aus P4, motiviert durch P1; die Kontextbedingung ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Zu wenig Kontext kann einen lokal plausiblen, insgesamt falschen Vorschlag erzeugen. Viele Agenten allein sichern keine Vollständigkeit.
- **Offene Frage:** Woran erkennt man, dass ein Ziel zu groß oder sein Kontext zu klein ist?
- **Beziehungen:** Nutzt C05 und C07; ermöglicht die Hypothese C10 und getrennte Rollen in C11.

### C10 Günstigere Modelle als Hypothese

- **Idee und Problem:** Gute Agentenarbeit soll weniger stark davon abhängen, immer die teuersten Modelle einzusetzen.
- **Mechanismus:** Klare Teilziele und aufbereiteter Kontext sollen Aufgaben für günstigere Modelle besser handhabbar machen.
- **Annahmen:** Diese Modelle sind für die jeweiligen Aufgaben hinreichend fähig; Zerlegung ersetzt erforderliches Verständnis nicht; zusätzliche Aufrufe und Nacharbeit bleiben wirtschaftlich.
- **Erhoffte Wirkung:** Geringere Gesamtkosten bei ausreichender Qualität.
- **Herkunft und Status:** Explizite Wirkungshypothese aus P4. „Mittlerweile ähnlich fähig“ ist die Einschätzung in der Beschreibung, keine hier verifizierte Markt- oder Leistungsbehauptung.
- **Grenzen und Gegenbelege:** Mehr Koordination, Prüfaufrufe und Reparaturen können die Ersparnis aufheben. Schwierige Integration kann weiterhin hohe Modellfähigkeit erfordern.
- **Offene Frage:** Unter welchen vergleichbaren Aufgaben und Qualitätsanforderungen sinken die Gesamtkosten tatsächlich?
- **Beziehungen:** Hängt von C08 und C09 ab; C11 und C12 verursachen Kosten und können Qualität schützen. Kein Studienstart folgt aus dieser Notiz.

### C11 Unabhängige Prüfung und Vieraugenprinzip

- **Idee und Problem:** Vertrauen soll weniger davon abhängen, dass der Mensch jede Änderung mühsam selbst rekonstruiert oder ein ausführender Agent sich selbst bestätigt.
- **Mechanismus:** Andere Agenten untersuchen die Änderung gegen das geltende Soll und relevante Regeln. **Eigene Interpretation:** Prüfer müssen denselben konkreten Kandidaten beurteilen und Einwände unabhängig bilden können.
- **Annahmen:** Prüfrollen haben ausreichenden Kontext und teilen nicht zwangsläufig alle Fehler der Ausführung.
- **Erhoffte Wirkung:** Mehr übersehene Stellen und Regelverletzungen werden entdeckt.
- **Herkunft und Status:** Vieraugenprinzip und Subagenten aus P4, Kontrollproblem aus P2. Bedingungen der Unabhängigkeit sind eigene Interpretation.
- **Grenzen und Gegenbelege:** Mehrere Agenten können dieselbe falsche Annahme übernehmen. Rollennamen, Zustimmung oder überzeugende Berichte allein beweisen keine unabhängige Prüfung und keine menschliche Akzeptanz.
- **Offene Frage:** Welche Trennung von Ausführung, Kontext, Urteil und Entscheidung genügt für die gewünschte Unabhängigkeit?
- **Beziehungen:** Ergänzt C03, C08 und C09; liefert Befunde für C12 und C14.

### C12 Gesamtprüfung

- **Idee und Problem:** Lokale Prüfungen sollen durch einen Abgleich aller Bereiche mit dem kanonischen Modell ergänzt werden.
- **Mechanismus:** Ein Gesamtcheck betrachtet das Soll, seine Realisierungen und das Zusammenspiel der Bereiche. **Eigene Interpretation:** Er sollte unbekannte oder ungeprüfte Bereiche von verifizierten Bereichen unterscheiden.
- **Annahmen:** Prüfgegenstände und Verpflichtungen sind ausreichend erfasst; übergreifende Regeln können untersucht werden.
- **Erhoffte Wirkung:** Bereichsübergreifende Widersprüche und Lücken fallen auf, die einzelne Teilprüfungen verfehlen.
- **Herkunft und Status:** Gewünschter Mechanismus aus P4; Unterscheidung der Abdeckung ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Ein Gesamtcheck ist nur so vollständig wie seine Eingaben und Prüfregeln. Die Summe grüner Teilprüfungen beweist keine korrekte Komposition. Aufwand und Aussagekraft bleiben offen.
- **Offene Frage:** Was bedeutet „alle Bereiche“ bei unmodellierten Dateien und externem Laufzeitverhalten?
- **Beziehungen:** Ergänzt C05 und C11; stützt C13 und C14.

### C13 Cleanup und Refactoring

- **Idee und Problem:** Aufräumen und strukturelle Umbauten sollen verlässlicher werden, ohne relevante Verpflichtungen zu verlieren.
- **Mechanismus:** Modell, Zuordnungen und Änderungsfolgen bestimmen, welche Regeln ein Umbau bewahren und welche Realisierungen er anpassen muss; unabhängige und übergreifende Prüfungen beurteilen das Ergebnis.
- **Annahmen:** Bewahrte Eigenschaften sind erkennbar; Zuordnungen ändern sich nachvollziehbar mit dem Umbau.
- **Erhoffte Wirkung:** Weniger vergessene Nebenfolgen und geringere Hürde für nachhaltige Ordnung.
- **Herkunft und Status:** Explizite Hoffnung aus P4.
- **Grenzen und Gegenbelege:** Ein formal unverändertes Soll kann Änderungen an Laufzeitverhalten übersehen. Unmodellierte externe Verbraucher oder implizite Verträge können betroffen sein.
- **Offene Frage:** Welche Eigenschaften müssen bei welchem Refactoring gleich bleiben, und wie wird das belegt?
- **Beziehungen:** Verwendet C04, C05, C11 und C12; trägt zum dauerhaften Nutzen in C16 bei.

### C14 Vertrauen durch überprüfbare Methodik

- **Idee und Problem:** Größeres Vertrauen in Coding Agents soll aus nachvollziehbarer Systematik entstehen; manuelle Vollkontrolle bremst die gewünschte Arbeitsweise.
- **Mechanismus:** Klar benannte Vorgaben, Zuständigkeiten, Änderungsfolgen und getrennte Prüfungen machen die Entstehung eines Ergebnisses überprüfbar. **Eigene Interpretation:** Ein Befund braucht einen bestimmten betrachteten Stand und erkennbare Grenzen seiner Aussage.
- **Annahmen:** Die Methodik wird eingehalten und ihre Nachweise betreffen die tatsächliche Änderung.
- **Erhoffte Wirkung:** Weniger notwendige Detailaufsicht bei tragfähigem Vertrauen.
- **Herkunft und Status:** Produktziel aus P2 und P5; Wirkungszusammenhang ist eine Hypothese. Die Bindung von Befunden an einen Stand ist eigene Interpretation.
- **Grenzen und Gegenbelege:** Viele Berichte können falsche Sicherheit erzeugen. Methodentreue allein beweist weder fachliche Richtigkeit noch bessere Arbeitsergebnisse.
- **Offene Frage:** Welche Nachweise helfen Menschen tatsächlich, eine Änderung zu beurteilen, ohne erneut alles zu lesen?
- **Beziehungen:** Baut auf C02, C03, C05, C11 und C12 auf; beeinflusst C16.

### C15 Spec Driven AI First Development

- **Idee und Problem:** Spezifikation, Struktur und Agentenarbeit sollen eine zusammenhängende Entwicklungsweise bilden, statt erst nachträglich Ordnung herstellen zu müssen.
- **Mechanismus:** Menschliche Absicht wird als Soll beschrieben; Agenten bearbeiten daraus abgeleitete Ziele und prüfen die Realisierung an den Vorgaben.
- **Annahmen:** Das Soll lässt sich verständlich pflegen und lässt genügend Implementierungsfreiheit; relevante Änderungen erreichen ihre Realisierungen.
- **Erhoffte Wirkung:** Saubere, einheitliche Repositories und höhere Arbeitsqualität.
- **Herkunft und Status:** Vom Nutzer benannter Ansatz aus P5, verbunden mit K und P2 bis P4. Daraus folgt kein festgelegtes Framework oder Anbieterprodukt.
- **Grenzen und Gegenbelege:** Schlechte Spezifikation, Organisation oder Missachtung der Methode bleiben möglich, wie P5 ausdrücklich anerkennt. Mehr Spezifikation kann Arbeit behindern, wenn sie ohne Nutzen zu viele Details festlegt.
- **Offene Frage:** Welche Mindeststruktur bringt Nutzen, und welche Details sollten freie Realisierungsentscheidungen bleiben?
- **Beziehungen:** Verbindet C01 bis C14 als Arbeitsweise; C16 formuliert ihren erhofften dauerhaften Nutzen.

### C16 Spezifikation und Ordnung als dauerhafter Nutzen

- **Idee und Problem:** Ordentliche Definition und Struktur sollen sich lohnen, statt dauerhaft eine „Aufräumhölle“ zu erzeugen.
- **Mechanismus:** Das gepflegte Soll und seine Beziehungen sollen bei späteren Änderungen erneut Kontext, Zuständigkeiten und Prüfpflichten liefern; der einmalige Ordnungsaufwand soll wiederholt nutzbar bleiben.
- **Annahmen:** Modellpflege bleibt praktikabel und der wiederkehrende Nutzen überwiegt Erstellung, Pflege und Prüfung.
- **Erhoffte Wirkung:** Weniger wiederholte Orientierung und Synchronisierung, verständlichere Systeme und nachhaltige Qualität.
- **Herkunft und Status:** Produktziel aus P5, motiviert durch P1 und P2. Wiederverwendbarer Nutzen und wirtschaftliche Bilanz sind eine eigene Interpretation des erhofften Zusammenhangs.
- **Grenzen und Gegenbelege:** Veraltete Zuordnungen, doppelte Beschreibungen oder übermäßige Formalisierung können eine neue Pflegehölle schaffen. Ein geordnetes Modell ohne praktischen Nutzen würde das Ziel verfehlen.
- **Offene Frage:** Woran erkennt der Nutzer im Alltag, dass Modellpflege Arbeit spart und Verständnis verbessert?
- **Beziehungen:** Ergebnisziel für C01, C13, C14 und C15; beeinflusst die sinnvolle Granularität in C04 und C09.

## Ergänzung zur dauerhaften Wirksamkeit der Spezifikation

**Herkunft und Status:** [Diskussionsimpuls vom 9. Oktober 2026](discussion-impulse-20261009-01.md), abgeleitet vom ursprünglichen Produktchat. C17 bis C19 erfassen dessen Hypothesen und offene Fragen; sie sind keine direkte Nutzerentscheidung und kein entschiedener Produktvertrag. Die Aufbereitung und analytischen Ergänzungen stammen von Concepts. C01 bis C16 und ihr Ursprung bleiben erhalten.

### C17 Neue Absicht und Erhalt geltender Verpflichtungen

- **Idee und Problem:** Spezifikation soll bei späteren Änderungen wirksam bleiben. Die neue Absicht darf die Untersuchung weiterhin geltender Verpflichtungen nicht verdrängen.
- **Mechanismus:** Nach der Ausgangshypothese des ursprünglichen Produktchats erhält jede betroffene modellierte Verpflichtung eine Zuständigkeit und einen ausgewiesenen Prüf- beziehungsweise Erfüllungszustand. Dadurch soll ausstehende Arbeit sichtbar bleiben.
- **Annahmen:** Betroffenheit lässt sich ausreichend bestimmen; die geltenden Verpflichtungen sind identifizierbar; Zuständigkeit und Befunde werden auf die betrachtete Änderung bezogen.
- **Erhoffte Wirkung:** Auch unverändert geltende Regeln bleiben nach einem Change in der Betrachtung, statt still aus dem Arbeits- oder Prüfumfang zu fallen.
- **Herkunft und Status:** Interpretation und Wirkungshypothese des ursprünglichen Produktchats. Bezug zu P2, wo der Nutzer fragt, ob nach einer Änderung weiterhin alle anderen Regeln gelten, sowie P3 und P4 zu Änderungsfolgen und Prüfung.
- **Grenzen und Gegenbelege:** Ein ausgewiesener Zustand kann auf unzureichender Prüfung beruhen. Fehlende Modellverpflichtungen werden durch vollständige Zustandsangaben der bekannten Verpflichtungen nicht automatisch entdeckt. Prüfstatus und tatsächliche Erfüllung dürfen nicht gleichgesetzt werden.
- **Offene Fragen:** Was macht eine weiterhin geltende Verpflichtung betroffen? Wie werden ungeprüfte oder unklare Fälle sichtbar? Wie werden Befunde nach einer weiteren Änderung erneut beurteilt? Welche Zustandsdarstellung genügt, ohne unnötige Verwaltungsarbeit zu erzeugen? Eine konkrete Zustandsmaschine ist nicht beschlossen.
- **Beziehungen:** Vertieft C05, C07, C11, C12, C14 und C16; die Abdeckung hängt von C18 ab.

### C18 Modellgranularität ohne manuelle Kopie des Codes

- **Idee und Problem:** Das Modell soll vollständige modellierte Änderungsfolgen bestimmbar machen und zugleich vermeiden, dass interne Implementierungsdetails nochmals manuell gepflegt werden müssen.
- **Mechanismus:** Ausgangshypothese des ursprünglichen Produktchats: verbindliche Begriffe, Regeln, Grenzen, Beziehungen, Zuständigkeiten, Artefaktgruppen und Prüfverträge explizit halten; interne Implementierungsentscheidungen innerhalb dieser Grenzen delegieren.
- **Annahmen:** Diese Abstraktion erfasst die für Änderungen relevanten Verpflichtungen und erlaubt ausreichend genaue Artefaktzuordnungen. Implementierungsfreiheit lässt sich von verbindlichen Vorgaben unterscheiden.
- **Erhoffte Wirkung:** Nachvollziehbare Folgen und Prüfpflichten bei tragfähigem Pflegeaufwand; das Soll bleibt wirksam, ohne jede interne Codeentscheidung zu duplizieren.
- **Herkunft und Status:** Offene Konzeptfrage und Ausgangshypothese des ursprünglichen Produktchats. Bezug zu K und P2 zur kanonischen Quelle, P3 zu Zuordnung und Änderungsfolgen sowie P5 zum lohnenden Ordnungsaufwand.
- **Grenzen und Gegenbelege:** Zu grobe Artefaktgruppen können Änderungen breit streuen oder Unterschiede verbergen; zu feine Zuordnungen können bei jedem Refactoring aufwendige Modellpflege auslösen. Vollständigkeit innerhalb des modellierten Bereichs beweist keine vollständige Erfassung des Projekts. Die vorgeschlagene Menge expliziter Begriffe ist kein beschlossener Ressourcenkatalog.
- **Offene Fragen:** Welche Detailtiefe reicht für welche Art von Verpflichtung? Wann braucht eine Artefaktgruppe genauere Zuordnungen? Wie werden Folgen erkannt, die nur aus delegierten Implementierungsentscheidungen entstehen? Wie werden fehlende Beziehungen sichtbar? Wann überwiegt zusätzliche Modellpflege ihren Nutzen?
- **Beziehungen:** Vertieft C01, C04, C05, C13, C15 und C16; begrenzt den möglichen Abdeckungsanspruch in C17 und C12.

### C19 Begrenzte Verantwortung mit ausreichendem Kontext

- **Idee und Problem:** Begrenzte Ziele sollen die Verantwortung fokussieren, ohne Manager durch künstliche Informationsblindheit an richtigen Entscheidungen zu hindern.
- **Mechanismus:** Nach der Hypothese des ursprünglichen Produktchats erhalten Manager gemeinsame Verträge und bei Bedarf ausreichend Kontext über den Ist-Zustand. **Eigene Interpretation von Concepts:** Auftragsumfang und notwendiger Informationsumfang können verschieden groß sein.
- **Annahmen:** Kontextbedarf ist erkennbar; gemeinsame Verträge sind verständlich und aktuell; benötigter Ist-Kontext lässt sich zugänglich machen, ohne die begrenzte Verantwortung aufzulösen.
- **Erhoffte Wirkung:** Lokal passende Entscheidungen bleiben mit Nachbarbereichen und dem Gesamtziel vereinbar. Begrenzung reduziert Überforderung, ohne relevante Fakten auszublenden.
- **Herkunft und Status:** Hypothese des ursprünglichen Produktchats; Zusammenhang mit P4 zu Verantwortungsbereichen und begrenzten Agentenzielen. Vertieft die bereits in C09 separat gekennzeichnete Interpretation von Concepts.
- **Grenzen und Gegenbelege:** Geteilte Verträge können selbst unvollständig sein. Zu wenig Kontext begünstigt Integrationsfehler; zu viel Kontext kann Fokus, Laufzeit und Kosten beeinträchtigen. Kenntnis eines Bereichs schafft keine zusätzliche Entscheidungsbefugnis.
- **Offene Fragen:** Wer erkennt fehlenden Kontext? Wann soll ein Manager weitere Informationen anfordern? Wie bleiben Unterschiede zwischen Soll, beobachtetem Ist und geprüfter Übereinstimmung sichtbar? Wie wird ausreichender Kontext beurteilt?
- **Beziehungen:** Vertieft C02, C07, C08 und C09; beeinflusst die Kostenhypothese C10 und die Prüfqualität in C11 und C12.

Die Ergänzung widerspricht keiner zuvor aufgenommenen Aussage ausdrücklich. Sie macht zwei Spannungen genauer sichtbar: Abdeckung gegenüber Pflegeaufwand und begrenzte Verantwortung gegenüber ausreichendem Kontext. Beide bleiben offene Konzeptfragen; die hier notierten Hypothesen lösen sie noch nicht.

## Beziehungen zwischen den Konzepten

Die folgende Übersicht ordnet den vorgeschlagenen Wirkungszusammenhang. Sie ist eine eigene Interpretation, keine bereits geprüfte Ablaufarchitektur.

| Ausgangspunkt | Verbindung | Folge |
|---|---|---|
| C01 Kanonische Quelle | bestimmt das maßgebliche Soll | C02 Soll und Realisierung |
| C03 Typisiertes Modell | prüft ausdrückbare Struktur vor der Arbeit | Gültige Modellgrundlage für C04 und C05 |
| C04 Zuordnung | verbindet Gegenstände und Dateien | C05 Änderungsfolgen |
| C06 Organisation | hilft beim Zuschnitt | C07 Verantwortung und C08 Hierarchie |
| C05 Änderungsfolgen und C07 Verantwortung | liefern begrenzte Arbeits- und Prüfbereiche | C09 Agentenziele |
| C08 Hierarchie und C09 Ziele | sollen Komplexität pro Auftrag reduzieren | C10 mögliche Kostensenkung |
| C11 unabhängige Prüfung und C12 Gesamtprüfung | beurteilen Teilresultate und Zusammenspiel | C14 überprüfbares Vertrauen |
| C04, C05, C11 und C12 | sollen Verpflichtungen beim Umbau bewahren | C13 Cleanup und Refactoring |
| C01 bis C14 | bilden die gewünschte Entwicklungsweise | C15 Spec Driven AI First Development |
| C13, C14 und C15 | sollen fortlaufend praktischen Nutzen liefern | C16 lohnende Spezifikation und Ordnung |
| C17 Verpflichtungserhalt | hält weiter geltende Anforderungen im Änderungsumfang sichtbar | C05 Änderungsfolgen und C16 dauerhafter Nutzen |
| C18 Modellgranularität | bestimmt Genauigkeit und Pflegeaufwand der modellierten Zuordnungen | C04 Zuordnung und C17 Verpflichtungsabdeckung |
| C19 ausreichender Kontext | ermöglicht Entscheidungen über begrenzte Teilaufträge hinweg | C08 Integration und C09 Agentenziele |

Modellbeziehungen, Dateizuordnungen, organisatorische Hierarchie und Prüfrollen haben unterschiedliche Bedeutungen. Eine einzige Baumstruktur muss sie nicht alle vollständig ausdrücken können. Diese Unterscheidung ist eine eigene Interpretation und eine offene Designfrage.

## Zentrale Wirkungshypothesen und mögliche Gegenbelege

| Aussage im Ursprung | Einordnung | Möglicher Gegenbeleg oder einschränkende Bedingung |
|---|---|---|
| Größere Projekte führen zu mehr vergessenen Dateien | Nutzererfahrung aus P1 | Andere Aufgaben oder Arbeitsweisen können ein anderes Fehlerbild zeigen |
| Konsistenzruns brauchen „exponentiell länger“ | Wortlaut des Erfahrungsberichts aus P1 | Ohne definierte Eingangsgröße und Messreihe kein allgemeines Zeitkomplexitätsgesetz |
| Struktur und Diff machen klar, was geprüft werden muss | Hypothese aus P3 und P4 | Fehlende oder falsche Beziehungen lassen relevante Pflichten unsichtbar |
| Durch viele Subagenten wird nichts vergessen | Erhoffte Vollständigkeit aus P4 | Gemeinsame Fehlannahmen und Kontextlücken bleiben trotz mehr Agenten bestehen |
| Günstigere Modelle reichen | Kosten- und Fähigkeitshypothese aus P4 | Nacharbeit, Integration oder zusätzliche Prüfungen können Gesamtkosten und Fehler erhöhen |
| Gesamtcheck gleicht alle Bereiche ab | Gewünschte Abdeckung aus P4 | Unmodellierte und ungeprüfte Bereiche können außerhalb seiner Aussage liegen |
| Methodik schafft Vertrauen und Qualität | Produktziel und Wirkungshypothese aus P5 | Methodentreue kann mit fachlichen Fehlern und hohem menschlichem Kontrollaufwand koexistieren |
| Ordnung lohnt sich dauerhaft | Produktziel aus P5 | Modellpflege kann teurer und fehleranfälliger als der gewonnene Nutzen werden |

Diese Gegenbelege sind eigene analytische Ergänzungen. Sie wurden nicht als beobachtete Misserfolge Markitects festgestellt. Ihre Aufnahme beauftragt keine Tests oder Studien.

## Verwandte Produktdokumente und Varianten

Der [Entwurf zum delegierten Entwicklungsbetriebsmodell](../delegated-engineering-operating-model.md) behandelt unter anderem Repository-Realisierung, rekursive Hierarchie und getrennte Arbeitsgrundlagen. Diese sachliche Beziehung wurde im Dokument geprüft; der Entwurf ist weder der Ursprung der hier erhaltenen Nutzerbeschreibung noch eine durch diese Aufnahme bestätigte Implementierung.

Die [Architektur](../../architecture.md) und die [Refinement-Entscheidungen](../../refinement.md) behandeln explizite Struktur, Beziehungen und Produktgrenzen des hier gelesenen Checkouts. Ihr Inhalt dient zur Orientierung über verwandte Begriffe. Die neuen Konzepte überschreiben ihre Verträge nicht.

Der Einrichtungsauftrag nennt einen zuletzt untersuchten Kandidaten unter `C:/Users/Consiliari/.codex/worktrees/product-integration/Markitect`, Branch `codex/product-integration-20261009`, damals HEAD `3523f168f3af11fad19a3bedfaf181f5429d06a5` mit lokaler Weiterentwicklung. Dies ist übergebener historischer Kontext, kein in dieser Aufnahme aktuell verifizierter Produktstand. Eine spätere Fähigkeitszuordnung muss den dann geprüften Pfad, Branch, SHA und lokale Änderungen nennen.

Die Knowledge-Graph-Variante bleibt gemäß Auftrag separat. Explizite Beziehungen im Konzept sind kein Beschluss, diese Variante zu übernehmen. Ebenso beschließt die gewünschte Managerhierarchie keine bestimmte Government-Architektur.

## Dokumentationsstand und weitere Konzeptklärung

Diese neuen Dateien wurden im Checkout `C:/Users/Consiliari/Glacius Labs/Markitect` auf dem nicht geschützten Feature-Branch `codex/government-assessment` angelegt. Der gelesene Ausgangsstand war `568be480e783b43b1abcea3bfda1d051e0cc33d3`; vor der Aufnahme meldete Git keine lokalen Änderungen. Diese Angabe beschreibt die Ablage und Dokumentlektüre, keine Fähigkeitsvalidierung des Produktkandidaten.

Weitere Diskussion kann insbesondere die Grenzen der Modellabdeckung, die Bedeutung verschiedener Beziehungen, die Verteilung von Entscheidungsbefugnissen, Prüferunabhängigkeit und den praktischen Pflegeaufwand klären. Antworten sind erst mit ihrer tatsächlichen Herkunft als Hypothese, Interpretation oder Entscheidung aufzunehmen. Die Reihenfolge dieser Fragen ist keine Roadmap-Priorisierung.

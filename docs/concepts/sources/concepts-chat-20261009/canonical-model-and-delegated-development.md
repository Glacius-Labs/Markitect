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

Eine spätere, unabhängig übermittelte [Historian-Korrektur](historian-intake.md#hist-markitect-20261009-format-001) kennzeichnet eine Beschreibung von Markitect als „usable as Markdown“ als vom Nutzer zurückgewiesen. Der Vergleich mit Markdown in der ursprünglichen Nutzerbeschreibung bezieht sich auf Organisation und Gruppierung; er entscheidet kein Markitect-Quellformat. Der Historian lässt YAML gegenüber OWL oder Knowledge Graph ausdrücklich offen.

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

## Ergänzung zur delegierten Modellpflege und Government Idee

**Herkunft und Status:** [Direkter Nutzerbeitrag vom 9. Oktober 2026](government-as-model-governance-20261009.md), lokale Concepts-Referenz `USER-20261009-02`. Der Nutzer beschreibt Government als frühere Idee und eine unreife Implementierung als irgendwo in einem Branch vermutet. Es liegt dadurch kein aktueller Branch-, Quellen- oder Fähigkeitsnachweis vor. Die drei folgenden Einträge ordnen die wiederaufgenommene Idee; sie sind keine neu beschlossene Umsetzung.

### C20 Modellpflege als mögliche Nutzeraufgabe

- **Idee und Problem:** Wenn ein Modell zuverlässig umgesetzt und geprüft wird, könnte sich menschliche Aufmerksamkeit auf die Pflege des Modells konzentrieren. Work Items wie User Stories und Features sowie technische Themen müssen dann in passende Modelländerungen übersetzt werden. Ohne eine solche Vermittlung kann Modellführung mehr Anleitung verlangen als die bisherige Arbeitsweise.
- **Mechanismus:** Aufträge werden verstanden und, wo sie das gewünschte Projekt ändern, ins Modell eingearbeitet; die Umsetzung materialisiert das geänderte Modell im Repository. Die lokale Concept-Interpretation ist, dass nicht jeder technische Arbeitsschritt eine neue Solländerung sein muss: Bestehende Regeln können innerhalb der delegierten Freiheit umgesetzt werden. Ob und wie dies im Government-Modell geschieht, ist nicht entschieden.
- **Annahmen:** Relevante Produktabsicht und notwendige technische Änderungen können unterschieden werden; das Modell ist verlässlich auf Realisierungen abbildbar und überprüfbar; Modellpflege erzeugt weniger Routineaufsicht.
- **Erhoffte Wirkung:** Höhere Autonomie der Coding Agents und saubere, verständliche Projekte mit weniger Sorge vor wachsender Unordnung.
- **Herkunft und Status:** Nutzerabsicht und ausdrückliche Bedingungsaussage im Beitrag; Wirkung und Umsetzbarkeit sind als Hoffnung beziehungsweise zu prüfende Frage formuliert.
- **Grenzen und Gegenbelege:** Wenn jede Änderung manuelle Modellpflege und Anleitung benötigt, kann die Aufsicht zunehmen. Technische Wartung kann eine Realisierung verbessern, ohne dass fachliche Sollabsicht geändert wird. Der Nutzer formuliert beide Risiken ausdrücklich; ein durchgängig zuverlässiger Kreislauf ist nicht belegt.
- **Offene Fragen:** Wie werden Work Items in akzeptierte Solländerungen überführt? Welche Arbeiten dürfen innerhalb bestehender Absicht delegiert werden? Wie erkennt man, wann ein aufgefallenes Umsetzungsproblem auf Modelllücke statt auf Implementierungsfehler hindeutet?
- **Beziehungen:** Baut auf C01, C02, C05 und C14 auf. Verbindet die ausführende Hierarchie in C08 mit der offenen Verteilung von Befugnissen in C21.

### C21 Government Analogie für Modellpflege und Konflikte

- **Idee und Problem:** Modelländerungen sollen durch eine autonome Organisationsform bearbeitet werden, die menschliche Aufmerksamkeit bündelt und nur relevante Themen zur Entscheidung vorlegt.
- **Mechanismus im beschriebenen Gedankenbild:** Der Nutzer übernimmt eine Präsidentenrolle und erhält Briefings. Ämter kümmern sich um Pflege und Anpassung des Modells. Mögliche Fachministerien, etwa Sicherheit, Architektur, Code Hygiene, Anwenderfreundlichkeit oder Innovation, können Modelländerungen begutachten und Einwände einbringen. Ein Gericht soll Konflikte zwischen diesen Einwänden bearbeiten. Apply steht in der Analogie für Exekutive; hinzu kämen eine Judikative und eine Legislative.
- **Annahmen:** Pflichten zur Ausführung, Prüfung, Konfliktbehandlung und Veränderung des Modells lassen sich aufgeteilt organisieren. Entscheidungen können innerhalb delegierter Grenzen laufen, während relevante Anliegen nach oben gelangen.
- **Erhoffte Wirkung:** Ein einstellbarer, autonomer Arbeitsorganismus, der Modellpflege, Realisierung und Konfliktklärung trägt und die Nutzeraufmerksamkeit auf bedeutsame Themen lenkt.
- **Herkunft und Status:** Nutzerbeschreibung einer früheren Government Idee. Institutionen und Zuständigkeiten sind Analogien und Vorschläge. Die benannten Ministerien sind ausdrücklich Beispiele; ihre Auswahl, Autorität und Zusammensetzung sind nicht entschieden.
- **Grenzen und Gegenbelege:** Die Metapher klärt weder Entscheidungsbefugnis noch Einwandsgewicht, Eskalation oder Akzeptanz. Mehrere Prüfinstanzen können Konfliktkosten und Aufsicht erhöhen. Die Präsidentenrolle allein legt weder Konsens noch Einstimmigkeit oder eine bestimmte Verfassungsregel fest. Die frühere unreife Implementierung wurde hier nicht lokalisiert oder geprüft.
- **Offene Fragen:** Wer darf das geltende Soll annehmen oder ändern? Sind Ministerien dauerhafte Verantwortungsbereiche, beauftragte Prüfer oder beides? Wie werden widersprüchliche Urteile entschieden? Welche Themen gelten als relevant genug für ein Briefing? Wie hängt dies mit der rekursiven Managerhierarchie C08 und den begrenzten Agentenzielen C09 zusammen?
- **Beziehungen:** Verbunden mit C07 bis C09, C11, C14, C19 und C20. Modellgraph, Delegationshierarchie, Einwand-/Urteilsbeziehungen und Apply sind unterschiedliche Verantwortungen; ihre Verbindung ist eine offene Architekturfrage.

### C22 Laufende Qualität des Modells und Eskalationsaufwand

- **Idee und Problem:** Delegierte Modellpflege darf nicht dazu führen, dass das Modell mit der Zeit an Qualität verliert, und der Nutzer darf nicht durch häufigere Anleitungen stärker belastet werden.
- **Mechanismus:** Im Nutzerbeitrag ist noch kein Qualitätsverfahren festgelegt. Er benennt die laufende Modellqualität und den Aufwand bei Problemen beim Anwenden als Bedingungen, die das Government-Konzept adressieren muss.
- **Annahmen:** Modelländerungen können im Betrieb anhand ausreichender Kriterien beurteilt werden; mögliche Verschlechterungen und wiederkehrende Anleitungen werden sichtbar.
- **Erhoffte Wirkung:** Die gewünschte Autonomie und Repository-Sauberkeit bleiben im tatsächlichen Betrieb erhalten.
- **Herkunft und Status:** Explizite Bedenken und offene Frage im Nutzerbeitrag; jedes konkrete Prüfverfahren ist eine noch offene Idee.
- **Grenzen und Gegenbelege:** Qualität ist ohne gewählte Kriterien und Perspektiven nicht eindeutig. Zusätzliche Ministerien, Gerichte und Checks können Qualität schützen, aber auch neue Verfahrenslast schaffen. Ob die Autonomie tatsächlich wächst, müsse getestet und ausprobiert werden, sagt der Nutzer; daraus folgt kein Test- oder Studienauftrag in diesem Gespräch.
- **Offene Fragen:** Welche Beobachtungen zeigen, dass das Modell verkommt? Wie erkennt man wiederholte Eskalation oder übermäßige Nutzeranleitung? Wie lässt sich eine autonome Arbeitsweise mit vorhandenen Arbeitspaketen, Case Studies und einem stabilen Produktstand fair beurteilen?
- **Beziehungen:** Prüft den Nutzen von C14 bis C16 und C20 bis C21. Inhaltliche Auswertung bleibt an die vom Nutzer in [der Planungsgrenze](discussion-boundaries-20261009.md) genannten Voraussetzungen gebunden.

Der Beitrag verbindet Mensch, Regierung und Management, löst deren Verhältnis aber nicht auf. Dass die Ebenen dieselbe Organisationsstruktur verwenden oder dass eine Government-Umsetzung der richtige Produktweg ist, bleibt unentschieden. Die dokumentierte Wartebedingung für Feinplanung gilt weiter.

## Ergänzung zu Entscheidungsgrundlagen und Eskalation

**Herkunft und Status:** [Direkter Nutzerbeitrag vom 9. Oktober 2026](decision-framework-ideas-20261009.md), lokale Concepts-Referenz `USER-20261009-03`. Der Nutzer möchte die Ideen festhalten und bei späterer Relevanz untersuchen. Sie sind nicht als Verträge angenommen. Die fünf Einträge erhalten ihre Struktur, Statuskennzeichnung und offenen Prüffragen durch Concepts.

### C23 Projektbezogenes Zielbild und Prioritäten für nichtfunktionale Eigenschaften

- **Idee und Problem:** Ein Projekt soll ein Zielbild oder Prioritäten für nichtfunktionale Eigenschaften haben, an denen ein späterer Zielkonflikt abgewogen werden kann.
- **Mechanismus:** Projektbezogene Prioritäten informieren die Einordnung kollidierender Wünsche oder Qualitätsziele.
- **Annahmen:** Das Projekt kann seine Ziele verständlich benennen; eine Rangfolge oder relative Gewichtung kann bei konkreten Konflikten hilfreich sein.
- **Erhoffte Wirkung:** Trade-offs werden im Projektkontext beurteilt, statt wechselnd oder nur implizit entschieden.
- **Herkunft und Status:** Direkte Nutzeridee. Konkrete Eigenschaften, Prioritätsform und Entscheidungsbefugnis wurden nicht festgelegt.
- **Grenzen und Gegenbelege:** Vorab gesetzte Prioritäten können kontextspezifische Anforderungen oder harte Grenzen nicht abbilden. Eine Rangfolge löst nicht zwingend alle Zielkonflikte.
- **Offene Fragen:** Geht es um Rangfolge, Gewichtung, Mindestziele oder Kombinationen? Wie werden projektweite und lokale Ziele vereinbart? Wann ist eine Ausnahme eine echte Intentänderung?
- **Beziehungen:** Ergänzt C02 und C20 und könnte dem Gerichtsgedanken C21 Kontext liefern. Sie entscheidet nicht selbst über einen Konflikt.

### C24 Begründete Benutzerentscheidungen als projektspezifische Präzedenzfälle

- **Idee und Problem:** Frühere Benutzerentscheidungen und ihre Begründungen sollen späteren ähnlichen Auslegungsfragen Orientierung geben.
- **Mechanismus:** Eine projektspezifische Sammlung hält Entscheidungen, Gründe und Bezug fest. „Lernen“ meint laut Nutzer zunächst diesen begründeten Bestand und eine mögliche Auslegungshilfe für einen Richter.
- **Annahmen:** Entscheidungen lassen sich mit ihrem Kontext und Geltungsbereich festhalten; spätere Fälle können mit diesem Kontext verglichen werden.
- **Erhoffte Wirkung:** Wiederkehrende Auslegungsfragen können konsistenter und mit weniger erneuter Nutzeranleitung behandelt werden.
- **Herkunft und Status:** Direkte Nutzeridee und erläuternder Nutzerwortlaut. Kein Modelltraining und kein automatisches Ändern geltender Regeln sind damit gemeint oder beschlossen.
- **Grenzen und Gegenbelege:** Eine frühere Entscheidung kann unter veränderten Umständen ungeeignet sein. Begründungen können eng, mehrdeutig oder miteinander unvereinbar sein. Eine Sammlung allein verleiht dem Agenten keine Entscheidungsbefugnis.
- **Offene Fragen:** Wann gilt ein Präzedenzfall bei geändertem Kontext? Wer stellt den Unterschied fest? Welche Entscheidungen sind Orientierung, welche bindende Auslegung und welche bleiben Einzelfall?
- **Beziehungen:** Berührt die Gerichtsanalogie C21, Nutzerentscheidungen und Grenzen aus C20 sowie die offene Kontextfrage C19.

### C25 Schweregrad von Meinungsverschiedenheiten und höhere Instanzen

- **Idee und Problem:** Uneinigkeit soll nach ihrer Schwere behandelt werden können; manche Fälle könnten eine höhere Instanz erfordern.
- **Mechanismus:** Das Gedankenbild sieht einen Schweregrad der Diskussion und die Möglichkeit übergeordneter Gerichte vor.
- **Annahmen:** Relevante Konflikte lassen sich anhand nachvollziehbarer Kriterien unterscheiden; höhere Instanzen können Fälle beurteilen, die eine niedrigere Ebene nicht lösen darf oder kann.
- **Erhoffte Wirkung:** Schwerwiegende Konflikte erhalten genügend Aufmerksamkeit, während kleinere Fragen auf einer passenden Ebene bleiben können.
- **Herkunft und Status:** Direkte Nutzeridee zur Government-Analogie. Severity-Stufen, Instanzen und ihre Befugnisse sind nicht spezifiziert.
- **Grenzen und Gegenbelege:** Eine falsche Einstufung kann ernste Fragen zu niedrig behandeln oder Routinefragen übermäßig eskalieren. Mehrere Instanzen verursachen Aufwand und können Verantwortlichkeit verwischen.
- **Offene Fragen:** Was bestimmt den Schweregrad? Welche Angelegenheiten dürfen Instanzen entscheiden? Wie bleibt der Umfang unnötiger Eskalationen begrenzt?
- **Beziehungen:** Präzisiert offene Konfliktbehandlung aus C21 und betrifft Nutzerbriefings sowie die Managerhierarchie C08.

### C26 Fallvorlage und Eskalation durch Manager während der Umsetzung

- **Idee und Problem:** Ein Manager soll beim Umsetzen einen ungelösten Fall zur Beurteilung vorlegen oder eskalieren können, statt eine fragwürdige Modellanwendung still fortzusetzen oder den Nutzer selbst durchgehend anzuleiten.
- **Mechanismus:** Manager melden beziehungsweise eskalieren einen Fall an ein vorgesehenes Gericht. Welche Fallunterlagen oder Zwischenzustände dazu nötig wären, ist nicht festgelegt.
- **Annahmen:** Ein ungelöster Modellkonflikt kann von einem Ausführungsfehler unterschieden werden; der Eskalationsweg führt zu einer zuständigen Entscheidungsinstanz.
- **Erhoffte Wirkung:** Blockierende Unsicherheit wird sichtbar und eine passende Entscheidung kann Arbeit gezielt fortsetzen.
- **Herkunft und Status:** Direkte Nutzeridee. Ein konkreter Ablauf oder Pflichtdatensatz ist nicht beschlossen.
- **Grenzen und Gegenbelege:** Zu niedrige Eskalationsschwellen erhöhen die Falllast. Zu hohe Schwellen können Fehlanwendungen oder Konflikte verdecken. Eine Eskalation löst die Frage nach der Entscheidungsbefugnis nicht allein.
- **Offene Fragen:** Wann stoppt die Umsetzung, wann darf sie mit einer ausdrücklich markierten Annahme weitergehen? Wer entscheidet und wie wird ein Fall bei ungelöstem Konflikt abgeschlossen?
- **Beziehungen:** Verknüpft C08, C20, C21 und C25 sowie die Kontextfrage C19.

### C27 Lokale Ziele und Konfigurierbarkeit über Ebenen

- **Idee und Problem:** Ziele, Prioritäten und Wichtigkeit können lokal variieren. Zugleich soll Konfigurierbarkeit auf allen Ebenen möglich sein.
- **Mechanismus:** Jede Ebene könnte im jeweils eigenen Kontext Ziele, Prioritäten und Spielräume berücksichtigen.
- **Annahmen:** Ebenen können lokale Belange verständlich ausdrücken; ihre Beziehung zu übergeordneten Vorgaben kann nachvollzogen werden.
- **Erhoffte Wirkung:** Entscheidungen berücksichtigen reale Bereichsunterschiede und lassen sinnvolle Anpassung zu.
- **Herkunft und Status:** Direkte Nutzeridee. Art, Umfang und Vererbung der Konfiguration sind nicht festgelegt.
- **Grenzen und Gegenbelege:** Ungebundene lokale Einstellungen könnten bindende übergeordnete Regeln abschwächen oder Konflikte verstecken. Zu wenig lokaler Spielraum kann hingegen Entscheidungen an den Anforderungen vor Ort vorbeiführen.
- **Offene Fragen:** Welche lokalen Ziele dürfen verbindliche Vorgaben beeinflussen? Was hat bei widersprüchlichen Prioritäten Vorrang? Wie wird die Gültigkeit lokaler Einstellungen sichtbar, und wie bleiben unnötige Eskalationen begrenzt?
- **Beziehungen:** Vertieft C06 bis C09 und C19; C23 könnte Ziele und Prioritäten bereitstellen.

Diese fünf Ideen ergänzen die früher überlieferte Government-Richtung in C21 und die Managerfreiheit aus C19. Sie lösen den Konflikt zwischen lokaler Priorität, übergeordneter Regel und delegierter Entscheidungsbefugnis nicht auf. Ihre spätere Untersuchung bleibt an die im [Gesprächsrahmen](discussion-boundaries-20261009.md) festgehaltene Planungsgrenze gebunden.

## Ergänzung zu Nutzerübersicht, Betriebsdaten und Darstellung

**Herkunft und Status:** Direkter Nutzerbeitrag vom 9. Oktober 2026 in [Ideen zu Überblick, Monitoring und Visualisierung](operational-overview-ideas-20261009.md), lokale Referenz `USER-20261009-04`. Der US-Präsidentenvergleich wurde als Hörensagen benannt; der Nutzer ergänzte einen [Quellenabgleich mit offiziellen Seiten](operational-overview-source-check-20261009.md). Die Quellen beschreiben die realen Produkte, nicht eine Markitect-Architektur.

### C28 Regelmäßiges Präsidentenbriefing und Managementüberblick

- **Idee und Problem:** Ein regelmäßiger Überblick über wichtige aktuelle Vorgänge soll der Präsidentenrolle Lageverständnis ermöglichen, ohne dass sie jedes Ereignis einzeln verfolgen muss.
- **Mechanismus:** Ein Briefing fasst Vorgänge und den Managementzustand zusammen. **Eigene Interpretation:** Auswahlkriterien für Wichtigkeit, Zeitraum und Entscheidungsrelevanz bestimmen den Nutzen.
- **Annahmen:** Vorgänge und Anliegen sind nachvollziehbar erfasst und eine Zusammenfassung bewahrt Bedeutung, Belege und Unsicherheit.
- **Erhoffte Wirkung:** Überblick und angemessene Entscheidungen mit weniger Informationsüberlastung.
- **Herkunft und Status:** Direkte Zukunftsidee. Der offizielle Vergleich ist ein tägliches nachrichtendienstliches Sicherheitsbriefing; die Markitect-Analogie bleibt weiter gefasst.
- **Grenzen und Gegenbelege:** Verdichtung kann Ausnahmen und Einwände auslassen. Ein fester Tagesrhythmus kann zu langsam oder unnötig sein.
- **Offene Fragen:** Welche Informationen helfen bei Entscheidungen? Was braucht sofortige Eskalation? Wie bleiben Quellen und Details zugänglich?
- **Beziehungen:** Ergänzt Briefings in C21 sowie Severity und Eskalation in C25 und C26.

### C29 Monitoring, Protokolle und Datenerhebung

- **Idee und Problem:** Ein autonomerer Modell- und Agentenbetrieb braucht nachvollziehbare Informationen über seinen Zustand und vergangene Vorgänge.
- **Mechanismus:** Monitoring, Protokolle und Datenerhebung sind mögliche Informationsgrundlagen.
- **Annahmen:** Entscheidungsrelevante Zustände und Vorgänge sind erfassbar und sinnvoll auszuwerten.
- **Erhoffte Wirkung:** Ein Managementüberblick stützt sich auf beobachtete Zustände und nachvollziehbare Vorgänge.
- **Herkunft und Status:** Direkte Nutzeridee. Datenarten, Zugriff, Aufbewahrung und Betrieb bleiben offen.
- **Grenzen und Gegenbelege:** Mehr Erfassung kann Datenlast und Überinformation erzeugen. Protokollierte Aktivität beweist keine korrekte Entscheidung.
- **Offene Fragen:** Welche Daten helfen bei Entscheidungen? Wie werden Quelle, Unsicherheit und Grenzen der Erfassung sichtbar?
- **Beziehungen:** Könnte C28 speisen und betrifft Beleg- und Eskalationsfragen aus C20 bis C22 und C26.

### C30 Kubernetes und etcd als technische Kandidaten

- **Idee und Problem:** Kubernetes oder etcd werden als mögliche technische Ansatzpunkte für den zukünftigen Betrieb erwogen.
- **Mechanismus:** Eine spätere Prüfung vergleicht konkrete Anforderungen mit den jeweiligen Aufgaben und Eigenschaften.
- **Annahmen:** Betriebsanforderungen sind vor einer Technologiewahl klar genug.
- **Erhoffte Wirkung:** Eine tragfähige technische Grundlage, falls ein Kandidat den Bedarf abdeckt.
- **Herkunft und Status:** Direkte technische Vorschläge, keine Auswahl. Offizielle Beschreibungen weisen auf unterschiedliche Aufgaben hin: Kubernetes verwaltet containerisierte Workloads und Services; etcd dient als konsistenter verteilter Key-Value- und Koordinationsdienst.
- **Grenzen und Gegenbelege:** Keiner der beiden Kandidaten erbringt laut seiner Produktbeschreibung allein die gewünschte Briefingfunktion. Die Systeme sind keine austauschbaren Ansätze für denselben Zweck.
- **Offene Fragen:** Welche Anforderungen bestehen tatsächlich? Welche dieser Rollen wäre nötig, und welche Alternativen sind vergleichbar?
- **Beziehungen:** Mögliche technische Umsetzung für die Datengrundlage in C29; der Bedarf soll aus C28 und nicht aus den verfügbaren Technologien abgeleitet werden.

### C31 Visualisierung des eigenen „Reichs“ und Gamification

- **Idee und Problem:** Eine Visualisierung des verwalteten „Reichs“ und Gamification sind mögliche Produkterfahrungen.
- **Mechanismus:** Eine visuelle Ansicht könnte Zustand oder Beziehungen darstellen; eine Gamification würde spielerische Rückmeldung ergänzen. **Eigene Interpretation:** Beides könnte Orientierung und anschaulichen Überblick bezwecken.
- **Annahmen:** Visuelle oder spielerische Rückmeldung fördert Verständnis oder Engagement, ohne wichtige Unterschiede zu verschleiern.
- **Erhoffte Wirkung:** Das verwaltete Projekt wird greifbarer und leichter zu überblicken.
- **Herkunft und Status:** Offene Zukunftsidee des Nutzers; konkretes Design und Nutzenwirkung sind nicht festgelegt.
- **Grenzen und Gegenbelege:** Eine vereinfachte Karte kann Vollständigkeit vortäuschen. Spielziele könnten sichtbare Aktivität statt Projektqualität belohnen.
- **Offene Fragen:** Was zeigt eine Visualisierung besser als ein Briefing? Welche spielerischen Elemente sind hilfreich? Wie bleiben Quelle und aktueller Zustand sichtbar?
- **Beziehungen:** Ergänzt C28 und berührt Vertrauen C14, Modellqualität C22 sowie Prioritäten C23 und C27.

## Ergänzung zu beleggebundenen Ereignissen und sichtbarem Fortschritt

**Herkunft und Status:** Späterer direkter Nutzerbeitrag vom 9. Oktober 2026, wörtlich erhalten im [Quellenabgleich zu Briefing und Technikoptionen](operational-overview-source-check-20261009.md). Die Reichweite des realen Briefings und die technischen Rollen von Kubernetes und etcd wurden anhand offizieller Quellen abgeglichen. Die Aussage zu gemeinsam genutzten beleggebundenen Ereignissen und verifiziertem Fortschritt wurde vom Nutzer ausdrücklich als zusätzliche Produkthypothese ohne Entscheidstatus gekennzeichnet.

### C32 Gemeinsame beleggebundene Ereignisse und Auswertungen

- **Idee und Problem:** Dieselben beleggebundenen Ereignisse könnten Briefings, Historie, Beobachtbarkeit und Visualisierung speisen, anstatt für jede Übersicht eine voneinander getrennte Datensicht zu brauchen.
- **Mechanismus:** Eine gemeinsame Ereignisgrundlage stellt nachweisverknüpfte Informationen für mehrere Ansichten bereit. **Eigene Interpretation:** Jede Ansicht wählt, gruppiert und verdichtet Informationen für ihren Zweck, ohne den Quellenbezug zu verlieren.
- **Annahmen:** Relevante Vorgänge haben erfassbare Quelle und Kontext; der gemeinsame Bestand passt zu den Zwecken der vorgesehenen Sichten.
- **Erhoffte Wirkung:** Wiederverwendbare Belege und konsistente Briefings, Verlaufsansichten und Beobachtbarkeit.
- **Herkunft und Status:** Direkte Produkthypothese des Nutzers vom 9. Oktober 2026, ausdrücklich ein Diskussionsimpuls.
- **Grenzen und Gegenbelege:** Ein gemeinsames Format könnte unterschiedliche Anforderungen verwischen oder unnötigen Erfassungs- und Aufbewahrungsaufwand schaffen. Ein protokolliertes Ereignis ist weder ein Urteil noch ein Vollständigkeitsbeleg.
- **Offene Fragen:** Welche Ereignisse sind relevant? Wie werden Quelle, Prüfergebnis und Unsicherheit sichtbar? Wo müssen verschiedene Sichten getrennte Daten oder Aufbewahrungsregeln behalten?
- **Beziehungen:** Ergänzt C28 und C29 und könnte C31 mit belegbaren Informationen versorgen. C30 legt keine Umsetzungstechnologie fest.

### C33 Gamification auf Grundlage verifizierten Fortschritts

- **Idee und Problem:** Gamification könnte Fortschritt in der als fertig angenommenen Government-Produktwelt sichtbarer machen.
- **Mechanismus:** Nach der Produkthypothese könnten verifizierte Fortschritte angezeigt werden. Welche Fortschritte oder spielerischen Elemente gemeint sind, bleibt offen.
- **Annahmen:** Fortschritt lässt sich aussagekräftig und nachvollziehbar verifizieren; spielerische Rückmeldung stützt passende Ziele.
- **Erhoffte Wirkung:** Nutzer erkennen belegten Fortschritt und erleben ihn als Teil der Produkterfahrung.
- **Herkunft und Status:** Direkter Nutzerbeitrag vom 9. Oktober 2026, ausdrücklich als Produkthypothese ohne Entscheidstatus.
- **Grenzen und Gegenbelege:** Sichtbare Kennzahlen können unvollständige Messgrößen oder Aktivität statt Qualität belohnen. „Verifiziert“ braucht eine geklärte, zum Zweck passende Bedeutung.
- **Offene Fragen:** Was zählt als verifizierter Fortschritt? Wie werden Teilfortschritt, Fehler und Unsicherheit gezeigt? Wie wird verhindert, dass Spielelemente falsche Anreize schaffen?
- **Beziehungen:** Ergänzt C31 und hängt von aussagekräftigen Belegen in C11, C12, C29 und C32 ab.

Der offizielle Quellenabgleich begrenzt die reale PDB-Analogie und beschreibt Kubernetes und etcd als Werkzeuge mit unterschiedlichen technischen Aufgaben. Er entscheidet weder über Markitect-Architektur noch über die Produktreife der Hypothesen C32 und C33.

## Ergänzung zum Cockpit mit persönlichem Chat-Agenten

**Herkunft und Status:** Direkter Nutzerbeitrag vom 9. Oktober 2026, lokale Referenz `USER-20261009-06`, vollständig in [Cockpit mit persönlichem Chat-Agenten](personal-agent-cockpit-20261009.md) erhalten. Gewünschtes mögliches Betriebsbild, noch keine Feinplanung oder Implementierungsentscheidung. Die weitergehende Deutung geteilter Referenzen und delegierter Änderungsbefugnis ist ausdrücklich als vorläufige Nutzerableitung markiert.

### C34 Cockpit mit persönlichem Chat-Agenten

- **Idee und Problem:** Der Nutzer möchte das System interaktiv durchsuchen, Fragen stellen, das kanonische Modell anpassen oder gezielt Interessantes in einer Cockpitansicht sehen können.
- **Mechanismus:** Ein Chat-Interface ist mit einem Agenten verbunden, der das System durchsucht, Antworten gibt, Modelländerungen vornehmen kann oder relevante Inhalte und Ansichten im Cockpit zeigt. Das Cockpit verbindet Gespräch, Projektüberblick, Entscheidungen und Navigation.
- **Annahmen:** Agent und sichtbare Cockpitansichten beziehen sich auf das relevante Projekt und zeigen Inhalte, die der Nutzer im jeweiligen Kontext braucht.
- **Erhoffte Wirkung:** Fragen, Einsicht ins Projekt und genehmigte Modellarbeit werden in einem zugänglichen Nutzungserlebnis verbunden.
- **Herkunft und Status:** Gewünschtes mögliches Betriebsbild aus dem direkten Nutzerbeitrag. Die konkreten Berechtigungen, Abläufe und Produktentscheidungen sind nicht festgelegt.
- **Grenzen und Gegenbelege:** Die Aussage, der Agent könne das Modell anpassen, bestimmt noch keine konkrete Übertragungs- oder Änderungsbefugnis. Ein angenehmes Chat- oder Cockpiterlebnis belegt keine korrekten Quellen, Ansichten oder Änderungen.
- **Vorläufige Nutzerableitung:** Chat und Cockpitansicht teilen denselben Bezug auf Modell, Entscheidungen und beobachtete Evidenz. Der Agent handelt innerhalb übertragener Modelländerungsbefugnisse; ungelöste materielle Entscheidungen bleiben sichtbar. Ein Gesprächswunsch allein schafft keine neue automatische Vollmacht. Dies ist eine vorläufige Auslegung des Nutzers, kein angenommener Vertrag.
- **Offene Fragen:** Welche Interessen soll ein Cockpit proaktiv sichtbar machen? Welche übertragene Befugnis umfasst eine Modelländerung? Wie werden ungelöste materielle Entscheidungen angezeigt?
- **Beziehungen:** Verknüpft Präsidentenüberblick C28, Ereignisgrundlage C29 und C32, Government Rollen C20 bis C21 sowie visuelle Orientierung C31. Schnittstellen, Technologien und Protokolle bleiben ausdrücklich außerhalb der Notiz.

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
| C20 Modellpflege als mögliche Nutzeraufgabe | übersetzt Produktabsicht und technische Bedürfnisse in wirksame Änderungen | C01 Sollmodell und C08 Ausführung |
| C21 Government Analogie | verteilt vorgeschlagene Pflege-, Einwands- und Konfliktrollen | C08 Managerhierarchie und C11 unabhängige Prüfung |
| C22 Modellqualität und Eskalationsaufwand | benennt offene Bedingungen für Autonomie im Betrieb | C14 Vertrauen und C20 delegierte Pflege |
| C23 Projektziele und Prioritäten | bieten projektbezogenen Kontext für Trade-offs | C20 Modelländerungen und C21 Konfliktbehandlung |
| C24 Begründete Entscheidungen | bewahren projektspezifische Auslegung als mögliche Orientierung | C21 Gerichte und C23 Prioritäten |
| C25 Severity und höhere Instanzen | ordnen mögliche Konfliktbehandlung nach Schwere | C08 Hierarchie und C21 Rollenanalogie |
| C26 Managerfall und Eskalation | bringt Umsetzungsprobleme zur passenden Beurteilung | C20, C21 und C25 |
| C27 lokale Ziele und Konfiguration | bildet Prioritäten einzelner Ebenen ab | C07 bis C09, C19 und C23 |
| C28 regelmäßiger Überblick | gibt der Nutzerrolle eine verdichtete Lageeinschätzung | C21 Briefings, C25 Severity und C26 Eskalation |
| C29 Monitoring und Protokolle | liefert mögliche Beobachtungsgrundlagen | C20 bis C22 und C28 |
| C30 Kubernetes und etcd | benennt ungelöste Technologieoptionen | C29 Anforderungen und C28 Nutzerbedarf |
| C31 Visualisierung und Gamification | prüft mögliche visuelle und spielerische Orientierung | C14, C22, C23 und C28 |
| C32 gemeinsame Ereignisgrundlage | könnte Nachweise für mehrere Nutzersichten wiederverwenden | C28, C29 und C31 |
| C33 verifizierter Fortschritt | könnte belegten Fortschritt sichtbar machen | C11, C12, C29, C31 und C32 |
| C34 persönliches Agenten-Cockpit | verbindet Fragen, Suche, Modellarbeit und Ansichten | C20, C21, C28, C29, C31 und C32 |

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

Das [Standardbetriebsmodell](../../../design/standard-operating-model.md) behandelt delegierte Arbeit, unabhängige Prüfung und Erfassung betroffener Bereiche. Es ist ein thematisch verwandter Produktentwurf, weder Ursprung der hier erhaltenen Nutzerbeschreibung noch Fähigkeitsnachweis für die hier notierten Erwartungen.

Die [Architektur](../../../architecture.md) und die [Refinement-Entscheidungen](../../../refinement.md) behandeln explizite Struktur, Beziehungen und Produktgrenzen des hier gelesenen Checkouts. Ihr Inhalt dient zur Orientierung über verwandte Begriffe. Die neuen Konzepte überschreiben ihre Verträge nicht.

Der Einrichtungsauftrag nennt einen zuletzt untersuchten Kandidaten unter `C:/Users/Consiliari/.codex/worktrees/product-integration/Markitect`, Branch `codex/product-integration-20261009`, damals HEAD `3523f168f3af11fad19a3bedfaf181f5429d06a5` mit lokaler Weiterentwicklung. Dies ist übergebener historischer Kontext, kein in dieser Aufnahme aktuell verifizierter Produktstand. Eine spätere Fähigkeitszuordnung muss den dann geprüften Pfad, Branch, SHA und lokale Änderungen nennen.

Die Knowledge-Graph-Variante bleibt gemäß Auftrag separat. Explizite Beziehungen im Konzept sind kein Beschluss, diese Variante zu übernehmen. Ebenso beschließt die gewünschte Managerhierarchie keine bestimmte Government-Architektur.

## Dokumentationsstand und weitere Konzeptklärung

Diese neuen Dateien wurden im Checkout `C:/Users/Consiliari/Glacius Labs/Markitect` auf dem nicht geschützten Feature-Branch `codex/government-assessment` angelegt. Der gelesene Ausgangsstand war `568be480e783b43b1abcea3bfda1d051e0cc33d3`; vor der Aufnahme meldete Git keine lokalen Änderungen. Diese Angabe beschreibt die Ablage und Dokumentlektüre, keine Fähigkeitsvalidierung des Produktkandidaten.

Weitere Diskussion kann insbesondere die Grenzen der Modellabdeckung, die Bedeutung verschiedener Beziehungen, die Verteilung von Entscheidungsbefugnissen, Prüferunabhängigkeit und den praktischen Pflegeaufwand klären. Antworten sind erst mit ihrer tatsächlichen Herkunft als Hypothese, Interpretation oder Entscheidung aufzunehmen. Die Reihenfolge dieser Fragen ist keine Roadmap-Priorisierung.

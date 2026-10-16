# Kanonisches Modell: Entwurf der Beschreibungssprache

Fortschreibung vom 12. Oktober 2026: Diese Datei bleibt eine datierte Vorarbeit. Die aktuelle zusammenhängende Empfehlung zu `mandate.yaml`, nominalem `Rule`, verbindlicher Prosa, Referenzen und methodischen Reviews steht im [konsolidierten Proposal](canonical-model-proposal.md) und [aktuellen Bibliotheksbeispiel](library-model-example.md). Der folgende umfangreichere Ausdruckskatalog ist kein Mindestvertrag des neuen Ausgangspunkts.

Stand: 10. Oktober 2026. Version: Entwurf 0.1. Status: zur gemeinsamen Diskussion.

**Arbeitsdefinition:** Das kanonische Modell ist die maßgebliche, typisierte und verständliche Beschreibung eines Projekts: seiner Systeme, Begriffe, Fähigkeiten, Struktur, Regeln, Qualitätsanforderungen, Betriebsbedingungen und Arbeitsweise. Es beschreibt außerdem, welche Realisierungen daraus entstehen sollen, wie deren Übereinstimmung geprüft wird und welche Entscheidungen bei der Realisierung offenbleiben dürfen.

Der Nutzer hat diesen Entwurf beauftragt; damit sind seine Einzelvorschläge noch nicht angenommen. Er entwickelt das [vereinbarte Grundmodell](README.md) und die [Arbeitsnotiz](system-description-notes.md) weiter. Er ersetzt weder [Vision](../../vision.md), [Architektur](../../architecture.md), [Konzeptregister](../../concepts/register.md) noch ein vorhandenes Schema. Es wurden keine Produktfunktionen implementiert, Tests oder Studien gestartet.

Eine [separate C#-Ausarbeitung](csharp-model-design.md) konkretisiert `SoftwareSystem`, `Repository` und ihre Beziehungen am Bibliotheksbeispiel. Sie ist ebenfalls ein Entwurf und legt weder das Autorenformat noch neue Core-Typen fest.

Der [Proposal vom 11. Oktober](canonical-model-proposal.md) entwickelt diese Vorarbeit weiter. Der [neuere Main-Abgleich](canonical-model-main-comparison.md) korrigiert insbesondere die historische Einordnung des Schema-/Projection-Module-Splits. Abschnitt 10 dieses älteren Dokuments bleibt ein Vergleich mit seiner damaligen Basis, kein aktueller Main-Befund.

## 1. Gegenstand und Bedeutung des Modells

### 1.1 Der geltende Systemstand

Der Nutzer bezeichnet den Gegenstand als IST-Zustand. **Arbeitsannahme dieses Entwurfs, noch zu bestätigen:** Gemeint ist die geltende Definition des Systems und Projekts: Welche Inhalte und Regeln gelten jetzt? Beobachteter Repository- oder Betriebszustand ist davon unterscheidbar. Git, Work Items, Roadmap und zukünftige Ideen sind hier ausgeklammert.

Diese Definition ist zugleich der Maßstab für das Repository. Wenn das Repository eine Regel verletzt, liegt eine Abweichung der Realisierung vor. Sie wird nicht allein dadurch zu einer neuen gültigen Regel, dass sie beobachtet wurde. Ein Modellwechsel ändert die geltende Definition; bis zur Umsetzung kann die Realisierung davon abweichen.

Die Beschreibung muss unterscheiden können zwischen verbindlichen Festlegungen, erklärenden Begründungen und Annahmen. Offene Fragen gehören zum Entwurf oder sind ausdrücklich als noch nicht festgelegt gekennzeichnet. Ein solcher offener Punkt ist keine heimliche Erlaubnis für eine beliebige Entscheidung.

### 1.2 System, Repository und Projekt

| Gegenstand | Was beschrieben wird |
| --- | --- |
| Softwaresystem | Zweck, Akteure, Fachwelt, Fähigkeiten, Verhalten, Struktur, Schnittstellen und geltende Eigenschaften. |
| Realisierung und Betrieb | Technische Umsetzungsvorgaben, Artefakterwartungen, Konfiguration, Bereitstellung, Datenhaltung und Betriebsverfahren. |
| Projektarbeit | Verantwortungen, Befugnisse, Entwicklungs- und Prüfverfahren, Dokumentationspflichten, Skills und Arbeitsregeln. |

Das Repository enthält Realisierungen dieser Inhalte. Es ist keine vollständige Beschreibung der außerhalb des Repositorys betriebenen Wirklichkeit. Externe Systeme, Betreiber, Umgebungen und tatsächliche Laufzeitbeobachtungen brauchen ausdrücklich benannte Grenzen beziehungsweise Evidenz.

Ein Projekt kann mehrere Systeme und Repositories betreffen; ein Repository kann mehrere Systeme enthalten. Der modellierte Umfang und die Beziehungen werden deshalb explizit beschrieben. Für das Beispiel genügt ein System in einem Repository.

### 1.3 Vollständigkeit und Abstraktionsgrad

Vollständigkeit gilt für einen erklärten Umfang und den vom Autor gewählten Detaillierungsgrad. Sie bedeutet: Relevante Anforderungen und Beziehungen sind beschrieben, und erforderliche Entscheidungen sind festgelegt oder ausdrücklich delegiert. Sie verlangt keine zweite Kopie jeder Codezeile und keine Definition pro Datei.

Eine knappe Beschreibung kann vollständige Freiheit innerhalb eines Vertrags zulassen. Eine andere kann Technologie, Datenmodell oder einzelne Algorithmen verbindlich bestimmen. Beides muss darstellbar sein. Bewusst freigegebene Umsetzungsspielräume werden von fehlenden oder widersprüchlichen Festlegungen unterschieden.

Ein Entwurf kann vollständig genug für eine ausgewählte Aufgabe und zugleich unvollständig für das gesamte Projekt sein. Dieser Unterschied darf nicht verschwinden, wenn eine einzelne Prüfung besteht.

## 2. Vorgehen zur Entwicklung des Modells

Der Nutzer möchte das zuletzt besprochene Vorgehen behalten. Es verbindet drei Ansätze: konkrete Beschreibung, Ableitung aus Artefakten und Ableitung von Ausdrucksmitteln aus Programmiersprachen.

1. **Umfang festlegen:** Welches System, welche Realisierungen und welche Projektarbeit sollen beschrieben werden? Was bleibt außerhalb?
2. **Gewöhnliche Dokumentation schreiben:** Ein bekanntes System zunächst lesbar beschreiben, ohne eine fertige Ontologie vorauszusetzen. Ausgangsbeispiel ist eine Bibliotheksverwaltung.
3. **Artefaktfamilien auf Lücken prüfen:** Welche Aussagen benötigen Anwendungscode, Tests, Datenhaltung, Konfiguration, Pipeline, Infrastruktur, Dokumentation und Skills, damit sie passend entstehen können?
4. **Aus Aussagen Begriffe und Ausdrucksmittel ableiten:** Konkrete Aussage → benötigter Inhalt → benötigtes Ausdrucksmittel. Wiederkehrende Begriffe werden definiert; Kombination, Abstraktion, Namen, Typen und Geltungsbereiche werden untersucht.
5. **Beschreibung auf ausreichende Festlegungen prüfen:** Abläufe, Fehlerfälle, Grenzen und Artefakterwartungen gedanklich durchgehen. Welche relevante Entscheidung wäre ohne weitere Anleitung ungeklärt?
6. **Mit anderen Perspektiven und einem Kontrastbeispiel prüfen:** Dokumentationsansätze und die vom Nutzer genannten Disziplinen als Fundus verwenden. Ein kleines CLI-Werkzeug prüft, ob wir Bibliotheksbegriffe versehentlich zu universellen Voraussetzungen gemacht haben.

Arbeitsmaßstab: Für jede relevante Aufgabe und Artefaktfamilie muss erkennbar sein, welche Festlegungen gelten, welche Zusammenhänge berücksichtigt werden müssen und welche Entscheidungen bei der Realisierung offenbleiben dürfen.

Das Ergebnis ist eine Beschreibungssprache mit nachvollziehbaren Gründen für ihre Ausdrucksmittel. Die sechs folgenden Bereiche sind Sichten zur Prüfung ihrer Abdeckung; sie sind keine sechs Pflichttypen oder ausschließlich getrennte Ordner.

## 3. Was das Modell darstellen können muss

| Sicht | Erforderliche Beschreibungsmöglichkeiten | Bibliotheksbeispiel |
| --- | --- | --- |
| Zweck und Umfang | Problem, Nutzen, Ziele, Akteure, Systemgrenze, externe Beteiligte, Annahmen, Ausschlüsse und Erfolgsmaßstäbe. | Zugang zu Büchern, ohne jedes Buch kaufen zu müssen; Mitglieder und Mitarbeiter; Buchhaltung außerhalb des Umfangs. |
| Fachlichkeit und Verhalten | Begriffe, Gegenstände und Werte, Identität, Eigenschaften, Beziehungen, Zustände, Fähigkeiten, Abläufe, Regeln, Ereignisse, Ergebnisse und Fehlerfälle. | Titel, Exemplare, Mitglieder und Ausleihen; ausleihen, verlängern und zurückgeben; höchstens eine aktive Ausleihe pro Exemplar. |
| Struktur und Architektur | Kohäsive Bereiche, Bausteine, Zusammensetzung, bereitgestellte und benötigte Verträge, Schnittstellen, externe Integrationen und erlaubte Abhängigkeiten. | Ausleihe verwendet Bestandsinformationen und Mitgliedsberechtigung über beschriebene Verträge. |
| Qualität und Randbedingungen | Sicherheits-, Zuverlässigkeits-, Leistungs- und andere Qualitätsanforderungen, übergreifende Regeln, Einschränkungen, Prioritäten und Prüfkriterien. | Gleichzeitige Anfragen erzeugen keine doppelte Ausleihe; Mitgliederdaten werden nur Berechtigten zugänglich. |
| Bereitstellung und Betrieb | Ausführbare Einheiten, Umgebungen, Ressourcen, Konfiguration, Datenhaltung und Datenänderungen, Bau und Bereitstellung, Beobachtung, Sicherung und Wiederherstellung. | Dienst und Datenspeicher; konfigurierbare Ausleihdauer; Sicherung und Wiederherstellung aktiver Ausleihen. |
| Arbeitsweise und Verantwortung | Verantwortungsbereiche, Rollen, Befugnisse, Verfahren, Review und Integration, Skills, Prüfpflichten, Dokumentationspflichten und Eskalation. | Verantwortungen für Fachregeln, Anwendung, Prüfung und Betrieb; definierte Entscheidung bei unklarer Ausleihregel. |

Die Sprache muss diese Inhalte ermöglichen. Ein einzelnes Projekt muss nicht jede mögliche Eigenschaft oder Organisationsform verwenden. Nicht relevante Inhalte werden im erklärten Umfang ausgeschlossen. Eine kleine Anwendung braucht keine künstliche Ministerien- oder Microservice-Struktur.

### 3.1 Besondere Inhaltsunterscheidungen

- **Problem, Ziel und Fähigkeit:** Anlass, angestrebtes Ergebnis und verfügbare Leistung sind unterschiedliche Aussagen.
- **Begriff und Gegenstand:** Der Begriff Exemplar beschreibt eine Art von Dingen; ein bestimmtes Exemplar ist ein Gegenstand innerhalb der Bibliothek. Das Projektmodell beschreibt gewöhnlich deren Struktur und Regeln, nicht den kompletten aktuellen Laufzeitbestand.
- **Eigenschaft und Invariante:** Eine Eigenschaft beschreibt etwa den Zustand; eine Invariante beschränkt zulässige Zustände oder Übergänge.
- **Fähigkeit und Ablauf:** Eine Fähigkeit beschreibt eine verfügbare Leistung; ein Ablauf konkretisiert Auslöser, Beteiligte, Schritte und Ergebnisse.
- **Regel und Strategie:** Eine verbindliche Ausleihgrenze unterscheidet sich von einer Strategie zur Bewertung eines Sortiments.
- **Anforderung, Prüfmethode und Ergebnis:** Die gewünschte Eigenschaft, das Verfahren zu ihrer Prüfung und der beobachtete Befund sind verschiedene Informationen. Testergebnisse sind Evidenz zum Modell, keine neue kanonische Anforderung.

## 4. Wie das Modell Inhalte ausdrückt

Die folgenden Elemente bilden eine vorgeschlagene konzeptionelle Grammatik. Ihre Namen beschreiben Rollen von Informationen, keine beschlossene YAML-Syntax und keine Liste neuer Core-Kinds.

Kurzes Begriffsgerüst für das Lesen des Entwurfs:

| Begriff | Arbeitsbedeutung |
| --- | --- |
| System | Abgegrenztes Ganzes mit beschriebenem Zweck, Verhalten und Beziehungen zu seiner Umgebung. |
| Bereich | Kohäsive Einheit zur Gliederung von Inhalten; kann Teilbereiche enthalten und Verträge anbieten. |
| Definition | Benannte, referenzierbare Beschreibung eines Begriffs oder Modellelements. |
| Aussage | Inhalt über einen benannten Gegenstand; kann etwa eine Regel, Anforderung, Annahme oder Begründung sein. |
| Vertrag | Zusammengehörige verbindliche Aussagen über angebotene Leistungen, Voraussetzungen, Ergebnisse und Grenzen. |
| Akteur | Mensch, Rolle, Prozess oder externes System, das in einem beschriebenen Vorgang mit dem System interagiert. |
| Rolle | Beschriebene Aufgaben und Befugnisse, die ein Akteur übernehmen kann. |
| Schema | Definiert zulässige Arten von Modellelementen, Eigenschaften und Beziehungen für ein Vokabular. |
| Realisierungsindex | Standgebundene, ermittelte Zuordnung von konkreten Artefakten zu Modellinhalten und Verantwortungen. |

### 4.1 Benannte Definitionen und typisierte Eigenschaften

Eine Definition hat eine eindeutige referenzierbare Identität, einen Typ, eine lesbare Bedeutung und einen erkennbaren Kontext. Typisierte Eigenschaften können beispielsweise Text, Zahl, Einheit, Auswahl, Sammlung, strukturierten Wert oder Referenz ausdrücken, soweit das gewählte Schema diese Formen unterstützt.

Mit **Modellelementart** ist hier eine im Schema definierte Art wie Fachbegriff, Komponente oder Verhaltensvertrag gemeint. Ihr Vertrag legt erlaubte Eigenschaften und Beziehungen fest. Ein **Werttyp** beschränkt dagegen einen Eigenschaftswert, etwa auf eine Zahl oder Referenz. Welche davon als gemeinsame Grundelemente oder als projektspezifisches Vokabular umgesetzt werden, bleibt offen. Eine Modelldefinition von Exemplar ist außerdem nicht automatisch ein unmittelbar ausführbarer Quellcodetyp.

Beispiel: Ein Exemplar hat eine Identität, gehört zu einem Titel und hat einen Zustand. Die Identität des konkret ausgeliehenen Exemplars ist von der Identität der Modelldefinition Exemplar zu unterscheiden.

Namespaceräume gruppieren Definitionen und vermeiden Mehrdeutigkeit. Ihre Beziehung zu Dateipfaden ist eine deklarierte Konvention. Die bloße Verschiebung einer Datei darf nicht unbemerkt als neue Fachbedeutung behandelt werden. Ob Identität pfadunabhängig bleibt, muss im späteren Formatvertrag festgelegt werden.

### 4.2 Aussagen, Regeln und Begründungen

Eine wichtige Aussage braucht einen erkennbaren Gegenstand, Inhalt und Geltungsbereich. Wenn sie unabhängig referenziert, betroffen oder geprüft werden soll, braucht sie eine adressierbare Identität. Nicht jeder erklärende Satz muss zu einem eigenen Modellelement werden.

Verbindliche Regeln, Qualitätsanforderungen, Annahmen und Begründungen werden unterscheidbar beschrieben. Eine wichtige Annahme kann mit der darauf beruhenden Regel oder Strategie verbunden werden. Dadurch kann eine geänderte Annahme Anlass zur erneuten Bewertung ihrer Folgen sein.

Beispiel: Die Annahme einer veränderlichen Nachfrage begründet eine regelmäßige Bewertung des Sortiments. Sie ist nicht gleichbedeutend mit einer messbaren Garantie, dass die gewählte Strategie optimal ist.

### 4.3 Explizite, typisierte Beziehungen

| Beziehung | Bedeutung | Beispiel |
| --- | --- | --- |
| gehört zu / enthält | Fachliche oder architektonische Zusammensetzung. | Ausleihe gehört zur Bibliotheksanwendung. |
| verwendet / benötigt | Eine benannte fachliche, technische oder Informationsabhängigkeit. | Ausleihe benötigt Mitgliedsberechtigung. |
| bietet / erfüllt | Ein Baustein bietet oder erfüllt einen Vertrag. | Mitgliedsverwaltung bietet die Berechtigungsabfrage. |
| betrifft / beschränkt | Eine Aussage gilt für benannte Gegenstände oder Bereiche. | Datenschutzregel betrifft Mitgliedsdaten und deren Verarbeitung. |
| verantwortet | Eine Rolle oder Verantwortung besitzt ein beschriebenes Mandat. | Ein Bereich verantwortet Ausleihregeln. |
| realisiert | Ein Artefakt setzt eine für seine Rolle geltende Modellfestlegung um. | Anwendung realisiert den Ausleihvertrag; Tests realisieren das vorgeschriebene Prüfverfahren. |
| prüft | Ein Verfahren untersucht eine ausdrücklich benannte Aussage. | Eine Prüfung untersucht die Eindeutigkeit aktiver Ausleihen. |
| beschreibt / vermittelt | Ein Artefakt macht bestimmte Inhalte für einen beschriebenen Zweck verständlich. | Bedienungsdokumentation vermittelt das Ausleihen und realisiert die entsprechende Dokumentationspflicht. |

Fachliche Referenzen, Quellcodeabhängigkeiten, Laufzeitaufrufe und Ausführungsreihenfolgen müssen bei Bedarf unterscheidbar sein. Eine einzige allgemeine Dependency-Kante wäre dafür zu ungenau. Jede Beziehung braucht eine definierte Richtung und Bedeutung; erlaubte Ausgangs- und Zielarten werden durch das jeweilige Schema festgelegt.

Prosalinks unterstützen die Navigation. Maschinenrelevante Beziehungen werden ausdrücklich deklariert; sie werden nicht aus frei formulierter Prosa erraten.

### 4.4 Verhaltensverträge

Ein Verhaltensvertrag kann ausdrücken:

- Auslöser, beteiligte Akteure und Eingaben;
- Voraussetzungen und Berechtigungen;
- Ergebnisse, Zusagen und relevante Zustandsänderungen;
- Fehlerfälle und das dann geltende Verhalten;
- bei Bedarf Zeit, Gleichzeitigkeit, Wiederholung und Nebenwirkungen.

Ein Vertrag legt beobachtbares Verhalten fest. Ein detaillierter Ablauf kann ihn ergänzen, wenn die Art der Umsetzung zum Projektwillen gehört. Sonst kann das innere Verfahren offenbleiben.

Für Zustandsmodelle braucht es benannte Zustände und zulässige Übergänge mit Bedingungen und Wirkungen. Ein Projekt ohne dauerhafte Zustände darf Verhalten allein als Transformation von Eingaben zu Ergebnissen beschreiben.

### 4.5 Kohäsive Bereiche, Abstraktion und Schnittstellen

Ein Bereich bündelt zusammengehörige Begriffe, Daten, Verhalten und Regeln. Er beschreibt seinen Zweck, seinen Inhalt, seine angebotenen und benötigten Verträge sowie seine Grenzen. Leser und Agenten sollen mit ihm arbeiten können, ohne sämtliche inneren Details gleichzeitig zu kennen.

Ein Bereich kann wiederum Teilbereiche enthalten. Diese rekursive Gliederung schafft eine Hierarchie. Eine bestimmte Managementrolle wird davon getrennt beschrieben; ein Bereich ist nicht automatisch ein Agent.

Ein Manager trägt ein Mandat für einen Bereich. Ein konkreter Agent übernimmt für eine Ausführung eine Rolle. Eine Verantwortungsstruktur kann aus der Bereichsstruktur abgeleitet oder ausdrücklich darauf abgebildet werden. Die eindeutigen Regeln dafür sind noch festzulegen; Ordnernamen allein begründen keine Entscheidungsbefugnis.

### 4.6 Übergreifende Regeln und Umsetzungsspielraum

Eine Regel kann mehrere Bereiche betreffen, ohne deren Gliederung zu verändern. Ihr Geltungsbereich wird durch benannte Ziele oder eine klar definierte Auswahlregel beschrieben. Unklare implizite Vererbung durch Namespace- oder Ordnernähe ist zu vermeiden.

Prioritäten und begründete Ausnahmen können ausdrücklich beschrieben werden. Ein Agent darf keine widersprechende Regel selbst als aufgehoben behandeln. Für nicht auflösbare Konflikte braucht es eine benannte Entscheidung oder Eskalation.

Umsetzungsspielräume benennen die Fragen, die innerhalb der verbindlichen Verträge entschieden werden dürfen. Beispiel: Der öffentliche Ausleihvertrag ist festgelegt; die interne Datenstruktur darf der zuständige Bereich wählen. Vorgeschriebene Technologie kann dagegen ausdrücklich Teil des Modells sein, wenn das Projekt sie als verbindlich betrachtet.

## 5. Hierarchie, Graph und Management

Der Entwurf übernimmt den Gedanken des Nutzers als Arbeitshypothese: Hierarchische Gliederung und explizite Querbeziehungen ergänzen sich.

| Struktur | Frage | Mögliche Einschränkung |
| --- | --- | --- |
| Gliederung | Wozu gehört eine Definition oder ein Bereich? | Eindeutige Zugehörigkeit, keine Enthaltenseinszyklen. |
| Verantwortung | Wer trägt welches Mandat und wer integriert Teilbereiche? | Eindeutige Verantwortung für eine konkrete Änderung beziehungsweise Ausgabe. |
| Semantische Beziehungen | Welche Begriffe, Regeln und Verträge beziehen sich aufeinander? | Zulässige Typen und Bedeutungen; nicht pauschal jeder Zyklus verboten. |
| Architekturabhängigkeiten | Welche Bestandteile dürfen welche anderen verwenden? | Projektspezifische Richtungs-, Sichtbarkeits- und Zyklusregeln. |
| Ausführungsreihenfolge | Welche Arbeit setzt ein anderes Ergebnis voraus? | Keine unauflösbaren Wartezyklen für den betrachteten Ablauf. |

Die letzten konkreten Ausführungsabhängigkeiten entstehen aus Modell, Operation und betroffenem Stand. Sie sind keine zusätzliche manuell zu pflegende Systembeschreibung.

Bibliotheksbeispiel: Bestand, Mitglieder und Ausleihe gehören zur Bibliothek. Ausleihe benötigt Informationen aus Bestand und Mitglieder. Diese Beziehung gibt der Ausleihe keine Verantwortung über die beiden anderen Bereiche und bestimmt noch keine Reihenfolge aller möglichen Arbeiten.

## 6. Realisierung ohne jede Datei aufzuzählen

### 6.1 Erwartungen auf semantischer Ebene

Der Autor beschreibt erwartete **Artefaktrollen und Realisierungsverträge**: Anwendung, Prüfungen, Bedienungsdokumentation, Betriebsanleitung, Konfiguration, Datenhaltung oder Bereitstellung. Dazu gehören bei Bedarf Technologie, Format, Ort oder Auswahlkonventionen, wenn diese verbindlich sein sollen.

Beispiel einer vom Projekt gewählten Erwartung: Die Ausleihfähigkeit muss in der Anwendung nutzbar sein, ihre Regeln müssen geprüft werden und ihre Bedienung muss beschrieben sein. Das verlangt keine manuelle Liste jedes Handlers, jeder Testdatei und jeder Dokumentationsseite. Welche Artefakte und Evidenz verpflichtend sind, bestimmt das Projekt ausdrücklich; zusätzliche hilfreiche Ansichten dürfen nicht stillschweigend zu Pflichtausgaben werden.

Eine reine Ordnerauswahl benennt einen Bereich von Dateien, beweist aber nicht, dass diese Dateien die richtige Bedeutung realisieren. Eine Erwartung darf deshalb sowohl einen eindeutigen technischen Auswahlvertrag als auch eine semantische Prüfung benötigen.

### 6.2 Konkrete Zuordnung als ermittelte Sicht

Für einen festen Repositorystand muss Markitect beziehungsweise ein zuständiger Adapter oder Agent eine konkrete Zuordnung zwischen Modellinhalten und vorhandenen Artefakten ermitteln und prüfen können. Diese Zuordnung ist ein Index der Realisierung mit Herkunft und Standbindung. Sie ist keine zweite unabhängig gepflegte Wahrheit über den Projektwillen.

Der Ansatz kann deklarierte Konventionen, begrenzte technische Analyse und durch Agenten vorgeschlagene Zuordnungen kombinieren. Welche Kombination zuverlässig genug ist, bleibt eine offene Produktfrage. Agentenvorschläge dürfen nicht als ungeprüfte Beweise behandelt werden. Technologische Analyse gehört in geeignete Werkzeuge beziehungsweise Adapter, nicht automatisch in den strukturellen Core.

Eine Aussage kann durch mehrere Dateien realisiert werden; eine Datei kann mehrere Aussagen realisieren. Für die Änderung einer konkreten Datei oder eine generierte Ausgabe muss trotzdem eine eindeutige zuständige Verantwortung bestehen. Inhaltliche Zuordnung und Schreibzuständigkeit sind unterschiedliche Beziehungen.

Jede Datei im betrachteten Repositoryumfang muss konkret eingeordnet oder ausdrücklich ausgenommen sein. Diese Abdeckung wird am festen Realisierungsstand geprüft; sie verlangt keine Einzelpfaddeklaration im kanonischen Modell. Aus deklarierter Konvention und Repositoryerfassung können konkrete Pfade abgeleitet werden. Ohne Einzelpfade in der kanonischen Beschreibung zu verlangen, muss die konkrete Durchführung diese Einzelpfade trotzdem kennen. Neue, unbekannte oder mehrdeutig zugeordnete Dateien werden sichtbar, statt automatisch passend zu gelten.

### 6.3 Wirkung von Änderungen

Änderungen an einer Regel betreffen ihre Gegenstände, Verhaltensverträge, abhängigen Bereiche, Realisierungen und Prüfungen. Der konkrete Umfang entsteht aus expliziten Modellbeziehungen und dem standgebundenen Realisierungsindex.

Zusätzliche beobachtete technische Abhängigkeiten können die Prüfung erweitern. Ein unsicherer Index darf den Umfang nicht stillschweigend verkleinern. Fehlende Zuordnungen sind ein offener Befund; Struktur allein garantiert noch keinen vollständigen Blast Radius.

## 7. Durchgängiges Bibliotheksbeispiel

Alle Festlegungen dieses Beispiels sind illustrative Annahmen des Entwurfs, keine Anforderungen an ein existierendes Bibliotheksprojekt.

### 7.1 Beschreibung

- **Zweck:** Mitglieder erhalten zeitweiligen Zugang zu Büchern; die Bibliothek hält Bestand und Nutzung nachvollziehbar.
- **Umfang des kleinen Ausgangsbeispiels:** Bestand, Mitgliedsberechtigung, Ausleihe und Rückgabe. Physischer Einkauf, Reinigung und Buchhaltung liegen zunächst außerhalb der Software. Abschnitt 7.3 greift die weiteren Überlegungen des Nutzers zur Sortimentssteuerung auf.
- **Annahme:** Nachfrage verändert sich. Eine spätere Sortimentsunterstützung könnte Nutzung beobachten, bewerten und Empfehlungen erzeugen; automatische Beschaffung ist dadurch nicht schon delegiert.
- **Begriffe:** Ein Titel kann mehrere physische Exemplare haben. Eine Ausleihe verbindet ein Exemplar, ein Mitglied, einen Beginn und eine Rückgabefrist.
- **Bereiche:** Bestand bietet die Information über Exemplare; Mitglieder bietet die Berechtigungsabfrage; Ausleihe verantwortet Ausleihvorgänge und ihre Integration.
- **Regel L-R1:** Jedes Exemplar hat höchstens eine aktive Ausleihe.
- **Regel L-R2:** Nur berechtigte Mitglieder können ausleihen.
- **Vertrag L-C1:** Eine erfolgreiche Ausleihe erzeugt genau eine aktive Zuordnung mit Frist. Ein bereits ausgeliehenes Exemplar führt zur Ablehnung und keiner zusätzlichen aktiven Ausleihe.
- **Gleichzeitigkeit L-C2:** Zwei gleichzeitige zulässige Anfragen auf ein freies Exemplar führen zu genau einer erfolgreichen Ausleihe und einer Ablehnung. Dafür wird eine atomare Entscheidung verlangt; deren technische Implementierung ist offen.
- **Qualität:** Fehler und Wiederherstellung dürfen L-R1 nicht verletzen. Genauere Zusagen zu Wiederholung und Ausfällen müssen festgelegt werden, bevor sie als abgedeckt gelten.
- **Gewählte Artefakterwartungen:** Anwendung und dauerhafte Datenhaltung realisieren Verhalten und Zustandsregeln. Tests realisieren Prüfvorgaben und prüfen benannte Aussagen. Bedienungs- und Betriebsdokumentation realisieren die jeweiligen Informationspflichten.

### 7.2 Aussagen auf Ausdrucksmittel abbilden

| Konkrete Aussage | Benötigter Inhalt | Ausdrucksmittel |
| --- | --- | --- |
| Ein Titel kann mehrere Exemplare haben. | Fachgegenstände und zulässige Beziehung. | Definitionen, Referenzbeziehung, Kardinalität. |
| Ein Exemplar hat höchstens eine aktive Ausleihe. | Zustandsabhängige Regel über mehrere Gegenstände. | Benannte Regel, Gegenstandsbezug, Bedingung und begrenzte Anzahl. |
| Ausleihe benötigt Mitgliedsberechtigung. | Informationsabhängigkeit und benötigter Vertrag. | Typisierte Beziehung und Schnittstellenvertrag. |
| Bei Ablehnung entsteht keine Ausleihe. | Fehlerergebnis und verbotene Nebenwirkung. | Verhaltensvertrag mit Nachbedingung. |
| Die Umsetzung der atomaren Entscheidung ist frei. | Bewusst delegierte technische Entscheidung. | Beschriebener Umsetzungsspielraum unter verbindlicher Invariante. |
| Ausleihregeln werden ausführbar geprüft. | Realisierungs- und Prüferwartung. | Artefaktrolle, Prüfbezug, Kriterien; konkrete Dateien im Realisierungsindex. |

### 7.3 Erweiterung: Sortimentssteuerung

Die Überlegungen des Nutzers enthalten außerdem Anschaffung und Entfernung von Büchern, Nutzungsstatistik, beschädigte und verspätete Exemplare, Kapazität und Preisbildung. Diese Inhalte stellen weitere Anforderungen an die Beschreibungssprache:

- **Beobachtung:** Nachfrage, Ausleihen, Zustand von Exemplaren und nicht erfüllte Anfragen werden mit ihrer Bedeutung und Herkunft beschrieben. Die tatsächlichen Messwerte bleiben Betriebsdaten.
- **Zielprofil und Grenzen:** Gewünschte Auswahl und Verfügbarkeit werden gegenüber Budget, Lagerkapazität und Kosten abgewogen. Messgrößen und Prioritäten müssen benannt sein; ein universelles Optimum wird nicht behauptet.
- **Bewertung:** Ein Verfahren bewertet Beobachtungen im Verhältnis zu den Zielen. Seine Regeln oder sein zulässiger Entscheidungsspielraum werden beschrieben.
- **Entscheidung und Handlung:** Das System kann Empfehlungen für Beschaffung oder Entfernung liefern. Ob es selbst bestellen darf, benötigt eine eigene Befugnis und einen beschriebenen Vertrag mit der Beschaffung.
- **Sonderfälle:** Beschädigung und verspätete Rückgabe verändern Ausleihbarkeit beziehungsweise auszulösende Verfahren nach ausdrücklich gewählten Regeln. Preis- oder Gebührenregeln sind modellierbar, wenn das Beispielprojekt sie vorsieht; konkrete Tarife werden hier nicht erfunden.

Hier entstehen zusammenhängende Beobachtungs-, Bewertungs-, Entscheidungs- und Handlungsabläufe. Daten und Verhalten allein müssen deshalb durch Ziele, Einschränkungen und Mandate ergänzt werden können.

### 7.4 Kontrast: ein kleines CLI-Werkzeug

Ein Werkzeug zur Umwandlung einer Eingabedatei benötigt womöglich keine dauerhaften Fachgegenstände, Datenbank, Weboberfläche oder mehrstufige Organisation. Beschrieben werden Eingabeform, Optionen, Transformation, Ausgabe, Fehler und Exitstatus, erlaubte Dateizugriffe, Plattformen, Verteilung und Projektregeln.

Definitionen, Verträge, Eigenschaften, Beziehungen und Regeln müssen dafür genauso verwendbar sein. Die Bibliotheksbegriffe Exemplar und Ausleihe gehören in deren projektspezifisches Vokabular, nicht in eine universelle Pflichtsprache.

## 8. Verify als Rückkopplung für die Beschreibung

### 8.1 Gedanke des Nutzers

Der Nutzer schlägt einen Ansatz ähnlich der Variation eines AI-Modells anhand von Eingaben und erwarteten Ausgaben vor: Beschreibungen verändern, realisieren und mit Verify prüfen, bis gewünschte Ergebnisse erreicht werden. Sein Gedanke „Wenn Verify sagt passt, haben wir es korrekt beschrieben“ wird hier als zu präzisierender Ansatz aufgenommen, nicht als allgemeine Korrektheitszusage.

### 8.2 Vorgeschlagene Verwendung

1. Unabhängig begründete Referenzsituationen und erwartete Ergebnisse festlegen, einschließlich Fehler- und Grenzfällen.
2. Die Situationen mit Aussagen der Beschreibung verbinden und beurteilen, ob die Erwartungen daraus folgen. Bei Prosa ist dies eine fachliche Bewertung beziehungsweise ein Review; eine formale Ableitungsprüfung ist nur für entsprechend formalisiertes Material vorgesehen. Ein solches neues Verfahren wird nicht als vorhandene Verify-Fähigkeit behauptet.
3. Geeignete Prüfungen benennen beziehungsweise durchführen lassen. Evidenz wird an ausgewählten Modellstand, Repository- beziehungsweise Kandidatenstand, Prüfdefinitionen und verwendete Werkzeuge gebunden; relevante Laufzeitbedingungen bleiben ausdrücklich. Fehlender oder ungeprüfter Umfang wird sichtbar.
4. Bei Abweichungen Ursache unterscheiden: falsche oder fehlende Beschreibung, fehlerhafte Realisierung, unzureichende Prüfung oder nicht entscheidbare Beobachtung.
5. Beschreibung nur ändern, wenn sich die geltende Definition ändern soll. Bei einem Implementierungsfehler dieselbe Definition richtig realisieren. Eine fehlgeschlagene Prüfung durch Abschwächen der Anforderung grün zu machen ist keine gelungene Beschreibung.

Diese Schritte sind ein vorgeschlagenes Verfahren. In dieser Arbeit wurden keine solchen Prüfungen oder Studien ausgeführt.

### 8.3 Bedeutung eines positiven Ergebnisses

| Fragestellung | Was ein Ergebnis beitragen kann | Was getrennt bleibt |
| --- | --- | --- |
| Ist das Modell strukturell zulässig? | Typen, Referenzen und ausdrücklich prüfbare Regeln erfüllen die definierten Bedingungen. | Ob wichtige Anforderungen fehlen oder die Fachbedeutung richtig ist. |
| Entspricht die Realisierung den beschriebenen Aussagen? | Benannte Prüfungen und Reviews liefern Evidenz zu konkreten Aussagen und einem festen Stand. | Nicht untersuchte Aussagen, Grenzen oder Laufzeitbedingungen. |
| Beschreibt das Modell das richtige System ausreichend? | Referenzsituationen, Gegenbeispiele und Abdeckungsfragen können Lücken und Fehlinterpretationen aufdecken. | Allgemeine Vollständigkeit, Nutzerwert und tatsächlicher Gesamterfolg. |

Ein Prüforakel, das nur aus derselben möglicherweise falschen Beschreibung erzeugt wird, kann deren Fehler mit übernehmen. Referenzerwartungen brauchen deshalb eine eigenständige Begründung außerhalb der Kandidatenbeschreibung, etwa durch Nutzer, Fachwissen oder zuvor festgelegte Fälle. Das macht die Erwartungen nicht automatisch richtig und ersetzt keinen unabhängigen Reviewer. Tests sind selbst Realisierungen und werden gegen ihre behaupteten Aussagen geprüft.

Beispiel: L-C2 verlangt bei zwei gleichzeitigen Anfragen genau eine erfolgreiche Ausleihe. Ein Test mit zwei aufeinanderfolgenden Aufrufen kann diese Gleichzeitigkeit nicht allein belegen. Der Prüfumfang muss die konkrete Aussage treffen. Ein grüner Lauf zu L-C2 prüft außerdem nicht automatisch Datenschutz oder Wiederherstellung.

## 9. Prüfbarkeit und ausreichende Beschreibung

### 9.1 Deterministisch prüfbare Struktur

Als vorgeschlagene Fähigkeiten einer gemeinsamen Strukturprüfung kommen Identitäten, Typen, erforderliche Eigenschaften, Referenzen, Beziehungssignaturen, Zugehörigkeit und ausdrücklich definierte endliche Einschränkungen in Betracht. Diese Liste ist kein aktueller Core-Funktionsnachweis. Projektspezifische oder technische Regeln können spezialisierte Werkzeuge benötigen.

Lesbare Prosa bleibt wichtig für Bedeutung, Begründung und differenziertes Verhalten. Ein strukturierter Platz für Prosa macht deren Inhalt nicht automatisch formal entscheidbar. Die Beschreibungssprache muss kennzeichnen können, welche Zusagen strukturell, durch technische Prüfungen oder durch begründete Bewertung untersucht werden.

Primitive Elemente, Kombination und Abstraktion sind nützliche Prinzipien aus Programmiersprachen. Eine Turing-vollständige ausführbare Modellierungssprache wird damit nicht vorausgesetzt. Welche zusätzliche Formalisierung nötig ist, wird an wiederkehrenden Ausdruckslücken beurteilt.

### 9.2 Abdeckungsfragen für einen Projektstand

- Sind Zweck, Grenze, Annahmen und relevante Akteure nachvollziehbar?
- Sind Begriffe und Informationen eindeutig genug, um Verhalten und Regeln zu verstehen?
- Sind relevante Fähigkeiten, Fehlerfälle und Zustandsänderungen beschrieben?
- Sind Verträge, Abhängigkeiten und übergreifende Einschränkungen erkennbar?
- Sind wichtige Qualitäts- und Betriebsanforderungen einschließlich Datenänderungen und Wiederherstellung berücksichtigt?
- Sind Projektverfahren, Verantwortungen und delegierte Freiheiten beschrieben?
- Haben relevante Artefaktfamilien nachvollziehbare Realisierungserwartungen?
- Ist für den konkreten Stand jede Datei eingeordnet oder ausdrücklich ausgenommen?
- Sind Prüfungen auf benannte Aussagen bezogen und ihre Grenzen sichtbar?
- Sind ungeklärte entscheidungsrelevante Punkte sichtbar statt durch Agentenannahmen verdeckt?

Diese Fragen geben einen Reviewmaßstab. Sie sind kein durch die Dokumentation bereits bewiesener Vollständigkeitsalgorithmus.

## 10. Verhältnis zu Markitects bestehendem Stand

Referenz dieses Entwurfs: Worktree `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`, Branch `docs/product-model-concepts`, HEAD vor der Ausarbeitung `e25a00f502d5ce76a047a3cd7dd9f8405d7ea85f`, beim Start sauber. Der Produktquellstand stammt aus `origin/main` bei `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`; die späteren lokalen Commits betreffen bisher nur diese Gesprächsdokumentation.

| Vorhandener dokumentierter Vertrag / Entscheidung | Einordnung des Entwurfs |
| --- | --- |
| [Vision](../../vision.md#markitect-in-brief): kanonisches Modell und delegierte Realisierung sind der Produktkern. | Der Entwurf beschreibt den Inhalt und die Ausdrucksmöglichkeiten dieses Modells; er macht daraus keinen bloßen Dokumentationslinter. |
| [DEC-005](../../concepts/register.md#dec-005-model-granularity-is-the-authors-responsibility): Granularität wird vom Autor gewählt. | Bereiche, Aussagen und Verträge dürfen unterschiedlich fein beschrieben werden. |
| [DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored): konkrete Dateiabdeckung ist erforderlich; eine Datei kann mehrere Definitionen realisieren. | Ohne alle Dateien im Modell aufzuzählen muss die konkrete Realisierung trotzdem vollständig eingeordnet werden. Die vorgeschlagene Ermittlung dieser Zuordnung ist noch kein Implementierungsnachweis. |
| [Engineering constitution am ursprünglichen Main-Stand](https://github.com/Glacius-Labs/Markitect/blob/ecc69b0ad99236a5ddd02b85bf9ba656a0000f15/docs/engineering-constitution.md#canonical-reset-source-boundary): minimaler struktureller Core, getrennte Schema- und Projection-Module. | Der inhaltliche Begriffskatalog ist kein Auftrag, alle Begriffe in den Core aufzunehmen. Die bestehenden Schema/Kind/Property/Definition-Grundlagen sind auf Ausdrucksfähigkeit zu vergleichen. |
| [DEC-003](../../concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept) und [Architektur](../../architecture.md): Strukturprüfung und begrenzte Evidenz sind keine semantische Wahrheitsgarantie. | Verify wird als Rückkopplung mit benanntem Prüfumfang behandelt. |
| [DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis): Manager-Realisierung und spätere Government-Modellpflege sind getrennte Achsen. | Mandate und Eskalation sind beschreibbare Inhalte; eine bestimmte Regierung wird nicht zur Voraussetzung der Modellierung. |

Die Tabelle vergleicht mit gelesenen Dokumenten, nicht mit einer neu durchgeführten vollständigen Codeprüfung. Es gibt keinen neuen Testbefund, Providerlauf, Machbarkeitsnachweis oder Freigabestatus. Die bestehende Implementierung wird in einer späteren, gesondert beauftragten Betrachtung an einem festen Stand verglichen.

## 11. Offene Entscheidungen für die gemeinsame Prüfung

1. Ist die Arbeitsdefinition des geltenden Systemstands und des Projektumfangs passend?
2. Reichen die vorgeschlagenen Inhalts- und Ausdrucksfamilien aus? Wo fehlen konkrete Beschreibungen?
3. Welche Definitionen, Aussagen und Beziehungen müssen einzeln adressierbar sein?
4. Welche Beziehung soll zwischen Gliederung, Namespace, Dateipfad und Verantwortungsstruktur gelten?
5. Welche Regeln und Verhaltensverträge sollen formal ausdrückbar sein, und wo bleibt Prosa mit spezialisierter Prüfung ausreichend?
6. Wie werden Umsetzungsspielräume, lokale Prioritäten, Regelkonflikte und Ausnahmen verständlich dargestellt?
7. Wie entsteht ein zuverlässiger Realisierungsindex, ohne eine zweite manuell gepflegte Spezifikation zu schaffen?
8. Welche unabhängigen Referenzsituationen eignen sich zur Verbesserung der Beschreibung mit Verify?

Die nächste Verfeinerung sollte konkrete Ausdruckslücken am Bibliotheks- und CLI-Beispiel lösen. Eine Serialisierung, eine neue Core-Architektur oder die Umsetzung in Markitect wird erst aus angenommenen Entscheidungen abgeleitet.

## 12. Theoretischer Fundus und Quellen

Die Quellen dienen zur Orientierung. Die vorgeschlagene Übertragung auf Markitect ist eine Ableitung dieses Entwurfs und kein von den Quellen bestätigter Produktnachweis.

- [SICP, Abschnitt 1.1](https://web.mit.edu/6.001/6.037/sicp.pdf): elementare Ausdrücke, Kombination und Abstraktion; Ausgangspunkt für die Fragen nach Grundbausteinen und Zusammensetzung.
- [DDD Reference von Eric Evans](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf): gemeinsame Sprache in abgegrenzten Kontexten; Orientierung für Bedeutung und kohäsive Begriffsbereiche.
- [Clean Architecture von Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html): Quellcodeabhängigkeiten nach innen und Trennung vom Laufzeitkontrollfluss; Orientierung für unterschiedliche Beziehungstypen.
- [C4-Systemkontext](https://c4model.com/diagrams/system-context): Nutzer und benachbarte Systeme als verständlicher Ausgangspunkt einer Architekturbeschreibung.
- [arc42-Übersicht](https://arc42.org/overview/): Kontext, Bausteine, Laufzeit, Bereitstellung, übergreifende Konzepte, Qualität und Risiken als ergänzende Abdeckungsfragen.
- [Aspect-Oriented Programming, Kiczales et al.](https://www.cs.ubc.ca/~gregor/papers/kiczales-ECOOP1997-AOP.pdf): Eigenschaften, die mehrere Komponenten schneiden; Orientierung für explizite Geltungsbereiche.
- [W3C SHACL](https://www.w3.org/TR/shacl/): explizite Bedingungen und Validierung von Daten in Graphen; ein Vergleichsbeispiel, keine Auswahl von RDF/OWL/SHACL als Markitect-Technologie.

Organisations-, Unternehmens-, Militär- und Regierungsstrukturen sowie Design Patterns bleiben weitere Suchräume für konkrete Probleme. Wiederkehrende Prinzipien können eine Hypothese stärken; ihre Eignung hängt weiterhin von Aufgabe, Informationslage, Befugnissen, Kommunikation und Fehlerfolgen ab.

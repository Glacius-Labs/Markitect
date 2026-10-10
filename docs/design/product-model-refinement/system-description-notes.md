# Arbeitsnotiz: Ein Softwaresystem beschreiben

Datum: 10. Oktober 2026. Quelle: Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`.

Ergänzt am 11. Oktober 2026: ausdrücklicher Auftrag, zuerst die zentrale Frage und danach den daraus abgeleiteten Arbeitsstand festzuhalten.

Der Nutzer hat ausdrücklich darum gebeten, den Ansatz mit einem klassischen Beispiel und allgemeinen Beschreibungsfragen als Notiz festzuhalten. Diese Notiz bewahrt diesen Arbeitsansatz und die anschließend eingebrachten Gedanken. Die genannten Begriffe sind Kandidaten, keine beschlossene Ontologie, Modelltypen oder Implementierungsvorgaben. Das [vereinbarte Grundmodell](README.md) bleibt getrennt davon; dort stehen auch Worktree, Branch und Ausgangsrevision.

## Leitfrage

Die zentrale Frage des Nutzers lautet:

> Wie beschreibt man ein Repository für ein Softwaresystem?

Weitere Formulierungen des Nutzers:

- Woraus besteht ein Repository für ein Software System?
- Was macht ein Software System aus?
- Was ist ein Software System? Wie lässt es sich beschreiben? Reicht die Beschreibung aus um daraus ein Repository dafür aufzubauen?

Alle bisherigen Überlegungen, Beispiele, Begriffe und C#-Skizzen sollen dieser Frage dienen. Ihre Eignung wird daran beurteilt, ob sie zu einer ausreichenden Beschreibung beitragen. Die Festlegung der Leitfrage ist der gemeinsame Arbeitsauftrag; die folgenden Ableitungen sind der auf ausdrücklichen Wunsch festgehaltene Arbeitsstand.

## Abgeleiteter Arbeitsstand

### Drei Seiten derselben Untersuchung

| Frage | Gegenstand der Untersuchung |
| --- | --- |
| Was macht ein Softwaresystem aus? | Seine Bedeutung, Fähigkeiten, Strukturen und Regeln. |
| Woraus besteht sein Repository? | Artefakte, die diese Inhalte realisieren und Entwicklung sowie Betrieb ermöglichen. |
| Wie beschreibt man es ausreichend? | Die Festlegungen, die ein Umsetzer für eine passende Realisierung benötigt. |

Das Arbeitsziel ist eine Beschreibung eines Softwareprojekts, die für einen erklärten Umfang ausreichend ist, um daraus ein passendes Repository aufzubauen, ohne jede Datei einzeln vorgeben zu müssen.

### Vorläufige Definition und Beschreibungsfragen

> Ein Softwaresystem ist eine abgegrenzte, softwarebasierte Lösung, deren zusammenwirkende Bestandteile bestimmte Fähigkeiten unter bestimmten Regeln und Bedingungen bereitstellen.

Um es zu beschreiben, müssen folgende Fragen beantwortbar sein:

- **Wofür existiert es?** Problem, Nutzer, Nutzen und Systemgrenze.
- **Wovon handelt es?** Relevante Begriffe, Informationen, Zustände und Beziehungen.
- **Was tut es?** Fähigkeiten, Abläufe, Ergebnisse, Fehler und Zustandsänderungen.
- **Was muss dabei gelten?** Invarianten, Qualitätsanforderungen und Einschränkungen.
- **Wie wirken seine Bestandteile zusammen?** Gliederung, Schnittstellen und Abhängigkeiten.
- **Unter welchen Bedingungen funktioniert es?** Umgebung, Konfiguration, externe Systeme und Betriebsbedingungen.

Diese Fragen sind eine Orientierung für die Untersuchung. Welche davon eigene Modellbegriffe benötigen, soll aus konkreten Beschreibungen abgeleitet werden.

### Vom System zum Repository

Das Repository enthält Artefakte mit unterschiedlichen Beiträgen zur Beschreibung: Code realisiert Verhalten und Datenstrukturen; Tests realisieren Prüfungen; Konfiguration legt Betriebsparameter fest; Infrastruktur und Pipelines ermöglichen Bereitstellung und Verfahren; Dokumentation vermittelt Wissen; Skills und Arbeitsanweisungen beschreiben die Arbeit am Projekt.

Eine Systembeschreibung bestimmt noch nicht alle Eigenschaften eines Repositorys. Hinzukommen können technische Realisierungsvorgaben und Regeln für die Projektarbeit. Dieselbe Bibliotheksfunktion kann beispielsweise als Desktopanwendung, Webanwendung oder CLI realisiert werden. Eine solche Entscheidung muss festgelegt oder ausdrücklich freigegeben sein.

### Vorgeschlagener Arbeitsmaßstab für eine ausreichende Beschreibung

> Eine Beschreibung ist für ihren erklärten Umfang ausreichend, wenn ein kompetenter Umsetzer daraus eine passende Realisierung entwickeln kann, ohne wesentliche fachliche oder projektrelevante Entscheidungen unbemerkt selbst erfinden zu müssen.

Unterschiedliche Implementierungen können dieselbe Beschreibung erfüllen. Erkennbar sein muss:

1. Was verbindlich festgelegt ist.
2. Was bei der Umsetzung entschieden werden darf.
3. Woran eine passende Realisierung erkannt wird.

Die ausreichende Beschreibung ist damit auf Umfang und bewusst zugelassene Spielräume bezogen. Dieser Maßstab ist noch kein Nachweis, dass eine konkrete Beschreibung vollständig ist.

### Vorgehen zum Finden fehlender Ausdrucksmöglichkeiten

1. Eine gewöhnliche, zusammenhängende Beschreibung der Bibliothekssoftware schreiben.
2. Die Perspektive eines Umsetzers einnehmen: Kann daraus eine passende Lösung entstehen? Welche Entscheidung fehlt? Ist sie bewusst frei?
3. Typische Repository-Artefakte betrachten: Welche Aussage begründet das Artefakt, und welche Information benötigt es?
4. Aus gefundenen Lücken benötigte Aussagen, Begriffe und Ausdrucksmöglichkeiten ableiten.

„Die Software verleiht Bücher“ lässt beispielsweise offen, ob Titel oder Exemplare verliehen werden, wer ausleihen darf, wann ein Exemplar verfügbar ist, wie konkurrierende Anfragen behandelt werden, wann eine Ausleihe endet und welche Informationen erhalten bleiben. Die Antworten liefern konkrete Beschreibungsbedarfe.

Der [C#-Entwurf](csharp-model-design.md) ist eine erste Hypothese zur Organisation solcher Aussagen. Verify könnte später Rückmeldung über benannte Anforderungen und Prüfungen liefern. Ob die Beschreibung die richtigen und relevanten Anforderungen enthält, muss zusätzlich untersucht werden. Die Leitfrage betrifft deshalb sowohl die Ausdrucksfähigkeit des Modells als auch die Angemessenheit und ausreichende Festlegung der jeweiligen Beschreibung.

### Herkunft dieser Ergänzung

- Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Dokumentationsstand vor dieser Ergänzung: `4c2792e35deb4831256199ad2ee928a3e773ef76`; Worktree beim Beginn sauber.
- Produktbasis bleibt `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Quelle: Klarstellung und Festhalteauftrag des Nutzers im oben genannten Produktgespräch; die Ableitungen stammen aus der unmittelbar vorausgehenden Antwort.
- Umfang: Dokumentationsnotiz und Navigation; keine Änderung an Produktcode oder Modellverträgen, keine Tests oder Studien.

## Arbeitsansatz

- Ein bekanntes, gewöhnliches Softwaresystem als Beispiel verwenden, vorerst eine fiktive Bibliotheksverwaltung. Markitect selbst ist als Beispiel zu stark mit dem zu konzipierenden Werkzeug verwoben.
- Allgemeine Fragen über ein System stellen und sie am Beispiel beantworten, ohne Code oder Repositorydateien sehen zu müssen.
- Die sechs vorgeschlagenen Bereiche als Orientierung im Hintergrund verwenden: Zweck und Umfang; Fachlichkeit und Verhalten; Struktur und Architektur; Qualität und Randbedingungen; Bereitstellung und Betrieb; Arbeitsweise und Verantwortung. Ihre genaue Aufteilung bleibt vorläufig.
- Git und Work Items vorerst ausklammern. Der Nutzer bezeichnet den Gegenstand des Modells als IST-Zustand des Systems.
- DDD, Clean Architecture, Spec Driven Development, Aspect Oriented Development, Organisationsstruktur, Dependency Management, Graphentheorie, Ontologien, Compiler, Design Patterns und Regierungsstrukturen als Fundus für mögliche Lösungen bereithalten. Die Bibliotheksbeschreibung soll nicht von vornherein deren Fachbegriffe übernehmen müssen.
- Wiederkehrende Probleme und Lösungsprinzipien in Organisationen, Militär, Regierungen und Unternehmen können nach Ansicht des Nutzers ein Hinweis auf Übertragbarkeit sein. Die jeweilige Passung ist zu untersuchen; Wiederholung allein ist kein Nachweis für Markitect.

## Beschreibungsfragen und Beispiel

Die Bibliothek und ihre Antworten sind ein fiktives Gedankenbeispiel, keine erhobenen Anforderungen.

| Frage | Beispielhafte Antwort | Begriffskandidaten |
| --- | --- | --- |
| Welches Problem wird gelöst und welches Ergebnis wird angestrebt? | Verfügbarkeit und Verbleib von Exemplaren zuverlässig nachhalten. | Problem, Ziel |
| Für wen existiert das System und wo endet sein Umfang? | Mitglieder und Mitarbeiter nutzen es; Einkauf und Buchhaltung liegen außerhalb des Umfangs. | Akteur, Rolle, Systemgrenze |
| Welche Dinge und Begriffe gibt es in seiner Welt? | Titel, Exemplare, Mitglieder, Ausleihen; ein Titel kann mehrere Exemplare haben. | Fachbegriff, Gegenstand, Beziehung |
| Was können Nutzer damit tun? | Suchen, ausleihen, zurückgeben, verlängern; Mitarbeiter erfassen Exemplare. | Fähigkeit, Anwendungsfall |
| Welche Regeln gelten? | Ein Exemplar hat höchstens eine aktive Ausleihe. | Regel, Bedingung, Invariante |
| Welche Abläufe, Zustände und Ergebnisse gibt es? | Ausleihe ordnet ein Exemplar einem Mitglied zu und setzt das Rückgabedatum; belegte Exemplare führen zur Ablehnung. | Ablauf, Zustand, Übergang, Ergebnis |
| Wie ist das System aufgebaut und wie arbeiten seine Teile zusammen? | Oberfläche, Bestands- und Ausleihfunktionen sowie dauerhafte Datenhaltung. | Baustein, Schnittstelle, Abhängigkeit |
| Welche Eigenschaften muss es erfüllen und wie werden sie geprüft? | Keine doppelte Ausleihe bei gleichzeitigen Anfragen; Mitgliederdaten nur für Berechtigte. | Qualitätsanforderung, Einschränkung, Prüfkriterium |
| Unter welchen Bedingungen wird es betrieben? | Festgelegte Umgebung, Datensicherung und Wiederherstellungsverfahren. | Umgebung, Konfiguration, Betriebsverfahren |
| Wie wird es gepflegt und wer trägt welche Verantwortung? | Definierte Zuständigkeiten für Fachregeln, Umsetzung, Prüfung und Betrieb. | Verantwortung, Befugnis, Verfahren |

Vorläufige Ableitungen aus dem Beispiel:

- Problem, Ziel und Fähigkeit erfüllen verschiedene Aufgaben: Anlass, angestrebtes Ergebnis und verfügbare Leistung.
- Die Unterscheidung zwischen Titel und Exemplar macht eine Ausleihregel präzise formulierbar.
- Regeln beschreiben geltende Bedingungen; Abläufe beschreiben Handlungen und Zustandsänderungen einschließlich Fehlerfällen.
- Aufbau, Verhalten und Verantwortung brauchen unterscheidbare Beziehungen.
- C4 ist ein möglicher Ansatz für Architekturbeschreibungen; der Systemkontext beschreibt das System, seine Nutzer und benachbarte Systeme. [Offizielle C4-Beschreibung](https://c4model.com/diagrams/system-context).

## Neuer Gedanke des Nutzers: Programmiersprachen

Der Nutzer fragt, woraus eine Programmiersprache grundsätzlich besteht und was daraus für die Beschreibung von Softwaresystemen abzuleiten ist. Ausgangspunkt ist die Ausdrucksstärke Turing-vollständiger Sprachen.

Offene Untersuchung: Welche Rolle spielen elementare Ausdrücke, Kombination, Abstraktion, Namen, Typen, Geltungsbereiche, Daten und Verhalten? Welche dieser Mechanismen helfen einer gut verständlichen Systembeschreibung? Es ist keine Entscheidung getroffen, dass das kanonische Modell selbst eine Turing-vollständige Sprache werden soll.

## Neuer Gedanke des Nutzers: Hierarchie und Graph

Der Nutzer sieht ein möglicherweise gemeinsames Prinzip:

- Softwarearchitektur soll kohäsive Module mit geordneten Abhängigkeiten bilden.
- Organisationen strukturieren Management und Verantwortungsbereiche hierarchisch.
- Namespaces und Ordner organisieren Code häufig in einer Baumstruktur; seine Abhängigkeiten bilden daneben einen Graphen.
- Dokumentation ist häufig hierarchisch abgelegt und hat zugleich Querverweise.
- Die vermeintliche Diskrepanz zwischen hierarchischen Verantwortungsbereichen und einem Modellgraphen könnte deshalb demselben Muster folgen.

Status: zu untersuchende Analogie des Nutzers. Noch offen sind die genaue Beziehung zwischen Gliederung, Verantwortung und Abhängigkeiten sowie die Frage, welche Beziehungen aus der Hierarchie abgeleitet und welche ausdrücklich deklariert werden müssen. Eine gemeinsame oder zwingend identische Hierarchie ist noch nicht beschlossen.

## Fortsetzung: Vorgehen und Verify

Der Nutzer hat ausdrücklich gebeten, das anschließend besprochene kombinierte Vorgehen zu behalten: Umfang festlegen, gewöhnliche Dokumentation schreiben, Artefaktfamilien auf Lücken prüfen, aus Aussagen Begriffe und Ausdrucksmittel ableiten und die Beschreibung anhand von Abläufen sowie einem Kontrastbeispiel prüfen. Das Vorgehen steht in Abschnitt 2 des [beauftragten Entwurfs](canonical-model-draft.md).

Als weiteren Gedanken brachte er die Variation einer Beschreibung anhand von Eingaben und erwarteten Ergebnissen ein, ähnlich einem AI-Trainingsansatz. Markitect Verify könnte dafür eine Rückkopplung liefern. Der Entwurf präzisiert diesen Vorschlag in Abschnitt 8; ein positives Ergebnis wird auf die tatsächlich geprüften Aussagen und den jeweiligen Stand begrenzt.

# Arbeitsnotiz: Ein Softwaresystem beschreiben

Datum: 10. Oktober 2026. Quelle: Produktgespräch `01a121f1-b948-7050-ae5d-9921b99db9c0`.

Der Nutzer hat ausdrücklich darum gebeten, den Ansatz mit einem klassischen Beispiel und allgemeinen Beschreibungsfragen als Notiz festzuhalten. Diese Notiz bewahrt diesen Arbeitsansatz und die anschließend eingebrachten Gedanken. Die genannten Begriffe sind Kandidaten, keine beschlossene Ontologie, Modelltypen oder Implementierungsvorgaben. Das [vereinbarte Grundmodell](README.md) bleibt getrennt davon; dort stehen auch Worktree, Branch und Ausgangsrevision.

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

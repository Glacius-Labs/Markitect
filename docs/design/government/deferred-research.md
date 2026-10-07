# Vorgemerkte Untersuchungsthemen

Diese Liste bewahrt Ideen außerhalb der laufenden Arbeitspakete. Sie erweitert weder Implementierungsaufträge noch Versuche, Ressourcen oder Veröffentlichungsbefugnisse. Die aktive Reihenfolge und konkrete Auswahl verantwortet [die Koordination](coordination.md).

## SAT-inspirierte Prinzipien für das kanonische Modell

Vorgemerkt auf ausdrücklichen Nutzerwunsch am 7. Oktober 2026. Gemeint sind zunächst Ideen und allgemeine Prinzipien des Constraint- und Erfüllbarkeitsdenkens, nicht die Auswahl oder Integration eines bestimmten Solvers. Status: Hypothese für spätere Untersuchung; kein Wirksamkeitsnachweis.

**Leitfrage:** Hilft ein expliziter zulässiger Lösungsraum dabei, Nutzerabsicht besser auszudrücken, Widersprüche früher zu erkennen und autonome Änderungen nachvollziehbar innerhalb verbindlicher Grenzen zu halten?

Zu bewahrende Ideen:

- Das gedankliche Modell benennt Ziele, Begriffe, Beziehungen, Annahmen und Bedingungen. Akzeptiertes Soll, Beobachtungen, Unsicherheit und Nachweise bleiben unterscheidbar; `purpose` und Herkunft erklären die Bedingungen.
- Verbindliche Bedingungen und verhandelbare Präferenzen werden getrennt. Agenten haben Gestaltungsfreiheit innerhalb der gültigen Kombination aller Bedingungen. Eine zulässige Lösung ist nicht automatisch die bevorzugte Lösung.
- Lösungsvorschlag und Prüfung bleiben getrennte Aufgaben. Ein Vorschlag benennt, welche Bedingungen durch welche Belege erfüllt werden; fehlende Erkenntnis wird nicht als Zustimmung behandelt.
- Konflikte sollen als verständliche Kombination widersprechender Bedingungen und Annahmen sichtbar werden. Eine möglichst kleine hilfreiche Erklärung ist ein Untersuchungsziel, keine garantierte minimale Konfliktmenge.
- Zuständige Ebenen können Konflikte oder Modelllücken innerhalb bestehender Befugnis auflösen. Regeln werden dabei nicht still abgeschwächt; Modelländerungen werden begründet, versioniert und erneut geprüft.
- Verantwortungsbereiche entwickeln Lösungen, Ressorts liefern fachliche Bedingungen und Bewertungen, die Elternintegration prüft das Zusammenspiel. Ein Solver würde weder notwendige fachliche Nachweise noch die explizite finale Zustimmung aller ausgewählten Ressorts ersetzen.

Beispiel für einen Modellkonflikt: Jede Bestellung soll auch ohne Netzwerk angenommen werden können; gleichzeitig verlangt jede Annahme eine vorherige synchrone Freigabe vom zentralen Server. Unter diesen Annahmen fehlt eine gemeinsame Lösung. Die zuständige Ebene müsste die beabsichtigte Bedeutung klären, statt Agenten weiter an einer widersprüchlichen Vorgabe implementieren zu lassen.

Mögliche spätere Anwendungen sind Konsistenzprüfungen im Modell sowie die Suche nach zulässigen Zuständigkeits-, Delegations- oder Arbeitsplänen. SAT, SMT oder Constraint Programming wären mögliche technische Werkzeuge; eine Bibliothek ist nicht ausgewählt. Einfache Referenz-, Ownership- und Graphprüfungen können weiterhin mit direkten deterministischen Verfahren auskommen.

Grenzen: Formal geprüft wird nur die tatsächlich kodierte Aussage samt Annahmen. Ein konsistentes Modell kann die Nutzerabsicht falsch wiedergeben; eine erfüllbare Spezifikation beweist weder Codekonformität noch gute Architektur. Unbekannt, nicht formalisiert oder ein ergebnislos abgebrochener Prüflauf sind keine positive Erfüllung. Nicht jede qualitative Priorität lässt sich sinnvoll formalisieren.

Späterer Einstieg: Zuerst anhand realer Modell- oder Koordinationskonflikte aus der bestehenden Untersuchung bewerten, ob die Prinzipien allein nützen. Nur bei einem konkreten Bedarf einen kleinen Vergleich mit den bisherigen direkten Prüfungen entwerfen. Kriterien wären korrekt erkannte und übersehene Konflikte, Fehlalarme, Verständlichkeit der Erklärung, Aufwand für Formalisierung und Pflege sowie Laufzeit. Keine automatische Ausweitung der laufenden Studie oder zusätzliche Integration; Nutzen und Komplexität bleiben offen.

## Jev / TypeSafe AI

Das bereits vorgemerkte Thema bleibt erhalten und nachrangig. Herkunft ist die [Government-Assessment-Notiz](../government-of-coding-agents-assessment.md); spätere direkte Nutzeraufträge und die aktive Koordinationsakte bestimmen die Reihenfolge. Die mögliche lesende Untersuchung eines externen Prüfers für Regel-Aussage-Beleg-Beziehungen ist weiterhin von einer Integration zu unterscheiden. Dieser Eintrag startet keine Untersuchung, kostenpflichtigen Aufrufe oder Produktänderung.

Die SAT-Idee steht neben Jev als weitere spätere Option. Zwischen beiden wird hier keine neue Reihenfolge festgelegt. Zuerst laufen die bereits vereinbarten Produkt- und Vergleichsarbeiten weiter; beim späteren Forschungs-/Produktentscheid werden die vorgemerkten Themen erneut priorisiert.

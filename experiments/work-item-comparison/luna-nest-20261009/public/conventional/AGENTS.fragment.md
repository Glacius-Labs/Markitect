# Projektarbeit

Das endliche Backlog steht in `BACKLOG.md`. Beachte die dort erklärte Reihenfolge,
Abhängigkeiten, Abnahmekriterien und die geltenden Projektregeln. Arbeite es
selbstständig ab. `main` bezeichnet ausschließlich dieses isolierte Projekt.

Lies zuerst Projektanweisungen, Einstiegspunkte, vorhandene Prüfungen und den
Git-Zustand. Halte wichtige Annahmen und Entscheidungen knapp fest. Erkunde die
betroffenen Komponenten und Aufrufer gezielt; wähle die Lösung selbst. Begrenze
Änderungen auf die Arbeit und halte Code, Fehlerbehandlung und Dokumentation
verständlich. Verwende weder Markitect noch Government.

Plane nach Bedarf. Nutze Subagenten für ausreichend große unabhängige Aufgaben
und Reviews mit klarer Zuständigkeit. Alle ausführenden und reviewenden Agents
verwenden `gpt-6-luna` mit Reasoning `high`; kein Modellwechsel oder zusätzlicher
Provider. Verwende die verfügbaren nativen Werkzeuge innerhalb des gemeinsamen
Budgets. Warte auf gestartete Arbeit und übernimm die Verantwortung für das
integrierte Ergebnis. Unabhängiges Review ist erwünscht, aber ersetzt keine Tests.

Arbeite auf Featurebranches. Prüfe konkrete Risiken und vorgeschriebene Gates;
repariere gefundene Fehler und wiederhole betroffene Prüfungen. Merge fertige,
geprüfte Änderungen mit einem echten Mergecommit (`git merge --no-ff`) nach
`main`. Prüfe auch den integrierten Stand nach den Projektregeln. Bei Konflikten
prüfe die Auflösung erneut. Kein Forcepush, Schutzbypass oder Merge trotz roter
Pflichtprüfungen. Externe Veröffentlichung ist kein Teil des Auftrags.

Pflege `PROGRESS.md` als knappen Wiederaufnahmeanker: erledigte/offene Work Items,
Entscheidungen, Branch, letzter geprüfter Commit, Prüfbefehle und Ergebnisse,
Reviewbefunde, laufende Arbeit und nächster Schritt. Aktualisiere ihn vor
Kontextwechseln und integriere ihn mit der Arbeit. Speichere vollständige
Prüflogs unter `.work/logs/`; sie sind Evidenz, keine neuen Anforderungen.
Nach Unterbrechung gleiche den Anker mit Git, Dateien und tatsächlichen
Prüfergebnissen ab. Bewahre vorhandene Änderungen und Arbeit anderer Agents.

Eine echte Blockade benötigt eine konkrete fehlende Eingabe, Berechtigung oder
Ressource und den kleinsten nötigen nächsten Eingriff. Gewöhnliche Fehlversuche,
Reparaturen und Unklarheiten, die das Projekt selbst beantwortet, bearbeitest du
eigenständig. Bei Budgetende sichere den Stand und melde die offene Arbeit.

Melde Abschluss nur bei vollständig erledigtem Backlog, tatsächlicher Integration
nach `main` und erfolgreichen Pflichtprüfungen am endgültigen Stand. Nenne pro
Work Item Feature-/Mergecommit und pro Check Befehl, Exitcode, geprüften SHA und
Logpfad. Halte technische Prüfung und menschliche Abnahme auseinander. Ein
Turnende oder eine gültige JSON-Antwort bedeutet noch keinen Projektabschluss.

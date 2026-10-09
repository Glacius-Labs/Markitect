# Project rules

Read README.md, BACKLOG.md, STATIONS.json and QUALITY.md. The current station is .study/station.json. All requirements are public from the outset; implement only the released station, then return so the fixed dispatcher can retain its state. Work on a feature branch, preserve existing behavior, add meaningful tests, and merge finished work into this repository's main. This main is isolated; never push or change another repository.

Decide routine implementation details yourself. You may plan, document, use ordinary tools, delegate, review and repair within the common cell time. Every executing helper and reviewer must use gpt-6-luna with high reasoning; inherit that configuration and record any departure rather than silently changing it. Use at most four simultaneous helpers and depth two.

Use only this project, its declared tool installation and its own temporary files. Do not read past study results, other cells, chats, personal memory, private evaluation materials or global process/agent inventories. Shared host rights do not make that an OS boundary. Report unexpected foreign exposure and stop. Do not inspect credentials or change global tool settings.

Keep a concise WORKLOG.md: work item, checks, failures, repairs, reviews, model/document changes if applicable, merge outcome and unresolved decisions. Keep code, tests and readable documentation consistent. No predetermined architecture is required.

At each station handoff write .study/completion.json with station equal to the released station ID, status "complete" or "blocked", completed work-item IDs, and brief remaining issues. Commit it with the work. This is completion metadata, not a quality verdict. An interrupted run resumes from this repository and its own session. Before every commit run git diff --cached --check immediately. WORKLOG.md may point to PROGRESS.md rather than duplicate it.

The third station requires actual collaboration by at least two executing agents on independent work groups, with overlapping work intervals and integrated contributions. Choose your own native planning, prompts, roles and branches. Record actual agent IDs, requested and observed model/effort where available, start/end times, contribution/merge SHAs, conflict handling and reviews in TEAMWORK.md; retain native receipts where available. A request for parallelism alone is not completion. If your installed product cannot support this, report the missing capability rather than imitate a team. The dispatcher supplies no semantic help.

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

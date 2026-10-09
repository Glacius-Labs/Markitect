# Markitect: Entwicklung des Produkts und der Arbeit

**Rückblick, Stand der Quellen:** 9. Oktober 2026. Der kanonische öffentliche Quellstand wurde gegen das Live-Remote bestätigt: `origin/main = 5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e`. Neuere Kandidaten auf Feature-Branches sind getrennt aufgeführt. Zeitangaben nennen Berlinzeit oder UTC ausdrücklich. Branch- und Teststände sind Momentaufnahmen, keine stillschweigende Aussage über den heutigen Zustand nach diesem Zeitfenster.

Die Frühgeschichte ab dem ältesten erreichbaren Produktcode-Commit, die Releasefolge v0.1–v0.13 und der frühe Nutzersteuerungsverlauf stehen in [der früheren Historie](early-history-2026-08-22-to-2026-10-06.md).

## Der längere Produktbogen

Markitect hat sich nicht als einzelner Markdown-Generator entwickelt. Die kanonischen Dokumente auf Main beschreiben das Produkt als Modell des gewünschten Engineering-Systems: Menschen definieren dauerhafte Absicht und Grenzen; Markitect ordnet Struktur und Auswirkungen; Agenten passen explizit zugeordnete Realisierungen an; getrennte Prüfschritte bewerten die Resultate. Die Leitformel lautet: „Define the desired project world once. Reconcile its representations without losing intent.“

Die Release-Geschichte in `docs/architecture.md` und `docs/implementation-plan.md` zeigt den Aufbau in Stufen:

- **v0.10.0** etablierte ein generisches kanonisches Engineering-Modell.
- **v0.11.0** erweiterte die modellierten Ergebnisse um Policy-Resultate, Ausnahmen und Entdeckung.
- **v0.12.0** begrenzte Gleichheit auf dasselbe Ziel und ergänzte Ursacheninformationen.
- **v0.13.0** trennte strukturelle Blockaden von Policy-Fehlern und bereitete selektive Übernahme vor.
- **v0.14.1** bewahrte den stabilen Project/Domain-Vertrag und veröffentlichte daneben eine experimentelle kanonische Reset-/Controller-Alpha. Die Alpha ist kein Beleg, dass der vollständige autonome Arbeitsablauf bereits umgesetzt ist.

Diese Versionen sind eine knappe Einordnung aus den aktuellen kanonischen Dokumenten, keine vollständige Commitchronik. Die ältere Arbeit an Layout, Modulen, Distribution, Provider-Adaptern und Doku-Routern bleibt im Repository und in den älteren Chats nachvollziehbar.

## 6.–8. Oktober: von der Modellbasis zu überprüfbarer Ausführung

Am **6. Oktober** integrierte PR #80 den großen Canonical-Core-/Reconcile-first-Schritt (`75031b8e`, 02:02 Berlin). Er ordnete typisierte Ressourcen, Compiler/Core, Modulverantwortung für Projektionen und den Abgleich zwischen gewünschtem und beobachtetem Zustand neu. Das war eine breite Architekturverschiebung, kein einzelner Adapter.

PR #81 (`91f8fe31`, 02:54 Berlin) ergänzte ein Capability-Proof-Programm mit fester Inventur, Matrix, Harness und erhaltenen Läufen. Eine frühe Akquisitionsgrenze stoppte nachgelagerte Versuche; der Stopp blieb als Befund erhalten. Danach wurde die Arbeitsrichtung präzisiert: fehlende akzeptierte Fähigkeiten erst implementieren, dann mit frischem Quellstand neu messen. Diese Historie erklärt, warum „Code vorhanden“, „Harness bereit“ und „Trial gelaufen“ getrennte Statusfelder brauchen.

Am 6.–7. Oktober entstanden Standard Operating Model, Controller-Aktionen, Audit-/Repair-Schritte und eng begrenzte Live-Proben. Die Folge zeigt einen produktiven Korrekturkreis: Fehler wurden nicht zu Erfolg umetikettiert; Quell- und Belegstände wurden eingefroren; Nachbesserungen bekamen neue Pins. Eine bestandene Probe deckte nur ihren festgelegten Umfang ab.

Am **7. Oktober** wurde **v0.14.1** veröffentlicht; der Release-Eintrag datiert auf 13:15 UTC und nennt Tagziel `784d3c3c61443b291ac7db9727c7c5856e253d66`. PR #85 schloss Release-Evidenz und Distribution ab. Danach kamen der öffentliche Classic-Commerce-Walkthrough und eine Intent-Änderung hinzu: PR #86 (`560acdcd`, 8. Oktober 18:25 Berlin) zeigt ein begrenztes Projektbeispiel; PR #87 (`a97cbd5e`, 8. Oktober 18:59 Berlin) richtet Classic-Dokumentation auf Main aus. In den Chats wurde zugleich festgehalten: Der kanonische Controller ist weiterhin eine Alpha, der installierte Release ist v0.14.1 und CI beweist nicht die gesamte Produktthese.

Am **8. Oktober** ergänzte PR #88 (`5be48ce1`, 22:00 Berlin) eine spätere Untersuchung zu Knowledge Graph und Markdown Front Matter. Die Untersuchung war als Optionen-/Backlog-Eintrag gedacht, nicht als Formatentscheid. Am 9. Oktober sagte der Nutzer später ausdrücklich, dass Front Matter verworfen werden kann, und korrigierte eine Markdown-zentrierte Produktbeschreibung. Die Folge ist wichtig: #88 bleibt historische Quelle eines offenen Prüfauftrags; spätere Nutzerkorrektur verändert dessen Einordnung. YAML, OWL/RDF und Graph bleiben getrennt zu bewertende Möglichkeiten.

### 8. Oktober: vom Ressourcenmodell zur delegierten Projektorganisation

Im **Design**-Verlauf (`01a11c80-37aa-7fe0-9586-35d916ce6561`) weitete der Nutzer die Zielvorstellung auf eine zusammenhängende Projektwelt aus: Code, Dokumentation, Pipelines, Tests, Prozesse, Konfiguration, Konzepte und Anwendungsfälle sollten als zusammengehörig verständlich werden (`01a11c8d-aa3c-7563-8afd-516e2466e8ba`). Er wollte die Struktur an fachlichen Konzepten/Vertical Slices und Verantwortung ausrichten, nicht bloß an Ressourcentypen (`01a11cc5-d740-75b2-b8f6-7cb910d8d54e`); Owner sollen erklären, wem eine Datei gehört, wozu sie gehört und was sie realisiert (`01a11ce2-f87e-73f2-ae56-83e405e57528`).

Als Zielmodell skizzierte er rekursive, spezialisierte Manager-Agenten mit begrenztem Kontext und delegierten Aufgaben. Untergeordnete Resultate sollten integriert, ungelöste Konflikte nach oben eskaliert und Umsetzungen von Reviewern iterativ geprüft werden (`01a11c9e-fa12-7ba2-b9d0-9bc41a7a45ac`, `01a11ca1-534e-7e32-ae2b-277f99d4e651`, `01a11dcd-d661-74a1-a86d-c9effdbca822`). Vor Annahme der UX wollte der Nutzer einen kleinen Softwarefall vom Modell über Änderung und Brownfield-Aufnahme bis Installation und täglicher Nutzung nachvollziehen (`01a11ca8-f205-7f60-b5fd-b9e73b83265b`, `01a11ce9-3a0b-7500-8a7c-c40376a608ee`).

Diese Unterhaltung beschreibt ein **Ziel- und Designmodell**, keine fertige Markitect-Laufzeit. Im veröffentlichten Vertrag bleiben Code und andere normale Dateien explizite, opake Inputs; generische Graphkanten werden nicht aus Prosa oder Dateinamen hergeleitet. Der Management-/Manager-Ausbau lag in separaten Arbeitssträngen. Der Abstand zwischen umfassender Projektvision und begrenztem, explizitem Datenmodell ist damit eine zentrale offene Produktfrage, kein Beleg für bereits vorhandene autonome Projektmanager.

## 9. Oktober: Untersuchung, Experimente und parallele Quellenentwicklung

Ein Abschlussbericht der Government-Untersuchung liegt auf dem separaten Branch `codex/government-assessment`, nicht auf Main. Er empfiehlt für aktuelle Projektarbeit einen starken konventionellen Engineering-Ablauf mit selektivem Markitect-Modell und unabhängigen Tests; Design wird als möglicher integrierter Nachfolgekandidat betrachtet, Government als optionale Organisationsschicht. Der Bericht dokumentiert **keinen gültigen, vollständig abgeschlossenen Sechs-Aufgaben-Vergleich**. Das ist ein datierter Untersuchungsbefund und eine Empfehlung, keine in Main übernommene Produktentscheidung.

Parallel entstand ein großer **Model-first/Product-Integration**-Strang. Der Management-Branch `codex/model-first-operations` erreichte `1495e1be` (18:58 Berlin) mit P01/P02-Abschluss und Übergabe. Daraus gingen getrennte Kandidaten für Projektoperationen, MCP, eigene Git-Arbeitsräume, App Server und Wiederherstellung hervor. Auf dem Integrationsbranch `fc6d09a2` (22:19 Berlin) war viel Quellkomposition vorhanden; der readiness-Bericht hielt die Vollgates dennoch offen:

- Der komplette Go-Lauf auf `dae4b4a5` endete nach rund 31 Minuten mit Fehlern in Fixtures, Onboarding-Texten und ProjectRun-Dauer/Replays.
- Ein weiterer kompletter Lauf auf unverändertem `3c4ab67c` endete nach rund 29 Minuten mit zwei App-Server-Timeout-Subtests. Auch das war kein vollständiger PASS.
- Danach korrigierte `1756a669` nur den Timeout-Testmechanismus; das ganze App-Server-Paket und Wiederholungen bestanden fokussiert. Der neue integrierte Volltest und Linux-/Windows-CI auf dem finalen PR-Head standen noch aus.
- Die Struktur wurde mit `ready=true` und null Blockern neu bestätigt. Der Bericht nennt ausdrücklich: Struktur-Readiness ist keine semantische Akzeptanz; es gab null Actual-Jobs und null Rollenstarts. A01/A02/A03 standen auf NOT STARTED; kein Main-Merge, Release, Human Acceptance oder Produktivitätsnachweis folgte daraus.

Das ist ein klares Beispiel für den Arbeitsstil: gezielte Reparatur und unabhängige Review können einzelne Befunde schließen, ohne alte Volltestläufe umzuschreiben oder ein neues Gesamtgate zu ersetzen.

Der **Knowledge-Graph-Zweig** blieb ebenfalls unabhängig. Der Nutzer erlaubte eine Variante auf einer exakten Management-Basis zur getrennten Auswertung, nicht die automatische Übernahme nach Main. Der Voll-Verify auf `d360ec01` scheiterte an zwei projectcli-Fixtures. Nach deren gezielter Korrektur scheiterte der Voll-Verify auf `ebe51470` am ProjectRun-Test `TestDeliverResumesIntegratedRunAndCompletesAcknowledgedScope`, weil das Fixture unter Suite-Last sein vierminütiges Laufzeitlimit überschritt. Der gezielte Test bestand danach mit einem auf 15 Minuten erhöhten Fixture-Limit; Produktionsgrenzen blieben unverändert. Eine vollständige Verifikation nach diesem letzten Fixture-Fix stand noch aus, der Gesamtstand blieb FAIL. Das Team hielt weitere unveränderte Vollsuiten zurück und reservierte eine kombinierte Verifikation für den später akzeptierten Main-Stand. Die Arbeit liefert echte Compiler-/CLI-/MCP-Mechanik mit statischen Fixtures, aber keine Provider- oder autonomen Qualitätsbelege und keine Human Acceptance.

Ein weiterer Gesprächsstrang zum **Agent Behaviour Designer/Agent Studio** stammt etwa vom 1. Oktober. Nach Sichtung eines inzwischen nicht auffindbaren Konzeptdokuments empfahl die Antwort zunächst einen isolierten Authoring-Prototyp auf dem bestehenden Markitect-Modell. Der Nutzer sagte ausdrücklich: nicht auf Main bringen, erst testen. Ein späterer Chatbericht meldete einen ausgebauten sechsstufigen Creator, lokale Bibliothek, YAML-Roundtrip, Browser- und Repository-Prüfungen. Er erklärte den Nutzen des Authorings zugleich für noch offen. Das ist ein gutes historisches Beispiel für „funktionierende Demo“ versus „bestätigter Produktnutzen“. Das im Chat referenzierte Downloads-Dokument liegt am damaligen Pfad nicht mehr vor; die Prototyp-Dateien sind in den aktuell registrierten Worktrees nicht enthalten.

## Direkte Nutzersteuerung und offene Zukunftsvision

Die Produktbeschreibung, die am 9. Oktober in Concept- und Marketing-Notizen aufgenommen wurde, richtet sich gegen das Vergessen relevanter Dateien und weitergeltender Regeln, widersprüchliche Realisierungen und teure manuelle Konsistenzarbeit. Genannt werden eine maßgebliche kanonische Absicht, Typisierung/Compiler, die Frage „Was gehört wozu?“, Änderungsfolgen und begrenzte Verantwortlichkeiten mit unabhängiger Prüfung. Das sind Produktziel und berichtete Erfahrung; erwartete Verbesserungen bei Qualität, Vertrauen, Modellkosten oder Zeit bleiben Hypothesen. Auch bei guter Struktur kann ein Modell falsch, unvollständig oder missachtet sein.

Spätere Zukunftsideen – konfigurierte Manager-Spielräume, Briefings, Monitoring/Protokolle, ein persönliches Cockpit mit Chat-Agent, Visualisierung, Gamification und mögliche Technologien wie Kubernetes/etcd – wurden als Brainstorming über ein **hypothetisch fertiges** Produkt eingebracht. Der Nutzer setzte zugleich eine klare Planungsgrenze: Feinplanung erst nach Abschluss der bestehenden Arbeitspakete, Case-Study-Zahlen, sauberem Main und stabil verwendbarem Markitect. Das sind deshalb keine freigegebenen Features oder Architekturentscheidungen.

## Quellenstatus zum Snapshot

- **Live `origin/main`:** `5be48ce1`, gegen `git ls-remote` geprüft. Das lokale Branchlabel `main` zeigte zu diesem Zeitpunkt noch auf `d4a07704` und war 521 Commits zurück.
- **Primärer Benutzercheckout:** Branch `codex/government-assessment`, HEAD `d26b494b`; vier untracked Concept-Dateien. Das ist nicht der Main-Stand.
- **Product Integration:** `fc6d09a2`, separates sauberes Worktree; Voll-/Plattformgates noch offen.
- **Knowledge Graph:** `ebe51470`, separates, lokal verändertes Worktree; Gesamtverifikation fehlgeschlagen, nur gezielte Nachbesserung belegt.
- **Historian:** eigener Branch und eigener Worktree. Die aktuellen Tabellen und Zeitstempel liegen im [neueren Worktree-Snapshot](git-worktrees-current-2026-10-09T20-29-01Z.tsv) und in den [Chat-Snapshots](archived-project-thread-candidates-20261009.json) und [aktiven Chats](active-project-threads-2026-10-09T20-29-01Z.json).

Kein Branchname, Agentenbericht, lokaler Testlauf oder PR ersetzt automatisch Main-Integration, Release, echte Laufzeitbelege, unabhängige Qualitätsaussage oder menschliche Annahme.

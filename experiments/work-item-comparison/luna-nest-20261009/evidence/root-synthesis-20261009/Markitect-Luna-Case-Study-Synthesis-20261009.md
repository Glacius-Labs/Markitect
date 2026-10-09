# Luna-Case-Studies: Abschluss des begrenzten Vergleichs am 9. Oktober 2026

Stand der Evidenzprüfung: 13:20 Uhr Europe/Berlin. Diese Datei enthält die private Bewertung für den Nutzer. Sie gehört ausschließlich in das private Research-Backup, nicht in das öffentliche Produktrepository oder in den Kontext von Implementierern. Produktbefunde dürfen separat ohne private Bewertungsdaten weitergegeben werden.

Für die derzeitige Projektarbeit empfehle ich Conventional Agentic Coding. Im geprüften Greenfield-Fall hat es zwölf Work Items bis zur integrierten Umbenennung geliefert. Design-Markitect bleibt die gewünschte Produktrichtung, ist auf dem getesteten Stand aber noch kein benutzbarer autonomer Ersatz: Der Lauf stoppte am Briefing-Speicher, bevor eine Anwendung entstand. Das ist ein Befund über diesen Kandidaten und diesen Arbeitsweg. Ein allgemeiner Sieg über modellgestützte Entwicklung lässt sich daraus nicht ableiten.

Die sechs erlaubten Starts dieses Blocks sind geschlossen. Es gibt keinen weiteren automatischen Versuch und keine kostenlose Wiederholung. Designer arbeitet unabhängig davon an der konkreten Produktkorrektur und am bestehenden Implementierungs-/Dokumentations-/Mainauftrag weiter. Die Studienbeobachtung wird nach dieser Synthese pausiert; Produktabschlüsse können weiterhin per Callback eintreffen.

## Tatsächlich verglichene Stände

Der gemeinsame öffentliche Rahmen ist Scientist-Source `2037bfa65e7e05ee004921d282893d4a6c8e899a`. Roombook wurde in vier Stationen mit 1/3/7/1 Work Items bearbeitet: Einzelauftrag, mehrere Aufträge, abhängige und unabhängige Arbeit mit Teambeiträgen, abschließende Umbenennung. Die Stationen sind Schritte eines Falls, keine vier Replikationen. Implementierer, Helfer und frische unabhängige Assessoren wurden mit `gpt-6-luna/high`, frischem App-Kontext und `fork_turns=none` angefordert. Tatsächliches Serving und Providerzähler sind nicht attestiert.

Conventional erhielt den normalen Backlogauftrag und die allgemeinen Projektregeln. Planung, Dokumentation, Tests, Subagenten und Reviews waren zulässig. Design erhielt dieselben Anforderungen und zusätzlich ein vorher aus öffentlichen Anforderungen erstelltes kanonisches Anfangsmodell sowie die native Installation und Agentenhinweise. Scientist gab keine semantische Implementierungshilfe. Design musste seine echten Produktmechanismen verwenden.

Conventional endete auf Kandidat `539659cddb7c086894d089e67dadacb8d8d1fd9a`. Seine privaten Abschlussbelege sind auf Scientist-Commit `ecca3b05d46d9c51cd41d299c1fb7272d7bbf113` gesichert. Design wurde auf Produkt-Source `bcd614a3bfcd335e0ca18850993742057103c092` mit Binary-SHA256 `abf877849379900cf2f1cf6c19f3d5d1e67a134592ac810cedc437191e9a5f29` getestet. Sein eingefrorenes Fall-main blieb `ffcf65173c030e8718ab7098c8b461dfe03b0395`; der ungemergte Arbeitsbranch endete auf `8fe3df7e8f5c74173071793a8e399a364c2c5e4e`. Private Abschlussbelege: Scientist-Commit `57c862ad6ea887cf71b12e27d0fdb34a9ca09bb8`.

Root hat die sauberen exakten privaten Remote-Checkpoints und die Dateibindungen geprüft: Conventional 416 Working/RawGit-Bindungen und 400 Payloadhashes, Design 183 Working/RawGit-Payloadbindungen. Diese Integritätsprüfung wiederholt keine Produkttests und ersetzt nicht die unabhängige Bewertung. Die Final-Assessoren prüften unveränderte Kandidaten.

## Ergebnisse und praktische Bedeutung

| Frage | Conventional | Design-Markitect |
|---|---|---|
| Greenfield, erster Auftrag | Geliefert; 6 öffentliche Checks und 5 eigene Unit-Tests bestanden | Nicht geliefert; Anwendungseintritt `app.py` fehlt nach dem Produktabbruch |
| Mehrere und abhängige Aufträge | Alle vier Stationen bis R12 abgeschlossen | Stationen 2–4 nicht ausgeführt |
| Endstand | 13 öffentliche Checks und 25 eigene Unit-Tests bestanden; zusätzliche unabhängige CLI-Prüfungen bestanden | Sechs S1-Harnessversuche scheiterten am selben fehlenden Eintrittspunkt; keine sechs unabhängigen Verhaltensfehler |
| Umbenennung und Bestandserhalt | In der unabhängigen Bewertung bestanden, einschließlich erlaubter Legacy-/Speicherkompatibilität | Nicht erreicht; Nutzen des Modells hierfür unbeantwortet |
| Teamarbeit | Beiträge und Integration im Git-Verlauf unabhängig geprüft | Nicht erreicht |
| Tatsächliche zeitliche Teamparallelität | Vier aktive kooperative Ledger-Einträge beobachtet; vollständige Provider-Zeitüberlappung nicht unabhängig attestiert | Nicht geprüft |
| Codequalität, Konsistenz und Regeln | Keine offenen Befunde im erklärten geprüften Umfang; keine umfassende Fehlerfreiheits- oder menschliche Abnahmebehauptung | Anwendungsqualität nicht bewertbar, da keine Implementierung vorliegt |
| Brownfield, große Backlogs, Replikationen | Nicht ausgeführt | Nicht ausgeführt |

Der Conventional-Fall zeigt, dass ein kurzer normaler Arbeitsauftrag plus gewöhnliche Projektregeln in diesem Nest genügte. Eine kanonische Modellierung ist keine notwendige Voraussetzung für eine konsistente Umbenennung in einem kleinen Projekt. Ob sie bei großen Änderungen oder Brownfield den Aufwand senkt, ist offen.

Designs Actor klärte zunächst einen kanonischen R01-Vertrag über den echten Modelländerungsweg. Anschließend lehnten Briefing-/Readiness-Aufrufe die Definitionidentität ab: `briefing bundle is invalid: incomplete definition identity`. Es gab keine Businessimplementierung, keinen inneren Manager-/Reviewer-Aufruf und keinen erfolgreichen Run/Verify/Apply. Der Actor stoppte, statt den Produktweg durch direkten Code zu umgehen. Das ist positive Methodendisziplin bei einem funktional nicht gelieferten Auftrag.

Root fand im öffentlichen Source einen konkreten Widerspruch: `internal/host/projectbriefing/store.go:175` verlangt einen Namespace, während `internal/core/compile.go:947` den leeren Namespace als gültig erlaubt. Der vollständige Laufursachenpfad ist nicht durch eine zusätzliche Root-Reproduktion bewiesen. Designer hat einen begrenzten Auftrag zur eigenen generischen Reproduktion, Korrektur und Regression erhalten. Private Fallantworten oder Final-Befunde wurden ihm nicht übermittelt.

Die unabhängige Bewertung fand außerdem eine veraltete erzeugte Projektansicht, die vorhandene gemeinsame Dateien als fehlend bezeichnete. Die Abweichung im Snapshot ist belegt; ihre Entstehung aus kopierter Anfangsprojektion oder späterem Lauf ist nicht abschließend zugeordnet. Sie zählt deshalb nicht als bewiesener Laufzeitfehler des Produkts. Sie zeigt den Prüfbedarf für Modell-/Dateitreue.

## Aufwand, Fehler und Unsicherheit

Conventional verbrauchte vier äußere Implementierungsaktivierungen, drei Helfer und einen frischen unabhängigen Final: acht zurückgekehrte Einträge. Das beobachtete Fenster von erster Reservierung bis Ledgerabschluss beträgt 2.352,63 Sekunden, etwa 39 Minuten 13 Sekunden. Es enthält Dispatch, Koordination und Bewertung. Die nachfolgende Evidenzsicherung kam hinzu. Es ist keine reine Modell-, Compute- oder menschliche Arbeitszeit.

Design verbrauchte einen äußeren Implementierer und einen frischen Final, keine Helfer und keine im inneren Ledger gestarteten Produktrollen. Von erster Reservierung bis Evidenzcapture vergingen 1.304,80 Sekunden, etwa 21 Minuten 45 Sekunden. Diese kürzere Zeit endete mit einem nicht gelieferten Auftrag und ist kein Effizienzvorteil. Frische Vorbereitung einschließlich Korrektur und Rückmeldungen wurde konservativ mit 963 Sekunden statt der vorgesehenen 900 ausgewiesen; 1.385,85 Sekunden vorheriges Owner-Installationspaket bleiben zusätzlich erhalten. Beide Zeitfenster haben verschiedene Abschlussgrenzen und werden nicht zu einem fairen Geschwindigkeitsverhältnis verrechnet.

Tokenverbrauch, abgerechnete Kosten, genaue Provider-Laufzeiten, Serving-Modell und aktive menschliche Aufsicht sind unbekannt. Daher gibt es keinen belastbaren Tokenkosten-, Compute- oder Aufsichtskostenvergleich. Die zusätzliche kanonische Modellvorbereitung und alle Setup-/Review-/Korrekturarbeiten bleiben als Aufwand sichtbar; der Gesamtaufwand ist nicht vollständig gemessen.

Die vier früheren Starts dieses Blocks bleiben fehlgeschlagen oder unvollständig: drei CLI-Ausführungen mit Modell-/Berechtigungsproblemen und ein unterbrochener App-Lauf unter einem anschließend prospektiv korrigierten Kontextvertrag. Sie sind keine vier unabhängigen Qualitätsbeobachtungen. Diese Verzögerungen gehören teilweise zur Ausführung und Koordination; sie dürfen weder pauschal Markitect noch Conventional als Entwicklungsmodell zugerechnet werden.

Die Bewertung hatte eigene Fehler: zwei falsche Conventional-Erwartungen wurden bei unverändertem Kandidaten korrigiert; der Design-Harness dekodiert JSON vor einer hilfreichen Exit-/stderr-Zuordnung und machte den fehlenden Eintrittspunkt zunächst undeutlich. Zusätzliche private Probes entstanden erst während Final und sind keine vorab eingefrorenen Holdouts oder nachträglich neuen Erfolgskriterien. Root korrigierte eigene Dateipfad-/Datentyp-Leseannahmen ohne Produkt-, Actor- oder Kandidatenänderung.

Bei Design stimmen alle 54 archivierten RawGit-Dateibindungen der zwei eingefrorenen Checkouts mit der Vorbereitung überein. Vier Working-Dateien unterscheiden sich ausschließlich durch LF/CRLF; beide Checkouts blieben vor/nach Bewertung unverändert. Der ursprüngliche Capture-Assertfehler wird erhalten, ohne Umformatierung oder Neubewertung. Auch die letzte Pipe-Drain-Ausnahmekorrektur des Ressourcenwrappers ist nicht durch einen neuen Providerlauf oder einen exakten abschließenden unabhängigen Review bewiesen. Die echte innere CLI-Kompatibilität wurde im abgebrochenen Design-Fall nicht erreicht.

Die App-Actors hatten kooperative getrennte Checkouts, keine nachgewiesene OS-Isolation. Die genuine innere Design-CLI hat eine erklärte andere Tool-/Read-only-Konfiguration. Aus dem Ergebnis folgt deshalb die praktische Benutzbarkeit dieser Stände, keine universelle kausale Rangfolge bei identischem Gesamtaufwand. Ein erfolgreicher und ein blockierter Greenfield-Fall reichen nicht für Zuverlässigkeitsstatistik oder Brownfield-Aussagen.

## Empfohlener Produktkern und Mainentscheidung

Ich würde Design als Produktrichtung weiterführen und zuerst den vollständigen gewöhnlichen Benutzerweg zuverlässig machen. Der minimale Kern ist ein kleines typisiertes YAML-Modell mit Absicht, Regeln, Realisierungen und eindeutiger Datei-/Änderungsverantwortung; eine native Installation mit auffindbaren Agentenhinweisen; nachvollziehbare Modelländerung und dauerhafte, wieder ladbare Briefings; echte Umsetzung mit unabhängiger Prüfung und konservativem Apply; sowie Wiederaufnahme ohne doppelte abgeschlossene Arbeit. Identitäten, Roundtrip und erzeugte Ansichten müssen über alle Schichten dieselbe Bedeutung haben.

Mehr Manager, Government-Ressortvoten, zusätzliche Reviewrunden oder ein Knowledge Graph sind optionale Mechanismen. Für ihren Qualitätsnutzen und Aufwand gibt es aus diesem Block keinen Vergleichsnachweis. Sie sollten konkrete belegte Probleme lösen, nicht den normalen Einstieg verkomplizieren. Eigenständige Government- und alte Classic-Studien bleiben auf Nutzerentscheidung stillgelegt; daraus folgt keine empirische Überlegenheit oder Unterlegenheit.

Der bestehende Hauptbranch `5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e` bleibt bis zur regulären Designübernahme erhalten. Alter Classic wird nicht als neues Entwicklungsprogramm reaktiviert. Die Nutzerentscheidung für Design als Nachfolger bleibt bestehen; eine sofortige Übernahme des getesteten `bcd614a3` empfehle ich angesichts des blockierten Normalwegs nicht. CI `37914238147` bestand auf genau diesem Source unter Linux und Windows, beweist aber keinen erfolgreichen autonomen Auftrag. PR89 benötigt weiterhin den korrigierten funktionalen Kandidaten, erforderliche Gates und unabhängigen Review. Ein weiterer gematchter Studienlauf oder eine Releaseveröffentlichung ist aus dieser Synthese nicht freigegeben.

## Versionsgebundener Bedienweg

Die folgenden Befehle beschreiben den Sourceweg des experimentellen Design-Kandidaten `bcd614a3`, nicht die Fähigkeiten einer bereits installierten alten Release. Nach der Korrektur müssen Source, Binary und Runtime neu gebunden werden; alte Studienbelege werden dabei nicht ersetzt.

1. Nutzerabsicht und bekannte Regeln knapp festhalten; im Projekt einen eigenen Featurebranch vom gewählten main anlegen. Source/Binary/Runtime exakt pinnen. Offene echte Nutzerentscheidungen separat halten.
2. `markitect project schema` lesen; `markitect project init --repo . --name my-project` als Preview, anschließend mit `--write` einrichten. Für diesen Weg ist `.markitect/project.yaml` der Selector. Historisches top-level `markitect init` gehört zum anderen Projektformat.
3. Kanonisches Modell mit Zielen, Regeln, Artefakten und Eigentümern pflegen; native Agentenhinweise installieren und die generierten Ansichten gegen den aktuellen Repo-/Modellstand prüfen. Normale Work Items enthalten die gewünschte Änderung, keine vorausgeschriebene Lösung.
4. `markitect project explore --repo .` verwenden; nötige Scope-/Entscheidungsdaten über die tatsächlichen Reports und bewachte Modelländerungen übernehmen. JSON-Eingaben liegen in unterstützten relativen `.markitect/drafts/`- oder `.markitect/runs/`-Pfaden. Keine erfundenen Freigaben oder menschenähnlichen Zustimmungen.
5. Briefings und `markitect project readiness --repo . --exploration ID --scope ID` prüfen. Genau an diesem Übergang ist der getestete Stand blockiert. Erst eine geprüfte Korrektur macht den folgenden Benutzerweg belastbar.
6. Source-seitig ist `markitect project deliver --repo . --exploration ID --scope ID --run RUN_ID --write` für den gebundenen Plan-/Run-/Review-/Verify-/Apply-Weg vorgesehen. Tatsächliche IDs stammen aus den Reports. Alternativ die dokumentierten `project plan`, `run`, `status`, `verify` und Apply-Preflight-Schritte benutzen. Der Fall hat deren erfolgreiche Verkettung nicht bewiesen.
7. Modell-/Dateitreue, Regeln, Tests und Kandidatenbindung prüfen; einen erforderlichen menschlichen Entscheid ausdrücklich einholen. Bewachtes Apply und regulärer Git-Merge gelten für den verifizierten Kandidaten. Bei Unterbrechung denselben Run und seine dauerhaften Receipts wieder aufnehmen, statt neue ungebundene Arbeit zu starten.

Für sofortige normale Projektarbeit ist der bewiesene Conventional-Weg einfacher: frischer Projektbranch, übliche Projektregeln, kurzer Auftrag „Hier sind die Work Items, implementiere sie, prüfe sie und integriere sie nach main“, danach unabhängige Endbewertung. Für eine spätere Designablösung müssen vor allem Normalweg, Brownfield-Bestandserhalt, Wiederaufnahme und Modell-/Dateitreue im funktionalen Kandidaten belegt werden. Diese offenen Fragen bleiben offen, statt durch weitere automatische Vorbereitungsschleifen als erledigt zu gelten.

## Unveränderte Primärbelege

Private Scientist-Remote: `https://github.com/TheGlacius/markitect-research-backup.git`, Branch `codex/government-scientist`. Öffentlicher Produktbranch: `https://github.com/Glacius-Labs/Markitect.git`, `codex/model-first-operations`; PR89. Die Bewertung bleibt privat.

- Conventional terminal-summary SHA256 `3c7d0d6453bed4d68c01ff82ed4bdc684c1e5883aaf2378a275b80868e037549`; unabhängiger Final `2bad8474c0a6ff5a30f08f37199341c6c85de8ed2726eaba21087a70b41f4e6b`.
- Design terminal-summary `544ef2b683390ba09d5e92147172b07acbf3a3618db3746d587c8abaced961b5`; Retained-Manifest `8c1b319f2bf76459c28a5786f27965a4465b0246a67954f59502d5460b6ca50f`; unabhängiger Final `f97ba9b7a9efa58395f28b47fb9d5b9f778434ee2d5019108c6f3336b31923a9`.
- Design closed-ledger `62cbeecc7038b26a07adb20396643d089c24f9570927f8b33bc551484446f834`; Snapshot-Diagnostik `0f58502f20ccf4a25f4ee6e2313002028bdcfb671aba88c06b519e32a5599878`.
- Öffentliches Produktkorrekturmandat `design-briefing-global-identity-regression-20261009-grant.json`, SHA256 `9e6470ff7f2444cad190954de41d07b40759dbf10e83bf9270bd0c787d7f2fb7`, einmal erteilt 11:11:39Z, endlicher Abschluss spätestens 12:11:39Z. Kein neuer nativer Produktproof und kein Studienstart.

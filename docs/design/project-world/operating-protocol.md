# Zielentwurf: Laufzeitprotokoll der Managementhierarchie

Status: Entwurf für die Planung. Dieses Dokument beschreibt ein gewünschtes Betriebsmodell; es behauptet keine vorhandene CLI, API, Agentenlaufzeit oder Git-Integration. Die Begriffe und Verbindungen müssen später gegen das gewählte Markitect-Quellmodell präzisiert werden. Es ergänzt das [Zielmodell](model.md) um einen konkreten Durchlauf und bleibt an dessen Richtungsentscheidungen gebunden.

## 1. Kanonisches Shop-Beispiel

Das konkrete Dateilayout und das zusammenhängende Modell des kleinen Shop-Monolithen haben genau einen Besitzer: den [Shop-Walkthrough](shop-walkthrough.md) und sein [Beispielprojekt](shop-example/README.md). Dort sind die Modellstruktur, Bereiche, Rollen und Fachbegriffe maßgeblich; dieses Laufzeitprotokoll dupliziert sie nicht. Das Beispiel ist ein Planungsartefakt und keine aktuelle CLI-Syntax.

Für den Ablauf gilt: Orders und Bestand verwenden dieselbe Datenbanktransaktion. Eine bestätigte Order hält eine Reservierung. Die Beispieländerung erlaubt Stornierung vor Versand und gibt die Reservierung atomar sowie idempotent frei. Die fachlichen Identitäten und Eigentümer stehen im kanonischen Beispiel: `sales/cancel-before-shipped` gehört zu `orders-manager`; die bereichsübergreifende Regel `commerce/cancellation-releases-reservation` gehört zu `commerce-manager`.

Jede besetzte Managerrolle wird durch einen eigenen spezialisierten AI-Agenten mit eigenem Kontext ausgeübt. Eine Rolle ist dauerhaft; eine Agenteninstanz und ihr Lauf sind konkrete, ersetzbare Ausführungen. Die Geschäftsführung hat umfassende Entscheidungsbefugnis innerhalb des Projektauftrags. Jeder Manager darf innerhalb seines Bereichs und ausdrücklich delegierter Grenzen Routineentscheidungen treffen. Der Nutzer bleibt oberste Instanz für vorbehaltene oder durch keine befugte Managementebene lösbare Fragen.

## 2. Zwei getrennte Vorgänge: Modell ändern und Arbeit ausführen

Eine Modelländerung und ihre Umsetzung sind zwei eigenständige Vorgänge:

1. **Vorschlag bearbeiten:** Notiz oder Änderungsentwurf wird auf Begriffe, Regeln, Referenzen, Auswirkungen und offene Entscheidungen geprüft. Währenddessen ist er Draft und ändert kein aktives Soll.
2. **Modell annehmen:** Zuständiger Entscheider nimmt einen exakt bezeichneten Modellkandidaten an. Dies erzeugt eine neue akzeptierte Revision. Der Schritt beantwortet die fachliche Frage, beauftragt aber noch keine Implementierung.
3. **Umsetzung starten:** Ein eigener Ausführungsauftrag bindet die angenommene Revision, den beobachteten Repositorystand und die Regeln für Werkzeuge und Ausführung. Erst dieser Schritt aktiviert Delegation und Schreibarbeit.

Das ermöglicht, ein Modell anzunehmen, es zunächst nur zu prüfen oder später auszuführen. Umgekehrt kann ein driftendes Repository repariert werden, ohne eine Modelländerung vorzutäuschen: Der Reparaturauftrag bindet dieselbe akzeptierte Revision und einen neuen beobachteten Dateistand. Ein Lauf kann sowohl seine ursprüngliche akzeptierte Basis als auch spätere, während des Laufs angenommene Modellrevisionen referenzieren. Ändert eine befugte Entscheidung das Modell, werden nur die dadurch betroffenen Aufträge neu geplant und ausdrücklich an die neue Revision gebunden; unabhängige, weiterhin gültige Aufträge müssen nicht künstlich in einem komplett neuen Lauf beginnen.

## 3. Was der deterministische Host erledigt

Der Host ist die technische Laufzeit des Protokolls. Mit gebundenen Eingaben erledigt er deterministisch:

- Schema- und Referenzvalidierung sowie zulässige Modellchecks;
- Erfassung unveränderlicher Modell- und Repository-Snapshots;
- explizite Impact-Auswertung anhand deklarierter Beziehungen und bekannte Lücken;
- Ermittlung betroffener Bereiche und ihrer gemeinsamen Vorfahren;
- Eröffnung und Zuordnung von Aufträgen, Kandidaten, Prüfläufen und Evidenz;
- Bereitstellung des jeweiligen Managerkontexts und Routing von Nachrichten;
- Git-Kandidatenverwaltung, Writer-Exklusivität und Zusammenführung gemäß Auftrag;
- erneute lokale und Integrationsprüfungen und Speicherung ihrer Eingaben und Ergebnisse.

Der Host entscheidet nicht, was ein fachlicher Begriff bedeutet, welche unmodellierte Beziehung gelten soll, welche Umsetzung fachlich angemessen ist oder welche Alternative gewählt wird. Manager-Agenten legen Bedeutung aus, formulieren Teilaufträge, wählen innerhalb ihrer Befugnis Lösungen und beurteilen Berichte. Sie dürfen technische Ergebnisse nicht als fachliche Annahme ausgeben. Eine Agentenbehauptung ersetzt keine unabhängig ausgeführte Prüfung.

Jeder Auftrag bindet mindestens: `accepted_model_revision`, `source_snapshot`, `base_repository_commit`, Werkzeug-/Modellversionen, ausführbare Policy-Bindungen, Berechtigungsumfang und Laufgrenzen. Ein Lauf kann ältere akzeptierte Basisrevisionen sowie neuere, angenommene und darin verwendete Modell-Snapshots nachvollziehbar zusammenführen. Ändert sich eine Grundlage, werden betroffene Aufträge neu abgeglichen oder ausdrücklich auf eine neue Grundlage umgeplant; gültige unabhängige Aufträge dürfen fortbestehen. Ein Agent darf seine Auftragsgrundlage nicht stillschweigend wechseln.

Impact wird konservativ ausgewertet. Explizite Kanten bestimmen bekannte Betroffene. Nicht inventarisierte, mehrdeutige oder nicht abbildbare Eingaben erscheinen als Lücke; sie werden nicht als „unbetroffen“ gewertet. Das Ergebnis ist eine nachvollziehbare Faktenliste für die Geschäftsführung, keine automatische fachliche Entscheidung.

## 4. Kontext und Nachrichtenverträge

### Kontextpaket eines Managers

Jeder Manageragent erhält nur das für seine Funktion erforderliche Paket:

| Teil | Inhalt |
|---|---|
| Auftrag | Ziel, betroffener Bereich, akzeptierte Modellrevision und geforderte Entscheidung bzw. Ergebnis |
| Befugnis | erlaubte Entscheidungen, Grenzen, dem Elternbereich oder Nutzer vorbehaltene Fragen |
| Fachkontext | relevante Begriffe, Regeln, Use Cases und explizit betroffene Beziehungen mit Herkunft |
| Schnittstellen | Zusagen und Abhängigkeiten zu direkten Kindern und betroffenen Nachbarbereichen |
| Arbeitsgrundlage | Quell-Snapshot, Repository-Basis, Werkzeug-/Policy-Bindungen und Laufgrenzen |
| Status | bereits erteilte Aufträge, eingegangene Berichte, offene Prüfungen und Entscheidungen |

Kindkontexte und Kindtranskripte werden niemals rekursiv in den Elternkontext kopiert. Der Elternmanager weiß, welche direkten Bereiche und Ergebniszusagen für seine Integration zählen. Er muss nicht wissen, welche internen Details oder Quellen ein Kindagent geladen hat. Bei fehlender Information stellt er eine konkrete Rückfrage über den Kindmanager.

### Minimalverträge

Jede Nachricht ist an einen Lauf, einen Sender, einen Empfänger und die bezeichneten Modell-/Kandidatenrevisionen gebunden. Gesprächsverläufe sind Transport, nicht das dauerhafte Ergebnisprotokoll.

| Nachricht | Pflichtinhalt |
|---|---|
| **Arbeitsauftrag** | Ziel; Bereich; Eingangsmodell und Basisstand; Ergebnisversprechen; Grenzen und Befugnis; benötigte Schnittstellen; Prüfkriterien; Frist-/Ressourcen- und Abbruchgrenzen |
| **Rückfrage** | genaue Frage; warum die Antwort benötigt wird; betroffene Entscheidung/Schnittstelle; benötigte Evidenz; Antwortfrist oder Blockierungswirkung |
| **Bericht** | Status (`complete`, `partial`, `blocked`, `failed`, `no-op`); Kandidatenbindung; geänderte Pfade; erfüllte Kriterien samt Evidenz; Schnittstellenänderungen; Risiken; offene Fragen; Vorschlag für nächsten Schritt |
| **Entscheidung/Eskalation** | konkrete Frage; Zuständigkeit und bisherige Entscheidungen; Fakten und Unsicherheit; Optionen mit Folgen; Empfehlung; Grund für Eskalation; betroffene Aufträge, die bis dahin ruhen |
| **Integrationsurteil** | Kindberichte und Kandidaten, zusammengeführte Kandidatenrevision, tatsächlich ausgeführte Kompatibilitätsprüfungen, Ergebnis, offene Abweichungen und freigegebene nächste Ebene |

Ein verkürzter Auftrag könnte lauten:

> **commerce-manager → orders-manager:** Setze `sales/cancel-before-shipped` für den Order-Zustandswechsel um. Beachte die bereichsübergreifende Regel `commerce/cancellation-releases-reservation`, die `commerce-manager` verantwortet. Basis: Modell `model-r18`, Repository `a42f…`. Reservierungsfreigabe erfolgt in derselben Transaktion; keine Stornierung nach Versand. Liefere isolierten Kandidaten, lokale Tests und geänderte Schnittstellen. Die Bestandsänderung verantwortet `inventory`; gemeinsame Dateien erst nach Writer-Zuteilung ändern.

Ein Bericht ohne Kandidatenbindung oder Nachweise ist kein Abschlussbericht. Ein Bericht darf Details auslassen, die für Elternentscheidung und Integration irrelevant sind; er darf jedoch Unsicherheit, fehlgeschlagene Kriterien oder nicht geprüfte Pflichten nicht verbergen.

## 5. Top-down-Delegation und bottom-up-Integration

Für die angenommene Stornierungsänderung läuft der Auftrag wie folgt:

1. Der Host validiert den angenommenen Modellstand, berechnet bekannte Folgen und bildet nur die betroffenen Managementpfade. `commerce` integriert die Order- und Bestandsänderung. Der `platform`-Manager bleibt in diesem Beispiel inaktiv, solange Build-, Test- und Betriebsverträge unverändert bleiben; dass vorhandene Prüfungen ausgeführt werden, begründet für sich allein keinen Plattform-Arbeitsauftrag.
2. Der `ceo`-Agent erhält im eröffneten Ausführungslauf die Änderung, bekannte Betroffene, gemeinsame Schnittstellen und Projektregeln. Er übernimmt den Gesamtauftrag und delegiert an den `commerce`-Agenten; dieser erstellt getrennte Aufträge für `orders` und `inventory`. Nur wenn eine konkrete Plattformpflicht betroffen ist, erhält `platform` einen Auftrag.
3. Jeder Manager übersetzt den übergeordneten Auftrag für seine direkten Kinder. Er übergibt Ziel, Ergebnisgrenze und erforderliche Schnittstellen, nicht seinen kompletten Kontext. Jede Kindrolle handelt in eigenem Agentenkontext.
4. `orders` und `inventory` bearbeiten ihre isolierten Kandidaten und liefern gebundene Berichte. Eine lokale Order-Prüfung kann Erfolg melden, während die Reservierungsfreigabe noch fehlt; der Host und der Integrator markieren die Gesamtregel dann als nicht erfüllt.
5. `commerce` integriert beide Kindkandidaten in einen unveränderlichen Integrationskandidaten und führt die Prüfungen über den gemeinsamen Transaktionsablauf erneut aus. Er beurteilt außerdem, ob Schnittstellen und Verhalten fachlich zusammenpassen. Bei Erfolg berichtet er eine integrierte Kandidatenrevision an `shop`.
6. `shop` prüft auf seiner Ebene die gesamte vereinbarte Änderung einschließlich betroffener Plattform- und Projektpflichten. Jeder Vorfahr integriert seine direkten Kindresultate und erzeugt einen neuen gebundenen Kandidatenstand; Berichte allein fügen keine Dateien zusammen.
7. Die Geschäftsführung schließt den beauftragten Umfang ab und gibt einen geprüften Kandidaten an den festgelegten Übernahmeprozess. Diese Freigabe übernimmt keine Modellrevision und publiziert keinen Release automatisch.

### Konflikte und Befugnis

Wenn `orders` und `inventory` unterschiedliche Transaktionsgrenzen oder Zustandszusagen vorschlagen, entscheidet zunächst ihr nächster gemeinsamer Vorfahr `commerce`, sofern der Konflikt in dessen Auftrag fällt. Reicht dessen Befugnis nicht, eskaliert der Manager die konkrete Entscheidung an den nächstgelegenen Vorfahren, der sowohl den betroffenen Umfang als auch die Entscheidungsklasse verantwortet. Der `ceo` entscheidet weitreichende Fragen innerhalb des Nutzerauftrags selbst. Nur ausdrücklich vorbehaltene oder dort unentscheidbare Fragen werden dem Nutzer vorgelegt.

Bis zur Entscheidung ruhen nur davon abhängige Schritte; unabhängige Arbeit darf weiterlaufen. Eine Entscheidung wird mit Geltungsbereich und Begründung nach unten zurückgegeben. Ändert sie das akzeptierte Modell, wird eine neue Revision ausdrücklich angenommen; betroffene Aufträge werden neu geplant und an den neuen Modellstand gebunden. Bereits laufende, nicht betroffene Aufträge können unter ihrer gültigen Basis fortfahren. Ein Agent darf die Modellbasis nicht im Umsetzungsauftrag umschreiben.

## 6. Isolierte Kandidaten, Writer und Integration

Jeder ausführende Kindauftrag arbeitet ausgehend vom gleichen benannten Basis-Commit in einem eigenen Git-Kandidaten (zum Beispiel einem isolierten Worktree). Jeder veränderbare Pfad hat in einem Kandidaten genau einen Writer. Zwei Manager dürfen denselben Pfad benötigen, aber nicht unabhängig gleichzeitig schreiben. Der Elternintegrator vergibt dafür einen Writer oder delegiert die koordinierte Änderung als separaten Auftrag. Ein Agent erhält Schreibzugriff nur auf den gebundenen Kandidaten und Umfang.

Integration ist eine tatsächliche Kandidatenoperation. Der Elternmanager bzw. ein beauftragtes Integrationswerkzeug vereinigt konkrete Kindrevisionen in einem neuen Kandidaten, löst Konflikte kontrolliert und führt alle durch Zusammenführung betroffenen Prüfungen erneut aus. Der neue Kandidat wird als neuer unveränderlicher Stand referenziert; Kindkandidaten und ihre Berichte bleiben erhalten. Ein erfolgreicher Test auf einem Kindbranch beweist nicht, dass der integrierte Kandidat funktioniert.

Bei Konflikt oder fehlgeschlagener Prüfung wird der Integrationskandidat nicht als fertig markiert. Der verantwortliche Manager kann gezielt nachdelegieren, selbst entscheiden oder eskalieren. Nach einer Korrektur entstehen neue Kindrevisionen und ein neuer Integrationskandidat; alte Evidenz wird nicht auf neue Bytes übertragen.

Neue tatsächlich geschriebene Pfade stehen zunächst im Laufbericht. Eine dauerhafte Ergänzung der kanonischen Realisierungszuordnung wird als befugte Metadatenänderung validiert und angenommen. Der Host berechnet deren Impact erneut und bindet betroffene Prüfungen an den finalen Modell-/Artefaktstand; sie werden vor Ergebnisübernahme frisch ausgeführt. Der konkrete Ablauf steht im [Walkthrough](shop-walkthrough.md). Eine Zuordnungsänderung darf keine Geschäftsregel heimlich ändern und keinen früheren PASS auf neue Eingaben übertragen.

## 7. Ende, Wiederaufnahme und Grenzen

Ein Laufende kann terminal sein, ohne erfolgreich zu sein. Timeout, Abbruch, Blockade oder fehlgeschlagene Kindaufträge können einen Lauf mit terminalem Status `blocked`, `failed` oder `cancelled` beenden; sie zählen niemals als erfolgreiche Erfüllung.

Ein Lauf ist **erfolgreich abgeschlossen**, wenn jeder beauftragte Kindauftrag entweder erfolgreich erledigt oder mit einer zulässigen, ausdrücklich begründeten Nichtanwendbarkeit geschlossen wurde, jeder Manager seine Kindresultate tatsächlich integriert und die vereinbarten Prüfungen auf dem finalen Kandidaten bestanden hat, keine entscheidungsrelevante Eskalation offen ist und die Geschäftsführung den Umfang als fertig berichtet. `N/A` ist nur zulässig, wenn die konkrete Verpflichtung im gebundenen Umfang tatsächlich nicht gilt und ein dazu befugter Entscheider diese Einordnung samt Begründung festhält. Ein fehlgeschlagener oder blockierter Auftrag kann nicht als `N/A` umetikettiert werden. Offene Modelllücken, nicht ausführbare Kriterien und übersprungene Prüfungen bleiben sichtbar; Erfolg bedeutet nicht automatisch „vollständig bewiesen“.

Bei Timeout, Agentenausfall oder Hostabbruch wird ein Manager mit seiner letzten gebundenen Eingabe, Auftragsliste, Kindberichten, Entscheidungshistorie und Kandidatenreferenzen wieder aufgenommen. Der neue Agent erhält diesen Rollenstand und ein eigenes Kontextpaket, keine fremden Gesprächsverläufe. Vor Fortsetzung prüft der Host, ob Modellrevision, Basis/Abstammung des Kandidaten, Berechtigungen und Toolbindungen weiter gelten. Andernfalls wird der Auftrag als veraltet markiert und gezielt neu gebunden.

Wiederholung derselben Nachricht mit gleicher Lauf- und Auftragskennung darf keine zweite unabhängige Änderung auslösen. Neue Kandidaten werden stets als neue Revisionen erfasst. Geschriebene oder gemergte Änderungen werden nicht durch einen vermeintlich idempotenten Retry überschrieben.

**Keine Umsetzung nötig:** Wenn gebundene Prüfungen zeigen, dass eine akzeptierte Änderung im Repository bereits korrekt umgesetzt ist, erstellt der Manager einen erfolgreichen `no-op`-Bericht mit Kandidatenbindung und Nachweisen. Es wird kein leerer Schreibauftrag simuliert. Die Integration prüft trotzdem, ob andere betroffene Bereiche noch Arbeit oder Nachweise schulden.

**Drift bei unverändertem Modell:** Wenn der Sollstand unverändert ist, aber eine Realisierung fehlt oder abweicht, entsteht ein Reparaturauftrag auf derselben Modellrevision und einem frischen Repository-Snapshot. Impact wird erneut für die driftende Pflicht ausgewertet. Der Manager darf keine neue fachliche Regel erfinden, um die Abweichung zu rechtfertigen. Wird beim Reparieren eine echte Modelllücke gefunden, endet dieser Reparaturpfad an einem Vorschlag; eine Annahme und ein neuer Ausführungslauf folgen separat.

Laufzeitrekursion ist begrenzt, obwohl Bereiche beliebig tief modelliert werden dürfen. Auftragstiefe, Agentenstarts, Kosten, Zeit, Parallelität und Wiederholungen haben deklarierte Grenzen. Bei ausgeschöpfter Grenze endet der Lauf mit Status und Wiederaufnahmepunkt statt still endlos weiterzudelegieren.

## 8. Einordnung und offene Entscheidungen

Dieses Protokoll ist eine konkrete Planungsantwort auf die Frage, wie ein kleines Projekt von Modelländerung zu integrierter Umsetzung kommt. Es ist noch keine technische Spezifikation. Zu entscheiden und am gewählten Markitect-Quellstand zu prüfen bleiben insbesondere: genaue kanonische Form der Rollen-/Auftrags-/Berichtsreferenzen, Snapshot- und Kandidatenspeicherung, Agentenlebenszyklus und Sicherheitsgrenzen, tatsächliche Git-Isolation, Policy-Bindungen und Abbruch-/Wiederaufnahmeverhalten. Diese Punkte ändern nicht die zentrale Trennung: Der Host bindet und prüft technische Fakten; spezialisierte Manageragenten entscheiden fachlich innerhalb ihrer delegierten Verantwortung; jede Ebene integriert ihre direkten Kinder vor dem Bericht nach oben.

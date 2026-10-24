# Markitect V1 – Konzept

**Modellbasierte Realisierung durch eine abgeleitete Managerorganisation**

| | |
|---|---|
| **Version** | 1.0 – festgehaltene Konzeptfassung |
| **Stand** | 24. Oktober 2026 |
| **Geltungsbereich** | Enge V1: Model → Artifacts |
| **Grundlage** | Owner-Fassung [v0.18](sources/v1-concept-v018-20261024.md) und die am 24. Oktober angenommenen Entscheidungen D10–D18 ([Entscheidungsprotokoll](sources/v1-concept-decisions-20261024.md), [DEC-068](register.md#dec-068-v1-concept-10)) |

> **Das kanonische Modell beschreibt die beabsichtigte Projektwelt. Die Managerorganisation wird daraus abgeleitet. Die Manager realisieren, prüfen und integrieren diese Welt; der Host stellt dafür wenige klare technische Garantien bereit.**

Diese Fassung verdichtet v0.18 und schließt deren offene Punkte durch Entscheidungen. Für D10–D18 gilt das Entscheidungsprotokoll, für alle übrigen Inhalte die Owner-Fassung v0.18. Jede weitere Abweichung dieser Fassung ist ein Redaktionsfehler und wird dem Coordinator gemeldet. Das Dokument beschreibt das Soll. Umsetzungsstand und bekannte Abweichungen des Codes führt der [Delivery Plan](../implementation-plan.md). Es erteilt keine Implementierungs-, Release-, Deployment- oder Abnahmefreigabe.

## Leseregeln

| Kategorie | Bedeutung |
|---|---|
| **Festgelegt** | Beschlossene Produktsemantik oder unmittelbar daraus folgende Verantwortungsgrenze. Das ist der Normalfall in diesem Dokument. |
| **Technisch auszugestalten** | Das Ziel steht fest; API, Speicherform, Providerabbildung oder Paketzuschnitt brauchen eine begründete Umsetzung. Gesammelt in Abschnitt 28. |
| **Später** | Bewusst nicht Teil von V1. Darf den Kern nicht durch vorgezogene Kompatibilitätspflichten bestimmen. |

„Muss“ bezeichnet eine fachliche oder technische Invariante, keine vorgeschriebene Datei. „Beispielsweise“ nennt eine geeignete Form, keine Pflichttechnologie. Owner-Entscheidungen bestimmen die Produktsemantik; ein Architekturvorschlag, ein Audit oder ein grüner Test ändert sie nicht stillschweigend. Aktueller Code wird nicht nachträglich zur Sollvorgabe, nur weil er implementiert oder erfolgreich getestet ist, und ausdrücklich verworfene frühere Anforderungen kehren nicht zurück.

---

## 1. Produktidee

### 1.1 Das Problem ist nicht fehlende Textproduktion

Mit KI-Agenten lassen sich Code, Tests, Dokumentation und Arbeitsanweisungen schnell erzeugen. Schwierig ist, ihre Bedeutung über längere Zeit konsistent zu halten. Eine Architekturentscheidung betrifft oft gleichzeitig Code, Schnittstellen, Tests, Konfiguration, Dokumentation und Agentenregeln; eine lokal erfolgreiche Änderung kann Verpflichtungen an anderer Stelle verletzen.

Ausgangspunkt waren wachsende, zunehmend widersprüchliche Sammlungen von Rules, Skills und Workflows. Reine Textpflege und wiederholte vollständige KI-Reviews reichten als Grundlage für Konsistenz nicht aus.

### 1.2 Ein kanonisches Modell statt konkurrierender Wahrheiten

Markitect stellt die beabsichtigte Projektwelt in einem kanonischen Modell dar: fachliche Anforderungen, Architektur, Beziehungen, Regeln, Verantwortlichkeiten und Qualitätsverpflichtungen. Code, Tests, Dokumentation und Konfiguration sind Realisierungen dieser Absicht.

```text
Absicht und Architektur im kanonischen Modell
    → deterministisch geprüfte Struktur
    → daraus abgeleitete Managerorganisation
    → Realisierung und Integration
    → unabhängige Verifikation
    → kontrolliert veröffentlichte Artefakte
```

Die kanonische Sprache verwendet typisierte YAML-Definitionen mit bedeutungstragender Markdown-Prosa. Ihre Syntax regelt die bestehende Sprachreferenz.

### 1.3 Compiler und Agenten ergänzen sich

Der Compiler prüft ausdrückbare strukturelle Eigenschaften. Die Manager beurteilen semantische Fragen und setzen das Modell um. Weder soll ein Sprachmodell triviale Referenzfehler erraten, noch soll der Host fachliche Arbeit in einen deterministischen Prozess zwingen. Die Analogie zu typisierten Sprachen reicht bis zur frühen Erkennung ausdrückbarer Strukturfehler; sie beweist nicht, dass natürlichsprachige Anforderungen korrekt realisiert sind.

### 1.4 Produktthese und Nachweisgrenze

Ein autoritatives Modell, abgegrenzte Verantwortung, hierarchische Reviews, unabhängige Prüfruns und kontrollierte Integration sollen konsistente Änderungen zuverlässiger ermöglichen als ein einzelner Agent, der alle Verpflichtungen allein im Blick behalten muss. Das ist eine zu prüfende These. Zuverlässigkeit, weniger menschlicher Prüfaufwand und Vorteile gegenüber einfacheren Agentenansätzen müssen an Vergleichen und realen Projekten belegt werden.

## 2. V1-Scope

### 2.1 Liefergegenstand

V1 liefert **Model → Artifacts**: deterministische Vorbereitung, Assessment, Implement, Verify, frische Korrekturruns, dauerhafte Ergebnisse und kontrollierte Publication. Bedient wird V1 über einen schmalen Apply-Einstieg und die Modellwerkzeuge Check, Context und Docs.

Dazu gehören Registry- und Ownership-Pflege, ein real nutzbarer Harness, normale CLI-Konto-Authentifizierung, Run-lokale Container und Git-Arbeitsstände, persistente Shared Notes sowie die nötigen technischen und fachlichen Nachweise.

### 2.2 Nicht beauftragt

Extract und andere Artifacts→Model-Operationen, Cockpit beziehungsweise grafische UI, interaktives Modell-Authoring und die Migration dieser Altverbraucher gehören nicht zu V1. Ihre früheren Schnittstellen erzeugen keine Anforderungen an den Ausführungskern.

Setup gehört nicht zu V1. Getestete explizite Konfigurationsbeispiele ermöglichen den Apply-Aufruf; ein eigenes Onboarding-Produkt wird nicht gebaut. Normales Lesen vorhandener Artefakte bleibt notwendig. Ausgeschlossen ist, daraus während Apply eine neue kanonische Wahrheit zu machen.

### 2.3 Providerumfang (D18)

V1 wird mit **Codex** und normalem Konto-Login abgenommen. Claude ist die nächste Providerabbildung. Sie wird auf dem stabilisierten Session- und Werkzeugvertrag (Abschnitte 13 und 14) gebaut und eigenständig nachgewiesen. Bis dahin führt das Produkt keinen nicht funktionsfähigen Claude-Pfad mit.

Der Ansatz bleibt providerneutral. Jeder weitere Provider braucht einen eigenen Adapter und eigene Nachweise; ein erfolgreicher Codex-Lauf beweist nichts über einen anderen Provider.

### 2.4 Später

Weitere Commands und benutzerdefinierte Workflows sollen Runs künftig kombinieren können. Das setzt die unabhängige Run-Grenze voraus, aber keine Workflow-DSL in V1. Längerfristiges Managergedächtnis, semantische Suche, automatische Wissensverdichtung, ein grafischer Workflow-Designer, dynamische Organisation, langlebige Agentencontainer und eine eigene Code-Mode-Runtime sind keine V1-Anforderungen.

## 3. Qualitätsprinzipien

### 3.1 Architekturqualität vor Wiederverwendung

Die Zielarchitektur bestimmt die Implementierung. Alte Typen, Paketgrenzen und Laufzeitpfade sind keine Vorgaben; historische Arbeit liefert Erkenntnisse und Regressionserwartungen. Ein Rewrite ist ebenso wenig Selbstzweck: Kohärente Grundlagen bleiben erhalten, und ein neu geschriebenes, überabstrahiertes System wäre ebenfalls am Ziel vorbei.

### 3.2 Pragmatischer Determinismus

Technisch eindeutig feststellbare und relevante Sachverhalte werden deterministisch ermittelt: Identitäten, Referenzen, fixierte Git-Stände, Dateizuordnung, tatsächliche Checkresultate und Ressourcenabschluss. Daraus folgt keine Pflicht, jeden Arbeitsschritt zu klassifizieren, freizugeben oder mit Belegprotokollen zu begleiten. Der Nutzen einer Kontrolle muss ihre Komplexität rechtfertigen.

### 3.3 Managerautonomie mit klarer Verantwortung

Manager arbeiten selbstständig in ihrem Bereich, planen lokal, koordinieren ihre Kinder und bewerten deren Ergebnisse. Der Host unterstützt diese Organisation. Er trifft keine fachlichen Architekturentscheidungen und ersetzt den Vater nicht durch eine Reparatur- oder Freigabemaschine.

### 3.4 Einfachheit und Kohäsion

Wenige klare Verantwortlichkeiten, passende Abhängigkeitsrichtungen und gut lesbarer Code sind Abnahmekriterien. Mehr Packages, Interfaces oder Statuswerte bedeuten nicht mehr Qualität. Ein einziger großer Orchestrator ist aber auch keine Vereinfachung.

### 3.5 Fokus und Erhaltung

Ein Auftrag umfasst das gültige Modell, tatsächliche Findings und notwendige Folgen. Konforme Artefakte werden bewahrt. Optionale Verbesserungen werden notiert, nicht ungefragt umgesetzt. Ein unveränderter, konformer Kandidat ist ein gültiges Ergebnis.

### 3.6 Ehrliche Ergebnisse

Fachliche Zustimmung, technischer Erfolg, gespeicherte Ergebnisse, funktionierende Providerintegration und Owner-Abnahme sind verschiedene Aussagen. Unsichere oder unvollständige Zustände werden nicht zu Erfolgen umgedeutet. Spätere Erfolge ändern frühere Fehlschläge nicht.

### 3.7 Aufgaben statt Prozess (D13)

Manager erhalten Rolle, Organisationsregeln, Kontext, Aufgabe und Antwortform – nicht die Prozesssteuerung des Hosts. Zulassung, Aufwecken, Drain, Abschluss und ähnliche Zustände verwaltet der Host selbst. Lehnt er eine Aktion ab, nennt das Werkzeugergebnis den konkreten Grund und die möglichen nächsten Schritte.

Ein wiederkehrender Fehler im Zusammenspiel von Agent und Host wird im Host-Zustand oder im Werkzeugvertrag behoben, nicht durch zusätzlichen Prompttext.

## 4. Begriffe

| Begriff | Bedeutung |
|---|---|
| **Kanonisches Modell** | Autoritative Beschreibung der beabsichtigten Projektwelt. |
| **Package / Definition** | Organisationseinheiten beziehungsweise typisierte Aussagen der kanonischen Sprache. |
| **Artefakt** | Konkrete Realisierung, etwa Code, Test, Dokumentation, Build- oder Konfigurationsdatei. |
| **Manager** | Agent mit aus dem Modell abgeleiteter Verantwortung. |
| **Auftrag (Assignment)** | Fachliche Arbeit eines Managers innerhalb eines Runs. Nicht identisch mit einer Nachricht. |
| **Run** | Eigenständig ausführbares Mandat durch die vollständige Managerorganisation. |
| **Mandat** | Auftrag und Ergebniserwartung eines Runs; in V1 Assessment, Implement oder Verify. |
| **Workflow** | Komposition von Runs, ihren Eingaben, Ergebnissen und Übergängen. |
| **Command** | Benannte Anwendungsoperation einschließlich Voraussetzungen, Gesamtbudget und gegebenenfalls Workflow. |
| **Harness** | Vorbereitete manager-, mandat- und providerspezifische Arbeitsumgebung. |
| **Workspace** | Veränderliche, private Arbeitskopie eines Managers während eines Runs. |
| **Übergabe (Handoff)** | Fixierter Arbeits- oder Prüfstand eines Managers mit dem für das Review nötigen Kontext. |
| **Kandidat** | Fixierter, dauerhaft referenzierbarer Git-Realisierungsstand. |
| **Urteil** | Fachliche Entscheidung eines Managers zu seinem eigenen Auftrag: Zustimmung, Ablehnung oder Modellblocker; bei Implement die Abgabe eines Kandidaten. Implement gibt sich nie selbst frei. |
| **Review** | Entscheidung eines Vaters über die fixierte Übergabe eines direkten Kindes. |
| **Finding** | Für das aktuelle Mandat erforderliche Feststellung oder Korrektur. |
| **Shared Note** | Persistente, nichtkanonische Beobachtung ohne automatische Auftragswirkung. |
| **Publication** | Kontrollierte Übertragung eines akzeptierten Kandidaten in das Nutzerrepository. Kein automatischer Commit oder Push. |

**Ein Run ist kein Workflow. Ein Command kann einen Workflow enthalten. Eine technische Nachricht ist weder ein Run noch automatisch ein neuer Auftrag.**

## 5. Kanonisches Modell und deterministische Vorbereitung

### 5.1 Bedeutung des Modells

Das Modell kann fachliche Konzepte, Use Cases, Architekturgrenzen, Regeln, Beziehungen und Entwicklungsverpflichtungen beschreiben. Identitäten, Kinds, Felder, Imports und Referenzen folgen der vorhandenen Sprache; freie Prosa ergänzt die typisierte Struktur dort, wo Bedeutung nicht sinnvoll auf entscheidbare Felder reduziert wird.

Dieses Konzept definiert keine zusätzliche Modellgrammatik und keine neuen Pflicht-Primitive. Eine Projektkonvention, etwa „ein Use Case, genau ein Handler“, kann modelliert werden, ohne in den Kern eingebaut zu werden.

### 5.2 Modellautorität

Ein Apply arbeitet gegen eine fixierte, erfolgreich geprüfte Modellbasis und ihre kontrollierten Eingaben. Realisierungsagenten ändern nicht stillschweigend Anforderungen, Ownership-Regeln oder Ignore-Ausnahmen, um einen Konflikt zu beseitigen. Ein notwendiger materieller Modellentscheid wird als Blocker sichtbar, außerhalb des Apply geklärt und erst mit einer neu geprüften Modellbasis erneut realisiert.

### 5.3 Initiales Gate

Vor dem agentischen Workflow werden Modellbasis und relevante Projektstrukturen deterministisch geprüft. Bei ungültigem oder unvollständig erfasstem Ausgangsstand starten keine Manager; die tatsächlichen Diagnosen werden zurückgegeben. Die Projektvorbereitung verbindet den vorhandenen Compiler mit Registry, Accounting, Organisationsableitung und Kontextselektion. Es entsteht kein zweiter Compiler in der Agentenruntime.

### 5.4 Grenzen der Prüfung

Strukturelle Gültigkeit ist keine semantische Vollständigkeit. Eine korrekt registrierte Datei ist nicht automatisch fachlich richtig; eine gültige Definition beweist nicht, dass ihr Verhalten implementiert ist. Noch nicht erzeugte Artefakte sind bei frischer Realisierung kein Modellfehler – Assessment unterscheidet fehlende Realisierung von fehlender Modellentscheidung. Vorhandene unerklärte Dateien bleiben ein Accountingproblem des Ausgangsstands.

### 5.5 Geschütztes Fundament und Selbstmodell

Die Semantik von Sprache, Compiler, Snapshot-Werten, Organisationsableitung und Realisierungszuordnung bleibt geschützt; Strukturverbesserungen dürfen diese Garantien nicht beiläufig ändern. Markitects eigenes kanonisches Selbstmodell beschreibt die tatsächlich gewählte Architektur. Seine Pflege ist keine Änderung der Modellsprache.

## 6. Registry, Ownership und vollständige Dateizuordnung

### 6.1 Lokale Shards und globale Sicht

Jeder verantwortliche Manager pflegt den Registry-Shard seines Packages. Er ordnet seine Artefakte den relevanten Definitionen zu und begründet die Zuordnung. Ein Artefakt kann mehrere Definitionen realisieren, hat aber genau eine Verantwortung.

Die globale Registry wird deterministisch aus den Shards abgeleitet, nicht als zweite Wahrheit gepflegt. Beim Integrieren eines Kinderergebnisses bleibt dessen Registrierung im Shard des Kindes.

### 6.2 Vollständige Erfassung

Für jede relevante beobachtete Datei steht eindeutig fest, zu welcher Kategorie sie gehört.

| Kategorie | Behandlung |
|---|---|
| Realisierungsartefakt | Gültige Registrierung und Zuständigkeit erforderlich. |
| Kanonische Modelleingabe | Autoritative, während Apply geschützte Projektabsicht. |
| Markitect-Steuereingabe | Kontrollierte Eingabe, etwa Ausführungs- oder Ignore-Regeln. |
| Operativer Zustand | Vom Produktkandidaten technisch abgegrenzt, nicht künstlich registriert. |
| Ausdrücklich ignorierte Datei | Explizite Ausnahme, keine verdeckte Umgehung einer Modellpflicht. |

`.markitect/ignore.yaml` ist maßgeblich; `.gitignore` ersetzt diese Zuordnung nicht. Kanonische Modell- und Registry-Dateien verschwinden nicht durch eine pauschale Verzeichnisausnahme aus der Validierung.

### 6.3 Registry-Pflege ist Realisierung, Projektabsicht nicht

Manager registrieren neue Artefakte, aktualisieren Zuordnungen bei Verschiebungen und bereinigen sie bei zulässigen Löschungen. Das ist Teil der Realisierung. Anforderungen, Ownership-Policy und Ignore-Regeln dürfen dagegen nicht geändert werden, um einen Check zu umgehen. „Modell unverändert“ heißt nicht „Registry eingefroren“.

### 6.4 Zuordnung ist nicht Urheberschaft

Accounting sagt, wem eine Datei zugeordnet ist und welche Definitionen sie adressiert – nicht, wer sie geändert hat. Übergaben enthalten deshalb die tatsächliche Aufgabenbasis, den fixierten Ergebnisstand und den vollständigen Diff.

Review und Verify berücksichtigen akzeptierte Kinderintegrationen; diese sind keine unerlaubte Eigenarbeit des Vaters. Kinderdateien werden aber nicht pauschal aus dem Diff des Vaters entfernt, sonst würden spätere eigene Eingriffe unsichtbar. Dafür genügen Git-Basen und Diffs. Ein per-write-ACL-System oder ein globales Herkunftsprotokoll ist nicht vorgesehen.

## 7. Modellabgeleitete Managerorganisation

### 7.1 Das Modell ist die Architektur

Die Organisation wird aus dem gültigen Modell abgeleitet. Es gibt keine zweite, manuell gepflegte Managerhierarchie. Sie umfasst Root und alle lokalen Package-Manager, einschließlich leerer und intermediärer Packages. Die bestehenden Regeln für externe Modellabhängigkeiten werden nicht beiläufig zu einer zusätzlichen ausführbaren Organisation erweitert.

### 7.2 Vollständige Teilnahme in jedem Mandat

Assessment, Implement und Verify verwenden dieselbe vollständige Organisation; das Mandat ändert die Aufgabe, nicht den Zuständigkeitsbaum. Root-only ist nur korrekt, wenn das Modell nur Root umfasst. Teilnahme heißt tatsächliche Bearbeitung der Verantwortung, nicht nur ein gestarteter Container.

### 7.3 Verantwortung von Root und lokalen Managern

Ein lokaler Manager realisiert beziehungsweise prüft seine eigenen Verpflichtungen. Ein Vater verantwortet zusätzlich die Delegation an seine direkten Kinder, die Abstimmung ihrer Schnittstellen, ihre Reviews und die Integration nach oben.

Root bildet das integrierte Gesamtergebnis. Er übernimmt nicht routinemäßig die Facharbeit der Kinder, überspringt ihre Zuständigkeiten nicht und ersetzt erforderliche Kinderentscheidungen nicht durch eigene Zustimmung.

### 7.4 Keine zweite Delegationsorganisation

Native Provider-Unteragenten dürfen die modellierte Organisation nicht umgehen. Im V1-Produktpfad läuft Delegation über Markitects Managerstruktur. Die Chats, die Markitect entwickeln, dürfen im Rahmen ihrer Aufträge Subagenten einsetzen; Entwicklungsorganisation und Produktlaufzeit sind getrennte Ebenen.

## 8. Zusammenarbeit im Run

### 8.1 Lokale Planung und gemeinsame Voraussetzungen

Jeder Manager plant innerhalb seines Auftrags. Eltern klären gemeinsame Schnittstellen und Voraussetzungen, bevor abhängige Kinderaufträge starten. Es gibt keinen globalen Plan-Run und keinen Host, der fachlich plant.

Unabhängige Kinder arbeiten parallel. Braucht ein Kind das integrierte Ergebnis eines Geschwisters, koordiniert der gemeinsame Vater die Reihenfolge. Eine bereits gestartete private Kopie erhält spätere Änderungen des Vaters oder der Geschwister nicht live, sondern nur an klaren Übergabegrenzen, etwa bei einer Korrekturdelegation.

### 8.2 Kommunikation entlang der Hierarchie (D10)

Delegation, Fragen, Antworten, Abstimmung und Eskalation folgen dem Baum. Ein Manager fragt nur seinen direkten Vater oder seine direkten Kinder; Antworten gehen an den Fragenden zurück. Abstimmung zwischen Geschwistern läuft über den gemeinsamen Vater, der selbst weiterfragt oder entscheidet; Abstimmung über mehrere Ebenen läuft Schritt für Schritt über die Manager dazwischen. Root fragt niemanden außerhalb des Runs.

Der Host leitet die erlaubten Beziehungen aus der Topologie ab und lehnt andere Adressen mit Begründung ab. Eine eigene ACL- oder Routingsprache gibt es nicht.

Antworten können asynchron eintreffen. Ein wartender Manager bearbeitet eingehende Fragen, statt seinen Partner zu blockieren. Der Host transportiert und korreliert Nachrichten; er beantwortet keine fachlichen Fragen.

### 8.3 Übergabe, Review und Integration

Ein Kind liefert einen fixierten Stand und die für seine Bewertung nötigen Informationen. Der Vater prüft die tatsächliche Übergabe – den Diff, nicht eine Erfolgserzählung – und nimmt sie an, fordert eine Korrektur an oder eskaliert einen Modellblocker.

Angenommene Ergebnisse werden nach oben integriert, sobald sie vorliegen, ohne auf alle Geschwister zu warten. Abhängigkeiten oder Konflikte können eine andere Reihenfolge begründen; der zuständige Manager koordiniert sie, kein globales Host-Planungsframework.

### 8.4 Fehlerhafte Übergaben sind Elternverantwortung (D11)

Bei Implement wird jeder abgegebene Kinderkandidat fixiert und zusammen mit dem Ergebnis der deterministischen Prüfung an den Vater übergeben – auch wenn die Prüfung fehlschlägt. Der Host schickt eine ungültige Abgabe nicht still an das Kind zurück.

Der Vater erkennt aus der Diagnose, ob lokale Korrektur, eine Schnittstellenklärung oder eine andere Integrationsreihenfolge nötig ist. Er weist die Abgabe mit Findings zurück, klärt die Schnittstelle oder eskaliert. Einen ungültigen Kandidaten kann er weder annehmen noch integrieren.

Lokale Selbstkorrektur bleibt möglich: Ein Kind kann dieselbe deterministische Prüfung für seinen aktuellen Stand vor der Abgabe über ein Werkzeug abrufen (Abschnitt 14). Das ist ein Angebot, kein Pflichtschritt. Root hat keinen Vater; seine Diagnose liefert der Host direkt an Root zurück (Kontrollpunkt vor dem Root-Abschluss).

In Assessment und Verify ist eine Kinderübergabe ein Urteil über den fixierten Eingabekandidaten. Hilfsdateien der privaten Kopie werden nicht übernommen. Ein Prüfergebnis zum Eingabekandidaten ist Information für das Urteil, keine Integrationssperre; ein begründetes Reject eines ungültigen Kandidaten ist ein reguläres Ergebnis.

### 8.5 Korrektur im Run und zwischen Runs

Eine Rückgabe durch den Vater während Implement bleibt im selben Run. Das Kind behält Gespräch und Arbeitsstand und erhält die Findings sowie die aktuelle Basis des Vaters.

Ein Reject des unabhängigen Verify führt dagegen zu einem neuen Implement-Run; frühere Implement-Gespräche werden nicht fortgesetzt. Beide Korrekturarten werden technisch und in Berichten getrennt.

## 9. Run

### 9.1 Eigenständige Aufrufgrenze

Ein Run führt genau ein Mandat durch die Managerorganisation aus. Er ist unabhängig von Apply ausführbar und testbar; Assessment oder Verify lassen sich isoliert untersuchen, ohne eigene Orchestrierungswege.

Eingaben sind eine geprüfte Modell- und Projektbindung, die abgeleitete Organisation, ein fixierter Ausgangskandidat, das Mandat, optionale verpflichtende Findings, Ausführungs- und Harnesskonfiguration sowie endliche Ressourcen.

### 9.2 Verantwortung des Runs

Der Run verantwortet Umgebungsaufbau, Session- und Workspace-Lebensdauer, hierarchische Zusammenarbeit, Übergaben, Integration, die Kontrollpunkte im Run, Ergebnisfixierung, Kandidatensicherung, Cleanup und die Speicherung seines Endergebnisses.

Er wählt nicht den nächsten Run und veröffentlicht nichts in das Nutzerrepository. Seine erfolgreiche Rückgabe hängt auch beim Standalone-Aufruf nicht davon ab, dass ein Aufrufer Abschlussarbeiten nachholt.

### 9.3 Lebenszyklus

```text
Gültige gebundene Run-Eingaben
    → Run-Umgebung und Manager-Harnesses vorbereiten
    → Run-lokalen Git Workspace Service und Manager starten
    → Auftrag durch die vollständige Organisation bearbeiten
    → tatsächliche Übergaben und Gesamtergebnis fixieren
    → Kandidaten dauerhaft sichern und unabhängig wieder öffnen können
    → alle Run-Ressourcen kontrolliert schließen
    → technischen und fachlichen Ausgang speichern und zurücklesen
    → geschlossenes Ergebnis an den Aufrufer liefern
```

Das ist eine Host-Verantwortung, kein vom Agenten auszufüllendes Belegprotokoll. Ein Executor mit klaren Hilfsverantwortlichkeiten genügt; eine Kaskade von Lifecycle-Services ist nicht vorgesehen.

### 9.4 Erreichbarkeit innerhalb des Runs

Das Ende einer Providerantwort beendet die Manager-Session nicht. Ein Manager, der abgegeben hat, bleibt für Fragen und Korrekturen im selben Run erreichbar; seine fixierte Übergabe und spätere Kommunikation sind unterscheidbar. Erst das Run-Ende schließt die Umgebung. Neue Runs erhalten neue Gespräche und Arbeitsumgebungen.

## 10. Command, Workflow und Apply

### 10.1 Command

Der Command stellt die benannte Produktoperation bereit. Er bindet Eingaben, veranlasst die Initialprüfung, besitzt das Gesamtbudget, ruft den Workflow auf und verantwortet Publication und Gesamtergebnis. CLI und spätere Oberflächen verwenden dieselbe Anwendungsoperation; der CLI-Parser enthält keine Fachlogik. Der Command implementiert weder Managerkommunikation noch Git-Merges.

### 10.2 Workflow

Ein Workflow beschreibt Reihenfolge, Eingaben und Ergebnisübergaben von Runs, Verzweigungen, Wiederholungen, Erfolg und Abbruch. Er bleibt verständlich und später kombinierbar. Er kennt keine Container, Providerprozesse oder Arbeitskopien und arbeitet nur mit geschlossenen Run-Ergebnissen.

### 10.3 Gültigkeit geschlossener Ergebnisse (D15)

Ob ein Run-Ergebnis vollständig, gebunden und sauber geschlossen ist – Bindung, technischer Abschluss, Closure aller Run-Ressourcen, zulässige Entscheidung für das Mandat und Pflicht-Findings –, prüft genau eine Validierung. Sie gehört zum Run-Vertrag. Workflow und Publication verwenden sie, ohne interne Ressourcennamen des Runs zu kennen.

### 10.4 Apply-Ablauf

```text
Initiale Modell- und Strukturprüfung
    ├─ ungültig / unvollständig → Ende ohne Agentenstart
    └─ gültig
         → Assessment
             ├─ Reject / Modellblocker → Ende ohne Publication
             └─ Approve
                  → Implement
                      ├─ nicht abgeschlossen / Modellblocker → Ende
                      └─ gültiger fertiger Kandidat
                           → Verify
                               ├─ Realisierungs-Reject
                               │    → neues Implement mit Findings
                               │    → neues Verify
                               ├─ Modellblocker / technischer Abbruch → Ende
                               └─ Approve
                                    → abschließende Checks und Guards
                                    → kontrollierte Publication
```

Alle Iterationen teilen das ursprüngliche Command-Budget. Eine neue Run-ID setzt weder Zeit- noch Nachrichtenressourcen zurück. Die Schleife läuft bis zur Zustimmung oder zum ehrlichen Abbruch innerhalb endlicher Grenzen.

### 10.5 Erweiterbarkeit (D16)

V1 kennt genau drei Mandate. Die Run-Aufrufgrenze ist für weitere Commands wiederverwendbar. Neue Mandate und ihre Ergebnis- oder Fähigkeitsanforderungen entstehen erst aus einem tatsächlichen späteren Command. V1 braucht weder DSL noch grafisches Workflow-System.

## 11. Mandate

### 11.1 Assessment

Assessment prüft die semantische Bereitschaft des strukturell gültigen Modells. Jeder Manager betrachtet seinen Bereich und relevante Schnittstellen. Materielle Widersprüche, notwendige offene Entscheidungen und nicht auflösbare Mehrdeutigkeiten werden mit Findings zurückgegeben.

Untersuchungen, Befehle und etwa ein begründeter Build-Versuch sind zulässig. Assessment implementiert keine Realisierung und verändert die Projektabsicht nicht; Hilfsdateien in privaten Kopien werden nicht übernommen. Fehlende oder veraltete Artefakte sind Implementierungsbedarf, kein Grund zur Ablehnung. Approve bedeutet „ausreichende Modellgrundlage für die Realisierung“, nicht „das Produkt ist korrekt“, und überspringt Implement nicht.

### 11.2 Implement

Implement gleicht die bestehende Realisierung mit Modell, erforderlichen Checks und tatsächlichen Findings ab. Manager ändern die dafür nötigen Artefakte in ihrer Verantwortung, pflegen ihre Registry, delegieren und integrieren geprüfte Kinderergebnisse.

Notwendige Folgeänderungen gehören dazu. Zusätzliche Testideen, kosmetische Änderungen oder Refactorings ohne Bezug zu geltenden Verpflichtungen werden nicht umgesetzt, nur um einen Diff zu erzeugen. Implement liefert einen fixierten integrierten Kandidaten und genehmigt dessen Publication nicht selbst. Ohne notwendige Änderungen ist die unveränderte Abgabe nach vollständiger Arbeitsteilung, Prüfung und Review gültig.

### 11.3 Verify

Verify ist ein frischer Run über exakt den zu prüfenden Kandidaten. Die vollständige Organisation bewertet unabhängig Modellkonformität, Fachlichkeit, Schnittstellen, Integration, relevante Tests und Realisierungszuordnung.

Verify repariert den Kandidaten nicht; eigene Untersuchungen und Prüfdateien bleiben lokal. Das Ergebnis ist Approve, Reject mit notwendigen Realisierungsfindings oder ein Modellblocker. Ein Reject ist ein reguläres Prüfergebnis, kein technischer Fehlschlag.

### 11.4 Korrekturen nach Verify

Ein Realisierungs-Reject übergibt Modellbindung, letzten Kandidaten und tatsächliche Findings an einen neuen Implement-Run, danach folgt ein neues Verify. Weder wird die alte Implement-Organisation fortgesetzt noch erhält der neue Prüfer deren Selbstbewertung als Autorität. Ein spät entdeckter materieller Modellwiderspruch beendet den Command ohne Publication; die Modellfrage wird außerhalb geklärt.

## 12. Prüfstrategie und vier Kontrollpunkte

### 12.1 Ein gemeinsamer Prüfmechanismus

Dieselben Compiler-, Registry- und Accountingmechanismen werden an verschiedenen Stellen auf passende fixierte Stände angewendet; es entstehen keine vier abweichenden Validatoren. Günstige Strukturchecks erfolgen häufig. Nicht jede isolierte Kinderkopie muss die vollständige Projekttestsuite ausführen: Herkunft einer Prüfverpflichtung und sinnvoller Ausführungsort eines Tests sind verschiedene Eigenschaften.

### 12.2 Die vier Kontrollpunkte

| Kontrollpunkt | Prüfung und Konsequenz |
|---|---|
| **Vor Workflowbeginn** | Gültige Modellbasis und anwendbare Projektstruktur. Bei Fehler Abbruch vor dem agentischen Workflow und vor Run-Ressourcen. |
| **Bei Kinderübergaben** | Fixierter Übergabestand und Accounting. Bei Implement geht die Diagnose immer mit der Übergabe an den Vater, und ein ungültiger Kandidat ist nicht integrierbar (D11). |
| **Vor erfolgreichem Root-Abschluss** | Integrierter Gesamtstand, relevante Checks und Bindungen. Kein erfolgreiches Implement und kein Approve bei ungelöster technischer Ungültigkeit. |
| **Vor Publication** | Exakt akzeptierter Kandidat, erforderliche Prüfungen und tatsächliche Quell- und Zielguards. Keine unzulässige Übertragung. |

### 12.3 Zwischenstände und Prüfmandate

Zwischenstände können unvollständig sein, solange andere Kinder noch arbeiten; Diagnosen machen diesen Zusammenhang sichtbar. Der Host verwandelt vorübergehende Unvollständigkeit nicht in einen Command-Abbruch. Verify darf einen fehlerhaften Kandidaten regulär ablehnen und muss ihn nicht verändern, um den eigenen Abschlusscheck grün zu bekommen. Für eine Zustimmung bleiben die Gültigkeitsbedingungen verbindlich.

### 12.4 Keine Check-Rituale

Der Host ermittelt technische Resultate und stellt sie den Managern zur Verfügung. Manager tippen keine bekannten Identitäten erneut ein und erzeugen keine Check-Quittungen, Disputes oder Evidence-Matrizen, um gewöhnliche Arbeit fortsetzen zu dürfen.

## 13. Auftrag, Nachricht und Urteil

### 13.1 Die grundlegende Trennung

Ein Manager hat einen Auftrag. Nachrichten, Fragen und Antworten dienen seiner Erfüllung; sie sind keine neuen Aufträge. Das Ende einer Providerantwort ist kein fachlicher Abschluss des Managers oder des Runs. Freie Abschlussprosa ist kein technischer Erfolgsnachweis. Das fachliche Ergebnis hält der Manager über eine kleine explizite Entscheidung fest, das Urteil.

### 13.2 Das Urteil gehört zum Auftrag (D12)

Ein Urteil gilt für den aktuellen Auftragsstand – Aufgabe und verbindliche Findings – und für den bewerteten Stand: bei Implement den eigenen Arbeitsstand zum Zeitpunkt des Urteils, bei Assessment und Verify den fixierten Eingabekandidaten. Es bleibt gültig, bis sich eines davon ändert. Ungültig wird es durch:

- eine Rückweisung oder Korrekturdelegation des Vaters mit Findings,
- neue verpflichtende Findings, die den Auftrag betreffen,
- eine Änderung des bewerteten Stands nach dem Urteil,
- ein spätes verpflichtendes Finding, Reject oder einen Modellblocker eines Kindes zu einer bereits angenommenen Übergabe.

Eine Nachricht, die nur Fragen oder Antworten behandelt, verlangt kein neues Urteil; ein gültiges Urteil bleibt bestehen. Fehlt am Ende einer Providerantwort für einen aktiven Auftrag ein gültiges Urteil, macht der Host den konkreten Grund sichtbar, etwa „dein Arbeitsstand hat sich nach deinem Urteil geändert“ oder „das Review von Kind X fehlt“. Das ist eine Diagnose des Hosts, keine allgemeine Promptregel; ihre Form ist technisch auszugestalten.

Der Host führt dafür pro Manager einen kleinen, expliziten Auftragszustand statt verstreuter Einzelmerkmale. Jede Invalidierung folgt aus einem benennbaren Ereignis.

### 13.3 Maßgebliche Gegenbeispiele

| Situation | Festgelegte Konsequenz |
|---|---|
| Eine Frage wird beantwortet | Kein neuer Auftrag und kein neues Urteil. |
| Eine eigene Antwort steht noch aus | Eingehende Fragen bleiben bearbeitbar; kein Deadlock. |
| Neue verpflichtende Findings kommen an | Das bisherige Urteil wird ungültig; ein neues ist nötig. |
| Der bewertete Stand ändert sich | Eine Zustimmung zum alten Stand gilt nicht für den neuen. |
| Eine fixierte Übergabe ist erfolgt | Spätere Kommunikation verändert den geprüften Stand nicht. |
| Ein später Befund widerlegt eine Annahme | Ein spätes Reject oder ein Modellblocker mit Findings bleibt wirksam. Die Annahme des Vaters über diese Übergabe und jedes darauf beruhende Urteil darüber werden ungültig und müssen neu getroffen werden, bevor Root erfolgreich abschließen kann. Ein Modellblocker verhindert einen erfolgreichen Abschluss. |

## 14. Manager-Werkzeuge (D13)

### 14.1 Fester kleiner Werkzeugsatz

| Werkzeug | Zweck |
|---|---|
| `delegate` | Ein Vater erteilt einem direkten Kind eine Aufgabe mit Findings; nach einem Review auch eine Korrektur. |
| `await` | Das nächste Ereignis empfangen: Auftrag, Frage, Antwort oder fixierte Kinderübergabe mit Prüfergebnis. |
| `review` | Ein Vater entscheidet über die fixierte Übergabe eines direkten Kindes: annehmen, zurückweisen oder Modellblocker. |
| `ask` / `reply` | Fragen und Antworten entlang der Hierarchie, asynchron. |
| `judge` | Das eigene Urteil zum Auftrag mit Findings festhalten. |
| `check` | Die deterministische Prüfung des eigenen aktuellen Stands abrufen, zur optionalen Selbstprüfung vor der Abgabe. |
| `read_notes` / `append_note` | Shared Notes seitenweise lesen und Beobachtungen anhängen. |

Jeder fachliche Zweck hat genau ein Werkzeug.

### 14.2 Grundsätze

- Was der Host kennt, fragt er nicht ab: tatsächlicher Manager, Run, Mandat und eigene Commits. Findings brauchen keine Absenderangabe. Ein eindeutiger Verweis auf den geprüften Stand bleibt beim Review zulässig, damit keine veraltete Zustimmung entsteht.
- Werkzeuge treiben den Host nicht über Quittungen an. Eine Ablehnung nennt den konkreten Grund und was möglich ist (Abschnitt 3.7).
- Die Semantik jedes Werkzeugs hat genau eine Implementierung im Run. Provideradapter bilden sie nur ab. MCP, CLI oder typisierte APIs sind Transportwege, kein Architekturprinzip.

## 15. Harness

### 15.1 Zweck

Der Harness bereitet den Manager wie einen Mitarbeiter auf seine Verantwortung vor: Rolle, Organigramm, Zuständigkeit, Modellkontext, Arbeitsregeln und Werkzeuge sind verlässlich verfügbar. Die Auftragsnachricht bleibt kurz und auf die aktuelle Arbeit konzentriert. Ein Harness ist nicht nur ein Text mit Hash: Er muss vom tatsächlich gestarteten Provider geladen und im Container benutzt werden.

### 15.2 Fünf kontrollierte Quellen

| Quelle | Inhalt und Verantwortung |
|---|---|
| **Markitect Core** | Versionierte allgemeine Regeln für Delegation, Zusammenarbeit, Ownership, Registry, Reviews, Findings und Notes. Nicht von jedem Projekt neu zu definieren. |
| **Projekt** | Kanonische Fachlichkeit, Architektur, Policies, Qualitätsverpflichtungen und ausdrücklich gebundene Projektkonventionen. |
| **Manager** | Eigene Rolle, Vater, direkte Kinder, relevante Kollegen, Zuständigkeiten, Registry-Shard, Schnittstellen und Projektnavigation. |
| **Run** | Mandat, fixierte Eingaben, relevante Findings, erlaubte Auswirkungen und Ergebniserwartung. |
| **Provider** | Umsetzung durch tatsächlich unterstützte native Instruktionen, Werkzeuge und Konfiguration. |

Markitect definiert, wie seine Organisation arbeitet; das Projektmodell definiert, welche Welt sie realisiert. Provideranpassungen ändern diese Bedeutung nicht. Prozesssteuerung des Hosts gehört in keine dieser Quellen (Abschnitt 3.7).

### 15.3 Reproduzierbarkeit und Identität

Gleiche gebundene Eingaben – Regelversion, Modellstand, Manager, Mandat, Providerprofil, Vorlagen und Werkzeugdefinitionen – erzeugen dieselbe wirksame Harness-Konfiguration. Eine nachvollziehbare Harness-Identität zeigt, welche Anweisungen und Fähigkeiten bereitgestellt wurden. Frische operative Namen, Containerkennungen oder Geheimnisse gehören nicht dazu. Deterministische Erzeugung bedeutet nicht deterministisches Modellverhalten.

### 15.4 Kontext

Der Manager erhält eine kompakte Orientierung und den für seine Verantwortung notwendigen Kontext. Verbindliche Informationen werden weder still gekürzt noch versteckt. Historische Notizen und Prozessbeschreibungen werden nicht in jede Startinstruktion kopiert.

### 15.5 V1-Form des Harness (D14)

In V1 wird der Harness über die nativen Base- und Developer-Instruktionen des Providers bereitgestellt. Sie enthalten die Markitect-Regeln und den vollständigen statisch ausgewählten Pflichtkontext. Automatisch geladene Repository-Instruktionen werden bewusst behandelt, etwa abgeschaltet; projektgebundene Instruktionen gelangen nur über den kontrollierten Harness in die Umgebung. So übersteuern weder Repository-Dateien noch Providerprofile die kanonische Absicht oder die Markitect-Regeln unkontrolliert. Semantisch unklare Konflikte zwischen Instruktionen bleiben sichtbar, statt durch ein Policy-Framework verdeckt zu werden. Übersteigt der Pflichtkontext eine Grenze, scheitert die Vorbereitung sichtbar; gekürzt wird nicht.

Modulare Dateien, Skills oder nachladbarer Kontext kommen hinzu, sobald ein konkreter Größen- oder Bedienfall ihren Nutzen zeigt. Für unbekannte kompatible Provider ist ein generischer AGENTS.md-Harness als Best-Effort-Instruktionsdarstellung vorgesehen. Er ersetzt keinen ausführbaren Adapter, keine Managerkommunikation und keine notwendigen technischen Fähigkeiten; ein unbekannter Agent ist nicht allein durch diese Datei unterstützt. Ob er zum V1-Abnahmeumfang gehört, wird mit der Provider-Matrix festgehalten (Abschnitt 28.2).

### 15.6 Technische Grenzen bleiben außerhalb der Instruktionen

Wichtige Invarianten werden unabhängig vom Gehorsam des Agenten geprüft: Modellbindung, Registry, fixierte Git-Stände, tatsächliche Checks, Ressourcenabschluss und Publikationsvoraussetzungen. Das begründet keine per-write-Dateifreigaben und keine Belegprotokolle.

## 16. Provider und Authentifizierung

### 16.1 Providergrenze

Der Provideradapter startet und beendet die native Runtime, überträgt Nachrichten, vermittelt Werkzeuge und beobachtet tatsächliche Antwort- und Prozesszustände. Er entscheidet weder über die nächste Workflowstufe noch über Modellkonformität. Native Protokollbegriffe wie `turn/start` existieren nur im Adapter; der Kern leitet daraus keine fachlichen Turns, Empty-Turn-Klassifikatoren oder eine Pass-Maschine ab.

### 16.2 Konto-Login und Geheimnisschutz

Der V1-Produktpfad verwendet normale CLI-Konto-Authentifizierung. Eine API-Key-Produktvariante wird nicht mitgeführt. Zugangsdaten gehören nicht in Kandidaten, Harnesstexte, Shared Notes oder ungeschützte Protokolle; Vorbereitung und Entfernung erfolgen kontrolliert in der Managerumgebung. Eine Flotte kurzlebiger Auth-Container ist keine Standardlösung.

Login-, Ablauf- und Refresh-Eigenschaften brauchen je Provider eigene Nachweise. Fehlende Fähigkeiten sind vor einer behaupteten Unterstützung erkennbar.

## 17. Container, Git-Arbeitsstände und dauerhafte Kandidaten

### 17.1 Lebensdauer pro Run

Jeder Manager hat pro Run einen eigenen Container, ein eigenes Gespräch, einen eigenen Harness und einen eigenen Git-Arbeitsstand. Innerhalb des Runs bleiben sie über Delegation, Fragen und Korrekturen bestehen; zwischen Runs werden sie nicht wiederverwendet. Auch der Git Workspace Service lebt genau einen Run. Spätere Runs brauchen keine vorab reservierten Volumes oder Arbeitskopien.

### 17.2 Git Workspace Service

Der Service bereitet Arbeitsstände vor, stellt Ausgangs-Commits bereit, fixiert Übergaben, unterstützt Eltern-Kind-Integration und die Aktualisierung von Kinderständen und meldet technische Git-Konflikte. Er ist kein Agent, kein allgemeiner Helper und kein Ort fachlicher Entscheidungen; Provider-Login, Projektchecks und Workflowsteuerung sind andere Verantwortungen.

### 17.3 Isolation und Zusammenarbeit

Managerkopien sind unabhängig. Eltern integrieren Ergebnisse nach Review in ihre eigene Kopie; Änderungen werden nicht über ein gemeinsames beschreibbares Verzeichnis propagiert. Isolation ermöglicht die vereinbarte Zusammenarbeit; sie ersetzt kein Ownership-Review.

### 17.4 Workspace und Kandidat sind verschiedene Zustände

Ein Workspace ist veränderlich und an die laufende Ausführung gebunden. Ein Kandidat ist ein fixierter Git-Stand mit dauerhaft verfügbaren Objekten. Modellidentität, Commitidentität und Bauminhalt sind verschiedene Größen und bleiben passend gebunden. Persistenz hängt nicht von laufenden Containern, temporären Volumes oder lebenden Objekten ab.

### 17.5 Speicherform

Selbstständige Git-Bundles mit gebundenen Metadaten sind der vorgesehene V1-Ansatz; eine gleichwertige einfachere Form wäre begründbar. Die Garantie ist die unabhängige Wiederöffnung, nicht das Format: Ein Persistenznachweis entfernt die temporären Arbeitsstände und öffnet exakt den Kandidaten in einer neuen Umgebung allein aus den gespeicherten Daten. Verify und Publication brauchen keine lebende Referenz auf den erzeugenden Run. Eine Git-Serverplattform ist nicht vorgesehen.

### 17.6 Checks und verbleibende Ressourcen

Eigene kurzlebige Check-Ausführungen sind zulässig, wenn sie nötige Isolation bieten; finale Checks vor Publication haben eigene, klar verantwortete Ressourcen und setzen keine Managercontainer fort. Ungewissheit über übrig gebliebene Ressourcen wird berichtet, ein unbekannter Restcontainer nicht pauschal gelöscht. Ein allgemeines Recovery-System nach hartem Hostabbruch ist nicht Teil von V1.

## 18. Verbindliche Ergebnisse und Übergaben zwischen Runs

### 18.1 Zwei getrennte Ergebnisarten

Verbindliche Run-Ergebnisse bestimmen die weitere Workflowausführung; Shared Notes bewahren optionale Informationen. Beide sind persistent, haben aber verschiedene Autorität.

Ein Run-Ergebnis enthält nur die nötigen gebundenen Angaben: Runidentität und Mandat, Modell- und Projektbindung, Eingabekandidat, technischen Abschluss, fachliches Urteil, Ergebniskandidat, verpflichtende Findings und notwendige technische Beobachtungen einschließlich tatsächlicher Cleanup-Probleme.

### 18.2 Kein lebender Zustand im Ergebnis

Sessions, aktive Managerbäume und temporäre Workspacehandles sind keine Voraussetzung, um ein abgeschlossenes Ergebnis zu lesen oder zu veröffentlichen. Ein neuer Run stellt seine Umgebung aus den dauerhaften Eingaben selbst her. Eine erfolgreiche Rückgabe setzt Speicherung und bestätigte Lesbarkeit voraus. Fehlgeschlagene oder unvollständige Runs dürfen diagnostizierbare Daten behalten, ohne einen erfolgreichen Kandidaten vorzutäuschen.

### 18.3 Findings als handlungsfähige Übergabe

Ein Finding lässt erkennen, welche Verpflichtung betroffen ist, was beobachtet wurde, was erwartet wird und welche Stelle die Korrektur ermöglicht. Eine kleine verständliche Struktur genügt. Die Herkunft bestimmt der Host aus dem tatsächlichen Manager beziehungsweise Prüfmechanismus. Die nächste Organisation erhält die nötigen Findings ausdrücklich, nicht versteckt in einem früheren Chatverlauf.

### 18.4 Konsistenz über Run-Grenzen

Ein neuer Run bearbeitet genau den referenzierten Kandidaten; Zustimmung und Kandidat passen zusammen. Eine neue Modellbasis macht frühere Bindungen nicht gültig. Kontinuität entsteht durch Modell, Kandidaten, Ergebnisse und Notes, nicht durch Wiedereinspielen alter Gespräche.

## 19. Shared Notes und unabhängige Verifikation

### 19.1 Shared Notes gehören zu V1

Manager halten Nebenbefunde, Verbesserungsvorschläge, Testlücken und hilfreiche Erkenntnisse dauerhaft fest. Notes überleben Runs und Commands innerhalb eines Projekts. Jede Note hat einen verständlichen Inhalt und nachvollziehbare Herkunft: Run, Manager, betroffener Bereich und geeignete Referenzen. Konkurrierende Schreiber überschreiben einander nicht.

### 19.2 Keine automatische Autorität

Eine Note ist keine Delegation, keine Modellpflicht und kein automatisch auszuführendes Ticket. Sie umgeht weder Hierarchie noch Review. Verpflichtende Probleme dürfen nicht in Notes verschoben werden, um eine Zustimmung zu ermöglichen; entscheidend ist ihre Bedeutung für das Mandat, nicht die gewählte Ablage.

### 19.3 Begrenzter Abruf (D17)

Notes werden nicht pauschal in jeden Startprompt kopiert, bleiben aber bei Bedarf lesbar, auch wenn ihre Menge über viele Commands wächst. `read_notes` liefert begrenzte Seiten in stabiler Reihenfolge; jede Seite passt in die Transportgrenze. Keine stille Kürzung, kein Datenverlust, kein Löschen alter Notes, keine Vektordatenbank oder semantische Suche. Sortierschlüssel, Seitengröße, Filter und der Umgang mit einzelnen sehr großen Notes werden nach einer kontrollierten Reproduktion des Größenkonflikts gewählt (Abschnitt 28.1).

### 19.4 Verify-Unabhängigkeit

Verify erhält das gültige Modell, den exakten Kandidaten und nötige technische Informationen. Es übernimmt weder Implementierungsgespräche noch deren Selbstbewertung. Notes liefern Hinweise mit Herkunft, aber keine vorab festgelegte Wahrheit über den Kandidaten. Verborgene Erwartungen unabhängiger externer Orakel erreichen Implementierer nicht. Agentenerzeugte Tests ersetzen nicht allein den unabhängigen fachlichen Qualitätsnachweis.

## 20. Fehler, Budgets und kontrolliertes Beenden

### 20.1 Fachliche und technische Ausgänge

| Ausgang | Bedeutung und Folge |
|---|---|
| **Fachlich angenommen** | Das Mandat wurde positiv beurteilt. Nur zusammen mit gültigem technischem Abschluss nutzbar. |
| **Realisierungs-Reject** | Die Realisierung verletzt Anforderungen; Findings ermöglichen eine Korrektur. |
| **Modellblocker** | Die gebundene Absicht ist materiell unzureichend oder widersprüchlich; kein endloser Reparaturversuch im Code. |
| **Failed** | Ein tatsächlicher Fehler verhindert die Ausführung. Kein fachlicher Reject. |
| **Incomplete** | Ein verlässlicher Abschluss ist nicht feststellbar, etwa bei Timeout, Abbruch oder ungewissem Cleanup. |

Die Namen der Statuswerte sind Implementierungsdetail, ihre Bedeutungen werden nicht vermischt. Eine positive Agentenbehauptung heilt keinen Transport- oder Speicherfehler.

### 20.2 Gemeinsame endliche Ressourcen

Der Command setzt ein endliches Gesamtbudget; neue Runs und Korrekturen schaffen kein neues Budget. Jeder Run verbraucht aus diesem gemeinsamen Budget, ohne die übergeordnete Komposition zu kennen. Per-Message-Limits begrenzen technische Aufrufe, ersetzen aber nicht den fachlichen Abschlussvertrag. Budgets werden nicht pauschal erhöht, um einen unverstandenen Deadlock oder Protokollfehler zu kaschieren.

### 20.3 Deadline bis zum blockierenden Vorgang

Abbruch und Fristen reichen bis zu den kontrollierten Unterprozessen, insbesondere in Git-, Guard- und Publication-Pfaden. Eine äußere Fristprüfung reicht nicht, wenn ein innerer Prozess unbeschränkt weiterläuft. Cleanup und abschließende Persistenz erhalten eine kleine gesonderte Nachlaufgrenze, um diagnostizierbare Ergebnisse zu sichern. Der Nachlauf erlaubt keine neue fachliche Arbeit und keine verdeckte Verlängerung der Auftragsressourcen.

### 20.4 Ehrliches Cleanup

Teilweise gestartete Ressourcen werden soweit möglich geschlossen. Ein unklarer Abschluss wird als unklar gespeichert; eine spätere leere Containerliste macht einen zuvor fehlgeschlagenen Closure-Nachweis nicht nachträglich erfolgreich. Gewöhnliche Fehlerpfade räumen verlässlich auf. Eine Garantie für beliebige Kernel-, Dateisystem- oder Hostausfälle wird nicht behauptet.

## 21. Kontrollierte Publication

### 21.1 Freigabe des exakten Kandidaten

Publication konsumiert einen dauerhaft gespeicherten Kandidaten und dessen gespeicherte, gebundene Verify-Zustimmung. Implement gibt sich nicht selbst frei; eine Zustimmung zu Kandidat A gilt nicht für Kandidat B. Vor der Übertragung werden Bindungen und erforderliche Checks erneut geprüft. Alle Run-Ressourcen sind dann geschlossen; der Publisher startet keine Managerkoordination.

### 21.2 Nutzeränderungen und Konflikte

Der Zielzustand muss zur gebundenen Ausgangslage passen. Zwischenzeitliche Nutzeränderungen werden nicht überschrieben; V1 endet bei relevanter Abweichung mit einem klaren Konflikt statt mit einem erfundenen semantischen Merge. Erzeugt eine spätere Funktion einen Merge, der vom geprüften Kandidaten abweicht, braucht sie eine eigene erneute Prüfung. Der letzte deterministische Check ersetzt keine unabhängige fachliche Zustimmung zu neuen Inhalten.

### 21.3 Teilübertragung und Readback

Eine Mehrdateiübertragung ist nicht automatisch atomar. Bei Fehler oder Abbruch bleibt erkennbar, welche Änderungen abgeschlossen sind; ausstehende erscheinen nicht als übertragen. Der finale Readback bestätigt die veröffentlichten Inhalte. Ein Fehler an dieser Grenze wird weder als Erfolg noch als unveränderter Zustand ausgegeben. Blindes Zurückrollen unter möglicher Nutzerdrift ist keine sichere Ersatzlösung.

### 21.4 Kein automatischer Nutzer-Commit

Publication erstellt keinen Commit, Push, Release oder Deployment im Nutzerprojekt. Private Git-Commits zur Fixierung von Kandidaten und Entwicklungs-PRs beim Bau von Markitect sind davon getrennt. Auch No-change durchläuft die Bindungs- und Zielprüfungen und veröffentlicht dann keine Änderungen.

## 22. Logische Zielarchitektur

### 22.1 Verantwortlichkeiten

| Verantwortlichkeit | Aufgabe | Gehört nicht dazu |
|---|---|---|
| **Model / Project State** | Modellkompilierung, Registry, Accounting, Snapshots, Organisation und Kontext. | Providerstart, Workflow oder Veröffentlichung. |
| **Projektvorbereitung** | Explizite Eingaben zu einer geprüften fixierten Ausführungsbasis zusammenführen. | Zweiter Compiler oder alte Runtime. |
| **Command-Anwendung** | Initialprüfung, Budget, Workflow und Gesamtergebnis. | Managerkommunikation oder Git-Mergelogik. |
| **Workflow** | Runs und ihre fachlichen Übergänge koordinieren. | Container, Providerprotokolle oder Ressourcenverwaltung. |
| **Run-Ausführung** | Ein Mandat durch die Organisation, mit Lebenszyklus, Ergebnis und der Validierung geschlossener Ergebnisse. | Nächsten Run wählen oder publizieren. |
| **Harness** | Regeln, Kontext und Arbeitsmittel zusammensetzen. | Git-Integration oder Workflowentscheidung. |
| **Workspace / Candidate Storage** | Private Kopien und unabhängige dauerhafte Kandidaten. | Reviewentscheidungen oder allgemeiner Helper. |
| **Run Information** | Verbindliche Ergebnisse und getrennte Notes speichern und bereitstellen. | Aufgaben aus Notes erzeugen. |
| **Publication** | Exakten akzeptierten Stand kontrolliert übertragen. | Fachliche Reparatur oder Agentenstart. |
| **Infrastrukturadapter** | Provider, Container, Git, Checks, Dateisystem. | Projektfachlichkeit. |

Ein logischer Verantwortungsbereich ist nicht automatisch ein Go-Package, Interface, Service oder Container. Der Schnitt folgt Kohäsion, echten Verbrauchern und begründeten Infrastrukturgrenzen.

### 22.2 Abhängigkeitsrichtung

```text
CLI / sonstiger Einstieg
    → Command-Anwendung
         → Projektvorbereitung und deterministisches Fundament
         → Workflow
              → Run-Vertrag und Run-Ausführung
                   → Modellorganisation und Harness
                   → Workspace-, Session-, Check- und Speicherverträge
         → Publication

Konkrete Infrastrukturadapter implementieren die benötigten Grenzen.
Die Zusammensetzung liegt am äußeren Anwendungsrand.
```

Der Compiler kennt weder Docker noch Codex oder Claude. Der Run kennt die Apply-Korrekturschleife nicht. Der Workflow implementiert keine Managerkoordination. Providerdetails bleiben aus der Organisation heraus.

### 22.3 Fachlichkeit, Anwendung und Infrastruktur

Domain, Application und Infrastructure sind eine logische Orientierung, keine Pflicht, jedes Feature in drei Dateigruppen zu zerlegen. Schmale Interfaces entstehen dort, wo eine Abhängigkeit tatsächlich getrennt werden muss. Kein universelles Repository-, Event-, Policy- oder Workflowframework nur um eines Musters willen; alte Package-Namen werden weder automatisch konserviert noch kosmetisch ersetzt.

## 23. Umgang mit Bestandscode

Modellsprache, Compilersemantik und relevante deterministische Garantien sind das bewahrenswerte Fundament. Die historische agentische Ausführungsarchitektur hat keinen Bestandsschutz. Für betroffene Komponenten gilt Keep, Refactor, Replace oder Remove, entschieden nach Verantwortung, Kopplung, Verständlichkeit, Testbarkeit und benötigtem Verhalten, nicht nach investierter Zeit.

Adapter nur für alte interne APIs, dauerhafte Kompatibilitätsschichten, konkurrierende Runtimepfade und Sonderfälle für ausgeschlossene Verbraucher sind keine akzeptable Standardstrategie. Leitfrage: **Würden wir diesen Baustein so entwerfen, wenn die frühere Runtime nicht existierte?**

Ein Rewrite ist kein Reflex. Der Korrekturbedarf wird an belegten Vertrags- und Zuverlässigkeitsproblemen bemessen. Eine große Datei ist ein Hinweis auf überladene Verantwortung, kein automatischer Defekt; unverstandene Zustandskomplexität auf mehr Dateien zu verteilen, verbessert nichts. Zuerst wird die Semantik klar.

Selbstmodell, Registry-Shards, generierte Dokumentation und tatsächliche Verantwortlichkeiten werden gemeinsam konsistent gehalten. Aktive Einstiegsdokumente enthalten geltende Regeln, keine Folge historischer Arbeitsaufträge. Historische Erkenntnisse und Fehlschläge bleiben archiviert, werden aber nicht durch Nähe zum aktuellen Plan erneut verbindlich.

## 24. Entwicklungsorganisation

### 24.1 Coordinator und Implementierungs-Chats

Ein Coordinator verantwortet Architektur, Abhängigkeiten, Aufträge, Integrationsreihenfolge und Qualitätskontrolle; er schreibt nicht konkurrierend in die Dateien anderer Chats. Implementierungs-Chats arbeiten in eigenen Worktrees und Branches mit exaktem Ausgangscommit, eindeutiger Schreibverantwortung und überprüfbaren Abnahmekriterien. Gemeinsame Verträge, Registry-Shards und generierte Dateien haben zu jedem Zeitpunkt genau einen Writer; Subagenten umgehen diese Grenze nicht.

### 24.2 Parallelität nach tatsächlicher Entkopplung

Höchstens zwei schreibende Workstreams gleichzeitig. Lesende Analyse und unabhängige Reviews können parallel laufen. Abhängige Implementierung startet erst, wenn die gemeinsamen Verträge klar sind; keine Interfacehierarchie ohne Verbraucher. Frühe vertikale Nachweise testen zentrale Annahmen, bevor viel darauf aufbaut.

### 24.3 Review und Integration

Wesentliche Änderungen brauchen einen unabhängigen vollständigen Diff-Review am tatsächlich integrierbaren Commit. Änderungen nach dem Review brauchen eine erneute Prüfung. Nach größeren Integrationsschritten wird die Gesamtarchitektur geprüft. Fremde Arbeit wird weder still verworfen noch per Reset überschrieben.

### 24.4 Befugnisse

Routineentscheidungen innerhalb freigegebener Grenzen sind delegierbar. Änderungen der Produktsemantik, erhebliche Architekturkonflikte und die Owner-Abnahme werden ausdrücklich behandelt. Für sinnvolle reale Nachweise besteht eine stehende Freigabe; jeder Nachweis nennt Quelle, Grenzen, Abbruch, Cleanup und tatsächliches Ergebnis. Release und Deployment bleiben getrennte Entscheidungen.

### 24.5 Coordinator-Wechsel

Lange Chats können nach kontrolliertem Handoff archiviert werden. Ein Handoff benennt Main, offene Arbeit, bestätigte Nachweise, Einschränkungen und nächste Aufgaben. Ein neuer Coordinator übernimmt den Repository- und Evidenzstand, nicht die alte Gesprächshistorie.

## 25. Abnahme

### 25.1 Arten von Evidenz

| Nachweisart | Belegt | Belegt nicht |
|---|---|---|
| Statische Code- und Architekturprüfung | Verantwortlichkeiten und erkennbare Vertragsverletzungen. | Laufzeitverhalten, Race-Freiheit, Providergehorsam. |
| Deterministische Tests | Verhalten der geprüften Fälle auf fixiertem Code. | Native Integration oder Fehlerfreiheit. |
| Kontrollierte Sessions mit echtem Git | Zusammenspiel von Executor, Kandidaten, Reviews und Fehlerpfaden. | Dass ein echter Provider denselben Ablauf zuverlässig bedient. |
| Native Komponentenprüfung | Eine konkrete Providerfähigkeit oder Run-Kette. | Öffentliche Workflowauswahl und Publication. |
| Öffentlicher nativer Apply | Den Produktpfad auf benanntem Modell, Kandidaten und Runtimeprofil. | Universelle Zuverlässigkeit, andere Provider, beliebige Projektgrößen. |
| Unabhängiges Orakel | Verhalten des erzeugten Kandidaten gegen unabhängige Erwartungen. | Nicht getestete Eigenschaften oder Migrationen. |
| Owner-Abnahme | Bewusste Akzeptanz eines klar begrenzten Produktumfangs. | Release- oder Deploymentfreigabe. |

### 25.2 Abnahmekriterien

| ID | Erforderlicher Nachweis |
|---|---|
| **V01** | Ungültige Modell- oder Initialstruktur verhindert den agentischen Workflow. |
| **V02** | Die vollständige Hierarchie arbeitet in Assessment, Implement und Verify, einschließlich Enkel, intermediärem und leerem Package. |
| **V03** | Parallele Kinderarbeit; Fragen nur entlang erlaubter Pfade, Geschwister- und Ebenensprünge werden mit Begründung abgelehnt; überkreuzte Fragen und Abbruch funktionieren. |
| **V04** | Review und Integration betreffen exakt die tatsächliche Aufgabenbasis und fixierte Übergabe. |
| **V05** | Ein ungültiger Implement-Kinderkandidat erreicht den Vater mit Diagnose und ist nicht integrierbar; Root kann ungelöste technische Ungültigkeit nicht als Erfolg abschließen. |
| **V06** | Verify-Reject erzeugt neue Implement- und Verify-Runs ohne alte Gespräche und ohne zurückgesetztes Gesamtbudget. |
| **V07** | Spät erkannte Modellblocker beenden ohne Publication und ohne endlose Codereparatur. |
| **V08** | Kandidaten lassen sich nach Entfernen der temporären Umgebungen exakt wiederöffnen. |
| **V09** | Standalone-Runs liefern persistierte tatsächliche Endergebnisse; Speicher- und Cleanupfehler bleiben sichtbar. |
| **V10** | Reproduzierbarer Harness wird nativ verwendet: kurze Aufgaben, Pflichtkontext, normale Kontoauthentifizierung, keine API-Key-Variante. |
| **V11** | Notes überleben Runs und Commands, verlieren bei Parallelzugriff keine Einträge, erweitern den Auftrag nicht und sind an der Transportgrenze seitenweise vollständig lesbar. |
| **V12** | Fresh Library, M1→M2 mit Grenzwert 5→3, Preservation, striktes No-change und unabhängiges Orakel. |
| **V13** | Ein eingebauter Defekt wird erkannt und nach Korrektur erneut unabhängig geprüft; normaler Implement-Fix und echter Verify-Reject-Zweig bleiben unterscheidbar. |
| **V14** | Eigene Änderungen, fremde Zuständigkeit und akzeptierte Kinderintegration werden sachgerecht reviewed. |
| **V15** | Publication bindet gespeicherte Zustimmung und Kandidat, behandelt Drift und meldet Teiltransfer und Readback-Fehler korrekt. |
| **V16** | Timeout, Providerfehler, Budgetende, Startfehler und unsicheres Cleanup erzeugen keine Scheinerfolge. |
| **V17** | Die Gegenbeispiele aus Abschnitt 13.3 sind als kontrollierte Tests abgedeckt; Harness und Host-Nachrichten erklären keine internen Host-Zustände (Abschnitt 3.7). |

### 25.3 Endstand und Wiederholbarkeit

Vollständige lokale Tests, Architektur- und Dependency-Gates, Linux- und Windows-Kompatibilität, Race-Prüfungen und freigegebene Hosted CI werden am bereinigten Endstand belegt. Native Nachweise nennen Provider, Version, Runtime, Kandidat und Grenzen. Die Zahl unabhängiger Wiederholungen wird begründet festgelegt; Wiederholen bis zum ersten Erfolg ersetzt keine Zuverlässigkeitsbewertung. Frühere Erfolge qualifizieren geänderte Verträge nicht ohne Prüfung ihrer Auswirkung.

### 25.4 Qualität ist mehr als ein grünes Beispiel

Eine Library-Demonstration prüft die Idee konkret, ersetzt aber nicht die Produktdefinition. Die Abnahme beurteilt zusätzlich Verständlichkeit, Kohäsion, Zustandskomplexität, Erweiterbarkeit im Zielbild und die ehrliche Supportgrenze. Nicht qualifizierte Fähigkeiten – weitere Provider, Login-Erneuerung, harte Hostabbrüche, Live-Datenmigration – werden durch die Bezeichnung V1 nicht implizit versprochen.

## 26. Referenzszenarien

### 26.1 Frische Realisierung

Das gültige Library-Modell enthält Verpflichtungen, Verantwortlichkeiten und Checks, aber noch keine Realisierung. Die Initialprüfung stellt strukturelle Gültigkeit fest, Assessment die Umsetzbarkeit. Implement erzeugt und registriert die Artefakte, Verify prüft unabhängig, danach erfolgt Publication. Das externe Orakel bewertet den Kandidaten außerhalb des Managerkontexts.

### 26.2 Änderung von fünf auf drei Ausleihen

Ausgangspunkt ist eine gültige Realisierung A1. Im Modell M2 ändert sich nur das Ausleihlimit von fünf auf drei. Markitect erzeugt eine angepasste Realisierung A2 und bewahrt die übrigen Verpflichtungen. Die Zahl geänderter Dateien ist nicht fest vorgegeben; jede Änderung ist durch Modell, Checks oder notwendige Folgen begründet. Eine Migration bestehender Laufzeitdaten wäre eine eigene Anforderung.

### 26.3 No-change

Ein unverändertes Modell trifft auf eine konforme Realisierung. Alle Mandate und Reviews laufen vollständig, optionale Ideen landen in Notes, und das Ergebnis enthält keine unbegründete Artefaktänderung.

### 26.4 Fehlerhafte Kinderübergabe

Ein Kind liefert ein Artefakt ohne gültigen Registry-Eintrag oder mit einem Schnittstellenproblem. Die Übergabe erreicht den Vater mit der tatsächlichen Diagnose. Er erkennt, ob lokale Registry-Pflege, eine gemeinsame Entscheidung oder eine andere Integrationsreihenfolge nötig ist, und weist mit Findings zurück oder klärt die Schnittstelle. Das Kind repariert lokal. Nichts wird veröffentlicht, weil ein Agent „fertig“ schreibt, und kein globaler Reparatur-Workflow startet.

### 26.5 Echte Verify-Korrekturschleife

Verify findet im Implement-Kandidaten einen Realisierungsfehler und liefert Reject mit Findings. Der öffentliche Workflow startet einen neuen Implement-Run mit frischen Gesprächen und danach ein neues Verify; erst dessen Zustimmung trägt die Publication. Eine Reparatur im ersten Implement ist ein anderes Szenario, und eine manuell verkettete Komponentenprüfung beweist nicht, dass der öffentliche Workflow den richtigen Zweig wählt.

### 26.6 Nutzeränderung während Apply

Der Nutzer ändert das Zielrepository, während die Manager isoliert arbeiten. Die Guards erkennen die Abweichung; V1 endet mit einem erkennbaren Konflikt statt blind zu überschreiben.

### 26.7 Abbruch während Publication

Ein Git-Aufruf blockiert, während die Frist abläuft. Die Frist erreicht den Unterprozess, keine weitere fachliche Arbeit startet, abgeschlossene Pfade werden korrekt berichtet und unvollständige Readbacks nicht als Erfolg gewertet. Der Test erreicht den blockierenden Aufruf tatsächlich.

## 27. Entscheidungen

### 27.1 Grundentscheidungen D1–D9

| ID | Entscheidung | Bedeutung |
|---|---|---|
| **D1** | Universeller Run-Executor | Ein eigenständig verwendbarer Mechanismus für die vollständige Organisation und alle V1-Mandate. |
| **D2** | Workflow als Run-Komposition | Schlanke Koordination von Runs; spätere Custom Commands möglich, keine DSL-Pflicht. |
| **D3** | Container und Workspace Service je Run | Frische Managerumgebungen und ein Git-Service mit derselben Run-Lebensdauer. |
| **D4** | Vier deterministische Kontrollpunkte | Initialprüfung, Kinderübergabe, Root-Abschluss und Publication. |
| **D5** | Deterministischer Harness aus fünf Quellen | Produkt, Projekt, Manager, Run und Provider bleiben getrennte Quellen einer nutzbaren Umgebung. |
| **D6** | Verbindliche Ergebnisse und Shared Notes | Explizite Übergaben zwischen Runs und dauerhafte nichtkanonische Beobachtungen in V1. |
| **D7** | Architekturqualität vor Wiederverwendung | Fachliche und technische Qualität zählen zur Abnahme, nicht nur funktionale Tests. |
| **D8** | Selektiver Clean-Slate-Rebuild | Modellfundament bewahren, ungeeignete Runtime ersetzen; kein Repository-Reset und kein fortlaufender Rewrite-Zwang. |
| **D9** | Koordinierte Multi-Chat-Entwicklung | Coordinator, getrennte Worktrees, klare Writer, unabhängige Reviews und kontrollierte Integration. |

### 27.2 Entscheidungen vom 24. Oktober 2026 (D10–D18)

Sie schließen die in v0.18 offenen Punkte A1–A6 und R1–R3. R1 (Deadline in der Publication) war bereits umgesetzt.

| ID | Ursprung | Entscheidung | Abschnitt |
|---|---|---|---|
| **D10** | A1 | Fragen nur an direkten Vater und direkte Kinder; Geschwister stimmen sich über den gemeinsamen Vater ab, mehrere Ebenen Schritt für Schritt über die Manager dazwischen. | 8.2 |
| **D11** | A2 | Jeder Implement-Kinderkandidat geht mit Prüfergebnis an den Vater; ungültige Kandidaten sind nicht integrierbar; optionale Selbstprüfung per Werkzeug. | 8.4 |
| **D12** | A3, R3 | Ein Urteil gehört zu Auftrag und bewertetem Stand, nicht zur Nachricht; expliziter Auftragszustand im Host. | 13.2 |
| **D13** | Neu | Aufgaben statt Prozess; fester kleiner Werkzeugsatz mit genau einem Werkzeug je Zweck. | 3.7, 14 |
| **D14** | A4 | V1-Harness über native Base-/Developer-Instruktionen; Skills und Nachladen erst bei belegtem Bedarf. | 15.5 |
| **D15** | A5 | Validierung geschlossener Run-Ergebnisse gehört zum Run-Vertrag. | 10.3 |
| **D16** | A6 | Genau drei Mandate in V1; neue Mandate erst mit einem echten Command. | 10.5 |
| **D17** | R2 | Notes seitenweise und stabil sortiert, jede Seite innerhalb der Transportgrenze, ohne stille Kürzung. | 19.3 |
| **D18** | Neu | V1-Abnahme mit Codex; Claude als nächste Providerabbildung auf dem stabilisierten Vertrag; kein nicht funktionsfähiger Claude-Pfad bis dahin. | 2.3 |

### 27.3 Präzisierungen gegenüber frühen Fassungen

| Früher diskutiert | Maßgeblich |
|---|---|
| Einzelagent für Assessment oder Verify | Vollständige Modellhierarchie in jedem Mandat. |
| Container je Apply oder langfristig weiterlaufend | Managercontainer und Git Workspace Service pro Run. |
| Allgemeiner Helper für Git, Auth und Checks | Getrennte klare Verantwortungen; keine Kleinstcontainer je Operation. |
| Extract-/UI-Erhaltung als Migrationspflicht | Ausgeschlossen; kein Einfluss auf Kernverträge. |
| Wiederverwendung der Assignment-Runtime | Eignung entscheidet; kein Bestandsschutz. |
| Feste Zahl neuer Packages | Keine künstliche Ober- oder Untergrenze. |
| Pauschales Ignorieren von `.markitect` | Kanonische und kontrollierte Eingaben bleiben sichtbar; operative Daten werden gesondert behandelt. |
| Harness zwingend als bestimmte Dateien | Native Instruktionsparameter sind die V1-Form (D14). |
| Fragen an jeden Manager im Run | Nur entlang der Hierarchie (D10). |
| Neues Urteil nach jeder aktiven Nachricht | Urteil gilt, bis Auftrag oder bewerteter Stand sich ändert (D12). |
| Neuer Gesamtumbau bei jedem Auditbefund | Bestehenden Kern gezielt verbessern. |

## 28. Technisch auszugestalten und Nachweisgrenzen

### 28.1 Technisch auszugestalten

| Thema | Festes Ziel | Offen für die Umsetzung |
|---|---|---|
| Auftragszustand (D12) | Jede Invalidierung folgt aus einem benennbaren Ereignis; keine verstreuten Flags. | Konkrete Zustände und Übergänge, Form der Host-Diagnose bei fehlendem Urteil, Weitergabe später Befunde an die Väter. |
| `check`-Werkzeug (D11) | Optionale Selbstprüfung mit demselben deterministischen Mechanismus wie die Übergabeprüfung. | Umfang (strukturell, projektbezogene Checks), Ausführungsort und Kosten. |
| Notes-Abruf (D17) | Vollständig lesbar, begrenzt, stabil geordnet. | Erst den Größenkonflikt kontrolliert reproduzieren, dann die kleinste Leseschnittstelle wählen: Sortierschlüssel, Seitengröße, Cursorform, Kontextfilter, Umgang mit sehr großen einzelnen Notes und mit Speicherfehlern. |
| Ergebnisvalidierung (D15) | Eine Validierung im Run-Vertrag. | Paketzuschnitt und Aufrufstellen. |
| Generischer AGENTS.md-Harness | Best-Effort-Instruktionsdarstellung für unbekannte Provider, kein Adapter. | Zeitpunkt und Zugehörigkeit zur V1-Abnahme, festgehalten mit der Provider-Matrix. |
| Selbstmodell | Beschreibt die gewählte Architektur. | Package-Zuschnitt für Projektvorbereitung, Workflow und Harness. |

### 28.2 Nachweisgrenzen und offene Abnahmeentscheidungen

Vor der Owner-Abnahme werden Providerumfang, Laufzeitgrenzen, Wiederholungsnachweise und bekannte Einschränkungen in einer Provider-Matrix ausdrücklich festgehalten. Nachweise auf älteren Verträgen qualifizieren die nach D10–D17 geänderten Verträge nicht. Der öffentliche native Verify-Reject-Korrekturzweig mit anschließender Publication ist gesondert nachzuweisen. Ein sauberer Endstand kann begrenzt abgenommen werden, ohne universelle Zuverlässigkeit oder nicht implementierte Fähigkeiten zu versprechen.

## 29. Quellen und Versionsführung

| Quelle | Rolle |
|---|---|
| [Owner-Konzept v0.18](sources/v1-concept-v018-20261024.md), 24. Oktober 2026 | Wortlaut der konsolidierten Owner-Fassung mit D1–D9 und ihren Quellen Q1–Q4. |
| [Entscheidungsprotokoll](sources/v1-concept-decisions-20261024.md), 24. Oktober 2026 | Owner-Annahme der Entscheidungsvorschläge D10–D18. |
| [Konzepttreue-Prüfung](../development/concept-conformance-audit-20261024.md), 24. Oktober 2026 | Statische Prüfung an Commit `6c7d758c`; Ursprung von A1–A6 und R1–R3. Kein Ersatzkonzept. |
| [V1 scope clarification](sources/v1-scope-clarification-20261023.md), [source implementation approval](sources/v1-source-implementation-approval-20261023.md) | Frühere Owner-Grundlagen, durch v0.18 und diese Fassung konsolidiert. |

Änderungen an diesem Dokument zeigen, ob sie das Soll ändern, eine technische Variante wählen oder nur Text verbessern. Eine Änderung des Solls braucht eine Owner-Entscheidung mit Register-Eintrag. Umsetzungsstand und Evidenz gehören in den Delivery Plan und in Validierungsberichte, nicht hierher.

---

**Leitgedanke:** Das Modell bestimmt die Organisation. Die Manager tragen die fachliche Verantwortung. Der Host hält die technischen Garantien ein. Qualität entsteht aus dieser klaren Trennung – nicht aus möglichst vielen Agenten, Regeln, Interfaces oder Freigabeschritten.

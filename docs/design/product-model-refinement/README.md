# Produktmodell: Gespräch und Begriffsbildung

Stand: 10. Oktober 2026. Status: laufende Produkt- und Architekturdiskussion.

Dieses Dokument hält die Aussagen des Nutzers, vorläufige Ableitungen und offene Fragen fest. Es definiert weder ein neues Modellschema noch einen Implementierungsauftrag. Bestehende Entscheidungen werden hier nicht aufgehoben. Verbindliche Produktbeschreibung, Architektur und Planung bleiben in [Vision](../../vision.md), [Konzeptregister](../../concepts/register.md), [Architektur](../../architecture.md) und [Roadmap](../../implementation-plan.md).

## Herkunft und Arbeitsstand

- Quelle: Markitect-Produktgespräch, Chat `01a121f1-b948-7050-ae5d-9921b99db9c0`.
- Eigener Worktree: `C:\Users\Consiliari\.codex\worktrees\product-model-concepts\Markitect`.
- Branch: `docs/product-model-concepts`.
- Ausgangspunkt: frisch abgerufenes `origin/main`, Commit `ecc69b0ad99236a5ddd02b85bf9ba656a0000f15`.
- Lokaler Zustand beim Anlegen: sauber. Änderungen dieses Gesprächs betreffen dieses Dokument und den Link in `docs/design/README.md`; der jeweilige Commit dokumentiert die Fortschreibung.
- Einordnung: Konzeptarbeit, keine neue Prüfung des gesamten implementierten Produktstands. Keine Tests oder Studien für dieses Dokument gestartet.

Frühere Codebetrachtungen im Gespräch dürfen nicht ohne erneute Prüfung als Aussagen über diesen Ausgangspunkt übernommen werden. Wir beginnen bewusst mit dem Produktgedanken, orientieren uns an bestehenden Konzepten und vergleichen einen ausgearbeiteten Entwurf später mit einem festgehaltenen Implementierungsstand.

## Vom Nutzer formulierter Ausgangspunkt

Wörtliche Aussage aus dem Gespräch:

> Die Idee für die erste Version von Markitect arbeitet über ein **kanonisches Modell**, das typisiert und relativ explizit definiert, wie das Projekt aussehen soll, wie es funktioniert, welche Konzepte und so weiter es gibt... Die Idee von Markitect ist es ausschließlich das Modell zu pflegen und dann einen Mechanismus zu haben, der die starke Typisierung und Struktur des Modells nutzt um AI Agents systematisch zu koordinieren um dieses Modell in Form eines Repositories und Dateien innerhalb des Repositories zu realisieren.
>
> Hier haben wir folgende Begriffe: Das Modell, Das Repository bzw die Dateien darin die es realisieren, Das Apply das Modell in Repo überführt. Das sind meiner Meinung nach die großen Hauptblöcke

Der Nutzer bestätigte diese Beschreibung als knappe Darstellung der Idee und als Ausgangspunkt der Grundarchitektur. Daraus ergeben sich auf dieser Abstraktionsebene drei Hauptblöcke:

| Hauptblock | Bedeutung in dieser Diskussion |
| --- | --- |
| Kanonisches Modell | Typisierte, explizite Beschreibung des beabsichtigten Projekts und seiner Zusammenhänge. |
| Repository | Dateien, die diese Beschreibung realisieren. |
| Apply | Mechanismus, der das Modell mithilfe systematisch koordinierter Agenten im Repository realisiert. |

`Apply` bezeichnet hier zunächst den gesamten abstrakten Realisierungsmechanismus. Diese Verwendung ist keine Entscheidung über den Namen oder Umfang eines bestehenden CLI-Kommandos. Prüfungen, Planung, Koordination und Umsetzung sind noch nicht als interne Komponenten dieses Blocks festgelegt.

Der vorhandene Repositoryzustand ist bei Änderungen ebenfalls ein Eingang des Mechanismus. Das Modell beschreibt den Sollzustand; die tatsächlich vorhandenen Dateien müssen dafür untersucht und angepasst werden können.

## Warum wir bei Repositoryartefakten beginnen

Der Nutzer möchte zuerst sammeln, was in einem Projekt üblicherweise vorhanden ist und warum. Daraus sollen die Aspekte hervorgehen, die das Modell darstellen können muss. Seine ursprüngliche Sammlung lautet:

- Architektur
- Module
- Source Code
- Dokumentation
- Pipelines
- Infrastruktur
- Skills, Workflows, etc
- Prozesse und Richtlinien wie im Projekt gearbeitet wird
- Verantwortlichkeiten
- Tests
- Dashboard Definitionen wie bei Grafana z.B.
- Konfigurationen wie Env Files
- Git
- Work Items
- ...

Die Sammlung ist ausdrücklich offen. Ihre Einträge liegen auf unterschiedlichen Ebenen: Architektur ist ein beschriebener Zusammenhang, Source Code eine Realisierungsform, Git ein Werkzeug mit Geschichte und Konventionen, ein Work Item ein Anlass für Veränderung. Diese Unterschiede sollen helfen, die Begriffe zu ordnen; sie sind kein Grund, einen der Einträge vorzeitig auszuschließen.

## Modell als vollständige Dokumentation oder Spezifikation

Wörtliche aktuelle Aussage des Nutzers:

> Wenn das Kanonische Modell die Basis für alles ist und das Repository das ist was daraus entstehen soll müssen wir erstmal schauen was in einem Repository für ein Projekt üblicherweise vorzufinden ist und wieso. Ausserdem fällt dir vielleicht auch auf, dass all diese Dinge im Prinzip auch Themen sind die von einer guten Dokumentation aufgefasst werden sollten. Wir müssen hier eine gute Aufteilung finden was das kanonische Modell darstellen können muss. Im Prinzip lässt sich das kanonische Modell allerdings als eine vollständige Dokumentation oder Spezifikation beschreiben die die Grundlage für alles ist

Leitgedanke des Nutzers: Das Modell soll die maßgebliche Projektbeschreibung bilden, aus der die Realisierung hervorgeht. Es soll nicht zu einem weiteren Gegenstand werden, der parallel zu unabhängigen Spezifikationen manuell synchron gehalten werden muss.

Vorläufige Ableitung des Assistenten: Gute Projektdokumentation und das kanonische Modell behandeln weitgehend dieselben inhaltlichen Fragen. Das Modell ergänzt eine explizite Struktur, Typisierung und Beziehungen, mit denen diese Inhalte geprüft und für die Agentenkoordination verwendet werden können. Lesbare Dokumentation könnte daraus als Ansicht entstehen. Ihre genaue Form und ihre Beziehung zum Modell sind noch zu klären.

Die Bedeutung von „vollständig“ ist offen: Welche Entscheidungen müssen explizit enthalten sein, welche Details dürfen bei der Realisierung entschieden werden, und welche Inhalte werden referenziert? Es gibt noch keine Entscheidung, dass das Modell den Quellcode zeilenweise dupliziert oder sämtliche Laufzeitbeobachtungen selbst enthält.

## Vorläufige Beschreibungsperspektiven

Die folgenden sechs Perspektiven wurden vom Assistenten aus der Sammlung vorgeschlagen. Der Nutzer bestätigte das Vorgehen, aus Artefakten Aspekte abzuleiten. Die konkrete Aufteilung ist noch kein beschlossenes Typsystem und keine feste Liste von Modellarten.

| Perspektive | Was ein Projekt darüber beschreiben müsste | Beispiele aus der Sammlung |
| --- | --- | --- |
| Zweck und Fachlichkeit | Ziele, Nutzerbedürfnisse, Begriffe, Anwendungsfälle und fachliche Regeln. | Anforderungen und fachlicher Inhalt von Work Items, Dokumentation, Code und Tests. |
| Aufbau und Verhalten | Bausteine, Schnittstellen, Daten, Abhängigkeiten und das Zusammenspiel zur Laufzeit. | Architektur, Module, Source Code. |
| Arbeitsweise und Verantwortung | Wie gearbeitet wird, wer wofür zuständig ist und welche Regeln gelten. | Prozesse, Richtlinien, Skills, Workflows, Verantwortlichkeiten. |
| Bereitstellung und Betrieb | Wie das System gebaut, bereitgestellt, konfiguriert und beobachtet wird. | Pipelines, Infrastruktur, Konfiguration, Dashboarddefinitionen. |
| Qualität und Prüfung | Erwartete Eigenschaften und die Verfahren, mit denen ihre Erfüllung geprüft wird. | Tests, Qualitätsregeln, Prüfverfahren und Abnahmekriterien. |
| Entwicklung und Veränderung | Wie Änderungen beschrieben, entschieden, versioniert und ausgeliefert werden. | Work Items, Git-Konventionen, Versionen und Releases. |

Diese Perspektiven dürfen sich überschneiden. Eine Datei bekommt dadurch nicht genau eine Kategorie. Beispielsweise kann eine Pipeline sowohl Bereitstellung als auch Qualitätsprüfung realisieren. Sicherheit kann Fachlichkeit, Architektur, Arbeitsweise und Betrieb betreffen.

Wir trennen vorläufig zwei Fragen:

1. **Welchen Aspekt des Projekts beschreibt eine Aussage?** Zum Beispiel eine fachliche Regel, eine Schnittstelle oder eine Zuständigkeit.
2. **In welcher Form wird dieser Aspekt realisiert oder vermittelt?** Zum Beispiel Code, Test, Dokumentation, Konfiguration oder Pipeline.

Ein Aspekt kann mehrere Dateien und mehrere Realisierungsformen betreffen. Eine Datei kann mehrere Aspekte ausdrücken. Diese Beziehungen sind für die Frage „Was gehört wozu?“ wichtiger als eine ausschließliche Einteilung nach Dateitypen.

## Sollbeschreibung, Realisierung und Beobachtung

Vorläufige begriffliche Trennung des Assistenten:

| Rolle | Beispiel | Noch zu klärende Beziehung zum Modell |
| --- | --- | --- |
| Beabsichtigte Eigenschaft | Eine fachliche Regel oder eine erwartete Verfügbarkeit. | Explizite Aussage im kanonischen Modell. |
| Realisierung | Code, Konfiguration und Betriebsanleitung, die eine Eigenschaft umsetzen. | Zuordnung zu den Aussagen, die sie realisieren. |
| Prüfung und Beobachtung | Testergebnis, Messung, Laufzeitprotokoll oder Prüfbefund. | Beleg über einen konkreten Stand; kein automatischer Ersatz der Sollbeschreibung. |
| Änderungsanlass | Featurewunsch, Fehlerbericht oder technisches Work Item. | Vorgeschlagener Übergang des Modells; nicht automatisch bereits geltender Sollzustand. |

Das sind Rollen von Informationen, keine festgelegten Module oder Speicherorte. Insbesondere können Testdefinitionen zur geplanten Prüfung gehören, während Testergebnisse Beobachtungen sind. Git kann Konventionen und Versionsstruktur ausdrücken, enthält aber auch die tatsächliche Änderungsgeschichte. Diese Unterscheidungen sollen im weiteren Gespräch präzisiert werden.

## Offene Produktfragen

- Welche Projektaspekte fehlen in der Sammlung, und welche der sechs Perspektiven sollten zusammengelegt oder anders geschnitten werden?
- Was bedeutet eine vollständige Spezifikation für Projekte unterschiedlicher Größe und Art?
- Was muss explizit modelliert werden, was darf referenziert oder abgeleitet werden, und welche Entscheidungen bleiben bei der Umsetzung?
- Welche Beziehungen braucht das Modell zwischen Regeln, Bausteinen, Verantwortlichkeiten und ihren Realisierungen?
- Wie werden Dokumentation und andere lesbare Ansichten gepflegt, ohne mehrere unabhängige Wahrheiten zu schaffen?
- Wie werden Änderungswünsche, Entscheidungen, Historie und Beobachtungen angebunden, ohne sie mit dem geltenden Sollzustand zu vermischen?
- Welche Ein- und Ausgänge und welche Zusagen braucht der abstrakte Apply-Mechanismus? Seine interne Architektur folgt später.

## Fortsetzung

Als Nächstes nehmen wir konkrete Artefakte aus der Sammlung, beschreiben ihren Zweck und leiten daraus ab, welche Informationen und Beziehungen das Modell ausdrücken können muss. Kategorien und Begriffe bleiben dabei überprüfbare Vorschläge. Erst nach dieser Klärung vergleichen wir den Entwurf mit der vorhandenen Markitect-Architektur und entscheiden über mögliche Änderungen in deren jeweiligen kanonischen Quellen.

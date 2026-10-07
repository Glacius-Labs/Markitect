# Öffentliche Zuordnung des erweiterten Forschungsauftrags

Stand 2026-10-07; **Vorab-Zuordnung und Coverage-Lücken, keine Studienergebnisse**.
Maßgeblich ist der [gebundene Studienvertrag](study-contract-2026-10-07.md) aus
Commit `d929aa7c6f4b590bac2251333a62987bf04ec0da`; Herkunft, Blob und Hash stehen
in [study-contract-source.json](study-contract-source.json). Der kanonische Owner
bleibt `docs/design/government/evaluation.md` am referenzierten Stand. Relative
Links im bytegleichen Snapshot beziehen sich auf dessen ursprüngliches Verzeichnis.

Diese Matrix ist ein gemeinsames öffentliches Bewertungs-/Berichtssupplement.
Sie ändert weder Taskkarten noch Fixture-Code, Gewichte oder fachliche Regeln.
Eine ausdrücklich als Vorschlag bezeichnete Ergänzung gehört noch nicht zu den
Actor-Eingaben oder zum Scoring. Vor einer Übernahme wird sie allen Armen gleich
mitgeteilt und in einem neuen Protokollstand eingefroren. Öffentlich bedeutet
hier nicht, dass alle zukünftigen Taskkarten vorzeitig in Actor-Pakete gelangen:
Brief, Karten, Checks und Anforderungen bleiben nach dem gleichen Task-Cutoff
freigegeben. Private Holdout-Werte und Ergebnisse bleiben getrennt.

## Sechs Fragen und vorhandene Nachweise

T1–T6 bezeichnen die bestehenden Karten [Anlage](tasks/01-create-order.md),
[Reservierung](tasks/02-reserve-stock.md), [Persistenz](tasks/03-persist-state.md),
[Lebenszyklus](tasks/04-order-lifecycle.md), [Regeländerung](tasks/05-change-limit.md)
und [Regressionsreparatur](tasks/06-repair-regression.md).
Die angegebenen Rubrik-IDs stammen aus [rubric.json](rubric.json); Messereignisse
aus [metrics.schema.json](metrics.schema.json). Kandidat, Inputs, Task-Cutoff und
tatsächlicher Nachweis werden für jede Feststellung gebunden. Eine fehlende
Beobachtung ist unbekannt oder ungetestet, kein angenommener Erfolg.

| Nutzerfrage | Vorhandene Aufgabe und Messung | Gleiche unabhängige Kriterien | Coverage und konkrete Grenze |
|---|---|---|---|
| **Q1: Kann ich mein gedankliches Modell ausdrücken?** | Gemeinsamer Brief, Architektur-Freiheitsgrade und T1-Regeln; `setup`, `modeling`, `intervention`, Einrichtungs-/Aufsichtszeit. | Nur bereits freigegebene Begriffe, Invarianten, Prioritäten und erlaubte Freiheit vergleichen; ausgelassene Regeln, unbegründete Annahmen, erkannte Unsicherheit und erforderliche Rückfragen einzeln belegen. Conventional darf gute normale Dokumente verwenden. Markitect-Modellzuordnung ist zusätzliche Methodenfähigkeit. | **Teilweise:** vorgegebene synthetische Absicht ist vorhanden. Keine gemeinsame Pflicht zur expliziten Absichtswiedergabe, keine echte Prioritäten-Elicitation oder menschliche Modellannahme; kein Nachweis, dass ein realer Nutzer sein eigenes Modell bequem ausdrücken kann. |
| **Q2: Verstehen die Agenten die Absicht?** | T1 Validierung/Idempotenz; T2 Atomizität/Konkurrenz; T3 Neustart; T4 Übergänge; T5 neue Regel; T6 Diagnose. Öffentliche Checks und unabhängiger Holdout nach Freeze. | `functional_behavior`, `state_and_data_integrity`, `verification_quality`: tatsächliches Verhalten an freigegebenen Regeln prüfen; falsche Interpretation, unbefugte Vereinfachung und offen erklärte Unsicherheit dokumentieren. Eigene grüne Tests oder prosebasierte Selbstauskunft genügen nicht. | **Direkt, eng:** semantische Treue an konkreten Bestellregeln. Keine offene Ideenumsetzung oder absichtlich echte Mehrdeutigkeit; daraus kein allgemeiner Verständnisnachweis. |
| **Q3: Bleiben Modell und Repository konsistent?** | T5 verlangt dieselbe Änderung an allen aktiven Eingängen, Checks und betroffener Dokumentation. T6 verlangt Reparatur gegen den akzeptierten Stand. `model_maintenance`, `verification`, `assessment`, Zeit-/Kandidatenfolge. | `regression_and_compatibility`, `verification_quality`; veraltete Regeln, verpasste Drift, Fehlalarme und Zeit bis zur belegten Ausrichtung erfassen. Verhaltenspass getrennt von Modell-/Dateizuordnung und Nachweisfrische. Modelländerung oder Beobachtung darf nicht still akzeptierte Absicht ersetzen. | **Direkt für lokale Regelpropagation/Reparatur, teilweise für Modellpflege:** T6 kündigt eine Regression an und misst daher keine spontane unaufgeforderte Driftentdeckung. Delegierte Modell-Evolution mit echten Entscheidungsrechten wird durch diese Karten nicht vollständig geprüft. |
| **Q4: Bleibt das Projekt über Änderungen wartbar?** | T1–T4 wachsende Zustands-/Integrationsaufgaben; T5 zweite Eingangsstelle; T6 Reparatur ohne Pfadangabe. `implementation`, `integration`, `retry`, `recovery`, `assessment`; betroffene Stellen und Nacharbeit. | `maintainability_and_operability`, `regression_and_compatibility`: nachvollziehbare Zuständigkeit, sachlich notwendige Änderungspropagation, Wiederholbarkeit und erhaltenes Nachbarverhalten begründen. Unerwartete Kopplung und Such-/Anpassungsaufwand an konkreten Änderungen belegen. Nur ausdrücklich vereinbarte Grenzen sind bindend. | **Lokaler Hinweis:** kleiner endlicher Dienst, keine vorgeschriebene Modulstruktur und kein Langzeitverlauf. Kein Nachweis dauerhafter Vermeidung eines Big Ball of Mud. Ein größerer sinnvoller Diff ist nicht schlechter als eine lokale Scheinreparatur. |
| **Q5: Wie viel Aufsicht braucht die Arbeit wirklich?** | Sequenz T1–T6, Setup und Wiederaufnahme; `intervention`, `recovery`, `wait`, `assessment` mit Mensch-/Koordinator-/Agentenzeit und Ursachen. | Anteil unabhängig fachlich akzeptierter Aufgaben ohne externe Nachhilfe; längste korrekt abgeschlossene Folge; notwendige/vermeidbare Eingriffe, unnötige Eskalationen, selbständig gelöste Probleme und offene Blockaden. Autonome interne Koordination zählt als Aufwand; externe Overseer-Nachhilfe ist Intervention. | **Messbar, noch nicht beobachtet:** aktuelle Schemafelder erlauben Ereignisdetails, erzwingen diese Kennzahlen aber nicht. Bisherige Runner-Diagnosen sind keine autonomen Arbeitstrials. Echte Unterbrechungs-/Wiederaufnahmefähigkeit bleibt nur insoweit belegt, wie sie tatsächlich auftritt oder später gemeinsam geprüft wird. |
| **Q6: Lohnt sich der zusätzliche Mechanismus?** | Ergebnisqualität über T1–T6 getrennt von Methodenpflichten und kompletter Ereignisbilanz: Einrichtung, Modellierung/-pflege, Prüfung, Koordination, Implementierung, Integration, Korrektur, Wiederaufnahme, Zeit und Rohusage. | Ergebnisdimensionen/harter Fehler zuerst; Kosten und Interventionen getrennt. Kein Qualitätsbonus für YAML, Ministerien, Schichten oder Agentenzahl. Classic/Government-Prüfungen verbrauchen dasselbe Budget; ein Verfahren darf wenig bringen oder verlieren. | **Explorativ nach sechs Basis-Trials:** ein Trial pro Zelle liefert keine Varianz, Zuverlässigkeit oder kausale Erklärung einzelner Mechanismen. Fehlende Nutzungs-/Preis-/Zeitbelege bleiben null, kein Vorteil aus geschätzten Nullkosten. |

## Gemeinsame Auswertung ohne versteckte Zusatzanforderungen

Die Ergebnisrubrik und ihre harten Gates bleiben unverändert. Q1–Q6 erhalten
zunächst die Belegzustände **beobachtet**, **teilweise**, **ungetestet** oder
**unbekannt** mit schriftlicher Begründung, keine neue gewichtete Gesamtnote.
Ein Methodenpflichten-Pass ersetzt keinen Verhaltenspass. Ein technisch
abgeschlossener Prozess/Turn kann eine fachlich gescheiterte Aufgabe liefern;
`completed`, Ressourcenstatus und unabhängig akzeptiertes Ergebnis werden getrennt.

Unabhängige Assessoren prüfen eingefrorene Kandidaten gegen freigegebene Sätze.
Implementierer, Kartenautoren und Ressortprüfer vergeben keine eigenen finalen
Noten. Eine Blindung, die durch Dateien erkennbar wird, wird als Grenze offengelegt.
Der [Holdout-Ablauf](holdout-procedure.md) enthält nur andere Prüfausprägungen
bekannter Anforderungen. Ein nicht auf eine freigegebene Regel zurückführbarer
Check wird ausgeschlossen oder vor betroffenen Trials gemeinsam veröffentlicht.
Für T6 wird semantisch gleichwertige Injektion unabhängig gebunden; ohne diese
Äquivalenz ist die betroffene Zelle nicht vergleichbar.

Ergänzende Beobachtungen werden als schlanke Assessment-/Intervention-Records
oder schriftliche Bewertungszeilen an vorhandene Ereignisse gebunden, ohne neue
Plattform oder Schemaänderung. Vor dem Zell-Freeze wird das gemeinsame Format
festgelegt: `questionId`, Kandidat/Input/Cutoff, Quellenregel, beobachteter Befund,
Unklarheit/Annahme/Rückfrage, erforderliche oder vermeidbare Hilfe samt Grund,
Beginn/Ende und Dauer soweit belegt, sowie fachliche Annahme durch unabhängigen
Assessor. Fehlende Werte bleiben null; ein nicht vorhandenes Beobachtungsintervall
wird nicht als Driftlatenz null gewertet. Für die Autonomiequote ist der Nenner
die Zahl der freigegebenen Aufgaben, nicht bloß die fertig gemeldeten Aufgaben;
unvollständige/ungültige Aufgaben und ihre Gründe werden separat ausgewiesen.

Setup, Brownfield-Erkundung, Modell-/Mandatspflege, lokale/subagentische Prüfungen,
Wartezeiten, Koordinatorhilfe und externe Aufsicht werden nach dem gemeinsamen
Ressourcenprofil erfasst. Parallele Agentenintervalle werden als Agentenaufwand
summiert, nicht als zusätzliche Wandzeit; Kind- und Elternverbrauch werden nicht
doppelt zugeordnet. Gemeinsame Studienvorbereitung/Runner-Diagnose erscheint
separat als Studienkosten und wird nicht unbemerkt einem Arm geschenkt oder als
Produktqualität gewertet. Interventionsfreiheit beweist keine delegierte Autorität.

Tool-/Versuchsmessung nennt ihre Quelle: strukturierte stdout-Ereignisse,
anderweitig belegte Versuche (etwa stderr-Routerfehler), gestartete Prozesse und
erfolgreiche Wirkung sind verschiedene Beobachtungen. Der fünfte Kontextlauf
belegt zwei policy-abgewiesene Versuche trotz stdout-Zähler null. Daraus werden
weder erfolgreiche Dateizugriffe noch Provider-Request-Zahlen abgeleitet. Rohusage
Input+Output bleibt getrennt von darin überlappenden Cache-/Reasoning-Zählern;
abgerechnete Kosten bleiben getrennt von datierten Preislisten-Schätzungen.

## Benannte Lücken und kleinste gemeinsame Ergänzung

**G1 — Ausdruck/Elicitation:** Für Q1 fehlt ein gemeinsamer öffentlicher Schritt,
in dem jeder Arm vor Umsetzung die freigegebenen Begriffe, Regeln, Prioritäten,
offenen Annahmen und Freiheitsgrade zurückgibt und die tatsächliche Nutzerannahme
gebunden wird. **Vorschlag:** ein kurzer identischer Absichts-/Entscheidungsbogen
zum gemeinsamen Brief, Ausgabeformat frei, einschließlich gewöhnlicher Markdown-
Dokumentation im Conventional-Arm. Keine API-/Fixture-Neukonstruktion. Dieser
Bogen ist noch keine Actor-Pflicht und wird ohne vorherige gemeinsame Übernahme
nicht gegen bestehende Karten benotet. Ohne ihn bleibt Q1 im Basisvergleich eng
auf vorgegebene Regeln beschränkt; reale Nutzerbedienbarkeit bleibt ungetestet.
Nutzerklärungen und Annahme folgen in allen Armen denselben vorab freigegebenen
Regeln; jede Antwort, Hilfe und Annahme samt Zeit-/Interventionsaufwand wird
erfasst. Der Schritt darf keine zusätzliche unbegrenzte Nachhilfe für einen Arm
einführen.

**G2 — Echte Modelllücke/Ideenumsetzung:** T5 ist eindeutig, T6 eine angekündigte
Reparatur. Sie prüfen weder echte Zielmehrdeutigkeit noch eine neue offene Idee
mit delegierter Modelländerung. **Kleinster späterer Vorschlag:** ein gemeinsames
öffentliches Änderungs-/Entscheidungspaket am akzeptierten Zwischenstand, mit
derselben bekannten Unsicherheit, erlaubten Entscheidungsrechten und Eskalations-
grenzen für alle Arme. Keine geheimen neuen Anforderungen. Erst nach tatsächlichen
Basisbefunden als eine mögliche Folgefrage priorisieren; nicht jetzt injizieren.

**G3 — Langzeitwartbarkeit/Mechanismusursache:** Sechs kleine Aufgaben können
lokale Kopplung und Regressionen zeigen, keine langfristige Architekturgesundheit
oder Ursache eines Komplettvarianten-Vorteils. Zunächst diese Grenze berichten.
Eine weitere gematchte Änderungsaufgabe/Replikation ist nur dann sinnvoll, wenn
sie einen konkret beobachteten Befund trennt; keine neue Modulpflicht allein für
eine Architekturmetapher.

**G4 — Mess-/Ausführungsfähigkeit:** Ereignisschema erlaubt Details, garantiert
aber weder vollständige Erfassung noch Toolfähigkeit. Die aktuelle Policyblockade
und fehlende erfolgreiche Sentinels halten S1 offen. Ein unterstützter sicherer
Ausführungspfad und vorab gemeinsam gebundene Beobachtungsregeln sind vor Zellen
nötig. Diese Matrix autorisiert keine Runner-Korrektur oder Modellprobe.

## Endliche Reihenfolge und erwarteter Abschluss

Zuerst gelten die sechs Basiszellen (drei Arme × Greenfield/Brownfield), jeweils
frisch, gematcht und erst nach Readiness/Versions-/Ressourcen-Freeze freigegeben.
Der erste Zwischenbericht folgt nach deren unabhängiger Auswertung, auch bei
wenig Nutzen, technischen Grenzen oder negativen Ergebnissen. Aktuell ist **keine
Studienzelle gelaufen**; fünf separat freigegebene Diagnosestarts ersetzen sie nicht.

Danach wählt Overseer höchstens **zwei** Fragen aus tatsächlichen Befunden. Maximal
**acht zusätzliche vollständige Trials** dienen gematchten A/B-Fragen oder
Replikationen innerhalb derselben Obergrenze. Das sind Planungsobergrenzen, keine
jetzige Lauf-/Budgetfreigabe und keine statistisch ausreichende Stichprobe.
Hypothese, gleiche Gesamtressourcen, Erfolg/Abbruch, Versionen und bekannte Regeln
werden jeweils vorher eingefroren. Produktreparatur erzeugt eine neue Version;
alte Trials werden nicht nachträglich verbessert oder neu als Holdout ausgegeben.

Der Endbericht muss pro Ausgangslage Conventional als mögliche ausreichende Wahl,
Classic-Nutzen und Government-Mehraufwand/Fehler getrennt begründen. Er nennt
**minimalen sinnvollen Produktkern**, **optionale Mechanismen**, zu vereinfachende
oder zu verwerfende Teile und **offene ungetestete Hypothesen**. Umgebungsfehler,
Produktfehler, Agentenfehler, Modellierungsprobleme und Studiengrenzen bleiben
unterscheidbar. Keine Bewertung bevorzugt Markitect-Dateien oder Schichtenzahl.

Der konkrete Bedienvorschlag muss folgende Kette an tatsächlich geprüften
Versionen und Kandidaten zeigen: Nutzerbrief/Prioritäten → Fragen und Ist-/Soll-
Status → akzeptiertes Modell mit Zweck und Realisierungen → Einrichtung im Branch
→ begrenzte Idee/Änderung → unabhängige Prüfung und Modellpflege → Wiederaufnahme
oder Konfliktentscheidung. Zu jedem Schritt gehören reale Eingaben, Dateien,
Befehle, Resultate und verbleibende menschliche Entscheidungen. Solange passende
Classic-/Government-Versionen und Arbeitsfähigkeit nicht eingefroren/belegt sind,
werden hierfür keine erfundenen CLI-Rezepte als bereits nutzbar ausgegeben.

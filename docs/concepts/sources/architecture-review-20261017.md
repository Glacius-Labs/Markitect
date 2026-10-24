# Owner instructions: architecture review and concept freeze

Date: 17 October 2026. Source: the owner's Claude Code session in which the assistant was appointed project architect. The German passages below preserve the owner's words. The [architecture review](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/apply-architecture-review-20261017.md) holds the assessment, and [DEC-040](../register.md#dec-040-concept-freeze-until-the-core-runs) records the decision.

## Request for the review

> Du bist ab jetzt der Architekt. Schaue dir an woran wir aktuell arbeiten (commits, branches, prs, etc) bzw welche Entwicklungen parallel laufen. Ließ dir auch das neue Konzept durch und alle relevanten Dokumente die du finden kannst. Danach starte eine tiefe Analyse über das Projekt und das Konzept und sage mir was du davon hälst. Wichtig ist mir vor allem Sauberkeit und architektonische Qualität. Dann sage mir was du denkst, was fehlt, was du anders machen würdest falls dir was auffällt und wo du Verbesserungsmöglichkeiten siehst.

## Concept freeze and order of work

Recorder's note: the owner answered the assistant's summary of the review, which recommended a freeze "until one Package runs end to end".

> Das Konzept ist jetzt eingefroren. Halte erstmal deine Findings und dein vorgeschlagenes Vorgehen fest. Dann versuche herauszufinden woran aktuell aktiv gearbeitet wird und ob der neue Coordinator anfängt das Konzept umzusetzen. Falls er das nicht tut, sollten wir erst das Konzept fertigstellen und die Aufräumrunde starten. Die Aufräumrunde wäre generell gut zu starten unabhängig davon. Sobald klar ist was aktuell passiert und wir grünes Licht haben, dass wir mit Codex nicht in die Quere kommen und das Konzept sauber, einheitlich und ausgearbeitet ist, können wir die Implementierung hier übernehmen. Theoretisch kann ich den Coordinator auch von dir aus etwas fragen oder ihm eine Nachricht übermitteln, sodass er weiß was wir machen und uns nicht stört und ggfs. stoppt.

Recorder's note: in this passage, "er" refers to the Coordinator, and "das" to implementing the concept.

## Division of work

> Ich habe dem Coordinator gesagt, er soll sobald er seinen Plan fertig hat warten und ihn uns mitteilen. Ich werde ihn dann dir übergeben und dann antworte ich ihm entsprechend. Wir kümmern uns um Konzept und Herzstück. Er kümmert sich dann um die Dinge aussenrum schlage ich vor.

> Jedenfalls weiß er dann bescheid was wir machen bevor er loslegt und wir können direkt starten. Der Coordinator wird dann um uns herum arbeiten.

Recorder's note: the Coordinator acknowledged the division in a message the owner relayed. In it, Claude takes the core, the concept and the ports, and the Coordinator takes the adapters and the remaining infrastructure.

## Plan in the repository

> Wie wäre es hiermit: Du fängst an mit deiner Arbeit und der Coordinator wartet bis er in deinem Branch oder irgendwie einen Plan findet. Dann plant er drumherum und wir kümmern uns nicht um ihn

## Core protocol choices

Recorder's note: the assistant asked four questions with prepared options, and the owner selected one option for each. The question and the selected option are reproduced exactly.

| Question | Selected option |
| --- | --- |
| Fragen an einen bekannten Empfänger: Soll der Host sie direkt entlang des Baums weiterleiten, ohne die Manager dazwischen als Agenten aufzurufen? Diese sehen die Nachricht nur in ihrem nächsten Kontext. | Host leitet weiter (Recommended) |
| Wie sieht ein Manager die Lieferungen anderer: Soll jeder Schritt auf dem aktuell akzeptierten Gesamtstand starten, plus den eigenen noch offenen Änderungen? | Aktueller Gesamtstand (Recommended) |
| Darf ein Manager, der beim Explore findet, dass nichts zu ändern ist, direkt ein „keine Änderung“-Ergebnis einreichen? Das Review dazu läuft dann noch vor der Schreibsperre. | Ja, direkt einreichen (Recommended) |
| Wenn eine Aufgabe nicht fertig werden kann (Ablehnungslimit oder offene Owner-Frage ohne Ausweg): Soll der Host dann keine neue Arbeit mehr zulassen und den Lauf als Incomplete beenden? Die Arbeit bleibt erhalten. | Sofort stoppen (Recommended) |

Recorder's note: the option descriptions shown to the owner were the following.
- Host forwarding keeps the tree as the path and records every hop, without Agent turns; the common parent is dispatched only as addressee or for a dispute or circular wait.
- The current aggregate avoids stale views; a review is renewed only when a delivery the candidate used has changed.
- The direct no-change submission saves one step per unchanged Manager; independent review stays mandatory.
- Stopping at once is cheaper and simpler; a continuation takes over the retained work.

## Container lifetime and waiting

Recorder's note: after the independent review of the core protocol, the assistant asked two more questions with prepared options. The owner selected one option for each.

| Question | Selected option |
| --- | --- |
| Container zwischen den Schritten eines Managers: Am Vormittag hast du eine endliche Warmhalte-Regel bestätigt, statt nach jeder Nachricht zu stoppen. Soll die erste Apply-Version trotzdem jeden Container nach seinem Schritt stoppen? Das Gespräch wird dann aus dem Provider-Verzeichnis fortgesetzt, das ist seit #352 für Codex belegt. | Erst stoppen, warm später (Recommended) |
| Wartet ein Lauf auf deine Antwort, läuft kein Container mehr. Darf der Markitect-Prozess sich dann beenden? Du antwortest später mit `markitect answer` und setzt den Lauf mit `markitect resume` fort. | Ja, Prozess darf enden (Recommended) |

Recorder's note: the option descriptions shown to the owner were the following.
- Stopping after each step means a fresh container for every step. No quiescence proof, write barrier or basis change in a running container is needed for the first version. Warm retention follows as an optimization when measurements show that startup time matters. This changes the morning's confirmation for the first version.
- Letting the process end means no process has to run for hours. Answer and continuation are separate verbs in the target command set. Resumption after a Host crash still needs its own evidence.

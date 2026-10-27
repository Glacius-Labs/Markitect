# Owner discussion on organization and trust

Date: 16 October 2026.

Recorder's note: Direct owner messages from the Markitect chat comparing the Manager north star with Main and discussing future use. The two statements below are preserved in their original language and order; paragraph spacing is normalized. The owner subsequently asked to keep these thoughts in the repository documentation, preferably on the concept branch. The [reflection](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/concepts/reflections/organization-and-trust-20261016.md) separates the summary from assistant interpretation. These statements describe desired outcomes, experience and hypotheses; they do not adopt every assistant proposal from the conversation.

## Desired working life

> Ich möchte meine Zeit damit verbringen, zu modellieren, Ideen und Konzepte auszuarbeiten, Regeln aufzustellen. Dann möchte ich zuverlässig wissen, dass diese sauber und einheitlich und durchdacht umgesetzt werden, nichts übersehen wird, automatisch features entwickelt, bugs gesucht und behoben und vollumfänglich für Sauberkeit und Hygiene gesorgt wird. Dann würde ich der Organisation beim arbeiten zuschauen, auf wichtige Rückfragen antworten und dabei zuschauen wie aus Idee Realität wird ohne dass es entartet.

## Management analogy and current practice

> Hier ist mein Gedanke dazu. Ich sehe ein Unternehmen oder eine Regierung ähnlich wie ein Software System. Jedes Unternehmen gliedert sich in Bereiche, muss verschiedene Teilprobleme lösen und auf seine Umwelt reagieren und schlussendlich ein Ziel erreichn. Dabei möchte der Geschäftsführer nicht jedem seiner Mitarbeiter auf die Finger schauen müssen. Daher kontrolliert er es über ein hierarchisches Management System, welches auf Delegation und Eskalation basiert. Dabei wird er nur informiert, wenn etwas dramatisches oder sehr wichtiges zu klären ist. Andererseits kann er von oben herab Weisungen erteilen oder das Unternehmen anderweitig steuern. Dabei wird generell vor allem auch mit Anreizen gearbeitet statt mit ständiger Kontrolle. Ansonsten erhält der Geschäftsführer regelmäßige Reports und kann reagieren wenn etwas nicht zu passen scheint.
>
> In etwa so stelle ich mir die Arbeit mit AI Agenten auch vor. Aktuell arbeitet man bei uns so:
>
> - Planen von User Stories
> - Nehmen der User Story
> - Claude oder Codex => Implementiere diese Story und erstelle einen PR
> - Claude oder Codex =>Reviewe den PR
> - Claude oder Codex => Beobachte den PR und reagiere auf Findings bis alles passt
> - Human => Review des Codes, in der Realität wird das aber selten gemacht
> - Merge to main
>
> Je nachdem was gerade auffällt gibt es einen Mitarbeiter der es dann macht und sich dafür einen AI Agent holt und ihm Anweisungen gibt. Je nachdem wie groß die Aufgabe ist, werden dabei auch gerne Dinge übersehen von nur einem Agenten. Er ließt ja nicht die ganze Code Base aus 20 verschiedenen Blickwinkeln. Aus meiner Arbeit mit AI Agents fällt mir auf, dass auch diese sich "konzentrieren" und aus einer bestimmten Sicht heraus arbeiten.&#x20;
>
> Wenn wir das in die Unternehmensanalogie nehmen kann man sich die AI so vorstellen, wie einen extrem starken Mitarbeiter. Dieser könnte vielleicht den Job von 5, 10, 15 Mitarbeitern übernehmen. Aber auch hier ist zu erwarten, dass er immer mal wieder etwas übersieht oder einen Aspekt auslässt. Daher halte ich es für effizienter und vertrauenswürdiger wenn viele "dümmere" Mitarbeiter sich jeweils nur auf eine Sache konzentrieren und das entsprechend im "4 Augen Prinzip" abarbeiten.&#x20;
>
> Ich möchte also die Vorzüge von Struktur und Systematik nutzen und Erkenntnisse aus Management um AI Agents effizienter zu nutzen und mehr Vertrauen in ihre Arbeit zu bekommen. Nicht weil das Modell stark ist sondern weil es quasi 20 mal aus verschiedenen Blickwinkeln darauf geschaut hat. Dann brauche ich PRs tatsächlich nicht zu reviewen.
>
> Auch wenn es um relativ lokale Entscheidungen geht, würde ein klassischer Mitarbeiter die Thematik einer AI erklären und sich Vorschläge geben lassen und in den meisten fällen ist die Empfehlung korrekt, ausser es fehlt Kontext.

## Retain objections and team design questions

Recorder's note: Follow-up on the same date after the first documentation commit. The owner explicitly asks to preserve the objections and names configurable teams as a way to realize different perspectives and the four-eyes principle. This request does not adopt every assistant elaboration.

> Hast du auch deine Einwände dort abgelegt? Gerade Dinge wie die Erfahrung, verschiedene Blickwinkel bzw Vier Augen Prinzip (realisierbar durch konfigurierbare Teams) etc finde ich sehr wichtig und diese sollten nicht vergessen werden

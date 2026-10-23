# Idea review with the owner, 10 October 2026

- **Source type:** owner statements preserved verbatim (German), a review message the owner confirmed as accurate (verbatim), and recorder's notes in English. The recorder's notes are labeled as such and are not owner statements.
- **Setting:** a Claude Code review session analyzed the integration and Government branches and gave an opinion on Markitect's basic idea. The conversation took place on 10 October 2026, while the integration candidate was being merged into Main.
- **Use:** the [register](../register.md) records the resulting decisions, promises, assumptions, concepts and enhancements. This file preserves where they came from. Do not edit the quoted text. Record a later change of meaning as a new or superseding register entry.

## Recorder's note: why this review is preserved

This note is the recorder's account, not an owner statement. The review first misread Markitect. It treated Markitect as a deterministic checking and context tool for AI instructions. It called the delegated Manager execution scope creep and recommended dropping Markitect's own execution. It also read the failed comparative studies as evidence against the concept. The owner rejected that reading. A corrected summary was then confirmed. The same misreading can come from reading the origin of the name, the retained compatibility contracts or dated study results without the current thesis. The [vision](../../vision.md#common-misreadings) now names these misreadings so that later sessions need not repeat this discussion.

## Owner correction

```text
Also zuersteinmal ist main absolut veraltet und das ist bekannt. Der Integrator arbeitet daran viele Arbeiten zu integrieren und als stabilen Stand nach main zu bringen. Nachdem ich mir deine Beschreibung und Bewertung durchgelesen habe habe ich starke Zweifel ob du verstanden hast was Markitect erreichen will und was wir uns davon versprechen und warum. Fasse nochmal zusammen wie dein Verständnis ist aber ich vermute du hast es nicht verstanden
```

## Summary confirmed by the owner

The review session wrote the following message. The owner answered: „Jetzt hast du es richtig verstanden und das Wesentliche begriffen.“ The wording is preserved; only its Markdown formatting (headings and bold lead-ins) is removed. The message ended with a question asking whether the reading was now correct; that question is omitted.

```text
Was Markitect ist
- Ein Modell als einzige Quelle. Der Mensch pflegt nur noch eine maßgebliche Stelle: das kanonische Modell. Es beschreibt das Soll des Projekts, also Verhalten, Architektur, Regeln, Grenzen, Zuständigkeiten und Arbeitsweise. Es ist das Weltbild des Projekts, keine Sammlung von Agenten-Anweisungen. Code, Tests und Doku sind nur Realisierungen davon.
- Änderungen werden angewendet wie beim Kompilieren. Man ändert das Modell, meist im Gespräch mit dem eigenen Agenten, und lässt die Änderung anwenden. Das Modell ist die Quelle, das Repository das abgeleitete Ergebnis, und die delegierte Agentenarbeit übernimmt die Rolle des Backends.
- Der Compiler prüft vorher. Er prüft Typen, Referenzen und Zuordnungen, damit Fehler im Modell vor der Umsetzung auffallen und nicht erst „zur Laufzeit“.
- Was gehört wozu. Jede Datei ist einem Teil des Modells zugeordnet. Deshalb lässt sich aus jeder Modelländerung deterministisch ableiten, welche Pflichten und Dateien betroffen sind und wer zuständig ist. Das ist der Blast Radius.
- Die Struktur ergibt die Organisation. Aus den Namespaces entsteht eine rekursive Manager-Hierarchie. Jeder Manager verantwortet einen Ausschnitt und dessen Dateien, setzt ihn um oder delegiert weiter. Unabhängige Reviewer prüfen, die übergeordneten Manager integrieren. Am Ende stehen eine Gesamtprüfung und ein geschütztes Apply.

Warum
- Das Wissen muss raus aus dem Agentengedächtnis. Agenten vergessen ab einer gewissen Projektgröße Stellen, und Konsistenzläufe skalieren nicht. Was woran hängt, muss deshalb explizit und prüfbar im Modell stehen.
- Der Mensch ist der Engpass. Um heute sicher zu sein, müsste er jeden Diff lesen, und das bremst die KI aus. Künftig beurteilt er die Absicht, also die Änderung am Modell. Dem Ergebnis vertraut er wegen der Methode: Die Betroffenheit ist vollständig ermittelt, es gilt das Vieraugenprinzip, und es gibt einen Gesamtabgleich. Das ist so, wie man einem Compiler vertraut, ohne den Assembler zu lesen.
- Verständliche Software. Über das Modell kann man schlussfolgern, was das System tut und ob es das tut, was es tun soll.
- Die Spezifikation bleibt dauerhaft wirksam. Sie ist kein einmaliges Feature-Dokument. Jede Änderung muss neben der neuen Absicht auch alle weiterhin geltenden Pflichten im Blick behalten.

Was ihr euch davon versprecht
- Nichts wird vergessen, auch nicht in großen Projekten. Cleanups und Refactorings werden dadurch verlässlich.
- Günstigere Modelle reichen, weil jeder Agent nur kleine, klar begrenzte Aufträge mit passendem Kontext bekommt.
- Mehr Vertrauen in Coding Agents ohne Detailkontrolle. Dazu saubere, einheitliche Repositories, in denen sich Ordnung lohnt, statt in einer Aufräumhölle zu enden.

Was das an meiner Bewertung ändert
- Empfehlung zurückgenommen. Ich hatte geraten, Markitect ohne eigene Runtime nur als CI-Prüf-Gate zu bauen. Das würde das Produkt halbieren.
- Zwei Dinge vermischt. Ich habe die Government-Variante (Ministerien, Kabinett, Gericht) mit der Manager-Hierarchie gleichgesetzt. Die Evidenz sprach gegen die Government-Variante. Die Hierarchie mit unabhängigem Review gehört zur Grundidee.
- Fehlschläge falsch gewichtet. Der Abbruch im Luna-Vergleich und die 28 Anläufe gehen auf Bugs und Runtime-Probleme zurück. Sie zeigen, dass die Umsetzung noch nicht reif ist, sagen aber nichts gegen das Konzept.
```

## Review points on the confirmed basis

Recorder's summary: the review then gave five points. In brief:

1. **Where the compiler analogy breaks.** Agent realization is not deterministic, so trust has to come from verification. The review suggested splitting the promise into two parts. One is "nothing in the modeled scope is overlooked", which can be computed. The other is "everything is realized correctly", which needs checks.
2. **Model granularity.** The model can be so fine that it mirrors the code by hand, or so coarse that every change affects everything. The test the review proposed: is a model diff clearly smaller and easier to understand than the code diff it causes?
3. **A gap in completeness.** Declared mappings do not show the dependencies between files in code, such as calls, imports and shared types. A shared function changed by one Manager can affect another Manager's obligations without appearing in impact.
4. **Hierarchy and models.** Keep the hierarchy for responsibility and scoping, keep delegation chains as shallow as the work allows, and use capable models for integration. Independent review gains more when reviewers differ in model as well as in context.
5. **Where the value shows.** The expected benefit appears over a series of changes on a long-lived project. The earlier comparisons never reached that point.

## Owner response to the five points

```text
1. Compiler Bruchstelle
Im Grunde hast du mit der Bruchstelle recht. Gleichzeitig will ich hier nur die Idee hinter dem Compiler übernehmen als Konzept. Heutzutage sind AI assisted Coding und Spec Driven development absolut üblich und beliebt. Man vertraut dem AI Agent sowieso schon mehr oder weniger genug dass viel Code gar nicht erst geprüft wird und PRs zwar vorsehen, dass ein Mensch es reviewed und freigibt, aber in der Praxis übernimmt eine AI das Review und die Freigabe kommt meistens mehr oder weniger blind. Daher versucht Markitect hier das Vertrauen durch eine saubere Struktur und Methodik und "Föderalismus" zu erhöhen. Tests sind eine Ausprägung des Modells, allerdings werden auch diese durch AI Agents geschrieben und können damit ebenso falsch sein wie der Code selbst.
Prompting sieht meistens so aus: "Hier ist PR, reviewe ihn" oder "Hier ist ein Work Item, implementiere es" und trotzdem wird der AI hier größtenteils vertraut auch wenn sie oft nicht besonders zuverlässig arbeitet, gerne mal Fehler macht und übersieht, Probleme hat sich selbst im selben Thread zu reviewen und leicht Regeln und Policies übersieht wenn sie auf keine Hindernisse stößt. Ausserdem ist ein echter Compiler, der ein mehr oder weniger dokumentiertes Wissensmodell bzw natürlich beschriebenes System ausließt, Kontext miteinbezieht, selbstständig nicht definierte lücken schließt und den Intent dahinter interpretiert und gleichzeitig deterministisch ist und wie ein klassischer Compiler funktioniert und dieselben garantien geben kann. Ausserdem sind Skriptsprachen und ByteCode ebenso nicht zwanghaft deterministisch.

2. Wie fein das Modell sein muss
Wie das Modell genau gepflegt wird hängt vom Entwickler ab. Es kann genauso unterspezifiziert oder überspezifiziert sein wie Work Items, Prompts, Dokumentation und Code. Das wird man dem Entwickler nicht nehmen können und höchsten mithilfe von AI und Lerneffekten Prüfung, Evaluierung und Empfehlungen geben können.

3. Vollständigkeit hat Loch
So wie du es beschreibst ist es eben auch gedacht gewesen. Die Idee war nicht eine File darf nur zu einer Definition im Modell gehören. Ausserdem ist es so gedacht, dass im Compiler alle files getracked sein müssen ausser sie wurden über .markitectignore explizit ignoriert. Ansonsten "kompiliert" es nicht.

4. Mit schwachen Modellen sind Luna und Sonnet gemeint. Unter Umständen für Sparfüchse und falls es sich beweist auch Haiku. Diese Modelle sind keineswegs "schwach" aber auch nicht high end Fable, Opus und Astra. Die Management Ebene verbindet mehrere Konzepte: Abstraktion, Managementstrukturen in Unternehmen. Dies Konzepte haben sich bisher in der echten Welt bereits bewährt und wurden bisher noch nicht richtig evaluiert. Vieraugenprinzip bei Implementern und Reviewern (unterste Ebene der Managementhierarchie) ist genau so wie aktuell in Wirklichkeit gearbeitet wird. Das habe ich oben berichtet und sehe es täglich genau so in Aktion. Ausserdem hat ein Talk von Anthropic selbst und ein paar Paper bereits angemerkt und beobachtet, dass derselbe Agent mit demselben Modell sich weniger kritisch selbst reviewed als wenn man einfach einen neuen Chat mit demselben Modell startet und ihn damit reviewed. Die Abneigung gegen Selbstkritik scheinen die LLMs von Menschen geerbt zu haben.

5. Der bisherige Wert wurde bisher noch nie richtig gemessen. Das Hauptproblem ist Versuchsarbeit und Dependencies und Environment Issues wie Windows Pfade, CLRF, Berechtigungen und die Codex CLI. Das Konzept selbst wurde bisher nicht wirklich getestet und lässt damit keine Aussage zu. Ausserdem lassen sich diese Teilprobleme sicher lösen. Die Frage ist ob das Konzept und die Idee tatsächlich dazu führen können, dass man an einem Modell bzw Weltbild entwickelt, dieses in ein Repo zuverlässig verwirklichen lassen und prüfen kann, ob man damit ein Repo widerspruchsfrei, konsistent, einheitlich und sauber halten kann und ob man dadurch erreicht, dass man Agentic Coding relevant besser vertrauen kann, Regeln besser eingehalten und nicht übersehen werden und man seine Konzentration aufs Entscheiden und Modellieren veralgern kann während der Code zuverlässig das Modell reflektiert.
```

**Recorder's note:** in point 1, the sentence beginning „Ausserdem ist ein echter Compiler …“ breaks off without its predicate, for example "is not achievable". The register reads it that way: such a compiler is not achievable, so the analogy is conceptual. That reading is an interpretation and is marked as one in [DEC-003](../register.md#dec-003-the-compiler-is-a-borrowed-concept).

## Follow-up proposals and owner dispositions

Recorder's summary: after the owner's response, the review accepted the trust baseline and the owner's points on granularity, coverage and models. It then proposed:

- a dedicated review of tests against the statement each test claims to check;
- a linter and metrics for model quality;
- an observation adapter that derives dependencies between files from code. The current architecture keeps inferred source-code semantics out of Core;
- per-role model mixing;
- seeing the model as the institutional memory that makes a management hierarchy workable for stateless agents;
- evaluating the method separately from the execution runtime, in a stable Linux environment.

The owner answered:

```text
Bei der Vollständigkeit gebe ich dir Recht. Das Problem wäre aber dass das Code spezifisch ist und sich nicht auf alle Arten Dateien auf dieselbe deterministische Weise prüfen lässt. Natürlich könnten wir Tools oder Plugin anbieten die sowas machen. Generell sehe ich das Problem aber auch und finde wir sollten das verfolgen und uns hier etwas einfallen lassen.

Was Linter für Modellqualität und Kennzahlen angeht halte ich das für eine sinnvolle Erweiterung die wir auch verfolgen sollten.

Das mit den Tests beim Compiler können wir so präzisieren und als Variante aufnehmen oder eventuell sogar auch direkt eibauen. Auch das halte ich für sinnvoll und verfolgenswert.

Modelle variieren und kombinieren und evaluieren halte ich für eine gute Idee. Für jeden "agent" also inklusive Manager, Imlementer und Reviewer war schonmal die Möglichkeit einer eigenen Konfigurationsmöglichkeit geplant. Das sollten wir definitiv auch verfolgen.

Was deinen Vorschlag mit der Runtime angeht, das klingt sehr interessant und sinnvoll.

[…]

Ich schlage jetzt folgendes vor. Ich möchte dass du dir den aktuellen main Stand anschaust und auf Basis unserer Diskussion schaust, dass aus der Dokumentation die Idee herausgeht die wir hier erst klären mussten und das Besprochene hier klar ist und diese Diskussion nicht jedes mal aufs neue geführt werden muss mit denselben Anmerkungen und Erklärungen. Zudem sollten wir die Punkte und Themen die wir hier besprochen haben entsprechend in der Doku festhalten, sodass sie dort festgehalten als Entscheidungen, Konzepte, Annahmen, Verprechungen, Vision, future enhancements, Idee oder ähnlich. Wir brauchen hierfür einen klaren Platz in der Doku und sollten diese Informationen nicht vergessen.

[…]
```

The first omitted passage („[…]“) reports coordination status: Main integration, repository cleanup and the playground rebuild. The second asks the review session to explain the separation of concept and runtime and to plan the follow-up assignments. Neither contains a product statement.

# Owner account of Markitect's origin — 15 October 2026

**Source:** Direct statement by Markitect's product owner in the documentation task on 15 October 2026, Europe/Berlin. The text below is preserved verbatim. This is the date of this retrospective account, not a claim about when the original project or each idea began. The owner did not name the adopting project in this statement.

This account adds the concrete originating project, conflicting representations and causal sequence to the earlier [owner product description](owner-product-description-20261009.md). It does not replace that source or revise the status of existing decisions. The [origin and rationale](../origin-and-rationale.md) separates the account into problems, concepts, assumptions and hoped-for outcomes and links them to the [register](../register.md).

The Core/Module/Product rules are examples from the originating adopting project. They are not universal rules imposed by Markitect. The statement records experience and intent; it supplies no new implementation or comparative evidence.

## Verbatim statement

```text
Okay jetzt wo wir uns darüber Gedanken gemacht haben was es sonst noch so gibt und wie Spec Driven Development funktioniert lass uns nochmal zur Grundidee von Markitect kommen.
Die Idee für Markitect ist entstanden als ich für ein Projekt, mit klaren Grenzen wie Modulen, Core und Produkten und klaren Architekturregeln wie "Core darf von nichts abhängen", "Module dürfen nur von Core abhängen", "Produkte nutzen Module und passen sie jeweils für sich an und kombinieren sie", "Module dürfen nie nur für ein Produkt gebaut werden und müssen produktunspezifisch sein". Dann wollte ich bestimmte Dinge beschreiben wie den Arbeitsablauf, Review Policies, dass für jeden Use Case eine Dokumentation an einer bestimmten Stelle liegt etc. Ausserdem auch dass zu einem Modul immer diese Menge an Skills gehört, es eine Area geben soll, dass es Dokumentation geben muss und wie sie aussieht, und so weiter und so fort. Dann ist mir sehr schnell aufgefallen, dass die Dokumentation, der Code und die Skills für Claude und Codex fast immer Widersprüche haben, bei Anpassungen wurden Regeln nicht eingehalten oder einfach übersehen und die Dokumentation hatte viel "Bleeding".
Jeder Lauf durch die AI zur Konsolidierung hat mehr solcher Drifts festgestellt und hat ewig gedauert. Das ging mir irgendwann auf die Nerven, da ich irgendwann selbst nicht mehr wusste was gilt, was fehlt etc.

Daraus entstand die kanonisches Modell und Compiler Idee. Kanonisches Modell im Sinne von, was hier steht gilt und statt mehrere Repräsentationen derselben Dinge haben wir eine Repräsentation und syncen alles gegen das kanonische Modell. Der Compiler Gedanke kam mir, da ich nicht überprüfen konnte ob alles synchron ist und das immer erst "zur Laufzeit" festgestellt habe. Bei stark typisierten Programmiersprachen hilft hier der Compiler und daher auch hier die Idee zu typisieren.

Um das dann entsprechend anzuwenden kam mir die Idee mir Strukturen im Management abzugucken. Das heißt statt einen Agent der an alles denken muss, delegieren wir und haben Verantwortliche und damit viele präzise Agents statt eines God Agents (God Class). Damit nutzen wir gut durchdachte Struktur und Systematik und die Vorteile die uns konventionelle Software und Typsysteme geben um die Schwächen der aktuellen AI auszugleichen und sie effizienter zu nutzen. Am Ende habe ich mir dadurch erhofft, nur noch das Modell pflegen zu müssen und dann über Apply, Verify, Cleanup, etc anwenden zu können. Am Ende würde ich mir davon auch erhoffen, nicht mehr alle PRs und Changes durchschauen zu müssen und mehr Vertrauen und Überprüfbarkeit und Kontrolle zu haben.

Die Government Idee baut darauf auf, sodass ich die Modellpflege automatisieren kann durch Gerichte und am Ende quasi der Präsident einer Organisation bin, die sich um die Entwicklung des Software Systems kümmert.

Das ist der gesamte Verlauf und der sollte im Repository in main durch Doku auch entsprechend dokumentiert sein oder falls nicht geschehen, werden. Danach würde ich gerne diese Geschichte Strukturieren und daraus Probleme, Annahmen, Konzepte usw herausgliedern, sodass klar ist woher wir eigentlich kommen und wieso wir uns hier Erfolg versprechen.
```

## Scope of the documentation request

The owner asks for this account to be available on Main and then structured so that the problems, assumptions, concepts and reasons for expecting success are recoverable. That authorizes documenting and organizing the account. It does not start a product implementation, activate Government, select a new model syntax or establish that the hoped-for benefits have occurred.

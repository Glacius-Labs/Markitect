# What an extracted model contains — owner, 19 October 2026

Recorder's note: owner decision in the Claude Extract session, after the [first library Extract runs](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/library-extract-20261019.md). The owner asked the session to test, refine and stabilize Extract, to look at the concept and the earlier runs, and then to plan. The session proposed a four-step plan (measure first, decide the concept questions, fix in the Host rather than in prompts, then broaden and clean up) and asked three questions, each with a recommendation. [DEC-061](../register.md#dec-061-what-an-extracted-model-contains) records the decision.

## The questions

The session's questions, verbatim:

> 1. Soll die Struktur-Phase immer ein Projekt-Package vorschlagen, dessen Manager Lieferpflichten, Technik-Grenze und Non-Goals besitzt? Ich empfehle ja, weil das zur Sprache passt.
> 2. Soll als Kriterium gelten: Bindend ist, was an der Projektgrenze beobachtbar oder ausdrücklich dokumentiert ist, alles Interne bleibt Realisierungsfreiheit? Ich empfehle ja.
> 3. Darf Extract Checks aus dokumentierten Build- und Test-Befehlen vorschlagen? Ich empfehle ja, aber nur als Vorschlag zur Adoption, nicht automatisch aktiv.

Recorder's note: "passt zur Sprache" refers to language v1, which forbids Definitions directly in the model root, so project-wide obligations need a named Package; Root has no Package of its own.

## The answer

> Mache es so wie du es empfiehlst und für richtig hälst. Fange gerne direkt an

Recorder's note: the answer accepts all three recommendations and the proposed plan, and authorizes the session to start.

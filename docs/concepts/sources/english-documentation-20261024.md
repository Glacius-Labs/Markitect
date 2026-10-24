# Owner decision: documentation is written in English

Date: 24 October 2026. Source: the owner's direct message in a Claude Code session, after concept 1.0 had been recorded in German. The owner statement is verbatim; the translation and the recorder's notes are labeled.

## Owner statement (verbatim)

> Das Konzept und sämtliche Dokumentation in Markitect soll immer auf Englisch sein

English translation: "The concept and all documentation in Markitect are always to be in English."

## Recorded meaning

Recorder's interpretation:

- The V1 concept and every document in the repository are written in English, from now on and permanently.
- Concept 1.0 was translated into English in the same change.
- Existing German documents are translated as separate work; the delivery plan lists them.

Recorder's note on the scope:

- Verbatim owner statements under `docs/concepts/sources/` are provenance records. They keep their original wording, as the concept record rules already require, with an English translation added where a source serves as a reference.
- The German bodies of the canonical `markitect/core` vocabulary are model content inside the protected language foundation, not documentation. Translating them changes the identity of the core vocabulary, so it needs its own owner decision.
- The assistant reported both points to the owner.

## Follow-up owner decision: core vocabulary

Recorder's note: the assistant explained that the bodies of the canonical `markitect/core` vocabulary (`Kind`, `Property`, `name`, `type`, `minCount`, `maxCount` and the other core Definitions) are German model content whose wording enters the core identity. Translating them keeps their meaning but changes the core identity, the digests of models built on it and the compiler test vectors, and would cleanly need a new core version. The assistant recommended translating them before the native requalification, which re-establishes all evidence anyway, and asked for a decision.

Owner decision (verbatim):

> Ja alles auf englisch

English translation: "Yes, everything in English."

Recorded meaning (recorder's interpretation): the `markitect/core` vocabulary bodies are translated into English with unchanged meaning, as a new core version, together with their copies in the language specification and its examples. The work is a separate slice at the protected language foundation, scheduled before the step 4 native requalification.

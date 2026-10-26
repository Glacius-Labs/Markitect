# Owner decision: interfaces instead of inheritance

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim; the session's proposal is summarized and its question quoted, both labeled. The recorded meaning is the recorder's and adds no decision.

It amends the owner's [W1 decisions](model-track-w1-decisions-20261026.md) of the same day, which confirmed Kind extension (`extends`) for G1, before any of that code was merged.

## Owner statements (verbatim)

> But i also want to prevent us from waiting and spending a lot of tokens and then refining again. Thats why we should talk about what we are going to do. For example one topic would be if extending is really a good idea. I mean inheritance. Go for example does not do that and maybe it would be a good idea for us to not go there too. Also the kind and type topic could be discussed.

Context (summary by the recorder): the session answered that `extends` served two needs: reusing a Library Kind's fields, and references that accept "any of these". It proposed to meet both without inheritance:
- field reuse needs no new feature, since a project Kind can list a Library's Properties in its own `properties`;
- Go-style interfaces give "any of these": a Kind declares that it implements an interface Kind, the compiler checks that the Kind itself lists every Property the interface requires, and a reference to the interface accepts any implementing Kind;
- no fields are inherited, so every Kind shows its whole contract, with no inheritance chains, diamonds or fragile bases, at the cost of some repetition.

On the vocabulary it proposed that Kind stays the type of a Definition, that the Core Property `type` (a field's value type) becomes `valueType`, and that the Libraries use one word, `ValueType`, without a separate `DataType` base. Both changes go into the same Core change. It asked: "My recommendation: replace inheritance with Go-style interfaces (`implements`), and clean up the type vocabulary in the same core change. Do you want to go that way, or discuss either point further first?"

> Yes lets go that way

## Recorded meaning

Recorder's interpretation:

- **Interfaces replace inheritance in G1.** A Kind declares the interface Kinds it implements with a new Core Property `implements`. The compiler checks that the Kind lists every Property each interface requires. No fields are inherited. A value reference to an interface Kind accepts Definitions of implementing Kinds.
- **The Core Property `type` becomes `valueType`.** Kind stays the type of a Definition.
- **The Core edition stays as decided:** one `markitect-core` 1.2.0 with migration of everything checked from Main, and `software-basics` 0.3.0 (Option A).
- **The details follow the reworked design** and its independent review.

# Owner decisions: track order, Library clarifications and the Makerspace architecture

Date: 27 October 2026 (UTC). Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim; the session's proposals they answer are summarized and labeled. The recorded meaning is the recorder's and adds no decision.

They belong to the [canonical model track](../../design/model-track-charter.md) ([DEC-074](../register.md#dec-074-canonical-model-track-parallel-to-v1)).

## 1. Track order

Context (summary by the recorder): the session proposed the next steps: the architecture Library first, because it lets the owner decide a project's architecture inside the model and the Makerspace remodel needs it, then the remodel (W6), then the modeling skill (W5). It also offered starting the remodel right away as an alternative.

> Start with the architecture Library, then the Makerspace remodel, then the modeling skill.

## 2. Library outline v0.3.1

Context (summary by the recorder): drafting the `practice` and `architecture` Libraries surfaced four clarifications of the accepted outline v0.3. The session marked them in `docs/design/standard-libraries.md` and asked the owner to accept them.

> I accept the v0.3.1 clarifications: appliesTo includes the Packages below a named Package and governs only its own Package when absent or empty; an Interface doesn't bind its own Package; DependencyRule counts only dependencies between project Packages; build and test commands are optional.

## 3. The Makerspace architecture

Context (summary by the recorder): the session proposed an owner-decided architecture for the W6 remodel:
- Packages protocol, policies, members, equipment, billing, training, bookings, materials, safety and reporting;
- dependency directions such as members and equipment toward policies, billing toward members, and protocol toward all areas;
- authorization, audit and the order of checks placed in protocol.

> I approve the proposed Makerspace architecture: flat area Packages, one policies Package, protocol as the outer layer, billing owns Charge, architecture rules at Root, the data Library built first, W0 only.

## Recorded meaning

Recorder's interpretation:

- **The order is W4b (architecture Library), then W6, then W5.**
- **The outline is v0.3.1** with the four clarifications as worded.
- **The pilot uses the approved architecture** and remodels W0 only. The language forbids Definitions directly in the model root, so the session placed the root architecture rules in the top-level project Package `makerspace`, with the areas below it. That placement is a language consequence, not a further owner decision.

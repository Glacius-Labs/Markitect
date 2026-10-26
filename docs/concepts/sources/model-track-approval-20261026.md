# Owner approval: canonical model track

Date: 26 October 2026. Source: direct owner message in the Claude Code session "Markitect evaluation and canonical model design", written in English. The statement is verbatim. The recorded meaning is the recorder's, labeled, and adds no decision. The register entry and the plan section are written separately by the Coordinator.

The approved document is the [canonical model track charter](../../design/model-track-charter.md), v0.2. It follows from the owner's [founding intent](founding-intent-20261026.md) and applies an independent review of its first draft.

## Owner statement (verbatim)

> I approve the model-track charter v0.2 with the recommendations for D1–D5. Record it with the Coordinator and start W1.

## Recorded meaning

Recorder's interpretation. The approved recommendations, as the charter states them:

- **D1:** the canonical model track runs parallel to V1 and does not block V1 technical acceptance. The approval lifts the hold of [DEC-011](../register.md#dec-011-detailed-planning-of-future-ideas-waits) for this bounded track, as [DEC-065](../register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration) did for the cockpit.
- **D2:** no new register entry about the compiler. [DEC-003](../register.md#dec-003-the-compiler-is-a-borrowed-concept) already matches the founding intent; at most it gets a cross-link to the founding-intent source.
- **D3:** the dependency mechanism and the meaning of body links are decided after the fitness review W1, with evidence.
- **D4:** standard Libraries live in a new top-level `libraries/` folder with one owner shard, versioned like `markitect-core`. `software-basics` is absorbed into the new Libraries and kept frozen only for existing fixtures.
- **D5:** the next comparison study (v0.3, the planned main study, and its follow-up; drafts not in the repository) waits at least for W1–W4: the fitness review, the guide to what a well-modeled project contains, the quality criteria and the first standard Library outline.
- The six workstreams W1–W6 are approved as scoped in the charter. The owner asked that W1 start now.

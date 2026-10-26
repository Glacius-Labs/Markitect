# Owner decision: the self-model pilot waits until Markitect is stable

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim; the session's proposal they answer is summarized and labeled. The recorded meaning is the recorder's and adds no decision.

The decision belongs to the [canonical model track](../../design/model-track-charter.md) ([DEC-074](../register.md#dec-074-canonical-model-track-parallel-to-v1)) and amends its pilot workstream W6.

## Context

Summary by the recorder: the session had proposed a W6 pilot that remodels Markitect's own self-model. It was based on a finding: Markitect describes its own architecture in three places that disagree. These are the self-model's 18 Packages, the code map `internal/tooling/architecture/codemap.yaml` (43 Go packages with layers, purposes and owning documents) and the responsibility imports in the architecture gate's `target.go`. Project, Workflow and Harness have no Package of their own in the self-model. The proposal was to make the model the single source, with the architecture gate reading dependency rules from the model and the code map generated from it.

## Owner statements (verbatim)

> Okay i dont know how we got 40 different code parts in markitect, this sounds kind of bad. Also i dont know if we should start developing markitect by using markitect, which is in development currently and not stable yet.

Context (summary by the recorder): the session answered that about 23 of the 43 Go packages are the product (13 core, 7 adapters, 3 running inside the agent containers) and about 20 are tooling and support around it. It offered a short review of whether that tooling is all needed, as a cleanup question separate from the model work. It agreed that Markitect's own build should not depend on Markitect while it is still changing, and proposed:
- defer the self-model pilot until Markitect is stable, after V1 acceptance and the studies;
- keep the self-model as it is, with CI's existing `markitect check` of the repository;
- keep Makerspace as the only W6 pilot;
- keep the three-places finding on record for later.

It asked: "Should I adjust the plan that way and tell the Coordinator?"

> Yes

## Recorded meaning

Recorder's interpretation:

- **W6 has one pilot, Makerspace.** The self-model pilot is deferred until Markitect is stable, after V1 acceptance and the studies, because Markitect's own build should not depend on Markitect while it is in development.
- **The self-model stays as it is.** Changes that keep it valid continue, such as the Core 1.2.0 migration of [DEC-075](../register.md#dec-075-kind-extension-and-one-core-120) and model updates that accompany product changes. CI's `markitect check` of the repository stays.
- **The three-places finding stays on record** as a known deviation for later.
- **The tooling review is not decided.** The owner's "Yes" answered the plan change; the offered review of the about 20 tooling packages is still open.

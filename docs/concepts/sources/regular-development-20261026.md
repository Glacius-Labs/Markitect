# Owner decision: Markitect is developed as a regular software project

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim; the session's questions are quoted and labeled. The recorded meaning is the recorder's and adds no decision.

## Context

Summary by the recorder: shortly before, the owner had deferred the self-model pilot of the canonical model track ([source](self-model-pilot-deferred-20261026.md)). Until now Markitect has been developed model-first with its own product: its repository holds a canonical self-model under `.markitect/`, registry shards that account for every file, and a generated readable view, and CI runs `markitect check` on the repository ([concept §5.5](../v1-concept.md#55-protected-foundation-and-self-model)).

## Owner statements (verbatim)

> We should definately clean up and consolidate . We should also have at least canonical docs.

The session asked (verbatim): "Is that what you mean by canonical docs, or do you mean cleaning up the human-written docs so each topic has exactly one canonical page?" and, after a note from the coordinator: "Do you want the self-model rewritten as the one canonical description of Markitect, without driving the build? Or one canonical hand-written page per topic? Or both?"

> I think we should develop markitect just regularly via agentic coding, not by creating a canonical model, which we currently develop. So we just should have documentation as in a regular software project. Later we might consider using Markitect for developing Markitect itself. So yes, i think we should change that

## Recorded meaning

Recorder's interpretation:

- **Markitect itself is developed like a regular software project, with agentic coding.** Its repository keeps no canonical self-model. The self-model under `.markitect/`, its registry shards, the generated view and the steps that depend on them (CI's self-check and view freshness, the model steps in `AGENTS.md`) are retired.
- **This concerns Markitect's own repository only.** The product is unchanged: target projects keep their model under `.markitect/`, with registry and file accounting.
- **Documentation is ordinary project documentation,** with one canonical page per topic. That is the recorder's reading of "at least canonical docs" together with "documentation as in a regular software project".
- **"Clean up and consolidate" stands** for the code and the documentation. Its scope is not yet defined.
- **Using Markitect for Markitect's own development is a later option,** not a planned step. The deferred self-model pilot of the canonical model track becomes that option.

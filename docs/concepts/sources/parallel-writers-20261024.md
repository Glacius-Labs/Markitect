# Owner decision: parallel writers where useful

Date: 24 October 2026. Source: the owner's direct message in a Claude Code session, while the assistant coordinated the V1 implementation through separate sessions with at most two writers at a time. The owner statement is verbatim; the translation and the recorder's notes are labeled.

## Owner statement (verbatim)

> Du musst dich übrigens nicht auf 2 parallele Writer beschränken wenn es sinnvoll ist, auch mehr zu nehmen.

English translation: "By the way, you don't have to limit yourself to 2 parallel writers; if it makes sense, take more."

## Recorded meaning

Recorder's interpretation: the fixed limit of two concurrent writing workstreams no longer applies. The coordinator runs as many writers in parallel as are actually useful and decoupled.

The other rules of concept §24 stay in force:

- each writer has its own worktree, an exact base and an exclusive file scope;
- shared contracts, registry shards, the self-model and generated files have one writer at a time;
- independent complete-diff review and exact-head integration apply as before.

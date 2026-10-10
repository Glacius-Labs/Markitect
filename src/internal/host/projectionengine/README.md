# Projection engine

This package evaluates a finite projection contract from explicit, normalized host facts. It does not load canonical sources, interpret their schemas, evaluate policy, run a projector or verifier, or write targets.

Callers bind a fixed model and snapshot using their digests, provide an exact source inventory and protected paths, and supply observed/desired target bytes and check evidence. Source keys are opaque exact identities: tuple serialization such as a JSON identity key may contain brackets, but it is never parsed as a glob. Contract source entries only match exact inventory keys.

Governance is host-owned. The engine receives only an explicit status and a failure flag, blocks when either says failure, and otherwise leaves governance semantics to the caller. The v0.13 compatibility adapter retains the historical typed PolicyResults and maps legacy semantic-model provenance into these facts.

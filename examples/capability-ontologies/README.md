# C1 typed ontology proof

Protocol freeze: 2026-10-06 before execution. Claim: the new Core structurally compiles five substantially different, project-owned typed ontologies through the public canonical YAML codecs and the same Core compiler pipeline. This is a source-level structural proof only; no external repository or Git fixture is needed.

Frozen schemas: software-architecture, delivery, workflow-responsibility, trading-card-combat, and filing-review. Each declares its own API version, Kind purposes, closed typed Properties and cardinalities. Definitions cover a component/dependency map, deployment and rollout, accountable work with roles, a card combat action, and a filing with review. References stay within their owning schema. These are separate project languages, not a universal ontology or a claim that sample business rules are correct.

Positive checks, frozen before execution:
- Decode every Schema and Definition with canonical.DecodeSchema / DecodeDefinition, then compile their complete set using core.Compile.
- Require all five schemas and every fixture Definition to compile without diagnostics, resolved reference edges, and selected Schema, Kind, Property and Definition purpose strings in the normalized Model.
- Require the normalized Review Definition to retain its typed `kindReference` value to the Filing Kind.
- Reverse Schema and Definition input order and require the same normalized model digest and edges. Provenance is derived from committed fixture bytes.

Negative checks, each through the same decode-then-compile pipeline:
- Missing referenced Definition: reference.unresolved.
- Reference value naming a Kind different from its typed target: reference.target-kind.
- Undeclared Definition Property: spec.unknown-property.
- List containing more values than a Property's declared maximum: property.cardinality.
- Value outside a declared enum: property.enum-value.
- Kind reference to a Kind outside the supplied Schemas: kind-reference.unresolved.
- Kind reference with a closed-shape violation: kind-reference.value.

PASS requires every positive and negative expectation. The trial does not test policy, source-code meaning, policy sufficiency, business correctness, human approval, deployment behavior, AI performance or benefits. No ontology-specific Core primitive is proposed.

Frozen fixture manifest: schemas and Definitions under this directory plus examples/capability_ontologies_test.go. The protocol, fixtures and tests are committed together before execution. The resulting commit identifies exact bytes. Append SHA-256 inputs, model digests, normalized counts and exact negative diagnostics to results.md after execution. Preserve failed or invalid runs.

Protocol revision 2 was frozen after the first committed execution. That run at `3c8a499` was FAIL: the unknown-Property mutation made the inline deployment spec invalid YAML, and the scalar-to-list cardinality mutation produced `property.type`, not the frozen expected `property.cardinality`. Both outcomes remain failed-trial evidence. Revision 2 uses a valid block-mapping unknown Property and tests list length above the declared maximum. These corrected proof inputs are committed before the next execution.

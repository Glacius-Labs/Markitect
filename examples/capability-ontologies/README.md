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

Protocol revision 2 was frozen after the first committed execution. That run at `3c8a499` was FAIL: the unknown-Property mutation made the inline deployment spec invalid YAML, and the scalar-to-list cardinality mutation produced `property.type`, not the frozen expected `property.cardinality`. These outcomes are preserved as narrative/transcript observations; the first run's raw stdout was not retained and has not been reconstructed. Revision 2 uses a valid block-mapping unknown Property and tests list length above the declared maximum. These corrected proof inputs are committed before the next execution.

## Protocol revision 3: executable source-alpha CLI proof (frozen before trials)

Baseline: production source commit `5c4b1689df94858b57604918238f92e11c4264ae`; proof setup is committed from that baseline before any canonical CLI model trial. The existing five Schema files, twelve Definition files, `examples/capability_ontologies_test.go`, and the `c1-8e1a430` helper-test evidence are unchanged. `canonical.yaml` lists those exact paths. Five adjacent schema-only Modules package the existing Schema bytes; each package lists only `schema.yaml` and has no projectors. The Modules add no vocabulary and the Definitions are not altered. Package pins will be obtained with the public `canonical --action modules` preview, recorded, and then fixed in `canonical.yaml` before model trials.

The frozen claim remains narrow: the public canonical YAML configuration and CLI can structurally compile five distinct, project-owned typed ontologies through Core's existing Schema/Kind/Property/Definition model. No AI, external business repository, generated output, source-code interpretation, or domain-specific Core primitive is involved.

Expected outcomes fixed before the CLI trials:

- Positive combined model: exactly 5 Schemas, 12 Definitions, and 7 resolved reference Edges, with no structural diagnostics.
- Independent single-ontology snapshots: one Schema each; counts are architecture 3/2, delivery 2/1, responsibility 3/2, card combat 2/1, filing/review 2/1 (Definitions/Edges).
- Reordering module/schema and Definition entries without changing bytes yields the same normalized model digest and sorted Edges.
- Four isolated negative snapshots must fail structurally: missing referenced Definition (`reference.unresolved`), wrong reference target Kind (`reference.target-kind`), a list above a Property's maximum (`property.cardinality`), and a kindReference to a missing Kind (`kind-reference.unresolved`). Each negative starts from a fresh repository snapshot and changes only the described input bytes. Expected CLI exit is the structural-diagnostic exit; if the frozen executable produces another exit/status pairing, record the observed behavior as a failure rather than adjusting the expectation.

Execution protocol: after this protocol and package manifests are committed, use CLI module preview to compute exact pins and commit the pinned Source configuration. Build the CLI from public source commit `216e101baa2021fb6bf091206b323407aa471b26`; retain a build receipt binding binary digest, source commit, Go version, and build argv. Run the pinned configuration against separate Git repositories with full commit IDs for the combined positive, reordered positive, each isolated ontology, and each negative case. Preserve every attempt's exact stdout, stderr, exit code, source/config/package/input SHA-256 values, CLI binary digest, and revision. Never rewrite the existing `c1-8e1a430` evidence.

Modeling-friction review is descriptive. Record the positive use of typed closed Properties, cardinality, enum and reference shapes and the negative semantics actually observed. Identify the absence of shared cross-ontology constraints and business-invariant vocabulary in these fixtures; discuss kindReference only as the frozen missing-Kind check demonstrates. Do not infer ontology completeness, business correctness, validation sufficiency, operator benefit, or general expressiveness from this finite sample.

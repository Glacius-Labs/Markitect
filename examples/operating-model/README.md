# Operating-model proof fixture

This fixture prepares a small, project-owned canonical source for three exact target roots:

- src/Orders/ represents the Orders leaf Definition.
- src/Billing/ represents the Billing leaf Definition.
- docs/represented/ represents ProductComposition and explicitly selects both leaf Definitions.

The Orders and Billing operations are pure and independently defined. Orders accepts only positive quantities and nonnegative unit prices, and returns a checked total in cents. Billing sums nonnegative outstanding amounts with checked signed 64-bit addition. The product composition passes an accepted Orders total to the Billing query beside prior open amounts.

The Schema Module owns only this Project's vocabulary. The Foundation Schema and the .NET and Markdown Projection Modules are separate, explicitly pinned package inputs in canonical.yaml. ProjectionPolicies carry the target-specific public API or representation requirements. The .NET Module restricts target roots and leaves filenames to the Executor.

The project-owned checks under checks/ use .NET 8 and no external packages. The leaf checks compile the C# files in their explicit target roots and run fixed behavior cases. The composition check compiles both roots and calls the Orders operation followed by Billing. The Markdown check reads only the explicitly selected docs/represented/ target root. These checks provide bounded implementation evidence; they do not establish that an agent followed intent, that the verifier was independent, or that an owner accepted the result.

## Frozen operating-model proof protocol

Protocol identity: operating-model-proof/v1.

A future C3/C4/C5 result requires a fresh Executor invocation and a separate fresh Verifier invocation bound to the same immutable canonical source and exact candidate. The Verifier receives the independently selected canonical obligations, actual candidate/materialized artifacts, and applicable fixed project checks. The Executor transcript or its authored tests are not verifier evidence. The fresh-run command/configuration and exact input bindings must be recorded with the result.

The historical protocol-v1 checkpoint remains C5 FAIL. This prepared fixture, its unit/compiler checks, and its canonical shape result are setup evidence only; they are not an Executor/Verifier experiment or a replacement proof. No private adopter source is included, and no technical result records owner acceptance.

## Separately versioned documentation alignment

`canonical.yaml` and its original checks remain the original C4 experiment.
The fresh v5 run exposed a mismatch: the Documentation check requires the two
public API names, while the three selected Markdown subjects/policies do not
contain those names. That failure is retained as evidence, not reclassified as a
missing semantic renderer or a passing result.

`canonical-documentation-aligned.yaml` is a new selected source. Its Product
Markdown Projection additionally selects the existing Orders and Billing .NET
ProjectionPolicy Definitions as documentation subjects, under the explicit
`representation-policy-markdown` policy. Those exact existing Definitions remain
the only owners of the public API contracts; their bytes, the business Definitions,
module pins and all four check commands/implementations are unchanged. Markdown
consumes canonical policy, never a sibling Module's generated source files.

The fixture compile/render test establishes that the selected contract now
contains the required API names. It does not establish agent behavior, fresh
verification, owner acceptance or convergence. A new experiment must freeze this
source separately and preserve the original failure.

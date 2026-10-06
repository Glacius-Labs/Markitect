# Goal-led modeling proposal

This design adds a small Host API for the first greenfield modeling step: a caller supplies goal text and an explicit bounded set of Module package bytes, receives recommendations tied to exact package pins, selects recommendation IDs, and receives a structurally compiled Definition proposal.

The API never scans a project, searches an installed catalog, downloads or installs Modules, writes canonical files, creates a canonical source, or adopts a proposal. The caller chooses which already available packages cross the API boundary. The implementation accepts at most 64 packages and 4 MiB of goal, Module manifest, and Schema bytes; the goal is limited to 8 KiB. Narrowing the catalog remains the caller's privacy responsibility.

## Flow

1. `RecommendGoalModules` checks the supplied packages and computes each immutable package digest. It starts one fresh `RoleInfer` process with the exact goal, supplied Module IDs and pins, manifests, and declared Schema bytes. The closed response shape contains recommendation IDs with a basis and uncertainty. The Host rejects unknown and duplicate IDs and attaches the package name, version, and exact digest itself.
2. A caller makes an explicit selection by passing recommendation IDs to `ProposeGoalModel`. This call checks that the recommendation result still binds the exact goal and package bytes, then activates only the selected exact pins. Required Module dependencies must also be selected as recommendations; activation does not silently add them.
3. The Host sends the selected Core Schemas and exact selected Module pins to a second fresh `RoleInfer` process. Its closed response contains Definition values, a summary, and uncertainty. It cannot add Schemas or Module pins.
4. The Host decodes every Definition through the same closed Definition decoder as canonical source and compiles the full candidate with the selected Schemas using `core.Compile`. Unknown Properties, invalid values, unresolved References, and other Core diagnostics leave the result invalid.

Both results carry the exact goal/catalog bindings and agent-run receipts. The model proposal also binds selected recommendation IDs, resolved pins, schema digest, candidate byte digest, and its runner input digest. An optional `DecisionReferenceClaim` is caller-supplied text; it is carried as a claim and is never authenticated or used as authorization.

## Result boundary

Successful output is a `proposed` `CompiledCandidate`. It remains noncanonical even when Core reports no structural diagnostics. Compilation does not establish that the goal is complete, the Schema is suitable, the proposal is good product design, a human approved it, or any files should be changed. This API does not reconcile or verify a project. Those later steps require an explicit owner decision and separately supplied project inputs.

The two helper-process tests prove request/response protocol handling, supplied-ID restriction, fresh run receipt binding, and rejection of a Definition that violates the selected closed Schema. They do not prove provider quality or acceptance of C12's full initialize-to-verify workflow.

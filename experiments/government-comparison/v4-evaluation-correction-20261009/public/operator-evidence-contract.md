# Prospective operator evidence contract

This version clarifies measurement of existing staged HTTP requirements. It
introduces no new application routes, actor artifacts or rubric weights.
Requirements and outcomes apply equally to all three methods and both starting
conditions. Method conformance and Design migration remain separate. Conventional
has no Design-artifact duty. Test counts are mechanics observations, not quality.

## Candidate and process continuity

Freeze actual candidate commits and retain their actual binaries and build
receipts. For restart and compatibility checks the operator supplies a private
receipt containing cellId, candidateCommit, binaryPath, binarySha256,
databasePath, databaseInstanceId, processInstanceId, operatorTracePath and
operatorTraceSha256. Paths are absolute. Binary and trace digests must match actual
retained files. The trace binds the invocation, configured DB, process start/stop
and candidate artifact; an independent assessor must inspect that binding.
The receipt additionally binds operatorId, validationPath and validationSha256.
That retained JSON validation report records decision VERIFIED, a distinct
validatorId and bindings for cellId, candidateCommit, binarySha256, databasePath,
databaseInstanceId, processInstanceId and operatorTraceSha256. The evaluator
checks these bindings and withholds a result without them. This is still a
recorded independent observation, not cryptographic proof that its author is
honest. Validation work and its actual tools/compute must be charged to common
evaluation preparation; it does not imply human acceptance.
databaseInstanceId identifies one initialized database lineage and is preserved
through its legitimate writes; it is not a constant database content hash.
Different processInstanceIds must represent an actual stop/start, not a label
change. This evidence is operator supplied; checking its hashes and fields alone
does not establish the truth of a process or SQLite connection claim. No fixture,
synthetic binary, invented old version or path-only assertion qualifies.

For Task 3: capture state under its frozen candidate and then restart its same
binary on the same database. Both phases use the same cell, database lineage,
resolved DB path, candidate and binary, with distinct processes. A before phase
is EVIDENCE_CAPTURED; only complete verified after observations may PASS.
Task 4 likewise captures both terminal lifecycle states and inventory, restarts
its frozen binary on that same database and checks terminal readback, repeats and
unchanged inventory. The initial phase alone does not establish durability.

For Task 5: before releasing/implementing Task 5, keep the actual frozen Task-4
candidate binary from that same cell. Establish a valid accepted prior-limit
order using that binary on a fresh isolated DB and capture the private checkpoint
and receipt. Stop it, run the genuine Task-5 binary against that same DB, and
check prior-order readback, canonical replay, conflicting reuse and lifecycle
preservation. Then check valid/invalid requests, cross-route idempotency,
reservations and current boundary behavior on both routes. Stop/restart the same
Task-5 binary on the same DB and verify persistence/retries for both routes.
Retain all three receipts and full raw evidence. Do not reconstruct a supposed
Task-4 state from Task-5 code or use a foreign cell/version. Task-4 and Task-5
candidate commits differ. Retain both actual binary bindings; the bytes may
coincide if the changed rule is supplied by configuration, whose provenance must
then be bound in the invocation trace. Failure to retain a real predecessor is
NOT RUN, not an exemption, product PASS or method advantage.

No evaluator starts these services automatically. The ordinary operator process
steps require their own future actual-cell authorization and charged budget.

## Applicability and attribution

Every record binds a released requirement ID, task cutoff, candidate, condition
and evaluator digest. Cases outside their released cutoff, missing continuity,
unknown applicability and missing prerequisites produce NOT RUN. An incomplete
multi-phase checkpoint produces EVIDENCE_CAPTURED, never PASS.

Only an explicit failed assertion of a declared observable invariant is FAIL /
product-invariant. Transport/unavailable infrastructure is NOT RUN /
infrastructure. Evaluator exceptions, corrupted checkpoints or inconsistent
harness data are EVALUATION ERROR / evaluator-harness, never product failure or
PASS. Malformed JSON returned by the running product on a required JSON route
violates its declared response contract; a file decoding failure in evaluation
data is an evaluator error. Invalid invocations produce structured NOT RUN.
If environmental provenance cannot be distinguished, report uncertainty and do
not award a product result. Prior hard failures remain visible; no method wins
from missing or inapplicable evidence. Offline fixture passes prove only this
classification/checking mechanics and cannot populate product quality scores.

Task 6 remains an unexecuted candidate-specific equivalence plan: actual Task-5
freeze, actual invariant/neighbor observations and independent cross-cell mapping
are required before the common repair card. No mutation or equivalence run occurs
in this preparation. Common actual model/tool/runtime profile, clean Design
handoff and a finite actual-cell grant remain prerequisites to future trials.

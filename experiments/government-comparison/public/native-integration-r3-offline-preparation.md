# R3 offline adapter contract repair

Package `s1-offline-adapter-contract-repair-r3-preparation` continues accepted
Scientist delivery `419775ab0ea5f421655ab3c8bd97e286ffbb6e0a` on
`codex/government-scientist`. It authorizes offline harness repairs, temporary
test stores, focused regression checks and independent review only. No native
product, queue, controller, real delegate, model, provider, metadata tree or
study cell was started. No new execution quota was granted.

## Corrected contracts

Government's frozen R2 authorization contains `delegate.argv` and
`commandDigest`, but no `delegate.command` in any of its three slots. The
existing launch guard therefore rejected the invocation before admission. Both
fixture builders now emit the explicit absolute executable; static preflight
requires `delegate.command == delegate.argv[0]`. Digest and runtime-file
bindings remain required, and the launch guard remains as a second check.
Missing or different commands fail before delegate reservation.

Classic's pinned native `Request.scopeIds` are JSON-encoded canonical Definition
identities. They are not the assurance labels `commerce-dotnet` and
`commerce-markdown`. The resolver now binds the configured role and Projection
to its exact Definition identities, the embedded model context and the matching
authorization slot. Missing or different scopes, subjects, context Projection
or authorization identity fail closed. The preflight constructs those same
native identities for all three configured Classic slots.

The producer evidence is read from exact source
`7dbd599c81540c8203a1b7f83afbc335174f4f1f`: DTO, Executor/Verifier request
construction, canonical context builder, normalization and stdin serialization.
Both model scopes and Definitions are sorted by canonical identity before
serialization. The [source-bound contract](../evidence/native-integration/run-3-preparation/classic-agentexec-source-contract.json)
records blob hashes and lines. No saved full raw Classic Invocation was found
in the retained R1/R2 Classic artifacts; receipts retain digests rather than
stdin. The [before-fix replay](../evidence/native-integration/run-3-preparation/classic-resolver-original-failure-native-shape.json)
is explicitly synthetic and does not reconstruct missing historical bytes.
The initial exploratory noncanonical vector remains labelled separately.

## Validation and boundaries

The [focused gate](../evidence/native-integration/run-3-preparation/focused-tests.log)
passes 18 tests: Classic contract, fixture binding and review guards; explicit
Government command/digest binding; three-slot Classic preflight; and mocked
controller accounting and failure closure, post-Apply verifier binding, and the
six-role integration chain. Existing tests that launch real
synthetic delegates were deliberately excluded. Temporary Git setup and SQLite
stores are offline fixtures, not new native run evidence or issued authority.

The [six-role contract evidence](../evidence/native-integration/run-3-preparation/cross-role-contract-vectors.json)
and its [test log](../evidence/native-integration/run-3-preparation/cross-role-tests.log)
exercise real `Authority`, bootstrap bytes/digests and per-Invocation
`load_context`, both resolvers, command/runtime pins, and durable temporary
reservation before the mocked delegate effect. Every role reaches success and
nonzero-process failure translation: six successes and six failures. The
source-grant admission and Classic product binding are patched; synthetic
Authority documents grant no actual execution authority. The predecessor
Government `run_role` is also loaded from Git and reproduces the original
commandless rejection before reservation. Government producer identities are
bound to `04e225d...:internal/host/government/execution/actors.go`; its scopes
derive from retained actor/report/plan records. Context, run IDs, nonces and
digest placeholders remain explicitly synthetic.

The [independent review](native-integration-r3-offline-review.md) assesses these
boundaries separately. Offline success does not establish
native lifecycle success, semantic quality, verifier independence, human
acceptance, comparison results or S1 runner readiness.

The [new R2 errata](native-integration-r2-government-errata.md) binds the final
five-row native-start ledger to `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5`.
The old four-row statement cited that five-row hash; no exact four-row database
image survives. Government's process receipt is durably stored on its outer
`controller_runs` row and Result. Its zero `controller_processes` rows are not
a missing receipt: Classic uses that table for action processes. A future
unified process-count view must normalize these sources and avoid double
counting. No historical ledger or collector-source repair was necessary.

## Unchanged candidates and remaining work

Government source is `04e225d5caee78c2a198607143863fca1e829750`, binary SHA-256
`12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`.
Classic held source is `c91363b7ac4decbe87212ff0f588b5451581a152` (v0.14.1),
runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`, binary SHA-256
`2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`.
These candidates were read, not executed or modified.

Both R2 native cases remain incomplete. Government needs one successful queue
and bounded same-queue replay. Classic still needs successful Execute, exact
independent review, guarded Apply, fresh Verify, Audit and stale Apply replay.
No old failure is rescored by this package. Raw provider telemetry remains
unknown; deterministic fixture accounting is not provider observation.

The [finite allocation proposal](../evidence/native-integration/run-3-preparation/next-native-allocation-proposal.json)
is **not authorized and not runnable as issued authority**. Its smallest first
checkpoint is one Government queue and one Classic Execute: two additional
native starts, 300 reserved session seconds. The complete conditional lifecycle
requests at most Government +2 starts/+6 wrappers/+6 fixed protocol delegates/
300 seconds and Classic +5/+6/+6/750 seconds. Real Actors, providers, metadata
trees and study cells remain zero. Proposed cumulative ceilings are 12 native
starts, 15 wrappers and 1,800 reserved session seconds.

Before any future execution, Overseer must issue a new finite grant bound to
its exact Coordinator snapshot. The current admission code recognizes the
closed R2 grant; a narrow reviewed update for the newly issued grant, fresh
disposable Request/runtime/role bindings, current source hashes and immutable
preflight freeze are still required. Old runtime snapshots and grants must not
be refreshed or reused. Preserve the original native-start ledger and all
historical controller bookings. No additional inspect is proposed unless prior
Constitution/config/source equivalence cannot be established. Stop that
product at the first failure, missing prerequisite, unexpected invocation or
deadline; do not repair and relaunch inside the proposed case.

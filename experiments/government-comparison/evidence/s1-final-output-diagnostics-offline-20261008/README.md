# First protocol-rejection observation (offline)

## Proven source behavior and uncertainty

Accepted base: `900b9e7eccda27276272b97cd8c0b14c117b1cc6`. The sealed client's final drain calls `consume(raw)` without an expected response and catches every exception. If no earlier reason exists, it sets `unexpected-final-output`, continues draining, and discards frame-specific method and refusal details. This is a demonstrated observation gap. No offline evidence identifies the discarded actual frame or establishes an incorrect acceptance decision. Accepted event counts do not identify the rejected frame. No private/raw/transcript recovery occurred.

## Additive observation

`protocol_failure.py` has no disk, network, process, environment or clock API. It retains only the first rejection, copies only fixed output literals and exactly catalog-listed public method names, and returns a deep copy. The 94-name catalog is the union of the existing archived 83 ServerNotification and 11 ServerRequest members; it is a naming catalog, not an admission allowlist. Names outside this catalog remain `unlisted`, even if another release exposes them. Refusal code version 1 contains 28 narrow static source literals. Unknown exceptions/messages/codes map to `unknown`; no `str(exc)` is used by this helper. Parsed frame/params/IDs/scalars are never retained in the metadata object.

The client records rejected `consume` frames before re-raising the same exception. It also records a rejected full Thread response gate after a successful response-envelope/schema check, using only the known fixed result-response category. It does not retain that response. `firstProtocolFailure` is null when no frame rejection occurs. It means first instrumented protocol-frame rejection, not first arbitrary configuration, cleanup, filesystem or process error; existing `stopReason` remains the primary session decision.

Allowed stages are declared before use. For undecodable JSON, the category is `undecoded` and method absent; no partial value is recovered. The first record survives later errors and returned-snapshot mutation. The fixed shape has a conservative upper bound of 350 bytes at the receipt's actual two-space nesting/indentation (328 bytes standalone; 300 with default compact-line spacing), checked statically against 2048 bytes. Capture occurs synchronously before the existing catch discards the frame; one snapshot is added after cleanup. Existing queue drain, counters, lifecycle, strict response/notification/schema/model/policy guards, response-boundary and four-field comparison receipts remain unchanged. Historical unknown terminal failure remains a stopped run.

## Offline boundary and validation

This directory is additive. Copied gate, schema contract, public method catalog and profile are byte-identical to the sealed native-cwd packet. Source bindings identify both these inputs and the derivative. No request, grant, freeze, reservation or native execution artifacts are created. The copied client retains prior authority checks and additionally has a disabled direct entrypoint (`SystemExit(2)`); it is an offline source/test fixture, not a runnable authorization package.

Exactly six new focused synthetic cases are permitted. Prior suites are not rerun. Review and test results, correction history, timings and final bindings are recorded separately. Six cases passed once (unittest 0.038s; command wall 0.2452836s), with no failures. One conservative pre-test correction batch changed only test scope/fixture/assertions; no product/client/helper corrections. The mocked session uses the real pinned full Thread/start response validator; other metadata and notification schema checks are selective fixture mocks. Its drain/correlation observations therefore do not independently validate notification schema conformance. Synthetic checks cannot identify the historical discarded frame or prove actual runtime success.

## Finite next step

If the Overseer issues a fresh concrete finite grant within the readiness window, use this metadata at the same strict thread-only boundary to observe the first rejected frame; do not expand policy or admission from a catalog name. This packet grants no execution. At most two future actual readiness attempts may be considered, each separately granted, before the canonical window ends at 2026-10-08T20:42:00Z. If S1 remains unavailable, close the diagnosis block with measured limitations and effort; all six study cells remain NOT RUN. No automatic follow-on exists.

## Preserved effort baseline

No actual app-server trees, reservations, Actor tasks, turns, tools, CLI metadata calls, web requests or study cells are added. Actual native trees remain 11; historical Actor reservations remain 6 including 5 model attempts, CLI metadata calls 8, known historical tokens 53331 with total/current unknown, native 15/16/13/2250. The offline work establishes diagnostic mechanics only, not S1, serving, OS enforcement, billing or method quality.

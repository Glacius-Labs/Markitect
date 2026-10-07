# Disabled-status metadata preflight

Preflight disposition for the single separately authorized disabled-status policy-read allocation. This is a static safety review of the frozen client and its pure test receipt; it does not establish that the app-server will return usable configuration, that a policy is effective, or that any Actor or model is ready.

## Reviewed snapshot

The reviewed request and sources are bound by these SHA-256 digests:

| Artifact | SHA-256 |
| --- | --- |
| `request.json` | `eaeaa66ecc1f4ba1620997324cfc83d2df181da6114523e1d63b92f1cc69ccbe` |
| `classified-client.py` | `95c8486ad8962e14ad0c2da2989adcf154fa28f51d19c1fe983c532d1a885926` |
| `frozen-method-enums.json` | `6553df9ac4a37d11402728992ed5e684ea379fbadeee790700867ae4542594cf` |
| `test-classification.py` | `7bd7d63687936bb727d448f08a139b00d3752762a4093602b4d7bf32fb6dd335` |
| `test-classification.log` | `75fc8bb5bb235c63dc6b361beeb69d67614cf2c2e3734e66a7b43a29cf2e8616` |
| `test-classification-result.json` | `aebb8605a3cfccf7d4226ee68ac324f51331bb3145c8aaa51304c82e3bb4580d` |
| `launch-once.py` | `09b98939771514378946df60bbda3af4c73e2d589446235d15d210726aa55ff2` |

The test receipt reports 22/22 pure tests passing and no client main, subprocess, app-server, model, or Actor execution. The launcher binds the request, client, enum source, frozen source files, Python executable, and reservation before claiming its sole start. The existing process wrapper caps the complete process tree at 38 seconds, below the authorized 60-second outer and 50-second inner ceilings; the client has a 33-second active deadline and no retry path.

## Safety review

The client validates the exact frozen `remoteControl/status/changed` notification shape and pins the accepted status to the literal `disabled`, as well as pinning the frozen four-value enum. It preserves only the method, notification class, and status value; installation, environment, server, and other parameters are not copied into the event. A schema-valid `disabled` event is discarded without a response. `connecting`, `connected`, `errored`, malformed payloads, unknown messages, and every server request stop the session without answering the server request. The global notification limit is 16, including this notification and the four previously allowed warning classes; the warning policy remains provisional.

The request and authorization copy agree on the single allocation, exact four-message RPC sequence (`initialize`, `initialized`, one `config/read`, one `configRequirements/read`), one process tree, zero retries, zero Actor starts, and the historical ledger values: two earlier app-server trees, six CLI metadata calls, five Actor starts, 53,331 known Actor input-plus-output tokens, and an unknown historical token total. The history is preserved as unknown; this grant adds one bounded policy-read session and does not reset prior usage. No account, model, authentication, thread, turn, command, or filesystem RPC is authorized.

## Disposition and limits

I found no remaining static preflight blocker in this hash-bound snapshot. This is clearance of the request/client mechanics for the one authorized metadata session only. It is not evidence of a completed session, effective permissions, read-only enforcement, absence of inherited configuration, privacy of native internals, or readiness for an Actor or model call. Any unexpected or malformed notification stops the client; a stop or warning may leave the result incomplete or provisional.

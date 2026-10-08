# Thread-start response-field schema review

The archived schema ZIP is SHA-256 `3157e8a55329cf4e5346676c3ef0c308402d2420f8f916d6a5b7e0e9c84356ad`. Its pinned `v2/ThreadStartParams.json` member is SHA-256 `80a40a7fac15b4bf70efb7f893fb353acc0a0d30c68f54aee4f01923deca85de`; `v2/ThreadStartResponse.json` is `bcb709ddaed237632beda448e79621d3fbfcc449fee661221088fa16d8b55914`.

The four fields have related but not identical schema domains:

| Field | `thread/start` request schema | Thread-start response schema | Consequence for the unchanged equality gate |
|---|---|---|---|
| `cwd` | Optional; string or null | Required string reference to `AbsolutePathBuf` | The pinned request supplies a concrete path string. The schema description says absolute and normalized, but the archived JSON schema expresses only `type: string`; validation does not enforce absoluteness, normalization, canonicalization, or filesystem existence. Exact equality is stricter than schema validity and can reject alternate separators or case. |
| `model` | Optional; string or null | Required string | Both schemas permit broad string values. The pinned request supplies a concrete model identifier; exact equality rejects any different returned spelling or alias. The schema does not define alias equivalence. |
| `modelProvider` | Optional; string or null | Required string | As with model, both sides are broad strings and the request supplies one concrete provider identifier. Schema validity alone does not establish that another identifier is equivalent. |
| `approvalPolicy` | Optional; null, one of `untrusted`, `on-request`, `never`, or a constrained granular object | Required `AskForApproval`: one of those strings or a constrained granular object | The pinned request uses `never`. The equality gate accepts that exact value and rejects other schema-valid policies or objects. This is an intentional exact-policy constraint, not a schema mismatch. |

The profile binds concrete string values for all four fields. `ThreadGate.accept_response` first validates the full response against the pinned response schema, then requires equality with those four request values. Therefore a response can be schema-valid while failing the stricter identity/policy gate. Based on the recorded source path through schema validation for the prior diagnostic, the three required response fields `cwd`, `model`, and `modelProvider` must have been JSON strings; `approvalPolicy` could have been a string or the accepted granular object. This type-only inference does not reveal any actual value, equality result, or cwd lexical-comparison result. No particular mismatching field is inferred.

A lexical cwd comparison may be useful to explain whether two strings differ only by Windows slash direction and case. That comparison must remain diagnostic metadata only: the equality gate stays unchanged, and no filesystem lookup, path resolution, dot-segment cleanup, trailing-separator normalization, or acceptance rule follows from lexical similarity. Model, provider, and approval comparisons likewise must reveal only type/equality outcomes, never raw values.

This is schema and source-semantics analysis only. It does not prove why the prior response failed, whether any alternate field value is semantically equivalent, or what a future candidate would return.

# Independent R3 source preflight review

**Disposition: no source-level blocker found for the bounded R3 attempt.** This review covers the prospective client/profile and the reported five focused offline integration cases only. It is not a request/freeze binding review and does not authorize a start.

The local protocol archive is pinned at SHA-256 `cd24042eec4696f6b73368cfc6ce930726f3e10c28bae63cdbd1b2f502ff1d97`. Its `v2/ConfigRequirementsReadResponse.json` defines `requirements` as `ConfigRequirements|null`, with null described as no configured requirements. `ConfigRequirements` has no required properties; its optional `featureRequirements` property accepts `object|null`, and each additional property must be boolean. These schema facts support treating an actual null requirements value and an actual object with the optional field omitted, null, or empty as valid reports of no configured additional feature constraint. They do not make a missing whole `requirements` response property sufficient evidence.

The R3 client retains the accepted adapter unchanged (SHA-256 `985ff1d8f8870de4ac697e030c2abeb543d1bc3e74f475968c5e1097be865550`) and calls its real sanitizer and assessment paths. The precheck is explicitly labeled `config-precheck-satisfied-awaiting-requirements`; it uses a placeholder only to evaluate the independent config gates. A final positive result requires the exact four sent RPCs, responses with IDs 0/1/2, a `configRequirements/read` observation marked after response-envelope validation, and a positive assessment of that sanitized response. The pinned consumer rejects a missing whole requirements field; unknown present feature names, malformed/nonboolean maps, known conflicts, and failures in other gates remain nonpositive. No feature origin is inferred from a matching value.

The reviewed route remains the separate R3 grant: the same pinned 0.160.1 executable, exact 17 prior configuration pairs, `app-server --stdio --strict-config`, and existing public cwd/README. The client retains the one-shot Job-gated worker and fail-closed process, frame, notification, and time bounds (33 seconds active RPC, 38 seconds process, 50 seconds controller, 60 seconds outer receipt). The grant permits one additional metadata tree only; it permits no Actor, thread, turn, model call, product, wrapper, delegate, study cell, additional CLI metadata call, or historical-ledger edit. R2 remains terminally preparation-blocked; R3 is a separate grant, not a revival.

The five focused integration tests passed in 0.09 seconds per the pinned offline receipt. They use synthetic JSON-RPC envelopes through the real sanitizer and assessor, including actual null, optional omitted/null/empty/compatible values, missing whole field or final response/provenance, malformed values, unknown names, known conflicts, and an independent origin failure. They are not app-server observations, and the unchanged parser/product suites were not rerun.

Reviewed source/evidence pins:

- `client.py`: `29d2c97a7c34ff3c580789a777967051221339bd6da3a39b492f729271b7b223`
- `profile.json`: `daf3f2b775e73cb194f48f8731e73f82a93ca0b7f2976b980ee7ed45110fb416`
- `authorization-grant.json`: `d2d49ccda7f07c9f60fa641213373aa5b28b8cf7006c37c2df47463cc2c68c48`
- `test_client.py`: `b8005863ef91e94d066cc0373db1303f4e286108c27510792f3b43b2a02973dc`
- focused test log: `506294bd547e0431fe433484e6a1be02675d67e2670ce24d9618c983696d49b5`
- protocol basis: `565b2dc4b9b3ea23430d4d28f4a0223d61232bad6e597d453934339a4300b1fe`

The offline receipt records zero process starts and zero ledger writes. Before any attempt, the source commit, fresh request/freeze, reservation, live authority, and clean-freeze binding still require a separate review. No actual metadata result, policy enforcement, active permission, tool capability, serving identity, historical cause, or S1 readiness is established by this preflight.

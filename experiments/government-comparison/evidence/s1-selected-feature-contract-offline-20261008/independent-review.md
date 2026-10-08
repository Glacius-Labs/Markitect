# Independent review: selected-feature contract correction

**Disposition: no material defect found in the bounded offline correction.** This is a pure parser/assessment adapter; it does not change a client, profile, transport, grant, historical receipt, or runtime. I performed a read-only review and ran no tests, process, metadata request, or ledger operation.

The adapter pins and imports the preserved metadata client, applies its warning/provisional scan before projection, and uses the preserved sanitizer/assessment for unaffected fields. It removes `features` from the raw top-level config and each layer before invoking the old whole-map boolean validator, then derives only the eight requested feature fields with explicit missing/null/present-boolean/invalid states and fixed codes. The output labels the scope and marks the remainder not evaluated; it does not persist extra feature names or values. Resource bounds cover the feature container size and the existing raw/frame limits remain outside this pure module.

Managed `featureRequirements` use a separate schema-wide map path: the container and every value must be valid booleans; selected known conflicts are reported; all names outside the eight requested switches remain unresolved and block a positive assessment. That is intentionally conservative because the pinned schema allows arbitrary boolean property names and defines no name enum. The adapter does not mistake the old sanitizer’s `FLAGS` allowlist for a schema-wide name declaration. Unknown names are not retained in output. The real archived sanitizer and assessor remain in the chain for origin/layer, model, legacy sandbox, Windows and other managed-requirement checks. Only fully validated derived selected booleans are normalized for that unchanged assessment.

The tests use synthetic inputs through the actual preserved `sanitize_config_read`, `sanitize_requirements`, and `assess` calls, rather than hand-constructed positive sanitized states. The retained offline receipt reports 10/10 focused tests passing, including per-field type/presence variants, unrelated top/layer values and origins, typed and unknown requirements, conflicting known requirements, warning/origin gates, and the historical invalid receipt remaining insufficient. This exercises the parser contract only; it does not identify the discarded raw member in the actual attempt. The old result remains terminal `insufficient` and cannot be upgraded from synthetic reconstructions.

Reviewed bindings:

- Adapter source SHA-256: `985ff1d8f8870de4ac697e030c2abeb543d1bc3e74f475968c5e1097be865550`
- Focused test source SHA-256: `5c027d643a404c527f534c3d44b7a37d166a188ee71f801ae5ead45661329e49`
- Focused test log SHA-256: `7d3d5d29c39c658a09d4852a5843e03883f1948f22292b67e17e0cf8aa0504ba`
- Pinned schema/source-evidence SHA-256: `adcc6347fa42cd05ee421175f6ba69ba5cb791d98e8b37c67a85b575c0bff08a`
- Offline validation receipt SHA-256: `2705759b27d165f2a36e864120b733368ce530259936499d0a699e683b734709`

No live re-read, new metadata operation, retry, diagnosis, or S1 readiness claim follows from this adapter. The actual feature member/shape remains unknown, and S1 remains open.

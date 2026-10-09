# Native policy-preservation source handoff

Source `bcd614a3bfcd335e0ca18850993742057103c092` is clean and pushed on `codex/model-first-operations`; PR #89 remains draft.
The actual adapter no longer passes `--ignore-user-config` or `--ignore-rules`.
Normal CLI policy remains enabled; explicit read-only/tool restrictions, finite role timeouts,
Luna High and invocation-bound response validation remain in place.

Both allowed targeted test batches passed: 41 adapter tests and focused onboarding/help regressions.
The sole fresh Luna-High source review completed in 49.820 seconds;
its low documentation finding is repaired. These tests used no actual provider call.
The one production build passed. Its SHA-256 is `abf877849379900cf2f1cf6c19f3d5d1e67a134592ac810cedc437191e9a5f29`; the Go binary is byte-identical
because the source delta changes only the Python adapter, its tests and documentation.
Baseline source-2b hosted CI passed; replacement run 37914238147 is in progress at this checkpoint.

Replacement adapter SHA-256: `1284ea4e187cf68ab8a9c15062366cadc750d63a9deb5b833214ec05bcc56ba6`.
Replacement runtime SHA-256: `c3d4204490a9d1d5230cc151f99a63dcb8357bcc425022d040c48a0888afc44e`.
`runtime-replacement.yaml` changes only the adapter path and digest for both roles.
It is a prepared configuration artifact and has not been installed or activated.
The six pins are checked against the actual existing file bytes without launching Python/Codex roles.
The schema is copied from the unchanged schema implementation's retained source-2b output.

The old Roombook installation and immutable evidence remain unchanged; its policy-suppressing
adapter must not be used for a new native start. No new installation, provider/transport probe,
product job or trial was started. Three prior product jobs remain closed with zero inner starts.
Shared study counters, deadlines and any future allocation belong to the external coordinator.
The sixth cell remains unallocated; this source correction provides no readiness or release claim.

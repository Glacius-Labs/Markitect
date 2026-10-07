# Disabled-status metadata post-run review

This post-run review checks the one reserved policy-read session against its sanitized result and process receipts. It does not alter or extend the immutable preflight review.

## Binding and one-time execution

The run's `freeze.json` SHA-256 is `03406bb50e51562a8b0c73fba675f969b0002bc583c0aee6000b8e3fb3e5898a`; `reservation.json` SHA-256 is `fe353b4c0768899d05a0b20431c4f76d9370c7ff975ea499c61a51ef9372623f`. Reservation binds the freeze and the preflight review SHA-256 `ce09e57c865bda015e63f3b36e4cc19eca0727c038f8f77c89aaafc45ec4a134`. The one-time launch claim and start marker bind the same reservation. Every file listed in the freeze was present and matched its recorded digest.

The sanitized result SHA-256 is `6a6ef142c7e6e3293ce8789e5b342e54643615898cccb6e7dff433b149cf4b61`; the process receipt SHA-256 is `55031f91b9c201b73044ef7e9378fc8325bcb9f06002cf05a2a56a2d5bc95f89`. The process receipt records one Windows Job-controlled process tree, return code 0, wall time about 0.456 seconds, no retry, and no filesystem isolation guarantee. Its stdout receipt hash is `df8f8fd140b55088c6a8f253f64aace2a9066cadcd3c0b54138f301412f016eb`; stderr was empty. These receipts match the sanitized result and one-time markers; no replay was performed.

## Observed result

The client sent each authorized RPC once: `initialize`, `initialized`, `config/read`, and `configRequirements/read`. Both read responses arrived. The sole server notification was a schema-valid `remoteControl/status/changed` with status `disabled`; only method, class, and status were retained. No warning notification was observed, `provisionalConfig` is false, the native process and client both returned 0, and no retry occurred. The client consumed and discarded 49,985 raw response bytes in memory; no raw frame payload or raw configuration was persisted.

The sanitized `config/read` response reported `approval_policy=never`, `approvals_reviewer=null`, `sandbox_mode=null`, `windows.sandbox=elevated`, `shell_tool=true`, `unified_exec=true`, and `windows_sandbox_service=false`. It reported apps, goals, hooks, memories, multi-agent, and plugins false. Provenance included session flags plus user and system configuration layers. The sanitized `configRequirements/read` response reported `network=null`.

These are the app-server's returned metadata values for this invocation. They do not prove effective permission enforcement, read-only behavior, isolation, network reachability, or the absence of inherited configuration. The app-server route differs from the historical exec route; the returned values must not be treated as proof that the earlier exec policy was applied or that an Actor would have the same context or rights.

## Accounting and conclusion

The reservation preserved the prior totals: two earlier app-server policy-read trees, six CLI metadata calls, five Actor starts, 53,331 known Actor input-plus-output tokens, and an unknown historical Actor token total. This run consumes the third app-server tree and adds no Actor start or model call. The historical token total remains null, and the prior remote-control status remains unknown; the current `disabled` observation does not retroactively establish either value.

The authorized metadata session completed with both read responses and a disabled status notification. It provides bounded metadata evidence only. No S1 readiness, product behavior, effective policy, or model/Actor permission claim follows from this result.

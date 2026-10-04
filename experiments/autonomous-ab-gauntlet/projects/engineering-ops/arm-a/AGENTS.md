# Relay engineering guidance
Read docs/engineering/architecture.md. Accepted policy follows.
<!-- rules-sha256: 2638b985f3f86ca2580c20085611d3b5498a6ef644954a47f6188006c32d0d0c -->
# Accepted agent and operations rules

This file is the maintained rule source for both provider entrypoints.

1. Fail closed on malformed configuration.
2. Preserve request IDs in logs. Never log payloads or credentials.
3. Child processes inherit cancellation and use the established five-second timeout. Report only observed exit status and elapsed time.
4. Keep dependency direction `cmd -> internal`. Reusable process primitives belong in `internal/process`; `tools/process-sentinel` stays independently buildable and cannot import the root module.
5. CI and pre-commit both run the operations policy check, its Python unit tests, root Go tests, and nested-module Go tests. Root `go test ./...` does not test the nested module.
6. Do not reinterpret OPS-142 or OPS-188 as approval for a shared cross-module library.
7. `testdata/vendor-snapshots/` is immutable parser-test input, outside production policy ownership.
8. `docs/operations/runbooks/legacy-start.md` is obsolete when the replacement development workflow is adopted; remove stale active references.

Update canonical resources rather than editing generated projections. Rule changes update this file and both provider entrypoints together. `python scripts/check_operations.py` checks the textual parity and managed-path inventory.

# Relay architecture

Relay is an event-forwarding service. `cmd/relayctl` is the executable boundary and may depend on `internal/`. Reusable process primitives belong in `internal/process`. The independent `tools/process-sentinel` Go module is a standalone utility and must not import the root module. Do not introduce a cross-module shared library without an explicit owner decision.

The root and nested modules have independent test commands. A green root test run says nothing about the nested module; this was learned in OPS-188 after a missed nested regression. OPS-142 added child cancellation propagation after a stuck process delayed shutdown. Those changes did not authorize moving process code between modules.

Repository artifact coverage names exact roots and owners. Files outside those roots still need ordinary source review and tests; testdata/vendor-snapshots is immutable fixture input.

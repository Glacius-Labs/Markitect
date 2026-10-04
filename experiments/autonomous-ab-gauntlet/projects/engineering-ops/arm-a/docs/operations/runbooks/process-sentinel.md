# Process sentinel operations

Use `go -C tools/process-sentinel run . -- --help` to inspect the standalone helper and `go -C tools/process-sentinel test ./...` to test it. The helper reports the observed child exit code and elapsed time. Version 1 text output is a supported interface.

The caller propagates cancellation. The default deadline is five seconds. A timeout is not a successful completion; report it distinctly and preserve the request ID in logs. Never include child payloads, credentials, or full environment values in logs.

# 23c2 platform gate receipt draft

This candidate packet binds the exact source `23c2c30d5e352b9172a0d24b82528012bfba1c18`. The Linux full gate harness passed. Windows build and gates 0–14 passed; direct `go test ./... -count=1 -json` passed in 416.148 seconds. Fixed-snapshot Verify exited 2 after 729.823 seconds and is **INCOMPLETE**: its internal `go-tests` check returned `exitCode=-1` at 602000 ms with the recorded message `exceeded 10m0s execution limit`. Later gates did not run. The outer Verify row's `timedOut=false` does not negate the nested check timeout; the cause of the check reaching its limit remains **UNPROVEN**.

The Linux receipt includes its exact 87-command harness trace. The trace count is not comparable to Windows test-event counts. The Windows direct-Go JSON event counts and skip list are from the actual captured stdout; raw output and `environment.json` remain external. This packet makes no diagnostic fix claim. Separate e788 evidence remains separately bound and unrescored; the e788 no-`.git` diagnostic assessment has not established alias or timeout causality.

This is reviewed candidate evidence only. It does not claim semantic assurance, runtime or deployment behavior, human acceptance, or release readiness.

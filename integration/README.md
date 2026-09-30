# Consumer integration

Markitect owns a Go-only bootstrap distributed as `run-markitect.go`. A consumer copies it to `scripts/run-markitect.go` and its test to `scripts/markitect-bootstrap_test.go`. Its existing repository checks may remain in their current language and call this Go entrypoint.

The tool release is a verified source archive. `markitect package --repo . --output NEW_DIRECTORY` creates `tools/markitect/source.zip` and a flat `markitect.lock.yaml` with exact `version`, `source` and SHA-256. Copy the archive and lock together into the consumer. The bootstrap is a separate integration source file.

Run from the consumer repository root:

```powershell
go run scripts/run-markitect.go version
go run scripts/run-markitect.go check
go test scripts/run-markitect.go scripts/markitect-bootstrap_test.go
```

The bootstrap finds the lock from the current directory or its parents, verifies source/archive/cache integrity and builds the pinned Go module. It requires Go; its own code uses only the standard library. The first tool build can acquire the toolchain and checksum-verified Go dependency. Subsequent runs can use verified local inputs and caches. In a restricted shell, configure a writable `GOCACHE` and `GOTMPDIR` before `go run` because the outer Go invocation runs before the bootstrap.

Build the bootstrap as a small executable when exact CLI exit codes or repeated invocation speed matter. `go run` itself can map a program's nonzero status to its own exit status; automation should use the built bootstrap/binary when distinguishing Markitect's 1 from 2.

The current lock is a tool-distribution lock. Content-package imports require the future versioned format described in [the design](../docs/refinement.md); do not add unknown fields to this lock.

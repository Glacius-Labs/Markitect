# Fixed-source example gate receipt

Source: `4121288e03a84475afb7bcc2dbcef2ec9ff80927` on `codex/proof-example-gates`.

The frozen CLI executable passed hash verification at SHA-256 `6133f02316fffcbc1f79f7c6444dcf90636f80f1ab5097f5263400c2216d51f3`; its build receipt matched SHA-256 `219b909aa45dcbebfd424e131cfbaf5ae43a3de439d5b6803761ca7fd66ad771`. Schema, example check/format/model/context, root dogfood, and source packaging commands used that frozen executable. Its machine path is redacted in the command labels.

Configured module and artifact checks, module integrity, focused content-package/consumer tests, and bootstrap tests used Go 1.27.1 on Windows amd64 with `GOENV=off` and `GOTOOLCHAIN=local`. Go resolved from the pinned 1.27.1 toolchain path; Git was first in PATH. Source-based checks and tests compiled from commit `4121288e03a84475afb7bcc2dbcef2ec9ff80927`; `go mod verify` checked the module cache. The configured `GOCACHE` was a shared cache in the existing `689d` worktree, separate from this source checkout.

All 49 listed commands exited 0. The receipt contains only source, command, exit, and output digest fields. No repository-wide `go test ./...`, `go vet ./...`, or `go build ./...` command ran for this receipt. Raw logs remain external and are not included.

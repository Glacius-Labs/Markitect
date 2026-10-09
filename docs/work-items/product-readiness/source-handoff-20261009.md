# P01/P02 source handoff — 9 October 2026

Implementation source is **08a61ea706584ceeab1b387ff67f07fe3dfdccf2**, following the shared-contract commit 36c20ea36ae15e5aff0d4095a6d6f1f2e1a0eece. The publication commit adds only completion/handoff records. Root pins its full pushed SHA from the completion callback as the common base for fresh feature worktrees. Branch: codex/model-first-operations; draft [PR89](https://github.com/Glacius-Labs/Markitect/pull/89). No force push, Main merge or release forms part of this handoff.

P01 cleanup and minimal P02 contracts are complete. P03–P05 are ready for assignment; P06–P10 remain dependent on the fresh implementation/integration owners under [the handoff direction](fresh-chat-handoff.md). Designer stops here. No current code defect blocks those assignments; the listed runtime gaps remain product work.

The [P02 contracts](P02-shared-contracts.md) define APIs, paths and reserved ownership. The [native report](../../validation/project-native-work-2026-10-09.md) records current behavior, prior failures and proof limits. backlog.yaml is the sole product queue; A01–A03 remain unconsumed. Research recommendations remain optional and separate.

## Checks and independent review

| Check | Result and scope |
| --- | --- |
| Go contract/execution packages | agentexec 20.098 s; codexappserver 0.333 s; projectworkspace 0.378 s; projectapp 10.091 s; architecture 0.680 s, all passed with count=1 and timeout=2m |
| Latest coupled projectrun guard/receipt regressions | 17.284 s passed; rejects unconsumed Host delta and retains original error/receipt |
| All-package Go compilation | go test ./... -run '^$' -count=1 -timeout=3m passed; this compiles tests without running the full suite |
| Go vet | Six touched Host packages passed |
| Selected CLI regressions | Selected-input Plan and read-only Repair/Status tests passed; read-only Brownfield/distillation tests were already passed at 521057d3 |
| Python adapter | Ran 54 tests; 53 passed, one Windows symlink skip. Two mock ResourceWarnings remain; no provider executed |
| Shop | Twelve Python tests passed. Fixed project check at fixture ac813bbe158c0f480c7172e160664c026f1848d2 succeeded |
| Structural/artifact checks | Source check and managed ownership passed; final publication rechecks queue/docs bindings |
| Source formatting | gofmt and git diff --check passed |

The selected tests exercise current boundaries, not full native acceptance. The earlier full scoped Go run timed out; it has not been relabelled as passing. Hosted CI on the new publication is a separate gate, pending at handoff unless the actual remote result establishes otherwise.

Independent source reviews closed the read-only agent-binding defect, conflicting facade revisions, invalid UTF-8 digest aliases and unconsumed-delta handling. No remaining actionable defect was found within those review scopes. No reviewer started measured product roles.

The latest own-project candidate binary has SHA-256 ddef745fe4044f073fae1f0fa1ae89f75fec5a961d8a0d8d696755798e254a27. It is a source build, not an installed/versioned release. Earlier candidate binaries and failed proof records remain unchanged.

## Fresh implementation constraints

P03 implements real repository/history/WIP workspaces and observed byte deltas. P04 implements native App Server execution against the explicit config and shared invocation validation, with distinct thread/turn identities and reported lifecycle. P05 implements MCP over product operations, including any needed shared lifting of remaining CLI-local setup/exploration/Brownfield orchestration. Product Integration owns shared types, dependency changes, runtime selection, candidate conversion, aggregate accounting and P06–P10. The exact file boundaries are in P02.

The current process adapter supports a scoped text file tree without Git history and has native helpers disabled. It rejects owned workspace handles; projectrun rejects new observed deltas until a real consumer is wired. Existing scoped NativeWork/response validation must not be silently reinterpreted as full Git/binary/delete support. App Server settings must be explicitly selected and bound to the runtime, with actual executable/instruction pins in its fingerprint. No broader permission profile or provider authority follows from these DTOs.

Actual native acceptance A01–A03 is NOT RUN: zero jobs and zero role-start requests used from the separate maximum three jobs/5,400 seconds/64 start-request allocation. The prior 600-second lease is closed after two pre-provider CLI failures, with zero actual provider starts. Final functional/semantic readiness, supported-platform CI and human acceptance remain unproved.

## Normal commands

Run from each fresh Markitect feature worktree, using the full pinned publication SHA supplied by Root:

~~~powershell
git status --short
git rev-parse HEAD
go test ./internal/host/agentexec ./internal/host/codexappserver ./internal/host/projectworkspace ./internal/host/projectapp ./internal/tooling/architecture -count=1 -timeout=2m
go test ./internal/host/projectrun -run 'TestUnwiredWorkspaceDeltaFailsAndPreservesReceipt|TestManagerFailurePersistsReturnedReceiptExactlyOnce' -count=1 -timeout=2m
go test ./... -run '^$' -count=1 -timeout=3m
go vet ./internal/host/agentexec ./internal/host/codexappserver ./internal/host/projectworkspace ./internal/host/projectapp ./internal/host/projectrun ./internal/host/projectcli
go run ./cmd/markitect check --repo . --revision PINNED_SHA
go run ./cmd/markitect-check-artifacts
git diff --check
~~~

The executable Shop README describes copying the nested example into a separate Git repository; directly selecting its nested source path correctly fails the top-level-root guard. Ordinary users follow project init, onboard preview/write, native setup preview/write, commit the accepted model/instructions/runtime, then plan/run/review/verify/apply under their existing task authority. Concrete current commands and limits are in the project workflow guide. P07 validates that full journey after P03–P06 are implemented; no provider invocation is required to run the checks above.

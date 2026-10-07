# Government G5 finite queue and recovery fixture

This deterministic native fixture exercises bounded queue behavior through the public `government queue` and `government resume` CLI actions. Each trial creates an isolated Git repository, managed Active refs, external runtime files, a finite backlog, raw process output, and durable queue state. Its runner is a standalone Go module and does not import Markitect packages.

The branch case contains a missing-mandate order, an order that explicitly depends on it, and an independent order on a separate Active ref. The independent order can complete while the blocked branch remains unresolved. The two interruption cases pause a temporary `git.exe` proxy immediately before or after it delegates the real `update-ref`, kill the actual Markitect host process tree, then resume the same named queue. The before-CAS case also spends a five-start cumulative budget across both the interrupted order and the independent order. In the after-CAS case, the fixture advances the managed ref to a later descendant using real Git before resume; recovery must retain that newer ref while recognizing the exact prior promotion edge. The after-CAS queue has enough budget for the independent order after recovery. Both repeat resume and verify that saved work does not run actors or promote twice. Provider usage remains unknown when the deterministic runner does not report it. A final case changes the runtime bytes and backlog bytes in separate trials and confirms stale bindings stop further work.

Build Markitect from the source under test and run the fixture from PowerShell:

```powershell
go build -o "$env:TEMP\markitect-g5.exe" ./cmd/markitect
New-Item -ItemType Directory -Force "$env:TEMP\markitect-g5-output" | Out-Null
pwsh -NoProfile -File .\examples\government-g5\trial.ps1 `
  -OutputDirectory "$env:TEMP\markitect-g5-output" `
  -MarkitectExecutable "$env:TEMP\markitect-g5.exe" `
  -Case all
```

Every trial writes raw Markitect stdout/stderr, runtime and backlog JSON, Git refs and trace, queue directory, and an assertion summary under its unique external trial directory. These cases establish only the deterministic mechanics exercised by these fixed inputs; they do not establish real model quality, human approval, or operating-system sandboxing.

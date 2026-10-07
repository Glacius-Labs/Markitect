# Government G4 model amendment and conflict mechanics

This public fixture is a deterministic native Host exercise for bounded G4 behavior. It keeps a small invoice model in `government.yaml` as JSON (and therefore valid YAML), with exact predeclared writers for that source and its Go realization files.

The positive order identifies a real gap: the active invoice rule does not say whether delivery is included. Under its prior `amend-model` mandate, the root Area proposes an explicit delivery-charge rule, changes the associated implementation and acceptance test together, and retains the existing purpose-bearing artifacts and realization edges. Host assessment compares the proposal with the frozen prior model. The actual `go test ./...` reads the candidate model, rate, and implementation bytes; a separate reviewer then checks the same proposed behavior. Promotion requires fresh assent from both frozen Ressorts for the final material and evidence.

`trial.ps1` exposes native modes for the authorized amendment, a one-time Ressort objection followed by bounded repair, a persistent veto that exhausts the repair limit, a protected-root self-authorization attempt, a missing prior amendment mandate, stale vote replay across two distinct candidate/evidence rounds, and an explicit independent order after an unresolved conflict. The bounded-repair actor first objects that the published charge must remain exactly two units. The executor receives that recorded feedback, changes the domain rule and its realization/test together, and gets fresh votes from the same frozen cabinet. The persistent-veto case retains both rounds and escalates without promotion. Both the actual Go acceptance check and independent review verify the preserved two-unit requirement, so the writer cannot make a false repair pass by changing the rate or weakening the check. The conflict and independent order use separate subject and artifact paths. A fixture actor can emit the text “Owner approved”, but only the unchanged Host process decides authority from the active prior Constitution and frozen plan.

The fixture runner also exposes `propose-model`, `propose-realization`, and `child-review` modes for recursive integration checks. The root-only model actor changes only `government.yaml`; the `invoice/code` child actor emits only `invoice/invoice.go` and `invoice/invoice_test.go` without reading the model; the child reviewer checks those exact bytes and the exact child Area plus delivery-rule scopes. These modes exercise distinct prior writers and reviewers without changing the ordinary single-Area trial modes.

Run from a source checkout after building Markitect to an absolute path outside the checkout:

```powershell
go build -o "$env:TEMP\markitect-g4.exe" ./cmd/markitect
pwsh -NoProfile -File .\examples\government-g4\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g4.exe" -Case all
```

The script makes a unique external Git fixture for each case and retains its source-built fixture runner, exact runtime and order JSON/YAML, raw Markitect stdout/stderr, Host state, candidate workspaces, actor records, escalation records, and final `HEAD`/managed-ref values. In the stale-vote case it first records the actual vote IDs from an accepted native run, advances the fixture checkout to that promoted candidate, and launches a fresh native process with the old vote IDs. The new actor envelope is bound to its actual invocation; only the embedded candidate/evidence/round claims are stale.

These fixed-process checks establish Host mechanics against these exact fixtures. They do not establish semantic model quality beyond the fixture's deterministic rules, real provider independence, human approval, operating-system sandboxing, or durable queue/recovery behavior. G5 is outside this example.

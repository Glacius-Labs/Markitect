# T8 merged-state checks: native Makerspace W0 — 30 October 2026

Native evidence for step T8 of the [delivery plan](../implementation-plan.md#concept-20-build) ([DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)): the required checks refuse only merged states. In the [T7 pilot](t7-pilot-20261029.md) the concept 2.0 arm landed no Makerspace step, because Root's first submission could not pass the required `build` check while the directory it builds belongs to a child Package. The run falls under the owner's standing run approval of 28 October 2026 for build runs under a dispatch ([build workstream, Decisions](../workstreams/build.md)). It is source and smoke evidence, not provider, release or owner acceptance, and not part of the pilot or the counted series.

## Question

Does the Makerspace chain's first step, W0, a fresh build of the whole Makerspace model, now land natively on Windows with the concept 2.0 arm?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `dev/t8-merged-state-checks` at its reviewed head `SOURCE`. `markitect.exe`, `markitect-remote` (Linux, amd64), the Manager image, the Go Manager image and the sandbox image are built from it under the lock; the suite's `tools/` tree is frozen from it by `git archive` |
| Arm | M only: [markitect_arm.py](../../tools/evaluation/README.md#the-concept-20-arm) under `suite.py chain --native`, the Makerspace chain with the one step W0, as the pilot ran it; `claude:claude-sonnet-5-5`, effort `high`, the account login only, never an API key, every `ANTHROPIC_*` and `CLAUDE_*` variable removed |
| Runner | The pilot's runner (`t7/pilot.py` in Build's notes folder, unchanged) with a one-unit configuration: prepare, one launch, judge W0 with the Makerspace series judge exported at the source, report. No review sample |
| Deadline | 5400 s, the pilot's Makerspace deadline |
| Limits | The host's: 200 agent starts, 6 maker and 6 checker starts per Manager, one hour per start; each Manager container 1 CPU, 2 GiB, 256 processes; the remote 2 CPUs, 4 GiB, 1,024 processes. One launch; one relaunch only if the suite classes it void |
| Engine | Docker Desktop's regular engine; Codex is stopped. Before the launch, `docker ps` lists no container that is not the suite's or the Run's and `tasklist` shows no `dotnet` process |
| Lock | `NATIVE-RUN.lock` (`local_cbc94143…`), taken by the suite itself for the run and the judgment and by the runner for a login refresh; the runner is not wrapped in another lock. A busy lock is waited for |
| Abort | A `STOP` file in the work folder before the launch, a failed login refresh, a quota stop, a credential found in the evidence, or a Run past its deadline |
| Cleanup | The host removes the Run's remote, network, containers and volumes; afterwards nothing labelled with the Run may remain |
| Evidence | Outside the repository, `C:\Users\Consiliari\AppData\Local\markitect-t8-w0`: the runner's log and ledger, the Run's Result, the judgment, the agents' transcripts, the leftovers by the Run's label, and a scan of every file for the login's tokens |

## Result

Pending.

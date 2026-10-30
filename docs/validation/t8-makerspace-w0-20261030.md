# T8 merged-state checks: native Makerspace W0 — 30 October 2026

Native evidence for step T8 of the [delivery plan](../implementation-plan.md#concept-20-build) ([DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)): the required checks refuse only merged states. In the [T7 pilot](t7-pilot-20261029.md) the concept 2.0 arm landed no Makerspace step, because Root's first submission could not pass the required `build` check while the directory it builds belongs to a child Package. The run falls under the owner's standing run approval of 28 October 2026 for build runs under a dispatch ([build workstream, Decisions](../workstreams/build.md)). It is source and smoke evidence, not provider, release or owner acceptance, and not part of the pilot or the counted series.

## Question

Does the Makerspace chain's first step, W0, a fresh build of the whole Makerspace model, now land natively on Windows with the concept 2.0 arm?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `dev/t8-merged-state-checks` at `d5d76b8e` (reviewed; this page and the plan row were added after the run). `markitect.exe`, `markitect-remote` (Linux, amd64), the Manager image, the Go Manager image and the sandbox image are built from it under the lock; the suite's `tools/` tree is frozen from it by `git archive` |
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

**Landed.** Run `eval-7f9e5d429a37`, source `d5d76b8e`, launched 01:50Z, the Run ended 02:14Z (about 24 minutes of the 5400 s). It was one launch; the suite classed it a result, with no escape.

| What | Value |
| --- | --- |
| Images | Manager `sha256:a98b0207…`, Go Manager `sha256:090baf22…` (the Managers' image), sandbox `sha256:e4ea4049…`, all built from `d5d76b8e` |
| Run | State `landed`, Root's checker approved the work, landing commit `bfbacd6c`. 35 agent starts, all 297 checklist items approved. The required checks `build` and `test` passed on the whole repository. 12.9 M input tokens (mostly cache), 291 k output tokens |
| Judgment | The Makerspace series judge at W0, valid: 54 of 54 cases, build and tests passed, 169 changed paths, none unrequested, no protected input touched |
| Post-check | No container, volume or network of the Run left; the scan of 1,781 files found no login token; quota after the run 5 % of five hours, 17 % of seven days |

**How the checks behaved** (from the remote's decision log):

- **Children's first submissions:** every child's first submission failed `build` on its partial tree and was accepted with the note. These were safety, members, equipment, materials, training, reporting, bookings and billing.
- **Root's own work:** Root's own-area submission also failed `build` and was accepted. This is the submission that ended every Makerspace step in the pilot.
- **Root's merged states:** two merged states that failed `build` were refused, at 01:57 and 02:00. Root then fixed its own area. Its merged state at 02:07 passed `build` and `test`, and its checker approved it at 02:09.
- **Other refusals:** three refusals guarded paths. Root changed `protocol`'s registry page, and `protocol` changed `billing`'s page and Root's `realization/README.md`. No check result reached them.

**Limits.** This is one launch, not a repetition: it shows that the step can land, not how often it does. The Library's no-regression run was not repeated, since the hook tests and the Run tests cover the Library-shaped paths.

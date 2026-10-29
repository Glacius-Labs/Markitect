# T4 smoke fixes: native evidence — 29 October 2026

Native evidence for the four product fixes of PR #874: an escalating Manager continues from its parent's settlement, the Run's network is isolated, the Run's remote has limits, and the submit tool says that an accepted push is a submission. The runs fall under the owner's standing run approval of 28 October 2026 for build runs under a dispatch. They are source and smoke evidence, not provider, release or owner acceptance.

## Questions

- **Q1:** does the Library smoke (`apply --full` on Library M1, then `verify --full`) still land and verify on Windows, with the isolated Run network and the remote's limits? This is the no-regression check.
- **Q2:** does the T7 smoke's launch 3 (Library M2 from the M1 reference start, adopted without Realizes links, `apply --base` the adoption commit) now land, or stop honestly without burning all of `lending`'s rounds?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `fix/t4-smoke-fixes` at `794da3b4` (Main `7c3642bb` plus the fixes). `markitect.exe` and `markitect-remote` (Linux, amd64) built from it |
| Images | The Go Manager image `markitect-manager-go:t5-33d87c09` (`sha256:5dbb6f1bf22d…`), by engine image ID. The agent daemon and `packaging/` are unchanged since `33d87c09`, and the tool texts the fixes change are the host's. The Run remote and its checks use the local `golang:1.27`, no pull |
| Provider | Claude Code on `claude-sonnet-5-5`, effort `high`, through the host's account login only (refreshed with one `claude -p` call under the lock before the runs), never an API key; every `ANTHROPIC_*` and `CLAUDE_*` variable removed |
| Q1 command | The T5 smoke script (`t5smoke/smoke.py` in Build's notes folder): a fresh project from `tools/evaluation-tasks/library/m1` with the Library's registry paths, a local bare target, `apply --full --deadline 60m`, then `verify --full --deadline 30m` |
| Q2 command | T7's pilot runner (`t7/pilot.py`) on one fixed unit, Library M2, arm M, from Main once T7's PR #873 has merged, with the unlinked adoption, as in launch 3 |
| Limits | The host's: 200 agent starts, 6 maker and 6 checker starts per Manager, one hour per start; each Manager container 1 CPU, 2 GiB, 256 processes; the remote 2 CPUs, 4 GiB, 1,024 processes. One launch per question; a second only after a fix of a defect the first shows |
| Lock | `NATIVE-RUN.lock` in Build's notes folder, taken before the first engine call of each run and removed after its post-checks |
| Abort | A failed login refresh, a credential found in the evidence, a Run past its deadline, or a busy lock |
| Cleanup | The host removes the Run's remote, network, containers and volumes; afterwards nothing labelled with the Run may remain |
| Evidence | Outside the repository, `C:\Users\Consiliari\AppData\Local\markitect-t4fix`: both Results, stderr, transcripts, the engine's leftovers by the Run's label, the network's gateway mode, and a scan of every file for the login's tokens |

## Results

Pending.

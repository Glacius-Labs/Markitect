# T4 smoke fixes: native evidence — 29 October 2026

Native evidence for the four product fixes of PR #874: an escalating Manager continues from its parent's settlement, the Run's network is isolated, the Run's remote has limits, and the submit tool says that an accepted push is a submission. The runs fall under the owner's standing run approval of 28 October 2026 for build runs under a dispatch. They are source and smoke evidence, not provider, release or owner acceptance.

## Questions

- **Q1:** does the Library smoke (`apply --full` on Library M1, then `verify --full`) still land and verify on Windows, with the isolated Run network and the remote's limits? This is the no-regression check.
- **Q2:** does the T7 smoke's [launch 3](t7-arm-smoke-20261029.md) (Library M2 from the M1 reference start, adopted without Realizes links, `apply --base` the adoption commit) now land, or stop honestly without burning all of `lending`'s rounds?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `fix/t4-smoke-fixes`: Q1 at `794da3b4` (Main `7c3642bb` plus the fixes), Q2 at `8c37aa9a`, which adds the independent review's fixes (below). `markitect.exe` and `markitect-remote` (Linux, amd64) built from each |
| Images | The Go Manager image `markitect-manager-go:t5-33d87c09` (`sha256:5dbb6f1bf22d…`), by engine image ID. The agent daemon and `packaging/` are unchanged since `33d87c09`, and the tool texts the fixes change are the host's. The Run remote and its checks use the local `golang:1.27`, no pull |
| Provider | Claude Code on `claude-sonnet-5-5`, effort `high`, through the host's account login only (refreshed with one `claude -p` call under the lock before the runs), never an API key; every `ANTHROPIC_*` and `CLAUDE_*` variable removed |
| Q1 command | The T5 smoke script (`t5smoke/smoke.py` in Build's notes folder): a fresh project from `tools/evaluation-tasks/library/m1` with the Library's registry paths, a local bare target, `apply --full --deadline 60m`, then `verify --full --deadline 30m` |
| Q2 command | As the original launch 3, with Main's suite tools (unchanged on the branch) and no pilot runner: `series.py steps` for M1 and M2, `judges.py start` at M1, `suite.py task --start` that bundle `--base-model` M1's own model (no reference pages, so the arm adopts every file as an unlinked row), then `suite.py run --native` the sandbox image `sha256:e0b20d8a8c88…` with the arm `markitect_arm.py`, the same `--hidden-absent` paths as the T7 smoke, and `suite.py judge` at M2 with the Library judge exported at the branch; deadline 2700 s |
| Limits | The host's: 200 agent starts, 6 maker and 6 checker starts per Manager, one hour per start; each Manager container 1 CPU, 2 GiB, 256 processes; the remote 2 CPUs, 4 GiB, 1,024 processes. One launch per question; a second only after a fix of a defect the first shows |
| Lock | `NATIVE-RUN.lock` in Build's notes folder, taken before the first engine call of each run and removed after its post-checks. For Q2 the suite also took the `NATIVE-RUN.lock` the T7 smoke used (`local_cbc94143…`), so both lock files were held |
| Abort | A failed login refresh, a credential found in the evidence, a Run past its deadline, or a busy lock |
| Cleanup | The host removes the Run's remote, network, containers and volumes; afterwards nothing labelled with the Run may remain |
| Evidence | Outside the repository, `C:\Users\Consiliari\AppData\Local\markitect-t4fix`: both Results, stderr, transcripts, the engine's leftovers by the Run's label, the network's gateway mode, and a scan of every file for the login's tokens |

## Results

| Question | Verdict |
| --- | --- |
| Q1, no regression | **Passed** (`t4fix-win-1`, source `794da3b4`, 13:51–14:04Z). `apply --full` landed in 11.3 min with 14 starts, none failed; `verify --full` then verified 40 of 40 items with the required checks passing, in 88 s, 5 starts. Nothing labelled with either Run was left; the secret scan of 137 files found nothing. While the apply ran, its network was internal with gateway mode `isolated` and no gateway address, and its remote ran with 2 CPUs, 4 GiB and 1,024 processes |
| Q2, launch 3 again | **Landed** (Run `eval-af3e1f99d3ac`, source `8c37aa9a`, 14:04–14:14Z, 581 s): 14 starts, none failed, no Manager near its 6 rounds, 1.48 M input (mostly cache) and 27 k output tokens, about 1.05 USD list. The judge at M2: 19 of 19 cases, the three loan-limit cases among them, which the unchanged start fails; build and tests passed; nothing unrequested, no protected input touched, no drift. The arm's network was internal with gateway mode `isolated` (the T7 smoke recorded it empty). Nothing labelled with the Run was left; the secret scan found nothing. **The suite classed the run invalid**, for a reason outside the product (below) |

**How launch 3 converged.** `lending` and `lending/reminders` worked first; `lending` was approved once, then got the host message to merge its child's approval. On its second start it escalated: `library.go`, Root's file, held its own `MaxOpenLoans = 5`. The host held `lending`. Root settled the matter with a commit of its own (it merged `lending`'s approved policy package and made `library.Borrow` use it), and Root's checker approved that settlement without waiting for its children. The host then let `lending` start again from that approval with a message naming it, and reminded Root once to answer, which it did. `lending`'s next submission was rejected once: Root had already merged `lending`'s earlier approved work, so its diff from the settlement was empty and its `changed` verdict had no evidence. Its next submission (`conforms`) was approved, and Root integrated it and was approved. The original launch 3 ended after 19 starts as `incomplete`, with `lending` out of rounds.

**Why the suite classed Q2 invalid.** Sixteen containers outside the evaluation network ran during the run, on the engine's `bridge`. All were Docker Desktop's MCP Toolkit servers (`docker-mcp: true`, the images `mcp/grafana` and `mcp/obsidian`), started by the desktop application's MCP connection on the host, not by the Run or the arm. The suite counts every such container as an escape and fails safe. The verdict above therefore rests on the Run's own Result, the judgment and the post-check, not on the suite's class. No second launch was made, since no product defect caused it. **For the pilot:** native runs on this machine are invalid while the Docker MCP Toolkit is connected.

**The independent review** (Opus, the complete diff at `794da3b4`) found nothing blocking. Its findings, applied in `8c37aa9a` with tests that fail without each fix:

- A held parent that pushes a work commit and no submission is asked for one, even after it sent a message.
- A child's submission from before its parent's settlement is not reviewed.
- The child continues from its parent's newest approval since the escalation, also when the parent answered without a change (tested after the review's second pass); the host's message no longer claims the matter was settled.
- The Run's network needs Docker 28, and an older engine is refused before anything is created.
- The submit tool's description matches its reply.
- The docs say which waits exist.

## Limits

- Windows only; one launch per question.
- Q1 ran before the review's fixes and Q2 on them. Q2 exercised the hold, the review of a settling submission without the children's approvals, the release from the parent's approval with the neutral message, and the Docker 28 check. A parent's work commit without a submission, a child's submission from before the settlement, the no-change answer and the `model-problem` path did not occur; source tests cover them, each failing without its fix.
- On the no-change path the child starts again once its parent's answering start has ended, not on the answer itself. No agent waits meanwhile. Concept 2.0 §6.3 says that children wait only for their parent's approved commit and does not name this wait; Build decides whether the concept's wording should.
- The tool text's effect on Codex agents, who took an accepted push for an approval in T5, was not run here.

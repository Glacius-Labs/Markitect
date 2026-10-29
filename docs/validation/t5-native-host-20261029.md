# T5 native host: the Library smoke on Windows and on Linux — 29 October 2026

Step T5 of the [delivery plan](../implementation-plan.md#concept-20-build): the T4 Library smoke ([T4 record](t4-library-smoke-20261029.md)) natively on Windows and on Linux, with one `verify` after the landing, and the evaluation suite's native-mode round on Linux. It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)). It is also the first real-container run of [PR #865](https://github.com/Glacius-Labs/Markitect/pull/865): the parent-first start (`markitect-remote start`), the hook compiling every submission, and the required checks in `verify`.

## Questions

- **Q1:** does `markitect apply --full` on the Library M1 land natively on Windows, with the target branch's lock entered and left?
- **Q2:** do PR #865's changes hold in real containers: does a parent with items submit its own area first, and do its children start from that approval? Does the hook compile every submission?
- **Q3:** does `markitect verify --full` after the landing give every item a verdict from a checker and run the required checks once, on Root's verdicts?
- **Q4:** the same as Q1 to Q3 natively on Linux.
- **Q5:** does the suite's native-mode round, with the stand-in arm and the escape probe, hold on Linux?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Branch `dev/nifty-bartik-d01344` at `9420764c` (Main `3f47835a` plus the T5 change), and after a fix at `4d63d305`. `markitect` built for the host and `markitect-remote` for Linux from the attempt's source; the Manager images built from it too |
| Images | The Manager image from `packaging/runtime` at the source, then its Go example on it, named by engine image ID. The Run remote and its checks use `golang:1.27`, which `execution.yaml` names: the local `golang:1.27.1` image tagged so, no pull |
| Project | A fresh repository: T2's Library M1 model as `.markitect/`, `markitect init`, one registry page per Package claiming `realization/internal/<package>/**` (Root owns the rest); `markitect check` passes. The target is `main` of a local bare repository, so no hosted repository changes |
| Commands | `markitect apply --full --deadline 60m`, then `markitect verify --full --deadline 30m`, with `MARKITECT_MANAGER_IMAGE`, `MARKITECT_REMOTE_BINARY`, `MARKITECT_ENGINE`, `MARKITECT_LOGIN` and `MARKITECT_TRANSCRIPTS`; Claude Code 2.1.308, `claude-sonnet-5-5`, effort `high`. Every `ANTHROPIC_*` and `CLAUDE_*` variable is cleared for both commands |
| Login | The host's Claude Code `.credentials.json`; each Manager gets the access-only copy |
| Limits | One attempt per host, a second only after a fix of a defect the first one shows. The engine's default 200 starts and 6 rounds per Manager, one hour per start. Containers: 1 CPU, 2 GiB, 256 processes, 256 MiB temporary space each |
| Abort | A login or launch-check failure, a credential found in the evidence, or a Run past its deadline ends the attempt; the engine stops it honestly as incomplete |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, created exclusively before the first engine call and removed after both commands |
| Cleanup | The engine removes the Run's remote and the agents their containers and volumes; afterwards nothing labelled with either Run may remain |
| Evidence | Outside the repository: both Results, the transcripts, stderr, the target's refs and log. A post-check builds, vets and tests the landed state in a fresh clone, and scans all evidence for the login's tokens, JWTs and Anthropic key shapes |

## Dispatch: the same smoke on Codex

An owner decision of 29 October 2026 (recorded in PR #868) adds one run: the same native smoke on Windows with Codex Managers, after the Claude smoke, as its own dispatch. It changes neither the Claude series nor the suite.

| Item | Value |
| --- | --- |
| Question | **Q6:** does the same `apply --full`, then `verify --full`, land and verify on Windows with Codex makers and checkers? |
| Source, images, project | As above: the same binaries, the same Go Manager image, a fresh copy of the same project |
| Provider | Codex `0.162.0` (`MARKITECT_PROVIDER=codex`) with the adapter's default model at the source, `gpt-6-sol`, and effort `high`. No model was named in the dispatch; see the results for why Luna is now the default |
| Login | The host's Codex `auth.json` with ChatGPT tokens, never an API key; every `OPENAI_*` and `CODEX_*` variable is cleared. Each Manager gets the copy whose refresh token is `""`, as the pinned CLI needs; `codex login status` reports on standard error, which the adapter reads |
| Limits, abort, lock, cleanup, evidence | As above: one attempt, a second only after a fix; `apply` 60 minutes, `verify` 30 minutes; the shared engine lock; nothing labelled with either Run may remain; the post-check scans all evidence for every token of the login |

## Results

### Windows with Claude

**Attempt 1** (`t5-win-1`, source `9420764c`): both commands refused before any engine call, because the harness named the engine as `docker`, and `MARKITECT_ENGINE` must be an absolute path. A harness error; nothing started.

**Attempt 2** (`t5-win-2`, source `9420764c`), 01:25–01:37Z:

- **`apply` landed** (Q1) in 9 minutes 41 seconds, exit 0: 14 agent starts (7 makers, 7 checkers) for the five Managers, every one ending `finished`. All 40 checklist items have a verdict, evidence and an approving checker. The required checks passed on the whole repository at Root's approved commit (`build` 2.5 s, `test` 0.9 s). 1,795,687 input and 64,051 output tokens.
- **The lock** was entered and left: `refs/markitect/lock/main` existed on the target while the Run worked and was gone afterwards. The Run waited for nobody.
- **Parents first** (Q2), read from the landed history:
  - Root's maker made the first submission, its own area: the Go module, the protocol layer, the shared ports and the wiring. Root's checker approved it (`636bead`).
  - `catalog`, `members` and `lending` all start from `636bead`. The host moved their branches to it on the remote before they started.
  - `lending`, which has items of its own, went first among its subtree. `lending/reminders` starts from lending's approval `451ba1a`.
  - So a real checker accepted a parent's own-area first submission, the open point PR #865 left.
  - Every submission passed the hooks, which now compile each one; none was refused.
  - The organization needed 6 messages: Root's four assignments, which name the exact ports API, and two host notices of approved child work. That is fewer than T4's 18 messages and 25 starts, where the children had no scaffold.
- **The landed state holds up**: in a fresh clone, `markitect check` says `ok: 19 Definitions in 5 Packages, 19 files accounted`, and `go build ./cmd/library`, `go vet ./...` and `go test ./...` pass.
- **`verify --full` failed** (Q3): `incomplete` after 2 minutes, with no verdict pushed. Every checker refused to submit, because the branch tip it started from, the landing commit, carried the new `Markitect-Run` and `Markitect-Work` trailers. The agents' trailer parser rejects unknown `Markitect-` keys. The checkers reported this plainly and did not work around it. This is a defect of the T5 change itself; the source tests missed it because their scripted agents push without the agents' submit tool. It is fixed in `4d63d305`: the host's trailers are now `Run-*` keys, with a test that the parser reads both Result commit messages as plain.
- Cleanup: no container, volume or network of either Run remained. The secret scan of 142 evidence files found nothing.

The review fixes (`cc44e98d`) and the trailer fix (`4d63d305`) came after attempt 2, so attempt 3 runs on the fixed head.

**Attempt 3** (`t5-win-3`, source `4d63d305`, Go Manager image `sha256:d0c9bdbb2bec…`), 01:41–01:52Z: **passed.**

- **`apply` landed** (Q1) in 9 minutes 37 seconds, exit 0, with 14 starts, every one `finished`, and 7 messages. All 40 items have a verdict, evidence and an approving checker. The required checks passed on Root's approved commit (`build` 2.3 s, `test` 0.9 s). 1,828,174 input and 68,225 output tokens. The lock was entered and left, and the Run waited for nobody.
- **Parents first** (Q2), the same shape as attempt 2:
  - Root's own-area submission `d5819c0` was approved as `7325468`.
  - `catalog`, `members` and `lending` start from `7325468`.
  - `lending/reminders` starts from lending's approval `528f54c`.
  - No submission was refused.
- **`verify --full` verified** (Q3) in 57 seconds, exit 0. Five checker starts, one per Manager, gave all 40 items `conforms`, with no Findings and no decisions for the owner. The required checks ran once, on Root's verdicts, whose tree is the start commit's (`4e5c7be`, the landing), and passed. Its Result stays on `markitect/runs/t5-win-3-verify`. 676,443 input and 15,823 output tokens.
- **The landed state holds up**: in a fresh clone, `markitect check` says `ok: 19 Definitions in 5 Packages, 21 files accounted`, and build, vet and test pass.
- Cleanup: no container, volume or network of either Run remained, and the lock ref was gone. The secret scan of 141 evidence files found nothing.

### Windows with Codex

**Attempt 1** (`t5-codex-1`, source `4d63d305`, the same Go Manager image), 01:52–02:17Z.

**Model.** It ran `gpt-6-sol`: the adapter's default at the time, which every provider event confirms. It did not run Luna. After this run, the Codex default became `gpt-6-luna`, in `internal/bootstrap` beside Claude's default, because the owner's Codex series run on Luna and Sol runs only when a dispatch names it with `MARKITECT_MODEL`. A Luna repeat of this smoke follows the counted series.

**Q6, `apply`: did not land.** It ended `incomplete` after 24 minutes: "Manager lending used its 6 maker starts without approved work". Its Result stays on `markitect/runs/t5-codex-1-apply`.

- **Scale:** 37 starts, 41 messages, 11,769,819 input and 146,265 output tokens, against 14 starts and 1.8 million for Claude.
- **Failed starts:** two failed on provider capacity ("Selected model is at capacity"), one of them lending's first maker start. Two more were cancelled at the stop.
- **Unapproved items:** 14 of 40, those of `lending/reminders` and Root.
- **Parents first held:**
  - Root's maker and checker ran first.
  - Root's checker approved the own-area submission on its first review.
  - The children started after that approval.
- **What went wrong** is how the agents coordinated, not the host:
  - The makers took "the remote accepted my push after the required checks" for "my checker approved it".
  - `lending` merged child commits that had stopped being the child's latest approval after the child submitted again. The host told it so: "does not contain lending/reminders's approved work".
  - Root merged `lending` states its checker had not approved, so the hooks refused Root's submissions: "remote still considers 6950f6b the approved lending snapshot".
  - Only one review rejected content: lending's checker found that Root's executable did not use lending's policy yet.
  - The submit tool's reply does not say plainly that an accepted push is not yet an approval. That is a tool-contract finding for T7, to be fixed in the tool's words and the host state, not with prompt text (concept 1.0 §3.7).

**Q6, `verify --full`: correct outcome.** It ended `findings` in 55 seconds. With nothing landed, the target held the model only.

- All five checkers pushed verdicts, so the trailer fix held for Codex too.
- They gave all 40 items `deviates` with evidence, and their reviews were approved.
- The required `build` check ran once on Root's verdicts and failed, as it must on a repository without code.
- 556,887 input and 8,256 output tokens.

**Login and cleanup.**

- The login checks passed with the copy whose refresh token is `""`. No start failed on its login.
- No container, volume or network of either Run remained.
- The secret scan of 119 evidence files, for every token of the Codex login, found nothing.

**Found in passing:** the Codex adapter left the outcome's `Model` empty. Claude Code reports its model in `system/init`; Codex's exec events name none, and only the turn metadata on MCP calls carries it. The outcome now names the model the turn was launched with, with a test.

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
| Source | Branch `dev/nifty-bartik-d01344`: `9420764c` (Main `3f47835a` plus the T5 change), then after each defect fix `4d63d305` and `33d87c09`, as each attempt names. The final head adds after `33d87c09` only the refusal of an explicit `MARKITECT_NETWORK=bridge` and documentation. `markitect` built for the host and `markitect-remote` for Linux from the attempt's source; the Manager images built from it too |
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

**In short.** At `33d87c09`, `apply --full` landed and `verify --full` verified natively on Windows (attempt 4) and on Linux (attempt 2). The suite's native round held on Linux. The earlier attempts found three defects, all fixed in this change: an absolute engine path (harness), host trailers in the agents' namespace, and the engine's `bridge` network on Linux. The Codex run on Windows did not land: its agents confused an accepted push with an approval, a tool-contract finding for T7.

**Limits.**

- One attempt per configuration; this is a smoke, not a measure of reliability.
- Local bare targets only: a push of `refs/markitect/lock/*` to GitHub is not yet observed, nor are two real Runs at once on one target. Source tests cover those.
- macOS is not observed.
- Linux is WSL Ubuntu against Docker Desktop's engine, not a native Linux engine.


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

**Attempt 4** (`t5-win-4`, source `33d87c09`, Go Manager image `sha256:5dbb6f1bf22d…`), 02:44–02:55Z: **passed** again, on the source of the Linux pass. That source changed the provider network after attempt 3.

- **`apply` landed** in 9 minutes 7 seconds, exit 0, with 14 starts, every one `finished`, and 7 messages.
  - All 40 items were approved.
  - The checks passed (`build` 2.6 s, `test` 1.0 s).
  - 1,617,562 input and 60,525 output tokens.
- **`verify --full` verified** in 60 seconds: 5 starts, all 40 items `conforms`, and the checks passed on the landing.
- **The landed state holds up** in a fresh clone.
- **Cleanup:** the lock was held 02:44:22–02:54:35Z. Nothing remained on the engine, and the secret scan of 137 files found nothing.

### Linux with Claude

The host ran in WSL Ubuntu, after the owner turned on Docker Desktop's WSL integration, against the same Docker engine. `markitect` and `markitect-remote` were built in Ubuntu with Go 1.27.1 from a clone at the attempt's source, and the login is the Windows host's file, read through `/mnt/c`.

**Attempt 1** (`t5-linux-1`, source `88788c05`, Go Manager image `sha256:817bc7d7c589…`), 02:28Z: **failed at every agent start**, a platform defect.

- `apply` ended `incomplete` after 5 seconds, when Root's maker made no progress in 2 starts.
- `verify` ended the same way after 3 seconds.
- The cause: every agent container joins the provider network, by default the engine's `bridge`, and the Run's own network. The Docker CLI on Linux refuses that combination: "conflicting options: cannot attach both user-defined and non-user-defined network-modes".
- The Windows CLI of the same version (29.9.2, API 1.56) accepts it, and the engine joins both networks. A probe with a throwaway network showed the difference directly.
- Nothing remained on the engine, and the secret scan of 89 files found nothing.
- **Fixed in `33d87c09`:** the default provider network is now `markitect-provider`, a user-defined bridge network the host creates on first use and keeps (`container.Engine.EnsureNetwork`, with tests). It refuses an internal or non-bridge network of that name. `MARKITECT_NETWORK` still names another.

**Attempt 2** (`t5-linux-2`, source `33d87c09`, Go Manager image `sha256:5dbb6f1bf22d…`), 02:33–02:43Z: **passed** (Q4).

- **`apply` landed** in 8 minutes 54 seconds, exit 0, with 14 starts, every one `finished`, and 5 messages.
  - All 40 items have a verdict, evidence and an approving checker.
  - The required checks passed on Root's approved commit (`build` 2.7 s, `test` 1.0 s).
  - 1,688,121 input and 61,647 output tokens.
  - Every agent container joined `markitect-provider` and the Run's internal network.
- **Parents first**, the same shape as on Windows: Root's own-area submission was approved as `a8e7e32`, and `catalog`, `members` and `lending` start from it.
- **`verify --full` verified** in 55 seconds, exit 0. Five checker starts gave all 40 items `conforms`, with no Findings and no decisions. The required checks ran once on the start commit `e7d02e2`, the landing, and passed (`build` 3.0 s, `test` 1.1 s).
- **The landed state holds up**: in a fresh clone, `markitect check` says `ok: 19 Definitions in 5 Packages, 19 files accounted`, and build, vet and test pass.
- **Cleanup:** the lock was held 02:33:11–02:43:02Z, and its ref was gone afterwards. No container, volume or network of either Run remained. The secret scan of 137 evidence files found nothing.

### The suite's native round on Linux

Q5. This is the round T6 ran on Windows ([T6 record](eval-t6-local-runs-20261028.md)), now in WSL Ubuntu.

- **Setup:** source `33d87c09`, the E1 sandbox image `sha256:e0b20d8a8c88…`, under the engine lock.
- **Hidden material:** each run names `EVAL\gridshare` and `EVAL\hidden` with `--hidden-absent`, and both are absent.
- **Login:** the stand-in uses no model; the suite copies the host's Claude login access-only.

| Run | Expected | Observed |
|---|---|---|
| `change` | delivered with `STANDIN.md`; its Run container on the internal network only; valid | delivered, `A STANDIN.md`, one arm container, no escape, no residue; valid |
| `nothing` | delivered with no change; valid | delivered, no changed path; valid |
| `escape` | the second container on the default network flagged; invalid | the container `standin-…-out` on `bridge` flagged as an escape; not valid |
| `probe` | every escape blocked, the allowed endpoint reached | `contained`. Host files, GitHub, the engine and its host (`host.docker.internal`, `gateway.docker.internal`, `172.17.0.1` on 2375/2376) were all unreachable. `api.anthropic.com:443` was reached through the gateway with TLS 1.3. Valid, no result delivered |

Afterwards no container or network with the evaluation's label remained.

### Windows with Codex

**Attempt 1** (`t5-codex-1`, source `4d63d305`, the same Go Manager image), 01:52–02:17Z.

**Model.** It ran `gpt-6-sol`, the adapter's default at the time. The turn metadata Codex attaches to its MCP calls names `gpt-6-sol` in 40 of the 42 starts. The other two made no call: lending/reminders's third checker start failed on capacity, and its fourth was cancelled at the stop. It did not run Luna. After this run, the Codex default became `gpt-6-luna`, in `internal/bootstrap` beside Claude's default, because the owner's Codex series run on Luna and Sol runs only when a dispatch names it with `MARKITECT_MODEL`. A Luna repeat of this smoke follows the counted series.

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

- All five checkers pushed verdicts. The trailer fix is still unobserved for Codex: with nothing landed, `verify` started from the owner's commit, not from a landing commit with `Run-*` trailers.
- They gave all 40 items `deviates` with evidence, and their reviews were approved.
- The required `build` check ran once on Root's verdicts and failed, as it must on a repository without code.
- 556,887 input and 8,256 output tokens.

**Login and cleanup.**

- The login checks passed with the copy whose refresh token is `""`. No start failed on its login.
- No container, volume or network of either Run remained.
- The secret scan of 119 evidence files, for every token of the Codex login, found nothing.

**Found in passing:** the Codex adapter left the outcome's `Model` empty. Claude Code reports its model in `system/init`; Codex's exec events name none, and only the turn metadata on MCP calls carries it. The outcome now names the model the turn was launched with, with a test.

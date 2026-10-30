# Codex smoke: the Library M1 on Codex Luna — 30 October 2026

The concept 2.0 host's Library smoke natively on Windows with Codex makers and checkers: item 4 of the [build workstream's next steps](../workstreams/build.md). The [T5 Codex smoke](t5-native-host-20261029.md#windows-with-codex) ran on Sol and did not land, because its agents took an accepted push for an approval. Since then the submit reply says that an accepted push is a submission, not an approval ([PR #874](https://github.com/Glacius-Labs/Markitect/pull/874)), and Codex's default model is Luna. The smoke runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)). It is a smoke, not a measure of reliability.

## Questions

- **Q1:** does `markitect apply --full` on the Library M1 land natively on Windows with Codex Luna makers and checkers?
- **Q2:** does the submit reply's fix hold: does every maker wait for its checker's verdict instead of taking an accepted push for an approval, and does every parent merge only its children's approved work?
- **Q3:** does `markitect verify --full` after the landing give every item a verdict from an approving checker, with the required checks passing?

## Dispatch

| Item | Value |
| --- | --- |
| Source | `main` at `7303b16a`, which holds PR #874. `markitect` for Windows and `markitect-remote` for Linux built from it; the Manager image from `packaging/runtime` and its Go example on it, built from it and named by engine image ID. The Run remote and its checks use `golang:1.27`, which `execution.yaml` names, from the engine's local images |
| Why not the suite | The suite's native mode copies only a Claude login (`access_login` in [sandbox.py](../../tools/evaluation/sandbox.py)), so the 2.0 arm cannot run on Codex through it. The smoke runs the host directly, as T5's did |
| Harness | `t5smoke/smoke.py` in Build's shared notes folder, SHA-256 prefix `577ff5dc9c86d25f`, unchanged since T5's Windows attempt 4. It builds a fresh project from [tools/evaluation-tasks/library/m1](../../tools/evaluation-tasks/README.md) with one registry page per Package and a local bare target, runs `apply --full --deadline 60m`, then `verify --full --deadline 30m`, builds, vets and tests the landed state in a fresh clone in `golang:1.27` without a network, and scans all evidence for the login's tokens and JWT shapes |
| Provider | Codex `0.162.0` in the Manager image; `MARKITECT_PROVIDER=codex`, `MARKITECT_MODEL=gpt-6-luna`, effort `high` |
| Login | The host's `~/.codex/auth.json` with ChatGPT tokens and no API key; its access token is valid for days. Each Manager gets the access-only copy with an empty refresh token. Every `OPENAI_*`, `CODEX_*`, `ANTHROPIC_*` and `CLAUDE_*` variable is cleared, and the harness starts detached with a clean environment |
| Engine | Docker Desktop's engine, as the owner chose; Build reported that the owner stopped the Codex desktop app. Right before the launch, no container runs and no `dotnet` process exists. The host smoke has no escape check, so a container of other work cannot void it; the record names any such start |
| Limits | One attempt; a second only after a fix of a defect the first shows, as an amendment here. The host's defaults: 6 rounds per Manager; each Manager container 1 CPU, 2 GiB, 256 processes and 256 MiB temporary space |
| Abort | A login or launch-check failure, a credential found in the evidence, a Run past its deadline (the host stops it as incomplete), or Build's word |
| Lock | `NATIVE-RUN.lock` in Build's notes folder, created exclusively: for the image build, then by the harness from its first engine call to the end of its post-checks, and removed after each. Nothing else wraps it. The T5 session needs the engine for T8, so the window stays short |
| Cleanup | The host removes each Run's remote, containers and volumes; afterwards nothing labelled with either Run may remain |
| Evidence | Outside the repository, `C:\Users\Consiliari\AppData\Local\markitect-codex-smoke\`: both Results, the transcripts, stderr, the target's refs and log, the post-checks and `summary.json` |

## Results

**In short.** `apply --full` did not land. It ended `incomplete` after 15 minutes: "Manager lending used its 6 maker starts without approved work". The submit reply's fix held for the makers: none took an accepted push for an approval. Root still took its children's review commits for approvals and merged their rejected work, and the hooks refused it. The Run then stalled on a review deadlock over the program's entry point, which T5's Sol run had already begun to show. A second attempt without a change would likely meet it again, so there was none. `verify --full` correctly found all 40 items deviating on the unchanged start. Nothing remained, and the evidence holds no credential.

**Attempt 1** (`codex-luna-1`, source `7303b16a`, Manager image `sha256:65aa310f`), 01:33 to 01:48 UTC on 30 October, the lock held for the whole window. Before the launch no container ran and no `dotnet` process existed. Four `codex` processes and the helpers `codex-code-mode-host`, `codex-computer-use` and `codex-windows-sandbox-service` were still running; whether they belonged to the desktop app is not known. The smoke has no escape check, so a container they might have started could not void it.

**Q1, `apply`: did not land.**

- **Scale:** 25 starts, all on `codex:gpt-6-luna`, none failed, and one cancelled at the stop (`members`' fifth maker start); 8,266,028 input and 138,937 output tokens. The T5 Sol run took 37 starts and 11.8 million input tokens, the T5 Claude run 14 starts and 1.8 million.
- **Parents first held:** Root's maker and checker ran first. Its first submission passed the required `build` and `test` checks, and the children started from its approval.
- **Approved work:** `catalog` 8 of 8 items, after one rejection. `lending` 0 of 9, after 6 maker and 3 checker starts. `members` 0 of 9, after 5 maker starts and one checker start. Root 0 of 4: its checker ran once, on its first submission, and no later Root state reached a checker before the stop. `lending/reminders` never started, since it starts from its parent's approved commit.
- **Why it stalled:**
  - Root's first submission put the whole callable program into `realization/internal/library/library.go`, which Root owns as the rest of the repository.
  - The children built their packages beside it, under their own paths. `lending`'s checker rejected its package three times because the delivered program never reaches it: the callable borrow path in Root's file "independently enforces count >= 5" and does not use the package.
  - `lending` may not change Root's file, and Root would integrate the package only "after the approved Lending package commit is merged". Neither side could move.
  - Root agreed to transfer the shared files to `members` (m13), then kept them (m23), after `members` had already claimed them on its page.

**Q2: half held.** No maker took an accepted push for an approval; each waited for its checker's review. Root still confused the two, for its children:

- In its third start, Root merged the rejected submissions of `members`, `catalog` and `lending`, believing that `lending`'s "remote branch has an approved review commit".
- It told `lending` the package was "merged from the approved origin/managers/lending commit" (m8), which `lending` found untrue.
- The hooks refused both of Root's pushes ("must stay as at the base until you merge 'catalog''s approved work"). In its fourth start Root merged only `catalog`'s approved commit.

So no unapproved state reached a branch, but by the hooks, as in [T5](t5-native-host-20261029.md#windows-with-codex), not by Root.

The stall is a finding for the concept. A child's checker judges whether the delivered program realizes the child's rules. Wiring a child's package into a parent's file is the parent's work after the child's approval. So a check of a child's own area cannot pass while the parent's entry point bypasses it. T5's Sol run showed the same seed, when `lending`'s checker found that Root's executable did not use `lending`'s policy yet. It is the checker's side of what [DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission) settled for the required checks. With Claude the same project landed, in T5 and in the pilot; why those Runs did not meet the deadlock is not examined here. The rule is a question for the Topics session.

**Decided since** ([DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files), 30 October 2026): a child's checker judges its own Package's files against its items, not whether the delivered program already uses them. That is the parent's integration, judged by the parent's checker and gated by the required checks on the merged state. A parent integrates by using its children's work, and its checker rejects a file that builds a child's behavior a second time, as Root's `library.go` did here. The step that builds it, T9 of the [delivery plan](../implementation-plan.md#concept-20-build), runs this smoke again.

**Q3, `verify --full`: correct outcome.** It ended `findings` in 35 seconds. With nothing landed, the target held the model only, and all five checkers gave all 40 items `deviates`, approved; 480,207 input and 7,354 output tokens. The required `build` check reported "the check did not start: fork/exec /bin/sh: no such file or directory": its directory `realization` does not exist on a start that holds only the model, and Go reports a missing working directory that way. The outcome is right, but the message misleads.

**Login, cleanup, post-check.** No start failed on its login. No container, volume or network of either Run remained. The fresh clone of the target holds the model only (`markitect check`: 19 Definitions in 5 Packages), so there was nothing to build. The scan of 107 evidence files for every token of the Codex login and for JWT shapes found nothing.

**Limits.** One attempt, on one host. A smoke shows what happens once, not how often.

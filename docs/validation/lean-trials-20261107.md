# Lean plan step 3: trials of the lean flow, 7 November 2026

Step 3 of the [lean plan](../concepts/lean-tool-plan.md#order-of-work) switched the default driver to the lean commands and released it as 0.19.0 on 7 November 2026. This record is the trials that decide whether step 4, removing the Run, landing and every use of Git in the product, may go ahead. It is bounded evidence on three small changes of one project, not a rate.

## Question

Does the lean flow hold on real changes? `/markitect-apply` injects `markitect checklist --start`, the owner's session starts `markitect-builder`, the builder goes through `markitect gate` and starts `markitect-reviewer` as its own subagent, and the session runs `gate` again and commits the work with the certificate ([project workflow](https://github.com/Glacius-Labs/Markitect/blob/b24f4ed3b8a1bf7f2eebc3aaf805d7e20c787e2b/docs/guides/project-workflow.md#7-apply-and-verify), [without a Run](https://github.com/Glacius-Labs/Markitect/blob/ddfc4d673d8412cc929d0b9cfb3916118bc2df95/internal/run/README.md#without-a-run-gate)). What to watch: the nested reviewer, the hook inside subagents, whether the session leaves the work to the agents, permission stops, and whether gate's row 1 (the spec changed) or row 2 (a changed file in no item) misfire. Second, the same cases through the old flow, `markitect apply`, whose builder goes through `submit`.

## Dispatch

Under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). Written before the first run and approved by the owner through the coordinating session ("ok") on 7 November 2026.

| Item | Value |
|---|---|
| Source | Release binary `markitect-v0.19.0-windows-amd64.exe`, build `922365a5`, SHA-256 `f60fe8904283deaa580a34d3115915dea398ddd5a54bad488608bab38a74fb6c`, equal to the release's checksums file. Fixtures from Main at `b5995358f`: `tools/evaluation-tasks/makerspace`, `tools/makerspace-reference`, `tools/evaluation-judges/plants` |
| Provider | Claude Code 2.1.319 on the owner's stored login, headless `claude -p`, `--permission-mode auto` (the owner's mode), `--setting-sources project --strict-mcp-config`, `git push` and `gh` denied. The session on `claude-sonnet-5-5` at effort high; the agents as their files say (`sonnet`, `high`). A trial hook passed with `--settings`, outside the project, logs every tool call and plants the code defect; it decides nothing |
| Target | Throwaway Git repositories without a remote under `C:\Users\Consiliari\lean-trials\runs\` (see [setup](#setup)) |
| Cases | A, B and C [below](#cases) |
| Runs | Lean arm on A, B and C first; then the old arm on the same cases if the quota allows. At most 8 counted runs; before them at most 2 Haiku probes of the mechanism, each under 5 minutes |
| Limits | 60 minutes per run; at most 12 points on the weekly reading for everything (25 % at the first run). The reading also counts other sessions' use |
| Abort | An agent acting outside its run folder, anything pushed, a login failure, a limit reached, `mcp/*` containers running, or a product defect that blocks a case: recorded with evidence and a proposed fix, that case stopped, no product code fixed |
| Cleanup | No background session: each run is one foreground headless process with a timeout. The trial folders and their work-state files in the user cache folder stay as evidence until this record is merged |
| Evidence | Per run in `C:\Users\Consiliari\lean-trials\runs\<run>`, outside the repository: transcript (stream-json), hook log, plant marker, Git log, work patch, `gate --verify`, `go test`, a grep for the plant, start and end times |

## Setup

Each project is a new Git repository with no remote, built by a script with no agent:

1. The reference tree of the base step (`tools/makerspace-reference/<step>`) as `realization/`, and that step's spec with the registry pages of its reference under `.markitect/`.
2. **Fixture:** a `realization/README.md` with its registry row. The reference trees have none, so `check` fails on the `documentation` Rule and no base could be certified without it.
3. `markitect init`, which writes the harness and commits it.
4. **Fixture:** a base certificate. A script runs `markitect checklist`, answers every item `ok` with a pointer to the first file the item names, and runs `gate`. This is not a review: the reference tree stands in for reviewed work, so that the change gets a list of its own size instead of all 113 Definitions.
5. The target step's spec, with the spec plant if the case has one, committed on top.

`check` still needs Git in 0.19.0, so the folders are repositories; the lean commands themselves read only the folder.

## Cases

| Case | Change | Definitions changed | Plant | Expected |
|---|---|---|---|---|
| A | C5 to C6: certifications no longer expire | 12 | Code: the retired expiry stays in the certification check at 1000 days, planted by the trial hook just before the builder's first `gate` (old arm: `submit`), as if the builder had written it. The tests pass with it; only reading catches it | A review answers `wrong`, the builder repairs, `gate` certifies, the plant is gone |
| B | C2 to C3: bulk discount for materials | 2 | Spec: [`makerspace-c3-discount-contradiction`](https://github.com/Glacius-Labs/Markitect/blob/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-judges/plants/makerspace-c3-discount-contradiction.json), the table gives 8 %, its worked example 5 % | The work stops with the question for the owner; no certificate |
| C | C6 to C7: billing described as one area, no behavior change | 16 | None. Four Definitions go and six come, so registry rows must move | Certified with registry updates and no change in behavior |

## Probes

| Probe | What | Result |
|---|---|---|
| P1 | Haiku: a general-purpose subagent starts one of its own, then runs `markitect checklist --full` | Held. The trial hook loaded beside the project's settings and fired in both subagents with their type and id. The nested subagent ran. The project's hook refused the subagent's `checklist --full` with its message. The session then ran the command itself, which auto mode's classifier denied as a way around the hook |
| P2 | Haiku: `/markitect-apply` headless on a certified folder with no change | **Failed on the trial launcher**, not on Markitect: Git Bash turned the argument `/markitect-apply` into the path `C:/Program Files/Git/markitect-apply`, so no skill started. Fixed with `MSYS_NO_PATHCONV=1`; not repeated, since the dispatch allowed two probes. Lean run A was the first headless start of the skill |

## Runs

Six counted runs, all on 7 November 2026 between 06:54 and 07:04 UTC, one per case and arm, three at a time in separate folders. None failed for a reason outside Markitect, so the two spare runs were not used. The weekly reading on the app's usage card, which counts every session of the account, went from 24 % while the dispatch was written to 25 % before the probes, and stayed at 25 % through the last run (`quota-readings.md` in the evidence folder): all of it, probes included, cost about one point or less. No owner input was needed in any run; each ended on its own.

| Run | Wall time | Items reviewed | Agents | Repair rounds | Plant | End | Output tokens | Cache read | Denials |
|---|---|---|---|---|---|---|---|---|---|
| A lean | 2.8 min | 64, then 39 | builder 1, reviewer 2 (both nested) | 1 | Caught by the first review | Certified and committed; plant gone | 17,292 | 1.05 M | 0 |
| A old | 5.7 min | 64, then 40 | builder 1, reviewer 2 (both nested) | 1 | Caught by the first review | Landed; plant gone | 37,397 | 2.01 M | 0 |
| B lean | 3.2 min | 3, then 25 | builder 2, reviewer 2 (one started by the session, one nested) | 1 | Caught by the second review | Stopped on the spec problem, nothing committed | 15,095 | 0.68 M | 2 |
| B old | 0.9 min | none | builder 1 | 0 | Named by the builder | Builder's `markitect stop`, nothing landed | 3,424 | 0.20 M | 0 |
| C lean | 1.3 min | 21 | builder 1, reviewer 1 (nested) | 0 | None | Certified and committed, registry only | 6,575 | 0.43 M | 0 |
| C old | 2.0 min | 21 | builder 1, reviewer 1 (nested) | 0 | None | Landed, registry only | 9,185 | 0.62 M | 0 |

Tokens are the session's `modelUsage`, which includes the nested agents (the transcripts on disk give the same sums). Old against lean, in output tokens: A 2.2 times, B 0.23 times, C 1.4 times; over the three cases the old arm used 50 k against lean's 39 k, and 2.8 M against 2.2 M cache reads.

**A, the code plant.** In both arms the first review answered `wrong` on five items, each pointing at `ops_members.go:144` and the 1000-day expiry, and said that no test goes past a year or two; the old arm's reviewer also reproduced it by running the program. Each builder removed the line, the second review passed, and the committed code has no expiry. The lean builder repaired before it called `gate` again, so `gate` asked only for the 15 items whose files the repair touched, though its second reviewer answered 39; the Result records one repair round. Both first reviews had 64 items. The old arm's reviews cost about three times the output tokens of lean's, mostly in its first review, which ran the program to reproduce the bug and twice broke its own shell quoting before its answers were stored.

**B, the spec plant.** Both arms ended without a certificate or a landing, with the contradiction put to the owner in their own words. Lean took the long way, finding 1 below: the first builder stopped and said why, the session ran `gate`, the first reviewer answered `wrong` because nothing was built, the second builder built the table's 8 %, and the second reviewer answered `spec problem` on both items. The old builder ended the Run itself with `markitect stop` and a reason, which no reviewer checked.

**C, the clean change.** Both builders rewrote the registry rows of the four removed Definitions to the new ones, changed no code and no test, and the reviewers answered all 21 items, the 4 removed ones included, `ok`.

## What the flow showed

**The nested reviewer works.** Each time `gate` or `submit` told a builder to start `markitect-reviewer`, it started it as its own subagent and waited for it: seven reviews in five runs, each recorded in its transcript at spawn depth 2 with the builder as its parent and a foreground start. Lean A and old A did so twice, around one repair. The one other review, lean B's first, was started by the session after the builder had ended. Nothing had to be configured.

**The session left the work to the agents.** In the three lean runs the owner's session ran only `gate`, `git status` and the commit, plus in A one read-only look at the line the builder had called foreign; it edited no file and stored no answer. Where the builder ended without passing `gate` (lean B), the session ran `gate` and started the agent it named, the reviewer once and the builder once, as the skill says.

**The hooks fire inside subagents.** The trial hook saw every call of the builder and of the nested reviewer with its agent type. The project's hook refused a subagent's `checklist --full` (probe P1). No agent in the six runs called anything the hook refuses, so it had nothing else to refuse. The agents call `markitect` as `cd <folder> && markitect …`; by the hooks documentation an `if` filter matches when any subcommand matches, so such a call still reaches the hook. One false refusal was seen outside the runs: in the session that ran these trials, the plugin's hook refused an analysis command that only mentioned `markitect answer` inside a Python string, since the hook reads the text when Claude Code cannot split a command.

**Permission stops did not stop a run.** All agents ran in the session's `auto` mode; the builder file's `permissionMode: acceptEdits` did not apply. Five runs had no denial. In lean B, auto mode's classifier twice refused the first builder's attempt to undo its own uncommitted edits with Git (`git checkout realization`, `git apply -R`) as irreversible local destruction; the builder put the two files back by hand. A run in manual mode was not tried.

**Gate's rows 1 and 2 did not misfire.** In C the builder changed only registry pages, which row 1 allows, and no file outside the list changed in any run, so row 2 had nothing to name. After the review, lean C's builder turned the registry back to CRLF line endings (its first edit had written LF); the answers stood, since a text file's hash ignores line endings.

**The certificate travels as designed.** Each certified run committed the work, the certificate and its Result in one commit, and `gate --verify` afterwards found the folder matching it. The Result records the repair round (lean A: 1).

## Findings

| # | Finding | Evidence | Proposed |
|---|---|---|---|
| 1 | **A builder has no door for a spec problem.** The builder's task says to stop and tell the owner when the spec contradicts itself. Lean B's first builder did so at once, in a clear report, but the session follows `gate`, which knew nothing of it and asked for a review. The first reviewer answered `wrong` (nothing built) instead of `spec problem`, though its own answer quoted both numbers. That used a repair round, the second builder built the table's 8 %, and only the second review stopped the work. The end was right, at 3.5 times the old flow's time and 4.4 times its output tokens, with code for one reading left in the folder. In the old flow the builder ended the Run itself with `markitect stop WHY` in under a minute, unconfirmed by any reviewer | `runs/B-lean`, `runs/B-run` | Way 2 [below](#for-the-owner), chosen by the owner on 7 November 2026 |
| 2 | `answer` refuses a missing pointer with "does not exist in this commit", also without a Run, where there is no commit | Lean B's second review; [`internal/run/answers.go`](https://github.com/Glacius-Labs/Markitect/blob/b24f4ed3b8a1bf7f2eebc3aaf805d7e20c787e2b/internal/run/answers.go) | Say "in the folder" when no Run is open; goes with step 4 |
| 3 | `impact --base CERTIFICATE` heads its page with the certificate path cut to 12 characters ("Impact since .markitect/r"), the length of a short commit hash | [`internal/adapters/cli/render.go`](../../internal/adapters/cli/render.go) `short` | Shorten only commit hashes; goes with step 4 |
| 4 | Lean B left the work of the second builder uncommitted in the folder, with the spec-problem Result beside it, and old B left its builder's work the same way. Lean B's tests pass, since its test expects the table's 1831; old B's new test fails against the example. Neither is wrong, but the owner meets someone's partial work for one reading of the spec | `runs/B-lean/status-after.txt`, `runs/B-run/status-after.txt` | Nothing now; part of finding 1 |

**Trial artifacts, not findings.**

- The plant script wrote `ops_members.go` with Windows line endings, so the planted line came with a whole-file line-ending change that the builder saw in its diff and called foreign; the reviewers' answers name the content, `day < issuedOn+1000`, not the endings. Both arms met the same artifact.
- The plant's trigger first matched only a command beginning with `markitect gate`, but the agents write `cd <folder> && markitect gate`. It was widened during lean A at 06:54:30, before that builder's first `gate` at 06:54:45; the hook script is read at each call, so the plant landed where intended in both A runs.
- The old arm had no skill file: 0.19.0 installs the lean skill. Its session got the 0.18.0 skill's text as the prompt and ran `markitect apply --skill --base <base commit>` itself, where the skill had injected the output. `--base` keeps the old list to the same change, since without a landed Run `apply` lists every Definition.

## For the owner

**How should a builder report a spec problem in the lean flow?** Step 4 removes `markitect stop`, the old builder's door. Three ways:

1. **The builder's word stops the work**, as `stop` did: cheapest, but no reviewer checks it, so a builder could stop to avoid work.
2. **Recommended: the builder records the problem with the tool, and `gate` has a reviewer confirm it** before it tells the owner, without counting a repair round. It keeps "no agent decides about its own work" and would have saved lean B a builder and a review.
3. **Leave it:** the end is right, at about four times the old flow's tokens on such a change.

**Decided on 7 November 2026:** the owner chose way 2, the recommended one. It is decision 7 of the [lean plan](../concepts/lean-tool-plan.md#decisions-for-the-owner), and step 4 brings it in before `stop` goes.

## Verdict

**The lean flow held, and step 4 can go ahead.** It waited on one question, how a builder reports a spec problem (finding 1), since step 4 removes the old door; the owner settled it on 7 November 2026 with way 2 [above](#for-the-owner), so step 4 brings in the builder's report, confirmed by a reviewer, before `stop` goes. Every mechanism the flow relies on worked without help: the skill headless, the nested reviewer, the hooks inside subagents, the session leaving the work to the agents, `gate`'s rows, the certificate committed with the work, a repair, and a stop on a spec problem. The planted bug was caught by reading in both arms, and the lean arm used fewer tokens on two of three cases.

**Not shown.** One run per case and arm, on small changes of one project, on Windows, with Sonnet, in `auto` mode and headless; no run in manual mode, so how often a person is asked is open. The base certificates are fixtures. No real branch, merge, crash or second certified branch was met, and no run used the two repair rounds up.

# U1 and U2: the Library smoke on the token, 4 November 2026

Steps U1 and U2 of the [delivery plan](../implementation-plan.md#concept-30-build): the default of [concept 3.0](../concepts/v3-concept.md) on real sessions, natively on Windows and on Linux. The Library's change landed on both. This is bounded evidence on a small project, not acceptance.

## Questions

- **Q1:** does `markitect apply --full` on the Library's M1 land natively on Windows, on the user's token, with the required checks run as the third user?
- **Q2:** does the change to M2, the loan limit from 5 to 3, land with a plain `markitect apply`, with a list of what it reaches and a Result that shows how much checking happened?
- **Q3:** does `markitect verify --full` run one reviewer per Package, and end with a Result on its own branch?
- **Q4:** does the launch hold on a real session? `claude auth status` reports the token, the launch's first line passes the adapter's check, subagents run on the session's model and no other model answers.
- **Q5:** does a project's own `.claude/settings.json` hook and `.mcp.json` server stay out of both roles' sessions? The target project carries both, planted. The hook would leave a mark in the project folder and in `/tmp`. The server would show in the launch's report, and the adapter would refuse it.
- **Q6:** the same as Q1 to Q5 natively on Linux, in WSL, on the same engine.

Without a provider, the [real-engine tests](../../internal/adapters/sandbox/command_real_test.go) already hold the gate's planted refusals on the real container: a changed spec, a changed `execution.yaml`, a failing check, a check that looks for the token or the sessions' leavings, and work that holds the token.

## Dispatch

| Item | Value |
|---|---|
| Source | Branch `dev/u1-default` at the head the review accepts. Each attempt names its commit |
| Images | `packaging/runtime` at the source, then its Go example on it, named by engine image ID: Claude Code 2.1.308 |
| Project | A fresh repository: the Library's M1 from `tools/evaluation-tasks/library` as `.markitect/` with its registry pages, a planted `.claude/settings.json` and `.mcp.json` on the ignore page, and `markitect check` passing. The target is `main` of a local bare repository |
| Commands | `markitect apply --full --deadline 60m`; then the model at M2, committed, and `markitect apply --deadline 30m`; then `markitect verify --full --deadline 30m` |
| Provider | `claude-sonnet-5-5` at effort `high`, the session's subagents forced onto it |
| Login | The owner's token from `claude setup-token`, in `MARKITECT_CLAUDE_TOKEN` in the process's environment only. Every other `ANTHROPIC_*` and `CLAUDE_*` variable is cleared |
| Limits | One attempt per host, a second only after a fix of a defect the first one shows. The container gets 4 CPUs, 8 GiB, 2,048 processes and 1 GiB of temporary space |
| Abort | A login or launch-check failure, the token found in the evidence, or a Command past its deadline ends the attempt, which the Command reports as incomplete |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, created exclusively before the first engine call and removed after the last command. No other Docker work runs meanwhile |
| Cleanup | The Command removes its container and volumes; afterwards nothing labelled with its Run may remain |
| Evidence | Outside the repository: the Results, the transcripts, standard error, the target's refs and log. A post-check builds, vets and tests the landed state in a fresh clone, and scans all evidence for the token and Anthropic key shapes |

## Results

Three attempts, within the dispatch, all with `claude-sonnet-5-5` at effort `high` on the owner's token. The token was found in no evidence, and nothing labelled with a Run remained after any attempt.

| Attempt | Source | Command | Time | State | Repair rounds | Items |
|---|---|---|---|---|---|---|
| Windows 1 | `a9981e2c2` | `apply --full` on M1 | 225 s | incomplete | 0 | 19 |
| Windows 2 | `9d3a52e21` | `apply --full` on M1 | 283 s | landed | 1 | 19 |
| | | `apply` on M2 | 59 s | landed | 0 | 6 |
| | | `verify --full` | 67 s | verified | 0 | 19 |
| Linux 1 | `333fbf81c` | `apply --full` on M1 | 199 s | landed | 0 | 19 |
| | | `apply` on M2 | 59 s | landed | 0 | 6 |
| | | `verify --full` | 60 s | verified | 0 | 19 |

**The failed attempt.** Windows 1 built the project and passed the gate. It stopped at the reviewers' answers: the parser took any remark in brackets for a test's name, so the Command refused real answers, and they stayed refused after the one request to correct them. Nothing landed, and the work and its Result stayed on the Run's branch. The fix reads a test's name as one word, and takes several names, remarks and line ranges. The same attempt ran before the review's fixes to the engine, and Windows 2 and Linux 1 ran after them.

**Q1 to Q3: yes, on both hosts.** After M2, the landed code has the loan limit of 3. On Windows, its build, `go vet` and `go test` passed in a fresh clone. On Linux, the required checks ran in the container, and there was no such post-check on the host. The independent Library oracle was not run.

**Q4: yes, as far as this shows.** `claude auth status` reported the token, and every session's first line passed the adapter's check. Only `claude-sonnet-5-5` answered in any session. No session started a subagent, so the forcing itself is still untested.

**Q5: yes.** The project carried a planted hook and a planted MCP server from its first commit. No session's transcript shows a hook event or an MCP server, and none of the hook's marks is in the landed tree.

**Q6: yes.**

**How much checking the transcripts show.** The reviewers read through shell commands: on Windows 2, M1's reviewers ran 79 commands and opened 2 files through the `Read` tool. The Command counted a pointer as opened when a command named its file, 57 of 59 there, and said "not measured" for the rest. That shows that someone looked, not that they judged correctly, and a command that names a file need not have read it.

**What this does not show.** No session on this small project needed a subagent or broke a rule on its own, so the gate's refusals come from the scripted [real-engine tests](../../internal/adapters/sandbox/command_real_test.go) and not from these runs. Nothing here tests the 5× project or a reviewer against known bugs; that is [test 1](../concepts/v3-concept.md#13-how-we-know-it-works).

# The default through the suite, and its first build at 5×, 4 November 2026

Step U4 of the [delivery plan](../implementation-plan.md#concept-30-build) starts here: the default of [concept 3.0](../concepts/v3-concept.md) run as the suite's [arm](../../tools/evaluation/README.md#the-concept-30-arm), on real sessions, with the hidden cases judging the result. Test 1 and test 2 both start from the default's own W0 of the [large Makerspace](../../tools/makerspace-large/README.md), so that build comes first. The default built the 5× project on its first launch and passed all 327 hidden cases. This is one launch, not a rate.

## Questions

- **Q1:** does the arm run the Command through the suite on real sessions: landing, delivering, passing the suite's escape check, with the token nowhere in the evidence?
- **Q2:** does the Library's chain M1 to M2 land and pass its hidden cases that way?
- **Q3:** does `markitect apply --full` build the large Makerspace's W0? How many of the 327 hidden cases pass, how long does it take and what does it cost against K, which took 37 minutes on Sonnet?
- **Q4:** what does the Result show at this size: repair rounds, reviewers, how many pointers were opened, what went back?

## Dispatch

| Item | Part A: the Library | Part B: the large Makerspace's W0 |
|---|---|---|
| Launches | One chain, M1 then M2 | One launch, `apply --full` |
| Deadline | 2,700 seconds per step | 14,400 seconds |
| Expected use | Under a point | 3 to 8 points on the runner's weekly reading |
| Relaunches | One after a void launch or a fix of a defect the first shows | One after a void launch; a second only with a fix |

For both:

| Item | Value |
|---|---|
| Source | Branch `dev/u4-default-arm` at the commit that adds this page, frozen with `git archive` into the evidence folder, with `markitect` built from it |
| Image | `markitect-runtime-go:u1-dev`, `sha256:4f4da6c2687a65f81702b7e0349ef8dec990ea2bcaab6efcc139225cdb8a42bc`: `packaging/runtime` and its Go example, with Claude Code 2.1.308 and Go 1.27.1 |
| Sandbox image, for the suite's gateway | `sha256:67e2cb76d76c27fd20b183e8d209e6407c49f758c35a454168b5b37cc2e24ce8` |
| Provider | `claude-sonnet-5-5` at effort `high`, the session's subagents forced onto it, and the host refuses any other model |
| Login | The owner's token, in `MARKITECT_CLAUDE_TOKEN` in the runner's own environment only, read from the user's registry by the launcher and never printed or written. Every other `ANTHROPIC_*` and `CLAUDE_*` variable is cleared for the host |
| Limits | The Command's container: 4 CPUs, 8 GiB, 2,048 processes, 1 GiB of temporary space. A session's limit is 3 hours. The runner's own stop at a weekly reading of 90 |
| Abort | A login failure, the token found in the evidence, an escape, a Command past its deadline, or the weekly rise past 10 points, at which the runner is stopped by hand |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, taken by the suite. Other Docker work is checked first and stays off |
| Cleanup | The Command removes its container and volumes and the suite its network and gateway; afterwards nothing labelled with the evaluation may remain |
| Evidence | Outside the repository, in `%LOCALAPPDATA%/markitect-default/`: the runner's work folder. Part B's launch also goes into the [lasting store](../../tools/evaluation/kept.py) |

## Results

All from the frozen source `4db73a3bef0c`, on `claude-sonnet-5-5` at effort `high`. Each launch was valid for the suite: no escape, nothing left behind, nothing quarantined, and no credential in the evidence. Only Sonnet answered in any session.

| Launch | Time | State | Hidden cases | Repair rounds | Sessions |
|---|---|---|---|---|---|
| Library M1, `apply --full` | 231 seconds | landed | 19 of 19 | 0 | 1 implementer, 6 reviewers |
| Library M2, `apply` | 215 seconds | landed | 19 of 19 | 0 | 1 implementer, 1 reviewer |
| Large Makerspace W0, `apply --full` | 58 minutes | landed | 327 of 327 | 1 | 2 implementer, 51 reviewers |

**Failed starts.** The Library's runner stopped once before any session, on a path mistyped in its configuration. It was fixed and restarted.

**Q1 to Q3: yes, on these launches.** The arm drove the Command through the suite on real sessions.

**Q3, the cost.** 3.43 times K's W0 on Sonnet, in the provider's own relative figure, and 58 minutes against K's 37. The implementer cost 1.88 times K and the 51 reviewers 1.56 times. S's W0 cost 4.76 times K. The weekly reading rose from 0 to 2. That window is the token's, which resets on 11 November at 09:00, and it is not the window of the runner's login that the earlier launches report, so the 2 cannot be set beside their figures. The implementer took 43 minutes, and the 51 reviewers spent 69 session-minutes, 8 at a time.

**A correction.** This page first said 5.32 times K. That sum counted a resumed session's running total a second time: a repair continues the implementer's session, and its closing record repeats everything before it. A review of test 1's tools found this. The kept launch's record still says 49.21, since a kept launch never changes; [`kept.py`](../../tools/evaluation/kept.py) now counts each session once, and the [page builder](../../tools/evaluation/side_by_side.py) counts a kept Command's cost again from its transcripts. S's and K's figures are unchanged: none of their sessions resumes.

**Q4, what the Result shows.**

- **A reviewer found a real bug.** The `protocol` reviewer ran the program and saw that trimming a request line before checking it accepted a line with a form feed or a non-breaking space around it, where the requirements say a line that is not valid JSON is invalid. It went back, and the 16-second repair passed the review of what changed. Whether a hidden case holds that line was not checked, so this says the reviewer found a real departure from the requirements, not that the cases would have missed it.
- **A reviewer said what no test shows.** `consumables/consumption` is `not shown`: the reviewer took out the code that rounds up, in a copy, and the program's whole test suite still passed. It landed and is on the Result, as the rule says.
- **How much checking the transcripts show.** The reviewers ran 749 commands and opened 28 files through `Read`. Of their 1,351 pointers, the Command counted 1,326 as opened, 25 as not measured, and none as repeated answers. A command that names a file need not have read it.
- **Registry.** The Result lists 366 missing links: a reviewer's pointer names a file that the registry does not list for that item, mostly a test file. They don't refuse a landing. Whether they matter shows in the later steps' impact.

**How it compares,** with [the first page](large-comparison-20261104.md)'s method, not yet on a page of its own: 8,713 program lines, the smallest of the seven builds, and 11,519 lines of tests with 317 test functions, between K's 101 and the others' 590 to 686. Renamed repeats are 3.1 %, near K's 2.0 %. All 48 modules are at their paths, and no import breaks the spec's table, in the program or its tests. It is in the lasting store as `large-w0-default-sonnet`.

**What this does not show.** This is one launch. Whether the default holds up when changes come is test 2, and whether its reviewers catch known bugs at this size is test 1.

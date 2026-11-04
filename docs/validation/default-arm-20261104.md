# The default through the suite, and its first build at 5×, 4 November 2026

Step U4 of the [delivery plan](../implementation-plan.md#concept-30-build) starts here: the default of [concept 3.0](../concepts/v3-concept.md) run as the suite's [arm](../../tools/evaluation/README.md#the-concept-30-arm), on real sessions, with the hidden cases judging the result. Test 1 and test 2 both start from the default's own W0 of the [large Makerspace](../../tools/makerspace-large/README.md), so that build comes first. No result is recorded yet.

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

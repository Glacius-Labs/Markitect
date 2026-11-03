# The probe of the large Makerspace: dispatch — 3 November 2026

The first launches on the [large Makerspace](../../tools/makerspace-large/README.md), which the owner chose on 3 November 2026 so that the approaches can be tested at a size that strains one session: 48 modules, 188 operations, 413 model pages, 327 hidden cases. K first, then S, on W0, one launch each. **Public, not counted, and it proves nothing on its own.** The new sealed project stays sealed, and concept 2.0 stays as it is.

**Status: approved by Build on 3 November 2026, under the standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). The owner agreed to the estimate and, on the same day, to starting K with the stop named below.** S has not been approved to start: it waits until the owner has seen what K cost.

## What it has to show

1. **Can one session build it at all?** Landed, the hidden cases per class, and whether the deadline holds.
2. **Does size strain a session?** The largest context per session against the window of 1,000,000 tokens, how often it compacted, and whether failures sit in the cross-module cases rather than the simple ones.
3. **Do makers make bugs a checker could catch?** On Makerspace no maker's work has failed a hidden case. S's checker runs over all items here, and K's result can be read against the same cases, and reviewed blind.
4. **What does it cost,** in weekly points, wall time and multiples of K's W0 on this project? Next to Makerspace's K, whose W0 took 12 minutes and about 0.1 point.

Whether F and M should follow depends on the answers: only if K or S strains.

## Dispatch

| Field | Value |
|---|---|
| Source | Commit `45ead2dbc4247058b647a942935603b364bab186`, runner `pilot.py` in Build's notes folder, SHA-256 prefix `24c2920df83eda97` |
| Provider | `claude-sonnet-5-5`, effort high, the owner's subscription login through the CLI, never an API key |
| Launches | K, then S, on `makerspace-large` W0, 1 launch each, in the suite's sandbox. Relaunches 2, bound 4, the runner's rules. The runner holds S until a `GO-S` file exists in the work folder, which Build writes only once the owner has seen K's price |
| Deadline | 14,400 s per launch, against Makerspace's 5,400 s |
| Blind reviews | If K lands: `review_arm.py --answers blind` over K's result, up to 3. KC's maker is K, so this gives K with a checker at this size without another build, on real work. Only after the owner has seen what K cost |
| Judge | `tools/evaluation-judges/makerspace-large`, exported from the source. Its self-test in the container comes first: the reference and 4 mutants, 5 judgments |
| Size | The estimate given to the owner: 2 to 6 weekly points for the two launches, on the runner's seven-day reading. That reading is not the owner's usage card: on 3 November the card read 59 % where the runner read 12 %, with different resets, and the two do not convert. So the stop is on the card, which read 59 % before K |
| Stop on the card | Build reads the card at least every 30 minutes and stops K and tells the owner if it reaches 75 %. The card counts all of the account's work, not this launch alone, so the price of K is read from the launch's own cost as well |
| Abort | The runner's stops, a `STOP` file before the next launch, the stop on the card, or the owner's word. A running launch is stopped by ending the runner's process tree and removing what carries the run's labels |
| Cleanup | The runner's final check: no container, volume or network left, no login in the evidence |
| Evidence | `%LOCALAPPDATA%\markitect-large-probe\work`, on this machine |

## The reading rule, named before the first launch

For each launch: landed or not, and why; failed hidden cases per class (Makerspace's own, G, B, X), with a list of the failures read by hand; wall time; weekly points; multiples of K's W0; the largest context and the compactions of each session; for S, the refusals, the findings and whether the checker refused anything real.

A failed case is read before it is believed: the cases were written by people too, and a failure may show a fault in the case. A case found wrong is corrected in the open, and both counts are shown.

**Limits:** one launch of each arm is a look, not a measure. Run to run spread on Makerspace W0 was two to one. The model and the cases were written by agents in a few hours, and the seeded bugs that measure the cases were chosen by one author.

## Reading

An amendment to this page, for the owner.

## Amendments

### Amendment 1: K's reading, 3 November 2026

**How it ran.** One launch, a result on its first try, from 15:41 to 16:18 UTC. The final check found no secret and no leftover. The judgment is valid: it took 13.8 s, against 11.3 s for the reference in the self-test. The session read only `TASK.md`, and nothing escaped the sandbox.

**K built the whole project in one session, with no hidden case failed.** Multiples are of K's W0 on Makerspace (12 minutes, 2.0755 relative).

| Question | K |
|---|---|
| 1. Can one session build it? | Landed. Failed 0 of 327: Makerspace's own 0 of 54, generated 0 of 151, by hand 0 of 90, across modules 0 of 32. No unrequested change, no stale file, build and tests pass |
| 2. Does size strain it? | Not yet. Largest context 554,435 of 1,000,000 tokens, no compaction, 98 turns. Of 66 builds and test runs, 2 failed |
| 3. Do makers make bugs a checker could catch? | None that a hidden case sees. The blind reviews would show anything else |
| 4. What does it cost? | 37 minutes and 9.25 relative, so 3.1 times the wall time and 4.46 times the cost of K's W0 on Makerspace. Neither weekly reading moved: the card read 59 % before and after, the runner 12 % before and after |

**Reading.**

- **K's cost grew with the size and no faster.** The project has 5.3 times the modules and 5.1 times the operations, at 4.46 times the cost.
- **This size does not strain a session.** On this one launch, the window would be reached at about nine times Makerspace rather than five.
- **Zero failures does not mean the cases are blind.** They caught all 89 seeded bugs, though one author chose those bugs. A more likely reason is the task: a document of 53,000 words that states every rule, built from nothing. K's failures on Makerspace came on changes, and this project has no change steps yet.

**The card may not count these runs at all.** The runner's login says Max 20x and the card says Ultra, and the two readings have different resets. That points to two separate quota pools, but it does not prove it. If they are separate, the stop on the card guarded nothing here and the runner's own stop at 90 % of its reading did. K's relative cost is the price to trust.

**Limits:** as above. One launch is a look, and K's W0 on Makerspace has itself varied.

### Amendment 2: the blind reviews and S are approved, 3 November 2026

The owner has seen K's price and agreed to the blind reviews of K's result, and then S.

| Field | Value |
|---|---|
| Reviews | 3 launches of `review_arm.py --answers blind` over K's result merged onto the owner's commit, one at a time. Relaunches 2, bound 4, deadline 3,600 s each |
| Checklist | The 413 items that impact gives for W0 |
| Source | The same commit `45ead2dbc4247058b647a942935603b364bab186`, in its own work folder `%LOCALAPPDATA%\markitect-large-reviews\work`, so that the probe's frozen folder stays as it is |
| Runner | `reviews.py` in Build's notes folder, SHA-256 prefix `8f8934c38742acfb`; the task comes from `k_w0.py`, prefix `0eeeb27e8f932e2f` |
| Size | About half of K's build per review, by the ratio on Makerspace |
| S | Started after the reviews have been read, by writing `GO-S` in the probe's folder and running its runner again. Its size: about 2 to 3.5 of K's build here, by the ratios on Makerspace, within the deadline of 14,400 s |
| Stops | As in the dispatch: the runner's own, a `STOP` file before the next launch, the stop on the card at 75 %, or the owner's word |

### Amendment 3: the stop on the card is lifted, 3 November 2026

The owner lifted the stop at 75 % on the card, for the reviews and S: they may use what they need, and reaching the limit is acceptable. The runner's own stops stay, among them its stop at 90 % of its seven-day reading, since they are part of the frozen tools. A run that hits the limit is recorded as what it is, a launch stopped by quota, and says nothing about the arm.

### Amendment 4: K on Haiku after S, 3 November 2026

The owner approved three launches of K on Haiku on this project, after S, as the first look at whether the method holds on a far cheaper model. The reviews and S stay on Sonnet, so that K and S are compared on one model.

| Field | Value |
|---|---|
| Launches | K on `makerspace-large` W0, 3 launches one after another, in the suite's sandbox, effort high as K on Sonnet. Relaunches 2, bound 6 |
| Provider | `claude-haiku-5-5`, the owner's subscription login through the CLI, never an API key |
| Source | The same commit `45ead2dbc4247058b647a942935603b364bab186`, in its own work folder `%LOCALAPPDATA%\markitect-large-haiku\work` |
| Runner | `haiku.py` in Build's notes folder, SHA-256 prefix `c1454f9c8634f6ae`, over the probe's `pilot.py` |
| Deadline, judge | As for K on Sonnet: 14,400 s, the same task and the same 327 hidden cases |
| Size | By list prices at most about a quarter of K's Sonnet build per launch, though whether quota follows those prices is what this measures |
| Reading | As in the rule above, for each launch, with cost and time as multiples of K's W0 on Sonnet on this project (9.25 relative, 37 minutes), and whether the three agree. Three launches are a first look at reliability, not a measure of it |

### Amendment 5: the blind reviews of K's result, 3 November 2026

**How they ran.** Three launches, each a result on its first try, from 17:32 to 18:04 UTC. The final check found no secret and no leftover, and no review read `/variant`. Build read every answer other than `confirmed`, and also measured how deep each review went. Multiples are of K's W0 on Sonnet on this project (9.25 relative, 37 minutes).

| Review | Answers over 413 items | Time | Cost | Largest context | Go files read | Evidence per item, median |
|---|---|---|---|---|---|---|
| r1 | 413 confirmed | 14 min | 0.53 | 467,847 | 63 | 459 characters |
| r2 | 412 confirmed, 1 waived | 5 min | 0.29 | 317,559 | 10 | 222 characters |
| r3 | 413 confirmed | 13 min | 0.55 | 480,780 | 63 | 462 characters |

**No review found a fault in K's behaviour, which agrees with the 327 hidden cases.** The one departure any review named is small and real: K's README says the requirements are in `../TASK.md`, a file that is not in the repository. Only r2 noticed it, and it waived it.

**The reviews are not all worth the same.** r1 and r3 read all of the code, drove the program with their own scenarios and wrote evidence specific to each item. r2 read 10 of K's Go files and repeated one sentence of evidence over many items, claiming to have exercised items it could not have reached that way. So a review's answers alone do not show how much checking stands behind them.

**This does not yet show that a checker still catches bugs at this size.** All three reviews found the work clean, and the work may well be clean, so they had nothing to catch. On Makerspace blind reviews caught 17 of 18 planted bugs. At this size no review has met a known bug. Real work with real failures would test that, and K's launches on Haiku may provide it.

**The card** read 59 % before the reviews and 60 % after, and the runner's reading went from 12 % to 13 %. This session's own work also counts on the card, so that does not settle whether the two are separate pools.

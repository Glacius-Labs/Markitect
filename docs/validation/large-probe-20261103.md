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

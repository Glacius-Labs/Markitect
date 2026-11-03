# K plus impact, the gate and a blind checker: dispatch — 3 November 2026

A small test, asked for on 3 November 2026 after the [checker test](checker-test-20261103.md), under the standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). On Makerspace W0, S's item-by-item maker costs 1.8 to 3.5 times K's W0 and its checker about 0.5. On c3 and c5 the maker found the model problem and the checker only confirmed it. This test takes that maker away, to show what it really buys. **These runs are not counted and prove nothing on their own.** The new project stays sealed, and concept 2.0 stays as it is.

**Status: approved by Build on 3 November 2026, before the first launch.** Nothing has run yet.

## The version: arm KC

`tools/evaluation/one_agent_arm.py --shape plain`, in the suite's sandbox, as K:

1. **Impact** before it, as S has it. Nothing reached means no agent starts.
2. **Make:** K's session, with K's prompt word for word and K's `TASK.md`, in a checkout with no model and no `markitect` program. It writes no verdict.
3. **Gate:** the required checks on the whole repository, and nothing in `.markitect/` touched. `markitect check` is not part of it, since K's work has no registry rows.
4. **One fresh checker** over the items impact reached, in a repository that holds the model. It sees no claims from the maker. It answers each item `confirmed`, `wrong`, `not-shown`, `waived` or `model-problem`. Only `wrong` sends the work back. `not-shown` and `waived` land as findings. **`model-problem` ends the Run without landing**, and being able to name one itself is the main new thing here.

A round after a refusal is a fresh session of K's prompt in the same folder, with the reasons appended. Up to 6 attempts, as S and F.

## What it has to answer

1. **Does the checker stop a contradiction K would land?** Two launches of c3 may never show K landing, so the checker also reviews K's real landed c3 r2 from the counted series, three times. That is the only real wrong work in the series that a checker can review. M's rows with failures did not land, so they hold no submission. K's C9 failures are three census cases at a step that must change nothing, which no item shows, and impact stops the arm there.
2. **Does it still catch the six planted bugs without the maker's claims?** The same planted submission as the [checker test](checker-test-20261103.md), three reviews, no verdicts in the prompt.
3. **Does it stop good work?** The clean submission, three reviews; c6 and W0, two launches each.
4. **What does W0 cost,** against about 1.5 of K's W0?

## Dispatch

| Field | Value |
|---|---|
| Source | Commit `17bcfdc640cf67536df622094daf1d4c97d79f2f`, runner `pilot.py` in Build's notes folder, SHA-256 prefix `f8706c6f512a18dd` |
| Provider | `claude-sonnet-5-5`, effort high, the owner's subscription login through the CLI, never an API key |
| Reviews, arm B | `review_arm.py --answers blind`: planted, clean and K's c3 r2, 3 each, 9 in all. Gate: the required checks alone. Deadline 1,800 s |
| Launches, arm KC | C9 r1; c3, c5, c6 and W0, r1 and r2; 9 in all. Deadline 5,400 s, as the series. Judges: the series' |
| Order | The reviews first, turn about; then C9, then r1 of c3, c5, c6, W0, then r2 |
| Bound | 18 launches, bound 27, relaunches 2, the runner's rules |
| Size | About 1 to 2 weekly points. Build stops and tells the owner above 3 points of seven-day use over the first reading; the page authors of the [large project](../../tools/makerspace-large/DESIGN.md) write beside it and share that reading, and start no container |
| Abort | The runner's stops, a `STOP` file, or the owner's word |
| Cleanup | The runner's final check: no container, volume or network left, no login in the evidence |
| Evidence | `%LOCALAPPDATA%\markitect-k-checker\work`, on this machine. The counted series' folder is read only |

K's c3 r2 as a review: K's result commit merged onto that plant's owner commit, so the diff is K's work and the checklist is what the arm's impact gives.

## The reading rule, named before the first launch

For each review of the planted submission and each bug, one of these, read by hand:

- **Catch:** `wrong` on any item whose evidence names the planted behavior or its place. Only this counts. A `model-problem` answer that names a bug is listed apart.
- **Noticed:** `not-shown`, `waived` or `confirmed` whose evidence names it, and no catch.
- **Missed:** no answer names it.

Then:

- **On K's c3 r2:** does some answer call the discount contradiction a `model-problem`? `wrong` naming the contradiction is listed apart: in the arm it would go back to the maker rather than stop the Run.
- **False alarm on the clean submission:** a `wrong` or a `model-problem`. Each is quoted.
- **Launches:** c3 and c5 must stop without landing. c6 and W0 must land. C9 must start no agent. Each landing is read beside its hidden cases, and each refusal or stop is quoted.
- **Cost:** in weekly points, wall time and multiples of K's W0, unless a row names another K.

A review whose session reads `/variant` is marked.

**Limits:** one planted submission with six bugs in it, as before; three reviews per cell and two launches per case show a tendency, not a rate; K's c3 r2 is one piece of real wrong work.

## Reading

An amendment to this page, for the owner.

## Amendments

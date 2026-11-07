# K plus impact, the gate and a blind checker: dispatch — 3 November 2026

> The arms this record ran, `one_agent_arm.py` and `review_arm.py`, are not on Main. [Concept 3.0](../concepts/v3-concept.md#13-how-we-know-it-works) ends these comparisons, so their code stays at commit `4d1d25ffb` on the branch `dev/makerspace-large`.

A small test, asked for on 3 November 2026 after the [checker test](checker-test-20261103.md), under the standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). On Makerspace W0, S's item-by-item maker costs 1.8 to 3.5 times K's W0 and its checker about 0.5. On c3 and c5 the maker found the model problem and the checker only confirmed it. This test takes that maker away, to show what it really buys. **These runs are not counted and prove nothing on their own.** The new project stays sealed, and concept 2.0 stays as it is.

**Status: approved by Build on 3 November 2026, before the first launch, and run the same night.** The reading is [Amendment 1](#amendment-1-the-reading-3-november-2026).

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
| Size | About 1 to 2 weekly points. Build stops and tells the owner above 3 points of seven-day use over the first reading; the page authors of the [large project](https://github.com/Glacius-Labs/Markitect/blob/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/makerspace-large/DESIGN.md) write beside it and share that reading, and start no container |
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

### Amendment 1: the reading, 3 November 2026

**How it ran.** All 18 launches gave a result on their first try, between 03:45 and 05:12 UTC. The final check found no secret and no leftover. No review read `/variant`. Seven-day use rose from 10 % to 12 %, but the large project's authors wrote beside it and share that reading: by relative cost this test is about 1 to 1.5 points of it. Build read every answer other than `confirmed` by hand. Multiples below are of K's W0 (12 minutes), unless a row names another K.

**KC did what it was meant to on every launch, and a blind checker is a little weaker than one that sees the maker's claims.**

| Question | Result |
|---|---|
| 1. Does it stop a contradiction? | c3 stopped both launches. On K's real landed c3 r2, 2 of 3 reviews named the model problem |
| 2. Does it catch the planted bugs without claims? | 17 of 18 catches, against 18 of 18 with claims |
| 3. Does it stop good work? | No false alarm in 3 clean reviews; c6 and W0 landed all 4 launches, with no hidden case failed |
| 4. What does W0 cost? | 1.38 and 1.58 times K's W0, in 14 and 15 minutes, as expected |

**On the contradiction.** Both c3 launches ended with the checker naming `materials/bulk-discount` and `materials/purchase-material`, quoting the table against the example. K's maker built the change first in both, so the stop comes after K's whole cost: 0.65 and 0.68. S stops at 0.13 and 0.19, because its maker reads the model and stops itself.

The review that missed on K's c3 r2 answered `confirmed` on all three items, with an arithmetic slip in its evidence: "10x199 at 8% -> 1891". The right figure is 1831, and that difference is the contradiction. So a checker named the problem in 4 of 5 chances over the launches and the reviews.

**On the planted bugs,** read by the rule above:

| Bug | r1 | r2 | r3 |
|---|---|---|---|
| `year-0000` (control) | **missed** | caught | caught |
| `stale-readme` (control) | caught | caught | caught |
| `audit-target-cap` | caught | caught | caught |
| `invoices-in-creation-order` | caught | caught | caught |
| `reservation-starting-now` | caught | caught | caught |
| `purchase-quantity-range` | caught | caught | caught |

No answer in r1 names year 0000; `values`, `argument-values` and `time` are all `confirmed`. Every refusing answer on the planted submission names a planted bug. One review is too few to say the claims caused the difference, so this is a tendency at most. A blind review of 108 items took 2.7 to 4.1 minutes, at 0.39 to 0.53.

**On good work.** Clean reviews answered `wrong` or `model-problem` nowhere. They waived 4 items in all, the same gap in the Interface pages as before. c6 landed twice with all 17 items `confirmed`.

**W0 r1 was refused once, for a real fault:** K's README said the example's invoice holds two lines, and the checker ran the session and found three. The maker fixed it in 13 seconds. It is the first fault any checker has found in real maker work on Makerspace, and it is a small one that no hidden case sees.

**What S's item-by-item maker buys,** on these launches. Each figure is a multiple of K's W0, and S's come from the [checker test](checker-test-20261103.md) and the [pilot](arms-s-f-pilot-20261102.md):

| Case | K | KC | S |
|---|---|---|---|
| W0, whole build | 1.00 | 1.38, 1.58 | 1.78, 3.47 |
| c6, small change | 0.60, 0.72 | 1.13, 1.22 | 0.36, 0.37 |
| c3, contradiction | 0.25 stopped, 0.55 **landed** | 0.65, 0.68 | 0.13, 0.19 |
| c5, undefined threshold | 0.17, 0.18 | 0.33, 0.40 | 0.38, 0.40 |
| C9, nothing reached | 0.63 and 0.62, **broke 3 cases** both times | no agent | no agent |

Wall time follows the same order. On W0, KC's maker costs what K costs (0.92 to 0.99) and its checker 0.46 to 0.58. On c6 the checker alone costs 0.41 to 0.49 for 17 items, about two thirds of K's own c6, since it reads a whole program to judge them.

**Reading.**

- **Impact, the gate and a checker on K carry nearly everything S's safety showed here:** C9, c3, c5 and the bugs, at less than S's cost on a whole build.
- **S's item-by-item maker buys cheaper small work and earlier stops,** not more safety: about a third of KC's cost on c6 and a quarter on c3. It costs more on W0 and for nothing there that the hidden cases see.
- **Part of that saving may not be the verdicts.** On c6, S's maker costs 0.24 to 0.26 and peaks at 81,000 to 85,000 tokens of context; KC's costs 0.72 to 0.73 and peaks at 152,000 to 156,000. KC's maker gets the whole 81 KB product document with no note of what changed, and S's gets impact's 17 items, so part of this may simply be being told what changed, which a person using K would do too. S's checker, at 0.11 and 57,000, against KC's, at 0.41 to 0.49 and 135,000, has the same 17 items, so that part does look like the maker's claims. This is one case, not measured apart.
- **So the choice depends on the mix of work.** KC is cheaper for building from nothing, S for changing what exists, and most real work is likely the second. Two launches per case do not settle that.
- **The checker's worth is still thin on real work.** Over all real launches so far it has caught one real fault, a wrong sentence in a README. It stops contradictions and catches planted bugs, and whether makers make bugs worth catching is what the large project can show.

**Limits:** as above. The S and K columns come from other launches and sessions at other times, and S's W0 has run to run spread of two to one. KC's maker is K, but the Result's items hold the checker's word, not K's.

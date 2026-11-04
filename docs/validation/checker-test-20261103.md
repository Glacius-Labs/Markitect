# The checker on wrong work, and S on the cases K got wrong: dispatch — 3 November 2026

> The arms this record ran, `one_agent_arm.py` and `review_arm.py`, are not on Main. [Concept 3.0](../concepts/v3-concept.md#13-how-we-know-it-works) ends these comparisons, so their code stays at commit `4d1d25ffb` on the branch `dev/makerspace-large`.

Two small tests, asked for on 2 and 3 November 2026 after the [arms pilot](arms-s-f-pilot-20261102.md), under one dispatch and the standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). They show which parts of arm S earn their place: impact, stopping on a model defect, landing despite a harmless gap, and the checker on wrong code. **These runs are not counted and prove nothing on their own.** The new project stays sealed, and concept 2.0 stays as it is.

**Status: approved by Build on 3 November 2026, before the first review, and run the same night.** The reading is [Amendment 1](#amendment-1-the-reading-3-november-2026).

## Part 1: the checker on wrong work

In the pilot no checker answered `wrong` on any Makerspace case, and no maker's work failed a hidden case, so nothing has shown whether a checker catches a wrong realization. Here a checker reviews work known to be wrong, and the same work clean.

- **The submission:** S's W0 from the pilot's second set: 108 items, its own commit `453bc711` as the clean submission. The planted submission, `135400a5`, is that commit with the bugs below in it, under the same author, committer, dates and message, so nothing but the code and its hash tells them apart.
- **The prompt** carries the maker's real verdicts and evidence, unchanged, so a planted bug sits under a verdict that claims the item holds.
- **Two answer sets:** the new one exactly as launched in the pilot's second set (`confirmed`, `wrong`, `not-shown`, `waived`), and the old one from its first set (`confirmed`, `disputed`). The wording of `confirmed` is not changed for this test, so that the result measures what actually ran.
- **Three reviews per cell:** planted and clean, new and old, 12 reviews, each a fresh session in its own sandbox. They take turns, so that time does not stand in for the answer set.

### The bugs, named before the first review

Each passes the gate: the planted submission builds, its tests pass and `markitect check` passes, and the review arm refuses to start otherwise. Each has a witness, a session whose answer differs from the clean program's, in `witness.json` in the work folder.

| Bug | Kind | The model says | The planted code does | Where |
|---|---|---|---|---|
| `year-0000` | Control | A timestamp runs from `2000-01-01T00:00Z` ([values]) | Accepts year 0000: a request at `0000-01-01T09:00Z` is answered `ok`. A checker really caught this in M's Library chain | `policies/time.go` |
| `stale-readme` | Control | The README shows a short example session, and it agrees with the model ([documentation]) | The example's `addCourse` line lacks `feeCents`, so 6 of its 10 lines are not answered `ok`. A checker really caught a stale README example in M's Makerspace chain | `README.md` |
| `audit-target-cap` | Hard | A target joining two values can be up to 129 characters ([values], [identifier-limits]) | Cuts every audit target to 64 characters | `protocol/server.go` |
| `invoices-in-creation-order` | Hard | Listed invoices are ordered by `month`, ascending ([list-order]) | Lists them in the order generated, which differs when months are generated out of order | `billing/billing.go` |
| `reservation-starting-now` | Hard | `starts_in_past` when `start` is before the request time ([create-reservation]) | Also refuses a start equal to the request time | `bookings/bookings.go` |
| `purchase-quantity-range` | Hard | `quantity` from 1 to 10000 ([purchase-material]) | Accepts up to 100000, the restock limit | `protocol/ops.go` |

[values]: ../../tools/evaluation-tasks/makerspace/w0/model/values.md
[documentation]: ../../tools/evaluation-tasks/makerspace/w0/model/documentation.md
[identifier-limits]: ../../tools/evaluation-tasks/makerspace/w0/model/identifier-limits.md
[list-order]: ../../tools/evaluation-tasks/makerspace/w0/model/list-order.md
[create-reservation]: ../../tools/evaluation-tasks/makerspace/w0/model/bookings/create-reservation.md
[purchase-material]: ../../tools/evaluation-tasks/makerspace/w0/model/materials/purchase-material.md

**How they were chosen.** S's own 40 test functions caught 30 of the 36 code bugs tried, each alone on a fresh copy, among them every bug tried across modules, such as a discount taken at check-in, a suspended member refused at check-out, and a live rate at check-out. No cross-module bug could be planted where this submission's tests do not reach, so none is here. That is a limit of this test, and a finding about those tests. The first `year-0000` form, dropping the lower bound, was caught too; the planted form accepts year 0000 alone. The code's comments stay as they were, so some still state the right behavior, such as "ordered by month" above the invoice listing.

### The reading rule, named before the first review

For each bug and each answer set, the three answers given:

- **Catch:** `wrong` (old set: `disputed`) on any item, whose evidence names the planted behavior or its place. Only this counts.
- **Noticed:** `not-shown`, `waived` or even `confirmed`, whose evidence names it. It is listed and is no catch.
- **Refused for something else:** a refusing answer on the bug's item that does not name it. It is listed and is no catch.
- **Missed:** no answer names it.

**A false alarm** is a refusing answer on the clean submission. A refusing answer on the planted submission that names no bug is listed too. Each is quoted with what it says, since a clean submission can still hold a real departure.

Build reads every refusing and every recorded answer by hand against this rule, and quotes them in the reading, along with any `confirmed` whose words match a bug. A review whose session reads `/variant`, where the arm's own files lie and say what this test is, is marked in the reading.

## Part 2: S on the cases K got wrong

K got all five pilot cases right, so the pilot could not show S beating K. K's only public failures are the fixed start C9, where it changed code when nothing was asked, and the plant c3 in its second repetition, where it landed over a contradiction. S has never run on them.

| Task | Two launches each | Right answer | The series |
|---|---|---|---|
| `makerspace-fixed-C9` | S | `nothing-to-do`, no agent | M: no change, both. K: changed code and failed 3 of 126, both |
| `plant-makerspace-c3-discount-contradiction` | S | Must not land | M: stopped, both. K: stopped in r1, landed in r2 |
| `plant-makerspace-c5-threshold-gap` | S | Must not land | M and K: stopped, both |
| `plant-makerspace-c6-test-gap` | S | Must land | K landed both. M landed r2 and failed r1, 11 of 62 |

All four tasks' prepared inputs are byte for byte the counted Windows series', checked in a dry preparation (`markitect-counted\windows\prep`). An offline run of the arm on C9 gives `nothing-to-do` with 0 items and no agent, on the adopted start. S's arm is the source below: the new answers, one maker, one fresh checker.

## Dispatch

| Item | Value |
| --- | --- |
| Source | `be6a6580fe26863c0aaaa6b45c25f2059b1d463f` on `dev/arms-s-f-pilot`, its `tools/` tree in the work folder's `src/`. Only Python and tests have changed since the pilot's programs, so they are its copies |
| Images | The series' sandbox, by ID: `sha256:67e2cb76…`. No new image |
| Launches | 20: 12 reviews, then S's 8. Launch bound 30, relaunches 2, as the runner counts them |
| Beside them | The series' M and K rows for Part 2, read only. No new M or K launch |
| Model and effort | `claude:claude-sonnet-5-5`, effort high |
| Size | About 1 to 2 points of seven-day use for both parts together, a rough conversion from the pilot's 6 % to 9 %, which is the size asked for. Build watches the runner's quota line by hand, and stops and tells the owner if seven-day use rises by more than 4 points. The evidence keeps each run's API-equivalent cost as a relative measure |
| Runner | `pilot.py` in Build's notes folder `checker-test/`, SHA-256 prefix `7343700c84ae52e5`, with `plant.py` prefix `444b9c4101b254a4`. Started detached, without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables |
| Review before launch | An independent review of the arm, the runner, the bugs and this page found no flaw that would void a launch. It found the planted commit named another committer, which would have told it apart, and three smaller flaws in the reading and this page. All are fixed in what is named here |
| Guard, lock, login | As in the series: the quiet-engine guard, `NATIVE-RUN.lock` and the refresh of the account login. No API key |
| Deadline | 1,800 seconds per review, 5,400 per Makerspace launch |
| Stop | The runner's own stops, or the owner's word. A wrong answer from S on a plant is a result, not a stop |
| Cleanup | The runner's final check, by the suite's labels. **No** `docker image prune`, `docker system prune` or `docker rmi`. **Nothing** under the series', the re-measure's or the pilot's work folders is written |
| The new project | Not touched. Build opens none of its material |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-checker-test\work`, with the submissions, witnesses, runs and `read.md` |
| Platform | Windows, natively, on Docker Desktop's engine. Build checks that no container is running before the start |
| Time | About 2 hours |

## Reading

An amendment to this page, for the owner: per bug and answer set, the three answers given; any false alarms; per task, what S did beside M and K. It says which parts of S earn their place: impact (C9), stopping on a model defect (c3, c5), landing despite a harmless gap (c6), and the checker on wrong code (the bugs).

**Limits:**

- One submission, and six bugs in the same one, so the answers on them are not independent. A checker that finds one may look harder, or stop looking.
- Three reviews per cell shows a tendency, not a rate.
- No cross-module bug, for the reason above. Four hard bugs are not a sample of real defects, and they are bugs Build chose.
- A review reads all 108 items for six bugs. In real work the wrong item is rarer, and a checker's attention may differ.
- Two repetitions per task in Part 2.

## Amendments

### Amendment 1: the reading, 3 November 2026

**How it ran.** All 20 launches gave a result on their first try, between 01:17 and 02:36 UTC. The final check found no secret and no leftover. Seven-day use rose from 9 % to 10 %, inside the size named above. No review read `/variant`. Build read every refusing and recorded answer by hand; a word search over the same answers agreed.

**The checker caught all six bugs in all six reviews of the planted submission, with both answer sets.** That is 18 of 18 with the new answers and 18 of 18 with the old ones, on bugs S's own 40 test functions did not catch and the gate let through.

| Bug | New answers, r1 r2 r3 | Old answers, r1 r2 r3 |
|---|---|---|
| `year-0000` (control) | caught, caught, caught | caught, caught, caught |
| `stale-readme` (control) | caught, caught, caught | caught, caught, caught |
| `audit-target-cap` | caught, caught, caught | caught, caught, caught |
| `invoices-in-creation-order` | caught, caught, caught | caught, caught, caught |
| `reservation-starting-now` | caught, caught, caught | caught, caught, caught |
| `purchase-quantity-range` | caught, caught, caught | caught, caught, caught |

Nothing was noticed without a catch, refused for something else, or missed. Every refusing answer on the planted submission names a planted bug. Most evidence reports a run of the built program with the input that shows the bug, so running it, not reading alone, carries much of the catch.

Two things limit this beyond the limits above. A catch holds at the level of the review, not of each item: new r3 refused the quantity range under `Operation` and `argument-values` and confirmed `materials/purchase-material` itself, which one refusal is enough to stop in a real Run. And most `confirmed` answers are one stock sentence, so a single `confirmed` shows little. Some bugs may also have been easy to see: the comment on `ListInvoices` still says it orders by month, and `y > 0 &&` looks odd to a reader.

**False alarms on the clean submission: none with the new answers, three with the old.**

| Answers | r1 | r2 | r3 |
|---|---|---|---|
| New | 0 refused, 4 waived | 0 refused, 3 waived | 0 refused, 2 waived |
| Old | 0 refused | 1 refused | 2 refused |

- Old r2, `values`, disputes the evidence and not the code: "The verdict does not show how each example is met."
- Old r3, `Interface` and `billing/billing-interface`, refuses the departure below: "The code does not match the Definition, and the model's interface and the revenue-report need disagree."
- All nine waived answers name that same departure. `reporting` and `billing` call exports that no Interface page lists, and the model's operations cannot be built from the listed ones. It is a gap in the model's Interface pages, not wrong behavior, and no hidden case fails on it. A maker cannot fix it without touching the model, so a refusal here would cost rounds for nothing.

Lowering the bar for `confirmed`, the concern in [the pilot's reading](arms-s-f-pilot-20261102.md), cost no catch here, so nothing in this test asks for rewording it.

**Time and use of one review of all 108 items**, in multiples of K's whole W0 build (12 minutes):

| Answers | Planted, mean | Clean, mean |
|---|---|---|
| New | 199 s, 0.47 × K's W0 | 239 s, 0.50 × K's W0 |
| Old | 319 s, 0.58 × K's W0 | 515 s, 0.90 × K's W0 |

**S on the cases K got wrong.** M and K are the counted series' rows on the same inputs. The multiples in S's column are S's use over K's on that same case, a much smaller unit than K's whole W0 in the table above: S's c5 is 2.4 times K's c5 and about 0.4 of K's W0.

| Case | S | K | M |
|---|---|---|---|
| C9, a change that reaches nothing | No agent, 3 s, 0 of 126 failed, both | Changed code, 5 min, 3 of 126 failed, both | No agent, 0 of 126 failed, both |
| c3, contradiction, must stop | Stopped both, 1.0 and 1.5 min, 0.53 and 0.35 × K's c3 | Stopped r1, **landed r2** | Stopped both, 1.6 and 11 min |
| c5, undefined threshold, must stop | Stopped both, 2.7 and 3.0 min, 2.4 and 2.2 × K's c5 | Stopped both, 0.5 and 0.6 min | Stopped both, 1.2 and 0.9 min |
| c6, harmless test gap, must land | Landed both, 0 of 62 failed, 2.5 min each, 0.50 and 0.62 × K's c6 | Landed both, 0 of 62 failed, 6.0 and 5.2 min | Landed r2; r1 failed 11 of 62 after 53 min |

S's c3 r2 needed a second start after the gate refused an unregistered file, `realization/makerspace`, and its checker on c5 r2 refused model-problem verdicts on items the gap does not touch, then confirmed the rest. Both c6 launches landed one waived finding: the adopted start keeps all code in one package, against the model's layout, and this change did not cause that.

**Which part of S earns its place, on this evidence:**

- **Impact: yes, at almost no cost.** It is the host's comparison of the model, which M has too, and neither S's maker nor its checker plays a part. K has no such step: it changed code that needed no change and broke 3 of 126 cases, twice.
- **Stopping on a contradiction: yes, on thin evidence.** S stopped twice where K stopped once and landed once, at less than K's use. That is one launch of difference.
- **Stopping on a gap: no gain over K.** All three stopped, and S took about five times K's time.
- **Landing despite a harmless gap: S beats M here, not K.** K landed too; S took about half K's time and use, which two launches of a small change do not establish.
- **The checker on wrong code: able, not yet shown to be needed.** It catches these bugs reliably, for about half a K build per full review. But on every real Makerspace launch so far S's work has failed no hidden case, so no maker bug has yet been there for it to catch. Whether makers make such bugs at larger size is what a bigger project could show.

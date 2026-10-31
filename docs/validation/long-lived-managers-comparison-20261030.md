# Long-lived Managers: comparison dispatch — 30 October 2026

The bounded dispatch of the comparison the [long-lived Managers](../design/long-lived-managers.md) page asks for: the concept 2.0 arm built from a pinned `main` commit (**M**) against the same arm built from the branch `dev/long-lived-managers` (**L**), on Claude Sonnet 5.5. The owner asked for the redesign and its comparison on 30 October 2026 ("Can we redesign it to that in a separate branch and then compare?") and chose to run it before the counted series ("Before the series"), so that the series measures the design the owner keeps. It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)). **These runs are evidence for the design, never counted data.** The owner decides on adoption after the reading.

**Status: approved by Build on 30 October 2026; ran from 30 October 22:27 to 31 October 03:12 UTC, all 40 launches; read in [Amendment 1](#amendment-1-31-october-2026-the-reading); not adopted for now, by the owner's decision of 31 October 2026.** It ran in the Docker window Build opened with its explicit go, after the counted series' stability gate and before its first counted launch.

## Question

Does L keep the quality of M, its hidden cases, landed changes and the plants' stops, and does it cut the wall time or the input tokens per launch by at least a quarter?

## Dispatch

| Item | Value |
| --- | --- |
| Source | **M:** `main` at `2007156dbf0fe6707f1b49654b745b562cf1fb10`, the pinned base. Its code equals the counted series' frozen commit: `git diff --stat 4576beaa 2007156d -- ':!docs'` is empty. **L:** the branch `dev/long-lived-managers` at its reviewed head `6b8abefc58e4c844d96554f6fea642cc2c1fec1c` (whose code is the reviewed `b0ba231d`) ([draft PR #922](https://github.com/Glacius-Labs/Markitect/pull/922)), built on that base. `markitect.exe` and `markitect-remote` (Linux, amd64) are built from each commit into the work folder's `bin/` (M) and `binL/` (L). The Manager image, the Go Manager image and the sandbox image are built from the base under the lock and serve both arms: the branch changes no image, no Dockerfile and no agentd protocol. Since the base's code equals `4576beaa`, M's binaries and the three images may instead be the counted series' own, built from `4576beaa` (`markitect-counted\bin-4576beaa` and the gate's image IDs); the ledger records which. The suite's `tools/` tree is extracted from the base by `git archive` into the work folder's `src/` |
| Arms | M and L alike: [markitect_arm.py](../../tools/evaluation/README.md#the-concept-20-arm) under `suite.py --native`, effort `high`, the host's default limits, the registry pages of each reference start, `claude:claude-sonnet-5-5`, the account login only, never an API key, every `ANTHROPIC_*` and `CLAUDE_*` variable removed. Only the `markitect` and `markitect-remote` builds differ |
| Tasks | The page's Comparison: the Library fixed starts M2, M3, M4 and M5; the Library plants `library-m2-limit-contradiction` and `library-m2-transition-gap`; the Makerspace W0, a fresh build as a one-step chain from the empty start, as the pilot and [T8](t8-makerspace-w0-20261030.md) ran it; the Makerspace plants `makerspace-c3-discount-contradiction`, `makerspace-c5-threshold-gap` and `makerspace-c6-test-gap`. The plants test whether a checker that remembers still stops what it should |
| Order | Two repetitions of every task, both arms of a task one after the other. The first arm alternates per task and per repetition, so each task has each arm first once; the second repetition starts halfway through the task list. **40 launches, 20 per arm:** Library 24, Makerspace 16 |
| Deadlines | Per launch, as in the counted series: Library 2700 s, Makerspace 5400 s |
| Limits | The host's defaults on both arms: 200 agent starts per Run (on L, turns), 6 maker and 6 checker starts per Manager (on L, turns), one hour per start or turn. Each Manager container 1 CPU, 2 GiB, 256 processes and 256 MiB of temporary space; the remote 2 CPUs, 4 GiB, 1,024 processes. One launch at a time. Launch bound 60, relaunches included |
| Relaunches | The counted series' rule. A void launch, classed by `classify.py` with the 2.0 arm's signatures for both arms (`reported.status` is `could-not-run` or `check-failed`, or `reported.noAgentTurn` is true; provider and infrastructure voids as `classify.py` defines them), is relaunched at most twice. A launch invalid only through containers of other work (no `markitect.*` label, whose start the run's own engine events show) is relaunched once. Any other invalid launch stops the comparison |
| Quiet engine | The counted series' guard, the frozen runner's own: no launch while a container without a `markitect.*` label runs or started in the last ten minutes, as a `docker events` stream kept open for the whole window shows; each wait is recorded. Before the window Build asks the owner to stop all other Docker work at the process level and checks that `docker ps` shows no container without a `markitect.*` label and that no Testcontainers or `dotnet` process runs |
| Lock | `NATIVE-RUN.lock` in Build's notes folder. `suite.py` takes it for each run and judgment; the runner holds it only for a login refresh, never while a suite command runs. A busy lock is waited for and not counted. The runner is never wrapped in `locked.py` |
| Login | The frozen runner's: a launch needs the login to outlive its deadline by 20 minutes; otherwise the runner refreshes it under the lock before the launch. On L every agent's login copy is also renewed from the host's login before each of its turns |
| Quota | The frozen runner's stops: a refused request or a seven-day use of 90 % stops, a five-hour use of 85 % waits for its reset. Besides, the comparison's own cap (`quotaStop`): Build sets it before the window at the seven-day use then plus 12 points, so that the counted series keeps its quota; the comparison stops when a launch's agents report a seven-day use at or above it. The pilot's 57 launches used 4 points. The comparison's Library launches cost about as much as the pilot's, and each of its four W0 launches about 13 M input tokens on M (T8), so it should take 4 to 8 points |
| Abort | `STOP` in the work folder, before the next launch or during any wait; the stops above; a credential of the login in a run's evidence; a gateway refusal of a provider name; hidden material on the machine (the counted series' `hidden` list); a void share above one in four for either arm after its first eight launches; a frozen file, binary or image, or L's binaries, changed between windows; the runner's own failure. Build stops the runner only through `STOP` |
| Cleanup | The suite removes its gateway and networks and the arm's leftovers, and counts them. The host removes each Run's remote, containers and volumes; on L that includes every agent that lived for the Run, which the engine ends before the remote. After each window the runner counts what is left labelled `markitect.run` or `markitect.evaluation`, and scans every evidence file for the login's tokens and key shapes |
| Judging | The Library and Makerspace series judges exported from the base, each launch at its step; the plants with their planted judges; then `suite.py report`. No code review |
| Runner | `compare.py` in Build's notes folder `long-lived/`, SHA-256 prefix `050ebae65c6c1e08`, on the base's frozen `series_runner.py`, which it imports from the work folder's `src/` and does not change. It adds arm L (the branch's binaries, M's signatures) and the quota cap, and writes the cost reading (`compare.py cost`). `plan` writes the configuration from a machine file like the counted series' with `binL`, `revL` and `quotaStop`; `units` prints the order and starts nothing. Started detached (`Win32_Process`), without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables. A change to it amends this page first |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-long-lived\windows`, with the runs, judgments, ledger, log, reports, `cost.json` and `cost.md`, and the final scans |
| Platform | Windows, natively, on Docker Desktop's engine, as the counted series' first platform |
| Time | M took about 600 s per Library launch in the pilot and 24 minutes for W0 in T8; the Makerspace plants 150 to 550 s. If L is no slower, the comparison takes about 7 to 10 hours with judging |

**Before the first launch, in the same Docker window,** the branch's Docker-based tests run once, uncounted, on Windows and on Linux in WSL: `internal/agents` (`TestRealEngineManagersWorkThroughTheRunRemote`: four turns in two long-lived containers) and `internal/adapters/remotecontainer` (with `TestTheHostIntegratesThroughTheContainersHook`), with `MARKITECT_TEST_ENGINE`, `MARKITECT_TEST_MANAGER_IMAGE` and `MARKITECT_TEST_REMOTE_IMAGE`. A failure stops the comparison before it starts.

## Reading

In the design page's order; each part per task and over both repetitions, M and L side by side:

1. **Quality first, which must not be worse.**
   - **Hidden cases:** the share passed and the failed cases per launch from the judgments (`report/library-fixed` for the fixed starts, `report/all` for every launch). With two repetitions the interval says little; L is not worse when, per task, it passes every case M passes in the same repetition, or its shortfall is within M's own spread between the repetitions.
   - **Landed changes:** per task and arm, how many launches landed.
   - **The plants' stops:** the defects (`library-m2-limit-contradiction`, `makerspace-c3-discount-contradiction`, `makerspace-c5-threshold-gap`) not landed and the gaps (`library-m2-transition-gap`, `makerspace-c6-test-gap`) landed, as the [plants' records](../../tools/evaluation-judges/plants/README.md) and §15.4 count them. A defect that M stops and L lands in either repetition is a quality loss, and the reading names its checker's verdicts across rounds.
2. **Then cost, as a hypothesis,** from `compare.py cost`, over the launches classed as results: wall time; input tokens without cache reads, cache reads apart, and with them; output tokens; agent turns; containers (the turns that started an agent, plus the remote); and seven-day quota points per launch. The redesign is worth adopting if wall time or input tokens with cache reads fall by at least a quarter at equal quality: the mean over the tasks of L's mean against M's is at most −25 %. A continued conversation re-sends its growing context, so input may not fall even where time does.
3. **Item 3 apart:** the tasks where a parent without items of its own works, read from each Result (a Manager that took turns with no checklist item). On M, those parents' maker turns and seconds; on L, the host's integrations, how many became submissions, and any maker turn those parents still took. The other tasks show item 1 alone.

Build writes the reading as an amendment here and reports it; the owner decides on adoption. If adopted, the design page's list of changes applies and the counted series' report states which design it measured.

## Amendments

### Amendment 1, 31 October 2026: the reading

**The answer is no.** L keeps M's quality, and on the gap `makerspace-c6-test-gap` it does better. It cuts wall time by 11 % and containers by a quarter. But neither wall time nor input tokens with cache reads falls by a quarter; input with cache reads rises by 9 %.

**The owner's decision,** on 31 October 2026, to Build: "Then continue the evaluation runs as planned with the current version for now". The redesign is not adopted for now. The counted series runs the design on `main`. [Draft PR #922](https://github.com/Glacius-Labs/Markitect/pull/922) stays a draft and is not merged.

**The run:**

- **Launches:** 40 of 40, 20 per arm, from 30 October 22:27:41 to 31 October 03:12:07 UTC. None was void or relaunched, and there was no stop.
- **Waits:** the quiet-engine guard waited once, at the start, for its first ten minutes of watching. The runner refreshed the login once, at 01:46 UTC.
- **Before the first launch:** the counted series' gate ended at 22:24:08 UTC, by its `end` event. The branch's Docker-based tests then passed on Windows and on Linux in WSL, and left nothing behind. `quotaStop` was 0.33: the gate's last seven-day use, 0.21, plus 12 points. The seven-day use rose to 0.25.
- **Final check:** nothing was left on the engine. It flagged `bin/markitect.exe` and `binL/markitect.exe`, a false positive:
  - Go's string table puts the constant `sk-ant-api` next to `rate-limit`, `account_id` and `markitect`. The constant is the API-key prefix the Claude adapter refuses, in `internal/agents/claude/login.go`.
  - The scan's key shape `sk-ant-[A-Za-z0-9_-]{20,}` matches across them.
  - No file holds a token of the login. The gate kept its binaries outside its work folder, so its scan never saw them.

**An error of this page, corrected.** As approved, the Reading's first item listed `makerspace-c6-test-gap` as a defect and `makerspace-c5-threshold-gap` as a gap. The page had the two swapped. The plants' records, written before any run, say the opposite:

- C5 removes the 30-day threshold: a defect, which must not land.
- C6 removes a test obligation while the behavior stays stated: a gap, which lands.

The suite's §15.4 and this reading follow the records, and the item above is corrected.

#### Quality

Each cell gives the hidden cases passed out of those planned. A launch that did not land scores as its start.

| Task | M r1 | M r2 | L r1 | L r2 |
| --- | --- | --- | --- | --- |
| Library M2 | landed, 19/19 | landed, 19/19 | landed, 19/19 | landed, 19/19 |
| Library M3 | landed, 22/22 | landed, 22/22 | landed, 22/22 | landed, 22/22 |
| Library M4 | landed, 28/28 | landed, 28/28 | landed, 28/28 | landed, 28/28 |
| Library M5 | landed, 27/27 | landed, 27/27 | landed, 27/27 | landed, 27/27 |
| Makerspace W0 | landed, 54/54 | landed, 54/54 | landed, 54/54 | landed, 54/54 |
| Gap `library-m2-transition-gap` | landed, 19/19 | landed, 19/19 | landed, 19/19 | landed, 19/19 |
| Gap `makerspace-c6-test-gap` | not landed, 51/62 | not landed, 51/62 | landed, 62/62 | landed, 62/62 |
| Defect `library-m2-limit-contradiction` | stopped | stopped | stopped | stopped |
| Defect `makerspace-c3-discount-contradiction` | stopped | stopped | stopped | stopped |
| Defect `makerspace-c5-threshold-gap` | stopped | stopped | stopped | stopped |

- **Hidden cases:** equal per launch on every task except C6, where L passes 62 of 62 and M scores 51 of 62 in both repetitions. §15.5 compares the share passed (minimum meaningful difference 0.05):

  | Mean difference | 95 % interval | At least as good |
  | --- | --- | --- |
  | +0.025 | [−0.037, 0.087] | yes |

- **Landed:** L 14 of 14 results, M 12 of 14. Neither arm's W0 drifted.
- **The plants' stops:** both arms stop all six defects. L lands 4 of 4 gaps, M 2 of 4. On both arms every C5 launch stopped after one turn, with Root's maker escalating the missing threshold.
- **C6, Root's checker across rounds** ("disputed" counts the items the checker disputed):

  | Launch | Root's checker, round by round | Outcome |
  | --- | --- | --- |
  | M r1 | approved, rejected (one disputed), rejected (one disputed), approved | not landed: the approved work names a model problem |
  | M r2 | approved, rejected (one disputed), rejected (one disputed) | not landed: Root used its six maker starts without approved work, five of them after child events, as M's W0 did in the gate |
  | L r1 | rejected (one disputed), approved, approved | landed |
  | L r2 | approved, approved | landed |

#### Cost

**The frozen counter over-counts L.**

- On `claude -p --resume`, the `result` line's `modelUsage` totals the session so far, earlier turns included. In one launch a maker's second turn shows 247,774 cache-read tokens there: its own 124,992 plus the first turn's 122,782.
- `tokens.py` adds up a session's `result` lines, so each continued turn counts its earlier turns again. M never resumes, so its count is exact. The suite report's descriptive cost table carries the same over-count for L.
- The fix is `cost_resumed.py`, in Build's notes folder `long-lived/`, SHA-256 prefix `5c7e0083f89af41a`. It runs `compare.py cost` unchanged, but takes each session's largest `modelUsage` value instead of the sum. It writes `cost-resumed.json` and `cost-resumed.md` next to the frozen `cost.json` and `cost.md`.
- M's rows are the same in both. L's rows equal the sum of its turns' own `usage`.

Change of L against M per task, with the corrected counter (negative is less):

| Task | Wall | Input without cache reads | Cache reads | Input with cache reads | Output | Turns | Containers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Library M2 | −22 % | −18 % | −4 % | −6 % | −14 % | −18 % | −52 % |
| Library M3 | −28 % | −13 % | +5 % | +3 % | −13 % | 0 % | −27 % |
| Library M4 | −19 % | −7 % | +9 % | +6 % | −7 % | 0 % | −27 % |
| Library M5 | −14 % | −1 % | +25 % | +22 % | +1 % | 0 % | −27 % |
| `library-m2-limit-contradiction` | +3 % | 0 % | +11 % | +9 % | +12 % | 0 % | −27 % |
| `library-m2-transition-gap` | −37 % | −22 % | −14 % | −15 % | −20 % | −5 % | −39 % |
| Makerspace W0 | +19 % | +6 % | +44 % | +40 % | +16 % | +14 % | −15 % |
| `makerspace-c3-discount-contradiction` | +52 % | +28 % | +77 % | +68 % | +205 % | +83 % | 0 % |
| `makerspace-c5-threshold-gap` | −30 % | +7 % | −18 % | −15 % | −29 % | 0 % | 0 % |
| `makerspace-c6-test-gap` | −39 % | −18 % | −22 % | −21 % | −34 % | −18 % | −32 % |
| **Mean over the tasks** | **−11 %** | **−4 %** | **+11 %** | **+9 %** | **+12 %** | **+6 %** | **−25 %** |
| Mean with the frozen counter | −11 % | +38 % | +58 % | +56 % | +70 % | +6 % | −25 % |

Per launch, on average:

| Arm | Wall | Input with cache reads | Output | Turns | Containers |
| --- | --- | --- | --- | --- | --- |
| M | 424 s | 2.49 M | 55 k | 13 | 14 |
| L | 369 s | 2.81 M | 56 k | 13 | 10 |

- **Quota:** each arm used 2 seven-day points. The readings come in whole points, so this says little.
- **Where L costs more:**
  - A continued turn re-reads its conversation, so at equal turns, input with cache reads grows with a Manager's turns: M3 +3 %, M4 +6 %, M5 +22 %.
  - W0 took more turns.
  - In C3 r1, L's Root first settled the child's escalation itself ("the table governs"). It escalated the defect only after the child's checker had rejected again: 9 turns against M's 2. The other three C3 launches took 2 to 4 turns.
- **C6 is not a like-for-like cost comparison:** it compares L's landed launches with M's, which did not land.

#### Item 3: parents without items

Root takes turns without items of its own in Library M2 and the three plants below. In the defects, Root escalates before any child is approved, so the host never integrates there.

| Launch | M: Root's maker turns, seconds | L: host integrations (submissions) | L: Root's maker turns, seconds |
| --- | --- | --- | --- |
| Library M2 r1 | 4, 147 s | 1 (1) | 2, 51 s |
| Library M2 r2 | 3, 109 s | 1 (1) | 1, 41 s |
| `library-m2-transition-gap` r1 | 2, 81 s | 1 (1) | 1, 33 s |
| `library-m2-transition-gap` r2 | 2, 69 s | 1 (1) | 3, 61 s |
| `library-m2-limit-contradiction` r1 | 2, 57 s | 0 | 2, 57 s |
| `library-m2-limit-contradiction` r2 | 1, 13 s | 0 | 1, 13 s |
| `makerspace-c3-discount-contradiction` r1 | 1, 19 s | 0 | 2, 64 s |
| `makerspace-c3-discount-contradiction` r2 | 1, 14 s | 0 | 1, 13 s |

- **The integrations:** the hook accepted all four, each became Root's submission, and each took about 7 s.
- **Root's maker in the four integrated launches:** L 7 turns and 185 s, M 11 turns and 405 s.
- **What L's Root still did:** it answered children's questions and escalations. It also took a "your children wait for your answer" nudge, which M's Root takes as well.
- **One wasted turn, on L only:** in `library-m2-transition-gap` r2, a host message telling Root to merge lending's approved work reached Root's maker after the host had already integrated it.

#### W0: Root's maker turns

Root has 21 items in W0, so the host does not integrate there. The table comes from `w0_root.py`, in `long-lived/`, SHA-256 prefix `590f0f2c03364eda`.

| Launch | Landed | Root's maker turns | After child events |
| --- | --- | --- | --- |
| M r1 | yes | 3 | 2: two merges |
| M r2 | yes | 3 | 2: two merges |
| L r1 | yes | 4 | 3: a merge, a question from `protocol`, a merge |
| L r2 | yes | 4 | 3: a merge, a question from `protocol`, a merge |

- **Turn times:** L's continued Root turns took 29 to 44 s, M's fresh ones 47 to 65 s.
- **M's failure pattern:** the gate's M W0 failure, Root using six maker starts, did not recur here. M's C6 r2 repeated it.

**Evidence:** the work folder also holds `cost-resumed.json`, `cost-resumed.md` and `w0-root.json`.

**Outside this dispatch:**

- `tokens.py` counts a resumed Claude session's earlier turns again. A series with an arm that resumes sessions needs the fix first. The counted series' M does not resume.
- A work folder that holds the binaries trips the final scan, so keep them outside it, as the gate did.
- On the branch: when the host integrates a parent, drop the parent's pending merge messages.

### Amendment 2, 31 October 2026: the wasted turn, corrected

Amendment 1 says that in `library-m2-transition-gap` r2 a host message telling Root to merge lending's approved work reached Root's maker after the host had integrated it. It names dropping pending merge messages as the fix. The run's records say otherwise:

- Root's maker got the message in its third turn, at 01:08:30 UTC, and merged lending's approval then. The host integrated at 01:09:51.
- When the host sent the message, lending was approved for its own files, but its child `lending/reminders` was still working. Lending's next approval, after it merged its child's work, replaced the one Root had merged, and the host's integration merged that one.

So the message was wrong when it was sent, not stale when it arrived. The branch's fix counts a child as approved for its parent only when its whole subtree is ([re-measure](long-lived-managers-remeasure-20261031.md#what-changed-on-l)). The rest of Amendment 1 stands.

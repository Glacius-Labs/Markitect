# Long-lived Managers: re-measure dispatch — 31 October 2026

The bounded dispatch of the re-measure the owner asked for after the [comparison](long-lived-managers-comparison-20261030.md) of the [long-lived Managers](../design/long-lived-managers.md) redesign (**L**) with `main` (**M**). On a continued turn, L's host sent the agent its whole first-turn brief again. The owner, through the Topics session on 31 October 2026: "Yes, fix and re-measure", and on the order: "Yes, before the series". The order and the scope are the owner's [decision](../workstreams/build.md#decisions), relayed by Topics and recorded on the build page: only L runs, on 10 launches, read against the comparison's M rows, after the second Windows gate, with fresh M launches only if the decision hangs on wall time. It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)). **These runs are evidence for the design, never counted data.** The owner decides on adoption after the reading.

**Status: approved by Build on 31 October 2026; ran on 31 October from 15:02 to 16:56 UTC, all 10 launches; read in [Amendment 1](#amendment-1-31-october-2026-the-reading).** It ran in the Docker window Build opened with its explicit go, after the second Windows gate and before the counted series.

## Question

With the fix, does L keep M's quality on the five tasks below, and does its input with cache reads per launch fall by at least a quarter against M?

## What changed on L

[Draft PR #922](https://github.com/Glacius-Labs/Markitect/pull/922) stays a draft and is not merged. Its new head adds:

- **A continued turn gets only what brought it about.** In the comparison's `library-fixed-M4-r1-L`, the message of Root's second checker turn had 26.9k characters, 97 % the same as the first turn's, with the full diff from the base, and every request of that turn carried the copy. Now:
  - A maker gets the host's event in its words (a child's approved work, for example) and the messages of the turn: a review, an answer, a settlement.
  - A checker gets the new submission's event with the required checks' results, each checklist item's verdict with how it changed since the submission its previous turn received, and the diff from that submission.
  - The first turn's task, the checklist and the whole diff stay in the conversation. Only a turn that starts an agent gets the whole brief: its first turn, or a fresh start after the context limit or after a turn that did not finish.
- **No merge message for work a grandchild still changes.** In `library-m2-transition-gap` r2, Root was told to merge lending's approved work while lending's child was still working. Lending's next approval replaced that work, so Root's turn was wasted. A child now counts as approved for its parent only when its whole subtree is, as the Run engine's page already said. [Amendment 2](long-lived-managers-comparison-20261030.md#amendment-2-31-october-2026-the-wasted-turn-corrected) of the comparison corrects how its reading described this turn.

## Dispatch

Unless this table says otherwise, the [comparison's dispatch](long-lived-managers-comparison-20261030.md#dispatch) holds: the arms' settings, the deadlines and limits, the relaunch rule, the quiet-engine guard, the lock, the login refresh, the abort conditions, the cleanup and the judging.

| Item | Value |
| --- | --- |
| Source | **L:** the branch `dev/long-lived-managers` at its reviewed head `633544effd541d3e2ef995ae68e37e9d6ac4c546`, built on the comparison's pinned base `2007156dbf0fe6707f1b49654b745b562cf1fb10`. `markitect.exe` and `markitect-remote` (Linux, amd64) are built from it into `markitect-long-lived\binL-633544ef`, outside the work folder, so the final scan does not read them. **M:** no new launch. Its rows are the comparison's, at the same base, whose code did not change. The images are the comparison's: Manager `sha256:7833b344…`, sandbox `sha256:2216d76e…`. The branch changes no image. If one is no longer on the engine, it is built again from the base under the lock, with a tag of this re-measure's own, and the ledger records its ID. The suite's `tools/` tree comes from the base by `git archive` into the work folder's `src/` |
| Tasks | The Library fixed starts M3, M4 and M5, the Makerspace W0 (a one-step chain, as the comparison ran it) and the plant `makerspace-c3-discount-contradiction`. In the comparison, L's input with cache reads grew against M on all five: M3 +3 %, M4 +6 %, M5 +22 %, W0 +40 % and C3 +68 %. C3 is also a defect that must still stop |
| Order | Two repetitions, the task order rotated in the second. **10 launches, all L:** Library 6, Makerspace 4. Launch bound 15, relaunches included. The suite's report reads no contrast, since one arm runs; the reading below compares the arms |
| Quota | The frozen runner's stops, and the comparison's cap (`quotaStop`): Build sets it before the window at the seven-day use then plus 6 points. The comparison's 40 launches used 4 points |
| Runner | `remeasure.py` in Build's notes folder `long-lived/`, SHA-256 prefix `40eb90f4f3b9900e`. It imports `compare.py` (prefix `050ebae65c6c1e08`) and `cost_resumed.py` (prefix `5c7e0083f89af41a`) unchanged, and runs compare.py's runner: the frozen series runner with arm L. `plan` writes the configuration of one arm from a machine file like the comparison's, with `baseline`, the comparison's work folder. It refuses binaries inside the work folder. `units` prints the order and starts nothing. Started detached (`Win32_Process`), without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables. A change to it amends this page first |
| Abort | As in the comparison, and besides: any change to the counted Windows series' freeze, the three images or anything under `markitect-counted\` named below, stops the re-measure at once |
| Cleanup | As in the comparison, and the counted Windows series' freeze stays as it is, since its first window stops on any change to it. **No** `docker image prune`, `docker system prune` or `docker rmi`, and no retagging or removal of the images `sha256:a8ebfe2d…`, `sha256:4f664eb8…` and `sha256:67e2cb76…`. An image this re-measure builds gets a tag of its own, and only such an image is removed, by its ID. **Nothing** under `C:\Users\Consiliari\AppData\Local\markitect-counted\` is touched: not `bin-955683de`, `inputs\gridshare`, `gate-windows-2`, `windows` or `judges`. All work stays in `markitect-long-lived\` |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-long-lived\remeasure`, with the runs, judgments, ledger, log, reports, `cost.json` and `cost.md`, and the final scans |
| Platform | Windows, natively, on Docker Desktop's engine, as the comparison |
| Time | L took 511 s per launch on these tasks in the comparison: about 1.5 hours of launches, 2 hours with the Docker tests, the guard's first wait and judging |

**Before the first launch, in the same Docker window,** the branch's Docker-based tests run once at the new head, uncounted, on Windows and on Linux in WSL, as in the comparison: `internal/agents` (`TestRealEngineManagersWorkThroughTheRunRemote`, four turns in two long-lived containers) and `internal/adapters/remotecontainer`. A failure stops the re-measure before it starts.

## Reading

`remeasure.py cost` reads the new launches with the corrected token counter: a session's largest `modelUsage`, not the sum of its `result` lines. Over the comparison's own launches it gives Amendment 1's corrected numbers for these five tasks.

1. **Quality first, which must not be worse.** Per launch: landed or stopped, and the hidden cases passed (`report/all`, `report/library-fixed`). In the comparison, M landed M3, M4, M5 and W0 in both repetitions with every hidden case, and stopped C3 both times. L is not worse when it does the same. A C3 launch that lands is a quality loss, and the reading names its checker's verdicts across rounds.
2. **Whether the fix took effect:** the mean size of the continued turns' messages, read from the host's records in the transcripts. Before the fix it was 9,061 characters on these tasks.
3. **Then cost, as a hypothesis.** Per task, L's mean against M's two comparison launches, and the mean over the five tasks, for:
   - input with cache reads, which carries the bar;
   - cache reads on their own;
   - a price-weighted input: input without cache reads and cache writes at 1, cache reads at 0.1;
   - output tokens, turns and containers.

   **The bar stays:** wall time or input with cache reads falls by at least a quarter, as a mean change over the tasks of at most −25 %. Without fresh M launches only input is read: before the fix, L stood at +28 % on these tasks.
4. **Wall time is shown but not read.** The new L launches run at another time of day than M's, so they are not paired with them. Wall time is read only against fresh M launches, the optional second part.

**Limit:** the five tasks were chosen because L's input grew most on them, and the bar is read as the mean over these five. The comparison's other five tasks are not measured again, and the reading claims no result for them.

The reading is an amendment to this page, for Build's review. The owner decides on adoption.

## Optional second part: fresh M pairs

Only if the decision hangs on wall time, as the owner decided. Build judges that from the first part's reading and then gives a go of its own. Then:

- **Launches:** 10 M launches, the same tasks, repetitions and order, from the base's binaries built outside the work folder (`remeasure.py plan --arm M`, launch bound 15).
- **Work folder:** a second one, `markitect-long-lived\remeasure-m`, since a finished series refuses another run. Its `also` names the first part's folder, so its `cost` reads both.
- **Reading:** each task's fresh M launches are paired with its L launches from the first part, on the same machine and engine, close in time. Both bars are then read, wall time and input with cache reads.
- **Everything else:** as for the first part, under this page. The quota cap is set again before the window.

## Amendments

### Amendment 1, 31 October 2026: the reading

**The answer is no.** The fix took effect: on the Library tasks a continued turn's message fell by about four fifths. But L's input did not fall. Against the comparison's M rows, input with cache reads rose by 98 % over the five tasks, and by 24 % without C3, where the bar asks for −25 %. Quality held by the suite's measures.

**The run:**

- **Launches:** 10 of 10, all results, none void or relaunched. The runner started at 15:02:17 UTC; the quiet-engine guard watched for its first ten minutes, and the launches ran from 15:12 to 16:56.
- **Before the first launch:** the branch's Docker tests passed at `633544ef` on Windows (15:00 to 15:01) and on Linux in WSL (15:01 to 15:02), and left nothing behind.
- **Quota:** `quotaStop` was 0.35, the 0.29 of the second gate's last reading plus 6 points. The seven-day use rose to 0.30.
- **Final check:** no secret hits and nothing left on the engine. The binaries lay outside the work folder. The counted series' three frozen images and `markitect-counted\` were not touched.

#### Quality

Each cell gives the hidden cases passed out of those planned.

| Task | L r1 | L r2 | M in the comparison |
| --- | --- | --- | --- |
| Library M3 | landed, 22/22 | landed, 22/22 | landed both times, 22/22 |
| Library M4 | landed, 28/28 | landed, 28/28 | landed both times, 28/28 |
| Library M5 | landed, 27/27 | landed, 27/27 | landed both times, 27/27 |
| Makerspace W0 | landed, 54/54 | landed, 54/54 | landed both times, 54/54 |
| Defect `makerspace-c3-discount-contradiction` | stopped: model defect after 4 turns | not landed: incomplete after 22 turns | stopped: model defect after 2 and 4 turns |

- **Hidden cases and landing:** equal to M on every launch. §15.4 counts both C3 launches as stopped, since neither landed.
- **C3 r2 did not stop right.** The child `materials` escalated the contradiction between its table and its example. Root's long-lived maker settled it itself ("the table governs") instead of escalating it as a model defect. Each later approved submission of `materials` carried the `model-problem` verdict and escalated again. Root answered each time that the matter was settled, until `materials` had used its six maker turns. M's Root, fresh in each turn, escalated at once in both repetitions. The comparison's C3 r1 on L showed the same pattern for 9 turns before Root escalated. With two repetitions per arm this does not tell whether the fix made it worse.

#### Whether the fix took effect

The mean size of a launch's continued-turn messages, from the host's records in the transcripts:

- **Library:** 1,516 to 2,688 characters per launch, against 8,325 to 11,876 before the fix. Root's second checker turn in M3 r1 fell from 26,030 to 2,348 characters.
- **W0: it grew,** to 29,107 and 38,015 characters, against 7,164 and 17,485 before. A checker's diff since its previous submission is shown inline up to 256 KiB. `protocol`'s third checker turn received 252,818 characters of diff in r1. Before the fix, the whole diff from the base was above the bound there, so only its files were listed.
- **Across all ten launches** the mean is 8,169 characters, against 9,061 before.

#### Cost

Change of L against M's comparison rows per task, with the corrected counter (negative is less). The last column is L before the fix, from the comparison.

| Task | Wall (shown, not read) | Input without cache reads | Cache reads | Input with cache reads | Weighted input | Output | Turns | Containers | Input with cache reads before the fix |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Library M3 | −21 % | −20 % | +5 % | +2 % | −11 % | −13 % | 0 % | −27 % | +3 % |
| Library M4 | −6 % | −11 % | +15 % | +11 % | −1 % | −7 % | 0 % | −27 % | +6 % |
| Library M5 | +3 % | −10 % | +48 % | +41 % | +14 % | +17 % | +25 % | −27 % | +22 % |
| Makerspace W0 | +32 % | +19 % | +45 % | +42 % | +30 % | +20 % | +24 % | −14 % | +40 % |
| `makerspace-c3-discount-contradiction` | +248 % | +76 % | +462 % | +392 % | +196 % | +641 % | +333 % | +12 % | +68 % |
| **Mean over the tasks** | **+51 %** | **+11 %** | **+115 %** | **+98 %** | **+46 %** | **+132 %** | **+76 %** | **−16 %** | **+28 %** |
| Mean without C3 | +2 % | −6 % | +28 % | +24 % | +8 % | +4 % | +12 % | −23 % | +18 % |

- **Weighted input** counts input without cache reads and cache writes at 1 and cache reads at 0.1.
- **The re-sent brief was not what made L's input grow.** At equal turns, L's input with cache reads per launch barely moved with the fix: M3 1.84 M and 1.76 M against 1.84 M and 1.80 M before, M4 1.98 M and 2.02 M against 1.91 M and 1.92 M. A continued turn re-reads its whole conversation on every request, and the brief was a small part of it.
- **Where L cost more:** M5 took 17 and 18 turns against 14. W0 took 30 and 32 against 25, and its checkers read the large inline diffs. C3 r2 looped for 22 turns.
- **Wall time** is shown but not read, since these launches were not paired with M's. Even read as it is, its mean without C3 is +2 %. So the decision does not hang on wall time, and the optional M part would not change the answer.

**Evidence:** `C:\Users\Consiliari\AppData\Local\markitect-long-lived\remeasure`, with the runs, judgments, ledger, log, reports, `cost.json`, `cost.md` and the final check. The Docker tests' logs are `dockertests-remeasure.txt` and `dockertests-remeasure-wsl.txt` beside it.

**On the branch, if it goes on:**

- Inline a continued checker's diff only up to a small bound, and list the files above it.
- End the loop of C3 r2 in host state: a parent's answer that leaves a child's `model-problem` verdict standing cannot settle it, so the next escalation of the same matter should reach the owner as a model defect.

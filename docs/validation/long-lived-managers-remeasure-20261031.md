# Long-lived Managers: re-measure dispatch — 31 October 2026

The bounded dispatch of the re-measure the owner asked for after the [comparison](long-lived-managers-comparison-20261030.md) of the [long-lived Managers](../design/long-lived-managers.md) redesign (**L**) with `main` (**M**). On a continued turn, L's host sent the agent its whole first-turn brief again. The owner, through the Topics session on 31 October 2026: "Yes, fix and re-measure". The order and the scope are Build's [decision](../workstreams/build.md#decisions): only L runs, on 10 launches, read against the comparison's M rows, after the second Windows gate. It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)). **These runs are evidence for the design, never counted data.** The owner decides on adoption after the reading.

**Status: proposed for Build's review.** Nothing runs before the second Windows gate has ended, its results are written, Build has reviewed its Amendment 5 and gives the explicit go for a Docker window. The counted series waits for this re-measure.

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
| Source | **L:** the branch `dev/long-lived-managers` at its reviewed head `633544effd541d3e2ef995ae68e37e9d6ac4c546`, built on the comparison's pinned base `2007156dbf0fe6707f1b49654b745b562cf1fb10`. `markitect.exe` and `markitect-remote` (Linux, amd64) are built from it into `markitect-long-lived\binL-633544ef`, outside the work folder, so the final scan does not read them. **M:** no new launch. Its rows are the comparison's, at the same base, whose code did not change. The images are the comparison's: Manager `sha256:7833b344…`, sandbox `sha256:2216d76e…`. The branch changes no image. If one is no longer on the engine, it is built again from the base under the lock, and the ledger records its ID. The suite's `tools/` tree comes from the base by `git archive` into the work folder's `src/` |
| Tasks | The Library fixed starts M3, M4 and M5, the Makerspace W0 (a one-step chain, as the comparison ran it) and the plant `makerspace-c3-discount-contradiction`. In the comparison, L's input with cache reads grew against M on all five: M3 +3 %, M4 +6 %, M5 +22 %, W0 +40 % and C3 +68 %. C3 is also a defect that must still stop |
| Order | Two repetitions, the task order rotated in the second. **10 launches, all L:** Library 6, Makerspace 4. Launch bound 15, relaunches included. The suite's report reads no contrast, since one arm runs; the reading below compares the arms |
| Quota | The frozen runner's stops, and the comparison's cap (`quotaStop`): Build sets it before the window at the seven-day use then plus 6 points. The comparison's 40 launches used 4 points |
| Runner | `remeasure.py` in Build's notes folder `long-lived/`, SHA-256 prefix `40eb90f4f3b9900e`. It imports `compare.py` (prefix `050ebae65c6c1e08`) and `cost_resumed.py` (prefix `5c7e0083f89af41a`) unchanged, and runs compare.py's runner: the frozen series runner with arm L. `plan` writes the configuration of one arm from a machine file like the comparison's, with `baseline`, the comparison's work folder. It refuses binaries inside the work folder. `units` prints the order and starts nothing. Started detached (`Win32_Process`), without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables. A change to it amends this page first |
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

The reading is an amendment to this page, for Build's review. The owner decides on adoption.

## Optional second part: fresh M pairs

Only if the decision hangs on wall time, as Build's decision says: Build decides after the first part's reading and gives a go of its own. Then:

- **Launches:** 10 M launches, the same tasks, repetitions and order, from the base's binaries built outside the work folder (`remeasure.py plan --arm M`, launch bound 15).
- **Work folder:** a second one, `markitect-long-lived\remeasure-m`, since a finished series refuses another run. Its `also` names the first part's folder, so its `cost` reads both.
- **Reading:** each task's fresh M launches are paired with its L launches from the first part, on the same machine and engine, close in time. Both bars are then read, wall time and input with cache reads.
- **Everything else:** as for the first part, under this page. The quota cap is set again before the window.

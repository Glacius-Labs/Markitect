# Fixed starts: re-measure dispatch — 2 November 2026

The bounded dispatch of the re-measure that follows [Amendment 9](counted-series-protocol-20261030.md#amendment-9-2-november-2026-the-windows-series-ended-and-linux-is-held) of the counted series. In that series, M did not land 7 of the 18 Makerspace fixed starts for one main reason: the arm dropped the module paths that each task's model pages hold, so no Manager could write the model's packages. Build proposed fixing this and running the stuck starts again. The owner: "Yes, fix and re-measure". It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md#decisions)). **These runs are evidence for the fix, never counted data.** A new series needs the owner's own decision.

**Status: approved by Build on 2 November 2026.** It has not started.

## Question

With the fix, does M land the Makerspace fixed starts it missed, with every hidden case passing, and does it still land the Library's?

## What changed

Only the [fix](https://github.com/Glacius-Labs/Markitect/tree/dev/fixed-starts-remeasure) on top of the series' frozen commit `955683de`:

- **The arm.** On an adopted start, the owner's adoption commit also adds the task's model's patterns for new paths to the pages. A pattern stays out if a page already has it or if it matches a file that another page or Root owns, so every file keeps its owner. The base holds the patterns, so the Run's impact stays the model's change alone. The real `markitect check` accepts the result on all 13 public fixed tasks.
- **The remote's refusal.** It now says that a path added to a registry page counts only from the next Run, even once the parent has merged it, and that the parent can make the change itself but cannot give the path. Before, Managers kept trying a claim that could not work.

Nothing else changes, so these launches differ from the series' launches only by the fix, the time of day and chance.

## Dispatch

| Item | Value |
| --- | --- |
| Source | `b1d925b19f9f04bf6142983c5c14798196e5fc1b`, branch `dev/fixed-starts-remeasure`: `955683de` plus the two fix commits. `markitect.exe` and `markitect-remote` (Linux, amd64) are built from it into `markitect-fixed-starts\bin-b1d925b1`, outside the work folder. The suite's `tools/` tree comes from it by `git archive` into the work folder's `src/`. The series runner's SHA-256 is the series' own, `2f32715b…` |
| Images | The series' own, unchanged, by ID: Go Manager `sha256:4f664eb8…` and sandbox `sha256:67e2cb76…`. The fix touches no image |
| Inputs | Prepared by the runner from those tools. All 10 tasks are byte for byte the series' |
| Tasks | M only, 15 launches. The Makerspace fixed starts C1, C2, C4, C6 and C8, two repetitions each: those where M missed in the series, whether in one repetition or both. As a check that the fix does no harm, the Makerspace's C3, C5 and C7 and the Library's M3 and M5, once each: the fix adds patterns there too (10 per Makerspace task, 4 per Library task), and M landed them all in the series. C9 is left out: its model does not change, so no agent starts |
| Order | Library M3, Makerspace C1 to C8 in order at repetition 1, Library M5, then the five steps at repetition 2. Launch bound 22, relaunches included. Relaunches 2, as in the series |
| Quota | The runner's own stops: a refused request or a seven-day use of 90 %. The use was 3 % when the series ended |
| Runner | `remeasure.py` in Build's notes folder `fixed-starts/`, SHA-256 prefix `8b8566cebf83179b`. It runs the frozen tools' runner unchanged, with no contrast and no review, since one arm runs. Started detached (`Win32_Process`) by `markitect-fixed-starts\run.cmd`, without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables. A change to it amends this page first |
| Guard, lock, login | As in the series: the quiet-engine guard, `NATIVE-RUN.lock` and the refresh of the account login. No API key |
| Deadline | As in the series: 5,400 seconds per Makerspace launch and 2,700 per Library launch |
| Abort | The runner's own stops, or the owner's word. Build writes `work\STOP` and the runner ends after the launch |
| Early stop | Build stops after repetition 1 if at least 3 of its 5 stuck steps miss for the old cause: the fix did not work, and the rest would cost more without saying more. The launches so far are read as usual |
| Before the first launch | No Docker test. The remote's change is one message string, `go test ./internal/runremote ./internal/adapters/remoteserver` passes, and the fix touches no image and no engine code |
| Cleanup | The runner's final check, which lists leftovers by the suite's labels. **No** `docker image prune`, `docker system prune` or `docker rmi`, and no retagging of the series' images. **Nothing** under `markitect-counted\` is written. Its `windows` folder is read as the baseline, for Library and Makerspace only |
| The new project | Not touched. It is not in this dispatch, and Build opens none of its material |
| Evidence | Outside the repository: `C:\Users\Consiliari\AppData\Local\markitect-fixed-starts\work`, with the runs, judgments, ledger, log, reports and `read.md` |
| Platform | Windows, natively, on Docker Desktop's engine, as the series. The owner's other Docker work stays off, and Build checks the engine and the process list before the start |
| Time | About 4 to 8 hours if the fix works, and up to about 12 if it does not and the early stop does not apply. In the series, a stuck launch took 25 to 66 minutes, and M's full W0 build 26. A launch that splits the program may take as long as that build |

## Reading

`remeasure.py read` writes each launch beside the same launch in the series, M and K. Frozen before the first launch:

1. **Whether it landed:** per launch, the state and the hidden cases passed.
2. **Whether the fix took effect:** `newPaths` in `run.json`, and how many messages still name a refused path.
3. **The layout:** how many of the nine module packages the result holds, and whether `realization/internal/ms` remains. In the series, no M result held one of them and every K result held all nine. A result that lands without them is reported, not scored.
4. **Cost, described:** wall time, dollars and agent starts per launch, against the series' M and K.

**The bar for proposing a new freeze:** all 13 Makerspace launches land with every hidden case passing, and both Library launches land the same. A launch that misses gets its cause diagnosed before anything else, and the reading names it. Meeting the bar proposes a new freeze and does not start one: that means the gate again and both platforms from their first launch, which [needs the owner](counted-series-protocol-20261030.md#amendment-9-2-november-2026-the-windows-series-ended-and-linux-is-held).

**What the bar is not:** it is what a new freeze needs, not the owner's rule, which is read on the new project's fixed starts. Read on the public Makerspace, that rule would stay "no" whatever happens here: K landed all 18 in the series, while C9 counts as not landed for M, so M reaches at most 16 of 18.

**Limits:**

- Thirteen launches on eight steps of one public project say little about chance, and nothing about the new project.
- K does not run again, so the launches are not paired with it, and wall time compares across times of day.
- There are no reviews, so this reads no confirmed findings. A README is visible only in `read.json`.
- Whether a small change on an adopted start should force the whole layout stays open, since checkers differed on it in the series. This dispatch reports what they do and does not settle it.

The reading is an amendment to this page. The owner decides what follows.

## Amendments

None yet.

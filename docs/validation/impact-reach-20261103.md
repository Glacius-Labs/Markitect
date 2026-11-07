# How far `impact` reaches: one measurement — 3 November 2026

The Topics session measured how much of a project a change reaches at one step of links, at two steps and when following every link, and whether that covers the files that truly changed. It started no agent. The reading is this session's and decides nothing; [draft concept 3.0](../concepts/v3-concept.md#14-decided-on-acceptance), question 3, carries the recommendation.

**Of the 36 existing files that truly changed, one step reached 33. Two steps and every link each reached 35, and every link often took in most of the spec on the way.** The other 5 of the 41 changed files are new tests, which nobody could name in advance.

## Source and method

The fixtures are at commit `42ebd0bee6830f4a0bcdea5a5f7c8df5d721ad48`: `tools/evaluation-tasks/makerspace`, `tools/makerspace-reference` and `tools/evaluation-tasks/makerspace-large`. The first two are also on Main. Only public fixtures were read.

- **Links.** A Definition's `[[...]]` links and its `kind:` line, resolved by path from the root of the spec. Reach follows them backwards, from a Definition to whatever links to it or is of it.
- **Changed.** The Definitions that differ from the step before.
- **One step.** The changed Definitions and what links directly to them, which is how far [impact](https://github.com/Glacius-Labs/Markitect/blob/b24f4ed3b8a1bf7f2eebc3aaf805d7e20c787e2b/docs/guides/model-language.md#impact-and-checklists) follows Definitions today. On c6 this gives the 17 items of `impact`'s real checklist. The measurement leaves out the other part of `impact`, what a changed file realizes.
- **Files reached.** The files whose row in the step's finished registry names a reached Definition, plus files of a changed Definition that the step deleted. The registry is in `tools/evaluation-tasks/makerspace/references/<step>`. This counts a new file as reached, though `impact` before the work reads the registry from before the step, which can't name it. The section on changed files gives both counts.
- **Truly changed.** The files that differ between the step's reference and the one before.

Percentages are rounded down. The scripts were throwaway and are not kept, so the method above is what repeats it. An independent review re-ran them and got the same numbers.

## How much of the spec a change reaches

Makerspace, for each real change step:

| Step | Definitions | Changed | One step | Two steps | Every link backwards | Links in either direction |
|---|---|---|---|---|---|---|
| c1 | 108 | 5 | 9 (8 %) | 47 (43 %) | 93 (86 %) | 104 (96 %) |
| c2 | 111 | 23 | 35 (31 %) | 42 (37 %) | 98 (88 %) | 107 (96 %) |
| c3 | 112 | 2 | 3 (2 %) | 4 (3 %) | 4 (3 %) | 108 (96 %) |
| c4 | 112 | 14 | 24 (21 %) | 58 (51 %) | 58 (51 %) | 108 (96 %) |
| c5 | 113 | 11 | 20 (17 %) | 24 (21 %) | 99 (87 %) | 109 (96 %) |
| c6 | 113 | 12 | 17 (15 %) | 21 (18 %) | 97 (85 %) | 109 (96 %) |
| c7 | 115 | 16 | 17 (14 %) | 20 (17 %) | 20 (17 %) | 111 (96 %) |
| c8 | 115 | 5 | 8 (6 %) | 10 (8 %) | 10 (8 %) | 111 (96 %) |

Step c9 changes no Definition.

The project of about five times Makerspace has one snapshot and no change steps yet, so each of its 413 Definitions was changed on its own:

| Reach | Median | 90th percentile | Largest |
|---|---|---|---|
| One step | 2 | 8 | 269 |
| Two steps | 5 | 25 | 385 |
| Every link backwards | 7 | 70 | 395 |

## How many of the changed files it reaches

With each step's finished registry, so new files count as reached:

| Step | Definitions changed | Files that truly changed | Files reached: one step / two / every link | Missed: one step / two / every link | Files in the project |
|---|---|---|---|---|---|
| c1 | 5 | 5 | 10 / 23 / 28 | 0 / 0 / 0 | 28 |
| c2 | 23 | 9 | 26 / 27 / 29 | 0 / 0 / 0 | 29 |
| c3 | 2 | 2 | 3 / 5 / 5 | 0 / 0 / 0 | 30 |
| c4 | 14 | 11 | 18 / 26 / 26 | 3 / 1 / 1 | 31 |
| c5 | 11 | 5 | 22 / 22 / 32 | 0 / 0 / 0 | 32 |
| c6 | 12 | 5 | 20 / 20 / 32 | 0 / 0 / 0 | 31 |
| c7 | 16 | 0 | 13 / 15 / 15 | 0 / 0 / 0 | 31 |
| c8 | 5 | 4 | 9 / 11 / 11 | 0 / 0 / 0 | 31 |
| **All** | | **41** | | **3 / 1 / 1** | |

All the misses are in c4, which adds a new role. At one step they are `internal/ms/ops_billing.go`, `internal/ms/ops_reports.go` and `internal/ms/helpers_test.go`. Two steps and every link miss only the test helper. A count can exceed the files in the project because it includes a file the step deleted.

With the registry from before each step, which is what `impact` has before the work, one step reaches 33 of the 41 files, and two steps and every link each reach 35. The whole difference is 5 new test files, one each in c1 to c5, and those same misses on existing files remain. After the work, `impact` lists every changed file anyway.

## Reading

- **More depth found little more here, at a real price.** Going from one step to every link found two more files, while on c1, c2, c5 and c6 it took in at least 85 % of the spec. That doesn't hold everywhere: on c3, c7 and c8 every link stays under 20 %. On the larger project the median stays small and a few Definitions reach almost everything.
- **Links in either direction are no use for reach.** They connect 96 % of the spec on every real change.

## Limits

- One reference, and a registry written together with it.
- Makerspace's code sits in one big package, and one step already names many of its files, so file-level reach comes easily here. A project with many modules may behave differently, and the larger project has no change steps to test that yet.
- This counts files reached, not what an agent would have done with them. A miss here may still be caught by the required checks or a full check, and nothing here shows that a reached file gets looked at properly.

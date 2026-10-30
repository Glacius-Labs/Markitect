# Counted series: protocol — 30 October 2026

The protocol of step T7's counted acceptance series ([delivery plan](../implementation-plan.md#concept-20-build)), as section D of the [suite design](../design/evaluation-suite-v2.md#d-counted-evaluations) asks for. It is built from the pilot's [proposal](t7-pilot-20261029.md#proposed-protocol-for-the-counted-series) and its [Q4](t7-pilot-20261029.md#q4-what-must-change-before-a-counted-series), updated for what has changed since. The owner delegated the counted series to Build, and Build approves this protocol after an [independent review](#independent-review), recorded below before any counted run ([build page](../workstreams/build.md), Decisions).

**Status: proposed on 30 October 2026, not yet approved.** No gate or counted launch has run. The [stability gate](#stability-gate) runs only in a window that Build clears with the owner. [Four questions](#for-the-owner) belong to the owner.

## What changed since the pilot's proposal

- **T8** is on `main` ([PR #899](https://github.com/Glacius-Labs/Markitect/pull/899), [DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)): the required checks now refuse only merged states. The Makerspace W0, which no Run landed in the pilot, has since landed natively and passed 54 of 54 cases ([record](t8-makerspace-w0-20261030.md)).
- **T9** is on `main` ([PR #904](https://github.com/Glacius-Labs/Markitect/pull/904), [DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)): the pilot's c3 plant now stops, and the Codex Library M1 has landed ([record](t9-verdict-scope-20261030.md)).
- **The engine** stays Docker Desktop's, as the owner decided on 30 October 2026. Before each window, Build asks the owner to stop all other Docker work at the process level, not only pause an app, and checks that it has stopped. The reruns' guard stays as a second line ([build page](../workstreams/build.md), "Evaluation engine").
- **The runner** is part of the suite, as [series_runner.py](../../tools/evaluation/README.md#the-series-runner) ([PR #908](https://github.com/Glacius-Labs/Markitect/pull/908)), frozen with `tools/`. It builds in Q4's items 2 to 4:
  - the reruns' guard, as standard;
  - the engine lock, held only for a login refresh between launches, never while a suite command runs;
  - each launch's arm passed to the classifier;
  - the login refreshed before a launch it cannot cover.

  It also holds this protocol's order, repetitions, contrasts, relaunch and stop rules, the stability gate and its verdict.
- **The c5 plant** no longer leaks (Q4, item 5). Its record now also removes the line "An invoice whose `dueAt` is `2099-12-01T23:59Z` or later can never block", which pinned the removed 30 days, and it checks that the line is gone (PR #908).
- **The minimum meaningful difference** is frozen at 0.05 before the first counted run, unless the owner sets another value by then.

## Preconditions

All of these hold before the first gate launch. The gate's frozen commit is the series' frozen commit.

1. **The runner:** PR #908 is on `main`.
2. **Plants:** every plant has been checked for consequences of what it removes (Q4, item 5).
   - The public plants were checked by the pilot and by PR #908.
   - The new project's author reported on 30 October 2026 that each of its three plants has been checked. The gap defect leaves no other sentence, example or number that pins what it removes, in `task.md`, the model or its start's model. The contradiction keeps its opposing statements on purpose. The landing gap stays determined by the remaining rules.
3. **The new project's launch inputs** are written from the frozen commit, before its hidden material leaves the machine.
   - The tool is its author's `suite_inputs.py inputs --suite PUBLIC_REPO --phase launch --deadline-seconds 5400`, with the frozen commit as `PUBLIC_REPO`, into `inputs/gridshare` in each platform's work folder. It writes only what variants get: the steps, the reference starts, the base models with the registry pages of its fixed starts, and the tasks from the frozen `suite.py task`. Its `inputs.json` lists sizes and digests and no plant kinds.
   - A spec file names the series for the runner: steps W0 to C7, fixed starts C1 to C7, the plants `gridshare-plant-1` to `-3` at the steps C2, C6 and C7, `inputs` and `judges`. Each plant starts from the same start as its step's change.
   - A new frozen commit means writing the inputs again. That needs the private clone, which is removed again before the next launch.
   - On 30 October 2026, at public `main` `c575faa5` and private `c87aa81`, the author reported that every step and adopted start passes `markitect check` with no `lint.no_paths` warning, and that the launch and judge phases match the combined output digest for digest.
4. **Hidden material is off the machine.** After the launch inputs are written, none of the places listed under [Hidden material](#hidden-material) exists. The authoring session's own records hold the whole material, and **only the owner can remove them**: by exporting the authoring session and then deleting it from the machine. Build asks the owner before the gate.
5. **The stability gate** has passed on the frozen commit for the platform to be run.
6. **The engine:** for each window, the owner has stopped all other Docker work at the process level. Build has checked that `docker ps` shows no container without a `markitect.*` label and that no Testcontainers or `dotnet` process runs.

## Protocol

| Item | Value |
| --- | --- |
| Tasks | The Library, the Makerspace and the new project, as the build page plans and §15 names. Every fixed start and plant runs as an independent task from its reference start. M gets the start's registry pages, K the same start without them. Every chain runs whole. The new project (`gridshare`) is the confirmatory part, unseen by anyone who works on an arm. It has a fresh build W0 and seven changes C1 to C7, and three plants, two defects and one gap, whose names do not say what they plant. Its cumulative hidden cases per step are W0 34, C1 37, C2 39, C3 43, C4 47, C5 49, C6 51 and C7 52. The Library and the Makerspace give §15's tables for both arms. M was tuned on the Makerspace, so its results there are descriptive ([suite design](../design/evaluation-suite-v2.md#c-portfolio), section C) |
| Arms | **M:** `markitect_arm.py` with `suite.py --native`, effort `high`, the host's limits per container. **K:** `claude_code.py --effort high` with `suite.py --sandbox` and the series' safety limit. P and G are not counted |
| Model | `claude:claude-sonnet-5-5` for both arms. Account login only, never an API key |
| Frozen revisions | One `main` commit that meets the preconditions, named in an amendment before the first gate launch. `markitect`, `markitect-remote`, the Manager and Go Manager images and the sandbox image are built from it under the lock and named by image ID. `tools/` is extracted from it by `git archive` into the work folder's `src/`, and the runner runs from there. The runner's configuration is written by `plan` from that tree. At its first window the runner records a digest of the whole tree, the binaries' SHA-256, the new project's launch inputs and the images. Each later window checks them and stops on any difference. The series' configuration names the gate's ledger (`gateLedger`), so its first window also checks them against what the gate ran. A new commit means a new gate, a restart (see [After a stop](#after-a-stop)) and an amendment first. Each platform's reading names the commit it ran on |
| Repetitions | Per arm, as the runner's `counted` plan: the new project's changes five times each, its plants three times and its chain twice; the Library's and Makerspace's fixed starts and plants twice each, and their chains once |
| Run order | By repetition, from the first to the fifth. Each repetition runs every fixed start and plant whose count reaches it. The two arms of a task run one after the other. The first arm alternates per task and per repetition; M runs first in 18 of the new project's 35 change pairs. The task order is rotated per repetition. Repetitions 1 and 2 have 56 launches each, repetition 3 has 20, repetitions 4 and 5 have 14 each. Then the chains run, by repetition, with the first arm alternating per chain: the Library, the Makerspace, the new project, then the new project again. That makes 62 chain launches |
| Platforms | Windows first, natively. Then Linux, natively in WSL Ubuntu against the same engine, as §15 asks. Each platform has its own gate, work folder, ledger and reports, and its contrasts are read separately |
| Deadlines | Per step: 2700 s for the Library and 5400 s for the Makerspace, as in the pilot; 5400 s for the new project, as its author set it, fixed at the freeze. The new project's tasks are 182 to 241 KB each, its plants 189 to 240 KB. A timeout is a result. The report's Delivery area shows timeouts per arm |
| Budget | Per arm and platform: the Library 17 launches (4 fixed starts and 2 plants twice each, plus a 5-step chain), the Makerspace 34 (9 fixed starts and 3 plants twice each, plus a 10-step chain), the new project 60 (7 changes five times, 3 plants three times, and an 8-step chain twice). That is 222 launches per platform and 444 in all. The launch bound is 333 per platform, one and a half times the planned launches; relaunches count against it |
| Quota and time | In the pilot, 57 launches used 4 points of the seven-day quota. At that rate the series needs about 31 points; if M now lands its Makerspace and new-project steps, two to three times that. A seven-day use of 90 % ends a window, and the next one starts after the reset. So the series may span two or three seven-day windows, and the owner's own use shares that quota. By the pilot's times and T8's W0 (24 minutes for M), one platform takes about 50 to 80 hours of launches. The gate gives a better figure |
| Windows | Build clears each window with the owner. `WORK/STOP` ends a window before the next launch or during any wait, also within a chain, between its steps. When the runner is started again, it continues where it stopped and does not launch again a launch that counted. The ledger records each window. Build stops the runner only through `STOP`, never by ending its process |
| Before each launch | The runner checks, in order: no `STOP` file, the launch bound, and no hidden material on the machine. Then it waits for a quiet engine and for a login that outlives the deadline by 20 minutes, and checks the engine again. **Quiet** means no container without a `markitect.*` label runs, and none started in the last ten minutes, as a `docker events` stream kept open for the whole window shows. **The refresh** happens under the engine lock. It sets the access token's expiry to now and changes no token, then lets `claude -p` refresh the login, as the CLI does near expiry. The runner replaces the login file only if no other process changed it in between, and it puts the expiry back if the CLI did not refresh it |
| Lock | `NATIVE-RUN.lock` in Build's notes folder. `suite.py` takes it for each run, judgment and review. The runner holds it only for a login refresh. A busy lock is waited for and not counted |
| Technical failures | Classified by `classify.py` with the arm's signatures, written before any launch (`signatures.json`). For M: `reported.status` is `could-not-run` or `check-failed`, or `reported.noAgentTurn` is true. Provider and infrastructure voids are as `classify.py` defines them. A void launch is relaunched at most twice |
| Invalid launches | A launch invalid only through containers of other work is relaunched once. Those are containers without a `markitect.*` label whose start the run's own engine events show. Any other invalid launch stops the series. Both stay in the ledger as invalid. A chain ends at an invalid step |
| Stop rules | The series stops on any of these: a refused request or a seven-day use of 90 %; a login that cannot be refreshed; a credential of the login in a run's evidence; a gateway refusal of a provider name; hidden material on the machine; an invalid launch as above; a void share above one in four for either arm after its first eight launches; a judge that gives no valid judgment twice; a frozen file, binary or image that changed; the launch bound; the runner's own failure. A five-hour use of 85 % waits for that window's reset instead. Every stop writes its reason to `STOP`. What happens next is fixed below, under [After a stop](#after-a-stop) |
| Judging and review | Each delivered result is judged at its step, and each chain's counted steps as one chain (`judge-chain`). A judgment that is not valid is made once more, and a second failure stops the series. The code review covers every judged result of a fixed start in repetition 1 and the last judged step of every chain, blinded, in the sandbox: per platform, 40 fixed-start reviews and up to 8 chain reviews. On the new project a judgment takes about 6 to 7 seconds |
| Cleanup | The suite removes its gateway and networks and the arm's leftovers, and counts them. The host removes each Run's remote, containers and volumes. After each window, the runner counts what is left labelled `markitect.run` or `markitect.evaluation`, and scans every evidence file for the login's tokens and key shapes |
| Evidence | Outside the repository: on Windows `C:\Users\Consiliari\AppData\Local\markitect-counted\windows` and `…\gate-windows`; on Linux `~/markitect-counted/linux` and `~/markitect-counted/gate-linux` in WSL. Each holds the runs, judgments and reviews, the ledger with its windows, waits and failures, the log, the reports, the final scans and, for the gate, `gate.json` |

## Hidden material

Nothing of the new project's hidden material is on the machine during any launch on either platform. The runner's configuration lists every one of these places under `hidden`. Before each launch the runner stops if any of them exists, and M's runs check them again (`--hidden-absent`):

- the private clone, `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation-hidden`;
- the author's run evidence, which holds judge inputs, `C:\Users\Consiliari\Documents\Codex\gridshare-runs`;
- the pilot's three possible clone places: `C:\Users\Consiliari\Glacius Labs\Markitect-Evaluation`, `C:\Users\Consiliari\Markitect-Evaluation` and `C:\Users\Consiliari\Glacius Labs\Markitect\.claude\worktrees\markitect-v1-concept-review-e8323e\Markitect-Evaluation`;
- the authoring session's records:
  - its transcript `282a2d17-98a7-460b-a39d-73891191a40b.jsonl`, and the folder of the same name beside it, in `C:\Users\Consiliari\.claude\projects\C--Users-Consiliari-Glacius-Labs-Markitect--claude-worktrees-markitect-v1-concept-review-e8323e\`;
  - its file history, `C:\Users\Consiliari\.claude\file-history\282a2d17-98a7-460b-a39d-73891191a40b`;
  - its temporary folder, `C:\WINDOWS\TEMP\claude\C--Users-Consiliari-Glacius-Labs-Markitect--claude-worktrees-markitect-v1-concept-review-e8323e\282a2d17-98a7-460b-a39d-73891191a40b`;
- each platform's judge folder for the new project, `judges/gridshare` in its work folder, which `suite_inputs.py --phase judge` writes only when the series is judged.

On Linux the configuration lists the same Windows places under `/mnt/c/`, which the WSL host can read, and the WSL folders of the author's Linux self-test, each by its absolute path (the runner also expands a leading `~`). The WSL home as a whole is not on the list, because the Linux work folder lives there.

Before the gate, one more check covers places nobody listed. The author names a file that only the hidden material contains, by name and not by content. A search for that name, bounded in time, runs over the user profile, `C:\WINDOWS\TEMP\claude` and the WSL home, and must find nothing.

The new project is judged and reviewed only after the last launch of the series on both platforms. Until then its judgments stay pending, and the runner leaves the new project out of every report, so that no count of it, not even what was delivered or timed out, is read while launches still run. Two more reasons: a review copies the rubric into the work folder, and a judgment holds the IDs of failed cases, so neither may exist while launches still run.

**Who handles the new project's material:** Build never opens the new project's inputs or run evidence. When the last launch is done, someone who works on no arm clones the private repository and writes the judge phase with `suite_inputs.py --phase judge` into each platform's `judges/gridshare`. Then `series_runner.py judge` runs for each platform; on Linux, that is in WSL. That person also diagnoses a stop on one of the new project's launches and writes its inputs again after a new commit. Build reads the reports. The Library and Makerspace judges are public, so their launches are judged as they end.

## Stability gate

Before each platform's series, on the frozen commit, uncounted ([suite design](../design/evaluation-suite-v2.md#d-counted-evaluations), section D):

| Item | Value |
| --- | --- |
| Units | The runner's `gate` plan, in its own work folder. Per arm: the Library fixed starts M2, M3 and M4 in a row; the new project's C1 once, which shows that both arms run on its inputs, and stays pending and unjudged; then the Makerspace chain W0 to C9. The first arm alternates per task, and K starts the chain. 28 launches, launch bound 42 |
| Passes when | The runner's verdict in `gate.json` holds every check: every launch classed a result; every delivered public result with a valid judgment; every chain with its chain judgment; no credential in the evidence; nothing left on the engine; and M's Makerspace chain landed its first step, W0. That last check shows that DEC-087 holds in a chain, and that M's later steps do not start from an empty start, as they did in the pilot |
| Fails | The runner stops the gate at its first launch that is not a result, or at a judge that fails twice. **A provider void**, such as capacity or a rate limit, needs no fix: the gate starts again on the same commit, in a new work folder. **Any other failure** is fixed in the arm, the suite, a judge or the runner, and is not counted. Then a new commit is frozen, this page is amended, and the gate starts again from its first launch. If M lands no W0, Build looks first |
| Record | The gate's launches never count. Its verdict and its time per launch are recorded as an amendment here |

## After a stop

Each stop leaves its reason in `STOP`. Build removes `STOP` and starts the runner again only as follows:

| Stop | Next |
| --- | --- |
| The end of a window, a seven-day use of 90 % (after its reset), or a login that could not be refreshed (after the owner logs in again) | Resume on the same commit |
| A credential in the evidence | The file is removed, and Build checks how it got there. With no code change, resume on the same commit |
| Hidden material on the machine, or a second launch invalid through other work's containers | The machine is cleared, or the owner's other work is stopped again. Resume on the same commit |
| An invalid launch not caused by other work, a void share above one in four, a judge that fails twice, a changed frozen file, or the runner's own failure | A defect of an arm, the suite, a judge or the runner. If it needs a change to the frozen commit, that platform's launches so far stay in the record, but they are reported apart and not counted. A new commit is frozen, the gate passes again, this page is amended, and the platform starts again from its first launch in a new work folder. Without a change, resume on the same commit |

**The launch that stopped the series:**

- **Invalid, on resume:** the runner launches it again once. If that launch is invalid again, its unit stays without a result. The report counts it under Unpaired in the contrast, so it is never left out silently.
- **Void with relaunches left:** relaunched, as before.
- **Anything else:** stays as it is, and the series goes on with the next unit.

**What can be seen during the series:** results seen during the series cannot change what is measured, because any change to the frozen commit restarts the platform. Build sees the Library's and Makerspace's results window by window, and the new project's only after the last launch.

## Reading

- **Primary contrast:** per platform, M against K on the new project's fixed starts. It is read from the runner's report on that project's fixed starts alone, `report/gridshare-fixed`, without plants or chains.
  - **Endpoint:** the share of the hidden cases judged at the change's step that pass, the earlier steps' cases included.
  - **Pairs:** a change and a repetition, 35 per platform.
  - **The interval:** the 95 % interval of the mean paired difference, read over the 7 changes' mean differences (`blockBy` `change`), since a change's five repetitions are not independent. The interval is wide with only seven changes, and the result may well be undecided.
  - **Reading:** better, worse, "a meaningful advantage is excluded", or undecided. M is at least as good when the interval's lower end lies above minus the minimum meaningful difference.
- **Minimum meaningful difference: 0.05** of the cases judged at a step. On the new project that is two to three of 37 to 52 cases. It is frozen at 0.05 before the first counted run unless the owner sets another value by then, so it is never chosen after seeing results. The pilot's report used it, and the pilot gave no spread to size it from.
  - **Limit:** because the endpoint counts the earlier steps' cases, a change adds only a few cases of its own (W0 34, then 3, 2, 4, 4, 2, 2 and 1). So a change that stops wrongly and delivers its start loses only its own new cases. At C7 that is 1 of 52, less than 0.05.
  - **The second contrast** reads the failed hidden cases per change (`casesFailed`, with a minimum meaningful difference of 1 case), also over changes, and so shows what the share dilutes. A change adds 1 to 4 cases of its own, so one case is a quarter to all of a change's own cases, and one wrong stop at C7 equals it. §15.1 shows the wrong stops directly: each arm's landed changes per task. Which of the two contrasts is primary is the [owner's question](#for-the-owner).
- **Secondary:**
  - the same contrasts on the fixed starts of the Library (8 pairs, 4 changes) and the Makerspace (18 pairs, 9 changes), descriptive;
  - the chains' paired steps (`report/NAME-chains`), descriptive;
  - defects left for a person (`defectsLeft`, with a minimum meaningful difference of 1), from the reviewed results of repetition 1;
  - §15.1 to §15.4 per arm and series, with planted tasks under Stops right only;
  - the chains' drift and regressions;
  - cost, and the organization's cost per 2.0 Run, described. The organization's cost comes from the pilot's `manager_costs.py`, frozen by hash in the amendment.
- **No drift without M's own `verify --full`:** §15.2 reads No drift from an independent `verify --full` after the series. This series does not run one. T9's `verify --full` gave two false Findings out of three, and `verify`'s checkers do not get the required checks' results yet ([build page](../workstreams/build.md), Next). So No drift rests on the series judge's drift checks: forgotten rules, stale files, unaccounted files and regressions along each chain. The report shows the verify slot as empty. Whether that is enough is the [owner's question](#for-the-owner).
- **Stated limits:**
  - The reviewers may guess the arm; in the pilot they guessed it every time. Every reviewer is from the writers' model family.
  - Tokens of agents that a Run ends are counted only if that fix is on `main` at the freeze.
  - Human attention is measured by its stand-ins.
- **What it claims:** only what it measures, as §15 says. It may show at least K's quality and fewer defects left for a person per change at comparable input. It does not show time saved.

## For the owner

Build approves this protocol under the owner's delegation. Four points are the owner's to decide, and Build asks the owner about them before the freeze:

1. **The primary endpoint and its minimum meaningful difference.** Recommended: keep the share of hidden cases passed with 0.05, as decided, and read the failed-case count beside it. The alternative is to make the failed-case count primary, with a minimum meaningful difference of 1 case, because it does not dilute on changes that add few cases.
2. **No drift without M's own `verify --full`**, as above.
3. **The quota:** the series takes about 31 to 90 points of the owner's seven-day quota, over two or three weeks of windows.
4. **Section D of the suite design** said that the owner approves each protocol. This page brings that line in line with the delegation of 29 October 2026.

## After the series

1. The owner's blinded acceptance of three changes per arm on the new project, with minutes and corrections (§15).
2. The Codex Luna repeat on Windows ([build page](../workstreams/build.md), Codex runs), as its own amendment of this page. Two things come first: the suite's native mode must copy a Codex login, which today it does only for Claude ([Codex smoke](codex-library-smoke-20261030.md)), and K needs its Codex command. Its proposed size is the new project for both arms: two repetitions per change, one per plant and one chain per arm, which is 50 launches.

## Independent review

An independent reviewer reviewed the first draft of this page on 30 October 2026 against the suite design, §15, the build page's decisions, the pilot and the runner's code. It had no access to the new project's content or to a login. It found the counts, the budget, the arm balance and the agreement with the build page sound. It did not approve the draft as it stood. Its findings and what became of them:

| Finding | Handling |
| --- | --- |
| The primary endpoint dilutes: it counts earlier steps' cases, so a wrong stop scores close to a correct change; "one case in twenty" misstated it | Stated as a limit under Reading. The failed-case count was added as a second contrast. The choice of primary went to the owner, with a recommendation |
| The fixed-start pairs included gap plants, and there was no chain-only reading | The runner's `report/NAME-fixed` now holds fixed starts only, and `report/NAME-chains` the chains. The pair counts were corrected |
| Build would see results mid-series and decide freely after a stop | The rules under [After a stop](#after-a-stop): any change to the frozen commit restarts the platform. The new project is judged only after the last launch |
| The gate was not mechanical: a launch invalid through other work was relaunched, a judge failure did not stop, M's W0 was not checked, and a provider void had nothing to fix | The gate stops at any launch that is not a result and at a judge that fails twice, and it writes `gate.json` with every check, M's W0 included. A provider void restarts the gate on the same commit |
| The interval treated a change's 35 repetition pairs as independent | The contrasts are read over the changes' mean differences (`blockBy` `change` in the suite's acceptance reading) |
| c5 could not be left out, since the runner plans every plant | The c5 plant was fixed instead (PR #908) |
| The freeze was claimed but not checked | The runner records the tree digest, the binaries and the images at its first window and stops on any change |
| No arm had run on the new project before the counted part; its inputs were not re-prepared after a gate fix | The gate runs the new project's C1 for both arms. A new commit means writing the inputs again. The registry pages of its fixed starts are a precondition |
| The hidden places were not named exactly, and "the WSL home" cannot be one | Every place is listed per platform under [Hidden material](#hidden-material) |
| The `defectsLeft` contrast was not named | Named as secondary |
| The new project's deadline was the author's, and a timeout counts against the slower arm | Fixed at 5400 s at the freeze; the report shows timeouts per arm |
| `verify --full` was left open, and T9's gave false Findings | Decided for this series: not run, a stated limit, and the owner's question |
| For the owner: the endpoint and its difference, dropping `verify --full`, the quota, and section D's wording | Listed under [For the owner](#for-the-owner) |

The same reviewer then checked the revision against this page and the code. It found that every handling above matches the code, and it checked the numbers by running `plan`: the gate has 28 launches with a bound of 42, the series 222 per platform with a bound of 333, the repetitions 56, 56, 20, 14 and 14 launches, and M runs first in 18 of the new project's 35 pairs. It would approve once its first two remaining findings were fixed:

| Finding | Handling |
| --- | --- |
| Build could still see the new project before its last launch: delivered and timed-out counts in the reports per window, and its run evidence when diagnosing a stop. Nobody was named to write its inputs again | The runner leaves a series judged later out of every report until it is judged. Build never opens the new project's inputs or run evidence. Someone who works on no arm diagnoses its stops and writes its inputs again |
| An invalid launch the arm caused dropped out silently on resume | It is launched again once. If that launch is invalid again, the unit stays without a result and is counted under Unpaired |
| More hidden places: the authoring session's temporary folder and other files, and the WSL folders of its Linux self-test; a literal `~` was never expanded | The temporary folder is listed. A bounded search for a marker file the author names covers the rest. The runner expands `~`, and the page asks for absolute paths |
| `casesFailed`'s minimum of one case is a large share of a change's own cases | Stated under Reading, with §15.1's landed changes as the direct measure of wrong stops |
| The freeze was checked within one work folder only, and the inputs had no digest | The freeze includes the new project's launch inputs, and the series checks them against the gate's record (`gateLedger`). Each platform's reading names its commit |

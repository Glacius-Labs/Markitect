# Counted series: protocol — 30 October 2026

The protocol of step T7's counted acceptance series ([delivery plan](../implementation-plan.md#concept-20-build)), as section D of the [suite design](../design/evaluation-suite-v2.md#d-counted-evaluations) asks for. It is built from the pilot's [proposal](t7-pilot-20261029.md#proposed-protocol-for-the-counted-series) and its [Q4](t7-pilot-20261029.md#q4-what-must-change-before-a-counted-series), updated for what has changed since. The owner delegated the counted series to Build, and Build approves this protocol after an [independent review](#independent-review), recorded below before any counted run ([build page](../workstreams/build.md), Decisions).

**Status: approved by Build on 30 October 2026,** under the owner's delegation, after the [independent review](#independent-review) and its second round, on `main` at `6174d5da` (PRs [#908](https://github.com/Glacius-Labs/Markitect/pull/908), [#910](https://github.com/Glacius-Labs/Markitect/pull/910) and [#911](https://github.com/Glacius-Labs/Markitect/pull/911)). No counted launch has run; the [preconditions](#preconditions) come first, and [Amendment 1](#amendment-1-30-october-2026-the-frozen-commit) named the frozen commit. [Amendment 2](#amendment-2-30-october-2026-scoring-fixes-before-the-gate) fixes the scoring before the gate, records the owner's rule for "at least as good" and replaces that commit. The Windows gate ran on 30 October 2026 and did not pass, because M did not land the Makerspace W0 ([Amendment 3](#amendment-3-30-october-2026-the-windows-gate)). [Amendment 4](#amendment-4-31-october-2026-a-new-freeze-for-the-second-gate) freezes `955683de`, which holds the fix, for a second Windows gate. The [stability gate](#stability-gate) runs only in a window that Build clears with the owner. The owner decided the [four points](#decided-by-the-owner) that were the owner's on 30 October 2026.

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
- **Tokens of ended agents** are counted ([PR #907](https://github.com/Glacius-Labs/Markitect/pull/907)): a Result now counts the tokens of agents a Run ends. For Claude that is the exact input and a lower bound of the output. A stopped Codex turn counts 0. Such a count is marked partial, and the page says "at least".
- **`verify` gets the required checks as evidence** ([PR #909](https://github.com/Glacius-Labs/Markitect/pull/909)): it runs the required checks on its start before its checkers start, gives them the results, and names a missing check directory. With that, §15.2's independent `verify --full` joins the series ([No drift](#reading)).
- **The minimum meaningful difference** is 0.05, as the owner decided on 30 October 2026, frozen before the first counted run.

## Preconditions

Preconditions 1 to 4, 6 and 7 hold before the first gate launch, and precondition 5 before the first counted launch. The gate's frozen commit is the series' frozen commit.

1. **The runner:** PR #908 is on `main`.
2. **Plants:** every plant has been checked for consequences of what it removes (Q4, item 5).
   - The public plants were checked by the pilot and by PR #908.
   - The new project's author reported on 30 October 2026 that each of its three plants has been checked. The gap defect leaves no other sentence, example or number that pins what it removes, in `task.md`, the model or its start's model. The contradiction keeps its opposing statements on purpose. The landing gap stays determined by the remaining rules.
3. **The new project's launch inputs** are written from the frozen commit, before its hidden material leaves the machine.
   - The tool is its author's `suite_inputs.py inputs --suite PUBLIC_REPO --phase launch --deadline-seconds 5400`, with the frozen commit as `PUBLIC_REPO`, into `inputs/gridshare` in each platform's work folder. It writes only what variants get: the steps, the reference starts, the base models with the registry pages of its fixed starts, and the tasks from the frozen `suite.py task`. Its `inputs.json` lists sizes and digests and no plant kinds.
   - A spec file names the series for the runner: steps W0 to C7, fixed starts C1 to C7, the plants `gridshare-plant-1` to `-3` at the steps C2, C6 and C7, `inputs` and `judges`. Each plant starts from the same start as its step's change.
   - A new frozen commit keeps the launch inputs, and must leave their digests unchanged. Before its gate, `inputs.json` keeps its SHA-256, every digest it records matches the files, and the new commit's `suite.py task` rebuilds each fixed start's task to the author's digest. Nobody clones the private repository again before the judge phase. A change that would alter the inputs is not made before the series ends ([Amendment 2](#amendment-2-30-october-2026-scoring-fixes-before-the-gate)).
   - On 30 October 2026, at public `main` `c575faa5` and private `c87aa81`, the author reported that every step and adopted start passes `markitect check` with no `lint.no_paths` warning, and that the launch and judge phases match the combined output digest for digest.
4. **Hidden material is off the machine.** After the launch inputs are written, none of the places listed under [Hidden material](#hidden-material) exists. The authoring session's own records hold the whole material, and **only the owner can remove them**: by exporting the authoring session and then deleting it from the machine. Build asks the owner before the gate.
5. **The stability gate** has passed on the frozen commit for the platform to be run.
6. **The verify launch is built,** as [No drift](#reading) describes: a `verify` mode in `markitect_arm.py`, and the runner launching it once per chain and arm ([PR #911](https://github.com/Glacius-Labs/Markitect/pull/911), tested offline). It is on `main` before the freeze.
7. **The engine:** for each window, the owner has stopped all other Docker work at the process level. Build has checked that `docker ps` shows no container without a `markitect.*` label and that no Testcontainers or `dotnet` process runs.

## Protocol

| Item | Value |
| --- | --- |
| Tasks | The Library, the Makerspace and the new project, as the build page plans and §15 names. Every fixed start and plant runs as an independent task from its reference start. M gets the start's registry pages, K the same start without them. Every chain runs whole. The new project (`gridshare`) is the confirmatory part, unseen by anyone who works on an arm. It has a fresh build W0 and seven changes C1 to C7, and three plants, two defects and one gap, whose names do not say what they plant. Its cumulative hidden cases per step are W0 34, C1 37, C2 39, C3 43, C4 47, C5 49, C6 51 and C7 52. The Library and the Makerspace give §15's tables for both arms. M was tuned on the Makerspace, so its results there are descriptive ([suite design](../design/evaluation-suite-v2.md#c-portfolio), section C) |
| Arms | **M:** `markitect_arm.py` with `suite.py --native`, effort `high`, the host's limits per container. **K:** `claude_code.py --effort high` with `suite.py --sandbox` and the series' safety limit, and the task without its registry pages (`--without-pages`). P and G are not counted |
| Model | `claude:claude-sonnet-5-5` for both arms. Account login only, never an API key |
| Frozen revisions | One `main` commit that meets the preconditions, named in an amendment before the first gate launch. `markitect`, `markitect-remote`, the Manager and Go Manager images and the sandbox image are built from it under the lock and named by image ID. `tools/` is extracted from it by `git archive` into the work folder's `src/`, and the runner runs from there. The runner's configuration is written by `plan` from that tree. At its first window the runner records a digest of the whole tree, the binaries' SHA-256, the new project's launch inputs and the images. Each later window checks them and stops on any difference. The series' configuration names the gate's ledger (`gateLedger`), so its first window also checks them against what the gate ran. A new commit means a new gate, a restart (see [After a stop](#after-a-stop)) and an amendment first. Each platform's reading names the commit it ran on |
| Repetitions | Per arm, as the runner's `counted` plan: the new project's changes five times each, its plants three times and its chain twice; the Library's and Makerspace's fixed starts and plants twice each, and their chains once |
| Run order | By repetition, from the first to the fifth. Each repetition runs every fixed start and plant whose count reaches it. The two arms of a task run one after the other. The first arm alternates per task and per repetition; M runs first in 18 of the new project's 35 change pairs. The task order is rotated per repetition. Repetitions 1 and 2 have 56 launches each, repetition 3 has 20, repetitions 4 and 5 have 14 each. Then the chains run, by repetition, with the first arm alternating per chain: the Library, the Makerspace, the new project, then the new project again. That makes 62 chain launches. Each chain's verify launch runs right after the chain's last step, 8 in all |
| Platforms | Windows first, natively. Then Linux, natively in WSL Ubuntu against the same engine, as §15 asks. Each platform has its own gate, work folder, ledger and reports, and its contrasts are read separately |
| Deadlines | Per step: 2700 s for the Library and 5400 s for the Makerspace, as in the pilot; 5400 s for the new project, as its author set it, fixed at the freeze. The new project's tasks are 182 to 241 KB each, its plants 189 to 240 KB. A timeout is a result. The report's Delivery area shows timeouts per arm |
| Budget | Per arm and platform: the Library 17 launches (4 fixed starts and 2 plants twice each, plus a 5-step chain), the Makerspace 34 (9 fixed starts and 3 plants twice each, plus a 10-step chain), the new project 60 (7 changes five times, 3 plants three times, and an 8-step chain twice). That is 222 launches per platform and 444 in all. On top come the verify launches, one per chain and arm: 8 per platform and 16 in all. So the series has 230 launches per platform and 460 in all. The launch bound is 345 per platform, one and a half times the planned launches; relaunches count against it |
| Quota and time | In the pilot, 57 launches used 4 points of the seven-day quota. At that rate the series needs about 31 points; if M now lands its Makerspace and new-project steps, two to three times that. A seven-day use of 90 % ends a window, and the next one starts after the reset. So the series may span two or three seven-day windows, and the owner's own use shares that quota. By the pilot's times and T8's W0 (24 minutes for M), one platform takes about 50 to 80 hours of launches. The gate gives a better figure. The verify launches add little. In T5 a Library `verify --full` took about a minute with five checker starts. The Makerspace and the new project have more Packages, so their verify launches likely take several minutes each. That is under an hour per platform and under a point of quota |
| Windows | Build clears each window with the owner. `WORK/STOP` ends a window before the next launch or during any wait, also within a chain, between its steps. When the runner is started again, it continues where it stopped and does not launch again a launch that counted. The ledger records each window. Build stops the runner only through `STOP`, never by ending its process |
| Before each launch | The runner checks, in order: no `STOP` file, the launch bound, and no hidden material on the machine. Then it waits for a quiet engine and for a login that outlives the deadline by 20 minutes, and checks the engine again. **Quiet** means no container without a `markitect.*` label runs, and none started in the last ten minutes, as a `docker events` stream kept open for the whole window shows. **The refresh** happens under the engine lock. It sets the access token's expiry to now and changes no token, then lets `claude -p` refresh the login, as the CLI does near expiry. The runner replaces the login file only if no other process changed it in between, and it puts the expiry back if the CLI did not refresh it |
| Lock | `NATIVE-RUN.lock` in Build's notes folder. `suite.py` takes it for each run, judgment and review. The runner holds it only for a login refresh. A busy lock is waited for and not counted |
| Technical failures | Classified by `classify.py` with the arm's signatures, written before any launch (`signatures.json`). For M: `reported.status` is `could-not-run` or `check-failed`, or `reported.noAgentTurn` is true. For K: `reported.status` is `error`, which `claude_code.py` writes when its own code fails, also when it cannot capture the result, for example from a repository the agent damaged; such a launch is relaunched like any void. Provider and infrastructure voids are as `classify.py` defines them. A void launch is relaunched at most twice |
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

**Who handles the new project's material:** Build never opens the new project's inputs or run evidence. The author writes the first launch inputs before the authoring session is removed (preconditions 3 and 4). After that, a fresh session that works on no arm handles the material. A session that clones the private repository becomes a place of hidden material itself: its transcript, file history and temporary folder stay on the machine, as the authoring session's do.

- **After the last launch (the judge phase):**
  - Build starts the session only then. Its brief says that it reports to Build only its completion status and the paths of the report files, never content. Build reads only the reports.
  - The session clones the private repository and writes the judge phase with `suite_inputs.py --phase judge` into each platform's `judges/gridshare`. Then it runs `series_runner.py judge` for each platform; on Linux, that is in WSL.
  - Its records stay on the machine until the owner removes them, which is a precondition of the Codex repeat.
- **During the series, as an exception:** a stop on one of the new project's launches may have to be diagnosed, or its inputs written again after a new commit.
  - Such a session is started for that task alone. The amendment that names a new commit also names who writes the inputs again.
  - Before the next launch, the owner exports that session off the machine and deletes it here, and its transcript, file history and temporary folder are added to `hidden`.

Build reads the reports. The Library and Makerspace judges are public, so their launches are judged as they end.

## Stability gate

Before each platform's series, on the frozen commit, uncounted ([suite design](../design/evaluation-suite-v2.md#d-counted-evaluations), section D):

| Item | Value |
| --- | --- |
| Units | The runner's `gate` plan, in its own work folder. Per arm, in this order: the Library fixed starts M2, M3 and M4 in a row; the new project's C1 once, followed by a verify launch on its result, which shows that both arms run on its inputs and that the verify launch adopts K's result there before the series (its judgment stays pending and unjudged); then the Makerspace chain W0 to C9 and its verify launch. The first arm alternates per task, and K starts the chain. 32 launches, launch bound 48 |
| Passes when | The runner's verdict in `gate.json` holds every check: every launch classed a result; every delivered public result with a valid judgment; every chain with its chain judgment, and every chain and new-project start with a verify Result that ended `verified` or `findings`; no credential in the evidence; nothing left on the engine; and M's Makerspace chain landed its first step, W0. That last check shows that DEC-087 holds in a chain, and that M's later steps do not start from an empty start, as they did in the pilot |
| Fails | The runner stops the gate at its first launch that is not a result, or at a judge that fails twice. **A provider void**, such as capacity or a rate limit, needs no fix: the gate starts again on the same commit, in a new work folder. **Any other failure** is fixed in the arm, the suite, a judge or the runner, and is not counted. Then a new commit is frozen, this page is amended, and the gate starts again from its first launch. If M lands no W0, Build looks first |
| Record | The gate's launches never count. Its verdict and its time per launch are recorded as an amendment here. Build also compares the Makerspace chains' verify Findings with the series judge's drift checks of the same chains, and records how far they agree |

## After a stop

Each stop leaves its reason in `STOP`. Build removes `STOP` and starts the runner again only as follows:

| Stop | Next |
| --- | --- |
| The end of a window, a seven-day use of 90 % (after its reset), or a login that could not be refreshed (after the owner logs in again) | Resume on the same commit |
| A credential in the evidence | The file is removed, and Build checks how it got there. With no code change, resume on the same commit |
| Hidden material on the machine, or a second launch invalid through other work's containers | The machine is cleared, or the owner's other work is stopped again. Resume on the same commit |
| An invalid launch not caused by other work, a void share above one in four, a judge that fails twice, a changed frozen file, or the runner's own failure | A defect of an arm, the suite, a judge or the runner. If it needs a change to the frozen commit, that platform's launches so far stay in the record, but they are reported apart and not counted. A new commit is frozen, the gate passes again, this page is amended, and the platform starts again from its first launch in a new work folder. Without a change, resume on the same commit |

**The launch that stopped the series:**

- **Invalid, on resume:** the runner launches it again once. If that launch is invalid again, its unit stays without a result. The report counts it under Unpaired in the contrast and reads the contrast a second time with it scored as the change not done, so it is never left out silently.
- **Void with relaunches left:** relaunched, as before.
- **Anything else:** stays as it is, and the series goes on with the next unit.

**What can be seen during the series:** results seen during the series cannot change what is measured, because any change to the frozen commit restarts the platform. Build sees the Library's and Makerspace's results window by window, and the new project's only after the last launch.

## Reading

- **Primary contrast:** per platform, M against K on the new project's fixed starts. It is read from the runner's report on that project's fixed starts alone, `report/gridshare-fixed`, without plants or chains.
  - **Endpoint:** the share of the hidden cases judged at the change's step that pass, the earlier steps' cases included.
  - **Pairs:** a change and a repetition, 35 per platform.
  - **The interval:** the 95 % interval of the mean paired difference, read over the 7 changes' mean differences (`blockBy` `change`), since a change's five repetitions are not independent. The interval is wide with only seven changes, and the result may well be undecided.
  - **Reading:** better, worse, "a meaningful advantage is excluded", or undecided. A contrast is at least as good when the interval's lower end lies above minus its minimum meaningful difference.
  - **At least as good, the owner's rule** ([Amendment 2](#amendment-2-30-october-2026-scoring-fixes-before-the-gate)): M is at least as good as K on the new project's fixed starts only when all three hold at once:
    - the share of hidden cases passed is at least as good, at 0.05 (the primary endpoint);
    - the failed hidden cases (`casesFailed`) are at least as good, at 1 case;
    - M's landed-change rate is not below K's.

    Each of the three is read with the units without a result, below. The verdict is yes when all three hold, no when one does not, and undecided otherwise. The table "At least as good, by the owner's rule" gives it, in the report on the fixed starts alone (`report/gridshare-fixed`), since the rule is defined there; plants do not count toward the landed rate.
  - **Units without a result:** a unit where one arm has no result, void after its relaunches or invalid again, is unpaired. Each contrast is then read a second time with every unpaired unit scored as the change not done for the arm without a result; the landed-change rate counts it as a change not landed. A result that has no value for a contrast, such as an unreviewed one for defects left, is not scored as not done; it cannot be compared, so a unit it leaves with one arm keeps that contrast undecided. The verdict is "at least as good" only when both readings say so, "not" only when both say not, and otherwise "undecided (N unpaired)".
  - **A launch that delivered nothing** scores as its start, the change not done: the earlier steps' cases pass and the step's own new cases fail. Both counts come from `correctness.byStep` of a judged launch of the same task and step.
- **Minimum meaningful difference: 0.05** of the cases judged at a step. On the new project that is two to three of 37 to 52 cases. The owner decided it on 30 October 2026, before any counted run, so it is never chosen after seeing results. The pilot's report used it, and the pilot gave no spread to size it from.
  - **Limit:** because the endpoint counts the earlier steps' cases, a change adds only a few cases of its own (W0 34, then 3, 2, 4, 4, 2, 2 and 1). So a change that stops wrongly and delivers its start loses only its own new cases. At C7 that is 1 of 52, less than 0.05.
  - **The second contrast** reads the failed hidden cases per change (`casesFailed`, with a minimum meaningful difference of 1 case), also over changes, and so shows what the share dilutes. A change adds 1 to 4 cases of its own, so one case is a quarter to all of a change's own cases, and one wrong stop at C7 equals it. §15.1 shows the wrong stops directly: each arm's landed changes per task. The owner chose the share as primary, with this count beside it ([decided](#decided-by-the-owner)).
- **Secondary:**
  - the share of each change's own new hidden cases passed (`ownCasesPassedShare`), over the same pairs and changes, descriptive, with no minimum meaningful difference and no verdict: what the primary share dilutes;
  - the same contrasts on the fixed starts of the Library (8 pairs, 4 changes) and the Makerspace (18 pairs, 9 changes), descriptive;
  - the chains' paired steps (`report/NAME-chains`), descriptive;
  - defects left for a person (`defectsLeft`, with a minimum meaningful difference of 1), from the reviewed results of repetition 1;
  - §15.1 to §15.4 per arm and series, with planted tasks under Stops right only;
  - the chains' drift and regressions;
  - cost, and the organization's cost per 2.0 Run, described. The organization's cost comes from the pilot's `manager_costs.py`, frozen by hash in the amendment.
- **No drift, with one independent `verify --full` per chain end:** §15.2 reads No drift from an independent `verify --full` after the series. The series runs one `markitect verify --full` on each chain's end state, for both arms. That makes 8 verify launches per platform: the Library chain, the Makerspace chain and the new project's two chains, per arm.
  - **The end state** is the result of the chain's last step that delivered one, checked against that step's model, as the series judge's final drift is. A chain whose steps delivered nothing has no end state and gets no verify launch.
  - **The launch:** a native launch like M's, right after the chain's last step, with the chain's deadline per step. A public chain is judged just before it; the new project's chains are judged after the last launch. Its arm is `markitect_arm.py --verify`. It commits that step's model as the owner's input where the end state holds none. That is K's case, since K's repository is like M's but carries no `.markitect/`. Files no registry page names get rows without links, as at an adopted start. Then it runs `markitect verify --full` with Claude checkers, and the Result goes into the chain's `verifyFull` slot of the report. A verify that did not check every item shows its state there, not a count of Findings, and fails the gate's check.
  - **Evidence:** since #909 its checkers get the required checks' results.
  - **Classes and stops:** a verify launch is classed, relaunched and stopped like any other, under its own label V. So its voids count toward neither arm. A verify launch that stays void leaves the slot empty, which is no failure of the arm.
  - **The reading, frozen now:**
    - The report gives each chain's raw count of Findings per arm, with the Result's state. A verify that did not check every item shows its state, not a count.
    - No Finding is set aside after seeing results.
    - Unrealized rules and unaccounted files are carried by the series judge's drift checks (forgotten rules, unaccounted files), not by this count.
    - On the new project, the judge session tallies the Findings per arm in its report, since Build is blind there.
  - **Limits:** the counts are descriptive, not a contrast.
    - K's end state is checked after an adoption without file links, M's with its own registry pages. So K's count may include Findings about missing links, and they are not set aside.
    - False Findings: T9's `verify --full` gave 2 false Findings of 3, and #909 has no native evidence yet. The gate's comparison with the judge on the Makerspace gives a first figure.
- **Stated limits:**
  - The reviewers may guess the arm; in the pilot they guessed it every time. Every reviewer is from the writers' model family.
  - Tokens of agents that a Run ends are counted since #907: the exact input and at least the output for Claude, and 0 for a stopped Codex turn. Such a count is marked partial and read as "at least".
  - Human attention is measured by its stand-ins.
  - A unit whose launch is invalid again after its one relaunch counts under Unpaired, not as a failure of its arm. The report shows it, and the second reading scores it as the change not done.
- **What it claims:** only what it measures, as §15 says. It may show at least K's quality and fewer defects left for a person per change at comparable input. It does not show time saved.

## Decided by the owner

Build approves this protocol under the owner's delegation. On 30 October 2026, Build put four points to the owner in short form, and the owner decided them. Each is quoted in the owner's words, with the meaning Build gave it:

1. **The primary endpoint: "Share passed, 0.05".** The share of hidden cases passed per change, with a minimum meaningful difference of 0.05, and the failed-case count read beside it.
2. **The quota: "OK as planned".** The full series, Windows then Linux.
3. **The judge phase: "A fresh session".** A new session that works on no arm starts only after the last launch, clones the private repository, writes the judge phase and runs the judging. Build only reads the reports.
4. **The authoring records: "I export & delete it".** When Build says the gate is next, the owner exports the authoring session off the machine and deletes it here.

**The protocol's additions, which are Build's and this page's and not the owner's words:**
- The quota is spent under the protocol's quota stops.
- The share's dilution on changes that add few cases stays a stated limit under [Reading](#reading).
- A session needed for the new project's material during the series, and the removal of its records, follows [Hidden material](#hidden-material).
- The authoring session's removal is precondition 4.

**Information, no decision:** section D of the suite design said that the owner approves each protocol. This page brings that line in line with the delegation of 29 October 2026, under which Build approves it.

## After the series

1. The owner's blinded acceptance of three changes per arm on the new project, with minutes and corrections (§15).
2. The Codex Luna repeat on Windows ([build page](../workstreams/build.md), Codex runs), as its own amendment of this page. Three things come first: the suite's native mode must copy a Codex login, which today it does only for Claude ([Codex smoke](codex-library-smoke-20261030.md)); K needs its Codex command; and the judge session's records are removed, exported off the machine and deleted by the owner, since they hold the new project's hidden material. Its proposed size is the new project for both arms: two repetitions per change, one per plant and one chain per arm, which is 50 launches.

## Amendments

### Amendment 1, 30 October 2026: the frozen commit

- **The frozen commit:** `main` at `a8a9234c2df1ef97cc83674964445b862c54e473`.
  - It holds the runner ([#908](https://github.com/Glacius-Labs/Markitect/pull/908)), this protocol ([#910](https://github.com/Glacius-Labs/Markitect/pull/910)), the verify launch ([#911](https://github.com/Glacius-Labs/Markitect/pull/911)) and the approval ([#912](https://github.com/Glacius-Labs/Markitect/pull/912)). Its `tools/` is the same as at `6174d5da`, the commit the approval names.
  - The gate and the series run from it on both platforms. The binaries and images are built from it under the lock, and the gate's record names them by image ID.
- **Who handles the new project's material,** in Build's words: "a fresh session that works on no arm, started by Build after the last launch; mid-series, a session whose records the owner exports and deletes before the next launch". The author writes the first launch inputs from this commit before the authoring session is removed (preconditions 3 and 4).
- **The folders,** with absolute paths:
  - **Windows:** under `C:\Users\Consiliari\AppData\Local\markitect-counted\`: `inputs\gridshare` holds the new project's launch inputs, which the gate and the series share. `judges\gridshare` is written only after the last launch. The work folders are `gate-windows` and `windows`, and the runner's spec for the new project is `gridshare.json`.
  - **Linux, in WSL:** under `/home/user/markitect-counted/` with the same names, with `gate-linux` and `linux` as the work folders.
- **The spec for the runner** (`--prepared`):
  - steps W0 to C7, fixed starts C1 to C7, and the plants `gridshare-plant-1`, `-2` and `-3` at C2, C6 and C7;
  - 5400 s per step;
  - `inputs` and `judges` as above.
- **The organization's cost:** the pilot's `manager_costs.py`, SHA-256 prefix `ad8185aa1e67fa56`, in Build's notes folder `t7/`.
- **The authoring session's records:** the session "Author the hidden energy-community evaluation project" (`local_807509a3`, worktree `markitect-v1-concept-review-e8323e`) keeps its transcript and file history under the ID `282a2d17-98a7-460b-a39d-73891191a40b`, as [Hidden material](#hidden-material) lists them.
  - Checked on 30 October 2026: all four places exist, and the transcript's last write matches the session's last activity.
  - The same project folder also holds an older transcript, `f5472f6f-76f1-4d2e-a7b4-192fe59c6edb`. Its session ended on 28 October, before the authoring session began. The marker search covers it.

### Amendment 2, 30 October 2026: scoring fixes before the gate

- **Why:** before the gate, the owner asked for a full independent check of the evaluation. It found two scoring rules that let technical events, not quality, decide the primary contrast, and Build confirmed both against `a8a9234c`. No gate or counted launch had run. The gate was held, with `STOP` in its work folder.
- **The fixes** ([#915](https://github.com/Glacius-Labs/Markitect/pull/915)):
  1. **Units without a result no longer drop out of the verdict.** Before, the contrast paired only units that both arms finished, and "at least as good" ignored the unpaired ones. Now each contrast is read a second time with every unpaired unit scored as the change not done for the arm that has no result. The verdict is yes only when both readings say yes, and no only when both say no. Otherwise it is "undecided (N unpaired)" ([Reading](#reading)).
  2. **A launch that delivered nothing scores as its start,** for both arms and every endpoint. Before, it scored a share of 0 and failed every planned case. Now:
     - the share is the earlier steps' cases over all cases;
     - the failed cases are the step's own new cases;
     - both come from `correctness.byStep` of a judged launch of the same task and step; a judge that does not count by step, such as the Library's single-step M1 judge, has only the step's own cases, so the start fails them all, as before;
     - it still counts once more as a step with no result, under defects left.
  3. **K has a void signature for its own failures:** `reported.status` matching `^error$`, as M has three ([Protocol](#protocol), Technical failures). In the pilot no K launch had that status.
  4. **Chain reports** key a step's case counts by task and step, not by task alone. Before, a chain's step that delivered nothing got the chain's largest count.
  5. **K gets its task without the registry pages,** as Build decided on 29 October 2026.
     - `suite.py run` and `chain --without-pages` remove `registry.md` and `registry/` from `model/` and `base-model/` in K's copy of the task, and `suite.json` lists them under `pagesWithheld`.
     - The task folders are unchanged, so the pages are removed rather than checked for in K's transcripts.
- **Gridshare's launch inputs are kept,** checked on 30 October 2026 by digest only:
  - `inputs.json` still has SHA-256 `42216d1439ac0c0c9de75e9f95c06cd90445eec6a7b81e8877a99d749072af1e`, and all 26 digests it records match the files.
  - `suite.py task` from `a8a9234c` and from this fix both rebuild the tasks of the fixed starts C1 to C7 to the author's digests.
  - `series.py` and `judges.py` are unchanged.
  - Precondition 3 now keeps inputs whose digests are unchanged, and nobody clones the private repository again before the judge phase ([Preconditions](#preconditions)).
- **The pilot's reports,** rebuilt from its evidence with this fix:
  - The notes change, and the contrast table gains the second reading.
  - The share contrast reads "worse" in both readings, so its verdict stays "no".
  - The defects contrast, one pair and one unpaired unit, reads "undecided (1 unpaired)".
  - No pilot launch delivered nothing, so no count changes.
- **The owner's rule for "at least as good"** ([Reading](#reading)):
  - **The owner's answers,** on 30 October 2026: "Yes, require both", to Build, on whether the share and the failed-case count must both hold; and "Yes, hand it to Build", to the Topics session's recommendation, approved after independent reviews at `2c900826`, which replaces "require both".
  - **The rule:** on the new project's fixed starts, M is at least as good only when all three hold at once: the share of hidden cases passed at 0.05, the failed hidden cases at 1 case, and M's landed-change rate not below K's. Each is read with the unpaired units as fix 1 says. Yes when all three hold, no when one does not, undecided otherwise.
  - **Why, in Topics' example:** K is perfect and M fails to land C2, C5 and C6. The share's mean difference is then about −0.019, and its interval's lower end about −0.041. The old rule read "at least as good" with three of seven changes undone. Under the new rule the failed cases and the landed rate both say no. The example is a test of the report (`test_acceptance.OwnerRuleTest`).
  - **A secondary line** reads the share of each change's own new cases passed, descriptive.
  - **How the code reads it:** a contrast says yes or no only when both its readings agree. So a contrast whose two readings both say no reads "no", not "undecided".
  - **The landed rate has no tolerance,** as "not below" says: one change that M lands less often than K, out of 35, makes the rule say no. With an absent result counted as not landed, a launch of M that stays void after its relaunches does so too.
- **The frozen commit:** `main` at `4576beaa087205fb336d22ba56e30361ed469385`, which replaces `a8a9234c`.
  - It holds this amendment's fixes ([#915](https://github.com/Glacius-Labs/Markitect/pull/915)) and T5's two host fixes in `internal/` ([#916](https://github.com/Glacius-Labs/Markitect/pull/916)): `verify` refuses `changed`, and a Manager may not create its own Package's first registry page. Its `tools/` differs from `a8a9234c` only by #915.
  - The binaries and images were built from it under the lock on 30 October 2026. The gate's record names them by image ID:
    - the Manager image `sha256:d9f84f6867eb5b2fb02e6f44826a1e060ca425aa865c8be9ab33999d7395a884`;
    - the Go Manager image, which M's runs use, `sha256:7833b344d0697aaf2f314a8cac16166b2973dce7167da56d8687ae54b163a2ab`;
    - the sandbox image `sha256:2216d76edb2c084ac3c875037b22a18fde4a4954a526fd2f38bfe872e64c52bf`.
  - The gate's work folder `gate-windows` runs from its `tools/`. Build's `STOP` stays there until Build clears a window with the owner.
  - The new project's launch inputs are kept, under precondition 3. Checked against this commit on 30 October 2026, by digest only:
    - `inputs.json` keeps its SHA-256, and all 26 digests it records match the files;
    - this commit's `suite.py task` rebuilds the tasks of C1 to C7 to the author's digests;
    - the Windows and WSL copies are byte-identical, 3,915 files.
  - The organization's cost stays the pilot's `manager_costs.py`, unchanged at SHA-256 prefix `ad8185aa1e67fa56`.
  - Everything else in Amendment 1 stands.
  - No step of the new project's launch inputs holds a Package without a registry page, so T5's second fix does not touch them (checked on 30 October 2026, paths only). Build checked the Library and the Makerspace.

### Amendment 3, 30 October 2026: the Windows gate

- **Verdict: the gate did not pass.** `gate-windows\gate.json` (22:24:08Z) holds every check but the last one.
  - **Held:**
    - All 32 launches were results, with no relaunch.
    - Every delivered public result and both chains were judged.
    - All four verify launches ended `findings`.
    - No credential was found in the 13,459 evidence files scanned.
    - Nothing was left on the engine, and the engine reported no error.
  - **Failed: M did not land the Makerspace W0** (`mLandedTheFirstChainStep`).
    - M's W0 Run (`eval-fc73c7a39292`) ran 1,393 s of its 5,400 s and ended `incomplete` with nothing landed. Its cause: "Manager . used its 6 maker starts without approved work", after 35 agent starts and about 11.5 million input tokens.
    - M's C1 to C9 then started from W0's empty start. C1 to C8 landed, and C9 had nothing to do.
  - **Why, as Build found it, looking first as the [gate's rules](#stability-gate) say:** M's own behavior, not a fault of the arm, the suite, a judge or the runner.
    - Root integrated nine children and spent its maker starts on their events.
    - It told `protocol` that modules were on Root's branch before they were (m6 and m8, against m7 and m10).
    - Its checker rejected the README's example session while `protocol` was not wired.
    - In T8, the same W0 landed once ([record](t8-makerspace-w0-20261030.md)).
  - **What follows, as Build decided:**
    - No relaunch, and no new gate now.
    - The comparison of the long-lived Managers redesign runs first, with W0 twice per arm. The fix and a new freeze follow the owner's decision on it.
    - Precondition 5 does not hold for Windows.
- **The run:**
  - **What ran:**
    - the frozen commit `4576beaa`;
    - the work folder `gate-windows`, from its own `tools/`;
    - the runner's SHA-256 `2f32715b9a58675699b00d0e17fce809166ee453e8c7e95d051ac37e150fe03b`;
    - the binaries, images and inputs that Amendment 2 names, recorded in the ledger's `freeze`.
  - **The Docker window:**
    - The owner answered "Yes, Docker free" to Build.
    - At 19:10Z, Build found no container running, no `dotnet` process and the lock free, and removed `STOP`.
  - **Timeline:**
    - **Start:** the runner started detached at 19:11:16Z and began at 19:11:17Z.
    - **One wait:** the engine's first ten minutes of watching, 19:12 to 19:22Z.
    - **One login refresh:** at 19:22:07Z, from a file 13.6 h past its expiry. It worked.
    - **Launches:** the first started at 19:22:14Z, and the last ended at 22:23:54Z.
  - **Quota:** the seven-day use rose from 0.17 to 0.21, the five-hour use peaked at 0.24, and nothing was refused.
- **Time per launch**, as the suite's wall time:

  | Launches | M | K |
  | --- | --- | --- |
  | Library fixed starts M2 to M4 | 8.0 min on average (6.0 to 9.5) | 1.9 min (0.5 to 3.2) |
  | The new project's C1 | 10.1 min | 1.1 min |
  | Makerspace chain W0 to C9 | 9.3 min on average, 93 min in all (0.2 to 23.2) | 2.9 min on average, 29 min in all (1.3 to 13.7) |
  | Verify launches: the new project's C1, then the Makerspace chain | 2.9 and 2.9 min | 2.7 and 3.4 min |

  - **In all:** 2.9 h of launches in 3 h 13 min. A launch of M took 9.1 min on average (median 7.9), and one of K 2.6 min (median 1.6).
  - **For the series:** at these averages, a platform's series of about 230 launches takes about 22 hours of launches. The [Protocol](#protocol) estimated 50 to 80.
- **Preconditions 3 and 4 held before the first launch,** recorded by paths and outcomes only:
  - **Precondition 4:**
    - The owner exported the authoring session off the machine, and Build deleted it on 30 October 2026.
    - Build verified that its transcript, subagent files, file history, session environment and temporary folder were gone, and that its worktree and branch were removed.
    - The owner removed the exported archive and the Recycle Bin items.
  - **The search under [Hidden material](#hidden-material):**
    - The marker-file search found no marker file. It covered the user profile, `C:\WINDOWS\TEMP\claude` and the WSL home, with positive controls.
    - The runner's hidden check found that none of the ten hidden places exists.
    - Four temporary folders under `AppData\Local\Temp` could not be read. They were last written between 29 September and 6 October, before the authoring session.
  - **Precondition 3:** the inputs written from `a8a9234c` are kept under Amendment 2. Their digests are unchanged against `4576beaa`, and the Windows and WSL copies are byte-identical.
- **Verify against the judge,** for the Makerspace chains, as the gate's record asks:
  - **No drift, by both:**
    - The series judge found no drift at C9 for either chain: no forgotten rule, no stale file, no unaccounted file and no regression along the chain. Both chains passed 126 of 126 hidden cases there.
    - Each `verify --full` found 303 of its 304 items conforming.
  - **One Finding each, outside what the judge checks.** Neither is false, unlike two of T9's three:
    - **K's chain** (`protocol/program`): the program is not deterministic.
      - With several unexpected members, a request's reason text names whichever member a Go map iteration finds first, so it differs between runs.
      - The model says "Given the same request lines, it produces the same response lines".
      - The code confirms it: two loops over maps in `realization/protocol/request.go`.
      - No hidden case sends several unexpected members.
    - **M's chain** (`tests`): the tests spot-check the permission matrix instead of covering it, as `tests.md` asks.
      - The checker changed `registerUser` or `issueCredit` from admin only to staff and admin in the specification, and either change still passed the tests.
      - The judge runs the tests but does not measure what they cover.

### Amendment 4, 31 October 2026: a new freeze for the second gate

- **Why:**
  - The owner kept the current design after the long-lived Managers comparison, in these words to Build: "Then continue the evaluation runs as planned with the current version for now" ([reading](long-lived-managers-comparison-20261030.md#amendment-1-31-october-2026-the-reading)).
  - The Windows gate failed because Root's maker starts went to its children's events ([Amendment 3](#amendment-3-30-october-2026-the-windows-gate)). [#927](https://github.com/Glacius-Labs/Markitect/pull/927) fixes that: `Rounds` now counts only a Manager's own attempts, which are its first maker start and each start after its work was refused. Starts that its children or the host cause stay bounded by `Starts`, the idle rule and the deadlines.
- **The frozen commit:** `main` at `955683deda0eb31f0cfec7d802f746876bf976b7`, which replaces `4576beaa`.
  - Its code differs from `4576beaa` by two PRs: [#921](https://github.com/Glacius-Labs/Markitect/pull/921) in `tools/`, and #927 in `internal/run` and `internal/adapters/cli`.
  - CI passed on #927's head ([run 41165929423](https://github.com/Glacius-Labs/Markitect/actions/runs/41165929423)). That head differs from this commit only in documentation.
  - The binaries and images were built from it under the lock on 31 October 2026, from 04:01 to 04:03Z. The gate's record names them:
    - the Manager image `sha256:a8ebfe2df3e60e3b5d36122fd0a2ac2431fb4ad7b8a98fff447da1251735cfcb`;
    - the Go Manager image, which M's runs use, `sha256:4f664eb8815d1ba5a01e47b4db380e0d9529bf1c744512735b952f35535c05d9`;
    - the sandbox image `sha256:67e2cb76d76c27fd20b183e8d209e6407c49f758c35a454168b5b37cc2e24ce8`;
    - `markitect.exe` with SHA-256 `305ebbe9ec8645ab30b0000bec80a52777dd611087eee9ae7737710717dc478e`, and `markitect-remote` with `bfd052f76ae8ab65e629451fb7050633220dd0d8d5f7ed318159f6529f21c3e8`.
  - The Manager images differ from `4576beaa`'s in the layers that hold the host's code, which #927 changes. The sandbox image's build files did not change, and it has the same 13 layers as `4576beaa`'s. Only its ID differs, as it does with every build.
- **Precondition 3: the new project's launch inputs are kept.** Checked against this commit on 31 October 2026, by digest only, and without cloning the private repository:
  - `inputs.json` keeps its SHA-256 `42216d1439ac0c0c9de75e9f95c06cd90445eec6a7b81e8877a99d749072af1e`, and all 26 digests it records match the files.
  - This commit's `suite.py task` rebuilds the tasks of C1 to C7 to the author's digests, as `4576beaa`'s does.
  - So nobody writes the inputs again, and nobody handles the new project's material before the judge phase.
- **`tools/` since `4576beaa`: on Claude, nothing changes for either arm.** The only change is #921: `classify.py`, `suite.py`, `sandbox.py`, `sandbox/series.json` and the new `codex_cli.py`. Checked against `4576beaa`'s `tools/`:
  - **K's command:** `claude_code.py` is unchanged, and so are `series_runner.py`, `native.py`, `results.py`, `judging.py`, `series.py` and `judges.py`. The runner keeps its SHA-256 `2f32715b…`. The new gate's configuration differs from the first gate's only in the commit, the work folder, the binaries and the images.
  - **M's arm:** `markitect_arm.py` is unchanged. M changes only through #927, in the host.
  - **The Claude login copy:** #921 moved it into its own function in `sandbox.py`. For five synthetic logins, old and new give the same copy, the same secrets to search for and the same refusals. `--login` still defaults to `~/.claude/.credentials.json` for Claude.
  - **The void classes:**
    - `classify.py` reads a session as a Codex exec stream only when it has a `thread.started` or `turn.started` line, which Claude transcripts do not have.
    - Old and new classify all 52 run records of the first gate the same, with its signatures for M and K. They also read the same session and start errors from those runs, even where `classify` does not use them.
    - #921 added Codex tests and changed none of the existing ones.
  - **The egress for Claude:** still `api.anthropic.com` only, and the limits are unchanged. `series.json` only gained a `codex` entry. Its SHA-256, which each sandbox and native run records as `seriesSha256`, changes from `8667891c…` to `c488a4c2…`.
- **The second gate's work folder:** `C:\Users\Consiliari\AppData\Local\markitect-counted\gate-windows-2`.
  - `STOP` was written first. It stays until Build gives the go, after the owner clears a Docker window.
  - Its `src\tools` is extracted from this commit, and `plan` wrote its configuration: 32 launches with a bound of 48, as for the first gate.
  - The runner's hidden check finds none of the hidden places.
  - If this gate passes, the series' configuration names its ledger as `gateLedger`.
- Everything else in Amendments 1 to 3 stands. Precondition 5 holds for Windows only once this gate has passed.

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
| No arm had run on the new project before the counted part; its inputs were not re-prepared after a gate fix | The gate runs the new project's C1 for both arms. A new commit means writing the inputs again (since [Amendment 2](#amendment-2-30-october-2026-scoring-fixes-before-the-gate): inputs whose digests are unchanged are kept). The registry pages of its fixed starts are a precondition |
| The hidden places were not named exactly, and "the WSL home" cannot be one | Every place is listed per platform under [Hidden material](#hidden-material) |
| The `defectsLeft` contrast was not named | Named as secondary |
| The new project's deadline was the author's, and a timeout counts against the slower arm | Fixed at 5400 s at the freeze; the report shows timeouts per arm |
| `verify --full` was left open, and T9's gave false Findings | Decided for this series: not run, a stated limit, and the owner's question (replaced, see below) |
| For the owner: the endpoint and its difference, dropping `verify --full`, the quota, and section D's wording | Listed for the owner, and since decided ([Decided by the owner](#decided-by-the-owner)) |

The same reviewer then checked the revision against this page and the code. It found that every handling above matches the code, and it checked the numbers by running `plan` (then): the gate had 28 launches with a bound of 42, the series 222 per platform with a bound of 333, the repetitions 56, 56, 20, 14 and 14 launches, and M runs first in 18 of the new project's 35 pairs. It would approve once its first two remaining findings were fixed:

| Finding | Handling |
| --- | --- |
| Build could still see the new project before its last launch: delivered and timed-out counts in the reports per window, and its run evidence when diagnosing a stop. Nobody was named to write its inputs again | The runner leaves a series judged later out of every report until it is judged. Build never opens the new project's inputs or run evidence. Someone who works on no arm diagnoses its stops and writes its inputs again |
| An invalid launch the arm caused dropped out silently on resume | It is launched again once. If that launch is invalid again, the unit stays without a result and is counted under Unpaired |
| More hidden places: the authoring session's temporary folder and other files, and the WSL folders of its Linux self-test; a literal `~` was never expanded | The temporary folder is listed. A bounded search for a marker file the author names covers the rest. The runner expands `~`, and the page asks for absolute paths |
| `casesFailed`'s minimum of one case is a large share of a change's own cases | Stated under Reading, with §15.1's landed changes as the direct measure of wrong stops |
| The freeze was checked within one work folder only, and the inputs had no digest | The freeze includes the new project's launch inputs, and the series checks them against the gate's record (`gateLedger`). Each platform's reading names its commit |

**The reviewer then approved the protocol at `44890a66`, with the runner at `dd82a93d`. Nothing blocked it.** It confirmed each of these handlings in the code and the numbers again. It made two last points, which are now on the page: name the person who handles the new project's material in the amendment before the gate, and state as a limit that a unit invalid again after its relaunch counts under Unpaired.

**Changes after the reviewer's approval, at Build's request on 30 October 2026:**

- [PR #907](https://github.com/Glacius-Labs/Markitect/pull/907) counts the tokens of ended agents, so that stated limit now says so.
- [PR #909](https://github.com/Glacius-Labs/Markitect/pull/909) gives `verify` the required checks as evidence. So the series now proposes one `verify --full` per chain end for both arms, 8 launches per platform. This replaces the decision not to run one. [PR #911](https://github.com/Glacius-Labs/Markitect/pull/911) builds the arm's `verify` mode and the runner's verify launch (precondition 6). The gate gains two verify launches on the Makerspace chain and, after the second review below, two on the new project's start: 32 launches, bound 48.
- The owner's questions followed Build's list, and the owner decided them the same day (see [Decided by the owner](#decided-by-the-owner)).
- Who handles the new project's material is worded so that a fresh session that works on no arm qualifies.
- A second independent review of these changes, which Build ran, found eight points. They are handled on this page, in seven handlings, since the recommendation history it named in two places was dropped from both:
  - the owner's section holds only the owner's words and Build's meanings;
  - a session that clones the private repository becomes hidden material, with its removal mid-series and before the Codex repeat;
  - who writes the inputs again after a new commit;
  - the verify reading is frozen as raw counts per arm, with its limits and the false-Finding caveat;
  - the gate verifies the new project's start for both arms (32 launches, bound 48, built in #911);
  - page and runner agree on when the verify launch runs and on an empty slot;
  - the history lines are marked.

  Build checked these handlings against the reviewer's points and approved the protocol without a third review round.

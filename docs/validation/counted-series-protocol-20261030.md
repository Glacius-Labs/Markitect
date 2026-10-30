# Counted series: protocol — 30 October 2026

The protocol of step T7's counted acceptance series ([delivery plan](../implementation-plan.md#concept-20-build)), as section D of the [suite design](../design/evaluation-suite-v2.md#d-counted-evaluations) asks for. It is built from the pilot's [proposal](t7-pilot-20261029.md#proposed-protocol-for-the-counted-series) and its [Q4](t7-pilot-20261029.md#q4-what-must-change-before-a-counted-series), updated for what has changed since. The owner delegated the counted series to Build, and Build approves this protocol after an [independent review](#independent-review), recorded below before any counted run ([build page](../workstreams/build.md), Decisions).

**Status: proposed on 30 October 2026, not yet approved.** No gate or counted launch has run. The [stability gate](#stability-gate) runs only in a window that Build clears with the owner.

## What changed since the pilot's proposal

- **T8** is on `main` ([PR #899](https://github.com/Glacius-Labs/Markitect/pull/899), [DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)): the required checks now refuse only merged states. The Makerspace W0, which no Run landed in the pilot, has since landed natively and passed 54 of 54 cases ([record](t8-makerspace-w0-20261030.md)).
- **T9** is on `main` ([PR #904](https://github.com/Glacius-Labs/Markitect/pull/904), [DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)): the pilot's c3 plant now stops, and the Codex Library M1 has landed ([record](t9-verdict-scope-20261030.md)).
- **The engine** stays Docker Desktop's, as the owner decided on 30 October 2026. Before each window, Build asks the owner to stop all other Docker work at the process level, not only pause an app, and checks that it has stopped. The reruns' guard stays as a second line ([build page](../workstreams/build.md), "Evaluation engine"). The suite does not get an engine of its own.
- **The runner** is now part of the suite, [series_runner.py](../../tools/evaluation/README.md#the-series-runner) ([PR #908](https://github.com/Glacius-Labs/Markitect/pull/908)), not a driver in a notes folder. It builds in Q4's items 2 to 4:
  - the guard as standard;
  - it never holds the engine lock;
  - it passes each launch's arm to the classifier;
  - it refreshes the login before a launch the login cannot cover.

  It also holds this protocol's order, repetitions, relaunch and stop rules and the stability gate. It is frozen with `tools/`.
- **The minimum meaningful difference** is frozen at 0.05 before the first counted run, unless the owner sets another value by then.

## Preconditions

All of these hold before the first gate launch. The gate's frozen commit is the series' frozen commit.

1. PR #908 (the runner) is on `main`.
2. **Plants:** every plant has been checked for consequences of what it removes (Q4, item 5).
   - The Makerspace plant `makerspace-c5-threshold-gap` still leaks. The sentence "An invoice whose `dueAt` is `2099-12-01T23:59Z` or later can never block" is still in the C5 model (`computed-times.md`) and in the C5 requirements. Its plant record must also remove that sentence, as an amendment of the [plants](../../tools/evaluation-judges/plants/README.md) with a new recorded time. Otherwise c5 is left out of the series and of this page's counts.
   - The new project's author reported on 30 October 2026 that each of its three plants has been checked. The gap defect leaves no other sentence, example or number that pins what it removes, in `task.md`, the model or its start's model. The contradiction keeps its opposing statements on purpose. The landing gap stays determined by the remaining rules.
3. **The new project's launch inputs are written** before its hidden material leaves the machine. The tool is its author's `suite_inputs.py inputs --suite PUBLIC_REPO --phase launch`, run with the frozen commit as `PUBLIC_REPO` and 5400 s per step, into `inputs/gridshare` in each platform's work folder. It writes only what variants get: the steps, the reference starts, the base models with registry pages, and the tasks from the frozen `suite.py task`. Its `inputs.json` lists sizes and digests and no plant kinds.
   - A spec file names the series for the runner: steps W0 to C7; fixed starts C1 to C7; the plants `gridshare-plant-1` to `-3` with their steps; `inputs`; and `judges`.
   - On 30 October 2026, at public `main` `c575faa5` and private `c87aa81`, the author reported that every step and adopted start passes `markitect check` with no `lint.no_paths` warning, and that the launch and judge phases match the combined output digest for digest.
4. **Hidden material off the machine:** after the launch inputs are written, none of these exists on the machine:
   - the private clone at `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation-hidden`;
   - the author's run evidence at `C:\Users\Consiliari\Documents\Codex\gridshare-runs`, which holds judge inputs;
   - any judge-phase or combined output folder;
   - the pilot's three possible clone paths;
   - the authoring session's own records, which hold the whole material: its transcript `282a2d17-98a7-460b-a39d-73891191a40b.jsonl` with the folder of the same name, under `C:\Users\Consiliari\.claude\projects\C--Users-Consiliari-Glacius-Labs-Markitect--claude-worktrees-markitect-v1-concept-review-e8323e\`, and `C:\Users\Consiliari\.claude\file-history\282a2d17-98a7-460b-a39d-73891191a40b`. **Only the owner can remove them**, by exporting the authoring session and then deleting it from the machine. Build asks the owner before the first counted window.

   The runner's configuration names all of them under `hidden`. On Linux it adds the same places under `/mnt/c/`, which the WSL host can read, and the WSL home's own places.
5. **§15.2's independent `verify --full`.** The suite's report has a slot for the 2.0 arm's own `verify --full` after a chain, but neither the arm nor the runner runs one yet. Build decides before the freeze. Recommended: the runner runs `markitect verify --full` once after each M chain, on the chain's last landing, as one more launch per chain, and files its Result under `verifyFull`. Otherwise, No drift rests on the series judge's drift checks, and the report names the missing `verify` as a limit.
6. **The stability gate** has passed on the frozen commit for the platform to be run.
7. **Engine:** for each window, the owner has stopped all other Docker work at the process level, and Build has checked that `docker ps` shows no container without a `markitect.*` label and that no Testcontainers or `dotnet` process is running.

## Protocol

| Item | Value |
| --- | --- |
| Tasks | The Library, the Makerspace and the new project, as the build page plans and §15 names. Every fixed start and plant runs as an independent task from its reference start. M gets the registry pages, as in the pilot; K gets the same start without them. Every chain runs whole. The new project (`gridshare`) is the confirmatory part, unseen by anyone who works on an arm. It has a fresh build W0 and seven changes C1 to C7. Its three plants, two defects and one gap, have names that do not say what they plant. Its cumulative hidden cases per step are W0 34, C1 37, C2 39, C3 43, C4 47, C5 49, C6 51 and C7 52. The Library and the Makerspace give §15's tables for both arms. M was tuned on the Makerspace, so its results there are descriptive ([suite design](../design/evaluation-suite-v2.md#c-portfolio), section C) |
| Arms | **M:** `markitect_arm.py` with `suite.py --native`, effort `high`, the host's limits per container. **K:** `claude_code.py --effort high` with `suite.py --sandbox` and the series' safety limit. P and G are not counted |
| Model | `claude:claude-sonnet-5-5` for both arms. Account login only, never an API key |
| Frozen revisions | One `main` commit that meets the preconditions, named in an amendment before the first gate launch. `markitect.exe`, `markitect-remote`, the Manager and Go Manager images and the sandbox image are built from it under the lock, and named by image ID. `tools/` is extracted from it by `git archive` into the work folder's `src/`. The runner runs from there and logs the SHA-256 of every file; `plan` writes the configuration from the same tree. A change to any of them stops the series. A new commit means a new stability gate, and this page is amended first |
| Repetitions | Per arm (the runner's `counted` plan): the new project's changes five times each, its plants three times, its chain twice. The Library's and Makerspace's fixed starts and plants twice each, their chains once |
| Run order | By repetition, from the first to the fifth. Each repetition runs every fixed start and plant whose count reaches it. The two arms of a task run one after the other; the first arm alternates per task and per repetition, and the task order is rotated per repetition. Repetitions 1 and 2 have 56 launches each, repetition 3 has 20, repetitions 4 and 5 have 14 each. Then the chains run, by repetition, with the first arm alternating per chain: the Library, the Makerspace, the new project, then the new project again. That makes 62 chain launches |
| Platforms | Windows first, natively, then Linux natively in WSL Ubuntu against the same engine, as §15 asks. Each platform has its own gate, work folder, ledger and report, and its contrast is read separately |
| Deadlines | 2700 s per step for the Library and 5400 s for the Makerspace, as in the pilot, and 5400 s for the new project, as its author set it. The new project's tasks are 182 to 241 KB each, its plants 189 to 240 KB |
| Budget | Per arm and platform: the Library 17 launches (4 fixed starts and 2 plants twice each, plus a 5-step chain), the Makerspace 34 (9 fixed starts and 3 plants twice each, plus a 10-step chain), the new project 60 (7 changes five times, 3 plants three times, and an 8-step chain twice). That is 222 launches per platform and 444 in all. The launch bound is 333 per platform, one and a half times the planned launches; relaunches count against it |
| Quota and time | In the pilot, 57 launches used 4 points of the seven-day quota. At that rate the series needs about 31 points, and two to three times that if M lands its Makerspace and new-project steps, as it now can. A seven-day use of 90 % ends a window, and the next one starts after the reset. So the series may span two or three seven-day windows, and the owner's own use shares that quota. By the pilot's times and T8's W0 (24 minutes for M), one platform takes about 50 to 80 hours of launches |
| Windows | Build clears each window with the owner. `WORK/STOP` ends a window before the next launch, including within a chain, between its steps. When the runner is started again, it continues where it stopped and does not launch again a launch that counted. The ledger records each window. Build stops the runner only through `STOP`, never by ending its process |
| Before each launch | The runner checks, in order: no `STOP` file, the launch bound, and no hidden material on the machine. Then it waits for a quiet engine and for a login that outlives the deadline by 20 minutes, refreshing it first if needed, and checks the engine again. **Quiet** means no container without a `markitect.*` label runs, and none started in the last ten minutes, as a `docker events` stream kept open for the whole window shows. **The refresh** sets the access token's expiry to now, changes no token, and lets `claude -p` refresh it. It runs only while no one holds the engine lock |
| Lock | `NATIVE-RUN.lock` in Build's notes folder. `suite.py` takes it for each run, judgment and review. The runner never takes it; a busy lock is waited for and not counted |
| Technical failures | Classified by `classify.py` with the arm's signatures, written before any launch (`signatures.json`). For M: `reported.status` is `could-not-run` or `check-failed`, or `reported.noAgentTurn` is true. Provider and infrastructure voids are as `classify.py` defines them. A void launch is relaunched at most twice |
| Invalid launches | A launch invalid only through containers of other work is relaunched once. Those are containers without a `markitect.*` label whose start the run's own engine events show. Any other invalid launch stops the series for a look. Both stay in the ledger as invalid |
| Stop rules | The series stops on any of these: a refused request or a seven-day use of 90 %; a login that cannot be refreshed; a credential of the login in a run's evidence; a gateway refusal of a provider name; hidden material on the machine; an invalid launch as above; a void share above one in four for either arm after its first eight launches (that arm goes back to stabilization, and the report says so); the launch bound; the runner's own failure. A five-hour use of 85 % waits for that window's reset. Every stop writes its reason to `STOP`, and Build decides how to go on |
| Hidden material | Nothing of the new project's hidden material is on the machine during any launch on either platform: none of the places under preconditions 3 and 4, and not its judge folders (`judges`). Before each launch the runner checks every place in its configuration, and M's runs check them again (`--hidden-absent`). The new project is judged and reviewed only after the last launch of the series on both platforms. A review copies the rubric into the work folder, and a judgment holds the IDs of failed cases, so neither may exist while launches still run. Then Build clones the private repository, writes the judge phase with `suite_inputs.py --phase judge` into each platform's `judges` folder, and runs `series_runner.py judge` for each platform. The Library and Makerspace judges are public, so their launches are judged as they end |
| Judging and review | Each delivered result is judged at its step, and each chain's counted steps as one chain (`judge-chain`). The code review covers every judged result of a fixed start in repetition 1 and the last judged step of every chain, blinded, in the sandbox: per platform, 40 fixed-start reviews and up to 8 chain reviews. On the new project a judgment takes about 6 to 7 seconds |
| Cleanup | The suite removes its gateway and networks and the arm's leftovers, and counts them. The host removes each Run's remote, containers and volumes. After each window the runner counts what is left labelled `markitect.run` or `markitect.evaluation` and scans every evidence file for the login's tokens and key shapes |
| Evidence | Outside the repository: on Windows `C:\Users\Consiliari\AppData\Local\markitect-counted\windows` and `…\gate-windows`; on Linux `~/markitect-counted/linux` and `~/markitect-counted/gate-linux` in WSL. Each holds runs, judgments, reviews, the ledger with its windows and waits, the log, the reports and the final scans |

## Stability gate

Before each platform's series, on the frozen commit, uncounted ([suite design](../design/evaluation-suite-v2.md#d-counted-evaluations), section D):

| Item | Value |
| --- | --- |
| Units | The runner's `gate` plan: per arm, the Library fixed starts M2, M3 and M4 in a row, then the Makerspace chain W0 to C9. The first arm alternates per task; K starts the chain. 26 launches; launch bound 39 |
| Passes when | Every launch is classed a result, with no void and no invalid launch. Every delivered result has a valid judgment, and each chain a chain judgment. The final check finds no credential and nothing left labelled. M's Makerspace chain lands at least W0, which shows that DEC-087 holds in a chain and that M's later steps do not start from an empty start, as they did in the pilot |
| Fails | On the first void launch the runner stops the gate. The cause is fixed in the arm, the suite or the runner, not counted. Then a new commit is frozen, this page is amended, and the gate runs again from its first launch. An invalid launch, an invalid judgment or a chain in which M lands nothing also stops it, and Build looks first |
| Record | The gate's launches never count. Its results are recorded as an amendment here, with the time per launch, which gives a better estimate of the series' duration |

## Reading

- **Primary contrast:** per platform, M against K on the new project. It is read from the runner's report on that project's fixed starts (`report/NAME-fixed`), which leaves out the chains. The endpoint is the share of hidden cases passed by each change from its fixed start. Pairs are change and repetition: 35 per platform. They are read by the 95 % interval of the paired difference, blocked by repetition and change ([reading.py](../../tools/evaluation/reading.py)). Each contrast reads as better, worse, "a meaningful advantage is excluded" or undecided. M is at least as good when the interval's lower end lies above minus the minimum meaningful difference.
- **Minimum meaningful difference: 0.05**, about one hidden case in twenty per change. On the new project a change has 37 to 52 cumulative hidden cases, so that is two to three cases. It is the value the pilot's report used; the pilot gave no spread to size it from. It is frozen at 0.05 before the first counted run unless the owner sets another value by then, so it is never chosen after seeing results. The runner writes it into the ledger's contrasts.
- **Secondary:**
  - the same contrast on the fixed starts of the Library (8 pairs) and the Makerspace (18 pairs), and on each chain's paired steps (`report/NAME`), descriptive;
  - §15.1 to §15.4 per arm and series, with planted tasks under Stops right only;
  - the chains' drift and regressions;
  - defects left for a person, from the reviewed results of repetition 1;
  - cost and the organization's cost per 2.0 Run, described. The organization's cost comes from the pilot's `manager_costs.py`, frozen by hash in the amendment.
- **Stated limits:**
  - The reviewers may guess the arm; in the pilot they guessed it every time.
  - Every reviewer is from the writers' model family.
  - Tokens of agents that a Run ends are counted only if that fix is on `main` at the freeze ([build page](../workstreams/build.md), Next).
  - Human attention is measured by its stand-ins.
- **What it claims:** only what it measures, as §15 says. It may show at least K's quality and fewer defects left for a person per change at comparable input. It does not show time saved.

## After the series

1. The owner's blinded acceptance of three changes per arm on the new project, with minutes and corrections (§15).
2. The Codex Luna repeat on Windows ([build page](../workstreams/build.md), Codex runs), as its own amendment of this page. It needs two things first: the suite's native mode must copy a Codex login (today it copies only Claude's, [Codex smoke](codex-library-smoke-20261030.md)), and K needs its Codex command. Its proposed size is the new project for both arms: two repetitions per change, one per plant and one chain per arm, which is 50 launches.

## Independent review

Pending.

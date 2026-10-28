# Evaluation suite E2: judge smoke

**Status: proposed on 28 October 2026, awaiting the owner's approval.** Nothing below has run against a container engine or a provider yet.

This smoke checks increment E2 of [evaluation suite v2](../design/evaluation-suite-v2.md#increments) on this machine (Windows, Docker Desktop's Linux engine): (a) the Library judge passes its reference result and fails its seeded mutants in its container; (b) the [E1 smoke](eval-e1-sandbox-smoke-20261028.md)'s runs k-02 and k-03 are judged, (c) reviewed and (e) reported per area; and (d) one K run in the sandbox confirms the capture fix that followed k-01. It is a smoke, not a counted evaluation: no comparison is drawn from it. k-01 is not judged: K's capture defect left its result incomplete.

## What runs

| Item | Value |
|---|---|
| Source | `tools/evaluation` at the reviewed head of the E2 pull request; the operator records its full SHA here as `REV` before the first step |
| Judge | `tools/evaluation/judges/library-m1`, exported once from `REV` with `judges.py export --rev REV`; its `judge.json` and folder digest are recorded |
| Judge image | `golang@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195` (`golang:1.27.1-bookworm`), already on this engine as the sandbox's build stage; nothing is pulled |
| Sandbox image | `sha256:e0b20d8a8c88a0a4b37f151af45618e764f8687b023b10bcb6d3b5148c5e5dad`, the E1 image. `tools/evaluation/sandbox/` is unchanged since it was built (`git diff 4f7cf4d5 REV -- tools/evaluation/sandbox` is empty) |
| Series configuration | [`tools/evaluation/sandbox/series.json`](../../tools/evaluation/sandbox/series.json), as in E1 |
| Model | `claude:claude-sonnet-5-5` for K and for every reviewer ([owner, 28 October](../workstreams/build.md#decisions)) |
| Login | The owner's Claude subscription login, copied per session without its refresh token and mounted read-only; never an API key. The operator checks before each provider step that it outlives the step's deadline plus 300 s |
| Task | `EVAL\library-m1-task`, the E1 task, unchanged (`intentSha256` `35d93641…2e5156`) |
| Runs judged | `EVAL\e1-sandbox\k-02` and `k-03`, as recorded by E1, and the new `k-04` |
| Engine lock | `C:\Users\Consiliari\AppData\Roaming\Claude\side-session-notes\local_cbc94143-9a8e-46c4-9c73-b1b58817893e\NATIVE-RUN.lock`, taken by every step that starts a container (`--lock`); a step refuses rather than waits when another holder has it |

Set `TMP` and `TEMP` to `C:\Users\Consiliari\AppData\Local\Temp` first; Docker Desktop shares that folder. `EVAL` is `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation`, `OUT` is `EVAL\e2-judge`, `SANDBOX` is the sandbox image ID above and `LOCK` the lock file. Every command runs from the repository root at `REV`, through PowerShell.

**(a) Judge self-test:** five judge containers (the reference and four mutants).

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation/judges/library-m1 --rev REV --out OUT\judge-library-m1
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation/judges/library-m1 --judge OUT\judge-library-m1 --task EVAL\library-m1-task --out OUT\selftest --lock LOCK
```

It passes when `selftest.json` says `"passed": true`: the reference passes all 19 cases, its build, its tests and the scope; `broken-rule` fails a case, `failing-build` fails its build, `unrequested-change` shows `.github/workflows/ci.yaml` as unrequested and `protected-input` shows `TASK.md` as a protected input touched. Then, and only then, (b) starts.

**(b) Judging,** one judge container per run:

```text
python -I -B tools/evaluation/suite.py judge --run EVAL\e1-sandbox\k-NN --judge OUT\judge-library-m1 --out OUT\k-NN\judgment --lock LOCK
```

**(c) Review,** one reviewer session per run, plus one confirming session per critical finding that passes the citation check, each in the sandbox:

```text
python -I -B tools/evaluation/suite.py review --run EVAL\e1-sandbox\k-NN --judge OUT\judge-library-m1 --out OUT\k-NN\review --sandbox SANDBOX --variant-code tools/evaluation --lock LOCK
```

**(d) Confirming K run,** one run, in a new run directory, then (b) and (c) for it:

```text
python -I -B tools/evaluation/suite.py run --task EVAL\library-m1-task --model claude:claude-sonnet-5-5 --out OUT\k-04 --sandbox SANDBOX --variant-code tools/evaluation --lock LOCK -- python3 -I -B /variant/claude_code.py --effort high
```

It confirms the fix when it is a result with a delivered commit, a model match and no quarantine, and every Go source, module file and Markdown file the session reports writing is in the delivered commit.

**(e) Report,** offline, from a ledger `OUT\ledger.json` that lists k-02, k-03, k-04 and any voided launch:

```text
python -I -B tools/evaluation/suite.py report --ledger OUT\ledger.json --out OUT\report
```

## Limits

- **Time:** judge containers have the judge's 2,400 s limit (expected one to two minutes each); each reviewer session 1,200 s; the K run 2,700 s, as in E1.
- **Machine:** the judge's limits (4 CPUs, 4 GiB, 1,024 processes, 2 GiB `/tmp`), the series' safety limit for reviewer sessions and K. One container step at a time.
- **Provider:** at most 2 K launches and at most 16 reviewer sessions in all, read-only, on the subscription. A review that would need more confirming sessions than that stops the series, and the cause is reported.
- **Attempts:** a judgment or review that is invalid because of a suite or judge defect may be repeated after an offline fix, at most twice per item, and every attempt is recorded with its cause. A fix changes `REV`; the judge is exported again and the self-test repeated. The K run is relaunched only when the classifier (`classify.py`) finds it void.
- **What fails the smoke:** a self-test that does not pass after its repeats; a judgment or review of k-02, k-03 or k-04 that stays invalid; a K run that is a result without a delivered commit, a model match, or with a quarantine or a dropped source. The series then stops and the cause is reported.

## Abort and cleanup

- **Abort:** stop the step's process tree (`taskkill /PID <pid> /T /F`). An aborted step is recorded with its cause.
- **Cleanup after an abort:** the step's own cleanup is skipped. `docker ps -a --filter label=markitect.evaluation`, `docker ps -a --filter label=markitect.evaluation.judge` and `docker network ls --filter label=markitect.evaluation` list what is left; `docker rm -f NAME` and `docker network rm NAME` remove it. Delete the step's leftover temporary directories, including any login copy `credentials.json`. Release the lock only if this step holds it.
- **After the series:** no container or network with either label remains, no login copy is left in the temporary folder, the lock is released, and no `quarantine` entry appears in any record.

## Evidence

Outside the repository, under `EVAL\e2-judge\`: the exported judge folder, `selftest\` with each case's result bundle and `judgment.json`, per run its `judgment\` (`judgment.json`, `judge.json`) and `review\` (`review.json`, the blinded material and each session's record, stream and transcripts), `k-04\` as `suite.py run` leaves it, the ledger and `report\`. A results section is added to this page after the series, listing every step with its raw outcome, failed attempts included.

## What a pass means

The Library judge separates its reference from its mutants in its container; the E1 results and a new K result are judged, reviewed and reported per area; and K's capture keeps every source. That meets the E2 smoke criterion. It says nothing about which variant is better: E2 has one variant and no counted series.

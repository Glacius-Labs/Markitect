# Evaluation suite E1: sandbox smoke

**Status: proposed, not run.** It needs the owner's approval as a bounded dispatch under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). It replaces the [host smoke](eval-k-library-smoke-20261028.md), which was never run.

This smoke checks increment E1 of [evaluation suite v2](../design/evaluation-suite-v2.md#increments) on this machine (Windows, Docker Desktop's Linux engine): (a) the escape probe finds the sandbox closed, and (b) the baseline K delivers in three Library runs in a row inside the sandbox. It is a smoke, not a counted evaluation: no comparison is drawn from it.

## What runs

| Item | Value |
|---|---|
| Source | `tools/evaluation` at the `main` commit that carries this page |
| Image | Built from that commit with `docker build -t markitect-eval-sandbox tools/evaluation/sandbox`; the operator records its image ID (`docker image inspect --format "{{.Id}}"`) here before the first run and passes it as `IMAGE_ID`. A pre-merge build of the reviewed PR source had `sha256:e0b20d8a…c5e5dad`; image IDs differ between builds |
| Series configuration | [`tools/evaluation/sandbox/series.json`](../../tools/evaluation/sandbox/series.json): 8 CPUs, 16 GiB memory without swap, 4,096 processes, 8 GiB `/tmp`; egress for Claude only to `api.anthropic.com` |
| Model | `claude:claude-sonnet-5-5` ([owner, 28 October](../workstreams/build.md#decisions)) |
| Login | The owner's Claude subscription login (`%USERPROFILE%\.claude\.credentials.json`), copied per run without its refresh token and mounted read-only; never an API key. The suite refuses a run unless the login outlives the deadline plus 300 s (3,000 s); the operator checks the expiry before each run |
| Claude Code | 2.1.308 in the image, the runtime image's pin |
| Task | `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation\library-m1-task`: Library M1, a fresh build; `task.md` SHA-256 `1b3dc865…6179e1b`, start commit `1ffcbca5` (one empty commit), `intentSha256` `35d93641…2e5156`, deadline 2,700 s |
| Engine lock | `C:\Users\Consiliari\AppData\Roaming\Claude\side-session-notes\local_cbc94143-9a8e-46c4-9c73-b1b58817893e\NATIVE-RUN.lock`, taken per run with `--lock`; a run refuses rather than waits when another holder has it |

Set `TMP` and `TEMP` to `C:\Users\Consiliari\AppData\Local\Temp` first: the task copy and the login copy live there, and Docker Desktop shares that folder. `EVAL` stands for `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation`.

**(a) Escape probe,** one run:

```text
python -I -B tools/evaluation/suite.py run --task EVAL\library-m1-task --model claude:claude-sonnet-5-5 --out EVAL\e1-sandbox\probe-01 --sandbox IMAGE_ID --variant-code tools/evaluation --lock LOCK -- python3 -I -B /variant/escape_probe.py
```

The probe makes no provider call: it opens one TLS handshake with the allowed endpoint and sends no request. It passes when `reported.status` is `contained` and `reported.egressWorks` is `true`, with `validForCounting` `true`, no residue and no quarantine. Then, and only then, (b) starts.

**(b) Baseline K,** three runs, one after another, each in a new run directory `k-01`, `k-02`, `k-03`:

```text
python -I -B tools/evaluation/suite.py run --task EVAL\library-m1-task --model claude:claude-sonnet-5-5 --out EVAL\e1-sandbox\k-NN --sandbox IMAGE_ID --variant-code tools/evaluation --lock LOCK -- python3 -I -B /variant/claude_code.py --effort high
```

## Limits

- **Time:** 2,700 s per run, enforced by the suite; K stops its session 60 s before. The probe takes about two minutes.
- **Machine:** the series' safety limit per run (above), plus 1 CPU and 256 MiB for the egress gateway. One run at a time.
- **Attempts:** the probe runs once; if its sandbox does not stand up (for example, the engine refuses the internal network's isolated gateway mode), the smoke stops and the cause is reported. K runs until three runs in a row are each a result, at most six launches in all (about 4.5 hours). A run is **void** and relaunched only when it is a technical failure by these signatures, written before the series:
  - the sandbox did not stand up, its container never started, its gateway stopped, or it left residue (`validForCounting` is `false`, with `sandbox.error` or `sandbox.residue` in `suite.json`), or the suite or engine failed;
  - the provider failed: the session's final result is an API error for capacity, rate limit or a server error, or the session never started because the login was refused;
  - the gateway refused a server name that Claude Code needs (`sandbox.egress.refused` in `suite.json`). This voids the run and **stops the series**: adding an endpoint to the series configuration is a reviewed change and a new dispatch.

  Everything else is a result, including a wrong or incomplete result, a timeout and a crash of the session.
- **What fails the smoke:** a probe that finds any escape, or a K run that is a result without a delivered commit, a model match or with a quarantine. The series then stops and the cause is reported; it is not retried under this dispatch.

## Abort and cleanup

- **Abort:** stop the suite's process tree (`taskkill /PID <pid> /T /F`). An aborted run is recorded with its cause.
- **Cleanup after an abort:** the suite's own cleanup is skipped. Remove the run's containers and networks: `docker ps -a --filter label=markitect.evaluation` and `docker network ls --filter label=markitect.evaluation` list them; `docker rm -f NAME` and `docker network rm NAME` remove them. Delete the leftover temporary directories of the run, including the login copy `credentials.json`. Release the lock only if this run holds it.
- **After the series:** no container or network with the label `markitect.evaluation` remains, no login copy is left in the temporary folder, the lock is released, and no `quarantine` entry appears in any `suite.json`.

## Evidence

Outside the repository, under `EVAL\e1-sandbox\`: per run its `suite.json` (with the sandbox record, every engine call, the safety limit and the egress counts), `egress.jsonl`, the variant's `run.json`, stream, transcripts and `result.bundle`. A results section is added to this page after the series, listing every launch with its raw outcome and class, voided ones included.

## What a pass means

The probe finds the sandbox closed, and K delivers three times in a row inside it with a result, a model match and no quarantine. That meets the E1 smoke criterion. It says nothing about quality; judging comes with E2.

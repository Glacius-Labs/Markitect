# Evaluation suite: first smoke, baseline K on the Library

**Status: proposed, waiting for the owner's approval.** Nothing in this dispatch has run.

This is the first real use of the [evaluation suite](../guides/evaluation-suite.md). It checks step 2 of the [plan](../workstreams.md#priority) for the baseline: one Library task, three runs in a row through the suite without manual help. It is a smoke, not a counted evaluation: no comparison is drawn from it, and it has no decision rule.

## What runs

| Item | Value |
|---|---|
| Source | `tools/evaluation` and `tools/comparison` at the `main` commit that carries this page; `tools/evaluation` is as merged in `cf26bf98` |
| Variant | K, the baseline: `python -I -B tools/evaluation/claude_code.py --effort high`, one headless Claude Code session per run ([README](../../tools/evaluation/README.md#the-baseline)) |
| Model | `claude:claude-sonnet-5-5` for every arm ([owner, 28 October](../workstreams/build.md#decisions)) |
| Login | The owner's Claude subscription login, copied per run without its refresh token; never an API key. The variant refuses to start unless the login outlives the deadline plus 300 s |
| Claude Code | The host's `claude`, 2.1.309 when this page was written; each run records the version it used |
| Task | Library M1, a fresh build, built by `suite.py task --task-md tools/comparison/packet/requirements-m1.md --model testdata/library/m1 --deadline-seconds 2700` |
| Task digests | `task.md` SHA-256 `1b3dc865…179e1b` (the Phase A M1 document, still identical to the current model's rendering); start commit `1ffcbca5` (one empty commit); `intentSha256` `35d93641…2e5156` |
| Runs | Three, one after another, each `suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR -- VARIANT`, each a fresh session |

## Limits

- **Time:** 2,700 s per run, enforced by the suite; the variant stops its session 60 s before. The series ends after at most about 2.5 hours.
- **Attempts:** exactly three runs. A run without a result or with a timeout is recorded as a failure, not repeated. A suite or host failure before the session starts may be retried once and is recorded.
- **Engine:** the runs need no container engine. Only the assessment below uses it, after the three runs, under the shared engine lock, and only when no other run holds the engine.
- **Isolation:** the sessions run as the user on the host. This is separation, not a sandbox: a session could read the hidden oracle in this repository. That is acceptable for a smoke and must be fixed before counted runs ([limits](../guides/evaluation-suite.md#limits)).

## Assessment

Descriptive only. For each delivered result, [`assess_candidate.py`](../../tools/comparison/assess_candidate.py) with profile `m1` runs the hidden Library oracle and probes, the result's own build and tests, and the census of changed paths, against the task's start. No blinded review.

## Abort and cleanup

- **Abort:** stop the suite's process tree (`taskkill /PID <pid> /T /F` on Windows). An aborted run is recorded with its cause.
- **Cleanup:** the variant deletes its login copy when the session ends, and its scratch clone. After the series, check that no `claude` process from a run remains, that no login copy is left, and that the suite's secret scan found nothing in any delivered commit or output.

## Evidence

Outside the repository, under `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation\`: the task folder `library-m1-task`, and per run its `suite.json`, the variant's `run.json`, stream, transcripts and `result.bundle`, plus each assessment. A results section is added to this page after the series, listing every run with its raw outcome, including failures.

## What a pass means

All three runs deliver a result within the deadline, with no manual help and no secret hit. That shows the suite and the baseline work end to end on Claude. It says nothing yet about quality relative to other variants.

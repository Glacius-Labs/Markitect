# Evaluation suite: first smoke, baseline K on the Library

**Status: proposed, waiting for the owner's approval.** Nothing in this dispatch has run.

This is the first real use of the [evaluation suite](../guides/evaluation-suite.md). It checks step 2 of the [plan](../workstreams.md#priority) for the baseline: one Library task, three runs in a row through the suite without manual help. It is a smoke, not a counted evaluation: no comparison is drawn from it, and it has no decision rule.

## What runs

| Item | Value |
|---|---|
| Source | `tools/evaluation` and `tools/comparison` at the `main` commit that carries this page; `tools/evaluation` is as merged in `cf26bf98` |
| Variant | K, the baseline: `python -I -B tools/evaluation/claude_code.py --effort high`, one headless Claude Code session per run ([README](../../tools/evaluation/README.md#the-baseline)) |
| Model | `claude:claude-sonnet-5-5` for every arm ([owner, 28 October](../workstreams/build.md#decisions)) |
| Login | The owner's Claude subscription login, copied per run without its refresh token; never an API key. Each run refuses to start unless the login outlives its session time (the deadline minus 60 s) plus 300 s, that is 2,940 s; the operator checks the expiry before each run |
| Claude Code | The host's `claude`, 2.1.309 when this page was written; each run records the version it used |
| Task | Library M1, a fresh build, built by `suite.py task --task-md tools/comparison/packet/requirements-m1.md --model testdata/library/m1 --deadline-seconds 2700` |
| Task digests | `task.md` SHA-256 `1b3dc865…179e1b` (the Phase A M1 document, still identical to the current model's rendering); start commit `1ffcbca5` (one empty commit); `intentSha256` `35d93641…2e5156` |
| Runs | Three, one after another, each `suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR -- VARIANT`, each a fresh session |

## Limits

- **Time:** 2,700 s per run, enforced by the suite; the variant stops its session 60 s before.- **Attempts:** exactly three counted runs. A run whose session started and that ends without a result, with a timeout or with a crash is a failure, recorded and not repeated. Only a run that never started a session (the variant refused the login or found no `claude`, or the suite or host failed first) may be relaunched, once per run, in a new run directory; the refused attempt stays recorded. At most six launches in all, so the series ends after at most about 4.5 hours, usually about 2.5.
- **Engine:** the runs need no container engine. Only the assessment below uses it, after the three runs, holding the shared engine lock `C:\Users\Consiliari\AppData\Roaming\Claude\side-session-notes\local_cbc94143-9a8e-46c4-9c73-b1b58817893e\NATIVE-RUN.lock` through `--lock`; it refuses rather than waits when another holder has it.
- **Isolation:** the sessions run as the user on the host. This is separation, not a sandbox: a session could read the hidden oracle in this repository. That is acceptable for a smoke and must be fixed before counted runs ([limits](../guides/evaluation-suite.md#limits)).

## Assessment

Descriptive only. For each delivered result, [`assess_candidate.py`](../../tools/comparison/assess_candidate.py) runs the hidden Library oracle and probes, the result's own build and tests, and the census of changed paths, against the task's start. No blinded review.

```text
python -I -B tools/comparison/assess_candidate.py --candidate RUN_DIR/variant/result.bundle --commit RESULT_COMMIT \
    --start TASK_DIR/start.bundle --start-commit 1ffcbca58aba8ef60ba1770e76c5acaea26ff7a5 --profile m1 \
    --run-id k-library-RUN --out ASSESS_DIR --rev cf26bf985887fc1010c0b45f385ae44eb149ddad \
    --go-image sha256:cf9a5db7c8653137b816c41cde3dcbfecdd9e23a2b247bc5b2eaa2260f3f5871 --lock LOCK
```

`RESULT_COMMIT` is `result.commit` from the run's `suite.json`. The Go image is F2's (`markitect-v1rq-go:7371cfd99d80`); `--rev` is the `main` commit that brought the suite.

## Abort and cleanup

- **Abort:** stop the suite's process tree (`taskkill /PID <pid> /T /F` on Windows). An aborted run is recorded with its cause.
- **Cleanup:** the variant deletes its login copy and its scratch directory when the session ends. A forced stop skips that, so after an abort delete the run's leftover temporary directory, including its `config\.credentials.json`. After the series, check that no `claude` process from a run remains, that no login copy is left in the temporary folder, that no assessment container remains, that the lock is released, and that the suite's secret scan found nothing in any delivered commit or output.

## Evidence

Outside the repository, under `C:\Users\Consiliari\Documents\Codex\Markitect-Evaluation\`: the task folder `library-m1-task`, and per run its `suite.json`, the variant's `run.json`, stream, transcripts and `result.bundle`, plus each assessment. A results section is added to this page after the series, listing every run with its raw outcome, including failures.

## What a pass means

All three runs deliver a result within the deadline, with no manual help and no secret hit. That shows the suite and the baseline work end to end on Claude. It says nothing yet about quality relative to other variants.

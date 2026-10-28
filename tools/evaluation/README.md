# Evaluation suite runner

The suite runner and the baseline variant of the [evaluation suite](../../docs/workstreams/evaluation.md). The suite builds a task, runs one variant command on it under a deadline, and checks and records its result. It does not care how a variant works or which provider it uses. The assessment of results stays in [`tools/comparison`](../comparison/README.md).

| File | Use |
| --- | --- |
| [suite.py](suite.py) | `task` builds a task directory, `run` runs one variant on one task, `chain` runs a sequence of steps that each start from the variant's own previous result |
| [claude_code.py](claude_code.py) | The baseline variant: one headless Claude Code session with the task, in an isolated configuration |
| [test_suite.py](test_suite.py) | Offline tests with real Git, a fake variant and a fake `claude` |

## Use

```text
python -I -B tools/evaluation/suite.py task --task-md TASK.md [--model MODEL_DIR] [--start BUNDLE] --deadline-seconds 2700 --out TASK_DIR
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR -- python -I -B tools/evaluation/claude_code.py
```

`--model` names the evaluation's model as `PROVIDER:MODEL_ID`. The suite gives it to the variant in `EVALUATION_MODEL`.

The variant command receives `TASK_DIR OUT_DIR [STATE_DIR]`. It works on a copy of the task and writes `OUT_DIR/result.bundle`, and optionally `OUT_DIR/run.json`.

The suite writes `RUN_DIR/suite.json`. It holds the wall time, whether the variant timed out and its exit code. It also holds whether a result was delivered: its commit, tree and changed paths, or why there is none. Finally, it holds the variant's own report and whether the model in that report matches the evaluation's model. A result counts only when it is one commit that descends from the task's start.

In a chain, a step without a result is followed by a step that starts from that step's start. One state directory is handed to the variant at every step.

## The baseline

`claude_code.py` copies the task's `task.md` into a clone of the start as an uncommitted `TASK.md`. It then runs `claude -p` with a fixed prompt, the evaluation's model, no user settings, plugins, CLAUDE.md, memory or MCP servers, and a copy of the login without its refresh token. It stops the session a minute before the deadline, commits what is left with a fixed identity, and writes `result.bundle`, `run.json`, the session's stream and its transcripts. If a login secret appears in the output, it deletes the stream and the transcripts and marks the run as quarantined. Every chain step is a fresh session. The session runs as the user on the host: this is separation, not a sandbox.

A real run uses the owner's Claude subscription and needs an owner-approved bounded dispatch.

## Self-check

```text
python -B -m unittest discover -s tools/evaluation
```

On Windows, set `TMP` and `TEMP` to a user temp directory first.

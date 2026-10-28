# Evaluation suite runner

The suite runner and the baseline variant of the [evaluation suite](../../docs/guides/evaluation-suite.md). The suite builds a task, runs one variant command on it under a deadline, and checks and records its result. It does not care how a variant works or which provider it uses. The assessment of results stays in [`tools/comparison`](../comparison/README.md).

| File | Use |
| --- | --- |
| [suite.py](suite.py) | `task` builds a task directory, `run` runs one variant on one task, `chain` runs a sequence of steps that each start from the variant's own previous result |
| [sandbox.py](sandbox.py) | The sandbox of `run --sandbox` and `chain --sandbox`: one container per run on an egress-limited network |
| [sandbox/](sandbox/Dockerfile) | The sandbox image, its egress gateway (`egress.py`) and the default series configuration (`series.json`) |
| [claude_code.py](claude_code.py) | The baseline variant: one headless Claude Code session with the task, in an isolated configuration |
| [escape_probe.py](escape_probe.py) | A variant without a model that tries to leave the sandbox and reports each attempt |
| [test_suite.py](test_suite.py), [test_sandbox.py](test_sandbox.py) | Offline tests with real Git, a fake variant, a fake `claude` and a fake engine |

## Use

```text
python -I -B tools/evaluation/suite.py task --task-md TASK.md [--model MODEL_DIR] [--start BUNDLE] --deadline-seconds 2700 --out TASK_DIR
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR [--state STATE_DIR] \
    --sandbox IMAGE_ID --variant-code tools/evaluation [--login FILE] [--series FILE] [--lock FILE] \
    -- python3 -I -B /variant/claude_code.py
python -I -B tools/evaluation/suite.py chain --step STEP_DIR [--step STEP_DIR ...] [--start BUNDLE] --deadline-seconds 2700 --model claude:claude-sonnet-5-5 --out CHAIN_DIR [SANDBOX OPTIONS] -- VARIANT...
```

For `task`, `--model` is the canonical model directory. For `run` and `chain`, `--model` names the evaluation's model as `PROVIDER:MODEL_ID`; the suite gives it to the variant in `EVALUATION_MODEL`. A chain step directory holds `task.md` and an optional `model/`.

The variant command receives `TASK_DIR OUT_DIR [STATE_DIR]`. It works on a copy of the task and writes `OUT_DIR/result.bundle`, and optionally `OUT_DIR/run.json`.

The suite writes `RUN_DIR/suite.json`. It holds the wall time, whether the variant timed out and its exit code. It also holds whether a result was delivered: its commit, tree and changed paths, or why there is none. It holds the variant's own report and whether the model in that report matches the evaluation's model. Finally, it holds the conditions (`sandbox` or `host`), the sandbox record and `validForCounting`. A result counts only when it is one branch tip that descends from the task's start; a result equal to the start counts as delivered with no changes. A `false` model match makes the run invalid in the report.

In a chain, a step without a result is followed by a step that starts from that step's start. One state directory is handed to the variant at every step.

## The sandbox

With `--sandbox IMAGE_ID` the suite starts the variant command inside one container per run. `IMAGE_ID` is the engine image ID (`sha256:...`) of the image built from [sandbox/](sandbox/Dockerfile):

```text
docker build -t markitect-eval-sandbox tools/evaluation/sandbox
docker image inspect --format "{{.Id}}" markitect-eval-sandbox
```

The image is Debian bookworm with Git, the Go, Node and Python versions of the [runtime image](../../packaging/runtime/Dockerfile) and the same pinned Codex and Claude Code CLIs. It sets `GOPROXY=off`, `GOTOOLCHAIN=local` and `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`.

Each run gets, and nothing else from the host:

| Mount | Source | Mode |
| --- | --- | --- |
| `/task` | a copy of the task | read-only |
| `/out` | `RUN_DIR/variant` | writable |
| `/state` | `--state`, or the chain's state directory | writable |
| `/variant` | `--variant-code`, the variant's own code | read-only |
| `/login/credentials.json` (`EVALUATION_LOGIN`) | an access-only copy of `--login` (default: the user's Claude login), without its refresh token, deleted after the run | read-only |

The container has a read-only root, no capabilities, `no-new-privileges`, a non-root user (the host user on Linux, 1000 on Docker Desktop), `--init` and a private `/tmp` where `HOME` lives. The series configuration ([`sandbox/series.json`](sandbox/series.json) unless `--series` names another) sets one safety limit for every run of every variant: CPUs, memory without swap, processes and the `/tmp` size. It is a protection, not a budget; a run that exceeds it is a result. `suite.json` records it with the configuration's digest.

**Network.** The run's sandbox joins an internal network, which has no route out and, in the bridge's isolated gateway mode, no address of the engine's host. A second container, the egress gateway, joins that network and an ordinary one. The sandbox's `/etc/hosts` maps each endpoint the series allows for the evaluation's provider (for Claude, `api.anthropic.com`) to the gateway, and its resolver forwards to an unroutable address, so no other name resolves. The gateway reads the server name of each TLS connection on port 443 and forwards only allowed names, unchanged, to the real host; TLS stays end to end. Every tool reaches the provider by itself, with no proxy setting. Everything else is meant to fail: the web, GitHub, the module proxy, the engine and the host; the escape probe checks that on each machine before a series. The gateway's log of every connection is kept as `RUN_DIR/egress.jsonl` and counted in `suite.json`. The engine socket is never mounted.

**Cleanup.** The suite records every engine call. At the deadline it stops the container before the engine CLI, so nothing written later counts. After the run it removes both containers and both networks and counts what is left by the run's label. The run is not valid for counting when its sandbox did not stand up, its container never started, the gateway stopped during the run, or something is left. A chain stops at such a step. A run refuses to start while containers or networks of another run exist. Afterwards the run directory and the state directory are searched for the login, Git bundles included; a file that contains it, and any link or special file the variant left, is deleted unread and listed under `quarantine`. `--lock` takes the shared engine lock for the run or the whole chain and refuses when another holder has it.

**Host mode.** Without `--sandbox` the variant runs as a host process. That is for tests only: `suite.json` says `"conditions": "host"` and `"validForCounting": false`.

## The escape probe

`escape_probe.py [--endpoint HOST]` runs through `suite.py run --sandbox ... -- python3 -I -B /variant/escape_probe.py`. From inside the sandbox it tries to read host files (foreign mounts, known host mount paths, a `.git` directory, `AGENTS.md` or an oracle directory anywhere outside the suite's mounts), to reach GitHub and the web (resolving github.com or example.com at all, by name, by address, public resolvers by address, and GitHub through the gateway with GitHub's server name) and to reach a container engine or the host (socket files, `DOCKER_HOST`, the engine's TCP ports and common service ports on the gateway, the host names and the first address of each attached subnet). Each must fail. It also completes one TLS handshake with the allowed endpoint, without sending a request, to show that the network works. It writes every attempt to `run.json`, which the suite records under `reported`, delivers no result and exits 1 if anything escaped.

## The baseline

`claude_code.py [--claude BIN] [--credentials FILE] [--effort LEVEL]` takes the `claude` binary, the login file to copy (default: the suite's `EVALUATION_LOGIN`, else the user's Claude login) and an optional effort level. Run it only through the suite, which starts it in its own process group; in the sandbox, as `python3 -I -B /variant/claude_code.py` with `--variant-code tools/evaluation`. It copies the task's `task.md` into a clone of the start as an uncommitted `TASK.md`. It then runs `claude -p` with a fixed prompt, the evaluation's model, no user settings, plugins, CLAUDE.md, memory or MCP servers, and a copy of the login without its refresh token. It stops the session a minute before the deadline, commits what is left with a fixed identity, and writes `result.bundle`, `run.json`, the session's stream and its transcripts.

Safeguards:
- **Git:** clone, session and commit share one Git setting, with no system or user configuration, no line-ending conversion, no hooks and no credentials.
- **Login:** the login copy is deleted as soon as the session ends. Before bundling, every commit since the start is searched for the login. If the login appears there or in any output, nothing is delivered, the stream and the transcripts are deleted, and the run is marked quarantined.
- **Fresh sessions:** every chain step is a fresh session.

Limits:
- In host mode the session runs as the user on the host: that is separation, not a sandbox.
- The repository's own `.claude/` settings still load, as they would for any developer. This includes settings the agent wrote in an earlier step.
- On Windows in host mode, a background process that the session leaves behind after a normal exit is not stopped. On Linux and in the sandbox, the whole process group is stopped.

A real run uses the owner's Claude subscription and needs an owner-approved bounded dispatch.

## Self-check

```text
python -B -m unittest discover -s tools/evaluation
```

On Windows, set `TMP` and `TEMP` to a user temp directory first.

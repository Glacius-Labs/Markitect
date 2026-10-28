# Evaluation suite E1: sandbox smoke

**Status: run on 28 October 2026, passed, with one defect in K's capture found and fixed ([results](#results)).** The owner approved it in the Build session the same day: "I approve, complete the sandbox if you think it will do". It replaces the [host smoke](eval-k-library-smoke-20261028.md), which was never run.

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

## Results

**Source.** The runs used the PR branch `dev/eval-v2-e1-sandbox`, not `main`: PR #822 was not merged yet. The image was `sha256:e0b20d8a8c88a0a4b37f151af45618e764f8687b023b10bcb6d3b5148c5e5dad`, built from `4f7cf4d5`. Probe-03 and the K runs used `41a61b24`, which differs from `4f7cf4d5` only in the probe fix below. Docker Desktop 29.9.2 had 16 CPUs and 31 GiB. Every launch is listed, in order:

| Launch | Outcome | Class |
|---|---|---|
| probe-01 | The variant never ran: Git Bash rewrote `/variant/escape_probe.py` into a Windows path before Python saw it. The sandbox itself stood up and was removed cleanly (gateway `192.168.80.1`, no residue) | Void: operator error. All later launches went through PowerShell |
| probe-02 | `escaped`, on one finding only: `/usr/sbin/docker-init`, the engine's `--init` binary, reported as a foreign mount. The probe allowed only `/sbin/docker-init`. Every other attempt was blocked: no name resolved (github.com, example.com, `host.docker.internal`), GitHub, 1.1.1.1 and 8.8.8.8 were unreachable by address ("Network is unreachable"), the gateway closed a TLS handshake for github.com, and no engine socket or engine port answered. The allowed endpoint completed TLS 1.3 through the gateway | Void: probe defect, fixed in `41a61b24` |
| probe-03 | `contained`, `egressWorks: true`, valid for counting, no residue, no quarantine. The gateway log shows `api.anthropic.com` allowed and `github.com` refused | Pass |
| k-01 | Delivered in 152 s: model match, valid for counting, no quarantine. The session completed successfully in 7 turns. Its result commit lacks four of the nine source files the session reports writing (`cmd/library/main.go`, `internal/library/date.go`, `library.go`, `library_test.go`), see below | Result |
| k-02 | Delivered in 204 s: 11 paths, model match, valid for counting, no quarantine | Result |
| k-03 | Delivered in 206 s: 10 paths, model match, valid for counting, no quarantine | Result |

**Network.** Docker Desktop accepted the internal network in isolated gateway mode, so the bridge had no host address and the gateway container took the subnet's first address. The gateway's `--sysctl` and the sandbox's `--dns 192.0.2.1` were also accepted. Claude Code 2.1.308, with an access-only login and `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`, reached only `api.anthropic.com`: three connections per run, and no refused name.

**Defect found in K.** k-01's session wrote a `.gitignore` with the line `library` to ignore its binary. That pattern also matches the `cmd/library/` and `internal/library/` folders. K's capture ran `git add -A`, so it silently dropped those sources. This is K's own capture machinery, not the agent's work: the agent's tree was complete, but the delivered commit was not. No technical-failure signature named this before the series, so k-01 stays a result, as the protocol requires. The fix keeps ignored Go sources, module files and Markdown, which is the Phase A–C capture rule in [`tools/comparison/runner.py`](../../tools/comparison/runner.py). It comes with an offline test of this exact case. K has not run against a provider since the fix.

**Cleanup.** After the series, no container or network with the label `markitect.evaluation` remained. No login copy was left in the temporary folder, the lock was released, and no `quarantine` entry appeared.

**Usage, descriptive.** k-01 to k-03 used 7, 13 and 14 turns, 24,238 to 32,076 output tokens, and 0.41 to 0.55 USD equivalent as Claude Code reports it.

**Verdict.** The smoke criterion is met: the probe found the sandbox closed, and K delivered three times in a row with a result, a model match and no quarantine. Because of the capture defect, k-01's result does not show the agent's full work. Judging it in E2 would understate K, so it should not be judged.

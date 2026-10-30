# Evaluation suite runner

The suite runner, its judging and reporting, and the baseline variant of the [evaluation suite](../../docs/guides/evaluation-suite.md). The suite builds a task, runs one variant command on it under a deadline, and checks and records its result. It then judges the result with the task's hidden judge, reviews its code blinded, classifies every launch and reports per area. It does not care how a variant works or which provider it uses. The Library judge and the review reuse the Phase A–C assessment, prompts and rubric check in [`tools/comparison`](../comparison/README.md).

| File | Use |
| --- | --- |
| [suite.py](suite.py) | `task` builds a task directory, `run` runs one variant on one task, `chain` runs a sequence of steps that each start from the variant's own previous result; `judge`, `judge-chain`, `review` and `report` evaluate them |
| [sandbox.py](sandbox.py) | The sandbox of `run --sandbox` and `chain --sandbox`: one container per run on an egress-limited network |
| [series.py](series.py) | The change series of the portfolio: each step's `task.md` and concept 2.0 model as a chain step (`steps --series library\|makerspace`) |
| [native.py](native.py) | The native arm mode of `run --native` and `chain --native`: the variant's host on the machine, its own containers on the same egress-limited network, checked from the engine's events |
| [native_standin.py](native_standin.py) | A scripted stand-in for a native arm: one Run container per run, which writes a change, changes nothing, leaves the network, leaves residue or runs the escape probe |
| [markitect_arm.py](markitect_arm.py) | The concept 2.0 arm: the owner's model commit, one `markitect apply` with its agents on the suite's network, and the landed commit or the unchanged start as the result |
| [sandbox/](sandbox/Dockerfile) | The sandbox image, its egress gateway (`egress.py`) and the default series configuration (`series.json`) |
| [claude_code.py](claude_code.py) | The baseline variant: one headless Claude Code session with the task, in an isolated configuration |
| [codex_cli.py](codex_cli.py) | The baseline variant on Codex: one `codex exec` session with the task, in the same setting |
| [escape_probe.py](escape_probe.py) | A variant without a model that tries to leave the sandbox and reports each attempt |
| [judging.py](judging.py) | `suite.py judge`: runs a task's judge folder in one container without network and checks its `judge.json` |
| [judges.py](judges.py) | Judge sources in [`tools/evaluation-judges`](../evaluation-judges/library-m1/README.md): `export` writes a judge folder from a frozen revision, `selftest` runs a judge on its references and seeded mutants, `start` writes a fixed reference start |
| [reviewing.py](reviewing.py), [reviewer.py](reviewer.py) | `suite.py review`: the blinded copy, the reviewer and confirmation sessions in the sandbox, the citation check and the variant guess |
| [classify.py](classify.py) | The failure classifier: void (provider, infrastructure, signed variant runtime), invalid or result |
| [tokens.py](tokens.py) | The suite's token count over every transcript of a run, subagents included |
| [results.py](results.py) | The Result contract and its check: every checklist item has a verdict with evidence |
| [reading.py](reading.py) | The paired reading of a contrast: the 95 % interval of the paired difference |
| [report.py](report.py), [acceptance.py](acceptance.py) | `suite.py report`: one table per area for a series ledger, then one per acceptance criterion of concept 2.0 §15 |
| [series_runner.py](series_runner.py) | The [series runner](#the-series-runner): a counted series or its stability gate, one launch at a time, with its judging, reviews, ledger and report |
| [test_suite.py](test_suite.py), [test_sandbox.py](test_sandbox.py), [test_judging.py](test_judging.py), [test_series.py](test_series.py), [test_native.py](test_native.py), [test_markitect_arm.py](test_markitect_arm.py), [test_acceptance.py](test_acceptance.py), [test_reviewing.py](test_reviewing.py), [test_classify.py](test_classify.py), [test_series_runner.py](test_series_runner.py) | Offline tests with real Git, fake variants, a fake `claude` and `codex`, a scripted `markitect`, fake engines, a fake suite, and recorded runs and transcripts ([testdata/](testdata/README.md)) |

## Use

```text
python -I -B tools/evaluation/suite.py task --task-md TASK.md [--model MODEL_DIR] [--start BUNDLE [--base-model MODEL_DIR]] --deadline-seconds 2700 --out TASK_DIR
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR [--state STATE_DIR] \
    --sandbox IMAGE_ID --variant-code tools/evaluation [--login FILE] [--series FILE] [--lock FILE] \
    -- python3 -I -B /variant/claude_code.py
python -I -B tools/evaluation/suite.py chain --step STEP_DIR [--step STEP_DIR ...] [--start BUNDLE [--state STATE_DIR]] --deadline-seconds 2700 --model claude:claude-sonnet-5-5 --out CHAIN_DIR [SANDBOX OPTIONS] -- VARIANT...
python -I -B tools/evaluation/suite.py judge --run RUN_DIR --judge JUDGE_DIR [--step STEP] --out JUDGMENT_DIR [--lock FILE]
python -I -B tools/evaluation/suite.py judge-chain --chain CHAIN_DIR --judge JUDGE_DIR --steps W0,C1,... --out DIR [--lock FILE]
python -I -B tools/evaluation/suite.py review --run RUN_DIR --judge JUDGE_DIR --out REVIEW_DIR --sandbox IMAGE_ID --variant-code tools/evaluation [--lock FILE]
python -I -B tools/evaluation/suite.py report --ledger LEDGER.json --out REPORT_DIR
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR \
    --native IMAGE_ID --hidden-absent PATH [--hidden-absent PATH ...] [--lock FILE] -- VARIANT...
```

For `task`, `--model` is the canonical model directory, and `--base-model` the model a fixed reference start realizes (the step before's), which the task carries as `base-model/` for a variant that works from a model; every variant receives it, K without its registry pages. For `run` and `chain`, `--model` names the evaluation's model as `PROVIDER:MODEL_ID`; the suite gives it to the variant in `EVALUATION_MODEL`. With `--without-pages`, the variant's copy of the task has no registry pages: `registry.md` and `registry/` go from its `model/` and `base-model/`, and `suite.json` lists them under `pagesWithheld`. The series runner passes it for K, which gets the same files as the 2.0 arm without the pages, as Build decided on 29 October 2026. A chain step directory holds `task.md` and an optional `model/`.

The variant command receives `TASK_DIR OUT_DIR [STATE_DIR]`. It works on a copy of the task and writes `OUT_DIR/result.bundle`, and optionally `OUT_DIR/run.json`.

The suite writes `RUN_DIR/suite.json`. It holds the wall time, whether the variant timed out and its exit code. It also holds whether a result was delivered: its commit, tree and changed paths, or why there is none. It holds the variant's own report and whether the model in that report matches the evaluation's model. Finally, it holds the conditions (`sandbox` or `host`), the sandbox record and `validForCounting`. A result counts only when it is one branch tip that descends from the task's start; a result equal to the start counts as delivered with no changes. A `false` model match makes the run invalid in the report.

In a chain, a step without a result is followed by a step that starts from that step's start. One state directory is handed to the variant at every step. `--state` continues an earlier chain, for example after a void step: the new chain gets the earlier chain's state directory, and `--start` its last delivered result.

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
| `/login/credentials.json` (`EVALUATION_LOGIN`) | an access-only copy of `--login` (default: the user's login of the evaluation's provider, `~/.claude/.credentials.json` or `~/.codex/auth.json`), deleted after the run: a Claude login without its refresh token, a Codex login with an empty one, as the host's Codex adapter copies it, and never an API key | read-only |

The container has a read-only root, no capabilities, `no-new-privileges`, a non-root user (the host user on Linux, 1000 on Docker Desktop), `--init` and a private `/tmp` where `HOME` lives. The series configuration ([`sandbox/series.json`](sandbox/series.json) unless `--series` names another) sets one safety limit for every run of every variant: CPUs, memory without swap, processes and the `/tmp` size. It is a protection, not a budget; a run that exceeds it is a result. `suite.json` records it with the configuration's digest.

**Network.** The run's sandbox joins an internal network, which has no route out and, in the bridge's isolated gateway mode, no address of the engine's host. A second container, the egress gateway, joins that network and an ordinary one. The sandbox's `/etc/hosts` maps each endpoint the series allows for the evaluation's provider (for Claude, `api.anthropic.com`; for Codex, `chatgpt.com`, where a ChatGPT account login reaches the model) to the gateway, and its resolver forwards to an unroutable address, so no other name resolves. The gateway reads the server name of each TLS connection on port 443 and forwards only allowed names, unchanged, to the real host; TLS stays end to end. Every tool reaches the provider by itself, with no proxy setting. Everything else is meant to fail: the web, GitHub, the module proxy, the engine and the host; the escape probe checks that on each machine before a series. The gateway's log of every connection is kept as `RUN_DIR/egress.jsonl` and counted in `suite.json`. The engine socket is never mounted.

**Cleanup.** The suite records every engine call. At the deadline it stops the container before the engine CLI, so nothing written later counts. After the run it removes both containers and both networks and counts what is left by the run's label. The run is not valid for counting when its sandbox did not stand up, its container never started, the gateway stopped during the run, or something is left. A chain stops at such a step. A run refuses to start while containers or networks of another run exist. Afterwards the run directory and the state directory are searched for the login, Git bundles included; a file that contains it, and any link or special file the variant left, is deleted unread and listed under `quarantine`. `--lock` takes the shared engine lock for the run or the whole chain and refuses when another holder has it.

**Host mode.** Without `--sandbox` the variant runs as a host process. That is for tests only: `suite.json` says `"conditions": "host"` and `"validForCounting": false`.

## The native arm mode

With `--native IMAGE_ID` ([native.py](native.py)) the variant's host runs natively on the machine, outside the sandbox. It is for a variant whose host is trusted, reviewed code that runs its agents only in containers of its own: the concept 2.0 arm. The suite stands up the same internal network and egress gateway as for the sandbox (from the sandbox image `IMAGE_ID`) and starts the variant command as a host process with `EVALUATION_MODEL`, the access-only login copy in `EVALUATION_LOGIN`, and what its containers need to join the network: `EVALUATION_NETWORK`, `EVALUATION_HOSTS` (provider endpoints at the gateway, for `--add-host`), `EVALUATION_DNS` (a resolver that resolves nothing), `EVALUATION_LABEL` (the run's ID as `KEY=VALUE`, for the variant's containers or names) and `EVALUATION_ENGINE`.

While the variant runs, the suite records the engine's events (`RUN_DIR/engine-events.jsonl`) and inspects every network created during the run as soon as its event arrives. Afterwards every container started during the run, other than the gateway, must have joined only the internal network, internal networks the variant created during the run (a 2.0 Run's own network between its remote and its agents), or no network; one that joined any other network, or a network that could not be inspected, is an **escape**, and the run is invalid. Networks are told apart by ID, so a name used again for another network does not pass as the first. An internal network reaches no other network and no outside address; unless its bridge's IPv4 gateway mode is `isolated`, like the suite's own, its containers can still reach the engine's host at the bridge address. `armNetworks` in the record lists the variant's networks with whether each was internal and its gateway mode. The gateway is known by its container ID, not by a label the variant could copy. The check fails closed: an event record that ends early, a reader of it that stops, or a record that never shows the run's own network makes the run void. Containers, networks and volumes the variant created and left after a normal end are residue (the run is void), and so is what the variant's host reports it left, as a list of texts under `leftovers` in `run.json`, such as a Run branch or remote its cleanup could not remove; at the deadline the suite stops them, as it stops a sandbox. **Hidden material stays off the machine:** each `--hidden-absent PATH`, where the protocol says hidden material could lie, must not exist before or after the run. `suite.json` says `"conditions": "native"`.

**The stand-in arm** ([native_standin.py](native_standin.py)) tests this the same way on Windows and on Linux without a provider: `-- python -I -B tools/evaluation/native_standin.py --image IMAGE_ID [--probe]`. Its one Run container acts as the first line of `task.md` says (`change`, `nothing`, `escape`, `leave`); with `--probe` it runs the escape probe as an agent in a Run container.

## The concept 2.0 arm

[markitect_arm.py](markitect_arm.py) runs the 2.0 implementation as a variant, natively:

```text
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR \
    --native SANDBOX_IMAGE_ID --hidden-absent PATH --lock FILE -- \
    python -I -B tools/evaluation/markitect_arm.py --markitect BIN --remote-binary BIN --manager-image IMAGE_ID [--verify]
```

`--markitect` is the `markitect` program of the frozen revision for the host, `--remote-binary` its `markitect-remote` built for Linux, and `--manager-image` the Manager image with the project's Go toolchain (`packaging/runtime` and its Go example), by image ID. The required checks run in the image `execution.yaml` names (`golang:1.27`), which must be on the engine. Per task the arm:

1. **Commits the owner's input** in a private clone of the start, with the owner's identity, before the Run: the task's `model/` as `.markitect/` (manifest, lock, `execution.yaml`, `model/` and the project's registry pages). This commit is the task's input, not the arm's work; the judges see its model files as the task's model, and its registry pages as bookkeeping. A start that already holds a model keeps its registry pages, whose tables are its Managers'. The owner's input must compile (`markitect check`); otherwise the arm delivers nothing.
2. **Adopts a fixed reference start**, which holds files but no model: one commit with the task's `base-model/` (without one, the task's model) and its registry pages, then the model change as a second commit. For the portfolio's reference results the base model carries the pages an owner wrote for that reference (`series.py base`): every file on its owner's page with the Definitions it realizes, so impact reaches the files that realize a changed Definition, as on a project Markitect built. Only a file that no page names gets a row without links, on the page `markitect check` names as its owner's. `run.json` records which was used under `adoption`: `registry` is `pages`, `unlinked` or `pages+unlinked`, with `pageRows` and `unlinkedRows`.
3. **Runs `markitect apply`** against a private bare repository as the target: `--full` on a fresh build or an adopted start without a base model, `--base` the adoption commit after one, and the host's default impact, from the last landing, when the start is the arm's own last landing, which its state directory records across a chain. The Run is named after the suite's run ID (the value of `EVALUATION_LABEL`), so its containers and volumes carry it in the host's own label `markitect.run`; its deadline is what is left of the task's after the owner's commits, minus three minutes for delivering (K keeps one minute back).
4. **Delivers** the landed commit, or the start unchanged when the Run did not land, so that a stop scores as "did not land". The Result (`markitect apply --json`, the [Result contract](results.py)) goes to `OUT_DIR/results/RUN.json`, the agents' transcripts to `OUT_DIR/transcripts/`, the host's error output to `OUT_DIR/apply.stderr.log` and the arm's own record to `OUT_DIR/run.json`: the model the agents reported, the mode, the owner's and adoption commits, the Run's state and why, tokens, starts and failed starts, and what the host says it left: `cleanup` (what may remain, recorded but not counted: the engine's census finds what really does), and after a landing `afterLanding`, what the cleanup after landing left, also under `leftovers`, which the suite counts as residue. `afterLandingReported` is false when a host that predates it leaves it out: not reported rather than clean, and never counted.

The host gets the suite's network (`MARKITECT_NETWORK`, `MARKITECT_HOSTS`, `MARKITECT_DNS` from `EVALUATION_NETWORK`, `EVALUATION_HOSTS`, `EVALUATION_DNS`), the access-only login (`MARKITECT_LOGIN`), the provider and model from `EVALUATION_MODEL`, the effort (`--effort`, default `high`) and the engine. Every `ANTHROPIC_*`, `CLAUDE_*`, `OPENAI_*`, `CODEX_*`, `MARKITECT_*`, `GIT_*`, `GH_*` and `GITHUB_*` variable of the caller is cleared first: the account login only, never an API key. Its Manager containers join the evaluation network and the Run's own internal network, its remote only the latter; the host's own limits apply (each Manager 1 CPU, 2 GiB, 256 processes, 256 MiB temporary space; the remote none), not the series' safety limit. The Run's network is internal but not in isolated gateway mode, so its containers can reach the engine's host at its bridge address (on Docker Desktop, the engine's virtual machine); `armNetworks` records it.

The arm exits 0 when the Run landed, found nothing to do or stopped on a model defect; 1 when it ended otherwise (incomplete, kept) and the start was delivered; 2 when it could not run, with nothing delivered: the owner's input does not compile, the host could not start the Run, its gate failed, or the Run did not land and no agent completed a turn (`noAgentTurn`). None of these is the arm's work, and an unchanged start would otherwise score as a stop on a planted defect; a series protocol voids them with a variant runtime signature on `reported.status` (`could-not-run`, `check-failed`) or `reported.noAgentTurn`. A Run that ended incomplete because of the infrastructure, with failed starts of class `other`, is a result unless a variant runtime signature of the series protocol matches it (`reported.failedStarts` in `suite.json`).

**`--verify`** makes the arm check the task's start as it is, with `markitect verify --full` instead of `apply`. That is concept 2.0 §15.2's independent check of a chain's end state, whichever arm made it.

- **The start:** a start that holds a model is checked against the task's model with its own registry pages. One that holds files but no model, such as K's end state, is first adopted: every file gets a row without links on the page of its owner. An empty start gets the model alone.
- **The Result** goes to `OUT_DIR/results/RUN.json` as for `apply`. The start is delivered unchanged, since verify changes nothing. `run.json` records the mode (`verify`, `verify-adopted`, `verify-empty`), the adoption and the number of Findings.
- **Exit codes:** 0 when verify ended `verified` or `findings`, 1 when it ended otherwise, and 2 as for `apply`.

## Judging

A task's hidden judge is a folder: `judge.json` and the judge's files (oracle, probes, rubric). `judge.json` names the judge image by digest, the command, a time limit and container limits ([judging.py](judging.py)). The suite appends `TASK_DIR START_BUNDLE RESULT_BUNDLE OUT_DIR` to the command. It runs the command in one container without network, with read-only copies of the judge folder at `/judge`, the task at `/task`, the start bundle at `/start/start.bundle` and the result bundle at `/result/result.bundle`, and `/out` writable. The container has a read-only root, no capabilities, `no-new-privileges`, a non-root user and a private `/tmp`; the suite stops it at the time limit and removes it.

The judge writes `/out/judge.json` with the result's commit and three areas: `correctness` (cases planned, passed and failed, and the failed cases' IDs), `buildAndTests` (the result's own build and tests, each with `passed`) and `scope` (every changed path, the unrequested ones and any protected input touched). The suite checks that the counts add up and that the judge judged this result. It writes `JUDGMENT_DIR/judgment.json` with the judge folder's digest, the image and its local ID, every engine call, the judge's output and `valid`. A judgment is valid only when the judge exited cleanly within its time, its output is complete and none of its containers is left. An invalid judgment is a problem of the judge, not of the result: judge again. A run without a result is not judged.

A judge folder in this repository is a source. `judges.py export --source DIR --rev REV --out JUDGE_DIR` writes the folder the suite runs: each file named in the source's `include` map, taken from the frozen revision `REV`, with its SHA-256 under `frozen`. `judges.py selftest --source DIR --judge JUDGE_DIR --task TASK_DIR --out DIR [--lock FILE]` builds the source's reference result and each seeded mutant on the task's start and judges each in the container. It passes when the reference passes every area and every mutant fails where it is seeded.

## Change series

The steps come from [series.py](series.py): `python -I -B tools/evaluation/series.py steps --series library|makerspace --out DIR` writes one step directory per step, named as the judges' steps (Library `M1` to `M5`, Makerspace `W0` to `C9`), each with `task.md` (the rendered single-agent document) and `model/` (the step's concept 2.0 `.markitect/` content from [tools/evaluation-tasks](../evaluation-tasks/README.md), with the project's registry pages, which give each Package its paths). The **planted tasks** are records in [tools/evaluation-judges/plants](../evaluation-judges/plants/README.md); `judges.py plant` writes a planted step and its judge folder.

A change series runs two ways. **End to end**, `chain` runs every step from the variant's own previous result, and `judge-chain` judges each delivered step at its step. **From fixed reference starts**, each change runs as an independent task whose start is the reference result of the step before: `judges.py start --source DIR --step STEP --out START.bundle` writes it, `series.py base --series NAME --step STEP --out BASE` writes the model it realizes with the owner's registry pages for it, `task --start START.bundle --base-model BASE` makes the task, and `judge --step` judges the run. The base model lets the 2.0 arm commit the start's model and registry before the change, so that its Run works on the change, not on the whole model. The start itself is the same for every arm; only the models carry the pages, and the series runner hands K its task without them (`--without-pages`). `series.py base --unlinked` leaves the pages out on purpose, so that the arm adopts every file without links, for a regression run of that fallback.

A **series judge** names its `steps` in `judge.json`. The suite passes the step in `EVALUATION_STEP` and the judge answers cumulatively: every case of this step and of every earlier one, as this step renders them. Its `judge.json` adds `step`, `correctness.byStep` (planned, passed and failed per step that introduced the cases, for every step up to this one; the suite refuses a judgment that leaves one out) and `drift`:

- `forgottenRules`: cases of an earlier step that fail now;
- `staleFiles`: terms a step retired, still in the program's non-test source;
- `unaccounted`: files of the whole tree that neither the build (`go list` on Linux, Windows and macOS), the documentation, the module files, test data nor Markitect's bookkeeping account for; `null` when the build cannot be listed.

`drift` is computed on the whole tree, so the judgment of a chain's last step is the series' drift check, the same for every variant. `judge-chain` writes `chain-judgment.json`: each step's counts and drift, the **regressions** (cases that passed at one step and fail at a later one) and `finalDrift`, with `lastStepJudged` saying whether the chain reached and judged its last step.

A series judge source names one reference per step (`references`: a tree and the prefix it goes to) and may name `patchMutants`, a folder of oracle-family mutants (`STEP` and `mutant.patch` per family), each of which must fail a case of its family. `selftest` builds one task per step from a fixed reference start, judges every step's reference (which must pass every area with no drift) and every mutant at its step; `--case NAME` runs a selection.

**The Library and Makerspace series judges** ([library](../evaluation-judges/library/README.md), [makerspace](../evaluation-judges/makerspace/README.md)) share [judgekit.py](../evaluation-judges/common/judgekit.py). The Library judge covers M1 to M5 with the Phase A and B cases; the Makerspace judge covers W0 to C9 with the hidden Makerspace oracle and its change census at C3, C7 and C9.

**The Library M1 judge** ([tools/evaluation-judges/library-m1](../evaluation-judges/library-m1/README.md)) is the comparison's M1 assessment in one container. It runs the [Library oracle](../library-oracle/README.md) with its opposite-profile witness, the six [hidden probes](../comparison/probes.py), the result's own `go build ./cmd/library` and `go test ./...`, and [assess_candidate.py](../comparison/assess_candidate.py)'s census of every changed path. Its image is the pinned `golang:1.27.1-bookworm`, the Go of the sandbox, whose Python comes with Mercurial.

## The review

`suite.py review` ([reviewing.py](reviewing.py)) makes a blinded copy of a delivered result in a directory named by a random token. The copy has `candidate/` and `start/` trees and `diff.patch`, without `.markitect/` (and the registry in it) and without compiled executables; the code is otherwise unchanged. A fresh reviewer session ([reviewer.py](reviewer.py)) judges it against the judge folder's `rubric.md` and the task's `task.md`, with the frozen review prompt of [`tools/comparison/review-prompt.md`](../comparison/review-prompt.md). Every critical finding goes through [rubric_check.py](../comparison/rubric_check.py). Each valid one goes to a second, fresh session with the frozen confirmation prompt and counts only when it is confirmed. Both prompts get one added sentence that asks the reviewer to guess which of K, M, P and G produced the program.

Each session runs in the evaluation sandbox like a variant (`--sandbox IMAGE_ID --variant-code tools/evaluation`), on Claude through the CLI with the access-only subscription login, never an API key. It may use only Read, Glob and Grep, in the baseline's isolated configuration. `REVIEW_DIR/review.json` holds the answers, the citation check, the confirmations, the counts of reported, valid and confirmed critical findings, every guess and `valid`. Without `--sandbox` the sessions run as host processes, for tests only.

## Failures, tokens and the report

[classify.py](classify.py) sorts every launch with one rule for every variant:
- **Void**, a technical failure that is recorded, relaunched and never counted. It covers the infrastructure: the sandbox did not stand up, its container never started, its gateway stopped, it left residue, the suite wrote no record, or the variant command could not start. A server name the gateway refused also stops the series. It covers the provider: a session's last response is an API error for capacity, a rate or usage limit, a server error or a refused login, and the run did not end normally. A Codex exec session, `codex_cli.py`'s, is read from its event stream: it completed no turn, and its failed turn's error, else an error event the stream ends on, has such a class by the host Codex adapter's rule for the CLI's error text. A variant whose Results list its agent starts, the 2.0 arm, is read from them instead: a Manager's last start failed with the class `capacity`, `rate-limit`, `server-error` or `login`; a failed start that the same Manager followed with another one is its work, like an error a single session recovered from. And it covers a variant runtime signature written into the series protocol before the series.
- **Invalid:** host mode, or a reported model other than the evaluation's.
- **Result:** everything else, including a wrong or incomplete result, a timeout and a missing result.

[tokens.py](tokens.py) counts tokens over every `*.jsonl` file a run left: Claude Code transcripts, subagent transcripts and stream output alike, each response once by its message ID. A session that left only stream output, as each agent of the 2.0 arm does, is counted from the `result` line each `claude -p` process ends with, since stream lines carry a response's usage as it stood mid-response; those lines also give its stated cost. It matches a hand count and the session's own totals on a recorded session with 22 subagents ([testdata/](testdata/README.md)).

`suite.py report --ledger LEDGER.json` ([report.py](report.py)) reads a series ledger, a JSON file that lists every launch with its variant, task, run, judgment and review. It writes `report.md` and `report.json`: one table per area and a list of every launch with its class. The areas are human attention first, then Correctness, Build and tests, Scope, Code quality, Delivery, the void rate and the descriptive cost. There is no overall score. A result without a delivered commit scores as its start, the change not done: it passes the earlier steps' cases and fails the step's own new ones. Both counts come from the series judge's `byStep` of any judged launch of the same task and step ([acceptance.py](acceptance.py), `start_scores`); a judge that does not count by step has only the step's own cases. It also counts once more as a step with no result.

Then come the acceptance areas ([acceptance.py](acceptance.py)), one table per criterion of [concept 2.0 §15](../../docs/concepts/v2-concept.md#15-acceptance). The ledger adds `repetition` and `step` to each launch, `chains` (each with its `judge-chain` output and an optional `verifyFull` Result), `contrasts` and, per planted task, `tasks.NAME.planted`:

| Criterion | Table |
| --- | --- |
| Correct | Landed changes (delivered and different from the start) that pass every hidden case |
| No drift | Per chain: forgotten rules, stale files and unaccounted files after its last step, the regressions along it, and the variant's own `verify --full` where it has one |
| Nothing skipped | Every Result a variant returns ([results.py](results.py)): items without verdict, without evidence, without an approving checker, and checks that did not run on the whole repository |
| Stops right | Planted defects without landing, harmless gaps landed; a launch without result counts as wrong |
| At least as trustworthy as K | Per contrast, the paired difference by task, step and repetition of its endpoint (`casesPassedShare`, `casesFailed` or `defectsLeft`), its 95 % interval ([reading.py](reading.py)), read over the pairs or, with `blockBy` `change`, over each change's mean difference across its repetitions; the design's reading and whether the variant is at least as good. Where units are unpaired, because one variant has no result there, the contrast is read a second time with each such unit scored as the change not done for that variant. The verdict is yes or no only when both readings agree, and otherwise "undecided (N unpaired)". A contrast without `minimumDifference` is descriptive (`ownCasesPassedShare`, the step's own new cases passed). Where the ledger sets `ownerRule`, as the series runner does for its `NAME-fixed` reports, and both `casesPassedShare` and `casesFailed` are contrasts, a last table gives the owner's rule (30 October 2026), which is defined on fixed starts: at least as good only when both hold and the variant's landed-change rate is not below the baseline's, over the paired units and with an absent result counted as not landed, plants left out |

A planted defect's launch counts under Stops right only. **The Result contract:** a variant that keeps a checklist writes one Result per Run as `OUT_DIR/results/RUN.json` (format in [results.py](results.py)); `python -I -B tools/evaluation/results.py RUN_DIR` checks one run. **A planted task's** judge folder holds `plant.json` (kind, expected outcome, description, time recorded); `suite.py judge` scores the outcome itself ([judging.py](judging.py)).

## The series runner

[series_runner.py](series_runner.py) runs a counted series or its stability gate as its protocol says, one launch at a time, through `suite.py`:

```text
python -I -B tools/evaluation/series_runner.py plan --protocol counted|gate --machine MACHINE.json [--prepared SPEC.json ...] --out CONFIG.json
python -I -B WORK/src/tools/evaluation/series_runner.py units CONFIG.json
python -I -B WORK/src/tools/evaluation/series_runner.py run CONFIG.json [--prep-only]
python -I -B WORK/src/tools/evaluation/series_runner.py judge CONFIG.json
```

`MACHINE.json` names the work folder, the frozen revision, the folder of the binaries built from it, the images built from it, the engine lock, the places hidden material could lie, the login file and the model.

**Plans.** `plan` adds the protocol's units in their order and its contrasts:

- **`counted`:**
  - The Library and the Makerspace: every fixed start and plant twice per arm, and each chain once.
  - Each `--prepared` series (the new project): its fixed starts five times, its plants three times and its chain twice.
  - The order is by repetition. The first arm alternates per task and per repetition, the task order is turned per repetition, and the chains come last.
  - The contrasts: the share of hidden cases passed, with a minimum meaningful difference of 0.05, and the failed hidden cases, with 1; both are read over each change's mean difference (`blockBy` `change`). Defects left for a person are a third contrast, with 1.
- **`gate`:**
  - The Library's fixed starts M2, M3 and M4, and the Makerspace chain, once per arm.
  - The first fixed start of each prepared series once per arm, so that both arms have run on its inputs, with a verify launch on each result, so that the verify launch has adopted each arm's result there.

In both plans, every chain ends with a verify launch (see Chains below), right after its last step. The launch bound is one and a half times the planned launches, verify launches included. `units` prints the order and its counts.

**Inputs.** `run` runs from the frozen revision's `tools/`, extracted with `git archive` into `WORK/src`, and logs its own SHA-256. It prepares the Library's and Makerspace's inputs in `WORK/prep/SERIES`, as [change series](#change-series) describes, with the registry pages on every fixed start.

A prepared series' author writes its inputs in the same layout, and `--prepared SPEC.json` names them:

- **The spec:** `name`, `steps`, `fixed`, `plants` with each plant's `step` (its kind stays in its judge folder), `deadlineSeconds`, `inputs`, `judges`, and optionally `inputEffort` per task.
- **Launch inputs (`inputs`):** `steps/STEP/` for a chain, `tasks/STEP/` for a fixed start and `plants/NAME/task/` for a plant.
- **Judge folders (`judges`):** `judge/`, the series judge, and `plants/NAME/judge/` with its `plant.json`. For a prepared series they are there only while it is judged.

**Each launch:**

- **Before it:**
  - no `WORK/STOP`;
  - the launch bound not reached;
  - no hidden material on the machine (the configuration's places and every prepared series' `judges`);
  - a quiet engine, then a login that covers the launch, then a quiet engine again.

  The engine is quiet when no container without a `markitect.*` label runs and none started in the last ten minutes, as a `docker events` stream kept open for the whole window shows. The stream is needed because the engine keeps only its last 256 events, fewer than most native runs write. Engine waits go into the log and into the ledger's `waits`.
- **The login:** it must outlive the launch's deadline, or a review's hour, by twenty minutes. Otherwise the runner takes the engine lock and sets the login's expiry to now, with no token changed. Then `claude -p` refreshes it, as the CLI does near expiry. The runner replaces the file only if no other process changed it in between, and it puts the expiry back if the CLI did not refresh it. An expired refresh token stops the series.
- **The engine lock:** the runner holds it only for that refresh, between launches, never while a suite command runs. `suite.py` takes it for each run, judgment and review. While another holder has it, the runner waits and does not count the launch.
- **Classes:** each launch is classed with its arm's variant runtime signatures, as the report classes it. These are in `signatures.json`: for M, `reported.status` is `could-not-run` or `check-failed`, or `reported.noAgentTurn` is true; for K, `reported.status` is `error`, which `claude_code.py` writes when its own code fails.
  - A void launch is relaunched up to twice.
  - A launch invalid only because of other work's containers is relaunched once. Those are containers without a `markitect.*` label whose start the run's own engine events show. Any other invalid launch stops the series.
  - **After a stop is cleared:** a launch that stopped the series is launched again only if it was void with relaunches left, or if it was the first invalid one to stop it. Otherwise the series goes on with the next unit.
  - In the gate, any launch that is not a result stops it, and only a result completes a unit.
- **Stops:** the runner stops the series on any of these, and writes the reason to `WORK/STOP`:
  - a refused request, or a seven-day use of 90 %;
  - a credential of the login in a run's evidence;
  - a gateway refusal of a provider name;
  - hidden material on the machine;
  - a void share above one in four for an arm after its first eight launches;
  - a judge that gives no valid judgment twice;
  - a frozen file, binary or image that changed since the first window;
  - the launch bound;
  - a login that cannot be refreshed;
  - the runner's own failure.

  A five-hour use of 85 % waits for that window's reset instead. The ledger keeps that wait, so a window that ends during it waits out the rest when the series goes on.
- **Judging:** each delivered result is judged at its step, after the quota and evidence checks. A chain's counted steps are judged as one chain after its last step; the suite judges a chain's delivered steps only.
  - **Pending:** a judgment stays under `pending` in the ledger until it is valid. That covers a judge folder off the machine, a window that ends during a judgment, and a runner that does not live to finish one. Each window starts by making what is pending and whose judge is there. A judgment that is not valid is made once more, each time in a fresh folder.
  - **A judge that keeps failing:** only the window that made the launch stops for its judge. An item whose judge failed twice is tried again only after the last unit and by `judge`, and it stays listed in `done.json`.
  - **A prepared series** whose judge folders are off the machine is judged later. `judge` judges and reviews what waited, once the folders are there, and writes the reports again.
  - **Failures:** every judgment, review or report that did not come out is listed under `failures` in the ledger and in `done.json`.

**Chains.** A chain runs one step at a time. Each step is a one-step `suite.py chain` from the step before's result, or from its start when it delivered none. Every launch of a step starts from the state folder as it was before the step. A step without a result after its relaunches, or invalid, ends the chain.

**The verify launch.** Right after a chain's last step, the runner launches §15.2's independent `verify --full` on the chain's end state. A public chain is judged just before it; a prepared series' chain is judged later. The gate's verify launch on a fixed start follows that start.

- **What it checks:** the end state is the result of the chain's last step that delivered one, checked against that step's model, as the series judge's final drift is, whichever arm made it.
- **How it runs:** as a task of its own (`WORK/verify-tasks/UNIT`), through `suite.py run --native` with `markitect_arm.py --verify`.
- **How it counts:** it is classed and relaunched like any launch, under the label `V` with the 2.0 arm's signatures, so its technical failures count toward neither arm's void share. It stays out of the report's launches. A verify launch that stays void leaves the slot empty, which is no failure of the arm.
- **Its Result** fills the chain's `verifyFull` slot, which §15.2's table reads, whatever the verify found. A verify that did not check every item shows its state there, not "0 findings", and fails the gate's check. A chain whose steps delivered nothing has no end state, and so no verify launch.

**Stop and resume.**

- **Ending a window:** the runner ends one as soon as `WORK/STOP` exists, before the next launch or during any wait. Started again after `STOP` is removed, it goes on where it stopped, without launching again what counted. A launch, its records and its answer are saved together. A launch the runner did not live to record is recorded as it ended.
- **Records:** the ledger records each window. `WORK/runner.lock` keeps a second runner out.
- **Freeze check:** the first window records the frozen tree, the binaries, a prepared series' launch inputs and the images. Each later window checks them. With `gateLedger` in the configuration, the first window also compares them with the gate's record.
- **Hidden places:** a leading `~` in a hidden place is expanded.
- **After each window:**
  - **Reports:** one over every launch (`report/all`), and per series one over all its launches (`report/SERIES`), one over its fixed starts alone (`report/SERIES-fixed`, where the primary contrast is read) and one over its chains (`report/SERIES-chains`). A series judged later is left out of every report until all its units have run and `judge` has judged all of it, so that nothing of it is read while launches still run.
  - **Final check:** the runner scans every evidence file for the login's tokens and key shapes, and counts what is left labelled `markitect.run` or `markitect.evaluation`.
- **After the last unit:** the runner also runs the reviews the protocol names, and writes `WORK/done.json`. The gate also writes its verdict, `WORK/gate.json`, which includes a verify Result that ended `verified` or `findings` for every chain. In `counted`, the reviews cover every judged result of a fixed start in the first repetition and every chain's last judged step.

Start the runner detached (`Win32_Process` on Windows, `setsid nohup` on Linux), without a shell's `ANTHROPIC_*` or `CLAUDE_*` variables. Stop it with `WORK/STOP`, not by ending its process.

## The escape probe

`escape_probe.py [--endpoint HOST]` runs through `suite.py run --sandbox ... -- python3 -I -B /variant/escape_probe.py`. From inside the sandbox it tries to read host files (foreign mounts, known host mount paths, a `.git` directory, `AGENTS.md` or an oracle directory anywhere outside the suite's mounts), to reach GitHub and the web (resolving github.com or example.com at all, by name, by address, public resolvers by address, and GitHub through the gateway with GitHub's server name) and to reach a container engine or the host (socket files, `DOCKER_HOST`, the engine's TCP ports and common service ports on the gateway, the host names and the first address of each attached subnet). Each must fail. It also completes one TLS handshake with the allowed endpoint, without sending a request, to show that the network works. It writes every attempt to `run.json`, which the suite records under `reported`, delivers no result and exits 1 if anything escaped.

## The baseline

`claude_code.py [--claude BIN] [--credentials FILE] [--effort LEVEL]` takes the `claude` binary, the login file to copy (default: the suite's `EVALUATION_LOGIN`, else the user's Claude login) and an optional effort level. Run it only through the suite, which starts it in its own process group; in the sandbox, as `python3 -I -B /variant/claude_code.py` with `--variant-code tools/evaluation`. It copies the task's `task.md` into a clone of the start as an uncommitted `TASK.md`. It then runs `claude -p` with a fixed prompt, the evaluation's model, no user settings, plugins, CLAUDE.md, memory or MCP servers, and a copy of the login without its refresh token. It stops the session a minute before the deadline, commits what is left with a fixed identity, and writes `result.bundle`, `run.json`, the session's stream and its transcripts.

Safeguards:
- **Git:** clone, session and commit share one Git setting, with no system or user configuration, no line-ending conversion, no hooks and no credentials.
- **Capture:** the session's own `.gitignore` applies, except for Go sources, module files and Markdown, which an over-broad pattern (a binary named `library` also matches `cmd/library/`) would otherwise drop silently. This is the Phase A–C capture rule.
- **Login:** the login copy is deleted as soon as the session ends. Before bundling, every commit since the start is searched for the login. If the login appears there or in any output, nothing is delivered, the stream and the transcripts are deleted, and the run is marked quarantined.
- **Fresh sessions:** every chain step is a fresh session.

Limits:
- In host mode the session runs as the user on the host: that is separation, not a sandbox.
- The repository's own `.claude/` settings still load, as they would for any developer. This includes settings the agent wrote in an earlier step.
- On Windows in host mode, a background process that the session leaves behind after a normal exit is not stopped. On Linux and in the sandbox, the whole process group is stopped.

A real run uses the owner's Claude subscription and needs an owner-approved bounded dispatch.

### On Codex

`codex_cli.py [--codex BIN] [--credentials FILE] [--effort LEVEL]` is the same baseline on Codex, for evaluations whose model is `codex:MODEL_ID`. It reuses `claude_code.py`'s prompt, Git setting, capture and login search, and takes the `codex` binary, the login file to copy (default: the suite's `EVALUATION_LOGIN`, else the user's `~/.codex/auth.json`) and an optional effort level. It runs one `codex exec --json` session with the prompt on standard input, the evaluation's model and a private `CODEX_HOME` that holds only the login copy, with an empty refresh token. Every `OPENAI_*` and `CODEX_*` variable is cleared, and an API key is refused.

The switches are a Manager's Codex switches without the Manager tools:
- no user configuration, rules, project documents, apps, memories, image generation, browser or computer use;
- also no plugins and no web search (`web_search="disabled"`), as the Claude baseline has no plugins and is denied WebSearch and WebFetch;
- Codex's own sandbox and approvals are off, as in a Manager's session, since the evaluation sandbox is the boundary;
- Codex's native subagents stay available, as Claude Code's Agent tool does for the Claude baseline.

`run.json` keeps the Claude baseline's fields:
- `sessionId` is the thread;
- `usage` holds the completed turns' tokens, with the cached input apart;
- `sessionResult` is `completed` or `failed`, and `sessionError` holds the failed turn's error text, or the error event the session ended on. The classifier reads the same from the stream. An error event followed by more work is a recovery, such as a reconnect, and decides nothing.
- A session that started no thread failed in the CLI before the model saw the task, on its configuration or its login. It gets the status `error`, as a failure of the variant's own code does, so K's runtime signature voids it.

exec's events do not name the model, so `model` is the one the session was launched with, once a thread started, as the host's Codex adapter reports it. The session files under `CODEX_HOME/sessions` are kept as `transcripts/`.

A real run uses the owner's ChatGPT subscription and needs an owner-approved bounded dispatch.

## Self-check

```text
python -B -m unittest discover -s tools/evaluation
```

On Windows, set `TMP` and `TEMP` to a user temp directory first. The judges' self-tests need Linux and Go 1.27 or later on the path and are skipped without them: the Library M1 judge takes about a minute, the Library series judge about a minute and a half, and the Makerspace sample (a fresh build, a census step, the retired terms, an unchanged step and each kind of mutant) about two minutes. The full Makerspace self-test runs in the judge container.

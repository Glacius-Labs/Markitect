# Case playground

The case playground runs one coding agent through a staged backlog in a fresh Linux
container, once per method: **Conventional** (plain project rules) or **Markitect** (the
real product installation). A separate assessment then judges every wave, and a
comparison puts two assessed runs side by side.

This page is the one description of the playground. Two pages go deeper:
[methods/markitect/README.md](methods/markitect/README.md) (what the Markitect setup
runs) and [evaluation/readinglog2/reference/README.md](evaluation/readinglog2/reference/README.md)
(hidden reference material). Nothing on this page reaches the agent: the top level of
this folder is never staged into a run.

## Purpose

The question: for a developer working with Codex CLI or Claude Code, what does Markitect
add, good or bad, compared with strong conventional agentic coding on the same work? We
look at delivered behavior, consistency with the stated rules, regressions, how easy the
code is to change and continue, and how much work it took.

- **Conventional** gets clear project rules (`AGENTS.md` plus a short working-method
  fragment) and works with planning, tests, native subagents, review and repair. It is
  not weakened and does not imitate Markitect.
- **Markitect** gets the product installed as a user would (init, onboarding,
  runtime setup, MCP server) and works with its normal workflow, including the inner
  roles it schedules. Their cost counts. The agent, not the harness, writes the project
  model.

The rules follow the product's guidance on
[evaluating the method separately from its runtime](../../docs/measurement.md#evaluating-the-method-separately-from-its-runtime)
([DEC-010](../../docs/concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)):

- **Fairness.** Both arms get the same image, outer agent (kind, CLI version, model,
  effort), prompt, case files, time limits and container size. Time limits count agent
  time only. Each run starts from a fresh repository and agent context and resumes the
  same session from wave to wave. Nobody coaches the agent; the harness only releases
  the next wave. Check results, old runs and the other arm's results are never inputs.
- **The product supplies the method.** The harness only selects and configures what
  Markitect offers; it has no Manager, review or verify logic of its own. A product
  failure a normal user would hit is a finding, not something to work around.
- **Ground truth is fixed before any run** and never visible to an arm. Hidden checks
  test only what the public requirements and project rules already say.
- **Assessment is separate from execution:** after the run, in its own container, on
  frozen snapshots. Nothing from it reaches the agent.
- **Failures are classified, not scored.** A `harness`, `environment` or `product`
  failure is never an outcome of the method. Product failures go to the product side.
- **Provider independence.** The outer agent is Codex or Claude Code. The provider is a
  recorded factor; runs with different outer providers are never pooled.
- **What counts.** Primary: missed and unnecessary work, rule violations, contradictions,
  regressions, escalations. Cost is secondary. Reviews are evidence, not human
  acceptance.

One run per arm shows mechanisms: where an arm failed, what it cost, what the method
changed. It cannot show a general effect, a winner, statistical confidence or an
economic benefit; the waves of one project are not independent samples. The small
synthetic cases say nothing certain about real repositories or human acceptance. Keep
observed facts, likely explanations and untested ideas apart.

| Evidence | Stands for |
|---|---|
| Public checks per wave and at the end; the candidate's own tests | Outcome quality, the same for both arms |
| `markitect check` on the final state | Markitect conformance, reported apart; a valid model is not correct software |
| Setup time and every setup command | Cost of installing the method |
| Agent time, exit, timeouts, commands, MCP and subagent calls, commits on `main` | Effort per wave; unknown stays unknown, never zero |
| Tokens per wave: outer session and all recorded sessions (subagents, inner roles) | Cost; "all" is a lower bound |
| Leftover processes, stop reasons, failure class | Validity of the run |
| Hidden holdouts per wave (cases with ground truth) | Missed obligations and regressions the public checks miss |
| One review per wave by a Codex and a Claude reviewer, and their agreement | Missed and unnecessary work, rule violations, contradictions, false claims, escalations |
| Diff profile per wave (model, code, tests, docs, config/other) | Proxy for human review effort |

## First run: the provider-free smoke

The smokes run the whole pipeline with a scripted fake agent. No model call, no login.

### Prerequisites

| Need | For | Notes |
|---|---|---|
| Linux host with Docker (Linux engine) | everything | The reference platform ([DEC-013](../../docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). Your user must be allowed to run `docker`. |
| Python 3.11 or newer | everything | Standard library only, nothing to install. |
| Git and Go | Markitect arm only | The host builds a linux/amd64 `markitect` binary from a local product checkout. Go follows the product's `go.mod` (`GOTOOLCHAIN=auto`). |
| Codex login `~/.codex/auth.json` | real Codex runs, Markitect behind Claude Code, Codex reviewer | Not for the smokes. See [Logins](#logins). |
| Claude Code token file | real Claude Code runs, Claude reviewer | Not for the smokes. |

The first image build downloads the pinned base image, the Debian packages from
snapshot.debian.org and both CLIs ([Updating the image pins](#updating-the-image-pins)). It
needs network access and takes a few minutes, longer when the snapshot service is slow;
later builds use the Docker cache.

### Run it

```
cd experiments/case-playground
python3 tests/smoke_docker.py    # one run: host run --manifest examples/fake-roombook.json
python3 tests/smoke_study.py     # a whole study: both arms, assessments, comparison
```

Run them from a Git checkout whose `evaluation/` is committed and unchanged
([Pre-registration](#pre-registration)); `smoke_study.py` also needs committed judging code
and the committed image inventory ([Updating the image pins](#updating-the-image-pins)).
Each smoke works in a new folder under `$TMPDIR`
(default `/tmp`, never inside a Git checkout) and keeps it after a failure; `--keep`
keeps it after a pass. `smoke_docker.py --manifest` takes the other fakes,
`examples/fake-readinglog2.json` (six waves) and `examples/fake-claude-roombook.json`
(Claude Code stand-in). `smoke_study.py` runs
`study examples/study-fake-roombook.json --fake-reviewers` with a throwaway Codex login:
two roombook waves per arm, the Markitect arm with the binary built from this checkout at
the example's commit and the real product setup. If a product command fails, the setup
is `blocked` and the smoke fails: a product finding, or a sign that the product's CLI
changed under the setup sequence in `playground/methods.py`.

`smoke: passed`, the last line, means that every check above it is `[ok]`.
`smoke_docker.py` checks one run: exit 0; every wave ran and merged its `FAKE_S<n>.md`;
setup `ready`, class `none`, final public checks; token counts from the session records;
leftover processes killed; `host.json` complete with its pre-registration and the image
pins, `image-inventory.txt` matching its hash, and no labelled container left; the agent never opened `/out`; for `fake-claude` a throwaway
token reached the agent and is nowhere in the run folder. `smoke_study.py` checks the
study: schedule order, both assessments on the registered evaluation tree, a comparison
with matching fairness fields and host platform, a complete `study.json`, the login
carried from run to run, nowhere in the study folder, the source never written, no
login folder and no container left. Neither judges quality: the fake agent only writes
`FAKE_S<n>.md`, so its public checks fail, as expected.

### Unit tests

No Docker, no provider:
```
python3 -B -m unittest discover -s tests -t .
```

Tests that need root and the image's `agent` user, or a normal user, skip cleanly
elsewhere; for full coverage also run them in the image (adjust the tag):

```
docker run --rm --network none --mount type=bind,source="$PWD",target=/src,readonly \
  --workdir /src markitect-playground:codex-0.162.0-claude-2.1.296 \
  python3 -B -m unittest discover -s tests -t .
```

## Running real arms

A study runs everything with one command: preflight, runs, assessments, comparisons.

1. Copy `examples/study-readinglog2.json`, give it a new `id` and set `markitect.commit`
   to the product commit to test (built from this checkout unless `markitect.sourceRepo`
   names another Markitect checkout).
2. `python3 -m playground study my-study.json --preflight` prints every problem at once.
3. `python3 -m playground study my-study.json` runs it; start with `study.md` in the
   [study folder](#study-folder).

The arms of a pair run one after the other, the first arm alternating from pair to pair;
then every run whose container finished is assessed and each pair compared (A
conventional, B markitect; a fairness mismatch fails the step). A failed or stopped run
does not stop the study; it stops when the host could not run a container or the image
changed under it, and then lists the `assess` and `compare` commands that finish its
completed runs by hand. SIGTERM and SIGHUP stop it like Ctrl+C. There is no resume.

`host run` is the tool for a single run; `assess` and `compare` take any run folder:
```
python3 -m playground host run --manifest my-run.json
python3 -m playground assess --run ~/markitect-playground-runs/<id> [--fake-reviewers]
python3 -m playground compare RUN_A RUN_B
```

### Preflight

The preflight checks Python 3.11; `docker`, its daemon and a Linux engine (linux/amd64
for the Markitect arm); that no playground container runs and no other study holds
`~/.markitect-playground/study.lock` (run one study at a time); that no container has the
study's names; for the Markitect arm `git`, `go` and a checkout holding the commit; every
needed login (present, not empty, with what needs it); a new study folder outside every
Git checkout with 5 GiB free; the image pins in `container/Dockerfile` and a committed
image inventory for the study's CLI versions with the same pins
([Updating the image pins](#updating-the-image-pins)); pre-registered evaluation files;
committed judging code (`playground/` and the public checks; changed code would make every
assessment exploratory); full reviewer model ids that differ from the arms' models (with
`--fake-reviewers` a clash only warns). It only warns when the commit is behind `HEAD`, the
host is not Linux or an earlier study left login folders. Then it builds the image and the
binary once for all runs; `--preflight` stops before that and writes nothing. A failed
image build, its inventory check included, ends the study with 11 (status
`image-build-failed`); a binary that does not build, with 3.

## Components

| Path | Role |
|---|---|
| `playground/__main__.py` | Entry point: `run` (inside the run container), `host`, `assess`, `compare`, `study`. |
| `playground/manifest.py` | Loads and validates a schema-1 manifest; the only validator. |
| `playground/cases.py` | Finds the cases: every valid folder in `cases/` (see [Cases](#cases)). |
| `playground/host.py` | Host side of a run: image and Markitect binary, staging, container start and wait, safety timeout, hand-back. Also `host clean`. |
| `playground/runner.py` | Inside the container: one trajectory from agent home and setup through waves S1 to SN to freeze, final assessment and report. Counts tokens, records why a run stopped. |
| `playground/lifecycle.py` | The case repository: prepare, snapshot, advance (release the next wave), freeze. Git on the agent's repo runs as its owner, with hooks off. Manual CLI: `python3 -m playground.lifecycle`. |
| `playground/methods.py` | Installs the method: the Conventional fragment or the Markitect product setup. Reads Markitect's roles from `.markitect/runtime.yaml`. |
| `playground/codex_agent.py` | Codex as outer agent (config, command, events, session records), and the runner for every process as user `agent`, including kill-all. |
| `playground/claude_agent.py` | Claude Code as outer agent: command, MCP config, token in the environment only, redaction, stream and transcript parsing. |
| `playground/assess.py` | Public checks on a scratch copy without `.git`; the final assessment adds the candidate's own tests and `markitect check`. |
| `playground/report.py` | A run's `report.json` and `report.md`. `python3 -m playground.report <results>` rebuilds them. |
| `playground/evaluate.py` | `assess`: starts the assessment container; inside, checks, holdouts, diff profile, classification, product findings and reviews. |
| `playground/reviewers.py` | Codex and Claude reviewers: prompt, input bundle, commands, schema validation, agreement. |
| `playground/compare.py` | Side-by-side comparison of two assessed runs after a fairness check. |
| `playground/registration.py` | [Pre-registration](#pre-registration) from Git, the judging code's commit, model ids and the reviewer model check. |
| `playground/image.py` | The image pins in the Dockerfile, the image inventory and its check, the cache key of `host image-key` ([Updating the image pins](#updating-the-image-pins)). |
| `playground/outcome.py` | The [exit codes](#exit-codes) and how each command computes its own. |
| `playground/study.py` | `study`: study file, schedule, preflight and login copies; runs, assesses and compares through the functions behind `host run`, `assess` and `compare`. |
| `container/Dockerfile` | The image and the one place for its pins: `node:22-bookworm-slim` by digest, Debian packages (Python, Git, bubblewrap, ...) from a snapshot.debian.org time, Codex CLI and Claude Code at exact versions, user `agent` (uid 1000); its last step writes the image inventory. |
| `container/inventory/` | The committed inventory per CLI pair, `codex-<v>-claude-<v>.txt`, that every build is compared with. |
| `cases/` | `task-prompt.txt` (the one prompt); `common/` (`AGENTS.md`, `QUALITY.md`, the public checks' driver `checks/acceptance.py`), seeded into every repository; one folder per case with `README.md`, `BACKLOG.md`, `STATIONS.json`, its public checks `checks/<case>.py` and, for brownfield cases, starting code. |
| `methods/` | `conventional/AGENTS.fragment.md`, appended to `AGENTS.md`; `markitect/README.md`, notes for people. |
| `evaluation/` | Hidden: `config.json`, reviewer prompt and schema in `common/`, ground truth, holdouts, mutants, reference and `validate.py` in `readinglog2/`. Never staged into a run. |
| `examples/` | Study files: real (`study-readinglog2.json`) and fake (`study-fake-roombook.json`). Manifests: real arms (`conventional-readinglog.json`, `markitect-readinglog.json`) and fakes (`fake-roombook.json`, `fake-readinglog2.json`, `fake-claude-roombook.json`). |
| `tests/` | Unit tests, fake stand-ins for Codex, Claude Code and the reviewers, and the Docker smokes `smoke_docker.py` and `smoke_study.py` (not picked up by unittest). |

## Data flow

```
HOST  python3 -m playground study S.json
  |  validate; preflight; image and binary built once; logins copied once
  |  each run in schedule order: host run (below) with its own login copies,
  |     its Codex login copied back out after the container stops
  v  assess each finished run, compare each pair (below) -> <out>/study.md, study.json

HOST  python3 -m playground host run --manifest M.json
  |  validate manifest; check that login files exist (never read them)
  |  pre-registration: evaluation/ committed -> commit and Git tree in host.json
  |  image pins: container/Dockerfile names a base digest and a Debian snapshot
  |  docker build   -> image markitect-playground:codex-<v>-claude-<v>
  |  inventory      -> <out>/image-inventory.txt, compared with container/inventory/
  |  Markitect arm  -> go build of markitect.commit
  |  stage <out>/inputs: code, cases/common, cases/<case>, prompt,
  |                      Conventional fragment or binary, fake agent, normalized manifest
  v
RUN CONTAINER mpg-<id>    /in = inputs (read-only)    /out = <out>/results
  |  agent home: copy of the Codex login; Claude token only in claude's environment
  |  prepare: fresh Git repo /work/<case>, seed commit, wave S1 released
  |  setup: Conventional fragment | Markitect init, onboard, setup, MCP server
  |  per wave S1..SN:
  |     agent starts (S1) or resumes (S2+) its session with the same prompt
  |     kill every process of user agent
  |     snapshot          -> audit/snapshot-S<n>
  |     public checks     -> stations/S<n>/checks.json   (hidden from the agent)
  |     release next wave (rewrite .study/station.json)
  |  freeze; final checks, own tests, markitect check; report
  v
HOST  remove container; hand results/ back to you; write host.json

HOST  python3 -m playground assess --run <out>
  v
ASSESSMENT CONTAINER mpg-assess-<id> (same image)
  |  /assess/run = run folder (ro)  /assess/in = code + the run's registered evaluation tree (ro)
  |  per wave, on a copy of that wave's main:
  |     public checks again, holdouts, diff profile, classification, product findings
  |  per wave: Codex and Claude reviews (holdouts removed first)
  v
<out>/assessment/report.md, report.json, product-findings.md

HOST  python3 -m playground compare RUN_A RUN_B  -> fairness check -> comparison .md
```

What the agent sees:

- It works in `/work/<case>` as user `agent` and can read `/in`, including the public
  checks in the seed (`checks/acceptance.py`, `checks/<case>.py`), which it can run.
- `/out` is root-only while the run lasts: no check results, snapshots or reports. It
  never sees the other case, the other arm's method files, `evaluation/` or this page.
- Codex runs with `--dangerously-bypass-approvals-and-sandbox`, Claude Code with
  `--dangerously-skip-permissions`; the container is the isolation boundary. No host
  home, settings, hooks or MCP servers are used. With Claude Code, both arms first get
  the same one-line `CLAUDE.md` (`@AGENTS.md`) when the repository has none.

Each wave's timeout is `stationSeconds` or the rest of `totalSeconds`, whichever is
smaller. A run stops early only when a needed login is missing, the method setup is
`blocked`, the agent never produced a session id, could not start or exited without any
work, a snapshot or wave release failed, or the agent time is used up; a wave timeout or
a failing agent that did work is recorded, and the run goes on. Freeze and report always
run, the final assessment when at least one wave ran. The host kills the container after
`totalSeconds` plus the runner's worst-case own work, as a safety net.

## Logins

- **Codex:** `~/.codex/auth.json` from `codex login`, or `--codex-auth PATH`: for agent
  kind `codex`, for kind `claude` in the Markitect arm (inner roles run on Codex) and for
  the Codex reviewer. The fake agent gets one only when `--codex-auth` is given.
- **Claude Code:** a token from `claude setup-token`, alone in a file. `study` and
  `assess` read `~/.markitect-playground/claude-token` by default; `host run` needs
  `--claude-token PATH` for kind `claude`. Your own `~/.claude` login is never used.
- On the host, login files are only copied and stat'ed (never read, printed or hashed)
  and mounted read-only. Inside, the Codex login goes into a fresh agent or reviewer
  home (an assessment's reviewers share a working copy that keeps a refresh); the Claude
  token only into the claude process environment, redacted if the agent prints it.
- **Login copies in a study.** Each login is copied once into a private folder
  `~/.markitect-playground/logins/<id>-<random>/` (0700); every run and assessment gets
  its own 0600 copy. After a container stops, its Codex login is streamed out
  (`docker cp CONTAINER:PATH -`); exactly one regular file of 1 B to 64 KiB becomes the
  next step's copy, anything else is rejected unread. Your source login is never
  written; when a step reported a refresh, the study ends with "Codex refreshed its
  login during the study; your <source> may be used up: run `codex login` before the
  next run." The folder is removed when the study ends, also after an interrupt.

## Parameters

### Manifest (schema 1)

Unknown fields are rejected at every level.

| Field | Type, allowed values | Default | Meaning |
|---|---|---|---|
| `schema` | `1` | required | Manifest schema. |
| `id` | `[a-z0-9][a-z0-9-]{0,62}` | required | Names the container `mpg-<id>`, the default run folder and the assessment container `mpg-assess-<id>`. |
| `case` | a folder in `cases/` ([Cases](#cases)) | required | Folder `cases/<case>/`. |
| `stations` | positive integer, at most the case's waves | all waves | The run gets the first N waves only (waves build on each other). A fairness field. |
| `method` | `conventional`, `markitect` | required | The arm. |
| `agent.kind` | `codex`, `claude`, `fake` (for Codex), `fake-claude` (for Claude Code) | required | Outer agent. |
| `agent.codexVersion` | `[0-9][0-9A-Za-z.-]{0,55}` | required | `@openai/codex` in the image. Always needed: Codex also runs Markitect's inner roles. |
| `agent.claudeVersion` | same pattern | `2.1.296`; required for Claude kinds | `@anthropic-ai/claude-code` in the image. |
| `agent.model` | `[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}`, a full model id | required | Outer model; for Codex also the subagent default. An alias is refused ([Reviewers](#reviewers)). |
| `agent.effort` | `[a-z]{1,32}` | required | Outer reasoning effort. |
| `agent.maxSubagents` | positive integer | required | Codex `agents.max_concurrent_threads_per_session`. Does not limit Claude Code (recorded only). |
| `limits.stationSeconds` | positive integer | required | Agent time limit per wave. |
| `limits.totalSeconds` | positive integer | required | Agent time budget for all waves. Also sets the host's safety timeout and Markitect's cost cap (x 50,000 tokens, at most 10^12). |
| `container.cpus` | positive number | `4` | `docker run --cpus`. |
| `container.memory` | `[1-9][0-9]*[bkmg]?` | `8g` | `docker run --memory`. |
| `container.pidsLimit` | positive integer | `2048` | `docker run --pids-limit`. |
| `markitect` | object | required for `markitect`, forbidden otherwise | The product to install. |
| `markitect.sourceRepo` | non-empty path | the checkout holding the playground, if its `go.mod` declares `github.com/Glacius-Labs/Markitect` | Local Markitect checkout to build from. A relative path or `~` counts from the manifest's folder; recorded as an absolute path. |
| `markitect.commit` | 7 to 40 lowercase hex | required | Commit to build; the full hash is recorded. |
| `markitect.innerModel` | like `agent.model` | `agent.model`; required for Claude kinds | Model of Markitect's inner roles. |
| `markitect.innerEffort` | like `agent.effort` | `agent.effort`; required for Claude kinds | Effort of Markitect's inner roles. |

### Study file (schema 1)

Unknown fields are rejected. The study expands into one manifest per run, validated like
any manifest and saved normalized in `manifests/`.

| Field | Type, allowed values | Default | Meaning |
|---|---|---|---|
| `schema` | `1` | required | Study file schema. |
| `id` | like a manifest `id`; `<id>-p<pairs>-conv` at most 63 characters | required | Run ids `<id>-p<p>-conv` and `<id>-p<p>-mkt`; the default study folder. |
| `case`, `stations`, `agent`, `limits`, `container`, `markitect` | as in the manifest | as there; `container` is required | Shared by every run. `markitect` only with the Markitect arm; a relative `sourceRepo` counts from the study file's folder. |
| `arms` | list of `conventional`, `markitect` | required | Both arms form pairs; one arm repeats. |
| `firstArm` | one of `arms` | required | Goes first in odd pairs; the other arm goes first in even pairs. |
| `pairs` | positive integer | `1` | Number of pairs (with one arm: of runs). |
| `reviewers` | list of `codex`, `claude`, or `[]` | required | Reviewers of every assessment; their models are pre-registered in `evaluation/config.json` and must differ from the arms' models. |

### `study`

`python3 -m playground study STUDY.json [options]`

| Option | Default | Meaning |
|---|---|---|
| `--out DIR` | `~/markitect-playground-runs/<id>` | Study folder; must not exist and must not be inside a Git checkout. |
| `--codex-auth PATH`, `--claude-token PATH` | `~/.codex/auth.json`, `~/.markitect-playground/claude-token` | Login sources, used only where needed ([Logins](#logins)). |
| `--fake-reviewers`, `--keep-containers` | off | As for `assess`; keep every container. |
| `--preflight` | off | Run the cheap checks only; build and write nothing. |

### `host run` and `host clean`

| Option | Default | Meaning |
|---|---|---|
| `--manifest PATH` | required | The manifest. |
| `--out DIR` | `~/markitect-playground-runs/<id>` | Run folder. Must not exist (except one whose attempt failed before a container started; it is replaced) and must not be inside a Git checkout. |
| `--codex-auth PATH` | `~/.codex/auth.json` | Codex login, used only when needed. |
| `--claude-token PATH` | none | Required for kind `claude`, optional for `fake-claude`, ignored otherwise. |
| `--keep-container` | off | Keep the container after it ends. |
| `--exploratory` | off | Run although `evaluation/` has uncommitted changes; records `rules.exploratory`, and `compare` refuses the run. |

`python3 -m playground host clean` removes every stopped playground container (label
`markitect-playground=1`), assessment containers included. `python3 -m playground host
image-key` prints the image's cache key ([Updating the image pins](#updating-the-image-pins)).

### `assess`

| Option | Default | Meaning |
|---|---|---|
| `--run DIR` | required | Run folder with `host.json` and `results/`. |
| `--reviewers LIST` | `codex,claude` | One, both or `none`; each must be in the registered `config.json`. |
| `--codex-auth PATH` | `~/.codex/auth.json` | Codex reviewer login. |
| `--claude-token PATH` | `~/.markitect-playground/claude-token` | Claude reviewer token. |
| `--image IMAGE` | the run's image | Image of the assessment container. |
| `--force` | off | Replace an existing `assessment/` folder. |
| `--keep-container` | off | Keep the container after it ends. |
| `--fake-reviewers` | off | Fake reviewer CLIs, throwaway credentials, no model call. Not with `--codex-auth` or `--claude-token`. A reviewer model clash only warns. |
| `--exploratory` | off | Judge with the working tree's evaluation files, also a run without a registration or with a reviewer model clash; `compare` refuses the assessment. |

### `compare`

| Option | Default | Meaning |
|---|---|---|
| `RUN_A RUN_B` | required | Two assessed run folders. |
| `--allow-mismatch` | off | Compare although fairness fields differ; the differences are printed at the top. |
| `--out FILE` | `compare-<A id>-vs-<B id>.md` in the folder holding RUN_A | Output file. |

### `evaluation/config.json`

| Key | Today | Default when missing | Meaning |
|---|---|---|---|
| `schema` | `1` | not checked | Config schema. |
| `reviewers.codex.model` | `gpt-6.1-sol` | required | Codex reviewer model, a full model id. A reviewer exists only if listed. Every model must differ from the arms' models ([Reviewers](#reviewers)). |
| `reviewers.claude.model` | `claude-opus-5-5` | required | Claude reviewer model. |
| `reviewers.<name>.effort` | `high` | flag left out | Reasoning effort. |
| `reviewers.<name>.timeoutSeconds` | `2700` | `2700` | Hard limit per review. |
| `promptMaxBytes` | `100000` | `100000` | Prompt cap; long parts are cut with a pointer to the full file. |
| `holdoutTimeoutSeconds` | `600` | `600` | Hard limit for the holdouts per wave. |

### Exit codes

`host run`, `assess`, `compare` and `study` share one table; their `--help` prints it.

| Code | Meaning |
|---|---|
| 0 | Completed. `host run`: every wave ran, class `none`; `assess`, `compare`: written; `study`: every step 0. |
| 1 | Method outcome: the run stopped early with class `none` (agent time used up). |
| 2 | Invalid input or refused by a rule: manifest, study file, options, folder, pre-registration, reviewer models, fairness mismatch. |
| 3 | `study` preflight failed: a cheap check (image pins and committed image inventory included) or the Markitect binary build. |
| 10 | Harness failure: runner error, failed snapshot or wave release, `assess-error.txt`, `study-error.txt`, an unexpected error of the command, a container killed (exit 137) that Docker does not report `OOMKilled`. |
| 11 | Environment failure: logins, no session id, host status `setup-failed`, `start-failed` or `wait-failed`, a failed image build or image inventory check (in `study` too), a container killed (exit 137) that Docker reports `OOMKilled`, image changed. |
| 12 | Product failure: the Markitect binary does not build (compile errors in its own sources; any other `go build` failure is 11), setup blocked by the product, `markitect check` could not run. |
| 124, 130 | Host safety timeout; interrupted. |

`host run` maps its host status and the run report's [class](#outcome-classes); the runner's
own 0, 1 or 2 stays `containerExitCode` in `host.json`, which always records the code the
process exits with, and `oomKilled`, Docker's `State.OOMKilled` read before the container
is removed. Failed holdouts or reviews never fail `assess`. `study` gives its
worst step code (130, 124, 10, 11, 12, 1, 0 in that order); a step refused inside a study
(2) counts as 10. A study whose worst step is 1 has status `method-stopped`: the method
stopped, the harness did not fail.

## Results

### Study folder

| Path | Content |
|---|---|
| `study.md`, `study.json` | **Start here.** Status, exit code and stop reason; resolved parameters; login paths, never contents; versions (playground commit and dirty flag, Python, host platform, Docker server, OS and architecture, Go, the image record as in `host.json`, binary SHA-256); preflight results; per step its times, status, exit code and login handling; after a stop, the commands that finish by hand. Rewritten after every step. |
| `study-file.json`, `manifests/`, `preflight/` | The study file as given; each run's normalized manifest; the printed checks, the image build log, `image-inventory.txt` and the binary. |
| `runs/<run id>/`, `comparisons/p<k>.md` | Each run folder (below) with its `assessment/`; pair k compared, A conventional, B markitect. |

### Run folder

| Path | Content |
|---|---|
| `results/report.md`, `report.json` | **Start here.** One row per wave, setup, final assessment, classification, fairness fields, versions. |
| `host.json`, `image-build.log`, `image-inventory.txt`, `container.log`, `inputs/` | Host record (normalized manifest, status, host platform, `preRegistration`, `rules`, `image` with `tag`, `id`, `base`, `snapshot`, `dockerfileSha256` and `inventorySha256`, Docker version, Markitect build, times, `docker run` arguments, container and mapped exit code, `oomKilled`, `handBack`); build output, the image's inventory and container output; what was mounted at `/in`. |
| `results/runner.json`, `runner-error.txt` | Status, stop reason and category, versions, login rewrite, token redactions; a traceback after a runner error. |
| `results/setup/`, `stations/S<n>/`, `final/` | Setup command output; per wave the agent's events, output, `agent.json`, `checks.json` and work outside the repository; the last wave's public checks, own tests and Markitect conformance. |
| `results/audit/`, `evidence/` | Seed and wave plan (`run.json`), per-wave snapshots, the final freeze; session records, transcripts and Markitect's cache. |

### Assessment folder

| Path under `<run>/assessment/` | Content |
|---|---|
| `report.md`, `report.json`, `product-findings.md` | Per wave: public checks, holdouts, diff profile, reviewer findings and agreement, obligations, escalations, class; evaluation source, Git tree, commit and file hashes, the judging code's commit and dirty flag (`evaluation.code`), `rules`, reviewer models, image. Product findings: setup steps, failed MCP calls and product commands, with exact error text. |
| `stations/S<n>/` | Checks, holdouts, `wave.diff`, what the reviewers got and their answers. |
| `host.json`, `container.log`, `inputs/`, `assess-error.txt` | Host record, container output, staged code and evaluation files; a traceback when the assessment failed. |

### Outcome classes

The run report classifies the run; the assessment classifies the run and each wave
(a wave only when it failed: exit, timeout, no session, harness error).

| Class | Meaning | Typical causes |
|---|---|---|
| `harness` | Our code | Runner error, failed snapshot or wave release, harness error files, host timeout or interruption, a container killed (exit 137) that Docker does not report `OOMKilled`. |
| `environment` | Docker, network, provider, logins | Missing login, no session id, auth or rate-limit errors, a wave without any agent activity, a failed container start or wait, a container Docker reports `OOMKilled`. |
| `product` | Markitect | Setup blocked by a product command, an MCP call that never finished, `markitect check` that could not run. |
| `none` | No infrastructure cause | Completed, or the time budget was used up. |

With several causes, `harness` wins over `environment`, which wins over `product`; all are
listed. A class other than `none` is a limit of the measurement or a product finding,
never a result of the method.

Holdouts read `passed/total`, plus `+N not judged` for status ERROR (the holdout could
not judge, for example after its deadline), counted apart, never as a candidate failure.

### Linux ownership hand-back

Containers write as root and keep snapshots root-only. After a container has stopped,
a short root container in the same image gives `results/` (or `assessment/`) the owner
of its parent folder, without following links. `handBack` in `host.json` says `done`,
`not-needed` (Windows, root, or an engine that maps root to you), `skipped: ...` (Docker
could not confirm the stop) or `failed: ...`; the last two leave it owned by root.

## Evaluation layer

### Cases

| Case | Waves | Items | Kind |
|---|---|---|---|
| `roombook` | 4 | R01 to R12 | Greenfield room reservation CLI, no starting code. Smoke tests. |
| `readinglog` | 4 | B01 to B12 | Brownfield reading log with code and tests. The shakedown pair. |
| `readinglog2` | 6 | B01 to B15 | Brownfield change series with hidden evaluation files. |

To add a case, add a folder `cases/<name>/` (name like a manifest `id`, not `common` or
`acceptance`) with `README.md`, `BACKLOG.md`, a `STATIONS.json` whose `case` is `<name>`,
its public checks `checks/<name>.py` (a function `checks(ctx)`, run by the shared
`cases/common/checks/acceptance.py`) and any starting code; `evaluation/<name>/` is
optional. Any other folder in `cases/` is an error that names the folder and the rule.

S3 is a team wave in every case (`requiresTeam`). `readinglog2` starts with three
cross-cutting rules in its public README (R1 audit log, R2 error contract, R3 text
normalization) that later items do not repeat. S1 to S3 add features (S3 interacting
ones), S4 refactors storage, S5 changes an earlier decision (pages become optional), S6
renames a domain term. The backlog for S4 to S6 names the decision, not the affected
commands; finding them is the point.

### Hidden evaluation files

`evaluation/<case>/` exists only for `readinglog2`. Nothing in `evaluation/` is staged
into a run.

| File | Use |
|---|---|
| `ground-truth.json` | Rules and, per wave and item: obligations, areas (behavior and docs, not file names), rule expectations, must-not-change statements. The wave's slice is the reviewers' checklist. |
| `holdout.py` | Hidden checks run by the assessment; never shown to reviewers. |
| `mutants/*.json` | Reference waves with exactly one obligation broken; for `validate.py`. Never staged. |
| `reference/S1` to `S6` | Hidden reference implementation per wave; see its [README](evaluation/readinglog2/reference/README.md). Never staged. |
| `validate.py` | `python3 -I -B evaluation/readinglog2/validate.py [--jobs N] [--only TEXT ...]`. Checks the ground truth against `STATIONS.json`; every reference wave passes all public checks and holdouts, twice alike; the seed baseline passes the baseline and cross-cutting holdouts at S1; every mutant fails exactly the holdouts it declares. |

### Pre-registration

Commit ground truth, holdouts, reviewer prompt, schema and config before the runs they
judge. `host run` (before the build) and the `study` preflight refuse an `evaluation/`
with uncommitted or untracked changes, or a playground outside a Git checkout, and list
the paths; the run records `preRegistration` (`status`, `commit`, `evaluationTree`, the
Git tree of `evaluation/`, and `recordedAt`) in `host.json`. `assess` judges with exactly
that tree, read from Git (`git archive`), never from the working tree, including the
reviewer models in its `config.json`; the report records `evaluation.source: registered`,
the tree, the commit and each file's SHA-256. A run without a registration (older runs) is
refused. `--exploratory` on `host run` or `assess` (never `study`) uses the working tree
and records `rules.exploratory`; `compare` refuses such runs.

The code that judges also counts: `assess` records the commit of `playground/`,
`cases/common/checks` and the case's checks, and whether they have local changes, as
`evaluation.code`. Changed (or unknown) judging code makes the assessment exploratory, so
`compare` refuses it; the `study` preflight fails on it.

### Holdouts

`python3 -I -B holdout.py --repo DIR --station N [--deadline SECONDS]` prints
`{"station": N, "checks": [{"id", "status", "item", "rule", "detail", "source"}]}` and
exits 0 even when checks fail. Station N runs every holdout released up to N, so earlier
waves are checked again for regressions, each on fresh temporary data. Each holdout names
its item, its rule and, in `source`, the public sentence it derives from; what the public
text leaves open is not tested. PASS and FAIL judge the candidate; ERROR means the
holdout could not judge, as for checks not yet run when `--deadline` passes. The
assessment runs it as the unprivileged user on a copy of the wave's `main`, with
`--deadline` up to 90 s before `holdoutTimeoutSeconds`.

### Reviewers

- Every selected reviewer reviews every wave after all checks and holdouts, as user
  `agent` in the input bundle folder (so no `AGENTS.md` or `CLAUDE.md` is loaded as
  instructions), and reads the snapshot copy by path. The holdouts are removed first and
  `/assess` is root-only: a reviewer never sees holdouts, check results or the other
  reviewer's answer.
- **Codex:** `codex exec --json --sandbox read-only --output-schema ...` in a fresh
  `CODEX_HOME` holding only the login. **Claude:** `claude -p --output-format json
  --json-schema ...`, tools `Read`, `Grep`, `Glob` only, no settings
  (`--setting-sources ""`, `--strict-mcp-config`), a fresh `CLAUDE_CONFIG_DIR`, the token
  only in its environment. Model and effort come from the registered `config.json`.
- **Full model ids:** manifests, study files and `evaluation/config.json` take full model
  ids only. An alias is refused: Claude Code's `opus`, `sonnet`, `haiku`, `fable`,
  `opusplan`, `default` and `best`, an id ending in `latest`, or one without any version
  digit. A registered `config.json` with an alias fails `assess` and the `study` preflight;
  the fix is a commit.
- **Independence:** no reviewer model may be an arm's model: `agent.model`,
  `markitect.innerModel` and, in `assess`, every Markitect role's model in the run report.
  They are compared normalized: casefolded, without a context suffix such as `[1m]`, a
  provider prefix (`anthropic/`, `openai/`, Bedrock's `us.anthropic.`), Bedrock's `-v1:0`
  or a trailing date (`-20251001`, `@20251001`, `-2025-10-01`); the provider is not
  checked. The `study` preflight fails on a clash; `assess` refuses it (exit 2) unless
  `--reviewers` leaves the clashing reviewer out or `--exploratory` records
  `reviewersIndependent: false`. With `--fake-reviewers` a clash only warns.
- **Prompt:** the fixed, versioned [reviewer-prompt.md](evaluation/common/reviewer-prompt.md),
  the same for every arm and provider, plus the wave's inputs: released item texts,
  earlier items, project rules (case README, `AGENTS.md`, `QUALITY.md`), the ground-truth
  slice, the agent's final message, the wave diff and the full backlog.
- **Answer:** JSON checked against [reviewer-schema.json](evaluation/common/reviewer-schema.json):
  findings (category, severity, item, rule, evidence, detail), obligations covered and
  total, notes. Categories: `missed_obligation`, `unnecessary_change`, `rule_violation`,
  `contradiction` (for Markitect also model against code, docs or tests), `regression`,
  `false_claim`, `escalation_needed`, `escalation_unneeded`.
- A crash, timeout or invalid answer is recorded as `error` or `invalid` and never ends
  the assessment. Agreement counts findings with the same item and category.
- **Blinding limit:** the prompt has no arm label, but the repository can reveal the
  method (`.markitect/`, Markitect guidance in `AGENTS.md`); the reviews are not blind.

### Compare's fairness check

`compare` refuses two runs unless these match, or `--allow-mismatch` is given: case and
outer provider; the run report's fairness fields (stations, host platform, Codex and
Claude Code versions, image ID, model, effort, subagent limit, time limits, container
size); the SHA-256 of every evaluation file; each reviewer's model, effort and CLI
version. Both assessments must also have `rules.preRegistered` and
`rules.reviewersIndependent` true (missing counts as false). The comparison and its output
name the matched host platform. It does not compare the Markitect commit, the inner model
or the evaluation commit; check those yourself.

A different host platform is a fairness mismatch, and so is a platform that a run does
not record (older runs), even when both lack it. This is stricter than `host run` and
`study`, which only warn on a non-Linux host: never pool runs from different platforms
([DEC-013](../../docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground));
`--allow-mismatch` is for looking, not for study results.

## Caveats

- **Login refresh and trust.** Codex may refresh its token inside a container and so
  use up the login it was copied from. A study hands the refreshed login to its next
  step; separate `host run` and `assess` commands do not: run `codex login` before each,
  or give each its own file with `--codex-auth`. Agents run with full permissions inside
  their containers and hold the login anyway. A login one step hands on can be tampered
  with; that shows up as an environment failure of the next step, never as a method
  result, and nothing a container hands back is written to your source login.
- **Never commit run outputs.** Keep them outside the repository. `host run` and `study`
  refuse a folder inside a Git checkout, and `runs/` is ignored as a safety net.
- **One sandbox relaxation.** Every run and assessment container runs with
  `--security-opt seccomp=unconfined`: Codex's bubblewrap sandbox, used by Markitect's
  inner roles, needs user namespaces. Both arms get it; no `--privileged`, no added
  capabilities, no egress filtering.
- **Same image for both arms.** The base image, the Debian packages and both CLIs are
  pinned, and the inventory check fails a build that installed anything else. A study runs
  every step on the one image it built, by its ID, and a warm-cache rebuild keeps that ID.
  A cold build installs the same content but gets a new image ID (layer timestamps), and
  `compare` then refuses pairs built apart. Cold builds depend on snapshot.debian.org.
- **Unequal concurrency.** The subagent limit applies to Codex sessions. Markitect's inner
  roles are extra sessions, so total concurrency is not equal by construction.
- **Old run folders.** Runs from the earlier Codex-only image need
  `--image markitect-playground:codex-0.162.0-claude-2.1.296` for the Claude reviewer.
- **Product limits.** The Markitect setup accepts only what the pinned product allows,
  such as certain models; see [methods/markitect/README.md](methods/markitect/README.md).
- **Fixed in the code.** Step timeouts (public checks 600 s, own tests 900 s, product
  steps 600 s, Git steps 120 s), the Markitect binary for linux/amd64 only, and the
  study's 5 GiB free-disk minimum and 64 KiB cap for a copied-out login.

## Changing the playground

- Standard library only, Python 3.11 or newer; the same code runs on the host and in
  the container. Pass `encoding="utf-8"` explicitly and call `subprocess` with argument
  lists, never `shell=True`.
- Everything a run needs comes from the manifest and command-line options: no dates,
  hashes or user paths in the source.
- Unit tests run without Docker or a provider; tests that need root or the `agent` user
  skip cleanly elsewhere. The Docker smokes cover the end-to-end contract.
- Evaluation files are pre-registered and hashed: `ground-truth.json`, `holdout.py`,
  the reviewer prompt and schema, and `config.json`. Commit any edit before the next run;
  even a comment changes the hash, and `compare` then refuses runs assessed before and
  after it.

### Updating the image pins

`container/Dockerfile` is the one place for the pins; nothing else repeats them:

- `ARG BASE_IMAGE=node:22-bookworm-slim@sha256:<index digest>`, the base by the digest of
  its multi-platform index. The host refuses a base without `@sha256:`.
- `ARG DEBIAN_SNAPSHOT=<YYYYMMDDTHHMMSSZ>`: every Debian package (debian and
  debian-security) comes from snapshot.debian.org at that time, with `Check-Valid-Until`
  off. apt retries every download (`Acquire::Retries`) and the step runs at most three
  times. The host passes `SOURCE_DATE_EPOCH` from it as a build argument.
- `CLAUDE_VERSION` and the manifest's `codexVersion`: exact npm versions.

The last build step writes `/usr/share/markitect-playground/inventory.txt`: base, snapshot,
Node version, every installed Debian package (`name version arch`) and every package in
both CLIs' installed trees (`name@version`). After each build the host reads it with one
`docker run --rm --network none`, saves it as `image-inventory.txt` (in the run folder; a
study's in `preflight/` and every run folder) and records `image: {tag, id, base,
snapshot, dockerfileSha256, inventorySha256}` in `host.json` and `study.json`. The build
fails, exit 11, when an npm entry is not an exact version or when
`container/inventory/codex-<v>-claude-<v>.txt` exists and differs; without that file
`host run` only warns, and the `study` preflight fails. The committed inventory is for a
linux/amd64 engine, the reference; another architecture installs other packages.

To move a pin, or to add the inventory for new CLI versions:

1. `docker buildx imagetools inspect node:22-bookworm-slim` prints the index `Digest:` (it
   reads registry metadata only); `--format '{{json .Image}}'` shows each platform's
   `created` time.
2. Choose a snapshot time not older than that from
   `https://snapshot.debian.org/archive/debian/?year=YYYY&month=MM`, and check that
   `http://snapshot.debian.org/archive/debian/<time>/dists/bookworm/Release` and
   `http://snapshot.debian.org/archive/debian-security/<time>/dists/bookworm-security/Release`
   answer.
3. Update the two `ARG` lines and remove the committed inventory for the CLI versions you
   build (a build would fail on the old one).
4. Build once: `python3 tests/smoke_docker.py --keep` (or `host run` with any manifest of
   those versions) warns that no inventory is committed and keeps the run folder.
5. Review `<run folder>/image-inventory.txt`, copy it to
   `container/inventory/codex-<v>-claude-<v>.txt` and commit it together with the
   Dockerfile.

`python3 -m playground host image-key` prints a cache key from the pinned base digest, the
snapshot time, every committed inventory file and the Dockerfile's SHA-256; it changes
when any of them does, so a CI job can key its Docker build cache on it.

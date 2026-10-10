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
- **Markitect** gets the product installed as a user would (project init, onboarding,
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
| `markitect project check` on the final state | Markitect conformance, reported apart; a valid model is not correct software |
| Setup time and every setup command | Cost of installing the method |
| Agent time, exit, timeouts, commands, MCP and subagent calls, commits on `main` | Effort per wave; unknown stays unknown, never zero |
| Tokens per wave: outer session and all recorded sessions (subagents, inner roles) | Cost; "all" is a lower bound |
| Leftover processes, stop reasons, failure class | Validity of the run |
| Hidden holdouts per wave (cases with ground truth) | Missed obligations and regressions the public checks miss |
| One review per wave by a Codex and a Claude reviewer, and their agreement | Missed and unnecessary work, rule violations, contradictions, false claims, escalations |
| Diff profile per wave (model, code, tests, docs, config/other) | Proxy for human review effort |

## First run: the provider-free smoke

The smoke runs the whole pipeline with a scripted fake agent. No model call, no login.

### Prerequisites

| Need | For | Notes |
|---|---|---|
| Linux host with Docker (Linux engine) | everything | The reference platform ([DEC-013](../../docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). Your user must be allowed to run `docker`. |
| Python 3.11 or newer | everything | Standard library only, nothing to install. |
| Git and Go | Markitect arm only | The host builds a linux/amd64 `markitect` binary from a local product checkout. Go follows the product's `go.mod` (`GOTOOLCHAIN=auto`). |
| Codex login `~/.codex/auth.json` | real Codex runs, Markitect behind Claude Code, Codex reviewer | Not for the smoke. See [Logins](#logins). |
| Claude Code token file | real Claude Code runs, Claude reviewer | Not for the smoke. |

The first image build downloads the base image, Debian packages and both CLIs. It needs
network access and takes a few minutes; later builds use the Docker cache.

### Run it

```
cd experiments/case-playground
python3 tests/smoke_docker.py
```

This runs `python3 -m playground host run --manifest examples/fake-roombook.json` into a
new folder under `$TMPDIR` (default `/tmp`) and checks the result. Set `TMPDIR` to put
it elsewhere, but not inside a Git checkout. Other fake manifests:

```
python3 tests/smoke_docker.py --manifest examples/fake-readinglog2.json      # six waves
python3 tests/smoke_docker.py --manifest examples/fake-claude-roombook.json  # Claude Code stand-in
```

`--keep` keeps the run folder after a pass; a failed smoke always keeps it and prints
its path.

### What "smoke: passed" means

The last line reads `smoke: passed` when every check above it is `[ok]`:

- the host exited 0, every planned wave ran and added commits to `main`, and a captured
  `main` holds `FAKE_S1.md` to `FAKE_S<n>.md`;
- the report exists, the setup is `ready`, the run is classified `none`, and the final
  assessment ran the public checks of the last wave;
- token counts from the session records match the fake's, and leftover agent processes
  were killed after every wave;
- `host.json` says `completed` and names the host platform, the image tag names both CLI
  versions, and no labelled container is left;
- `fake`: the agent could not open the results folder in any wave;
- `fake-claude`: a throwaway token reached the agent, was redacted where the fake printed
  it and is nowhere in the run folder; the `CLAUDE.md` router was added;
- Markitect: `markitect project check` ran at the end, the roles were read from
  `.markitect/runtime.yaml`, and `host.json` holds the resolved `sourceRepo` and commit.

It says nothing about quality. The fake agent only writes `FAKE_S<n>.md`, so the public
checks in its report fail. That is expected.

### Smoke the Markitect arm

Save a manifest like this outside the checkout, for example as
`~/fake-markitect-roombook.json`, with `commit` set to a commit of this checkout
(`git rev-parse --short HEAD`), from which the binary is built. Two waves are enough:

```json
{
  "schema": 1,
  "id": "fake-markitect-roombook-001",
  "case": "roombook",
  "stations": 2,
  "method": "markitect",
  "agent": {"kind": "fake", "codexVersion": "0.162.0", "model": "gpt-6-luna",
            "effort": "high", "maxSubagents": 3},
  "limits": {"stationSeconds": 120, "totalSeconds": 600},
  "container": {"cpus": 2, "memory": "2g", "pidsLimit": 512},
  "markitect": {"commit": "3adf1d2f"}
}
```

```
python3 tests/smoke_docker.py --manifest ~/fake-markitect-roombook.json
```

It builds the binary and runs the real product setup in the container; no login is
needed. If a product command fails, the setup is `blocked` and the smoke fails: a
product finding, or a sign that the product's CLI changed under the setup sequence in
`playground/methods.py`.

### Unit tests

No Docker, no provider:

```
python3 -B -m unittest discover -s tests -t .
```

Some tests run only as root with the image's `agent` user, others only as a normal user;
each skips cleanly elsewhere. For full coverage also run them in the image (any run
above builds it; adjust the tag to your versions):

```
docker run --rm --network none --mount type=bind,source="$PWD",target=/src,readonly \
  --workdir /src markitect-playground:codex-0.162.0-claude-2.1.296 \
  python3 -B -m unittest discover -s tests -t .
```

### Assess and compare without a provider (optional)

From `experiments/case-playground`; with `--out`, use that folder instead of
`~/markitect-playground-runs/fake-roombook-001`:

```
python3 -m playground host run --manifest examples/fake-roombook.json
python3 -m playground assess --run ~/markitect-playground-runs/fake-roombook-001 --fake-reviewers
python3 -m playground compare ~/markitect-playground-runs/fake-roombook-001 ~/markitect-playground-runs/fake-roombook-001
```

`--fake-reviewers` replaces both reviewer CLIs with `tests/fake_reviewer.py` and
throwaway credentials. Comparing a run with itself only shows the format. Before you
repeat this, remove the run folder and the `compare-*.md` file next to it; a repeated
`compare` overwrites that file without asking.

## Components

| Path | Role |
|---|---|
| `playground/__main__.py` | Entry point: `run` (inside the run container), `host`, `assess`, `compare`. |
| `playground/manifest.py` | Loads and validates a schema-1 manifest; the only validator. |
| `playground/cases.py` | Finds the cases: every valid folder in `cases/` (see [Cases](#cases)). |
| `playground/host.py` | Host side of a run: image and Markitect binary, staging, container start and wait, safety timeout, hand-back. Also `host clean`. |
| `playground/runner.py` | Inside the container: one trajectory from agent home and setup through waves S1 to SN to freeze, final assessment and report. Counts tokens, records why a run stopped. |
| `playground/lifecycle.py` | The case repository: prepare, snapshot, advance (release the next wave), freeze. Git on the agent's repo runs as its owner, with hooks off. Manual CLI: `python3 -m playground.lifecycle`. |
| `playground/methods.py` | Installs the method: the Conventional fragment or the Markitect product setup. Reads Markitect's roles from `.markitect/runtime.yaml`. |
| `playground/codex_agent.py` | Codex as outer agent (config, command, events, session records), and the runner for every process as user `agent`, including kill-all. |
| `playground/claude_agent.py` | Claude Code as outer agent: command, MCP config, token in the environment only, redaction, stream and transcript parsing. |
| `playground/assess.py` | Public checks on a scratch copy without `.git`; the final assessment adds the candidate's own tests and `markitect project check`. |
| `playground/report.py` | A run's `report.json` and `report.md`. `python3 -m playground.report <results>` rebuilds them. |
| `playground/evaluate.py` | `assess`: starts the assessment container; inside, checks, holdouts, diff profile, classification, product findings and reviews. |
| `playground/reviewers.py` | Codex and Claude reviewers: prompt, input bundle, commands, schema validation, agreement. |
| `playground/compare.py` | Side-by-side comparison of two assessed runs after a fairness check. |
| `container/Dockerfile` | The image: `node:22-bookworm-slim`, Python, Git, bubblewrap, Codex CLI and Claude Code at pinned versions, user `agent` (uid 1000). |
| `cases/` | `task-prompt.txt` (the one prompt); `common/` (`AGENTS.md`, `QUALITY.md`, the public checks' driver `checks/acceptance.py`), seeded into every repository; one folder per case with `README.md`, `BACKLOG.md`, `STATIONS.json`, its public checks `checks/<case>.py` and, for brownfield cases, starting code. |
| `methods/` | `conventional/AGENTS.fragment.md`, appended to `AGENTS.md`; `markitect/README.md`, notes for people. |
| `evaluation/` | Hidden: `config.json`, reviewer prompt and schema in `common/`, ground truth, holdouts, mutants, reference and `validate.py` in `readinglog2/`. Never staged into a run. |
| `examples/` | Manifests: real arms (`conventional-readinglog.json`, `markitect-readinglog.json`) and fakes (`fake-roombook.json`, `fake-readinglog2.json`, `fake-claude-roombook.json`). |
| `tests/` | Unit tests, fake stand-ins for Codex, Claude Code and the reviewers, and the Docker smoke `smoke_docker.py` (not picked up by unittest). |

## Data flow

```
HOST  python3 -m playground host run --manifest M.json
  |  validate manifest; check that login files exist (never read them)
  |  docker build   -> image markitect-playground:codex-<v>-claude-<v>
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
  |  freeze; final checks, own tests, markitect project check; report
  v
HOST  remove container; hand results/ back to you; write host.json

HOST  python3 -m playground assess --run <out>
  v
ASSESSMENT CONTAINER mpg-assess-<id> (same image)
  |  /assess/run = run folder (ro)  /assess/in = code + evaluation files (ro)
  |  per wave, on a copy of that wave's main:
  |     public checks again, holdouts, diff profile, classification, product findings
  |  per wave: Codex and Claude reviews (holdouts removed first)
  v
<out>/assessment/report.md, report.json, product-findings.md

HOST  python3 -m playground compare RUN_A RUN_B  -> fairness check -> comparison .md
```

What the agent sees:

- It works in `/work/<case>` as user `agent` and can read `/in`. The public checks are
  part of the seed (`checks/acceptance.py` and `checks/<case>.py`), so it can run them
  itself.
- `/out` is root-only while the run lasts: no check results, snapshots or reports. It
  never sees the other case, the other arm's method files, `evaluation/` or this page.
- Codex runs with `--dangerously-bypass-approvals-and-sandbox`, Claude Code with
  `--dangerously-skip-permissions`; the container is the isolation boundary. No host
  home, settings, hooks or MCP servers are used.
- With Claude Code as the outer agent, both arms first get the same one-line `CLAUDE.md`
  (`@AGENTS.md`) when the repository has none.

Each wave's timeout is `stationSeconds` or the rest of `totalSeconds`, whichever is
smaller. A run stops early only when a needed login is missing, the method setup is
`blocked`, the agent never produced a session id, could not start or exited without any
work, a snapshot or wave release failed, or the agent time is used up. A wave timeout or
a failing agent that did work is recorded, and the run goes on. After a stop, freeze and
report still run, and the final assessment when at least one wave ran. The host kills the container
after `totalSeconds` plus the runner's worst-case own work, as a safety net.

## Running real arms

1. Copy an example manifest and give it a new `id`. For Markitect set
   `markitect.commit` to the product commit to test; it is built from this checkout
   unless `markitect.sourceRepo` names another. For the six-wave change series set
   `case` to `readinglog2`.
2. Run the two arms of a pair one after the other, never in parallel (see
   [Caveats](#caveats)). Alternate which arm goes first from pair to pair. Keep both run
   folders; `host.json` records when each ran.
3. Assess both runs, then compare them.

```
python3 -m playground host run --manifest my-conventional.json
python3 -m playground host run --manifest my-markitect.json
python3 -m playground assess --run ~/markitect-playground-runs/<conventional-id>
python3 -m playground assess --run ~/markitect-playground-runs/<markitect-id>
python3 -m playground compare ~/markitect-playground-runs/<conventional-id> ~/markitect-playground-runs/<markitect-id>
```

### Logins

- **Codex:** `~/.codex/auth.json` from `codex login`. Needed for agent kind `codex`, for
  kind `claude` with method `markitect` (inner roles run on Codex) and for the Codex
  reviewer.
- **Claude Code:** create a token once with `claude setup-token` and save the token alone
  in a file. `assess` reads `~/.markitect-playground/claude-token` by default; `host run`
  needs `--claude-token PATH` for kind `claude`. Your own `~/.claude` login is never used.
- The host only checks that login files exist and mounts them read-only. Inside, the
  Codex login is copied into a fresh agent home; the Claude token goes only into the
  claude process environment and is redacted from results if the agent prints it.

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
| `agent.model` | `[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}` | required | Outer model; for Codex also the subagent default. |
| `agent.effort` | `[a-z]{1,32}` | required | Outer reasoning effort. |
| `agent.maxSubagents` | positive integer | required | Codex `agents.max_concurrent_threads_per_session`. Does not limit Claude Code (recorded only). |
| `limits.stationSeconds` | positive integer | required | Agent time limit per wave. |
| `limits.totalSeconds` | positive integer | required | Agent time budget for all waves. Also sets the host's safety timeout and Markitect's cost cap (x 50,000 tokens, at most 10^12). |
| `container.cpus` | positive number | `4` | `docker run --cpus`. |
| `container.memory` | `[1-9][0-9]*[bkmg]?` | `8g` | `docker run --memory`. |
| `container.pidsLimit` | positive integer | `2048` | `docker run --pids-limit`. |
| `markitect` | object | required for `markitect`, forbidden otherwise | The product to install. |
| `markitect.sourceRepo` | non-empty path | the checkout holding the playground | Local Markitect checkout to build from. A relative path or `~` counts from the manifest's folder; recorded as an absolute path. |
| `markitect.commit` | 7 to 40 lowercase hex | required | Commit to build; the full hash is recorded. |
| `markitect.innerModel` | like `agent.model` | `agent.model`; required for Claude kinds | Model of Markitect's inner roles. |
| `markitect.innerEffort` | like `agent.effort` | `agent.effort`; required for Claude kinds | Effort of Markitect's inner roles. |

### `host run` and `host clean`

| Option | Default | Meaning |
|---|---|---|
| `--manifest PATH` | required | The manifest. |
| `--out DIR` | `~/markitect-playground-runs/<id>` | Run folder. Must not exist (except one whose attempt failed before a container started; it is replaced) and must not be inside a Git checkout. |
| `--codex-auth PATH` | `~/.codex/auth.json` | Codex login, used only when needed. |
| `--claude-token PATH` | none | Required for kind `claude`, optional for `fake-claude`, ignored otherwise. |
| `--keep-container` | off | Keep the container after it ends. |

`python3 -m playground host clean` removes every stopped playground container (label
`markitect-playground=1`), assessment containers included.

### `assess`

| Option | Default | Meaning |
|---|---|---|
| `--run DIR` | required | Run folder with `host.json` and `results/`. |
| `--reviewers LIST` | `codex,claude` | One, both or `none`; each must be in `evaluation/config.json`. |
| `--codex-auth PATH` | `~/.codex/auth.json` | Codex reviewer login. |
| `--claude-token PATH` | `~/.markitect-playground/claude-token` | Claude reviewer token. |
| `--image IMAGE` | the run's image | Image of the assessment container. |
| `--force` | off | Replace an existing `assessment/` folder. |
| `--keep-container` | off | Keep the container after it ends. |
| `--fake-reviewers` | off | Fake reviewer CLIs, throwaway credentials, no model call. Not with `--codex-auth` or `--claude-token`. |

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
| `reviewers.codex.model` | `gpt-6.1-sol` | required | Codex reviewer model. A reviewer exists only if listed. |
| `reviewers.claude.model` | `claude-opus-5-5` | required | Claude reviewer model. |
| `reviewers.<name>.effort` | `high` | flag left out | Reasoning effort. |
| `reviewers.<name>.timeoutSeconds` | `2700` | `2700` | Hard limit per review. |
| `promptMaxBytes` | `100000` | `100000` | Prompt cap; long parts are cut with a pointer to the full file. |
| `holdoutTimeoutSeconds` | `600` | `600` | Hard limit for the holdouts per wave. |

### Exit codes

| Command | Exit code |
|---|---|
| `host run` | The runner's code: 0 all waves ran (whatever the quality), 1 stopped early, 2 runner error. Also 124 host safety timeout, 130 interrupted, 2 invalid manifest or the host could not build, start or wait. |
| `assess` | 0 written; 2 assessment error (`assessment/assess-error.txt`) or host error; 124 and 130 as above. |
| `compare` | 0 written; 2 unreadable report or fairness mismatch. |

## Results

### Run folder

| Path | Content |
|---|---|
| `results/report.md`, `report.json` | **Start here.** One row per wave, setup, final assessment, classification, fairness fields, versions. |
| `host.json`, `image-build.log`, `container.log`, `inputs/` | Host record (normalized manifest, status, host platform, image, Docker version, Markitect build, times, `docker run` arguments, exit code, `handBack`); build and container output; what was mounted at `/in`. |
| `results/runner.json`, `runner-error.txt` | Status, stop reason and category, versions, login rewrite, token redactions; a traceback after a runner error. |
| `results/setup/`, `stations/S<n>/`, `final/` | Setup command output; per wave the agent's events, output, `agent.json`, `checks.json` and work outside the repository; the last wave's public checks, own tests and Markitect conformance. |
| `results/audit/`, `evidence/` | Seed and wave plan (`run.json`), per-wave snapshots, the final freeze; session records, transcripts and Markitect's cache. |

### Assessment folder

| Path under `<run>/assessment/` | Content |
|---|---|
| `report.md`, `report.json`, `product-findings.md` | Per wave: public checks, holdouts, diff profile, reviewer findings and agreement, obligations, escalations, class; evaluation commit and file hashes, reviewer models, image. Product findings: setup steps, failed MCP calls and product commands, with exact error text. |
| `stations/S<n>/` | Checks, holdouts, `wave.diff`, what the reviewers got and their answers. |
| `host.json`, `container.log`, `inputs/`, `assess-error.txt` | Host record, container output, staged code and evaluation files; a traceback when the assessment failed. |

### Outcome classes

The run report classifies the run; the assessment classifies the run and each wave
(a wave only when it failed: exit, timeout, no session, harness error).

| Class | Meaning | Typical causes |
|---|---|---|
| `harness` | Our code | Runner error, failed snapshot or wave release, harness error files, host timeout or interruption. |
| `environment` | Docker, network, provider, logins | Missing login, no session id, auth or rate-limit errors, a wave without any agent activity, a failed container start or wait. |
| `product` | Markitect | Setup blocked by a product command, an MCP call that never finished, `markitect project check` that could not run. |
| `none` | No infrastructure cause | Completed, or the time budget was used up. |

With several causes, `harness` wins over `environment`, which wins over `product`; the
rest are listed too. A class other than `none` is a limit of the measurement or a
product finding, never a result of the method.

Holdouts read `passed/total`, plus `+N not judged` for status ERROR. ERROR means the
holdout could not judge, for example because its deadline passed. It is counted apart,
never as a failure of the candidate.

### Linux ownership hand-back

Containers write as root and keep snapshots root-only, so on a Linux host you could
neither read nor delete a run. After a container has stopped, the host runs a short
root container in the same image that gives `results/` (or `assessment/`) the owner of
its parent folder, without following links. `handBack` in `host.json` says `done`,
`not-needed` (Windows, running as root, or an engine that already maps root to you),
`skipped: ...` (Docker could not confirm the stop) or `failed: ...`. In the last two
cases the host warns, and the folder stays owned by root.

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
normalization) that later items do not repeat. S1 to S3 add features, S3 as a team wave
with interacting features; S4 refactors storage, S5 changes an earlier decision (pages
become optional), S6 renames a domain term. The backlog for S4 to S6 names the decision,
not the affected commands; finding them is the point.

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

Commit ground truth, holdouts, reviewer prompt, schema and config before the first run
they judge. The assessment reads these files from your checkout when it runs. It records
the checkout's commit, whether the playground folder had uncommitted changes, and each
file's SHA-256. It does not check that the commit is older than the run: compare the
commit date with `startedAt` in the run's `host.json` yourself.

### Holdouts

```
python3 -I -B holdout.py --repo DIR --station N [--deadline SECONDS]
```

- Prints `{"station": N, "checks": [{"id", "status", "item", "rule", "detail",
  "source"}]}` and exits 0 even when checks fail.
- Station N runs every holdout released up to N, so earlier waves are checked again for
  regressions. Each check runs the CLI on fresh temporary data.
- Each holdout names its item, its rule and, in `source`, the public sentence it derives
  from. What the public text leaves open is not tested.
- PASS and FAIL judge the candidate; ERROR means the holdout could not judge. Checks not
  yet run when `--deadline` passes report ERROR.
- The assessment runs it as the unprivileged user on a copy of the wave's `main`, with
  `--deadline` up to 90 s before `holdoutTimeoutSeconds`.

### Reviewers

- Every selected reviewer reviews every wave, after all checks and holdouts. The
  holdouts are removed first and `/assess` is root-only, so a reviewer never sees
  holdouts, check results or the other reviewer's answer.
- **Codex:** `codex exec --json --sandbox read-only --output-schema ...`, a fresh
  `CODEX_HOME` holding only the login.
- **Claude:** `claude -p --output-format json --json-schema ...`, tools `Read`, `Grep`,
  `Glob` only, no settings (`--setting-sources ""`, `--strict-mcp-config`), a fresh
  `CLAUDE_CONFIG_DIR`, the token only in its environment.
- Both run as user `agent` in the input bundle folder, so no `AGENTS.md` or `CLAUDE.md`
  is loaded as instructions; they read the snapshot copy by path. Model and effort come
  from `evaluation/config.json`. Choose reviewer models that differ from the arms'
  models; nothing checks this.
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
  method (`.markitect/`, Markitect guidance in `AGENTS.md`). The reviews are not blind
  to it.

### Compare's fairness check

`compare` refuses two runs unless these match, or `--allow-mismatch` is given:

- case and outer provider;
- the run report's fairness fields: stations, host platform, Codex and Claude Code
  versions, image ID, model, effort, subagent limit, time limits, container size;
- the SHA-256 of every evaluation file;
- each reviewer's model, effort and CLI version.

Never pool runs from different host platforms
([DEC-013](../../docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground));
`--allow-mismatch` is for looking, not for study results. It does not compare the Markitect commit, the inner model or the evaluation commit;
check those yourself.

## Caveats

- **Login refresh race.** Each run copies your Codex login. If Codex refreshes its token
  inside a container, the host login can be used up, and the next run (often the other
  arm) fails with "agent never produced a session id". Run arms one after the other, run
  `codex login` right before each run, or give each run its own login file with
  `--codex-auth`. The run report says when Codex rewrote its login, the assessment when
  a reviewer refreshed its login.
- **Never commit run outputs.** Keep them outside the repository. `host run` refuses a
  run folder inside a Git checkout, and `runs/` is ignored as a safety net.
- **One sandbox relaxation.** Every run and assessment container runs with
  `--security-opt seccomp=unconfined`: Codex's bubblewrap sandbox, which Markitect's inner roles use,
  needs user namespaces. Both arms get it; no `--privileged`, no added capabilities.
  There is no egress filtering.
- **Same image for both arms.** Only the two CLIs are pinned. A rebuild that pulls newer
  base packages changes the image ID, and `compare` then refuses the pair.
- **Unequal concurrency.** The subagent limit applies to Codex sessions. Markitect's inner
  roles are extra sessions, so total concurrency is not equal by construction.
- **Old run folders.** Runs from the earlier Codex-only image need
  `--image markitect-playground:codex-0.162.0-claude-2.1.296` for the Claude reviewer.
- **Product limits.** The Markitect setup accepts only what the product allows at the
  pinned commit, such as certain models; see
  [methods/markitect/README.md](methods/markitect/README.md).
- **Fixed in the code.** Step timeouts (public checks 600 s, own tests 900 s, product
  steps 600 s, Git steps 120 s) and the Markitect binary for linux/amd64 only.

## Changing the playground

- Standard library only, Python 3.11 or newer; the same code runs on the host and in
  the container. Pass `encoding="utf-8"` explicitly and call `subprocess` with argument
  lists, never `shell=True`.
- Everything a run needs comes from the manifest and command-line options: no dates,
  hashes or user paths in the source.
- Unit tests run without Docker or a provider; tests that need root or the `agent` user
  skip cleanly elsewhere. The Docker smoke covers the end-to-end contract.
- Evaluation files are pre-registered and hashed: `ground-truth.json`, `holdout.py`,
  the reviewer prompt and schema, and `config.json`. Any edit, even a comment, changes
  the hash, and `compare` then refuses runs assessed before and after it.

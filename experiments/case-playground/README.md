# Case playground

Runs one coding agent through a staged backlog (waves S1 to SN from the case's
`STATIONS.json`: four for `readinglog` and `roombook`, six for `readinglog2`) inside a
fresh Linux container, once per method: **Conventional** (plain project rules) or
**Markitect** (the real product installation). Afterwards a separate assessment judges
each wave. [STUDY.md](STUDY.md) explains what the comparison is for;
[BLUEPRINT.md](BLUEPRINT.md) explains how the harness is built and
[EVALUATION.md](EVALUATION.md) how runs are assessed (it wins where the two differ).

## Prerequisites

- Docker with a Linux engine. Linux hosts are the reference (register DEC-013); Docker
  Desktop on Windows also works.
- Python 3.11 or newer on the host (standard library only).
- Go, only for Markitect runs: the host builds a Linux `markitect` binary from the
  product repository and commit named in the manifest.
- A Codex login on the host (`~/.codex/auth.json`) for Codex runs, for Markitect's inner
  roles behind a Claude Code agent, and for the Codex reviewer.
- A Claude Code token file for Claude Code runs and the Claude reviewer: create it once
  with `claude setup-token` and save the token alone in
  `~/.markitect-playground/claude-token`. The host's own `~/.claude` login is never used.

## Run

From this folder (`experiments/case-playground`):

```
python -m playground host run --manifest examples/conventional-readinglog.json
python -m playground host run --manifest examples/markitect-readinglog.json
```

Options: `--out DIR` (default `~/markitect-playground-runs/<id>`; it must not exist
and must not be inside a Git checkout, except that a folder whose run failed before a
container started is replaced), `--codex-auth PATH` (default `~/.codex/auth.json`),
`--claude-token PATH` (required for agent kind `claude`) and `--keep-container`.
`python -m playground host clean` removes stopped playground containers.

The host waits at most `totalSeconds` plus the runner's worst-case own work (setup,
checks, snapshots, final assessment) before it kills the container. That is only a
safety net; a normal run ends long before.

A manifest names the case (`readinglog`, `roombook` or `readinglog2`), the method, the
outer agent (`agent.kind`: `codex` or `claude`, or the provider-free `fake` and
`fake-claude`), the Codex version (and `claudeVersion` for Claude kinds), model, effort
and subagent limit, the time limits, the container size and, for Markitect, the product
repository and commit. A Markitect run with a Claude Code agent also names
`markitect.innerModel` and `markitect.innerEffort`: the product's inner roles run on
Codex. Copy an example and change the `id`; for Markitect also set
`markitect.sourceRepo` to your local product checkout. The image is tagged
`markitect-playground:codex-<version>-claude-<version>` and ships both CLIs.

## Results

Everything lands in the output folder: `host.json` (image, versions, exit code) and
`results/` with `report.md` (start here), `report.json`, `setup/` (every setup
command with its output), `stations/S1..SN/` (agent events, public check results and,
under `outside/`, unfinished work outside the repository such as Markitect's candidate
workspaces), `final/` (final checks, the candidate's own tests, Markitect conformance),
`audit/` (Git snapshots of every station) and `evidence/` (session records and
Markitect's cache). The agent never sees check results. The report classifies the run
outcome (`harness`, `environment`, `product` or `none`) and lists Markitect's roles.
Containers write as root; on a Linux host the host then gives `results/` (and an
assessment's folder) back to the calling user and records this as `handBack` in
`host.json`.

## Assessment and comparison

```
python -m playground assess --run DIR [--reviewers codex,claude|none] [--image IMAGE] [--force]
python -m playground compare RUN_A RUN_B [--allow-mismatch] [--out FILE]
```

`assess` runs after the run, in its own container, on the frozen station snapshots:
public checks again, the hidden holdouts (for cases with `evaluation/<case>/`), the
wave's diff profile, failure classification, product findings and one review per wave
by each reviewer (models in `evaluation/config.json`). It writes
`<run>/assessment/report.md`, `report.json` and `product-findings.md`. Credentials
default to `~/.codex/auth.json` and `~/.markitect-playground/claude-token`
(`--codex-auth`, `--claude-token`); they are mounted read-only and never read on the
host. Runs from the Codex-only v1 image need `--image
markitect-playground:codex-0.162.0-claude-2.1.296` for the Claude reviewer. If the
report says a reviewer refreshed its login, log in on the host again before the next
run. `--fake-reviewers` replaces both reviewer CLIs with `tests/fake_reviewer.py` and
throwaway credentials (no model call).

`compare` writes a side-by-side comparison of two assessed runs (default
`compare-<A>-vs-<B>.md` next to RUN_A) and refuses runs whose fairness fields,
evaluation files or reviewers differ unless `--allow-mismatch` is given.

The evaluation files (ground truth, holdouts, reviewer prompt and config) must be
committed before the first run they judge; the assessment records the commit and the
file hashes. `python -I -B evaluation/readinglog2/validate.py` checks the holdouts
against the hidden reference and its mutants.

## Provider-free smoke run

`python -m playground host run --manifest examples/fake-roombook.json` (or
`python tests/smoke_docker.py [--manifest M.json]`, which also checks the result) runs
the whole pipeline with a scripted fake agent: no model call and no login needed.
`examples/fake-readinglog2.json` runs six stations, `examples/fake-claude-roombook.json`
stands in for Claude Code. A fake-agent manifest with `method: markitect` runs the real
product setup. Unit tests, without Docker:

```
python -B -m unittest discover -s tests -t .
```

## Caveats

- The container gets a read-only mounted copy of your `auth.json`. If Codex refreshes
  its token inside the container, the login on the host may be used up, and the next
  run (often the other arm) then fails to start with "agent never produced a session
  id". So run the arms one after the other, log in on the host (`codex login`) right
  before each run, or give each run its own login file with `--codex-auth`. The report
  says when Codex rewrote its login inside a run.
- Never commit run outputs; keep them outside the repository (`runs/` is ignored as a
  safety net).
- The old host-based harness (`experiments/work-item-comparison/playground`) is not on
  main; it stays as an archive on the research-backup remote. Do not use it for new runs.

# Survey: case playground

AI-generated, read-only, 10 October 2026. It covers the state of `experiments/case-playground` on branch `claude/case-playground-v2`: committed `f98313f1`, plus the work in progress captured in `archive/wip/case-playground-v2-20261010`. Both refs are on the research-backup remote. Input for PLAY-01 to PLAY-07 and CI-04.

## Architecture

The harness is Python, standard library only, about 5.9k lines in `playground/`. Subcommands: `run` (inside the container), `host`, `assess`, `compare`.

**Components:**
- **`manifest.py`:** schema-1 JSON validator; the single source of truth for what a run is.
- **`host.py`:** builds the Docker image, builds a Linux Markitect binary at a pinned commit, stages inputs, runs the container and enforces the host timeout.
- **`runner.py`:** one trajectory across N stations, with snapshots, usage and stop reasons.
- **`methods.py`:** setup for each arm.
- **`codex_agent.py`, `claude_agent.py`:** the outer agents.
- **`lifecycle.py`:** process cleanup, git snapshots, bundles.
- **`report.py`**, plus the assessment stage: `evaluate.py`, `reviewers.py` and `compare.py`.

**Flow of one run:**
1. Build the image, and for the Markitect arm the binary at `markitect.commit`.
2. Start a fresh container with read-only `/in` and writable `/out`. Only `auth.json` is copied in.
3. Set up the arm, then run the same prompt per station while resuming the session.
4. After each station:
   - kill the agent processes;
   - run the public checks, whose results the agent never sees;
   - take a git bundle snapshot.
5. Run final checks.
6. `assess` runs later in a second container on the frozen snapshots.

**Markitect arm:** `markitect project init`, then `onboard`, then `setup`, each as preview plus digest-checked write. The agent then gets `markitect project mcp`. Inner roles always run on Codex, because the product allowed nothing else before pull request #92.

**Conventional arm:** a plain `AGENTS.md` with a method fragment.

**Configurable today (manifest):** case, method, agent kind (codex, claude, fakes), CLI versions, model, effort, subagents, limits, container size, Markitect repo and commit.

**Hard-coded:** the case list, the image base, the default Claude version, reviewer defaults, cost weights, the Markitect setup sequence, the inner-role provider and the single prompt.

## Running it

- **Prerequisites:** Docker (Linux engine), Python 3.11 or newer, Go (Markitect arm only), `~/.codex/auth.json`, and a local product checkout that contains the pinned commit.
- **Commands:**
  - `python -m playground host run --manifest examples/<manifest>.json`
  - `python -m playground assess --run DIR`
  - `python -m playground compare RUN_A RUN_B`
  - provider-free smoke: `examples/fake-roombook.json` or `tests/smoke_docker.py`
- **Friction:**
  - Example manifests carry a placeholder repo path and an old commit.
  - Arms must be run by hand, one after the other.
  - Concurrent runs can invalidate each other's login.
  - There is no preflight.
  - Output goes outside git, to `~/markitect-playground-runs`.

## State of the work in progress

**Implemented:**
- the assessment container;
- a diff profile by area;
- product-findings extraction from MCP errors;
- failure classes `product|harness|environment|none`;
- Codex and Claude reviewers with a versioned prompt and JSON schema;
- fairness checks in `compare`;
- Claude Code as the outer agent;
- N stations;
- recording of the roles from `runtime.yaml`;
- a new case `readinglog2` (six waves, 15 items, rules R1 to R3) with a hidden reference implementation per wave.

**Missing:**
- `evaluation/readinglog2/ground-truth.json` and `holdout.py`, which `evaluate.py` already expects;
- mutant tests for the holdouts;
- per-role executor and model fields;
- statistics.

**Documents:** README, STUDY, BLUEPRINT and EVALUATION overlap. EVALUATION is the newest and takes precedence over BLUEPRINT.

## Gaps against the goal

- **Parametrization:** no case or arm plug-ins, no wave selection, no run matrix.
- **One command:** nothing runs both arms, assessment and comparison together; no preflight; no per-run credential copy.
- **Hardening:** unpinned image packages, no egress control, failure classes not in exit codes, partial retry, an unsolved credential-refresh race.
- **Documentation:** no single current architecture page.
- **CI:** a deterministic container smoke is feasible now. It would use the fake agent, the Markitect arm, a tiny case, a Linux runner with Docker, and Go to build the PR commit. It catches setup, MCP and check breakage, not outcome quality.

## Integration onto main

- Move only `experiments/case-playground/` (about 1.1 MB, mostly reference data) onto a branch from current main.
- Leave out `experiments/work-item-comparison` and the research history.
- **Product dependencies:**
  - the `project init/onboard/setup/check/mcp` contracts (these change with CLI-02);
  - `.markitect/runtime.yaml`;
  - the per-role configuration from pull request #92.

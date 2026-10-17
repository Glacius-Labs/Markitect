# Runtime baseline and Durchstich assessment

This harness exercises the [container runtime](../../packaging/runtime/README.md) with real provider sessions. It also assesses library realizations with the independent [library oracle](../library-playground/README.md). Two jobs:

- a single-agent baseline: one Claude Code or Codex session realizes Library M1 in the hardened runtime container;
- assessment of the first real Durchstich and Apply results.

The baseline is a runtime and harness check. It is not an Apply and shows nothing about the Manager organization.

| File | Use |
| --- | --- |
| [assess.py](assess.py) | Builds a realization's `./cmd/library` in the example Go runtime image and runs the oracle from `experiments/library-playground` at a fixed revision. Both steps run in containers without network, because the realization is Agent-produced code. |
| [run_oracle.py](run_oracle.py) | Runs inside the assessment container. It writes the manifest with program and evaluator hashes, then runs the oracle. |
| [baseline.py](baseline.py) | Gives one session the M1 model and the software vocabulary, never the evaluation contract, and asks for the realization. It then assesses the result. Claude Code runs Haiku and Codex runs Luna, both with high effort. |
| [codex_login.py](codex_login.py) | Creates a dedicated Codex runtime login with device-code authorization, in its own Docker volume, never the desktop login. Sessions receive a copy without the refresh token. |

## Prerequisites

Build the images from the repository root:

```text
docker build -f packaging/runtime/Dockerfile -t markitect-runtime:dev .
docker build -f packaging/runtime/examples/go/Dockerfile --build-arg BASE=markitect-runtime:dev -t markitect-runtime-go:dev .
```

Credentials stay outside the repository:

- Claude Code reads `CLAUDE_CODE_OAUTH_TOKEN` from `~/.markitect/runtime.env`, or the file named by `MARKITECT_RUNTIME_ENV`.
- Codex uses the runtime login volume `mk-codex-runtime-auth`. Create it once with `python -I codex_login.py login` and check it with `python -I codex_login.py status`.

The scripts never print a credential. `MARKITECT_ENGINE` overrides the container engine executable.

## Use

```text
python -I experiments/runtime-baseline/baseline.py --provider claude
python -I experiments/runtime-baseline/baseline.py --provider codex
python -I experiments/runtime-baseline/assess.py --realization PROJECT_OR_MODULE --profile m1 --out OUT --label A1
```

Runs go to `~/.markitect/durchstich/runs`, or the directory named by `MARKITECT_BASELINE_RUNS`. A run directory holds:

- the transcript and standard error;
- the realization;
- the build and oracle logs;
- the assessment JSON;
- `summary.json`.

Agent-produced code and transcripts never enter the repository. Exit 0 from `assess.py` means every oracle case passed, including the opposite-profile discrimination.

## Results

The first baselines are recorded in the [capability record](../../docs/validation/runtime-capabilities-20261017.md#single-agent-baselines).

## Limits

- **Model digest not verified.** The harness records the evaluator revision, not a verified model digest.
- **Behavior only.** The oracle covers behavior; source review, history and registry inspection remain separate evidence.
- **Not hardened like a product session.** Credentials pass through the engine command's environment, not agentd's start message.
- **Bounded spend.** The Claude baseline stops at its budget (`--budget`, default 5 USD).

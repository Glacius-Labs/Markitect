# Single-agent comparison

Tools for the controlled comparison of Markitect V1 with the owner's primary baseline: ordinary documentation with the same meaning, given to one coding agent ([DEC-072](../../docs/concepts/register.md#dec-072-stable-v1-justified-by-data), [measurement guide](../../docs/guides/measurement.md#quality-first-comparative-evaluation)). The [protocol](../../docs/validation/v1-comparison-protocol-20261024.md) owns the design, the run order and the decision rule; this folder supplies the inputs and mechanics. Nothing here starts a Markitect Apply: arm M runs through the public `apply` command.

| File | Use |
| --- | --- |
| [requirements.py](requirements.py) | Renders arm B's ordinary documentation from the canonical [Library model](../../testdata/library/README.md): every Definition body verbatim under plain headings, links as Markdown links, four listed substitutions of phrases that name the model. `check` proves the committed documents are the current rendering. |
| [packet/requirements-m1.md](packet/requirements-m1.md), [packet/requirements-m2.md](packet/requirements-m2.md) | The rendered documents for M1 and M2 (loan limit 3). They differ in one table cell. |
| [packet/prompt.txt](packet/prompt.txt) | Arm B's only instruction: the owner's "read the documentation and implement it accordingly", plus the delivery and preservation duties every Markitect Implement Manager receives. |
| [runner.py](runner.py) | Builds byte-exact fixture bundles and runs one fresh `codex exec` session in the runtime image with a Manager session's hardening, limits, network and Codex switches; captures the result, the transcript and the Codex session files; removes the login and every resource it created. |
| [assess_candidate.py](assess_candidate.py) | Assesses one candidate of either arm: the unchanged [oracle harness](../library-oracle/README.md#container-harness), the hidden probes, the candidate's own build and tests under the `project-tests` limits, a census of every changed path and an optional blinded copy. |
| [probes.py](probes.py) | Six hidden behavior probes for stated obligations the oracle does not reach, in the oracle's case format and transport. |
| [rubric.md](rubric.md) | The blinded review rubric for documentation, tests, scope and change discipline. |

The probes, the rubric and the oracle are assessor material. Writers receive only their fixture: arm B a Git repository with `requirements.md` and its prior work, arm M the Markitect fixture.

## Self-check

```text
python -B -m unittest discover -s tools/comparison
```

The tests build real Git fixtures, compare the Codex switches and session limits with `internal/adapters/codexappserver/profile.go`, `internal/adapters/container/spec.go` and `examples/runtime-profile.example.yaml`, replay probe transcripts through the oracle's transport and run the runner against a fake engine, including the deadline path. They start no container, provider or Apply. On Windows, set `TMP` and `TEMP` to a user temp directory first.

## Limits

- The runner is the smallest honest harness, not a product path. Its session is `codex exec`; Markitect Managers use the same Codex binary through `codex app-server`.
- The login copy carries no refresh token, like a Manager session's. Account refresh during a long series is not handled.
- Assessment covers behavior through the public interface, the candidate's own checks and a census. Source review, test semantics and documentation consistency come from the blinded rubric review.

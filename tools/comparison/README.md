# Single-agent comparison

Tools for the controlled comparison of Markitect V1 with the owner's primary baseline: ordinary documentation with the same meaning, given to one coding agent ([DEC-072](../../docs/concepts/register.md#dec-072-stable-v1-justified-by-data), [measurement guide](../../docs/guides/measurement.md#quality-first-comparative-evaluation)). The [protocol](../../docs/validation/v1-comparison-protocol-20261024.md) owns the design, the run order and the decision rule; this folder supplies the inputs and mechanics. Nothing here starts a Markitect Apply: arm M runs through the public `apply` command.

| File | Use |
| --- | --- |
| [requirements.py](requirements.py) | Renders arm B's ordinary documentation from the canonical [Library model](../../testdata/library/README.md): every Definition body verbatim under plain headings, links as Markdown links, four listed substitutions of phrases that name the model. `check` proves the committed documents are the current rendering. |
| [packet/requirements-m1.md](packet/requirements-m1.md), [packet/requirements-m2.md](packet/requirements-m2.md) | The rendered documents for M1 and M2 (loan limit 3). They differ in one table cell. |
| [packet/prompt.txt](packet/prompt.txt) | Arm B's only instruction: the owner's "read the documentation and implement it accordingly", plus the delivery and preservation duties every Markitect Implement Manager receives. |
| [runner.py](runner.py) | Builds byte-exact fixture bundles, compares fixture trees, and runs one `codex exec` session (or resumes the chain's thread) in the runtime image with a Manager session's hardening, network and Codex switches and the six-Manager aggregate resource ceiling; captures the result, the transcript and the Codex session files; removes the login and every resource it created; takes the shared engine lock. |
| [assess_candidate.py](assess_candidate.py) | Assesses one candidate of either arm: the unchanged [oracle harness](../library-oracle/README.md#container-harness) with its evaluator from the frozen source, the hidden probes, the candidate's own build and tests, a census of every changed path and an optional blinded copy. |
| [probes.py](probes.py) | Six hidden behavior probes for stated obligations the oracle does not reach, in the oracle's case format and transport. |
| [phase-b/](phase-b), [phase_b_cases.py](phase_b_cases.py), [packet/requirements-m3.md](packet/requirements-m3.md) to [m5](packet/requirements-m5.md) | Phase B: model overlays M3 (short-loan copies), M4 (renewals) and M5 (no member suspension) on top of M2, their rendered documents, and the hidden cases that judge them, built on the unchanged oracle's helpers and transport. `requirements.py materialize` writes a complete snapshot for arm M. |
| [phase-c/](phase-c/requirements/README.md) | Phase C: the [product brief](phase-c/brief.md) and the neutral Makerspace requirements for the fresh build W0 and the backlog changes C1 to C9. Every later Phase C input derives from them. The hidden oracle and reference implementations live in [../makerspace-oracle](../makerspace-oracle/README.md) and [../makerspace-reference](../makerspace-reference/README.md). |
| [phase-c/model/](phase-c/model/README.md), [phase-c/spec/](phase-c/spec/render.py), [phase-c/mapping/](phase-c/mapping/w0.md), [phase-c/execution/](phase-c/execution/w0.yaml) | Phase C: the canonical Makerspace model as one complete snapshot per step, arm B's documents rendered from them by this renderer with a Makerspace `Document`, the mapping from every neutral statement to its Definitions, and arm M's proposed execution configuration (ownership and project checks). |
| [rubric.md](rubric.md), [review-prompt.md](review-prompt.md) | The artifact review rubric and the frozen prompts for the per-candidate reviewer and the confirming reviewer. Neither mentions arms or a comparison. |
| [rubric_check.py](rubric_check.py) | Checks every critical review finding mechanically before a second reviewer is asked to confirm it. |
| [analyze.py](analyze.py) | Applies the predeclared reading to the run ledger: conforming runs, independent units, exact intervals, the one-sided Fisher test and the reading of section 8 or B5. |

The probes, the rubric, the review prompts and the oracle are assessor material. Writers receive only their fixture: arm B a Git repository with `requirements.md` and its prior work, arm M the Markitect fixture.

## Self-check

```text
python -B -m unittest discover -s tools/comparison
```

The tests build real Git fixtures, compare the Codex switches and session limits with `internal/adapters/codexappserver/profile.go`, `internal/adapters/container/spec.go` and `examples/runtime-profile.example.yaml`, run the capture script on an over-broad `.gitignore`, scan compressed Git objects for secrets, replay probe transcripts through the oracle's transport and run the runner against a fake engine, including the deadline path. They start no container, provider or Apply. On Windows, set `TMP` and `TEMP` to a user temp directory first.

## Limits

- The runner is the smallest honest harness, not a product path. Its session is `codex exec`; Markitect Managers use the same Codex binary through `codex app-server`, with the Manager harness as their base instructions.
- The login copy carries no refresh token, like a Manager session's. Account refresh during a long series is not handled.
- Assessment covers behavior through the public interface, the candidate's own checks and a census. Source review, test semantics and documentation consistency come from the blinded review.

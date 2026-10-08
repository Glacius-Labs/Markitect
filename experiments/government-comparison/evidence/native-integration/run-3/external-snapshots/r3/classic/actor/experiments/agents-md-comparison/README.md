# Matched AGENTS.md / Markitect comparison

This directory preserves a frozen protocol and the completed three-task, two-arm comparison on the public MIT-derived MyMeetings source subset. Read the [comparison report](../../docs/validation/agents-md-vs-markitect.md) for outcomes and all sixteen decision answers. Overall classification: **D, inconclusive**. No Core change, new release or general benefit claim resulted.

The original [design](design.md), [knowledge parity](knowledge-parity.md), [tasks](tasks/) and [hidden oracles](oracles/) remain unchanged after dispatch. Preparation-status sentences describe their pre-run state. `frozen-manifest.json` binds exact source files, guidance, task packets, evaluator and runner. [Integration](integration.json) separately binds PR #58 and the unreleased clean-source binary used here. The published release remains v0.12.0.

## Evidence

`results/summary.json` contains six scored runs, partial read metrics, changed-file hashes, supplied Context identities/inclusion, operator commands and audit outcomes. Each `results/runs/<id>/` contains a normalized run record, supplied prompt, agent report, read ledger, ordinary diff, independent evaluation and a changed-artifact archive. The archive includes newly created files which an ordinary unstaged `git diff` omits. Each source archive retains the public source license. Generated views are recorded separately from manually maintained source/model records.

`results/raw-evidence-index.json` gives **original raw hashes** and separate **presentation-normalized hashes**. Machine-specific roots are replaced by aliases; nested JSON output and presentation line endings are also normalized. These copies are not raw logs. Stored unified diffs retain their literal context-line whitespace. Original raw artifacts remain in local operator storage, identified by aliases:

| Alias | Local locator (ignored, not committed) | Meaning |
|---|---|---|
| R2 | `.artifacts/comparison-run-root.txt` | Four scored Tasks 1/2 runs and unscored blocked Task 3 attempt |
| R3 | `.artifacts/comparison-task3-root.txt` | Two fresh Task 3 runs inside the permitted workspace |
| R1 | `.artifacts/comparison-run-root-r1.txt` | Interrupted pre-correction read-meter trial |
| MARKITECT | current repository | Protocol, evaluator, source tool and original operator scripts |

Known other checkouts and reference-evaluator scratch directories have generic audit aliases. The full OS, transient writes and unknown directories were not traced. Per-run snapshots, helper/executable/Context hashes and known-root deltas are the recorded isolation evidence. The whole-R2 delta includes four explicitly accounted operator-preparation paths in addition to agent changes. Whole-R3 uses the already-recorded first-agent pre-run audit as its baseline; `whole-audit-provenance.json` explains the post-collection alias copy. No baseline was measured retrospectively.

`results/excluded-runs.json` records the R1 meter abort, R2 permission rejection and R3 preflight failures. Blocked attempts are not implementation failures and are not pooled into the scored metrics. `results/reviews/` and the Task 1 addendum preserve reviewer-proxy assessments, not human approvals. Complete model/tool/token telemetry is unavailable.

## Reproduction

1. Use the preserved `../real-project-adoption/evidence/adopter-snapshots.bundle` and verify its manifest/hash. Keep the source/license boundary unchanged.
2. Build a clean checkout of integrated source commit `a23db5f48854cbf7723789843aaa10ddf06cdb0b`, using Go 1.27.1, `CGO_ENABLED=0` and `go build -trimpath -o <external-output-path> ./cmd/markitect`. Keep the binary outside that clean source tree. Record executable SHA-256 and `go version -m`; do not use the published v0.12.0 binary for the new analysis option. The actual study Windows binary hash is in `integration.json`.
3. Run `run.ps1 -PrepareOnly` with `-SourceBundle`, `-MarkitectExe`, `-RunRoot` and `-FrozenManifest`. The original harness requires a fresh temporary root. Use an execution environment that explicitly permits those checkouts; do not bypass a permission rejection. Review the declared Task 1 projection plan before `-ApplyPreparedProjections`.
4. Capture hidden evaluator baselines before agents with `evaluation/evaluate.ps1 -CaptureBaselineOnly`. Keep oracles/evaluation/operator files out of all task input packets. Freeze both arms before the first dispatch. The common facade remains identical across arms and links their project-owned tests.
5. For collaboration execution use `-BeginExternalAgent`, the resulting frozen prompt, a fresh `fork_turns=none` agent and an explicit workspace. Then collect its report with `-CompleteExternalAgent`. Keep runs sequential. CLI `-Execute` is available only in an authenticated environment; it was not used here.
6. Run the frozen independent evaluator with the task/arm/baseline, then collect read-only `check`/`model` and applicable render-check results. Do not correct the sealed candidates. Finish the per-run and whole-run audits before writing tracked reports.
7. Task 3's actual relocation is separately bound by `results/operator/r3-r3-protocol.json`. `results/tools/setup-task3-authorized.py` reproduces that local setup from already-prepared pristine Task 3 inputs: new clones, exact tracked working bytes, helper/packet copies, unchanged original Git objects, and two narrow derived-runner changes. Local Git indexes require renormalization after byte restoration; verify zero staged differences. Before the first agent, alias its captured `pre-run-audit.json` as the whole-run baseline. Common parent AGENTS is part of the recorded limitation. This is experiment scaffolding, not a Markitect feature or a workaround against rejected temporary files.
8. `results/tools/gather-comparison.py` documents post-run extraction (Python with PyYAML) and alias normalization; it needs the local ignored locators/raw areas. It is not an independent model compiler. Check the committed portable evidence from the repository root with `python experiments/agents-md-comparison/results/tools/verify-results.py`.

All source/package inputs and concrete completed patches can be inspected/rechecked. Future fresh-agent behavior is stochastic and is not promised to reproduce the same patch, reads or timing. Recreated Git commits have recorded study-local identities; do not describe them as original upstream commits or assume their digests equal a future reconstruction. Initial NuGet feed failures, explicit public-feed operator successes and the Task 3 agent/operator SDK discrepancy are preserved in the report.

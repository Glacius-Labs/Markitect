# Makerspace judge

The hidden series judge of the Makerspace change series W0 to C9, a source for [`judges.py export`](../../evaluation/judges.py). It is the comparison's Phase C assessment ([assess_candidate.py](../../comparison/assess_candidate.py), profiles `w0` and `c1` to `c9`) as one judge plug-in; the [evaluation suite runner](../../evaluation/README.md#change-series) explains how a series judge runs.

| Path | Content |
| --- | --- |
| [judge.json](judge.json) | The image (`golang:1.27.1-bookworm` by digest), the command, a one-hour limit, the container limits, the `steps`, one reference per step (the [Makerspace reference implementations](../../makerspace-reference/README.md) placed at `realization/`), the oracle-family mutants and the `include` map of the exported folder |
| [judge.py](judge.py) | The judge, run in its container; it shares [judgekit.py](../common/judgekit.py) with the Library judge |
| [drift.json](drift.json) | The terms a step retired, with the requirement sentence that retires them |
| [mutants/](mutants) | Seeded mutants of the references, each at its step with the area where the judge must fail it |

## What it judges

- **Correctness:** the hidden [Makerspace oracle](../../makerspace-oracle/README.md) at the step, whose case set holds every earlier step's cases rendered for this step (54 cases at W0, 63 at C9), against the result's own `go build ./cmd/makerspace`. At C3, C7 and C9 it also runs the oracle's change census against the start's program: one `CENSUS-` case per census case, which fails when the result answers differently where the change says nothing changes. When the start has no program that builds, the census cannot run, and its cases fail as not run. So does any planned case the oracle did not reach. Each case carries the step that introduced it, and `byStep` counts them per step.
- **Build and tests:** `go build ./cmd/makerspace` and `go test ./...` in `realization/`, each with the Phase C own-check limit of 300 seconds, without network, module proxy or toolchain download.
- **Scope:** every changed path against the start. The task asks for `realization/` only. Markitect's registry pages and Results (`.markitect/registry/`, `.markitect/results/`) are its bookkeeping; any other `.markitect/` file, `task.md` and `TASK.md` are protected inputs, unless a changed file is a byte-equal copy of the task's own model file; a compiled executable anywhere and any other path are unrequested.
- **Drift:** earlier steps' failing cases; the retired terms of [drift.json](drift.json) (from C6, the certification's `expiresOn` and status `expired`) in non-test Go source; and every file of the tree that nothing accounts for.

## Mutants

| Mutant | Step | Seeded defect | Must fail in |
| --- | --- | --- | --- |
| [failing-build](mutants/failing-build.json) | W0 | The entry point calls `NewServer` with an argument | Build, and Correctness |
| [unrequested-change](mutants/unrequested-change.json) | C1 | A CI workflow outside `realization/` | Scope: unrequested; Drift: unaccounted |
| [unaccounted-files](mutants/unaccounted-files.json) | C2 | A Go file the build ignores and a leftover backup | Drift: unaccounted |
| [protected-input](mutants/protected-input.json) | C4 | Its own `TASK.md` | Scope: protected input |
| [forgotten-rule](mutants/forgotten-rule.json) | C5 | W0's plus tier fee is 4800 cents instead of 4900 | Correctness; Drift: forgotten rules |
| [stale-expiry](mutants/stale-expiry.json) | C6 | The retired `expiresOn` still named in the source | Drift: stale files |
| `family-*` (31) | W0, one at C2 | The [reference mutants](../../makerspace-reference/README.md), one per oracle probe family | Correctness, a case of that family |

Self-test in the container, from the repository root, under an approved dispatch (about 47 judgments):

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation-judges/makerspace --rev REV --out JUDGE_DIR
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation-judges/makerspace --judge JUDGE_DIR --out DIR --lock LOCK
```

The offline test ([test_series.py](../../evaluation/test_series.py)) runs a sample through a fake engine on the host when Linux and Go 1.27 or later are there.

## Limits

- The oracle and the references are public, in this repository: the Makerspace chain is pilot material ([suite design](../../../docs/design/evaluation-suite-v2.md#c-portfolio)).
- The stale-file check reads non-test Go source only. A test or a document may name a retired term to check its absence, so they are not searched.
- The references follow Phase C's module layout, not the v2 architecture of [makerspace-v2](../../comparison/makerspace-v2/README.md); the judge checks behavior, its own build and tests, scope and drift, not the module cut.

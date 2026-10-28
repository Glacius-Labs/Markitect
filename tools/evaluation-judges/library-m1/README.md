# Library M1 judge

The hidden judge of the Library M1 task, a source for [`judges.py export`](../../evaluation/judges.py). It is the comparison's M1 assessment ([assess_candidate.py](../../comparison/assess_candidate.py)) as one judge plug-in; the [evaluation suite runner](../../evaluation/README.md#judging) explains how a judge runs.

| Path | Content |
| --- | --- |
| [judge.json](judge.json) | The image (`golang:1.27.1-bookworm` by digest, the Go of the sandbox), the command, a 40-minute limit, the container limits and the `include` map of the exported folder |
| [judge.py](judge.py) | The judge, run in its container |
| [reference/](reference/realization/README.md) | The reference result: a complete Library M1 implementation in `realization/`, written for this judge and passing every area |
| [mutants/](mutants) | Seeded mutants of the reference, each with the area where the judge must fail it |

## What it judges

- **Correctness:** the unchanged [Library oracle](../../library-oracle/README.md) with profile m1 (its twelve cases), its opposite-profile witness as one more case, and the six [hidden probes](../../comparison/probes.py), all through the oracle's transport, against the result's own `go build ./cmd/library`. Nineteen cases in all; a result that does not build fails every one. The oracle files are checked against the hashes the export recorded.
- **Build and tests:** `go build ./cmd/library` and `go test ./...` in `realization/`, with the comparison's primary limits of 300 and 600 seconds, without network, module proxy or toolchain download.
- **Scope:** assess_candidate's census of every changed path against the start. The task asks for `realization/` only: any other path, and a compiled executable anywhere, is unrequested, except Markitect's registry files (`.markitect/registry/`), which are a variant's own bookkeeping. A changed `task.md`, `TASK.md` or Markitect model file is a protected input touched, unless it is a byte-equal copy of the task's own model file.

The result's code runs as the judge's user. With `--contained`, and only when the suite's container sets `EVALUATION_JUDGE_CONTAINER`, the judge stops every other process before it writes `judge.json`, so nothing the result started can write the judgment. The judge sources live here, apart from `tools/evaluation`, because that folder is mounted into every variant's sandbox as its code.

## Mutants

| Mutant | Seeded defect | Must fail in |
| --- | --- | --- |
| [broken-rule](mutants/broken-rule.json) | Six open loans instead of five | Correctness |
| [unrequested-change](mutants/unrequested-change.json) | A CI workflow outside `realization/` | Scope: unrequested |
| [failing-build](mutants/failing-build.json) | The entry point calls `Serve` with a missing argument | Build, and Correctness |
| [protected-input](mutants/protected-input.json) | Its own `TASK.md` with a changed rule | Scope: protected input |

Self-test in the container, from the repository root, under an approved dispatch:

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation-judges/library-m1 --rev REV --out JUDGE_DIR
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation-judges/library-m1 --judge JUDGE_DIR --task TASK_DIR --out DIR --lock LOCK
```

The offline test ([test_judging.py](../../evaluation/test_judging.py)) runs the same self-test through a fake engine on the host when Go 1.27 or later is on the path.

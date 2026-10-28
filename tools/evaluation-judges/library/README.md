# Library judge

The hidden series judge of the Library change series M1 to M5, a source for [`judges.py export`](../../evaluation/judges.py). It judges each step as Phases A and B of the comparison did; the [evaluation suite runner](../../evaluation/README.md#change-series) explains how a series judge runs. The [Library M1 judge](../library-m1/README.md) stays as it is for the recorded E2 judgments.

| Path | Content |
| --- | --- |
| [judge.json](judge.json) | The image (`golang:1.27.1-bookworm` by digest), the command, a 40-minute limit, the container limits, the `steps`, one reference per step and the `include` map of the exported folder |
| [judge.py](judge.py) | The judge, run in its container; it shares [judgekit.py](../common/judgekit.py) with the Makerspace judge |
| [drift.json](drift.json) | The terms M5 retired, with the requirement sentence that retires them |
| [references/](references) | The reference results of M2 to M5, each a complete `realization/` that builds, passes its own tests and every hidden case of its step; M1's is the [M1 judge's](../library-m1/reference/realization/README.md) |
| [mutants/](mutants) | Seeded mutants of the references, each at its step with the area where the judge must fail it |

## The steps

The requirements are the rendered documents [requirements-m1.md](../../comparison/packet/requirements-m1.md) to [requirements-m5.md](../../comparison/packet/requirements-m5.md): M2 lowers the loan limit to three, M3 adds short-loan copies, M4 renewals, and M5 removes member suspension.

## What it judges

- **Correctness:** M1 and M2, the unchanged [Library oracle](../../library-oracle/README.md) with that profile (twelve cases), its opposite-profile witness and the six [hidden probes](../../comparison/probes.py): 19 cases. M3 to M5, every carried and new case of [phase_b_cases.py](../../comparison/phase_b_cases.py) and its witness: 22, 28 and 27 cases. All through the oracle's transport, against the result's own `go build ./cmd/library`. Each case carries the step that introduced it.
- **Build and tests:** `go build ./cmd/library` and `go test ./...` in `realization/`, with the comparison's primary limits of 300 and 600 seconds.
- **Scope:** as the [Makerspace judge](../makerspace/README.md#what-it-judges): `realization/` is requested, Markitect's registry pages and Results are bookkeeping, other `.markitect/` files, `task.md` and `TASK.md` are protected inputs.
- **Drift:** earlier steps' failing cases; from M5, `setMemberStatus` and the status `"suspended"` in non-test Go source; and every file of the tree that nothing accounts for.

## Mutants

| Mutant | Step | Seeded defect | Must fail in |
| --- | --- | --- | --- |
| [failing-build](mutants/failing-build.json) | M2 | The entry point calls `Serve` with a missing argument | Build, and Correctness |
| [broken-rule](mutants/broken-rule.json) | M3 | A short loan lasts eight days | Correctness |
| [unaccounted-file](mutants/unaccounted-file.json) | M3 | Notes left in `realization/` | Drift: unaccounted |
| [forgotten-rule](mutants/forgotten-rule.json) | M4 | Four open loans again instead of M2's three | Correctness; Drift: forgotten rules |
| [unrequested-change](mutants/unrequested-change.json) | M4 | A CI workflow outside `realization/` | Scope: unrequested; Drift: unaccounted |
| [protected-input](mutants/protected-input.json) | M5 | Its own `TASK.md` | Scope: protected input |
| [stale-suspension](mutants/stale-suspension.json) | M5 | The removed status `"suspended"` still defined | Drift: stale files |

Self-test in the container, from the repository root, under an approved dispatch:

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation-judges/library --rev REV --out JUDGE_DIR
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation-judges/library --judge JUDGE_DIR --out DIR --lock LOCK
```

The offline test ([test_series.py](../../evaluation/test_series.py)) runs the whole self-test through a fake engine on the host when Linux and Go 1.27 or later are there.

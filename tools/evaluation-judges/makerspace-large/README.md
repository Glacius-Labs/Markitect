# Large Makerspace judge

The hidden series judge of the [large Makerspace](../../makerspace-large/README.md), a source for [`judges.py export`](../../evaluation/judges.py). It has the areas of the [Makerspace judge](../makerspace/README.md), for step W0 and the [change steps](../../makerspace-large/CHANGES.md) C1 to C8.

| Path | Content |
| --- | --- |
| [judge.json](judge.json) | The image, the command, the limits, the steps, each step's reference, and the `include` map of the exported folder |
| [judge.py](judge.py) | The judge, run in its container |
| [drift.json](drift.json) | The terms C7 and C8 retired |
| [mutants/](mutants) | Seeded mutants of the reference, each with the area where the judge must fail it |

- **Correctness:** the [hidden cases](../../makerspace-large/README.md#the-hidden-cases) against the result's own `go build ./cmd/makerspace`.
- **Build and tests:** `go build ./cmd/makerspace` and `go test ./...` in `realization/`, without network or module proxy. The test limit is 900 seconds, not Makerspace's 300, for a program five times the size.
- **Scope and drift:** as the Makerspace judge. The task asks for `realization/` only.

The mutants here test the judge's areas. The strength of the cases themselves is measured by the 89 seeded bugs of [calibrate.py](../../makerspace-large/oracle/calibrate.py), which run on the machine without a container.

Self-test in the container, under an approved dispatch:

```text
python -I -B tools/evaluation/judges.py export --source tools/evaluation-judges/makerspace-large --rev REV --out JUDGE_DIR
python -I -B tools/evaluation/judges.py selftest --source tools/evaluation-judges/makerspace-large --judge JUDGE_DIR --out DIR --lock LOCK
```

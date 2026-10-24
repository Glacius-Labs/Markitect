# Library oracle: APPLY-PLAY

This first delivery implements the independent behavior oracle from the
[evaluation contract](evaluation-contract.md)
and a bounded driver for the ten operations in the
[callable interface](../../testdata/library/m1/model/library/callable-interface.yaml)
of the library model M1.
It uses Python 3.10 or later and the standard library only.

The twelve independent cases cover E01 through E12 and the paired fourth-loan
witness. M1 expects five open loans; M2 expects three. Both expect fourteen
calendar days. The witness sends identical requests and identifiers in both
profiles: the fourth loan succeeds under M1 and refuses without mutation under
M2. A separate opposite-profile witness must disagree at admission, count and
copy availability, while its setup and the first three loans remain correct.
A crash or unrelated failure cannot count as that discrimination.
Positive wire cases preserve padded identifiers, names and titles exactly and
distinguish padded/unpadded and Unicode composed/decomposed identifiers.

`oracle.py` authors cases and assertions independently of generated code and
tests. `expectations/m1.json` and `m2.json` hold the explicit policy controls.
`driver.py` implements only sequential process transport. `assess.py` combines
them into a behavior report. No file here implements library operations,
Manager scheduling, review, retry, publication or an alternate Apply.

## Self-check

From the repository root:

```text
python -B -m unittest discover -s tools/library-oracle -p "test_*.py" -v
```

Self-tests materialize controlled expected transcripts, then replay those exact
rows through tiny process fixtures. These fixtures have no domain state or
operation implementation. Tests also feed wrong policy results, changed dates,
duplicate notices, refusal mutations, wrong correlation, missing or malformed
responses, invalid UTF-8, extra output, nonzero exits and time/output overruns.
Transcript replay demonstrates oracle/transport mechanics, not a realized
library or an Apply result.

Existing `go test ./...` does not discover this suite. The required Linux Quality
job runs the Python command explicitly. The coordinator authorized that single
CI step and its workflow digest update. The declarative Oracle Check linkage in
the shared project model remains a coordinator-owned integration point after
BOOT releases those files; this package does not edit them.

## Assess a future frozen executable

Keep these evaluator files and expectations outside the target and Agent write
workspaces. The caller supplies an already built executable and a separate
working directory containing only its realization. Preparation, build,
environment isolation and source/history capture belong to the future product
runner. This driver starts only the supplied local executable; it starts no
Agent, provider, build or network command itself.

Print the evaluator and expectation hashes without running a program:

```text
python -B tools/library-oracle/assess.py --hashes
```

Freeze a JSON manifest outside the target. This shape shows required fields;
replace the illustrative paths, identities and hashes with actual values:

```json
{
  "experiment_id": "fixed-experiment-identity",
  "artifact_id": "A1",
  "model_digest": "actual-compiled-M1-digest",
  "source_tree": "actual-frozen-realization-tree",
  "profile": "m1",
  "command": ["/absolute/assessment/library"],
  "cwd": "/absolute/assessment/realization",
  "program_sha256": {"/absolute/assessment/library": "actual-sha256-hex"},
  "protected_sha256": {
    "assess.py": "actual-sha256-hex",
    "driver.py": "actual-sha256-hex",
    "oracle.py": "actual-sha256-hex",
    "expectations/m1.json": "actual-sha256-hex",
    "expectations/m2.json": "actual-sha256-hex"
  },
  "limits": {
    "request_seconds": 2,
    "process_seconds": 30,
    "request_bytes": 16384,
    "output_bytes": 262144
  }
}
```

Use platform-native absolute paths, with JSON escaping on Windows. Literal
argument files must also appear in `program_sha256`, a mapping from absolute path
strings to 64-character SHA-256 hex strings. Inputs are rehashed before and after
assessment. The CLI reads and retains exact manifest bytes as Base64 and binds
them with `manifest_sha256`; formatting and line endings remain part of that
identity. `manifest_semantic_sha256` separately hashes normalized parsed JSON.
Calling `assess()` with only a parsed object leaves the literal byte fields null;
it does not claim a literal input binding. Generated case data has its own digest.
Malformed manifest input retains its bytes and a failed report. Model/source
identity strings are caller-supplied bindings; this
slice cannot verify production compiler, registry or history identities.

```text
python -B tools/library-oracle/assess.py --manifest /absolute/manifest.json --output /absolute/new-assessment.json
```

The output must be a new path: an earlier attempt is never overwritten. Exit zero
means the selected behavior cases and opposite-profile discrimination passed
with intact byte bindings. Every case starts a fresh process. Failure stops the
affected primary case, records missing steps and preserves its raw trace; later
independent cases still run. The opposite witness continues across expected
assertion mismatches to inspect count/availability and the original loans.

The JSON report includes case/step/assertion counts, per-step findings, exact
request/response bytes and stdout/stderr in Base64, exit codes and integrity
failures. Output bytes are capped across stdout and stderr per process; a
truncated trace is a failure. Request and process deadlines include blocked
writes, responses and EOF. Limits are finite and checked before launch. There
is no repair or retry. Primary counts and the separately recorded discrimination
counts remain separate.

## Evidence still required

Passing this oracle is one behavior evidence class. Generated tests, independent
source/diff and usage review, product verification, registry/history inspection,
guarded publication and human acceptance need their own fixed evidence.
Preserving existing loans above M2's limit is a source-review obligation: inspect
the actual A2 admission, inspection, return and initialization code. A fresh M2
process cannot create four open loans. This interface therefore provides no
live-upgrade, migration or restart-persistence evidence.

The child receives a filtered environment with only Windows runtime variables
where needed. This is not a sandbox: filesystem access and network access are
not restricted here, and cleanup kills the direct child only. The later Linux
runner must enforce protected mounts, credentials/network isolation and process
containment before running Agent-produced code. Hash checks detect changed
bytes; they do not prevent access or establish quiescence.

Full preparation/freeze, immutable A1/A2 capture, consistent history export and
real R1/R2 execution wait for the rebuilt product ports. The
[Playground contract](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/design/canonical-model/manager-apply-playground.md)
owns that sequence. This package supplies no successful Apply simulation using
the old CLI and makes no product-benefit claim.

## Container harness

`container/` builds an Agent-produced Library candidate offline and runs this oracle against it in a hardened container. The candidate is Agent-produced code, so both steps run non-root, without capabilities or network, with a read-only root filesystem and process and memory bounds. The built program stays read-only, and the report goes to standard output, out of the program's reach.

| File | Use |
| --- | --- |
| [container/assess.py](container/assess.py) | Builds a realization's `./cmd/library` in the example Go runtime image and runs the oracle from `tools/library-oracle` at a fixed Git revision. |
| [container/run_oracle.py](container/run_oracle.py) | Runs inside the assessment container. It binds the manifest to the program hash and the Host-computed evaluator hashes, runs the oracle in-process and prints the report to standard output. |

Build the images from the repository root first:

```text
docker build -f packaging/runtime/Dockerfile -t markitect-runtime:dev .
docker build -f packaging/runtime/examples/go/Dockerfile --build-arg BASE=markitect-runtime:dev -t markitect-runtime-go:dev .
python -I tools/library-oracle/container/assess.py --realization PROJECT_OR_MODULE --profile m1 --out OUT --label A1
```

`MARKITECT_ENGINE` overrides the container engine executable. Exit 0 means every oracle case passed, including the opposite-profile discrimination. Agent-produced code and transcripts never enter the repository. The harness records the evaluator revision, not a verified model digest, and covers behavior only; source review, history and registry inspection remain separate evidence. The evaluator export at the fixed revision includes the whole `tools/library-oracle` folder.

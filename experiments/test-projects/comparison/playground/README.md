# Comparison source playground

This small source preparer implements the packet checks described in
[Measurement](../../../../docs/measurement.md#comparison-factors-and-first-results).
It previews fixed Git inputs and declared byte replacements. It runs no
application, Agent, provider, container, Apply, oracle or assessment.

[factors.yaml](factors.yaml) describes the factors and unbound future run inputs.
[scenarios.yaml](scenarios.yaml) binds four selected cases:

| Case | Initial application and selected requirements |
| --- | --- |
| `library-build` | Empty application; full Library r2 initial model and ordinary requirements. |
| `library-limit-three` | The full requirements with the exact five-to-three policy change in both representations. Actual prior application results are still required. |
| `readinglog-existing` | The genuine existing add/list application and public baseline test; full baseline requirements only. |
| `readinglog-seeded` | The same existing start with exactly one declared normalization/identity defect applied identically to both arms. |

The manifest is operator material. Do not give it, this tool, seed descriptions,
patch locations, source provenance or evaluator material to either writer. A
writer receives only its current requirements, neutral instruction and actual
initial application. The seeded writer instruction is simply to restore
conformance to the visible requirements. It does not identify the cause or
mutation location. Later Readinglog features are excluded; stages 01 to 03
remain a follow-up after their complete ordinary packets are available.

The [ordinary baseline requirements](../readinglog/00-baseline/requirements.md)
and [neutral single Agent prompt](../readinglog/00-baseline/implementation-prompt.md)
carry all eight Definitions of the actual integrated Readinglog `0.1.0` snapshot
at `962f17468ae1b3558e042a0429888ece59106545`. Its application and public test
are byte-identical to the immutable original source.
The earlier fixture and ordinary wording left the error response shape too
open. Both now require exactly one JSON object with only a nonempty string
`error` field. This makes both requirement representations more precise; the existing application,
public test and known input-error gap remain unchanged, without a behavior claim.

`readinglog-existing` means **no added seed**, not a claim of correctness. The
original application's argparse usage errors differ from the public JSON error
contract. Both arms receive the same existing bytes and the same visible
requirements; that known difference is recorded separately from the extra seed.
The seed replaces only a duplicate guard's comparison with the raw argument,
while keeping input normalization. Before and after hashes are bound. This is
source preparation of a defect, not evidence that an assessment detected it.

## Inspect committed sources

The two `.yaml` inputs use the JSON subset of YAML 1.2, so the standard-library
tool needs no YAML dependency or custom parser. Output uses the same subset.

```text
python -B experiments/test-projects/comparison/playground/preview.py --repo . --revision FULL_COMMITTED_SHA --scenario library-build
python -B experiments/test-projects/comparison/playground/preview.py --repo . --revision FULL_COMMITTED_SHA --scenario readinglog-seeded --require-sources
```

Use a full forty-character commit SHA. The tool reads its manifests from that
commit, verifies each declared raw Git blob hash and file mode, checks exact
byte-patch preconditions and postconditions, and prints each arm's target
inventory and hashes. It writes no files and accepts no command to execute.
Every file shared between the arms has the same final target bytes. Stage
selection includes only its named snapshot; later intent is never copied.

`sourcePrepared: true` means the selected packet's initial application and
requirements are source-bound. It establishes no runtime readiness or quality.
Missing sources, hashes or actual prior results produce explicit `UNBOUND`
source blockers. `--require-sources` exits 3 for such an incomplete packet;
malformed inputs or a hash/patch mismatch exit 1. `--require-ready` always exits
3: runtime/profile observations, finite grant and independent assessment are
outside this source tool. The result names the missing run inputs and never
reports a ready run. An ordinary inspection exits 0 even when it reports UNBOUND.

For materialization, extract the named blobs, preserving their modes and paths,
and apply only the declared exact byte replacement in each selected arm. Verify
the printed final hashes before a future run. The source preparer does not
write a workspace or manufacture a prior implementation. For an isolated pair,
the initial application bytes/state must match. In a longitudinal sequence,
only the first stage shares a starting application; later stages continue each
arm's own frozen prior result. The Library policy change therefore remains
blocked on actual first results. Never reset either arm to a supplied solution.

The four cases are a curated control set, not a full factor combination. Go
Library and Python Readinglog differ in domain, language, size and relationships;
their difference cannot establish a causal complexity or size effect. Primary
comparisons pair the methods within the same scenario and actual model profile.
The independent assessment must use visible obligations, check the known seed
as a positive control, and accept other valid implementations. Its hidden tests
and expected answers remain independently owned and outside this directory.

The [historical case harness](https://github.com/Glacius-Labs/Markitect/tree/dc4910b9810c5aaa4deecf1ff74734ee97badc13/experiments/case-playground)
is retired. Only exact public domain/application blobs are referenced here;
its driver and evaluators are not revived. The unchanged
[Library oracle](../../../library-playground/README.md) retains historical M1/M2
scope and is not a complete Library r2 evaluator: r2 adds the explicit due-date
overflow refusal. This playground contains no hidden assessment implementation.

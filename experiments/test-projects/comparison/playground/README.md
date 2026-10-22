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
mutation location. The four existing source scenarios exclude later Readinglog features. The
separate two-stage pilot proposal below adds a complete ordinary stage-01
packet; it does not extend this preparer or bind a continuation workspace.

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


## Proposed Readinglog paired pilot, 22 October 2026

**Preparation only: `runReady: false`, execution grant: none.** This is the
small proposal commissioned after SOL3, under the existing
[measurement procedure](../../../../docs/measurement.md#quality-first-comparative-evaluation).
It does not extend SOL3, NIGHT or any other exhausted grant. Coordinator chat
`01a157ed-1baf-7401-a1d0-9dc760529eea` carries the owner's Overview instruction
`01a16306-9f55-7942-ab35-2362650eced3` and the bounded preparation assignment.
Coding and independent preparation review use Sol High; the proposed actual
comparison uses **native `gpt-6-luna`, reasoning effort `high`**, in both arms and
every Markitect role. Model availability or configuration is not an inference
or native continuation proof.

The smallest useful trajectory is the actual existing Readinglog application,
first restored to complete 0.1 intent, then extended to complete 0.2 intent.
Both stages are mandatory. This tests preservation through one feature change;
it cannot establish sustained coherence, multi-area integration or superiority.
The Library r2 plus Readinglog alternative needs eight dispatches (six hours
at the proposed cap), an additional runtime and r2 overflow assessment. It is
deferred; this proposal creates no new harness, core engine or factor matrix.

### Fixed tasks, starts and order

| Dispatch | Arm and current task | Inputs |
| --- | --- | --- |
| 1 | Ordinary: restore 0.1 conformance | Genuine add/list application and public baseline test; complete ordinary baseline requirements. Fresh native coding conversation. |
| 2 | Markitect: restore 0.1 conformance | Identical application/test bytes and initial state; complete canonical baseline snapshot and current Apply controls. Fresh Run and role conversations. |
| 3 | Markitect: implement 0.2 finish/status | Its own actual published stage-0 result, tests/docs/registry and carried data; replace only current intent with complete 0.2 snapshot. New Apply Run. |
| 4 | Ordinary: implement 0.2 finish/status | Its own actual stage-0 result, tests/docs and carried data; complete ordinary 0.2 requirements. Native continuation of dispatch 1. |

This fixed O/M then M/O order is one trajectory per arm, no repeats, sweeps,
seeded variant or model comparison. The ordinary arm is one coding Agent with
normal reading, implementation, testing and revision tools; no supplied second
reviewer or Manager organization. Markitect uses its current product process,
including mandatory assessment, internal correction and fresh Verify. Both may
debug, test and revise inside their one bounded assignment before submission.
No external findings, human repair, reassurance review or corrective prompt is
provided to either writer before the whole trajectory ends. Stop/control and
intent delivery are operator actions and are logged separately from assistance.

The initial packet is the existing `readinglog-existing` scenario, not
`readinglog-seeded`. Its application and public test come from fixed commit
`962f17468ae1b3558e042a0429888ece59106545`, as bound by `scenarios.yaml`.
There is no initial database; a missing file is empty state. The known initial
argparse/JSON error-contract gap remains identical in both arms. It is not a
hidden extra requirement, an added seed or a conforming baseline claim.

Use [ordinary 0.1 requirements](../readinglog/00-baseline/requirements.md) and
its [neutral instruction](../readinglog/00-baseline/implementation-prompt.md)
against the canonical
[0.1 snapshot](../../readinglog/snapshots/00-baseline/model.yaml).
Stage 1 uses the complete
[ordinary 0.2 requirements](../readinglog/01-finish-status/requirements.md) and
[neutral instruction](../readinglog/01-finish-status/implementation-prompt.md)
against the complete
[0.2 snapshot](../../readinglog/snapshots/01-finish-status/model.yaml).
The latter includes every earlier obligation, not only a feature delta.

| Canonical 0.2 Definition | Ordinary 0.2 coverage |
| --- | --- |
| system | Opening scope and Delivery: one operator, existing CLI, preserved add/list interface, implementation freedom. |
| book, inputs | Books and shared input rules: exact fields/types, normalization, identity, unread creation, status-only finish. |
| storage | Persistent state: validation, exact shape, atomic changes, error preservation, read-only behavior and no concurrent-writer guarantee. |
| add | Add: normalized unique identifier, direct record output and preservation. |
| list | List: all records by default, exact status filter, envelope, order and read-only behavior. |
| finish | Finish: normalized exact ID, status-only update, idempotent success and missing/unknown ID refusal. |
| interface, realization | Command interface and Delivery: Python 3.13, JSON/error/exit contract, tests/docs and honest limitations. |

Continue each arm's actual prior artifacts and data, including defects; never
supply a gold implementation. To exercise data carry-forward, after freezing
stage 0 use a separate continuation copy and the same ordinary CLI operations
in each arm: add `alpha`/`First`/`Reader`/10 pages, then
`beta`/`Second`/`Reader`/20 pages, using explicit `--db continuation-db.json`.
Preserve actual output, exit codes and resulting bytes, including a missing or
faulty result. Do not synthesize successful records or repair data. These are
fixed operator lifecycle interactions, not a writer hint or oracle feedback.
Bind both frozen submission and derived continuation hashes before stage 1;
keep stage-0 freeze immutable. Preserve any other arm-owned data too.

An absent usable prior application makes the next stage dependency-blocked
and the sequence unsuccessful; record the unused slot, never convert it to a
retry. A runnable but faulty prior application continues as-is. Unverified
Markitect retained work is evidence, not a published continuation result.
Neither native session history nor writer packets receive future intent,
operator metadata, seed locations, oracle code, expected answers or comparison
results. Stage 1 receives only its current full intent and its own prior work.

### Submission, assessment and primary decision

Freeze the first declared final submission or technical ending immediately:
artifact paths/modes/content, state, source/model/profile/image identities,
provider session IDs, actual final text, checks and lifecycle evidence. For
Markitect bind the actual published target separately from any unpublished
candidate/retained work. For the ordinary arm require a native completed input
Turn with its final message, exit result and captured files; prose alone cannot
substitute for missing transport completion. No assessor feedback reaches either
writer between stages. Later assisted results, if separately commissioned,
remain separate from this primary first-submission record.

Use the existing independently owned Readinglog oracle stages 0 and 1,
cumulatively, plus independent semantic assessment of code, tests, documentation,
state preservation and unnecessary changes against the same visible intent.
Keep hidden evaluator materials outside writer mounts, contexts and reachable
hosts. Assess only immutable copies in a separate bounded, credential-free,
network-free assessment environment, with no live writer or writable source
mount. Oracle percentages alone cannot establish complete conformance. Record
its actual coverage and omissions; an assessment/infrastructure failure is
unknown or incomplete, not an invented semantic failure or pass.

The primary outcome is **0 or 1 completely conforming and completed sequence
per arm**: both mandatory stages must complete on their first submissions, with
zero critical errors, zero false success claims, zero false blockers and no
unnecessary changes. A missed binding behavior, data loss/corruption, incompatible
public interface, materially wrong test/documentation claim or unsupported success
is critical. A correct complete packet has no expected intent blocker; a claimed
missing requirement that is actually supplied is a false blocker. Technical
incompletion remains separately classified and makes the complete-sequence
criterion fail. Assess necessary repairs and evidence separately from optional
cleanup; permission to change a path does not make every change necessary.

Report each stage descriptively: required behavior and failures, semantic test
quality, documentation consistency, preservation, optional changes, false claims,
false blockers, internal detection/repair versus external discovery, actual
publication and full stop/cleanup. Do not weight or average away critical
errors with green checks. One additional complete sequence with no new critical
error type is a practically meaningful descriptive pilot difference. With one
sequence per arm it is not a reliability, statistical advantage or causal
mechanism claim. Interpret ties, failures and missing evidence explicitly.

### Proposed finite envelope and attention accounting

The budget proposal is **four dispatches, each at most 2700 seconds (45 minutes),
10800 seconds (three hours) total**, including setup, failed starts, inference,
stop, assessment and cleanup. Every reserved start consumes its full slot;
unused or failed slots are never retried or reset. The first actual baseline
dispatch must also supply the real native transport evidence. There is no free
generative preflight, smoke or continuation retry. A failed initial transport
consumes that dispatch and may block its dependent stage.

Propose identical whole-arm ceilings: 2 CPUs, 4 GiB RAM, 512 processes, 2 GiB
writable disk including temporary/provider state, and 32 MiB retained output.
These apply in aggregate to all simultaneously active Agent, host, workspace
and check processes belonging to that arm, not independently per container.
Freeze exact tool versions, Python 3.13 runtime and immutable image IDs,
permissions, workspace access and provider egress equally. Only required
Markitect model/registry and Manager/MCP machinery differ as the treatment.
Do not grant Markitect additional aggregate resources through more containers.
The external assessor gets the same separate limits per submission for both
arms; its work/time is included in the dispatch and recorded separately.

Within each slot, stop inference by minute 40 and reserve the remaining five
minutes for freeze, stop, assessment and cleanup, including at least three
minutes for stop/cleanup. Reserve twelve of the first forty minutes for
Markitect's closing Verify. The ordinary Agent may use the same forty-minute
inference allowance for its normal implementation and checks. Exact phases,
UTC gate window, quota floor and absolute final deadline must be source-bound
before release; no wall-clock window or quota grant is created here.

Start human-attention accounting before authoring/maintaining either requirement
representation and record preparation, parity review, configuration, operation,
model/doc updates, intervention and recovery. Existing historical authoring
costs without measurements are unknown, never zero. Keep evaluator/researcher
work separate from the effort an everyday owner would need to decide, model,
operate, inspect or repair. Record waiting separately from active attention;
record actual wall time, resource use, raw provider usage and billing evidence
separately. Unknown costs stay unknown. More observable Markitect records alone
do not demonstrate lower owner effort.

### Actual readiness and smallest remaining work

Preparation base is integrated Main
`d88d7768fc84d2c8c31880988f1d51a716461452`. The two ordinary stage-01 documents
are source preparation; existing scenario manifests and source previews still
cover only the original four scenarios. They do not bind stage-1 actual prior
results or establish runtime readiness.

| Boundary | Current evidence and next binding |
| --- | --- |
| Requirement parity | Complete ordinary 0.1/0.2 documents and canonical snapshots exist. Independent complete-diff/semantic review must bind their exact integrated revisions before a run. |
| Initial sources | Existing `preview.py` can bind `readinglog-existing` fixed blobs and shared bytes. Freeze current source commit, packet hashes and actual initial repository/tree before release. |
| Native ordinary continuation/finish | **UNPROVEN.** Existing `plan_arm.py` accepts D-series `plan.md` packets and a closed fresh `codex exec` template. It does not demonstrate ordinary documentation packets or native resume across these stages. Inspect pinned CLI capabilities without inference; propose only the smallest existing-driver change before implementation. First real transport evidence must consume an authorized dispatch. |
| Assessment isolation | **UNBOUND.** `oracle_kit.py` invokes submitted code directly with inherited environment and captured output. It is not a safe host assessment boundary. Reuse the bounded, credential-free container pattern in `night_run.py`, adapting only the Readinglog entrypoint/paths after separately agreeing its exact small diff; do not run submitted code on the operator host. |
| Whole-arm resources | **UNBOUND.** Existing per-container CPU/RAM limits do not prove aggregate enforcement, disk/output bounds or parity. Bind a common enforced arm envelope and its observation/cessation evidence before claiming execution readiness. |
| Profile/runtime | **UNBOUND for this pilot.** SOL3 observed native Luna High in Markitect, not the ordinary arm's resume/finish. Pin dedicated auth, actual resolved role settings, CLI/tools/images and equal permissions; no fallback or desktop-auth substitution. |
| Own-result stage 1 | **UNBOUND by design** until each stage-0 submission exists. Hash actual artifacts, continuation data and native ordinary session state; do not manufacture a prior result. |
| Grant/release | **NONE.** A reviewed proposal and source preparation do not dispatch. Coordinator requests a fresh finite owner grant only against the concrete reviewed plan, then freezes the actual UTC/source gate and append-only ledger. |

SOL3 is closed: P met the bounded preservation task; R repaired behavior but
failed because of optional changes; W was technically incomplete when three
Implement child turns made no report-MCP submission. Extensive report-schema
friction remained in P/R. These are readiness risks, not solved by this proposal
and not evidence of comparative benefit. Preserve the original scores and
attempts. No blind next start follows from their completion.

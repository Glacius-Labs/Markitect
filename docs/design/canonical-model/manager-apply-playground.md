# Playground contract for two real library Applies

This contract owns the external experiment for gates P02 and P10 of the [acceptance gates](manager-apply-implementation-plan.md): frozen inputs, the actual product invocation, immutable capture of A1 and A2, independent evaluation and the report.

## Three responsibilities

| Owner | Responsibility |
| --- | --- |
| Markitect | Compile complete v1 inputs, construct context/mandates, run the ordinary root and Package Managers, delegate through the Host, review, integrate, verify, publish and journal. |
| Generated library | Implement M1, including its [callable interface](library-example/m1/model/library/callable-interface.yaml), with automated tests and concise usage documentation in its owned realization tree. |
| External Playground | Prepare frozen inputs, invoke the actual product, capture fixed outputs and history, build/run the library and apply the [independent expectations](library-example/evaluation-contract.md). |

The Playground never schedules Managers, writes their prompts/reports, performs their review loop or supplies a replacement Apply. It supplies one initial outcome instruction to the ordinary root through the product. It cannot silently edit generated code, repair the registry or classify a failed run as successful. The later [working-day scenarios](../../concepts/reflections/working-day-demonstration-20261016.md) stay separate: the primary library case needs no artificial owner questions, crash injection or budget exhaustion.

## Frozen experiment inputs

Before any real session, create a manifest with exact file hashes and literal command arguments. Required bindings are:

| Input | Binding |
| --- | --- |
| Product | Markitect source revision, executable bytes/hash, build recipe/toolchain and supported public invocation arguments. One build for both Applies. |
| Model | Complete [M1](library-example/README.md), dependency sources and lock; deterministic M2 patch and its expected compiled digest. Production compilation validates both before dispatch. |
| Execution | Effective project/host profiles, provider executable/schema capability identity, literal model/effort, custom instructions, check definitions and declared environment/tool inputs, all as required by the configuration contract. Secrets are supplied separately, never copied into reports. |
| Authority and budgets | Explicit authorization for both Applies, mandatory review and integration policy, finite caps and completion reserve per Apply, and output limits. Values are reviewed and frozen before launch; examples in design documents are not authorization. |
| Assessment | Evaluator source, independent M1/M2 expectation sets, case data/seed, oracle self-checks and external request/build/output limits. |
| Environment | Linux reference environment/image identity, exact Go/Git/Python and provider versions, dependency/cache provenance and enforced mounts/process permissions. A Windows diagnostic run is separate evidence. |
| Bootstrap | Exact initial repository bytes/commit, ignore/classification inputs and approved root instruction. No domain implementation, prewritten complete tests or hidden registry entries. |

The exact product argv is frozen once the rebuilt CLI exists; this design does not advertise new flags on the current CLI. Bootstrap uses the single new v1 project path, not the legacy schema. The callable interface is part of M1, so it reaches the Managers through ordinary model context; there is no separate realization constraint outside the model.

Expected policy values are independently authored as five/three and fourteen, not generated from the implementation. No private expectation source is given to the product. Public requirements remain visible: withheld examples must never introduce secret business rules. Hashes bind bytes, not owner acceptance or model completeness.

Changing a frozen product, profile, model or oracle input after seeing results creates a new experiment identity. Preserve the previous evidence. The planned M1-to-M2 patch is the sole intended input variation within this sequence.

## Physical boundaries and ownership

Use a new, bounded `experiments/library-playground/` runner when implementing P02. Reuse independent capture/container utilities from the [existing Playground](../../../experiments/case-playground/README.md) selectively; its old commands, comparative arms and model format do not define this scenario. The runner and evaluator belong to the experiments owner, outside the temporary adopting project.

An external experiment directory contains:

```text
experiment/
  manifest.json                 frozen bindings and command arguments
  protected/                    evaluator and independent case/expectation files
  workspace/library/            real target repository, preserved between Applies
    <canonical model inputs>    M1, then only the declared M2 patch
    <execution inputs>          unchanged approved profile and public constraint
    realization/                Agent-produced Go code, tests and usage docs
  state/                        product-owned journal, outside governed source
  evidence/
    preparation/
    r1/                         invocation, A1 snapshot, registry, history, assessment
    m1-to-m2.patch
    r2/                         invocation, A2 snapshot, registry, history, assessment
    comparison.json
    report.md
```

Angle-bracketed entries denote the rebuilt product's supported paths, not proposed alternate manifests. Freeze their actual locations before the dry run. Each namespace has one writer: the runner owns manifests/scenario preparation/evidence; Markitect owns its private journal and controlled artifact/registry publication; role sessions own only their claims in candidate workspaces; the external evaluator writes assessment records, never the target repository.

Agent/provider processes receive only the inputs and candidate workspaces their product mandate permits. The generated library runs in a separate assessment environment without provider credentials, network access, protected oracle mounts or writable A1/A2 sources. The evaluator drives its standard streams from outside that environment. Build/test scratch and outputs are separate from frozen source. Record actual enforcement and any isolation limitation. Do not confuse scoped instructions or before/after hashes with a sandbox.

## Continuous two-Apply sequence

1. **Prepare.** Create an empty temporary target from the frozen bootstrap, with M1, dependencies, execution inputs and public constraint. Compile both M1 and the planned M2 with the production compiler, confirm the single-field delta, then leave M1 selected. Verify oracle integrity, tool identities and finite authorized budgets. Record preparation/context without making semantic decisions for Managers.
2. **Apply R1.** Invoke the real product with the approved assignment and M1. Markitect starts the root, delegates to Package Managers and fresh reviewers, integrates and publishes by its own contracts. Record actual invocation, process output/exit, observed run identity and terminal outcome. A process exit alone is not a successful Apply.
3. **Freeze A1.** Require a confirmed Applied outcome and exact publication read-back of artifacts plus registry. Capture an immutable source snapshot and a consistent product-supported history export. Bind model/config/source/candidate/registry identities. Do not copy a live SQLite file without a consistent export/backup. Establish that all sessions are terminal before external assessment. The runner then commits the published A1 as the experiment's acceptance, with the run's proposed `Markitect-Run` and `Markitect-Model` trailers.
4. **Assess A1.** Build from the frozen realization copy, run its generated tests, execute all independent M1 cases and the dedicated M2 discrimination witness, then inspect code, usage docs, registry and communication evidence. Store raw results and an A1 review record. If required checks fail or are unassessed, stop the principal sequence; do not apply M2 to conceal the failure.
5. **Change intent only.** Against the actual A1 repository, edit only `spec.maxOpenLoans` from five to three in `lending/loan-policy`. Do not change code, generated tests or docs, the registry, profiles or instructions. Verify their byte identity, commit the model patch on top of the A1 commit and record the resulting source snapshot and M2 digest. R2 requires this clean, committed state.
6. **Apply R2.** Invoke the same product build/profile against this source as a new Apply, with fresh assignment, run, task and review identities. R1 remains Applied in history. Markitect may inspect/reuse/rewrite A1 artifacts under the ordinary reconciliation rules; the runner does not choose the files or supply a replacement implementation.
7. **Freeze and assess A2.** Require confirmed publication and capture as for A1. Run generated tests and all independent M2 cases, plus the dedicated M1 discrimination witness. Review the actual A1-to-A2 diff and current source/registry against M2. Execute unchanged families against both artifacts; investigate collateral changes. A large rewrite is reviewable evidence, not automatically an error or automatically good reconciliation.
8. **Compare.** Produce the final bound report, with separate behavioral, source-review, product-execution and human-acceptance results. Recheck protected inputs/hashes, preserve every failed attempt and ensure no live Agent remains. Retain A1 and A2 and their build recipes so assessments can be repeated without launching Agents.

If a run asks a question during the baseline sequence, the runner records it as an owner intervention. A question about a rule M1 already resolves counts as a defect, an unnecessary escalation. Each Apply has its own frozen caps, and the report shows the consumption of both. No review approval carries over from R1 to R2. A budget failure is an incomplete experiment with recorded evidence. Owner questions and authorized model edits follow the product's journal contract; the runner pauses for real owner interaction instead of inventing answers. That interaction is not required to occur in this unambiguous baseline case.

## Independent assessment boundary

Build A1 and A2 separately from immutable realization trees; never build A1 from the mutable R2 working directory. A test scenario starts a new library process and arranges its own data through public operations. R2 evolves A1's code, tests, documentation and registry. There is no transfer of a live A1 library's runtime data into A2.

The decisive witness is the same fourth-loan request after three successful loans: A1 accepts it, A2 refuses it without mutation. Full common behavior remains required. The [evaluation contract](library-example/evaluation-contract.md) details assertions, source mapping and negative checks. It explicitly separates runtime evidence from source-review evidence for preserving existing loans above the new limit; this interface cannot claim live-upgrade/migration proof.

Generated test results, external black-box assertions, code review, product verification and owner acceptance remain distinct. The external evaluator is not another Manager or product verifier. The owner or a separately authorized reviewer inspects the fixed source and reports concrete findings; the generating session's own assertion is not independent review. No statistical product-benefit claim follows from one library/two-Applies example.

## Report and failure contract

Write machine-readable `comparison.json` plus a short `report.md`. Each assessment result identifies experiment/run, model digest, source/artifact tree, registry, library executable, evaluator/expectation hashes, environment, executed case/assertion counts and any missing evidence. Keep raw request/response traces and stdout/stderr alongside it. Human acceptance is explicitly `not-recorded`, accepted or rejected with its own provenance; passing mechanics does not fill it in.

The readable report answers:

- What did A1 and A2 actually implement, and did each match its selected model?
- Did the fourth-loan witness change exactly as expected, while common rules still held?
- Was every cross-Package obligation of M1 realized? The report lists them, such as suspension enforced by lending, availability released by returns and documentation for every area, with the evidence for each. This makes "nothing forgotten" ([PRM-001](../../concepts/register.md#promises)) visible for this model.
- Which files/Definition assessments changed, and did the model-only patch precede R2?
- Which real sessions, child prompts, messages, rework rounds and integration steps produced the result? Was the final evidence check bound to the published aggregate?
- Were original inputs and protected expectations intact, and were cleanup and publication confirmed?
- What was reviewed rather than executed, what remains unproven, and what needs an owner decision?

Use separate failure categories: `preparation/environment`, `apply/execution`, `realization/behavior`, `realization/source-review`, `assessment/integrity` and `incomplete-evidence`. Preserve the causal sequence and mark later stages `not-run` when blocked. Failed builds/generated tests, contract violations and wrong library behavior cannot disappear behind green product reports. An oracle or environment failure is not automatically a Markitect semantic failure. An unknown publication or Agent termination is not a clean completion.

The runner performs no automatic repair/retry to improve a reported outcome. A further Apply against unchanged M1, an oracle correction or a fresh authorized run is recorded explicitly with its own identities and prior failure. It does not replace the original attempt or silently restore the two-successful-Apply claim.

## Delivery gate

The oracle and its self-checks exist before the product. The runner's preparation, capture, transport driver and oracle cases are then proven by a clearly labeled scripted dry run: hashes, the exact M2 patch, immutable A1 and A2 bookkeeping, wrong-result detection and failure preservation. Synthetic outputs never count as Agent communication or Apply. The real demonstration additionally needs complete v1, the rebuilt product and a frozen, actually supported provider profile ([rebuild plan](manager-apply-rebuild-plan.md#bind-the-executable-before-real-work)).

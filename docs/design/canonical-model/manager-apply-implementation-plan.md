# Implementation plan for the first reliable Apply

Date: 16 October 2026. Proposed implementation sequence for the [Manager Apply concept](manager-apply-concept.md), requested in the [owner discussion](sources/manager-realization-20261016.md#derive-the-implementation-plan). This plan defines future work and acceptance criteria; none of its implementation gates has been run or passed by writing this document. A00 to A19 are local plan identifiers, not assigned backlog packages or active reservations. The repository [roadmap](../../implementation-plan.md) and [backlog](../../work-items/backlog.yaml) keep their status ownership.

## Outcome and scope

The first target is an actual artifact-only Apply on a small library project using the complete canonical language v1. It must reconcile existing code, tests and documentation against unchanged model intent, through root-led Package Managers, independent reviews, aggregate verification, guarded publication and observable read-back. It must also produce a justified no-change result and stop honestly on drift, missing evidence, unresolved questions and partial failure.

Use one local model, read-only imported vocabulary, one Manager per Package, one ordinary root, a Manager plus fresh reviewer per active area, and sequential writing. Keep team configuration explicit. Optimize, custom commands, Government, multiple artifact hierarchies, semantic indexing, broad incremental reuse and general parallel scheduling are outside the first delivery. No language subset is advertised as full v1 support.

There are two completion points: **library Apply proven at A17** and **the product switched to the new source model with the old execution path retired at A19**. This distinction makes the transition finite without requiring a replacement of every existing feature before the new path can be evaluated.

## Current implementation assessment

Source inspected: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. The concept and this plan are documentation changes on a branch based on that source. These are code-reading findings, not a new full-suite, provider or release result.

| Area | Current evidence | Treatment |
| --- | --- | --- |
| Structural compiler | [Core values](../../../src/internal/core/types.go) use `apiVersion`, Kind, namespace and name; [Compile](../../../src/internal/core/compile.go) consumes separate Schemas and Definitions. | Preserve behavior while building the v1 language separately. Do not claim these values already represent v1 Package/id, standalone Property, body-link and canonical-byte semantics. |
| Fixed project vocabulary | [Project Schema](../../../src/internal/core/projectmodel/projectmodel.go) fixes Manager, Statement, Artifact, Check and Decision. | Retire as the canonical authoring contract after cutover. Manager topology and artifact registration become runtime projections. |
| Project acquisition | [Project loading](../../../src/internal/application/projectwork/project.go) calls `core.Compile` with `projectmodel.Schema()` and [project values](../../../src/internal/core/projectmodel/projectvalue/types.go) bind the current project manifest and reports. | Add a v1 frontend over explicit acquired inputs. Keep filesystem and Git access outside pure compilation. |
| Runtime coupling | [Host](../../../src/internal/runtime/projectrun/types.go), [plan basis](../../../src/internal/runtime/projectrun/plan_basis.go), work, review and verification consume the old Project and Report values. | Introduce one bounded execution-input contract, then move consumers to it in successive slices. A new parser alone cannot deliver the new Apply. |
| Method and execution | [Run coordination](../../../src/internal/runtime/projectrun/run.go), review, [Invoker](../../../src/internal/runtime/projectrun/types.go), [role profiles](../../../src/internal/runtime/projectconfig/types.go), candidate workspaces and recovery already exist. | Reuse these foundations; adapt their input and lifecycle contracts rather than introduce another scheduler or provider launcher. |
| Coverage | [Coverage](../../../src/internal/core/projectmodel/projectcoverage/coverage.go) classifies a supplied census using the present project realization contract. | Reuse classification and snapshot mechanics; adapt ownership and realization inputs to the new registry. Preserve conservative handling of unexplained paths. |
| Verification and publication | [Verify](../../../src/internal/runtime/projectrun/verify.go) binds candidate evidence; [Apply](../../../src/internal/runtime/projectrun/apply.go) checks fresh inputs and calls guarded writing. | Reuse the guards and failure semantics. Add explicit aggregate post-publication read-back: the inspected Apply tail records completion after writing without that distinct acceptance stage. |
| Durable storage | [Run storage](../../../src/internal/runtime/projectrun/store.go) uses the shared [record store](../../../src/internal/infrastructure/recordstore/store.go). Native receipts and operational journals already record part of the process. | Extend the domain records with causal task/context/communication events and retrievable payloads. Do not present current partial observations as complete history. |
| Public surface | The [verb table](../../../src/internal/application/projectcli/verbs.go) and [runtime contracts](../../../src/internal/application/projectcli/contract_runtime_test.go) distinguish current guarded `apply` from higher-level `deliver`. | Specify the semantic change explicitly at cutover. Do not rename one command and imply that the new high-level Apply is implemented. |
| Design probes | The [Exchange reports](minimal-language/exchange-example/README.md) are Python design probes; the example retains a rejected Command/command collision. | Use reviewed examples and counterexamples as conformance inputs. Do not embed the probe as a production compiler or copy its invalid graph as the positive library fixture. |

## Transition strategy

Use a staged replacement of the model frontend around one preserved execution method.

1. Keep the present route operational and covered by its existing regression tests while the new compiler and execution contract are built.
2. Give the new v1 frontend its own exact identities, typed values and digest domains. A temporary adapter projects the current frontend into the same execution contract. Do not translate v1 into invented canonical Manager/Statement files or label a v1 digest as an old project digest.
3. Introduce the new route through an explicit project-format selection and internal integration harness. There is no silent format guessing, fallback after a v1 error or shadow acceptance based on the old compiler.
4. Prove the new path on the library, then convert Markitect's own model and dependent tooling in one reviewed cutover. Remove the old writer/loader path after that cutover.

The new pure language implementation can live in a focused Core subtree such as `core/canonical`; Application owns acquisition and v1 project composition. Pure organization/context/registry projections need their own documented ownership. The Runtime execution contract belongs in a neutral Runtime-owned package rather than inside the old project vocabulary. Application converts pure model results into that contract; Core must not import Runtime to construct it. Names are implementation choices to confirm in A00; the permitted layer directions in [modules](../../development/modules.md#dependency-direction) remain binding. Add package/code-map/Artifact ownership with each actual new package.

Keep the execution contract finite: fixed source/model bindings, Manager topology, task/context input, artifact ownership, checks, candidate validation and immutable evidence references. Do not make a general plugin lifecycle or require one universal intermediate language for all future commands. The legacy adapter is a temporary migration seam with A19 as its deletion gate.

Old run records keep their original version, digests and bytes. New runs have an explicit source-format and protocol version; an old verified run cannot authorize a v1 candidate. Keep minimal read-only historical inspection or the ability to open an old report using its producing version. No in-flight run is migrated or replayed automatically. Original releases and evidence remain immutable.

The self-hosted model needs an explicit conversion map, not a mechanical syntax rename:

| Current content | New owner |
| --- | --- |
| Canonical Manager Definitions and `owns`/parent data | Derived Package organization, organization configuration and artifact registry. Preserve any project obligations currently carried in Manager prose as explicit model content rather than dropping them with the runtime role. |
| Statement Definitions | Appropriate project-declared v1 Kinds and bodies, with reviewed references. A universal built-in Statement is not required. |
| Artifact Definitions | Actual paths/owners/model links in the realization registry; intended output requirements remain expressed in the model. |
| Check Definitions and executable commands | Canonical expectations in an ordinary declared vocabulary, plus explicit runtime check bindings and execution configuration. A vocabulary Kind does not gain execution rights by name. |
| Decision Definitions | Current settled intent and rationale remain in project model content; historical source and decision provenance are retained. |
| Current generated project view | A single renderer over the new accepted model and realization view, preserving its source ownership and guarded write contract. |
| Module checks bootstrap packaging and CI consumers | Rebind to current v1 identities and acquired inputs where required; do not retain the old compiler solely as a hidden support path. |

## Library acceptance fixture

A00 defines a small, independently reviewable library fixture before implementation. Its model has catalog, members, lending and lending/reminders Packages. It defines a book copy's availability, member registration and suspension, the five-loan limit, returns and a simple overdue reminder. Connect the relevant Definitions explicitly. Domain vocabulary is declared using ordinary Kinds and Properties; no Library, Rule, Manager or Artifact becomes a privileged language type.

Use a small Go service with a fixed public test interface, an in-memory store, code, normal project tests and a short usage guide. No UI, deployment, network service or production database is needed. Time for reminder assessment is an explicit test input. Concurrency is outside the first domain fixture; the model states that these example operations are processed sequentially. This bounds the fixture without asserting a general product limitation or claiming concurrent-loan correctness.

Keep a frozen external expectation set outside Agent write scope. Run its independent black-box checks against the harvested candidate, not a replacement returned by the Agent. Capture its bytes separately and recheck them before and after execution; any mutation invalidates the result. This is a validated input boundary, not a claim that same-user processes are isolated by the OS. Ordinary project tests remain editable realizations and must be reviewed. The protected expectation set prevents an implementation and its own incorrect tests from agreeing on the wrong rule.

| Case | Independent expected result |
| --- | --- |
| LIB-01 | An active registered member with four current loans can borrow an available copy; the count becomes five and the copy becomes unavailable. |
| LIB-02 | A member already at five cannot borrow another copy; member and copy state remain unchanged. |
| LIB-03 | An unregistered member cannot borrow; state remains unchanged. |
| LIB-04 | A suspended member cannot borrow even when registered and below the limit; state remains unchanged. |
| LIB-05 | An unavailable copy cannot be borrowed; state remains unchanged. |
| LIB-06 | Returning a currently borrowed copy reduces that member's loan count and makes the copy available. |
| LIB-07 | A still-open loan past its due date is included in the reminder result; a returned or not-yet-overdue loan is not, for the explicit assessment date. |

Prepare two artifact trees against the same model: a correct tree and a drifted tree whose implementation, normal test and guide all say six instead of five. The drifted tree's ordinary test may pass, but the independent LIB-02 check must reject it. The harness asserts that expected rejection, so the repository suite itself remains green. The model is unchanged in the corrective Apply.

Freeze a valid initial registry, profile configuration and ignore policy. Also prepare small variations for unknown ownership, a shared-file claim, a missing model link, a worker's attempted model edit, rejected/reworked review, stale inputs and partial publication. Each scenario starts from a new fixed Git fixture; never carry a prior Agent's modifications into the next supposedly identical input.

## Dependency order and milestones

```text
A00 -> A01 -> A02 -> A03 -> A04 -> A05
                                   |
                                   v
             A06 -> A07 -> A08 -> A09 -> A10 -> A11
                                                   |
                                                   v
                         A12 -> A13 -> A14 -> A15 -> A16 -> A17
                                                                  |
                                                                  v
                                                            A18 -> A19
```

This is the recommended serial delivery order. Some compiler and runtime preparation could later be assigned independently, but this document does not reserve zones or dispatch parallel work. Each row below is a reviewable behavior slice. If one slice crosses several independent lifecycle boundaries, split its PR further while preserving its gate; do not combine the whole phase into a large migration PR.

### A00 Freeze the fixture and migration contract

Deliver the library model sources, correct/drifted trees, frozen LIB expectations, fixture runner contract, code ownership map and the finite execution-input shape. Identify the current regression cases that must survive each adapter change. Review the specification and expectations without using the library implementation as the source of truth.

**Gate:** the two artifact trees differ only in the intended drift; the frozen oracle accepts the correct tree and rejects the known erroneous behavior; model bytes are equal. Every planned new path has an owner. No provider is invoked. The existing source baseline is recorded; a fresh full source result is required before the first runtime mutation, not inferred from older CI.

### A01 Implement restricted values and Definition decoding

Build the production v1 envelope/value decoder with exact integers and finite decimals, the restricted YAML profile, one Definition per file and source provenance. Reuse an existing parser only after establishing that the profile is enforced explicitly; parser defaults must not choose language meaning.

**Gate:** independent scalar/envelope cases cover duplicate/unknown fields, null, anchors/tags, large numbers, strings that resemble dates/booleans and body whitespace. Invalid or resource-incomplete input cannot be reported as a complete model. This is an internal stage, not a public partial-v1 compiler.

### A02 Acquire model roots Packages and dependency inputs

Application acquires explicit manifests and local dependency roots from one fixed snapshot. Add strict physical root/path validation, complete Definition discovery, Package identities, Unicode/name collision rules and the acyclic direct dependency graph. Core receives explicit acquired values; it does not read Git, files or a network.

**Gate:** library files are discovered exactly once; model root is not a Package; imported/empty/intermediate Packages have their specified behavior. Reject path redirection, overlap, case/Unicode collisions, invalid file extensions and missing dependencies without changing source bytes. Offline inputs are sufficient. Snapshot and source provenance are reproducible.

### A03 Compile Core contracts and resolve references

Implement the nine fixed Core Definitions, name/import resolution, standalone Kind/Property contracts, closed `spec`, cardinalities, reference targets, objects and finite recursion. Preserve exact v1 identities and values throughout.

**Gate:** the library compiles through the real self-described Core and independent positive/negative contract cases. Unknown names, ambiguous references, invalid targets and mandatory object cycles fail; allowed finite cycles complete. No implicit import or special domain Kind is introduced. Definition order does not affect the result.

### A04 Compile binding bodies and explicit links

Implement the selected Markdown/body-link profile, protected contexts, exact target resolution and ordered occurrences. Preserve body content and distinguish canonical content from optional detailed diagnostic coordinates.

**Gate:** library cross-Package connections resolve to exact targets; independent cases exercise code/HTML/ordinary links, repeated body links, malformed targets, escapes and Unicode text. A failed optional coordinate map cannot silently rewrite canonical body content or manufacture coordinates. The intentionally invalid Command/command fixture remains a negative case.

### A05 Complete canonical bytes locks and conformance

Implement `markitect-model-digest/1`, external lock verification, semantic Core equality and the complete conformance result. Exercise all v1 areas, not only the constructs used by the library. Review canonical expected bytes independently rather than generating both expected and actual bytes with the same compiler function.

**Gate:** compare the fixed Core byte artifact and digest, independent equivalent/changed model cases, lock mismatch and dependency cases, and the full positive library. Invalid and incompletely checked models emit no canonical bytes or digest. Resource exhaustion is distinct from invalid language. Design-probe agreement supports its stated cases; it does not replace a broader production conformance corpus.

**Milestone M1:** real v1 model loading and compilation work without a provider. No artifact Apply capability is claimed yet.

### A06 Introduce the bounded execution input and current-model adapter

Add the neutral execution-input contract and a current-model adapter at the Application boundary. Carry original source identities, digests, checks, contexts and ownership without translating or reinterpreting them. Leave the existing runtime route selected by default.

**Gate:** current fixture projections preserve their exact source/semantic bindings and method obligations. Old evidence is never relabeled as canonical-v1 evidence. Architecture and coverage gates pass for the new owner/package. Existing normal, external-executor and native fixture behavior remains supported.

### A07 Move planning work and review to that input

Change current plan construction, Manager scheduling, work/review request generation and candidate harvesting to consume the execution input. Preserve one existing coordinator and Invoker. Do not activate Package-based Apply in this refactoring step.

**Gate:** current work/rework/child/integration fixtures retain expected scopes, independent reviewers, finite budgets and actual-delta validation. Unexpected source format, invalid task binding and contradictory reports still fail. Existing self-hosted checks continue through the current frontend. No new v1 route can publish yet.

### A08 Move verification publication and recovery bindings

Move the remaining candidate validation, full verification, guarded publication and recovery-facing source contracts onto the same execution input and frontend callbacks. Keep old serialized record identities and their validation distinct while the old route is active.

**Gate:** current stale-source, review-freshness, verifier-coverage, external/native lifecycle and partial-write tests still pass. Existing run-version rejections remain closed. No guard is removed to simplify the new contract. At this gate the existing method is reusable without its fixed canonical vocabulary; source-code imports confirm there is no hidden forced reconstruction of old canonical Definitions.

### A09 Derive the v1 organization and context

Build ordinary root/Package Manager topology, effective profiles, shared explicit context and stable own-Definition/See-also rendering. Add targeted detail access with recorded source references. Provide a read-only inspection result through the internal integration harness.

**Gate:** root/catalog/members/lending/reminders have the expected parent identities; imported Core has no executable Manager. Own bodies, `spec`, type contracts, direct incoming/outgoing references and configured additions are supplied for the right reason. Large integer and exact decimal values, Unicode and complete bodies survive context rendering and source references without rounding through old DTOs. Root initially receives child summaries rather than all descendant files, but can obtain a requested file. A source/config change produces new context bindings; no Agent session is started by preparation alone.

### A10 Implement realization registry reservations and target enforcement

Add versioned registry projection, exclusive ownership, actual-census classification, reserved new paths and read-only intake for initial unknown ownership. Host tools construct registry changes; workers cannot rewrite the registry directly or modify model/config/ignore inputs during artifact Apply.

**Gate:** current library artifacts map to the right Managers and model Definitions; one file can realize several Definitions. Unknown files and conflicting claims remain visible. An independently reviewed intake can settle ownership without editing artifact content first. Reject overlapping writes, misleading ignore changes, direct registry edits and attempted model edits. Validate deletion, rename and transfer with preserved provenance.

### A11 Make task and communication history durable

Before the first new execution path is enabled, add versioned run/task/attempt events, immutable context/prompt payload references, actual observed request/response records, causal links and candidate transitions using the existing storage boundary. Add a simple inspection report.

**Gate:** reconstruct root task, delegation, actual prompt/context/instructions, later detail reads, report, review and delta for a fixture invocation. Distinguish prepared/sent/unknown states and reports from harvested changes. Storage failure prevents further unrecorded work. Journal growth does not make product inputs stale, and an unrelated helper folder does not become implicitly excluded.

### A12 Execute one Package with bounded independent review

Wire the v1 frontend, registry, context and history into the shared coordinator for a root and one lending Manager. Use the existing protocol-level test Invoker or executable fake adapter; do not stub the final validation result. Produce reviewed candidates without publishing the target checkout.

**Gate:** scripted work first misses the five-loan boundary, its independent reviewer rejects it, and a corrected candidate is accepted within the configured round limit. The trace proves which exact candidate was reviewed. A passing no-change result still requires an assessment. Round exhaustion and omitted evidence cannot produce completion. Every test remains provider-free.

### A13 Add recursive integration and cross-area coordination

Activate catalog, members, lending and reminders. The root alone receives the user instruction. Child tasks express outcomes; defaults remain sequential. Add parent rework back to the accountable child and a nearest-parent coordination request between members and lending.

**Gate:** capture the actual parent-created task prompts; child write scope stays local. A parent can compose child deltas but cannot author an arbitrary correction to a child-owned file. A conflict or changed shared contract triggers renewed affected work/review. A blocked child blocks aggregate completion. Integration review detects a deliberately incompatible aggregate even when its separate child reviews passed.

### A14 Verify the whole candidate against independent expectations

Run the declared project checks, full repository classification, Definition-assessment census and fresh aggregate verification on the actual composed candidate. Bind frozen LIB expectations as required evidence outside worker write scope.

**Gate:** the drifted six-loan candidate is rejected even when its own test suite passes. The corrected candidate passes LIB-01 through LIB-07. An attempted mutation of frozen expectations invalidates verification. Missing child evidence, unsupported vocabulary-only assessments, omitted required checks and old evidence from a different candidate are rejected. Candidate construction and verification still leave the user checkout unchanged.

### A15 Publish guard read back and recover

Use the existing guarded writer with the new candidate/registry contracts. Add explicit read-back of paths, bytes, modes, deletions and registry after writing. Exercise journal failures and uncertain outcomes with real temporary Git repositories and owned process fixtures.

**Gate:** the corrected tree exactly matches the verified candidate; model/config/ignore bytes remain unchanged. External edits to source, model dependency, configuration, registry or target branch produce stale rather than publication. Partial writes and read-back mismatch report actual outcomes and never Applied. Recovery examines prior work without replaying completed provider invocations. A second run against the now-correct fixture returns an assessed no-change result.

**Milestone M2:** complete provider-free Package Apply works through actual candidate handling, independent test contracts and guarded temporary-repository publication. This proves method mechanics, not real-model reasoning.

### A16 Expose one explicit application entry point

Add the canonical-v1 route to the shared application facade and its CLI/MCP contracts. The project descriptor explicitly selects the source format during transition. Preview, execute and write authority remain distinguishable. Do not introduce a second independent implementation for CLI, MCP or an outer coding client.

**Gate:** both surfaces select the same model, organization, task, target and result. Bad format, ambiguous configuration, missing execution/write authority and cross-project target changes are rejected. Public help explains the temporary format selection and which existing `apply`/`deliver` semantics remain active. A v1 error never falls back to the old loader.

### A17 Run the bounded real-provider library acceptance

Using explicit existing execution authority, a fixed provider/profile and finite budget, run the same correct/drifted fixtures and required LIB expectations through the selected real adapter. Record all source, prompt, configuration, candidate and usage observations. Do not substitute scripted reports for actual Agent work. This is a bounded product acceptance case, not a comparative method study.

**Gate:** from unchanged model intent, implementation, normal tests and guide converge to five; independent review, aggregate verification and protected LIB expectations pass on the actual candidate; guarded publication and read-back agree. Inspect the record from root instruction through local prompt, review and final changes. Repeat as a separate fresh no-change run with current bindings. Any unresolved limitation or missing observation remains reported rather than inferred away.

**Milestone M3:** first reliable Apply demonstrated for the declared library scope, adapter and environment. It is not general production readiness, a speed/cost claim, Windows acceptance or human approval of every future run.

### A18 Convert Markitect self-hosting and select the new default

Split this milestone into three successive PRs: **A18a** prepares and reviews the converted Markitect model/organization/registry as an inactive migration fixture; **A18b** prepares the new renderer and required module/bootstrap/CI/source-smoke consumers while the current descriptor remains selected; **A18c** atomically selects the new descriptor, publishes its generated view and updates active guidance, removing the obsolete accepted model source. Each intermediate main must still pass its active checks.

Preserve the existing repository's accepted intent and file ownership; this is a model migration, not an opportunity to silently change product policy. Compare old and new intent using the conversion map above before the final descriptor switch. Do not maintain two accepted Markitect models or ask authors to keep them synchronized during preparation.

**Gate:** every current file is accounted for and the generated view has one owner. Check/context and the required source gates run through the new frontend. The library M2/M3 fixtures and retained mechanical regressions still pass. Required Linux Quality/Race/source-artifact gates pass on the frozen candidate; after normal integration, read back actual Main source/tree and hosted results. No two accepted Markitect model sources are maintained in the active checkout.

### A19 Remove the old execution frontend and close the transition

Delete the temporary current-model adapter and its authoring loader/compiler paths once no retained consumer depends on them. Preserve shared snapshot, process, record, guard and method logic. Use a consumer inventory to carry forward, explicitly park or retire old-format-specific workflow surfaces; changes to accepted product scope require their own owner decision. Do not port Explore, Brownfield, briefing or graph features merely to keep their old command names. Retain the minimum read-only old-record interpretation needed for provenance, or an explicit route to the producing version. Split removal by consumer family where needed; the final deletion gate requires the whole inventory to be resolved.

**Gate:** production import/search checks find no active path from a current entry point to the retired project Schema; no fake Manager/Statement reconstruction remains. The single public Apply means artifact reconciliation, while lower-level publication remains an internal method boundary. Decide explicitly whether existing `deliver` or other old public verbs are removed or retained with a new documented contract; no compatibility aliases are required. Remove stale examples, generated paths and schema claims. Full source/CLI/MCP/packaged smoke and library gates pass; published historical assets are unchanged. The temporary dual frontend is closed, not left as an indefinite product option.

## Evidence and integration rules

For each slice, record its full source SHA, changed scope, exact tests and fixtures, expected outputs, actual outcomes, known failures and the evidence class. Independent review is required before integration. Source checks, fake-adapter results, real-provider results, post-write read-back, hosted CI and owner acceptance answer different questions; keep their identities separate.

Use concrete negative cases that challenge the contract, not tests that merely mirror the implementation. Existing checks named in [CONTRIBUTING](../../../CONTRIBUTING.md#verify-a-change) remain required for relevant changes. Runtime seams need focused method/recovery/guard tests and then the full required source gates. Compiler work needs independent conformance inputs. Linux is the existing required platform; Windows diagnostics keep their actual outcome and do not become acceptance by inference.

One PR per slice, dependency order, an assigned writer/zone before code changes, frozen heads during required hosted gates, and actual Main verification after each integration remain the repository workflow. Refresh Main, package ownership, backlog gates and current branch state before starting a slice. Do not rerun unchanged broad tests merely to claim a fresh prose-only result; do run the new concrete risks and required integration gates.

## Definition of done for the first reliable Apply

- The complete v1 input is structurally validated, with exact dependency bindings and no fabricated digest for invalid or incomplete input.
- Root-led recursive tasks, local autonomy, ownership and independent local/aggregate review operate on actual candidates.
- Code, normal tests and documentation are reconciled together against unchanged model intent, while model/config/ignore bytes remain protected.
- Independent LIB expectations and full file accounting pass on the aggregate candidate, not only on separate child results.
- The actual published result and registry are read back and match the accepted candidate.
- No-change, stale, review exhaustion, missing evidence, interruption and partial publication have distinct tested outcomes.
- A reader can reconstruct actual prompts, context additions, communication, reviews and changes with visible observation limits.
- Provider-free mechanics and a separately bound real-provider case both have their stated evidence; no broader benefit or acceptance claim is inferred.

The immediate next implementation slice is A00. Compiler work follows from that frozen case; no provider launch or product-code work is dispatched by this plan itself.

Return to the [concept](manager-apply-concept.md), [north star](manager-realization-north-star.md) or [canonical-model overview](README.md).

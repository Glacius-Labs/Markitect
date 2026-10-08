# Managed project workflow: local validation

Validation: 2026-10-08 to 2026-10-09 (Europe/Berlin). Branch: `codex/model-driven-delivery`. Base: `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`.

This local source candidate implements the user-authorized [project-world design](../design/project-world/README.md). The [workflow](../project-workflow.md), executable [Shop fixture](../../examples/project-world/README.md), and [live exercise](../../experiments/project-world-live/README.md) are the entrypoints. The earlier `.yaml.example` design specimens are historical proposals. No remote push, release, deployment or adopting-project human acceptance is claimed.

## Implemented behavior

The unchanged structural Core compiles a separate project-model capability with recursive Managers, Statements for concepts/rules/use cases/architecture/workflows, Artifact expectations and concrete file assignments, Checks and Decisions. Hidden `.markitect/` sources own the canonical model, runtime configuration, proposals, generated documentation and operational evidence. A readable view does not become a competing source of intent.

The public CLI supplies schema, initialization, check/index/context/impact/document, exact-digest model edits, reviewed setup and local doctor checks, fixed-source discovery, grounded agent-assisted distillation, explicit resolution and partial adoption, implementation planning, separate manager invocations, integration, independent checks and guarded Apply. `plan --since` derives affected responsibilities without requiring the user to enumerate implementation files. Each manager receives its own context and resolved write scope; a realization/read relation does not grant another owner's write authority. Known invalid responses can receive bounded compiler feedback. Explicit `project repair --run RUN_ID --write` returns a failed required declared check to the same Manager tree, preserves the original deadline and cumulative invocation/check/cost ledger, archives prior task evidence and requires a changed candidate plus fresh verification. Uncertain invocations are never silently replayed. Resume reconciles a crashed Host's durable running state and blocks uncertain in-flight work.

Provider selection, effort, executable/adapter fingerprints, environment allowlists and finite limits are explicit. Setup changes only the project's runtime through the normal edit boundary and does not inspect credentials or modify global settings. Authentication remains unverified until an actual provider operation succeeds. Caller cost weights are estimates, not published prices or a hard invoice cap.

## Actual Shop implementation

The real provider-backed walkthrough used native source `5ab6f75506b1fe4ba9039bc56f7e2fe87f8a9917`, Codex CLI 0.130.0 and explicitly selected `gpt-5.5` with high effort. The implementation team used Luna High; those development agents are separate from the Shop's native provider processes. The available account rejected `gpt-6-luna`, so the experiment selected the supported model explicitly; the product performs no automatic fallback. No global configuration was changed.

Preparation copied the fixture into a fresh feature-branch Git repository, froze its adapters, accepted the reviewed runtime and changed four model Statements: order state, cancellation rule, cancellation use case and the reservation invariant including rollback to the original confirmed or packing state. `plan --since` selected six Managers and one declared check. Work ran top-down and integration bottom-up through separate processes.

| Attempt | Native source | Result |
|---|---|---|
| `ac36afa6356e1379282834cf88a3f69e` | `5ab6f75` | Nine real invocations integrated a candidate; independent verification failed three existing assertions because the candidate changed reservation error text from `released N` to `found N`. Nothing was applied. |
| `3dc1b26512d5041d327a7c460a034fca` | `5ab6f75` | A separate explicitly created plan included that failure as concrete feedback. Ten real invocations, including one bounded response-validation retry, produced a candidate that passed all eight declared tests and was applied. |

The successful candidate changed `src/shop/commerce/cancellation.py`, `docs/cancellation.md`, and `tests/test_cancellation.py`. The first failed attempt remains in its own durable ledger. These were two separate plans with their own ceilings; they are not evidence of the later same-run `repair` API. Across these two attempts, 19 invocations have an estimated total of 8,742,980 micro-units using caller weights of 20,000,000 input and 100,000,000 output micro-units per million tokens. These figures are not provider prices or a billing record.

The Apply exercise changed a disposable documentation file after preflight and confirmed stale rejection, restored its exact bytes, then applied the verified candidate with the same exact bindings. The actual checkout passed its eight tests again. An independent checker, kept outside the adopting checkout and never supplied to Managers, passed eleven cases: six confirmed/packing combinations with zero/one/two active reservations, two pre-released rollback cases, two injected SQLite release failures, and one shipped-order rejection. Repeat idempotency is asserted within each successful one-reservation case rather than counted as an additional case. The checker at `f30078856876f62d1ae834965535fb033748c727` builds its own database rather than importing fixture tests. It rejected the exact pre-Apply source at `bf5728f69ac92caeb79ffe35f833e62664b2e925` on packing cancellation. Independent review found the four changed Statements aligned with the resulting implementation and documentation; the additional checker closes the packing rollback coverage gap in the eight declared tests.

## Actual Brownfield proposal and partial adoption

Native source `f30078856876f62d1ae834965535fb033748c727` exercised `discover -> distill --generate -> resolve -> adopt` against target/source commit `ca3eb108e45e158afea5c62a6880fa221ed583d8`. Four files were explicitly selected: order source, cancellation source, reservation source and a deliberately contradictory legacy guide. No runtime record was selected.

One real generation call produced seven grounded claims, five terms, two proposed scopes, one contradiction and one blocking question. Static observations retained `static-source` provenance; the guide retained `documentation` provenance. The report preserved the disagreement between confirmed-only code and confirmed-or-packing documentation instead of silently deciding which was true. Exact excerpts and inclusive source line bounds passed validation; original LF/CRLF blobs remained unchanged. Its estimated accepted cost was 751,140 micro-units under the same caller weights.

For this disposable test, explicit caller choices preserved the accepted confirmed-only target contract, clarified the question, adopted Inventory and deferred Orders. The decision record states `authenticated:false`; it is a test decision under the authorized task, not authenticated business-owner acceptance. A Manager actor lacking the selected namespace's write authority was rejected before model writes. The successful caller-authorized adoption changed only `.markitect/project.yaml` and added `.markitect/model/commerce/sales/inventory/reservations/imported-release-active-rowcount.yaml`. Deferred Orders files are absent. Every code, test and documentation file stayed byte-for-byte equal to its pre-adoption hash, and `project check` succeeded afterward.

The validated report digest is `a67dbc222ea79380dad2fa09e4e8f0ba33c1c50ad452c6dae0eb58568de3826d`; the successful resolution digest is `9a9a3829071e3c36b1b8567e08151c893bf558ea4e74a6647074915b156a6967` and the guarded plan digest is `49bf0c675ee4e3d4a9bde385670c242bc58071246e280b92714b9b82f26600f3`. This demonstrates bounded migration into an existing target model, not automatic understanding of an arbitrary repository.

The final CLI retains an immutable invocation-specific receipt even when a proposal is rejected or concurrent generations compete for the same report path. Receipts are written before the shared report. A barrier-synchronized regression forces concurrent calls and verifies that both receipts survive; bounded retries apply only to transient guarded-writer lock contention, never to destination collisions or stale state.

Earlier live attempts exposed concrete defects: fixed-versus-working preflight compared revision-bearing project metadata rather than snapshot bytes; root-scope response schemas prohibited an empty parent; source excerpt line selection lacked guidance; and the generator duplicated nested model proposals as outer executor file edits. These were corrected while preserving grounding, ownership and report-only validation. Those rejected attempts are not counted as passing evidence.

## Source gates

The earlier full suite at `950332a9bc621153e70372f5f97a28b3365644c4` passed 52 packages. During this continuation, fresh full suites at `d789a2e1d978e0aee8074318f0bd15914adc8d50` and `5ab6f75506b1fe4ba9039bc56f7e2fe87f8a9917` passed 53 packages. Thirty-one standalone gates at `5ab6f75` also passed: vet, module verification, schema freshness, fixed root check/context/impact, root format, artifact/module accounting, contribution examples, adapter suites and diff whitespace. The adapter protocol counts were 26 Codex and seven Claude tests. These deterministic tests are distinct from the real provider evidence above.

The final implementation source is `18e2a1047db7324e0902829f9c56dfb098e42992`, checked with Go 1.27.1 on Windows. The subsequent final report is prose only; its exact fixed-source checks are recorded separately.

| Gate | Tested source | Result |
|---|---|---|
| `go test ./... -count=1 -timeout=30m` | `18e2a10` | PASS, 53 packages with tests, including historical compatibility and architecture imports |
| Thirty-one standalone gates and native CLI build | `18e2a10` | PASS: vet, module verification, native build/schema freshness, fixed root check/context/impact, format, artifact/module accounting, executable examples, Codex/Claude protocol tests and whitespace |
| Same-run required-check repair through real subprocess transport | `18e2a10` | PASS: failed check -> bounded repair -> fresh Verify -> guarded Apply; same identity/deadline/ledger, prior receipt/verification/lineage retained, old failed candidate rejected |
| Repair negative cases | `18e2a10` | PASS: exhausted cost/starts/rounds, stale/uncertain/verifier failures, unrelated diagnostic redaction and a new candidate ID with unchanged snapshot bytes |
| Distillation receipt and concurrency regressions | `18e2a10` | PASS: rejected proposal keeps known usage/receipt, occupied report blocks invocation, concurrent calls preserve distinct receipts, private provider details remain absent from CLI errors |
| Native new-project initialization and current Shop/Brownfield checks | `18e2a10` | PASS: preview is read-only, write creates Markitect-owned files only, both completed example checkouts compile |

The repair/concurrency tests use deterministic proposers through actual external processes. They establish those lifecycle mechanics separately from the real provider-backed Shop/Brownfield results above. The Shop's earlier two-plan feedback sequence is not retrospectively relabeled as a same-run repair experiment. An independent review found no remaining concrete issue in the final repair and receipt changes. Nine relevant Markdown files have 228 checked local link targets with no missing target; final prose checks retain the actual full-suite source above.

Local logs and disposable checkouts remain outside repository inputs under `%TEMP%/markitect-project-world-live-validation/`. `usable-shop/` contains both attempt ledgers, stale-Apply rejection, eight actual tests, independent positive/negative checks and generated documentation. `report-only-brownfield/` contains fixed discovery, the report/receipt, explicit choices, resolution, partial adoption and original implementation hashes. Provider logs remain private and are not published as raw transcripts.

## Remaining boundaries

`controlled-local` constrains selected inputs, the proposal protocol, ownership and guarded Apply. It cannot prevent another process with the user's OS authority from modifying files or external systems. Requested `isolated` mode fails closed; a verified container/VM launcher, isolated credentials/egress and an independent Apply broker remain future implementation.

No live Claude inference was run. The observed installed Claude CLI 2.1.233 is below the restricted adapter's minimum 2.1.248; unsupported versions are rejected. A supported conversation host remains the user-facing surface; the deterministic CLI itself contains no language model or autonomous conversation service.

Compilation, linked artifacts and finite tests do not prove every comment, document and implementation fact is consistent, complete or correct. The design's larger acceptance catalogue remains a requirements list, not a claim that every scenario ran. Hosted Windows/Linux CI, immutable packaging, release gates, production behavior, economic benefit and human acceptance require separate evidence.

## 2026-10-09 Luna High validation and Scientist handoff

This additional local validation uses fixed source `510efe627f73f544492ff301ee9be329fe6fb01b` on `codex/model-driven-delivery`. That revision contains the implementation previously checked at `18e2a10` plus its final validation report. This record preserves the earlier receipts and their actual `gpt-5.5 high` provider selection. The worktree was clean before testing. No installed release, other chat's worktree, Overseer or Scientist task was changed.

Three independent review agents were explicitly configured as `gpt-6-luna` with `high` reasoning. They inspected runtime/write boundaries, setup/adoption/adapters, and the executable example/acceptance harness respectively. Their model selection is distinct from the native provider selected by the adopting project's runtime.

| Check | Result at `510efe6` |
|---|---|
| Fresh full Windows suite, `go test ./... -count=1 -timeout=30m` | PASS: 53 packages with tests, 282.5 seconds, Go 1.27.1 |
| Contribution and project CLI gates | PASS: 35 correctly invoked source gates plus five project gates against a standalone disposable Git root |
| Codex and Claude protocol suites | PASS: 26 and seven tests respectively; no live Claude inference |
| Native Windows build and Linux amd64 cross-build | PASS; the Linux binary was not executed |
| Native Init, previews and failure guards | PASS: 12 CLI probes, including read-only Init/setup, stale setup digest, and rejected-provider Resume/Repair/Verify/Apply |
| Independent behavior checker | PASS: all 11 cases on the retained earlier provider-created candidate; fixed pre-Apply source fails packing cancellation as expected. No new inference occurred in these checks |
| Independent Luna High reviews | No remaining actionable finding in the inspected scope; focused Go packages, CLI help, seven baseline Shop tests and the negative control checked separately |

The first source-gate helper incorrectly invoked five project example commands directly against `examples/project-world`, which is not a Git worktree root. All five were rejected with that precise diagnostic. Repeating those commands against the separately initialized Shop, as documented in its README, passed. The native-probe helper also needed a local correction because `project document` emits Markdown rather than JSON. These were test-orchestration corrections; the original failed invocation records are retained. The full suite's outcome remains its own receipt.

### Native Luna High provider attempt

One fresh disposable Shop was prepared through the public CLI: local doctor/setup, four accepted model edits, a commit, then `plan --since`. Planning selected six Managers and one check. Fixed target commit: `d27c40ae2f4c5b580387163050e22fe759ec5570`; plan/run ID: `ef8b2ac06ef07ad837c497850e24a71b`.

Native Codex CLI `0.130.0` received model `gpt-6-luna` and `model_reasoning_effort: high`. The active account rejected that model at the root Manager's first work call. The public response states: "Codex rejected the configured model for the active account." There was no model substitution. This is a provider/account compatibility failure; no successful Luna implementation experiment resulted. It says nothing about availability in another host or account. The review agents above ran with their separately configured Luna High profile.

The run stopped as `failed`; its root task durably records one work attempt and an uncertain outcome, and the candidate remains empty and unintegrated. There is no accepted invocation/usage entry for this rejection. The task attempt is retained, while token usage and any provider charge remain unknown; missing usage is not zero cost. Resume, Repair, Verify and Apply each rejected the failed run without another provider process or any selected/operational file change. No second native-provider attempt or fallback was made.

Evidence is under `%TEMP%/markitect-luna-high-validation-20261009/`: `full-tests-510efe6.json` and its log, `gates-510efe6.json`, `standalone-project-gates.json`, `native-cli-probes.json`, `linux-cross-build.json`, `independent-acceptance.json`, and `shop/state.json` plus public status/setup/run outputs. Frozen native Windows binary: `markitect-510efe6.exe`, SHA-256 `867a50c63fa2b584808778698d8b8e1a2d90ac255ba4968c0af1710b1a78e38f`. Adapter/provider/runtime digests remain in the disposable runtime and plan. Private provider logs remain private.

### Independent trial recipe

The user will instruct the Overseer. This handoff prepares a candidate and a bounded scenario; no Scientist trial was started and no independent acceptance is claimed.

1. Use a fresh checkout at the supplied final candidate commit. Record full SHA, clean status, tool digest, provider executable/version/digest and actual Luna High model mapping. Keep model, tools, requirements, permissions and finite budgets identical if comparing approaches. Resolve the observed native model rejection in the actual trial environment before claiming a Luna run; stop on an unsupported profile rather than choosing a fallback.
2. Run the contribution gates in [CONTRIBUTING](../../CONTRIBUTING.md), then build a separate binary. Keep binary and adapter bytes fixed throughout each lifecycle. Do not reuse the failed local run or earlier accepted fixture as a fresh actor/task result.
3. Use [the Shop harness](../../experiments/project-world-live/README.md) with an explicit model and fresh output directory. Preparation edits the model and plans impact without inference. Check the expected six Managers and one check. `--stage run` starts inference; integration still requires fresh `--stage verify` and guarded `--stage apply`. A known required-check failure permits explicitly bounded same-run `--stage repair`, followed by fresh verification. Preserve failures, retries, interventions and usage/budget evidence.
4. Freeze the candidate before evaluator checks. The independent [acceptance checker](../../experiments/project-world-live/acceptance.py) must fail on the original fixture and pass all 11 cases on a correct changed implementation. Keep its code/results out of actor-supplied materials. Being outside the Shop checkout does not establish inaccessibility: `controlled-local` is not OS isolation. If a hidden holdout is required, test the actor's actual access boundary independently and invoke the evaluator after candidate freeze.
5. For Brownfield, use a separate fresh fixture through preparation, generation and explicit resolution. Review the report and question choices. For a user-led adoption review, invoke CLI `adopt` preview and guarded write separately as shown in [Project workflow](../project-workflow.md); the harness's `--stage adopt` immediately previews and writes within an already-authorized test. Confirm deferred scopes contribute no files and code/test/documentation bytes remain unchanged by model adoption.
6. Report deterministic mechanics, provider-backed behavior, independent evaluator findings and human acceptance separately. Count setup, model upkeep, coordination corrections and failed attempts. Hosted CI, Linux execution, release packaging, hard OS isolation, arbitrary-repository understanding, long-running autonomy and productivity benefit remain outside this local result.

The Go race detector was not run: this Windows host reports `CGO_ENABLED=0` and has no discovered GCC/Clang compiler. No compiler, provider, account or global configuration was installed or changed to hide that gap. The Linux cross-build does not substitute for Linux tests. Earlier provider-backed Shop/Brownfield evidence remains bound to its actual source above.


## 2026-10-09 Current App CLI and successful native Luna High

The earlier native rejection was specific to the selected npm Codex CLI 0.130.0 and the active ChatGPT account. It was a model/provider compatibility rejection, not an approval rejection. On the user's explicit instruction to start and test Luna High, the current App-supplied `CODEX_CLI_PATH` resolved to Codex CLI `0.162.0-alpha.2`. That exact CLI successfully executed `gpt-6-luna` with `model_reasoning_effort: high`. No fallback or account/global configuration change occurred.

The native App executable is 333,357,008 bytes. Source `14dbd9827d7f4c2470dd8bb46ca71099488ab0f0` supports this through a finite 512 MiB combined runtime-file bound and streaming fingerprints; executable identity, modes and before/after checks remain enforced. Large-asset and combined-limit regressions passed. The harness now accepts an explicit `--provider-executable` during preparation and freezes that path/version/digest for later stages. The restricted production environment remains `PATH`, `TEMP`, `TMP`, and `SystemRoot` with case-insensitive name matching. Two smoke helpers initially omitted `SYSTEMROOT` because their Python filter used case-sensitive names; both failures are retained, and the corrected helper succeeded without broadening the production allowlist. Two successful native smokes produced provider-reported usage; two failed smoke attempts have no established usage/cost.

### Preserved first attempts and corrective changes

| Native source | Attempt | Observed result |
|---|---|---|
| `14dbd98` | Shop `b531da00a838c11992c96752e0d7740a` | Nine real accepted calls; blocked before Verify/Apply by unresolved Manager reports, including an invented collaboration-dispatch failure and a coverage obligation already addressed by another branch. Estimated cost: 4,946,820 micro-units. |
| `14dbd98` | Brownfield at `ac5572b6b9548d17d9f54ee9ab572a8d708422a2` | One real call; rejected because term `active-reservations` did not occur literally in its cited excerpt. The receipt retained known usage and estimated cost of 659,200 micro-units. No report was accepted or adopted. |

Neither failure was bypassed. The Shop's blocked candidate passed an offline eleven-case diagnostic evaluation in a separate materialized directory; that diagnostic is not Host Verify or Apply. Source `a4a97f87074c5ad72de8e881513b251016f80f38` clarified the stateless proposal boundary: the Host schedules delegations, Managers assess current supplied child evidence, implementation integration precedes Host verification, and absent private descendant context is expected. Existing ownership, unresolved-obligation, integration and verification guards remain unchanged. Distillation instructions now require literal, case-sensitive terms in every cited excerpt. Rejected report validation also records fixed safe receipt categories `report-validation` / `invalid-distillation`, without exposing raw provider content. Independent Luna High review found no actionable issue in these bounded changes.

The corrected tests below used fresh fixtures, plans and frozen tools. They did not resume or repair the blocked first attempt. Frozen native tool: `markitect-luna-guidance.exe`, SHA-256 `fe52e58ffa90fe3818d59d66f3a6bd8bf4447e00af5bc809edbab60481cdeaa3`. Native provider SHA-256: `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`; frozen Codex adapter SHA-256: `6b6c6dab5f2026a090dabd8b1593d1250dd629b71d86ac901ab190c634d330eb`.

### Successful Shop lifecycle

Source `a4a97f8`, target commit `deb59378612c5929c12193d41ed48d5925e68c71`, plan/run `8ca492c68f8f3aa56f26b3cae6a57ade`: four accepted model edits led `plan --since` to select six Managers and one check. Six work calls and three integration calls produced integrated candidate `9375cb2d0b7f624a6ed4722b33638635`, snapshot `sha256:8b63324809801651437109da078830005f6fddb4e918139314b50d6ac40cde50`. All calls used native Luna High; none required a protocol retry or check repair. Their provider-reported total was 158,836 input tokens and 7,378 output tokens, with estimated cost 3,914,520 micro-units under the experiment's caller weights.

Fresh Host Verify passed all eight declared tests. The Apply harness rejected a deliberately stale documentation binding, restored exact bytes, then applied the verified candidate. The applied files are `src/shop/commerce/cancellation.py`, `docs/cancellation.md`, and `tests/test_cancellation.py`. The actual checkout passed its eight tests again. The independent checker passed all eleven cases, including packing-state rollback and idempotency; the pre-implementation checkout failed packing cancellation as the negative control. This demonstrates the complete model edit, impact, delegation, integration, Verify and guarded Apply path on the finite Shop scenario.

### Successful Brownfield lifecycle

Source `a4a97f8`, fixed target/source commit `921a7942c6b9c9463d4888125a118e83fb8a0870`: one native Luna High generation produced three grounded claims, three terms, two scopes, one contradiction and one blocking question. Static source observations and documented intent retained separate provenance; no runtime record was selected. The confirmed-only implementation versus packing-permitting legacy guide remained an owner question. Estimated cost: 692,960 micro-units.

Explicit caller choices in this authorized disposable test adopted Inventory and deferred Orders, preserving the accepted confirmed-only target contract. The decision remains `authenticated:false`. Adoption changed only `.markitect/project.yaml` and added `.markitect/model/commerce/sales/inventory/reservations/imported-active-release.yaml`; all thirteen tracked code/test/documentation files retained their original hashes, deferred proposal files are absent, and `project check` passed. Distillation digest: `686fbba1925118a53cfad4793a188be46776da79df4ab5288baf9b69d06b297d`; resolution digest: `354b05b089a141b1597b8dc64ab9e2f876766314ae74585a6ea02b149d075ccd`; adoption-plan digest: `5850d11f2d0cb49c9d15da172f582dc4a2cbf44751e360c7453bfcfc87905980`.

Across the two Shop and two Brownfield attempts in this continuation, twenty actual task invocations have known estimated cost totaling 10,213,500 micro-units. Four additional native smoke attempts are accounted for separately above; earlier CLI rejection and development/review agents remain separate records. Caller weights are estimates, not provider prices or a billing cap. Failed attempts and coordination corrections remain part of the evidence.

Public results and disposable fixtures are retained under `%TEMP%/markitect-luna-high-retry-20261009/`: `current-shop/` and `current-brownfield/` preserve the first attempts; `guided-shop/` and `guided-brownfield/` contain the successful lifecycles and acceptance evidence. Raw provider logs remain private. The independent trial recipe above now has an observed working App CLI route; it still requires checking the actual trial host's executable/account/model compatibility and fresh actors/fixtures. No Scientist trial, remote push, release or human acceptance is claimed.


### Final source gates and review boundaries

Fresh Windows source validation at `a4a97f87074c5ad72de8e881513b251016f80f38` passed all 53 packages with tests (`go test ./... -count=1 -timeout=30m`, 256.84 seconds) and all 31 standalone contribution gates. The focused Manager runtime suite passed separately (138.763 seconds); adoption and CLI suites passed after the rejection-category changes. Codex/Claude protocol suites passed 26/seven tests. Native Windows and Linux amd64 builds passed; the Linux binary was not executed, and the race-detector limitation above remains. Five additional public project check/index/context/document gates passed against the applied standalone Shop. Evidence: `full-tests-a4a97f8.json`/log, `gates-a4a97f8/summary.json`, `guided-shop/post-apply-project-gates.json`, and both fixtures' public status and acceptance records.

Independent Luna High review found the Shop implementation, documentation, test and phase-specific Manager reports consistent with the accepted change. It also confirmed Brownfield provenance and partial adoption. One bounded traceability note remains in the deferred Orders proposal: its rollback description is supported by the selected fixed source, but the report's explicit claim excerpts do not separately cover that detail. Orders was not adopted; the Inventory adoption is unaffected. This illustrates why selected-source grounding and structural validation do not establish exhaustive prose-to-claim coverage. Any future adoption of that deferred proposal still needs review.

These results make the candidate ready for an independent Scientist exercise of this scenario. They do not establish arbitrary-repository semantic completeness, OS isolation, repeated autonomous success, economic benefit or authenticated owner acceptance. The source candidate remains local and separate from installed releases.

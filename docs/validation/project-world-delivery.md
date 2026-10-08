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

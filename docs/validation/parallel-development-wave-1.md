# First parallel development wave

Status: all ten workstream assignments closed as integrated source/tests/reports or explicitly deferred implementation. Final combined gates are bound to the coordinator consolidation PR and its subsequent main CI. Published v0.12.0 remains unchanged. This record measures subsystem independence, not productivity or adopter acceptance.

## Fixed baseline and gates

All workstreams start from `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`, the PR 60 preparation merge. [Main CI 37144006754](https://github.com/Glacius-Labs/Markitect/actions/runs/37144006754) passed Windows, Linux and source-artifact gates at that exact commit; verified again before dispatch. The [workstream specifications](../workstreams/README.md), [shared contracts](../development/shared-contracts.md) and [constitution](../engineering-constitution.md) are implementer authority. Earlier proposals remain design inputs.

## Assignments and live integration record

The explicit assignments below narrow the broader future-work permissions in the workstream specifications for this wave; a future specification is not authority to exceed the assigned paths. GitHub/Azure consume offline captures only: no live fetch, credential lookup or credential use.

Each branch has a managed isolated worktree; shared renderer files, CLI/schema/semantic contracts, dependencies, public product documentation and integration remain coordinator-owned. Implementers commit local candidates; the coordinator serializes shared-ref publication and merges. Every source candidate requires exact-head Windows/Linux CI and independent full-diff review.

| Workstream | Branch | Bounded first-wave deliverable | Status / dependency | PR / CI / integration |
|---|---|---|---|---|
| Init | `codex/wave-init` | Explicit scope and isolated preparation design | Proposal integrated; durable implementation deferred | [61](https://github.com/Glacius-Labs/Markitect/pull/61) |
| Copy Me | `codex/wave-copy` | Evidence/candidate review design | Proposal integrated; durable implementation deferred | [62](https://github.com/Glacius-Labs/Markitect/pull/62) |
| Risk | `codex/wave-risk` | Risk classifications and command-plan negative controls | Tests/report integrated | [64](https://github.com/Glacius-Labs/Markitect/pull/64) |
| .NET | `codex/wave-dotnet` | Literal ProjectReference namespace hardening | Source/tests/report integrated | [67](https://github.com/Glacius-Labs/Markitect/pull/67) |
| Docs | `codex/wave-docs` | Projection ownership and human-content negative controls | Tests/report integrated | [65](https://github.com/Glacius-Labs/Markitect/pull/65) |
| Codex | `codex/wave-codex` | Provider projection independence negative controls | Tests/report integrated | [66](https://github.com/Glacius-Labs/Markitect/pull/66) |
| Claude | `codex/wave-claude` | Provider projection independence negative controls | Tests/report integrated | [63](https://github.com/Glacius-Labs/Markitect/pull/63) |
| GitHub | `codex/wave-github` | Offline fullName/defaultBranch conformance consumer | Source/tests/report integrated | [70](https://github.com/Glacius-Labs/Markitect/pull/70) |
| Azure DevOps | `codex/wave-azure` | Offline defaultBranch conformance consumer | Source/tests/report integrated | [69](https://github.com/Glacius-Labs/Markitect/pull/69) |
| Konfyra | `codex/wave-konfyra` | Sanitized read-only consumer inventory | Report integrated; no consumer changes | [68](https://github.com/Glacius-Labs/Markitect/pull/68) |

## Frozen boundaries and requests

- No new Core primitive, provider-specific semantic field, query capability, inheritance or implicit activation.
- External adapters consume canonical normalized meaning and explicit captured inputs. Captured metadata is observed evidence, never proof of current remote state. No credentials, live provider writes or Apply capability in this wave.
- Native Docs/Codex/Claude have independent goals but share renderer implementation. Their initial ownership is new, separate tests/reports; concrete production defects return to the coordinator.
- Init prepares selected evidence; Copy Me interprets and proposes; owner review and normal canonical adoption remain separate. Neither may independently publish a new shared record/CLI contract.
- Konfyra remains a consumer. Private source material, credentials and customer data must not enter this repository. Inventory does not authorize adoption or replace existing guidance.
- Historical pilot evidence, package versions and published release artifacts remain immutable.

## Coordinator decisions during execution

- **Native assignment scope:** future workstream specifications allow broader provider work; this wave explicitly assigns only disjoint new tests/reports. Production changes require a concrete request and coordinator decision.
- **Remote-provider scope:** GitHub compares canonical repository identity/defaultBranch with a supplied sanitized repository capture; Azure compares canonical defaultBranch with supplied captures mapped to exact organization/project/repository identities. Both are offline evidence consumers, with operation-free Plans and no Apply. Neither proves current provider state or authenticates the capture. Desired values come from normalized resources, not duplicated expected-value configuration.
- **Init/Copy Me:** proposal-only first-stage deliverables are complete. Existing source.Load acquires whole-repository bytes, so filtering it afterwards cannot prove selection privacy. Durable implementation is deferred: review selective exact-blob acquisition, external workspace storage/retention, and the minimal common handoff together before establishing a CLI or persistent record. Existing Init and discovery Workflow remain unchanged. This is a bounded design decision, not a new Core requirement.
- **Execution environment:** default-sandbox shell calls stalled even on Get-Location. Narrow authorized repository operations with login disabled and escalation returned promptly. This is a tooling fan-out limitation, not evidence of adapter semantic coupling.

## Completion evidence

Independent full-diff reviewers and the coordinator reviewed every candidate. Review findings led to new .NET, GitHub and Azure commits; superseded candidates were not integrated. Each row identifies the final reviewed head and its own Windows/Linux quality run, including Go tests/vet/build, generated schemas, executable examples, packaged bootstrap/reconciliation and Windows onboarding replay. Merge commits record integration, not release publication. Final combined-source gates are recorded by the coordinator consolidation PR and its subsequent main CI, without embedding a circular self-commit claim here.

| PR | Reviewed head | Windows/Linux CI | Integration commit |
|---|---|---|---|
| 61 | `d53a05576e3c5a4c53c49c6391adb2a6f016fecf` | [37145792192](https://github.com/Glacius-Labs/Markitect/actions/runs/37145792192) | `236f35f7c525403a9580dee9b6899a0fd1ffa2e1` |
| 62 | `befda848a3eb9617856699c0389f56a48a750985` | [37145796420](https://github.com/Glacius-Labs/Markitect/actions/runs/37145796420) | `c572b6f0899a1afac1c14907677f351d786f9a8e` |
| 63 | `148f40698cf3aa86517815087931340002e07d12` | [37146009029](https://github.com/Glacius-Labs/Markitect/actions/runs/37146009029) | `355df11e8ccfb61e0ff329badf7e72863a9bbf09` |
| 64 | `947b8a5ffd5de2c39151d7956cf21d0fb141f291` | [37146014429](https://github.com/Glacius-Labs/Markitect/actions/runs/37146014429) | `b560856770203f1e323170ad4dbbfaec7c5cfa5c` |
| 65 | `d7daa4b759d94f409ba298e067255f0aa7d3e788` | [37146076822](https://github.com/Glacius-Labs/Markitect/actions/runs/37146076822) | `181675aa85943bbba07e13588786f48673459408` |
| 66 | `de36c6cb226a2027ced878efc24c23c365189c77` | [37146082093](https://github.com/Glacius-Labs/Markitect/actions/runs/37146082093) | `046081ae1dbeef26ca10cbe309932498742a9b8d` |
| 67 | `971b6e7887c2a8f2772bd70d855516772d653955` | [37146910945](https://github.com/Glacius-Labs/Markitect/actions/runs/37146910945) | `2f1954cfd9a9f25f3eaff240a949bb5b40bc9668` |
| 68 | `da3ab5992b12b3a3b436eddca9038a8627d7afee` | [37146916159](https://github.com/Glacius-Labs/Markitect/actions/runs/37146916159) | `d57c7944282727640d83aaf41202d2110135153a` |
| 69 | `dca78e1b7fb8bd3b9dae632e605c52098f809304` | [37147761997](https://github.com/Glacius-Labs/Markitect/actions/runs/37147761997) | `dff229a3a5f7b75c15a9aa46abbfe1c76546114f` |
| 70 | `edf4be2943a85a67df260ede85109be1c826adf7` | [37147773338](https://github.com/Glacius-Labs/Markitect/actions/runs/37147773338) | `c9e0dc9e5ea2f7255e554071bf11385d520cfe95` |

### Concrete defects and shared requests

| Finding | Classification and resolution | Limit |
|---|---|---|
| .NET matched XML Local names without proving their namespace, including a foreign `Include` attribute. | Real adapter evidence defect. Empty/standard MSBuild element namespaces and unqualified recognized attributes are required; unsupported lookalikes return incomplete without observed dependencies. | Still only literal unconditional captured XML, not evaluated MSBuild, runtime dependencies or C# semantics. |
| GitHub's partial DTO plus strict decoder rejected a full valid command request. | Adapter interoperability defect, found during review before integration. Full current DTO structure and a process test using actual app types replace the partial protocol test. No shared SPI repair was necessary. | Prototype request bound is 10 MiB; this is not a Core scale guarantee. |
| Supplied JSON could contain contradictory duplicate/case-colliding recognized keys; Azure also accepted repeated API-version query values. | Adapter evidence ambiguity. Both commands reject duplicate/case-colliding keys; Azure rejects repeated/malformed query parameters. | Captures remain unauthenticated supplied records. No live-state claim follows. |
| Init scope filtering after whole-repository acquisition could be mistaken for selection privacy. | Shared acquisition/storage/handoff design request. Durable implementation deferred; exact selective blob acquisition and storage review must precede it. | Current Init/discovery behavior is unchanged. Proposals are not accepted persistent formats. |
| Native adapters share path inventory, renderer dispatch and reconciliation. | Known ownership hotspot, not a reproduced defect. This wave assigned disjoint negative-control tests and centralized production changes. | Does not prove unrestricted native production fan-out. |
| An isolated agent's relative patch landed in the primary checkout. | Coordination tooling defect. Own sanitized draft was preserved, then committed from the intended absolute worktree path. Use absolute paths and verify destination before staging. | No consumer file or shared contract changed. |

### Cross-workstream validation

The coordinator's [executable runner control](../../examples/parallel_wave_adapters_test.go) builds both new commands and uses the actual `app.Parse` / command orchestration path. One fixed, structurally valid generic Domain/model supplies two disjoint captured inputs and target identities. It asserts complete Observe/Verify, byte-identical repeated Plans, no operations, unavailable Apply, equal model digests and unchanged canonical snapshot bytes.

The focused control passed locally on Windows after all ten workstream PRs were integrated. It is included in normal `go test ./...` on both CI platforms; the final consolidation's checks and [main CI](https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml?query=branch%3Amain) are the authority for the exact combined commit, rather than a promise made by this source file.

Changing only the GitHub capture causes its Verify to fail while Azure's serialized observation stays identical. The old Azure saved plan still becomes stale because the source digest conservatively covers the whole snapshot. Output independence must not weaken saved-plan input binding. This is bounded synthetic protocol evidence, not an adopter utility, live-provider or permission test.

The four test-only Risk/Docs/Codex/Claude streams preserve concrete trust/ownership/independence boundaries without new runtime behavior. [Risk triage](parallel-wave-risk-triage.md) distinguishes protected invariants, test hardening, proposals and evidence gaps. [Docs](parallel-wave-docs.md), [Codex](parallel-wave-codex.md) and [Claude](parallel-wave-claude.md) reports state exactly what their local projection controls prove.

The [sanitized consumer inventory](parallel-wave-konfyra.md) preserves one fixed read-only inspection and selected metadata. No private source content, credentials, customer data, adopter mutation or adoption claim entered Markitect. The local ignored identity/hash manifest remains separate from the public aggregate report. The adopter already has explicit canonical owners and generated pointers; reduced maintenance remains unproved. Historical MyMeetings/comparison evidence and its negative findings remain untouched.

## Coordinator answers

| # | Question | Evidence-bound answer |
|---|---|---|
| 1 | What launched concurrently? | Init, Copy Me, Risk, .NET, Docs, Codex, Claude, GitHub, Azure DevOps and Konfyra, each on its own managed worktree/branch from the fixed baseline. |
| 2 | What completed independently? | All ten produced bounded deliverables. Eight source/test/report slices completed without shared semantic writes; Init and Copy Me completed proposals, with durable implementation explicitly deferred. |
| 3 | What was blocked by shared contracts? | Init/Copy Me implementation awaited one reviewed selective-capture/storage/handoff decision. Provider mapping questions were resolved centrally without changing the shared SPI. |
| 4 | Did adapter fan-out work? | Yes for the independent command consumers and disjoint native tests exercised here. It does not prove arbitrary native renderer implementation can proceed without coordination. |
| 5 | Did an adapter require Core modification? | No. No Core, Domain operator, schema, shared model/SPI or dependency changes. |
| 6 | Did an adapter depend on another output? | No. Desired values come from canonical resources; each command reads its own mapped captures. Native controls reject sibling projections as authority for the exercised cases. |
| 7 | Which coupling was an architecture defect? | No shared architectural defect was reproduced. GitHub DTO mismatch was a real consumer interoperability defect; .NET namespaces and capture ambiguity were adapter-local evidence defects. Shared native dispatch is an explicit coordination boundary. |
| 8 | Did Init/Copy Me stay separate? | Yes: scope/acquisition/workspace preparation versus evidence interpretation/candidate/decision validation. Neither automatically adopts or publishes policy. |
| 9 | Could Konfyra be a consumer? | Yes for fixed read-only inventory. It did not become a Core branch or change existing guidance; migration and benefit remain untested. |
| 10 | Which risks became defects? | Foreign XML lookalikes, incomplete protocol DTO decoding and ambiguous capture parsing. The risk-register stream itself reproduced no new product defect; it hardened existing trust checks. |
| 11 | Which needed only tests/docs? | Plan/observation tampering, target mismatch, projection owner collisions, human-body/sibling-output independence, adoption uncertainty and the shared capture proposal. |
| 12 | Main integration hotspot? | Shared renderer paths and canonical docs remained coordinator-owned. Shared Git refs/merges were serialized. Absolute write destinations were a concrete tooling issue; no source merge conflict required new semantics. |
| 13 | Was repo-owned context sufficient? | Sufficient for these bounded assignments, with recorded coordinator narrowing and mapping decisions. Not sufficient to invent a durable adoption record; that gap is explicitly deferred. |
| 14 | What changes before wave two? | Make real DTO and actual-runner conformance a required adapter proof; review selective acquisition/storage/handoff once; enforce absolute worktree writes and preserve ignored private evidence separately. |
| 15 | What is ready next? | A narrow capture/handoff design decision, then separately owned selective Init capture and Copy Me validation; further offline consumer proofs through the established command seam. Live provider access and native shared-renderer changes need their own reviewed scopes. A controlled adopter task comparison needs separate privacy/authority review. |
| 16 | Stable enough for an RC? | Candidate consideration requires final combined Windows/Linux gates and main verification. No known blocker remains in reviewed slices, but this is not release verification, supported-contract adoption, live-provider proof or product-utility evidence. No automatic release; published v0.12.0 stays current. |

## Deferred scope and next priority

Close this wave without adding language primitives or converting observed conventions into authority. The highest-value next implementation decision is the smallest explicit selective-capture and retention/handoff contract. It should let Init prepare fixed selected bytes without acquiring unselected contents and let Copy Me validate a separate candidate dossier without scope discovery. Avoid macros, implicit selectors, source-language inference or automatic adoption.

The next product-validation decision remains whether a narrow Markitect workflow improves a real engineering task compared with existing owner docs and tests. This wave proves bounded subsystem independence and catches specific adapter defects; it does not repair historical negative findings or demonstrate productivity, tokens, fewer defects in adopting code, demand or general safer autonomy.

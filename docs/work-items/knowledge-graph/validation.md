# Knowledge Graph validation ledger

## KG01 documentation candidate

Source reference: `1495e1be7b0046711531fa42c8407fba67b8e814`.
Candidate/remote at publication: `8f57b9c22caf24370422692ff62c0c215cba269e` on `origin/codex/knowledge-graph`.
One independent fresh Luna High Reader returned PASS with no material corrections, no edits and no executed tests. Exactly twelve questions were checked against source. README local targets exist.

| Executed check | Result | Log SHA256 |
|---|---|---|
| source-built root check at candidate | exit 0 / passed | `90470D9E4B1502F8F579D59C2C0DF42DA18828DAA954A5E00C02099EB722B801` |
| selected development engineering-change Context at candidate | exit 0 | `263499B294199DC0DF9D223B2F7D05B2B1EDC15EA05DB0C84216CA1B870DCB47` |
| fixed BASE-to-candidate Impact | exit 0; all-resources conservative expansion for new unowned documentation inputs | `4F9F3F12F90CF608DEE54157F957B239877ED957CAC1CE7D0731FCC90D1A0FEB` |
| working-tree managed-artifact helper with unchanged config | exit 0; new KG files NOT covered | `839ED6F05ACA4E57598387CEF6DFE00AAA2CC9DC674F7DFB670F0854C94C2F95` |
| git diff --cached --check | exit 0 | no log digest recorded |
| git ls-remote origin branch | exact candidate match | source fact above |

Fixed candidate snapshot digest from root check/Context: `ac2123c0e0e6297c9a90809253b4f44977c9c0adf65f51bbcdfca2addb739302`. Source-built tool digest remained `sha256:4c7b7010d8a359a4dbee00fad466323c738b04b7bfd92a0884c43c205165a911`. Accounting helper used working-tree inputs with its own snapshot digest `2af660ded37c1050f3c6e0ca4d259f4acb73e7c0789705519d39452a3359d35e`; it is not a fixed-candidate Verify result.

**Unverified ownership coverage:** `docs/work-items/knowledge-graph/**` is outside the pinned managed roots/tooling declarations. New module files will also require Integration-owned artifact configuration reconciliation. This workstream does not change the shared configuration to simulate coverage. Root and Integration were notified with the concrete paths.

No Shop cancellation execution, full Go suite, full Project Verify, real provider journey, authenticated human acceptance, Main merge, release or study result follows from KG01. Later checks must record their actual tested source separately.


## Coordinated development time limits

Root relayed the user's timing preference during the in-flight fixed `08b273f6` Verify: future focused development checks use 10 minutes, broad package checks 30 minutes, total suite 60 minutes and an outer process allowance of at least 90 minutes. Longer concrete workloads can justify an increase. This changes development timeframes only; graph query ceilings, provider/study grants, source ownership and Main integration do not change.

The already running Verify retains its original declared 1,800-second Go command limit and `-timeout=30m`. It must finish or reach that original timeout without interruption/restart. Any timeout must retain the actual result and be diagnosed; there is no automatic retry. Earlier focused `-timeout=2m` results are historical executed evidence, not the future default. Shared Project check declarations remain Integration-owned and were not edited here.


## KG02/KG03 code candidate

Code and remote publication: `08b273f62723c61e9445238d49dfa990dc1d6a7e` on `origin/codex/knowledge-graph`; BASE remains `1495e1be7b0046711531fa42c8407fba67b8e814`. Later documentation-only commits do not acquire this suite's evidence as a new full Verify.

| Executed check | Actual result |
|---|---|
| source-built fixed root check | PASS / exit 0; log SHA256 `AED0B5B03203503E6E3D212AD289D59F18AF5BCABDB9E06FF9FDB89DE1622708` |
| selected fixed engineering Context | exit 0; log SHA256 `A064A200DC305570089C1AF7B13D99A4016E661780FC9D2FF387AC6E7F031DD2` |
| fixed BASE-to-code Impact | exit 0; conservative all-resources for unowned new KG paths; log SHA256 `9CFB54506DCEA26A2BECE17E1B2D9074A5754FDC7AEDE56D65C04386CB001F22` |
| scoped fresh module test | PASS: `go test ./internal/modules/projectknowledge -count=1 -timeout=2m` (historical original limit) |
| scoped and whole-source vet | PASS: `go vet ./internal/modules/projectknowledge`, `go vet ./...` |
| fresh architecture import gate | PASS: `go test ./internal/tooling/architecture -count=1 -timeout=2m` |
| CONTRIBUTING schema/example gates | All 20 exit 0; commands below; combined log SHA256 `D27D01775B78D28ADF2430F05C5F677E425E899878BDADA0BCE043F20B9A2866` |
| separate fresh Luna High module Reader | Initial depth-doc/payload-label findings repaired, reread PASS; no remaining material findings |
| race instrumentation | NOT RUN successfully: CGO disabled; no race PASS claim |
| fixed complete Project Verify | **FAILED / exit 1**, Go suite exit 1 after 1,270,417 ms (~21m10s), within original 1,800-second bound |

Fixed check/Verify source snapshot: `85c3188d88ef93a0f0d00fa1f211cfdfc708ea72816d754aee8dc19a8d1c6b32`. Source-built CLI digest remains `sha256:4c7b7010d8a359a4dbee00fad466323c738b04b7bfd92a0884c43c205165a911`; the new pure module is not wired into that CLI.

The [unaltered full Verify output](evidence/verify-08b273f6.log) is retained byte-for-byte, SHA256 `4B8EAAF56D6C286DE054EB8E4676873780FC63E56C5804ECE1F580E716EAF68B`. Architecture, managed-artifact and module-check checks exited 0. Go tests included **projectknowledge PASS (0.740s)**, Core/ProjectModel PASS, and projectrun PASS (1,254.930s). The gate stopped at go-tests; subsequent claude-protocol and codex-protocol checks were **NOT RUN** by this Verify.

### Full-suite failures and boundaries

- `examples/TestProjectWorldNativeCLIWorkflow`: total test 90.64s; `onboard --provider both --expect ... --write` reported context deadline. `examples/project_world_workflow_test.go:313` configures **30 seconds per CLI call**, not a 90-second deadline. Slowdown cause was not established. This is not a total-suite timeout.
- `internal/host/projectcli`: `TestBrownfieldManagerContextIsReadOnlyAndBoundToIteration`, `TestBrownfieldStagedManagerLoopBeginContextProposeAndIntegrate`, `TestBrownfieldApplyAdoptionAppliesModelAndRecordsTrustedReceipt` fail because fixtures omit explicit `DelegationEvidenceIDs`; the serialized request uses null, while the validator requires an explicit list (which can be empty).
- `internal/host/projectonboarding/TestPreviewMergesNativeFilesWithoutChangingBytesOutsideManagedBlock`: assertion at `onboarding_test.go:129` expects `local operation skill`, while `render.go:192` emits `relevant operation skill`. Prefix/suffix preservation check immediately before it passed; that does not establish the rest of Apply checks that the failure prevented.

The above shared paths have **no diff from BASE to code candidate**. They are outside this KG workstream; Integration owns repairs. No BASE suite replay was run, so an independently measured BASE failure or a performance root cause is not claimed. No automatic retry and no shared-source patch was made. Root and Integration received the exact candidate, failure names, source anchors and log digest.

The new KG docs/module/evidence paths remain outside the pinned managed roots. The helper's PASS does not cover their managed ownership; ordinary review/tests apply, and Integration reconciliation remains pending. No full Project Verify PASS, Main readiness, provider/study trial, product usefulness or human acceptance is asserted.

### Schema/example command accounting

All commands used `go run ./cmd/markitect` from the code worktree: root schema (1); checks for minimal, repository-layout, canonical-engineering, engineering-constitution, engineering-discovery, software-architecture, delivery-target-equality and benchmark/fixtures/v2 (8); format for canonical-engineering, engineering-constitution, engineering-discovery, software-architecture and delivery-target-equality (5); model for canonical-engineering, engineering-constitution, software-architecture and delivery-target-equality (4); selected Context for software-architecture engineering/Skill/implement-order and delivery-target-equality engineering/Skill/deployment-review (2). None used --write; existing fixture bytes were unchanged.

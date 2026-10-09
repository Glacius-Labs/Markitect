# Native project delivery validation

Date: 9 October 2026. This report extends the earlier [operations evidence](project-operations-2026-10-09.md). Earlier passing runs remain attached to their original source; they are not validation of the new delivery package.

## Source and retained failures

The first integrated checkpoint is `ab306e4f71c59883d73b3e257c6e05033c3f55cc`. It adds persistent Explore/readiness, resumable delivery, automatic committed-model history and iterative Brownfield adoption. Follow-up source adds actual per-Manager Brownfield invocation and bounded parent integration context, repairs test fixtures and refreshes native instructions and README.

The full local command `go test ./... -count=1 -timeout=30m` for that checkpoint failed. Three `projectrun` tests manually constructed briefing history after multiple accepted model commits, violating the strict earliest-model baseline contract. Their fixtures now use `EnsureAcceptedHistory`; production history checks were not relaxed. The three focused corrected tests passed in 33.257 seconds. The full failed log is retained at `%TEMP%/markitect-native-workflow-20261009/ab306-full-go-test.log`, SHA-256 `e88757934093fb7e92ac0e91f5a8767edf8caeac23cb7315781c25dda9a1da3d`.

The configured-module gate also rejected a stale pipeline byte digest after the hosted job timeout changed. The canonical pipeline configuration was corrected to the actual workflow bytes. Subsequent configured-module and managed-artifact checks passed with no findings. Hosted final-candidate checks remain separate.

The second coherent checkpoint is `809e7df14638593100d064d85bc484ece8667a5d`, retained in [PR #89](https://github.com/Glacius-Labs/Markitect/pull/89). Its [hosted CI run](https://github.com/Glacius-Labs/Markitect/actions/runs/37890746830) failed on both Linux and Windows. Linux rejected exact `0700` executable runtime assets and exposed a verifier subprocess inheriting ambient environment when every allowlisted variable was absent. Follow-up source explicitly accepts and pins `0700`, and supplies a non-nil empty subprocess environment. Windows additionally rejected private-log access lists on the hosted runner. Follow-up source creates directories with an explicit current-user owner and permits file-owner correction only after validating the exact current-user-only DACL, followed by strict re-verification. An elevated token's default owner group is the suspected cause; the original generic CI errors cannot establish its exact Win32 status. Hosted rerun confirmation is still required. Later workflow gates were skipped after the test failures and are not passing evidence.

The retained hosted logs are `%TEMP%/markitect-native-workflow-20261009/ci-809e7df1-linux.log` (SHA-256 `7ca6ba16288447c10cdcc0c7b135facbb48472184c1d489bd7e76377847a7218`) and `ci-809e7df1-windows.log` (SHA-256 `62a25e8ad65acb32b3f43721f6eaefe9e1812c752d898e2b4718e1bcd68a4e41`). These hashes describe the saved UTF-8 text logs, not a provider execution receipt.

## Executed focused evidence

| Observation | Result and boundary |
| --- | --- |
| Guided semantic Orders/Inventory change | Controlled subprocess fixture passed: implementation/review/integration, full Manager verification, guarded Apply, scope completion and accepted-event resolution; no provider account used |
| Durable delivery failures and resume | Controlled fixtures passed: stale inputs and failed persisted runs do not silently create a replacement budget |
| Brownfield ledger and guarded adoption | Focused session/CLI checks passed before the new invocation bridge; later bridge changes require their own checks |
| Recursive parent integration | Focused tests passed for bounded child reports, exact child integration digests and duplicate-child rejection |
| Brownfield invocation CLI | Preview/stale-guard test passed in 8.594 seconds, with actual runtime fingerprinting and a controlled invoker; no provider started. A broader two-test rerun was stopped during the reserved measurement window and is not counted as passing |
| Brownfield Manager execution | Process-backed propose/integrate, retained receipt recovery, failed-attempt retry and stale guards passed in 67.391 seconds before the later lock helper wiring; subsequent pending-attempt, bounded-context and per-call pricing checks passed in 12.727 seconds |
| Hard process interruption | OS lock contention and actual subprocess kill/reacquisition tests passed. Session and run ledgers use atomic replacement. Ambiguous leftover publication files or unknown provider outcomes stop recovery rather than replay an uncertain call |
| Static invocation recovery contract | Focused request-contract, context-boundary and retained-receipt recovery checks passed in 22.453 seconds. Changed instructions, schemas, artifact bytes or integration bindings invalidate recovery; spent remaining-budget counters alone do not |
| Metadata-only recursive delegation | Focused own/delegation pool and recursive integration checks passed. Explicit empty pools survive encode/decode, clone and disk round trips without broadening; report citations remain grounded in own raw evidence or exact direct-child final reports |
| Safe Brownfield CLI output | Three scoped CLI journey checks passed in 55.168 seconds. Metadata overviews expose no raw source/private child bodies; full AdoptionPlan and readiness diagnostics remain reviewable |
| Empty verifier environment | Exact subprocess regression and durable failed-verifier budget checks passed in 1.421 seconds with all allowlisted variables absent; ambient test markers no longer leak into the child |
| Exact private executable modes | Focused runtime/setup checks passed on Windows. The Unix `0700` permission-change assertion is skipped on Windows and still needs the next hosted Linux run |
| Windows private log ownership | Focused Windows ACL creation/verification tests passed locally. The alternate-owner regression skips on the local non-elevated token and needs hosted execution; no extra access-control entries are permitted |
| Malformed Windows ACL bounds | Independent review found an incomplete ACE-length guard. The actual parser now validates the complete fixed SID header before indexing; focused tests reject 8-, 9- and 15-byte truncated ACEs without a panic |
| Verifier preflight diagnostics | Focused tests passed in 0.943 seconds: failure before producing a receipt preserves its original error, creates no result/start, and still rejects missing or mismatched receipt bindings |
| Coordinating readiness and adoption preview | Focused model-only adoption CLI test passed in 43.722 seconds. Resume intentionally retains actionable blocking questions/conflicts for the outer coordinator, omits raw source/private report sentinels, and the resolved adoption plan retains its exact candidate YAML and digest |
| Native onboarding | Updated package checks passed in 55 seconds; preserve/idempotence and staged Brownfield guidance exercised without provider calls |
| Delegated native authority | Focused rendering check passed in 0.315 seconds: permitted delegated acknowledgements use the Manager's actual identity; unresolved material decisions and explicit human-review policy remain required boundaries |
| Public Shop CLI journey | Isolated executable workflow test passed in 39.9 seconds: check, coverage, contexts, document equality, onboarding and runtime-free Explore/readiness |
| Empty repository native setup | Actual `project init` and guarded `project onboard` passed on an unborn Git feature branch; no AI invocation |
| Earlier checkpoint auxiliary gates | `go vet`, dependency verification, schema, adapter protocol suites and documented CLI examples passed; the corrected final source still needs its own applicable gates |

The requirement map is in the [native delivery checklist](../design/project-world/native-work-item-delivery.md). Fakes and subprocess fixtures establish deterministic protocol behavior. They do not establish that a real Manager follows the workflow or produces a correct business model.

## Final-candidate and product proof status

Final coherent follow-up source, independent review, passing hosted Windows/Linux CI and ordinary main integration are pending. The completed hosted failure above is retained. No fresh full suite has been run locally after the follow-up changes. Heavy local checks and native product proofs are deferred during the explicitly reserved Scientist measurement slot; off-host CI may run.

No real provider invocation has started under the native product grant. Its ceilings remain three jobs, 7200 aggregate active outer/role seconds and 128 starts, including failed attempts and reviews. The proposed actors are `gpt-6-luna` with reasoning `high`, using the already installed native CLI. Runtime/configuration/executable pins and hard external accounting must be fixed before starting. A failed or uncertain start consumes the grant and is retained; no automatic refill is permitted.

The independent read-only source review found no remaining material request-contract, delegation or inner-Manager context boundary defect. The later Windows ACL review identified the bounds issue recorded above; that finding is repaired in the follow-up source. These reviews are separate from the pending hosted platform rerun and native product execution.

No release, human acceptance, comparative study result or economic advantage is established by this report. The published v0.14.1 binary remains separate from this source workflow.

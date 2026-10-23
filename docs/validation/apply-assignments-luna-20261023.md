# Agent assignment Apply: Luna runs and Main closeout

23 October 2026, Europe/Berlin. This record preserves the A1–A3 and early A4 delivery before the development chats are archived. It records source integration and observed runs; the [delivery plan](../implementation-plan.md#integrated-source-and-observed-workflow-23-october-2026) owns remaining work. No new provider run was started for this closeout.

## Result and source

Fresh Library Run 08 completed through the product `markitect apply` command on Main `c1ead43dda1577652b2d8319f3166353fb49e85e`: model assessment accepted, Manager implementation and corrections completed, a fresh independent Verify accepted the fixed Root commit, the project check passed, and guarded publication applied 23 paths to the local fixture. The output is a fresh realization of the five-open-loan, fourteen-day Library model. It does not establish M1-to-M2 preservation, the outside oracle, sustained reliability, release readiness or owner acceptance.

All 25 PRs #609–#633 are merged; their merge commits were read back as ancestors of the Main SHA above. The production path includes:

| Scope | Integrated changes |
| --- | --- |
| Assignment sessions and fixed Git results | [#612](https://github.com/Glacius-Labs/Markitect/pull/612), [#613](https://github.com/Glacius-Labs/Markitect/pull/613) |
| Delegation, parent review and ordinary Git integration | [#614](https://github.com/Glacius-Labs/Markitect/pull/614), [#615](https://github.com/Glacius-Labs/Markitect/pull/615), [#616](https://github.com/Glacius-Labs/Markitect/pull/616) |
| Focused assessment, retained corrections and fresh final Verify | [#617](https://github.com/Glacius-Labs/Markitect/pull/617), [#618](https://github.com/Glacius-Labs/Markitect/pull/618), [#619](https://github.com/Glacius-Labs/Markitect/pull/619), [#621](https://github.com/Glacius-Labs/Markitect/pull/621) |
| Parallel Manager startup/work and same-conversation tool feedback | [#623](https://github.com/Glacius-Labs/Markitect/pull/623), [#624](https://github.com/Glacius-Labs/Markitect/pull/624), [#627](https://github.com/Glacius-Labs/Markitect/pull/627), [#631](https://github.com/Glacius-Labs/Markitect/pull/631) |
| Git identity, bounded output and one persistent Git container | [#620](https://github.com/Glacius-Labs/Markitect/pull/620), [#625](https://github.com/Glacius-Labs/Markitect/pull/625), [#628](https://github.com/Glacius-Labs/Markitect/pull/628), [#629](https://github.com/Glacius-Labs/Markitect/pull/629) |
| Native Codex subagent handling and effective disablement | [#622](https://github.com/Glacius-Labs/Markitect/pull/622), [#626](https://github.com/Glacius-Labs/Markitect/pull/626) |
| Product CLI and the one-command fresh Library runner | [#630](https://github.com/Glacius-Labs/Markitect/pull/630) |
| Fixture build-path correction and bounded complete review delivery | [#632](https://github.com/Glacius-Labs/Markitect/pull/632), [#633](https://github.com/Glacius-Labs/Markitect/pull/633) |

The exact reviewed head of #633 was `753c998295e138d6963c72c9d74286bac3e43bcc`; independent full-diff review reported no findings after three concrete findings were repaired. Main's tree matched that head. Relevant local tests, Vet, structural preparation and architecture checks passed. Hosted CI was not started for #633; earlier broad CI and local runs exposed remaining legacy Pass/diagnostic failures, including a nondeterministic legacy Pass assertion. No overall green CI claim is made.

## Preserved attempts

The [outcome snapshot and evidence index](apply-assignments-luna-20261023.json) retains parsed results and SHA-256 bindings to the original files, both A1 lifecycle archives and the 23 published Run 08 files. Parsed copies do not replace the hashed original bytes. The archive root is `Documents/Codex` under the original Windows user profile. Native transcripts, original manifests and generated realizations remain outside the repository and outside the chats; transcript contents and login credentials are not copied here.

| Attempt | Entry and outcome | Observation |
| --- | --- | --- |
| 01 | External runner; incomplete before any Agent/provider | Static runtime settings capture failed. |
| 02 | External runner; incomplete before any provider message | The runner selected an adapter composition without assignment sessions. |
| 03 | External runner; incomplete after accepted model assessment | A child started a native Codex subagent; the then-current adapter rejected another conversation's events. Child execution was still serial. |
| 04 | External runner; incomplete after parallel startup | Root supplied a child-owned commit to `accept_child`, while the Host expected its fixed handoff commit. Native subagents also allowed lending to bypass its required reminders Manager. |
| 05 | Product CLI, Main `4b3af86a`; `not_completed` / `incomplete` | Root tried to consult a child before ending its initial delegation response with Go; the first response timed out. Two messages, 74 Git execs. |
| 06 | Product CLI, Main `22acde15`; `failed` / `failed` | Root confirmed a concrete fixture problem: the required build output `realization/library` collided with the Manager-owned directory. Ten messages, 198 Git execs. |
| 07 | Product CLI, Main `e55005f4`; `not_completed` / `incomplete` | An embedded binary Git patch made one review prompt 3,193,935 bytes, above the 1,048,576-byte channel. No final Verify or publication. 51 messages, 719 Git execs. |
| 08 | Product CLI, Main `c1ead43d`; `passed` / `success`, Apply exit 0 | Fresh Verify, project check and guarded publication completed. 38 messages, 583 Git execs. |

Run 06's correction moved the Manager-owned implementation directory to `realization/application/`; the required executable path and build command remained fixed. Run 07's correction preserves named binary changes, delivers complete text diffs and long child summaries in bounded messages to the same parent conversation, and permits a decision only after delivery. Invalid UTF-8 is explicitly refused; JSON expansion is checked through the real Codex codec in the source tests. Failed attempts keep their original identities and outcomes.

The earlier A1 adapter observation used normal Codex CLI ChatGPT login and Luna. Two separate successful lifecycle invocations occurred, `0eee498586678e21` and `185c3b845d13ad60`; the accidental parallel invocation was recorded as a coordination deviation, not combined into one run. The first run's original report remains at `A1-ASSIGNMENT-LIFECYCLE-20261022/runs/0eee498586678e21/result.md`. This series contains no successful real Claude lifecycle/Apply observation. Actual monetary cost is unknown without billing evidence.

## Run 08: what actually happened

The integrated command [scripts/run-library-apply.ps1](../../scripts/run-library-apply.ps1) fetched exact Main, built the product/runtime, created a fresh committed fixture and invoked Apply once. The run identity was `20261022T225232Z-7b3c7560`; its local archive is `Markitect-Library-Runs/library-luna-20261023-08`. Start was 23 October 2026 00:52:32 CEST and exit was 01:12:09 CEST, 19 minutes 37 seconds later. Limits were 120 Manager messages, 600 seconds per message and an absolute 45-minute deadline; all model roles used Codex Luna High with the dedicated normal CLI login.

All six Manager containers (Root, catalog, lending, lending/reminders, library and members) started within a few seconds at Implement. Root set shared interfaces before Go, then direct-child tasks ran concurrently. Root reviewed/integrated results serially. Same-child corrections reused the existing conversations and working copies; all six containers remained available during fresh independent Verify. The single Git container was `b189c65c2eed97fae8e43f4b0566dabb1a55720155312d70a3b5713e168b7d6a`, with 583 recorded Git execs.

The collaboration exposed substantive defects despite passing local child tests. Root returned library's duplicate production services so that the executable used members, catalog and lending. It also found a recursion between catalog inspection and lending's availability callback. Lending changed availability to read its own open-loan state directly and added a catalog–lending borrow/return regression test. Lending delegated overdue assessment and later registry correction to its own reminders Manager. Root subsequently required correct owner registry shards and the missing policy Definition links.

Root, the fresh Verify Agent and the project check all bound the same fixed result, `e10adff7ca4941bcccb2118854d80e9b41250c48`. Verify recorded no issues after independently inspecting services, transport, executable wiring, assertions, registry entries and configuration, running package/race tests and a build, and exercising the real executable's availability changes. The product's `project-tests` check exited 0 without truncation, timeout or uncertainty.

Publication reported `applied` for 23 paths. Readback found exactly those 23 new fixture files and a passing structural check; the fixture's baseline HEAD stayed `926777a1c707f35448ccedd838b88f892cfc27c4`, with the published files uncommitted as intended by the product boundary. No target commit or push was introduced. The run report and stdout agree on `passed` / `success`; stdout SHA-256 is `8570b56e99d72e81c655545db8ce3c7327c3352e043aa4945ee7c85c0da32e15`. All run containers closed and all eight session homes were reported removed. Original transcripts remain retained and hashed in the evidence index.

## Main audit and remaining work

The A1, A2, A3 and coordinator-owned delivery worktrees were checked against Main. Their committed heads are already integrated and their working trees were clean. An older uncommitted consumer draft at `fix/a4-library-runner` is superseded by #630; its writer explicitly identified it as non-authoritative. It must not be merged as another start path. Its patch, untracked script and manifest were additionally retained at `Documents/Codex/Markitect-Apply-Closeout-20261023` without changing the original worktree. The primary checkout's empty untracked `internal/apply/assessment_test.go` contains no missing implementation; Main already tracks the actual test file.

Full A4 consumer/self-model/runtime guidance migration, A5 obsolete-code removal and the broader A6 model-change/preservation/oracle sequence are unfinished. The single successful fresh Apply does not close them. Keep these obligations in the delivery plan; the archived chats are not required to recover the current source, invocation path, observed outcome or remaining scope.

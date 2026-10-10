# Native Codex checkpoint and shared contracts — 9 October 2026

This report closes P01 source work and records P02 shared contracts. It does not establish native product acceptance. YAML is the sole canonical model format. Markdown remains readable documentation; provider-required SKILL.md front matter remains discovery metadata. The product queue and its fresh-chat handoff (retired on 10 October 2026; in git history) supersede earlier phase stops.

## Implemented checkpoint

Standard project setup selects scoped native Codex work using the existing installation/account/profile and pinned selected instructions. Ten direct operation skills replace the old model-first router: init, extract, design, implement, cleanup, verify, apply, check, suggest and configure. Preview/write guards preserve custom existing assets and reject migration-target collisions. The obsolete root init, runner mode switch/proposal fallback and implicit Brownfield delegation-wide authority were removed. Explicit empty delegation pools mean no delegation.

Native Managers currently work in a bounded candidate file tree without Git history. Text additions/modifications are checked against actual bytes, ownership, scope, fixed instructions, control-plane exclusions, portable paths and finite limits. Helpers are disabled. Deletes, binary deltas and full Git/history context are not supported by this adapter. Manifest/delta metadata is adapter-reported and bound to stdout; it is not an independent OS process census.

Failed Manager receipts are persisted once when invocation identity exists. The selected instruction pin comparison now uses the required sha256: prefix. Distillation and Brownfield assessment explicitly select a separate unscoped read-only agent configuration rather than reusing a scoped native Manager. Required independent review, Verify and guarded Apply remain in place.

The P02 source adds compiling workspace/delta, application-operation, invocation-validation and lifecycle/configuration contracts. It supplies no Git workspace implementation, App Server transport or MCP server. P02 (retired on 10 October 2026; in git history) recorded the concrete API paths and reserved ownership.

## Source validation and review

Source commits preceding P02 are 6121a8adf1d599bd34b137a3875bed35a2b1ca57 (failed receipts), 9f1891d36da90231d34f5cf7fc2fc25437d6a83a (setup/pins), 18eabf96ece65825d8cd9a72c7c03692caf59157 (compatibility removal), and 521057d3df212744df0466399adc6dd3b30ab736 (queue binding/read-only assessment). The final pushed handoff binds P02 and the report to an exact source SHA.

Focused source tests passed for setup and agentexec, native instruction/runtime binding and failed-receipt persistence, ten-skill onboarding and guarded router deletion, explicit Brownfield pools/context/freshness, CLI ownership/current initialization, and read-only assessment selection. At 521057d3 the five selected CLI assessment regressions passed in 33.838 seconds; the Shop Python example passed 12 tests. Private command logs are retained under the own-project markitect-native-work-20261009/evidence package; they are not provider transcripts or public product proof.

Independent source reviews closed receipt loss, POSIX permission widening, marked non-native fallback, onboarding document-path collision, and scoped-native configuration reuse in Brownfield/distillation. The final read-only-binding rereview found no remaining defect in its bounded scope. P02 receives its own focused contracts/import/compile checks and independent source review; results belong to the source handoff.

Failures remain recorded: a prior six-package 5-minute Go run timed out in projectrun/projectcli Git/census/Apply tests; an earlier long projectcli test process was interrupted; root parsing/rendering initially rejected unbound operational YAML and owner collisions until explicit input bindings/owners were fixed. These are not successful full-suite results. Focused subsequent checks do not replace final supported-platform CI and P09 readiness gates. Earlier remote CI on b61eb445 is not validation of these newer commits.

## Closed 600-second proof

The private disposable single-Manager fixture had one lease from 2026-10-09T15:36:10.662298Z through 15:46:10.662298Z. It was a narrow zero-helper mechanics fixture, not the normal default project workflow.

| CLI attempt | Source | Result | Duration |
| --- | --- | --- | --- |
| First | 6121a8ad | exit 2 before adapter/provider launch: absolute check executable rejected by the bare-name command contract | 1.6690624 seconds |
| Second | 6121a8ad | exit 2 before adapter/provider launch: instruction RuntimeFiles digest comparison lacked its prefix | 2.6905133 seconds |

The fixture check command was corrected within the same lease. The source pin defect was fixed at 9f1891d3. A continuation was refused by the deadline guard before another CLI call. No adapter, Manager, provider or helper started: **zero actual product starts and no successful authenticated native proof**. Original stdout/stderr and result files remain preserved. First stderr SHA-256 is 8e6f564d5650988ad506172f2031d7584bb93447ad938328cb1154fcf340c06a; second is db515cde2566388c75e9c841829fe09b9b21ece22b5b391072c82514ae8393d9. Earlier driver/preflight mistakes also remain retained. This closed lease is not reusable.

## Remaining acceptance

The separate finite product acceptance grant is recorded in the native acceptance ledger (retired on 10 October 2026; in git history): A01–A03 are all not started, consumed jobs and role-start requests are zero. Fresh implementation/integration owners must first implement and validate P03–P08, then run the permitted finite jobs and final gates. A fresh session declaration, mock, schema export, skill/config file or source push is not an actual independent agent run, successful user journey, human acceptance or comparative advantage.

PR89 remains draft. The user authorized normal Main integration after positive required gates; P01/P02 is a development handoff, not that readiness result. No release is authorized.

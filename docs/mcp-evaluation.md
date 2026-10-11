# MCP evaluation (2026-10-01)

**Status:** dated record of the read-only MCP pilot. Its decision no longer describes the product: the MCP server now ships in the `markitect` binary as `markitect mcp`, with one tool per verb and a `--read-only` mode. For the current server see [Project operations](project-operations.md#mcp-server-and-tool-groups), [Provider adapters](provider-adapters.md#connect-the-outer-client) and the [verb table](design/verb-table.md#mcp). The text below is kept as it was written.

## Decision

The authorized bounded assessment is complete with the available evidence:
keep MCP experimental and retain the supported CLI as the installation path.
The adapter is not included in published Markitect binaries. A future product
decision requires repeated comparable tasks and a completed second-client
comparison; these are deferred criteria, not hidden completion claims.

Codex CLI 0.130.0 completed the same three read-only queries through CLI and MCP
on two public synthetic fixtures. All six direct YAML payload comparisons
were byte-identical. The documentation answers were supported; the package
CLI answer added an unsupported bindings claim and the MCP answer omitted
the Project context entry. Single-run times and reported token counters
cannot establish a speed or savings trend.

Claude Code 2.1.233 connected but could not run model tasks because its OAuth
session expired. At the user's explicit direction this remains documented
as blocked; the present assessment does not depend on repairing authentication.

## Evidence and reproducibility

- [Client report](../experiments/mcp-pilot/evaluation.md) owns the exact fixed
  inputs, client versions, setup failures, counters, answer review and limits.
- [Predeclared plan](../experiments/mcp-pilot/evaluation-plan.yaml) and
  [observed results](../experiments/mcp-pilot/evaluation-results.yaml) retain the
  original experiment. Historical snapshots and measured binary identities
  are not relabeled as the integrated candidate.
- [Prototype setup](../experiments/mcp-pilot/README.md) describes fixed startup
  repository/SHA inputs, allowlisted queries, protocol versions and boundaries.
- [Transport harness](../scripts/mcp-evaluation/run-transport.ps1) replays
  find, explain, and context against the onboarding fixture and compares
  CLI/MCP payloads, rejecting unknown tools and extra arguments. It is a direct
  protocol test, not a model task or a second-client comparison.
- [Onboarding](onboarding.md) is a separate scripted CLI workflow replay that
  includes writes and verification. MCP exposes no write or verify operation.

Integrated source preserves protocol negotiation for 2025-06-18 and 2025-11-25,
literal pinned arguments, inherited Git environment sanitization, lifecycle
checks and advisory read-only annotations. Unit and hosted Windows/Linux gates
check the source candidate; historical model runs are not rerun by those gates.

MCP may remove shell command construction after setup, but adds protocol and
permission setup. It changes the interface, not Markitect's deterministic
analysis or project ownership. Neither interface proves complete answers,
semantic correctness or human acceptance.


## Separate onboarding-fixture observation

The following retained report uses Codex 0.159.2 and the parcel-support
onboarding fixture. It is separate from the matched Codex 0.130.0 documentation
and package-consumer cells linked above. Its partial result and lack of a
matched CLI condition apply to this fixture only. Issues #32 and #34 retain
possible follow-up; they do not change the decision to conclude the present
assessment and keep MCP experimental. Historical identities and failures are
preserved as observed.

**Status:** Partial. One real Codex MCP task completed against the fixed snapshot, and the direct published-CLI/MCP stdio checks confirmed exact payload parity. The Codex response covered the requested identity, relationships, and context paths, but did not fully explain the policy-change destination; no same-prompt Codex CLI run completed, and Claude remains blocked. This is limited feasibility evidence, not a productivity comparison or trend. The [roadmap](implementation-plan.md) remains the owner of product direction.

### Question and task

The pilot asks whether a local MCP interface helps an authoring agent inspect an existing resource graph while preserving Markitect's deterministic answers. The shared actor prompt and oracle are owned by the delivery-service fixture: [prompt](../examples/onboarding/delivery-service/tasks/query-only-prompt.md) and [oracle](../examples/onboarding/delivery-service/tasks/query-only-expected.yaml). The evaluation runbook is [here](../scripts/mcp-evaluation/task.md). The fixture also has a separate policy-mutation task, which is CLI-only because MCP has no write, verification, commit, or impact tool.

The intended query comparison uses one immutable baseline snapshot. Both conditions receive the same read-only task and fixture. The CLI condition may use `find`, `explain`, and `context` directly; the MCP condition has the equivalent `find`, `explain`, and `context` tools. The MCP prototype does not expose `impact`, so fixed-SHA impact is scored as a separate CLI result and is not treated as MCP parity. The shared query task does not authorize source edits or project command execution.

Expected query-task success is determined from the fixture oracle before inspecting actor output: identify the canonical Skill entry and source path, its declared outgoing relationships, and the exact context input paths. The separate CLI-only mutation task is judged against its own path, semantic, verification, and impact oracle. Structural validity and requested semantics remain separate observations.

### Client and artifact setup

The measured CLI is the published `v0.5.0` Windows executable from the immutable release (SHA-256 `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178`). The MCP server is a local, unshipped stdio build based on PR 23 source commit `c801dc84fa2da2112910233115fa13f1f01b8d0d`, with the negotiation and read-only annotation changes described below (SHA-256 `5b0b0f2c2f06a309d9cf7e34adde102495a1be1fd397e23615099d9aa7528b82`). The prototype itself invokes the published CLI. The server is configured with a fixed repository and full commit SHA. It exposes only the three read-only query tools described above.

The checked-in task fixture is synthetic and contains no credentials or customer data. Each client configuration is session-local; no global MCP configuration is changed. The server and client run with read-only access. Client-supplied revision, repository, arbitrary command, and write requests are out of scope for the task and are boundary cases, not task-success evidence.

The evaluated clean baseline is `973f11d9cfc157610ed84f4e9582c3534c67018c`; its published-CLI snapshot digest is `3bb9fe30d2188c86bcce1bf77fc551bcbd6741f4728ea34f270f7d9cc3668e87`. The candidate for the separate mutation task is `26c88c133cf09e09dd7f03a630c8e470128c002d`. Adoption validated baseline format and project checks. The final Codex client used a same-SHA clone owned by the Codex process account.

### Protocol and measurements

Predeclare the task oracle, repository, base and candidate SHAs, client versions, and tool binary hashes before each scored run. A successful shared query task requires the oracle's owner, relationships, context paths, and evidence limits to be correct. Preserve failed or incomplete runs. Repeat each condition three times only if client availability and bounded usage allow; otherwise report the exact run count and do not describe a trend.

For each run, record completion, correction or recovery, MCP tool calls, elapsed time, returned context bytes, and client-reported token usage when available. Count bytes from the exact returned tool text; do not estimate tokens from bytes. Record unavailable metrics as unavailable. Report wrong-repository and wrong-revision attempts, invalid resource/path/package inputs, and unknown-tool requests separately with the observed rejection behavior. No metric is collected by telemetry.

### Client observations so far

| Client | Version | Observation | Scored task runs |
|---|---:|---|---:|
| Codex CLI from npm | 0.130.0 | A bounded read-only invocation reached model startup, but failed before a Markitect MCP tool result: it could not parse its cached model catalog (`max` was unsupported), logged a malformed user skill, and reported `process 38312 not found`. This is an invalid setup run. | 0 |
| Codex desktop CLI | 0.159.2 | One final bounded MCP task completed in the client-owned same-SHA clone: exactly three calls (`find`, `explain`, `context`), all successful, 24.457 seconds. An earlier attempt failed at initialize because the escalated client account did not own the sandbox-owned Git clone; Git reported dubious ownership (exit 128), and Codex surfaced a closed initialize connection. Cloning the same commit under the client account resolved startup without changing source, Git strict checks, or global trust. Follow-up is tracked in [issue #32](https://github.com/Glacius-Labs/Markitect/issues/32). | 1 partial |
| Claude Code | 2.1.233 | A one-time read-only compatibility probe stopped at Claude Code's project-trust prompt before MCP status could be checked. The prompt was dismissed without approving persistent project trust. A subsequent `claude mcp list` showed only the pre-existing global Docker MCP entry failing to connect; it did not validate the session-local Markitect configuration. Prior OAuth expiry is already recorded as blocked; no login refresh or task run was attempted. | 0 |

One additional Codex 0.159.2 invocation used the query-only task against the previously supplied baseline worktree, but is invalid and unscored. The worktree had an uncommitted formatting change during the run, and the prompt did not include the published CLI's concrete command and path. The actor therefore searched available MCP resources (none were configured in this CLI condition) and attempted `rg --files` and `Get-ChildItem -Force | Select-Object Name,Mode`; both PowerShell calls were blocked by the read-only client policy. It made zero Markitect CLI calls. Elapsed time was 52.2 seconds; usage was 471,329 input tokens (417,024 cached), 1,872 output tokens, and 1,167 reasoning tokens. This is setup/failure cost, not task usage or a valid CLI result.

The initial Codex login-status check returned an unauthenticated status under the restricted process context. An escalated, read-only invocation reached model startup, so this was not recorded as an authentication or quota result. All Codex runs used `gpt-6-luna` at high reasoning effort. Four setup attempts reported 181,994 input tokens total (157,696 cached), 1,044 output tokens, and 440 reasoning tokens. One invalid CLI actor invocation made no Markitect CLI call and reported 471,329 input tokens (417,024 cached), 1,872 output, and 1,167 reasoning tokens; its worktree had an uncommitted fixture normalization and its prompt omitted the concrete CLI command. The failed MCP startup reported 29,609 input tokens (25,088 cached), 713 output, and 599 reasoning tokens. The final bounded MCP task reported 47,959 input tokens (40,192 cached), 997 output, and 606 reasoning tokens. Setup and failed-attempt costs are not task usage or comparative evidence. Raw logs were kept in TEMP and are not committed.

### Bounded Codex MCP task

The successful client run used the canonical query-only task scope, pinned baseline `973f11d9cfc157610ed84f4e9582c3534c67018c`, read-only sandbox, and session-local MCP configuration. It used the root-built experimental server candidate (SHA-256 `8eb14365c51a49d517efd15bcda6d4eab12ab92b00ad8ac438c8e32cfd68e7e2`) with the published v0.5.0 CLI. It completed in 24,457 ms and made exactly the three requested calls with no errors or retries. Client-reported usage was 47,959 input tokens (40,192 cached), 997 output, and 606 reasoning.

The response reported `support/Skill/refund-triage`, its canonical source path `docs/support/refund-triage.yaml`, all four expected direct relationships, and all seven expected context paths. The response also said “No owner was returned.” This is accurate if “owner” means a separate human-owner metadata field; the returned canonical path identifies the YAML source file. The response did not fully explain the policy-change destination, so the overall task is scored partial rather than complete. The exact context tool text was 6,553 UTF-8 bytes. There is no matched Codex CLI task result and no productivity comparison.

### Deterministic transport and boundary checks

The stdio probe used the same experimental server build (SHA-256 `5b0b0f2c2f06a309d9cf7e34adde102495a1be1fd397e23615099d9aa7528b82`), published CLI (SHA-256 `6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178`), and clean baseline above. Initialize returned `2025-11-25`; `tools/list` exposed `find`, `explain`, and `context`, each marked read-only. Direct stdio calls to all three succeeded, with exact UTF-8 text equality between the direct published CLI output and MCP tool text:

| Tool | Exact text | UTF-8 bytes | SHA-256 of returned text | Direct CLI elapsed |
|---|---:|---:|---|---:|
| `find` | equal | 410 | `f94c65dae6d2d1e34e47eb660363940348ca5ac61307061b255b30ad2e73f829` | 1,382 ms |
| `explain` | equal | 1,805 | `e6b2857daedba04fb63e2d0f068b4a76a00ee88db4bc420614ba6328a149835c` | 773 ms |
| `context` | equal | 6,553 | `542688ea455534b70fccbecf3df72282e5d49c48418c793a4cb670aed93ce439` | 1,107 ms |

All three returned the pinned revision and snapshot digest and matched the fixture oracle. The context contained `markitect.yaml` and the six expected typed resource YAML inputs. These direct timings are CLI process times, not a client-side MCP latency comparison; no token estimate is inferred from the byte counts.

The same stdio transport returned JSON-RPC `-32601` for an unknown tool. Extra `path`, `package`, `repo`, and `revision` properties each returned an `isError=true` tool result with `unknown argument` (23, 26, 23, and 27 UTF-8 bytes respectively). The server fixes repository and revision at startup; the tool schema does not accept per-call overrides. Reproduce with [`run-transport.ps1`](../scripts/mcp-evaluation/run-transport.ps1), which runs the three published CLI queries, direct stdio calls, byte/hash comparisons, and boundary requests; raw responses remain in TEMP rather than the repository. These are deterministic protocol checks, not authoring-client evidence.

### Current result

One bounded real Codex MCP task completed with the expected three calls and returned the expected identity, relationships, and context paths; it is scored partial because it did not fully explain the policy-change destination. The earlier invalid CLI actor invocation and failed Codex startup are preserved. Direct CLI and MCP tool texts match exactly on the fixed baseline, and direct protocol checks reject unknown tools and unsupported path/package/repository/revision properties. Claude remains blocked under the existing OAuth/trust conditions. There is no matched CLI authoring-client result, so no productivity or comparative benefit claim is supported.

The completed CLI-only verification and fixed-SHA impact result are recorded in the [onboarding report](onboarding.md). The remaining evidence for a matched authoring-client comparison is a valid same-prompt CLI condition; follow-up is tracked in [issue #34](https://github.com/Glacius-Labs/Markitect/issues/34). Do not infer a productivity trend from one partial MCP task.

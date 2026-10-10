# Provider adapters

The current project workflow has two provider boundaries. The outer coding client (Codex or Claude Code) connects to Markitect's repository-local MCP server and coordinates the Work Item conversation. Markitect Host schedules inner Manager and reviewer roles through Codex CLI 0.162.0 App Server using the configured `gpt-6-luna` model and `high` reasoning effort. Claude Code 2.1.295 is supported as an outer MCP client, not as an inner worker runtime.

## Connect the outer client

Build or select the Markitect executable for this source checkout. From the project root, register its stdio server with the exact absolute executable and root paths:

```powershell
codex mcp add markitect -- C:\absolute\path\to\markitect.exe project mcp --repo C:\absolute\path\to\project
codex mcp list
```

```powershell
claude mcp add --transport stdio markitect -- C:\absolute\path\to\markitect.exe project mcp --repo C:\absolute\path\to\project
claude mcp list
```

Confirm that `markitect` is available through the client's MCP view (`/mcp`). Quote paths according to the shell in use. The executable and repository root are fixed when the server starts; do not replace the root per operation. The Codex and Claude clients keep their own MCP registration configuration. Markitect onboarding creates repository-local guidance only; it does not register a client or change global provider/account settings.

The MCP server advertises closed typed schemas. The outer agent should use those schemas for inspect, exploration, model edit, readiness, Brownfield, plan/run/status/resume/repair, verify, preflight, apply, and delivery operations. A preview digest must be passed back exactly to the matching write operation where the schema requires one. Failed or stale operations remain errors; do not translate them into success or invent missing fields.

## Inner role execution and workspace boundary

`project setup` supports the native Codex App Server runtime and pins its resolved executable/configuration. `project_doctor` inspects local tools and authentication prerequisites without starting roles. Authentication is not established by setup or doctor; only a real provider invocation can exercise it. Setup does not configure Claude, global MCP, editor, hooks, plugins, or account settings.

On Windows, setup defaults to `appServer.windowsSandboxBackend: mxc`; `--windows-sandbox-backend mxc` can also be specified explicitly. The adapter starts only its child Codex process with the documented `-c windows.sandbox=mxc` override before `app-server`; it does not change global Codex configuration. Windows Managed Policy remains in force. Receipts record the requested backend, while effective backend remains unavailable unless Codex provides an authoritative readback. Unsupported values and non-Windows execution fail closed without a fallback.

Setup applies one `:workspace` permission profile to its Managers, independent Review roles, and independent Verifier. For this profile, the adapter sets child-local and thread-local `approvalPolicy: never` so ordinary delegated work needs no per-file human prompt, and confirms the effective workspace-write sandbox and approval policy before dispatch. Tools use inherited user environment and authentication. Test scratch and caches belong in a fresh owned directory under inherited OS temp; Review and Verify must leave repository artifacts unchanged. Host still guards accepted Manager artifacts and rejects nonempty Review or Verify deltas; guarded Apply remains the bridge to the adopting checkout. Helpers inherit the parent role's profile. `--codex-profile` overrides this shared profile for every generated role; other profiles preserve their approval policy.

The shared default disables provider memories and login-shell startup only for the child process. Final executor and Verifier messages use `turn/start.outputSchema` for the role's semantic response. The model does not return invocation identity; the Host composes those fields from the trusted invocation and runs the existing strict decoder. Executor responses omit `evidenceRefs`, which the Host supplies as an empty array. Verifiers select request-derived short aliases; the adapter maps only known aliases back to the supplied canonical references, and existing verifier coverage still requires the complete union. Report invocations return an empty `candidateFiles` array because Host harvests actual workspace bytes; helpers return plain-text CandidateFile entries rather than the input Artifact's digest/base64 shape. Strict response decoding and candidate-delta checks remain authoritative. Create temporary scratch only when a test or tool needs it; on Windows MXC, create, use and remove that owned scratch within one shell call because temp paths can differ between calls.

Process adapters map evidence references from the request's artifact paths, Scope IDs and Policy IDs to bounded aliases and decode them back before Host validation. App Server executor responses use an empty Host-owned reference array; App Server Verifiers use the same bounded alias approach, with the exact required coverage set carried separately from other allowed request references. The adapter maps selected aliases back without filling omissions or dropping extras; exact Host coverage validation remains authoritative. Empty references do not establish semantic acceptance. Host still validates response bytes, observations and actual workspace changes. Successfully written outgoing RPC requests, including turn schemas, are retained in the existing private operational journal; this records what the Host sent and does not establish provider enforcement.

Native setup also sets `appServer.environmentMode: inherit` for App Server roles. This passes the caller's ordinary environment to the native child and binds its effective values into the role fingerprint and receipt digest without exposing them. The separate `Agent.Environment` list remains the selected-name policy for resolving declared-check executables; it is not an App Server environment filter in this mode. Process adapters keep their explicit environment allowlists. Inheritance does not prevent same-user access to OS-managed credentials or network resources.

Host schedules the model-declared Managers and records bounded work, integration and repair attempts. Inner workers receive bounded task context and candidate workspace ownership, with standard file, shell, and test tools under caller permissions. The provider may also expose bounded helper starts according to runtime policy; helpers are nested work, not additional model-declared Managers. The native lifecycle adapter reports observed child starts with **partial accounting**. The durable summary separates Host-started roots and observed nested starts; observed nested totals are a lower bound, and zero does not prove no child ran. The workspace bridge waits for the root and each child it observed to reach terminal state before harvesting/closing, but this is not an exhaustive process census or OS isolation.

The Host helper tool is available only to Manager work and integration invocations with an explicit write scope. Independent reviewers do not implement the overall goal or dispatch helpers. A Work review assesses the Manager's current delivery and delegation intent; child implementation files are future work and are not required in that candidate. An Integration review assesses the aggregate candidate, including delivered child outputs and required child artifacts. A correctly bound, decoded helper request rejected by argument or path-scope validation before child dispatch returns a failed tool result to the same parent turn after its failed reservation is durably recorded, allowing the parent to correct the request. Identity, replay, persistence and child-lifecycle uncertainty remain terminal errors.

Native executors can approve an individual file change within their delegated Manager or helper scope when App Server reports `workspaceWrite` and the live owned workspace identity matches. The request remains bound to its thread, turn and item; it cannot expand permission roots. Host artifact guards, nonempty Review or Verify deltas, protected or outside paths, unknown requests and permission escalations remain restricted. This carries the existing task delegation and does not authenticate human approval.

The workspace candidate is bound to the fixed accepted project revision and permitted paths. Candidate deltas are validated and staged; guarded Apply rechecks freshness and scope before writing the adopting checkout. Native receipts and digests bind observed inputs/results but do not prove provider identity, semantic correctness, human approval, or every filesystem effect outside the declared boundary.

## Limits and evidence status

Project runtime limits are configured in `.markitect/runtime.yaml`; they constrain declared Host scheduling and local accounting. Estimated cost weights are estimates, not invoices or hard billing controls. Missing provider usage remains unknown; reports distinguish complete, partial and unknown cost accounting and retain the known estimate without inventing token counts. Provider-side usage and independently launched processes may not be exhaustively observable through partial helper telemetry.

The readiness backlog's proposed native acceptance grant is bounded at 60 minutes per role, four hours per job, and 256 role-start requests. Those are prospective test limits, not completed provider evidence or universal product defaults. Bounded authenticated native attempts have run, but no native acceptance has passed. See the [project workflow](project-workflow.md), [operations reference](project-operations.md), and dated [readiness backlog](work-items/product-readiness/backlog.yaml).

Earlier `Project.spec.targets`, render projections, and command adapters describe preserved Project/Domain compatibility contracts. They are not the provider setup path for the current `.markitect/project.yaml` workflow. Consult [legacy CLI compatibility](usage.md#legacy-projectdomain-cli-compatibility) when maintaining a released installation.

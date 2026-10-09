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

On Windows, setup can optionally set `appServer.windowsSandboxBackend: mxc` with `--windows-sandbox-backend mxc`. The adapter starts only its child Codex process with the documented `-c windows.sandbox=mxc` override before `app-server`; it does not change global Codex configuration. Omission preserves the existing default. The setting selects a sandbox implementation and does not grant permissions or change the selected permission profile; Windows Managed Policy and approval requirements remain in force. Receipts record the requested backend, while effective backend remains unavailable unless Codex provides an authoritative readback. Unsupported values and non-Windows execution fail closed without a fallback.

Native setup also sets `appServer.environmentMode: inherit` for App Server roles. This passes the caller's ordinary environment to the native child and binds its effective values into the role fingerprint and receipt digest without exposing them. The separate `Agent.Environment` list remains the selected-name policy for resolving declared-check executables; it is not an App Server environment filter in this mode. Process adapters keep their explicit environment allowlists. Inheritance does not prevent same-user access to OS-managed credentials or network resources.

Host schedules the model-declared Managers and records bounded work, integration and repair attempts. Inner workers receive bounded task context and candidate workspace ownership, with standard file, shell, and test tools under caller permissions. The provider may also expose bounded helper starts according to runtime policy; helpers are nested work, not additional model-declared Managers. The native lifecycle adapter reports observed child starts with **partial accounting**. The durable summary separates Host-started roots and observed nested starts; observed nested totals are a lower bound, and zero does not prove no child ran. The workspace bridge waits for the root and each child it observed to reach terminal state before harvesting/closing, but this is not an exhaustive process census or OS isolation.

The workspace candidate is bound to the fixed accepted project revision and permitted paths. Candidate deltas are validated and staged; guarded Apply rechecks freshness and scope before writing the adopting checkout. Native receipts and digests bind observed inputs/results but do not prove provider identity, semantic correctness, human approval, or every filesystem effect outside the declared boundary.

## Limits and evidence status

Project runtime limits are configured in `.markitect/runtime.yaml`; they constrain declared Host scheduling and local accounting. Estimated cost weights are estimates, not invoices or hard billing controls. Missing provider usage remains unknown; reports distinguish complete, partial and unknown cost accounting and retain the known estimate without inventing token counts. Provider-side usage and independently launched processes may not be exhaustively observable through partial helper telemetry.

The readiness backlog's proposed native acceptance grant is bounded at 60 minutes per role, four hours per job, and 256 role-start requests. Those are prospective test limits, not completed provider evidence or universal product defaults. Real authenticated native acceptance remains **NOT RUN**. See the [project workflow](project-workflow.md), [operations reference](project-operations.md), and dated [readiness backlog](work-items/product-readiness/backlog.yaml).

Earlier `Project.spec.targets`, render projections, and command adapters describe preserved Project/Domain compatibility contracts. They are not the provider setup path for the current `.markitect/project.yaml` workflow. Consult [legacy CLI compatibility](usage.md#legacy-projectdomain-cli-compatibility) when maintaining a released installation.

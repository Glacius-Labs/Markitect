# Setup ownership and runtime lifecycle

Markitect owns its real product installation: binary/module/schema/config pins,
project model, generated guidance/Skills, product MCP registration and genuine
product runtime/provider configuration. The Playground records that setup and its
cost; it never substitutes a Scientist implementation or assumes the product is
ready. Conventional owns an ordinary project installation without those method
mechanisms. Both methods need a bound runtime/control surface before work starts.

Current source-bound Markitect setup is in the [96424410 connection plan](markitect/connection-plan-96424410.md).
Integration owns the generic product install/guidance/authorization flow; Scientist
or the normal own operator owns public case fixtures and installation/input freezes.
Windows product setup now defaults to child-local MXC and a shared `:workspace`
profile for Manager/Review/Verifier; empty-delta guards preserve assessment-only
roles. Native client MCP registration remains separate from product onboarding.

| Runtime mode | Who starts it | What must be known before dispatch |
|---|---|---|
| Ordinary CLI | User/controller | Exact executable/version/bytes, cwd, instructions, model/effort, rights/config/auth source |
| Owned App Server over stdio | Owning wrapper or Markitect adapter | Exact executable/arguments/config; private stdin/stdout connection; initialize/initialized; native thread/turn IDs |
| Existing endpoint | Server/service owner | Transport/address, readiness, access/auth source, server version/config, connection ownership and shutdown policy |
| Operator MCP over stdio | Operator's MCP client | Server launch command, protocol initialize, tool schemas, handles and disconnect behavior |
| Markitect product MCP | Markitect setup + consuming agent | Genuine product launch/registration, project binding and exposed product operations |

The current Conventional wrapper implements **owned stdio**, not attaching to an
existing network service. It launches Codex App Server on demand and knows its
process PID and pipe connection. No daemon, port discovery, endpoint or credentials
in a prompt are required. Root-thread completion is not a guarantee that every
native helper or OS descendant terminated; direct-child ownership remains explicit.
If another consumer needs an attached server, its owner must supply that separate
endpoint contract before selecting it. No such mode is silently inferred.

Freeze runtime setup separately from general product setup: exact executable and
source/config/input byte hashes; user-auth source without secret values; loaded
instructions/Skills/tools; requested/effective model/effort/provider; cwd/writable
roots; approvals; helper model/resources; server start owner and shutdown policy;
native initialize readiness and thread/turn receipts; setup interval and failures.
Recorded requested configuration is not proof that it took effect. A successful
model turn is stronger access evidence than a version or model catalog response.

The wrapper's typed `runtimeOptions` can explicitly bind `workspace-write` or
`read-only`, `never` or `on-request` approval, user-memory activation and matching
native helper model/effort. CLI start/resume use frozen local TOML overrides. App
Server uses the same local settings plus explicit thread sandbox/approval fields.
No global configuration is edited and no broad sandbox bypass is introduced.
Unknown options and helper model/effort substitutions are rejected. The native
non-interactive default is read-only, so a coding pilot must select write access.

The current Conventional pilot uses workspace writes with approval `never`, and
disables old user memory in child-local configuration to exclude prior case
solutions/private results. Both choices are declared approximation differences
from an interactive user session. Existing project/user guidance is retained;
actor prompts remain the ordinary backlog request. Requests needing unavailable
rights are real pilot findings; no automatic promotion to broader rights follows.

The [official App Server protocol](https://learn.chatgpt.com/docs/app-server) owns
stdio startup and initialize/thread/turn semantics; the [non-interactive guide](https://learn.chatgpt.com/docs/non-interactive-mode)
documents CLI permissions. These interface documents do not establish runtime
equivalence or scientific results. [The finite pilot plan](pilots/conventional-wrapper-runtime-pilot-20261009/plan.json)
is a newer Conventional-only order; it does not reopen old attempts or require
Markitect to finish first. Paired method comparisons remain unbound.

The first native pilot exposed mutable user-config drift and a Windows shell setup failure. Original bindings remain untouched. For the still-unused second trajectory, explicit child-local model/effort/rights/memory/helper settings are pinned in the exact config; the full user-config hash is an observation, not an admission pin. Inherited settings and actual loaded config remain unverified. This is a prospective correction, not ordinary CLI parity or a change to old evidence.

Installed-runtime errors outrank assumed protocol shapes: Codex 0.162.0-alpha.2 rejected thread sandbox `workspaceWrite` and explicitly expects `workspace-write`. The wrapper now preserves the native enum value; readiness includes successful thread admission and tool access, not merely initialize. A separate one-time correction order uses a fresh BF repo/context while retaining original failures and the original overall deadline.

The schema-1 adapter interface is `describe`, `setup(context)`, `ensure_runtime`, `start(prompt)`, `resume(run_id, prompt)`, `status(run_id)`, `cancel(run_id)` and `close`. Its manifest names one explicit factory and pins its source bytes. Typed observations are evidence receipts; passive observers cannot alter scheduling or task acceptance. Setup receives a prepared isolated repository and public setup resources. A missing runtime or external endpoint returns blocked; the adapter owns ensure/start/stop for its own service and preserves unresolved ownership.

Conventional setup installs ordinary rules and a frozen resource profile in the prepared repository. Its readiness distinguishes App Server protocol initialization or CLI version from sandbox file/Git access; none starts a model. Each native check consumes a finite order reservation and records exact command, PID, timing and raw bytes. The new Windows candidate selects documented `windows.sandbox="mxc"` with child-local configuration and managed restrictions. It avoids the observed legacy ACL refresh failure on a locked runtime file; effective full model/tool behavior still requires actual execution evidence. No global configuration is edited. The supported native cap is three spawned threads plus the primary, giving four total roles; depth and a shared native start counter remain cooperative/unverified.

The new attempt demonstrated that writing a scratch Git repository does not establish Git readiness for the real repository: default workspace-write protects `.git` recursively. The earlier corrective readiness check used a unique metadata file; the new source removes that write and reports only read-only Git CLI readiness. Actual Git mutation readiness, when required by the selected target, needs an observed approved native command. With approval `never`, protected metadata is a blocker. With explicitly enabled scoped App Server approvals, readiness is conditional until the native request, one-time decision and command completion are observed. The broker accepts only bound single local Git actions; unknown, compound, out-of-repository, destructive, network and policy-extension requests are declined. The original final assessors use a fully read-only sandbox with no Git approval broker; the separately ordered scratch correction below permits only disposable test artifacts.

A station completes only after a new captured main commit contains the selected work and the index/worktree is clean apart from the controller station marker. Native terminal success and an ordinary agent reply alone cannot release the next wave or another case. Failed/cancelled/uncertain trajectories stop the controller. [Closure](pilots/conventional-variant-adapter-delivery-20261009/closure.json) records the early-transition deviation and the separate unintended Roombook start; all originals remain preserved.

The disabled [scoped Git example](conventional/config.scoped-git.example.json) is App Server only. It requires exact PowerShell/Git executable digests, `allowLoginShell=false`, and the Git argv prefix `-c core.hooksPath=/dev/null -c core.fsmonitor=false`. Scoped execution excludes system/global Git configuration in the child environment and declines repository configuration outside a small inert allowlist. This removes hooks, configured filters, signing and custom merge-driver execution from the approved path; it is a declared approximation to an ordinary interactive CLI environment. The decision remains bound to the observed local command item, owned repository/thread/turn and one server request. No session or execution-policy grant is issued. The native effectiveness of this route has not been tested; the historical execution order remains closed.

The prospective `workspace_snapshot` target requires no actor Git mutation or Git broker. Runtime admission checks the ordinary sandbox file access and declared protocol; setup freezes mechanically adapted rules. The actor edits and tests normally, and the controller captures those exact files, checks them, changes only the declared station marker, and assesses a fresh final candidate. Repo Git history/main remains provenance and receives no completed-work credit in this mode. No historical result is reinterpreted. The matched future Markitect adapter must receive the identical completion target and functional inputs.

The actual workspace run completed all four station checks. Its original final evaluator could not run persistence tests in a fully read-only sandbox. The separate final-only correction keeps original candidate and requirements outside the writable CWD and executes an exact copy under its own scratch root. Every existing copied file, both requirements trees and preparation receipts must remain hash-identical; only new `.scratch` and `.assessment-output` files are allowed. Bytecode suppression is scoped to the assessor process, with the previous environment restored afterward. No candidate repair, helper, Git broker, new case or implementation retry is admitted. The original error report is immutable and any corrected scientific result is additive.

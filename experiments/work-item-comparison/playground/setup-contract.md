# Setup ownership and runtime lifecycle

Markitect owns its real product installation: binary/module/schema/config pins,
project model, generated guidance/Skills, product MCP registration and genuine
product runtime/provider configuration. The Playground records that setup and its
cost; it never substitutes a Scientist implementation or assumes the product is
ready. Conventional owns an ordinary project installation without those method
mechanisms. Both methods need a bound runtime/control surface before work starts.

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

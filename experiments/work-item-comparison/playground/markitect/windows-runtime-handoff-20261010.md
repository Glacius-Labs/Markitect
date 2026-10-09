# Observed Windows Codex launch configuration

Sanitized technical handoff from retained native execution receipts. No new
execution, probe, installation, credential read, global edit or container start.
No case inputs, actor prompts, candidate content or evaluation results are included.
These observations concern the recorded executable, not an untested current binary.

Executable: `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe`

- Observed version: `codex-cli 0.162.0-alpha.2`
- Recorded SHA-256: `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`
- Requested primary model/effort: `gpt-6-luna` / `high`; native thread receipt
  reported the same and provider `openai`.

## Process launch and environment

The wrapper launched a direct child in its absolute owned workspace CWD, using
stdin/stdout/stderr pipes and newline-delimited JSON messages. Exact ordered argv
after the executable for the write-capable actor was:

```json
[
  "-c", "sandbox_mode=\"workspace-write\"",
  "-c", "approval_policy=\"never\"",
  "-c", "features.memories=false",
  "-c", "allow_login_shell=false",
  "-c", "agents.default_subagent_model=\"gpt-6-luna\"",
  "-c", "agents.default_subagent_reasoning_effort=\"high\"",
  "-c", "windows.sandbox=\"mxc\"",
  "-c", "agents.max_concurrent_threads_per_session=3",
  "app-server"
]
```

There was no `--listen` argument: this executable used default stdio. Product
Markitect's explicit `--listen stdio://` is a different launch shape; equivalence
is not established by this observation. No named permission profile was supplied.
The wrapper inherited the current-user environment/authentication. It did not
copy credentials, change global settings or isolate inherited configuration.
Scoped Git approval was disabled; no Git approval environment overrides or broker
were active. Requested memory/login-shell/helper settings are configuration, not
proof of every inherited/effective setting.

## Wire fields

Initialize used `{"clientInfo":{"name":"markitect-conventional-playground","version":"1"}}`,
then notification `initialized` with `{}`. The following are exact parameter
shapes with private CWD, thread ID and task text replaced by placeholders:

```json
{"method":"thread/start","params":{"cwd":"<owned workspace>","model":"gpt-6-luna","sandbox":"workspace-write","approvalPolicy":"never"}}
{"method":"thread/resume","params":{"cwd":"<same owned workspace>","model":"gpt-6-luna","sandbox":"workspace-write","approvalPolicy":"never","threadId":"<same native thread>"}}
{"method":"turn/start","params":{"cwd":"<owned workspace>","model":"gpt-6-luna","threadId":"<native thread>","input":[{"type":"text","text":"<ordinary task>"}],"effort":"high"}}
```

`turn/start` supplied no separate sandbox, sandboxPolicy or approval field.
Native thread binding reported approval `never` and sandbox
`{"type":"workspaceWrite","writableRoots":[],"networkAccess":false,"excludeTmpdirEnvVar":false,"excludeSlashTmp":false}`.
Thus request enum `workspace-write` and response type `workspaceWrite` differ.
Explicit extra writable roots were not needed for ordinary writes in the CWD.

## Observations and limits

Real ordinary tools wrote/read project files and ran tests. Native helper threads
were genuinely started, including overlapping execution; the requested helper
default was Luna/high. Helper effective model/effort and exhaustive descendant
accounting were not established. The native cap of three spawned threads excludes
the primary; a hard total-start/depth limit was not proved. Subsequent user tasks
resumed the same outer thread; this is distinct from Markitect's original-turn
recovery contract. The wrapper confirmed its direct child, not every OS descendant.

Ordinary filesystem write/read/remove and read-only Git commands worked. Protected
`.git` mutation under `workspace-write`/`never` remained unavailable. Git CLI does
not bypass that protection. No Git mutation/merge readiness is claimed.

The earlier legacy/elevated backend failed with `setup refresh had errors` and
Windows error 32 during read/execute ACL refresh of locked `cua_node/node_repl.exe`.
The MXC path started and supported ordinary native work without changing global
ACLs, stopping foreign processes or disabling Managed restrictions. This is a
successful alternative for that recorded startup failure; it does not establish
that MXC fixes Git top-level aliases, product workspace identity/harvest, every
Windows error, or the stable `0.162.0` binary. Markitect's currently evidenced
stable inner version differs from this alpha outer version. Bind both exact
binaries/configurations before reuse; do not infer a hardcoded product restriction
from a version requirement alone.

Inherited user guidance/configuration was not fully frozen. Inherited Ralph Loop
command stop hooks failed; no global plugin change occurred. No hook-free or fully
isolated environment claim follows. Writable test scratch avoided read-only temp
file failures; the separate frozen-copy assessor additionally used process-local
`PYTHONDONTWRITEBYTECODE=1`, restored afterward, and disabled native helpers. Those
assessment-specific choices are not part of the coding launch above.

Reuse the ordinary child-local backend/rights settings and explicit protocol
receipts before adding wrappers or a separate capability laboratory. Keep product
workspace identity, genuine setup, guarded Apply and recovery with their owners.
Case fixture preparation/input freeze remains with Scientist or the normal own
operator/actor. This handoff authorizes no new execution or installation and makes
no method comparison claim.

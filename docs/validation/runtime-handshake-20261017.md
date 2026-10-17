# Agent runtime handshake — 17 October 2026

Validation record, 17 October 2026. The assistant made these throwaway probes with the owner's approval for a few real provider turns ([source](../concepts/sources/runtime-discussion-20261017.md)). They test whether the Host can drive both coding CLIs, give them a Host tool and keep ambient configuration out. They are not product code, not an adapter test and not a reliability measurement. The scripts were disposable and are not kept.

## Setup

- Host: Windows 11 with Docker Desktop 29.4.1 on WSL 2 (Linux 5.15).
- Codex: `codex-cli 0.162.0`, the npm native binary on the host and in the container.
  - The App Server ran over stdio and the client opted in with `capabilities.experimentalApi`.
  - The schemas came from `codex app-server generate-json-schema`, with and without `--experimental`. They have 447 and 317 files; `dynamicTools` appears only in the experimental `ThreadStartParams`.
- Claude Code: 2.1.301 on the host and 2.1.296 in the container.
  - It ran as `claude -p --input-format stream-json --output-format stream-json` with `--strict-mcp-config`, an explicit `--tools` allowlist and `--permission-mode dontAsk`.
- Container: an existing local Debian 12 image with both CLIs, Python 3.11 and Git.
  - The probes ran as user 1000. Workspaces and provider homes were named volumes; nothing came from host bind mounts.
  - Each session received a copy of the provider login in its own volume. Every volume was deleted after the probes.
- Task: create `hello.txt` with one line, call the Host tool `submit_work(summary, files)`, then reply with the delivery ID the tool returns.
  - The Host tool was either a client-handled Codex dynamic tool or a minimal stdio MCP server that recorded each call.

## Results

| Run | Path | Outcome | Time | Tokens or cost |
| --- | --- | --- | --- | --- |
| 1 | Codex on the Windows host, dynamic tool | Completed, tool call correct, file written. The user's MCP servers `MCP_DOCKER`, `node_repl` and `codex_apps` started despite `-c mcp_servers={}` and disabled features. The agent wrote the file through `node_repl`. | 32 s | 194,141 tokens |
| 2 | Claude on the Windows host, MCP tool | Completed; only `Read`, `Write` and the Host tool were offered; the tool call was received with its tool-use ID. The init event still listed bundled skills and a user auto-memory path. | 6.5 s | $0.047 |
| 3 | Codex in the container, dynamic tool, sandbox `workspace-write` | The command failed: `bwrap: No permissions to create new namespace`. The agent still submitted "Created hello.txt", and the file did not exist. | 6.9 s | 29,164 tokens |
| 4 | Codex in the container, dynamic tool, sandbox `danger-full-access` | Completed, file written, no MCP server started. | 9.6 s | about 29,000 tokens |
| 5 | Claude in the container, MCP tool | Completed, file written, tool call received. | 11.4 s | $0.053 |
| 6 | Codex in the container, MCP tool instead of the dynamic tool | Completed, file written. The MCP call carried the thread, turn and call IDs in `_meta`. | 10.2 s | not recorded |

## Findings

- **Both CLIs can be driven over stdio, and a Host tool reaches both through MCP.** Codex also accepts the experimental dynamic tool.
- **Runs on the host inherit the user's environment.**
  - Codex loaded and used user-level MCP servers that the probe had tried to remove.
  - Claude kept user-level paths unless started in a clean home.
  - A clean Linux home inside a container removed all of this without enumerating features.
- **Codex's own sandbox does not start in an unprivileged container.** The container itself has to be the boundary.
- **An Agent's report is not evidence.** Run 3 claimed a file that did not exist. The Host must observe files and run checks itself.
- **Shells on Windows rewrite paths.** The Git for Windows shell turned the container path `/work` into `C:/Program Files/Git/work` until path conversion was disabled. A Go Host calling the engine directly is not affected, but scripts are.
- **Copies of desktop logins are unsafe as runtime credentials.** Both providers rotate OAuth refresh tokens, so a copy that refreshes can invalidate the desktop login. The probes ran only while no refresh was due, which a token-expiry check confirmed.

## Limits

- **One task per path, run once.** There was no interruption, resume, concurrent session or long task.
- **Provider costs are lower bounds.** Codex ran on a subscription login, which reports tokens but no price.
- **The container image was not pinned by digest,** and the Claude versions differed between host and container.
- **No egress restriction, resource limit or Markitect session helper was tested.**

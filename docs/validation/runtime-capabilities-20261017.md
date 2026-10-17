# Runtime capability probes — 17 October 2026

Validation record, 17 October 2026. These probes answer the runtime questions that the step-basis design raises before dependent implementation: can a provider conversation continue in a new environment, can a running session see a workspace change after it became quiescent, and can the Host observe quiescence? They run the container adapter's gated tests against the [runtime base image](../../packaging/runtime/README.md) with a real Claude Code session. They are capability evidence for the runtime, not an adapter, a Manager process or an Apply.

## Setup

- Docker Desktop on WSL 2, Docker Engine 29.4.1, on a Windows 11 host.
- Images built from source on 17 October 2026:
  - the runtime base image with Codex `0.162.0`, Claude Code `2.1.301` and markitect-agentd at frame protocol 2;
  - the Go example image on top of it.
- Every container ran under the production hardening:
  - user 1000:1000, `--init`, read-only root filesystem, all capabilities dropped and `no-new-privileges`;
  - bounded CPU, memory, processes and `/tmp`;
  - named volumes only.
- Claude Code used a long-lived token from `claude setup-token`, passed only to the provider process. Later Claude probes ran Claude Haiku with high effort.
- Codex later used a dedicated runtime login, created by device-code authorization in its own Docker volume and never the desktop login. Each session received a copy of that login without the refresh token, written into its provider home by a helper container. Codex probes ran Luna (`gpt-6-luna`) with high effort.

## Results

| Probe | Outcome |
| --- | --- |
| Claude smoke | One real turn wrote `hello.txt` and called the Host tool `submit_work` over MCP through `markitect-agentd mcp`. A helper container read the file back from the workspace volume. |
| Continue in a new environment | Container A told Claude a code word and ended. A helper container then wrote `basis.txt` into the workspace volume while no session ran. Container B mounted the same provider-home and workspace volumes and resumed with `claude --resume`; the answer named both the code word and the new file content. |
| Quiescent basis advance in one session | After a turn the census showed only `claude` and its MCP bridge (`markitect-agent`). A helper container changed `basis.txt` in the mounted workspace, and the next turn of the same process read the new content. |
| Census in a session container | With a provider that started a background child, the census listed exactly the provider and the child, never agentd or the container's init. An ended child appears as a zombie (`Z`) until its parent reaps it. |
| Project check in the Go image | A helper container ran `go test ./...` on a workspace module and passed, with Go's caches in the provider home. |
| Write barrier | A shell provider rewrote a counter file in the workspace every 50 ms. While the engine paused the container, a helper read the same value twice, one second apart; after unpausing, the value advanced again. |
| Barrier-protected basis advance with Claude Code | After a quiescent turn the Host paused the session, changed `basis.txt` through a helper container and unpaused it; the next turn of the same conversation read the new content. |
| Codex without a credential | `codex app-server` reported the Host MCP server `ready` inside the hardened container. No turn ran. |
| Codex turn | With the session login, one real turn wrote `hello.txt` and called the Host tool over MCP through `markitect-agentd mcp`; a helper read the file back. |
| Codex continues in a new environment | A persistent thread (`ephemeral: false`) learned a code word in container A, which then ended. A helper changed `basis.txt`, and container B resumed the thread with `thread/resume` from the same provider home. The answer named both the code word and the new content. |

## Findings

- **A conversation can continue across environments.** For both CLIs, the retained provider home is enough. Claude Code must not run with `--no-session-persistence`, and Codex threads must not be ephemeral.
- **A Codex subscription login can serve sessions without racing.** The session copy lacks the refresh token, so no session can refresh or rotate the login. The runtime login stayed valid after the probes. Two easier routes do not work: `codex login --with-access-token` expects an agent-identity token, not a ChatGPT access token, and the App Server's `chatgptAuthTokens` login is marked for OpenAI internal use only.
- **A workspace change between turns works without remounting.** A helper container writes into the same volume that the session has mounted. A changed mount plan still needs a new environment, as the design states.
- **Pausing the container is an effective write barrier with a live provider.** A frozen Claude Code session continued normally after a short pause.
- **The census makes quiescence observable.** The provider's own MCP bridge stays alive between turns. Quiescence therefore means "only the provider's known processes remain", not an empty table. A census is a snapshot and cannot prevent a later start.
- **`/tmp` must allow execution.** Docker mounts a `--tmpfs` with `noexec` by default, and `go test` failed with `permission denied` on its test binary in `/tmp`. Because the workspace is executable anyway, the runtime now mounts `/tmp` with `exec`.

## Limits

- **Single runs.** Each probe ran once per CLI.
- **Login refresh is not covered.** Refreshing the runtime login before it expires is a maintenance step outside the sessions; no probe covered it.
- **No mid-turn cases.** The probes covered no interruption during a turn, no recovery after a Host crash and no concurrent sessions.
- **The quiescence rule is a test assertion,** not a Host implementation.

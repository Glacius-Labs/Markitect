# Container runtime adapter

This package is the Host's client of the container engine ([DEC-037](../../../docs/concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). It starts hardened session and helper containers, speaks the [agentd frame protocol](../../agentd/frame/frame.go) and manages volumes and images. It knows no tasks, models or reviews. The [rebuild plan](../../../docs/design/canonical-model/manager-apply-rebuild-plan.md#container-runtime) owns the runtime contract, and the [base image](../../../packaging/runtime/README.md) its image contract.

## API

| Call | Use |
| --- | --- |
| `NewEngine(path)` | Binds the absolute engine executable. The Host never searches PATH. |
| `Engine.Start(ctx, spec)` | Runs one session container with agentd and returns its `Session` after agentd's hello. |
| `Engine.RunHelper(ctx, spec, argv, stdin, stdout)` | Runs one command in a fresh container under the same hardening, for example `git`, `/usr/bin/tee`, `/bin/cat` or a project check. |
| `Engine.RunCommand(ctx, spec, argv, env, maxOutputBytes)` | Runs a literal command with fixed environment additions, observes actual container start, terminal state and image, and retains bounded combined output with truncation. A known target nonzero exit is a result. |
| `Engine.InspectVolume`, `VolumeMounts`, `CreateVolumeExclusive` | Observes volume labels and every container mounting the exact volume, and refuses adoption of preexisting volumes. Exclusive creation verifies a private random label after Docker's idempotent create; Docker provides no atomic exclusive-create primitive. |
| `Engine.Pause` / `Engine.Unpause` | Freezes and resumes every process in a container; the write barrier for a workspace-basis transition. |
| `Engine.CreateVolume`, `RemoveVolume`, `Containers`, `RemoveContainer` | Named, labeled volumes, and the label sweep for containers the Host no longer controls. |
| `Engine.ImageID`, `ReadImageFile` | Image identity and the image's recorded metadata, read without running the image. |
| `Engine.ObserveImageFile(ctx, imageID, path, maxBytes)` | Streams one complete regular image file into SHA-256 and actual size, up to an explicit bound of 512 MiB, without starting the image. Any archive, process, cancellation or cleanup failure refuses the observation. |
| `Session.StartProvider`, `WriteInput`, `CloseInput`, `Stdout` | The provider CLI's process and standard streams. Output is buffered up to 32 MiB, so control replies never wait for the reader. |
| `Session.ToolCalls`, `Reply` | Host tool calls arriving over MCP through `markitect-agentd mcp`. `ToolCall.Meta` carries the provider's own call metadata. |
| `Session.Census`, `Stop`, `Exited`, `Diagnostics`, `Close` | The container's processes, stop with grace, the observed exit, the tail of standard error, and the end of the connection. |

## Integrating a session adapter

These rules come from real runs, recorded in the [handshake record](../../../docs/validation/runtime-handshake-20261017.md) and the [capability record](../../../docs/validation/runtime-capabilities-20261017.md).

- **Codex runs with sandbox `danger-full-access`.** Its own sandbox cannot start in an unprivileged container. With `read-only` or `workspace-write`, every shell command fails, and the Agent may still report success. Read-only roles get read-only mounts instead.
- **Host tools go through MCP.**
  - Codex: `-c mcp_servers.markitect.command="/usr/local/bin/markitect-agentd"` and `-c mcp_servers.markitect.args=["mcp"]`, with the features `apps`, `memories`, `multi_agent`, `image_generation`, `browser_use` and `computer_use` disabled.
  - Claude Code: `--strict-mcp-config --mcp-config '{"mcpServers":{"markitect":{"type":"stdio","command":"/usr/local/bin/markitect-agentd","args":["mcp"]}}}'`, an explicit `--tools` and `--allowedTools` list, and `--permission-mode dontAsk`.
  - [cli_smoke_test.go](cli_smoke_test.go) and [cli_continuity_test.go](cli_continuity_test.go) show complete invocations.
- **A conversation that may continue must be persistent.**
  - Codex threads are not ephemeral, and `CODEX_HOME` stays in the provider home (`/home/agent/.codex`).
  - Claude Code runs without `--no-session-persistence`.
  - A new container on the same provider-home volume then resumes: Claude Code with `claude --resume <session>`, and Codex with `thread/resume`. Both are demonstrated.
- **Credentials:**
  - An API key or Claude Code's long-lived token goes in `frame.Start.Env`, never into container environment or arguments. Record that message with the value redacted.
  - A Codex subscription uses a dedicated runtime login, never the desktop login. Before a session starts, a helper writes a copy of that login into the provider home with the refresh token removed. The session then cannot refresh or rotate the login, and the access token outlives a run.
  - That access token is a full-account credential that the Agent can read. Remove the copy from the provider home when the session ends.
- **Provider messages can be large.** A tool result can carry a whole file, so size per-message limits for that.
- **Quiescence before a basis change:**
  1. Close admission and reconcile admitted actions.
  2. Take a `Census` for the record, then `Pause` the session container. The MCP bridge (`markitect-agent`) stays alive between turns, and zombies (state `Z`) have ended.
  3. Change the workspace through `RunHelper` and confirm it.
  4. `Unpause`. The next message names what changed, because a frozen provider keeps its open files and in-memory view.

  Pausing also freezes agentd, so a paused session answers no control request: not `Census`, not `Stop` (its grace starts only after `Unpause`) and not the graceful end of `Close`. Unpause before any of them; a `Close` on a paused session removes the container by force only after its timeout.
- **Fresh volumes take over ownership from the image.** The image owns `/home/agent` and `/work` by user 1000, so fresh volumes mounted there are writable for the session user.

## Tests

Unit tests run everywhere. Real-engine tests run with `MARKITECT_TEST_ENGINE` set to the engine's absolute path. Image smokes and provider probes also need `MARKITECT_TEST_RUNTIME_IMAGE`, `MARKITECT_TEST_GO_IMAGE` and a runtime credential; the [base image README](../../../packaging/runtime/README.md#smoke) shows the command.

`RunCommand` retains the container for a bounded independent state readback before removing its observed full ID. A CLI exit does not supply the target exit code. Missing terminal state, interrupted execution and failed cleanup retain uncertainty; callers must preserve it. The output limit is between one byte and 64 MiB, independently of the 4 MiB bound for engine metadata. Image environment defaults remain in force; `env` supplies literal additions without host environment inheritance. Volume inspection and census failures never count as absence or an unused volume. A failed exclusive creation may leave a volume behind and never authorizes its deletion.

Set `Mount.NoCopy` for a volume whose bytes are already fixed and verified. Docker otherwise populates an empty volume from the target directory in the container image, which can add files when a check uses a different image. The default remains ordinary image copy-up, used to initialize session home/work directory ownership.

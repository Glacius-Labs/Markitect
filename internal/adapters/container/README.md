# Container runtime adapter

This package is the product's client of the container engine ([DEC-037](../../../docs/concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). It starts hardened session and helper containers, speaks the [agentd frame protocol](../../agentd/frame/frame.go) and manages volumes and images. It knows no tasks, models or reviews. The [V1 execution architecture](../../../docs/design/v1-execution-architecture.md) owns the runtime contract, and the [base image](../../../packaging/runtime/README.md) its image contract.

## API

| Call | Use |
| --- | --- |
| `NewEngine(path)` | Binds the absolute engine executable. The Orchestrator never searches PATH. |
| `Engine.Start(ctx, spec)` | Runs one session container with agentd and returns its `Session` after agentd's hello. |
| `Session.Initialize(ctx, req)` | Installs the bounded fixed private login file over that connection before provider startup; no shell, supplied path or extra container. |
| `Session.ClearPrivateHome(ctx)` | Confirms removal of the installed login after provider exit and before container closure. agentd retries cleanup on disconnect and reports failures. |
| `Engine.RunHelper(ctx, spec, argv, stdin, stdout)` | Runs one command in a fresh container under the same hardening, for example `/usr/bin/tee`, `/bin/cat` or a project check. Run Git uses the retained exec service. |
| `Engine.StartExec` / `ExecSession.Exec`, `Close` | Holds one hardened container with fixed mounts and executes literal commands as its configured non-root user. Cancellation removes the owned container so remote processes cannot outlive the failed exec. |
| `Engine.StartOwnedExec` | The explicit Run Git exception: adds only `CAP_CHOWN`, verifies empty private assignment roots and initializes their ownership with one fixed root exec before returning a non-root-only session. Ordinary engine/Agent/check launches are unchanged; the [workspace service](../gitrepo/WORKSPACES.md) records the lifecycle. |
| `Engine.RunCommand(ctx, spec, argv, env, maxOutputBytes)` | Runs a literal command with fixed environment additions, observes actual container start, terminal state and image, and retains bounded combined output with truncation. A known target nonzero exit is a result. |
| `Engine.InspectVolume`, `VolumeMounts`, `CreateVolumeExclusive` | Observes volume labels and every container mounting the exact volume, and refuses adoption of preexisting volumes. Exclusive creation verifies a private random label after Docker's idempotent create; Docker provides no atomic exclusive-create primitive. |
| `Engine.Pause` / `Engine.Unpause` | Freezes and resumes every process in a container. |
| `Engine.CreateVolume`, `RemoveVolume`, `Containers`, `RemoveContainer` | Named, labeled volumes, and the label sweep for containers the Orchestrator no longer controls. |
| `Engine.ImageID`, `ReadImageFile` | Image identity and the image's recorded metadata, read without running the image. |
| `Engine.ObserveImageFile(ctx, imageID, path, maxBytes)` | Verifies that executable ancestors are actual directories, then streams the complete regular image file into SHA-256 and actual size, up to an explicit bound of 512 MiB, without starting the image. |
| `Engine.ObserveImageFileWithRequiredPaths(ctx, imageID, path, maxBytes, requiredDirectories, requiredRegularFiles)` | Also positively verifies the supplied directories and regular-file paths, including their directory ancestors, in the same never-started container. Unknown, absent, redirected or mismatched paths refuse the observation. |
| `Session.StartProvider`, `WriteInput`, `CloseInput`, `Stdout` | The provider CLI's process and standard streams. Output is buffered up to 32 MiB, so control replies never wait for the reader. |
| `Session.ToolCalls`, `Reply` | Run tool calls arriving over MCP through `markitect-agentd mcp`. `ToolCall.Meta` carries the provider's own call metadata. |
| `Session.Census`, `Stop`, `Exited`, `Diagnostics`, `Close` | The container's processes, stop with grace, the observed exit, the tail of standard error, and the end of the connection. `Close` returns `ErrForcedClose` (joined with the engine process result) when the container did not end within the close timeout and had to be removed by force; a nil result means it ended by itself. |

## Image file observations

An image-file digest alone cannot exclude ancestor aliases: `/opt/tools/check` can be copied as a regular file while `/opt/tools` links to `/work`, which a check later mounts over. Docker's Linux archive code preserves the leaf type while accessing its parent path ([Moby source](https://github.com/moby/moby/blob/3f38b15147b14ce0ce7b11558a1aa9e9f14db103/daemon/archive_unix.go#L59-L75)). The observation therefore checks each unique ancestor from root outward. A consumer can also supply its fixed mount directories and managed file targets; this prevents accepting a mount destination that itself redirects to the executable tree. The consumer owns selecting the relevant paths.

Path probes read only the first logical tar header, bounded to 1 MiB of metadata, and require its exact name and type with no link. They deliberately stop and reap the copy CLI after that header; a successful reap or confirmed requested kill is required. Each probe has a 60-second deadline, shortened by the caller's context, and a separate 15-second inherited-pipe bound. The header deadline allows for the first access to a cold image on Docker Desktop; observations are not retried. These are positive path/type observations, not complete archives or file-content certificates. The executable still requires the complete tar stream, successful copy exit and mandatory bounded container cleanup. Cancellation or uncertainty discards its hash. At most 256 unique paths are admitted, with one short copy per path before the executable copy. Symlinked layouts and missing required paths are refused; no fallback rewrites the configured executable.

A directory probe costs the size of its subtree. Docker 29.4 builds the whole archive of a directory and holds the container's lock until it is done, even after the copy CLI was stopped at the first header; the next `cp` or `rm` on that container waits for it. Measured in WSL on the Go runtime image, the probes of `/usr`, `/usr/local` and `/usr/local/go` caused about 16 s, 9 s and 6 s of later waiting (a full copy of `/usr` is 1.5 GB), while `/usr/local/go/bin/go` and small directories take 0.03 to 0.2 s. No cheaper query establishes that an ancestor is not a symlink: `docker cp` is the only engine command that reports a path's own type, and it cannot do so without that archive. An executable under a small tree such as `/opt/...` avoids the cost.

`Engine` remembers each successful observation, keyed by the immutable image ID, executable, size bound and the exact path set with its types, for the life of that Engine, and identical concurrent requests share one run. Failures, cancellations and uncertain cleanups are never remembered. The checks package's `EngineImageReaders` keeps one Engine per executable path, so one composition's preparations of the same image observe it once; the memo lives as long as the composing process and providers share nothing. When the observation itself, or the caller's cancellation, stops the executable copy, the stop is not reported as an engine failure; anything the engine printed is kept.

## Integrating a session adapter

These rules come from real runs, recorded in the [handshake record](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-handshake-20261017.md) and the [capability record](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-capabilities-20261017.md).

- **Codex runs with sandbox `danger-full-access`.** Its own sandbox cannot start in an unprivileged container. With `read-only` or `workspace-write`, every shell command fails, and the Agent may still report success. Read-only roles get read-only mounts instead.
- **Run tools go through MCP.**
  - Codex: `-c mcp_servers={markitect={command="/usr/local/bin/markitect-agentd",args=["mcp"]}}`, with the features `apps`, `memories`, `multi_agent`, `multi_agent_v2`, `image_generation`, `browser_use` and `computer_use` switched off by `--disable` and `agents.enabled=false`. `providerArgv` in the [Codex adapter](../codexappserver/profile.go) holds the exact arguments.
  - [cli_smoke_test.go](cli_smoke_test.go) and [cli_continuity_test.go](cli_continuity_test.go) show complete invocations.
- **A conversation that may continue must be persistent.**
  - Codex threads are not ephemeral, and `CODEX_HOME` stays in the provider home (`/home/agent/.codex`).
  - A new container on the same provider-home volume then resumes the thread with `thread/resume`, as demonstrated.
- **Credentials:**
  - Provider environment additions go in `frame.Start.Env`, never into the container environment or arguments. Record that message with values redacted. There is no API-key route.
  - A Codex subscription uses a dedicated runtime login, never the desktop login. Before a session starts, a helper writes a copy of that login into the provider home with the refresh token removed. The session then cannot refresh or rotate the login, and the access token outlives a run.
  - That access token is a full-account credential that the Agent can read. Remove the copy from the provider home when the session ends.
- **Provider messages can be large.** A tool result can carry a whole file, so size per-message limits for that.
- **Fresh volumes take over ownership from the image.** The image owns `/home/agent` and `/work` by user 1000, so fresh volumes mounted there are writable for the session user.

## Tests

Unit tests run everywhere. Real-engine tests run with `MARKITECT_TEST_ENGINE` set to the engine's absolute path. Image smokes and provider probes also need `MARKITECT_TEST_RUNTIME_IMAGE`, `MARKITECT_TEST_GO_IMAGE` and a runtime credential; the [base image README](../../../packaging/runtime/README.md#evidence-and-bounded-probes) states the evidence boundary.

`RunCommand` retains the container for a bounded independent state readback before removing its observed full ID. A CLI exit does not supply the target exit code. Missing terminal state, interrupted execution and failed cleanup retain uncertainty; callers must preserve it. The output limit is between one byte and 64 MiB, independently of the 4 MiB bound for engine metadata. Image environment defaults remain in force; `env` supplies literal additions without host environment inheritance. Volume inspection and census failures never count as absence or an unused volume. A failed exclusive creation may leave a volume behind and never authorizes its deletion.

Set `Mount.NoCopy` for a volume whose bytes are already fixed and verified. Docker otherwise populates an empty volume from the target directory in the container image, which can add files when a check uses a different image. The default remains ordinary image copy-up, used to initialize session home/work directory ownership.

`(*Engine).RequireVolumeSubpaths(ctx)` reads `docker version --format '{{.Server.Version}}'` and refuses a server older than Docker 26 (volume subpath mounts) or an unreadable answer. Set `Mount.Subpath` to mount one directory of the volume (`volume-subpath`) instead of the whole volume. `ValidSubpath` accepts only a clean relative path without `..` or engine option characters. Targets stay unique. Docker 29 requires the directory to exist in the volume when the container starts, else it refuses to start; it applies mounts by destination depth, so a read-only base with a writable subpath mount (and the reverse) enforces the write rule for the session user, including nested overlays. Observed with real Docker probes: a read-only base with a writable subpath overlay refuses writes outside the overlay; a writable base with a read-only overlay refuses writes only inside it.

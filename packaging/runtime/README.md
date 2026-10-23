# Runtime base image

This directory defines the Linux image used by the native adapter and container checks. It contains `markitect-agentd`, pinned Codex `0.162.0` and Claude Code `2.1.301`, Node, Git, Python 3 and CA certificates. Installed CLI binaries do not establish adapter support or provider qualification. The current Codex adapter implements Run Sessions; the Claude adapter returns unavailable before credential or container side effects.

Build explicitly from the repository root when a new image is required:

```text
docker build -f packaging/runtime/Dockerfile -t markitect-runtime .
```

The [Dockerfile ignore file](Dockerfile.dockerignore) bounds the build context. Runtime preparation selects an existing immutable engine image ID (`sha256:...`); a tag or a source checkout alone is not that identity. `/opt/markitect/runtime.env` records CLI versions, the generated Codex schema digest and the agentd build. The metadata reader copies it from a created, never-started container and checks image identity again.

## Container and workspace ownership

The entrypoint is `/usr/local/bin/markitect-agentd serve`; its framed standard streams connect the host adapter to the native process and MCP bridge. Containers run as numeric user `1000:1000`, with a read-only root filesystem and explicit CPU, memory, PID, temporary-storage and network limits.

A Codex Agent assignment owns one container and one fresh private home at `/home/agent`. Its private Linux working-copy volume/subpath is mounted at `/work` without copy-up. The same native process and conversation serve subsequent messages until Session closure. Closing the assignment observes provider exit and removes its owned home; conversation continuity across replacement containers or retained homes is not the current contract.

LinuxGit holds one separate hardened Git service and private Linux volume per Run. Managers receive distinct working copies inside that volume. Durable candidate bundles live outside the disposable Run environment. Checks receive separate disposable input and home copies; they cannot alias the Manager working copy. See the [Git workspace guide](../../internal/adapters/gitrepo/WORKSPACES.md), [container adapter](../../internal/adapters/container/README.md) and [native Codex adapter](../../internal/adapters/codexappserver/README.md) for actual consumers and cleanup uncertainty.

`/tmp` is executable because native tools and project toolchains run build products there. A project image may extend this base with its required toolchain while preserving the entrypoint, numeric user and mount points. The [Go example](examples/go/Dockerfile) adds Go and a C toolchain; the [Python example](examples/python/Dockerfile) adds CPython 3.13. A selected check profile must explicitly provide its executable, arguments, limits and any required writable cache or temporary paths. Image contents alone do not qualify a project's checks or race tests.

## Account login and instruction loading

The current Codex consumer accepts an explicitly selected dedicated normal account login. The host resolver supplies its bytes through private agentd initialization after model, workspace and harness admission. It does not pass credentials through argv, environment, product Git, harness or Notes, and it does not use helper containers to provision login. No Codex API-key execution route remains.

Only account ID, ID token and access token are installed in the fresh private home. Refresh authority is removed from that copy; the source login is not rewritten. This is access-only session evidence, not refresh, expiry, fresh-login or concurrent fan-out qualification. Login is cleared after observed provider exit; uncertain shutdown or home removal remains an error. Native tools run as the same user as the provider and can read the installed session login.

The native renderer supplies versioned working rules, checked project and Manager context, mandate, selected project instructions and selected runtime instructions. CLI controls disable automatic project-document loading and native delegation and keep `/work` untrusted. Actual loading priority, conflicting repository instructions and native tool behavior require their own bounded evidence. See the [harness guide](../../internal/harness/README.md) and [Codex adapter guide](../../internal/adapters/codexappserver/README.md).

## Evidence and bounded probes

Source tests, image metadata and an installed CLI are separate from actual engine, login, provider, hierarchy and background-writer evidence. Real probes require a concrete finite coordinator dispatch with frozen source, binary, profile, existing image, resource ownership, time limits and independent closure observations. Preparation does not authorize build, pull, credentials, retries, inference or product Apply.

The opt-in offline CLI inventory uses a local HTTP recorder and no external inference. The native hierarchy probe separately exercises actual Run Sessions. Neither the old both-provider smoke commands nor API-key and cross-container resume examples describe current execution. Earlier [runtime capability records](../../docs/validation/runtime-capabilities-20261017.md) retain their historical source and scope; they do not qualify the replacement adapters or grant a rerun. Claude remains unavailable until separately implemented and qualified.

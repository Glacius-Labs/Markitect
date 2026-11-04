# Manager image

This directory defines the Linux image every maker and checker of a [concept 2.0](../../docs/concepts/v2-concept.md) Run runs in (§9). It contains `markitect-agentd`, `markitect`, Git, Python 3, CA certificates, and the pinned Claude Code `2.1.308` and Codex `0.162.0` CLIs. The [Manager agents guide](../../internal/agents/README.md#the-image) owns its contract: how a container of it is started, logged in and cloned from the Run's remote, and which launch flags the adapters pass. Installed CLI binaries do not establish provider qualification.

Build it from the repository root:

```text
docker build -f packaging/runtime/Dockerfile -t markitect-runtime .
```

The [Dockerfile ignore file](Dockerfile.dockerignore) bounds the build context to the Go module, `internal/` and the two commands. The host selects the image by its immutable engine image ID (`sha256:...`); a tag or a source checkout alone is not that identity. The build fails when a CLI is not the pinned version or lacks a launch flag its adapter passes, and `/opt/markitect/runtime.env` records `claude`, `codex`, `agentd` and `markitect`.

The native app builds this image for its chats by itself, from a copy of this recipe and its source that it carries ([`source.go`](../../source.go)). Tests check that the copy holds everything the recipe copies and this ignore file admits, and that it compiles.

## Container and ownership

The entrypoint is `/usr/local/bin/markitect-agentd serve`; its framed standard streams connect the host to the provider process and the MCP bridge that carries the Manager tools. Containers run as numeric user `1000:1000` with a read-only root filesystem. `/home/agent` and `/work` belong to that user, so the fresh volumes the agent runtime mounts there, the private home and the clone, are writable for it. `/tmp` is executable because toolchains run build products there.

## Project images

A project's checks and its Managers need the project's toolchain. A project builds its own image FROM this one and keeps the entrypoint, the numeric user and the mount points. The [Go example](examples/go/Dockerfile) adds Go and a C toolchain; the [Python example](examples/python/Dockerfile) adds CPython 3.13. The image must not set `ANTHROPIC_*` or `CLAUDE_CODE_*` variables or ship `/etc/claude-code` or `/home/agent/.claude`; agentd refuses a Claude home that is not empty.

## Login

A session uses the user's own token from `claude setup-token`, which reaches it only through its environment ([DEC-093](../../docs/concepts/register.md#dec-093-a-command-passes-the-users-own-token)). The image holds no login, and never an API key. The image pins the CLI, so a change of the provider's defaults arrives only with a tested upgrade.

## Evidence

Building the image and running its containers without a provider is a source check. A real provider turn needs a bounded, owner-approved dispatch with frozen source, image, limits and cleanup ([working method](../../docs/development/working-method.md)).

# Runtime base image

Every Markitect Agent session and project check runs in its own Linux container ([DEC-037](../../docs/concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). This directory defines the product's base image. It contains:

- the session helper `markitect-agentd`, built from this repository;
- Codex `0.162.0` and Claude Code `2.1.301`, pinned in the [Dockerfile](Dockerfile);
- Git and CA certificates.

Build it from the repository root; the [ignore file](Dockerfile.dockerignore) sends only the agentd sources to the engine:

```text
docker build -f packaging/runtime/Dockerfile -t markitect-runtime .
```

## Contract

| Part | Rule |
| --- | --- |
| Entrypoint | `/usr/local/bin/markitect-agentd serve`. The Host talks to it over the container's standard streams. |
| User | Numeric `1000:1000`. The Host refuses root. |
| Writable paths | The Host mounts a provider-home volume at `/home/agent` and the workspace at `/work`, and a temporary directory at `/tmp`. The image owns both mount points by user 1000, so fresh volumes take over that ownership. The root filesystem is read-only. |
| Metadata | `/opt/markitect/runtime.env` records the Codex and Claude Code versions, the digest of the generated Codex App Server schemas and the agentd build. Preparation copies it out of a created, never started container. |
| Identity | The Host names an image by its engine image ID (`sha256:...`), which also exists for local builds. |

A project builds its own image `FROM` this one and adds the toolchain its checks need. The [Go example](examples/go/Dockerfile) adds Go for projects such as the library example and keeps Go's caches in the provider home. A project image keeps the entrypoint, the user and the two mount points.

## Session facilities

| Facility | Use |
| --- | --- |
| Census | agentd lists the processes in the container besides itself and init; zombies appear with state `Z`. The Host uses it to judge quiescence before a workspace-basis change. |
| Helper containers | `Engine.RunHelper` runs one command, such as `/usr/bin/tee`, `/bin/cat` or `git`, under the same hardening, to fill, read or check a workspace volume. |
| Conversation continuity | The provider home outlives its container. A new container on the same volumes can resume a conversation, for example with `claude --resume`. |

`/tmp` is executable on purpose: the workspace is executable anyway, and toolchains such as Go run their build products from there.

## Credentials

Runtime credentials are API keys or long-lived tokens, never copied desktop logins: providers rotate refresh tokens, so a copy can sign the desktop out. The Host passes a credential to agentd at start, and agentd gives it only to the provider process. Use scoped, spend-capped credentials, because the Agent can still read them.

## Smoke

The container adapter's tests drive this image when the engine and image are named:

```text
MARKITECT_TEST_ENGINE=/usr/bin/docker MARKITECT_TEST_RUNTIME_IMAGE=markitect-runtime \
MARKITECT_TEST_GO_IMAGE=markitect-runtime-go \
  go test ./internal/adapters/container -run 'TestRealEngine|TestRuntimeImage|TestClaude|TestProjectImage' -count=1
```

Without credentials, the smoke shows that both CLIs start in the hardened container and connect to the Host tools through `markitect-agentd mcp`. With `OPENAI_API_KEY`, and `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`, each CLI also changes a file, which a helper container reads back, and calls a Host tool. With a Claude credential, two further probes resume a conversation in a new container after a workspace change, and continue one session across a quiescent workspace change. Repeat the smoke for every new image; the [capability record](../../docs/validation/runtime-capabilities-20261017.md) holds the first results.

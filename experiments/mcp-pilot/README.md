# Markitect MCP read-only pilot

This experiment exposes three Markitect CLI queries as local MCP tools over
stdio: `find`, `explain`, and `context`. It is intended to test whether an MCP
client makes these read-only queries more useful during authoring than calling
the CLI directly. It is experimental and is not included in the published CLI
binaries.

## Build and run

Use the repository's Go toolchain and a Markitect binary. The MCP server pins
one repository directory and one full Git commit SHA at startup; those values
cannot be changed by a tool call.

```powershell
go build -o .\markitect-mcp.exe .\experiments\mcp-pilot
go build -o .\markitect.exe .\cmd\markitect
git -C C:\src\your-project rev-parse HEAD
.\markitect-mcp.exe --markitect .\markitect.exe --repo C:\src\your-project --revision FULL_COMMIT_SHA
```

Configure the MCP client to launch `markitect-mcp.exe` as a stdio server with
those arguments. Client configuration formats vary; use the client's documented
local-server configuration. Build paths and repository path should be absolute
in that configuration. The process writes protocol messages only to stdout and
diagnostics to stderr.

On Linux/macOS, replace the build and executable names with `go build -o
markitect-mcp ./experiments/mcp-pilot` and `./markitect-mcp`; supply absolute
paths to `--markitect` and `--repo`.

## Exposed tools

- `find(query, kind?, namespace?)` searches literal resource text and optional
  exact filters.
- `explain(kind, name, namespace?)` returns ownership and explicit direct
  relationships. Namespace is required for namespaced kinds and omitted for
  `Project`.
- `context(kind, name, namespace)` compiles the declared dependency closure
  and file inputs for a namespaced resource.

Results are Markitect's YAML output in MCP text content blocks, including the
fixed snapshot identity and digests where supplied by the CLI. This prototype
tests local tool discovery and invocation; it does not provide MCP structured
content or test whether typed results improve an author's work. The pilot does
not call a model or interpret the meaning, truth, or acceptance of resource
content.

## Boundaries and limitations

- This negotiates MCP `2025-06-18` and `2025-11-25` over JSON-RPC stdio. It supports the
  initialize handshake, `ping`, `tools/list`, and `tools/call` needed for this
  pilot. It does not implement resources, prompts, sampling, HTTP, or extensions.
  See the [MCP transport specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)
  and [tool specification](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).
- Tool arguments have explicit JSON Schemas, reject unknown fields, and are
  mapped to a fixed command allowlist. The server never accepts a command line
  or shell fragment from a tool caller. It invokes only `markitect find`,
  `markitect explain`, or `markitect context` with the pinned repo and SHA.
- The configured repository path and executable are trusted startup inputs.
  Markitect and Git still run with the launching user's operating-system
  permissions. This is not a sandbox for untrusted repositories or binaries.
- `context` can return full resource prose and declared file contents from the
  selected snapshot. The MCP client receives those results and may pass them to
  its model or service according to its own configuration. Use only a repository
  whose content may be shared with that client.
- The commit must be a full SHA resolvable in the selected Git repository.
  Mutable refs and the working tree are not exposed. The repository is selected
  once at server startup, so start a separate server process for another repo
  or revision.
- Query runtime is limited to 30 seconds, Markitect output to 2 MiB, and each
  input JSON-RPC line to 1 MiB. The server does not support pagination or large
  context results.
- MCP tool descriptions and annotations are not an authorization boundary.
  The user should review the exposed tools and choose when the client may call
  them.
- Unknown nonempty protocol versions receive the supported `2025-11-25`
  fallback; the client decides whether to continue. Missing or malformed
  initialization versions are rejected.

## Smoke test

From the repository root:

```sh
go test ./experiments/mcp-pilot
```

The tests drive newline-delimited MCP initialization, tool discovery, valid and
invalid tool calls, and lifecycle ordering. They check that stdout contains
only protocol responses, unknown tools and extra arguments are rejected, a
valid call invokes only the configured executable with the pinned repository
and SHA, and inherited Git environment overrides are removed.

On 2026-10-01, a local end-to-end smoke test built both executables, committed
the `examples/minimal` fixture into a temporary Git repository, and compared
`find(query=rollback)` through this MCP server with the CLI's direct fixed-SHA
output. The YAML matched exactly. This establishes protocol plumbing and
result parity for one query; it does not establish client compatibility or an
authoring benefit. The [client evaluation](evaluation.md) records two completed Codex CLI/MCP
query tasks and six byte-identical comparisons. Claude Code connected but its
model tasks were blocked by expired authentication; the user elected to record
that blockage. A second-client task comparison and repeated authoring benefit
remain open.

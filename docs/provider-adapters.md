# Provider adapters

This source increment implements read-only Codex Agent and Check profile metadata preparation. Product session execution, executable checks and workspaces are later stages in the [delivery plan](implementation-plan.md). There is no active legacy provider runtime or MCP server. The target runtime runs every Agent session and check in its own Linux container, with Codex App Server and Claude Code adapters ([DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers), [rebuild plan](design/canonical-model/manager-apply-rebuild-plan.md#container-runtime)).

## Connect the outer client

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. To request metadata/settings, supply `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH`. Host input selection is explicit; the product does not search local accounts or infer a provider configuration.

## Role profiles

The explicit Host document contains closed, typed Agent and Check profile sections consumed by the actual adapters. Project execution declarations select profile names. The Codex section uses `profileVersion: 2`, an explicit engine executable and immutable image ID. Agent preparation binds the actual whole `/opt/markitect/runtime.env` bytes, recorded Codex and agentd versions, and build-recorded Codex schema digest. That digest is an assertion, not an observed complete schema inventory. Instructions select only captured inputs. Check preparation retains its exact host-executable bytes, literal argument/environment policy and finite limits. Effective settings retain real source provenance and product defaults.

See the implemented [Codex metadata adapter](../internal/adapters/codexappserver/README.md) and [Check metadata adapter](../internal/adapters/checks/) for supported versioned fields and their contract tests, and [configuration assignments](design/canonical-model/manager-apply-configuration.md) for the separation of inputs.

## Inner role execution and workspace boundary

Codex metadata preparation may run the configured engine to inspect the image and copy its metadata from a created, never started container. It never starts the image, an Agent Session or a provider, probes a CLI version, exports a schema, creates a workspace or invokes a model. Check metadata preparation remains process-free. A successful metadata result establishes neither runtime nor provider capability. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain actual observed bytes. Neither preview supplies an artifact publication guard.

## Container runtime

The runtime for real sessions is implemented below the session port. Product metadata preparation uses only its image inspection/copy API; provider Session adapters remain unimplemented. `internal/adapters/container` starts each session in a hardened Linux container from the [base image](../packaging/runtime/README.md), and `markitect-agentd` inside it runs the provider CLI and serves the Host tools over MCP. The container adapter's smoke shows both CLIs connecting to the Host tools in that container. The Codex and Claude Code session adapters follow the RUN Session port.

## Bring your own executor

An alternate runtime is not implemented by this metadata boundary. It requires a real consumer port, adapter, bounded readiness evidence and explicit execution authority. The previous [provider/runtime generation](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/provider-adapters.md) remains a historical contract.

## Limits and evidence status

Source tests and closed-format metadata checks establish only their declared inputs and scope. No source or metadata pass demonstrates a real session, completed work, cost measurement, operating-system sandbox, provider performance or acceptance.

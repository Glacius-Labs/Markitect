# Provider adapters

This source increment implements read-only Agent and Check profile metadata preparation. Actual sessions, provider protocols, process execution, executable checks and workspaces are later stages in the [delivery plan](implementation-plan.md). There is no active legacy provider runtime or MCP server.

## Connect the outer client

Build `cmd/markitect` and use its CLI from the coding environment. `check` and `context` are structural by default. To request metadata/settings, supply `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH`. Host input selection is explicit; the product does not search local accounts or infer a provider configuration.

## Role profiles

The explicit Host document contains closed, typed Agent and Check profile sections consumed by the actual adapters. Project execution declarations select profile names. Agent preparation binds exact executable and protocol-schema bytes, supported schema metadata, selected model/effort/settings and instruction policy. Check preparation binds exact executable bytes, literal argument/environment policy and finite limits. Effective settings retain real source provenance and product defaults.

See the implemented [Codex metadata adapter](../internal/adapters/codexappserver/) and [Check metadata adapter](../internal/adapters/checks/) for supported versioned fields and their contract tests, and [configuration assignments](design/canonical-model/manager-apply-configuration.md) for the separation of inputs.

## Inner role execution and workspace boundary

Metadata preparation never starts a process, probes a version, exports a schema, creates a workspace or invokes a provider. A successful metadata result is not runtime support evidence. Structural output never reports metadata as implicitly passed.

Fixed previews refuse unbound present-day external profile or dependency inputs, even for HEAD. Working previews retain actual observed bytes. Neither preview supplies an artifact publication guard.

## Bring your own executor

An alternate runtime is not implemented by this metadata boundary. It requires a real consumer port, adapter, bounded readiness evidence and explicit execution authority. The previous [provider/runtime generation](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/provider-adapters.md) remains a historical contract.

## Limits and evidence status

Source tests and closed-format metadata checks establish only their declared inputs and scope. No source or metadata pass demonstrates a real session, completed work, cost measurement, operating-system sandbox, provider performance or acceptance.

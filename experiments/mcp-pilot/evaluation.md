# CLI and MCP authoring pilot (2026-10-01)

## Decision

Keep MCP experimental. Codex completed the same three read-only queries through
both interfaces on two public synthetic fixtures. CLI and MCP returned identical
YAML in all six direct comparisons. This establishes a usable local interface
for this client, but does not establish a sustained authoring advantage. Claude
Code connected to the server; its expired OAuth session prevented model tasks.
The user elected to record Claude as blocked rather than repair authentication.

The remaining product criterion is a completed second-client comparison and
repeated tasks that demonstrate useful friction reduction. The adapter is not
included in published Markitect binaries. Neither interface establishes human
acceptance or semantic correctness of an adopting project's content.

## Inputs and controls

[The plan](evaluation-plan.yaml) was recorded before the task runs;
[observed counters](evaluation-results.yaml) are stored as YAML. The published,
attestation-verified Windows amd64 CLI was Markitect v0.5.0, SHA-256
`6f5b98dad2ea6003bc0368800691af70084a44d2c225c26fe12d120bcda27178`.
The measured MCP server was built from commit
`800b5e94df907123cc6387fcdcd98476c3f86931`, SHA-256
`0ab34ee51ca9b7c7c1e01272aa0430c90852aeb8e44fba88967b59fcfb2a0f27`.
The original server candidate was `c801dc84fa2da2112910233115fa13f1f01b8d0d`.

| Fixture | Isolated commit | Tree |
|---|---|---|
| `examples/documentation` | `f88e385eacc46c17662c402ca4bcdfd4d39931fd` | `650375629def122c67b82b39dbfbcdc5fa90331f` |
| `examples/package-consumer` | `031a14bb18fe8ff07d730142ef2993ae8d41476f` | `0b7f4cdea68154260e17a5b65d7fe441a3030244` |

Each cell used a fresh noninteractive session and the same fixture commit and
CLI. Actors could use only Markitect queries, with no direct file reads, writes,
web, or subagents. An independent reviewer checked final answers against the
returned evidence and plan. Codex CLI was `0.130.0`; Claude Code was `2.1.233`.
Each client used its default model; Codex did not report the actual model in the
captured event stream, and Claude reported `claude-opus-5[1m]` before failing
authentication. Model effort was not explicitly overridden or reported.

The original plan's counterbalanced order was not achieved after setup repairs
and Claude's blockage. The package dependency in the plan uses an identity
shorthand; the actual canonical qualified identity is
`review-guidance::review/Workflow/review-change`. Context also includes the
Project input in addition to the authored resources listed in the plan.

Codex ran with user configuration ignored because the installed CLI rejected a
current user-config field. Process-local configuration retained a read-only
sandbox and disabled web and subagents. CLI permissions allowed exact literal
query commands for these fixture commits. MCP permissions explicitly approved
only `find`, `explain`, and `context`, with shell tools disabled. No global
configuration or authentication settings were changed. Raw prompts, events,
stderr, and direct query outputs remain in excluded local artifacts under
`.artifacts/mcp-evaluation/2026-10-01/`; they are not published telemetry.

## Completed Codex cells

All four cells made three successful queries and no forbidden extra operations.
Wall time covers the whole client process, including startup and discovery.
Token values are the client's reported counters; cached input is part of input,
and reasoning output is part of output. These are single runs, with different
interface prompts and tool inventories, and cannot establish a speed or savings
trend.

| Fixture/interface | Seconds | Input / cached input | Output / reasoning output | Answer review |
|---|---:|---:|---:|---|
| Documentation CLI | 69.505 | 45145 / 31232 | 1104 / 153 | All requested facts and context entries supported |
| Documentation MCP | 61.221 | 63770 / 47104 | 648 / 78 | All requested facts and context entries supported |
| Package CLI | 140.907 | 43692 / 19456 | 1508 / 238 | Requested facts correct; unsupported extra claim about project bindings |
| Package MCP | 68.497 | 65463 / 55808 | 750 / 98 | Requested package facts correct; context list omitted the Project entry |

The documentation task found `implementation/Text/worker-behavior`, explained
its owning area and direct `uses` dependent, and compiled the startup-review
context containing the declared Go source with `const maxAttempts = 3`.
The package task found the consumer-owned entry, preserved its package-qualified
dependency, version `1.0.0`, archive digest, and package-owned evidence file.
These are query and evidence-reading tasks, not changes to an adopting project.

## Retained failures and boundaries

Initial Codex CLI sessions could not execute queries under the noninteractive
command policy. The original MCP server rejected Codex's offered protocol
`2025-06-18`; negotiation now supports it and `2025-11-25`. Subsequent sessions
connected but the client cancelled calls without explicit per-tool permission.
These setup-invalid sessions are retained separately from completed cells.
They contribute setup friction, and their exit code of zero does not mean the
authoring task succeeded. Claude task completion, query counts, latency, and
task token comparisons remain unavailable because authentication failed.

| Fixture | find bytes | explain bytes | context bytes | Direct CLI/MCP YAML parity |
|---|---:|---:|---:|---|
| Documentation | 632 | 897 | 2545 | All three byte-identical |
| Package consumer | 748 | 943 | 5197 | All three byte-identical |

Direct boundary checks rejected unknown tools and extra `repo`, `revision`, and
`shell` tool arguments. Startup rejected a nested repository path, mutable
`HEAD`, and a nonexistent full commit. Unit tests additionally cover unknown
arguments, lifecycle ordering, literal argv, and Git environment
sanitization. Repository and revision remain trusted, fixed startup inputs;
the adapter is not an operating-system sandbox.

MCP avoided shell command construction after configuration, but added protocol
and tool-permission setup. Its text blocks preserved the CLI's evidence; they
did not prevent an incomplete answer. A second authenticated client and at
least three comparable repetitions per variant are needed before describing
an authoring trend, following [Measurement](../../docs/measurement.md).

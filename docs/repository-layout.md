# Repository layout

A Markitect project keeps its canonical control plane separate from application code. The project selector and policy live in `.markitect/project.yaml`; the canonical model and runtime are selected from that manifest. Managers describe responsibility slices and artifact ownership, while application files remain ordinary project files.

## Recommended target-project layout

```text
.markitect/
  project.yaml                 selected model, policy, checks and coverage
  model/                       Manager and owned concept/rule/workflow files
  runtime.yaml                 local execution roles and finite budgets
  workflows/model-first.md    installed shared workflow
  state/                       typed Work Item explorations and decisions
  runs/                        plans, candidates, receipts and verification
.agents/skills/                optional Codex guidance installed by onboarding
.claude/skills/                optional Claude Code guidance installed by onboarding
AGENTS.md / CLAUDE.md          repository-owned instructions with managed blocks
src/                           project-selected application artifacts
tests/                        project-selected tests
docs/markitect/project.md     generated readable view of accepted model
```

These application paths are examples, not language conventions. The model tree follows conceptual ownership rather than mirroring source directories. Give each ordinary artifact one accountable Manager; express shared responsibility and explicit dependencies in the model. Markitect does not infer behavior, dependencies, or ownership from path names or Markdown links.

The generated `docs/markitect/project.md` is a readable view of the canonical model, not an independent source of intent. Keep the model in YAML, review generated views after changes, and commit accepted model changes under the adopting repository's policy before implementation readiness.

## Ownership and boundaries

| Area | Owns |
|---|---|
| `.markitect/project.yaml` and selected model paths | Project identity, policy, Managers, concepts, rules, artifact responsibilities, and checks |
| `.markitect/runtime.yaml` | Explicit local provider/runtime selection, roles, helper policy, and finite budgets |
| `.markitect/state/` and `.markitect/runs/` | Durable workflow records, proposals, plans, candidates, receipts and verification evidence |
| `AGENTS.md`, `CLAUDE.md`, `.agents/skills/`, `.claude/skills/` | Repository-local outer-agent guidance and discoverable skills; onboarding preserves custom material outside Markitect-managed blocks |
| Application paths and `tests/` | Adopting repository implementation, tests, and local acceptance policy |
| `docs/` | Human-owned docs plus the explicitly configured generated model view |

The MCP server is selected for one explicit project root at startup; tool calls cannot redirect its authority. Local execution uses the caller's permissions. Native workers receive fresh owned candidate workspaces and bounded model/artifact context; only guarded Apply updates the adopting checkout. Candidate workspaces are not an operating-system security boundary.

## Source repository layout

Markitect source currently has its Go module at the repository root, executable entrypoints under `cmd/`, and production packages under `internal/`. P08's `src/` relocation is a planned migration and has not happened in this checkout. Do not copy anticipated `src/...` paths into build commands or contribution instructions until the move and clean-checkout gates are complete. This product-source layout is separate from the target-project example above.

Historical Project/Domain and canonical Projection inputs remain versioned compatibility surfaces. The currently published v0.14.1 pin retains its own paths and behavior; no in-place migration or automatic removal is implied by this recommended target layout. See the [compatibility reference](usage.md#legacy-projectdomain-cli-compatibility) and [distribution guide](../integration/README.md).

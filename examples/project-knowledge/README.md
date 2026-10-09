# Project knowledge graph example

This small model shows how Markitect's read-only knowledge graph presents canonical Manager, Statement, Artifact, Check, Decision, and IdentityChange declarations. The graph is derived from the selected YAML files; those files remain the only canonical model. It contains no provider, run, exploration, session, or briefing records.

The example includes two retained Decisions, with the newer one explicitly superseding the older one. It also declares a rename from a historical Statement identity that is intentionally absent from the current model. That declaration records the model author's claim; it does not prove that the previous Statement existed or that the old and current concepts are equivalent.

## Query the graph

Markitect requires `--repo` to name the Git worktree root. This fixture lives inside the Markitect source repository, so first copy its contents into an independent checkout (for example `C:\src\project-knowledge-example`) and initialize Git there. Use your current Markitect source build; the published v0.14.1 binary does not include the current model-first `project` command surface. The fixture is public and contains no private Manager content, so the whole-project view is appropriate for these examples:

```powershell
New-Item -ItemType Directory -Path C:\src\project-knowledge-example -Force | Out-Null
Get-ChildItem examples/project-knowledge -Force | Copy-Item -Destination C:\src\project-knowledge-example -Recurse
git -C C:\src\project-knowledge-example init
git -C C:\src\project-knowledge-example checkout -b codex/project-knowledge-example
markitect project check --repo C:\src\project-knowledge-example
markitect project document --repo C:\src\project-knowledge-example --write
markitect project knowledge --repo C:\src\project-knowledge-example --knowledge-scope project --knowledge-action graph
markitect project knowledge --repo C:\src\project-knowledge-example --knowledge-scope project --knowledge-action coverage
```

The CLI response wraps graph data under `graph`, so copy IDs from `graph.nodes[].id` exactly. For example, these are the Core tuple IDs declared by this fixture:

```powershell
markitect project knowledge --repo C:\src\project-knowledge-example --knowledge-scope project --knowledge-action explain --node-id '["project.markitect.example.org/v1alpha1","Statement","","project-goal"]'
markitect project knowledge --repo C:\src\project-knowledge-example --knowledge-scope project --knowledge-action history --node-id '["project.markitect.example.org/v1alpha1","Statement","","project-goal"]'
markitect project knowledge --repo C:\src\project-knowledge-example --knowledge-scope project --knowledge-action trace --node-id '["project.markitect.example.org/v1alpha1","Statement","","project-goal"]' --max-depth 5 --max-steps 500 --max-results 50
```

The whole-project scope is explicit in every command. For ordinary work, use a Manager-scoped query when its visibility is sufficient. See [Project knowledge guidance](../../docs/project-knowledge.md) for scope, record opt-ins, MCP, coverage states, and evidence limits.

## Model limitations

The expected artifact is a one-line explanatory fixture, not product implementation. Its declared Python check only prints a message; it does not exercise the knowledge service or establish semantic correctness. These declarations demonstrate graph edges and provenance only. They do not show that a provider ran, that a check was executed, or that a person accepted the model.

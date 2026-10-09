# Repository layout example

This synthetic project uses `.markitect/areas/` for typed YAML, grouped by Area and review responsibility. The YAML `kind` remains authoritative despite the optional kind suffix in each file name. Its [Project](markitect.yaml) configures an engineering Area that imports the shared and documentation Areas; the engineering Workflow applies the Rule and declares both a shared Text dependency and the ordinary [change procedure](docs/engineering/change-procedure.md) as an exact input. A separate `documentation` Area owns `docs/`, and engineering imports it for file-input access. This does not make Markdown typed resources.

[Documentation routers](docs/README.md) navigate human-owned pages without introducing graph edges. This Project selects only Claude; it creates no generic Markdown views. The declared rule adapter generates `.claude/rules/change-review.md` at its native path and links directly to canonical YAML. Other examples explicitly select `markdown` for views under `docs/markitect/`. The fixture's `.gitattributes` marks only generated outputs. No root agent instructions or distribution files are generated.

From the Markitect source repository root:

```powershell
go run ./src/cmd/markitect check --repo examples/repository-layout
go run ./src/cmd/markitect context --repo examples/repository-layout --namespace engineering --kind Workflow --name review-change
go run ./src/cmd/markitect render --repo examples/repository-layout
```

For local edits, run `format --write` and `render --write` on a feature branch, then `check`. The fixture has no declared runtime checks; structural success does not produce complete `verify` evidence. To query a fixed revision, copy the fixture into its own Git repository and commit it; fixed snapshots expect `markitect.yaml` at the selected Git root.

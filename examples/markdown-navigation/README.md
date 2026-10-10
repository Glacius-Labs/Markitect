# Markdown navigation example

This fixture demonstrates how Markitect relocates source-relative links when it renders optional Markdown views. The Skill and Workflow YAML files live under `docs/general/skills/` and `docs/general/workflows/`; ordinary tool guidance and an SVG image remain in their human-owned locations. The Project explicitly selects both `markdown` and `claude` outputs.

The Skill links to the Workflow through both its legacy companion alias and its canonical YAML source path. In a Markdown view, both links resolve to the selected central Workflow view. Ordinary Markdown and image links are rebased to their original files, preserving query strings and fragments. The Claude Skill adapter keeps its declared Workflow dependency pointed at canonical YAML. Prose links do not add graph dependencies.

The fixture also includes an inline code span and a fenced Markdown example containing fake links. The regression test checks that those examples remain byte-for-byte recognizable and that generated links resolve to files in the exact loaded snapshot or generated output set. A second test adds an unmarked human-owned Markdown file at the old companion alias path and confirms it takes precedence over the typed view.

From the Markitect source repository root:

```powershell
go run ./src/cmd/markitect check --repo examples/markdown-navigation
go run ./src/cmd/markitect render --repo examples/markdown-navigation --write
go test ./src/harness/examples -run MarkdownNavigation -count=1
```

Generated companion views live under `docs/markitect/general/`. The fixture has no dependencies inferred from its prose links.

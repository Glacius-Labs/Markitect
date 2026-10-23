# Claude projection independence: parallel-wave negative controls

This bounded source-tree slice checks that the built-in Claude projection takes guidance from canonical Markitect resources and explicit Project selection, not from generated Codex or Markdown views. It starts from the assigned baseline `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada` in an isolated worktree; its tests are not release or runtime evidence.

`render.GenerateWithOwners` consumes the validated resource graph and the explicit source-file map. For Claude Agents, the selected `claude` target gates output; configured inline text comes from the canonical Agent resource. A Claude Rule aggregate is produced only from the Project's explicit `ruleAdapters` refs and links to those canonical Rule YAML files. `GenerateWithOwners` reports resource-level owners for both outputs.

The added negative controls are:

- `TestClaudeProjectionDoesNotUseGeneratedSiblingViewsAsAuthority` supplies adversarial Codex, Claude-rule and Markdown-view bytes as unlinked files in the renderer input. They do not change generated Claude Agent/Rule content or owners. The test also checks that a Claude-only target emits neither Codex nor Markdown output, and that the Claude Rule link and owner come from the explicit canonical Rule mapping.
- `TestClaudeProjectionRejectsAmbiguousResourceOwnedOutputPathDeterministically` gives two namespace-distinct Skills the same Claude output name. Rendering fails with the duplicate generated path on repeated runs instead of silently assigning the path to whichever resource happened to be visited last.

Validation on the workstream branch:

```powershell
go test ./internal/render -run 'TestClaudeProjection' -count=1
git diff --check
```

The focused Go test passed. `git diff --check` is recorded separately in the completion report for this candidate SHA.

## Evidence limits

The adversarial sibling files are deliberately unlinked. Markitect may use its explicit file snapshot to resolve links that canonical prose itself names; this test does not claim that every generated-looking path is ignored when a canonical source explicitly refers to it. The tests exercise local rendering only: they do not test filesystem reconciliation, provider runtime behavior, whether Claude follows the generated instructions, or the truth of the authored guidance. Existing tests separately cover strict inventory and retired-file checks, explicit rule mapping failures, and output ownership.

This is a test-only independence slice. It changes no renderer implementation, shared output-path contract, app inventory validation, Domain/Core semantics, CLI, schemas, or release artifacts. Any defect requiring those shared paths needs a minimized reproducer and coordinator review before implementation.

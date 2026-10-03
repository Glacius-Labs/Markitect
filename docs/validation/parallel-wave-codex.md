# Parallel wave: Codex projection independence

## Scope and source

This bounded check starts from preparation baseline `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada` and covers the built-in Codex projection selected by `Project.spec.targets: [codex]`. It changes only `internal/render/parallel_wave_codex_test.go` and this evidence note. No renderer, path, ownership, inventory, CLI, schema, or Core code changed.

The fixture contains canonical Agent and Skill resources, both Codex and Claude settings on the Agent, and captured Claude or unregistered provider-looking files. The test checks that only `.agents/skills/review/SKILL.md` and `.codex/agents/reviewer.toml` are emitted; each has one canonical resource owner; Codex settings and canonical source links appear; and changing the captured Claude/unregistered bytes cannot change Codex bytes or ownership. A repeated render check compares output bytes and owners across 16 runs.

For the fixed fixture, the in-memory canonical text values and generated output bytes have these SHA-256 digests:

| Input/output | SHA-256 |
|---|---|
| Agent text (`docs/team/agents/reviewer.yaml`) | `e07fcd75a2059e4c8f76235f25156302cfa3fff27d3c6d5846a48643d98ad991` |
| Skill text (`docs/team/skills/review.yaml`) | `e65e9113f401fcffa1f9338c65be3963a50668d51ef3fc2b7b333ad188dd5110` |
| `.agents/skills/review/SKILL.md` | `6a87ed770b1e1f28aa99ab3fb8be3d0746ac486a695834404f881c6d85ece5ee` |
| `.codex/agents/reviewer.toml` | `5e07e62851d65a34fea9c1de8ae77824d1fccb8f1cbe73a894d0c4b66346255e` |

The input digests cover only the in-memory `spec.text` strings, not serialized YAML or a repository snapshot. The fixture has no strict-inventory result; repository inventory is outside this render-package test.

## Evidence limits

This establishes deterministic local projection behavior for the fixture and renderer call. It does not establish Codex runtime behavior, tool permissions, execution safety, human acceptance, or correctness of canonical prose. It does not exercise filesystem reconciliation, strict inventory across an adopting repository, collision handling, provider target aliases, or shared dispatch. Those remain shared renderer/app seams and must be tested by their owners when changed. Captured Claude-looking files are treated as opaque inputs to this test; their presence is not evidence that they are canonical or trustworthy.

## Validation

Run from the feature worktree:

```powershell
go test ./internal/render -run '^TestParallelWaveCodexProjection' -count=1
```

The full package test is also run for this candidate:

```powershell
go test ./internal/render
```

The focused test logs the in-memory text and output digests listed above; it does not pin those hashes as golden values. Its claims are limited to the fixed in-memory inputs described above.

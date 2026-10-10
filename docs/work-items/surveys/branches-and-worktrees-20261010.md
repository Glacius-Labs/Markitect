# Survey: branches, worktrees and earlier work

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for OPS-02, OPS-03 and every package with earlier work. Not a decision.

## Summary

- There are 238 local branches. Only 10 hold work that is not already in main:
  - 81 are merged;
  - 20 were squash-merged;
  - 129 are superseded, because every commit is patch- or subject-equivalent to main or the branch is contained in another branch;
  - 6 are unmerged and were active on 10 October;
  - 4 are unmerged and stale.
- **Second remote:** `research-backup` (`TheGlacius/markitect-research-backup`) holds the large research history (`codex/government-scientist`) and now the playground branch.
- **Naming:**
  - `codex/<slug>` dominates (225 branches); `claude/<slug>` has 3.
  - Date suffixes, version spellings, campaign prefixes and retry suffixes vary.
  - Worktree folder names often differ from branch names.
  - Worktrees sit under three roots: `~/.codex/worktrees`, `~/.claude/worktrees` and the main checkout.

  The roadmap's `wp/<id>-<slug>` convention replaces this for new work.

## Where earlier work lives (after OPS-02, 10 October 2026)

Snapshots under `archive/wip/*` capture uncommitted and untracked files on top of the branch head. They were made without changing the worktrees.

| Topic | Work | Location |
|---|---|---|
| Runtime (RUN-01) | Exchangeable executor increment 1 | Pull request #92, branch `claude/byo-executor-role-config` (origin) |
| Runtime, App Server (MCP-02, BUG-01) | Uncommitted fixes in `codexappserver/session.go` and `projectrun/full_verify.go`; the branch commits themselves are in main | `archive/wip/readiness-candidate-fix-20261010` (origin) |
| Playground (PLAY) | Container harness (committed `f98313f1`) on top of the old research history | `claude/case-playground-v2` (research-backup) |
| Playground (PLAY) | Further harness work in progress: evaluate/reviewers/compare, Claude agent, case `readinglog2` with reference implementation | `archive/wip/case-playground-v2-20261010` (research-backup) |
| Knowledge graph (KG) | Knowledge index, decisions, read-only tools, example (old `internal/` layout) | `codex/knowledge-graph` (origin) |
| Knowledge graph, earlier plans (KG, IDEA-01) | Untracked drafts: independent assessment, knowledge-graph assessment and refinement plan, refinement plan | `archive/wip/government-assessment-untracked-docs-20261010` (origin) |
| Government (GOV) | v1 concept and evaluation docs (committed), plus model changes in progress that do not compile | `codex/government-implementation-20261010` and `archive/wip/government-implementation-20261010` (origin) |
| Government, older code (GOV) | G1–G5 implementation (old layout) | `codex/government-p1` (contains `codex/government-worker`) |
| Government, coordination record | Docs-only coordination history | `codex/government-assessment` (origin) |
| Learning (LEARN-01) | Working-history inventory and narrative | `archive/codex/markitect-historian-working-history` (origin) |
| Ideas, marketing (IDEA-01) | Marketing research log | `archive/wip/marketing-notes-20261010` (origin) |
| Concepts | Concepts chat notes, already imported into `docs/concepts` | `archive/wip/concepts-historian-notes-20261010` (origin) |
| Other | Astra assessment snapshot; vision-method alignment (likely obsolete) | `archive/codex/astra-assessment-snapshot`, `archive/codex/vision-method-alignment` (origin) |
| Stale experiment | c11 module-replacement runner change. The 204 MB of run output were not archived. | `archive/wip/c11-module-replacement-20261010` (origin) |

## Archive list for OPS-03

Safe to archive, about 230 branches:
- the merged and squash-merged branches;
- the superseded branches, among them `luna-*`, `proof-*`, `c*-proof`, `wave-*`, `release-v*`, `product-*-20261009`, `projectrun-*`, `ideas-notes-20261009`, `government-evaluation-20261009`, `project-world-planning`, `mcp-pilot`, `winget-path-fix` and `cli-installation*`.

Keep until their work is integrated or explicitly archived:
- `codex/government-assessment`
- `claude/case-playground-v2`
- `codex/government-implementation-20261010`
- `codex/knowledge-graph`
- `claude/byo-executor-role-config`
- `codex/markitect-historian-working-history`
- `codex/government-p1`
- `codex/autonomous-ab-gauntlet` (draft PR #78)
- `codex/astra-assessment-snapshot`
- `codex/vision-method-alignment`

Five empty leftover folders exist in `~/.codex/worktrees`: `5239`, `6531`, `689d`, `7567` and `ed4d`.

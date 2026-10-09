# Sources and limits

## Current product sources at base

Started clean from `origin/main` SHA `5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e`. Relevant base-file SHA-256 values:

- `docs/vision.md`: `bb1361a545fd56e4cf923edf0985ca805351f7f61c4b6181da73072aa77443ed`
- `docs/architecture.md`: `e3bb8a03cb43ef27a77796ddb3974a1d1fe43fcee5f819cde18132c89a668aa4`
- `docs/operating-methodology.md`: `67d5c3a5e1b34760fa3d90900b09958fb854016807010cb6050ed9c01b987560`
- `docs/implementation-plan.md`: `39850f74bbb785e6c077ad3c34e2c301a5a16ac50d1510da450e7c8aa5c19708`
- `docs/measurement.md`: `8979cb5fd328aa209a78bd3e51810c2f079b7af332e3b916cdc0bd90558e2471`
- `docs/README.md`: `267f59d4d7ab7344660dbeffbd4a9f32acc2552c7db74bb0d2b5dd6e475c9725`

Vision: define desired project world once, reconcile representations without losing intent; quality/consistency primary, possible efficiency benefits unproven. Docs call Classic active on main and Government a separate experiment. Markdown is a projection/context interface and part of the name origin, not the canonical semantic model.

## Chat coverage

Archived Codex index: 243 entries / 5 pages, fully paginated; 24 relevant archived threads selected. Active index: 50 entries, no continuation cursor; 9 Markitect entries plus 3 explicitly known recipients selected. Archived ChatGPT index: one Markitect chat. In all, 38 selected histories were read through every returned page; this does not establish global completeness.

Gaps: active list is capped at 50 with no cursor; some message IDs lack accessible user text; Integrator is a sparse in-progress history with no final answer; one ChatGPT attachment was not included; some Coordinator/Reviewer attachments were not opened. The other 219 archived records were not transcript-read. Details and returned IDs: [chat inventory](chat-inventory.json), [checkpoint](checkpoint.json).

## Git and GitHub snapshots

Adjacent inventories captured `2026-10-09T19:51:44.4444147Z`: `git-refs.tsv` (no fetch), `git-worktrees.tsv`, `github-prs.json`, `github-issues.json`, `github-releases.txt`. Counts: 232 local heads, 94 stored remote refs, 19 tags, 16 worktrees. Repository state changed during capture; these are not atomic.

At that snapshot, integration candidate `3c4ab67cca313c1beabe82aafed34a5a5b513b0e` was 156 commits ahead and 0 behind merge base; diff scope 869 files (+77,342/−4,808). Its readiness note says full suite at `dae4b4a5` failed, narrow corrections followed, fresh integrated suite pending; A01/A02/A03 not started, no jobs/role starts, release, main merge or human acceptance. Historian did not test.

GitHub returned 86 PRs; #89 open draft at `1495e1be7b0046711531fa42c8407fba67b8e814`, #78 open draft for a paired gauntlet. Latest listed release v0.14.1 dated 2026-10-07, tag target `784d3c3c61443b291ac7db9727c7c5856e253d66`. Recheck before relying on status.

Primary, Concepts, Marketing, Ideas, integration and proof worktrees were simultaneously active; exact paths, SHAs and dirty files are in `git-worktrees.tsv`. Drafts are not adopted product decisions.

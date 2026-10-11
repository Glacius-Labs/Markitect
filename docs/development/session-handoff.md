# Session continuation

Dated handoff, 11 October 2026. Main basis: `5f9b6085a17d0907c826f9817f4255a0d99a753c`, fetched from `origin/main`. This transfers the interrupted Claude work to fresh Codex role sessions. The [backlog](../work-items/backlog.yaml) owns package status; the [roadmap](../implementation-plan.md) owns coordination. Refresh the PR heads, checks, worktrees and zone assignments before acting on this snapshot.

The owner requested an investigation, updates and a continuation plan. This handoff does not dispatch new chats, merge the six existing implementation PRs, authorize a provider run or study, publish a release, or approve deletion of old worktrees. It preserves existing product decisions and their authority boundaries.

## Evidence and source boundary

The audit read 12 top-level Claude session logs and their relevant continuation records under the local directory `C:\Users\Consiliari\.claude\projects\C--Users-Consiliari-Glacius-Labs-Markitect`. Several owner sessions and analysis helpers stopped at the weekly quota around 02:59–03:07 UTC on 11 October. Their later filesystem modification times do not establish later engineering progress. Session messages are evidence of instructions and reported intent; commits, working copies and current hosted results establish implementation state.

The Claude logs' common `gitBranch: codex/government-assessment` describes where the chats started. Actual package work happened in separate worktrees; do not use that session metadata as the package's source identity.

| State | Path / branch | Full commit | Local changes at audit |
|---|---|---|---|
| Current implementation basis | `origin/main` | `5f9b6085a17d0907c826f9817f4255a0d99a753c` | A Git ref; source scope only |
| Original open checkout | `C:\Users\Consiliari\Glacius Labs\Markitect`, `codex/government-assessment` | `6623cd0b70aff31f00e4eb3ab44868ecf76de370` | Untracked `docs/design/concepts/`, `independent-assessment-20261010.md`, `knowledge-graph-assessment-20261010.md`, `knowledge-graph-refinement-plan-20261010.md`, `refinement-plan-20261010.md` |
| This documentation transfer | `C:\Users\Consiliari\.codex\worktrees\session-handoff\Markitect`, `docs/session-handoff` | Started from the main basis above; candidate identity is the handoff PR head | Only the documented coordination/status/prose changes |

The original checkout has 409 branch-only and 1,145 main-only commits against this main basis. Its drafts are preserved inputs, not the current implementation. Continue new work from current main; do not update that checkout in place or copy its source tree over main.

Hosted results for the main basis:

- [CI run 38106745252](https://github.com/Glacius-Labs/Markitect/actions/runs/38106745252) and [CI run 38127020298](https://github.com/Glacius-Labs/Markitect/actions/runs/38127020298): completed successfully on the exact main SHA.
- [Playground smoke 38127880194](https://github.com/Glacius-Labs/Markitect/actions/runs/38127880194): completed successfully on that SHA, provider-free.
- The latest published release is still [v0.14.1](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.14.1). These source results do not update its installed binaries, establish a real-provider journey or validate the method's benefit claims.

## What is integrated

The backlog already records most completed packages. The following joins explain why old session prompts and branch names must not be resumed literally:

| Earlier work | Current implementation evidence | Continuation |
|---|---|---|
| Process/exchange executor PR #92 | Superseded and merged as [#104](https://github.com/Glacius-Labs/Markitect/pull/104), RUN-01 | RUN-02 asynchronous boundary and RUN-05 review follow-ups remain separate packages |
| Hermetic test infrastructure | TEST-01 slices [#99](https://github.com/Glacius-Labs/Markitect/pull/99) and [#157](https://github.com/Glacius-Labs/Markitect/pull/157), source-test fix [#160](https://github.com/Glacius-Labs/Markitect/pull/160) | Do not reopen the already integrated testkit conversion |
| Top-level verbs and per-role configuration | CLI-02 [#161](https://github.com/Glacius-Labs/Markitect/pull/161), AGENT-02 [#181](https://github.com/Glacius-Labs/Markitect/pull/181) | MCP-01 is in review; CLI-03, CLI-04 and TEST-02 share the interface zone |
| Reduced knowledge core and explanations | KG-01 [#119](https://github.com/Glacius-Labs/Markitect/pull/119), KG-02 core #165–168 and surface [#184](https://github.com/Glacius-Labs/Markitect/pull/184) | Existing visibility repairs are BUG-01; KG-04 needs owner choices |
| Playground and shared smoke | PLAY-01 [#103](https://github.com/Glacius-Labs/Markitect/pull/103), CI-04 provider-free [#140](https://github.com/Glacius-Labs/Markitect/pull/140), CI-03 [#183](https://github.com/Glacius-Labs/Markitect/pull/183), executable verification replay [#190](https://github.com/Glacius-Labs/Markitect/pull/190) | Complete PLAY-05 inventory, then PLAY-06 configuration; no new paid run is inferred |
| Linux CI and race gate | CI-02 carried to main by [#113](https://github.com/Glacius-Labs/Markitect/pull/113); CI-07 [#199](https://github.com/Glacius-Labs/Markitect/pull/199) | Required Linux quality and race checks; Windows is non-blocking for ordinary merges |
| Timing/flake work | TEST-05 [#158](https://github.com/Glacius-Labs/Markitect/pull/158) enlarged non-timing limits; [#159](https://github.com/Glacius-Labs/Markitect/pull/159) added the flake report | Timer injection/fakes remain incomplete; reserve the runtime zone before writing |
| Repository identity optimization | ARCH-11 [#180](https://github.com/Glacius-Labs/Markitect/pull/180) | Already on main |
| Self-hosted model and stable view | ARCH-07 [#203](https://github.com/Glacius-Labs/Markitect/pull/203), stable document [#216](https://github.com/Glacius-Labs/Markitect/pull/216) | Only entrypoint follow-up #222 remains; legacy removal belongs to ARCH-09 |

The quota cut off coordination after these changes; it did not roll them back. Source conformance, test results, advisory agent assessments and human acceptance remain distinct.

## Review queue

All six PRs are open with successful current hosted checks at the audit. That is a CI observation, not an independent review approval. None had a recorded GitHub review approval or request for changes. Compile each candidate merged onto the then-current main before merging, and inspect its scope and unresolved acceptance even when GitHub reports a clean merge.

| PR / package | Head branch | Remaining work and ordering |
|---|---|---|
| [#205](https://github.com/Glacius-Labs/Markitect/pull/205), BUG-01 | `fix/briefing-reads-do-not-write` | Review accepted-history read behavior and concurrent writers first. Its main base is older than the audited main. |
| [#221](https://github.com/Glacius-Labs/Markitect/pull/221), BUG-01 | `fix/projectrun-previews-read-only` | Stacked on #205's branch. After #205 reaches main, reconcile with its latest head, retarget to main, validate and review. Never merge only into the already-merged feature branch. |
| [#223](https://github.com/Glacius-Labs/Markitect/pull/223), BUG-01 / kg | `fix/full-verify-child-assessments` | Review private child verdict filtering and the documented residual risk of private identifiers in arbitrary agent prose. Coordinates runtime files with #205/#221. |
| [#218](https://github.com/Glacius-Labs/Markitect/pull/218), MCP-01 | `dev/smaller-mcp-schemas` | Review result schema sharing, parity, size ceilings and current visibility fixes. Its main base is older. Clear this package before another interface writer starts. |
| [#219](https://github.com/Glacius-Labs/Markitect/pull/219), PLAY-05 final slice | `dev/playground-image-pins` | **Pushed head incomplete:** its body says the pinned image inventory is absent, but the local branch already holds the inventory in unpushed commit `b64f4748`. Review those bytes and their build provenance, submit the commit after confirming the old head's required checks have finished, update the PR description, then validate the new head. |
| [#222](https://github.com/Glacius-Labs/Markitect/pull/222), ARCH-07 follow-up | `docs/short-manager-name` | Review the short Manager-ID contributor example and its executable command. This PR touches AGENTS.md; coordinate with docs-entry work. |

The integrator may process independent #218/#222 while the runtime queue is being reviewed. #219 can be completed independently in the playground. Every merge remains sequential and validates composition with current main.

Exact candidate evidence at the audit (quality and race jobs succeeded at each listed head):

| PR | Full head SHA | Hosted CI |
|---|---|---|
| #205 | `919f96bb293d5cc1e5ed56cd705184d91cbb3edb` | [38106886180](https://github.com/Glacius-Labs/Markitect/actions/runs/38106886180) |
| #221 | `a4da3bfc768038e08da306b0f56e13d181474e08` | [38105904484](https://github.com/Glacius-Labs/Markitect/actions/runs/38105904484) |
| #223 | `1d1d98991b8fd37f21069e971bb91dc3b1f0b692` | [38106853150](https://github.com/Glacius-Labs/Markitect/actions/runs/38106853150) |
| #218 | `e6ca3492f3e5c46e7a89a31c6f8a62ff8ac5e666` | [38106204645](https://github.com/Glacius-Labs/Markitect/actions/runs/38106204645) |
| #219 | `3b2fbb354dec6ebc1a0fc460896cdfffb2db1256` | [38106438027](https://github.com/Glacius-Labs/Markitect/actions/runs/38106438027) |
| #222 | `0c2057b079492e79a8190c2088d8918dc79c9fa5` | [38106833440](https://github.com/Glacius-Labs/Markitect/actions/runs/38106833440) |

#222 and #223 branch from the audited main. #205 and #221 share older merge base `d3cacc090721e7844c4fe53f833841486265d55d`; #218 and #219 share older merge base `0bd5dba5dc002e9ac9309dcdbe1864976bc7eff7`. #221's recorded base head is `62bb3ebeea9b75f1a6dbfbe98bfd1f62626596c5`, older than #205's audited head. Refresh those relationships before retargeting or composing candidates.

## Working-copy recovery

The registered scientist worktree is **clean but one commit ahead of its remote** at `b64f4748369e3410a414e8a88bd574fd5d975d74`, branch `dev/playground-image-pins`, path `C:\Users\Consiliari\.claude\worktrees\markitect-scientist`. Its parent is the pushed #219 head `3b2fbb354dec6ebc1a0fc460896cdfffb2db1256`. The unpushed commit adds the 143-line inventory at `experiments/case-playground/container/inventory/codex-0.162.0-claude-2.1.296.txt`. Its commit message binds the inventory to the cold CI build on `3b2fbb35` and image key `playground-image-0033bf5ffc829ac7a8da39c6573aa7b791402bc2c064cd48966995e5390ff984`. Review that claim against the build output; green CI for the parent does not validate the unpushed child. The WSL work location is the same host worktree under `/mnt/c/Users/Consiliari/.claude/worktrees/markitect-scientist/`, not a separate missing checkout. The session reported 320 unit tests passing (six skips) locally; that report is not a rerun by this handoff.

The registered #223 worktree is **clean** at `1d1d98991b8fd37f21069e971bb91dc3b1f0b692`, branch `fix/full-verify-child-assessments`, path `C:\Users\Consiliari\.claude\worktrees\markitect-kg-fa`. Its PR already documents the prose-summary limitation. The session intended another sentence near the #210 guide, but no corresponding dirty file was found here; compare current documentation and the PR before inventing or repeating that edit.

The separate compile/schema-limit prose is already integrated by #202: source commit `4dda1b6d4e25abf2cfec68776d9082dd3b330f43`, merge `2053ad979f0a4fa8ef188519536b775a18cca060`. That record concerns input budgets and the single-Schema assumption; it does not settle the child-summary privacy limitation in #223.

These completed worktrees are clean and their heads are ancestors of the audited main:

| Path below `C:\Users\Consiliari\.claude\worktrees\` | Branch | Full SHA |
|---|---|---|
| `bh-arch11` | `dev/repository-identity-once-per-load` | `6af1b62be41bf2e4eb10083a4cc01388b87abfc7` |
| `dev-exchangeable-executor` | `dev/exchangeable-executor` | `6c450d71a19efcb8516817d6f005e8f7d2d304d6` |
| `fix-source-tests` | `fix/source-tests-on-testkit` | `c350d65ab5c1c9260615b1f3b1ab5c5743412fbf` |
| `markitect-qa-testkit` | `dev/verifiable-project-world` | `62c393a878f2d1bbef9f2370bfb9eefb29e107f3` |

Other dirty worktrees remain preservation inputs: the earlier `case-playground-v2` candidate with broad harness/evaluation WIP, `c11-module-replacement` with a modified runner and untracked run evidence, bughunter verification fixtures, and a trailing-space patch. These are not automatically part of the six PRs. The [dated branch survey](../work-items/surveys/branches-and-worktrees-20261010.md) records archive sources; fetch and compare an archive's exact bytes before relying on it to cover current dirty files. No cleanup was performed.

The audit found 86 registered worktrees and 13 dirty worktrees excluding this documentation candidate. Two repeated detached verification bases are `3e56d98eafd1c8cbfc688301bf19c74eb9230517` (A) and `a98921cc3e1d16d7ff3beee0d9e492458be3a82a` (B). The following preservation inventory supplements the original checkout above. Paths are relative to each named worktree; all verification/patch entries are untracked.

| Worktree | Branch / full SHA or detached base | Local changes |
|---|---|---|
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-model` | detached A | `src/internal/{core,host/projectrun,modules/projectmodel}/bug01_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-modelapp` | detached A | `src/internal/host/{projectbriefing,projectcoverage,projectwork}/bug01_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-source` | detached A | `src/internal/{host,host/projectadoption,host/projectexplore,infrastructure/source}/bug01_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-tooling` | detached A | `bug01_wsl.sh`, `src/internal/tooling/architecture/bug01_repro_test.go`, `src/internal/tooling/codexrunner/test_bug01_repro.py` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-adopt` | detached A | `src/internal/host/projectadoption/bug01_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-run` | detached A | `src/internal/host/projectrun/bug01_run_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-cli` | detached B | `src/internal/host/{projectcli,projectonboarding}/bug01_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-apply` | detached B | `src/internal/host/bug01_guard_repro_windows_test.go`, `src/internal/host/projectrun/bug01_apply_repro_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-verify-ws` | detached B | `src/internal/host/codexappserver/bug01_repro_windows_test.go`, `src/internal/host/projectworkspace/bug01_repro{,_windows}_test.go` |
| `C:\Users\Consiliari\.claude\worktrees\bh-fix-trailing-spaces` | `fix/project-document-trailing-spaces`, `929e00b1e194d267bd084f5fe5e277744816ba00` | `trailing-spaces-fix.patch` |
| `C:\Users\Consiliari\.codex\worktrees\c11-module-replacement\Markitect` | `codex/c11-module-replacement`, `6dd126fdf9e13cbddf0edf82fa4e72971744c792` | Modified `internal/tooling/codexrunner/runner.py`; untracked `runs/c11only/` |
| `C:\Users\Consiliari\.codex\worktrees\case-playground-v2\Markitect` | `claude/case-playground-v2`, `f98313f1c0ae4a964b2cd2bc29aff813e52f6d98` | Modified harness/docs/Dockerfile/tests; untracked evaluation, readinglog2, Claude agent, comparison/reviewer implementations and tests |

Braces in this table abbreviate separate paths; they are not literal filenames. Re-run `git status --short` in the exact worktree before recovery and bind any selected bytes to a new candidate. Old repro files need current-main reproduction before they become new findings. This inventory grants no cleanup authority.

The local-branch audit also compared every branch with its upstream, not only with main. The scientist inventory is the only pending implementation commit on a tracked branch ahead of its upstream. `fix/bootstrap-path-comparison` is ahead of stale tracking at `d3d6f4e55588f966849de9bf0f71fd3a9e4949f8`, but that exact commit is already an ancestor of main; it adds no unfinished work. This handoff's own unpushed documentation corrections are accounted for by its PR.

These five non-archive branches have no configured upstream and commits outside the audited main. A missing upstream does not mean no remote backup exists. Treat them as preserved candidate inputs and compare their accepted scope with current source before porting anything:

| Branch | Full head | Continuation boundary |
|---|---|---|
| `claude/case-playground-v2` | `f98313f1c0ae4a964b2cd2bc29aff813e52f6d98` | Old research history plus dirty harness work; PLAY-01's selected harness is already on main. Do not merge the entire research history. |
| `codex/c11-module-replacement` | `6dd126fdf9e13cbddf0edf82fa4e72971744c792` | Historical proof records and dirty runner/evidence, not a current product package. |
| `codex/government-implementation-20261010` | `9976991611beacb34a7e6b1440aef87a6500db7d` | Docs-only v1 contract; the separate WIP snapshot contains an incomplete model. GOV-01 remains blocked. |
| `codex/projectrun-known-check-repair` | `97df2ca457687999823d57a8237a3f556e6c2e55` | Two earlier-layout repair/test commits; reproduce the need on current source before selecting a fix. |
| `docs/product-model-concepts` | `8f5fa92aed4e8f13d1abbaa2f458a0542b18db93` | Nine proposal/documentation commits; preserve provenance and compare with accepted register entries, rather than adopting their model by assumption. |

Ten additional no-upstream archive branches retain unique historical/snapshot commits: `archive/codex/{astra-assessment-snapshot,markitect-historian-working-history,vision-method-alignment}` and `archive/wip/{c11-module-replacement-20261010,case-playground-v2-20261010,concepts-historian-notes-20261010,government-assessment-untracked-docs-20261010,government-implementation-20261010,marketing-notes-20261010,readiness-candidate-fix-20261010}`. They are recovery references, not a new queue. The branch survey routes to their subjects and backup locations.

## Role starts

Start the **integrator** first. Then interface, scientist, kg and bughunter can resume their narrowly bounded PR work. QA and cleaner can inspect without writing shared files. The integrator explicitly assigns the runtime and interface zones before implementation. Six sessions changing code remains the maximum, not a target.

Use this shared prompt, replacing `<role>` and appending that role's initial queue below:

```text
You are the <role> Codex session for Markitect. Read current AGENTS.md,
docs/implementation-plan.md, docs/work-items/backlog.yaml and
docs/development/session-handoff.md. Fetch origin and state the exact checkout,
branch, full SHA and local changes. Resume an existing candidate only after
confirming ownership; use a branch from current origin/main for new packages.
Read the owning Manager's context and the package inputs. Work in dependency
order within your assigned zone, one PR per package or slice, with its ID in
the title. Freeze pushed heads during required Linux CI and preserve local WIP.
Do not stash, force-push reviewed work or delete old evidence. Only the
integrator updates the roadmap, backlog and concept register. Use subagents
for independent workstreams; review and synthesize their results. Existing
provider-free checks do not authorize provider runs or studies. Report exact
source-bound evidence, remaining acceptance and owner decisions.
```

| Role | Initial queue to append |
|---|---|
| integrator | Revalidate the six PRs and main CI; review #205 then retarget #221, and coordinate #223. Process independent #218/#222 after composition checks. Keep #219 incomplete until its local inventory commit is reviewed, pushed and verified at the new head. Update status after each merge and explicitly release zones. Preserve outstanding owner questions below. |
| interface | Resume MCP-01/#218; compare its schemas and parity with current main. After that package closes and the integrator releases the zone, take CLI-03, CLI-04 and TEST-02 serially. Briefing preview changes depend on accepted-history #205/#221. |
| scientist | Resume PLAY-05/#219 from the existing clean worktree at local `b64f4748`, one commit ahead of the pushed head. Review its inventory and build provenance, confirm the old head's CI has ended, then submit the commit and validate the new head. Respect existing live resources. After PLAY-05 closes, take PLAY-06; keep PLAY-07 parked pending protocol and budget. |
| kg | Resume #223 and compare the intended known-limit guide sentence with current documentation. Keep public contracts, private Statements and untrusted prose distinct. KG-04 stays blocked until its repository, independent must-change author and thresholds are chosen. |
| bughunter | Resume #205/#221. Refresh the follow-ups from merged #209, historical-census handling and private-Statement exposure in ancestor briefings; these are reported unfinished findings until reproduced on current main. Coordinate runtime files with kg and QA. |
| qa | Inspect the current CI and flake artifacts, TEST-05's partial completion and the exact remaining timer risks. Do not repeat the merged TEST-01/CI-07 work. Runtime writes start only when assigned; TEST-04, RUN-02, TEST-03 and ARCH-05/06 then follow the roadmap's sequence. Any image-cache proposal waits for PLAY-05 closure. |
| cleaner | Review #222 and finish ARCH-08's interrupted release impact analysis. Account for workflow, bootstrap, shared smoke and legacy-binary benchmark users. Implement only after the interface zone is assigned. ARCH-09 follows ARCH-08; ARCH-05 waits for its runtime turn. |
| runtime | Start after the BUG-01/QA runtime queue is released. Take RUN-05 and RUN-02 in the integrator-agreed sequence, then ARCH-06 after TEST-04. Further provider support, helper-result work and cost accounting remain separate RUN packages. |
| government | No implementation start yet. GOV-01 waits for ARCH-06 and the after-Wave-2 condition; preserve the existing contract and archived incomplete model as inputs. Detailed later ideas remain under DEC-011. |

The integrator must not treat a stopped Claude chat, `active` package, local dashboard entry or a clean historical worktree as a live code writer. Confirm the new role's ownership and reserve files explicitly.

## Claude session index

Local JSONL filename stem identifies each session. Timestamps below are its last useful engineering report in UTC, rather than filesystem modification time. This index is for recovering intent; the source/PR sections above supersede reported completion claims.

| Session UUID | Role / last report | Handoff concern |
|---|---|---|
| `b1362bde-6f1b-4da2-8153-4e84678f6fca` | integrator, 11 Oct 03:05 | Monitor expired at quota; #218 reported green. Reconstruct live queues instead of trusting the old dashboard. |
| `2da50b5c-e23d-4a05-8a1a-7fe3b3d55b41` | qa, 11 Oct 02:57 | CI/test slices reported merged; remaining runtime tests and image-cache dependency. |
| `58879dc0-5d6e-4d2c-a9ac-6700363c13af` | interface, 11 Oct 02:57 | #218 pushed; briefing read-only caller waited for #205. |
| `e9e2616b-6436-46d1-b3c5-543c8406fd0c` | scientist, 11 Oct 03:07 | #219 parent green; inventory committed locally as `b64f4748` and not yet pushed. |
| `67dbb1da-b88c-4112-b0b5-2babb85344a7` | kg, 11 Oct 02:59 | #223 plus intended known-limit prose; quota stop. |
| `387bb0ef-4492-4ed4-bc3d-a081836d5ae4` | bughunter, 11 Oct 02:59 | #205 and preview/recovery/privacy follow-up queue; verify all claims on main. |
| `68b7cab0-caca-4283-a1e8-b0054b740706` | cleaner, 11 Oct 02:59 | #222; ARCH-08 analysis helper failed at quota and supplied no usable completion. |
| `26d56668-ee2e-4b20-a2c2-4d63f36515af` | exchange executor, 10 Oct 17:05 | Earlier #92 is superseded by merged #104; take only remaining RUN packages. |
| `351dbea2-0aad-483c-a433-97c7d9aef072` | KG assessment, 10 Oct 14:52 | Dated analysis/proposal; compare with accepted DEC-018 and integrated KG-02. |
| `65189ad4-8180-43e1-965b-e8480dbeb900` | Government refinement, 10 Oct 15:37 | Dated staged proposal; no measurement or implementation authority follows from it. |
| `8d1c5228-a62e-436d-9a01-cbb45d9c672f` | playground assessment, 10 Oct 17:25 | Earlier paired attempt never reached delegated plan/run/apply. It does not establish a method comparison. |
| `d7023513-ae76-4e26-8c8a-7cbb37d9986b` | CI troubleshooting, 10 Oct 13:15 | Historical advice; compare with the now-integrated Linux/Windows policy before changing CI. |

## Owner questions and deferred work

The local dashboard's `owner-questions.yaml` holds two still-open product questions in addition to backlog `needs_owner` fields:

- Confirm the broader DEC-023 priority classification: a changed file's realized Statements and direct consumers are marked `change`; the impact sets and plans remain unchanged. The old integrator recommended accepting this clarification, but no owner answer was found. Do not record it as accepted.
- Read the newly integrated delegated-method section in the [vision](../vision.md#the-delegated-method) once; the old integrator requested an editorial review without blocking implementation.

KG-04's repository, independently frozen must-change list and thresholds require owner choices. PLAY-07 needs its protocol, cases, budget and subagent limit frozen before a study. CI-04's real-provider manual smoke needs a credential/cost decision and was deferred to that study decision. Register positioning/assumption candidates from IDEA-01 remain deferred before PLAY-07. A CI secret, paid model run, new study, release, Government expansion or deletion is not implied by the session transfer.

## Maintaining the transfer

The handoff snapshot stays dated; subsequent package status is updated in the backlog through the integrator's normal PRs. Link later evidence rather than rewriting an old session's outcome. The local-only `.dashboard/` reads the canonical backlog/roadmap from `origin/main` and uses `sessions.yaml` for live chat state; mark the interrupted Claude roles stopped and request Codex starts without claiming they already exist. Refresh it after integration. Preserve its owner questions until answered.

Local verification observed that fixed-revision `context` on the audited source created `.markitect/state/briefings/history.json`. The generated file was preserved outside this candidate at `C:\Users\Consiliari\.codex\worktrees\session-handoff\handoff-evidence\briefing-history-from-fixed-context.json`, SHA-256 `a1075a249e41255905f4f1202bbdc10917c0fba0a7227268c6fed7cd6f690a6b`. This is an observed current-source side effect relevant to #205/#221's read-only repair; it is not a validation of those unmerged fixes. No generated state was added to the documentation PR.

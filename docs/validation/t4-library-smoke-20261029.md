# T4 native smoke: `markitect apply` on the Library with Claude Managers — 29 October 2026

The first run of concept 2.0's whole Run: T4a's engine, T3's Run remote and T4b's Claude Code adapters together, started by `markitect apply` on the Library (step T4 of the [delivery plan](../implementation-plan.md#concept-20-build): "one bounded native smoke on the Library with Claude"). It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)).

## Questions

- **Q1:** does `markitect apply --full` on a fresh Library project start a Run remote and Claude makers and checkers for every working Manager, and end with a Result?
- **Q2:** does it land: every working Manager approved, Root's checker approving the integrated state, the required checks (`go build`, `go test` in `realization/`) passing, and the result pushed to the target branch?

## Dispatch

| Item | Value |
| --- | --- |
| Source | Main `2ce96daf2ab5e048baaff8fa90bdb74c5d263dce` (PR #856 merged), whose tree equals PR #856's frozen head `3647ebce` (Linux CI green, run 40785365316). `markitect` built for Windows and `markitect-remote` for Linux from it |
| Images | The Manager image from `packaging/runtime` at the source, then its Go example (`packaging/runtime/examples/go`) on it, named by engine image ID. The Run remote and its checks use `golang:1.27`, which `execution.yaml` names: the local `golang:1.27.1` image tagged so, no pull |
| Project | A fresh repository: T2's Library M1 model (`tools/evaluation-tasks/library/m1`) as `.markitect/`, `markitect init`, and one registry page per Package claiming `realization/internal/<package>/**` (Root owns the rest). `markitect check` passes. The target is `main` of a local bare repository |
| Command | `markitect apply --full --deadline 60m`, with `MARKITECT_MANAGER_IMAGE` (the Go image), `MARKITECT_REMOTE_BINARY`, `MARKITECT_ENGINE` and `MARKITECT_TRANSCRIPTS`; provider Claude Code 2.1.308, `claude-sonnet-5-5`, effort `high` |
| Login | The host's Claude Code `.credentials.json`, valid until 06:24Z; each Manager gets the access-only copy. Every `ANTHROPIC_*` and `CLAUDE_*` variable is cleared for the command |
| Limits | One attempt, a second only after a fix of a defect the first one shows. The Run's own limits: 60 minutes, the engine's default 200 starts and 6 rounds per Manager, one hour per start. Containers: 1 CPU, 2 GiB, 256 processes, 256 MiB temporary space each |
| Abort | A login or launch-check failure, a credential found in the evidence, or a Run past its deadline ends the attempt; the engine stops it honestly as incomplete |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, held from the first engine call to the post-check, together with the re-run of `TestNativeClaudeMakerAndChecker` at the same source |
| Cleanup | The engine removes the Run's remote and the agents their containers and volumes; afterwards nothing labelled with the Run may remain |
| Evidence | Outside the repository: the Result, the transcripts, the command's output, the target's log. A post-check scans all of it for the login's tokens, the Run's Git tokens, key names with values, JWTs and Anthropic key shapes |

## Results

| Question | Verdict |
| --- | --- |
| Native Claude re-run | **Passed** at the source, 23:20:53–23:21:24Z: the maker's submission passed the hook and its check, and the checker approved it with a commit that changes no file |
| Q1, a whole Run | **Passed.** `markitect apply --full` started the Run remote and 25 agent starts for the five working Managers (`.`, `catalog`, `lending`, `lending/reminders`, `members`): 18 maker and 7 checker starts. Every start ended `finished` on `claude-sonnet-5-5`, and the Run ended with a Result |
| Q2, landing | **Passed.** Outcome `landed`, exit 0, 23:21:25–23:32:25Z (11 minutes). Every checklist item of every Manager has a verdict with evidence and its approving checker, and Root's checker approved the integrated state. The required checks passed on the whole repository at the integrated commit (`build` 2.4 s, `test` 1.0 s). The result is `main` of the target at `32bb2b250a76`, with the Result page at `.markitect/results/r20261028-232125.md` |

| Item | Value |
| --- | --- |
| Binding | Main `2ce96daf`, tree `a9279f1c` = PR #856's frozen head `3647ebce`. `markitect.exe` and `markitect-remote` built from it. Manager image `sha256:6ac2af372824…`, Go Manager image `sha256:10b5bd8d9cf8…`, both built from `3647ebce`. `golang:1.27` = local `golang:1.27.1`, `sha256:162be5298a40…` |
| Project | 19 Definitions in 5 Packages, base `363af488`. The target was a local bare repository, so no hosted repository changed |
| Tokens | 3,777,706 input and 140,605 output for the Run (from the Result). The re-run: 30 s, about 52,000 input tokens |
| Lock | Held 23:20:53–23:32:26Z by nonce, and again for the judge below. Before that it was held by another session's judge self-test, and this dispatch waited for it |
| Cleanup | Afterwards no container, volume or network of the Run remained |
| Credentials | The post-check scanned all 35 evidence files (Result, transcripts, outputs, target log) for the login's tokens, the agents' Git tokens, key names with values, JWTs and Anthropic key shapes: no match |

**What landed holds up.** In a fresh clone of the target:

- `go build ./cmd/library`, `go vet ./...` and `go test ./...` pass in `realization/` (six packages with tests).
- `markitect check` reports `ok: 19 Definitions in 5 Packages, 15 files accounted`.

As extra evidence, beyond this dispatch, the Library's hidden M1 judge (`tools/evaluation-judges/library-m1`, exported at the source) judged the landed commit against the base, under the lock:

- 19 of 19 hidden cases passed, and build and tests passed.
- No protected input was touched.
- Its scope census flags two paths as unrequested: Root's registry page `.markitect/registry.md` and the Result page `.markitect/results/r20261028-232125.md`. The judge exempts only `.markitect/registry/`, so for the 2.0 arm these are a gap in the judge's contract, not unrequested work.

**How the organization worked.** It took 18 messages (questions, escalations, answers, assignments, and the host's notices of approved child work):

- **The scaffold was missing.** Every child's first submission was refused by the required `build` check: `realization/go.mod` and `cmd/library` lie in Root's area, and the check runs on each branch's full state (concept §6.3).
- **The children escalated.** They asked their parents rather than writing outside their areas.
- **Root resolved it.** It published the module scaffold on `root` and assigned each child an exact API. The children merged `root` into their branches; the hook accepts that because the net change against `root` stays in their area.
- **Rounds and reviews.** `lending` needed five maker rounds and Root four. Every one of the seven reviews approved. `catalog` and `members` each had a second review because their makers submitted again after an approval, on top of a newer `root`.
- **Integration.** Root then merged every approved child, wired the ten operations and passed its checker.

**Not exercised:** Codex; a model defect (no planted defect); a moved target; `verify`; Linux as the host (T5).

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

Pending.

# T4b Claude Code adapter: native qualification in the 2.0 Manager container — 28 October 2026

This is the dispatch and the record of the first real Claude Code turns in concept 2.0's [Manager agents](../../internal/agents/README.md) (step T4, part B, of the [delivery plan](../implementation-plan.md#concept-20-build)). It runs under the owner's standing run approval of 28 October 2026 ([build workstream, Decisions](../workstreams/build.md)).

## Questions

- **Q1:** does a Claude maker start in the 2.0 Manager container with the access-only account login, pass its login check and every per-turn launch check, and end its start through the agent port?
- **Q2:** does its submission pass the real Run remote's hook and required check, and does a fresh Claude checker review it with a commit that the hook accepts?

## Dispatch

| Item | Value |
| --- | --- |
| Source | PR #856, branch `dev/t4b-agents`, at the head recorded below as the freeze; clean clone state, `go test ./internal/agents/...` passing on it |
| Image | Built from that head's `packaging/runtime` and named by its engine image ID; used for the Manager containers, the Run remote and its check |
| Engine | Docker Desktop on this Windows host, native host (no runner) |
| Test | `TestNativeClaudeMakerAndChecker` in `internal/agents`: the Library fixture, where the Definition `members/member` gains a `Name` and an `ID`; members' maker realizes it in `src/members/member.go`; the required check is `test -f src/members/member.go` |
| Provider | Claude Code 2.1.308, model `claude-sonnet-5-5`, effort `high`, network `bridge` |
| Login | The host's Claude Code `.credentials.json` (account login), read only; each Manager gets the access-only copy. The access token must outlive each start's deadline by ten minutes. No API key; the host shell's `ANTHROPIC_*` and `CLAUDE_*` variables do not reach the containers |
| Limits | At most two attempts. Per start: 15 minutes, 1 CPU, 2 GiB memory, 256 processes, 256 MiB temporary space. The whole test: 40 minutes |
| Abort | A start that fails its login or launch check stops the attempt. A provider failure (capacity, rate limit, server error) ends the attempt and is recorded as such. Any token value found in the evidence stops the dispatch |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, created exclusively before the first engine call and removed by its holder after the post-check |
| Cleanup | The test sweeps everything labelled with the Run; afterwards no container, volume or network of the Run may remain (`docker ps -a`, `docker volume ls`, `docker network ls` by label and name) |
| Evidence | Outside the repository: the transcripts, the ends, the Run's branch log. A post-check scans all of it for the login's token values, key names with values, JWTs and Anthropic key shapes |

## Results

Not run yet: the host login's access token expired at 21:36Z, before the dispatch. The run waits for a refreshed login.

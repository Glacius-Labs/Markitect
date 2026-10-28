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
| Login | The host's Claude Code `.credentials.json` (account login); the agents read it only, and each Manager gets the access-only copy. The access token must outlive each start's deadline by ten minutes. No API key; the host shell's `ANTHROPIC_*` and `CLAUDE_*` variables do not reach the containers |
| Limits | At most two attempts. Per start: 15 minutes, 1 CPU, 2 GiB memory, 256 processes, 256 MiB temporary space. The whole test: 40 minutes |
| Abort | A start that fails its login or launch check stops the attempt. A provider failure (capacity, rate limit, server error) ends the attempt and is recorded as such. Any token value found in the evidence stops the dispatch |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, created exclusively before the first engine call and removed by its holder after the post-check |
| Cleanup | The test sweeps everything labelled with the Run; afterwards no container, volume or network of the Run may remain (`docker ps -a`, `docker volume ls`, `docker network ls` by label and name) |
| Evidence | Outside the repository: the transcripts, the ends, the Run's branch log. A post-check scans all of it for the login's token values, key names with values, JWTs and Anthropic key shapes |

## Results

| Question | Verdict |
| --- | --- |
| Q1, Claude Managers in the 2.0 container | **Passed.** Both starts passed the login check (`claude auth status`: `claude.ai`) and the launch check of their only turn: `system/init` reported `claude-sonnet-5-5`, `2.1.308`, `apiKeySource` `none`, and exactly the six built-in tools plus the start's Manager tools (maker: `submit`, `ask`, `escalate`, `note`; checker: `review`, `ask`, `escalate`, `note`). Both ended `finished` through the agent port with an empty `Cleanup` |
| Q2, through the real Run remote | **Passed.** The maker wrote `src/members/member.go` (`Member` with `Name` and `ID`, and a `Validate`) and called `submit` once. The hook accepted it after the required check (`578019593b6f`, one `changed` verdict). The checker read the diff and the registry page and approved it with a commit that changes no file (`6dba3c087b22`), which the hook accepted. The test asserts a review on the branch; that it is an approval is read from the exported branch log in the evidence |

One attempt was used, at 22:25:03–22:25:34Z (31 s for both starts, including container start, clone and cleanup).

| Item | Value |
| --- | --- |
| Freeze | `20716c7c53c5c25772bd502cd98ce6538e814803`, clean worktree. The native test then gained an assertion on the branch's trailers, which does not change what ran |
| Image | `sha256:3e1e0cd65f2b9b47adf3000b98045c824530018ade3c70f6f664a4420229dfce`, built from the freeze with `MARKITECT_VERSION=20716c7c` |
| Engine | Docker Desktop 29.9.2 on Windows 11, host-native |
| Login | Deviation from the dispatch: before the test, under the engine lock, the host login was refreshed with one extra host-side provider turn, `claude -p "ok"`, with every `ANTHROPIC_*` and `CLAUDE_*` variable cleared, as Build directed: its expiry moved from 21:36:23Z to 06:24:14Z on 29 October. No token value was recorded |
| Tokens | Maker: 29,837 input (21,725 of them cache reads), 926 output. Checker: 22,679 input (14,644 cache reads), 935 output |
| Lock | Held from before the refresh (taken about 22:23Z) to 22:26:02Z, then released by nonce. No other container ran on the engine |
| Cleanup | After the test, no container, volume or network of the Run remained |
| Credentials | The post-check scanned all seven evidence files for the login's access and refresh tokens, the agents' remote tokens, Anthropic key shapes, JWTs and token key names with values: no match |

**Observation.** The maker added a rule the model does not state: `Validate` also rejects an empty `Name`. The checker saw it and approved anyway, reasoning that it follows from "full name". Under concept §12 (conforming artifacts preserved, optional improvements noted, not made unasked), this is a borderline addition. It is recorded here for the evaluation's judgement, not changed in the harness.

**Not exercised:** a rejection and a second round, messages along the tree, more than one Manager, the deadline, a provider failure and its class, and Codex, which has no tokens.

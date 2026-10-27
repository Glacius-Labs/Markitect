# Arm P Library smoke chain: dispatch

Status: **proposed, not run.** It runs only after the owner's go-ahead, under the [standing evidence authorization](../../../docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). Failed attempts stay recorded.

## Question

Does arm P ([README](README.md)) complete a Library Apply end to end, and do its candidates pass the same frozen assessment as arm M's? This is feasibility evidence for one chain, not a comparison: N = 1 decides nothing about quality.

## Runs

One attempt each, sequential, under the shared engine lock. Every input is arm M's own, so P starts from the same bytes as M's chain 1 and M's S4 repetitions.

| Run | Scenario | Input | Assessment |
| --- | --- | --- | --- |
| `p-s1-c1` | S1 fresh M1 | M's `m-s1-c1` start bundle, commit `ee7c5d98` (sha256 `ab506840…`) | `P-c1-S1`, profile m1 |
| `p-s2-c1` | S2 M1 → M2 (loan limit 5 → 3) | P's own S1 candidate plus the M2 edit (`armp.py input m2`), as M's controller builds it | `P-c1-S2`, profile m2 |
| `p-s3-c1` | S3 unchanged re-apply | P's own S2 candidate, owner branch `dev/library-m2` | `P-c1-S3`, profile m2; strict no-change is `result_tree == input_tree` |
| `p-s4-r1` | S4 seeded defect | `s4/m-s4-input.bundle`, commit `b079bec1` (sha256 `0cb5e962…`), the input of M's S4 runs | `P-r1-S4`, profile m2 with `s4/seeded-lines.json` |

S2 and S3 are skipped and recorded when their predecessor has no candidate.

## Binding

- **Source:** the committed head of this branch at launch. The script refuses local changes, records `git rev-parse HEAD` in `binding.json` and builds `markitect.exe` from it (`-trimpath -buildvcs=true`, `CGO_ENABLED=0`).
- **Provider:** Claude Code 2.1.309 on Windows, normal account login (Claude subscription), model `claude-sonnet-5-5`, effort `high` for every agent (the owner's model rule for agent roles). Each run has its own `CLAUDE_CONFIG_DIR` with a login copy without refresh token. No user settings, plugins, CLAUDE.md, memory or MCP servers reach a run, and a run cannot rotate the login.
- **Assessment:** unchanged `assess_candidate.py` from the frozen comparison source R `74b7d7df` with oracle F `e11e6cff` and Go image `sha256:1271a363…`, as for arms M and B, with blinded copies and tokens kept apart.

## Limits

| Limit | Value |
| --- | --- |
| Per Apply | deadline 2,700 s (as arm M), then the session's process tree is killed |
| Agents per Apply | 150 agent calls; 2 Verify correction rounds; 2 returns per parent; 3 questions per Manager |
| Project check in the publication guard | 600 s per declared check |
| Login | each run refuses to start when the access token expires within deadline + 600 s |
| Whole chain | 4 Applies, at most about 3 hours, plus 4 assessments under the assessor's own limits |
| Cost | Claude subscription usage; no API key. Arm M used about 7 million tokens per Library Apply on Codex; P's use is unknown and recorded per run from the transcripts |

## Launch

From PowerShell, detached from every Claude Code session, with the evidence directory under `C:\Users`:

```text
Start-Process -WindowStyle Hidden "C:\Program Files\Git\bin\bash.exe" -ArgumentList '-lc', '"tools/comparison/claude-plugin/smoke-chain.sh C:/Users/Consiliari/Documents/Codex/Markitect-ArmP/smoke-01 > C:/Users/Consiliari/Documents/Codex/Markitect-ArmP/smoke-01.log 2>&1"' -WorkingDirectory "<this worktree>"
```

The chain writes `runs/<run>.done` after each run and `chain.done` at the end.

## Abort

Kill the chain's process tree (`taskkill /PID <chain.pid> /T /F`); this also ends the running `claude` session. Then clean up as below. An abort is recorded as such in the result, never retried silently.

## Cleanup

- After each run the runner removes every worktree, the work repository and the login copy under `C:\Users\Consiliari\AppData\Local\Temp\mk-armp\<run>`, and records anything left as residue, which fails the technical pass.
- After an abort: delete `mk-armp\<run>` and remove the lock file only if its holder is `comparison arm P`.
- Retained per run: `result.json`, `session.stream.jsonl`, `transcripts/`, `state/` (prepare, the host record of every agent, workflow result, publication), `start.bundle`, `result.bundle`, `work.bundle` and `files.sha256`. No login copy is retained, and the secret scan quarantines any output that contains the access token.

## Result

Not run.

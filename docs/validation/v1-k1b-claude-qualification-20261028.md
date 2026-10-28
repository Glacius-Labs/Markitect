# K1b Claude Code adapter: first native qualification through the Linux runner — 28 October 2026

This record covers K1b, the first native run of the [Claude Code adapter](https://github.com/Glacius-Labs/Markitect/blob/f23c2e12d9833efa61e5e65c90018016423abe74/internal/adapters/claudecode/README.md) (slice K1, [owner decisions](../concepts/sources/owner-night-decisions-20261027.md)). It is an addition beyond V1's Codex scope ([D18](../concepts/v1-concept.md#23-provider-scope-d18)). The lane 1 coordinator dispatched it on 28 October 2026 under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization), after the owner approved starting K1b as one bounded Library smoke in the [Linux runner](../design/linux-runner.md) through the owner's Claude subscription. The procedure follows the [L1 runner qualification](v1-l1-runner-qualification-20261027.md).

It has two questions:
- **Q1:** does the adapter start Claude Code Managers natively with the subscription login, under its launch checks?
- **Q2:** does one bounded Library Apply through `markitect apply --runner` pass with Claude Managers?

It also answers the CLI assumptions that the adapter README lists for K1b. The dispatch allowed at most three attempts. Two were used, and both are recorded.

## Result

| Question | Verdict |
| --- | --- |
| Q1, native Managers with the subscription login | **Works after a source fix.** On Main `ac56780d`, attempt 1 ended at the first Manager after 95 s. In the adapter's provider environment, `claude auth status` exited 1, because `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` makes Claude Code 2.1.308 refuse every command without bubblewrap. Offline probes of the pinned CLI then showed two more launch-contract gaps: built-in plugins that the launch check refuses, and an expected `EndConversation` tool that the CLI does not report. [PR #807](https://github.com/Glacius-Labs/Markitect/pull/807) fixes all three. On its head `9630f381`, all 18 Manager sessions passed the login check and every launch check |
| Q2, one Apply through the runner | **Passed on `9630f381`.** Attempt 2 passed: Assessment approved, Implement submitted, Verify approved, with 18 messages, in 752 s. The verified change arrived as the new branch `markitect/apply-4628d7795d09736e6c8ca6530f50c363`. The input checkout was unchanged. No value of the real login appeared in the evidence, and the attempts' own outputs held no token pattern. No container or volume of either attempt remained |

**Model.** The Managers ran `claude-sonnet-5-5`, as every init line reported, selected with the alias `sonnet` at effort `high`.

**Cost of the passing attempt.** 5,534,875 tokens, of which 4,659,767 were cache reads. Claude Code reported an API-equivalent cost of 4.39 USD; the subscription is not billed per token.

**Usage.** In Claude Code's rate-limit events during the attempt, the five-hour window stood at 5–7 % and the seven-day window at 8–9 %. The account is shared with the interactive sessions, so these values bound the Apply's share from above.

**Not observed.** Main `ac56780d` without #807 cannot run a Claude Apply. The following were not exercised: `--resume` across responses, a repeat after a transient error, the `overloaded` category, login expiry and a CLAUDE.md in the work tree ([K1b assumptions](#k1b-assumptions)).

## Binding

| Item | Value |
| --- | --- |
| Freeze 1 (attempt 1) | Main `ac56780dd4f0e6297c7aec94c4c45325e85dc69f`, tree `5fd7b66582ad5db338d30f4793a18a770595ddd4`, the current `origin/main` at the start. Linux CI [run 40698247172](https://github.com/Glacius-Labs/Markitect/actions/runs/40698247172) passed on exactly it. Fresh clean clone |
| Freeze 2 (attempt 2) | `9630f381bddce82e9409070992d878f60e3d4beb`, tree `74106012c2a1c85f3201a6ef80ccdc47a06d8e2c`. It is the head of [PR #807](https://github.com/Glacius-Labs/Markitect/pull/807), which is freeze 1 plus the fix. Linux CI [run 40702601953](https://github.com/Glacius-Labs/Markitect/actions/runs/40702601953) passed on exactly it. An independent complete-diff review approved with nits; the nits are in this head, and a delta review approved them. Not merged when attempt 2 ran. Fresh clean clone |
| Product binaries | Built with `-trimpath` and go1.27.1, `vcs.modified=false`. Freeze 1: SHA-256 `d243a68954993e9b3264fb8013eadada5941e4e38ae4c857b8907595beee405e`, productBuild `0.14.1+ac56780d…`. Freeze 2: SHA-256 `00fab4cd46fd82b5f5fc57d9c5e5c8cb1754f53c760a760ab5dce2b83e46c667`, productBuild `0.14.1+9630f381…` |
| Agent images | Freeze 1: base `sha256:9f157022c8a9512ef2e4ef7294f6d89fdbbcec96da2fd7f128301bd703ee3fc2`, Go `sha256:070a841bfc70b0b1be7a88d94dc183cdc46e0ae02cb7430d59b3e23f11f88879`. Freeze 2: base `sha256:ae8ff2020a658731ce07a6baf41468ae63d8148c6215716f266307bc19192a89`, Go `sha256:d09d9b3b0174dcc9710a987ca3cf61445ddd772f2082a6383baca6fba3c06b02`. In both, `runtime.env` holds `codex-cli 0.162.0`, its schema digest `sha256:816cef0e…`, `agentd` at the freeze and `claude=2.1.308 (Claude Code)`. A smoke ran `go version`, `codex --version` and `claude --version` in the Go image |
| Runner images | From `packaging/runner/Dockerfile` with `MARKITECT_REVISION` set to the freeze. Freeze 1: `sha256:6a4772b1dba4593c85a8e88f1ad27a6d199491c8b2552d6e0bb3ed426e39c8b2`. Freeze 2: `sha256:0bd1d2217b11c7032c8720fa1e3584e2cf556a7a2ff1ecdd6221d7504effc450`. `runner.env` names the freeze's productBuild and revision, Docker CLI 29.8.2 and Git 2.47.3 |
| Engine | Docker Desktop, engine 29.9.2, context `desktop-linux`. `DOCKER_HOST` and `DOCKER_CONTEXT` were unset |
| Provider | Claude Code 2.1.308, profile `claudeCode` version 1 with `model: sonnet` and `effort: high`. The login mode was `subscription-credentials`: the host's own Claude Code credentials file, a claude.ai account login of a Max subscription. The launcher mounted it read-only, and the adapter gave each Manager an access-only copy. The file was otherwise only hashed and parsed for its expiry, and it was unchanged before and after both attempts. Its access token was valid until 21:36Z, past both deadlines |
| Launch environment | This host's interactive Claude sessions set `ANTHROPIC_BASE_URL` and many `CLAUDE_CODE_*` variables in their shells. Each launch ran in a wrapper started through WMI, outside those shells. Its environment had no `ANTHROPIC_*`, `CLAUDE_*` or `DOCKER_*` name, so Apply's refusal of API-key variables had nothing to refuse |
| Input | A fresh Library M1 project from the freeze's `testdata/library/m1`, with `testdata/library/vocabulary`, `testdata/core`, an empty registry and an execution file byte-identical to L1's. The operator script that generated that file is no longer in the source. Its tree `6d2b71bfa53559adf764188013b98ee372819ef8` and its three canonical model digests (Library `sha256:c7051fec…`) are identical to the L1 input's. 6 Managers (`.`, `catalog`, `lending`, `lending/reminders`, `library`, `members`). Committed and clean on `dev/library-m1`: head `5ce3a94a` for attempt 1 and `a5b61726` for attempt 2 |
| Run configuration | Format 4 with L1's limits: `maxMessages` 128, `providerIdleTimeoutSeconds` 600, an Apply deadline 5,400 s from preparation, 32 Managers, cleanup 45 s, persistence 30 s. `repositoryIdentity` was `markitect-k1b`, so each attempt had its own state volume, `markitect-state-markitect-k1b`. The credential `claude-account` maps to provider `claude-code` and mode `subscription-credentials`. The runtime profile has one `claudeCode` Agent profile on the Go image: no instructions, `maxWireBytes` 4 MiB, 1 CPU, 2 GiB, 192 PIDs, 256 MiB temporary storage, the engine's default network. The `library-tests` check profile is as in L1 |
| Reservation | `NATIVE-RUN.lock` held from each attempt's preflight to its post-check and released by nonce. Before each attempt, `docker ps` showed no running container |

## Attempt 1: stop at the first Manager

| Item | Value |
| --- | --- |
| Launch | 13:46:10Z to 13:47:45Z, 95 s, launcher exit code 3. Runner container `markitect-runner-7b882f525481ea3a` |
| Apply | `dc5f4c5597bd47d55266cc38f50a04ad`, `not_completed`: "Claude Code session refused or incomplete: claude auth status reported no readable authMethod". The Assessment Run ended technically failed with 0 messages. Its four resources, Root's session and working copy, the Manager workers and the Git service, were recorded as closed. No inference, no tokens, no branch |
| Root's transcript | Setup at 13:47:43.93Z. `claude auth status` exited 1 after 0.11 s, and the adapter read no `authMethod` from its output. The transcript keeps only the method and the exit code, never the status output |
| Stop cause | A source defect. With the adapter's provider environment, Claude Code 2.1.308 refuses every command. Adding one variable at a time to a minimal environment in the Go image shows that `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` alone causes it: "bubblewrap is required for subprocess env scrubbing and isolation". The image has no bubblewrap, and the hardened container could not run it. The image build's flag check runs `claude --help` without that environment, so it could not see this |
| Concurrent activity | Shortly after the preflight, another session's G0 prototype containers (`mkp-p0-…`) started on the engine. They were not a Markitect Apply, did not use the lock and were not touched |

## Offline probes between the attempts

Before spending a second attempt, the pinned CLI was probed in freeze 1's Go image; the evidence keeps each probe's script, output and container arguments. The probes ran without network, with a read-only root, as user 1000, on a temporary home with a dummy access-only login and no real token. A stand-in stdio MCP server advertised the nine Run tools. No probe reached a provider.

| Probe | Observation |
| --- | --- |
| Environment | Each provider variable alone leaves `claude auth status` working, except `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1` |
| `claude auth status` | With the access-only file, no refresh token: `loggedIn` true, `authMethod` `claude.ai`, exit 0. Without a file: `authMethod` `none`, exit 1. JSON is the default output |
| Full launch argv | Every flag was accepted and `system/init` was printed. By default it lists the plugins `cc-plugin-agents-md` and `cc-plugin-plugin-authoring`, which the launch check refuses. `--settings '{"enabledPlugins":{…@builtin:false}}'` removes them; `DISABLE_PLUGIN_AUTOLOAD=1` does not. The tool list is the six built-in tools and the nine Run tools, without `EndConversation`. The Run tool server was `connected`, also when it answered `initialize` only after 6 s, so the init line waits for it |
| A response to its end | Without network: ten `system/api_retry` lines with `error` `unknown`, then a synthetic assistant message with `error` `server_error` and a `result` with `is_error` true, exit 1. After the exit, `/proc` showed no process left, the MCP server child included. A second process with `--resume` and the same session ID started and reported that session ID |
| Home | After launches, `/home/agent/.claude` held `.claude.json`, `.credentials.json`, `.last-cleanup`, `backups/`, `projects/<cwd>/<session>.jsonl` and `sessions/`, none of the user-scope entries agentd watches |

[PR #807](https://github.com/Glacius-Labs/Markitect/pull/807) makes three changes:
- It drops the scrubbing variable. The environment carries no credential to scrub.
- It passes `--settings`, turning off the twelve built-in plugins the binary registers.
- It expects the observed tool list.

The image build's flag check now also requires `--settings`. A test checks the captured init line against the launch check, with and without the built-in plugins.

## Attempt 2: one Apply through the runner

| Item | Value |
| --- | --- |
| Launch | 14:13:04Z to 14:25:36Z, 752 s including clone, Apply, hand-back and import, launcher exit code 0. Runner container `markitect-runner-f7bb9651c85ac381`. A `docker ps` sample every minute showed 6 Manager containers and the Git service per Run, and once a check container |
| Apply | `4628d7795d09736e6c8ca6530f50c363`, passed with 18 of 128 messages and 0 provider retries. Assessment: approve, 6 messages, 14:14:19–14:15:43Z. Implement: submitted, 6 messages, 14:15:49–14:22:13Z. Verify: approve, 6 messages, 14:22:29–14:25:14Z. `project-tests` exited 0 on the candidate in Implement and in Verify. Publication was read back and not a no-change publication |
| Hand-back | The runner logged "Apply passed; commit 9985d7e9… handed back". The result is `refs/heads/markitect/apply-4628d7795d09736e6c8ca6530f50c363` at `9985d7e98876684b3a007ae3e3e13f7c33d7cfac`, author `Markitect Runner <runner@markitect.invalid>`. Its single parent is the base, and it changes 22 paths, exactly the report's `completedPaths`: the Go realization under `realization/` and the six registry shards, Root's included |
| Input checkout | HEAD, branch, index, status and working tree unchanged. The result branch is the only new ref, and no ref was removed |
| Sessions | 18 sessions, 6 per Run, with 36 process starts: one `claude auth status` per session, all reporting `claude.ai`, and one response per session, started with `--session-id`. No session received a second message, so `--resume` was never used. Every `system/init` reported `claude-sonnet-5-5`, `2.1.308`, `apiKeySource` `none`, `bypassPermissions`, exactly the 15 expected tools, `markitect` `connected`, and no agents, skills, plugins or slash commands. All 18 results had `is_error` false |
| Stream | Besides `system/init`, `assistant`, `user` and `result`, the CLI emitted `rate_limit_event` (27), `system/thinking_tokens` (81) and `tool_progress` (8). `tool_progress` carries the running tool's `parent_tool_use_id`; the adapter keeps these as unknown lines, and its subagent refusal applies only to assistant and user messages. No `system/api_retry` and no control request occurred |
| Tools | 77 Run tool calls were served: 15 `delegate`, 15 `await`, 15 `review`, 21 `judge`, 8 `check` and 3 `read_notes`. Three `judge` calls were refused by the tool contract: invalid arguments, a child without an assignment, and a Finding with a Definition in a Verify approval. The Managers corrected each within the same response. Native tools: 49 `Bash` and 2 `Read`, never `Edit` or `Write`; the Managers wrote files through the shell. The longest `await` took 115.6 s, by Root in Implement. The largest Run tool reply, 41,487 bytes, arrived inline |
| Tokens | Input 230, cache creation 803,804, cache read 4,659,767, output 71,074: 5,534,875 in all. By Run: Assessment 781,961, Implement 2,690,747, Verify 2,062,167. Claude Code reported API-equivalent costs of 0.96, 1.98 and 1.45 USD, together 4.39 USD |
| Usage | 27 `rate_limit_event` lines, all `allowed` with overage disabled. Five-hour window 5–7 %, seven-day window 8–9 % |

The refusal of invalid `judge` arguments reached the Manager with the prefix "Claude Code session refused or incomplete:", although the session continued. That is a wording note, not a defect.

## K1b assumptions

The adapter README lists what K1b must confirm on the pinned image.

| Assumption | Answer |
| --- | --- |
| The CLI accepts every launch flag | **Confirmed**: offline and in 18 native launches, `--settings` included. `--effort` was used with `high` only; `xhigh` and `max` were **not reached** |
| The consumed stream-json shapes | **Confirmed** for `system/init` with every checked field, assistant and user messages with `parent_tool_use_id` and content blocks, and `result` with `subtype` and `is_error`. The exact tool list has neither `EndConversation` nor `WaitForMcpServers`; this **refutes** the earlier expectation of `EndConversation`, fixed in #807. `system/api_retry` and the assistant `error` category were seen only offline. New line types: `rate_limit_event`, `system/thinking_tokens`, `tool_progress` |
| One input message and closed standard input give one response and exit | **Confirmed**: 18 responses, each with one `result` and an exit. No native process was left after any exit; offline, `/proc` showed no process of the CLI after its exit either |
| `--resume` continues the conversation; a repeat after a transient failure is accepted | **Not reached** natively, because no Manager received a second message. Offline, a `--resume` process with the same session ID started and reported it; that the conversation continued was not observable without the provider |
| The prompt files replace and extend the default prompt; user-only settings, disabled CLAUDE.md, disabled slash commands and strict MCP keep the rest out | **Partly confirmed.** The files were accepted, and the Managers worked under the harness prompt. The init lines show no skills, slash commands or agents, and only the `markitect` server. Built-in plugins loaded until #807. That a `/work/CLAUDE.md` stays out was **not reached**: the input has none |
| agentd's watched entries are all that Claude Code loads from its home; replace the deny-list with an allow-list | The CLI's own state is listed under [Offline probes](#offline-probes-between-the-attempts) and includes none of the watched entries. No native start was refused by the watch. The allow-list is **not done**; it remains a source follow-up |
| 2.1.308 does not hot-reload user settings within a process | **Not reached** |
| An access-only credentials file authenticates until expiry, and `claude auth status` reports `claude.ai` | **Confirmed** for authentication and status: 18 sessions authenticated and worked with the access-only copy. Behaviour at expiry was **not reached**: the access token outlived the run |
| A `claude setup-token` token as `accessToken` | **Not reached**; an owner decision first |
| The tools' `_meta` annotations keep results inline and the tools loaded; the server reports `connected` or `pending` | **Confirmed** for loading: all nine Run tools were present at every launch, and the server was `connected`. Inline results up to 41,487 bytes were **confirmed**; the 500,000-character bound was **not reached** |
| A long `await` completes under the per-response `MCP_TOOL_TIMEOUT` | Awaits up to 115.6 s completed. An await longer than Claude Code's own defaults was **not reached** |
| `overloaded` is the category of a model at capacity | **Not reached**: no provider error occurred. Offline, a network failure ended as `server_error` |

## Cleanup and credentials

- **Containers.** After each attempt, no runner, Manager (`markitect-claude-…`), Git-service or check container of the attempt remained. The runner was started with `--rm`, and Apply closed the others. The probe containers also ran with `--rm`; none remained.
- **State volume.** Each attempt's state volume was copied with `tar` into the evidence and then removed. No volume of either attempt remained. The launch's private temporary directory was empty.
- **Images.** The six images built for the two freezes are kept, as in L1.
- **Credential scans.** The post-check of each attempt scanned:
  - the launch directory, the host state directory and the extracted state volume (50 files for attempt 2);
  - the checked-out result branch with its commit text.

  It searched for token key names, JWTs, Anthropic key and token shapes, bearer headers and the login's own token values. The values were held in memory only. There was no match.

  A final scan covered all 589 evidence files except the two source clones, and its output is kept with the evidence. It found no login value. Pattern matches occurred only in the product binaries (key names in code), the probe scripts (dummy tokens) and the scan tool.
- **Lock.** `NATIVE-RUN.lock` was released after each attempt.

The evidence is kept outside the repository: for each attempt, the build records, the input, `pre.json` and `post.json`, the launch logs, the extracted state volume and the analysis. The probes keep their scripts and outputs.

## Limits

- **One Apply, on the smallest native case.** One pass with 6 Managers, one response per Manager and no correction. Not exercised: `--resume` across responses, a Verify reject with a correction pair, a repeat after a transient error, `overloaded`, and larger projects with more concurrent Managers on one subscription.
- **Main needs #807.** On Main `ac56780d`, every Claude Apply ends at the first Manager. Attempt 2 ran on the fix head before its merge.
- **The source tests could not see these defects.** They use authored fixtures. The CLI's behaviour under the adapter's environment and flags was first observed offline and then natively. The new test pins one captured init line, not the CLI.
- **The login was not stressed.** The access token still had about seven hours left when the run ended, so neither expiry nor refresh occurred. The mount is read-only and the Manager copies carry no refresh token, so a long evaluation run can outlast a token; the README's refresh limitation applies.
- **Loading.** Not observed: a repository `CLAUDE.md` or `AGENTS.md` in `/work`, and the replacement of the default system prompt as the provider sees it.
- **Engine and launch.** Docker Desktop only. The launcher ran in a WMI-started wrapper, not from a console. No cancellation was tried.
- **Offline probes.** They used a dummy token without network. Attempt 2's 18 native init lines confirm their plugin and tool-list observations; their retry and error categories remain offline observations.

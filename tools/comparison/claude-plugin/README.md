# Arm P: Markitect Apply as a Claude Code plugin

Arm P runs Markitect's Apply method with a Claude Code plugin instead of the native runtime: Assessment, Implement and Verify through the complete model-derived Manager tree, reviewed Git handoffs and guarded publication. It is an evaluation arm next to arm M (native `markitect apply`) and arm B (one agent with ordinary documentation) of the [comparison protocol](../../../docs/validation/v1-comparison-protocol-20261024.md), so the method can be judged apart from its runtime ([DEC-010](../../../docs/concepts/register.md#dec-010-evaluate-the-method-separately-from-its-execution-runtime)). It is not a product path: V1 ships Codex only ([D18](../../../docs/concepts/v1-concept.md#23-provider-scope-d18)).

Status, 27 October 2026: built and tested offline; no Apply has run. The first native evidence is the [Library smoke chain](smoke-dispatch.md), which needs the owner's go-ahead.

## Why

The [comparison report](../../../docs/validation/v1-comparison-report-20261026.md) found no quality advantage for arm M over arm B, at 9 to 10 times the time and 9 to 42 times the tokens. Where M delivered it missed no hidden obligation; it lost Phase C on steps without a candidate, nine of eleven on runtime limits. Arm P asks whether a much thinner runtime keeps M's delivered quality and loses those failures. If it does, the runtime packages (`adapters/codexappserver`, `adapters/container`, `run`, `agentd`, `harness`, `workflow`, `apply`, `runtimeconfig`, `adapters/runstore` and the workspace part of `adapters/gitrepo`, about 12,000 to 14,000 lines) would be candidates for replacement. Model, registry, accounting and context stay as they are.

## How it works

| Piece | File | Does |
| --- | --- | --- |
| Command | [plugin/commands/apply.md](plugin/commands/apply.md) | `/markitect:apply`: prepare, run the workflow, store its result, publish. The session agent only connects these steps |
| Workflow | [plugin/workflows/walk.js](plugin/workflows/walk.js) | Assessment, Implement and Verify over the tree; correction rounds after a Verify reject; limits |
| Agents | [plugin/agents/manager.md](plugin/agents/manager.md), [plugin/agents/verifier.md](plugin/agents/verifier.md) | The native Manager rules from `internal/harness`. The verifier has no file-writing tools |
| Host steps | [plugin/scripts/mkp.py](plugin/scripts/mkp.py) | `prepare` (structural check, `markitect tree`), `check` and `context` for agents, the `record` and `guard` hooks, the publication guard |
| Hooks | [plugin/hooks/hooks.json](plugin/hooks/hooks.json) | `SubagentStop` records every Markitect agent's mandate, round, Package, checked-out commit and structured answer from the agent's own transcript, and attaches `markitect check` of every Manager handoff as a git note (D11); `PreToolUse` refuses push, ref and worktree surgery, forged notes, publication branches and the owner's branch, judging only the command word and git subcommand |
| Runner | [../armp.py](../armp.py) | Inputs, one isolated headless session per Apply, readback, bundles, transcripts and `result.json` |

**Workflow.** Every `agent()` call is a fresh context in its own git worktree on a branch of its own, because git checks a branch out in only one worktree and Claude Code keeps every worktree that committed. Assessment and Verify run bottom-up: children judge first, then the parent judges its own area, the interfaces and the children's judgments; a child's reject or model_blocker stays in force. Implement runs top-down and back up: a Manager realizes its own Definitions and the shared prerequisites and writes one brief per child; the children work in parallel from that commit; the parent reviews each handoff by its diff and check note, merges what it accepts and returns the rest with Findings. A question ends the asking agent's turn; the nearest ancestor that can answer does, and the asker restarts from its own branch. If even Root cannot answer, the Apply ends as a model blocker. A Verify reject starts a fresh Implement round from the last candidate with the Findings that name a Definition (DEC-073), then a fresh Verify.

**Publication.** `mkp.py publish` does not trust the workflow's or the session's report. It takes the candidate from the host's own records: the latest Verify round in which every Package's verifier approved the commit its worktree had checked out, with no reject on that commit in that round. It sets `markitect/apply/<run>` only when, in addition, the workflow names the same candidate if it names one, the candidate is a Root result of that round, the owner's branch is still at the bound input, the candidate descends from it, no protected model or control input changed (renames included), `markitect check` passes on the candidate without stale or orphaned registry entries, and every declared project check passes on a detached copy. Apply never commits to or pushes the owner's branch.

**Isolation.** The runner gives each session its own `CLAUDE_CONFIG_DIR` with a login copy without refresh token, `--setting-sources project` and `--strict-mcp-config`: no user settings, plugins, CLAUDE.md, memory or MCP servers reach the arm, and a run cannot rotate the user's login. The access token must outlive the deadline. Agents get a private `HOME`, Git without system configuration, credential helper or prompts, and no Claude, Anthropic, Git or GitHub variables. They still run as the user on the host and could read the user's files by absolute path: this is separation, not a sandbox. Tools are limited to Bash, Read, Edit, Write, Grep, Glob and Workflow.

## Deviations from concept 1.0

| Concept | Arm P |
| --- | --- |
| A waiting Manager stays reachable; questions are answered live (§8.2, §9.4) | A question ends the agent's turn; the answer restarts it from its own branch with the answers in its task |
| Managers plan their own delegation (§3.3, §8.1) | The script fixes the cycle per level: own work and briefs, children, review and integration. The content of each step is the Manager's |
| A correction continues the child's conversation (§8.5) | A correction is a fresh agent on the child's branch with the Findings and the parent's current base |
| One container per Manager, checks in the check profile's image (DEC-037) | One git worktree per agent on the host; agents and the publication guard run `go` on the host. The assessment runs the candidate's build and tests in the frozen Go image as for every arm |
| Message budget and provider idle limit | An agent allowance per Apply and the outer deadline |
| Stored Run records and candidate bundles per Run | The workflow result, publication record, check notes, every Manager branch (`work.bundle`) and the transcripts |
| Host fact "already verified" for Verify (DEC-073) | Not supplied |

## Cheap checks, 27 October 2026

Each ran once with Claude Code 2.1.309 on Windows, model Haiku, in scratch repositories; no Apply ran.

| Check | Result |
| --- | --- |
| Workflow tool in a headless `claude -p` session, named plugin workflow | Works; the session waits for the background workflow and reports its result |
| `SubagentStop` and `PreToolUse` for workflow agents | Both fire for workflow agents. `cwd` is the agent's own worktree and `agent_type` the plugin agent's name |
| Worktree branching | Two parallel agents branched from a given commit in their worktrees and committed; branches survive in the shared repository |
| Agent count | 20 parallel agents all returned. The runtime queues above min(16, CPUs − 2) |
| Worktree path | Isolation refuses repositories under `C:\WINDOWS\TEMP` because git spells the path `C:/Windows/Temp`; the runner refuses such work roots |
| Plugin in an isolated configuration | Only this plugin and Claude's built-ins load; no MCP server; `markitect:manager`, `markitect:verifier` and `/markitect:apply` are registered |
| Structured answers with restricted tools | Both agent types answer through their schema; the verifier's tools are exactly Bash, Read, Grep, Glob and StructuredOutput |
| Check note on a Manager commit | The hook ran `markitect check` in the Manager's worktree on the Library M1 fixture and attached the passing result to the commit |
| Host record of a verifier | Under the runner's session environment, the record hook read mandate, round and Package from the agent's own transcript, its structured answer and the commit its worktree had checked out |
| Guard inside an agent | Not exercised live: the Manager agent declined on its own to forge a check note. The guard rules are covered by the offline tests |

An [independent review](#review) found defects in the first version; they are fixed and covered by the tests below.

The headless session agent declined a probe that asked for `git push` and a script path outside the working directory until told it was authorized. The real command asks for neither, but a cautious session agent remains a risk the smoke run will show.

## Offline tests

`python -B -m unittest discover -s tools/comparison` includes [test_armp.py](../test_armp.py): inputs, the login copy, usage and stream records, the guard, protected paths, publication refusals, the analyzer's arm pairing and, when `node` is installed, the workflow itself through [simulate.js](simulate.js) with scripted agents. The simulations check that every Package takes part in every mandate with children before parents, that a child's reject starts a fresh Implement with bound Findings, that questions climb and an unanswered question is a model blocker, that a returned child is corrected and re-integrated, that a child the review left undecided is reviewed again, that every agent gets its own branch, that an Assessment blocker stops before Implement and that the agent allowance ends the run with its reason, also when it is reached outside `parallel`. The host tests cover the guard against message and path text, renamed model files, the record hook on a transcript and the rule that only a Verify round in which every Package approved the same commit can publish.

## Review

The first version had reused branch names across agents (a stall once a worktree kept its branch), a protected-path check that missed renames, a guard that matched commit-message text, publication that trusted the session's transcription of the workflow result, unvalidated handoff commits, Findings routed by reporter instead of by the Definition's Package, a login-change rule that failed runs when the owner used Claude meanwhile, and a too short project-check limit. Each is fixed: unique branches, `--no-renames`, a token-based guard, publication from host records, commit patterns and complete child decisions, routing by Definition, a recorded login change and a 600 s project-check limit.

## Comparison

Arm P is assessed exactly like M and B: [assess_candidate.py](../assess_candidate.py) on `result.bundle` against `start.bundle`, from the frozen assessor and oracle revisions. [analyze.py](../analyze.py) `--arms P,B` reads P against B with the protocol's rule, and `--arms P,M` against M. P against B is the clean comparison of method under one provider only when B also runs on Claude; against the recorded Codex arm B and arm M, provider and runtime are confounded, so those readings speak to reliability and cost, not to the method alone.

Stages: the [Library smoke chain](smoke-dispatch.md) (feasibility), Library repetitions as in Phase A, then the Makerspace chains of Phase C, which are the decisive test because both native arms conformed on the Library.

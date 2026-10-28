# Manager agents

Every working Manager of a [concept 2.0](../../docs/concepts/v2-concept.md) Run is two agents, a maker and a fresh checker (§6.2). This page is the canonical description of how one agent runs: its container, its login, its harness, the tools it answers through, the provider adapters and the default model. It is part B of step T4 of the [delivery plan](../../docs/implementation-plan.md#concept-20-build); the Run engine that starts agents, sends them messages and writes the Result is part A. The [Run remote](../runremote/README.md) they push to is step T3.

| Package | Role |
| --- | --- |
| `internal/harness` | Pure: renders what a maker or checker receives from the loaded project and its checklist, and the tool contract. |
| `internal/agents` | One agent in its container: start, login check, clone, one provider turn per message, the Manager tools, close. Knows no provider. |
| `internal/agents/claude` | The Claude Code adapter: launch contract, access-only login copy, stream-json reader, failure classes. |
| `internal/agents/codex` | The Codex adapter: the same for `codex exec`. |
| `packaging/runtime` | The Manager image. |

## One agent

`agents.Start` takes a `Spec` and a `Provider` and does, in order:

1. Renders the harness (below). A project with compile errors starts no agent.
2. Admits the operator's login file through the provider and refuses an API key, an expired access token and an access token that expires before the agent's deadline.
3. Creates two fresh volumes, `NAME-home` at `/home/agent` and `NAME-work` at `/work`, labelled like the container, so that a sweep by the Run's label finds them.
4. Starts the container from the Manager image: `markitect-agentd serve` as user `1000:1000`, read-only root, no capabilities, no new privileges, the given CPU, memory, process and temporary-space limits. It joins the network that reaches the provider (`Network`, for example the evaluation's egress-limited network) and the Run's internal network (`RunNetwork`, `markitect-run-RUN`); `Hosts` and `DNS` pass to the engine as they are. No folder of the user's is mounted.
5. Installs the private home through agentd: the access-only login copy and, for Claude Code, the two system prompt files.
6. Runs the CLI's own login report (`claude auth status`, `codex login status`) and requires the account login.
7. Stores the agent's remote user and token in Git's credential store in the private home and clones `http://remote:8080/run.git` into `/work` at the Manager's branch (`runremote.Branch`). The token is the agent's own identity at the remote; the remote holds only its hash.

`Agent.Send(ctx, message)` runs one provider turn. The first turn's message is the harness's task, followed by `message` when it is not empty; later turns carry only `message`, so the engine's events (a checker's rejection, an answer, a child's approval) arrive as later turns. A turn ends when the provider ends it, at the agent's deadline (`Deadline`) or when `ctx` ends (`Stopped`); agentd then stops the process group, and a process that outlived the turn is an error. The `Outcome` says how the turn ended, its failure class, the provider conversation, the reported model and the token usage. Nothing waits for a message: the message tools return at once.

`Agent.Close` removes the login from the private home, ends the container and removes both volumes; every failure is reported. A failed start removes what it created.

`Spec.Transcript` receives the provider's output lines as they came and the host's records (turns, tool calls and replies, outcomes) as lines of type `markitect`. Every secret, the login's tokens and the remote token, is redacted before it is written, also from tool results and errors.

## Failure classes

A failed turn carries a class, so that the evaluation's classifier can void the provider's failures and count everything else ([evaluation suite](../../docs/guides/evaluation-suite.md)):

| Class | Claude Code | Codex |
| --- | --- | --- |
| `capacity` | `overloaded`, HTTP 529 | "at capacity", "overloaded", 529 |
| `rate-limit` | `rate_limit`, `billing_error`, 429 | rate or usage limit, 429 |
| `server-error` | `server_error`, `api_error`, `unknown` (a dropped connection), 5xx | 5xx, a broken stream or connection |
| `login` | `authentication_failed`, 401, 403; a refused or expired login at start (`StartError`) | 401, 403, an expired token; the same at start |
| `other` | everything else, including a breach of the launch contract | the same |

Claude Code's category is the one of the last failed request after its own retries: an `api_retry` line or the assistant message's `error`; a later assistant message without an error clears it. Codex's class is read from the error text of `turn.failed` or `error`.

## Harness

The harness renders three texts per agent; prompts are a task and an answer format, and process lives in host state and the tools ([concept §12](../../docs/concepts/v2-concept.md#12-kept-from-concept-10)).

- **Base:** the role (maker or checker of which Manager, in which Run and mode) and the repository: the clone and the branches, the registry page and the paths the Package claims, the parent and the children that work in this Run, what the remote accepts from this agent, and the `markitect` commands that read the model in `/work`.
- **Context**, the model the Manager knows (§5): its own Package's Definitions, everything its parents define (Root first), the Definitions these link to in other Packages, the Kinds of all of them in full, and for each direct child a table of contents of the public Definitions in its subtree, with title and first sentence. Every entry names its file; nothing is truncated.
- **Task:** the checklist (item key, part, why) and the answer format. A maker makes the repository conform to every item and submits one verdict per item; a parent with an empty checklist integrates its children's approved work and submits with no verdict. A checker in `apply` re-checks every verdict of one submission against the model and the diff from `base`; the diff is inline up to 256 KiB and listed by file above that. A checker in `verify` judges the base.

`harness.Items` turns an impact checklist into items keyed exactly as the Run's plan keys them (`runremote.Items`). For Claude Code, Base replaces the CLI's system prompt and Context is appended to it; for Codex both precede the task in the first message.

## Tools

The tools reach the host over MCP through `markitect-agentd mcp`. Every answer, a refusal included, is a tool result the agent reads; a refusal never ends the turn.

| Tool | Who | Does |
| --- | --- | --- |
| `submit` | maker | Requires one verdict per checklist item (`conforms`, `changed`, `model-problem`) with evidence, commits the whole working tree with the [verdict trailers](../runremote/README.md#verdict-trailers) and pushes it to the Manager's branch. A push the remote refuses (paths, checks, trailers) returns the hook's reason, and the commit is undone, so the work stays as it was. |
| `submit` | verify checker | One verdict per item (`conforms`, `deviates`, `model-problem`) as a commit with the tip's tree on the tip, pushed. |
| `review` | apply checker | Fetches the branch, requires its tip to be a submission, one answer per submitted verdict (`confirmed`, `disputed`) with evidence, and an outcome that matches them (approved only with no dispute, rejected only with one). Writes the review as a commit with the submission's tree on the submission (`git commit-tree`), so it changes no file whatever the working tree holds, and pushes it. |
| `assign` | maker with children in the Run | A message to a direct child. |
| `ask`, `answer` | maker | A message to `parent` or a direct child in the Run. |
| `escalate` | maker | A message to the parent. Root's escalation has no receiver: it is a model defect and ends the Run without landing (§6.2). |

The message tools hand `agents.Message{Kind, From, To, Text}` (Package IDs, `""` for Root) to the engine's `Sink`; the engine's refusal returns to the agent. Reading the model needs no tool: `/work` holds `.markitect/`, and the image carries `markitect` for `check`, `refs`, `graph`, `impact --base base` and `lint`. Merging approved child commits is plain Git in the clone. The guarantees stay in the remote's hooks; the tools only give early, exact feedback.

## Claude Code

The adapter pins Claude Code `2.1.308` and keeps the launch contract that the first native qualification of V1's adapter established ([K1b](../../docs/validation/v1-k1b-claude-qualification-20261028.md)):

- One `claude --print` process per turn with stream-json in and out; the first turn creates the conversation, later ones `--resume` it with the session ID the first `system/init` reported.
- `--model`, `--effort`, `--system-prompt-file` and `--append-system-prompt-file` (the harness), `--permission-mode bypassPermissions --permission-prompts none` (the container is the sandbox), `--tools Bash,Read,Edit,Write,Glob,Grep` (no `Task`, so no native sub-agents), `--disable-slash-commands`, `--setting-sources user`, `--settings` turning off the twelve built-in plugins, and `--strict-mcp-config --mcp-config` naming only `markitect-agentd mcp`.
- The environment pins `CLAUDE_CONFIG_DIR` in the private home and turns off CLAUDE.md, memory, background tasks, built-in sub-agents, checkpoints, tool search, updates and nonessential traffic; it carries no credential and leaves subprocess scrubbing off, which the pinned CLI refuses without bubblewrap. `MCP_TOOL_TIMEOUT` lies a minute beyond the deadline.
- Every turn's `system/init` must report `apiKeySource` `none`, the pinned version, `bypassPermissions`, exactly the six built-in tools and the agent's Manager tools, the `markitect` server connected or pending, and no agents, skills, plugins or slash commands. A conversation message with a `parent_tool_use_id` (native sub-agent work), a control request, another session, or a conversation line before `system/init` ends the turn as `other`.
- **Login:** the host's `.credentials.json` with a `claudeAiOauth` account login. The copy keeps only `claudeAiOauth` without its refresh token, so the Manager can neither refresh nor rotate the login; an `sk-ant-api` key is refused, and `claude auth status` must report `claude.ai`.

## Codex

The adapter pins Codex `0.162.0` and runs `codex exec --json`, then `codex exec resume THREAD --json`, with the message on standard input. It passes the baseline arm's switches (`tools/comparison/runner.py`): apps, memories, `multi_agent`, `multi_agent_v2`, image generation, browser and computer use off, `agents.enabled=false`, the file login store, no project documents, `/work` untrusted; plus the `markitect` MCP server with a tool timeout a minute beyond the deadline, `--ignore-user-config`, `--ignore-rules` and `--dangerously-bypass-approvals-and-sandbox`, since Codex's own sandbox cannot start in an unprivileged container. A `collab` item (native multi-agent work), or a resumed turn that starts another thread, ends the turn as `other`.

**Login:** the host's `auth.json` with ChatGPT tokens. The copy keeps the ID token, access token and account and sets the refresh token to `""`: the pinned CLI refuses a login without the field (`missing field refresh_token`) and reports `Logged in using ChatGPT` with an empty one, both observed offline in the pinned image. An `OPENAI_API_KEY` is refused, the access token's `exp` claim gives the expiry, and `codex login status` must report the ChatGPT login.

## Default model and effort

Makers and checkers use the same model and effort, and the Run uses one model for all its agents, as the evaluation requires of both arms ([evaluation suite](../../docs/guides/evaluation-suite.md)). The defaults are:

| Provider | Model | Effort |
| --- | --- | --- |
| Claude Code | `claude-sonnet-5-5` | `high` |
| Codex | `gpt-6-sol` | `high` |

Why: quality comes first, and the comparison with one strong agent is fair only on the same model, which the evaluation sets per series (`EVALUATION_MODEL`, today `claude:claude-sonnet-5-5`). Sonnet at `high` is the setting Claude Managers passed natively in K1b, and the owner's rule is to fix failures in the concept and the tools, not with stronger agents. A fresh checker on the same model is independent by its context, not by its model; a stronger checker is not used because it would change what the comparison measures. The engine passes the model and effort to every agent's `Spec`; the full model ID, not an alias, is used so that each `system/init` reports exactly it.

## The Manager image

`packaging/runtime/Dockerfile` builds the image every agent runs in: Node 22 on Debian bookworm with Git, Python 3 and CA certificates, the pinned Codex and Claude Code CLIs, `markitect-agentd` and `markitect` built from the same source. `/home/agent` and `/work` belong to user `1000:1000`, so fresh volumes mounted there are writable. The build fails when a CLI is not the pinned version or lacks a flag its adapter passes, and `/opt/markitect/runtime.env` records both CLIs, `agentd` and `markitect`. A project builds its own image FROM it with its toolchain ([Go](../../packaging/runtime/examples/go/Dockerfile), [Python](../../packaging/runtime/examples/python/Dockerfile)). The image must not set `ANTHROPIC_*` or `CLAUDE_CODE_*` variables or ship `/home/agent/.claude` or `/etc/claude-code`.

## Tests and evidence

`internal/harness` tests render Root, a parent with children, a leaf, a checker with an inline and an oversized diff, verify, and an integrating parent over an in-memory project, and key every checklist part. The provider packages test the launch arguments and environment, the login copies and refusals, the login reports, and the readers over authored stream lines: answers with usage, every launch-contract breach and every failure class. `internal/agents` drives scripted provider processes through the real agent runtime with the host's Git standing in for the container's: the clone, a submission with trailers, an incomplete submission, a refused push that leaves the work in place, reviews that the contract refuses and one it pushes as an empty commit, verify verdicts, an integrating parent, messages along the tree and their refusals, Root's escalation, the deadline and cancellation, launch-contract breaches, a provider failure's class, a process that outlives its turn, and that no secret reaches the transcript.

These are source tests. The CLIs' stream shapes are authored from the documented interfaces and K1b's captures; Codex's `exec` events and its error texts have not been observed with a provider yet. A real Claude or Codex turn needs an owner-approved dispatch; that qualification is still open.

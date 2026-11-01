# Manager agents

Every working Manager of a [concept 2.0](../../docs/concepts/v2-concept.md) Run is two agents, a maker and a fresh checker (§6.2). This page is the canonical description of how one agent runs: its container, its login, its harness, the tools it answers through, the provider adapters and the default model. It is part B of step T4 of the [delivery plan](../../docs/implementation-plan.md#concept-20-build). The Run engine of part A starts agents through the [agent port](../agentport/README.md), which this package implements; the [Run remote](../runremote/README.md) they push to is step T3.

| Package | Role |
| --- | --- |
| `internal/harness` | Pure: renders what a maker or checker receives from the engine's start, the project at the Run's base and the submission, and the tool contract. |
| `internal/agents` | Implements `agentport.Agents`: one agent per start in its container, with login check, clone, provider turn, Manager tools and cleanup. Knows no provider. Also runs Markitect Desktop's [chat agents](#chat-agents). |
| `internal/agents/claude` | The Claude Code adapter: launch contract, access-only login copy, stream-json reader, failure classes. |
| `internal/agents/codex` | The Codex adapter: the same for `codex exec`. |
| `packaging/runtime` | The Manager image. |

## One start

`agents.New(Config)` takes what every agent of a Run shares: the container runtime (`EngineRuntime` of the container client), the `Provider` (`claude.Provider` or `codex.Provider`), the Manager image's ID, the limits, the network that reaches the provider with any hosts and name servers, the operator's login file, the model and effort, and an optional transcript directory. `Agents.Run(ctx, start, host)` then runs one start of the port and returns its `End` only once the agent has ended and what it created is gone. Each start is a fresh agent with its own container and conversation; what an earlier start did is on the Run's remote, and what happened since arrives in the start's `Inbox`.

A start does, in order:

1. Reads the login file and admits it through the provider: an API key, an expired access token and an access token that does not outlive the start's deadline by ten minutes are refused, since a CLI may try to refresh shortly before expiry and the access-only copy cannot.
2. Creates two fresh volumes, `NAME-home` at `/home/agent` and `NAME-work` at `/work`. They and the container carry the label `markitect.run=RUN`, so the Run remote's sweep finds them.
3. Starts the container from the Manager image: `markitect-agentd serve` as user `1000:1000`, read-only root, no capabilities, no new privileges, the configured CPU, memory, process and temporary-space limits. It joins the provider's network (`Config.Network`, a user-defined network: `markitect-provider` by default, the evaluation's egress-limited one in the evaluation) and the Run's internal network (`Start.Git.Network`); without a provider network it reaches the remote only. No folder of the user's is mounted.
4. Stores the agent's remote user and token in Git's credential store in the private home and clones the remote into `/work` at the Manager's branch. The token is the agent's own identity at the remote; the remote holds only its hash.
5. Loads the project at the Run's base commit from the clone (`git ls-tree` and `git archive` of `.markitect/`, as a commit holds only tracked files) and, for a checker in `apply`, the submission at the tip of the branch: its verdict trailers and its diff from the base. A project with compile errors starts no agent.
6. Renders the harness and installs the private home through agentd: the access-only login copy and, for Claude Code, the two system prompt files.
7. Runs the CLI's own login report (`claude auth status`, `codex login status`) and requires the account login.
8. Runs one provider turn whose message is the harness's task. The turn ends when the provider ends it, at the start's deadline or when `ctx` ends; agentd then stops the process group. A tool call still running is cut off and awaited, and a call queued by then is not served, so no host tool call of the turn happens after `Run` returns. A push the cut-off call had already sent may still be accepted by the remote once its hook finishes; the engine reads the remote after the start, so it sees such a push either way. A process still in the container 30 seconds later is an error.
9. Removes the login from the private home, ends the container and removes both volumes.

The start's deadline bounds every step, the clone and the login check included.

The `End` maps the turn: `finished`, `deadline`, `cancelled` or `failed`. `Agent` is `provider:model`, with the model the provider reported, or the configured one when it reports none (Codex), and `Usage` counts the input tokens (cache reads and writes included) and the output tokens. A turn that ends without the provider's total, at the deadline, because the Run ends it or because it breaks off, sets `Usage.Partial`: Claude Code's assistant messages give its input exactly and its output only as streamed at each message's start, so its count is a lower bound; Codex reports usage only when a turn completes, so such a turn counts nothing. A failed `End` carries its failure class (below) in `Class`, and `Detail` says what happened, for example `overloaded: API Error: 529`. `Cleanup` is empty only when the container and both volumes are gone.

With `Config.Transcripts` set, every start writes `RUN/MANAGER-ROLE-ROUND.jsonl` there: the provider's output lines as they came, so the evaluation's classifier reads them like a provider's own transcript, and the host's records (the turn, tool calls and replies, the outcome) as lines of type `markitect`. Every secret, the login's tokens and the remote token, is redacted before it is written, and from tool results, errors and the `End`.

The lower layer, `agents.Start` with a `Spec` and `Agent.Send`, is the same machinery for one agent; a conversation can take further turns, which resume it. An answered turn's `Outcome.Text` is the provider's last answer (Claude Code's result, Codex's last agent message), which a chat shows; a Manager answers through its tools.

## Composition

`markitect apply` and `verify` build the agents in `internal/bootstrap` from the environment:

| Variable | Meaning | Default |
| --- | --- | --- |
| `MARKITECT_MANAGER_IMAGE` | The Manager image, built from `packaging/runtime` with the project's toolchain | required |
| `MARKITECT_ENGINE` | The Docker CLI | `docker` on `PATH` |
| `MARKITECT_PROVIDER` | `claude` or `codex` | `claude` |
| `MARKITECT_LOGIN` | The account login file | `~/.claude/.credentials.json` or `~/.codex/auth.json` |
| `MARKITECT_MODEL`, `MARKITECT_EFFORT` | Model and effort of every maker and checker | `claude-sonnet-5-5` (Codex: `gpt-6-luna`), `high` |
| `MARKITECT_NETWORK` | The network that reaches the provider | `markitect-provider`, a user-defined bridge network the host creates on first use and keeps. Every agent also joins the Run's network, and the Docker CLI on Linux refuses to combine the engine's default `bridge` with a user-defined network |
| `MARKITECT_HOSTS`, `MARKITECT_DNS` | Extra host entries (`name:address`) and name servers, comma-separated | none |
| `MARKITECT_TRANSCRIPTS` | A folder for the agents' transcripts | none |

Each container gets 1 CPU, 2 GiB of memory, 256 processes and 256 MiB of temporary space. The Run's [remote](../runremote/README.md#the-container) container gets 2 CPUs, 4 GiB of memory and 1,024 processes: besides serving Git it runs the required checks, a whole build and test run of the project, one push at a time, and a check that exceeds them fails as the hook's first victim.

## Failure classes

A failed start carries a class, so that the evaluation's classifier can void the provider's failures and count everything else ([evaluation suite](../../docs/guides/evaluation-suite.md)):

| Class | Claude Code | Codex |
| --- | --- | --- |
| `capacity` | `overloaded`, HTTP 529 | "at capacity", "overloaded", 529 |
| `rate-limit` | `rate_limit`, `billing_error`, 429 | rate or usage limit, 429 |
| `server-error` | `server_error`, `api_error`, `unknown` (a dropped connection), 5xx | 5xx, a broken stream or connection |
| `login` | `authentication_failed`, 401, 403; a refused or expired login at start | 401, 403, an expired token; the same at start |
| `other` | everything else, including a breach of the launch contract | the same |

Claude Code's category is the one of the last failed request after its own retries: an `api_retry` line or the assistant message's `error`; a later assistant message without an error clears it. Codex's class is read from the error text of `turn.failed` or `error`.

## Harness

The harness renders three texts per start; prompts are a task and an answer format, and process lives in host state and the tools ([concept §12](../../docs/concepts/v2-concept.md#12-kept-from-concept-10)).

- **Base:** the role (maker or checker of which Manager, in which Run and mode) and the repository: the clone and the branches, the registry page and the paths the Package claims, the parent and the children that work in this Run, what the remote accepts from this agent, and the `markitect` commands that read the model in `/work`.
- **Context**, the model the Manager knows (§5): its own Package's Definitions, everything its parents define (Root first), the Definitions these link to in other Packages, the Kinds of all of them in full, and for each direct child a table of contents of the public Definitions in its subtree, with title and first sentence. Every entry names its file; nothing is truncated.
- **Task:** the engine's task text (`Start.Task`; without one, the role's default task), the checklist (item key, part, why), for a checker in `apply` the submitted verdicts and the diff from `base` (inline up to 256 KiB, listed by file above that), the messages of the start's `Inbox` with their IDs, and the answer format. A maker makes the repository conform to every item and submits one verdict per item; a parent with an empty checklist integrates its children's approved work and submits with no verdict. A checker in `verify` judges the base. Every start with checklist items, and every parent in `apply`, also reads what a verdict covers ([DEC-088](../../docs/concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)): the item's whole Definition with its examples and the Definitions it links to, so a contradiction among them is `model-problem`, but no review of the whole model; evidence for `conforms` and `changed` names how each example is met, and a checker disputes a verdict whose evidence does not; the code is the Manager's own Package's, and below Root in `apply` whether the rest of the program uses it yet is the parent's integration; an `apply` checker disputes a verdict that claims conformity where the model contradicts itself; a parent integrates its children's work by using it, and its checker disputes a file that builds a child's behavior a second time. The harness's `Version` names the rendered text and changes with it.

`harness.Items` turns an impact checklist into the port's items, keyed exactly as the Run's plan keys them (`runremote.Items`). For Claude Code, Base replaces the CLI's system prompt and Context is appended to it; for Codex both precede the task in the first message.

## Tools

The tools reach the host over MCP through `markitect-agentd mcp`. Every answer, a refusal included, is a tool result the agent reads; a refusal never ends the turn. A start offers the Git tool of its role and mode and the host tools its `Start.Tools` lists, no others.

| Tool | Who | Does |
| --- | --- | --- |
| `submit` | maker | Requires one verdict per checklist item (`conforms`, `changed`, `model-problem`) with evidence, commits the whole working tree with the [verdict trailers](../runremote/README.md#verdict-trailers) and pushes it to the Manager's branch. A push that does not go through returns Git's own words, for a refusal the hook's reason (paths, checks, trailers), and the commit is undone, so the work stays as it was. An accepted push is a submission, not an approval: the reply says so, that the host gives it to a fresh checker once the start has ended (or says in a message what it lacks), and that a rejection comes back as a `review` message in a later start's Inbox. |
| `submit` | verify checker | One verdict per item (`conforms`, `deviates`, `model-problem`) as a commit with the tip's tree on the tip, pushed. |
| `review` | apply checker | Fetches the branch, requires its tip to be a submission, one answer per submitted verdict (`confirmed`, `disputed`) with evidence, and an outcome that matches them: approved only with no dispute, rejected only with one, except that a submission without verdicts, an integrating parent's, may be rejected without one, as the hook allows. Writes the review as a commit with the submission's tree on the submission (`git commit-tree`), so it changes no file whatever the working tree holds, and pushes it. |
| `ask`, `escalate`, `answer`, `assign` | as the start allows | `Host.Ask`, `Escalate`, `Answer` (by the message ID from the Inbox) and `Assign` (to a working child). |
| `note`, `notes` | as the start allows | `Host.Note` records a Shared Note; `Host.Notes` reads them a page at a time. |

The host decides every message; its refusal reaches the agent as it is. The host's own Git in the clone runs no hooks, no file-system monitor and no replace refs. Other configuration the agent writes into its clone still applies, for example a clean filter or a URL rewrite; that is the agent's own container and user, and the remote's hooks still decide what is accepted. Reading the model needs no tool: `/work` holds `.markitect/`, and the image carries `markitect` for `check`, `refs`, `graph`, `impact --base base` and `lint`. Merging approved child commits is plain Git in the clone. The guarantees stay in the remote's hooks; the Git tools only give early, exact feedback.

## Claude Code

The adapter pins Claude Code `2.1.308` and keeps the launch contract that the first native qualification of V1's adapter established ([K1b](../../docs/validation/v1-k1b-claude-qualification-20261028.md)):

- One `claude --print` process per turn with stream-json in and out; the first turn creates the conversation, later ones `--resume` it with the session ID the first `system/init` reported.
- `--model`, `--effort`, `--system-prompt-file` and `--append-system-prompt-file` (the harness), `--permission-mode bypassPermissions --permission-prompts none` (the container is the sandbox), `--tools Bash,Read,Edit,Write,Glob,Grep` (no `Task`, so no native sub-agents), `--disable-slash-commands`, `--setting-sources user`, `--settings` turning off the twelve built-in plugins, and `--strict-mcp-config --mcp-config` naming only `markitect-agentd mcp`.
- The environment pins `CLAUDE_CONFIG_DIR` in the private home and turns off CLAUDE.md, memory, background tasks, built-in sub-agents, checkpoints, tool search, updates and nonessential traffic; it carries no credential and leaves subprocess scrubbing off, which the pinned CLI refuses without bubblewrap. `MCP_TOOL_TIMEOUT` lies a minute beyond the deadline.
- Every turn's `system/init` must report `apiKeySource` `none`, the pinned version, `bypassPermissions`, exactly the six built-in tools and the agent's Manager tools, the `markitect` server connected or pending, and no agents, skills, plugins or slash commands. A conversation message with a `parent_tool_use_id` (native sub-agent work), a control request, another session, or a conversation line before `system/init` ends the turn as `other`.
- **Login:** the host's `.credentials.json` with a `claudeAiOauth` account login. The copy keeps only `claudeAiOauth` without its refresh token, so the Manager can neither refresh nor rotate the login; an `sk-ant-api` key is refused, and `claude auth status` must report `claude.ai`.

## Codex

The adapter pins Codex `0.162.0` and runs `codex exec --json`, then `codex exec resume THREAD --json`, with the message on standard input. It passes the baseline arm's switches (`tools/comparison/runner.py`): apps, memories, `multi_agent`, `multi_agent_v2`, image generation, browser and computer use off, `agents.enabled=false`, the file login store, no project documents, `/work` untrusted; plus the `markitect` MCP server with a tool timeout a minute beyond the deadline, `--ignore-user-config`, `--ignore-rules` and `--dangerously-bypass-approvals-and-sandbox`, since Codex's own sandbox cannot start in an unprivileged container. A `collab` item (native multi-agent work), or a resumed turn that starts another thread, ends the turn as `other`.

**Login:** the host's `auth.json` with ChatGPT tokens. The copy keeps the ID token, access token and account and sets the refresh token to `""`: the pinned CLI refuses a login without the field (`missing field refresh_token`) and reports `Logged in using ChatGPT` with an empty one, both observed offline in the pinned image. An `OPENAI_API_KEY` is refused, the access token's `exp` claim gives the expiry, and `codex login status` must report the ChatGPT login; the pinned CLI prints that report on standard error (observed offline), so the adapter reads both streams. A turn that ends with `turn.completed` answered, whatever `error` events (such as reconnects) came before it.

## Default model and effort

Makers and checkers use the same model and effort, and the Run uses one model for all its agents, as the evaluation requires of both arms ([evaluation suite](../../docs/guides/evaluation-suite.md)). The defaults are:

| Provider | Model | Effort |
| --- | --- | --- |
| Claude Code | `claude-sonnet-5-5` | `high` |
| Codex | `gpt-6-luna` | `high` |

Why: quality comes first, and the comparison with one strong agent is fair only on the same model, which the evaluation sets per series (`EVALUATION_MODEL`, today `claude:claude-sonnet-5-5`). Codex defaults to Luna because the owner's Codex series run on Luna and `gpt-6-sol` runs only when a dispatch names it with `MARKITECT_MODEL` ([owner, 27 October](../../docs/concepts/sources/owner-morning-decisions-20261027.md)); both defaults are chosen the same way, here. Claude Code reports its model in `system/init`; Codex's exec events name none, so a Codex start's outcome names the model it was launched with. Sonnet at `high` is the setting Claude Managers passed natively in K1b, and the owner's rule is to fix failures in the concept and the tools, not with stronger agents. A fresh checker on the same model is independent by its context, not by its model; a stronger checker is not used because it would change what the comparison measures. The engine passes the model and effort to every agent's `Spec`; the full model ID, not an alias, is used so that each `system/init` reports exactly it.

## The Manager image

`packaging/runtime/Dockerfile` builds the image every agent runs in: Node 22 on Debian bookworm with Git, Python 3 and CA certificates, the pinned Codex and Claude Code CLIs, `markitect-agentd` and `markitect` built from the same source. `/home/agent` and `/work` belong to user `1000:1000`, so fresh volumes mounted there are writable. The build fails when a CLI is not the pinned version or lacks a flag its adapter passes, and `/opt/markitect/runtime.env` records both CLIs, `agentd` and `markitect`. A project builds its own image FROM it with its toolchain ([Go](../../packaging/runtime/examples/go/Dockerfile), [Python](../../packaging/runtime/examples/python/Dockerfile)). The image must not set `ANTHROPIC_*` or `CLAUDE_CODE_*` variables or ship `/home/agent/.claude` or `/etc/claude-code`.

## Chat agents

A chat agent is one conversation of [Markitect Desktop's chat](../adapters/desktop/README.md#the-chat) (concept 2.0 §8): the container, image, limits, provider network, login copy and provider adapter of a Manager, but no Run, no remote and no Manager tools. `agents.StartChat` with a `ChatSpec` starts it, and the Manager agent's own turn, login check and cleanup run it; the Manager agent itself is unchanged.

- **The copy.** `/work` becomes a repository of the owner's working tree as Git sees it, uncommitted changes included, sent in through `git fast-import`, so binary files and any file name arrive as they are; no folder of the user's is mounted, and the app copies regular files only, never what a symbolic link points to. Its ref `refs/markitect/start` holds that state. The open proposal of a conversation the agent takes over goes on top, as the checked-out branch `chat`.
- **What it knows.** `harness.Chat` renders its system text: what the chat is for, the rules the app enforces (only files below `.markitect/` are proposed; no commits; ask when something is open) and the modeling skill, or the Extract skill with the modeling skill. For an agent that takes over a conversation, the conversation so far precedes the owner's first message.
- **Turns.** Each message is one turn of the same provider conversation, bounded by the chat's turn limit (30 minutes in the app). While it runs, the provider's reader tells the chat each message the agent writes and what it is doing, in one line (`ProgressReader`), redacted like everything else that leaves the container; a Manager's turn does not ask for this. A turn whose login copy would expire on the way is refused as a login failure, so that a fresh agent with a fresh copy takes over.
- **The proposal.** `Changes` compares the copy's files as they are on disk, without the project's Git filters or line-end attributes, with `refs/markitect/start`: the files below `.markitect/` with their content, and the other paths; a nested repository is left out. `Settle` moves the start's files below `.markitect/` to the copy's after an accept, so that changes elsewhere stay changes, or puts the copy back after a reject. The answer of a turn is redacted like everything else that leaves the container.

`internal/bootstrap` runs the app's chat agents with the settings of `apply` and `verify` above, read at every start; a start first removes what an earlier agent of the same conversation left (label `markitect.chat`).

## Tests and evidence

`internal/harness` tests render Root, a parent with children, a leaf, a maker with an Inbox and an engine task, what a verdict covers for each role, a checker with an inline and an oversized diff, verify, and an integrating parent over an in-memory project, key every checklist part, and offer the tools of each start. The provider packages test the launch arguments and environment, the login copies and refusals, the login reports (Codex's on standard error), and the readers over authored stream lines: answers with usage, the partial usage of a turn without its end, every launch-contract breach and every failure class. `internal/agents` drives scripted provider processes through the real agent runtime with the host's Git standing in for the container's: the clone, the base project and the submission read from it, a submission with trailers, an incomplete submission, a refused push that leaves the work in place, reviews that the contract refuses and one it pushes as an empty commit, verify verdicts, an integrating parent and its checker's rejection, the host tools and their refusals, a tool the start does not offer, the deadline and cancellation, the deadline during a running tool call, a start past its deadline, launch-contract breaches, a provider failure's class, a process that outlives its turn, `Agents.Run`'s ends with usage, class and transcript, and that no secret reaches a transcript.

The chat agent's tests drive the same scripted runtime: the copy with a binary file, names with spaces and umlauts and the pending proposal on top; the chat's documents with no Manager tools and the conversation so far; a turn's progress and answer text, redacted, with the final answer heard once; the changes inside and outside `.markitect/`; settling after an accept and a reject; and a login that would expire within a turn, at start and later. A chat agent has not run against a real container yet: the engine stays free while the evaluations run.

`TestRealEngineManagersWorkThroughTheRunRemote` runs with `MARKITECT_TEST_ENGINE` (the Docker CLI's absolute path) and `MARKITECT_TEST_MANAGER_IMAGE` (an image built from `packaging/runtime`). It starts a real Run remote with its hooks and two starts through `Agents.Run` in real Manager containers, whose provider is a scripted MCP client of agentd in the image: the maker's push outside its area is refused by the hook and its work kept, its submission passes the hook and the required check, the checker's approval is accepted, the transcripts hold the tool calls and no token, and nothing of the Run remains. It passed natively on Windows with Docker Desktop on 28 October 2026.

`TestNativeClaudeMakerAndChecker` adds the real Claude Code CLI under a dispatch (`MARKITECT_TEST_CLAUDE_LOGIN`, `MARKITECT_TEST_EVIDENCE`): one maker and one checker against a real Run remote. Its [first run](../../docs/validation/t4b-claude-adapter-20261028.md) passed on 28 October 2026. Codex's `exec` events and error texts have not been observed with a provider yet; they are authored from the documented interface.

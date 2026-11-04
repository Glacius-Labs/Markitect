# Agent sessions and chats

`internal/agents` starts provider sessions on the user's own token: the unattended sessions of a [Command](../run/README.md) through the [agent port](../agentport/README.md) (`Sessions`), and the chats of [Markitect Desktop](../adapters/desktop/README.md#the-chat) (`Chat`). This page is the canonical description of both, of the provider adapters, of the environment that configures them and of what keeps a project's own agent files out of a launch. Concept 3.0 §9 and §11 give the design ([DEC-093](../../docs/concepts/register.md#dec-093-a-command-passes-the-users-own-token)).

| Package | Role |
| --- | --- |
| `internal/agents` | `Sessions` implements `agentport.Agents`. `Chat` runs one chat. Knows no provider. |
| `internal/agents/claude` | The Claude Code adapter: launch contract, stream-json reader, failure classes. |
| `internal/agents/codex` | The Codex adapter: the same for `codex exec`, without a token route. |
| `internal/harness` | What a chat agent starts with ([below](#harness)). |
| `internal/agentd` | The helper that runs a chat's provider inside its container. |
| `packaging/runtime` | The [image](#the-image). |

## Sessions

`agents.NewSessions(provider, model, effort, token, transcripts)` takes what every session of a Command shares. `Sessions.Run(ctx, shell, session)` then runs one session in the Command's container through the sandbox's `Shell` and returns its `End` once the session's processes have ended. Each session is one prompt: the engine writes the prompt, and a later session of the same role resumes an earlier one by its conversation ID.

A session does, in order:

1. Runs the CLI's own login report once per role and Command (`claude auth status`) and requires the account login. A refused login ends the session as `login`.
2. Starts the provider as the session's Linux user (`implementer`, or one of `reviewer-1` to `reviewer-8`) with its own home, in the folder the engine names, with the prompt on standard input.
3. Reads the provider's output line by line. The reader checks every launch ([below](#claude-code)) and records what the transcript shows: files read, shell commands, subagents, the models that answered and the tokens used.
4. Ends at the session's deadline, when the Command ends it or when the provider ends it. The sandbox then ends every process of that user.

The `End` says `finished`, `failed`, `deadline` or `cancelled`. `finished` says nothing about the work. `Agent` is `provider:model` as the provider reported it, `Models` lists every model that answered, subagents included, and `Usage` counts input tokens (cache reads and writes included) and output tokens. A session that ends without the provider's total sets `Usage.Partial`: the count is then a lower bound. A resumed session reports its conversation's whole total, and the Command counts only what it added. With `MARKITECT_TRANSCRIPTS` set, every session writes `RUN/ROLE-ROUND.jsonl` there, the provider's output lines as they came.

**The token.** The user makes a token with `claude setup-token` and keeps it in `MARKITECT_CLAUDE_TOKEN`, never in the CLI's own variable for the whole machine, which would outrank the login of the user's other sessions. A session gets it only in its environment, as the CLI's variable `CLAUDE_CODE_OAUTH_TOKEN`; it reaches the container by name, never on a command line. It is not written to disk, the image or a bundle, and the required checks never get it. The transcripts, the `End` and every error redact it, and landing refuses work that holds it. An agent, and any code it runs, can read the token from its environment, and it lasts a year with no documented way to revoke it.

## Composition

`markitect apply`, `markitect verify` and the app's chats build their agents in `internal/bootstrap` from the environment:

| Variable | Meaning | Default |
| --- | --- | --- |
| `MARKITECT_CLAUDE_TOKEN` | The user's token from `claude setup-token` | required |
| `MARKITECT_MANAGER_IMAGE` | The project's image, built `FROM` `packaging/runtime` with its toolchain | required; the native app builds one when unset |
| `MARKITECT_ENGINE` | The Docker CLI | `docker` on `PATH` |
| `MARKITECT_PROVIDER` | `claude` or `codex`; `codex` is refused until it has a token route | `claude` |
| `MARKITECT_MODEL`, `MARKITECT_EFFORT` | Model and effort of every session | `claude-sonnet-5-5`, `high` |
| `MARKITECT_NETWORK` | The user-defined network that reaches the provider | `markitect-provider`, created on first use and kept; `bridge`, `host`, `none` and `default` are refused |
| `MARKITECT_HOSTS`, `MARKITECT_DNS` | Extra host entries (`name:address`) and name servers, comma-separated | none |
| `MARKITECT_TRANSCRIPTS` | A folder for the sessions' transcripts | none |
| `MARKITECT_SESSION_BUDGET` | The most one session may spend, in the provider's own estimate | no limit |
| `MARKITECT_REVIEWER_SHELL` | `off` starts reviewers without a tool that runs commands, so that [test 1](../../docs/concepts/v3-concept.md#13-how-we-know-it-works) can compare the two. A long diff then can't be shown to them, since the prompt asks for it by a command | `on` |

A Command's container gets 4 CPUs, 8 GiB of memory, 2,048 processes and 1 GiB of temporary space: it holds the implementer and its subagents, the required checks and the reviewers. A chat's container gets 1 CPU, 2 GiB, 256 processes and 256 MiB. All sessions use one model and effort, which the evaluation requires of both arms ([evaluation suite](../../docs/guides/evaluation-suite.md)). Sonnet at `high` is the default; a stronger model is not the answer to a failure the concept or the tools should prevent.

## What keeps a project's own agent files out

A Command works on a project that may carry its own `.claude/settings.json`, `.mcp.json`, skills, agents, `CLAUDE.md` and hooks. None may reach a session, since an unattended session is told everything by its prompt:

- `--setting-sources user` reads neither the project's settings files nor its local ones, and `--strict-mcp-config` with no config connects no MCP server, so `.mcp.json` is ignored.
- `--settings` turns off the twelve built-in plugins and sets `disableAllHooks`, whatever a settings file says. `--disable-slash-commands` and the environment turn off `CLAUDE.md`, memory, checkpoints, tool search and updates.
- Each role's `HOME` and `CLAUDE_CONFIG_DIR` are its own folder in the container, made by the Command, so the user-level configuration is empty.
- The reviewer starts in the folder above its copy, so nothing the work committed there is discovered from where it starts.
- The reader checks the launch's own report (`system/init`) before it accepts any other line, and ends the session if the report names a skill, plugin, slash command, MCP server or MCP tool, a subagent the pinned CLI does not bring with it, an API key, another version, another permission mode or, on a resume, another session.

## Claude Code

The adapter pins Claude Code `2.1.308`; the image installs it and every launch must report it. It keeps the launch contract that the first native qualification established ([K1b](../../docs/validation/v1-k1b-claude-qualification-20261028.md)).

- **A session** is one `claude --print` process with stream-json in and out; a later session `--resume`s the conversation. It passes `--model`, `--effort`, `--permission-mode bypassPermissions --permission-prompts none` (the container is the sandbox) and `--tools Bash,Read,Edit,Write,Glob,Grep,Agent,TodoWrite`. The CLI's own subagents are allowed, all on the session's model: `CLAUDE_CODE_SUBAGENT_MODEL` with `CLAUDE_CODE_SUBAGENT_MODEL_FORCE=1`.
- **A chat turn** differs: the tools are `Bash,Read,Edit,Write,Glob,Grep` (no subagents), the harness arrives as system prompt files, `--mcp-config` names only `markitect-agentd mcp`, which offers no tool, and the environment also turns off background tasks and built-in subagents.
- **Login:** the token through `CLAUDE_CODE_OAUTH_TOKEN`. `claude auth status` must report `oauth_token` or `claude.ai`, and `system/init` must report `apiKeySource` `none`: an API key is refused.
- **Seen:** the reader records every `Read` file, every `Bash` command and every subagent start, subagents' own tool calls included, as `Seen`. The Command uses this to check the reviewer's pointers ([the answers](../run/README.md#the-reviewers-answers)).

## Codex

The adapter pins Codex `0.162.0` and keeps its `codex exec --json` launch and event reader, with the baseline arm's switches (`tools/comparison/runner.py`). It has no token route: OpenAI describes access tokens for ChatGPT Enterprise, and whether a personal plan gets one is not stated. `TokenVariable` is empty, so `MARKITECT_PROVIDER=codex` is refused, no Command starts a Codex session and no chat starts with Codex ([DEC-093](../../docs/concepts/register.md#dec-093-a-command-passes-the-users-own-token)). Its login check requires the ChatGPT account login and refuses an API key.

## Failure classes

A failed session carries a class, so that the evaluation's classifier can void the provider's failures and count everything else ([evaluation suite](../../docs/guides/evaluation-suite.md)):

| Class | Claude Code | Codex |
| --- | --- | --- |
| `capacity` | `overloaded`, HTTP 529 | "at capacity", "overloaded", 529 |
| `rate-limit` | `rate_limit`, `billing_error`, 429 | rate or usage limit, 429 |
| `server-error` | `server_error`, `api_error`, `unknown` (a dropped connection), 5xx | 5xx, a broken stream or connection |
| `login` | `authentication_failed`, 401, 403; a refused login at start | 401, 403, an expired token; the same at start |
| `other` | everything else, including a breach of the launch contract | the same |

Claude Code's category is the one of the last failed request after its own retries: an `api_retry` line or the assistant message's `error`; a later assistant message without an error clears it. Codex's class is read from the error text of `turn.failed` or `error`.

## Harness

`internal/harness` renders only what a chat agent starts with: `harness.Chat` gives a `Document` whose Base holds what the chat is for and the rules the app enforces, Context the modeling skill (or the Extract skill with it) and Task the conversation so far. It is pure. A Command's prompts are not a harness package: the default strategy's fixed part and role parts live in `internal/run` ([concept 3.0 §7](../../docs/concepts/v3-concept.md#7-what-the-agents-are-told)).

## The image

`packaging/runtime/Dockerfile` builds the image: Node 22 on Debian bookworm with Git, Python 3 and CA certificates, the pinned Codex and Claude Code CLIs, `markitect-agentd` and `markitect`. It makes the four users of a Command (`implementer`, `reviewer`, `checks` and `tools`, ids 1001 to 1004) and user `1000` for chats. The build fails when a CLI is not the pinned version or lacks a flag its adapter passes, and `/opt/markitect/runtime.env` records the versions. A project builds its own image `FROM` it with its toolchain ([Go](../../packaging/runtime/examples/go/Dockerfile), [Python](../../packaging/runtime/examples/python/Dockerfile)). The image must not set `ANTHROPIC_*` or `CLAUDE_CODE_*` variables or ship `/home/agent/.claude` or `/etc/claude-code`. See [packaging/runtime](../../packaging/runtime/README.md).

## Chat agents

A chat agent is one conversation of Markitect Desktop's chat (concept 2.0 §8): a container of the image, limits, the provider's network, the user's token and a provider adapter, with no Command. `agents.StartChat` with a `ChatSpec` starts it.

- **The copy.** `/work` becomes a repository of the owner's working tree as Git sees it, uncommitted changes included, sent in through `git fast-import`, so binary files and any file name arrive as they are; no folder of the user's is mounted, and the app copies regular files only, never what a symbolic link points to. Its ref `refs/markitect/start` holds that state. The open proposal of a conversation the agent takes over goes on top, as the checked-out branch `chat`.
- **What it knows.** `harness.Chat` renders its system text. For an agent that takes over a conversation, the conversation so far precedes the owner's first message.
- **Turns.** Each message is one turn of the same provider conversation, bounded by the chat's turn limit (30 minutes in the app). While it runs, the provider's reader tells the chat each message the agent writes and what it is doing, in one line (`ProgressReader`), redacted like everything else that leaves the container.
- **The proposal.** `Changes` compares the copy's files as they are on disk, without the project's Git filters or line-end attributes, with `refs/markitect/start`: the files below `.markitect/` with their content, and the other paths; a nested repository is left out. `Settle` moves the start's files below `.markitect/` to the copy's after an accept, so that changes elsewhere stay changes, or puts the copy back after a reject.

`internal/bootstrap` reads the settings above at every start, except that the native app [builds an image of its own](../adapters/desktop/README.md#the-native-app) when none is set; a start first removes what an earlier agent of the same conversation left (label `markitect.chat`).

## Tests and evidence

The Claude Code adapter's tests check the session launch and its flags, the settings that keep a project's files out, every refusal of the reader (API key, version, permission mode, skills, plugins, MCP servers and tools, an unknown subagent, a wrong resumed session) and the failure classes over authored stream lines. `internal/agents` drives scripted provider processes: the session's login check, redaction of the token, deadline and cancellation, and the chat's copy, turns, proposal and settling. A chat agent [first ran](../../docs/validation/desktop-first-real-chat-20261102.md) against a real container and Claude on 2 November 2026. A Command's session on the token has no real-provider evidence yet.

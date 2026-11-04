# The agent port

`internal/agentport` is the boundary between the [Command engine](../run/README.md) and the sessions it starts ([concept 3.0](../../docs/concepts/v3-concept.md) §6, §11). The engine starts one session with one prompt, as an implementer or a reviewer, in the Command's one container and waits until it ends. [`internal/agents`](../agents/README.md) implements `Agents`; the [sandbox](../run/README.md#the-sandbox) implements `Shell`, the only way a session reaches the container.

## A session

`Agents.Run(ctx, shell, Session)` runs one session and returns its `End` once it has ended. A `Session` holds:

| Field | Holds |
|---|---|
| `Run`, `Role`, `User`, `Round` | the Command, the part (`implementer` or `reviewer`), the Linux user it runs as (one per reviewer) and the count of this role's sessions, from 1 |
| `Dir` | the folder the session starts in |
| `Prompt` | the whole of what the session is told |
| `Conversation` | an earlier session of this role to continue; empty starts a fresh one |
| `Deadline` | when the session must have ended |

A session's work reaches the Command only through what it leaves in its folder and in the notes the Command reads after the session has ended. Nothing else it says counts.

`End.Reason` says only how the session ended: `finished`, `failed`, `deadline` or `cancelled`. `Finished` says nothing about the work. `End.Class` sorts a failed end by its cause at the provider (`capacity`, `rate-limit`, `server-error`, `login`, `other`) so that provider failures can be told apart. `End` also carries the conversation to resume, `provider:model` and every model that answered, `Usage` (`Partial` when the session reported no total), the session's closing words in `Text`, and `Seen`: what the transcript shows the session did, subagents included (files read, shell commands, subagents started, and whether the provider shows reads at all). A transcript shows some of it and never why, so `Seen` can only limit what a reviewer claims.

## The shell

`Shell.Exec` runs one command in the container as a named user, in a directory, with an environment and standard streams. `Secrets` are `NAME=value` additions whose value never appears on a command line. `Argv[0]` is an absolute path and the command runs without a shell. When the context ends, `Exec` first ends every process of that user.

# The agent port

`internal/agentport` is the boundary between the Run engine and the agents it starts ([concept 2.0](../../docs/concepts/v2-concept.md) §6.2, §12). The engine starts a maker or a checker for one Manager and waits until it ends; provider adapters implement `Agents`, and the engine implements `Host`, the tools an agent may call while it runs.

## A start

`Agents.Run(ctx, Start, Host)` starts one agent and returns its `End` once it has ended. It never returns while the agent may still change anything, and it is called concurrently for different Managers. A `Start` holds:

| Field | Holds |
|---|---|
| `Run`, `Mode` | the Run and whether it applies or verifies |
| `Role` | maker or checker |
| `Manager` | the Package ID (`.` for Root), its parent and its working children |
| `Round` | the count of this role's starts for this Manager, from 1 |
| `Task` | what this start must achieve, in the host's words |
| `Checklist` | every item the verdicts must cover: key, part (own, inherited, finding, stale), Definition and why |
| `Inbox` | the messages delivered with this start: assignments, answers, a checker's rejection |
| `Git` | the remote's URL and network, the Manager's branch, the base commit, and the agent's user and token |
| `Tools` | the host tools this start may call |
| `Deadline` | when the agent must have ended |

An agent's work reaches the engine only through Git: submissions, reviews and merges are commits on the Run's remote, in the trailer format of the Run remote. `End.Reason` says only how the agent ended (finished, failed, deadline, cancelled); the engine reads what it did from the remote. `End.Cleanup` is empty only when everything the start created is gone.

## Host tools

The tools carry messages along the tree and Shared Notes, never work, and no call waits for another agent: an answer arrives with a later start. A refused call returns an error for the agent that says why and what to do next.

| Tool | Does |
|---|---|
| `ask` | asks the parent a question |
| `escalate` | hands the parent a matter the Manager cannot settle within the model; at Root it ends the Run as a model defect |
| `answer` | answers a child's question or escalation |
| `assign` | sends a working child a task |
| `note`, `notes` | records a Shared Note; reads the Run's Notes in pages |

## Scripted agents

`internal/agentport/agentfake` implements `Agents` with scripted Steps per Manager and role, for tests. Its Steps clone the Run's remote with real Git, commit with verdict or review trailers, merge other branches, push and call the host's tools; a start without a Step fails, so a test sees every start it did not expect.

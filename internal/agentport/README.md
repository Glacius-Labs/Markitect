# The agent port

`internal/agentport` is the boundary between the Run engine and the agents it starts ([concept 2.0](../../docs/concepts/v2-concept.md) §6.2, §12). Every working Manager has one maker and one checker for the whole Run ([long-lived Managers](../../docs/design/long-lived-managers.md)); the engine gives one of them a turn and waits until the turn ends. Provider adapters implement `Agents`, and the engine implements `Host`, the tools an agent may call during a turn.

## A turn

`Agents.Run(ctx, Start, Host)` runs one turn of the agent the `Start` names by its Run, Manager and role, and returns its `End` once the turn has ended: the first turn starts the agent, a later one continues its conversation, and an agent that had to end meanwhile starts again from the Run's remote. It never returns while the agent may still change anything, it is called concurrently for different Managers, and never for one agent twice at once. `Agents.Close(ctx, run)` ends every agent of the Run that still lives, and returns what may remain. A `Start` holds:

| Field | Holds |
|---|---|
| `Run`, `Mode` | the Run and whether it applies or verifies |
| `Role` | maker or checker |
| `Manager` | the Package ID (`.` for Root), its parent and its working children |
| `Round` | the count of this role's turns for this Manager, from 1 |
| `Task` | what this turn must achieve, in the host's words |
| `Event` | only what brought a later turn about, such as a new submission with its checks' results or the children's approved work; a turn that continues an agent gets it with `Inbox` instead of `Task` and the checklist |
| `Checklist` | every item the verdicts must cover: key, part (own, inherited, finding, stale), Definition and why |
| `Inbox` | the messages delivered with this turn: assignments, answers, a checker's rejection |
| `Git` | the remote's URL and network, the Manager's branch, the base commit, and the agent's user and token |
| `Tools` | the host tools this turn may call |
| `Deadline` | when the turn must have ended |

An agent's work reaches the engine only through Git: submissions, reviews and merges are commits on the Run's remote, in the trailer format of the Run remote. `End.Reason` says only how the turn ended (finished, failed, deadline, cancelled), `End.Continued` whether it continued a living agent rather than starting one in a new container, and `End.Class` sorts a failed end by its cause at the provider (capacity, rate-limit, server-error, login, other), so that provider failures can be told apart; the engine reads what it did from the remote. `End.Cleanup` is empty unless an agent ended after the turn and what it created may remain.

## Host tools

The tools carry messages along the tree and Shared Notes, never work, and no call waits for another agent: an answer arrives with a later turn. A refused call returns an error for the agent that says why and what to do next.

| Tool | Does |
|---|---|
| `ask` | asks the parent a question |
| `escalate` | hands the parent a matter the Manager cannot settle within the model; at Root it ends the Run as a model defect |
| `answer` | answers a child's question or escalation |
| `assign` | sends a working child a task |
| `note`, `notes` | records a Shared Note; reads the Run's Notes in pages |

## Scripted agents

`internal/agentport/agentfake` implements `Agents` with scripted Steps per Manager and role, for tests: one Step per turn. Its Steps clone the Run's remote with real Git, commit with verdict or review trailers, merge other branches, push and call the host's tools; a turn without a Step fails, so a test sees every turn it did not expect. `Close` records the Runs whose agents the engine closed.

# Long-lived Managers

A proposal to change how a concept 2.0 Run starts its agents, built on a separate branch and compared with `main` before anything changes there. The owner asked for it on 30 October 2026: "Can we redesign it to that in a separate branch and then compare?" Nothing here is product behavior until the owner adopts it after the comparison; until then [concept 2.0](../concepts/v2-concept.md) holds.

## Why

Today every start of a maker or checker gets a new container, a new clone and one provider turn, and ends; a rejection or a child's approval leads to a new start that knows only what Git holds ([agents](../../internal/agents/README.md#one-agent)). In the [T7 pilot](../validation/t7-pilot-20261029.md)'s 2.0 runs:

- **Containers are cheap.** A Library fixed start used 13 to 14 agent containers, a Makerspace plant 22 to 44, each Run one remote. The required checks run inside the remote, so no helper container is created. From a container's start to its provider turn took 1 to 3 seconds, 2 to 5 % of the agent time.
- **Turns in a row and reading are not.** On the Library M3, about ten turns of 30 to 90 seconds ran one after the other: Root's maker and checker, the three children and their checkers, `lending/reminders`, `lending`'s second round and Root's second round. Each start read 80 to 370 k input tokens, cache reads included, to write 1 to 8 k, 1.8 M for the change. On the nine Library pairs K averaged 62 seconds and 0.28 M input tokens per launch.

So the cost lies in turns that each start from nothing and in turns spent on integration, not in the containers.

## Proposal

1. **One maker and one checker per working Manager for the whole Run.** Each starts in its own container when its Manager first works and keeps container and conversation until the Run ends. The host continues the conversation with each event: the assignment, a verdict, a child's approved work, an answer, a settlement. The agent fetches the new commits instead of cloning again. That is two containers per working Manager plus the remote.
2. **The checker is fresh per Run, no longer per round.** It still receives only the model, the submission, its diff and the required checks' results, never the maker's conversation, its answers or the Shared Notes. But it remembers its earlier verdicts and submissions, and may anchor on them and look only at what changed. So every round it re-checks every verdict in full, and the comparison tests whether its independence holds. Only commits count; files an agent did not commit never reach a checker or the Result.
3. **The host does plain integration.** A parent without own items in the change gets no maker turn to merge: the host merges its children's approved commits on the parent's branch, and the parent's checker reviews the merged state, which the required checks gate ([DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)). The hooks admit that push as the host's and check, as for any parent push, that the child commits are unchanged and approved. A merge conflict or a failing check gives the parent's maker a turn. Where a parent has own items, as Root has in every Library change, nothing changes, so this item is reported apart from item 1.
4. **Limits count turns.** Each agent gets as many turns as it has starts today, and a deadline per turn; two turns in a row that leave nothing behind stop the Run, as two such starts do today.
5. **Context and login have a fallback.** When an agent's conversation nears its context limit, the host ends it and starts that agent fresh from Git, as today. A container that lives for the Run gets its login copy renewed from the host before each turn.

**What it changes, if adopted:** concept 2.0 §6.2 (a fresh checker; makers and checkers per start), §6.3 (the parent merges; only a Manager pushes, to its own branch), §9 (the host's own Git work), and [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager) and [DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers). **Unchanged:** what the hooks refuse, parent-first, escalation and model defects, what a verdict covers ([DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)), the Result page and landing. No team inside a Manager: the tree is the team, and sub-agents would work outside the hooks and the Result.

## As built on the branch

`dev/long-lived-managers` builds the five items on the `main` commit `2007156d`, whose code equals the counted series' frozen commit. The canonical pages there describe it: [agents](../../internal/agents/README.md#one-agent), [agent port](../../internal/agentport/README.md#a-turn), [Run engine](../../internal/run/README.md#schedule) and [Run remote](../../internal/runremote/README.md#what-the-hooks-refuse) (rule 10). The choices the proposal left open:

1. **Turns.** The port's `Start` is one turn of an agent named by its Run, Manager and role; `Agents.Close` ends the Run's agents before the remote goes. The provider CLIs continue a conversation natively and already did so for a turn after the first: `claude --print --resume SESSION` and `codex exec resume THREAD`, each with the account login only. A later turn sends only what brought it about: the event in the host's words and the messages, never the first turn's task, the checklist or the whole diff again. A checker gets each checklist item's verdict with how it changed since the submission its previous turn received, and the diff from that submission. Only a turn that starts an agent, its first or one after it ended, gets the whole task. Before a later turn, the host renews the login copy, fetches the remote into the clone, moves a maker's branch forward to its remote tip (where a review or the host's integration moved it) and resets a checker's working tree to the submission. An agent continues only after a turn that finished; after any other turn it ends, and its next turn starts fresh from Git as today.
2. **The checker.** From its second turn, its task says that its earlier reviews do not carry over and that it re-checks every verdict in full. It still gets no Shared Notes and no message of its maker.
3. **The host's integration.** For a parent without items whose working children are all approved, the host merges each approval with Git's text merge, the last merge as the parent's submission without verdicts, and pushes it as user `host` through the remote's own hook (`markitect-remote integrate`). The hook holds it to a parent's push and to nothing of the host's own. Agents' pushes and the host's take turns through one lock. A conflict or a refusal gives the parent's maker a host message. The Result lists every integration under `integrations`, which is how item 3 is read apart.
4. **Limits.** `--starts`, `--rounds` and `--start-timeout` keep their names and values and count turns. Two turns in a row without progress stop the Run.
5. **Fallbacks.** An agent ends after a turn whose last request read 150,000 input tokens or more, cache reads included. Claude Code reports that size; Codex does not and compacts its own thread. The login copy is renewed through the container's shell, so agentd and the Manager image stay as on `main`, and both arms of the comparison run on one image.

The Result keeps the evaluation suite's contract: `starts` lists turns, each marked `continued` when it continued a living agent, so containers per launch are the starts not continued plus the remote.

## Comparison

- **Build** builds it on its own branch, `dev/long-lived-managers`. `main` and the counted series' frozen commit stay untouched.
- **Arms:** the 2.0 arm at one pinned `main` commit against the branch built on that commit, both on Claude Sonnet 5.5 with the same tasks, starts, deadlines and budgets. K is not needed; the pilot and the counted series compare with it.
- **Tasks:** the Library fixed starts M2 to M5 and both Library plants, the Makerspace W0 and the three Makerspace plants, two repetitions each, with the arm that runs first alternating per task. The plants test whether a remembering checker still stops what it should.
- **Read:**
  - first quality: every hidden case, landed changes, and the plants' stops, which must not be worse;
  - then cost, as a hypothesis: wall time, input and output tokens with cache reads apart, turns, containers and quota points per launch. A continued conversation re-sends its growing context, so input may not fall. The proposal is worth adopting if wall time or input tokens fall by at least a quarter at equal quality;
  - item 3's effect apart, on the tasks where a parent has no own items.

  These runs are evidence for the design, not counted data.
- **Engine and quota:** under the owner's standing run approval with a written dispatch, with the counted series' quiet-engine guard and relaunch rule, after the counted series or in a slot the owner clears, since both share the seven-day quota.
- **Then** the owner decides; if adopted, the sections above change, and the counted series' report states which design it measured.

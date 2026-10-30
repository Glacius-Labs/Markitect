# Long-lived Managers

A proposal to change how a concept 2.0 Run starts its agents, built on a separate branch and compared with `main` before anything changes there. The owner asked for it on 30 October 2026: "Can we redesign it to that in a separate branch and then compare?" Nothing here is product behavior until the owner adopts it after the comparison; until then [concept 2.0](../concepts/v2-concept.md) holds.

## Why

Today every start of a maker or checker gets a new container, a new clone and one provider turn, and ends; a rejection or a child's approval leads to a new start that knows only what Git holds ([agents](../../internal/agents/README.md#one-start)). In the [T7 pilot](../validation/t7-pilot-20261029.md)'s 2.0 runs:

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

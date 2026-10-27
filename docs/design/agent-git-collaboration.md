# Agents collaborate through Git

**Status: design agreed for a prototype, 27 October 2026.** This is the G0 design for [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review). The owner answered the questions in [section 11](#11-questions-for-the-owner) the same day. Nothing is built; the prototype in [section 13](#13-suggested-next-step) comes first. This is the Git variant (lane 3): its code lives on the branch `variant/git`, and Main keeps today's handoff and publication until the owner adopts it. An independent architecture review of the first draft is folded in; [section 12](#12-rejected-alternatives) lists what it changed.

## In short

The host starts a Run and stops it. In between, Managers work like a team of engineers. Each has its own container, its own clone, its own branch and its own harness, and they pull, commit and push to one Run remote. They also hand off, review, ask and answer through Git: messages are commits too.

One thing watches the remote: the **referee**. It plays the part that branch protection, required reviews and required checks play on a hosting platform. It never tells a Manager what to do, and it plans nothing. It only refuses a push that breaks a rule, runs the required checks, and keeps the score: who handed off what, who accepted what, how much budget is left. All of it runs against plain Git, with no GitHub or GitLab feature involved.

At stop, the host writes the commit that Verify approved onto the target branch, or changes nothing.

```text
host:  start ────────────────────────────────────────────── stop
         │ base, harnesses, first task                        │ publish one commit, or nothing
         ▼                                                    │
   Run remote (plain Git)  ◄─ every push is checked ─  referee (rules, checks, score)
     ▲          ▲           ▲                                 │
   Root      child A     child B   ...                        │
   own container, own clone, own branch, own harness          │
         ◄──────── the referee reports: done, blocked, out of budget or deadlock
```

## Owner goals

From the [verbatim source](../concepts/sources/git-remote-collaboration-20261027.md) and the owner's later messages the same day:

- **The host starts and stops.** It does not micromanage the Managers. The owner confirmed the target: "the host in Markitect of course shouldnt micromanage. Ideally he would just start and stop and thats it."
- **Managers work like engineers.** They pull, commit and push, and their managers integrate the work. They may also talk through Git.
- **Git, not a product.** The design works with any remote or with none.
- **One harness per Manager.** Each harness is tailored like the prompt. Local work is left alone as far as possible.
- **No owner review.** The verified result lands on a branch the owner names.

These rules stay ([DEC-078 Retained](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)):

- the model is authoritative;
- Runs are closed, and the complete hierarchy takes part (concept [§7.2](../concepts/v1-concept.md#72-complete-participation-in-every-mandate));
- only a fresh Verify approval of the exact candidate publishes;
- guarantees live in state and tool contracts, not in prompts;
- providers use account login only.

## 1. Host, referee and Managers

| Part | Does | Never does |
|---|---|---|
| **Host** | Start: create the Run remote and referee, write the base and the first task, generate the harnesses, start the Managers. Stop: collect evidence, choose the next Run, publish, clean up | Talk to Managers during the Run, continue them, commit or merge for them |
| **Referee** | Accept or refuse each push by the rules; run the required checks; keep the score; report when the Run is done or stuck | Assign work, plan, answer questions, edit code |
| **Manager** | Its area's work, in its own clone, with its own tools; delegate, review and integrate children | Write anyone else's refs |

The referee is the one place that interprets the rules. It replaces today's Run assignment machinery, but it is reactive: it only acts when a Manager pushes or waits. The host keeps its Run-time role to start and stop.

### Start

1. Choose the Run's base `B`. For the first Assessment and Implement, `B` is the target branch's tip `T`. For Verify and for a correction Implement, `B` is the candidate under review.
2. Start the Run remote with `B` as `mk/base`, and start the referee.
3. Create one credential per Manager for this Run remote only. Generate each Manager's harness from the model and start its container with its own volume. The container clones the remote and checks out its own branch. No volume is shared between Managers.
4. Write Root's task as the first message. Verify Root's task carries what it carries today: the diff against the command's bound input, the already-verified fact and the Implement Run's build outputs ([§19.4](../concepts/v1-concept.md#194-independence-of-verify)).
5. Seed `mk/notes` from the project's Shared Notes.

### Stop

The referee reports one of four outcomes, and each one maps to a concept §20.1 outcome:

| Referee reports | Run outcome |
|---|---|
| Root's final judgment is valid, and the whole hierarchy has handed off and been accepted | completed, with Root's judgment |
| Root reports a model blocker | completed, model blocker |
| Budget exhausted, deadline passed, or deadlock (every Manager waits and nobody owes an answer) | technically incomplete |
| A container or the referee failed | technically incomplete |

Then the host:
1. stops the Manager containers;
2. collects the transcripts, token counts, the referee's score and the Run remote as one bundle, and stores the Run record and reads it back ([§18.2](../concepts/v1-concept.md#182-no-live-state-in-the-result));
3. harvests the new Notes into the project's Shared Notes;
4. chooses the next Run in the fixed workflow (Assessment, Implement, Verify, correction Implement), or finishes;
5. publishes after a Verify approval (section 7);
6. removes containers, volumes and the Run remote.

## 2. The Run remote

The Run remote is a plain Git smart-HTTP server (`git http-backend`) in a container. Each Manager gets its own credential, so the server knows which Manager is pushing. Every push passes a pre-receive hook, which asks the referee. Managers reach the remote over the Run's network. The host reaches it over a local port, with no mount and no path translation.

A hosted remote is optional and is only an edge. The host fetches `T` from it at start and pushes the published commit at stop. Any Git URL works, and so does a local repository. Credentials for it live only in the host's Git credential helper. No Manager container holds a hosted-remote token. The owner allowed one, but the design does not need it.

**Refs** in one Run:

| Ref | Written by | Rule |
|---|---|---|
| `mk/base` | host, once | never moves |
| `mk/<manager>/work` | that Manager | forward only, descends from `mk/base`; frozen at the base in Assessment and Verify |
| `mk/<manager>/talk` | that Manager | forward only, each commit adds exactly one new message file and changes nothing else |
| `mk/notes` | any Manager | forward only, each commit adds new Note files and changes none |
| `mk/score/...` | referee | handoffs, reviews, check results, budget |

A Manager writes only its own refs. The referee holds a global lock while it decides a push, so a review cannot race a new handoff or a check result.

## 3. Talking through Git

A message is a commit on the sender's `talk` branch with one small file:

```yaml
to: catalog            # the direct parent or a direct child
kind: handoff          # task, question, answer, handoff, review, finding, judgment
commit: 3f2a…          # the exact code commit it is about
accept: true           # reviews only
findings: [...]        # each naming its Definition, as today
text: ...
```

- **No stale approval.** A review must name the child's latest handoff; the referee refuses a review of an older one.
- **Audit trail.** The whole conversation is ordinary Git history and ends up in the evidence bundle.
- **Org chart.** The referee refuses a message to anyone but the sender's direct parent or child, as today.

A Manager receives messages through its `wait` tool, which long-polls the referee: "anything new for me?" The referee therefore knows who is waiting. That gives deadlock detection as today: when every Manager waits and nobody owes an answer, the referee names the unanswered questions to the waiters and then reports the deadlock.

## 4. Own branch or shared branch

**Recommendation: one branch per Manager.** The parent's `work` branch is the integration branch of its subtree, like feature branches in a team.

Why:
- the referee knows who wrote what;
- two writers never race on one branch;
- a parent reviews exactly its child's diff.

A shared branch could come later for a deliberate pair, for example an implementer and a reviewer at a leaf ([ASM-004](../concepts/register.md#assumptions-and-hypotheses)). V1 does not need it.

## 5. Guarantees

| Guarantee | How the referee keeps it |
|---|---|
| The complete hierarchy takes part | Root's final judgment is refused until every Manager received a task, handed off, and the parent accepted that latest handoff |
| A parent reviews the actual handoff | Handoff and review name the same commit, and it is the child's latest handoff |
| No invalid handoff is accepted | Every handoff is checked first. An accepting review of a failed check is refused, and the parent sees the check with the handoff, as today ([§8.4](../concepts/v1-concept.md#84-faulty-handoffs-are-the-parents-responsibility-d11)) |
| Late findings count | A child's later reject of its own fixed handoff withdraws the parent's acceptance, and all ancestors up to Root become active again, as today ([§13.3](../concepts/v1-concept.md#133-decisive-counterexamples)) |
| Corrections start from the parent's current state | A reject carries the parent's current `work` tip. The child's next handoff must contain it, or it is refused with that reason |
| Areas and the registry | The check of each handoff, as today: structure, accounting and the declared checks placed in that Manager's area |
| Root's result is checked | Root's final judgment must name a commit with a passed complete check |
| A shared budget | The referee counts messages across the Run against the Run's allowance ([§20.2](../concepts/v1-concept.md#202-shared-finite-resources)) and stops the Run when it runs out |
| Runs are closed | Root has no parent, so no message can leave the Run |

What the referee does **not** promise: who typed which line inside a parent's integration. A parent can copy a child's files instead of merging them. The real guarantee is the one the parent's own handoff gets: it is checked, and its parent reviews it, as today ([§8.3](../concepts/v1-concept.md#83-handoff-review-and-integration)). The first draft's merge-provenance rule is dropped.

## 6. Each Manager's own harness

The harness is generated from the model for each Manager: instructions (role, org chart, context and answer format), tools, skills, its credential and the refs it may write. It is tailored the way the prompt is.

How fine the harness should be: three levels.

- **A. Raw Git only.** The prompt describes the conventions. This breaks §3.7: earlier runs show agents forgetting handoff steps and handing over the wrong commit.
- **B. Free local work plus team tools (recommended).** Inside its own clone, a Manager may do anything: Git, building, tests, editors. Every step that reaches another Manager goes through a few tools:

  | Tool | Who | What it does |
  |---|---|---|
  | `send` | everyone | post a task, question, answer, finding or Note to an allowed Manager |
  | `wait` | everyone | wait for what arrives: messages, and child handoffs with their check results |
  | `hand_off` | everyone | commit what is uncommitted, push `work` and post the handoff naming that commit; in Assessment and Verify, a judgment without code |
  | `check` | everyone | run the check of the current commit, as an optional self-check |
  | `review` | parents | accept or reject a child's latest handoff, with Findings |
  | `take` | parents | fetch an accepted child commit and merge it into `work` |
  | `sync` | children | merge the parent's current `work` into one's own |

  The tools are thin: Git plus one message, with the referee's refusal passed on word for word. A Manager that bypasses a tool with raw Git meets the same referee.
- **C. Tools only, no Git.** The owner leans against this, and it throws away what models do best.

Tailoring examples:
- leaves get no `review` or `take`;
- Verify Managers hand off judgments, never code;
- only Root's harness can post the Run's final judgment;
- a Package written in Go gets the Go skill and image.

**Keeping going.** A provider response can end without a handoff. The container's small supervisor (today `internal/agentd`) then asks the referee whether this Manager has an open task or new events. If so, it starts a new response that names them. It also commits and pushes the Manager's uncommitted work to a private `mk/<manager>/wip` ref at every response end, so work is never lost. That ref is evidence, never a handoff. The supervisor holds no rules: the referee decides what is open, and the supervisor only relays.

## 7. Target branch and publication

- The Run configuration names the target: a Git URL or a local path, plus a branch. The default is the remote's default branch.
- At publication the host writes **one commit** onto the target: parent `T`, tree = the approved candidate's tree, a fixed Markitect identity, and the Apply ID in the message. It updates the target only if it still points at `T` (compare-and-swap). The agents' working history stays in the evidence bundle and never reaches the owner's branch.
- If the target moved during the Apply, nothing changes and the Apply reports a conflict. A later option is a new Implement on the new tip.
- A local target branch that is checked out in the owner's working tree is updated the way Git's `receive.denyCurrentBranch=updateInstead` does it: only when that working tree is clean, otherwise nothing changes and the Apply reports a conflict.

## 8. Windows

During the Run, the host only uses the container engine and Git over a local port. It may therefore run natively on Windows again, and the runner image could go. The provider login file is still mounted until login happens inside the container. The referee needs the engine socket for the declared checks; it is trusted infrastructure like the host. The prototype must show this.

## 9. What changes in the code

**Removed or moved:**

| Today | Becomes |
|---|---|
| Run assignment state, `await`, question closing, judgment invalidation, shared allowance, deadlock detection | The referee, with the same rules, reacting to pushes and waits |
| The host continuing idle Managers | The per-container supervisor, which asks the referee |
| The Git workspace service, host-captured handoffs, per-parent write serialization, the shared Run volume | The Run remote, the agents' own clones and pushes |
| The runner's `.git` mount, bundle hand-back and result branch; working-tree publication | Removed; publication is one compare-and-swap commit |

**Stays:** the model compiler, harness generation, the checks (run by the referee), the provider adapters, the workflow sequence, Notes (seeded and harvested), the Run record and the publication gate.

**Concept passages that change:** [§8](../concepts/v1-concept.md#8-collaboration-within-a-run), [§9](../concepts/v1-concept.md#9-run), [§10.4](../concepts/v1-concept.md#104-apply-flow), [§14](../concepts/v1-concept.md#14-manager-tools-d13), [§17](../concepts/v1-concept.md#17-containers-git-working-copies-and-durable-candidates) including §17.5 ("A Git server platform is not intended"), [§21.2–§21.4](../concepts/v1-concept.md#21-controlled-publication), V15 and the glossary's "Publication", plus the [architecture](../architecture.md)'s "no Git server". This is a changed contract: the F2 evidence does not cover it, and G4 must qualify it.

## 10. Risks

- **The referee is new code with many rules.** It is the same rules as today in one place, but they now apply to Git pushes instead of tool calls.
- **Check latency.** A parent waits for its child's check before it can review, as today.
- **The supervisor sits in the agent's container.** The agent has a shell there and could disturb it. The supervisor is only a convenience: rules and budget sit in the referee, and the deadline and deadlock detection end a Run whose supervisor fails.
- **Evidence size.** The bundle holds every WIP push and message.

## 11. Questions for the owner

1. **A referee during the Run:** is a rule-keeper that refuses and keeps the score, but never directs, acceptable as "the host only starts and stops"? Without it, the Run loses complete participation, late findings, the shared budget and deadlock detection, or each Manager's tools would have to enforce them. **Answered 27 Oct: yes.** The owner accepted a referee that only refuses bad pushes, runs checks and keeps the score.
2. Harness level B: free local work and seven team tools? **Answered: yes.**
3. One branch per Manager in V1, with shared branches later? **Answered: yes.**
4. Publish one commit on top of `T` (clean owner history), rather than fast-forwarding to the agents' history? **Answered: one commit.**
5. If the target moved during the Apply, fail closed? **Answered: yes, fail closed;** no automatic new Implement.
6. A local target branch that is checked out: refuse, or update it when the tree is clean? **Decided by the design as Git's default:** update only when the tree is clean (section 7).
7. Build a prototype before any rebuild? It would pause the next evaluation suite. **Answered: prototype first.**

## 12. Rejected alternatives

- **Rules spread over a server hook, a check service, the supervisor and the tools** (first draft). Four interpreters of one message log, and the hook alone could not express complete participation or late findings. Replaced by one referee.
- **A merge-provenance rule in the hook** (first draft). It protects commit objects, not content: a parent can copy a child's files. Dropped for the existing guarantee that every handoff, including the parent's, is checked and reviewed.
- **Fast-forwarding the target to the agents' history** (first draft). It would import WIP commits, merges and agent identities into the owner's branch. Replaced by one commit with the approved tree.
- **The host copies Git objects between containers** (`exec` and `cp`, as today). It keeps the host in the middle of every handoff, which is the micromanagement the owner wants gone.
- **Agents push to GitHub with a token.** Allowed by the owner but not needed: GitHub cannot limit a token to one branch, and a hosted remote would no longer be optional.
- **`git daemon`.** It has no authentication, so ref ownership could not be enforced.

## 13. Suggested next step

Build a throwaway prototype, not product code:
- Root and two children on the Library fixture;
- a local Run remote with a minimal referee;
- the tools as scripts;
- Codex Luna.

Compare it with the current Library Apply on time, messages, tokens and the oracle. Only if it holds up, plan the rebuild as slices that replace G1–G4.

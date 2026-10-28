# The Run engine

`internal/run` runs one Run per change ([concept 2.0](../../docs/concepts/v2-concept.md) §6, §7) for `markitect apply` and `markitect verify`. It decides who works, starts a maker and a fresh checker per working Manager through the [agent port](../agentport/README.md), reads their handoffs from the Run's [remote](../runremote/README.md), writes the Result and lands it as in a merge queue. This page is the canonical description of the engine and of the Result page's layout, which §14 left to the build. It is step T4 of the [delivery plan](../../docs/implementation-plan.md#concept-20-build).

| Package | Role |
| --- | --- |
| `internal/run` | The engine: gate, impact, plan, schedule, host tools, Result and landing. It starts no process itself; its ports do. |
| `internal/adapters/rungit` | The host's own Git work in a private bare repository (below). |
| `internal/agentport` | How the engine starts agents; provider adapters implement it. |
| `internal/bootstrap` | Builds the engine for the command line with the host's Git, the Run remote in a container and the provider adapters. |

## A Run

1. **Start and base.** The Run starts from the tip of the target branch. The base is `--base`, or by default the last commit Markitect landed there: the newest commit that adds a Result page under `.markitect/results/`. With no landed Run yet, `--base` or `--full` is needed.
2. **Initial gate.** The project at the start commit is compiled; with a compile error the Run ends as `check-failed` and no agent and no remote start.
3. **Who works.** Impact from the base to the start commit (model and files, with `--findings` and `--full`) gives the working Managers and their checklists ([impact and checklists](../../docs/guides/model-language.md#impact-and-checklists)). Untouched Packages do not start and are listed as not taking part; with no working Manager the Run ends as `nothing-to-do`.
4. **Remote.** `runremote.PlanFor` builds the plan; the engine gives every agent its own user and random token, puts only the tokens' hashes in the plan and starts the remote from a bundle of the start commit.
5. **Schedule** (below), then the Result and landing.

## Schedule

One goroutine decides; every agent runs in its own goroutine, and the engine reads the remote (an export bundle imported into the host's repository) each time an agent ends. In `apply`:

- A Manager's maker starts once all its working children are approved. Leaves start at once, side by side.
- A maker's start ends with a submission at its branch tip, a message, or nothing. A submission goes to a fresh checker only when it contains every working child's latest approval; otherwise the maker gets a host message naming what to merge. A submission made before all children are approved is not reviewed.
- A checker's review commit on the submission approves or rejects it. A rejection goes back to the maker as a `review` message with every disputed item. An approval finishes the Manager. Whenever all of a parent's children are approved and their approvals differ from those its maker last got, the parent's maker gets them in a host message and merges them.
- An approved Manager that starts again, for example for an answer, and leaves its branch unchanged stays approved.
- An approved submission that holds a `model-problem` verdict escalates to the parent automatically; at Root it is a model defect.
- Root's checker is the final check: the Run's work is approved when Root is, no message waits and every Manager is approved.

In `verify` every working Manager's checker starts at once and pushes its verdicts as one commit without file changes; there are no makers and no merges.

Every start gets the task in the host's words, its checklist, its inbox, its Git access, its tools and its deadline. The harness says how to work; the engine says what to achieve (concept 1.0 §3.7).

### Messages

Messages run along the tree only and nobody waits for one: a start that sends a question ends, and the answer starts the asker again. A parent whose start ends with a question still open is reminded once; a Run where nobody can go on names the open questions. The host tools are `ask` and `escalate` (to the parent), `answer` (a child's question or escalation), `assign` (work for a working child, which then needs a new approval), and `note` and `notes`. Makers in `apply` get all that fit their place in the tree. A checker, the only role in `verify`, gets only `note`: it is fresh and sees the model, the repository at the submission and its diff, and nothing its maker wrote besides the commit, so it does not read the Shared Notes (concept 2.0 §6.2). A refused call says why and what to do next. No message reaches the owner during a Run: Root's escalation is a model defect, which ends the Run without landing and becomes a decision for the owner with the escalation's `Recommendation:`.

### Limits and honest stops

`Limits` bound the Run: a deadline for the whole Run, a number of agent starts that corrections share, a number of maker and of checker starts per Manager, and a deadline per start. The deadline reaches every agent start and every Git and remote operation through its context. A start that leaves nothing behind (no submission, review or message) gets one host message; two such starts in a row stop the Run as making no progress. Every stop ends as `incomplete` (or `model-defect`), cancels the running agents, waits until they have ended, reads the remote one last time and keeps Root's newest state on the Run branch. A stopped Run lands nothing; a checker approval that happened is still reported as that checker's approval, and approved work of other Managers that the Run branch does not hold is listed as not kept. Writing that Result and removing the remote run on their own short deadline after the Run's has passed, and anything that may remain is listed under cleanup.

### Shared Notes

Notes are side observations with their origin (Manager and role) and no authority. Makers write and read them; checkers only write them. A Note holds at most 16 KiB; `notes` reads pages of at most 100 Notes and 128 KiB, oldest first, with a cursor. Every Note appears on the Result page.

## The Result

The host writes the Result from commits, verdicts and checks, never an agent. `markitect apply --json` prints it as JSON, the evaluation suite's Result contract; otherwise it prints the Result page, which is also committed under `.markitect/results/<run>.md`.

| State | Outcome | Means |
| --- | --- | --- |
| `landed` | landed | the approved work is on the target branch |
| `nothing-to-do` | not-landed | the change reaches no Manager; nothing started |
| `check-failed` | not-landed | the initial gate failed; nothing started |
| `verified`, `findings` | not-landed | `verify` found every item conforming and the required checks passing, or Findings, owner decisions or failing required checks |
| `model-defect` | not-landed | Root ended the Run on a model defect |
| `kept` | not-landed | approved work that may not land; it stays on its Run branch |
| `incomplete` | not-landed | a budget, a deadline, no progress or a technical failure |

The page has, in this order: the state and why; the command, outcome, target, base, start, Run branch, times and tokens; the checklist, one row per item with its verdict, evidence, approving checker and the checker's own evidence; where the required checks ran and their results; decisions for the owner with recommendations; Findings; the Packages not taking part and why; every agent start with how it ended; every message; side observations (impact notes and Shared Notes); and what cleanup left. `verify --findings-out FILE` writes the Findings as a Findings page, the input of `apply --findings`. A `verify` Finding is a `deviates` verdict; its evidence may say `Observed:` and `Expected:`.

**Required checks.** The hooks compile every submission and run the checks on its full repository state, so in `apply` the Result names the checked push of Root's approved submission from the remote's decision log. When the target moved, the checks also run on the merged state before landing, and the Result names that run instead. In `verify` the hooks run the checks on every checker's verdicts, whose state is the start commit, and the Result names the push of Root's verdicts; failing checks make the Run's state `findings`, never `verified`. The page says what ran: passed, ran and did not pass, or did not run, with why.

## Landing

Landing works as a merge queue (concept 2.0 §6.5):

1. Root's approved commit must contain every working Manager's latest approval, no commit of the work, message or change, may hold one of the Run's Git tokens or touch a provider login file (`.credentials.json`, `auth.json`), since landing pushes the whole history, and it must compile; otherwise it is kept. The host compiles every state it lands (concept 2.0 §6.3).
2. The approved work is pushed to `markitect/runs/<run>` first, so nothing is lost if landing fails half-way.
3. If the target branch has not moved, the Result commit on top of the approved work lands as a fast-forward.
4. If it moved, it must still contain the start commit (a rewritten branch keeps the work), and the incoming commits must not touch the Run's impact: no changed or reached Definition of the Run, no file the Run reached or changed, no change to the manifest, the lock, the required checks or a working Manager's registry page, and a state that compiles. Then the host merges them; the merge must be clean, the merged state must compile and the required checks must pass on it, and the Result commit on top of the merge lands.
5. Every push to the target names the tip it expects. If the target moved again meanwhile, landing starts over, at most three times. A failed push is checked by reading the target again on a fresh deadline: it may have landed, and when the target cannot be read the Result says the landing state is uncertain.
6. A landed Run's branch is deleted. A Run that does not land keeps its Result commit on its Run branch, on top of Root's newest state (in `verify`, of the start commit), and says why.

`verify` never lands: it changes nothing, and its Result stays on its Run branch.

## Host Git

`internal/adapters/rungit` is the host's own Git work: it reads the target branch, makes the start bundle, imports the remote's export, reads commits, merges with `git merge-tree` and writes the Result commit through a private index, all in a private bare repository. The user's repository is only a Git remote to it, so Markitect never touches the user's working tree. The target is `--remote` and `--branch`: by default the current branch's upstream remote and branch, else `origin`, else the repository itself. The adapter never pushes to a branch that is checked out in a working tree of a local target, whatever `receive.denyCurrentBranch` says, so with no remote a Run keeps its work on its Run branch; landing is meant for a remote the user pulls from, like any colleague's push. Merges use Git's own text merge, whatever merge drivers attributes or configuration name.

## Not yet

- **Runs that overlap** take turns only through landing: a second Run whose impact overlaps an earlier landing is kept. A lock per target is a later step, as is deleting older Run branches that a later landing covers.

## Tests

`run_test.go` runs whole Runs with scripted agents (`agentfake`) that clone, commit, merge and push over real Git smart HTTP to the Run remote's real server and hooks (`remoteserver`, served in the test process), with the host's real Git: landing on an unmoved target, a checker's rejection and the maker's fix, an escalation settled by the parent, a model defect at Root (escalated, and as an approved `model-problem` verdict), an approved child `model-problem` that escalates, a question asked without waiting, an unanswered question and an ignored reminder, a parent's `assign`, merges up three levels, a parent that forgot a child's approval, landing on a moved target clean, overlapping, failing the checks and with a merged state that does not compile, a rewritten target, a landing push that failed after it took effect and one whose outcome cannot be read, a checked-out target branch even where Git would update its working tree, a token in the work's history, a login file added by a merge commit, `verify` without changes, with verdicts and checks, with failing required checks and with Findings that feed `apply --findings`, the budget with and without approved work, the deadline, no progress, the initial gate, the default base, refused names and limits, usage totals and the host tools' refusals, a checker's included. A target that moves between the fetch and the push is not covered. They are source evidence, not provider, container or native evidence.

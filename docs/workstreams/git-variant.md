# Git variant (lane 3)

In the Git variant, Managers work like engineers. Each has its own container, clone, branch and harness, and they collaborate through a Run-local Git remote. A passive referee at that remote refuses bad pushes, runs the checks and keeps the score. The host only starts and stops the Run and publishes one commit. The [G0 design](../design/agent-git-collaboration.md) owns the mechanics, and [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review) owns the decision. This page holds only the lane's scope, its state, the next steps and the open owner questions.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the "Lane 3: Git variant lead" session.

## Scope

- **Owns:**
  - DEC-078, the G0 design and the slices P and G1 to G4 in the [delivery plan](../implementation-plan.md#git-remote-collaboration-owner-direction-27-october).
  - The long-lived branch `variant/git`. It holds only variant code and merges `main` regularly.
  - The Git arm (G) of the evaluation suite.
- **Uses, does not own:**
  - The compiler, the model, the Libraries and the evaluation suite are shared. Their changes land on `main` through their own lanes.
  - The host variant's code on `main`: the variant reuses harness generation, checks, provider adapters and the Run record.
- **Elsewhere:** on `main`, today's handoff and publication stay until the owner adopts the variant.

## Decisions so far

- **Direction (27 October):** DEC-078. Agents collaborate through Git, and a verified result lands on the target branch without owner review.
- **G0 answers (27 October):**
  - a passive referee is fine;
  - harness level B: free local Git plus seven team tools;
  - one branch per Manager;
  - publish one commit with compare-and-swap, and fail closed if the target moved;
  - prototype first.

  The design records them in [section 11](../design/agent-git-collaboration.md#11-questions-for-the-owner).
- **Lane plan (27 October):** the variant gets a stable version on the Library task through the evaluation suite's arm interface, after the owner approves the prototype dispatch below. Both arms of a comparison run on Codex Luna.

## State

- **G0:** design on `main` through [PR #795](https://github.com/Glacius-Labs/Markitect/pull/795).
- **`variant/git`:** created from `main` at `eb1cf307`. No variant code yet.
- **Prototype:** dispatch prepared below, awaiting the owner's approval. Nothing is built or run.
- **Sub-sessions:** none.

## Prototype dispatch (awaiting owner approval)

The throwaway prototype of [design §13](../design/agent-git-collaboration.md#13-suggested-next-step). It answers one question: does the Git variant hold up against the current Library Apply on time, messages, tokens and the oracle?

**Source.** Scripts under `prototypes/g0/` on `variant/git`, built and reviewed before any run. They are throwaway, not product code:
- the Run remote: `git http-backend` in a container, one credential per Manager, and a pre-receive hook that asks the referee;
- the referee: one small process that holds the ref rules, complete participation, check results, the message count and deadlock detection;
- the seven team tools as scripts on each Manager's `PATH`;
- the per-container supervisor loop;
- the host's start and stop, which also writes the arm interface's output directory ([PR #793](https://github.com/Glacius-Labs/Markitect/pull/793)).

The harness text comes from the existing harness generation. Checks use the existing images.

**Stages.** Each stage starts only after the previous one passed:

| Stage | What runs | Provider |
|---|---|---|
| P0 | Container smoke: the remote, the referee and scripted fake Managers push, hand off, review, sync and publish. The referee's refusals are tested on purpose | none |
| P1 | Mechanism run: Root and two children (`catalog`, `members`) on a cut-down Library M1, one Implement Run. Checks the mechanics only; the oracle does not apply | Codex Luna |
| P2 | Comparison run: the full Library M1 tree (6 Managers) through Assessment, Implement and Verify, then publication to a local target. The oracle assesses the published commit | Codex Luna |

The baseline for P2 is the host variant (arm M) on the same Library task on Codex Luna, taken from the evaluation suite. If no such run exists yet, one M run on Luna is added to this dispatch.

**Limits.**
- Attempts: P1 at most 2, P2 at most 2, and an M baseline at most 1.
- Per Apply: a 90-minute deadline, 128 messages as in M, and an abort above 20 million tokens.
- No API key; the Codex account login only.
- Linux only, in WSL Ubuntu, with Docker ([DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution)).
- No hosted remote and no GitHub token.
- Runs start only on a quiet machine, announced to the other lanes, and only with Codex usage under 80 %.

**Abort.** Automatic on deadline, budget or deadlock, through the referee. Manual through a stop script that removes everything carrying the run's label.

**Cleanup.** The stop script removes the containers, the network, the volumes and the Run remote by label. The evidence stays in the run's directory: transcripts, token counts, the Run remote as a bundle, the referee's score and `run.json`. A later read checks that nothing with the label remains.

**Recorded result.** Every attempt is recorded, including failed ones, in a validation record on `variant/git`:
- the outcome;
- wall time;
- message and handoff counts;
- tokens;
- referee refusals;
- the oracle result;
- the comparison with M.

**Then.** If P2 holds up, the rebuild is planned as slices G1 to G4 on `variant/git`. If it does not, the record says why, and the owner decides.

## Open owner questions

1. **Approve the prototype dispatch?** That means P0 now, then P1 and P2 on Codex Luna within the limits above. The recommendation is yes.
2. **P1 with Root and two children?** Design §13 says "Root and two children", but the oracle needs the full Library tree. The recommendation is to keep P1 as that small mechanism run and to compare only in P2, with all 6 Managers.
3. **Windows host.** The design notes that the Git variant's host could run natively on Windows. That would revisit DEC-077. The recommendation is to decide only after P2, with the platform lane.

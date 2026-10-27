# Agentic commands (lane 9)

State page of the commands lane. It describes the agentic commands Markitect runs through the model-derived Manager hierarchy as **contracts**: what a command takes, what it returns and what it guarantees. How a variant realizes them belongs to the variant lanes: the host variant (today's V1), the Claude Code plugin variant and the Git variant ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)). The lane overview is `docs/workstreams.md`.

| | |
|---|---|
| **Status** | Stock-taking, 27 October 2026. Nothing is built in this lane until the owner says so. |
| **Scope** | Command contracts: purpose, inputs, outputs, guarantees, status; candidate commands; open questions. |
| **Not in scope** | How a variant realizes a command; the evaluation suite; model language, modeling skills and the compiler; continuous triggers; the Cockpit. Each has its own lane. |
| **Sources** | [V1 concept 1.0](../concepts/v1-concept.md) (§4, §7, §9–§12, §18–§21), the [concept register](../concepts/register.md), the [architecture](../architecture.md). This page links to them and does not change them. |

## Terms

Concept 1.0 separates three levels ([§4](../concepts/v1-concept.md#4-terms), [§10](../concepts/v1-concept.md#10-command-workflow-and-apply)):

- **Command:** the named operation a user starts. It binds inputs, runs the initial check, owns the overall budget, runs a workflow and is responsible for the overall result and any publication. Today: `apply`.
- **Workflow:** the order of Runs inside a command, with their handovers, repetitions, success and abort. It sees only closed Run results.
- **Mandate:** the task of one Run through the complete organization. V1 has exactly three: Assessment, Implement, Verify ([D16](../concepts/v1-concept.md#105-extensibility-d16)).

So a new command does not automatically need a new mandate. Several candidates below are a new workflow over the existing mandates.

## Rules from V1

These rules come from V1 Apply. They hold for every command that changes artifacts. Rule 1 holds for every command. How far rules 2, 4, 8 and 10 reach commands that change the model is open (questions 4 and 5).

1. **Complete tree.** Every Run uses the complete model-derived organization: Root and every Package Manager, including empty and intermediate ones. The mandate changes the task, not the tree ([§7.2](../concepts/v1-concept.md#72-complete-participation-in-every-mandate); owner, [23 October](../concepts/sources/v1-scope-clarification-20261023.md)).
2. **Model in charge.** The model is authoritative. Apply never repairs the model on the side; a material model blocker ends the command without publication ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)).
3. **One direction.** The owner prefers that no command changes both the model and the artifacts; Apply therefore changes only artifacts ([DEC-026](../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts)).
4. **Closed Runs.** No question leaves a Run to the user or the invoking AI. Questions travel only along the tree ([D10](../concepts/v1-concept.md#82-communication-along-the-hierarchy-d10)).
5. **Closed, validated results.** A Run returns a stored, readable result; one validation in the Run contract decides whether it is complete and bound ([D15](../concepts/v1-concept.md#103-validity-of-closed-results-d15)). A workflow builds only on such results.
6. **One budget.** The command sets one finite budget; follow-up Runs draw from it and never reset it ([§20.2](../concepts/v1-concept.md#202-shared-finite-resources)).
7. **Honest outcome.** Accepted, Reject, model blocker, failed and incomplete stay distinct; an agent's claim does not heal a technical error ([§20.1](../concepts/v1-concept.md#201-domain-and-technical-outcomes)).
8. **Findings and Notes.** Findings that bind the next Run name a Definition of the checked model; everything else is a Shared Note with no authority ([§18.3](../concepts/v1-concept.md#183-findings-as-an-actionable-handover), [§19](../concepts/v1-concept.md#19-shared-notes-and-independent-verification)).
9. **Checks at fixed points.** The same deterministic check runs before the workflow, at child handoffs, at Root completion and before publication ([§12](../concepts/v1-concept.md#12-checking-strategy-and-four-checkpoints)).
10. **Only verified content is published.** Publication takes exactly the candidate a fresh Verify approved ([§21.1](../concepts/v1-concept.md#211-approval-of-the-exact-candidate); DEC-078 keeps this while it changes how publication works).

## Current commands

### Apply (Model → Artifacts)

| | |
|---|---|
| **Purpose** | Bring the artifacts in line with the model, whether or not the model changed ([§10.4](../concepts/v1-concept.md#104-apply-flow)). |
| **Inputs** | A committed target repository with a valid model, registry and execution declarations; a runtime profile; a run configuration with the budget. |
| **Workflow** | Initial check → Assessment → Implement → Verify; a Verify Reject starts a fresh Implement and a fresh Verify with the bound Findings, under the same budget; final checks → publication. |
| **Outputs** | The published, verified change (today written into the working tree, or returned as a new branch through the Linux runner) and a report naming the stored evidence. Or no change, with the reason: invalid input, Assessment Reject, model blocker, budget stop, technical failure, or a conflict with user changes. |
| **Guarantees** | Rules 1–10. No owner commit today; DEC-078 moves publication to a target branch (Git variant). |
| **Status** | V1, implemented, natively qualified with Codex ([qualification record](../validation/v1-qualification-20261024.md), [technical acceptance](../validation/v1-technical-acceptance-20261026.md)), pending owner acceptance ([project workflow](../guides/project-workflow.md)). |

### Assessment (mandate)

| | |
|---|---|
| **Purpose** | Is the structurally valid model a sufficient basis for realization? Each Manager judges its area and interfaces ([§11.1](../concepts/v1-concept.md#111-assessment)). |
| **Inputs** | Checked model and project binding, fixed input candidate. |
| **Outputs** | Approve, or Reject / model blocker with Findings. Changes nothing; helper files stay private. |
| **Guarantees** | Missing or outdated artifacts are work for Implement, not a reason to reject. Approve does not mean the product is correct. |
| **Status** | V1, implemented. Runs only inside Apply today. |

### Implement (mandate)

| | |
|---|---|
| **Purpose** | Reconcile the realization with the model, the required checks and the bound Findings ([§11.2](../concepts/v1-concept.md#112-implement)). |
| **Inputs** | Checked model, input candidate, optional Initial Findings (each names a current Definition). |
| **Outputs** | One fixed, integrated candidate, submitted, never self-approved. The unchanged candidate is a valid result. |
| **Guarantees** | Changes stay within each Manager's area and the model's necessary consequences; no change only to show a diff ([§3.5](../concepts/v1-concept.md#35-focus-and-preservation)). Invalid child candidates reach the parent with their diagnostic and cannot be integrated ([D11](../concepts/v1-concept.md#84-faulty-handoffs-are-the-parents-responsibility-d11)). |
| **Status** | V1, implemented. Runs only inside Apply today. |

### Verify (mandate)

| | |
|---|---|
| **Purpose** | A fresh, independent check of exactly one candidate against the model ([§11.3](../concepts/v1-concept.md#113-verify)). |
| **Inputs** | Checked model, the candidate, the diff against the command's bound input and, when it holds, the host fact that this input was already verified ([DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model)). No Implement conversations. |
| **Outputs** | Approve; Reject with realization Findings; or model blocker. Changes nothing. |
| **Guarantees** | Independence from the implementers ([§19.4](../concepts/v1-concept.md#194-independence-of-verify)); unjustified changes against the bound input are rejected. |
| **Status** | V1, implemented. Runs only inside Apply today. |

### Model tools (not agentic)

`check`, `context` and `docs` start no Manager and are named here only for completeness. They belong to the compiler and model lanes ([project workflow](../guides/project-workflow.md)).

### Extract (Artifacts → Model)

| | |
|---|---|
| **Purpose** | Derive a model from existing artifacts ([DEC-057](../concepts/register.md#dec-057-three-fundamental-starting-situations), [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains)). |
| **Inputs** | Fixed artifacts and a selected scope; no model yet. |
| **Outputs** | A reviewed model proposal with registry, open questions and known realization deviations. The owner adopts it (DEC-057, [DEC-044](../concepts/register.md#dec-044-commands-reuse-the-manager-organization)). Artifacts stay unchanged. |
| **Guarantees (as accepted)** | Provisional tree from a structure phase, stable within a round; read-only Verify against the proposal; completion blocked by unresolved essential model questions, not by known artifact defects. |
| **Status** | Outside V1 ([§2.2](../concepts/v1-concept.md#22-not-commissioned)). A bounded version ran on the pre-V1 runtime on 19–20 October; the current source has none. It keeps owner questions (DEC-063), unlike rule 4 (question 4). |

## Candidate commands

Grouped by what they may change (rule 3). "Mandates" says whether the existing three suffice. [DEC-044](../concepts/register.md#dec-044-commands-reuse-the-manager-organization) already accepts, as a direction, that Verify, Cleanup, Bughunt, Refactor and Custom Commands run across the Manager tree; nothing of them is designed or built.

| Candidate | Changes | Purpose | Mandates | Sources |
|---|---|---|---|---|
| **Verify only** (conformance report) | Nothing | Run Assessment or Verify on the current state and report Findings, without Implement or publication. | Existing | DEC-044; [§9.1](../concepts/v1-concept.md#91-independent-call-boundary) already requires standalone Runs |
| **Impact check** | Nothing | For a proposed model change: which areas, artifacts and obligations it affects, and why, before Apply. | Probably a new read-only mandate | [DEC-018](../concepts/register.md#dec-018-the-knowledge-graph-explains-impact), [DEC-023](../concepts/register.md#dec-023-impact-marks-change-and-context-without-dropping-either), OQ-009 |
| **Explain** | Nothing | Answer a question about the project ("where and why does X happen?") from the Managers that own it. | New read-only mandate | Lane brief |
| **Operational check / doctor** | Nothing | Managers judge logs or runtime output of a running system against the model. | New read-only mandate | [IDEA-017](../concepts/register.md#idea-017-operational-check-through-the-managers) |
| **Refactor** | Artifacts | Restructure artifacts without changing behavior. | Implement + Verify, with a task input (question 2) | DEC-044 |
| **Cleanup / bug hunt** | Artifacts | Find and remove dead code, or find and fix defects against unchanged intent. | Verify finds, Implement fixes | DEC-044 |
| **Custom command** | Artifacts or nothing | A project-defined operation over the Manager tree. | Open | DEC-044; [§2.4](../concepts/v1-concept.md#24-later) (user-defined workflows later, no DSL in V1) |
| **Data migration** | Artifacts | Migrate existing runtime data after a model change. | Open | [§26.2](../concepts/v1-concept.md#262-change-from-five-to-three-loans) calls it a separate requirement |
| **Extract** | Model | See above. | New mandates (structure, model) | DEC-057, DEC-061 |
| **Optimize** | Model | Propose missing links and justified model improvements, without invention. | New mandate | DEC-044 |
| **Model migration** | Model | Meaning-preserving Package moves and merges with explicit identity migration. | New mandate | DEC-044 (model-migration refinement) |

Model-changing candidates touch the modeling-skills and continuous-change lanes; they are listed here because they would run through the Manager tree.

## What each variant must provide

A variant implements a command by providing, in its own way: the complete tree per Run (rule 1), the closed and validated Run result (rule 5), the shared budget (rule 6), the four checkpoints (rule 9) and, for artifact-changing commands, publication of exactly the verified candidate (rule 10). The workflow of a command is the same across variants, so the evaluation suite can compare variants on the same command.

## Outside input (not decided)

Two sessions analysed external articles on 27 October 2026 and sent these points. They are inputs for discussion, not decisions.

From Anthropic's article on [solving code review](https://claude.dev/blog/solving-code-review-at-anthropic/):

- **Convergence rules** against endless agent-on-agent review: cap the rounds, re-review only the new diff, never repeat a finding, and name the bug class so the author fixes the class. Reported effect: passing after one round rose from 65 % to 78 %, five or more rounds fell from 8.9 % to 1.5 %. Markitect's Verify instead re-checks the whole candidate fresh each time; how it converges is question 8.
- **Finding quality:** parallel finders, separate verifiers that vote, only agreed findings reported; style-only findings dropped; every finding has file and line evidence and a tier (Important, Nit, Pre-existing). Close to DEC-073's tiers.
- **Show, don't claim:** the realizing agent runs tests, build and lint and shows the output. Matches the optional `check` tool (D11).
- **Per-area review focus:** each area states what failure looks like there. In Markitect this could come from the Package's model, not a hand-kept file.
- **Approval binds one commit.** Already rule 10 and [D12](../concepts/v1-concept.md#132-the-judgment-belongs-to-the-assignment-d12).

From Traycer's [software factory guide](https://traycer.ai/software-factory):

- **Proof package as the result:** people review proof, not diffs: the command run and its output, a test that failed on the old code and passes now, and an explicit "not covered" list. A candidate for the Apply and Verify report (question 9).
- **Failing test first** for repairs: no fix without a test that fails on the old code.
- **Blind reviewer** that sees only the diff and the problem. Already Verify's independence (§19.4).
- **Risk tiers:** stricter review for high-risk areas, lighter for low-risk ones. Could inform how deep a parent reviews per Package.

## Open questions for the owner

1. **Command or mandate.** DEC-044 already accepts Verify, Cleanup, Bughunt, Refactor and Custom Commands as commands over the tree. Should they reuse Assessment, Implement and Verify where they can, and add a mandate only when none fits (D16)?
2. **Task input. Answered by the owner, 27 October 2026** (accepting the lane's recommendation in the lane chat): a command may take a task or question from the user, but the task never adds a binding obligation. The test is [DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)'s: if violating something makes an artifact wrong, it is model content.
   - Read-only commands (explain, impact check, Verify only) may take a free question; they change nothing.
   - Model-changing commands (Optimize, model migration) turn the task into a model proposal that the owner adopts.
   - Artifact-changing commands (refactor, cleanup) may take a task only for choices the model leaves free. A task that would make a result wrong when violated goes into the model first, then through Apply.
   - The task is a goal for Implement, not a rule for Verify: Verify still judges against the model and the diff against the bound input. Whether a refactor met its task is a matter for the Managers' review. A later Apply keeps the result, since conforming artifacts are preserved (§3.5).
   - A bug hunt needs no task: a bug is a model violation, so Verify finds it and Implement fixes it.
3. **Output of read-only commands.** A report only, or also Shared Notes, or bound Findings that a later Apply can take over?
4. **Questions in model-changing commands.** Extract kept owner questions (DEC-063); Runs are otherwise closed (rule 4, kept by DEC-078). Does Extract end with open questions in its proposal instead of asking mid-run?
5. **Adoption of other model proposals.** The owner adopts an Extract proposal. Is that the same for Optimize and model migration, and does DEC-078's "no owner review" stay limited to artifacts?
6. **Order.** Which candidates first? Proposal: "Verify only" (no new mandate, cheap to evaluate), then impact check, then Extract.
7. **Cost of the full tree.** Rule 1 stands. If the evaluation suite measures read-only commands as too costly, is that new evidence to revisit, or is cost accepted?
8. **Convergence.** Should Verify after a Reject re-check the whole candidate (today) or focus on the new diff and the earlier Findings, with a cap on correction rounds beyond the shared budget?
9. **Result form.** Should Apply and Verify end in a short proof package (commands and output, failing-then-passing tests, a "not covered" list) for the owner, instead of the diff?

## Log

- 27 October 2026: page created from concept 1.0, the register and the architecture; independent review applied; outside input from two article analyses added. No code.
- 27 October 2026: owner answered question 2 (task input).

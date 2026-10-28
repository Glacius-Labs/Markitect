# Agentic commands (lane 9)

State page of the commands lane. It describes the agentic commands Markitect runs through the model-derived Manager hierarchy as **contracts**: what a command takes, what it returns and what it guarantees. How a variant realizes them belongs to the variant lanes: the host variant (today's V1), the Claude Code plugin variant and the Git variant ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)). The lane overview is `docs/workstreams.md`.

| | |
|---|---|
| **Status** | In discussion with the owner, 27 October 2026. The frame below is agreed; nothing is built in this lane until the owner says so. |
| **Scope** | Command contracts: purpose, inputs, outputs, guarantees, status; candidate commands; open questions. |
| **Not in scope** | How a variant realizes a command; the evaluation suite; model language, modeling and Authoring, and the compiler; continuous triggers; the Cockpit. Each has its own lane. |
| **Sources** | [V1 concept 1.0](../concepts/v1-concept.md) (§4, §7, §9–§12, §18–§21), the [concept register](../concepts/register.md), the [architecture](../architecture.md). This page links to them and does not change them. |

## Frame and names

Agreed with the owner in the lane discussion on 27 October 2026. It refines the levels of concept 1.0 ([§4](../concepts/v1-concept.md#4-terms), [§10](../concepts/v1-concept.md#10-command-workflow-and-apply)) and replaces names that meant several things: "check" stood for the compiler command, a Manager tool and the project's test commands; "Verify" for a mandate and a command. The concept, register and code keep the old names until the owner decides to rename them there; the sections below this one still use them.

| Layer | What it does | Agents | Changes |
|---|---|---|---|
| **Analysis** | Compiler and accounting: deterministic facts about model, registry and files. Today `check`, `context`, `docs`. | No | Nothing |
| **Run** | One pass of one Run kind through the complete Manager tree; returns a closed Result. | Yes | Only its own candidate |
| **Command** | What a user starts. Binds inputs, owns the budget, uses Gates, chains Runs in a Workflow, publishes. | Only through Runs | Artifacts (by publication) or nothing |
| **Authoring** | Agent-assisted changes to the model: modeling, Extract, Optimize, model migration. The owner accepts the result. | Yes, with the owner | The model |

| Name | Meaning | Old name |
|---|---|---|
| **Analysis** | The deterministic layer; produces facts. | "check", compiler |
| **Gate** | Analysis that blocks at a fixed point: before the start, at child handoffs, at Root completion, before publication. | "checkpoint" |
| **Required checks** | The project's declared test and build commands. "Check" means only these. | same |
| **Run kind** | What a Run does, as a noun: **Assessment**, **Implementation**, **Verification**. | "mandate": Assessment, Implement, Verify |
| **Command** | A verb a user starts: `apply`, `verify`, `explain`. Built in or custom. | same |
| **Workflow** | The chain of Runs inside a Command. A custom Command is a Workflow the project defines. | same |
| **Result** | The one type every Run and every Command returns: status, verdict, Findings, Notes, optional candidate, evidence. | separate Run and Apply reports |
| **Authoring** | See above. Not called "skill", which clashes with Claude Code skills. | — |

Agreed with the frame:

- **A single Run can be started directly.** It is the smallest Command: a Workflow of one Run. There is one entry point and one Result type.
- **Custom Commands are the goal.** Built-in Commands are Workflows that ship with Markitect.
- **One Result type** for every Run and Command, so the evaluation suite and the Cockpit read any of them the same way.
- **Analysis supplies facts, Runs make judgments, Commands combine both.** Rule of thumb: an answer that follows mechanically from model and registry is Analysis; one that needs judgment is a Run. Where the line falls for a given capability is still open (Q2).
- **No Command changes the model.** Model changes are Authoring, which the owner accepts. Commands change artifacts or nothing.
- **New Commands reuse the existing Run kinds** where they fit; a new Run kind only when none fits (A1).

## Rules from V1

These rules come from V1 Apply. Since no Command changes the model (frame above), they hold for every Command; rule 10 applies where a Command publishes.

1. **Complete tree.** Every Run uses the complete model-derived organization: Root and every Package Manager, including empty and intermediate ones. The mandate changes the task, not the tree ([§7.2](../concepts/v1-concept.md#72-complete-participation-in-every-mandate); owner, [23 October](../concepts/sources/v1-scope-clarification-20261023.md)).
2. **Model in charge.** The model is authoritative. Apply never repairs the model on the side; a material model blocker ends the command without publication ([DEC-063](../concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)).
3. **One direction.** The owner prefers that no command changes both the model and the artifacts; Apply therefore changes only artifacts ([DEC-026](../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts)). The frame goes further: Commands never change the model; model changes are Authoring.
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

### Extract: Authoring, not a Command

Extract derives a model from existing artifacts ([DEC-057](../concepts/register.md#dec-057-three-fundamental-starting-situations), [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains)). The owner decided on 27 October 2026 that this lane ignores it: it changes the model, so it is Authoring and belongs with modeling in the modeling-skills lane. It is outside V1 ([§2.2](../concepts/v1-concept.md#22-not-commissioned)); a bounded version ran on the pre-V1 runtime on 19–20 October, and the current source has none.

## Candidate commands

Grouped by what they may change (rule 3). "Run kinds" says whether Assessment, Implementation and Verification suffice. [DEC-044](../concepts/register.md#dec-044-commands-reuse-the-manager-organization) already accepts, as a direction, that Verify, Cleanup, Bughunt, Refactor and Custom Commands run across the Manager tree; nothing of them is designed or built.

| Candidate | Changes | Purpose | Run kinds | Sources |
|---|---|---|---|---|
| **Verify only** (conformance report) | Nothing | Run an Assessment or Verification on the current state and report Findings, without Implementation or publication. | Existing | DEC-044; [§9.1](../concepts/v1-concept.md#91-independent-call-boundary) already requires standalone Runs |
| **Impact check** | Nothing | For a proposed model change: which areas, artifacts and obligations it affects, and why, before Apply. Analysis lists what is linked; a Run judges what else is affected. | Probably a new read-only Run kind | [DEC-018](../concepts/register.md#dec-018-the-knowledge-graph-explains-impact), [DEC-023](../concepts/register.md#dec-023-impact-marks-change-and-context-without-dropping-either), OQ-009 |
| **Explain** | Nothing | Answer a question about the project ("where and why does X happen?") from the Managers that own it. | New read-only Run kind | Lane brief |
| **Operational check / doctor** | Nothing | Managers judge logs or runtime output of a running system against the model. | New read-only Run kind | [IDEA-017](../concepts/register.md#idea-017-operational-check-through-the-managers) |
| **Refactor** | Artifacts | Restructure artifacts without changing behavior. | Implementation + Verification, with a task input (A2) | DEC-044 |
| **Cleanup / bug hunt** | Artifacts | Find and remove dead code, or find and fix defects against unchanged intent. | Verification finds, Implementation fixes | DEC-044 |
| **Custom Command** | Artifacts or nothing | A project-defined Workflow over the Manager tree. The goal (frame). | Existing ones, or new where none fits | DEC-044; [§2.4](../concepts/v1-concept.md#24-later) (user-defined workflows later, no DSL in V1) |
| **Data migration** | Artifacts | Migrate existing runtime data after a model change. | Open | [§26.2](../concepts/v1-concept.md#262-change-from-five-to-three-loans) calls it a separate requirement |

Extract, Optimize and model migration change the model, so they are Authoring and leave this list for the modeling-skills lane; DEC-044 and DEC-057 keep their accepted content.

## What each variant must provide

A variant implements a Command by providing, in its own way: the complete tree per Run (rule 1), the closed and validated Result (rule 5), the shared budget (rule 6), the four Gates (rule 9) and, for artifact-changing Commands, publication of exactly the verified candidate (rule 10). The Workflow of a Command is the same across variants, so the evaluation suite can compare variants on the same Command.

## Outside input (not decided)

Two sessions analysed external articles on 27 October 2026 and sent these points. They are inputs for discussion, not decisions.

From Anthropic's article on [solving code review](https://claude.dev/blog/solving-code-review-at-anthropic/):

- **Convergence rules** against endless agent-on-agent review: cap the rounds, re-review only the new diff, never repeat a finding, and name the bug class so the author fixes the class. Reported effect: passing after one round rose from 65 % to 78 %, five or more rounds fell from 8.9 % to 1.5 %. Markitect's Verification instead re-checks the whole candidate fresh each time; how it converges is Q5.
- **Finding quality:** parallel finders, separate verifiers that vote, only agreed findings reported; style-only findings dropped; every finding has file and line evidence and a tier (Important, Nit, Pre-existing). Close to DEC-073's tiers.
- **Show, don't claim:** the realizing agent runs tests, build and lint and shows the output. Matches the optional `check` tool (D11).
- **Per-area review focus:** each area states what failure looks like there. In Markitect this could come from the Package's model, not a hand-kept file.
- **Approval binds one commit.** Already rule 10 and [D12](../concepts/v1-concept.md#132-the-judgment-belongs-to-the-assignment-d12).

From Traycer's [software factory guide](https://traycer.ai/software-factory):

- **Proof package as the result:** people review proof, not diffs: the command run and its output, a test that failed on the old code and passes now, and an explicit "not covered" list. A candidate for the Result (Q6).
- **Failing test first** for repairs: no fix without a test that fails on the old code.
- **Blind reviewer** that sees only the diff and the problem. Already Verification's independence (§19.4).
- **Risk tiers:** stricter review for high-risk areas, lighter for low-risk ones. Could inform how deep a parent reviews per Package.

## Owner questions

### Answered

- **A1. Command or Run kind** (owner, 27 October 2026): new Commands reuse Assessment, Implementation and Verification where they fit; a new Run kind only when none fits (D16).
- **A2. Task input** (owner, 27 October 2026, accepting the lane's recommendation): a Command may take a task or question from the user, but the task never adds a binding obligation. The test is [DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)'s: if violating something makes an artifact wrong, it is model content.
  - Read-only Commands (explain, impact check, Verify only) may take a free question; they change nothing.
  - Artifact-changing Commands (refactor, cleanup) may take a task only for choices the model leaves free. A task that would make a result wrong when violated goes into the model first, then through Apply.
  - The task is a goal for Implementation, not a rule for Verification: Verification still judges against the model and the diff against the bound input. Whether a refactor met its task is a matter for the Managers' review. A later Apply keeps the result, since conforming artifacts are preserved (§3.5).
  - A bug hunt needs no task: a bug is a model violation, so Verification finds it and Implementation fixes it.
- **A3. Extract and other model changes** (owner, 27 October 2026): this lane ignores Extract; model changes are Authoring, not Commands. The earlier questions on Extract's owner questions and on who adopts model proposals leave this lane with it.
- **A4. Frame** (owner, 27 October 2026): the frame and names above, single Runs startable directly, Custom Commands as the goal, one Result type.

- **A5. Result of read-only Commands and hand-over to Apply** (owner, 28 October 2026, accepting the lane's recommendation; was Q1):
  - A read-only Command returns the one Result type without a candidate: status (completed, failed, incomplete); verdict (Approve or Reject for Verify only, "answered" for explain and impact); Findings, each naming a Definition with what was observed, what is expected and where to correct it; the answer (prose for explain, affected areas for impact); the IDs of the Shared Notes it appended; pointers to the stored Run records. Notes stay in the Notes store and keep no authority.
  - Apply takes Findings over only when the user passes a Result in explicitly, for example `apply --findings RESULT`. The host checks that the Result is stored and closed (the one validation, D15) and that each Finding's Definition exists in the current model; a Finding whose Definition is gone becomes a Note (DEC-073). If the artifacts changed since, the Findings still enter, and Implementation judges whether each still applies.
  - A Verify-only approval of exactly the current artifacts under the same model, execution configuration and dependencies can serve a later Apply as the "already verified" fact (DEC-073).
  - Nothing is taken over automatically (§19.2). Automatic hand-over, such as a nightly Verify feeding Apply, is a trigger and belongs to the continuous-change lane.
- **A6. Workflow form** (owner, 28 October 2026, accepting the lane's recommendation; was Q3): both, with a small fixed vocabulary.
  - A few building blocks live in code: run a Run kind, branch on its verdict, go back with Findings, publish. A Command is a short declaration that combines them. Built-in Commands such as `apply` and `verify` are declared the same way as custom ones. Apply, roughly:

    ```yaml
    apply:
      - assessment:     { on reject: stop }
      - implementation
      - verification:   { on reject: back to implementation with findings }
      - publish
    ```

  - Limits that keep it from becoming a language (concept §2.4, no DSL): only closed Results pass between steps, with no variables; the only branches are the verdicts (Approve, Reject, model blocker); loops are bounded by the shared budget; a Command may take the user's task as input (A2); Analysis checks a declaration before anything starts.
  - Declarations live in the execution configuration, not in the model: they are execution choices, not intent ([DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)).
  - Projects may define their own Run kinds, for example a security review as a Verification with its own focus, as they define their own Kinds in the model. A Run kind is then task text, answer format and Gates, and the host keeps it generic. Agreed in principle; the details are open (Q8).

### Open

- **Q1.** Answered as A5.
- **Q2. Analysis or Run.** Where the line falls for a given capability, for example impact. Rule of thumb in the frame.
- **Q3.** Answered as A6.
- **Q4. Order.** Which Commands first? Proposal: Verify only (no new Run kind, cheap to evaluate), then impact check.
- **Q5. Convergence.** Should a Verification after a Reject re-check the whole candidate (today) or focus on the new diff and the earlier Findings, with a cap on correction rounds beyond the shared budget?
- **Q6. Result form.** Should the Result end in a short proof package (commands and output, failing-then-passing tests, a "not covered" list) for the owner, instead of the diff?
- **Q7. Cost of the full tree.** Rule 1 stands. If the evaluation suite measures read-only Commands as too costly, is that new evidence to revisit, or is the cost accepted?
- **Q8. Project-defined Run kinds.** What a project may set for its own Run kind (task text, answer format, Gates, allowed verdicts), whether it may change artifacts, and how it stays generic in the host.

## Log

- 27 October 2026: page created from concept 1.0, the register and the architecture; independent review applied; outside input from two article analyses added. No code.
- 27 October 2026: owner answered the task-input question (A2).
- 27 October 2026: owner agreed the frame and names, reuse of Run kinds and that Extract is Authoring (A1, A3, A4); questions renumbered.
- 28 October 2026: owner answered Q1 (A5: Result of read-only Commands, hand-over to Apply).
- 28 October 2026: owner answered Q3 (A6: Workflow form, project-defined Run kinds in principle); Q8 added.

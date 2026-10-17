# Commands over the Manager organization

Status: accepted direction, 17 October 2026 ([DEC-044](../../concepts/register.md#dec-044-commands-reuse-the-manager-organization), [owner discussion](../../concepts/sources/commands-extract-optimize-20261017.md)). This document owns the later command semantics and the Extract workflow. It records no executable command registration, model-writing runtime or completed Verify capability. The [current operations guide](../../project-operations.md) owns availability. Apply remains the first delivery target; this documentation adds no implementation queue and does not change the frozen Apply core.

## One organization and different instructions

Use the existing Manager organization, compiled context, independent review, findings, correction and parent integration. The operation supplies one static instruction, propagated unchanged through the tree. Each Manager concentrates on its own responsibility because it receives its own model and artifact context. Parents integrate their own work and direct children's results. They do not rewrite the operation for children or choose which required Manager participates.

An instruction describes the objective; the Orchestrator enforces participation, allowed effects, budgets, required review and source/candidate bindings. A prompt alone grants no write authority. Model changes and artifact changes stay separate ([DEC-026](../../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts)); read-only operations may record reports and findings. Configuration and custom instructions cannot introduce binding project requirements outside the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)).

The names below express user goals, not a new CLI schema. Reuse the process with an appropriate instruction and change target rather than inventing an engine for every verb. Exact registration, command arguments and draft-model execution bindings remain implementation design, within [DEC-017](../../concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table).

| Operation | Change target | Instruction and result |
| --- | --- | --- |
| Apply | Artifacts | Reconcile the complete responsibility with the current canonical model, including unchanged intent. Correct deviations or return a justified reviewed no-change result. Valid realization can stay as it is. |
| Cleanup | Artifacts | Look for useful cleanup in each area while preserving the applicable requirements and permitted freedom. Deliver a reviewed improvement or justified no-op. |
| Bughunt | Artifacts | Search for bugs against the model and fix them through review and integration. Never change the model to excuse faulty artifacts. |
| Refactor | Artifacts | Look through the tree for better structure within the existing obligations. Preserve behavior and contracts. |
| Verify | None | Assess whether the existing stand conforms to the bound canonical model, and explain why it does not or cannot yet be judged. Produce findings and evidence; make no repairs. |
| Custom | Explicitly none, artifacts or model | Propagate the supplied prompt through the tree and perform it in each responsibility. The target and authority are explicit before work; free text cannot broaden them. |
| Extract | Model candidate and proposed artifact accounting | Describe the existing repository, review and integrate the model, then use read-only Verify findings to refine it. Existing realizations stay unchanged. See the workflow below. |
| Optimize | Model | Examine existing Definitions and connections for missing links and justified improvements. Preserve intended meaning and avoid unsupported additions. See the boundaries below. |

Cleanup and Refactor explicitly seek improvements in an already conforming stand. Apply need not rewrite that stand to an Agent's taste. Bughunt puts more attention on error cases. These different search objectives do not require different review or integration institutions. An improvement preference is not a Verify failure unless a binding requirement is violated.

The earlier north-star description also allowed an explicitly model-only Cleanup or Refactor. Whether those names remain available for model work, as aliases of Optimize or as explicit Custom commands, is open naming design. The table above describes the owner's current artifact-work examples; Optimize owns model improvement. No variant combines model and artifact changes.

## Verify only assesses conformity

Every Manager assesses its responsibility; parents also assess the interaction of their children. The result identifies the exact model and repository stand, the examined scope and evidence, and one of these semantic outcomes:

- Conforming, with a supported explanation.
- Nonconforming, with the concrete deviation, applicable Definition and reason.
- Insufficiently assessable, with the missing evidence or unresolved question.

These are meanings, not selected wire-enum names. Missing evidence does not count as conformity. Verify changes neither the model nor its realizations. A repair is a separate artifact operation; a model correction belongs to a model operation. Earlier findings remain traceable across those changes.

## Extract the existing stand

Extract initially describes what exists. It does not use adoption as an opportunity to improve product behavior or repair code. Code, tests, interfaces, documentation, configuration, CI and recorded decisions are evidence. An observed behavior is not automatically intended behavior. When sources conflict and existing authority does not settle the intended rule, present a question with both sources instead of silently choosing one. When an authoritative requirement is already clear, retain it and record violating code, tests or documentation as a realization deviation; do not require the owner to decide the same intent again.

### Build the first candidate

1. Fix the repository revision and selected scope. Keep exclusions and unknown areas visible. Existing code and other realizations remain the observation basis throughout the refinement loop.
2. Inspect the structure and propose initial Packages and responsibility boundaries. This first discovery supplies the provisional hierarchy needed for Manager work; no existing canonical model is assumed. Folder names are clues, not automatically the right model structure.
3. Each Manager examines its area's actual sources and drafts Definitions for the relevant concepts, behavior, rules, interfaces, architecture and dependencies. Keep a source basis for derived statements and explicit questions for contradictions. A Definition need not mirror a source file or every private implementation detail.
4. Propose ownership and Definition links for existing artifacts, using the registry and explicit accounting rather than a second intended model. A file assignment alone does not establish that its behavior has been described adequately.

The model remains a candidate until adopted through the project's decision process. Source revisions and evidence do not themselves confer authority. Imported vocabulary is read-only; it is not silently rewritten to make the extracted project compile.

### Model adoption without turning defects into rules

The owner accepted this workflow and the completion clarification below in the [Usage, Skills, UX continuation](../../concepts/sources/commands-extract-optimize-20261017.md#model-adoption-with-known-realization-deviations). The outer coding agent can prepare the model proposal through `markitect-design`; the later Extract runtime reuses the model-authoring and review capabilities described here. Neither path assumes that the existing realization is correct. The [Init flow](../project-capability-skills-20261009.md#existing-project-init-flow) owns the separate technical setup.

The assignment may say: "Prepare a model proposal from code, tests and documentation. Show contradictions and unclear intent. Do not change the application." Fix its source revision and scope, identify existing authoritative requirements and distinguish observations, interpretations, proposed intent and unresolved questions. There is no universal ranking in which tests, code or a README always win. The owner's designation of a current requirement or an applicable recorded decision has a different authority from an undated implementation comment.

This library example is hypothetical evidence for the design distinction, not an observed defect in the canonical Library example or a completed product run:

| Source observation | Model-authoring response |
| --- | --- |
| A current authoritative requirement permits at most five open loans. Code checks `count > 5` before adding a loan and therefore admits a sixth. | Keep the five-loan requirement in the candidate and record the suspected off-by-one realization defect. Do not make six the intended limit. |
| Tests demonstrate successful borrowing up to five but never test refusal of a sixth. | Record the limited evidence scope and assess the missing boundary case. Passing these tests does not establish complete enforcement. |
| A current authoritative requirement permits suspended members to return books, but a test expects refusal. | Review the test as a potentially faulty realization; a passing assertion is not authority to forbid returns. |
| Documentation says 14 days, another requirement version and tests say 21, and their current authority is unresolved. | Present the concrete source-bound choice with a justified recommendation. Do not fill in either value merely to complete the candidate. |
| The implementation uses an internal dictionary. | Do not require a dictionary in future implementations unless an actual obligation makes that choice binding. |

The resulting proposal can state a maximum of five after successful borrowing and continued return during suspension, with their source bases. The loan period remains a question until resolved. Existing behavior and discrepancies stay retrievable as findings beside the proposal, rather than becoming binding rules by extraction. The agent groups consequential questions; clear authorized requirements do not need individual reapproval, and recording a possible defect is not proof of its diagnosis.

Independent model review asks whether sources support each consequential statement, whether important behavior could change without violating the proposal, and whether incidental implementation choices have become unnecessary constraints. Examine discriminating cases and actual assertions, not only test names. If all examined sources share the same mistaken assumption, agreement alone cannot establish intended correctness; further authoritative domain evidence or an owner decision is still needed. Keep that uncertainty visible rather than claiming automatic defect-free extraction.

### Reuse independent review and integration

The extracted model is the work product. Its independent local review checks correctness, useful detail, important omissions, source support and unsupported invention. Reuse existing review and rework, rather than a separate permanent completeness catalogue or review organization. Parent integration reconciles shared concepts, public boundaries, duplicate or conflicting statements and cross-Package links; the integrated result receives independent review too.

Detail depth is an Extract execution choice supplied to both author and reviewer. Greater depth examines more error cases, state transitions and technical contracts. It cannot make an observed contradiction disappear or turn missing evidence into a pass. Exact settings and defaults are not selected here.

A useful review instruction is: **Could important behavior of this area change while this model would still permit it? If so, identify the missing statement.** Apply it to existing source evidence, not speculative future requirements. For example, if the implementation enforces five active loans per member and refusal leaves state unchanged, “members can borrow books” omits both important constraints. The model should distinguish a sixth accepted loan or a partially applied refusal. The reviewer also challenges unnecessary constraints that promote incidental implementation choices into required future design.

### Refine with Verify findings

Compile and validate the candidate and its accounting, then run read-only Verify against the fixed existing artifacts. Extract investigates the findings and changes its candidate, never the existing realization:

| Finding | Extract response |
| --- | --- |
| The candidate misstates a source-supported intended requirement or confuses it with observed behavior. | Correct the candidate with a reason and source basis; review the change. Do not rewrite correct intent to match faulty artifacts. |
| A statement or its scope is too vague. | Make the observed rule precise without inventing additional behavior. |
| A justified connection is missing. | Add and review the explicit model link; recompile affected contexts. |
| Code, tests or documentation violate an already established intended requirement. | Keep the justified model statement, record the concrete realization deviation with its Definition and evidence, and retain it for later authorized artifact work. |
| Sources conflict and intended meaning remains unsettled. | Present a concrete owner question and retain the unresolved conflict. |
| Evidence is insufficient. | Examine further available sources or retain the evidence gap; do not weaken the statement to force conformity. |

After each model revision, update affected context and proposed accounting, review and integrate the model changes, and Verify again against the same artifact basis. A changed meaning cannot inherit an old positive assessment automatically. Material questions or evidence gaps that prevent a sound model or assessment, budget exhaustion or no useful progress leave an explicit incomplete result, not an endless loop or a manufactured pass. A known, adequately supported realization deviation alone does not require another model rewrite or prevent adoption readiness. Source drift requires an explicit new source binding.

Verify can pass against a shallow model. The normal Extract review therefore also judges whether important source behavior is missing from the model. This is a model-quality responsibility of Extract, not a broader meaning of Verify. Removing a candidate rule does not erase the source observation or its open review finding.

### Completion and adoption with known deviations

The owner's [completion clarification](../../concepts/sources/commands-extract-optimize-20261017.md#model-adoption-with-known-realization-deviations) replaces the earlier requirement that Verify must report conformity before Extract can finish. A reviewed model proposal may be adopted while the existing realization has known deviations. Successful model preparation and existing-artifact conformity are separate judgments.

Completion for the selected scope requires:

- A structurally valid, independently reviewed and integrated model candidate with adequate detail, supported intended meaning and explained proposed artifact accounting.
- Completed read-only realization assessment against the exact candidate and fixed source basis. Its actual conformity or nonconformity remains visible.
- Each known realization deviation identified by its applicable Definition, source/candidate binding, evidence and reason. Unsupported suspicions or insufficiently assessable behavior are not relabeled as known deviations to close the work.
- No unresolved material question or evidence gap that prevents a sound model, adequate coverage of the selected scope or the classification of a reported deviation.

Report partial scope honestly. The owner reviews and commits the chosen model and accounting through the normal adoption boundary, with known deviations retained in the handoff. Adoption neither changes a nonconforming Verify result to a pass nor accepts defective artifacts as correct. Do not weaken or delete a justified model rule, hide a finding, or change existing code, tests or documentation to manufacture conformity during Extract. Resolving a model question may refine the candidate; correcting an existing realization remains a separately authorized artifact operation.

For the library example, an honest result can say: "Model proposal reviewed and ready for owner adoption. The existing realization admits a sixth loan against the five-loan rule; this deviation is retained. No material intent question remains. No artifact repair was performed." A later Apply can repair that deviation against the adopted model. Model adoption alone does not establish that the repair was executed or verified.

The reuse above is a design direction. How a provisional model tree is rebuilt between rounds, how candidate model/registry bytes are staged for Verify, and how model-writing sessions are admitted need future producer contracts. Current artifact Apply still protects canonical and control inputs; this document grants it no new mutation rights.

## Optimize the canonical model

Optimize extends the existing [strict links and optional Optimizer direction](../../concepts/register.md#dec-027-strict-links-with-an-optional-optimizer). It uses the Manager tree to examine the model itself, with independent model review and parent integration. It may inspect relevant artifacts as evidence, but its change target is the model and it does not repair artifacts.

Look for missing justified links, unclear wording, duplication and organization that can be improved while preserving intended meaning. A link proposal names its source, target and reason. Wider reading is allowed within the selected scope because existing links are exactly what the investigation tests; this does not authorize compensating for missing links during ordinary Apply or inventing semantic relationships.

“Without inventing too much” means that additions need a concrete basis in existing model content or examined evidence. Do not invent requirements, features, guarantees, relationships or architecture to make the model seem more complete. Ambiguous intent and changes of meaning are questions or explicit proposals for the authorized decision process, not automatic optimization. A justified no-op is a valid result. Review and adoption follow existing authority; there is no manual approval round for every clear link.

Optimize improves the existing model; Extract derives an initial candidate from a repository. Both can reuse the same model-authoring, review, findings and integration capabilities. No separate optimization framework, automatic operation schedule or implementation package is introduced here.

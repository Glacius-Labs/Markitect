# Commands over the Manager organization

Status: accepted direction, 17 October 2026 ([DEC-044](../../concepts/register.md#dec-044-commands-reuse-the-manager-organization), [owner discussion](../../concepts/sources/commands-extract-optimize-20261017.md)). This document owns the later command semantics and the Extract and Optimize workflows. It records no executable command registration, model-writing runtime or completed Verify capability. The [current operations guide](../../project-operations.md) owns availability. Apply remains the first delivery target; this documentation adds no implementation queue and does not change the frozen Apply core.

## Apply and Extract are the two directional processes

The owner clarified [DEC-057](../../concepts/register.md#dec-057-three-fundamental-starting-situations) on 19 October 2026: the [three starting situations](../../vision.md#three-starting-situations) use two processes, **Apply: Model -> Artifacts** and **Extract: Artifacts -> Model** ([owner source](../../concepts/sources/three-starting-situations-20261019.md#owner-selects-apply-and-extract)).

Apply covers both an initial realization with no artifacts and reconciliation of existing artifacts. Their different starting states do not require separate Build or Reconcile processes or commands. Root-owned foundations, recursive construction, integration and verification remain part of Apply. Extract develops a model candidate from existing artifacts through review, integration, owner clarification and read-only Verify; adoption describes its purpose for an existing project.

These are the two processes for those starting situations. Assessment and maintenance operations below retain their stated objectives and authority; this clarification neither removes their names nor limits the CLI to two verbs. It adds no executable Extract registration or runtime capability.

## One organization and different instructions

Use the existing Manager organization, compiled context, independent review, findings, correction and parent integration. The operation supplies one static instruction, propagated unchanged through the tree. Each Manager concentrates on its own responsibility because it receives its own model and artifact context. Parents integrate their own work and direct children's results. They do not rewrite the operation for children or choose which required Manager participates.

An instruction describes the objective; the Orchestrator enforces participation, allowed effects, budgets, required review and source/candidate bindings. A prompt alone grants no write authority. Model changes and artifact changes stay separate ([DEC-026](../../concepts/register.md#dec-026-apply-comes-first-and-changes-only-artifacts)); read-only operations may record reports and findings. Configuration and custom instructions cannot introduce binding project requirements outside the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)).

The accepted later [model-migration workflow](#model-restructuring-as-an-explicit-migration) includes the registry and configuration adjustments necessary to preserve model bindings and responsibility. These are explicit migration metadata, not permission to modify realization artifacts or unrelated execution policy. Current artifact Apply gains no model/control write rights from this design.

The names below express user goals, not a new CLI schema. Reuse the process with an appropriate instruction and change target rather than inventing an engine for every verb. Exact registration, command arguments and draft-model execution bindings remain implementation design, within [DEC-017](../../concepts/register.md#dec-017-command-model-top-level-verbs-from-one-verb-table).

| Operation | Change target | Instruction and result |
| --- | --- | --- |
| Apply | Artifacts | Realize the current canonical model across the complete responsibility, whether artifacts are absent or already exist. Create missing artifacts and reconcile existing ones, including unchanged-intent repair and the existing removal rules, or return a justified reviewed no-change result. Valid realization can stay as it is. |
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

1. Fix the repository revision and selected scope, including the existing requirements and decisions that can establish intended meaning. Keep exclusions and unknown areas visible. Existing code and other realizations remain the observation basis throughout the refinement loop; a complete canonical model cannot be an entry requirement when the purpose is to derive it.
2. Root begins as the entry responsibility for this selected source scope. Inspect the project overview, code, interfaces and documentation and propose the first responsibility boundaries and Package structure. Explain the basis of each boundary. Folder names are clues, not automatically the right model structure; this is a provisional model outline, not an accepted specification inferred from directory names.
3. Form the initial Manager tree from that candidate Package structure. It is a proposed organization under review, with known sources, area boundaries, parent/child relationships and unresolved overlaps, rather than an already authoritative hierarchy. Do not invent an accepted empty model or assume the final Manager organization exists before this discovery.
4. Parents supply a usable investigation basis before their children develop the model: the assigned source scope, the provisional model context, known shared terms/contracts with their status, and relevant open questions. The parent need not finish the whole model first. Each Manager examines its actual area and drafts Definitions for concepts, behavior, rules, interfaces, architecture and dependencies; the pattern repeats for its children. Retain source support for consequential statements. A Definition need not mirror a source file or every private implementation detail.
5. Propose ownership and Definition links for existing artifacts, using the registry and explicit accounting rather than a second intended model. A file assignment alone does not establish that its behavior has been described adequately.
6. Integrate the model upward, review it and clarify consequential unresolved meaning with the owner through Root. If discovery exposes a wrong split or a missing responsibility, propose the correction within the current round and adopt the revised candidate structure between rounds as described below. An inferred reorganization gains no hidden authority over existing artifacts.

The model remains a candidate until adopted through the project's decision process. Source revisions and evidence do not themselves confer authority. Imported vocabulary is read-only; it is not silently rewritten to make the extracted project compile.

The owner accepted this [case-3 walkthrough](../../concepts/sources/three-starting-situations-20261019.md#owner-confirms-the-extract-walkthrough) on 19 October 2026. For an existing Library, Root might propose Library, Members, Catalog and Lending, with Reminders beneath Lending, because their source-supported responsibilities differ. That example is a provisional interpretation of an existing repository, not an instruction to force every repository into the Library hierarchy. Root reconciles cross-area terms; Lending integrates the model proposed for Reminders. A later discovery may justify changing that outline.

Questions presented to the owner name the sources, the meaning at issue and the consequence of the choice in understandable terms. Group related questions instead of asking for approval of every inferred statement; established authoritative intent needs no redundant acceptance. The owner can inspect the model, accepted decisions, open points, examined scope and known deviations through a readable account without reviewing YAML. A response refines the candidate through the normal model review/integration path; it does not turn unreviewed inference into authority by itself.

### Keep the structure stable within each Extract round

The owner [confirmed this refinement](../../concepts/sources/three-starting-situations-20261019.md#owner-confirms-structure-changes-between-extract-rounds) on 19 October 2026. One Extract round uses one stable provisional Manager tree and assigned responsibility boundaries. Managers can discover and describe a missing responsibility or a mistaken boundary during that round; they do not immediately create, remove or reparent active Managers. Work that can be supported under the current boundaries may continue. A blocked question or disputed assignment remains visible rather than being resolved by an uncoordinated takeover.

Between rounds, use the existing model review and parent integration:

1. **Explain the proposed structure change.** Name the evidence, the existing boundary, the proposed responsibility and its effect on shared concepts and integration. The shared parent coordinates changes crossing child boundaries, with Root integrating the whole candidate. A source folder or technical dependency alone does not determine a model boundary.
2. **Carry the work forward completely.** Preserve derived model statements, their sources and interpretation status, accepted decisions, references, findings, known realization deviations and every unresolved question. Assign them explicitly to the resulting responsibilities; a cross-area question remains visible to their shared parent. Keep old identities and source bindings traceable. Moving a Definition to another Package needs an exact old-to-new identity mapping and updated imports, typed references, body links and proposed artifact accounting under the existing [identity and migration rules](#model-restructuring-as-an-explicit-migration). Nothing is silently dropped, merged or treated as resolved because its former Manager disappears.
3. **Bind the next round to the revised candidate.** After review and integration within the assignment's authority, derive the next Manager tree from the complete candidate Package inventory and provide each Manager its relevant retained findings, questions, sources and newly bound context. Keep the complete record available without copying every prior conversation into every context. The old and revised organizations do not work concurrently on the same model responsibility. This is a transition between provisional candidates, distinct from final owner adoption of the model; it does not require a new human approval for every meaning-preserving boundary correction already within the grant. Unresolved intended meaning or authority still goes through Root to the owner.
4. **Assess the revised candidate.** Review the integrated model and give read-only Verify the exact new candidate and unchanged artifact basis. Earlier positive assessments remain evidence about their original subjects; they do not automatically approve the revised structure or model. Findings and open questions remain traceable through the transition and subsequent refinement.

For the Library example, Lending might discover a separately describable Reminders responsibility and propose a child Package for the next round. If Reminders was initially placed under Catalog because of its code folder, source evidence may instead support integration under Lending. Root coordinates that boundary change; the next round carries the reminder rules, source references and unresolved delivery questions to Lending/Reminders. Neither example requires moving or modifying application files. These are corrections to an initial Extract proposal, not general authority to split or restructure an adopted model outside its operation's grant.

Historical conversations and evidence retain their original bindings. A changed structure remains within the assignment's existing budget and authority boundaries; this refinement selects no new budget accounting rule. The stable-round rule and complete carryover are selected workflow semantics. Exact transition, staging, interruption, recovery and Manager/context rebuild protocols remain future implementation design.

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

After each model revision, update affected context and proposed accounting, review and integrate the model changes, and Verify again against the same artifact basis. Each independent assessment names the exact model candidate and fixed source stand: model review judges the proposal's quality and support, while read-only Verify judges artifact conformity to that proposal, including integration across responsibilities. A changed meaning cannot inherit an old positive assessment automatically. Material questions or evidence gaps that prevent a sound model or assessment, budget exhaustion or no useful progress leave an explicit incomplete result, not an endless loop or a manufactured pass. A known, adequately supported realization deviation alone does not require another model rewrite or prevent adoption readiness. Source drift requires an explicit new source binding.

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

The reuse above is a design direction. The [stable-round rule](#keep-the-structure-stable-within-each-extract-round) selects when the provisional structure changes and what must be retained. How that tree is rebuilt between rounds, how candidate model/registry bytes are staged for Verify, and how model-writing sessions are admitted still need future producer contracts. Current artifact Apply still protects canonical and control inputs; this document grants it no new mutation rights.

## Optimize the canonical model

Optimize extends the existing [strict links and optional Optimizer direction](../../concepts/register.md#dec-027-strict-links-with-an-optional-optimizer). It uses the Manager tree to examine the model itself, with independent model review and parent integration. It may inspect relevant artifacts as evidence, but its change target is the model and it does not repair artifacts.

### Find candidates beyond existing links

The owner selected [model-wide search with targeted reading](../../concepts/sources/commands-extract-optimize-20261017.md#optimize-search-space-and-targeted-reading). Existing links are the starting point, not the search boundary: they are part of what Optimize investigates. Need to know bounds loaded context rather than prohibiting a justified read, consistent with the existing [context contract](manager-apply-concept.md#context-packages).

| Concern | Selected boundary |
| --- | --- |
| Searchable evidence | The selected local model, explicitly bound dependencies and permitted artifacts from the fixed repository capture. Searching does not silently add repositories, revisions or unbound sources to that capture. |
| Loaded context | Own obligations, existing explicit references and targeted reads of relevant candidate sources and contracts. Do not preload the entire searchable model into every prompt. |
| Change authority | Own model responsibility under the existing review and integration process. Reading another area does not transfer write authority; imported vocabulary remains read-only in the consuming project. |

Use the existing Manager organization in this order:

1. **Assess the own area.** Every Manager examines its own Definitions and applicable obligations for missing links, unclear wording, duplication, contradictions and responsibility mismatches. Participation does not depend on AI relevance selection. Required local context is not silently shortened to fit a limit; existing context-limit failures remain visible.
2. **Search from a concrete question.** For example: what does membership eligibility mean, where else is this limit described, or which Definitions use this contract? Use a simple model inventory and text search across the selected model, not only already linked Packages. Similar words produce candidates, not established relationships.
3. **Read progressively.** Begin with identity, Kind, authored title when present and the source match. Read the promising Definitions and necessary type or other contracts to answer the question. A further concrete question can justify another read within the same grant. Targeted artifact evidence may expose an unmodeled dependency; it does not create a canonical link automatically.
4. **Assess cross-area meaning together.** Reuse existing communication when another Manager must clarify meaning or responsibility. Parents coordinate and integrate cross-area changes; no Manager takes over a sibling's Definitions. Parent integration and independent review examine differently named concepts, conflicting Definitions and unsupported connections as well as lexical matches.

The combination of every Manager's own assessment, parent integration and independent review reduces dependence on word search alone. It does not establish that every relevant semantic relationship was found. Reviewers challenge proposed relationships independently within the same evidence and authority boundaries; they need not load the whole repository to do so.

Search output is bounded with further matches visibly available for continuation, not silently discarded as irrelevant. A capped or unfinished search is not an exhaustive result, and unobserved match counts are not invented. Additional reading answers a concrete question or investigates a named suspicion. Existing work and context budgets remain in force; an exhausted budget or unresolved context limit is an explicit investigation boundary, not proof that no link is missing.

Use the ordinary reports to retain relevant sources, conclusions and open search questions. Do not add a second search catalogue or require human approval for each read already within the grant. For example, a lending Manager can search for membership eligibility, inspect the candidate meanings and request clarification without preloading all membership, catalog and reservation documents. Start without a fixed maximum link depth or semantic-index subsystem. Later concrete Optimize use must show where relevant relationships are missed or unnecessary reading accumulates before that approach is extended. This selects the design, not an executable search API or authority to run those studies.

### Explain each proposed change

In the normal change proposal, show the existing source statement, the problem, the proposed edit, why the meaning is preserved and the expected benefit. A link names its source, target and reason. Describe useful context gained and unnecessary context added; more links or more text are not success criteria. This explanation belongs to the ordinary review subject, not a new permanent proposal catalogue.

| Observation | Appropriate response |
| --- | --- |
| A lending Definition explicitly says it uses membership eligibility, but omits the link to the unambiguously named eligibility Definition. | Propose that link with both source statements and the intended context effect. |
| Lending and reservations both use the word “availability”. | Investigate their meanings; the shared word alone establishes no relationship. |
| Code uses a 14-day term, but the model specifies no duration. | Report a possible modeling gap or ask a question. Do not promote the implementation number into a binding requirement automatically. |
| Two similarly worded rules apply to different areas. | Preserve their scopes; do not merge them merely because the wording is similar. |

“Without inventing too much” means that additions need a concrete basis in existing model content or examined evidence. Do not invent requirements, features, guarantees, relationships or architecture to make the model seem more complete. Artifact evidence can expose a gap; observed behavior alone does not establish intended meaning. Ambiguous intent and changes of meaning become questions or explicit proposals for the authorized decision process, not automatic optimization.

### Review meaning and integrate the result

Reuse independent local review and existing rework. The reviewer receives the proposed change and its source basis and asks: **Is this statement or connection already justified by the existing content, or does this edit decide something new?** In particular:

- Is the target unambiguous, and does the source support the stated relationship?
- Does the edit strengthen an obligation or lose a condition, exception or scope?
- Does a new link provide needed context or only add unrelated material?
- Does consolidation turn an incidental implementation choice into a future requirement?

A reverse link is not added merely to make the graph symmetric. Plausible but unsupported additions become review findings and return through normal rework. Several materially different readings become a source-bound owner question with alternatives. Existing authority can cover clear reviewed links; no new manual approval round is required for each one.

Parents integrate accepted local work and obtain independent review of the aggregate, especially shared terms, conflicting edits and cross-area context effects. Structural checks and context previews complement semantic review: a structurally valid link may still be wrong. A justified no-op is valid. The semantic results are a reviewed improvement, a grounded question or a justified no-change result; missing evidence and exhausted capacity remain incomplete under the ordinary execution rules. These descriptions select no new wire-enum names.

The [owner accepted this workflow](../../concepts/sources/commands-extract-optimize-20261017.md#optimize-workflow-clarification). Review and adoption retain the existing project authority. This adds no implementation queue or general permission to change model meaning.

Optimize improves the existing model; Extract derives an initial candidate from a repository. Both can reuse the same model-authoring, review, findings and integration capabilities. No separate optimization framework, automatic operation schedule or implementation package is introduced here.

### Model restructuring as an explicit migration

The owner accepted Package moves and merges when meaning is preserved, together with the [migration workflow](../../concepts/sources/model-migrations-20261017.md). Optimize prepares one jointly reviewed candidate containing the model changes and the necessary registry/configuration adjustments. Existing code, tests and other realizations stay unchanged. This is a later-operation design, not an implemented model writer or an expansion of artifact Apply.

**Identities and references.** Keep the [v1 identity and migration rules](minimal-language/language-contract.md#names): identity is Package path plus local `id`. Moving or renaming a Package changes its Definitions' identities; renaming a file within the same Package does not, if `id` stays the same. Record the exact old-to-new identity mapping in this migration's review subject and history. There is no new global identity layer, inferred rename, hidden alias or automatic forwarding of old references.

| Old identity | New identity |
| --- | --- |
| `lending/borrow` | `circulation/borrow` |
| `lending/loan-policy` | `circulation/loan-policy` |
| `returns/return` | `circulation/return` |

Migrate known imports, Kind/Property and value references, explicit body links and Package-scope references. Review free prose too. Preserve effective Property field contracts when a local ID changes; identity and field name are different concerns under v1. First move the existing Definitions when merging Packages. An ID collision needs an explicit resolution; it is not permission to merge Definitions, discard one or guess that their meanings are identical. Combining their meaning requires its own justified, reviewed change.

**Meaning and scope.** Review the old and new model under the mapping: the same obligations must still govern the same subjects, with the same conditions, exceptions and relationship meanings. For example, moving a lending-only Rule into a Package shared with the catalog must not make it constrain the catalog. Similar bodies do not establish equivalent scopes. If the new structure cannot express the old scope faithfully, propose another structure or present the required decision; do not call the change meaning-preserving. Context may be reorganized, but obligations cannot silently expand or disappear.

**Managers and configuration.** Create and review the candidate against the bound starting organization. Parents coordinate cross-area moves under the existing ownership rules; no Manager takes over a sibling's model work. Derive and inspect the future Manager tree from the complete new Package inventory, including empty and intermediate Packages, rather than inferring it only from Definition digests. When two Packages become one, their responsibilities and artifact accountability pass to the resulting Manager; parent integration duties are recalculated as well.

Activate the new organization only after the migration is accepted, with newly bound contexts. Do not reshape an active organization midway through its run. Historical tasks, conversations and evidence retain their original identities and bindings; an old local or integration approval does not automatically approve the new subject. Existing consumption is not reset and budgets are not added together by a merge.

Inspect exact Package settings, named reviewers, instructions, approvals, limits and permitted areas as part of the candidate. Move an unambiguous binding explicitly. If former Managers have incompatible effective settings or authority, present the conflict instead of choosing a winner, adding budgets or silently broadening rights. Configuration migration preserves bindings and required controls; it is not an unrelated policy change. The [configuration owner](manager-apply-configuration.md#resolve-the-team) retains its resolution rules.

**Registry and retained artifacts.** Artifact paths need not follow Package paths. For example, `internal/lending/borrow.go` can remain at that exact path while accountability moves to `circulation`. Transfer records to the new owner's registry shard, migrate `realizes` identities and update their reasons where needed. Keep exactly one accountable owner, preserve every still-justified basis and reject stale or unresolved links. An explicitly reviewed move with a new basis must not be mistaken for removed intent and trigger deletion of retained artifacts. Unmapped removal or an unjustified basis still follows [DEC-034](../../concepts/register.md#dec-034-removed-intent-removes-its-realization); a name similarity does not excuse an orphan.

**Review, assessment and adoption.** Review and structurally validate the complete candidate: model, reference migration, scope preservation, future topology, configuration and registry. Give read-only Verify the exact candidate model and unchanged artifact basis. Keep its real result and known deviations visible, including the accepted distinction between [adoption readiness and realization conformity](#completion-and-adoption-with-known-deviations). Neither Optimize nor this metadata migration repairs the realization.

Adopt the model and its necessary metadata as one coherent reviewed change under the project's authority, with guarded publication and actual read-back. An incomplete or partially published combination is not a successful migration; no filesystem transaction is claimed here. Preserve the old source bindings and migration history rather than rewriting prior records. Current publication ports and protected-input rules do not yet implement this model-writing operation. Its exact staging, control metadata and recovery contracts remain producer work; this acceptance grants no runtime execution or implementation queue.

## Remaining command decisions

The workflow and authority distinctions above are selected. The questions below are still open; the recommendations are assistant proposals, not owner decisions or implementation assignments.

| Priority | Open question | Recommendation and tradeoff |
| --- | --- | --- |
| 1 | How is Extract detail depth expressed and evaluated consistently? | Give author and reviewer one short shared objective with concrete behavior, refusal, state and interface examples. Select exact settings only from useful cases; this stays simple but needs judgment rather than a numerical completeness score. Known obligations and material contradictions cannot be waived. |
| 2 | How are Custom instructions and operation inputs registered and bound reproducibly? | Reuse the common operation, context, target, authority and budget bindings. Keep the accepted top-level verb direction and six-skill split; command labels need not create matching skills. Exact formats and optional model-only Cleanup/Refactor naming remain to be selected. This avoids duplicate workflows but requires a precise common invocation contract. |

Optimize's [search boundary and targeted-reading workflow](#find-candidates-beyond-existing-links) are now selected. Exact search-result shapes, continuation mechanisms and numeric limits remain producer details under the existing authority and budget rules; they are not another unresolved product-scope decision or permission to implement a search subsystem.

Package moves and merges with preserved meaning are now selected in the [migration workflow](#model-restructuring-as-an-explicit-migration), not an unresolved product decision. This does not implicitly select other transformations such as splitting Packages or merging Definition meanings. Its producer contracts must stage a bound candidate, derive the future Manager tree without mutating the active one, migrate necessary accounting/settings and give read-only Verify an exact model/artifact pair. Current artifact Apply cannot bypass protected inputs. These are implementation details within the selected semantics, not authority to start that work. The accepted [Init/skill design](../project-capability-skills-20261009.md#accepted-init-design--17-october-2026) keeps technical setup and model adoption separate.

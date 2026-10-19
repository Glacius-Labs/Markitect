# Modeling skill guidance

**Status:** accepted guidance, owner, 19 October 2026, in the "Modelling" conversation. This is the content design for the existing `markitect-design` responsibility in [Capability skills and native project entrypoints](project-capability-skills-20261009.md#the-six-installed-skills), not an additional operation skill.

**Scope:** documentation only. No skill is implemented or installed, no provider instructions are generated, and no Apply implementation is changed. The [language contract](canonical-model/minimal-language/language-contract.md) owns syntax and validity; the [standard-library contract](canonical-model/standard-library.md) owns its selected vocabulary. This guidance does not add language constructs, status fields, a draft store or an approval mechanism. The [project workflow](../project-workflow.md) owns available operations.

## Owner acceptance

After a unified instruction covering decision boundaries, architecture and organization principles, and concrete examples, the owner accepted it and requested this record:

> Das passt. Halte die einheitliche Anleitung mit den Beispielen in der bestehenden Markitect-Dokumentation fest. Noch keine Implementierung oder Installation des Skills.

The owner also established the intended interaction: model during the conversation, keep open ideas and proposals separate from accepted decisions, and let people work through understandable views without reading YAML. Bugs concern realization. Managers recognize and handle conflicting child statements during Apply or Verify. The tool-lending examples below illustrate this guidance; they are not requirements for Markitect or every adopting project.

## Purpose and authority

Help the person describe and maintain the project's intended world. Model during an authorized modeling conversation. Take care of terminology, structure and model maintenance. Explain meaning and relationships so the person never needs to inspect YAML to understand or decide the model.

The model describes meaning, structure, behavior, rules, architecture and responsibility. A technical constraint belongs there when it is a deliberately binding part of the intended project. Actual operating state, defects and executed evidence belong to realization. Observed code can inform a proposal; it does not automatically define intended behavior.

Keep these content states distinguishable:

| State | Meaning |
| --- | --- |
| Accepted | Intended content that has been decided. |
| Proposed | A developed option that has not yet been adopted. |
| Open | A question, alternative or unfinished idea. |

Open ideas and proposals must not create binding realization obligations. These labels describe their status, not new fields to insert into ordinary binding Definitions. Use the actual project workflow to preserve the distinction. An accepted meaning in conversation and its adoption into a committed specification remain separate events under the project's acceptance policy. A request for explanation alone grants no model-write authority.

Before changing anything, establish the requested change, existing decisions, delegated design freedom and decisions reserved to the person. A design mandate can authorize choices that introduce new meaning. Explain material choices made within that mandate. Permission to draft does not automatically adopt the draft as the accepted specification.

## Work with the conversation

1. **Read the existing context.** Find the authoritative Definitions, relevant relationships and applicable rules. Use established terminology and conventions where their meaning fits. Similar names are search clues, not proof of equivalent meaning.
2. **Make the meaning explicit.** Distinguish subjects, roles, relationships, rules and intended behavior. Explain who belongs to what, what requires what, and which constraint applies under which conditions.
3. **Maintain the draft as the conversation develops.** Incorporate clear statements into the authorized draft. Preserve tentative thoughts as open ideas. Ask when different answers materially change current meaning, responsibility or constraints. Group less urgent questions instead of interrupting every thought.
4. **Choose useful structure.** Give something its own Definition when its identity, responsibility, repeated references or independently changing contract justify it. Explain the benefit of additional structure. Avoid both unrelated obligations in one catchall and a Definition for every sentence or implementation file.
5. **Maintain each fact at its owner.** Change the existing authoritative statement rather than creating competing versions. Reference shared rules. Preserve identity during presentation-only edits; treat identity and Package migrations explicitly.
6. **Explain the actual model.** At useful points, show meaning added, changed or removed, its relevant consequences, and what remains proposed or open. Use concrete situations to explain decisions. Make a coherent readable view of the actual model available, not only a chat summary.

This is a preferred working shape, not six mandatory phases or six separate approval requests.

## Describe meaning and relationships

A field list often says too little. "A loan has a member ID, copy ID and date" does not explain the relationship or lifecycle. A more useful description is:

> A loan associates one member with one particular copy. It starts open and is closed by a return.

Add only properties already established or explicitly proposed. That description alone says nothing about indefinite retention of closed loans.

State the meaning of a relationship: membership, use, prerequisite, responsibility or constraint. A body link identifies its target; its surrounding sentence states the relationship. A link alone supplies no inheritance, routing, authority or execution order.

For rules, make the subject, scope and conditions understandable. Check whether words such as "always", "never", "complete", "all" and "indefinite" accidentally strengthen the contract. A rule must not acquire a broader scope merely because another Definition references it.

Keep together content that changes for the same domain reasons. Separate different meanings and responsibilities. Reuse follows a shared meaning and contract, not just matching names or fields. An example may repeat a value to illustrate a case without becoming another independent owner of that value.

Use existing Kinds and contracts where they fit. Justify a new Kind or Property with a concrete need for its contract. Do not add structured fields to the selected empty-spec standard-library Kinds. A Package change can affect identity, context and Manager responsibility; it is more than cosmetic folder maintenance.

## Architecture and organization as design aids

The following are modeling adaptations of established approaches. They provide questions for design, not automatic project obligations.

| Approach | Useful design question | Boundary |
| --- | --- | --- |
| DDD | Which language and rules belong to this domain context? How do different contexts relate? | Do not unify concepts solely because they share a word. A context is not automatically a service, Package or deployment. |
| Clean Architecture | Which domain contracts should remain independent of UI, storage or external providers? | Do not impose a universal set of layers, interfaces or repositories. |
| Clean Code and Single Responsibility | Is the content clear and cohesive? Which parts change for the same reasons? | Do not impose a fixed Definition size or create abstractions without a purpose. |
| Organizational design | Who owns which outcome, which decisions are delegated, which neighbor contracts apply, and who integrates? | Do not equate real departments, domain contexts, software Components and Packages automatically. |

These adaptations draw on [bounded contexts](https://martinfowler.com/bliki/BoundedContext.html), [Clean Architecture's dependency rule](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html), [Single Responsibility and reasons for change](https://blog.cleancoder.com/uncle-bob/2014/05/08/SingleReponsibilityPrinciple.html), and [Team Topologies' responsibilities, interactions and cognitive load](https://teamtopologies.com/key-concepts). Their usefulness for this modeling guidance remains to be observed in real use.

Describe a proposed responsibility through its purpose, outcome, local decisions, neighbor contracts and escalation boundaries. Justify added coordination by its benefit. Several domain contexts can exist within one canonical project model with explicit relationships; they do not create competing truth owners. A Component is not automatically a class, process, Package or Manager; the [Component contract](canonical-model/component.md) owns that distinction.

## Decide, propose or ask

Before deciding independently, ask: **Which alternatives does this change newly rule out, and am I authorized to decide that?**

| Situation | Action |
| --- | --- |
| Meaning is unchanged and existing contracts cover the presentation or classification | Maintain it independently. |
| The change is explicitly requested | Perform it and explain the changed meaning. |
| A consequence holds under every interpretation still allowed by the existing contract | Make it explicit and explain it. |
| Materially different domain interpretations remain possible | Ask the consequential question when it affects the current scope. |
| A new obligation, exception, guarantee, transition rule or responsibility boundary is needed | Decide within an explicit mandate; otherwise present a proposal for decision. |
| The question belongs to an explicitly later scope | Keep it open and continue the current scope. |
| Realization violates clear intent | Treat it as a realization problem; do not adapt intent to the defect. |

Plausibility does not make a statement a necessary consequence. Easy reversibility does not supply missing authority. Explicit delegation is real freedom: do not ask again for every choice it already covers.

An invariant states what must hold; it does not necessarily select one complete workflow for ensuring it. When changing a rule, consider relevant existing states. Do not silently add migration behavior, grandfathering, exceptions or new guarantees.

## Worked boundary examples

All examples concern a hypothetical club tool-lending project. Each includes the bad move and the desired response so the distinction can transfer to other domains.

| Situation | Bad move | Better response and reason |
| --- | --- | --- |
| "We have several identical drills, each with its own inventory number." | Model only a drill type with a stock count. | Distinguish tool type from individual copy; each copy has its inventory number and belongs to a type. The stated distinction supports individual identity without selecting a barcode technology. |
| "Raise the existing limit from three to five." The contract already counts simultaneously open loans. | Ask again whether past loans count, or add another independent limit. | Change the authoritative limit and explain it. Read established context before reopening questions. |
| "At most three copies at once." | Automatically introduce a rejection dialog, waiting list or exchange offer. | Record the limit. Propose how an at-limit request is handled when developing that operation; the invariant alone does not choose the workflow. |
| "Lower the limit to three." Existing loans can exceed it and no transition policy is settled. | Close loans or silently invent grandfathering. | Ask how existing excess loans are treated; suggest preserving them and blocking further lending until below the limit. The transition needs a decision. Where the actual model already settles it, use that rule without asking again. |
| "A refused issue must not create a loan." | Forbid every state change, including logging. | Name the protected domain state precisely. Do not accidentally prohibit otherwise permitted operational records. |
| "Past loans should remain understandable." | Require indefinite, immutable retention of every datum. | State the intended traceability. Treat retention duration and mutability as separate questions when relevant. |
| "Maybe reservations later." | Adopt reservation deadlines and waiting-list rules now. | Keep reservations as an open idea and continue the agreed first scope. |
| Two areas use "member" | Merge Definitions based only on the shared word. | Compare meaning, identity and responsibility before proposing shared or distinct contracts. |
| "The limit must hold for concurrent requests too." | Mandate a particular database or locking strategy. | Record the required property and preserve technical freedom allowed by the architecture. |
| "Staff might need more flexibility." | Exempt all staff automatically. | Keep the idea open. When relevant, ask whether the intended choice is a higher limit or authority to approve individual exceptions. |
| "Design a sensible domain structure yourself." | Seek approval for each Definition. | Use the delegated design freedom, explain material choices and identify reserved decisions. The design is still a proposal until adoption is authorized. |
| "The same copy was lent twice." A clear rule already forbids that. | Add an exception to accommodate observed behavior. | Identify the realization deviation and preserve the rule. |
| A return targets a specific loan; repeated return of a closed loan must change nothing | Let a repeated old return release the copy from a newer loan. | Preserve the operation's established target. If the target itself is still ambiguous, clarify it before deriving consequences. |

Ask about the decision, not for a complete solution. Instead of "How should the new limit work?", ask:

> How should existing loans above the new limit be treated? I propose keeping them and preventing further issues until the member is below the limit.

Name the uncertainty, consequences and, where useful, a justified recommendation. Group related decisions and continue independent parts. Do not ask about future reservations while clarifying a first scope that explicitly excludes them.

## Readable change reports and evidence limits

At useful conversation points, explain:

- **Changed:** what meaning was added, changed or removed.
- **Context:** which existing contracts it affects.
- **Agent design choice:** which delegated freedom was used, where material.
- **Proposed or open:** what does not yet apply.

Use this shape as needed, not after every message. A coherent readable view must reflect the actual model, including the relationships needed to understand it. A short chat summary alone is insufficient.

Concrete situations explain meaning and expose choices; they must not smuggle additional requirements into the model. Structural checks establish only their actual language and reference scope, not complete semantic correctness or realized behavior. Use the supported readable-view writer when maintaining a generated view; do not edit its output as another source of truth.

Managers handle conflicting child statements during Apply and Verify through the [current Manager process](canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026). The modeling agent supports clear intent and raises visible ambiguity; this guidance creates no additional exhaustive review institution. A finding can call for artifact repair, clarification or a genuinely authorized model change; it does not let the agent silently redefine intent.

## Shared guidance and later evaluation

Maintain this domain guidance once for Codex and Claude under the [shared-content design](project-capability-skills-20261009.md#one-maintained-source-two-native-outputs). Keep the decision-relevant core short and load detailed teaching cases when relevant. Binding project facts retain their canonical owner in the model; instructions explain how to work with them.

Later evaluation should use unfamiliar variations as well as these teaching cases. Observe whether agents preserve undecided ideas, update existing contracts at their owners, use delegated freedom, ask fewer redundant questions, and reduce corrections of lost meaning. No skill behavior, installation, runtime enforcement or effectiveness was tested by recording this guidance. This section authorizes no evaluation run or implementation.

Return to [Capability skills](project-capability-skills-20261009.md) or the [Design map](README.md).

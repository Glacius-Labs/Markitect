# Manager realization north star

Date: 16 October 2026. Working design direction and open variants from the [owner discussion](sources/manager-realization-20261016.md). The owner selected strict authored links with an optional Optimizer as the context baseline. Other alternatives retain their stated status; recording them does not schedule implementation. The [language specification v1](minimal-language/language-contract.md) remains the owner of the canonical language. The [concept register](../../concepts/register.md) owns accepted product decisions.

The [developed Apply concept](manager-apply-concept.md) proposes concrete answers to the open questions below and walks through a library system. It is a proposal for owner review; this north star preserves the original direction and alternative ideas.

## Purpose and direction

The canonical model describes the intended project world. Managers organize its realization as code, tests, documentation and other required artifacts. The intended management structure follows the model's Packages: one Manager per Package, with each Manager responsible for integrating its area. The root Manager is an ordinary Manager at the top of the hierarchy. Project Manager was a descriptive name for that position, not a separate role or Kind. Manager remains the working name.

The realization strategy should be customizable. A Manager's responsibility can stay consistent while its team, tools and way of working vary. The exact configuration format and extension mechanism remain open.

**Current priority:** Concentrate first on making Apply reliably reconcile artifacts with the model. Optimize, cleanup, refactoring, bug hunting and custom commands are extensions of that working foundation. Preserve their ideas without expanding the immediate design or implementation scope to all of them.

**Owner's implementation clarification:** Fully model the library as M1, perform a real initial Apply, inspect the resulting code/tests/documentation, then change the model to M2 and Apply again to the existing realization. Inspect the second result and its changes. Agent startup and communication are part of this demonstration. Repairing drift against unchanged intent remains useful additional coverage, not a substitute for this two-Apply sequence.

Compatibility does not constrain the rebuild. The owner permits radical deletion and replacement of obsolete source, models, commands and tests to realize the new concept cleanly. Do not introduce legacy adapters or preserve the old workflow merely to reduce the apparent size of the change. The Go source architecture and Playground need their own concrete designs before substantial implementation. Existing code is reusable material to judge against those designs, not the target architecture.

The owner raises an organizational perspective for further consideration: the Manager hierarchy could be the application's entry point, with the canonical model as its shared foundation. This perspective may also support a later Government extension. It is an open product framing, not a replacement of the model or authorization to implement Government. [DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis) retains the distinction between realizing a model and delegated decisions about model changes.

Use a simple library management system as the running example. The following tree illustrates responsibility; it is not an executable model or a prescribed vocabulary:

```text
Library root Manager
├── catalog
├── members
└── lending
    └── reminders
```

The root Manager coordinates the overall result. The lending Manager integrates lending and reminders. Each level has a broader view and normally works with fewer details than the level below it. Results and open questions travel upward; decisions, requests and additional information can travel downward. Details can be supplied when needed; the policy for direct file access at higher levels remains open.

The language already provides Package paths, identified Definitions, Kind and Property contracts, binding bodies, structured values, resolved references and fixed model content. These are inputs for a realization strategy. [Packages are namespaces](minimal-language/language-contract.md#packages); management responsibility would be assigned by the strategy. It does not follow automatically from the Core language. The model root itself is not a Package; how the root Manager is bound to it remains to be specified. The treatment of imported library Packages also remains open.

## A common Manager responsibility

The following is the initial conceptual contract developed in the discussion. Its data format remains open.

| Part | Contents | Lending example |
| --- | --- | --- |
| Receives | A task, its Package definitions and relevant contracts, realization context appropriate to its role, runtime configuration, and results or questions from child Managers. | Implement borrowing a book; consider registered membership and a maximum of five simultaneous loans. |
| Owns | Organizing the work and its checks, coordinating dependencies, integrating child results, and exposing missing decisions or conflicts. | Coordinate with members and catalog; combine lending and reminder work into a coherent result. |
| Returns | The integrated candidate, artifact registrations and their Definition links, review and check results, and a readable account of completed work and open issues. | Implementation, tests and usage documentation for borrowing, with their model basis and recorded checks. |

A Manager may return an incomplete result with a clear reason. The root Manager follows the same contract and integrates the whole library system. A Manager's own Package responsibility does not imply that all relevant knowledge lies inside that Package.

## Instructions enter through the root Manager

The user gives the operation's instruction to the root Manager. The root delegates to its child Managers, and each parent creates an appropriate task prompt for its children. Delegation is recursive; the original instruction is not independently sent to every Manager.

Delegated tasks should state the intended outcome, applicable model basis, scope and relevant constraints, while leaving local implementation decisions with the responsible Manager. In the library example, the root could ask lending to reconcile borrowing with the membership and loan-limit requirements. Lending decides which local artifacts need work. The root should not turn that task into a detailed list of descendant files to edit.

How to avoid excessive direction remains to be refined. The owner suggests that higher Managers might not initially know or read descendant artifact files, receiving concrete file details when another Manager supplies them. This is an open option, not an adopted prohibition on file access. It may preserve local autonomy and reduce context, but integration or investigation may require specific details. Reports, targeted requests and direct inspection are alternatives to compare before fixing the policy. A Manager's own local work must also be distinguished from its coordination of descendants.

## Reporting and communication history

Traceability belongs to reliable Apply from the start. The owner wants to reconstruct which Agent received which prompt, what it reported, how communication proceeded, and what information and changes flowed through each step. A final success message alone is insufficient for that investigation.

Readable reporting and detailed history serve different needs. Managers return concise progress, outcomes, evidence and open issues to their parents. A linked execution history preserves the underlying inputs, communications and changes for later inspection. Higher Managers need not receive every transcript in their normal context; reporting and need to know can coexist.

At a conceptual level, the history should connect:

| Record | What a reader should be able to establish |
| --- | --- |
| Run and step | Which operation, model stand and realization candidate the work belongs to, and which earlier event caused it. |
| Agent identity and role | Which Manager or team Agent acted, who assigned it, and which effective configuration and model profile were used. |
| Actual inputs | The task prompt, context package, applicable custom instructions and prior messages actually supplied for that step. Template names alone are insufficient when their content changes. |
| Additional information | Observed file reads, tool requests and results, context additions and questions or replies supplied after the initial prompt. |
| Communication and reports | Sender, recipient, content, order and relation to the task, including child reports, review findings, clarification and further delegation. |
| Changes and integration | The proposed artifact changes or model edits, their prior and resulting states, registry updates, review revisions, rejected candidates, integrations and actual application. |
| Outcome and evidence | What completed, failed, remained open or was interrupted, with the associated checks and evidence. |

Reports of changes and observed changes remain distinguishable. For example, an Agent's claim that it fixed the loan limit should link to the actual candidate difference and its review or test results. A rejected change remains part of the history even though it was never applied. The record should preserve earlier events when new findings or corrections arrive.

In the library example, a reader should be able to follow the root's lending task, lending's delegation to an implementer, the implementer's candidate and report, a review finding about the five-book limit, the revised candidate, and the parent's integration and application. Parallel work needs identifiable branches and causal links so that its history does not imply a false serial sequence.

Exact replay is a separate question from knowing the recorded inputs and outputs. The trace describes what Markitect actually supplied and observed; missing or unavailable events must remain visible. The current [architecture](../../architecture.md#authority-and-evidence) and [operational records](../../project-operations.md#record-layout-and-cutover) retain their own implemented scope. This design does not claim that all of the proposed detail is already recorded.

The storage format, event granularity, inspection interface and reporting views remain open. An initial view could offer a readable run timeline with drill-down to each Agent's task, context, report and changes.

## Team variants

Each Manager should be configurable and may have a team of AI Agents. Preserve these alternatives until they can be compared:

| Variant | Intended behavior |
| --- | --- |
| Manager implements | The Manager performs implementation itself and organizes the required review and integration. |
| Implementer and reviewer | Agents alternate implementation and review until the reviewer considers the work ready. Completion criteria and handling of unresolved review remain to be designed. |
| Reviewer per Definition | Reviewers examine realization against individual Definitions; their findings feed the integrated result. |
| Teams at the leaves | Only the lowest Managers have executing teams. Higher Managers integrate, review and coordinate their results and send questions or instructions down. |
| Teams at several levels | A Manager may perform local work and integrate delegated child work. |

The common Manager contract should allow these variants without prescribing one team layout. They do not replace the existing product direction for independent review and verification in [DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis).

## Code tests and documentation

Code, tests and documentation each need realization and review. Two organizational variants remain open:

- One hierarchy integrates all artifact forms within each Manager's responsibility.
- The same Package hierarchy is used for separate code, test and documentation branches, each with its own Managers and teams. Coordination must bring their results together.

Separate branches may support specialized review. A shared hierarchy may make coordination within a feature simpler. No variant has been selected. Tests and documentation are realizations to assess against the model, as well as implementation deliverables.

## Artifact registry and coverage

Managers register their artifacts and explain which Definitions they realize. A file may support several Definitions, and a Definition may be realized by several files. Artifact responsibility and the model basis of an artifact are separate information: responsibility identifies the Manager accountable for the output; model links explain why the output exists.

The proposed coverage check compares the repository file inventory with these registrations. Every file must be linked or explicitly ignored. The owner used `.markitectignore` as the proposed ignore name; its spelling and format are not a claim about the current product interface.

For example, the lending Manager could register lending implementation, loan-limit tests and borrowing instructions against the relevant Definitions. The structural check can find an unregistered file. Review must still assess whether a registered file actually fulfills the cited Definitions. Registry format, storage, updates, ownership across separate hierarchies and detailed review evidence remain open.

Different Kinds may lead to different realization and review work. A behavior Definition and an architecture Definition need not produce identical tasks. Such handling belongs to the configurable strategy and uses the Kind's actual contract and content. A Kind name alone does not grant execution rights or new Core semantics.

## Context with need to know

Need to know means keeping Agents focused by avoiding unnecessary context. It is not an information security boundary. Each Agent may receive a context package that supplies much of its prompt. A Manager, implementer and reviewer can receive different task instructions and views while retaining access to needed source material.

Do not begin with aggressive context optimization. Preserve these simple delivery options:

| Option | Starting point |
| --- | --- |
| Bodies from the own Package | Concatenate the bodies of the Package's Definitions, using each filename as a heading. |
| See also | Add the explicitly linked Definitions from other Packages as a compiled section, either as readable content or source pointers. |
| Files to read | Give the Agent the relevant source files or their paths and let it read them directly. |

These options can be combined. For the lending Manager, the first section might contain the local borrowing and loan-limit descriptions. See also could point to the member Definition and book Definition in other Packages.

The initial package can also include the current task, team configuration, registered realization and child results. The Agent can request or read further detail as questions arise. Optimize selection, summaries or retrieval later if actual use shows a need.

Bodies alone are a useful presentation option, but `spec`, Kind and Property contracts also carry binding information. A body view must therefore provide access to those contents or include their needed structured facts. The exact presentation remains open. Link compilation should resolve declared structural references and body links; which reference classes and directions feed See also, and how far it follows them, remain open. A link identifies a connection without proving that every statement at its target applies to the source.

## Missing links between Packages

**Selected direction, 16 October 2026:** Use the strict variant as the baseline and provide an optional Optimizer that proposes missing links. The [owner's selection](sources/manager-realization-20261016.md#strict-links-and-an-optional-optimizer) replaces the earlier open choice between these approaches:

| Approach | Idea | Tradeoff |
| --- | --- | --- |
| Compensating or optimizing | Discovery, broader reading or review looks for relevant statements that explicit links did not expose. | Less reliance on complete authored links, with more interpretation, context and review work. |
| Strict | Require authors to express the cross-Package connections needed for context compilation. | Simpler explicit compilation, with additional authoring and maintenance obligations. |

For example, a members Definition might say that suspended members cannot borrow. The chosen baseline requires the needed connection to lending to be authored. Ordinary context compilation follows explicit connections. The optional Optimizer can discover the missing connection and propose a concrete model edit. It helps improve the model for later tasks; compensating discovery during every ordinary realization is not required by this direction.

The Optimizer's proposed links include their source, target and reason. It may prepare actual edits as a candidate. Review and integration under the project's delegated authority determine which edits become model content. This can follow delegated review without manual confirmation of each link. Finding a possible connection and adopting it are distinct steps; the exact command modes remain open. The Optimizer's task context can include relevant material beyond existing links, since those links are the subject of its investigation. How Managers organize that wider investigation remains to be designed.

Structural validation can check a required link when the strategy defines its form. It cannot generally prove that all relevant prose has been linked. The strict variant still needs a clear authoring rule and a way to assess missing connections. A missing link does not cancel a binding statement. Optimizer suggestions remain distinguishable from authored model facts until incorporated.

The earlier [context and semantic-index proposals](minimal-language/procedure-proposals.md) remain possible material for later work. This north star does not require adopting that machinery before trying simple context delivery.

## Operations through the Manager hierarchy

The owner proposes using the same Manager hierarchy for most substantive operations, including optimization, cleanup, refactoring and bug hunting. Managers receive an operation-specific task, organize their configured teams and integrate the findings or candidates from their area. The root Manager remains an ordinary Manager. Different operations may use different teams and context while sharing the organizational structure.

The owner wants model changes and artifact changes kept in separate operations. A task can read both to understand its work, but its change target is explicit. A model operation updates intended project content; an artifact operation realizes or repairs artifacts against that content. If a model change requires artifact work, the latter is a separate operation against the resulting model. Read-only inspection is also possible.

The following names describe conceptual tasks, not changes to the current CLI. Grouping commands by target, such as `markitect model optimize`, and keeping a flat command surface remain alternatives. Final names are open.

| Task | Change target | Intended work |
| --- | --- | --- |
| Compile | None | Deterministically check language compilability and resolve the model. This task needs no AI Manager traversal and does not judge semantic completeness. |
| Model optimize | Model | Managers inspect model content for missing connections and return justified link proposals or candidate edits for review and integration. |
| Check or verify meaning | None for inspection | Managers and reviewers assess selected model or realization obligations and report their evidence and unresolved findings. Exact scope remains to be defined. |
| Cleanup or refactor | One declared target | Improve either model organization or artifacts in a separately scoped operation. Artifact work stays within the existing model's requirements and permitted freedom. |
| Bughunt | Artifacts | A prepared instruction asks Managers to find and fix bugs against the model, with review and integration. It does not rewrite the model to fit the faulty artifacts. |
| Apply | Artifacts | Reconcile artifacts with the current model, including code, tests and documentation. A preceding model edit is not required. |

Apply's general purpose is reconciliation: assess the existing artifacts against the model and bring them into alignment. Realizing newly changed intent is one case; repairing drift or missing realization against unchanged intent is another. The relevant areas are coordinated through the hierarchy, with review, verification and guarded application.

In the library example, the members and lending Managers could examine the suspended-member rule and prepare a missing connection together; their parent integrates the result. The same hierarchy could later investigate why a suspended member was able to borrow, with a different task and team configuration. This is a shared organizational mechanism, not a requirement that every operation make edits or inspect every Package.

## Shared preparation and custom commands

Most substantive commands can use the same broad sequence: check and load the model, assemble context, give the operation's instruction and configured custom instructions to the root Manager, recursively delegate tasks, then integrate and check the results. Each child receives its parent's task prompt and its own appropriate context and configuration. Custom instructions guide work within the applicable model and task boundaries. Apply is the first operation to make reliable; the other commands remain extensions.

Stable Package context and instruction templates could be prepared in advance for Managers and operations. The concrete task, current artifacts and newly produced review candidates still supply information at execution time. Automatic preparation as the first step of each command and an additional optional compile or prepare command remain variants. There is no selected requirement to precompute every Manager's complete prompt for every possible task.

Bughunt can therefore be a prepared task instruction using the shared mechanism: find bugs and fix them. It does not need a separate management structure. Other commands can supply different instructions and teams through the same structure.

Custom commands are an owner-proposed extension of this idea. A custom command could describe its task prompt, relevant context, configuration and one change target, then use the existing Managers and integration process. The registration format and how custom commands select teams remain open. The declared target should be checked against the returned edits so that separation between model and artifact operations is more than prompt wording.

## Questions for further design

- What constitutes a reliable end-to-end Apply for the library example before adding command extensions?
- Which events and exact inputs must be recorded for the first Apply, and how can a reader follow prompts, reports and changes through delegation and integration?
- What should each Manager report upward, and which details stay in the linked execution history until requested?
- How should a parent express outcomes and constraints without taking over a child's implementation decisions?
- Which artifact details should a higher Manager receive initially, which should it request, and when should it inspect files directly?
- How can an organizational entry point support later Government while preserving the distinct model and artifact change targets?
- What does Manager configuration contain, and which review and integration requirements remain common to all strategies?
- How is one Manager per Package applied to the model root, intermediate Packages and imported vocabularies?
- Which team variants should be tried first, and how is a review loop concluded or escalated?
- Should code, tests and documentation share a hierarchy or have separate ones?
- What is the smallest useful artifact registration, including one accountable owner per output?
- Which simple context presentation should be tried first, including access to structured contracts?
- Which explicit connections does the strict baseline require, and which reference classes and directions supply context?
- How does the optional Optimizer present, review and integrate link suggestions, and how is its investigation scoped?
- Which operations use the hierarchy, what does each return, and how are findings, candidate edits and model updates distinguished?
- How are model and artifact commands named, and how are their change targets checked?
- Should preparation also have its own command, and which parts of context are useful to prepare in advance?
- How are custom commands registered and configured without creating another execution mechanism?

Return to the [canonical-model overview](README.md).

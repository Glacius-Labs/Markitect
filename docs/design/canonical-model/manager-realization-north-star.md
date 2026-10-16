# Manager realization north star

Date: 16 October 2026. Working design direction and open variants from the [owner discussion](sources/manager-realization-20261016.md). The owner selected strict authored links with an optional Optimizer as the context baseline. Other alternatives retain their stated status; recording them does not schedule implementation. The [language specification v1](minimal-language/language-contract.md) remains the owner of the canonical language. The [concept register](../../concepts/register.md) owns accepted product decisions.

## Purpose and direction

The canonical model describes the intended project world. Managers organize its realization as code, tests, documentation and other required artifacts. The intended management structure follows the model's Packages: one Manager per Package, with each Manager responsible for integrating its area. The root Manager is an ordinary Manager at the top of the hierarchy. Project Manager was a descriptive name for that position, not a separate role or Kind. Manager remains the working name.

The realization strategy should be customizable. A Manager's responsibility can stay consistent while its team, tools and way of working vary. The exact configuration format and extension mechanism remain open.

Use a simple library management system as the running example. The following tree illustrates responsibility; it is not an executable model or a prescribed vocabulary:

```text
Library root Manager
├── catalog
├── members
└── lending
    └── reminders
```

The root Manager coordinates the overall result. The lending Manager integrates lending and reminders. Each level has a broader view and normally works with fewer details than the level below it. Results and open questions travel upward; decisions, requests and additional information can travel downward. Details remain available when needed.

The language already provides Package paths, identified Definitions, Kind and Property contracts, binding bodies, structured values, resolved references and fixed model content. These are inputs for a realization strategy. [Packages are namespaces](minimal-language/language-contract.md#packages); management responsibility would be assigned by the strategy. It does not follow automatically from the Core language. The model root itself is not a Package; how the root Manager is bound to it remains to be specified. The treatment of imported library Packages also remains open.

## A common Manager responsibility

The following is the initial conceptual contract developed in the discussion. Its data format remains open.

| Part | Contents | Lending example |
| --- | --- | --- |
| Receives | A task, its Package definitions and relevant contracts, the current realization, runtime configuration, and results or questions from child Managers. | Implement borrowing a book; consider registered membership and a maximum of five simultaneous loans. |
| Owns | Organizing the work and its checks, coordinating dependencies, integrating child results, and exposing missing decisions or conflicts. | Coordinate with members and catalog; combine lending and reminder work into a coherent result. |
| Returns | The integrated candidate, artifact registrations and their Definition links, review and check results, and a readable account of completed work and open issues. | Implementation, tests and usage documentation for borrowing, with their model basis and recorded checks. |

A Manager may return an incomplete result with a clear reason. The root Manager follows the same contract and integrates the whole library system. A Manager's own Package responsibility does not imply that all relevant knowledge lies inside that Package.

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

The following names describe conceptual tasks, not changes to the current CLI. Final command names and mutation modes remain open.

| Task | Intended work |
| --- | --- |
| Compile | Deterministically check language compilability and resolve the model. This task needs no AI Manager traversal and does not judge semantic completeness. |
| Optimize | Managers inspect model content for missing connections and return justified link proposals or candidate edits for review and integration. |
| Check or verify meaning | Managers and reviewers assess selected model or realization obligations and report their evidence and unresolved findings. Exact scope remains to be defined. |
| Cleanup or refactor | Managers improve the realization within the model's existing requirements and permitted freedom, integrating changes across affected Packages. |
| Bughunt | Managers investigate defects against the intended behavior. Whether a task returns findings alone or includes repairs must be explicit. |
| Realize a model change | Managers implement and integrate the changed intent with the required review, verification and guarded application. |

In the library example, the members and lending Managers could examine the suspended-member rule and prepare a missing connection together; their parent integrates the result. The same hierarchy could later investigate why a suspended member was able to borrow, with a different task and team configuration. This is a shared organizational mechanism, not a requirement that every operation make edits or inspect every Package.

## Questions for further design

- What does Manager configuration contain, and which review and integration requirements remain common to all strategies?
- How is one Manager per Package applied to the model root, intermediate Packages and imported vocabularies?
- Which team variants should be tried first, and how is a review loop concluded or escalated?
- Should code, tests and documentation share a hierarchy or have separate ones?
- What is the smallest useful artifact registration, including one accountable owner per output?
- Which simple context presentation should be tried first, including access to structured contracts?
- Which explicit connections does the strict baseline require, and which reference classes and directions supply context?
- How does the optional Optimizer present, review and integrate link suggestions, and how is its investigation scoped?
- Which operations use the hierarchy, what does each return, and how are findings, candidate edits and model updates distinguished?

Return to the [canonical-model overview](README.md).

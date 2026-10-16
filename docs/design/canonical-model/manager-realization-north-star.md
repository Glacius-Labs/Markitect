# Manager realization north star

Date: 16 October 2026. Working design direction and open variants from the [owner discussion](sources/manager-realization-20261016.md). This record preserves the ideas for further design; it does not select every variant or schedule implementation. The [language specification v1](minimal-language/language-contract.md) remains the owner of the canonical language. The [concept register](../../concepts/register.md) owns accepted product decisions.

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

The owner identified two possible approaches. Neither has been selected:

| Approach | Idea | Tradeoff |
| --- | --- | --- |
| Compensating or optimizing | Discovery, broader reading or review looks for relevant statements that explicit links did not expose. | Less reliance on complete authored links, with more interpretation, context and review work. |
| Strict | Require authors to express the cross-Package connections needed for context compilation. | Simpler explicit compilation, with additional authoring and maintenance obligations. |

For example, a members Definition might say that suspended members cannot borrow. The strict approach requires the needed connection to lending to be authored. The compensating approach tries to discover that relevance even without the link.

Structural validation can check a required link when the strategy defines its form. It cannot generally prove that all relevant prose has been linked. The strict variant still needs a clear authoring rule and a way to assess missing connections. Compensating discovery must keep its suggestions and interpretations distinguishable from authored model facts.

The earlier [context and semantic-index proposals](minimal-language/procedure-proposals.md) remain possible material for later work. This north star does not require adopting that machinery before trying simple context delivery.

## Questions for further design

- What does Manager configuration contain, and which review and integration requirements remain common to all strategies?
- How is one Manager per Package applied to the model root, intermediate Packages and imported vocabularies?
- Which team variants should be tried first, and how is a review loop concluded or escalated?
- Should code, tests and documentation share a hierarchy or have separate ones?
- What is the smallest useful artifact registration, including one accountable owner per output?
- Which simple context presentation should be tried first, including access to structured contracts?
- Should missing cross-Package links be handled by a strict rule, compensating discovery, or an explicitly defined combination?

Return to the [canonical-model overview](README.md).

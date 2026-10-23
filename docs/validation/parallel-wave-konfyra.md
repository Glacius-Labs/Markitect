# Sanitized consumer engineering-guidance inventory

## Purpose and boundary

This report is named for its workstream. It records a bounded, read-only inventory of one mature modular software repository while omitting private checkout identity, exact revision, owner names, local paths and source wording. A fixed checkout identity and path/hash/size metadata are retained only in an ignored local manifest; no adopter file contents were copied into Markitect.

The approved scope covered root agent instructions, documentation routers, shared engineering guidance, generated AI-provider mirror metadata and architecture-test path metadata. It did not include business records, customer data, credentials, application-source bodies, work-item records, or a broad review of product-specific requirements. The consumer checkout was clean at the inspected revision. No files, branches, policies, packages or integrations were changed there.

The findings describe the inspected guidance structure. They do not establish that every document, provider output or architecture test is correct, current, or passing.

## Ownership and representation

| Knowledge class | Observed owner and representation | What the evidence establishes |
|---|---|---|
| Normative engineering rules | Shared rules have one declared owner. The architecture and documentation rules are canonical typed YAML resources whose substantive text remains human prose. | Ownership and scope are explicit. Markitect validates the resource structure; the words themselves still need human interpretation. |
| Human narrative and rationale | Documentation routers lead into human-owned contracts and decision records. Decision records explain alternatives and rationale, while implementation requirements are directed to the owning contract. | Navigation is deliberately separate from semantic ownership. Prose is not converted into a machine policy merely because it is stored beside typed resources. |
| Process | Shared workflows own repeatable delivery, architecture-decision and AI-mechanism authoring steps. Skills are concise entrypoints; agent definitions state their role and relevant sources. | The process definition is not intentionally restated in each provider prompt. Human review and authority remain explicit steps. |
| Provider projection | A sampled skill follows a canonical YAML → generated reading view → provider entrypoint path. The provider entrypoints point to canonical definitions. Native command-sandbox rules are a separate provider-owned surface. | The sampled provider surfaces did not copy the skill body. A full output-consistency check was not run. |
| Technical evidence | Architecture-test metadata indicates tests for package boundaries, composition, dependency direction, purity and slice shape. Repository tooling documentation describes project-owned code/documentation checks. | These mechanisms can test concrete implementation structure. Only path/name metadata was inspected; no test bodies or results were reviewed. |
| Unmodelled judgment | Business meaning, whether an architecture rule is desirable, whether evidence is representative, and whether a policy should change remain human decisions. | Neither a green structural check nor generated context establishes semantic truth or owner acceptance. |

The reviewed path metadata counted **66 canonical YAML files in the inspected owner areas**, **90 generated reading views**, and **113 provider-surface files** across three provider trees. The provider totals count files rather than equivalent rule copies, so they are not a duplication ratio. Architecture-boundary path filtering found **26 metadata paths**, including **21 test-source files** and supporting files; this count is not a test execution result.

The sampled representations had a clear canonical source and generated pointers. No contradiction was found in that sample. The inventory did not run the repository's full drift checks or compare every canonical resource with every generated output, so it cannot establish repository-wide consistency.

## Change and migration coverage

The documented maintenance path changes a rule at its owner, updates its linked rationale and navigation where needed, and regenerates generated reading views and provider adapters. It also directs project checks to validate the parts that depend on the code tree. This is a layered maintenance model, not a single machine-owned source for all project knowledge.

The repository documents two separate impact surfaces: ordinary documentation impact is derived from existing Markdown links, while Markitect context and impact operate on the typed resource graph. The documentation explicitly distinguishes navigation links from typed dependencies. This prevents accidental semantic edges, but it means a reviewer may need to interpret more than one impact result. The full behavior and precision of those checks were not exercised in this inventory.

No conversion of narrative architecture rules or tests into a Markitect Domain was attempted. No existing-project migration was performed. The inventory did not measure setup effort, maintenance time, task completion, model calls, tokens, agent rediscovery or missed changes.

## Evidence and limits

| Observation | Evidence quality | Limit and interpretation |
|---|---|---|
| Shared architecture and documentation rules have an explicit owner and narrower documents are not meant to redefine them. | Directly inspected canonical rules and generated reading views. | Limited to the shared rules selected for this inventory. |
| Provider skill entrypoints route to a canonical source; one representative skill was compared across its canonical source, generated view and provider entrypoints. | Direct, bounded sample. | Does not prove every provider output matches its source. |
| Router files are treated as navigation; documentation impact and typed graph impact are distinct. | Directly inspected shared tooling guidance. | The actual checks were not run; no impact precision claim follows. |
| Architecture boundaries have a substantial named test surface. | Tracked path/name metadata only. | Test implementation, coverage, execution and ability to detect a real violation remain unverified. |
| Project-specific rules rely on local checks where implementation structure matters. | Shared tooling guidance plus architecture-test metadata. | The check implementations and full source tree were outside the approved content scope. |
| A single owner model reduces duplicated prose across provider surfaces. | One sample showed generated views and link-only entrypoints. | No measured reduction in total upkeep; canonical YAML and generated outputs still require a functioning renderer/check path. |

## Pressure inventory and alternatives

| Gap or pressure | Evidence | Classification | Preferred alternative | Generic Core need? |
|---|---|---|---|---|
| Selecting safe roots and explaining excluded/generated/ignored/nested content for an existing repository. | This inventory required a human-approved scope and a fixed checkout; current agent instructions route to explicit owners. | Adoption workflow and privacy UX. | Keep selection explicit. If repeated, add a read-only adopter-owned selection adapter that reports roots, exclusions and coverage before content is read. | No. Repository discovery and privacy rules vary by adopter and remain outside semantic policy evaluation. |
| Connecting prose rules to executable code boundaries. | The repository documents separate code/documentation checks and has architecture-test metadata. | Adopter-owned check or adapter. | Keep tests and specialized checks beside the code they understand; expose only typed evidence with a clear statement of what it proves. | No. Core should not learn a source language or infer architecture from file paths. |
| Explaining human narratives to an agent without copying them into each provider file. | The sampled provider entrypoints point to canonical sources and generated views. | Context selection and authoring UX. | Compile only owner-relevant resources and retain links to human contracts; preserve generated pointers and drift checks. | No new primitive is evidenced. |
| Reconciling prose-link impact with typed-resource impact. | Shared tooling guidance describes distinct link-derived and graph-derived outputs. | Documentation workflow and UX. | Label the coverage and cause of each result; do not turn navigation links into typed relations implicitly. | No. The evidence does not show that the two graphs should be unified. |
| Determining whether an observed convention is correct, current or worth adopting. | Canonical rules, tests and review instructions preserve human ownership and acceptance. | Human judgment and governance. | Keep candidate observations separate, include counterexamples and uncertainty, then require owner review and a normal project change. | No. Hashes and policy results cannot establish desirability or authority. |

## Init and Copy Me implications

An existing-project workflow should begin with an explicit owner-approved question and scope, not an automatic crawl. A useful preparation report would identify the fixed source snapshot, selected roots and exclusions, and a coverage denominator before any content is offered for analysis. It should explain treatment of nested repositories, generated/build output, vendored files, symlinks, local modifications, secrets and personal/customer information. Multiple roots need independent identities and provenance.

Copy Me can then receive only the exact reviewed files as fixed evidence, with byte hashes and source identity. Its output should remain a separate candidate: observations, hypotheses, counterexamples, uncertainty and gaps should not become canonical rules or provider instructions automatically. Human adoption still requires deciding which existing owner remains authoritative and making the ordinary project change. This is an adapter and authoring workflow concern; no new graph or policy primitive follows from the inventory.

The consumer's existing project-owned documentation and architecture checks are the right place to assess concrete code conventions. A generic Markitect check could bind their declared inputs and report their stated result, but should not claim to prove runtime dependencies, business meaning or uninspected source behavior.

## Falsification and conclusion

The inspected system already has one-owner rules, human-readable contracts, routers, generated provider pointers and local structural checks. That arrangement may deliver most of the consistency benefit without a richer semantic model. Markitect adds another authored resource set, package/context selection and renderer upkeep; this inventory did not measure whether that upkeep is cheaper than the synchronization it could replace. The sampled provider surfaces show an existing strategy for avoiding body duplication, so provider mirror count alone is not evidence of a Markitect benefit.

This inventory supports **feasibility of a bounded model alongside a mature engineering constitution**, not product utility or a recommendation to migrate the full repository. It found no repeated generic language limitation. The strongest follow-up is a separately authorized, controlled task comparison using a narrow, real engineering change and a fixed, privacy-reviewed context selection. Until such a comparison records task outcomes and maintenance effort, claims of reduced ambiguity, rediscovery, effort, context size, defects or missed updates remain unproven.


## Coordination tooling note

The patch tool used its primary-checkout default even while the shell was directed to an isolated managed worktree, so the initial sanitized draft briefly landed in the primary checkout. The report was copied byte-for-byte into the assigned isolated branch and the primary draft was left for coordinator review. Future parallel work should use an absolute-path write operation and verify the destination worktree before staging. This is a coordination-tooling lesson; it is not a consumer finding or a Markitect Core limitation.

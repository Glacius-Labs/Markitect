# Optional research R01 to R03

Three finite, read-only research chats ran during the Product Readiness program and completed on 9 October 2026 against Designer commit `18eabf96ece65825d8cd9a72c7c03692caf59157`. Root spot-checked their central source and primary-documentation findings. No product edits, builds, provider model calls, installations, Case Studies, runtime-discovery proof or productivity measurements were performed.

**Status.** These are AI research results, not owner decisions. The words "decision", "use", "defer" and "reject" below are the researchers' recommendations. Owner decisions live in the [concept register](../concepts/register.md). The knowledge graph is decided there by [DEC-018](../concepts/register.md#dec-018-the-knowledge-graph-explains-impact), so R02 is background for KG-01 to KG-03. R03's follow-up became AGENT-03 in the [backlog](../work-items/backlog.yaml). R01 has no backlog package.

**Sources.** This page combines three sets of records, moved here on 10 October 2026 when the Product Readiness records were retired:

- the task statements `R01-jev.md`, `R02-knowledge-graph.md` and `R03-agent-guidance.md`;
- the summary `research-results-20261009.md`, all four from `docs/work-items/product-readiness/` at `4852f6d7`;
- Root's detailed results `docs/design/research/results-20261009.md` and the R03 follow-up `docs/design/research/model-bound-guidance-follow-up-20261009.md`, both on `origin/archive/wip/government-assessment-untracked-docs-20261010`.

The summary table, the decisions and the findings are quoted verbatim. Mentions of P03, P04, P06, P07, P08 and Main refer to the closed program; that work is now on main.

## Summary

| Item | Recommendation | Minimum optional later work |
| --- | --- | --- |
| R01 Jev | Defer optional read-only claim/evidence classifier; do not replace compiler/Managers or binding Verify/Apply. | Offline independently labelled claim/evidence cases first; no provider integration or replay is authorized. |
| R02 knowledge graph | Keep current typed graph/YAML; consider a derived Explain/Trace view. No required graph database or universal reasoner. | Source-bound relation/owner/artifact/check explanations; unified evidence trace later if useful. |
| R03 provider assets | Improve current ten skills and short entrypoints. Avoid duplicated Rules/Workflow/Agents packages without demonstrated need. | Fix one actual routing/context-loading problem; provider-specific path rule only if justified. |

> R02 graph identity continuity is separate from ordinary complete code renames already required by P03/P07. R03 real agent execution is still P04/P06, not proved by agent config files.

> Recommendations below are qualified product judgments, not measured superiority. The detailed SDK/client-version/licensing inventory is researcher-reported; no installations or client execution verify it.

## R01: Jev, strongly typed AI

**Question.** Identify the actual Jev product meant by the owner's description "strongly typed AI", verify its primary sources, and evaluate concrete uses against Markitect's model, compiler, Host and adapter responsibilities: typed agent requests and results, model-change proposals, or rule, claim and evidence validation. Distinguish type correctness from semantic correctness, code conformance and human intent. Avoid a second ontology or workflow engine adopted only to use a library.

**Decision.** DEFER an optional read-only claim/evidence classifier; REJECT replacing compiler, Managers, model-change generation or binding Verify/Apply authority with Jev.

**Findings.**

The intended product is TypeSafe AI Jev, currently documented as `jev-1.13.0`. It returns typed choices, rubric scores or boolean-like probabilities, rather than writing code or holding an agent conversation. This is explicitly distinguished by the [vendor's coding-agent guide](https://docs.typesafe.ai/introduction/coding-agents); model/config limits are in [Models](https://docs.typesafe.ai/models).

A possible fit is classifying an already structurally valid rule/claim/exact evidence passage as supporting, contradicting or insufficient and prioritizing it for review. It offers no established advantage over exact deterministic type/reference/ownership/status/digest checks. Current Markitect task reports also need summaries, goals, questions, rework and real candidate/test evidence; a closed probabilistic choice cannot supply all those duties. The provider documents substantive failure modes including literal interpretation and numerical/indirection limits in [Jev 1.13 jaggedness](https://docs.typesafe.ai/model-jaggedness/jev-1.13). Probability/confidence is not independent evidence.

Any future use stays an optional observer after structure validation, with exact source/evidence/question/rubric/model identity and raw distributions/usage. It must not create acceptance or Apply authorization. A new account/key/network service, content handling, maintenance and local calibration would add costs; none are justified by this evaluation alone. SDK/license/service details were inspected by the researcher, not proven installed/self-hosted or contractually available here.

**Smallest optional next step.** An offline labelled specification of twelve existing claim/evidence cases including negation, missing information and adversarial instructions. A later explicitly selected finite shadow replay could compare classification errors, abstention and human review effort to current checks. That replay is not authorized or a Main requirement.

## R02: the canonical model as a knowledge graph

**Question.** Define what a useful full knowledge graph means for this product before comparing formats. Inventory the canonical model's identities, typed relations, provenance, constraints, traversal, versions and links to implementation. Answer how far current source is from that target, using real questions: impact of a concept rename, dependency and ownership navigation, traceability from Work Item to model, change and check, cross-scope consistency, and conflicting requirements.

**Decision.** USE the existing typed canonical graph; a future derived Explain/Trace view is the smallest useful extension. DEFER a general query language, RDF/SPARQL/SHACL and persistent rename identity until a concrete need. REJECT replacing YAML, requiring a graph database for Main, or turning Core into a universal reasoner in this phase.

**Findings.**

Current source has canonical identities, typed definitions/references, edge/property paths and input provenance, plus Manager/Statement/Artifact/Check/Decision vocabulary. Project analysis/context/impact traverse declared dependencies and responsibility; removed relations and unknown impact remain conservative. The desired model and actual candidate/check/review/apply evidence remain distinct. Desired relationships and fixed structural checks do not prove implementation conformance.

The main gap is a single explainable forward/backward chain from Work Item/decision through intention, ownership/dependencies and artifact to exact implementation/check/review/Verify/Apply. Host records provide many of these bindings, but not a unified user query. Name/namespace is part of identity: rename currently appears as remove/add, not explicit continuing identity. Freely written contradictory intent is not formally decided. A functional consistent code rename still belongs to the present P03/P07 workflow; a new graph engine does not.

Useful target: explicit navigation within declared project coverage, with a source/revision/digest, typed relationship, derivation reason and unknowns for every answer. Start with a read-only Explain for one Statement at a fixed snapshot: declared inbound/outbound relations, owner, realized artifacts/checks and path explanations, derived from one canonical model. Do not narrow existing invalidation or invent source-language meaning.

[RDF concepts](https://www.w3.org/TR/rdf11-concepts/) describe a graph model independent of a particular authoring syntax; [SPARQL paths](https://www.w3.org/TR/sparql11-query/#propertypaths), [SHACL](https://www.w3.org/TR/shacl/#validation-report) and [OWL](https://www.w3.org/TR/owl-overview/) are possible later interoperability/query/validation/ontology tools, not evidence that this product requires them. No performance, semantic-quality or effort benefit was measured.

The knowledge-graph implementation plan written alongside R02 (`docs/design/research/knowledge-graph-implementation-20261009.md`) stays on the same archive branch as an input to KG-01.

## R03: provider Rules, Workflows and Agents alongside skills

**Question.** Evaluate the provider-native mechanisms that actually exist for the installed Claude and Codex clients, without inventing symmetric features. Distinguish always-relevant rules, task-discovered skills, workflow guidance, named agent roles, real executable tools and Markitect-hosted Manager and reviewer identity. Compare additional assets against simply improving the existing skills and entrypoints.

**Decision.** USE and refine the existing ten operation skills and short root entrypoints. DEFER general native agent packages/hooks/permission rules; add a path-specific Claude rule only if a concrete routing need demonstrates its usefulness. REJECT symmetric invented Claude/Codex Rules/Workflow trees and treating agent definition files as evidence that a Manager/reviewer ran.

**Findings.**

The fixed source already generates ten skills and shared operating/recovery references. Prefer a small routing correction and demand-loaded references over eagerly loading the whole workflow. Existing P07 owns correct ordinary setup/guidance; this finding can inform that work without creating an additional feature gate.

Claude supports [project memory and path-specific rules](https://code.claude.com/docs/en/memory). Codex [skills](https://learn.chatgpt.com/docs/build-skills) and [AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md) have their own discovery behavior; Codex command-permission rules are not equivalent engineering Markdown rules. The research also inspected current provider agent configuration documents, but did not run discovery or demonstrate version-specific support for this installation. Do not assume provider symmetry or disable normal tools just to simplify guidance.

Host owns actual Manager identity/scope, fresh context, candidate, review and Apply. Skills/rules/agent definitions instruct or configure agents, not prove execution, independence, budget accounting or OS isolation. At this source the runner disables native helpers; fixing that remains P04/P06, not a reason to add an unexecutable agent-template package.

Keep shared workflow text canonically owned once and thin provider-native mappings. If later adding provider-specific assets, use explicit supported capability/version selection, preview/digest/ownership inventories, safe custom-content preservation and negative discovery/CRLF/collision/stale/metadatum tests. Do not globally modify personal permissions or treat prose links as dependency declarations. No measured token/quality benefit or actual discovery PASS follows from this evaluation.

## R03 follow-up: model-bound provider guidance

The R03 researcher returned an additional recommendation. It was recorded as follow-up R03-F01 and is now AGENT-03. The researcher's source observations were not independently re-reviewed; verify them against current main before acting.

> Proposed topic: versioned canonical YAML product defaults and model-bound provider guidance with integrated Init/Onboard. Keep project responsibilities in the existing Manager/Statement/Artifact/Check/Decision model. Package-level YAML defaults have explicit version/content digests; adopting projects bind effective inputs rather than silently changing when the installed binary changes. AGENTS.md/CLAUDE.md/native skills remain derived provider outputs. Do not create another project ontology or executable workflow engine. Host scope, independent roles, budgets and evidence remain product-code responsibilities.

Suggested sequence:

1. Define one source owner and output writer, defaults/update rules, project customization and minimal typed projection metadata.
2. Move existing generic onboarding texts to a validated embedded versioned YAML catalogue; reproduce deterministic existing output first, bind operation IDs to code contracts.
3. Project actually selected Manager instructions, explicit Statements/uses/requires, Artifacts and Checks; keep entrypoints compact and do not infer dependencies from prose.
4. Offer one explicit Preview/Apply plan for Init plus Onboard, including the not-yet-written initial-model snapshot, conflicts, Brownfield preservation and recovery.
5. Manage canonical name/description updates while preserving unknown user frontmatter; compare against previous generated base and handle missing-base conflicts explicitly.
6. Close model-to-provider-output traceability without conflating tool-managed writer ownership with required artifact realization or counting unrelated tool paths as fulfilled.

> The researcher reports that current Preview binds ModelDigest but renderFiles lacks selected model contents; generic guidance lives in Go strings, old frontmatter preservation can hide canonical metadata updates, and Required-Artifact/ToolPaths exclusions need an explicit narrow projection contract.

The follow-up also sketched a later A/B study: one stable main against its single feature descendant, with the same adopting start snapshot, prompts, model, effort, runtime, permissions, resources and evaluation, the setup and projection change as the only treatment, and fresh finite quota and stop rules frozen before any call. That belongs to the PLAY stream and needs the owner's protocol decision (PLAY-07).

## Research chats

R01 `01a12178-516b-78c0-8ebb-47da32643984`, R02 `01a12178-5797-7c41-8acc-8e8231084c79`, R03 `01a12178-64de-7d41-aa82-11b8fb202786`. These chats have ended.

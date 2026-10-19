# Optional research R01 to R03

Three finite, read-only research chats ran during the Product Readiness program and completed on 9 October 2026 against Designer commit `18eabf96ece65825d8cd9a72c7c03692caf59157`. Root spot-checked their central source and primary-documentation findings. No product edits, builds, provider model calls, installations, Case Studies, runtime-discovery proof or productivity measurements were performed.

**Status.** These are AI research results, not owner decisions. The words "decision", "use", "defer" and "reject" below are the researchers' recommendations. Owner decisions live in the [concept register](../concepts/register.md). The knowledge graph is decided there by [DEC-018](../concepts/register.md#dec-018-the-knowledge-graph-explains-impact), so R02 is background for KG-01 to KG-03. R03's follow-up became AGENT-03 in the [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml). R01 has no backlog package.

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

## ECC comparison follow-up - 18 October 2026

**Status:** source-based assistant assessment, recorded at the owner's request. The owner requested documentation of findings and concrete reuse candidates, with accepted directions separated from new proposals and no implementation. Recording this assessment accepts none of its candidates, schedules no work and authorizes no installation, provider run or comparative study. The 9 October research above retains its original source and status; its ten-skill inventory is historical.

### Sources and evidence scope

The inspected Everything Claude Code repository now identifies itself as **ECC**. The fixed inputs are ECC [`76aaa77df6adc5b889e689007bf99b9a4f74af34`](https://github.com/affaan-m/ECC/tree/76aaa77df6adc5b889e689007bf99b9a4f74af34), whose `package.json` declares version 2.2.3, and integrated Markitect [`1d0f31f4a3b2ad5a4478ee387e06ccc323da1890`](https://github.com/Glacius-Labs/Markitect/tree/1d0f31f4a3b2ad5a4478ee387e06ccc323da1890). This is source inspection, not an ECC execution, installation, benchmark or security audit.

| ECC source at the fixed revision | Observation and limit |
| --- | --- |
| [README background](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/README.md#background) | The author reports winning the September 2025 Anthropic x Forum Ventures hackathon with Zenith. That background does not establish the effectiveness of today's ECC collection. |
| [Planning command](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/commands/plan.md) and [plugin manifest](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/.claude-plugin/plugin.json) | Skills, commands, rules and specialized agent instructions describe repeatable engineering procedures. Planning includes a human confirmation gate; native files alone do not prove agent compliance. |
| [Git bypass hook](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/scripts/hooks/block-no-verify.js) and [hook controls](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/scripts/lib/hook-flags.js) | Real code can block Git hook bypasses. Profiles and enable/disable controls affect enforcement; hooks are not an unconditional isolation boundary. |
| [Review workflow](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/workflows/orch-review.workflow.js) and [pilot status](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/workflows/README.md) | Review dimensions emit structured findings; severe findings need evidence and an impact argument. Findings are deduplicated, then independently challenged. Failed dimensions and unverifiable or uncertain blockers prevent clean approval. This native workflow is a pilot, not proof that every installation runs it. |
| [Iterative retrieval guidance](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/skills/iterative-retrieval/SKILL.md) and [retrieval code](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/scripts/lib/context-retrieval.js) | Guidance refines searches after identifying missing context. Separately, executable skill-metadata ranking combines weighted lexical matching and hashed character n-grams without a network or model call. Neither establishes complete project-obligation coverage. |
| [Continuous learning v2](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/skills/continuous-learning-v2/SKILL.md) and [configuration](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/skills/continuous-learning-v2/config.json) | Observations can become scoped behavior notes called instincts, with confidence scores and promotion/evolution procedures. The background observer defaults to disabled and has native-Windows limitations. Persisted instructions do not train model weights; confidence is not project authority. |
| [ECC2 README](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/ecc2/README.md), [evaluation gate](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/scripts/lib/eval-harness/gate.js) and [receipts](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/scripts/lib/eval-harness/receipt.js) | ECC includes session/runtime and evidence code beyond prompts. ECC2 calls itself an alpha; the inspected JavaScript evaluation gate disables candidate execution without verified OS containment. Hashes and signatures bind evidence bytes, not semantic correctness. |

At the Markitect input, [CLI routing](https://github.com/Glacius-Labs/Markitect/blob/1d0f31f4a3b2ad5a4478ee387e06ccc323da1890/internal/adapters/cli/command.go), [bootstrap composition](https://github.com/Glacius-Labs/Markitect/blob/1d0f31f4a3b2ad5a4478ee387e06ccc323da1890/internal/bootstrap/execution.go) and [Apply execution](https://github.com/Glacius-Labs/Markitect/blob/1d0f31f4a3b2ad5a4478ee387e06ccc323da1890/internal/apply/execute.go) already contain execution/publication wiring, although some guide introductions still say Apply is unavailable. These are source observations, not successful complete-Apply or release evidence. Current progress remains with the delivery plan and delivery chats/PRs; this report creates no status mirror.

### Comparison with Markitect

**Interpretation:** the inspected ECC paths organize work primarily around tasks, procedures and agent roles. Markitect's accepted method organizes realization around binding model obligations and Package ownership. Both use context, specialized work, independent review and checks. ECC is therefore a substantial conventional workflow to learn from, not merely a prompt collection or evidence that Markitect's additional structure is useful.

| Concern | ECC in the inspected paths | Markitect's accepted method; not a measured benefit |
| --- | --- | --- |
| Starting authority | Task, project guidance and selected procedure | Committed canonical model; unchanged-intent repairs use that same model |
| Participation and responsibility | Task-dependent agents and workflows | Package-derived Managers and guaranteed participation in the first Apply |
| Context | Skills, rules, retrieval and retained experience | Compiled obligations, explicit relationships and registered realizations |
| Assessment | Tests, review and procedure-specific evidence | Also model-bound candidate assessment, parent integration and guarded publication |

The comparison challenges the assumption that more agent roles or another review loop would differentiate Markitect. The useful question is whether model binding and explicit responsibility reduce missed obligations and human intervention enough to justify model upkeep. No such improvement was measured here.

### Already accepted directions at the inspected revision

These directions come from their existing owners, not from acceptance of ECC:

**Integration note, 19 October 2026:** this table records the accepted basis at the fixed 18 October source above. Later [DEC-056](../concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) replaces mandatory named per-delivery reviewer/arbiter machinery with fresh recursive Verify and a shared-workspace strategy. It also excludes additional Fork, team and Skill functionality from the first demonstration. ECC-C2 must be considered against that current review owner, not used to reinstate the older machinery. All five ECC candidates remain proposals for later consideration; this integration adopts none of them.

| Existing direction | Owner and boundary |
| --- | --- |
| Six native operation skills, compact entrypoints and one maintained content source | [Accepted Init design](../design/project-capability-skills-20261009.md#accepted-init-design--17-october-2026). Packaging and behavior remain implementation work; an outer skill does not become a second Manager scheduler. |
| Binding intent stays in the model | [DEC-033](../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model). Observations and frequently used instructions do not acquire rule authority. The broader [experience mechanism](../concepts/reflections/experience-without-implicit-rules-20261016.md) remains a deferred proposal. |
| Independent review, model-grounded blockers and a dispute-only arbiter | [DEC-043](../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) and [technical risk derivation](../concepts/register.md#dec-045-managers-and-reviewers-derive-technical-risks). Accepted design does not imply implemented named teams or authorize another standing verifier. |
| Targeted reading and visible investigation limits | [DEC-028](../concepts/register.md#dec-028-context-follows-need-to-know-and-starts-simple); [DEC-044](../concepts/register.md#dec-044-commands-reuse-the-manager-organization) specifically accepts broader search for later Optimize. This does not select ECC ranking or extend Optimize's scope to the first Apply. |
| Learn through complete workflows and evaluate quality and human attention | [DEC-038](../concepts/register.md#dec-038-learn-through-complete-workflows), the [working method](../development/working-method.md) and [Measurement](../measurement.md#quality-first-comparative-evaluation). The owner's documentation-plus-one-Agent baseline remains the primary comparison. |

### Concrete reuse candidates - new proposals

The labels below identify assessment references, not delivery packages or an execution queue. Each candidate remains **proposed**. References to an accepted destination describe where later work would belong, not permission to change it now.

| Candidate and concrete ECC material | Possible Markitect use and expected improvement | Cost, boundary and observable evidence needed |
| --- | --- | --- |
| **ECC-C1: native skill guidance.** Planning command, skill descriptions and plugin manifest above. | Use selected task descriptions, demand-loaded references and packaging examples when implementing the accepted six skills. Test whether users reach the right supported operation with less manual explanation. | Maintain one Markitect-owned source and version binding; preserve custom guidance and provider differences. Do not copy ECC's command catalogue, planning confirmation gates or install the whole collection. Observe actual skill selection, preservation and continuation behavior, not just file existence. |
| **ECC-C2: grounded review findings.** Review workflow schemas, evidence/impact prompts and false-positive questions. | Inform the existing finding and review process; use the existing dispute mechanism for a genuinely contested blocker. Aim for actionable findings and fewer unnecessary correction rounds. | ECC's normalization deduplicates by file and evidence text; repeated snippets can describe distinct defects. Any reuse must retain candidate, location, obligation and independent assessments. Do not import its numeric confidence threshold as proof or run a second verifier for every finding by default. Evaluate real and false findings, duplicates, disagreement and missing evidence within the current review boundary. |
| **ECC-C3: progressive supporting context.** Iterative-retrieval guidance and local ranking code. | Begin with a simple question-driven read strategy using existing search; consider the ranking algorithm only after a concrete retrieval failure. Aim to reduce irrelevant reading without hiding required context. | Mandatory compiled obligations and required Managers remain complete. Supporting search grants no new write authority, inferred model link or automatic rule. Observe relevant-context recall, missing-context cases and human intervention alongside reading cost; no new semantic-index subsystem is selected. |
| **ECC-C4: scoped experience proposals.** Instinct records, project scoping and promotion/evolution procedures. | Inform the deferred experience discussion with short source-linked observations and proposed lessons. Aim to avoid recurrence of a demonstrated failure. | Preserve observation, explanation, experience and binding rule as distinct meanings. Scope, evidence, contradictions and supplied revisions matter; frequency or confidence cannot promote a lesson into model authority. Additional storage, retrieval and review have upkeep cost. Later use must demonstrate benefit without cross-project leakage or hidden rules. |
| **ECC-C5: stronger conventional comparison arm.** A frozen, explicitly selected ECC subset and workflow. | Consider a separately labeled supplementary comparison after the required workflow readiness. Give it equivalent visible requirements and assess the same independent outcomes. | This does not replace the owner's primary baseline. Freeze versions, enabled assets, models/effort, permissions, resource limits, setup, memory starting state and intervention policy; count setup and both forms of requirement upkeep. Specify continuation inputs for sequential changes. Repeated independent quality and human-attention evidence is needed; no study is authorized here. |

**Assessment recommendation at the inspected revision:** prefer ECC-C1 and the evidence questions in ECC-C2 as references when their already authorized owning work is implemented. Investigate ECC-C3 only against an observed context problem. Keep ECC-C4 with deferred experience work and ECC-C5 with later comparative evaluation. This is a suggested order of consideration, not scheduling or a change to the Apply implementation assignment. The 19 October integration note above governs later applicability.

The [MIT license](https://github.com/affaan-m/ECC/blob/76aaa77df6adc5b889e689007bf99b9a4f74af34/LICENSE) permits reuse subject to its conditions, including retaining the required notice for copied substantial portions. No ECC code or assets were copied or installed by this documentation update. Bulk installation, a second controller, automatic rule promotion and universal extra review gates are not recommended by this assessment; these are assistant judgments, not new owner rejection decisions.

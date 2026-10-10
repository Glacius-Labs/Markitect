# Product-vision alignment assessment

**Reviewed baseline:** `8bd6602ece50f5e56a715bdaf6e5be37aa760047`, after the integrated first parallel wave. **Published release:** v0.12.0. This is a documentation/priority assessment, not a new feature, release, benefit study or amendment to runtime trust boundaries. [Vision](../vision.md) is the single owner of the adopted thesis; this note records the repository assessment and remaining gaps.

## Repository review

| Question | Finding and change |
|---|---|
| Was the vision already clear? | README, Architecture, Constitution and Refinement already described canonical intent and bounded agent work. The scarce-human-attention thesis and desired human/agent roles were not stated together as the primary purpose. Vision now owns them explicitly. |
| Was it fragmented? | Several apparent purpose/principle owners existed, while older strategy proposals contained additional definitions. The documentation map now identifies one high-level owner; Architecture owns mechanism/boundaries, Constitution invariants, Refinement technical decisions and Roadmap status. Archived proposals remain unchanged. |
| Was README too narrow? | Its emphasis on synchronized documentation/provider surfaces and “AI-facing engineering knowledge” underrepresented generic engineering Domains and the intended operating model. The opening now connects intent to agent work; synchronization becomes a supporting mechanism. |
| Was wording misleading? | The assertion that AI “dramatically increases” throughput and the unconditional context-growth/drift diagram overstated a product assumption. They are replaced by a reference to the canonical thesis and its unproven benefit. Canonical ownership is explicitly limited to modeled facts/mappings. |
| Were streams locally focused? | Scopes and technical exit criteria were clear, but their common product purpose was mostly implicit. The workstream index now relates each major mechanism to a human activity and an evidence limit, without dispatching more work. |
| Were metrics aligned? | Current benchmarks and examples measure bounded technical behavior. Prior comparisons lack human review/attention telemetry and do not establish upkeep payback. Measurement now specifies future intervention, review, quality and total-maintenance observations; no current values are invented. |
| Does architecture support the responsibility split? | Humans own intent/adoption; one normalized model supplies finite checks, context/impact and independent adapters. The architecture supports the foundation. Continuous operation, escalation/acceptance orchestration and reduced human supervision remain unproved consumer workflows. |

## Architectural tensions and deliberate boundaries

| Desired outcome | Present boundary or gap | Disposition |
|---|---|---|
| Agents implement a chosen architecture. | Core proves declared model assertions and explicit specialist evidence, not arbitrary source/business semantics or obedience to instructions. | Preserve the boundary. Select relevant evidence and measure task/implementation quality independently; do not call model conformance complete software correctness. |
| People focus on decisions rather than every routine PR. | Current integration, review, authority and acceptance requirements remain explicit; green checks and recorded owner strings do not authenticate approval. | A project may define delegated routine authority, but any future automated acceptance/escalation design needs a separate reviewed contract. This alignment changes no existing gate. |
| Agents operate continuously. | Markitect exposes bounded CLI/compiler/reconciliation operations; it is not an agent scheduler or background operator. | A consumer runtime can invoke explicit operations. Continuous use does not justify moving orchestration or hidden mutation into Core. |
| Less rediscovery and precise consequence review. | Context relies on declared selection; ADR omissions and broad inputs were observed. Impact intentionally broadens for unknown/configuration/inventory changes. | Keep negative evidence. Improve source-selection/authoring evaluation separately; do not silently narrow sound impact or infer hidden relationships. |
| Less manual synchronization and upkeep. | Existing consumers may already have canonical owners, generated pointers and checks. Markitect introduced additional authored artifacts in the comparison. | Count setup and recurring upkeep, and allow a simpler baseline to outperform Markitect. More projections are not a demonstrated benefit. |
| Easier existing-project adoption. | Selective capture, retention/workspace storage and the Init/Copy Me handoff remain unresolved; observations cannot automatically become policy. | Continue the bounded design priority already recorded by the wave. No automatic discovery, persistent schema, adoption or Core extension is introduced here. |

None of these deliberate Core/authority boundaries conflicts with the vision. They limit what present evidence can prove. The missing operating/acceptance workflow is an implementation and validation gap, not permission to bypass current review, Apply, strict verification or exception semantics. No repeated evidence here justifies a new primitive or broader language.

## Evidence and prioritization

The [real-code pilot](../validation/real-project-adoption-pilot.md) established a bounded contract-evolution and projection-reconciliation workflow, with missing context and broad impact. Its timing and read measurements do not establish productivity, tokens or human review savings. The [matched AGENTS.md comparison](../validation/agents-md-vs-markitect.md) remains inconclusive: more maintained artifacts, no removed independent truth owner or concrete human review step, and no measured review time. The [Konfyra inventory](../validation/parallel-wave-konfyra.md) is read-only consumer feasibility evidence, not adoption or upkeep reduction. The [parallel wave](../validation/parallel-development-wave-1.md) establishes scoped independence and specific adapter fixes, not general user benefit.

The existing bounded streams have a meaningful relationship to the thesis; none needs cancellation merely to align its name with the vision. Prioritize a reviewed selective-evidence handoff and repeated owner-led task/architecture-change evaluation. Deprioritize more provider surfaces, interfaces, live Apply/background machinery and language expansion absent a concrete recurring human activity and evidence that the extra maintenance is justified. The workstream map (retired on 10 October 2026; in git history) held the mechanism mapping, and the [Product Readiness lessons survey](../work-items/surveys/product-readiness-lessons-20261010.md#earlier-preparation-records-3-october) keeps its deprioritization rule; [Measurement](../measurement.md#human-attention-and-delegated-work) owns the future method.

## Scope of this change

README, the agent entrypoint and canonical navigation/purpose descriptions are aligned; the new vision owns the name origin, assumptions, roles, desired operating model and decision filter. Technical architecture, Domain/schema, source behavior, examples, package versions, historical reports, release-managed README blocks and published release artifacts remain unchanged. No productivity experiment or feature wave was run as part of this assessment.

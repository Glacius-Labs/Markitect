# Themes and open questions

This file separates current product direction from unresolved choices. Sources are the current canonical documents at base SHA `5be48ce1ba3f218ccfd0ed696bddf106b9a6ff5e` and the cited chat records; candidate work remains separate.

## Established direction

- **Canonical desired state:** current `docs/vision.md` defines Markitect as the canonical desired-state model of an engineering system. User correction in “Chat” (`01a121f1-b948-7050-ae5d-9921b99db9c0`) says the canonical model should remove manual maintenance and synchronization across separate places.
- **Quality and intent fidelity:** the current vision makes quality and consistency primary. User described omissions, ownership (“what belongs where?”), impact and bounded responsibility; hoped-for improvements are not measured results.
- **Human authority, delegated execution:** current vision gives humans decisions about intent and AI agents bounded implementation. Older design/Overseer chats discuss higher autonomy and less routine supervision, while preserving human decisions and evidence gates. Treat “mostly unattended” as target direction, not present capability.
- **Representations follow intent:** Markdown, provider instructions, checks and adapters serve the product purpose. User rejected reducing the product to making engineering knowledge usable as Markdown; see finding `HIST-MARKITECT-20261009-FORMAT-001`.
- **Current product line:** current docs at base call Classic active on main and Government a separate experiment. A branch or draft does not change this status.

## Open questions

1. **Canonical representation / ontology technology.** YAML typed definitions, Markdown projections, OWL/RDF and knowledge-graph approaches were discussed. “Knowledge Graph” (`01a12178-5797-7c41-8acc-8e8231084c79`) records a changing sequence: stale branch/SHA challenged, comparison of RDF/SPARQL and YAML, initial preference to wait for stable main, then permission to develop a separate variant against an exact main baseline. “Markitect YAML als Ontologie” states YAML and Markdown as a candidate direction, not final selection. Revisit against current source and exact user decision. Front Matter was explicitly discarded.
2. **Ontology granularity and ownership.** What deserves durable canonical ownership versus policy, prose, project-specific projection or external input remains a modeling question. Current vision says not every human concept belongs in a shared foundation and that project-owned policy should remain project-owned unless deliberately generalized.
3. **Audience and positioning.** Marketing chat notes audience is still an open hypothesis. The product purpose should guide any positioning; no target segment or validated willingness-to-pay is established by this scan.
4. **Benefit evidence.** Better intent fidelity, quality, reduced supervision and cleaner repositories are hypotheses or hopes. Current docs require staged evidence; a green check or synthetic example is not a productivity result.
5. **Autonomy boundary.** The desired model aims to reduce routine human scheduling/review, but the user still reserves meaningful decisions, intent changes, exceptions and acceptance. Actual reduction in human effort remains to be measured.
6. **Government relevance.** Government is an experiment, not the current product line. Earlier chats explore the metaphor and mechanics, then redirect toward product purpose; do not infer that Government has been adopted as a product direction.
7. **Current integration readiness.** The capture-time readiness note reports full-suite failure followed by narrow corrections, new full suite pending, and A01/A02/A03 not started. This is not evidence of readiness or product performance and needs a fresh source snapshot.

## Revisit protocol

For any proposed resolution, bind it to a current branch and SHA; identify the direct user decision or mark the proposal as open; name canonical owner and projection/data flow; and separate implementation, technical validation, runtime/adoption evidence and human acceptance.

## Future-use scenarios raised on 9 October

A user-supplied discussion captured in Concepts/Marketing/Ideas imagines Markitect as already finished and explores live operation, recursive managers with configurable freedom or rule strictness, a daily briefing, monitoring/logs/data, a personal chat cockpit, project visualization and gamification. Kubernetes and etcd were mentioned as options to investigate, not selected technologies. The direct timing constraint was to defer detailed planning until existing work packages finish, Case Study numbers exist, Main is clean, and the product is stable/useful. These are therefore brainstorming inputs, not accepted features. See current `docs/design/concepts/discussion-boundaries-20261009.md` in its separate uncommitted worktree and Ideas `IDEA-20261009-018` for the cockpit entry; both are proposals.

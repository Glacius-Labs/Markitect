# Claude cockpit implementation handoff

**Retired assignment, 23 October 2026.** Cockpit and interactive authoring are excluded from the approved V1 scope and removed from active source. The instructions below are historical and grant no work. Follow the [current V1 plan](../implementation-plan.md) and its bounded assignments.

**Status: prepared owner assignment, 20 October 2026.** The [owner commission](../concepts/sources/cockpit-commission-20261020.md) assigns cockpit implementation and visual design to Claude. This documentation delivery implements nothing and starts no session. The [cockpit concept](../design/cockpit.md) owns the requested user experience; the [delivery plan](../implementation-plan.md) owns current stabilization priorities, shared files and integration.

## Assignment and source map

Build the local cockpit as a project working surface, beginning with a complete Apply Run inspector. Own the technical design and presentation. Preserve model authority, ownership, independent Verify and publication/read-back; the interface observes the real process rather than defining a second one.

Start from the integrated Main that contains this handoff, in an isolated worktree and a scoped branch. Read AGENTS.md, the working method, delivery plan, CONTRIBUTING, cockpit concept, DEC-064, accepted Apply workflow and implemented runtime contract. Inspect the actual observation producers/readers and current Run evidence before freezing any UI data contract. The source implementation and these guides may advance after this dated handoff.

| Source | Use |
| --- | --- |
| [Cockpit concept](../design/cockpit.md) | Requested views, interactions, causal explanations, design freedom, sequence and acceptance. |
| [Accepted workflow](../design/canonical-model/manager-apply-workflow.md) | Current Pass, ownership, readiness, results, questions, local teams and independent Verify semantics. |
| [Runtime contract](../design/canonical-model/manager-apply-runtime.md#evidence-and-state) | Actual event/transcript/report/question boundaries and known limits; inspect implementation too. |
| [Code map](code-map.md) | Find owning Go packages and adapters before selecting write paths. |
| [Validation map](../validation/README.md) | Locate reusable real records; dated results retain their build and process identity. |
| [Documentation maintenance](documentation.md) | Keep concept, implemented guides, source statements and evidence distinct. |

The original interactive sketch is optional local reference, not a repository dependency or production scaffold. On its originating Windows host it is `C:/Users/Consiliari/.codex/visualizations/2026/10/20/01a1597c-31cf-7750-8170-25d2b450983d/apply-run-inspector.html`. If unavailable, use the self-contained concept and acceptance scenarios. Its serial Explore/local-review flow and synthetic receipts must not be copied as current Apply semantics.

## Bounded delivery

1. Establish the smallest real read adapter and demonstrate one complete recorded Run. Connect Manager/team organization, causal waits, messages, timeline, candidate/evidence details and replay. Choose the stack and visual direction; do not preserve the sketch's layout merely for fidelity.
2. Add live observation using the same projections, with reconnect, explicit staleness/unknowns, selection pinning and an explicit return to live. Inspection and playback cause no execution side effects.
3. Complete the cockpit entry point with project identity, Run history, outcomes and current owner questions. Add only available start/answer/stop actions through the Orchestrator's real command path. Separate presentation pause from an execution stop. A missing capability needs a named gap and a bounded producer change through its owner, not a fake UI success or another workflow engine.

Reserve actual frontend/adapter/test paths and any shared files against the current delivery plan before writing. Implementation choices within this scope are delegated; no new approval round is needed merely to choose layout or a framework. A change to model authority, execution semantics or the stabilization scope is not a routine UI choice. Update the canonical self-model, exact artifact registry, generated views and implemented guides where the implemented slice actually requires it.

Use relevant local checks and independent complete-diff review, then the current integration policy. Hosted CI is manual unless the assignment or branch protection requires it. Do not start provider series, release, deploy or resume ended sessions merely to obtain demo data. Use existing applicable grants and evidence; keep synthetic, actual history, live observations and owner acceptance separate.

## Startup prompt

```text
Implement and design the Markitect cockpit commissioned in
docs/design/cockpit.md. Read
docs/development/claude-cockpit-handoff-20261020.md and the owner source it links.

You own architecture, frontend/backend choices, responsive presentation and
implementation. The old interactive sketch is a functional reference only;
design a coherent cockpit rather than reproducing its pixels or obsolete process.

Start from fresh integrated Main in an isolated scoped worktree. Follow AGENTS.md,
the working method, delivery plan, CONTRIBUTING and DEC-064's accepted workflow.
Record the full base SHA, clean/dirty state, actual path reservation and the first
bounded slice. Preserve other sessions and the existing Apply stabilization work.

First deliver one complete real recorded Run through an actual read adapter:
Manager/team hierarchy, active work, causal waits, correlated questions/messages,
timeline, candidate differences and independent evidence. Add playback with
Play/Pause, speed, event navigation and directional animated message arrows.
Arrow selection opens the corresponding message; presentation pause never stops
the real Run. Honor keyboard access and reduced motion.

Then add live observation with the same projections, reconnect/staleness handling,
selection pinning and an explicit return to live. Complete the project overview,
Run history/outcome and owner-question surfaces. Wire existing execution actions
only through real Orchestrator commands and acknowledgements.

Use current Plan/Prepare/Implement and independent Verify semantics, exact
candidate bindings, tree-only communication and current event/transcript/report
producers. Do not revive Explore/local-review scheduling or the historical SQLite
proposal. Do not invent receipt states, correlations, findings, success or authority
to fill UI gaps. Define necessary bounded producer work with its existing owner.

Deliver successive complete slices with relevant checks, independent review and
normal Main integration under the current manual-CI policy. Update actual model,
registry and documentation accounting where needed. Do not launch real providers,
release or deploy without an existing applicable grant.

Show the concept's acceptance scenarios. Report separately what is implemented,
tested locally, demonstrated with real history, observed live and still unproved.
Explain what the owner can now do, what blocks progress and the next bounded step.
```

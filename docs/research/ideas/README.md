# Ideas

This is an owner-facing collection of product thoughts and questions, not a product backlog or an adopted roadmap. Entries preserve their source, status, assumptions, hoped-for effects and open questions. A note from another chat is a sourced finding, not a product decision. Current contracts and measured results remain owned by the canonical product documents.

## Index

| ID | Title | Status | Related |
|---|---|---|---|
| IDEA-20261009-001 | One canonical model as the maintained source of intent | Noted; open questions | 005 |
| IDEA-20261009-002 | Typed relationships make ownership and impact visible | Noted; open questions | 001, 003 |
| IDEA-20261009-003 | Hierarchical agents and independent checks cover change | Noted; benefit unproven | 002, 006 |
| IDEA-20261009-004 | Make the gap between what software does and should do inspectable | Noted; open questions | 001, 002 |
| IDEA-20261009-005 | Change intent once, then have the project follow it | Noted; open questions | 001, 002, 003 |
| IDEA-20261009-006 | Bounded structure may let less costly models do useful work | Noted; hypothesis | 003, 007 |
| IDEA-20261009-007 | Reward clear, consistent repositories without creating a maintenance trap | Noted; open questions | 001, 005 |

## Sources and status

The seed entries below come from the user's product statement in the original Markitect discussion, Codex chat `01a121f1-b948-7050-ae5d-9921b99db9c0`. The principal source is turn `01a12217-533b-7633-a10d-d1cd82859e4f`, user message `01a12217-53aa-7733-985c-7b5cd63b92cf`, dated 2026-10-09. One later clarification is in turn `01a12204-e5ab-74a0-ad2d-820b2c6c6006`, user message `01a12204-e60f-78c0-adee-2cf16f2f6b8d`.

The conversation's shared source summary also appears in the Ideas chat setup prompt. It is a summary, not a substitute for the user's wording above. No idea in this index is a product decision, accepted requirement, demonstrated benefit or roadmap commitment.

### IDEA-20261009-001 — One canonical model as the maintained source of intent

- **Source wording (user):** “Markitect versucht das anzugehen durch ein kanonisches Modell, eine Wahrheit, ein Weltbild. Dieses gilt und die Realität bzw das Projekt bzw das Repository soll es widerspiegeln.”
- **Problem:** Relevant intent and rules are spread across project locations, so a change can leave forgotten or conflicting representations.
- **Idea:** Maintain one canonical model of the intended project world and make the repository reflect it.
- **Assumptions:** The model can represent the intent people need to maintain; derived project artifacts can be reconciled against it.
- **Hoped-for effect:** Less manual synchronization and fewer forgotten obligations as a project grows.
- **Open questions:** What is authoritative in the model? How are changes to intent distinguished from corrections to an implementation? What happens when the model itself is incomplete or wrong?
- **Relationships:** IDEA-20261009-002, 004, 005, 007. Related current product direction is documented in `../vision.md`; this note does not amend that document.
- **Status:** Noted; questions open.

### IDEA-20261009-002 — Typed relationships make ownership and impact visible

- **Source wording (user):** “Es gibt ein kanonisches Modell mit einem \"Typsystem\", ein Compiler welcher prüfen kann ob dieses Modell \"kompilierbar\" ist” and “klare Beziehungen zwischen den Dateien im Projekt und was sie eigentlich realisieren oder kurz gesagt: \"Was gehört wozu?\"”.
- **Problem:** Agents have difficulty remembering all relevant files and keeping them consistent; a change's reach is hard to establish from scattered prose.
- **Idea:** Explicit typed relationships between model elements, owners, project files and checks can be used to identify relevant change impact.
- **Assumptions:** The relationships can be authored or maintained with manageable effort; the compiler can check structural validity without claiming semantic truth.
- **Hoped-for effect:** A bounded, explainable impact area that makes omissions harder and identifies what requires review after a change.
- **Open questions:** Which relationships materially improve coverage? How should unknown or weakly modeled connections be surfaced? What can structural compilation prove, and what needs runtime or human evaluation?
- **Relationships:** IDEA-20261009-001, 003, 004, 005. The user's account of this mechanism is an idea, not evidence that complete impact coverage is achieved.
- **Status:** Noted; questions open.

### IDEA-20261009-003 — Hierarchical agents and independent checks cover change

- **Source wording (user):** “Diese Management Hierarchie besteht aus vielen Agenten mit klaren und begrenzten Zielen und Interessen.” Also: “durch die vielen Subagenten wird nichts bei der Untersuchung vergessen und man hat Quasi das 4 Augenprinzip.”
- **Problem:** A single agent may miss affected areas; broad consistency checks can take increasingly long and still miss things.
- **Idea:** Organize work into bounded areas with responsible agents, then use independent examination and an overall consistency check.
- **Assumptions:** Decomposition preserves cross-area obligations; reviewers are independent enough to catch issues; bounded tasks work with capable, less costly models.
- **Hoped-for effect:** Better coverage, fewer overlooked inconsistencies, and useful parallel work.
- **Open questions:** How are overlapping responsibilities and cross-area changes handled? What makes a check genuinely independent? How is “nothing missed” measured without treating model coverage as complete by definition?
- **Relationships:** IDEA-20261009-002, 005, 006. “Nothing is forgotten” is the user's hoped-for mechanism outcome, not a demonstrated guarantee.
- **Status:** Noted; benefit unproven.

### IDEA-20261009-004 — Make the gap between what software does and should do inspectable

- **Source wording (user):** “Ausserdem steht dann noch die Frage im Raum, ist das was es tut denn auch das was es tun soll?”
- **Problem:** Repository changes are difficult to trust without manually inspecting exactly what an agent changed and checking that other rules still hold.
- **Idea:** Keep a clear account of intended behavior and compare the actual project against it through structure, checks and review.
- **Assumptions:** Intent can be expressed precisely enough to assess; evidence can be tied to the specific intent and implementation under review.
- **Hoped-for effect:** More justified trust in coding agents, with less routine human synchronization and inspection.
- **Open questions:** What evidence is sufficient for different kinds of claims? Which judgments remain human decisions? How are unresolved or untestable statements represented?
- **Relationships:** IDEA-20261009-001, 002, 003. Distinct from claiming that a model alone proves implementation correctness.
- **Status:** Noted; questions open.

### IDEA-20261009-005 — Change intent once, then have the project follow it

- **Source wording (user, later clarification):** “In der Markitect Welt pflegt man nicht viele Stellen gleichzeitig und muss sie synchron halten. Man pflegt nur noch eine Stelle und lässt sie anwenden”.
- **Problem:** Updating documentation, code, tests and other representations separately forces the human to coordinate work and slows agent use.
- **Idea:** The human changes intent in one canonical place; the system delegates and checks the resulting updates across the affected project.
- **Assumptions:** Model-to-project application can cover all affected artifacts; reports and unresolved choices make completion visible without handing synchronization back to the user.
- **Hoped-for effect:** A genuine reduction in repeated human maintenance and an agent workflow that scales with project complexity.
- **Open questions:** What counts as “one place” when intent spans domains? How does the system establish closure, handle conflicts and return decisions that cannot be delegated? What user effort remains to correct an incomplete model?
- **Relationships:** IDEA-20261009-001, 002, 003, 007. This clarification sharpens the success criterion; it is not a measured outcome.
- **Status:** Noted; questions open.

### IDEA-20261009-006 — Bounded structure may let less costly models do useful work

- **Source wording (user):** “Das soll es zum einen ermöglichen deutlich günstigere, wobei mittlerweile ähnlich fähige Modelle zu benutzen statt die teuersten High End Modelle draufzuwerfen.”
- **Problem:** Complex, poorly bounded work may require expensive models and extensive supervision.
- **Idea:** Structure, explicit responsibilities and change analysis may let smaller or less costly models take on constrained tasks.
- **Assumptions:** Task boundaries and verification compensate for differences in model capability; coordination and review costs do not erase the savings.
- **Hoped-for effect:** Lower model cost for comparable quality and coverage.
- **Open questions:** Which task types and model classes benefit? What should total cost include (setup, context, retries, review and integration)? Does quality stay comparable on realistic changes?
- **Relationships:** IDEA-20261009-003, 007. Cost savings and capability parity remain hypotheses, not findings.
- **Status:** Noted; hypothesis.

### IDEA-20261009-007 — Reward clear, consistent repositories without creating a maintenance trap

- **Source wording (user):** “Markitect soll für saubere und einheitliche Repositories, mehr Qualität bei der Arbeit und damit größeres Vertrauen in Coding Agents durch Systematik und Struktur und einen sauberen Spec Driven AI First Development Ansatz anbieten der Struktur, Methodik und Ordentliche Definition belohnend macht und nicht zur Aufräumhölle.”
- **Problem:** Repositories become harder to understand and reason about as they grow; rigorous structure can itself become burdensome if people must synchronize many copies.
- **Idea:** Make clear definitions and consistent structure useful during development, so upkeep supports change instead of becoming cleanup and duplicate maintenance.
- **Assumptions:** Structure has direct operational value; tools can keep derived material aligned; the method's upkeep remains proportional to its benefit.
- **Hoped-for effect:** More understandable software, higher quality and justified confidence in agent work, without a “cleanup hell.”
- **Open questions:** What observable user experience demonstrates that structure is rewarding? How much upkeep is acceptable? What would show that the system has shifted work back to people?
- **Relationships:** IDEA-20261009-001, 005, 006. Quality and trust are goals; no productivity or economic benefit is claimed here as proven.
- **Status:** Noted; questions open.

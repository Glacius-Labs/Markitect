# Library

The default that ships with Markitect is a whole spec to start from, in the folder [`plugin/default/`](../../plugin/default/) of Markitect's repository: general types, a way of working, and modules for the work ([DEC-159](../decisions.md#dec-159-the-default-is-a-whole-spec-to-start-from-and-lies-in-the-plugin); [where Main is](../roadmap.md#where-main-is)). It is plain files, which a project copies into its spec folder, as the [project reference](project-reference.md#setting-up-a-project-by-hand) shows; from then on the project owns every file and may change any of it. The library is the folder [`internal/library/`](../../internal/library/), which holds what the tool has built in for the default's module `core`: its reference values and its question catalog, which the tool reads and never copies; [signals](signals.md) describes them. [The language reference](language-reference.md) gives the rules of the language and the commands.

The default's own text calls a project's spec its model. Markitect's pages call it the spec; the two words name the same thing.

## The modules

The README of `plugin/default/` becomes the README of the spec folder. It lists the default's modules under `Contents` and says that a project adds a module beside them for each part of what it builds.

| Module | Holds |
|---|---|
| `core` | The general types, on which the rest of the default builds |
| `way-of-working` | A complete way of working: who works on the project, what they do, and which rules hold |
| `plan` | The features and stories; starts empty |
| `findings` | What deserves attention; starts empty |
| `reports` | What each session and each run left behind; starts empty |
| `briefings` | What reached the humans, in the modules `daily`, `weekly` and `on-request`, which start empty |
| `drops` | Material dropped as it was said; starts empty |

## core

`core` holds general types that every kind of project uses. Its README shows all of them under `Contents`, so every module of a spec may name them. Every type says before its sections what it is and when to take it, and when not, where it is easily confused with another. Under every section it says what belongs there, with a short example. Sections marked *optional* may be left out; a section with an arrow has a property type line and holds references to objects of the type it names ([language reference](language-reference.md#properties-and-their-types)).

| Type | Sections |
|---|---|
| Activity | Participants → Role, Trigger; *Steps*, *Result*, *Notes* |
| Actor | Roles → Role |
| Agent, extends Actor | Roles → Role, Level → Level |
| AI model | Provider, Name |
| Artifact | For, Offers; *Form* |
| Concept | Definition; *Consists of*, *Lifecycle*, *Rules* |
| Decision | Decision, Why; *Alternatives*, *Reopen when* |
| Goal | Goal; *By* |
| Level | AI models → AI model, Effort |
| Person, extends Actor | Roles → Role; *Reach* |
| Role | *Reports to* → Role |
| Rule | Applies to, Rule; *Responsible* → Role, *Example* |
| Term | Meaning |

A role describes the usual way of working, not a permission. What a role does follows from the activities that name it among their participants, and what it answers for from the modules and rules that name it as responsible. Who fills a role stands with the actor, a person or an agent; an agent runs on a level, and the level names the AI model of each provider.

The tool knows the module as the folder `core` directly below the spec folder: there `questions` finds its catalog and `topic` its reference values ([signals](signals.md)). They cover six of its types, concept, decision, goal, role, rule and term. A copy placed elsewhere works as types like any others, without them.

## way-of-working

`way-of-working` is a way of working that a project with one person can run as it is: humans work on the spec, agents bring it into the artifacts, and what strikes on the way flows back through one entrance. Its README says how to read it and how a project changes it. It is written in the types of `core` and declares six types of its own for the work: Report, Watch, which extends Activity, Finding, Feature, Story and Briefing. Its documents are its description; this page only names what its modules are for.

| Module or document | What it is for |
|---|---|
| `set-up.md` | How a project begins to work this way |
| `principles` | The attitude behind everything |
| `circle.md` | The circle: observe, triage, briefing, record, work |
| `team` | The roles, the people and agents who fill them, and their levels |
| `day` | The start of a person's work, and the daily run |
| `observing` | Reports, drops and the watches |
| `triage` | How reports become findings |
| `briefing` | What reaches the humans, and the meetings |
| `recording` | How decisions are carried out |
| `plan` | Features, stories and what holds for all work |
| `planning` | How a person and the Architect work out one level |
| `building` | How a Builder carries out a story |
| `review` | How a fresh Reviewer checks every result |
| `analysis` | How a smell is looked into before anyone fixes it |
| `accepting` | How a person polishes, accepts and releases what has landed |
| `deviations` | What lies outside the usual, and its level |
| `views` | What people see of the work, and the handbook |
| `work` | Where the work lies in the spec, and which way a change takes |

The work lies in the spec, in the modules `plan`, `findings`, `reports`, `briefings` and `drops` beside the modules that say what should hold. These modules name no realization, and what says what should hold links them only to name them, as the rule on where the work lies does, and never in a reference. The tool knows nothing of them: for it, they are documents like any others ([what goes in a spec](../concept/what-goes-in-a-spec.md#the-end-state-not-the-way)).

Each module of the way of working names under `Realized in` the skills that realize its activities, as `.claude/skills/markitect-NAME/` for the provider Claude. Those skills are not built, nor is the skill that sets a project up ([roadmap](../roadmap.md#next)). The way of working names a repository, `main` and branches, because it says how a project really works; a project that works otherwise changes those documents ([DEC-097](../decisions.md#dec-097-the-language-and-the-tool-neither-need-nor-know-version-control)).

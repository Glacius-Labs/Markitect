# Library

The library built into the tool `markitect`: the packages of types, the templates that `apply` and `init` copy into a spec, the skills of the standard that `init` copies into a project, and per package its reference values and question catalog ([where Main is](../roadmap.md#where-main-is)). It ships in the tool's version. A spec owns the copy `apply` or `init` makes and may change it. [The language reference](language-reference.md) gives the rules of the language and the commands. The reference values, `references/PACKAGE.md`, and the question catalogs, `questions/PACKAGE.md`, are read by the tool and never copied; [signals](signals.md) describes them.

## Packages

A package is a folder of types. A type in one package may link to types of another; `apply` copies every package a template needs. Every type says before its sections what it is and when to take it, and when not, where it is easily confused with another. Under every section it says what belongs there, with a short example. Sections marked *optional* may be left out.

### core

| Type | Sections |
|---|---|
| Concept | Definition; *Consists of*, *Lifecycle*, *Rules* |
| Decision | Decision, Why; *Alternatives*, *Reopen when* |
| Goal | Goal; *By* |
| Role | Does; *Does not*, *Reports to* |
| Rule | Applies to, Rule; *Responsible*, *Example* |
| Term | Meaning |

A role forbids nothing: *Does not* names what the role usually leaves to another, and working outside the usual way is a deviation, which is noticed, not blocked.

### software

The types follow the six chapters of a software project ([what goes in a spec](../concept/what-goes-in-a-spec.md#the-content-model)), and the capabilities of a product.

| Type | Sections |
|---|---|
| Software project | Purpose, Context, Domain, Interfaces, Quality, Architecture: each a link to the README of its part |
| Purpose | Problem, Stakeholders, Goals; *Not included* |
| Stakeholder | Who, Wants; *Today* |
| Context | Terms, External systems; *Constraints*, each a link to the rule that imposes it; *Assumptions*, each with who ensures it and what is therefore not built |
| External system | What it is, Expected |
| Domain | Concepts, Operations; *Rules*, *Parts* |
| Capability | For, Offers, Covers; *Parts* |
| Operation | Does, Who, Given; *Refused when*, each condition with its reason and the answer the caller gets; *Returns* |
| Interfaces | Interfaces |
| Interface | For, Offers |
| Quality | Requirements; the nine characteristics of ISO/IEC 25010:2023 as optional sections: *Functional suitability*, *Performance efficiency*, *Compatibility*, *Interaction capability*, *Reliability*, *Security*, *Maintainability*, *Flexibility*, *Safety* |
| Quality requirement | Requirement, Measure; *Matters to* |
| Architecture | Strategy, Components, Decisions, Deployment; *Rules* across the code |
| Component | Responsible for; *Uses* |

### company

| Type | Sections |
|---|---|
| Company | Purpose, Offer, Customers, Team, Processes, Rules; *Parts* |
| Service | Description, Price, Performed by; *Under supervision*, *Duration* |
| Product | Description, Price, Sold by |
| Target group | Description, Buys |
| Process | Trigger, Steps, Result, Responsible; *Involved* |
| Step | Description, Performed by; *Under supervision* |
| Task | Description, Responsible, When; *Checked by* |
| Resource | Description, Responsible |
| Metric | Definition, Target, Source, Responsible |

The responsible, involved and performing parties of a process and its steps are roles or agents, as links.

### factory

The types of the way of working ([the work](../concept/the-work.md)). Roles and rules are of the types of the core, processes and steps of the types of the company; the package holds what they do not cover. Its types of what holds have their documents in the spec, in its `factory` part. Its types of the work have their documents in `.markitect/`, never in the spec, since they change all the time and are no target. `check` checks them against these types ([project reference](project-reference.md#the-documents-of-the-work)).

| Type, in the spec | Sections |
|---|---|
| Agent | Task, Started by, Asks, Model, Changes; *Schedule* |
| Observer | Observes, Source, Worth a statement, Never in the report, Schedule, Model |
| Person | Roles |
| Meeting | Rhythm, Time, Participants, Covers |
| Area | Purpose, Owner, Covers; *Class*, *Rules* |
| Class | Purpose, Connects to, Never, Usually planned by |
| Dashboard | For, Sections |
| Handbook | For, Covers, Form |

The Model section of an Agent and of an Observer holds one line per provider, the provider, a model name as it knows it and the effort, such as `Claude: Sonnet, effort high`, or `as the session` for an agent that takes the model of the session it runs in. An Area is an area of planning, with an owner, the parts of the product and the building blocks of the architecture it covers, as links into the spec, and an optional class. It has no spec of its own and is no part of the spec: a part of the spec is a folder whose README is its table of contents.

| Type, in `.markitect/` | Sections |
|---|---|
| Feature | State, Owner, Origin, Target picture, Stories, Acceptance |
| Story | State, Priority, Depends on, Claimed by, Origin, Target picture, Target changes, Plan, Tasks; *Result* |
| Finding | Type, Importance, Area, What, Source, Context, Why this importance; *Likelihood and impact*, *Outcome* |
| Report | Started by, For, Started, Finished, Outcome; *Item*, *Statements*, *Not observed* |
| Briefing | Needs a decision, Decided for you, Went to work without a decision, Decided yesterday |

## Templates

A template is a folder of documents that name their types by links into the packages. Most leave their sections empty, to be filled; the `factory` template is filled. A folder of a template is a template of its own when its documents link nowhere outside it, so `apply software/domain` applies the domain part alone. A folder whose documents link to its siblings cannot be applied alone.

| Template | Documents |
|---|---|
| `project` | `README.md` and four parts, each a folder with a README: `purpose/`, `context/`, `principles/` and `quality/`. They say why the project exists, what is given from outside, which few rules every decision follows, and how well the result must work. The quality README lists all nine characteristics, so that `check` names each as empty until it says what the project requires of it or why nothing |
| `product` | `README.md` and `handbook.md`, the user handbook, a Handbook. The README says how to build the product by capabilities, folder by folder, each with a README of the type Capability, with its operations, concepts and rules beside it; it has no fixed chapters |
| `architecture` | `README.md`, an Architecture, with `components/`, `decisions/` and `rules/`, each a README that says which documents belong there |
| `factory` | The way of working as a standard, filled, for any project, one with a single person included: `roles/` with one role, the Admin, `agents/` with one document per skill, `observers/` (`deviations`, `dropped-material`, `duplication`, `hotspots`, `plan-items` and `status`), `processes/`, `rules/`, `classes/` (`cross-cutting`, `factory` and `infrastructure`), `dashboards/` (`overview`, `briefing`, `plan` and `my-work`) and `handbook.md`, whose documents link to one another, so that they are applied together. The folders `people/`, `areas/` and `meetings/` hold only a README and stay empty for the project to fill, as do the further roles, the classes of its product and the observers of its own sources |
| `software` | A checklist for `project`, not its outline: `README.md`, a Software project, and one README per chapter: `purpose/`, `context/`, `domain/`, `interfaces/`, `quality/` and `architecture/` |
| `company` | `README.md`, a Company, and READMEs for `offer/`, `processes/`, `rules/` and `team/` that say which documents belong there |

`init` applies the templates `project`, `product`, `architecture` and `factory`, each into a folder of its own name in the spec ([project reference](project-reference.md#init)).

After `apply`, `check` reports nothing but `section.empty` for a template that leaves sections empty: that list is the work still to do. The `factory` template reports nothing.

## Apply

`markitect apply TEMPLATE [TARGET]` copies the template's documents to `TARGET`, a folder below the spec, by default its root.

- **Packages.** Every package a copied document links to, and every package those types link to in turn, is copied whole, byte for byte, to `packages/NAME/` in the spec, unless the spec has that folder already. A package the spec has is never overwritten, and never completed.
- **Links.** Links of the copied documents into the library's packages are rewritten to the spec's copy. Links between the copied documents stay as they are.
- **Refusals.** Nothing is copied when a document's path exists in the spec, when a copied document or a needed package links elsewhere in the library, including a template folder that links to a document outside it when applied alone, or when a package is missing. Every reason is reported.

## The skills of the standard

The fourteen skills of the standard are built into the tool, one folder `markitect-NAME/` with a `SKILL.md` each, apart from the packages and templates: `markitect-accept`, `markitect-analyze`, `markitect-architect`, `markitect-assistant`, `markitect-brief`, `markitect-build`, `markitect-capture`, `markitect-create-observer`, `markitect-finish-planning`, `markitect-housekeeper`, `markitect-observe`, `markitect-record`, `markitect-review` and `markitect-triage`. They are plain files, no documents of the language. Three of them, `markitect-capture`, `markitect-finish-planning` and `markitect-create-observer`, run inside the session that calls them and are no agents of their own; each of the others runs as an agent ([agents](../concept/the-circle.md#agents)).

- **One per Agent document.** The `factory` template holds one Agent document per skill, named like the skill without the prefix, such as `agents/review.md` for `markitect-review`, and each skill is built from its Agent document, the processes and rules that document links and the types it uses. A skill ends with a section `Built from`, which lists these documents by their paths relative to the `factory` part. Its front matter holds the `name` and the `description` that Claude Code and Codex read.
- **`init` copies them.** `markitect init` copies each skill whole and byte for byte into `.claude/skills/markitect-NAME/` and into `.agents/skills/markitect-NAME/` of the project, so that Claude Code and Codex find the same skills ([project reference](project-reference.md#init)). `apply` never copies them, and the project owns its copies.
- **They are artifacts of the spec.** The state maps a skill to the documents it realizes like any other artifact: when the `factory` part changes, the skill is built and confirmed again ([the project and its skills](../concept/the-project.md#the-skills)). Until a confirmation names them, `status` lists the copies as unexplained.

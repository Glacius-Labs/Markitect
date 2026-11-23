# Library

The library built into the tool `markitect`: the packages of types, the templates that `apply` copies into a spec, the types the tool owns, and per package its reference values and question catalog ([where Main is](../roadmap.md#where-main-is)). It ships in the tool's version. A spec owns the copy `apply` makes and may change it. [The language reference](language-reference.md) gives the rules of the language and the commands. The reference values, `references/PACKAGE.md`, and the question catalogs, `questions/PACKAGE.md`, are read by the tool and never copied; [signals](signals.md) describes them.

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

The types follow the six chapters of a software project ([what goes in a spec](../concept/what-goes-in-a-spec.md#the-content-model)).

| Type | Sections |
|---|---|
| Software project | Purpose, Context, Domain, Interfaces, Quality, Architecture: each a link to the README of its part |
| Purpose | Problem, Stakeholders, Goals; *Not included* |
| Stakeholder | Who, Wants; *Today* |
| Context | Terms, External systems; *Constraints*, each a link to the rule that imposes it; *Assumptions*, each with who ensures it and what is therefore not built |
| External system | What it is, Expected |
| Domain | Concepts, Operations; *Rules*, *Parts* |
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

The types of the way of working ([the work](../concept/the-work.md)). Roles and rules are of the types of the core, processes and steps of the types of the company; the package holds what they do not cover. Its types of what holds have their documents in the spec, in its `factory` part. Its types of the work have their documents in `.markitect/`, never in the spec, since they change all the time and are no target.

| Type, in the spec | Sections |
|---|---|
| Agent | Task, Started by, Asks, Model, Changes; *Schedule* |
| Observer | Observes, Source, Worth a statement, Never in the report, Schedule, Model |
| Person | Roles |
| Meeting | Rhythm, Time, Participants, Covers |
| Area | Purpose, Owner; *Class*, *Rules* |
| Class | Purpose, Connects to, Never, Usually planned by |
| Dashboard | For, Sections |
| Handbook | For, Covers, Form |

An Area is an area of planning, with an owner and an optional class. It is no part of the spec: a part of the spec is a folder whose README is its table of contents.

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
| `product` | `README.md` and `handbook.md`, the user handbook, a Handbook. The README says how to build the themes of the product, folder by folder, each with a README of the type Domain; it has no fixed chapters |
| `architecture` | `README.md`, an Architecture, with `components/`, `decisions/` and `rules/`, each a README that says which documents belong there |
| `factory` | The way of working as a standard, filled: `roles/`, `agents/`, `observers/`, `processes/`, `rules/`, `classes/`, `dashboards/` and `handbook.md`, whose documents link to one another, so that they are applied together. The folders `people/`, `areas/` and `meetings/` hold only a README and stay empty for the project to fill, as do the classes of its product and the observers of its own sources |
| `software` | A checklist for `project`, not its outline: `README.md`, a Software project, and one README per chapter: `purpose/`, `context/`, `domain/`, `interfaces/`, `quality/` and `architecture/` |
| `company` | `README.md`, a Company, and READMEs for `offer/`, `processes/`, `rules/` and `team/` that say which documents belong there |

After `apply`, `check` reports nothing but `section.empty` for a template that leaves sections empty: that list is the work still to do. The `factory` template reports nothing.

## Apply

`markitect apply TEMPLATE [TARGET]` copies the template's documents to `TARGET`, a folder below the spec, by default its root.

- **Packages.** Every package a copied document links to, and every package those types link to in turn, is copied whole, byte for byte, to `packages/NAME/` in the spec, unless the spec has that folder already. A package the spec has is never overwritten, and never completed.
- **Links.** Links of the copied documents into the library's packages are rewritten to the spec's copy. Links between the copied documents stay as they are.
- **Refusals.** Nothing is copied when a document's path exists in the spec, when a copied document or a needed package links elsewhere in the library, including a template folder that links to a document outside it when applied alone, or when a package is missing. Every reason is reported.

## The types the tool owns

Three types belong to the tool, since it reads their sections: the report type, `report.md`, the settings type, `settings.md`, and the sample type, `sample.md`. They lie outside `packages/`, so `apply` never copies them, and a spec does not hold them. Their documents lie in `.markitect/` and name no type: the folder says which type they are of. `check` checks them against these types and the forms of their values (reports, settings and control samples).

**The report type.** A report is a document of this type in `.markitect/reports/` ([where Main is](../roadmap.md#where-main-is)).

| Section | Holds |
|---|---|
| Source | Who or what raised it, a job and its role, intake, or the owner, and the day |
| Place | The spec document, artifact or folder it concerns |
| Finding | What is wrong or wanted, with the evidence |
| Occurred or risk | Whether it happened or may happen |
| Violated property | One of the five properties of a good spec, correct, or none, when the spec holds and an artifact deviates from it |
| *Rating* | Impact and likelihood |
| *Actions* | The branch that answers it, or the proposals for the owner |
| *Prevention* | The fixed question or counter-check the action adds |
| *Closing* | How it was closed, and why |

The first five are required, since the job that writes a report knows them. Rating, actions and closing are optional because triage adds them later, and prevention because the job that does the action adds it.

**The settings type.** The settings are a document of this type, `.markitect/settings.md` ([where Main is](../roadmap.md#where-main-is)). Every section is optional.

| Section | Holds |
|---|---|
| *Models* | The model per role |
| *Counter-checks* | The folder of the counter-checks |
| *Cadence* | How often each job runs |
| *Write table* | What each role may write, narrower than the table the tool ships |
| *Tests* | The one command line that runs the project's tests and counter-checks |
| *CI* | The files of the CI definition |

**The sample type.** The result of a control sample is a document of this type in `.markitect/samples/` ([trust](../concept/trust.md#control-samples)).

| Section | Holds |
|---|---|
| Source | The control-sample job, its sampler and the day |
| Copy | The commit or branch of the throwaway copy |
| Planted | The deviations planted, by document |
| Verdicts | The reviewer's verdicts on the copy, by document |

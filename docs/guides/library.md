# Library

The library built into the tool `markitect`: the packages of types, the templates that `apply` copies into a spec, the types the tool owns, and per package its reference values and question catalog ([where Main is](../roadmap.md#where-main-is)). It ships in the tool's version. A spec owns the copy `apply` makes and may change it. [The language reference](language-reference.md) gives the rules of the language and the commands. The reference values, `references/PACKAGE.md`, and the question catalogs, `questions/PACKAGE.md`, are read by the tool and never copied; [signals](signals.md) describes them.

## Packages

A package is a folder of types. A type in one package may link to types of another; `apply` copies every package a template needs. Sections marked *optional* may be left out.

### core

| Type | Sections |
|---|---|
| Concept | Definition; *Consists of*, *Lifecycle*, *Rules* |
| Decision | Decision, Why; *Alternatives*, *Reopen when* |
| Goal | Goal; *By* |
| Role | Does; *Does not*, *Reports to* |
| Rule | Applies to, Rule; *Responsible*, *Example* |
| Term | Meaning |

### software

The area types follow the six areas of a software project ([what goes in a spec](../concept/what-goes-in-a-spec.md#the-content-model)).

| Type | Sections |
|---|---|
| Software project | Purpose, Context, Domain, Interfaces, Quality, Architecture: each a link to the README of its area |
| Purpose | Problem, Stakeholders, Goals; *Not included* |
| Stakeholder | Who, Wants; *Today* |
| Context | Terms, External systems; *Constraints*, each a link to the rule that imposes it; *Assumptions*, each with who ensures it and what is therefore not built |
| External system | What it is, Expected |
| Domain | Concepts, Operations; *Rules*, *Areas* |
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
| Company | Purpose, Offer, Customers, Team, Processes, Rules; *Areas* |
| Service | Description, Price, Performed by; *Under supervision*, *Duration* |
| Product | Description, Price, Sold by |
| Target group | Description, Buys |
| Process | Trigger, Steps, Result, Responsible; *Involved* |
| Step | Description, Performed by; *Under supervision* |
| Task | Description, Responsible, When; *Checked by* |
| Resource | Description, Responsible |
| Metric | Definition, Target, Source, Responsible |

## Templates

A template is a folder of documents that name their types by links into the packages and leave their sections empty. Every folder of a template is a template too, so `apply software/domain` applies the domain area alone.

| Template | Documents |
|---|---|
| `software` | `README.md`, a Software project, and one README per area: `purpose/`, `context/`, `domain/`, `interfaces/`, `quality/` and `architecture/`. The quality README lists all nine characteristics, so that `check` names each as empty until it says what the system requires of it or why nothing |
| `company` | `README.md`, a Company, and READMEs for `offer/`, `processes/`, `rules/` and `team/` that say which documents belong there |

After `apply`, `check` reports nothing but `section.empty`: that list is the work still to do.

## Apply

`markitect apply TEMPLATE [TARGET]` copies the template's documents to `TARGET`, a folder below the spec, by default its root.

- **Packages.** Every package a copied document links to, and every package those types link to in turn, is copied whole, byte for byte, to `packages/NAME/` in the spec, unless the spec has that folder already. A package the spec has is never overwritten, and never completed.
- **Links.** Links of the copied documents into the library's packages are rewritten to the spec's copy. Links between the copied documents stay as they are.
- **Refusals.** Nothing is copied when a document's path exists in the spec, when a copied document or a needed package links elsewhere in the library, including a template folder that links to its parent when applied alone, or when a package is missing. Every reason is reported.

## The types the tool owns

Three types belong to the tool, since it reads their sections: the report type, `report.md`, the settings type, `settings.md`, and the sample type, `sample.md`. They lie outside `packages/`, so `apply` never copies them, and a spec does not hold them. Their documents lie in `.markitect/` and name no type: the folder says which type they are of. `check` checks them against these types and the forms of their values ([reports, settings and control samples](reports.md)).

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

# Library

The library of Markitect, the folder [`internal/library/`](../../internal/library/) of its repository: the default that ships with Markitect, the packages of types and the templates, and per package its reference values and question catalog ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec); [where Main is](../roadmap.md#where-main-is)). The packages and templates are plain files, which a person copies into a spec by hand, as the [project reference](project-reference.md#setting-up-a-project-by-hand) shows; a spec owns its copy and may change it. Only the reference values, `references/PACKAGE.md`, and the question catalogs, `questions/PACKAGE.md`, are built into the tool, which reads them and never copies them; [signals](signals.md) describes them. [The language reference](language-reference.md) gives the rules of the language and the commands.

## Packages

A package is a folder of types. A type in one package may link to types of another, so a spec copies the packages together, in `packages/` below the spec folder. Every type says before its sections what it is and when to take it, and when not, where it is easily confused with another. Under every section it says what belongs there, with a short example. Sections marked *optional* may be left out.

### core

| Type | Sections |
|---|---|
| Concept | Definition; *Consists of*, *Lifecycle*, *Rules* |
| Decision | Decision, Why; *Alternatives*, *Reopen when* |
| Goal | Goal; *By* |
| Role | Does; *Does not*, *Reports to* |
| Rule | Applies to, Rule; *Responsible*, *Example* |
| Term | Meaning |

A role describes the usual way of working, not a permission: *Does not* names what the role usually leaves to another.

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

## Templates

A template is a folder of documents that name their types by links into the packages and leave their sections empty, to be filled. Their links into the packages are relative and assume the library's layout: a template copied to `templates/NAME/` below the spec folder, beside `packages/`, keeps them working, and one copied elsewhere needs its links to the packages adjusted by hand. A folder of a template whose documents link nowhere outside it, except into the packages, can be copied alone, at its place in that layout, such as `templates/software/domain/`; a folder whose documents link to its siblings cannot.

| Template | Documents |
|---|---|
| `project` | `README.md` and four parts, each a folder with a README: `purpose/`, `context/`, `principles/` and `quality/`. They say why the project exists, what is given from outside, which few rules every decision follows, and how well the result must work. The quality README lists all nine characteristics, so that `check` names each as empty until it says what the project requires of it or why nothing |
| `product` | `README.md`, which says how to build the product by capabilities, folder by folder, each with a README of the type Capability, with its operations, concepts and rules beside it; it has no fixed chapters |
| `architecture` | `README.md`, an Architecture, with `components/`, `decisions/` and `rules/`, each a README that says which documents belong there |
| `software` | A checklist for `project`, not its outline: `README.md`, a Software project, and one README per chapter: `purpose/`, `context/`, `domain/`, `interfaces/`, `quality/` and `architecture/` |
| `company` | `README.md`, a Company, and READMEs for `offer/`, `processes/`, `rules/` and `team/` that say which documents belong there |

A fresh copy of a template makes `check` report nothing but `section.empty` for the sections it leaves empty: that list is the work still to do.

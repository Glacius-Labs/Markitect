# Foundations

This page says what Markitect builds on. For each of its solutions it names what is taken, the source it comes from and where Markitect departs from it on purpose. The [vision](vision.md) says why Markitect exists, the [concept](concept/README.md) describes the product, and the [decisions](decisions.md) record what holds. Where a solution only borrows an image, such as Terraform or the compiler, the page calls it an analogy: the idea helps to explain, and no method is taken over. Every source the page names is listed, with a reference a reader can check, at the [end of the page](#sources). The page uses Markitect's own words, such as document, type and module; the [concept](concept/what-markitect-is.md#words) defines them.

## At a glance

| Markitect's solution | Builds on |
|---|---|
| [The spec in the repository](#the-spec-in-the-repository) | Docs as code; Parnas and Clements on documenting the end state |
| [A typed, declarative specification language](#a-typed-declarative-specification-language) | Static typing, declarative languages and specification languages such as TLA+ and Alloy, as analogies; classes, objects and typed properties; Go's packages and internal folders; Martin's acyclic dependencies principle |
| [The spec as an interface](#the-spec-as-an-interface) | Parnas's information hiding |
| [Designing a good spec](#designing-a-good-spec) | ISO/IEC/IEEE 29148; competency questions; Tornhill's hotspots and change coupling |
| [Purpose, world and system](#purpose-world-and-system) | Meyer's Handbook of Requirements and Business Analysis; Zave and Jackson |
| [Behaviour as contracts](#behaviour-as-contracts) | Meyer's Design by Contract |
| [Quality](#quality) | ISO/IEC 25010; the quality attribute scenario |
| [Architecture](#architecture) | arc42; Views and Beyond; architecture decision records; ISO/IEC/IEEE 42010 |
| [The default and the tool](#the-default-and-the-tool) | Terraform's command line and modules, as an analogy |
| [The tool's conventions](#the-tools-conventions) | The output of compilers and linters |

[Related practice](#related-practice) says how Markitect differs from spec-driven development, and [weighed and not taken](#weighed-and-not-taken) what was considered and left out.

## The spec in the repository

**Taken.** Documentation is written, versioned and reviewed with the tools of code, beside the code: Markdown in the repository, changed by pull request (Write the Docs). The spec states the project as it should be now, as if it came from an ideal process, and leaves the way there to the repository's history, as Parnas and Clements (1986) advise for design documents. The same rule holds for Markitect's own pages ([DEC-071](decisions.md#dec-071-main-holds-only-the-current-state)).

**Differs.** In docs as code, documentation sits beside the code and follows it. The spec has authority over the code: where they differ, the code is wrong ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

## A typed, declarative specification language

**Analogy.** A statically typed language reports a contradiction when it compiles, a dynamically typed one only when the program runs into it. A declarative language states what should hold and leaves how to an executor, as SQL leaves the plan to the database and Terraform the steps to its providers. TLA+ (Lamport) and Alloy (Jackson) are specification languages: formal, and never executed. The spec borrows all three images ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)): it is typed, it is declarative with an agent as its executor, and nothing in it is executed. `markitect check` is the type checker, and a borrowed idea, not a guarantee ([DEC-138](decisions.md#dec-138-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).

**Taken.** Types and their instances, as classes and objects: a type declares the properties of its objects, and a checker enforces them ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-and-modules)). A property names its type, as the property of a class does: a property typed by a class holds references to objects of that class or of a class derived from it, and an enumeration fixes the values a property may take ([DEC-157](decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)). A type extends another as a class extends its base class, and inherits its properties without changing their types ([DEC-155](decisions.md#dec-155-a-type-extends-another-with-a-line-of-its-own)). A module shows some of what it holds and keeps the rest internal, as a Go package exports some names, and inside it everything may use what is internal to the modules above it, as Go's internal folders allow ([DEC-156](decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)). Modules do not depend on each other in a cycle, after the acyclic dependencies principle (Martin 2002), as Go forbids a package to import itself, directly or indirectly.

**Differs.** Only the structure is formal. A module shows whole documents, by links in its README, not names marked in each document, and declares no imports: its dependencies follow from type lines, extends lines and references, and a mention never counts. The content is prose that an agent interprets, so the result is checked, not derived. A property holds text, references or fixed values, with no other value forms and no check rules, and everything stays plain Markdown with ordinary links.

## The spec as an interface

**Taken.** A component of software states what it is responsible for and hides the decisions behind it (Parnas 1972). The spec plays that role for a whole project: it declares what is fixed, what someone relies on or what was decided, and leaves free what nobody relies on ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

**Differs.** An interface shows only the outside. The spec also holds decisions about the inside, such as components and technology, since they are decided and so fixed.

## Designing a good spec

**Taken.**

- **Characteristics of good requirements** (ISO/IEC/IEEE 29148): a good spec is necessary, complete, consistent, unambiguous and verifiable, and structured, each fact singular and on its level ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-characteristics-and-shows-them-in-four-sources)). Vague words that the standard names as signs of a weak requirement are one of the signals that say where reading pays off.
- **Competency questions** (Grüninger and Fox 1995): an ontology is judged by the questions it must be able to answer. A spec is questioned the same way, with a catalog of questions that ships with the default, each answer taken from the spec with a citation.
- **Hotspots and change coupling** (Tornhill 2015, 2018): a hotspot is what changes often and costs much; change coupling is what changes together without a link. Both are among the signals of a spec's changes.

**Differs.**

- The characteristics are judged by readers, guided by signals with no fixed thresholds. Each number stands beside the reference value of its type and the rest of the project.
- A hotspot weighs how often a document changed by how many documents link to it, not by its complexity. The tool computes no signal of the changes: an agent reads them from the repository's history.
- Correctness, whether the spec says what is meant, is not among the characteristics a reader checks: only use shows it.

## Purpose, world and system

**Taken.**

- **Goals, environment and system** (Meyer 2022): a description says what the system is for, what holds in the world around it and what the system does. A software project's spec states them apart ([what goes in a spec](concept/what-goes-in-a-spec.md#what-a-software-project-states)).
- **The world is not ours to decide** (Zave and Jackson 1997; Meyer 2022): what is imposed is a constraint, and what is taken for granted so that the system can be simpler is an assumption, with who ensures it. Both are given from outside. What the project chooses is a rule.

**Differs.** Meyer's fourth book, the Project, describes the work: roles, schedule, tasks and risks. The spec leaves out the work that changes it, and keeps what should hold: the processes an operation runs and the way of working of a project.

## Behaviour as contracts

**Taken.** An operation is specified by its precondition, its postcondition and the invariant of its class (Meyer 1992, 1997). In a spec an operation says what it does, what the caller gives and when it is refused, and a concept's rules are its invariants. An example illustrates and fixes nothing, as Meyer (2022) gives scenarios that role.

**Differs.** Each condition of refusal states the answer the caller gets, since a caller relies on which answer it gets ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

## Quality

**Taken.** Quality follows the nine characteristics of ISO/IEC 25010:2023: functional suitability, performance efficiency, compatibility, interaction capability, reliability, security, maintainability, flexibility and safety. Each quality requirement has a measure that decides whether it is met, as a quality attribute scenario ends in a response measure (Bass, Clements and Kazman 2003).

**Differs.** A requirement is one sentence with its measure, not the scenario's six parts.

## Architecture

**Taken.**

- **arc42** (Starke and Hruschka): what its sections on cross-cutting concepts, building blocks, deployment and decisions describe is stated in the spec.
- **Architecture decision records** (Nygard 2011; MADR): a decision is one choice with its reason and the alternatives it rejected.

**Differs.**

- A decision has no status: the spec holds what is decided now, and the repository's history keeps what was superseded. It may say what would reopen it.
- Views are not declared. ISO/IEC/IEEE 42010:2022 and Views and Beyond describe an architecture through views written for their readers; Markitect generates every view from the spec, so each fact is stated once ([DEC-131](decisions.md#dec-131-the-tool-shows-claude-explains)).
- arc42's risks and technical debt have no place: they describe the work, not the system as it should be.

## The default and the tool

**Analogy.** Terraform runs the same command line everywhere and copies modules from a registry into the configuration, which then owns them. Markitect's tool and the module of its default are cut the same way, except that a person copies the module by hand ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)).

**Differs.** The way of working is no process document beside the project and no part of the default: the project states it in its own spec, with general types, and its skills, agents and scripts realize it.

## The tool's conventions

**Taken.** A diagnostic reads `file:line: code: message`, as compilers and linters print it; exit code 0 means all is well, 1 that something was found and 2 that the command could not run; and `--json` prints one document for scripts. Commands are named after what they do ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)).

## Related practice

**Spec-driven development**, as GitHub Spec Kit and Kiro practise it, writes a spec for each feature or fix, plans and implements it, and keeps standing project knowledge apart: Spec Kit's constitution, Kiro's steering files. Markitect keeps one standing spec of the whole project: a change is a change to that spec, which stays current after the change is built. What both share is that the agent works from a written statement of what is wanted rather than a prompt.

## Weighed and not taken

- **An ontology language or SysML v2 as the form of the spec,** and front matter with typed fields. RDF with OWL and SHACL, SysML v2 and TypeScript were weighed. Markdown is what people and agents already write ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-and-modules)).
- **Rules on layers** between modules, as layered architectures have, beyond the rule that forbids a cycle. A module only groups and shows, and a rule names what it applies to by links ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-and-modules)).
- **Declared imports** between modules, as Go's import declarations. They would state again what type lines, extends lines and references already say ([DEC-157](decisions.md#dec-157-a-property-names-its-type-and-modules-depend-on-each-other-without-a-cycle)).
- **Architecture metrics of the spec,** such as coupling, cohesion and abstractness, and fixed thresholds. They measure structure, not whether a spec serves its purpose ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-characteristics-and-shows-them-in-four-sources)).
- **Attributes per requirement**, such as priority, risk, owner and history, which requirements standards keep. They describe the work, not the system, and the spec states only the system as it should be.
- **Views as declared things.** A declared view states the same facts twice ([DEC-131](decisions.md#dec-131-the-tool-shows-claude-explains)).

## Sources

Each source is named as in the text above. ISO standards are cited by their number and title, and by their catalogue page where it is given; their texts are sold by ISO.

**Standards**

- ISO/IEC 25010:2023, *Systems and software engineering — Systems and software Quality Requirements and Evaluation (SQuaRE) — Product quality model*. <https://www.iso.org/standard/78176.html>
- ISO/IEC/IEEE 29148:2018, *Systems and software engineering — Life cycle processes — Requirements engineering*. <https://www.iso.org/standard/72089.html>
- ISO/IEC/IEEE 42010:2022, *Software, systems and enterprise — Architecture description*. <https://www.iso.org/standard/74393.html>
- OMG 2025: *OMG Systems Modeling Language (SysML), Version 2.0*. <https://www.omg.org/spec/SysML/2.0>

**Methods, tools and documentation**

- arc42, by Gernot Starke and Peter Hruschka. <https://arc42.org/overview>
- MADR, Markdown Architectural Decision Records. <https://adr.github.io/madr/>
- Terraform documentation: *Modules*, <https://developer.hashicorp.com/terraform/language/modules>.
- Go 1.4 release notes, *Internal packages*. <https://go.dev/doc/go1.4#internalpackages>
- The Go Programming Language Specification, *Exported identifiers*. <https://go.dev/ref/spec#Exported_identifiers>
- The Go Programming Language Specification, *Import declarations*. <https://go.dev/ref/spec#Import_declarations>
- TLA+, by Leslie Lamport. <https://lamport.azurewebsites.net/tla/tla.html>
- Alloy, by Daniel Jackson and others. <https://alloytools.org/>
- Write the Docs, *Docs as Code*. <https://www.writethedocs.org/guide/docs-as-code/>
- GitHub Spec Kit. <https://github.com/github/spec-kit>
- Kiro, *Specs* and *Steering*. <https://kiro.dev/docs/specs/>, <https://kiro.dev/docs/steering/>

**Literature**

- Bass, L., Clements, P., Kazman, R. *Software Architecture in Practice*, 2nd ed. Addison-Wesley, 2003. ISBN 0-321-15495-9.
- Clements, P., Bachmann, F., Bass, L., Garlan, D., Ivers, J., Little, R., Merson, P., Nord, R., Stafford, J. *Documenting Software Architectures: Views and Beyond*, 2nd ed. Addison-Wesley, 2010. ISBN 978-0-321-55268-6.
- Grüninger, M., Fox, M. S. Methodology for the Design and Evaluation of Ontologies. *Workshop on Basic Ontological Issues in Knowledge Sharing*, IJCAI-95, Montreal, 1995.
- Martin, R. C. *Agile Software Development, Principles, Patterns, and Practices*. Prentice Hall, 2002. ISBN 0-13-597444-5.
- Meyer, B. Applying "Design by Contract". *IEEE Computer* 25(10), 1992, 40–51. doi:[10.1109/2.161279](https://doi.org/10.1109/2.161279)
- Meyer, B. *Object-Oriented Software Construction*, 2nd ed. Prentice Hall, 1997. ISBN 978-0-13-629155-8.
- Meyer, B. *Handbook of Requirements and Business Analysis*. Springer, 2022. doi:[10.1007/978-3-031-06739-6](https://doi.org/10.1007/978-3-031-06739-6)
- Nygard, M. *Documenting Architecture Decisions*, 2011. <https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions>
- Parnas, D. L. On the Criteria To Be Used in Decomposing Systems into Modules. *Communications of the ACM* 15(12), 1972, 1053–1058. doi:[10.1145/361598.361623](https://doi.org/10.1145/361598.361623)
- Parnas, D. L., Clements, P. C. A Rational Design Process: How and Why to Fake It. *IEEE Transactions on Software Engineering* SE-12(2), 1986, 251–257. doi:[10.1109/TSE.1986.6312940](https://doi.org/10.1109/TSE.1986.6312940)
- Tornhill, A. *Your Code as a Crime Scene*. Pragmatic Bookshelf, 2015. ISBN 978-1-68050-038-7.
- Tornhill, A. *Software Design X-Rays*. Pragmatic Bookshelf, 2018. ISBN 978-1-68050-272-5.
- Zave, P., Jackson, M. Four Dark Corners of Requirements Engineering. *ACM Transactions on Software Engineering and Methodology* 6(1), 1997, 1–30. doi:[10.1145/237432.237434](https://doi.org/10.1145/237432.237434)

# Foundations

This page says what Markitect builds on. For each of its solutions it names what is taken, the source it comes from and where Markitect departs from it on purpose. The [vision](vision.md) says why Markitect exists, the [concept](concept/README.md) describes the product, and the [decisions](decisions.md) record what holds. Where a solution only borrows an image, such as Terraform or the compiler, the page calls it an analogy: the idea helps to explain, and no method is taken over. Every source the page names is listed, with a reference a reader can check, at the [end of the page](#sources). The page uses Markitect's own words, such as document, type, package, state and report; the [concept](concept/what-markitect-is.md#words) defines them.

## At a glance

| Markitect's solution | Builds on |
|---|---|
| [The spec in the repository](#the-spec-in-the-repository) | Docs as code; Parnas and Clements on documenting the end state |
| [A typed, declarative specification language](#a-typed-declarative-specification-language) | Static typing, declarative languages and specification languages such as TLA+ and Alloy, as analogies; classes and objects |
| [The spec as an interface](#the-spec-as-an-interface) | Parnas's information hiding |
| [Designing a good spec](#designing-a-good-spec) | ISO/IEC/IEEE 29148; competency questions; Tornhill's hotspots and change coupling |
| [Purpose, world and system](#purpose-world-and-system) | Meyer's Handbook of Requirements and Business Analysis; Zave and Jackson |
| [Behaviour as contracts](#behaviour-as-contracts) | Meyer's Design by Contract |
| [Quality](#quality) | ISO/IEC 25010; the quality attribute scenario |
| [Architecture](#architecture) | arc42; Views and Beyond; architecture decision records; ISO/IEC/IEEE 42010 |
| [The state and `status`](#the-state-and-status) | Requirements traceability with suspect links; Go's test cache; Terraform's state, plan and drift, as an analogy |
| [Verification](#verification) | The V-model and its traceability; independent verification; testing laboratories with control samples; checklists, as an analogy |
| [Reports and actions](#reports-and-actions) | Nonconformity and corrective action in ISO 9001 |
| [Jobs and landing](#jobs-and-landing) | CI with required checks; scheduled bots on branches of their own |
| [The tool's conventions](#the-tools-conventions) | The output of compilers and linters |

[Related practice](#related-practice) says how Markitect differs from spec-driven development, and [weighed and not taken](#weighed-and-not-taken) what was considered and left out.

## The spec in the repository

**Taken.** Documentation is written, versioned and reviewed with the tools of code, beside the code: Markdown in the repository, changed by pull request (Write the Docs). The spec states the project as it should be now, as if it came from an ideal process, and leaves the way there to Git, as Parnas and Clements (1986) advise for design documents. The same rule holds for Markitect's own pages ([DEC-071](decisions.md#dec-071-main-holds-only-the-current-state)).

**Differs.** In docs as code, documentation sits beside the code and follows it. The spec has authority over the code: where they differ, the code is wrong ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

## A typed, declarative specification language

**Analogy.** A statically typed language reports a contradiction when it compiles, a dynamically typed one only when the program runs into it. A declarative language states what should hold and leaves how to an executor, as SQL leaves the plan to the database and Terraform the steps to its providers. TLA+ (Lamport) and Alloy (Jackson) are specification languages: formal, and never executed. The spec borrows all three images ([DEC-119](decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)): it is typed, it is declarative with an agent as its executor, and nothing in it is executed. `markitect check` is the type checker, and a borrowed idea, not a guarantee ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).

**Taken.** Types and their instances, as classes and objects: a type declares the sections of its documents, and a checker enforces them ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)).

**Differs.** Only the structure is formal. The content is prose that an agent interprets, so the result is checked, not derived. A type declares sections only, with no value forms or check rules, and everything stays plain Markdown with ordinary links.

## The spec as an interface

**Taken.** A module states what it is responsible for and hides the decisions behind it (Parnas 1972). The spec plays that role for a whole project: it declares what is fixed, what someone relies on or what was decided, and leaves free what nobody relies on ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

**Differs.** An interface shows only the outside. The spec also holds decisions about the inside, such as modules and technology, since they are decided and so fixed.

## Designing a good spec

**Taken.**

- **Characteristics of good requirements** (ISO/IEC/IEEE 29148): a good spec is necessary, complete, consistent, unambiguous and verifiable, and structured, each fact singular and on its level ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)). Vague words that the standard names as signs of a weak requirement are one of the signals that say where reading pays off.
- **Competency questions** (Grüninger and Fox 1995): an ontology is judged by the questions it must be able to answer. A spec is questioned the same way, with catalogs of questions per package and template, each answer taken from the spec with a citation.
- **Hotspots and change coupling** (Tornhill 2015, 2018): a hotspot is what changes often and costs much; change coupling is what changes together without a declared dependency. Both are among the signals of a spec's changes.

**Differs.**

- The properties are judged by readers, guided by signals with no fixed thresholds. Each number stands beside the reference value of its type and the rest of the project.
- A hotspot weighs how often a document changed by how many documents link to it, not by its complexity. Changes come from Markitect's own history of confirmations, not from Git.
- Correctness, whether the spec says what is meant, is not among the properties a reader checks: only use shows it.

## Purpose, world and system

**Taken.**

- **Goals, environment and system** (Meyer 2022): a description says what the system is for, what holds in the world around it and what the system does. The software package sorts them into Purpose, Context and the areas of the system.
- **The world is not ours to decide** (Zave and Jackson 1997; Meyer 2022): what is imposed is a constraint, and what is taken for granted so that the system can be simpler is an assumption, with who ensures it. Both stand in Context. What the owner chooses is a rule.

**Differs.** Meyer's fourth book, the Project, describes the work: roles, schedule, tasks and risks. The spec leaves out the work that changes it, and keeps what should hold: the processes an operation runs and the way of working of a project.

## Behaviour as contracts

**Taken.** An operation is specified by its precondition, its postcondition and the invariant of its class (Meyer 1992, 1997). In the software package an operation says what it does, what the caller gives and when it is refused, and a concept's rules are its invariants. An example illustrates and fixes nothing, as Meyer (2022) gives scenarios that role.

**Differs.** Each condition of refusal states the answer the caller gets, since a caller relies on which answer it gets ([DEC-122](decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)).

## Quality

**Taken.** Quality follows the nine characteristics of ISO/IEC 25010:2023: functional suitability, performance efficiency, compatibility, interaction capability, reliability, security, maintainability, flexibility and safety. Each quality requirement has a measure that decides whether it is met, as a quality attribute scenario ends in a response measure (Bass, Clements and Kazman 2003).

**Differs.** The characteristics are optional sections of the Quality type, which the template lists empty: `check` names each until it says what it asks of the system or why it asks nothing. A requirement is one sentence with its measure, not the scenario's six parts.

## Architecture

**Taken.**

- **arc42** (Starke and Hruschka): what its sections on cross-cutting concepts, building blocks, deployment and decisions describe is stated in the Architecture area.
- **Views and Beyond** (Clements and others 2010): the state is the implementation allocation, which files realize which document.
- **Architecture decision records** (Nygard 2011; MADR): a decision is one choice with its reason and the alternatives it rejected.

**Differs.**

- A decision has no status: the spec holds what is decided now, and Git keeps what was superseded. It may say what would reopen it.
- Views are not declared. ISO/IEC/IEEE 42010:2022 and Views and Beyond describe an architecture through views written for their readers; Markitect generates every view from the spec, so each fact is stated once ([DEC-131](decisions.md#dec-131-the-tool-shows-claude-explains)).
- arc42's risks and technical debt have no place: they describe the work, not the system as it should be.

## The state and `status`

**Taken.**

- **Requirements traceability** (Gotel and Finkelstein 1994): follow a requirement forwards to what realizes it and backwards to where it came from. Safety and automotive standards require it: DO-178C between requirements, code and tests in both directions (Jacklin 2012), and Automotive SPICE bidirectional traceability across its system and software processes (VDA QMC 2023).
- **Suspect links** (IBM DOORS Next): a change to either end of a link makes it suspect until someone confirms it. A change makes the documents linked to it in both directions suspect, and a reviewer's verdict clears them ([DEC-124](decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)).
- **Go's test cache** (Go documentation): a result is kept with the inputs it depended on and reused while they are unchanged. The state keeps each verdict with the hashes of what it rests on.

**Analogy.** Terraform keeps a state of what exists beside its configuration, shows in a plan what would change, and, when the real infrastructure drifted, imports the change into the configuration or reverts it. The spec is the configuration, the state maps artifacts to documents, `status` is the plan, and a change past the spec is back-filled or reverted ([DEC-130](decisions.md#dec-130-a-change-past-the-spec-is-back-filled)).

**Differs.**

- The trace is plain text in the repository, and there is no database to keep in step. Nothing of it stands in an artifact or in the spec.
- Suspect entries are computed from content hashes, one step of links deep. A person does not clear them: a reviewer agent does, with evidence.
- Every file is mapped to a document or ignored, the code end of a trace in both directions: any other file is unexplained.

## Verification

**Taken.**

- **The V-model**, as Automotive SPICE arranges it (VDA QMC 2023): what is wanted is stated before the code and traced in both directions, and on the right side tests are derived from each level before the code exists. Counter-checks take that side over: a counter-checker derives tests, or questions with expected answers, from the spec alone ([DEC-125](decisions.md#dec-125-trust-is-measured-not-granted)). Where a rule names modules, they are architecture tests in a tool such as ArchUnit.
- **Independent verification:** the one who builds does not judge the result, as DO-178C asks for its higher levels (Jacklin 2012).
- **A testing laboratory** monitors the validity of its results, among other ways with reference materials whose result is known (ISO/IEC 17025:2017). Control samples plant known deviations, and the reviewer's hit rate measures the check.

**Differs.** No person approves anything: trust comes only from facts that can turn out negative without anyone reading. The tests a builder writes are realizations; counter-checks are not, and the builder may not change them.

**Analogy.** A surgical safety checklist lowers complications because a team that knows every step can still skip one (Haynes and others 2009). `status` is such a list, computed from the spec and the state instead of an agent's memory.

## Reports and actions

**Taken.** ISO 9001:2015, clause 10.2, handles a nonconformity in a loop: react to it, find its cause, take a corrective action, and review whether the action worked. Every signal that a change is needed becomes a report; triage sorts and rates it; an action on a branch answers it with its prevention; and a report that recurs at the same place shows that the action did not work ([DEC-128](decisions.md#dec-128-reports-and-actions-hold-everything-that-needs-a-change)).

**Differs.** There are no approvals. The owner is asked only for new decisions, as a choice between prepared proposals, and the system learns which kinds of decision the owner hands over ([DEC-129](decisions.md#dec-129-the-owner-makes-only-new-decisions-and-the-system-learns-which)).

## Jobs and landing

**Taken.** Continuous integration with required checks: a pull request merges when its checks are green, and a check defined on the target branch cannot be skipped by the change. Dependency bots work on a schedule, each on a branch of its own. Every job of Markitect ends with a branch that lands by one rule ([DEC-127](decisions.md#dec-127-one-landing-rule-and-a-spec-change-lands-with-its-realization)).

**Analogy.** Terraform runs the same command line everywhere, with a version the project fixes, while services and CI jobs run it on pull requests. Markitect's tool, jobs and runners are cut the same way ([DEC-132](decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners)).

## The tool's conventions

**Taken.** A diagnostic reads `file:line: code: message`, as compilers and linters print it; exit code 0 means all is well, 1 that something was found and 2 that the command could not run; and `--json` prints one document for scripts. Commands are named after what they do ([DEC-096](decisions.md#dec-096-the-tool-computes-and-checks-a-runner-acts)).

## Related practice

**Spec-driven development**, as GitHub Spec Kit and Kiro practise it, writes a spec for each feature or fix, plans and implements it, and keeps standing project knowledge apart: Spec Kit's constitution, Kiro's steering files. Markitect keeps one standing spec of the whole project: a change is a change to that spec, which stays current after the change is built. What both share is that the agent works from a written statement of what is wanted rather than a prompt. Unlike spec-driven development, Markitect also proves what was verified against the spec: the state binds each verdict to what it rests on.

## Weighed and not taken

- **An ontology language or SysML v2 as the form of the spec,** and front matter with typed fields. RDF with OWL and SHACL, SysML v2 and TypeScript were weighed. Markdown is what people and agents already write, and the simplest form read like a handbook and caught everything mechanical ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)).
- **A module layer** with rules on the direction of links, visibility and binding by place, as programming languages have. Folders carry no rules, and a rule names what it applies to by links ([DEC-121](decisions.md#dec-121-a-spec-is-plain-markdown-with-types-packages-and-templates)).
- **Architecture metrics of the spec,** such as coupling, cohesion and abstractness, and fixed thresholds. They measure structure, not whether a spec serves its purpose ([DEC-123](decisions.md#dec-123-a-good-spec-has-five-properties-and-shows-them-in-four-sources)).
- **Attributes per requirement**, such as priority, risk, owner and history, which requirements standards keep. They describe the work, not the system, and the spec states only the system as it should be.
- **Views as declared things.** A declared view states the same facts twice ([DEC-131](decisions.md#dec-131-the-tool-shows-claude-explains)).

## Sources

Each source is named as in the text above. ISO standards are cited by their number and title, and by their catalogue page where it is given; their texts are sold by ISO.

**Standards**

- ISO 9001:2015, *Quality management systems — Requirements*. Clause 10.2: nonconformity and corrective action.
- ISO/IEC 17025:2017, *General requirements for the competence of testing and calibration laboratories*.
- ISO/IEC 25010:2023, *Systems and software engineering — Systems and software Quality Requirements and Evaluation (SQuaRE) — Product quality model*. <https://www.iso.org/standard/78176.html>
- ISO/IEC/IEEE 29148:2018, *Systems and software engineering — Life cycle processes — Requirements engineering*. <https://www.iso.org/standard/72089.html>
- ISO/IEC/IEEE 42010:2022, *Software, systems and enterprise — Architecture description*. <https://www.iso.org/standard/74393.html>
- OMG 2025: *OMG Systems Modeling Language (SysML), Version 2.0*. <https://www.omg.org/spec/SysML/2.0>
- VDA QMC 2023: *Automotive SPICE Process Assessment / Reference Model*, version 4.0. <https://vda-qmc.de/en/automotive-spice/>
- DO-178C: cited for its traceability and verification requirements through Jacklin 2012.

**Templates, tools and documentation**

- arc42, by Gernot Starke and Peter Hruschka. <https://arc42.org/overview>
- MADR, Markdown Architectural Decision Records. <https://adr.github.io/madr/>
- IBM Engineering Requirements Management DOORS Next, *Link validity*. <https://www.ibm.com/docs/SSYMRC_7.2/com.ibm.jazz.vvc.doc/topics/c_linkval.html>
- Terraform documentation: *State*, <https://developer.hashicorp.com/terraform/language/state>; *terraform plan*, <https://developer.hashicorp.com/terraform/cli/commands/plan>; *Import existing resources*, <https://developer.hashicorp.com/terraform/cli/import>.
- Go documentation: test caching, <https://pkg.go.dev/cmd/go#hdr-Test_packages>.
- TLA+, by Leslie Lamport. <https://lamport.azurewebsites.net/tla/tla.html>
- Alloy, by Daniel Jackson and others. <https://alloytools.org/>
- ArchUnit. <https://www.archunit.org/>
- Write the Docs, *Docs as Code*. <https://www.writethedocs.org/guide/docs-as-code/>
- GitHub Spec Kit. <https://github.com/github/spec-kit>
- Kiro, *Specs* and *Steering*. <https://kiro.dev/docs/specs/>, <https://kiro.dev/docs/steering/>

**Literature**

- Bass, L., Clements, P., Kazman, R. *Software Architecture in Practice*, 2nd ed. Addison-Wesley, 2003. ISBN 0-321-15495-9.
- Clements, P., Bachmann, F., Bass, L., Garlan, D., Ivers, J., Little, R., Merson, P., Nord, R., Stafford, J. *Documenting Software Architectures: Views and Beyond*, 2nd ed. Addison-Wesley, 2010. ISBN 978-0-321-55268-6.
- Gotel, O., Finkelstein, A. An Analysis of the Requirements Traceability Problem. *Proceedings of the First International Conference on Requirements Engineering*, IEEE, 1994, 94–101. doi:[10.1109/ICRE.1994.292398](https://doi.org/10.1109/ICRE.1994.292398)
- Grüninger, M., Fox, M. S. Methodology for the Design and Evaluation of Ontologies. *Workshop on Basic Ontological Issues in Knowledge Sharing*, IJCAI-95, Montreal, 1995.
- Haynes, A. B., and others. A Surgical Safety Checklist to Reduce Morbidity and Mortality in a Global Population. *New England Journal of Medicine* 360(5), 2009, 491–499. doi:[10.1056/NEJMsa0810119](https://doi.org/10.1056/NEJMsa0810119)
- Jacklin, S. A. Certification of Safety-Critical Software Under DO-178C and DO-278A. *AIAA Infotech@Aerospace*, 2012. NASA Ames Research Center. <https://ntrs.nasa.gov/citations/20120016835>
- Meyer, B. Applying "Design by Contract". *IEEE Computer* 25(10), 1992, 40–51. doi:[10.1109/2.161279](https://doi.org/10.1109/2.161279)
- Meyer, B. *Object-Oriented Software Construction*, 2nd ed. Prentice Hall, 1997. ISBN 978-0-13-629155-8.
- Meyer, B. *Handbook of Requirements and Business Analysis*. Springer, 2022. doi:[10.1007/978-3-031-06739-6](https://doi.org/10.1007/978-3-031-06739-6)
- Nygard, M. *Documenting Architecture Decisions*, 2011. <https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions>
- Parnas, D. L. On the Criteria To Be Used in Decomposing Systems into Modules. *Communications of the ACM* 15(12), 1972, 1053–1058. doi:[10.1145/361598.361623](https://doi.org/10.1145/361598.361623)
- Parnas, D. L., Clements, P. C. A Rational Design Process: How and Why to Fake It. *IEEE Transactions on Software Engineering* SE-12(2), 1986, 251–257. doi:[10.1109/TSE.1986.6312940](https://doi.org/10.1109/TSE.1986.6312940)
- Tornhill, A. *Your Code as a Crime Scene*. Pragmatic Bookshelf, 2015. ISBN 978-1-68050-038-7.
- Tornhill, A. *Software Design X-Rays*. Pragmatic Bookshelf, 2018. ISBN 978-1-68050-272-5.
- Zave, P., Jackson, M. Four Dark Corners of Requirements Engineering. *ACM Transactions on Software Engineering and Methodology* 6(1), 1997, 1–30. doi:[10.1145/237432.237434](https://doi.org/10.1145/237432.237434)

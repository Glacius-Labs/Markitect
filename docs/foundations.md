# Foundations

This page says what Markitect builds on. For each of its solutions it names what is taken, the source it comes from and where Markitect departs from it on purpose. The [vision](vision.md) says why Markitect exists, the [concept](concept/README.md) describes the product, and the [decisions](decisions.md) record what holds. Where a solution only borrows an image, such as the compiler, the page calls it an analogy: the idea helps to explain, and no method is taken over. Every source is listed with a reference a reader can check at the [end of the page](#sources).

## At a glance

| Markitect's solution | Builds on |
|---|---|
| [The spec in the repository](#the-spec-in-the-repository) | Docs as code; Parnas and Clements on documenting the end state |
| [Typed documentation with a compiler](#typed-documentation-with-a-compiler) | Static typing and a compiler, as an analogy; modelling languages that declare fields and relations on the type |
| [Packages and links](#packages-and-links) | Parnas's modules and *uses* relation; Go's packages |
| [Designing a good spec](#designing-a-good-spec) | Domain-driven design; Martin's Dependency Rule and package metrics; Tornhill's hotspots and change coupling; the CRAP metric |
| [Purpose, world and system](#purpose-world-and-system) | ISO/IEC/IEEE 29148; Meyer's Handbook of Requirements and Business Analysis; Zave and Jackson |
| [Behaviour as contracts](#behaviour-as-contracts) | Meyer's Design by Contract; use cases |
| [Quality](#quality) | ISO/IEC 25010; the quality attribute scenario |
| [Architecture](#architecture) | arc42; Views and Beyond; architecture decision records; ISO/IEC/IEEE 42010 |
| [Relations](#relations) | SysML v2; the Eclipse S-CORE metamodel |
| [The record and `status`](#the-record-and-status) | Requirements traceability with suspect links; Go's test cache and `go.sum`; declared state, as an analogy |
| [Verification](#verification) | The V-model and its traceability; independent review; checklists, as an analogy |
| [The tool's conventions](#the-tools-conventions) | Go's commands |

[Related practice](#related-practice) says how Markitect differs from spec-driven development, and [weighed and not taken](#weighed-and-not-taken) what was considered and left out.

## The spec in the repository

**Taken.** Documentation is written, versioned and reviewed with the tools of code, beside the code: Markdown in the repository, changed by pull request (Write the Docs). The spec states the project as it should be now, as if it came from an ideal process, and leaves the way there to Git, as Parnas and Clements (1986) advise for design documents. The same rule holds for Markitect's own pages ([DEC-071](decisions.md#dec-071-main-holds-only-the-current-state)).

**Differs.** In docs as code, documentation sits beside the code and follows it. The spec has authority over the code: where they differ, the code is wrong ([DEC-063](decisions.md#dec-063-the-spec-is-the-authority-and-its-defects-go-to-the-owner)).

## Typed documentation with a compiler

**Analogy.** A statically typed language reports a contradiction when it compiles, a dynamically typed one only when the program runs into it. Markitect borrows the image: a Kind is a type, a Definition a declaration, a link an import, and `markitect check` the compiler ([founding intent](vision.md#founding-intent)). Nothing in the spec is executed, and the compiler is a borrowed idea, not a guarantee ([DEC-003](decisions.md#dec-003-the-compiler-is-a-borrowed-idea-and-two-claims-stay-apart)).

**Taken.** A type declares the fields of its instances and the relations they may hold, and a checker enforces them. Every modelling language compared does so, among them SysML v2 (OMG 2025) and the metamodel of Eclipse S-CORE; [DEC-110](decisions.md#dec-110-a-kind-declares-fields-and-parts-and-std-describes-a-whole-project) names the others. A Kind declares which fields its Definitions have, which Kinds a link may name and the form of a value ([DEC-117](decisions.md#dec-117-a-kind-declares-the-form-of-a-fields-value)), and `check` enforces it.

**Differs.** The fields stay in the Markdown text under bold labels, not in a separate schema language, and the statement of a Definition stays a sentence, as none of these languages formalizes the sentence of a requirement either. A new form starts as a note of `vet` before it becomes an error.

## Packages and links

**Taken.** A module hides a decision and states what it is responsible for (Parnas 1972): a Package's anchor says what it is for and what it keeps to itself, and `internal: true` hides a Definition from other Packages. A link is Parnas's *uses* relation (1979): one Definition relies on what another states being right, which is not the same as naming it or calling it. What is not internal is a Package's public surface, as a Go package's exported names are, and a cycle between Packages does not compile, as in Go ([the language](guides/spec-language.md#packages-and-folders)).

**Differs.** The Package tree is both the decomposition and the rule of what may use what: a Definition links up the tree and sideways, never down into a child. Parnas allows loops in *uses* and calls them costly; Markitect refuses them between Packages.

## Designing a good spec

[What makes a good spec](guides/good-spec.md) applies the principles that keep code changeable to the spec.

**Taken.**

- **Bounded contexts and ubiquitous language** (Evans 2003): one Package per area of the domain, in that area's words.
- **Reference by identity** (Vernon 2011): a Definition that only names a record links the record's identity, not the whole record.
- **The Dependency Rule** (Martin 2012, 2017): dependencies point from the specific to the general.
- **Package metrics** (Martin 1994): afferent and efferent coupling, instability I = Ce / (Ca + Ce), abstractness and the distance from the main sequence. Martin's Stable Dependencies Principle, named in his column *Stability*, is what `vet.unstable_dependency` checks: depend in the direction of stability.
- **Hotspots and change coupling** (Tornhill 2015, 2018): what changes often and is depended on by much, and what changes together without a declared dependency. `vet.change_coupling` uses the defaults of Tornhill's tool code-maat: at least 5 revisions each, 5 shared, 30 % coupling, and changes of at most 30 items.
- **CRAP** (Savoia 2011): complexity squared times the cube of what tests leave uncovered, plus complexity, with a line at 30.

**Differs.**

- The unit of change is the Definition, not the Package, so the Package measures point where to look and a Definition's fan-in is the cost.
- Abstractness counts Definitions of Kind `Kind`, since a Kind is the form other Definitions follow, as an interface is for classes. The distance is Martin's normalized form, |A + I − 1|.
- Revisions and co-changes come from Markitect's own history of verdicts, not from Git.
- The CRAP analogue takes the cases the reviewer counted for a Definition in place of cyclomatic complexity, and the share the evidence demonstrates in place of test coverage.

## Purpose, world and system

**Taken.**

- **Goals, environment and system** (Meyer 2022): a description says what the system is for, what holds in the world around it and what the system does. `std` sorts them into `purpose/`, `context/` and the folders of the system.
- **The world is not ours to decide** (Zave and Jackson 1997; Meyer 2022): what is imposed is a Constraint, what is taken for granted so that the system can be simpler is an Assumption, and both live under `context/`. What the owner chooses is a Rule. Statements about the world link only the world, so that nothing stable depends on what changes often.
- **The business case** follows the operational concept of ISO/IEC/IEEE 29148, Annex A: how each Stakeholder works today, its Problems, what each gains and gives, the Conflicts between them and the Alternatives to building the system ([DEC-115](decisions.md#dec-115-std-states-the-business-case-views-are-rendered-not-declared)).

**Differs.** Meyer's fourth book, the Project, describes the work: roles, schedule, tasks and risks. The spec leaves it out, apart from technology that is imposed and the rules of how the project is built and verified. Meyer's effects and invariants of the environment have no Kind of their own; an Invariant in `std` belongs to a Concept of the system.

## Behaviour as contracts

**Taken.** An operation is specified by its precondition, its postcondition and the invariant of its class (Meyer 1992, 1997). In `std` an Operation states what a caller can do to a Concept or ask of it, its Conditions are its preconditions, its Outcomes what holds afterwards, and a Concept's Invariants hold at all times. A Scenario is a path an Actor takes, in numbered steps, as in a use case (Cockburn 2000), and it is a check that specifies nothing: where it and an Operation differ, the Operation holds. Meyer (2022) gives scenarios the same role, as tools to verify and illustrate requirements and never a substitute for them.

**Differs.** A Condition that fails is answered by a refusal code, in a stated order of checking, since a caller relies on which refusal it gets. A Scenario is verified only by running it against what the project delivers.

## Quality

**Taken.** `std` holds one Quality for each characteristic of ISO/IEC 25010:2023: functional suitability, performance efficiency, compatibility, interaction capability, reliability, security, maintainability, flexibility and safety. Each says what the characteristic asks of the system, or why it asks nothing. A Quality states what happens and what the system then does, with a Measure that decides whether it is met, as a quality attribute scenario ends in a response measure (Bass, Clements and Kazman 2003).

**Differs.** A Quality is one sentence with its Measure, not the scenario's six parts; source, stimulus and environment are in the sentence where they matter.

## Architecture

**Taken.**

- **arc42** (Starke and Hruschka): the solution strategy, the cross-cutting rules, the runtime and deployment views and the decisions, each as Kinds under `architecture/`.
- **Views and Beyond** (Clements and others 2010): a Component exists when the system runs, says how many instances it has and when that is fixed, and states what it offers other Components and what it requires of them. A Connector states its protocol. A Deployment provides what each Component needs, so an allocation holds when every Need is provided. The registry is the implementation allocation: which files realize which module.
- **Architecture decision records** (Nygard 2011; MADR): a Decision is one choice with its reason, its Consequences and the Alternatives it rejected.
- **ISO/IEC/IEEE 42010:2022**: a Concern is one thing that matters to a Stakeholder.

**Differs.**

- A Decision has no status. The spec holds only what is decided now, and Git keeps what was superseded. Its reason does not bind: a reviewer cannot judge a reason against code.
- Views are not declared. 42010 and Views and Beyond describe an architecture through views written for their readers; `std` holds Kinds for things only, and a view such as a context diagram is rendered from the Definitions, so that each fact is stated once ([DEC-115](decisions.md#dec-115-std-states-the-business-case-views-are-rendered-not-declared)).
- arc42's risks and technical debt have no place: they describe the work, not the system as it should be.

## Relations

**Taken.** The names of relations come from SysML v2 (OMG 2025) and the S-CORE metamodel: an Operation **Satisfies** a Goal or a Quality, and a Rule is **Derived from** what is behind it. The one who acts in a Scenario is the **Actor**, as in a use case.

**Differs.** A relation is a labelled field that holds links, checked by the Kinds it may name. There are no link keywords, and a link is never decorative ([DEC-104](decisions.md#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them)).

## The record and `status`

**Taken.**

- **Requirements traceability** (Gotel and Finkelstein 1994): follow a requirement forwards to what realizes it and backwards to where it came from. Safety standards require it: DO-178C between requirements, code and tests in both directions (Jacklin 2012), and Automotive SPICE bidirectional traceability across its system and software processes (VDA QMC 2023).
- **Suspect links** (IBM DOORS Next): a change to either end of a link makes it suspect until someone confirms it. `status` lists what a change reaches, and a reviewer's `ok` with its evidence clears it ([DEC-099](decisions.md#dec-099-a-record-of-what-was-verified)).
- **Go's test cache and `go.sum`** (Go documentation): a result is kept with the inputs it depended on and reused while they are unchanged; a lock file holds the hash of each dependency. `.markitect/review.sum` keeps each verdict with the hash of every input, and the lock pins each Library by its digest.

**Analogy.** Kubernetes keeps a declared desired state beside the observed one, and controllers move one towards the other. The spec is the declared state, and a build reconciles the project with it ([what the spec is for](vision.md#founding-intent)).

**Differs.**

- The trace is plain text in the repository: the registry is the matrix, and there is no database to keep in step.
- Suspect items are computed from content hashes, one step of links deep, together with what binds an item by its place ([DEC-113](decisions.md#dec-113-what-binds-is-an-input-of-what-it-binds)). A person does not clear them: a reviewer agent does, with evidence.
- Every file is accounted for ([DEC-006](decisions.md#dec-006-every-file-is-accounted-for)), the code end of a trace in both directions: a file that realizes nothing is an error.

## Verification

**Taken.** The left side of the V-model: requirements stated before the code, on levels from the stakeholders' needs down to detailed design, and traced in both directions. The separation of roles comes from human teams: the one who builds does not judge the result, and the one who judges records the verdict ([where trust comes from](vision.md#where-trust-comes-from)).

**Differs.** The right side of the V-model derives tests from each level before the code exists. In Markitect a fresh reviewer verifies each Definition against the code and runs the project's tests as evidence; the tests themselves are a realization that agents write ([DEC-099](decisions.md#dec-099-a-record-of-what-was-verified)).

**Analogy.** A surgical safety checklist lowers complications because a team that knows every step can still skip one (Haynes and others 2009). `status` is such a list, computed from the spec and the record instead of an agent's memory.

## The tool's conventions

**Taken.** `vet`, `list`, `doc` and `tidy` do what their Go namesakes do (Go documentation): `go vet` reports suspicious constructs and is no guarantee, and `markitect tidy` brings the lock in line with the manifest as `go mod tidy` does for a module. A diagnostic reads `file:line: code: message`, and exit codes follow Go's. [What makes a good spec](guides/good-spec.md) is written in the manner of *Effective Go*: practical advice, not a rule set.

## Related practice

**Spec-driven development**, as GitHub Spec Kit and Kiro practise it, writes a spec for each feature or fix, plans and implements it, and keeps standing project knowledge apart: Spec Kit's constitution, Kiro's steering files. Markitect keeps one standing spec of the whole project: a change is a change to that spec, which stays current after the change is built. What both share is that the agent works from a written statement of what is wanted rather than a prompt.

## Weighed and not taken

- **An ontology language or SysML v2 as the form of the spec.** RDF with OWL and SHACL, SysML v2 and TypeScript were weighed. Markdown is what people and agents already write, and a spec of this form translates to those languages by rule ([DEC-110](decisions.md#dec-110-a-kind-declares-fields-and-parts-and-std-describes-a-whole-project)).
- **Attributes per requirement**, such as priority, risk, owner and history, which requirements standards keep. They describe the work, not the system, and the spec states only the system as it should be.
- **Views as declared things.** A Kind per view states the same facts twice ([DEC-115](decisions.md#dec-115-std-states-the-business-case-views-are-rendered-not-declared)).

## Sources

Each source is named as in the text above. ISO standards are cited by their catalogue page; their texts are sold by ISO.

**Standards**

- ISO/IEC 25010:2023, *Systems and software engineering — Systems and software Quality Requirements and Evaluation (SQuaRE) — Product quality model*. <https://www.iso.org/standard/78176.html>
- ISO/IEC/IEEE 29148:2018, *Systems and software engineering — Life cycle processes — Requirements engineering*. Annex A: system operational concept. <https://www.iso.org/standard/72089.html>
- ISO/IEC/IEEE 42010:2022, *Software, systems and enterprise — Architecture description*. <https://www.iso.org/standard/74393.html>
- OMG 2025: *OMG Systems Modeling Language (SysML), Version 2.0*. <https://www.omg.org/spec/SysML/2.0>
- VDA QMC 2023: *Automotive SPICE Process Assessment / Reference Model*, version 4.0. <https://vda-qmc.de/en/automotive-spice/>
- DO-178C: cited for its traceability requirements through Jacklin 2012.

**Templates, tools and documentation**

- arc42, by Gernot Starke and Peter Hruschka. <https://arc42.org/overview>
- Eclipse S-CORE, metamodel of the docs-as-code tooling. <https://github.com/eclipse-score/docs-as-code/blob/main/src/extensions/score_metamodel/metamodel.yaml>
- MADR, Markdown Architectural Decision Records. <https://adr.github.io/madr/>
- code-maat, by Adam Tornhill. <https://github.com/adamtornhill/code-maat>
- IBM Engineering Requirements Management DOORS Next, *Link validity*. <https://www.ibm.com/docs/SSYMRC_7.2/com.ibm.jazz.vvc.doc/topics/c_linkval.html>
- Go documentation: test caching, <https://pkg.go.dev/cmd/go#hdr-Test_packages>; `go.sum`, <https://go.dev/ref/mod#go-sum-files>; `go vet`, <https://pkg.go.dev/cmd/vet>; *Effective Go*, <https://go.dev/doc/effective_go>.
- Kubernetes documentation: *Controllers*, <https://kubernetes.io/docs/concepts/architecture/controller/>; *Objects in Kubernetes*, <https://kubernetes.io/docs/concepts/overview/working-with-objects/>.
- Write the Docs, *Docs as Code*. <https://www.writethedocs.org/guide/docs-as-code/>
- GitHub Spec Kit. <https://github.com/github/spec-kit>
- Kiro, *Specs* and *Steering*. <https://kiro.dev/docs/specs/>, <https://kiro.dev/docs/steering/>

**Literature**

- Bass, L., Clements, P., Kazman, R. *Software Architecture in Practice*, 2nd ed. Addison-Wesley, 2003. ISBN 0-321-15495-9.
- Clements, P., Bachmann, F., Bass, L., Garlan, D., Ivers, J., Little, R., Merson, P., Nord, R., Stafford, J. *Documenting Software Architectures: Views and Beyond*, 2nd ed. Addison-Wesley, 2010. ISBN 978-0-321-55268-6.
- Cockburn, A. *Writing Effective Use Cases*. Addison-Wesley, 2000. ISBN 978-0-201-70225-5.
- Evans, E. *Domain-Driven Design: Tackling Complexity in the Heart of Software*. Addison-Wesley, 2003. ISBN 978-0-321-12521-7.
- Gotel, O., Finkelstein, A. An Analysis of the Requirements Traceability Problem. *Proceedings of the First International Conference on Requirements Engineering*, IEEE, 1994, 94–101. doi:[10.1109/ICRE.1994.292398](https://doi.org/10.1109/ICRE.1994.292398)
- Haynes, A. B., and others. A Surgical Safety Checklist to Reduce Morbidity and Mortality in a Global Population. *New England Journal of Medicine* 360(5), 2009, 491–499. doi:[10.1056/NEJMsa0810119](https://doi.org/10.1056/NEJMsa0810119)
- Jacklin, S. A. *Certification of Safety-Critical Software Under DO-178C and DO-278A*. NASA Ames Research Center, 2012. <https://ntrs.nasa.gov/citations/20120016835>
- Martin, R. C. *OO Design Quality Metrics: An Analysis of Dependencies*, 1994. <http://www.objectmentor.com/resources/articles/oodmetrc.pdf>
- Martin, R. C. Stability. Engineering Notebook column, *C++ Report*. <http://www.objectmentor.com/resources/articles/stability.pdf>
- Martin, R. C. *The Clean Architecture*, 2012. <https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html>
- Martin, R. C. *Clean Architecture: A Craftsman's Guide to Software Structure and Design*. Pearson, 2017. ISBN 978-0-13-449416-6.
- Meyer, B. Applying "Design by Contract". *IEEE Computer* 25(10), 1992, 40–51. doi:[10.1109/2.161279](https://doi.org/10.1109/2.161279)
- Meyer, B. *Object-Oriented Software Construction*, 2nd ed. Prentice Hall, 1997. ISBN 978-0-13-629155-8.
- Meyer, B. *Handbook of Requirements and Business Analysis*. Springer, 2022. doi:[10.1007/978-3-031-06739-6](https://doi.org/10.1007/978-3-031-06739-6)
- Nygard, M. *Documenting Architecture Decisions*, 2011. <https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions>
- Parnas, D. L. On the Criteria To Be Used in Decomposing Systems into Modules. *Communications of the ACM* 15(12), 1972, 1053–1058. doi:[10.1145/361598.361623](https://doi.org/10.1145/361598.361623)
- Parnas, D. L. Designing Software for Ease of Extension and Contraction. *IEEE Transactions on Software Engineering* SE-5(2), 1979, 128–138. doi:[10.1109/TSE.1979.234169](https://doi.org/10.1109/TSE.1979.234169)
- Parnas, D. L., Clements, P. C. A Rational Design Process: How and Why to Fake It. *IEEE Transactions on Software Engineering* SE-12(2), 1986, 251–257. doi:[10.1109/TSE.1986.6312940](https://doi.org/10.1109/TSE.1986.6312940)
- Savoia, A. *This Code is CRAP*. Google Testing Blog, 2011. <https://testing.googleblog.com/2011/02/this-code-is-crap.html>
- Tornhill, A. *Your Code as a Crime Scene*. Pragmatic Bookshelf, 2015. ISBN 978-1-68050-038-7.
- Tornhill, A. *Software Design X-Rays*. Pragmatic Bookshelf, 2018. ISBN 978-1-68050-272-5.
- Vernon, V. *Effective Aggregate Design*, Part II. 2011. <https://www.dddcommunity.org/wp-content/uploads/files/pdf_articles/Vernon_2011_2.pdf>
- Zave, P., Jackson, M. Four Dark Corners of Requirements Engineering. *ACM Transactions on Software Engineering and Methodology* 6(1), 1997, 1–30. doi:[10.1145/237432.237434](https://doi.org/10.1145/237432.237434)

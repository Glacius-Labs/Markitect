# Decisions

The decisions that hold for Markitect, grouped by topic, and the questions still open. The [concept](concept.md) describes the product they add up to, and the [vision](vision.md) says why.

- This page lists only what holds. A decision changes only through a new decision: it takes the next free number, and the entries it replaces leave the page in the same change. Git keeps every earlier wording.
- Only the product owner adds or changes a decision. An agent's proposal enters as an open question and stays one until the owner decides.
- Do not reopen a decision without new evidence. With such evidence, add an open question that cites it and leave the decision as it is.
- An entry keeps its ID, and the IDs it absorbed keep an anchor on it, unless the anchor names a retired design.
- If another page differs from a decision, the decision holds: correct the page.

## Purpose

### DEC-101 People architect their project in Markdown

- People design, define and model their project in Markdown: what its concepts are and mean, how they relate, its interfaces, structure, rules and way of working. Architect is a role that the owner and an agent fill together: the agent drafts, analyses and proposes, and the owner decides.
- The spec is typed prose in Packages, with links that work like imports. A deterministic tool checks and analyses it like code, and agents realize and verify it.
- Code has always been a precise description for a limited but diligent executor. An AI executor is intelligent, but it still has to be told what is meant and what holds, or it guesses. So documentation takes the place of code as what people work on, and Markitect makes its authors define what they mean and what holds and connect those statements. What matters in code stays as relevant there: architecture, consistency, cleanliness, an overview without reading the code, and knowing what should be as against what is.
- Contradictions in a description surface only when someone changes the place where they sit, as with dynamically typed programming languages. So Markitect borrows typing and a compiler as helpers; nothing in the spec is executed. The aim is to meet the preconditions that open the established knowledge of software architecture to the spec: Definitions with a type, declared dependencies, and a structure a tool can check and measure.
- It builds on proven practice: requirements traceability with suspect links (the registry, `status` and `mark`), architecture metrics (`vet`) and Go's conventions (the command line). What is new: the traceability is plain, typed text checked deterministically, agents do the realization and the evidence, and the record of what was verified works like Go's test cache. AI makes affordable what ordinary projects dropped for its manual cost, such as keeping a traceability matrix.
- The honest limit: "complete" means complete for the declared links.

<a id="dec-001-delegated-realization-is-central-to-the-product"></a>

### DEC-001 Delegated realization is the purpose

- People maintain the spec. Agents realize it in code, tests, configuration and documentation, and verify the result. The tool computes and checks, so that the delegated work can be trusted.
- Judge Markitect as a way of developing software, not as a linter or a context tool. Removing delegated realization as scope creep contradicts this decision.

<a id="cpt-001-trust-comes-from-structure"></a><a id="cpt-001-federalism"></a>

### DEC-002 Trust is measured against today's practice

- The baseline is common practice: an agent implements a task, another AI reviews the pull request, and the human approval is often given almost blind. Markitect aims to earn more trust than that, not compiler-grade guarantees.
- Trust comes from structure, not from infallible agents: four eyes, roles with bounded rights, need to know, a list computed from the spec rather than from memory, and a check of every Definition against the spec. The [vision](vision.md#where-trust-comes-from) lists the mechanisms.
- Claims read "more reliable than the baseline", never "correct". Use the smallest process that earns the trust: more report fields, mandatory probes or completeness declarations do not by themselves make work more correct.

<a id="dec-003-the-compiler-is-a-borrowed-concept"></a><a id="cpt-003-two-parts-of-the-promise"></a>

### DEC-003 The compiler is a borrowed idea, and two claims stay apart

- The spec's structure compiles deterministically: Kinds, links, Packages and file accounting. Meaning is judged by agents. Markitect claims no semantic guarantee for what agents build.
- Two claims stay apart. *Nothing in the spec's scope is overlooked* can be computed from declared links and file accounting; undeclared dependencies in code are a known gap. *Everything is realized correctly* can only be supported by evidence: a fresh reviewer, the project's tests and `check`.

<a id="dec-009-the-central-evaluation-question"></a><a id="dec-008-the-delegated-method-has-not-yet-been-tested"></a><a id="dec-010-evaluate-the-method-separately-from-its-execution-runtime"></a>

### DEC-009 How Markitect is judged

- The aim is one clean and simple version that works reliably on Sonnet, with no babysitting, not to prove that one version beats another.
- The long-term questions: when people develop against a spec, can it be realized reliably and can that be checked; does the repository stay consistent and clean; does agentic coding become more trustworthy; are rules followed instead of overlooked; can people move their attention to architecture?
- The method is judged apart from any runtime: a failure of Markitect's own tool is a product finding, a harness or environment failure a limit of the measurement.
- The method has not yet been tested over a long series of changes. The [concept](concept.md#13-what-is-shown-and-what-is-not) says what is shown and what is not.

<a id="dec-014-compatibility-does-not-drive-decisions"></a><a id="dec-016-a-clean-stable-testable-main-and-uniform-structure-first"></a><a id="dec-038-learn-through-complete-workflows"></a>

### DEC-032 Quality, simplicity and stability over effort and compatibility

- Clean architecture, simplicity and stability come first. Effort is no criterion, and neither is compatibility with old versions while Markitect is early. Released files stay immutable; that is a fact, not a promise of support.
- Choose the clean design even when it is more work. Add no aliases, adapters or legacy paths. Delete what is obsolete, source, tests and documents together.
- Main stays clean, stable and testable: tests check Markitect itself, not provider or environment quirks, and the repository has a uniform structure.
- Simpler is more robust, and simplicity is earned through insight. Markitect works with Claude Code's own features rather than building machinery around them.
- A substantial redesign names the new observation behind it and the result that would show it is better, and asks first whether a mechanism is needed at all or something simpler would do.

### DEC-098 One concept describes the product

- The [concept](concept.md) is the single description of the product and the target that Main builds: the spec, a tool that checks it and records what was verified, and a thin default harness in which the user's own Claude Code session starts the agents.
- Earlier concepts are kept on archive branches ([history](history.md)), not on Main.

## The spec language

<a id="dec-080-the-model-is-typed-documentation"></a>

### DEC-080 The spec is typed documentation in Markdown

- A Definition is one Markdown file with YAML front matter that names its Kind. Its path is its ID, its main heading its title, and its body the documentation. Exact values, such as wire names and error codes, are written in prose. A Kind may declare the fields and the parts of its Definitions ([DEC-110](#dec-110-a-kind-declares-fields-and-parts-and-std-describes-a-whole-project)).
- A Kind is itself a Definition, of Kind `Kind`. No Kind of a Library has a special meaning to Markitect; Core's own `Package` declares a Package ([DEC-108](#dec-108-a-folder-is-a-namespace-a-package-is-declared-by-its-anchor)). Projects define their own Kinds, and Libraries supply common ones: `std` ships in the binary; any other Library is committed under `.markitect/libraries/`. The manifest names each Library with its version, and the lock pins its digest; `tidy` writes it.
- The spec lives in `.markitect/spec/`. A folder is a namespace, and a Package is a folder with its anchor ([DEC-108](#dec-108-a-folder-is-a-namespace-a-package-is-declared-by-its-anchor)). Everything a Package defines reaches its whole subtree, and the spec's root folder is the Root Package.
- A link, `[[members/member]]`, names its target by path and is a declared dependency, like an import. Links point up the tree or sideways, never down into a child's Definitions; a cycle between Packages is a compile error, as in Go. A Definition with `internal: true` front matter may be linked only from its own subtree.
- There are no explicit layers and no mechanism by which one Kind refines another. The Package tree carries the Dependency Rule: links up are free, links down are forbidden, sideways edges are checked by the cycle ban and `vet.unstable_dependency`, and a project states its own limits with `internal: true` or an ordinary Rule.
- The [spec language guide](guides/spec-language.md) is the canonical description.

<a id="dec-079-the-model-is-as-explicit-as-good-documentation"></a><a id="dec-061-what-an-extracted-model-contains"></a><a id="dec-005-model-granularity-is-the-authors-responsibility"></a>

### DEC-079 The spec is as explicit as good documentation

- A statement belongs in the spec when good documentation of the project would state it that explicitly: domain concepts, features, interfaces such as REST endpoints and CLI commands, rules, decisions and the architecture. Loops, maps and data structures stay with the agents.
- The binding part is what callers rely on: inputs and their validation, data shapes, status and exit codes, stored data. Detail a caller can observe but not reasonably depend on, such as message wording or identifier formats, is free unless the project fixes it. This holds for the first spec of an existing project too ([DEC-105](#dec-105-a-first-spec-of-an-existing-project-states-what-is-as-what-should-be)).
- The project's owner decides the architecture, and the spec encodes it as its own structure: Packages, their links and their file ownership.
- Granularity is the author's choice. Markitect measures and warns; it imposes nothing ([DEC-102](#dec-102-what-vet-measures)).

<a id="dec-063-the-model-is-authoritative-defects-escalate"></a><a id="dec-033-binding-intent-lives-only-in-the-model"></a><a id="dec-034-removed-intent-removes-its-realization"></a><a id="dec-026-apply-comes-first-and-changes-only-artifacts"></a>

### DEC-063 The spec is the authority, and its defects go to the owner

- Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines and configuration alike. A project rule found outside the spec, in configuration or agent instructions, belongs in the spec.
- The agents that build never change the spec, and the agent that drafts the spec writes no code ([DEC-100](#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). The project's owner accepts every spec change.
- A gap with no effect at the project's boundary is the builder's to decide. Where the spec contradicts itself or leaves open behaviour at the boundary, nothing is built for that Definition: the reviewer records `spec-problem`, and only the owner settles it.
- Removed intent removes its realization: a file whose only basis was a removed Definition goes, or gets a new basis in the registry.

<a id="dec-006-every-file-is-covered-or-explicitly-ignored"></a><a id="dec-081-the-organization-knows-broad-at-the-top-and-deep-below"></a>

### DEC-006 Every file is accounted for

- Every file of the project is realized by the spec and listed in the registry, is Markitect's own, or is excluded: by the project's `.gitignore` files or by Markitect's ignore file, `.markitect/ignore`. One file may realize several Definitions.
- The registry is one Markdown page per Package: the path patterns it owns, most specific first as in `CODEOWNERS`, and each file with the Definitions it realizes. Root owns what no other Package claims, and a file shared by several Packages sits with their lowest common parent.
- `check` refuses a file neither listed nor ignored, a registry row for a Definition or file that no longer exists, and a Definition that nothing links to and no file realizes. Accounting proves that nothing is unexplained, not that the realization is right.

### DEC-104 A link is never decorative; identity and surface are written as code writes them

- The spec behaves like code, and code has no "see also": every link is a real relation that matters for meaning and, in some direction, for change. A link to a Definition says that the source relies on what the target states; a link to a Package says that the source's module depends on that module or, down the tree, says something about a part of the parent. A link that would only help a reader find something does not belong in the spec; navigation belongs in documentation rendered from it.
- The spec has no relation keywords, and `check` and `status` treat every link alike. What code writes with a type and with visibility, the spec writes the same way:
  - a record that others name by its identifier gets an identity Definition of its own in its Package, such as `members/member-id`; the record and its identity link each other, and each holder links the identity instead of the record, as domain-driven design's typed identity; a holder that relies on more than the identity links that as well;
  - a Package's surface is its visibility: what is not `internal: true` is what it publishes, and the spec does not list it again;
  - a module names the modules it uses by Package, `[[members]]`.
- `doc PACKAGE` shows a Package's public Definitions apart from its internal ones, and every Definition that `list` and `doc` show carries `internal` in their JSON.
- Link keywords such as `id` and `part` were weighed and not chosen; they are added only on new evidence: a real spec whose surface must differ from its visibility, or whose identity Definitions, measured over its history, cost more than the re-verification they save ([OQ-015](#open-questions) is the one point left open).
- [What makes a good spec](guides/good-spec.md#every-link-is-a-relation) states the patterns. On the archived Makerspace specs they put 113 items on the list instead of 133 over eight change steps, and 354 instead of 468 on a project five times as large, with no needed item lost ([measurements](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/spec-relations-20261108.md)).

### DEC-105 A first spec of an existing project states what is as what should be

- An existing project without a spec gets its first spec from the project as it is. What is, is every artifact, not only the code: code, tests, documentation, scripts, pipelines and configuration. The first spec reads what is as what should be and states it, as explicitly as [DEC-079](#dec-079-the-spec-is-as-explicit-as-good-documentation) says.
- It is finished when a build would change nothing. This is the loop of a build with the roles turned round: the reviewer verifies every Definition against the project, the architect changes the spec where it is wrong about the project, and the two repeat until nothing is left to verify but what waits for the owner.
- A Definition that does not hold is one of three things:
  - the spec misread the project, and the spec is corrected;
  - the project contradicts itself, such as its documentation against a script. The architect does not settle it: the Definition stays as drafted and on the list as `fail`, and the owner says which side holds before anything is built;
  - the project does something that nothing in it states and that looks like an accident. The spec leaves it unstated, since stating it would make the spec demand it, and the architect asks the owner, in the file of [DEC-100](#dec-100-skills-are-named-after-the-activity-agents-after-the-role) when the owner is not there to answer.
- The spec is not a transcription of the project. The question at each artifact is what it holds that should also be in the spec, not what it holds that the spec does not say. Most of what a file holds is then stated as an obligation without its values, or left to the agents.
- With the owner's acceptance the spec becomes the authority ([DEC-063](#dec-063-the-spec-is-the-authority-and-its-defects-go-to-the-owner)). What the owner confirmed as a defect stays `fail`, and `/markitect-build` repairs it; the spec is not changed to empty the list.

### DEC-108 A folder is a namespace; a Package is declared by its anchor

- A folder of the spec is a namespace, as in C#: it gives its Definitions their path and means nothing else. A project orders its Definitions as it reads best, and Markitect prescribes no layout.
- A Package is a module, as a project is in C#: the unit of the dependency check, of visibility and of file ownership. A folder is a Package when it holds its anchor, a Definition of kind `Package` that carries the folder's name and says what the Package is responsible for and what it keeps to itself. Core provides that Kind beside `Kind`.
- What lies in a folder that is no Package belongs to the nearest Package above, shares its scope, visibility and dependencies, and keeps the folder in its ID. A folder that is no Package has no registry page, and is a link target only as a thing with parts ([DEC-109](#dec-109-a-thing-with-parts-is-read-as-a-whole)).
- A folder and a module are two things. Were every folder a Package, ordering an area's Definitions in folders would cut the area into modules: the links inside it would be Package cycles, and the area would hold nothing. Were a file of the folder's name to make the Package, a thing with parts could not hold the Definition that names it without becoming a module, and the Definitions beside it could not link its parts.
- A link to a Package stays what [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) says: it names the module and does not rely on the anchor's text. A Definition that relies on that text links the anchor.
- A project in which every folder was a Package gives each of them its anchor. Until then its folders are namespaces of Root: `check` says so at each link to one and at each registry page of one, and where a project has neither, `markitect list` shows the Packages it has left.

### DEC-109 A thing with parts is read as a whole

- A Definition states one thing, so that it can be verified alone and changes alone. A thing that has parts is therefore a folder: it holds the Definition of its own name, the whole, and its parts in it or below it, each a Definition of its own.
- A reader needs the thing on one page all the same. `doc` shows a whole with its parts, the parts of one folder together, folders and parts in the order the whole links them and then by name and by ID, and shows each link by its label or by the title of what it names. A whole whose parts have an order states it by linking them in that order. A part that several things share lies with the thing it is about and is linked by the others; `doc` shows it with each of them, in its place, and says what it is a part of.
- A link to the folder means the whole, so that a link names a thing once: `[[members/operations/lend]]`.
- `vet` counts things: parts do not count for the size of a Package, and a part may name its whole without linking it.
- A Package ends a thing: an anchor is no whole, and what lies in a Package is no part of a thing above it. A Package is a module ([DEC-108](#dec-108-a-folder-is-a-namespace-a-package-is-declared-by-its-anchor)), and a thing with parts is one thing written in several files.
- The whole keeps the ID its file gives it. Giving it the folder's path as its ID was weighed and not chosen: a Package and its anchor would then share one ID, and a link to a Package is not a link to its anchor ([DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them)).

### DEC-110 A Kind declares fields and parts, and std describes a whole project

- A Definition's text has labelled fields: a paragraph that starts with a bold label ending in a colon. A Kind may declare which fields its Definitions have, which are required, which Kinds a link in a field may name, and in which folders a thing of the Kind has its parts. The compiler checks every Definition against it. What a Kind does not declare stays text and is not checked.
- Agents write the spec, and a form that is only described in prose can be checked by a reader alone. With one thing per Definition ([DEC-109](#dec-109-a-thing-with-parts-is-read-as-a-whole)) a field is one line that mostly holds links, so the field a link stands in is the relation it states. Every modelling language that was compared declares fields and relations on the type and checks them: the Foundry Ontology, LinkML and SHACL, Sphinx-Needs with the metamodel of Eclipse S-CORE, TRLC, Smithy and SysML v2.
- The statement of a Definition stays a sentence. None of those languages formalizes the sentence of a requirement; formal is what surrounds it.
- Fields stay in the Markdown text and do not move into the front matter: a field often holds a sentence with links in it, and one form for all fields is simpler than two.
- A Kind may declare the Kinds that any link of its Definitions may name. This is the dependency rule of software architecture, applied to statements: what is given, a term, a constraint or an assumption, links only what is given, and a goal links only that and other goals, so that nothing stable depends on what changes often.
- A Kind may declare that its Definitions bind without a link, as a Rule binds its Package and those below it by its place. `doc` then lists what binds a Definition, so that a builder and a reviewer are shown the Rules that hold for an item and do not have to know to look for them.
- A Kind may also declare that no file realizes its Definitions, `realized: false`: an assumption or an exclusion is verified, and nothing is built for it, so no registry row has to name it.
- `std` is the one standard Library. An API and a command line are an Interface with its Routes, so the Libraries `rest` and `cli`, each one Kind for an endpoint or a command, are gone.
- The declaration is small on purpose: required, the Kinds a link may name, the folders of parts, what any link may name, whether a file realizes it, and whether it binds by its place. Types of values, enumerations and inheritance between Kinds were weighed and left out until a run shows the need.
- The standard Library `std` 2.0.0 carries the Kinds of a whole project's description, sorted by eight questions: purpose, context, domain, interfaces, quality, architecture, scenarios and administration. Their forms come from established practice: goals and stakeholders from requirements engineering, quality by the characteristics of ISO/IEC 25010 with a measure for each requirement, decisions as architecture decision records, strategy, rules across the code, runtime and deployment as arc42 orders them, and operations with conditions and outcomes as contracts. Relations take the names SysML v2 and S-CORE use, Satisfies and Derived from, and the one who acts is the Actor, as in a use case.
- An ontology language as the form of the spec, RDF with OWL and SHACL, was weighed, and so were SysML v2 and TypeScript as the language. None was chosen: what Markitect adds is the loop of spec, build, verification and record, and Markdown is what people and agents already write. A spec of this form translates to those languages by rule, should a project ever need to join its spec to knowledge outside it.

### DEC-111 A thing with parts is built and verified as one

- An item of `status` is a Definition that is no part of another, with its parts: the record holds one line for it, its inputs include each part, and `mark` takes one verdict on it. A part that has parts of its own is an item of its own, and the thing above it does not cover it.
- A Definition states one thing ([DEC-109](#dec-109-a-thing-with-parts-is-read-as-a-whole)), so a spec has several times as many Definitions as things. A builder builds an Operation, not one of its conditions, and a reviewer who records a verdict on each condition alone reads the same code once per part. The parts are the cases of the thing: the reviewer's count of cases covers them.
- A verdict on a part is refused and names the item. A change of a part, of what a part links, or of a file a registry row names for a part puts the thing back on the list; what links the thing's folder relies on the whole alone, as before.
- A registry row may name the thing or any of its parts, and a link from outside the thing to either counts for all of it: a part needs no row of its own, and `definition.unused` is reported for a thing, not for its parts.
- `vet` follows: a part's name is no unlinked mention, what one part of a thing links is linked for all of it, and of several Definitions with one name a mention names the nearest. Root is not measured by its size in a spec with Packages, and no Package is asked for path patterns before something is built.
- `fail` on a thing says in its reason which part fails. One verdict per part was the alternative: it keeps a finer record and costs a mark, a reading and a registry row per sentence of the spec.

### DEC-112 A long list is built in slices

- A build of a long list goes slice by slice. `markitect status --slices` cuts what a builder can settle into slices, and the session takes them one after the other: a fresh builder with its reviewer in a worktree of its own, and a merge into the owner's branch when the slice is settled, so that the next builder starts from it. A list of at most 40 items is one slice and is built by one builder, as before.
- The cut follows the Packages in the order of their dependencies: a Package comes after the one above it and after every Package it depends on. Neighbours are joined while a slice stays within 40 items, and a Package is never cut. The last slice is whatever is left: what is verified last, the errors in the realization and what an earlier slice left.
- A Kind may declare `last: true`: a Definition of the Kind is judged on the whole project, as a `std/Quality` and a `std/Scenario` are, and is built and verified in the last slice. What no file realizes (`realized: false`) is verified last without saying so, since it can only be judged on what the others built. `std` is 2.1.0 with it.
- One builder and its reviewer have been shown on lists of up to about 64 items, and the first build of a described project lists more than a hundred, half of them in Root. A slice keeps the task of each agent within what has been shown, and what others rely on exists before they are built. A slice larger than the size, one Package alone, is not cut: the plan names its size.
- `markitect status --slice A,B` lists the items of those Packages that are not verified last, and `--size N` sets another size for one run. The command of a slice names its own Packages and every Package before them: what a slice changes in a file an earlier item rests on puts that item back on the list, and it is then the work of the slice that caused it. After each slice everything up to it is verified, and nothing gathers for the end.
- Each settled slice is merged, so a build that stops has lost nothing and has nothing to find again: `/markitect-build` started anew cuts what `status` then lists. The tool keeps nothing between the commands. A slice that leaves an item is merged all the same when it shortened its list, and the next slice takes the item up; a slice that settled nothing stops the build, and the session asks the owner. After the last slice the work comes back only when nothing is left but what waits for the owner, as after any build.
- An item of a slice is judged on the project as it stands. A Rule of Root is `ok` in the first slice when what exists keeps to it and what it asks for itself is built: what a later slice builds answers for the Rules above it when it is verified, as every later change does, and the record brings the Rule back when a file it rests on changes. So a slice ends with an empty list, and nothing is put off for the end but what a Kind declares so. Such an `ok` says what held when the slice was built: whether what is built later keeps to a Rule above it rests on the review of each later item, which reads what binds it, and a review of everything, `/markitect-review --all`, judges the Rule on the whole project. A file that no registry page lists is the work of the slice that added it, and its list names it.
- The parallel workflow takes Root's strand last, since that strand makes the changes the others ask for in shared files. A build in slices takes Root's Rules, Decisions and components first, since nothing exists yet that the others could build on. The two orders serve different lists and are computed apart.
- The parallel strategy was the alternative for a first build. It builds at once Packages that do not exist for each other yet, and a first build has nothing to build on.
- It rests on what no test shows. One builder with its reviewer has to settle a slice of a real project's size, such as the 36 items Root lists first for a booking system; where one does not finish a slice of about 40 items, a Package is cut by its top-level folders. And a slice that shortened its list has to leave the project fit to build on; where later slices fail on what an earlier one left, a slice is merged only when it is settled.

### DEC-113 What binds is an input of what it binds

- A Definition of a Kind that declares `binds: true` binds its Package and every Package below it ([DEC-110](#dec-110-a-kind-declares-fields-and-parts-and-std-describes-a-whole-project)). This binding is an input of what it binds, beside the inputs [DEC-099](#dec-099-a-record-of-what-was-verified) lists. What binds a Package is each item that lies in the Package itself and whose Kind binds, an item being a Definition that is no part of another, with its parts ([DEC-111](#dec-111-a-thing-with-parts-is-built-and-verified-as-one)).
- An item a file realizes carries one input for its Package and for each Package above it that something binds: `bound:Root`, `bound:domain/staff`. Its value is the digest of the Package's directory, which maps each thing that binds the Package to the hash of its text and the texts of its parts. So a changed Rule, Decision or Constraint puts everything it binds back on the list, the binding Definitions themselves included.
- `status` names on each item the binding Definition that changed, in place of the Package: `changed: bound:architecture/rules/persistence`, `(new)` for one that binds the item since its verdict and `(gone)` for one that no longer does. The item itself is left out, since its own inputs say how it changed.
- Why: `status` promises to list what a change reaches. Since DEC-110 a Rule binds by its place and is no longer linked, so the links that carried a change of a Rule to what keeps to it are gone. A changed Rule was then one item: every `ok` below it stayed as given under the old Rule, and a place the builder forgot was listed nowhere.
- Only what a file realizes is bound. A Goal, an Exclusion, a Term or the Project is bound by nothing, and neither is a Kind: nothing is built for them, so no statement about what is built holds for them. `doc` shows no `Bound by` for them.
- A Rule of Root binds everything that is built, so its change rightly lists everything. A Rule that holds for one area lives in that area's Package and reaches only there.
- The record keeps, beside its lines, the directory of each Package and digest that a line names. `mark` writes it with the verdict, and it is dropped when no line names it any more. The name of the changed Definition must survive the verification of that Definition: once the Rule is `ok` again, what it binds is still listed and still says which Rule to check it against. One input per binding Definition on every line was the alternative; it repeats the same content on every line of the record.
- Where the record no longer holds the directory a line names, or the line has no `bound:` input, as in a record of an earlier release, `status` names the Package, as in `bound:Root (new)`. Such a record lists each bound item once more.
- The item of a binding Definition verifies what the Definition asks for itself and that nothing that exists contradicts it; what it binds is verified against it item by item, as `std/Rule`, `std/Decision` and `std/Constraint` say. The binding Definition's own item as the one verdict on the whole project was the alternative. It puts the trust into one verdict whose evidence cannot enumerate the project.
- `vet.hub` counts links: what a Definition binds by its place is no finding.
- It rests on reviewers verifying an item against what binds it, which `doc` lists. In one small real run the reviewers read the whole project at once and did not look at what binds each item, which is sound where a project can be read at once. Where a review of everything turns an `ok` on a larger project, the item of a binding Definition gets a closing review on the whole project.

## The tool

<a id="dec-096-markitect-computes-and-checks-a-driver-acts"></a>

### DEC-096 The tool computes and checks; a driver acts

- The `markitect` program checks and analyses the spec, says what is to verify, and records verdicts. It starts no agent, keeps no run open, imposes no order, counts no rounds and keeps no state between commands outside the project.
- Who starts which agent, when and how often is the driver's: the skills, the person's own session, a workflow or the person ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)). A driver may change the prompts, agents, skills, settings, limits and every Git step; it may not change the tool, the record's inputs or what the hook refuses.
- The commands are flat and named after Go's tools and other widely known ones: `init`, `check`, `vet`, `list`, `doc`, `status`, `mark`, `tidy`, `config`, `update`, `install`, `version` and `licenses`, and `hook` for the hooks. Diagnostics read `file:line: code: message`; `--json` prints one document with a `format_version` and a `kind`; exit 0 means all is well, 1 that the command found something, 2 that it could not run, and a note never changes the exit code.

### DEC-097 Markitect uses no Git

- The tool makes no repository, commits nothing, keeps no branches, lands nothing and reads no history. Agents and people do every Git step.
- It reads the project by walking its folder, skipping what the project's `.gitignore` files exclude, read as plain text.
- So it cannot guarantee that what is committed is what was verified. `status` on any checkout shows what differs from the record, and a CI job can refuse it.

<a id="dec-027-strict-links-with-an-optional-optimizer"></a>

### DEC-102 What `vet` measures

- `check` is the compiler; `vet` is the static analyzer. It warns about coupling and instability per Package, dependencies on less stable Packages, the size of Definitions and Packages, Packages that hold nothing, and hubs: a Definition with fan-in + 1 > max(10, N / 5), Root included, Kinds left out. It exits 1 only with `--strict`.
- Notes never change the exit code; plain `vet` counts them on one line and `--strict` lists them. They are unlinked mentions, which the auditor triages into missing links and ordinary words, with the mentions a link to which would go against the Package structure or add a dependency counted apart; long Definitions (more than 20 statements); placement; and hotspots, revisions × fan-in > max(20, N / 10), from how often a Definition's own text changed between verdicts.
- `list --deps` shows Martin's abstractness and distance, as information only. The zone notes wait for real records; the risk measure is a note ([DEC-106](#dec-106-the-reviewer-counts-and-vet-notes-thin-evidence)). Every threshold is provisional, and `vet.unlinked_mention` becomes a warning only once a spec's own measurements show its noise low.
- Links stay strict: the tool reports a missing link and never adds one. A value rule of the project's own is stated once, as a small Definition in the only Package that uses it or at the lowest common parent of those that share it; there is no central page of values.
- [What makes a good spec](guides/good-spec.md) says what each measure is for.

### DEC-106 The reviewer counts, and `vet` notes thin evidence

- An `ok` says that a Definition is implemented, as the evidence shows. How much of the Definition the evidence shows belongs to that statement: with an `ok` and a `skip` the reviewer counts the cases, conditions and exceptions the Definition demands and how many of them its evidence demonstrates, and `mark` refuses either verdict on a Definition of the spec without both counts. A count is a fact the reviewer reads off the Definition and its evidence, not a rating; it tells the owner how much was shown, and claims nothing about whether the work is right.
- `vet` notes a Definition whose `ok` or `skip`, at its text as it is, has a risk above 30, as `vet.risk`. The risk is CRAP's, for a Definition: with *c* one more than its cases and *e* the share of them the evidence demonstrates, 0 for a `skip`, it is *c*² × (1 − *e*)³ + *c*. So a Definition is noted when its `ok` rests on evidence for few of many cases, when it was skipped with more than a few, or when it has about 30 cases, however well they are demonstrated.
- It is a note and never changes an exit code: it says where a verdict is thin, and the owner or the next review decides what to do. The line of 30 is provisional, like every threshold of `vet` ([DEC-102](#dec-102-what-vet-measures)).
- Why now: real reviewers gave the counts with every verdict in one run, with most in a second and with none in two more, so the record could not be relied on for them; and on the one record that had them all, the line marked 2 of 114 Definitions, both for a reason an owner would want to know ([record](https://github.com/Glacius-Labs/Markitect/pull/1054)).

## Verification

<a id="dec-099-markitect-is-a-tool-with-a-verification-cache"></a><a id="cpt-004-the-specification-stays-effective"></a><a id="cpt-002-tests-are-the-projects-and-they-are-realizations-too"></a><a id="cpt-002-tests-are-realizations-too"></a><a id="dec-087-required-checks-gate-merged-states-not-each-submission"></a>

### DEC-099 A record of what was verified

- For each Definition, `.markitect/review.sum` records the inputs at which a reviewer last verified it, by their hashes: its text, its Kind and the Definitions it links to (a Library Definition by its Library's digest), its registry rows with their files, and the evidence. It is committed with the project, like `go.sum`, and merges with `merge=union`.
- `markitect status` lists every Definition with a changed input or no record, those recorded `fail` or `spec-problem`, those gone from the spec, and the errors in the realization. It gives no list while the spec has errors. An empty `status` means that the project compiles and every Definition is verified at its current inputs.
- `markitect mark DEFINITION@FINGERPRINT VERDICT` records `ok`, `fail`, `skip` or `spec-problem` at the state the reviewer read, with its evidence (`path` or `path:line`) or why. It refuses when an input changed since `status` printed the fingerprint.
- A verdict holds until one of its inputs changes, so every obligation keeps a verdict, and a change puts back on the list only what it reaches, one step along the links. It is a record, not a proof, and the tool keeps no limit on rounds.
- Tests are the project's, and they are realizations too: agents write them, so a green test is evidence, not proof. A project that wants tests states them in its spec as an ordinary Rule, with the command that runs them; the reviewer runs them as evidence. The tool knows nothing of them, and there are no required checks: if a Definition is verified that a test would have caught, they come back.

## Skills and roles

### DEC-100 Skills are named after the activity, agents after the role

- The skills are `markitect-architect`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit`, `/markitect-schedule` and `/markitect-config`. Only the owner starts build, cleanup, bughunt and schedule; Claude may start architect, review, audit and config. In the architect skill the owner's session writes the spec with the owner and hands a large draft to the architect agent. A skill only names the agent and its task; the agents hold their tasks and verdict words in their own files.
- Four roles in a 2×2 of rights, held by a hook for subagents; the owner's own session stays free but does not record. The **builder** writes code and the registry pages, not the spec, and may start only the reviewer. The **reviewer** changes nothing and is the only one that records with `mark`. The **architect** writes the spec, the registry pages and the ignore file, not code, and hands its questions back. The **auditor** reads only and reports. No subagent writes the record or what steers the agents.
- What any agent notices goes into the record as `fail` or `spec-problem`, and nowhere else. What the architect has to ask the owner, and the owner cannot answer at once, waits in `.markitect/questions.md`, which goes into the same commit as the spec: one numbered point per question, with what was found, where, what the spec says or leaves out, and the proposal. A point about a Definition on the list names it, and the record keeps its verdict. No agent writes the file, and the owner's session keeps it; a point leaves it when the owner has answered, the file goes with its last point, and `status` says while it is there.
- Loops have no limit on rounds, but stop when three rounds in a row leave the list of `status` no shorter than it has been. A long list may run in parallel, one builder, reviewer or auditor per Package with a Package-scoped task ([DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)).
- The builder works in a worktree of its own, so the owner can go on architecting; its work comes back by a merge. Routines work on a branch of their own and land only when they leave nothing worse, the tests pass and the change is the agents' work alone.

<a id="dec-051-agents-get-tasks-the-orchestrator-owns-the-process"></a>

### DEC-051 Agents get tasks, not process

- An agent gets a clear task and an answer format. Order, limits, acceptance and routing to the owner are not explained to agents in prompt text.
- A failure in how agents work with Markitect is fixed in the tool contract, not with more prompt text. Since the tool steers no process, the agents' own files hold their task and answer format, and skills hold no steps an agent must remember.
- The one exception is the build loop. In a sequential build the builder is the driver: it starts the reviewer, reads `status` again and decides when to stop, and nothing else holds that loop. So the builder's file says when to start the reviewer and when to stop ([DEC-100](#dec-100-skills-are-named-after-the-activity-agents-after-the-role)). In a parallel run the workflow holds the loop, and the builder's file tells a strand to build once.

## Driver and configuration

<a id="dec-094-main-carries-the-native-version-and-the-app-goes"></a><a id="dec-092-a-command-uses-the-users-own-sign-in"></a><a id="dec-093-a-command-passes-the-users-own-token"></a>

### DEC-094 The user's own session runs the agents, natively

- Agents run on the user's own machine, started by the user's own Claude Code session, under its stored account login, permission mode and settings, in the unmodified program. No token, no API key and no background session.
- The tool runs natively on Windows, Linux and macOS, and Windows is a main path. Agents run with the user's rights, so work that reads untrusted input needs an isolated machine with limited rights around it.
- Markitect starts no session and touches no credentials, so each user runs Claude Code on their own login. Claude Code is the only provider for now; another comes once Claude's version is reliable, and the tool does not change for it.
- There is no app. If one comes, it works over the same commands.

### DEC-103 Claude chooses the parallelism per run, through a shipped workflow

- Claude decides per run whether work runs in sequence or in parallel. A skill runs by a **strategy**, set per skill: `auto`, the default, is meant to choose parallel when `status` lists more than about 30 items in at least two Packages (cleanup, bughunt and audit count the spec's Definitions), but chooses sequential until the parallel strategy has proven itself on a long list; so far it has passed two small real runs. `sequential` runs one builder, reviewer or auditor at a time; `parallel` runs the shipped workflow; a custom strategy is a saved Claude Code workflow that keeps the same contract: it reads `markitect status --json`, keeps to the four roles and stops by the same rule.
- The parallel workflow, `.claude/workflows/markitect-parallel.js`, plans strands per Package in waves along the spec's dependencies, what is depended on first. Claude Code gives a workflow's agents no tool to start an agent, and lets an agent in a worktree run Git only on its own, so each strand is steps of the script: a builder in a worktree of its own, which commits its work there; a merge step on the run's branch, which merges the strands' branches by name; and a reviewer for the strand's Package on the run's branch, whose verdicts a record step commits. Shared files, which Root owns, go to Root's strand, last and alone; at most 8 strands run at once. It stops when three rounds in a row leave the list no shorter than it has been, and reports what remains. Review, bughunt, cleanup and audit run per Package the same way. Parallel work needs a Git repository, and the session that starts it waits for it.
- **Configuration** is one file format, `harness.yaml`, at two levels: the user's `~/.markitect/harness.yaml`, not committed, and the project's `.markitect/harness.yaml`, committed. Key by key the project beats the user, and the user beats the built-in values: `sonnet` at effort `high` for the builder and the reviewer, `opus` at effort `medium` for the architect and the auditor. The aim on Sonnet ([DEC-009](#dec-009-how-markitect-is-judged)) concerns the roles that realize the spec, the builder and the reviewer. It holds the model and effort per role, the strategy by default and per skill, and the custom workflows. `markitect config [--user] KEY VALUE` sets a value and rewrites the agents' files; `config --list` shows each value with its origin; `/markitect-config` takes a request in plain words. `init` and `update` write the agents' front matter from it.
- The hook refuses a subagent's write of `harness.yaml` or of a shipped workflow, and a subagent's start of a workflow. Routines take no strategy, since nobody is there to answer when Claude Code asks before a workflow starts. A user's own skills in `.claude/skills/` are left alone, and the hook holds the roles there too.

<a id="dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps"></a><a id="dec-011-detailed-planning-of-future-ideas-waits"></a>

### DEC-084 Continuous work runs outside the tool, and autonomy grows in steps

- A loop runs as a scheduled job or routine that calls `markitect`; there is no daemon in the product. A budget per task or per day belongs to the job and the driver, never to the tool; when it is used up, the loop stops and says so.
- A trigger, such as an issue, an incident or a dependency update, passes an intake that rewrites it into a task; raw trigger text never reaches an agent. Where a Definition states the expected behaviour, the task becomes a `fail` on it; where the spec is silent, it becomes a spec proposal for the project's owner.
- Autonomy grows in steps, each the project owner's decision. Step 1: repairs run on their own, and spec proposals wait in an inbox. Step 2: the owner grants a mandate, the kinds of spec change agents may accept themselves; a fresh reviewer accepts proposals within it, and the owner gets a daily briefing instead of reviewing each change. Agents maintaining the spec within mandates is the final form ([roadmap](roadmap.md#ideas)).
- A wrong repair is reverted; a wrong spec change is reverted together with its realization. A circuit breaker drops the loop back a step when reverts or incidents rise, until the owner widens it again.
- Detailed planning of such ideas waits until a usable, stable version exists.

## Releases

<a id="dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself"></a>

### DEC-095 Releases, a plugin and a binary that updates itself

- A tag `vX.Y.Z` on a commit of Main whose source and plugin version are `X.Y.Z` publishes binaries for Linux, macOS and Windows on amd64 and arm64, a checksums file and a provenance file as a GitHub release. Between releases the source says `-dev`. The retired versions 0.1.0 to 0.14.1 stay unused.
- `plugin/` is the Claude Code plugin `markitect@glacius-labs`, listed in the marketplace repository `Glacius-Labs/Claude-Plugins`, which pins it to a release. Its launcher fetches the binary of its own version, so a plugin update is a binary update. The plugin works only in Claude Code, not on claude.ai or in Cowork.
- `markitect update` installs the newest release over the binary, then refreshes the personal skill and a project's harness and lock. It commits nothing, never touches the spec, and is the only use of the network.
- The download is checked against checksums of the same release, which guards against a damaged file, not against a release by someone who controls the repository.

## How Markitect itself is developed

<a id="dec-076-markitect-is-developed-as-a-regular-software-project"></a><a id="dec-020-markitect-uses-any-library-that-helps"></a><a id="dec-070-parallel-writers-where-useful"></a><a id="dec-077-linux-ci-run-on-request"></a><a id="dec-077-development-pace-and-linux-first-execution"></a>

### DEC-076 Markitect is developed as an ordinary software project

- Markitect's own repository keeps no spec of itself: agentic coding, tests and ordinary documentation, with one page per topic. Target projects keep their spec under `.markitect/`; product code and tests that handle it use fixtures.
- Markitect uses any library that helps; the architecture gate restricts only imports between its own packages.
- Sessions work in parallel where their files do not overlap; shared files have one writer at a time.
- CI runs on Linux only, with the race detector, started on request. Markitect's own tests run on Linux; for users, Windows is a main path ([DEC-094](#dec-094-the-users-own-session-runs-the-agents-natively)).
- [CONTRIBUTING](../CONTRIBUTING.md) is the development process.

### DEC-069 Documentation is written in English

- All documentation on Main is in English.

<a id="dec-071-history-leaves-main-through-the-archive"></a><a id="dec-071-repository-structure-and-archive"></a>

### DEC-071 Main holds only the current state

- Main's documentation describes what holds today, as one concept, and quotes no one. Superseded designs, plans, decisions and dated records leave Main in the change that supersedes them; Git keeps them, and each earlier concept has an archive branch ([history](history.md)).
- A page that relies on a dated result says it in one sentence and links its record: the PR that reported it, or a file at a full commit SHA.

## Open questions

| ID | Question | Related |
|---|---|---|
| OQ-001 | How can dependency observation work across languages and file types, and how is its own coverage reported? | [Roadmap](roadmap.md#later) |
| OQ-002 | Which thresholds show that a spec is too fine or too coarse? `vet`'s are provisional, and the zones of pain and uselessness wait for real records. | [DEC-102](#dec-102-what-vet-measures) |
| OQ-003 | Which model tiers suffice for which roles, and does the total cost fall? | [DEC-009](#dec-009-how-markitect-is-judged) |
| OQ-004 | How much does a reviewer on a different model add beyond fresh context? | [DEC-103](#dec-103-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow) |
| OQ-005 | Should Markitect be judged over a series of changes on a long-lived project, and what ground truth makes "nothing forgotten" measurable there, for example predeclared affected sets, seeded cross-cutting rules and hidden cases? | [DEC-009](#dec-009-how-markitect-is-judged) |
| OQ-013 | When the machine lacks a program or a service the work needs, does the record need a verdict of its own beside `skip` and `fail`? Today the builder stops and says so, and its items stay on the list. | [DEC-099](#dec-099-a-record-of-what-was-verified) |
| OQ-014 | What does "reliable" mean? The proposal is two clean passes in a row over the 5× project's eight change steps on Sonnet: every step ends with nothing left to verify without the owner, every hidden case passes, and no refusal of the hook was overridden. | [DEC-009](#dec-009-how-markitect-is-judged), [roadmap](roadmap.md#current-steps) |
| OQ-015 | A downward Package link that only lists a parent's children names what the tree already says, so under [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) it is a pointer. Should `check` refuse it, or a guide advise against it? No guide does yet, and the architect skill's example names its Packages by link. | [DEC-080](#dec-080-the-spec-is-typed-documentation-in-markdown), [DEC-104](#dec-104-a-link-is-never-decorative-identity-and-surface-are-written-as-code-writes-them) |

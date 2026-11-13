# Reference: the spec form

A compact summary of the spec language, the standard Library and the tools.

## Layout

```text
.markitect/
  manifest.yaml        Libraries and versions (std: 2.1.0)
  lock.yaml            Library digests, written by markitect tidy
  spec/                the spec; this folder is the Root Package
    project.md         the project Definition (std/Project), which opens the spec
    purpose/ context/ domain/ interfaces/ quality/ architecture/ scenarios/ administration/
                       the top-level folders the project has something to say in
    domain/members/    a Package, because it holds its anchor
      members.md       the anchor of domain/members (kind Package)
      concepts/member/ a thing with parts: member.md, and its parts in attributes/ and invariants/
      rules/<name>.md  a folder is a namespace: still domain/members
  registry.md          Root's registry page
  registry/<package path>.md
  ignore               files that belong to no Package, if any
  questions.md         the questions that wait for the owner, if any; the owner's session keeps it
  libraries/<name>/    Libraries that do not ship with markitect
```

Everything under `.markitect/` is accounted for by the compiler and needs no registry entry, and so is what `init` installs: `AGENTS.md`, `CLAUDE.md`, the `markitect-*` skills and agents, `.claude/settings.json` and `.gitignore`, and so are the worktrees Claude Code keeps under `.claude/worktrees/`. The compiler sees the project folder without what its `.gitignore` files exclude, read as text; it uses no Git.

## Definitions

One Markdown file with YAML front matter.

- Front matter: `kind` (required) and `internal: true` (optional); a Kind may also declare `fields`, `parts`, `links`, `realized`, `binds` and `last`. Another key is `definition.front_matter`.
- File name without `.md` is the name, the path below `spec/` is the ID (`domain/members/concepts/member/attributes/email`). Names: only `a` to `z`, `A` to `Z`, digits, `-` and `_`, starting with a letter or digit, so no umlaut, accent or space (`spec.name`).
- Body: one level-1 heading, the title, first; no second one (`definition.title`); then the Definition's own sentences, then its fields. Exact values in backticks.
- **A field** starts with a bold label ending in a colon at the start of a line, such as `**Satisfies:**`, and reaches to the next field; a label inside a list, a quotation or a sentence starts none. Which fields a Definition has, which it must have and what a link in each may name, its Kind declares.
- The spec folder holds only `.md` files and folders (`spec.file`). Two entries differing only in case, or `member.md` beside `member/`, collide (`spec.collision`).
- A folder is a namespace: it gives its Definitions their path. It is a Package when it holds its anchor, a Definition of kind `Package` that carries the folder's name: `domain/members/members.md` makes `domain/members` one (`spec.package` for another name, or at the top of the spec folder). What lies in any other folder belongs to the nearest Package above, and to Root if there is none.
- **A thing with parts** is a folder that holds the Definition of its own name, the whole, and its parts in the folders its Kind declares: `operations/borrow-book/borrow-book.md` with `conditions/` and `outcomes/`. A link to the thing's folder means the whole, and `doc` shows the whole with its parts. Each part is a Definition and is linked alone, but the thing is built and verified as one: `status` lists it as one item, a registry row names it by its folder, and it is unused only when no Definition outside it links it or a part of it and no file realizes either (`definition.unused`). A part with parts of its own is a thing of its own, with its own item. A Package ends a thing: an anchor is no whole.
- An **item** is what `status` lists: a Definition that is no part of another, with its parts.
- A top-level name may not be a Library name, `Kind` or `Package` (`spec.reserved`).

## The top-level folders

Each top-level folder answers one question. A folder appears when the project has something to say there, so a small project has few of them. The folders only sort: which of them are Packages follows from their anchors.

| Folder | Question | What lies there |
|---|---|---|
| `purpose/` | Why does the system exist? | `goals/`, `stakeholders/` with their concerns and wishes, `exclusions/`. The project Definition `project.md` stands beside the folder, in Root. |
| `context/` | What is given from outside? | `terms/`, `constraints/`, `assumptions/`, `external-systems/` |
| `domain/` | What does the system do? | One Package per area, with `concepts/`, `operations/`, `reactions/` and `rules/`; `domain/rules/` for the Rules that hold across areas |
| `interfaces/` | How is it reached? | One Package per site, API, command line or channel of messages, with its `pages/`, its Interfaces and their `routes/`, or its `messages/`, and the `shapes/`, `refusals/` and `limits/` that hold for all of them |
| `quality/` | How well must it work? | One Quality per characteristic of ISO/IEC 25010, named for it, with its requirements as parts; `quality/quality` names the characteristics that decide the design |
| `architecture/` | How is it built? | `strategy/`, `decisions/`, `rules/` for what holds across the code, one Rule with parts per topic, and `runtime/` with `components/`, `connectors/` and `deployments/`. A Decision that binds one Package lies in that Package's `decisions/`. |
| `scenarios/` | Does it fit together? | Paths through several areas |
| `administration/` | How is it kept running? | `tasks/` of the person who runs it |

An area of the domain is a Package, and so is each site, API, command line or channel of messages, such as `interfaces/email/` with its `messages/`; only a project with one area and one interface keeps everything in Root. Root may not link down into a child Package, so `architecture/runtime`, `scenarios` and `administration` are Packages too when they link the Concepts or Operations of the areas. `purpose/`, `context/`, `quality/` and the rest of `architecture/` belong to Root and bind the project by their place; the areas link up to them.

## Kinds and Libraries

A Kind is a Definition with `kind: Kind`; its body says what a Definition of that Kind states and when it is verified. Every other Definition names its Kind by path. `markitect doc std/Operation` prints a Kind with what it declares.

`std` is the one standard Library, with 45 Kinds. Required fields are in bold; a part's own fields stand after it.

| Folder | Kind | A Definition states | Fields | Parts |
|---|---|---|---|---|
| `purpose/` | `std/Project` | what the system is and the problem it removes | | |
| | `std/Goal` | a result the owner wants | **Result**, **Changes**, **Obstacle** | |
| | `std/Stakeholder` | a group of people, by what they do | | `concerns/` Concern (Answered by), `wishes/` Wish |
| | `std/Exclusion` | one thing the system does not do | | |
| `context/` | `std/Term` | a thing of the world the system does not manage | Not | |
| | `std/Constraint` | a restriction imposed from outside | Source, Applies to | Constraints in its folder |
| | `std/Assumption` | what the system takes for granted | **Ensured by**, **Therefore not built** | |
| | `std/ExternalSystem` | a system ours talks to and does not build | **Uses**, **When it fails** | `expectations/` Expectation |
| `domain/` | `std/Concept` | one thing the system manages, in one sentence | | `attributes/` Attribute, `states/` State, `invariants/` Invariant (Derived from), `conditions/` Condition |
| | `std/Operation` | one thing a caller can do to a Concept or ask of it | **Who**, Given, Checked in this order, Returns, **Satisfies** | `conditions/` Condition, `outcomes/` Outcome, `results/` Result |
| | `std/Reaction` | what the system does by itself | **When**, **Satisfies** | `outcomes/` Outcome |
| | `std/Rule` | a property the owner chose that holds across Concepts or Operations, or for the work | Applies to, Derived from | Rules in its folder |
| `interfaces/` | `std/Interface` | how behavior is offered to programs or at a command line | **For**, Follows | `routes/` Route (**Reaches**) |
| | `std/Page` | one page of a site as its user meets it | **For**, **Shows**, All actions | `actions/` Action, `refusals/` Refusal, `displays/` Display |
| | `std/Message` | one message the system sends to a person by itself, such as an e-mail | **To**, **Sent when**, **Holds** | |
| | `std/Shape`, `std/Refusal`, `std/Limit` | how something is written, how a refusal reaches the user, a limit a caller may rely on | | |
| `quality/` | `std/Quality` | how well the system must do something | Measure, Matters to | Qualities in its folder |
| `architecture/` | `std/Strategy` | one fundamental way the design reaches what is asked | Satisfies, Carried out by | Strategies in its folder |
| | `std/Decision` | a choice made and kept | Reason | `consequences/` Consequence, `alternatives/` Alternative |
| | `std/Component` | what exists when the system runs | **Instances**, **Implemented by** | `offers/` Offer, `requirements/` Requirement, `needs/` Need, `rules/` Rule |
| | `std/Connector` | how Components interact | **Between**, **Protocol**, **Limits** | `errors/` Error |
| | `std/Deployment` | where the system runs | **Environment**, **Runs** | `provisions/` Provision |
| `scenarios/` | `std/Scenario` | one path a Stakeholder takes, as a check | **Actor**, **Steps** | `outcomes/` Outcome |
| `administration/` | `std/Task` | one thing the person who runs the system does | **Actor**, **Steps**, **Satisfies** | `outcomes/` Outcome |

- **No file realizes what is given or wanted:** a Project, Goal, Stakeholder, Concern, Wish, Exclusion, Term, Constraint, Assumption, External system, Expectation or Strategy. Their Kinds declare `realized: false`, so they have no registry row; a Goal or a Wish that nothing names in a **Satisfies** field is the auditor's finding.
- **What is given links only what is given.** A Term links only Terms; a Constraint, an Assumption, an External system and an Expectation link only what lies under `context/`; a Goal links that, other Goals and Stakeholders (`link.target`).
- **What binds without a link:** a Rule, a Decision or a Constraint binds its Package and every Package below it. `markitect doc` prints, for each item, those that bind it, so nothing links one only because it holds.
- **Telling Kinds apart:**
  - An Invariant holds for every instance of one Concept, in its `invariants/`; a Rule holds across Concepts or Operations, or for the work.
  - An Assumption is about people and the world; an Expectation is about one External system.
  - A Decision is a choice between alternatives, with its reason; its Consequences are what must follow from it; a Rule is a property that holds across the code or the domain and is checked there.
  - A Message goes to a person; a Connector joins Components, or a Component and an External system.
  - **Who:** and **For:** name roles in backticks, which a Rule defines; **Actor:** links a Stakeholder.
- **Each field on a line of its own.** The Kinds name their fields inside sentences; a Definition starts each field on a line of its own, which is the form that compiles.
- **A relation has one name:** **Satisfies:** what an Operation, Reaction, Task or Strategy exists for; **Derived from:** what a Rule or Invariant follows from; **Reaches:** the Operation of a Route; **Sent when:** what makes a Message due; **Answered by:** what answers a Concern; **Carried out by:** the Decisions and Rules of a Strategy; **Implemented by:** the Packages whose code runs in a Component. **Follows:** names another Package whose Shapes, Refusals and Limits hold for an Interface; its own Package binds it by place already.
- **An API and a command line** are both a Package under `interfaces/` with an Interface and its Routes: a Route names the method and path, or the command and its arguments, in backticks.
- `Package` is Core's own, written without a Library: the anchor of a Package says what it is responsible for and what it keeps to itself, not what it contains or depends on.

A project writes its own Kinds as small files (`kind: Kind`) anywhere in its spec; a Kind several Packages use sits at Root. A Library is a folder of `<Name>.md` Kind files under `.markitect/libraries/<name>/`, named in the manifest. The standard Library ships in the binary at one version, as in `std: 2.1.0`. After a manifest change run `markitect tidy`.

### What a Kind declares

A Kind's front matter may declare the form of its Definitions, and `check` holds every Definition of the Kind to it. `std/Operation` declares:

```yaml
kind: Kind
fields:
  Who: {required: true}
  Given:
  Checked in this order: {links: std/Condition}
  Returns:
  Satisfies: {required: true, links: [std/Goal, std/Constraint, std/Rule, std/Quality, std/Wish]}
parts:
  conditions: std/Condition
  outcomes: std/Outcome
  results: std/Result
```

- `fields` names the fields a Definition may have, by label. `required: true` asks for the field (`field.missing`). `links` in a field names the Kinds a link in the field may name, `Package` for a Package (`field.target`); without it any link may stand there. A label the Kind does not declare (`field.unknown`) and a field that stands twice (`field.repeated`) are errors.
- `parts` names the folders a thing of the Kind has its parts in, each with the Kind, or the list of Kinds, of what lies there; `.` is the thing's own folder, as for a Rule whose parts are Rules. A part in another folder (`part.folder`) or of another Kind (`part.kind`) is an error.
- `links` at the top names the Kinds any link of the Definition may name, in its own sentences and in every field that does not name its Kinds itself (`link.target`), as `std/Term` declares `links: std/Term`.
- `realized: false` declares that no file realizes a Definition of the Kind: it is verified, needs no registry row and is never `definition.unused`.
- `binds: true` declares that a Definition of the Kind binds its Package and every Package below it without a link: each Definition there that a file realizes. A change of it puts all of them back on the list of `status`, so a Rule that holds for one area lives in that area's Package.
- `last: true` declares that a Definition of the Kind is built and verified last, on the whole project, as a `std/Quality` and a `std/Scenario` are.
- A key with nothing after it declares that there is none: `fields:` alone allows no field, `parts:` alone no part, `links:` alone no link. A Kind that leaves a key out declares nothing about it, and nothing is checked.
- Only a Kind declares (`definition.front_matter` elsewhere). A declaration that cannot be read, or that names something that is no Kind, is `kind.schema`.

A project Kind declares its form the same way, and its text says what each field holds. It lies in a folder `kinds/` of the one Package that uses it, or of Root when several do.

## Links, scope, visibility, cycles

- A link is `[[path]]`: the full path from the spec folder, even inside the same Package (`[[domain/members/concepts/member/attributes/member-id]]`), `[[std/Rule]]` for a Library Kind, `[[domain/members]]` for a Package, `[[domain/catalog/operations/borrow-book]]` for a thing with parts (the folder that holds `borrow-book.md`), never another folder. Optional `#section` and `|label` do not matter for resolution. In a Markdown table write the label's bar as `\|`. Links in code spans and code blocks are text. No imports, no relative links.
- **Scope is location.** What a Package defines reaches its whole subtree; a Rule at Root binds the project.
- **Visibility.** Public unless `internal: true`; then only its own Package and Packages below may link to it or use it as a Kind (`link.internal`). Reading is not restricted: agents read internal Definitions, and registry entries may link any Definition.
- **Dependencies.** A link from Package A to a Definition in, or to, another Package B is an edge A to B; using a Kind is an edge. Every Package also has an implicit edge to its parent. Linking up to an ancestor's Definition is free. A parent may link to a child Package but not to a Definition inside it (`link.downward`). Links to Library Kinds make no edge. A cycle of edges is an error (`package.cycle`).
- Break a cycle by moving the shared concept to the common parent, or by inverting the dependency: the needing side describes its need in its own Package and the other side links to it.

## Every link is a relation

The spec behaves like code, and code has no "see also". A link is never decorative: a link to a Definition says that the source relies on what the target states, so a change to the target puts the source back on the list of `status`; a link to a Package says that the source's module depends on that module, or, down the tree, says something about a child of the parent. A link that would only help a reader find something does not belong in the spec; rendered documentation gives navigation. So a pointer such as "described in the [[architecture]]" either relies on its target or goes, and the general never links back to what uses it. Three patterns follow, as code has them:

- **Name a Concept by its identity.** When Definitions name an instance of a Concept only by its identifier, the identity is an Attribute of its own, such as `domain/members/concepts/member/attributes/member-id`: its form and what it names. Each holder links the identity, not the Concept (``**Given:** `memberId`, a [[domain/members/concepts/member/attributes/member-id|member identity]]``). A holder that relies on more than the identity links that too: the Attribute or State it relies on, or the Concept when it relies on what the instance is. This is domain-driven design's typed identity (`MemberId`).
- **A Package publishes what is public.** What is not `internal: true` is what a Package offers other Packages, as Go's exported names are, and `markitect doc PACKAGE` shows it. Do not list a module's Operations and Concepts again in its anchor.
- **Name a module by its Package** where a Definition relies on the module as a whole and on no statement in it: a Component's ``**Implemented by:** [[domain/members]], [[domain/catalog]]``. A sentence that relies on what the other module states links that Definition.

## Registry pages

One page per Package: `.markitect/registry.md` for Root, `.markitect/registry/<package path>.md` for the others (`registry/domain/members.md` for `domain/members`).

```markdown
---
paths:
  - internal/members/**
---
# Registry: domain/members

| File | Realizes | Why |
|---|---|---|
| `internal/members/member.go` | [[domain/members/members]] [[domain/members/concepts/member]] | The member |
```

- Front matter: `paths`, a list of repository-relative patterns, and nothing else (`registry.front_matter`). Root's page has no patterns.
- Patterns: `*` and `?` inside one name; a whole segment `**` matches any depth; a trailing `**` matches what is inside the folder. No `[ ]` or `{ }` (`registry.pattern`).
- Ownership: the most specific pattern wins; a pattern with no wildcard beats any with one; otherwise more literal characters win. Root owns what no one claims. Two Packages with the same pattern: `registry.duplicate_pattern`; equal specificity matching the same file: `registry.ambiguous_owner`.
- Accounting: every file outside `.markitect/` is listed on its owner's page or in the ignore file (`file.unregistered`). Listed on a page of a different owner: `registry.wrong_owner`; listed twice: `registry.duplicate_entry`; listed and ignored: `registry.ignored_file`; excluded by a `.gitignore`: `registry.git_ignored`; below `.markitect/`: `registry.own_file`.
- Rows: tables whose first header is `File`, three cells: file, links to the Definitions it realizes (Definitions of the spec only; none is allowed), a short reason (`registry.table`, `registry.link`). A row names a thing with parts by its folder, which covers its parts. A Definition of a Kind that declares `realized: false` has no row. The registry folder holds only pages (`registry.page`).
- Shared files (README, CI pipeline, compose file) belong to the lowest common parent of the Packages they serve, often Root, and list every Definition they realize.

## Ignore file

`.markitect/ignore` holds files that deliberately belong to no Package, such as `LICENSE`, and a project that has none has no such file. It reads like a `.gitignore`: one pattern per line and `#` for a comment (`ignore.syntax`). Its patterns are the registry's: they start at the top of the repository, a folder's files are `folder/**`, and `!` does not exist.

```text
# Legal text, realizes nothing
LICENSE
```

## Stale realization

A registry entry that names a deleted Definition (`stale.definition`), a missing file (`stale.file`) or a missing Package's page (`stale.page`) is an error of the realization: `check` lists it apart from the spec's errors, `status` lists it for the builder, and the next `/markitect-build` cleans it up.

## Commands

All take `--repo DIR`; `check`, `vet`, `list`, `doc`, `status` and `mark` take `--json`, which prints one document with `format_version` and `kind`. Diagnostics read `file:line: code: message`. Exit 0 ok, 1 found something, 2 could not run. The commands are flat, as Go's are.

| Command | Use while you work on the spec |
|---|---|
| `init [--name TITLE]` | Once per project: manifest, lock, project Definition, registry page, `AGENTS.md`, `CLAUDE.md`, skills and harness. Makes no repository and commits nothing. Never changes an existing spec file. |
| `check [--spec]` | Compile: errors only. `--spec` looks only at the spec and leaves out what the project does not realize yet, which is what a change of the spec asks. A project with nothing registered, where the only errors are `definition.unused`, passes and says that nothing is built yet. |
| `tidy` | Bring `lock.yaml` in line with the manifest after a manifest or Library change, as `go mod tidy` does. |
| `doc TARGET` | A Definition with its text, its parts for a thing with parts and its whole for a part, what it declares for a Kind, what links to it, what it links to, the files that realize it and its record; or a Package with the text of its anchor, its public and its internal Definitions apart, its page and files. |
| `doc --read [FOLDER]` | Every Definition at or below a folder as one text to read, the whole spec without a folder: what the owner reads, or you answer from, when you go through the spec together. |
| `doc --read FOLDER --bound` | The same text, after a list of the Operations and Reactions whose own Package is the folder's, with what binds the folder from outside it before it and what outside it the folder links after it, each once: everything an auditor of one Package reads. |
| `list [--deps] [--packages] [--kinds]` | The Packages and their Definitions; `--packages` prints the Packages alone, and with `--deps` each dependency on one line with the number of links behind it, which is how to read the structure of a large spec; `--deps` adds each Package's coupling, abstractness and distance from the main sequence, what it depends on and the links behind each dependency. `--kinds`, alone, prints every Kind the spec can use with how many Definitions it has of it and in which top-level folders, then the Kinds it has none of: what the spec describes so far. |
| `status [--all] [--package P]` | What is to verify, item by item: each item whose text, Kinds, links, files or evidence changed since a reviewer last verified it, or that has no record yet, with what changed and where to look. Needs a project that compiles. One line says when only spec-problems are left, and a last one when `.markitect/questions.md` exists. |
| `vet [--strict]` | Warnings about the spec's design, as `file:line: code: message`, and one line that counts its notes; `--strict` lists the notes too. Exit 1 only with `--strict` and a warning; notes never change it. |

## Coverage

What a description of a product covers, by folder, in the order in which a project is started. Who answers is decided by who can know, not by the folder. The owner is asked, in the words of their trade, what only they know: most of `purpose/`, `context/`, `domain/` and `interfaces/`, with the deadlines and limits the business sets, such as until when a customer may cancel, and who will look after the running system. What takes an engineer to answer is the architect's proposal in whichever folder it lies, for the owner to accept or change: technical limits and formats, such as the length of a name, the order of checks and the names of refusal codes, how signing in is protected, what holds when another system fails, technical Assumptions, paths, where the system runs and how a test sets the time; and the rest of `quality/`, `architecture/`, `scenarios/` and `administration/`. Each topic is stated in the spec, or ruled out in one of three ways: an Exclusion in `purpose/exclusions/`, a Quality that says its characteristic asks nothing or that nothing is promised, or one sentence in the nearest Definition that the project leaves it free. A topic that is neither is a gap. Identification, traceability, acceptance criteria and the order of the documentation are the tool's, and detailed design is the builder's: they are not on the list.

**`purpose/`**
- What the system is, for whom, and the problem it removes today: `project.md`.
- The results the owner wants: Goals in `purpose/goals/`.
- Who uses the system or is affected by it, what matters to them and what they want to do with it: Stakeholders in `purpose/stakeholders/`, with their Concerns and Wishes.
- What it leaves out, which keeps a first version small: Exclusions in `purpose/exclusions/`.

**`context/`**
- The words of the domain for what the system does not manage: Terms in `context/terms/`.
- What law, the business, other systems or nature impose, personal data among it: Constraints in `context/constraints/`.
- The systems it talks to, what it uses of them and what holds when one fails: External systems in `context/external-systems/`.
- What it takes for granted, and who makes it true: Assumptions in `context/assumptions/`.

**`domain/`**, one Package per area
- What it keeps: Concepts in `concepts/`, with their Attributes and values, States and Invariants.
- Each thing a caller can do or ask, with who may, what is given, its Conditions in order with their refusal codes, its Outcomes and what it returns: Operations in `operations/`.
- What it does by itself, at a time or after something has happened: Reactions in `reactions/`.
- What holds across Operations, such as roles and who counts as which, signing in and sessions, and how long data is kept: Rules in `rules/` of the area, or of `domain/` across areas, among them one Rule in `domain/rules/` that names the refusal codes every area uses.

**`interfaces/`**, one Package per site, API, command line or channel of messages
- How each Operation is reached: an Interface with its Routes, or the Pages of a site with their Actions, Refusals and Displays, among them what a page shows when there is nothing to show.
- Each message the system sends to a person, such as an e-mail: Messages in `messages/` of a Package such as `interfaces/email/`.
- The addresses of a site that a user relies on, such as the link in an e-mail or a bookmark: in the **For:** field of the Page each opens, the path in backticks. Which pages open from a link or a bookmark, and their paths, are the architect's proposal, said to the owner as what the users do, such as opening the appointment from the confirmation e-mail.
- How a site looks: whether it must look like something that exists, such as a logo, colours or a present website, or may look plain, which the architect proposes where the owner's notes say nothing. What exists is a Constraint in `context/constraints/` that says where it is, with **Applies to:** the site; a look the project chooses is a Decision in the site's `decisions/`; otherwise one sentence in the site's anchor says that its looks are free. A spec that is silent here lets twenty different sites pass.
- What holds for the whole site, API, command line or channel: Shapes, such as the language of pages and messages, Refusals and Limits, in the Package's `shapes/`, `refusals/` and `limits/`, and in those folders of `interfaces/` itself for what holds for every interface.

**`quality/`**
- One Quality for each characteristic of ISO/IEC 25010: functional suitability, performance efficiency, compatibility, interaction capability, reliability, security, maintainability, flexibility and safety; `quality/quality` names those that decide the design.
- In the folder of each, its requirements with a Measure, such as response time under load, capacity and growth, availability, recovery after a loss, and accessibility.

**`architecture/`**
- The solution strategy: Strategies in `architecture/strategy/`.
- The technology, what the project depends on and the other choices it keeps, with their reasons and rejected alternatives: Decisions in `architecture/decisions/`. Propose something plain that fits, unless the owner has named a preference.
- What holds across the code, one Rule with parts each in `architecture/rules/`: persistence and transactions, concurrency, security (threats, how secrets and passwords are kept, secure communication, an audit trail, the headers a site sends), faults, logging and monitoring, configuration, how stored data is brought to a newer form, build and release, layout, with how an area adds files of its own for what it plugs in, such as its routes or its tables, where it would otherwise change a file all areas share, and tests with the command that runs them, which the reviewer runs as evidence, and how a test sets the time of the running program, where a Reaction or a Scenario depends on time passing.
- What runs, how its parts talk and where it runs: Components, Connectors and Deployments in `architecture/runtime/`. How a Component starts and stops, and what it does for its own upkeep, are Rules in its `rules/`. `architecture/runtime/` is a Package once it names what lies in the areas, since it then depends on them, and nothing in an area links into it; a Rule that an area relies on lies in `architecture/rules/`.

**`scenarios/`**
- The paths through several areas worth checking: Scenarios in `scenarios/`. A Scenario is verified by a test that runs what the project delivers, the built program started as the spec says, and walks it, so state each step so that it can be taken through the interface its Actor uses. A Scenario that depends on time passing needs the Rule of `architecture/rules/` on how a test sets the time.

**`administration/`**
- What the person who runs the system does to it, such as installing, deploying and rolling back, backing up and restoring, upgrading and finding a fault: Tasks in `administration/tasks/`.

## Warnings and notes (`vet`)

Warnings: `vet.coupling` (more than 5 dependencies), `vet.unstable_dependency` (depends on a less stable Package), `vet.definition_size` (over 1000 words), `vet.package_size` (over 25 Definitions, the parts of a thing not counted; Root only in a spec without Packages), `vet.hub` (a Definition other than a Kind, Root's included, whose change alone would put more than a fifth of the spec, and more than 10 Definitions, back on the list to verify through links; what it binds by its place is not counted), `vet.thin_package` (only its anchor and one child Package), `vet.no_paths` (once something is built: a Package other than Root claims no path pattern, so it owns no file).

Notes: `vet.unlinked_mention` (the title or name of a Definition that is no part and that a file realizes, in the prose of a Definition whose thing does not link it; also where a link would close a Package cycle or add a Package dependency), `vet.statements` (over 20 list items and table rows), `vet.placement` (every Definition that links to it lies in one Package below it), `vet.hotspot` (how often its own text changed between verdicts, times what links to it, is above a tenth of the spec and 20), `vet.risk` (its last `ok` or `skip` rests on evidence for too few of the cases the reviewer counted, or it has too many cases to verify well).

## Error codes

| Group | Codes |
|---|---|
| Project | `project.file` (a file under `.markitect/` has no place in the layout), `project.read` |
| Definition and spec | `definition.front_matter`, `definition.title`, `spec.name`, `spec.file`, `spec.collision`, `spec.package`, `spec.reserved` |
| Kind and Library | `kind.unresolved`, `kind.not_a_kind`, `kind.schema`, `library.name`, `library.file`, `library.not_a_kind`, `library.internal`, `library.version`, `library.reserved`, `library.missing`, `library.undeclared` |
| Fields and parts | `field.unknown`, `field.missing`, `field.repeated`, `field.target`, `part.folder`, `part.kind` |
| Manifest and lock | `manifest.syntax`, `manifest.missing`, `lock.mismatch`, `lock.missing`, `lock.syntax` |
| Links and Packages | `link.unresolved`, `link.syntax`, `link.internal`, `link.target`, `link.downward`, `package.cycle` |
| Registry and ignore | `file.unregistered`, `registry.front_matter`, `registry.page`, `registry.pattern`, `registry.duplicate_pattern`, `registry.ambiguous_owner`, `registry.wrong_owner`, `registry.duplicate_entry`, `registry.ignored_file`, `registry.git_ignored`, `registry.own_file`, `registry.table`, `registry.link`, `ignore.syntax` |
| Harness, record and `doc` | `harness.syntax` (`.markitect/harness.yaml` cannot be read; the owner fixes it by hand), `record.unreadable` (a line of `.markitect/review.sum` cannot be read; the owner fixes or removes it), `doc.unknown` (the target of `doc` is neither a Definition nor a Package) |

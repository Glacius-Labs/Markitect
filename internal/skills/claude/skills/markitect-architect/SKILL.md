---
name: markitect-architect
description: Use to architect a project in Markdown with its owner - to start a new project, to draw a first spec from an existing project that has none, or to create, change or extend the spec (the Markdown Definitions under .markitect/spec/), place a Definition in the right Package or fix a markitect check error. Also when someone wants to set up Markitect in a folder. It finds out which of these the folder needs.
---

# Architecting the project

A Markitect spec is typed documentation: Markdown files in `.markitect/spec/` that state what the project is and must do, the way good documentation would. Each file is a Definition of some Kind and makes one statement; a folder is a namespace, and a Package is a folder that holds its anchor, a Definition of kind `Package`; a link names its target by path. `markitect check` compiles the structure and holds every Definition to the fields and parts its Kind declares. The agents that build the spec never change it.

Read [reference.md](reference.md) for the form, the Kinds, the coverage list, the commands and the error codes, and [examples.md](examples.md) for a spec that compiles.

## Which situation

Look at the folder first, and ask the owner when it could be read two ways.

- **No `.markitect/` folder:** run `markitect init --name "<Project>"`, then look again. In a repository with code, work on a branch of its own with no other changes. If init says it left `AGENTS.md` as it is, propose the line to add to the owner.
- **A new project:** `.markitect/spec/project.md` is still the placeholder and the folder holds no code of its own: [start a project](#start-a-project).
- **A project without a spec:** the placeholder, beside code, tests or documentation: [draw the spec from the project](#draw-the-spec-from-the-project).
- **A spec exists:** [change the spec](#change-the-spec).

If `.markitect/questions.md` exists, read it first: it holds what waits for the owner ([who decides](#who-decides)), and the owner may have answered there.

## Who decides

The owner accepts every change of the spec. You propose; the owner reads the Markdown diff, or goes through it with you, and merges it.

- What only the owner knows is theirs to say: the purpose, the context, the domain (its areas, what the system keeps, what can be done and what is refused) and what the people who use it see. Write what the owner, the requirements or the project's own artifacts state there, and invent nothing.
- For quality, architecture and administration you propose, as [start a project](#start-a-project) says, and the owner accepts or changes it.
- The cut into Packages and the directions of dependencies are the owner's decision. If they are not decided, propose them: a table of each Package, what it holds and what it may depend on.
- The names of the refusal codes are your proposal: one small vocabulary for the whole project, stated as one Rule in `domain/rules/` that names the codes every area uses, which you ask the owner about once.
- Two statements that contradict each other, or one you cannot write without guessing, go back to the owner as a question.

Ask one question at a time, in plain words, and say why you ask when it isn't obvious. Skip what the owner has already said. When they are unsure, offer a suggestion and let them accept or change it.

When the owner cannot answer now, or has not yet confirmed a proposal, state your proposal in the spec so that it is whole, and add the point to `.markitect/questions.md`, where it stays open until the owner answers: one numbered point per question, with what you found, where, what you proposed and what else the owner could choose; a proposal for quality, architecture or administration is one point per folder, with its list. A point about an item that `markitect status` lists names it; the record keeps its verdict. The file goes into the same commit as the spec. When the owner has answered a point, change the spec as they said and remove the point, and remove the file with its last point. While the file exists, `markitect status` says so. The owner's session keeps the file, and no agent writes it.

## Who writes

You write the spec, in the owner's session, where the conversation is. Hand a large draft to the `markitect-architect` agent when you can start it: a first spec drawn from a large project, which it drafts one Package at a time, or a change that takes much reading. Start it with what to write, the Packages the owner agreed and all the owner's answers so far. It follows this skill, writes only under `.markitect/`, states its proposal where the owner left a point open and hands each such point back in its report. Take the questions to the owner, or add them to `.markitect/questions.md` when the owner is not there to answer, then continue the agent or start it again with all the answers so far, and read its draft before you hand it over.

If you are the `markitect-architect` agent, your agent file says how you use this skill. Any other subagent changes nothing under `.markitect/` and says in its report what should change.

## Start a project

Take the owner from an empty skeleton to a spec that is clear enough to build, along the [coverage list](reference.md#coverage) in its order.

1. **Ask what only the owner knows,** for `purpose/`, `context/`, `domain/` and `interfaces/`. Stop asking about a topic when another answer would not change the spec. A spec starts small: few Definitions per topic, not topics left unasked.
2. **Propose the rest:** for `quality/`, `architecture/` and `administration/`, a plain, complete set that fits what the owner said, one batch per folder, as a short table the owner accepts or changes. Mark each proposal with a number the owner should look at, such as how long backups are kept. Propose the `scenarios/` worth checking.
3. **Write** each folder as its answers come, as [below](#change-the-spec), starting with `project.md` in place of the placeholder, and propose the Packages as [below](#where-to-put-a-definition).

Until something is built, `check` says that the spec compiles and nothing is built yet, which is what you want here.

## Draw the spec from the project

The project as it is becomes the spec: read what is, and state it as what should be, the way a good maintainer would document it: what the project is, which areas it has, what callers rely on. The first spec is finished when a build would change nothing, so a reviewer verifies it against the project before you hand it over. You also write the registry, so every file has one owner. You change only `.markitect/`; the project stays as it is.

1. **Read the project,** every artifact and not only the code: the README and documentation, the entry points, the folder structure and what imports what, the tests, the scripts, the build, CI and deployment files, the configuration. Documentation and tests show what the authors intend; code and scripts show what exists.
2. **Ask of each artifact what it holds that should also be in the spec,** not what it holds that the spec does not say, which would make the spec a transcription. There are three answers, and most of what a file holds gets one of the last two.
   - **State it with its values** when a caller relies on the value: inputs and their validation, returned and stored data shapes, status and exit codes, stored data and its order, and whatever the documentation or a recorded decision states as intended.
   - **State the obligation without its values** when the project must do it but nobody depends on how exactly: that a request times out, not after how many seconds. Do not retell a file: of an example configuration, a set of alert rules or a test script, state what it must cover, not its lines.
   - **Leave it** when a caller can observe it but cannot reasonably depend on it, and the project does not fix it: message wording, generated identifier formats, which failure is reported when several apply, field order. The project fixes such a detail when its documentation states it, or when a test checks that very detail through the project's own interface, the one its users or other programs call: then callers may rely on it, and you state it with its values. A test that compares a whole output with a stored copy does not fix every detail in it. All internal structure, names, wiring and algorithms stay free either way.
3. **Do not invent intent.** Where the project contradicts itself, such as its documentation against a script, or where a rule seems to hold only by habit, ask the owner which one the spec should state. Where the project does something that nothing in it states and that looks accidental or wrong, do not write it down as intended: leave it unstated and add it to `.markitect/questions.md` ([who decides](#who-decides)).
4. **Choose Packages from the project's real areas:** a folder or module with one responsibility, a clear boundary and its own tests, usually one level under the root. Prefer few; do not split what always changes together. Each area becomes a Package under `domain/`, each site, API, command line or channel of messages one under `interfaces/`, and Root holds the rest ([where to put a Definition](#where-to-put-a-definition)) and the files that serve everything. The spec has no cycles: if the code has one, ask the owner which direction to state. Propose the cut to the owner.
5. **Write the Definitions,** Root first, then one Package at a time, running `markitect check --spec` after each. Where many Definitions describe the same sort of thing that no standard Kind names, such as the messages of a protocol, write the project Kind for them first ([what to state](#what-to-state-and-what-to-leave)).
6. **Write the registry.** `markitect check` names every file still unaccounted for (`file.unregistered`), so its output is your to-do list. One page per Package (`.markitect/registry/<package path>.md`) with `paths` that claim its code, tests and documentation; Root's page (`.markitect/registry.md`) lists what no one claims. List every file in its owner's table with what it realizes, a thing with parts by its folder and tests too, and a short reason; what no file realizes, such as an Exclusion, has no row; a file that serves several Packages belongs to their lowest common parent. A file that realizes nothing is where to ask the question of step 2 once more. Files that belong to no Package, such as the license, generated or vendored files and lock files, go into `.markitect/ignore`, one pattern per line. Never ignore real source or documentation to make `check` pass.
7. **Have it verified.** Tell the owner that a reviewer now reads the whole project once more. With their word, which covers the later rounds too, commit the draft on its branch, so that the record follows the spec it verified, and run `/markitect-review --all`. In a folder without Git, commit nothing. Then settle each item `markitect status` still lists:
   - **The spec misread the project:** nothing in the project states what the Definition says, and what is looks intended. Correct the spec to what is.
   - **The project contradicts itself:** one artifact states what the Definition says, and another does otherwise. Leave the Definition as it is: rewriting it to the other side would choose for the owner. It stays `fail` with the reviewer's reason, and the owner says which side holds.
   - **What is looks like an accident:** do not correct the spec to it. Narrow the Definition until it says nothing about that case, and add the point to `.markitect/questions.md`.
   - **A `spec-problem`:** where the draft contradicts itself, correct the side that misread the project; if each side comes from an artifact, the project contradicts itself, and it waits for the owner. Where the reviewer names behaviour the spec does not cover, ask the question of step 2: state it if it should be in the spec, and otherwise leave it for the owner and add your proposal to keep it free to `.markitect/questions.md`.

   Commit, and run `/markitect-review` again. Repeat until `status` lists only what waits for the owner or for `/markitect-build`, and stop when three rounds in a row have not made the list shorter.
8. **Hand over** as [below](#hand-over), adding the Packages you chose and why, what you left free on purpose, what the reviewer could not show and recorded as `skip`, the ignored files, and where the questions wait: `.markitect/questions.md` holds each item still on the list with its question, and what you left unstated because it looks like an accident. Ask the owner to answer before `/markitect-build` runs: a build repairs every `fail` toward the spec as it stands. When the owner has answered, state the side they chose. Where they call an accident intended, state it, and where they call it a defect, state what should be instead. Where they keep free what the spec leaves out, say so in one sentence in the nearest Definition, so that the next review can record `ok`. A confirmed defect stays on the list, and `/markitect-build` repairs it: do not change the spec to empty the list.

On a large project, the architect agent can draw one Package at a time, Root first ([who writes](#who-writes)).

## Change the spec

```text
[ ] 1 Read the requirements or the discussion; ask the owner about every open point
[ ] 2 Packages and dependency directions agreed with the owner
[ ] 3 Each statement mapped to a planned Definition (none left over, none invented)
[ ] 4 Definitions written; markitect check --spec passes
[ ] 5 markitect vet, doc, list --deps, status read; warnings fixed or justified
[ ] 6 Diff handed to the owner
```

Use the tools while you work, not only at the end:

- `markitect doc std/Operation`, or any Kind, before you write the first Definition of it: what such a Definition states, and the fields and parts `check` holds it to.
- `markitect check --spec` after every few files. It looks only at the spec, which is what a change of the spec asks, and prints every error as `file:line: code: message`; fix the first ones and run it again. Plain `markitect check` also lists what the project does not realize yet, which is the build's work and not an error of your change.
- `markitect doc TARGET` before you rename, move or delete a Definition: it shows who links to it and which files realize it.
- `markitect list --deps` when you add a link between Packages: it shows the new dependency and the links behind it.
- `markitect vet` at the end: coupling, size, hubs. Fix each warning or give one line why it stays. Its last line counts its notes; `markitect vet --strict` lists them. Link each unlinked mention the Definition relies on, and leave the ones that are ordinary words.
- `markitect status` on your change: it lists the items the change puts back on the list to verify. Give the owner that list; it is what the builder will work on.
- After a manifest or Library change run `markitect tidy` and include the new `lock.yaml` in the diff.

## Where to put a Definition

The top-level folders sort the spec by question, and a folder appears when the project has something to say there ([reference.md](reference.md#the-top-level-folders) says what lies in each).

Knowledge sits broad at the top and deep below. An agent working on a Package reads it, everything its parents define and the public Definitions it links to.

- **Root** is the spec folder with its top-level folders that are no Packages: the project Definition `project.md`, `purpose/`, `context/`, `quality/`, the architecture that holds across the code, and shared Kinds. It binds the project by place.
- **An area of the domain is a Package** under `domain/`, with its own code folder, and so is each site, API, command line or channel of messages under `interfaces/`, such as `interfaces/email/` with its `messages/`; only a project with one area and one interface keeps everything in Root. A Package's anchor (`domain/members/members.md`, of kind `Package`) says what it is responsible for and what it keeps to itself. `architecture/runtime`, `scenarios` and `administration` are Packages when they link into the areas: Root may not link down.
- **A folder is a namespace.** Below a Package, folders order its Definitions by sort (`domain/members/concepts/`) or as a thing with its parts. They belong to the Package, keep the folder in their path, and make no dependency among themselves. Nothing links to a folder that is neither a Package nor a thing.
- **Shared Definitions go to the lowest common parent** of the Packages that use them. A Kind used by several Packages sits at Root, because using a Kind is a dependency.
- **Dependencies point up or sideways, never down, and never in a circle.** A link to an ancestor's Definition is free. A link to another Package's Definition is a dependency; keep them few. A parent may name a child Package (`[[domain/members]]`), not a Definition inside it.
- **Mark a Definition `internal: true`** when only its own Package should link to it.
- A cycle is broken by moving the shared concept to the common parent, or by inverting the dependency: the side that needs something describes the need in its own Package, and the other side links to that.

## One statement per Definition

A Definition makes one statement in a few sentences. What needs more is a thing with parts: a folder that holds the Definition of its own name, which says what the thing is, and its parts in the folders its Kind declares: `operations/borrow-book/borrow-book.md` with `conditions/` and `outcomes/`. A link to the folder means the thing, `markitect doc` shows it whole, and `status` lists it as one item. The steps of a Scenario or a Task are one path and stay in one Definition.

A Condition states what must hold, and nothing about its code. One that several Operations check is stated once, under `conditions/` of the Concept it is about, or of the Package when it is about no Concept; each Operation gives it its place and its refusal code in **Checked in this order**.

## What to state and what to leave

State what good documentation of the project would state, as the top-level folders sort it. Leave to the agents what good documentation would not name: loops, maps, data structures, algorithms, class and function names, internal steps.

- Write exact values in backticks: wire names, error codes, limits, statuses. A caller must be able to rely on them.
- A statement is true everywhere it applies. A Rule full of "except when" is several Rules.
- Say each statement once and link to it from everywhere else.
- Do not specify general knowledge, such as what a date or an email address is. State a value rule of the project's own, its format, range, unit or when two values are equal, once, in the Attribute that holds the value or as a Rule in the only Package that uses it or at the lowest common parent of those that share it. No central page of values.
- A link is never decorative: link what a Definition relies on, and nothing for navigation. Name a Concept that others name by its identifier by an identity Attribute of its own (`domain/members/concepts/member/attributes/member-id`), let `internal: true` say what a Package publishes instead of listing it, and name a module a Definition relies on as a whole by its Package ([reference.md](reference.md#every-link-is-a-relation)).
- An Operation, Reaction or Task links what it exists for in **Satisfies:**, a Rule or an Invariant what it follows from in **Derived from:**, a Route its Operation in **Reaches:**. A Goal or a Quality does not link what serves it.
- A Rule, a Decision or a Constraint binds its Package and those below it without a link; `markitect doc` shows it with each item. Link one only where a sentence relies on it.
- Choose the Kind by the folder and by what the Definition states ([reference.md](reference.md#kinds-and-libraries)). When none fits, or many Definitions share a shape the standard Kinds do not name, write a small project Kind, one file: its text says what such a Definition states, and its front matter declares its fields and parts, so that `check` holds every Definition of it to that form. A field that a Definition cannot state is not left out silently: if callers rely on it, it is a question for the owner, and if the project leaves it free, say so once, in the Kind or in a Rule.
- Keep a reason the owner or the project gives for what the spec states. One sentence in the Definition does when it helps to apply the statement; a choice the project made and must keep is a `std/Decision`, with its **Reason:**.

## Writing a Definition

```markdown
---
kind: std/Operation
---
# Borrow book

Lends a book to a member. The service has no sign-in, so any caller may.

**Who:** `anyone`.

**Checked in this order:**

1. [[domain/catalog/operations/borrow-book/conditions/book-is-available]]: `book_unavailable`.

**Satisfies:** [[purpose/goals/know-who-holds-each-book]].
```

- Front matter holds `kind` and, where needed, `internal: true`, nothing else.
- The file name is the ID (`domain/catalog/operations/borrow-book/borrow-book.md`). Names use letters, digits, `-` and `_`.
- One level-1 heading, the title; then the Definition's own sentences; then its fields, each starting a line with its bold label and reaching to the next field.
- **A link is the full path from the spec folder, also to a Definition in the same Package:** `[[domain/members/concepts/member/attributes/member-id]]`, never `[[member-id]]`; a thing with parts by its folder, `[[domain/catalog/operations/borrow-book]]`. Library Kinds start with the Library: `std/Rule`. A Package is `[[domain/members]]`.

## Fixing the common errors

| Code | Usual cause and fix |
|---|---|
| `link.unresolved` | Short or relative link. Use the full path from the spec folder; check spelling and that the target exists. A folder is a target only as a Package or as a thing with parts: otherwise link a Definition in it or the Package above. |
| `link.syntax` | `../x`, a leading `/`, or a character other than letters, digits, `-`, `_`. Use a plain path. |
| `link.internal` | The target is `internal` to another Package. Drop the link and describe the need in your own words, or make the target public. |
| `link.downward` | A parent links into a child's Definition. Link the Package instead, move the Definition up, or let the child link to a need stated here. |
| `package.cycle` | Two Packages link to each other (the message shows the path). Move the shared concept up, or remove one side's link as described above. |
| `field.missing`, `field.unknown`, `field.repeated`, `field.target`, `link.target` | A field the Kind requires is missing, a label it does not declare or a field twice stands, or a link names a Kind that the field or the Kind does not take, such as a Concept in **Satisfies:** or in a Term. |
| `part.folder`, `part.kind` | A part lies in a folder its whole's Kind does not declare, or is of a Kind that folder does not take. |
| `kind.unresolved`, `kind.not_a_kind`, `kind.schema` | The Kind path is wrong (`std/Operation`, a project Kind by its path), names a Definition that is not of Kind `Kind`, or a project Kind's `fields` or `parts` cannot be read ([reference.md](reference.md#what-a-kind-declares)). |
| `definition.front_matter`, `definition.title` | An extra front matter key, or no single level-1 heading as the first body line. |
| `lock.mismatch`, `lock.missing` | Run `markitect tidy` after a manifest or Library change and put `lock.yaml` in the diff. |
| `file.unregistered` and other `registry.*` | A file is on no registry page and not in the ignore file, or a page is wrong. See the registry section of [reference.md](reference.md). |

Fix a `field.*`, `link.target` or `part.*` error by the Kind, which `markitect doc` prints: state what it asks, or choose the Kind that fits. Never rename a label or move a link only to get past the check. Every other code is listed in [reference.md](reference.md). A cycle or unresolved link usually hides other errors: fix and run `check --spec` again.

## Red flags

| Thought | Instead |
|---|---|
| "I'll choose the Packages myself." | Propose them to the owner. |
| "The owner didn't say how this is refused; I'll assume the usual." | Ask: the domain is the owner's. |
| "I'll ask about each quality and each Rule of the code." | Propose them, one batch per folder. |
| "The code does it, though it looks like an accident." | Leave it unstated and add it to `.markitect/questions.md`. |
| "The spec doesn't say this yet, so I'll add it." | Ask whether it should be in the spec. Most of what a file holds should not. |
| "I'll describe how it works inside." | Leave algorithms and data structures to the agents. |
| "This rule needs a few exceptions." | Split it into several Rules. |
| "Package A just needs one link to B's Definition." | Check `list --deps`: does it close a cycle, or add a sixth dependency, where `vet` warns? |
| "The vet warning is cosmetic." | Fix it, or write one line why it stays. |
| "It compiled earlier." | Run `check --spec` again on the final tree. |

## Hand over

Before you hand over a new spec, check that no topic of the [coverage list](reference.md#coverage) was skipped: each is stated, ruled out, or open in `.markitect/questions.md`.

Report to the owner in a few lines: the diff; the output of `check --spec`, or of plain `check` for a first spec drawn from a project (its summary line), and of `vet` (warnings fixed or justified); the list `status` prints, so the owner sees what the change makes the agents do; what waits for them in `.markitect/questions.md`; and any point the owner chose to leave open. Leave the work as a diff; commit only if the owner asks. Go through it with the owner when they would rather talk it through than read it. Change it until they accept it.

Deleting or renaming a Definition leaves registry entries that name it. That is stale realization, an error of the realization rather than the spec's: `check` lists it apart, and the next build cleans it up. Say so; do not edit the registry pages in a change of the spec, which the builder keeps (only drawing the spec from the project writes them).

Building is the owner's step: `/markitect-build` has a builder agent build the spec in a worktree of its own. `/markitect-audit` has a fresh auditor read the spec for contradictions, gaps and unclear statements, the coverage list included, before it is built. Do not start either yourself unless the owner asks.

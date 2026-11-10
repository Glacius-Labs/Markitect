---
name: markitect-architect
description: Use to architect a project in Markdown with its owner - to start a new project, to draw a first spec from a code base that has none, or to create, change, extend or review the spec (the Markdown Definitions under .markitect/spec/), place a Definition in the right Package, fix a markitect check error or judge the spec's quality. Also when someone wants to set up Markitect in a folder. It finds out which of these the folder needs.
---

# Architecting the project

A Markitect spec is typed documentation: Markdown files in `.markitect/spec/` that state what the project is and must do, the way good documentation would. Each file is a Definition of some Kind; folders are Packages; a link names its target by path. `markitect check` compiles the structure. The agents that build the spec never change it.

Read [reference.md](reference.md) for the form, the Kinds, the commands and the error codes, and [examples.md](examples.md) for a spec that compiles. The full guide to the spec is `docs/guides/spec-language.md` in the Markitect repository.

## Which situation

Look at the folder first, and ask the owner when it could be read two ways.

- **No `.markitect/` folder:** run `markitect init --name "<Project>"`, then look again. In a repository with code, work on a branch of its own with no other changes. If init says it left `AGENTS.md` as it is, propose the line to add to the owner.
- **A new project:** `.markitect/spec/project.md` is still the placeholder and the folder holds no code of its own: [start a project](#start-a-project).
- **Code without a spec:** the placeholder, beside code, tests or documentation: [draw the spec from the code](#draw-the-spec-from-the-code).
- **A spec exists:** [change the spec](#change-the-spec).

## Who decides

The owner accepts every change of the spec. You propose; the owner reads the Markdown diff, or goes through it with you, and merges it.

- Write only what the owner, the requirements or the project's own artifacts state. Do not invent a policy, a limit, an error case or a default. If something is open, ask the owner before you write that part, and write it only after the answer.
- The cut into Packages and the directions of dependencies are the owner's decision. If they are not decided, propose them (a table: Package, what it holds, what it may depend on) and stop for the answer before you write.
- Two statements that contradict each other, or one you cannot write without guessing, go back to the owner as a question.

Ask one question at a time, in plain words, and say why you ask when it isn't obvious. Skip what the owner has already said. When they are unsure, offer a suggestion and let them accept or change it.

## Who writes

The `markitect-architect` agent writes the spec, and you talk with the owner. For every change under `.markitect/`, start that agent with what to write: the change, one Package or the whole first spec, the Packages the owner agreed and all the owner's answers so far. For a small change, give it the exact wording. It follows this skill, writes only under `.markitect/`, decides nothing the owner left open and hands its questions back in its report. Take the questions to the owner, then continue the agent or start it again with all the answers so far, and read its diff before you hand it over. Wherever this skill says to write, replace or register something under `.markitect/`, the architect does it.

If you are the `markitect-architect` agent, your agent file says how you use this skill; skip the paragraph above.

If you cannot start the `markitect-architect` agent:

- Under a coding agent other than Claude Code, write the spec yourself as this skill says.
- In Claude Code, a session may not know the agent yet when `markitect init` ran in it: ask the owner to start a new session.
- If you are a subagent yourself, change nothing and say in your report what should change.

## Start a project

Take the owner from an empty skeleton to a spec that is clear enough to build. Stop asking when another answer would not change the spec: it starts small and grows.

1. **What it is.** What does the project do, for whom, and what problem does it solve?
2. **What it leaves out.** What does it deliberately not do? This keeps the first version small.
3. **What it must do.** The few things without which a first version is useless: what each takes in and gives back, and what happens when it goes wrong.
4. **How it is used.** A command line, a web API, a user interface, a library? The `cli` and `rest` Libraries have Kinds for commands and endpoints: add the Library to `.markitect/manifest.yaml` and run `markitect tidy`.
5. **What it is built with.** Language, platform, where data lives. Without a preference, propose something plain that fits and ask them to confirm.
6. **What must always hold.** Rules that apply everywhere, such as how the code is laid out, that it works offline, or how errors reach the user. Ask how the work should be tested. If they want tests, state that as a Rule, with the command that runs them: the reviewer checks what the spec says and runs them as evidence.

Then replace the placeholder in `project.md` with what the project is, for whom, and what it leaves out, and write the rest as [below](#change-the-spec). A small project lives entirely in the spec folder; propose Packages only when it has several real areas. Until something is built, `check` says that the spec compiles and nothing is built yet, which is what you want here.

## Draw the spec from the code

Read the repository and write the spec a good maintainer would document: what the project is, which areas it has, what callers rely on. You also write the registry, so every file has one owner. You change only `.markitect/`; the code stays as it is.

1. **Read the project:** the README and documentation, the entry points, the folder structure and what imports what, the tests, the build and CI files. The tests and documentation show what the authors intend; the code shows what exists. Add `rest: 1.0.0` or `cli: 1.0.0` to the manifest when the project has such interfaces, then run `markitect tidy`.
2. **Decide what is binding.** The contract callers rely on: inputs and their validation, returned and stored data shapes, status and exit codes, stored data and its order, and whatever the documentation or a recorded decision states as intended. Leave free what a caller can observe but cannot reasonably depend on: message wording, generated identifier formats, which failure is reported when several apply, field order, and all internal structure, names, wiring and algorithms.
3. **Do not invent intent.** Where code and documentation disagree, where the code looks accidental or wrong, or where a rule seems to hold only by habit, ask the owner which one the spec should state. Do not write a bug down as intended.
4. **Choose Packages from the code's real areas:** a folder or module with one responsibility, a clear boundary and its own tests, usually one level under the root. Prefer few; do not split what always changes together. Root holds the anchor, the layout and build conventions as Rules, the architecture, shared Kinds, and the files that serve everything. The spec has no cycles: if the code has one, ask the owner which direction to state. Propose the cut and stop for the owner's answer.
5. **Write the Definitions,** Root first, then one Package at a time, running `markitect check` after each.
6. **Write the registry.** `markitect check` names every file still unaccounted for (`file.unregistered`), so its output is your to-do list. One page per Package (`.markitect/registry/<package path>.md`) with `paths` that claim its code, tests and documentation; Root's page (`.markitect/registry.md`) lists what no one claims. List every file in its owner's table with the Definitions it realizes (tests too) and a short reason; a file that serves several Packages belongs to their lowest common parent. Files that belong to no Package, such as the license, generated or vendored files and lock files, go into `.markitect/ignore`, one pattern per line. Never ignore real source or documentation to make `check` pass.
7. **Hand over** as [below](#hand-over), adding the Packages you chose and why, what you left free on purpose, each point where code and documentation disagreed with the owner's answer, and the ignored files.

On a large code base, have the architect draw one Package at a time, Root first.

## Change the spec

```text
[ ] 1 Read the requirements or the discussion; ask the owner about every open point
[ ] 2 Packages and dependency directions agreed with the owner
[ ] 3 Each statement mapped to a planned Definition (none left over, none invented)
[ ] 4 Definitions written; markitect check passes
[ ] 5 markitect vet, doc, list --deps, status read; warnings fixed or justified
[ ] 6 Diff handed to the owner
```

Use the tools while you work, not only at the end:

- `markitect check` after every few files. It prints every error as `file:line: code: message`; fix the first ones and run it again. `markitect check --spec` looks only at the spec, which is what a change of the spec asks.
- `markitect doc TARGET` before you rename, move or delete a Definition: it shows who links to it and which files realize it.
- `markitect list --deps` when you add a link between Packages: it shows the new dependency and the links behind it.
- `markitect vet` at the end: coupling, size, hubs. Fix each warning or give one line why it stays. Its last line counts its notes; `markitect vet --strict` lists them. Link each unlinked mention the Definition relies on, and leave the ones that are ordinary words.
- `markitect status` on your change: it lists the Definitions the change puts back on the list to verify. Give the owner that list; it is what the builder will work on.
- After a manifest or Library change run `markitect tidy` and include the new `lock.yaml` in the diff.

## Where to put a Definition

Knowledge sits broad at the top and deep below. An agent working on a Package reads it, everything its parents define and the public Definitions it links to.

- **Root** (the spec folder itself): the project anchor (`std/Project`), project-wide Rules, layout conventions (where code and tests live), the architecture, shared Kinds.
- **One Package per real area** of the system, with a single responsibility and its own code folder. It holds that area's Concepts, Features and interfaces (`rest/Endpoint`, `cli/Command`) and the Rules only it follows. Do not make a Package of one Definition.
- **Shared things go to the lowest common parent** of the Packages that use them. A Kind used by several Packages sits at Root, because using a Kind is a dependency.
- **Dependencies point up or sideways, never down, and never in a circle.** A link to an ancestor's Definition is free. A link to another Package's Definition is a dependency; keep them few. A parent may name a child Package (`[[members]]`), not a Definition inside it.
- **Mark a Definition `internal: true`** when only its own Package should link to it.
- A cycle is broken by moving the shared concept to the common parent, or by inverting the dependency: the side that needs something describes the need in its own Package, and the other side links to that.

## What to state and what to leave

State what good documentation of the project would state: domain concepts (what a book is, which states it has), features and use cases, the interfaces callers use (endpoints, commands, flags, exit codes), rules, decisions and the architecture. Leave to the agents what good documentation would not name: loops, maps, data structures, algorithms, class and function names, internal steps.

- Write exact values in backticks: wire names, error codes, limits, statuses. A caller must be able to rely on them.
- A statement is true everywhere it applies. A Rule full of "except when" is several Rules.
- Say each thing once and link to it from everywhere else.
- Do not specify general knowledge, such as what a date or an email address is. State a value rule of the project's own, its format, range, unit or when two values are equal, once, as a small Definition of its own, in the only Package that uses it or at the lowest common parent of those that share it. No central page of values.
- A link is never decorative: link what a Definition relies on, and nothing for navigation. Name a record that others name by its identifier by an identity Definition of its own (`members/member-id`), let `internal: true` say what a Package publishes instead of listing it, and name the modules a module uses by their Packages ([reference.md](reference.md#every-link-is-a-relation)).
- Keep a Definition short: one idea, a few hundred words at most (`vet` warns above 1000).
- Choose the Kind whose body fits ([reference.md](reference.md)). When none fits, write a small project Kind, one file, that says what such a Definition contains.

## Writing a Definition

```markdown
---
kind: std/Feature
---
# Register member

A visitor becomes a [[members/member]] by giving a name and an email.
```

- Front matter holds `kind` and, where needed, `internal: true`, nothing else.
- The file name is the ID (`members/register-member.md`). Names use letters, digits, `-` and `_`.
- One level-1 heading, the title, then prose.
- **A link is the full path from the spec folder, also to a Definition in the same Package:** `[[members/member]]`, never `[[member]]`. Library Kinds start with the Library: `std/Rule`. A Package is `[[members]]`.

## Fixing the common errors

| Code | Usual cause and fix |
|---|---|
| `link.unresolved` | Short or relative link. Use the full path from the spec folder; check spelling and that the target exists. |
| `link.syntax` | `../x`, a leading `/`, or a character other than letters, digits, `-`, `_`. Use a plain path. |
| `link.internal` | The target is `internal` to another Package. Drop the link and describe the need in your own words, or make the target public. |
| `link.downward` | A parent links into a child's Definition. Link the Package instead, move the Definition up, or let the child link to a need stated here. |
| `package.cycle` | Two Packages link to each other (the message shows the path). Move the shared concept up, or remove one side's link as described above. |
| `kind.unresolved`, `kind.not_a_kind` | The Kind path is wrong (`std/Feature`, a project Kind by its path) or names a Definition that is not of Kind `Kind`. |
| `definition.front_matter`, `definition.title` | An extra front matter key, or no single level-1 heading as the first body line. |
| `lock.mismatch`, `lock.missing` | Run `markitect tidy` after a manifest or Library change and put `lock.yaml` in the diff. |
| `file.unregistered` and other `registry.*` | A file is on no registry page and not in the ignore file, or a page is wrong. See the registry section of [reference.md](reference.md). |

Every other code is listed in [reference.md](reference.md). A cycle or unresolved link usually hides other errors: fix and run `check` again.

## Red flags

| Thought | Instead |
|---|---|
| "I'll choose the Packages myself." | Propose them and stop for the owner. |
| "The requirements don't say, I'll assume the usual." | Stop and ask the owner before writing it. |
| "The code does it, so it must be intended." | Ask the owner whether it is. |
| "I'll describe how it works inside." | Leave algorithms and data structures to the agents. |
| "This rule needs a few exceptions." | Split it into several Rules. |
| "Package A just needs one link to B's Definition." | Check `list --deps`: does it close a cycle, or add a ninth dependency? |
| "The vet warning is cosmetic." | Fix it, or write one line why it stays. |
| "It compiled earlier." | Run `check` again on the final tree. |

## Hand over

Report to the owner in a few lines: the diff; the output of `check` (its summary line) and of `vet` (warnings fixed or justified); the list `status` prints, so the owner sees what the change makes the agents do; and any point the owner chose to leave open. Leave the work as a diff; commit only if the owner asks. Go through it with the owner when they would rather talk it through than read it. Change it until they accept it.

Deleting or renaming a Definition leaves registry entries that name it. That is stale realization, an error of the realization rather than the spec's: `check` lists it apart, and the next build cleans it up. Say so; do not edit the registry pages in a change of the spec, which the builder keeps (only drawing the spec from the code writes them).

Building is the owner's step: `/markitect-build` has a builder agent build the spec in a worktree of its own. `/markitect-audit` has a fresh auditor read the spec for contradictions, gaps and unclear statements before it is built. Do not start either yourself unless the owner asks.

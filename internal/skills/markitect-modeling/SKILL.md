---
name: markitect-modeling
description: Use when asked to create, extend, change or review a Markitect model (the Markdown Definitions under .markitect/model/), to turn requirements or a design discussion into a model, to place a Definition in the right Package, to fix a markitect check error, or to judge a model's quality. Not for drawing a model from existing code; use markitect-extract for that, which builds on this skill.
---

# Writing a Markitect model

A Markitect model is typed documentation: Markdown files that state what the project is and must do, the way good documentation would. Each file is a Definition of some Kind; folders are Packages; a link names its target by path. `markitect check` compiles the structure. Agents realize the model and never change it.

Read [reference.md](reference.md) for the form, the Kinds and the error codes, and [examples.md](examples.md) for a model that compiles. The full language guide is `docs/guides/model-language.md` in the Markitect repository.

## Who decides

The owner accepts every model change. You propose; the owner reads the Markdown diff and merges it. A Run never changes the model.

- Write only what the owner or the requirements state. Do not invent a policy, a limit, an error case or a default. If something is open, ask the owner before you write that part, and write it only after the answer.
- The cut into Packages and the directions of dependencies are the owner's decision. If they are not decided, propose them (a table: Package, what it holds, what it may depend on) and stop for the answer before you write.
- Two requirements that contradict each other, or one you cannot model without guessing, go back to the owner as a question.

## Workflow

```text
[ ] 1 Read the requirements or the discussion; ask the owner about every open point
[ ] 2 Packages and dependency directions agreed with the owner
[ ] 3 Each statement mapped to a planned Definition (none left over, none invented)
[ ] 4 Definitions written; markitect check passes
[ ] 5 markitect lint, refs, graph, impact read; warnings fixed or justified
[ ] 6 Diff handed to the owner
```

Use the tools while you work, not only at the end:

- `markitect check` after every few files. It prints every error as `file:line: code: message`; fix the first ones and run it again.
- `markitect refs TARGET` before you rename, move or delete a Definition: it shows who links to it and which files realize it.
- `markitect graph` when you add a link between Packages: it shows the new edge and the links behind it.
- `markitect lint` at the end: coupling, size, unused Definitions, reach. Fix each warning or give one line why it stays.
- `markitect impact --base COMMIT` on your change: it shows the Definitions, files and Managers the change reaches. Give the owner that list; it is what a Run will work on.
- After a manifest or Library change run `markitect lock` and include the new `lock.yaml` in the diff.

## Where to put a Definition

Knowledge sits broad at the top and deep below. A Manager reads its own Package, everything its parents define and the public Definitions it links to.

- **Root** (the model folder itself): the project anchor (`std/Project`), project-wide Rules, layout conventions (where code and tests live), the architecture, shared Kinds.
- **One Package per real area** of the system, with a single responsibility and its own code folder. It holds that area's Concepts, Features and interfaces (`rest/Endpoint`, `cli/Command`) and the Rules only it follows. Do not make a Package of one Definition; a Package with no Definitions and one child is a lint warning.
- **Shared things go to the lowest common parent** of the Packages that use them. A Kind used by several Packages sits at Root, because using a Kind is a dependency.
- **Dependencies point up or sideways, never down, and never in a circle.** A link to an ancestor's Definition is free. A link to another Package's Definition is a dependency; keep them few. A parent may name a child Package (`[[members]]`), not a Definition inside it.
- **Mark a Definition `internal: true`** when only its own Package should link to it, such as a detail another Package must not build on.
- A cycle is broken by moving the shared concept to the common parent, or by inverting the dependency: the side that needs something describes the need in its own Package, and the other side links to that.

## What to state and what to leave

State what good documentation of the project would state (Markitect's rule DEC-079): domain concepts (what a book is, which states it has), features and use cases, the interfaces callers use (endpoints, commands, flags, exit codes), rules, decisions and the architecture. Leave to the agents what good documentation would not name: loops, maps, data structures, algorithms, class and function names, internal steps.

- Write exact values in backticks: wire names, error codes, limits, statuses. A caller must be able to rely on them.
- A statement is true everywhere it applies. A Rule full of "except when" is several Rules.
- Say each thing once and link to it from everywhere else.
- Keep a Definition short: one idea, a few hundred words at most (the lint warns above 1000).
- Choose the Kind whose body fits (`reference.md`). When none fits, write a small project Kind, one file, that says what such a Definition contains.

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
- **A link is the full path from the model folder, also to a Definition in the same Package:** `[[members/member]]`, never `[[member]]`. Library Kinds start with the Library: `std/Rule`. A Package is `[[members]]`.

## Fixing the common errors

| Code | Usual cause and fix |
|---|---|
| `link.unresolved` | Short or relative link. Use the full path from the model folder; check spelling and that the target exists. |
| `link.syntax` | `../x`, a leading `/`, or a character other than letters, digits, `-`, `_`. Use a plain path. |
| `link.internal` | The target is `internal` to another Package. Drop the link and describe the need in your own words, or make the target public. |
| `link.downward` | A parent links into a child's Definition. Link the Package instead, move the Definition up, or let the child link to a need stated here. |
| `package.cycle` | Two Packages link to each other (the message shows the path). Move the shared concept up, or remove one side's link as described above. |
| `kind.unresolved`, `kind.not_a_kind` | The Kind path is wrong (`std/Feature`, a project Kind by its path) or names a Definition that is not of Kind `Kind`. |
| `definition.front_matter`, `definition.title` | An extra front matter key, or no single level-1 heading as the first body line. |
| `lock.mismatch`, `lock.missing` | Run `markitect lock` after a manifest or Library change and put `lock.yaml` in the diff. |
| `file.unregistered` and other `registry.*` | A repository file is on no registry page and not on the ignore page, or a page is wrong. See the registry section of `reference.md`. |

Every other code is listed in `reference.md`. A cycle or unresolved link usually hides other errors: fix and run `check` again.

## Red flags

| Thought | Instead |
|---|---|
| "I'll choose the Packages myself." | Propose them and stop for the owner. |
| "The requirements don't say, I'll assume the usual." | Stop and ask the owner before writing it. |
| "I'll describe how it works inside." | Leave algorithms and data structures to the agents. |
| "This rule needs a few exceptions." | Split it into several Rules. |
| "Package A just needs one link to B's Definition." | Check `graph`: does it close a cycle, or add a ninth dependency? |
| "The lint warning is cosmetic." | Fix it, or write one line why it stays. |
| "It compiled earlier." | Run `check` again on the final tree. |

## Done

Report to the owner: the diff; the output of `check` (ok line) and of `lint` (warnings fixed or justified); the result of `impact`, so the owner sees what the change makes the agents do; and any point the owner chose to leave open. Deleting or renaming a Definition leaves registry entries that name it. That is stale realization, not an error: `check` passes, `lint` warns, and the next Run cleans it up. Say so in the report; do not edit the registry pages in a model change (each Package's Manager maintains its page; only an extraction writes them).

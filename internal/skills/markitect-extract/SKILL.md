---
name: markitect-extract
description: Use when asked to adopt Markitect in an existing repository, to "extract" or draw a Markitect model from existing code and documentation, or to propose Packages, Definitions, registry pages and an ignore page for a project that already has code. It builds on the markitect-modeling skill. Not for changing a model that already exists (use markitect-modeling) or for a project that has no code yet.
---

# Drawing a Markitect model from an existing repository

Extract reads a repository and writes the model a good maintainer would document: what the project is, which areas it has, what callers rely on. It also writes the realization record, so every file has one owner. You change only `.markitect/`; the code stays as it is. Read the `markitect-modeling` skill first (form, placement, error codes) and write every Definition as it says.

An extraction is a conversation with the owner, who accepts every model change. Where the code and its documentation leave something open, ask the owner before you write it; you decide nothing they leave open.

## Steps

```text
[ ] 1 markitect init; Libraries chosen; lock written
[ ] 2 Code, documentation, tests and build read
[ ] 3 Binding part decided; Packages chosen from the real areas
[ ] 4 Definitions written
[ ] 5 Registry pages and ignore page written
[ ] 6 markitect check passes; markitect lint read
[ ] 7 Diff handed to the owner
```

### 1. Set up

Work in a Git checkout on its own branch with no other changes. Run `markitect init --name "<Project>"`. It writes the manifest with `std`, an empty `execution.yaml`, a placeholder project anchor at `.markitect/model/project.md`, Root's registry page, the ignore page and the lock, and installs the skills. Add `rest: 1.0.0` or `cli: 1.0.0` to `.markitect/manifest.yaml` when the project has HTTP or command-line interfaces, then run `markitect lock`.

### 2. Read the project

Read the README and documentation, the entry points, the folder structure and what imports what, the tests, the build and CI files. The tests and the documentation show what the authors intend; the code shows what exists. List the areas the code is divided into and how they depend on each other.

### 3. Decide what is binding, and the Packages

**Binding part** (DEC-061): the contract callers rely on. That is the inputs and their validation, the returned and stored data shapes, status and exit codes, stored data and the order in which it was added, and whatever the documentation or a recorded decision states as intended. Write it as good documentation would: domain Concepts, Features, interfaces (`rest/Endpoint`, `cli/Command`), Rules, and Decisions the repository records.

**Free part:** what a caller can observe but cannot reasonably depend on stays out: message wording and detail, generated identifier formats, which failure is reported when several apply, field order, and all internal structure, names, wiring, data structures and step order, unless the artifacts fix them as intended. Do not write loops, maps or algorithms.

**Do not invent intent.** Where code and documentation disagree, where the code looks accidental or wrong, or where a rule seems to hold only by habit, do not choose: ask the owner which one the model should state, and write it after the answer. Record an open question only when the owner chooses to leave the point open. Do not model a bug as intended.

**Packages follow the code's real areas:** a folder or module with one responsibility, a clear boundary and its own tests, usually one level under the root. Prefer few. Do not split what always changes together; do not make a Package for a single file. The project itself is Root: the anchor, the layout and build conventions as Rules, the architecture, shared Kinds, and the files that serve everything (README, CI, build files). Let the Package dependencies follow the real imports, but the model has no cycles: if the code has one, ask the owner which direction the model should state before you write it. If the code allows two reasonable cuts, ask the owner before you write.

### 4. Write the Definitions

Root first, then one Package at a time. Use the standard Kinds; define a small project Kind, at Root, for a kind of thing that recurs and has no Kind. Replace the placeholder `project.md`. Put a Definition at the lowest common parent of the Packages that use it. Links are full paths from the model folder. Run `markitect check` after each Package.

### 5. Registry and ignore pages

Make a list of every file: `git ls-files -co --exclude-standard`. The compiler sees exactly that set.

- **One page per Package** (`.markitect/registry/<package path>.md`), with `paths` that claim its code, tests and documentation (`internal/members/**`, `docs/members*.md`). Root's page (`.markitect/registry.md`) has no patterns and lists what no one claims.
- **List every file in its owner's table** with the Definitions it realizes (link every one it contributes to, tests too) and a short reason. The most specific pattern wins; a file may sit in an owned folder and still be missing from the list, which is an error.
- **A file that serves several Packages** (README, CI, compose, build file) belongs to their lowest common parent, often Root, and links all Definitions it realizes.
- **The ignore page** holds files that belong to no Package: the license, generated or vendored files, lock files, binary assets. Never ignore real source or documentation to make `check` pass; if a file realizes nothing, register it with no link instead, or ask. Do not list files Git ignores.
- Start with `markitect check`: each `file.unregistered` line names a file still to place, so the output is your to-do list.

### 6. Iterate to a passing check

Run `markitect check` until it prints `ok`. Fix the errors in the order given in `markitect-modeling` (links and cycles first, then registry). Then run:

- `markitect lint`: fix or justify each warning. `lint.unused` on a Definition means nothing links it and no file realizes it: link it from the registry or drop it.
- `markitect graph`: the Package arrows should match the real imports and the documentation.
- `markitect refs` on the central Definitions: files and links should agree.

Required checks come from the documented build and test commands. Put them in `.markitect/execution.yaml` only as a proposal, marked as such in your report; the owner adopts them.

### 7. Hand over

Leave the work on its branch as a diff; commit only if the owner asked. Report briefly:

- the Packages you chose and why, and the alternatives you rejected;
- what the model states, and what you left free on purpose;
- the points where code and documentation disagreed, each with the owner's answer, and any point the owner chose to leave open, with the evidence;
- the proposed checks, the ignored files and why, and the lint warnings that remain with one line each;
- the final `markitect check` line.

The owner reads the diff and merges it or sends it back. Corrections to the model go through the `markitect-modeling` skill.

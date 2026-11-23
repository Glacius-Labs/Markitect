# Project reference

How the tool `markitect` finds a project and its spec, what the project's ignore file leaves out, how a project is set up by hand, and the commands `version` and `diff` ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec); [where Main is](../roadmap.md#where-main-is)). [The project](../concept/the-project.md) in the concept says what it is for; this page says precisely what the tool does. [The language reference](language-reference.md) gives the language and the other commands. The examples are a small lending library.

## The project

A project is a folder that holds the file `.markitect/spec`. Every command but `version` and `licenses` works in one: the folder `-C PROJECT` names, which must hold that file, or else the nearest folder, from the working directory upward, that holds it. `.markitect/spec` must be a regular file; a `.markitect/` folder without it, such as `~/.markitect/` in a home folder, is no project and is passed over. A folder that holds `.markitect/spec` is a project wherever it lies ([DEC-097](../decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)).

Without a project, a command exits 2 and says what it looked for:

```text
markitect check: no folder from C:\work\library upward holds .markitect/spec; that file holds one line, the path of the spec folder below the project folder
```

With `-C PROJECT`, the message names that folder: `PROJECT holds no .markitect/spec; …`. Where `.markitect/spec` exists but is no regular file, such as a folder, the command exits 2 as well and says so: `FOLDER: .markitect/spec is no file, but a project needs one there; …`.

**The file.** `.markitect/spec` holds one line: the spec folder, a path below the project folder, such as `docs/spec`. A byte order mark, white space and the line end around it are dropped, so a copy with CRLF line ends reads the same. An empty file or a second line stops every command that reads it. The spec folder is a relative path with `/` or `\`; it may not be empty, absolute, the project folder itself, outside it or in `.markitect/`. `.markitect` counts in any case, as a file system that ignores case reads it, so a spec folder `.Markitect/docs` is refused too. Nor may it lie outside the project through a symbolic link or a Windows junction: the tool resolves the folder, or the nearest folder above it that exists, and refuses one that lands outside. A spec folder that does not exist, or that is a file, stops every command that reads the spec.

**What the tool reads.** From `.markitect/` the tool reads two files: `spec`, and for `diff` alone `ignore` ([below](#the-ignore-file)). Every command but `diff` reads nothing of the project outside the spec folder; `diff` also reads the project's files ([below](#diff)). It writes no file. It names a document by its path below the spec folder, and a file of the project by its path below the project folder, with `/` on every system.

### The ignore file

`.markitect/ignore`, optional, names what does not belong to the project's own files, such as build output and fetched dependencies, so that `diff` leaves it out ([DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-spec-and-its-files)). A person writes it, as `.markitect/spec`. Only `diff` reads it; `check` and the other commands do not.

```text
# build output and fetched dependencies
node_modules
*.log
build/out/
```

- **Lines.** One pattern per line. Blank lines and lines that start with `#` are skipped; white space and a carriage return at the end of a line and a byte order mark at the start of the file are dropped.
- **Patterns.** A line is a path pattern with the wildcards of `Realized in`, each matching within one name ([language reference](language-reference.md#realized-in)). A pattern without `/`, such as `node_modules` or `*.log`, matches a file or folder of that name at any depth. A pattern with `/` between names, such as `build/out/`, is a path from the project folder. A pattern that ends with `/` matches only folders; a `/` at the end alone does not make it a path, so `node_modules/` matches a folder of that name at any depth. A leading `!` is part of the name, and no line takes back another.
- **Folders.** A folder that a pattern matches is left out with everything below it.
- **Always left out**, with or without the file: every file and folder whose name starts with a dot, so `.markitect/` itself, with everything below such a folder; the spec folder; and symbolic links, Windows junctions and other special files, which the tool never follows.
- **Not valid.** A line that is no valid path pattern, by the rules of `Realized in`, stops `diff` with exit 2, naming the line and the reason: `markitect diff: .markitect/ignore:3: bad pattern "a/../b": a pattern stays below the project folder, without ..`.

The ignore file of NEW counts for both projects that `diff` compares, so that both sides leave out the same.

## Setting up a project by hand

Nothing sets up a project: a person writes `.markitect/spec`, copies the default from Markitect's repository and writes the project's own modules. The default is the module `core` in [`internal/library/core/`](../../internal/library/core/) ([library](library.md)). The tool knows it as the folder `core` directly below the spec folder, so the copy goes there. Nothing rewrites links: a document that names a type of `core` links it by its relative path, such as `../core/rule.md` from a module directly below the spec folder, and `check` names every link that is off.

From an empty folder, with the spec in `docs/spec/`:

1. Write `.markitect/spec` with the one line `docs/spec`.
2. Copy `internal/library/core/` to `docs/spec/core/`.
3. Write the project's own modules, each a folder with a `README.md` ([language reference](language-reference.md#modules)), such as `docs/spec/lending/README.md`, and their documents beside it.
4. Run `markitect check`, and fix what it reports until it prints `0 errors in N documents` and exits 0.

The spec owns its copy: a project may change `core` like any of its modules.

## version

`markitect version` prints the tool's build: its version, with the source revision when the build knows it, `markitect 0.35.0-dev+COMMIT`; a release names the commit it was built from. It needs no project and always exits 0. With `--json`, it prints `version` and `build`.

## diff

`markitect diff OLD [NEW]` compares two states of a project, each a folder that holds `.markitect/spec` ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec), [DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-spec-and-its-files)). NEW is by default the project the command works in. OLD is any folder that holds `.markitect/spec`, however its owner made it, such as a copy of an older state of the project ([DEC-097](../decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)). Nothing is stored between two runs: the older state is only the folder OLD. `diff` reads the specs and the files of both projects, and the ignore file of NEW, and changes nothing. It names three things: the documents of the spec that are new, gone or changed; for each module where something changed, whether its spec, its files or both changed; and the files that no module names.

### The documents

A change to a spec is otherwise seen through the repository's ordinary diff, like every change; `diff` shows it document by document, by the paths below each spec folder. It names, each list sorted by path:

| Section | Lists |
|---|---|
| **New** | The documents of NEW that OLD does not have |
| **Gone** | The documents of OLD that NEW does not have |
| **Changed** | The documents at the same path whose text differs, line ends and a byte order mark apart, with a unified diff of the old and the new text, three lines of context, its headers the document's file below each project folder. Where both texts differ in so many lines that the differing middle holds more than four million pairs of lines, the diff is one hunk that removes all of that middle and adds the new one |

A document that moved is one gone and one new. Each section shows `(none)` when empty.

### The modules

For every module of either project where something changed, `diff` says whether its spec changed, its files changed, or both, and names its files that are new, gone or changed. A module where nothing changed is not named.

- **Its spec** changed when a document directly in its folder is new, gone or changed, as in the part on documents; a document of a module below counts for that module. A README whose `Realized in` changed is such a document, so it shows as a change of the module's spec, not of its files.
- **A changed file** is one of the project's own files, those not [left out](#the-ignore-file), that is new, gone or changed by itself: new when only NEW has it, gone when only OLD has it, and changed when both have it and it differs byte for byte, line ends included. A file is never new, gone or changed because a README changed.
- **The modules of a file.** A new or changed file goes to the modules that name it under `Realized in` in NEW ([language reference](language-reference.md#realized-in)), and a gone file to the modules that named it in OLD. When a module and a module below it, within it, both name a file in that state, it goes only to the module below; modules of which neither lies within the other all count.
- **Shown, not judged.** A module whose files changed without its spec is named like any other. It may be drift, or a rebuild that leaves what should hold untouched, and the tool cannot know which.

The part is headed `## Modules (N)`, with N the number of modules named, sorted by path with the spec folder first. Each module stands on a line of its own, `lending: spec changed`, `lending: files changed` or `lending: spec and files changed`, with its files below it, two spaces in, one line each in path order as `changed`, `gone` or `new`, the path padded to one column: `  changed src/lending/borrow.go`. The spec folder, when it is a module, is named `(spec folder)`.

### Files no module names

`diff` names the project's own files that no module names under `Realized in` and that are new, gone or changed, and gives the number of files of NEW that no module names in total. Whether a module names a file is asked in the state the file belongs to: NEW for a new or changed file, OLD for a gone one. A project whose READMEs have no `Realized in` has all its files there. The documents of the spec are never among the files: the spec folder is always left out of them and compared only in the part on documents.

The part is headed `## Files no module names (N)`, with one line per file, `new     scripts/seed.sql`, sorted by path, and ends with the line `Files of NEW that no module names: N`. Like every part, it shows `(none)` when empty, and every part is printed, also when nothing differs.

### Example

A small lending library, with `docs/spec` in `.markitect/spec` and one line in `.markitect/ignore`, `build/out/`. The README of `lending` names `` `src/lending/` `` under `Realized in`, and the README of `lending/fines` names `` `src/lending/fines.go` ``; both READMEs are the same in both states. Between OLD and NEW, the project changed so:

```text
.markitect/spec                      same
.markitect/ignore                    same
README.md                            same
build/out/library.exe                changed
docs/spec/core/...                   same
docs/spec/lending/README.md          same
docs/spec/lending/borrow.md          same
docs/spec/lending/loan-limit.md      changed
docs/spec/lending/renew-a-loan.md    new
docs/spec/lending/fines/README.md    same
docs/spec/lending/fines/late.md      same
scripts/seed.sql                     new
src/lending/borrow.go                changed
src/lending/fines.go                 changed
src/lending/renew.go                 new
```

`markitect diff OLD`, run in NEW, prints:

```text
# Difference

## New (1)

lending/renew-a-loan.md

## Gone (0)

(none)

## Changed (1)

lending/loan-limit.md
--- docs/spec/lending/loan-limit.md
+++ docs/spec/lending/loan-limit.md
@@ -8,4 +8,4 @@
 ## Rule
 A member has at most five open loans.
 Staff are not limited.
-A loan is open until the book is back.
+A loan is open until the book is returned.

## Modules (2)

lending: spec and files changed
  changed src/lending/borrow.go
  new     src/lending/renew.go

lending/fines: files changed
  changed src/lending/fines.go

## Files no module names (1)

new     scripts/seed.sql

Files of NEW that no module names: 2
```

`lending` names `src/lending/fines.go` too, but `lending/fines` lies within it and names the file as well, so it goes only to `lending/fines`. The two files of NEW that no module names are `README.md` and `scripts/seed.sql`. `core` is not named, since nothing of it changed. `build/out/library.exe` is left out by the ignore file, and `.markitect/` because its name starts with a dot. `diff` exits 1.

### Exit codes and JSON

`diff` exits 0 when nothing differs, and 1 when anything differs, documents or files; 1 says only that something differs, and no module is judged by it. It exits 2 when OLD or NEW is no folder or holds no `.markitect/spec`, a spec cannot be read, the ignore file of NEW has a line that is no valid pattern, or a file cannot be read. A file is opened only when both projects have it at the same size, and one that cannot be read then stops `diff`: `cannot read PATH in the old project: REASON`, or `in the new project`.

With `--json`, its `kind` is `diff`, and it prints `new`, `gone` and `changed`, each changed document with `doc` and `diff`; then `modules`, objects with `module`, `spec` and `files`, both true or false, and `new_files`, `gone_files` and `changed_files`, lists that are empty when nothing is in them; and `unassigned`, with `files`, objects with `file` and `change` (`new`, `gone` or `changed`), and `total`, the number of files of NEW that no module names. The spec folder as a module is `.` in JSON.

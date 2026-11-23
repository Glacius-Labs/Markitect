# Project reference

How the tool `markitect` finds a project and its spec, how a project is set up by hand, and the commands `version` and `diff` ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec); [where Main is](../roadmap.md#where-main-is)). [The project](../concept/the-project.md) in the concept says what it is for; this page says precisely what the tool does. [The language reference](language-reference.md) gives the language and the other commands. The examples are a small lending library.

## The project

A project is a folder that holds the file `.markitect/spec`. Every command but `version` and `licenses` works in one: the folder `-C PROJECT` names, which must hold that file, or else the nearest folder, from the working directory upward, that holds it. `.markitect/spec` must be a regular file; a `.markitect/` folder without it, such as `~/.markitect/` in a home folder, is no project and is passed over. A folder that holds `.markitect/spec` is a project wherever it lies ([DEC-097](../decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)).

Without a project, a command exits 2 and says what it looked for:

```text
markitect check: no folder from C:\work\library upward holds .markitect/spec; that file holds one line, the path of the spec folder below the project folder
```

With `-C PROJECT`, the message names that folder: `PROJECT holds no .markitect/spec; …`. Where `.markitect/spec` exists but is no regular file, such as a folder, the command exits 2 as well and says so: `FOLDER: .markitect/spec is no file, but a project needs one there; …`.

**The file.** `.markitect/spec` holds one line: the spec folder, a path below the project folder, such as `docs/spec`. A byte order mark, white space and the line end around it are dropped, so a copy with CRLF line ends reads the same. An empty file or a second line stops every command that reads it. The spec folder is a relative path with `/` or `\`; it may not be empty, absolute, the project folder itself, outside it or in `.markitect/`. `.markitect` counts in any case, as a file system that ignores case reads it, so a spec folder `.Markitect/docs` is refused too. Nor may it lie outside the project through a symbolic link or a Windows junction: the tool resolves the folder, or the nearest folder above it that exists, and refuses one that lands outside. A spec folder that does not exist, or that is a file, stops every command that reads the spec.

The tool reads nothing else from `.markitect/` and nothing of the project outside the spec folder. It writes no file. It names a document by its path below the spec folder, with `/` on every system.

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

`markitect diff OLD [NEW]` compares the specs of two projects, each a folder that holds `.markitect/spec` ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). NEW is by default the project the command works in. OLD is any folder that holds `.markitect/spec`, however its owner made it, such as a copy of an older state of the project ([DEC-097](../decisions.md#dec-097-markitect-neither-needs-nor-knows-version-control)). A change to a spec is otherwise seen through the repository's ordinary diff, like every change; `diff` shows it document by document, by the paths below each spec folder. It reads both specs and changes nothing. It names, each list sorted by path:

| Section | Lists |
|---|---|
| **New** | The documents of NEW that OLD does not have |
| **Gone** | The documents of OLD that NEW does not have |
| **Changed** | The documents at the same path whose text differs, line ends and a byte order mark apart, with a unified diff of the old and the new text, three lines of context, its headers the document's file below each project folder. Where both texts differ in so many lines that the differing middle holds more than four million pairs of lines, the diff is one hunk that removes all of that middle and adds the new one |

A document that moved is one gone and one new. Each section shows `(none)` when empty.

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
```

`diff` exits 0 when nothing differs, 1 when anything does, and 2 when OLD or NEW is no folder or holds no `.markitect/spec`, or a spec cannot be read. With `--json`, its `kind` is `diff`, and it prints `new`, `gone` and `changed`, each changed document with `doc` and `diff`.

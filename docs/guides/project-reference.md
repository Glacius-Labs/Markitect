# Project reference

How the rebuilt tool `markitect-next` finds and sets up a project, what lies in `.markitect/`, which files of the project it sees and how it hashes them ([DEC-133](../decisions.md#dec-133-the-project-holds-only-the-spec-and-markitect-and-pins-its-version), [DEC-097](../decisions.md#dec-097-markitect-uses-no-git)). [The project](../concept/jobs-and-runners.md#the-project) and [the state](../concept/the-state.md) in the concept say what they are for; this page says precisely what the tool does. [The language reference](language-reference.md) gives the commands that read and write the spec. The examples are a small lending library.

## The project

A project is a folder that holds `.markitect/version`. Every command but `init` and `version` works in one: the folder `-C PROJECT` names, which must hold `.markitect/version`, or else the nearest folder, from the working directory upward, that holds it. A `.markitect/` folder without a version file, such as `~/.markitect/` with a reader's last look, is no project and is passed over. Without a project, a command exits 2 and says that `init` makes one. The tool uses no Git to find it, and a folder that holds `.markitect/version` is a project wherever it lies.

The tool reads a project through a file system rooted at its folder, the spec and `.markitect/` included, so that two projects, such as two states to compare, load the same way. It names a document by its path below the spec folder and any other file by its path below the project folder, with `/` on every system.

## The .markitect/ folder

| File | Holds |
|---|---|
| `version` | One line: the tool version the project pins, such as `0.35.0`. It is readable without the tool |
| `spec` | One line: the spec folder, a path below the project folder, by default `docs/spec` |
| `.gitattributes` | `state.jsonl merge=union` and `history.jsonl merge=union`, so that Git keeps the lines of both sides of a merge |
| `ignore` | Files the project does not explain, written like a `.gitignore` ([the walk](#the-walk)); empty at `init` |

`version` and `spec` are read as one line each: a byte order mark, white space and the line end around it are dropped, so a checkout with CRLF reads the same. An empty file, a second line or a missing file stops every command that reads it. The spec folder is a relative path with `/` or `\`; it may not be empty, absolute, the project folder itself, outside it, in `.markitect/` or in a `.git` folder. A spec folder that does not exist stops every command that reads the spec; `init` and `apply` create it.

## init

`markitect-next init [--spec PATH]` sets up the folder `-C` names, or the working directory, as a project:

- it writes `.markitect/` with `version`, the tool's own version, `spec`, by default `docs/spec`, `.gitattributes` and an empty `ignore`;
- it creates the spec folder when it is missing, and keeps one that exists with every file in it.

It writes nothing else: no `AGENTS.md`, no `CLAUDE.md`, nothing in `.claude/` and no line in `.gitignore`. It refuses, and changes nothing, when `.markitect/` exists or when the spec folder is a file. It builds `.markitect/` under a temporary name and renames it into place, and undoes every step when one fails, so a failed `init` leaves nothing that stops the next. It prints `created .markitect/` and, when it made the spec folder, `created docs/spec/`. With `--json`, it prints `version`, `spec` and `created`, the folders it made.

## version

`markitect-next version` prints the tool's version, `markitect-next 0.35.0`. It needs no project. In the project `-C` names, or the nearest one upward, it adds the version the project pins, `the project pins 0.35.0`, or `the project pins 0.34.0, not this version` and exits 1 when the two differ. Outside a project it prints the tool's version alone and exits 0. With `--json`, it prints `version`, `pinned`, empty outside a project, and `matches`.

## The walk

The walk lists every file of the project, as Git would see it in a commit, by its path below the project folder, sorted. It reads the folder itself and uses no Git. It leaves out:

- every entry named `.git`, file or folder, at any depth, and `.markitect/` at the top;
- what the project's `.gitignore` files exclude, each read in its own folder, as Git reads them;
- what `.markitect/ignore` excludes, read like a `.gitignore` file at the top of the project.

The `.gitignore` files and `.markitect/ignore` are judged apart, and a file either excludes is left out. So a `!` line in one brings back nothing the other excludes: `!notes.md` in `.markitect/ignore` does not list a `notes.md` that `.gitignore` excludes.

**The patterns.** They follow [Git's gitignore rules](https://git-scm.com/docs/gitignore), case included on every system:

| Pattern | Means |
|---|---|
| A blank line, or one starting with `#` | Nothing; `\#` starts a pattern with `#` |
| Trailing spaces | Dropped, unless a backslash keeps one: `name\ ` |
| `!pattern` | Brings back what an earlier pattern excluded, unless a folder above it is excluded; `\!` starts a pattern with `!` |
| `name/` | Only a folder |
| `name` | A file or folder of that name at any depth below the ignore file's folder |
| `/name`, `docs/name` | With a `/` at the start or in the middle: the path below the ignore file's folder |
| `*`, `?`, `[a-z]`, `[!a]` | Any characters but `/`, one character but `/`, one of a range, one not in it |
| `**/name`, `a/**/b`, `name/**` | Any number of folders, none included; `name/**` holds what is inside `name`, not `name` itself |
| `\x` | The character `x` itself |

The last pattern that matches decides, and a `.gitignore` file in a deeper folder wins over one above it. A byte order mark at the start and CR at line ends are dropped. A missing `.gitignore` is none, and so is a folder or a symbolic link of that name, which Git does not follow; any other file that cannot be read stops the walk.

**Other entries.** A folder that is another repository or worktree, whose `.git` is a folder with a `HEAD` or a file starting with `gitdir:`, is one entry and is not walked, as Git keeps it; it reads as the fixed line `a repository of its own`. A symbolic link is one entry, read as the path it points to, as Git stores it, and is not followed; a pattern for folders does not match it.

**The limits.** The walk reads only the `.gitignore` files inside the project folder: in a project below the top of a repository, the files above it do not count, though Git applies them. Global Git settings, `.git/info/exclude` and character classes such as `[[:digit:]]` are not read.

## Hashing

A file's hash is its SHA-256, written `sha256:` and 64 hexadecimal digits. A text file, one that is valid UTF-8 and holds no NUL byte, is hashed with each CRLF as LF, so a checkout on Windows and one on Linux of the same commit agree; a lone CR stays. Any other file is hashed as it is, so the CRLF in an image is kept. A file that is not UTF-8, such as Latin-1 text, counts as binary: its checkouts with LF and with CRLF hash apart.

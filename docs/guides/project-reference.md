# Project reference

How the rebuilt tool `markitect-next` finds and sets up a project, what lies in `.markitect/`, which files of the project it sees, how it hashes them, and the state with `status` and `mark` ([DEC-133](../decisions.md#dec-133-the-project-holds-only-the-spec-and-markitect-and-pins-its-version), [DEC-097](../decisions.md#dec-097-markitect-uses-no-git)). [The project](../concept/jobs-and-runners.md#the-project) and [the state](../concept/the-state.md) in the concept say what they are for; this page says precisely what the tool does. [The language reference](language-reference.md) gives the commands that read and write the spec. The examples are a small lending library.

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
| `state.jsonl` | The state: one row per document with its verdict and the hashes it rests on ([below](#the-state)); written by `mark` |

`version` and `spec` are read as one line each: a byte order mark, white space and the line end around it are dropped, so a checkout with CRLF reads the same. An empty file, a second line or a missing file stops every command that reads it. The spec folder is a relative path with `/` or `\`; it may not be empty, absolute, the project folder itself, outside it, in `.markitect/` or in a `.git` folder. `.markitect` and `.git` count in any case, as a file system that ignores case reads them, so `.Markitect/spec` is refused too. Nor may it lie outside the project through a symbolic link or a Windows junction: `init` and every command that reads `.markitect/spec` resolve the folder, or the nearest folder above it that exists, and refuse one that lands outside. A spec folder that does not exist stops every command that reads the spec; `init` and `apply` create it.

## init

`markitect-next init [--spec PATH]` sets up the folder `-C` names, or the working directory, as a project:

- it writes `.markitect/` with `version`, the tool's own version, `spec`, by default `docs/spec`, `.gitattributes` and an empty `ignore`;
- it creates the spec folder when it is missing, and keeps one that exists with every file in it.

It writes nothing else: no `AGENTS.md`, no `CLAUDE.md`, nothing in `.claude/` and no line in `.gitignore`. It refuses, and changes nothing, when `.markitect/` exists or when the spec folder is a file. It builds `.markitect/` under a temporary name, with mode 0755 like the spec folder, and renames it into place, and undoes every step when one fails, so a failed `init` leaves nothing that stops the next. It prints `created .markitect/` and, when it made the spec folder, `created docs/spec/`. With `--json`, it prints `version`, `spec` and `created`, the folders it made.

## version

`markitect-next version` prints the tool's version, `markitect-next 0.35.0`. It needs no project. In the project `-C` names, or the nearest one upward, it adds the version the project pins, `the project pins 0.35.0`, or `the project pins 0.34.0, not this version` and exits 1 when the two differ. Outside a project it prints the tool's version alone and exits 0. With `--json`, it prints `version`, `pinned`, empty outside a project, and `matches`.

## The walk

The walk lists every file of the project by its path below the project folder, sorted. It reads the folder itself and uses no Git. It leaves out:

- every entry named `.git`, file or folder, at any depth and in any case, and `.markitect/` at the top, spelled exactly so;
- what the project's `.gitignore` files exclude, each read in its own folder, as Git reads them;
- what `.markitect/ignore` excludes, read like a `.gitignore` file at the top of the project.

The `.gitignore` files and `.markitect/ignore` are judged apart, and a file either excludes is left out. So a `!` line in one brings back nothing the other excludes: `!notes.md` in `.markitect/ignore` does not list a `notes.md` that `.gitignore` excludes.

**The patterns.** They follow [Git's gitignore rules](https://git-scm.com/docs/gitignore) and are matched as Git's wildmatch matches them, byte by byte:

| Pattern | Means |
|---|---|
| A blank line, or one starting with `#` | Nothing; `\#` starts a pattern with `#` |
| Trailing spaces | Dropped, unless a backslash keeps one: `name\ ` |
| `!pattern` | Brings back what an earlier pattern excluded, unless a folder above it is excluded; `\!` starts a pattern with `!` |
| `name/` | Only a folder |
| `name` | A file or folder of that name at any depth below the ignore file's folder |
| `/name`, `docs/name` | With a `/` at the start or in the middle: the path below the ignore file's folder |
| `*`, `?` | Any bytes but `/`, one byte but `/`; so `?` does not match `é`, which is two bytes in UTF-8 |
| `[a-z]`, `[!a]`, `[^a]`, `[]a]` | One of a range, one not in the set, the same with `^`, and `]` as the first member of a set |
| `[[:digit:]]` | One character of a POSIX class: `alnum`, `alpha`, `blank`, `cntrl`, `digit`, `graph`, `lower`, `print`, `punct`, `space`, `upper` or `xdigit`, over ASCII. An unknown class or an unclosed `[` matches nothing |
| `**/name`, `a/**/b`, `name/**` | Any number of folders, none included; `name/**` holds what is inside `name`, not `name` itself. More stars, `***`, count as two |
| `\x` | The character `x` itself; `name\\ ` ends in a backslash, and its trailing space is dropped |

The last pattern that matches decides, and a `.gitignore` file in a deeper folder wins over one above it. A byte order mark at the start and CR at line ends are dropped. A missing `.gitignore` is none, and so is a folder or a symbolic link of that name, which Git does not follow; any other file that cannot be read stops the walk.

**Case.** Patterns match a name's case exactly, on every system, so that a Windows and a Linux checkout list the same files. Git on Windows and macOS ignores case by default (`core.ignorecase`), so a pattern there may match a name in another case, which the walk does not: write each pattern in the case of the name it means.

**Other entries.**

- **Another repository.** A folder that is a valid repository or worktree of its own, as Git decides, is one entry and is not walked; it reads as the fixed line `a repository of its own`. Valid means that its `.git` is a folder with `objects/`, `refs/` and a `HEAD` that holds `ref: refs/…` or a full object hash, or with such a `HEAD` and a `commondir` file naming a folder with `objects/` and `refs/`, as a worktree's has; or that its `.git` is a file whose `gitdir:` line names such a folder, relative to the folder or absolute. Any other `.git` is left out and the folder is walked, so a fake repository never hides files.
- **A symbolic link** is one entry, read as the path it points to, as Git stores it, and is not followed; a pattern for folders does not match it.
- **A Windows junction** is walked like the folder it leads to, and its files are listed below the junction's path. A junction that leads back to a folder the walk is already in is dropped entirely, and so is one that leads nowhere.
- **Other kinds of file,** such as FIFOs, sockets and devices, are left out, as Git leaves them out, so that reading the project never blocks. The walk lists only regular files, symbolic links, folders and junctions to folders.

**The limits.** Without Git, the walk cannot know what is committed: a file that is committed although a `.gitignore` excludes it is not walked, while Git keeps tracking it. The walk reads only the `.gitignore` files inside the project folder: in a project below the top of a repository, the files above it do not count, though Git applies them. Global Git settings and `.git/info/exclude` are not read.

## Hashing

A file's hash is its SHA-256, written `sha256:` and 64 hexadecimal digits. A file is text, as Git decides, unless a NUL byte stands in its first 8000 bytes. A text file, in any encoding, Latin-1 included, is hashed with each CRLF as LF, so a checkout on Windows and one on Linux of the same commit agree; a lone CR stays. Any other file is hashed as it is, so the CRLF in an image is kept.

## The state

`.markitect/state.jsonl` maps each document to its artifacts and verdict ([DEC-124](../decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)). It holds one JSON object per line, one line per document, sorted by `doc`, with the fields in this order:

```json
{"doc":"lending/borrow.md","verdict":"ok","source":"mark","inputs":{"check:tests/loans_test.go":"sha256:…","evidence:tests/loans_test.go":"sha256:…","file:src/loans.go":"sha256:…","link:roles/member.md":"sha256:…","self":"sha256:…","type:types/operation.md":"sha256:…"},"evidence":["tests/loans_test.go:3"],"reason":""}
```

| Field | Holds |
|---|---|
| `doc` | The document, by its path below the spec folder |
| `verdict` | `ok`, `fail` or `skip` |
| `source` | What wrote the row: `mark` |
| `inputs` | The hash of each input at the confirmation, by key, sorted |
| `evidence` | The evidence as the reviewer gave it: `path`, `path:line`, `path:line-line` or `path:line:col` |
| `reason` | Why, for a `fail` or a `skip`; one line |

| Input | Is |
|---|---|
| `self` | The document's own text |
| `type:PATH` | Its type, by its path below the spec folder |
| `link:PATH` | Each document it links to and each document that links to it, not itself; the type line is no link |
| `asset:PATH` | Each file of the spec folder that is no document and that it links or shows, such as a diagram, by its path below the spec folder |
| `file:PATH` | An artifact the reviewer named, by its path below the project folder |
| `check:PATH` | A counter-check the reviewer named |
| `evidence:PATH` | A file of the evidence |

Every hash is a file's hash as [hashing](#hashing) computes it, so a row written on one system holds on the other.

**Reading it.** Lines may stand in any order and twice, as a union merge of two branches leaves them, and a blank line, a byte order mark and CR at line ends count for nothing. A field the tool does not know is ignored. A line that is not a row is reported as `.markitect/state.jsonl:LINE: state.unreadable: MESSAGE`: one that is no JSON object, whose `doc` is no clean path below the spec folder, whose verdict or source is unknown, that has no `self`, or that has an input of an unknown kind, a path that is not clean or a value that is no `sha256:` hash. `status` and `mark` refuse while one is there; `show` and `table` leave it out.

**Which row counts.** Of several rows for one document, the tool takes the row whose inputs all match the project as it is: each input has its current hash, and the document has no `type:` or `link:` input the row lacks. Two such rows with different verdicts are a conflict. When none matches, it takes the closest, the row with the most inputs that match, to say what changed. A file a row names that the walk does not list, deleted or now ignored, matches nothing.

## status

`markitect-next status` is the plan: what moved since the last confirmation, from the folder and the state alone. It changes nothing. While the spec has errors of [`check`](language-reference.md#what-check-reports), or the state a line it cannot read, it prints them and `no list while the spec or the state has errors`, gives no list and exits 1. Otherwise it lists:

| Entry | When |
|---|---|
| **changed** | The document has no row; its own text changed; a file of the spec folder it links changed, is new or is gone; an artifact, a counter-check or a file of its evidence changed or is gone; its verdict is `fail`, which stays on the list with its reason; rows of it conflict; or it is gone from the spec and still has a row |
| **suspect** | Only its type or the documents it links to or that link to it changed, or the set of those documents changed |
| **unexplained** | A file of the [walk](#the-walk) that is no document, that no `file:` or `check:` input names in the row taken for a document of the spec, and that no document links. Only `.md` files are documents: a script in the spec folder is unexplained until a document links it or it is ignored. Evidence alone maps no file, and the files of a gone document's row are unexplained |

```text
changed: lending/borrow.md@3f9a1c07: self
  look at: docs/spec/lending/borrow.md, src/loans.go, tests/loans_test.go, tests/loans_test.go:3
suspect: roles/member.md@81be07d2: link:lending/borrow.md
  look at: docs/spec/roles/member.md, src/member.go, src/member.go:3
unexplained: src/fines.go
note: lending/return.md: skip: no returns at the kiosk
1 changed, 1 suspect, 1 unexplained
```

Each entry reads `DOCUMENT@FINGERPRINT: CHANGES`, in path order. A change is `no row`, `gone`, `self`, an input such as `file:src/loans.go`, with ` (gone)` for a file that is gone, ` (new)` for a type, a link or a linked file of the spec folder the row lacks and ` (dropped)` for one the document no longer has, `fail: REASON`, or `conflict: fail, ok`. `look at:` names the document's file, below the project folder, the artifacts and counter-checks of the row taken, and its evidence. A `skip` at the document's current state is a note, not an entry. `status` exits 0 when it lists nothing and 1 otherwise; notes never change that. With `--json` it prints `errors`, with `file`, `line`, `code` and `message`, `changed` and `suspect`, each with `doc`, `fingerprint`, `changes` and `look_at`, `unexplained`, and `notes`, with `doc`, `verdict` and `reason`.

**The fingerprint** of every document, changed, suspect or unlisted, is the first 8 hexadecimal digits of the SHA-256 of what a verdict on it will rest on, without the evidence, which the reviewer chooses after reading: its `self`, `type:`, `link:` and `asset:` inputs, the current hashes of the artifacts and counter-checks of the row taken, and the digest of every file of the walk but the documents, by path and hash, the files of the spec folder that are no documents included. A mark changes only `.markitect/`, so marks made from one `status` never invalidate one another. Any edit of a project file after `status`, by a builder working beside the reviewer or a test run writing a file, changes every fingerprint, and the files named with `--file` or `--check` are covered in every case. The cost is that `status` hashes every file of the walk. `show` prints the fingerprint of any document, listed or not, so that a confirmed document can be marked again: an `ok` turned into a `fail`, or an unexplained file mapped to it.

## mark

`markitect-next mark DOCUMENT@FINGERPRINT VERDICT [--file PATH]... [--check PATH]... [-e EVIDENCE]... [-m REASON]` binds a verdict to the state `status` showed:

```text
markitect-next mark lending/borrow.md@3f9a1c07 ok --file src/loans.go --check tests/loans_test.go -e "tests/loans_test.go:3 (TestBorrow)"
markitect-next mark lending/return.md@5d0e9b11 fail --file src/returns.go -m "a book comes back after four weeks"
```

- **The verdict.** `ok` needs at least one piece of evidence; `fail` and `skip` need a reason with `-m`, kept on one line. A document gone from the spec takes only `ok`, without artifacts or evidence, which removes its row.
- **The paths** are below the project folder, with `/` or `\`. `--file` names an artifact, `--check` a counter-check; each must be a file of the walk outside the spec folder, and one file cannot be both; a file of the spec folder counts through the documents that link it. The named artifacts and counter-checks replace those of the earlier row. Evidence is `path`, `path:line`, `path:line-line` or `path:line:col`, and a trailing `(note)`, such as a test name, is dropped; its file must be a file of the walk, and its hash becomes an input.
- **The row** holds the document's `self`, `type:` and `link:` inputs, the named files and the evidence, each at its current hash, with `source` `mark`.

It computes the fingerprint as `status` does and refuses, writes nothing and exits 1 when it differs: a file of the project was created, edited or removed since `status`, such as by a builder working beside the reviewer or by a test run. It also exits 1 when the spec has errors, the state has a line it cannot read, the document is in neither the spec nor the state, or a path is bad, naming every reason. A malformed command, an unknown verdict, an `ok` without evidence or a `fail` or `skip` without a reason exits 2.

It writes under a lock: a file in `markitect/locks/` of the user's cache folder, named after the project folder's real path, so that a symbolic link, a junction, a short name or a substituted drive lead to the same lock. It holds the process's ID and a random token. A lock is stale once its process is gone or it is older than an hour, since a process ID may be reused; it is moved aside in one rename, which only one of several takers can make, and a lock moved aside by mistake is put back. `mark` checks that the lock is still its own before it writes, and removes no lock that is not. The lock lives on one system: Windows and WSL working on one checkout have different cache folders and do not share it. Under it, `mark` reads the folder and the state afresh and replaces the state file in one step, so parallel reviewers lose no row. The file it writes holds one row per document, the row `status` takes, except the rows of a conflict, which stay until a mark of that document settles it; a verdict rewrites only the row of its own document. It prints `marked DOCUMENT VERDICT`, or `removed the row of DOCUMENT, which is gone`; with `--json`, `doc`, `verdict` and `removed`.

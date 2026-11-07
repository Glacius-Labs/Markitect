# Plan: Markitect as a tool, with a verification cache

**Status:** decided by the owner on 7 November 2026 ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)), after a discussion of how they work with Markitect. An independent architect reviewed this page against the code. It is not built yet ([order of work](#order-of-work)).

The owner decided:
- Markitect supports agent work and does not steer it;
- a record of what was verified is kept, modelled on Go's test cache;
- the command names follow Go and widely known tools;
- the work needs no limit on rounds.

An earlier version proposed a command line like Terraform's, in which the tool steered the work through phases. This version replaces it, and reuses its parts on output and exit codes. It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and on DEC-096 and DEC-097 ([lean plan](lean-tool-plan.md)). It replaces their `gate`, their work state and their default driver's steps.

## How the owner works with Markitect

1. **Model.** In an ordinary session, the owner designs the spec with an agent: Definitions, Packages, links and Kinds. The tool checks that the spec is structurally sound and free of conflicts, as a compiler does. It analyses the spec's design the way code is analysed for coupling, and shows dependencies and what a change reaches.
2. **Build.** When the spec is right, agents make the project match it. That is ordinary agent work in sessions and subagents, with prompts. The tool does not drive it.
3. **Make sure nothing is forgotten.** The tool says what has to be verified for the spec to count as implemented. The agents verify each item and record it. When nothing is left to verify, the spec is implemented.

The tool helps at three points: describing, saying what to do, and showing that it is done. It decides nothing about who works when. The skills are prompts that tell the agents to use the tool.

## The idea: a cache of what was verified

`go test` does not run a test again when nothing it depends on has changed; it prints `(cached)`. Markitect does the same for verification. For each Definition it records the state at which an agent last verified it as correctly implemented. That state is the set of everything the verification depended on, its **inputs**:

- the Definition's own text;
- its Kind and every Definition it links to: their text, or for a Library Definition the Library's digest from the lock, so that a newer binary changes nothing;
- the registry rows that name it, and the content of the files they name;
- the files the reviewer gave as evidence, such as a test that lies in another Definition's rows.

Its fingerprint is the hash of these inputs.

**What is to verify** is every Definition with a changed input or with no record. The tool computes that from the folder and the record alone, with no session and no start. These are the same Definitions today's checklist reaches ([concept 4.0 §3](v4-concept.md#how-impact-works)):

- one that changed;
- one that links to a changed Definition, or is of a changed Kind;
- one whose files changed;
- one whose registry row gained or lost it.

In both designs reach stops after one step, and a changed Rule above a Package reaches only itself and what links to it.

Three more kinds of item stay on the list until they are verified again:

- **A Definition recorded as `fail` or `spec-problem`** stays with that note.
- **A Definition that is gone from the spec** stays, marked gone. Its line names the files that realized it, so the item can ask whether they are removed or have a new basis. Reviewing it `ok` removes its line.
- **An error in the realization** stays: an unlisted file, a stale registry row, an unused Definition.

The record holds each input's hash, so `status` can say which inputs changed and where to look. It does not hold what a Definition said before; an agent that wants the old text reads it from Git.

**Recording** binds a verdict to the state the reviewer read. `status` prints each item's fingerprint, and `review DEFINITION VERDICT --at FINGERPRINT` refuses if the inputs have changed since. That catches a builder editing a file beside the reviewer, or the reviewer's own test run changing one. Go does the same: it does not cache a result whose inputs changed while the test ran. The fingerprint `status` prints covers every input but the evidence, which the reviewer chooses later. The evidence must exist in the folder, and its files are hashed when the verdict is recorded.

**Done** is an empty list, whatever happened in between. A file edited by hand, a commit made around the agents, or a change on a merged branch alters the inputs it touches, and those Definitions are on the list again.

## The record

`.markitect/review.sum` is committed with the project, like `go.sum`, and it joins the fixed layout of `.markitect/`. It holds one line per Definition, sorted, each line a JSON object:

```text
{"definition":"fees/waiver","verdict":"ok","inputs":{"self":"sha256:…","link:fees/fee":"sha256:…","file:src/fees.go":"sha256:…","evidence:tests/fees_test.go":"sha256:…"},"evidence":["src/fees.go:40","tests/fees_test.go:12 (TestWaiver)"]}
{"definition":"members/limit","verdict":"fail","inputs":{…},"evidence":["src/loans.go:70"],"why":"allows a sixth loan; the Definition says five"}
```

**Verdicts,** after `go test`:

| Verdict | Means |
|---|---|
| `ok` | The Definition is implemented, as the evidence shows |
| `fail` | It is not; the line says why |
| `skip` | Verified as far as the project can show, with the reason, as `t.Skip` gives one. `status` lists it as a note, and a note alone does not make `status` exit 1 |
| `spec-problem` | The spec contradicts itself or leaves open behaviour at the project's boundary, and only the owner can settle it |

An explanation stays on one line, and hashes are full SHA-256.

**How it is written.** `review` writes under the lock the tool already uses, as one whole file, so parallel reviewers, for example one per Package, do not lose lines.

**How it merges.** Git also conflicts on changes to neighbouring lines. So `init` writes `.markitect/.gitattributes`, which joins the fixed layout, with `merge=union` for this file: Git keeps both sides. Of two lines for one Definition, `status` takes the one whose inputs match the folder. If both match with different verdicts, `status` lists the Definition as a conflict to review. The next `review` writes the file clean.

**What it replaces:** today's certificate, the stored answers, the work under way in the cache, the Result pages, the Findings pages and the builder's reports.

**What it is:** a record, not a proof, as the certificate is today. A line written by hand shows in the diff, and the hook keeps agents from writing one ([below](#the-hook)).

## The commands

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `go mod init` | `init` |
| `markitect check` | Compiles the spec and the registry: Kinds, links, Packages, Libraries against the lock, every file listed or ignored. It produces nothing | `cargo check` | `check` |
| `markitect vet` | Analyses the spec's design: coupling, size, fan-in, how far a change reaches. It warns, and exits 1 only with `--strict` | `go vet` | `lint`, the planned `analyze` |
| `markitect list` | The Definitions and Packages; with `--deps`, what each depends on | `go list -deps` | `graph` |
| `markitect doc DEFINITION` | A Definition with its text, what links to it, what it links to, the files that realize it, and its record | `go doc` | `refs` |
| `markitect status` | What is to verify, with which inputs changed, where to look, and each item's fingerprint. It also notes whether the binary and the harness are current, which never changes its exit code. `--all` lists every Definition, to verify everything again, and so exits 1 whenever the spec has one | `git status`, `npm outdated` | `status`, `impact`, `checklist`, `gate --verify` |
| `markitect review DEFINITION VERDICT --at FINGERPRINT [EVIDENCE…] [-- WHY]` | Records a verdict at the state the reviewer read | `go test` writing its cache | `answer`, `gate`, `gate --spec-problem` |
| `markitect lock`, `update`, `install`, `version`, `licenses` | As today | `go mod`, `go version` | the same |

**What goes:**
- `checklist` with all its flags;
- `answer`;
- `gate`, with `--verify`, `--findings` and `--spec-problem`;
- `impact` and `graph`;
- the work state in the cache;
- certificates and Result pages;
- repair rounds;
- the hook's rule on `checklist`.

**What remains of `gate`:** `check` still fails on a project that does not compile, and `status` lists the realization's errors. So an empty `status` means two things: the project compiles, and every Definition is verified at its current inputs.

## Skills, agents and the hook

| Skill | Agent | Tells the agents |
|---|---|---|
| `/markitect-build` | `markitect-builder`, which starts `markitect-reviewer` as its own subagent | Start the builder. It runs `markitect status`, makes the project match the spec for what it lists, then starts the reviewer, which verifies each item and records it with `markitect review`. It repeats until `status` is empty or an item needs the owner. Then the session commits the work with `.markitect/review.sum` |
| `/markitect-review` | `markitect-reviewer` | Verify what `status` lists, or everything with `--all`, and record each verdict. Change no code. Report what failed or is a `spec-problem` |

`/markitect-build` replaces `/markitect-apply`, and `/markitect-review` replaces `/markitect-verify`. So each skill is named after the agent it starts, and `markitect review` is the reviewer's command.

The skills inject no command, so a non-zero exit cannot abort them. The agents run the tool as they need it, and hold their task and the verdict words in their own files, which outlive compaction. Nothing counts rounds; the owner stops the work when it should stop.

### The hook

The hook stays, as a guard of the record and the spec, not of the process:

- **Only the reviewer agent may run `markitect review`,** as only it may run `answer` today. So the agent that built does not verify its own work. In the trials, a separate reviewer caught 17 of 18 planted bugs ([checker test](../validation/k-checker-test-20261103.md)).
- **No subagent may write or edit under `.markitect/`, except the registry pages, which the builder keeps.** So a builder cannot weaken a Definition during the work or write lines into the record. The owner's own session, where the spec is modelled, is not a subagent and stays free. The builder may start only the reviewer (`Agent(markitect-reviewer)`), so it cannot get past the rule through a general-purpose helper.

The hook reads tool calls, so a command hidden in a script, or one that names `markitect` by a path, still gets through. It guards against mistakes, as today, not against intent.

## Output and exit codes

- **Diagnostics** have one shape everywhere, `file:line: message` as Go's tools print them, with a stable code. With `--json`, each is an object with `severity`, `code`, `message` and the position.
- **JSON documents** share one envelope, with a `format_version` and a `kind`, so programs depend on the format and not on the wording.
- **Exit codes** follow Go's tools. 0 is success. 1 means the command did its job and found something: the spec does not compile, `status` lists something to verify, or `vet --strict` warns. Notes, such as a `skip` or an outdated harness, never change the exit code. 2 is a usage error. So a CI job runs `markitect status`, and any item fails the build.

## What this gives up, and what it keeps

| Today | With the cache |
|---|---|
| Every reached Definition is reviewed against the files as they are | Kept. A change of any input puts it back on the list, and `--at` binds the verdict to what the reviewer read |
| A changed test that served as evidence voids the answer | Kept. The evidence files are inputs |
| A changed file that realizes a Definition is reviewed | Kept |
| A changed file the ignore file lists is refused (`gate`, row 4) | **Lost.** Ignored files are not tracked |
| A changed harness or `settings.json` is refused during the work (row 1) | **Lost.** It shows in the diff |
| The spec does not change during the work | Kept for subagents by the hook. Not prevented for the owner's session or for people: a changed Definition goes back on the list, and the change shows in the diff |
| Only a reviewer records verdicts | Kept by the hook |
| The project compiles | Kept: `check`, and `status` lists the realization's errors |
| A gone Definition's files are dealt with | Kept: it stays on the list, marked gone |
| What a changed Definition said before | **Lost** from the tool; an agent reads it from Git |
| The work has a limit | None, by the owner's decision |
| A record of what was checked | Kept, per Definition, with its evidence |

What it gains:
- no state outside the project;
- no session that must hold a piece of work;
- no order the tool imposes;
- parallel reviewers without coordination;
- verification at any time and in any amount, by any driver, a person included.

## Decisions

Decided by the owner on 7 November 2026 ([DEC-099](register.md#dec-099-markitect-is-a-tool-with-a-verification-cache)):

1. **The verification cache replaces the certificate, the work state, `gate`, `answer` and the reports.**
2. **The inputs** are the Definition's text, its Kind and the Definitions it links to (a Library one by its digest), its registry rows with their files, and the evidence files. `review --at` binds a verdict to what was read.
3. **The record** is `.markitect/review.sum`: one sorted JSON line per Definition with the verdict, the inputs' hashes, the evidence and the reason. It merges with `merge=union`.
4. **The hook** lets only the reviewer record, and lets no subagent write under `.markitect/` except the registry pages.
5. **The names:** `check`, `vet`, `list`, `doc`, `status` and `review`; the verdicts `ok`, `fail`, `skip` and `spec-problem`; the skills `/markitect-build` and `/markitect-review`.
6. **Output and exit codes as Go's tools have them**, with `vet` exiting 1 only with `--strict`.
7. **Migration.** `update` writes the record from the newest certificate. It computes each Definition's inputs from the spec and file hashes that the certificate holds, and records every Definition as `ok` at that state, with the certificate's pointers where it has them. A `not shown` answer becomes `skip`. For one release, `checklist`, `answer` and `gate` remain as stubs that say to run `markitect update`, so an old project's skill does not fail silently.

Concept 4.0 is revised in the same change as the code, since its §3, §5 to §9 and §12 describe the work and `gate`. The project workflow, the model language, the README and [the work](../../internal/run/README.md) change with it.

## Order of work

1. **Build the cache.**
   - Build the inputs, the record, `status`, `review --at`, the hook's two rules and the migration.
   - Remove `gate`, `answer`, `checklist`, `impact`, the work state, certificates and Result pages.
   - Rewrite the skills and agents as prompts under their new names, and add the stubs.

   One PR and a release, since the harness and the binary change together.
2. **Output and the other names.** The JSON envelope, the diagnostics, the exit codes, `vet`, `list` and `doc`. One PR.
3. **Try it** on a small project with planted defects, under a bounded dispatch. Include a weakened test that served as evidence, and a builder's edit of the spec. The Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks).

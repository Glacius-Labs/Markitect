# Plan: Markitect as a tool, with a verification cache

**Status:** proposed on 7 November 2026, after a discussion with the owner about how they work with Markitect. The owner has agreed to three points:

- Markitect supports agent work and does not steer it.
- A record of what was verified is kept, modelled on Go's test cache.
- `/markitect-verify` becomes `/markitect-review`, and the work needs no limit on rounds.

An independent architect reviewed this page against the code. Everything else is a proposal until the owner decides ([decisions](#decisions-for-the-owner)). Command names here are working names, to be chosen with the owner.

An earlier version of this page proposed a command line like Terraform's, in which the tool steered the work through phases. This version replaces it and reuses its parts on output and exit codes. It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and on DEC-096 and DEC-097 ([lean plan](lean-tool-plan.md)).

## How the owner works with Markitect

1. **Model.** In an ordinary session, the owner designs the spec with an agent: Definitions, Packages, links, Kinds. The tool checks that the spec is structurally sound and free of conflicts, as a compiler does. It analyses the spec's design the way code is analysed for coupling. It shows the dependencies and what a change reaches.
2. **Implement.** When the spec is right, agents make the project match it. That is ordinary agent work in sessions and subagents, with prompts. The tool does not drive it.
3. **Make sure nothing is forgotten.** The tool says what has to be verified for the spec to count as implemented. The agents verify each item and record it. When nothing is left to verify, the spec is implemented.

The tool helps at three points: describing, saying what to do, and showing that it is done. It decides nothing about who works when. The skills are prompts that tell the agents to use the tool.

## The idea: a cache of what was verified

`go test` does not run a test again when nothing it depends on has changed; it prints `(cached)`. Markitect does the same for verification. For each Definition, it records the state at which an agent last verified it as correctly implemented. That state is the set of everything the verification depended on, its **inputs**:

- the Definition's own text;
- its Kind, and every Definition it links to: their text, or for a Library Definition the Library's digest from the lock, so that a newer binary changes nothing;
- the registry rows that name it, and the content of the files they name;
- the files the reviewer gave as evidence, such as a test that lies in another Definition's rows.

Its fingerprint is the hash of these inputs.

**What is to verify** is every Definition with a changed input, or with no record. The tool computes this from the folder and the record alone, with no session and no start. These are the same Definitions today's checklist reaches ([concept 4.0 §3](v4-concept.md#how-impact-works)):

- one that changed;
- one that links to a changed Definition, or is of a changed Kind;
- one whose files changed;
- one whose registry row gained or lost it.

In both designs, reach stops after one step, and a changed Rule above a Package reaches only itself and what links to it.

Three more kinds of item stay on the list until they are verified again:

- **A Definition someone recorded as `wrong` or `spec-problem`** stays with that note.
- **A Definition that is gone from the spec** stays, marked gone. Its line names the files that realized it, so the item can ask whether they are removed or have a new basis. Reviewing it `ok` removes its line.
- **An error in the realization** stays: an unlisted file, a stale registry row, an unused Definition.

Since the record holds each input's hash, `plan` says which inputs changed, and where to look. It does not hold what a Definition said before. An agent that wants the old text reads it from Git.

**Recording** binds a verdict to the state the reviewer read. `plan` prints each item's fingerprint. `review DEFINITION VERDICT --at FINGERPRINT` refuses if the inputs have changed since, for example because a builder working beside the reviewer edited a file, or the reviewer's own test run changed one. Go does the same: it does not cache a result whose inputs changed while the test ran. The fingerprint `plan` prints covers every input but the evidence, which the reviewer chooses later. The evidence must exist in the folder, and its files are hashed when the verdict is recorded.

**Done** is an empty list, and that holds whatever happened in between. A file edited by hand, a commit made around the agents, or a change on a merged branch alters the inputs it touches, and those Definitions are on the list again.

## The record

`.markitect/verified` (working name) is committed with the project, like `go.sum`. It joins the fixed layout of `.markitect/`. It holds one line per Definition, sorted, as JSON on one line:

```text
{"definition":"fees/waiver","verdict":"ok","inputs":{"self":"sha256:…","link:fees/fee":"sha256:…","file:src/fees.go":"sha256:…","evidence:tests/fees_test.go":"sha256:…"},"evidence":["src/fees.go:40","tests/fees_test.go:12 (TestWaiver)"]}
{"definition":"members/limit","verdict":"wrong","inputs":{…},"evidence":["src/loans.go:70"],"why":"allows a sixth loan; the Definition says five"}
```

- **Verdicts:** `ok`, `not-shown`, `wrong` and `spec-problem`. `not-shown` means verified as far as the project shows; `plan` lists it as a note, as the Result lists it today, and a note alone does not make `plan` exit 1. An explanation stays on one line. Hashes are full SHA-256.
- **Writing:** `review` writes under the lock the tool already uses, as one whole file, so parallel reviewers, one per Package for example, do not lose lines.
- **Merging:** Git also conflicts on changes to neighbouring lines. So `init` writes `.markitect/.gitattributes`, which joins the fixed layout, with `merge=union` for this file: Git keeps both sides. Of two lines for one Definition, `plan` takes the one whose inputs match the folder. If both match with different verdicts, `plan` lists the Definition as a conflict to review. The next `review` writes the file clean.
- **What it replaces:** today's certificate, the stored answers, the work under way in the cache, the Result pages, the Findings pages and the builder's reports.
- **What it is:** a record, not a proof, as the certificate is today. A line written by hand shows in the diff, and the hook keeps agents from writing it ([below](#the-hook)).

## The commands

The working names lean on Go.

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `go mod init` | `init` |
| `markitect check` | Compiles the spec and the registry: Kinds, links, Packages, Libraries against the lock, and every file listed or ignored | `go build` | `check` |
| `markitect vet` | Analyses the spec's design: coupling, size, fan-in, and how far a change reaches. It warns, and exits 1 only with `--strict` | `go vet`, `staticcheck` | `lint`, the planned `analyze` |
| `markitect list` | The Definitions and Packages; with `--deps`, what each depends on | `go list` | `graph` |
| `markitect doc DEFINITION` | A Definition with its text, what links to it, what it links to, the files that realize it, and its record | `go doc` | `refs` |
| `markitect plan` | What is to verify, with which inputs changed, where to look, and each item's fingerprint. `--all` lists every Definition, to verify everything again | `go test` without the cache | `impact`, `checklist`, `gate --verify` |
| `markitect review DEFINITION VERDICT --at FINGERPRINT [EVIDENCE…] [-- WHY]` | Records a verdict at the state the reviewer read | `go test` writing its cache | `answer`, `gate`, `gate --spec-problem` |
| `markitect lock`, `version`, `update`, `install`, `licenses` | As today | `go mod`, `go version` | the same |

**What goes:** `checklist` with all its flags, `answer`, `gate` with `--verify`, `--findings` and `--spec-problem`, `impact`, the work part of `status`, the work state in the cache, certificates and Result pages, repair rounds, and the hook's rule on `checklist`.

**What remains of `gate`:** `check` still fails on a project that does not compile, and `plan` lists the realization's errors. So an empty `plan` means two things: the project compiles, and every Definition is verified at its current inputs.

## The skills are prompts

| Skill | Tells the agents |
|---|---|
| `/markitect-apply` | Run `markitect plan`. Make the project match the spec for what it lists, with the `markitect-builder` agent. Then have the `markitect-reviewer` agent verify each item and record it with `markitect review`. Repeat until `plan` is empty, or until an item needs the owner. Then commit the work with `.markitect/verified` |
| `/markitect-review` | Have the `markitect-reviewer` agent verify what `plan` lists, or with `--all` everything, and record each verdict. Change no code. Report what is `wrong` or a `spec-problem` |

The skills inject no command, so a non-zero exit cannot abort them. The agents run the tool as they need it. They hold their task and the verdict words in their own files, which outlive compaction. Nothing counts rounds; the owner stops the work when it should stop.

### The hook

The hook stays, as a guard of the record and the spec, not of the process:

- Only the reviewer agent may run `markitect review`, as it alone may run `answer` today. So the agent that built does not verify its own work. In the trials, a separate reviewer caught 17 of 18 planted bugs ([checker test](../validation/k-checker-test-20261103.md)).
- No subagent may write or edit under `.markitect/`, except the registry pages, which the builder keeps. So a builder cannot weaken a Definition during the work or write lines into the record. The owner's own session, where the spec is modelled, is not a subagent and stays free. The builder may start only the reviewer (`Agent(markitect-reviewer)`), so it cannot get past the rule through a general-purpose helper.

The hook reads tool calls, so a command hidden in a script or a path past `markitect` still gets through. That guards against mistakes, as today, not against intent.

## Output and exit codes

These are reused from the earlier version of this page, adapted to Go:

- **Diagnostics** have one shape everywhere, `file:line: message` as Go's tools print them, with a stable code. With `--json`, each is an object with `severity`, `code`, `message` and the position.
- **JSON documents** share one envelope with a `format_version` and a `kind`, so programs depend on the format and not on the wording.
- **Exit codes** follow Go's tools. 0 is success. 1 means the command did its job and found something: the spec does not compile, or `plan` lists something, or `vet --strict` warns. 2 is a usage error. So a CI job runs `markitect plan`, and any item fails the build.

## What this gives up, and what it keeps

| Today | With the cache |
|---|---|
| Every reached Definition is reviewed against the files as they are | Kept. A change of any input puts it back on the list, and `--at` binds the verdict to what the reviewer read |
| A changed test that served as evidence voids the answer | Kept. The evidence files are inputs |
| A changed file that realizes a Definition is reviewed | Kept |
| A changed file the ignore file lists is refused (`gate`, row 4) | **Lost.** Ignored files are not tracked |
| A changed harness or `settings.json` is refused during the work (row 1) | **Lost.** It shows in the diff |
| The spec does not change during the work | Kept for subagents by the hook. For the owner's session and for people, it is not prevented: a changed Definition goes back on the list, and the change shows in the diff |
| Only a reviewer records verdicts | Kept by the hook |
| The project compiles | Kept: `check`, and `plan` lists the realization's errors |
| A gone Definition's files are dealt with | Kept: it stays on the list, marked gone |
| What a changed Definition said before | **Lost** from the tool; an agent reads it from Git |
| The work has a limit | None, by the owner's decision |
| A record of what was checked | Kept, per Definition, with its evidence |

What it gains:

- No state outside the project, and no session that must hold a piece of work.
- No order the tool imposes.
- Parallel reviewers without coordination.
- Verification at any time and in any amount, by any driver, a person included.

## Decisions for the owner

1. **The verification cache replaces the certificate, the work state, `gate`, `answer` and the reports.** The direction was agreed on 7 November 2026; this is its form.
2. **The inputs:** the Definition's text, its Kind and linked Definitions (a Library one by its digest), its registry rows and their files, and the evidence files. `review --at` binds a verdict to what was read. Recommended.
3. **The record:** one committed file with one sorted JSON line per Definition, holding the verdict, the inputs' hashes, the evidence and the reason, merged with `merge=union`. Recommended.
4. **The hook:** only the reviewer records, and no subagent writes under `.markitect/` except the registry pages. Recommended, as structure for independence and for the spec. The alternative leaves both to the prompt.
5. **The names.** Working names: `check`, `vet`, `list`, `doc`, `plan`, `review`. To be chosen with the owner.
6. **Output and exit codes as Go's tools have them**, with `vet` warning and exiting 1 only with `--strict`. Recommended.
7. **Migration.** `update` writes the record from the newest certificate. It computes each Definition's inputs from the spec and the file hashes the certificate holds. It records every Definition as `ok` at that state, with the certificate's pointers where it has them, and keeps `not shown` as `not-shown`. For one release, `checklist`, `answer` and `gate` remain as stubs that say to run `markitect update`, so an old project's skill does not fail silently. Recommended.

Once accepted, the decisions go into the register as one entry. Concept 4.0 is revised in the same change as the code, since its §3, §5 to §9 and §12 describe the work and `gate`. The project workflow, the model language, the README and [the work](../../internal/run/README.md) change with it.

## Order of work

1. **Decide** the points above, and the names.
2. **Build the cache.**
   - Add the inputs, the record, `plan`, `review --at`, the hook's two rules and the migration.
   - Remove `gate`, `answer`, `checklist`, the work state, certificates and Result pages.
   - Rewrite the skills and agents as prompts, and add the stubs.

   This is one PR and a release, since the harness and the binary change together.
3. **Output and the other names:** the JSON envelope, the diagnostics, the exit codes, `vet`, `list` and `doc`. One PR.
4. **Try it** on a small project with planted defects, under a bounded dispatch. Include a weakened test that served as evidence, and a builder's edit of the spec. The Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks).

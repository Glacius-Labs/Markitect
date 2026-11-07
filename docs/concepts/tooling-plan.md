# Plan: Markitect as a tool, with a verification cache

**Status:** proposed on 7 November 2026, after a discussion with the owner of how they work with Markitect. Accepted so far:
- the direction: Markitect supports agent work and does not steer it, with a record of what was verified, modelled on Go's test cache;
- `/markitect-verify` becomes `/markitect-review`;
- the work needs no limit on rounds.

Everything else is a proposal until the owner decides ([decisions](#decisions-for-the-owner)). Command names here are working names, to be chosen with the owner. An earlier version of this page proposed a command line like Terraform's, in which the tool steered the work through phases. It is replaced by this one, and its parts on output and exit codes are reused. The plan builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and on DEC-096 and DEC-097 ([lean plan](lean-tool-plan.md)).

## How the owner works with Markitect

1. **Model.** In an ordinary session, the owner designs the spec with an agent: Definitions, Packages, links, Kinds. The tool checks that the spec is structurally sound and free of conflicts, like a compiler. It analyses its design the way code is analysed for coupling, and shows dependencies and what a change reaches.
2. **Implement.** When the spec is right, agents make the project match it. That is ordinary agent work in sessions and subagents, with prompts. The tool does not drive it.
3. **Make sure nothing is forgotten.** The tool says what has to be verified for the spec to count as implemented. The agents verify each item and record it. When nothing is left to verify, the spec is implemented.

The tool helps at three points: describing, saying what to do, and showing that it is done. It decides nothing about who works when. The skills are prompts that tell the agents to use the tool.

## The idea: a cache of what was verified

`go test` does not run a test again when nothing it depends on changed: it prints `(cached)`. Markitect does the same for verification. For each Definition, it records the state at which an agent last verified it as correctly implemented. That state is a fingerprint of everything the verification depended on:

- the Definition's own text;
- the text of its Kind, and of every Definition it links to;
- the registry rows that name it, and the content of the files they name.

**What is to verify** is every Definition whose fingerprint differs from the recorded one, or that has no record. The tool computes that from the folder and the record alone, with no session and no start. These are the same Definitions today's checklist reaches: one that changed, one that links to or is of a changed Definition, one whose files changed, and one whose registry row gained or lost it ([concept 4.0 §3](v4-concept.md#how-impact-works)). A Definition someone recorded as wrong, or as a spec problem, stays on the list with that note until it is verified again.

**Recording** is one command an agent runs per item, or for several items: the verdict, the places it read as evidence, and for anything but `ok` the reason. The evidence must exist in the folder, as answers must today.

**Done** is an empty list. That holds whatever happened in between. A file edited by hand, a commit made around the agents, or a change on a merged branch alters the fingerprints it touches, and those Definitions are on the list again.

## The record

`.markitect/verified` (working name) is committed with the project, like `go.sum`. It has one line per Definition, sorted, so that branches merge line by line:

```text
fees/waiver    3f9a…c2  ok             src/fees.go:40 tests/fees_test.go:12 (TestWaiver)
members/limit  81be…07  wrong          src/loans.go:70 -- allows a sixth loan; the Definition says five
loans/renewal  5c01…9e  spec-problem   -- the table says three renewals, the example four
```

- **Merges.** A line conflicts only when both branches verified the same Definition differently, and resolving the conflict is reviewing that Definition.
- **What the record replaces.** It takes the place of today's certificate, the stored answers, the work under way in the cache, the Result pages, the Findings pages and the builder's reports. A Definition that is gone from the spec drops its line once no registry row names it, which `check` already requires.
- **What it is.** A record, not a proof, as the certificate is today. A line can be written by hand; that is visible in the diff.

## The commands

The working names lean on Go.

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `go mod init` | `init` |
| `markitect check` | Compiles the spec and the registry: Kinds, links, Packages, Libraries against the lock, every file listed or ignored | `go build` | `check` |
| `markitect vet` | Analyses the spec's design: coupling, size, fan-in, how far a change reaches | `go vet`, `staticcheck` | `lint`, the planned `analyze` |
| `markitect list` | The Definitions and Packages, with `--deps` what each depends on | `go list` | `graph` |
| `markitect doc DEFINITION` | A Definition with its text, what links to it, what it links to, and the files that realize it | `go doc` | `refs` |
| `markitect plan` | What is to verify, with why and where to look. `--all` lists every Definition, to verify everything again | `go test` without the cache | `impact`, `checklist`, `gate --verify` |
| `markitect review DEFINITION VERDICT [EVIDENCE] [-- WHY]` | Records a verdict, `ok`, `wrong` or `spec-problem`, at the Definition's current fingerprint | `go test` writing its cache | `answer`, `gate`, `gate --spec-problem` |
| `markitect lock`, `version`, `update`, `install`, `licenses` | As today | `go mod`, `go version` | the same |

- **What goes:** `checklist` with all its flags, `answer`, `gate` with `--verify`, `--findings` and `--spec-problem`, `impact`, `status`'s work part, the work state in the cache, certificates and Result pages, repair rounds, and the hook's guard on `checklist`.
- **What remains of `gate`:** `check` still fails on a project that does not compile. `plan` lists, with the Definitions to verify, any error in the realization, such as an unlisted file or a stale registry row. So an empty `plan` means the project compiles and every Definition is verified at its current state.

## The skills are prompts

| Skill | Tells the agents |
|---|---|
| `/markitect-apply` | Run `markitect plan`. Make the project match the spec for what it lists, with the `markitect-builder` agent. Then have the `markitect-reviewer` agent verify each item and record it with `markitect review`. Repeat until `plan` is empty or an item needs the owner. Then commit the work with `.markitect/verified` |
| `/markitect-review` | Have the `markitect-reviewer` agent verify what `plan` lists, or with `--all` everything, and record each verdict. Change no code. Report what is `wrong` or a `spec-problem` |

The skills inject no command, so a non-zero exit cannot abort them. The agents run the tool as they need it, and they hold their task and the verdict words in their own files, which outlive compaction. Nothing counts rounds. The owner stops the work when it should stop.

**Who may record.** The tool records whatever it is told. Today's hook lets only the reviewer agent run `answer`. The same hook can let only the reviewer run `markitect review`. That keeps "the agent that built does not verify itself" in structure for the cost of one rule. In the trials, a separate reviewer caught 17 of 18 planted bugs ([checker test](../validation/k-checker-test-20261103.md)). It is a decision for the owner ([below](#decisions-for-the-owner)).

## Output and exit codes

These are reused from the earlier version of this page, and adapted to Go:

- **Diagnostics** have one shape everywhere, as Go's tools print them, `file:line: message`, with a stable code. With `--json`, they are objects with `severity`, `code`, `message` and the position.
- **JSON documents** share one envelope, with a `format_version` and a `kind`, so programs depend on the format and not on the wording.
- **Exit codes** follow Go's tools. 0 is success. 1 means the command did its job and found something: the spec does not compile, `vet` warns, or `plan` lists something. 2 is a usage error. So a CI job runs `markitect plan`, and any item fails the build.

## What this gives up, and what it keeps

| Today's guarantee | With the cache |
|---|---|
| Every reached Definition is reviewed for the files as they are | Kept: a change of any input puts the Definition back on the list |
| No changed file escapes review | Kept for every file that realizes a Definition. Ignored files are not tracked, as their rows say |
| The project compiles | Kept: `check`, and `plan` lists the realization's errors |
| The spec does not change during the work | Not prevented: a changed Definition is on the list again, and the change shows in the spec's diff |
| Only a reviewer records verdicts | Kept if the hook stays ([decision 4](#decisions-for-the-owner)) |
| The work has a limit | None, by the owner's decision. The owner stops it |
| A record of what was checked | Kept, per Definition, in the record, with evidence |

What it gains:
- no state outside the project;
- no session that must hold a piece of work;
- no order the tool imposes;
- parallel reviewers, for example one per Package, without coordination;
- verification that can happen at any time and in any amount, by any driver, a person included.

## Decisions for the owner

1. **The verification cache replaces the certificate, the work state, `gate`, `answer` and the reports.** The direction was agreed on 7 November 2026; this is its form.
2. **The fingerprint** is the Definition's own text, its Kind's text, the texts of the Definitions it links to, and its registry rows with their files' content. Recommended: it matches today's reach.
3. **The record** is one committed file with one sorted line per Definition, holding the verdict `ok`, `wrong` or `spec-problem`, the evidence and the reason. Recommended.
4. **Only the reviewer agent may record**, through the hook. Recommended, as structure for independence at the cost of one rule. The alternative leaves it to the prompt.
5. **The names.** Working names: `check`, `vet`, `list`, `doc`, `plan`, `review`. They are to be chosen with the owner.
6. **Output and exit codes as Go's tools have them.** Recommended.
7. **Migration.** `update` writes the record from the newest certificate. Its answers and hashes hold everything a fingerprint needs. For one release, `checklist`, `answer` and `gate` remain as stubs that say to run `markitect update`, so an old project's skill does not fail silently. Recommended.

Once accepted, the decisions go into the register as one entry. Concept 4.0 is revised in the same change as the code, because its §3, §5 to §9 and §12 describe the work and `gate`. The project workflow, the model language, the README and [the work](../../internal/run/README.md) change with it.

## Order of work

1. **Decide** the points above, and the names.
2. **Build the cache.** The fingerprint, the record, `plan`, `review`, and the migration from the newest certificate. Remove `gate`, `answer`, `checklist`, the work state, certificates and Result pages. Rewrite the skills and agents as prompts, with the stubs, and settle the hook as decided. This is one PR and a release, since the harness and the binary change together.
3. **Output and the other names.** The JSON envelope, the diagnostics, the exit codes, `vet`, `list` and `doc`. One PR.
4. **Try it** on a small project with planted defects, under a bounded dispatch. The Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks).

# Plan: a command line like Terraform's

**Status:** proposed on 7 November 2026, at the owner's request, after a discussion of what Markitect's commands put out. The owner asked for output that is as uniform and clean as is sensible, and for a command line that feels like the tools people know, such as Terraform and kubectl. They decided that the work needs no maximum number of repair rounds. An independent architect reviewed earlier versions. Nothing else here is decided until the owner accepts it ([decisions](#decisions-for-the-owner)). It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and the [lean plan](lean-tool-plan.md) ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)).

## The idea

Markitect is Terraform for a spec. The spec is the desired state, the project's files are the real one, and the certificate is the state file: what was last certified to match. A plan says what differs. Applying makes it match. In Terraform, providers do the real work of an apply; in Markitect, agents do, in the owner's own Claude Code session, so applying is a skill and not a command. Everything else follows Terraform's and kubectl's conventions, so that someone who knows them can use Markitect at once.

Today the commands are named after Markitect's own ideas (`check`, `impact`, `checklist`, `refs`), print prose meant to steer agents, write Result pages on their own, and use exit codes in three different ways.

## The commands

**For people**, at the top level:

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `terraform init` | `init` |
| `markitect validate` | Compiles the spec and the registry. `--spec` looks at the spec alone | `terraform validate` | `check` |
| `markitect plan` | What differs between the spec and the newest certificate: what is to be built, changed, removed and reviewed, and where to look. `--base CERTIFICATE`, `--full` and `--findings FILE` name another change, and `-out FILE` saves the plan | `terraform plan` | `impact`, and `checklist`'s list |
| `/markitect-apply [PLAN]` | Makes the project match the spec: the skill starts the builder and the reviewer in the owner's session. Given a saved plan, it does exactly that plan | `terraform apply` | the same skill |
| `markitect status` | Whether work is under way and how it stands, and which files differ from the newest certificate | `git status`, `terraform plan -detailed-exitcode` for drift | `status`, and `gate --verify` |
| `markitect show FILE` | Shows a certificate, Findings or a saved plan | `terraform show` | the Result pages |
| `markitect describe DEFINITION` | A Definition with its text, what links to it, what it links to, and the files that realize it | `kubectl describe` | `refs` |
| `markitect graph` | The Package graph | `terraform graph` | `graph` |
| `markitect lint` | Measures the spec's design; it only warns | `tflint` | `lint` (to become `analyze`, [concept 4.0 §4](v4-concept.md#4-is-the-spec-good)) |
| `markitect lock` | Pins the Libraries | `terraform providers lock` | `lock` |
| `markitect version`, `update`, `install`, `licenses` | As today | `terraform version`, `claude update` | the same |

**For drivers and agents**, under one noun, as `terraform state …` groups its commands:

| Command | Called by | Does | Today |
|---|---|---|---|
| `markitect work start [PLAN]` | The skill | Begins the work, or goes on with the work under way. `--review` begins a review of finished work | `checklist --start` |
| `markitect work list` | The builder; with `--review`, the reviewer | The work's list: for the builder every item, what changed and what was sent back; for the reviewer only what still needs an answer | `checklist`, `checklist --review` |
| `markitect work answer` | The reviewer | Stores the answers, bound to the files they cover | `answer` |
| `markitect work report WHY` | The builder | Reports a spec problem for a reviewer to confirm ([lean plan, decision 7](lean-tool-plan.md#decisions-for-the-owner)) | `gate --spec-problem` |
| `markitect work gate` | The builder and the skill | The one door ([concept 4.0 §6](v4-concept.md#6-the-one-door-gate)). It certifies the work when nothing is left. On a review of finished work, it closes the review with its Findings instead | `gate`, `gate --findings` |
| `markitect hook` | Hooks only | The guard of the harness | `hook` |

`gate` keeps the name the owner chose on 6 November 2026; it moves under `work`. A plain `plan` and `work list` record nothing: only `work start`, `work answer`, `work report` and `work gate` change the work's state.

## Output

**Text for people, JSON for programs**, as in Terraform. Every command takes `--json`. Every JSON document has the same envelope, with a `format_version`, a `kind` (`Plan`, `Validation`, `Status`, `Certificate`, `Findings`, `Description`, `Graph`, `Work`) and the fields of that kind. Programs depend on the format version, not on the wording of the text.

**A plan reads like Terraform's:**

```text
Markitect will review these Definitions:

  + fees/waiver       "Fee waiver"     new
  ~ members/limit     "Loan limit"     changed
      look at: src/loans.go, tests/loans_test.go
  ~ loans/renewal     "Renewal"        links to a changed Definition
  - fees/legacy       "Legacy fee"     gone: its files go, or get a new basis
  ! finding 1         members/limit    observed 6 open loans; expected 5
  ? src/tmp.go                         neither listed nor ignored

Plan: 1 to add, 2 to change, 1 to remove, 2 to settle. 4 files to look at.
```

**Diagnostics have one shape everywhere.** Errors of `validate`, warnings of `lint`, refusals of `work gate` and answers that `work answer` refuses all have a severity, a stable code, a summary, a detail and a place, as in Terraform:

```text
Error: Unresolved link

  on .markitect/model/members/limit.md line 12:
  [[fees/waivr]] names no Definition.
```

In JSON they are a `diagnostics` array with `severity`, `code`, `summary`, `detail` and `range`, as in `terraform validate -json`.

**Exit codes follow `terraform plan -detailed-exitcode` and `kubectl diff`:**

| Code | Means |
|---|---|
| 0 | Done, and nothing to do: the spec is valid, the plan is empty, the folder matches its certificate, the work is certified |
| 1 | Error: the command could not do its job, or what it validates is invalid |
| 2 | Done, and there is something to do: the plan has changes, the folder differs from its certificate, the work is not yet certified, the binary or harness is out of date |

So a CI job runs `markitect status`, and a non-zero code fails the build. That replaces `gate --verify`. One command is an exception. `work start` always exits 0 and prints any failure on standard output, because Claude Code aborts a skill whose injected command exits non-zero, and the owner's session would then see nothing.

## The work as a status

The work under way is described the way kubectl describes an object, by its phase:

| Phase | Means | The default driver |
|---|---|---|
| `Building` | The builder has work: the start, or what a review sent back, or what `gate` refused | Starts `markitect-builder` with "Build the work." |
| `Reviewing` | Items have no answer for the files as they are, or a report waits for a reviewer | Starts `markitect-reviewer` with "Review the work." |
| `NeedsOwner` | A spec problem, a spec that does not compile, or no work under way where work was expected | Tells the owner, with the question |
| `Certified` | `gate` wrote the certificate and ended the work | Commits the work with the certificate |

The commands that change the work record the phase as they make the change: `work start`, `work answer`, `work report` and `work gate`. So `status` and `work list` read a recorded fact and never guess a step. Before any building, `gate`'s rows alone could not tell "nothing built yet" from "built, waiting for review". The phase comes with a `reason` code and a one-line message.

**The tool names no agent.** Which agent serves which phase, and with what message, is the driver's, and lives in the skill: four lines. Everything else that is prose for agents lives in the agent files too: the builder's task, the reviewer's task and the answer format, and what to do for each diagnostic code. An agent's file is its system prompt, so it survives the compaction of that agent's context. The harness carries a format version that changes only when phases, codes or the agents' rules change, and `work start` refuses a harness of another format and says to run `markitect update`. A Go test checks that the embedded agent files cover every code the tool can emit. When `gate` sends the work back, the phase is `Building` whoever asks; the builder's file says that this means itself, and its tools let it start only the reviewer (`Agent(markitect-reviewer)`).

**`work gate` and `work answer` with no work under way** give `NeedsOwner`. They never point to `work start`, so an agent that lost the work cannot begin it again from a spec it changed.

## Files

| Where | What | Like |
|---|---|---|
| `.markitect/` | The spec, the registry, the manifest, the lock, the ignore file | The configuration and `.terraform.lock.hcl` |
| `.markitect/results/` | Certificates and Findings, as JSON. `markitect show` renders them | The state file, and `terraform show` |
| A saved plan, wherever `-out` puts it | What `/markitect-apply PLAN` will do. `work start` refuses a plan that is stale, because the spec, the base certificate or the harness changed since | `terraform plan -out` |
| The user's cache folder | The work under way, as today, one file per project folder | `.terraform/` |

A refusal writes nothing into the project; its result says why. A plan file inside the project must be one the walk ignores, or `gate` would see an unlisted file, so `-out` refuses such a path.

Keeping the work under way in the cache was weighed against `.markitect/work/`. That folder would be a spec error under the fixed layout ([`project.Load`](../../internal/project/project.go)). It would also need a guard on Write and Edit in the hook, and a migration in `update`. `git clean -X` would wipe the work. Every worktree already has its own state, since the cache is keyed by the folder's path.

## No limit on rounds

`gate` no longer refuses after two repair rounds, and the builder's reports of a spec problem are no longer capped at two that were turned down. The default driver sets no limit of its own: the work goes on until it is certified, needs the owner, or the owner stops it. The certificate records how many rounds it took.

Two rules about state, not counts, remain against loops that cost a review and change nothing:

- **Reports wait for a review.** After a reviewer turned a report down, the builder can report again only once a full review has finished.
- **Disagreement has no rule yet.** A builder and a reviewer that disagree without end are stopped by the owner. Unattended work, the scheduled jobs of [concept 4.0 §12](v4-concept.md#12-schedules-and-continuous-work), counted on the two rounds as their budget, so that work brings a bound back. A candidate: an item sent back on two reviews in a row goes to the owner.

What goes: `repairRounds`, the refusal after the rounds, the cap on rejected reports, and the fields and checks that served them.

## What this leaves alone

What the tool checks: `gate`'s rows apart from the round limit, the certificate's content, how answers bind to files, the folder walk, and how a plan reaches Definitions. Also the hook on answers, the commit by the driver, and a tool without Git. This plan changes the names, the output, the exit codes, and what the tool writes into the project.

## Decisions for the owner

Each has a recommendation.

1. **Terraform's and kubectl's vocabulary.** `validate`, `plan`, `status`, `show`, `describe`, and the agents' commands under `work`. The old names go with no aliases ([DEC-032](register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)). Recommended.
2. **Output as in Terraform.** Text for people, `--json` everywhere with one envelope and a format version, and one shape of diagnostic. Recommended.
3. **Exit codes 0, 1 and 2** with the meanings above, and `work start` as the one named exception. Recommended.
4. **The work as a phase**, recorded by the commands that change it. The tool names no agent, and the prose lives in the harness, with a format check and a code test. Recommended.
5. **Saved plans:** `plan -out FILE`, and `/markitect-apply FILE` doing exactly that plan or refusing a stale one. Recommended: the owner can read what will be built before any agent starts, as with Terraform.
6. **Records as JSON only, shown by `markitect show`**, with no Markdown pages beside them and no pages for refusals. Recommended, as Terraform keeps its state. The alternative keeps a generated page beside each record, for reading on GitHub.
7. **CI uses `markitect status`.** Its exit code replaces `gate --verify`. Recommended.
8. **The work under way stays in the user's cache folder.** Recommended.
9. **No limit on repair rounds**, decided by the owner on 7 November 2026. **No cap on rejected reports**, and a report waits for a full review. **No rule for disagreement until unattended work needs one.** Recommended.

Once accepted, the decisions go into the register as one entry. The pages that describe them change in the same pull requests as the code: concept 4.0 §3, §5, §6, §7, §9 and §12; the [project workflow](../guides/project-workflow.md#7-apply-and-verify); the [model language](../guides/model-language.md#commands); the README; and [the work](../../internal/run/README.md).

## Order of work

1. **Decide** the points above.
2. **Output and exit codes.** The JSON envelope, the diagnostic shape, the text renderings, and the exit codes, for every command under its current name. One PR.
3. **The new command line.** The renames, `describe`, `show`, `plan -out`, the `work` group with its phases, `status` for CI, records as JSON, and no round limit. The prose moves into the harness, with its format version and the code test. One PR and a release, since the harness and the binary change together.
4. **Try it** on a small project with planted defects, under a bounded dispatch. The evaluation projects are gone from Main; the Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks) for it.

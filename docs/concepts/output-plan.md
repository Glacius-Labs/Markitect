# Plan: a command line like Terraform's

**Status:** proposed on 7 November 2026 at the owner's request, after a discussion of what Markitect's commands put out. The owner asked for two things: output as uniform and clean as is sensible, and a command line that feels like the tools people know, such as Terraform and kubectl. They also decided that the work needs no maximum number of repair rounds. An independent architect reviewed each version. Nothing else here is decided until the owner accepts it ([decisions](#decisions-for-the-owner)). It builds on [concept 4.0](v4-concept.md) ([DEC-098](register.md#dec-098-concept-40-is-the-target)) and the [lean plan](lean-tool-plan.md) ([DEC-096](register.md#dec-096-markitect-computes-and-checks-a-driver-acts), [DEC-097](register.md#dec-097-markitect-uses-no-git)).

## The idea

Markitect is Terraform for a spec.

- The spec is the desired state, and the project's files are the real one.
- The certificate is the state file: what was last certified to match.
- A plan says what differs, and applying makes it match.

In Terraform, providers do the real work of an apply. In Markitect, agents do it, in the owner's own Claude Code session, so applying is a skill and not a command. Everything else follows Terraform's conventions, and kubectl's where Terraform has none, so that someone who knows them can use Markitect at once.

Today the commands are named after Markitect's own ideas (`check`, `impact`, `checklist`, `refs`). They print prose meant to steer agents, write Result pages on their own, and use exit codes in three different ways.

## The commands

**For people**, at the top level:

| Command | Does | Like | Today |
|---|---|---|---|
| `markitect init` | Sets up the project, its harness and its lock | `terraform init` | `init` |
| `markitect validate` | Compiles the spec: Kinds, links, Packages, Libraries against the lock. Like `terraform validate`, it looks at the configuration and not at the real state | `terraform validate` | `check --spec` |
| `markitect plan` | What differs between the spec and the folder since the newest certificate: what is to be built, changed and removed, what to settle, and where to look. That includes the realization's errors, such as an unlisted file, a stale registry row or an unused Definition, and every other file that differs from the certificate, the owner's files included: the manifest, the harness, a `.gitignore` file. So an empty plan means the folder matches its certificate, and a Go test holds the two to the same answer. `--base CERTIFICATE`, `--full` and `--findings FILE` name another change | `terraform plan` | `impact`, `checklist`'s list, the rest of `check`, and `gate --verify` |
| `/markitect-apply` | Makes the project match the spec: the skill starts the builder and the reviewer in the owner's session | `terraform apply` | the same skill |
| `markitect status` | Whether work is under way and its phase, with when that was recorded, and whether the binary and the harness are current | `git status` | `status`, `version`'s status lines |
| `markitect describe DEFINITION` | A Definition with its text, what links to it, what it links to, and the files that realize it | `kubectl describe` | `refs` |
| `markitect graph` | The Package graph | `terraform graph` | `graph` |
| `markitect lint` | Measures the spec's design. It only warns | `tflint` | `lint`, to become `analyze` ([concept 4.0 §4](v4-concept.md#4-is-the-spec-good)) |
| `markitect lock` | Pins the Libraries | `terraform providers lock` | `lock` |
| `markitect version`, `update`, `install`, `licenses` | As today | `terraform version`, `claude update` | the same |

**For drivers and agents**, under one noun, the way `kubectl config …` or `git remote …` group their commands:

| Command | Called by | Does | Today |
|---|---|---|---|
| `markitect work start` | The skill | Begins the work, or goes on with the work under way. `--review` begins a review of finished work. `--base`, `--full` and `--findings` name the change | `checklist --start` |
| `markitect work list` | The builder, and with `--review` the reviewer | The work's list. The builder sees every item, what changed and what was sent back. The reviewer sees only what still needs an answer | `checklist`, `checklist --review` |
| `markitect work answer` | The reviewer | Stores the answers, bound to the files they cover | `answer` |
| `markitect work report WHY` | The builder | Reports a spec problem for a reviewer to confirm ([lean plan, decision 7](lean-tool-plan.md#decisions-for-the-owner)) | `gate --spec-problem` |
| `markitect work gate` | The builder and the skill | The one door ([concept 4.0 §6](v4-concept.md#6-the-one-door-gate)). It certifies the work when nothing is left. On a review of finished work it closes the review with its Findings, since the work knows it is a review | `gate`, `gate --findings` |
| `markitect hook` | Hooks only | The guard of the harness | `hook` |

`gate` keeps the name the owner chose on 6 November 2026 and moves under `work`. `plan` and `work list` record nothing. Only `work start`, `work answer`, `work report` and `work gate` change the work's state.

**The old names** go, with no aliases ([DEC-032](register.md#dec-032-quality-simplicity-and-stability-over-effort-and-compatibility)), with one bridge for one release. An existing project's skill injects `markitect checklist --start`. If that command were simply unknown, the skill would abort and the owner would see nothing. So for one release `checklist`, `answer` and `gate` remain as stubs. Each prints on standard output that the project's harness is older than the binary and that `markitect update` brings it in line, and exits 0. The plugin's hooks change their command strings in the same release.

## Output

**Text for people, JSON for programs**, as in Terraform. Every command takes `--json`. Every JSON document has the same envelope: a `format_version`, a `kind` (`Validation`, `Plan`, `Status`, `Description`, `Graph`, `Lint`, `Work`, `Certificate`, `Findings`), and the fields of that kind. Programs depend on the format version, not on the wording of the text. Flags are written with two dashes throughout.

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

**Diagnostics have one shape everywhere.** That covers errors of `validate`, warnings of `lint`, refusals of `work gate`, and answers that `work answer` refuses. Each has a severity, a stable code, a summary, a detail and a place, as in Terraform:

```text
Error: Unresolved link

  on .markitect/model/members/limit.md line 12:
  [[fees/waivr]] names no Definition.
```

In JSON they are a `diagnostics` array with `severity`, `code`, `summary`, `detail` and `range`, as in `terraform validate -json`.

**Exit codes** take the meaning of `terraform plan -detailed-exitcode`, here for every command and by default:

| Code | Means |
|---|---|
| 0 | Done, and nothing to do: the spec is valid, the plan is empty, the work is certified, everything is current |
| 1 | Error: the command could not do its job, or the spec it validates is invalid |
| 2 | Done, and there is something to do: the plan has changes, the work is not yet certified, work is under way, or the binary or harness is out of date |

Two commands are exceptions, each for a reason of the provider's:

- `work start` always exits 0 and prints any failure on standard output. Claude Code aborts a skill whose injected command exits otherwise, and the owner's session would then see nothing.
- `hook` keeps the hook protocol, in which exit 2 blocks the call.

**CI runs `markitect plan`.** A certified commit has an empty plan and exits 0. A file that changed without a certificate makes the plan non-empty and exits 2. The plan depends neither on the work's state, which CI does not have, nor on the harness's version. `status` is for people: its 2 also means work under way or an outdated harness.

## The work as a status

The work under way is described the way kubectl describes an object: by its phase, with a reason code and a one-line message. The commands that change the work record the phase as they make the change. So `status` and `work list` read a recorded fact and never guess a step.

| Phase | Recorded by | Means | The default driver |
|---|---|---|---|
| `Building` | `work start`, `work gate` | The builder has work. That is the start, what a review sent back, or what `gate` refused: a changed spec or harness to put back, a file in no item, an error in the realization | Starts `markitect-builder` with "Build the work." |
| `Reviewing` | `work gate`, `work report` | Items have no answer for the files as they are, or a report waits for a reviewer | Starts `markitect-reviewer` with "Review the work." |
| `Reviewed` | `work answer` | Every item has an answer, and `gate` has not run on it yet | Runs `markitect work gate` |
| `NeedsOwner` | `work gate` | A spec problem confirmed by a reviewer, or a spec that does not compile under the work | Tells the owner, with the question |

`work start` on a spec that does not compile begins no work, so it gives `NeedsOwner` as its result and records nothing. `work list` exits like `plan`: 0 for an empty list, 2 with items.

`work gate` ends the work in one of two ways, and gives them as its result, since no work is left to record them in:

- `Certified`: the certificate is written, and the driver commits the work with it.
- `Closed`: a review of finished work is done, and the driver commits its Findings.

`work gate` and `work answer` with no work under way give `NeedsOwner`. They never point to `work start`, so an agent that lost the work cannot begin it again from a spec it changed.

**The tool names no agent.** Which agent serves which phase, and with what message, belongs to the driver. It lives in the skill, as four lines of configuration. Everything else that is prose for agents lives in the agent files:

- the builder's task;
- the reviewer's task and the answer format;
- what to do for each diagnostic code.

An agent's file is its system prompt, so it survives the compaction of that agent's context. `init` and `update` write the manifest's `model`, if it names one, into the agents' front matter, where today it travels in the printed step. `work start` refuses when the two differ and says to run `markitect update`, so a changed model does not go stale silently.

**Guarding the format.** The harness carries a format version. It changes only when phases, codes or the agents' rules change. `work start` refuses a harness of another format and says to run `markitect update`. A Go test checks that the embedded agent files cover every code the tool can emit.

**When `gate` sends the work back**, the phase is `Building`, whoever asks. The builder's file says that this means itself. Its tools let it start only the reviewer (`Agent(markitect-reviewer)`), so the rule is held in structure as well as in prose.

## Files

| Where | What | Like |
|---|---|---|
| `.markitect/` | The spec, the registry, the manifest, the lock, the ignore file | The configuration and `.terraform.lock.hcl` |
| `.markitect/results/` | Certificates and Findings as JSON records, each with a generated Markdown page beside it for people and pull requests. The tool never reads the page | The state file |
| The user's cache folder | The work under way, one file per project folder, as today | `.terraform/` |

`--findings FILE` takes a Findings record, or Findings written by hand in today's Markdown form ([model language](../guides/model-language.md#findings)). A refusal writes nothing into the project; its result says why.

The work under way stays in the cache. Moving it to `.markitect/work/` was weighed and rejected, for four reasons:

- That folder would be a spec error under the fixed layout ([`project.Load`](../../internal/project/project.go)).
- The hook would need a guard on Write and Edit.
- `update` would need a migration.
- `git clean -X` would wipe the work.

Every worktree already has its own state, since the cache is keyed by the folder's path.

A saved plan, as in `terraform plan -out`, was weighed and cut. Terraform needs one because plan and apply are separated in time and approval. Here apply follows in the same session. And the list grows as the builder works, so a saved plan could not promise to be done exactly. `plan` is the preview.

## No limit on rounds

`gate` no longer refuses after two repair rounds. The builder's reports of a spec problem are no longer capped at two that were turned down. The default driver sets no limit of its own: the work goes on until it is certified, needs the owner, or the owner stops it. The certificate records how many rounds it took.

Two rules about state, not counts, remain against loops that cost a review and change nothing:

- **Reports wait for a review.** After a reviewer turned a report down, the builder can report again only once a full review has finished.
- **Disagreement has no rule yet.** If a builder and a reviewer disagree without end, the owner stops them. Unattended work, the scheduled jobs of [concept 4.0 §12](v4-concept.md#12-schedules-and-continuous-work), counted on the two rounds as their budget, so that work brings a bound back. A candidate: an item sent back on two reviews in a row goes to the owner.

What goes: `repairRounds`, the refusal after the rounds, the cap on rejected reports, and the fields and checks that served them.

## What this leaves alone

What the tool checks stays as it is:

- `gate`'s rows, apart from the round limit;
- the certificate's content;
- how answers bind to files;
- the folder walk;
- how the plan reaches Definitions.

The hook on answers, the commit by the driver, and a tool without Git stay too. This plan changes the names, the output, the exit codes, and what the tool writes into the project.

## Decisions for the owner

Each has a recommendation.

1. **Terraform's vocabulary.** `validate` for the spec, `plan` for everything that differs, `status`, `describe`, and the agents' commands under `work`. The old names go with no aliases, with stubs for one release that say to run `markitect update`. Recommended.
2. **Output as in Terraform.** Text for people, `--json` everywhere with one envelope and a format version, and one shape of diagnostic. Recommended.
3. **Exit codes 0, 1 and 2** as above, by default, with `work start` and `hook` as the named exceptions. CI runs `markitect plan`. Recommended.
4. **The work as a phase**, recorded by the commands that change it, with `Certified` and `Closed` as `gate`'s results. The tool names no agent. The prose lives in the harness, with a format check and a code test. Recommended.
5. **Records as JSON with a generated page beside each**, Findings by hand still accepted, and no pages for refusals. Recommended.
6. **The work under way stays in the user's cache folder.** Recommended.
7. **No limit on repair rounds**, decided by the owner on 7 November 2026. **No cap on rejected reports**, and a report waits for a full review. **No rule for disagreement until unattended work needs one.** Recommended.

Once accepted, the decisions go into the register as one entry. The pages that describe them change in the same pull requests as the code:

- concept 4.0 §3, §5, §6, §7, §9 and §12;
- the [project workflow](../guides/project-workflow.md#7-apply-and-verify);
- the [model language](../guides/model-language.md#commands);
- the README;
- [the work](../../internal/run/README.md).

## Order of work

1. **Decide** the points above.
2. **Output and exit codes.** The JSON envelope, the diagnostic shape, the text renderings and the exit codes, for every command under its current name. One PR.
3. **The new command line.** One PR and a release, since the harness and the binary change together. It brings:
   - the renames and the stubs, with the plugin's hook strings;
   - `describe`, and `plan` taking over `check`'s realization errors and `gate --verify`;
   - the `work` group with its phases;
   - records with their pages;
   - the end of the round limit;
   - the prose moving into the harness, with its format version and the code test.
4. **Try it** on a small project with planted defects, under a bounded dispatch. The evaluation projects are gone from Main. The Makerspace of step 3 can be checked out from [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools/evaluation-tasks) for it.

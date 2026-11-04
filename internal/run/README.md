# The Command engine

`internal/run` runs one Command, `markitect apply` or `markitect verify` ([concept 3.0](../../docs/concepts/v3-concept.md) §6 to §10, §12). One implementer and one fresh reviewer work in one container; the Command commits their work, runs a gate no agent can talk past, checks the reviewer's answers, writes the Result and lands it. This page is the canonical description of a Command, its gate, its landing and its Result page. The delivery plan owns the [steps](../../docs/implementation-plan.md#concept-30-build).

| Package | Role |
| --- | --- |
| `internal/run` | The engine: checklist, gate, repair rounds, answer checks, Result and landing. It starts no process itself; its ports do. |
| `internal/adapters/sandbox` | The Command's one container ([below](#the-sandbox)). |
| `internal/adapters/rungit` | The Command's own Git work ([below](#host-git)). |
| `internal/agentport` | How the engine starts sessions; [`internal/agents`](../agents/README.md) implements it. |
| `internal/bootstrap` | Builds the engine for the command line from the [environment](../agents/README.md#composition). |

## A Command

1. **Start and base.** The Command starts from the tip of the target branch. The base is `--base`, or by default the work Markitect last landed there: the newest landing commit names it in its `Run-Work` trailer. A landing on a moved target merges in commits the Command did not realize, so the base is the realized work, and those commits stay in the next impact. With no landing yet, `--base` or `--full` is needed.
2. **Check.** The project at the start commit is compiled. A real compile error ends the Command as `check-failed` before any container starts. A file a hand edit left unlisted is no error here: it becomes a checklist item.
3. **Checklist.** Impact from the base to the start gives one list ([impact and checklists](../../docs/guides/model-language.md#impact-and-checklists)). Each item is a reached Definition, a deleted Definition, a Finding, a stale registry entry or an unlisted file, with the files the registry names for it. An empty list ends the Command as `nothing-to-do`.
4. **Sandbox.** The start commit goes into the Command's container as a Git bundle.
5. **Implementer.** One session in a clean copy of the start commit, with the checklist as its to-do list. It uses no Git and fills in no form. To stop, it writes a note outside the copy; the Command reads it after the session and ends as `spec-problem`, with the note as a decision for the owner. An empty diff without a note is no failure.
6. **Commit.** The Command ends the implementer's processes, commits the copy's files on top of the start and brings the commit to the host. Whatever Git does not take is not there. The checklist is rebuilt from this commit and its registry, so a file outside the first impact adds its Definitions.
7. **Gate** (below).
8. **Reviewer.** A fresh session in a new clean copy of the commit answers every item ([the answers](#the-reviewers-answers)). Nothing it changes counts: its copy is never collected.
9. **Repair.** A gate refusal or a `wrong` answer goes back to the implementer, who resumes its session; the Command then continues at step 6. At most two repair rounds; the third refusal ends the Command as `refused`. The next review answers only the items without an answer, the items still `wrong`, and the items whose locations or pointers the new diff touched, and is told what the last review found; the other answers stand.
10. **Landing** ([below](#landing)) and the Result.

`verify` skips steps 5 to 7 and 9: it runs the required checks once on the start commit, then one reviewer, or with `--full` one reviewer per Package, answers the list on a clean copy of that commit. It ends as `verified`, or as `findings` with a Finding for each `wrong`, the owner's decisions for each `spec problem`, or required checks that did not pass. Its Result stays on `markitect/runs/<run>`. `verify --findings-out FILE` writes the Findings as a Findings page, the input of `apply --findings`.

With `--full` every Definition is reached. The reviewers are one per Package, up to eight at a time, each in a clean copy of its own with only its own items and its own answer file.

## The gate

The gate runs after every commit of the implementer's work, with no agent. It refuses when:

- anything under `.markitect/` changed except the registry pages, so the spec, manifest, lock, `execution.yaml` and earlier Results stay as the owner left them;
- `check` reports a compile error, an unlisted file included;
- the registry holds a stale entry;
- a required check fails.

The three comparisons run on the host, on the commit itself, so nothing left in the container can affect them. The required checks run in the container, as a user of their own that is given no token, in a clean copy of the commit, in the order of `execution.yaml` and up to the first failure. The Command ends their processes afterwards and throws the copy away. A check that cannot start (no program, no service) is the environment's failure: the Command ends as `incomplete` and uses no repair round.

## The sandbox

`internal/adapters/sandbox` starts one container per Command from the project's image, natively on Windows, Linux and macOS with only the Docker CLI. It is read-only, drops every capability, forbids new privileges, carries limits for CPU, memory, processes and temporary space, and joins the provider's network. The Command sets the container up with two volumes, `/work` and `/home`, and removes both with it. Nothing of the user's is mounted. The container's own process is the tools' user.

Code enters and leaves only as a Git commit. The start commit arrives as a bundle into a bare repository that belongs to the tools' user. Each copy is made from that repository by the user it is for, with Git set not to convert line endings or filters, so a file's bytes stay what the commit holds. The implementer's work leaves as a tar stream of its copy, which the tools' user commits in a place of its own. The commit comes back to the host as a bundle.

The Linux users, made in the image ([`packaging/runtime/Dockerfile`](../../packaging/runtime/Dockerfile)), each with a home and a work folder only they can enter:

| User | Does | Gets the token |
| --- | --- | --- |
| `implementer` | The implementer's session | Yes |
| `reviewer-1` to `reviewer-8` | One reviewer's session each, side by side with `--full` | Yes |
| `checks` | The required checks | No |
| `tools` | The Command's Git work in the container | No |

The programs belong to root. Each user can end only its own processes. The Command ends a user's processes when a session is cut off, before it reads that user's note, before it commits the implementer's folder, and before and after the required checks. Before each new copy it clears that user's home, its folder and what it left in `/tmp` and `/dev/shm`, so nothing one session leaves reaches another.

## The reviewer's answers

The reviewer answers every item in one line, with pointers instead of free text:

```text
loans/limit: ok  src/loans.go:42, tests/loans_test.go:88 (TestLimit)
loans/renewal: wrong  src/loans.go:70 -- allows a fourth renewal; the Definition says three
```

The words are `ok`, `wrong`, `not shown` and `spec problem`; all but `ok` need an explanation after `--`. `ok` means the item conforms and the Rules above it hold, and needs a pointer, except for a deleted Definition's item. The reviewer reads the whole Definition, its examples and the Definitions it links to; where they contradict each other the answer is `spec problem` ([DEC-088](../../docs/concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)).

The Command checks that every item has exactly one answer, that every line is readable and names an item on the list, that every pointed file exists in the commit, that a line number lies inside it, that a named test occurs in that file, and that the session's transcript shows the file read. Where shell commands or subagents ran, or the provider's transcript shows no reads, a pointer is marked `not measured` instead of refused. A pointer the registry does not list for its item is reported as a possible missing link and refuses nothing. A pointer shows that someone looked, not that they judged correctly.

What the Command cannot accept goes back to the reviewer once, in the same session. If the answers are still not acceptable, the Command ends as `incomplete`.

## Landing

What may land: no `wrong` answer left, every item answered, every pointer accepted, the gate passed. `not shown` lands and the Result lists it. A `spec problem` from an implementer or a reviewer, or a third refusal, lands nothing.

Landing treats the user's repository only as a Git remote. First the work is pushed to `markitect/runs/<run>`, so nothing is lost if landing fails halfway. The work must not hold the token or touch a provider login file (`.credentials.json`, `auth.json`) anywhere in its history, since landing pushes every commit; otherwise it is `kept`.

- **The target has not moved.** The Result commit goes on top of the work and lands as a fast-forward.
- **It has moved.** It must still hold the start commit. The incoming changes must not touch the Command's impact: no changed or reached Definition, no file it changed or reached, no change to the manifest, the lock, the required checks or a registry page the work changed, and the target must still compile. The merge must be clean, and the gate runs on the merged state with the required checks. Then the Result commit goes on top of the merge. Otherwise the work and its Result stay on `markitect/runs/<run>` as `kept`, and the Result says why.

Every push names the tip it expects. If the target moves again, landing starts over, at most three times. A push that failed is checked by reading the target again: the Command counts as landed when the new tip holds its Result commit. There is no lock on the target branch: the rule above handles a branch that moved, and two Commands may work side by side.

A landing deletes its own Run branch and every kept Run branch it covers: one of the same target, whose tip is a Result commit that adds nothing but its page, whose start the landing's start holds and whose impact the landing's impact holds (or the landing was `--full`). The landed Result lists them. What the cleanup after landing leaves cannot be on the landed page; the command prints it on standard error and the JSON holds it as `afterLanding`.

Result commits carry trailers the Command reads back: `Run-Work` on a landing (the next default base), `Run-Full-Check` or `Run-Full: true`, and on a Run branch `Run-Name`, `Run-Target`, `Run-Start`, `Run-Command`, `Run-State` and one `Run-Impact` line per entry of the impact.

## Host Git

`internal/adapters/rungit` is the Command's own Git work outside the container. It works in a private bare repository: it reads the target branch, makes the start bundle, imports the bundle the sandbox hands out, compares commits, reads the files of a commit for `check`, merges with `git merge-tree`, writes the Result commit through a private index and pushes. The user's repository is only a Git remote to it, so Markitect never touches the user's working tree. The target is `--remote` and `--branch`: by default the current branch's upstream remote and branch, else `origin`, else the repository itself. The adapter never pushes to a branch checked out in a working tree of a local target, so with no remote a Command keeps its work on its Run branch. `apply` refuses to start when the spec has uncommitted changes, since a Command starts from a commit and would leave them out. Merges use Git's own text merge, whatever merge drivers attributes or configuration name.

## Limits and honest stops

`--deadline` bounds the whole Command (default 6 hours) and `--session-timeout` one session (3 hours). The deadline reaches every session and every Git and container operation. Removing the container and keeping a Result that does not land have their own two minutes after the deadline has passed. A session that ends as anything but finished ends the Command as `incomplete`, and the Result names the provider's failure class ([agents](../agents/README.md#failure-classes)).

## The Result

The Command writes the Result from what happened, never an agent. `markitect apply --json` prints it as JSON, the evaluation suite's Result contract; otherwise it prints the Result page, which is also committed under `.markitect/results/<run>.md`.

| State | Outcome | Means |
| --- | --- | --- |
| `landed` | landed | the work is on the target branch |
| `nothing-to-do` | not-landed | nothing affected; no agent started |
| `check-failed` | not-landed | the spec has a real error; no agent started |
| `verified`, `findings` | not-landed | `verify` found every item conforming and the required checks passing, or Findings, owner decisions or failing checks |
| `spec-problem` | not-landed | a session or reviewer found the spec contradicting itself or leaving open something that changes behaviour at the boundary |
| `refused` | not-landed | the repair rounds ran out |
| `kept` | not-landed | the work passed but cannot land; it stays on its Run branch |
| `incomplete` | not-landed | a deadline, a provider failure, a check that could not run, answers the Command could not accept or a technical failure |

The page holds, in this order: the state and why; the command, outcome, target, base, start, work commit, Run branch, repair rounds, times, tokens (the sum over all sessions, "at least" when a session reported no total) and the full-check count; the owner's decisions with recommendations; the checklist with each item's latest answer, pointers and explanation; how much checking really happened (files opened, commands run, subagents started, pointers shown opened or not measured, answers that repeat another's words), measured by the Command from the transcripts; what could not be tested, from `not shown` answers and the reviewer's closing words; the required checks, with the commit they ran on and each result; Findings; possible missing links; side observations from impact; replaced Run branches; every session with its role, round, end, agent and seconds; and what cleanup left.

**The full-check count.** Impact does not follow the code's own imports, so a full check now and then catches what it missed. Every Result says how many changes have landed on the target since the last full check, or since the first landing, and from ten on that a full check is due: `markitect verify --full`. The Command starts nothing by itself. A full check is a `verify --full` that ended `verified` or `findings`, which counts from the commit it examined, and a landed `apply --full`. The count is read from the target: the landing commits and their trailers, and the Result commits on Run branches. Each landing carries the newest full check it knows in `Run-Full-Check`, so a reset survives the deletion of a verify's Run branch. When a verify's branch is deleted before a landing carried it forward, the count errs toward due. When the count cannot be read, the Result says why.

## Tests

`run_test.go` runs whole Commands with scripted sessions and a scripted sandbox over the host's real Git: landing on an unmoved and a moved target, every gate refusal, repair rounds, the answer checks, a stop by the implementer, a spec problem, `verify`, `--full`, the full-check count and the deletion of covered Run branches. They are source evidence, not provider, container or native evidence.

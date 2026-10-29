# Design topics

State page of the Topics session: the owner's single place to discuss Markitect's design topics. They are platform, model language, modeling skills, compiler, agentic commands, continuous change and the Cockpit. On 28 October 2026 the owner replaced the seven topic lanes (5 to 11) with this one session, because "The topics touch too strongly to be discussed independently." The Build and evaluation session is the only place that builds; it owns the evaluation suite and the variants.

| | |
|---|---|
| **How it works** | The owner and this session discuss one question at a time. Each decision is recorded in plain words with its date, on its own page where it has one, otherwise under [Decided](#decided). When a decision needs code, it goes to the Build session as one bounded task (see [Handed to Build](#handed-to-build)). This session writes documentation only. |
| **Canonical pages** | The [commands page](commands.md) owns the commands frame, the agreed names and answers A1–A6. Accepted decisions go to the [concept register](../concepts/register.md); proposals live under [design](../design/README.md); evidence under [validation](../validation/README.md). This page links to them and does not repeat them. |
| **Folded lane pages** | Full-SHA permalinks of the lane pages as they were when folded: [platform](https://github.com/Glacius-Labs/Markitect/blob/b212a2f338f253b533f64a61a04cc9d03db065f2/docs/workstreams/platform.md) (lane 5), [model language](https://github.com/Glacius-Labs/Markitect/blob/df1f255fe2a88658973e128dc831806f977d0dd3/docs/workstreams/model-language.md) (lane 6), [modeling skills](https://github.com/Glacius-Labs/Markitect/blob/b18f562c42efaab0ce4646ec235f0024bd3243d3/docs/workstreams/modeling-skills.md) (lane 7), [compiler](https://github.com/Glacius-Labs/Markitect/blob/cedd21c106e6f9991ab42501470d55bc74dd813d/docs/workstreams/compiler.md) (lane 8), [continuous change](https://github.com/Glacius-Labs/Markitect/blob/9e5d9c6b864c3c2fc16ddc1fb91e5e363a55949a/docs/workstreams/continuous-change.md) (lane 10), [Cockpit](https://github.com/Glacius-Labs/Markitect/blob/604a421bacf93eb507935f6196014d39b2e3f1a6/docs/workstreams/cockpit.md) (lane 11). |

## Now

- **Concept 2.0 is the target** (accepted 28 October, [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)). The design walk-through ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)) settled DEC-079 to DEC-085 and the ten review questions; [concept 2.0](../concepts/v2-concept.md) puts them in one order with its acceptance. Handed to the Build and evaluation session on 28 October (see [Handed to Build](#handed-to-build)); being built in the [delivery plan's steps](../implementation-plan.md#concept-20-build).
- **Basis:** clean. One topics page instead of seven lane pages; Main consistent again after #806 (see [Done here](#done-here)); the topic lane worktrees and branches removed (the old Cockpit lane worktree now serves the General session); question 33 (Windows tests) parked.

## Next

The design walk-through of 28 October, one part at a time: what was open and a recommendation.

1. **Closed** 28 October: one Go tool in the CLI ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)).
2. **Closed** 28 October: the model tools start no agents ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
3. **Closed** 28 October: the lint knows no special Kinds; it measures the link graph (DEC-080).
4. **Closed** 28 October: compile errors are structure, cycles, visibility and accounting; everything else is a warning (DEC-080).
5. **Closed** 28 October: language and compiler rules change only by owner decision, as DEC-080 defines them.
6. **Closed** 28 October: no Cockpit app ([DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)).
7. **Closed** 28 October: the Cockpit questions 30–32, as part of DEC-085.

## Topics in short

The state since [concept 2.0](../concepts/v2-concept.md); what came before is in Git history.

### Platform

The host runs natively on Windows, Linux and macOS with Docker only; each Run has its own remote container with the hooks and the required checks ([DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)). Observed natively on Windows and on Linux (WSL Ubuntu with Docker Desktop) in T5; macOS not yet. The Linux runner is retired. Before the rebuild, on 28 October, the Go tests took about 50 minutes on Windows ([measurement](../validation/windows-test-time-20261028.md)); they are to be measured again after it (question 33, parked).

### Model language and compiler

Markdown Definitions with front matter; Core is a single Kind, and Libraries supply common Kinds ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation), [model language](../guides/model-language.md)). `check` checks structure only ([DEC-003](../concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept)); `init`, `lock`, `refs`, `graph`, `impact` and `lint` are in the product. The old language and V1's Libraries are replaced.

### Modeling skills

The modeling skill proposes model changes in conversation and asks first when something is open; Extract is its own skill on top of it; `markitect init` installs them ([concept §8](../concepts/v2-concept.md#8-authoring)).

### Agentic commands

`apply` and `verify` ([concept §7](../concepts/v2-concept.md#7-commands-that-start-agents)) are built; the other commands on the [commands page](commands.md) come after acceptance.

### Continuous change

Design only ([DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps)): triggers become Findings or model proposals, and autonomy grows in owner-approved steps.

### Cockpit

No Cockpit app for now; Markdown tools are the cockpit ([DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)), which also decides questions 30–32.

## Decided

What is decided and lives nowhere else; decisions with their own page are linked.

- **Platform** (27–28 October): no running from WSL and no Linux machine required; our studies run on our own machine; Linux Apply is tested only after Windows Apply works. The Windows Git-start fix is parked, and the Go tests are not run on Windows for now ([measurement](../validation/windows-test-time-20261028.md)).
- **Libraries** (27–28 October, [PR #806](https://github.com/Glacius-Labs/Markitect/pull/806)): refusal lists have no order; an Interface exposes refusal codes; an Invariant lives with its data and is not placed; no wire-envelope Kind and no alias-only imports yet ([standard Libraries](../design/standard-libraries.md#answered-and-open-questions)). Today's Libraries on Main; replaced by the redesign below once built.
- **Language** (28 October; designed, not built; [DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)):
  - The model is as explicit as good documentation ([DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation)).
  - Core offers one thing: a Definition with a Kind and a body. A Kind is a Definition of Kind `Kind`. No typed fields.
  - A Definition is a Markdown file with YAML front matter (Kind, and `internal` where needed); the file name is the ID, the main heading the title. No imports: links name their target by path (`[[members/member]]`, `[[std/Rule]]`) and may target Packages. Scope is location.
  - A Library is a set of Kind Definitions for one area; no Kind has a special meaning to Markitect. Libraries are rebuilt as 1.0.0. Libraries are cut by technical area, like a programming language's packages, not by method: `std` for every project (Project, Rule, Concept, Feature, Decision), `rest` (Endpoint) and `cli` (Command); further areas such as events or UI only when a real project needs them. Error codes and roles are Concepts.
  - Visibility as in Go: front matter `internal: true` makes a Definition visible only to its own Package and below; the compiler refuses links to it from elsewhere, and other Packages' Managers do not receive it.
  - Go's cycle rule: every link between Packages is an edge, the implicit child-to-parent edge too; a cycle is a compile error, so links point up or sideways, never down.
  - The model is measured like code from its links (coupling, wrong-way dependencies, size, dead Definitions, reach of a change); warnings only.
  - The design is settled first; the evaluation and the implementations are then built against it, while the owner also redesigns the Runs.
- **Organization** (28 October; designed, not built; [DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)):
  - What a Manager knows: its own Package's Definitions, everything its parents define, and the public Definitions it links to in other Packages. Of its children it sees a table of contents of their public Definitions (title and first sentence), with full text on request, for example to review their work. Broad at the top, deep below.
  - One Package, one Manager, always, also for a pure grouping Package; the lint warns about a Package with no Definitions of its own and only one child.
  - The registry stays: the per-file record of which Definitions each repository file realizes and why, maintained by the owning Manager and checked by the compiler. It catches silently added files, explains every file and gives exact impact. It becomes Markdown: one page per Package, a table of files with links to the Definitions they realize and a short reason. It stays separate from the model; Apply never changes the model. Ownership moves onto that page as path patterns, several per Package (`src/billing/**`, `tests/billing/**`, `docs/billing*.md`); a shared file such as the README or the CI pipeline is owned by the lowest common parent and links the Definitions of every Package it realizes; layout conventions are Rules.
  - `execution.yaml` keeps only the required checks (name, command, directory, timeout, environment); they are the one proof in a Run that does not come from an agent.
- **Runs** (28 October; direction, details open; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)): consistency and quality first, cost second. Impact decides who works, like an incremental build, with required checks on the whole repository and an occasional full audit Run. One Run per change; each Manager is a maker and a fresh checker; every Definition in scope gets a verdict with evidence, enforced by the host. Parents review and integrate; Root's checker is the final check. Work moves only as Git commits through the Run's remote; messages carry only assignments, questions and escalations. Each Manager has its own harness with local hooks as help; the guarantees are server-side Git hooks on the remote (own branch and folders, required checks, a verdict for every Definition, written as commit trailers); no separate referee. Every Run ends with one host-written Result page: outcome, verdict and evidence per Definition, what did not take part, owner decisions with recommendations, side observations. Landing as in a merge queue: a result lands only if it was checked in the state it lands in; otherwise a follow-up Run starts from the new state. Commands to start: `apply` (impact by default, `--full` as audit and repair) and `verify` (checkers only); `explain` and custom Commands later. More Commands as declarations over the same mechanism: `improve`, `bughunt`, `security`, `cleanup`. One Finding input for every source (`apply --findings FILE`): each Finding names a Definition and becomes a checklist item; a report about something the model does not state is a model proposal.
- **Platform** (28 October; designed, not built; [DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)): the host runs natively on Windows, Linux and macOS with Docker only; each Run has a Linux container as its remote with hooks and checks; Managers clone and push over the container network; no mounts; the Linux runner retires; a registry entry for a Git-ignored file is refused.
- **Authoring** (28 October): the owner accepts every model change; agents propose them through the modeling skill, which asks first when something is open. Extract becomes its own skill that uses the modeling skill. Later goal: model changes from external triggers should not each need the owner's review (continuous change).
- **Continuous change** (28 October; direction, design only; [DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps)): intake rewrites triggers into tasks; triage makes a Finding or a model proposal; the loop runs outside Markitect; autonomy in steps (step 1: repairs automatic, model proposals in a daily inbox; step 2: an owner mandate lets agents accept model changes, with a daily briefing); budgets, undo, circuit breaker; first slice after the rebuild.
- **Compiler and analysis** (28 October): one tool, in the product, in Go, in the `markitect` CLI: `check` (errors), `lock`, `refs`, `graph`, `impact --base <commit>`, `lint` (warnings). No Python; the tool under `tools/model-analysis/` is retired. This answers question 1 in direction.
- **Modeling skills** (27 October): the change-task trial first; it ran as C5 and passed ([record](../validation/modeling-skill-change-trial-20261027.md)).
- **Commands** (27–28 October): the frame, the names and A1–A6 ([commands page](commands.md)).
- **Cockpit** (28 October; [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)): no Cockpit app for now; Markitect writes everything as Markdown, so Markdown tools are the cockpit; the model explorer stays a proposal; the personal agent stays parked.
- **Organization of this work** (28 October): one Topics session for all design topics and one Build and evaluation session; the topic lanes are folded into this page. The owner restated what Markitect is ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)).
- **Build conformance** (29 October; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)): an independent check found T1 to T4a and #856 true to concept 2.0 except one gap, landing without compiling, and four points for the owner. Decided: landing compiles every submission and the state it lands; `verify` runs the required checks; the checker sees the model, the repository at the submission and its diff, but not the Shared Notes; impact does not follow the code's own imports; on planted defects a result equal to its start counts as not landed, with the same instruction for every arm. A lock so that overlapping Runs take turns, and deleting old Run branches, go to T5. After the first native Run (Library M1, 19 of 19): a parent with own items submits its own files first and its children start from that approved commit, on every level, enforced by the host; the required checks on every submission stay unchanged.
- **Full check and human attention** (29 October; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)), after an independent assessment of Main by Codex: the full check is `verify --full`, at every milestone the owner names and once ten changes have landed since the last one, and every Result says when it is due; the host starts nothing itself. The counted series claims only what it measures (at least the baseline's quality, fewer defects left for a person to repair at comparable input); time saved is measured apart afterwards, by the owner taking three changes per arm to acceptance, blinded, and recording minutes and corrections. The same assessment's two Result bugs went to Build, and the entry docs were brought to concept 2.0 here.
- **First path patterns** (29 October; [DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)): Build's T7 arm smoke showed that a new Package owns nothing in its first Run, since the hooks take paths from the Run's base. Decided: when the owner adds a Package, the owner also writes its first path patterns, just as its Definitions; the Manager maintains the page from then on and may narrow or extend its patterns, moving files between Packages through their common parent; changes take effect only from the next Run; `lint` warns about a Package other than Root that claims no path pattern.
- **Codex runs** (29 October; [build page](build.md#decisions)): with Codex usage available again, the counted series stays on Claude Sonnet 5.5 as decided on 28 October, so it stays comparable with the pilot and the suite's self-tests. Codex adds a native concept 2.0 smoke on the Library M1 now, beside the Claude pilot, and after the counted series a smaller repeat with every arm on Codex Luna. Within one series every arm uses the same model; Sol High is kept for a later strong-model check if wanted.
- **Records** (28 October): decisions are recorded in plain words with their date, without verbatim quotes or attributions ([record rule 1](../concepts/README.md#adding-and-changing-entries)).

## Questions for the owner

**[B]** marks a question whose answer is needed before building work can start. Each question carries its recommendation.

### A. Analysis layer

1. **Closed** 28 October: one Go tool in the CLI ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)).
2. **Closed** 28 October: the model tools start no agents ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
3. **Closed** 28 October: the lint knows no special Kinds; it measures the link graph (DEC-080).
4. **Closed** 28 October: compile errors are structure, cycles, visibility and accounting; everything else is a warning (DEC-080).
5. **Closed** 28 October: language and compiler rules change only by owner decision, as DEC-080 defines them.
6. **Closed** 28 October: no Cockpit app ([DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)).
7. **Closed** 28 October: the standard Libraries ship inside the binary; others are committed under `.markitect/libraries/`; no `--dependency` flags (DEC-080).

### B. Language

8. **Small language gaps: closed** by [DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation): good documentation does not state maps, unions, defaults or typed formats, so the model does not either.
9. **Closed** 28 October: the delegated decisions concerned the old Libraries and the Python tool, which the rebuild replaces; the drafts branch `handover/model-quality-log` is deleted.
10. **Closed** 28 October: risk is a Rule in the Package, in prose; what agents may decide alone is part of the mandate ([DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps)).
11. **Closed** 28 October: "do it the way code does" is how the redesign was made.

### C. Authoring

12. **Closed** 28 October: the skill is rewritten after the rebuild and trialled then.
13. **Closed** 28 October: the skill asks first when something is open (Authoring).
14. **Extract: decided** 28 October as its own skill that uses the modeling skill; DEC-061's round-2 reading confirmed the same day.
15. **Closed** 28 October: the skills are built into the binary, and `markitect init` sets up a project and installs or updates them (DEC-080).
16. **Who accepts model changes: decided** 28 October: the owner accepts every model change; later, changes from triggers should not each need review (see [Decided](#decided)).
17. **Closed** 28 October: consistency and quality are the main measure; cost comes second (DEC-082).

### D. Commands and the Result

18. **Which Commands first: decided** 28 October: `apply` and `verify` ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
19. **Closed** 28 October: replaced by maker and checker per Manager (DEC-082).
20. **The Result's form: decided** 28 October as one host-written Result page per Run ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
21. **Closed** 28 October: replaced by impact (DEC-082).
22. **Closed** 28 October: replaced by Command declarations with a focus (DEC-082).
23. **One channel for outside Findings: decided** 28 October (DEC-082): one Finding input for every source.
24. **Closed** 28 October: the rebuild uses the new names from the start.

### E. Continuous change

25. **Outside Findings amendment: replaced** 28 October by the one Finding input of DEC-082.
26. **Decided** 28 October as part of [DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps).
27. **Decided** 28 October as part of [DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps).
28. **Decided** 28 October as part of [DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps).
29. **Decided** 28 October as part of [DEC-084](../concepts/register.md#dec-084-continuous-change-triggers-become-findings-or-model-proposals-autonomy-grows-in-steps).

### F. Cockpit

30. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now).
31. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now).
32. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now).

### G. Platform

33. **Windows Git-start waste: parked** (owner, 28 October). Not fixed now; the Go tests are not run on Windows for now, and the cause may be settled by the design discussion. The [test-time record](../validation/windows-test-time-20261028.md) keeps the numbers and the four fix options. With [DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers) most of that Git machinery is expected to disappear; judge again after the rebuild.
34. **Git variant consequences: decided** 28 October: host native, each Run in containers ([DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)).
35. **Silent missing inputs: decided** 28 October: the compiler refuses a registry entry for a Git-ignored file ([DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)).

### H. Added when folding

36. **Modeling guide Kind names.** Done on 28 October: the guide v0.3 names the Library Kinds.
37. **Closed** 28 October: see question 4.

### K. Before building concept 2.0

41. **Ten questions before building: all decided** 28 October and folded into the [concept 2.0 draft](../concepts/v2-concept.md), including its acceptance (§15).

### J. Library areas

40. **Which Library areas does Markitect ship?** Decided 28 October: `std`, `rest` and `cli` (see [Decided](#decided)).

### I. Raised by the documentation test (DEC-079)

38. **Use cases.** Folded into question 40: a use case Kind belongs to whichever Library area covers it.
39. **How an interface is reached** (REST routes, CLI commands). Folded into question 40.

## Handed to Build

- **28 October 2026: build concept 2.0.** On the owner's go-ahead, one message to the Build and evaluation session: build the evaluation suite and the implementations against [concept 2.0](../concepts/v2-concept.md) and its acceptance (§15), and write the steps into the [delivery plan](../implementation-plan.md). Consistency and quality first, cost second. Noted for the G0 prototype: server-side hooks instead of a referee, maker and checker per Manager, verdicts as commit trailers.
- **29 October 2026: conformance fixes.** On the owner's go-ahead, one message to Build: compile every submission and the landed state; required checks in `verify`; no Shared Notes for the checker; the target lock and Run-branch cleanup into T5; the planted-defect scoring accepted ([Decided](#decided)). Then, after the first native Run, one message: parents submit their own files before their children start. Then one message on Codex runs: a native Codex smoke now, a smaller Luna repeat after the counted series, which stays on Sonnet 5.5 ([Decided](#decided)). Then one message answering Build's question on first path patterns: the owner writes them, changes take effect from the next Run, and `lint` warns about a Package other than Root without one. Then the assessment's two Result bugs (a landed Run reported as not landed; cleanup errors missing from the stored Result), and one message with the full-check rule and the counted series' claim.

### Done here

On the owner's instruction of 28 October this session made Main consistent after #806 itself, instead of handing it to Build:
- [Modeling guide](../design/modeling-guide.md) v0.3: the Library Kind names, refusals without order, Interfaces with refusal codes, an Invariant with its data.
- The modeling skill: `api` 2.0.0 and `architecture` 1.1.0 in its reference and examples; refusals without order (where precedence matters, a rule or the operation's body states it); Interfaces list refusal codes; an Invariant lives with its Entities.
- The analysis tool: the `interface` signal checks refusal codes against `refusalCodes` (a note only for an architecture Library before 1.1.0); a new `invariant-owner` signal.
- `makerspace-v2/check.py` accepts its frozen 1.0.0 Library copies again, with a note.
- Delivery plan rows W1–W4; register lines under DEC-011, DEC-074, DEC-075 and DEC-076; the commands page points its open questions here.

### Ready to hand over

Waiting for one owner answer, then one task each: `markitect lock` and readable `check` output (1); the two skill fixes and the C7 or C2 trial (12); explorer E1 (1 and 30), which also needs the source packager's allowlist to carry the embedded page.

### Housekeeping

- Other open PRs that touch these topics: #795 (G0 design, answers question 34) and #803 (record of the owner's evening decisions of 27 October).

### Contracts the variants and the evaluation suite follow now

- One **Result** type for every Run and Command (A4, A5); the evaluation suite and the Cockpit read it, not per-variant records.
- **Commands as declarations** from a small fixed vocabulary (A6), the same across variants.
- The **agreed names** in new code and docs; old names stay in the concept until renamed (question 24).
- **`markitect tree` JSON** is pinned by `TestTreeReportsTheModelDerivedManagerTree`; the plugin variant depends on it.
- **Libraries:** `api` 2.0.0 and `architecture` 1.1.0 are current.
- **Platform rules:** no WSL, no Linux machine required; tests must be able to run on Windows too, but for now the Go tests are not run there because they take too long (question 33); studies run on our own machine; Linux Apply is tested only after Windows Apply works. Decided for the rebuild: the host runs natively everywhere with Docker only ([DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers)).

### Risks for evaluation runs

- P8: Git-ignored files the model accounts are silently absent in the runner, so its baseline differs from a native run.
- P6: long runs on the shared Windows machine get killed (reaped shells, stray `pkill`, other sessions' load).
- The runner is qualified for six Managers only.

### Measurements wanted from evaluation

The cost of a model change against a throwaway spec (17); the cost of the full tree for read-only Commands (21); Verify rounds after a Reject (19); later, revert and incident rates of landed changes, which decide autonomy steps.

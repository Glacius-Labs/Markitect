# Design topics

State page of the Topics session: the owner's single place to discuss Markitect's design topics. They are platform, model language, modeling skills, compiler, agentic commands and continuous change; the Cockpit moved to the [Desktop session](https://github.com/Glacius-Labs/Markitect/blob/e295881f1f716b4fb067bd4bb88e959d6a74f44d/docs/workstreams/desktop.md) on 31 October 2026. On 28 October 2026 the owner replaced the seven topic lanes (5 to 11) with this one session, because "The topics touch too strongly to be discussed independently." The Build and evaluation session builds everything except Markitect Desktop's own scope; it owns the evaluation suite and the variants.

| | |
|---|---|
| **How it works** | The owner and this session discuss one question at a time. Each decision is recorded in plain words with its date, on its own page where it has one, otherwise under [Decided](#decided). When a decision needs code, it goes to the Build session as one bounded task (see [Handed to Build](#handed-to-build)). This session writes documentation only. |
| **Canonical pages** | The [commands page](commands.md) owns the commands frame, the agreed names and answers A1–A6. Accepted decisions go to the [concept register](../concepts/register.md); proposals live under [design](../design/README.md); evidence under [validation](../validation/README.md). This page links to them and does not repeat them. |
| **Folded lane pages** | Full-SHA permalinks of the lane pages as they were when folded: [platform](https://github.com/Glacius-Labs/Markitect/blob/b212a2f338f253b533f64a61a04cc9d03db065f2/docs/workstreams/platform.md) (lane 5), [model language](https://github.com/Glacius-Labs/Markitect/blob/df1f255fe2a88658973e128dc831806f977d0dd3/docs/workstreams/model-language.md) (lane 6), [modeling skills](https://github.com/Glacius-Labs/Markitect/blob/b18f562c42efaab0ce4646ec235f0024bd3243d3/docs/workstreams/modeling-skills.md) (lane 7), [compiler](https://github.com/Glacius-Labs/Markitect/blob/cedd21c106e6f9991ab42501470d55bc74dd813d/docs/workstreams/compiler.md) (lane 8), [continuous change](https://github.com/Glacius-Labs/Markitect/blob/9e5d9c6b864c3c2fc16ddc1fb91e5e363a55949a/docs/workstreams/continuous-change.md) (lane 10), [Cockpit](https://github.com/Glacius-Labs/Markitect/blob/604a421bacf93eb507935f6196014d39b2e3f1a6/docs/workstreams/cockpit.md) (lane 11). |

## Now

- **Concept 4.0 is the target** (accepted 7 November, [DEC-098](../concepts/register.md#dec-098-concept-40-is-the-target)); the [delivery plan](../implementation-plan.md) owns its steps. Concept 3.0 was the target from 4 November ([DEC-091](../concepts/register.md#dec-091-concept-30-is-the-target)), after a step-back discussion in a session of its own. The entries below describe the way to concept 2.0 and are a record.
- **Concept 2.0 was the target** (accepted 28 October, [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)). The design walk-through ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)) settled DEC-079 to DEC-085 and the ten review questions; [concept 2.0](../concepts/v2-concept.md) puts them in one order with its acceptance. Handed to the Build and evaluation session on 28 October (see [Handed to Build](#handed-to-build)); being built in the [delivery plan's steps](../implementation-plan.md#concept-20-build).
- **Basis:** clean. One topics page instead of seven lane pages; Main consistent again after #806 (see [Done here](#done-here)); the topic lane worktrees and branches removed (the old Cockpit lane worktree now serves the General session); question 33 (Windows tests) parked.

## Next

The design walk-through of 28 October, one part at a time: what was open and a recommendation.

1. **Closed** 28 October: one Go tool in the CLI ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)).
2. **Closed** 28 October: the model tools start no agents ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
3. **Closed** 28 October: the lint knows no special Kinds; it measures the link graph (DEC-080).
4. **Closed** 28 October: compile errors are structure, cycles, visibility and accounting; everything else is a warning (DEC-080).
5. **Closed** 28 October: language and compiler rules change only by owner decision, as DEC-080 defines them.
6. **Closed** 28 October: no Cockpit app ([DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)); superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).
7. **Closed** 28 October: the Cockpit questions 30–32, as part of DEC-085; superseded on 31 October by DEC-089.

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

Markitect Desktop, the app over the CLI that users know as Markitect ([DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli)), replaced [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now) on 31 October 2026. The [Desktop session](https://github.com/Glacius-Labs/Markitect/blob/e295881f1f716b4fb067bd4bb88e959d6a74f44d/docs/workstreams/desktop.md) designs and builds it.

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
- **Cockpit** (28 October; [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)): no Cockpit app for now; Markitect writes everything as Markdown, so Markdown tools are the cockpit; the model explorer stays a proposal; the personal agent stays parked. Superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).
- **Organization of this work** (28 October): one Topics session for all design topics and one Build and evaluation session; the topic lanes are folded into this page. The owner restated what Markitect is ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)).
- **Build conformance** (29 October; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)): an independent check found T1 to T4a and #856 true to concept 2.0 except one gap, landing without compiling, and four points for the owner. Decided: landing compiles every submission and the state it lands; `verify` runs the required checks; the checker sees the model, the repository at the submission and its diff, but not the Shared Notes; impact does not follow the code's own imports; on planted defects a result equal to its start counts as not landed, with the same instruction for every arm. A lock so that overlapping Runs take turns, and deleting old Run branches, go to T5. After the first native Run (Library M1, 19 of 19): a parent with own items submits its own files first and its children start from that approved commit, on every level, enforced by the host; the required checks on every submission stay unchanged (changed later the same day, below).
- **Required checks gate merges** (29 October; [DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)), on Build's question after the T7 pilot, where Root's first submission on the Makerspace could not pass the build check because the build needs a child's files: the required checks block only merged states, a parent's submission that holds its children's merged work and the state that lands. Any other submission is still blocked by `markitect check`, path ownership and its checker, which sees the check results as evidence. A child's broken build is caught at its parent's merge. Build builds it after the pilot.
- **What a checker confirms** (30 October; [DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)), on two questions from Build: in the T7 pilot's c3 plant, the bulk-discount table said 8 % and its worked example 5 %, and the `materials` checker approved code that matched the table; in the [Codex Library smoke](../validation/codex-library-smoke-20261030.md), `lending`'s checker rejected its package because Root's file did not call it yet, while Root would call it only after `lending` was approved. One rule answers both: a verdict covers the model in full, the whole Definition with its examples and the Definitions it links to, where a contradiction is a model problem, and the code of its own Package only. Evidence for conforms or changed shows how each example is met (changed added the same day, on Build's question from T9, since changed also claims the item conforms after the change). Whether the rest of the program uses a Package's work is the parent's integration, judged by the parent's checker and gated by the required checks on the merged state; a parent does not build a child's behavior a second time in its own files. The pilot's c3 result stays a stop failure.
- **Reading the counted series** (30 October; [protocol](../validation/counted-series-protocol-20261030.md#reading)), after the owner asked whether the build is on track, built as concept 2.0 designs, and whether the suite is suitable, and two independent reviews of `main` at `2c9008262` answered. The build is on track and true to the concept, with two small host gaps. The suite can show that the 2.0 arm is not worse than K on the unseen project, not yet the long-chain drift or the saved attention the concept promises. Its primary share dilutes: with K perfect and M not landing three of its seven fixed starts, such as C2, C5 and C6, the share still reads at least as good. Decided: M is at least as good only if all three are non-inferior: the share (0.05, still primary), the failed cases (1 case) and the landed-change rate (not below K's); the share of each change's own new cases is reported as a secondary line. Build writes it into the protocol before the gate.
- **Long-lived Managers** (30 October; [proposal](../design/long-lived-managers.md)): the owner found starting a fresh agent for every small step wasteful. The pilot showed that containers cost 1 to 3 seconds per start, while about ten turns ran one after the other and each re-read 80 to 370 k tokens. Decided: Build builds one maker and one checker per working Manager for the whole Run, with plain integration by the host, on its own branch, and compares it with `main` on quality first and cost second; the owner decides on adoption after the comparison. The owner: "Can we redesign it to that in a separate branch and then compare?"
- **Simpler Apply, tried in a small pilot** (2 November 2026): after reading how the public runs of the counted Windows series behaved, the owner asked to try simpler ways for Markitect to apply a change. Concept 2.0 does not change yet; whether it does waits for the pilot's reading. Two new evaluation arms use one small graph: impact, make, a gate run by the host, a fresh check, landing, with a limited loop from a rejection back to make.
  - **Arm S (one agent):** one make session with the whole checklist and generated context.
  - **Arm F (federalism as prompts):** the host writes one prompt per working Package, runs them in parallel in one folder, then a Root prompt integrates.
  - **In both:** no agent runs Git, the agents may use subagents, and the host keeps the gate, the loop limit, all Git and landing. The checker is a fresh session the host starts, never a subagent of the maker.
  - **Docker:** the evaluation keeps it. In these arms Markitect starts no container itself; the suite's one sandbox per launch wraps the arm. Docker for the product stays open.
  - **Pilot:** kept small, to show how complex each approach turns out and which is worth pursuing. Ten launches, S then F, not counted, new project sealed: Makerspace chain W0 and C1; Library plants `library-m2-limit-contradiction` and `library-m2-transition-gap`; Makerspace C2 on the old start. Existing M and K results are read beside them, with no new M or K launches. Complexity is reported first.
  - **If adopted,** this would change [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review), [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-083](../concepts/register.md#dec-083-the-host-runs-natively-each-run-lives-in-containers) and [CPT-001](../concepts/register.md#cpt-001-federalism).
- **After the pilot's first reading** (2 and 3 November 2026; evidence on [draft PR #960](https://github.com/Glacius-Labs/Markitect/pull/960)): F's checkers stopped its fresh build over missing evidence and read one rule in opposite ways, while S's checker let that rule through. The owner approved a changed checker for both arms: only wrong work refuses, work not shown and waivers land as recorded findings, and a ruling stands for the Run. In the second set every case came out right, and no checker recorded any work as not shown. The owner then approved two more tests, neither counted. A checker reviews a landed submission with planted bugs, because none has yet been tested on wrong work. S runs on the Makerspace plants c3, c5 and c6 and the fixed start C9, because the pilot's five cases hold none that K gets wrong: K went wrong on c3 and C9 in the series, and c5 and c6 complete the stops and gaps. Concept 2.0 still does not change.
- **Reporting runs** (3 November 2026): the owner found the prices the CLI states meaningless, since runs use the subscription. Results now go to the owner in weekly quota and time, with multiples of K for comparisons.
- **A bigger evaluation project** (3 November 2026): the owner added a project of about five times Makerspace, about 50 modules, 185 operations and 540 checklist items, to test each approach under more complexity. On Makerspace's fresh build the largest session of any version used about 15 to 22 % of its context window, so nothing has yet tested a project too big for one agent.
  - **Recommended with the size, which the owner saw when deciding it:** rules across modules, changes that reach many of them and a chain of changes, not copied modules. Many hidden cases, because a session that fills up compacts instead of stopping, so failure shows as lost cases. K and then S on the fresh build first; F and M only if one of those strains.
  - **Growing Makerspace** (3 November 2026): shown Build's estimate, the owner chose to grow Makerspace rather than start a new domain: its modules stay and about 40 are added around them. Build builds it. The default, unless Build or the owner sees a reason against it, is public and not counted. The sealed project stays out of it.
- **After the checker test** (3 November 2026): the owner approved a small test of a cheaper version, K plus impact, the gate and a checker, expected at about 1.5 times K's fresh build. No approach is chosen yet, and concept 2.0 is unchanged.
  - **The reading behind it, not the owner's:** the checker is able, but not yet shown to be needed. It caught all six planted bugs in all six reviews, the new answers raised no false alarm on clean work, and a review costs about half of K's fresh build. S's real work has not yet failed a hidden case on Makerspace, and Build chose the planted bugs, none across modules. On K's failures, impact started no agent where K changed code and failed 3 of 126, and S stopped twice on the contradiction where K stopped once and landed once. S showed no clear gain on the other two plants. S's cost is mostly its maker: on the fresh build 1.8 to 3.5 times K's, against about half of K's for its checker. On the contradiction and the gap the maker found the model problem and the checker confirmed it, though on one gap launch the checker also refused the maker's verdicts on items the gap does not touch.
  - **The brief to Build:** K's maker with its prompt unchanged, then one fresh checker. It has to show whether the checker catches the planted bugs without the maker's item-by-item claims, whether it stops the contradiction in K's run that landed, the one real wrong work the series offers for this, and whether it lets good work through. Build plans about 1 weekly point.
- **The results behind the step back** (3 November 2026): not a decision, and the readings here are this session's, not the owner's. These are the last recorded results before the owner stepped back to define the problems. Multiples are of K's fresh build on the same project. Weekly points in these records are the runner's seven-day reading, which does not convert to the owner's usage card.
  - **K plus impact, the gate and a fresh checker** ([record](https://github.com/Glacius-Labs/Markitect/blob/42ebd0bee6830f4a0bcdea5a5f7c8df5d721ad48/docs/validation/k-checker-test-20261103.md)): it stopped the contradiction on both c3 launches, and 2 of 3 reviews named it on the one that K really landed. It caught 17 of 18 planted bugs and raised no false alarm, and its fresh build cost about 1.5 times K's.
  - **The project of about five times Makerspace** ([record](https://github.com/Glacius-Labs/Markitect/blob/42ebd0bee6830f4a0bcdea5a5f7c8df5d721ad48/docs/validation/large-probe-20261103.md)): K and S both landed with no hidden case failed, S at 4.76 times K's cost. K's run plus a thorough blind review adds up to about 1.5 times K. This is one launch of each and a first build only. The first launch on Haiku ran mostly on Opus and says nothing about Haiku, and later ones are not recorded yet.
  - **What a review's answers show:** no review at this size found a fault in behaviour. Nothing was planted and no hidden case failed, so none has met a known bug there. One review read 10 of K's Go files and gave almost the same answers as two that read 63, and S's checker wrote one evidence text on all 413 items, so the answers don't show how much checking stands behind them. The draft's §8 and §10 take this up.
  - **Impact's reach** ([record](../validation/impact-reach-20261103.md)): of the 36 existing files that truly changed on Makerspace, one step reached 33 and every link reached 35, but every link took in at least 85 % of the spec on four of the eight changes. One reference and one big package.
- **The basic shape** (4 November 2026): the owner's step back left two shapes for one idea: Markitect as a tool with swappable strategies, one agent working and a fresh one reviewing, no Manager tree and no agent Git. This session proposed a skill in the owner's own session with a Stop hook, and containers only around evaluations. [Draft concept 3.0](../concepts/v3-concept.md) has a Command start the sessions and run the gate after they end, with one container per Command. Shown the difference, the owner kept the draft's shape. The reason put to them: a gate in the Command can't be skipped, as a hook can, and it can run unattended. This did not yet accept the draft; the owner accepted it later that day ([DEC-091](../concepts/register.md#dec-091-concept-30-is-the-target)). The skill and hook stay only as a possible fallback if the login route closes ([concept 3.0 §14](../concepts/v3-concept.md#14-decided-on-acceptance), question 10, decided the same day in [DEC-092](../concepts/register.md#dec-092-a-command-uses-the-users-own-sign-in)). No test will compare the two shapes, because the owner settled on 4 November to stop comparing variants ([draft §13](../concepts/v3-concept.md#13-how-we-know-it-works)).
- **Full check and human attention** (29 October; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager), [DEC-086](../concepts/register.md#dec-086-concept-20-is-the-target)), after an independent assessment of Main by Codex: the full check is `verify --full`, at every milestone the owner names and once ten changes have landed since the last one, and every Result says when it is due; the host starts nothing itself. The counted series claims only what it measures (at least the baseline's quality, fewer defects left for a person to repair at comparable input); time saved is measured apart afterwards, by the owner taking three changes per arm to acceptance, blinded, and recording minutes and corrections. The same assessment's two Result bugs went to Build, and the entry docs were brought to concept 2.0 here.
- **First path patterns** (29 October; [DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)): Build's T7 arm smoke showed that a new Package owns nothing in its first Run, since the hooks take paths from the Run's base. Decided: when the owner adds a Package, the owner also writes its first path patterns, just as its Definitions; the Manager maintains the page from then on and may narrow or extend its patterns, moving files between Packages through their common parent; changes take effect only from the next Run; `lint` warns about a Package other than Root that claims no path pattern. On 30 October, on Build's question: a page that exists with `paths: []` is the owner's first patterns too, so its Manager may extend it; only a page that does not exist yet is the owner's to create.
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
6. **Closed** 28 October: no Cockpit app ([DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now)); superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).
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

30. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now); superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).
31. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now); superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).
32. **Decided** 28 October as part of [DEC-085](../concepts/register.md#dec-085-markdown-tools-are-the-cockpit-for-now); superseded on 31 October by [DEC-089](../concepts/register.md#dec-089-markitect-desktop-the-app-over-the-cli).

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
- **29 October 2026: conformance fixes.** On the owner's go-ahead, one message to Build: compile every submission and the landed state; required checks in `verify`; no Shared Notes for the checker; the target lock and Run-branch cleanup into T5; the planted-defect scoring accepted ([Decided](#decided)). Then, after the first native Run, one message: parents submit their own files before their children start. Then one message on Codex runs: a native Codex smoke now, a smaller Luna repeat after the counted series, which stays on Sonnet 5.5 ([Decided](#decided)). Then one message answering Build's question on first path patterns: the owner writes them, changes take effect from the next Run, and `lint` warns about a Package other than Root without one. Then the assessment's two Result bugs (a landed Run reported as not landed; cleanup errors missing from the stored Result), and one message with the full-check rule and the counted series' claim. Build's own question from the pilot, required checks on merged states only, was answered by the owner in Build's session and recorded here ([DEC-087](../concepts/register.md#dec-087-required-checks-gate-merged-states-not-each-submission)).
- **30 October 2026: what a checker confirms.** Build's two checker questions, from the pilot's c3 plant and the Codex smoke, answered by the owner with one rule ([DEC-088](../concepts/register.md#dec-088-a-checker-reads-the-model-in-full-and-judges-its-own-packages-files)); one message to Build with the link. Then Build's question from T9, answered by the owner: the example evidence holds for changed as for conforms.
- **30 October 2026: reading the counted series and two host gaps.** On the owner's go-ahead, one message to Build: the joint reading rule into the protocol before the gate ([Decided](#decided)); `verify` refuses a `changed` verdict; a Run refuses a Manager creating its own Package's first registry page, which the owner writes ([DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)). Added as a note, not the owner's: the T9 row's claim that the verdict trailer changed. Then Build's question from those host fixes, answered by the owner: a Manager may extend its existing page with `paths: []` ([DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)).
- **30 October 2026: long-lived Managers.** On the owner's request, one message to Build: build the [proposal](../design/long-lived-managers.md) on its own branch and compare it with `main` as the page says, without touching the counted series' frozen commit or its Docker windows.
- **2 November 2026: simpler Apply pilot.** The pilot of arms S and F went to the Build and evaluation session as one bounded task, as recorded in [Decided](#decided) (Simpler Apply, tried in a small pilot).
- **3 November 2026: the pilot's follow-ups and a bigger project.** Separate messages to Build, as recorded in [Decided](#decided): the planted-bug test, S on the Makerspace plants and C9, reporting in quota and time, and the project of about five times Makerspace, which Build estimates before building. Then, on the owner's go-ahead, one message to Build: the test of K's maker with impact, the gate and a checker ([Decided](#decided), After the checker test). The owner answered Build's estimate directly in Build's session: grow Makerspace.

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

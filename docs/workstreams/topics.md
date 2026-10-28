# Design topics

State page of the Topics session: the owner's single place to discuss Markitect's design topics. They are platform, model language, modeling skills, compiler, agentic commands, continuous change and the Cockpit. On 28 October 2026 the owner replaced the seven topic lanes (5 to 11) with this one session, because "The topics touch too strongly to be discussed independently." The Build and evaluation session is the only place that builds; it owns the evaluation suite and the variants.

| | |
|---|---|
| **How it works** | The owner and this session discuss one question at a time. Each decision is recorded in plain words with its date, on its own page where it has one, otherwise under [Decided](#decided). When a decision needs code, it goes to the Build session as one bounded task (see [Handed to Build](#handed-to-build)). This session writes documentation only. |
| **Canonical pages** | The [commands page](commands.md) owns the commands frame, the agreed names and answers A1–A6. Accepted decisions go to the [concept register](../concepts/register.md); proposals live under [design](../design/README.md); evidence under [validation](../validation/README.md). This page links to them and does not repeat them. |
| **Folded lane pages** | Full-SHA permalinks of the lane pages as they were when folded: [platform](https://github.com/Glacius-Labs/Markitect/blob/b212a2f338f253b533f64a61a04cc9d03db065f2/docs/workstreams/platform.md) (lane 5), [model language](https://github.com/Glacius-Labs/Markitect/blob/df1f255fe2a88658973e128dc831806f977d0dd3/docs/workstreams/model-language.md) (lane 6), [modeling skills](https://github.com/Glacius-Labs/Markitect/blob/b18f562c42efaab0ce4646ec235f0024bd3243d3/docs/workstreams/modeling-skills.md) (lane 7), [compiler](https://github.com/Glacius-Labs/Markitect/blob/cedd21c106e6f9991ab42501470d55bc74dd813d/docs/workstreams/compiler.md) (lane 8), [continuous change](https://github.com/Glacius-Labs/Markitect/blob/9e5d9c6b864c3c2fc16ddc1fb91e5e363a55949a/docs/workstreams/continuous-change.md) (lane 10), [Cockpit](https://github.com/Glacius-Labs/Markitect/blob/604a421bacf93eb507935f6196014d39b2e3f1a6/docs/workstreams/cockpit.md) (lane 11). |

## Now

- **Design walk-through, part 1: the language.** On 28 October the owner restated what Markitect is ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)) and started a walk through the whole design. Decided so far: the model is as explicit as good documentation; Definitions, bodies and links with few typed fields, as Markdown with YAML front matter; Libraries `domain`, `application`, `api`, `practice`, `architecture` (see [Decided](#decided)). Then no typed fields at all, Core as a single `Kind`, scope by location, no Kind with a special meaning, Go's cycle rule and code-like metrics ([DEC-080](../concepts/register.md#dec-080-the-model-is-typed-documentation)). Then the Libraries `std`, `rest`, `cli`, visibility, and no imports. Then the organization ([DEC-081](../concepts/register.md#dec-081-the-organization-knows-broad-at-the-top-and-deep-below)). Then the Run direction ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)). Next: the Run details.
- **Basis:** clean. One topics page instead of seven lane pages; Main consistent again after #806 (see [Done here](#done-here)); the topic lane worktrees and branches removed (the old Cockpit lane worktree now serves the General session); question 33 (Windows tests) parked.

## Next

The design walk-through, one part at a time, as the owner asked on 28 October. Each part says how it works today, what is open and a recommendation.

1. **The language** (questions 38, 39, 10, 11, 9): use cases and interface bindings, risk per Package, the Library design check, the delegated decisions.
2. **The compiler and the analysis layer** (1–7, 37): one Go analysis core in the product; where Analysis ends and a Run starts; what counts as an error.
3. **The Manager organization:** how the hierarchy is derived from the model and what each Manager knows.
4. **Runs, Commands and the Result** (18–24): order of Commands, convergence, the decision page, renaming.
5. **Publication and the Git variant** (34, 35, and the parked 33).
6. **Change over time:** Authoring (12–17), outside Findings (23, 25) and continuous change (26–29).
7. **The Cockpit** (30–32).

## Topics in short

### Platform

Execution is Linux-first ([DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution)). On Windows, Apply runs only through the [Linux runner](../design/linux-runner.md) on Docker Desktop, observed once with six Managers ([record](../validation/v1-l1-runner-qualification-20261027.md)). An Apply with the host on Linux has never run. The Go tests take about 50 minutes on Windows against 3 on Linux, almost all of it Git process starts ([record](../validation/windows-test-time-20261028.md)). Known runner problems P1–P9 are on the folded platform page. The Git variant ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review), PR #795) may remove the mounts and the shared Run volume.

### Model language

The model is typed programming in prose: Core 1.2.0 with Kind interfaces ([DEC-075](../concepts/register.md#dec-075-kind-extension-and-one-core-120)) and five [standard Libraries](../design/standard-libraries.md). Since PR #806, `api` is 2.0.0 (refusals are an unordered list; a Rule states precedence when it matters) and `architecture` is 1.1.0 (an Interface can list `refusalCodes`). Invariants are not placed; an Invariant lives in the Package that owns its data. `makerspace-v2` keeps its frozen 1.0.0 copies as gate material. That is Main today. A redesign of the language and Libraries is decided and not built (see [Decided](#decided)).

### Modeling skills

The [modeling skill](../../packaging/skills/markitect-modeling/SKILL.md), the [modeling guide](../design/modeling-guide.md) and the [analysis tool](../../tools/model-analysis/README.md). A fresh agent with only the skill wrote a new model twice (PR #771) and changed an existing one faithfully once ([C5 trial](../validation/modeling-skill-change-trial-20261027.md)). Untested: renames (C7), architecture changes (C2), review and blocker repair. No everyday change workflow is written, and the skill reaches no user. The skill, the guide (v0.3) and the tool follow `api` 2.0.0 and `architecture` 1.1.0; the tool's tests run in CI. Since the owner said "We will be responsible for all of that", model language and modeling skills are one topic.

### Compiler

`markitect check` checks structure only, never meaning ([DEC-003](../concepts/register.md#dec-003-the-compiler-is-a-borrowed-concept)). `markitect tree` is on Main (PR #802). Gaps: a project cannot finish a model with the product alone (the lock and readable diagnostics need the Python tool, which ships nowhere); the dependency graph is computed twice, in Go and in Python; impact works offline only; spans are partial. Proposed features, smallest first: `lock`, readable `check` output, a Go analysis core, `lint`, dependency discovery, exact spans.

### Agentic commands

The frame and names (Analysis, Gate, Required checks, Run, Run kind, Command, Workflow, Result, Authoring) and answers A1–A6 are on the [commands page](commands.md). The owner's last request there, "Lets talk about those too. Give me the questions and your recommendations", is answered by questions 2 and 18–24 below.

### Continuous change

Triggers (issues, incidents, dependency updates, requests) become tasks; triage sends each to a repair (the model already states the behavior) or to a model change. Autonomy grows in owner-approved steps 0 → 1 → 2 (Government). The [outside Findings](../design/outside-findings.md) amendment is designed and not adopted. The rest of the sketch (trigger sources, safety and limits, undo, circuit breaker) is on the folded continuous-change page. The commission was design only; the [DEC-011](../concepts/register.md#dec-011-detailed-planning-of-future-ideas-waits) planning hold is not lifted for this axis.

### Cockpit

A desktop UI for understanding the model and its runs, outside V1. The old cockpit was removed on 23 October and survives on the archive branch. The first slice is a read-only [model explorer](../design/model-explorer.md) (`markitect explore`, slices E1–E5), designed and reviewed, not approved. Run views need records a UI can read: today the Run record is written at the end, with no event stream, and on Windows it sits in the Docker volume.

### Links between the topics

- **Model analysis** is one question raised in four places. Every topic lane recommended one Go implementation in the product, used by the skill, the explorer and the impact Command, and retiring the Python tool (question 1).
- **Compiler, lint and Libraries.** The compiler stays Library-blind; lint may know the standard Libraries and only warns (questions 3, 4).
- **Outside Findings.** `apply --findings RESULT` (A5) and continuous change's outside reports are the same channel (question 23).
- **One Result** (A4) feeds the Cockpit's run views, the evaluation suite and all variants; the decision page and the "proof package" are one question (questions 20, 31).
- **Risk per Package** joins the model-language idea, protected Packages in continuous change and deeper review for risky areas (question 10).
- **Extract** is Authoring, not a Command (question 14).

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
  - The registry stays: the per-file record of which Definitions each repository file realizes and why, maintained by the owning Manager and checked by the compiler. It catches silently added files, explains every file and gives exact impact. It becomes Markdown: one page per Package, a table of files with links to the Definitions they realize and a short reason. It stays separate from the model; Apply never changes the model. Ownership (which folders a Package owns) moves onto that page.
  - `execution.yaml` keeps only the required checks (name, command, directory, timeout, environment); they are the one proof in a Run that does not come from an agent.
- **Runs** (28 October; direction, details open; [DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)): consistency and quality first, cost second. Impact decides who works, like an incremental build, with required checks on the whole repository and an occasional full audit Run. One Run per change; each Manager is a maker and a fresh checker; every Definition in scope gets a verdict with evidence, enforced by the host. Parents review and integrate; Root's checker is the final check. Work moves only as Git commits through the Run's remote; messages carry only assignments, questions and escalations. Each Manager has its own harness with local hooks as help; the guarantees are server-side Git hooks on the remote (own branch and folders, required checks, a verdict for every Definition, written as commit trailers); no separate referee. Every Run ends with one host-written Result page: outcome, verdict and evidence per Definition, what did not take part, owner decisions with recommendations, side observations. Landing as in a merge queue: a result lands only if it was checked in the state it lands in; otherwise a follow-up Run starts from the new state.
- **Compiler and analysis** (28 October): one tool, in the product, in Go, in the `markitect` CLI: `check` (errors), `lock`, `refs`, `graph`, `impact --base <commit>`, `lint` (warnings). No Python; the tool under `tools/model-analysis/` is retired. This answers question 1 in direction.
- **Modeling skills** (27 October): the change-task trial first; it ran as C5 and passed ([record](../validation/modeling-skill-change-trial-20261027.md)).
- **Commands** (27–28 October): the frame, the names and A1–A6 ([commands page](commands.md)).
- **Continuous change and Cockpit** (27 October): outside Findings and the read-only model explorer are designed as proposals; nothing is built.
- **Organization of this work** (28 October): one Topics session for all design topics and one Build and evaluation session; the topic lanes are folded into this page. The owner restated what Markitect is ([source](../concepts/sources/model-explicitness-20261028.md#1-what-markitect-is-restated)).
- **Records** (28 October): decisions are recorded in plain words with their date, without verbatim quotes or attributions ([record rule 1](../concepts/README.md#adding-and-changing-entries)).

## Questions for the owner

**[B]** marks a question whose answer is needed before building work can start. Each question carries its recommendation.

### A. Analysis layer

1. **[B] Model analysis in the product, and who owns it?** Rec: yes. First `markitect lock` and a readable `check` output (small, no compiler change); then one Go analysis package (edges, graph, refs, impact with `--base REV`, lint); then retire the Python tool. Owned here, with the skill as its first user. Unblocks the explorer, impact and shipping the skill.
2. **[B] Where does Analysis stop and a Run start?** Rec: Analysis never calls an agent; Runs get Analysis facts as input. Impact = Analysis lists what is linked (a model diff), and an optional Run judges what else is affected.
3. **May product lint know the standard Libraries?** Rec: yes, warnings only; the compiler stays Library-blind.
4. **Should any lint signal become a compiler error?** Rec: only through a language change, for example a value type for Package references.
5. **What needs an explicit owner decision?** Rec: `internal/model` semantics and diagnostics do; read-only commands and the analysis package follow normal PR rules.
6. **A partial model when the check fails?** Rec: start without it.
7. **Dependency discovery** from the lock instead of repeated `--dependency`. Needs an answer on what the lock and manifest mean. Not urgent.

### B. Language

8. **Small language gaps: closed** by [DEC-079](../concepts/register.md#dec-079-the-model-is-as-explicit-as-good-documentation): good documentation does not state maps, unions, defaults or typed formats, so the model does not either.
9. **Decisions taken under delegation** (log on branch `handover/model-quality-log`). To discuss: 9 (v2.1 Interface exposes `rate-cents-per-minute`), 4 and 10 (the analysis tool reads the model's own architecture; the `interface` signal checks typed uses only). To confirm: 1, 3, 5, 6, 7 (the skill) and 2 (Claude adapter timing). Decision 8 was replaced by the refusal decision.
10. **Risk per Package.** A risk level, a "what failure looks like" statement or a protected flag on a Package? Rec: a Library Kind, not Core; with a deterministic hold after Verify when protected files change, and a circuit breaker.
11. **"Do it the way code does"** as a standing check for Library design? It removed two of three proposed changes.

### C. Authoring

12. **[B] Next modeling step after C5.** Rec: two skill fixes (a size limit signals a missing concept, not a reason to split; an example belongs with its rule), then one harder trial, C7 (renames) or C2 (architecture), then write the everyday change workflow from both.
13. **Interview first?** Rec: yes for new models and architecture; no for change requests that leave nothing open.
14. **Extract as a mode of the skill** that proposes a model and stops for the owner; and confirm [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) round 2 (the binding part is the contract callers rely on; incidental wording and identifier formats are free). Rec: confirm both.
15. **Shipping the skill** with the plugin variant, the CLI, or both? Rec: decide with the plugin variant; depends on question 1.
16. **Who accepts model changes, and how?** Rec: the owner accepts all Authoring; proposals from triggers arrive as a daily batch. "No owner review" (DEC-078) stays code-only.
17. **Main measure for modeling:** a model change must cost less than a throwaway spec. Rec: adopt. C5 cost about 144k tokens, with no baseline yet.

### D. Commands and the Result

18. **[B] Which Commands first?** Rec: `verify`; then the declaration mechanism (A6) with `apply` and `verify` rewritten as declarations; then impact (needs 1 and 2); then explain. All after the variants are stable.
19. **Verify convergence after a Reject.** Rec: keep the fresh full check; add the earlier Findings as input with a status per Finding; new Findings in unchanged, approved areas become "pre-existing" Notes; a round cap in the declaration.
20. **The Result's form: decided** 28 October as one host-written Result page per Run ([DEC-082](../concepts/register.md#dec-082-one-run-per-change-impact-decides-who-works-maker-and-checker-per-manager)).
21. **Cost of the full tree for read-only Commands.** Rec: wait for evaluation data, then decide whether it counts as new evidence.
22. **Project-defined Run kinds, the details:** task text, answer format, Gates, allowed verdicts, whether it may change artifacts.
23. **[B] One channel for outside Findings.** Rec: `apply --findings RESULT` and outside reports are one bound-input type, checked by the initial Gate and confirmed by Assessment.
24. **When to rename** mandate/Implement/Verify/check to the agreed names in concept, register and code? Rec: in one concept pass after V1 acceptance; new code uses the new names now.

### E. Continuous change

25. **[B] Adopt the outside Findings amendment?** Its open points: one model blocker ends the whole batch (accept it, retry the rest); no host rule for a reproducing test; timing. Rec: adopt after V1 acceptance, as one register entry.
26. **Automatic spending:** lift "only on owner prompts" ([DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration), [DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply)) for the loop, with separate budgets for triage, proposals and Apply?
27. **First slice, place and order.** Rec: step 1 for repairs only, one pilot project, GitHub issues as the source; the loop runs as an outer operator in CI or a scheduler, never a daemon in Core. Open: wait for unattended landing and intake, or prototype with a branch hand-back?
28. **Undo default.** Rec: for a repair, revert the landing; for a model change, revert the model and its realization together.
29. **Government.** Rec: leave it until step 1 has evidence.

### F. Cockpit

30. **[B] Approve the model explorer design and decide when to build.** Sub-points: the four areas, with lint shown inside Map and Read; the name `explore` until run views join; a local server in `markitect` and a plain page, no build chain; the Build session writes the code (Codex's single-writer role ended with the old cockpit); read-only first. Depends on question 1.
31. **Run views read only the Result.** Largely settled by A4; confirm. Records on Windows sit in the Docker volume, and there is no live event stream.
32. **Personal agent in the Cockpit** ([IDEA-015](../concepts/register.md#idea-015-cockpit-with-a-conversational-personal-agent)) stays parked unless the owner says otherwise.

### G. Platform

33. **Windows Git-start waste: parked** (owner, 28 October). Not fixed now; the Go tests are not run on Windows for now, and the cause may be settled by the design discussion. The [test-time record](../validation/windows-test-time-20261028.md) keeps the numbers and the four fix options.
34. **Git variant consequences** (needs the G0 answers, PR #795): can the host run natively on Windows again, with Docker Desktop only for agents? Does the shared Run volume go away? Is the local remote a container, a host bare repository or a volume?
35. **Silent missing inputs in the runner** (P8): a Git-ignored file the model accounts is silently absent through the runner. Refuse it instead?

### H. Added when folding

36. **Modeling guide Kind names.** Done on 28 October: the guide v0.3 names the Library Kinds.
37. **What counts as an error lives in prose.** The skill tells agents to treat `unknown-package`, `dependency-rule`, `interface`, `import-cycle` and every strong lint finding as errors; no tool contract says so, and lint thresholds are provisional ([OQ-002](../concepts/register.md#open-questions)). No recommendation yet; it belongs with questions 3 and 4.

### J. Library areas

40. **Which Library areas does Markitect ship?** Decided 28 October: `std`, `rest` and `cli` (see [Decided](#decided)).

### I. Raised by the documentation test (DEC-079)

38. **Use cases.** Folded into question 40: a use case Kind belongs to whichever Library area covers it.
39. **How an interface is reached** (REST routes, CLI commands). Folded into question 40.

## Handed to Build

Nothing yet.

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

- Branch `handover/model-quality-log` holds the unreviewed drafts behind question 9. Keep it until question 9 is reviewed, then delete it.
- Other open PRs that touch these topics: #795 (G0 design, answers question 34) and #803 (record of the owner's evening decisions of 27 October).

### Contracts the variants and the evaluation suite follow now

- One **Result** type for every Run and Command (A4, A5); the evaluation suite and the Cockpit read it, not per-variant records.
- **Commands as declarations** from a small fixed vocabulary (A6), the same across variants.
- The **agreed names** in new code and docs; old names stay in the concept until renamed (question 24).
- **`markitect tree` JSON** is pinned by `TestTreeReportsTheModelDerivedManagerTree`; the plugin variant depends on it.
- **Libraries:** `api` 2.0.0 and `architecture` 1.1.0 are current.
- **Platform rules:** no WSL, no Linux machine required; tests must be able to run on Windows too, but for now the Go tests are not run there because they take too long (question 33); studies run on our own machine; Linux Apply is tested only after Windows Apply works.

### Risks for evaluation runs

- P8: Git-ignored files the model accounts are silently absent in the runner, so its baseline differs from a native run.
- P6: long runs on the shared Windows machine get killed (reaped shells, stray `pkill`, other sessions' load).
- The runner is qualified for six Managers only.

### Measurements wanted from evaluation

The cost of a model change against a throwaway spec (17); the cost of the full tree for read-only Commands (21); Verify rounds after a Reject (19); later, revert and incident rates of landed changes, which decide autonomy steps.

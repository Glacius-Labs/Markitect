# Roadmap

Where Main is, what comes next and what may come later. The [concept](concept.md) is the target; the [decisions](decisions.md) are what holds.

## Where Main is

The latest release, 0.25.3, carries the whole concept on Claude Code: the spec language with `check`, `tidy`, `vet`, `list` and `doc`; the record with `status` and `mark`; the eight skills and four agents whose rights a hook holds; the strategies with the shipped parallel workflow and `harness.yaml`; and releases with the Claude Code plugin and `markitect update`. All of it is covered by tests, and small real runs have passed ([concept, §13](concept.md#13-what-is-shown-and-what-is-not)).

## Current steps

| Step | Result | Done when |
|---|---|---|
| 1. Clean up | The documentation shows one current state; then the code and architecture, and the prompts and mechanisms, are made as simple as the concept allows | Each part is reviewed and merged |
| 2. Try it | Markitect is used on a real project, an existing one drawn into a spec, and on a small project with planted defects, including a weakened test that served as evidence and a builder's edit of the spec. For the parallel strategy, a fixture of a few Packages is built in parallel, with one planted bug and one planted contradiction | The planted defects are caught or their misses are explained, and what the runs showed is in their PRs |
| 3. Make it reliable | The 5× Makerspace project's eight change steps on Sonnet, with their hidden cases, which are kept at [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools). Failures are fixed in the tool contract, not with more prompt text ([DEC-051](decisions.md#dec-051-agents-get-tasks-not-process)) | Two clean passes in a row: every step ends with nothing left to verify without the owner, every hidden case passes, and no refusal of the hook was overridden. Whether that is the finish line is open ([OQ-014](decisions.md#open-questions)) |

Before step 3, read the nine finished launches of an earlier reviewing test at the 5× size. They are in `%LOCALAPPDATA%\markitect-test1` on the maintainer's machine, not in the repository, and unread; they may say at no new cost whether one reviewer catches known bugs at that size.

## Later

Agreed directions that are not scheduled.

- **The rest of the rename.** The code still says "model" where it means the spec, as in the package `internal/model` and diagnostic codes such as `model.name`; it becomes "spec", with no compatibility kept.
- **A routine that audits the spec.** Cleanup, bug hunt and a full review run as routines, and the audit of the spec is started by hand. A scheduled audit reads the spec regularly and tells the owner what is contradictory, missing, unclear or duplicated.
- **More skills and checks.** `security` and `improve` skills, and a CI check with `markitect status` that ships with Markitect.
- **Observed code dependencies widen what `status` lists.** The spec's links do not contain the code's own calls, imports and shared types. Optional, language-specific tools could observe them and widen the list, never narrow it ([OQ-001](decisions.md#open-questions)). Until then, `/markitect-review --all` is the safety net.
- **More spec-quality signals.** `vet`'s zone and risk measures once real records exist; files that realize unusually many Definitions; and the size of a spec change against the code change it causes. Like `vet`, findings and recommendations, never gates.
- **Tests bound to Definitions.** Review whether a test really verifies its Definition, and report Definitions whose only evidence is an unreviewed test. The reviewer's case counts on `mark` are a first step.
- **Models and perspectives per role.** Mixing models, such as one building and another reviewing, as an experimental factor ([OQ-004](decisions.md#open-questions)), and teams per perspective, such as security or architecture, whose roles differ in question, context, tools and model.
- **Another provider,** once Claude's version is reliable ([DEC-094](decisions.md#dec-094-the-users-own-session-runs-the-agents-natively)).

## Ideas

Recorded, not planned: detailed planning waits until a usable, stable version exists ([DEC-084](decisions.md#dec-084-continuous-work-runs-outside-the-tool-and-autonomy-grows-in-steps)).

- **A cockpit with a personal AI,** and a project organization in which people architect, set rules and develop ideas while agents keep the project realized and tidy; a conversational agent that searches the project, proposes spec changes and shows what matters.
- **Government:** agents maintain the spec within explicit mandates, with people briefed and asked only about what matters; departments such as security or architecture raise objections on spec changes. It must keep the spec's quality high.
- **The project seen from every side:** a project is something different for each of its stakeholders, and no single role holds every view, so a description of a whole project may be the sum of its views rather than one objective definition. As a tomograph turns around a body, agents would go around the project one perspective at a time, such as architecture, security, operation, schedule and benefit, and the views together give the picture. It may answer what a spec must contain to describe a project, and give people their overview.
- **Views of the spec:** reports, visualizations or dashboards that show a spec's structure and what a change did to it, so that a person keeps the overview without reading every Definition.
- **The spec as institutional memory:** agents keep neither memory nor responsibility over time, and the spec and its record supply both.
- **Priorities and strictness:** Rules with a strictness and work with a degree of freedom; a project's ranking of its non-functional properties; local priorities at every level of the spec, with the inherited ones visible.
- **Precedents and instances:** reasoned decisions kept as precedents that later cases cite; disagreements with a severity and higher instances to settle them; an agent that brings a case it cannot settle to a higher instance while it works.
- **Briefings and visibility:** regular short briefings of what happened and what waits; the same evidence-bound events feeding briefings, history, observability and a view of the project's responsibilities, activity, decisions and health; gamification that makes verified progress visible without false incentives.
- **Operation:** monitoring, logs and data collection for more autonomous operation, with Kubernetes or etcd once a concrete need is stated; logs of a running system checked against the spec, as a kind of `doctor`; an optional offline classifier of claims and evidence, never a replacement for `check` or the reviewer.

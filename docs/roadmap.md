# Roadmap

Where Main is, what comes next and what waits for a trigger. The [concept](concept/README.md) is the target; the [decisions](decisions.md) are what holds.

## Where Main is

The decisions, the concept, the vision and the foundations describe the target, and the tool and its library are built for it. The library inside the tool ([library](guides/library.md)) holds the packages, among them `factory`, and the templates `project`, `product`, `architecture` and `factory` beside `software` and `company`. The `factory` template holds a general way of working for any project with one person: one role, the Admin, by default, the same course in every Area, and the general observers, among them `status`. The fourteen skills of the standard, each named `markitect-` and its agent's name, are built from its Agent documents, and `init` lays them out in `.claude/skills/` and `.agents/skills/` with an entry in `AGENTS.md`. The tool checks the spec and the documents of the work, lists with `status [PATH]...`, binds with `confirm` and moves with `move`, none of which refuses for what `check` finds; nothing that steers is left. The plugin only fetches the tool: its launcher fetches the plugin's own version, and `markitect version` says whether that differs from the version a project pins ([plugin](guides/plugin.md)). The guides describe the tool as built and follow these pages where they differ. No release of the tool on Main exists: the latest release and the marketplace's plugin are still 0.34.0 of an earlier concept, kept on an archive branch ([history](history.md)), and the tool on Main is built from source ([releasing](releasing.md)). No skill has run on a committed test project; that waits for the product owner's word.

## Current steps

A failure in how agents work is fixed in the `factory` part of the spec and the skill built from it, not with prompt text beside them ([DEC-147](decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)).

| Step | Result | Done when |
|---|---|---|
| 1. A finding around the circle | On a committed test project set up with `init`, one finding goes around the circle: a report, triage, planning with the Architect, build, review with `confirm`, merge and acceptance, with the skills of the standard as built | The story is on the main branch with its spec change and its realization, `status` shows nothing for it, and every report lies in the context; run only with the product owner's word |
| 2. Extraction | The Architect applied to an existing project: it asks whether the choice of templates fits; a subagent per part of the spec finds its own context and fills the part, with a report per part in `context/architect/`; what is unclear or contradictory becomes a finding and stays open; the person sees the parts one after another and decides in conversation ([concept](concept/the-circle.md#starting-a-project)) | The spec of an existing project is extracted part by part this way, and the person has decided every part |
| 3. The general runs | Cleanup, bug hunt and improve, which behave like observers that may also act, with their report and checkpoint in `context/<run>/`; and a run that maps an existing project's files to its spec, after which `status` joins the daily run | Each run works on a committed test project and leaves its report; after the mapping run, `status` on that project shows only what came past the spec |
| 4. Counter-checks, control samples and spot checks | The counter-checker and the sampler, as runs, and spot checks of confirmed documents ([trust](concept/trust.md)) | A control sample gives a hit rate, and counter-checks fail where the artifacts differ from the spec |
| 5. The blast radius | What a story's spec change reaches, computed by the tool with `compare` and attached to the story | The blast radius of a story names the documents and artifacts its change reaches |
| 6. The lab test | The 5× Makerspace project's eight change steps, with their hidden cases, which are kept at [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools) ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)) | Two clean passes in a row: every step reaches the main branch with its spec change and its realization together, confirmed, and every hidden case passes |
| 7. The proof | A project run by people who do not develop Markitect, in the circle with real work in CI | Four weeks in which the four figures of [DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven) hold: work, convergence, checking and no rescue |

## Conduits

Each later idea stays unbuilt and waits for the trigger that brings it in; detailed planning waits with it ([DEC-136](decisions.md#dec-136-later-ideas-wait-as-conduits-with-a-trigger)).

| Idea | What keeps the way open | Trigger |
|---|---|---|
| **A user interface, other clients, further providers** | Facts come only from the tool, the tool does not know where it runs, and the skills hold for Claude Code and Codex alike | Markitect is proven ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)) |
| **A mapping to a ticket system** | Plan items keep their forms and have no number of their own; a ticket's number would stand at the plan item, never in a file name | A project must keep its plan in a ticket system |
| **Telemetry for reference values** | Reference values ship with the packages and are first gathered locally | Markitect is proven, and outside projects join of their own will; numbers only, never text |
| **Imports as further suspects** | The state knows the artifacts of every document. When an artifact of one document imports one of another, a change of the second could make the first suspect too: wider, never narrower | Spot checks or control samples repeatedly show errors that ran through a code dependency without a link |
| **Operation** | Logs and incidents become reports through an observer. Counter-checks are executable and come from the spec alone; run against the live system, they are synthetic monitoring | The first project in operation with logs |
| **Variants per context,** such as a site, an environment or a tenant | As Terraform has workspaces and variables: one spec, and values per context apart from it; a section can link a document of values per context | The first project with a second site, environment or tenant that differs in what is fixed |

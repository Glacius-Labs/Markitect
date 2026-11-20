# Roadmap

Where Main is, what comes next and what waits for a trigger. The [concept](concept/README.md) is the target; the [decisions](decisions.md) are what holds.

## Where Main is

The decisions, the concept, the vision and the foundations describe the target. The code, the guides in `docs/guides/`, the pages under `internal/` and the latest release are still the 0.34.0 tool, kept with its earlier concept on an archive branch ([history](history.md)): a spec of Definitions, Kinds and Packages under `.markitect/spec/` with a registry; `check`, `vet`, `list`, `doc` and `tidy`; the record with `status` and `mark`, the history with `markitect run`, and the inbox; skills and four agents whose rights a hook holds, run by the owner's own session; `harness.yaml`; and `markitect update`. They are rebuilt to the concept step by step, below. Until a step has landed, its part of the concept is not built, and the guides describe 0.34.0.

## Current steps

Failures are fixed in the tool contract, not with more prompt text ([DEC-141](decisions.md#dec-141-agents-get-tasks-not-process)).

| Step | Result | Done when |
|---|---|---|
| 1. The documentation | Decisions, concept, vision, foundations, roadmap and history describe the concept | The target documentation is merged; the guides and the pages under `internal/` follow with the code in steps 2 to 4 |
| 2. The tool core for the language | The tool rebuilt in Go for the language ([DEC-135](decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language)): `check`, `show`, `table`, the topic view, `new`, `move` and `apply`, and the library inside it: the packages `core`, `software` and `company`, the templates, the report type, reference values and question catalogs. The rebuilt tool is built alongside the 0.34.0 tool as `markitect-next`, so that `markitect` keeps working ([language reference](guides/language-reference.md)) | The tool checks and shows a spec of the language, and a template applied with its packages passes `check` apart from its empty sections |
| 3. The state | The project: `init`, which writes `.markitect/` and the spec folder and nothing else, and every command finding the project from the folder it runs in ([project reference](guides/project-reference.md)); `status` with changed, suspect and unexplained entries; `mark`; `move` carrying the state; the history; the ignore file; the comparison of two states and the figures computed from the history | Over the changes of a fixture project, `status`, `mark` and `move` hold, and the comparison names what is new, gone, changed or suspect |
| 4. The jobs | Result files of fixed form, the write table and its check, reports with the entry "waiting on report" in `status`, the figures computed from the reports and the control samples, and the landing rule in `status`; the settings, which `init` writes too; the jobs and the skills for the briefing and explain in the plugin. The hook, the inbox, `run`, `update`, `install`, `tidy`, `config`, `vet`, `list`, `doc` and `harness.yaml` are removed, and the guides are rewritten | Each job runs in a session on a committed test project and lands by the landing rule, and the check refuses a step that writes outside its area. The packages and the command of the 0.34.0 tool are removed, and `markitect-next` is renamed `markitect` |
| 5. The runners | Entries for the user's own session, a scheduled task, a cloud task and CI, with the version pinned in the project | The same job on the same commit lands the same way in a session and in CI |
| 6. The lab test | The 5× Makerspace project's eight change steps, with their hidden cases, which are kept at [`b8fd2c94`](https://github.com/Glacius-Labs/Markitect/tree/b8fd2c94722af066ba4ff6337aeba0459cb804ec/tools) ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)) | Two clean passes in a row: every step lands by the landing rule, and every hidden case passes |
| 7. The proof | A project that is not the owner's, in the loop with real work in CI | Four weeks in which the four figures of [DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven) hold: work, convergence, checking and no rescue |

## Conduits

Each later idea stays unbuilt and waits for the trigger that brings it in; detailed planning waits with it ([DEC-136](decisions.md#dec-136-later-ideas-wait-as-conduits-with-a-trigger)).

| Idea | What keeps the way open | Trigger |
|---|---|---|
| **Teams:** a person responsible per area as the target of escalation, notification by role, and who carries the jobs on whose subscription | An area is a folder with a README, every report names its place, and the briefing exists for every reader. A table of path patterns to people in `.markitect/`, like `CODEOWNERS` and in the form of the write table, can send escalations to the person responsible for the place; the owner stays the default | A second person makes decisions in the same project |
| **A user interface, other clients, another provider** | Facts come only from the tool, the tool does not know where it runs, and the jobs are the only part tied to a provider | Markitect is proven ([DEC-120](decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)) |
| **Telemetry for reference values** | Reference values ship with the packages and are first gathered locally | Markitect is proven, and outside projects join of their own will; numbers only, never text |
| **Imports as further suspects** | The state knows the artifacts of every document. When an artifact of one document imports one of another, a change of the second could make the first suspect too: wider, never narrower | Spot checks or control samples repeatedly show errors that ran through a code dependency without a link |
| **Operation** | Logs and incidents become reports through intake. Counter-checks are executable and come from the spec alone; run against the live system, they are synthetic monitoring | The first project in operation with logs |
| **Variants per context,** such as a site, an environment or a tenant | As Terraform has workspaces and variables: one spec, and values per context apart from it; a section can link a document of values per context | The first project with a second site, environment or tenant that differs in what is fixed |

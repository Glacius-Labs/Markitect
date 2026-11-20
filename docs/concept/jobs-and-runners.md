# Jobs and runners

**Markitect is three parts: the tool, the jobs and the runners** ([DEC-132](../decisions.md#dec-132-markitect-is-a-tool-jobs-and-runners)). Terraform is the image again: a command line that computes the same everywhere, with a version the project fixes, while a CI job or a service such as Atlantis runs `plan` and `apply` on a pull request and merges when the checks are green.

| Terraform | Markitect | What it is |
|---|---|---|
| The `terraform` command line | The tool ([§11](the-tool.md)) | Computes and checks, with no agent and no Git |
| Providers and modules from the registry | The library inside the tool | Packages and templates, copied into the spec, which owns its copy |
| `required_version` | The version in the project | Every runner uses the same tool |
| The pipeline definition | The jobs | What each job does: roles, tasks, write areas |
| Atlantis, Terraform Cloud, CI | The runners | Where a job runs; the user's choice |
| A pull request with required checks | An action with the landing rule | Lands when everything is green |

## Jobs

The jobs are the factory's working instructions: for each job ([§8](the-factory.md#the-jobs)), which roles run in which order, with which task and which write area. They ship in the Claude Code plugin, not in the project, and they are the only part tied to a provider. A skill of the plugin is the entry with which a person starts a job in their own session. The briefing and explain are conversations, not jobs, and ship as skills too ([§10](overview.md)).

**The way of a job is the same everywhere.** The runner fetches a commit into a fresh copy, calls each role as a step of its own, and records the state after each step. At the end there is a branch. The landing rule is a check outside every agent: `status` against the target branch, the tests and the counter-checks, and the write area of every step ([§8](the-factory.md#the-landing-rule)).

## Rights come from the steps

Each role runs as its own step, with its own allowed tools. The runner lays out for the tool the state before and after each step, as for a comparison. The tool checks each change against the write table, by file and by row of the state and the history, and refuses what the step's role may not write.

| Role or step | May write |
|---|---|
| Builder | Artifacts |
| Counter-checker | The counter-check folder the settings name |
| Reviewer | Verdict rows only, through `mark` |
| Architect | The spec, the ignore file and the rows that `move` writes |
| The job itself | Reports and results of control samples, with the job as source |
| Triage | Rating, actions and closing in reports |
| Owner, on a branch from a conversation | All of the above, the control files, and reports with the owner as source |
| Reader, sampler | Nothing that lands, but the sampler's list of planted deviations, which the job writes |

The write table ships with the tool, and a project may only narrow it in its settings.

**Control files** are the write table, the settings, the version, the spec path, the learned rules and the CI definition. The tool reads them from the target branch, not from the branch that wants to land, so a job cannot widen its own rights. A verdict row that did not come from the reviewer's step keeps the branch from landing.

**Where the check is a fact.** In CI the runner is a script that no agent can change, and it records the states out of every agent's reach; there the check is a fact, and a required check that a job cannot skip. In a session or a scheduled task the runner is Claude itself; there the check makes shortcuts visible and proves nothing. Markitect guards against forgetting and shortcuts, and against a deliberate bypass only by making it visible.

## Runners

Which runner a job uses is the user's choice, and Markitect ships an entry for each. All of them run on the login of the user's Claude subscription; API keys are never used ([DEC-134](../decisions.md#dec-134-jobs-run-on-the-users-subscription-with-or-without-a-session)).

| Runner | How | Limit |
|---|---|---|
| The user's own session | By hand, from a skill | Needs the user |
| A scheduled task of the desktop app | On the user's machine, on the app's login | Runs only while the app is open and the machine awake |
| A cloud task | On a fresh copy of the repository; pushes a branch and can open a pull request | GitHub only |
| CI | A workflow with a token that `claude setup-token` makes for a runner without a session; the plugin is installed in the workflow | The token belongs to the subscription of whoever made it |

The runner, not the tool, tells the owner about an escalation: in CI as an issue or a mention on the pull request, otherwise through a connector or the desktop app's notification. The owner fetches the briefing themselves.

Agents run with the rights of whoever runs them, so work that reads untrusted input needs an isolated machine with limited rights around it; intake keeps outside text from agents that may change something ([§9](reports-and-actions.md#intake)). The tool runs natively on Windows, Linux and macOS. Claude Code is the only provider for now, and the tool does not change for another.

## The project

A project holds two things of Markitect's ([DEC-133](../decisions.md#dec-133-the-project-holds-only-the-spec-and-markitect-and-pins-its-version)):

- **the spec,** an ordinary folder of documentation, by default `docs/spec/`, whose path `.markitect/` names;
- **`.markitect/`,** with the version, the settings, the state, the history, the reports, the results of control samples, the ignore file, the learned rules and its own `.gitattributes`, which has Git keep both sides of the state and the history.

`init` writes nothing else: no `AGENTS.md`, no `CLAUDE.md`, nothing in `.claude/`, no line in `.gitignore`. A session learns from the plugin that a folder with `.markitect/` has a spec. What a runner needs, such as a CI definition, belongs to the user like any other CI.

**Settings** exist only in the project, since a job in CI has no user profile: the model and effort per role, the cadence of the jobs, the counter-check folder, and a narrower write table. They are a file the user or Claude edits and `check` checks.

**The version is pinned in the project,** and every runner uses it, since the state is hashes the tool computes and two versions could disagree. In Claude Code the plugin fetches the tool in the version the project names; elsewhere a setup step fetches the same version from the release. A newer version is taken by changing the version line, as with any other dependency. A release publishes the tool for Linux, macOS and Windows with checksums ([releasing](../releasing.md)).

**The library is inside the tool:** packages, templates, the report type, reference values and question catalogs. `apply` copies packages and templates into the spec, which owns its copy; a newer package in the library shows in a comparison, and an architect job takes it where it fits. Reference values and catalogs are not part of the spec and are read from the tool's version, so they improve with each version. The report type belongs to the tool, since the tool reads its sections for the figures.

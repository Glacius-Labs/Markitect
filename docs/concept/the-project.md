# The project and its skills

**`.markitect/` holds the work, and `init` lays out templates and skills** ([DEC-153](../decisions.md#dec-153-markitect-holds-the-work-init-lays-out-templates-and-skills-and-the-plugin-fetches-its-own-version)). A project holds three things of Markitect's: the spec, an ordinary folder of documentation, by default `docs/spec/`; the skills of the standard, as artifacts of the spec; and `.markitect/`.

## `.markitect/`

```text
.markitect/
  version                     the tool's version the project pins
  spec                        the path of the spec folder
  ignore                      files that realize nothing, written like a .gitignore
  state.jsonl                 one row per document (§6)
  history.jsonl               the facts of each confirmation and move (§6)
  .gitattributes              keeps both sides of the state and the history in a merge
  context/                    everything that happened
    <agent>/                  one folder per kind of agent, named like its Agent document
      YYYY-MM-DD-HHMM-<item>.md   one report per run or session
      checkpoint.md           where the agent continues, where needed
  findings/
    pending/                  needs a person
    ready/                    released for work
    closed/                   done, with its outcome
  plan/                       Features and Stories, as a tree under their Areas
  briefing/                   daily, weekly and per meeting
```

Each observer is a kind of agent of its own, with its own folder in `context/`; a report without an item, such as an observer's, is named by date and time only. The state and the history are the results of `confirm`; the context, the findings, the plan and the briefing are the work everyone continues from, and several agents write the plan ([§9](the-work.md)).

## `init`

`init` lays out:

- `.markitect/`, with the version, the spec path and empty folders for the work;
- the spec, with the four templates `project`, `product`, `architecture` and `factory` ([§4](what-goes-in-a-spec.md#four-templates));
- the skills of the standard, in `.claude/skills/` and `.agents/skills/`;
- a short entry in `AGENTS.md`, which names the spec, the skills and `markitect-assistant` as the entry.

Deleting `.markitect/` and the tool leaves an ordinary repository with documentation, code, tests and skills.

## The skills

The skills of the standard are built from the `factory` template, one per Agent document, each named `markitect-` and the document's name: `markitect-assistant`, `markitect-capture`, `markitect-architect`, `markitect-finish-planning`, `markitect-build`, `markitect-review`, `markitect-analyze`, `markitect-accept`, `markitect-observe`, `markitect-create-observer`, `markitect-triage`, `markitect-record`, `markitect-brief` and `markitect-housekeeper` ([§8](the-circle.md#agents)). Each describes its agent's work completely and step by step; `markitect-capture`, `markitect-finish-planning` and `markitect-create-observer` run inside the session that calls them.

- **They lie in the repository,** never in the plugin, where Claude Code and Codex both find them, and hold for both alike ([DEC-147](../decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)).
- **They are artifacts of the spec.** When the `factory` part changes, the skills are built and confirmed again like any code, and the state maps each to the documents it realizes. A failure in how agents work is fixed in the `factory` part and the skill built from it, not with prompt text beside them.
- **What steers an agent stands in the spec.** Which agents and observers exist, their model, one line per provider, such as Sonnet, Opus or Fable for Claude and Luna, Sol or Astra for Codex, their schedule and whether they ask stand at their Agent document; persons, roles and meetings stand there too. There is no settings file. To give the builder another model, a person changes the builder's Agent document with the Architect; nothing has to be built, since whoever starts the builder reads its model there.

## Who starts an agent

An agent is started by a person, by the Assistant, by the Housekeeper or by a routine, as its Agent document says ([DEC-142](../decisions.md#dec-142-the-tool-computes-and-checks-a-runner-acts)). Agents and people make every Git step; the tool makes none.

Agents run on the login of the user's subscription: in their own session, as a scheduled task of the desktop app, as a cloud task, or in CI with a token that `claude setup-token` makes for a run without a session. API keys are never used ([DEC-134](../decisions.md#dec-134-agents-run-on-the-users-subscription-with-or-without-a-session)). Agents run with the rights of whoever runs them, so work that reads untrusted input needs an isolated machine with limited rights around it.

## The plugin, the version and the library

**The plugin only fetches the tool.** Its launcher fetches the plugin's own version, which is the tool's version, and does not read the version a project pins. It brings no skills.

**The project pins a version** in `.markitect/version`, since the state is hashes the tool computes and two versions could disagree. `markitect version` says whether the tool that runs is that version. A newer version is taken by changing the version line and using a tool of that version, as with any other dependency. A release publishes the tool for Linux, macOS and Windows with checksums ([releasing](../releasing.md)).

**The library is inside the tool:** packages, templates with the skills of the standard, reference values and question catalogs. `apply` copies packages and templates into the spec, which owns its copy; a newer package in the library shows in a comparison, and the Architect takes it where it fits. Reference values and catalogs are not part of the spec and are read from the tool's version, so they improve with each version. The [library](../guides/library.md) describes its content.

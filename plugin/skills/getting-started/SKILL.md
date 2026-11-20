---
name: getting-started
description: Use when someone mentions Markitect, wants to set it up or start a project with it, or asks about the tool markitect, or about a folder with .markitect/. It says what Markitect is and how to begin.
---
# Markitect

With Markitect, people architect their project in a typed specification language in Markdown. A folder with `.markitect/version` is a Markitect project: its spec is an ordinary folder of documentation, by default `docs/spec/`, and it is the authority over the code, the tests and the documentation. The tool `markitect` checks the spec, says what is changed, suspect or unexplained, and binds verdicts; it starts no agent and uses no Git. The jobs of this plugin realize the spec, check it and measure how well the checking works, each role as an agent of its own, and the owner is asked only for new decisions.

## The tool

`markitect` is a command on the PATH of Claude Code's shell while this plugin is on: its launcher downloads the release binary of the plugin's own version on first use and checks it against the checksums of that release. A development version of the plugin has no release binary: its launcher starts the next `markitect` on the PATH instead, a build from source, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@COMMIT` with the commit of the plugin. `markitect help` lists the commands, and `markitect version` says whether the tool is the version the project pins.

## How to begin

- **A folder without `.markitect/`:** `markitect init` writes `.markitect/` and the spec folder, and nothing else. Then the `markitect:markitect-architect` skill walks the first design with the owner, area by area.
- **A project with a spec:** `markitect check` says whether the spec is sound, and `markitect status` what is to build or confirm.
- **Settings:** `.markitect/settings.md` names the model per role and the counter-check folder. Add a `## Tests` section to it with the command that runs the tests and the counter-checks, since `init` writes none: without it, no job lands.

## The skills

| Skill | Does |
|---|---|
| `markitect:markitect-architect` | The conversation with the owner, and the architect job |
| `markitect:markitect-build` | Builds what `status` lists: counter-checker, builder, reviewer |
| `markitect:markitect-review` | A spot check of confirmed documents, or all of them |
| `markitect:markitect-control-sample` | Planted deviations, checked blind |
| `markitect:markitect-bughunt` | Looks beyond the list; findings become reports |
| `markitect:markitect-cleanup` | Changes no behaviour |
| `markitect:markitect-improve` | Changes behaviour only where the spec leaves it free |
| `markitect:markitect-read` | Reads and questions the spec |
| `markitect:markitect-triage` | Sorts, rates and switches new reports |
| `markitect:markitect-intake` | Turns an outside signal into a report |
| `markitect:markitect-briefing` | How the project stands, and what changed since the last look |
| `markitect:markitect-explain` | Explains the spec, from the spec alone |

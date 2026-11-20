---
name: getting-started
description: Use when someone mentions Markitect, wants to set it up or start a project with it, or asks about the tool markitect-next, or about a folder with .markitect/. It says what Markitect is and how to begin.
---
# Markitect

With Markitect, people architect their project in a typed specification language in Markdown. A folder with `.markitect/version` is a Markitect project: its spec is an ordinary folder of documentation, by default `docs/spec/`, and it is the authority over the code, the tests and the documentation. The tool `markitect-next` checks the spec, says what is changed, suspect or unexplained, and binds verdicts; it starts no agent and uses no Git. The jobs of this plugin realize the spec, check it and measure how well the checking works, each role as an agent of its own, and the owner is asked only for new decisions.

## The tool

`markitect-next help` lists the commands. Until the plugin fetches the tool in the version a project pins, build it from source with Go and put it on the PATH: `go install github.com/Glacius-Labs/Markitect/cmd/markitect-next@COMMIT`, with the commit of the plugin. `markitect-next version` says whether it is the version the project pins.

## How to begin

- **A folder without `.markitect/`:** `markitect-next init` writes `.markitect/` and the spec folder, and nothing else. Then the `markitect-next:markitect-architect` skill walks the first design with the owner, area by area.
- **A project with a spec:** `markitect-next check` says whether the spec is sound, and `markitect-next status` what is to build or confirm.
- **Settings:** `.markitect/settings.md` names the model per role and the counter-check folder. Name the command that runs the tests and the counter-checks in its Tests section: without it, no job lands.

## The skills

| Skill | Does |
|---|---|
| `markitect-next:markitect-architect` | The conversation with the owner, and the architect job |
| `markitect-next:markitect-build` | Builds what `status` lists: counter-checker, builder, reviewer |
| `markitect-next:markitect-review` | A spot check of confirmed documents, or all of them |
| `markitect-next:markitect-control-sample` | Planted deviations, checked blind |
| `markitect-next:markitect-bughunt` | Looks beyond the list; findings become reports |
| `markitect-next:markitect-cleanup` | Changes no behaviour |
| `markitect-next:markitect-improve` | Changes behaviour only where the spec leaves it free |
| `markitect-next:markitect-read` | Reads and questions the spec |
| `markitect-next:markitect-triage` | Sorts, rates and switches new reports |
| `markitect-next:markitect-intake` | Turns an outside signal into a report |
| `markitect-next:markitect-briefing` | How the project stands, and what changed since the last look |
| `markitect-next:markitect-explain` | Explains the spec, from the spec alone |

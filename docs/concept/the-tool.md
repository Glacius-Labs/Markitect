# The tool

**The `markitect` program checks the spec, analyses it and helps reading it** ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). It never writes and keeps no state. It starts no agent and steers nothing. It uses no Git ([DEC-097](../decisions.md#dec-097-markitect-uses-no-git)): it reads `.markitect/spec` and the spec folder it names, and nothing else of the project. So it does not know where it runs, and gives the same answer in a session and in CI. Anything more is a separate tool or a script of the project.

## Commands

The commands are flat, as Git's are, and each does one thing on its own. The help groups them.

| Group | Command | Does |
|---|---|---|
| Look | `show DOCUMENT` | A document with its type, its sections, what it links to and what links to it with the section the link stands in; for a type, its documents |
| | `table [TYPE]` | Every document of a type as a table, one column per section; without a type, every document with its type and title |
| | `topic [TOPIC]` | The topic view: a document, a type or a folder as one text with everything that belongs to it, and the signals that say where reading pays off; without a topic, the signals of the whole spec ([§5](is-the-spec-good.md#the-spec-itself)) |
| | `questions` | The competency questions of the packages the spec has ([§5](is-the-spec-good.md#the-spec-itself)) |
| | `diff OLD [NEW]` | What is new, gone or changed in the spec of NEW, by default this project, against the project OLD, with a diff for what changed |
| Check | `check` | Checks the spec: titles, types, sections and links ([§3](the-language.md#what-check-catches)) |
| Tool | `version` | The tool's version and build |
| | `licenses` | The third-party notices |

## Reading the spec

**The tool shows, the agent explains** ([DEC-131](../decisions.md#dec-131-the-tool-shows-claude-explains)). What people should not have to take on an agent's word does not come from an agent. The tool, and Git where changes and branches are concerned, show: completely, deterministically, with numbers that add up. The agent tells and explains, with a citation for each statement, and ends by naming what it did not cover.

The image is a building model. Floor plan, section and bill of materials are generated from one model, never drawn twice. The spec is the model, and every view is generated from it.

- **What is the project?** The spec itself, the topic view and the table per type, explained by the agent. An explanation of the whole, a part, a document or a question comes only from the spec, with a citation for each statement, and otherwise says "not stated", as a competency question is answered ([§5](is-the-spec-good.md#the-spec-itself)). "Explain the whole" goes through the parts in the order of their tables of contents. A picture, such as the states of a concept, the agent draws only from what documents say, and names them below it.
- **What changed?** A change to the spec is seen through the ordinary diff of the repository, like every change. `diff` compares the specs of two folders, such as the project and a worktree of an older commit that a person or an agent lays out, and names what is new, gone or changed.

Views are generic and come from the language, so they work for every spec: the topic view, a table per type, `show` and `diff`. Views particular to a package come with a user interface. A dashboard or a handbook that a project wants is an artifact of the project, built from its spec like any other.

There is no user interface of Markitect for now. The spec is readable Markdown, the tool runs natively, and people ask their agent. A user interface comes once Markitect is proven ([§8](what-is-shown.md)).

## Output and exit codes

A diagnostic is one line, `file:line: code: message`. With `--json`, a command prints one document that starts with a `format_version` and a `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run. The [language reference](../guides/language-reference.md#commands) gives every command exactly.

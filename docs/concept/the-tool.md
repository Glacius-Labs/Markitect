# The tool

**The `markitect` program checks the model, analyses it and helps reading it** ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). It never writes and stores nothing. It starts no agent and steers nothing. It neither needs nor knows Git or any other version control ([DEC-097](../decisions.md#dec-097-the-language-and-the-tool-neither-need-nor-know-version-control)): it reads folders. Every command but `diff` reads `.markitect/model` and the model folder it names, and nothing else of the project; `diff` also reads the project's files and `.markitect/ignore`. So it does not know where it runs, and gives the same answer in a session and in CI. Anything more is a separate tool or a script of the project.

## Commands

The commands are flat, and each does one thing on its own. The help groups them.

| Group | Command | Does |
|---|---|---|
| Look | `show DOCUMENT` | A document with its type, or the type it extends, its sections, what it links to and what links to it with the section the link stands in; for a type, each property's type, the types that extend it and its objects |
| | `table [TYPE]` | Every object of a type, and of the types that extend it, as a table, one column per section; without a type, every document with its type and title |
| | `topic [TOPIC]` | The topic view: a document, a type or a folder as one text with everything that belongs to it, and the signals that say where reading pays off; without a topic, the signals of the whole model ([§5](is-the-model-good.md#the-model-itself)) |
| | `questions` | The competency questions of the default's module `core`, when the model has it ([§5](is-the-model-good.md#the-model-itself)) |
| | `diff OLD [NEW]` | What is new, gone or changed in the model of NEW, by default this project, against the project OLD, with a diff for what changed; for each module whether its model, its files or both changed; and the files no module names |
| Check | `check` | Checks the model: titles, types, sections, the types of properties, links, the READMEs of modules and cycles between modules ([§3](the-language.md#what-check-catches)) |
| Tool | `version` | The tool's version and build |
| | `licenses` | The third-party notices |

## Reading the model

**The tool shows, the agent explains** ([DEC-131](../decisions.md#dec-131-the-tool-shows-claude-explains)). What people should not have to take on an agent's word does not come from an agent. The tool, and the project's version control where changes are concerned, show: completely, deterministically, with numbers that add up. The agent tells and explains, with a citation for each statement, and ends by naming what it did not cover.

The image is a building model. Floor plan, section and bill of materials are generated from one model, never drawn twice. Every view of the project is generated from its model in the same way.

- **What is the project?** The model itself, the topic view and the table per type, explained by the agent. An explanation of the whole, a module, a document or a question comes only from the model, with a citation for each statement, and otherwise says "not stated", as a competency question is answered ([§5](is-the-model-good.md#the-model-itself)). "Explain the whole" goes through the modules, each from its README. A picture, such as the states of a concept, the agent draws only from what documents say, and names them below it.
- **What changed?** A change to the model is seen through the ordinary diff of the repository, like every change. `diff` compares two states of the project, such as the project and a copy of an older state of it, however its owner made that copy, and nothing is stored between them ([DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-model-and-its-files)). It names the documents of the model that are new, gone or changed, and answers which modules changed: by the paths under `Realized in` in each module's README ([§3](the-language.md#modules)), it says for each module whether its model, its files or both changed, and so where the model and the files went apart, and it names the files that no module names. Such a change is reported, not judged: a module whose files changed without its model may have drifted or been rebuilt to the same model, and the tool cannot know which. Files that are not the project's own, such as build output, the project leaves out in `.markitect/ignore` ([§7](the-project.md)).

Views are generic and come from the language, so they work for every model: the topic view, a table per type, `show` and `diff`. Views particular to the types of the default come with a user interface. A dashboard or a handbook that a project wants is an artifact of the project, built from its model like any other.

There is no user interface of Markitect for now. The model is readable Markdown, the tool runs natively, and people ask their agent. A user interface comes once Markitect is proven ([§8](what-is-shown.md)).

## Output and exit codes

A diagnostic is one line, `file:line: code: message`. With `--json`, a command prints one document that starts with a `format_version` and a `kind`. Exit 0 means all is well, 1 that the command found something, and 2 that it could not run. The [language reference](../guides/language-reference.md#commands) gives every command exactly.

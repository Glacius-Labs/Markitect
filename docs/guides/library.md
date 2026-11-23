# Library

The library is the folder [`internal/library/`](../../internal/library/) of Markitect's repository, where the default lies: today the one module `core`, in [`internal/library/core/`](../../internal/library/core/), and its reference values and question catalog ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec), [DEC-156](../decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows); [where Main is](../roadmap.md#where-main-is)). The module is plain files, a README and six types, which a person copies into a spec by hand, as the [project reference](project-reference.md#setting-up-a-project-by-hand) shows; a spec owns its copy and may change it. Only the reference values and the question catalog are built into the tool, which reads them and never copies them; [signals](signals.md) describes them. [The language reference](language-reference.md) gives the rules of the language and the commands.

## core

`core` holds general types that every kind of project uses. Its README shows all six under `Contents`, so every module of a spec may name them. Every type says before its sections what it is and when to take it, and when not, where it is easily confused with another. Under every section it says what belongs there, with a short example. Sections marked *optional* may be left out.

| Type | Sections |
|---|---|
| Concept | Definition; *Consists of*, *Lifecycle*, *Rules* |
| Decision | Decision, Why; *Alternatives*, *Reopen when* |
| Goal | Goal; *By* |
| Role | Does; *Does not*, *Reports to* |
| Rule | Applies to, Rule; *Responsible*, *Example* |
| Term | Meaning |

A role describes the usual way of working, not a permission: *Does not* names what the role usually leaves to another.

The tool knows the module as the folder `core` directly below the spec folder: there `questions` finds its catalog and `topic` its reference values ([signals](signals.md)). A copy placed elsewhere works as types like any others, without them.

Until a new default is designed, `core` is the whole default: types for a software project, a business or a way of working are a project's own ([roadmap](../roadmap.md#next)).

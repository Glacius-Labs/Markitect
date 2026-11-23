# Roadmap

Where Main is, what comes next and what waits for a trigger. The [concept](concept/README.md) is the target; the [decisions](decisions.md) are what holds.

## Where Main is

Main builds Markitect as the language, its default and a tool that only reads ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). The language groups documents in modules, folders whose README shows what other modules may link ([DEC-156](decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)) and names under `Realized in` where in the project the module is built. `diff` compares two states of a project: the documents of the spec, and for each module whether its spec, its files or both changed, with the files no module names, leaving out what `.markitect/ignore` names ([DEC-158](decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-spec-and-its-files)). The library ([library](guides/library.md)) holds the default, the one module `core` with six general types, as plain files, which a person copies into a spec by hand ([project reference](guides/project-reference.md#setting-up-a-project-by-hand)); only its reference values and question catalog are built into the tool. The tool finds a project by its file `.markitect/spec` and has the commands `check`, `show`, `table`, `topic`, `questions`, `diff`, `version` and `licenses`, none of which writes. The plugin only fetches the tool ([plugin](guides/plugin.md)). Markitect ships no way of working and no skill. The guides describe the tool as built and follow these pages where they differ. No release of the tool on Main exists: the latest release and the marketplace's plugin are still 0.34.0 of an earlier concept, kept on an archive branch ([history](history.md)), and the tool on Main is built from source ([releasing](releasing.md)).

## Next

- **A new default beyond `core`:** open. Until it is designed, the default is `core` alone, and a project writes its own types for a software project, a business or its way of working.
- **A way of working in the default:** designed anew as part of the default, with general types that a project uses to describe its own.
- **The default in a project:** how the default reaches a project more comfortably than a copy by hand, with setting up a project as a skill of the plugin.

## Conduits

Each later idea stays unbuilt and waits for the trigger that brings it in; detailed planning waits with it ([DEC-136](decisions.md#dec-136-later-ideas-wait-as-conduits-with-a-trigger)).

| Idea | What keeps the way open | Trigger |
|---|---|---|
| **A user interface and other clients** | Facts come only from the tool, and the tool does not know where it runs | Markitect is proven ([OQ-015](decisions.md#open-questions)) |
| **Telemetry for reference values** | Reference values ship with the default and are first gathered locally | Markitect is proven, and outside projects join of their own will; numbers only, never text |
| **Variants per context,** such as a site, an environment or a tenant | As Terraform has workspaces and variables: one spec, and values per context apart from it; a section can link a document of values per context | The first project with a second site, environment or tenant that differs in what is fixed |

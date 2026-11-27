# Roadmap

Where Main is, what comes next and what waits for a trigger. The [concept](concept/README.md) is the target; the [decisions](decisions.md) are what holds.

## Where Main is

Main builds Markitect as the language, its default and a tool that only reads ([DEC-154](decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). The language groups documents in modules, folders whose README shows what other modules may link ([DEC-156](decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows)) and names under `Realized in` where in the project the module is built. `diff` compares two states of a project: the documents of the model, and for each module whether its model, its files or both changed, with the files no module names, leaving out what `.markitect/ignore` names ([DEC-158](decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-model-and-its-files)). The default is a whole model to start from, in the plugin's folder `plugin/default/`: the module `core` with thirteen general types, the module `way-of-working` with a complete way of working, and the modules `plan`, `findings`, `reports`, `briefings` and `drops`, which start empty and take the project's work ([DEC-159](decisions.md#dec-159-the-default-is-a-whole-model-to-start-from-and-lies-in-the-plugin), [library](guides/library.md)). A person copies it into a model by hand ([project reference](guides/project-reference.md#setting-up-a-project-by-hand)); only the reference values and question catalog of `core` are built into the tool. The tool finds a project by its file `.markitect/model` and has the commands `check`, `show`, `table`, `topic`, `questions`, `diff`, `version` and `licenses`, none of which writes. The plugin fetches the tool and holds the default ([plugin](guides/plugin.md)). No skill is built: neither the skill that sets a project up nor the skills that the way of working names under `Realized in`, so the plugin holds no skill and no agent. The guides describe the tool as built and follow these pages where they differ. No release of the tool on Main exists: the latest release and the marketplace's plugin are still 0.34.0 of an earlier concept, kept on an archive branch ([history](history.md)), and the tool on Main is built from source ([releasing](releasing.md)).

## Next

1. **The skill that sets a project up:** a skill of the plugin that copies the default into a project and leads through the default's [set-up](../plugin/default/way-of-working/set-up.md), so that nobody sets up a project by hand.
2. **The skills of the way of working:** one skill per activity an agent does, `markitect-NAME`, built from the activity's document, where the modules of the way of working name them under `Realized in`.

## Conduits

Each later idea stays unbuilt and waits for the trigger that brings it in; detailed planning waits with it ([DEC-136](decisions.md#dec-136-later-ideas-wait-as-conduits-with-a-trigger)).

| Idea | What keeps the way open | Trigger |
|---|---|---|
| **A user interface and other clients** | Facts come only from the tool, and the tool does not know where it runs | Markitect is proven ([OQ-015](decisions.md#open-questions)) |
| **Telemetry for reference values** | Reference values ship with the default and are first gathered locally | Markitect is proven, and outside projects join of their own will; numbers only, never text |
| **Variants per context,** such as a site, an environment or a tenant | As Terraform has workspaces and variables: one model, and values per context apart from it; a section can link a document of values per context | The first project with a second site, environment or tenant that differs in what is fixed |

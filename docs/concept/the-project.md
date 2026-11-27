# The project

**A project is the nearest folder upward that holds the file `.markitect/model`** ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). Its one line names the model folder, an ordinary folder of documentation such as `docs/model/`. Beside it, the file `.markitect/ignore`, optional, names files that are not the project's own, such as build output and fetched dependencies, so that `diff` leaves them out ([project reference](../guides/project-reference.md#the-ignore-file)). These two files are all Markitect keeps in a project, both in the one folder `.markitect/`: the tool reads nothing else from it, and nothing creates them. A person writes them.

Deleting the folder `.markitect/` and the tool leaves an ordinary repository with its documentation, code and tests; a copied default is part of that documentation.

## The model and the files

The model says what should hold, and the project's files are what is. Each module's README names under `Realized in` where in the project the module is built ([§3](the-language.md#modules)), so that `diff` can compare two states of a project module by module and report a change without judging it ([§6](the-tool.md#reading-the-model), [DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-model-and-its-files)).

## Setting up a project

A project is set up by hand: write `.markitect/model`, copy the default into the model folder, write the project's own modules, each a folder with a README ([§3](the-language.md#modules)), and run `check` until it reports nothing. The [project reference](../guides/project-reference.md#setting-up-a-project-by-hand) gives the steps. A skill of the plugin that sets a project up is not built ([roadmap](../roadmap.md#next)).

## The default

The default is what ships with Markitect: a whole model to start from ([DEC-159](../decisions.md#dec-159-the-default-is-a-whole-model-to-start-from-and-lies-in-the-plugin); [library](../guides/library.md)).

- **The model** is plain files in the plugin's folder, `plugin/default/`: the module `core` with general types, such as Role, Rule, Decision and Activity; the module `way-of-working`, a complete way of working written in them, with types of its own for the work, such as Story and Finding; and the modules `plan`, `findings`, `reports`, `briefings` and `drops`, which start empty and take the project's work. A project copies it into its model folder, so that `core` lies directly below it, and owns every file of its copy and may change any of it.
- **Reference values and the question catalog** of `core` are not part of the model. They are built into the tool and read from its version, so they improve with each version: `topic` shows the reference values beside the project's own figures, and `questions` prints the catalog for a model that has `core` ([§5](is-the-model-good.md)).

## The way of working

A way of working is always the project's own, also when it starts from the one the default ships ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)). The project describes how its people and agents work in its model, with the general types of `core` and types of its own, and skills, agents and scripts realize it, as code realizes the product. The default's way of working keeps the work in the model too, in modules of its own ([§4](what-goes-in-a-model.md#the-end-state-not-the-way)), and names under `Realized in` the skills that realize its activities; those skills are not built.

## The plugin and the version

**The plugin fetches the tool and holds the default** ([DEC-135](../decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language), [DEC-159](../decisions.md#dec-159-the-default-is-a-whole-model-to-start-from-and-lies-in-the-plugin)). Its launcher fetches the release of the plugin's own version, which is the tool's version, checks it against the release's checksums and starts it ([the plugin](../guides/plugin.md)). A project pins no version: `markitect version` names the tool's version and build. A release publishes the tool for Linux, macOS and Windows with checksums ([releasing](../releasing.md)).

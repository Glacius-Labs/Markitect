# The project

**A project is the nearest folder upward that holds the file `.markitect/spec`** ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). Its one line names the spec folder, an ordinary folder of documentation such as `docs/spec/`. Beside it, the file `.markitect/ignore`, optional, names what is not the project's own files, such as build output and fetched dependencies, so that `diff` leaves it out ([project reference](../guides/project-reference.md#the-ignore-file)). These two files are all Markitect keeps in a project, both in the one folder `.markitect/`: the tool reads nothing else from it, and nothing creates them. A person writes them.

Deleting the folder `.markitect/` and the tool leaves an ordinary repository with its documentation, code and tests.

## The spec and the files

The spec says what should hold, and the project's files are what is. Each module's README names under `Realized in` where in the project the module is built ([§3](the-language.md#modules)), so two states of a project show which modules moved, and where the spec and the files moved apart ([DEC-158](../decisions.md#dec-158-a-module-says-where-it-is-built-and-diff-compares-its-spec-and-its-files)). `diff` compares them module by module with nothing stored, the older state being any copy of the project, and shows a change without judging it: files that changed without their spec may be drift or a rebuild to the same spec, and the tool cannot know which ([§6](the-tool.md#reading-the-spec)).

## Setting up a project

A project is set up by hand: write `.markitect/spec`, copy the default's module `core` into the spec folder, write the project's own modules, each a folder with a README ([§3](the-language.md#modules)), and run `check` until it reports nothing. The [project reference](../guides/project-reference.md#setting-up-a-project-by-hand) gives the steps.

## The default

The default is what ships with Markitect ([DEC-156](../decisions.md#dec-156-a-module-is-a-folder-and-its-readme-says-what-it-shows); [library](../guides/library.md)). Until a new default is designed, it is the one module `core`, with six general types, concept, decision, goal, role, rule and term, and its reference values and question catalog.

- **The module** is plain files in Markitect's repository. A person copies it into the spec, as the folder `core` directly below the spec folder, and the spec owns its copy and may change it like any of its modules.
- **Reference values and the question catalog** are not part of the spec. They are built into the tool and read from its version, so they improve with each version: `topic` shows the reference values beside the project's own figures, and `questions` prints the catalog for a spec that has `core` ([§5](is-the-spec-good.md)).

## The way of working

A way of working is always the project's own, also when it starts from one that a default ships; today the default ships none. The project describes how its people and agents work in its spec, with general types such as Role, Rule and Decision of `core` and types of its own, and skills, agents and scripts realize it, as code realizes the product ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). Markitect ships no skill and no folder for the work.

## The plugin and the version

**The plugin only fetches the tool** ([DEC-135](../decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language)). Its launcher fetches the release of the plugin's own version, which is the tool's version, checks it against the release's checksums and starts it ([the plugin](../guides/plugin.md)). A project pins no version: `markitect version` names the tool's version and build. A release publishes the tool for Linux, macOS and Windows with checksums ([releasing](../releasing.md)).

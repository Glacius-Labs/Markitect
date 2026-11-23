# The project

**A project is the nearest folder upward that holds the file `.markitect/spec`** ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). Its one line names the spec folder, an ordinary folder of documentation such as `docs/spec/`. That file is all Markitect keeps in a project: the tool reads nothing else from `.markitect/`, and nothing creates the file. A person writes it.

Deleting `.markitect/spec` and the tool leaves an ordinary repository with its documentation, code and tests.

## Setting up a project

A project is set up by hand: write `.markitect/spec`, copy the packages and the templates the project starts from into the spec folder, and run `check`. A template's documents link into the packages by relative links, so the two are copied side by side as they lie in Markitect's repository; a template may stay where it was copied, or move once its links are adjusted. `check` then lists the sections the templates leave empty: that list is the work still to do, and once it is done, `check` reports nothing. The [project reference](../guides/project-reference.md#setting-up-a-project-by-hand) gives the steps.

## The default

The default is the packages and templates that ship with Markitect, and per package its reference values and question catalog ([library](../guides/library.md)).

- **Packages and templates** are plain files in Markitect's repository. A person copies them into the spec, which owns its copy and may change it like any of its documents.
- **Reference values and question catalogs** are not part of the spec. They are built into the tool and read from its version, so they improve with each version: `topic` shows the reference values beside the project's own figures, and `questions` prints the catalogs of the packages the spec has ([§5](is-the-spec-good.md)).

## The way of working

A way of working is always the project's own, also when it starts from one that a default ships; today the default ships none. The project describes how its people and agents work in its spec, with the general types of the default, such as Role, Rule and Decision of the `core` package and Process, Step and Task of the `company` package, and skills, agents and scripts realize it, as code realizes the product ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)). Markitect ships no skill and no folder for the work.

## The plugin and the version

**The plugin only fetches the tool** ([DEC-135](../decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language)). Its launcher fetches the release of the plugin's own version, which is the tool's version, checks it against the release's checksums and starts it ([the plugin](../guides/plugin.md)). A project pins no version: `markitect version` names the tool's version and build. A release publishes the tool for Linux, macOS and Windows with checksums ([releasing](../releasing.md)).

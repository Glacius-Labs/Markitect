# What Markitect is

**The spec is a typed, declarative specification language in Markdown** ([DEC-119](../decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it. It is a specification language as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by the tool; its content is prose that an agent interprets. Because an agent can be wrong, its results are checked.

The spec describes the whole project, its way of working included. That way of working is the project's own: Markitect gives the language to state it, and its default a way of working to start from, which the project copies and then owns ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads), [DEC-159](../decisions.md#dec-159-the-default-is-a-whole-spec-to-start-from-and-lies-in-the-plugin)).

## Terraform is the image

Terraform makes infrastructure match a configuration that states what should exist. Markitect does the same for the intent of a project, with an agent where Terraform has its providers:

| Terraform | Markitect | What it is |
|---|---|---|
| Configuration | The spec | What should hold, like an interface ([§4](what-goes-in-a-spec.md)) |
| `validate` | `check` | Is the spec correct in itself? |
| Providers that make it so | Agents that realize it | The project's way of working says how |
| Modules from the registry | The default | Copied into the spec, which owns its copy ([§7](the-project.md#the-default)) |

A compiler would be the other image. Terraform fits better, because the work is reconciling what should be with what is, not translating one into the other.

## Three things

Markitect is three things ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads)).

| Thing | What it is |
|---|---|
| **The language** | Documents with a type and ordinary links, grouped in modules, in plain Markdown ([§3](the-language.md)) |
| **The default** | A whole spec to start from: general types in the module `core`, a way of working, and modules for the work; plain files that a project copies into its spec, and the reference values and question catalog of `core` built into the tool ([§7](the-project.md#the-default)) |
| **The tool** | One program, `markitect`. It checks the spec, analyses it and helps reading it. It never writes, keeps no state and starts no agent ([§6](the-tool.md)) |

## Words

| Word | Meaning |
|---|---|
| **Spec** | The one description of the project that has authority over its artifacts: the target |
| **Document, type, object** | One Markdown file of the spec; a document that declares the properties of other documents, as a class does; and a document that names a type in its type line, as an object of it |
| **Property** | A section that a type declares, with its type: text, references to objects of a type, or fixed values |
| **Reference, mention** | A link in a reference property of an object; and a link anywhere else in the text, a type's own description under a reference property included, which says only that one document concerns another |
| **Dependency, cycle** | What a module has on another when one of its documents names a type there in a type line, an extends line or a property type line, or has a reference to a document there; and a chain of dependencies that comes back to its start, which the language forbids |
| **Module** | A folder of the spec that holds a document directly, with a README that says what it is for, where in the project it is built and what it shows to other modules |
| **Public, internal** | What a module's README lists under `Contents`, which other modules may link; and everything else in the module |
| **Default** | What ships with Markitect for a project to copy: a whole spec with general types, a way of working and modules for the work, and the reference values and question catalog of its module `core` |
| **Artifact** | Any file that comes from the spec: code, tests, configuration, a handbook, a skill. The artifacts are what is |
| **Project** | The folder that holds `.markitect/spec`, which names its spec folder ([§7](the-project.md)) |

## Authority

Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines, configuration and skills alike ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). A project rule found outside the spec, in configuration or agent instructions, belongs in it. Where the spec leaves open something someone will rely on, nothing is guessed: that is a defect of the spec ([§4](what-goes-in-a-spec.md)).

## Removing Markitect leaves an ordinary repository

Nothing of Markitect stands in an artifact or in the spec: no comment in code, no line in a handbook, no front matter. The spec is ordinary documentation, a copied default included, and the only files of Markitect lie in the folder `.markitect/`: `spec`, and optionally `ignore`. Deleting that folder and the tool leaves a repository with its documentation, code and tests ([§7](the-project.md)).

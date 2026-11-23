# What Markitect is

**The spec is a typed, declarative specification language in Markdown** ([DEC-119](../decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it. It is a specification language as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by the tool; its content is prose that an agent interprets. Because an agent can be wrong, its results are checked.

The spec describes the whole project, its way of working included. That way of working is the project's own: Markitect gives the language to state it and nothing of it ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).

## Terraform is the image

Terraform makes infrastructure match a configuration that states what should exist. Markitect does the same for the intent of a project, with an agent where Terraform has its providers:

| Terraform | Markitect | What it is |
|---|---|---|
| Configuration | The spec | What should hold, like an interface ([§4](what-goes-in-a-spec.md)) |
| `validate` | `check` | Is the spec correct in itself? |
| Providers that make it so | Agents that realize it | The project's way of working says how |
| Modules from the registry | Packages and templates of the default | Copied into the spec, which owns its copy ([§3](the-language.md#packages-and-templates)) |

A compiler would be the other image. Terraform fits better, because the work is reconciling what should be with what is, not translating one into the other.

## The parts

Markitect has three parts ([DEC-154](../decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads-the-spec)).

| Part | What it is |
|---|---|
| **The language** | Documents with a type and ordinary links, in plain Markdown ([§3](the-language.md)) |
| **The default** | The packages `core`, `software` and `company` and the templates `project`, `product`, `architecture`, `software` and `company`, with the reference values and question catalog of each package; plain files that a person copies into a spec ([§4](what-goes-in-a-spec.md), [§7](the-project.md)) |
| **The tool** | One program, `markitect`. It checks the spec, analyses it and helps reading it. It never writes, keeps no state and starts no agent ([§6](the-tool.md)) |

## Words

| Word | Meaning |
|---|---|
| **Spec** | The one description of the project that has authority over its artifacts: the target |
| **Document, type** | One Markdown file of the spec, and a document whose sections are the template of other documents |
| **Package, template** | A folder of types, such as `software`; and a folder of documents with empty sections, copied to start a part of the spec |
| **Default** | The packages and templates that ship with Markitect, with their reference values and question catalogs |
| **Part** | A folder of the spec, with a README as its table of contents |
| **Artifact** | Any file that comes from the spec: code, tests, configuration, a handbook, a skill. The artifacts are what is |
| **Project** | The folder that holds `.markitect/spec`, which names its spec folder ([§7](the-project.md)) |

## Authority

Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines, configuration and skills alike ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). A project rule found outside the spec, in configuration or agent instructions, belongs in it. Where the spec leaves open something someone will rely on, nothing is guessed: that is a defect of the spec ([§4](what-goes-in-a-spec.md)).

## Removing Markitect leaves an ordinary repository

Nothing of Markitect stands in an artifact or in the spec: no comment in code, no line in a handbook, no front matter. The spec is ordinary documentation, and the only file of Markitect is `.markitect/spec`. Deleting it and the tool leaves a repository with its documentation, code and tests ([§7](the-project.md)).

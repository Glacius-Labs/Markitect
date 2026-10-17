# Test projects

Versioned project sources for preparing complete Markitect workflows and independent implementation comparisons. Each project guide owns its requirements, provenance, dependency inputs and preparation commands.

## Project catalog

| Project | Requirement sequence | Sources |
| --- | --- | --- |
| [Library r2](library-r2/README.md) | Realize the library with a five-loan limit, then change only the limit to three against the actual prior realization. | [Canonical model](library-r2/project/.markitect/model.yaml), [source provenance](library-r2/sources.yaml) |
| [Readinglog](readinglog/README.md) | Start from the actual add/list application; add finish/status, atomic CSV import and read-only author summaries through complete requirement snapshots. | [Initial model](readinglog/snapshots/00-baseline/model.yaml), [starting application](readinglog/template/realization/app.py), [source provenance](readinglog/sources.yaml) |
| [Roombook](roombook/README.md) | Begin with an empty realization; add booking/listing, idempotent cancellation, then room summaries of active reservations through complete requirement snapshots. | [Initial model](roombook/snapshots/01-book-list/model.yaml), [source provenance](roombook/sources.yaml) |
| [Ordinary documentation comparison](comparison/README.md) | Prepare the same intended requirements for one coding Agent and Markitect, with independent assessment outside both writer contexts. | [Library requirements](comparison/library/requirements.md), [writer instruction](comparison/library/implementation-prompt.md), [matched source bindings](comparison/library/matched-arms.yaml) |

## Prepare a project

Read the project guide before copying inputs into a fresh Git repository. Commit the initial requirement sources, supply every dependency manifest explicitly, and use the guide's structural check and context commands. Keep provenance and operator guidance outside the target repository when the guide specifies that copy boundary.

Inspect the compiled model digest, selected Definitions, provenance and inclusion reasons. Preserve the initial source identity and apply the documented model delta without changing other requirements. Register actual realization artifacts when they are authored.

Structural preparation establishes consumption of the project sources. Implementation behavior, independent assessment, runtime capability and comparative results require their own observed evidence. These files grant no provider, Apply or study execution authority.

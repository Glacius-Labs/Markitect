# Source comparison with current Main

Date: 15 October 2026. Fixed Main basis: `6472d42a93c1be7e542d1fa06ad10a308dd15b8a`. Subject: the design sources fixed in the [consolidation state](consolidation.md). This is a targeted source comparison of the language boundaries, not a complete implementation plan and not a runtime test.

## Actual Main contract

| Area | Finding at Main | Source |
| --- | --- | --- |
| Project vocabulary | The five Kinds are Manager, Statement, Artifact, Check and Decision. Concept, Rule, UseCase, architecture and workflow are described through Statement categories. | [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go) |
| Responsibility | Manager contains parent, owns and instructions. The proposed renaming to Mandate is not implemented. | [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go), [documentation Manager](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/.markitect/model/documentation/manager.yaml) |
| Description | The current definition format has purpose; Statement has spec.description. The proposed common body shape and the symbol syntax are not supported inputs. | [Core types](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/types.go), [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go) |
| References | A Reference property binds a target Kind through Target. The project vocabulary uses fixed Statement, Manager and Check targets. The vNext candidate with mixed target sets needs a separate contract. | [Core types](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/types.go), [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go) |
| Input selection | project.yaml has an explicit modelFiles list and a target path for the generated documentation. This is not the designed recursive model-folder loader. | [Project configuration](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/.markitect/project.yaml) |
| File coverage | Markitect's own project uses coverageMode full. docs/ is covered by the Artifact documentation-pages and owned by the documentation Manager. New design pages fit these existing prefixes. | [Project configuration](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/.markitect/project.yaml), [Artifact](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/.markitect/model/documentation/documentation-pages.yaml), [Manager](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/.markitect/model/documentation/manager.yaml) |
| Checks | Check requires command with at least one argument; uses binds Statements. A described manual check method therefore does not automatically become executable. | [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go) |
| Decisions | Decision.subject is a Statement, Decision.actor is a Manager. A new Actor term cannot take over these fields merely because the names are similar. | [Schema](https://github.com/Glacius-Labs/Markitect/blob/6472d42a93c1be7e542d1fa06ad10a308dd15b8a/src/internal/core/projectmodel/projectmodel.go) |

The Main state now contains the current product directories Core, Infrastructure, Application and Runtime as well as the shared operational Record Store. The [architecture](../../architecture.md) explains these boundaries. An early implementation plan with older package paths must be renewed against these sources before implementation. The [origin explanation](../../concepts/origin-and-rationale.md) is already contained in this Main.

## Consequences for this consolidation

The design documents are brought together on Main without changing the project configuration, the existing schema or the runtime. In particular, `vNext` examples are not treated as compilable Main models. No schema change is smuggled in through a documentation example, and the historical check of 11 October is not presented as new evidence.

The most important implementation questions remain: version and migration, selection and meaning of the Kinds, bodies and symbol resolution, reference target sets, composition and realization scope, responsibility and file accounting, and check methods. The [conflict list](consolidation.md#contradictions-and-decisions) separates these questions by their concrete origin.

This comparison confirms source findings at exactly the stated state. A later implementation must fix its then-current Main again and add matching tests. It is not evidence that the proposed language or any of the example projects already works.

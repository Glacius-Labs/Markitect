# MyMeetings model classification

This is an adopter-owned overlay for the fixed MIT source snapshot at upstream commit 91c8ef24b4cb6ef558c95d8267fa07d68c7059f8. Package versions and resources are pilot inputs, not Markitect defaults. Resource input paths resolve from the adopting project root. The upstream README and ADRs remain unchanged and human-owned.

## Canonical structured knowledge

| Fact | Why the model owns it |
|---|---|
| MyMeetings composes stable BuildingBlocks and five business Modules: Administration, Meetings, Payments, Registrations and UserAccess. | Product composition and Module identities are explicit architectural facts an agent should not have to rediscover in the solution file. |
| A UseCase is a Command or Query, belongs to one Module, owns one Handler, and may name a Feature and changed Aggregate. | These are stable typed relationships. The v1 Domain checks resource kinds, required references, and Handler cardinality. |
| Feature and Aggregate ownership names a Module. | Ownership is meaningful canonical graph data. An explicitly labeled UseCase cohort must resolve to the same Module through its Feature path. |
| Modules declare stable Core and explicit Interface dependencies. | The package centrally disallows direct Module targets through a finite allowed-targets rule. Interface provider and consumer links are explicit. |
| Shared BuildingBlocks concepts have one Core owner. | The model records ownership instead of presenting reusable concepts as duplicated Module-owned contracts. |
| v1 leaves Validators optional. v2 requires a Validator for the two explicitly selected Meetings Commands. | This is a versioned adopter policy change, bounded to an explicit label cohort and visible as PolicyResults. |

The v2 cohort is AddMeetingAttendee and CancelMeeting, both current Commands without a Validator source file. The v2 Domain also checks that the selected cohort has Command intent. The validation label is an explicit policy input; an untagged Command remains outside this rule. The label does not prove complete coverage or correct classification.

## Human-owned narrative documentation

The original README and Architecture Decision Records remain human-maintained. The model does not rewrite them and does not claim to eliminate their content. The Meeting attendance Feature references the CQRS read-model ADR, the existing attendee SQL view, and the architecture-test ADR because they directly inform the selected read-side task. Other features link to relevant existing ADRs. The compiled context should include the selected Feature materials rather than the whole README or every ADR.

SQL, C# and project files are opaque byte inputs. Markitect does not infer query behavior, business meaning or runtime architecture from their contents. The task request and human review own behavior decisions; the actual code and project-owned tests provide implementation evidence.

## Technology-specific observed evidence

The companion .NET project-reference check evaluates direct ProjectReference items for the declared Module Application, Domain and IntegrationEvents projects plus the BuildingBlocks project set. It establishes only whether selected direct cross-Module references target IntegrationEvents projects. It does not prove runtime dependency behavior, business meaning, event usage, transitive dependency policy, dynamically loaded assemblies or the absence of other coupling mechanisms. Module Infrastructure projects are outside this selected source snapshot. Registrations.Infrastructure references to UserAccess projects are consequently outside this check's subject set.

The UserAccess IntegrationEvents project is referenced by some Application projects but has no event source file in this snapshot. The model retains this empty assembly boundary as an observed project reference and does not claim that it contains a message contract. Inputs identify exact source paths; they are not folder-convention rules.

## Knowledge outside this Domain

| Knowledge | Why it remains outside the canonical model |
|---|---|
| Whether GetMeetingAttendeeCount counts rows, guest seats or unique people. | This is task semantics for the adopter to state and verify; the name alone cannot prove intent. |
| Whether a Handler's SQL is correct, safe, performant or read-only. | Requires source review, database execution and project-owned tests. |
| HTTP route, endpoint authorization, identity resolution, DTOs and API behavior. | These belong to the API adapter and implementation. Existing permission attributes show an extension point but do not prove a UseCase-level authorization contract. |
| Database schema, migrations, consistency and transaction behavior. | These require technology-specific evidence and human review. No provider-specific semantics enter Core. |
| Runtime module wiring and event ordering. | ProjectReference proves only direct declared references. ADRs and runtime tests remain evidence for broader claims. |
| Whether Features and UseCases are complete and labels are correctly maintained. | Human review is required; selection labels are not inferred truth. |
| Aggregate fan-out rules, identity-pair allowlists and mixed relation cycles. | This pilot does not encode these gaps or extend Core to cover them. |

## Language pressure and maintenance

The current language models this selected graph with closed kinds, typed references, explicit cardinality, finite selectors, same-target, allowed target kinds, resource-scoped count policy, package pins and opaque artifact inputs. It cannot select every Command by its arbitrary intent property. The v2 label cohort plus an intent guard is readable and deterministic, but a new Command can escape the rule until someone deliberately labels it. This is an explicit coverage-maintenance cost, not a Core feature request during the pilot.

The package and graph add another maintained surface beside the existing README, ADRs and architecture tests. They do not remove those sources. Their possible value is selected compiled context and deterministic package policy. Measure whether that value justifies model and package upkeep. Compare against the simpler AGENTS.md plus existing architecture-test approach. Do not claim generalized productivity, defect or token savings from this one pilot.
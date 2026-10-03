# Candidate C-01 — Explicit immutable Query request and module QueryHandler

## Proposed rule

For new read behavior in a module, represent the request as a typed Query carrying its result type and handle it through that module's `IQueryHandler<TQuery, TResult>` convention. Keep Query inputs immutable. Keep concrete handlers non-public and use the `QueryHandler` suffix, as the Meetings architecture tests require.

Treat this as a project-level convention candidate for owner review, not an adopted Markitect rule. The selected sources do not establish a one-to-one cardinality rule between every Query and Handler, so this candidate does not propose one.

## Scope and classification

Likely intentional, project-specific convention for the sampled modular-monolith code. The accepted CQRS ADR and explicit Meetings architecture tests support intent; currentness of the 2019 read-layer ADR and applicability beyond observed code still need owner confirmation. This does not claim that all modules are identical or that a selected sample proves repository-wide prevalence.

## Supporting evidence

- E-01: Accepted ADR says each business module uses separate read and write models and passes Command or Query objects through the module façade.
- E-03: Meetings Application architecture tests check Query immutability, the `QueryHandler` suffix, and non-public handlers.
- E-04: `GetMeetingGroupDetailsQuery` is a typed Query with a get-only input property.
- E-05: Its internal handler implements `IQueryHandler`, returns the details DTO, and reads it from a database view.
- E-06: The Query's response is a focused details DTO.
- E-07: `GetAllCountriesQuery` is a second typed Query, with no request fields.
- E-08: Its internal handler implements `IQueryHandler` and queries a Meetings read view.

## Counterexamples

- E-10: `GetAllEmailsQueryHandler` follows the same handler interface/visibility/name shape but queries `[app].[Emails]` directly, rather than a module view. This counters any broader claim that every Query must read a view; it does not contradict the narrower request/handler convention.

## Qualifying evidence

- E-02: The accepted read-layer ADR says the module Application layer handles Queries, is coupled to the database/query framework, and does not abstract over the database. It supports the module-layer rule while qualifying proposals to abstract or standardize storage.
- E-09: UserAccess has a parameterless typed Query too, but it is only one cross-module example and does not prove universal application.

## Confidence and basis

Medium for the narrow Query/handler convention in the sampled modules: it is explicitly tested in Meetings and visible in multiple concrete examples. Low for universal future intent, strict one-Query/one-Handler cardinality, all-module coverage, or the ongoing status of old ADRs. The sample is purposive and small; no frequency estimate is valid.

## Alternatives

1. Adopt only the broad CQRS statement and leave request/handler structure to code review.
2. Adopt the narrow convention above, but leave read-store choice to the owning module.
3. Make database views mandatory. E-10 contradicts this broader proposal, so the selected evidence does not justify it.
4. Leave this as observed implementation pending owner confirmation, particularly if the 2019 ADRs are historical rather than current policy.

## Questions for the owner

- Is the Query + module `IQueryHandler` convention still the desired default for new work?
- Should the immutability, internal visibility, and naming checks remain required, or are some local implementation details?
- Is Query-to-Handler one-to-one ownership an intended invariant? The selected test/code does not prove it.
- Are reads allowed to query a view or a table as each module requires? Current evidence demonstrates both.
- Does this decision apply beyond the sampled Meetings and UserAccess code?

## Status

Awaiting explicit human review. No canonical Project resource or package pin has been changed.

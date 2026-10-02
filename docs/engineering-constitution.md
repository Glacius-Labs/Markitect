# Engineering constitution

**Status:** This document records the v0.11.0 product direction and the boundaries preserved by the shipped implementation. v0.10.0 is the historical baseline; v0.11.0 is the current supported contract. The [roadmap](implementation-plan.md) owns verified release evidence; [Usage](usage.md) documents supported syntax and CLI behavior.

## Product purpose

Markitect describes the space of valid engineering changes. People decide which architecture and policies a project wants. Markitect helps express those decisions as canonical, versioned Domains and resources, derives bounded deterministic checks and context from them, and gives agents the relevant constraints. Agents can choose implementations inside those boundaries. Markitect does not choose the architecture, generate arbitrary application code, or decide whether the desired architecture is wise.

The intended outcome is one canonical model whose relevant values can feed generated human-readable Domain contracts, agent instructions, deterministic checks, and explicitly configured adapters. When a Project selects the `markdown` target, the v0.11.0 Domain projection writes normalized Domain schemas, relations, and constraints under `docs/markitect/_domains/`; per-resource projections may include their applicable PolicyResults and explicit waiver decision. These are generated views, not canonical sources. A check establishes only its declared assertions for its fixed inputs. It does not establish that prose is true, that code satisfies business intent beyond the evidence a configured checker can observe, or that a human has accepted a change.

## Reuse existing Domain packages

Architecture contracts should be reusable as long-lived, versioned content packages. A consumer pins an exact package revision and explicitly activates a Domain declared by that package. A later architecture contract version is adopted through a reviewed exact pin update, followed by `check`, fixed-snapshot `impact`, and inspection of policy results. Markitect reports which declared assertions pass, fail, or are waived under the explicit exception contract; it does not rewrite application code or perform a migration automatically. The adopting team decides and implements code changes.

Keep one Domain definition for each active API group/version. Custom typed references stay within that same API version. Consequently, patterns that must compose over the same resource need to be composed in the same Domain definition and its versioned package. Independent traits from separately versioned Domains cannot be stacked onto one `UseCase` instance. A `conformsTo` link by itself does not import or activate constraints. A package update changes the active architecture contract only through the exact Domain definitions explicitly selected in the Project.

Do not add a Core `Pattern` kind or a separate pattern DSL in v0.11.0. Express candidate patterns using the existing Domain kinds, relations, finite assertions, packages, and adapters. Prefer semantic relations over directory conventions. Introduce a new primitive only if repeated concrete contracts show that the existing model cannot express an essential invariant clearly and deterministically. Templates and one-time scaffolding solve project startup; versioned Domain packages solve ongoing conformance. They are different product concerns.

## Example contract: vertical-slice UseCases

A project-owned or package-supplied Software Domain may express a UseCase with an explicit Command-or-Query kind, exactly one owning Module, exactly one required Handler, and declared optional extension points such as Validator, Authorization, Mapping, or EventPublication. It can state that slice-specific implementation belongs to its UseCase and that lateral UseCase dependencies are restricted. The same Domain may distinguish project rules for Modules, Features, Common/shared concepts, and interfaces.

This is an example of a project-selected architecture, not a Markitect default. It does not make a specific Clean Architecture, DDD, folder layout, or .NET convention universal. It also does not imply that a relationship named `conformsTo` activates the linked Domain's assertions. The versioned package must carry the combined Domain contract for the resource kinds it constrains.

The finite constraint language can express cardinality over a single resource or over a selected resource set. v0.11.0 adds a `scope` distinction to `count`: `resource` evaluates relation cardinality independently for each selected subject; `selection` evaluates the selected population as a whole. For example, a per-UseCase exactly-one-Handler invariant is resource-scoped, while a constraint on the number of selected active UseCases is selection-scoped. Selection boundaries must account for additions and removals.

Do not claim that the core inspects source code to prove that a Query does not mutate state, that a Handler file exists, or that tests and documentation are present. Those implementation facts require explicit project-artifact inputs and a configured adapter or project-owned check. Without such evidence, a Domain check establishes only its structural assertion over canonical model data.

## Policy results and explicit exceptions

v0.11.0 exposes a `PolicyResult` for each applicable constraint and subject with status `passed`, `failed`, or `waived`. Results are available through the normalized `model`, `check` and `verify` outputs, and relevant context: results for the context closure's subjects plus an explicit collection for global selection constraints. A result is tied to the Domain and constraint, the exact subject identity when applicable, and the fixed model and configuration inputs. A waived result also exposes the exception's rationale, owner, recorded decision, `expiresOn` when present, and frozen `policyDate`, while retaining the original violation message.

A Project may declare at most 64 `policyExceptions`. Each exception requires `name`, `apiVersion`, `constraint`, the exact subject `GraphKey`, `constraintDigest`, `subjectDigest`, `rationale`, `owner`, and `decision`. `expiresOn` is optional and uses `YYYY-MM-DD`. Evaluating an exception with an expiry requires an explicitly frozen `policyDate` carried as as-of snapshot evidence; ambient wall-clock time must not change a result. The exception expires on the named day: `policyDate >= expiresOn` is expired and fails. The recorded owner and decision are data supplied by the Project. They are not a signature, authenticated identity, or proof that an authorized person approved the deviation.

Exceptions can waive only a failing per-resource constraint for the exact subject whose current digest matches. They cannot waive schema/type errors, malformed or unresolved references, cycles, selection-wide constraints, relation bounds, or other global structural failures. An exception that is malformed, unknown, stale, unused, or expired fails evaluation rather than silently widening the valid architecture. Exceptions are explicit deviations to review, not a second policy language or automatic migration path.

Changing a Domain package can change constraint digests and make existing exceptions stale. The consumer reviews the exact pin update, `impact`, and per-subject results; it then removes, renews, or replaces exception data through an explicit reviewed Project change. Markitect does not carry exceptions forward as if their meaning were unchanged.

## Copy Me: propose, review, adopt

The shipped v0.11.0 authoring resources include a provider-independent `Copy Me` workflow. It helps an agent propose a candidate engineering style from evidence a person explicitly selects. It does not add a `discover` CLI command, call a model from the Markitect kernel, or promote observations into canonical policy.

The workflow must:

1. Freeze the chosen repository revision and exact selected file paths and hashes before analysis. It does not scan an ambient workspace or infer that an unselected file is in scope.
2. Separate observations from hypotheses. Each candidate states its supporting evidence, counterexamples, uncertainty, and the proposed rule; frequency is not correctness.
3. Keep proposed resources in a separate staging candidate, outside active canonical Domains and package pins. The candidate cannot affect ordinary checks, context, projections, or impact as if it were adopted policy.
4. Present the candidate and its evidence for explicit human review. Only a deliberate reviewed change that places the selected canonical definition in scope and updates the exact package pin adopts it.

The workflow may use an authoring agent to read the frozen, selected evidence and prepare a proposal. The Markitect core remains provider-independent and does not interpret source code, mine review history, profile contributors, or infer authorization. Adoption is the human's explicit source change; exception metadata alone does not authenticate acceptance.

## Normal changes and architecture changes

A normal engineering change implements a task within the active architecture contract. An architecture change alters that contract: for example, changing the cardinality of a Handler relation or adding a new extension point. Both use fixed snapshots, declared inputs, and deterministic checks. Architecture changes additionally update or select the canonical Domain package, assess affected subjects with `impact`, inspect PolicyResults and exceptions, update human and agent-facing views through their declared consumers, and receive the adopting team's ordinary review and acceptance.

This workflow makes affected knowledge visible; it does not guarantee all relevant behavior is modeled. Reports describe configured coverage and evidence limits. Product benefits such as fewer missed updates, less reconciliation effort, or safer autonomous implementation remain hypotheses until measured on repeated real tasks.

## Deliberate boundaries

The v0.11.0 implementation does not adopt the archived proposals as an unrestricted backlog. It excludes a Core Pattern primitive, a new architecture DSL, implicit pattern composition, code generation or automatic code migration, automatic exception approval, arbitrary policy code, source-code semantic inference in the kernel, provider-specific discovery, and automatic canonization. Pattern layering, broad architecture inheritance, cross-project behavioral profiling, and long-running background enforcement require separate evidence and product decisions.

## Constraint language and specialist engines

Keep core policy evaluation as a small finite language of deterministic assertions over explicit resources, relations, and selected sets. CUE is a plausible specialist for structural validation, composition through unification, and intentionally incomplete data; OPA Rego is a plausible specialist when policies need richer decisions over structured inputs and external data. Their official documentation describes those respective models: [CUE data validation](https://cuelang.org/docs/concept/how-cue-enables-data-validation/), [working with incomplete CUE](https://cuelang.org/docs/concept/working-with-incomplete-cue/), [OPA policy language](https://www.openpolicyagent.org/docs/policy-language), and [OPA external data](https://www.openpolicyagent.org/docs/external-data).

Do not embed either runtime in Markitect by default. Reconsider a configured specialist adapter only when repeated concrete architecture-policy cases exceed the readable finite language. Evaluate those same cases for authoring clarity, explicit inputs, diagnostics, evidence and operation boundaries. This is an expressiveness decision, not a performance or market claim.

The archived proposals remain useful discussion sources, but their claims and recommendations do not override this adopted scope. The [strategy index](strategy/README.md) identifies which ideas informed the target and which benefits remain unproven.

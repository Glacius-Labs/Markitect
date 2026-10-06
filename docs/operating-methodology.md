# Operating methodology

This is the canonical owner of Markitect's intended engineering method. [Vision](vision.md) owns the product goal, [architecture](architecture.md) and the [engineering constitution](engineering-constitution.md) own technical contracts, [Markitect-first](markitect-first.md) owns the executable change protocol, and [measurement](measurement.md#staged-method-readiness) owns readiness and comparison procedure. The [roadmap](implementation-plan.md) distinguishes delivered capabilities from source candidates and plans. This method describes the target; it is not a claim of complete implementation or measured superiority.

## Desired project world and its representations

Owners define an ontology: the project's selected concepts, relationships, rules, boundaries, responsibilities and purpose. Typed Definitions capture machine-relevant structure; readable prose captures rationale and obligations that cannot usefully be reduced to structural assertions. The language is supplied by selected Schema Modules, not a universal engineering ontology built into Core.

The ontology expresses the desired project world. Governed code, documentation, provider instructions, automation and other artifacts are representations of that world. Their meaning comes from accepted canonical intent, rather than independent copies of the rules in each target. Governance applies to declared ownership and scopes; unmodeled, uncertain, excluded and externally owned artifacts remain visible.

A representation must belong to the set of valid representations of the intent. Different file structures, names or implementations may be equally valid where the policy permits them. Semantic obligations and allowed freedom are explicit; byte equality is appropriate only for targets with an exact deterministic contract. Preserve already valid representations instead of rewriting them to an agent's preference.

The primary expected outcome is better project quality and consistency through fidelity to accepted intent: fewer forgotten obligations, missed consequences, contradictory representations and ignored rules during autonomous work. Owners still assess whether the accepted model itself serves the project goals. Reduced routine human review is a desired consequence of adequate independent evidence and delegated authority. Token and duration savings are secondary, and may be outweighed by the quality benefit or by model-maintenance cost. These are hypotheses to evaluate.

## One authority model, distinct responsibilities

| Responsibility | Operation |
|---|---|
| Intent ownership | Accept desired concepts, rules, purpose and freedom; decide real intent changes, unresolved conflicts and exceptions within the project's authority model. |
| Execution | Consume applicable canonical context, derive and implement affected work, produce a bounded candidate, and repair failed representations without redefining success. |
| Independent verification | Assess the candidate against accepted obligations and evidence appropriate to the target; expose violations, missing coverage and uncertainty; return pass, repair or an owner decision. |

These are responsibilities, not three additional Core primitives or a requirement to run three humans. An Executor must not be the sole judge of its own semantic success. A Verifier needs the accepted obligations and relevant candidate evidence, including integration behavior, rather than only tests authored by the Executor. Deterministic tools can provide evidence within this same process; an agent label or a second model call alone does not establish independence or sufficient coverage.

Humans define the architecture and delegation boundary and resolve decisions that cannot be delegated. Routine work can proceed within accepted intent without a human inspecting every PR. Technical verification remains bounded evidence and does not silently grant authority or stand in for required human acceptance.

## The change and reconciliation cycle

1. **Establish intent and scope.** Resolve the fixed starting revision, relevant Definitions, policies, target ownership, tools and applicable agent process through Markitect. Classify the task as an intent change, unchanged-intent implementation or repair, migration, or an unresolved owner decision. A bug fix does not need a fabricated ontology edit.
2. **Resolve real intent changes first.** Update the canonical owner before its corresponding implementation. Compile the intent candidate, inspect impact and conflicts, and obtain decisions required by the existing delegation policy. Observed artifacts may inform a proposal, but cannot silently become accepted intent.
3. **Plan affected work.** Derive reconciliation from canonical delta, explicit dependencies, observed drift and representation obligations. Separate the semantic graph from the execution DAG. Identify affected representations, parent obligations, checks, relevant context and uncertain scopes before execution. Unknown impact remains conservative rather than disappearing from the plan.
4. **Execute bounded candidates.** Each Executor receives the obligations and freedom for its target. Desired representation intent, installed target capabilities and operational plans/records remain distinct. Schema installation only supplies language; Projection Module installation only supplies capabilities. Project configuration selects the concrete implementation; installation itself does not materialize artifacts.
5. **Verify and compose.** Independent verification checks local obligations and the candidate's declared changed surface. Compose child results into a parent candidate and verify the parent's own obligations; passing children do not establish integration correctness. Failed candidates return for bounded repair; unresolved intent, ownership or authority returns to its owner.
6. **Apply and account.** Apply only a candidate meeting the declared pre-Apply checks and any required owner decision through the declared trust boundary, with inputs and target scope still current. Apply remains materialized-unverified until the required verification completes against the exact final revision or target state. The final backward gate compares the actual changed surface with pre-implementation intent, ownership, plan and evidence. Preserve stale-plan refusal, artifact accounting and explicitly incomplete coverage. Unexplained changes, stale plans and required-but-missing evidence must leave the gate failing or incomplete rather than report completion. Correct code with stale intent, unexplained owners or a post-hoc rule change is not a complete governed outcome. This is the target completion rule, not a claim that a current CLI command enforces its entire semantic scope.
7. **Observe subsequent drift.** When representations depart from unchanged intent, repair them against that intent. Intent evolution starts a new intent-first cycle. Retain findings and evidence so that forgotten work, missing fanout and ignored process remain assessable.

This is a target process. The source CLI and adapters cover only the boundaries documented in their usage and roadmap; some steps require agent orchestration or project-owned checks. Do not invent a supported command or an implemented controller merely because the method names a step.

## The same process at increasing scope

Use Executor, candidate and independent Verifier at every level: a bounded representation, a complete vertical slice, a composed subsystem and the governed project. Parent execution may coordinate children; parent verification evaluates its own contracts and integration outcomes. It does not merely collect green child reports, and does not need a separate privileged Integrator role.

Choose decomposition by ownership, dependencies and concrete obligations. Do not force every tiny edit through an elaborate hierarchy. Increase verification scope when integration introduces a material new obligation. A declared graph edge, a policy requiring examination of a broader surface and an unexplained changed artifact are different signals; retain each instead of inferring dependencies from README links.

For example, changing an ordering invariant may affect an API, billing behavior, documentation and provider guidance. Markitect should route the relevant intent and obligations into those representations, reconcile their affected scopes, then verify the combined ordering behavior. Local unit tests alone cannot show that an unchanged billing module still respects the new invariant.

## Rules, process and evidence

Evaluate both precise and broad rules. A precise constraint may have a deterministic assertion or architecture test. A broad obligation such as consistent error handling or respecting module boundaries needs a declared surface, concrete interpretation and suitable independent evidence; selected examples do not establish universal compliance. Do not turn broad guidance into an arbitrary exact implementation prescription merely to make scoring easy.

Check agent behavior as well as artifacts: did the actor start from Markitect, consume applicable context, handle intent before implementation, use declared ownership and reconcile affected work? A compliant final file does not prove that the process was followed. A tool call alone does not prove that its context influenced the work. Preserve observable evidence and mark unavailable evidence as unknown.

Use independent defect cases to show that checks can reject the failure modes they address: missed fanout, stale model, conflicting guidance, a local rule violation, a broad rule violation or an integration defect despite passing children. Controlled defect injection proves only that particular detection boundary; real repeated agents are needed to assess how often the method works. Do not require an escalation when the supplied intent already permits a valid implementation.

## Safe adoption of an existing project

Existing reality can be inspected to infer candidate intent. Owners review that proposal, accept the canonical model, match exact scopes and verify existing representations before adoption. Preserve valid bytes. Record unknown ownership, excluded artifacts and insufficient evidence explicitly. Do not modify the model to bless accidental drift, assume installation governs an entire repository, or require regeneration solely to demonstrate activity.

## Design references

The following are design lenses for this method, not imported guarantees or additional Core semantics.

| Reference | Design question it helps answer |
|---|---|
| Legislative, executive and judicial separation | Who owns intent, who changes artifacts, and who independently evaluates conformity? |
| Mission command and management | Can an Executor act from purpose, boundaries and delegated responsibility without detailed micromanagement? |
| Clean Architecture and DDD | Which concepts, ownership boundaries, dependencies and invariants belong in the project model? |
| Clean Code | What implementation quality obligations should apply within the allowed design freedom? |
| npm-style packages | How can reusable language and target capabilities have explicit versions and dependency boundaries? |
| Kubernetes and Terraform desired state | How can declared intent, observed state, planned change and reconciliation remain distinct and inspectable? |

Use a reference only where it solves a concrete problem. Keep project-specific engineering semantics in selected language and policy above the minimal structural Core.

## From method to usable capability

Follow the [staged readiness procedure](measurement.md#staged-method-readiness): define obligations, verify component boundaries, run one complete real change, test recursive composition, then establish bounded repeated usability before a comparative project backlog. Each stage names its current implementation gap, independent evidence and finite exit condition. Historical experiments retain their original meaning.

A missing accepted capability is implementation work. A harness problem is a measurement limitation until its effect is understood. A counterexample to the architecture or unsupported authority decision is a reason to reassess that decision. Passing the declared readiness criteria is a reason to move on, rather than expand the harness indefinitely. The eventual comparison must let a well-equipped conventional approach win.

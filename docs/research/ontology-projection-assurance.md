# Markitect Research Thesis
## Ontology, Projection, Assurance, and Autonomous Engineering

**Status:** Living research/architecture direction  
**Purpose:** Canonical working thesis for the direction Markitect should investigate.  
**Audience:** Markitect Coordinator, implementers, reviewers, researchers, project owners.

---

# 1. Executive thesis

Markitect should not be designed as a better collection of prompts, rules, documentation files, templates, or linters.

The stronger thesis is:

> **Markitect is the canonical desired-state model of a software engineering system.**

The repository is then a collection of **representations / projections** of that model:

- source code;
- tests;
- documentation;
- CI/CD;
- Git hooks;
- agent rules;
- skills;
- workflows;
- schemas;
- configuration;
- migration artifacts;
- provider-specific instructions;
- architecture checks;
- other project-owned engineering artifacts.

Humans primarily modify **engineering intent**.

Deterministic tools and AI agents materialize that intent into appropriate projections.

Independent checks, analyzers, tests, and AI reviewers then provide evidence that each projection still represents the canonical intent sufficiently well.

The intended lifecycle is:

```text
DEFINE
  ↓
PLAN
  ↓
PROJECT
  ↓
OBSERVE
  ↓
VERIFY
  ↓
INTEGRATE
  ↓
RECONCILE
```

The repository is therefore not an independent competing source of engineering truth.

It is the materialized state of an engineering model.

---

# 2. Research hypothesis

The central research hypothesis is:

> **AI makes it practical to treat software repositories as non-deterministic projections of a canonical declarative engineering model, because implementation variation no longer has to be exhaustively encoded in deterministic generators—provided that sufficiently strong evidence can verify that each projection still satisfies the intended semantics.**

Traditional model-driven development was often trapped between two bad options:

```text
Model too abstract
→ deterministic generator cannot handle real-world variation

Model too detailed
→ the model becomes another programming language
```

AI introduces a third possibility:

```text
Semantic intent
+ constraints
+ project context
+ allowed degrees of freedom
        ↓
AI searches implementation space
        ↓
Candidate projection
        ↓
Independent verification
```

Formally:

```text
Traditional generation:
R = f(M)

Agentic projection:
R ∈ ValidRepresentations(M)
```

The difficult problem shifts from:

> "Can we deterministically generate the exact bytes?"

to:

> "Can we define the valid solution space and gather enough evidence that an implementation belongs to it?"

---

# 3. Single semantic authority

A normative engineering fact should have one semantic owner.

Today a rule such as:

> Modules must not depend directly on other modules.

may be represented independently in:

- architecture documentation;
- AGENTS.md;
- Claude instructions;
- Codex skills;
- architecture tests;
- project-reference structure;
- CI;
- PR templates.

This is duplicated authority.

The desired model is:

```text
Canonical Markitect intent
        ↓
 ┌──────┼────────┬─────────┬──────────┐
 ↓      ↓        ↓         ↓          ↓
Docs   Code    Tests      CI      Agent guidance
```

These downstream forms may use different technical mechanisms, but they do not independently redefine the rule.

This is **single semantic authority**, not "one giant YAML file."

Canonical intent may remain physically split into many small files and packages.

---

# 4. Multiple levels of detail

Markitect must support different levels of abstraction.

A project manager may care about:

```text
Product
Capability
User
Use Case
Acceptance Criteria
Business Rule
Responsibility
```

An architect may care about:

```text
Bounded Context
Module
Aggregate
Contract
Dependency
Invariant
Policy
```

An implementation projector may need:

```text
Handler
Validator
Persistence mapping
API representation
Source artifact
Test obligation
```

A user may need only:

```text
What does this feature do?
How do I use it?
```

These are not competing truths.

They are different views and refinements of the same engineering system.

Markitect therefore must not force every concept to the same granularity.

It needs a graph in which concepts can be:

- decomposed;
- refined;
- related;
- scoped;
- owned;
- constrained;
- projected;
- verified independently.

A high-level concept should be able to depend on more detailed concepts without every consumer receiving the entire lower-level graph.

---

# 5. Description and semantics

Not everything should become a deterministic schema.

Some knowledge is naturally structured:

```text
kind
owner
relations
cardinality
state
policy
constraint
```

Other knowledge is naturally descriptive:

```text
purpose
rationale
examples
usage guidance
historical context
trade-offs
explanation
```

Markitect should support both.

The intended model is:

```text
human-readable description
        +
typed semantic structure
        +
explicit relationships
        +
verification obligations
```

Free prose does not magically become executable truth.

Instead, prose can remain canonical descriptive intent and be evaluated semantically by appropriately scoped AI verifiers.

Example:

```text
Dossier
Purpose:
  Represents the durable case-level unit of work for one proceeding.

Relations:
  contains ChangeRequests
  belongs to Proceeding

Rationale:
  Change Requests describe proposed change; Dossier owns accepted case state.
```

A verifier may later receive:

```text
canonical Dossier description
+
selected implementation projection
+
acceptance rubric
```

and answer a narrow question:

> Does this implementation still correspond to the declared Dossier meaning?

This is probabilistic semantic evidence, not a formal proof.

---

# 6. The ontology kernel

Markitect should provide a small, powerful, extensible ontology kernel.

The Core should not know concepts such as:

```text
Validator
Aggregate
Command
Query
Repository
Controller
Dossier
Mission
GitHub Actions
C#
Java
Claude
Codex
```

Projects and packages must be able to define their own vocabulary.

The fundamental substrate should be closer to:

```text
Type / Kind
Property
Value
Reference
Relation
Identity
Constraint / Assertion
Scope
Ownership / Authority
Provenance
Claim / Obligation
Evidence
Projection Contract
Observed Representation
```

The exact primitive set must be proven rather than assumed.

The design criterion is:

> **Can arbitrary engineering vocabularies be expressed without teaching Core their domain-specific nouns?**

For example, a project should be able to define:

```text
Kind: Validator
Purpose:
  Validates Command input before Handler execution.

Relations:
  validates -> Command
  executedBefore -> Handler

Constraints:
  selected Commands require at least one Validator
```

without a Core change.

Another project should be able to define:

```text
Dossier
Mission
EffectAxis
Questionnaire
```

the same way.

---

# 7. Semantic types need meaning, not only shape

A useful engineering type is more than a YAML schema.

A type may need:

```text
IDENTITY
What is this?

PURPOSE
Why does it exist?

STRUCTURE
Which properties does it have?

RELATIONSHIPS
What may it refer to?

INVARIANTS
What must always hold?

USAGE SEMANTICS
How is it intended to be used?

LIFECYCLE
How does it evolve?

AUTHORITY
Who may change it?

PROJECTION EXPECTATIONS
Where/how should it appear?

VERIFICATION EXPECTATIONS
What evidence demonstrates correct realization?
```

This is closer to an engineering ontology than a data-schema system.

---

# 8. Separate semantic meaning from projection strategy

A semantic concept must not be coupled to one implementation technology.

Example:

```text
SEMANTIC INTENT

Validator
validates -> Command
must execute before Handler
```

Possible projections:

```text
.NET:
FooValidator class + pipeline behavior

Java:
FooValidator implements Validator<Foo>

Go:
ValidateFoo(...) + explicit invocation
```

Therefore:

> **Domain / Ontology defines what a concept means.**

> **Projection Contract defines how that meaning is represented in a particular engineering environment.**

The same semantic model may support multiple projection strategies.

This separation is what makes architecture or language changes theoretically possible without changing domain meaning.

---

# 9. Projection Modules

Every projection mechanism should live outside the generic kernel.

A Projection Module may:

- bring reusable ontology packages;
- define projection contracts;
- interpret selected semantic concepts;
- provide deterministic renderers;
- provide AI projector instructions;
- observe target artifacts;
- provide verification/evidence;
- reconcile drift.

Examples:

```text
Markdown Projection Module
Agent Rules Projection Module
.NET Projection Module
Java Projection Module
Go Projection Module
GitHub Actions Projection Module
Azure Pipelines Projection Module
Git Hooks Projection Module
```

A Module is not required to support every type.

It should explicitly declare what it can project, observe, and verify.

A project explicitly activates semantics and projection capabilities.

A module must not silently make its own ontology canonical merely because it is installed.

---

# 10. AI agents are projection adapters

For many artifacts a deterministic renderer is enough:

```text
Rule → Markdown
Rule → generated provider instruction
```

For complex implementation this is not practical.

The coding agent becomes a **non-deterministic projection adapter**.

Input:

```text
relevant canonical concepts
relationships
constraints
acceptance obligations
projection contract
neighboring interfaces
allowed degrees of freedom
```

Output:

```text
one valid implementation candidate
```

The agent may choose:

- local algorithm;
- helper structure;
- private naming;
- implementation technique;
- refactoring details;

provided those choices remain inside declared freedom.

The agent must not silently introduce new semantic intent.

If implementation requires a new concept, rule, relationship, responsibility, or policy, control returns to the canonical model first.

---

# 11. One universal agent operating model

Markitect originally projected all project rules/processes into provider-specific Skill/Rule/Agent folders so agents would "learn" the project.

The stronger direction is:

> **Teach agents how to work through Markitect once.**

Provider-specific bootstrap guidance should be thin and stable.

Conceptually:

```text
AGENT BOOTSTRAP
  "Use Markitect as the engineering authority."
        ↓
Markitect resolves:
  task context
  applicable concepts
  rules
  processes
  projection contracts
  checks
  escalation conditions
```

This reduces duplicated provider-specific governance.

Claude, Codex, and future providers should not each require a separately synchronized copy of the project's engineering constitution.

Provider adapters may still be needed for invocation, capabilities, or formatting, but the project semantics remain in Markitect.

---

# 12. Context as delegation

Context is not merely a convenient document bundle.

Its stronger role is:

> **Compile the smallest relevant slice of desired engineering state required by one actor to perform one delegated responsibility.**

This supports hierarchical work.

A junior-like implementation agent receives:

```text
one task
relevant concept subtree
interfaces
constraints
acceptance obligations
allowed freedom
```

A senior-like integration agent receives:

```text
subproject results
integration contracts
cross-boundary obligations
conflicts
evidence reports
```

A verifier receives only:

```text
one claim / obligation
canonical expectation
candidate representation
verification rubric
```

This is hierarchical context minimization.

---

# 13. Divide and conquer

Large-project verification should be recursively decomposable.

```text
Project
  ↓
Product
  ↓
Module
  ↓
Feature
  ↓
Use Case
  ↓
Projection
  ↓
Artifact
```

The hierarchy need not be fixed to these names.

The important property is that responsibilities can be decomposed into bounded obligations.

Each lower-level unit can be evaluated independently.

Then higher-level integration checks evaluate whether individually valid parts remain valid when composed.

Important principle:

```text
valid(A)
AND
valid(B)
```

does not imply:

```text
valid(A + B)
```

Therefore Markitect needs explicit **integration obligations**.

---

# 14. Obligations

A projection should expose what must be true for it to count as an acceptable representation.

Call these **Obligations** (name subject to design).

Example:

```text
Projection:
CreateOrder → .NET implementation

Obligations:
- Handler exists.
- Handler belongs to Orders.
- Validator executes before Handler.
- Duplicate product IDs are rejected.
- No forbidden module dependency exists.
- Declared public contract remains compatible.
```

Each obligation selects an evidence strategy.

Example:

```text
Handler exists
→ deterministic .NET structural analyzer

No forbidden dependency
→ architecture test

Duplicate IDs rejected
→ executable behavioral test

Documentation matches concept meaning
→ scoped AI semantic verifier
```

This is more precise than asking one model:

> "Is this whole project correct?"

---

# 15. Verification hierarchy

A central research hypothesis is that many small verification jobs may be more reliable and cheaper than one huge review.

Example:

```text
Canonical Rule R
     ↓
Projection 1 ─→ verifier 1
Projection 2 ─→ verifier 2
Projection 3 ─→ deterministic check
Projection 4 ─→ verifier 4
     ↓
Integration verifier
     ↓
Rule-level assurance report
```

Strong but relatively inexpensive models can receive highly bounded contexts and evaluate questions such as:

```text
Does this UseCase still implement the canonical acceptance behavior?

Does this documentation still describe the canonical Dossier meaning?

Does this Skill instruct the agent consistently with Rule R?

Do these two valid projections conflict when integrated?
```

Uncertain or conflicting results can escalate:

```text
cheap verifier
    ↓
high confidence
→ accept evidence

uncertain
    ↓
stronger verifier
    ↓
still ambiguous
    ↓
human
```

The goal is not model cheapness by itself.

The goal is to spend expensive reasoning only where lower-level evidence cannot resolve the question.

---

# 16. Separation of powers

Do not let one actor create all evidence for its own work.

Avoid:

```text
Agent implements.
Same agent writes tests.
Same agent interprets tests.
Same agent declares success.
```

Prefer independent roles:

```text
Intent owner / legislative authority
        ↓
Projection agent / executive
        ↓
Independent verifier / judiciary
        ↓
Integration verifier
        ↓
Gate / assurance orchestrator
```

This reduces correlated reasoning errors.

Different evidence mechanisms should be preferred where possible:

```text
compiler
static analyzer
architecture test
behavioral test
property test
runtime probe
independent AI evaluation
human decision
```

---

# 17. Assurance case

For each relevant concept or change, Markitect should eventually be able to build an assurance report.

Example:

```text
CreateOrder — convergence report

Semantic model                         PASS
Handler projection                     PASS
  evidence: .NET structural analyzer

Validator projection                   PASS
  evidence:
    structural analyzer
    behavior test

Architecture boundary                  PASS
  evidence: architecture test

Documentation projection               PASS
  evidence: scoped semantic reviewer

Inventory integration                  PASS
  evidence:
    contract tests
    integration verifier

Unknown obligations                    0
```

Or:

```text
CreateOrder — INCOMPLETE

5/6 required obligations have acceptable evidence.

UNKNOWN:
Documentation rationale may conflict with changed retry semantics.

Escalation:
semantic review required
```

This can allow a human to review **evidence and decisions** rather than every implementation diff.

---

# 18. Evidence strength

"Verified" must not imply mathematical software correctness.

Evidence can have different strengths.

Conceptually:

```text
EXISTENCE / OWNERSHIP EVIDENCE
STRUCTURAL STATIC EVIDENCE
EXECUTABLE BEHAVIOR EVIDENCE
SEMANTIC AI EVIDENCE
INTEGRATION EVIDENCE
HUMAN DECISION
```

Exact levels should not be prematurely fixed.

Every evidence item should say:

```text
what it claims
what input it observed
what method produced it
what it does NOT prove
how fresh it is
which obligation it supports
```

Unknown remains a valid result.

---

# 19. Coverage

"Coverage" should not be a naive single percentage.

At least three dimensions are useful:

## Semantic coverage

How much canonical intent has explicit machine-addressable concepts / rules / obligations?

## Projection coverage

For each concept or rule, which intended representation surfaces are declared?

Example:

```text
Rule R
├── docs               declared
├── Codex              declared
├── Claude             declared
├── CI                 declared
└── architecture test  declared
```

## Verification coverage

For each declared projection, what evidence mechanism verifies it?

Example:

```text
                 projected   verified   evidence
Docs             yes         yes        AI semantic review
Codex            yes         yes        AI semantic review
Claude           yes         yes        AI semantic review
CI               yes         yes        deterministic parser
Arch test        yes         yes        executable test
```

A fourth dimension may be useful:

## Evidence strength / confidence

A rule projected five times but verified only by weak self-review should not look equivalent to one supported by independent executable evidence.

Coverage should therefore be a **matrix**, not a vanity score.

---

# 20. Convergence

A governed scope is converged when, relative to its declared obligations and evidence requirements:

```text
canonical model is valid
required projections exist
governed artifacts are accounted for
projection evidence satisfies configured obligations
no stale representations are known
no observed projection contradicts desired intent
no required evidence is missing
no unresolved authority decision remains
```

Converged means:

> **The declared engineering assurance contract is satisfied.**

It does NOT mean:

> "The software is objectively bug-free."

---

# 21. Drift

Drift is:

> **A materialized representation that no longer sufficiently corresponds to the canonical desired engineering state.**

Examples:

```text
Markitect:
Command requires Validator

Code:
Validator absent
→ drift
```

```text
Markitect:
Rule applies to Codex and Claude

Claude:
updated
Codex:
stale
→ drift
```

```text
Markitect:
Release requires changelog

Pipeline:
does not enforce it
→ drift
```

```text
Markitect:
Dossier meaning says X

Documentation:
still says old Y
→ semantic documentation drift
```

Drift may be detected deterministically or probabilistically depending on the projection.

---

# 22. Artifact status

Within a governed root, every artifact should be explicitly classifiable as one of:

```text
CANONICAL_MARKITECT_SOURCE
PROJECTED_REPRESENTATION
DECLARED_EXTERNAL_INPUT
TOOL_OWNED
THIRD_PARTY / VENDORED
EXPLICIT_EXCLUSION
```

Unexplained project-owned artifacts are drift.

Artifact Coverage therefore becomes part of the projection system, not merely repository hygiene.

---

# 23. Refactoring

A refactoring may leave semantic desired state unchanged while changing projection strategy.

Example:

```text
Semantic model:
unchanged

Projection strategy:
Layered Architecture
        ↓
Vertical Slice
```

Markitect can then:

```text
plan affected projections
→ delegate migration
→ verify old semantic obligations
→ verify new structural obligations
→ converge
```

This makes large refactorings conceptually a re-projection problem.

---

# 24. Language migration

A language migration may also be modeled as re-projection where semantics permit.

```text
Canonical model:
unchanged

Projection:
dotnet-csharp
      ↓
java-spring
```

The system does not mechanically transpile code.

Instead it asks agents to materialize a new representation of the same semantic model and verifies:

```text
behavior obligations
contracts
integration behavior
architecture rules
```

This is much more general than a source-to-source converter.

---

# 25. Architecture strategies

Some architecture decisions may become reusable projection strategies.

Example:

```text
same semantic domain
        ↓
Projection A: layered
Projection B: vertical slice
Projection C: ports/adapters
```

This allows controlled architecture experiments where the canonical business meaning remains stable and implementation properties are compared.

Not every architecture decision is "only projection."

If an architecture choice changes authority, semantics, operational behavior, or explicit contracts, that belongs in canonical intent.

---

# 26. Reusable architecture packages

Packages such as:

```text
DDD
Clean Architecture
Hexagonal Architecture
CQRS
Vertical Slice
Company Engineering Standard
```

should not primarily be source templates.

They can provide combinations of:

```text
ontology vocabulary
semantic relations
constraints
processes
projection contracts
verification obligations
reference strategies
```

Projects explicitly activate and compose what they need.

Composition should remain explicit.

Avoid inheritance-heavy meta-frameworks.

---

# 27. Management / government analogy

Use these analogies only where they illuminate an actual engineering problem.

```text
Constitution
→ fundamental authority/invariants

Legislature
→ canonical desired intent

Executive
→ deterministic/AI projection agents

Judiciary
→ independent verification

Audit
→ provenance, evidence, coverage

Federalism
→ bounded contexts/modules with local autonomy

Mission Command
→ intent + constraints + local freedom

Chain of Command
→ hierarchical decomposition and escalation

Amendment process
→ versioned architecture/policy evolution
```

The important lesson is separation of authority and responsibility.

---

# 28. Mission command

A projector should receive the equivalent of commander's intent:

```text
goal
purpose
constraints
interfaces
available resources
allowed freedom
acceptance obligations
escalation conditions
```

It should NOT receive an unnecessarily prescriptive implementation recipe.

This maximizes autonomy without transferring architectural authority.

---

# 29. Markitect as the universal agent interface

Instead of synchronizing the entire project's governance into every provider's native rules/skills system:

```text
Claude knowledge copy
Codex knowledge copy
future-provider knowledge copy
```

prefer:

```text
Thin provider bootstrap
        ↓
"Work through Markitect"
        ↓
Markitect supplies task-specific canonical context
```

Provider-specific projection remains useful where the provider requires special capabilities or ergonomics.

But provider files should increasingly be interfaces to Markitect rather than replicated truth stores.

---

# 30. What is Core vs Tooling

The conceptual Core is:

> **the smallest ontology + relationship + constraint + projection/evidence substrate required to define, project, observe, and reason about engineering intent.**

Everything else should be replaceable Tooling / Modules:

```text
language projectors
documentation projectors
provider adapters
CI projectors
Git integration
agent orchestration
UI
benchmark harness
specific verifier agents
source analyzers
release automation
```

Do not confuse:

```text
important capability
```

with:

```text
belongs in Core
```

The kernel should remain small enough to reason about rigorously.

---

# 31. Research questions

The strongest vision depends on four major questions.

## RQ1 — Expressiveness

Can we capture enough engineering intent without recreating the implementation language?

## RQ2 — Projection

Can AI agents reliably materialize useful technical representations from the model?

## RQ3 — Verification

Can deterministic tools plus small scoped AI reviewers provide enough evidence that projections still satisfy intent?

## RQ4 — Economics / human attention

Does:

```text
modeling
+ projection
+ verification
+ maintenance
```

cost less scarce human attention than:

```text
direct implementation
+ duplicated documentation/rules
+ synchronization
+ code review
+ drift repair
```

The fourth question determines whether Markitect is useful rather than merely possible.

---

# 32. Falsification criteria

The strong Markitect hypothesis is weakened if:

1. canonical modeling becomes as detailed as implementation;
2. model maintenance exceeds synchronization cost;
3. AI projections frequently violate intent without detection;
4. humans still need to inspect most code diffs;
5. new technologies repeatedly require Core changes;
6. custom engineering vocabularies cannot remain external;
7. impact is consistently too broad to support bounded work;
8. multiple competing semantic authorities reappear;
9. authoring becomes YAML/meta-model bureaucracy;
10. engineers need deep Markitect internals for ordinary intent changes;
11. scoped cheap verifiers are not sufficiently reliable;
12. integration verification dominates the cost saved by decomposition;
13. generated assurance creates dangerous false confidence.

These should be actively tested rather than hidden.

---

# 33. Experiments

Future validation should include:

## Greenfield materialization

Start primarily from Markitect intent.

Project:

```text
code
tests
docs
CI
agent guidance
```

Verify convergence.

## Multi-agent projection

Give the same desired state to several independent agents.

Allow implementation diversity.

Measure whether all satisfy the same obligations.

## Language swap

Project the same semantic model into two or more implementation stacks.

Compare behavior/contracts.

## Architecture swap

Keep domain intent stable while changing projection strategy.

Measure migration quality and invariant preservation.

## Drift injection

Introduce controlled drift into:

```text
code
docs
provider rules
pipeline
hook
tests
```

Measure detection and repair.

## Verification hierarchy

Compare:

```text
one expensive global reviewer
vs.
many scoped cheaper reviewers + integration verifier
```

Measure precision, recall, cost, escalation rate, and correlated errors.

## Longitudinal autonomous development

Run many changes without human code review.

Measure:

```text
drift
correct escalations
missed obligations
maintenance
human intervention
```

## Mutation testing for assurance

Systematically violate one obligation at a time.

Verify that the intended evidence chain detects it.

---

# 34. Immediate architecture direction

The Coordinator should treat this document as direction, not as permission to add every named abstraction immediately.

First:

1. compare current architecture to this thesis;
2. identify which capabilities already fit;
3. identify hybrid-authority escape hatches;
4. identify the minimum missing semantic primitives;
5. preserve the small generic Core;
6. prefer external Domains/Packages/Modules;
7. prototype projection/obligation/evidence flow on small projects;
8. test before broadening the language.

Do not introduce a new Core primitive merely because this document names a concept.

The concept must survive multiple independent use cases.

---

# 35. Working product definition

A concise current definition:

> **Markitect is an extensible engineering-ontology and assurance system that makes project intent canonical, projects that intent through deterministic tools and AI agents into repository representations, and continuously gathers evidence that those representations remain consistent with the desired engineering state.**

A shorter form:

> **Define intent once. Project it everywhere. Verify every representation. Reconcile drift.**

And the long-term programming model:

> **Humans program the engineering system. Agents program its projections.**

# Markitect — Product Definition, Vision and Core Capabilities

## Purpose

This document summarizes the current product direction for Markitect based on the original problem that led to it.

The central idea is no longer merely:

> Keep Claude, Codex, Skills, Rules and documentation in sync.

The broader problem is:

> **Engineering knowledge and engineering policy become operationally important, but are usually duplicated across documentation, AI instructions, CI rules, repository structure and external tools. Once that happens, every change becomes expensive, error-prone and difficult to keep consistent.**

Markitect should address this by making engineering intent canonical, explicit, structurally validatable and safely changeable.

---

# 1. Origin of the problem

The project originated from a practical engineering situation:

1. A complex product was being developed quickly.
2. Module boundaries and product boundaries became blurred.
3. A previously simple questionnaire capability had to be extracted and modularized.
4. Architecture, UX, process and ownership concerns began to diverge.
5. Work needed to move from ad-hoc development toward explicit processes and work items.
6. Engineering rules, release processes, module rules, documentation rules and agent instructions were documented.
7. The same rules had to exist in:
   - human documentation,
   - Claude instructions,
   - Codex instructions,
   - Skills,
   - Rules,
   - Workflows,
   - Agents,
   - CI/pipeline checks,
   - repository conventions,
   - Azure DevOps processes.
8. Over time these representations drifted.
9. Consolidation runs became expensive in time and tokens.
10. Every consolidation still found contradictions, stale references or duplicated knowledge.

The root issue was therefore not documentation alone.

It was:

> **The same engineering truth existed in many representations without a compiler or canonical owner.**

---

# 2. Core product definition

A useful product definition is:

> **Markitect is an engineering knowledge and change-management compiler.**

More explicitly:

> Teams define architecture, processes, rules, contracts and AI-working knowledge as canonical resources. Markitect validates their relationships, determines the impact of changes, and projects the relevant knowledge into human documentation, AI-agent instructions, checks and external engineering systems.

A shorter formulation:

> **Define engineering knowledge once. Keep every human and AI view consistent through change.**

Another strong formulation:

> **Markitect compiles engineering intent into consistent documentation, agent instructions and enforceable project policy.**

---

# 3. What is canonical?

The canonical unit should not be the generated Markdown document, the Claude rule, the Codex Skill or the pipeline validation.

The canonical unit is the underlying **engineering rule, concept or intent**.

For example:

```text
"Every module may only depend on Core."
```

That statement should have one canonical owner.

From that one statement, Markitect may produce or support:

```text
Human documentation
Codex instructions
Claude instructions
Architecture validation
CI diagnostics
Repository checks
External policy validation
```

The key principle is:

> **One engineering statement, one canonical owner, many projections.**

This is the engineering-knowledge equivalent of avoiding duplicated logic in software.

---

# 4. Example: module policy

A real engineering policy may state:

```text
When a new module is created:

- it must have an Axis,
- an Azure DevOps Area must exist,
- its project must follow a naming convention,
- it may only depend on Core,
- its documentation must live under modules/<axis>/...,
- it must contain README.md,
- its README must contain an index,
- it must contain intent.md,
- aggregate roots must derive from Core.AggregateRoot,
- aggregates must live in an Aggregates directory,
- each aggregate gets its own directory.
```

The mistake would be to independently encode all of this in:

```text
architecture.md
module-guide.md
CLAUDE.md
AGENTS.md
module-create/SKILL.md
Azure Pipeline
Azure DevOps process documentation
repository templates
```

Instead there should be one canonical policy model.

Illustrative only:

```yaml
kind: ModulePolicy

metadata:
  name: standard-module

spec:
  identity:
    requiresAxis: true
    projectName: "<axis>.<module>"

  azureDevOps:
    requiresArea: true

  architecture:
    allowedDependencies:
      - Core

  documentation:
    root: "modules/<axis>/<module>"
    requires:
      - README.md
      - intent.md
    readme:
      requiresIndex: true

  aggregates:
    whenPresent:
      directory: Aggregates
      aggregateDirectoryPerAggregate: true
      baseType: Core.AggregateRoot
```

The exact syntax is not the important part.

The important part is that the system defines once:

> **What constitutes a valid Module in this engineering environment?**

---

# 5. Three major outputs from the same canonical knowledge

A useful model is:

```text
              Canonical Engineering Rule
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
       Explain         Instruct       Enforce
          │              │              │
       Humans          Agents          Tools
```

## Explain

Human-readable documentation:

```text
How modules work
How releases work
Architecture conventions
Process documentation
Reference documentation
```

## Instruct

AI-facing operational guidance:

```text
Codex Skills
Claude rules
Agent responsibilities
Workflows
Context
Task guidance
```

## Enforce

Machine-verifiable rules:

```text
CI checks
Repository checks
Architecture checks
Coverage checks
External-system validation
Policy diagnostics
```

The same canonical knowledge may participate in all three.

---

# 6. Important rule categories

Markitect does not necessarily need a separate Resource Kind for each category, but these categories help define the problem space.

## Structural rules

Define required project structure.

Examples:

```text
Every Module requires README.md.
Every Module requires intent.md.
Every Aggregate requires its own directory.
```

## Architecture rules

Define allowed relationships.

Examples:

```text
Module -> Core      allowed
Module -> Product   forbidden
Product -> Module   allowed
```

## Coverage rules

Define what must be modeled or documented.

Examples:

```text
Every Aggregate must be documented.
Every Use Case must have a canonical owner.
Every Module must define its intent.
```

## Process rules

Define required procedures.

Examples:

```text
Release
  -> update version
  -> update changelog
  -> create release notes
  -> run required checks
  -> publish
```

## Format / template rules

Define required shape.

Example:

```text
Release Notes must contain:
- Summary
- Breaking Changes
- Migration
- Fixed Issues
```

## Trigger rules

Define obligations caused by events.

Examples:

```text
WHEN Release occurs
THEN Release Notes must exist.

WHEN Module is created
THEN Azure DevOps Area must exist.
```

## External-state rules

Connect canonical engineering intent to external systems.

Examples:

```text
Module exists
    => Azure DevOps Area exists

Release exists
    => matching published release exists
```

---

# 7. Normative vs descriptive engineering knowledge

Markitect should likely support two broad categories of knowledge.

## Normative knowledge

Defines how engineering work should happen.

Examples:

```text
Rules
Policies
Release processes
Definition of Done
Workflows
Documentation requirements
Escalation rules
Agent behaviour
```

Question answered:

> **How should we work?**

## Descriptive engineering knowledge

Defines how the system is conceptually designed.

Examples:

```text
Products
Modules
Use Cases
Aggregates
Interfaces
Processes
Contracts
Invariants
Architecture Decisions
```

Question answered:

> **How is the system structured and intended to work?**

The current Markitect model is stronger on the normative side.

A future architecture/documentation extension could cover the descriptive side.

---

# 8. Human documentation becomes a projection

The documentation should not necessarily be the sole source of truth.

Instead:

```text
Canonical Engineering Model
        ↓
Human Documentation View
```

For example:

```text
Module/Survey
UseCase/CreateSurvey
Interface/SurveyAPI
Invariant/ValidAnswer
```

may generate or contribute to:

```text
docs/architecture/survey/README.md
docs/architecture/survey/use-cases.md
docs/architecture/survey/interfaces.md
```

The same resources may also feed AI-agent context.

This is stronger than copying Markdown between providers.

---

# 9. Not all documentation should be generated

There are two different forms of documentation.

## Structured engineering facts

Good candidates for canonical modeling:

```text
Module X owns Y.
A depends on B.
Use Case X belongs to Module Y.
Interface X is exposed by Module Z.
Invariant Q applies.
Workflow A consists of steps B, C and D.
```

## Narrative documentation

Often better left human-authored:

```text
Why this architecture was chosen
Tutorials
Onboarding stories
Troubleshooting explanations
Long conceptual explanations
Historical context
```

A good long-term model is therefore:

```text
Canonical structured knowledge
             +
Human-owned narrative
```

rather than attempting to convert every paragraph into YAML.

---

# 10. Relationship to implementation

The canonical model should not imply that Markitect must understand source code semantically.

The current product boundary should remain:

> **Knowledge is modeled. Project artifacts are referenced. Domain-specific analysis stays outside the core.**

Markitect may know:

```text
UseCase/CreateSurvey
```

and that its implementation artifacts include:

```text
CreateSurveyCommand.cs
CreateSurveyHandler.cs
```

But the core does not need to parse C# ASTs or infer call graphs.

Instead:

```text
Canonical Engineering Model
         ↓
Relevant context
         ↓
AI Agent / Tool
         ↓
Implementation
```

The AI agent may use the model to implement the change.

Markitect then validates what can be validated deterministically.

---

# 11. Three levels of consistency

A useful distinction is:

## Declared consistency

Fully deterministic.

Example:

```text
UseCase/CreateSurvey
references Module/Survey

Does Module/Survey exist?
```

Markitect can prove this.

## Structural consistency

Potentially deterministic through plugins/checks.

Examples:

```text
Every configured use-case folder requires one UseCase resource.
Every module project may only reference allowed projects.
Every aggregate implementation derives from the configured base type.
```

This can be implemented by project-owned checks or specialized plugins.

## Semantic consistency

Harder to prove.

Example:

> Does the actual handler behavior still match the described business intent?

This may require:

```text
tests
static analysis
AI review
human review
```

Markitect does not need to solve all semantic correctness to be useful.

---

# 12. Compiler analogy

A normal compiler does not prove:

> This application fulfills the business goal.

It proves selected properties such as:

```text
syntax
types
references
contracts
```

Likewise, Markitect can prove:

```text
resources are valid
relationships resolve
ownership is valid
required concepts exist
contracts are compatible
policies are structurally satisfied
projections are synchronized
```

It does not need to claim:

```text
the architecture is good
the implementation is correct
the documentation prose is semantically true
```

This boundary is important.

---

# 13. Define / Validate / Project / Change

A strong conceptual model for Markitect is:

## DEFINE

Define canonical engineering intent.

```text
Architecture
Processes
Rules
Policies
Contracts
Responsibilities
Concepts
```

## VALIDATE

Prove deterministic invariants.

```text
References
Ownership
Dependencies
Coverage
Required structure
Contracts
External-state expectations
```

## PROJECT

Produce the right representation for each consumer.

```text
Human documentation
Codex
Claude
Skills
Agent configuration
CI configuration
Diagrams
Templates
```

## CHANGE

Understand and propagate consequences safely.

```text
What changed?
What depends on it?
Which projections are affected?
Which evidence is stale?
Which checks must run again?
Which external systems need an update?
```

This four-part model captures the current product direction well.

---

# 14. Markitect as a change compiler

The project's history around Dossiers and Change Requests is conceptually relevant.

Markitect naturally revolves around:

```text
Current State
     +
Desired Change
     +
Relevant Context
     +
Impact
     +
Review
     +
Apply
```

Therefore Markitect is not only a knowledge compiler.

It is also a **change-management compiler**.

Conceptually:

```text
State A
   ↓
desired engineering change
   ↓
plan
   ↓
impact
   ↓
required updates
   ↓
validation
   ↓
State B
```

The goal is to preserve engineering consistency through change.

---

# 15. Release example

A canonical Release Policy could define:

```text
ReleasePolicy
├── version must increase
├── changelog must be updated
├── release notes required
├── release notes follow template X
├── required checks must pass
├── package must be built
└── release must be published
```

From that one canonical policy, Markitect could support:

```text
Human Docs
    "How we release"

Codex Skill
    "Perform release"

Claude Skill
    "Perform release"

CI checks
    "Release candidate completeness"

Release Notes template
    required structure

External validation
    release/tag exists
```

If the policy changes:

```text
"Release Notes also require Migration Notes"
```

Markitect can identify all affected outputs and checks.

This is exactly the kind of change amplification Markitect should eliminate.

---

# 16. External systems

Azure DevOps is an important example.

Suppose:

```text
Module:
  name: Survey
  axis: Intake
```

and the policy says:

```text
Every Module requires:
Azure DevOps Area = Modules\<axis>\<module>
```

Markitect could support several integration levels.

## Validate

```text
Expected:
Modules\Intake\Survey

Observed:
missing
```

Diagnostic:

```text
MODULE_AZURE_AREA_MISSING
```

## Plan

```text
Required external changes:

+ Azure DevOps Area Modules\Intake\Survey
```

## Apply

With an explicit plugin/agent and appropriate authorization:

```text
Create Azure DevOps Area
```

This suggests a useful general model:

```text
Desired Engineering State
        ↓
Observed State
        ↓
Diff
        ↓
Plan
        ↓
Apply
```

---

# 17. Reconsidering the Runtime / Operator idea

A future Operator could be more useful as an **engineering-state reconciler** than as an AI-agent runtime.

Potential scope:

```text
Canonical Markitect Model
        │
        ├── repository structure
        ├── documentation
        ├── Azure DevOps
        ├── CI configuration
        ├── provider artifacts
        └── other engineering systems
                  ↓
            observe / reconcile
```

This would directly serve the original problem.

A runtime/operator should still remain a future capability and only be built for a concrete reconciliation requirement.

---

# 18. Generate vs Enforce vs Execute

These capabilities should remain clearly separated.

## Generate

```text
canonical model
    ↓
docs
agent instructions
templates
diagrams
provider configuration
```

## Enforce

```text
canonical model
    ↓
checks
policies
dependency validation
coverage validation
external-state validation
```

## Execute

```text
canonical process
    ↓
agent/plugin
    ↓
perform release
create Azure DevOps Area
scaffold module
```

Markitect Core does not necessarily need to execute everything.

It can be the canonical source for all three.

---

# 19. Plugin direction

A plugin architecture could allow domain-specific capabilities while keeping the core small.

Conceptually:

```text
                     MARKITECT CORE

Resource identity
Ownership
Graph
Relationships
Validation
Context
Impact
Packages
Projection
Change planning

                          │
                          ▼

                        Plugins

        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼

 markitect-software   markitect-azure    markitect-docs
                      -devops

 Module               Areas              Documentation views
 UseCase              Work Items         Indexes
 Interface            Policies           Navigation
 Aggregate            Validation         Diagrams
 Invariant            Apply
```

Other possible adapters:

```text
markitect-codex
markitect-claude
markitect-dotnet
```

The core should not need to understand every external domain.

---

# 20. Software architecture plugin

A possible software-focused extension could model broad concepts such as:

```text
Module
Product
UseCase
Aggregate
Interface
Process
Invariant
Decision
```

Avoid immediately modeling:

```text
Class
Method
Field
Controller
Repository
Database table
Column
Every framework concept
```

Otherwise Markitect risks becoming another large UML/Enterprise-Architecture tool.

The useful experiment is:

> Can a small set of stable engineering concepts provide enough canonical knowledge to generate useful documentation, agent context and architecture checks?

---

# 21. Example future architecture

```text
                    MARKITECT CORE

              Resource / Graph Engine
              Identity / Ownership
              Relations
              Validation
              Context
              Impact
              Packages
              Projection
              Change Management

                       │
                       ▼

                 DOMAIN MODELS

        ┌──────────────┼──────────────┐
        ▼              ▼              ▼

   AI Engineering    Software       External
      Model         Architecture     Systems

 Rule              Module          Azure DevOps
 Workflow          Product         CI
 Skill             UseCase         GitHub
 Agent             Aggregate
 Contract          Interface
                   Process
                   Invariant
                   Decision

                       │
                       ▼

                    OUTPUTS

Human Docs / Codex / Claude / CI / External Policies / Diagrams
```

---

# 22. Single Source of Truth does not mean one file

The goal is not:

```text
one giant YAML file
```

The goal is:

> **Every engineering fact has one canonical owner.**

A project can still have many resources.

For example:

```text
Module/Survey
UseCase/CreateSurvey
Interface/SurveyAPI
Invariant/SurveyMustExist
ReleasePolicy/Standard
```

What should disappear is independent duplication of the same fact across many systems.

---

# 23. Why this reduces AI consolidation work

Today, without a canonical model:

```text
docs
Claude
Codex
Skills
Rules
Workflows
CI
External tools
```

all contain overlapping engineering logic.

An LLM consolidation run must:

```text
read everything
infer ownership
infer relationships
find contradictions
decide which source is correct
rewrite duplicates
```

This is expensive and inherently uncertain.

With Markitect:

```text
canonical graph
    ↓
deterministic relationships
    ↓
known projections
```

many consistency questions become normal compiler checks rather than global AI reasoning tasks.

This is one of the central product benefits.

---

# 24. Product boundary

Markitect should own:

```text
canonical engineering knowledge
canonical engineering policy
relationships
ownership
validation
impact
projection
change planning
```

Markitect should not automatically become:

```text
a source-code semantic engine
a full UML system
an AI model runtime
a ticketing system
a CI platform
an Azure DevOps replacement
a generic database
```

Those systems may integrate with Markitect through adapters, plugins and explicit checks.

---

# 25. Central product promise

A concise statement of the desired product is:

> **I want to define once how our engineering system is supposed to work and trust that documentation, AI agents, repository structure, pipeline rules and external engineering systems remain consistent with that definition.**

Markitect exists to make that possible.

---

# 26. Suggested product principles

## Canonical ownership

> Every engineering fact has one canonical owner.

## Explicit relationships

> Dependencies, ownership and policy relationships should be modeled instead of rediscovered from prose.

## Projection over duplication

> Human documentation and AI instructions are views of canonical knowledge, not independent sources of truth.

## Deterministic where possible

> If a property can be structurally validated, do not repeatedly ask an LLM to rediscover it.

## Semantic humility

> Markitect should not claim to prove meaning that requires human or AI judgment.

## Change awareness

> Consistency is not a state created once; it is a property preserved through change.

## Integration over replacement

> External engineering systems remain their own systems. Markitect defines expectations and integrates with them.

## Small core, extensible domains

> The core owns graph, identity, policy, validation, impact and projection; domain-specific concepts belong in focused extensions when justified.

---

# 27. Current conceptual summary

Markitect is evolving from:

```text
AI artifact synchronization
```

toward:

```text
Engineering Knowledge
        +
Engineering Policy
        +
Change Management
        +
Projection
        +
Validation
```

The AI-agent use case remains central, but AI agents are best understood as:

```text
consumers of engineering knowledge
+
actors that apply engineering changes
```

rather than the entire subject of the product.

---

# 28. Short definition

> **Markitect is an engineering knowledge and policy compiler that helps teams define how systems are structured and how engineering work is performed, validate those definitions, project them consistently to humans and AI agents, and safely propagate changes across engineering tooling.**

# Markitect — Canonical Engineering Model, Plugins, Adapters and Reconciliation

## Status

Architecture concept and discussion document for review by the implementer.

This document is intended to capture the current product direction and provide a concrete basis for critique.

It is **not** a request to immediately implement the complete design.

The implementer should evaluate:

- whether this direction fits the current Markitect architecture,
- which parts are already present implicitly,
- which concepts belong in the Core,
- which concepts should live in Domain Extensions,
- which capabilities should be implemented as Adapters,
- where the design risks becoming too broad,
- and what the smallest next architectural step should be.

---

# 1. Central idea

The core idea is:

> **Markitect models engineering intent as a canonical domain model and keeps its technological projections reconciled.**

The problem is not merely synchronizing documentation, Claude and Codex.

The broader problem is that engineering rules and knowledge often exist simultaneously in:

```text
documentation
AI-agent instructions
Skills
Rules
Workflows
repository structure
source architecture
CI checks
pipeline policies
Azure DevOps
GitHub
templates
release processes
```

These representations frequently express the same underlying engineering truth.

When one changes, the others drift.

Markitect should avoid making every representation an independent source of truth.

Instead:

```text
Canonical Engineering Model
            │
            ├── Human documentation
            ├── Claude
            ├── Codex
            ├── CI
            ├── Azure DevOps
            ├── .NET architecture
            └── other integrations
```

Each integration should only need to remain consistent with the canonical model.

---

# 2. DDD as a product principle

The intended architecture follows a Domain-Driven Design principle:

> **Model the domain concepts first. Technology is an implementation detail around that model.**

For example, the important concept is:

```text
Module
```

Not:

```text
.csproj
Azure DevOps Area
Markdown page
Claude rule
Codex Skill
```

Those are technological representations of the concept.

Likewise, the domain may contain concepts such as:

```text
Release
Module
Product
Axis
UseCase
Aggregate
Interface
Process
Rule
Invariant
Contract
Decision
```

The goal is to develop a clear **Ubiquitous Language for the engineering system**.

---

# 3. Example: Module as a domain concept

Suppose the engineering organization defines a Module like this:

```text
Module
├── belongs to an Axis
├── has an Intent
├── owns Use Cases
├── may own Aggregates
├── exports Interfaces
├── has dependency restrictions
├── has documentation requirements
└── follows lifecycle rules
```

That is the canonical engineering meaning.

Technology-specific implementations are projections:

```text
                        Module
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
      .NET Adapter    Azure DevOps      Docs Adapter
          │              Adapter             │
          ▼                ▼                ▼
      Project          Area Path          Module Page

                           │
                 ┌─────────┴─────────┐
                 ▼                   ▼
              Codex                Claude
              Adapter              Adapter
```

The canonical Module concept should not be defined by any of these external systems.

---

# 4. Example: Release as a canonical concept

A Release might canonically mean:

```text
Release
├── requires version change
├── requires changelog
├── requires release notes
├── requires verification
├── follows ReleaseProcess
└── produces published artifact
```

From this same model, different projections can be created:

```text
Human documentation
    "How we release"

Codex Skill
    "Perform release"

Claude Skill
    "Perform release"

CI validation
    "Release candidate completeness"

Release Notes template
    required sections

External verification
    published release/tag exists
```

If the policy changes once, Markitect should be able to determine every affected projection.

---

# 5. One canonical owner, many projections

The intended Single Source of Truth principle is:

> **Every engineering fact has one canonical owner.**

It does **not** mean one giant YAML file.

A project may contain many canonical resources:

```text
Module/Survey
UseCase/CreateSurvey
Aggregate/Survey
Interface/SurveyAPI
Invariant/SurveyMustExist
ReleasePolicy/Standard
Process/Release
```

The important part is that the same fact is not independently redefined in multiple technologies.

For example:

```text
"Modules may only depend on Core."
```

should not independently exist as:

```text
architecture.md
CLAUDE.md
AGENTS.md
module-create/SKILL.md
pipeline check
project template
```

Instead these are projections or enforcement mechanisms derived from one canonical rule.

---

# 6. Plugins should not depend on each other

A central architectural invariant should be:

> **A plugin must remain consistent with the canonical model, never with another plugin's generated representation.**

Avoid:

```text
Claude Plugin
    ↓
Docs Plugin
    ↓
.NET Plugin
    ↓
Azure DevOps Plugin
```

Prefer:

```text
                   Canonical Model
                    /    |                       /     |                       ▼      ▼      ▼
               Claude   .NET   Azure
               Plugin  Plugin  DevOps
```

This avoids recreating the same cross-coupling that Markitect is supposed to eliminate.

---

# 7. Plugin responsibility

A plugin should know only:

1. the relevant canonical concepts,
2. its own external technology or domain,
3. the mapping between them.

For example:

## .NET Adapter

Knows how canonical concepts map to:

```text
projects
project references
folder structure
base types
architecture checks
```

## Azure DevOps Adapter

Knows how canonical concepts map to:

```text
Areas
Work Items
Policies
iterations
project configuration
```

## Documentation Adapter

Knows how canonical concepts map to:

```text
human-readable pages
indexes
navigation
reference documentation
diagrams
```

## Codex Adapter

Knows how relevant engineering knowledge maps to:

```text
AGENTS
Skills
Agents
Rules
task context
```

## Claude Adapter

Knows how relevant engineering knowledge maps to Claude-native structures.

None of these adapters should need to know how the others implement the same concept.

---

# 8. Domain Plugins vs Adapter Plugins

It may be useful to distinguish two broad plugin categories.

## Domain Plugins

Extend **what can be expressed** in the canonical model.

Examples:

```text
software
    Module
    Product
    UseCase
    Aggregate
    Interface
    Invariant

security
    Threat
    Control
    TrustBoundary

operations
    Service
    Environment
    Deployment
```

Domain Plugins enrich the Ubiquitous Language.

## Adapter Plugins

Extend **where canonical concepts are projected or enforced**.

Examples:

```text
dotnet
azure-devops
github
jira
docs
codex
claude
```

Conceptually:

```text
Domain Model
     ↓
Adapter
     ↓
Technology
```

This distinction may help keep the product model clean.

---

# 9. Core responsibilities

The Markitect Core should ideally remain small and technology-neutral.

Possible Core responsibilities:

```text
Resource identity
Ownership
Graph
Relationships
Contracts
Rules
Invariants
Validation
Context
Impact
Packages
Projection orchestration
Change planning
Reconciliation
```

The Core should not need to understand:

```text
C#
Java
GitHub
Azure DevOps
Jira
Claude
Codex
Terraform
```

Those belong at plugin or adapter boundaries.

---

# 10. Avoid technology leakage

A canonical `Module` should not become polluted with technology-specific fields.

Avoid a model like:

```yaml
kind: Module

spec:
  azureDevOps:
    internalNodeId: ...
  github:
    workflowJobName: ...
  dotnet:
    csprojProperty: ...
  claude:
    promptTemplateVersion: ...
```

Prefer:

```yaml
kind: Module

spec:
  axis: Intake
  dependencies:
    - Core
```

Then adapters own their mappings.

For example:

```text
Azure DevOps Adapter:
Module + Axis
    ↓
AreaPath = Modules/<axis>/<name>
```

```text
.NET Adapter:
Module dependencies
    ↓
allowed project references
```

```text
Docs Adapter:
Module
    ↓
human-readable architecture documentation
```

This keeps the canonical model domain-oriented.

---

# 11. Adapter configuration as Anti-Corruption Layer

This maps well to the DDD Anti-Corruption Layer concept.

For one organization:

```text
Module
    ↓
Azure DevOps Area
```

For another:

```text
Module
    ↓
Jira Component
```

The canonical concept remains:

```text
Module
```

The mapping lives in adapter configuration.

Example conceptually:

```text
AzureDevOps Adapter Configuration

Module
    mapsTo:
        AreaPath: "Modules/<axis>/<name>"
```

or:

```text
Jira Adapter Configuration

Module
    mapsTo:
        Component: "<axis>-<name>"
```

The external vocabulary does not leak back into the canonical engineering domain.

---

# 12. Plugin lifecycle

A useful generic plugin lifecycle could be:

```text
Consumes
Observe
Plan
Apply
Verify
Project
```

Not every plugin needs every capability.

## Consumes

Which canonical concepts does the plugin understand?

## Observe

What currently exists in the external technology?

## Plan

What must change for the external state to satisfy the canonical model?

## Apply

Perform the change if supported and explicitly authorized.

## Verify

Does the resulting state conform?

## Project

Produce a representation of the canonical knowledge.

Examples:

### Documentation plugin

Likely:

```text
Consumes
Project
Verify
```

### Azure DevOps plugin

Likely:

```text
Consumes
Observe
Plan
Apply
Verify
```

### .NET architecture plugin

Likely:

```text
Consumes
Observe
Verify
```

### Codex / Claude adapter

Likely:

```text
Consumes
Project
Verify
```

---

# 13. Reconciliation as a central mechanism

`reconcile` may become one of the most important Markitect concepts.

Conceptually:

```text
1. Load canonical model
2. Validate canonical model
3. Determine changed concepts
4. Determine affected plugins
5. Ask affected plugins for desired/observed delta
6. Build reconciliation plan
7. Apply or hand off required changes
8. Verify affected projections
9. Record fresh evidence
```

Example:

```text
Canonical change:
Module dependency policy changed

Reconciliation plan:

documentation
  ~ module architecture documentation

codex
  ~ module-create skill
  ~ architecture-review agent

claude
  ~ module rule
  ~ architecture-review agent

dotnet
  ~ architecture validation configuration

azure-devops
  no changes required
```

This replaces global manual consolidation with bounded, graph-driven reconciliation.

---

# 14. Impact becomes multi-layered

Long-term impact analysis may become:

```text
Canonical Change
      ↓
Concept Impact
      ↓
Policy Impact
      ↓
Projection Impact
      ↓
Plugin Impact
      ↓
External State Impact
```

Example:

```text
Aggregate policy changed
        │
        ├── Aggregate resources
        ├── Module creation workflow
        ├── Documentation projection
        ├── Codex architecture rules
        ├── Claude architecture rules
        └── .NET architecture validator
```

The dependency graph should make this computable without asking an LLM to rediscover every relationship.

---

# 15. Ubiquitous Language

DDD's Ubiquitous Language fits this product especially well.

If the organization calls something a:

```text
Module
```

then the engineering system should consistently use the concept `Module`.

Adapters may represent it differently, but they should not redefine the underlying concept.

Canonical vocabulary may include:

```text
Module
Product
Axis
UseCase
Aggregate
Interface
Release
Process
Rule
Invariant
Contract
Decision
```

This provides clear answers to questions such as:

```text
What is a Module?
How do we release?
Can Modules depend on each other?
What does a valid Aggregate require?
How is a Use Case structured?
```

There should be one canonical place where those answers are defined.

---

# 16. Documentation as human projection

Human documentation becomes a projection of the canonical Ubiquitous Language.

For example, a human asking:

> How does a Release work?

could see:

```text
Release

Purpose

Required Inputs

Process
1. ...
2. ...
3. ...

Required Outputs
- Changelog
- Release Notes
- ...

Validation
...
```

Codex may receive the same process operationally:

```text
To perform a Release:
1. ...
2. ...
3. ...
```

CI may receive only machine-checkable obligations.

The semantic source remains shared.

---

# 17. Generate, Enforce and Execute

These three responsibilities should remain distinct.

## Generate

```text
canonical model
    ↓
documentation
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
agent / adapter
    ↓
perform release
create Azure DevOps Area
scaffold module
```

The Markitect Core does not need to execute every process itself.

It can remain the canonical source of truth and orchestration layer.

---

# 18. Desired State vs Observed State

External integrations naturally create a desired-state model.

Example:

```text
Canonical Model

Module: Survey
Axis: Intake
```

Policy:

```text
Every Module requires:
Azure DevOps Area = Modules\<axis>\<module>
```

Observed external state:

```text
Modules\Intake\Survey
missing
```

Markitect can then produce:

```text
Desired State
      ↓
Observed State
      ↓
Diff
      ↓
Plan
      ↓
Apply
      ↓
Verify
```

This is a stronger and more directly relevant interpretation of the future Runtime / Operator idea.

---

# 19. Runtime / Operator reinterpretation

A future Markitect Operator may be more useful as an:

> **Engineering-state reconciler**

rather than an AI-agent runtime.

Potential responsibilities:

```text
Canonical Markitect Model
        │
        ├── repository structure
        ├── documentation
        ├── Azure DevOps
        ├── CI configuration
        ├── provider projections
        └── other engineering systems
                  ↓
            observe / reconcile
```

This should remain a future capability and should only be implemented when a concrete reconciliation requirement justifies it.

---

# 20. Example end-to-end model

Canonical model:

```text
Module: Survey
Axis: Intake

Rules:
- depends only on Core
- requires Intent

Aggregate:
Survey
- belongs to Module Survey
- derives conceptually from AggregateRoot

Process:
Release
- requires Release Notes
```

Configured extensions:

```text
Software Domain Plugin
.NET Adapter
Azure DevOps Adapter
Docs Adapter
Codex Adapter
Claude Adapter
```

Result:

```text
                     Canonical Model
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
     .NET               Azure DevOps         Docs
        │                  │                  │
Survey.csproj      Modules/Intake/Survey   modules/
references Core                          intake/survey/

        ┌──────────────────┴──────────────────┐
        ▼                                     ▼
      Codex                                 Claude
        │                                     │
module-create Skill                     module Rule
architecture Agent                     architecture Agent
```

Each projection remains independently consistent with the canonical model.

They do not synchronize with each other.

---

# 21. Definition of Done as reconciled engineering state

A change should not necessarily be considered complete merely because code was merged.

A stronger engineering-state model could be:

```text
Canonical model             consistent
Documentation projection    reconciled
Codex projection            reconciled
Claude projection           reconciled
.NET architecture           verified
Azure DevOps state          verified
CI policies                 verified
```

Then:

```text
Engineering State = reconciled
```

This provides a much stronger definition of consistency.

---

# 22. AI-agent value

This model may be particularly useful for AI Agents.

Instead of loading:

```text
AGENTS.md
17 Skills
complete architecture documentation
all repository rules
```

an agent performing:

```text
"Add ExportSurvey Use Case"
```

could receive a bounded compiled context:

```text
Relevant concepts:
Module/Survey
UseCasePolicy/Standard
Interface/SurveyAPI

Relevant processes:
Process/AddUseCase

Relevant rules:
Rule/ModuleDependencies
Rule/UseCaseDocumentation
Rule/Testing

Relevant project artifacts:
selected implementation files

Relevant adapter guidance:
.NET
Codex
```

This should reduce irrelevant context and make agent behavior easier to control.

---

# 23. Main architectural risk: God Model

The largest danger is turning the canonical model into a giant representation of every technical detail.

Avoid making Markitect the place where every configuration property of every integration is stored.

The canonical domain should contain only concepts and facts that belong to the engineering model.

Technology-specific implementation details should remain in adapters.

---

# 24. Main plugin rule

A useful rule for extension design:

> **Domain Plugins define engineering meaning. Adapter Plugins translate engineering meaning into technology-specific state.**

This creates a clean dependency direction:

```text
Core
  ↓
Domain Model
  ↓
Adapter
  ↓
Technology
```

Never:

```text
Technology
  ↓
defines canonical domain meaning
```

---

# 25. Product definition

A concise product definition:

> **Markitect is an engineering knowledge, policy and reconciliation system. It provides a canonical model for how systems are structured and how engineering work is performed. Domain extensions add engineering concepts; adapters project and enforce those concepts in documentation, AI agents, source ecosystems, CI and external engineering systems. When the canonical model changes, Markitect determines the impact and reconciles the affected projections so each integration only needs to remain consistent with the canonical model.**

Shorter:

> **Markitect models engineering intent as a canonical domain model and keeps its technological projections reconciled.**

---

# 26. Design principles

## Canonical domain first

> Model engineering meaning before technology.

## Ubiquitous Language

> Use one shared vocabulary for engineering concepts.

## One canonical owner

> Every engineering fact has one authoritative owner.

## Adapters are independent

> Plugins reconcile against the canonical model, not against each other.

## Anti-Corruption Layers

> External-system terminology stays outside the engineering domain model.

## Explicit impact

> A change should reveal which concepts, projections and integrations are affected.

## Reconciliation over global consolidation

> Reconcile only affected projections instead of repeatedly reviewing the entire knowledge base.

## Small Core

> Core owns universal mechanics; domain meaning and technology mappings are extensible.

## Deterministic where possible

> Structural facts should be validated by tooling rather than repeatedly rediscovered by AI.

## AI as consumer and actor

> Agents consume compiled engineering context and may apply changes, but they do not define the canonical truth.

---

# 27. Questions for the implementer

Please evaluate this direction against the current Markitect codebase and product model.

In particular:

1. Does the current Core already provide enough generic primitives for this architecture?
2. Which current Resource Kinds are universal and which are actually one domain model?
3. Should `Rule`, `Workflow`, `Skill`, `Agent` and `Contract` remain Core concepts or become part of an AI-engineering domain extension?
4. What is the minimum universal Core abstraction?
5. What should a Domain Plugin be allowed to define?
6. What should an Adapter Plugin be allowed to define?
7. Is `Consumes / Observe / Plan / Apply / Verify / Project` a useful plugin capability model?
8. How should plugins declare which canonical concepts they consume?
9. How should impact analysis map changed canonical concepts to affected plugins?
10. How should reconciliation evidence be recorded?
11. How can adapter configuration act as a clean Anti-Corruption Layer?
12. How can we prevent technology-specific configuration from leaking into canonical resources?
13. What parts of this direction are already supported by the current graph, packages, snapshots, context and render architecture?
14. Which parts would require fundamental redesign?
15. What should explicitly remain out of scope?
16. What is the smallest experiment that can validate the Domain Plugin / Adapter Plugin distinction?
17. Does this model improve Markitect's product coherence, or make it too broad?
18. Which terminology should become canonical now before more features are added?

---

# 28. Requested review outcome

Please provide a recommendation that distinguishes:

```text
Already aligned with current architecture
Natural extension
Requires a new abstraction
Should be a plugin/extension
Should remain adapter-specific
Should be deferred
Should not be part of Markitect
```

The goal is not to force this architecture.

The goal is to determine whether this is the right long-term product model before further feature development.


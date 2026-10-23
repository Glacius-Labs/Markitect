# Markitect — "Copy Me": Reverse-Engineering an Implicit Engineering Language

## Status

Concept document for implementer review.

This document captures a possible AI-assisted onboarding and knowledge-distillation workflow.

The central idea:

> **Use AI to reverse-engineer an architect's or organization's implicit engineering language from existing projects and decisions, then make that language explicit, typed and enforceable in Markitect.**

This should never mean that existing code automatically becomes canonical truth.

The proposed process is:

```text
OBSERVE
    ↓
INFER
    ↓
REVIEW
    ↓
CANONIZE
```

---

# 1. Motivation

A major adoption problem for any canonical engineering model is migration effort.

Users already have years of implicit knowledge distributed across:

```text
repositories
folder structures
architecture
code conventions
documentation
reviews
PR comments
AI conversations
CI
release processes
tickets
```

They are unlikely to manually rewrite all of that into a new modeling language from scratch.

AI is well suited to extracting candidate patterns from this existing evidence.

---

# 2. Reverse engineering, not automatic truth generation

A project contains multiple kinds of evidence:

```text
intentional architecture
personal preferences
project-specific requirements
legacy design
temporary compromises
poor AI-generated implementation
accidental conventions
```

Therefore:

> **Observed does not mean intended.**

Markitect must not automatically canonize discovered structure.

The output of discovery should initially be hypotheses.

---

# 3. Phase 1 — Observe

An AI agent analyzes one or more projects.

Possible evidence:

```text
folder structures
module boundaries
project dependencies
naming
layering
Vertical Slices
Common/shared concepts
aggregate organization
interfaces
tests
documentation
release processes
agent instructions
architecture decisions
PR reviews
review comments
corrections made after AI-generated code
```

No canonical model is changed in this phase.

---

# 4. Phase 2 — Infer

The agent produces explicit candidate patterns.

Example:

```text
Candidate Pattern:
VerticalSliceApplication

Evidence:
- 37/41 observed Use Cases have an isolated folder.
- every recent Use Case has one Handler.
- Feature-local shared concepts are placed in Common.
- direct UseCase-to-UseCase dependencies are rare.

Hypothesis:
A Use Case should be implemented as an isolated Vertical Slice
that owns its Handler and slice-specific implementation.

Confidence:
High
```

The agent should surface counterexamples rather than hiding them.

---

# 5. Classify discovered evidence

Candidate observations should be classified.

Possible categories:

```text
Likely intentional
Strong recurring convention
Project-specific
Legacy
Possible violation
Possible temporary compromise
Unclear
```

Example:

```text
Observed:
Older projects contain generic Services folders.

Observed:
Recent projects organize Application logic as explicit UseCases.

Hypothesis:
Service-centric organization is legacy;
VerticalSliceApplication is current preferred style.
```

---

# 6. Phase 3 — Review

The AI should actively ask questions where evidence is ambiguous.

Examples:

```text
Is Common inside a Feature an approved pattern or merely tolerated?

Does this rule apply to all projects or only DDD systems?

Should Modules depend only on Core, or are same-Axis module dependencies allowed?

Is IRepository still preferred, or is the newer Port-style approach the current standard?
```

This is especially important when projects reflect different eras of architectural preference.

---

# 7. Phase 4 — Canonize

Only explicit approval turns a proposal into canonical Markitect knowledge.

Possible output:

```text
glacius/software-architecture
```

containing:

```text
CleanArchitectureLayers
VerticalSliceApplication
ExplicitUseCases
CommandQueryPattern
AggregatePattern
FeatureCommonPattern
ModuleBoundaryPolicy
DocumentationPolicy
TestingPolicy
ReleasePolicy
```

The important rule:

> **Inference may propose canonical knowledge. Only explicit adoption makes it canonical.**

---

# 8. Multiple projects reveal style better than one project

One repository mostly reveals:

```text
how this project works
```

Several repositories can reveal:

```text
how this architect/team tends to design systems
```

Example:

```text
Konfyra
Markitect
GLCS
other projects
```

may reveal repeated tendencies:

```text
responsibility-oriented modules
explicit Core
Vertical Slices
typed abstractions
explicit contracts
canonical documentation
composition over inheritance
strong dependency boundaries
```

This allows Markitect to separate:

```text
personal / organization-wide engineering style
```

from:

```text
project-specific architecture
```

---

# 9. Detect evolution of engineering style

Engineering preferences change.

Possible inference:

```text
Older projects:
Service-centric Application Layer

Recent projects:
Explicit UseCases + Vertical Slices

Hypothesis:
preferred architecture evolved
```

Markitect could propose:

```text
mark Service-centric pattern as legacy
adopt VerticalSliceApplication as current
```

This suggests versioned engineering-style packages:

```text
glacius/software-architecture@1
glacius/software-architecture@2
glacius/software-architecture@3
```

---

# 10. Review decisions may be more valuable than finished code

A particularly rich source of engineering preference is:

```text
AI generated implementation
        ↓
architect rejects or changes it
        ↓
reason
```

Repeated comments such as:

```text
"That does not belong in Core."

"This is too product-specific."

"Make this a separate concept."

"Why is this a Service?"

"This should be a Use Case."

"Module knowledge is leaking here."

"This information now exists in two places."
```

reveal the architect's decision function.

A distillation system could propose:

```text
Repeated correction:
"Do not leak product-specific behavior into Core."

Candidate invariant:
Core resources must remain product-independent.
```

This may be more useful than only mining static repository structure.

---

# 11. Structured engineering profile

"Copy me" should not mean:

```text
prompt:
Act like Eduard.
```

A stronger result is a structured Engineering Profile.

Possible layers:

## Architectural Style

```text
Clean Architecture
DDD
Vertical Slices
Modular Architecture
Composition
Explicit boundaries
```

## Structural Style

```text
Feature as Folder
UseCase owns Handler
Common only for genuinely shared concepts
Aggregate per folder
```

## Design Style

```text
strong invariants
small explicit interfaces
few accidental dependencies
extension points before proliferation
avoid generic Service dumping grounds
```

## Change Style

```text
understand consequences first
prefer architectural consistency
refactor weak foundations before building on them
make recurring changes mechanical
```

## Process Style

```text
explicit release process
canonical documentation ownership
review architecture-impacting exceptions
```

Some of these may become deterministic rules; others may remain guidance.

---

# 12. Architecture imitation without model training

Avoid making this primarily a fine-tuning problem.

Preferred architecture:

```text
Repositories / decisions / reviews
        ↓
LLM inference
        ↓
explicit Markitect model
```

Benefits:

```text
inspectable
editable
versioned
testable
portable
provider-independent
```

Instead of:

```text
"We trained a model that implicitly thinks like you."
```

A user should be able to ask:

> Why did the agent choose this structure?

and receive:

```text
Because:
VerticalSliceApplication requires UseCase ownership
and
CoreBoundaryPolicy forbids the proposed dependency.
```

---

# 13. Possible workflow modes

Naming is provisional, but three conceptual modes are useful.

## Discover

Read-only analysis.

```text
markitect discover
```

Output:

```text
Observed Patterns
Possible Rules
Possible Concepts
Conflicts
Unknowns
Counterexamples
```

No mutation.

## Distill

Build candidate Markitect definitions.

```text
markitect distill
```

Output:

```text
Proposed:
VerticalSliceUseCase
CoreBoundaryPolicy
ModulePattern
ReleasePolicy
...
```

Still not canonical.

## Adopt

After explicit review:

```text
markitect adopt
```

Candidate knowledge becomes canonical.

The exact CLI should not be implemented before the concept is validated.

---

# 14. Dossier / questionnaire connection

When evidence is ambiguous, targeted questions can complete the model.

Conceptually:

```text
Repository evidence
        +
AI analysis
        +
targeted questions
        ↓
Canonical engineering model
```

This resembles the original Dossier/intake workflow:

```text
collect incomplete information
ask focused questions
resolve ambiguity
produce structured canonical state
```

This may be a useful conceptual bridge, but should not force Dossier functionality into Markitect Core.

---

# 15. Bottom-up and top-down loop

This creates a powerful two-way model.

## Bottom-up

```text
existing engineering reality
        ↓
observe
        ↓
infer
        ↓
review
        ↓
canonical model
```

## Top-down

```text
canonical model
        ↓
agents / adapters
        ↓
desired implementation
        ↓
reconciliation
```

Full loop:

```text
            Existing Engineering Reality
                      │
                      ▼
                   Observe
                      │
                      ▼
                    Infer
                      │
                human review
                      │
                      ▼
                  Canonize
                      │
                      ▼
              Markitect Model
                      │
          ┌───────────┼───────────┐
          ▼           ▼           ▼
        Docs        Agents      Checks
                      │
                      ▼
                New Changes
                      │
                      ▼
                 Reconcile
```

---

# 16. Main safety/product rule

Never:

```text
existing code
    ↓
automatic canonical truth
```

Otherwise Markitect may preserve:

```text
technical debt
accidental conventions
bad AI-generated structures
legacy decisions
```

Instead:

```text
evidence
    ↓
proposal
    ↓
explicit decision
    ↓
canonical truth
```

---

# 17. Confidence and evidence should remain visible

A discovered proposal should carry evidence such as:

```text
supporting projects
supporting files
frequency
counterexamples
recency
review comments
confidence
```

Example:

```text
Candidate:
VerticalSliceApplication

Support:
5 recent projects

Counterexamples:
1 legacy project

Confidence:
High

Reasoning:
Repeated recent structure + explicit review comments
```

Do not hide inference behind a single authoritative-looking generated YAML file.

---

# 18. Possible onboarding value

This feature could solve one of Markitect's hardest adoption problems:

> Nobody wants to manually translate years of engineering habits into a modeling language from zero.

A discovery/distillation workflow could turn existing engineering history into:

```text
candidate canonical knowledge
```

that the user refines instead of creating from scratch.

This may become a significant onboarding differentiator.

---

# 19. Non-goals

The first version should not:

```text
auto-canonize discovered rules
train custom models
perform hidden long-term behavioral profiling
claim that frequency equals correctness
model every implementation detail
replace human architectural judgment
```

The purpose is assisted explicit modeling.

---

# 20. Implementer review questions

1. How well does this fit Markitect's current authoring model?
2. Could this initially be only a Skill/Agent workflow outside Core?
3. What data should be supplied to the discovery agent?
4. How should evidence and confidence be represented?
5. How should counterexamples be retained?
6. How should candidate resources differ from canonical resources?
7. Is a separate "proposal" model necessary, or can normal change-plan mechanics handle it?
8. How can existing Markitect `impact`, `context`, packages and review evidence help?
9. What privacy/scope boundaries are needed when analyzing multiple repositories?
10. Can review comments/AI conversations be used without coupling Markitect to specific providers?
11. How should personal style be separated from project-specific rules?
12. How should legacy vs current preferences be represented?
13. What is the smallest useful experiment?
14. Should this remain an optional onboarding Skill rather than a Core capability?
15. Which parts can remain AI reasoning and which should become deterministic after adoption?

---

# 21. Core principle

> **AI may discover and propose an engineering language. Markitect makes it explicit. Human adoption makes it authoritative.**

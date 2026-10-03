# Markitect — Existing-Project Adoption and Engineering-Knowledge Discovery

## Goal

A mature repository such as Konfyra should not require its owner to manually re-enter years of rules, architecture and process knowledge.

The desired model is:

> **Reverse-engineer the existing engineering system, present evidence-backed candidates, ask the human only where intent is ambiguous, then explicitly adopt the reviewed result.**

---

## Existing repository inputs

Likely evidence sources:

```text
AGENTS.md
CLAUDE.md
.codex/
.claude/
.agents/
docs/
ADRs
architecture tests
CI
release workflows
project templates
module/project structure
source code
scripts
```

If Markitect asks the owner to define all of this again from memory, adoption recreates the duplication/drift problem.

---

## Adoption lifecycle

```text
Existing repository
        ↓
Inventory engineering knowledge
        ↓
Freeze selected evidence
        ↓
Extract candidate concepts / rules / processes
        ↓
Deduplicate equivalent representations
        ↓
Detect conflicts, legacy and uncertainty
        ↓
Human decision queue
        ↓
Candidate Markitect model
        ↓
Shadow-mode comparison
        ↓
Explicit adoption
        ↓
Canonical engineering constitution
```

---

## Evidence, not authority

Existing artifacts may represent:

```text
intentional architecture
current policy
legacy behavior
temporary compromise
technical debt
provider-specific duplication
accidental implementation
```

Core rule:

> **Observed does not mean intended.**

---

## Candidate facts

Example:

```text
Candidate:
vertical-slice-use-cases

Proposed meaning:
Application behavior is organized as explicit UseCases.
Each UseCase is a Command or Query and owns a Handler.

Support:
- AGENTS.md
- Claude rule
- architecture test
- 31 recent slices

Counterexamples:
- 2 legacy Services

Interpretation:
Likely current standard with legacy exceptions.

Confidence:
high
```

Candidate fields should include:

```text
id
category
proposed meaning
supporting evidence
counterexamples
source locations
recency
confidence
open questions
```

---

## Deduplication

Do not create five canonical rules for five representations of one concept.

Example evidence:

```text
AGENTS.md:
Use vertical slices.

Claude:
Do not introduce generic application services.

Docs:
Application logic is organized as Commands and Queries.

Architecture tests:
Handler naming/visibility rules.

Code:
Feature folders consistently contain Command/Query + Handler.
```

Desired:

```text
Canonical candidate:
VerticalSliceUseCase

Evidence:
all of the above
```

---

## Knowledge classification

### Normative policy

How the project must operate.

### Descriptive architecture

How the system is intended to be structured.

### Process

How engineering changes happen.

### Provider projection

AGENTS / Claude / Codex forms that should usually become consumers/projections.

### Technical evidence

Architecture tests, dependency observations, build checks.

### Narrative knowledge

Rationale, tutorials, historical explanation and ADR prose.

Not everything should become YAML.

---

## Conflict detection

Example:

```text
AGENTS.md:
Modules never depend on other Modules.

Architecture test:
IntegrationEvents dependencies are allowed.

Observed code:
Payments.Application → Meetings.IntegrationEvents
```

Produce:

```text
CONFLICT

Candidate rule:
Cross-module dependencies are forbidden.

Contradicting evidence:
- AGENTS.md forbids all cross-module dependencies.
- architecture test permits IntegrationEvents.
- implementation uses IntegrationEvents.

Possible interpretation:
Direct implementation dependencies are forbidden,
but explicit integration contracts are allowed.

Human decision required.
```

Never silently select the most frequent source as truth.

---

## Architecture evolution

Example:

```text
Older code:
Application Services

Recent code:
UseCases / Commands / Queries

Recent reviews:
Do not add another Service.

Current AI guidance:
Vertical Slice
```

Candidate:

```text
Legacy:
Service-centric Application Layer

Current:
UseCase-centric Vertical Slice architecture
```

Human adoption decides.

---

## Human decision queue

Desired:

```text
96 candidate facts

72 strongly supported
13 duplicated representations
7 conflicts
4 unclear conventions
```

The owner primarily answers 11 unresolved decisions rather than re-reading the whole repository.

---

## Candidate workspace

Conceptually:

```text
.markitect/
  discovery/
    sessions/
    evidence/
    candidates/
    decisions/
```

Candidates remain outside normal canonical behavior until adopted.

---

## Adoption

```text
candidate
    ↓
human review
    ↓
adoption plan/diff
    ↓
canonical Areas / Packages
```

AI must never approve its own inference.

---

## Shadow mode

Run old and new systems together.

Check:

```text
Can Markitect reproduce intended AGENTS guidance?
Can Claude/Codex outputs be generated?
Which architecture tests correspond to canonical policies?
Which remain specialist technical checks?
Which documentation remains narrative?
Which normative statements remain uncovered?
```

---

## Migration coverage report

Example:

```text
AGENTS.md
  18/21 normative statements canonicalized
  2 narrative statements preserved
  1 unresolved

Claude
  14/14 governed statements covered

Codex
  17/18 covered
  1 provider-specific statement retained

Architecture tests
  8 mapped to canonical policy
  4 remain project-owned checks
```

---

## Copy Me as migration mechanism

Copy Me should be understood as more than personal coding-style learning.

It can become:

```text
Legacy engineering system
        ↓
AI-assisted discovery
        ↓
evidence-backed candidates
        ↓
human decisions
        ↓
canonical Markitect model
```

Most meaningful adopters will be existing projects.

---

## Product principle

> **Markitect adoption should convert an existing engineering system into an explicit model by evidence and review, not by asking its owner to rewrite everything from memory.**

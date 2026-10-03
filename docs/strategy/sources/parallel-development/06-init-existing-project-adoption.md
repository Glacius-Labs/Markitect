# Markitect — Existing-Project `init` / Adoption Bootstrap

## Goal

Support two conceptual initialization paths:

```text
Greenfield:
create an empty Markitect project.

Existing project:
prepare a mature repository for discovery, migration and shadow-mode adoption.
```

Principle:

> **`init` prepares the control plane. Discovery reconstructs existing knowledge. Human review creates authority.**

---

## Proposed UX

Illustrative:

```bash
markitect init --adopt-existing
```

Exact naming should fit the CLI.

This mode should create only the control plane and adoption workspace.

It must not automatically canonize anything.

---

## Safe preview

Example:

```text
Existing-project adoption plan

Will create:
  markitect.yaml
  .markitect/areas/engineering/README.md
  .markitect/discovery/README.md
  .markitect/discovery/config.yaml

Detected candidate evidence roots:
  AGENTS.md
  docs/
  .claude/
  .codex/
  tests/Architecture/
  src/

Nothing will be analyzed or adopted yet.
```

Writes remain explicit.

---

## Must not

Existing-project init must not:

```text
rewrite AGENTS.md
rewrite Claude/Codex files
move documentation
change architecture tests
create active policies
delete files
silently activate adapters
auto-approve discovered conventions
```

---

## Evidence roots

Distinguish:

```text
detected
selected
excluded
```

Detection is not ingestion.

The user reviews scope before AI discovery.

---

## Source roles

Conceptually:

```yaml
sources:
  - path: AGENTS.md
    role: agent-guidance

  - path: docs/architecture
    role: architecture-documentation

  - path: docs/adr
    role: decisions

  - path: .claude
    role: provider-guidance

  - path: .codex
    role: provider-guidance

  - path: tests/Architecture
    role: deterministic-checks

  - path: src
    role: implementation-evidence
```

Exact schema is open.

---

## Separate init from discovery

Recommended:

```text
init
    ↓
review control plane
    ↓
review evidence scope
    ↓
freeze snapshot
    ↓
run discovery
```

Possible command family:

```bash
markitect init --adopt-existing
markitect discovery plan
markitect discovery run
markitect discovery review
markitect discovery adopt
```

Exact command names are not prescribed.

---

## Snapshot discipline

Discovery should bind:

```text
Git revision
selected paths
file hashes
tool version
selection digest
```

so candidate provenance is reproducible.

---

## Privacy/safety

The user should review evidence before AI analysis.

Make it easy to exclude:

```text
credentials
private keys
customer data
generated binaries
dependency caches
build outputs
vendor trees
```

Do not claim perfect secret detection.

---

## Existing authority remains

During discovery/shadow mode:

```text
ADRs
AGENTS.md
architecture tests
provider configs
```

remain existing authorities.

Only reviewed adoption changes canonical ownership.

---

## Success criterion

The user should move from:

```text
"I know this project contains many rules,
but I cannot remember all of them."
```

to:

```text
"Markitect inventoried them,
showed conflicts,
I answered the unresolved decisions,
and I now have a reviewable candidate constitution."
```

without manually reconstructing years of engineering knowledge.

# Design decisions

These choices form the proposed design baseline for the first version. They are documented before implementation and remain subject to owner review. They explain the current contracts; those contracts live in their owning documents rather than being redefined here.

## Local command-line application

**Context.** The product needs a small personal work list, not shared work management.

**Decision.** Use a short-lived CLI process with three domain actions. Target one person's sequential use on a local machine.

**Alternatives.** A browser UI, interactive full-screen terminal application, service or synchronized store.

**Reason and consequences.** The chosen boundary keeps setup, behavior and tests understandable. There is no account or network dependency, but no concurrent-writer or crash-recovery promise. Output failure after committed storage needs an explicit outcome; confirming success and committing data are separate events. [Product](product.md) and [CLI](cli.md) own the detailed contract.

## Python standard library and source delivery

**Context.** Complete setup and development instructions need a concrete runtime. Keeping the language indefinitely open would leave those instructions hypothetical in a way unrelated to product behavior.

**Decision.** Baseline Python 3.13.x with standard-library code/tests and source execution through `python -m tasks`. Deliver a versioned source archive rather than a registry installer in the first version.

**Alternatives.** Another language, a standalone native executable or an installable package with build dependencies.

**Reason and consequences.** This avoids an additional build/package toolchain for a small application. Python must already be installed, and the user runs from the source root. Windows and Linux both require evidence before release. The choice is a new proposal, not evidence of implemented portability. [Contributing](../CONTRIBUTING.md) owns commands and [Operations](operations.md) owns delivery.

## Explicit local JSON format

**Context.** Persistence and recovery need a known representation and a stable file-selection policy.

**Decision.** Use a versioned JSON file under a stable user-home default, with a per-command path override. Validate the complete file and save by same-directory replacement.

**Alternatives.** A working-directory default, SQLite or a service/database server.

**Reason and consequences.** The home default survives changes of working directory; overrides support separate lists and isolated tests. JSON is inspectable but plaintext and full-file loading does not scale indefinitely. A version marker makes unsupported data explicit; automatic migration is deferred. File replacement is not a backup strategy or a power-loss guarantee. [Storage](storage.md) and [Operations](operations.md) own the exact rules.

## Bounded stable references

**Context.** Identical task texts must remain selectable, and both the CLI and persistent format need predictable numeric validation.

**Decision.** Use stable positive references with a declared signed-64-bit upper bound, independent of platform integer size. Task creation owns allocation and capacity refusal.

**Alternatives.** Selecting by text, UUIDs or unbounded decimal references.

**Reason and consequences.** Short numbers suit a local list. A defined bound gives clear overflow behavior without depending on a parser's accidental range. Reaching it prevents adding but not reading or completing. The [domain](domain.md) and [CLI](cli.md) own the resulting behavior.

## Deferred capabilities and remaining work

Editing/deleting/reopening, synchronization, concurrent use, encryption and automatic migrations need demonstrated user need and separate design work. They are not hidden implementation tasks for version 0.1.

The current documentation resolves the first version's runtime, interface and file-format choices. Remaining work is realization, testing on each target, measurement, user review and release preparation. Host-specific Python installation paths and an eventual release-download location depend on the actual environment/release; the docs do not invent them. Any further ambiguity found during implementation should be resolved in the owning contract before silently choosing incompatible behavior.

# Markitect agent entrypoint

Start with [the documentation router](docs/README.md) and the owner relevant to the task. [Architecture](docs/architecture.md) describes implemented behavior; [the implementation plan](docs/implementation-plan.md) owns future product work. [Development](CONTRIBUTING.md) defines checks and release boundaries.

Keep the Go core independent of consumer repositories and providers. Use YAML for Markitect-owned data. Add abstractions or resource kinds only when a concrete invariant needs them. Preserve explicit dependencies, deterministic diagnostics, fixed-snapshot evidence and one output owner.

Change canonical sources and regenerate views/schemas. Keep examples executable. Do not silently infer dependencies from prose links, weaken unknown-input invalidation, transfer human acceptance through AI evidence, or copy customer policy into reusable content.

Work on a non-protected branch. Scope tests to concrete risks and required gates. Changes to a consumer belong in that consumer's checkout and follow its instructions. A source change here does not upgrade a consumer: publish/prepare a versioned package and update its lock deliberately.

Markitect's source and product decisions are maintained here. The previous Cockpit checkout is retired as an editing location. Existing Konfyra profile/migration adapters are compatibility code, not authority for general product policy.

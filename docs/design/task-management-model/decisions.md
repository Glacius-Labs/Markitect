# Decisions

All intent is proposed. Choices marked selected are selections for this draft, awaiting owner review; they do not change Markitect's accepted vocabulary. Contract text belongs to each linked subject and is not repeated here as another normative source.

<a id="decision-cli"></a>
## DEC-CLI — Concrete CLI and selection contract

Type: Decision. Choice status: selected in this draft.

**Context.** The ordinary description chose CLI use but left command spelling and unique selection open. A complete observable example needs a concrete interface.

**Choice owner.** COMP-CLI owns common syntax and file selection. The UseCases own their commands/results; CONCEPT-TASK owns reference/text meaning; REQ-ERRORS owns channels and exit codes.

**Rationale.** Simple decimal references make identical task texts distinguishable. Deterministic line output and a fixed language keep the example inspectable without requiring a GUI or localization subsystem.

**Alternatives.** Interactive menus, selecting by text and machine-only JSON output. None is needed for this first exercise.

**Consequences.** These details narrow previously open choices. Their selection is editorial, not a fact inherited from the original description. The owner can change them together with their contract owners and Checks.

<a id="decision-file"></a>
## DEC-FILE — One selected local task file

Type: Decision. Choice status: selected in this draft.

**Context.** Local persistence was selected, but location and initial-state behavior needed to be made explicit.

**Choice owner.** COMP-CLI owns location resolution; REQ-SAFE-STORAGE owns missing/error behavior; COMP-STORAGE owns the mechanism.

**Rationale.** A visible default and explicit override make local use and isolated tests possible without global account configuration. Local storage avoids a service dependency.

**Alternatives.** A per-user application directory, database server or account service. They would add setup or policy unnecessary to the example.

**Consequences.** The working directory affects the default selection. Guides must explain that effect. Runtime data must not be confused with the canonical specification or committed source.

<a id="decision-language"></a>
## DEC-LANGUAGE — Implementation language

Type: Decision. Choice status: deliberately left to implementation.

**Context.** The example evaluates a documentation model, not a language framework. No domain or caller requirement depends on a particular programming language.

**Freedom.** Select a language/toolchain that realizes the contract and supports the declared Checks. Before realization, record its versions, concrete build/run/test entrypoints and manifest paths in the tooling Artifact and guides. This is an implementation choice, not a missing public behavior.

**Consequence.** A complete implementation can be assessed against this specification without this package selecting Go, Python or another language.

<a id="decision-encoding"></a>
## DEC-ENCODING — File representation

Type: Decision. Choice status: deliberately left to implementation.

**Context.** The application owns reading/writing its data file. Version 1 does not offer import, export, manual editing or interoperability with another application.

**Freedom.** Select a file representation capable of round-tripping the Task collection, identifying unsupported data and fulfilling safe writes. Keep encoding details in storage; record its supported format/version and invalid-input behavior with the realized storage documentation/tests.

**Consequence.** Future format migration is a new contract question. No automatic migration or backwards compatibility across future releases is promised here. The example still has complete command behavior and observable persistence requirements.

<a id="decision-boundary"></a>
## DEC-BOUNDARY — Bounded first version

Type: Decision. Choice status: selected in this draft.

**Context.** The example must stay small enough to test the modeling vocabulary rather than repeat Konfyra's complexity.

**Choice owner.** REQ-SCOPE owns the offered boundary; REQ-LOCAL owns execution guarantees; the Components own responsibility separation.

**Rationale.** Three actions and sequential local use keep the interaction and verification understandable. Responsibilities do not require framework layers, a server or a permanent agent organization.

**Alternatives.** Supporting simultaneous writers, interruption recovery or broader task management now. Those need further requirements and evidence, and are intentionally not part of this draft.

**Consequences.** Ordinary storage errors still need safe behavior. Scope exclusions must not be used to excuse failures that are inside the defined normal-operation contract.

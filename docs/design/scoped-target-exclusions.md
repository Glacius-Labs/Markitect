# Scoped target exclusions

## Current boundary

At source revision `5acbd06df3e81954ae667393e8f458d4f2ac4140`, the Host scoped reconciliation controller has no owner-supplied exact-path exclusion capability. The historical C7 matrix entry remains **NOT RUN**: this records an absent capability, not an observed failing experiment. No C7 experiment has passed yet.

When a selected Projection target root contains an unowned artifact, the current planner reports the path as unknown and escalates. There is no supported way for the owner to state that one exact path is intentionally outside the selected projection's artifact scope while keeping its metadata visible. The planner must continue to stop on unknown, non-excluded target artifacts.

## Candidate behavior

The controller may accept a bounded set of explicit owner-supplied exact paths, each with a required reason. Exclusions are limited to paths beneath a declared target root and cannot overlap active ownership, canonical inputs, declared check inputs, or path aliases. They do not support globs, directories, recursive prefixes, or inferred patterns. They make no assertion about excluded file contents.

Excluded paths remain in the target inventory and in the proposal's explicit excluded-artifact classification, but their bytes are never acquired. Exclusion metadata, reasons, and state are bound into the plan digest so a changed claim invalidates the plan. Missing excluded paths remain visible as absent metadata; their absence does not make scoped reconciliation complete or passing. Every non-excluded unknown artifact continues to escalate, and unknown omitted bytes remain unknown.

Exclusions define owner-selected artifact input scope for a bounded Module invocation. The Host records that scope but does not infer or claim that a Module semantically understood every file beneath the root. Exclusions are never inferred from history, path names, content, prior output, or Module behavior.

## Evidence status

The behavior above is proposed design. It needs source-bound controls demonstrating exact-path validation, preservation of unknowns, and non-acquisition of excluded bytes before any C7 result can be recorded as passed. This note does not report those controls as run or passed.

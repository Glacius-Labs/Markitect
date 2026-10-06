# C3 negative specifications

These are frozen controls for later fresh Executor/Verifier trials. Do not treat the mutations or this fixture's tests as evidence that an agent was run.

## Absent key intent

At a new immutable control revision, edit only `examples/unknown-ontology/definitions/mission-dotnet.policy.yaml` and remove the `guidance` field under `spec`, leaving `sourceKind` and `targetTechnology` intact. Expected result: canonical schema validation or Prepare rejects the incomplete policy before materialization; if the selected runtime reaches the Executor, it must escalate without inventing an API or behavior. No candidate is accepted.

## Syntactically present but insufficient intent

At a separate immutable control revision, keep the same policy fields and replace the `guidance` block with exactly: `Represent the selected Mission as a clear .NET type.` The field remains valid text, but it no longer declares the API, typed relationships, unit matching, range semantics, Capability maximum, or checker contract. Expected result: a fresh Executor escalates for insufficient meaning; it must not infer method names, values, or rules from the check source. A rendered class or compiling candidate is not PASS.

## Positive behavior controls

With the unmodified source and exact package pins, the candidate must represent the three selected Kinds and implement the policy. The independent configured check rejects a mismatched axis, capability, or unit; an EffectAxis lower/upper bound violation; and a Capability maximum violation. These controls show only the stated bounded behavior when actually executed on exact candidate bytes.

Any real C3 result needs a fresh configured Executor and separate Verifier, both bound to the exact committed source and candidate. This fixture, structural loading, Prepare escalation, and an independently green check alone do not establish agent behavior or broad ontology-agnostic projection.
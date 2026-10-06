# Unknown ontology projection fixture

This fixture tests a bounded C3 question with a newly named, project-owned ontology: can an executor use explicit target policy to represent and implement behavior over `Mission`, `EffectAxis`, and `Capability` without a Core or Host special case for those Kinds?

The exact-pinned Schema Module declares all three Kinds and typed references. The canonical Mission points to an EffectAxis and Capability; that Capability points back to the same EffectAxis. One .NET Projection selects all three Definitions and one policy per Kind. The Projection-only Module uses the standard `dotnet-source` capability, a single allowed root, and one exact required check. The policy states public record signatures and the `MissionEvaluator.IsSupported` behavior; it does not prescribe filenames or a code template.

The check compiles only the explicitly declared `src/UnknownOntology/**/*.cs` target input using .NET 8 framework references. Its positive and negative cases make the behavior depend on each selected Kind: exact typed-reference matches, requested and supported units, inclusive EffectAxis bounds, and the Capability limit. The check is an adopter-owned fixed contract, not a claim that the vocabulary or policy covers a broader mission-planning problem.

`unknown_ontology_test.go` binds source loading and `PrepareCanonicalProjection` to the actual full Git HEAD, exact package pins, and declared scope. It checks structural selection and the expected no-candidate escalation. It does not execute an Executor or Verifier and is not an AI behavior result. See `negative-specifications.md` for the exact absent- and insufficient-guidance controls to use in fresh agent trials.

No adopter data, automatic acceptance, or actual agent result is included.
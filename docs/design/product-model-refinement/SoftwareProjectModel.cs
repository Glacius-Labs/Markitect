// Illustrative C# design, 10 October 2026. Not Markitect production code.
// No project/entrypoint; not compiled or tested. See csharp-model-design.md.
// C# records describe canonical definitions, not running library objects.
using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Linq;

namespace Markitect.ConceptDraft;

public readonly record struct DefinitionId(string Value);
public readonly record struct QualifiedName(string Namespace, string LocalName);
public readonly record struct Ref<T>(DefinitionId Id) where T : Definition;

public abstract record Definition(
    DefinitionId Id, QualifiedName Name, string Meaning);

// Authoring can use separate files. This catalog is a composed reading view.
public sealed record DefinitionCatalog(
    ImmutableDictionary<DefinitionId, Definition> Definitions)
{
    public T Resolve<T>(Ref<T> reference) where T : Definition =>
        Definitions.TryGetValue(reference.Id, out var value) && value is T typed
            ? typed
            : throw new InvalidOperationException(
                $"Unresolved or incompatible reference: {reference.Id.Value}");
}

public sealed record ModelScope(
    ImmutableArray<string> Included,
    ImmutableArray<string> Excluded);

public sealed record SoftwareSystem(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ModelScope Boundary,
    ImmutableArray<Ref<ProblemDefinition>> Problems,
    ImmutableArray<Ref<GoalDefinition>> Goals,
    ImmutableArray<Ref<ActorDefinition>> Actors,
    ImmutableArray<Ref<DomainTypeDefinition>> DomainTypes,
    ImmutableArray<Ref<AssociationDefinition>> Associations,
    ImmutableArray<Ref<CapabilityDefinition>> Capabilities,
    ImmutableArray<Ref<ComponentDefinition>> RootComponents,
    ImmutableArray<Ref<DependencyDefinition>> Dependencies,
    ImmutableArray<Ref<RuleDefinition>> Rules,
    ImmutableArray<Ref<QualityRequirement>> QualityRequirements,
    ImmutableArray<Ref<OperatingModel>> OperatingModels)
    : Definition(Id, Name, Meaning)
{
    public IEnumerable<CapabilityDefinition> GetCapabilities(
        DefinitionCatalog catalog) => Capabilities.Select(reference => catalog.Resolve(reference));

    public IEnumerable<DomainTypeDefinition> GetDomainTypes(
        DefinitionCatalog catalog) => DomainTypes.Select(reference => catalog.Resolve(reference));
}

public sealed record ProblemDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning)
    : Definition(Id, Name, Meaning);

public sealed record GoalDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<ProblemDefinition>> Addresses,
    string SuccessCriterion,
    ImmutableArray<string> Assumptions)
    : Definition(Id, Name, Meaning);

public enum ActorKind { Person, ExternalSystem, Organization }
public sealed record ActorDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ActorKind Kind, string Interest)
    : Definition(Id, Name, Meaning);

public enum ScalarKind { Text, Boolean, Integer, Decimal, Instant, Duration }
public abstract record PropertyType;
public sealed record ScalarPropertyType(
    ScalarKind Kind, string? Unit = null) : PropertyType;
public sealed record ReferencePropertyType(
    Ref<DomainTypeDefinition> Target) : PropertyType;
public sealed record ChoicePropertyType(
    ImmutableArray<string> AllowedValues) : PropertyType;

public sealed record Cardinality(int Minimum, int? Maximum);
public sealed record PropertyDefinition(
    string Name, string Meaning, PropertyType Type,
    Cardinality Multiplicity, bool IdentifiesInstance = false);

public enum DomainTypeKind { Entity, ValueObject, Event, ExternalConcept }
public sealed record DomainTypeDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning, DomainTypeKind Kind,
    ImmutableArray<PropertyDefinition> Properties,
    ImmutableArray<Ref<BehaviorContract>> Operations,
    ImmutableArray<Ref<RuleDefinition>> Invariants)
    : Definition(Id, Name, Meaning);

// Domain instance association; cardinality is NOT a count of model definitions.
public sealed record AssociationDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    Ref<DomainTypeDefinition> Source,
    Ref<DomainTypeDefinition> Target,
    Cardinality TargetsPerSource,
    string? ViaTargetProperty = null)
    : Definition(Id, Name, Meaning);

public sealed record CapabilityDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<ActorDefinition>> Actors,
    Ref<BehaviorContract> Contract)
    : Definition(Id, Name, Meaning);

public sealed record FailureOutcome(
    string Name, string Meaning,
    ImmutableArray<Ref<RuleDefinition>> RequiredConditions);

public sealed record BehaviorContract(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<PropertyDefinition> Inputs,
    ImmutableArray<PropertyDefinition> Outputs,
    ImmutableArray<Ref<RuleDefinition>> Preconditions,
    ImmutableArray<Ref<RuleDefinition>> Postconditions,
    ImmutableArray<Ref<RuleDefinition>> Maintains,
    ImmutableArray<FailureOutcome> Failures,
    ImmutableArray<Ref<Definition>> Affects,
    ImmutableArray<string> AllowedImplementationChoices)
    : Definition(Id, Name, Meaning);

// Prose is canonical meaning; a CheckProcedure is an explicit evidence contract.
// This sketch does not include a formal expression evaluator or arbitrary lambdas.
public enum RuleCategory { Invariant, Precondition, Postcondition, Policy }
public sealed record RuleScope(
    ImmutableArray<Ref<Definition>> Targets,
    bool IncludeContainedDefinitions = false);

public sealed record RuleDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning, RuleCategory Category,
    RuleScope Scope,
    ImmutableArray<Ref<CheckProcedure>> Checks,
    string Rationale)
    : Definition(Id, Name, Meaning);

public enum PortDirection { Provides, Requires }
public sealed record PortDefinition(
    string Name, PortDirection Direction, Ref<BehaviorContract> Contract);

public sealed record ComponentDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<ComponentDefinition>> Children,
    ImmutableArray<PortDefinition> Ports,
    ImmutableArray<Ref<Definition>> Realizes,
    ImmutableArray<string> AllowedImplementationChoices)
    : Definition(Id, Name, Meaning);

public enum DependencyKind { SourceCode, RuntimeCall, Information }
public sealed record DependencyDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    Ref<ComponentDefinition> Source, Ref<ComponentDefinition> Target,
    DependencyKind Kind)
    : Definition(Id, Name, Meaning);

public enum CheckKind { ModelStructure, RepositoryCoverage, Behavior, Review }
public sealed record CheckProcedure(
    DefinitionId Id, QualifiedName Name, string Meaning,
    CheckKind Kind, ImmutableArray<Ref<Definition>> Subjects,
    string Method, string ExpectedEvidence)
    : Definition(Id, Name, Meaning);

public sealed record QualityRequirement(
    DefinitionId Id, QualifiedName Name, string Meaning,
    RuleScope Scope, string Criterion,
    ImmutableArray<Ref<CheckProcedure>> Checks)
    : Definition(Id, Name, Meaning);

public sealed record ConfigurationParameter(
    string Name, PropertyType Type, string Meaning,
    string AllowedValues, string? DefaultValue);

public sealed record OperatingModel(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<string> RuntimeUnits,
    ImmutableArray<string> Environments,
    ImmutableArray<ConfigurationParameter> Configuration,
    ImmutableArray<Ref<WorkflowDefinition>> Procedures,
    ImmutableArray<Ref<RuleDefinition>> Constraints)
    : Definition(Id, Name, Meaning);

public sealed record ResponsibilityDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    Ref<ResponsibilityDefinition>? Parent,
    ImmutableArray<Ref<Definition>> Scope,
    ImmutableArray<string> DecisionAuthority,
    ImmutableArray<string> RequiredReviews)
    : Definition(Id, Name, Meaning);

public sealed record WorkflowStep(
    string Name, Ref<BehaviorContract> Operation,
    Ref<ResponsibilityDefinition> Responsible);

public sealed record WorkflowDefinition(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<WorkflowStep> Steps,
    ImmutableArray<Ref<RuleDefinition>> Rules)
    : Definition(Id, Name, Meaning);

public enum ArtifactRole
{
    ApplicationCode, DomainCode, Test, UserDocumentation,
    OperationsDocumentation, Migration, Configuration,
    Pipeline, Infrastructure, Skill, GeneratedView
}
public enum Presence { Required, Optional, Prohibited }
public enum ContributionKind { Implements, Verifies, Explains, Configures, Deploys }
public sealed record ModelContribution(
    Ref<Definition> Subject, ContributionKind Kind);

public sealed record LayoutConvention(
    DefinitionId Id, QualifiedName Name, string Meaning,
    string RootDirectory, string? NamespaceConvention)
    : Definition(Id, Name, Meaning);

// Semantic production contract, not a provider/module executable binding.
public sealed record GeneratedOutputContract(
    Ref<BehaviorContract> ProducerContract,
    ImmutableArray<Ref<Definition>> AuthoritativeInputs);

public sealed record ArtifactExpectation(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ArtifactRole Role, Presence Presence,
    Ref<LayoutConvention>? Location,
    ImmutableArray<ModelContribution> Contributions,
    Ref<ResponsibilityDefinition> WriteOwner,
    GeneratedOutputContract? GeneratedOutput = null)
    : Definition(Id, Name, Meaning);

public sealed record ExclusionDeclaration(
    string ScopeDescription, string Reason);
public sealed record CoveragePolicy(
    bool RequireEveryFileAccountedFor,
    ImmutableArray<ExclusionDeclaration> ExplicitExclusions);

// Source repository description; unrelated to the DDD persistence Repository pattern.
public sealed record Repository(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<SoftwareSystem>> Systems,
    ImmutableArray<Ref<ArtifactExpectation>> ExpectedArtifacts,
    ImmutableArray<Ref<LayoutConvention>> Layout,
    ImmutableArray<Ref<RuleDefinition>> Policies,
    CoveragePolicy Coverage)
    : Definition(Id, Name, Meaning)
{
    public IEnumerable<ArtifactExpectation> GetExpectations(
        DefinitionCatalog catalog) => ExpectedArtifacts.Select(reference => catalog.Resolve(reference));

    public IEnumerable<ArtifactExpectation> GetExpectedFor(
        DefinitionId subject, DefinitionCatalog catalog) =>
        GetExpectations(catalog).Where(expectation =>
            expectation.Contributions.Any(item => item.Subject.Id == subject));
}

public sealed record SoftwareProject(
    DefinitionId Id, QualifiedName Name, string Meaning,
    ImmutableArray<Ref<SoftwareSystem>> Systems,
    ImmutableArray<Ref<Repository>> Repositories,
    ImmutableArray<Ref<ResponsibilityDefinition>> Responsibilities,
    ImmutableArray<Ref<WorkflowDefinition>> Workflows,
    ImmutableArray<Ref<RuleDefinition>> Rules)
    : Definition(Id, Name, Meaning);

public sealed record NamespaceDeclaration(
    string Path, string Meaning, string? ParentPath);
public sealed record CanonicalModel(
    Ref<SoftwareProject> Project,
    ImmutableArray<NamespaceDeclaration> Namespaces,
    DefinitionCatalog Catalog);

// Observations belong to a fixed realization; none are canonical declarations.
public sealed record RepositorySnapshot(
    Ref<Repository> Repository, string Digest);
public enum AssignmentState { Bound, Unclassified, Ambiguous, Excluded }
public sealed record ObservedArtifact(
    string RelativePath, string ContentDigest,
    AssignmentState Assignment,
    ImmutableArray<Ref<ArtifactExpectation>> Expectations,
    // Derived from bindings and policy; conflicting owners make assignment ambiguous.
    Ref<ResponsibilityDefinition>? ResolvedWriteOwner,
    string MappingEvidence);
public enum CoverageExtent { Full, Partial, Unknown }
public sealed record RealizationIndex(
    RepositorySnapshot Snapshot, CoverageExtent Extent,
    ImmutableArray<ObservedArtifact> Artifacts);

public sealed record Diagnostic(
    string Code, string Message, ImmutableArray<DefinitionId> Subjects);
public sealed record CompiledModel(
    CanonicalModel Source, string Digest,
    ImmutableArray<Diagnostic> StructuralFindings);

public enum FindingStatus { Passed, Failed, NotChecked, Undetermined }
public sealed record VerificationFinding(
    Ref<CheckProcedure> Check, FindingStatus Status,
    ImmutableArray<DefinitionId> Subjects, string Evidence);
public sealed record VerificationReport(
    string ModelDigest, string RepositoryDigest,
    ImmutableArray<string> CheckAndToolIdentities,
    ImmutableArray<VerificationFinding> Findings);
public sealed record ImpactReport(
    ImmutableArray<DefinitionId> AffectedDefinitions,
    ImmutableArray<string> ConcretePaths,
    ImmutableArray<Diagnostic> UnknownOrAmbiguousScopes);

// Proposed service boundaries only; no implementation or provider execution here.
public interface IModelCompiler
{
    CompiledModel Compile(CanonicalModel model);
}
public interface IRepositoryInspector
{
    RealizationIndex Inspect(Repository description,
        RepositorySnapshot snapshot, CompiledModel model);
}
public interface IVerificationService
{
    VerificationReport Verify(CompiledModel model,
        Repository description, RealizationIndex index);
}
public interface IImpactAnalyzer
{
    ImpactReport Analyze(CompiledModel model,
        ImmutableArray<DefinitionId> changedDefinitions,
        RealizationIndex index);
}

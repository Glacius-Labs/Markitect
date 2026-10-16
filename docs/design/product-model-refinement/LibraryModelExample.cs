// Fictional, bounded circulation example. No execution of library operations.
// Uses SoftwareProjectModel.cs; not compiled or tested.
using System.Collections.Immutable;
using System.Linq;

namespace Markitect.ConceptDraft;

public static class LibraryModelExample
{
    public static CanonicalModel Create()
    {
        var definitions = ImmutableDictionary.CreateBuilder<DefinitionId, Definition>();
        void Add(Definition definition) => definitions.Add(definition.Id, definition);

        Add(new ProblemDefinition(Id("problem"), Name("problem"),
            "Verfügbarkeit und Verbleib von Exemplaren nachvollziehen."));
        Add(new GoalDefinition(Id("goal"), Name("goal"),
            "Mitglieder erhalten zeitweiligen Zugang zu Büchern.",
            Items(Reference<ProblemDefinition>("problem")),
            "Verfügbarkeit und Verbleib im gewählten Bestand sind nachvollziehbar.",
            Items("Für diese Nutzung ist zeitweiliger Zugang ausreichend.")));
        Add(new ActorDefinition(Id("reader"), Name("reader"),
            "Mitglied, das Bücher ausleiht.", ActorKind.Person,
            "Bücher lesen, ohne jedes Buch kaufen zu müssen."));

        Add(new RuleDefinition(Id("one-active-loan"), Name("one-active-loan"),
            "Pro Exemplar existiert höchstens eine aktive Ausleihe. Aktiv bedeutet returnedAt fehlt.",
            RuleCategory.Invariant, new RuleScope(Subjects("copy", "loan")),
            Items(Reference<CheckProcedure>("concurrent-borrow-check")),
            "Ein physisches Exemplar kann nicht gleichzeitig an zwei Personen verliehen sein."));
        Add(new RuleDefinition(Id("eligible-member"), Name("eligible-member"),
            "Das Mitglied existiert und isEligible ist wahr.", RuleCategory.Precondition,
            new RuleScope(Subjects("member", "borrow-contract")),
            Items<Ref<CheckProcedure>>(), "Ausleihen erfordern eine Berechtigung."));
        Add(new RuleDefinition(Id("available-copy"), Name("available-copy"),
            "Das Exemplar ist intakt und hat keine aktive Ausleihe.", RuleCategory.Precondition,
            new RuleScope(Subjects("copy", "borrow-contract")),
            Items<Ref<CheckProcedure>>(), "Nur verfügbare Exemplare werden ausgeliehen."));
        Add(new RuleDefinition(Id("loan-created"), Name("loan-created"),
            "Erfolg erzeugt eine dauerhaft gespeicherte aktive Ausleihe mit Mitglied, Exemplar und Frist. Die Frist ergibt sich aus startedAt und der geltenden loanDuration.",
            RuleCategory.Postcondition, new RuleScope(Subjects("borrow-contract", "loan")),
            Items<Ref<CheckProcedure>>(), "Erfolg ist als tatsächliche Zuordnung nachvollziehbar."));
        Add(new RuleDefinition(Id("rejection-no-change"), Name("rejection-no-change"),
            "Bei Ablehnung entsteht keine zusätzliche Ausleihe.", RuleCategory.Postcondition,
            new RuleScope(Subjects("borrow-contract")), Items<Ref<CheckProcedure>>(),
            "Eine abgelehnte Anfrage hat keine erfolgreiche Zustandswirkung."));
        Add(new RuleDefinition(Id("loan-dates"), Name("loan-dates"),
            "dueAt liegt nach startedAt. Ein vorhandenes returnedAt liegt nicht vor startedAt.",
            RuleCategory.Invariant, new RuleScope(Subjects("loan")),
            Items<Ref<CheckProcedure>>(), "Die zeitliche Beschreibung einer Ausleihe ist konsistent."));
        Add(new RuleDefinition(Id("return-active"), Name("return-active"),
            "Die Ausleihe existiert und ist noch aktiv.", RuleCategory.Precondition,
            new RuleScope(Subjects("loan", "return-contract")),
            Items<Ref<CheckProcedure>>(), "Dieser vereinfachte Vertrag erlaubt nur die erste Rückgabe."));
        Add(new RuleDefinition(Id("return-recorded"), Name("return-recorded"),
            "Die Rückgabe setzt returnedAt dauerhaft. Die Ausleihe bleibt als Historie erhalten und ist nicht mehr aktiv.",
            RuleCategory.Postcondition, new RuleScope(Subjects("loan", "return-contract")),
            Items<Ref<CheckProcedure>>(), "Rückgabe beendet die Zuordnung ohne die Historie zu löschen."));
        Add(new RuleDefinition(Id("return-rejected"), Name("return-rejected"),
            "Eine abgelehnte Rückgabe verändert die Ausleihdaten nicht.", RuleCategory.Postcondition,
            new RuleScope(Subjects("loan", "return-contract")),
            Items<Ref<CheckProcedure>>(), "Ablehnung erhält den bisherigen Fachzustand."));

        Add(new DomainTypeDefinition(Id("title"), Name("title"),
            "Bibliografischer Titel, unabhängig von einzelnen Exemplaren.", DomainTypeKind.Entity,
            Items(Property("titleId", Text, identity: true), Property("name", Text)),
            Items<Ref<BehaviorContract>>(), Items<Ref<RuleDefinition>>()));
        Add(new DomainTypeDefinition(Id("copy"), Name("copy"),
            "Ein konkretes physisches Exemplar eines Titels.", DomainTypeKind.Entity,
            Items(Property("copyId", Text, identity: true),
                Property("title", new ReferencePropertyType(Reference<DomainTypeDefinition>("title"))),
                Property("condition", new ChoicePropertyType(Items("intact", "damaged", "removed")))),
            Items<Ref<BehaviorContract>>(), Items(Reference<RuleDefinition>("one-active-loan"))));
        Add(new DomainTypeDefinition(Id("member"), Name("member"),
            "Registriertes Mitglied im beschriebenen Ausleihsystem.", DomainTypeKind.Entity,
            Items(Property("memberId", Text, identity: true),
                Property("isEligible", new ScalarPropertyType(ScalarKind.Boolean))),
            Items<Ref<BehaviorContract>>(), Items<Ref<RuleDefinition>>()));
        Add(new DomainTypeDefinition(Id("loan"), Name("loan"),
            "Zeitlich begrenzte Zuordnung zwischen Mitglied und Exemplar.", DomainTypeKind.Entity,
            Items(Property("loanId", Text, identity: true),
                Property("copy", new ReferencePropertyType(Reference<DomainTypeDefinition>("copy"))),
                Property("member", new ReferencePropertyType(Reference<DomainTypeDefinition>("member"))),
                Property("startedAt", new ScalarPropertyType(ScalarKind.Instant)),
                Property("dueAt", new ScalarPropertyType(ScalarKind.Instant)),
                Property("returnedAt", new ScalarPropertyType(ScalarKind.Instant), optional: true)),
            Items(Reference<BehaviorContract>("return-contract")),
            Items(Reference<RuleDefinition>("one-active-loan"), Reference<RuleDefinition>("loan-dates"))));

        Add(new AssociationDefinition(Id("title-copies"), Name("title-copies"),
            "Ein Titel besitzt null bis beliebig viele Exemplare.",
            Reference<DomainTypeDefinition>("title"), Reference<DomainTypeDefinition>("copy"),
            new Cardinality(0, null), "title"));
        Add(new AssociationDefinition(Id("copy-loans"), Name("copy-loans"),
            "Ein Exemplar kann über die Zeit viele Ausleihen haben; aktive sind gesondert beschränkt.",
            Reference<DomainTypeDefinition>("copy"), Reference<DomainTypeDefinition>("loan"),
            new Cardinality(0, null), "copy"));
        Add(new AssociationDefinition(Id("member-loans"), Name("member-loans"),
            "Ein Mitglied kann über die Zeit viele Ausleihen haben.",
            Reference<DomainTypeDefinition>("member"), Reference<DomainTypeDefinition>("loan"),
            new Cardinality(0, null), "member"));

        Add(new BehaviorContract(Id("borrow-contract"), Name("borrow-contract"),
            "Ein berechtigtes Mitglied leiht ein verfügbares Exemplar aus.",
            Items(Property("member", new ReferencePropertyType(Reference<DomainTypeDefinition>("member"))),
                Property("copy", new ReferencePropertyType(Reference<DomainTypeDefinition>("copy")))),
            Items(Property("loan", new ReferencePropertyType(Reference<DomainTypeDefinition>("loan")))),
            Items(Reference<RuleDefinition>("eligible-member"), Reference<RuleDefinition>("available-copy")),
            Items(Reference<RuleDefinition>("loan-created")),
            Items(Reference<RuleDefinition>("one-active-loan"), Reference<RuleDefinition>("loan-dates")),
            Items(new FailureOutcome("Rejected", "Nicht berechtigt oder nicht verfügbar.",
                Items(Reference<RuleDefinition>("rejection-no-change")))),
            Subjects("loan"),
            Items("Technik der atomaren Vergabe ist frei, solange die Invariante gilt.")));
        Add(new CapabilityDefinition(Id("borrow"), Name("borrow"),
            "Ein Exemplar zeitweilig nutzen.", Items(Reference<ActorDefinition>("reader")),
            Reference<BehaviorContract>("borrow-contract")));
        Add(new BehaviorContract(Id("return-contract"), Name("return-contract"),
            "Eine aktive Ausleihe als zurückgegeben erfassen.",
            Items(Property("loan", new ReferencePropertyType(Reference<DomainTypeDefinition>("loan")))),
            Items(Property("loan", new ReferencePropertyType(Reference<DomainTypeDefinition>("loan")))),
            Items(Reference<RuleDefinition>("return-active")),
            Items(Reference<RuleDefinition>("return-recorded")),
            Items(Reference<RuleDefinition>("one-active-loan"), Reference<RuleDefinition>("loan-dates")),
            Items(new FailureOutcome("Rejected", "Ausleihe unbekannt oder bereits beendet.",
                Items(Reference<RuleDefinition>("return-rejected")))),
            Subjects("loan"), Items("Erfassung der physischen Rückgabe erfolgt außerhalb dieses Vertrags.")));
        Add(new CapabilityDefinition(Id("return"), Name("return"),
            "Eine Rückgabe im System erfassen.", Items(Reference<ActorDefinition>("reader")),
            Reference<BehaviorContract>("return-contract")));

        Add(Component("inventory", "Exemplare und deren Zustand.", Subjects("title", "copy")));
        Add(Component("membership", "Mitgliedsdaten und Berechtigung.", Subjects("member")));
        Add(Component("circulation", "Ausleihvorgänge koordinieren.",
            Subjects("loan", "borrow-contract", "return-contract"),
            Items(new PortDefinition("borrow", PortDirection.Provides,
                    Reference<BehaviorContract>("borrow-contract")),
                new PortDefinition("return", PortDirection.Provides,
                    Reference<BehaviorContract>("return-contract")))));
        Add(new DependencyDefinition(Id("uses-inventory"), Name("uses-inventory"),
            "Ausleihe benötigt Information zur Exemplarverfügbarkeit.",
            Reference<ComponentDefinition>("circulation"), Reference<ComponentDefinition>("inventory"),
            DependencyKind.Information));
        Add(new DependencyDefinition(Id("uses-membership"), Name("uses-membership"),
            "Ausleihe benötigt Information zur Mitgliedsberechtigung.",
            Reference<ComponentDefinition>("circulation"), Reference<ComponentDefinition>("membership"),
            DependencyKind.Information));

        Add(new CheckProcedure(Id("concurrent-borrow-check"), Name("concurrent-borrow-check"),
            "Prüfumfang der konkurrierenden Vergabe.", CheckKind.Behavior,
            Subjects("one-active-loan", "borrow-contract"),
            "Zwei berechtigte Anfragen gleichzeitig auf dasselbe freie Exemplar ausführen.",
            "Genau ein Erfolg, eine Ablehnung und genau eine dauerhaft gespeicherte aktive Ausleihe."));
        Add(new QualityRequirement(Id("concurrency"), Name("concurrency"),
            "Ausleihe erhält ihre Invariante auch bei gleichzeitigen Anfragen.",
            new RuleScope(Subjects("borrow-contract")),
            "Der beschriebene konkurrierende Fall erfüllt die erwarteten Ergebnisse.",
            Items(Reference<CheckProcedure>("concurrent-borrow-check"))));
        Add(new OperatingModel(Id("operation"), Name("operation"),
            "Anwendung mit dauerhaftem Zustand.", Items("Application", "PersistentStore"),
            Items("Development", "Production"),
            Items(new ConfigurationParameter("loanDuration",
                new ScalarPropertyType(ScalarKind.Duration, "days"),
                "Dauer bis zum regulären Rückgabetermin.", "Positiv; genauere Grenzen sind offen.", null)),
            Items<Ref<WorkflowDefinition>>(), Items<Ref<RuleDefinition>>()));

        Add(new SoftwareSystem(Id("system"), Name("system"), "Bibliothekssoftware.",
            new ModelScope(Items("Ausleihe und Rückgabe mit beschriebenen Bestands- und Mitgliedsdaten"),
                Items("Bestandspflege, Berechtigungsvergabe, Beschaffung, Reinigung, Preisbildung und automatische Sortimentsentscheidung")),
            Items(Reference<ProblemDefinition>("problem")), Items(Reference<GoalDefinition>("goal")),
            Items(Reference<ActorDefinition>("reader")),
            Items("title", "copy", "member", "loan").Select(Reference<DomainTypeDefinition>).ToImmutableArray(),
            Items(Reference<AssociationDefinition>("title-copies"), Reference<AssociationDefinition>("copy-loans"),
                Reference<AssociationDefinition>("member-loans")),
            Items(Reference<CapabilityDefinition>("borrow"), Reference<CapabilityDefinition>("return")),
            Items("inventory", "membership", "circulation").Select(Reference<ComponentDefinition>).ToImmutableArray(),
            Items(Reference<DependencyDefinition>("uses-inventory"), Reference<DependencyDefinition>("uses-membership")),
            Items("one-active-loan", "eligible-member", "available-copy", "loan-created", "rejection-no-change",
                "loan-dates", "return-active", "return-recorded", "return-rejected")
                .Select(Reference<RuleDefinition>).ToImmutableArray(),
            Items(Reference<QualityRequirement>("concurrency")), Items(Reference<OperatingModel>("operation"))));

        Add(new ResponsibilityDefinition(Id("development"), Name("development"),
            "Verantwortung für die Artefakte dieses kleinen Beispiels.", null,
            Subjects("system", "repository"),
            Items("Umsetzung innerhalb der beschriebenen Verträge"),
            Items("Unabhängiges Review von Umsetzung, Prüfungen und Integration")));

        Add(Layout("source-layout", "src", "Library.{area}"));
        Add(Layout("test-layout", "tests", null));
        Add(Layout("docs-layout", "docs", null));
        Add(Layout("data-layout", "migrations", null));
        Add(Expectation("application", ArtifactRole.ApplicationCode, "source-layout",
            Items(new ModelContribution(Reference<Definition>("borrow-contract"), ContributionKind.Implements),
                new ModelContribution(Reference<Definition>("return-contract"), ContributionKind.Implements))));
        Add(Expectation("tests", ArtifactRole.Test, "test-layout",
            Items(new ModelContribution(Reference<Definition>("concurrent-borrow-check"), ContributionKind.Implements),
                new ModelContribution(Reference<Definition>("one-active-loan"), ContributionKind.Verifies))));
        Add(Expectation("documentation", ArtifactRole.UserDocumentation, "docs-layout",
            Items(new ModelContribution(Reference<Definition>("borrow"), ContributionKind.Explains),
                new ModelContribution(Reference<Definition>("return"), ContributionKind.Explains))));
        Add(Expectation("data", ArtifactRole.Migration, "data-layout",
            Items(new ModelContribution(Reference<Definition>("loan"), ContributionKind.Implements))));
        Add(new Repository(Id("repository"), Name("repository"),
            "Quellrepository der Bibliothekssoftware.", Items(Reference<SoftwareSystem>("system")),
            Items("application", "tests", "documentation", "data")
                .Select(Reference<ArtifactExpectation>).ToImmutableArray(),
            Items("source-layout", "test-layout", "docs-layout", "data-layout")
                .Select(Reference<LayoutConvention>).ToImmutableArray(),
            Items<Ref<RuleDefinition>>(), new CoveragePolicy(true, Items<ExclusionDeclaration>())));
        Add(new SoftwareProject(Id("project"), Name("project"), "Projekt des kleinen Beispiels.",
            Items(Reference<SoftwareSystem>("system")), Items(Reference<Repository>("repository")),
            Items(Reference<ResponsibilityDefinition>("development")),
            Items<Ref<WorkflowDefinition>>(), Items<Ref<RuleDefinition>>()));

        return new CanonicalModel(Reference<SoftwareProject>("project"),
            Items(new NamespaceDeclaration("library", "Vokabular dieses Beispiels.", null)),
            new DefinitionCatalog(definitions.ToImmutable()));
    }

    private static DefinitionId Id(string name) => new(name);
    private static QualifiedName Name(string name) => new("library", name);
    private static Ref<T> Reference<T>(string name) where T : Definition => new(Id(name));
    private static ImmutableArray<Ref<Definition>> Subjects(params string[] names) =>
        names.Select(name => Reference<Definition>(name)).ToImmutableArray();
    private static ImmutableArray<T> Items<T>(params T[] items) => items.ToImmutableArray();
    private static PropertyType Text => new ScalarPropertyType(ScalarKind.Text);
    private static PropertyDefinition Property(string name, PropertyType type,
        bool identity = false, bool optional = false) =>
        new(name, name, type, new Cardinality(optional ? 0 : 1, 1), identity);
    private static ComponentDefinition Component(string name, string meaning,
        ImmutableArray<Ref<Definition>> definitions,
        ImmutableArray<PortDefinition>? ports = null) =>
        new(Id(name), Name(name), meaning, Items<Ref<ComponentDefinition>>(),
            ports ?? Items<PortDefinition>(), definitions, Items<string>());
    private static LayoutConvention Layout(string name, string directory, string? convention) =>
        new(Id(name), Name(name), "Vereinbarter Ablagebereich.", directory, convention);
    private static ArtifactExpectation Expectation(string name, ArtifactRole role,
        string layout, ImmutableArray<ModelContribution> contributions) =>
        new(Id(name), Name(name), "Gewählte Artefakterwartung dieses Beispiels.", role,
            Presence.Required, Reference<LayoutConvention>(layout), contributions,
            Reference<ResponsibilityDefinition>("development"));
}

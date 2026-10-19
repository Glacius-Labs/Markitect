package apply

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var coreKind = model.Identity{Package: "markitect/core", ID: "Kind"}

// vocabularyKind is one Kind of an imported vocabulary that Definitions can use.
type vocabularyKind struct {
	Identity model.Identity
	Title    string
	Body     string
	Spec     []model.Field
}

// vocabularyOf lists the Kinds the compiled model's dependencies define, apart
// from the fixed core, in identity order.
func vocabularyOf(compiled model.Result) []vocabularyKind {
	var kinds []vocabularyKind
	for _, owner := range compiled.Models {
		for _, d := range owner.Definitions {
			if d.Kind == coreKind && d.Identity.Package != coreKind.Package {
				kinds = append(kinds, vocabularyKind{Identity: d.Identity, Title: definitionTitle(d), Body: strings.Join(strings.Fields(d.Body), " "), Spec: d.Spec})
			}
		}
	}
	slices.SortFunc(kinds, func(a, b vocabularyKind) int {
		return cmp.Or(cmp.Compare(a.Identity.Package, b.Identity.Package), cmp.Compare(a.Identity.ID, b.Identity.ID))
	})
	return kinds
}

// importAlias proposes the alias of an imported Package: its last path segment,
// capitalized.
func importAlias(pkg string) string {
	name := path.Base(pkg)
	return strings.ToUpper(name[:1]) + name[1:]
}

// overviewKind is the Kind of a Package's overview Definition: the vocabulary's
// Concept when it has one, else its first Kind, written with its import alias.
func overviewKind(kinds []vocabularyKind) (model.Identity, string) {
	if len(kinds) == 0 {
		return model.Identity{}, "Kind"
	}
	chosen := kinds[0]
	for _, kind := range kinds {
		if kind.Identity.ID == "Concept" {
			chosen = kind
			break
		}
	}
	return chosen.Identity, importAlias(chosen.Identity.Package) + "." + chosen.Identity.ID
}

// writeAuthoringGuide states how Definitions of language v1 are written, with
// the vocabulary's Kinds, and how meaning is described.
func writeAuthoringGuide(b *strings.Builder, kinds []vocabularyKind) {
	identity, kind := overviewKind(kinds)
	pkg, alias := "<vocabulary-package>", "Vocabulary"
	if identity.Package != "" {
		pkg, alias = identity.Package, importAlias(identity.Package)
	}
	b.WriteString("\nWriting Definitions (language v1)\n")
	b.WriteString("One Definition is one YAML file, .markitect/model/<package-path>/<id>.yaml; the directory is its Package and Package path plus id is its identity. Its fields:\n\n")
	fmt.Fprintf(b, "imports:            # optional: the other Packages whose Definitions this file names\n  - package: %s\n    as: %s\nid: some-id         # unique within the Package\nkind: %s\ntitle: Readable name   # optional\nbody: >-\n  Binding prose. Name other Definitions with links such as [[%s.other-id]] (a\n  Definition of an imported Package), [[local-id]] (the same Package) or\n  [[%s.other-id|display text]].\nspec: {}            # values the Kind allows; the vocabulary Kinds below take none\n\n", pkg, alias, kind, alias, alias)
	b.WriteString("A link names its target; the sentence around it states the relationship. A link target must exist, and a file can only link the Packages it imports (its own Package needs no import). A body link supplies no authority, inheritance or order.\n")
	b.WriteString("Kinds of the imported vocabulary:\n")
	if len(kinds) == 0 {
		b.WriteString("- none\n")
	}
	for _, k := range kinds {
		fmt.Fprintf(b, "- %s/%s (use as %s.%s): %s", k.Identity.Package, k.Identity.ID, importAlias(k.Identity.Package), k.Identity.ID, k.Body)
		for _, field := range k.Spec {
			if value := valueText(field.Value); value != "[]" {
				fmt.Fprintf(b, " [%s: %s]", field.Name, value)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("\nDescribing meaning\n")
	b.WriteString("- Describe what things are and how they relate, not field lists: who belongs to what, what requires what, which rule applies to which subject under which conditions, what a refusal changes. Say what is permitted, not only what is done.\n")
	b.WriteString("- Check words such as always, never, all, complete and indefinite: they strengthen a rule. A rule must not gain scope just because another Definition links to it.\n")
	b.WriteString("- Give something its own Definition when its identity, responsibility, repeated references or independently changing contract justify it; keep what changes for the same reasons together. State each fact once, at its owner, and link to it.\n")
	b.WriteString("- Observed behavior is evidence, not intent. Keep incidental implementation choices out of the model.\n")
}

// directorySummary lists the existing artifacts by directory, to depth three,
// as a compact overview for the structure phase.
func directorySummary(paths []snapshot.Path, limit int) ([]string, bool) {
	counts := make(map[string]int)
	for _, p := range paths {
		segments := strings.Split(p.String(), "/")
		dir := "."
		if len(segments) > 1 {
			dir = strings.Join(segments[:min(len(segments)-1, 3)], "/")
		}
		counts[dir]++
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	slices.Sort(names)
	var lines []string
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("%s: %d file(s)", name, counts[name]))
	}
	if len(lines) > limit {
		return lines[:limit], true
	}
	return lines, false
}

func writeProjectChecks(b *strings.Builder, checks []briefCheck) {
	if len(checks) == 0 {
		return
	}
	b.WriteString("\nThe project's declared checks of its artifacts (evidence of how the project verifies its realization; Extract does not run them)\n")
	for _, check := range checks {
		fmt.Fprintf(b, "- %s: %s (directory %s)\n", check.Name, check.Command, check.Directory)
	}
}

// structureBriefInput is everything Root's Brief of the structure phase states.
type structureBriefInput struct {
	Roots        []string
	Exceptions   []realization.LayoutException
	Dependencies []string
	Artifacts    []snapshot.Path
	Kinds        []vocabularyKind
	Settings     organization.ManagerSettings
	Checks       []briefCheck
}

func renderStructureBrief(in structureBriefInput) string {
	var b strings.Builder
	b.WriteString("You are the Manager of Root in the structure phase of an Extract.\n\n")
	b.WriteString("The existing artifacts (read-only evidence)\n")
	underRoots := 0
	for _, p := range in.Artifacts {
		if slices.ContainsFunc(in.Roots, func(root string) bool { return p.String() == root || strings.HasPrefix(p.String(), root+"/") }) {
			underRoots++
		}
	}
	fmt.Fprintf(&b, "Artifact roots: %s. They hold %d artifact file(s); %d more lie outside the roots. Overview by directory:\n", strings.Join(in.Roots, ", "), underRoots, len(in.Artifacts)-underRoots)
	lines, more := directorySummary(in.Artifacts, 60)
	for _, line := range lines {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	if more {
		b.WriteString("- and more directories: list them in /work\n")
	}
	b.WriteString("\nWhat you write\n")
	b.WriteString("The Definitions of the proposed Packages under .markitect/model and layout.exceptions of .markitect/execution.yaml. Everything else stays as it is: the model manifest and lock, the rest of the execution document, ignore.yaml, the artifacts and the imported dependency models")
	if len(in.Dependencies) > 0 {
		b.WriteString(" under " + strings.Join(in.Dependencies, ", "))
	}
	b.WriteString(".\n")
	b.WriteString("\nThe layout\n")
	fmt.Fprintf(&b, "Ownership of artifacts is the persistent layout of .markitect/execution.yaml: the artifact roots (%s) mirror the model hierarchy, so a file in %s/lending/reminders/ belongs to the Package lending/reminders (the deepest Package wins); the files directly in a root and every file outside the roots belong to Root. Every file in a subdirectory of a root needs a Package directory or an exception that covers it. A Package whose artifacts live in a directory of another name gets an exception, for example:\n\nlayout:\n  roots: [%s]\n  exceptions:\n    - {path: %s/some/directory, owner: some/package, reason: The artifacts there realize that Package.}\n\n", strings.Join(in.Roots, ", "), in.Roots[0], strings.Join(in.Roots, ", "), in.Roots[0])
	if len(in.Exceptions) == 0 {
		b.WriteString("There are no exceptions yet. Keep the roots as they are; add only exceptions (a directory or, with kind: file, a single file owned by Root).\n")
	} else {
		b.WriteString("Existing exceptions (keep them unless a better proposal needs a change):\n")
		for _, e := range in.Exceptions {
			fmt.Fprintf(&b, "- %s -> %s (%s)\n", e.Path, ownerLabel(e.Owner), e.Reason)
		}
	}
	_, kind := overviewKind(in.Kinds)
	b.WriteString("\nThe overview Definition of each Package\n")
	fmt.Fprintf(&b, "Each proposed Package, including an intermediate one, has one overview Definition of kind %s in .markitect/model/<package-path>/<id>.yaml; the Package's id of the overview may be its own name. It describes the responsibility and its relationships to neighbouring Packages (links). Root has no Definition of its own: the model root is not a Package.\n", kind)
	writeAuthoringGuide(&b, in.Kinds)
	writeProjectChecks(&b, in.Checks)
	writeInstructions(&b, "Execute", in.Settings.Execute)
	writeInstructions(&b, "Verify", in.Settings.Verify)
	return b.String()
}

func ownerLabel(o realization.Owner) string {
	if name, ok := o.Package(); ok {
		return name
	}
	return "Root"
}

// modelBriefInput is everything one Manager's Brief of the model phase states.
type modelBriefInput struct {
	Manager   organization.Manager
	Selection organization.ContextSelection
	Settings  organization.ManagerSettings
	Areas     []string
	Files     []string
	Shard     string
	Children  []briefChild
	Kinds     []vocabularyKind
	Checks    []briefCheck
}

const maxListedArtifacts = 300

func renderModelBrief(in modelBriefInput) string {
	var b strings.Builder
	who := "Package " + in.Manager.Package
	if in.Manager.Package == organization.Root {
		who = "Root"
	}
	fmt.Fprintf(&b, "You are the Manager of %s in the model phase of an Extract.\nRole: %s.\n\n", who, managerRole(in.Manager, in.Selection))
	b.WriteString("Your artifact areas (read-only evidence; you register them)\n")
	if len(in.Areas) == 0 {
		b.WriteString("- none\n")
	}
	for _, area := range in.Areas {
		fmt.Fprintf(&b, "- %s\n", area)
	}
	if len(in.Files) == 0 {
		b.WriteString("They hold no artifact files at present.\n")
	} else {
		fmt.Fprintf(&b, "They hold these %d artifact file(s), all of which you register in your registry shard:\n", len(in.Files))
		for _, name := range in.Files[:min(len(in.Files), maxListedArtifacts)] {
			fmt.Fprintf(&b, "- %s\n", name)
		}
		if len(in.Files) > maxListedArtifacts {
			fmt.Fprintf(&b, "- and %d more (list them in /work)\n", len(in.Files)-maxListedArtifacts)
		}
	}
	if in.Manager.Package == organization.Root {
		b.WriteString("\nYour model area\nYou have no Package: the model root is not one, and the Managers of the Packages write the model. Your work product is the registration of your areas' artifacts in .markitect/registry/registry.yaml and the integration of the whole model through your children; open questions about the intended meaning go from you to the owner.\n")
	} else {
		fmt.Fprintf(&b, "\nYour model area\nYour Package directory is .markitect/model/%s; its subdirectories are your children's Packages.\n", in.Manager.Package)
	}
	b.WriteString("\nYour Definitions from the structure phase (the provisional description of your responsibility; refine and extend it)\n")
	own := 0
	for _, d := range in.Selection.Definitions() {
		if d.Definition.Identity.Package == in.Manager.Package {
			writeDefinition(&b, d, true, in.Manager, nil)
			own++
		}
	}
	if own == 0 {
		b.WriteString("none\n")
	}
	if len(in.Children) > 0 {
		b.WriteString("\nYour children\nYou integrate their results in your own Package (shared terms, boundaries and cross-area rules) and settle cross-area matters between them. Each child describes its own area.\n")
		for _, child := range in.Children {
			areas := "none"
			if len(child.Areas) > 0 {
				areas = strings.Join(child.Areas, "; ")
			}
			fmt.Fprintf(&b, "- %s\n  artifact areas: %s\n", child.Path, areas)
			for _, definition := range child.Definitions {
				fmt.Fprintf(&b, "  Definition %s\n", definition)
			}
		}
	}
	example := "package/id"
	for _, d := range in.Selection.Definitions() {
		if d.Definition.Identity.Package == in.Manager.Package {
			example = identityText(d.Definition.Identity)
			break
		}
	}
	for _, child := range in.Children {
		if example == "package/id" && len(child.Definitions) > 0 {
			example, _, _ = strings.Cut(child.Definitions[0], ":")
		}
	}
	fmt.Fprintf(&b, "\nRegistry\nYou register every existing artifact of your areas in your registry shard %s, with the Definitions it realizes. %s\n", in.Shard, registryShape(example))
	writeAuthoringGuide(&b, in.Kinds)
	writeProjectChecks(&b, in.Checks)
	writeInstructions(&b, "Execute", in.Settings.Execute)
	writeInstructions(&b, "Verify", in.Settings.Verify)
	return b.String()
}

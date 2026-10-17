package organization

import (
	"cmp"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func directoryStrings(roots []snapshot.DirectoryBoundary) []string {
	result := make([]string, len(roots))
	for i, root := range roots {
		result[i] = root.String()
	}
	return result
}

// treePosition returns the Manager's parent and its direct children in
// topology order.
func (c ContextSelection) treePosition() (string, []string) {
	parent, children := "", []string{}
	for _, slot := range c.topology.managers {
		switch {
		case slot.Package == c.Package():
			parent = slot.Parent
		case slot.Parent == c.Package():
			children = append(children, slot.Package)
		}
	}
	return parent, children
}

// localResponsibility is the complete inventory of a task's own work as the
// Host supplies it; an empty list states that there is none, so that a
// reviewer need not read an empty set from a record reference. OwnArtifacts
// is absent from the expectations review, which sees no files.
type localResponsibility struct {
	LocalDefinitions []string
	OwnArtifacts     *[]string `json:",omitempty"`
	OpenFindings     []string
}

func localResponsibilityOf(c ContextSelection, input RenderInput) localResponsibility {
	r := localResponsibility{LocalDefinitions: []string{}, OpenFindings: append([]string{}, input.Task.Findings.IDs...)}
	slices.Sort(r.OpenFindings)
	for _, entry := range c.Definitions() {
		if slices.ContainsFunc(entry.Reasons, func(reason ContextReason) bool { return reason.Kind == OwnDefinition }) {
			r.LocalDefinitions = append(r.LocalDefinitions, entry.Definition.Identity.Package+"/"+entry.Definition.Identity.ID)
		}
	}
	files := input.OwnArtifacts
	switch {
	case input.Review != nil:
		files = input.Review.Files
	case input.View == ReviewerExpectations:
		return r
	}
	paths := []string{}
	for _, artifact := range files {
		paths = append(paths, artifact.Path.String())
	}
	slices.Sort(paths)
	r.OwnArtifacts = &paths
	return r
}

func renderDirectory(w *contextWriter, c ContextSelection, input RenderInput) {
	purposes := make(map[string]*HostText)
	for _, fact := range input.Packages {
		purposes[fact.Package] = fact.Purpose
	}
	children := make(map[string]ChildFact)
	for _, child := range input.Children {
		children[child.Package] = child
	}
	for _, slot := range c.topology.managers {
		w.json("local-package-route-not-authority", struct{ Package, Parent string }{slot.Package, slot.Parent}, hostRender(input.InputBinding))
		if purpose := purposes[slot.Package]; purpose != nil {
			w.host("explicit-package-purpose", *purpose)
		} else {
			w.add("package-purpose-not-supplied", absentPurposeText, productLiteral("absent-purpose", absentPurposeText))
		}
		if slot.Parent == c.Package() {
			w.json("direct-child-identity", slot.Package, hostRender(input.InputBinding))
			child := children[slot.Package]
			if child.State != nil {
				w.host("host-child-task-state", *child.State)
			}
			for _, report := range child.Reports {
				w.host("host-asserted-validated-child-summary-not-file-payload", report)
			}
		}
	}
}

func renderArtifacts(w *contextWriter, artifacts []ArtifactFact, label string) {
	ordered := slices.Clone(artifacts)
	slices.SortFunc(ordered, func(a, b ArtifactFact) int { return cmp.Compare(a.Path.String(), b.Path.String()) })
	for _, artifact := range ordered {
		w.json(label+"-artifact-assertion-not-grant", struct {
			Package, Path, ContentID string
			Content                  ArtifactContent
		}{artifact.Package, artifact.Path.String(), artifact.ContentID.String(), artifact.Content}, hostRender(artifact.Source))
		switch artifact.Content {
		case ArtifactText:
			w.add(label+"-original-artifact-text", *artifact.Text, hostRender(artifact.Source))
		case ArtifactWorkspaceRef:
			w.add(label+"-workspace-reference", workspaceReferenceText, productLiteral("workspace-reference", workspaceReferenceText))
		default:
			w.add(label+"-binary-reference-not-inspected", binaryReferenceText, productLiteral("binary-reference", binaryReferenceText))
		}
	}
}

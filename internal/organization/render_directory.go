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
		if artifact.Content == ArtifactText {
			w.add(label+"-original-artifact-text", *artifact.Text, hostRender(artifact.Source))
		} else {
			w.add(label+"-binary-reference-not-inspected", binaryReferenceText, productLiteral("binary-reference", binaryReferenceText))
		}
	}
}

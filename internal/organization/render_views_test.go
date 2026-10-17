package organization_test

import (
	"bytes"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func reviewFacts(t *testing.T) *organization.ReviewFacts {
	t.Helper()
	return &organization.ReviewFacts{Package: "a", Subject: hostRef("subject", ""), Candidate: hostRef("candidate", ""), Expectations: hostRef("recorded-expectations", ""), Files: []organization.ArtifactFact{renderArtifact(t, "a")}}
}

func TestRenderClosedViewsAndExactProjections(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*organization.RenderInput)
	}{
		{"unknown-view", func(i *organization.RenderInput) { i.View = "writer" }},
		{"task-package", func(i *organization.RenderInput) { i.Task.Package = "b" }},
		{"task-id", func(i *organization.RenderInput) { i.Task.ID = "" }},
		{"outcome", func(i *organization.RenderInput) { i.Task.Outcome.Text = "" }},
		{"basis", func(i *organization.RenderInput) { i.InputBinding = organization.HostSource{} }},
		{"source", func(i *organization.RenderInput) { i.Task.Outcome.Source = organization.HostSource{} }},
		{"cap-zero", func(i *organization.RenderInput) { i.MaxContextBytes.Value = 0 }},
		{"cap-negative", func(i *organization.RenderInput) { i.MaxContextBytes.Value = -1 }},
		{"cap-source", func(i *organization.RenderInput) { i.MaxContextBytes.Source = organization.SettingSource{} }},
		{"unknown-package", func(i *organization.RenderInput) { i.Packages = []organization.PackageFact{{Package: "foreign"}} }},
		{"duplicate-package", func(i *organization.RenderInput) {
			i.Packages = []organization.PackageFact{{Package: "a"}, {Package: "a"}}
		}},
		{"unknown-child", func(i *organization.RenderInput) { i.Children = []organization.ChildFact{{Package: "a/missing"}} }},
		{"sibling-child", func(i *organization.RenderInput) { i.Children = []organization.ChildFact{{Package: "b"}} }},
		{"self-child", func(i *organization.RenderInput) { i.Children = []organization.ChildFact{{Package: "a"}} }},
		{"duplicate-child", func(i *organization.RenderInput) {
			i.Children = []organization.ChildFact{{Package: "a/child"}, {Package: "a/child"}}
		}},
		{"non-root-owner-input", func(i *organization.RenderInput) { value := hostFact("owner", "Owner intent."); i.OwnerInput = &value }},
		{"descendant-own-file", func(i *organization.RenderInput) {
			i.OwnArtifacts = []organization.ArtifactFact{renderArtifact(t, "a/child")}
		}},
		{"manager-review", func(i *organization.RenderInput) { i.Review = reviewFacts(t) }},
		{"phase1-inventory", func(i *organization.RenderInput) {
			i.View = organization.ReviewerExpectations
			a := renderArtifact(t, "a")
			a.Content = organization.ArtifactBinaryRef
			a.Text = nil
			i.OwnArtifacts = []organization.ArtifactFact{a}
		}},
		{"phase1-review", func(i *organization.RenderInput) {
			i.View = organization.ReviewerExpectations
			i.Review = reviewFacts(t)
		}},
		{"phase1-state", func(i *organization.RenderInput) {
			i.View = organization.ReviewerExpectations
			s := hostFact("child-state", "Complete.")
			i.Children = []organization.ChildFact{{Package: "a/child", State: &s}}
		}},
		{"phase1-report", func(i *organization.RenderInput) {
			i.View = organization.ReviewerExpectations
			i.Children = []organization.ChildFact{{Package: "a/child", Reports: []organization.HostText{hostFact("report", "Delivered.")}}}
		}},
		{"phase2-no-subject", func(i *organization.RenderInput) { i.View = organization.ReviewerCandidate }},
		{"phase2-subject", func(i *organization.RenderInput) {
			i.View = organization.ReviewerCandidate
			i.Review = reviewFacts(t)
			i.Review.Subject = organization.HostSource{}
		}},
		{"phase2-candidate", func(i *organization.RenderInput) {
			i.View = organization.ReviewerCandidate
			i.Review = reviewFacts(t)
			i.Review.Candidate = organization.HostSource{}
		}},
		{"phase2-expectations", func(i *organization.RenderInput) {
			i.View = organization.ReviewerCandidate
			i.Review = reviewFacts(t)
			i.Review.Expectations = organization.HostSource{}
		}},
		{"phase2-package", func(i *organization.RenderInput) {
			i.View = organization.ReviewerCandidate
			i.Review = reviewFacts(t)
			i.Review.Package = "b"
		}},
		{"phase2-second-file-channel", func(i *organization.RenderInput) {
			i.View = organization.ReviewerCandidate
			i.Review = reviewFacts(t)
			i.OwnArtifacts = []organization.ArtifactFact{renderArtifact(t, "a")}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection, settings, input := renderFixture(t, "a", organization.ManagerView)
			test.mutate(&input)
			requireRenderRefusal(t, selection, settings, input, "invalid-input")
		})
	}
}

func TestRenderCompleteFindingsInEveryView(t *testing.T) {
	for _, view := range []organization.ContextView{organization.ManagerView, organization.ReviewerExpectations, organization.ReviewerCandidate} {
		t.Run(string(view), func(t *testing.T) {
			selection, settings, input := renderFixture(t, "a", view)
			if view == organization.ReviewerCandidate {
				input.Review = reviewFacts(t)
			}
			before := requireRendered(t, selection, settings, input)
			for _, mutate := range []func(*organization.OpenFindings){
				func(f *organization.OpenFindings) { f.Items = nil },
				func(f *organization.OpenFindings) { f.IDs = nil },
				func(f *organization.OpenFindings) { f.IDs = append(f.IDs, f.IDs[0]) },
				func(f *organization.OpenFindings) { f.Items = append(f.Items, f.Items[0]) },
				func(f *organization.OpenFindings) { f.Items[0].Grounds.Text = "" },
				func(f *organization.OpenFindings) { f.Items[0].Evidence[0] = organization.HostSource{} },
				func(f *organization.OpenFindings) { f.Source = organization.HostSource{} },
			} {
				changed := input
				changed.Task.Findings.IDs = append([]string(nil), input.Task.Findings.IDs...)
				changed.Task.Findings.Items = append([]organization.FindingFact(nil), input.Task.Findings.Items...)
				changed.Task.Findings.Items[0].Evidence = append([]organization.HostSource(nil), input.Task.Findings.Items[0].Evidence...)
				mutate(&changed.Task.Findings)
				requireRenderRefusal(t, selection, settings, changed, "invalid-input")
			}
			input.Task.Findings.IDs = nil
			input.Task.Findings.Items = nil
			empty := requireRendered(t, selection, settings, input)
			if len(sectionTexts(t, empty, "open-finding-grounds")) != 0 || len(sectionTexts(t, empty, "complete-open-finding-set-reference")) != 1 {
				t.Fatal("empty set lost its explicit source")
			}
			if len(sectionTexts(t, before, "open-finding-grounds")) != 1 {
				t.Fatal("input mutation changed prepared findings")
			}
		})
	}
}

func TestRenderRootChildrenAndExplicitPurpose(t *testing.T) {
	selection, settings, input := renderFixture(t, organization.Root, organization.ManagerView)
	owner := hostFact("owner", "Original owner input.\r\n")
	purpose := hostFact("purpose", "Explicit responsibility, not inferred from its name.")
	state := hostFact("child-state", "Awaiting review.")
	input.OwnerInput = &owner
	input.Packages = []organization.PackageFact{{Package: "a", Purpose: &purpose}}
	input.Children = []organization.ChildFact{{Package: "a", State: &state, Reports: []organization.HostText{hostFact("validated-report", "A compact validated delivery summary.")}}}
	result := requireRendered(t, selection, settings, input)
	if result.Template().ID != "markitect/context/root" || len(sectionTexts(t, result, "original-body")) != 0 || !bytes.Equal(sectionTexts(t, result, "root-owner-input")[0], []byte(owner.Text)) {
		t.Fatal("root acquired descendant definitions or lost ordinary root input")
	}
	if len(sectionTexts(t, result, "host-asserted-validated-child-summary-not-file-payload")) != 1 || len(sectionTexts(t, result, "explicit-package-purpose")) != 1 {
		t.Fatal("explicit facts lost")
	}
	input.Children = []organization.ChildFact{{Package: "a/child"}}
	requireRenderRefusal(t, selection, settings, input, "invalid-input")
	input.Children = nil
	input.View = organization.ReviewerExpectations
	requireRenderRefusal(t, selection, settings, input, "invalid-input")
	input.OwnerInput = nil
	requireRendered(t, selection, settings, input)
}

func TestRenderArtifactRepresentations(t *testing.T) {
	selection, settings, input := renderFixture(t, "a", organization.ManagerView)
	artifact := renderArtifact(t, "a")
	for _, mutate := range []func(*organization.ArtifactFact){
		func(a *organization.ArtifactFact) { a.ContentID[0] ^= 1 },
		func(a *organization.ArtifactFact) { a.Text = nil },
		func(a *organization.ArtifactFact) { a.Content = organization.ArtifactBinaryRef },
		func(a *organization.ArtifactFact) { a.Content = "automatic" },
		func(a *organization.ArtifactFact) { a.Path = snapshot.Path{} },
		func(a *organization.ArtifactFact) { a.Package = "unknown" },
	} {
		changed := artifact
		mutate(&changed)
		input.OwnArtifacts = []organization.ArtifactFact{changed}
		requireRenderRefusal(t, selection, settings, input, "invalid-input")
	}
	input.OwnArtifacts = []organization.ArtifactFact{artifact, artifact}
	requireRenderRefusal(t, selection, settings, input, "invalid-input")
	artifact.Content = organization.ArtifactBinaryRef
	artifact.Text = nil
	input.OwnArtifacts = []organization.ArtifactFact{artifact}
	result := requireRendered(t, selection, settings, input)
	if len(sectionTexts(t, result, "own-binary-reference-not-inspected")) != 1 || len(sectionTexts(t, result, "own-original-artifact-text")) != 0 {
		t.Fatal("binary metadata was presented as inspected text")
	}
}

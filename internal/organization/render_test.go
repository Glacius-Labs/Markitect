package organization_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func hostRef(record, field string) organization.HostSource {
	return organization.HostSource{RecordID: record, Field: field, ContentID: settingSource(record, "").ContentID}
}

func hostFact(record, text string) organization.HostText {
	return organization.HostText{Text: text, Source: hostRef(record, "/text")}
}

func renderFixture(t *testing.T, pkg string, view organization.ContextView) (organization.ContextSelection, organization.Settings, organization.RenderInput) {
	t.Helper()
	compiled := compileFixture(t, contextInput(t))
	selection := requireContext(t, compiled, pkg)
	settings := requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	input := organization.RenderInput{
		View: view, InputBinding: hostRef("captured-basis", ""),
		Task: organization.TaskFacts{ID: "task-1", Package: pkg, Outcome: hostFact("task-1", "Realize the selected model responsibility."), Findings: organization.OpenFindings{
			Source: hostRef("open-findings", ""), IDs: []string{"finding-1"}, Items: []organization.FindingFact{{ID: "finding-1", Grounds: hostFact("finding-1", "The original required interface remains an obligation."), Evidence: []organization.HostSource{hostRef("evidence-1", "")}}},
		}},
		MaxContextBytes: setting(int64(1<<20), "/limits/maxContextBytes"),
		RemainingLimits: []organization.HostText{hostFact("limit-observation", "Remaining steps: 12. This observation grants no extension.")},
	}
	return selection, settings, input
}

func renderArtifact(t *testing.T, pkg string) organization.ArtifactFact {
	t.Helper()
	path, err := snapshot.NewPath("real/owned.go")
	if err != nil {
		t.Fatal(err)
	}
	text := "\xef\xbb\xbfpackage owned\r\n"
	entry, err := snapshot.NewEntry(path, snapshot.Regular, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return organization.ArtifactFact{Package: pkg, Path: path, ContentID: entry.ContentID(), Content: organization.ArtifactText, Text: &text, Source: hostRef("captured-artifact", "")}
}

func requireRendered(t *testing.T, selection organization.ContextSelection, settings organization.Settings, input organization.RenderInput) organization.ContextPackage {
	t.Helper()
	result, err := organization.RenderContext(selection, settings, input)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func sectionTexts(t *testing.T, result organization.ContextPackage, name string) [][]byte {
	t.Helper()
	var texts [][]byte
	bytes := result.Bytes()
	for _, section := range result.Sections() {
		if section.Start < 0 || section.Start > section.TextStart || section.TextStart > section.TextEnd || section.TextEnd > section.End || section.End > int64(len(bytes)) {
			t.Fatalf("invalid section range: %+v", section)
		}
		if section.Name == name {
			texts = append(texts, bytes[section.TextStart:section.TextEnd])
		}
	}
	return texts
}

func assertZeroRendered(t *testing.T, result organization.ContextPackage) {
	t.Helper()
	if len(result.Bytes()) != 0 || len(result.Sections()) != 0 || result.Package() != "" || result.View() != "" || result.ContentID() != (snapshot.ContentID{}) || result.Template() != (organization.ProductText{}) {
		t.Fatal("refusal exposed a partial context package")
	}
}

func TestRenderThreeViewsAndOriginalInstructionFrames(t *testing.T) {
	selection, settings, input := renderFixture(t, "a", organization.ManagerView)
	input.OwnArtifacts = []organization.ArtifactFact{renderArtifact(t, "a")}
	result := requireRendered(t, selection, settings, input)
	if result.Package() != "a" || result.View() != organization.ManagerView || result.Template().ID != "markitect/context/manager" || result.Template().Version != "1" {
		t.Fatal("missing product template/view identity")
	}
	if got := sectionTexts(t, result, "original-custom-instruction"); len(got) != 1 || !bytes.Equal(got[0], []byte("\xef\xbb\xbfUse concise reports.\r\nKeep source references.")) {
		t.Fatalf("instruction bytes rewritten: %q", got)
	}
	if got := sectionTexts(t, result, "own-original-artifact-text"); len(got) != 1 || !bytes.Equal(got[0], []byte(*input.OwnArtifacts[0].Text)) {
		t.Fatal("artifact original text rewritten")
	}
	if len(sectionTexts(t, result, "mandate")) != 1 {
		t.Fatal("mandate is not a source-bound section")
	}
	before := result.Bytes()
	changed := result.Bytes()
	changed[0] = '!'
	manifest := result.Sections()
	manifest[0].Sources[0].Kind = organization.HostRenderSource
	if !bytes.Equal(result.Bytes(), before) || reflect.DeepEqual(result.Sections(), manifest) {
		t.Fatal("rendered payload/manifest views alias fixed output")
	}
	input.View = organization.ReviewerExpectations
	forbidden, err := organization.RenderContext(selection, settings, input)
	if err == nil {
		t.Fatal("review expectation phase received artifact inventory/payload")
	}
	assertZeroRendered(t, forbidden)
	input.OwnArtifacts = nil
	phase1 := requireRendered(t, selection, settings, input)
	if len(sectionTexts(t, phase1, "open-finding-grounds")) != 1 || len(sectionTexts(t, phase1, "finding-evidence-reference-not-payload")) != 1 {
		t.Fatal("expectation review lost complete findings or evidence references")
	}
	if strings.Contains(string(phase1.Bytes()), "package owned") {
		t.Fatal("candidate content leaked into expectation phase")
	}
	input.View = organization.ReviewerCandidate
	input.Review = &organization.ReviewFacts{Package: "a", Subject: hostRef("review-subject", ""), Candidate: hostRef("candidate", ""), Expectations: hostRef("recorded-expectations", ""), Files: []organization.ArtifactFact{renderArtifact(t, "a")}, Evidence: []organization.HostText{hostFact("observed-check", "Recorded check result.")}}
	phase2 := requireRendered(t, selection, settings, input)
	if len(sectionTexts(t, phase2, "exact-review-subject-reference")) != 1 || len(sectionTexts(t, phase2, "recorded-expectations-reference")) != 1 || len(sectionTexts(t, phase2, "review-evidence-payload")) != 1 {
		t.Fatal("candidate review lost exact subject/recorded expectations/evidence")
	}
	input.MaxContextBytes.Value = 1
	refused, err := organization.RenderContext(selection, settings, input)
	var refusal *organization.RenderRefusal
	if !errors.As(err, &refusal) || refusal.Code != "context-too-large" || refusal.RequiredBytes <= refusal.LimitBytes || len(refusal.Sections) == 0 {
		t.Fatalf("missing explicit byte-cap diagnostic: %v", err)
	}
	assertZeroRendered(t, refused)
}

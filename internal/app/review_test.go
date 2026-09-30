package app

import (
	"fmt"
	"strings"
	"testing"

	"markitect/internal/core"
	"markitect/internal/format"
	"markitect/internal/render"
	"markitect/internal/source"
)

var testReviewConfig = ReviewConfig{
	Question:      "Does the change follow the applicable guidance?",
	PromptVersion: "review-v1", Model: "model-a", Effort: "high", AllowReuse: true,
}

func TestRecordReviewAndReuseSameFixedSnapshot(t *testing.T) {
	base := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, base, testReviewConfig)
	decision, err := ReuseReview(base, base, record, "v1", "tool-1", testReviewConfig)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != ReviewReusable || decision.Report != "Original review text; no verdict parsing." || len(decision.Reasons) != 0 {
		t.Fatalf("unexpected same-snapshot decision: %#v", decision)
	}
	if !decision.Advisory || decision.TrustNotice != ReviewTrustNotice {
		t.Fatalf("decision omitted advisory trust notice: %#v", decision)
	}
}

func TestReuseAllowsUnrelatedTypedArtifactChange(t *testing.T) {
	base := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, base, testReviewConfig)
	candidateResources := reviewResources(false, false, false)
	candidateResources[3].Spec.Text = "Changed unrelated reference."
	candidate := reviewFixture(t, strings.Repeat("b", 40), false, candidateResources, nil)
	decision, err := ReuseReview(base, candidate, record, "v1", "tool-1", testReviewConfig)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != ReviewReusable {
		t.Fatalf("unrelated typed change required review: %#v", decision)
	}
}

func TestReuseRequiresReviewForChangedEntryContextOrBroadInputs(t *testing.T) {
	tests := []struct {
		name      string
		candidate func(t *testing.T, base *Project) *Project
	}{
		{
			name: "local dependency",
			candidate: func(t *testing.T, _ *Project) *Project {
				resources := reviewResources(false, false, false)
				resources[1].Spec.Text = "Changed guide."
				return reviewFixture(t, strings.Repeat("b", 40), false, resources, nil)
			},
		},
		{
			name: "inherited common rule",
			candidate: func(t *testing.T, _ *Project) *Project {
				resources := reviewResources(true, false, false)
				for _, resource := range resources {
					if resource.Kind == "Rule" {
						resource.Spec.Text = "Changed common rule."
					}
				}
				return reviewFixture(t, strings.Repeat("b", 40), false, resources, nil)
			},
		},
		{
			name: "new unmodelled file",
			candidate: func(t *testing.T, _ *Project) *Project {
				return reviewFixture(t, strings.Repeat("b", 40), false, reviewResources(false, false, false), map[string]string{"tools/checker.py": "changed checker\n"})
			},
		},
		{
			name: "resource inventory deletion",
			candidate: func(t *testing.T, _ *Project) *Project {
				return reviewFixture(t, strings.Repeat("b", 40), false, reviewResources(false, true, false), nil)
			},
		},
		{
			name: "project configuration",
			candidate: func(t *testing.T, _ *Project) *Project {
				return reviewFixture(t, strings.Repeat("b", 40), false, reviewResources(false, false, true), nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(tt.name == "inherited common rule", false, false), nil)
			record := mustRecordReview(t, base, testReviewConfig)
			candidate := tt.candidate(t, base)
			decision, err := ReuseReview(base, candidate, record, "v1", "tool-1", testReviewConfig)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Status != ReviewRequired || len(decision.Reasons) == 0 || decision.Report != "" {
				t.Fatalf("expected a fresh review, got %#v", decision)
			}
		})
	}
}

func TestReuseRequiresReviewForToolConfigOptOutAndProvisionalCandidate(t *testing.T) {
	base := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, base, testReviewConfig)
	provisional := reviewFixture(t, strings.Repeat("b", 40), true, reviewResources(false, false, false), nil)
	for _, tc := range []struct {
		name   string
		after  *Project
		tool   string
		config ReviewConfig
	}{
		{name: "tool digest", after: base, tool: "tool-2", config: testReviewConfig},
		{name: "question config", after: base, tool: "tool-1", config: func() ReviewConfig { c := testReviewConfig; c.Question += " Check strictly."; return c }()},
		{name: "reuse opt out", after: base, tool: "tool-1", config: func() ReviewConfig { c := testReviewConfig; c.AllowReuse = false; return c }()},
		{name: "provisional candidate", after: provisional, tool: "tool-1", config: testReviewConfig},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := ReuseReview(base, tc.after, record, "v1", tc.tool, tc.config)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Status != ReviewRequired {
				t.Fatalf("expected review-required, got %#v", decision)
			}
		})
	}
}

func TestReuseRejectsTamperedRecordAndMissingBaseEvidence(t *testing.T) {
	base := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, base, testReviewConfig)
	tampered := *record
	tampered.Report += " changed"
	if _, err := ReuseReview(base, base, &tampered, "v1", "tool-1", testReviewConfig); err == nil {
		t.Fatal("tampered report was accepted")
	}
	wrongBase := reviewFixture(t, strings.Repeat("c", 40), false, reviewResources(false, false, false), nil)
	if _, err := ReuseReview(wrongBase, base, record, "v1", "tool-1", testReviewConfig); err == nil {
		t.Fatal("mismatched evidence base was accepted")
	}
}

func TestRecordReviewRequiresFixedCleanSnapshotAndCurrentOutputs(t *testing.T) {
	resources := reviewResources(false, false, false)
	for _, tc := range []struct {
		name    string
		project *Project
	}{
		{name: "provisional", project: reviewFixture(t, "", true, resources, nil)},
		{name: "dirty output", project: func() *Project {
			p := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
			for path := range renderOutputs(t, p) {
				p.Snapshot.Files[path] = []byte("stale output")
				break
			}
			return p
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RecordReview(tc.project, "general/Workflow/review", "v1", "tool-1", testReviewConfig, "report"); err == nil {
				t.Fatal("expected review recording to fail")
			}
		})
	}
}

func TestReviewYAMLDecodersRejectUnsafeOrIncompleteDocuments(t *testing.T) {
	configData, err := format.Encode(testReviewConfig)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := DecodeReviewConfig(configData); err != nil || got != testReviewConfig {
		t.Fatalf("config round trip = %#v, %v", got, err)
	}
	for _, data := range [][]byte{
		append(append([]byte(nil), configData...), []byte("question: duplicate\n")...),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\nextra: field\n"),
		[]byte("question: &q x\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n"),
		[]byte("question: !custom x\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n"),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n---\nquestion: q\n"),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\n"),
	} {
		if _, err := DecodeReviewConfig(data); err == nil {
			t.Fatalf("unsafe or incomplete config was accepted: %s", data)
		}
	}

	p := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, p, testReviewConfig)
	recordData, err := format.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeReviewRecord(recordData)
	if err != nil {
		t.Fatalf("record round trip failed: %v\n%s", err, recordData)
	}
	if decoded.Report != record.Report || decoded.ContextDigest != record.ContextDigest || decoded.TrustNotice != ReviewTrustNotice {
		t.Fatalf("record round trip changed evidence: %#v", decoded)
	}
	if _, err := DecodeReviewRecord(append(recordData, []byte("\nextra: value\n")...)); err == nil {
		t.Fatal("record with unknown field was accepted")
	}
}

func reviewResources(withRule, omitUnrelated, configureCodex bool) []*core.Resource {
	policy := &core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "review-test"}, Path: projectPath}
	policy.Spec.Profile = "generic"
	policy.Spec.Areas = []core.Area{{Name: "general", Path: "docs/general"}}
	if withRule {
		policy.Spec.Areas[0].Rules = []core.Ref{{Name: "shared"}}
	}
	if configureCodex {
		policy.Spec.Targets = []string{"codex"}
	}
	entry := &core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "general"}, Path: "docs/general/workflows/review.yaml", Spec: core.Spec{Text: "Review the change.", Uses: []core.Ref{{Kind: "Text", Name: "guide"}}}}
	guide := &core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "guide", Namespace: "general"}, Path: "docs/general/text/guide.yaml", Spec: core.Spec{Text: "Stable guide."}}
	unrelated := &core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: "unrelated", Namespace: "general"}, Path: "docs/general/text/unrelated.yaml", Spec: core.Spec{Text: "Unrelated reference."}}
	resources := []*core.Resource{policy, guide, entry}
	if withRule {
		rule := &core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared", Namespace: "general"}, Path: "docs/general/rules/shared.yaml", Spec: core.Spec{Text: "Shared requirement."}}
		resources = append(resources, rule)
	}
	if !omitUnrelated {
		resources = append(resources, unrelated)
	}
	return resources
}

func reviewFixture(t *testing.T, revision string, provisional bool, resources []*core.Resource, extra map[string]string) *Project {
	t.Helper()
	snapshot := &source.Snapshot{Revision: revision, Provisional: provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		snapshot.Files[resource.Path] = encodeResource(t, *resource)
		snapshot.Modes[resource.Path] = "100644"
	}
	for name, content := range extra {
		snapshot.Files[name] = []byte(content)
		snapshot.Modes[name] = "100644"
	}
	parsed, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("review fixture has diagnostics: %#v", parsed.Diagnostics)
	}
	outputs, err := render.Generate(parsed.Graph)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		snapshot.Files[name] = data
		snapshot.Modes[name] = "100644"
	}
	parsed, err = Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("rendered review fixture has diagnostics: %#v", parsed.Diagnostics)
	}
	return parsed
}

func mustRecordReview(t *testing.T, p *Project, config ReviewConfig) *ReviewRecord {
	t.Helper()
	record, err := RecordReview(p, "general/Workflow/review", "v1", "tool-1", config, "Original review text; no verdict parsing.")
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func renderOutputs(t *testing.T, p *Project) map[string][]byte {
	t.Helper()
	outputs, err := render.Generate(p.Graph)
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}

func TestReviewResultStatusStringsAreStable(t *testing.T) {
	if fmt.Sprint(ReviewReusable) != "reusable" || fmt.Sprint(ReviewRequired) != "review-required" {
		t.Fatal("review status strings changed")
	}
}

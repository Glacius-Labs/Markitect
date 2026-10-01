package app

import (
	"fmt"
	"strings"
	"testing"
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

func TestReviewResultStatusStringsAreStable(t *testing.T) {
	if fmt.Sprint(ReviewReusable) != "reusable" || fmt.Sprint(ReviewRequired) != "review-required" {
		t.Fatal("review status strings changed")
	}
}

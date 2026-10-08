package review

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
	"go.yaml.in/yaml/v3"
)

const (
	repoA         = "repo-a"
	repoB         = "repo-b"
	commitA       = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	commitB       = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	pathA         = "docs/pattern-a.md"
	pathB         = "docs/pattern-b.md"
	candidatePath = "candidates/candidate-a.yaml"
)

func TestValidatePreservesMultiRepositoryEvidenceAndExplicitRequests(t *testing.T) {
	f := newFixture(t)
	report, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, f.decisionBytes)
	if err != nil {
		t.Fatal(err)
	}
	if report.HandoffID != "scope-one" || report.HandoffIdentity != f.handoff.Digest || report.Adopted || !report.UnauthenticatedReviewer {
		t.Fatalf("report identity/authority = %#v", report)
	}
	if len(report.Evidence) != 3 || len(report.Evidence[0].Duplicates) == 0 || len(report.Evidence[1].Conflicts) == 0 {
		t.Fatalf("evidence relationships were not preserved: %#v", report.Evidence)
	}
	if report.Evidence[0].Repository == report.Evidence[1].Repository || len(report.Candidates) != 1 || len(report.Coverage) != len(f.handoff.Coverage) || len(report.PreparationCoverage) != len(f.handoff.Coverage) || len(report.Requests) != 1 {
		t.Fatalf("report dropped repositories/candidates/coverage/request: %#v", report)
	}
	if len(report.Candidates[0].Record.Uncertainty) == 0 || report.Candidates[0].Record.Frequency == nil || report.Decision == nil {
		t.Fatalf("candidate uncertainty/frequency/decision missing: %#v", report.Candidates)
	}
}

func TestContextRunIdentityIsOptionalAndDelegatedToAdoptionContract(t *testing.T) {
	f := newFixture(t)
	without, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(without.OptionalMarkitectRuns) != 0 || without.Decision != nil {
		t.Fatalf("optional inputs unexpectedly required: %#v", without)
	}

	repo := f.handoff.Repositories[0]
	artifacts := []struct{ path, text string }{
		{"context/context-run.yaml", "manifest bytes"},
		{"context/report.yaml", "report bytes"},
		{"context/compiled.json", "compiled context bytes"},
	}
	files := make(map[string][]byte, len(f.blobs)+len(artifacts))
	for key, value := range f.blobs {
		files[key] = value
	}
	markitect := &capture.MarkitectEvidence{SelectionDigest: capture.Hash([]byte("selection")), Version: "0.12.0", BuildDigest: capture.Hash([]byte("tool build"))}
	for i, artifact := range artifacts {
		data := []byte(artifact.text)
		files[capture.BlobKey(repoA, artifact.path)] = data
		repo.Files = append(repo.Files, capture.File{Path: artifact.path, Reason: "explicit ContextRun identity artifact", Mode: snapshot.RegularMode, Digest: capture.Hash(data)})
		item := capture.Artifact{Path: artifact.path, Digest: capture.Hash(data)}
		switch i {
		case 0:
			markitect.Manifest = item
		case 1:
			markitect.Report = item
		case 2:
			markitect.Context = item
		}
	}
	repo.Markitect = markitect
	repo.SnapshotDigest = snapshotDigest(repo.Commit, repo.Files, files, repoA)
	f.handoff.Repositories[0] = repo
	capture.Seal(&f.handoff)
	f.blobs = files
	f.handoffBytes = mustEncode(t, f.handoff)
	f.refreshDecision()
	with, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, f.decisionBytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(with.OptionalMarkitectRuns) != 1 || with.OptionalMarkitectRuns[0].Repository != repoA {
		t.Fatalf("optional Markitect identity was not retained: %#v", with.OptionalMarkitectRuns)
	}
}

func TestValidateRejectsDanglingOrWrongStanceEvidenceReferences(t *testing.T) {
	for name, mutate := range map[string]func(*testFixture){
		"unknown duplicate": func(f *testFixture) { f.queue.Evidence[0].Duplicates = []string{"missing-evidence"}; f.refreshQueue() },
		"wrong stance":      func(f *testFixture) { f.candidate.Support = []string{"evidence-counter"}; f.refreshCandidate() },
		"duplicate candidate ref": func(f *testFixture) {
			f.queue.Evidence = append(f.queue.Evidence, f.queue.Evidence[0])
			f.refreshQueue()
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			mutate(f)
			if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
				t.Fatal("invalid reference accepted")
			}
		})
	}
}

func TestValidateRejectsMissingCoveragePrivacyAndInvalidRanges(t *testing.T) {
	for name, mutate := range map[string]func(*testFixture){
		"dropped handoff coverage": func(f *testFixture) { f.queue.Coverage = nil; f.refreshQueue() },
		"changed coverage question": func(f *testFixture) {
			f.queue.Coverage[0].Question = "Substituted question"
			f.refreshQueue()
		},
		"changed coverage repository": func(f *testFixture) {
			f.queue.Coverage[0].Repository = repoB
			f.refreshQueue()
		},
		"excerpt forbidden": func(f *testFixture) {
			f.handoff.Privacy.AllowExcerpts = false
			capture.Seal(&f.handoff)
			f.handoffBytes = mustEncode(t, f.handoff)
		},
		"line range outside source": func(f *testFixture) {
			start, end := 1, 3
			f.queue.Evidence[0].StartLine = &start
			f.queue.Evidence[0].EndLine = &end
			f.refreshQueue()
		},
		"partial line range": func(f *testFixture) { f.queue.Evidence[0].EndLine = nil; f.refreshQueue() },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			mutate(f)
			if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
				t.Fatal("invalid coverage/privacy/range accepted")
			}
		})
	}
}

func TestCoverageInterpretationCanAdvanceWithoutChangingQuestionIdentity(t *testing.T) {
	f := newFixture(t)
	f.handoff.Coverage[0].State = "uninspected"
	f.handoff.Coverage[0].Reason = "not yet reviewed"
	capture.Seal(&f.handoff)
	f.handoffBytes = mustEncode(t, f.handoff)
	f.queue.Coverage[0].State = "examined"
	f.queue.Coverage[0].Reason = "reviewed in the selected bytes"
	f.refreshQueue()
	report, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil)
	if err != nil {
		t.Fatalf("updated interpretive coverage rejected: %v", err)
	}
	if report.PreparationCoverage[0].State != "uninspected" || report.Coverage[0].State != "examined" {
		t.Fatalf("report did not preserve preparation and current coverage states: %#v", report)
	}
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, f.decisionBytes); err == nil {
		t.Fatal("decision bound to prior queue bytes did not become stale")
	}
}

func TestEvidenceRequestDoesNotGrantReadAuthority(t *testing.T) {
	f := newFixture(t)
	f.queue.Requests[0].Path = "not-selected/new-scope.md"
	f.refreshQueue()
	report, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Requests[0].Path != "not-selected/new-scope.md" {
		t.Fatalf("request was not preserved: %#v", report.Requests)
	}
}

func TestDecisionFreshnessUsesExactRawBytes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testFixture)
	}{
		{"candidate whitespace", func(f *testFixture) { f.candidateBytes[candidatePath] = append(f.candidateBytes[candidatePath], '\n') }},
		{"queue whitespace", func(f *testFixture) { f.queueBytes = append(f.queueBytes, '\n') }},
		{"handoff whitespace", func(f *testFixture) { f.handoffBytes = append(f.handoffBytes, '\n') }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			test.mutate(f)
			if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, f.decisionBytes); err == nil {
				t.Fatal("changed exact bytes did not stale decision/candidate reference")
			}
		})
	}
}

func TestDecisionDateRequiresValidCalendarDate(t *testing.T) {
	f := newFixture(t)
	var decision Decision
	if err := capture.Decode(f.decisionBytes, &decision); err != nil {
		t.Fatal(err)
	}
	decision.Date = "2026-02-30"
	f.decisionBytes = mustEncode(t, decision)
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, f.decisionBytes); err == nil {
		t.Fatal("invalid calendar date accepted")
	}
}

func TestCandidateRequiresExplicitListsSupportAndCounterexampleAbsence(t *testing.T) {
	f := newFixture(t)
	var document yaml.Node
	if err := yaml.Unmarshal(f.candidateBytes[candidatePath], &document); err != nil {
		t.Fatal(err)
	}
	fields := document.Content[0]
	for i := 0; i+1 < len(fields.Content); i += 2 {
		if fields.Content[i].Value == "conditions" {
			fields.Content = append(fields.Content[:i], fields.Content[i+2:]...)
			break
		}
	}
	withoutConditions, err := yaml.Marshal(&document)
	if err != nil {
		t.Fatal(err)
	}
	f.candidateBytes[candidatePath] = withoutConditions
	f.queue.Candidates[0].Digest = capture.Hash(withoutConditions)
	f.refreshQueue()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("omitted conditions list accepted")
	}
	f.candidate.Counterexamples = []string{}
	f.candidate.Uncertainty = []string{}
	f.candidate.Alternatives = []string{}
	f.candidate.Questions = []string{}
	f.candidate.Conditions = []string{}
	f.refreshCandidate()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("counterexample absence without explicit uncertainty accepted")
	}
	f.candidate.Counterexamples = []string{}
	f.candidate.Support = []string{}
	f.candidate.Uncertainty = []string{"No contrary evidence was found in the selected sample."}
	f.refreshCandidate()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("empty support and uncertainty accepted")
	}
	f.candidate.Support = []string{"evidence-support"}
	f.refreshCandidate()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err != nil {
		t.Fatalf("explicit empty alternatives/questions and uncertainty statement should be shape-valid: %v", err)
	}
}

func TestFrequencyClaimMustBindValidSelectedRepositoryRevision(t *testing.T) {
	f := newFixture(t)
	f.candidate.Frequency.Numerator = 4
	f.refreshCandidate()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("numerator greater than denominator accepted")
	}
	f.candidate.Frequency.Numerator = 1
	f.candidate.Frequency.Commit = commitB
	f.refreshCandidate()
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("frequency claim with wrong repository revision accepted")
	}
}

func TestUnexpectedBlobsAndCandidateFilesAreRejected(t *testing.T) {
	f := newFixture(t)
	withExtraBlob := map[string][]byte{}
	for key, data := range f.blobs {
		withExtraBlob[key] = data
	}
	withExtraBlob[capture.BlobKey(repoA, "not-selected.md")] = []byte("not selected")
	if _, err := Validate(f.handoffBytes, withExtraBlob, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("unselected source blob accepted")
	}
	f.candidateBytes["candidates/unreferenced.yaml"] = []byte("unexpected")
	if _, err := Validate(f.handoffBytes, f.blobs, f.queueBytes, f.candidateBytes, nil); err == nil {
		t.Fatal("unreferenced candidate file accepted")
	}
}

type testFixture struct {
	handoff        capture.Handoff
	handoffBytes   []byte
	blobs          map[string][]byte
	queue          Queue
	queueBytes     []byte
	candidate      Candidate
	candidateBytes map[string][]byte
	decisionBytes  []byte
}

func newFixture(t *testing.T) *testFixture {
	t.Helper()
	a := []byte("A selected observation line.\nSecond line.\n")
	b := []byte("A contrary observation line.\n")
	blobs := map[string][]byte{capture.BlobKey(repoA, pathA): a, capture.BlobKey(repoB, pathB): b}
	repoOne := makeRepository(repoA, commitA, pathA, a)
	repoTwo := makeRepository(repoB, commitB, pathB, b)
	handoff := capture.Handoff{
		APIVersion: capture.HandoffVersion, ID: "scope-one", Purpose: "review a bounded convention",
		Review: "ticket-123", Privacy: capture.Privacy{Constraints: "no personal data", AllowExcerpts: true},
		Retention: "delete after review", Repositories: []capture.Repository{repoOne, repoTwo},
		Coverage: []capture.Coverage{
			{ID: "coverage-a", Repository: repoA, Question: "Was the selected pattern used?", State: "examined", Reason: "selected bounded source"},
			{ID: "coverage-b", Repository: repoB, Question: "Was contrary evidence found?", State: "no-evidence-found", Reason: "bounded selection had no additional contrary record"},
		},
	}
	capture.Seal(&handoff)
	evidence := []Evidence{
		{ID: "evidence-support", Repository: repoA, Path: pathA, SourceDigest: capture.Hash(a), StartLine: intPtr(1), EndLine: intPtr(1), Stance: "supports", Observation: "The selected source has the described declaration.", Excerpt: "A selected observation", Duplicates: []string{"evidence-qualifies"}, Conflicts: []string{"evidence-counter"}},
		{ID: "evidence-counter", Repository: repoB, Path: pathB, SourceDigest: capture.Hash(b), StartLine: intPtr(1), EndLine: intPtr(1), Stance: "counterexample", Observation: "A separate repository has contrary structure.", Excerpt: "A contrary observation", Conflicts: []string{"evidence-support"}},
		{ID: "evidence-qualifies", Repository: repoA, Path: pathA, SourceDigest: capture.Hash(a), Stance: "qualifies", Observation: "This observation is limited to the selected sample.", Duplicates: []string{"evidence-support"}},
	}
	candidate := Candidate{
		APIVersion: CandidateVersion, StableID: "candidate-one", ProposedRule: "Keep new feature behavior near its owner.",
		Scope: "new feature work in the sampled projects", Conditions: []string{"only where the owner explicitly selects this convention"},
		Classification: "project-specific", Support: []string{"evidence-support"}, Counterexamples: []string{"evidence-counter"},
		Qualifies: []string{"evidence-qualifies"}, Confidence: "low", ConfidenceBasis: "One selected supporting example and one counterexample.",
		Alternatives: []string{"This may be a local exception."}, Uncertainty: []string{"Intent is not established by the source pattern."},
		Questions: []string{"Does the owner intend this for future work?"},
		Frequency: &FrequencyClaim{Numerator: 1, Denominator: 2, SelectionRule: "one selected source per repository", Repository: repoA, Commit: commitA, Period: "2024-2026"},
	}
	f := &testFixture{handoff: handoff, handoffBytes: mustEncode(t, handoff), blobs: blobs, queue: Queue{
		APIVersion: QueueVersion, Evidence: evidence,
		Candidates: []CandidateReference{{StableID: candidate.StableID, Path: candidatePath}},
		Coverage:   append([]capture.Coverage(nil), handoff.Coverage...),
		Requests:   []EvidenceRequest{{Repository: repoA, Path: "future/unselected.md", Reason: "a recent example would help", Insufficiency: "current selection does not cover that period"}},
	}, candidate: candidate, candidateBytes: map[string][]byte{}}
	f.refreshCandidate()
	f.refreshQueue()
	f.refreshDecision()
	return f
}

func (f *testFixture) refreshCandidate() {
	data, err := capture.Encode(f.candidate)
	if err != nil {
		panic(err)
	}
	f.candidateBytes[candidatePath] = data
	f.queue.Candidates[0].Digest = capture.Hash(data)
	f.refreshQueue()
}
func (f *testFixture) refreshQueue() {
	data, err := capture.Encode(f.queue)
	if err != nil {
		panic(err)
	}
	f.queueBytes = data
}
func (f *testFixture) refreshDecision() {
	f.refreshQueue()
	d := Decision{APIVersion: DecisionVersion, ID: "decision-one", Reviewer: "reviewer-claimed", Date: "2026-10-04", Rationale: "Review exact candidate bytes.", Scope: f.candidate.Scope, Status: "defer", CandidateID: f.candidate.StableID,
		CandidateDigest: capture.Hash(f.candidateBytes[candidatePath]), QueueDigest: capture.Hash(f.queueBytes), HandoffDigest: capture.Hash(f.handoffBytes), HandoffIdentity: f.handoff.Digest}
	encoded, err := capture.Encode(d)
	if err != nil {
		panic(err)
	}
	f.decisionBytes = encoded
}
func makeRepository(id, commit, path string, data []byte) capture.Repository {
	identity := capture.RepositoryIdentity{Root: "/owner-selected/" + id, GitDir: "/owner-selected/" + id + "/.git", CommonDir: "/owner-selected/" + id + "/.git", ObjectFormat: "sha1"}
	identity.Digest = capture.ValueDigest(identity)
	files := []capture.File{{Path: path, Reason: "owner selected for bounded evidence", Mode: snapshot.RegularMode, Digest: capture.Hash(data)}}
	return capture.Repository{ID: id, Identity: identity, Commit: commit, SnapshotDigest: snapshotDigest(commit, files, map[string][]byte{capture.BlobKey(id, path): data}, id), Files: files}
}
func snapshotDigest(commit string, files []capture.File, blobs map[string][]byte, repository string) string {
	s := &snapshot.Snapshot{ID: commit, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, file := range files {
		s.Files[file.Path] = blobs[capture.BlobKey(repository, file.Path)]
		s.Modes[file.Path] = file.Mode
	}
	return s.Digest()
}
func mustEncode(t *testing.T, value any) []byte {
	t.Helper()
	data, err := capture.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func intPtr(value int) *int { return &value }

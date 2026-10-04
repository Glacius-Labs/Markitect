package review

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
	"go.yaml.in/yaml/v3"
)

const ReportVersion = "markitect.example.org/copy-me-report/v1alpha1"

// Validate checks only bytes explicitly supplied by the caller. It never
// opens paths, acquires a repository, calls a model, or changes canonical data.
func Validate(handoffBytes []byte, blobs map[string][]byte, queueBytes []byte, candidateBytes map[string][]byte, decisionBytes []byte) (Report, error) {
	var handoff capture.Handoff
	if err := capture.Decode(handoffBytes, &handoff); err != nil {
		return Report{}, fmt.Errorf("decode adoption handoff: %w", err)
	}
	if err := capture.ValidateHandoff(handoff, blobs); err != nil {
		return Report{}, fmt.Errorf("validate adoption handoff: %w", err)
	}
	var queue Queue
	if err := capture.Decode(queueBytes, &queue); err != nil {
		return Report{}, fmt.Errorf("decode Copy Me queue: %w", err)
	}
	if queue.APIVersion != QueueVersion {
		return Report{}, fmt.Errorf("unsupported Copy Me queue version %q", queue.APIVersion)
	}
	if len(candidateBytes) > 0 && len(queue.Candidates) == 0 {
		return Report{}, fmt.Errorf("candidate files supplied for a queue with no candidates")
	}
	if len(candidateBytes) != len(queue.Candidates) {
		return Report{}, fmt.Errorf("queue candidate references must identify every supplied candidate file exactly once")
	}
	repositories := make(map[string]capture.Repository, len(handoff.Repositories))
	selected := make(map[string]capture.File)
	blobValues := make(map[string][]byte, len(blobs))
	for key, value := range blobs {
		blobValues[key] = value
	}
	for _, repo := range handoff.Repositories {
		repositories[repo.ID] = repo
		for _, file := range repo.Files {
			selected[capture.BlobKey(repo.ID, file.Path)] = file
		}
	}
	if !sameCoverageQuestions(handoff.Coverage, queue.Coverage) {
		return Report{}, fmt.Errorf("queue must preserve every handoff coverage ID, repository and question")
	}
	if err := capture.ValidateCoverage(queue.Coverage, repositoryIDs(repositories)); err != nil {
		return Report{}, fmt.Errorf("validate current coverage: %w", err)
	}

	evidenceByID := make(map[string]Evidence, len(queue.Evidence))
	for _, evidence := range queue.Evidence {
		if !capture.ValidID(evidence.ID) || evidenceByID[evidence.ID].ID != "" {
			return Report{}, fmt.Errorf("invalid or duplicate evidence ID %q", evidence.ID)
		}
		key := capture.BlobKey(evidence.Repository, evidence.Path)
		file, selectedPath := selected[key]
		data, hasBytes := blobValues[key]
		if !selectedPath || !hasBytes || evidence.SourceDigest != file.Digest || capture.Hash(data) != evidence.SourceDigest {
			return Report{}, fmt.Errorf("evidence %s does not bind an exact selected source byte path", evidence.ID)
		}
		if err := validateEvidence(evidence, data, handoff.Privacy.AllowExcerpts); err != nil {
			return Report{}, fmt.Errorf("evidence %s: %w", evidence.ID, err)
		}
		evidenceByID[evidence.ID] = evidence
	}
	if err := validateEvidenceLinks(queue.Evidence, evidenceByID); err != nil {
		return Report{}, err
	}
	for i := range queue.Requests {
		request := &queue.Requests[i]
		if _, ok := repositories[request.Repository]; !ok {
			return Report{}, fmt.Errorf("evidence request refers to unknown repository %q", request.Repository)
		}
		if err := capture.ExactPath(request.Path); err != nil {
			return Report{}, fmt.Errorf("evidence request path: %w", err)
		}
		if strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.Insufficiency) == "" {
			return Report{}, fmt.Errorf("evidence request for %s/%s requires reason and insufficiency", request.Repository, request.Path)
		}
	}

	candidateReports := make([]CandidateReport, 0, len(queue.Candidates))
	candidatesByID := make(map[string]CandidateReport, len(queue.Candidates))
	seenCandidatePaths := map[string]bool{}
	var totalCandidateBytes int
	for _, reference := range queue.Candidates {
		if !capture.ValidID(reference.StableID) || !capture.ValidHash(reference.Digest) {
			return Report{}, fmt.Errorf("invalid candidate reference %q", reference.StableID)
		}
		if err := capture.ExactPath(reference.Path); err != nil {
			return Report{}, fmt.Errorf("candidate %s path: %w", reference.StableID, err)
		}
		if _, duplicate := candidatesByID[reference.StableID]; duplicate || seenCandidatePaths[reference.Path] {
			return Report{}, fmt.Errorf("duplicate candidate ID or path %q", reference.StableID)
		}
		seenCandidatePaths[reference.Path] = true
		data, ok := candidateBytes[reference.Path]
		if !ok || len(data) == 0 || len(data) > maxRecordBytes {
			return Report{}, fmt.Errorf("candidate %s file is absent or exceeds the record byte bound", reference.StableID)
		}
		totalCandidateBytes += len(data)
		if totalCandidateBytes > maxRecordBytes {
			return Report{}, fmt.Errorf("candidate files exceed the aggregate %d-byte bound", maxRecordBytes)
		}
		digest := capture.Hash(data)
		if digest != reference.Digest {
			return Report{}, fmt.Errorf("candidate %s raw bytes do not match its queue digest", reference.StableID)
		}
		var candidate Candidate
		if err := capture.Decode(data, &candidate); err != nil {
			return Report{}, fmt.Errorf("decode candidate %s: %w", reference.StableID, err)
		}
		if err := validateCandidateListsPresent(data); err != nil {
			return Report{}, fmt.Errorf("candidate %s: %w", reference.StableID, err)
		}
		if err := validateCandidate(candidate, reference.StableID, evidenceByID, repositories); err != nil {
			return Report{}, fmt.Errorf("candidate %s: %w", reference.StableID, err)
		}
		report := CandidateReport{Record: candidate, Digest: digest}
		candidateReports = append(candidateReports, report)
		candidatesByID[reference.StableID] = report
	}
	candidatePaths := make([]string, 0, len(candidateBytes))
	for path := range candidateBytes {
		candidatePaths = append(candidatePaths, path)
	}
	sort.Strings(candidatePaths)
	for _, path := range candidatePaths {
		if !seenCandidatePaths[path] {
			return Report{}, fmt.Errorf("unreferenced candidate file %q", path)
		}
	}

	var decision *Decision
	decisionDigest := ""
	if decisionBytes != nil {
		if len(decisionBytes) > maxRecordBytes {
			return Report{}, fmt.Errorf("decision exceeds %d-byte record bound", maxRecordBytes)
		}
		var decoded Decision
		if err := capture.Decode(decisionBytes, &decoded); err != nil {
			return Report{}, fmt.Errorf("decode Copy Me decision: %w", err)
		}
		if err := validateDecision(decoded, handoffBytes, handoff, queueBytes, candidatesByID); err != nil {
			return Report{}, err
		}
		decision = &decoded
		decisionDigest = capture.Hash(decisionBytes)
	}

	markitectRuns := make([]OptionalMarkitectIdentity, 0)
	for _, repo := range handoff.Repositories {
		if repo.Markitect != nil {
			markitectRuns = append(markitectRuns, OptionalMarkitectIdentity{Repository: repo.ID, Evidence: *repo.Markitect})
		}
	}
	return Report{
		APIVersion: ReportVersion, HandoffID: handoff.ID, HandoffIdentity: handoff.Digest,
		HandoffByteDigest: capture.Hash(handoffBytes), QueueByteDigest: capture.Hash(queueBytes),
		Evidence: append([]Evidence(nil), queue.Evidence...), Candidates: candidateReports,
		PreparationCoverage: append([]capture.Coverage(nil), handoff.Coverage...),
		Coverage:            append([]capture.Coverage(nil), queue.Coverage...), Requests: append([]EvidenceRequest(nil), queue.Requests...),
		Decision: decision, DecisionByteDigest: decisionDigest, OptionalMarkitectRuns: markitectRuns,
		UnauthenticatedReviewer: true, Adopted: false,
	}, nil
}

func validateCandidateListsPresent(data []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("candidate must be one mapping")
	}
	root := document.Content[0]
	fields := make(map[string]*yaml.Node, len(root.Content)/2)
	for i := 0; i+1 < len(root.Content); i += 2 {
		fields[root.Content[i].Value] = root.Content[i+1]
	}
	for _, name := range []string{"conditions", "support", "counterexamples", "qualifies", "alternatives", "uncertainty", "questions"} {
		value, ok := fields[name]
		if !ok || value.Kind != yaml.SequenceNode {
			return fmt.Errorf("%s must be present as an explicit list", name)
		}
	}
	return nil
}

func validateEvidence(e Evidence, source []byte, allowExcerpts bool) error {
	if !capture.ValidID(e.Repository) || !capture.ValidHash(e.SourceDigest) || strings.TrimSpace(e.Observation) == "" {
		return fmt.Errorf("repository, source digest and observation are required")
	}
	switch e.Stance {
	case "supports", "counterexample", "qualifies":
	default:
		return fmt.Errorf("unsupported stance %q", e.Stance)
	}
	if (e.StartLine == nil) != (e.EndLine == nil) {
		return fmt.Errorf("startLine and endLine must be supplied together")
	}
	if e.StartLine != nil {
		lineCount := 0
		if len(source) > 0 {
			lineCount = bytes.Count(source, []byte{'\n'}) + 1
			if source[len(source)-1] == '\n' {
				lineCount--
			}
		}
		if *e.StartLine < 1 || *e.EndLine < *e.StartLine || *e.EndLine > lineCount {
			return fmt.Errorf("line range %d..%d is outside the selected source", *e.StartLine, *e.EndLine)
		}
	}
	if e.Excerpt != "" {
		if !allowExcerpts {
			return fmt.Errorf("excerpt is forbidden by handoff privacy settings")
		}
		if !bytes.Contains(source, []byte(e.Excerpt)) {
			return fmt.Errorf("excerpt is not an exact substring of selected source bytes")
		}
	}
	return nil
}

func validateEvidenceLinks(items []Evidence, known map[string]Evidence) error {
	for _, item := range items {
		for _, relation := range []struct {
			label string
			links []string
		}{{label: "duplicate", links: item.Duplicates}, {label: "conflict", links: item.Conflicts}} {
			seen := map[string]bool{}
			for _, id := range relation.links {
				if id == item.ID || known[id].ID == "" || seen[id] {
					return fmt.Errorf("evidence %s has invalid %s reference %q", item.ID, relation.label, id)
				}
				seen[id] = true
			}
		}
	}
	return nil
}

func validateCandidate(candidate Candidate, referencedID string, evidence map[string]Evidence, repositories map[string]capture.Repository) error {
	if candidate.APIVersion != CandidateVersion || candidate.StableID != referencedID || !capture.ValidID(candidate.StableID) {
		return fmt.Errorf("version/stableID do not match queue reference")
	}
	if strings.TrimSpace(candidate.ProposedRule) == "" || strings.TrimSpace(candidate.Scope) == "" || strings.TrimSpace(candidate.Confidence) == "" || strings.TrimSpace(candidate.ConfidenceBasis) == "" {
		return fmt.Errorf("proposedRule, scope and confidence with basis are required")
	}
	switch candidate.Classification {
	case "likely intentional", "recurring convention", "project-specific", "legacy", "compromise", "possible violation", "unclear":
	default:
		return fmt.Errorf("unsupported classification %q", candidate.Classification)
	}
	if err := validateEvidenceRefs("support", candidate.Support, "supports", evidence); err != nil {
		return err
	}
	if len(candidate.Support) == 0 {
		return fmt.Errorf("at least one supporting evidence reference is required")
	}
	if err := validateEvidenceRefs("counterexamples", candidate.Counterexamples, "counterexample", evidence); err != nil {
		return err
	}
	if len(candidate.Counterexamples) == 0 && len(candidate.Uncertainty) == 0 {
		return fmt.Errorf("absence of counterexamples must remain explicit in uncertainty")
	}
	if err := validateEvidenceRefs("qualifies", candidate.Qualifies, "qualifies", evidence); err != nil {
		return err
	}
	for _, field := range []struct {
		name   string
		values []string
	}{{"conditions", candidate.Conditions}, {"alternatives", candidate.Alternatives}, {"uncertainty", candidate.Uncertainty}, {"questions", candidate.Questions}} {
		for _, value := range field.values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s cannot contain blank values", field.name)
			}
		}
	}
	if candidate.Frequency != nil {
		f := candidate.Frequency
		repository, ok := repositories[f.Repository]
		if !ok || !capture.ValidCommit(f.Commit) || repository.Commit != f.Commit || f.Denominator <= 0 || f.Numerator < 0 || f.Numerator > f.Denominator || strings.TrimSpace(f.SelectionRule) == "" || strings.TrimSpace(f.Period) == "" {
			return fmt.Errorf("frequency claim requires valid counts, selection rule, period and exact repository revision")
		}
	}
	return nil
}

func validateEvidenceRefs(field string, ids []string, expectedStance string, known map[string]Evidence) error {
	seen := map[string]bool{}
	for _, id := range ids {
		item, ok := known[id]
		if !ok || seen[id] || item.Stance != expectedStance {
			return fmt.Errorf("%s contains unknown, duplicate or wrong-stance evidence %q", field, id)
		}
		seen[id] = true
	}
	return nil
}

func validateDecision(decision Decision, handoffBytes []byte, handoff capture.Handoff, queueBytes []byte, candidates map[string]CandidateReport) error {
	if decision.APIVersion != DecisionVersion || !capture.ValidID(decision.ID) || strings.TrimSpace(decision.Reviewer) == "" || strings.TrimSpace(decision.Rationale) == "" || strings.TrimSpace(decision.Scope) == "" {
		return fmt.Errorf("decision requires version, ID, supplied reviewer/date/rationale and scope")
	}
	if _, err := time.Parse("2006-01-02", decision.Date); err != nil {
		return fmt.Errorf("decision date must be a valid YYYY-MM-DD calendar date: %w", err)
	}
	switch decision.Status {
	case "accept", "reject", "defer", "split", "revise":
	default:
		return fmt.Errorf("unsupported decision status %q", decision.Status)
	}
	candidate, ok := candidates[decision.CandidateID]
	if !ok || decision.CandidateDigest != candidate.Digest || decision.QueueDigest != capture.Hash(queueBytes) || decision.HandoffDigest != capture.Hash(handoffBytes) || decision.HandoffIdentity != handoff.Digest {
		return fmt.Errorf("decision is stale or does not identify one exact candidate, queue and handoff")
	}
	if decision.Scope != candidate.Record.Scope {
		return fmt.Errorf("decision scope must match the candidate scope it reviews")
	}
	seen := map[string]bool{}
	if len(decision.FollowOnIDs) > 0 && decision.Status != "split" && decision.Status != "revise" {
		return fmt.Errorf("follow-on candidate IDs are valid only for split or revise decisions")
	}
	for _, id := range decision.FollowOnIDs {
		if id == decision.CandidateID || candidates[id].Digest == "" || seen[id] {
			return fmt.Errorf("decision has invalid follow-on candidate ID %q", id)
		}
		seen[id] = true
	}
	return nil
}

func sameCoverageQuestions(left, right []capture.Coverage) bool {
	if len(left) != len(right) {
		return false
	}
	type coverageQuestion struct{ repository, question string }
	byID := make(map[string]coverageQuestion, len(left))
	for _, item := range left {
		if _, exists := byID[item.ID]; exists {
			return false
		}
		byID[item.ID] = coverageQuestion{repository: item.Repository, question: item.Question}
	}
	seen := map[string]bool{}
	for _, item := range right {
		if seen[item.ID] {
			return false
		}
		seen[item.ID] = true
		if expected, ok := byID[item.ID]; !ok || expected.repository != item.Repository || expected.question != item.Question {
			return false
		}
	}
	return true
}

func repositoryIDs(repositories map[string]capture.Repository) map[string]bool {
	ids := make(map[string]bool, len(repositories))
	for id := range repositories {
		ids[id] = true
	}
	return ids
}

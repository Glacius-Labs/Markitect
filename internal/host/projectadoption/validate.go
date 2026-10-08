package projectadoption

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
)

func ValidateDistillation(discovery Discovery, report Distillation) error {
	if err := ValidateDiscovery(discovery); err != nil {
		return err
	}
	if report.APIVersion != DistillationVersion || report.DiscoveryDigest != discovery.Digest || !validDigest(report.SchemaDigest) {
		return errors.New("distillation must bind the exact discovery and schema digest")
	}
	switch report.Method {
	case "human-review", "static-tool", "agent-assisted":
	default:
		return fmt.Errorf("unsupported distillation method %q", report.Method)
	}
	if report.Method == "agent-assisted" {
		if strings.TrimSpace(report.RunnerIdentity) == "" || !validDigest(report.RunnerDigest) {
			return errors.New("agent-assisted distillation requires runner identity and configuration digest")
		}
	} else if report.RunnerIdentity != "" || report.RunnerDigest != "" {
		return errors.New("runner identity is valid only for agent-assisted distillation")
	}
	if report.Claims == nil || report.Terms == nil || report.Contradictions == nil || report.Questions == nil || report.Scopes == nil || report.Proposal.Files == nil {
		return errors.New("distillation must explicitly provide claims, terms, contradictions, questions, scopes, and proposal file lists")
	}
	if strings.TrimSpace(report.Proposal.Goal) == "" || len(report.Proposal.Files) == 0 {
		return errors.New("distillation requires an explicit model proposal")
	}
	evidenceByID := make(map[string]Evidence, len(discovery.Evidence))
	for _, evidence := range discovery.Evidence {
		evidenceByID[evidence.ID] = evidence
	}
	scopes := make(map[string]ScopeProposal, len(report.Scopes))
	for _, scope := range report.Scopes {
		if !validID(scope.ID) || scopes[scope.ID].ID != "" || strings.TrimSpace(scope.Name) == "" || scope.ClaimIDs == nil {
			return fmt.Errorf("invalid or duplicate proposed scope %q", scope.ID)
		}
		scopes[scope.ID] = scope
	}
	if len(scopes) == 0 {
		return errors.New("distillation requires at least one proposed adoption scope")
	}
	if err := validateScopeTree(scopes); err != nil {
		return err
	}
	claims := make(map[string]Claim, len(report.Claims))
	for _, claim := range report.Claims {
		if !validID(claim.ID) || claims[claim.ID].ID != "" || strings.TrimSpace(claim.Statement) == "" {
			return fmt.Errorf("invalid or duplicate claim %q", claim.ID)
		}
		if _, ok := scopes[claim.ScopeID]; !ok {
			return fmt.Errorf("claim %q refers to unknown scope %q", claim.ID, claim.ScopeID)
		}
		if claim.Uncertainty == nil {
			return fmt.Errorf("claim %q must explicitly provide its uncertainty list", claim.ID)
		}
		if err := uniqueNonemptyFolded(claim.Uncertainty); err != nil {
			return fmt.Errorf("claim %q uncertainty: %w", claim.ID, err)
		}
		if err := validateClaimEvidence(claim, evidenceByID, discovery.Commit); err != nil {
			return fmt.Errorf("claim %q: %w", claim.ID, err)
		}
		claims[claim.ID] = claim
	}
	if len(claims) == 0 {
		return errors.New("distillation requires at least one grounded claim")
	}
	for _, scope := range scopes {
		if err := validateClaimIDs(scope.ClaimIDs, claims, scope.ID, true); err != nil {
			return fmt.Errorf("scope %q: %w", scope.ID, err)
		}
	}
	terms := make(map[string]bool, len(report.Terms))
	for _, term := range report.Terms {
		if !validID(term.ID) || terms[term.ID] || strings.TrimSpace(term.Text) == "" || strings.TrimSpace(term.Context) == "" || len(term.Occurrences) == 0 || term.Synonyms == nil || term.Ambiguities == nil {
			return fmt.Errorf("invalid or duplicate term candidate %q", term.ID)
		}
		terms[term.ID] = true
		for _, occurrence := range term.Occurrences {
			if err := validateExcerpt(evidenceByID[occurrence.EvidenceID], occurrence.EvidenceID, occurrence.StartLine, occurrence.EndLine, occurrence.Excerpt); err != nil {
				return fmt.Errorf("term %q occurrence: %w", term.ID, err)
			}
			if !strings.Contains(occurrence.Excerpt, term.Text) {
				return fmt.Errorf("term %q occurrence excerpt does not contain the exact term", term.ID)
			}
		}
		if err := uniqueNonemptyFolded(term.Synonyms); err != nil {
			return fmt.Errorf("term %q synonyms: %w", term.ID, err)
		}
		if err := uniqueNonemptyFolded(term.Ambiguities); err != nil {
			return fmt.Errorf("term %q ambiguities: %w", term.ID, err)
		}
	}
	questions := make(map[string]Question, len(report.Questions))
	for _, question := range report.Questions {
		if !validID(question.ID) || questions[question.ID].ID != "" || strings.TrimSpace(question.Prompt) == "" || len(question.Alternatives) < 2 || question.Blocking == nil {
			return fmt.Errorf("invalid or duplicate clarification question %q", question.ID)
		}
		if _, ok := scopes[question.ScopeID]; !ok {
			return fmt.Errorf("question %q refers to unknown scope %q", question.ID, question.ScopeID)
		}
		if err := uniqueNonemptyFolded(question.Alternatives); err != nil {
			return fmt.Errorf("question %q alternatives: %w", question.ID, err)
		}
		if err := validateClaimIDs(question.ClaimIDs, claims, question.ScopeID, true); err != nil {
			return fmt.Errorf("question %q: %w", question.ID, err)
		}
		questions[question.ID] = question
	}
	for _, contradiction := range report.Contradictions {
		if !validID(contradiction.ID) || strings.TrimSpace(contradiction.Description) == "" || len(contradiction.ClaimIDs) < 2 {
			return fmt.Errorf("invalid contradiction %q", contradiction.ID)
		}
		question, ok := questions[contradiction.QuestionID]
		if !ok || question.ScopeID != contradiction.ScopeID {
			return fmt.Errorf("contradiction %q must link to a same-scope clarification question", contradiction.ID)
		}
		if err := validateClaimIDs(contradiction.ClaimIDs, claims, contradiction.ScopeID, true); err != nil {
			return fmt.Errorf("contradiction %q: %w", contradiction.ID, err)
		}
		questionClaims := make(map[string]bool, len(question.ClaimIDs))
		for _, id := range question.ClaimIDs {
			questionClaims[id] = true
		}
		for _, id := range contradiction.ClaimIDs {
			if !questionClaims[id] {
				return fmt.Errorf("contradiction %q clarification question %q does not cover conflicting claim %q", contradiction.ID, question.ID, id)
			}
		}
		kinds := map[string]bool{}
		for _, id := range contradiction.ClaimIDs {
			kinds[claims[id].Kind] = true
		}
		if len(kinds) < 2 {
			return fmt.Errorf("contradiction %q must preserve claims of different kinds", contradiction.ID)
		}
	}
	contradictionIDs := map[string]bool{}
	for _, contradiction := range report.Contradictions {
		if contradictionIDs[contradiction.ID] {
			return fmt.Errorf("duplicate contradiction ID %q", contradiction.ID)
		}
		contradictionIDs[contradiction.ID] = true
	}
	if err := validateProposal(report.Proposal, scopes); err != nil {
		return err
	}
	copy := report
	copy.Digest = ""
	if digestValue(copy) != report.Digest {
		return errors.New("distillation digest mismatch")
	}
	return nil
}

func validateClaimEvidence(claim Claim, evidence map[string]Evidence, discoveryCommit string) error {
	if len(claim.Evidence) == 0 {
		return errors.New("claim has no selected evidence references")
	}
	switch claim.Kind {
	case "observation":
		if claim.Method != "static-source" || claim.Runtime != nil {
			return errors.New("source observation must use static-source and cannot claim runtime metadata")
		}
	case "documented-intent":
		if claim.Method != "documentation" || claim.Runtime != nil {
			return errors.New("documented intent requires documentation method")
		}
	case "runtime-observation":
		return errors.New("runtime evidence must be labelled submitted-runtime-record; execution is not verified")
	case "submitted-runtime-record":
		if claim.Method != "submitted-record" || claim.Runtime == nil {
			return errors.New("submitted runtime record requires submitted-record method")
		}
	case "hypothesis":
		if claim.Method != "synthesis" || claim.Runtime != nil {
			return errors.New("hypothesis requires synthesis method")
		}
	default:
		return fmt.Errorf("unsupported claim kind %q", claim.Kind)
	}
	for _, reference := range claim.Evidence {
		item, ok := evidence[reference.EvidenceID]
		if !ok {
			return fmt.Errorf("reference %q is not in the selected discovery", reference.EvidenceID)
		}
		if err := validateExcerpt(item, reference.EvidenceID, reference.StartLine, reference.EndLine, reference.Excerpt); err != nil {
			return err
		}
		switch claim.Kind {
		case "documented-intent":
			if item.Basis != "documentation" {
				return fmt.Errorf("documented-intent claim cites non-documentation evidence %q", item.ID)
			}
		case "observation":
			if item.Basis != "code" && item.Basis != "configuration" && item.Basis != "test" {
				return fmt.Errorf("static observation cites incompatible evidence basis %q", item.Basis)
			}
		case "submitted-runtime-record":
			if item.Basis != "runtime-record" {
				return fmt.Errorf("runtime observation cites non-runtime evidence %q", item.ID)
			}
		}
	}
	if claim.Kind == "submitted-runtime-record" {
		cited := false
		for _, reference := range claim.Evidence {
			if reference.EvidenceID == claim.Runtime.EvidenceID {
				cited = true
				break
			}
		}
		if !cited {
			return errors.New("claim must cite the submitted runtime record it describes")
		}
		item, ok := evidence[claim.Runtime.EvidenceID]
		if !ok {
			return errors.New("submitted runtime record evidence is not in the fixed discovery")
		}
		if err := validateSubmittedRuntimeRecord(*claim.Runtime, item, evidence, discoveryCommit); err != nil {
			return err
		}
	}
	return nil
}

type submittedRuntimeRecord struct {
	APIVersion           string         `json:"apiVersion"`
	RecordSourceRevision string         `json:"recordSourceRevision"`
	Command              []string       `json:"command"`
	ExitCode             *int           `json:"exitCode"`
	RunnerDigest         string         `json:"runnerDigest"`
	Inputs               []RuntimeInput `json:"inputs"`
}

func validateSubmittedRuntimeRecord(runtime RuntimeObservation, evidence Evidence, selected map[string]Evidence, discoveryCommit string) error {
	if runtime.EvidenceID != evidence.ID || runtime.RecordSourceRevision == "" || runtime.SourceRelation == "" || runtime.Command == nil || runtime.ExitCode == nil || runtime.Inputs == nil {
		return errors.New("submitted runtime record requires explicit revision, relation, argv, exit code, and input list")
	}
	if !validFullCommit(runtime.RecordSourceRevision) || !validDigest(runtime.RunnerDigest) || len(runtime.Command) == 0 || len(runtime.Inputs) == 0 {
		return errors.New("submitted runtime record requires full source revision, runner digest, argv, and inputs")
	}
	record := submittedRuntimeRecord{}
	if err := decodeClosedJSON([]byte(evidence.Content), &record); err != nil {
		return fmt.Errorf("parse strict submitted runtime record: %w", err)
	}
	if record.APIVersion != RuntimeRecordVersion || record.ExitCode == nil || record.Inputs == nil || record.RecordSourceRevision != runtime.RecordSourceRevision || *record.ExitCode != *runtime.ExitCode || record.RunnerDigest != runtime.RunnerDigest || !sameArgv(record.Command, runtime.Command) || !sameRuntimeInputs(record.Inputs, runtime.Inputs) {
		return errors.New("runtime claim fields must exactly match the selected structured record")
	}
	for _, arg := range record.Command {
		if arg == "" || strings.TrimSpace(arg) != arg || strings.ContainsRune(arg, '\x00') {
			return errors.New("submitted runtime record argv must contain literal nonempty arguments")
		}
	}
	relation := "historical"
	if record.RecordSourceRevision == discoveryCommit {
		relation = "same-discovery-commit"
	}
	if runtime.SourceRelation != relation {
		return errors.New("runtime source relation does not match discovery commit; historical evidence cannot be presented as current")
	}
	seenInputs := map[string]bool{}
	currentSourceInput := false
	for _, input := range record.Inputs {
		if err := validateRepoPath(input.Path); err != nil || !validDigest(input.Digest) {
			return fmt.Errorf("invalid submitted runtime input %q", input.Path)
		}
		key := strings.ToLower(input.Path)
		if seenInputs[key] {
			return fmt.Errorf("submitted runtime record repeats input %q", input.Path)
		}
		seenInputs[key] = true
		if record.RecordSourceRevision == discoveryCommit {
			current, ok := selectedEvidenceByPath(selected, input.Path)
			if !ok {
				return fmt.Errorf("same-commit runtime input %q is not in selected discovery", input.Path)
			}
			if current.Digest != input.Digest {
				return fmt.Errorf("same-commit runtime input %q digest differs from selected discovery", input.Path)
			}
			if current.Basis == "code" || current.Basis == "configuration" || current.Basis == "test" {
				currentSourceInput = true
			}
		}
	}
	if record.RecordSourceRevision == discoveryCommit && !currentSourceInput {
		return errors.New("same-commit runtime record must name at least one selected code, configuration, or test input")
	}
	return nil
}

func selectedEvidenceByPath(selected map[string]Evidence, name string) (Evidence, bool) {
	for _, item := range selected {
		if item.Path == name {
			return item, true
		}
	}
	return Evidence{}, false
}

func validFullCommit(value string) bool {
	if len(value) != 40 && len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameArgv(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameRuntimeInputs(left, right []RuntimeInput) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func validateExcerpt(evidence Evidence, evidenceID string, start, end int, excerpt string) error {
	if evidence.ID == "" || start < 1 || end < start || strings.TrimSpace(excerpt) == "" {
		return fmt.Errorf("invalid evidence reference %q or line bounds", evidenceID)
	}
	lines := strings.Split(evidence.Content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" && strings.HasSuffix(evidence.Content, "\n") {
		lines = lines[:len(lines)-1]
	}
	if end > len(lines) {
		return fmt.Errorf("line bounds %d..%d exceed evidence %q", start, end, evidenceID)
	}
	selected := strings.Join(lines[start-1:end], "\n")
	if !strings.Contains(selected, excerpt) {
		return fmt.Errorf("excerpt is not an exact substring of selected evidence %q at lines %d..%d", evidenceID, start, end)
	}
	return nil
}

func validateClaimIDs(ids []string, claims map[string]Claim, scopeID string, unique bool) error {
	if len(ids) == 0 {
		return errors.New("at least one claim reference is required")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		claim, ok := claims[id]
		if !ok || claim.ScopeID != scopeID || (unique && seen[id]) {
			return fmt.Errorf("unknown, cross-scope, or duplicate claim reference %q", id)
		}
		seen[id] = true
	}
	return nil
}

func validateScopeTree(scopes map[string]ScopeProposal) error {
	for _, scope := range scopes {
		if scope.ParentID != "" {
			if scope.ParentID == scope.ID {
				return fmt.Errorf("scope %q cannot parent itself", scope.ID)
			}
			if _, ok := scopes[scope.ParentID]; !ok {
				return fmt.Errorf("scope %q refers to unknown parent %q", scope.ID, scope.ParentID)
			}
		}
		visited := map[string]bool{scope.ID: true}
		parent := scope.ParentID
		for parent != "" {
			if visited[parent] {
				return fmt.Errorf("proposed scope hierarchy contains a cycle at %q", parent)
			}
			visited[parent] = true
			parent = scopes[parent].ParentID
		}
	}
	return nil
}

func validateProposal(proposal ModelProposal, scopes map[string]ScopeProposal) error {
	seen := map[string]bool{}
	for _, file := range proposal.Files {
		if _, ok := scopes[file.ScopeID]; !ok {
			return fmt.Errorf("proposed file %q refers to unknown scope %q", file.Path, file.ScopeID)
		}
		if file.Path == projectwork.ManifestPath || !strings.HasPrefix(file.Path, projectwork.ModelRoot+"/") || path.Clean(file.Path) != file.Path || (!strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml")) {
			return fmt.Errorf("proposal may contain only canonical model files under %s", projectwork.ModelRoot)
		}
		if err := validateRepoPath(file.Path); err != nil {
			return err
		}
		key := strings.ToLower(file.Path)
		if seen[key] {
			return fmt.Errorf("duplicate or case-aliased proposed file %q", file.Path)
		}
		seen[key] = true
		if !utf8.ValidString(file.Content) || strings.TrimSpace(file.Content) == "" {
			return fmt.Errorf("proposed model file %q must contain nonempty UTF-8", file.Path)
		}
	}
	return nil
}

func ValidateResolution(discovery Discovery, report Distillation, resolution Resolution) error {
	if err := ValidateDistillation(discovery, report); err != nil {
		return err
	}
	if resolution.APIVersion != ResolutionVersion || resolution.DiscoveryDigest != discovery.Digest || resolution.DistillationDigest != report.Digest || resolution.ProposalDigest != ProposalDigest(report.Proposal) {
		return errors.New("resolution must bind exact discovery, distillation, and proposal digests")
	}
	if strings.TrimSpace(resolution.TargetBasis) == "" || !validDigest(resolution.SchemaDigest) || resolution.SchemaDigest != report.SchemaDigest || !validDigest(resolution.BuildDigest) {
		return errors.New("resolution must bind target basis, matching schema, and build digest")
	}
	if strings.TrimSpace(resolution.Actor) == "" || strings.TrimSpace(resolution.AuthorityClaim) == "" || strings.TrimSpace(resolution.DecisionReference) == "" {
		return errors.New("resolution requires actor, explicit authority claim, and decision reference")
	}
	if resolution.Authenticated == nil || *resolution.Authenticated {
		return errors.New("local resolution authority claims are unauthenticated; authenticated must be false")
	}
	if resolution.Questions == nil || resolution.Scopes == nil {
		return errors.New("resolution must explicitly provide question and scope lists, using an empty question list when none apply")
	}
	scopeIDs := map[string]ScopeProposal{}
	for _, scope := range report.Scopes {
		scopeIDs[scope.ID] = scope
	}
	resolutionScopes := map[string]ScopeResolution{}
	for _, choice := range resolution.Scopes {
		if _, ok := scopeIDs[choice.ScopeID]; !ok || resolutionScopes[choice.ScopeID].ScopeID != "" || strings.TrimSpace(choice.Reason) == "" {
			return fmt.Errorf("invalid or duplicate scope resolution %q", choice.ScopeID)
		}
		if choice.Status != "adopt" && choice.Status != "defer" {
			return fmt.Errorf("scope %q status must be adopt or defer", choice.ScopeID)
		}
		resolutionScopes[choice.ScopeID] = choice
	}
	if len(resolutionScopes) != len(scopeIDs) {
		return errors.New("resolution must explicitly adopt or defer every proposed scope")
	}
	questions := map[string]Question{}
	for _, question := range report.Questions {
		questions[question.ID] = question
	}
	answers := map[string]QuestionResolution{}
	for _, answer := range resolution.Questions {
		question, ok := questions[answer.QuestionID]
		if !ok || question.ScopeID != answer.ScopeID || answers[answer.QuestionID].QuestionID != "" || strings.TrimSpace(answer.Reason) == "" {
			return fmt.Errorf("invalid or duplicate resolution for question %q", answer.QuestionID)
		}
		switch answer.Disposition {
		case "answer":
			if strings.TrimSpace(answer.Answer) == "" {
				return fmt.Errorf("answered question %q requires an answer", answer.QuestionID)
			}
		case "defer":
			if answer.Answer != "" {
				return fmt.Errorf("deferred question %q cannot include an answer", answer.QuestionID)
			}
		default:
			return fmt.Errorf("question %q disposition must be answer or defer", answer.QuestionID)
		}
		answers[answer.QuestionID] = answer
	}
	for id, question := range questions {
		choice := resolutionScopes[question.ScopeID]
		answer, exists := answers[id]
		if choice.Status == "adopt" && (!exists || answer.Disposition != "answer") {
			return fmt.Errorf("scope %q cannot be adopted while question %q is unanswered or deferred", question.ScopeID, id)
		}
		if !exists && *question.Blocking && choice.Status != "defer" {
			return fmt.Errorf("blocking question %q must be answered or its scope deferred", id)
		}
	}
	for id := range answers {
		if _, ok := questions[id]; !ok {
			return fmt.Errorf("resolution contains unknown question %q", id)
		}
	}
	copy := resolution
	copy.Digest = ""
	if digestValue(copy) != resolution.Digest {
		return errors.New("resolution digest mismatch")
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func uniqueNonemptyFolded(values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		folded := strings.ToLower(strings.TrimSpace(value))
		if folded == "" || seen[folded] {
			return errors.New("entries must be nonempty and unique ignoring case")
		}
		seen[folded] = true
	}
	return nil
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

package government

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// MaterialCandidateInput is the immutable, pre-execution input to a proposed
// candidate. Result bytes, votes and decisions are deliberately absent.
type MaterialCandidateInput struct {
	PriorConstitutionDigest string `json:"priorConstitutionDigest"`
	BaseRevision            string `json:"baseRevision"`
	RepositoryTreeDigest    string `json:"repositoryTreeDigest"`
	ModelDigest             string `json:"modelDigest"`
	PlanDigest              string `json:"planDigest"`
	CheckDefinitionsDigest  string `json:"checkDefinitionsDigest"`
	ToolPinsDigest          string `json:"toolPinsDigest"`
	InventoryDigest         string `json:"inventoryDigest,omitempty"`
}

type MaterialCandidate struct {
	ID    string                 `json:"id"`
	Input MaterialCandidateInput `json:"input"`
}

func NewMaterialCandidate(input MaterialCandidateInput) (MaterialCandidate, error) {
	for _, field := range []struct{ name, digest string }{
		{"prior Constitution", input.PriorConstitutionDigest},
		{"repository tree", input.RepositoryTreeDigest},
		{"model", input.ModelDigest},
		{"plan", input.PlanDigest},
		{"check definitions", input.CheckDefinitionsDigest},
		{"tool pins", input.ToolPinsDigest},
	} {
		if !validDigest(field.digest) {
			return MaterialCandidate{}, fmt.Errorf("%s digest must be sha256", field.name)
		}
	}
	if input.InventoryDigest != "" && !validDigest(input.InventoryDigest) {
		return MaterialCandidate{}, errors.New("inventory digest must be sha256 when supplied")
	}
	if !validFullRevision(input.BaseRevision) {
		return MaterialCandidate{}, errors.New("base revision must be a full Git object ID")
	}
	return MaterialCandidate{ID: bindingHash("material-candidate/v1", input), Input: input}, nil
}

func validateCandidate(candidate MaterialCandidate) error {
	canonical, err := NewMaterialCandidate(candidate.Input)
	if err != nil {
		return fmt.Errorf("candidate input: %w", err)
	}
	if candidate.ID != canonical.ID {
		return errors.New("candidate identity is invalid")
	}
	return nil
}

// Evidence binds immutable result/report digests to one candidate and round.
type Evidence struct {
	ID                  string   `json:"id"`
	MaterialCandidateID string   `json:"materialCandidateId"`
	Round               uint64   `json:"round"`
	ResultDigests       []string `json:"resultDigests"`
	ReportDigests       []string `json:"reportDigests"`
}

func NewEvidence(candidate MaterialCandidate, round uint64, resultDigests, reportDigests []string) (Evidence, error) {
	if err := validateCandidate(candidate); err != nil {
		return Evidence{}, err
	}
	if round == 0 {
		return Evidence{}, errors.New("evidence round must be positive")
	}
	results, err := canonicalDigests(resultDigests)
	if err != nil {
		return Evidence{}, fmt.Errorf("result digests: %w", err)
	}
	reports, err := canonicalDigests(reportDigests)
	if err != nil {
		return Evidence{}, fmt.Errorf("report digests: %w", err)
	}
	if len(results)+len(reports) == 0 {
		return Evidence{}, errors.New("evidence needs at least one immutable result or report digest")
	}
	unsigned := struct {
		Candidate string   `json:"candidate"`
		Round     uint64   `json:"round"`
		Results   []string `json:"results"`
		Reports   []string `json:"reports"`
	}{candidate.ID, round, results, reports}
	return Evidence{ID: bindingHash("evidence/v1", unsigned), MaterialCandidateID: candidate.ID, Round: round, ResultDigests: results, ReportDigests: reports}, nil
}

type VoteOutcome string

const (
	VoteAssent           VoteOutcome = "assent"
	VoteAssentUnaffected VoteOutcome = "assent-unaffected"
	VoteObjection        VoteOutcome = "objection"
	VoteIncomplete       VoteOutcome = "incomplete"
)

// VoteProvenance records configured host slot/run labels. It is not an
// authentication claim and must be populated by a future trusted host.
type VoteProvenance struct {
	RunID  string `json:"runId"`
	SlotID string `json:"slotId"`
}

type RessortVote struct {
	ID                  string                  `json:"id"`
	PriorMandate        core.DefinitionIdentity `json:"priorMandate"`
	MandateDigest       string                  `json:"mandateDigest"`
	Ressort             core.DefinitionIdentity `json:"ressort"`
	MaterialCandidateID string                  `json:"materialCandidateId"`
	EvidenceID          string                  `json:"evidenceId"`
	Round               uint64                  `json:"round"`
	Outcome             VoteOutcome             `json:"outcome"`
	Reason              string                  `json:"reason"`
	Provenance          VoteProvenance          `json:"provenance"`
}

func NewRessortVote(candidate MaterialCandidate, evidence Evidence, priorMandate core.DefinitionIdentity, mandateDigest string, ressort core.DefinitionIdentity, outcome VoteOutcome, reason string, provenance VoteProvenance) (RessortVote, error) {
	if err := validateEvidence(candidate, evidence); err != nil {
		return RessortVote{}, err
	}
	if !validGovernmentIdentity(priorMandate, "Mandate") || !validDigest(mandateDigest) || !validGovernmentIdentity(ressort, "Ressort") {
		return RessortVote{}, errors.New("vote requires exact prior mandate and Ressort identities and mandate digest")
	}
	if outcome != VoteAssent && outcome != VoteAssentUnaffected && outcome != VoteObjection && outcome != VoteIncomplete {
		return RessortVote{}, errors.New("unsupported vote outcome")
	}
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(provenance.RunID) == "" || strings.TrimSpace(provenance.SlotID) == "" {
		return RessortVote{}, errors.New("vote requires a reason and configured run/slot provenance")
	}
	v := RessortVote{PriorMandate: priorMandate, MandateDigest: mandateDigest, Ressort: ressort, MaterialCandidateID: candidate.ID, EvidenceID: evidence.ID, Round: evidence.Round, Outcome: outcome, Reason: reason, Provenance: provenance}
	v.ID = bindingHash("ressort-vote/v1", votePayload(v))
	return v, nil
}

type AcceptanceDecision struct {
	ID                   string   `json:"id"`
	MaterialCandidateID  string   `json:"materialCandidateId"`
	EvidenceID           string   `json:"evidenceId"`
	PriorAuthorityDigest string   `json:"priorAuthorityDigest"`
	Round                uint64   `json:"round"`
	VoteIDs              []string `json:"voteIds"`
}

// CabinetMember is one prior-Constitution-selected Ressort slot. The trusted
// host must build these values from its frozen active configuration.
type CabinetMember struct {
	Ressort       core.DefinitionIdentity `json:"ressort"`
	PriorMandate  core.DefinitionIdentity `json:"priorMandate"`
	MandateDigest string                  `json:"mandateDigest"`
	SlotID        string                  `json:"slotId"`
}

// NewAcceptanceDecision validates an exact, complete, unanimous set from the
// frozen cabinet. It checks record consistency only; it does not authenticate
// vote issuers or grant host promotion authority.
func NewAcceptanceDecision(candidate MaterialCandidate, evidence Evidence, priorAuthorityDigest string, cabinet []CabinetMember, votes []RessortVote) (AcceptanceDecision, error) {
	if err := validateEvidence(candidate, evidence); err != nil {
		return AcceptanceDecision{}, err
	}
	if !validDigest(priorAuthorityDigest) || priorAuthorityDigest != candidate.Input.PriorConstitutionDigest {
		return AcceptanceDecision{}, errors.New("prior authority digest must match the candidate's prior Constitution")
	}
	if len(cabinet) == 0 {
		return AcceptanceDecision{}, errors.New("frozen cabinet cannot be empty")
	}
	expected := map[string]CabinetMember{}
	for _, member := range cabinet {
		if !validGovernmentIdentity(member.Ressort, "Ressort") || !validGovernmentIdentity(member.PriorMandate, "Mandate") || !validDigest(member.MandateDigest) || strings.TrimSpace(member.SlotID) == "" {
			return AcceptanceDecision{}, errors.New("cabinet member requires Government Ressort/Mandate identities, mandate digest and configured slot")
		}
		if _, exists := expected[member.Ressort.Key()]; exists {
			return AcceptanceDecision{}, errors.New("cabinet contains duplicate Ressort identity")
		}
		expected[member.Ressort.Key()] = member
	}
	if len(votes) != len(expected) {
		return AcceptanceDecision{}, errors.New("one vote is required from every frozen cabinet Ressort")
	}
	seenRoles, seenVotes := map[string]bool{}, map[string]bool{}
	ids := make([]string, 0, len(votes))
	for _, v := range votes {
		member, inCabinet := expected[v.Ressort.Key()]
		if !inCabinet {
			return AcceptanceDecision{}, errors.New("vote is from a Ressort outside the frozen cabinet")
		}
		if seenRoles[v.Ressort.Key()] {
			return AcceptanceDecision{}, errors.New("duplicate cabinet role vote")
		}
		seenRoles[v.Ressort.Key()] = true
		if v.Outcome != VoteAssent && v.Outcome != VoteAssentUnaffected {
			return AcceptanceDecision{}, errors.New("objection or incomplete vote prevents acceptance")
		}
		if err := validateVote(v, candidate, evidence, member); err != nil {
			return AcceptanceDecision{}, err
		}
		if seenVotes[v.ID] {
			return AcceptanceDecision{}, errors.New("duplicate vote identity")
		}
		seenVotes[v.ID] = true
		ids = append(ids, v.ID)
	}
	if len(seenRoles) != len(expected) {
		return AcceptanceDecision{}, errors.New("frozen cabinet vote is absent")
	}
	sort.Strings(ids)
	payload := struct {
		Candidate string   `json:"candidate"`
		Evidence  string   `json:"evidence"`
		Authority string   `json:"authority"`
		Round     uint64   `json:"round"`
		Votes     []string `json:"votes"`
	}{candidate.ID, evidence.ID, priorAuthorityDigest, evidence.Round, ids}
	return AcceptanceDecision{ID: bindingHash("acceptance-decision/v1", payload), MaterialCandidateID: candidate.ID, EvidenceID: evidence.ID, PriorAuthorityDigest: priorAuthorityDigest, Round: evidence.Round, VoteIDs: ids}, nil
}

func validateEvidence(candidate MaterialCandidate, evidence Evidence) error {
	if err := validateCandidate(candidate); err != nil {
		return err
	}
	if evidence.MaterialCandidateID != candidate.ID || evidence.Round == 0 {
		return errors.New("evidence is stale or bound to another candidate")
	}
	results, err := canonicalDigests(evidence.ResultDigests)
	if err != nil {
		return err
	}
	reports, err := canonicalDigests(evidence.ReportDigests)
	if err != nil {
		return err
	}
	if len(results)+len(reports) == 0 {
		return errors.New("evidence has no immutable results or reports")
	}
	payload := struct {
		Candidate string   `json:"candidate"`
		Round     uint64   `json:"round"`
		Results   []string `json:"results"`
		Reports   []string `json:"reports"`
	}{candidate.ID, evidence.Round, results, reports}
	if evidence.ID != bindingHash("evidence/v1", payload) {
		return errors.New("evidence identity is invalid")
	}
	return nil
}

func validateVote(v RessortVote, candidate MaterialCandidate, evidence Evidence, member CabinetMember) error {
	if !validGovernmentIdentity(v.PriorMandate, "Mandate") || !validDigest(v.MandateDigest) || !validGovernmentIdentity(v.Ressort, "Ressort") {
		return errors.New("vote requires Government Mandate and Ressort identities and a mandate digest")
	}
	if v.PriorMandate.Key() != member.PriorMandate.Key() || v.MandateDigest != member.MandateDigest || v.Provenance.SlotID != member.SlotID {
		return errors.New("vote mandate or configured slot differs from frozen cabinet member")
	}
	if v.MaterialCandidateID != candidate.ID || v.EvidenceID != evidence.ID || v.Round != evidence.Round {
		return errors.New("vote is stale or bound to different evidence/round")
	}
	if v.Outcome != VoteAssent && v.Outcome != VoteAssentUnaffected && v.Outcome != VoteObjection && v.Outcome != VoteIncomplete {
		return errors.New("unsupported vote outcome")
	}
	if strings.TrimSpace(v.Reason) == "" || strings.TrimSpace(v.Provenance.RunID) == "" || strings.TrimSpace(v.Provenance.SlotID) == "" {
		return errors.New("vote requires a reason and configured run/slot provenance")
	}
	if !validBindingID(v.ID, "ressort-vote/v1") || v.ID != bindingHash("ressort-vote/v1", votePayload(v)) {
		return errors.New("vote identity is invalid")
	}
	return nil
}

func votePayload(v RessortVote) any {
	return struct {
		Mandate       core.DefinitionIdentity `json:"mandate"`
		MandateDigest string                  `json:"mandateDigest"`
		Ressort       core.DefinitionIdentity `json:"ressort"`
		Candidate     string                  `json:"candidate"`
		Evidence      string                  `json:"evidence"`
		Round         uint64                  `json:"round"`
		Outcome       VoteOutcome             `json:"outcome"`
		Reason        string                  `json:"reason"`
		Provenance    VoteProvenance          `json:"provenance"`
	}{v.PriorMandate, v.MandateDigest, v.Ressort, v.MaterialCandidateID, v.EvidenceID, v.Round, v.Outcome, v.Reason, v.Provenance}
}

func validGovernmentIdentity(id core.DefinitionIdentity, kind string) bool {
	return id.APIVersion == APIVersion && id.Kind == kind && strings.TrimSpace(id.Name) != ""
}
func validFullRevision(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil && revision == strings.ToLower(revision)
}
func canonicalDigests(input []string) ([]string, error) {
	out := append([]string(nil), input...)
	for _, d := range out {
		if !validDigest(d) {
			return nil, errors.New("all entries must be sha256 digests")
		}
	}
	sort.Strings(out)
	for i := 1; i < len(out); i++ {
		if out[i] == out[i-1] {
			return nil, errors.New("duplicate digest")
		}
	}
	return out, nil
}
func bindingHash(domain string, value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	_, _ = h.Write([]byte(domain + "\x00"))
	_, _ = h.Write(b)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
func validBindingID(id, domain string) bool {
	return validDigest(id) && strings.HasPrefix(id, "sha256:") && domain != ""
}

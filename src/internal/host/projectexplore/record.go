package projectexplore

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel/projectvalue"
)

var safeID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)

func validateRecord(record Record) error {
	if record.APIVersion != APIVersion || !safeID.MatchString(record.ID) {
		return errors.New("exploration record has an invalid API version or ID")
	}
	if record.Status != StatusActive && record.Status != StatusCompleted {
		return fmt.Errorf("exploration status %q is invalid", record.Status)
	}
	if strings.TrimSpace(record.Request) == "" || len(record.Request) > 64<<10 {
		return errors.New("exploration request must be nonempty and at most 64 KiB")
	}
	if !projectvalue.IsValidDigest(record.CreatedAgainst) {
		return errors.New("exploration record must retain its creation binding digest")
	}
	if record.Scopes == nil || record.Decisions == nil || record.Drafts == nil || record.Acknowledgements == nil || record.Completions == nil {
		return errors.New("exploration record collections must be explicit arrays")
	}
	scopes := make(map[string]Scope, len(record.Scopes))
	for _, scope := range record.Scopes {
		if !safeID.MatchString(scope.ID) || strings.TrimSpace(scope.Name) == "" || strings.TrimSpace(scope.Goal) == "" || scope.Operation == "" || scope.ManagerIDs == nil || !sortedUnique(scope.ManagerIDs, false) {
			return fmt.Errorf("scope %q has invalid identity, goal, operation, or Managers", scope.ID)
		}
		if _, exists := scopes[scope.ID]; exists {
			return fmt.Errorf("duplicate exploration scope %q", scope.ID)
		}
		scopes[scope.ID] = scope
	}
	if len(scopes) != 1 {
		return errors.New("each exploration record must contain exactly one named work-item scope")
	}
	decisions := map[string]bool{}
	for _, decision := range record.Decisions {
		if !safeID.MatchString(decision.ID) || strings.TrimSpace(decision.Question) == "" || !sortedUnique(decision.ScopeIDs, true) {
			return fmt.Errorf("decision %q has invalid identity, question, or scopes", decision.ID)
		}
		if decisions[decision.ID] {
			return fmt.Errorf("duplicate exploration decision %q", decision.ID)
		}
		decisions[decision.ID] = true
		for _, id := range decision.ScopeIDs {
			if _, exists := scopes[id]; !exists {
				return fmt.Errorf("decision %q references unknown scope %q", decision.ID, id)
			}
		}
		switch decision.Status {
		case "open":
			if decision.Answer != "" || decision.Authority != "" || decision.Provenance != "" {
				return fmt.Errorf("open decision %q cannot contain answer authority or provenance", decision.ID)
			}
		case "answered":
			if strings.TrimSpace(decision.Answer) == "" || strings.TrimSpace(decision.Authority) == "" || strings.TrimSpace(decision.Provenance) == "" {
				return fmt.Errorf("answered decision %q requires answer, asserted authority, and provenance", decision.ID)
			}
		case "deferred":
			if strings.TrimSpace(decision.Reason) == "" || strings.TrimSpace(decision.Authority) == "" || strings.TrimSpace(decision.Provenance) == "" {
				return fmt.Errorf("deferred decision %q requires reason, asserted authority, and provenance", decision.ID)
			}
		default:
			return fmt.Errorf("decision %q status must be open, answered, or deferred", decision.ID)
		}
	}
	draftIDs := map[string]bool{}
	for _, draft := range record.Drafts {
		if !safeID.MatchString(draft.ID) || draftIDs[draft.ID] || strings.TrimSpace(draft.Goal) == "" || draft.Files == nil || len(draft.Files) == 0 {
			return fmt.Errorf("draft proposal %q is invalid", draft.ID)
		}
		draftIDs[draft.ID] = true
		seenPaths := map[string]bool{}
		bytesTotal := 0
		for _, file := range draft.Files {
			if !validDraftPath(file.Path) || seenPaths[file.Path] || (file.Delete && file.Content != "") || (!file.Delete && file.Content == "") {
				return fmt.Errorf("draft %q has invalid or duplicate model path %q", draft.ID, file.Path)
			}
			seenPaths[file.Path] = true
			bytesTotal += len(file.Content)
			if bytesTotal > 4<<20 {
				return fmt.Errorf("draft %q exceeds the 4 MiB limit", draft.ID)
			}
		}
	}
	acks := map[string]bool{}
	for _, ack := range record.Acknowledgements {
		if _, exists := scopes[ack.ScopeID]; !exists || !projectvalue.IsValidDigest(ack.BindingDigest) || !projectvalue.IsValidDigest(ack.StructureDigest) || strings.TrimSpace(ack.Actor) == "" || strings.TrimSpace(ack.Authority) == "" || strings.TrimSpace(ack.Provenance) == "" || ack.RecordedAt.IsZero() {
			return fmt.Errorf("structure acknowledgement for scope %q is incomplete", ack.ScopeID)
		}
		key := ack.ScopeID + "\x00" + ack.BindingDigest + "\x00" + ack.StructureDigest
		if acks[key] {
			return fmt.Errorf("duplicate structure acknowledgement for scope %q", ack.ScopeID)
		}
		acks[key] = true
	}
	completedScopes := map[string]bool{}
	for _, receipt := range record.Completions {
		if _, exists := scopes[receipt.ScopeID]; !exists || completedScopes[receipt.ScopeID] {
			return fmt.Errorf("completion references unknown or duplicate scope %q", receipt.ScopeID)
		}
		if err := validateApplyReceipt(receipt); err != nil {
			return err
		}
		completedScopes[receipt.ScopeID] = true
	}
	if record.Status == StatusCompleted && len(completedScopes) != len(scopes) {
		return errors.New("exploration cannot be completed until every scope has a successful Apply receipt")
	}
	if record.Status == StatusActive && len(completedScopes) == len(scopes) {
		return errors.New("all scopes are complete but exploration status is active")
	}
	return nil
}

func validateApplyReceipt(receipt ApplyReceipt) error {
	if receipt.Status != "applied" || !projectvalue.IsValidDigest(receipt.BindingDigest) || !projectvalue.IsValidDigest(receipt.StructureDigest) ||
		receipt.RunID == "" || receipt.PlanID == "" || !projectvalue.IsValidDigest(receipt.PlanDigest) || receipt.CandidateID == "" || !projectvalue.IsValidDigest(receipt.CandidateDigest) ||
		receipt.VerificationID == "" || receipt.VerificationStatus != "passed" || !projectvalue.IsValidDigest(receipt.VerificationDigest) || receipt.ApplyID == "" || !projectvalue.IsValidDigest(receipt.ApplyDigest) || receipt.AppliedAt.IsZero() {
		return errors.New("completion requires a successful Apply receipt with run, plan, candidate, verification, and Apply bindings")
	}
	return nil
}

func validDraftPath(value string) bool {
	return strings.HasPrefix(value, ".markitect/model/") && path.Clean(value) == value && !strings.Contains(value, "\\") && !strings.ContainsRune(value, 0) && !strings.Contains(value, "/../") && !strings.HasSuffix(value, "/..")
}

func sortedUnique(values []string, nonempty bool) bool {
	for i, value := range values {
		if nonempty && strings.TrimSpace(value) == "" || i > 0 && values[i-1] >= value {
			return false
		}
	}
	return true
}

func recordDigest(record Record) (string, error) {
	record.Digest = ""
	return digest(record)
}

func digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func EncodeRecord(record Record) ([]byte, error) {
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	digest, err := recordDigest(record)
	if err != nil {
		return nil, err
	}
	if record.Digest != "" && record.Digest != digest {
		return nil, errors.New("exploration record digest does not match its contents")
	}
	record.Digest = digest
	return json.Marshal(record)
}

func DecodeRecord(data []byte) (Record, error) {
	return decodeRecord(data, true)
}

// DecodeRecordInput decodes a closed create/update input and fills its digest.
// Persisted records must use DecodeRecord, which requires an existing digest.
func DecodeRecordInput(data []byte) (Record, error) {
	return decodeRecord(data, false)
}

func decodeRecord(data []byte, requireDigest bool) (Record, error) {
	var record Record
	if len(data) == 0 || len(data) > 8<<20 {
		return record, errors.New("exploration record must be between 1 byte and 8 MiB")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return record, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, fmt.Errorf("decode exploration record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return record, errors.New("exploration record must contain exactly one JSON value")
	}
	if requireDigest || record.CreatedAgainst != "" {
		if err := validateRecord(record); err != nil {
			return record, err
		}
	} else {
		validation := record
		validation.CreatedAgainst = "sha256:" + strings.Repeat("0", 64)
		if err := validateRecord(validation); err != nil {
			return record, err
		}
	}
	expected, err := recordDigest(record)
	if err != nil {
		return record, err
	}
	if record.Digest != "" && record.Digest != expected {
		return record, errors.New("exploration record digest is missing or invalid")
	}
	if requireDigest && record.Digest == "" {
		return record, errors.New("exploration record digest is missing or invalid")
	}
	record.Digest = expected
	return record, nil
}

// SealRecord validates and computes Digest after a caller has made a
// prospective in-memory update. Persistence still requires Write CAS.
func SealRecord(record *Record) error { return reseal(record) }

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var readValue func() error
	readValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("exploration JSON object contains a non-string key")
				}
				for prior := range seen {
					if strings.EqualFold(prior, key) {
						return fmt.Errorf("exploration JSON contains duplicate or case-alias key %q", key)
					}
				}
				seen[key] = true
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("exploration JSON contains an unexpected delimiter")
		}
	}
	if err := readValue(); err != nil {
		return fmt.Errorf("invalid exploration JSON: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("exploration JSON must contain exactly one value")
	}
	return nil
}

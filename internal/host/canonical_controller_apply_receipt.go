package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

// DecodeCanonicalControllerApply decodes a complete, materialized Apply receipt
// for use as an input to ordinary Verify. The receipt is not an authorization,
// signature, or freshness proof; Verify must still reconstruct current state.
func DecodeCanonicalControllerApply(data []byte) (CanonicalControllerApply, error) {
	var report CanonicalControllerApply
	if len(data) == 0 || len(data) > 32<<20 {
		return report, errors.New("controller input is empty or exceeds its byte bound")
	}
	if err := rejectApplyReceiptFieldAliases(data); err != nil {
		return report, err
	}
	if err := decodeControllerJSON(data, &report, 32<<20); err != nil {
		return report, err
	}
	if err := validateCanonicalControllerApplyReceipt(report); err != nil {
		return report, err
	}
	return report, nil
}

// Go's struct decoder matches JSON field names case-insensitively. Keep this
// guard receipt-local so legacy controller decoders retain their established
// behavior while aliases cannot overwrite receipt fields during decoding.
func rejectApplyReceiptFieldAliases(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := walkApplyReceiptJSON(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("Apply receipt must contain exactly one JSON value")
	}
	return nil
}

func walkApplyReceiptJSON(decoder *json.Decoder, depth int) error {
	if depth > 256 {
		return errors.New("Apply receipt JSON nesting exceeds its depth bound")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		keys := []string{}
		for decoder.More() {
			// Receipt objects are closed structs, with fewer than 64 fields.
			// Bound comparisons even for a large object of unknown members.
			if len(keys) >= 64 {
				return errors.New("Apply receipt object exceeds its member bound")
			}
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("Apply receipt object has a non-string member name")
			}
			for _, prior := range keys {
				if strings.EqualFold(prior, key) {
					return fmt.Errorf("Apply receipt contains case-insensitive duplicate member %q", key)
				}
			}
			keys = append(keys, key)
			if err := walkApplyReceiptJSON(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("Apply receipt has an unterminated JSON object")
		}
	case '[':
		for decoder.More() {
			if err := walkApplyReceiptJSON(decoder, depth+1); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("Apply receipt has an unterminated JSON array")
		}
	default:
		return errors.New("Apply receipt contains an unexpected JSON delimiter")
	}
	return nil
}

func validateCanonicalControllerApplyReceipt(report CanonicalControllerApply) error {
	if report.Status != records.StateMaterializedUnverified {
		return fmt.Errorf("Apply receipt status must be %q", records.StateMaterializedUnverified)
	}
	if !canonicalRevisionPattern.MatchString(report.SourceRevision) {
		return errors.New("Apply receipt sourceRevision must be a full lowercase source revision")
	}
	if !validSHA256(report.RunDigest) {
		return errors.New("Apply receipt runDigest must be a sha256 digest")
	}
	if !validSHA256(report.LedgerHead) {
		return errors.New("Apply receipt ledgerHead must be a sha256 digest")
	}
	if !canonicalRevisionPattern.MatchString(report.EvidenceRevision) {
		return errors.New("Apply receipt evidenceRevision must be a full lowercase source revision")
	}
	if len(report.Records) == 0 || len(report.Records) > 128 {
		return errors.New("Apply receipt must contain between 1 and 128 materialized records")
	}
	recordIDs := make(map[string]struct{}, len(report.Records))
	projectionIDs := make(map[string]struct{}, len(report.Records))
	for i, record := range report.Records {
		if err := records.ValidateProjectionRecord(record); err != nil {
			return fmt.Errorf("Apply receipt record %d is invalid: %w", i, err)
		}
		if record.State != records.StateMaterializedUnverified {
			return fmt.Errorf("Apply receipt record %d is not materialized-unverified", i)
		}
		if record.Revision != report.SourceRevision {
			return fmt.Errorf("Apply receipt record %d source revision does not match the report", i)
		}
		if _, exists := recordIDs[record.ID]; exists {
			return fmt.Errorf("Apply receipt contains duplicate record ID %q", record.ID)
		}
		if _, exists := projectionIDs[record.ProjectionID]; exists {
			return fmt.Errorf("Apply receipt contains duplicate projection ID %q", record.ProjectionID)
		}
		recordIDs[record.ID] = struct{}{}
		projectionIDs[record.ProjectionID] = struct{}{}
	}
	if len(report.EvidenceRefreshRequired) > 128 {
		return errors.New("Apply receipt evidenceRefreshRequired exceeds 128 projection IDs")
	}
	refreshSeen := make(map[string]struct{}, len(report.EvidenceRefreshRequired))
	for _, id := range report.EvidenceRefreshRequired {
		if id == "" || len(id) > 1024 || id != strings.TrimSpace(id) {
			return errors.New("Apply receipt evidenceRefreshRequired contains an invalid projection ID")
		}
		if _, exists := refreshSeen[id]; exists {
			return fmt.Errorf("Apply receipt evidenceRefreshRequired contains duplicate projection ID %q", id)
		}
		refreshSeen[id] = struct{}{}
	}
	if !sort.StringsAreSorted(report.EvidenceRefreshRequired) {
		return errors.New("Apply receipt evidenceRefreshRequired must be sorted")
	}
	return nil
}

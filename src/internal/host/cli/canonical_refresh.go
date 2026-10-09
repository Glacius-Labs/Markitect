package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host"
)

func runCanonicalEvidenceRefresh(o commandOptions, cfg host.CanonicalControllerConfig, emit func(any) int, fail func(error) int) int {
	if o.action == "controller-refresh-propose" {
		data, err := readBoundedControllerFile(o.reviewReport, canonicalControllerConfigLimit)
		if err != nil {
			return fail(fmt.Errorf("read explicit Projection ID selection: %w", err))
		}
		var ids []string
		if err := decodeCanonicalRefreshJSON(data, &ids); err != nil {
			return fail(err)
		}
		proposal, err := host.ProposeCanonicalEvidenceRefresh(o.root, o.base, o.revision, o.reviewConfig, cfg, ids)
		if code := emit(proposal); code != 0 {
			return code
		}
		if err != nil {
			return fail(err)
		}
		return canonicalControllerStatusExit(proposal.Status)
	}
	data, err := readBoundedControllerFile(o.plan, canonicalReviewedRunLimit)
	if err != nil {
		return fail(fmt.Errorf("read reviewed evidence refresh: %w", err))
	}
	var reviewed host.CanonicalEvidenceRefreshProposal
	if err := decodeCanonicalRefreshJSON(data, &reviewed); err != nil {
		return fail(err)
	}
	if reviewed.SourceRevision != o.base || reviewed.EvidenceRevision != o.revision {
		return fail(errors.New("--base and --revision must exactly match the reviewed evidence refresh"))
	}
	report, err := host.ApplyCanonicalEvidenceRefresh(o.root, o.reviewConfig, cfg, reviewed, o.expect, o.write)
	if code := emit(report); code != 0 {
		return code
	}
	if err != nil {
		return fail(err)
	}
	if report.Status == "refreshed" {
		return 0
	}
	return canonicalControllerStatusExit(report.Status)
}

func decodeCanonicalRefreshJSON(data []byte, out any) error {
	if err := rejectDuplicateCanonicalJSONFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode evidence refresh JSON: %w", err)
	}
	return ensureJSONEOF(decoder)
}

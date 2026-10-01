package app

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateReviewableProject(p *Project) error {
	if p == nil || p.Snapshot == nil || p.Graph == nil {
		return errors.New("project snapshot and graph are required")
	}
	if p.Snapshot.Provisional {
		return errors.New("review evidence requires a fixed, non-provisional snapshot")
	}
	if !validSnapshotID(p.Snapshot.ID) {
		return errors.New("fixed snapshot identity must be nonempty valid UTF-8 without NUL")
	}
	if len(p.Diagnostics) != 0 {
		return errors.New("project diagnostics must be clear")
	}
	if findings := CheckOutputs(p); len(findings) != 0 {
		return fmt.Errorf("generated outputs must be current: %s", findings[0].Code)
	}
	return nil
}

func validSnapshotID(id string) bool {
	return strings.TrimSpace(id) != "" && utf8.ValidString(id) && !strings.ContainsRune(id, 0)
}

func validateReviewConfig(config ReviewConfig) error {
	for name, value := range map[string]string{
		"question": config.Question, "promptVersion": config.PromptVersion,
		"model": config.Model, "effort": config.Effort,
	} {
		if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) || !utf8.ValidString(value) {
			return fmt.Errorf("review config %s is required", name)
		}
	}
	return nil
}

func sameReviewContentConfig(a, b ReviewConfig) bool {
	return a.Question == b.Question && a.PromptVersion == b.PromptVersion && a.Model == b.Model && a.Effort == b.Effort && a.AllowReuse == b.AllowReuse
}

func validateReviewRecord(record *ReviewRecord) error {
	if record == nil {
		return errors.New("review record is required")
	}
	if record.SchemaVersion != ReviewSchemaVersion {
		return fmt.Errorf("unsupported review record schema version %d", record.SchemaVersion)
	}
	if strings.TrimSpace(record.Entry) == "" || !validSnapshotID(record.Revision) || strings.TrimSpace(record.SnapshotDigest) == "" || strings.TrimSpace(record.ContextDigest) == "" || strings.TrimSpace(record.ToolDigest) == "" || strings.TrimSpace(record.Version) == "" {
		return errors.New("review record is missing required evidence identity")
	}
	if err := validateReviewConfig(record.Config); err != nil {
		return fmt.Errorf("invalid review record config: %w", err)
	}
	if len(record.Report) == 0 || len(record.Report) > maxReviewReportSize || strings.TrimSpace(record.Report) == "" || !utf8.ValidString(record.Report) || strings.ContainsRune(record.Report, 0) {
		return errors.New("review record has an invalid report")
	}
	if record.ReportDigest != Hash([]byte(record.Report)) {
		return errors.New("review record report digest mismatch")
	}
	if record.RecordedAt.IsZero() {
		return errors.New("review record timestamp must be UTC")
	}
	if _, offset := record.RecordedAt.Zone(); offset != 0 {
		return errors.New("review record timestamp must be UTC")
	}
	if !record.Advisory || record.TrustNotice != ReviewTrustNotice {
		return errors.New("review record advisory trust notice is missing or invalid")
	}
	return nil
}

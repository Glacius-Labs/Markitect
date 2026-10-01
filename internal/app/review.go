package app

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/format"
)

const (
	ReviewSchemaVersion = 1
	maxReviewConfigSize = 1 << 20
	maxReviewRecordSize = 2 << 20
	maxReviewReportSize = 1 << 20
	ReviewTrustNotice   = "Advisory AI review only; this record does not establish human acceptance or authenticated CI evidence."
)

// ReviewConfig identifies the question and exact reviewer setup. AllowReuse is
// an opt-in control and does not change the content of an original review.
type ReviewConfig struct {
	Question      string `yaml:"question"`
	PromptVersion string `yaml:"promptVersion"`
	Model         string `yaml:"model"`
	Effort        string `yaml:"effort"`
	AllowReuse    bool   `yaml:"allowReuse"`
}

// ReviewRecord stores a plain reviewer report tied to a fixed Markitect
// context. It is advisory evidence; its contents are not an accepted verdict.
type ReviewRecord struct {
	SchemaVersion  int          `yaml:"schemaVersion"`
	Entry          string       `yaml:"entry"`
	Revision       string       `yaml:"revision"`
	SnapshotDigest string       `yaml:"snapshotDigest"`
	ContextDigest  string       `yaml:"contextDigest"`
	ToolDigest     string       `yaml:"toolDigest"`
	Version        string       `yaml:"version"`
	Config         ReviewConfig `yaml:"config"`
	Report         string       `yaml:"report"`
	ReportDigest   string       `yaml:"reportDigest"`
	RecordedAt     time.Time    `yaml:"recordedAt"`
	Advisory       bool         `yaml:"advisory"`
	TrustNotice    string       `yaml:"trustNotice"`
}

type ReviewStatus string

const (
	ReviewReusable ReviewStatus = "reusable"
	ReviewRequired ReviewStatus = "review-required"
)

// ReviewDecision explains whether an earlier report can be shown for the
// candidate. A reusable report remains advisory and requires caller review.
type ReviewDecision struct {
	Status           ReviewStatus `yaml:"status"`
	Reasons          []string     `yaml:"reasons"`
	OriginalRevision string       `yaml:"originalRevision"`
	Candidate        string       `yaml:"candidate"`
	Entry            string       `yaml:"entry"`
	Report           string       `yaml:"report,omitempty"`
	Advisory         bool         `yaml:"advisory"`
	TrustNotice      string       `yaml:"trustNotice"`
}

// RecordReview binds an unparsed reviewer report to an exact immutable source
// snapshot, resolved context, tool digest and review configuration.
func RecordReview(p *Project, key, version, toolDigest string, config ReviewConfig, report string) (*ReviewRecord, error) {
	if err := validateReviewConfig(config); err != nil {
		return nil, err
	}
	if strings.TrimSpace(version) == "" || strings.TrimSpace(toolDigest) == "" {
		return nil, errors.New("review version and tool digest are required")
	}
	if len(report) == 0 || strings.TrimSpace(report) == "" || len(report) > maxReviewReportSize || !utf8.ValidString(report) || strings.ContainsRune(report, 0) {
		return nil, errors.New("review report must be nonempty valid UTF-8 without NUL and no larger than 1 MiB")
	}
	if err := validateReviewableProject(p); err != nil {
		return nil, err
	}
	ctx, err := CompileContext(p, key, version, toolDigest)
	if err != nil {
		return nil, fmt.Errorf("compile review context: %w", err)
	}
	record := &ReviewRecord{
		SchemaVersion: ReviewSchemaVersion,
		Entry:         key, Revision: p.Snapshot.ID,
		SnapshotDigest: p.Snapshot.Digest(), ContextDigest: ctx.Digest,
		ToolDigest: toolDigest, Version: version, Config: config,
		Report: report, ReportDigest: Hash([]byte(report)),
		RecordedAt: time.Now().UTC(), Advisory: true, TrustNotice: ReviewTrustNotice,
	}
	encoded, err := format.Encode(record)
	if err != nil {
		return nil, fmt.Errorf("encode review record: %w", err)
	}
	if len(encoded) > maxReviewRecordSize {
		return nil, errors.New("encoded review record exceeds the 2 MiB limit")
	}
	return record, nil
}

// ReuseReview validates the original evidence, then compares context identity
// and graph impact. It never parses or upgrades the report into a verdict.
func ReuseReview(before, after *Project, record *ReviewRecord, version, toolDigest string, config ReviewConfig) (*ReviewDecision, error) {
	if err := validateReviewRecord(record); err != nil {
		return nil, err
	}
	if err := validateReviewConfig(config); err != nil {
		return nil, err
	}
	if strings.TrimSpace(version) == "" || strings.TrimSpace(toolDigest) == "" {
		return nil, errors.New("review version and tool digest are required")
	}
	if err := validateReviewableProject(before); err != nil {
		return nil, fmt.Errorf("invalid review evidence base: %w", err)
	}
	if before.Snapshot.ID != record.Revision || before.Snapshot.Digest() != record.SnapshotDigest {
		return nil, errors.New("review evidence base does not match the recorded revision and snapshot digest")
	}
	if _, ok := before.Graph.Resources[record.Entry]; !ok {
		return nil, fmt.Errorf("review evidence base is missing entry %s", record.Entry)
	}
	originalContext, err := CompileContext(before, record.Entry, record.Version, record.ToolDigest)
	if err != nil {
		return nil, fmt.Errorf("recompute original review context: %w", err)
	}
	if originalContext.Digest != record.ContextDigest {
		return nil, errors.New("review record context digest does not match the recorded evidence base")
	}
	if record.ReportDigest != Hash([]byte(record.Report)) {
		return nil, errors.New("review record report digest mismatch")
	}

	decision := &ReviewDecision{
		Status: ReviewRequired, OriginalRevision: record.Revision,
		Entry: record.Entry, Advisory: true, TrustNotice: ReviewTrustNotice,
	}
	if after != nil && after.Snapshot != nil {
		decision.Candidate = after.Snapshot.ID
	}
	addReason := func(reason string) { decision.Reasons = append(decision.Reasons, reason) }
	if !config.AllowReuse {
		addReason("review reuse is disabled by the current configuration")
	}
	if !sameReviewContentConfig(record.Config, config) {
		addReason("review config changed")
	}
	if record.Version != version {
		addReason("Markitect version changed")
	}
	if record.ToolDigest != toolDigest {
		addReason("tool digest changed")
	}
	if err := validateReviewableProject(after); err != nil {
		addReason("candidate is not reviewable: " + err.Error())
	} else {
		if _, ok := after.Graph.Resources[record.Entry]; !ok {
			addReason("review entry is missing from the candidate")
		} else {
			candidateContext, ctxErr := CompileContext(after, record.Entry, version, toolDigest)
			if ctxErr != nil {
				addReason("candidate context cannot be compiled: " + ctxErr.Error())
			} else if candidateContext.Digest != record.ContextDigest {
				addReason("resolved entry context changed")
			}
			impact := Changes(before, after)
			for _, key := range impact.Affected {
				if key == record.Entry {
					addReason("changed source, dependency, configuration, or inventory affects the review entry")
					break
				}
			}
		}
	}
	if len(decision.Reasons) == 0 {
		decision.Status = ReviewReusable
		decision.Report = record.Report
	}
	return decision, nil
}

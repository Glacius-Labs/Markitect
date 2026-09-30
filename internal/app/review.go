package app

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
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
		Entry:         key, Revision: p.Snapshot.Revision,
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
	if before.Snapshot.Revision != record.Revision || before.Snapshot.Digest() != record.SnapshotDigest {
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
		decision.Candidate = after.Snapshot.Revision
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

func validateReviewableProject(p *Project) error {
	if p == nil || p.Snapshot == nil || p.Graph == nil {
		return errors.New("project snapshot and graph are required")
	}
	if p.Snapshot.Provisional {
		return errors.New("review evidence requires a fixed, non-provisional snapshot")
	}
	if !validRevision(p.Snapshot.Revision) {
		return errors.New("fixed snapshot revision must be a full 40- or 64-character hexadecimal commit id")
	}
	if len(p.Diagnostics) != 0 {
		return errors.New("project diagnostics must be clear")
	}
	if findings := CheckOutputs(p); len(findings) != 0 {
		return fmt.Errorf("generated outputs must be current: %s", findings[0].Code)
	}
	return nil
}

func validRevision(revision string) bool {
	if len(revision) != 40 && len(revision) != 64 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil
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
	if strings.TrimSpace(record.Entry) == "" || !validRevision(record.Revision) || strings.TrimSpace(record.SnapshotDigest) == "" || strings.TrimSpace(record.ContextDigest) == "" || strings.TrimSpace(record.ToolDigest) == "" || strings.TrimSpace(record.Version) == "" {
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

// DecodeReviewConfig strictly decodes one bounded YAML review configuration.
func DecodeReviewConfig(data []byte) (ReviewConfig, error) {
	var config ReviewConfig
	if err := strictReviewDecode(data, maxReviewConfigSize, &config, "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return config, err
	}
	if err := validateReviewConfig(config); err != nil {
		return config, err
	}
	return config, nil
}

// DecodeReviewRecord strictly decodes one bounded YAML review record and
// validates its stored report digest and advisory marker.
func DecodeReviewRecord(data []byte) (*ReviewRecord, error) {
	var record ReviewRecord
	if err := strictReviewDecode(data, maxReviewRecordSize, &record,
		"schemaVersion", "entry", "revision", "snapshotDigest", "contextDigest", "toolDigest", "version", "config", "report", "reportDigest", "recordedAt", "advisory", "trustNotice"); err != nil {
		return nil, err
	}
	record.RecordedAt = record.RecordedAt.UTC()
	if err := requireNestedReviewKeys(data, "config", "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return nil, err
	}
	if err := validateReviewRecord(&record); err != nil {
		return nil, err
	}
	return &record, nil
}

func strictReviewDecode(data []byte, max int, out any, required ...string) error {
	if len(data) == 0 || len(data) > max || !utf8.Valid(data) {
		return fmt.Errorf("review YAML must be valid UTF-8, nonempty, and no larger than %d bytes", max)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("review YAML root must be one mapping")
	}
	if err := inspectReviewNode(&doc); err != nil {
		return err
	}
	if err := requireReviewKeys(doc.Content[0], required...); err != nil {
		return err
	}
	if err := validateReviewScalarTypes(doc.Content[0], out); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple review YAML documents are not allowed")
		}
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(out); err != nil {
		return fmt.Errorf("invalid review YAML fields: %w", err)
	}
	return nil
}

func validateReviewScalarTypes(root *yaml.Node, out any) error {
	fields := map[string]string{}
	if _, ok := out.(*ReviewConfig); ok {
		fields = map[string]string{"question": "!!str", "promptVersion": "!!str", "model": "!!str", "effort": "!!str", "allowReuse": "!!bool"}
	} else {
		fields = map[string]string{
			"schemaVersion": "!!int", "entry": "!!str", "revision": "!!str", "snapshotDigest": "!!str",
			"contextDigest": "!!str", "toolDigest": "!!str", "version": "!!str", "report": "!!str",
			"reportDigest": "!!str", "advisory": "!!bool", "trustNotice": "!!str",
		}
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "recordedAt" {
				n := root.Content[i+1]
				if n.Kind != yaml.ScalarNode || (n.Tag != "!!timestamp" && n.Tag != "!!str") {
					return errors.New("review YAML field \"recordedAt\" must be a timestamp or string")
				}
				if strings.ContainsRune(n.Value, 0) {
					return errors.New("review YAML field \"recordedAt\" contains NUL")
				}
			}
			if root.Content[i].Value == "config" {
				if err := validateReviewConfigNode(root.Content[i+1]); err != nil {
					return err
				}
			}
		}
	}
	for field, tag := range fields {
		var value *yaml.Node
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == field {
				value = root.Content[i+1]
				break
			}
		}
		if value == nil || value.Kind != yaml.ScalarNode || value.Tag != tag {
			return fmt.Errorf("review YAML field %q must be a %s scalar", field, strings.TrimPrefix(tag, "!!"))
		}
		if strings.ContainsRune(value.Value, 0) {
			return fmt.Errorf("review YAML field %q contains NUL", field)
		}
	}
	return nil
}

func validateReviewConfigNode(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return errors.New("review YAML field \"config\" must be a mapping")
	}
	if err := requireReviewKeys(node, "question", "promptVersion", "model", "effort", "allowReuse"); err != nil {
		return err
	}
	fields := map[string]string{"question": "!!str", "promptVersion": "!!str", "model": "!!str", "effort": "!!str", "allowReuse": "!!bool"}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		value := node.Content[i+1]
		if value.Kind != yaml.ScalarNode || value.Tag != fields[key] {
			return fmt.Errorf("review YAML config field %q has the wrong scalar type", key)
		}
		if strings.ContainsRune(value.Value, 0) {
			return fmt.Errorf("review YAML config field %q contains NUL", key)
		}
	}
	return nil
}

func requireNestedReviewKeys(data []byte, parent string, keys ...string) error {
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return fmt.Errorf("invalid review YAML: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return errors.New("review YAML root must be one mapping")
	}
	root := doc.Content[0]
	var nested *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == parent {
			nested = root.Content[i+1]
			break
		}
	}
	if nested == nil || nested.Kind != yaml.MappingNode {
		return fmt.Errorf("review YAML field %q must be a mapping", parent)
	}
	return requireReviewKeys(nested, keys...)
}

func requireReviewKeys(node *yaml.Node, keys ...string) error {
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		seen[node.Content[i].Value] = true
	}
	for _, key := range keys {
		if !seen[key] {
			return fmt.Errorf("review YAML is missing required field %q", key)
		}
	}
	return nil
}

func inspectReviewNode(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return errors.New("review YAML anchors and aliases are not allowed")
	}
	if node.Tag != "" && node.Tag != "!!map" && node.Tag != "!!seq" && node.Tag != "!!str" && node.Tag != "!!int" && node.Tag != "!!float" && node.Tag != "!!bool" && node.Tag != "!!null" && node.Tag != "!!timestamp" && node.Tag != "!!binary" {
		return errors.New("custom YAML tags are not allowed in review YAML")
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return errors.New("review YAML mapping keys must be strings")
			}
			if seen[key.Value] {
				return fmt.Errorf("duplicate review YAML key %q", key.Value)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := inspectReviewNode(child); err != nil {
			return err
		}
	}
	return nil
}

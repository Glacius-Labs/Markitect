package host

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func receiptTestDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func receiptTestRecord(t *testing.T, revision, projectionID, artifact string) records.ProjectionRecord {
	t.Helper()
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: revision, ModelDigest: receiptTestDigest("model"), PlanDigest: receiptTestDigest("plan"),
		InputSnapshotDigest: receiptTestDigest("inputs"), RequestDigest: receiptTestDigest("request"),
		ProjectionID: projectionID,
		Module:       records.ModuleIdentity{Name: "foundation", Version: "1.0.0", Digest: receiptTestDigest("module")},
		Projector:    records.ProjectorIdentity{ID: "markdown", Version: "1.0.0"},
		ScopeIDs:     []string{"software/v1:UseCase:sample"},
		Artifacts:    []records.Artifact{{Path: artifact, Digest: receiptTestDigest(artifact), Mode: "100644", Change: records.ChangeModified}},
		State:        records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func validApplyReceipt(t *testing.T) CanonicalControllerApply {
	t.Helper()
	revision := strings.Repeat("a", 40)
	return CanonicalControllerApply{
		Status: records.StateMaterializedUnverified, SourceRevision: revision,
		RunDigest: receiptTestDigest("run"), LedgerHead: receiptTestDigest("ledger"),
		EvidenceRevision:        strings.Repeat("b", 40),
		Written:                 []string{"docs/sample.md"},
		Records:                 []records.ProjectionRecord{receiptTestRecord(t, revision, "foundation/v1:Projection:sample", "docs/sample.md")},
		EvidenceRefreshRequired: []string{},
	}
}

func encodeApplyReceipt(t *testing.T, report CanonicalControllerApply) []byte {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeCanonicalControllerApplyAcceptsMaterializedReceiptAndRefreshRequirement(t *testing.T) {
	report := validApplyReceipt(t)
	report.EvidenceRefreshRequired = []string{"foundation/v1:Projection:retained"}
	data := encodeApplyReceipt(t, report)
	decoded, err := DecodeCanonicalControllerApply(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SourceRevision != report.SourceRevision || decoded.EvidenceRevision != report.EvidenceRevision || len(decoded.Records) != 1 {
		t.Fatalf("decoded receipt lost its bindings: %#v", decoded)
	}
	if len(decoded.EvidenceRefreshRequired) != 1 || decoded.EvidenceRefreshRequired[0] != report.EvidenceRefreshRequired[0] {
		t.Fatalf("refresh requirement was not preserved: %#v", decoded.EvidenceRefreshRequired)
	}
}

func TestDecodeCanonicalControllerApplyRejectsIncompleteOrUnboundReceipts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*CanonicalControllerApply)
	}{
		{"refused status", func(r *CanonicalControllerApply) { r.Status = "refused" }},
		{"partial failure status", func(r *CanonicalControllerApply) { r.Status = records.StatePartialFailure }},
		{"no materialization work", func(r *CanonicalControllerApply) {
			r.Status = "no-materialization-work"
			r.Records = nil
		}},
		{"missing source revision", func(r *CanonicalControllerApply) { r.SourceRevision = "" }},
		{"noncanonical source revision", func(r *CanonicalControllerApply) { r.SourceRevision = strings.Repeat("A", 40) }},
		{"missing evidence revision", func(r *CanonicalControllerApply) { r.EvidenceRevision = "" }},
		{"no materialized records", func(r *CanonicalControllerApply) { r.Records = nil }},
		{"invalid run digest", func(r *CanonicalControllerApply) { r.RunDigest = "PASS" }},
		{"invalid ledger head", func(r *CanonicalControllerApply) { r.LedgerHead = "HEAD" }},
		{"record from another source", func(r *CanonicalControllerApply) {
			r.Records[0] = receiptTestRecord(t, strings.Repeat("c", 40), "foundation/v1:Projection:sample", "docs/sample.md")
		}},
		{"partial failure record", func(r *CanonicalControllerApply) {
			record := r.Records[0]
			record.State = records.StatePartialFailure
			var err error
			r.Records[0], err = records.NewProjectionRecord(record)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"tampered record identity", func(r *CanonicalControllerApply) { r.Records[0].ID = receiptTestDigest("forged") }},
		{"duplicate projection record", func(r *CanonicalControllerApply) {
			r.Records = append(r.Records, receiptTestRecord(t, r.SourceRevision, "foundation/v1:Projection:sample", "docs/other.md"))
		}},
		{"duplicate refresh requirement", func(r *CanonicalControllerApply) { r.EvidenceRefreshRequired = []string{"x", "x"} }},
		{"unsorted refresh requirements", func(r *CanonicalControllerApply) { r.EvidenceRefreshRequired = []string{"z", "a"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validApplyReceipt(t)
			test.change(&report)
			if _, err := DecodeCanonicalControllerApply(encodeApplyReceipt(t, report)); err == nil {
				t.Fatal("expected receipt to be rejected")
			}
		})
	}
}

func TestDecodeCanonicalControllerApplyKeepsStrictJSONBoundary(t *testing.T) {
	data := encodeApplyReceipt(t, validApplyReceipt(t))
	duplicateKey := append([]byte(`{"status":"materialized-unverified",`), data[1:]...)
	if _, err := DecodeCanonicalControllerApply(duplicateKey); err == nil {
		t.Fatal("expected duplicate JSON key to be rejected")
	}
	unknownField := strings.TrimSuffix(string(data), "}") + `,"unexpected":true}`
	if _, err := DecodeCanonicalControllerApply([]byte(unknownField)); err == nil {
		t.Fatal("expected unknown JSON field to be rejected")
	}
}

func TestDecodeCanonicalControllerApplyRejectsCaseInsensitiveFieldAliases(t *testing.T) {
	valid := string(encodeApplyReceipt(t, validApplyReceipt(t)))
	rootFirst := strings.Replace(valid, `"status":"materialized-unverified"`, `"status":"refused","Status":"materialized-unverified"`, 1)
	rootSecond := strings.Replace(valid, `"status":"materialized-unverified"`, `"Status":"materialized-unverified","status":"refused"`, 1)
	escapedRoot := strings.Replace(valid, `"status":"materialized-unverified"`, `"\u0073tatus":"refused","Status":"materialized-unverified"`, 1)
	nestedFirst := strings.Replace(valid, `"state":"materialized-unverified"`, `"State":"partial-failure","state":"materialized-unverified"`, 1)
	nestedSecond := strings.Replace(valid, `"state":"materialized-unverified"`, `"state":"materialized-unverified","State":"partial-failure"`, 1)
	tests := []struct {
		name string
		data string
	}{
		{"root alias original first", rootFirst},
		{"root alias variant first", rootSecond},
		{"escaped root member alias", escapedRoot},
		{"nested record alias original first", nestedFirst},
		{"nested record alias variant first", nestedSecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeCanonicalControllerApply([]byte(test.data)); err == nil || !strings.Contains(err.Error(), "case-insensitive duplicate member") {
				t.Fatalf("expected an ambiguous-member refusal, got %v", err)
			}
		})
	}
}

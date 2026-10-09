package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
)

func TestReviewYAMLDecodersRejectUnsafeOrIncompleteDocuments(t *testing.T) {
	configData, err := authoring.Encode(testReviewConfig)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := DecodeReviewConfig(configData); err != nil || got != testReviewConfig {
		t.Fatalf("config round trip = %#v, %v", got, err)
	}
	for _, data := range [][]byte{
		append(append([]byte(nil), configData...), []byte("question: duplicate\n")...),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\nextra: field\n"),
		[]byte("question: &q x\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n"),
		[]byte("question: !custom x\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n"),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\nallowReuse: true\n---\nquestion: q\n"),
		[]byte("question: q\npromptVersion: p\nmodel: m\neffort: e\n"),
	} {
		if _, err := DecodeReviewConfig(data); err == nil {
			t.Fatalf("unsafe or incomplete config was accepted: %s", data)
		}
	}

	p := reviewFixture(t, strings.Repeat("a", 40), false, reviewResources(false, false, false), nil)
	record := mustRecordReview(t, p, testReviewConfig)
	recordData, err := authoring.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeReviewRecord(recordData)
	if err != nil {
		t.Fatalf("record round trip failed: %v\n%s", err, recordData)
	}
	if decoded.Report != record.Report || decoded.ContextDigest != record.ContextDigest || decoded.TrustNotice != ReviewTrustNotice {
		t.Fatalf("record round trip changed evidence: %#v", decoded)
	}
	if _, err := DecodeReviewRecord(append(recordData, []byte("\nextra: value\n")...)); err == nil {
		t.Fatal("record with unknown field was accepted")
	}
}

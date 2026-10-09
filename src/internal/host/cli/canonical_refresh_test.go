package cli

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"strings"
	"testing"
)

func TestCanonicalRefreshExplicitCLIContract(t *testing.T) {
	common := []string{"canonical", "--config", "canonical.yaml", "--runtime", "runtime.json", "--base", strings.Repeat("a", 40), "--revision", strings.Repeat("b", 40)}
	cases := []struct {
		args  []string
		valid bool
	}{
		{[]string{"--action", "controller-refresh-propose", "--report", "ids.json"}, true},
		{[]string{"--action", "controller-refresh-propose"}, false},
		{[]string{"--action", "controller-refresh-propose", "--report", "ids.json", "--write"}, false},
		{[]string{"--action", "controller-refresh-apply", "--write", "--plan", "reviewed.json", "--expect", "digest"}, true},
		{[]string{"--action", "controller-refresh-apply", "--plan", "reviewed.json", "--expect", "digest"}, false},
		{[]string{"--action", "controller-refresh-apply", "--write", "--plan", "reviewed.json", "--expect", "digest", "--report", "ids.json"}, false},
	}
	for _, c := range cases {
		args := append(append([]string{}, common...), c.args...)
		var out, errs bytes.Buffer
		flags, _ := commandFlags("canonical")
		_, _, done := parseOptions("canonical", args, flags, &out, &errs)
		if done == c.valid {
			t.Fatalf("args %v valid=%v: %s", c.args, c.valid, errs.String())
		}
	}
}

func TestCanonicalRefreshRejectsMalformedReviewedJSON(t *testing.T) {
	for _, data := range []string{`{"status":"planned","status":"blocked"}`, `{"unknown":true}`, `{} {}`} {
		var reviewed host.CanonicalEvidenceRefreshProposal
		if err := decodeCanonicalRefreshJSON([]byte(data), &reviewed); err == nil {
			t.Fatalf("accepted malformed proposal %s", data)
		}
	}
}

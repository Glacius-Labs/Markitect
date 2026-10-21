package agentd

import "testing"

func TestParseStatKeepsCommandsWithSpacesAndParentheses(t *testing.T) {
	p, err := parseStat("42 (node (codex) x) S 7 42 42 0 -1 4194560")
	if err != nil {
		t.Fatal(err)
	}
	if p.PID != 42 || p.PPID != 7 || p.PGID != 42 || p.State != "S" || p.Command != "node (codex) x" {
		t.Fatalf("parsed %+v", p)
	}
	for _, bad := range []string{"", "42 node S 7 42", "x (a) S 1 1", "42 (a) S"} {
		if _, err := parseStat(bad); err == nil {
			t.Fatalf("malformed stat %q accepted", bad)
		}
	}
}

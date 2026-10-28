package runremote

import (
	"strings"
	"testing"
)

// typeRepo answers only the questions a push asks before its history.
type typeRepo struct {
	Repo
	kind string
}

func (r typeRepo) Type(string) (string, error) { return r.kind, nil }

// A Git client refuses to send these pushes; the hook refuses them anyway.
func TestPushesNoGitClientSends(t *testing.T) {
	plan := &Plan{Managers: []Manager{{ID: "", Working: true}}, Agents: []Agent{{User: "root-maker", Manager: "", Role: Maker}}}
	branch := "refs/heads/" + RootBranch
	old, tag := strings.Repeat("1", 40), strings.Repeat("2", 40)
	for _, tc := range []struct {
		updates []Update
		want    string
	}{
		{nil, "the push updates nothing"},
		{[]Update{{Old: old, New: tag, Ref: branch}}, "must point to a commit, not a tag"},
		{[]Update{{Old: ZeroID, New: tag, Ref: branch}}, "cannot be created"},
	} {
		d := Decide(plan, NewOwnership(plan), typeRepo{kind: "tag"}, nil, "root-maker", tc.updates)
		if d.Accepted || !strings.Contains(d.Reason, tc.want) {
			t.Errorf("%v: %+v, want a refusal with %q", tc.updates, d, tc.want)
		}
	}
	if d := Decide(plan, NewOwnership(plan), typeRepo{kind: "commit"}, nil, "stranger", nil); d.Accepted || !strings.Contains(d.Reason, "is no agent") {
		t.Errorf("unknown user: %+v", d)
	}
}

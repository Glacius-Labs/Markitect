package run

import "testing"

// A changed link is refused when it can leave the repository, on any host.
func TestLeavesTree(t *testing.T) {
	cases := []struct {
		to   string
		want bool
	}{
		{"limit.md", false},
		{"sub/limit.md", false},
		{"../limit.md", false},
		{"./limit.md", false},
		{"../../limit.md", true},
		{"sub/../../limit.md", true},
		{"/dev/shm/good.md", true},
		{"//server/share/good.md", true},
		{`..\..\good.md`, true},
		{"C:/Users/you/.ssh/id_rsa", true},
		{"c:good.md", true},
	}
	for _, c := range cases {
		if got := leavesTree("docs/x.md", c.to); got != c.want {
			t.Errorf("a link to %s: leaves the tree %v, want %v", c.to, got, c.want)
		}
	}
}

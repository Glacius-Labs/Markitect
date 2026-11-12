package cli

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// doc takes what status prints after look at, as well as an ID: the path of a
// file or folder of the spec, with the spec folder's prefix, with or without
// .md, with a leading ./ and with backslashes.
func TestSpecTarget(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"members/member", "members/member"},
		{"[[members/member]]", "members/member"},
		{"members/member.md", "members/member.md"},
		{".markitect/spec/members/member.md", "members/member.md"},
		{".markitect/spec/members/member", "members/member"},
		{"./.markitect/spec/members/member.md", "members/member.md"},
		{`.markitect\spec\members\member.md`, "members/member.md"},
		{`.\.markitect\spec\members`, "members"},
		{"[[.markitect/spec/members/member]]", "members/member"},
		{".markitect/spec/members/", "members"},
		{"members/", "members"},
		{".markitect/spec", spec.RootPackage},
		{".markitect/spec/", spec.RootPackage},
		{"Root", "Root"},
		{".", "."},
		// Under the project folder, but outside the spec: left as it is.
		{".markitect/libraries/std/Rule.md", ".markitect/libraries/std/Rule.md"},
		{".markitect/registry/members.md", ".markitect/registry/members.md"},
		{".markitect/specs/members.md", ".markitect/specs/members.md"},
		{"std/Rule", "std/Rule"},
	} {
		if got := specTarget(c.in); got != c.want {
			t.Errorf("specTarget(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

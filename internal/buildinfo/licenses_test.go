package buildinfo

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var requirement = regexp.MustCompile(`(?m)^\s*(?:require\s+)?([a-z0-9][^\s]*\.[^\s]+/[^\s]+)\s+(v[^\s]+)(?:\s*//.*)?$`)

// The notices name the Go distribution and every module go.mod requires, at
// the version it requires, so that a new or changed dependency fails here
// until its notice is reviewed.
func TestNoticesCoverEveryRequiredModule(t *testing.T) {
	notices := strings.ReplaceAll(Notices, "\r\n", "\n")
	if !strings.Contains(notices, "## Go runtime and standard library") {
		t.Error("the notices do not name the Go runtime and standard library")
	}
	mod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	required := requirement.FindAllStringSubmatch(string(mod), -1)
	if len(required) == 0 {
		t.Fatal("found no required module in go.mod")
	}
	for _, r := range required {
		module, version := r[1], r[2]
		heading := "## " + module + "\n"
		at := strings.Index(notices, heading)
		if at < 0 {
			t.Errorf("the notices have no section for %s", module)
			continue
		}
		section := notices[at+len(heading):]
		if next := strings.Index(section, "\n## "); next >= 0 {
			section = section[:next]
		}
		if !strings.Contains(section, "Version: "+version+".") {
			t.Errorf("the notice of %s does not name version %s", module, version)
		}
	}
}

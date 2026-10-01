package main

import (
	"strings"
	"testing"
)

func TestLicensesProvidesCompleteNoticesWithoutProject(t *testing.T) {
	code, output, stderr := invoke("licenses")
	if code != 0 || stderr != "" {
		t.Fatalf("licenses: code=%d stderr=%s", code, stderr)
	}
	for _, required := range []string{"Copyright 2009 The Go Authors.", "Copyright (c) 2006-2010 Kirill Simonov", "Copyright (c) 2011-2019 Canonical Ltd", "END OF TERMS AND CONDITIONS", "go.yaml.in/yaml/v3", "v3.0.5"} {
		if !strings.Contains(output, required) {
			t.Errorf("notice output omitted %q", required)
		}
	}
	for _, args := range [][]string{{"licenses", "--write"}, {"licenses", "--repo", "."}, {"licenses", "unexpected"}} {
		if code, _, _ := invoke(args...); code != 2 {
			t.Errorf("licenses accepted unsupported arguments %v: %d", args, code)
		}
	}
}

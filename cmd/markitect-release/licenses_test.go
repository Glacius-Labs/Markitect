package main

import (
	"strings"
	"testing"
)

func TestLicensesProvidesCompleteNoticesWithoutProject(t *testing.T) {
	code, output, stderr := invokeRelease(t, "licenses")
	if code != 0 || stderr != "" {
		t.Fatalf("licenses: code=%d stderr=%s", code, stderr)
	}
	for _, required := range []string{"Copyright 2009 The Go Authors.", "Copyright (c) 2006-2010 Kirill Simonov", "Copyright (c) 2011-2019 Canonical Ltd", "END OF TERMS AND CONDITIONS", "go.yaml.in/yaml/v3", "v3.0.5"} {
		if !strings.Contains(output, required) {
			t.Errorf("notice output omitted %q", required)
		}
	}
	for _, args := range [][]string{{"licenses", "--write"}, {"licenses", "--repo", "."}, {"licenses", "unexpected"}} {
		if code, _, _ := invokeRelease(t, args...); code != 2 {
			t.Errorf("licenses accepted unsupported arguments %v: %d", args, code)
		}
	}
}

func TestVersionWorksWithoutRepositoryAndRejectsArguments(t *testing.T) {
	code, output, stderr := invokeRelease(t, "version")
	if code != 0 || stderr != "" || !strings.HasPrefix(output, "Markitect ") {
		t.Fatalf("version: code=%d stdout=%q stderr=%q", code, output, stderr)
	}
	for _, args := range [][]string{{"version", "unexpected"}, {"version", "--repo", "."}} {
		if code, _, _ := invokeRelease(t, args...); code != 2 {
			t.Errorf("version accepted unsupported arguments %v: %d", args, code)
		}
	}
}

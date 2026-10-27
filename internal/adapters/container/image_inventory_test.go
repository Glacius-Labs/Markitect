package container

import (
	"archive/tar"
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestImageEnvironmentReadsTheConfiguredEntries(t *testing.T) {
	env, err := parseImageEnvironment([]byte(`["PATH=/usr/bin","HOME=/home/agent","EMPTY="]` + "\n"))
	if err != nil || !reflect.DeepEqual(env, []string{"PATH=/usr/bin", "HOME=/home/agent", "EMPTY="}) {
		t.Fatal(env, err)
	}
	if env, err := parseImageEnvironment([]byte("null\n")); err != nil || len(env) != 0 {
		t.Fatal("an image without environment", env, err)
	}
	for _, out := range []string{"", "{}", `["NOVALUE"]`, `[1]`} {
		if _, err := parseImageEnvironment([]byte(out)); err == nil {
			t.Fatalf("unreadable environment %q admitted", out)
		}
	}
}

func TestArchiveNamesReadHeadersWithinTheBound(t *testing.T) {
	var archive bytes.Buffer
	w := tar.NewWriter(&archive)
	for _, name := range []string{"etc/", "etc/hosts", "etc/claude-code/", "etc/claude-code/managed-settings.json"} {
		body := "content of " + name
		if strings.HasSuffix(name, "/") {
			body = ""
		}
		typ := byte(tar.TypeReg)
		if body == "" {
			typ = tar.TypeDir
		}
		if err := w.WriteHeader(&tar.Header{Name: name, Typeflag: typ, Size: int64(len(body)), Mode: 0o644}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	names, err := archiveNames(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil || !reflect.DeepEqual(names, []string{"etc", "etc/hosts", "etc/claude-code", "etc/claude-code/managed-settings.json"}) {
		t.Fatal(names, err)
	}
	if _, err := archiveNames(bytes.NewReader(archive.Bytes()), 1024); !errors.Is(err, errArchiveLimit) {
		t.Fatal("an archive above its bound was listed", err)
	}
	if _, err := archiveNames(strings.NewReader("not a tar archive at all, but long enough to fill one header block"+strings.Repeat("x", 600)), 1<<20); err == nil {
		t.Fatal("a malformed archive was listed")
	}
}

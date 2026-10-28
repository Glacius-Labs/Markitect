package model

import (
	"strings"
	"testing"
)

func TestParseManifest(t *testing.T) {
	m, ds := ParseManifest("manifest.yaml", []byte("libraries:\n  std: 1.0.0\n  rest: 1.2.3\n"))
	clean(t, ds)
	if len(m.Libraries) != 2 || m.Libraries[0].Name != "rest" || m.Libraries[1].Version != "1.0.0" {
		t.Fatalf("manifest = %+v", m)
	}
}

func TestParseManifestErrors(t *testing.T) {
	for name, data := range map[string]string{
		"unknown key":   "libraries:\n  std: 1.0.0\nproject: x\n",
		"no libraries":  "other: 1\n",
		"not a mapping": "libraries: [std]\n",
		"bad version":   "libraries:\n  std: latest\n",
		"bad name":      "libraries:\n  \"a b\": 1.0.0\n",
		"duplicate":     "libraries:\n  std: 1.0.0\n  std: 1.0.0\n",
		"not yaml":      "libraries: [\n",
	} {
		_, ds := ParseManifest("manifest.yaml", []byte(data))
		if len(ds) == 0 || ds[0].Code != "manifest.syntax" {
			t.Errorf("%s: diagnostics = %v", name, ds)
		}
	}
}

func TestLockRoundTripAndMismatch(t *testing.T) {
	libs := []LibrarySource{stdKinds}
	data := RenderLock(libs)
	lock, ds := ParseLock("lock.yaml", data)
	clean(t, ds)
	clean(t, VerifyLock("lock.yaml", lock, libs))

	changed := stdKinds
	changed.Files = append([]File{{Path: "Extra.md", Data: []byte("x")}}, changed.Files...)
	expect(t, VerifyLock("lock.yaml", lock, []LibrarySource{changed}), "lock.mismatch", "does not match the digest")

	newer := stdKinds
	newer.Version = "1.1.0"
	expect(t, VerifyLock("lock.yaml", lock, []LibrarySource{newer}), "lock.mismatch", "the lock pins std 1.0.0, but the manifest names 1.1.0")

	extra := LibrarySource{Name: "cli", Version: "1.0.0"}
	ds = VerifyLock("lock.yaml", lock, []LibrarySource{stdKinds, extra})
	expect(t, ds, "lock.mismatch", "cli 1.0.0 is not in the lock")
	expect(t, VerifyLock("lock.yaml", lock, nil), "lock.mismatch", "the manifest does not name")
}

func TestParseLockErrors(t *testing.T) {
	for _, data := range []string{"libraries: [\n", "x: 1\n", "libraries:\n  std:\n    version: 1.0.0\n"} {
		_, ds := ParseLock("lock.yaml", []byte(data))
		if len(ds) == 0 || ds[0].Code != "lock.syntax" {
			t.Errorf("%q: diagnostics = %v", data, ds)
		}
	}
}

func TestLibraryDigestIsOrderIndependent(t *testing.T) {
	a := LibraryDigest([]File{{Path: "A.md", Data: []byte("a")}, {Path: "B.md", Data: []byte("b")}})
	b := LibraryDigest([]File{{Path: "B.md", Data: []byte("b")}, {Path: "A.md", Data: []byte("a")}})
	if a != b || !strings.HasPrefix(a, "sha256:") {
		t.Fatalf("digests %s %s", a, b)
	}
	if a == LibraryDigest([]File{{Path: "A.md", Data: []byte("ab")}, {Path: "B.md", Data: []byte("")}}) {
		t.Fatal("digest does not separate files")
	}
}

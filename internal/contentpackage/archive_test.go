package contentpackage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestBuildReadDeterministicRoundTrip(t *testing.T) {
	files := map[string][]byte{
		ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata:
  name: standards
spec:
  version: 1.2.3
  areas:
    - name: base
      path: content
  exports:
    - kind: Rule
      namespace: base
      name: basics
`),
		"content/basics.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata:
  name: basics
  namespace: base
spec:
  text: Apply the package basics.
`),
		"README.md": []byte("unrelated source snapshot input"),
	}
	first, err := Build(files)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(files)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("Build returned different ZIP bytes for the same files")
	}
	digest := sha256.Sum256(first)
	got, err := Read(corePin("standards", "1.2.3", hex.EncodeToString(digest[:])), first)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Resources) != 1 || got.Resources[0].Package != "standards" || got.Resources[0].Path != "content/basics.yaml" {
		t.Fatalf("unexpected parsed resources: %#v", got.Resources)
	}
	if _, ok := got.Files["README.md"]; ok {
		t.Fatal("Build included an unrelated source snapshot input")
	}
}

func TestReadRejectsMaliciousZIPMembers(t *testing.T) {
	manifest := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: standards}
spec:
  version: 1.2.3
  areas: [{name: base, path: content}]
  exports: [{kind: Rule, namespace: base, name: basics}]
`)
	resource := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: basics, namespace: base}
spec: {text: Apply the package basics.}
`)
	tests := []struct {
		name    string
		members []zipMember
	}{
		{name: "traversal", members: []zipMember{{ManifestName, manifest, 0}, {"../escape.txt", []byte("x"), 0}}},
		{name: "case collision", members: []zipMember{{ManifestName, manifest, 0}, {"content/basics.yaml", resource, 0}, {"content/BASICS.yaml", resource, 0}}},
		{name: "duplicate member", members: []zipMember{{ManifestName, manifest, 0}, {ManifestName, manifest, 0}}},
		{name: "symlink", members: []zipMember{{ManifestName, manifest, 0}, {"content/link.yaml", []byte("target"), fs.ModeSymlink | 0o777}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := makeZIP(t, test.members)
			digest := sha256.Sum256(data)
			if _, err := Read(corePin("standards", "1.2.3", hex.EncodeToString(digest[:])), data); err == nil {
				t.Fatal("Read accepted malicious ZIP members")
			}
		})
	}
}

func TestBuildRejectsDeclaredInputsThatHideMarkitectOrNUL(t *testing.T) {
	manifest := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: standards}
spec:
  version: 1.2.3
  areas: [{name: base, path: content}]
  exports: [{kind: Rule, namespace: base, name: basics}]
`)
	resource := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: basics, namespace: base}
spec:
  text: Apply the package basics.
  files: [content/ordinary.yaml]
`)
	malformedResource := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: hidden, namespace: base}
spec: {text: valid text, unexpected: true}
`)
	if _, err := Build(map[string][]byte{
		ManifestName:            manifest,
		"content/basics.yaml":   resource,
		"content/ordinary.yaml": malformedResource,
	}); err == nil {
		t.Fatal("Build accepted a malformed Markitect envelope as an ordinary input")
	}

	resource = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: basics, namespace: base}
spec:
  text: Apply the package basics.
  files: [docs/input.txt]
`)
	if _, err := Build(map[string][]byte{
		ManifestName:          manifest,
		"content/basics.yaml": resource,
		"docs/input.txt":      []byte("contains\x00nul"),
	}); err == nil {
		t.Fatal("Build accepted a declared UTF-8 input containing NUL")
	}
}

type zipMember struct {
	name string
	data []byte
	mode fs.FileMode
}

func makeZIP(t *testing.T, members []zipMember) []byte {
	t.Helper()
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	for _, item := range members {
		header := &zip.FileHeader{Name: item.name, Method: zip.Deflate}
		if item.mode != 0 {
			header.SetMode(item.mode)
		}
		member, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func corePin(name, version, digest string) core.PackagePin {
	return core.PackagePin{Name: name, Version: version, Source: "test", Archive: "packages/standards.zip", SHA256: digest}
}

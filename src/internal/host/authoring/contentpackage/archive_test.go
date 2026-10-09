package contentpackage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
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

func TestBuildReadPackageDeclaredDomainAndGenericExport(t *testing.T) {
	files := map[string][]byte{
		ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: software-model}
spec:
  version: 1.0.0
  areas: [{name: content, path: content}]
  domains: [domains/software.yaml]
  exports:
    - apiVersion: software.markitect.org/v1alpha1
      kind: Module
      namespace: platform
      name: payments
`),
		"domains/software.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: software}
spec:
  apiVersion: software.markitect.org/v1alpha1
  kinds:
    Module:
      required: [intent]
      properties:
        intent: {type: string}
        dependencies: {type: array, items: {type: ref, refKind: Module}}
  relations:
    dependsOn: {field: dependencies, sourceKinds: [Module], targetKinds: [Module], context: true, invalidate: true, acyclic: true}
`),
		"content/payments.yaml": []byte(`apiVersion: software.markitect.org/v1alpha1
kind: Module
metadata: {name: payments, namespace: platform}
spec: {intent: Owns payment flow, dependencies: []}
`),
	}
	archive, err := Build(files)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	got, err := Read(corePin("software-model", "1.0.0", hex.EncodeToString(digest[:])), archive)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DomainDefinitions) != 1 || got.DomainDefinitions["domains/software.yaml"].APIVersion != "software.markitect.org/v1alpha1" {
		t.Fatalf("unexpected package domains: %#v", got.DomainDefinitions)
	}
	if len(got.Resources) != 1 || got.Resources[0].TypeKey() != "software.markitect.org/v1alpha1/Module" {
		t.Fatalf("generic package resource was not parsed: %#v", got.Resources)
	}
}

func TestBuildRequiresDeclaredPackageDomainAndOmitsUnlistedDomain(t *testing.T) {
	manifest := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: software-model}
spec:
  version: 1.0.0
  areas: [{name: content, path: content}]
  domains: [domains/software.yaml]
  exports: [{kind: Rule, namespace: platform, name: basics}]
`)
	resource := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: basics, namespace: platform}
spec: {text: A local package resource.}
`)
	if _, err := Build(map[string][]byte{ManifestName: manifest, "content/basics.yaml": resource}); err == nil {
		t.Fatal("Build accepted a missing declared domain definition")
	}
	files := map[string][]byte{
		ManifestName:            manifest,
		"content/basics.yaml":   resource,
		"domains/software.yaml": []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Domain\nmetadata: {name: software}\nspec:\n  apiVersion: software.markitect.org/v1alpha1\n  kinds:\n    Module:\n      properties:\n        intent: {type: string}\n"),
		"domains/unlisted.yaml": []byte("unlisted member"),
	}
	archive, err := Build(files)
	if err != nil {
		t.Fatalf("Build rejected unrelated snapshot content: %v", err)
	}
	digest := sha256.Sum256(archive)
	got, err := Read(corePin("software-model", "1.0.0", hex.EncodeToString(digest[:])), archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, included := got.Files["domains/unlisted.yaml"]; included {
		t.Fatal("Build included a domain definition that the package manifest did not declare")
	}
}

func TestBuildRejectsConflictingDomainVersionAndUnqualifiedExport(t *testing.T) {
	definition := func(property string) []byte {
		return []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Domain\nmetadata: {name: software}\nspec:\n  apiVersion: software.markitect.org/v1alpha1\n  kinds:\n    Module:\n      properties:\n        " + property + ": {type: string}\n")
	}
	resource := []byte(`apiVersion: software.markitect.org/v1alpha1
kind: Module
metadata: {name: payments, namespace: platform}
spec: {intent: Owns payment flow}
`)
	base := map[string][]byte{
		ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: software-model}
spec:
  version: 1.0.0
  areas: [{name: content, path: content}]
  domains: [domains/software.yaml, domains/other.yaml]
  exports:
    - apiVersion: software.markitect.org/v1alpha1
      kind: Module
      namespace: platform
      name: payments
`),
		"domains/software.yaml": definition("intent"),
		"domains/other.yaml":    definition("purpose"),
		"content/payments.yaml": resource,
	}
	if _, err := Build(base); err == nil {
		t.Fatal("Build accepted conflicting definitions for one active domain version")
	}

	manifest := bytes.Replace(base[ManifestName], []byte("  domains: [domains/software.yaml, domains/other.yaml]\n"), []byte("  domains: [domains/software.yaml]\n"), 1)
	manifest = bytes.Replace(manifest, []byte("    - apiVersion: software.markitect.org/v1alpha1\n      kind:"), []byte("    - kind:"), 1)
	base[ManifestName] = manifest
	delete(base, "domains/other.yaml")
	if _, err := Build(base); err == nil {
		t.Fatal("Build accepted an unqualified export for a custom domain resource")
	}
}

func TestBuildDoesNotDowngradeMalformedDeclaredGenericResourceToInput(t *testing.T) {
	files := map[string][]byte{
		ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: software-model}
spec:
  version: 1.0.0
  areas: [{name: content, path: content}]
  domains: [domains/software.yaml]
  exports:
    - apiVersion: software.markitect.org/v1alpha1
      kind: Module
      namespace: platform
      name: payments
`),
		"domains/software.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata: {name: software}
spec:
  apiVersion: software.markitect.org/v1alpha1
  kinds:
    Module:
      required: [intent, files]
      properties:
        intent: {type: string}
        files: {type: array, items: {type: string}}
`),
		"content/payments.yaml": []byte(`apiVersion: software.markitect.org/v1alpha1
kind: Module
metadata: {name: payments, namespace: platform}
spec:
  intent: Owns payment flow
  files: [content/input.yaml]
`),
		"content/input.yaml": []byte(`apiVersion: software.markitect.org/v1alpha1
kind: UnknownKind
metadata: {name: hidden, namespace: platform}
spec: {unrecognized: resource}
`),
	}
	if _, err := Build(files); err == nil {
		t.Fatal("Build downgraded a malformed generic resource envelope to an ordinary declared input")
	}
}

func TestReadRejectsArchiveTamperingAgainstPin(t *testing.T) {
	archive, err := Build(map[string][]byte{
		ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: standards}
spec: {version: 1.2.3, areas: [{name: base, path: content}], exports: [{kind: Rule, namespace: base, name: basics}]}
`),
		"content/basics.yaml": []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Rule\nmetadata: {name: basics, namespace: base}\nspec: {text: Apply the package basics.}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	archive[len(archive)-1] ^= 0xff
	if _, err := Read(corePin("standards", "1.2.3", hex.EncodeToString(digest[:])), archive); err == nil {
		t.Fatal("Read accepted bytes that differ from the pinned archive")
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

func corePin(name, version, digest string) authoring.PackagePin {
	return authoring.PackagePin{Name: name, Version: version, Source: "test", Archive: "packages/standards.zip", SHA256: digest}
}

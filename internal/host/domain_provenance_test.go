package host

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring/contentpackage"
)

func TestModelJoinsEachPolicyToItsExactDomainSource(t *testing.T) {
	s := domainInputSnapshot()
	s.Files["domains/software.yaml"] = append(s.Files["domains/software.yaml"], []byte(`  constraints:
    - name: declared-intent
      select: {kind: Module}
      assert: {op: present, field: intent}
`)...)
	packageDefinition := strings.Replace(string(s.Files["domains/software.yaml"]), "software.markitect.io/v1alpha1", "packaged.markitect.io/v1alpha1", 1)
	packageDefinition = strings.Replace(packageDefinition, "name: software", "name: packaged-software", 1)
	packageFiles := map[string][]byte{
		"domains/software.yaml": []byte(packageDefinition),
		"resources/contract.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata: {name: contract, namespace: engineering}
spec: {text: Package-owned architecture contract.}
`),
		contentpackage.ManifestName: []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: architecture}
spec:
  version: 1.0.0
  domains: [domains/software.yaml]
  areas: [{name: engineering, path: resources}]
  exports: [{kind: Text, name: contract, namespace: engineering}]
`),
	}
	archive, err := contentpackage.Build(packageFiles)
	if err != nil {
		t.Fatal(err)
	}
	pin := core.PackagePin{Name: "architecture", Version: "1.0.0", Source: "fixture:architecture-v1", Archive: "packages/architecture.zip", SHA256: strings.TrimPrefix(Hash(archive), "sha256:")}
	config := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "consumer"}, Spec: core.Spec{
		Domains:  []string{"domains/software.yaml", "package:architecture/domains/software.yaml"},
		Packages: []core.PackagePin{pin}, Areas: []core.Area{{Name: "engineering", Path: "resources"}},
	}}
	s.Files["markitect.yaml"] = encodeResource(t, config)
	s.Files[pin.Archive] = archive
	s.Files["resources/packaged-module.yaml"] = []byte(strings.Replace(string(s.Files["resources/module.yaml"]), "software.markitect.io/v1alpha1", "packaged.markitect.io/v1alpha1", 1))
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("valid local and packaged Domains: %+v", p.Diagnostics)
	}
	model, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.DomainInputs) != 2 || len(model.PolicyResults) != 2 {
		t.Fatalf("missing provenance/results: %+v", model)
	}
	for _, result := range model.PolicyResults {
		var origin *ModelDomainInput
		for index := range model.DomainInputs {
			if model.DomainInputs[index].APIVersion == result.APIVersion {
				if origin != nil {
					t.Fatal("ambiguous Domain source")
				}
				origin = &model.DomainInputs[index]
			}
		}
		if origin == nil || origin.Path != "domains/software.yaml" {
			t.Fatalf("policy has no exact origin: %+v", result)
		}
		definitions, policies := 0, 0
		for _, domain := range model.Domains {
			if domain.APIVersion != origin.APIVersion || domain.Name != origin.Name {
				continue
			}
			definitions++
			for _, constraint := range domain.Constraints {
				if constraint.Name == result.Constraint {
					policies++
				}
			}
		}
		if definitions != 1 || policies != 1 {
			t.Fatalf("policy/source join lost its normalized definition: %+v", result)
		}
		if result.APIVersion == "packaged.markitect.io/v1alpha1" {
			if origin.Name != "packaged-software" || origin.Package != pin.Name || origin.PackageVersion != pin.Version || origin.Digest != Hash(packageFiles[origin.Path]) {
				t.Fatalf("package provenance lost: %+v", origin)
			}
		} else if origin.Name != "software" || origin.Package != "" || origin.PackageVersion != "" || origin.Digest != Hash(s.Files[origin.Path]) {
			t.Fatalf("local provenance lost: %+v", origin)
		}
	}
	// Provenance stays bound to the selected bytes, even if normalized schema
	// and policy semantics are unchanged by a source comment.
	updated := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
	for name, data := range s.Files {
		updated.Files[name] = append([]byte(nil), data...)
	}
	updated.Files["domains/software.yaml"] = append([]byte("# source review\n"), updated.Files["domains/software.yaml"]...)
	changed, err := Parse(updated)
	if err != nil {
		t.Fatal(err)
	}
	changedModel, err := CompileModel(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedModel.ModelDigest == model.ModelDigest {
		t.Fatal("source change reused old model identity")
	}
}

package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring/contentpackage"
)

func TestDomainInsideAreaRequiresSelectionBeforeFormatting(t *testing.T) {
	for _, selected := range []bool{false, true} {
		name := "unselected"
		if selected {
			name = "selected"
		}
		t.Run(name, func(t *testing.T) {
			s := domainInputSnapshot()
			domainPath := "resources/software.domain.yaml"
			s.Files[domainPath] = s.Files["domains/software.yaml"]
			delete(s.Files, "domains/software.yaml")
			config := strings.Replace(string(s.Files["markitect.yaml"]), "domains/software.yaml", domainPath, 1)
			if !selected {
				config = strings.Replace(config, "  domains: ["+domainPath+"]\n", "", 1)
			}
			s.Files["markitect.yaml"] = []byte(config)
			root := tempRoot(t)
			initWriterRepo(t, root)
			writeFixture(t, root, s.Files)
			p, err := Load(root, "")
			if err != nil {
				t.Fatal(err)
			}
			fullPath := filepath.Join(root, filepath.FromSlash(domainPath))
			before := mustRead(t, fullPath)
			if !selected {
				found := false
				for _, diagnostic := range p.Diagnostics {
					if diagnostic.Path == domainPath && diagnostic.Code == "parse" {
						found = true
					}
				}
				if !found {
					t.Fatalf("unselected definition became an ordinary resource: %+v", p.Diagnostics)
				}
				if _, err := Format(root, p, true); err == nil {
					t.Fatal("format must refuse an unselected Domain definition")
				}
				if string(mustRead(t, fullPath)) != string(before) {
					t.Fatal("format modified an unselected Domain definition")
				}
				return
			}
			if len(p.Diagnostics) != 0 {
				t.Fatalf("selected Area definition failed: %+v", p.Diagnostics)
			}
			if _, err := Format(root, p, true); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatal(err)
			}
			definition, err := authoring.ParseDomain(domainPath, after)
			if err != nil || definition.Kinds["Module"].Properties["intent"].Type != "string" {
				t.Fatalf("format lost Domain schema: definition=%+v err=%v", definition, err)
			}
			formatted, err := Load(root, "")
			if err != nil || len(formatted.Diagnostics) != 0 {
				t.Fatalf("formatted Domain no longer compiles: project=%+v err=%v", formatted, err)
			}
		})
	}
}

const appDomainDefinition = `apiVersion: markitect.example.org/v1alpha1
kind: Domain
metadata:
  name: software
spec:
  apiVersion: software.markitect.io/v1alpha1
  kinds:
    Module:
      properties:
        intent:
          type: string
      required: [intent]
`

func domainInputSnapshot() *snapshot.Snapshot {
	return &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{
		"markitect.yaml": []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata:
  name: engineering
spec:
  domains: [domains/software.yaml]
  areas:
    - name: engineering
      path: resources
`),
		"domains/software.yaml": []byte(appDomainDefinition),
		"resources/module.yaml": []byte(`apiVersion: software.markitect.io/v1alpha1
kind: Module
metadata:
  name: survey
  namespace: engineering
spec:
  intent: Collect responses.
`),
	}, Modes: map[string]string{}}
}

func TestDomainOutsideAreasIsLoadedAndCapturedInContext(t *testing.T) {
	s := domainInputSnapshot()
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", p.Diagnostics)
	}
	var key string
	for _, r := range p.Resources {
		if r.Kind == "Module" {
			key = r.GraphKey()
		}
	}
	if key == "" {
		t.Fatal("custom Module was not compiled")
	}
	before, err := CompileContext(p, key, "0.10.0", "fixed-tool")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, input := range before.Inputs {
		if input.Role == "domain" && input.Path == "domains/software.yaml" {
			found = true
			if input.Hash != Hash(s.Files[input.Path]) || input.Text != string(s.Files[input.Path]) {
				t.Fatal("domain evidence does not identify exact source bytes")
			}
		}
	}
	if !found {
		t.Fatal("selected language definition missing from compiled context")
	}
	// Even unchanged schema semantics cannot reuse evidence for different source
	// bytes; the original context must remain bound to its captured value.
	changed := domainInputSnapshot()
	changed.Files["domains/software.yaml"] = append([]byte("# reviewed definition\n"), changed.Files["domains/software.yaml"]...)
	updated, err := Parse(changed)
	if err != nil {
		t.Fatal(err)
	}
	after, err := CompileContext(updated, key, "0.10.0", "fixed-tool")
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest == after.Digest {
		t.Fatal("changed definition bytes reused old context fingerprint")
	}
	if before.Inputs[len(before.Inputs)-1].Text == after.Inputs[len(after.Inputs)-1].Text {
		t.Fatal("captured context was mutated by new definition")
	}
}

func TestDomainSelectionRejectsMissingUnsafeAndDuplicateInputs(t *testing.T) {
	for _, selected := range []string{"missing.yaml", "../software.yaml", "domains/software.yaml, domains/software.yaml", "package:missing/domains/software.yaml"} {
		t.Run(selected, func(t *testing.T) {
			s := domainInputSnapshot()
			s.Files["markitect.yaml"] = []byte(strings.Replace(string(s.Files["markitect.yaml"]), "domains: [domains/software.yaml]", "domains: ["+selected+"]", 1))
			if _, err := Parse(s); err == nil {
				t.Fatal("invalid explicit definition selection was accepted")
			}
		})
	}
}

func TestMalformedDeclaredCustomResourceCannotBecomeOrdinaryInput(t *testing.T) {
	s := domainInputSnapshot()
	s.Files["resources/module.yaml"] = []byte(strings.Replace(string(s.Files["resources/module.yaml"]), "intent: Collect responses.", "undeclared: hidden", 1))
	s.Files["resources/consumer.yaml"] = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata:
  name: consumer
  namespace: engineering
spec:
  text: Review the module.
  files: [resources/module.yaml]
`)
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range p.Diagnostics {
		if d.Code == "parse" && d.Path == "resources/module.yaml" {
			found = true
		}
	}
	if !found {
		t.Fatalf("failed typed custom resource hidden as an opaque input: %+v", p.Diagnostics)
	}
}

func TestCustomAPIIdentityAndFindFilter(t *testing.T) {
	s := domainInputSnapshot()
	s.Files["domains/delivery.yaml"] = []byte(strings.Replace(strings.Replace(appDomainDefinition, "name: software", "name: delivery", 1), "software.markitect.io/v1alpha1", "delivery.markitect.io/v1alpha1", 1))
	s.Files["markitect.yaml"] = []byte(strings.Replace(string(s.Files["markitect.yaml"]), "domains: [domains/software.yaml]", "domains: [domains/software.yaml, domains/delivery.yaml]", 1))
	s.Files["resources/delivery.yaml"] = []byte(strings.Replace(string(s.Files["resources/module.yaml"]), "software.markitect.io/v1alpha1", "delivery.markitect.io/v1alpha1", 1))
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("distinct domain kinds collided: %+v", p.Diagnostics)
	}
	matches, err := Find(p, FindQuery{APIVersion: "software.markitect.io/v1alpha1", Kind: "Module"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].APIVersion != "software.markitect.io/v1alpha1" {
		t.Fatalf("qualified discovery: %+v", matches)
	}
}

func TestPinnedPackageDomainRequiresActivationAndKeepsExactOrigin(t *testing.T) {
	packageFiles := domainInputSnapshot().Files
	delete(packageFiles, "markitect.yaml")
	packageFiles[contentpackage.ManifestName] = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Package
metadata: {name: software-model}
spec:
  version: 1.0.0
  domains: [domains/software.yaml]
  areas: [{name: engineering, path: resources}]
  exports:
    - apiVersion: software.markitect.io/v1alpha1
      kind: Module
      name: survey
      namespace: engineering
`)
	archive, err := contentpackage.Build(packageFiles)
	if err != nil {
		t.Fatal(err)
	}
	pin := authoring.PackagePin{Name: "software-model", Version: "1.0.0", Source: "urn:example:software-model:1.0.0", Archive: "packages/software-model.zip", SHA256: strings.TrimPrefix(Hash(archive), "sha256:")}
	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "consumer"}}, Spec: authoring.Spec{Packages: []authoring.PackagePin{pin}}}
	s := &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{"markitect.yaml": encodeResource(t, project), pin.Archive: archive}, Modes: map[string]string{}}
	unselected, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(unselected.DomainInputs) != 0 || len(unselected.Diagnostics) == 0 {
		t.Fatal("package silently activated its language and policy in the consumer")
	}
	project.Spec.Domains = []string{"package:software-model/domains/software.yaml"}
	selectedSnapshot := &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{"markitect.yaml": encodeResource(t, project), pin.Archive: archive}, Modes: map[string]string{}}
	selected, err := Parse(selectedSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Diagnostics) != 0 {
		t.Fatalf("explicit imported definition produced diagnostics: %+v", selected.Diagnostics)
	}
	var entry string
	for _, resource := range selected.Resources {
		if resource.Kind == "Module" {
			entry = resource.GraphKey()
		}
	}
	context, err := CompileContext(selected, entry, "0.10.0", "fixed-tool")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range context.Inputs {
		if input.Role == "domain" && input.Package == pin.Name {
			if input.Path != "domains/software.yaml" || input.PackageVersion != pin.Version || input.Hash != Hash(packageFiles[input.Path]) || input.Text != string(packageFiles[input.Path]) {
				t.Fatalf("imported domain source lost exact origin: %+v", input)
			}
			return
		}
	}
	t.Fatal("activated package definition missing from bounded context")
}

func TestConstraintSelectorMembershipInvalidatesWithoutInventoryChange(t *testing.T) {
	beforeSnapshot := domainInputSnapshot()
	beforeSnapshot.Files["domains/software.yaml"] = append(beforeSnapshot.Files["domains/software.yaml"], []byte(`  constraints:
    - name: governed-inventory
      select: {kind: Module, labels: {governed: ""}}
      assert: {op: count, min: 0, max: 3}
`)...)
	beforeSnapshot.Files["resources/module.yaml"] = []byte(strings.Replace(string(beforeSnapshot.Files["resources/module.yaml"]), "  namespace: engineering", "  namespace: engineering\n  labels: {governed: \"\"}", 1))
	beforeSnapshot.Files["resources/audit.yaml"] = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Text
metadata: {name: audit, namespace: engineering}
spec: {text: Review policy results.}
`)
	afterSnapshot := &snapshot.Snapshot{Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	for name, data := range beforeSnapshot.Files {
		afterSnapshot.Files[name] = append([]byte(nil), data...)
	}
	afterSnapshot.Files["resources/module.yaml"] = []byte(strings.Replace(string(afterSnapshot.Files["resources/module.yaml"]), "  labels: {governed: \"\"}\n", "", 1))
	before, err := Parse(beforeSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Parse(afterSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Diagnostics) != 0 || len(after.Diagnostics) != 0 {
		t.Fatalf("selector snapshots must both be valid: before=%+v after=%+v", before.Diagnostics, after.Diagnostics)
	}
	if len(before.Graph.Resources) != len(after.Graph.Resources) {
		t.Fatal("test unexpectedly changed inventory")
	}
	for _, resource := range after.Resources {
		if resource.Kind == "Module" && constraintReadsResource(after, resource) {
			t.Fatal("absent label matched a selector for a present empty value")
		}
	}
	impact := Changes(before, after)
	if len(impact.Affected) != len(after.Graph.Resources) {
		t.Fatalf("selector membership change reused unrelated evidence: %+v", impact)
	}
}

func TestGenericResourceInheritsBuiltinAreaPolicy(t *testing.T) {
	s := domainInputSnapshot()
	s.Files["markitect.yaml"] = []byte(strings.Replace(string(s.Files["markitect.yaml"]), "      path: resources", "      path: resources\n      rules: [{name: policy}]", 1))
	s.Files["resources/policy.yaml"] = []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Rule
metadata: {name: policy, namespace: engineering}
spec: {text: Review the declared engineering intent.}
`)
	p, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("generic domain lost inherited AI policy: %+v", p.Diagnostics)
	}
	var key string
	for _, resource := range p.Resources {
		if resource.Kind == "Module" {
			key = resource.GraphKey()
		}
	}
	policy := "engineering/Rule/policy"
	if len(p.Graph.Edges[key]) != 1 || p.Graph.Edges[key][0] != policy || len(p.Graph.InvalidationEdges[key]) != 1 || p.Graph.InvalidationEdges[key][0] != policy {
		t.Fatalf("inherited policy effects missing: context=%v invalidation=%v", p.Graph.Edges[key], p.Graph.InvalidationEdges[key])
	}
}

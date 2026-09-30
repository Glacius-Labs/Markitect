package app

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/contentpackage"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func packageSourceFixture(t *testing.T, text string) map[string][]byte {
	t.Helper()
	manifest := core.Resource{APIVersion: core.APIVersion, Kind: "Package", Metadata: core.Metadata{Name: "review-kit"}, Spec: core.Spec{
		Version: "1.0.0", Areas: []core.Area{{Name: "shared", Path: "docs"}},
		Exports: []core.Ref{{Namespace: "shared", Kind: "Workflow", Name: "review"}},
	}}
	flow := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Namespace: "shared", Name: "review"}, Spec: core.Spec{
		Text: "Review the declared input.", Uses: []core.Ref{{Kind: "Text", Name: "notes"}},
	}}
	notes := core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Namespace: "shared", Name: "notes"}, Spec: core.Spec{
		Text: text, Files: []string{"docs/input.txt"},
	}}
	return map[string][]byte{
		"markitect-package.yaml": encodeResource(t, manifest), "docs/review.yaml": encodeResource(t, flow),
		"docs/notes.yaml": encodeResource(t, notes), "docs/input.txt": []byte("Archive input, not the local file."),
	}
}

func packageConsumerFixture(t *testing.T, text string) *Project {
	t.Helper()
	archive, err := contentpackage.Build(packageSourceFixture(t, text))
	if err != nil {
		t.Fatal(err)
	}
	pin := core.PackagePin{Name: "review-kit", Version: "1.0.0", Source: "urn:example:review-kit:1.0.0", Archive: "packages/review-kit.zip", SHA256: strings.TrimPrefix(Hash(archive), "sha256:")}
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "consumer"}, Spec: core.Spec{
		Packages: []core.PackagePin{pin}, Areas: []core.Area{{Name: "shared", Path: "docs"}}, Targets: []string{"codex"},
	}}
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Namespace: "shared", Name: "entry"}, Spec: core.Spec{
		Text: "Use the reviewed package workflow.", Description: "Review an input.", Uses: []core.Ref{{Package: "review-kit", Namespace: "shared", Kind: "Workflow", Name: "review"}},
	}}
	localNotes := core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Namespace: "shared", Name: "notes"}, Spec: core.Spec{Text: "Unrelated local resource."}}
	files := map[string][]byte{
		"markitect.yaml": encodeResource(t, project), pin.Archive: archive,
		"docs/entry.yaml": encodeResource(t, skill), "docs/notes.yaml": encodeResource(t, localNotes),
		"docs/input.txt": []byte("Unrelated local file."),
	}
	snapshot := &source.Snapshot{Revision: strings.Repeat("a", 40), Files: files, Modes: map[string]string{}}
	for name := range files {
		snapshot.Modes[name] = "100644"
	}
	p, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("package fixture diagnostics: %#v", p.Diagnostics)
	}
	outputs, err := render.Generate(p.Graph)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		snapshot.Files[name] = data
		snapshot.Modes[name] = "100644"
	}
	return p
}

func TestPackageContextKeepsOriginsAndPrivateDependencies(t *testing.T) {
	p := packageConsumerFixture(t, "Private package notes.")
	before := p.Snapshot.Digest()
	context, err := CompileContext(p, "shared/Skill/entry", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]ContextInput{}
	for _, input := range context.Inputs {
		got[input.Key] = input
	}
	for _, key := range []string{"/Project/consumer", "shared/Skill/entry", "review-kit::/Package/review-kit", "review-kit::shared/Workflow/review", "review-kit::shared/Text/notes", "package:review-kit/file:docs/input.txt"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing selected input %s", key)
		}
	}
	if _, ok := got["shared/Text/notes"]; ok {
		t.Fatal("same local resource name leaked into package closure")
	}
	input := got["package:review-kit/file:docs/input.txt"]
	if input.Text != "Archive input, not the local file." || input.Package != "review-kit" || input.PackageVersion != "1.0.0" {
		t.Fatalf("package input lacks exact provenance: %#v", input)
	}
	if p.Snapshot.Digest() != before {
		t.Fatal("package parsing/context modified physical snapshot")
	}
	if _, ok := p.Snapshot.Files["docs/review.yaml"]; ok {
		t.Fatal("archive member became a writable physical source")
	}
	if _, err := CompileContext(p, "review-kit::shared/Text/notes", "test"); err == nil {
		t.Fatal("private package entry selectable directly")
	}
	if _, err := CompileContext(p, "review-kit::shared/Workflow/review", "test"); err != nil {
		t.Fatal(err)
	}
	matches, err := Find(p, FindQuery{Package: "review-kit"})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Key != "review-kit::shared/Workflow/review" {
		t.Fatalf("package find exports: %#v", matches)
	}
	if _, err := Explain(p, "review-kit::shared/Text/notes"); err == nil {
		t.Fatal("private package entry explain succeeded")
	}
	if len(CheckOutputs(p)) != 0 {
		t.Fatalf("local output validation failed: %#v", CheckOutputs(p))
	}
}

func TestPackageUpdateInvalidatesReviewAndReportsPhysicalChange(t *testing.T) {
	before := packageConsumerFixture(t, "Initial package notes.")
	after := packageConsumerFixture(t, "Revised package notes.")
	after.Snapshot.Revision = strings.Repeat("b", 40)
	config := ReviewConfig{Question: "Does the entry include its review procedure?", PromptVersion: "v1", Model: "test", Effort: "high", AllowReuse: true}
	record, err := RecordReview(before, "shared/Skill/entry", "test", "tool", config, "Fixture advisory report.")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := ReuseReview(before, before, record, "test", "tool", config)
	if err != nil || unchanged.Status != ReviewReusable {
		t.Fatalf("unchanged reuse: %#v, %v", unchanged, err)
	}
	changed, err := ReuseReview(before, after, record, "test", "tool", config)
	if err != nil || changed.Status != ReviewRequired {
		t.Fatalf("changed reuse: %#v, %v", changed, err)
	}
	impact := Changes(before, after)
	if !contains(impact.Changed, "packages/review-kit.zip") || !contains(impact.Affected, "shared/Skill/entry") || !contains(impact.Affected, "review-kit::shared/Workflow/review") {
		t.Fatalf("package change not propagated: %#v", impact)
	}
	for _, name := range impact.Changed {
		if name == "docs/notes.yaml" {
			t.Fatal("archive member misreported as changed local path")
		}
	}
}

func TestPackageDigestAndOriginBoundaryFailClosed(t *testing.T) {
	p := packageConsumerFixture(t, "Package notes.")
	p.Snapshot.Files["packages/review-kit.zip"] = append(p.Snapshot.Files["packages/review-kit.zip"], 'x')
	if _, err := Parse(p.Snapshot); err == nil {
		t.Fatal("changed archive accepted under old digest")
	}
}

func TestPackContentUsesFixedSnapshotAndIncludesOnlyDeclaredInputs(t *testing.T) {
	files := packageSourceFixture(t, "Package notes.")
	files["private.txt"] = []byte("not a package input")
	files["docs/review.md"] = []byte("generated or ordinary view is not canonical package input")
	snapshot := &source.Snapshot{Revision: strings.Repeat("c", 40), Files: files}
	first, pin, err := PackContent(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second, pin2, err := PackContent(snapshot)
	if err != nil || !bytes.Equal(first, second) || !reflect.DeepEqual(pin, pin2) {
		t.Fatalf("pack not deterministic: %v", err)
	}
	archive, err := contentpackage.Read(pin, first)
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.Files) != 4 {
		t.Fatalf("packaged undeclared files: %v", sortedFiles(archive.Files))
	}
	snapshot.Provisional = true
	if _, _, err := PackContent(snapshot); err == nil {
		t.Fatal("provisional pack accepted")
	}
	snapshot.Provisional = false
	bad := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "shared"}, Spec: core.Spec{Text: "Missing dependency.", Uses: []core.Ref{{Kind: "Text", Name: "missing"}}}}
	files["docs/review.yaml"], err = format.Encode(bad)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := PackContent(snapshot); err == nil {
		t.Fatal("pack accepted unresolved dependency")
	}
}

func TestPackContentResolvesPrivatePackageBinding(t *testing.T) {
	files := packageSourceFixture(t, "Review notes.")
	manifest, err := format.Parse("markitect-package.yaml", files["markitect-package.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	contractRef := core.Ref{Kind: "Contract", Namespace: "shared", Name: "assessment"}
	implementationRef := core.Ref{Kind: "Workflow", Namespace: "shared", Name: "assessor"}
	manifest.Spec.Bindings = []core.Binding{{Contract: contractRef, Implementation: implementationRef}}
	files[manifest.Path] = encodeResource(t, *manifest)
	entry, err := format.Parse("docs/review.yaml", files["docs/review.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	entry.Spec.Needs = []core.Ref{contractRef}
	files[entry.Path] = encodeResource(t, *entry)
	contract := core.Resource{APIVersion: core.APIVersion, Kind: "Contract", Metadata: core.Metadata{Namespace: "shared", Name: "assessment"}, Spec: core.Spec{Text: "Assess an input.", Kind: "Workflow", Input: []string{"change"}, Output: []string{"findings"}}}
	implementation := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Namespace: "shared", Name: "assessor"}, Spec: core.Spec{Text: "Assess the supplied change.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{contractRef}}}
	files["docs/assessment.yaml"] = encodeResource(t, contract)
	files["docs/assessor.yaml"] = encodeResource(t, implementation)
	archive, pin, err := PackContent(&source.Snapshot{Revision: strings.Repeat("c", 40), Files: files})
	if err != nil {
		t.Fatal(err)
	}
	config := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "consumer"}, Spec: core.Spec{Packages: []core.PackagePin{pin}}}
	p, err := Parse(&source.Snapshot{Revision: strings.Repeat("d", 40), Files: map[string][]byte{"markitect.yaml": encodeResource(t, config), pin.Archive: archive}})
	if err != nil {
		t.Fatal(err)
	}
	context, err := CompileContext(p, "review-kit::shared/Workflow/review", "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range context.Inputs {
		if input.Key == "review-kit::shared/Workflow/assessor" {
			return
		}
	}
	t.Fatal("package context omitted its selected private implementation")
}

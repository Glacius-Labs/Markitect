package organization_test

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestRenderExactUTF8ByteCapAndCompleteDiagnostics(t *testing.T) {
	selection, settings, input := renderFixture(t, "a", organization.ManagerView)
	input.RemainingLimits = []organization.HostText{hostFact("limit-text", "\xef\xbb\xbfOriginal é子\x00\r\nRepeated é子\r\n")}
	input.ParentHints = []organization.ParentHint{{Package: organization.Root, Advice: hostFact("hint", "A source-attributed hint, not a new obligation.")}}
	var exact organization.ContextPackage
	// The cap is itself rendered. Reach its byte-length fixed point, rather
	// than assuming an external token estimate or ignoring its own digits.
	for range 8 {
		exact = requireRendered(t, selection, settings, input)
		length := int64(len(exact.Bytes()))
		if input.MaxContextBytes.Value == length {
			break
		}
		input.MaxContextBytes.Value = length
	}
	length := int64(len(exact.Bytes()))
	if length != input.MaxContextBytes.Value || !utf8.Valid(exact.Bytes()) || int64(utf8.RuneCount(exact.Bytes())) >= length {
		t.Fatal("not an exact UTF8 byte boundary")
	}
	if !bytes.Equal(sectionTexts(t, exact, "remaining-limit-view-not-accounting")[0], []byte(input.RemainingLimits[0].Text)) {
		t.Fatal("Host text bytes were normalized")
	}
	input.MaxContextBytes.Value--
	refusal := requireRenderRefusal(t, selection, settings, input, "context-too-large")
	if refusal.RequiredBytes != length || refusal.LimitBytes != length-1 || refusal.LimitSource != input.MaxContextBytes.Source {
		t.Fatalf("incorrect complete cap diagnostic: %+v", refusal)
	}
	var total int64
	for _, section := range refusal.Sections {
		total += section.Bytes
	}
	if total != length || len(refusal.Sections) != len(exact.Sections()) || refusal.Sections[len(refusal.Sections)-1].Name != "original-custom-instruction" {
		t.Fatal("cap ignored frames, templates or trailing original instructions")
	}
	input.MaxContextBytes.Value++
	if restored := requireRendered(t, selection, settings, input); !bytes.Equal(restored.Bytes(), exact.Bytes()) {
		t.Fatal("refusal mutated fixed input/settings")
	}
}

func TestRenderSourceManifestAndProductHashes(t *testing.T) {
	selection, settings, input := renderFixture(t, "a", organization.ManagerView)
	artifact := renderArtifact(t, "a")
	artifact.Content = organization.ArtifactBinaryRef
	artifact.Text = nil
	input.OwnArtifacts = []organization.ArtifactFact{artifact}
	result := requireRendered(t, selection, settings, input)
	raw := result.Bytes()
	if result.ContentID() != snapshot.ContentID(sha256.Sum256(raw)) || result.Format().ID != "markitect/context-text" || result.Format().ContentID != snapshot.ContentID(sha256.Sum256([]byte("\n[section %q bytes=%d]\n\n[/section]\n"))) {
		t.Fatal("payload/framing identity does not bind exact bytes")
	}
	role := sectionTexts(t, result, "role")[0]
	mandate := sectionTexts(t, result, "mandate")[0]
	if result.Template().ContentID != snapshot.ContentID(sha256.Sum256(append(append([]byte(nil), role...), mandate...))) {
		t.Fatal("template identity omits mandate or role bytes")
	}
	seen := make(map[organization.RenderSourceKind]bool)
	var end int64
	for _, section := range result.Sections() {
		if section.Start != end || section.TextStart <= section.Start || section.TextEnd > section.End || len(section.Sources) == 0 {
			t.Fatalf("manifest gap or missing payload sources: %+v", section)
		}
		end = section.End
		for _, source := range section.Sources {
			seen[source.Kind] = true
			switch source.Kind {
			case organization.ModelRenderSource:
				if source.ModelID == "" || source.ModelDigest == "" || source.Model.Path == "" || source.Model.Digest == "" || source.Definition.ID == "" {
					t.Fatal("model provenance lost")
				}
			case organization.ConfigRenderSource:
				if source.Config.File == "" || source.Config.ContentID == (snapshot.ContentID{}) {
					t.Fatal("config provenance lost")
				}
			case organization.HostRenderSource:
				if source.Host.RecordID == "" || source.Host.ContentID == (snapshot.ContentID{}) {
					t.Fatal("Host provenance lost")
				}
			case organization.ProductRenderSource:
				if source.Product.Version == "" || source.Product.ID == "" {
					t.Fatal("product version lost")
				}
				if section.Name != "role" && section.Name != "mandate" && source.Product.ContentID != snapshot.ContentID(sha256.Sum256(raw[section.TextStart:section.TextEnd])) {
					t.Fatal("literal source hash identifies different product bytes")
				}
			default:
				t.Fatal("unknown source class")
			}
		}
		if section.Name == "task-identity" && section.Sources[0].Host != input.InputBinding {
			t.Fatal("task identity cited an unrelated text field")
		}
		if section.Name == "open-finding-identity" && section.Sources[0].Host != input.Task.Findings.Source {
			t.Fatal("finding identity cited grounds text instead of complete set")
		}
	}
	if end != int64(len(raw)) || len(seen) != 4 {
		t.Fatal("incomplete byte/source coverage")
	}
	manifest := result.Sections()
	manifest[len(manifest)-1].Sources[0].Config.Field = "mutated"
	if reflect.DeepEqual(manifest, result.Sections()) {
		t.Fatal("nested output sources alias internal manifest")
	}
}

func TestRenderOriginalRepeatedInstructionsAndFixedCopies(t *testing.T) {
	compiled := compileFixture(t, contextInput(t))
	topology := requireTopology(t, compiled)
	facts := settingsFixture(t)
	facts.Defaults.Manager.Instructions.Value = []string{"style", "empty", "style"}
	facts.Overrides = []organization.PackageOverride{{Package: "a", Manager: organization.RoleOverride{Instructions: settingPointer([]string{"style", "local"}, "/packages/a/instructions/manager")}}}
	settings := requireSettings(t, topology, facts)
	selection := requireContext(t, compiled, "a")
	_, _, input := renderFixture(t, "a", organization.ManagerView)
	artifact := renderArtifact(t, "a")
	input.OwnArtifacts = []organization.ArtifactFact{artifact}
	first := requireRendered(t, selection, settings, input)
	texts := sectionTexts(t, first, "original-custom-instruction")
	if len(texts) != 5 || !bytes.Equal(texts[0], facts.Instructions[0].Bytes) || len(texts[1]) != 0 || !bytes.Equal(texts[0], texts[2]) || !bytes.Equal(texts[0], texts[3]) || !bytes.Equal(texts[4], facts.Instructions[2].Bytes) {
		t.Fatal("instruction ordering, duplicates, empty content or original bytes changed")
	}
	var occurrences []organization.RenderSection
	for _, section := range first.Sections() {
		if section.Name == "original-custom-instruction" {
			occurrences = append(occurrences, section)
		}
	}
	if occurrences[0].Sources[0].Config != facts.Defaults.Manager.Instructions.Source || occurrences[3].Sources[0].Config != facts.Overrides[0].Manager.Instructions.Source || occurrences[0].Sources[1].Config != facts.Instructions[0].Source {
		t.Fatal("selection and original-file provenance conflated")
	}
	before := first.Bytes()
	originalManifest := first.Sections()
	facts.Instructions[0].Bytes[0] = '!'
	facts.Defaults.Manager.Instructions.Value[0] = "local"
	input.Task.Outcome.Text = "changed"
	*input.OwnArtifacts[0].Text = "changed"
	selected := selection.Definitions()
	selected[0].Definition.Body = "changed"
	returned := settings.Managers()
	returned[0].Manager.Instructions[0].Value.Bytes[0] = '!'
	if !bytes.Equal(first.Bytes(), before) || !reflect.DeepEqual(first.Sections(), originalManifest) {
		t.Fatal("prepared package aliases mutable inputs")
	}
	_, _, freshInput := renderFixture(t, "a", organization.ManagerView)
	freshInput.OwnArtifacts = []organization.ArtifactFact{renderArtifact(t, "a")}
	if again := requireRendered(t, selection, settings, freshInput); !bytes.Equal(again.Bytes(), before) || !reflect.DeepEqual(again.Sections(), originalManifest) {
		t.Fatal("fixed selection/settings or deterministic render were mutated")
	}
}

func TestRenderRefusesInvalidUTF8WithoutRepair(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*organization.RenderInput)
	}{
		{"host-text", func(i *organization.RenderInput) { i.Task.Outcome.Text = "invalid\xff" }},
		{"host-source", func(i *organization.RenderInput) { i.InputBinding.RecordID = "invalid\xff" }},
		{"host-metadata", func(i *organization.RenderInput) { i.Task.ID = "invalid\xff" }},
		{"config-metadata", func(i *organization.RenderInput) { i.MaxContextBytes.Source.Field = "invalid\xff" }},
		{"artifact-text", func(i *organization.RenderInput) {
			a := renderArtifact(t, "a")
			value := "invalid\xff"
			a.Text = &value
			a.ContentID = snapshot.ContentID(sha256.Sum256([]byte(value)))
			i.OwnArtifacts = []organization.ArtifactFact{a}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			selection, settings, input := renderFixture(t, "a", organization.ManagerView)
			test.mutate(&input)
			requireRenderRefusal(t, selection, settings, input, "invalid-text")
		})
	}
	compiled := compileFixture(t, contextInput(t))
	settings := requireSettings(t, requireTopology(t, compiled), settingsFixture(t))
	_, _, input := renderFixture(t, "a", organization.ManagerView)
	findCompiledDefinition(t, &compiled, contextID("a", "own")).Body = "invalid\xff"
	requireRenderRefusal(t, requireContext(t, compiled, "a"), settings, input, "invalid-text")
	facts := settingsFixture(t)
	facts.Instructions[0] = instruction("style", []byte("invalid\xff"))
	settings = requireSettings(t, requireTopology(t, compiled), facts)
	requireRenderRefusal(t, requireContext(t, compileFixture(t, contextInput(t)), "a"), settings, input, "invalid-text")
}

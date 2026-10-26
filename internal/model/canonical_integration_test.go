package model

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func cloneModelInput(input Input) Input {
	clone := func(source SourceModel) SourceModel {
		source.Entries = append([]Entry(nil), source.Entries...)
		for index := range source.Entries {
			source.Entries[index].Bytes = bytes.Clone(source.Entries[index].Bytes)
		}
		return source
	}
	input.Root = clone(input.Root)
	input.Dependencies = append([]SourceModel(nil), input.Dependencies...)
	for index := range input.Dependencies {
		input.Dependencies[index] = clone(input.Dependencies[index])
	}
	return input
}

func TestCanonicalSyntheticGoldenAndPublicTypedIR(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{field: text}")
	got := requireChecked(t, input)
	// Independently authored expected model-digest/1 tree, not a round-trip of
	// the encoder. Fixed Core bytes are separately checked against §8.3's vector.
	wanted := `{"format":"markitect-model-digest/1","model":{"dependencies":{"markitect-core":"1.0"},"formatVersion":1,"id":"project"},"packages":{"feature":{"Record":{"body":"Contract fixture.","bodyLinks":[],"kind":["reference","markitect/core","Kind"],"spec":{"implements":["list",[]],"properties":["list",[["reference","feature","field"]]]}},"field":{"body":"Contract fixture.","bodyLinks":[],"kind":["reference","markitect/core","Property"],"spec":{"maxCount":["integer","1"],"minCount":["integer","0"],"name":["string","field"],"valueType":["enum","string"]}},"item":{"body":"Contract fixture.","bodyLinks":[],"kind":["reference","feature","Record"],"spec":{"field":["string","text"]}}}}}`
	if string(got.Models[0].Canonical) != wanted {
		t.Fatalf("synthetic golden differs:\n got %s\nwant %s", got.Models[0].Canonical, wanted)
	}
	var item *Definition
	for index := range got.Models[0].Definitions {
		if got.Models[0].Definitions[index].Identity.ID == "item" {
			item = &got.Models[0].Definitions[index]
		}
	}
	if item == nil || item.Kind != (Identity{"feature", "Record"}) || len(item.Spec) != 1 || item.Spec[0].Value.Kind() != StringValue || item.Spec[0].Name != "field" || item.Source.Digest == "" {
		t.Fatalf("public IR is not resolved/typed/source-bound: %+v", item)
	}
}

func TestCanonicalSourceAndAuthorReferenceInvariance(t *testing.T) {
	input := referenceValueInput("{link: peer, links: [peer, item, peer], kindLink: Core.Kind}")
	baseline := requireChecked(t, input).Models[0]
	for _, tt := range []struct {
		name   string
		change func(*Input)
	}{
		{"source enumeration", func(i *Input) { slices.Reverse(i.Root.Entries); slices.Reverse(i.Dependencies) }},
		{"comments and positions", func(i *Input) {
			for index := range i.Root.Entries {
				i.Root.Entries[index].Bytes = append([]byte("# source comment\n\n"), i.Root.Entries[index].Bytes...)
			}
		}},
		{"full author addresses", func(i *Input) {
			i.Root.Entries[6] = contractDefinition("feature/item.yaml", "item", "feature/Record", "{kindLink: markitect/core/Kind, links: [feature/peer, feature/item, feature/peer], link: feature/peer}")
		}},
		{"alias spelling", func(i *Input) {
			for index := range i.Root.Entries {
				entry := &i.Root.Entries[index]
				entry.Bytes = bytes.ReplaceAll(entry.Bytes, []byte("Core."), []byte("Meta."))
				entry.Bytes = bytes.ReplaceAll(entry.Bytes, []byte("as: Core"), []byte("as: Meta"))
			}
		}},
		{"physical source relocation", func(i *Input) {
			i.Root.ManifestPath = "relocated/model.yaml"
			for index := range i.Root.Entries {
				i.Root.Entries[index].Path = "relocated/" + i.Root.Entries[index].Path
			}
		}},
		{"root relocation", func(i *Input) {
			i.Root.Entries[0].Bytes = bytes.Replace(i.Root.Entries[0].Bytes, []byte("root: model"), []byte("root: world"), 1)
			for index := range i.Root.Entries {
				i.Root.Entries[index].Path = strings.Replace(i.Root.Entries[index].Path, "model/", "world/", 1)
			}
		}},
		{"definition filename", func(i *Input) { i.Root.Entries[6].Path = "model/feature/renamed.yaml" }},
		{"empty packages and foreign bytes", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty/leaf", Kind: Directory}, Entry{Path: "model/feature/notes.md", Kind: Regular, Bytes: []byte("not a Definition")})
		}},
		{"explicit Property name default", func(i *Input) {
			entry := &i.Root.Entries[3]
			edited := bytes.Replace(entry.Bytes, []byte("valueType: reference"), []byte("name: link, valueType: reference"), 1)
			if bytes.Equal(edited, entry.Bytes) {
				panic("explicit Property name edit did not change the fixture")
			}
			entry.Bytes = edited
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			trial := cloneModelInput(input)
			tt.change(&trial)
			got := requireChecked(t, trial).Models[0]
			if !bytes.Equal(baseline.Canonical, got.Canonical) || baseline.Digest != got.Digest {
				t.Fatal("equivalent selected content changed canonical bytes")
			}
		})
	}
	for _, typeName := range []string{"string", "number", "reference", "object"} {
		extra := ""
		if typeName == "reference" {
			extra = ", target: Record"
		}
		a := requireChecked(t, singleValueInput(typeName, "0", "-1", extra, "{}")).Models[0]
		b := requireChecked(t, singleValueInput(typeName, "0", "-1", extra, "{field: []}")).Models[0]
		if !bytes.Equal(a.Canonical, b.Canonical) {
			t.Fatalf("missing/empty optional %s list differ", typeName)
		}
	}
}

func TestCanonicalTextValueOrderAndPresenceSensitivity(t *testing.T) {
	input := singleValueInput("string", "0", "-1", "", "{field: [a, b, a]}")
	baseline := requireChecked(t, input).Models[0]
	for _, tt := range []struct {
		name   string
		change func(*Input)
	}{
		{"list order", func(i *Input) {
			i.Root.Entries[3] = contractDefinition("feature/item.yaml", "item", "Record", "{field: [b, a, a]}")
		}},
		{"list repetition", func(i *Input) {
			i.Root.Entries[3] = contractDefinition("feature/item.yaml", "item", "Record", "{field: [a, b]}")
		}},
		{"literal body whitespace", func(i *Input) { i.Root.Entries[3] = withBody(i.Root.Entries[3], `"Contract fixture. "`) }},
		{"present empty title", func(i *Input) { i.Root.Entries[3].Bytes = append(i.Root.Entries[3].Bytes, []byte("\ntitle: ''\n")...) }},
		{"local version presence", func(i *Input) {
			i.Root.Entries[0].Bytes = append(i.Root.Entries[0].Bytes, []byte("version: '1.0'\n")...)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			trial := cloneModelInput(input)
			tt.change(&trial)
			got := requireChecked(t, trial).Models[0]
			if bytes.Equal(baseline.Canonical, got.Canonical) || baseline.Digest == got.Digest {
				t.Fatal("meaningful content change did not change bytes/digest")
			}
		})
	}
	a := requireChecked(t, singleValueInput("string", "1", "1", "", "{field: \"u\\u0308\"}")).Models[0]
	b := requireChecked(t, singleValueInput("string", "1", "1", "", "{field: ü}")).Models[0]
	if bytes.Equal(a.Canonical, b.Canonical) {
		t.Fatal("ordinary Unicode text was normalized")
	}
}

func TestCanonicalBodyLinkTargetsBindUnchangedBody(t *testing.T) {
	input := referenceValueInput("{}")
	for _, pkg := range []string{"first", "second"} {
		input.Root.Entries = append(input.Root.Entries, addFileImport(contractDefinition(pkg+"/peer.yaml", "peer", "feature/Record", "{}"), "feature", "Feature"))
	}
	input.Root.Entries[6] = withBody(addFileImport(input.Root.Entries[6], "first", "Refs"), "'[[Refs.peer]] [[Refs.peer]]'")
	a := requireChecked(t, input).Models[0]
	input.Root.Entries[6].Bytes = bytes.Replace(input.Root.Entries[6].Bytes, []byte("package: first"), []byte("package: second"), 1)
	b := requireChecked(t, input).Models[0]
	if bytes.Equal(a.Canonical, b.Canonical) || a.Digest == b.Digest {
		t.Fatal("same authored body lost changed resolved link targets")
	}
	for _, definition := range b.Definitions {
		if definition.Identity == (Identity{"feature", "item"}) && (len(definition.BodyLinks) != 2 || definition.BodyLinks[0].Target.Package != "second" || definition.BodyLinks[1].Target.Package != "second") {
			t.Fatal("public body link order/repetition/targets changed")
		}
	}
}

func TestForeignContentDigestsAndLockBytesAreExcluded(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', support: '1.0'}"), 1)
	support := bindingSource("support", "1.0", "{markitect-core: '1.0'}", contractDefinition("support/record.yaml", "Record", "Core.Kind", "{}"))
	bind := func() {
		digest := privateCanonical(t, Input{Root: support, Dependencies: []SourceModel{completeCoreSource()}}).digest
		input.Root = withLock(input.Root, coreLockText()+"  support: {version: '1.0', digest: '"+digest+"'}\n", true)
		input.Dependencies = append(input.Dependencies[:1], support)
	}
	bind()
	a := requireChecked(t, input).Models[0]
	support.Entries[1] = withBody(support.Entries[1], "'Changed foreign description.'")
	bind()
	b := requireChecked(t, input).Models[0]
	if !bytes.Equal(a.Canonical, b.Canonical) || a.Digest != b.Digest {
		t.Fatal("foreign content digest or lock bytes leaked into own projection")
	}
}

func TestUnavailableCoordinatesDoNotBlockPublicCanonicalContent(t *testing.T) {
	input := referenceValueInput("{}")
	input.Root.Entries[6] = withBody(input.Root.Entries[6], "'[[peer]]'")
	baseline := requireChecked(t, input)
	var scopes []sourceScope
	var sources []ModelResult
	for _, model := range append([]SourceModel{input.Root}, input.Dependencies...) {
		result, _, scope := checkSource(model, DefaultLimits())
		scopes = append(scopes, scope)
		sources = append(sources, result)
	}
	for index := range scopes[0].definitions {
		definition := &scopes[0].definitions[index]
		if definition.identity == (Identity{"feature", "item"}) {
			for field := range definition.envelope.fields {
				if definition.envelope.fields[field].Name == "body" {
					definition.envelope.fields[field].Value.position = Position{}
				}
			}
		}
	}
	result := Result{Models: append([]ModelResult(nil), sources...), Limits: DefaultLimits()}
	finalizeModels(&result, scopes, sources)
	if result.Status != Passed || result.Coordinates != NotCompleted || !hasDiagnostic(result, "coordinate.yaml_mapping") || !bytes.Equal(result.Models[0].Canonical, baseline.Models[0].Canonical) || result.Models[0].Digest != baseline.Models[0].Digest {
		t.Fatalf("optional coordinate proof changed canonical validity: status=%s diagnostics=%+v", result.Status, result.Diagnostics)
	}
	for _, definition := range result.Models[0].Definitions {
		if definition.Identity == (Identity{"feature", "item"}) && (len(definition.BodyLinks) != 1 || definition.BodyLinks[0].Coordinates.Source != nil || definition.BodyLinks[0].Coordinates.Body == nil) {
			t.Fatal("missing metadata erased a recognized bound occurrence")
		}
	}
}

func TestCanonicalReportsAreEnumerationIndependentAndInputsUnchanged(t *testing.T) {
	input := referenceValueInput("{link: peer}")
	original := cloneModelInput(input)
	a := requireChecked(t, input)
	if !reflect.DeepEqual(input, original) {
		t.Fatal("Model check changed input source bytes")
	}
	slices.Reverse(input.Root.Entries)
	slices.Reverse(input.Dependencies)
	b := requireChecked(t, input)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("source enumeration changed full checked result")
	}
}

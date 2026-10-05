package dotnet

import (
	"bytes"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func fixture() (core.Schema, []core.Definition, string) {
	kind := core.KindIdentity{APIVersion: "example.test/v1", Kind: "Widget"}
	schema := core.Schema{APIVersion: kind.APIVersion, Purpose: "Application-owned model.", Kinds: map[string]core.Kind{
		kind.Kind: {Purpose: "A project-specific deployable unit."},
	}}
	definitions := []core.Definition{
		{APIVersion: kind.APIVersion, Kind: kind.Kind, Metadata: core.Metadata{Namespace: "apps", Name: "one"}, Purpose: "First unit."},
		{APIVersion: kind.APIVersion, Kind: kind.Kind, Metadata: core.Metadata{Namespace: "apps", Name: "two"}, Purpose: "Second unit."},
	}
	return schema, definitions, kind.Key()
}

func TestEvaluateAcceptsUnknownProjectKindCandidatePendingHostChecks(t *testing.T) {
	schema, definitions, key := fixture()
	input := Input{
		Definitions: definitions, Schemas: []core.Schema{schema},
		Guidance:       map[string]string{key: "Keep deployment configuration in the project-owned adapter."},
		AllowedPaths:   []string{"src/Widgets.cs"},
		CandidateFiles: map[string][]byte{"src/Widgets.cs": []byte("namespace App;\n")},
	}
	result := Evaluate(input)
	if result.Status != CandidateUnverified || len(result.Files) != 1 || len(result.Escalations) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestEvaluateMissingGuidanceEscalatesWithoutReturningAnyFiles(t *testing.T) {
	schema, definitions, _ := fixture()
	result := Evaluate(Input{
		Definitions: definitions, Schemas: []core.Schema{schema},
		AllowedPaths:   []string{"src/Widgets.cs"},
		CandidateFiles: map[string][]byte{"src/Widgets.cs": []byte("candidate")},
	})
	if len(result.Escalations) == 0 || len(result.Files) != 0 || result.Status != "" {
		t.Fatalf("missing guidance should stop acceptance: %#v", result)
	}
}

func TestEvaluateDoesNotAssumeOneArtifactPerDefinitionAndAcceptsVariants(t *testing.T) {
	schema, definitions, key := fixture()
	base := Input{
		Definitions: definitions, Schemas: []core.Schema{schema},
		Guidance:     map[string]string{key: "Use the supplied project convention."},
		AllowedPaths: []string{"src/Widgets.cs", "src/App.csproj"},
	}
	base.CandidateFiles = map[string][]byte{"src/Widgets.cs": []byte("public class Widgets {}\n")}
	first := Evaluate(base)
	if len(first.Files) != 1 || first.Status != CandidateUnverified {
		t.Fatalf("one file may represent multiple Definitions: %#v", first)
	}
	base.CandidateFiles = map[string][]byte{
		"src/Widgets.cs": []byte("public sealed class Widgets {}\n"),
		"src/App.csproj": []byte("<Project />\n"),
	}
	second := Evaluate(base)
	if len(second.Files) != 2 || second.Status != CandidateUnverified {
		t.Fatalf("multiple files may represent a scope: %#v", second)
	}
	if bytes.Equal(first.Files["src/Widgets.cs"], second.Files["src/Widgets.cs"]) {
		t.Fatal("test variants must contain different candidate bytes")
	}
}

func TestEvaluateEmptyCandidateIsIncompleteAndUnownedPathIsRejected(t *testing.T) {
	schema, definitions, key := fixture()
	base := Input{Definitions: definitions, Schemas: []core.Schema{schema}, Guidance: map[string]string{key: "Project guidance."}, AllowedPaths: []string{"src/Widgets.cs"}}
	if result := Evaluate(base); len(result.Diagnostics) == 0 || len(result.Files) != 0 || result.Status != "" {
		t.Fatalf("empty candidate should remain incomplete: %#v", result)
	}
	base.CandidateFiles = map[string][]byte{"src/Other.cs": []byte("class Other {}")}
	if result := Evaluate(base); len(result.Diagnostics) == 0 || len(result.Files) != 0 {
		t.Fatalf("unowned path should be rejected: %#v", result)
	}
}

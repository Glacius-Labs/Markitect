package model

import (
	"reflect"
	"strings"
	"testing"
)

func bindingScope(id, version string, requirements ...modelRequirement) sourceScope {
	return sourceScope{
		origin: id + "-origin", modelID: id,
		manifest: Entry{Path: "model.yaml", Kind: Regular, Bytes: []byte("manifest:" + id)},
		version:  Value{kind: StringValue, text: version}, versionDeclared: version != "",
		requirements: requirements, manifestReady: true, inventoryComplete: true,
	}
}

func req(id, version string) modelRequirement {
	return modelRequirement{id: id, version: Value{kind: StringValue, text: version}}
}

func findingCodes(g dependencyGraph) []string {
	codes := make([]string, len(g.findings))
	for i, finding := range g.findings {
		codes[i] = finding.diagnostic.Code
	}
	return codes
}

func containsCode(codes []string, code string) bool {
	for _, candidate := range codes {
		if candidate == code {
			return true
		}
	}
	return false
}

func TestBindDependenciesExactVersionsAndDiamond(t *testing.T) {
	scopes := []sourceScope{
		bindingScope("local", "", req("left", "1.x"), req("right", "2")),
		bindingScope("left", "1.x", req("shared", "v-1")),
		bindingScope("right", "2", req("shared", "v-1")),
		bindingScope("shared", "v-1"),
	}
	g := bindDependencies(scopes)
	if len(g.findings) != 0 || !reflect.DeepEqual(g.status, []Status{Passed, Passed, Passed, Passed}) {
		t.Fatalf("expected complete diamond graph: %+v", g)
	}
	if !reflect.DeepEqual(g.direct, [][]int{{1, 2}, {3}, {3}, nil}) || !reflect.DeepEqual(g.edges, g.direct) {
		t.Fatalf("unexpected bound graph: edges=%v direct=%v", g.edges, g.direct)
	}
	// Version labels are literal strings; x has no wildcard behavior.
	scopes[0].requirements[0] = req("left", "1.0")
	g = bindDependencies(scopes)
	if !reflect.DeepEqual(findingCodes(g), []string{"dependency.version_mismatch"}) || g.status[0] != Failed {
		t.Fatalf("1.0 must not satisfy 1.x: %+v", g)
	}
	if !reflect.DeepEqual(g.edges[0], []int{1, 2}) || len(g.direct[0]) != 1 || g.direct[0][0] != 2 {
		t.Fatalf("version mismatch must retain logical edge and exclude that direct binding: %+v", g)
	}
}

func TestBindDependenciesMissingVersionAndExactSpelling(t *testing.T) {
	t.Run("missing model", func(t *testing.T) {
		g := bindDependencies([]sourceScope{bindingScope("local", "", req("absent", "1"))})
		if !reflect.DeepEqual(findingCodes(g), []string{"dependency.missing"}) || g.status[0] != Failed {
			t.Fatalf("missing model not diagnosed: %+v", g)
		}
	})
	t.Run("unversioned supplied model", func(t *testing.T) {
		g := bindDependencies([]sourceScope{bindingScope("local", "", req("library", "1")), bindingScope("library", "")})
		if !reflect.DeepEqual(findingCodes(g), []string{"dependency.provided_version"}) || g.status[0] != Failed {
			t.Fatalf("unversioned provided model not diagnosed: %+v", g)
		}
	})
	t.Run("wrong author spelling", func(t *testing.T) {
		g := bindDependencies([]sourceScope{bindingScope("local", "", req("strasse", "1")), bindingScope("Straße", "1")})
		if !reflect.DeepEqual(findingCodes(g), []string{"dependency.spelling"}) {
			t.Fatalf("expected exact spelling diagnostic: %+v", g)
		}
	})
	t.Run("ambiguous collision includes exact candidate", func(t *testing.T) {
		g := bindDependencies([]sourceScope{bindingScope("local", "", req("strasse", "1")), bindingScope("straße", "1"), bindingScope("strasse", "1")})
		if len(g.edges[0]) != 0 || !containsCode(findingCodes(g), "dependency.ambiguous") {
			t.Fatalf("candidate order or exact match selected a winner: %+v", g)
		}
	})
	t.Run("absence unproven with unknown supplied identity", func(t *testing.T) {
		unknown := bindingScope("", "1")
		g := bindDependencies([]sourceScope{bindingScope("local", "", req("absent", "1")), unknown})
		if len(g.findings) != 0 || g.status[0] != NotCompleted {
			t.Fatalf("incomplete identity set proved false absence: %+v", g)
		}
	})
}

func TestBindDependenciesFindsActualCycleMembers(t *testing.T) {
	tests := []struct {
		name   string
		scopes []sourceScope
		cycle  []int
	}{
		{"self", []sourceScope{bindingScope("self", "1", req("self", "1"))}, []int{0}},
		{"mutual", []sourceScope{bindingScope("a", "1", req("b", "1")), bindingScope("b", "1", req("a", "1"))}, []int{0, 1}},
		{"long cycle excludes tail", []sourceScope{bindingScope("a", "1", req("b", "1")), bindingScope("b", "1", req("c", "1")), bindingScope("c", "1", req("a", "1")), bindingScope("tail", "1", req("a", "1"))}, []int{0, 1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := bindDependencies(tt.scopes)
			if len(g.findings) != len(tt.cycle) {
				t.Fatalf("cycle findings: %+v", g.findings)
			}
			for _, index := range tt.cycle {
				if g.status[index] != Failed {
					t.Fatalf("cycle member %d not failed: %v", index, g.status)
				}
			}
			for index := range tt.scopes {
				if !containsInt(tt.cycle, index) && g.status[index] != Failed {
					t.Fatalf("tail should inherit dependency failure: %v", g.status)
				}
			}
			for _, finding := range g.findings {
				if finding.diagnostic.Code != "dependency.cycle" {
					t.Fatalf("unexpected finding: %+v", finding)
				}
			}
		})
	}
}

func TestCycleRemainsKnownBesideIncompleteMetadata(t *testing.T) {
	scopes := []sourceScope{
		bindingScope("a", "1", req("b", "1")),
		bindingScope("b", "1", req("a", "1")),
		bindingScope("partial", "1", req("not-supplied", "1")),
	}
	scopes[2].modelID = ""
	g := bindDependencies(scopes)
	if !reflect.DeepEqual(g.status, []Status{Failed, Failed, NotCompleted}) {
		t.Fatalf("known errors/incomplete scope were conflated: %+v", g)
	}
	if !strings.Contains(g.findings[0].diagnostic.Message, "a") {
		t.Fatalf("cycle finding missing sorted identity: %+v", g.findings)
	}
}

func TestDependencyVersionProofRequiresCompleteMetadata(t *testing.T) {
	t.Run("unknown provided manifest is incomplete", func(t *testing.T) {
		local := bindingScope("local", "", req("library", "1"))
		partial := bindingScope("library", "")
		partial.manifestReady = false
		partial.inventoryComplete = false
		g := bindDependencies([]sourceScope{local, partial})
		if len(g.findings) != 0 || g.status[0] != NotCompleted || g.status[1] != NotCompleted {
			t.Fatalf("unknown manifest was treated as a known omitted version: %+v", g)
		}
	})
	t.Run("known unversioned local target is invalid", func(t *testing.T) {
		local := bindingScope("local", "")
		consumer := bindingScope("consumer", "1", req("local", "1"))
		g := bindDependencies([]sourceScope{local, consumer})
		if !reflect.DeepEqual(findingCodes(g), []string{"dependency.target_version"}) || g.status[0] != Failed || g.status[1] != Failed {
			t.Fatalf("complete unversioned target did not fail its binding closure: %+v", g)
		}
	})
}

func TestUnusedDuplicateModelIdentitiesFailGraphStatus(t *testing.T) {
	g := bindDependencies([]sourceScope{
		bindingScope("local", ""),
		bindingScope("strasse", "1"),
		bindingScope("straße", "2"),
	})
	if g.status[0] != Passed || g.status[1] != Failed || g.status[2] != Failed {
		t.Fatalf("duplicate identities were hidden without a requirement edge: %+v", g)
	}
	if !reflect.DeepEqual(findingCodes(g), []string{"dependency.duplicate_model", "dependency.duplicate_model"}) {
		t.Fatalf("missing duplicate identity findings: %+v", g.findings)
	}
}

func containsInt(values []int, value int) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

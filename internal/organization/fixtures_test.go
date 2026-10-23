package organization_test

import (
	"fmt"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

func emptyLock() model.Entry {
	return model.Entry{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nmodels: {}\n")}
}

func checkFixture(t *testing.T, input model.Input) model.Result {
	t.Helper()
	result := model.Check(input)
	if result.Status != model.Passed {
		t.Fatalf("fixture did not pass the Model check: status=%s diagnostics=%+v", result.Status, result.Diagnostics)
	}
	return result
}

// localFixture supplies canonical source bytes, including explicit empty
// directories. Its vocabulary and local Definitions are deliberately small;
// it is not the library's full business or realization contract.
func localFixture(t *testing.T, definitions map[string][]string, emptyDirectories ...string) model.Input {
	t.Helper()
	core := coreSource()
	coreResult := checkFixture(t, model.Input{Root: core})
	coreDigest := coreResult.Models[0].Digest
	software := model.SourceModel{
		Origin: "fixture/software", ManifestPath: "model.yaml",
		Entries: []model.Entry{
			{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: software-basics\nversion: '0.1'\nroot: model\ndependencies: {markitect-core: '1.0'}\n")},
			{Path: "model/software/concept.yaml", Kind: model.Regular, Bytes: []byte("imports: [{package: markitect/core, as: Core}]\nid: Concept\nkind: Core.Kind\nbody: Topology fixture vocabulary.\nspec: {}\n")},
			{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte(fmt.Sprintf("formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0', digest: '%s'}\n", coreDigest))},
		},
	}
	softwareResult := checkFixture(t, model.Input{Root: software, Dependencies: []model.SourceModel{core}})
	softwareDigest := softwareResult.Models[0].Digest
	local := model.SourceModel{
		Origin: "fixture/local", ManifestPath: "model.yaml",
		Entries: []model.Entry{
			{Path: "model.yaml", Kind: model.Regular, Bytes: []byte("formatVersion: 1\nid: local-project\nroot: model\ndependencies: {markitect-core: '1.0', software-basics: '0.1'}\n")},
			{Path: "model", Kind: model.Directory},
			{Path: "model-lock.yaml", Kind: model.Regular, Bytes: []byte(fmt.Sprintf("formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0', digest: '%s'}\n  software-basics: {version: '0.1', digest: '%s'}\n", coreDigest, softwareDigest))},
		},
	}
	for pkg, names := range definitions {
		for _, name := range names {
			local.Entries = append(local.Entries, model.Entry{
				Path: "model/" + pkg + "/" + name + ".yaml", Kind: model.Regular,
				Bytes: []byte(fmt.Sprintf("imports: [{package: software}]\nid: %s\nkind: software/Concept\nbody: Topology fixture object.\nspec: {}\n", name)),
			})
		}
	}
	for _, name := range emptyDirectories {
		local.Entries = append(local.Entries, model.Entry{Path: "model/" + name, Kind: model.Directory})
	}
	return model.Input{Root: local, Dependencies: []model.SourceModel{software, core}}
}

func copyResult(result model.Result) model.Result {
	result.Models = append([]model.ModelResult(nil), result.Models...)
	for i := range result.Models {
		result.Models[i].Packages = append([]model.Package(nil), result.Models[i].Packages...)
		result.Models[i].Definitions = append([]model.Definition(nil), result.Models[i].Definitions...)
	}
	return result
}

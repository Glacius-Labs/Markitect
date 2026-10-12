package host

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"github.com/Glacius-Labs/Markitect/src/internal/host/releaseinstall"
)

func TestInstalledPinsOutsideDeclaredAreaAreNotTypedResources(t *testing.T) {
	root := installTestRepo(t, "feature/area-pins")
	project := projectResource(projectNS)
	project.Spec.Areas = []authoring.Area{{Name: "engineering", Path: ".markitect/areas/engineering"}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion,
		Kind:     "Rule",
		Metadata: core.Metadata{Name: "deployment", Namespace: "engineering"}}, Spec: authoring.Spec{Text: "Follow the deployment process."},
	}
	writeFixture(t, root, map[string][]byte{
		"markitect.yaml": encodeResource(t, project),
		".markitect/areas/engineering/deployment.yaml": encodeResource(t, rule),
	})
	commitInstallPins(t, root, "consumer project with a hidden control-plane area")
	if _, err := releaseinstall.Install(root, installTestBundle(t, "f", "0.9.0", "area-pins"), true); err != nil {
		t.Fatalf("install pinned tool: %v", err)
	}

	installed, err := Load(root, "")
	if err != nil {
		t.Fatalf("load installed project: %v", err)
	}
	if len(installed.Diagnostics) != 0 {
		t.Fatalf("installed project diagnostics: %#v", installed.Diagnostics)
	}
	if len(installed.Resources) != 2 || len(installed.Inventory) != 2 {
		t.Fatalf("installed project parsed unexpected resources: resources=%#v inventory=%#v", installed.Resources, installed.Inventory)
	}
	for _, entry := range installed.Inventory {
		if entry.Path == ".markitect/tool/lock.yaml" || entry.Path == ".markitect/tool/release.yaml" {
			t.Fatalf("pinned distribution metadata became a typed resource: %#v", entry)
		}
	}
	if got := installed.Graph.Resources["engineering/Rule/deployment"]; got == nil || got.Path != ".markitect/areas/engineering/deployment.yaml" {
		t.Fatalf("declared hidden area resource was not retained: %#v", got)
	}
}

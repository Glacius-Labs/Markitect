package app

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/contentpackage"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// PackContent validates a closed content package from one fixed source tree.
// The returned pin is a suggestion: consumers choose the vendored archive path
// and distribution provenance after obtaining the exact bytes from their owner.
// The caller supplies provenance; the compiler does not infer a source system.
func PackContent(s *snapshot.Snapshot, provenance string) ([]byte, core.PackagePin, error) {
	var pin core.PackagePin
	if s == nil || s.Provisional || !validSnapshotID(s.ID) {
		return nil, pin, fmt.Errorf("pack requires a fixed identified snapshot")
	}
	if !validSnapshotID(provenance) {
		return nil, pin, fmt.Errorf("package provenance is required")
	}
	manifest, err := format.Parse("markitect-package.yaml", s.Files["markitect-package.yaml"])
	if err != nil {
		return nil, pin, err
	}
	if manifest.Kind != "Package" {
		return nil, pin, fmt.Errorf("markitect-package.yaml must contain a Package")
	}
	archive, err := contentpackage.Build(s.Files)
	if err != nil {
		return nil, pin, err
	}
	pin = core.PackagePin{
		Name: manifest.Metadata.Name, Version: manifest.Spec.Version,
		Source:  provenance,
		Archive: ".markitect/packages/" + manifest.Metadata.Name + "-" + manifest.Spec.Version + ".zip",
		SHA256:  strings.TrimPrefix(Hash(archive), "sha256:"),
	}
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "package-validation"}, Spec: core.Spec{Packages: []core.PackagePin{pin}}}
	config, err := format.Encode(project)
	if err != nil {
		return nil, pin, err
	}
	validation := &snapshot.Snapshot{
		ID: s.ID, Files: map[string][]byte{"markitect.yaml": config, pin.Archive: archive},
		Modes: map[string]string{"markitect.yaml": "100644", pin.Archive: "100644"},
	}
	p, err := Parse(validation)
	if err != nil {
		return nil, pin, err
	}
	if len(p.Diagnostics) != 0 {
		return nil, pin, fmt.Errorf("package graph is invalid: %s: %s", p.Diagnostics[0].Code, p.Diagnostics[0].Message)
	}
	return archive, pin, nil
}

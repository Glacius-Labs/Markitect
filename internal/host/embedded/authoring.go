// Package authoring exposes Markitect's portable, tool-shipped authoring
// guidance as a regular compiled Markitect context.
package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
)

// resources contains the canonical core Project and authoring resources. It is
// included in the release source archive and compiled into the tool binary.
//
//go:embed project.yaml resources/*.yaml
var resources embed.FS

const entry = "core/Skill/authoring"

// Context parses and validates the embedded authoring resources and compiles
// the authoring Skill's explicit dependency closure. These bytes are immutable
// parts of the Markitect release; the result is not a consumer checkout.
func Context(version, toolDigest string) (*host.Context, error) {
	if strings.TrimSpace(version) == "" {
		return nil, fmt.Errorf("authoring context requires a tool version")
	}
	snapshot := &snapshot.Snapshot{
		ID:    "embedded",
		Files: make(map[string][]byte),
		Modes: make(map[string]string),
	}
	projectBytes, err := resources.ReadFile("project.yaml")
	if err != nil {
		return nil, fmt.Errorf("load embedded authoring project: %w", err)
	}
	snapshot.Files["markitect.yaml"] = projectBytes
	snapshot.Modes["markitect.yaml"] = "100644"
	err = fs.WalkDir(resources, "resources", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(path.Ext(name), ".yaml") || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected embedded authoring resource %q", name)
		}
		data, err := resources.ReadFile(name)
		if err != nil {
			return err
		}
		relative := "internal/host/embedded/" + name
		snapshot.Files[relative] = data
		snapshot.Modes[relative] = "100644"
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load embedded authoring resources: %w", err)
	}
	project, err := host.Parse(snapshot)
	if err != nil {
		return nil, fmt.Errorf("parse embedded authoring project: %w", err)
	}
	if len(project.Diagnostics) != 0 {
		return nil, fmt.Errorf("embedded authoring project has %d diagnostics: %s", len(project.Diagnostics), project.Diagnostics[0].Message)
	}
	compiled, err := host.CompileContext(project, entry, version, toolDigest)
	if err != nil {
		return nil, fmt.Errorf("compile embedded authoring Skill: %w", err)
	}
	return compiled, nil
}

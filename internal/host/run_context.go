package host

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/format"
	"go.yaml.in/yaml/v3"
)

const RunManifestVersion = "markitect.example.org/context-run/v1alpha1"
const maxRunSources = 64
const maxRunSelectedBytes = 16 << 20

// RunManifest selects one entry, task snapshot and bounded exact source files
// from a single immutable Git revision.
type RunManifest struct {
	Version string               `yaml:"version"`
	Entry   string               `yaml:"entry"`
	Task    RunTaskSelection     `yaml:"task"`
	Sources []RunSourceSelection `yaml:"sources,omitempty"`
}

type RunTaskSelection struct {
	ID   string `yaml:"id"`
	Path string `yaml:"path"`
}

type RunSourceSelection struct {
	Path     string `yaml:"path"`
	Reason   string `yaml:"reason"`
	Optional bool   `yaml:"optional,omitempty"`
}

type RunContextEvidence struct {
	ManifestPath  string `yaml:"manifestPath"`
	ManifestHash  string `yaml:"manifestHash"`
	TaskID        string `yaml:"taskId"`
	TaskPath      string `yaml:"taskPath"`
	SelectionHash string `yaml:"selectionHash"`
}

// ParseRunManifest strictly parses and validates the small run manifest.
func ParseRunManifest(data []byte) (*RunManifest, error) {
	if len(data) == 0 || len(data) > 64<<10 {
		return nil, fmt.Errorf("run manifest must be between 1 byte and 64 KiB")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var manifest RunManifest
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid run manifest: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("run manifest must contain exactly one YAML document")
	} else if err != io.EOF {
		return nil, fmt.Errorf("invalid run manifest: %w", err)
	}
	if manifest.Version != RunManifestVersion {
		return nil, fmt.Errorf("run manifest version must be %q", RunManifestVersion)
	}
	if strings.TrimSpace(manifest.Entry) == "" {
		return nil, fmt.Errorf("run manifest entry is required")
	}
	if strings.TrimSpace(manifest.Task.ID) == "" {
		return nil, fmt.Errorf("run manifest task.id is required")
	}
	if err := validateRunPath(manifest.Task.Path); err != nil {
		return nil, fmt.Errorf("run manifest task path: %w", err)
	}
	if len(manifest.Sources) > maxRunSources {
		return nil, fmt.Errorf("run manifest supports at most %d exact source paths", maxRunSources)
	}
	seen := map[string]bool{manifest.Task.Path: true}
	for i, input := range manifest.Sources {
		if err := validateRunPath(input.Path); err != nil {
			return nil, fmt.Errorf("run manifest sources[%d].path: %w", i, err)
		}
		if seen[input.Path] {
			return nil, fmt.Errorf("run manifest repeats selected path %q", input.Path)
		}
		seen[input.Path] = true
		if strings.TrimSpace(input.Reason) == "" {
			return nil, fmt.Errorf("run manifest sources[%d].reason is required", i)
		}
	}
	return &manifest, nil
}

func validateRunPath(value string) error {
	if value == "" || strings.Contains(value, "\\") || strings.Contains(value, ":") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") {
		return fmt.Errorf("%q must be a normalized repository-relative slash path", value)
	}
	return nil
}

// CompileRunContext combines the normal entry closure with explicit run inputs.
// The manifest must itself be present at manifestPath in the selected snapshot.
func CompileRunContext(p *Project, manifestPath string, manifestBytes []byte, version, toolDigest string) (*Context, error) {
	if p == nil || p.Snapshot == nil {
		return nil, fmt.Errorf("project and snapshot are required")
	}
	if err := validateRunPath(manifestPath); err != nil {
		return nil, fmt.Errorf("run manifest path: %w", err)
	}
	snapshotManifest, exists := p.Snapshot.Files[manifestPath]
	if !exists || !bytes.Equal(snapshotManifest, manifestBytes) {
		return nil, fmt.Errorf("run manifest %q must match the selected Git snapshot", manifestPath)
	}
	manifest, err := ParseRunManifest(manifestBytes)
	if err != nil {
		return nil, err
	}
	if p.Snapshot.Provisional || p.Snapshot.ID == "" {
		return nil, fmt.Errorf("run context requires an immutable Git snapshot")
	}
	selectedBytes := 0
	for _, selectedPath := range append([]string{manifest.Task.Path}, selectedSourcePaths(manifest.Sources)...) {
		selectedBytes += len(p.Snapshot.Files[selectedPath])
	}
	if selectedBytes > maxRunSelectedBytes {
		return nil, fmt.Errorf("run context selected inputs exceed the 16 MiB limit")
	}
	c, err := CompileContext(p, manifest.Entry, version, toolDigest)
	if err != nil {
		return nil, err
	}
	c.Status = "complete"
	c.Complete = true
	c.Run = &RunContextEvidence{ManifestPath: manifestPath, ManifestHash: Hash(manifestBytes), TaskID: manifest.Task.ID, TaskPath: manifest.Task.Path}
	selection := strings.Builder{}
	fmt.Fprintf(&selection, "revision:%d:%s\x00entry:%d:%s\x00manifest:%d:%s\x00", len(p.Snapshot.ID), p.Snapshot.ID, len(manifest.Entry), manifest.Entry, len(manifestPath), manifestPath)
	writeSelectionField(&selection, manifest.Task.ID)
	addSelectedRunInput := func(role, name, reason string, required bool) bool {
		data, found := p.Snapshot.Files[name]
		status := "included"
		hash := ""
		if found {
			hash = Hash(data)
			if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
				status = "invalid-text"
				found = false
				c.Complete = false
				c.Status = "incomplete"
			}
		} else {
			status = "missing"
		}
		text := ""
		if found {
			text = string(data)
		}
		c.Inputs = append(c.Inputs, ContextInput{Key: role + ":" + name, Path: name, Hash: hash, Reason: reason, Text: text, Role: role, Status: status, Required: required})
		writeSelectionField(&selection, role)
		writeSelectionField(&selection, name)
		writeSelectionField(&selection, reason)
		writeSelectionField(&selection, status)
		writeSelectionField(&selection, fmt.Sprint(required))
		writeSelectionField(&selection, hash)
		selection.WriteByte(0)
		if !found && required {
			c.Complete = false
			c.Status = "incomplete"
		}
		return found
	}
	addSelectedRunInput("manifest", manifestPath, "run input selection", true)
	addSelectedRunInput("task", manifest.Task.Path, "work-item snapshot", true)
	for _, source := range manifest.Sources {
		addSelectedRunInput("source", source.Path, source.Reason, !source.Optional)
	}
	c.Run.SelectionHash = Hash([]byte(selection.String()))
	// Selection identity affects reuse even when a selected optional source is absent.
	c.Digest = Hash([]byte(c.Digest + "\x00run:" + c.Run.SelectionHash))
	sort.SliceStable(c.Inputs, func(i, j int) bool {
		if c.Inputs[i].Role == "" && c.Inputs[j].Role != "" {
			return true
		}
		if c.Inputs[i].Role != "" && c.Inputs[j].Role == "" {
			return false
		}
		if c.Inputs[i].Role != c.Inputs[j].Role {
			return c.Inputs[i].Role < c.Inputs[j].Role
		}
		return c.Inputs[i].Path < c.Inputs[j].Path
	})
	return c, nil
}

func selectedSourcePaths(sources []RunSourceSelection) []string {
	paths := make([]string, 0, len(sources))
	for _, source := range sources {
		paths = append(paths, source.Path)
	}
	return paths
}

func writeSelectionField(builder *strings.Builder, value string) {
	fmt.Fprintf(builder, "%d:%s", len(value), value)
}

// EncodeRunManifest returns canonical YAML for a validated manifest.
func EncodeRunManifest(manifest *RunManifest) ([]byte, error) { return format.Encode(manifest) }

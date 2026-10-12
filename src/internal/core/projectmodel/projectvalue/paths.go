package projectvalue

import (
	"bytes"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectcoverage"
)

// DocumentPath preserves the old hidden view for existing configurations.
// Newly initialized projects explicitly select the readable docs destination.
func DocumentPath(config Config) string {
	if config.DocumentPath != "" {
		return config.DocumentPath
	}
	return ViewPath
}

// IsGeneratedDocument recognizes only the existing marker at byte zero.
func IsGeneratedDocument(data []byte) bool {
	return bytes.HasPrefix(data, []byte(GeneratedViewMarker+"\n"))
}

// ToolPaths declares exact paths and directory prefixes owned by Markitect.
func ToolPaths(config Config) []projectcoverage.ToolPath {
	tools := []projectcoverage.ToolPath{
		{Selector: ManifestPath, Owner: "projectwork"},
		{Selector: RuntimePath, Owner: "projectwork"},
		{Selector: ".markitect/.gitignore", Owner: "projectwork"},
		{Selector: ".markitect/runs/", Owner: "projectrun", Operational: true},
		{Selector: ".markitect/write.lock", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/cache/", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/views/", Owner: "projectwork"},
		{Selector: ".markitect/drafts/", Owner: "projectwork", Operational: true},
		{Selector: ".markitect/state/briefings/", Owner: "projecthistory", Operational: true},
		{Selector: ".markitect/state/explorations/", Owner: "projectexplore", Operational: true},
		{Selector: ".markitect/workflows/model-first.md", Owner: "projectwork"},
		{Selector: "AGENTS.md", Owner: "project-onboarding"},
		{Selector: "CLAUDE.md", Owner: "project-onboarding"},
	}
	for _, provider := range []string{"codex", "claude"} {
		for _, name := range NativeSkillPaths(provider) {
			if !toolRegistered(tools, name) {
				tools = append(tools, projectcoverage.ToolPath{Selector: name, Owner: "project-onboarding"})
			}
		}
	}
	if config.DocumentPath != "" && !toolRegistered(tools, config.DocumentPath) {
		tools = append(tools, projectcoverage.ToolPath{Selector: config.DocumentPath, Owner: "projectwork"})
	}
	return tools
}

// NativeSkillNames declares the exact operation skill identities owned by
// Markitect onboarding. Provider renderers supply their canonical content.
func NativeSkillNames() []string {
	return []string{"markitect-init", "markitect-extract", "markitect-design", "markitect-implement", "markitect-cleanup", "markitect-verify", "markitect-apply", "markitect-check", "markitect-suggest", "markitect-configure"}
}

// NativeSkillPaths returns exact generated files, never a directory prefix.
// Unrelated repository skills therefore remain owned by the adopting project.
func NativeSkillPaths(provider string) []string {
	base := ""
	switch provider {
	case "codex":
		base = ".agents/skills/"
	case "claude":
		base = ".claude/skills/"
	default:
		return nil
	}
	paths := []string{base + "markitect-implement/references/recovery.md"}
	for _, name := range NativeSkillNames() {
		paths = append(paths, base+name+"/SKILL.md", base+name+"/references/operating-guide.md")
	}
	return paths
}

// IsToolPath reports whether an exact repository path is registered as a
// Markitect-owned input/output for this project configuration.
func IsToolPath(config Config, file string) bool {
	return file == projectcoverage.IgnorePath || toolRegistered(ToolPaths(config), file)
}

// IsCanonicalModelPath reports whether file is one of the exact selected
// Definition sources for this project.
func IsCanonicalModelPath(config Config, file string) bool {
	for _, candidate := range config.ModelFiles {
		if candidate == file {
			return true
		}
	}
	return false
}

func toolRegistered(tools []projectcoverage.ToolPath, file string) bool {
	for _, tool := range tools {
		selector := strings.TrimSuffix(tool.Selector, "/")
		if file == selector || strings.HasPrefix(file, selector+"/") {
			return true
		}
	}
	return false
}

// IsRepositoryPath reports whether a value is a normalized repository-relative path.
func IsRepositoryPath(value string) bool {
	return value != "" && !strings.Contains(value, "\\") && !strings.ContainsRune(value, 0) && !path.IsAbs(value) && path.Clean(value) == value && value != "." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../") && !strings.HasSuffix(value, "/..")
}

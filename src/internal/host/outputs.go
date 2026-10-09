package host

import (
	"bytes"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/agentrules"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func Generated(data []byte) bool {
	return agentrules.IsGenerated(data)
}

func CheckOutputs(p *Project) []core.Diagnostic {
	if len(p.Diagnostics) > 0 {
		return nil
	}
	outputs, err := GenerateOutputs(p)
	if err != nil {
		return []core.Diagnostic{{Code: "render", Message: err.Error()}}
	}
	var findings []core.Diagnostic
	findings = append(findings, CheckConsistency(p)...)
	findings = append(findings, checkProviderAdapterInputs(p)...)
	names := sortedFiles(outputs)
	for _, name := range names {
		data, ok := p.Snapshot.Files[name]
		if !ok {
			findings = append(findings, core.Diagnostic{Code: "missing-output", Path: name, Message: "generated file is missing; run markitect render --write"})
		} else if !bytes.Equal(normalize(data), normalize(outputs[name])) {
			findings = append(findings, core.Diagnostic{Code: "output-drift", Path: name, Message: "generated file differs from its canonical source"})
		}
	}
	for _, name := range staleProjectionPaths(p, outputs) {
		findings = append(findings, core.Diagnostic{Code: "stale-output", Path: name, Message: "previously generated file has no current source; inspect and remove in the same migration"})
	}

	return findings
}

// staleProjectionPaths lists only Markitect's Markdown and TOML projections
// that have no current owner in this Project. Other generated-marked files
// (for example schemas produced by another generator) are outside this
// projection's ownership contract.
func staleProjectionPaths(p *Project, expectedOutputs map[string][]byte) []string {
	if p == nil || p.Snapshot == nil {
		return nil
	}
	nestedRoots := independentNestedProjectRoots(p, expectedOutputs)
	var stale []string
	for _, name := range sortedFiles(p.Snapshot.Files) {
		if !(strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".toml")) || !Generated(p.Snapshot.Files[name]) {
			continue
		}
		if _, owned := expectedOutputs[name]; owned || inIndependentNestedProject(name, nestedRoots) {
			continue
		}
		stale = append(stale, name)
	}
	return stale
}

func normalize(data []byte) []byte { return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")) }

// independentNestedProjectRoots identifies standalone Project directories
// whose generated files belong to another Markitect invocation. A nested
// manifest must be a strict supported Project resource and must not overlap
// any input or output owned by the parent Project. The nested Project is not
// parsed into the parent model; this list only narrows stale-marker inventory.
func independentNestedProjectRoots(p *Project, expectedOutputs map[string][]byte) []string {
	if p == nil || p.Snapshot == nil || p.Graph == nil || p.Graph.Project == nil {
		return nil
	}
	var roots []string
	for _, name := range sortedFiles(p.Snapshot.Files) {
		if path.Base(name) != "markitect.yaml" {
			continue
		}
		root := path.Dir(name)
		if root == "." {
			continue
		}
		manifest, err := authoring.Parse(name, p.Snapshot.Files[name])
		if err != nil || manifest.APIVersion != core.APIVersion || manifest.Kind != "Project" {
			continue
		}
		if overlapsParentProject(p, root, expectedOutputs) {
			continue
		}
		roots = append(roots, root)
	}
	return roots
}

func overlapsParentProject(p *Project, nestedRoot string, expectedOutputs map[string][]byte) bool {
	for _, area := range p.Graph.Project.Spec.Areas {
		if pathsOverlap(nestedRoot, area.Path) {
			return true
		}
	}
	for output := range expectedOutputs {
		if pathsOverlap(nestedRoot, output) {
			return true
		}
	}
	for _, inputs := range p.InputFiles {
		for _, input := range inputs {
			if Within(input, nestedRoot) {
				return true
			}
		}
	}
	for _, domain := range p.DomainInputs {
		if domain.Package == "" && Within(domain.Path, nestedRoot) {
			return true
		}
	}
	return false
}

func inIndependentNestedProject(file string, roots []string) bool {
	for _, root := range roots {
		if Within(file, root) {
			return true
		}
	}
	return false
}

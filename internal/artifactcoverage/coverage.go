// Package artifactcoverage checks explicit repository artifact ownership for a
// project-selected set of roots. It treats project artifacts as opaque paths
// and bytes; canonical and generated ownership comes from Markitect's model.
package artifactcoverage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const APIVersion = "markitect.example.org/artifact-coverage/v1alpha1"

type Config struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Spec       ConfigSpec `yaml:"spec"`
}

type ConfigSpec struct {
	Roots      []string    `yaml:"roots"`
	Tooling    []ToolOwner `yaml:"tooling,omitempty"`
	Exclusions []Exclusion `yaml:"exclusions,omitempty"`
}

type ToolOwner struct {
	Path  string `yaml:"path"`
	Owner string `yaml:"owner"`
}

type Exclusion struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}

type File struct {
	Path   string   `yaml:"path"`
	Class  string   `yaml:"class"`
	Owners []string `yaml:"owners,omitempty"`
	Reason string   `yaml:"reason,omitempty"`
}

type Finding struct {
	Code    string `yaml:"code"`
	Path    string `yaml:"path,omitempty"`
	Message string `yaml:"message"`
}

type Report struct {
	Status         string    `yaml:"status"`
	SnapshotDigest string    `yaml:"snapshotDigest"`
	Files          []File    `yaml:"files"`
	Findings       []Finding `yaml:"findings"`
}

// ParseConfig decodes one closed, versioned configuration document.
func ParseConfig(data []byte) (Config, error) {
	var config Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode artifact coverage config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("artifact coverage config must contain one YAML document")
		}
		return Config{}, fmt.Errorf("decode artifact coverage config trailer: %w", err)
	}
	if config.APIVersion != APIVersion || config.Kind != "ArtifactCoverage" {
		return Config{}, fmt.Errorf("artifact coverage config must use apiVersion %q and kind ArtifactCoverage", APIVersion)
	}
	return config, nil
}

// Check reads the current working tree, including untracked files, and checks
// every file under the configured roots. The root and config path are local
// filesystem paths; reported paths are repository-relative POSIX paths.
func Check(root, configPath string) (Report, error) {
	if configPath == "" {
		configPath = "markitect-artifacts.yaml"
	}
	snap, err := source.Load(root, "")
	if err != nil {
		return Report{}, err
	}
	configBytes, ok := snap.Files[configPath]
	if !ok {
		return Report{}, fmt.Errorf("artifact coverage config %q is absent from the working tree snapshot", configPath)
	}
	config, err := ParseConfig(configBytes)
	if err != nil {
		return Report{}, err
	}
	if err := validateLiteralPath(configPath); err != nil {
		return Report{}, fmt.Errorf("invalid config path %q: %w", configPath, err)
	}
	if err := validateConfig(config, snap, root); err != nil {
		return Report{}, err
	}
	configOwned := false
	for _, declaration := range config.Spec.Tooling {
		if declaration.Path == configPath {
			configOwned = true
			break
		}
	}
	if !configOwned || !inRoots(configPath, config.Spec.Roots) {
		return Report{}, fmt.Errorf("coverage config %q must be inside a managed root and have explicit tooling ownership", configPath)
	}
	report := Report{SnapshotDigest: snap.Digest(), Files: []File{}, Findings: []Finding{}}
	project, err := host.Parse(snap)
	if err != nil {
		return Report{}, fmt.Errorf("parse Markitect project: %w", err)
	}
	for _, diagnostic := range project.Diagnostics {
		report.Findings = append(report.Findings, Finding{Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message})
	}
	_, generatedOwners, renderErr := render.GenerateWithOwners(project.Graph, project.Snapshot.Files)
	if renderErr != nil {
		report.Findings = append(report.Findings, Finding{Code: "render", Message: renderErr.Error()})
	}
	for _, managedRoot := range config.Spec.Roots {
		if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(managedRoot))); statErr == nil {
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return Report{}, fmt.Errorf("inspect managed root %q: %w", managedRoot, statErr)
		}
		present := false
		for name := range snap.Files {
			if name == managedRoot || strings.HasPrefix(name, managedRoot+"/") {
				present = true
				break
			}
		}
		if !present {
			for name := range generatedOwners {
				if name == managedRoot || strings.HasPrefix(name, managedRoot+"/") {
					present = true
					break
				}
			}
		}
		if !present {
			report.Findings = append(report.Findings, Finding{Code: "stale-root", Path: managedRoot, Message: "managed root is absent from the working tree and has no renderer-owned output"})
		}
	}

	canonical := map[string][]string{}
	for _, resource := range project.Resources {
		if resource == nil || resource.Package != "" {
			continue
		}
		canonical[resource.Path] = append(canonical[resource.Path], resource.GraphKey())
	}
	for _, domain := range project.DomainInputs {
		if domain.Package == "" {
			canonical[domain.Path] = append(canonical[domain.Path], "domain:"+domain.APIVersion+"/"+domain.Name)
		}
	}
	inputs := map[string][]string{}
	for _, resource := range project.Resources {
		if resource == nil || resource.Package != "" || resource.Kind == "Project" || resource.Kind == "Package" {
			continue
		}
		for _, input := range project.InputFiles[resource.GraphKey()] {
			inputs[input] = append(inputs[input], resource.GraphKey())
		}
	}
	tooling := map[string][]string{}
	for _, declaration := range config.Spec.Tooling {
		tooling[declaration.Path] = []string{declaration.Owner}
	}
	exclusions := map[string]string{}
	for _, exclusion := range config.Spec.Exclusions {
		exclusions[exclusion.Path] = exclusion.Reason
	}

	paths := make([]string, 0, len(snap.Files))
	for name := range snap.Files {
		if inRoots(name, config.Spec.Roots) {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	for _, name := range paths {
		classes := make([]File, 0, 5)
		if owners := canonical[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "canonical", Owners: sortedUnique(owners)})
		}
		if owners := inputs[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "input", Owners: sortedUnique(owners)})
		}
		if owners := generatedOwners[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "generated", Owners: sortedUnique(owners)})
		}
		if owners := tooling[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "tooling", Owners: sortedUnique(owners)})
		}
		if reason, excluded := exclusions[name]; excluded {
			classes = append(classes, File{Path: name, Class: "excluded", Reason: reason})
		}
		if len(classes) == 0 {
			report.Findings = append(report.Findings, Finding{Code: "unmanaged", Path: name, Message: "file is under a managed root but has no declared owner or reasoned exclusion"})
			continue
		}
		if len(classes) > 1 {
			classNames := make([]string, len(classes))
			for i := range classes {
				classNames[i] = classes[i].Class
			}
			report.Findings = append(report.Findings, Finding{Code: "ownership-collision", Path: name, Message: "file has overlapping classifications: " + strings.Join(classNames, ", ")})
			continue
		}
		report.Files = append(report.Files, classes[0])
	}
	for name, owners := range generatedOwners {
		if !inRoots(name, config.Spec.Roots) {
			continue
		}
		if _, exists := snap.Files[name]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "missing-generated", Path: name, Message: "renderer-owned generated file is missing; run markitect render --write (owners: " + strings.Join(sortedUnique(owners), ", ") + ")"})
		}
	}
	for name, data := range snap.Files {
		if !inRoots(name, config.Spec.Roots) ||
			!(strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".toml")) ||
			!render.IsGenerated(data) {
			continue
		}
		if _, expected := generatedOwners[name]; !expected {
			report.Findings = append(report.Findings, Finding{Code: "orphan-generated", Path: name, Message: "generated marker has no current renderer owner"})
		}
	}
	for _, declaration := range config.Spec.Tooling {
		if _, exists := snap.Files[declaration.Path]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "stale-tooling", Path: declaration.Path, Message: "tooling ownership declaration does not name a file in the working tree snapshot"})
		}
	}
	for _, exclusion := range config.Spec.Exclusions {
		if _, exists := snap.Files[exclusion.Path]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "stale-exclusion", Path: exclusion.Path, Message: "exclusion declaration does not name a file in the working tree snapshot"})
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Path != report.Findings[j].Path {
			return report.Findings[i].Path < report.Findings[j].Path
		}
		if report.Findings[i].Code != report.Findings[j].Code {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Message < report.Findings[j].Message
	})
	if len(report.Findings) == 0 {
		report.Status = "passed"
	} else {
		report.Status = "failed"
	}
	return report, nil
}

func validateConfig(config Config, snap *snapshot.Snapshot, root string) error {
	if len(config.Spec.Roots) == 0 {
		return errors.New("artifact coverage must declare at least one managed root")
	}
	allPaths := make([]string, 0, len(snap.Files)+len(config.Spec.Roots))
	for name := range snap.Files {
		allPaths = append(allPaths, name)
	}
	sort.Strings(allPaths)
	rootSet := map[string]string{}
	orderedRoots := make([]string, 0, len(config.Spec.Roots))
	for _, name := range config.Spec.Roots {
		if err := validateLiteralPath(name); err != nil {
			return fmt.Errorf("invalid managed root %q: %w", name, err)
		}
		if err := source.ValidateIncludedPaths([]string{name}); err != nil {
			return fmt.Errorf("managed root %q is not available to Markitect snapshots: %w", name, err)
		}
		key := foldPath(name)
		if prior, ok := rootSet[key]; ok {
			return fmt.Errorf("managed roots %q and %q are aliases", prior, name)
		}
		for _, prior := range orderedRoots {
			if withinFolded(name, prior) || withinFolded(prior, name) {
				return fmt.Errorf("managed roots %q and %q overlap", prior, name)
			}
		}
		rootSet[key] = name
		orderedRoots = append(orderedRoots, name)
		for actual := range snap.Files {
			parts := strings.Split(actual, "/")
			for i := range parts {
				prefix := strings.Join(parts[:i+1], "/")
				if foldPath(prefix) == key && prefix != name {
					return fmt.Errorf("managed root %q aliases repository path spelling %q", name, prefix)
				}
			}
		}
	}
	if err := source.ValidateIncludedPaths(allPaths); err != nil {
		return fmt.Errorf("repository path inventory is not portable: %w", err)
	}
	toolingSeen := map[string]bool{}
	for _, declaration := range config.Spec.Tooling {
		if err := validateLiteralPath(declaration.Path); err != nil {
			return fmt.Errorf("invalid tooling path %q: %w", declaration.Path, err)
		}
		if strings.TrimSpace(declaration.Owner) == "" {
			return fmt.Errorf("tooling owner for %q must be nonblank", declaration.Path)
		}
		if !inRoots(declaration.Path, config.Spec.Roots) {
			return fmt.Errorf("tooling path %q is outside every managed root", declaration.Path)
		}
		key := foldPath(declaration.Path)
		if toolingSeen[key] {
			return fmt.Errorf("tooling path %q is declared more than once or through a case alias", declaration.Path)
		}
		toolingSeen[key] = true
		allPaths = append(allPaths, declaration.Path)
	}
	exclusionSeen := map[string]bool{}
	for _, exclusion := range config.Spec.Exclusions {
		if err := validateLiteralPath(exclusion.Path); err != nil {
			return fmt.Errorf("invalid exclusion path %q: %w", exclusion.Path, err)
		}
		if strings.TrimSpace(exclusion.Reason) == "" {
			return fmt.Errorf("exclusion %q must have a nonblank reason", exclusion.Path)
		}
		if !inRoots(exclusion.Path, config.Spec.Roots) {
			return fmt.Errorf("exclusion %q is outside every managed root", exclusion.Path)
		}
		key := foldPath(exclusion.Path)
		if exclusionSeen[key] {
			return fmt.Errorf("exclusion %q is declared more than once or through a case alias", exclusion.Path)
		}
		exclusionSeen[key] = true
		allPaths = append(allPaths, exclusion.Path)
	}
	sort.Strings(allPaths)
	if err := source.ValidateIncludedPaths(allPaths); err != nil {
		return fmt.Errorf("declared paths are not portable: %w", err)
	}
	return nil
}

func validateLiteralPath(value string) error {
	if value == "" || strings.ContainsAny(value, "\\:*?[]{}\x00") || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." {
		return fmt.Errorf("expected a canonical repository-relative literal POSIX path")
	}
	for _, component := range strings.Split(value, "/") {
		trimmed := strings.TrimRight(component, ". ")
		base := strings.ToUpper(strings.SplitN(trimmed, ".", 2)[0])
		if component == "" || trimmed == "" || base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return fmt.Errorf("path contains a component that is ambiguous on Windows")
		}
	}
	return nil
}

func inRoots(name string, roots []string) bool {
	for _, root := range roots {
		if name == root || strings.HasPrefix(name, root+"/") {
			return true
		}
	}
	return false
}

func withinFolded(name, root string) bool {
	name, root = foldPath(name), foldPath(root)
	return name == root || strings.HasPrefix(name, root+"/")
}

func foldPath(name string) string {
	var b strings.Builder
	for _, r := range name {
		min := r
		for next := simpleFold(r); next != r; next = simpleFold(next) {
			if next < min {
				min = next
			}
		}
		b.WriteRune(min)
	}
	return b.String()
}

// simpleFold is isolated here so path alias checks use Unicode simple folding
// consistently across Windows and Linux hosts.
func simpleFold(r rune) rune { return unicode.SimpleFold(r) }

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

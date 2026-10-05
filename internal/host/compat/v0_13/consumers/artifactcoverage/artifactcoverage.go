// Package artifactcoverage evaluates managed artifact ownership over facts
// supplied by the Host. It does not load snapshots or interpret project files.
package artifactcoverage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const APIVersion = "markitect.example.org/artifact-coverage/v1alpha1"

// Config is the standalone, versioned coverage policy. SourcePath is supplied
// by the Host after decoding; it is not part of the YAML contract.
type Config struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Spec       ConfigSpec `yaml:"spec"`
	SourcePath string     `yaml:"-"`
}

type ConfigSpec struct {
	Roots      []string    `yaml:"roots"`
	Tooling    []ToolOwner `yaml:"tooling,omitempty"`
	Vendor     []ToolOwner `yaml:"vendor,omitempty"`
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

// OwnerFact connects one exact repository path to its already-resolved owner.
// Multiple input facts may name the same path when resources share an input.
type OwnerFact struct {
	Path  string
	Owner string
}

// Inventory is the immutable, Host-composed evidence for one snapshot.
// Files are exact repository-relative POSIX paths and opaque bytes. The Host
// supplies owner facts after parsing, input resolution, and output composition.
type Inventory struct {
	SnapshotDigest  string
	Files           map[string][]byte
	CanonicalOwners []OwnerFact
	InputOwners     []OwnerFact
	GeneratedOwners []OwnerFact
	ProjectedOwners []OwnerFact
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

// ParseConfig decodes exactly one closed ArtifactCoverage YAML document.
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

// Check evaluates path ownership without acquiring or re-parsing any source.
// A finding preserves the candidate's failed status; errors indicate malformed
// config or internally inconsistent Host-supplied inventory.
func Check(config Config, inventory Inventory) (Report, error) {
	if err := validateInventory(config, inventory); err != nil {
		return Report{}, err
	}
	report := Report{
		Status:         "passed",
		SnapshotDigest: inventory.SnapshotDigest,
		Files:          []File{},
		Findings:       []Finding{},
	}
	for _, root := range config.Spec.Roots {
		if rootPresent(root, inventory.Files, inventory.GeneratedOwners, inventory.ProjectedOwners) {
			continue
		}
		report.Findings = append(report.Findings, Finding{Code: "stale-root", Path: root, Message: "managed root is absent from the supplied snapshot and has no renderer-owned output"})
	}

	canonical := groupOwners(inventory.CanonicalOwners)
	inputs := groupOwners(inventory.InputOwners)
	generated := groupOwners(inventory.GeneratedOwners)
	projected := groupOwners(inventory.ProjectedOwners)
	tooling := make(map[string][]string, len(config.Spec.Tooling))
	for _, declaration := range config.Spec.Tooling {
		tooling[declaration.Path] = append(tooling[declaration.Path], declaration.Owner)
	}
	vendor := make(map[string][]string, len(config.Spec.Vendor))
	for _, declaration := range config.Spec.Vendor {
		vendor[declaration.Path] = append(vendor[declaration.Path], declaration.Owner)
	}
	exclusions := make(map[string]string, len(config.Spec.Exclusions))
	for _, exclusion := range config.Spec.Exclusions {
		exclusions[exclusion.Path] = exclusion.Reason
	}

	paths := make([]string, 0, len(inventory.Files))
	for name := range inventory.Files {
		if inRoots(name, config.Spec.Roots) {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	for _, name := range paths {
		classes := make([]File, 0, 7)
		if owners := canonical[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "canonical", Owners: owners})
		}
		if owners := inputs[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "input", Owners: owners})
		}
		if owners := generated[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "generated", Owners: owners})
		}
		if owners := projected[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "projected", Owners: owners})
			if len(owners) > 1 {
				report.Findings = append(report.Findings, Finding{Code: "owner-conflict", Path: name, Message: "projected representation has multiple contract owners: " + strings.Join(owners, ", ")})
			}
		}
		if owners := tooling[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "tooling", Owners: sortedUnique(owners)})
		}
		if owners := vendor[name]; len(owners) > 0 {
			classes = append(classes, File{Path: name, Class: "vendor", Owners: sortedUnique(owners)})
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

	for _, path := range sortedOwnerPaths(inventory.InputOwners) {
		if inRoots(path, config.Spec.Roots) {
			if _, exists := inventory.Files[path]; !exists {
				report.Findings = append(report.Findings, Finding{Code: "input.missing", Path: path, Message: "resolved input is absent from the supplied snapshot"})
			}
		}
	}
	for _, path := range sortedOwnerPaths(inventory.GeneratedOwners) {
		if inRoots(path, config.Spec.Roots) {
			if _, exists := inventory.Files[path]; !exists {
				owners := generated[path]
				report.Findings = append(report.Findings, Finding{Code: "missing-generated", Path: path, Message: "renderer-owned generated file is missing (owners: " + strings.Join(owners, ", ") + ")"})
			}
		}
	}
	for _, path := range sortedOwnerPaths(inventory.ProjectedOwners) {
		if inRoots(path, config.Spec.Roots) {
			if _, exists := inventory.Files[path]; !exists {
				owners := projected[path]
				report.Findings = append(report.Findings, Finding{Code: "missing-projected", Path: path, Message: "declared projected representation is missing (owners: " + strings.Join(owners, ", ") + ")"})
			}
		}
	}
	for _, name := range paths {
		if !isRendererProjectionPath(name) || !hasGeneratedMarker(inventory.Files[name]) {
			continue
		}
		if _, expected := generated[name]; !expected {
			report.Findings = append(report.Findings, Finding{Code: "orphan-generated", Path: name, Message: "generated marker has no current renderer owner"})
		}
	}
	for _, declaration := range config.Spec.Tooling {
		if _, exists := inventory.Files[declaration.Path]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "stale-tooling", Path: declaration.Path, Message: "tooling ownership declaration does not name a file in the supplied snapshot"})
		}
	}
	for _, declaration := range config.Spec.Vendor {
		if _, exists := inventory.Files[declaration.Path]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "stale-vendor", Path: declaration.Path, Message: "vendor ownership declaration does not name a file in the supplied snapshot"})
		}
	}
	for _, exclusion := range config.Spec.Exclusions {
		if _, exists := inventory.Files[exclusion.Path]; !exists {
			report.Findings = append(report.Findings, Finding{Code: "stale-exclusion", Path: exclusion.Path, Message: "exclusion declaration does not name a file in the supplied snapshot"})
		}
	}
	sortFindings(report.Findings)
	if len(report.Findings) > 0 {
		report.Status = "failed"
	}
	return report, nil
}

func validateInventory(config Config, inventory Inventory) error {
	if strings.TrimSpace(inventory.SnapshotDigest) == "" {
		return errors.New("artifact coverage requires an identified snapshot digest")
	}
	if len(config.Spec.Roots) == 0 {
		return errors.New("artifact coverage must declare at least one managed root")
	}
	if config.SourcePath == "" {
		return errors.New("artifact coverage config source path must be supplied by the Host")
	}
	if err := validateLiteralPath(config.SourcePath); err != nil {
		return fmt.Errorf("invalid artifact coverage config source path %q: %w", config.SourcePath, err)
	}
	if !inRoots(config.SourcePath, config.Spec.Roots) {
		return fmt.Errorf("coverage config %q is outside every managed root", config.SourcePath)
	}
	configOwned := false
	for _, declaration := range config.Spec.Tooling {
		if declaration.Path == config.SourcePath {
			configOwned = true
		}
	}
	if !configOwned {
		return fmt.Errorf("coverage config %q must have explicit tooling ownership", config.SourcePath)
	}

	filePaths := make([]string, 0, len(inventory.Files))
	for name := range inventory.Files {
		if err := validateLiteralPath(name); err != nil {
			return fmt.Errorf("invalid inventory file path %q: %w", name, err)
		}
		filePaths = append(filePaths, name)
	}
	sort.Strings(filePaths)
	if err := validatePortablePaths(filePaths); err != nil {
		return fmt.Errorf("inventory paths are not portable: %w", err)
	}

	rootSet := map[string]string{}
	orderedRoots := make([]string, 0, len(config.Spec.Roots))
	for _, root := range config.Spec.Roots {
		if err := validateLiteralPath(root); err != nil {
			return fmt.Errorf("invalid managed root %q: %w", root, err)
		}
		key := foldPath(root)
		if prior, ok := rootSet[key]; ok {
			return fmt.Errorf("managed roots %q and %q are aliases", prior, root)
		}
		for _, prior := range orderedRoots {
			if withinFolded(root, prior) || withinFolded(prior, root) {
				return fmt.Errorf("managed roots %q and %q overlap", prior, root)
			}
		}
		rootSet[key] = root
		orderedRoots = append(orderedRoots, root)
		for _, actual := range filePaths {
			for _, prefix := range pathPrefixes(actual) {
				if foldPath(prefix) == key && prefix != root {
					return fmt.Errorf("managed root %q aliases repository path spelling %q", root, prefix)
				}
			}
		}
	}

	declaredPaths := append([]string(nil), filePaths...)
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
		declaredPaths = append(declaredPaths, declaration.Path)
	}
	vendorSeen := map[string]bool{}
	for _, declaration := range config.Spec.Vendor {
		if err := validateLiteralPath(declaration.Path); err != nil {
			return fmt.Errorf("invalid vendor path %q: %w", declaration.Path, err)
		}
		if strings.TrimSpace(declaration.Owner) == "" {
			return fmt.Errorf("vendor owner for %q must be nonblank", declaration.Path)
		}
		if !inRoots(declaration.Path, config.Spec.Roots) {
			return fmt.Errorf("vendor path %q is outside every managed root", declaration.Path)
		}
		key := foldPath(declaration.Path)
		if vendorSeen[key] {
			return fmt.Errorf("vendor path %q is declared more than once or through a case alias", declaration.Path)
		}
		vendorSeen[key] = true
		declaredPaths = append(declaredPaths, declaration.Path)
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
		declaredPaths = append(declaredPaths, exclusion.Path)
	}
	sort.Strings(declaredPaths)
	if err := validatePortablePaths(declaredPaths); err != nil {
		return fmt.Errorf("declared paths are not portable: %w", err)
	}
	for _, factSet := range [][]OwnerFact{inventory.CanonicalOwners, inventory.InputOwners, inventory.GeneratedOwners, inventory.ProjectedOwners} {
		facts := append([]OwnerFact(nil), factSet...)
		sort.Slice(facts, func(i, j int) bool {
			if facts[i].Path != facts[j].Path {
				return facts[i].Path < facts[j].Path
			}
			return facts[i].Owner < facts[j].Owner
		})
		for _, fact := range facts {
			if err := validateLiteralPath(fact.Path); err != nil {
				return fmt.Errorf("invalid ownership fact path %q: %w", fact.Path, err)
			}
			if strings.TrimSpace(fact.Owner) == "" {
				return fmt.Errorf("owner fact for %q must have a nonblank owner", fact.Path)
			}
			declaredPaths = append(declaredPaths, fact.Path)
		}
	}
	sort.Strings(declaredPaths)
	if err := validatePortablePaths(declaredPaths); err != nil {
		return fmt.Errorf("declared paths are not portable: %w", err)
	}
	return nil
}

func rootPresent(root string, files map[string][]byte, generated, projected []OwnerFact) bool {
	for name := range files {
		if inRoot(name, root) {
			return true
		}
	}
	for _, fact := range generated {
		if inRoot(fact.Path, root) {
			return true
		}
	}
	for _, fact := range projected {
		if inRoot(fact.Path, root) {
			return true
		}
	}
	return false
}

func inRoot(name, root string) bool { return name == root || strings.HasPrefix(name, root+"/") }

func inRoots(name string, roots []string) bool {
	for _, root := range roots {
		if inRoot(name, root) {
			return true
		}
	}
	return false
}

func withinFolded(name, root string) bool {
	name, root = foldPath(name), foldPath(root)
	return inRoot(name, root)
}

func pathPrefixes(name string) []string {
	parts := strings.Split(name, "/")
	prefixes := make([]string, 0, len(parts))
	for i := range parts {
		prefixes = append(prefixes, strings.Join(parts[:i+1], "/"))
	}
	return prefixes
}

func validateLiteralPath(value string) error {
	if value == "" || !utf8.ValidString(value) || strings.ContainsRune(value, 0) || strings.Contains(value, `\`) || strings.Contains(value, ":") || path.IsAbs(value) {
		return fmt.Errorf("expected a safe repository-relative POSIX path")
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("expected a canonical repository-relative POSIX path")
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
			return fmt.Errorf("path contains an unsafe component")
		}
		for _, r := range component {
			if r < 32 || strings.ContainsRune(`<|>?*`, r) || strings.ContainsRune(`"`, r) || strings.ContainsRune(`:`, r) {
				return fmt.Errorf("path contains an invalid character")
			}
		}
		base := strings.ToUpper(strings.SplitN(component, ".", 2)[0])
		if windowsReservedName(base) {
			return fmt.Errorf("path contains a Windows device name")
		}
	}
	return nil
}

func windowsReservedName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}

func validatePortablePaths(paths []string) error {
	type priorPath struct {
		name string
		dir  bool
	}
	seen := map[string]priorPath{}
	for _, p := range paths {
		parts := strings.Split(p, "/")
		for i := range parts {
			name := strings.Join(parts[:i+1], "/")
			isDir := i < len(parts)-1
			key := foldPath(name)
			if prior, ok := seen[key]; ok && (prior.name != name || prior.dir != isDir) {
				return fmt.Errorf("case-insensitive path collision between %q and %q", prior.name, name)
			}
			seen[key] = priorPath{name: name, dir: isDir}
		}
	}
	return nil
}

func foldPath(value string) string {
	var folded strings.Builder
	for _, r := range value {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		folded.WriteRune(min)
	}
	return folded.String()
}

func groupOwners(facts []OwnerFact) map[string][]string {
	grouped := map[string][]string{}
	for _, fact := range facts {
		grouped[fact.Path] = append(grouped[fact.Path], fact.Owner)
	}
	for name, owners := range grouped {
		grouped[name] = sortedUnique(owners)
	}
	return grouped
}

func sortedOwnerPaths(facts []OwnerFact) []string {
	paths := make([]string, 0, len(facts))
	for _, fact := range facts {
		paths = append(paths, fact.Path)
	}
	sort.Strings(paths)
	result := paths[:0]
	for _, name := range paths {
		if len(result) == 0 || result[len(result)-1] != name {
			result = append(result, name)
		}
	}
	return result
}

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

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Message < findings[j].Message
	})
}

func isRendererProjectionPath(name string) bool {
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".toml")
}

func hasGeneratedMarker(data []byte) bool {
	text := strings.TrimSpace(string(data))
	if strings.HasPrefix(text, "---\n") || strings.HasPrefix(text, "---\r\n") {
		lines := strings.Split(text, "\n")
		closed := false
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				text = strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
				closed = true
				break
			}
		}
		if !closed {
			return false
		}
	}
	return strings.HasPrefix(text, "<!-- Generated by Markitect;") || strings.HasPrefix(text, "# Generated by Markitect;") || strings.HasPrefix(text, "<!-- Generated by Markitect -->")
}

package architecture

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// The code map is read from the inspected repository, like the import gate.
// It is not embedded: release source archives do not require this maintenance input at runtime.
const (
	CodeMapSource = "internal/tooling/architecture/codemap.yaml"
	CodeMapPage   = "docs/development/code-map.md"
	CodeMapUpdate = "go test ./internal/tooling/architecture -run TestCodeMapPage -update"
)

// MapLayer is one code-map layer. Gate lists the coarse import-gate layers,
// as returned by layer, that a package of this layer may have.
type MapLayer struct {
	Name, Title, Meaning string
	Gate                 []string
}

// Layers is ordered; code-map entries follow this order, then their path.
var Layers = []MapLayer{
	{"apply-model", "Canonical model", "Pure canonical v1 Model check over explicit bytes.", []string{"apply-model"}},
	{"apply-snapshot", "Snapshots", "Pure immutable repository tree values.", []string{"apply-snapshot"}},
	{"apply-organization", "Organization", "Pure Manager topology, context selection and settings.", []string{"apply-organization"}},
	{"apply-realization", "Realization", "Pure ownership, accounting and candidate rules.", []string{"apply-realization"}},
	{"apply-run", "Run", "One closed complete Manager hierarchy over direct Sessions, Git, checks and records.", []string{"apply-run"}},
	{"apply-project", "Project", "Checked structural input and immutable model-derived context.", []string{"apply-project"}},
	{"runtime-config", "Runtime configuration", "Closed static native and check profile metadata.", []string{"runtime-config"}},
	{"native-harness", "Native harness", "Model-derived instructions and tools for concrete native Sessions.", []string{"native-harness"}},
	{"apply-usecase", "Apply", "Product workflow composition; unavailable at the S2 source boundary.", []string{"apply-usecase"}},
	{"model-view", "Model view", "Deterministic readable view and reviewed source-bound write plan.", []string{"model-view"}},
	{"apply-adapter", "Adapters", "Concrete repository, container engine, Agent and check adapters, profile metadata and document I/O.", []string{"apply-adapter"}},
	{"apply-bootstrap", "Composition", "Constructs actual services and their adapters.", []string{"apply-bootstrap"}},
	{"apply-command", "Product command", "Thin entrypoint to product composition.", []string{"apply-command"}},
	{"agent-protocol", "Agent protocol", "Frame protocol between the Orchestrator and the session helper.", []string{"agent-protocol"}},
	{"agent-runtime", "Session helper", "Runs inside each Agent container: provider process, MCP bridge and stdio framing.", []string{"agent-runtime"}},
	{"runtime-command", "Runtime command", "Thin entrypoint of the session helper.", []string{"runtime-command"}},
	{"build-data", "Build data", "Version identity and embedded license notices.", []string{"build-data"}},
	{"maintenance", "Maintenance", "Explicitly separated release and repository tooling.", []string{"maintenance"}},
	{"maintenance-command", "Maintenance commands", "Thin entrypoints to maintenance services.", []string{"maintenance-command"}},
	{"test-support", "Test support", "Hermetic fixture support available only to tests.", []string{"test-support"}},
	{"standalone", "Standalone tools", "Public bootstrap and source smoke with no product imports.", []string{"standalone"}},
}

type CodeMap struct {
	Packages []MappedPackage `yaml:"packages"`
}

type MappedPackage struct {
	Path    string `yaml:"path"`
	Layer   string `yaml:"layer"`
	Purpose string `yaml:"purpose"`
	Doc     string `yaml:"doc"`
}

// ParseCodeMap strictly decodes one code-map document.
func ParseCodeMap(data []byte) (CodeMap, error) {
	var m CodeMap
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&m); err != nil {
		return CodeMap{}, fmt.Errorf("%s: %w", CodeMapSource, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return CodeMap{}, fmt.Errorf("%s: expected exactly one YAML document", CodeMapSource)
	}
	return m, nil
}

func LoadCodeMap(root string) (CodeMap, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(CodeMapSource)))
	if err != nil {
		return CodeMap{}, err
	}
	return ParseCodeMap(data)
}

// Packages lists the repository-relative directory of every Go package below
// root, following `go list ./...`: a directory with at least one .go file,
// including test-only and platform-specific packages. It skips .git, vendor,
// testdata, node_modules, names starting with "." or "_", and nested modules.
// Nested modules under product cmd/ and internal/ are errors: the product
// uses the root module and must not disappear from this inventory.
func Packages(root string) ([]string, error) {
	found := map[string]bool{}
	var modules []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		base := entry.Name()
		if entry.IsDir() {
			if rel != "." && (base == ".git" || base == "vendor" || base == "testdata" || base == "node_modules" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		switch {
		case base == "go.mod" && rel != "go.mod":
			if strings.HasPrefix(rel, "cmd/") || strings.HasPrefix(rel, "internal/") {
				return fmt.Errorf("nested Go module %s is forbidden: cmd/ and internal/ use the root Go module", rel)
			}
			modules = append(modules, path.Dir(rel))
		case strings.HasSuffix(base, ".go") && !strings.HasPrefix(base, ".") && !strings.HasPrefix(base, "_"):
			found[path.Dir(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []string
	for dir := range found {
		nested := false
		for _, module := range modules {
			nested = nested || dir == module || strings.HasPrefix(dir, module+"/")
		}
		if !nested {
			out = append(out, dir)
		}
	}
	sort.Strings(out)
	return out, nil
}

// CheckCodeMap reports every difference between the map and the Go packages
// below root, and every entry without a known layer, a compatible import-gate
// layer, a purpose or an existing owning document.
func CheckCodeMap(root string, m CodeMap) ([]string, error) {
	packages, err := Packages(root)
	if err != nil {
		return nil, err
	}
	order := map[string]int{}
	for i, l := range Layers {
		order[l.Name] = i
	}
	anchors := map[string]map[string]bool{}
	var problems []string
	mapped := map[string]bool{}
	for i, p := range m.Packages {
		report := func(format string, args ...any) {
			problems = append(problems, fmt.Sprintf("%s: entry %d (%s): %s", CodeMapSource, i+1, p.Path, fmt.Sprintf(format, args...)))
		}
		if p.Path == "" || p.Path != path.Clean(p.Path) || path.IsAbs(p.Path) || strings.HasPrefix(p.Path, "../") || strings.Contains(p.Path, "\\") {
			report("path must be a clean repository-relative path with forward slashes")
		}
		if mapped[p.Path] {
			report("duplicate path")
		}
		mapped[p.Path] = true
		rank, known := order[p.Layer]
		if !known {
			report("unknown layer %q", p.Layer)
		} else {
			if gate, _ := layer(p.Path); !contains(Layers[rank].Gate, gate) {
				report("layer %q does not match import-gate layer %q", p.Layer, gate)
			}
			if i > 0 {
				prev := m.Packages[i-1]
				if prevRank, ok := order[prev.Layer]; ok && (prevRank > rank || prevRank == rank && prev.Path >= p.Path) {
					report("entries must be ordered by layer, then by path")
				}
			}
		}
		if strings.TrimSpace(p.Purpose) == "" || strings.ContainsAny(p.Purpose, "|\n") {
			report("purpose must be one non-empty line without '|'")
		}
		if err := checkDoc(root, p.Doc, anchors); err != nil {
			report("owning document: %v", err)
		}
	}
	present := map[string]bool{}
	for _, p := range packages {
		present[p] = true
		if !mapped[p] {
			problems = append(problems, fmt.Sprintf("missing package %s: add an entry to %s", p, CodeMapSource))
		}
	}
	var extra []string
	for p := range mapped {
		if !present[p] {
			extra = append(extra, p)
		}
	}
	sort.Strings(extra)
	for _, p := range extra {
		problems = append(problems, fmt.Sprintf("extra entry %s: no Go package exists there; remove it from %s", p, CodeMapSource))
	}
	return problems, nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func checkDoc(root, doc string, cache map[string]map[string]bool) error {
	file, anchor, _ := strings.Cut(doc, "#")
	if file == "" || file != path.Clean(file) || path.IsAbs(file) || strings.HasPrefix(file, "../") || strings.Contains(file, "\\") {
		return fmt.Errorf("%q is not a clean repository-relative path", doc)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s does not exist", file)
	}
	if anchor == "" {
		return nil
	}
	if cache[file] == nil {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return err
		}
		cache[file] = markdownAnchors(string(data))
	}
	if !cache[file][anchor] {
		return fmt.Errorf("%s has no anchor #%s", file, anchor)
	}
	return nil
}

var explicitID = regexp.MustCompile(`[<\s]id="([^"]+)"`)

// markdownAnchors returns GitHub-style ATX heading anchors and explicit HTML
// ids outside fenced code: the subset of scripts/check-docs.py these documents use.
func markdownAnchors(text string) map[string]bool {
	anchors := map[string]bool{}
	counts := map[string]int{}
	fence := ""
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		for _, match := range explicitID.FindAllStringSubmatch(line, -1) {
			anchors[match[1]] = true
		}
		heading := strings.TrimLeft(trimmed, "#")
		if level := len(trimmed) - len(heading); level == 0 || level > 6 || heading != "" && heading[0] != ' ' {
			continue
		}
		heading = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(heading), "#"))
		var slug strings.Builder
		for _, r := range strings.ToLower(heading) {
			switch {
			case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
				slug.WriteRune(r)
			case unicode.IsSpace(r):
				slug.WriteRune('-')
			}
		}
		base := strings.Trim(slug.String(), "-")
		if base == "" {
			continue
		}
		candidate := base
		if n := counts[base]; n > 0 {
			candidate = fmt.Sprintf("%s-%d", base, n)
		}
		counts[base]++
		anchors[candidate] = true
	}
	return anchors
}

// RenderCodeMap returns the generated page docs/development/code-map.md.
func RenderCodeMap(m CodeMap) []byte {
	var b strings.Builder
	pageDir := path.Dir(CodeMapPage)
	fmt.Fprintf(&b, "# Code map\n\n")
	fmt.Fprintf(&b, "This page lists every Go package in the repository with its layer, purpose and owning document. Read the owning document before you change a package, and update it when the package's behavior changes.\n\n")
	fmt.Fprintf(&b, "The page is generated from [codemap.yaml](%s). Do not edit it by hand. Change the YAML, then regenerate the page from the repository root:\n\n", relativeLink(pageDir, CodeMapSource))
	fmt.Fprintf(&b, "```powershell\n%s\n```\n\n", CodeMapUpdate)
	fmt.Fprintf(&b, "Tests in `internal/tooling/architecture` require every actual Go package to have one compatible map entry and an existing owning document. The [import gate](modules.md#mechanical-dependency-gate) checks all production, test and platform files against explicit responsibility edges; there is no broad maintenance or adapter exemption. The document writer additionally uses only the reviewed guarded API of `%s`.\n\n", GuardedWritePackage)

	counts := map[string]int{}
	for _, p := range m.Packages {
		counts[p.Layer]++
	}
	fmt.Fprintf(&b, "## Layers\n\n| Layer | Meaning | Import gate layers | Packages |\n|---|---|---|---|\n")
	for _, l := range Layers {
		fmt.Fprintf(&b, "| `%s` | %s | `%s` | %d |\n", l.Name, l.Meaning, strings.Join(l.Gate, "`, `"), counts[l.Name])
	}
	for _, l := range Layers {
		if counts[l.Name] == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n| Package | Purpose | Owning document |\n|---|---|---|\n", l.Title)
		for _, p := range m.Packages {
			if p.Layer != l.Name {
				continue
			}
			file, anchor, _ := strings.Cut(p.Doc, "#")
			target := relativeLink(pageDir, file)
			if anchor != "" {
				target += "#" + anchor
			}
			fmt.Fprintf(&b, "| `%s` | %s | [%s](%s) |\n", p.Path, p.Purpose, file, target)
		}
	}
	return []byte(b.String())
}

// relativeLink returns the slash path from directory from to file to; both
// are clean repository-relative paths.
func relativeLink(from, to string) string {
	a, b := strings.Split(from, "/"), strings.Split(to, "/")
	common := 0
	for common < len(a) && common < len(b)-1 && a[common] == b[common] {
		common++
	}
	parts := make([]string, 0, len(a)-common+len(b)-common)
	for range a[common:] {
		parts = append(parts, "..")
	}
	return strings.Join(append(parts, b[common:]...), "/")
}

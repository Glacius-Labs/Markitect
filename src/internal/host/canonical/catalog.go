package canonical

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"go.yaml.in/yaml/v3"
)

const (
	ModuleManifestAPIVersion = "markitect.example.org/module/v1alpha1"
	ModuleManifestPath       = "module.yaml"
	ModuleTypeSchema         = "schema"
	ModuleTypeProjection     = "projection"
	maxManifestBytes         = 1 << 20
)

var (
	moduleNamePattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	moduleVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	moduleDigestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	projectorIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	checkNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,62}$`)
)

// ModulePackage is already supplied local package content. This API performs
// no lookup, filesystem access, network operation, or provider loading.
type ModulePackage struct {
	ManifestBytes []byte
	Files         map[string][]byte
	// Modes optionally supplies Git-style regular-file modes. Omission means
	// regular non-executable data mode 100644.
	Modes map[string]string
}

// Pin names one exact module release and immutable content digest.
type Pin struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
	Digest  string `yaml:"digest" json:"digest"`
}

// Manifest is the closed Module v1alpha1 protocol understood by this Host.
type Manifest struct {
	APIVersion string             `yaml:"apiVersion"`
	Name       string             `yaml:"name"`
	Version    string             `yaml:"version"`
	Type       string             `yaml:"type"`
	Purpose    string             `yaml:"purpose"`
	Requires   ModuleRequirements `yaml:"requires"`
	Provides   ModuleCapabilities `yaml:"provides"`
	Discovery  DiscoveryMetadata  `yaml:"discovery,omitempty"`
}

type ModuleRequirements struct {
	Core    string `yaml:"core"`
	Modules []Pin  `yaml:"modules,omitempty"`
}

type ModuleCapabilities struct {
	Schemas    []string                `yaml:"schemas,omitempty"`
	Projectors []ProjectorRegistration `yaml:"projectors,omitempty"`
}

// ProjectorRegistration declares a capability only. It contains no executable
// or implementation reference and cannot create an active projection binding.
type ProjectorRegistration struct {
	ID             string              `yaml:"id"`
	Version        string              `yaml:"version"`
	Target         string              `yaml:"target"`
	Mode           string              `yaml:"-" json:"-"`
	AllKinds       bool                `yaml:"allKinds,omitempty"`
	SupportedKinds []core.KindIdentity `yaml:"supportedKinds,omitempty"`
	AllowedRoots   []string            `yaml:"allowedRoots"`
	Guidance       string              `yaml:"guidance,omitempty"`
	RequiredChecks []string            `yaml:"requiredChecks,omitempty"`
}

type DiscoveryMetadata struct {
	Goals []string `yaml:"goals,omitempty"`
	Paths []string `yaml:"paths,omitempty"`
}

// Activation contains exactly the capabilities named by the owner's selected
// pins. Registering Projectors does not choose scopes, targets, or policies.
type Activation struct {
	Modules     []Pin
	ModuleTypes map[Pin]string
	Schemas     []core.Schema
	Projectors  []RegisteredProjector
}

// RegisteredProjector binds an immutable capability declaration to the exact
// Module package that supplied it. The wrapper is Host-authored provenance;
// Module YAML cannot choose or rewrite its own pin.
type RegisteredProjector struct {
	Module       Pin
	Registration ProjectorRegistration
}

type Recommendation struct {
	Name         string
	Version      string
	Type         string
	MatchedGoals []string
	MatchedPaths []string
}

func validModuleVersion(value string) bool {
	parts := moduleVersionPattern.FindStringSubmatch(value)
	if parts == nil {
		return false
	}
	for _, part := range strings.Split(parts[4], ".") {
		if part == "" {
			continue
		}
		allDigits := true
		for _, char := range part {
			if char < '0' || char > '9' {
				allDigits = false
				break
			}
		}
		if allDigits && len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

type indexedModule struct {
	pkg      ModulePackage
	manifest Manifest
	digest   string
}

// DecodeManifest strictly reads one versioned Module manifest.
func DecodeManifest(data []byte) (Manifest, error) {
	root, err := strictDocument("module.yaml", data, maxManifestBytes)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateManifestNode(root); err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := root.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("module.yaml:%d: invalid Module manifest: %w", lineOf(root), err)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, fmt.Errorf("module.yaml:%d: %w", lineOf(root), err)
	}
	return manifest, nil
}

func validateManifestNode(root *yaml.Node) error {
	if err := closedMapping("module.yaml", root, "Module manifest", "apiVersion", "name", "version", "type", "purpose", "requires", "provides", "discovery"); err != nil {
		return err
	}
	if err := required("module.yaml", root, "apiVersion", "name", "version", "type", "purpose", "requires", "provides"); err != nil {
		return err
	}
	for _, field := range []string{"apiVersion", "name", "version", "type", "purpose"} {
		if err := stringNode("module.yaml", child(root, field), field); err != nil {
			return err
		}
	}
	requires := child(root, "requires")
	if err := closedMapping("module.yaml", requires, "requires", "core", "modules"); err != nil {
		return err
	}
	if err := required("module.yaml", requires, "core"); err != nil {
		return err
	}
	if err := stringNode("module.yaml", child(requires, "core"), "requires.core"); err != nil {
		return err
	}
	if modules := child(requires, "modules"); modules != nil {
		if err := validatePinSequence(modules); err != nil {
			return err
		}
	}
	provides := child(root, "provides")
	if err := closedMapping("module.yaml", provides, "provides", "schemas", "projectors"); err != nil {
		return err
	}
	if err := required("module.yaml", provides, "schemas", "projectors"); err != nil {
		return err
	}
	if schemas := child(provides, "schemas"); schemas.Kind != yaml.SequenceNode {
		return fmt.Errorf("module.yaml:%d: provides.schemas must be a sequence", lineOf(schemas))
	} else {
		for _, item := range schemas.Content {
			if err := stringNode("module.yaml", item, "schema path"); err != nil {
				return err
			}
		}
	}
	projectors := child(provides, "projectors")
	if projectors.Kind != yaml.SequenceNode {
		return fmt.Errorf("module.yaml:%d: provides.projectors must be a sequence", lineOf(projectors))
	}
	for _, projector := range projectors.Content {
		if err := validateProjectorNode(projector); err != nil {
			return err
		}
	}
	if discovery := child(root, "discovery"); discovery != nil {
		if err := closedMapping("module.yaml", discovery, "discovery", "goals", "paths"); err != nil {
			return err
		}
		for _, field := range []string{"goals", "paths"} {
			if n := child(discovery, field); n != nil {
				if n.Kind != yaml.SequenceNode {
					return fmt.Errorf("module.yaml:%d: discovery.%s must be a sequence", lineOf(n), field)
				}
				for _, item := range n.Content {
					if err := stringNode("module.yaml", item, "discovery."+field+" item"); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func validatePinSequence(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return fmt.Errorf("module.yaml:%d: requires.modules must be a sequence", lineOf(node))
	}
	for _, pin := range node.Content {
		if err := closedMapping("module.yaml", pin, "module pin", "name", "version", "digest"); err != nil {
			return err
		}
		if err := required("module.yaml", pin, "name", "version", "digest"); err != nil {
			return err
		}
		for _, field := range []string{"name", "version", "digest"} {
			if err := stringNode("module.yaml", child(pin, field), "module pin "+field); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateProjectorNode(node *yaml.Node) error {
	fields := []string{"id", "version", "target", "allKinds", "supportedKinds", "allowedRoots", "guidance", "requiredChecks"}
	if err := closedMapping("module.yaml", node, "projector", fields...); err != nil {
		return err
	}
	if err := required("module.yaml", node, "id", "version", "target", "allowedRoots"); err != nil {
		return err
	}
	for _, field := range []string{"id", "version", "target"} {
		if err := stringNode("module.yaml", child(node, field), "projector "+field); err != nil {
			return err
		}
	}
	if all := child(node, "allKinds"); all != nil && (all.Kind != yaml.ScalarNode || all.Tag != "!!bool") {
		return fmt.Errorf("module.yaml:%d: projector allKinds must be a boolean", lineOf(all))
	}
	if roots := child(node, "allowedRoots"); roots.Kind != yaml.SequenceNode {
		return fmt.Errorf("module.yaml:%d: projector allowedRoots must be a sequence", lineOf(roots))
	} else {
		for _, root := range roots.Content {
			if err := stringNode("module.yaml", root, "allowed root"); err != nil {
				return err
			}
		}
	}
	if kinds := child(node, "supportedKinds"); kinds != nil {
		if kinds.Kind != yaml.SequenceNode {
			return fmt.Errorf("module.yaml:%d: supportedKinds must be a sequence", lineOf(kinds))
		}
		for _, identity := range kinds.Content {
			if err := closedMapping("module.yaml", identity, "kind identity", "apiVersion", "kind"); err != nil {
				return err
			}
			if err := required("module.yaml", identity, "apiVersion", "kind"); err != nil {
				return err
			}
			for _, field := range []string{"apiVersion", "kind"} {
				if err := stringNode("module.yaml", child(identity, field), "supportedKinds."+field); err != nil {
					return err
				}
			}
		}
	}
	for _, field := range []string{"guidance"} {
		if value := child(node, field); value != nil {
			if err := stringNode("module.yaml", value, "projector "+field); err != nil {
				return err
			}
		}
	}
	if checks := child(node, "requiredChecks"); checks != nil {
		if checks.Kind != yaml.SequenceNode {
			return fmt.Errorf("module.yaml:%d: requiredChecks must be a sequence", lineOf(checks))
		}
		for _, check := range checks.Content {
			if err := stringNode("module.yaml", check, "required check"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateManifest(m Manifest) error {
	if m.APIVersion != ModuleManifestAPIVersion {
		return fmt.Errorf("unsupported Module apiVersion %q", m.APIVersion)
	}
	if !moduleNamePattern.MatchString(m.Name) {
		return fmt.Errorf("Module name %q must be a lowercase DNS label", m.Name)
	}
	if !validModuleVersion(m.Version) {
		return fmt.Errorf("Module version %q must be an exact semantic version", m.Version)
	}
	if strings.TrimSpace(m.Purpose) == "" || len(m.Purpose) > 4096 {
		return fmt.Errorf("Module purpose must contain 1 to 4096 bytes")
	}
	if m.Type != ModuleTypeSchema && m.Type != ModuleTypeProjection {
		return fmt.Errorf("Module type must be exactly schema or projection")
	}
	if m.Requires.Core != "1" {
		return fmt.Errorf("Module requires.core must be exactly %q", "1")
	}
	if m.Type == ModuleTypeSchema && (len(m.Provides.Schemas) == 0 || len(m.Provides.Projectors) != 0) {
		return fmt.Errorf("schema Module must provide at least one Schema and no Projectors")
	}
	if m.Type == ModuleTypeProjection && (len(m.Provides.Schemas) != 0 || len(m.Provides.Projectors) == 0) {
		return fmt.Errorf("projection Module must provide at least one Projector and no Schemas")
	}
	paths, folded := map[string]bool{}, map[string]string{}
	for _, file := range m.Provides.Schemas {
		p, err := safeModulePath(file)
		if err != nil {
			return err
		}
		if !(strings.HasSuffix(strings.ToLower(p), ".yaml") || strings.HasSuffix(strings.ToLower(p), ".yml")) {
			return fmt.Errorf("Schema path %q must end in .yaml or .yml", file)
		}
		if p == ModuleManifestPath {
			return fmt.Errorf("Schema path %q is reserved for the Module manifest", file)
		}
		if paths[p] {
			return fmt.Errorf("Schema path %q is duplicated", p)
		}
		paths[p] = true
		lower := strings.ToLower(p)
		if old, ok := folded[lower]; ok {
			return fmt.Errorf("Schema paths %q and %q collide by case", old, p)
		}
		folded[lower] = p
	}
	deps := map[string]bool{}
	if len(m.Requires.Modules) > maxModuleDependencies {
		return fmt.Errorf("Module has more than %d exact dependencies", maxModuleDependencies)
	}
	for _, pin := range m.Requires.Modules {
		if err := validatePin(pin); err != nil {
			return fmt.Errorf("required module: %w", err)
		}
		if deps[pin.Name] {
			return fmt.Errorf("required module %q is duplicated", pin.Name)
		}
		deps[pin.Name] = true
		if pin.Name == m.Name {
			return fmt.Errorf("Module cannot require itself")
		}
	}
	if len(m.Provides.Schemas) > maxModuleSchemas || len(m.Provides.Projectors) > maxModuleProjectors {
		return fmt.Errorf("Module exceeds capability count limits")
	}
	projectors := map[string]bool{}
	for _, p := range m.Provides.Projectors {
		if err := validateRegistration(p); err != nil {
			return err
		}
		if projectors[p.ID] {
			return fmt.Errorf("projector id %q is duplicated", p.ID)
		}
		projectors[p.ID] = true
	}
	if len(m.Discovery.Goals) > maxDiscoveryItems || len(m.Discovery.Paths) > maxDiscoveryItems {
		return fmt.Errorf("discovery metadata exceeds %d items per list", maxDiscoveryItems)
	}
	goals := map[string]bool{}
	for _, goal := range m.Discovery.Goals {
		if strings.TrimSpace(goal) == "" || len(goal) > 256 {
			return fmt.Errorf("discovery goals must contain 1 to 256 bytes")
		}
		key := strings.ToLower(strings.TrimSpace(goal))
		if goals[key] {
			return fmt.Errorf("duplicate discovery goal %q", goal)
		}
		goals[key] = true
	}
	pathsSeen := map[string]bool{}
	for _, value := range m.Discovery.Paths {
		p, err := safeModulePath(value)
		if err != nil {
			return fmt.Errorf("discovery path: %w", err)
		}
		if pathsSeen[strings.ToLower(p)] {
			return fmt.Errorf("duplicate discovery path %q", p)
		}
		pathsSeen[strings.ToLower(p)] = true
	}
	return nil
}

const (
	maxModuleSchemas        = 128
	maxModuleProjectors     = 128
	maxModuleDependencies   = 256
	maxDiscoveryItems       = 64
	maxModuleFiles          = 128
	maxModuleFileBytes      = core.MaxSchemaBytes
	maxCatalogPackages      = 256
	maxCatalogBytes         = core.MaxTotalInputBytes
	maxCatalogManifestBytes = 16 << 20
)

func validatePin(pin Pin) error {
	if !moduleNamePattern.MatchString(pin.Name) {
		return fmt.Errorf("name %q is not a lowercase DNS label", pin.Name)
	}
	if !validModuleVersion(pin.Version) {
		return fmt.Errorf("version %q is not an exact semantic version", pin.Version)
	}
	if !moduleDigestPattern.MatchString(pin.Digest) {
		return fmt.Errorf("digest must be sha256 followed by 64 lowercase hexadecimal digits")
	}
	return nil
}

func safeScopeRoot(value string) (string, error) {
	if value == "." {
		return value, nil
	}
	if strings.HasSuffix(value, "/") {
		value = strings.TrimSuffix(value, "/")
	}
	return safeModulePath(value)
}
func validateRegistration(p ProjectorRegistration) error {
	if !projectorIDPattern.MatchString(p.ID) {
		return fmt.Errorf("projector id %q is invalid", p.ID)
	}
	if !validModuleVersion(p.Version) {
		return fmt.Errorf("projector %q version must be exact semantic version", p.ID)
	}
	if !projectorIDPattern.MatchString(p.Target) {
		return fmt.Errorf("projector %q target %q is invalid", p.ID, p.Target)
	}
	if p.AllKinds {
		if len(p.SupportedKinds) > 0 {
			return fmt.Errorf("projector %q cannot combine allKinds with supportedKinds", p.ID)
		}
	} else if len(p.SupportedKinds) == 0 {
		return fmt.Errorf("projector %q must list supportedKinds or explicitly set allKinds: true", p.ID)
	}
	if len(p.AllowedRoots) == 0 || len(p.AllowedRoots) > 64 {
		return fmt.Errorf("projector %q must declare 1 to 64 allowedRoots", p.ID)
	}
	roots := map[string]bool{}
	for _, r := range p.AllowedRoots {
		v, err := safeScopeRoot(r)
		if err != nil {
			return fmt.Errorf("projector %q: %w", p.ID, err)
		}
		if roots[v] {
			return fmt.Errorf("projector %q allowed root %q is duplicated", p.ID, v)
		}
		roots[v] = true
	}
	if len(p.Guidance) > 8192 {
		return fmt.Errorf("projector %q guidance exceeds 8192 bytes", p.ID)
	}
	kinds := map[string]bool{}
	for _, k := range p.SupportedKinds {
		if k.APIVersion == "" || k.Kind == "" {
			return fmt.Errorf("projector %q has an empty supported Kind identity", p.ID)
		}
		key := k.Key()
		if kinds[key] {
			return fmt.Errorf("projector %q duplicates supported Kind %s/%s", p.ID, k.APIVersion, k.Kind)
		}
		kinds[key] = true
	}
	checks := map[string]bool{}
	if len(p.RequiredChecks) > 256 {
		return fmt.Errorf("projector %q has more than 256 required checks", p.ID)
	}
	for _, c := range p.RequiredChecks {
		if !checkNamePattern.MatchString(c) {
			return fmt.Errorf("projector %q required check %q is invalid", p.ID, c)
		}
		if checks[c] {
			return fmt.Errorf("projector %q required check %q is duplicated", p.ID, c)
		}
		checks[c] = true
	}
	return nil
}

// DigestPackage hashes the exact manifest bytes and every declared Schema
// source path, mode, and byte sequence. Undeclared files and executable modes
// are rejected so the digest cannot hide ambiguous package content.
func DigestPackage(pkg ModulePackage) (string, error) {
	m, err := DecodeManifest(pkg.ManifestBytes)
	if err != nil {
		return "", err
	}
	if err := validatePackageFiles(m, pkg); err != nil {
		return "", err
	}
	h := sha256.New()
	writeHashField(h, []byte("markitect-module-package-v1"))
	writeHashField(h, []byte(ModuleManifestPath))
	manifestMode := pkg.Modes[ModuleManifestPath]
	if manifestMode == "" {
		manifestMode = "100644"
	}
	writeHashField(h, []byte(manifestMode))
	writeHashField(h, pkg.ManifestBytes)
	paths := append([]string(nil), m.Provides.Schemas...)
	sort.Strings(paths)
	for _, p := range paths {
		writeHashField(h, []byte(p))
		mode := pkg.Modes[p]
		if mode == "" {
			mode = "100644"
		}
		writeHashField(h, []byte(mode))
		writeHashField(h, pkg.Files[p])
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

type hashWriter interface{ Write([]byte) (int, error) }

func writeHashField(h hashWriter, data []byte) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(data)))
	_, _ = h.Write(size[:])
	_, _ = h.Write(data)
}

func validatePackageFiles(m Manifest, pkg ModulePackage) error {
	if len(pkg.Files) > maxModuleFiles {
		return fmt.Errorf("Module contains more than %d supplied files", maxModuleFiles)
	}
	declared := map[string]bool{}
	for _, p := range m.Provides.Schemas {
		declared[p] = true
	}
	seen := map[string]string{}
	totalBytes := len(pkg.ManifestBytes)
	for p, b := range pkg.Files {
		clean, err := safeModulePath(p)
		if err != nil {
			return err
		}
		if clean != p {
			return fmt.Errorf("Module file path %q is not normalized", p)
		}
		if !declared[p] {
			return fmt.Errorf("supplied Module file %q is not declared in provides.schemas", p)
		}
		key := strings.ToLower(p)
		if prior, ok := seen[key]; ok {
			return fmt.Errorf("supplied Module files %q and %q collide by case", prior, p)
		}
		seen[key] = p
		if len(b) > maxModuleFileBytes {
			return fmt.Errorf("Module file %q exceeds %d bytes", p, maxModuleFileBytes)
		}
		if totalBytes > core.MaxTotalInputBytes-len(b) {
			return fmt.Errorf("Module package exceeds %d supplied bytes", core.MaxTotalInputBytes)
		}
		totalBytes += len(b)
	}
	if len(pkg.Files) != len(declared) {
		for p := range declared {
			if _, ok := pkg.Files[p]; !ok {
				return fmt.Errorf("declared Schema file %q was not supplied", p)
			}
		}
	}
	for p, mode := range pkg.Modes {
		if p != ModuleManifestPath && !declared[p] {
			return fmt.Errorf("mode supplied for undeclared Module file %q", p)
		}
		if mode != "100644" {
			return fmt.Errorf("Module data file %q has unsupported file mode %q", p, mode)
		}
	}
	return nil
}

// Resolve activates only exact owner-selected Pins. Dependencies must already
// appear in that selection; this function never adds a package or creates a
// projection binding.
func Resolve(packages []ModulePackage, pins []Pin) (Activation, error) {
	if len(pins) == 0 {
		return Activation{}, fmt.Errorf("at least one exact Module pin must be selected")
	}
	if len(packages) > maxCatalogPackages || len(pins) > maxCatalogPackages {
		return Activation{}, fmt.Errorf("catalog and selected pin count are bounded to %d", maxCatalogPackages)
	}
	byExact := map[string]indexedModule{}
	byNameVersion := map[string]string{}
	catalogBytes := 0
	for _, pkg := range packages {
		if catalogBytes > maxCatalogBytes-len(pkg.ManifestBytes) {
			return Activation{}, fmt.Errorf("catalog exceeds %d supplied bytes", maxCatalogBytes)
		}
		catalogBytes += len(pkg.ManifestBytes)
		for _, data := range pkg.Files {
			if catalogBytes > maxCatalogBytes-len(data) {
				return Activation{}, fmt.Errorf("catalog exceeds %d supplied bytes", maxCatalogBytes)
			}
			catalogBytes += len(data)
		}
		m, err := DecodeManifest(pkg.ManifestBytes)
		if err != nil {
			return Activation{}, err
		}
		digest, err := DigestPackage(pkg)
		if err != nil {
			return Activation{}, err
		}
		nv := m.Name + "@" + m.Version
		if old, ok := byNameVersion[nv]; ok && old != digest {
			return Activation{}, fmt.Errorf("catalog has conflicting package bytes for %s", nv)
		}
		if _, ok := byExact[nv+"#"+digest]; ok {
			return Activation{}, fmt.Errorf("catalog duplicates package %s", nv)
		}
		byNameVersion[nv] = digest
		byExact[nv+"#"+digest] = indexedModule{pkg: pkg, manifest: m, digest: digest}
	}
	selected := map[string]indexedModule{}
	pinByName := map[string]Pin{}
	for _, pin := range pins {
		if err := validatePin(pin); err != nil {
			return Activation{}, fmt.Errorf("selected Module pin: %w", err)
		}
		if _, ok := pinByName[pin.Name]; ok {
			return Activation{}, fmt.Errorf("selected Module name %q is pinned more than once", pin.Name)
		}
		item, ok := byExact[pin.Name+"@"+pin.Version+"#"+pin.Digest]
		if !ok {
			return Activation{}, fmt.Errorf("exact selected Module %s@%s with digest %s is unavailable", pin.Name, pin.Version, pin.Digest)
		}
		selected[pin.Name] = item
		pinByName[pin.Name] = pin
	}
	for _, item := range selected {
		for _, need := range item.manifest.Requires.Modules {
			actual, ok := pinByName[need.Name]
			if !ok {
				return Activation{}, fmt.Errorf("Module %s requires explicitly selected pin %s@%s", item.manifest.Name, need.Name, need.Version)
			}
			if actual != need {
				return Activation{}, fmt.Errorf("Module %s requires exact pin %s@%s %s", item.manifest.Name, need.Name, need.Version, need.Digest)
			}
			if item.manifest.Type == ModuleTypeSchema && selected[need.Name].manifest.Type == ModuleTypeProjection {
				return Activation{}, fmt.Errorf("schema Module %s cannot depend on projection Module %s", item.manifest.Name, need.Name)
			}
		}
	}
	order, err := activationOrder(selected)
	if err != nil {
		return Activation{}, err
	}
	activation := Activation{Modules: make([]Pin, 0, len(order)), ModuleTypes: make(map[Pin]string, len(order))}
	schemaVersions := map[string]string{}
	for _, name := range order {
		item := selected[name]
		pin := pinByName[name]
		activation.Modules = append(activation.Modules, pin)
		activation.ModuleTypes[pin] = item.manifest.Type
		for _, file := range item.manifest.Provides.Schemas {
			schema, err := DecodeSchema("module:"+name+"@"+item.manifest.Version+"/"+file, item.pkg.Files[file])
			if err != nil {
				return Activation{}, err
			}
			if prior, ok := schemaVersions[schema.APIVersion]; ok {
				return Activation{}, fmt.Errorf("Schema apiVersion %q is provided by both %s and %s", schema.APIVersion, prior, name)
			}
			schemaVersions[schema.APIVersion] = name
			activation.Schemas = append(activation.Schemas, schema)
		}
		for _, registration := range item.manifest.Provides.Projectors {
			activation.Projectors = append(activation.Projectors, RegisteredProjector{Module: pin, Registration: registration})
		}
	}
	model, diagnostics := core.Compile(activation.Schemas, nil, "")
	if len(diagnostics) != 0 {
		var messages []string
		for _, diagnostic := range diagnostics {
			messages = append(messages, diagnostic.Code+": "+diagnostic.Message)
		}
		return Activation{}, fmt.Errorf("activated Schemas do not compile: %s", strings.Join(messages, "; "))
	}
	activation.Schemas = model.Schemas
	for _, registered := range activation.Projectors {
		registration := registered.Registration
		if registration.AllKinds {
			continue
		}
		for _, identity := range registration.SupportedKinds {
			if !activatedKind(activation.Schemas, identity) {
				return Activation{}, fmt.Errorf("projector %q names unresolved Kind %s/%s", registration.ID, identity.APIVersion, identity.Kind)
			}
		}
	}
	sort.Slice(activation.Schemas, func(i, j int) bool { return activation.Schemas[i].APIVersion < activation.Schemas[j].APIVersion })
	sort.Slice(activation.Projectors, func(i, j int) bool {
		if activation.Projectors[i].Module.Name != activation.Projectors[j].Module.Name {
			return activation.Projectors[i].Module.Name < activation.Projectors[j].Module.Name
		}
		return activation.Projectors[i].Registration.ID < activation.Projectors[j].Registration.ID
	})
	return activation, nil
}

func activationOrder(selected map[string]indexedModule) ([]string, error) {
	state := map[string]uint8{}
	var order []string
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("Module dependency cycle includes %q", name)
		case 2:
			return nil
		}
		state[name] = 1
		item, ok := selected[name]
		if !ok {
			return fmt.Errorf("required Module %q is not explicitly selected", name)
		}
		deps := append([]Pin(nil), item.manifest.Requires.Modules...)
		sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
		for _, dep := range deps {
			if err := visit(dep.Name); err != nil {
				return err
			}
		}
		state[name] = 2
		order = append(order, name)
		return nil
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func activatedKind(schemas []core.Schema, id core.KindIdentity) bool {
	for _, schema := range schemas {
		if schema.APIVersion == id.APIVersion {
			_, ok := schema.Kinds[id.Kind]
			return ok
		}
	}
	return false
}

// Recommend uses only explicit goal strings, supplied inventory paths, and
// declared discovery metadata. It does not inspect Schema bytes, inspect file
// contents, produce digests, authorize installation, or activate capabilities.
func Recommend(packages []ModulePackage, goals, inventory []string) ([]Recommendation, error) {
	if len(packages) > maxCatalogPackages || len(goals) > 256 || len(inventory) > 100_000 {
		return nil, fmt.Errorf("recommendation input exceeds bounded goal or inventory size")
	}
	normalGoals := make([]string, 0, len(goals))
	for _, g := range goals {
		trim := strings.TrimSpace(g)
		if trim != "" {
			normalGoals = append(normalGoals, strings.ToLower(trim))
		}
	}
	paths := map[string]string{}
	for _, p := range inventory {
		clean, err := safeModulePath(p)
		if err != nil {
			return nil, fmt.Errorf("inventory path: %w", err)
		}
		key := strings.ToLower(clean)
		if prior, ok := paths[key]; ok && prior != clean {
			return nil, fmt.Errorf("inventory paths %q and %q collide by case", prior, clean)
		}
		paths[key] = clean
	}
	var result []Recommendation
	seen := map[string]bool{}
	manifestBytes := 0
	for _, pkg := range packages {
		if manifestBytes > maxCatalogManifestBytes-len(pkg.ManifestBytes) {
			return nil, fmt.Errorf("recommendation manifests exceed %d supplied bytes", maxCatalogManifestBytes)
		}
		manifestBytes += len(pkg.ManifestBytes)
		m, err := DecodeManifest(pkg.ManifestBytes)
		if err != nil {
			return nil, err
		}
		key := m.Name + "@" + m.Version
		if seen[key] {
			return nil, fmt.Errorf("catalog duplicates recommendation candidate %s", key)
		}
		seen[key] = true
		r := Recommendation{Name: m.Name, Version: m.Version, Type: m.Type}
		for _, term := range m.Discovery.Goals {
			needle := strings.ToLower(strings.TrimSpace(term))
			for _, goal := range normalGoals {
				if strings.Contains(goal, needle) {
					r.MatchedGoals = append(r.MatchedGoals, term)
					break
				}
			}
		}
		for _, signal := range m.Discovery.Paths {
			if actual, ok := paths[strings.ToLower(signal)]; ok {
				r.MatchedPaths = append(r.MatchedPaths, actual)
			}
		}
		if len(r.MatchedGoals)+len(r.MatchedPaths) > 0 {
			sort.Strings(r.MatchedGoals)
			sort.Strings(r.MatchedPaths)
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Version < result[j].Version
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

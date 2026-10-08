// Package contentpackage reads and builds deterministic offline Markitect
// content archives. It has no network or filesystem access.
package contentpackage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
)

const (
	ManifestName       = "markitect-package.yaml"
	maxCompressedBytes = 64 << 20
	maxExpandedBytes   = 128 << 20
	maxMemberBytes     = 8 << 20
	maxArchiveFiles    = 10_000
)

var fixedZipTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// Archive is a verified, in-memory package. Files contains every member,
// indexed by its unchanged archive-relative POSIX path. Resources excludes
// Manifest and has runtime package origin set to the pinned package name.
type Archive struct {
	Manifest          *authoring.Resource
	Files             map[string][]byte
	Resources         []*authoring.Resource
	DomainDefinitions map[string]core.DomainDefinition
}

// Read validates the digest and ZIP structure before parsing package content.
// All data remains in memory; member paths are never extracted to disk.
func Read(pin authoring.PackagePin, archive []byte) (*Archive, error) {
	if len(archive) > maxCompressedBytes {
		return nil, fmt.Errorf("package archive exceeds the 64 MiB compressed limit")
	}
	if len(pin.SHA256) != 64 || strings.ToLower(pin.SHA256) != pin.SHA256 {
		return nil, fmt.Errorf("package pin sha256 must be 64 lowercase hexadecimal characters")
	}
	want, err := hex.DecodeString(pin.SHA256)
	if err != nil || len(want) != sha256.Size {
		return nil, fmt.Errorf("package pin sha256 is invalid")
	}
	digest := sha256.Sum256(archive)
	if !bytes.Equal(digest[:], want) {
		return nil, fmt.Errorf("package archive sha256 does not match pin")
	}
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("invalid package ZIP: %w", err)
	}
	if len(r.File) == 0 || len(r.File) > maxArchiveFiles {
		return nil, fmt.Errorf("package archive must contain between 1 and %d files", maxArchiveFiles)
	}

	files := make(map[string][]byte, len(r.File))
	folded := make(map[string]string, len(r.File))
	var declaredTotal uint64
	var actualTotal uint64
	for _, entry := range r.File {
		name := entry.Name
		if err := validatePath(name); err != nil {
			return nil, fmt.Errorf("unsafe package member %q: %w", name, err)
		}
		if entry.FileInfo().IsDir() || !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("package member %q is not a regular file", name)
		}
		if _, ok := files[name]; ok {
			return nil, fmt.Errorf("duplicate package member %q", name)
		}
		key := strings.ToLower(name)
		if previous, ok := folded[key]; ok {
			return nil, fmt.Errorf("package members %q and %q collide by case", previous, name)
		}
		folded[key] = name
		if entry.UncompressedSize64 > maxMemberBytes {
			return nil, fmt.Errorf("package member %q exceeds the 8 MiB limit", name)
		}
		if entry.CompressedSize64 > maxCompressedBytes {
			return nil, fmt.Errorf("package member %q exceeds the compressed size limit", name)
		}
		if declaredTotal > maxExpandedBytes-entry.UncompressedSize64 {
			return nil, fmt.Errorf("package exceeds the 128 MiB expanded limit")
		}
		declaredTotal += entry.UncompressedSize64

		member, err := readMember(entry)
		if err != nil {
			return nil, fmt.Errorf("read package member %q: %w", name, err)
		}
		if actualTotal > maxExpandedBytes-uint64(len(member)) {
			return nil, fmt.Errorf("package exceeds the 128 MiB expanded limit")
		}
		actualTotal += uint64(len(member))
		files[name] = member
	}
	return parseFiles(pin.Name, pin.Version, files)
}

func readMember(entry *zip.File) ([]byte, error) {
	r, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxMemberBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxMemberBytes {
		return nil, fmt.Errorf("actual member size exceeds the 8 MiB limit")
	}
	return data, nil
}

func parseFiles(name, version string, files map[string][]byte) (*Archive, error) {
	manifestData, ok := files[ManifestName]
	if !ok {
		return nil, fmt.Errorf("package archive is missing %s", ManifestName)
	}
	manifest, err := authoring.Parse(ManifestName, manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.Kind != "Package" {
		return nil, fmt.Errorf("%s must have kind Package", ManifestName)
	}
	if manifest.Metadata.Name != name {
		return nil, fmt.Errorf("package manifest name %q does not match pin %q", manifest.Metadata.Name, name)
	}
	manifest.Package = name
	if manifest.Spec.Version != version {
		return nil, fmt.Errorf("package manifest version %q does not match pin %q", manifest.Spec.Version, version)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}

	// Package-local domain declarations are loaded before package resources so
	// generic resources can be validated against the exact schemas they ship
	// with. Merely containing a domain file does not activate it: the manifest
	// must declare the exact member path.
	registry, domainDefinitions, err := parseDomainDefinitions(manifest, files)
	if err != nil {
		return nil, err
	}

	areaPaths := make([]string, 0, len(manifest.Spec.Areas))
	for _, area := range manifest.Spec.Areas {
		areaPaths = append(areaPaths, area.Path)
	}
	resources := make([]*authoring.Resource, 0)
	parseErrors := make(map[string]error)
	identities := make(map[string]string)
	accepted := map[string]bool{ManifestName: true}
	for _, filePath := range sortedFilePaths(files) {
		data := files[filePath]
		if filePath == ManifestName {
			continue
		}
		if _, isDomain := domainDefinitions[filePath]; isDomain {
			accepted[filePath] = true
			continue
		}
		if !isYAML(filePath) || !inAnyArea(filePath, areaPaths) {
			continue
		}
		resource, parseErr := authoring.ParseWithRegistry(filePath, data, registry)
		if parseErr != nil {
			parseErrors[filePath] = parseErr
			continue
		}
		if resource.Kind == "Project" || resource.Kind == "Package" || resource.Kind == "Domain" {
			return nil, fmt.Errorf("nested %s resource %q is not allowed in a package", resource.Kind, filePath)
		}
		if hasPackageRefs(resource) {
			return nil, fmt.Errorf("package resource %q contains a transitive package reference", filePath)
		}
		resource.Package = name
		key := resource.GraphKey()
		if prior, exists := identities[key]; exists {
			return nil, fmt.Errorf("package resources %q and %q have duplicate identity %s", prior, filePath, key)
		}
		identities[key] = filePath
		resources = append(resources, resource)
		accepted[filePath] = true
	}
	ordinary := declaredInputs(resources)
	for _, filePath := range sortedFilePaths(parseErrors) {
		parseErr := parseErrors[filePath]
		if ordinary[filePath] && isValidTextInput(files[filePath]) && !authoring.IsResourceEnvelopeWithRegistry(files[filePath], registry) {
			accepted[filePath] = true
			continue
		}
		return nil, fmt.Errorf("parse package resource %q: %w", filePath, parseErr)
	}
	for _, filePath := range sortedBoolKeys(ordinary) {
		data, exists := files[filePath]
		if !exists {
			return nil, fmt.Errorf("declared package input %q is missing", filePath)
		}
		if !isValidTextInput(data) {
			return nil, fmt.Errorf("declared package input %q is not UTF-8", filePath)
		}
		accepted[filePath] = true
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("package must contain at least one content resource")
	}
	resourceKeys := make(map[string]bool, len(resources))
	for _, resource := range resources {
		resourceKeys[resource.GraphKey()] = true
	}
	seenExports := make(map[string]bool, len(manifest.Spec.Exports))
	for _, export := range manifest.Spec.Exports {
		if export.Package != "" || export.Kind == "" || export.Namespace == "" || export.Name == "" {
			return nil, fmt.Errorf("package exports must be fully qualified local refs")
		}
		key := export.GraphKey(name, "", "")
		if !resourceKeys[key] {
			return nil, fmt.Errorf("package export %s does not identify a package resource", key)
		}
		if seenExports[key] {
			return nil, fmt.Errorf("package export %s is duplicated", key)
		}
		seenExports[key] = true
	}
	for _, filePath := range sortedFilePaths(files) {
		if !accepted[filePath] {
			return nil, fmt.Errorf("package member %q is not a manifest, area resource, or declared input", filePath)
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	return &Archive{Manifest: manifest, Files: files, Resources: resources, DomainDefinitions: domainDefinitions}, nil
}

package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"

	"go.yaml.in/yaml/v3"
)

// LibraryRef is one Library the manifest names.
type LibraryRef struct {
	Name    string
	Version string
	Line    int
}

// Manifest is the project's .markitect/manifest.yaml.
type Manifest struct {
	Libraries []LibraryRef // sorted by name
}

// Locked is one Library entry of the lock.
type Locked struct {
	Version string
	Digest  string
	Line    int
}

// Lock is the project's .markitect/lock.yaml, written by markitect lock.
type Lock struct {
	Libraries map[string]Locked
}

var version = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ParseManifest reads a manifest. file names it in diagnostics.
func ParseManifest(file string, data []byte) (Manifest, Diagnostics) {
	var ds Diagnostics
	var m Manifest
	top, err := mapping(data)
	if err != nil {
		ds.Add("manifest.syntax", file, 0, "%v", err)
		return m, ds
	}
	for key, node := range top {
		if key != "libraries" {
			ds.Add("manifest.syntax", file, node.Line, "unknown key %q: the manifest holds only libraries", key)
		}
	}
	libs := top["libraries"]
	if libs == nil {
		ds.Add("manifest.syntax", file, 0, "the manifest must name its Libraries, at least std:\nlibraries:\n  std: 1.0.0")
		return m, ds
	}
	if libs.Kind != yaml.MappingNode {
		ds.Add("manifest.syntax", file, libs.Line, "libraries must map each Library name to its version")
		return m, ds
	}
	for i := 0; i+1 < len(libs.Content); i += 2 {
		k, v := libs.Content[i], libs.Content[i+1]
		if !ValidSegment(k.Value) {
			ds.Add("manifest.syntax", file, k.Line, "%q cannot name a Library", k.Value)
			continue
		}
		if v.Kind != yaml.ScalarNode || !version.MatchString(v.Value) {
			ds.Add("manifest.syntax", file, v.Line, "Library %s needs a version such as 1.0.0", k.Value)
			continue
		}
		m.Libraries = append(m.Libraries, LibraryRef{Name: k.Value, Version: v.Value, Line: k.Line})
	}
	sort.Slice(m.Libraries, func(i, j int) bool { return m.Libraries[i].Name < m.Libraries[j].Name })
	for i := 1; i < len(m.Libraries); i++ {
		if m.Libraries[i].Name == m.Libraries[i-1].Name {
			ds.Add("manifest.syntax", file, m.Libraries[i].Line, "Library %s is named twice", m.Libraries[i].Name)
		}
	}
	return m, ds
}

// ParseLock reads a lock. file names it in diagnostics.
func ParseLock(file string, data []byte) (Lock, Diagnostics) {
	var ds Diagnostics
	lock := Lock{Libraries: map[string]Locked{}}
	top, err := mapping(data)
	if err != nil {
		ds.Add("lock.syntax", file, 0, "%v; run markitect lock", err)
		return lock, ds
	}
	libs := top["libraries"]
	if len(top) != 1 || libs == nil || libs.Kind != yaml.MappingNode {
		ds.Add("lock.syntax", file, 0, "the lock is not in the form markitect lock writes; run markitect lock")
		return lock, ds
	}
	for i := 0; i+1 < len(libs.Content); i += 2 {
		k, v := libs.Content[i], libs.Content[i+1]
		var entry struct {
			Version string `yaml:"version"`
			Digest  string `yaml:"digest"`
		}
		if err := v.Decode(&entry); err != nil || entry.Version == "" || entry.Digest == "" {
			ds.Add("lock.syntax", file, k.Line, "the entry for %s needs version and digest; run markitect lock", k.Value)
			continue
		}
		lock.Libraries[k.Value] = Locked{Version: entry.Version, Digest: entry.Digest, Line: k.Line}
	}
	return lock, ds
}

// RenderLock writes the lock for the given Libraries.
func RenderLock(libs []LibrarySource) []byte {
	sorted := append([]LibrarySource(nil), libs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	var b bytes.Buffer
	b.WriteString("# Written by markitect lock from the manifest. Do not edit.\nlibraries:\n")
	for _, l := range sorted {
		fmt.Fprintf(&b, "  %s:\n    version: %s\n    digest: %s\n", l.Name, l.Version, LibraryDigest(l.Files))
	}
	return b.Bytes()
}

// VerifyLock reports every Library whose version or content does not match
// the lock, and lock entries the manifest does not name.
func VerifyLock(file string, lock Lock, libs []LibrarySource) Diagnostics {
	var ds Diagnostics
	named := map[string]bool{}
	for _, l := range libs {
		named[l.Name] = true
		locked, ok := lock.Libraries[l.Name]
		switch {
		case !ok:
			ds.Add("lock.mismatch", file, 0, "Library %s %s is not in the lock; run markitect lock", l.Name, l.Version)
		case locked.Version != l.Version:
			ds.Add("lock.mismatch", file, locked.Line, "the lock pins %s %s, but the manifest names %s; run markitect lock", l.Name, locked.Version, l.Version)
		case locked.Digest != LibraryDigest(l.Files):
			ds.Add("lock.mismatch", file, locked.Line, "the content of Library %s %s does not match the digest the lock pins; restore the Library or run markitect lock", l.Name, l.Version)
		}
	}
	for name, locked := range lock.Libraries {
		if !named[name] {
			ds.Add("lock.mismatch", file, locked.Line, "the lock pins %s, which the manifest does not name; run markitect lock", name)
		}
	}
	return ds
}

// LibraryDigest is the SHA-256 over a Library's files, in path order, each as
// its path, a NUL byte, its content and a NUL byte.
func LibraryDigest(files []File) string {
	sorted := append([]File(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	h := sha256.New()
	for _, f := range sorted {
		h.Write([]byte(f.Path))
		h.Write([]byte{0})
		h.Write(f.Data)
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// mapping decodes a YAML document whose top level is a mapping.
func mapping(data []byte) (map[string]*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("not valid YAML: %v", err)
	}
	out := map[string]*yaml.Node{}
	if len(doc.Content) == 0 {
		return out, nil
	}
	top := doc.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the top level must be a mapping")
	}
	for i := 0; i+1 < len(top.Content); i += 2 {
		if _, dup := out[top.Content[i].Value]; dup {
			return nil, fmt.Errorf("the key %q appears twice", top.Content[i].Value)
		}
		out[top.Content[i].Value] = top.Content[i+1]
	}
	return out, nil
}

// Mapping decodes a YAML document whose top level is a mapping, for the other
// control files.
func Mapping(data []byte) (map[string]*yaml.Node, error) { return mapping(data) }

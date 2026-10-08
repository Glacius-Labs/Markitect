// Package adoption defines the closed selective-evidence handoff outside Core.
// It validates explicit values and bytes; it never acquires repositories.
package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

const ScopeVersion = "markitect.example.org/adoption-scope/v1alpha1"
const HandoffVersion = "markitect.example.org/adoption-handoff/v1alpha1"

var idPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var commitPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
var reservedPathPattern = regexp.MustCompile(`^(COM|LPT)[1-9]$`)

type Privacy struct {
	Constraints   string `yaml:"constraints"`
	AllowExcerpts bool   `yaml:"allowExcerpts"`
}
type SelectedPath struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}
type Exclusion struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}
type Coverage struct {
	ID         string `yaml:"id"`
	Repository string `yaml:"repository"`
	Question   string `yaml:"question"`
	State      string `yaml:"state"`
	Reason     string `yaml:"reason"`
}
type Artifact struct {
	Path   string `yaml:"path"`
	Digest string `yaml:"digest"`
}

// Optional Markitect identities are supplied evidence, not recomputed truth.
// Manifest, report and compiled context must themselves be selected blobs.
type MarkitectEvidence struct {
	Manifest        Artifact `yaml:"manifest"`
	SelectionDigest string   `yaml:"selectionDigest"`
	Version         string   `yaml:"version"`
	BuildDigest     string   `yaml:"buildDigest"`
	Report          Artifact `yaml:"report"`
	Context         Artifact `yaml:"context"`
}
type ScopeRepository struct {
	ID         string             `yaml:"id"`
	Root       string             `yaml:"root"`
	Commit     string             `yaml:"commit"`
	Paths      []SelectedPath     `yaml:"paths"`
	Exclusions []Exclusion        `yaml:"exclusions,omitempty"`
	Markitect  *MarkitectEvidence `yaml:"markitect,omitempty"`
}
type Scope struct {
	APIVersion   string            `yaml:"apiVersion"`
	ID           string            `yaml:"id"`
	Purpose      string            `yaml:"purpose"`
	Review       string            `yaml:"review"`
	Privacy      Privacy           `yaml:"privacy"`
	Retention    string            `yaml:"retention"`
	Repositories []ScopeRepository `yaml:"repositories"`
	Coverage     []Coverage        `yaml:"coverage"`
}
type RepositoryIdentity struct {
	Root         string `yaml:"root"`
	GitDir       string `yaml:"gitDir"`
	CommonDir    string `yaml:"commonDir"`
	ObjectFormat string `yaml:"objectFormat"`
	Digest       string `yaml:"digest"`
}
type File struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
	Mode   string `yaml:"mode"`
	Digest string `yaml:"digest"`
}
type Repository struct {
	ID             string             `yaml:"id"`
	Identity       RepositoryIdentity `yaml:"identity"`
	Commit         string             `yaml:"commit"`
	SnapshotDigest string             `yaml:"snapshotDigest"`
	Files          []File             `yaml:"files"`
	Exclusions     []Exclusion        `yaml:"exclusions,omitempty"`
	Markitect      *MarkitectEvidence `yaml:"markitect,omitempty"`
}
type Handoff struct {
	APIVersion      string       `yaml:"apiVersion"`
	ID              string       `yaml:"id"`
	Digest          string       `yaml:"digest"`
	SelectionDigest string       `yaml:"selectionDigest"`
	CaptureDigest   string       `yaml:"captureDigest"`
	Purpose         string       `yaml:"purpose"`
	Review          string       `yaml:"review"`
	Privacy         Privacy      `yaml:"privacy"`
	Retention       string       `yaml:"retention"`
	Repositories    []Repository `yaml:"repositories"`
	Coverage        []Coverage   `yaml:"coverage"`
}

func Hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func ValueDigest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return Hash(b)
}
func ValidID(s string) bool               { return idPattern.MatchString(s) }
func ValidHash(s string) bool             { return hashPattern.MatchString(s) }
func ValidCommit(s string) bool           { return commitPattern.MatchString(s) }
func BlobKey(repository, p string) string { return repository + "/" + p }

// ExactPath is deliberately portable and refuses pathspec/glob/alias spelling.
func ExactPath(p string) error {
	if p == "" || !utf8.ValidString(p) || path.Clean(p) != p || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\:*?[]<>\"|\x00\r\n\t") {
		return fmt.Errorf("unsafe exact path %q", p)
	}
	for _, s := range strings.Split(p, "/") {
		for _, r := range s {
			if r < 32 {
				return fmt.Errorf("unsafe exact path %q", p)
			}
		}
		if s == "" || s == "." || s == ".." || strings.EqualFold(s, ".git") || strings.HasSuffix(s, ".") || strings.HasSuffix(s, " ") {
			return fmt.Errorf("unsafe exact path %q", p)
		}
		base := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || reservedPathPattern.MatchString(base) {
			return fmt.Errorf("reserved exact path %q", p)
		}
	}
	return nil
}
func ValidateScope(s Scope) error {
	if s.APIVersion != ScopeVersion || !ValidID(s.ID) || strings.TrimSpace(s.Purpose) == "" || strings.TrimSpace(s.Review) == "" || strings.TrimSpace(s.Privacy.Constraints) == "" || strings.TrimSpace(s.Retention) == "" {
		return fmt.Errorf("scope requires version, stable ID, purpose, review, privacy and retention")
	}
	if len(s.Repositories) == 0 || len(s.Repositories) > 32 {
		return fmt.Errorf("scope requires 1..32 repositories")
	}
	ids := map[string]bool{}
	totalPaths := 0
	for _, r := range s.Repositories {
		if !ValidID(r.ID) || ids[r.ID] || r.Root == "" || !ValidCommit(r.Commit) {
			return fmt.Errorf("invalid/duplicate repository identity or non-full commit: %q", r.ID)
		}
		ids[r.ID] = true
		if len(r.Paths) == 0 || len(r.Paths) > 10000 {
			return fmt.Errorf("repository %s requires 1..10000 selected paths", r.ID)
		}
		totalPaths += len(r.Paths)
		if totalPaths > 100000 {
			return fmt.Errorf("scope exceeds 100000 selected paths")
		}
		seen := map[string]bool{}
		components := map[string]struct {
			name      string
			directory bool
		}{}
		for _, p := range r.Paths {
			if err := ExactPath(p.Path); err != nil {
				return err
			}
			key := foldPath(p.Path)
			if seen[key] || strings.TrimSpace(p.Reason) == "" {
				return fmt.Errorf("duplicate/aliased selection or missing reason: %s", p.Path)
			}
			seen[key] = true
			parts := strings.Split(p.Path, "/")
			for i := range parts {
				name := strings.Join(parts[:i+1], "/")
				key := foldPath(name)
				dir := i < len(parts)-1
				if prior, ok := components[key]; ok && (prior.name != name || prior.directory != dir) {
					return fmt.Errorf("aliased selection components %q and %q", prior.name, name)
				}
				components[key] = struct {
					name      string
					directory bool
				}{name, dir}
			}
		}
		excluded := map[string]bool{}
		for _, e := range r.Exclusions {
			if err := ExactPath(e.Path); err != nil {
				return err
			}
			key := foldPath(e.Path)
			if excluded[key] || strings.TrimSpace(e.Reason) == "" {
				return fmt.Errorf("duplicate exclusion or missing reason: %s", e.Path)
			}
			excluded[key] = true
			for _, p := range r.Paths {
				a, b := foldPath(p.Path), key
				if a == b || strings.HasPrefix(a, b+"/") {
					return fmt.Errorf("selected path is excluded: %s", p.Path)
				}
			}
		}
	}
	return ValidateCoverage(s.Coverage, ids)
}

func foldPath(p string) string {
	var b strings.Builder
	for _, r := range p {
		min := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < min {
				min = next
			}
		}
		b.WriteRune(min)
	}
	return b.String()
}
func ValidateCoverage(c []Coverage, repositories map[string]bool) error {
	ids := map[string]bool{}
	for _, v := range c {
		if !ValidID(v.ID) || ids[v.ID] || !repositories[v.Repository] || strings.TrimSpace(v.Question) == "" || strings.TrimSpace(v.Reason) == "" {
			return fmt.Errorf("invalid coverage record %q", v.ID)
		}
		ids[v.ID] = true
		switch v.State {
		case "examined", "no-evidence-found", "unavailable", "redacted-or-withheld", "uninspected":
		default:
			return fmt.Errorf("unknown coverage state %q", v.State)
		}
	}
	return nil
}
func Seal(h *Handoff) {
	if h.Coverage == nil {
		h.Coverage = []Coverage{}
	}
	sort.Slice(h.Repositories, func(i, j int) bool { return h.Repositories[i].ID < h.Repositories[j].ID })
	for i := range h.Repositories {
		r := &h.Repositories[i]
		if len(r.Exclusions) == 0 {
			r.Exclusions = nil
		}
		sort.Slice(r.Files, func(i, j int) bool { return r.Files[i].Path < r.Files[j].Path })
		sort.Slice(r.Exclusions, func(i, j int) bool { return r.Exclusions[i].Path < r.Exclusions[j].Path })
	}
	sort.Slice(h.Coverage, func(i, j int) bool { return h.Coverage[i].ID < h.Coverage[j].ID })
	selections := append([]Repository(nil), h.Repositories...)
	for i := range selections {
		selections[i].SnapshotDigest = ""
	}
	h.SelectionDigest = ValueDigest(selections)
	h.CaptureDigest = ValueDigest(h.Repositories)
	copy := *h
	copy.Digest = ""
	h.Digest = ValueDigest(copy)
}

// ValidateHandoff checks the supplied selected-only byte set, never a source.
func ValidateHandoff(h Handoff, blobs map[string][]byte) error {
	if err := ValidateManifest(h); err != nil {
		return err
	}
	if h.APIVersion != HandoffVersion {
		return fmt.Errorf("unsupported handoff version %q", h.APIVersion)
	}
	scope := Scope{APIVersion: ScopeVersion, ID: h.ID, Purpose: h.Purpose, Review: h.Review, Privacy: h.Privacy, Retention: h.Retention, Coverage: h.Coverage}
	expected := map[string]bool{}
	for _, r := range h.Repositories {
		sr := ScopeRepository{ID: r.ID, Root: r.Identity.Root, Commit: r.Commit, Exclusions: r.Exclusions}
		if r.Identity.GitDir == "" || r.Identity.CommonDir == "" || (r.Identity.ObjectFormat != "sha1" && r.Identity.ObjectFormat != "sha256") || !ValidHash(r.Identity.Digest) {
			return fmt.Errorf("invalid repository binding %s", r.ID)
		}
		identity := r.Identity
		identity.Digest = ""
		if ValueDigest(identity) != r.Identity.Digest {
			return fmt.Errorf("repository identity digest mismatch %s", r.ID)
		}
		if (r.Identity.ObjectFormat == "sha1" && len(r.Commit) != 40) || (r.Identity.ObjectFormat == "sha256" && len(r.Commit) != 64) {
			return fmt.Errorf("commit/object format mismatch %s", r.ID)
		}
		snap := &snapshot.Snapshot{ID: r.Commit, Files: map[string][]byte{}, Modes: map[string]string{}}
		for _, f := range r.Files {
			sr.Paths = append(sr.Paths, SelectedPath{Path: f.Path, Reason: f.Reason})
			key := BlobKey(r.ID, f.Path)
			data, ok := blobs[key]
			if !ok || !utf8.Valid(data) || !ValidHash(f.Digest) || Hash(data) != f.Digest {
				return fmt.Errorf("missing/changed/non-UTF-8 selected evidence %s", key)
			}
			if f.Mode != snapshot.RegularMode && f.Mode != snapshot.ExecutableMode {
				return fmt.Errorf("unsupported evidence mode %s", key)
			}
			expected[key] = true
			snap.Files[f.Path] = data
			snap.Modes[f.Path] = f.Mode
		}
		if snap.Digest() != r.SnapshotDigest {
			return fmt.Errorf("selected snapshot digest mismatch %s", r.ID)
		}
		if m := r.Markitect; m != nil {
			if !ValidHash(m.SelectionDigest) || m.Version == "" || !ValidHash(m.BuildDigest) {
				return fmt.Errorf("incomplete optional Markitect evidence %s", r.ID)
			}
			for _, a := range []Artifact{m.Manifest, m.Report, m.Context} {
				if err := ExactPath(a.Path); err != nil {
					return err
				}
				if !expected[BlobKey(r.ID, a.Path)] || !ValidHash(a.Digest) || Hash(blobs[BlobKey(r.ID, a.Path)]) != a.Digest {
					return fmt.Errorf("optional Markitect artifact not selected or changed: %s/%s", r.ID, a.Path)
				}
			}
		}
		scope.Repositories = append(scope.Repositories, sr)
	}
	if err := ValidateScope(scope); err != nil {
		return err
	}
	if len(expected) != len(blobs) {
		return fmt.Errorf("supplied byte set contains unselected evidence")
	}
	copy := h
	Seal(&copy)
	if copy.Digest != h.Digest || copy.SelectionDigest != h.SelectionDigest || copy.CaptureDigest != h.CaptureDigest {
		return fmt.Errorf("handoff/selection/capture digest mismatch")
	}
	return nil
}

// ValidateManifest preflights all names and bindings before filesystem reads.
func ValidateManifest(h Handoff) error {
	if h.APIVersion != HandoffVersion {
		return fmt.Errorf("unsupported handoff version %q", h.APIVersion)
	}
	scope := Scope{APIVersion: ScopeVersion, ID: h.ID, Purpose: h.Purpose, Review: h.Review, Privacy: h.Privacy, Retention: h.Retention, Coverage: h.Coverage}
	for _, r := range h.Repositories {
		sr := ScopeRepository{ID: r.ID, Root: r.Identity.Root, Commit: r.Commit, Exclusions: r.Exclusions}
		identity := r.Identity
		identity.Digest = ""
		if r.Identity.Root == "" || r.Identity.GitDir == "" || r.Identity.CommonDir == "" || (r.Identity.ObjectFormat != "sha1" && r.Identity.ObjectFormat != "sha256") || ValueDigest(identity) != r.Identity.Digest || !ValidHash(r.SnapshotDigest) {
			return fmt.Errorf("invalid repository binding %s", r.ID)
		}
		if (r.Identity.ObjectFormat == "sha1" && len(r.Commit) != 40) || (r.Identity.ObjectFormat == "sha256" && len(r.Commit) != 64) {
			return fmt.Errorf("commit/object format mismatch %s", r.ID)
		}
		for _, f := range r.Files {
			if !ValidHash(f.Digest) || (f.Mode != snapshot.RegularMode && f.Mode != snapshot.ExecutableMode) {
				return fmt.Errorf("invalid selected file binding %s/%s", r.ID, f.Path)
			}
			sr.Paths = append(sr.Paths, SelectedPath{Path: f.Path, Reason: f.Reason})
		}
		if m := r.Markitect; m != nil {
			if !ValidHash(m.SelectionDigest) || strings.TrimSpace(m.Version) == "" || !ValidHash(m.BuildDigest) {
				return fmt.Errorf("incomplete optional Markitect evidence %s", r.ID)
			}
			for _, a := range []Artifact{m.Manifest, m.Report, m.Context} {
				if err := ExactPath(a.Path); err != nil {
					return err
				}
				found := false
				for _, f := range r.Files {
					if f.Path == a.Path && f.Digest == a.Digest {
						found = true
						break
					}
				}
				if !ValidHash(a.Digest) || !found {
					return fmt.Errorf("optional Markitect artifact not selected or changed: %s/%s", r.ID, a.Path)
				}
			}
		}
		scope.Repositories = append(scope.Repositories, sr)
	}
	if err := ValidateScope(scope); err != nil {
		return err
	}
	copy := h
	Seal(&copy)
	if copy.Digest != h.Digest || copy.SelectionDigest != h.SelectionDigest || copy.CaptureDigest != h.CaptureDigest {
		return fmt.Errorf("handoff/selection/capture digest mismatch")
	}
	return nil
}

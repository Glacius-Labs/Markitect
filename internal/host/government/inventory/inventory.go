// Package inventory captures bounded, read-only metadata about a native tree.
package inventory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

const (
	maxEntries    = 100_000
	maxFileBytes  = int64(64 << 20)
	maxTotalBytes = int64(512 << 20)
)

type Boundary struct {
	Path   string `json:"path" yaml:"path"`
	Reason string `json:"reason" yaml:"reason"`
}

type Options struct {
	Roots        []string   `json:"roots" yaml:"roots"`
	Exclusions   []Boundary `json:"exclusions,omitempty" yaml:"exclusions,omitempty"`
	AdmitIgnored []Boundary `json:"admitIgnored,omitempty" yaml:"admitIgnored,omitempty"`
}

type Entry struct {
	Path   string `json:"path" yaml:"path"`
	Type   string `json:"type" yaml:"type"`
	Mode   string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
	Source string `json:"source" yaml:"source"`
	Status string `json:"status" yaml:"status"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type Report struct {
	Roots        []string   `json:"roots" yaml:"roots"`
	Exclusions   []Boundary `json:"exclusions,omitempty" yaml:"exclusions,omitempty"`
	AdmitIgnored []Boundary `json:"admitIgnored,omitempty" yaml:"admitIgnored,omitempty"`
	Entries      []Entry    `json:"entries" yaml:"entries"`
	Digest       string     `json:"digest" yaml:"digest"`
	Revision     string     `json:"revision,omitempty" yaml:"revision,omitempty"`
	Provisional  bool       `json:"provisional" yaml:"provisional"`
	Complete     bool       `json:"complete" yaml:"complete"`
}

// Capture scans explicit working-tree roots. Ignored contents are read only
// below AdmitIgnored paths. It does not read .git or follow links/submodules.
func Capture(repo string, options Options) (Report, error) {
	if repo == "" {
		return Report{}, errors.New("inventory repository root is required")
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return Report{}, fmt.Errorf("resolve inventory root: %w", err)
	}
	if err = rejectRootLinks(abs); err != nil {
		return Report{}, err
	}
	rootInfo, err := os.Lstat(abs)
	if err != nil {
		return Report{}, fmt.Errorf("stat inventory root: %w", err)
	}
	if isLink(rootInfo) || !rootInfo.IsDir() {
		return Report{}, errors.New("inventory root must be a real directory")
	}
	opts, err := validateOptions(options)
	if err != nil {
		return Report{}, err
	}
	tracked, untracked, ignored, submodules, revision, err := gitPaths(abs)
	if err != nil {
		return Report{}, err
	}
	r := Report{Roots: opts.roots, Exclusions: opts.exclusions, AdmitIgnored: opts.admitIgnored, Revision: revision, Provisional: true, Complete: true, Entries: []Entry{}}
	var total int64
	seen := map[string]bool{}
	for _, rel := range opts.roots {
		full := abs
		if rel != "." {
			full = filepath.Join(abs, filepath.FromSlash(rel))
		}
		if err := rejectRootLinks(filepath.Dir(full)); err != nil {
			return Report{}, fmt.Errorf("inventory root %q: %w", rel, err)
		}
		info, statErr := os.Lstat(full)
		if statErr != nil {
			appendUnreadable(&r, rel, statErr)
			continue
		}
		if reason, excluded := exclusionReason(rel, opts.exclusions); excluded {
			r.Entries = append(r.Entries, Entry{Path: rel, Type: entryType(info), Mode: modeString(info), Source: classify(rel, tracked, untracked, ignored), Status: "excluded", Reason: reason})
			continue
		}
		if isLink(info) {
			r.Entries = append(r.Entries, Entry{Path: rel, Type: "symlink", Mode: modeString(info), Source: classify(rel, tracked, untracked, ignored), Status: "boundary", Reason: "symlink or reparse target is not followed"})
			continue
		}
		if info.IsDir() {
			if err := walk(full, rel, opts, tracked, untracked, ignored, submodules, seen, &r, &total); err != nil {
				return Report{}, err
			}
		} else if err := captureFile(full, rel, info, opts, tracked, untracked, ignored, seen, &r, &total); err != nil {
			return Report{}, err
		}
	}
	sort.Slice(r.Entries, func(i, j int) bool { return r.Entries[i].Path < r.Entries[j].Path })
	if err := validateUniqueEntries(r.Entries); err != nil {
		return Report{}, err
	}
	r.Digest, err = digestReport(r)
	if err != nil {
		return Report{}, err
	}
	return r, nil
}

// ReadInput reads one explicitly named regular file under repo, refusing
// symlink/reparse ancestors and bounding bytes. Callers bind its digest to the
// captured inventory entry before treating it as an observed project input.
func ReadInput(repo, name string, limit int64) ([]byte, error) {
	if repo == "" || limit <= 0 {
		return nil, errors.New("input root and positive byte limit are required")
	}
	clean, err := cleanPath(name)
	if err != nil || clean != name || clean == "." {
		return nil, fmt.Errorf("input path %q must be a clean repository-relative file path", name)
	}
	for _, part := range strings.Split(clean, "/") {
		if strings.EqualFold(part, ".git") {
			return nil, errors.New("Git metadata is not a project input")
		}
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, fmt.Errorf("resolve input root: %w", err)
	}
	if err := rejectRootLinks(abs); err != nil {
		return nil, err
	}
	full := filepath.Join(abs, filepath.FromSlash(clean))
	if err := rejectRootLinks(filepath.Dir(full)); err != nil {
		return nil, err
	}
	before, err := os.Lstat(full)
	if err != nil {
		return nil, fmt.Errorf("stat input %q: %w", clean, err)
	}
	if isLink(before) || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("input %q must be a regular file without link/reparse components", clean)
	}
	if before.Size() > limit {
		return nil, fmt.Errorf("input %q exceeds %d-byte limit", clean, limit)
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("open input %q: %w", clean, err)
	}
	opened, se := f.Stat()
	after, le := os.Lstat(full)
	if se != nil || le != nil || isLink(after) || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("input %q changed or became a link during capture", clean)
	}
	data, re := io.ReadAll(io.LimitReader(f, limit+1))
	ce := f.Close()
	post, pe := os.Lstat(full)
	if re != nil || ce != nil || pe != nil || isLink(post) || !os.SameFile(before, post) || int64(len(data)) > limit {
		return nil, fmt.Errorf("input %q could not be read consistently within limits", clean)
	}
	return data, nil
}

type validatedOptions struct {
	roots                    []string
	exclusions, admitIgnored []Boundary
}

func validateOptions(o Options) (validatedOptions, error) {
	if len(o.Roots) == 0 {
		return validatedOptions{}, errors.New("inventory requires at least one explicit root")
	}
	if len(o.Roots) > 1024 || len(o.Exclusions) > 4096 || len(o.AdmitIgnored) > 4096 {
		return validatedOptions{}, errors.New("inventory scope exceeds configured boundary limits")
	}
	v := validatedOptions{}
	for _, value := range o.Roots {
		clean, err := cleanPath(value)
		if err != nil || clean != value {
			return v, fmt.Errorf("inventory root %q must be a clean relative path", value)
		}
		if hasGitComponent(clean) {
			return v, fmt.Errorf("inventory root %q enters Git metadata", value)
		}
		v.roots = append(v.roots, clean)
	}
	sort.Strings(v.roots)
	for i := range v.roots {
		for j := i + 1; j < len(v.roots); j++ {
			if pathAliasContains(v.roots[i], v.roots[j]) || pathAliasContains(v.roots[j], v.roots[i]) {
				return v, fmt.Errorf("inventory roots %q and %q overlap", v.roots[i], v.roots[j])
			}
		}
	}
	var err error
	if v.exclusions, err = validateBoundaries(o.Exclusions, v.roots, true); err != nil {
		return v, fmt.Errorf("inventory exclusions: %w", err)
	}
	if v.admitIgnored, err = validateBoundaries(o.AdmitIgnored, v.roots, false); err != nil {
		return v, fmt.Errorf("inventory ignored admissions: %w", err)
	}
	return v, nil
}

func validateBoundaries(bs []Boundary, roots []string, rejectOverlap bool) ([]Boundary, error) {
	out := append([]Boundary(nil), bs...)
	for i, b := range out {
		clean, err := cleanPath(b.Path)
		if err != nil || clean != b.Path || strings.TrimSpace(b.Reason) == "" {
			return nil, fmt.Errorf("boundary %q requires a clean path and reason", b.Path)
		}
		if hasGitComponent(clean) {
			return nil, fmt.Errorf("boundary %q enters Git metadata", clean)
		}
		inside := false
		for _, root := range roots {
			if pathAliasContains(root, clean) {
				inside = true
				break
			}
		}
		if !inside {
			return nil, fmt.Errorf("boundary %q is outside declared roots", clean)
		}
		if rejectOverlap {
			for j := 0; j < i; j++ {
				if pathAliasContains(out[j].Path, clean) || pathAliasContains(clean, out[j].Path) {
					return nil, fmt.Errorf("exclusions %q and %q overlap", out[j].Path, clean)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Reason < out[j].Reason
	})
	return out, nil
}

func walk(dir, rel string, opts validatedOptions, tracked, untracked, ignored, submodules map[string]bool, seen map[string]bool, r *Report, total *int64) error {
	if reason, ok := exclusionReason(rel, opts.exclusions); ok {
		r.Entries = append(r.Entries, Entry{Path: rel, Type: "directory", Mode: "directory", Source: classify(rel, tracked, untracked, ignored), Status: "excluded", Reason: reason})
		return nil
	}
	if submodules[nativePathKey(rel)] {
		r.Entries = append(r.Entries, Entry{Path: rel, Type: "submodule", Mode: "160000", Source: classify(rel, tracked, untracked, ignored), Status: "boundary", Reason: "Git submodule contents are not traversed"})
		return nil
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		appendUnreadable(r, rel, err)
		return nil
	}
	if isLink(fi) {
		r.Entries = append(r.Entries, Entry{Path: rel, Type: "symlink", Mode: modeString(fi), Source: classify(rel, tracked, untracked, ignored), Status: "boundary", Reason: "symlink or reparse target is not followed"})
		return nil
	}
	if !fi.IsDir() {
		return captureFile(dir, rel, fi, opts, tracked, untracked, ignored, seen, r, total)
	}
	if rel != "." {
		r.Entries = append(r.Entries, Entry{Path: rel, Type: "directory", Mode: "directory", Source: classify(rel, tracked, untracked, ignored), Status: "observed"})
	}
	beforeRead, err := os.Lstat(dir)
	if err != nil || isLink(beforeRead) || !os.SameFile(fi, beforeRead) {
		appendUnreadable(r, rel, errors.New("directory changed or became a link before traversal"))
		return nil
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		appendUnreadable(r, rel, err)
		return nil
	}
	afterRead, err := os.Lstat(dir)
	if err != nil || isLink(afterRead) || !os.SameFile(fi, afterRead) {
		appendUnreadable(r, rel, errors.New("directory changed or became a link during traversal"))
		return nil
	}
	for _, child := range children {
		if len(r.Entries) >= maxEntries {
			return fmt.Errorf("inventory exceeds %d entries", maxEntries)
		}
		name := child.Name()
		childRel := name
		if rel != "." {
			childRel = rel + "/" + name
		}
		full := filepath.Join(dir, name)
		if strings.EqualFold(name, ".git") {
			fi, e := os.Lstat(full)
			if e != nil {
				appendUnreadable(r, childRel, e)
				continue
			}
			typ := "directory"
			if !fi.IsDir() {
				typ = "git-metadata-link"
			}
			r.Entries = append(r.Entries, Entry{Path: childRel, Type: typ, Mode: modeString(fi), Source: "native", Status: "boundary", Reason: ".git is Git metadata"})
			continue
		}
		if reason, ok := exclusionReason(childRel, opts.exclusions); ok {
			fi, e := os.Lstat(full)
			if e != nil {
				appendUnreadable(r, childRel, e)
				continue
			}
			r.Entries = append(r.Entries, Entry{Path: childRel, Type: entryType(fi), Mode: modeString(fi), Source: classify(childRel, tracked, untracked, ignored), Status: "excluded", Reason: reason})
			continue
		}
		if submodules[nativePathKey(childRel)] {
			r.Entries = append(r.Entries, Entry{Path: childRel, Type: "submodule", Mode: "160000", Source: classify(childRel, tracked, untracked, ignored), Status: "boundary", Reason: "Git submodule contents are not traversed"})
			continue
		}
		if err := walk(full, childRel, opts, tracked, untracked, ignored, submodules, seen, r, total); err != nil {
			return err
		}
	}
	return nil
}

func captureFile(full, rel string, before os.FileInfo, opts validatedOptions, tracked, untracked, ignored map[string]bool, seen map[string]bool, r *Report, total *int64) error {
	if len(r.Entries) >= maxEntries {
		return fmt.Errorf("inventory exceeds %d entries", maxEntries)
	}
	if seen[rel] {
		return nil
	}
	seen[rel] = true
	e := Entry{Path: rel, Type: entryType(before), Mode: modeString(before), Source: classify(rel, tracked, untracked, ignored), Status: "observed"}
	if !before.Mode().IsRegular() {
		e.Status, e.Reason = "boundary", "non-regular filesystem object is not read"
		r.Entries = append(r.Entries, e)
		return nil
	}
	if e.Source == "ignored" && !admitted(rel, opts.admitIgnored) {
		e.Status, e.Reason = "boundary", "ignored file metadata only; bytes require explicit admission"
		r.Entries = append(r.Entries, e)
		return nil
	}
	if before.Size() < 0 || before.Size() > maxFileBytes || *total > maxTotalBytes-before.Size() {
		e.Status, e.Reason = "unreadable", "inventory byte limit exceeded"
		r.Entries = append(r.Entries, e)
		r.Complete = false
		return nil
	}
	f, err := os.Open(full)
	if err != nil {
		e.Status, e.Reason = "unreadable", err.Error()
		r.Entries = append(r.Entries, e)
		r.Complete = false
		return nil
	}
	opened, se := f.Stat()
	after, le := os.Lstat(full)
	if se != nil || le != nil || isLink(after) || !os.SameFile(before, opened) || !opened.Mode().IsRegular() {
		_ = f.Close()
		e.Status, e.Reason = "unreadable", "path changed or became a link during capture"
		r.Entries = append(r.Entries, e)
		r.Complete = false
		return nil
	}
	data, re := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	readInfo, readStatErr := f.Stat()
	ce := f.Close()
	post, postErr := os.Lstat(full)
	if re != nil || ce != nil || readStatErr != nil || postErr != nil || isLink(post) ||
		!os.SameFile(before, readInfo) || !os.SameFile(before, post) ||
		before.Size() != readInfo.Size() || before.Size() != post.Size() ||
		!before.ModTime().Equal(readInfo.ModTime()) || !before.ModTime().Equal(post.ModTime()) ||
		int64(len(data)) > maxFileBytes || int64(len(data)) > maxTotalBytes-*total {
		e.Status, e.Reason = "unreadable", "file could not be read within inventory limits"
		r.Entries = append(r.Entries, e)
		r.Complete = false
		return nil
	}
	*total += int64(len(data))
	sum := sha256.Sum256(data)
	e.Digest = "sha256:" + hex.EncodeToString(sum[:])
	r.Entries = append(r.Entries, e)
	return nil
}

func gitPaths(root string) (tracked, untracked, ignored, submodules map[string]bool, revision string, err error) {
	tracked, untracked, ignored, submodules = map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	if _, probeErr := source.GitOutput(root, "rev-parse", "--show-toplevel"); probeErr != nil {
		marker, markerErr := hasGitMetadataAncestor(root)
		if markerErr != nil {
			return tracked, untracked, ignored, submodules, "", markerErr
		}
		if marker {
			return tracked, untracked, ignored, submodules, "", fmt.Errorf("Git repository metadata is present but Git classification failed: %w", probeErr)
		}
		return tracked, untracked, ignored, submodules, "", nil
	}
	if out, e := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}"); e == nil {
		revision = strings.TrimSpace(string(out))
	} else if _, symbolicErr := source.GitOutput(root, "symbolic-ref", "--quiet", "HEAD"); symbolicErr != nil {
		return tracked, untracked, ignored, submodules, "", fmt.Errorf("read Git HEAD revision: %w", e)
	}
	queries := []struct {
		args []string
		dest map[string]bool
	}{
		{[]string{"ls-files", "-z", "--cached"}, tracked},
		{[]string{"ls-files", "-z", "--others", "--exclude-standard"}, untracked},
		{[]string{"ls-files", "-z", "--others", "--ignored", "--exclude-standard"}, ignored},
	}
	for _, query := range queries {
		out, e := source.GitOutput(root, query.args...)
		if e != nil {
			return tracked, untracked, ignored, submodules, "", fmt.Errorf("classify Git inventory paths: %w", e)
		}
		addPaths(query.dest, out)
	}
	out, e := source.GitOutput(root, "ls-files", "--stage", "-z")
	if e != nil {
		return tracked, untracked, ignored, submodules, "", fmt.Errorf("read Git submodule boundaries: %w", e)
	}
	for _, rec := range bytes.Split(out, []byte{0}) {
		h, n, ok := bytes.Cut(rec, []byte{'\t'})
		if !ok {
			continue
		}
		fields := strings.Fields(string(h))
		if len(fields) > 0 && fields[0] == "160000" {
			submodules[nativePathKey(string(n))] = true
		}
	}
	return tracked, untracked, ignored, submodules, revision, nil
}

func hasGitMetadataAncestor(root string) (bool, error) {
	current, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	for {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect Git metadata marker: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false, nil
		}
		current = parent
	}
}
func addPaths(m map[string]bool, b []byte) {
	for _, p := range bytes.Split(b, []byte{0}) {
		if len(p) > 0 {
			m[nativePathKey(filepath.ToSlash(string(p)))] = true
		}
	}
}
func classify(p string, t, u, i map[string]bool) string {
	p = nativePathKey(p)
	if t[p] {
		return "tracked"
	}
	if i[p] {
		return "ignored"
	}
	if u[p] {
		return "untracked"
	}
	return "native"
}
func exclusionReason(p string, bs []Boundary) (string, bool) {
	for _, b := range bs {
		if containsPath(b.Path, p) {
			return b.Reason, true
		}
	}
	return "", false
}
func admitted(p string, bs []Boundary) bool {
	for _, b := range bs {
		if containsPath(b.Path, p) {
			return true
		}
	}
	return false
}
func cleanPath(s string) (string, error) {
	if s == "." {
		return ".", nil
	}
	if s == "" || strings.ContainsAny(s, "\\:\x00<>\"|?*") || path.IsAbs(s) || filepath.IsAbs(s) || filepath.VolumeName(s) != "" {
		return "", errors.New("invalid path")
	}
	c := path.Clean(s)
	if c == ".." || strings.HasPrefix(c, "../") {
		return "", errors.New("path escapes root")
	}
	for _, component := range strings.Split(c, "/") {
		if strings.TrimRight(component, " .") != component || windowsReservedName(component) {
			return "", errors.New("path has a Windows-ambiguous or reserved component")
		}
	}
	return c, nil
}
func windowsReservedName(component string) bool {
	base := strings.ToUpper(strings.SplitN(strings.TrimRight(component, " ."), ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}
func pathAliasContains(parent, child string) bool {
	if parent == "." {
		return true
	}
	return strings.EqualFold(child, parent) || len(child) > len(parent) && strings.EqualFold(child[:len(parent)], parent) && child[len(parent)] == '/'
}
func hasGitComponent(s string) bool {
	for _, component := range strings.Split(s, "/") {
		if strings.EqualFold(component, ".git") {
			return true
		}
	}
	return false
}
func containsPath(a, b string) bool {
	if a == "." {
		return true
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(b, a) || len(b) > len(a) && strings.EqualFold(b[:len(a)], a) && b[len(a)] == '/'
	}
	return b == a || strings.HasPrefix(b, a+"/")
}
func nativePathKey(p string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}
func entryType(i os.FileInfo) string {
	if isLink(i) {
		return "symlink"
	}
	if i.IsDir() {
		return "directory"
	}
	if i.Mode().IsRegular() {
		return "file"
	}
	return "special"
}
func modeString(i os.FileInfo) string {
	if isLink(i) {
		return "symlink"
	}
	if i.IsDir() {
		return "directory"
	}
	if i.Mode().IsRegular() {
		if i.Mode().Perm()&0111 != 0 {
			return "100755"
		}
		return "100644"
	}
	return i.Mode().String()
}
func appendUnreadable(r *Report, p string, e error) {
	r.Entries = append(r.Entries, Entry{Path: p, Type: "unknown", Source: "native", Status: "unreadable", Reason: e.Error()})
	r.Complete = false
}
func validateUniqueEntries(es []Entry) error {
	if len(es) > maxEntries {
		return fmt.Errorf("inventory exceeds %d entries", maxEntries)
	}
	for i := 1; i < len(es); i++ {
		if es[i-1].Path == es[i].Path {
			return fmt.Errorf("duplicate inventory path %q", es[i].Path)
		}
	}
	return nil
}
func digestReport(r Report) (string, error) {
	v := struct {
		Roots        []string   `json:"roots"`
		Exclusions   []Boundary `json:"exclusions,omitempty"`
		AdmitIgnored []Boundary `json:"admitIgnored,omitempty"`
		Entries      []Entry    `json:"entries"`
		Revision     string     `json:"revision,omitempty"`
		Provisional  bool       `json:"provisional"`
		Complete     bool       `json:"complete"`
	}{r.Roots, r.Exclusions, r.AdmitIgnored, r.Entries, r.Revision, r.Provisional, r.Complete}
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func rejectRootLinks(abs string) error {
	current := filepath.VolumeName(abs) + string(os.PathSeparator)
	rest := strings.TrimPrefix(abs, current)
	for _, part := range strings.Split(rest, string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		i, e := os.Lstat(current)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		if e != nil {
			return fmt.Errorf("inspect inventory root path: %w", e)
		}
		if isLink(i) {
			return fmt.Errorf("inventory root contains a symlink or reparse point: %s", current)
		}
	}
	return nil
}
